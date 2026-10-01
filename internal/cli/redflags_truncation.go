package cli

import (
	"fmt"
	"slices"
	"strings"
)

// unexported constants.
const (
	redFlagsBlockStart = "\nred_flags:\n"
	// redFlagsPreviewBudget is engram query's red_flags preview budget,
	// measured in rendered YAML bytes — not a cap on what a runbook may
	// carry. engram show never applies it (recall-runbook-surfacing,
	// "Show never truncates red flags"); only the query preview callers in
	// this package do.
	redFlagsPreviewBudget = 1200
)

// capRedFlagsForPreview truncates an overlong red_flags list in a query
// preview to fit within redFlagsPreviewBudget, keeping entries from the END
// of the list rather than the start. basename is the note's real basename,
// substituted into the omission marker so `engram show <basename>` (the
// command the marker names) actually resolves and returns the full list —
// engram show itself never truncates (D1/#772).
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
func capRedFlagsForPreview(content, basename string) string {
	blockStart := strings.Index(content, redFlagsBlockStart)
	if blockStart < 0 {
		return content
	}

	listStart := blockStart + len(redFlagsBlockStart)

	entryLines, consumed := scanRedFlagsEntries(content[listStart:])
	if consumed <= redFlagsPreviewBudget {
		return content
	}

	total := len(entryLines)
	kept := entryLines

	// The marker's length depends on how many entries it says were dropped
	// and on the total count, both of which depend on how many entries fit
	// — so fit, measure the real marker, then re-fit against it, until the
	// kept set stops shrinking (converges in at most a few passes, since
	// dropped/total only grow in digit-length rarely).
	for {
		marker := redFlagsOmittedMarker(total-len(kept), total, basename)

		refit := keepNewestEntriesWithinBudget(entryLines, redFlagsPreviewBudget-len(marker))
		if len(refit) == len(kept) {
			break
		}

		kept = refit
	}

	marker := redFlagsOmittedMarker(total-len(kept), total, basename)
	newList := marker + strings.Join(kept, "")

	return content[:listStart] + newList + content[listStart+consumed:]
}

// keepNewestEntriesWithinBudget walks entryLines from the end, keeping
// whole lines until adding the next (older) one would exceed budget.
func keepNewestEntriesWithinBudget(entryLines []string, budget int) []string {
	kept := make([]string, 0, len(entryLines))
	size := 0

	for _, line := range slices.Backward(entryLines) {
		size += len(line)
		if size > budget {
			break
		}

		kept = append([]string{line}, kept...)
	}

	return kept
}

// redFlagsOmittedMarker renders the in-band omission marker naming how many
// entries were dropped, the list's total, and the real `engram show
// <basename>` command that returns every entry (recall-runbook-surfacing,
// "Oversized red_flags list keeps its newest entry in query").
func redFlagsOmittedMarker(dropped, total int, basename string) string {
	return fmt.Sprintf(
		"    - \"[%d EARLIER RED_FLAGS OMITTED — run engram show %s for all %d]\"\n",
		dropped, basename, total,
	)
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
