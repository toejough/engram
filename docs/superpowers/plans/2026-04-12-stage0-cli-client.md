# Stage 0: CLI Client Commands — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add new CLI commands (`engram post`, `engram intent`, `engram learn`, `engram subscribe`, `engram status`) that are thin HTTP clients to the engram API server.

**Architecture:** A new `internal/apiclient` package provides the HTTP client library. New targ commands in `internal/cli` wrap it. The API server doesn't exist yet — tests use `httptest.NewServer` fakes. Existing commands are untouched.

**Tech Stack:** Go stdlib `net/http`, `encoding/json`, `net/http/httptest` (tests), targ CLI framework, gomega assertions.

---

### Task 1: Create the API client library — PostMessage

**Files:**
- Create: `internal/apiclient/client.go`
- Create: `internal/apiclient/client_test.go`

- [ ] **Step 1: Write the failing test for PostMessage**

```go
package apiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/apiclient"

	. "github.com/onsi/gomega"
)

func TestPostMessage_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var gotBody apiclient.PostMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal(http.MethodPost))
		g.Expect(r.URL.Path).To(Equal("/message"))

		decErr := json.NewDecoder(r.Body).Decode(&gotBody)
		g.Expect(decErr).NotTo(HaveOccurred())

		w.WriteHeader(http.StatusOK)
		resp := apiclient.PostMessageResponse{Cursor: 42}
		encErr := json.NewEncoder(w).Encode(resp)
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, srv.Client())
	resp, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
		From: "lead-1",
		To:   "engram-agent",
		Text: "hello world",
	})
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(resp.Cursor).To(Equal(42))
	g.Expect(gotBody.From).To(Equal("lead-1"))
	g.Expect(gotBody.To).To(Equal("engram-agent"))
	g.Expect(gotBody.Text).To(Equal("hello world"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestPostMessage_Success ./internal/apiclient/`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write minimal implementation**

```go
// Package apiclient provides a thin HTTP client for the engram API server.
package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// HTTPDoer abstracts http.Client for testing.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is a thin HTTP client for the engram API server.
type Client struct {
	baseURL string
	doer    HTTPDoer
}

// New creates a Client. Pass http.DefaultClient as doer in production.
func New(baseURL string, doer HTTPDoer) *Client {
	return &Client{baseURL: baseURL, doer: doer}
}

// PostMessageRequest is the request body for POST /message.
type PostMessageRequest struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}

// PostMessageResponse is the response from POST /message.
type PostMessageResponse struct {
	Cursor int    `json:"cursor"`
	Error  string `json:"error,omitempty"`
}

// PostMessage posts a message to the chat via the API server.
func (c *Client) PostMessage(
	ctx context.Context,
	req PostMessageRequest,
) (PostMessageResponse, error) {
	body, marshalErr := json.Marshal(req)
	if marshalErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: marshaling request: %w", marshalErr)
	}

	httpReq, reqErr := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.baseURL+"/message",
		bytes.NewReader(body),
	)
	if reqErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: posting message: %w", doErr)
	}
	defer httpResp.Body.Close()

	var resp PostMessageResponse

	decErr := json.NewDecoder(httpResp.Body).Decode(&resp)
	if decErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("apiclient: server returned %d: %s", httpResp.StatusCode, resp.Error)
	}

	return resp, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestPostMessage_Success ./internal/apiclient/`
Expected: PASS

- [ ] **Step 5: Write test for PostMessage validation error**

```go
func TestPostMessage_ValidationError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		resp := apiclient.PostMessageResponse{
			Error: "missing required field: situation",
		}
		encErr := json.NewEncoder(w).Encode(resp)
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, srv.Client())
	resp, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
		From: "lead-1",
		To:   "engram-agent",
		Text: "bad message",
	})
	g.Expect(err).To(HaveOccurred())
	g.Expect(resp.Error).To(ContainSubstring("missing required field"))
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `targ test -- -run TestPostMessage_ValidationError ./internal/apiclient/`
Expected: PASS (the existing implementation already handles non-200 status)

- [ ] **Step 7: Commit**

```bash
git add internal/apiclient/client.go internal/apiclient/client_test.go
git commit -m "feat(apiclient): add PostMessage HTTP client

