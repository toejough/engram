package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// VocabMigrateArgs holds parsed flags for `engram vocab migrate-tags`.
type VocabMigrateArgs struct {
	Vault string `targ:"flag,name=vault,env=ENGRAM_VAULT_PATH,desc=vault root (default $XDG_DATA_HOME/engram/vault)"` //nolint:lll // unbreakable env+desc struct-tag string
}

// RunVocabMigrateTags performs the one-shot idempotent migration of vault
// notes from the legacy vocab: frontmatter key + Vocab: body-line + hub-file
// (vocab.<term>.md / vocab.index.md) representation to the #674 tags:
// convention (#678 Task 7). Every step is independently idempotent: a second
// run mints nothing, rewrites nothing, and deletes nothing (all-zero counts,
// family note "present"). Assignment is PRESERVED, never re-scored: each
// member's existing vocab: term list maps verbatim onto tags: vocab/<term>.
//
// Data-safety gate (FIX 1/2): a hub file is deleted ONLY when its minted
// replacement actually exists post-mint — a term whose definition mint
// failed (or whose note was too broken to even identify a term) keeps its
// vocab.<term>.md + sidecar; vocab.index.md keeps its sidecar too unless the
// vocab-definition family note actually exists afterward. Any such failure
// is printed in the (still-accurate) counts line AND fails the run
// (non-zero exit, wrapped error naming what failed) so the operator stops
// and investigates rather than trusting a silent exit-0 that quietly lost a
// term's description+exemplars for good.
func RunVocabMigrateTags(ctx context.Context, args VocabMigrateArgs, deps VocabDeps, stdout io.Writer) error {
	release, lockErr := acquireOptionalLock(deps.Lock, args.Vault)
	if lockErr != nil {
		return fmt.Errorf("vocab migrate-tags: acquiring vault lock: %w", lockErr)
	}

	defer release()

	when := deps.Now()

	names, listErr := deps.ListMD(args.Vault)
	if listErr != nil {
		return fmt.Errorf("vocab migrate-tags: listing vault: %w", listErr)
	}

	vocabVersion := migrationVocabVersion(args.Vault, names, deps.ReadFile, stdout)

	definitionsMinted, defFailures := migrateTermDefinitions(ctx, deps, args.Vault, &names, when)
	familyMinted := ensureVocabFamilyNote(ctx, deps, args.Vault, &names, vocabVersion, when, "migrate-tags")
	familyOK := familyNoteExists(args.Vault, names, deps.ReadFile)

	membersRewritten := migrateMembers(deps, args.Vault, names)

	skipHubFiles := hubFileSkipSet(defFailures, familyOK)

	hubFilesDeleted := deleteHubNotes(deps, args.Vault, names, skipHubFiles)
	sidecarsDeleted := deleteHubSidecars(deps, args.Vault, hubSidecarSkipSet(skipHubFiles))

	familyStatus := "present"

	switch {
	case familyMinted:
		familyStatus = "minted"
	case !familyOK:
		familyStatus = "failed"
	}

	_, _ = fmt.Fprintf(stdout,
		"members rewritten: %d, definitions minted: %d, family note: %s, hub files deleted: %d, sidecars deleted: %d\n",
		membersRewritten, definitionsMinted, familyStatus, hubFilesDeleted, sidecarsDeleted)

	if len(defFailures) > 0 || !familyOK {
		return migrationFailureError(defFailures, familyOK)
	}

	return nil
}

// unexported variables.
var (
	errVocabMigrateDefinitionMintFailed = errors.New("vocab migrate-tags: definition mint failed")
)

// legacyVocabMemberFrontmatter is the raw frontmatter shape of a not-yet-
// migrated member note's legacy vocab: key (inline list, e.g. "vocab: [a,
// b]"). Parsed directly here rather than through a typed helper because the
// member structs (factFrontmatterDoc etc.) no longer carry a Vocab field.
type legacyVocabMemberFrontmatter struct {
	Vocab []string `yaml:"vocab"`
}

// legacyVocabTermFrontmatter is the raw frontmatter shape of an old-shape
// vocab.<term>.md term note (type: vocab). VocabFrontmatter/
// ParseVocabFrontmatter were deleted in #678 Task 6; this migration-only shim
// parses the two fields the migration needs directly from raw frontmatter.
type legacyVocabTermFrontmatter struct {
	Term        string `yaml:"term"`
	Description string `yaml:"description"`
}

