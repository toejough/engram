package auto

import (
	"strings"
	"unicode/utf8"
)

// Activity returns the lines that tell what automatic memory did for a session and the user was not shown yet, oldest
// first (793d4p).
type Activity func(session string) []Line

// Line is a line that tells the user what records of what automatic memory did say (793d4p), with the file of each
// record, which is marked once the line has been shown.
type Line struct {
	Text  string
	Files []string
}

// ActivityLines returns the lines that tell what the records of a session say automatic memory did, for the records
// whose line the user was not shown: one line for each of the kinds written, skipped and failed that has a record, in
// that order, each line telling all the records of its kind (793d4p). A record of any other kind says nothing.
func ActivityLines(records []Record) []Line {
	kinds := []string{RecordWritten, RecordSkipped, RecordFailed}
	lines := make([]Line, 0, len(kinds))

	for _, kind := range kinds {
		told := make([]Record, 0, len(records))

		for _, record := range records {
			if record.Kind == kind {
				told = append(told, record)
			}
		}

		if len(told) > 0 {
			lines = append(lines, Line{Text: activityText(kind, told), Files: filesOf(told)})
		}
	}

	return lines
}

// unexported constants.
const (
	// activityClock is the form of the time a line opens with.
	activityClock = "15:04:05"
	// bothEnds is the number of ends of a text that a cut in its middle keeps.
	bothEnds = 2
	// deliveredParts is how many parts the line of what was delivered has at most: the notes and the passages.
	deliveredParts = 2
	// shownLimit is how many characters a line shown to the user holds at most (793d4p).
	shownLimit = 240
)

// activityText is the text of the line that tells what the records of a kind say: the time of the oldest, and a phrase
// that names the notes written or gives the reason of each record, cut in its middle where it is long.
func activityText(kind string, records []Record) string {
	parts := make([]string, 0, len(records))

	for _, record := range records {
		parts = append(parts, partText(record))
	}

	var phrase string

	switch kind {
	case RecordWritten:
		phrase = "wrote the " + noteWord(len(parts)) + " " + strings.Join(parts, ", ")
	case RecordSkipped:
		phrase = "left out: " + strings.Join(parts, "; ")
	default:
		phrase = "failed: " + strings.Join(parts, "; ")
	}

	return squeezed(records[0].At.Format(activityClock)+" "+phrase, shownLimit)
}

// deliveredLine is the line that tells the notes and the passages delivered, cut in its middle where it is long
// (855d8a): the notes, then the passages, joined by `and`.
func deliveredLine(notes, passages []Candidate) string {
	parts := make([]string, 0, deliveredParts)

	if len(notes) > 0 {
		parts = append(parts, "the "+noteWord(len(notes))+" "+idsOf(notes))
	}

	if len(passages) > 0 {
		parts = append(parts, "the "+passageWord(len(passages))+" "+idsOf(passages))
	}

	return squeezed("delivered "+strings.Join(parts, " and "), shownLimit)
}

// filesOf returns the file of each record.
func filesOf(records []Record) []string {
	files := make([]string, 0, len(records))

	for _, record := range records {
		files = append(files, record.File)
	}

	return files
}

// idsOf is the ID of each candidate, written as one line, separated by commas.
func idsOf(candidates []Candidate) string {
	ids := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		ids = append(ids, oneLine(candidate.ID))
	}

	return strings.Join(ids, ", ")
}

// noteWord is note, or notes when there are several.
func noteWord(count int) string {
	if count == 1 {
		return "note"
	}

	return "notes"
}

// partText is what a record adds to the line of its kind: the file name of the note written, or the reason, with the
// lesson it is about when it has one, written as one line and redacted (793d4j, 793d4p).
func partText(record Record) string {
	switch {
	case record.Kind == RecordWritten:
		return oneLine(record.Note)
	case record.Slug == "":
		return oneRedacted(record.Reason)
	case record.Kind == RecordSkipped:
		return "the lesson " + oneLine(record.Slug) + ": " + oneRedacted(record.Reason)
	default:
		return "writing " + oneLine(record.Slug) + ": " + oneRedacted(record.Reason)
	}
}

// passageWord is passage, or passages when there are several.
func passageWord(count int) string {
	if count == 1 {
		return "passage"
	}

	return "passages"
}

// squeezed is the text cut in its middle to limit characters, so that both ends stay: an error names the step at its
// beginning and the cause at its end.
func squeezed(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}

	runes := []rune(text)
	head := (limit - 1) / bothEnds
	tail := limit - 1 - head

	return string(runes[:head]) + "…" + string(runes[len(runes)-tail:])
}
