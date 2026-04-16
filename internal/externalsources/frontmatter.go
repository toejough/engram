package externalsources

import (
	"bytes"
	"strings"
)

// Frontmatter holds the YAML frontmatter fields engram extracts from rules
// and skill markdown files. Only fields engram actually uses are populated.
type Frontmatter struct {
	Name        string
	Description string
	Paths       []string
}

// ParseFrontmatter splits a markdown file into its frontmatter (if any) and
// the remaining body bytes. Files without a leading "---" line return an
// empty Frontmatter and the original body unchanged.
//
// This is intentionally a small, hand-rolled parser covering only the fields
// engram cares about (name, description, paths). It supports literal-scalar
// strings, the ">" folded-block style for description, and the "- item"
// list style for paths.
func ParseFrontmatter(body []byte) (Frontmatter, []byte) {
	if !bytes.HasPrefix(body, []byte(yamlFence+"\n")) {
		return Frontmatter{}, body
	}

	rest := body[len(yamlFence)+1:]

	closingFence := []byte("\n" + yamlFence + "\n")

	yamlBytes, remainder, found := bytes.Cut(rest, closingFence)
	if !found {
		return Frontmatter{}, body
	}

	return parseYAMLBlock(string(yamlBytes)), remainder
}

// unexported constants.
const (
	yamlFence = "---"
)

// parserState captures the in-progress frontmatter and any continuation
// context (folded scalar accumulator, list mode) between lines.
type parserState struct {
	matter              Frontmatter
	foldedLines         []string
	inFoldedDescription bool
	inPathsList         bool
}

func (state *parserState) consumeLine(raw string) {
	trimmed := strings.TrimSpace(raw)

	if state.continueFolded(raw, trimmed) {
		return
	}

	if state.continuePathsList(trimmed) {
		return
	}

	state.startKey(trimmed)
}

// continueFolded extends a ">" folded-block scalar when the line is indented;
// otherwise it flushes the accumulated value and lets the caller try other
// continuations or new keys. It returns true when the line was consumed.
func (state *parserState) continueFolded(raw, trimmed string) bool {
	if !state.inFoldedDescription {
		return false
	}

	if strings.HasPrefix(raw, "  ") || strings.HasPrefix(raw, "\t") {
		state.foldedLines = append(state.foldedLines, trimmed)

		return true
	}

	state.flushFolded()

	return false
}

// continuePathsList appends a "- item" line to the paths slice when in list
// mode; any other line ends the list. It returns true when the line was
// consumed as a list item.
func (state *parserState) continuePathsList(trimmed string) bool {
	if !state.inPathsList {
		return false
	}

	if item, ok := strings.CutPrefix(trimmed, "- "); ok {
		state.matter.Paths = append(state.matter.Paths, strings.Trim(item, `"'`))

		return true
	}

	state.inPathsList = false

	return false
}

// flushFolded writes any accumulated folded-block lines into the description
// and resets the accumulator. Safe to call when no folded block is in flight.
func (state *parserState) flushFolded() {
	if !state.inFoldedDescription {
		return
	}

	state.matter.Description = strings.Join(state.foldedLines, " ")
	state.inFoldedDescription = false
	state.foldedLines = nil
}

// startKey dispatches a fresh key line. Only the keys engram cares about
// (name, description, paths) are recognised; everything else is ignored.
func (state *parserState) startKey(trimmed string) {
	switch trimmed {
	case "description: >":
		state.inFoldedDescription = true

		return
	case "paths:":
		state.inPathsList = true

		return
	}

	if value, ok := strings.CutPrefix(trimmed, "name:"); ok {
		state.matter.Name = strings.TrimSpace(value)

		return
	}

	if value, ok := strings.CutPrefix(trimmed, "description:"); ok {
		state.matter.Description = strings.TrimSpace(value)
	}
}

func newParserState() *parserState {
	return &parserState{}
}

// parseYAMLBlock walks a frontmatter YAML block line-by-line, tracking whether
// the current line continues a multi-line value (folded description or paths
// list) before dispatching scalar keys.
func parseYAMLBlock(yamlBlock string) Frontmatter {
	state := newParserState()

	for raw := range strings.SplitSeq(yamlBlock, "\n") {
		state.consumeLine(raw)
	}

	state.flushFolded()

	return state.matter
}
