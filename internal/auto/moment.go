package auto

import (
	"bytes"
	_ "embed" // the instructions of the judgement are a text file of the package
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// Exported constants.
const (
	// CueBatchDone is the cue of a batch of tool calls that resolved, before the agent's next message (855d2c).
	CueBatchDone = "batch-done"
	// CueBeforeTool is the cue of a tool call about to run (855d2c).
	CueBeforeTool = "before-tool"
	// CueFailure is the cue of a tool call that failed (855d2c).
	CueFailure = "failure"
	// CueTurnEnd is the cue of a turn that ended, the learn moment (855d4a).
	CueTurnEnd = "turn-end"
)

// Carry is what a step hands the next one with a request, and the mod reads no part of.
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

// Step is what a step prints for a judgement: the request, and the carry for the next step.
type Step struct {
	Request ModelRequest `json:"request"`
	Carry   Carry        `json:"carry"`
	// Write tells the mod that a write follows the delivery: a deep recall's.
	Write bool `json:"write,omitempty"`
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

// Moment returns the request that judges whether a moment is one for recall, or, at the end of a turn, one for learn
// (855d4a): the conversation lines before it and the line of the moment, as the window of a judgement of a turn is
// built (793d4g).
func Moment(record []byte, input MomentInput) Step {
	messages, last := messagesOf(record)

	if input.Cue == CueTurnEnd {
		return learnMoment(messages, last, input)
	}

	before := linesOf(messages[max(len(messages)-contextLines, 0):])
	turn := []string{escaped(userLabel + messageText(input.Text))}

	if slices.Contains([]string{CueBatchDone, CueBeforeTool, CueFailure}, input.Cue) {
		before, turn = toolWindow(before, turn[0], eventLines(input.Cue, input.Event))
	} else {
		before, turn = fit(before, turn)
	}

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
	// callAndFailure is the lines of the event of a call that failed: the call and the failure.
	callAndFailure = 2
	// learnMomentTimeoutMs is the row of the model call of the learn-moment judgement (793d2b).
	learnMomentTimeoutMs = 10000
	// linesPerCall is the lines of a call of a batch: the call and its response.
	linesPerCall    = 2
	momentEffort    = "low"
	momentMaxTokens = 200
	// windowShape is the prompt of a request that holds the window only: the window between its tags.
	windowShape = "<window>\n%s\n</window>"
)

//go:embed prompts/moment_learn.txt
var learnMomentInstructions string

//go:embed prompts/moment_recall.txt
var recallMoment string

// compact is the text of a JSON value on one line, redacted after it is decoded: a credential written in the escaped
// form of JSON is redacted as the text it spells, in every string and key of the value, the value of a sensitive name
// is replaced whole, and the value is written again without escaping the characters of HTML. A value that is
// not JSON is returned as it is, for the line to redact.
func compact(raw json.RawMessage) string {
	value, isJSON := decoded(string(raw))
	if !isJSON {
		return string(raw)
	}

	if text, isText := value.(string); isText {
		return redactedText(text)
	}

	return encoded(redactedValue(value), string(raw))
}

// decoded is the value of a text of JSON, numbers kept as written, and whether the text is JSON.
func decoded(text string) (any, bool) {
	if !json.Valid([]byte(text)) {
		return nil, false
	}

	var value any

	decoder := json.NewDecoder(strings.NewReader(text))
	decoder.UseNumber()

	return value, decoder.Decode(&value) == nil
}

// encoded is a decoded value written on one line without escaping the characters of HTML, or the fallback.
func encoded(value any, fallback string) string {
	var buffer bytes.Buffer

	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)

	encodeErr := encoder.Encode(value)
	if encodeErr != nil {
		return fallback
	}

	return strings.TrimSuffix(buffer.String(), "\n")
}

// eventLines writes the event of a tool cue as the lines of the window that end it (855d4a): the call, with its input;
// for a failure the error after it; for a batch each call with its response.
func eventLines(cue string, event ToolEvent) []string {
	messages := make([]message, 0, linesPerCall*len(event.Calls)+callAndFailure)

	for _, call := range event.Calls {
		messages = append(messages,
			message{label: toolLabel, text: call.Tool + " with " + compact(call.Input)},
			message{label: toolLabel, text: call.Tool + " returned: " + compact(call.Response)})
	}

	if len(event.Calls) == 0 {
		messages = append(messages, message{label: toolLabel, text: event.Tool + " with " + compact(event.Input)})
	}

	if cue == CueFailure {
		messages = append(messages, message{label: toolLabel, text: event.Tool + " failed: " + event.Error})
	}

	return linesOf(messages)
}

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

// redactedItem is the value of a key redacted: replaced whole when the key names a secret and the value is a scalar.
func redactedItem(key string, item any) any {
	switch item.(type) {
	case string, json.Number, bool:
		if sensitiveKey.MatchString(key) {
			return redacted
		}
	}

	return redactedValue(item)
}

// redactedText is a string of a value redacted: a string that is an object or a list written as JSON (a response that
// was encoded twice) is decoded once more and redacted as a value, any other string as text.
func redactedText(text string) string {
	if trimmed := strings.TrimSpace(text); strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if inner, isJSON := decoded(trimmed); isJSON {
			return encoded(redactedValue(inner), Redact(text))
		}
	}

	return Redact(text)
}

// redactedValue is a decoded JSON value with every string and every key of an object redacted, and the scalar value of
// a key that names a secret replaced whole.
func redactedValue(value any) any {
	switch typed := value.(type) {
	case string:
		return redactedText(typed)
	case []any:
		for index, item := range typed {
			typed[index] = redactedValue(item)
		}

		return typed
	case map[string]any:
		redactedMap := make(map[string]any, len(typed))

		for key, item := range typed {
			redactedMap[Redact(key)] = redactedItem(key, item)
		}

		return redactedMap
	default:
		return value
	}
}

// toolWindow is the context and the turn's lines of the window of a tool cue: the last lines of the
// conversation, the text the turn began with before them when they do not hold it, and the lines of the event.
func toolWindow(before []string, turnLine string, event []string) ([]string, []string) {
	if !slices.Contains(before, turnLine) {
		before = slices.Insert(before, 0, turnLine)
	}

	return fit(before, event)
}
