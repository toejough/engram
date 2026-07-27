package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/toejough/engram/internal/embed"
)

// unexported constants.
const (
	// vocabIndexFilename is the retired pre-tags vocab hub file (#678):
	// enumerated every term. Never rewritten by regenVocab, only removed.
	vocabIndexFilename = "vocab.index.md"
)

// legacyVocabTermFrontmatter is the raw frontmatter shape of a pre-tags
// vocab.<term>.md term note (type: vocab). Read-only here, to harvest seed
// content for --regen-vocab (#712) — never resurrects the retired
// `vocab migrate-tags` converter (#681) and never writes this shape back.
type legacyVocabTermFrontmatter struct {
	Term        string `yaml:"term"`
	Description string `yaml:"description"`
}

// vocabRegenResult reports what regenVocab did — or, under dry-run, would
// do — for the update report to render (#712).
type vocabRegenResult struct {
	OldFilesRemoved int
	MembersCleaned  int
	TermsSeeded     int
	NotesAssigned   int
}

// cleanLegacyMembers strips the legacy vocab: frontmatter key / Vocab: body
// line from every non-hub note in names, returning the count of notes that
// carried one. dryRun leaves the vault untouched (read-only pass, used both
// for the dry-run report and to size the real pass before writing).
func cleanLegacyMembers(deps VocabDeps, vault string, names []string, dryRun bool) int {
	cleaned := 0

	for _, name := range names {
		if isOldVocabHubFilename(name) {
			continue
		}

		notePath := filepath.Join(vault, name)

		raw, readErr := deps.ReadFile(notePath)
		if readErr != nil || len(raw) == 0 {
			continue
		}

		updated, changed := stripLegacyVocabChannel(string(raw))
		if !changed {
			continue
		}

		cleaned++

		if dryRun {
			continue
		}

		writeErr := deps.WriteFile(notePath, []byte(updated))
		if writeErr != nil && deps.LogWarning != nil {
			deps.LogWarning("update --regen-vocab: rewriting %s: %v", notePath, writeErr)
		}
	}

	return cleaned
}

// harvestLegacyVocabSeed reads every old-format vocab.<term>.md term note in
// names and returns its term+description+exemplars as bootstrap seed data —
// the live machinery (writeAndEmbedSeedTerms, the same helper
// RunVocabBootstrap uses) mints the current-format definition note from it.
// Unreadable or unparseable term notes are silently skipped (best-effort
// harvesting; a single bad file must never fail the whole regen).
func harvestLegacyVocabSeed(vault string, names []string, readFile func(string) ([]byte, error)) []SeedTerm {
	seed := make([]SeedTerm, 0, len(names))

	for _, name := range names {
		if !isOldVocabTermFilename(name) {
			continue
		}

		term, description, exemplars, ok := parseLegacyTermNote(vault, name, readFile)
		if !ok {
			continue
		}

		seed = append(seed, SeedTerm{Term: term, Description: description, Exemplars: exemplars})
	}

	return seed
}

// isOldVocabHubFilename reports whether name is an old-format vocab hub file
// — either a vocab.<term>.md term note or vocab.index.md — using the same
// prefix+suffix detection as oldVocabFilesPresent (update.go). The ".md"
// suffix guard excludes vocab.centroids.json, a current always-present file.
func isOldVocabHubFilename(name string) bool {
	return strings.HasPrefix(name, oldVocabFilePrefix) && strings.HasSuffix(name, oldVocabFileSuffix)
}

// isOldVocabTermFilename reports whether name is an old-format
// vocab.<term>.md TERM note specifically — a hub filename that is not the
// retired vocab.index.md.
func isOldVocabTermFilename(name string) bool {
	return isOldVocabHubFilename(name) && name != vocabIndexFilename
}

// oldVocabFilePaths returns the full paths of every old-format vocab hub
// file (term notes + vocab.index.md) found in names.
func oldVocabFilePaths(vault string, names []string) []string {
	paths := make([]string, 0, len(names))

	for _, name := range names {
		if isOldVocabHubFilename(name) {
			paths = append(paths, filepath.Join(vault, name))
		}
	}

	return paths
}

// parseExemplarsSection returns the bullet text (the "- " prefix stripped)
// of an "Exemplars:" section in body, preserving order — carried over
// verbatim into the minted definition note.
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
// description, and verbatim exemplar bullets from an old-shape
// vocab.<term>.md term note. ok=false when unreadable, empty, has no
// parseable frontmatter, or the term: key is empty.
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

