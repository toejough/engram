// Package vaultgraph builds and analyzes the wikilink graph of an agent-memory vault.
// Public entry point: StartingPoints — emits one canonical wikilink per starting
// point (every MOC + the in-degree winner of each MOC-less connected component).
package vaultgraph

import (
	"regexp"
	"strings"
)

// ParseBasename returns the filename without its ".md" extension, plus ok=true
// if filename has a ".md" extension. Returns "", false for non-md filenames.
// The basename is the canonical graph-node key — it matches the wikilink target.
func ParseBasename(filename string) (string, bool) {
	if !strings.HasSuffix(filename, mdExt) {
		return "", false
	}

	return strings.TrimSuffix(filename, mdExt), true
}

// ParseWikilinks returns the deduped list of wikilink targets in body, in first-appearance order.
// Whitespace-only or empty link bodies are dropped. Wikilinks inside fenced code blocks (``` or
// ~~~) are skipped, matching Obsidian, so verbatim transcript text cannot manufacture graph edges.
// Self-links and broken-link filtering happen later, in the graph builder, where the full note set
// is available.
func ParseWikilinks(body []byte) []string {
	seen := make(map[string]struct{})

	var (
		out   []string
		fence fenceState
	)

	for line := range strings.SplitSeq(string(body), "\n") {
		// Fence marker lines (openers and closers) never carry resolvable wikilinks, and
		// any line inside an open fence is literal content — skip both.
		if fence.toggle(line) || fence.open {
			continue
		}

		for _, match := range wikilinkPattern.FindAllStringSubmatch(line, -1) {
			target := match[1]
			if target == "" {
				continue
			}

			if _, dup := seen[target]; dup {
				continue
			}

			seen[target] = struct{}{}

			out = append(out, target)
		}
	}

	return out
}

// unexported constants.
const (
	backtickFence = '`'
	mdExt         = ".md"
	// minFenceLength is the shortest run of fence characters that opens or closes a code block.
	minFenceLength = 3
	tildeFence     = '~'
)

// unexported variables.
var (
	wikilinkPattern = regexp.MustCompile(`\[\[([^\]\n]+)\]\]`)
)

// fenceState tracks whether the parser is currently inside a fenced code block, plus the
// fence character and run length of the open fence, so a shorter fence cannot close a longer one.
type fenceState struct {
	open   bool
	char   byte
	length int
}

// toggle inspects a single line and updates fence state, returning true if the line is a fence
// marker (an opener or a matching closer) and therefore should not be scanned for wikilinks.
// A closer matches the open fence's character and runs at least as long, with only trailing
// whitespace after the run (Obsidian-accurate — an info-string line cannot close a block).
func (f *fenceState) toggle(line string) bool {
	char, runLength, rest := leadingFenceRun(line)
	if runLength < minFenceLength {
		return false
	}

	if !f.open {
		f.open = true
		f.char = char
		f.length = runLength

		return true
	}

	if char == f.char && runLength >= f.length && strings.TrimSpace(rest) == "" {
		f.open = false
		f.char = 0
		f.length = 0

		return true
	}

	// Inside a block, a non-matching fence line is literal content, not a marker.
	return false
}

// leadingFenceRun reports the fence character (backtick or tilde), the length of its run, and the
// remainder of the line after the run, for a line whose first non-space content is that run.
// runLength is 0 when the line does not begin (modulo leading spaces) with a fence run.
func leadingFenceRun(line string) (fenceChar byte, runLength int, rest string) {
	trimmed := strings.TrimLeft(line, " ")
	if trimmed == "" {
		return 0, 0, ""
	}

	first := trimmed[0]
	if first != backtickFence && first != tildeFence {
		return 0, 0, ""
	}

	count := 0
	for count < len(trimmed) && trimmed[count] == first {
		count++
	}

	return first, count, trimmed[count:]
}
