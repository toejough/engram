//go:build network

package acceptance_test

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// unexported constants.
const (
	credentialFile = ".config/claude-key.env"
	engramMarker   = "[engram memory]"
	// fakeAPIKey and offlineBaseURL send a prompt to an endpoint that is not there: the prompt's hooks run, the model
	// is asked and never answers.
	fakeAPIKey        = "sk-ant-acceptance-not-a-real-key"
	liveBudgetUSD     = "2"
	liveModel         = "sonnet"
	maxLineBytes      = 16 << 20
	offlineBaseURL    = "http://127.0.0.1:9"
	offlineWait       = 60 * time.Second
	pollInterval      = 50 * time.Millisecond
	recordFlushWait   = 10 * time.Second
	resultsBuffered   = 16
	sessionTimeout    = 5 * time.Minute
	stopGrace         = 10 * time.Second
	tokenVariable     = "CLAUDE_CODE_OAUTH_TOKEN"
	uuidBytes         = 16
	uuidVariantBits   = 0x80
	uuidVariantByte   = 8
	uuidVersionByte   = 6
	uuidVersionNibble = 0x40
)

// unexported variables.
var (
	errNoRequest = errors.New("the session never asked the model")
	errNoToken   = errors.New("no credentials")
	tokenPattern = regexp.MustCompile(`^\s*(?:export\s+)?` + tokenVariable + `=(.*?)\s*$`)
)

// conversation is one running `claude` process that takes its prompts on standard input as stream-json messages.
type conversation struct {
	session  *session
	command  *exec.Cmd
	stdin    io.WriteCloser
	cancel   context.CancelFunc
	results  chan string // the line of each result the process prints, one for each prompt it has answered
	ended    sync.Once
	turnEnds time.Time // when the process printed the result of the last prompt it answered: the turn ended
}

// end closes the process's input, which ends the session, and waits for the process; stderr is logged when it fails.
func (c *conversation) end(stderr string) {
	c.ended.Do(func() {
		_ = c.stdin.Close()

		waitErr := c.command.Wait()
		c.cancel()

		if waitErr != nil {
			c.session.machine.t.Logf("claude ended with %v\n%s", waitErr, stderr)
		}
	})
}

// readResults sends the line of each result event that the process prints to the results channel.
func (c *conversation) readResults(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(nil, maxLineBytes)

	for scanner.Scan() {
		var event struct {
			Type string `json:"type"`
		}

		if json.Unmarshal(scanner.Bytes(), &event) == nil && event.Type == "result" {
			c.results <- scanner.Text()
		}
	}

	close(c.results)
}

// say sends one prompt to the running session and returns what the session's record holds of it once the model has
// answered: the record is read while the process runs, as the next prompt's hooks read it.
func (c *conversation) say(prompt string) *turn {
	t := c.session.machine.t
	t.Helper()

	message, marshalErr := json.Marshal(map[string]any{
		"type": "user", "message": map[string]string{"role": "user", "content": prompt},
	})
	if marshalErr != nil {
		t.Fatalf("encoding the prompt: %v", marshalErr)
	}

	_, writeErr := c.stdin.Write(append(message, '\n'))
	if writeErr != nil {
		t.Fatalf("sending the prompt: %v", writeErr)
	}

	select {
	case result, open := <-c.results:
		if !open {
			t.Fatalf("claude ended before it answered %q", prompt)
		}

		c.turnEnds = time.Now()

		return c.session.answeredTurn(prompt, result)
	case <-time.After(sessionTimeout):
		t.Fatalf("claude did not answer %q within %s", prompt, sessionTimeout)

		return nil
	}
}

// startSession starts a session in a project directory of its own, outside any repository.
func (m *machine) startSession(project string) *session {
	m.t.Helper()

	dir := filepath.Join(m.root, "projects", project)
	mkdirAll(m.t, dir)

	return &session{machine: m, id: newUUID(m.t), project: dir}
}

// session is a Claude Code session in one project of a machine. Its prompts are separate `claude -p` runs that
// resume the session, so each prompt loads the machine's mods afresh, as a user's next prompt would after a restart.
type session struct {
	machine *machine
	id      string
	project string
	prompts int
}

// agentRan reports whether the agent ran a shell command in the session whose text holds a word.
func (s *session) agentRan(word string) bool {
	for _, row := range s.rows() {
		if slices.ContainsFunc(row.commands(), func(command string) bool { return strings.Contains(command, word) }) {
			return true
		}
	}

	return false
}

