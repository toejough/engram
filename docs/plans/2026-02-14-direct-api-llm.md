# ISSUE-211: Direct API LLM Client — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace `ClaudeCLIExtractor` (~59s/call) with `DirectAPIExtractor` (~730ms/call) using macOS Keychain OAuth token.

**Architecture:** New `DirectAPIExtractor` struct calls `https://api.anthropic.com/v1/messages` directly via `net/http`. Auth token extracted from macOS Keychain (`security find-generic-password`). Falls back to `ClaudeCLIExtractor` when Keychain/API unavailable.

**Tech Stack:** Go `net/http`, `os/exec` (Keychain), `encoding/json`. gomega + rapid for tests.

---

### Task 1: Keychain Auth — Failing Tests

**Files:**
- Create: `internal/memory/auth_test.go`

**Step 1: Write failing tests for `GetKeychainToken`**

```go
package memory_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

func TestGetKeychainToken_ReturnsToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken":  "sk-ant-oat01-test-token",
			"refreshToken": "sk-ant-oart01-refresh",
			"expiresAt":    "2099-12-31T23:59:59Z",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(token).To(Equal("sk-ant-oat01-test-token"))
}

func TestGetKeychainToken_ReturnsErrWhenKeychainFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.")
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrAuthUnavailable)).To(BeTrue())
	g.Expect(token).To(BeEmpty())
}

func TestGetKeychainToken_ReturnsErrOnMalformedJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("not json"), nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrAuthUnavailable)).To(BeTrue())
	g.Expect(token).To(BeEmpty())
}

func TestGetKeychainToken_ReturnsErrOnEmptyAccessToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrAuthUnavailable)).To(BeTrue())
	g.Expect(token).To(BeEmpty())
}

func TestGetKeychainToken_ReturnsErrOnExpiredToken(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat01-expired",
			"expiresAt":   "2020-01-01T00:00:00Z",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrAuthUnavailable)).To(BeTrue())
	g.Expect(token).To(BeEmpty())
}

func TestGetKeychainToken_MissingExpiresAtTreatedAsValid(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat01-no-expiry",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	token, err := auth.GetToken(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(token).To(Equal("sk-ant-oat01-no-expiry"))
}

func TestGetKeychainToken_PassesCorrectSecurityArgs(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedName string
	var capturedArgs []string
	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat01-x",
			"expiresAt":   "2099-12-31T23:59:59Z",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := memory.KeychainAuth{
		CommandRunner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			capturedName = name
			capturedArgs = args
			return credsJSON, nil
		},
	}

	_, err := auth.GetToken(context.Background())
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(capturedName).To(Equal("security"))
	g.Expect(capturedArgs).To(ContainElements("find-generic-password", "-s", "Claude Code-credentials", "-w"))
}
```

**Step 2: Run tests to verify they fail**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestGetKeychainToken -v`
Expected: FAIL — `KeychainAuth` type and `ErrAuthUnavailable` not defined.

---

### Task 2: Keychain Auth — Implementation

**Files:**
- Create: `internal/memory/auth.go`

**Step 3: Write minimal implementation**

```go
package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

var ErrAuthUnavailable = errors.New("auth unavailable")

