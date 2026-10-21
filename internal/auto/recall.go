package auto

import (
	_ "embed" // the instructions of the judgement are a text file of the package
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// Exported constants.
const (
	KindNote    = "note"
	KindPassage = "passage"
	// LookupLimit is how many items the lookup is asked for: twice the most candidates, so that the floor of passages
	// has passages to keep when the notes score higher.
	LookupLimit = 2 * MaxCandidates
	// MaxCandidates is how many candidates the judgement holds at most (855d8b).
	MaxCandidates = 30
)

// Exported variables.
var (
	ErrPhrasesReply = errors.New(`the reply of the phrases judgement is not {"phrases": [...]} with a phrase`)
	ErrRecallDepth  = errors.New("the carry of the recall holds no depth of glance or deep")
)

// Candidate is a note or a passage the lookup matched, as the candidate judgement is asked about it (855d4b): its ID is
// the file name of a note or the ID of a chunk, and its text is the body of the note or the text of the chunk.
type Candidate struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Type      string `json:"type"`
	Date      string `json:"date"`
	Situation string `json:"situation"`
	Text      string `json:"text"`
	Parent    bool   `json:"parent"`
	Pending   bool   `json:"pending"`
	// Source is where a note came from, which the delivery marks a note written automatically by (793d4c); a passage
	// has none.
	Source string `json:"source,omitempty"`
}

// RecallInput is the reply of the phrases judgement and its carry, as the memory mod hands them over (855d4b).
type RecallInput struct {
	Session string `json:"session"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
}

// PhrasesOf returns the phrases of the reply of the phrases judgement that the lookup is run for: the first 3 at
// glance and the first 10 at deep (855d4b). A depth that is neither is an error.
func PhrasesOf(reply, depth string) ([]string, error) {
	if depth != DepthGlance && depth != DepthDeep {
		return nil, ErrRecallDepth
	}

	var parsed struct {
		Phrases []string `json:"phrases"`
	}

	readErr := json.Unmarshal([]byte(replyValue(reply)), &parsed)
	phrases := slices.DeleteFunc(parsed.Phrases, isBlank)

	if readErr != nil || len(phrases) == 0 {
		return nil, ErrPhrasesReply
	}

	return phrases[:min(len(phrases), phraseLimit(depth))], nil
}

// Recall returns the request that judges the candidates the lookup matched (855d4b): the window of the carry and a
// line for each candidate, and false when no candidate fits the budget. The matched set is in the order of the scores:
// the cap on the count keeps a floor of passages, and the budget ends the list at the first candidate that does not
// fit. The request shows the situation and the text of a candidate redacted and written as one line, and its ID as it
// is, since a path is no credential; the carry holds each candidate as it is, raw and not cut.
func Recall(input RecallInput, matched []Candidate) (Step, bool) {
	matched = underCap(matched)
	lines := make([]string, 0, len(matched))
	held := make([]Candidate, 0, len(matched))
	used := 0

	for _, candidate := range matched {
		shown := candidate
		shown.Situation = oneRedacted(candidate.Situation)
		shown.Text = oneRedacted(candidate.Text)
		text := shown.Text

		if candidate.Kind == KindPassage {
			text = clipped(text, passageLimit)
		}

		used += utf8.RuneCountInString(text)

		if used > candidateBudget {
			break
		}

		lines = append(lines, candidateLine(shown, text))
		held = append(held, candidate)
	}

	if len(lines) == 0 {
		return Step{}, false
	}

	deep := input.Carry.Depth == DepthDeep
	system := candidateInstructions

	if deep {
		system += deepInstructions
	}

	return Step{
		Request: ModelRequest{
			Model: midTier, System: system,
			Prompt:    fmt.Sprintf(candidatesShape, escaped(input.Carry.Window), strings.Join(lines, "\n")),
			MaxTokens: recallMaxTokens,
		},
		Carry: Carry{Cue: input.Carry.Cue, Depth: input.Carry.Depth, Window: input.Carry.Window, Candidates: held},
		Write: deep,
	}, true
}

// unexported constants.
const (
	// candidateBudget is how many characters of text the candidates hold in all (855d8b).
	candidateBudget = 60000
	// candidatesShape is the prompt of the request: the window and the section of candidates, each between its tags.
	candidatesShape = "<window>\n%s\n</window>\n<candidates>\n%s\n</candidates>"
	deepPhrases     = 10
	glancePhrases   = 3
	// parentMarker stands in the line of a candidate that comes from the parent vault.
	parentMarker = "parent"
	// passageFloor is how many of the candidates under the cap are passages at least, when so many matched.
	passageFloor = 10
	// passageLimit is how many characters of a chunk a candidate line holds at most: a stretch of conversation is
	// longer than a note's lines.
	passageLimit = 1200
	// recallMaxTokens holds a deep reply for the most candidates with fields and lessons: 21,900 characters at the
	// most, 7,300 tokens at 3 characters to the token.
	recallMaxTokens = 8000
	// undated stands in the line of a candidate for the date of one that has none: a chunk indexed before the index
	// kept the time.
	undated = "undated"
)

//go:embed prompts/candidates.txt
var candidateInstructions string

// deepInstructions are added to those of the candidates at deep depth: the lessons and the revisions.
//
//go:embed prompts/candidates_deep.txt
var deepInstructions string

// candidateLine is the line of a candidate: its ID, then in parentheses its date, for a note its type and a marker when
// it comes from the parent vault and its situation, or for a passage the word passage, and then its text, written as
// one line and redacted; the ID is written as one line and not redacted (855d8b, 793d4j).
func candidateLine(candidate Candidate, text string) string {
	date := candidate.Date
	if date == "" {
		date = undated
	}

	about := []string{oneRedacted(date)}

	if candidate.Kind == KindPassage {
		about = append(about, KindPassage)
	} else {
		about = append(about, oneRedacted(candidate.Type))

		if candidate.Parent {
			about = append(about, parentMarker)
		}

		about = append(about, oneRedacted(candidate.Situation))
	}

	// a part the candidate has none of is left out
	about = slices.DeleteFunc(about, func(part string) bool { return part == "" })

	return escaped("- " + oneLine(candidate.ID) + " (" + strings.Join(about, "; ") + "): " + text)
}

// phraseLimit is how many phrases the lookup is run for at a depth.
func phraseLimit(depth string) int {
	if depth == DepthGlance {
		return glancePhrases
	}

	return deepPhrases
}

// underCap returns the candidates under the cap on the count, in the order they were matched: the notes fill the cap
// but for the floor of passages, and the passages fill what the notes leave.
func underCap(matched []Candidate) []Candidate {
	passages := 0

	for _, candidate := range matched {
		if candidate.Kind == KindPassage {
			passages++
		}
	}

	notesLeft := min(len(matched)-passages, MaxCandidates-min(passageFloor, passages))
	passagesLeft := min(passages, MaxCandidates-notesLeft)
	kept := make([]Candidate, 0, notesLeft+passagesLeft)

	for _, candidate := range matched {
		if candidate.Kind == KindPassage {
			if passagesLeft > 0 {
				kept = append(kept, candidate)
				passagesLeft--
			}
		} else if notesLeft > 0 {
			kept = append(kept, candidate)
			notesLeft--
		}
	}

	return kept
}
