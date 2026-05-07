# Companion Evaluation Cycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace Anthropic Haiku with a pluggable `--llm-cmd` backend, introduce `engram cycle` as a per-turn evaluation orchestrator, switch recall to always-synthesize prose reports, and broaden learn-time dedup with deterministic slug auto-increment.

**Architecture:** New `internal/llmcmd` package implements the existing `Extractor` / `FindingSummarizer` / `llmCaller` interfaces by spawning a configurable shell command (stdin prompt → stdout response). Existing recall + learn pipelines route through this new backend; Anthropic is deleted. New `internal/cycle` package + `cycle` CLI subcommand drive a two-LLM-call orchestration that the OpenCode plugin shells out to instead of doing TS-side companion logic.

**Tech Stack:** Go 1.22+, gomega for assertions, manual fakes for DI testing, targ build runner, BurntSushi/toml for memory storage, Bun-runtime TypeScript for the plugin.

**Spec:** `docs/superpowers/specs/2026-05-05-companion-evaluation-cycle-design.md`

---

## Phase A — `llmcmd` package foundation

The new package wraps `--llm-cmd` invocation. It must implement three things engram already depends on:
1. `recall.Extractor` and `recall.FindingSummarizer` (from `internal/recall/orchestrate.go`).
2. `cli.llmCaller` signature `func(ctx, model, system, user) (string, error)` (from `internal/cli/learn.go`).

The `model` parameter is ignored (the model is encoded in the cmd string).

### Task A1: Skeleton + stdin/stdout exec

**Files:**
- Create: `internal/llmcmd/llmcmd.go`
- Create: `internal/llmcmd/llmcmd_test.go`
- Create: `internal/llmcmd/export_test.go`

- [ ] **Step 1: Write the failing test**

```go
// internal/llmcmd/llmcmd_test.go
package llmcmd_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/llmcmd"
)

func TestRun_PipesPromptToStdinAndReturnsStdout(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// `cat` echoes stdin to stdout — perfect filter for testing.
	runner := llmcmd.New("cat")

	out, err := runner.Run(context.Background(), "hello world")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("hello world"))
}

func TestRun_NonZeroExitReturnsError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := llmcmd.New("false")

	_, err := runner.Run(context.Background(), "anything")
	g.Expect(err).To(MatchError(ContainSubstring("llm-cmd exited")))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test ./internal/llmcmd/...`
Expected: FAIL with build error (package doesn't exist yet).

- [ ] **Step 3: Write minimal implementation**

```go
// internal/llmcmd/llmcmd.go
package llmcmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

const (
	defaultShell = "/bin/sh"
)

// Runner spawns a shell command, pipes the prompt to stdin, returns stdout.
type Runner struct {
	cmdString string
}

// New returns a Runner that invokes cmdString via /bin/sh -c when called.
func New(cmdString string) *Runner {
	return &Runner{cmdString: cmdString}
}

// Run pipes prompt to the command's stdin and returns trimmed stdout.
func (r *Runner) Run(ctx context.Context, prompt string) (string, error) {
	cmd := exec.CommandContext(ctx, defaultShell, "-c", r.cmdString)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("llm-cmd exited: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimRight(stdout.String(), "\n"), nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test ./internal/llmcmd/...`
Expected: PASS, both subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/llmcmd/
git commit -m "feat(llmcmd): stdin/stdout shell-cmd runner"
```

### Task A2: Wall-clock timeout

**Files:**
- Modify: `internal/llmcmd/llmcmd.go`
- Modify: `internal/llmcmd/llmcmd_test.go`

- [ ] **Step 1: Write the failing test**

Append to `llmcmd_test.go`:

```go
func TestRun_TimeoutKillsProcess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := llmcmd.NewWithTimeout("sleep 60", 50*time.Millisecond)

	start := time.Now()
	_, err := runner.Run(context.Background(), "irrelevant")
	elapsed := time.Since(start)

	g.Expect(err).To(MatchError(ContainSubstring("timeout")))
	g.Expect(elapsed).To(BeNumerically("<", 5*time.Second))
}
```

Add `"time"` import.

- [ ] **Step 2: Run test, verify failure**

Run: `targ test ./internal/llmcmd/...`
Expected: FAIL — `NewWithTimeout` undefined.

- [ ] **Step 3: Implement timeout**

In `internal/llmcmd/llmcmd.go`:

```go
const (
	defaultShell      = "/bin/sh"
	defaultTimeout    = 60 * time.Second
)

type Runner struct {
	cmdString string
	timeout   time.Duration
}

// New returns a Runner with the default 60s timeout.
func New(cmdString string) *Runner {
	return NewWithTimeout(cmdString, defaultTimeout)
}

// NewWithTimeout sets a custom wall-clock timeout.
func NewWithTimeout(cmdString string, timeout time.Duration) *Runner {
	return &Runner{cmdString: cmdString, timeout: timeout}
}

func (r *Runner) Run(ctx context.Context, prompt string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, defaultShell, "-c", r.cmdString)
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
		return "", fmt.Errorf("llm-cmd timeout after %s: %w", r.timeout, timeoutCtx.Err())
	}

	if err != nil {
		return "", fmt.Errorf("llm-cmd exited: %w (stderr: %s)", err, stderr.String())
	}

	return strings.TrimRight(stdout.String(), "\n"), nil
}
```

Add imports `"errors"` and `"time"`.

- [ ] **Step 4: Verify tests pass**

Run: `targ test ./internal/llmcmd/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/llmcmd/
git commit -m "feat(llmcmd): wall-clock timeout"
```

### Task A3: Recursion-guard env-var injection

When engram spawns `--llm-cmd` (which often re-enters opencode loading the engram plugin), the plugin must be able to short-circuit to avoid spawning a companion of its own. The runner sets `ENGRAM_COMPANION_MODE=1` in the spawned process's environment.

- [ ] **Step 1: Write failing test**

Append to `llmcmd_test.go`:

```go
func TestRun_SetsRecursionGuardEnvVar(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Print ENGRAM_COMPANION_MODE — confirms it was passed to child.
	runner := llmcmd.New(`printf '%s' "$ENGRAM_COMPANION_MODE"`)

	out, err := runner.Run(context.Background(), "")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(Equal("1"))
}
```

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/llmcmd/...`
Expected: FAIL — empty output, env var not set.

- [ ] **Step 3: Implement env-var injection**

In `Run`, before `cmd.Run()`:

```go
cmd.Env = append(os.Environ(), "ENGRAM_COMPANION_MODE=1")
```

Add `"os"` import.

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/llmcmd/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/llmcmd/
git commit -m "feat(llmcmd): inject ENGRAM_COMPANION_MODE=1 for recursion guard"
```

### Task A4: Implement `Extractor` and `FindingSummarizer`

Recall's interfaces use `(model, system, user)` style. Llmcmd merges system+user into a single prompt for the underlying cmd.

**Files:**
- Create: `internal/llmcmd/extractor.go`
- Create: `internal/llmcmd/extractor_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/llmcmd/extractor_test.go
package llmcmd_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/llmcmd"
)

func TestExtractor_ExtractRelevant_PromptIncludesContentAndQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Echo the entire prompt back so we can inspect it.
	ext := llmcmd.NewExtractor(llmcmd.New("cat"))

	out, err := ext.ExtractRelevant(context.Background(), "the content body", "the query body")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("the content body"))
	g.Expect(out).To(ContainSubstring("the query body"))
	g.Expect(out).To(ContainSubstring("Extract only content relevant"))
}

func TestExtractor_SummarizeFindings_PromptIncludesBufferAndQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := llmcmd.NewExtractor(llmcmd.New("cat"))

	out, err := ext.SummarizeFindings(context.Background(), "buffer contents", "the query")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("buffer contents"))
	g.Expect(out).To(ContainSubstring("the query"))
}
```

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/llmcmd/...`
Expected: FAIL — `NewExtractor` undefined.

- [ ] **Step 3: Implement**

