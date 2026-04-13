# Stage 0: CLI Client Commands — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add new CLI commands (`engram post`, `engram intent`, `engram learn`, `engram subscribe`, `engram status`) that are thin HTTP clients to the engram API server.

**Architecture:** A new `internal/apiclient` package provides the HTTP client library. All I/O is injected via the `HTTPDoer` interface — no `http.DefaultClient` or any direct I/O inside `internal/`. CLI command handlers in `internal/cli` are pure functions that accept an `API` interface (the apiclient contract) via parameter — they never construct clients themselves. Thin wiring in `Run()` creates the real `http.DefaultClient` and passes it through. Context flows from `Run()` via `signal.NotifyContext` — never `context.Background()` in handlers. Tests use `httptest.NewServer` for the apiclient HTTP contract tests and `imptest`-generated mocks for CLI handler tests. Property-based tests with `pgregory.net/rapid` verify invariants.

**Tech Stack:** Go stdlib `net/http`, `encoding/json`, `net/http/httptest`, `pgregory.net/rapid`, `imptest` (interactive mocks), targ CLI framework, gomega assertions.

**DI boundary:** `internal/apiclient` defines an `API` interface. `internal/cli` command handlers accept `API` as a parameter. The only place `http.DefaultClient` appears is in the thin wiring in `Run()`/`Targets()`. Tests use imptest-generated mocks of the `API` interface to verify handler behavior without HTTP.

---

### Task 1: Create the API client library — PostMessage

**Files:**
- Create: `internal/apiclient/client.go`
- Create: `internal/apiclient/client_test.go`

- [ ] **Step 1: Write the failing property test — PostMessage always sends correct method, path, and body**

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
	"pgregory.net/rapid"
)

func TestPostMessage_AlwaysSendsCorrectMethodPathAndBody(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "from")
		to := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "to")
		text := rapid.StringMatching(`.{1,200}`).Draw(rt, "text")

		var gotMethod, gotPath string
		var gotBody apiclient.PostMessageRequest

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			decErr := json.NewDecoder(r.Body).Decode(&gotBody)
			g.Expect(decErr).NotTo(HaveOccurred())

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 1})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		_, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
			From: from, To: to, Text: text,
		})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(gotMethod).To(Equal(http.MethodPost))
		g.Expect(gotPath).To(Equal("/message"))
		g.Expect(gotBody.From).To(Equal(from))
		g.Expect(gotBody.To).To(Equal(to))
		g.Expect(gotBody.Text).To(Equal(text))
	})
}

func TestPostMessage_AlwaysReturnsCursorFromServer(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: cursor})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		resp, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
			From: "a", To: "b", Text: "c",
		})
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return
		}

		g.Expect(resp.Cursor).To(Equal(cursor))
	})
}

func TestPostMessage_AlwaysReturnsErrorForNon200(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		// Generate any non-200 status code.
		status := rapid.SampledFrom([]int{
			http.StatusBadRequest, http.StatusUnauthorized,
			http.StatusInternalServerError, http.StatusServiceUnavailable,
		}).Draw(rt, "status")
		errMsg := rapid.StringMatching(`[a-z ]{5,50}`).Draw(rt, "errMsg")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Error: errMsg})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		_, err := client.PostMessage(context.Background(), apiclient.PostMessageRequest{
			From: "a", To: "b", Text: "c",
		})
		g.Expect(err).To(HaveOccurred())
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestPostMessage ./internal/apiclient/`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write minimal implementation to make tests pass**

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
		ctx, http.MethodPost, c.baseURL+"/message", bytes.NewReader(body),
	)
	if reqErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	return c.doJSON(httpReq)
}

// doJSON executes a request and decodes the JSON response.
func (c *Client) doJSON(httpReq *http.Request) (PostMessageResponse, error) {
	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: %w", doErr)
	}
	defer httpResp.Body.Close()

	var resp PostMessageResponse

	decErr := json.NewDecoder(httpResp.Body).Decode(&resp)
	if decErr != nil {
		return PostMessageResponse{}, fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf(
			"apiclient: server returned %d: %s", httpResp.StatusCode, resp.Error,
		)
	}

	return resp, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run TestPostMessage ./internal/apiclient/`
Expected: PASS

- [ ] **Step 5: Refactor — review for DRY, SOLID, simplification**

Check: Is `doJSON` too specific to `PostMessageResponse`? Yes — it should be generic. But YAGNI: we'll refactor when the second method needs it. Leave as-is for now.

- [ ] **Step 6: Commit**

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

- [ ] **Step 1: Write failing property tests**

