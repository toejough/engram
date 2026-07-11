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
// representative member notes; they are rendered into the definition note's
// body, which IS the term's embedding text (description alone under-feeds the
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
	// DeleteFile removes a file by path. Used by refit to delete removed/renamed
	// definition notes and their sidecars.
	DeleteFile func(path string) error
	// ListVecJSON returns the .vec.json filenames in vault — used by
	// migrate-tags to sweep hub sidecars, including any orphan with no
	// surviving .md counterpart, found by listing directly rather than
	// deriving paths from the .md listing. Optional; nil skips the sweep.
	ListVecJSON func(vault string) ([]string, error)
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
// performs only the mechanical part (create the definition note, embed, minor version bump).
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

// RunVocabBootstrap mints a bare-vocab-tagged definition fact note per seed
// term (idempotent: a term that already has a definition note is left
// untouched), mints the vocab-definition family note when absent, embeds
// every minted note, and mechanically assigns vocab terms to ALL existing
// non-vocab notes. No vocab.index.md is ever written — the index is retired
// (#678).
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

	names, _ := deps.ListMD(args.Vault)

	ensureVocabFamilyNote(ctx, deps, args.Vault, &names, initialVocabVersion, when, "bootstrap")
	writeAndEmbedSeedTerms(ctx, deps, args.Vault, &names, seed, when)

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
		memberCounts = retagAllNotesTwoPass(deps, args.Vault, terms, floor, buildLastRefitDoc(names, when))
	}

	_, _ = fmt.Fprintf(stdout, "vocab bootstrap: %d terms, %d notes assigned\n", len(seed), sumCounts(memberCounts))

	return nil
}

// RunVocabPropose mints a new term's definition note, embeds it, and bumps
// the minor version (persisted onto the vocab-definition family note). The
// LLM gate (check: no existing term covers it, projected attachment ≤ 20% of
// vault) runs AGENT-SIDE before calling this command.
func RunVocabPropose(ctx context.Context, args VocabProposeArgs, deps VocabDeps, stdout io.Writer) error {
	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return fmt.Errorf("vocab propose: acquiring vault lock: %w", lockErr)
	}

	defer release()

	when := deps.Now()

	// Read the current version (from the vocab-definition family note), bump
	// it, and persist the bump onto that same family note.
	newVersion := bumpAndPersistVocabVersion(deps, args.Vault, bumpMinorVersion, "vocab propose")

	names, _ := deps.ListMD(args.Vault)

	f := definitionNoteFactFields(args.Term, args.Description, vocabLifecycleSource("propose", newVersion))
	slug := definitionNoteSlug(args.Term)

	mintErr := mintDefinitionNote(ctx, deps, args.Vault, &names, slug, f, "", nil, when)
	if mintErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab propose: embedding %s failed: %v", args.Term, mintErr)
	}

	_, _ = fmt.Fprintf(stdout, "vocab propose: created %s (version → %s)\n", args.Term, newVersion)

	return nil
}

// RunVocabRefit applies a refit plan to the vocab set. When --emit-request is
// set, prints the JSON payload to feed the LLM and exits. Otherwise, the plan
// file drives: new term creation, renames (definition note re-minted in
// place + member rewrites), removals (definition note + sidecar deletion +
// member clearing), re-tag of all members, and a major version bump.
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

	// Read the current version (from the vocab-definition family note), bump
	// it, and persist the bump onto that same family note.
	newVersion := bumpAndPersistVocabVersion(deps, args.Vault, bumpMajorVersion, "vocab refit")

	applyRefitRemovals(deps, args.Vault, plan.Removals)
	applyRefitRenames(ctx, deps, args.Vault, plan.Renames, newVersion, when)

	names, _ := deps.ListMD(args.Vault)
	applyRefitNewTerms(ctx, deps, args.Vault, &names, plan.NewTerms, newVersion, when)

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
		refitNames, _ := deps.ListMD(args.Vault)
		_ = retagAllNotesTwoPass(deps, args.Vault, terms, DefaultVocabFloor, buildLastRefitDoc(refitNames, when))
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
	vocabVersion := loadCurrentVocabVersion(args.Vault, deps.ListMD, deps.ReadFile)

	// Read refit_pending from centroids (migration: absent = OK, no false fire).
	refitPending := false
	refitReason := ""

	centroidsDoc, centroidsOK := readCentroidsDoc(args.Vault, deps.ReadFile)
	if centroidsOK {
		refitPending = centroidsDoc.RefitPending
		refitReason = centroidsDoc.RefitReason
	}

	sort.Strings(termNames)

	qaPairs := countQAPairs(names)
	printStatsReport(stdout, termNames, memberCounts, totalNotes, untaggedCount,
		vocabVersion, refitPending, refitReason, qaPairs)

	return nil
}