type KeychainAuth struct {
	CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func NewKeychainAuth() *KeychainAuth {
	return &KeychainAuth{
		CommandRunner: defaultCommandRunner,
	}
}

func (k *KeychainAuth) GetToken(ctx context.Context) (string, error) {
	user := os.Getenv("USER")
	out, err := k.CommandRunner(ctx, "security", "find-generic-password",
		"-s", "Claude Code-credentials",
		"-a", user,
		"-w",
	)
	if err != nil {
		return "", fmt.Errorf("%w: keychain read failed: %v", ErrAuthUnavailable, err)
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
			ExpiresAt   string `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal(out, &creds); err != nil {
		return "", fmt.Errorf("%w: failed to parse keychain credentials: %v", ErrAuthUnavailable, err)
	}
	if creds.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("%w: no accessToken in keychain credentials", ErrAuthUnavailable)
	}

	// Check expiry if present
	if creds.ClaudeAiOauth.ExpiresAt != "" {
		expiry, err := time.Parse(time.RFC3339, creds.ClaudeAiOauth.ExpiresAt)
		if err == nil && time.Now().After(expiry) {
			return "", fmt.Errorf("%w: token expired at %s", ErrAuthUnavailable, creds.ClaudeAiOauth.ExpiresAt)
		}
	}

	return creds.ClaudeAiOauth.AccessToken, nil
}
```

Note: `defaultCommandRunner` is already defined in `llm.go` — reuse it.

**Step 4: Run tests to verify they pass**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestGetKeychainToken -v`
Expected: All PASS.

**Step 5: Commit**

```
feat(memory): add KeychainAuth for OAuth token extraction (ISSUE-211)
```

---

### Task 3: DirectAPIExtractor — Core Call — Failing Tests

**Files:**
- Create: `internal/memory/llm_api_test.go`

Write tests for the core `callAPI` method using `net/http/httptest`. This is the foundational method all interface methods will delegate to.

**Step 6: Write failing tests**

```go
package memory_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/toejough/projctl/internal/memory"
)

func TestDirectAPIExtractor_Extract_ReturnsObservation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	obs := memory.Observation{
		Type:      "correction",
		Concepts:  []string{"git"},
		Principle: "Never amend pushed commits",
	}
	obsJSON, _ := json.Marshal(obs)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		g.Expect(r.Header.Get("Authorization")).To(Equal("Bearer test-token"))
		g.Expect(r.Header.Get("anthropic-beta")).To(Equal("oauth-2025-04-20"))
		g.Expect(r.Header.Get("anthropic-version")).To(Equal("2023-06-01"))

		// Verify body
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		g.Expect(body.Messages).To(HaveLen(1))

		// Return content block with the JSON observation
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(obsJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("test-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Extract(context.Background(), "test content")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(result.Type).To(Equal("correction"))
	g.Expect(result.Principle).To(Equal("Never amend pushed commits"))
}

func TestDirectAPIExtractor_Extract_ReturnsErrLLMUnavailableOn401(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "invalid token"},
		})
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("bad-token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Extract(context.Background(), "content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(errors.Is(err, memory.ErrLLMUnavailable)).To(BeTrue())
	g.Expect(result).To(BeNil())
}

func TestDirectAPIExtractor_Extract_ReturnsErrOnBadJSON(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": "not valid json"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	result, err := extractor.Extract(context.Background(), "content")
	g.Expect(err).To(HaveOccurred())
	g.Expect(result).To(BeNil())
}

func TestDirectAPIExtractor_Extract_RespectsContextCancellation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // block until cancelled
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := extractor.Extract(ctx, "content")
	g.Expect(err).To(HaveOccurred())
}

func TestDirectAPIExtractor_Extract_SendsCorrectModel(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var capturedModel string
	obs := memory.Observation{Type: "pattern", Concepts: []string{"x"}, Principle: "p"}
	obsJSON, _ := json.Marshal(obs)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		json.Unmarshal(bodyBytes, &body)
		capturedModel = body.Model
		resp := map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": string(obsJSON)},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	extractor := memory.NewDirectAPIExtractor("token",
		memory.WithBaseURL(srv.URL),
		memory.WithModel("claude-haiku-4-5-20251001"),
	)

	_, err := extractor.Extract(context.Background(), "content")
	g.Expect(err).ToNot(HaveOccurred())
	g.Expect(capturedModel).To(Equal("claude-haiku-4-5-20251001"))
}

func TestDirectAPIExtractor_ImplementsLLMExtractor(t *testing.T) {
	t.Parallel()

	var _ memory.LLMExtractor = &memory.DirectAPIExtractor{}
}

func TestDirectAPIExtractor_ImplementsSkillCompiler(t *testing.T) {
	t.Parallel()

	var _ memory.SkillCompiler = &memory.DirectAPIExtractor{}
}

func TestDirectAPIExtractor_ImplementsSpecificityDetector(t *testing.T) {
	t.Parallel()

	var _ memory.SpecificityDetector = &memory.DirectAPIExtractor{}
}
```

**Step 7: Run tests to verify they fail**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestDirectAPIExtractor -v`
Expected: FAIL — `DirectAPIExtractor` type not defined.

---

### Task 4: DirectAPIExtractor — Core Implementation

**Files:**
- Create: `internal/memory/llm_api.go`

**Step 8: Write minimal implementation**

