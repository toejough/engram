package auto

import (
	"fmt"
	"slices"
	"strings"
)

// summaryJob shows the user the one line of the turn (793a5h): the summary of what the System did since the previous
// one, with the warnings of the turn on the same line (793b302), or nothing; a turn with nothing to say clears the
// line a previous turn left, so that a stale count is not read as this turn's. The line is built from constants and
// counts only, never from a note, an offer or a lesson (793b297).
func (e *engine) summaryJob() Done {
	var total Counts

	for _, counts := range e.job.Tally {
		total.Delivered, total.Added = total.Delivered+counts.Delivered, total.Added+counts.Added
		total.Amended, total.Curated = total.Amended+counts.Amended, total.Curated+counts.Curated
	}

	var parts []string

	for _, warning := range e.job.Warnings {
		if !slices.Contains(parts, warning) {
			parts = append(parts, warning)
		}
	}

	kinds := []struct {
		count      int
		noun, verb string
	}{
		{total.Delivered, "item", "delivered"}, {total.Added, "note", "added"}, {total.Amended, "note", "amended"},
		{total.Curated, "offer", "curated"},
	}
	phrases := make([]string, 0, len(kinds))

	for _, part := range kinds {
		if plural := map[bool]string{true: "", false: "s"}[part.count == 1]; part.count > 0 {
			phrases = append(phrases, fmt.Sprintf("%d %s%s %s", part.count, part.noun, plural, part.verb))
		}
	}

	if last := len(phrases) - 1; last > 0 {
		phrases = []string{strings.Join(phrases[:last], ", ") + " and " + phrases[last]}
	}

	if len(phrases) > 0 {
		parts = append(parts, "Memory activity: "+phrases[0])
	}

	line := strings.Join(parts, ". ")

	return Done{Show: line != "" || e.job.LineShown, Line: line, LineShown: line != ""}
}