```go
func TestWaitForResponse_AlwaysSendsCorrectQueryParams(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "from")
		to := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "to")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.Expect(r.Method).To(Equal(http.MethodGet))
			g.Expect(r.URL.Path).To(Equal("/wait-for-response"))
			g.Expect(r.URL.Query().Get("from")).To(Equal(from))
			g.Expect(r.URL.Query().Get("to")).To(Equal(to))
			g.Expect(r.URL.Query().Get("after-cursor")).To(Equal(fmt.Sprintf("%d", cursor)))

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.WaitResponse{Cursor: cursor + 1})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		_, err := client.WaitForResponse(context.Background(), apiclient.WaitRequest{
			From: from, To: to, AfterCursor: cursor,
		})
		g.Expect(err).NotTo(HaveOccurred())
	})
}

func TestWaitForResponse_AlwaysRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // Block until client disconnects.
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := apiclient.New(srv.URL, srv.Client())
	_, err := client.WaitForResponse(ctx, apiclient.WaitRequest{
		From: "a", To: "b", AfterCursor: 0,
	})
	g.Expect(err).To(HaveOccurred())
}

func TestWaitForResponse_AlwaysReturnsFaithfulText(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		text := rapid.StringMatching(`.{1,500}`).Draw(rt, "text")
		cursor := rapid.IntRange(1, 100000).Draw(rt, "cursor")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.WaitResponse{
				Text: text, Cursor: cursor,
			})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		resp, err := client.WaitForResponse(context.Background(), apiclient.WaitRequest{
			From: "a", To: "b", AfterCursor: 0,
		})
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return
		}

		g.Expect(resp.Text).To(Equal(text))
		g.Expect(resp.Cursor).To(Equal(cursor))
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestWaitForResponse ./internal/apiclient/`
Expected: FAIL — WaitForResponse, WaitRequest, WaitResponse not defined

- [ ] **Step 3: Write minimal implementation**

```go
// WaitRequest is the query for GET /wait-for-response.
type WaitRequest struct {
	From        string
	To          string
	AfterCursor int
}

// WaitResponse is the response from GET /wait-for-response.
type WaitResponse struct {
	Text   string `json:"text"`
	Cursor int    `json:"cursor"`
	From   string `json:"from"`
	To     string `json:"to"`
}

// WaitForResponse long-polls for a response message matching the filter.
func (c *Client) WaitForResponse(
	ctx context.Context,
	req WaitRequest,
) (WaitResponse, error) {
	url := fmt.Sprintf(
		"%s/wait-for-response?from=%s&to=%s&after-cursor=%d",
		c.baseURL, req.From, req.To, req.AfterCursor,
	)

	httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return WaitResponse{}, fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return WaitResponse{}, fmt.Errorf("apiclient: waiting for response: %w", doErr)
	}
	defer httpResp.Body.Close()

	var resp WaitResponse

	decErr := json.NewDecoder(httpResp.Body).Decode(&resp)
	if decErr != nil {
		return WaitResponse{}, fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return resp, fmt.Errorf("apiclient: server returned %d", httpResp.StatusCode)
	}

	return resp, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run TestWaitForResponse ./internal/apiclient/`
Expected: PASS

- [ ] **Step 5: Refactor — extract shared GET-with-JSON-decode pattern**

Both `WaitForResponse` and the upcoming `Subscribe`/`Status` methods follow the same pattern: build GET URL, do request, decode JSON. Extract a `doGet` helper:

```go
// doGet executes a GET request and decodes the JSON response into dest.
func (c *Client) doGet(ctx context.Context, url string, dest any) error {
	httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if reqErr != nil {
		return fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return fmt.Errorf("apiclient: %w", doErr)
	}
	defer httpResp.Body.Close()

	decErr := json.NewDecoder(httpResp.Body).Decode(dest)
	if decErr != nil {
		return fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("apiclient: server returned %d", httpResp.StatusCode)
	}

	return nil
}
```

Rewrite `WaitForResponse` to use it. Also refactor `PostMessage` to use a `doPost` helper:

```go
// doPost executes a POST request with JSON body and decodes the response into dest.
func (c *Client) doPost(ctx context.Context, path string, body, dest any) error {
	data, marshalErr := json.Marshal(body)
	if marshalErr != nil {
		return fmt.Errorf("apiclient: marshaling request: %w", marshalErr)
	}

	httpReq, reqErr := http.NewRequestWithContext(
		ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data),
	)
	if reqErr != nil {
		return fmt.Errorf("apiclient: building request: %w", reqErr)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	return c.doAndDecode(httpReq, dest)
}

// doAndDecode executes a request and decodes the JSON response into dest.
func (c *Client) doAndDecode(httpReq *http.Request, dest any) error {
	httpResp, doErr := c.doer.Do(httpReq)
	if doErr != nil {
		return fmt.Errorf("apiclient: %w", doErr)
	}
	defer httpResp.Body.Close()

	decErr := json.NewDecoder(httpResp.Body).Decode(dest)
	if decErr != nil {
		return fmt.Errorf("apiclient: decoding response: %w", decErr)
	}

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("apiclient: server returned %d", httpResp.StatusCode)
	}

	return nil
}
```