```go
package memory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type DirectAPIExtractor struct {
	token   string
	model   string
	baseURL string
	timeout time.Duration
	client  *http.Client
}

type DirectAPIOption func(*DirectAPIExtractor)

func WithBaseURL(url string) DirectAPIOption {
	return func(d *DirectAPIExtractor) { d.baseURL = url }
}

func WithModel(model string) DirectAPIOption {
	return func(d *DirectAPIExtractor) { d.model = model }
}

func WithTimeout(timeout time.Duration) DirectAPIOption {
	return func(d *DirectAPIExtractor) { d.timeout = timeout }
}

func NewDirectAPIExtractor(token string, opts ...DirectAPIOption) *DirectAPIExtractor {
	d := &DirectAPIExtractor{
		token:   token,
		model:   "claude-haiku-4-5-20251001",
		baseURL: "https://api.anthropic.com",
		timeout: 30 * time.Second,
		client:  &http.Client{},
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// callAPI sends a prompt to the Anthropic API and returns the raw text response.
func (d *DirectAPIExtractor) callAPI(ctx context.Context, prompt string, maxTokens int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	body := map[string]any{
		"model":      d.model,
		"max_tokens": maxTokens,
		"messages": []map[string]any{
			{"role": "user", "content": prompt},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", d.baseURL+"/v1/messages", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLLMUnavailable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w: API returned %d", ErrLLMUnavailable, resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("%w: failed to decode API response: %v", ErrLLMUnavailable, err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("%w: API error: %s", ErrLLMUnavailable, result.Error.Message)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("%w: empty response content", ErrLLMUnavailable)
	}

	return []byte(result.Content[0].Text), nil
}
```

**Step 9: Add all interface methods (same prompts as `ClaudeCLIExtractor`)**

Each method on `DirectAPIExtractor` mirrors the corresponding method on `ClaudeCLIExtractor` — same prompt construction, same JSON parsing — but calls `d.callAPI()` instead of `c.runClaude()`. Copy the method bodies from `llm.go` and replace `c.runClaude(ctx, prompt)` with `d.callAPI(ctx, prompt, 1024)`.

Methods to implement:
- `Extract(ctx, content) → (*Observation, error)` — maxTokens: 256
- `Synthesize(ctx, memories) → (string, error)` — maxTokens: 512
- `Curate(ctx, query, candidates) → ([]CuratedResult, error)` — maxTokens: 2048
- `Decide(ctx, newContent, existing) → (*IngestDecision, error)` — maxTokens: 256
- `CompileSkill(ctx, theme, memories) → (string, error)` — maxTokens: 4096
- `IsNarrowLearning(ctx, learning) → (bool, string, error)` — maxTokens: 256
- `Rewrite(ctx, content) → (string, error)` — maxTokens: 512
- `AddRationale(ctx, content) → (string, error)` — maxTokens: 512

**Step 10: Run tests to verify they pass**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestDirectAPIExtractor -v`
Expected: All PASS.

**Step 11: Commit**

```
feat(memory): add DirectAPIExtractor for direct Anthropic API calls (ISSUE-211)
```

---

### Task 5: Remaining Interface Method Tests

**Files:**
- Modify: `internal/memory/llm_api_test.go`

Add tests for `Synthesize`, `Curate`, `Decide`, `CompileSkill`, `IsNarrowLearning`, `Rewrite`, `AddRationale` — following the same mock HTTP server pattern. Each method needs:
1. Happy path (valid response)
2. LLM unavailable (server returns 401)
3. Invalid JSON response (where applicable)

**Step 12: Write tests for remaining methods**

Use the same `httptest.NewServer` pattern. For each method, the mock server returns the expected response format. These should all pass immediately since the implementation from Task 4 already handles them.

**Step 13: Run all tests**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run "TestDirectAPIExtractor|TestGetKeychainToken" -v`
Expected: All PASS.

**Step 14: Commit**

```
test(memory): add comprehensive tests for DirectAPIExtractor methods (ISSUE-211)
```

---

### Task 6: Constructor with Fallback — Failing Tests

**Files:**
- Modify: `internal/memory/llm_api_test.go`

**Step 15: Write failing tests for `NewLLMExtractor`**

`NewLLMExtractor` is the factory function that tries DirectAPI first, falls back to CLI.

```go
func TestNewLLMExtractor_ReturnsDirectAPIWhenTokenAvailable(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	creds := map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "sk-ant-oat01-valid",
			"expiresAt":   "2099-12-31T23:59:59Z",
		},
	}
	credsJSON, _ := json.Marshal(creds)

	auth := &memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return credsJSON, nil
		},
	}

	extractor := memory.NewLLMExtractor(memory.WithAuth(auth))
	_, isDirectAPI := extractor.(*memory.DirectAPIExtractor)
	g.Expect(isDirectAPI).To(BeTrue())
}

func TestNewLLMExtractor_FallsBackToCLIWhenKeychainFails(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	auth := &memory.KeychainAuth{
		CommandRunner: func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("keychain not available")
		},
	}

	extractor := memory.NewLLMExtractor(memory.WithAuth(auth))
	_, isCLI := extractor.(*memory.ClaudeCLIExtractor)
	g.Expect(isCLI).To(BeTrue())
}
```