// unexported constants.
const (
	// definitionNoteSlugSegments is the expected dot-separated segment count
	// of a note filename ("<id>.<date>.<slug>.md" minus the ".md" extension):
	// id, date, slug.
	definitionNoteSlugSegments = 3
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
	// vocabDefinitionPrefix is the leading segment of a term-definition note's
	// slug: "vocab-<term>-definition".
	vocabDefinitionPrefix = "vocab-"
	// vocabDefinitionSuffix is the trailing segment of a term-definition
	// note's slug: "vocab-<term>-definition".
	vocabDefinitionSuffix = "-definition"
	// vocabFamilySlug is the slug of the family definition note that carries
	// the vault-wide vocab_version (the bare-vocab-tagged note whose slug is
	// NOT a term definition).
	vocabFamilySlug = "vocab-definition"
	// vocabIndexFilename is the filename of the (retired) machine-generated
	// vocab MOC. Kept only for the defensive stats-scan skip and for
	// vocab_centroids.go's old-shape sidecar-metadata scan; #678 Task 7's
	// migration reader is the last place that still needs to recognize it.
	vocabIndexFilename = "vocab.index.md"
	// vocabNotePerm is the file permission used for vocab note writes.
	vocabNotePerm = fs.FileMode(0o600)
	// vocabNotePrefix is the filename prefix shared by all old-shape vocab
	// term notes (pre-#678; retained for isVocabKindFilename/isVocabTermFilename,
	// which vocab_centroids.go's old-shape metadata scan still depends on).
	vocabNotePrefix = "vocab."
)

// unexported variables.
var (
	errVocabBootstrapBadSeed     = errors.New("vocab bootstrap: cannot parse seed YAML")
	errVocabBootstrapMissingSeed = errors.New("vocab bootstrap: --seed file is required")
	errVocabFamilyNoteMissing    = errors.New("vocab: family definition note (vocab-definition) not found")
	errVocabRefitBadPlan         = errors.New("vocab refit: cannot parse plan YAML")
	errVocabRefitMissingPlan     = errors.New("vocab refit: --plan file is required unless --emit-request")
)

// definitionNoteFields is the minimal frontmatter shape read from a
// bare-vocab-tagged definition note: the term's description ("object:", the
// fact-note field a definition's body is derived from) and, on the family
// note only, the vault-wide vocab_version.
type definitionNoteFields struct {
	Object       string `yaml:"object,omitempty"`
	VocabVersion string `yaml:"vocab_version,omitempty"`
}

// noteMiniDoc is used to parse only the type: key from an arbitrary note's
// frontmatter — the minimal surface needed by extractNoteVocabTags to
// exclude vocab/vocab-index-typed notes from member scanning.
type noteMiniDoc struct {
	Type string `yaml:"type"`
}

// refitTermEntry is the JSON shape of a term entry in the refit-request payload.
type refitTermEntry struct {
	Term        string `json:"term"`
	Description string `json:"description"`
}

// applyRefitNewTerms mints a fresh definition note for each new term in the
// refit plan (mintDefinitionNote — same fresh-Luhmann-id mint bootstrap/propose
// use). names is updated in place so ids allocated in this loop cannot collide.
func applyRefitNewTerms(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	names *[]string,
	newTerms []SeedTerm,
	newVersion string,
	when time.Time,
) {
	for _, term := range newTerms {
		f := definitionNoteFactFields(term.Term, term.Description, vocabLifecycleSource("refit", newVersion))
		slug := definitionNoteSlug(term.Term)

		mintErr := mintDefinitionNote(ctx, deps, vault, names, slug, f, "", term.Exemplars, when)
		if mintErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab refit: creating new term %s: %v", term.Term, mintErr)
		}
	}
}

// applyRefitRemovals deletes the definition note AND its embedding sidecar
// for each removed term, located by scanning the vault for a definition note
// whose slug parses to that term (termFromDefinitionSlug).
func applyRefitRemovals(deps VocabDeps, vault string, removals []string) {
	if deps.DeleteFile == nil {
		return
	}

	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab refit: listing vault for removals: %v", listErr)
		}

		return
	}

	for _, term := range removals {
		path, ok := findDefinitionNotePathForTerm(vault, names, term, deps.ReadFile)
		if !ok {
			continue
		}

		deleteDefinitionNoteAndSidecar(deps, path)
	}
}

