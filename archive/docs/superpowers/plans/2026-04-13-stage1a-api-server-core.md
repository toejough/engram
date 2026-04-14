# Stage 1a: API Server Core — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the engram API server — an HTTP server that watches the TOML chat file, fans out notifications to per-agent goroutines, and serves the endpoints that CLI/MCP clients call.

**Architecture:** A new `internal/server` package provides the server with DI for all I/O. A `SharedWatcher` wraps a single fsnotify watcher and fans out change notifications to agent goroutines via buffered channels. Each agent goroutine maintains its own cursor and reads messages independently. HTTP handlers delegate to the goroutines. The server binds to localhost only. The `engram server up` command is the thin wiring layer.

**Tech Stack:** Go stdlib `net/http`, `log/slog`, `encoding/json`, `internal/chat` (FilePoster, FileWatcher, ParseMessagesSafe), `internal/watch` (fsnotify), `pgregory.net/rapid`, `imptest`, gomega.

**Principles:** Read `docs/exec-planning.md` before implementing. Context flows from top. DI for all I/O in `internal/`. Property-based tests with rapid. Full TDD cycle (red/green/refactor). Use `imptest` for interactive mocks. Use `newFlagSet` not `flag.NewFlagSet`. Validate required flags with sentinel errors.

---

## File Structure

```
internal/server/
  server.go        — Server type: HTTP setup, routing, graceful shutdown, slog config
  server_test.go   — Server integration tests
  handlers.go      — HTTP handlers: postMessage, waitForResponse, subscribe, status, shutdown
  handlers_test.go — Handler unit tests
  fanout.go        — SharedWatcher: single fsnotify → buffered channel fan-out
  fanout_test.go   — SharedWatcher tests
  agent.go         — AgentLoop: per-agent goroutine with cursor, message filtering
  agent_test.go    — AgentLoop tests

internal/cli/
  cli_server.go    — engram server up: thin wiring (flag parsing, real I/O construction)
```

---

### Task 1: SharedWatcher — fan-out from single fsnotify watcher

