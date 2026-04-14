# Stage 1b: Engram-Agent Lifecycle — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add engram-agent lifecycle management to the API server — invoking `claude -p --resume`, parsing structured JSON responses, routing results to the chat file, and recovering from errors with a structured escalation ladder.

**Architecture:** A new `EngramAgent` type in `internal/server` manages the claude -p session. It owns the session ID, invocation counter (for skill refresh), and the error/recovery state machine. A new stream parser extracts structured JSON (`{"action": "surface", ...}`) from the stream-json JSONL envelope. The `EngramAgent` integrates with the existing `AgentLoop` — when the engram-agent goroutine receives a message, it invokes `EngramAgent.Process()` which runs `claude -p` and routes the response. All I/O (process execution, chat posting) is injected via interfaces.

**Tech Stack:** Go stdlib `os/exec`, `encoding/json`, `bufio`, `internal/server` (from 1a), `internal/chat`, `pgregory.net/rapid`, `imptest`, gomega.

**Principles:** Read `docs/exec-planning.md`. Context flows from top. DI for all I/O. Property-based tests. Full TDD cycle. imptest for interactive mocks.

---

## File Structure

```
internal/server/
  stream.go        — Stream parser: reads stream-json JSONL, extracts structured JSON response
  stream_test.go   — Stream parser tests
  engram.go        — EngramAgent: session lifecycle, invocation, error recovery
  engram_test.go   — EngramAgent tests
  (existing files from 1a: fanout.go, agent.go, handlers.go, server.go, validate.go, refresh.go)
```

---

### Task 1: Stream parser — extract structured JSON from stream-json JSONL

The stream-json output from `claude -p` is a series of JSONL lines. Each line is a JSON object with a `type` field. We need to extract `assistant` text events and parse the inner structured JSON.

**Files:**
- Create: `internal/server/stream.go`
- Create: `internal/server/stream_test.go`

- [ ] **Step 1: Write failing test — parses surface action from stream-json**

```go
package server_test

import (
	"strings"
	"testing"

	"engram/internal/server"

	. "github.com/onsi/gomega"
)

func TestParseStreamResponse_SurfaceAction(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Simulated stream-json output from claude -p.
	input := strings.Join([]string{
		`{"type":"system","session_id":"sess-123"}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"{\"action\":\"surface\",\"to\":\"lead-1\",\"text\":\"Memory: use DI\"}"}]}}`,
	}, "\n")

	result, parseErr := server.ParseStreamResponse(strings.NewReader(input))
	g.Expect(parseErr).NotTo(HaveOccurred())
	if parseErr != nil {
		return
	}

	g.Expect(result.SessionID).To(Equal("sess-123"))
	g.Expect(result.Action).To(Equal("surface"))
	g.Expect(result.To).To(Equal("lead-1"))
	g.Expect(result.Text).To(Equal("Memory: use DI"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — ParseStreamResponse not defined

- [ ] **Step 3: Write minimal implementation**

```go
package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// AgentResponse is the parsed structured output from the engram-agent.
type AgentResponse struct {
	SessionID string `json:"-"`          // From the system event, not the agent's JSON.
	Action    string `json:"action"`     // "surface", "log-only", or "learn"
	To        string `json:"to"`         // Recipient for surface/learn actions.
	Text      string `json:"text"`       // Content (surfaced memory or learn outcome).
	Saved     bool   `json:"saved"`      // For learn action: whether memory was persisted.
	Path      string `json:"path"`       // For learn action: file path of saved memory.
}

var (
	// ErrNoAssistantText is returned when no assistant text is found in the stream.
	ErrNoAssistantText = errors.New("stream: no assistant text found")
	// ErrMalformedAction is returned when the assistant text is not valid structured JSON.
	ErrMalformedAction = errors.New("stream: malformed action JSON in assistant text")
)

