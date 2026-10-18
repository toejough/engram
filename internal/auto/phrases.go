package auto

import (
	_ "embed" // the instructions of the judgement are a text file of the package
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Exported constants.
const (
	DepthDeep   = "deep"
	DepthGlance = "glance"
	DepthNone   = "none"
)

// Exported variables.
var (
	ErrMomentReply = errors.New(`the reply of the moment judgement is not {"depth": none, glance or deep}`)
)

// PhrasesInput is the reply of the moment judgement and its carry, as the memory mod hands them over (855d4v).
type PhrasesInput struct {
	Session string `json:"session"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
}

// Phrases returns the request that phrases the lookup for the depth the moment judgement gave (855d4v): the ten angles
// of the recall skill's Step 1, and the window of the carry.
func Phrases(input PhrasesInput) (Step, bool, error) {
	var reply struct {
		Depth string `json:"depth"`
	}

	readErr := json.Unmarshal([]byte(unfenced(input.Reply)), &reply)
	if readErr != nil || !slices.Contains([]string{DepthNone, DepthGlance, DepthDeep}, reply.Depth) {
		return Step{}, false, ErrMomentReply
	}

	if reply.Depth == DepthNone {
		return Step{}, false, nil
	}

	return Step{
		Request: ModelRequest{
			Model: phrasesModel, System: strings.Replace(phraseInstructions, "{{count}}", countFor(reply.Depth), 1),
			Prompt: fmt.Sprintf(windowShape, escaped(input.Carry.Window)), MaxTokens: phrasesMaxTokens,
		},
		Carry: Carry{Cue: input.Carry.Cue, Depth: reply.Depth, Window: input.Carry.Window},
	}, true, nil
}

// unexported constants.
const (
	// the phrases asked for at glance and at deep, by the depth: about 3 at glance, and all ten angles at deep
	deepCount        = "Write exactly 10 short queryable phrases, one for each of these ten angles"
	glanceCount      = "Write about 3 short queryable phrases, from the angles below that fit the moment best"
	phrasesMaxTokens = 1000
	phrasesModel     = "sonnet"
)

//go:embed prompts/phrases.txt
var phraseInstructions string

// countFor is the sentence of the instructions that asks for the phrases of a depth.
func countFor(depth string) string {
	if depth == DepthGlance {
		return glanceCount
	}

	return deepCount
}
