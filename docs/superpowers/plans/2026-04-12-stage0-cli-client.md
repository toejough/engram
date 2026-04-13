# Stage 0: CLI Client Commands — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add new CLI commands (`engram post`, `engram intent`, `engram learn`, `engram subscribe`, `engram status`) that are thin HTTP clients to the engram API server.

**Architecture:** A new `internal/apiclient` package provides the HTTP client library with DI for testing. New targ commands in `internal/cli` wrap it. Context flows from `Run()` via `signal.NotifyContext` to all command handlers — never `context.Background()` in handlers. The API server doesn't exist yet — tests use `httptest.NewServer` fakes. Existing commands are untouched. Property-based tests with `pgregory.net/rapid` verify invariants for all generated inputs.

**Tech Stack:** Go stdlib `net/http`, `encoding/json`, `net/http/httptest`, `pgregory.net/rapid`, targ CLI framework, gomega assertions.

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

### Task 4: CLI wiring — context flow and `engram post`

**Files:**
- Create: `internal/cli/cli_api.go`
- Create: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/cli.go` (add cases to Run, add context threading)
- Modify: `internal/cli/targets.go` (add args struct, flags func, targ registration)

- [ ] **Step 1: Write failing property test — post always sends from/to/text faithfully**

```go
// In cli_api_test.go
package cli_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"engram/internal/apiclient"
	"engram/internal/cli"

	. "github.com/onsi/gomega"
	"pgregory.net/rapid"
)

func TestRunPost_AlwaysSendsFromToTextFaithfully(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "from")
		to := rapid.StringMatching(`[a-z][a-z0-9\-]{1,15}`).Draw(rt, "to")
		text := rapid.StringMatching(`[A-Za-z0-9 .,!]{1,80}`).Draw(rt, "text")

		var gotBody apiclient.PostMessageRequest

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decErr := json.NewDecoder(r.Body).Decode(&gotBody)
			g.Expect(decErr).NotTo(HaveOccurred())

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 1})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		var stdout, stderr bytes.Buffer
		err := cli.Run(
			[]string{"engram", "post", "--from", from, "--to", to, "--text", text, "--addr", srv.URL},
			&stdout, &stderr, nil,
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(gotBody.From).To(Equal(from))
		g.Expect(gotBody.To).To(Equal(to))
		g.Expect(gotBody.Text).To(Equal(text))
	})
}

func TestRunPost_AlwaysPrintsCursorOnSuccess(t *testing.T) {
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

		var stdout, stderr bytes.Buffer
		err := cli.Run(
			[]string{"engram", "post", "--from", "a", "--to", "b", "--text", "c", "--addr", srv.URL},
			&stdout, &stderr, nil,
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(stdout.String()).To(Equal(fmt.Sprintf("%d\n", cursor)))
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestRunPost ./internal/cli/`
Expected: FAIL — unknown command: post

- [ ] **Step 3: Add PostArgs and PostFlags to targets.go**

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

- [ ] **Step 4: Thread context through Run for API commands**

Modify `Run()` in `cli.go` to create context for API commands and pass it:

```go
	case "post", "intent", "learn", "status":
		apiCtx, apiStop := signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)
		defer apiStop()

		return runAPIDispatch(apiCtx, cmd, subArgs, stdout)
	case "subscribe":
		subCtx, subStop := signal.NotifyContext(
			context.Background(),
			os.Interrupt,
			syscall.SIGTERM,
		)
		defer subStop()

		return runSubscribe(subCtx, subArgs, stdout)
```

- [ ] **Step 5: Implement runAPIDispatch and runPost in cli_api.go**

```go
package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"

	"engram/internal/apiclient"
)

const defaultAPIAddr = "http://localhost:7932"

func runAPIDispatch(ctx context.Context, cmd string, args []string, stdout io.Writer) error {
	switch cmd {
	case "post":
		return runPost(ctx, args, stdout)
	default:
		return fmt.Errorf("%w: %s", errUnknownCommand, cmd)
	}
}

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

	resp, err := client.PostMessage(ctx, apiclient.PostMessageRequest{
		From: from, To: to, Text: text,
	})
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}

	_, printErr := fmt.Fprintf(stdout, "%d\n", resp.Cursor)

	return printErr
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `targ test -- -run TestRunPost ./internal/cli/`
Expected: PASS

- [ ] **Step 7: Refactor — review for DRY, naming, SOLID**

Check: `runAPIDispatch` switch is clean. `runPost` follows the same flag-parse → client-call → print pattern the other commands will use. The flag parsing could become a helper, but with only one command so far, YAGNI.

- [ ] **Step 8: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram post command with context flow from Run