// ParseStreamResponse reads stream-json JSONL from an io.Reader (the stdout of
// claude -p --output-format=stream-json) and extracts the session ID and
// structured JSON response from the assistant text.
func ParseStreamResponse(reader io.Reader) (AgentResponse, error) {
	scanner := bufio.NewScanner(reader)

	var sessionID string
	var assistantText string

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
			Message   *struct {
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		}

		if jsonErr := json.Unmarshal(line, &event); jsonErr != nil {
			continue // Skip malformed lines.
		}

		if event.SessionID != "" {
			sessionID = event.SessionID
		}

		if event.Type == "assistant" && event.Message != nil {
			for _, block := range event.Message.Content {
				if block.Type == "text" && block.Text != "" {
					assistantText = block.Text
				}
			}
		}
	}

	if assistantText == "" {
		return AgentResponse{}, ErrNoAssistantText
	}

	var resp AgentResponse
	if jsonErr := json.Unmarshal([]byte(assistantText), &resp); jsonErr != nil {
		return AgentResponse{}, fmt.Errorf("%w: %w", ErrMalformedAction, jsonErr)
	}

	resp.SessionID = sessionID

	return resp, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Write additional tests**

- `TestParseStreamResponse_LogOnlyAction` — parses log-only response
- `TestParseStreamResponse_LearnAction` — parses learn response with saved/path
- `TestParseStreamResponse_NoAssistantText_ReturnsError` — only system events, no assistant
- `TestParseStreamResponse_MalformedJSON_ReturnsError` — assistant text is not JSON
- Property test: `TestParseStreamResponse_AlwaysExtractsSessionID` — any session ID string is captured

- [ ] **Step 6: Run all tests, verify pass**

- [ ] **Step 7: Refactor — review for DRY, SOLID**

The stream parser is self-contained. No external dependencies (just `io.Reader`). Clean DI boundary.

- [ ] **Step 8: Commit**

```bash
git add internal/server/stream.go internal/server/stream_test.go
git commit -m "feat(server): add stream-json parser for engram-agent output

AI-Used: [claude]"
```

---

### Task 2: EngramAgent — session lifecycle, invocation, structured routing

The core type that manages the engram-agent's claude -p session. It tracks the session ID, invocation count, and provides `Process(ctx, message) error` which invokes claude -p and routes the response.

**Files:**
- Create: `internal/server/engram.go`
- Create: `internal/server/engram_test.go`

- [ ] **Step 1: Write failing test — Process invokes claude and posts surface result to chat**

```go
package server_test

import (
	"context"
	"strings"
	"testing"

	"engram/internal/chat"
	"engram/internal/server"

	. "github.com/onsi/gomega"
)

func TestEngramAgent_Process_SurfaceAction_PostsToChat(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var postedMsg chat.Message

	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, prompt, sessionID string) (string, error) {
			// Return stream-json with a surface response.
			return strings.Join([]string{
				`{"type":"system","session_id":"sess-1"}`,
				`{"type":"assistant","message":{"content":[{"type":"text","text":"{\"action\":\"surface\",\"to\":\"lead-1\",\"text\":\"Memory: use DI\"}"}]}}`,
			}, "\n"), nil
		},
		PostToChat: func(msg chat.Message) (int, error) {
			postedMsg = msg
			return 1, nil
		},
		Logger: slog.Default(),
	})

	err := agent.Process(t.Context(), chat.Message{From: "lead-1", To: "engram-agent", Text: "testing"})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(postedMsg.To).To(Equal("lead-1"))
	g.Expect(postedMsg.Text).To(ContainSubstring("Memory: use DI"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — EngramAgent not defined

- [ ] **Step 3: Write minimal implementation**

```go
package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"engram/internal/chat"
)

// RunClaudeFunc invokes claude -p and returns the raw stdout as a string.
// prompt is the input text. sessionID is empty for first invocation, non-empty for --resume.
type RunClaudeFunc func(ctx context.Context, prompt, sessionID string) (string, error)

// EngramAgentConfig configures the engram-agent lifecycle manager.
type EngramAgentConfig struct {
	RunClaude  RunClaudeFunc
	PostToChat PostFunc
	Logger     *slog.Logger
}

// EngramAgent manages the engram-agent's claude -p session.
type EngramAgent struct {
	config     EngramAgentConfig
	sessionID  string
	invocations int
	refresh    *RefreshTracker
}

// NewEngramAgent creates an EngramAgent.
func NewEngramAgent(config EngramAgentConfig) *EngramAgent {
	return &EngramAgent{
		config:  config,
		refresh: NewRefreshTracker(SkillRefreshInterval),
	}
}

