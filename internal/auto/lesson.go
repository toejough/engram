package auto

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Exported constants.
const (
	RecordFailed  = "failed"
	RecordSkipped = "skipped"
	RecordWritten = "written"
)

// Lesson is one lesson a reply of the model holds (793d4h): a feedback lesson has a behavior, an impact and an action,
// and a fact lesson a subject, a predicate and an object.
type Lesson struct {
	Kind      string `json:"kind"`
	Slug      string `json:"slug"`
	Situation string `json:"situation"`
	Covered   string `json:"covered"`
	Behavior  string `json:"behavior"`
	Impact    string `json:"impact"`
	Action    string `json:"action"`
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	DoneWhen  string `json:"done_when"` //nolint:tagliatelle // the name the learn skill gives the field of a runbook
	Body      string `json:"body"`
	// Chunks, Supersedes, Type and Claim are those of a lesson of a deep delivery (855d4j): the passages that evidence
	// it, the note it corrects and the tie and the claim that changes.
	Chunks     []string `json:"chunks"`
	Supersedes Ties     `json:"supersedes"`
	Type       string   `json:"type"`
	Claim      string   `json:"claim"`
}

// defect returns what is wrong with the lesson, or nothing when it is whole: it has a kind it knows and the fields of
// its kind, a slug of 1 to 64 lower-case letters, digits and dashes, and a situation.
func (l Lesson) defect() string {
	names := []string{fieldSituation}
	values := []string{l.Situation}

	switch l.Kind {
	case kindFeedback:
		names = append(names, "behavior", "impact", "action")
		values = append(values, l.Behavior, l.Impact, l.Action)
	case kindFact:
		names = append(names, "subject", "predicate", "object")
		values = append(values, l.Subject, l.Predicate, l.Object)
	case kindRunbook:
		names = append(names, "done_when", "body")
		values = append(values, l.DoneWhen, l.Body)
	default:
		return fmt.Sprintf("the kind %q is not feedback, fact or runbook", l.Kind)
	}

	if !isSlug(l.Slug) {
		return "the slug is not 1 to 64 lower-case letters, digits and dashes"
	}

	if empty := slices.IndexFunc(values, isBlank); empty >= 0 {
		return "the lesson has no " + names[empty]
	}

	return ""
}

// redacted is the lesson with each text of it written as one line (793d4h) and redacted (793d4j): a secret cannot be a
// slug, because the redaction makes the slug something that is not one, and a field cannot make a line of the note of
// its own.
func (l Lesson) redacted() Lesson {
	return Lesson{
		Kind: l.Kind, Slug: oneRedacted(l.Slug), Situation: oneRedacted(l.Situation), Covered: oneRedacted(l.Covered),
		Behavior: oneRedacted(l.Behavior), Impact: oneRedacted(l.Impact), Action: oneRedacted(l.Action),
		Subject: oneRedacted(l.Subject), Predicate: oneRedacted(l.Predicate), Object: oneRedacted(l.Object),
		DoneWhen: oneRedacted(l.DoneWhen), Body: oneRedacted(l.Body), Chunks: l.Chunks, Supersedes: l.Supersedes,
		Type: l.Type, Claim: oneRedacted(l.Claim),
	}
}

// Record is what became of a reply of the model, or of the failure to get one, as the display of what automatic memory
// did reads it (793d4k): a lesson that was written or left out, or a failure. A reply with no lesson makes none. A
// lesson has a slug, and a note that was written has a file name.
type Record struct {
	At      time.Time `json:"at"`
	Session string    `json:"session"`
	Turn    string    `json:"turn"`
	Kind    string    `json:"kind"`
	Note    string    `json:"note,omitempty"`
	Slug    string    `json:"slug,omitempty"`
	Reason  string    `json:"reason,omitempty"`
	// File is the name of the file the record is kept in, which the line that tells it is marked by (793d4p); it is
	// no part of the record.
	File string `json:"-"`
}

// Reply is the model's reply to the request of the judgement of a turn, or the failure to get one, as the memory mod
// hands it over (793d4h).
type Reply struct {
	Session string `json:"session"`
	Turn    string `json:"turn"`
	// Cue is set for the write of a deep delivery, whose reply is the candidate judgement's (855d4j).
	Cue     string `json:"cue"`
	Text    string `json:"reply"`
	Failure string `json:"failure"`
	// Placement is the reply of the placement judgement, and Carry the placement's carry, which names the notes the
	// session has seen (855d4i).
	Placement string `json:"placement"`
	Carry     Carry  `json:"carry"`
	// PlacementFailure is why the placement could not be had, when it could not: the lessons are written at the top
	// (855d2b).
	PlacementFailure string `json:"placementFailure"`
}

// Skipped is a lesson of a reply that is left out, or the whole reply when it is not a list, with the reason
// (793d4k).
type Skipped struct {
	Slug   string
	Reason string
}