AI-Used: [claude]"
```

---

### Task 5: CLI command — `engram intent`

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Write failing property test — intent always does two-step post-then-wait**

```go
func TestRunIntent_AlwaysPostsThenWaits(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		from := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "from")
		to := rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "to")
		situation := rapid.StringMatching(`[A-Za-z0-9 ]{5,50}`).Draw(rt, "situation")
		action := rapid.StringMatching(`[A-Za-z0-9 ]{5,50}`).Draw(rt, "action")
		memory := rapid.StringMatching(`[A-Za-z0-9 ]{5,80}`).Draw(rt, "memory")

		var calls []string

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls = append(calls, r.Method+" "+r.URL.Path)

			w.WriteHeader(http.StatusOK)

			switch {
			case r.Method == http.MethodPost:
				encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 10})
				g.Expect(encErr).NotTo(HaveOccurred())
			case r.Method == http.MethodGet:
				encErr := json.NewEncoder(w).Encode(apiclient.WaitResponse{Text: memory, Cursor: 12})
				g.Expect(encErr).NotTo(HaveOccurred())
			}
		}))
		defer srv.Close()

		var stdout, stderr bytes.Buffer
		err := cli.Run(
			[]string{
				"engram", "intent",
				"--from", from, "--to", to,
				"--situation", situation, "--planned-action", action,
				"--addr", srv.URL,
			},
			&stdout, &stderr, nil,
		)
		g.Expect(err).NotTo(HaveOccurred())

		// Invariant: always post first, then wait.
		g.Expect(calls).To(HaveLen(2))
		g.Expect(calls[0]).To(Equal("POST /message"))
		g.Expect(calls[1]).To(HavePrefix("GET /wait-for-response"))

		// Invariant: output always contains the surfaced memory text.
		g.Expect(stdout.String()).To(ContainSubstring(memory))
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestRunIntent ./internal/cli/`
Expected: FAIL — unknown command: intent

- [ ] **Step 3: Add IntentArgs/IntentFlags to targets.go, implement runIntent**

In targets.go:
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

Add to `runAPIDispatch`:
```go
	case "intent":
		return runIntent(ctx, args, stdout)
```

In cli_api.go:
```go
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

	text := fmt.Sprintf("INTENT: situation=%s planned_action=%s", situation, plannedAction)

	postResp, postErr := client.PostMessage(ctx, apiclient.PostMessageRequest{
		From: from, To: to, Text: text,
	})
	if postErr != nil {
		return fmt.Errorf("intent: posting: %w", postErr)
	}

	waitResp, waitErr := client.WaitForResponse(ctx, apiclient.WaitRequest{
		From: to, To: from, AfterCursor: postResp.Cursor,
	})
	if waitErr != nil {
		return fmt.Errorf("intent: waiting: %w", waitErr)
	}

	_, printErr := fmt.Fprintln(stdout, waitResp.Text)

	return printErr
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestRunIntent ./internal/cli/`
Expected: PASS

- [ ] **Step 5: Refactor — review for DRY with runPost**

Both `runPost` and `runIntent` parse `--from`, `--to`, `--addr` flags. Extract a shared `apiFlags` struct and `parseAPIFlags` helper if this pattern repeats with `runLearn` too. Wait until Task 6 to confirm the pattern before extracting.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/targets.go
git commit -m "feat(cli): add engram intent command (post + wait two-step)

AI-Used: [claude]"
```

---

### Task 6: CLI command — `engram learn`

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Write failing property tests**

