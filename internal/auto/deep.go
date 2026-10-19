package auto

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Amendment is a note a deep delivery amends (855d4j): the passages that evidence it, as chunk sources, and the fields
// its judgement revises, by the name the learn skill gives them.
type Amendment struct {
	Note   string
	Chunks []string
	Fields map[string]string
}

// noteToAmend returns the note of the carry that a verdict of covered or near on an ID asks to amend.
func (c Carry) noteToAmend(id, verdict string) (Candidate, bool) {
	index := slices.IndexFunc(c.Candidates, func(one Candidate) bool { return one.ID == id && one.Kind == KindNote })
	if index < 0 || (verdict != verdictCovered && verdict != verdictNear) {
		return Candidate{}, false
	}

	return c.Candidates[index], true
}

// passages returns the IDs among those given that are passages of the carry, once each.
func (c Carry) passages(ids []string) []string {
	var held []string

	for _, id := range ids {
		if !slices.Contains(held, id) && slices.ContainsFunc(c.Candidates, func(one Candidate) bool {
			return one.ID == id && one.Kind == KindPassage
		}) {
			held = append(held, id)
		}
	}

	return held
}

// ties returns the supersessions among those given that name a note of the carry with a type and a claim, those of the
// lesson when the tie holds none.
func (c Carry) ties(ties Ties, kind, claim string) Ties {
	var held Ties

	for _, tie := range ties {
		tie.Type, tie.Claim = firstOf(tie.Type, kind), firstOf(tie.Claim, claim)

		index := slices.IndexFunc(c.Candidates, func(one Candidate) bool {
			name := strings.TrimSpace(tie.Note)

			return one.Kind == KindNote && (one.ID == name || one.ID == name+".md")
		})

		if index >= 0 && isSupersedesType(tie.Type) && !isBlank(tie.Claim) {
			held = append(held, Supersession{Note: c.Candidates[index].ID, Type: tie.Type, Claim: oneRedacted(tie.Claim)})
		}
	}

	return held
}

// DeepWrite is what the reply of a deep recall says to write (855d4j): the whole lessons, the amendments, and what is
// left out with the reason.
type DeepWrite struct {
	Lessons    []Lesson
	Amendments []Amendment
	Skipped    []Skipped
}

// addCandidate adds the amendment a candidate entry of the reply asks for, or the reason it is left out.
func (d *DeepWrite) addCandidate(raw json.RawMessage, carry Carry, seen *[]string) {
	var entry struct {
		ID      string            `json:"id"`
		Verdict string            `json:"verdict"`
		Chunks  []string          `json:"chunks"`
		Fields  map[string]string `json:"fields"`
	}

	if json.Unmarshal(raw, &entry) != nil {
		d.Skipped = append(d.Skipped, Skipped{Reason: "an entry of the candidates is not an object of strings"})

		return
	}

	note, amended := carry.noteToAmend(entry.ID, entry.Verdict)
	if !amended || slices.Contains(*seen, entry.ID) {
		return
	}

	*seen = append(*seen, entry.ID)

	if reason := refusal(note); reason != "" {
		d.Skipped = append(d.Skipped, Skipped{Reason: reason})

		return
	}

	amendment := Amendment{Note: note.ID, Chunks: carry.passages(entry.Chunks), Fields: map[string]string{}}

	if entry.Verdict == verdictNear {
		d.addFields(&amendment, note, entry.Fields)
	}

	if len(amendment.Chunks) > 0 || len(amendment.Fields) > 0 {
		d.Amendments = append(d.Amendments, amendment)
	}
}

// addFields adds the fields a near note has to the amendment, and the reason for each field it has not.
func (d *DeepWrite) addFields(amendment *Amendment, note Candidate, fields map[string]string) {
	known := fieldsOfKind(note.Type)

	for _, name := range slices.Sorted(func(yield func(string) bool) {
		for name := range fields {
			if !yield(name) {
				return
			}
		}
	}) {
		switch {
		case !slices.Contains(known, name):
			d.Skipped = append(d.Skipped, Skipped{
				Reason: fmt.Sprintf("the note %s has no field %s", oneLine(note.ID), oneLine(name)),
			})
		case !isBlank(fields[name]):
			amendment.Fields[name] = oneRedacted(fields[name])
		}
	}
}

// addLessons adds the whole facts and feedback of the lessons with the chunks and supersessions the carry holds.
func (d *DeepWrite) addLessons(list string, carry Carry) {
	lessons, skipped := ParseLessons(list)
	d.Skipped = append(d.Skipped, skipped...)

	for _, lesson := range lessons {
		if lesson.Kind == kindRunbook {
			d.Skipped = append(d.Skipped, Skipped{Slug: lesson.Slug, Reason: "a deep delivery writes facts and feedback only"})

			continue
		}

		lesson.Chunks = carry.passages(lesson.Chunks)
		lesson.Supersedes = carry.ties(lesson.Supersedes, lesson.Type, lesson.Claim)
		d.Lessons = append(d.Lessons, lesson)
	}
}

// ReadDeep reads what the reply of the candidate judgement at deep depth says to write (855d4j). Only an ID the carry
// holds is acted on: a passage as a chunk source, a note as a supersession or an amendment, and a note of the parent
// vault or a pending copy is never amended (855b34).
func ReadDeep(reply string, carry Carry) DeepWrite {
	var parsed struct {
		Candidates *[]json.RawMessage `json:"candidates"`
		Lessons    json.RawMessage    `json:"lessons"`
	}

	if json.Unmarshal([]byte(replyValue(reply)), &parsed) != nil || parsed.Candidates == nil {
		return DeepWrite{Skipped: []Skipped{{Reason: notDeepReply}}}
	}

	deep := DeepWrite{}

	if len(parsed.Lessons) > 0 && string(parsed.Lessons) != "null" {
		deep.addLessons(string(parsed.Lessons), carry)
	}

	seen := make([]string, 0, len(*parsed.Candidates))

	for _, raw := range *parsed.Candidates {
		deep.addCandidate(raw, carry, &seen)
	}

	return deep
}

// unexported constants.
const (
	fieldSituation = "situation"
	notDeepReply   = "the reply is not the object of the candidates and the lessons"
	verdictCovered = "covered"
	verdictNear    = "near"
)

// fieldsOfKind returns the fields of a note that a deep delivery revises, by the kind of the note.
func fieldsOfKind(kind string) []string {
	switch kind {
	case kindFact:
		return []string{fieldSituation, "subject", "predicate", "object"}
	case kindFeedback:
		return []string{fieldSituation, "behavior", "impact", "action"}
	case kindRunbook:
		return []string{fieldSituation, "done_when", "body"}
	default:
		return nil
	}
}

// firstOf returns the text if it is not empty, and the fallback otherwise.
func firstOf(text, fallback string) string {
	if text == "" {
		return fallback
	}

	return text
}

// refusal returns why a note is never amended, or nothing: a note of the parent vault and a pending copy are not
// (855b34).
func refusal(note Candidate) string {
	name := oneLine(note.ID)

	switch {
	case note.Parent:
		return "the note " + name + " is from the parent vault and is not amended"
	case note.Pending:
		return "the note " + name + " is a pending copy and is not amended"
	default:
		return ""
	}
}
