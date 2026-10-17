package auto

import (
	"bytes"
	"context"
	_ "embed" // the instructions of the judgement are a text file of the package
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	sessionctx "github.com/toejough/engram/internal/context"
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

// TurnEnd is a turn that has ended, as the memory mod hands it over: the session it was in, its final text, the text it
// began with and the origin of the prompt it began with (793d4g).
type TurnEnd struct {
	Session string `json:"session"`
	Answer  string `json:"answer"`
	Text    string `json:"text"`
	Origin  string `json:"origin"`
}

// Judge returns the request for the model that judges the turn a session's record holds, and false when the record
// holds no prompt (793d4g). It reads the record once, and redacts the lines of the window only. The turn begins at the
// last prompt that has the text the turn began with, or at the last prompt of the record when none has, and ends before
// the next prompt and after its final text; the lines before it are the context, and the final text is the turn's last
// line when no line of the turn holds its beginning. The notes are the ones nearest to the turn, which the lookup finds
// for the text of the turn's prompts and the model may name as already stating a lesson.
func Judge(ctx context.Context, record []byte, end TurnEnd, lookup Lookup) (Judgement, bool, error) {
	messages, last := messagesOf(record)
	if last < 0 {
		return Judgement{}, false, nil
	}

	start := startOf(messages, last, end.Text)
	before := linesOf(messages[max(start-contextLines, 0):start])
	turn := linesOf(messages[start:endOf(messages, start)])

	notes, err := lookup(ctx, promptsOf(turn))
	if err != nil {
		return Judgement{}, false, fmt.Errorf("looking up the notes: %w", err)
	}

	before, turn = fit(before, withAnswer(turn, end.Answer))
	window := slices.Concat(before, []string{divider}, turn)
	text := strings.NewReplacer("{window}", strings.Join(window, "\n"), "{notes}", noteLines(notes)).Replace(instructions)

	return Judgement{
		Model: judgeModel, Effort: judgeEffort, Prompt: text, MaxTokens: judgeMaxTokens, TimeoutMs: judgeTimeoutMs,
	}, true, nil
}

// unexported constants.
const (
	agentLabel = "Agent: "
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
	toolLabel = "Agent used "
	userLabel = "User: "
	// windowLimit is how many characters the window holds at most.
	windowLimit = 8000
)

//go:embed prompts/judge.txt
var instructions string

// message is a message of the conversation before it is a line of the window: the label that opens its line, and its
// text. The text of a tool's name is written as it is, and the text of every other message is redacted.
type message struct {
	label string
	text  string
	plain bool
}

// agentLine is the line of a text of the agent.
func agentLine(text string) string {
	return agentLabel + messageText(text)
}

// agentMessagesOf reads the texts and the tool uses of one message of the agent.
func agentMessagesOf(content json.RawMessage) []message {
	blocks := blocksOf(content)
	messages := make([]message, 0, len(blocks))

	for _, block := range blocks {
		switch {
		case block.Type == "text" && holdsText(block.Text):
			messages = append(messages, message{label: agentLabel, text: block.Text})
		case block.Type == "tool_use":
			messages = append(messages, message{label: toolLabel, text: oneLined(block.Name), plain: true})
		}
	}

	return messages
}

// clipped is the first limit characters of the text.
func clipped(text string, limit int) string {
	if runes := []rune(text); len(runes) > limit {
		return string(runes[:limit])
	}

	return text
}

// endOf is the index of the message after the last of the turn that begins at start: the next prompt, or the end.
func endOf(messages []message, start int) int {
	next := slices.IndexFunc(messages[start+1:], func(one message) bool { return one.label == userLabel })
	if next < 0 {
		return len(messages)
	}

	return start + 1 + next
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

// holdsText reports whether a message has a line of its own: its text is not empty once its white space, control and
// format characters are gone, as oneLined writes it, or it holds a break of its own, which oneLined writes as a mark.
func holdsText(text string) bool {
	return strings.ContainsFunc(text, func(r rune) bool {
		return isLineBreak(r) || (!unicode.IsSpace(r) && !unicode.IsControl(r) && !unicode.Is(unicode.Cf, r))
	})
}

// isLineBreak reports a character that ends a line.
func isLineBreak(r rune) bool {
	return r == '\n' || r == '\r' || r == '\v' || r == '\f' || r == '\u0085' ||
		unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r)
}

// linesOf writes messages as one line each, redacted, written as one line and cut to the limit of a line.
func linesOf(messages []message) []string {
	lines := make([]string, 0, len(messages))

	for _, one := range messages {
		if one.plain {
			lines = append(lines, one.label+one.text)
		} else {
			lines = append(lines, one.label+messageText(one.text))
		}
	}

	return lines
}

// messageText is the text of a message as a line holds it: redacted before anything is cut, written as one line, and
// cut to the limit of a line (793d4j, 793d4g).
func messageText(text string) string {
	return clipped(oneLined(Redact(text)), lineLimit)
}

// messagesOf reads the prompts of the user and the texts and tool uses of the agent out of a session's record, in
// order, and returns the messages with the index of the last prompt, or -1 when there is none. A tool's result is not a
// prompt, nor is a row that the harness wrote (793d4g): one marked as meta, or stamped with an origin the user does not
// send. A message with no text of its own is none. Nothing is redacted: the lines of the window are the only ones that
// are.
func messagesOf(record []byte) ([]message, int) {
	messages := make([]message, 0, bytes.Count(record, []byte("\n"))+1)
	prompt := -1

	for raw := range bytes.SplitSeq(record, []byte("\n")) {
		var row recordRow

		if json.Unmarshal(raw, &row) != nil || row.IsSidechain {
			continue
		}

		switch row.Type {
		case "user":
			if row.IsMeta || (row.Origin.Kind != "" && !UserSent(row.Origin.Kind)) {
				continue
			}

			if text := promptTextOf(row.Message.Content); holdsText(text) {
				prompt = len(messages)
				messages = append(messages, message{label: userLabel, text: text})
			}
		case "assistant":
			messages = append(messages, agentMessagesOf(row.Message.Content)...)
		}
	}

	return messages, prompt
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

// promptTextOf reads the text of a prompt of the user: the content is a string, or text blocks, each cleaned of the
// noise of the harness. The content that holds the result of a tool is no prompt.
func promptTextOf(content json.RawMessage) string {
	var text string

	if json.Unmarshal(content, &text) == nil {
		return userText(text)
	}

	var parts []string

	for _, block := range blocksOf(content) {
		if block.Type == "tool_result" {
			return ""
		}

		if block.Type != "text" {
			continue
		}

		if cleaned := userText(block.Text); cleaned != "" {
			parts = append(parts, cleaned)
		}
	}

	return strings.Join(parts, "\n")
}

// promptsOf is the text of the prompts among the lines of a turn, which the notes nearest to the turn are looked up for
// (793d4g).
func promptsOf(turn []string) string {
	var texts []string

	for _, line := range turn {
		if text, isPrompt := strings.CutPrefix(line, userLabel); isPrompt {
			texts = append(texts, text)
		}
	}

	return clipped(strings.Join(texts, " "), lookupLimit)
}

// sizeOf is how many characters the lines take in the window, a line break after each.
func sizeOf(lines []string) int {
	size := 0

	for _, line := range lines {
		size += utf8.RuneCountInString(line) + 1
	}

	return size
}

// startOf is the index of the message the turn begins at: the last prompt that has the text, or the last prompt, which
// last is the index of, when none has it or there is no text.
func startOf(messages []message, last int, text string) int {
	want := oneLined(text)
	if want == "" {
		return last
	}

	for index := last; index >= 0; index-- {
		if messages[index].label == userLabel && oneLined(messages[index].text) == want {
			return index
		}
	}

	return last
}

// userText is the text of a message of the user without the noise of the harness, as the transcript-reader reads it
// (792b15c3), or nothing when none of it is the user's.
func userText(text string) string {
	cleaned, drop := sessionctx.CleanText(text)
	if drop {
		return ""
	}

	return cleaned
}

// withAnswer ends the turn with the final text: the lines after the last line that is the text belong to another turn,
// and the text is added as the last line unless a line of the turn already holds its beginning.
func withAnswer(turn []string, answer string) []string {
	line := agentLine(answer)

	for index, existing := range slices.Backward(turn) {
		if existing == line {
			return turn[:index+1]
		}
	}

	probe := clipped(strings.TrimPrefix(line, agentLabel), answerProbe)

	if probe == "" || slices.ContainsFunc(turn, func(existing string) bool { return strings.Contains(existing, probe) }) {
		return turn
	}

	return append(turn, line)
}
