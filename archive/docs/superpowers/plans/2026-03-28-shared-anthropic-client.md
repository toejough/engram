# Shared Anthropic API Client — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract duplicated Anthropic API types, constants, and HTTP call logic into a shared `internal/anthropic` package.

**Architecture:** Create `internal/anthropic` with shared types, constants, `HTTPDoer` interface, and a `Client.Call()` method that returns the text from `Content[0].Text`. Migrate `extract`, `classify`, and `cli` to use it. Update model constant references in `signal`, `maintain`, and `instruct`.

**Tech Stack:** Go, net/http, encoding/json

---

### Task 1: Create shared anthropic package with types and Call method

**Files:**
- Create: `internal/anthropic/anthropic.go`
- Create: `internal/anthropic/anthropic_test.go`

- [ ] **Step 1: Write failing tests for Client.Call**

```go
package anthropic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/anthropic"
)

type fakeDoer struct {
	lastRequest *http.Request
	response    *http.Response
	err         error
}

func (f *fakeDoer) Do(req *http.Request) (*http.Response, error) {
	f.lastRequest = req
	return f.response, f.err
}

func makeAPIResponse(t *testing.T, g Gomega, text string) []byte {
	t.Helper()

	resp := struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}{
		Content: []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{{Type: "text", Text: text}},
	}

	data, err := json.Marshal(resp)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return nil
	}

	return data
}

func TestCall_ReturnsTextContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	body := makeAPIResponse(t, g, "hello world")
	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
		},
	}

	client := anthropic.NewClient("test-token", doer)
	result, err := client.Call(context.Background(), anthropic.HaikuModel, "system", "user", 1024)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(result).To(Equal("hello world"))
}

func TestCall_SetsCorrectHeaders(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	body := makeAPIResponse(t, g, "ok")
	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(body)),
		},
	}

	client := anthropic.NewClient("my-token", doer)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(doer.lastRequest).NotTo(BeNil())

	if doer.lastRequest == nil {
		return
	}

	g.Expect(doer.lastRequest.Header.Get("Authorization")).To(Equal("Bearer my-token"))
	g.Expect(doer.lastRequest.Header.Get("Anthropic-Version")).To(Equal("2023-06-01"))
	g.Expect(doer.lastRequest.Header.Get("Anthropic-Beta")).To(Equal("oauth-2025-04-20"))
	g.Expect(doer.lastRequest.Header.Get("Content-Type")).To(Equal("application/json"))
}

func TestCall_NilResponse(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	doer := &fakeDoer{response: nil}
	client := anthropic.NewClient("token", doer)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).To(MatchError(anthropic.ErrNilResponse))
}

func TestCall_EmptyContent(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	emptyResp, err := json.Marshal(struct {
		Content []struct{} `json:"content"`
	}{Content: []struct{}{}})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	doer := &fakeDoer{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(emptyResp)),
		},
	}

	client := anthropic.NewClient("token", doer)
	_, callErr := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(callErr).To(MatchError(anthropic.ErrNoContentBlocks))
}

func TestCall_NoToken(t *testing.T) {
	t.Parallel()

	g := NewGomegaWithT(t)

	client := anthropic.NewClient("", nil)
	_, err := client.Call(context.Background(), anthropic.HaikuModel, "sys", "usr", 1024)
	g.Expect(err).To(MatchError(anthropic.ErrNoToken))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test -- -run 'TestCall_' ./internal/anthropic/...`
Expected: FAIL — package does not exist

- [ ] **Step 3: Write minimal implementation**

