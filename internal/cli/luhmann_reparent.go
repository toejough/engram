package cli

import (
	"context"
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

// RebuildNoteSidecars re-embeds each note at paths and overwrites its
// .vec.json sidecar (via writeAmendedSidecar, the same machinery amend and
// register-skills use), so notes whose content RenameAndRewriteReferences
// rewrote are not left stale for `engram embed status`.
func RebuildNoteSidecars(
	ctx context.Context, deps RenameRewriteDeps, embedder embed.Embedder, paths []string,
) error {
	sidecarDeps := AmendDeps{Write: deps.WriteFile, Embedder: embedder}

	for _, path := range paths {
		raw, readErr := deps.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", path, readErr)
		}

		embedErr := writeAmendedSidecar(ctx, sidecarDeps, path, string(raw))
		if embedErr != nil {
			return embedErr
		}
	}

	return nil
}

// RenameAndRewriteReferences renames every vault note whose basename is a key in
// renameMap to its mapped new basename (plus its .vec.json sidecar), updates the
// renamed note's own frontmatter luhmann: field to the new ID, and — in the same
// pass — rewrites every note's [[old-basename]] wikilink (including the legacy
// [[old-basename.md]] form) in its body AND any frontmatter string field,
// "Supersedes: [[old-basename]]" body line, and
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
//
// It returns the (post-rename) path of every note whose references it
// rewrote, in ListMD order: those notes' embedded content changed, so their
// .vec.json sidecars are stale until rebuilt (RebuildNoteSidecars). A renamed
// note whose only change is its luhmann: field is not listed — frontmatter
// outside situation: does not feed embed.ContentHash.
func RenameAndRewriteReferences(
	deps RenameRewriteDeps, vault string, renameMap map[string]string,
) ([]string, error) {
	if len(renameMap) == 0 {
		return nil, nil
	}

	names, err := deps.ListMD(vault)
	if err != nil {
		return nil, fmt.Errorf("listing %s: %w", vault, err)
	}

	rewritten := make([]string, 0, len(names))

	for _, name := range names {
		path, renameErr := renameAndRewriteOneNote(deps, vault, name, renameMap)
		if renameErr != nil {
			return nil, renameErr
		}

		if path != "" {
			rewritten = append(rewritten, path)
		}
	}

	return rewritten, nil
}

// unexported variables.
var (
	reparentWikilinkPattern = regexp.MustCompile(`\[\[([^\]\n]+)\]\]`)
)

// renameAndRewriteOneNote handles a single vault note: rewrites its references
// (regardless of whether it is itself being renamed), and — if it is being
// renamed — renames the note file and its sidecar and updates its own luhmann:
// frontmatter field.
//
// It returns the note's final path when its references were rewritten, or ""
// when they were not.
func renameAndRewriteOneNote(
	deps RenameRewriteDeps, vault, name string, renameMap map[string]string,
) (string, error) {
	basename, ok := vaultgraph.ParseBasename(name)
	if !ok {
		return "", nil
	}

	oldPath := filepath.Join(vault, name)

	raw, err := deps.ReadFile(oldPath)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", oldPath, err)
	}

	updated, refsChanged := rewriteNoteReferences(string(raw), renameMap)

	finalPath := oldPath

	newBasename, renaming := renameMap[basename]
	if renaming {
		newPath, renameErr := renameOneNote(deps, oldPath, vault, newBasename, updated)
		if renameErr != nil {
			return "", renameErr
		}

		finalPath = newPath
	} else if refsChanged {
		writeErr := deps.WriteFile(oldPath, []byte(updated))
		if writeErr != nil {
			return "", fmt.Errorf("writing %s: %w", oldPath, writeErr)
		}
	}

	if !refsChanged {
		return "", nil
	}

	return finalPath, nil
}

// renameOneNote renames oldPath's note file and its .vec.json sidecar to
// newBasename, updates the note's own luhmann: frontmatter field, writes the
// (already reference-rewritten) updated content to the new path, and returns
// that new path.
func renameOneNote(deps RenameRewriteDeps, oldPath, vault, newBasename, updated string) (string, error) {
	newID, _ := luhmann.FromBasename(newBasename)
	updated = rewriteLuhmannIDField(updated, newID)

	newPath := filepath.Join(vault, newBasename+mdExt)

	renameErr := deps.Rename(oldPath, newPath)
	if renameErr != nil {
		return "", fmt.Errorf("renaming %s to %s: %w", oldPath, newPath, renameErr)
	}

	// Best-effort: not every note has an embedding sidecar.
	_ = deps.Rename(embed.SidecarPath(oldPath), embed.SidecarPath(newPath))

	writeErr := deps.WriteFile(newPath, []byte(updated))
	if writeErr != nil {
		return "", fmt.Errorf("writing %s: %w", newPath, writeErr)
	}

	return newPath, nil
}

