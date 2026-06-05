package cli

import (
	"strings"
)

// parsedEpisodeBody is the decomposition of an episode note body into the
// pieces needed to re-render it in the D6 3-section format. relations are
// "basename|rationale" entries (the same shape episodeRelatedBullets consumes
// via the rationale after "|").
type parsedEpisodeBody struct {
	summary    string
	transcript string
	relations  []string
}

// backtickFenceRun reports the backtick-run length of a line that is nothing
// but a run of at least episodeFenceMin backticks, and whether the line is such
// a fence. A line with any non-backtick rune (or too few backticks) is not a
// fence.
func backtickFenceRun(line string) (run int, ok bool) {
	if len(line) < episodeFenceMin {
		return 0, false
	}

	for _, r := range line {
		if r != backtickRune {
			return 0, false
		}
	}

	return len(line), true
}

// bulletRationale returns the rationale text following the em-dash in a
// relation bullet, or "" when the line has none.
func bulletRationale(line string) string {
	const emDash = " — "

	_, after, found := strings.Cut(line, emDash)
	if !found {
		return ""
	}

	return strings.TrimRight(strings.TrimSpace(after), ".")
}

// findTranscriptFence locates a migrated body's ## Transcript fence: it returns
// the index of the opening fence line (the line after "## Transcript"), the
// fence's backtick-run length, and whether such a structure exists.
func findTranscriptFence(lines []string) (fenceLineIdx, fenceRun int, ok bool) {
	for i := 0; i+1 < len(lines); i++ {
		if lines[i] != episodeTranscriptHeading {
			continue
		}

		run, isFence := backtickFenceRun(lines[i+1])
		if isFence {
			return i + 1, run, true
		}
	}

	return 0, 0, false
}

// headingSectionBefore returns the prose under heading, scanning only the lines
// before limit (the transcript fence), trimmed of surrounding blank lines. The
// section ends at the next "## " heading or at limit.
func headingSectionBefore(lines []string, limit int, heading string) string {
	for i := range limit {
		if lines[i] != heading {
			continue
		}

		var section strings.Builder

		for j := i + 1; j < limit; j++ {
			if strings.HasPrefix(lines[j], "## ") {
				break
			}

			section.WriteString(lines[j])
			section.WriteString("\n")
		}

		return strings.Trim(section.String(), "\n")
	}

	return ""
}

// isRelationBlock reports whether block is a "Related to:" marker followed only
// by "- " bullets (and blank lines) through to the end — the exact shape
// renderRelatedSection emits. A non-bullet line means the marker is verbatim
// transcript text, not an authored relation block.
func isRelationBlock(block string) bool {
	_, after, found := strings.Cut(block, relatedSectionMarker)
	if !found {
		return false
	}

	sawBullet := false

	for line := range strings.SplitSeq(after, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if !strings.HasPrefix(trimmed, "- ") {
			return false
		}

		sawBullet = true
	}

	return sawBullet
}

// matchingFenceCloser returns the index of the first line at or after start
// that is a backtick fence of at least fenceRun backticks — the closer for an
// opener of length fenceRun. Returns len(lines) when no closer exists.
func matchingFenceCloser(lines []string, start, fenceRun int) int {
	for i := start; i < len(lines); i++ {
		run, isFence := backtickFenceRun(lines[i])
		if isFence && run >= fenceRun {
			return i
		}
	}

	return len(lines)
}

