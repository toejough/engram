package embed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"slices"
)

// Exported constants.
const (
	// AnsweredByBodyMarker prefixes the machine-written `Answered by: [[…]]` body
	// line on QA question notes. Same exclusion rationale as VocabBodyMarker.
	AnsweredByBodyMarker = "Answered by:"
	// AnswersBodyMarker prefixes the machine-written `Answers: [[…]]` body line on
	// QA answer notes. Same exclusion rationale as VocabBodyMarker.
	AnswersBodyMarker = "Answers:"
	// ContributorsBodyMarker prefixes the machine-written `Contributors: [[…]], …`
	// body line on QA answer notes. Excluded from BodyText/ContentHash so a
	// contributors-only write leaves the body vector and hash unchanged.
	ContributorsBodyMarker = "Contributors:"
	// RelatedSectionMarker is retained for backward compatibility: unmigrated
	// vault bodies still carry "Related to:" sections (the ritual was removed
	// 2026-07-02; bodies are stripped by the vocab migration). Hash exclusion
	// must keep working for them until that migration lands, then this can go.
	RelatedSectionMarker = "Related to:"
	// SupersedesBodyMarker prefixes the machine-written `Supersedes: [[…]] —
	// type: claim` body lines (replace-whole channel, written by learn/amend).
	// Excluded from BodyText/ContentHash for the same reason as VocabBodyMarker.
	// The cli writer's line matching aliases this constant — keep them in sync.
	SupersedesBodyMarker = "Supersedes:"
	// VocabBodyMarker prefixes the machine-written `Vocab: [[vocab.…]]` body
	// line (replace-whole channel, written by WriteVocabAssignment AFTER a
	// note is embedded). Excluding it from BodyText/ContentHash keeps a
	// vocab-assigning write from staling the sidecar and keeps [[vocab.…]]
	// wikilink noise out of body vectors on re-embed. The cli writer's line
	// matching aliases this constant — keep them in sync.
	VocabBodyMarker = "Vocab:"
)

// BodyText returns the note body (frontmatter stripped) with all
// machine-written channel content removed: `Vocab:` and `Supersedes:` body
// lines (replace-whole channels) and any trailing "Related to:" section.
// It is the body-vector source for every note type. Dropping channel content
// means a channel-only edit (vocab assignment, supersession write, link edit)
// leaves the body vector and ContentHash unchanged (D3).
//
// Machine lines are stripped BEFORE the Related-to pass: the writers append
// their lines after an unmigrated note's trailing "Related to:" block, and a
// non-bullet line after the block would otherwise disqualify it.
//
// Trailing blank lines are normalized to a single newline LAST: the learn
// renderers end bodies with "\n\n" while the channel writers trim trailing
// blanks before appending their line — the original count is unrecoverable
// after a write, so the hash must be insensitive to it on both sides.
func BodyText(raw []byte) []byte {
	return normalizeTrailingBlanks(stripRelatedToSection(stripMachineLines(ExtractBody(raw))))
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

// isMachineLine reports whether a body line (CRLF-stripped) is a
// machine-written channel line that should be excluded from BodyText and
// ContentHash. Recognised prefixes: Vocab:, Supersedes:, Contributors:,
// Answered by:, and Answers:.
func isMachineLine(trimmed []byte) bool {
	return bytes.HasPrefix(trimmed, []byte(VocabBodyMarker)) ||
		bytes.HasPrefix(trimmed, []byte(SupersedesBodyMarker)) ||
		bytes.HasPrefix(trimmed, []byte(ContributorsBodyMarker)) ||
		bytes.HasPrefix(trimmed, []byte(AnsweredByBodyMarker)) ||
		bytes.HasPrefix(trimmed, []byte(AnswersBodyMarker))
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

// normalizeTrailingBlanks trims trailing BLANK LINES from body, restoring a
// single trailing newline when any content remains. Bodies already ending in
// exactly one newline (or none) are returned byte-identical — only the blank
// lines themselves are normalized, never the last content line.
func normalizeTrailingBlanks(body []byte) []byte {
	lines := bytes.Split(body, []byte("\n"))
	// bytes.Split never returns nil in practice; the guard satisfies nilaway.
	if lines == nil {
		return body
	}

	end := len(lines)
	for end > 0 && len(bytes.TrimSpace(lines[end-1])) == 0 {
		end--
	}

	// No trailing newline at all, or exactly one — already normal.
	if end == len(lines) || end == len(lines)-1 && len(lines[len(lines)-1]) == 0 {
		return body
	}

	if end == 0 {
		return nil
	}

	result := bytes.Join(lines[:end], []byte("\n"))

	return append(result, '\n')
}

// stripMachineLines removes machine-written channel lines (`Vocab:`,
// `Supersedes:`, `Contributors:`, `Answered by:`, and `Answers:` prefixes —
// exactly the writers' replace-whole line matching) from body. When any line
// is removed, trailing blank lines are trimmed and a single trailing newline
// restored, mirroring the writers' append form ("body\n" →
// "body\n\nVocab: …\n" must strip back to "body\n"). A body with no machine
// lines is returned byte-identical so pre-channel hashes never churn.
func stripMachineLines(body []byte) []byte {
	lines := bytes.Split(body, []byte("\n"))
	kept := make([][]byte, 0, len(lines))
	removed := false

	for _, line := range lines {
		trimmed := bytes.TrimRight(line, "\r")
		if isMachineLine(trimmed) {
			removed = true

			continue
		}

		kept = append(kept, line)
	}

	if !removed {
		return body
	}

	for len(kept) > 0 && len(bytes.TrimSpace(kept[len(kept)-1])) == 0 {
		kept = kept[:len(kept)-1]
	}

	result := bytes.Join(kept, []byte("\n"))
	if len(result) > 0 && result[len(result)-1] != '\n' {
		result = append(result, '\n')
	}

	return result
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
