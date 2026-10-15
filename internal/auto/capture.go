package auto

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Lesson is what the capture judgement found worth a note; Covered is the path of the nearest note that already
// states it, or empty.
type Lesson struct {
	Kind, Slug, Situation, Covered string
	Values                         [][2]string
}

// Message is a message of the session, in the shape of `$.session.messages()`: the speaker is its role, a structured
// field, never anything in its text.
type Message struct {
	Role        string            `json:"role"`
	Text        string            `json:"text"`
	ToolUses    []messageTool     `json:"toolUses"`
	ToolResults []json.RawMessage `json:"toolResults"`
}

// unexported constants.
const (
	amendCmd      = "amend"
	answerProbe   = 80
	learnArgs     = 9
	lessonLimit   = 10
	nearLimit     = 8
	needleLimit   = 40
	partLimit     = 1200
	pollMs        = 100
	pollSlow      = 5
	pollTries     = 24
	recentCount   = 6
	roleAssistant = "assistant"
	roleUser      = "user"
	slugLimit     = 60
	sourceNote    = "captured from the conversation"
	targetFlag    = "--target"
	windowLimit   = 8000
)

// unexported variables.
var (
	// what a harness puts in a user turn that the user did not say: injected reminders, a skill's body, the output of a
	// local command
	harnessText = regexp.MustCompile(
		`^[` + space + `]*(?:<system-reminder|<local-command-|Base directory for this skill:)`)
	// every character that breaks a line, or could be read as breaking one: the line feed and its kin, the separators
	// of the files, groups and records, the control characters, next line, the line and paragraph separators (793b396)
	oneLine     = regexp.MustCompile(`[` + space + `]*[\x00-\x08\x0a-\x1f\x7f-\x9f\x{2028}\x{2029}]+[` + space + `]*`)
	slugEdge    = regexp.MustCompile(`^-+|-+$`)
	slugRun     = regexp.MustCompile(`[^a-z0-9]+`)
	summaryText = regexp.MustCompile(`<summary>((?s:.*?))</summary>`)
)

// capture judges a window against the nearest notes and makes the writes it calls for. Secrets are redacted before
// anything reaches the model or a note. A window with no words of the user is nothing to judge.
func (e *engine) capture(conversation, source string) {
	text := redact(conversation)
	if !strings.Contains(text, "User:") {
		return
	}

	near := e.neighbours(parseQuery(e.exec(queryArgv(cueCapture, queryText(text), nil), deadlineNearest)))
	parse := func(reply string) []Lesson { return parseLessons(reply, near) }
	lessons := judged(e, e.tiers.captureRequest(text, near), judgeCapture, parse)
	e.logf("capture judged %d lesson(s) against %d nearest note(s)", len(lessons), len(near))

	for _, lesson := range lessons {
		if lesson.Covered != "" {
			e.exec([]string{engramCmd, amendCmd, targetFlag, lesson.Covered, "--activate"}, deadlineCommand)
		} else {
			e.learn(lesson, source)
		}
	}
}

// captureJob is the capture of a turn that ended (793b279): the window of the turn, judged against the nearest notes.
// The turns of its session are then marked captured, so that the backlog does not read the session again once it is
// quiet (793e8g); a failure of the mark is the System's own bookkeeping and goes to the debug log only. A turn whose
// capture did not end because the session did is read by the backlog capture, once its source is quiet.
func (e *engine) captureJob() Done {
	if conversation := windowFromMessages(e.job.Messages, e.job.Answer); conversation != "" {
		e.capture(conversation, sourceFor(e.job.SessionID, e.job.Turn, ""))
		e.attempt("", func() { e.markTurns(e.job.Transcript) })
	}

	return Done{}
}

// learn writes an absent lesson with the existing `engram learn`; a free-text value is `--flag=value` so that one that
// starts with a dash is not read as a flag. `engram learn` prints the path of the note, which the session remembers.
func (e *engine) learn(lesson Lesson, source string) {
	argv := make([]string, 0, learnArgs+len(lesson.Values))
	argv = append(argv, engramCmd, "learn", lesson.Kind, "--slug", lesson.Slug, "--position", "top", "--source="+source)
	argv = append(argv, "--situation="+lesson.Situation)
	values := make([]string, 0, len(lesson.Values))

	for _, field := range lesson.Values {
		argv, values = append(argv, "--"+field[0]+"="+field[1]), append(values, field[1])
	}

	lines := strings.Split(strings.TrimSpace(e.exec(argv, deadlineCommand)), "\n")
	e.count(Counts{Added: 1})

	if name := path.Base(lines[len(lines)-1]); strings.HasSuffix(name, ".md") {
		body := strings.Join(values, ". ")
		e.work.Written = append(e.work.Written, Note{Path: name, Situation: lesson.Situation, Body: body})
	}
}