- [ ] **Step 6: Run all tests to verify refactoring didn't break anything**

Run: `targ test -- -run "TestPostMessage|TestWaitForResponse" ./internal/apiclient/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/apiclient/client.go internal/apiclient/client_test.go
git commit -m "feat(apiclient): add WaitForResponse, extract shared HTTP helpers

AI-Used: [claude]"
```

---

### Task 3: API client — Subscribe and Status

**Files:**
- Modify: `internal/apiclient/client.go`
- Modify: `internal/apiclient/client_test.go`

- [ ] **Step 1: Write failing property tests for Subscribe**

```go
func TestSubscribe_AlwaysSendsAgentAndCursor(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		agent := rapid.StringMatching(`[a-z][a-z0-9\-]{0,19}`).Draw(rt, "agent")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.Expect(r.URL.Path).To(Equal("/subscribe"))
			g.Expect(r.URL.Query().Get("agent")).To(Equal(agent))
			g.Expect(r.URL.Query().Get("after-cursor")).To(Equal(fmt.Sprintf("%d", cursor)))

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.SubscribeResponse{Cursor: cursor + 1})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		_, err := client.Subscribe(context.Background(), apiclient.SubscribeRequest{
			Agent: agent, AfterCursor: cursor,
		})
		g.Expect(err).NotTo(HaveOccurred())
	})
}

func TestSubscribe_AllMessagesReturnedFaithfully(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		msgCount := rapid.IntRange(0, 5).Draw(rt, "msgCount")
		messages := make([]apiclient.ChatMessage, 0, msgCount)

		for range msgCount {
			messages = append(messages, apiclient.ChatMessage{
				From: rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "from"),
				To:   rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "to"),
				Text: rapid.StringMatching(`.{1,100}`).Draw(rt, "text"),
			})
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.SubscribeResponse{
				Messages: messages, Cursor: 99,
			})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		resp, err := client.Subscribe(context.Background(), apiclient.SubscribeRequest{
			Agent: "test", AfterCursor: 0,
		})
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return
		}

		g.Expect(resp.Messages).To(HaveLen(msgCount))

		for i, msg := range resp.Messages {
			g.Expect(msg.From).To(Equal(messages[i].From))
			g.Expect(msg.To).To(Equal(messages[i].To))
			g.Expect(msg.Text).To(Equal(messages[i].Text))
		}
	})
}
```

- [ ] **Step 2: Write failing property test for Status**

```go
func TestStatus_AllAgentsReturnedFaithfully(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		agentCount := rapid.IntRange(0, 10).Draw(rt, "agentCount")
		agents := make([]string, 0, agentCount)

		for range agentCount {
			agents = append(agents, rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "agent"))
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			g.Expect(r.URL.Path).To(Equal("/status"))

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.StatusResponse{
				Running: true, Agents: agents,
			})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		client := apiclient.New(srv.URL, srv.Client())
		resp, err := client.Status(context.Background())
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return
		}

		g.Expect(resp.Agents).To(Equal(agents))
	})
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `targ test -- -run "TestSubscribe|TestStatus" ./internal/apiclient/`
Expected: FAIL — types and methods not defined

- [ ] **Step 4: Write minimal implementation**

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

	var resp SubscribeResponse
	if err := c.doGet(ctx, url, &resp); err != nil {
		return SubscribeResponse{}, err
	}

	return resp, nil
}

// StatusResponse is the response from GET /status.
type StatusResponse struct {
	Running bool     `json:"running"`
	Agents  []string `json:"agents"`
}

// Status returns the server's health and connected agents.
func (c *Client) Status(ctx context.Context) (StatusResponse, error) {
	var resp StatusResponse
	if err := c.doGet(ctx, c.baseURL+"/status", &resp); err != nil {
		return StatusResponse{}, err
	}

	return resp, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `targ test -- -run "TestSubscribe|TestStatus" ./internal/apiclient/`
Expected: PASS

- [ ] **Step 6: Refactor — review for DRY, SOLID, simplification**

Check: `Subscribe` and `WaitForResponse` both build URLs with `fmt.Sprintf`. Consider if a URL builder is warranted. With only two call sites, it's not — YAGNI.

Check: All response types are standalone structs. Consider if they should share a base. They shouldn't — each endpoint has different semantics.

- [ ] **Step 7: Run all apiclient tests**

Run: `targ test ./internal/apiclient/`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/apiclient/client.go internal/apiclient/client_test.go
git commit -m "feat(apiclient): add Subscribe and Status clients

AI-Used: [claude]"
```

---

### Task 4: API interface + imptest mock generation