// answered reports whether the session's record holds an answer of the agent after the last occurrence of a prompt.
func (s *session) answered(prompt string) bool {
	rows := s.rows()
	start := -1

	for index, row := range rows {
		if text, isPrompt := row.promptText(); isPrompt && text == prompt {
			start = index
		}
	}

	if start < 0 {
		return false
	}

	return slices.ContainsFunc(rows[start+1:], func(row transcriptRow) bool { return row.Type == "assistant" })
}

// answeredTurn waits until the session's record shows the model's answer to a prompt, then cuts the turn out of it.
func (s *session) answeredTurn(prompt, result string) *turn {
	s.machine.t.Helper()

	deadline := time.Now().Add(recordFlushWait)

	for !s.answered(prompt) {
		if time.Now().After(deadline) {
			s.machine.t.Fatalf("the session's record shows no answer to %q within %s", prompt, recordFlushWait)
		}

		time.Sleep(pollInterval)
	}

	return s.turnOf(prompt, result)
}

// blocksNaming counts the blocks automatic memory delivered in the whole session that name a note.
func (s *session) blocksNaming(note string) int {
	count := 0

	for _, row := range s.rows() {
		for _, entry := range row.delivered() {
			if strings.HasPrefix(entry, engramMarker) && strings.Contains(entry, note) {
				count++
			}
		}
	}

	return count
}

// converse starts one `claude` process that takes its prompts as they are sent, on standard input: the session stays
// open between the prompts, as a user's does, and its record is written while the process runs.
func (s *session) converse() *conversation {
	s.machine.t.Helper()

	requireClaude(s.machine.t)

	token := claudeToken(s.machine.t)
	args := []string{
		"-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose", "--model", liveModel,
		"--max-budget-usd", liveBudgetUSD, "--permission-mode", "default", "--permission-prompts", "none",
		"--debug-file", filepath.Join(s.machine.root, fmt.Sprintf("claude-debug-%d.log", s.prompts)),
		"--session-id", s.id,
	}

	s.prompts++

	env := s.machine.env(tokenVariable + "=" + token)
	s.machine.assertIsolated(s.project, env)

	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	command := exec.CommandContext(ctx, lookPath("claude"), args...)
	command.Dir = s.project
	command.Env = env

	stdin, stdinErr := command.StdinPipe()
	stdout, stdoutErr := command.StdoutPipe()

	if stdinErr != nil || stdoutErr != nil {
		s.machine.t.Fatalf("connecting to claude: %v %v", stdinErr, stdoutErr)
	}

	var stderr bytes.Buffer

	command.Stderr = &stderr

	startErr := command.Start()
	if startErr != nil {
		s.machine.t.Fatalf("starting claude: %v", startErr)
	}

	chat := &conversation{
		session: s, command: command, stdin: stdin, cancel: cancel, results: make(chan string, resultsBuffered),
	}

	go chat.readResults(stdout)

	s.machine.t.Cleanup(func() { chat.end(redact(stderr.String(), token)) })

	return chat
}

// endOnceAsked waits until the session's record shows one more request to the model than before, then ends the run.
func (s *session) endOnceAsked(command *exec.Cmd, done chan error, before int) {
	s.machine.t.Helper()

	deadline := time.After(offlineWait)

	for asked := false; !asked; {
		select {
		case <-done:
			asked = true
		case <-deadline:
			s.machine.t.Fatalf("%v within %s", errNoRequest, offlineWait)
		case <-time.After(pollInterval):
			asked = s.requests() > before
		}
	}

	_ = command.Process.Signal(syscall.SIGTERM)

	select {
	case <-done:
	case <-time.After(stopGrace):
		_ = command.Process.Kill()
		<-done
	}
}

// requests counts the requests to the model that the session's record holds.
func (s *session) requests() int {
	count := 0

	for _, row := range s.rows() {
		if row.Type == "api-request" {
			count++
		}
	}

	return count
}

// rows reads the session's record, which is empty until Claude Code has written it.
func (s *session) rows() []transcriptRow {
	matches, err := filepath.Glob(filepath.Join(s.machine.transcripts, "*", s.id+".jsonl"))
	if err != nil || len(matches) == 0 {
		return nil
	}

	data, readErr := os.ReadFile(matches[0])
	if readErr != nil {
		return nil
	}

	var rows []transcriptRow

	for line := range bytes.SplitSeq(data, []byte("\n")) {
		var row transcriptRow

		if json.Unmarshal(line, &row) == nil {
			rows = append(rows, row)
		}
	}

	return rows
}

