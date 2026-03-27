# Token Resolver Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move OAuth token sourcing from duplicated shell hooks into the Go binary, resolving once per invocation.

**Architecture:** New `internal/tokenresolver/` package with a `Resolver` interface. Default impl chains env var → macOS Keychain lookup (Darwin only). Each CLI command that needs a token calls `resolver.Resolve(ctx)` instead of `os.Getenv`. Three hooks lose their Keychain lookup blocks.

**Tech Stack:** Go, `os/exec` (for `security` CLI), `encoding/json`, DI via injected functions.

**Spec:** `docs/superpowers/specs/2026-03-22-token-resolver-design.md`

**Build/test commands:** `targ test`, `targ check-full`

---

### Task 1: Create `internal/tokenresolver/` package with Resolver interface and tests

**Files:**
- Create: `internal/tokenresolver/tokenresolver.go`
- Create: `internal/tokenresolver/tokenresolver_test.go`

- [ ] **Step 1: Write failing tests for the resolver**

```go
package tokenresolver_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/onsi/gomega"
	"engram/internal/tokenresolver"
)

func TestResolver_EnvVarPresent(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	r := tokenresolver.New(
		func(key string) string { return "env-token-123" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			t.Fatal("should not call command executor when env var present")
			return nil, nil
		},
		"darwin",
	)

	token, err := r.Resolve(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(token).To(gomega.Equal("env-token-123"))
}

func TestResolver_KeychainFallback(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	keychainJSON, _ := json.Marshal(map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "keychain-token-456",
		},
	})

	r := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return keychainJSON, nil
		},
		"darwin",
	)

	token, err := r.Resolve(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(token).To(gomega.Equal("keychain-token-456"))
}

func TestResolver_KeychainSkippedOnNonDarwin(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	r := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			t.Fatal("should not call command executor on non-Darwin")
			return nil, nil
		},
		"Linux",
	)

	token, err := r.Resolve(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(token).To(gomega.Equal(""))
}

func TestResolver_KeychainError_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	r := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return nil, errors.New("security: SecKeychainSearchCopyNext: access denied")
		},
		"darwin",
	)

	token, err := r.Resolve(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(token).To(gomega.Equal(""))
}

func TestResolver_KeychainMalformedJSON(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	r := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return []byte("not json"), nil
		},
		"darwin",
	)

	token, err := r.Resolve(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(token).To(gomega.Equal(""))
}

func TestResolver_KeychainMissingField(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	keychainJSON, _ := json.Marshal(map[string]any{
		"someOtherField": "value",
	})

	r := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return keychainJSON, nil
		},
		"darwin",
	)

	token, err := r.Resolve(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(token).To(gomega.Equal(""))
}

func TestResolver_KeychainEmptyAccessToken(t *testing.T) {
	t.Parallel()
	g := gomega.NewWithT(t)

	keychainJSON, _ := json.Marshal(map[string]any{
		"claudeAiOauth": map[string]any{
			"accessToken": "",
		},
	})

	r := tokenresolver.New(
		func(_ string) string { return "" },
		func(_ context.Context, _ string, _ ...string) ([]byte, error) {
			return keychainJSON, nil
		},
		"darwin",
	)

	token, err := r.Resolve(context.Background())
	g.Expect(err).NotTo(gomega.HaveOccurred())
	if err != nil {
		return
	}
	g.Expect(token).To(gomega.Equal(""))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `targ test`
Expected: FAIL — `tokenresolver` package doesn't exist yet.

- [ ] **Step 3: Write minimal implementation**

```go
package tokenresolver

import (
	"context"
	"encoding/json"
)

const envKey = "ENGRAM_API_TOKEN"

// Resolver resolves an API token from available sources.
type Resolver struct {
	getenv  func(string) string
	execCmd func(ctx context.Context, name string, args ...string) ([]byte, error)
	goos    string
}

// New creates a Resolver with injected dependencies.
func New(
	getenv func(string) string,
	execCmd func(ctx context.Context, name string, args ...string) ([]byte, error),
	goos string,
) *Resolver {
	return &Resolver{getenv: getenv, execCmd: execCmd, goos: goos}
}

// Resolve returns an API token, checking env var first then macOS Keychain.
// Returns "" with no error if no token is available.
func (r *Resolver) Resolve(ctx context.Context) (string, error) {
	if token := r.getenv(envKey); token != "" {
		return token, nil
	}

	if r.goos != "darwin" {
		return "", nil
	}

	return r.resolveKeychain(ctx)
}