// applyRefitRenames re-mints each renamed term's definition note (same
// Luhmann id + date, new slug + re-embedded body — see renameDefinitionNote)
// and substitutes vocab/<from> → vocab/<to> in every member note's tags.
func applyRefitRenames(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	renames []TermRename,
	newVersion string,
	when time.Time,
) {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab refit: listing vault for renames: %v", listErr)
		}

		return
	}

	for _, rename := range renames {
		renameDefinitionNote(ctx, deps, vault, names, rename, newVersion, when)

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

		if isQAQuestionFilename(name) {
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

	if isVocabDefinitionNote(string(noteData)) {
		return nil // a definition note must never acquire its own term tag
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

// buildLastRefitDoc builds a vocabLastRefitDoc stamped with the current note
// count and date. Pure: callers pass the names they already listed. Used by
// bootstrap and refit to seed last_refit so the trigger checker has a baseline
// from the moment the vocab set is (re)initialised.
func buildLastRefitDoc(names []string, now time.Time) *vocabLastRefitDoc {
	return &vocabLastRefitDoc{
		NoteCount: countNonVocabNoteFiles(names),
		Date:      now.Format(dateFormat),
	}
}

// bumpAndPersistVocabVersion reads the current vocab_version from the
// vocab-definition family note, applies bump (bumpMinorVersion for propose,
// bumpMajorVersion for refit), and persists the result onto that same family
// note in place. A missing family note (pre-bootstrap vaults) is logged via
// site and skipped, not fatal to the rest of the command. Returns the new
// version for the caller to pass to mintDefinitionNote / applyRefitNewTerms.
func bumpAndPersistVocabVersion(deps VocabDeps, vault string, bump func(string) string, site string) string {
	currentVersion := loadCurrentVocabVersion(vault, deps.ListMD, deps.ReadFile)
	newVersion := bump(currentVersion)

	versionErr := writeVocabVersionToFamilyNote(vault, newVersion, deps.ListMD, deps.ReadFile, deps.WriteFile)
	if versionErr != nil && deps.LogWarning != nil {
		deps.LogWarning("%s: writing family note version: %v", site, versionErr)
	}

	return newVersion
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

// clearRemovalsFromNoteContent filters out removed terms from a note's
// vocab/<term> tags. Returns the original content unchanged if no removals
// apply or the frontmatter is unreadable.
func clearRemovalsFromNoteContent(raw []byte, removalSet map[string]bool) string {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return string(raw)
	}

	currentTerms := vocabTermsFromTags(parseTagsFromFrontmatter(string(frontmatter)))
	kept := filterKeptTerms(currentTerms, removalSet)

	return WriteVocabAssignment(string(raw), kept)
}

// clearRemovedTermsFromMembers removes the given terms from all member notes'
// vocab/<term> tags.
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
		if isVocabRewriteExcluded(name) {
			continue
		}

		clearRemovedTermsFromNote(deps, filepath.Join(vault, name), removals, removalSet)
	}

	return nil
}

// clearRemovedTermsFromNote clears removed terms from a single member note's
// vocab channels. Skips bare-vocab DEFINITION notes (which must never be
// rewritten by term removal) and notes that mention no removed term.
func clearRemovedTermsFromNote(deps VocabDeps, notePath string, removals []string, removalSet map[string]bool) {
	raw, readErr := deps.ReadFile(notePath)
	if readErr != nil {
		return
	}

	if isVocabDefinitionNote(string(raw)) {
		return
	}

	if !noteContainsAnyRemoval(string(raw), removals) {
		return
	}

	updated := clearRemovalsFromNoteContent(raw, removalSet)
	if updated == string(raw) {
		return
	}

	writeErr := deps.WriteFile(notePath, []byte(updated))
	if writeErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab refit: clearing removed terms in %s: %v", notePath, writeErr)
	}
}

// collectCurrentTermEntries scans names for term identity and returns a list
// of {term, description} entries for the refit-request payload. Term identity
// is read SOLELY from the bare-vocab-tagged definition fact note (#678
// Task 5: the union with the old-shape vocab.<term>.md term note is retired —
// a single read source means a term can never appear twice in this list). The
// family note (slug vocab-definition) never contributes an entry —
// termFromDefinitionSlug returns false for it.
func collectCurrentTermEntries(names []string, vault string, deps VocabDeps) []refitTermEntry {
	currentTerms := make([]refitTermEntry, 0)

	for _, name := range names {
		if entry, ok := definitionNoteTermEntry(vault, name, deps.ReadFile); ok {
			currentTerms = append(currentTerms, entry)
		}
	}

	return currentTerms
}

