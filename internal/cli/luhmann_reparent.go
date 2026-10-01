package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

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
// renamed note's own frontmatter luhmann: field to the new ID, appends the old
// basename to the renamed note's aliases (in that same write), and — in the
// same pass — rewrites every note's [[old-basename]] wikilink (including the legacy
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
// Before any rename or write, a pre-flight (preflightRenameNotes) renders
// every renamed note's new content and refuses the whole invocation — one
// joined error naming each refused note — when that content's frontmatter
// does not decode as a YAML mapping (errRenameUndecodable: e.g. the old
// luhmann: value carried an anchor another key aliases) or its luhmann: is
// not the new id (errRenameStaleLuhmann).
//
// A note with CRLF line endings is converted to LF (toLF) before any
// rewrite, but only when it is written anyway — renamed, or its references
// changed — and in that same single write. A CRLF note the rename does not
// otherwise touch is never written.
//
// It returns the (post-rename) path of every note whose references it
// rewrote, and of every note it converted from CRLF, in ListMD order: those
// notes' embedded content changed, so their .vec.json sidecars are stale
// until rebuilt (RebuildNoteSidecars). A renamed LF note whose only changes
// are its luhmann: field and aliases is not listed — frontmatter outside
// situation: does not feed embed.ContentHash.
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

	preflightErr := preflightRenameNotes(deps, vault, names, renameMap)
	if preflightErr != nil {
		return nil, preflightErr
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

// unexported constants.
const (
	aliasesKey = "aliases"
)

// unexported variables.
var (
	// errRenameStaleLuhmann refuses a renamed note whose rewritten luhmann:
	// would not be its new id.
	errRenameStaleLuhmann = errors.New("rename: luhmann: would not be the new id")
	// errRenameUndecodable refuses a renamed note whose rewritten
	// frontmatter would not decode as a YAML mapping.
	errRenameUndecodable    = errors.New("rename: rewritten frontmatter does not decode")
	reparentWikilinkPattern = regexp.MustCompile(`\[\[([^\]\n]+)\]\]`)
)

// appendAliasField appends alias to content's frontmatter aliases: list
// (design D4 H2: a renamed note answers to its old basename) and drops
// current, the note's new basename, from it (a note renamed back to a
// former name is not its own alias). It creates the list when absent and
// leaves content unchanged when the list already holds exactly that or
// content has no frontmatter. Only the aliases: key is touched:
// an existing list, in any YAML sequence style, is replaced in place by the
// block form the frontmatter writer emits; a new list goes before offer:
// when present (the writer's key order), else at the end.
func appendAliasField(content, alias, current string) (string, error) {
	frontmatter, body, ok := splitFrontmatterAndBody(content)
	if !ok {
		return content, nil
	}

	var existing struct {
		Aliases []string `yaml:"aliases"`
	}

	lines := strings.Split(frontmatter, "\n")
	start := yamlKeyLineIndex(frontmatter, aliasesKey)
	end := start

	if start >= 0 {
		end = yamlValueEndLine(lines, start, aliasesKey)

		unmarshalErr := yaml.Unmarshal([]byte(strings.Join(lines[start:end], "\n")), &existing)
		if unmarshalErr != nil {
			return "", fmt.Errorf("parsing aliases: %w", unmarshalErr)
		}
	}

	aliases := slices.DeleteFunc(slices.Clone(existing.Aliases), func(name string) bool { return name == current })
	if !slices.Contains(aliases, alias) {
		aliases = append(aliases, alias)
	}

	if slices.Equal(aliases, existing.Aliases) {
		return content, nil
	}

	existing.Aliases = aliases
	rendered, _ := yaml.Marshal(existing)
	block := strings.TrimSuffix(string(rendered), "\n")

	if start < 0 {
		return fmStart + insertYAMLBlock(frontmatter, block, yamlKeyLineIndex(frontmatter, "offer")) + fmEnd + body, nil
	}

	kept := make([]string, 0, len(lines)-(end-start)+1)
	kept = append(kept, lines[:start]...)
	kept = append(kept, block)
	kept = append(kept, lines[end:]...)

	return fmStart + strings.Join(kept, "\n") + fmEnd + body, nil
}

// checkRenamedNote stamps updated (an LF, reference-rewritten note) as the
// rename of oldBasename to newBasename and returns a refusal naming
// oldBasename when the result's frontmatter does not decode as a YAML
// mapping or its luhmann: is not newBasename's id; nil otherwise.
func checkRenamedNote(updated, oldBasename, newBasename string) error {
	stamped, stampErr := stampRenamedNote(updated, oldBasename, newBasename)
	if stampErr != nil {
		return fmt.Errorf("%w: %s: %w", errRenameUndecodable, oldBasename, stampErr)
	}

	frontmatter, found := splitFrontmatter([]byte(stamped))
	if !found {
		return nil
	}

	var document yaml.Node

	decodeErr := yaml.Unmarshal(frontmatter, &document)
	if decodeErr != nil {
		return fmt.Errorf("%w: %s: %w", errRenameUndecodable, oldBasename, decodeErr)
	}

	if document.Kind == 0 {
		return nil
	}

	if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("%w: %s: frontmatter is not a mapping", errRenameUndecodable, oldBasename)
	}

	var probe struct {
		Luhmann *string `yaml:"luhmann"`
	}

	probeErr := document.Decode(&probe)
	if probeErr != nil {
		return fmt.Errorf("%w: %s: %w", errRenameUndecodable, oldBasename, probeErr)
	}

	newID, _ := luhmann.FromBasename(newBasename)
	if probe.Luhmann != nil && *probe.Luhmann != newID {
		return fmt.Errorf("%w: %s: luhmann: %q, want %q", errRenameStaleLuhmann, oldBasename, *probe.Luhmann, newID)
	}

	return nil
}