AI-Used: [claude]"
```

---

### Task 2: API client — WaitForResponse

**Files:**
- Modify: `internal/apiclient/client.go`
- Modify: `internal/apiclient/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestWaitForResponse_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal(http.MethodGet))
		g.Expect(r.URL.Path).To(Equal("/wait-for-response"))
		g.Expect(r.URL.Query().Get("from")).To(Equal("engram-agent"))
		g.Expect(r.URL.Query().Get("to")).To(Equal("lead-1"))
		g.Expect(r.URL.Query().Get("after-cursor")).To(Equal("10"))

		w.WriteHeader(http.StatusOK)
		resp := apiclient.WaitForResponseResponse{
			Text:   "Relevant memory: always use DI",
			Cursor: 15,
		}
		encErr := json.NewEncoder(w).Encode(resp)
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, srv.Client())
	resp, err := client.WaitForResponse(context.Background(), apiclient.WaitForResponseRequest{
		From:        "engram-agent",
		To:          "lead-1",
		AfterCursor: 10,
	})
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(resp.Text).To(Equal("Relevant memory: always use DI"))
	g.Expect(resp.Cursor).To(Equal(15))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestWaitForResponse_Success ./internal/apiclient/`
Expected: FAIL — WaitForResponse not defined

- [ ] **Step 3: Write minimal implementation**

```go
// WaitForResponseRequest is the query for GET /wait-for-response.
type WaitForResponseRequest struct {
	From        string
	To          string
	AfterCursor int
}

// WaitForResponseResponse is the response from GET /wait-for-response.
type WaitForResponseResponse struct {
	Text   string `json:"text"`
	Cursor int    `json:"cursor"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// WaitForResponse long-polls for a response message matching the filter.
func (c *Client) WaitForResponse(
	ctx context.Context,
	req WaitForResponseRequest,
) (WaitForResponseResponse, error) {
	url := fmt.Sprintf(
		"%s/wait-for-response?from=%s&to=%s&after-cursor=%d",
		c.baseURL, req.From, req.To, req.AfterCursor,
	)

	httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return WaitForResponseResponse{}, fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return WaitForResponseResponse{}, fmt.Errorf("apiclient: waiting for response: %w", doErr)
	}
	defer httpResp.Body.Close()

	var resp WaitForResponseResponse

	decErr := json.NewDecoder(httpResp.Body).Decode(&resp)
	if decErr != nil {
		return WaitForResponseResponse{}, fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("apiclient: server returned %d", httpResp.StatusCode)
	}

	return resp, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestWaitForResponse_Success ./internal/apiclient/`
Expected: PASS

- [ ] **Step 5: Write test for context cancellation**

```go
func TestWaitForResponse_ContextCancelled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Block forever — context cancellation should unblock the client.
		select {}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	client := apiclient.New(srv.URL, srv.Client())
	_, err := client.WaitForResponse(ctx, apiclient.WaitForResponseRequest{
		From:        "engram-agent",
		To:          "lead-1",
		AfterCursor: 0,
	})
	g.Expect(err).To(HaveOccurred())
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `targ test -- -run TestWaitForResponse_ContextCancelled ./internal/apiclient/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/apiclient/client.go internal/apiclient/client_test.go
git commit -m "feat(apiclient): add WaitForResponse long-poll client

AI-Used: [claude]"
```

---

### Task 3: API client — Subscribe

**Files:**
- Modify: `internal/apiclient/client.go`
- Modify: `internal/apiclient/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestSubscribe_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal(http.MethodGet))
		g.Expect(r.URL.Path).To(Equal("/subscribe"))
		g.Expect(r.URL.Query().Get("agent")).To(Equal("lead-1"))
		g.Expect(r.URL.Query().Get("after-cursor")).To(Equal("5"))

		w.WriteHeader(http.StatusOK)
		resp := apiclient.SubscribeResponse{
			Messages: []apiclient.ChatMessage{
				{From: "engram-agent", To: "lead-1", Text: "memory surfaced"},
			},
			Cursor: 8,
		}
		encErr := json.NewEncoder(w).Encode(resp)
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, srv.Client())
	resp, err := client.Subscribe(context.Background(), apiclient.SubscribeRequest{
		Agent:       "lead-1",
		AfterCursor: 5,
	})
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(resp.Messages).To(HaveLen(1))
	g.Expect(resp.Messages[0].Text).To(Equal("memory surfaced"))
	g.Expect(resp.Cursor).To(Equal(8))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestSubscribe_Success ./internal/apiclient/`
Expected: FAIL — Subscribe not defined

- [ ] **Step 3: Write minimal implementation**

```go
// ChatMessage represents a single message from the chat file.
type ChatMessage struct {
	From string `json:"from"`
	To   string `json:"to"`
	Text string `json:"text"`
}

