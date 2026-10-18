package auto

import (
	"unicode/utf8"
)

// Activity returns the lines that tell what automatic memory did for a session and the user was not shown yet, oldest
// first (793d4p).
type Activity func(session string) []Line

// Line is a line that tells the user what a record of what automatic memory did says (793d4p), with the file of the
// record, which is marked once the line has been shown.
type Line struct {
	Text string
	File string
}

// ActivityLines returns the lines that tell what the records of a session say automatic memory did, for the records
// whose line the user was not shown, oldest first and five at most (793d4p). A record of the kind nothing says nothing.
func ActivityLines(records []Record) []Line {
	lines := make([]Line, 0, activityLimit)

	for _, record := range records {
		text := activityText(record)
		if text == "" {
			continue
		}

		lines = append(lines, Line{Text: text, File: record.File})

		if len(lines) == activityLimit {
			break
		}
	}

	return lines
}

// unexported constants.
const (
	// activityClock is the form of the time a line opens with.
	activityClock = "15:04:05"
	// activityLimit is how many lines tell what automatic memory did at one prompt.
	activityLimit = 5
	// bothEnds is the number of ends of a text that a cut in its middle keeps.
	bothEnds = 2
	// reasonLimit is how many characters of a reason a line holds at most.
	reasonLimit = 240
)

// activityText is the text of the line that tells what a record says, or nothing for a record that says nothing.
func activityText(record Record) string {
	var phrase string

	switch record.Kind {
	case RecordWritten:
		phrase = "wrote the note " + oneLine(record.Note)
	case RecordSkipped:
		phrase = "left out"

		if record.Slug != "" {
			phrase += " the lesson " + oneLine(record.Slug)
		}

		phrase += ": " + reasonText(record.Reason)
	case RecordFailed:
		phrase = "failed: "

		// a failed record with a slug is the write of a lesson
		if record.Slug != "" {
			phrase += "writing " + oneLine(record.Slug) + ": "
		}

		phrase += reasonText(record.Reason)
	default:
		return ""
	}

	return record.At.Format(activityClock) + " " + phrase
}

// reasonText is a reason written as one line, then redacted, and only then cut (793d4j, 793d4p).
func reasonText(reason string) string {
	return squeezed(oneRedacted(reason), reasonLimit)
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