// regenVocab migrates a vault holding pre-tags vocab.<term>.md /
// vocab.index.md files to the current tags-based format (#712): it harvests
// each old term note's term+description+exemplars as seed data, mints
// current-format definition notes for them via the live bootstrap machinery
// (writeAndEmbedSeedTerms, the same helper RunVocabBootstrap uses), strips
// the legacy vocab: frontmatter key / Vocab: body line from member notes,
// removes the old hub files (+ best-effort sidecar cleanup), and re-tags
// every member fresh via retagAllNotesTwoPass. Assignment is NOT preserved
// verbatim (unlike the retired `vocab migrate-tags` converter, #681) — the
// converter itself is never resurrected; members are re-scored from the
// regenerated term centroids, the live/current assignment path. Honors
// dryRun: nothing is written, embedded, or removed; only counted. A vault
// with nothing to regenerate returns a zero-valued result (cheap no-op).
func regenVocab(ctx context.Context, vault string, deps VocabDeps, dryRun bool) (vocabRegenResult, error) {
	names, listErr := deps.ListMD(vault)
	if listErr != nil {
		return vocabRegenResult{}, fmt.Errorf("update: regen-vocab: listing vault: %w", listErr)
	}

	seed := harvestLegacyVocabSeed(vault, names, deps.ReadFile)
	oldPaths := oldVocabFilePaths(vault, names)
	membersToClean := cleanLegacyMembers(deps, vault, names, true)

	result := vocabRegenResult{
		OldFilesRemoved: len(oldPaths),
		MembersCleaned:  membersToClean,
		TermsSeeded:     len(seed),
	}

	if len(oldPaths) == 0 && membersToClean == 0 {
		return result, nil // nothing to regenerate
	}

	if dryRun {
		return result, nil
	}

	when := deps.Now()

	ensureVocabFamilyNote(ctx, deps, vault, &names, initialVocabVersion, when, "update --regen-vocab")
	writeAndEmbedSeedTerms(ctx, deps, vault, &names, seed, when)

	cleanLegacyMembers(deps, vault, names, false)
	removeOldVocabFiles(deps, oldPaths)

	terms, termsErr := loadTermVectors(vault, deps.ListMD, deps.ReadFile)
	if termsErr != nil {
		return result, fmt.Errorf("update: regen-vocab: loading term vectors: %w", termsErr)
	}

	if len(terms) > 0 {
		refreshedNames, _ := deps.ListMD(vault)
		memberCounts := retagAllNotesTwoPass(deps, vault, terms, DefaultVocabFloor,
			buildLastRefitDoc(vault, refreshedNames, deps.ReadFile, when))
		result.NotesAssigned = sumCounts(memberCounts)
	}

	return result, nil
}

// removeOldVocabFiles deletes every old-format vocab hub file at paths, plus
// a best-effort delete of its embedding sidecar (which may not exist —
// term/index notes are not always embedded, and a missing-sidecar error is
// silently ignored). Returns the count of hub files actually removed.
func removeOldVocabFiles(deps VocabDeps, paths []string) int {
	removed := 0

	for _, path := range paths {
		delErr := deps.DeleteFile(path)
		if delErr != nil {
			if deps.LogWarning != nil {
				deps.LogWarning("update --regen-vocab: removing %s: %v", path, delErr)
			}

			continue
		}

		removed++

		_ = deps.DeleteFile(embed.SidecarPath(path)) // best-effort; sidecar may not exist
	}

	return removed
}

// removeYAMLLine removes the first line beginning with prefix from text,
// reporting whether one was found.
func removeYAMLLine(text, prefix string) (string, bool) {
	lines := strings.Split(text, "\n")
	kept := make([]string, 0, len(lines))
	found := false

	for _, line := range lines {
		if !found && strings.HasPrefix(line, prefix) {
			found = true

			continue
		}

		kept = append(kept, line)
	}

	if !found {
		return text, false
	}

	return strings.Join(kept, "\n"), true
}

// stripLegacyVocabChannel removes a legacy "vocab:" frontmatter key line and
// a legacy "Vocab:" body line from content, reporting whether either was
// present. Neither present returns content unchanged.
func stripLegacyVocabChannel(content string) (string, bool) {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		return content, false
	}

	newFrontmatter, hadKey := removeYAMLLine(frontmatter, "vocab:")
	newBody, hadBodyLine := removeYAMLLine(body, "Vocab:")

	if !hadKey && !hadBodyLine {
		return content, false
	}

	return fmStart + newFrontmatter + fmEnd + newBody, true
}