The CLI handlers must not construct HTTP clients or do any I/O. They accept an `apiclient.API` interface and write to an `io.Writer`. The `Run()` function in `cli.go` is the thin wiring layer that constructs real clients.

**Files:**
- Modify: `internal/apiclient/client.go` (add `API` interface)
- Create: `internal/cli/cli_api_test.go` (imptest generate directive)
- Create: `internal/cli/export_api_test.go` (test export helpers)

- [ ] **Step 1: Add API interface to apiclient package**

```go
// In internal/apiclient/client.go, add after the type definitions:

// API is the contract for engram API operations. CLI handlers accept this
// interface — they never construct HTTP clients. Satisfied by *Client.
type API interface {
	PostMessage(ctx context.Context, req PostMessageRequest) (PostMessageResponse, error)
	WaitForResponse(ctx context.Context, req WaitRequest) (WaitResponse, error)
	Subscribe(ctx context.Context, req SubscribeRequest) (SubscribeResponse, error)
	Status(ctx context.Context) (StatusResponse, error)
}
```

- [ ] **Step 2: Add imptest generate directive and run generation**

Create `internal/cli/cli_api_test.go`:
```go
package cli_test

//go:generate impgen apiclient.API --dependency
```

Run: `go generate ./internal/cli/...`
Expected: generates `generated_MockAPI_test.go` in `internal/cli/`

- [ ] **Step 3: Commit**

```bash
git add internal/apiclient/client.go internal/cli/cli_api_test.go internal/cli/generated_MockAPI_test.go
git commit -m "feat(apiclient): add API interface, generate imptest mock

AI-Used: [claude]"
```

---

### Task 5: Pure handler `doPost` + imptest test + thin wiring

Handler is a pure function: `doPost(ctx, api, from, to, text, stdout) error`. No flag parsing, no client construction, no I/O except writing to the injected `io.Writer`. The thin wiring function `runPost` parses flags, constructs the real client, and calls `doPost`.

**Files:**
- Create: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Create: `internal/cli/export_api_test.go`
- Modify: `internal/cli/cli.go` (add case + context)
- Modify: `internal/cli/targets.go` (PostArgs, PostFlags, registration)

- [ ] **Step 1: Write failing property test — doPost always passes from/to/text to API**

```go
// In internal/cli/cli_api_test.go
package cli_test

import (
	"bytes"
	"fmt"
	"testing"

	"engram/internal/apiclient"
	"engram/internal/cli"

	. "github.com/onsi/gomega"
	. "github.com/toejough/imptest/match"
	"pgregory.net/rapid"
)

//go:generate impgen apiclient.API --dependency

func TestDoPost_AlwaysPassesFromToTextToAPI(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		to := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "to")
		text := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,80}`).Draw(rt, "text")
		cursor := rapid.IntRange(0, 100000).Draw(rt, "cursor")

		mock, imp := MockAPI(rt)
		var stdout bytes.Buffer

		call := StartExportDoPost(rt, cli.ExportDoPost, rt.Context(), mock, from, to, text, &stdout)

		imp.PostMessage.ArgsShould(
			BeAny,
			Equal(apiclient.PostMessageRequest{From: from, To: to, Text: text}),
		).Return(apiclient.PostMessageResponse{Cursor: cursor}, nil)

		call.ReturnsShould(BeNil())

		g.Expect(stdout.String()).To(Equal(fmt.Sprintf("%d\n", cursor)))
	})
}
```

Also in `internal/cli/cli_api_test.go`, add the wrapper generate:
```go
//go:generate impgen cli.ExportDoPost --target
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go generate ./internal/cli/... && targ test -- -run TestDoPost ./internal/cli/`
Expected: FAIL — ExportDoPost not defined

- [ ] **Step 3: Implement doPost in cli_api.go and export for testing**

In `internal/cli/cli_api.go`:
```go
package cli

import (
	"context"
	"fmt"
	"io"

	"engram/internal/apiclient"
)

const defaultAPIAddr = "http://localhost:7932"

// doPost posts a message via the API and prints the cursor.
// Pure function — no I/O construction. Accepts API interface.
func doPost(
	ctx context.Context,
	api apiclient.API,
	from, to, text string,
	stdout io.Writer,
) error {
	resp, err := api.PostMessage(ctx, apiclient.PostMessageRequest{
		From: from, To: to, Text: text,
	})
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}

	_, printErr := fmt.Fprintf(stdout, "%d\n", resp.Cursor)

	return printErr
}
```

In `internal/cli/export_api_test.go`:
```go
package cli

import (
	"context"
	"io"

	"engram/internal/apiclient"
)

// ExportDoPost exposes doPost for testing.
var ExportDoPost = doPost
```

- [ ] **Step 4: Add thin wiring: runPost, context flow, targ registration**

In `internal/cli/cli_api.go`, add the thin wiring wrapper:
```go
import (
	"flag"
	"net/http"
)