// Process invokes the engram-agent with the given message and routes the response.
func (e *EngramAgent) Process(ctx context.Context, msg chat.Message) error {
	e.invocations++

	prompt := msg.Text
	if e.refresh.ShouldRefresh() {
		prompt = "SKILL REFRESH: Reload /use-engram-chat-as and /engram-agent.\n\n" + prompt
		e.config.Logger.Info("skill refresh triggered", "turn", e.invocations)
	}

	output, runErr := e.config.RunClaude(ctx, prompt, e.sessionID)
	if runErr != nil {
		return fmt.Errorf("engram-agent: invoking claude: %w", runErr)
	}

	resp, parseErr := ParseStreamResponse(strings.NewReader(output))
	if parseErr != nil {
		return fmt.Errorf("engram-agent: parsing response: %w", parseErr)
	}

	// Capture session ID from first invocation.
	if resp.SessionID != "" {
		e.sessionID = resp.SessionID
	}

	return e.routeResponse(resp)
}

func (e *EngramAgent) routeResponse(resp AgentResponse) error {
	switch resp.Action {
	case "surface":
		_, postErr := e.config.PostToChat(chat.Message{
			From: "engram-agent", To: resp.To, Text: resp.Text,
		})
		if postErr != nil {
			return fmt.Errorf("engram-agent: posting surface: %w", postErr)
		}

		e.config.Logger.Info("engram-agent responded",
			"action", "surface", "to", resp.To, "text_len", len(resp.Text))
	case "log-only":
		_, postErr := e.config.PostToChat(chat.Message{
			From: "engram-agent", To: "log", Text: resp.Text,
		})
		if postErr != nil {
			return fmt.Errorf("engram-agent: posting log: %w", postErr)
		}

		e.config.Logger.Info("engram-agent responded", "action", "log-only")
	case "learn":
		_, postErr := e.config.PostToChat(chat.Message{
			From: "engram-agent", To: resp.To, Text: resp.Text,
		})
		if postErr != nil {
			return fmt.Errorf("engram-agent: posting learn: %w", postErr)
		}

		e.config.Logger.Info("engram-agent responded",
			"action", "learn", "saved", resp.Saved, "path", resp.Path)
	default:
		e.config.Logger.Warn("engram-agent: unknown action", "action", resp.Action)
	}

	return nil
}

// SessionID returns the current session ID (empty if not yet invoked).
func (e *EngramAgent) SessionID() string { return e.sessionID }

// ResetSession clears the session ID, forcing a fresh session on next invocation.
func (e *EngramAgent) ResetSession() {
	e.config.Logger.Info("engram-agent session reset", "old_session_id", e.sessionID)
	e.sessionID = ""
}
```

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Write additional tests**

- `TestEngramAgent_Process_LogOnlyAction_PostsWithSentinelTo` — verify to="log"
- `TestEngramAgent_Process_LearnAction_PostsToOriginator` — verify to field from response
- `TestEngramAgent_Process_CapturesSessionID` — verify SessionID() after first invocation
- `TestEngramAgent_Process_SkillRefreshEvery13Turns` — invoke 13 times, verify prompt includes refresh on 13th
- `TestEngramAgent_ResetSession_ClearsSessionID` — verify SessionID() is empty after reset

- [ ] **Step 6: Run all tests, verify pass**

- [ ] **Step 7: Refactor**

- [ ] **Step 8: Commit**

```bash
git add internal/server/engram.go internal/server/engram_test.go
git commit -m "feat(server): add EngramAgent lifecycle with session tracking and routing

