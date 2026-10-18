package auto

import (
	"encoding/json"
	"errors"
	"slices"
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

// phraseLimit is how many phrases the lookup is run for at a depth.
func phraseLimit(depth string) int {
	if depth == DepthGlance {
		return glancePhrases
	}

	return deepPhrases
}

// unexported constants.
const (
	deepPhrases   = 10
	glancePhrases = 3
)