// runPost is the thin wiring layer: parses flags, constructs real client, calls doPost.
func runPost(ctx context.Context, args []string, stdout io.Writer) error {
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

	return doPost(ctx, client, from, to, text, stdout)
}
```

In `cli.go` add context + dispatch:
```go
	case "post", "intent", "learn", "status":
		apiCtx, apiStop := signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)
		defer apiStop()

		return runAPIDispatch(apiCtx, cmd, subArgs, stdout)
```

In `cli_api.go` add the dispatch:
```go
func runAPIDispatch(ctx context.Context, cmd string, args []string, stdout io.Writer) error {
	switch cmd {
	case "post":
		return runPost(ctx, args, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}
```

In `targets.go` add:
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
	return BuildFlags("--from", a.From, "--to", a.To, "--text", a.Text, "--addr", a.Addr)
}
```

Register in `BuildTargets`:
```go
		targ.Targ(func(a PostArgs) { run("post", PostFlags(a)) }).
			Name("post").Description("Post a message to the engram chat"),
```

- [ ] **Step 5: Re-generate imptest wrappers and run tests**

Run: `go generate ./internal/cli/... && targ test -- -run TestDoPost ./internal/cli/`
Expected: PASS

- [ ] **Step 6: Refactor — review for DRY, SOLID**

Check: `doPost` is a pure function with single responsibility. `runPost` is thin wiring only. Clean separation.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/export_api_test.go internal/cli/generated_* internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram post with DI (pure handler + thin wiring)

AI-Used: [claude]"
```

---

### Task 6: Pure handler `doIntent` + imptest test + thin wiring

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/export_api_test.go`
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Write failing property test — doIntent always posts then waits, returns memory text**

```go
//go:generate impgen cli.ExportDoIntent --target

func TestDoIntent_AlwaysPostsThenWaitsAndReturnsSurfacedMemory(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "from")
		to := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "to")
		situation := rapid.StringMatching(`[A-Za-z0-9 ]{5,50}`).Draw(rt, "situation")
		action := rapid.StringMatching(`[A-Za-z0-9 ]{5,50}`).Draw(rt, "action")
		memory := rapid.StringMatching(`[A-Za-z0-9 ]{5,80}`).Draw(rt, "memory")
		cursor := rapid.IntRange(1, 100000).Draw(rt, "cursor")

		mock, imp := MockAPI(rt)
		var stdout bytes.Buffer

		call := StartExportDoIntent(
			rt, cli.ExportDoIntent,
			rt.Context(), mock, from, to, situation, action, &stdout,
		)

		// Expect post first.
		imp.PostMessage.ArgsShould(BeAny, BeAny).
			Return(apiclient.PostMessageResponse{Cursor: cursor}, nil)

		// Then expect wait with correct from/to reversal and cursor.
		imp.WaitForResponse.ArgsShould(
			BeAny,
			Equal(apiclient.WaitRequest{From: to, To: from, AfterCursor: cursor}),
		).Return(apiclient.WaitResponse{Text: memory, Cursor: cursor + 1}, nil)

		call.ReturnsShould(BeNil())

		g.Expect(stdout.String()).To(ContainSubstring(memory))
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go generate ./internal/cli/... && targ test -- -run TestDoIntent ./internal/cli/`
Expected: FAIL — ExportDoIntent not defined

- [ ] **Step 3: Implement doIntent, export, thin wiring, targ registration**

In `cli_api.go`:
```go
// doIntent posts an intent and blocks until the engram-agent responds.
// Pure function — no I/O construction.
func doIntent(
	ctx context.Context,
	api apiclient.API,
	from, to, situation, plannedAction string,
	stdout io.Writer,
) error {
	text := fmt.Sprintf("INTENT: situation=%s planned_action=%s", situation, plannedAction)

	postResp, postErr := api.PostMessage(ctx, apiclient.PostMessageRequest{
		From: from, To: to, Text: text,
	})
	if postErr != nil {
		return fmt.Errorf("intent: posting: %w", postErr)
	}

	waitResp, waitErr := api.WaitForResponse(ctx, apiclient.WaitRequest{
		From: to, To: from, AfterCursor: postResp.Cursor,
	})
	if waitErr != nil {
		return fmt.Errorf("intent: waiting: %w", waitErr)
	}

	_, printErr := fmt.Fprintln(stdout, waitResp.Text)

	return printErr
}

// runIntent is the thin wiring layer.
func runIntent(ctx context.Context, args []string, stdout io.Writer) error {
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

	return doIntent(ctx, client, from, to, situation, plannedAction, stdout)
}
```

Add to `runAPIDispatch`:
```go
	case "intent":
		return runIntent(ctx, args, stdout)
```

In `export_api_test.go`:
```go
// ExportDoIntent exposes doIntent for testing.
var ExportDoIntent = doIntent
```

In `targets.go`:
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
		"--from", a.From, "--to", a.To,
		"--situation", a.Situation, "--planned-action", a.PlannedAction,
		"--addr", a.Addr,
	)
}
```

Register:
```go
		targ.Targ(func(a IntentArgs) { run("intent", IntentFlags(a)) }).
			Name("intent").Description("Announce intent and wait for surfaced memories"),