```go
func TestRunLearn_FeedbackAlwaysIncludesAllFields(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		situation := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "situation")
		behavior := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "behavior")
		impact := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "impact")
		action := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "action")

		var gotBody apiclient.PostMessageRequest

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decErr := json.NewDecoder(r.Body).Decode(&gotBody)
			g.Expect(decErr).NotTo(HaveOccurred())

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 1})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		var stdout, stderr bytes.Buffer
		err := cli.Run(
			[]string{
				"engram", "learn", "--from", "lead-1", "--type", "feedback",
				"--situation", situation, "--behavior", behavior,
				"--impact", impact, "--action", action,
				"--addr", srv.URL,
			},
			&stdout, &stderr, nil,
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(gotBody.Text).To(ContainSubstring(situation))
		g.Expect(gotBody.Text).To(ContainSubstring(behavior))
		g.Expect(gotBody.Text).To(ContainSubstring(impact))
		g.Expect(gotBody.Text).To(ContainSubstring(action))
		g.Expect(gotBody.Text).To(ContainSubstring("feedback"))
	})
}

func TestRunLearn_FactAlwaysIncludesAllFields(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		situation := rapid.StringMatching(`[A-Za-z0-9 ]{5,40}`).Draw(rt, "situation")
		subject := rapid.StringMatching(`[A-Za-z0-9 ]{3,20}`).Draw(rt, "subject")
		predicate := rapid.StringMatching(`[A-Za-z0-9 ]{3,20}`).Draw(rt, "predicate")
		object := rapid.StringMatching(`[A-Za-z0-9 ]{3,30}`).Draw(rt, "object")

		var gotBody apiclient.PostMessageRequest

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			decErr := json.NewDecoder(r.Body).Decode(&gotBody)
			g.Expect(decErr).NotTo(HaveOccurred())

			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.PostMessageResponse{Cursor: 1})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		var stdout, stderr bytes.Buffer
		err := cli.Run(
			[]string{
				"engram", "learn", "--from", "lead-1", "--type", "fact",
				"--situation", situation, "--subject", subject,
				"--predicate", predicate, "--object", object,
				"--addr", srv.URL,
			},
			&stdout, &stderr, nil,
		)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(gotBody.Text).To(ContainSubstring(situation))
		g.Expect(gotBody.Text).To(ContainSubstring(subject))
		g.Expect(gotBody.Text).To(ContainSubstring(predicate))
		g.Expect(gotBody.Text).To(ContainSubstring(object))
		g.Expect(gotBody.Text).To(ContainSubstring("fact"))
	})
}

func TestRunLearn_InvalidTypeAlwaysErrors(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		// Generate any type string that is NOT "feedback" or "fact".
		badType := rapid.StringMatching(`[a-z]{1,10}`).
			Filter(func(s string) bool { return s != "feedback" && s != "fact" }).
			Draw(rt, "badType")

		var stdout, stderr bytes.Buffer
		err := cli.Run(
			[]string{
				"engram", "learn", "--from", "a", "--type", badType,
				"--situation", "x", "--addr", "http://localhost:1",
			},
			&stdout, &stderr, nil,
		)
		g.Expect(err).To(HaveOccurred())
		g.Expect(err.Error()).To(ContainSubstring("must be 'feedback' or 'fact'"))
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run TestRunLearn ./internal/cli/`
Expected: FAIL — unknown command: learn

- [ ] **Step 3: Add LearnArgs/LearnFlags, implement runLearn**

In targets.go:
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
		"--from", a.From, "--type", a.Type, "--situation", a.Situation,
		"--behavior", a.Behavior, "--impact", a.Impact, "--action", a.Action,
		"--subject", a.Subject, "--predicate", a.Predicate, "--object", a.Object,
		"--addr", a.Addr,
	)
}
```

Register and add to `runAPIDispatch`:
```go
	case "learn":
		return runLearn(ctx, args, stdout)
```

In cli_api.go:
```go
var errLearnTypeMustBeFeedbackOrFact = errors.New("learn: --type must be 'feedback' or 'fact'")

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

	text, buildErr := buildLearnText(learnType, situation, behavior, impact, action, subject, predicate, object)
	if buildErr != nil {
		return buildErr
	}

	client := apiclient.New(addr, http.DefaultClient)

	resp, err := client.PostMessage(ctx, apiclient.PostMessageRequest{
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run TestRunLearn ./internal/cli/`
Expected: PASS

- [ ] **Step 5: Refactor — extract shared flag parsing pattern**

All three commands (`runPost`, `runIntent`, `runLearn`) parse `--addr` and create a client. Extract:

```go
func parseAddr(fs *flag.FlagSet) *string {
	addr := fs.String("addr", defaultAPIAddr, "API server address")
	return addr
}

func newClientFromAddr(addr string) *apiclient.Client {
	return apiclient.New(addr, http.DefaultClient)
}
```

Review if this is worth it — three call sites is borderline. If it reduces duplication meaningfully, keep it. If it just moves two lines into one, skip it.

- [ ] **Step 6: Run all CLI tests**

Run: `targ test -- -run "TestRunPost|TestRunIntent|TestRunLearn" ./internal/cli/`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/targets.go
git commit -m "feat(cli): add engram learn command (feedback and fact types)

AI-Used: [claude]"
```

---

### Task 7: CLI commands — `engram subscribe` and `engram status`

**Files:**
- Modify: `internal/cli/cli_api.go`
- Modify: `internal/cli/cli_api_test.go`
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/targets.go`

- [ ] **Step 1: Write failing test for subscribe — always prints messages as they arrive**

```go
func TestRunSubscribe_AlwaysPrintsReceivedMessages(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	firstCall := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if firstCall {
			firstCall = false
			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.SubscribeResponse{
				Messages: []apiclient.ChatMessage{
					{From: "engram-agent", To: "lead-1", Text: "memory: use DI"},
				},
				Cursor: 5,
			})
			g.Expect(encErr).NotTo(HaveOccurred())

			return
		}

		// Second call: block until client disconnects.
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())

	var stdout, stderr bytes.Buffer

	done := make(chan error, 1)
	go func() {
		done <- cli.RunWithContext(
			ctx,
			[]string{"engram", "subscribe", "--agent", "lead-1", "--addr", srv.URL},
			&stdout, &stderr, nil,
		)
	}()

	// Wait for output, then cancel.
	g.Eventually(func() string { return stdout.String() }).
		Should(ContainSubstring("memory: use DI"))
	cancel()
	<-done
}
```

- [ ] **Step 2: Write failing test for status — always prints JSON with agents**

```go
func TestRunStatus_AlwaysPrintsAgentList(t *testing.T) {
	t.Parallel()
	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		agentCount := rapid.IntRange(0, 5).Draw(rt, "agentCount")
		agents := make([]string, 0, agentCount)

		for range agentCount {
			agents = append(agents, rapid.StringMatching(`[a-z]{3,10}`).Draw(rt, "agent"))
		}

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			encErr := json.NewEncoder(w).Encode(apiclient.StatusResponse{
				Running: true, Agents: agents,
			})
			g.Expect(encErr).NotTo(HaveOccurred())
		}))
		defer srv.Close()

		var stdout, stderr bytes.Buffer
		err := cli.Run(
			[]string{"engram", "status", "--addr", srv.URL},
			&stdout, &stderr, nil,
		)
		g.Expect(err).NotTo(HaveOccurred())

		// Invariant: all agent names appear in output.
		for _, agent := range agents {
			g.Expect(stdout.String()).To(ContainSubstring(agent))
		}
	})
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `targ test -- -run "TestRunSubscribe|TestRunStatus" ./internal/cli/`
Expected: FAIL — RunWithContext not defined, unknown commands