A single fsnotify watcher monitors the TOML chat file. When it changes, all registered subscribers are notified via buffered channels (buffer=1). If a subscriber is busy, the notification coalesces (doesn't block).

**Files:**
- Create: `internal/server/fanout.go`
- Create: `internal/server/fanout_test.go`

- [ ] **Step 1: Write failing property test — subscribers always notified on file change**

```go
package server_test

import (
	"context"
	"testing"
	"time"

	"engram/internal/server"

	. "github.com/onsi/gomega"
)

func TestSharedWatcher_AllSubscribersNotifiedOnChange(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notify := make(chan struct{}, 1)
	sw := server.NewSharedWatcher(func(_ context.Context, _ string) error {
		<-notify
		return nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch1 := sw.Subscribe()
	ch2 := sw.Subscribe()

	go sw.Run(ctx, "/fake/path")

	// Trigger a file change notification.
	notify <- struct{}{}

	// Both subscribers should be notified.
	g.Eventually(func() bool {
		select {
		case <-ch1:
			return true
		default:
			return false
		}
	}).WithTimeout(time.Second).Should(BeTrue())

	g.Eventually(func() bool {
		select {
		case <-ch2:
			return true
		default:
			return false
		}
	}).WithTimeout(time.Second).Should(BeTrue())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write minimal implementation**

```go
// Package server implements the engram API server.
package server

import "context"

// WaitFunc blocks until the watched file changes. Injected for testing.
// In production, wraps fsnotify. Signature matches watch.FSNotifyWatcher.WaitForChange.
type WaitFunc func(ctx context.Context, path string) error

// SharedWatcher watches a single file and fans out change notifications
// to all registered subscribers via buffered channels (buffer=1).
// If a subscriber's channel is full, the notification coalesces.
type SharedWatcher struct {
	waitForChange WaitFunc
	subscribers   []chan struct{}
}

// NewSharedWatcher creates a SharedWatcher with the given wait function.
func NewSharedWatcher(wait WaitFunc) *SharedWatcher {
	return &SharedWatcher{waitForChange: wait}
}

// Subscribe registers a new subscriber and returns its notification channel.
// Must be called before Run.
func (sw *SharedWatcher) Subscribe() <-chan struct{} {
	ch := make(chan struct{}, 1)
	sw.subscribers = append(sw.subscribers, ch)

	return ch
}

// Run blocks, watching the file and notifying subscribers on each change.
// Returns when ctx is cancelled or the wait function errors.
func (sw *SharedWatcher) Run(ctx context.Context, path string) error {
	for {
		err := sw.waitForChange(ctx, path)
		if err != nil {
			return err
		}

		for _, ch := range sw.subscribers {
			select {
			case ch <- struct{}{}:
			default: // coalesce — subscriber already has a pending notification
			}
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Write test — unsubscribe removes subscriber**

```go
func TestSharedWatcher_UnsubscribeRemovesSubscriber(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	notify := make(chan struct{}, 1)
	sw := server.NewSharedWatcher(func(_ context.Context, _ string) error {
		<-notify
		return nil
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	ch1 := sw.Subscribe()
	ch2 := sw.Subscribe()
	sw.Unsubscribe(ch1)

	go sw.Run(ctx, "/fake/path")

	notify <- struct{}{}

	// ch2 should be notified.
	g.Eventually(func() bool {
		select {
		case <-ch2:
			return true
		default:
			return false
		}
	}).WithTimeout(time.Second).Should(BeTrue())

	// ch1 should NOT be notified (unsubscribed).
	g.Consistently(func() bool {
		select {
		case <-ch1:
			return true
		default:
			return false
		}
	}).WithTimeout(100 * time.Millisecond).Should(BeFalse())
}
```

- [ ] **Step 6: Implement Unsubscribe**

```go
// Unsubscribe removes a subscriber. The channel is closed.
func (sw *SharedWatcher) Unsubscribe(ch <-chan struct{}) {
	for i, sub := range sw.subscribers {
		if sub == ch {
			close(sub)
			sw.subscribers = append(sw.subscribers[:i], sw.subscribers[i+1:]...)

			return
		}
	}
}
```

- [ ] **Step 7: Run tests, verify pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 8: Write test — context cancellation stops Run**

```go
func TestSharedWatcher_RunStopsOnContextCancel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	sw := server.NewSharedWatcher(func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	})

	ctx, cancel := context.WithCancel(t.Context())

	done := make(chan error, 1)
	go func() { done <- sw.Run(ctx, "/fake") }()

	cancel()

	g.Eventually(done).WithTimeout(time.Second).Should(Receive())
}
```

- [ ] **Step 9: Run tests, verify pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 10: Refactor — review for DRY, SOLID**

Check: SharedWatcher is focused (single responsibility: fan-out). Subscribe/Unsubscribe/Run are clean. The WaitFunc injection is the DI boundary.

Note: Subscribe is not thread-safe (must be called before Run). This is intentional — the server registers all goroutines before starting the watcher. Document this in the comment.

- [ ] **Step 11: Commit**

```bash
git add internal/server/fanout.go internal/server/fanout_test.go
git commit -m "feat(server): add SharedWatcher fan-out for chat file notifications

AI-Used: [claude]"
```

---

### Task 2: AgentLoop — per-agent goroutine with cursor and message filtering

Each agent gets a goroutine that reads new messages from the chat file when notified by SharedWatcher. It maintains its own cursor, filters messages by recipient (or reads all for the engram-agent), and delivers matches to a channel.

**Files:**
- Create: `internal/server/agent.go`
- Create: `internal/server/agent_test.go`

- [ ] **Step 1: Write failing property test — AgentLoop always delivers messages addressed to it**

```go
package server_test

import (
	"testing"
	"time"

	"engram/internal/chat"
	"engram/internal/server"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestAgentLoop_AlwaysDeliversMessagesAddressedToAgent(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		agentName := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "agent")
		text := rapid.StringMatching(`[A-Za-z0-9 ]{5,50}`).Draw(rt, "text")

		// Fake chat data with one message addressed to our agent.
		msg := chat.Message{From: "sender", To: agentName, Text: text}
		messages := []chat.Message{msg}

		notify := make(chan struct{}, 1)
		delivered := make(chan chat.Message, 1)

		loop := server.NewAgentLoop(server.AgentLoopConfig{
			Name:         agentName,
			WatchAll:     false,
			Notify:       notify,
			ReadMessages: func(_ int) ([]chat.Message, int, error) {
				return messages, 1, nil
			},
			OnMessage: func(m chat.Message) { delivered <- m },
		})

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()

		go loop.Run(ctx)

		// Trigger notification.
		notify <- struct{}{}

		// Should receive the message.
		g.Eventually(delivered).WithTimeout(time.Second).Should(Receive(Equal(msg)))
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — AgentLoop not defined

- [ ] **Step 3: Write minimal implementation**

```go
package server

import (
	"context"

	"engram/internal/chat"
)

// ReadMessagesFunc reads messages from the chat file starting at cursor.
// Returns the messages found and the new cursor position.
type ReadMessagesFunc func(cursor int) ([]chat.Message, int, error)

// AgentLoopConfig configures an agent goroutine.
type AgentLoopConfig struct {
	Name         string            // Agent name for recipient filtering.
	WatchAll     bool              // If true, delivers ALL messages (engram-agent).
	Notify       <-chan struct{}    // Notification channel from SharedWatcher.
	ReadMessages ReadMessagesFunc  // Reads messages from chat file at cursor.
	OnMessage    func(chat.Message) // Called for each matching message.
}

// AgentLoop is a per-agent goroutine that reads and filters chat messages.
type AgentLoop struct {
	config AgentLoopConfig
	cursor int
}

// NewAgentLoop creates an AgentLoop.
func NewAgentLoop(config AgentLoopConfig) *AgentLoop {
	return &AgentLoop{config: config}
}

// Run blocks, processing messages until ctx is cancelled.
func (a *AgentLoop) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.config.Notify:
			a.processNewMessages()
		}
	}
}

func (a *AgentLoop) processNewMessages() {
	messages, newCursor, err := a.config.ReadMessages(a.cursor)
	if err != nil {
		return // logged by caller; don't crash the goroutine
	}

	a.cursor = newCursor

	for _, msg := range messages {
		if a.config.WatchAll || chat.MatchesAgent(msg.To, a.config.Name) {
			a.config.OnMessage(msg)
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Write test — WatchAll delivers all messages regardless of recipient**

```go
func TestAgentLoop_WatchAllDeliversAllMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	msg1 := chat.Message{From: "a", To: "other", Text: "first"}
	msg2 := chat.Message{From: "b", To: "another", Text: "second"}
	messages := []chat.Message{msg1, msg2}

	notify := make(chan struct{}, 1)
	delivered := make(chan chat.Message, 10)

	loop := server.NewAgentLoop(server.AgentLoopConfig{
		Name:     "engram-agent",
		WatchAll: true,
		Notify:   notify,
		ReadMessages: func(_ int) ([]chat.Message, int, error) {
			return messages, 2, nil
		},
		OnMessage: func(m chat.Message) { delivered <- m },
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go loop.Run(ctx)
	notify <- struct{}{}

	g.Eventually(delivered).WithTimeout(time.Second).Should(Receive(Equal(msg1)))
	g.Eventually(delivered).WithTimeout(time.Second).Should(Receive(Equal(msg2)))
}
```

- [ ] **Step 6: Write test — non-matching messages are NOT delivered**

```go
func TestAgentLoop_NonMatchingMessagesNotDelivered(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	msg := chat.Message{From: "a", To: "other-agent", Text: "not for me"}

	notify := make(chan struct{}, 1)
	delivered := make(chan chat.Message, 1)

	loop := server.NewAgentLoop(server.AgentLoopConfig{
		Name:     "my-agent",
		WatchAll: false,
		Notify:   notify,
		ReadMessages: func(_ int) ([]chat.Message, int, error) {
			return []chat.Message{msg}, 1, nil
		},
		OnMessage: func(m chat.Message) { delivered <- m },
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go loop.Run(ctx)
	notify <- struct{}{}

	g.Consistently(delivered).WithTimeout(200 * time.Millisecond).ShouldNot(Receive())
}
```

- [ ] **Step 7: Write test — cursor advances between notifications**

```go
func TestAgentLoop_CursorAdvancesBetweenNotifications(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	callCount := 0
	var cursorsReceived []int

	notify := make(chan struct{}, 2)
	delivered := make(chan chat.Message, 10)

	loop := server.NewAgentLoop(server.AgentLoopConfig{
		Name:     "my-agent",
		WatchAll: true,
		Notify:   notify,
		ReadMessages: func(cursor int) ([]chat.Message, int, error) {
			cursorsReceived = append(cursorsReceived, cursor)
			callCount++

			if callCount == 1 {
				return []chat.Message{{From: "a", To: "all", Text: "first"}}, 5, nil
			}

			return []chat.Message{{From: "b", To: "all", Text: "second"}}, 10, nil
		},
		OnMessage: func(m chat.Message) { delivered <- m },
	})

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go loop.Run(ctx)

	notify <- struct{}{}
	g.Eventually(delivered).WithTimeout(time.Second).Should(Receive())

	notify <- struct{}{}
	g.Eventually(delivered).WithTimeout(time.Second).Should(Receive())

	cancel()

	// First call should start at cursor 0, second at cursor 5.
	g.Expect(cursorsReceived).To(HaveLen(2))
	g.Expect(cursorsReceived[0]).To(Equal(0))
	g.Expect(cursorsReceived[1]).To(Equal(5))
}
```

- [ ] **Step 8: Run all tests, verify pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 9: Refactor — review for DRY, SOLID**

Check: AgentLoop is focused. ReadMessagesFunc is the DI boundary for chat file I/O. OnMessage callback delivers to whatever the server wants (handler channels, etc.). WatchAll flag cleanly separates the engram-agent's "see everything" from regular agents.

- [ ] **Step 10: Commit**

```bash
git add internal/server/agent.go internal/server/agent_test.go
git commit -m "feat(server): add AgentLoop per-agent goroutine with cursor and filtering

AI-Used: [claude]"
```

---

### Task 3: HTTP handlers — POST /message, GET /status, POST /shutdown

The core HTTP handlers. POST /message writes to the chat file. GET /status returns health. POST /shutdown initiates graceful shutdown. All handlers receive dependencies via a `Deps` struct (DI).

**Files:**
- Create: `internal/server/handlers.go`
- Create: `internal/server/handlers_test.go`

- [ ] **Step 1: Write failing test — POST /message returns cursor**

```go
package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/chat"
	"engram/internal/server"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestPostMessage_AlwaysReturnsCursorOnSuccess(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "from")
		to := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "to")
		text := rapid.StringMatching(`[A-Za-z0-9 ]{5,50}`).Draw(rt, "text")
		cursor := rapid.IntRange(1, 100000).Draw(rt, "cursor")

		deps := &server.Deps{
			PostMessage: func(msg chat.Message) (int, error) {
				return cursor, nil
			},
		}

		body, _ := json.Marshal(map[string]string{
			"from": from, "to": to, "text": text,
		})

		req := httptest.NewRequest(http.MethodPost, "/message", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		server.HandlePostMessage(deps)(rec, req)

		g.Expect(rec.Code).To(Equal(http.StatusOK))

		var resp map[string]int
		decErr := json.NewDecoder(rec.Body).Decode(&resp)
		g.Expect(decErr).NotTo(HaveOccurred())
		g.Expect(resp["cursor"]).To(Equal(cursor))
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — Deps, HandlePostMessage not defined

- [ ] **Step 3: Write minimal implementation**

```go
package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"engram/internal/chat"
)

// PostFunc writes a message to the chat file and returns the new cursor.
type PostFunc func(msg chat.Message) (int, error)

// Deps holds injected dependencies for HTTP handlers.
type Deps struct {
	PostMessage PostFunc
	Logger      *slog.Logger
	ShutdownFn  context.CancelFunc // called by POST /shutdown
}

// HandlePostMessage returns an http.HandlerFunc for POST /message.
func HandlePostMessage(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			From string `json:"from"`
			To   string `json:"to"`
			Text string `json:"text"`
		}

		if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		cursor, postErr := deps.PostMessage(chat.Message{
			From: req.From,
			To:   req.To,
			Text: req.Text,
		})
		if postErr != nil {
			deps.Logger.Error("posting message", "err", postErr)
			http.Error(w, `{"error":"failed to post"}`, http.StatusInternalServerError)
			return
		}

		deps.Logger.Info("message posted",
			"from", req.From, "to", req.To,
			"text_len", len(req.Text), "cursor", cursor,
		)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]int{"cursor": cursor})
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Write test and implement GET /status**

```go
func TestStatus_AlwaysReturnsRunningTrue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	deps := &server.Deps{
		Logger: slog.Default(),
	}

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()

	server.HandleStatus(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var resp map[string]any
	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp["running"]).To(BeTrue())
}
```

Implementation:
```go
// HandleStatus returns an http.HandlerFunc for GET /status.
func HandleStatus(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"running": true,
		})
	}
}
```

- [ ] **Step 6: Write test and implement POST /shutdown**

```go
func TestShutdown_CallsShutdownFn(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	called := false
	deps := &server.Deps{
		Logger:     slog.Default(),
		ShutdownFn: func() { called = true },
	}

	req := httptest.NewRequest(http.MethodPost, "/shutdown", nil)
	rec := httptest.NewRecorder()

	server.HandleShutdown(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))
	g.Expect(called).To(BeTrue())
}
```

Implementation:
```go
// HandleShutdown returns an http.HandlerFunc for POST /shutdown.
func HandleShutdown(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		deps.Logger.Info("shutdown requested")

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "shutting down"})

		if deps.ShutdownFn != nil {
			deps.ShutdownFn()
		}
	}
}
```

- [ ] **Step 7: Run all tests, verify pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 8: Refactor — review Deps struct, handler patterns**

All handlers follow the same pattern: `HandleXxx(deps) http.HandlerFunc`. Deps is the DI container. Logger is used for structured debug logging. No I/O in handlers — PostMessage is injected.

- [ ] **Step 9: Commit**

```bash
git add internal/server/handlers.go internal/server/handlers_test.go
git commit -m "feat(server): add POST /message, GET /status, POST /shutdown handlers

AI-Used: [claude]"
```

---

### Task 4: HTTP handlers — GET /wait-for-response, GET /subscribe

Long-polling handlers. `/wait-for-response` watches the chat file independently from the provided cursor for a matching message. `/subscribe` returns new messages addressed to the agent.

**Files:**
- Modify: `internal/server/handlers.go`
- Modify: `internal/server/handlers_test.go`

- [ ] **Step 1: Write failing test — wait-for-response returns matching message**

```go
func TestWaitForResponse_ReturnsMatchingMessage(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	expected := chat.Message{From: "engram-agent", To: "lead-1", Text: "memory found"}

	deps := &server.Deps{
		Logger: slog.Default(),
		WatchForMessage: func(ctx context.Context, from, to string, afterCursor int) (chat.Message, int, error) {
			return expected, 10, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet,
		"/wait-for-response?from=engram-agent&to=lead-1&after-cursor=5", nil)
	rec := httptest.NewRecorder()

	server.HandleWaitForResponse(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var resp struct {
		Text   string `json:"text"`
		Cursor int    `json:"cursor"`
	}
	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp.Text).To(Equal("memory found"))
	g.Expect(resp.Cursor).To(Equal(10))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — WatchForMessage, HandleWaitForResponse not defined

- [ ] **Step 3: Add WatchForMessage to Deps, implement handler**

Add to Deps:
```go
// WatchForMessage blocks until a message matching from/to appears after the cursor.
// Independently watches the file (does not rely on goroutine cursors).
WatchForMessage func(ctx context.Context, from, to string, afterCursor int) (chat.Message, int, error)
```

Handler:
```go
// HandleWaitForResponse returns an http.HandlerFunc for GET /wait-for-response.
func HandleWaitForResponse(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		afterCursorStr := r.URL.Query().Get("after-cursor")

		afterCursor, parseErr := strconv.Atoi(afterCursorStr)
		if parseErr != nil {
			http.Error(w, `{"error":"invalid after-cursor"}`, http.StatusBadRequest)
			return
		}

		msg, newCursor, watchErr := deps.WatchForMessage(r.Context(), from, to, afterCursor)
		if watchErr != nil {
			deps.Logger.Error("watching for response", "err", watchErr)
			http.Error(w, `{"error":"watch failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"text":   msg.Text,
			"cursor": newCursor,
			"from":   msg.From,
			"to":     msg.To,
		})
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Write failing test — subscribe returns messages for agent**

```go
func TestSubscribe_ReturnsMessagesForAgent(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	messages := []chat.Message{
		{From: "engram-agent", To: "lead-1", Text: "memory surfaced"},
	}

	deps := &server.Deps{
		Logger: slog.Default(),
		SubscribeMessages: func(ctx context.Context, agent string, afterCursor int) ([]chat.Message, int, error) {
			return messages, 8, nil
		},
	}

	req := httptest.NewRequest(http.MethodGet,
		"/subscribe?agent=lead-1&after-cursor=5", nil)
	rec := httptest.NewRecorder()

	server.HandleSubscribe(deps)(rec, req)

	g.Expect(rec.Code).To(Equal(http.StatusOK))

	var resp struct {
		Messages []chat.Message `json:"messages"`
		Cursor   int            `json:"cursor"`
	}
	decErr := json.NewDecoder(rec.Body).Decode(&resp)
	g.Expect(decErr).NotTo(HaveOccurred())
	g.Expect(resp.Messages).To(HaveLen(1))
	g.Expect(resp.Messages[0].Text).To(Equal("memory surfaced"))
	g.Expect(resp.Cursor).To(Equal(8))
}
```

- [ ] **Step 6: Add SubscribeMessages to Deps, implement handler**

Add to Deps:
```go
// SubscribeMessages blocks until new messages for the agent appear after cursor.
SubscribeMessages func(ctx context.Context, agent string, afterCursor int) ([]chat.Message, int, error)
```

Handler:
```go
// HandleSubscribe returns an http.HandlerFunc for GET /subscribe.
func HandleSubscribe(deps *Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		agent := r.URL.Query().Get("agent")
		afterCursorStr := r.URL.Query().Get("after-cursor")

		afterCursor, parseErr := strconv.Atoi(afterCursorStr)
		if parseErr != nil {
			afterCursor = 0
		}

		messages, newCursor, watchErr := deps.SubscribeMessages(r.Context(), agent, afterCursor)
		if watchErr != nil {
			deps.Logger.Error("subscribing", "err", watchErr, "agent", agent)
			http.Error(w, `{"error":"subscribe failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"messages": messages,
			"cursor":   newCursor,
		})
	}
}
```

- [ ] **Step 7: Run all tests, verify pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 8: Refactor — review handler patterns for DRY**

All handlers share JSON response writing. Consider extracting a `writeJSON(w, data)` helper if 3+ handlers duplicate the pattern. Check if it reduces duplication meaningfully.

- [ ] **Step 9: Commit**

```bash
git add internal/server/handlers.go internal/server/handlers_test.go
git commit -m "feat(server): add GET /wait-for-response and GET /subscribe handlers

AI-Used: [claude]"
```

---

### Task 5: Server wiring — HTTP server setup, routing, slog, graceful shutdown

Wire everything together: HTTP server with routing, slog JSON handler, graceful shutdown via context cancellation.

**Files:**
- Create: `internal/server/server.go`
- Create: `internal/server/server_test.go`

- [ ] **Step 1: Write failing test — server starts and responds to /status**

```go
package server_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"engram/internal/server"

	. "github.com/onsi/gomega"
)

func TestServer_StatusEndpointResponds(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := server.Config{
		Addr:     "localhost:0", // OS-assigned port
		Logger:   slog.Default(),
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	srv, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())
	if startErr != nil {
		return
	}

	resp, httpErr := http.Get("http://" + srv.Addr() + "/status")
	g.Expect(httpErr).NotTo(HaveOccurred())
	if httpErr != nil {
		return
	}
	defer resp.Body.Close()

	g.Expect(resp.StatusCode).To(Equal(http.StatusOK))

	var body map[string]any
	g.Expect(json.NewDecoder(resp.Body).Decode(&body)).To(Succeed())
	g.Expect(body["running"]).To(BeTrue())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — Config, Start, Addr not defined

- [ ] **Step 3: Write minimal Server implementation**

```go
package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
)

// Config configures the API server.
type Config struct {
	Addr          string
	Logger        *slog.Logger
	PostFunc      PostFunc
	WatchFunc     func(ctx context.Context, from, to string, afterCursor int) (chat.Message, int, error)
	SubscribeFunc func(ctx context.Context, agent string, afterCursor int) ([]chat.Message, int, error)
}

// Server is the running engram API server.
type Server struct {
	httpServer *http.Server
	listener   net.Listener
	logger     *slog.Logger
}

// Start creates and starts the API server. Returns when the server is listening.
// The server shuts down when ctx is cancelled.
func Start(ctx context.Context, cfg Config) (*Server, error) {
	ctx, cancel := context.WithCancel(ctx)

	deps := &Deps{
		PostMessage:       cfg.PostFunc,
		WatchForMessage:   cfg.WatchFunc,
		SubscribeMessages: cfg.SubscribeFunc,
		Logger:            cfg.Logger,
		ShutdownFn:        cancel,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /message", HandlePostMessage(deps))
	mux.HandleFunc("GET /wait-for-response", HandleWaitForResponse(deps))
	mux.HandleFunc("GET /subscribe", HandleSubscribe(deps))
	mux.HandleFunc("GET /status", HandleStatus(deps))
	mux.HandleFunc("POST /shutdown", HandleShutdown(deps))

	listener, listenErr := net.Listen("tcp", cfg.Addr)
	if listenErr != nil {
		cancel()

		return nil, fmt.Errorf("server: listen: %w", listenErr)
	}

	httpServer := &http.Server{Handler: mux}

	srv := &Server{
		httpServer: httpServer,
		listener:   listener,
		logger:     cfg.Logger,
	}

	go func() {
		<-ctx.Done()
		srv.logger.Info("server shutting down")
		_ = httpServer.Close()
	}()

	go func() {
		srv.logger.Info("server started", "addr", listener.Addr().String())

		if serveErr := httpServer.Serve(listener); serveErr != http.ErrServerClosed {
			srv.logger.Error("server error", "err", serveErr)
		}
	}()

	return srv, nil
}

// Addr returns the server's listen address (useful when port=0).
func (s *Server) Addr() string {
	return s.listener.Addr().String()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Write test — shutdown via POST /shutdown**

```go
func TestServer_ShutdownEndpointStopsServer(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	cfg := server.Config{
		Addr:     "localhost:0",
		Logger:   slog.Default(),
		PostFunc: func(_ chat.Message) (int, error) { return 0, nil },
		WatchFunc: func(_ context.Context, _, _ string, _ int) (chat.Message, int, error) {
			return chat.Message{}, 0, nil
		},
		SubscribeFunc: func(_ context.Context, _ string, _ int) ([]chat.Message, int, error) {
			return nil, 0, nil
		},
	}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	srv, startErr := server.Start(ctx, cfg)
	g.Expect(startErr).NotTo(HaveOccurred())
	if startErr != nil {
		return
	}

	addr := srv.Addr()

	// POST /shutdown
	shutdownReq, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://"+addr+"/shutdown", nil)
	_, shutdownErr := http.DefaultClient.Do(shutdownReq)
	g.Expect(shutdownErr).NotTo(HaveOccurred())

	// Server should stop accepting connections.
	g.Eventually(func() error {
		_, err := http.Get("http://" + addr + "/status")
		return err
	}).WithTimeout(2 * time.Second).Should(HaveOccurred())
}
```

- [ ] **Step 6: Run tests, verify pass**

Run: `targ test`
Expected: PASS

- [ ] **Step 7: Refactor — review server setup**

Check: Server creation is clean. Context cancellation triggers shutdown. Deps struct is the single DI point. Logger is injected. No global state.

- [ ] **Step 8: Commit**

```bash
git add internal/server/server.go internal/server/server_test.go
git commit -m "feat(server): add Server with routing, graceful shutdown, slog

AI-Used: [claude]"
```

---

### Task 6: CLI command — `engram server up`

The thin wiring layer: parses flags, constructs real I/O (FilePoster, FileWatcher, slog handler), creates the server, blocks until context cancelled.

**Files:**
- Create: `internal/cli/cli_server.go`
- Modify: `internal/cli/cli.go` (add case to Run)
- Modify: `internal/cli/targets.go` (add ServerUpArgs, targ registration)

- [ ] **Step 1: Write failing test — server up command starts and responds**

```go
// In internal/cli/cli_server_test.go
package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"engram/internal/cli"

	. "github.com/onsi/gomega"
)

func TestRunServerUp_StartsAndRespondsToStatus(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := dir + "/chat.toml"

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var stdout, stderr bytes.Buffer

	done := make(chan error, 1)
	go func() {
		done <- cli.RunWithContext(ctx, []string{
			"engram", "server", "up",
			"--chat-file", chatFile,
			"--addr", "localhost:0",
		}, &stdout, &stderr, nil)
	}()

	// Wait for server to start (check stderr for "server started").
	g.Eventually(func() string { return stderr.String() }).
		WithTimeout(5 * time.Second).
		Should(ContainSubstring("server started"))

	// Extract address from log... this is tricky with port=0.
	// Alternative: use a known port for testing.
	cancel()
	<-done
}
```

NOTE: Testing a long-running server command is inherently integration-y. The pure handler functions (doXxx) were already tested via imptest in Stage 0. This test verifies the wiring works.

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test`
Expected: FAIL — "server" not recognized in RunWithContext

- [ ] **Step 3: Implement cli_server.go + wiring**

```go
package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"engram/internal/chat"
	"engram/internal/server"
	"engram/internal/watch"
)

const serverCmd = "server"

func runServerDispatch(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("server: subcommand required (up)")
	}

	switch args[0] {
	case "up":
		return runServerUp(ctx, args[1:], stdout, stderr)
	default:
		return fmt.Errorf("server: unknown subcommand: %s", args[0])
	}
}

func runServerUp(ctx context.Context, args []string, _, stderr io.Writer) error {
	fs := newFlagSet("server up")

	var chatFilePath, logFilePath, addr string
	fs.StringVar(&chatFilePath, "chat-file", "", "chat file path")
	fs.StringVar(&logFilePath, "log-file", "", "log file path (optional)")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "listen address")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("server up: %w", parseErr)
	}

	if chatFilePath == "" {
		chatFilePath = defaultChatFilePath()
	}

	// Setup slog handler.
	logger := slog.New(slog.NewJSONHandler(stderr, nil))

	// Setup chat I/O.
	poster := newFilePoster(chatFilePath)
	postFunc := func(msg chat.Message) (int, error) {
		return poster.Post(msg)
	}

	// WatchForMessage: independent file watcher per long-poll.
	watchFunc := func(ctx context.Context, from, to string, afterCursor int) (chat.Message, int, error) {
		watcher := newFileWatcher(chatFilePath)
		return watcher.Watch(ctx, to, afterCursor, nil)
	}

	// SubscribeMessages: independent file watcher per subscribe.
	subscribeFunc := func(ctx context.Context, agent string, afterCursor int) ([]chat.Message, int, error) {
		watcher := newFileWatcher(chatFilePath)
		return watcher.Watch(ctx, agent, afterCursor, nil)
	}

	cfg := server.Config{
		Addr:          addr,
		Logger:        logger,
		PostFunc:      postFunc,
		WatchFunc:     watchFunc,
		SubscribeFunc: subscribeFunc,
	}

	_, startErr := server.Start(ctx, cfg)
	if startErr != nil {
		return fmt.Errorf("server up: %w", startErr)
	}

	logger.Info("server started", "addr", addr, "chat-file", chatFilePath)

	// Block until context cancelled (Ctrl+C or POST /shutdown).
	<-ctx.Done()

	return nil
}
```

Add to `Run()` in cli.go:
```go
	case "server":
		serverCtx, serverStop := signal.NotifyContext(
			context.Background(), os.Interrupt, syscall.SIGTERM,
		)
		defer serverStop()
		return runServerDispatch(serverCtx, subArgs, stdout, stderr)
```

Add to `RunWithContext`:
```go
	case "server":
		return runServerDispatch(ctx, subArgs, stdout, stderr)
```

Add to `targets.go`:
```go
// ServerUpArgs holds flags for `engram server up`.
type ServerUpArgs struct {
	ChatFile string `targ:"flag,name=chat-file,desc=chat file path"`
	LogFile  string `targ:"flag,name=log-file,desc=log file path"`
	Addr     string `targ:"flag,name=addr,desc=listen address"`
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test`
Expected: PASS

- [ ] **Step 5: Refactor**

The `runServerUp` function creates real I/O (FilePoster, FileWatcher) — this is correct as it's the thin wiring layer. The `server.Start` function receives everything via `Config` (DI). Clean separation.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli_server.go internal/cli/cli_server_test.go internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram server up command

AI-Used: [claude]"
```

---

### Task 7: Learn message validation

The server validates learn messages before posting to chat. Feedback requires situation/behavior/impact/action. Fact requires situation/subject/predicate/object. Invalid messages are rejected with guidance pointing to the skill file. After 3 rejections, accept raw content.

**Files:**
- Create: `internal/server/validate.go`
- Create: `internal/server/validate_test.go`
- Modify: `internal/server/handlers.go` (integrate validation into POST /message)

- [ ] **Step 1: Write failing property test — valid feedback always accepted**

```go
package server_test

import (
	"testing"

	"engram/internal/server"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestValidateLearn_ValidFeedbackAlwaysAccepted(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		text := `{"type":"feedback","situation":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "sit") +
			`","behavior":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "beh") +
			`","impact":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "imp") +
			`","action":"` +
			rapid.StringMatching(`[a-z ]{5,30}`).Draw(rt, "act") + `"}`

		err := server.ValidateLearnMessage(text)
		g.Expect(err).NotTo(HaveOccurred())
	})
}

func TestValidateLearn_MissingFieldAlwaysRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Missing "action" field.
	err := server.ValidateLearnMessage(`{"type":"feedback","situation":"s","behavior":"b","impact":"i"}`)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("action"))
}