// parseEpisodeBody decomposes an episode body, handling both the legacy
// verbatim shape (transcript text, optionally followed by a "Related to:"
// block) and the already-migrated D6 shape (## Summary / fenced ## Transcript
// / ## Related). The same parser handling both shapes is what makes
// migrate-episodes a fixed point: re-parsing a migrated body yields the same
// pieces, so re-rendering produces byte-identical output.
//
// The migrated parse is fence-aware: the verbatim transcript routinely
// contains "## " headings and ``` runs (episodes capture agents editing notes),
// so section boundaries are recognized only OUTSIDE the ## Transcript fence.
// A body is treated as migrated only when a "## Transcript" heading is
// immediately followed by a backtick-fence line; otherwise it is legacy, even
// if the verbatim text merely mentions "## Transcript".
func parseEpisodeBody(body string) parsedEpisodeBody {
	// A migrated body ALWAYS begins with "## Summary\n" (renderEpisodeBody emits
	// it first; embed.ExtractBody leaves it at offset 0). Gating on this prefix
	// before trusting findTranscriptFence means a legacy transcript that merely
	// contains a "## Transcript" line + fence can never be misread as migrated.
	if !strings.HasPrefix(body, episodeSummaryHeading+"\n") {
		return parseLegacyEpisodeBody(body)
	}

	lines := strings.Split(body, "\n")

	transcriptStart, fenceRun, isMigrated := findTranscriptFence(lines)
	if !isMigrated {
		return parseLegacyEpisodeBody(body)
	}

	return parseMigratedEpisodeBody(lines, transcriptStart, fenceRun)
}

// parseLegacyEpisodeBody treats the body as a verbatim transcript, peeling a
// trailing "Related to:" block (if present) into authored relations. The block
// is only recognized when every non-blank line after the marker is a "- "
// bullet (the exact shape renderRelatedSection produced) — a "Related to:"
// followed by prose is verbatim transcript text, not an authored block, so the
// whole body stays the transcript. The transcript is everything before a real
// block.
func parseLegacyEpisodeBody(body string) parsedEpisodeBody {
	idx := strings.LastIndex(body, relatedSectionMarker)
	if idx == -1 || !isRelationBlock(body[idx:]) {
		return parsedEpisodeBody{transcript: strings.TrimRight(body, "\n")}
	}

	transcript := strings.TrimRight(body[:idx], "\n")

	return parsedEpisodeBody{
		transcript: transcript,
		relations:  parseRelationBullets(body[idx:]),
	}
}

// parseMigratedEpisodeBody pulls the three sections out of a D6 body given the
// pre-located transcript fence. The summary is the prose under ## Summary
// (before the transcript), the transcript is the verbatim lines between the
// opening fence (at fenceLineIdx) and the matching closer, and the relations
// come from the ## Related block AFTER the closing fence.
func parseMigratedEpisodeBody(lines []string, fenceLineIdx, fenceRun int) parsedEpisodeBody {
	summary := headingSectionBefore(lines, fenceLineIdx, episodeSummaryHeading)

	closerIdx := matchingFenceCloser(lines, fenceLineIdx+1, fenceRun)
	transcript := ""

	if closerIdx > fenceLineIdx+1 {
		transcript = strings.Join(lines[fenceLineIdx+1:closerIdx], "\n") + "\n"
	}

	relations := parseRelationBullets(relatedBlockAfter(lines, closerIdx))

	return parsedEpisodeBody{
		summary:    summary,
		transcript: transcript,
		relations:  relations,
	}
}

// parseRelationBullets extracts "basename|rationale" entries from a block of
// "- [[basename]] — rationale" bullets. The em-dash separator is optional; a
// bullet with no rationale yields just the basename. Bare-id targets are left
// as-is (caller resolves them to full basenames).
func parseRelationBullets(block string) []string {
	var relations []string

	for line := range strings.SplitSeq(block, "\n") {
		match := wikilinkRE.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		target := match[1]

		rationale := bulletRationale(line)
		if rationale != "" {
			relations = append(relations, target+"|"+rationale)

			continue
		}

		relations = append(relations, target)
	}

	return relations
}

// relatedBlockAfter returns the text from the first "## Related" heading found
// after closerIdx to the end — the authored relation bullets. Returns "" when
// there is no ## Related section after the transcript.
func relatedBlockAfter(lines []string, closerIdx int) string {
	for i := closerIdx + 1; i < len(lines); i++ {
		if lines[i] == episodeRelatedHeading {
			return strings.Join(lines[i+1:], "\n")
		}
	}

	return ""
}