// collectVaultStats scans vault names and returns per-term member counts,
// term names, total note count, and untagged note count. Term identity is
// read SOLELY from the bare-vocab-tagged definition note (definitionNoteTerm)
// — #678 Task 5: the union with the old-shape vocab.<term>.md filename scan is
// retired, matching collectCurrentTermEntries's single-read-source rationale.
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

		if isQAQuestionFilename(name) {
			continue
		}

		if term, ok := definitionNoteTerm(vault, name, deps.ReadFile); ok {
			termNames = append(termNames, term)

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

// definitionNoteExistsForTerm reports whether a bare-vocab-tagged definition
// note already exists for term — bootstrap's idempotency check (a second run
// with the same seed mints nothing for a term that already has one).
func definitionNoteExistsForTerm(
	vault string,
	names []string,
	term string,
	readFile func(string) ([]byte, error),
) bool {
	_, ok := findDefinitionNotePathForTerm(vault, names, term, readFile)

	return ok
}

// definitionNoteFactFields builds the situation/subject/predicate/object
// factFields for a term's definition note (the brief's concrete shape:
// situation "recalling what the <term> vocab term covers, or assigning vocab
// terms", subject "the <term> vocab term", predicate "covers", object the
// caller-supplied description). Tagged bare "vocab" only — never vocab/<term>
// (a definition must never assign its own term). Luhmann is left unset:
// callers (mintDefinitionNote for a fresh mint, renameDefinitionNote for a
// rename) set it once the note's id is known.
func definitionNoteFactFields(term, description, source string) factFields {
	return factFields{
		Situation: fmt.Sprintf("recalling what the %s vocab term covers, or assigning vocab terms", term),
		Subject:   fmt.Sprintf("the %s vocab term", term),
		Predicate: "covers",
		Object:    description,
		Source:    source,
		Tier:      tierL2,
		Tags:      []string{typeVocab},
	}
}

// definitionNoteLocation scans names for the definition note whose slug
// (termFromDefinitionSlug) matches term, returning its basename, Luhmann id +
// date (idAndDateFromNoteFilename — preserved across a rename), and its
// object-field description. ok=false when no matching, readable, well-formed
// definition note is found.
func definitionNoteLocation(
	vault string,
	names []string,
	term string,
	readFile func(string) ([]byte, error),
) (name, id, date, description string, ok bool) {
	for _, candidate := range names {
		t, raw, matchOK := readVocabDefinitionNote(vault, candidate, readFile)
		if !matchOK || t != term {
			continue
		}

		luhmannID, dateStr, idOK := idAndDateFromNoteFilename(candidate)
		if !idOK {
			continue
		}

		fields, fieldsOK := readDefinitionNoteFields(raw)
		if !fieldsOK {
			continue
		}

		return candidate, luhmannID, dateStr, fields.Object, true
	}

	return "", "", "", "", false
}

// definitionNotePath joins vault, id, date, and slug into a note filename of
// the form "<id>.<date>.<slug>.md" — the same shape learnPath renders, but
// taking a raw date STRING (a rename preserves the OLD note's exact date
// text rather than re-deriving one from a time.Time).
func definitionNotePath(vault, id, date, slug string) string {
	return filepath.Join(vault, fmt.Sprintf("%s.%s.%s.md", id, date, slug))
}

// definitionNoteSlug builds a term-definition note's slug:
// "vocab-<term>-definition".
func definitionNoteSlug(term string) string {
	return vocabDefinitionPrefix + term + vocabDefinitionSuffix
}

// definitionNoteTerm returns the term parsed from a bare-vocab-tagged
// definition note's slug (termFromDefinitionSlug), or ("", false) when name
// is not a definition note or its slug does not match the term shape (the
// family note, slug "vocab-definition", or any non-matching slug).
func definitionNoteTerm(vault, name string, readFile func(string) ([]byte, error)) (string, bool) {
	term, _, ok := readVocabDefinitionNote(vault, name, readFile)

	return term, ok
}

// definitionNoteTermEntry returns a refitTermEntry (term + object-field
// description) for a bare-vocab-tagged definition note, or ok=false when name
// is not a definition note, its slug does not match the term shape, or its
// frontmatter is unparseable.
func definitionNoteTermEntry(vault, name string, readFile func(string) ([]byte, error)) (refitTermEntry, bool) {
	term, raw, ok := readVocabDefinitionNote(vault, name, readFile)
	if !ok {
		return refitTermEntry{}, false
	}

	fields, fieldsOK := readDefinitionNoteFields(raw)
	if !fieldsOK {
		return refitTermEntry{}, false
	}

	return refitTermEntry{Term: term, Description: fields.Object}, true
}

// deleteDefinitionNoteAndSidecar deletes notePath and its embedding sidecar,
// logging (not failing) either error. Shared by applyRefitRemovals and
// renameDefinitionNote's old-note cleanup.
func deleteDefinitionNoteAndSidecar(deps VocabDeps, notePath string) {
	delErr := deps.DeleteFile(notePath)
	if delErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab refit: deleting %s: %v", notePath, delErr)
	}

	sidecarPath := embed.SidecarPath(notePath)

	sidecarDelErr := deps.DeleteFile(sidecarPath)
	if sidecarDelErr != nil && deps.LogWarning != nil {
		deps.LogWarning("vocab refit: deleting %s: %v", sidecarPath, sidecarDelErr)
	}
}

// embedDefinitionNote embeds a definition note and writes its sidecar.
// Failures are warned-and-skipped: a missing sidecar is recoverable via
// `engram embed apply`.
func embedDefinitionNote(ctx context.Context, deps VocabDeps, notePath, content string) {
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
	// One vault pass for all stats (names already in hand); unreadable notes
	// count as untagged, matching the trigger path's convention.
	totalNotes, untaggedCount, memberCounts := collectTriggerVaultStatsFromNames(vault, names, deps.ReadFile)

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

// ensureVocabFamilyNote mints the vocab-definition family note when absent
// (findVocabFamilyNote returns errVocabFamilyNoteMissing); a no-op when
// already present — bootstrap's (and migrate-tags') idempotency requirement.
// The minted note documents the tags: convention WITHOUT enumerating any term
// (a maintained term list is the stale-index problem reborn — see
// TestVocabFamilyNote_NeverEnumeratesTerms). Returns true only when a mint
// was attempted AND succeeded (mintErr == nil); false both when one already
// existed (no mint attempted — the idempotent no-op) and when a mint was
// attempted but failed (#678 Task 7 FIX 2 — the prior unconditional `true`
// after an attempt let migrate-tags' counts summary report "minted" for a
// family note that was never actually written; callers needing to
// distinguish "already present" from "mint failed" re-check existence via
// findVocabFamilyNote, e.g. RunVocabMigrateTags's familyOK gate).
func ensureVocabFamilyNote(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	names *[]string,
	vocabVersion string,
	when time.Time,
	site string,
) bool {
	_, _, findErr := findVocabFamilyNote(vault, *names, deps.ReadFile)
	if findErr == nil {
		return false // already present — idempotent no-op
	}

	f := familyDefinitionFactFields(vocabLifecycleSource(site, vocabVersion))

	mintErr := mintDefinitionNote(ctx, deps, vault, names, vocabFamilySlug, f, vocabVersion, nil, when)
	if mintErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab %s: minting family note: %v", site, mintErr)
		}

		return false
	}

	return true
}

