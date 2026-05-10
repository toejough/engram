// Package vaultgraph builds and analyzes the wikilink graph of an agent-memory vault.
// Public entry point: StartingPoints — emits one canonical wikilink per starting
// point (every MOC + the in-degree winner of each MOC-less connected component).
package vaultgraph

import (
	"regexp"
	"strings"
	"unicode"
)

// LuhmannFromBasename extracts the leading Luhmann ID from a basename of the form
// `<luhmann-id>.<YYYY-MM-DD>.<slug>`. Returns ("", false) when the leading
// dot-segment is not a valid Luhmann ID (starts with a digit, alternating
// digit/letter segments). Used for tie-breaking in component-winner selection.
func LuhmannFromBasename(basename string) (string, bool) {
	dotIdx := strings.IndexByte(basename, '.')
	if dotIdx <= 0 {
		return "", false
	}

	candidate := basename[:dotIdx]
	if !unicode.IsDigit(rune(candidate[0])) {
		return "", false
	}

	for _, r := range candidate {
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) {
			return "", false
		}
	}

	return candidate, true
}

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
// Whitespace-only or empty link bodies are dropped. Self-links and broken-link filtering
// happen later, in the graph builder, where the full note set is available.
func ParseWikilinks(body []byte) []string {
	matches := wikilinkPattern.FindAllSubmatch(body, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))

	for _, m := range matches {
		target := string(m[1])
		if target == "" {
			continue
		}

		if _, dup := seen[target]; dup {
			continue
		}

		seen[target] = struct{}{}

		out = append(out, target)
	}

	return out
}

// unexported constants.
const (
	mdExt = ".md"
)

// unexported variables.
var (
	wikilinkPattern = regexp.MustCompile(`\[\[([^\]\n]+)\]\]`)
)
