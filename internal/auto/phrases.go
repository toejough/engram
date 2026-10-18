package auto

import (
	_ "embed" // the instructions of the judgement are a text file of the package
	"encoding/json"
	"fmt"
)

// PhrasesInput is the reply of the moment judgement and its carry, as the memory mod hands them over (855d4v).
type PhrasesInput struct {
	Session string `json:"session"`
	Cue     string `json:"cue"`
	Reply   string `json:"reply"`
	Carry   Carry  `json:"carry"`
}

// Phrases returns the request that phrases the lookup for the depth the moment judgement gave (855d4v): the ten angles
// of the recall skill's Step 1, and the window of the carry.
func Phrases(input PhrasesInput) (Step, bool, error) {
	var reply struct {
		Depth string `json:"depth"`
	}

	_ = json.Unmarshal([]byte(unfenced(input.Reply)), &reply)

	return Step{
		Request: ModelRequest{
			Model: phrasesModel, System: phraseInstructions,
			Prompt: fmt.Sprintf(windowShape, input.Carry.Window), MaxTokens: phrasesMaxTokens,
		},
		Carry: Carry{Cue: input.Cue, Depth: reply.Depth, Window: input.Carry.Window},
	}, true, nil
}

// unexported constants.
const (
	phrasesMaxTokens = 1000
	phrasesModel     = "sonnet"
)

//go:embed prompts/phrases.txt
var phraseInstructions string
