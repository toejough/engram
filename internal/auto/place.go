package auto

import (
	_ "embed" // the instructions of the placement are a text file of the package
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Exported variables.
var (
	// ErrPlaceReply is the failure of a placement given a reply that is not the lessons judgement's (855d4h).
	ErrPlaceReply = errors.New("the reply of the lessons judgement is not a list of lessons")
)

// PlaceInput is the reply of the lessons judgement and its carry, as the memory mod hands them over (855d4h).
type PlaceInput struct {
	Session string `json:"session"`
	Turn    string `json:"turn"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
}

// LessonsOf returns the whole lessons of the reply of the lessons judgement, or an error when the reply is no list
// of lessons (855d4h).
func LessonsOf(reply string) ([]Lesson, error) {
	lessons, skipped := ParseLessons(reply)
	if len(lessons) == 0 && slices.ContainsFunc(skipped, func(left Skipped) bool { return left.Reason == notAList }) {
		return nil, ErrPlaceReply
	}

	return lessons, nil
}

// Place returns the request that places the lessons against the notes the session has seen (855d4h), and false when
// there is no lesson to place or no note seen: every lesson is then written at the top and ties no note.
func Place(lessons []Lesson, seen []Note) (Step, bool) {
	if len(lessons) == 0 || len(seen) == 0 {
		return Step{}, false
	}

	lines := make([]string, 0, len(lessons))
	for _, lesson := range lessons {
		lines = append(lines, escaped("- "+lesson.Slug+" ("+lesson.Kind+"; "+lesson.Situation+")"))
	}

	names := make([]string, 0, len(seen))
	for _, note := range seen {
		names = append(names, note.Path)
	}

	return Step{
		Request: ModelRequest{
			Model: midTier, System: placeInstructions,
			Prompt:    fmt.Sprintf(placeShape, strings.Join(lines, "\n"), seenLines(seen)),
			MaxTokens: placeMaxTokens, TimeoutMs: placeTimeoutMs,
		},
		Carry: Carry{Seen: names},
	}, true
}

// SeenNotes returns the notes of the vault a session has seen (855d4h): those whose file name its transcript
// shows as delivered or looked up (held), and those whose source says they were written automatically in it.
func SeenNotes(vault []Note, held, session string) []Note {
	written := SourceFor(session)

	return slices.DeleteFunc(slices.Clone(vault), func(note Note) bool {
		return !namedIn(held, note.Path) && !strings.Contains(note.Source, written)
	})
}

// Uncovered returns the lessons that no note of the vault covers: the writing leaves a lesson whose `covered` names a
// file of the vault out, so the placement is not asked about it (855d4h).
func Uncovered(lessons []Lesson, vault []Note) []Lesson {
	return slices.DeleteFunc(slices.Clone(lessons), func(lesson Lesson) bool {
		return slices.ContainsFunc(vault, func(note Note) bool {
			return lesson.Covered != "" && strings.TrimSuffix(lesson.Covered, ".md")+".md" == note.Path
		})
	})
}

// unexported constants.
const (
	// placeMaxTokens is a ceiling and not a spend: an entry for each lesson, with its supersessions.
	placeMaxTokens = 1500
	// placeShape is the prompt of the request: the lessons and the seen notes, each between its tags.
	placeShape = "<lessons>\n%s\n</lessons>\n<notes>\n%s\n</notes>"
	// placeTimeoutMs is the row of the model call of the placement judgement (793d2b).
	placeTimeoutMs = 15000
)

//go:embed prompts/place.txt
var placeInstructions string

// seenLines writes the seen notes, one line each, in the form of a line of a block: a file name as it is, since a name
// is no credential, the situation and the body written as one line and redacted, and the body cut to the
// limit of a note (855d4h).
func seenLines(notes []Note) string {
	lines := make([]string, 0, len(notes))

	for _, note := range notes {
		lines = append(lines, escaped("- "+oneLine(note.Path)+" ("+oneRedacted(note.Situation)+"): "+
			clipped(oneRedacted(note.Body), noteLimit)))
	}

	return strings.Join(lines, "\n")
}