AI-Used: [claude]"
```

---

### Task 3: Error recovery protocol — retry → reset → retry → escalate

The unified error/recovery ladder: 3 retries on current session → session reset → 3 retries on fresh session → escalate.

**Files:**
- Modify: `internal/server/engram.go`
- Modify: `internal/server/engram_test.go`

- [ ] **Step 1: Write failing test — retries on malformed output then succeeds**

```go
func TestEngramAgent_ProcessWithRecovery_RetriesOnMalformedThenSucceeds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	callCount := 0
	agent := server.NewEngramAgent(server.EngramAgentConfig{
		RunClaude: func(_ context.Context, _, _ string) (string, error) {
			callCount++
			if callCount <= 2 {
				// First 2 calls return malformed output.
				return `{"type":"assistant","message":{"content":[{"type":"text","text":"not json at all"}]}}`, nil
			}
			// Third call succeeds.
			return validSurfaceOutput("Memory found"), nil
		},
		PostToChat: func(_ chat.Message) (int, error) { return 1, nil },
		Logger:     slog.Default(),
	})

	err := agent.ProcessWithRecovery(t.Context(), chat.Message{Text: "test"})
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(callCount).To(Equal(3)) // 2 failures + 1 success
}
```

Define helper:
```go
func validSurfaceOutput(text string) string {
	return strings.Join([]string{
		`{"type":"system","session_id":"sess-1"}`,
		`{"type":"assistant","message":{"content":[{"type":"text","text":"{\"action\":\"surface\",\"to\":\"lead-1\",\"text\":\"` + text + `\"}"}]}}`,
	}, "\n")
}
```

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Implement ProcessWithRecovery**

```go
const (
	maxRetriesPerSession = 3
	maxSessionResets     = 1
)

// ProcessWithRecovery invokes Process with the unified error recovery protocol.
// Retries on same session (3x) → reset → retries on fresh session (3x) → escalate.
func (e *EngramAgent) ProcessWithRecovery(ctx context.Context, msg chat.Message) error {
	// Track recent messages for session reset context.
	e.trackMessage(msg)

	for sessionAttempt := range maxSessionResets + 1 {
		for retry := range maxRetriesPerSession {
			err := e.Process(ctx, msg)
			if err == nil {
				return nil
			}

			e.config.Logger.Warn("engram-agent error",
				"err", err,
				"retry", retry+1,
				"session_attempt", sessionAttempt,
				"session_id", e.sessionID,
			)
		}

		// Exhausted retries on this session. Reset if we haven't already.
		if sessionAttempt < maxSessionResets {
			e.ResetSession()
			e.reinjectContext(ctx)
		}
	}

	// Escalate.
	return e.escalate(msg)
}

func (e *EngramAgent) escalate(msg chat.Message) error {
	errMsg := fmt.Sprintf(
		"engram-agent cannot produce valid output after %d attempts. "+
			"The skill/server contract needs manual intervention. Last input: %s",
		(maxSessionResets+1)*maxRetriesPerSession, msg.Text,
	)

	_, postErr := e.config.PostToChat(chat.Message{
		From: "engram-server", To: "lead", Text: errMsg,
	})
	if postErr != nil {
		e.config.Logger.Error("failed to post escalation", "err", postErr)
	}

	e.config.Logger.Error("engram-agent escalation", "msg", errMsg)

	return fmt.Errorf("engram-agent: %s", errMsg)
}
```

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Write additional tests**

- `TestEngramAgent_ProcessWithRecovery_ExhaustsRetriesThenResetsSession` — verify session is reset after 3 failures
- `TestEngramAgent_ProcessWithRecovery_EscalatesAfterAllAttempts` — verify escalation posted to chat
- `TestEngramAgent_ProcessWithRecovery_ExecutionErrorTriggersRetry` — RunClaude returns error, retried

- [ ] **Step 6: Run all tests, verify pass**

- [ ] **Step 7: Refactor**

- [ ] **Step 8: Commit**

```bash
git add internal/server/engram.go internal/server/engram_test.go
git commit -m "feat(server): add ProcessWithRecovery error escalation ladder