```

- [ ] **Step 4: Re-generate and run tests**

Run: `go generate ./internal/cli/... && targ test -- -run TestDoIntent ./internal/cli/`
Expected: PASS

- [ ] **Step 5: Refactor — review for DRY between doPost and doIntent**

Both accept `(ctx, api, ..., stdout)`. The pattern is consistent. No extraction needed yet — each has different logic.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/export_api_test.go internal/cli/generated_* internal/cli/targets.go
git commit -m "feat(cli): add engram intent with DI (post + wait two-step)

AI-Used: [claude]"
```

---

### Task 7: Pure handler `doLearn` + imptest test + thin wiring

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/export_api_test.go`
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Write failing property tests — learn always includes correct structured fields**

```go
//go:generate impgen cli.ExportDoLearn --target

func TestDoLearn_FeedbackAlwaysIncludesAllFields(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		situation := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "situation")
		behavior := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "behavior")
		impact := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "impact")
		action := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "action")

		mock, imp := MockAPI(rt)
		var stdout bytes.Buffer

		call := StartExportDoLearn(
			rt, cli.ExportDoLearn,
			rt.Context(), mock, "lead-1", "feedback",
			situation, behavior, impact, action, "", "", "",
			&stdout,
		)

		// Capture the text sent to PostMessage.
		args := imp.PostMessage.ArgsShould(BeAny, BeAny).GetArgs()
		req := args[1].(apiclient.PostMessageRequest)

		g.Expect(req.Text).To(ContainSubstring(situation))
		g.Expect(req.Text).To(ContainSubstring(behavior))
		g.Expect(req.Text).To(ContainSubstring(impact))
		g.Expect(req.Text).To(ContainSubstring(action))
		g.Expect(req.Text).To(ContainSubstring("feedback"))
		g.Expect(req.From).To(Equal("lead-1"))
		g.Expect(req.To).To(Equal("engram-agent"))

		imp.PostMessage.Return(apiclient.PostMessageResponse{Cursor: 1}, nil)
		call.ReturnsShould(BeNil())
	})
}