```go
// Package anthropic provides a shared client for the Anthropic Messages API.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// Exported errors.
var (
	ErrNilResponse    = errors.New("anthropic: nil response")
	ErrNoContentBlocks = errors.New("anthropic: response contained no content blocks")
	ErrNoToken        = errors.New("anthropic: no API token configured")
)

// Model constants.
const (
	HaikuModel = "claude-haiku-4-5-20251001"
)

// API constants.
const (
	defaultAPIURL = "https://api.anthropic.com/v1/messages"
	apiVersion    = "2023-06-01"
	betaHeader    = "oauth-2025-04-20"
)

// HTTPDoer is the interface for making HTTP requests.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// contentBlock is a content block in an Anthropic API response.
type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// message is a single message in the Anthropic messages API.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// request is the request body for the Anthropic messages API.
//
//nolint:tagliatelle // Anthropic API requires snake_case JSON field names.
type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

// response is the response body from the Anthropic messages API.
type response struct {
	Content []contentBlock `json:"content"`
}

// Client calls the Anthropic Messages API.
type Client struct {
	token  string
	client HTTPDoer
	apiURL string
}

// NewClient creates a Client. Pass http.DefaultClient as doer in production.
func NewClient(token string, doer HTTPDoer) *Client {
	return &Client{
		token:  token,
		client: doer,
		apiURL: defaultAPIURL,
	}
}

// CallerFunc is the function signature used by packages that receive an LLM
// caller via dependency injection (signal, maintain, instruct).
type CallerFunc func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error)

// Caller returns a CallerFunc backed by this client with the given maxTokens.
func (c *Client) Caller(maxTokens int) CallerFunc {
	return func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
		return c.Call(ctx, model, systemPrompt, userPrompt, maxTokens)
	}
}

// Call makes a single call to the Anthropic Messages API and returns the text response.
func (c *Client) Call(
	ctx context.Context,
	model, systemPrompt, userPrompt string,
	maxTokens int,
) (string, error) {
	if c.token == "" {
		return "", ErrNoToken
	}

	reqBody := request{
		Model:     model,
		MaxTokens: maxTokens,
		System:    systemPrompt,
		Messages:  []message{{Role: "user", Content: userPrompt}},
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("anthropic: marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.apiURL,
		bytes.NewReader(reqBytes),
	)
	if err != nil {
		return "", fmt.Errorf("anthropic: creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Anthropic-Version", apiVersion)
	req.Header.Set("Anthropic-Beta", betaHeader)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic: calling API: %w", err)
	}

	if resp == nil {
		return "", ErrNilResponse
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("anthropic: reading response: %w", err)
	}

	var apiResp response

	if jsonErr := json.Unmarshal(body, &apiResp); jsonErr != nil {
		return "", fmt.Errorf("anthropic: parsing response: %w", jsonErr)
	}

	if len(apiResp.Content) == 0 {
		return "", ErrNoContentBlocks
	}

	return apiResp.Content[0].Text, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test -- -run 'TestCall_' ./internal/anthropic/...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/anthropic/
git commit -m "refactor: add shared internal/anthropic client package

Extracts duplicated API types, constants, and HTTP call logic.
Provides Client.Call() for direct callers and CallerFunc for DI consumers.

Refs #409

AI-Used: [claude]"
```

### Task 2: Migrate extract package to use shared client

**Files:**
- Modify: `internal/extract/extract.go` — remove duplicated types/constants/HTTP logic, use `anthropic.Client`
- Modify: `internal/extract/extract_test.go` — update to use `anthropic.HTTPDoer`

- [ ] **Step 1: Run existing extract tests to confirm green baseline**

Run: `targ test -- ./internal/extract/...`
Expected: PASS

- [ ] **Step 2: Refactor extract.go**

Remove from `extract.go`:
- The `HTTPDoer` interface (use `anthropic.HTTPDoer`)
- The `anthropicContentBlock`, `anthropicMessage`, `anthropicRequest`, `anthropicResponse` types
- The `anthropicAPIURL`, `anthropicModel`, `anthropicVersion` constants
- The `sendRequest` method
- The `callLLM` method's HTTP handling

Replace `LLMExtractor` fields:
```go
type LLMExtractor struct {
	client   *anthropic.Client
	guidance []ExtractionGuidance
}
```

Replace constructor:
```go
func New(token string, client anthropic.HTTPDoer, opts ...Option) *LLMExtractor {
	e := &LLMExtractor{
		client: anthropic.NewClient(token, client),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}
```

Replace `callLLM`:
```go
func (e *LLMExtractor) callLLM(
	ctx context.Context,
	transcript string,
) ([]memory.CandidateLearning, error) {
	text, err := e.client.Call(ctx, anthropic.HaikuModel, SystemPromptWithGuidance(e.guidance), transcript, maxResponseTokens)
	if err != nil {
		return nil, err
	}
	return parseLLMText(text)
}
```

Update `parseLLMResponse` → `parseLLMText` to accept `string` instead of `*http.Response` (skip the body reading / anthropicResponse parsing since the shared client already does it).

