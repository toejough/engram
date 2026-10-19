package auto

import (
	"cmp"
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

// DeepWrite is what the reply of a deep recall says to write (855d4j): the whole lessons, the amendments, and what is
// left out with the reason.
type DeepWrite struct {
	Lessons    []Lesson
	Amendments []Amendment
	Skipped    []Skipped
}

// ReadDeep reads what the reply of the candidate judgement at deep depth says to write (855d4j). Only an ID the carry
// holds is acted on: a passage as a chunk source, a note as a supersession or an amendment, and a note of the parent
// vault or a pending copy is never amended (855b34).
func ReadDeep(reply string, carry Carry) DeepWrite {
	var parsed struct {
		Candidates []json.RawMessage `json:"candidates"`
		Lessons    json.RawMessage   `json:"lessons"`
	}

	if json.Unmarshal([]byte(replyValue(reply)), &parsed) != nil {
		return DeepWrite{Skipped: []Skipped{{Reason: notDeepReply}}}
	}

	deep := DeepWrite{}

	if len(parsed.Lessons) > 0 && string(parsed.Lessons) != "null" {
		deep.addLessons(string(parsed.Lessons), carry)
	}

	seen := make([]string, 0, len(parsed.Candidates))

	for _, raw := range parsed.Candidates {
		deep.addCandidate(raw, carry, &seen)
	}

	return deep
}

// unexported constants.
const (
	notDeepReply = "the reply is not the object of the candidates and the lessons"
)

// unexported variables.
var (
	// fieldsOfKind are the fields of a note that a deep delivery revises, by the kind of the note.
	fieldsOfKind = map[string][]string{
		kindFact:     {"situation", "subject", "predicate", "object"},
		kindFeedback: {"situation", "behavior", "impact", "action"},
		kindRunbook:  {"situation", "done_when", "body"},
	}
)

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

	index := slices.IndexFunc(carry.Candidates, func(one Candidate) bool {
		return one.ID == entry.ID && one.Kind == KindNote
	})
	if index < 0 || (entry.Verdict != "covered" && entry.Verdict != "near") || slices.Contains(*seen, entry.ID) {
		return
	}

	*seen = append(*seen, entry.ID)
	note := carry.Candidates[index]
	name := oneLine(note.ID)

	switch {
	case note.Parent:
		d.Skipped = append(d.Skipped, Skipped{Reason: "the note " + name + " is from the parent vault and is not amended"})

		return
	case note.Pending:
		d.Skipped = append(d.Skipped, Skipped{Reason: "the note " + name + " is a pending copy and is not amended"})

		return
	}

	amendment := Amendment{Note: note.ID, Chunks: carry.passages(entry.Chunks), Fields: map[string]string{}}

	if entry.Verdict == "near" {
		d.addFields(&amendment, note, entry.Fields)
	}

	if len(amendment.Chunks) > 0 || len(amendment.Fields) > 0 {
		d.Amendments = append(d.Amendments, amendment)
	}
}

// addFields adds the fields a near note has to the amendment, and the reason for each field it has not.
func (d *DeepWrite) addFields(amendment *Amendment, note Candidate, fields map[string]string) {
	known := fieldsOfKind[note.Type]

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

// addLessons adds the whole facts and feedback of the reply's lessons with the chunks and supersessions the carry holds.
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
		tie.Type, tie.Claim = cmp.Or(tie.Type, kind), cmp.Or(tie.Claim, claim)

		if slices.ContainsFunc(c.Candidates, func(one Candidate) bool {
			return one.ID == strings.TrimSpace(tie.Note) && one.Kind == KindNote
		}) && isSupersedesType(tie.Type) && !isBlank(tie.Claim) {
			held = append(held, Supersession{Note: tie.Note, Type: tie.Type, Claim: oneRedacted(tie.Claim)})
		}
	}

	return held
}
