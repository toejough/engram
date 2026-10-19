package auto

import (
	_ "embed" // the instructions of the judgement are a text file of the package
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Exported constants.
const (
	// CueTurnEnd is the cue of a turn that ended, the learn moment (855d4a).
	CueTurnEnd = "turn-end"
)

// Carry is what a step hands the next one with a request, and the mod reads no part of (855d1b).
type Carry struct {
	Cue    string `json:"cue,omitempty"`
	Depth  string `json:"depth,omitempty"`
	Window string `json:"window,omitempty"`
	// Candidates are the notes and passages the lookup found, in full, for the judgement of them.
	Candidates []Candidate `json:"candidates,omitempty"`
	// Seen names, by file name, the notes the session has seen, which the placement judged the lessons against (855d4h).
	Seen []string `json:"seen,omitempty"`
}

// MomentInput is a moment, as the memory mod hands it over (855d4a).
type MomentInput struct {
	Session string `json:"session"`
	Cue     string `json:"cue"`
	Origin  string `json:"origin"`
	Text    string `json:"text"`
	// Turn and Answer are the turn's ID and final text, for the cue turn-end.
	Turn   string `json:"turn"`
	Answer string `json:"answer"`
	// Event is what a tool cue is about: a tool call about to run, a failure, or a batch of calls with their responses
	// (855d4a).
	Event ToolEvent `json:"event"`
}

// ToolCall is a call of a batch, with the response it resolved with.
type ToolCall struct {
	Tool     string          `json:"tool"`
	Input    json.RawMessage `json:"input"`
	Response json.RawMessage `json:"response"`
}

// ToolEvent is the event of a tool cue (855d4a): for before-tool the tool and its input, for failure the tool, its
// input and the error, and for batch-done the calls with their responses.
type ToolEvent struct {
	Tool  string          `json:"tool"`
	Input json.RawMessage `json:"input"`
	Error string          `json:"error"`
	Calls []ToolCall      `json:"calls"`
}

// Step is what a step prints for a judgement: the request, and the carry for the next step.
type Step struct {
	Request ModelRequest `json:"request"`
	Carry   Carry        `json:"carry"`
	// Write tells the mod that a write follows the delivery: a deep recall's.
	Write bool `json:"write,omitempty"`
}

// Moment returns the request that judges whether a moment is one for recall, or, at the end of a turn, one for learn
// (855d4a): the conversation lines before it and the line of the moment, as the window of a judgement of a turn is
// built (793d4g).
func Moment(record []byte, input MomentInput) Step {
	messages, last := messagesOf(record)

	if input.Cue == CueTurnEnd {
		return learnMoment(messages, last, input)
	}

	before := linesOf(messages[max(len(messages)-contextLines, 0):])
	before, turn := fit(before, []string{escaped(userLabel + messageText(input.Text))})
	window := strings.Join(slices.Concat(before, turn), "\n")

	return Step{
		Request: ModelRequest{
			Model: cheapTier, Effort: momentEffort, System: recallMoment,
			Prompt: fmt.Sprintf(windowShape, window), MaxTokens: momentMaxTokens,
		},
		Carry: Carry{Cue: input.Cue, Window: window},
	}
}

// unexported constants.
const (
	// learnMomentTimeoutMs is the row of the model call of the learn-moment judgement (855d2b).
	learnMomentTimeoutMs = 10000
	momentEffort         = "low"
	momentMaxTokens      = 200
	// windowShape is the prompt of a request that holds the window only: the window between its tags.
	windowShape = "<window>\n%s\n</window>"
)

//go:embed prompts/moment_learn.txt
var learnMomentInstructions string

//go:embed prompts/moment_recall.txt
var recallMoment string

// learnMoment is the request that judges whether a turn that ended is a learn moment (855d4a): the window of the turn,
// as the judgement of the turn builds it, which a turn of many tool calls could not be judged without.
func learnMoment(messages []message, last int, input MomentInput) Step {
	start, finish := 0, 0
	if last >= 0 {
		start = startOf(messages, last, input.Text)
		finish = endOf(messages, start)
	}

	window := strings.Join(windowOf(messages, start, finish, input.Answer), "\n")

	return Step{
		Request: ModelRequest{
			Model: cheapTier, Effort: momentEffort, System: learnMomentInstructions,
			Prompt: fmt.Sprintf(windowShape, window), MaxTokens: momentMaxTokens, TimeoutMs: learnMomentTimeoutMs,
		},
		Carry: Carry{Cue: input.Cue, Window: window},
	}
}