// extractNoteVocabTags reads a non-vocab note and returns its member terms,
// read SOLELY from the tags: vocab/<term> namespace (#678 Task 5: the union
// with the legacy `vocab:` frontmatter key is retired). Returns nil, false
// when the note is unreadable, has no parseable frontmatter, has unparseable
// YAML, is a vocab/vocab-index type note, or is a bare-vocab DEFINITION note
// (excluded).
func extractNoteVocabTags(deps VocabStatsDeps, vault, name string) ([]string, bool) {
	notePath := filepath.Join(vault, name)

	raw, readErr := deps.ReadFile(notePath)
	if readErr != nil || len(raw) == 0 {
		return nil, false
	}

	if isVocabDefinitionNote(string(raw)) {
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

	return vocabTermsFromTags(parseTagsFromFrontmatter(string(frontmatter))), true
}

// familyDefinitionFactFields builds the factFields for the vocab-definition
// family note: subject "the vocab tag family", object documents the tags:
// convention WITHOUT enumerating any term name — a maintained term list in
// this note is the stale-index problem reborn (issue #678's most explicit
// warning; see TestVocabFamilyNote_NeverEnumeratesTerms). Luhmann is left
// unset — mintDefinitionNote sets it once allocated.
func familyDefinitionFactFields(source string) factFields {
	return factFields{
		Situation: "recalling how the vocab tag family works, or checking the vault-wide vocab_version",
		Subject:   "the vocab tag family",
		Predicate: "covers",
		Object: "the tags: convention for vocab terms: a definition note carries a bare vocab tag " +
			"documenting one term's meaning, and a member note carries vocab/<term> tags assigning it " +
			"to that term; this note's frontmatter carries the vault-wide vocab_version",
		Source: source,
		Tier:   tierL2,
		Tags:   []string{typeVocab},
	}
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

// findDefinitionNotePathForTerm scans names for the bare-vocab-tagged
// definition note whose slug (termFromDefinitionSlug) matches term, and
// returns its full path. Shared by applyRefitRemovals (delete) and
// definitionNoteExistsForTerm (bootstrap's idempotency check).
func findDefinitionNotePathForTerm(
	vault string,
	names []string,
	term string,
	readFile func(string) ([]byte, error),
) (string, bool) {
	for _, name := range names {
		t, ok := definitionNoteTerm(vault, name, readFile)
		if ok && t == term {
			return filepath.Join(vault, name), true
		}
	}

	return "", false
}

// findVocabFamilyNote scans names for the bare-vocab-tagged family definition
// note (slug "vocab-definition", the one bare-vocab note termFromDefinitionSlug
// excludes from term identity) and returns its full path + raw content.
// Returns errVocabFamilyNoteMissing when no such note is found.
func findVocabFamilyNote(
	vault string,
	names []string,
	readFile func(string) ([]byte, error),
) (string, []byte, error) {
	for _, name := range names {
		notePath := filepath.Join(vault, name)

		raw, readErr := readFile(notePath)
		if readErr != nil || len(raw) == 0 {
			continue
		}

		if !isVocabDefinitionNote(string(raw)) {
			continue
		}

		if slugFromNoteFilename(name) != vocabFamilySlug {
			continue
		}

		return notePath, raw, nil
	}

	return "", nil, errVocabFamilyNoteMissing
}

// idAndDateFromNoteFilename extracts the leading "<id>.<date>" segments from
// a note filename of the form "<id>.<date>.<slug>.md", reusing
// extractLuhmannFromFilename for the id (the same extractor learn's ListIDs
// uses). Returns ok=false when the filename has no valid leading Luhmann id
// or fewer than three dot-separated segments (mirrors slugFromNoteFilename's
// shape check).
func idAndDateFromNoteFilename(name string) (id, date string, ok bool) {
	luhmannID, idOK := extractLuhmannFromFilename(name)
	if !idOK {
		return "", "", false
	}

	const mdExt = ".md"

	parts := strings.SplitN(strings.TrimSuffix(name, mdExt), ".", definitionNoteSlugSegments)
	if len(parts) != definitionNoteSlugSegments {
		return "", "", false
	}

	return luhmannID, parts[1], true
}

// isVocabKindFilename reports whether a filename is a vocab note of any kind
// (old-shape term note OR index), so both are excluded from member
// assignment scans (a new-shape definition note is excluded by content —
// isVocabDefinitionNote — inside assignVocabToNote instead).
func isVocabKindFilename(name string) bool {
	return strings.HasPrefix(name, vocabNotePrefix)
}

// isVocabRewriteExcluded reports whether a filename is skipped by the vocab
// member-note rewrite loops (removal/rename): vocab-kind files, and QA question
// notes — which carry no vocab by design (D5'); the guard enforces that
// invariant rather than relying on it.
func isVocabRewriteExcluded(name string) bool {
	return isVocabKindFilename(name) || isQAQuestionFilename(name)
}

// isVocabTermFilename reports whether a filename is an old-shape vocab term
// note (prefix "vocab." and not "vocab.index.md"). Retained for
// vocab_centroids.go's old-shape sidecar-metadata scan (firstTermSidecarMeta).
func isVocabTermFilename(name string) bool {
	return strings.HasPrefix(name, vocabNotePrefix) && name != vocabIndexFilename
}

// loadCurrentVocabVersion reads the vocab_version field from the
// vocab-definition family note (bare "vocab" tag, slug "vocab-definition") —
// the version's home per #678 Task 4. Returns initialVocabVersion ("1.0")
// — the same migration-safe default the prior vocab.index.md-based read
// used — when the vault cannot be listed, the family note is absent or
// unreadable, or its vocab_version key is empty/unparseable.
func loadCurrentVocabVersion(
	vault string,
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
) string {
	names, listErr := listMD(vault)
	if listErr != nil {
		return initialVocabVersion
	}

	_, raw, findErr := findVocabFamilyNote(vault, names, readFile)
	if findErr != nil {
		return initialVocabVersion
	}

	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return initialVocabVersion
	}

	var doc definitionNoteFields

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

// loadTermVectors scans vault for bare-vocab-tagged definition notes and
// returns each term's name + body vector from its sidecar (#678 Task 5: the
// union with the old-shape vocab.<term>.md term note is retired — a single
// read source means a term can never be double-counted). names are visited
// in the order listMD returns them (filename-sorted for the real OS-backed
// implementation — os.ReadDir sorts by name); the FIRST definition note seen
// per term wins and any later duplicate is skipped — a cheap dedup guard
// against a hand-edited vault carrying two definition notes for the same
// term (structurally impossible via this package's own writers, but not
// enforced at the filesystem level). The family note is excluded via
// termFromDefinitionSlug. Returns nil when no definition notes exist.
func loadTermVectors(
	vault string,
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
) ([]TermWithVector, error) {
	names, err := listMD(vault)
	if err != nil {
		return nil, fmt.Errorf("loading term vectors: listing vault: %w", err)
	}

	seen := make(map[string]bool, len(names))
	result := make([]TermWithVector, 0, len(names))

	for _, name := range names {
		term, ok := definitionNoteTerm(vault, name, readFile)
		if !ok || seen[term] {
			continue
		}

		seen[term] = true

		notePath := filepath.Join(vault, name)

		sidecarData, readErr := readFile(embed.SidecarPath(notePath))
		if readErr != nil {
			continue // sidecar not yet embedded — skip
		}

		sidecar, sidecarErr := embed.UnmarshalSidecar(sidecarData)
		if sidecarErr != nil || len(sidecar.BodyVector) == 0 {
			continue
		}

		result = append(result, TermWithVector{Term: term, Vector: sidecar.BodyVector})
	}

	return result, nil
}

// mintDefinitionNote allocates a fresh top-level Luhmann id
// (nextDefinitionLuhmannID, reusing the same allocator learn's
// writeLearnUnderLock uses) and writes + embeds a brand-new definition note
// (writeAndEmbedDefinitionNote). names is updated in place with the minted
// filename so a subsequent mint in the same lock-held loop cannot collide on
// the id. Callers mint-and-forget the path (idempotency and lookup both go
// through the vault scan, not the return value), so only the error is
// returned.
func mintDefinitionNote(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	names *[]string,
	slug string,
	f factFields,
	vocabVersion string,
	exemplars []string,
	when time.Time,
) error {
	id, idErr := nextDefinitionLuhmannID(*names)
	if idErr != nil {
		return fmt.Errorf("allocating luhmann id: %w", idErr)
	}

	f.Luhmann = id
	path := learnPath(vault, id, slug, when)

	writeErr := writeAndEmbedDefinitionNote(ctx, deps, path, f, vocabVersion, exemplars, when)
	if writeErr != nil {
		return writeErr
	}

	*names = append(*names, filepath.Base(path))

	return nil
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
		ListVecJSON:  (&osVaultFS{}).ListVecJSON,
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

// nextDefinitionLuhmannID computes the next top-level Luhmann id from names,
// reusing extractLuhmannFromFilename (the same id extractor learn's ListIDs
// uses) and nextLuhmannID (the same allocator writeLearnUnderLock uses for
// learn, learn.go:647/luhmann.go:126) — a single source of truth for id
// assignment across learn and vocab minting.
func nextDefinitionLuhmannID(names []string) (string, error) {
	existing := make([]string, 0, len(names))

	for _, name := range names {
		if id, ok := extractLuhmannFromFilename(name); ok {
			existing = append(existing, id)
		}
	}

	id, idErr := nextLuhmannID(existing, "", positionTop)
	if idErr != nil {
		return "", fmt.Errorf("next luhmann id: %w", idErr)
	}

	return id, nil
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
	qaPairs int,
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

	// QA stats: pair count + round-2 gate readiness.
	_, _ = fmt.Fprintf(stdout, "qa pairs: %d\n", qaPairs)

	if qaPairs >= qaRound2MinPairs {
		_, _ = fmt.Fprintf(stdout, "qa round-2 gate: READY (%d>=%d)\n", qaPairs, qaRound2MinPairs)
	} else {
		_, _ = fmt.Fprintf(stdout, "qa round-2 gate: accumulating (%d/%d)\n", qaPairs, qaRound2MinPairs)
	}
}

// readDefinitionNoteFields parses a definition note's minimal frontmatter
// fields (object:, vocab_version:). ok=false when raw has no parseable
// frontmatter or its YAML is malformed.
func readDefinitionNoteFields(raw []byte) (definitionNoteFields, bool) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return definitionNoteFields{}, false
	}

	var doc definitionNoteFields

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil {
		return definitionNoteFields{}, false
	}

	return doc, true
}

// readVocabDefinitionNote reads name's content and, when it is a bare-vocab
// definition note whose slug parses to a term (termFromDefinitionSlug),
// returns (term, raw, true). Returns ok=false for the family note (slug
// "vocab-definition"), any non-definition note, or an unreadable/empty file.
func readVocabDefinitionNote(vault, name string, readFile func(string) ([]byte, error)) (string, []byte, bool) {
	raw, readErr := readFile(filepath.Join(vault, name))
	if readErr != nil || len(raw) == 0 || !isVocabDefinitionNote(string(raw)) {
		return "", nil, false
	}

	term, ok := termFromDefinitionSlug(slugFromNoteFilename(name))
	if !ok {
		return "", nil, false
	}

	return term, raw, true
}

// renameDefinitionNote locates term rename.From's definition note, re-renders
// it under rename.To (new slug, SAME Luhmann id + date — preserved from the
// old filename via definitionNoteLocation), re-embeds it
// (writeAndEmbedDefinitionNote — the body text embeds the term name, so a
// rename changes ContentHash; the sidecar must be regenerated, never copied),
// then deletes the old .md + .vec.json. A rename carries the description
// forward but not exemplars — refit's re-tag pass regenerates member
// exemplar context, so a rename mints no exemplar section (matches the prior
// behavior of the old-shape writer this replaces).
func renameDefinitionNote(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	names []string,
	rename TermRename,
	newVersion string,
	when time.Time,
) {
	oldName, id, dateStr, description, ok := definitionNoteLocation(vault, names, rename.From, deps.ReadFile)
	if !ok {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab refit: rename %s→%s: no definition note found", rename.From, rename.To)
		}

		return
	}

	f := definitionNoteFactFields(rename.To, description, vocabLifecycleSource("refit", newVersion))
	f.Luhmann = id

	newPath := definitionNotePath(vault, id, dateStr, definitionNoteSlug(rename.To))

	writeErr := writeAndEmbedDefinitionNote(ctx, deps, newPath, f, "", nil, when)
	if writeErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab refit: renaming %s→%s: %v", rename.From, rename.To, writeErr)
		}

		return
	}

	if deps.DeleteFile == nil {
		return
	}

	deleteDefinitionNoteAndSidecar(deps, filepath.Join(vault, oldName))
}

