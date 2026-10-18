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
	// MaxCandidates is how many candidates the judgement holds at most (855d8b).
	MaxCandidates = 30
)

// Exported variables.
var (
	ErrPhrasesReply = errors.New(`the reply of the phrases judgement is not {"phrases": [...]} with a phrase`)
)

// Candidate is a note or a passage the lookup matched, as the candidate judgement is asked about it (855d4b): its ID is
// the file name of a note or the ID of a chunk, and its text is the body of the note or the text of the chunk.
type Candidate struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Date      string `json:"date"`
	Situation string `json:"situation"`
	Text      string `json:"text"`
	Parent    bool   `json:"parent"`
	Pending   bool   `json:"pending"`
}

// RecallInput is the reply of the phrases judgement and its carry, as the memory mod hands them over (855d4b).
type RecallInput struct {
	Session string `json:"session"`
	Cue     string `json:"cue"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
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

// Recall returns the request that judges the candidates the lookup matched (855d4b): the window of the carry and a
// line for each candidate.
func Recall(input RecallInput, matched []Candidate) Step {
	lines := make([]string, 0, min(len(matched), MaxCandidates))
	used := 0

	for _, candidate := range matched {
		text := candidateText(candidate)
		used += utf8.RuneCountInString(text)

		// the last of the matched set is dropped first (855d8b)
		if len(lines) == MaxCandidates || used > candidateBudget {
			break
		}

		lines = append(lines, candidateLine(candidate, text))
	}

	matched = matched[:len(lines)]

	deep := input.Carry.Depth == DepthDeep
	system := candidateInstructions

	if deep {
		system += deepInstructions
	}

	return Step{
		Request: ModelRequest{
			Model: recallModel, System: system,
			Prompt:    fmt.Sprintf(candidatesShape, input.Carry.Window, strings.Join(lines, "\n")),
			MaxTokens: recallMaxTokens,
		},
		Carry: Carry{Cue: input.Cue, Depth: input.Carry.Depth, Window: input.Carry.Window, Candidates: matched},
		Write: deep,
	}
}

// unexported constants.
const (
	// candidateBudget is how many characters of text the candidates hold in all (855d8b).
	candidateBudget = 60000
	// candidatesShape is the prompt of the request: the window and the section of candidates, each between its tags.
	candidatesShape = "<window>\n%s\n</window>\n<candidates>\n%s\n</candidates>"
	deepPhrases     = 10
	glancePhrases   = 3
	// passageLimit is how many characters of a chunk a candidate line holds at most: a stretch of conversation is
	// longer than a note's lines (855b33).
	passageLimit    = 1200
	recallMaxTokens = 4000
	recallModel     = "sonnet"
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

// candidateLine is the line of a candidate: its ID, its date, its situation or the word passage, and its text, written
// as one line and redacted (855d8b, 793d4j).
func candidateLine(candidate Candidate, text string) string {
	about := candidate.Situation
	if candidate.Kind == KindPassage {
		about = KindPassage
	}

	date := candidate.Date
	if date == "" {
		date = undated
	}

	return escaped("- " + oneRedacted(candidate.ID) + " (" + oneRedacted(date) + "; " + oneRedacted(about) + "): " + text)
}

// candidateText is the text of a candidate as its line holds it: a note whole, and a chunk cut to passageLimit
// characters, after the redaction (855b33).
func candidateText(candidate Candidate) string {
	text := oneRedacted(candidate.Text)
	if candidate.Kind == KindPassage {
		return clipped(text, passageLimit)
	}

	return text
}

// phraseLimit is how many phrases the lookup is run for at a depth.
func phraseLimit(depth string) int {
	if depth == DepthGlance {
		return glancePhrases
	}

	return deepPhrases
}