// migrateTermFailure records one old-shape vocab.<term>.md hub note whose
// migration did not complete: either the note itself was too broken to even
// identify a term (unreadable, no parseable frontmatter, or no term: key —
// Term is "" in that case) or a parseable term's definition mint errored
// (Term is set). Either way hubFile's .md + sidecar must survive the run
// (#678 Task 7 FIX 1 — see deleteHubNotes/deleteHubSidecars' skip parameter)
// and the run must exit non-zero (migrationFailureError) naming Term when
// known, else hubFile.
type migrateTermFailure struct {
	hubFile string
	term    string
}

// deleteHubNotes deletes every vocab.*.md hub file (old-shape term notes plus
// the retired vocab.index.md) found in names, except any name present in
// skip — #678 Task 7 FIX 1/2's data-safety gate: never delete a hub whose
// replacement definition (or, for vocab.index.md, the family note) failed to
// mint. Returns the count actually deleted.
func deleteHubNotes(deps VocabDeps, vault string, names []string, skip map[string]bool) int {
	deleted := 0

	for _, name := range names {
		if !isVocabKindFilename(name) {
			continue
		}

		if skip[name] {
			continue
		}

		if deleteVaultFile(deps, filepath.Join(vault, name)) {
			deleted++
		}
	}

	return deleted
}

// deleteHubSidecars deletes every vocab.*.vec.json sidecar in vault — hub
// term/index sidecars AND any orphan with no surviving .md counterpart —
// found by listing .vec.json files directly (deps.ListVecJSON) rather than
// deriving paths from the .md listing, which would miss the orphan. skip
// carries SIDECAR filenames (hubSidecarSkipSet's output — the
// embed.SidecarPath of each name deleteHubNotes is also skipping): a hub
// whose .md survives must keep its sidecar too. Returns the count actually
// deleted; 0 when ListVecJSON is not wired.
func deleteHubSidecars(deps VocabDeps, vault string, skip map[string]bool) int {
	if deps.ListVecJSON == nil {
		return 0
	}

	names, listErr := deps.ListVecJSON(vault)
	if listErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab migrate-tags: listing sidecars: %v", listErr)
		}

		return 0
	}

	deleted := 0

	for _, name := range names {
		if !strings.HasPrefix(name, vocabNotePrefix) {
			continue
		}

		if skip[name] {
			continue
		}

		if deleteVaultFile(deps, filepath.Join(vault, name)) {
			deleted++
		}
	}

	return deleted
}

// deleteVaultFile deletes path via deps.DeleteFile, warning-and-returning
// false on a nil dep or a delete error — never fatal to the rest of the
// migration.
func deleteVaultFile(deps VocabDeps, path string) bool {
	if deps.DeleteFile == nil {
		return false
	}

	delErr := deps.DeleteFile(path)
	if delErr != nil {
		if deps.LogWarning != nil {
			deps.LogWarning("vocab migrate-tags: deleting %s: %v", path, delErr)
		}

		return false
	}

	return true
}

// familyNoteExists reports whether the vocab-definition family note is
// present in vault, post-mint — #678 Task 7 FIX 1/2's gate for
// vocab.index.md: the retired index hub (and its sidecar) must never be
// deleted unless its replacement family note actually exists.
func familyNoteExists(vault string, names []string, readFile func(string) ([]byte, error)) bool {
	_, _, findErr := findVocabFamilyNote(vault, names, readFile)

	return findErr == nil
}

// hubFileSkipSet builds RunVocabMigrateTags' skip set for deleteHubNotes: the
// hub filename of every failed term (migrateTermDefinitions' defFailures),
// plus vocab.index.md when the family note it's being replaced by does not
// exist (familyOK false) — #678 Task 7 FIX 1/2.
func hubFileSkipSet(failures []migrateTermFailure, familyOK bool) map[string]bool {
	skip := make(map[string]bool, len(failures)+1)

	for _, failure := range failures {
		skip[failure.hubFile] = true
	}

	if !familyOK {
		skip[vocabIndexFilename] = true
	}

	return skip
}

// hubSidecarSkipSet converts a deleteHubNotes skip set (hub .md filenames)
// into the equivalent deleteHubSidecars skip set (sidecar filenames, via
// embed.SidecarPath) — a hub note that survives must keep its sidecar too.
func hubSidecarSkipSet(skipHubFiles map[string]bool) map[string]bool {
	skip := make(map[string]bool, len(skipHubFiles))

	for name := range skipHubFiles {
		skip[embed.SidecarPath(name)] = true
	}

	return skip
}

