package embed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

// BodyText returns the note body (frontmatter stripped). It is the
// body-vector source for every note type.
func BodyText(raw []byte) []byte {
	return ExtractBody(raw)
}

// ContentHash returns a sha256: prefixed hex digest covering BOTH embed
// sources — the situation: field and the body — joined by a 0x00 separator.
// Hashing both means staleness detection tracks either source: editing a
// note's situation OR its body changes the hash, marking the stored dual
// vectors stale.
func ContentHash(raw []byte) string {
	hasher := sha256.New()
	hasher.Write(SituationText(raw))
	hasher.Write([]byte{0})
	hasher.Write(BodyText(raw))

	return "sha256:" + hex.EncodeToString(hasher.Sum(nil))
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

// SituationText returns the `situation:` frontmatter field for any note
// type ("" when absent or unparseable). It is the situation-vector source.
func SituationText(raw []byte) []byte {
	delim := []byte(frontmatterDelim)
	if !bytes.HasPrefix(raw, delim) {
		return nil
	}

	rest := raw[len(delim):]

	frontmatter, _, ok := bytes.Cut(rest, delim)
	if !ok {
		return nil
	}

	situation := extractFrontmatterField(frontmatter, "situation")
	if situation == "" {
		return nil
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
