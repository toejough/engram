package cli

import "strings"

// unexported constants.
const (
	redFlagsBlockStart    = "\nred_flags:\n"
	redFlagsOmittedMarker = "    - \"[EARLIER RED_FLAGS OMITTED — run `engram show <basename>` for the full list]\"\n"
	redFlagsPreviewBudget = 1200
)

// capRedFlagsForPreview truncates an overlong red_flags list to fit within
// redFlagsPreviewBudget, keeping entries from the END of the list rather
// than the start.
//
// Callers append new red_flags entries (engram learn runbook --red-flag /
// engram amend --red-flag), so the newest entry is conventionally last —
// engram#763's reproduction was exactly this: a runbook's 15th (newest)
// red_flags entry, appended after 14 pre-existing ones, sat past an
// external ~2KB truncation boundary and never reached the agent. Dropping
// from the front instead of the back keeps the entry most likely to matter
// right now.
//
// Content with no red_flags block, or a red_flags list already within
// budget, is returned unchanged (byte-identical) — this function must never
// touch a note that was never at truncation risk.
func capRedFlagsForPreview(content string) string {
	blockStart := strings.Index(content, redFlagsBlockStart)
	if blockStart < 0 {
		return content
	}

	listStart := blockStart + len(redFlagsBlockStart)

	entryLines, consumed := scanRedFlagsEntries(content[listStart:])
	if consumed <= redFlagsPreviewBudget {
		return content
	}

	kept := keepNewestEntriesWithinBudget(entryLines, redFlagsPreviewBudget-len(redFlagsOmittedMarker))
	newList := redFlagsOmittedMarker + strings.Join(kept, "")

	return content[:listStart] + newList + content[listStart+consumed:]
}

// keepNewestEntriesWithinBudget walks entryLines from the end, keeping
// whole lines until adding the next (older) one would exceed budget.
func keepNewestEntriesWithinBudget(entryLines []string, budget int) []string {
	kept := make([]string, 0, len(entryLines))
	size := 0

	for i := len(entryLines) - 1; i >= 0; i-- {
		size += len(entryLines[i])
		if size > budget {
			break
		}

		kept = append([]string{entryLines[i]}, kept...)
	}

	return kept
}

// scanRedFlagsEntries reads consecutive "    - " list-item lines starting at
// the beginning of rest (immediately after the "red_flags:\n" key), and
// returns those lines plus their total combined byte length. Scanning stops
// at the first line that is not a list item — the next frontmatter key, or
// the frontmatter close.
func scanRedFlagsEntries(rest string) ([]string, int) {
	lines := strings.SplitAfter(rest, "\n")

	entryLines := make([]string, 0, len(lines))

	consumed := 0

	for _, line := range lines {
		if !strings.HasPrefix(line, "    - ") {
			break
		}

		entryLines = append(entryLines, line)
		consumed += len(line)
	}

	return entryLines, consumed
}
