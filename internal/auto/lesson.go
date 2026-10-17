package auto

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Exported constants.
const (
	RecordFailed  = "failed"
	RecordNothing = "nothing"
	RecordSkipped = "skipped"
	RecordWritten = "written"
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

// defect returns what is wrong with the lesson, or nothing when it is whole: it has a kind it knows and the fields of
// its kind, a slug of lower-case letters, digits and dashes, and a situation.
func (l Lesson) defect() string {
	names := []string{"situation"}
	values := []string{l.Situation}

	switch l.Kind {
	case kindFeedback:
		names = append(names, "behavior", "impact", "action")
		values = append(values, l.Behavior, l.Impact, l.Action)
	case kindFact:
		names = append(names, "subject", "predicate", "object")
		values = append(values, l.Subject, l.Predicate, l.Object)
	default:
		return fmt.Sprintf("the kind %q is not feedback or fact", l.Kind)
	}

	if !slugPattern.MatchString(l.Slug) {
		return "the slug is not lower-case letters, digits and dashes"
	}

	if empty := slices.IndexFunc(values, isBlank); empty >= 0 {
		return "the lesson has no " + names[empty]
	}

	return ""
}

// Record is what became of a reply of the model, or of the failure to get one, as the display of what automatic memory
// did reads it (793d4k): a lesson that was written or left out, a reply with no lesson, or a failure. A lesson has a
// slug, and a note that was written has a file name.
type Record struct {
	At      time.Time `json:"at"`
	Session string    `json:"session"`
	Turn    string    `json:"turn"`
	Kind    string    `json:"kind"`
	Note    string    `json:"note,omitempty"`
	Slug    string    `json:"slug,omitempty"`
	Reason  string    `json:"reason,omitempty"`
}

// Reply is the model's reply to the request of the judgement of a turn, or the failure to get one, as the memory mod
// hands it over (793d4h).
type Reply struct {
	Session string `json:"session"`
	Turn    string `json:"turn"`
	Text    string `json:"reply"`
	Failure string `json:"failure"`
}

// Skipped is a lesson of a reply that is left out, or the whole reply when it is not a list, with the reason
// (793d4k).
type Skipped struct {
	Slug   string
	Reason string
}

// ParseLessons reads the lessons a reply of the model holds: a JSON list, with or without a code fence around it. A
// lesson that is not whole is left out, and a reply that is not a list holds none, each with the reason (793d4h).
func ParseLessons(reply string) ([]Lesson, []Skipped) {
	var lessons []Lesson

	decodeErr := json.Unmarshal([]byte(unfenced(reply)), &lessons)
	if decodeErr != nil {
		return nil, []Skipped{{Reason: "the reply is not a list of lessons"}}
	}

	whole := make([]Lesson, 0, len(lessons))

	var skipped []Skipped

	for _, lesson := range lessons {
		if reason := lesson.defect(); reason != "" {
			skipped = append(skipped, Skipped{Slug: lesson.Slug, Reason: reason})

			continue
		}

		whole = append(whole, lesson)
	}

	return whole, skipped
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