// ParseLessons reads the lessons a reply of the model holds: a JSON list, with or without a code fence around it. A
// lesson that is not whole is left out, and a reply that is not a list holds none, each with the reason. A reply that
// ends before its list does keeps the lessons before the cut, and the cut is a reason of its own (793d4h).
func ParseLessons(reply string) ([]Lesson, []Skipped) {
	lessons, ended, isList := readList(replyValue(reply))
	if !isList {
		return nil, []Skipped{{Reason: notAList}}
	}

	whole := make([]Lesson, 0, len(lessons))

	var skipped []Skipped

	if !ended {
		skipped = append(skipped, Skipped{Reason: "the reply was cut before the end of its list"})
	}

	for _, found := range lessons {
		lesson := found.redacted()

		if reason := lesson.defect(); reason != "" {
			skipped = append(skipped, Skipped{Slug: shownSlug(lesson.Slug), Reason: reason})

			continue
		}

		whole = append(whole, lesson)
	}

	return whole, skipped
}

// RecordedReason is a reason as a record keeps it (793d4k): written as one line, redacted and only then cut in its
// middle to the bound of a record, so that the step a failure names at its beginning and the cause at its end stay.
func RecordedReason(reason string) string {
	return squeezed(oneRedacted(reason), recordedReasonLimit)
}

// SourceFor is the source of a note written from a turn of a session: it says the note was written automatically
// (793d4h).
func SourceFor(session string) string {
	return automaticSource + " (session " + session + ")"
}

// unexported constants.
const (
	kindFact     = "fact"
	kindFeedback = "feedback"
	kindRunbook  = "runbook"
	// notAList is the reason recorded for a reply that is no list of lessons (793d4k).
	notAList = "the reply is not a list of lessons"
	// recordedReasonLimit is how many characters of a reason a record keeps at most (793d4k).
	recordedReasonLimit = 300
	// slugLimit is how many characters a slug has at most (793d4h).
	slugLimit = 64
)

// unexported variables.
var (
	slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// endsTooSoon reports whether a failure to read a text of the given length is the text ending before its value does,
// and not a text that is wrong.
func endsTooSoon(err error, length int) bool {
	var syntaxErr *json.SyntaxError

	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		(errors.As(err, &syntaxErr) && syntaxErr.Offset >= int64(length))
}

// isBlank reports whether the text is empty once its white space is gone.
func isBlank(text string) bool {
	return strings.TrimSpace(text) == ""
}

// isSlug reports whether a text is a slug: 1 to slugLimit lower-case letters, digits and dashes (793d4h).
func isSlug(text string) bool {
	return len(text) <= slugLimit && slugPattern.MatchString(text)
}

// oneRedacted is the text written as one line and then redacted: the redaction comes second, because writing a text as
// one line drops the characters that are not shown, and a credential that they split is whole only once they are gone
// (793d4j).
func oneRedacted(text string) string {
	return Redact(oneLine(text))
}

// readList reads the lessons of a JSON list one by one, so that the lessons before a cut are kept. ended is false when
// the text stops before its list does, and isList is false when the text is no list of lessons, or has more after it.
func readList(text string) (lessons []Lesson, ended, isList bool) {
	decoder := json.NewDecoder(strings.NewReader(text))

	open, openErr := decoder.Token()
	if openErr != nil || open != json.Delim('[') {
		return nil, false, false
	}

	for decoder.More() {
		var lesson Lesson

		decodeErr := decoder.Decode(&lesson)
		if decodeErr != nil {
			return lessons, false, endsTooSoon(decodeErr, len(text))
		}

		lessons = append(lessons, lesson)
	}

	_, closeErr := decoder.Token()
	if closeErr != nil {
		return lessons, false, endsTooSoon(closeErr, len(text))
	}

	_, afterErr := decoder.Token()

	return lessons, true, errors.Is(afterErr, io.EOF)
}

// replyValue is the JSON value of a model's reply, the one rule for every reply (855b83): the reply, unfenced, when it
// is valid JSON, and otherwise, when the model reasoned before it, the last value that begins with a brace or a bracket
// and runs to the end of the reply. A reply with no such value, such as a list that was cut off, is the unfenced reply.
func replyValue(reply string) string {
	text := unfenced(reply)

	if json.Valid([]byte(text)) {
		return text
	}

	ended := strings.TrimSpace(strings.TrimSuffix(text, "```"))

	for at := len(ended) - 1; at >= 0; at-- {
		if (ended[at] == '{' || ended[at] == '[') && json.Valid([]byte(ended[at:])) {
			return ended[at:]
		}
	}

	return text
}

// shownSlug is the slug of a lesson that is left out when it is one, and nothing when it is the model's text (793d4k).
func shownSlug(slug string) string {
	if isSlug(slug) {
		return slug
	}

	return ""
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
