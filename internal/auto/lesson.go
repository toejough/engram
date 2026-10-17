package auto

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Exported variables.
var (
	ErrNotALessonList = errors.New("the reply is not a list of lessons")
)

// Lesson is one lesson a reply of the model holds (793d4h): a feedback lesson has a behavior, an impact and an action,
// and a fact lesson a subject, a predicate and an object.
type Lesson struct {
	Kind      string `json:"kind"`
	Slug      string `json:"slug"`
	Situation string `json:"situation"`
	Behavior  string `json:"behavior"`
	Impact    string `json:"impact"`
	Action    string `json:"action"`
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
}

// isWhole reports whether the lesson has a slug of lower-case letters, digits and dashes, a situation, and the fields
// of its kind.
func (l Lesson) isWhole() bool {
	var fields []string

	switch l.Kind {
	case kindFeedback:
		fields = []string{l.Behavior, l.Impact, l.Action}
	case kindFact:
		fields = []string{l.Subject, l.Predicate, l.Object}
	default:
		return false
	}

	return slugPattern.MatchString(l.Slug) && !isBlank(l.Situation) && !slices.ContainsFunc(fields, isBlank)
}

// Reply is the model's reply to the request of the judgement of a turn, as the memory mod hands it over (793d4h).
type Reply struct {
	Session string `json:"session"`
	Text    string `json:"reply"`
}

// ParseLessons reads the lessons a reply of the model holds: a JSON list, with or without a code fence around it. A
// lesson that is not whole is left out (793d4h).
func ParseLessons(reply string) ([]Lesson, error) {
	var lessons []Lesson

	decodeErr := json.Unmarshal([]byte(unfenced(reply)), &lessons)
	if decodeErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotALessonList, decodeErr)
	}

	whole := make([]Lesson, 0, len(lessons))

	for _, lesson := range lessons {
		if lesson.isWhole() {
			whole = append(whole, lesson)
		}
	}

	return whole, nil
}

// SourceFor is the source of a note written from a turn of a session: it says the note was written automatically
// (793d4h).
func SourceFor(session string) string {
	return "written automatically (session " + session + ")"
}

// unexported constants.
const (
	kindFact     = "fact"
	kindFeedback = "feedback"
)

// unexported variables.
var (
	slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// isBlank reports whether the text is empty once its white space is gone.
func isBlank(text string) bool {
	return strings.TrimSpace(text) == ""
}

// unfenced returns the reply without the white space around it and the code fence a model may put around a list.
func unfenced(reply string) string {
	text := strings.TrimSpace(reply)
	if !strings.HasPrefix(text, "```") {
		return text
	}

	_, body, _ := strings.Cut(text, "\n")

	return strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(body), "```"))
}