// legacyMemberTerms determines a member note's migration term list from its
// raw frontmatter/body. A vocab: key wins — its list IS the preserved
// assignment (migrate=true, no re-scoring). Absent key with a stray Vocab:
// body line is the channel-consistency repair branch: the body-line terms are
// DISCARDED (never parsed as assignment) in favor of whatever vocab/ tags the
// note already carries (empty pre-migration — zero real instances, defensive
// only). Neither channel present means nothing to migrate.
func legacyMemberTerms(frontmatter, body string) (terms []string, migrate bool) {
	if yamlKeyLineIndex(frontmatter, "vocab") >= 0 {
		return parseLegacyVocabList(frontmatter), true
	}

	if removeVocabBodyLine(body) != body {
		return vocabTermsFromTags(parseTagsFromFrontmatter(frontmatter)), true
	}

	return nil, false
}

// migrateMembers rewrites every non-hub note carrying a legacy vocab channel
// (a vocab: frontmatter key, or a stray Vocab: body line with no key) through
// WriteVocabAssignment, preserving the note's existing term assignment
// verbatim. Returns the count of notes actually rewritten.
func migrateMembers(deps VocabDeps, vault string, names []string) int {
	rewritten := 0

	for _, name := range names {
		if isVocabKindFilename(name) {
			continue
		}

		notePath := filepath.Join(vault, name)

		raw, readErr := deps.ReadFile(notePath)
		if readErr != nil || len(raw) == 0 {
			continue
		}

		content := string(raw)

		frontmatter, body, ok := splitFrontmatterAndBody(content)
		if !ok {
			continue
		}

		terms, migrate := legacyMemberTerms(frontmatter, body)
		if !migrate {
			continue
		}

		updated := WriteVocabAssignment(content, terms)

		writeErr := deps.WriteFile(notePath, []byte(updated))
		if writeErr != nil {
			if deps.LogWarning != nil {
				deps.LogWarning("vocab migrate-tags: rewriting %s: %v", notePath, writeErr)
			}

			continue
		}

		rewritten++
	}

	return rewritten
}

// migrateTermDefinitions mints a vocab-<term>-definition note (Task 5's
// shape) for every old-shape vocab.<term>.md term note that has no
// definition note yet (definitionNoteExistsForTerm — idempotent). names is
// updated in place so a mint in this loop cannot collide with a later one on
// the Luhmann id. Returns the count of definitions actually minted, plus one
// migrateTermFailure per term hub note that was unparseable/unreadable OR
// whose mint errored (#678 Task 7 FIX 1) — the caller (RunVocabMigrateTags)
// uses this list to keep each failed term's hub file (and sidecar) out of
// this run's deletion sweep and to fail the run non-zero.
func migrateTermDefinitions(
	ctx context.Context,
	deps VocabDeps,
	vault string,
	names *[]string,
	when time.Time,
) (minted int, failures []migrateTermFailure) {
	failures = make([]migrateTermFailure, 0)

	for _, name := range *names {
		if !isVocabTermFilename(name) {
			continue
		}

		term, description, exemplars, ok := parseLegacyTermNote(vault, name, deps.ReadFile)
		if !ok {
			if deps.LogWarning != nil {
				deps.LogWarning("vocab migrate-tags: skipping unreadable term note %s", name)
			}

			failures = append(failures, migrateTermFailure{hubFile: name})

			continue
		}

		if definitionNoteExistsForTerm(vault, *names, term, deps.ReadFile) {
			continue
		}

		f := definitionNoteFactFields(term, description, fmt.Sprintf("migrated from %s under #678", name))

		mintErr := mintDefinitionNote(ctx, deps, vault, names, definitionNoteSlug(term), f, "", exemplars, when)
		if mintErr != nil {
			if deps.LogWarning != nil {
				deps.LogWarning("vocab migrate-tags: minting definition for %s: %v", term, mintErr)
			}

			failures = append(failures, migrateTermFailure{hubFile: name, term: term})

			continue
		}

		minted++
	}

	return minted, failures
}

// migrationFailureError builds RunVocabMigrateTags' non-zero-exit error for
// #678 Task 7 FIX 1/2: one wrapped error naming every term whose definition
// mint failed (migrateTermFailure.term when the note was parseable, else
// .hubFile for a note too broken to identify a term) plus the family note
// when its own mint failed (familyOK false). The counts line is already
// printed by the caller before this error is returned, so the operator sees
// exactly what succeeded before being told to stop and fix the rest.
func migrationFailureError(failures []migrateTermFailure, familyOK bool) error {
	labels := make([]string, 0, len(failures)+1)

	for _, failure := range failures {
		if failure.term != "" {
			labels = append(labels, failure.term)
		} else {
			labels = append(labels, failure.hubFile)
		}
	}

	if !familyOK {
		labels = append(labels, "vocab-definition (family note)")
	}

	return fmt.Errorf("%w: %s", errVocabMigrateDefinitionMintFailed, strings.Join(labels, ", "))
}