// run starts `claude -p` for one prompt and waits for it to finish, or for the model to have been asked when offline.
func (s *session) run(prompt string, credentials []string, secret string, offline bool) *turn {
	s.machine.t.Helper()

	requireClaude(s.machine.t)

	args := []string{
		"-p", prompt, "--output-format", "stream-json", "--verbose", "--model", liveModel,
		"--max-budget-usd", liveBudgetUSD, "--permission-mode", "default", "--permission-prompts", "none",
		"--debug-file", filepath.Join(s.machine.root, fmt.Sprintf("claude-debug-%d.log", s.prompts)),
	}

	if s.prompts == 0 {
		args = append(args, "--session-id", s.id)
	} else {
		args = append(args, "--resume", s.id)
	}

	s.prompts++

	env := s.machine.env(credentials...)
	s.machine.assertIsolated(s.project, env)

	ctx, cancel := context.WithTimeout(context.Background(), sessionTimeout)
	defer cancel()

	command := exec.CommandContext(ctx, lookPath("claude"), args...)
	command.Dir = s.project
	command.Env = env

	var stdout, stderr bytes.Buffer

	command.Stdout = &stdout
	command.Stderr = &stderr

	before := s.requests()

	startErr := command.Start()
	if startErr != nil {
		s.machine.t.Fatalf("starting claude: %v", startErr)
	}

	done := make(chan error, 1)

	go func() { done <- command.Wait() }()

	if offline {
		s.endOnceAsked(command, done, before)
	} else if waitErr := <-done; waitErr != nil {
		s.machine.t.Fatalf("claude failed: %v\n%s", waitErr, redact(stderr.String(), secret))
	}

	return s.turnOf(prompt, stdout.String())
}

// send sends one prompt to the live model and returns what the session's record holds of it.
func (s *session) send(prompt string) *turn {
	s.machine.t.Helper()

	token := claudeToken(s.machine.t)

	return s.run(prompt, []string{tokenVariable + "=" + token}, token, false)
}

// sendOffline sends the prompt promptHello to an endpoint that is not there: the prompt's hooks run, the model is asked
// and never answers, and the run is ended once the session's record shows the request. It needs no login and costs
// nothing.
func (s *session) sendOffline() *turn {
	s.machine.t.Helper()

	endpoint := []string{"ANTHROPIC_API_KEY=" + fakeAPIKey, "ANTHROPIC_BASE_URL=" + offlineBaseURL}

	return s.run(promptHello, endpoint, "", true)
}

// turnOf cuts the last occurrence of a prompt, and what followed it, out of the session's record.
func (s *session) turnOf(prompt, stdout string) *turn {
	s.machine.t.Helper()

	rows := s.rows()
	start := -1

	for index, row := range rows {
		if text, isPrompt := row.promptText(); isPrompt && text == prompt {
			start = index
		}
	}

	if start < 0 {
		s.machine.t.Fatalf("the session's record holds no prompt %q (%d rows)", prompt, len(rows))
	}

	end := len(rows)

	for index := start + 1; index < len(rows); index++ {
		if _, isPrompt := rows[index].promptText(); isPrompt {
			end = index

			break
		}
	}

	found := &turn{rows: rows[start:end]}
	found.reply, found.cost = resultOf(stdout)

	return found
}

// transcriptRow is one line of the session's record, with the fields the tests read.
type transcriptRow struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Message   struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
	Attachment struct {
		Type    string          `json:"type"`
		Content json.RawMessage `json:"content"`
	} `json:"attachment"`
}

// commands returns the shell commands the agent started in the row.
func (r transcriptRow) commands() []string {
	var blocks []struct {
		Type  string `json:"type"`
		Name  string `json:"name"`
		Input struct {
			Command string `json:"command"`
		} `json:"input"`
	}

	if r.Type != "assistant" || json.Unmarshal(r.Message.Content, &blocks) != nil {
		return nil
	}

	var commands []string

	for _, block := range blocks {
		if block.Type == "tool_use" && block.Name == "Bash" {
			commands = append(commands, block.Input.Command)
		}
	}

	return commands
}

// delivered returns the context a hook put beside the prompt.
func (r transcriptRow) delivered() []string {
	if r.Type != "attachment" || r.Attachment.Type != "hook_additional_context" {
		return nil
	}

	var content []string

	if json.Unmarshal(r.Attachment.Content, &content) != nil {
		return nil
	}

	return content
}

// promptText returns the text of a row that holds a prompt the user typed.
func (r transcriptRow) promptText() (string, bool) {
	if r.Type != "user" {
		return "", false
	}

	var text string

	if json.Unmarshal(r.Message.Content, &text) != nil {
		return "", false
	}

	return text, true
}

// turn is what one prompt left in the session's record.
type turn struct {
	rows  []transcriptRow // from the prompt's own row to the row before the next prompt's
	reply string          // the agent's final text
	cost  float64         // dollars
}

