package embed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"slices"
)

// Exported constants.
const (
	RelatedSectionMarker = "Related to:"
)

// BodyText returns the note body (frontmatter stripped) with any trailing
// "Related to:" section removed. It is the body-vector source for every
// note type. Dropping the relation block means a link-only edit (adding or
// changing [[wikilinks]] under "Related to:") leaves the body vector and
// ContentHash unchanged (D3).
func BodyText(raw []byte) []byte {
	return stripRelatedToSection(ExtractBody(raw))
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
	// relatedSectionBulletPfx is the prefix of every rendered relation bullet
	// ("- [[target]] — rationale."). A "Related to:" marker line counts as a
	// block only when every following non-blank line starts with this prefix.
	relatedSectionBulletPfx = "- [["
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

// isRelatedToBlock reports whether the lines that follow a "Related to:"
// marker form a relation block: every non-blank line must start with
// relatedSectionBulletPfx, and at least one bullet must be present. A line
// that is neither blank nor a bullet (prose) disqualifies the block, so an
// inline "Related to:" mention is not stripped.
func isRelatedToBlock(after [][]byte) bool {
	sawBullet := false

	for _, line := range after {
		trimmed := bytes.TrimRight(line, "\r")
		if len(bytes.TrimSpace(trimmed)) == 0 {
			continue
		}

		if !bytes.HasPrefix(trimmed, []byte(relatedSectionBulletPfx)) {
			return false
		}

		sawBullet = true
	}

	return sawBullet
}

// stripRelatedToSection removes a trailing "Related to:" relation block from
// body, returning body unchanged when no such block is present. The block is
// recognised conservatively (see isRelatedToBlock): a "Related to:" marker
// line whose following non-blank lines are all relation bullets. Recognising
// only the LAST marker, and only when the lines after it qualify, leaves prose
// that mentions "Related to:" inline untouched.
//
// Implementation note: bytes.Split(body, "\n") on a newline-terminated body
// produces a trailing empty element. Lines[:i] for i pointing at the marker
// therefore ends with the blank line(s) before the marker — joining with "\n"
// faithfully restores the body up to and including its final trailing newline.
// Do NOT bytes.TrimRight the result: that would remove the single trailing
// newline that is part of the body (CA-15 fix).
func stripRelatedToSection(body []byte) []byte {
	lines := bytes.Split(body, []byte("\n"))
	// bytes.Split never returns nil in practice, but nilaway cannot prove that
	// and flags the lines[i+1:] / lines[:i] indexing below; the guard satisfies
	// it without a //nolint suppression (project rule: fix, don't suppress).
	if lines == nil {
		return body
	}

	for i, line := range slices.Backward(lines) {
		if bytes.Equal(bytes.TrimRight(line, "\r"), []byte(RelatedSectionMarker)) {
			if isRelatedToBlock(lines[i+1:]) {
				result := bytes.Join(lines[:i], []byte("\n"))
				// Restore the trailing newline when no blank line preceded the
				// marker (lines[:i] joined without a trailing empty element
				// would otherwise drop the newline that ended the last body line).
				if len(result) > 0 && result[len(result)-1] != '\n' {
					result = append(result, '\n')
				}

				return result
			}
		}
	}

	return body
}