func (r *Resolver) resolveKeychain(ctx context.Context) (string, error) {
	out, err := r.execCmd(ctx, "security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	if err != nil {
		return "", nil
	}

	var creds struct {
		ClaudeAiOauth struct {
			AccessToken string `json:"accessToken"`
		} `json:"claudeAiOauth"`
	}

	if err := json.Unmarshal(out, &creds); err != nil {
		return "", nil
	}

	return creds.ClaudeAiOauth.AccessToken, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `targ test`
Expected: All PASS.

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: PASS (except `check-uncommitted`).

- [ ] **Step 6: Commit**

```
feat(tokenresolver): add token resolver with env var and Keychain strategies
```

---

### Task 2: Wire resolver into CLI commands

**Files:**
- Modify: `internal/cli/cli.go:1094,1175,1199,1203,1315` (5 call sites)
- Modify: `internal/cli/flush.go:47` (1 call site)

- [ ] **Step 1: Create a shared resolver instance in cli.go**

Add a helper that creates a resolver with real dependencies (`os.Getenv`, `exec.CommandContext`, `runtime.GOOS`). Each `run*` function calls it.

```go
import (
	"os/exec"
	"runtime"
	"engram/internal/tokenresolver"
)

func newTokenResolver() *tokenresolver.Resolver {
	return tokenresolver.New(
		os.Getenv,
		func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, args...).Output()
		},
		runtime.GOOS,
	)
}
```

- [ ] **Step 2: Replace `os.Getenv("ENGRAM_API_TOKEN")` in `runRecall` (line 1315)**

Replace:
```go
token := os.Getenv("ENGRAM_API_TOKEN")
```
With:
```go
token, resolveErr := newTokenResolver().Resolve(ctx)
if resolveErr != nil {
    return resolveErr
}
```

(Use the appropriate context variable available in each function — may be `context.Background()` if none exists.)

- [ ] **Step 3: Replace in `runCorrect` (line 1094)**

Same pattern as step 2.

- [ ] **Step 4: Replace in `runInstructAudit` (line 1175)**

Same pattern as step 2.

- [ ] **Step 5: Replace in `runLearn` (line 1199) and `runMaintain` (line 1203)**

Same pattern — resolve token, pass to `RunLearn`/`RunMaintain`.

- [ ] **Step 6: Replace in `flush.go` (line 47)**

Same pattern — resolve token, pass to `RunLearn`.

- [ ] **Step 7: Run full checks**

Run: `targ check-full`
Expected: PASS (except `check-uncommitted`).

- [ ] **Step 8: Commit**

```
feat(cli): wire token resolver into all commands that need API tokens
```

---

### Task 3: Remove Keychain lookup from hooks

**Files:**
- Modify: `hooks/stop.sh` (remove lines 25–34)
- Modify: `hooks/user-prompt-submit.sh` (remove lines 28–37)
- Modify: `hooks/pre-tool-use.sh` (remove lines 26–31)

- [ ] **Step 1: Remove Keychain block from `hooks/stop.sh`**

Remove the block:
```bash
# Platform-aware OAuth token retrieval (DES-3)
TOKEN=""
if [[ "$(uname)" == "darwin" ]]; then
    TOKEN=$(security find-generic-password \
        -s "Claude Code-credentials" -w 2>/dev/null \
        | python3 -c \
        "import sys,json; print(json.load(sys.stdin)['claudeAiOauth']['accessToken'])" \
        2>/dev/null) || true
fi
export ENGRAM_API_TOKEN="${TOKEN:-${ENGRAM_API_TOKEN:-}}"
```

- [ ] **Step 2: Remove Keychain block from `hooks/user-prompt-submit.sh`**

Same block, remove it.

- [ ] **Step 3: Remove Keychain block from `hooks/pre-tool-use.sh`**

Same block (single-line variant), remove it.

- [ ] **Step 4: Verify hooks still work**

Run: `bash -n hooks/stop.sh && bash -n hooks/user-prompt-submit.sh && bash -n hooks/pre-tool-use.sh`
Expected: No syntax errors.

- [ ] **Step 5: Commit**

```
refactor(hooks): remove duplicated Keychain token sourcing

The Go binary now resolves its own OAuth token via
internal/tokenresolver, so hooks no longer need to do it.

Closes #363
```
