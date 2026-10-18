package auto

import (
	_ "embed" // the instructions of the judgement are a text file of the package
	"fmt"
	"slices"
	"strings"
)

// Carry is what a step hands the next one with a request, and the mod reads no part of (855d1b).
type Carry struct {
	Cue    string `json:"cue"`
	Depth  string `json:"depth,omitempty"`
	Window string `json:"window"`
	// Candidates are the notes and passages the lookup found, in full, for the judgement of them.
	Candidates []Candidate `json:"candidates,omitempty"`
}

// ModelRequest is the request of a call of the model that a step prints: the fields the host's model call takes
// (855d1b). The effort is the cheap tier's only, and the time limit a capture's.
type ModelRequest struct {
	Model     string `json:"model"`
	Effort    string `json:"effort,omitempty"`
	System    string `json:"system"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"maxTokens"`
	TimeoutMs int    `json:"timeoutMs,omitempty"`
}

// MomentInput is a moment, as the memory mod hands it over (855d4a).
type MomentInput struct {
	Session string `json:"session"`
	Cue     string `json:"cue"`
	Origin  string `json:"origin"`
	Text    string `json:"text"`
}

// Step is what a step prints for a judgement: the request, and the carry for the next step.
type Step struct {
	Request ModelRequest `json:"request"`
	Carry   Carry        `json:"carry"`
}

// Moment returns the request that judges whether a moment is one for recall (855d4a): the conversation lines before it
// and the line of the moment, as the window of a judgement of a turn is built (793d4g).
func Moment(record []byte, input MomentInput) Step {
	messages, _ := messagesOf(record)
	before := linesOf(messages[max(len(messages)-contextLines, 0):])
	before, turn := fit(before, []string{escaped(userLabel + messageText(input.Text))})
	window := strings.Join(slices.Concat(before, turn), "\n")

	return Step{
		Request: ModelRequest{
			Model: momentModel, Effort: momentEffort, System: recallMoment,
			Prompt: fmt.Sprintf(windowShape, window), MaxTokens: momentMaxTokens,
		},
		Carry: Carry{Cue: input.Cue, Window: window},
	}
}

// unexported constants.
const (
	momentEffort    = "low"
	momentMaxTokens = 200
	momentModel     = "haiku"
	// windowShape is the prompt of a request that holds the window only: the window between its tags.
	windowShape = "<window>\n%s\n</window>"
)

//go:embed prompts/moment_recall.txt
var recallMoment string