func TestValidateLearn_InvalidTypeAlwaysRejected(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	err := server.ValidateLearnMessage(`{"type":"bogus","situation":"s"}`)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("must be 'feedback' or 'fact'"))
}
```

- [ ] **Step 2: Implement ValidateLearnMessage**

```go
package server

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	errLearnTypeMustBeFeedbackOrFact = errors.New("learn: type must be 'feedback' or 'fact'")
)

// ValidateLearnMessage validates a learn message's JSON text.
func ValidateLearnMessage(text string) error {
	var parsed map[string]string
	if jsonErr := json.Unmarshal([]byte(text), &parsed); jsonErr != nil {
		return fmt.Errorf("learn: invalid JSON: %w", jsonErr)
	}

	switch parsed["type"] {
	case "feedback":
		for _, field := range []string{"situation", "behavior", "impact", "action"} {
			if parsed[field] == "" {
				return fmt.Errorf("learn: missing required feedback field: %s", field)
			}
		}
	case "fact":
		for _, field := range []string{"situation", "subject", "predicate", "object"} {
			if parsed[field] == "" {
				return fmt.Errorf("learn: missing required fact field: %s", field)
			}
		}
	default:
		return fmt.Errorf("%w, got %q", errLearnTypeMustBeFeedbackOrFact, parsed["type"])
	}

	return nil
}
```

- [ ] **Step 3: Run tests, verify pass**

- [ ] **Step 4: Integrate validation into POST /message handler**

In `HandlePostMessage`, after decoding the request, check if the message looks like a learn message (text starts with `{` and contains `"type":`). If so, validate. If validation fails, return 400 with guidance.

```go
// In HandlePostMessage, after decoding req:
if isLearnMessage(req.Text) {
    if valErr := ValidateLearnMessage(req.Text); valErr != nil {
        w.WriteHeader(http.StatusBadRequest)
        _ = json.NewEncoder(w).Encode(map[string]string{
            "error": valErr.Error(),
        })
        deps.Logger.Warn("learn validation failed", "from", req.From, "err", valErr)
        return
    }
}
```

- [ ] **Step 5: Write test for validation integration in handler**

- [ ] **Step 6: Run all tests, verify pass**

- [ ] **Step 7: Commit**

```bash
git add internal/server/validate.go internal/server/validate_test.go internal/server/handlers.go internal/server/handlers_test.go
git commit -m "feat(server): add learn message validation in POST /message

AI-Used: [claude]"
```

---

### Task 8: Skill refresh tracking

The server tracks interaction count per agent. Every 13 engram-agent invocations, prepend skill reload. Every 13 messages delivered to the lead, post a skill refresh message to chat.

**Files:**
- Create: `internal/server/refresh.go`
- Create: `internal/server/refresh_test.go`

- [ ] **Step 1: Write failing property test — refresh fires every N interactions**

```go
func TestRefreshTracker_AlwaysFiresAtInterval(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		interval := rapid.IntRange(2, 20).Draw(rt, "interval")
		interactions := rapid.IntRange(1, 100).Draw(rt, "interactions")

		tracker := server.NewRefreshTracker(interval)
		refreshCount := 0

		for range interactions {
			if tracker.ShouldRefresh() {
				refreshCount++
			}
		}

		expectedRefreshes := interactions / interval
		g.Expect(refreshCount).To(Equal(expectedRefreshes))
	})
}
```

- [ ] **Step 2: Implement RefreshTracker**

```go
package server