// rewriteLuhmannIDField updates content's frontmatter luhmann: field to newID
// (double-quoted, matching the vault convention — see quotedString). The
// key's entire value is replaced — continuation lines included (a block
// scalar, a plain value on following indented lines, a multi-line quoted
// scalar; see yamlValueEndLine) — so the result is always one
// `luhmann: "<id>"` line and the frontmatter still decodes. Content with no
// frontmatter, or frontmatter with no luhmann: key, is returned unchanged.
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
	end := yamlValueEndLine(lines, idx, "luhmann")

	rewritten := make([]string, 0, len(lines)-(end-idx)+1)
	rewritten = append(rewritten, lines[:idx]...)
	rewritten = append(rewritten, fmt.Sprintf("luhmann: %q", newID))
	rewritten = append(rewritten, lines[end:]...)

	return fmStart + strings.Join(rewritten, "\n") + fmEnd + body
}

// rewriteNoteReferences rewrites content's frontmatter supersedes: note: fields
// and every [[old-basename]] (or legacy [[old-basename.md]]) occurrence anywhere
// in the note — every frontmatter string value (action:, situation:,
// red_flags: items, ...) as well as the body, including "Supersedes:
// [[old-basename]] — ..." lines — to the corresponding new basename per
// renameMap. Returns the possibly-updated content and whether anything changed.
//
// Wikilinks are replaced in the raw text rather than by re-rendering YAML: a
// basename is [a-z0-9.-] only, so swapping one for another inside a plain,
// single-quoted, or double-quoted scalar needs no escaping and cannot break
// the scalar's quoting, and a space-free [[...]] can never straddle a folded
// line break.
func rewriteNoteReferences(content string, renameMap map[string]string) (string, bool) {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		return rewriteWikilinks(content, renameMap)
	}

	newFrontmatter, supersedesChanged := rewriteSupersedesFrontmatterNotes(frontmatter, renameMap)
	newFrontmatter, frontLinksChanged := rewriteWikilinks(newFrontmatter, renameMap)

	newBody, bodyChanged := rewriteWikilinks(body, renameMap)
	if !supersedesChanged && !frontLinksChanged && !bodyChanged {
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

// yamlBlockValueEndLine returns the index one past the last continuation
// line of an unquoted value whose key line is lines[idx]: following indented
// lines (a block scalar, a plain scalar or nested collection continued below
// the key) and interior blank lines, plus column-0 `- ` items when the key
// line holds no value (emptyKeyLine, a compact sequence). Trailing blank
// lines are left in place.
func yamlBlockValueEndLine(lines []string, idx int, emptyKeyLine bool) int {
	end := idx + 1

	for next := idx + 1; next < len(lines); next++ {
		line := lines[next]

		switch {
		case strings.TrimSpace(line) == "":
			continue
		case line[0] == ' ' || line[0] == '\t',
			emptyKeyLine && (line == "-" || strings.HasPrefix(line, "- ")):
			end = next + 1
		default:
			return end
		}
	}

	return end
}

// yamlQuoteCloses reports whether text holds the closing quote of a YAML
// scalar quoted with quote: an unescaped `"` for a double-quoted scalar, or
// a single quote that is not one of a doubled pair (two single quotes
// escape one) for a single-quoted scalar.
func yamlQuoteCloses(text string, quote byte) bool {
	for i := 0; i < len(text); i++ {
		switch {
		case quote == '"' && text[i] == '\\':
			i++
		case text[i] != quote:
			continue
		case quote == '\'' && i+1 < len(text) && text[i+1] == '\'':
			i++
		default:
			return true
		}
	}

	return false
}

// yamlQuotedValueEndLine returns the index one past the line closing the
// quoted scalar rest opens on lines[idx] (rest is the key line's value,
// starting at its opening quote).
func yamlQuotedValueEndLine(lines []string, idx int, rest string) int {
	quote := rest[0]

	if yamlQuoteCloses(rest[1:], quote) {
		return idx + 1
	}

	for next := idx + 1; next < len(lines); next++ {
		if yamlQuoteCloses(lines[next], quote) {
			return next + 1
		}
	}

	return len(lines)
}

// yamlValueEndLine returns the index one past the last line of the value of
// the top-level key whose `key:` line is lines[idx]: a quoted value left
// open on the key line runs through the line that closes it
// (yamlQuotedValueEndLine); any other value runs through its continuation
// lines (yamlBlockValueEndLine).
func yamlValueEndLine(lines []string, idx int, key string) int {
	rest := strings.TrimSpace(strings.TrimPrefix(lines[idx], key+":"))

	if rest != "" && (rest[0] == '"' || rest[0] == '\'') {
		return yamlQuotedValueEndLine(lines, idx, rest)
	}

	return yamlBlockValueEndLine(lines, idx, rest == "")
}