Keep `maxResponseTokens = 2048` as a local constant (it's domain-specific).

- [ ] **Step 3: Run extract tests to verify they pass**

Run: `targ test -- ./internal/extract/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/extract/
git commit -m "refactor(extract): use shared anthropic client

Remove duplicated types, constants, and HTTP logic.

Refs #409

AI-Used: [claude]"
```

### Task 3: Migrate classify package to use shared client

**Files:**
- Modify: `internal/classify/classify.go` — same pattern as extract
- Modify: `internal/classify/classify_test.go` — update mock type

- [ ] **Step 1: Run existing classify tests to confirm green baseline**

Run: `targ test -- ./internal/classify/...`
Expected: PASS

- [ ] **Step 2: Refactor classify.go**

Same pattern as extract: remove duplicated types/constants/HTTP logic, use `anthropic.Client`.

Replace `LLMClassifier` fields:
```go
type LLMClassifier struct {
	client *anthropic.Client
}
```

Replace constructor:
```go
func New(token string, httpClient anthropic.HTTPDoer) *LLMClassifier {
	return &LLMClassifier{
		client: anthropic.NewClient(token, httpClient),
	}
}
```

Replace the `classify` method body to use `c.client.Call(...)` and parse the returned text string instead of the raw HTTP response.

Keep `maxResponseTokens = 1024` as local constant.

- [ ] **Step 3: Run classify tests to verify they pass**

Run: `targ test -- ./internal/classify/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/classify/
git commit -m "refactor(classify): use shared anthropic client

Remove duplicated types, constants, and HTTP logic.

Refs #409

AI-Used: [claude]"
```

### Task 4: Migrate cli package to use shared client

**Files:**
- Modify: `internal/cli/cli.go` — remove duplicated types/constants/`callAnthropicAPI`/`makeAnthropicCaller`, use `anthropic.Client`

- [ ] **Step 1: Run existing cli tests to confirm green baseline**

Run: `targ test -- ./internal/cli/...`
Expected: PASS

- [ ] **Step 2: Refactor cli.go**

Remove from `cli.go`:
- `anthropicContentBlock`, `anthropicMessage`, `anthropicRequest`, `anthropicResponse` types
- `anthropicMaxTokens`, `anthropicVersion`, `haikuModel`, `maintainModel` constants
- `errNilAPIResponse`, `errNoContentBlocks` error variables
- `callAnthropicAPI` function
- `makeAnthropicCaller` function

Replace with:
```go
func makeAnthropicCaller(
	token string,
) func(ctx context.Context, model, systemPrompt, userPrompt string) (string, error) {
	client := anthropic.NewClient(token, &http.Client{})
	return client.Caller(anthropicMaxTokens)
}
```

Keep `anthropicMaxTokens = 1024` as local constant (or inline it in the `Caller` call). Move `AnthropicAPIURL` to the anthropic package if tests need to override it — or keep the exported var and set it on the client. Check if any tests override `AnthropicAPIURL`.

Update all references to `haikuModel` → `anthropic.HaikuModel` and `maintainModel` → `anthropic.HaikuModel` (they're the same value).

- [ ] **Step 3: Run cli tests to verify they pass**

Run: `targ test -- ./internal/cli/...`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/cli/ internal/anthropic/
git commit -m "refactor(cli): use shared anthropic client

Remove duplicated types, constants, callAnthropicAPI, makeAnthropicCaller.

Refs #409

AI-Used: [claude]"
```

### Task 5: Update model constants in signal, maintain, instruct

**Files:**
- Modify: `internal/signal/llm_confirm.go` — replace `confirmerModel` with `anthropic.HaikuModel`
- Modify: `internal/maintain/maintain.go` — replace `maintainModel` with `anthropic.HaikuModel`
- Modify: `internal/instruct/audit.go` — replace `haikuModel` with `anthropic.HaikuModel`

- [ ] **Step 1: Replace model constants**

In each file, remove the local model constant definition and replace usages with `anthropic.HaikuModel`. Add `"engram/internal/anthropic"` to imports.

- [ ] **Step 2: Run full test suite**

Run: `targ test`
Expected: PASS

- [ ] **Step 3: Run check-full for lint**

Run: `targ check-full`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add internal/signal/ internal/maintain/ internal/instruct/
git commit -m "refactor: use anthropic.HaikuModel in signal, maintain, instruct

Single source of truth for model constant. Closes #409

AI-Used: [claude]"
```
