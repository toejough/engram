package auto

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrPhrasesReply is the failure of a step that is given a reply of the phrases judgement that holds no phrase (855d4b).
var ErrPhrasesReply = errors.New(`the reply of the phrases judgement is not {"phrases": [...]} with a phrase`)

// RecallInput is the reply of the phrases judgement and its carry, as the memory mod hands them over (855d4b).
type RecallInput struct {
	Session string `json:"session"`
	Cue     string `json:"cue"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
}

// The kinds of a candidate (855d8b).
const (
	KindNote    = "note"
	KindPassage = "passage"
)

// Candidate is a note or a passage the lookup matched, as the candidate judgement is asked about it (855d4b): its ID is
// the file name of a note or the ID of a chunk, and its text is the body of the note or the text of the chunk.
type Candidate struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Situation string `json:"situation"`
	Text      string `json:"text"`
}

// Recall returns the request that judges the candidates the lookup matched (855d4b): the window of the carry and a
// line for each candidate.
func Recall(input RecallInput, matched []Candidate) Step {
	lines := make([]string, 0, len(matched))

	for _, candidate := range matched {
		lines = append(lines, candidateLine(candidate))
	}

	return Step{
		Request: ModelRequest{Prompt: fmt.Sprintf(candidatesShape, input.Carry.Window, strings.Join(lines, "\n"))},
		Carry:   Carry{Cue: input.Cue, Depth: input.Carry.Depth, Window: input.Carry.Window},
	}
}

// PhrasesOf returns the phrases of the reply of the phrases judgement that the lookup is run for: the first 3 at
// glance and the first 10 at deep (855d4b).
func PhrasesOf(reply, depth string) ([]string, error) {
	var parsed struct {
		Phrases []string `json:"phrases"`
	}

	readErr := json.Unmarshal([]byte(unfenced(reply)), &parsed)
	phrases := slices.DeleteFunc(parsed.Phrases, isBlank)

	if readErr != nil || len(phrases) == 0 {
		return nil, ErrPhrasesReply
	}

	return phrases[:min(len(phrases), phraseLimit(depth))], nil
}

// candidateLine is the line of a candidate: its ID, its situation or the word passage, and its text, written as one
// line and redacted (855d8b, 793d4j).
func candidateLine(candidate Candidate) string {
	about := candidate.Situation
	if candidate.Kind == KindPassage {
		about = KindPassage
	}

	return escaped("- " + oneRedacted(candidate.ID) + " (" + oneRedacted(about) + "): " + oneRedacted(candidate.Text))
}

// phraseLimit is how many phrases the lookup is run for at a depth.
func phraseLimit(depth string) int {
	if depth == DepthGlance {
		return glancePhrases
	}

	return deepPhrases
}

// unexported constants.
const (
	// candidatesShape is the prompt of the request: the window and the section of candidates, each between its tags.
	candidatesShape = "<window>\n%s\n</window>\n<candidates>\n%s\n</candidates>"
	deepPhrases   = 10
	glancePhrases = 3
)
