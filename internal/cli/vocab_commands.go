package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// RefitPlan is the parsed shape of a refit plan YAML file.
type RefitPlan struct {
	NewTerms []SeedTerm   `yaml:"new_terms"`
	Renames  []TermRename `yaml:"renames"`
	Removals []string     `yaml:"removals"`
}

// SeedTerm is one entry in the bootstrap seed YAML file:
// [{term, description, exemplars}]. Exemplars are situation lines from
// representative member notes; they are rendered into the term-note body,
// which IS the term's embedding text (description alone under-feeds the
// embedding — measured r@5 45.5% vs the 64.6% member-centroid baseline).
type SeedTerm struct {
	Term        string   `yaml:"term"`
	Description string   `yaml:"description"`
	Exemplars   []string `yaml:"exemplars"`
}

// TermRename is one rename entry in a RefitPlan.
type TermRename struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// VocabBootstrapArgs holds parsed flags for `engram vocab bootstrap`.
type VocabBootstrapArgs struct {
	Vault    string  `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll // unbreakable env+desc struct-tag string
	SeedFile string  `targ:"flag,name=seed,required,desc=YAML seed file: list of {term+description} entries (required)"`
	Floor    float32 `targ:"flag,name=floor,desc=minimum cosine similarity for vocab assignment (default 0.35)"`
}

// VocabDeps holds injected I/O dependencies for all vocab write commands
// (bootstrap, propose, refit). Follows the newOsXxxDeps pattern.
type VocabDeps struct {
	// Lock acquires an exclusive flock on vault/.luhmann.lock.
	Lock func(vault string) (func(), error)
	// ListMD returns the .md filenames in vault.
	ListMD func(vault string) ([]string, error)
	// ReadFile reads raw bytes from a path (notes AND sidecars).
	ReadFile func(path string) ([]byte, error)
	// WriteFile atomically writes data to path (create or overwrite).
	WriteFile func(path string, data []byte) error
	// DeleteFile removes a file by path. Used by refit to delete removed term notes.
	DeleteFile func(path string) error
	// WriteSidecar writes an embedding sidecar atomically.
	WriteSidecar func(path string, data []byte) error
	// Embedder embeds text into a vector. Optional; nil skips embedding.
	Embedder embed.Embedder
	// LogWarning logs a non-fatal warning. Optional; nil silences warnings.
	LogWarning func(format string, args ...any)
	// Now returns the current time for created/updated timestamps.
	Now func() time.Time
}

// VocabProposeArgs holds parsed flags for `engram vocab propose`.
// The LLM gate runs agent-side before calling this command; `engram vocab propose`
// performs only the mechanical part (create term note, embed, regen index, minor version bump).
type VocabProposeArgs struct {
	Vault       string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll // unbreakable env+desc struct-tag string
	Term        string `targ:"flag,name=term,required,desc=kebab-case term name (required)"`
	Description string `targ:"flag,name=description,required,desc=one-line term description (required)"`
}

// VocabRefitArgs holds parsed flags for `engram vocab refit`.
// The LLM derivation runs agent-side; `engram vocab refit --plan <yaml>` applies
// the mechanical part. Use --emit-request to print the JSON payload to feed the LLM.
type VocabRefitArgs struct {
	Vault       string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll // unbreakable env+desc struct-tag string
	PlanFile    string `targ:"flag,name=plan,desc=YAML refit plan file (required unless --emit-request)"`
	EmitRequest bool   `targ:"flag,name=emit-request,desc=print JSON payload to feed the LLM and exit"`
}

// VocabStatsArgs holds parsed flags for `engram vocab stats`.
type VocabStatsArgs struct {
	Vault string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll // unbreakable env+desc struct-tag string
}

// VocabStatsDeps holds injected deps for the read-only vocab stats command.
type VocabStatsDeps struct {
	ListMD   func(vault string) ([]string, error)
	ReadFile func(path string) ([]byte, error)
}

// RunVocabBootstrap seeds term notes from a YAML seed file, embeds them,
// mechanically assigns vocab terms to ALL existing non-vocab notes, and
// generates vocab.index.md. Idempotent: re-running refreshes assignments.
func RunVocabBootstrap(ctx context.Context, args VocabBootstrapArgs, deps VocabDeps, stdout io.Writer) error {
	if args.SeedFile == "" {
		return errVocabBootstrapMissingSeed
	}

	floor := args.Floor
	if floor == 0 {
		floor = DefaultVocabFloor
	}

	seedData, readErr := deps.ReadFile(args.SeedFile)
	if readErr != nil {
		return fmt.Errorf("vocab bootstrap: reading seed: %w", readErr)
	}

	var seed []SeedTerm

	unmarshalErr := yaml.Unmarshal(seedData, &seed)
	if unmarshalErr != nil {
		return fmt.Errorf("%w: %w", errVocabBootstrapBadSeed, unmarshalErr)
	}

	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return fmt.Errorf("vocab bootstrap: acquiring vault lock: %w", lockErr)
	}

	defer release()

	when := deps.Now()

	writeAndEmbedSeedTerms(ctx, deps, args.Vault, seed, initialVocabVersion, when)

	// Load term vectors from the just-written sidecars.
	terms, termsErr := loadTermVectors(args.Vault, deps.ListMD, deps.ReadFile)
	if termsErr != nil {
		return fmt.Errorf("vocab bootstrap: loading term vectors: %w", termsErr)
	}

	// Centroid two-pass over all non-vocab member notes: pass 1 against the
	// description+exemplar embeddings, pass 2 against the member centroids.
	// Seed last_refit so the trigger checker has a starting baseline.
	memberCounts := make(map[string]int)

	if len(terms) > 0 {
		memberCounts = retagAllNotesTwoPass(deps, args.Vault, terms, floor, buildLastRefitDoc(deps, args.Vault, when))
	}

	entries := buildIndexEntries(seed, memberCounts)

	// Write vocab.index.md.
	indexContent := renderVocabIndexContent(entries, initialVocabVersion, when)
	indexPath := filepath.Join(args.Vault, vocabIndexFilename)

	writeErr := deps.WriteFile(indexPath, []byte(indexContent))
	if writeErr != nil {
		return fmt.Errorf("vocab bootstrap: writing index: %w", writeErr)
	}

	_, _ = fmt.Fprintf(stdout, "vocab bootstrap: %d terms, %d notes assigned\n", len(seed), sumCounts(memberCounts))

	return nil
}

// RunVocabPropose creates a new term note, embeds it, regenerates vocab.index.md,
// and bumps the minor version. The LLM gate (check: no existing term covers it,
// projected attachment ≤ 20% of vault) runs AGENT-SIDE before calling this command.
func RunVocabPropose(ctx context.Context, args VocabProposeArgs, deps VocabDeps, stdout io.Writer) error {
	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return fmt.Errorf("vocab propose: acquiring vault lock: %w", lockErr)
	}

	defer release()

	when := deps.Now()

	// Read the current version and bump it.
	currentVersion := loadCurrentVocabVersion(args.Vault, deps.ReadFile)
	newVersion := bumpMinorVersion(currentVersion)

	// Write and embed the new term note.
	embedErr := writeAndEmbedTermNote(ctx, deps, args.Vault, args.Term, args.Description, nil, newVersion, when)
	if embedErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab propose: embedding %s failed: %v", args.Term, embedErr)
	}

	// Regenerate the index with all current term notes.
	indexErr := regenVocabIndex(deps, args.Vault, newVersion, when)
	if indexErr != nil {
		return fmt.Errorf("vocab propose: regenerating index: %w", indexErr)
	}

	_, _ = fmt.Fprintf(stdout, "vocab propose: created %s (version → %s)\n", args.Term, newVersion)

	return nil
}

// RunVocabRefit applies a refit plan to the vocab set. When --emit-request is
// set, prints the JSON payload to feed the LLM and exits. Otherwise, the plan
// file drives: new term creation, renames (term note + member rewrites),
// removals (term note deletion + member clearing), re-tag of all members,
// major version bump, and index regen.
func RunVocabRefit(ctx context.Context, args VocabRefitArgs, deps VocabDeps, stdout io.Writer) error {
	if args.EmitRequest {
		return emitRefitRequest(args.Vault, deps, stdout)
	}

	if args.PlanFile == "" {
		return errVocabRefitMissingPlan
	}

	plan, planErr := loadRefitPlan(args.PlanFile, deps.ReadFile)
	if planErr != nil {
		return planErr
	}

	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return fmt.Errorf("vocab refit: acquiring vault lock: %w", lockErr)
	}

	defer release()

	when := deps.Now()

	currentVersion := loadCurrentVocabVersion(args.Vault, deps.ReadFile)
	newVersion := bumpMajorVersion(currentVersion)

	applyRefitRemovals(deps, args.Vault, plan.Removals)
	applyRefitRenames(ctx, deps, args.Vault, plan.Renames, newVersion, when)
	applyRefitNewTerms(ctx, deps, args.Vault, plan.NewTerms, newVersion, when)

	// Clear removed terms from all member notes.
	if len(plan.Removals) > 0 {
		clearErr := clearRemovedTermsFromMembers(deps, args.Vault, plan.Removals)
		if clearErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: clearing removed terms from members: %v", clearErr)
		}
	}

	// Re-tag all members against the new term set (centroid two-pass).
	// Seed last_refit so the trigger checker has a fresh baseline after refit.
	terms, _ := loadTermVectors(args.Vault, deps.ListMD, deps.ReadFile)

	if len(terms) > 0 {
		_ = retagAllNotesTwoPass(deps, args.Vault, terms, DefaultVocabFloor, buildLastRefitDoc(deps, args.Vault, when))
	}

	// Regenerate index.
	indexErr := regenVocabIndex(deps, args.Vault, newVersion, when)
	if indexErr != nil {
		return fmt.Errorf("vocab refit: regenerating index: %w", indexErr)
	}

	_, _ = fmt.Fprintf(stdout, "vocab refit applied: version → %s\n", newVersion)

	return nil
}

// RunVocabStats prints a vocab health report: per-term member counts, vault
// untagged rate, hub terms (>25% of vault), orphan terms (<2 members), and a
// verdict line (OK or REFIT_PENDING with reason) from vocab.centroids.json.
func RunVocabStats(args VocabStatsArgs, deps VocabStatsDeps, stdout io.Writer) error {
	names, listErr := deps.ListMD(args.Vault)
	if listErr != nil {
		return fmt.Errorf("vocab stats: listing vault: %w", listErr)
	}

	termNames, memberCounts, totalNotes, untaggedCount := collectVaultStats(names, deps, args.Vault)
	vocabVersion := loadCurrentVocabVersion(args.Vault, deps.ReadFile)

	// Read refit_pending from centroids (migration: absent = OK, no false fire).
	refitPending := false
	refitReason := ""

	centroidsDoc, centroidsOK := readCentroidsDoc(args.Vault, deps.ReadFile)
	if centroidsOK {
		refitPending = centroidsDoc.RefitPending
		refitReason = centroidsDoc.RefitReason
	}

	sort.Strings(termNames)
	printStatsReport(stdout, termNames, memberCounts, totalNotes, untaggedCount,
		vocabVersion, refitPending, refitReason)

	return nil
}

// unexported constants.
const (
	// hubThreshold is the fraction of vault notes a term must tag to be
	// flagged as a hub in the stats report (>25% of vault).
	hubThreshold = 0.25
	// initialVocabVersion is the version set by bootstrap.
	initialVocabVersion = "1.0"
	// orphanMemberThreshold is the minimum member count below which a term is
	// flagged as an orphan in the stats report (<2 members).
	orphanMemberThreshold = 2
	// pctMultiplier converts a fraction to a percentage.
	pctMultiplier = 100.0
	// versionPartCount is the expected number of "major.minor" version components.
	versionPartCount = 2
	// vocabIndexFilename is the filename of the machine-generated vocab MOC.
	vocabIndexFilename = "vocab.index.md"
	// vocabNotePerm is the file permission used for vocab note writes.
	vocabNotePerm = fs.FileMode(0o600)
	// vocabNotePrefix is the filename prefix shared by all vocab term notes.
	vocabNotePrefix = "vocab."
)

// unexported variables.
var (
	errVocabBootstrapBadSeed     = errors.New("vocab bootstrap: cannot parse seed YAML")
	errVocabBootstrapMissingSeed = errors.New("vocab bootstrap: --seed file is required")
	errVocabRefitBadPlan         = errors.New("vocab refit: cannot parse plan YAML")
	errVocabRefitMissingPlan     = errors.New("vocab refit: --plan file is required unless --emit-request")
)

// noteMiniDoc is used to parse only the vocab: key from an arbitrary note's
// frontmatter — the minimal surface needed by stats and assignment scanning.
type noteMiniDoc struct {
	Type  string   `yaml:"type"`
	Vocab []string `yaml:"vocab,omitempty"`
}

// refitTermEntry is the JSON shape of a term entry in the refit-request payload.
type refitTermEntry struct {
	Term        string `json:"term"`
	Description string `json:"description"`
}

// vocabIndexEntry is one entry in the generated vocab.index.md body.
type vocabIndexEntry struct {
	Term        string
	Description string
	MemberCount int
}

// vocabIndexFrontmatterDoc is the YAML shape of vocab.index.md frontmatter.
type vocabIndexFrontmatterDoc struct {
	Type         string `yaml:"type"`
	VocabVersion string `yaml:"vocab_version"`
	Created      string `yaml:"created"`
}

// applyRefitNewTerms creates new term notes for each new term in the plan.
func applyRefitNewTerms(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	newTerms []SeedTerm,
	newVersion string,
	when time.Time,
) {
	for _, term := range newTerms {
		newErr := writeAndEmbedTermNote(ctx, deps, vault, term.Term, term.Description, term.Exemplars, newVersion, when)
		if newErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: creating new term %s: %v", term.Term, newErr)
		}
	}
}

// applyRefitRemovals deletes term notes for all removed terms.
func applyRefitRemovals(deps VocabDeps, vault string, removals []string) {
	if deps.DeleteFile == nil {
		return
	}

	for _, term := range removals {
		termPath := termNotePath(vault, term)

		delErr := deps.DeleteFile(termPath)
		if delErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: deleting %s: %v", termPath, delErr)
		}
	}
}

// applyRefitRenames deletes old term notes, creates new ones, and rewrites members.
func applyRefitRenames(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	renames []TermRename,
	newVersion string,
	when time.Time,
) {
	for _, rename := range renames {
		// Delete old term note.
		if deps.DeleteFile != nil {
			oldPath := termNotePath(vault, rename.From)

			delErr := deps.DeleteFile(oldPath)
			if delErr != nil && deps.LogWarning != nil {
				deps.LogWarning("vocab refit: deleting old term %s: %v", oldPath, delErr)
			}
		}

		// Create new term note (description carried from old term note if available).
		desc := loadTermDescription(vault, rename.From, deps.ReadFile)

		// Exemplars are refit-maintained; a rename carries only the description
		// forward (the refit plan's re-tag pass regenerates exemplar context).
		embedErr := writeAndEmbedTermNote(ctx, deps, vault, rename.To, desc, nil, newVersion, when)
		if embedErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: creating renamed term %s: %v", rename.To, embedErr)
		}

		// Rewrite member notes: replace old term with new term in both channels.
		rewriteErr := rewriteMemberTermRename(deps, vault, rename.From, rename.To)
		if rewriteErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: rewriting members for rename %s→%s: %v",
				rename.From, rename.To, rewriteErr)
		}
	}
}

// assignTermsToAllNotes scans all non-vocab notes in vault, loads their sidecar
// body vectors, assigns terms, and rewrites both vocab channels. Returns the
// per-term member count and any scan-level error.
func assignTermsToAllNotes(
	deps VocabDeps,
	vault string,
	terms []TermWithVector,
	floor float32,
) (map[string]int, error) {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return nil, fmt.Errorf("listing vault: %w", listErr)
	}

	memberCounts := make(map[string]int)

	for _, name := range names {
		if isVocabKindFilename(name) {
			continue
		}

		assigned := assignVocabToNote(deps, vault, name, terms, floor)

		for _, term := range assigned {
			memberCounts[term]++
		}
	}

	return memberCounts, nil
}

// assignVocabToNote loads the sidecar for a single note, assigns terms, and
// writes the updated content. Returns the assigned terms (or nil if skipped).
func assignVocabToNote(deps VocabDeps, vault, name string, terms []TermWithVector, floor float32) []string {
	notePath := filepath.Join(vault, name)

	sidecarData, sidecarErr := deps.ReadFile(embed.SidecarPath(notePath))
	if sidecarErr != nil {
		return nil // no sidecar → skip
	}

	sidecar, unmarshalErr := embed.UnmarshalSidecar(sidecarData)
	if unmarshalErr != nil || len(sidecar.BodyVector) == 0 {
		return nil
	}

	assigned := AssignVocabTerms(sidecar.BodyVector, terms, floor)

	noteData, readErr := deps.ReadFile(notePath)
	if readErr != nil {
		return assigned
	}

	updated := WriteVocabAssignment(string(noteData), assigned)
	if updated == string(noteData) {
		return assigned
	}

	writeErr := deps.WriteFile(notePath, []byte(updated))
	if writeErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab: writing %s: %v", notePath, writeErr)
	}

	return assigned
}

// buildIndexEntries maps seed terms to member counts to produce index entries.
func buildIndexEntries(seed []SeedTerm, memberCounts map[string]int) []vocabIndexEntry {
	entries := make([]vocabIndexEntry, 0, len(seed))

	for _, term := range seed {
		entries = append(entries, vocabIndexEntry{
			Term:        term.Term,
			Description: term.Description,
			MemberCount: memberCounts[term.Term],
		})
	}

	return entries
}

// buildLastRefitDoc builds a vocabLastRefitDoc stamped with the current note
// count and date. Used by bootstrap and refit to seed last_refit so the trigger
// checker has a baseline from the moment the vocab set is (re)initialised.
func buildLastRefitDoc(deps VocabDeps, vault string, now time.Time) *vocabLastRefitDoc {
	names, _ := deps.ListMD(vault)

	return &vocabLastRefitDoc{
		NoteCount: countNonVocabNoteFiles(names),
		Date:      now.Format(dateFormat),
	}
}

// bumpMajorVersion increments the major component and resets the minor to 0.
func bumpMajorVersion(ver string) string {
	parts := strings.SplitN(ver, ".", versionPartCount)
	if len(parts) != versionPartCount {
		return ver
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return ver
	}

	return strconv.Itoa(major+1) + ".0"
}

// bumpMinorVersion increments the minor component of a "major.minor" version string.
func bumpMinorVersion(ver string) string {
	parts := strings.SplitN(ver, ".", versionPartCount)
	if len(parts) != versionPartCount {
		return ver
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return ver
	}

	return parts[0] + "." + strconv.Itoa(minor+1)
}

// clearRemovalsFromNoteContent filters out removed terms from a note's vocab channels.
// Returns the original content unchanged if no removals apply or the frontmatter is unreadable.
func clearRemovalsFromNoteContent(raw []byte, removalSet map[string]bool) string {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return string(raw)
	}

	var doc noteMiniDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return string(raw)
	}

	kept := filterKeptTerms(doc.Vocab, removalSet)

	return WriteVocabAssignment(string(raw), kept)
}

// clearRemovedTermsFromMembers removes the given terms from all member notes'
// vocab: frontmatter list and Vocab: body line.
func clearRemovedTermsFromMembers(deps VocabDeps, vault string, removals []string) error {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return fmt.Errorf("listing vault: %w", listErr)
	}

	removalSet := make(map[string]bool, len(removals))

	for _, removed := range removals {
		removalSet[removed] = true
	}

	for _, name := range names {
		if isVocabKindFilename(name) {
			continue
		}

		notePath := filepath.Join(vault, name)

		raw, readErr := deps.ReadFile(notePath)
		if readErr != nil {
			continue
		}

		if !noteContainsAnyRemoval(string(raw), removals) {
			continue
		}

		updated := clearRemovalsFromNoteContent(raw, removalSet)
		if updated == string(raw) {
			continue
		}

		writeErr := deps.WriteFile(notePath, []byte(updated))
		if writeErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: clearing removed terms in %s: %v", notePath, writeErr)
		}
	}

	return nil
}

// collectCurrentTermEntries scans names for vocab term notes and returns
// a list of {term, description} entries for the refit-request payload.
func collectCurrentTermEntries(names []string, vault string, deps VocabDeps) []refitTermEntry {
	currentTerms := make([]refitTermEntry, 0)

	for _, name := range names {
		if name == vocabIndexFilename || !isVocabTermFilename(name) {
			continue
		}

		notePath := filepath.Join(vault, name)

		raw, readErr := deps.ReadFile(notePath)
		if readErr != nil {
			continue
		}

		frontmatter, ok := splitFrontmatter(raw)
		if !ok {
			continue
		}

		var doc VocabFrontmatter

		unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
		if unmarshalErr != nil {
			continue
		}

		currentTerms = append(currentTerms, refitTermEntry{Term: doc.Term, Description: doc.Description})
	}

	return currentTerms
}

// collectNoteStats scans non-vocab notes to count total and untagged notes.
func collectNoteStats(names []string, vault string, deps VocabDeps) (totalNotes, untaggedCount int) {
	for _, name := range names {
		if isVocabKindFilename(name) {
			continue
		}

		totalNotes++

		notePath := filepath.Join(vault, name)

		raw, readErr := deps.ReadFile(notePath)
		if readErr != nil {
			continue
		}

		frontmatter, ok := splitFrontmatter(raw)
		if !ok {
			untaggedCount++

			continue
		}

		var doc noteMiniDoc

		unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
		if unmarshalErr != nil || len(doc.Vocab) == 0 {
			untaggedCount++
		}
	}

	return totalNotes, untaggedCount
}

// collectVaultStats scans vault names and returns per-term member counts,
// term names, total note count, and untagged note count.
func collectVaultStats(
	names []string,
	deps VocabStatsDeps,
	vault string,
) (termNames []string, memberCounts map[string]int, totalNotes, untaggedCount int) {
	termNames = make([]string, 0)
	memberCounts = make(map[string]int)

	for _, name := range names {
		if name == vocabIndexFilename {
			continue
		}

		if isVocabTermFilename(name) {
			termNames = append(termNames, termNameFromFilename(name))

			continue
		}

		tags, ok := extractNoteVocabTags(deps, vault, name)
		if !ok {
			continue
		}

		totalNotes++

		if len(tags) == 0 {
			untaggedCount++

			continue
		}

		for _, tag := range tags {
			memberCounts[tag]++
		}
	}

	return termNames, memberCounts, totalNotes, untaggedCount
}

// countMembersFromNotes scans all non-vocab notes and counts per-term members
// by parsing the vocab: frontmatter key. Uses scanNonVocabNotes (vocab_trigger.go)
// to share the loop with collectTriggerVaultStats.
func countMembersFromNotes(
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
	vault string,
) (map[string]int, error) {
	names, listErr := listMD(vault)
	if listErr != nil {
		return nil, fmt.Errorf("listing vault: %w", listErr)
	}

	counts := make(map[string]int)

	scanNonVocabNotes(vault, names, readFile, func(_ string, raw []byte, readErr error) {
		if readErr != nil || len(raw) == 0 {
			return
		}

		frontmatter, ok := splitFrontmatter(raw)
		if !ok {
			return
		}

		var doc noteMiniDoc

		if yaml.Unmarshal(frontmatter, &doc) != nil {
			return
		}

		for _, term := range doc.Vocab {
			counts[term]++
		}
	})

	return counts, nil
}

// embedTermNote embeds a term note and writes its sidecar. Failures are
// warned-and-skipped: a missing sidecar is recoverable via `engram embed apply`.
func embedTermNote(ctx context.Context, deps VocabDeps, notePath, content string) {
	if deps.Embedder == nil || deps.WriteSidecar == nil {
		return
	}

	sidecar, embErr := embed.BuildSidecar(ctx, deps.Embedder, []byte(content))
	if embErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab: embed failed for %s: %v", notePath, embErr)
		}

		return
	}

	writeErr := deps.WriteSidecar(embed.SidecarPath(notePath), embed.MarshalSidecar(sidecar))
	if writeErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab: sidecar write failed for %s: %v", notePath, writeErr)
	}
}

// emitRefitRequest prints the JSON payload that the agent feeds to the LLM
// for deriving a refit plan. The payload contains current_terms (name+description),
// stats, and instruction text.
func emitRefitRequest(vault string, deps VocabDeps, stdout io.Writer) error {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return fmt.Errorf("vocab refit --emit-request: listing vault: %w", listErr)
	}

	currentTerms := collectCurrentTermEntries(names, vault, deps)
	memberCounts, _ := countMembersFromNotes(deps.ListMD, deps.ReadFile, vault)
	totalNotes, untaggedCount := collectNoteStats(names, vault, deps)

	type statsBlock struct {
		TotalNotes    int            `json:"totalNotes"`
		UntaggedCount int            `json:"untaggedCount"`
		MemberCounts  map[string]int `json:"memberCounts"`
	}

	payload := map[string]any{
		"current_terms": currentTerms,
		"stats": statsBlock{
			TotalNotes:    totalNotes,
			UntaggedCount: untaggedCount,
			MemberCounts:  memberCounts,
		},
		"instruction": "Review the current vocabulary term set and propose updates. " +
			"Preserve terms whose meaning held. Merge orphans (< 2 members). " +
			"Split hub terms (> 25% of vault). Output a refit plan YAML: " +
			"{new_terms: [{term, description}], renames: [{from, to}], removals: [term...]}.",
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	encErr := enc.Encode(payload)
	if encErr != nil {
		return fmt.Errorf("vocab refit --emit-request: encoding JSON: %w", encErr)
	}

	return nil
}

// extractNoteVocabTags reads a non-vocab note and returns its `vocab:` tag list.
// Returns nil, false when the note is unreadable, has no parseable frontmatter,
// has unparseable YAML, or is a vocab/vocab-index type note (should be excluded).
func extractNoteVocabTags(deps VocabStatsDeps, vault, name string) ([]string, bool) {
	notePath := filepath.Join(vault, name)

	raw, readErr := deps.ReadFile(notePath)
	if readErr != nil || len(raw) == 0 {
		return nil, false
	}

	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return nil, false
	}

	var doc noteMiniDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return nil, false
	}

	if doc.Type == typeVocab || doc.Type == typeVocabIndex {
		return nil, false
	}

	return doc.Vocab, true
}

// filterKeptTerms returns vocab terms that are not in the removal set.
func filterKeptTerms(vocab []string, removalSet map[string]bool) []string {
	kept := make([]string, 0, len(vocab))

	for _, tag := range vocab {
		if !removalSet[tag] {
			kept = append(kept, tag)
		}
	}

	return kept
}

// isVocabKindFilename reports whether a filename is a vocab note of any kind
// (term note OR index), so both are excluded from member assignment scans.
func isVocabKindFilename(name string) bool {
	return strings.HasPrefix(name, vocabNotePrefix)
}

// isVocabTermFilename reports whether a filename is a vocab term note
// (prefix "vocab." and not "vocab.index.md").
func isVocabTermFilename(name string) bool {
	return strings.HasPrefix(name, vocabNotePrefix) && name != vocabIndexFilename
}

// loadCurrentVocabVersion reads the vocab_version field from vocab.index.md.
// Returns initialVocabVersion ("1.0") when the index file is absent or unreadable.
func loadCurrentVocabVersion(
	vault string,
	readFile func(string) ([]byte, error),
) string {
	indexPath := filepath.Join(vault, vocabIndexFilename)

	raw, err := readFile(indexPath)
	if err != nil {
		return initialVocabVersion
	}

	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return initialVocabVersion
	}

	var doc vocabIndexFrontmatterDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil || doc.VocabVersion == "" {
		return initialVocabVersion
	}

	return doc.VocabVersion
}

// loadRefitPlan reads and parses a refit plan YAML file.
func loadRefitPlan(planFile string, readFile func(string) ([]byte, error)) (RefitPlan, error) {
	planData, readErr := readFile(planFile)
	if readErr != nil {
		return RefitPlan{}, fmt.Errorf("vocab refit: reading plan: %w", readErr)
	}

	var plan RefitPlan

	unmarshalErr := yaml.Unmarshal(planData, &plan)
	if unmarshalErr != nil {
		return RefitPlan{}, fmt.Errorf("%w: %w", errVocabRefitBadPlan, unmarshalErr)
	}

	return plan, nil
}

// loadTermDescription reads the description field from a term note's frontmatter.
// Returns "" when the note is absent or unreadable.
func loadTermDescription(vault, term string, readFile func(string) ([]byte, error)) string {
	notePath := termNotePath(vault, term)

	raw, err := readFile(notePath)
	if err != nil {
		return ""
	}

	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return ""
	}

	var doc VocabFrontmatter

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return ""
	}

	return doc.Description
}

// loadTermVectors scans vault for vocab.*.md files (excluding vocab.index.md)
// and returns the term name + body vector from each note's sidecar.
// Returns nil when no term notes exist (no-op path for backward compat).
func loadTermVectors(
	vault string,
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
) ([]TermWithVector, error) {
	names, err := listMD(vault)
	if err != nil {
		return nil, fmt.Errorf("loading term vectors: listing vault: %w", err)
	}

	result := make([]TermWithVector, 0, len(names))

	for _, name := range names {
		if name == vocabIndexFilename || !isVocabTermFilename(name) {
			continue
		}

		notePath := filepath.Join(vault, name)

		sidecarData, readErr := readFile(embed.SidecarPath(notePath))
		if readErr != nil {
			continue // sidecar not yet embedded — skip
		}

		sidecar, sidecarErr := embed.UnmarshalSidecar(sidecarData)
		if sidecarErr != nil || len(sidecar.BodyVector) == 0 {
			continue
		}

		term := termNameFromFilename(name)
		result = append(result, TermWithVector{Term: term, Vector: sidecar.BodyVector})
	}

	return result, nil
}

// newOsVocabDeps wires VocabDeps to the real filesystem + bundled embedder.
func newOsVocabDeps() VocabDeps {
	return VocabDeps{
		Lock: (&osLearnFS{}).Lock,
		ListMD: func(vault string) ([]string, error) {
			return (&osVaultFS{}).ListMD(vault)
		},
		ReadFile: (&osVaultFS{}).ReadFile,
		WriteFile: func(path string, data []byte) error {
			return atomicWriteFile(path, data, vocabNotePerm)
		},
		DeleteFile: func(path string) error {
			deleteErr := os.Remove(filepath.Clean(path))
			if deleteErr != nil {
				return fmt.Errorf("deleting %s: %w", path, deleteErr)
			}

			return nil
		},
		WriteSidecar: (&osEmbedFS{}).Write,
		Embedder:     sharedEmbedder,
		LogWarning:   logWarningToStderrf,
		Now:          time.Now,
	}
}

// newOsVocabStatsDeps wires VocabStatsDeps to the real filesystem.
func newOsVocabStatsDeps() VocabStatsDeps {
	return VocabStatsDeps{
		ListMD:   (&osVaultFS{}).ListMD,
		ReadFile: (&osVaultFS{}).ReadFile,
	}
}

// noteContainsAnyRemoval reports whether the note content string contains
// any of the removal term names.
func noteContainsAnyRemoval(content string, removals []string) bool {
	for _, removed := range removals {
		if strings.Contains(content, removed) {
			return true
		}
	}

	return false
}

// printStatsReport writes the formatted vocab stats report to stdout.
// refitPending and refitReason are read from vocab.centroids.json by the caller
// (RunVocabStats). Absent centroids → refitPending=false → verdict: OK (migration-safe).
func printStatsReport(
	stdout io.Writer,
	termNames []string,
	memberCounts map[string]int,
	totalNotes, untaggedCount int,
	vocabVersion string,
	refitPending bool,
	refitReason string,
) {
	_, _ = fmt.Fprintf(stdout, "vocab stats (version: %s)\n", vocabVersion)
	_, _ = fmt.Fprintf(stdout, "terms: %d  member-notes: %d  untagged: %d\n",
		len(termNames), totalNotes, untaggedCount)

	untaggedRate := 0.0
	if totalNotes > 0 {
		untaggedRate = float64(untaggedCount) / float64(totalNotes) * pctMultiplier
	}

	_, _ = fmt.Fprintf(stdout, "untagged-rate: %.1f%%\n", untaggedRate)

	for _, term := range termNames {
		count := memberCounts[term]
		flags := ""

		if totalNotes > 0 && float64(count)/float64(totalNotes) > hubThreshold {
			flags += " [hub]"
		}

		if count < orphanMemberThreshold {
			flags += " [orphan]"
		}

		_, _ = fmt.Fprintf(stdout, "  %s: %d members%s\n", term, count, flags)
	}

	// Verdict line — single source of truth is the persisted flag.
	if refitPending {
		_, _ = fmt.Fprintf(stdout, "verdict: REFIT_PENDING (%s)\n", refitReason)
	} else {
		_, _ = fmt.Fprintln(stdout, "verdict: OK")
	}
}

// regenVocabIndex regenerates vocab.index.md from the current term notes in vault.
func regenVocabIndex(deps VocabDeps, vault, vocabVersion string, when time.Time) error {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return fmt.Errorf("listing vault: %w", listErr)
	}

	// Collect term descriptions from term notes.
	termDescriptions := make(map[string]string)
	termNames := make([]string, 0)

	for _, name := range names {
		if name == vocabIndexFilename || !isVocabTermFilename(name) {
			continue
		}

		notePath := filepath.Join(vault, name)

		raw, readErr := deps.ReadFile(notePath)
		if readErr != nil {
			continue
		}

		frontmatter, ok := splitFrontmatter(raw)
		if !ok {
			continue
		}

		var doc VocabFrontmatter

		unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
		if unmarshalErr != nil {
			continue
		}

		termNames = append(termNames, doc.Term)
		termDescriptions[doc.Term] = doc.Description
	}

	sort.Strings(termNames)

	// Count members per term by scanning non-vocab notes.
	memberCounts, _ := countMembersFromNotes(deps.ListMD, deps.ReadFile, vault)

	entries := make([]vocabIndexEntry, 0, len(termNames))

	for _, term := range termNames {
		entries = append(entries, vocabIndexEntry{
			Term:        term,
			Description: termDescriptions[term],
			MemberCount: memberCounts[term],
		})
	}

	indexContent := renderVocabIndexContent(entries, vocabVersion, when)
	indexPath := filepath.Join(vault, vocabIndexFilename)

	writeErr := deps.WriteFile(indexPath, []byte(indexContent))
	if writeErr != nil {
		return fmt.Errorf("writing vocab.index.md: %w", writeErr)
	}

	return nil
}

// renameTermInVocabList parses the note's current vocab: frontmatter list
// (noteMiniDoc pattern, mirroring clearRemovalsFromNoteContent) and returns
// the list with fromTerm substituted by toTerm. changed=false when the note
// has no parseable frontmatter or its list does not contain fromTerm.
func renameTermInVocabList(raw []byte, fromTerm, toTerm string) ([]string, bool) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return nil, false
	}

	var doc noteMiniDoc

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return nil, false
	}

	renamed := make([]string, len(doc.Vocab))
	changed := false

	for i, term := range doc.Vocab {
		if term == fromTerm {
			renamed[i] = toTerm
			changed = true

			continue
		}

		renamed[i] = term
	}

	return renamed, changed
}

// renderTermNoteContent produces the content of a vocab term note. The body
// (description + exemplar list) IS the term's embedding text: exemplars are
// situation lines from representative members, and including them moves the
// term vector toward its members' vector neighborhood.
func renderTermNoteContent(
	term, description string,
	exemplars []string,
	vocabVersion string,
	when time.Time,
) string {
	var body strings.Builder

	body.WriteString(description)
	body.WriteString("\n")

	if len(exemplars) > 0 {
		body.WriteString("\nExemplars:\n")

		for _, exemplar := range exemplars {
			body.WriteString("- " + exemplar + "\n")
		}
	}

	return marshalFrontmatter(VocabFrontmatter{
		Type:         typeVocab,
		Term:         term,
		Description:  description,
		VocabVersion: vocabVersion,
		Created:      when.Format(dateFormat),
	}) + body.String()
}

// renderVocabIndexContent produces the content of vocab.index.md.
func renderVocabIndexContent(entries []vocabIndexEntry, vocabVersion string, when time.Time) string {
	frontmatter := marshalFrontmatter(vocabIndexFrontmatterDoc{
		Type:         typeVocabIndex,
		VocabVersion: vocabVersion,
		Created:      when.Format(dateFormat),
	})

	lines := make([]string, 0, len(entries))

	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("[[vocab.%s]] — %s — %d members",
			entry.Term, entry.Description, entry.MemberCount))
	}

	body := strings.Join(lines, "\n")
	if body != "" {
		body += "\n"
	}

	return frontmatter + body
}

// rewriteMemberTermRename scans all member notes and substitutes fromTerm →
// toTerm in the two vocab channels ONLY (the vocab: frontmatter list, then
// both channels rewritten via the single writer). Prose that merely contains
// the term name as a substring is never touched — a whole-note ReplaceAll
// would corrupt situation/body text mentioning the term.
func rewriteMemberTermRename(deps VocabDeps, vault, fromTerm, toTerm string) error {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return fmt.Errorf("listing vault: %w", listErr)
	}

	for _, name := range names {
		if isVocabKindFilename(name) {
			continue
		}

		notePath := filepath.Join(vault, name)

		raw, readErr := deps.ReadFile(notePath)
		if readErr != nil {
			continue
		}

		renamed, changed := renameTermInVocabList(raw, fromTerm, toTerm)
		if !changed {
			continue
		}

		updated := WriteVocabAssignment(string(raw), renamed)
		if updated == string(raw) {
			continue
		}

		writeErr := deps.WriteFile(notePath, []byte(updated))
		if writeErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: rewriting %s: %v", notePath, writeErr)
		}
	}

	return nil
}

// sumCounts returns the total of all values in a map[string]int.
func sumCounts(m map[string]int) int {
	total := 0

	for _, v := range m {
		total += v
	}

	return total
}

// termNameFromFilename extracts the term name from a vocab term filename.
// "vocab.eval-methodology.md" → "eval-methodology".
func termNameFromFilename(name string) string {
	return strings.TrimPrefix(strings.TrimSuffix(name, ".md"), vocabNotePrefix)
}

// termNotePath returns the full path of the term note for the given term name.
func termNotePath(vault, term string) string {
	return filepath.Join(vault, vocabNotePrefix+term+".md")
}

// writeAndEmbedSeedTerms writes and embeds all seed terms. Failures are
// logged-and-skipped so a single bad embed doesn't abort bootstrap.
func writeAndEmbedSeedTerms(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	seed []SeedTerm,
	version string,
	when time.Time,
) {
	for _, term := range seed {
		embedErr := writeAndEmbedTermNote(ctx, deps, vault, term.Term, term.Description, term.Exemplars, version, when)
		if embedErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab bootstrap: embedding %s failed: %v", term.Term, embedErr)
		}
	}
}

// writeAndEmbedTermNote creates or overwrites a vocab term note with the given
// name, description, exemplars, and version, then writes its embedding sidecar.
func writeAndEmbedTermNote(
	ctx context.Context,
	deps VocabDeps,
	vault, term, description string,
	exemplars []string,
	vocabVersion string,
	when time.Time,
) error {
	content := renderTermNoteContent(term, description, exemplars, vocabVersion, when)
	notePath := termNotePath(vault, term)

	writeErr := deps.WriteFile(notePath, []byte(content))
	if writeErr != nil {
		return fmt.Errorf("writing term note %s: %w", term, writeErr)
	}

	embedTermNote(ctx, deps, notePath, content)

	return nil
}