// SubscribeRequest is the query for GET /subscribe.
type SubscribeRequest struct {
	Agent       string
	AfterCursor int
}

// SubscribeResponse is the response from GET /subscribe.
type SubscribeResponse struct {
	Messages []ChatMessage `json:"messages"`
	Cursor   int           `json:"cursor"`
}

// Subscribe long-polls for new messages addressed to the named agent.
func (c *Client) Subscribe(
	ctx context.Context,
	req SubscribeRequest,
) (SubscribeResponse, error) {
	url := fmt.Sprintf(
		"%s/subscribe?agent=%s&after-cursor=%d",
		c.baseURL, req.Agent, req.AfterCursor,
	)

	httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return SubscribeResponse{}, fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return SubscribeResponse{}, fmt.Errorf("apiclient: subscribing: %w", doErr)
	}
	defer httpResp.Body.Close()

	var resp SubscribeResponse

	decErr := json.NewDecoder(httpResp.Body).Decode(&resp)
	if decErr != nil {
		return SubscribeResponse{}, fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("apiclient: server returned %d", httpResp.StatusCode)
	}

	return resp, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestSubscribe_Success ./internal/apiclient/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/apiclient/client.go internal/apiclient/client_test.go
git commit -m "feat(apiclient): add Subscribe long-poll client