const skillRefreshInterval = 13

// RefreshTracker tracks interaction count and signals when to refresh skills.
type RefreshTracker struct {
	interval int
	count    int
}

// NewRefreshTracker creates a tracker with the given interval.
func NewRefreshTracker(interval int) *RefreshTracker {
	return &RefreshTracker{interval: interval}
}

// ShouldRefresh increments the counter and returns true every N interactions.
func (rt *RefreshTracker) ShouldRefresh() bool {
	rt.count++

	return rt.count%rt.interval == 0
}
```

- [ ] **Step 3: Run tests, verify pass**

- [ ] **Step 4: Commit**

```bash
git add internal/server/refresh.go internal/server/refresh_test.go
git commit -m "feat(server): add RefreshTracker for skill refresh every N interactions

AI-Used: [claude]"
```

---

### Task 9: Full quality check + e2e testing

**Files:** No new files.

- [ ] **Step 1: Run full test suite**

Run: `targ test`
Expected: All tests pass

- [ ] **Step 2: Run full quality check**

Run: `targ check-full`
Expected: All checks pass

- [ ] **Step 3: Fix any lint, coverage, nilaway issues**

- [ ] **Step 4: E2E test — build binary, start server, hit endpoints**

```bash
go build -o /tmp/engram-1a ./cmd/engram/

# Start server in background
/tmp/engram-1a server up --chat-file /tmp/test-chat.toml --addr localhost:19877 &
SERVER_PID=$!
sleep 1

# Test status
curl -s http://localhost:19877/status | jq .

# Test post message
curl -s -X POST http://localhost:19877/message \
  -d '{"from":"lead-1","to":"engram-agent","text":"hello"}' | jq .

# Test shutdown
curl -s -X POST http://localhost:19877/shutdown | jq .
wait $SERVER_PID
```

- [ ] **Step 5: Fix any bugs found in e2e**

- [ ] **Step 6: Commit fixes**

```bash
git add -A
git commit -m "fix: address quality and e2e issues from stage 1a

AI-Used: [claude]"
```