// neighbours are the notes the judge compares a window with: the vault's nearest, then the notes this session already
// wrote, so that a lesson it recorded is never recorded twice whatever the query ranked.
func (e *engine) neighbours(items []Note) []Note {
	near := []Note{}

	for _, found := range items {
		if found.Kind != chunkKind && found.Situation != "" && len(near) < nearLimit {
			near = append(near, found)
		}
	}

	for _, note := range e.work.Written {
		if len(near) < nearLimit && !slices.ContainsFunc(near, func(found Note) bool { return found.Path == note.Path }) {
			near = append(near, note)
		}
	}

	return near
}

// subagentJob is the capture of a subagent that stopped: its transcript file lags its stop event (SP5), so it is read
// again until it holds the last message, or the tries are spent. The file lags its stop by about 100 ms, so the first
// four polls are 100 ms apart (a capture starts within 200 ms of the file being complete, 793d27b1) and the rest 500 ms
// apart, since every poll is a round whose record holds the file it read; the patience is 9.9 s in all.
func (e *engine) subagentJob() Done {
	var file string

	for tries := 1; ; tries++ {
		file, _ = e.read(e.job.Path)
		if (file != "" && transcriptHolds(file, e.job.Last)) || tries >= pollTries {
			break
		}

		e.wait(pollMs * (1 + (pollSlow-1)*min(tries/pollSlow, 1)))
	}

	e.capture(windowFromTranscript(file, e.job.Last), sourceFor(e.job.SessionID, e.job.Turn, e.job.Agent))

	return Done{}
}

// messageTool is a tool an assistant message used.
type messageTool struct {
	Tool string `json:"tool"`
}

// transcriptBlock is a block of the content of a transcript row; a row's content is a string or a list of them.
type transcriptBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
}

// transcriptRow is a row of a transcript file: a Claude Code row has `type` user or assistant and its content in
// `message`; a Pi row has `type` message and its role in `message.role`.
type transcriptRow struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// workState is what the queued jobs keep between them (the mod hands it back and never reads it): the notes this
// session wrote, the offers curate left alone, and the sources that could not be read.
type workState struct {
	Written    []Note   `json:"written"`
	Skipped    []string `json:"skipped"`
	Unreadable []string `json:"unreadable"`
}

// agentLine is what the agent said, as one line of a window: redacted before it is cut and kept on one line, whether it
// is a message of the transcript or the answer or last message the event brought (793b396).
func agentLine(text string) string {
	return "Agent: " + clip(oneLined(redact(strings.TrimSpace(text))))
}

func backlogSourceFor(path, position string) string {
	return fmt.Sprintf("%s; written automatically (source %s, position %s)", sourceNote, path, position)
}

// clip shortens a message to the limit of a part of a window.
func clip(text string) string {
	if len([]rune(text)) > partLimit {
		return cutEnd(text, partLimit) + "..."
	}

	return text
}

// linesOf renders the messages.
func linesOf(messages []Message) []string {
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		lines = append(lines, render(message)...)
	}

	return lines
}

// messageOf is the message of a row: the text blocks (the harness's own text left out of a user's turn) and the names
// of the tools it used.
func messageOf(role string, content json.RawMessage) Message {
	var (
		blocks []transcriptBlock
		text   string
	)

	if json.Unmarshal(content, &text) == nil {
		blocks = append(blocks, transcriptBlock{Type: "text", Text: text})
	} else {
		_ = json.Unmarshal(content, &blocks)
	}

	message := Message{Role: role}
	texts := make([]string, 0, len(blocks))

	for _, block := range blocks {
		if block.Type == "text" && (role != roleUser || !harnessText.MatchString(block.Text)) {
			texts = append(texts, block.Text)
		}

		if block.Type == "tool_use" || block.Type == "toolCall" {
			message.ToolUses = append(message.ToolUses, messageTool{Tool: block.Name})
		}
	}

	message.Text = strings.Join(texts, "\n")

	return message
}