AI-Used: [claude]"
```

---

### Task 4: API client — Status

**Files:**
- Modify: `internal/apiclient/client.go`
- Modify: `internal/apiclient/client_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestStatus_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.Expect(r.Method).To(Equal(http.MethodGet))
		g.Expect(r.URL.Path).To(Equal("/status"))

		w.WriteHeader(http.StatusOK)
		resp := apiclient.StatusResponse{
			Running: true,
			Agents:  []string{"engram-agent", "lead-1"},
		}
		encErr := json.NewEncoder(w).Encode(resp)
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	client := apiclient.New(srv.URL, srv.Client())
	resp, err := client.Status(context.Background())
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(resp.Running).To(BeTrue())
	g.Expect(resp.Agents).To(ConsistOf("engram-agent", "lead-1"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestStatus_Success ./internal/apiclient/`
Expected: FAIL — Status not defined

- [ ] **Step 3: Write minimal implementation**

```go
// StatusResponse is the response from GET /status.
type StatusResponse struct {
	Running bool     `json:"running"`
	Agents  []string `json:"agents"`
}

// Status returns the server's health and connected agents.
func (c *Client) Status(ctx context.Context) (StatusResponse, error) {
	httpReq, reqErr := http.NewRequestWithContext(
		ctx, http.MethodGet, c.baseURL+"/status", nil,
	)
	if reqErr != nil {
		return StatusResponse{}, fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return StatusResponse{}, fmt.Errorf("apiclient: checking status: %w", doErr)
	}
	defer httpResp.Body.Close()

	var resp StatusResponse

	decErr := json.NewDecoder(httpResp.Body).Decode(&resp)
	if decErr != nil {
		return StatusResponse{}, fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("apiclient: server returned %d", httpResp.StatusCode)
	}

	return resp, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestStatus_Success ./internal/apiclient/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/apiclient/client.go internal/apiclient/client_test.go
git commit -m "feat(apiclient): add Status health check client

AI-Used: [claude]"
```

---

### Task 5: CLI command — `engram post`

**Files:**
- Create: `internal/cli/cli_api.go`
- Create: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/cli.go` (add case to Run switch)
- Modify: `internal/cli/targets.go` (add args struct, flags func, targ registration)

- [ ] **Step 1: Write the failing test for the Run dispatch**

```go
// In cli_api_test.go
package cli_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/apiclient"
	"engram/internal/cli"

	. "github.com/onsi/gomega"
)

func TestRunPost_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body apiclient.PostMessageRequest
		decErr := json.NewDecoder(r.Body).Decode(&body)
		g.Expect(decErr).NotTo(HaveOccurred())
		g.Expect(body.From).To(Equal("lead-1"))
		g.Expect(body.To).To(Equal("engram-agent"))
		g.Expect(body.Text).To(Equal("hello"))

		w.WriteHeader(http.StatusOK)
		encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 7})
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.Run(
		[]string{
			"engram", "post",
			"--from", "lead-1",
			"--to", "engram-agent",
			"--text", "hello",
			"--addr", srv.URL,
		},
		&stdout, &stderr, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("7"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunPost_Success ./internal/cli/`
Expected: FAIL — unknown command: post

- [ ] **Step 3: Add PostArgs and PostFlags to targets.go**

Add to `internal/cli/targets.go`:

```go
// PostArgs holds flags for `engram post`.
type PostArgs struct {
	From string `targ:"flag,name=from,desc=sender agent name"`
	To   string `targ:"flag,name=to,desc=recipient agent name"`
	Text string `targ:"flag,name=text,desc=message content"`
	Addr string `targ:"flag,name=addr,desc=API server address"`
}

// PostFlags returns the CLI flag args for the post subcommand.
func PostFlags(a PostArgs) []string {
	return BuildFlags(
		"--from", a.From,
		"--to", a.To,
		"--text", a.Text,
		"--addr", a.Addr,
	)
}
```

- [ ] **Step 4: Add the `post` case to Run in cli.go**

Add to the switch in `Run()`:

```go
	case "post":
		return runPost(subArgs, stdout)
```

- [ ] **Step 5: Implement runPost in cli_api.go**

```go
// Package cli — API client commands.
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"

	"engram/internal/apiclient"
)

const defaultAPIAddr = "http://localhost:7932"

func runPost(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("post", flag.ContinueOnError)

	var from, to, text, addr string
	fs.StringVar(&from, "from", "", "sender agent name")
	fs.StringVar(&to, "to", "", "recipient agent name")
	fs.StringVar(&text, "text", "", "message content")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("post: %w", parseErr)
	}

	client := apiclient.New(addr, http.DefaultClient)
	resp, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
		From: from,
		To:   to,
		Text: text,
	})

	if err != nil {
		return fmt.Errorf("post: %w", err)
	}

	_, printErr := fmt.Fprintf(stdout, "%d\n", resp.Cursor)

	return printErr
}
```

- [ ] **Step 6: Register in Targets function**

Add to `BuildTargets` in `targets.go`:

```go
		targ.Targ(func(a PostArgs) { run("post", PostFlags(a)) }).
			Name("post").Description("Post a message to the engram chat"),
```

- [ ] **Step 7: Run test to verify it passes**

Run: `targ test -- -run TestRunPost_Success ./internal/cli/`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram post command as API client

AI-Used: [claude]"
```

---

### Task 6: CLI command — `engram intent`

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/cli.go` (add case)
- Modify: `internal/cli/targets.go` (add args, flags, registration)

- [ ] **Step 1: Write the failing test**

```go
func TestRunIntent_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/message":
			callCount++
			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 10})
			g.Expect(encErr).NotTo(HaveOccurred())

		case r.Method == http.MethodGet && r.URL.Path == "/wait-for-response":
			callCount++
			g.Expect(r.URL.Query().Get("from")).To(Equal("engram-agent"))
			g.Expect(r.URL.Query().Get("to")).To(Equal("lead-1"))
			g.Expect(r.URL.Query().Get("after-cursor")).To(Equal("10"))

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.WaitForResponseResponse{
				Text:   "Relevant memory: use DI everywhere",
				Cursor: 12,
			})
			g.Expect(encErr).NotTo(HaveOccurred())

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.Run(
		[]string{
			"engram", "intent",
			"--from", "lead-1",
			"--to", "engram-agent",
			"--situation", "about to refactor the chat module",
			"--planned-action", "split poster into two files",
			"--addr", srv.URL,
		},
		&stdout, &stderr, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("Relevant memory: use DI everywhere"))
	g.Expect(callCount).To(Equal(2))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunIntent_Success ./internal/cli/`
Expected: FAIL — unknown command: intent

- [ ] **Step 3: Add IntentArgs and IntentFlags to targets.go**

```go
// IntentArgs holds flags for `engram intent`.
type IntentArgs struct {
	From          string `targ:"flag,name=from,desc=sender agent name"`
	To            string `targ:"flag,name=to,desc=recipient agent name"`
	Situation     string `targ:"flag,name=situation,desc=situational context"`
	PlannedAction string `targ:"flag,name=planned-action,desc=what you plan to do"`
	Addr          string `targ:"flag,name=addr,desc=API server address"`
}

// IntentFlags returns the CLI flag args for the intent subcommand.
func IntentFlags(a IntentArgs) []string {
	return BuildFlags(
		"--from", a.From,
		"--to", a.To,
		"--situation", a.Situation,
		"--planned-action", a.PlannedAction,
		"--addr", a.Addr,
	)
}
```

- [ ] **Step 4: Add `intent` case to Run and implement runIntent**

Add to cli.go switch:
```go
	case "intent":
		return runIntent(subArgs, stdout)
```

Add to cli_api.go:
```go
func runIntent(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("intent", flag.ContinueOnError)

	var from, to, situation, plannedAction, addr string
	fs.StringVar(&from, "from", "", "sender agent name")
	fs.StringVar(&to, "to", "", "recipient agent name")
	fs.StringVar(&situation, "situation", "", "situational context")
	fs.StringVar(&plannedAction, "planned-action", "", "what you plan to do")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("intent: %w", parseErr)
	}

	client := apiclient.New(addr, http.DefaultClient)

	// Step 1: Post the intent message.
	text := fmt.Sprintf("INTENT: situation=%s planned_action=%s", situation, plannedAction)
	postResp, postErr := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
		From: from,
		To:   to,
		Text: text,
	})
	if postErr != nil {
		return fmt.Errorf("intent: posting: %w", postErr)
	}

	// Step 2: Block until the engram-agent responds.
	waitResp, waitErr := client.WaitForResponse(context.Background(), apiclient.WaitForResponseRequest{
		From:        to,
		To:          from,
		AfterCursor: postResp.Cursor,
	})
	if waitErr != nil {
		return fmt.Errorf("intent: waiting: %w", waitErr)
	}

	_, printErr := fmt.Fprintln(stdout, waitResp.Text)

	return printErr
}
```

- [ ] **Step 5: Register in BuildTargets**

```go
		targ.Targ(func(a IntentArgs) { run("intent", IntentFlags(a)) }).
			Name("intent").Description("Announce intent and wait for surfaced memories"),
```

- [ ] **Step 6: Run test to verify it passes**

Run: `targ test -- -run TestRunIntent_Success ./internal/cli/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram intent command (post + wait-for-response)

AI-Used: [claude]"
```

---

### Task 7: CLI command — `engram learn`

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/cli.go` (add case)
- Modify: `internal/cli/targets.go` (add args, flags, registration)

- [ ] **Step 1: Write the failing test**

```go
func TestRunLearn_Feedback(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var gotBody apiclient.PostMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decErr := json.NewDecoder(r.Body).Decode(&gotBody)
		g.Expect(decErr).NotTo(HaveOccurred())

		w.WriteHeader(http.StatusOK)
		encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 20})
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.Run(
		[]string{
			"engram", "learn",
			"--from", "lead-1",
			"--type", "feedback",
			"--situation", "refactoring chat module",
			"--behavior", "split large files into focused modules",
			"--impact", "easier to test and review",
			"--action", "continue splitting when files exceed 300 lines",
			"--addr", srv.URL,
		},
		&stdout, &stderr, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gotBody.From).To(Equal("lead-1"))
	g.Expect(gotBody.Text).To(ContainSubstring("feedback"))
	g.Expect(gotBody.Text).To(ContainSubstring("refactoring chat module"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunLearn_Feedback ./internal/cli/`
Expected: FAIL — unknown command: learn

- [ ] **Step 3: Add LearnArgs and LearnFlags to targets.go**

```go
// LearnArgs holds flags for `engram learn`.
type LearnArgs struct {
	From      string `targ:"flag,name=from,desc=sender agent name"`
	Type      string `targ:"flag,name=type,desc=feedback or fact"`
	Situation string `targ:"flag,name=situation,desc=when this applies"`
	Behavior  string `targ:"flag,name=behavior,desc=(feedback) what was done"`
	Impact    string `targ:"flag,name=impact,desc=(feedback) what resulted"`
	Action    string `targ:"flag,name=action,desc=(feedback) what to do next"`
	Subject   string `targ:"flag,name=subject,desc=(fact) subject"`
	Predicate string `targ:"flag,name=predicate,desc=(fact) predicate"`
	Object    string `targ:"flag,name=object,desc=(fact) object"`
	Addr      string `targ:"flag,name=addr,desc=API server address"`
}

// LearnFlags returns the CLI flag args for the learn subcommand.
func LearnFlags(a LearnArgs) []string {
	return BuildFlags(
		"--from", a.From,
		"--type", a.Type,
		"--situation", a.Situation,
		"--behavior", a.Behavior,
		"--impact", a.Impact,
		"--action", a.Action,
		"--subject", a.Subject,
		"--predicate", a.Predicate,
		"--object", a.Object,
		"--addr", a.Addr,
	)
}
```

- [ ] **Step 4: Add `learn` case and implement runLearn**

Add to cli.go switch:
```go
	case "learn":
		return runLearn(subArgs, stdout)
```

Add to cli_api.go:
```go
func runLearn(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("learn", flag.ContinueOnError)

	var from, learnType, situation, addr string
	var behavior, impact, action string
	var subject, predicate, object string
	fs.StringVar(&from, "from", "", "sender agent name")
	fs.StringVar(&learnType, "type", "", "feedback or fact")
	fs.StringVar(&situation, "situation", "", "when this applies")
	fs.StringVar(&behavior, "behavior", "", "(feedback) what was done")
	fs.StringVar(&impact, "impact", "", "(feedback) what resulted")
	fs.StringVar(&action, "action", "", "(feedback) what to do next")
	fs.StringVar(&subject, "subject", "", "(fact) subject")
	fs.StringVar(&predicate, "predicate", "", "(fact) predicate")
	fs.StringVar(&object, "object", "", "(fact) object")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("learn: %w", parseErr)
	}

	var text string

	switch learnType {
	case "feedback":
		text = fmt.Sprintf(
			`{"type":"feedback","situation":%q,"behavior":%q,"impact":%q,"action":%q}`,
			situation, behavior, impact, action,
		)
	case "fact":
		text = fmt.Sprintf(
			`{"type":"fact","situation":%q,"subject":%q,"predicate":%q,"object":%q}`,
			situation, subject, predicate, object,
		)
	default:
		return fmt.Errorf("learn: --type must be 'feedback' or 'fact', got %q", learnType)
	}

	client := apiclient.New(addr, http.DefaultClient)
	resp, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
		From: from,
		To:   "engram-agent",
		Text: text,
	})

	if err != nil {
		return fmt.Errorf("learn: %w", err)
	}

	_, printErr := fmt.Fprintf(stdout, "%d\n", resp.Cursor)

	return printErr
}
```

- [ ] **Step 5: Register in BuildTargets**

```go
		targ.Targ(func(a LearnArgs) { run("learn", LearnFlags(a)) }).
			Name("learn").Description("Record a learning (feedback or fact)"),
```

- [ ] **Step 6: Run test to verify it passes**

Run: `targ test -- -run TestRunLearn_Feedback ./internal/cli/`
Expected: PASS

- [ ] **Step 7: Write test for fact type**

```go
func TestRunLearn_Fact(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var gotBody apiclient.PostMessageRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		decErr := json.NewDecoder(r.Body).Decode(&gotBody)
		g.Expect(decErr).NotTo(HaveOccurred())

		w.WriteHeader(http.StatusOK)
		encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 21})
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.Run(
		[]string{
			"engram", "learn",
			"--from", "lead-1",
			"--type", "fact",
			"--situation", "project architecture",
			"--subject", "engram",
			"--predicate", "uses",
			"--object", "dependency injection for all I/O",
			"--addr", srv.URL,
		},
		&stdout, &stderr, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gotBody.Text).To(ContainSubstring("fact"))
	g.Expect(gotBody.Text).To(ContainSubstring("dependency injection"))
}
```

- [ ] **Step 8: Run test to verify it passes**

Run: `targ test -- -run TestRunLearn_Fact ./internal/cli/`
Expected: PASS

- [ ] **Step 9: Write test for invalid type**

```go
func TestRunLearn_InvalidType(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.Run(
		[]string{
			"engram", "learn",
			"--from", "lead-1",
			"--type", "bogus",
			"--situation", "test",
			"--addr", "http://localhost:1",
		},
		&stdout, &stderr, nil,
	)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("must be 'feedback' or 'fact'"))
}
```

- [ ] **Step 10: Run test to verify it passes**

Run: `targ test -- -run TestRunLearn_InvalidType ./internal/cli/`
Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram learn command (feedback and fact types)

AI-Used: [claude]"
```