```go
// internal/llmcmd/extractor.go
package llmcmd

import (
	"context"
	"fmt"
)

const (
	extractRelevantSystemPrompt = `Extract only content relevant to the following query. ` +
		`Return relevant excerpts verbatim or very lightly paraphrased in service of grammatical ` +
		`correctness and consistency. Return nothing if irrelevant.`
)

// Extractor implements recall.Extractor and recall.FindingSummarizer
// by composing a single-prompt call through the underlying Runner.
type Extractor struct {
	runner *Runner
}

// NewExtractor wires a Runner into the Extractor adapter.
func NewExtractor(runner *Runner) *Extractor {
	return &Extractor{runner: runner}
}

// ExtractRelevant composes the existing extract prompt and calls the runner.
func (e *Extractor) ExtractRelevant(ctx context.Context, content, query string) (string, error) {
	prompt := fmt.Sprintf(
		"%s\n\nQuery: %s\n\nContent:\n%s",
		extractRelevantSystemPrompt, query, content,
	)

	out, err := e.runner.Run(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("extracting relevant: %w", err)
	}

	return out, nil
}

// SummarizeFindings is wired to the new synthesis prompt added in Phase C.
// For now it uses a temporary shape so phase A is testable in isolation.
func (e *Extractor) SummarizeFindings(ctx context.Context, content, query string) (string, error) {
	prompt := fmt.Sprintf(
		"Synthesize the following findings into a coherent report.\n\n"+
			"Query: %s\n\nFindings:\n%s",
		query, content,
	)

	out, err := e.runner.Run(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summarizing findings: %w", err)
	}

	return out, nil
}
```

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/llmcmd/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/llmcmd/
git commit -m "feat(llmcmd): implement Extractor and FindingSummarizer adapters"
```

### Task A5: Implement `llmCaller` for learn dedup

The dedup path in `internal/cli/learn.go:45` has a different shape: `func(ctx, model, system, user) (string, error)`. Llmcmd merges them.

- [ ] **Step 1: Write failing test**

```go
// internal/llmcmd/dedup_test.go
package llmcmd_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/llmcmd"
)

func TestCallerFunc_PromptIncludesSystemAndUser(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	call := llmcmd.CallerFunc(llmcmd.New("cat"))

	out, err := call(context.Background(), "ignored-model", "system part", "user part")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("system part"))
	g.Expect(out).To(ContainSubstring("user part"))
}
```

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/llmcmd/...`
Expected: FAIL — `CallerFunc` undefined.

- [ ] **Step 3: Implement**

```go
// internal/llmcmd/dedup.go
package llmcmd

import (
	"context"
	"fmt"
)

// CallerFunc returns a function with the signature expected by
// internal/cli/learn.go's llmCaller — model is ignored, system+user are
// concatenated into a single prompt and run through the shell command.
func CallerFunc(runner *Runner) func(context.Context, string, string, string) (string, error) {
	return func(ctx context.Context, _model, system, user string) (string, error) {
		prompt := system + "\n\n" + user

		out, err := runner.Run(ctx, prompt)
		if err != nil {
			return "", fmt.Errorf("calling llm-cmd: %w", err)
		}

		return out, nil
	}
}
```

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/llmcmd/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/llmcmd/
git commit -m "feat(llmcmd): CallerFunc for learn dedup"
```

---

## Phase B — Wire `--llm-cmd` flag through CLI

### Task B1: Add `--llm-cmd` flag with env-var fallback

**Files:**
- Modify: `internal/cli/targets.go`
- Modify: `internal/cli/cli.go`

- [ ] **Step 1: Read existing flag definitions**

```bash
grep -n "targ:\"flag" internal/cli/targets.go | head -20
```

Note the registration pattern (struct tags).

- [ ] **Step 2: Write failing test**

```go
// internal/cli/cli_test.go (new test, not file)
func TestResolveLLMCmd_PrefersFlagOverEnv(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "from-env")

	got := cli.ExportResolveLLMCmd("from-flag")
	g.Expect(got).To(Equal("from-flag"))
}

func TestResolveLLMCmd_FallsBackToEnv(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "from-env")

	got := cli.ExportResolveLLMCmd("")
	g.Expect(got).To(Equal("from-env"))
}

func TestResolveLLMCmd_EmptyWhenNeitherSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	got := cli.ExportResolveLLMCmd("")
	g.Expect(got).To(Equal(""))
}
```

Add to `internal/cli/export_test.go`: `var ExportResolveLLMCmd = resolveLLMCmd`.

- [ ] **Step 3: Verify failure**

Run: `targ test ./internal/cli/...`
Expected: FAIL — `resolveLLMCmd` undefined.

- [ ] **Step 4: Implement**

In `internal/cli/cli.go`, add:

```go
const (
	envLLMCmd = "ENGRAM_LLM_CMD"
)

// resolveLLMCmd returns the explicit flag value if set, otherwise the
// ENGRAM_LLM_CMD env var, otherwise the empty string.
func resolveLLMCmd(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	return os.Getenv(envLLMCmd)
}
```

In `internal/cli/targets.go`, add a `LLMCmd string` field to the relevant args structs (start with the shared base if there is one; otherwise add to each command's args):

```go
LLMCmd string `targ:"flag,name=llm-cmd,desc=command to invoke for LLM calls (overrides ENGRAM_LLM_CMD)"`
```

- [ ] **Step 5: Verify tests pass**

Run: `targ test ./internal/cli/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/
git commit -m "feat(cli): add --llm-cmd flag with ENGRAM_LLM_CMD fallback"
```

### Task B2: Required-flag check for cycle/recall/learn

When neither `--llm-cmd` nor `ENGRAM_LLM_CMD` is set, the command must exit non-zero with a clear error.

- [ ] **Step 1: Write failing tests**

```go
func TestRequireLLMCmd_ErrorsWhenMissing(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	err := cli.ExportRequireLLMCmd("")
	g.Expect(err).To(MatchError(ContainSubstring("llm-cmd is required")))
	g.Expect(err).To(MatchError(ContainSubstring("ENGRAM_LLM_CMD")))
}

func TestRequireLLMCmd_OkWhenFlagSet(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	err := cli.ExportRequireLLMCmd("opencode run -m foo")
	g.Expect(err).NotTo(HaveOccurred())
}
```

Add to `export_test.go`: `var ExportRequireLLMCmd = requireLLMCmd`.

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/cli/...`
Expected: FAIL — `requireLLMCmd` undefined.

- [ ] **Step 3: Implement**

In `internal/cli/cli.go`:

```go
var errLLMCmdRequired = errors.New(
	"llm-cmd is required: set --llm-cmd flag or ENGRAM_LLM_CMD environment variable",
)

func requireLLMCmd(flagValue string) error {
	if resolveLLMCmd(flagValue) == "" {
		return errLLMCmdRequired
	}

	return nil
}
```

