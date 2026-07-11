package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
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

	definitionsMinted := migrateTermDefinitions(ctx, deps, args.Vault, &names, when)
	familyMinted := ensureVocabFamilyNote(ctx, deps, args.Vault, &names, vocabVersion, when, "migrate-tags")

	membersRewritten := migrateMembers(deps, args.Vault, names)

	hubFilesDeleted := deleteHubNotes(deps, args.Vault, names)
	sidecarsDeleted := deleteHubSidecars(deps, args.Vault)

	familyStatus := "present"
	if familyMinted {
		familyStatus = "minted"
	}

	_, _ = fmt.Fprintf(stdout,
		"members rewritten: %d, definitions minted: %d, family note: %s, hub files deleted: %d, sidecars deleted: %d\n",
		membersRewritten, definitionsMinted, familyStatus, hubFilesDeleted, sidecarsDeleted)

	return nil
}

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

// deleteHubNotes deletes every vocab.*.md hub file (old-shape term notes plus
// the retired vocab.index.md) found in names. Returns the count actually
// deleted.
func deleteHubNotes(deps VocabDeps, vault string, names []string) int {
	deleted := 0

	for _, name := range names {
		if !isVocabKindFilename(name) {
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
// deriving paths from the .md listing, which would miss the orphan. Returns
// the count actually deleted; 0 when ListVecJSON is not wired.
func deleteHubSidecars(deps VocabDeps, vault string) int {
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
// the Luhmann id. Returns the count of definitions actually minted.
func migrateTermDefinitions(ctx context.Context, deps VocabDeps, vault string, names *[]string, when time.Time) int {
	minted := 0

	for _, name := range *names {
		if !isVocabTermFilename(name) {
			continue
		}

		term, description, exemplars, ok := parseLegacyTermNote(vault, name, deps.ReadFile)
		if !ok {
			if deps.LogWarning != nil {
				deps.LogWarning("vocab migrate-tags: skipping unreadable term note %s", name)
			}

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

			continue
		}

		minted++
	}

	return minted
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