---

### Task 8: CLI command — `engram subscribe`

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/cli.go` (add case)
- Modify: `internal/cli/targets.go` (add args, flags, registration)

- [ ] **Step 1: Write the failing test**

```go
func TestRunSubscribe_PrintsMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		g.Expect(r.URL.Query().Get("agent")).To(Equal("lead-1"))

		w.WriteHeader(http.StatusOK)

		if callCount == 1 {
			encErr := json.NewEncoder(w).Encode(apiclient.SubscribeResponse{
				Messages: []apiclient.ChatMessage{
					{From: "engram-agent", To: "lead-1", Text: "memory: use DI"},
				},
				Cursor: 5,
			})
			g.Expect(encErr).NotTo(HaveOccurred())
		} else {
			// Second call: block until context cancelled (simulates waiting).
			<-r.Context().Done()
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	done := make(chan error, 1)
	go func() {
		done <- cli.RunWithContext(
			ctx,
			[]string{
				"engram", "subscribe",
				"--agent", "lead-1",
				"--addr", srv.URL,
			},
			&stdout, &stderr, nil,
		)
	}()

	// Give it time to print the first batch.
	time.Sleep(100 * time.Millisecond)
	cancel()
	err := <-done

	// Context cancellation is not an error.
	if err != nil {
		g.Expect(err.Error()).To(ContainSubstring("context canceled"))
	}

	g.Expect(stdout.String()).To(ContainSubstring("memory: use DI"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunSubscribe_PrintsMessages ./internal/cli/`
Expected: FAIL — RunWithContext not defined, unknown command: subscribe

- [ ] **Step 3: Add RunWithContext to cli.go**

Add a context-accepting variant of Run:

```go
// RunWithContext is like Run but accepts a context for cancellation.
func RunWithContext(
	ctx context.Context,
	args []string,
	stdout, stderr io.Writer,
	stdin io.Reader,
) error {
	if len(args) < minArgs {
		return errUsage
	}

	cmd := args[1]
	subArgs := args[minArgs:]

	switch cmd {
	case "subscribe":
		return runSubscribe(ctx, subArgs, stdout)
	default:
		return Run(args, stdout, stderr, stdin)
	}
}
```

- [ ] **Step 4: Add SubscribeArgs, SubscribeFlags, and implement runSubscribe**

In targets.go:
```go
// SubscribeArgs holds flags for `engram subscribe`.
type SubscribeArgs struct {
	Agent       string `targ:"flag,name=agent,desc=agent name to subscribe as"`
	AfterCursor string `targ:"flag,name=after-cursor,desc=cursor position to start from"`
	Addr        string `targ:"flag,name=addr,desc=API server address"`
}

// SubscribeFlags returns the CLI flag args for the subscribe subcommand.
func SubscribeFlags(a SubscribeArgs) []string {
	return BuildFlags(
		"--agent", a.Agent,
		"--after-cursor", a.AfterCursor,
		"--addr", a.Addr,
	)
}
```

Register in BuildTargets:
```go
		targ.Targ(func(a SubscribeArgs) { run("subscribe", SubscribeFlags(a)) }).
			Name("subscribe").Description("Subscribe to messages for an agent"),
```

In cli_api.go:
```go
func runSubscribe(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("subscribe", flag.ContinueOnError)

	var agent, addr string
	var afterCursor int
	fs.StringVar(&agent, "agent", "", "agent name to subscribe as")
	fs.IntVar(&afterCursor, "after-cursor", 0, "cursor position to start from")
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("subscribe: %w", parseErr)
	}

	client := apiclient.New(addr, http.DefaultClient)
	cursor := afterCursor

	for {
		resp, err := client.Subscribe(ctx, apiclient.SubscribeRequest{
			Agent:       agent,
			AfterCursor: cursor,
		})
		if err != nil {
			return fmt.Errorf("subscribe: %w", err)
		}

		for _, msg := range resp.Messages {
			_, printErr := fmt.Fprintf(stdout, "[%s -> %s] %s\n", msg.From, msg.To, msg.Text)
			if printErr != nil {
				return printErr
			}
		}

		cursor = resp.Cursor
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `targ test -- -run TestRunSubscribe_PrintsMessages ./internal/cli/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram subscribe command (long-poll loop)

AI-Used: [claude]"
```

---

### Task 9: CLI command — `engram status`

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/cli.go` (add case)
- Modify: `internal/cli/targets.go` (add args, flags, registration)

- [ ] **Step 1: Write the failing test**

```go
func TestRunStatus_Success(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		encErr := json.NewEncoder(w).Encode(apiclient.StatusResponse{
			Running: true,
			Agents:  []string{"engram-agent", "lead-1"},
		})
		g.Expect(encErr).NotTo(HaveOccurred())
	}))
	defer srv.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	err := cli.Run(
		[]string{
			"engram", "status",
			"--addr", srv.URL,
		},
		&stdout, &stderr, nil,
	)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(stdout.String()).To(ContainSubstring("running"))
	g.Expect(stdout.String()).To(ContainSubstring("engram-agent"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunStatus_Success ./internal/cli/`
Expected: FAIL — unknown command: status

- [ ] **Step 3: Add StatusArgs, StatusFlags, and implement runStatus**

In targets.go:
```go
// StatusArgs holds flags for `engram status`.
type StatusArgs struct {
	Addr string `targ:"flag,name=addr,desc=API server address"`
}

// StatusFlags returns the CLI flag args for the status subcommand.
func StatusFlags(a StatusArgs) []string {
	return BuildFlags("--addr", a.Addr)
}
```

Register in BuildTargets:
```go
		targ.Targ(func(a StatusArgs) { run("status", StatusFlags(a)) }).
			Name("status").Description("Check engram server status"),
```

Add to cli.go switch:
```go
	case "status":
		return runStatus(subArgs, stdout)
```

In cli_api.go:
```go
func runStatus(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)

	var addr string
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("status: %w", parseErr)
	}

	client := apiclient.New(addr, http.DefaultClient)

	resp, err := client.Status(context.Background())
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(resp)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestRunStatus_Success ./internal/cli/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram status command

AI-Used: [claude]"
```

---

### Task 10: Full suite green + quality check

**Files:**
- No new files

- [ ] **Step 1: Run full test suite**

Run: `targ test`
Expected: All tests pass

- [ ] **Step 2: Run full quality check**

Run: `targ check-full`
Expected: All checks pass (lint, coverage, vet)

- [ ] **Step 3: Fix any issues found**

Address any lint, coverage, or vet issues. Common fixes:
- Add `t.Parallel()` to any subtests missing it
- Add nil guards after gomega assertions
- Fix line length > 120 chars
- Name any magic numbers as constants

- [ ] **Step 4: Re-run quality check**

Run: `targ check-full`
Expected: All checks pass

- [ ] **Step 5: Commit any fixes**

```bash
git add -A
git commit -m "fix: address lint and coverage issues from stage 0

AI-Used: [claude]"
```