// messagesFromTranscript reads the rows of a transcript file in the shape of `$.session.messages()`. Any other row,
// and any other role (a tool result, program output a harness records as a message), is nobody's words.
func messagesFromTranscript(file string) []Message {
	var messages []Message

	for line := range strings.SplitSeq(file, "\n") {
		var row transcriptRow
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}

		role := map[bool]string{true: row.Message.Role, false: row.Type}[row.Type == "message"]
		if role == roleUser || role == roleAssistant {
			messages = append(messages, messageOf(role, row.Message.Content))
		}
	}

	return messages
}

// oneLined keeps a message on one line of the window, so that every line begins with a label this package wrote: a
// label inside a message's own words is that message's, not another speaker's.
func oneLined(text string) string { return oneLine.ReplaceAllLiteralString(text, " ⏎ ") }

// parseLessons reads the judge's reply strictly (793b299): exactly one JSON document, an array of at most ten lessons,
// each an object with exactly the keys of its kind (`feedback`: behavior, impact, action; `fact`: subject, predicate,
// object, beside kind, slug, situation and covered), every string non-empty and bounded, `covered` null or the number
// of a nearest note. Anything else is an error for the whole reply and nothing is written. A lesson whose slug an
// earlier one has is not read further.
func parseLessons(text string, near []Note) []Lesson {
	var lessons []Lesson

	for _, entry := range list(text, lessonLimit) {
		fields, _ := entry.(map[string]any)
		kind, _ := fields["kind"].(string)
		names := noteFields(kind)
		check(names != nil, "a lesson has a kind that is neither feedback nor fact")

		fields = object(entry, append([]string{"kind", "slug", "situation", "covered"}, names...)...)
		number, isNumber := whole(fields["covered"])
		check(fields["covered"] == nil || (isNumber && number >= 1 && number <= len(near)), "covered is no nearest note")

		lesson := Lesson{Kind: kind, Slug: slugOf(short(fields["slug"]))}
		if slices.ContainsFunc(lessons, func(seen Lesson) bool { return seen.Slug == lesson.Slug }) {
			continue
		}

		if isNumber {
			lesson.Covered = near[number-1].Path
		}

		lesson.Situation = redact(short(fields["situation"]))

		for _, name := range names {
			lesson.Values = append(lesson.Values, [2]string{name, redact(short(fields[name]))})
		}

		lessons = append(lessons, lesson)
	}

	return lessons
}

// queryText is what the nearest notes are looked up with: what the user said first, then the rest.
func queryText(window string) string {
	var first, rest []string

	for line := range strings.SplitSeq(window, "\n") {
		if strings.HasPrefix(line, "User:") {
			first = append(first, line)
		} else {
			rest = append(rest, line)
		}
	}

	return strings.Join(append(first, rest...), " ")
}

// recentTexts is what the in-turn classification reads (793b320): the last texts of the user and the agent, no tool
// input or result, each redacted and cut like the lines of a window, the oldest cut first to the window limit. The
// step's own text is added when the messages lag behind it.
func recentTexts(messages []Message, answer string) string {
	spoken := make([]Message, 0, len(messages))

	for _, message := range messages {
		if strings.TrimSpace(message.Text) != "" {
			spoken = append(spoken, Message{Role: message.Role, Text: message.Text})
		}
	}

	lines := linesOf(spoken[max(len(spoken)-recentCount, 0):])
	lines = withAnswer(lines, answer, agentLine)

	return cutStart(strings.Join(lines, "\n"), windowLimit)
}

// render turns a message into the lines the judge reads. Only what the user and the agent said is read (793b297): a
// tool's input and result are untrusted content (a file, a web page, another program's output), so they never enter
// the window, only the name of the tool that was used. Every part is redacted before it is cut. A task notification is
// not the user speaking: only its summary is kept, because its result is a subagent's reply, which that subagent's
// own capture has judged.
func render(message Message) []string {
	lines := make([]string, 0, len(message.ToolUses)+1)

	said := strings.TrimSpace(message.Text)
	label := map[bool]string{true: "User", false: "Agent"}[message.Role == roleUser]

	if label == "User" && strings.HasPrefix(said, "<task-notification>") {
		summary, found := summaryOf(said)
		said = "Task notification: " + map[bool]string{true: summary, false: "a task finished"}[found]
	}

	if said != "" {
		lines = append(lines, label+": "+clip(oneLined(redact(said))))
	}

	for _, use := range message.ToolUses {
		lines = append(lines, clip("Agent used "+oneLined(use.Tool)))
	}

	return lines
}