// migrationVocabVersion resolves the vault-wide vocab_version the migration
// stamps onto the family note: vocab.index.md's key when the (not-yet-
// deleted) index file is present, else the vocab-definition family note's key
// (the idempotent-second-run case — the index is gone by then), else the
// default "1.0" with a warning printed to stdout (neither source exists).
func migrationVocabVersion(
	vault string,
	names []string,
	readFile func(string) ([]byte, error),
	stdout io.Writer,
) string {
	if version, ok := vocabVersionFromIndexNote(vault, names, readFile); ok {
		return version
	}

	if version, ok := vocabVersionFromFamilyNote(vault, names, readFile); ok {
		return version
	}

	_, _ = fmt.Fprintf(stdout,
		"warning: vocab migrate-tags: no vocab.index.md or vocab-definition family note found; "+
			"defaulting vocab_version to %s\n", initialVocabVersion)

	return initialVocabVersion
}

// parseExemplarsSection returns the bullet text (the "- " prefix stripped) of
// an "Exemplars:" section in body, preserving order — carried over verbatim
// into the minted definition note (renderDefinitionNoteContent re-renders an
// identical "- <text>" line per entry). Returns nil when no Exemplars: marker
// is present.
func parseExemplarsSection(body string) []string {
	const marker = "Exemplars:"

	_, after, found := strings.Cut(body, marker)
	if !found {
		return nil
	}

	lines := strings.Split(after, "\n")
	exemplars := make([]string, 0, len(lines))

	for _, line := range lines {
		item, isBullet := strings.CutPrefix(line, "- ")
		if !isBullet {
			continue
		}

		exemplars = append(exemplars, item)
	}

	return exemplars
}

// parseLegacyTermNote reads name's raw content and extracts the term,
// description, and verbatim exemplar bullets (parseExemplarsSection) from an
// old-shape vocab.<term>.md term note. ok=false when unreadable, empty, has
// no parseable frontmatter, or the term: key is empty.
func parseLegacyTermNote(
	vault, name string,
	readFile func(string) ([]byte, error),
) (term, description string, exemplars []string, ok bool) {
	raw, readErr := readFile(filepath.Join(vault, name))
	if readErr != nil || len(raw) == 0 {
		return "", "", nil, false
	}

	frontmatter, body, fmOK := splitFrontmatterAndBody(string(raw))
	if !fmOK {
		return "", "", nil, false
	}

	var doc legacyVocabTermFrontmatter

	unmarshalErr := yaml.Unmarshal([]byte(frontmatter), &doc)
	if unmarshalErr != nil || doc.Term == "" {
		return "", "", nil, false
	}

	return doc.Term, doc.Description, parseExemplarsSection(body), true
}

// parseLegacyVocabList parses the raw inline vocab: [a, b] list from
// frontmatter. Returns nil when absent or unparseable.
func parseLegacyVocabList(frontmatter string) []string {
	var doc legacyVocabMemberFrontmatter

	unmarshalErr := yaml.Unmarshal([]byte(frontmatter), &doc)
	if unmarshalErr != nil {
		return nil
	}

	return doc.Vocab
}

// vocabVersionFromFamilyNote reads vocab_version from the vocab-definition
// family note, when present.
func vocabVersionFromFamilyNote(
	vault string,
	names []string,
	readFile func(string) ([]byte, error),
) (string, bool) {
	_, raw, findErr := findVocabFamilyNote(vault, names, readFile)
	if findErr != nil {
		return "", false
	}

	return vocabVersionFromNoteBytes(raw)
}

// vocabVersionFromIndexNote reads vocab_version from vocab.index.md, when
// present in names.
func vocabVersionFromIndexNote(
	vault string,
	names []string,
	readFile func(string) ([]byte, error),
) (string, bool) {
	for _, name := range names {
		if name != vocabIndexFilename {
			continue
		}

		raw, readErr := readFile(filepath.Join(vault, name))
		if readErr != nil {
			return "", false
		}

		return vocabVersionFromNoteBytes(raw)
	}

	return "", false
}

// vocabVersionFromNoteBytes parses the vocab_version frontmatter key from raw
// note bytes. ok=false when the frontmatter is unparseable or the key is
// empty.
func vocabVersionFromNoteBytes(raw []byte) (string, bool) {
	frontmatter, ok := splitFrontmatter(raw)
	if !ok {
		return "", false
	}

	var doc definitionNoteFields

	unmarshalErr := yaml.Unmarshal(frontmatter, &doc)
	if unmarshalErr != nil || doc.VocabVersion == "" {
		return "", false
	}

	return doc.VocabVersion, true
}