- [ ] **Step 4: Add RunWithContext to cli.go**

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

- [ ] **Step 5: Add SubscribeArgs, StatusArgs, implement runSubscribe and runStatus**

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
	return BuildFlags("--agent", a.Agent, "--after-cursor", a.AfterCursor, "--addr", a.Addr)
}

// StatusArgs holds flags for `engram status`.
type StatusArgs struct {
	Addr string `targ:"flag,name=addr,desc=API server address"`
}

// StatusFlags returns the CLI flag args for the status subcommand.
func StatusFlags(a StatusArgs) []string {
	return BuildFlags("--addr", a.Addr)
}
```

Register both in `BuildTargets`.

Add `"status"` to the `runAPIDispatch` case in `Run()`:
```go
	case "status":
		return runStatus(ctx, args, stdout)
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

func runStatus(ctx context.Context, args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)

	var addr string
	fs.StringVar(&addr, "addr", defaultAPIAddr, "API server address")

	if parseErr := fs.Parse(args); parseErr != nil {
		return fmt.Errorf("status: %w", parseErr)
	}

	client := apiclient.New(addr, http.DefaultClient)

	resp, err := client.Status(ctx)
	if err != nil {
		return fmt.Errorf("status: %w", err)
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")

	return enc.Encode(resp)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `targ test -- -run "TestRunSubscribe|TestRunStatus" ./internal/cli/`
Expected: PASS

- [ ] **Step 7: Refactor — deduplicate flag parsing across all 5 commands**

All commands parse `--addr` with the same default. Review whether a shared `parseAddrFlag` is warranted now that all 5 commands exist. If 4+ commands share the exact same pattern, extract it. Otherwise leave as-is.

Review `runSubscribe` loop — does context cancellation propagate cleanly? The `client.Subscribe` call uses `ctx`, so yes.

- [ ] **Step 8: Run full test suite**

Run: `targ test ./internal/cli/ ./internal/apiclient/`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add internal/cli/cli_api.go internal/cli/cli_api_test.go internal/cli/cli.go internal/cli/targets.go
git commit -m "feat(cli): add engram subscribe and status commands

AI-Used: [claude]"
```

---

### Task 8: Full suite green + quality check

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
- Name magic numbers as constants
- Wrap errors with context: `fmt.Errorf("context: %w", err)` not bare `return err`
- Use `http.NewRequestWithContext` not `http.NewRequest`

- [ ] **Step 4: Re-run quality check**

Run: `targ check-full`
Expected: All checks pass

- [ ] **Step 5: Final refactor pass across all new code**

Review all new files for:
- **DRY:** Any repeated patterns across `cli_api.go` that should be extracted?
- **SOLID:** Does each function have a single responsibility?
- **Naming:** Are variable names descriptive? (`addr` not `a`, `cursor` not `c`)
- **Error wrapping:** Every error wrapped with context?

- [ ] **Step 6: Run full quality check after refactoring**

Run: `targ check-full`
Expected: All checks pass

- [ ] **Step 7: Commit any fixes**

```bash
git add -A
git commit -m "fix: address lint, coverage, and style issues from stage 0

AI-Used: [claude]"
```