Add `"errors"` import if missing.

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/cli/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/
git commit -m "feat(cli): requireLLMCmd helper for non-empty enforcement"
```

### Task B3: Route recall through `llmcmd` instead of Anthropic

**Files:**
- Modify: `internal/cli/cli.go` (or wherever Recall is wired up — find it)

- [ ] **Step 1: Find current Anthropic wiring**

```bash
grep -n "anthropic" internal/cli/cli.go internal/cli/*.go
```

Identify the construction point where `recall.NewSummarizer` is given the Anthropic client.

- [ ] **Step 2: Write failing integration test**

In `internal/cli/recall_test.go` (create if missing):

```go
package cli_test

import (
	"bytes"
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRecall_UsesLLMCmdRunner(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Plant: a fake "llm-cmd" that just echoes the prompt prefix back.
	args := cli.RecallArgs{
		Query:   "anything",
		LLMCmd:  `printf 'extracted-from-llm-cmd'`,
	}

	var stdout bytes.Buffer
	err := cli.RunRecall(context.Background(), args, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("extracted-from-llm-cmd"))
}
```

(Adjust the args struct name to match what already exists in `internal/cli/`.)

- [ ] **Step 3: Verify failure**

Run: `targ test ./internal/cli/...`
Expected: FAIL — Anthropic client still wired in.

- [ ] **Step 4: Replace Anthropic client with llmcmd in recall wiring**

In `internal/cli/cli.go` (or the file currently constructing recall summarizer), replace:

```go
// before:
caller := makeAnthropicCaller(token)
summarizer := recall.NewSummarizer(caller)
```

with:

```go
// after:
runner := llmcmd.New(resolveLLMCmd(args.LLMCmd))
summarizer := llmcmd.NewExtractor(runner)
```

(The exact lines depend on current shape — locate the construction point and swap the implementation. The recall package's `Extractor` and `FindingSummarizer` interfaces are the seam.)

- [ ] **Step 5: Verify pass**

Run: `targ test ./internal/cli/... ./internal/recall/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/
git commit -m "refactor(cli): route recall through llmcmd backend"
```

### Task B4: Route learn dedup through `llmcmd`

- [ ] **Step 1: Find current learn dedup wiring**

```bash
grep -n "makeAnthropicCaller\|makeConflictDeps" internal/cli/learn.go
```

- [ ] **Step 2: Write failing test**

```go
func TestLearn_DedupRunsThroughLLMCmd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// llm-cmd that always returns NONE — write should succeed.
	dataDir := t.TempDir()
	args := cli.LearnFeedbackArgs{
		Situation: "x",
		Behavior:  "y",
		Impact:    "z",
		Action:    "w",
		Source:    "agent",
		DataDir:   dataDir,
		LLMCmd:    `printf 'NONE'`,
	}

	var stdout bytes.Buffer
	err := cli.RunLearnFeedback(context.Background(), args, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(stdout.String()).To(ContainSubstring("CREATED:"))
}
```

- [ ] **Step 3: Verify failure**

Run: `targ test ./internal/cli/...`
Expected: FAIL — Anthropic still wired.

- [ ] **Step 4: Replace `makeConflictDeps` and `makeAnthropicCaller`**

In `internal/cli/learn.go`, replace `makeConflictDeps`:

```go
// makeConflictDeps wires real I/O deps for conflict detection using llm-cmd.
func makeConflictDeps(llmCmdString string) (llmCaller, memoryLister) {
	if llmCmdString == "" {
		return nil, memory.NewLister()
	}

	runner := llmcmd.New(llmCmdString)
	caller := llmcmd.CallerFunc(runner)

	return caller, memory.NewLister()
}
```

Update callers (`runLearnFact`, `runLearnFeedback`) to pass through `args.LLMCmd`:

```go
caller, lister := makeConflictDeps(resolveLLMCmd(args.LLMCmd))
```

Add `"engram/internal/llmcmd"` import. Remove `resolveToken`, `makeAnthropicCaller`.

- [ ] **Step 5: Verify pass**

Run: `targ test ./internal/cli/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/
git commit -m "refactor(learn): route dedup through llmcmd backend"
```

---

## Phase G — Delete Anthropic package

### Task G1: Delete `internal/anthropic/` and verify

- [ ] **Step 1: Confirm no remaining imports**

```bash
grep -rn "engram/internal/anthropic" internal/ cmd/
```

Expected: zero matches. If any remain, fix them before deletion.

- [ ] **Step 2: Delete the package**

```bash
git rm -r internal/anthropic/
```

- [ ] **Step 3: Verify build + tests**

```bash
targ check-full
```

Expected: PASS — no references to the deleted package.

- [ ] **Step 4: Commit**

```bash
git commit -m "refactor: remove internal/anthropic package"
```

---

## Phase C — Recall pipeline updates

### Task C1: Collapse `Result` to single `Report` field

**Files:**
- Modify: `internal/recall/orchestrate.go`
- Modify: `internal/recall/recall_test.go`
- Modify: `internal/recall/orchestrate_test.go`
- Modify: any callers reading `Result.Summary` or `Result.Memories`

- [ ] **Step 1: Find all callers**

```bash
grep -rn "Result.Summary\|\.Memories\b" internal/ cmd/ | grep -v _test.go
```

- [ ] **Step 2: Update tests first to expect `Report`**

In `internal/recall/orchestrate_test.go`, replace assertions like `result.Summary` with `result.Report`. Tests should now fail.

- [ ] **Step 3: Verify failure**

Run: `targ test ./internal/recall/...`
Expected: FAIL — `Report` field doesn't exist.

- [ ] **Step 4: Update `Result` struct**

```go
// internal/recall/orchestrate.go
type Result struct {
	Report string `json:"report"`
}
```

Update `FormatResult`:

```go
func FormatResult(w io.Writer, result *Result) error {
	_, err := fmt.Fprint(w, result.Report)
	if err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	return nil
}
```

Update internal use sites in `recallModeA` / `recallModeB` to populate `Report` instead of `Summary` / `Memories`. Bare mode initially returns the existing concatenation as `Report` — Phase C2 swaps it to synthesis.

- [ ] **Step 5: Verify tests pass**

Run: `targ test ./internal/recall/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/recall/
git commit -m "refactor(recall): collapse Result to single Report field"
```

### Task C2: New synthesis prompt

**Files:**
- Modify: `internal/recall/summarize.go`
- Modify: `internal/recall/summarize_test.go`
- Modify: `internal/llmcmd/extractor.go` (the temporary `SummarizeFindings` body)

- [ ] **Step 1: Write failing test**

In `internal/llmcmd/extractor_test.go`, add:

```go
func TestExtractor_SummarizeFindings_RequestsDirectiveAdvice(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	ext := llmcmd.NewExtractor(llmcmd.New("cat"))

	out, err := ext.SummarizeFindings(context.Background(), "sources here", "the topic")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out).To(ContainSubstring("directive advice"))
	g.Expect(out).To(ContainSubstring("imperative voice"))
	g.Expect(out).To(ContainSubstring("cite the specific memory or outcome"))
}
```

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/llmcmd/...`
Expected: FAIL — current prompt doesn't include those phrases.

- [ ] **Step 3: Replace synthesis prompt**

In `internal/llmcmd/extractor.go`:

```go
const synthesisPromptHeader = `You are synthesizing engram memory sources into a coherent report for an AI agent.

The sources include facts, behavioral feedback, action records, and outcomes drawn from prior project work. Weave them into a narrative that captures what has been learned and tried.

Then end with directive advice — concrete instructions, warnings, or constraints the reader must apply going forward. Use imperative voice ("Do X", "Avoid Y", "Verify Z before W"). Cite the specific memory or outcome that grounds each piece of advice. Do not hedge with "consider", "you might", or "think about" — issue clear guidance derived from prior evidence.

Output the report only — no preamble, no list of sources, no JSON.`

func (e *Extractor) SummarizeFindings(ctx context.Context, content, query string) (string, error) {
	prompt := synthesisPromptHeader + "\n\n"

	if query != "" {
		prompt += "Focus on material relevant to: " + query + "\n\n"
	}

	prompt += "Sources:\n" + content

	out, err := e.runner.Run(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("summarizing findings: %w", err)
	}

	return out, nil
}
```

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/llmcmd/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/llmcmd/
git commit -m "feat(llmcmd): synthesis prompt demands directive advice"
```

### Task C3: Bare-mode (no-query) goes through synthesis

**Files:**
- Modify: `internal/recall/orchestrate.go`
- Modify: `internal/recall/orchestrate_test.go`

- [ ] **Step 1: Write failing test**

In `internal/recall/orchestrate_test.go`:

```go
func TestRecallModeA_RunsSynthesisOverTranscriptsAndMemories(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Fake summarizer captures inputs.
	fakeSum := &fakeSummarizer{summarizeOut: "synthesized prose"}
	finder := &fakeFinder{entries: []recall.FileEntry{{Path: "/tmp/x", Mtime: time.Now()}}}
	reader := &fakeReader{content: "USER: hi\nASSISTANT: hello", size: 25}

	orch := recall.NewOrchestrator(finder, reader, fakeSum, nil, "")

	result, err := orch.Recall(context.Background(), "")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(fakeSum.summarizeCalled).To(BeTrue())
	g.Expect(result.Report).To(Equal("synthesized prose"))
}
```

(Use existing fake patterns from the test file.)

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/recall/...`
Expected: FAIL — bare mode does not call summarizer today.

- [ ] **Step 3: Update `recallModeA`**

```go
func (o *Orchestrator) recallModeA(
	ctx context.Context,
	sessions []FileEntry,
) (*Result, error) {
	var builder strings.Builder

	bytesRead := 0

	for _, entry := range sessions {
		if ctx.Err() != nil {
			break
		}

		content, size, readErr := o.reader.Read(entry.Path, DefaultModeABudget-bytesRead)
		if readErr != nil {
			continue
		}

		builder.WriteString(content)

		bytesRead += size
		if bytesRead >= DefaultModeABudget {
			break
		}
	}

	memories := o.findSessionMemories(sessions)
	if memories != "" {
		builder.WriteString("\n\n=== MEMORIES ===\n")
		builder.WriteString(memories)
	}

	if builder.Len() == 0 {
		return &Result{}, nil
	}

	report, err := o.summarizer.SummarizeFindings(ctx, builder.String(), "")
	if err != nil {
		return nil, fmt.Errorf("synthesizing bare-mode recall: %w", err)
	}

	return &Result{Report: report}, nil
}
```

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/recall/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/recall/
git commit -m "feat(recall): bare mode runs synthesis"
```

---

## Phase D — Learn pipeline updates

### Task D1: Broaden `BuildIndex` to include content fields

**Files:**
- Modify: `internal/memory/memory.go`
- Modify: `internal/memory/memory_test.go`

- [ ] **Step 1: Read existing `BuildIndex`**

```bash
grep -n "func BuildIndex" internal/memory/memory.go
```

- [ ] **Step 2: Write failing test**

In `internal/memory/memory_test.go`:

```go
func TestBuildIndex_IncludesContentFields(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	mems := []*memory.Stored{
		{
			Type:      "feedback",
			Situation: "doing X",
			FilePath:  "/tmp/feedback/doing-x.toml",
			Content: memory.ContentFields{
				Behavior: "behaved poorly",
				Impact:   "bad outcome",
				Action:   "do better",
			},
		},
		{
			Type:      "fact",
			Situation: "knowing Y",
			FilePath:  "/tmp/fact/knowing-y.toml",
			Content: memory.ContentFields{
				Subject:   "Y",
				Predicate: "is",
				Object:    "true",
			},
		},
	}

	idx := memory.BuildIndex(mems)
	g.Expect(idx).To(ContainSubstring("behaved poorly"))
	g.Expect(idx).To(ContainSubstring("bad outcome"))
	g.Expect(idx).To(ContainSubstring("do better"))
	g.Expect(idx).To(ContainSubstring("Y"))
	g.Expect(idx).To(ContainSubstring("is"))
	g.Expect(idx).To(ContainSubstring("true"))
}
```

- [ ] **Step 3: Verify failure**

Run: `targ test ./internal/memory/...`
Expected: FAIL — index doesn't include content fields today.

- [ ] **Step 4: Update `BuildIndex`**

In `internal/memory/memory.go`:

```go
func BuildIndex(memories []*Stored) string {
	var builder strings.Builder

	for _, mem := range memories {
		name := NameFromPath(mem.FilePath)

		fmt.Fprintf(&builder, "%s | %s | %s\n", mem.Type, name, mem.Situation)

		if mem.Type == "feedback" {
			if mem.Content.Behavior != "" {
				fmt.Fprintf(&builder, "  behavior: %s\n", mem.Content.Behavior)
			}
			if mem.Content.Impact != "" {
				fmt.Fprintf(&builder, "  impact: %s\n", mem.Content.Impact)
			}
			if mem.Content.Action != "" {
				fmt.Fprintf(&builder, "  action: %s\n", mem.Content.Action)
			}
		} else {
			if mem.Content.Subject != "" {
				fmt.Fprintf(&builder, "  subject: %s\n", mem.Content.Subject)
			}
			if mem.Content.Predicate != "" {
				fmt.Fprintf(&builder, "  predicate: %s\n", mem.Content.Predicate)
			}
			if mem.Content.Object != "" {
				fmt.Fprintf(&builder, "  object: %s\n", mem.Content.Object)
			}
		}
	}

	return builder.String()
}
```

- [ ] **Step 5: Verify pass**

Run: `targ test ./internal/memory/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/memory/
git commit -m "feat(memory): BuildIndex includes content fields for richer dedup"
```

### Task D2: Drop CONTRADICTION from dedup prompt and parser

**Files:**
- Modify: `internal/cli/learn.go`
- Modify: `internal/cli/learn_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestParseConflictResponse_IgnoresContradictionLines(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	got := cli.ExportParseConflictResponse("CONTRADICTION: foo", dataDir, &bytes.Buffer{})
	g.Expect(got).To(BeFalse())
}

func TestParseConflictResponse_RecognizesDuplicate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	got := cli.ExportParseConflictResponse("DUPLICATE: foo", dataDir, &bytes.Buffer{})
	g.Expect(got).To(BeTrue())
}
```

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/cli/...`
Expected: FAIL — contradiction is currently treated as a conflict.

- [ ] **Step 3: Update prompt and parser**

In `internal/cli/learn.go`:

```go
const (
	conflictDetectionSystemPrompt = `You are a memory deduplication checker. ` +
		`Given an index of existing memories and a new memory, determine if the new memory ` +
		`is a true duplicate of any existing one (matching on type, situation, AND content).

Respond with one of:
- "NONE" if no duplicates found
- "DUPLICATE: <name>" if it duplicates an existing memory (one per line)

Only output the result lines, nothing else.`
)

func parseConflictResponse(response, dataDir string, stdout io.Writer) bool {
	trimmed := strings.TrimSpace(response)

	if trimmed == "NONE" {
		return false
	}

	lines := strings.Split(trimmed, "\n")
	foundConflict := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "DUPLICATE:") {
			foundConflict = true
			parseConflictLine(line, dataDir, stdout)
		}
	}

	return foundConflict
}
```

Update `describeNewMemory` to include content fields (mirroring `BuildIndex` output).

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/cli/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/
git commit -m "refactor(learn): drop CONTRADICTION from dedup prompt and parser"
```

### Task D3: Slug auto-increment with `O_EXCL`

**Files:**
- Modify: `internal/tomlwriter/` (or wherever the file write happens)
- Modify: corresponding test

- [ ] **Step 1: Find current write logic**

```bash
grep -rn "func.*Write\b" internal/tomlwriter/
```

- [ ] **Step 2: Write failing test**

```go
// internal/tomlwriter/writer_test.go (or matching existing test file)
func TestWrite_AutoIncrementsOnSlugCollision(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	w := tomlwriter.New()

	rec := &memory.MemoryRecord{
		SchemaVersion: 2,
		Source:        "agent",
		Situation:     "shared situation",
		Type:          "feedback",
		Content: memory.ContentFields{Behavior: "first", Impact: "i", Action: "a"},
	}

	path1, err := w.Write(rec, "shared-situation", dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	rec2 := *rec
	rec2.Content.Behavior = "second-distinct"

	path2, err := w.Write(&rec2, "shared-situation", dataDir)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(path1).NotTo(Equal(path2))
	g.Expect(path2).To(HaveSuffix("-1.toml"))
}

func TestWrite_AutoIncrementIsRaceFree(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	w := tomlwriter.New()

	const concurrent = 10

	type res struct {
		path string
		err  error
	}

	results := make(chan res, concurrent)

	for i := 0; i < concurrent; i++ {
		go func(i int) {
			rec := &memory.MemoryRecord{
				SchemaVersion: 2,
				Source:        "agent",
				Situation:     "race situation",
				Type:          "feedback",
				Content:       memory.ContentFields{Behavior: fmt.Sprintf("b%d", i)},
			}
			path, err := w.Write(rec, "race-situation", dataDir)
			results <- res{path, err}
		}(i)
	}

	paths := make(map[string]bool)
	for i := 0; i < concurrent; i++ {
		r := <-results
		g.Expect(r.err).NotTo(HaveOccurred())
		g.Expect(paths[r.path]).To(BeFalse())
		paths[r.path] = true
	}

	g.Expect(paths).To(HaveLen(concurrent))
}
```

- [ ] **Step 3: Verify failure**

Run: `targ test ./internal/tomlwriter/...`
Expected: FAIL — current writer overwrites or doesn't auto-increment race-free.

- [ ] **Step 4: Implement atomic auto-increment**

```go
// in tomlwriter/Write or equivalent:
func (w *Writer) Write(rec *memory.MemoryRecord, slug, dataDir string) (string, error) {
	dir := filepath.Join(dataDir, rec.Type)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating dir: %w", err)
	}

	candidate := slug
	for i := 0; ; i++ {
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", slug, i)
		}

		path := filepath.Join(dir, candidate+".toml")

		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if errors.Is(err, os.ErrExist) {
			continue
		}

		if err != nil {
			return "", fmt.Errorf("opening %s: %w", path, err)
		}

		// f acquired exclusively. Encode and close.
		encoder := toml.NewEncoder(f)
		if encErr := encoder.Encode(rec); encErr != nil {
			f.Close()
			os.Remove(path)
			return "", fmt.Errorf("encoding TOML: %w", encErr)
		}

		if closeErr := f.Close(); closeErr != nil {
			return "", fmt.Errorf("closing %s: %w", path, closeErr)
		}

		return path, nil
	}
}
```

- [ ] **Step 5: Verify pass**

Run: `targ test ./internal/tomlwriter/...`
Expected: PASS, including the race test (it must succeed even with concurrent goroutines).

- [ ] **Step 6: Commit**

```bash
git add internal/tomlwriter/
git commit -m "feat(tomlwriter): O_EXCL atomic auto-increment for slug collisions"
```

### Task D4: Refactor `writeMemory` to return name + persisted

The cycle command needs to know whether each candidate learning was actually written, and the slug name (post auto-increment). Today's `writeMemory` returns just `error`. This task changes the signature.

**Files:**
- Modify: `internal/cli/learn.go`
- Modify: `internal/cli/learn_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestWriteMemory_ReturnsNameAndPersisted_OnSuccess(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	rec := &memory.MemoryRecord{
		SchemaVersion: 2,
		Source:        "agent",
		Situation:     "test situation",
		Type:          "feedback",
		Content:       memory.ContentFields{Behavior: "b", Impact: "i", Action: "a"},
	}

	name, persisted, err := cli.ExportWriteMemory(
		context.Background(),
		rec, "test situation",
		&dataDir, true, // noDupCheck
		&bytes.Buffer{}, "test", nil, nil,
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(persisted).To(BeTrue())
	g.Expect(name).To(Equal("test-situation"))
}

func TestWriteMemory_ReturnsNotPersisted_OnDuplicate(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	dupCaller := func(_ context.Context, _, _, _ string) (string, error) {
		return "DUPLICATE: existing", nil
	}

	// Plant an existing memory so the dedup detector has something to match.
	planted := &memory.MemoryRecord{
		SchemaVersion: 2, Source: "agent", Situation: "existing",
		Type: "feedback", Content: memory.ContentFields{Behavior: "b1"},
	}
	_, _, _ = cli.ExportWriteMemory(
		context.Background(),
		planted, "existing", &dataDir, true,
		&bytes.Buffer{}, "test", nil, nil,
	)

	rec := &memory.MemoryRecord{
		SchemaVersion: 2, Source: "agent", Situation: "different",
		Type: "feedback", Content: memory.ContentFields{Behavior: "b2"},
	}

	name, persisted, err := cli.ExportWriteMemory(
		context.Background(),
		rec, "different",
		&dataDir, false,
		&bytes.Buffer{}, "test", dupCaller, memory.NewLister(),
	)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(persisted).To(BeFalse())
	g.Expect(name).To(BeEmpty())
}
```

Add to `export_test.go`: `var ExportWriteMemory = writeMemory`.

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/cli/...`
Expected: FAIL — old signature returns just `error`.

- [ ] **Step 3: Update signature and implementation**

In `internal/cli/learn.go`:

```go
func writeMemory(
	ctx context.Context,
	record *memory.MemoryRecord,
	situation string,
	dataDir *string,
	noDupCheck bool,
	stdout io.Writer,
	cmdName string,
	caller llmCaller,
	lister memoryLister,
) (string, bool, error) {
	defaultErr := applyDataDirDefault(dataDir)
	if defaultErr != nil {
		return "", false, fmt.Errorf("%s: %w", cmdName, defaultErr)
	}

	slug := tomlwriter.Slugify(situation)

	if !noDupCheck {
		conflict, checkErr := checkForConflicts(ctx, record, *dataDir, stdout, caller, lister)
		if checkErr != nil {
			return "", false, fmt.Errorf("%s: %w", cmdName, checkErr)
		}

		if conflict {
			return "", false, nil
		}
	}

	writer := tomlwriter.New()

	filePath, writeErr := writer.Write(record, slug, *dataDir)
	if writeErr != nil {
		return "", false, fmt.Errorf("%s: %w", cmdName, writeErr)
	}

	name := memory.NameFromPath(filePath)

	_, printErr := fmt.Fprintf(stdout, "CREATED: %s\n", name)
	if printErr != nil {
		return name, true, fmt.Errorf("%s: %w", cmdName, printErr)
	}

	return name, true, nil
}
```

Update callers in the same file (`runLearnFact`, `runLearnFeedback`) to discard the new returns:

```go
_, _, err := writeMemory(...)
return err
```

(Or keep them and do something useful — not needed for this task.)

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/cli/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/
git commit -m "refactor(learn): writeMemory returns (name, persisted, err)"
```

---

## Phase E — Cycle command

### Task E1: Cycle output schema

**Files:**
- Create: `internal/cycle/output.go`
- Create: `internal/cycle/output_test.go`

- [ ] **Step 1: Write failing test**

```go
package cycle_test

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cycle"
	"engram/internal/memory"
)

func TestOutput_MarshalsLearnedAndRecalled(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	out := cycle.Output{
		Learned: []cycle.LearnedMemory{
			{
				Name: "doing-x",
				Record: memory.MemoryRecord{
					SchemaVersion: 2,
					Source:        "agent",
					Situation:     "doing X",
					Type:          "feedback",
					Content:       memory.ContentFields{Behavior: "b", Impact: "i", Action: "a"},
				},
			},
		},
		Recalled: []cycle.RecalledReport{
			{Query: "q1", Report: "r1"},
		},
	}

	bs, err := json.Marshal(out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var roundtrip map[string]interface{}
	g.Expect(json.Unmarshal(bs, &roundtrip)).To(Succeed())
	g.Expect(roundtrip).To(HaveKey("learned"))
	g.Expect(roundtrip).To(HaveKey("recalled"))
}

func TestOutput_EmptyArraysWhenNothingHappened(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	// Use NewOutput so the slices are non-nil — JSON should render
	// "learned":[] not "learned":null.
	out := cycle.NewOutput()

	bs, err := json.Marshal(out)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(string(bs)).To(ContainSubstring(`"learned":[]`))
	g.Expect(string(bs)).To(ContainSubstring(`"recalled":[]`))
}
```

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/cycle/...`
Expected: FAIL — package doesn't exist.

- [ ] **Step 3: Implement**

```go
// internal/cycle/output.go
package cycle

import "engram/internal/memory"

// Output is the JSON shape returned by `engram cycle`.
type Output struct {
	Learned  []LearnedMemory  `json:"learned"`
	Recalled []RecalledReport `json:"recalled"`
}

// LearnedMemory is a memory record that was actually persisted by this cycle.
type LearnedMemory struct {
	Name string `json:"name"`
	memory.MemoryRecord
}

// RecalledReport is a query and the synthesized prose report it produced.
type RecalledReport struct {
	Query  string `json:"query"`
	Report string `json:"report"`
}

// NewOutput returns an Output with non-nil empty slices so JSON serializes
// "learned":[] / "recalled":[] rather than null.
func NewOutput() *Output {
	return &Output{
		Learned:  []LearnedMemory{},
		Recalled: []RecalledReport{},
	}
}
```

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/cycle/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cycle/
git commit -m "feat(cycle): JSON output schema"
```

### Task E2: Cycle prompts

**Files:**
- Create: `internal/cycle/prompts.go`
- Create: `internal/cycle/prompts_test.go`

- [ ] **Step 1: Write failing tests**

```go
package cycle_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cycle"
)

func TestLearnExtractionPrompt_IncludesTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cycle.LearnExtractionPrompt("the transcript body")
	g.Expect(prompt).To(ContainSubstring("the transcript body"))
	g.Expect(prompt).To(ContainSubstring("Output a JSON array"))
	g.Expect(prompt).To(ContainSubstring("feedback"))
	g.Expect(prompt).To(ContainSubstring("fact"))
}

func TestQueryProposalPrompt_IncludesTranscript(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	prompt := cycle.QueryProposalPrompt("transcript here")
	g.Expect(prompt).To(ContainSubstring("transcript here"))
	g.Expect(prompt).To(ContainSubstring("NO QUERIES"))
	g.Expect(prompt).To(ContainSubstring("1-5 targeted recall queries"))
}
```

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/cycle/...`
Expected: FAIL — funcs undefined.

- [ ] **Step 3: Implement**

```go
// internal/cycle/prompts.go
package cycle

const (
	learnExtractionHeader = `You are reviewing a project session transcript to identify learnings worth preserving.

Examine the transcript and propose any new learnings: corrections you observe, completed work that taught a lesson, decisions made, or facts established.

Output a JSON array of objects, each with:
- "type": "feedback" or "fact"
- "situation": short context phrase identifying when this applies
- For feedback: "behavior", "impact", "action"
- For fact: "subject", "predicate", "object"

Return [] if there is nothing learnable.

Transcript:
`

	queryProposalHeader = `You are reviewing a project session transcript to decide if memories should be recalled.

If the project is starting new research, taking new action, shifting approach, or otherwise embarking on something where prior memories could help, propose 1-5 targeted recall queries. Each query is 5-15 words capturing a specific facet to recall about.

Output one query per line, no numbering, no commentary.

If nothing in the transcript warrants recall, output exactly:
NO QUERIES

Transcript:
`
)

// LearnExtractionPrompt returns the LLM Call A prompt for the given transcript.
func LearnExtractionPrompt(transcript string) string {
	return learnExtractionHeader + transcript
}

// QueryProposalPrompt returns the LLM Call B prompt for the given transcript.
func QueryProposalPrompt(transcript string) string {
	return queryProposalHeader + transcript
}
```

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/cycle/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cycle/
git commit -m "feat(cycle): learn-extraction and query-proposal prompts"
```

### Task E3: Cycle orchestration

**Files:**
- Create: `internal/cycle/cycle.go`
- Create: `internal/cycle/cycle_test.go`

- [ ] **Step 1: Write failing tests**

```go
package cycle_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cycle"
)

// Hand-rolled fakes per project convention.

type fakeRunner struct {
	calls    []string
	responses []string
}

func (f *fakeRunner) Run(ctx context.Context, prompt string) (string, error) {
	idx := len(f.calls)
	f.calls = append(f.calls, prompt)
	if idx >= len(f.responses) {
		return "", nil
	}
	return f.responses[idx], nil
}

type fakeTranscript struct {
	content string
}

func (f *fakeTranscript) Read(projectDir string, budget int) (string, error) {
	return f.content, nil
}

func TestCycle_EmptyTranscriptReturnsEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := &fakeRunner{responses: []string{`[]`, `NO QUERIES`}}
	transcripts := &fakeTranscript{content: ""}

	c := cycle.New(runner, transcripts, nil, nil)

	out, err := c.Run(context.Background(), "/tmp/anything")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.Learned).To(BeEmpty())
	g.Expect(out.Recalled).To(BeEmpty())
}

func TestCycle_PersistsLearnedFromLLMResponseA(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	llmA := `[{"type":"feedback","situation":"doing X","behavior":"b","impact":"i","action":"a"}]`
	runner := &fakeRunner{responses: []string{llmA, "NO QUERIES"}}
	transcripts := &fakeTranscript{content: "USER: did X\nASSISTANT: ok"}

	persister := &fakePersister{}
	c := cycle.New(runner, transcripts, persister, nil)

	out, err := c.Run(context.Background(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(persister.feedbackCalls).To(HaveLen(1))
	g.Expect(persister.feedbackCalls[0].Situation).To(Equal("doing X"))
	g.Expect(out.Learned).To(HaveLen(1))
}

func TestCycle_RunsRecallPerProposedQuery(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := &fakeRunner{responses: []string{
		"[]",
		"query one\nquery two",
	}}
	transcripts := &fakeTranscript{content: "transcript"}
	recaller := &fakeRecaller{reports: map[string]string{
		"query one": "report one",
		"query two": "report two",
	}}

	c := cycle.New(runner, transcripts, nil, recaller)

	out, err := c.Run(context.Background(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.Recalled).To(HaveLen(2))
	g.Expect(out.Recalled[0].Query).To(Equal("query one"))
	g.Expect(out.Recalled[0].Report).To(Equal("report one"))
}

// fakePersister and fakeRecaller defined as test helpers (mirror writeMemory and recall.Run signatures).
```

(Define `fakePersister` and `fakeRecaller` to satisfy the new package's interfaces.)

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/cycle/...`
Expected: FAIL — `cycle.New` undefined.

- [ ] **Step 3: Implement orchestrator**

```go
// internal/cycle/cycle.go
package cycle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"engram/internal/memory"
)

// Runner runs a single LLM prompt and returns the response text.
type Runner interface {
	Run(ctx context.Context, prompt string) (string, error)
}

// TranscriptReader returns the recent project transcript under a budget.
type TranscriptReader interface {
	Read(projectDir string, budget int) (string, error)
}

// Persister persists a candidate learning. Returns the slug-name written
// (post auto-increment) and whether dedup skipped it.
type Persister interface {
	WriteFeedback(ctx context.Context, situation, behavior, impact, action string) (name string, persisted bool, err error)
	WriteFact(ctx context.Context, situation, subject, predicate, object string) (name string, persisted bool, err error)
}

// Recaller runs the existing recall pipeline for a single query.
type Recaller interface {
	Recall(ctx context.Context, projectDir, query string) (report string, err error)
}

// Cycle orchestrates a single per-turn evaluation cycle.
type Cycle struct {
	runner      Runner
	transcripts TranscriptReader
	persister   Persister
	recaller    Recaller
	budget      int
}

const (
	defaultTranscriptBudget = 15 * 1024
	maxQueries              = 5
	noQueriesSentinel       = "NO QUERIES"
)

// New wires a Cycle. Any of persister/recaller may be nil for partial use.
func New(runner Runner, transcripts TranscriptReader, persister Persister, recaller Recaller) *Cycle {
	return &Cycle{
		runner:      runner,
		transcripts: transcripts,
		persister:   persister,
		recaller:    recaller,
		budget:      defaultTranscriptBudget,
	}
}

type learnCandidate struct {
	Type      string `json:"type"`
	Situation string `json:"situation"`
	Behavior  string `json:"behavior,omitempty"`
	Impact    string `json:"impact,omitempty"`
	Action    string `json:"action,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Predicate string `json:"predicate,omitempty"`
	Object    string `json:"object,omitempty"`
}

// Run executes one cycle: extract → persist → propose queries → per-query recall.
func (c *Cycle) Run(ctx context.Context, projectDir string) (*Output, error) {
	out := NewOutput()

	transcript, err := c.transcripts.Read(projectDir, c.budget)
	if err != nil {
		return out, fmt.Errorf("reading transcript: %w", err)
	}

	c.runLearningStep(ctx, transcript, out)
	c.runRecallStep(ctx, transcript, projectDir, out)

	return out, nil
}

func (c *Cycle) runLearningStep(ctx context.Context, transcript string, out *Output) {
	if c.persister == nil {
		return
	}

	resp, err := c.runner.Run(ctx, LearnExtractionPrompt(transcript))
	if err != nil {
		return
	}

	candidates, parseErr := parseLearnCandidates(resp)
	if parseErr != nil {
		return
	}

	for _, cand := range candidates {
		c.persistOne(ctx, cand, out)
	}
}

func (c *Cycle) persistOne(ctx context.Context, cand learnCandidate, out *Output) {
	switch cand.Type {
	case "feedback":
		name, ok, err := c.persister.WriteFeedback(ctx, cand.Situation, cand.Behavior, cand.Impact, cand.Action)
		if err != nil || !ok {
			return
		}
		out.Learned = append(out.Learned, LearnedMemory{
			Name: name,
			MemoryRecord: memory.MemoryRecord{
				Type:      "feedback",
				Situation: cand.Situation,
				Source:    "agent",
				Content: memory.ContentFields{
					Behavior: cand.Behavior,
					Impact:   cand.Impact,
					Action:   cand.Action,
				},
			},
		})

	case "fact":
		name, ok, err := c.persister.WriteFact(ctx, cand.Situation, cand.Subject, cand.Predicate, cand.Object)
		if err != nil || !ok {
			return
		}
		out.Learned = append(out.Learned, LearnedMemory{
			Name: name,
			MemoryRecord: memory.MemoryRecord{
				Type:      "fact",
				Situation: cand.Situation,
				Source:    "agent",
				Content: memory.ContentFields{
					Subject:   cand.Subject,
					Predicate: cand.Predicate,
					Object:    cand.Object,
				},
			},
		})
	}
}

func (c *Cycle) runRecallStep(ctx context.Context, transcript, projectDir string, out *Output) {
	if c.recaller == nil {
		return
	}

	resp, err := c.runner.Run(ctx, QueryProposalPrompt(transcript))
	if err != nil {
		return
	}

	queries := parseQueries(resp)
	for _, q := range queries {
		report, recErr := c.recaller.Recall(ctx, projectDir, q)
		if recErr != nil || report == "" {
			continue
		}
		out.Recalled = append(out.Recalled, RecalledReport{
			Query:  q,
			Report: report,
		})
	}
}

func parseLearnCandidates(s string) ([]learnCandidate, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == "[]" {
		return nil, nil
	}

	var out []learnCandidate
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return nil, fmt.Errorf("parsing learn candidates: %w", err)
	}

	return out, nil
}

func parseQueries(s string) []string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" || trimmed == noQueriesSentinel {
		return nil
	}

	lines := strings.Split(trimmed, "\n")
	queries := make([]string, 0, len(lines))

	for _, line := range lines {
		q := strings.TrimSpace(line)
		if q == "" || q == noQueriesSentinel {
			continue
		}
		queries = append(queries, q)
		if len(queries) >= maxQueries {
			break
		}
	}

	return queries
}
```

- [ ] **Step 4: Verify pass**

Run: `targ test ./internal/cycle/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cycle/
git commit -m "feat(cycle): orchestrator with learning extraction and query-driven recall"
```

### Task E4: Cycle CLI command

**Files:**
- Create: `internal/cli/cycle.go`
- Create: `internal/cli/cycle_test.go`
- Modify: `internal/cli/targets.go` (register cycle target)

- [ ] **Step 1: Write failing test**

```go
package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestRunCycle_RequiresLLMCmd(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	t.Setenv("ENGRAM_LLM_CMD", "")

	args := cli.CycleArgs{ProjectDir: "/tmp/x"}

	var stdout bytes.Buffer
	err := cli.RunCycle(context.Background(), args, &stdout)
	g.Expect(err).To(MatchError(ContainSubstring("llm-cmd is required")))
}

func TestRunCycle_EmitsValidJSONWithEmptyArrays(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()

	// Sentinel responses: empty learnings, no queries.
	cmdString := `case "$(cat)" in *learnings*) printf '[]';; *) printf 'NO QUERIES';; esac`

	args := cli.CycleArgs{
		ProjectDir: dir,
		LLMCmd:     cmdString,
		DataDir:    dir,
	}

	var stdout bytes.Buffer
	err := cli.RunCycle(context.Background(), args, &stdout)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	var decoded struct {
		Learned  []interface{} `json:"learned"`
		Recalled []interface{} `json:"recalled"`
	}
	g.Expect(json.Unmarshal(stdout.Bytes(), &decoded)).To(Succeed())
	g.Expect(decoded.Learned).To(BeEmpty())
	g.Expect(decoded.Recalled).To(BeEmpty())
}
```

- [ ] **Step 2: Verify failure**

Run: `targ test ./internal/cli/...`
Expected: FAIL — `RunCycle` and `CycleArgs` undefined.

- [ ] **Step 3: Implement**

```go
// internal/cli/cycle.go
package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"engram/internal/cycle"
	"engram/internal/llmcmd"
)

// CycleArgs holds the flag values for `engram cycle`.
type CycleArgs struct {
	LLMCmd           string `targ:"flag,name=llm-cmd,desc=LLM command (or use ENGRAM_LLM_CMD)"`
	ProjectDir       string `targ:"flag,name=project-dir,desc=project working directory"`
	DataDir          string `targ:"flag,name=data-dir,desc=engram data directory"`
	TranscriptBudget int    `targ:"flag,name=transcript-budget,desc=max bytes of transcript fed to LLM"`
}

// RunCycle executes one engram cycle and writes the JSON output to stdout.
func RunCycle(ctx context.Context, args CycleArgs, stdout io.Writer) error {
	if err := requireLLMCmd(args.LLMCmd); err != nil {
		return fmt.Errorf("cycle: %w", err)
	}

	cmdString := resolveLLMCmd(args.LLMCmd)
	runner := llmcmd.New(cmdString)

	dataDir := args.DataDir
	if defaultErr := applyDataDirDefault(&dataDir); defaultErr != nil {
		return fmt.Errorf("cycle: %w", defaultErr)
	}

	transcripts := &transcriptReaderAdapter{
		finder: recall.NewFinder(),
		reader: recall.NewReader(),
	}
	persister := &cyclePersisterAdapter{
		dataDir: dataDir,
		caller:  llmcmd.CallerFunc(runner),
		lister:  memory.NewLister(),
		stdout:  io.Discard,
	}
	recaller := &cycleRecallerAdapter{
		dataDir:    dataDir,
		summarizer: llmcmd.NewExtractor(runner),
	}

	c := cycle.New(runner, transcripts, persister, recaller)

	out, err := c.Run(ctx, args.ProjectDir)
	if err != nil {
		return fmt.Errorf("cycle: %w", err)
	}

	bs, marshalErr := json.MarshalIndent(out, "", "  ")
	if marshalErr != nil {
		return fmt.Errorf("cycle: marshalling output: %w", marshalErr)
	}

	_, writeErr := stdout.Write(append(bs, '\n'))
	if writeErr != nil {
		return fmt.Errorf("cycle: writing output: %w", writeErr)
	}

	return nil
}

// transcriptReaderAdapter implements cycle.TranscriptReader by reusing
// the existing recall finder + reader.
type transcriptReaderAdapter struct {
	finder recall.Finder
	reader recall.Reader
}

func (a *transcriptReaderAdapter) Read(projectDir string, budget int) (string, error) {
	sessions, err := a.finder.Find(projectDir)
	if err != nil {
		return "", fmt.Errorf("finding sessions: %w", err)
	}

	var b strings.Builder
	used := 0

	for _, s := range sessions {
		remaining := budget - used
		if remaining <= 0 {
			break
		}

		content, _, readErr := a.reader.Read(s.Path, remaining)
		if readErr != nil {
			continue
		}

		b.WriteString(content)
		used = b.Len()
	}

	return b.String(), nil
}

// cyclePersisterAdapter implements cycle.Persister by calling the existing
// writeMemory function (refactored in Task D4 to return name + persisted).
type cyclePersisterAdapter struct {
	dataDir string
	caller  llmCaller
	lister  memoryLister
	stdout  io.Writer
}

func (a *cyclePersisterAdapter) WriteFeedback(
	ctx context.Context, situation, behavior, impact, action string,
) (string, bool, error) {
	rec := &memory.MemoryRecord{
		SchemaVersion: memorySchemaVersion,
		Source:        "agent",
		Situation:     situation,
		Type:          typeFeedback,
		Content: memory.ContentFields{
			Behavior: behavior, Impact: impact, Action: action,
		},
	}
	dataDir := a.dataDir

	return writeMemory(ctx, rec, situation, &dataDir, false, a.stdout, "cycle", a.caller, a.lister)
}

func (a *cyclePersisterAdapter) WriteFact(
	ctx context.Context, situation, subject, predicate, object string,
) (string, bool, error) {
	rec := &memory.MemoryRecord{
		SchemaVersion: memorySchemaVersion,
		Source:        "agent",
		Situation:     situation,
		Type:          typeFact,
		Content: memory.ContentFields{
			Subject: subject, Predicate: predicate, Object: object,
		},
	}
	dataDir := a.dataDir

	return writeMemory(ctx, rec, situation, &dataDir, false, a.stdout, "cycle", a.caller, a.lister)
}

// cycleRecallerAdapter implements cycle.Recaller by running the existing
// recall pipeline scoped to the project dir, with --llm-cmd as the backend.
type cycleRecallerAdapter struct {
	dataDir    string
	summarizer recall.SummarizerI
}

func (a *cycleRecallerAdapter) Recall(ctx context.Context, projectDir, query string) (string, error) {
	orch := recall.NewOrchestrator(
		recall.NewFinder(),
		recall.NewReader(),
		a.summarizer,
		memory.NewLister(),
		a.dataDir,
	)

	result, err := orch.Recall(ctx, query, projectDir)
	if err != nil {
		return "", fmt.Errorf("recalling for query %q: %w", query, err)
	}

	return result.Report, nil
}
```

Add imports: `"io"`, `"strings"`, `"engram/internal/memory"`, `"engram/internal/recall"`.

Note: `recall.NewFinder()` and `recall.NewReader()` are the public constructors of the existing implementations. If those names differ in the codebase, locate the equivalents via `grep -n "func New" internal/recall/`.

- [ ] **Step 4: Wire into targ**

Add the cycle command to `internal/cli/targets.go`:

```go
// in the registration section
{
	Name: "cycle",
	Help: "run a single learn-and-recall evaluation cycle",
	Args: &CycleArgs{},
	Run: func(ctx context.Context, args interface{}, stdout io.Writer) error {
		return RunCycle(ctx, *args.(*CycleArgs), stdout)
	},
},
```

(Adapt to the actual registration pattern in `targets.go`.)

- [ ] **Step 5: Verify pass**

Run: `targ test ./internal/cli/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/ internal/cycle/
git commit -m "feat(cli): engram cycle subcommand"
```

---

## Phase F — Plugin TS changes

### Task F1: Replace inline companion logic with `engram cycle` call

**Files:**
- Modify: `opencode/plugins/engram.ts`

- [ ] **Step 1: Read current plugin shape**

```bash
sed -n '260,365p' opencode/plugins/engram.ts
```

(Confirm the system.transform function is around lines 266-364 from the prior read.)

- [ ] **Step 2: Write the replacement**

Replace the entire `system.transform` body with:

```typescript
"experimental.chat.system.transform": async (input: any, output) => {
  const before = output.system[0]
  const reminder = await getReminder("system")
  const sessionID = input?.sessionID

  if (process.env.ENGRAM_COMPANION_MODE === "1") {
    output.system[0] = before + reminder
    return
  }

  const projectDir = input?.directory ?? process.cwd()
  const llmCmd = `${ENGRAM_BIN} ` // placeholder; actual cmd injected below
  const cycleResult = await runEngramCycle(projectDir, sessionID)
  const block = formatCycleResult(cycleResult)

  output.system[0] = before + reminder + (block ? "\n\n" + block : "")
},
```

Add helpers above the `EngramPlugin` export:

```typescript
const COMPANION_MODEL = "opencode/qwen3.6-plus"

interface CycleResult {
  learned: any[]
  recalled: { query: string; report: string }[]
}

async function runEngramCycle(projectDir: string, sessionID: string): Promise<CycleResult> {
  const llmCmd = `opencode run -m ${COMPANION_MODEL}`
  const proc = Bun.spawn(
    [ENGRAM_BIN, "cycle", "--llm-cmd", llmCmd, "--project-dir", projectDir],
    { stdout: "pipe", stderr: "pipe" },
  )
  await proc.exited
  if (proc.exitCode !== 0) {
    const err = (await proc.stderr.text()).slice(0, 2000)
    console.error(`[engram] cycle failed: ${err}`)
    return { learned: [], recalled: [] }
  }

  const stdout = (await proc.stdout.text()).trim()
  if (!stdout) return { learned: [], recalled: [] }

  try {
    return JSON.parse(stdout) as CycleResult
  } catch (parseErr) {
    console.error(`[engram] cycle JSON parse failed: ${String(parseErr).slice(0, 500)}`)
    return { learned: [], recalled: [] }
  }
}

function formatCycleResult(result: CycleResult): string {
  if (!result.recalled.length) return ""

  let block = "## Recalled memories\n"
  for (const { query, report } of result.recalled) {
    block += `\n### Query: ${query}\n${report}\n`
  }

  return block.trimEnd()
}
```

Remove the now-unused helpers: `runEngramRecall`, `runEngramRecallWithQuery`, `runCompanion`, `readCompanionSession`, `writeCompanionSession`, `extractLatestUserMessage`, `companionTrace`, `logCompanionInjection`, `debugFireHeader`, etc. Keep only `logTransform`, `getReminder`, `ensureBinary`, and the constants/imports actually still in use.

(Optional: keep a thin `companionTrace`-style logger pointed at a single `cycle-trace.jsonl` for observability.)

- [ ] **Step 3: Verify the plugin still loads**

```bash
cd opencode && bun install && cd ..
opencode --print-logs --log-level DEBUG run -m opencode/qwen3.6-plus 'echo plugin-loaded' 2>&1 | grep -i 'engram\|error\|cycle'
```

Expected: no plugin-load errors. The cycle invocation should fire (look for "engram cycle" in logs).

- [ ] **Step 4: Run the existing plugin test scaffolding if any**

```bash
cd opencode && bun test 2>&1 | tail -20
```

Expected: PASS or "no tests" — there may be no TS tests, in which case the manual smoke test above is the gate.

- [ ] **Step 5: Commit**

```bash
git add opencode/plugins/engram.ts
git commit -m "refactor(plugin): use engram cycle for learn+recall"
```

---

## Phase H — Validation

### Task H1: End-to-end planted-token replay

**Files:**
- Create: `internal/cycle/integration_test.go`

- [ ] **Step 1: Write the integration test**

```go
//go:build integration

package cycle_test

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
)

func TestE2E_PlantedTokenSurfacesInRecalledReport(t *testing.T) {
	g := NewWithT(t)
	dataDir := t.TempDir()
	projectDir := t.TempDir()

	// Plant a fact memory containing a unique token.
	const token = "INTEGRATION-TOKEN-91827364"
	plant := exec.Command(
		"engram", "learn", "fact",
		"--source", "agent",
		"--no-dup-check",
		"--situation", "asked about the integration test token",
		"--subject", "engram",
		"--predicate", "verification token is",
		"--object", token,
		"--data-dir", dataDir,
	)
	g.Expect(plant.Run()).To(Succeed())

	// Plant a transcript file in the project dir asking about that situation.
	// (Adapt path/format to whatever the finder discovers.)
	// ... write a JSONL or matching file ...

	// Run cycle.
	cycle := exec.Command(
		"engram", "cycle",
		"--llm-cmd", "opencode run -m opencode/qwen3.6-plus",
		"--project-dir", projectDir,
		"--data-dir", dataDir,
	)
	out, err := cycle.Output()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(strings.Contains(string(out), token)).To(BeTrue())
}
```

(If the integration build tag isn't standard in this repo, drop the tag or use the project's existing convention for integration tests.)

- [ ] **Step 2: Run**

```bash
targ test --tags=integration ./internal/cycle/...
```

(Or whatever the project convention is. If integration tests don't run by default in `targ test`, document the manual invocation in the plan's validation log.)

- [ ] **Step 3: Commit**

```bash
git add internal/cycle/integration_test.go
git commit -m "test(cycle): planted-token end-to-end integration test"
```

### Task H2: `--llm-cmd` failure-path tests

- [ ] **Step 1: Write failure-path tests**

In `internal/cycle/cycle_test.go`, add:

```go
func TestCycle_LLMCallAFailureProducesEmptyLearned(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	runner := &fakeRunner{errors: []error{errors.New("boom"), nil}}
	transcripts := &fakeTranscript{content: "anything"}

	c := cycle.New(runner, transcripts, &fakePersister{}, &fakeRecaller{})

	out, err := c.Run(context.Background(), "/tmp/x")
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(out.Learned).To(BeEmpty())
}

func TestCycle_LLMCallBFailureProducesEmptyRecalled(t *testing.T) {
	// similar shape: runner errors on second call
}

func TestCycle_PerQueryRecallFailureSkipsEntry(t *testing.T) {
	// recaller returns error for one query; that query is dropped from output
}
```

(Extend `fakeRunner` to support a per-call error array. Verify behavior matches the spec's failure-mode table.)

- [ ] **Step 2: Run**

```bash
targ test ./internal/cycle/...
```

Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/cycle/
git commit -m "test(cycle): llm-cmd failure paths produce empty arrays"
```

### Task H3: Plugin smoke test

Manual validation, not a Go test:

- [ ] **Step 1: Set up clean environment**

```bash
rm -f ~/.local/share/engram/companion-session/*.txt
rm -f ~/.local/share/engram/companion-trace.jsonl
rm -f ~/.local/share/engram/companion-injections.log
rm -f ~/.local/share/engram/companion-debug.log
```

- [ ] **Step 2: Plant a verification token**

```bash
engram learn fact \
  --source agent \
  --no-dup-check \
  --situation "asked about plugin integration verification" \
  --subject "plugin" \
  --predicate "integration token is" \
  --object "PLUGIN-INTEGRATION-VERIFY-99181"
```

- [ ] **Step 3: Run a fresh opencode session that prompts about the token**

```bash
opencode run -m opencode/qwen3.6-plus 'What plugin integration verification details do you remember?' 2>&1 | tail -50
```

Expected: response contains `PLUGIN-INTEGRATION-VERIFY-99181`.

- [ ] **Step 4: Run full check**

```bash
targ check-full
```

Expected: PASS.

- [ ] **Step 5: Clean up planted memory and commit any test-cleanup changes**

```bash
rm ~/.local/share/engram/memory/facts/asked-about-plugin-integration-verification.toml
git status  # confirm no spurious changes
```

---

## Self-review pass after writing the plan

Once all tasks above are complete, re-run a final check:

- [ ] `targ check-full` — all linters and tests pass.
- [ ] `grep -rn "anthropic" internal/ cmd/` — no remaining references.
- [ ] `grep -rn "claude-haiku" internal/ cmd/` — no remaining references.
- [ ] `engram cycle --help` — flag list shows `--llm-cmd`, `--project-dir`, `--transcript-budget`.
- [ ] `engram recall --query 'anything'` (with `ENGRAM_LLM_CMD` set) — produces a prose report.
- [ ] `engram recall` (no query, with env set) — produces a prose report drawn from session history.
- [ ] Plugin smoke test from Task H3 passes.

---

## Out of scope (re-stated for the implementer)

- **Cost optimization** (caching cycle output per primary turn). Acceptable to ship without; user has flagged for follow-up.
- **MCP wrapper**. Separate effort.
- **Companion session reuse across cycle internal calls**. All `--llm-cmd` invocations are independent.
- **`--force` flag on learn**. Not needed under new write logic.
