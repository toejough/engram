package cli

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/toejough/engram/internal/embed"
	"github.com/toejough/engram/internal/luhmann"
	"github.com/toejough/engram/internal/vaultgraph"
)

// RenameRewriteDeps carries the filesystem capabilities RenameAndRewriteReferences
// needs. Placed alongside the other cli-level deps structs (VocabDeps, LearnDeps)
// rather than in internal/vaultgraph because vaultgraph.VaultFS is deliberately
// read-only (ScanVault/ParseWikilinks only) — this helper needs Rename + WriteFile,
// and it reuses cli's existing frontmatter/supersedes string-rewrite conventions
// (splitFrontmatterAndBody, the scrubSupersedesFrontmatter family) rather than
// introducing a second YAML-adjacent parsing path in vaultgraph.
type RenameRewriteDeps struct {
	// ListMD returns the .md filenames (not paths) directly under vault.
	ListMD func(vault string) ([]string, error)
	// ReadFile reads raw bytes from a path (notes AND sidecars).
	ReadFile func(path string) ([]byte, error)
	// WriteFile writes data to path (create or overwrite).
	WriteFile func(path string, data []byte) error
	// Rename moves oldPath to newPath.
	Rename func(oldPath, newPath string) error
}

// RenameAndRewriteReferences renames every vault note whose basename is a key in
// renameMap to its mapped new basename (plus its .vec.json sidecar), updates the
// renamed note's own frontmatter luhmann: field to the new ID, and — in the same
// pass — rewrites every note's [[old-basename]] wikilink (including the legacy
// [[old-basename.md]] form), "Supersedes: [[old-basename]]" body line, and
// frontmatter supersedes: list note: field naming an old basename, to the
// corresponding new basename.
//
// The full renameMap is applied in one pass over freshly-read content, so a note
// that is itself being renamed AND references another note also being renamed in
// this run (cascading renames, design.md Decision 4) resolves its outgoing
// reference to the OTHER note's new basename, never a stale intermediate old
// basename.
//
// A note referencing no renamed basename, and not itself renamed, is left
// completely untouched — WriteFile is never called for it.
//
// renameMap being empty is a no-op: ListMD is not even called.
func RenameAndRewriteReferences(deps RenameRewriteDeps, vault string, renameMap map[string]string) error {
	if len(renameMap) == 0 {
		return nil
	}

	names, err := deps.ListMD(vault)
	if err != nil {
		return fmt.Errorf("listing %s: %w", vault, err)
	}

	for _, name := range names {
		renameErr := renameAndRewriteOneNote(deps, vault, name, renameMap)
		if renameErr != nil {
			return renameErr
		}
	}

	return nil
}

// unexported variables.
var (
	reparentWikilinkPattern = regexp.MustCompile(`\[\[([^\]\n]+)\]\]`)
)

// renameAndRewriteOneNote handles a single vault note: rewrites its references
// (regardless of whether it is itself being renamed), and — if it is being
// renamed — renames the note file and its sidecar and updates its own luhmann:
// frontmatter field.
func renameAndRewriteOneNote(deps RenameRewriteDeps, vault, name string, renameMap map[string]string) error {
	basename, ok := vaultgraph.ParseBasename(name)
	if !ok {
		return nil
	}

	oldPath := filepath.Join(vault, name)

	raw, err := deps.ReadFile(oldPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", oldPath, err)
	}

	updated, refsChanged := rewriteNoteReferences(string(raw), renameMap)

	newBasename, renaming := renameMap[basename]
	if !renaming {
		if !refsChanged {
			return nil
		}

		writeErr := deps.WriteFile(oldPath, []byte(updated))
		if writeErr != nil {
			return fmt.Errorf("writing %s: %w", oldPath, writeErr)
		}

		return nil
	}

	return renameOneNote(deps, oldPath, vault, newBasename, updated)
}