func TestDoLearn_InvalidTypeAlwaysErrors(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		badType := rapid.StringMatching(`[a-z]{1,10}`).
			Filter(func(s string) bool { return s != "feedback" && s != "fact" }).
			Draw(rt, "badType")

		mock, _ := MockAPI(rt)
		var stdout bytes.Buffer

		call := StartExportDoLearn(
			rt, cli.ExportDoLearn,
			rt.Context(), mock, "lead-1", badType,
			"sit", "", "", "", "", "", "",
			&stdout,
		)

		call.ReturnsShould(HaveOccurred())
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go generate ./internal/cli/... && targ test -- -run TestDoLearn ./internal/cli/`
Expected: FAIL — ExportDoLearn not defined

- [ ] **Step 3: Implement doLearn, buildLearnText, export, thin wiring**

In `cli_api.go`:
```go
import "errors"

var errLearnTypeMustBeFeedbackOrFact = errors.New("learn: --type must be 'feedback' or 'fact'")

// doLearn posts a structured learning message via the API.
// Pure function — no I/O construction.
func doLearn(
	ctx context.Context,
	api apiclient.API,
	from, learnType, situation,
	behavior, impact, action,
	subject, predicate, object string,
	stdout io.Writer,
) error {
	text, buildErr := buildLearnText(
		learnType, situation, behavior, impact, action, subject, predicate, object,
	)
	if buildErr != nil {
		return buildErr
	}

	resp, err := api.PostMessage(ctx, apiclient.PostMessageRequest{
		From: from, To: "engram-agent", Text: text,
	})
	if err != nil {
		return fmt.Errorf("learn: %w", err)
	}

	_, printErr := fmt.Fprintf(stdout, "%d\n", resp.Cursor)

	return printErr
}

func buildLearnText(
	learnType, situation, behavior, impact, action, subject, predicate, object string,
) (string, error) {
	switch learnType {
	case "feedback":
		text, marshalErr := json.Marshal(map[string]string{
			"type": "feedback", "situation": situation,
			"behavior": behavior, "impact": impact, "action": action,
		})
		if marshalErr != nil {
			return "", fmt.Errorf("learn: %w", marshalErr)
		}

		return string(text), nil
	case "fact":
		text, marshalErr := json.Marshal(map[string]string{
			"type": "fact", "situation": situation,
			"subject": subject, "predicate": predicate, "object": object,
		})
		if marshalErr != nil {
			return "", fmt.Errorf("learn: %w", marshalErr)
		}

		return string(text), nil
	default:
		return "", fmt.Errorf("%w, got %q", errLearnTypeMustBeFeedbackOrFact, learnType)
	}
}

// runLearn is the thin wiring layer.
func runLearn(ctx context.Context, args []string, stdout io.Writer) error {
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

	client := apiclient.New(addr, http.DefaultClient)

	return doLearn(ctx, client, from, learnType, situation,
		behavior, impact, action, subject, predicate, object, stdout)
}
```

Add to `runAPIDispatch`:
```go
	case "learn":
		return runLearn(ctx, args, stdout)
```

In `export_api_test.go`:
```go
// ExportDoLearn exposes doLearn for testing.
var ExportDoLearn = doLearn
```

In `targets.go` add LearnArgs, LearnFlags, register in BuildTargets (same as before — see full LearnArgs struct in the spec).

- [ ] **Step 4: Re-generate and run tests**

Run: `go generate ./internal/cli/... && targ test -- -run TestDoLearn ./internal/cli/`
Expected: PASS

- [ ] **Step 5: Refactor — extract shared patterns across doPost, doIntent, doLearn**

All three accept `(ctx, api, ..., stdout)` and call `api.PostMessage`. The `doPost` and `doLearn` both just print the cursor. Consider extracting a `postAndPrintCursor` helper:

```go
func postAndPrintCursor(
	ctx context.Context, api apiclient.API,
	req apiclient.PostMessageRequest, stdout io.Writer,
) error {
	resp, err := api.PostMessage(ctx, req)
	if err != nil {
		return err
	}
	_, printErr := fmt.Fprintf(stdout, "%d\n", resp.Cursor)
	return printErr
}
```

Only extract if it genuinely reduces duplication. If it makes the code harder to follow, skip it.

- [ ] **Step 6: Run all tests**

Run: `go generate ./internal/cli/... && targ test ./internal/cli/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/export_api_test.go internal/cli/generated_* internal/cli/targets.go
git commit -m "feat(cli): add engram learn with DI (feedback and fact types)

AI-Used: [claude]"
```

---

### Task 8: Pure handlers `doSubscribe` + `doStatus` + imptest tests + thin wiring

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/export_api_test.go`
- Modify: `internal/cli/cli.go` (add RunWithContext + subscribe context)
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Write failing property test — doStatus always prints all agents**

```go
//go:generate impgen cli.ExportDoStatus --target
//go:generate impgen cli.ExportDoSubscribe --target

func TestDoStatus_AlwaysPrintsAllAgents(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		agentCount := rapid.IntRange(0, 5).Draw(rt, "agentCount")
		agents := make([]string, 0, agentCount)

		for range agentCount {
			agents = append(agents, rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "agent"))
		}

		mock, imp := MockAPI(rt)
		var stdout bytes.Buffer

		call := StartExportDoStatus(rt, cli.ExportDoStatus, rt.Context(), mock, &stdout)

		imp.Status.ArgsShould(BeAny).
			Return(apiclient.StatusResponse{Running: true, Agents: agents}, nil)

		call.ReturnsShould(BeNil())

		for _, agent := range agents {
			g.Expect(stdout.String()).To(ContainSubstring(agent))
		}
	})
}
```

- [ ] **Step 2: Write failing test — doSubscribe always prints messages and advances cursor**

```go
func TestDoSubscribe_AlwaysPrintsMessagesAndAdvancesCursor(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mock, imp := MockAPI(t)
	var stdout bytes.Buffer

	ctx, cancel := context.WithCancel(t.Context())

	go cli.ExportDoSubscribe(ctx, mock, "lead-1", 0, &stdout)

	// First subscribe call: return one message.
	imp.Subscribe.ArgsShould(
		BeAny,
		Equal(apiclient.SubscribeRequest{Agent: "lead-1", AfterCursor: 0}),
	).Return(apiclient.SubscribeResponse{
		Messages: []apiclient.ChatMessage{
			{From: "engram-agent", To: "lead-1", Text: "use DI"},
		},
		Cursor: 5,
	}, nil)

	// Second subscribe call: should use cursor=5. Block until cancelled.
	imp.Subscribe.ArgsShould(
		BeAny,
		Equal(apiclient.SubscribeRequest{Agent: "lead-1", AfterCursor: 5}),
	).Return(apiclient.SubscribeResponse{}, context.Canceled)

	g.Eventually(func() string { return stdout.String() }).
		Should(ContainSubstring("use DI"))

	cancel()
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go generate ./internal/cli/... && targ test -- -run "TestDoStatus|TestDoSubscribe" ./internal/cli/`
Expected: FAIL — exports not defined

- [ ] **Step 4: Implement doSubscribe, doStatus, exports, thin wiring**

In `cli_api.go`:
```go
// doSubscribe long-polls for messages and prints them. Runs until ctx cancelled.
// Pure function — no I/O construction.
func doSubscribe(
	ctx context.Context,
	api apiclient.API,
	agent string,
	afterCursor int,
	stdout io.Writer,
) error {
	cursor := afterCursor

	for {
		resp, err := api.Subscribe(ctx, apiclient.SubscribeRequest{
			Agent: agent, AfterCursor: cursor,
		})
		if err != nil {
			return fmt.Errorf("subscribe: %w", err)
		}

		for _, msg := range resp.Messages {
			if _, printErr := fmt.Fprintf(
				stdout, "[%s -> %s] %s\n", msg.From, msg.To, msg.Text,
			); printErr != nil {
				return printErr
			}
		}

		cursor = resp.Cursor
	}
}

// doStatus prints server health as JSON. Pure function.
func doStatus(
	ctx context.Context,
	api apiclient.API,
	stdout io.Writer,
) error {
	resp, err := api.Status(ctx)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(resp)
}

// runSubscribe is the thin wiring layer.
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

	return doSubscribe(ctx, client, agent, afterCursor, stdout)
}

// runStatus is the thin wiring layer.
func runStatus(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)

	var addr string
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("status: %w", parseErr)
	}

	client := apiclient.New(addr, http.DefaultClient)

	return doStatus(ctx, client, stdout)
}
```

Add `"status"` to `runAPIDispatch`. Add `subscribe` case with context in `cli.go`:
```go
	case "subscribe":
		subCtx, subStop := signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)
		defer subStop()

		return runSubscribe(subCtx, subArgs, stdout)
```

Also add `RunWithContext` for tests that need to pass context directly:
```go
// RunWithContext is like Run but accepts a context for long-running commands.
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

In `export_api_test.go`:
```go
// ExportDoSubscribe exposes doSubscribe for testing.
var ExportDoSubscribe = doSubscribe

// ExportDoStatus exposes doStatus for testing.
var ExportDoStatus = doStatus
```

In `targets.go` add SubscribeArgs, SubscribeFlags, StatusArgs, StatusFlags and register in BuildTargets.

- [ ] **Step 5: Re-generate and run tests**

Run: `go generate ./internal/cli/... && targ test -- -run "TestDoStatus|TestDoSubscribe" ./internal/cli/`
Expected: PASS

- [ ] **Step 6: Refactor — deduplicate across all 5 thin wiring functions**

All `runXxx` functions follow the same pattern: parse flags → construct client → call pure handler. Review whether a shared `newClientFromFlags` or `withClient` helper reduces duplication. Five call sites with the same `apiclient.New(addr, http.DefaultClient)` pattern — extract if it's more than two lines repeated.

- [ ] **Step 7: Run full test suite**

Run: `targ test ./internal/cli/ ./internal/apiclient/`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/export_api_test.go internal/cli/generated_* internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram subscribe and status with DI

AI-Used: [claude]"
```

---

### Task 9: Full suite green + quality check

**Files:**
- No new files

- [ ] **Step 1: Run full test suite**

Run: `targ test`
Expected: All tests pass

- [ ] **Step 2: Run full quality check**

Run: `targ check-full`
Expected: All checks pass (lint, coverage, vet)

- [ ] **Step 3: Fix any issues found**

Common fixes:
- Add `t.Parallel()` to any subtests missing it
- Add nil guards after gomega assertions (`if err != nil { return }`)
- Fix line length > 120 chars
- Name magic numbers as constants (`defaultAPIAddr` already done)
- Wrap errors with context: `fmt.Errorf("context: %w", err)` not bare `return err`
- Use `http.NewRequestWithContext` not `http.NewRequest`
- Ensure `generated_*` files are committed

- [ ] **Step 4: Re-run quality check**

Run: `targ check-full`
Expected: All checks pass

- [ ] **Step 5: Final refactor pass across all new code**

Review all new files for:
- **DRY:** Are the thin wiring functions (`runPost`, `runIntent`, `runLearn`, `runSubscribe`, `runStatus`) consistent? Can the `--addr` + client construction be shared?
- **SOLID:** Each `doXxx` has single responsibility? Each `runXxx` is pure wiring?
- **Naming:** Descriptive names? Consistent conventions?
- **Error wrapping:** Every error wrapped with context?
- **No I/O in `internal/`:** Only the `runXxx` wiring functions touch `http.DefaultClient`. All `doXxx` functions accept `apiclient.API`.

- [ ] **Step 6: Run full quality check after refactoring**

Run: `targ check-full`
Expected: All checks pass

- [ ] **Step 7: Commit any fixes**

```bash
git add -A
git commit -m "fix: address lint, coverage, and style issues from stage 0

AI-Used: [claude]"
```
