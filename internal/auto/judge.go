package auto

import (
	"bytes"
	_ "embed" // the instructions of the judgement are a text file of the package
	"encoding/json"
	"slices"
	"strings"
	"unicode"
)

// Judgement is the request of the call of the model that judges a turn: the fields the host's model call takes
// (793d4g).
type Judgement struct {
	Model     string `json:"model"`
	Effort    string `json:"effort"`
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"maxTokens"`
	TimeoutMs int    `json:"timeoutMs"`
}

// TurnEnd is a turn that has ended, as the memory mod hands it over: the session it was in and its final text (793d4g).
type TurnEnd struct {
	Session string `json:"session"`
	Answer  string `json:"answer"`
}

// Judge returns the request for the model that judges the turn a session's record ends with, and false when the record
// holds no prompt (793d4g). The turn is the last prompt of the record and everything after it, the lines before it are
// the context, and the final text is the turn's last line when no line of the turn holds its beginning.
func Judge(record []byte, answer string) (Judgement, bool) {
	messages := messagesOf(record)

	start := -1

	for index, entry := range messages {
		if entry.isPrompt {
			start = index
		}
	}

	if start < 0 {
		return Judgement{}, false
	}

	window := slices.Concat(linesOf(messages[:start]), []string{divider}, withAnswer(linesOf(messages[start:]), answer))
	prompt := strings.NewReplacer("{window}", strings.Join(window, "\n"), "{notes}", noNotes).Replace(instructions)

	return Judgement{
		Model: judgeModel, Effort: judgeEffort, Prompt: prompt, MaxTokens: judgeMaxTokens, TimeoutMs: judgeTimeoutMs,
	}, true
}

// unexported constants.
const (
	// answerProbe is how many characters of the final text tell that a line of the turn already holds it.
	answerProbe = 80
	// divider is the line that ends the context and opens the turn.
	divider        = "--- the turn ---"
	judgeEffort    = "low"
	judgeMaxTokens = 1500
	judgeModel     = "haiku"
	judgeTimeoutMs = 60000
	// lineBreak stands in a line for a break of the message's own.
	lineBreak = " ⏎ "
	noNotes   = "(none)"
)

//go:embed prompts/judge.txt
var instructions string

// message is a prompt of the user, or a text or a tool use of the agent, as the one line it makes in the window.
type message struct {
	line     string
	isPrompt bool
}

// agentLine is the line of a text of the agent.
func agentLine(text string) string {
	return "Agent: " + oneLined(text)
}

// agentMessagesOf reads the texts and the tool uses of one message of the agent.
func agentMessagesOf(content json.RawMessage) []message {
	blocks := blocksOf(content)
	messages := make([]message, 0, len(blocks))

	for _, block := range blocks {
		switch {
		case block.Type == "text" && oneLined(block.Text) != "":
			messages = append(messages, message{line: agentLine(block.Text)})
		case block.Type == "tool_use":
			messages = append(messages, message{line: "Agent used " + oneLined(block.Name)})
		}
	}

	return messages
}

// isLineBreak reports a character that ends a line.
func isLineBreak(r rune) bool {
	switch r {
	case '\n', '\r', '\v', '\f', '\u0085', ' ', ' ':
		return true
	default:
		return false
	}
}

// linesOf returns the lines of the messages.
func linesOf(messages []message) []string {
	lines := make([]string, 0, len(messages))

	for _, entry := range messages {
		lines = append(lines, entry.line)
	}

	return lines
}

// messagesOf reads the prompts of the user and the texts and tool uses of the agent out of a session's record, in
// order. A tool's result is not a prompt, and a message with no text of its own makes no line.
func messagesOf(record []byte) []message {
	messages := make([]message, 0, bytes.Count(record, []byte("\n"))+1)

	for line := range bytes.SplitSeq(record, []byte("\n")) {
		var row recordRow

		if json.Unmarshal(line, &row) != nil || row.IsSidechain {
			continue
		}

		switch row.Type {
		case "user":
			if text := promptTextOf(row.Message.Content); oneLined(text) != "" {
				messages = append(messages, message{line: "User: " + oneLined(text), isPrompt: true})
			}
		case "assistant":
			messages = append(messages, agentMessagesOf(row.Message.Content)...)
		}
	}

	return messages
}

// oneLined writes text as one line: a break of the text's own is written ` ⏎ `, any other white space is a space, a
// control or format character is dropped, and runs of spaces are one space, so that no message makes a line of its own
// (793d4g).
func oneLined(text string) string {
	var line strings.Builder

	for _, r := range text {
		switch {
		case isLineBreak(r):
			line.WriteString(lineBreak)
		case unicode.IsSpace(r):
			line.WriteRune(' ')
		case unicode.IsControl(r), unicode.Is(unicode.Cf, r):
			continue
		default:
			line.WriteRune(r)
		}
	}

	return strings.Join(strings.Fields(line.String()), " ")
}

// promptTextOf reads the text of a prompt of the user: the content is a string, or text blocks. The content that holds
// the result of a tool is no prompt.
func promptTextOf(content json.RawMessage) string {
	var text string

	if json.Unmarshal(content, &text) == nil {
		return text
	}

	var parts []string

	for _, block := range blocksOf(content) {
		if block.Type == "tool_result" {
			return ""
		}

		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}

	return strings.Join(parts, "\n")
}

// withAnswer adds the final text as the last line of the turn unless a line of the turn already holds its beginning.
func withAnswer(turn []string, answer string) []string {
	line := agentLine(answer)
	probe := strings.TrimPrefix(line, "Agent: ")

	if runes := []rune(probe); len(runes) > answerProbe {
		probe = string(runes[:answerProbe])
	}

	if probe == "" || slices.ContainsFunc(turn, func(existing string) bool { return strings.Contains(existing, probe) }) {
		return turn
	}

	return append(turn, line)
}
