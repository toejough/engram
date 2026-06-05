package embed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

// ContentHash returns a sha256: prefixed hex digest of the note's embedded
// text (see Text): the situation: field for episodes, the body otherwise. It
// hashes the same bytes that get embedded so staleness detection tracks the
// embed source — editing an episode's situation changes the hash even when
// the body is byte-identical, marking the stored vector stale.
func ContentHash(raw []byte) string {
	sum := sha256.Sum256(Text(raw))

	return "sha256:" + hex.EncodeToString(sum[:])
}

// ExtractBody returns the markdown body of a note with the leading YAML
// frontmatter block stripped. If the note has no leading frontmatter, it
// is returned unchanged.
//
// Frontmatter format: a leading "---\n" line, arbitrary lines (which may
// themselves be blank), and a closing "---\n" line. Anything after the
// closing delimiter is the body. A single leading blank line after the
// closing delimiter is also stripped so notes whose frontmatter blocks
// differ but whose bodies match produce identical hashes.
func ExtractBody(raw []byte) []byte {
	delim := []byte(frontmatterDelim)
	if !bytes.HasPrefix(raw, delim) {
		return raw
	}

	rest := raw[len(delim):]

	_, body, ok := bytes.Cut(rest, delim)
	if !ok {
		return raw
	}

	return bytes.TrimPrefix(body, []byte("\n"))
}

// Text returns the text that should be embedded for a note. For episode
// notes it returns the trimmed `situation:` frontmatter field, which is
// designed to match task queries. For all other note types (or when
// frontmatter is absent / situation is missing), it falls back to
// ExtractBody.
func Text(raw []byte) []byte {
	delim := []byte(frontmatterDelim)
	if !bytes.HasPrefix(raw, delim) {
		return ExtractBody(raw)
	}

	rest := raw[len(delim):]

	frontmatter, _, ok := bytes.Cut(rest, delim)
	if !ok {
		return ExtractBody(raw)
	}

	noteType := extractFrontmatterField(frontmatter, "type")
	if noteType != "episode" {
		return ExtractBody(raw)
	}

	situation := extractFrontmatterField(frontmatter, "situation")
	if situation == "" {
		return ExtractBody(raw)
	}

	return []byte(situation)
}

// unexported constants.
const (
	frontmatterDelim = "---\n"
)

// extractFrontmatterField scans the frontmatter block (content between
// the two "---\n" delimiters, excluding the delimiters themselves) for a
// top-level `key: value` line and returns the trimmed value. Returns ""
// if no matching key is found.
func extractFrontmatterField(frontmatter []byte, key string) string {
	prefix := []byte(key + ": ")

	for line := range bytes.SplitSeq(frontmatter, []byte("\n")) {
		if bytes.HasPrefix(line, prefix) {
			return string(bytes.TrimSpace(line[len(prefix):]))
		}
	}

	return ""
}
