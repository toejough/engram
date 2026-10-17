//go:build network

package acceptance_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// unexported constants.
const (
	credentialFile    = ".config/claude-key.env"
	engramMarker      = "[engram memory]"
	liveBudgetUSD     = "2"
	liveModel         = "sonnet"
	sessionTimeout    = 5 * time.Minute
	tokenVariable     = "CLAUDE_CODE_OAUTH_TOKEN"
	uuidBytes         = 16
	uuidVariantBits   = 0x80
	uuidVariantByte   = 8
	uuidVersionByte   = 6
	uuidVersionNibble = 0x40
)

// unexported variables.
var (
	errNoToken   = errors.New("no credentials")
	tokenPattern = regexp.MustCompile(`^\s*(?:export\s+)?` + tokenVariable + `=(.*?)\s*$`)
)

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

// run starts `claude -p` for one prompt and waits for it to finish.
func (s *session) run(prompt string, credentials []string, secret string) *turn {
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

	err := command.Run()
	if err != nil {
		s.machine.t.Fatalf("claude failed: %v\n%s", err, redact(stderr.String(), secret))
	}

	return s.turnOf(prompt, stdout.String())
}

// send sends one prompt to the live model and returns what the session's record holds of it.
func (s *session) send(prompt string) *turn {
	s.machine.t.Helper()

	token := claudeToken(s.machine.t)

	return s.run(prompt, []string{tokenVariable + "=" + token}, token)
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
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
	Attachment struct {
		Type     string          `json:"type"`
		Content  json.RawMessage `json:"content"`
		HookName string          `json:"hookName"`
	} `json:"attachment"`
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

// claudeToken returns the login the live tests use, from the environment or the owner's credential file, and
// skips the test when there is none. It is handed to the child's environment only, never to a command line or a log.
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

	t.Skipf("%v: set %s, or put it in ~/%s", errNoToken, tokenVariable, credentialFile)

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

// requireClaude skips the test when Claude Code is not installed.
func requireClaude(t *testing.T) {
	t.Helper()

	_, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("claude is not on PATH")
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
