package auto

import (
	"bytes"
	_ "embed" // the instructions of the judgement are a text file of the package
	"encoding/json"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
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
// the context, and the final text is the turn's last line when no line of the turn holds its beginning. The notes are
// the ones nearest to the turn, which the model may name as already stating a lesson.
func Judge(record []byte, answer string, notes []Note) (Judgement, bool) {
	lines, prompt := linesOf(record)
	if prompt < 0 {
		return Judgement{}, false
	}

	context, turn := fit(lines[max(prompt-contextLines, 0):prompt], withAnswer(lines[prompt:], answer))
	window := slices.Concat(context, []string{divider}, turn)
	text := strings.NewReplacer("{window}", strings.Join(window, "\n"), "{notes}", noteLines(notes)).Replace(instructions)

	return Judgement{
		Model: judgeModel, Effort: judgeEffort, Prompt: text, MaxTokens: judgeMaxTokens, TimeoutMs: judgeTimeoutMs,
	}, true
}

// TurnText is the text of the prompts of the turn a session's record ends with, which the notes nearest to the turn are
// looked up for (793d4g), or nothing when the record holds no prompt.
func TurnText(record []byte) string {
	lines, prompt := linesOf(record)
	if prompt < 0 {
		return ""
	}

	var texts []string

	for _, line := range lines[prompt:] {
		if text, isPrompt := strings.CutPrefix(line, "User: "); isPrompt {
			texts = append(texts, text)
		}
	}

	return clipped(strings.Join(texts, " "), lookupLimit)
}

// unexported constants.
const (
	// answerProbe is how many characters of the final text tell that a line of the turn already holds it.
	answerProbe = 80
	// contextLines is how many lines of the conversation before the turn the window holds at most.
	contextLines = 6
	// divider is the line that ends the context and opens the turn.
	divider        = "--- the turn ---"
	judgeEffort    = "low"
	judgeMaxTokens = 1500
	judgeModel     = "haiku"
	judgeTimeoutMs = 60000
	// leftOut stands in the turn for the lines of its middle that the window could not hold.
	leftOut = "--- some of the turn is left out ---"
	// lineBreak stands in a line for a break of the message's own.
	lineBreak = " ⏎ "
	// lineLimit is how many characters of a message a line holds at most.
	lineLimit = 1200
	// lookupLimit is how many characters of the text of the turn the notes nearest to it are looked up for.
	lookupLimit = 500
	noNotes     = "(none)"
	// noteLimit is how many characters of the body of a note a line holds at most.
	noteLimit = 400
	// windowLimit is how many characters the window holds at most.
	windowLimit = 8000
)

//go:embed prompts/judge.txt
var instructions string

// agentLine is the line of a text of the agent.
func agentLine(text string) string {
	return "Agent: " + messageText(text)
}

// agentLinesOf reads the lines of the texts and the tool uses of one message of the agent.
func agentLinesOf(content json.RawMessage) []string {
	blocks := blocksOf(content)
	lines := make([]string, 0, len(blocks))

	for _, block := range blocks {
		switch {
		case block.Type == "text" && messageText(block.Text) != "":
			lines = append(lines, agentLine(block.Text))
		case block.Type == "tool_use":
			lines = append(lines, "Agent used "+oneLined(block.Name))
		}
	}

	return lines
}

// clipped is the first limit characters of the text.
func clipped(text string, limit int) string {
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit])
	}

	return text
}

// fit cuts the window to windowLimit characters: the oldest lines of the context go first, and then the middle of the
// turn, where leftOut stands in place of the lines left out.
func fit(context, turn []string) ([]string, []string) {
	for len(context) > 0 && sizeOf(context)+sizeOf(turn) > windowLimit {
		context = context[1:]
	}

	if sizeOf(turn) <= windowLimit {
		return context, turn
	}

	budget := windowLimit - sizeOf([]string{turn[0], leftOut})
	last := len(turn)

	for last > 1 && budget >= sizeOf(turn[last-1:last]) {
		last--
		budget -= sizeOf(turn[last : last+1])
	}

	return context, slices.Concat([]string{turn[0], leftOut}, turn[last:])
}

// isLineBreak reports a character that ends a line.
func isLineBreak(r rune) bool {
	return r == '\n' || r == '\r' || r == '\v' || r == '\f' || r == '\u0085' ||
		unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r)
}

// linesOf reads the prompts of the user and the texts and tool uses of the agent out of a session's record, in order,
// as one line each, and returns the lines with the index of the line of the last prompt, or -1 when there is none. A
// tool's result is not a prompt, and a message with no text of its own makes no line.
func linesOf(record []byte) ([]string, int) {
	lines := make([]string, 0, bytes.Count(record, []byte("\n"))+1)
	prompt := -1

	for raw := range bytes.SplitSeq(record, []byte("\n")) {
		var row recordRow

		if json.Unmarshal(raw, &row) != nil || row.IsSidechain {
			continue
		}

		switch row.Type {
		case "user":
			if text := messageText(promptTextOf(row.Message.Content)); text != "" {
				prompt = len(lines)
				lines = append(lines, "User: "+text)
			}
		case "assistant":
			lines = append(lines, agentLinesOf(row.Message.Content)...)
		}
	}

	return lines, prompt
}

// messageText is the text of a message as a line holds it: redacted before anything is cut, written as one line, and
// cut to the limit of a line (793d4j, 793d4g).
func messageText(text string) string {
	return clipped(oneLined(Redact(text)), lineLimit)
}

// noteLines writes the notes nearest to the turn, one line each, in the form of a line of a block: redacted and written
// as one line as the delivery writes it, and with the body cut to the limit of a note (793d4g, 793d4j). It writes
// (none) when there are no notes.
func noteLines(notes []Note) string {
	if len(notes) == 0 {
		return noNotes
	}

	lines := make([]string, 0, len(notes))

	for _, note := range notes {
		lines = append(lines, "- "+oneLine(Redact(note.Path))+" ("+oneLine(Redact(note.Situation))+"): "+
			clipped(oneLine(Redact(note.Body)), noteLimit))
	}

	return strings.Join(lines, "\n")
}

// oneLined writes text as one line as oneLine does, with a break of the text's own written ` ⏎ `, so that no message
// makes a line of its own (793d4g).
func oneLined(text string) string {
	var broken strings.Builder

	for _, r := range text {
		if isLineBreak(r) {
			broken.WriteString(lineBreak)
		} else {
			broken.WriteRune(r)
		}
	}

	return oneLine(broken.String())
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

// sizeOf is how many characters the lines take in the window, a line break after each.
func sizeOf(lines []string) int {
	size := 0

	for _, line := range lines {
		size += utf8.RuneCountInString(line) + 1
	}

	return size
}

// withAnswer adds the final text as the last line of the turn unless a line of the turn already holds its beginning.
func withAnswer(turn []string, answer string) []string {
	line := agentLine(answer)
	probe := clipped(strings.TrimPrefix(line, "Agent: "), answerProbe)

	if probe == "" || slices.ContainsFunc(turn, func(existing string) bool { return strings.Contains(existing, probe) }) {
		return turn
	}

	return append(turn, line)
}