**Step 16: Run tests to verify they fail**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestNewLLMExtractor -v`
Expected: FAIL — `NewLLMExtractor` not defined.

---

### Task 7: Constructor with Fallback — Implementation

**Files:**
- Modify: `internal/memory/llm_api.go`

**Step 17: Implement `NewLLMExtractor`**

```go
// LLMClient is the union interface for all LLM functionality.
type LLMClient interface {
	LLMExtractor
	SkillCompiler
	SpecificityDetector
}

type LLMExtractorOption func(*llmExtractorConfig)

type llmExtractorConfig struct {
	auth *KeychainAuth
}

func WithAuth(auth *KeychainAuth) LLMExtractorOption {
	return func(c *llmExtractorConfig) { c.auth = auth }
}

// NewLLMExtractor creates the best available LLM client.
// Tries DirectAPIExtractor first (via Keychain token), falls back to ClaudeCLIExtractor.
func NewLLMExtractor(opts ...LLMExtractorOption) LLMClient {
	cfg := &llmExtractorConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	auth := cfg.auth
	if auth == nil {
		auth = NewKeychainAuth()
	}

	token, err := auth.GetToken(context.Background())
	if err == nil {
		return NewDirectAPIExtractor(token)
	}

	return NewClaudeCLIExtractor()
}
```

Note: `DirectAPIExtractor` needs to satisfy `LLMClient` (it already implements all three sub-interfaces). `ClaudeCLIExtractor` also needs to satisfy `LLMClient` — check that it already implements `SkillCompiler` and `SpecificityDetector` (it does per `llm.go`).

**Step 18: Run tests to verify they pass**

Run: `go test -tags sqlite_fts5 ./internal/memory/ -run TestNewLLMExtractor -v`
Expected: All PASS.

**Step 19: Commit**

```
feat(memory): add NewLLMExtractor factory with API/CLI fallback (ISSUE-211)
```

---

### Task 8: Wire into CLI — Failing Tests (Optional)

The CLI wiring is thin enough that integration testing via the existing test suite is sufficient. This task is a straightforward replacement.

**Files:**
- Modify: `cmd/projctl/memory_optimize.go` (3 wiring sites)
- Modify: `cmd/projctl/memory.go` (1 wiring site)

**Step 20: Replace `NewClaudeCLIExtractor()` with `NewLLMExtractor()`**

In `cmd/projctl/memory_optimize.go`, replace the three wiring sites:

```go
// Line 77-82: Legacy optimize
if !args.NoLLM {
	extractor := memory.NewLLMExtractor()
	opts.SkillCompiler = extractor
	opts.SpecificDetector = extractor
	opts.Extractor = extractor
}

// Line 194-196: Interactive optimize
if !args.NoLLM {
	extractor = memory.NewLLMExtractor()
}
```

In `cmd/projctl/memory.go`, replace line 39-41:

```go
if !args.NoLLM {
	opts.Extractor = memory.NewLLMExtractor()
}
```

And the curated query extractor in `cmd/projctl/memory.go` line 269-271:

```go
if tier == memory.TierCurated {
	extractor = memory.NewLLMExtractor()
}
```

**Step 21: Run existing tests to verify nothing breaks**

Run: `go test -tags sqlite_fts5 ./...`
Expected: All existing tests PASS unchanged.

**Step 22: Build and verify**

Run: `go build ./cmd/projctl/`
Expected: Clean build, no errors.

**Step 23: Commit**

```
feat(memory): wire DirectAPIExtractor as default LLM client (ISSUE-211)
```

---

### Task 9: Manual Smoke Test

**Step 24: Run `projctl memory optimize --review --yes` and verify speed**

Run: `time projctl memory optimize --review --yes`

Expected: LLM calls complete in <2s each (vs ~60s before). Total runtime dramatically reduced.

**Step 25: Run with `--no-llm` to verify fallback flag still works**

Run: `projctl memory optimize --review --yes --no-llm`

Expected: Completes without LLM calls (same as before).

**Step 26: Commit — close ISSUE-211**

Update `docs/issues.md`: set ISSUE-211 status to Complete.

```
docs: close ISSUE-211 — direct API LLM client
```

---

## Summary

| Task | What | Files |
|------|------|-------|
| 1-2 | Keychain auth (test + impl) | `auth.go`, `auth_test.go` |
| 3-4 | DirectAPIExtractor core (test + impl) | `llm_api.go`, `llm_api_test.go` |
| 5 | Remaining method tests | `llm_api_test.go` |
| 6-7 | Factory with fallback (test + impl) | `llm_api.go`, `llm_api_test.go` |
| 8 | Wire into CLI | `memory_optimize.go`, `memory.go` |
| 9 | Smoke test + close issue | manual, `issues.md` |