// slugOf is a kebab-case tag for the file name of a note.
func slugOf(text string) string {
	slug := slugRun.ReplaceAllLiteralString(strings.ToLower(text), "-")
	slug = cutEnd(slugEdge.ReplaceAllLiteralString(slug, ""), slugLimit)

	return map[bool]string{true: "captured-lesson", false: slug}[slug == ""]
}

// sourceFor names a note written from a live turn; a note written from a subagent's transcript names the subagent
// too. backlogSourceFor names a note written from a transcript read after the fact: the source and the position in it.
func sourceFor(sessionID string, turn int, agentID string) string {
	if agentID != "" {
		agentID = ", subagent " + agentID
	}

	return fmt.Sprintf("%s; written automatically (session %s, turn %d%s)", sourceNote, sessionID, turn, agentID)
}

// transcriptHolds says whether the subagent's file is complete: it is not when the stop event fires (SP5); it is when
// an assistant row holds the last message.
func transcriptHolds(file, last string) bool {
	needle := cutEnd(strings.TrimSpace(last), needleLimit)

	return last == "" || slices.ContainsFunc(messagesFromTranscript(file), func(message Message) bool {
		return message.Role == roleAssistant && strings.Contains(message.Text, needle)
	})
}

// window is lines written as a window, cut to the window limit.
func window(lines []string) string { return cutEnd(strings.Join(lines, "\n"), windowLimit) }

// windowFromMessages is the turn: the assistant message before the turn's last human prompt (a correction needs the
// message it corrects), the prompt, and everything after it. The answer is appended when the messages lag behind it.
func windowFromMessages(messages []Message, answer string) string {
	start := -1

	for index, message := range messages {
		if message.Role == roleUser && strings.TrimSpace(message.Text) != "" && len(message.ToolResults) == 0 {
			start = index
		}
	}

	if start < 0 {
		return ""
	}

	if start > 0 && messages[start-1].Role == roleAssistant {
		start--
	}

	lines := withAnswer(linesOf(messages[start:]), answer, agentLine)

	return window(lines)
}

// windowFromTranscript is a subagent's whole transcript, with the last message appended when the file lags.
func windowFromTranscript(file, last string) string {
	lines := linesOf(messagesFromTranscript(file))
	if strings.TrimSpace(last) != "" && !transcriptHolds(file, last) {
		lines = append(lines, agentLine(last))
	}

	return window(lines)
}

// windowsFromMessages is a whole conversation as capture windows, oldest first: each holds whole messages up to the
// window limit and starts with the last message of the one before (a correction needs the message it corrects).
func windowsFromMessages(messages []Message) []string {
	var (
		current [][]string
		size    int
	)

	windows := []string{}

	for _, message := range messages {
		lines := render(message)
		length := len([]rune(strings.Join(lines, "\n"))) + 1

		if len(lines) == 0 {
			continue
		}

		if len(current) > 0 && size+length > windowLimit {
			windows = append(windows, strings.Join(slices.Concat(current...), "\n"))
			current = current[len(current)-1:]
			size = len([]rune(strings.Join(current[0], "\n"))) + 1
		}

		current, size = append(current, lines), size+length
	}

	if len(current) > 0 {
		windows = append(windows, strings.Join(slices.Concat(current...), "\n"))
	}

	return windows
}

// withAnswer adds the answer of the turn when the lines lag behind it.
func withAnswer(lines []string, answer string, format func(string) string) []string {
	trimmed := strings.TrimSpace(answer)
	probe := cutEnd(oneLined(redact(trimmed)), answerProbe)

	if trimmed == "" || slices.ContainsFunc(lines, func(line string) bool { return strings.Contains(line, probe) }) {
		return lines
	}

	return append(lines, format(trimmed))
}