// isTopLevelYAMLLine reports whether line starts a top-level frontmatter key:
// it is non-blank and starts neither with indentation nor with a column-0
// sequence item or comment (both of which continue the preceding key).
func isTopLevelYAMLLine(line string) bool {
	if line == "" {
		return false
	}

	switch line[0] {
	case ' ', '\t', '-', '#':
		return false
	default:
		return true
	}
}

// preflightRenameNotes renders the content the rename would write for every
// note in names that renameMap renames — CRLF converted, references
// rewritten, luhmann: set, alias appended (the same steps
// renameAndRewriteOneNote takes) — and refuses each whose frontmatter would
// not decode as a YAML mapping (errRenameUndecodable) or whose luhmann:
// would not be its new id (errRenameStaleLuhmann). Every refusal is
// collected into one joined error, returned before anything is renamed or
// written. A note with no frontmatter, or none with a luhmann: key, passes.
func preflightRenameNotes(deps RenameRewriteDeps, vault string, names []string, renameMap map[string]string) error {
	refusals := make([]error, 0, len(renameMap))

	for _, name := range names {
		basename, ok := vaultgraph.ParseBasename(name)
		if !ok {
			continue
		}

		newBasename, renaming := renameMap[basename]
		if !renaming {
			continue
		}

		path := filepath.Join(vault, name)

		raw, err := deps.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		updated, _ := rewriteNoteReferences(string(toLF(raw)), renameMap)

		refusal := checkRenamedNote(updated, basename, newBasename)
		if refusal != nil {
			refusals = append(refusals, refusal)
		}
	}

	return errors.Join(refusals...)
}

// preflightRenames is preflightRenameNotes over every note in vault, for a
// caller (reparent --dry-run) that previews a rename without applying it.
func preflightRenames(deps RenameRewriteDeps, vault string, renameMap map[string]string) error {
	if len(renameMap) == 0 {
		return nil
	}

	names, err := deps.ListMD(vault)
	if err != nil {
		return fmt.Errorf("listing %s: %w", vault, err)
	}

	return preflightRenameNotes(deps, vault, names, renameMap)
}

// renameAndRewriteOneNote handles a single vault note: converts it from CRLF
// to LF (toLF), rewrites its references (regardless of whether it is itself
// being renamed), and — if it is being renamed — renames the note file and
// its sidecar and updates its own luhmann: frontmatter field. A note neither
// renamed nor carrying a renamed reference is never written, so a CRLF note
// it does not otherwise touch keeps its bytes.
//
// It returns the note's final path when its references were rewritten or a
// renamed note was converted from CRLF, or "" otherwise.
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

	lf := toLF(raw)
	converted := !bytes.Equal(lf, raw)

	updated, refsChanged := rewriteNoteReferences(string(lf), renameMap)

	finalPath := oldPath

	newBasename, renaming := renameMap[basename]
	if renaming {
		newPath, renameErr := renameOneNote(deps, oldPath, vault, basename, newBasename, updated)
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

	// A converted note is written only when renamed or rewritten; either
	// way its embedded content changed (design D5).
	if !refsChanged && (!converted || !renaming) {
		return "", nil
	}

	return finalPath, nil
}

// renameOneNote renames oldPath's note file and its .vec.json sidecar to
// newBasename, updates the note's own luhmann: frontmatter field, appends
// oldBasename to its aliases (design D4 H2), writes the (already
// reference-rewritten) updated content to the new path in one write, and
// returns that new path.
func renameOneNote(deps RenameRewriteDeps, oldPath, vault, oldBasename, newBasename, updated string) (string, error) {
	updated, aliasErr := stampRenamedNote(updated, oldBasename, newBasename)
	if aliasErr != nil {
		return "", fmt.Errorf("recording alias on %s: %w", oldPath, aliasErr)
	}

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
// Only lines inside the top-level supersedes: block are touched: other
// note: values (a parent link's note, design D4) name notes in another vault,
// never a local reference, so a local rename must leave them alone.
func rewriteSupersedesFrontmatterNotes(frontmatter string, renameMap map[string]string) (string, bool) {
	lines := strings.Split(frontmatter, "\n")
	changed := false
	inSupersedes := false

	for i, line := range lines {
		if isTopLevelYAMLLine(line) {
			inSupersedes = strings.TrimRight(line, " ") == "supersedes:"

			continue
		}

		if !inSupersedes {
			continue
		}

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

// stampRenamedNote is a renamed note's own frontmatter edit: luhmann: set to
// newBasename's id and oldBasename appended to its aliases (design D4 H2).
func stampRenamedNote(content, oldBasename, newBasename string) (string, error) {
	newID, _ := luhmann.FromBasename(newBasename)

	return appendAliasField(rewriteLuhmannIDField(content, newID), oldBasename, newBasename)
}

// toLF converts every CRLF line ending in raw to LF; a lone CR is left as
// it is. A rename applies it to a note it writes before any rewrite, so a
// CRLF note's frontmatter splits and is rewritten (design D5).
func toLF(raw []byte) []byte {
	return bytes.ReplaceAll(raw, []byte("\r\n"), []byte("\n"))
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