AI-Used: [claude]"
```

---

### Task 4: POST /reset-agent endpoint + integration

Wire the EngramAgent into the server. Add the POST /reset-agent endpoint. Connect the engram-agent goroutine (from AgentLoop) to invoke EngramAgent.ProcessWithRecovery when messages arrive.

**Files:**
- Modify: `internal/server/handlers.go` (add HandleResetAgent)
- Modify: `internal/server/handlers_test.go`
- Modify: `internal/server/server.go` (add route, add EngramAgent to Config)

- [ ] **Step 1: Write failing test — POST /reset-agent resets session**

```go
func TestResetAgent_ResetsEngramAgentSession(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	resetCalled := false
	deps := &server.Deps{
		Logger:     slog.Default(),
		ResetAgent: func() { resetCalled = true },
	}

	req := httptest.NewRequest(http.MethodPost, "/reset-agent", nil)
	rec := httptest.NewRecorder()

	server.HandleResetAgent(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(resetCalled).To(BeTrue())
}
```

- [ ] **Step 2: Implement HandleResetAgent**

Add `ResetAgent func()` to `Deps`. Add handler:

```go
// HandleResetAgent returns an http.HandlerFunc for POST /reset-agent.
func HandleResetAgent(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if deps.ResetAgent != nil {
			deps.ResetAgent()
		}

		deps.Logger.Info("engram-agent session reset via API")
		writeJSON(w, http.StatusOK, map[string]string{"status": "reset"})
	}
}
```

- [ ] **Step 3: Add route to server.go and EngramAgent to Config**

In server.go, add the route:
```go
mux.HandleFunc("POST /reset-agent", HandleResetAgent(deps))
```

Add to Config:
```go
ResetAgentFunc func() // Called by POST /reset-agent
```

Wire in Start:
```go
deps.ResetAgent = cfg.ResetAgentFunc
```

- [ ] **Step 4: Run all tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add internal/server/handlers.go internal/server/handlers_test.go internal/server/server.go
git commit -m "feat(server): add POST /reset-agent endpoint

AI-Used: [claude]"
```

---

### Task 5: Wire EngramAgent into server startup (cli_server.go)

Connect the EngramAgent to the server in the CLI wiring layer. When the server starts, create an EngramAgent with a real RunClaude function (builds exec.Cmd). Wire its ProcessWithRecovery as the OnMessage callback for the engram-agent AgentLoop.

**Files:**
- Modify: `internal/cli/cli_server.go`
- Modify: `internal/cli/cli_server_test.go`

- [ ] **Step 1: Add RunClaude wiring to buildServerConfig**

The real RunClaude function builds an exec.Cmd following the same pattern as `buildClaudeCmd` in cli_agent.go:

```go
func buildRunClaude(claudeBinary string) server.RunClaudeFunc {
	return func(ctx context.Context, prompt, sessionID string) (string, error) {
		args := []string{"-p",
			"--dangerously-skip-permissions",
			"--verbose",
			"--output-format=stream-json",
		}
		if sessionID != "" {
			args = append(args, "--resume", sessionID)
		}
		args = append(args, prompt)

		cmd := exec.CommandContext(ctx, claudeBinary, args...)
		output, runErr := cmd.Output()
		if runErr != nil {
			return "", fmt.Errorf("running claude: %w", runErr)
		}

		return string(output), nil
	}
}
```

- [ ] **Step 2: Wire EngramAgent into server config**

In `buildServerConfig`, create the EngramAgent and wire its ProcessWithRecovery as a callback.

- [ ] **Step 3: Test with a mock (unit test verifying the wiring exists)**

This is thin wiring — the main test is that the server starts. The EngramAgent's behavior is already tested in Tasks 1-3.

- [ ] **Step 4: Run all tests, verify pass**

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli_server.go internal/cli/cli_server_test.go
git commit -m "feat(cli): wire EngramAgent into server startup

AI-Used: [claude]"
```

---

### Task 6: Quality check + e2e testing

- [ ] **Step 1: Run full test suite**

Run: `targ test`
Expected: All pass

- [ ] **Step 2: Run full quality check**

Run: `targ check-full`
Expected: All pass (except check-uncommitted during work)

- [ ] **Step 3: Fix any issues**

- [ ] **Step 4: E2E test**

Build binary. Start server. POST a message. Verify the engram-agent goroutine would be triggered (can't fully test without a real claude binary, but verify the server starts, accepts messages, and the wiring is connected).

```bash
go build -o /tmp/engram-1b ./cmd/engram/
/tmp/engram-1b server up --chat-file /tmp/e2e-1b.toml --addr localhost:19878 &
sleep 1
curl -s http://localhost:19878/status | jq .
curl -s -X POST http://localhost:19878/message -d '{"from":"user","to":"engram-agent","text":"test"}'
curl -s -X POST http://localhost:19878/reset-agent | jq .
curl -s -X POST http://localhost:19878/shutdown
```

- [ ] **Step 5: Commit fixes**

```bash
git add -A
git commit -m "fix: address quality and e2e issues from stage 1b

AI-Used: [claude]"
```