// agentActsAt returns the index, in the turn's rows, of the first row that shows the agent acting (a reply or a
// request to the model), or -1.
func (t *turn) agentActsAt() int {
	for index, row := range t.rows {
		if row.Type == "assistant" || row.Type == "api-request" {
			return index
		}
	}

	return -1
}

// blockAt returns the index, in the turn's rows, of the row that delivered the first block, or -1.
func (t *turn) blockAt() int {
	for index, row := range t.rows {
		for _, entry := range row.delivered() {
			if strings.HasPrefix(entry, engramMarker) {
				return index
			}
		}
	}

	return -1
}

// blocks returns the blocks automatic memory delivered with the prompt, in order.
func (t *turn) blocks() []string {
	var blocks []string

	for _, row := range t.rows {
		for _, entry := range row.delivered() {
			if strings.HasPrefix(entry, engramMarker) {
				blocks = append(blocks, entry)
			}
		}
	}

	return blocks
}

// contexts returns every entry any hook put beside the prompt, whoever wrote it.
func (t *turn) contexts() []string {
	var entries []string

	for _, row := range t.rows {
		entries = append(entries, row.delivered()...)
	}

	return entries
}

// hold returns how long the prompt waited before the model was asked: from the prompt's own row to the first request
// to the model, or -1 when the model was not asked.
func (t *turn) hold() time.Duration {
	started, err := time.Parse(time.RFC3339Nano, t.rows[0].Timestamp)
	if err != nil {
		return -1
	}

	for _, row := range t.rows {
		asked, parseErr := time.Parse(time.RFC3339Nano, row.Timestamp)
		if row.Type == "api-request" && parseErr == nil {
			return asked.Sub(started)
		}
	}

	return -1
}

// toolResultsNaming reports whether a result of a tool the agent ran in this turn names a note: the agent looked it up.
func (t *turn) toolResultsNaming(note string) bool {
	for _, row := range t.rows {
		var blocks []struct {
			Type    string          `json:"type"`
			Content json.RawMessage `json:"content"`
		}

		if row.Type == "user" && json.Unmarshal(row.Message.Content, &blocks) == nil {
			for _, block := range blocks {
				if block.Type == "tool_result" && strings.Contains(string(block.Content), note) {
					return true
				}
			}
		}
	}

	return false
}

// claudeToken returns the login the live tests use, from the environment or the owner's credential file, and
// fails the test when there is none (793b53). It is handed to the child's environment only, never to a command line
// or a log.
func claudeToken(t *testing.T) string {
	t.Helper()

	if token := os.Getenv(tokenVariable); token != "" {
		return token
	}

	home, err := os.UserHomeDir()
	if err == nil {
		data, readErr := os.ReadFile(filepath.Join(home, credentialFile))
		if readErr == nil {
			for line := range strings.SplitSeq(string(data), "\n") {
				match := tokenPattern.FindStringSubmatch(line)
				if match != nil {
					return strings.Trim(match[1], `"'`)
				}
			}
		}
	}

	t.Fatalf("%v: set %s, or put it in ~/%s", errNoToken, tokenVariable, credentialFile)

	return ""
}

// newUUID returns a random version 4 UUID, the form `claude --session-id` takes.
func newUUID(t *testing.T) string {
	t.Helper()

	var raw [uuidBytes]byte

	_, err := rand.Read(raw[:])
	if err != nil {
		t.Fatalf("reading random bytes: %v", err)
	}

	raw[uuidVersionByte] = raw[uuidVersionByte]&0x0f | uuidVersionNibble
	raw[uuidVariantByte] = raw[uuidVariantByte]&0x3f | uuidVariantBits

	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:])
}

// redact removes a secret from text that is about to be logged.
func redact(text, secret string) string {
	if secret == "" {
		return text
	}

	return strings.ReplaceAll(text, secret, "[redacted]")
}

// requireClaude fails the test when Claude Code is not installed (793b53).
func requireClaude(t *testing.T) {
	t.Helper()

	_, err := exec.LookPath("claude")
	if err != nil {
		t.Fatal("claude is not on PATH")
	}
}

// resultOf reads the final text and the cost of a stream-json run.
func resultOf(stdout string) (string, float64) {
	var (
		reply string
		cost  float64
	)

	for line := range strings.SplitSeq(stdout, "\n") {
		var message map[string]any

		if json.Unmarshal([]byte(line), &message) != nil || message["type"] != "result" {
			continue
		}

		reply, _ = message["result"].(string)
		cost, _ = message["total_cost_usd"].(float64)
	}

	return reply, cost
}
