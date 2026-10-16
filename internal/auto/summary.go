package auto

import (
	"fmt"
	"strings"
)

// unexported constants.
const (
	summaryPrefix = "Memory activity: "
)

// summaryJob shows the user the one line of the turn (793a5h): the summary of what the System did since the previous
// one, or nothing. A warning is its own line, shown when its step failed, and is not repeated here (793b442). The line
// is built from constants and counts only, never from a note, an offer or a lesson (793b297).
func (e *engine) summaryJob() Done {
	var total Counts

	for _, counts := range e.job.Tally {
		total.Delivered, total.Added = total.Delivered+counts.Delivered, total.Added+counts.Added
		total.Amended, total.Curated = total.Amended+counts.Amended, total.Curated+counts.Curated
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

	if len(phrases) == 0 {
		return Done{}
	}

	return Done{Show: true, Line: summaryPrefix + phrases[0]}
}