// renameTermInVocabList parses the note's current vocab/<term> tags (tags:
// frontmatter, vocabTermsFromTags) and returns the list with fromTerm
// substituted by toTerm. changed=false when the note has no parseable
// frontmatter or its vocab tags do not contain fromTerm.
func renameTermInVocabList(raw []byte, fromTerm, toTerm string) ([]string, bool) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return nil, false
	}

	currentTerms := vocabTermsFromTags(parseTagsFromFrontmatter(string(frontmatter)))

	renamed := make([]string, len(currentTerms))
	changed := false

	for i, term := range currentTerms {
		if term == fromTerm {
			renamed[i] = toTerm
			changed = true

			continue
		}

		renamed[i] = term
	}

	return renamed, changed
}

// renderDefinitionNoteContent renders a definition note's content: the
// standard fact frontmatter — marshaled through factFrontmatterDoc, never
// hand-rendered, so its declared field order is the render order — plus the
// standard fact body formula (renderFactBody) and a verbatim Exemplars:
// section when exemplars are supplied. The body IS the note's embedding
// text: exemplars move the term's vector toward its members' neighborhood
// (see AssignVocabTerms). vocabVersion is "" for a term definition (only the
// family note carries the vault-wide version).
//
// renderFactBody's formula always appends its own trailing period after
// f.Object ("... covers <object>."); when the caller-supplied description
// already ends in "." (a real vault shape — e.g. cost-optimization's stored
// description) that formula would double-punctuate ("...loss..") (#678
// Task 7 FIX 3). Fixed source-side: the body call gets a COPY of f with one
// trailing period trimmed off Object, so the rendered sentence never
// double-punctuates; the frontmatter render above already ran against the
// original f, so the note's object: field still carries the description
// verbatim, trailing period included.
func renderDefinitionNoteContent(f factFields, vocabVersion string, exemplars []string, when time.Time) string {
	f.VocabVersion = vocabVersion

	frontmatter := renderFactFrontmatter(f, when)

	bodyFields := f
	bodyFields.Object = strings.TrimSuffix(f.Object, ".")

	content := frontmatter + renderFactBody(bodyFields)

	if len(exemplars) == 0 {
		return content
	}

	var body strings.Builder

	body.WriteString("Exemplars:\n")

	for _, exemplar := range exemplars {
		body.WriteString("- " + exemplar + "\n")
	}

	return content + body.String()
}