// renameOneNote renames oldPath's note file and its .vec.json sidecar to
// newBasename, updates the note's own luhmann: frontmatter field, and writes
// the (already reference-rewritten) updated content to the new path.
func renameOneNote(deps RenameRewriteDeps, oldPath, vault, newBasename, updated string) error {
	newID, _ := luhmann.FromBasename(newBasename)
	updated = rewriteLuhmannIDField(updated, newID)

	newPath := filepath.Join(vault, newBasename+mdExt)

	renameErr := deps.Rename(oldPath, newPath)
	if renameErr != nil {
		return fmt.Errorf("renaming %s to %s: %w", oldPath, newPath, renameErr)
	}

	// Best-effort: not every note has an embedding sidecar.
	_ = deps.Rename(embed.SidecarPath(oldPath), embed.SidecarPath(newPath))

	writeErr := deps.WriteFile(newPath, []byte(updated))
	if writeErr != nil {
		return fmt.Errorf("writing %s: %w", newPath, writeErr)
	}

	return nil
}

// rewriteLuhmannIDField updates content's frontmatter luhmann: field to newID
// (double-quoted, matching the vault convention — see quotedString). Content
// with no frontmatter, or frontmatter with no luhmann: key, is returned
// unchanged.
func rewriteLuhmannIDField(content, newID string) string {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		return content
	}

	idx := yamlKeyLineIndex(frontmatter, "luhmann")
	if idx == -1 {
		return content
	}

	lines := strings.Split(frontmatter, "\n")
	lines[idx] = fmt.Sprintf("luhmann: %q", newID)

	return fmStart + strings.Join(lines, "\n") + fmEnd + body
}

// rewriteNoteReferences rewrites content's frontmatter supersedes: note: fields
// and every [[old-basename]] (or legacy [[old-basename.md]]) occurrence in the
// body — including "Supersedes: [[old-basename]] — ..." lines, which use the
// same wikilink syntax — to the corresponding new basename per renameMap.
// Returns the possibly-updated content and whether anything changed.
func rewriteNoteReferences(content string, renameMap map[string]string) (string, bool) {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		return rewriteWikilinks(content, renameMap)
	}

	newFrontmatter, frontChanged := rewriteSupersedesFrontmatterNotes(frontmatter, renameMap)

	newBody, bodyChanged := rewriteWikilinks(body, renameMap)
	if !frontChanged && !bodyChanged {
		return content, false
	}

	return fmStart + newFrontmatter + fmEnd + newBody, true
}

// rewriteSupersedesFrontmatterNotes rewrites every supersedes: list entry's
// note: value naming an old basename (with or without the .md suffix — the
// convention is to store the full filename, but both forms are tolerated) to
// the corresponding new basename, preserving whichever suffix form was present.
func rewriteSupersedesFrontmatterNotes(frontmatter string, renameMap map[string]string) (string, bool) {
	lines := strings.Split(frontmatter, "\n")
	changed := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "- "))

		value, found := strings.CutPrefix(trimmed, "note:")
		if !found {
			continue
		}

		noteValue := strings.TrimSpace(value)
		hadMDSuffix := strings.HasSuffix(noteValue, mdExt)
		base := strings.TrimSuffix(noteValue, mdExt)

		newBasename, renaming := renameMap[base]
		if !renaming {
			continue
		}

		newValue := newBasename
		if hadMDSuffix {
			newValue += mdExt
		}

		lines[i] = strings.Replace(line, noteValue, newValue, 1)
		changed = true
	}

	return strings.Join(lines, "\n"), changed
}

// rewriteWikilinks replaces every [[X]] or [[X.md]] occurrence in text where X
// (basename-normalized) is a key of renameMap with [[<new-basename>]].
func rewriteWikilinks(text string, renameMap map[string]string) (string, bool) {
	changed := false

	rewritten := reparentWikilinkPattern.ReplaceAllStringFunc(text, func(match string) string {
		target := strings.TrimSuffix(match[2:len(match)-2], mdExt)

		newBasename, found := renameMap[target]
		if !found {
			return match
		}

		changed = true

		return "[[" + newBasename + "]]"
	})

	return rewritten, changed
}