// renderVocabVersionLine renders a standalone "vocab_version: ..." YAML line
// via yaml.Marshal (the same quoting yaml.v3 applies to factFrontmatterDoc's
// VocabVersion field — a bare numeric-looking value like "6.1" is quoted),
// with the trailing newline stripped for insertYAMLBlock.
func renderVocabVersionLine(newVersion string) string {
	body, _ := yaml.Marshal(definitionNoteFields{VocabVersion: newVersion})

	return strings.TrimSuffix(string(body), "\n")
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
		if isVocabRewriteExcluded(name) {
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

// rewriteVocabVersionKey replaces the vocab_version frontmatter value with
// newVersion in place (same line position), leaving every other frontmatter
// key and the body untouched. Appends the key at the end of frontmatter when
// absent (defensive — the family note is expected to already carry it).
// Reuses yaml.Marshal's quoting so the rewritten line matches the exact
// convention the rest of the codebase writes (e.g. `vocab_version: "6.1"`).
func rewriteVocabVersionKey(content, newVersion string) string {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		return content
	}

	insertAt := yamlKeyLineIndex(frontmatter, "vocab_version")
	frontmatter = removeYAMLKey(frontmatter, "vocab_version")
	frontmatter = insertYAMLBlock(frontmatter, renderVocabVersionLine(newVersion), insertAt)

	return fmStart + frontmatter + fmEnd + body
}

// slugFromNoteFilename extracts the <slug> segment from a note filename of
// the form "<id>.<date>.<slug>.md": everything after the first two
// dot-separated segments (id, date) and before the ".md" extension — id and
// date never contain dots (luhmann.FromBasename IDs are digit/letter only;
// dateFormat is "2006-01-02"), so the third SplitN segment carries the whole
// slug even if the slug itself contains dots. Returns "" for a non-.md
// filename or one with fewer than three dot-separated segments.
func slugFromNoteFilename(name string) string {
	const mdExt = ".md"

	if !strings.HasSuffix(name, mdExt) {
		return ""
	}

	base := strings.TrimSuffix(name, mdExt)

	parts := strings.SplitN(base, ".", definitionNoteSlugSegments)
	if len(parts) != definitionNoteSlugSegments {
		return ""
	}

	return parts[2]
}

// sumCounts returns the total of all values in a map[string]int.
func sumCounts(m map[string]int) int {
	total := 0

	for _, v := range m {
		total += v
	}

	return total
}

// termFromDefinitionSlug parses a definition note's slug into its term name:
// "vocab-retrieval-design-definition" → ("retrieval-design", true). Returns
// ("", false) for the family slug ("vocab-definition") and any slug that does
// not match the "vocab-<term>-definition" shape (missing prefix/suffix, or a
// term-less "vocab--definition"). Terms may themselves contain dashes:
// "vocab-skill-and-guidance-design-definition" → "skill-and-guidance-design".
func termFromDefinitionSlug(slug string) (string, bool) {
	if slug == vocabFamilySlug {
		return "", false
	}

	if !strings.HasPrefix(slug, vocabDefinitionPrefix) || !strings.HasSuffix(slug, vocabDefinitionSuffix) {
		return "", false
	}

	term := strings.TrimSuffix(strings.TrimPrefix(slug, vocabDefinitionPrefix), vocabDefinitionSuffix)
	if term == "" {
		return "", false
	}

	return term, true
}

// vocabLifecycleSource renders the source: field for a lifecycle-minted
// definition note: "vocab lifecycle (<site> v<version>)" — e.g. "vocab
// lifecycle (refit v7.0)" (the #678 Task 5 brief's concrete example).
func vocabLifecycleSource(site, version string) string {
	return fmt.Sprintf("vocab lifecycle (%s v%s)", site, version)
}

// writeAndEmbedDefinitionNote renders (renderDefinitionNoteContent — through
// factFrontmatterDoc, never hand-rendered), writes, and embeds a definition
// note at path. Shared by a fresh mint (mintDefinitionNote) and a rename
// (renameDefinitionNote): a rename's body text embeds the NEW term name, so
// its sidecar must be regenerated from the re-rendered body here — never
// copied from the old note's sidecar, which would carry a stale vector +
// ContentHash.
func writeAndEmbedDefinitionNote(
	ctx context.Context,
	deps VocabDeps,
	path string,
	f factFields,
	vocabVersion string,
	exemplars []string,
	when time.Time,
) error {
	content := renderDefinitionNoteContent(f, vocabVersion, exemplars, when)

	writeErr := deps.WriteFile(path, []byte(content))
	if writeErr != nil {
		return fmt.Errorf("writing definition note %s: %w", path, writeErr)
	}

	embedDefinitionNote(ctx, deps, path, content)

	return nil
}

// writeAndEmbedSeedTerms mints a definition note for each seed term that does
// not already have one in the vault (definitionNoteExistsForTerm) —
// bootstrap's idempotency contract: a second run with the same seed mints
// nothing. names is updated in place so Luhmann id allocation cannot collide
// within the same lock-held loop.
func writeAndEmbedSeedTerms(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	names *[]string,
	seed []SeedTerm,
	when time.Time,
) {
	for _, term := range seed {
		if definitionNoteExistsForTerm(vault, *names, term.Term, deps.ReadFile) {
			continue
		}

		f := definitionNoteFactFields(term.Term, term.Description, vocabLifecycleSource("bootstrap", initialVocabVersion))
		slug := definitionNoteSlug(term.Term)

		mintErr := mintDefinitionNote(ctx, deps, vault, names, slug, f, "", term.Exemplars, when)
		if mintErr != nil && deps.LogWarning != nil {
			deps.LogWarning("vocab bootstrap: embedding %s failed: %v", term.Term, mintErr)
		}
	}
}

// writeVocabVersionToFamilyNote rewrites the vocab_version frontmatter key on
// the vocab-definition family note, in place, leaving every other frontmatter
// key and the body untouched. Returns errVocabFamilyNoteMissing when no
// family note exists in vault — callers log-and-continue (propose/refit are
// mechanical mints; a missing family note is not fatal to the rest of the
// command for a pre-bootstrap vault).
func writeVocabVersionToFamilyNote(
	vault, newVersion string,
	listMD func(string) ([]string, error),
	readFile func(string) ([]byte, error),
	writeFile func(string, []byte) error,
) error {
	names, listErr := listMD(vault)
	if listErr != nil {
		return fmt.Errorf("vocab: listing vault for family note: %w", listErr)
	}

	notePath, raw, findErr := findVocabFamilyNote(vault, names, readFile)
	if findErr != nil {
		return findErr
	}

	updated := rewriteVocabVersionKey(string(raw), newVersion)

	writeErr := writeFile(notePath, []byte(updated))
	if writeErr != nil {
		return fmt.Errorf("vocab: writing family note version: %w", writeErr)
	}

	return nil
}
