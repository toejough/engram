# Flush Pipeline Simplification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the evaluate step from flush and the flush call from PreCompact, simplifying two pipelines that do unnecessary work.

**Architecture:** Two independent deletions. Flush becomes learn → context-update (2 steps). PreCompact becomes a no-op (it was running flush redundantly with the Stop hook). The surfacing log — currently consumed by evaluate — needs a new cleanup mechanism since nothing else reads it.

**Tech Stack:** Go (internal/cli), bash (hooks), targ build system

---

## File Structure

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/cli/flush.go` | Modify | Remove evaluate step from FlushRunner |
| `internal/cli/flush_test.go` | Modify | Update tests for 2-step pipeline |
| `hooks/pre-compact.sh` | Modify | Remove flush call (becomes no-op — was redundant with Stop hook) |
| `hooks/stop.sh` | Modify | Update comment referencing 3-step pipeline |

Files NOT touched (kept for standalone/diagnostic use):
- `internal/evaluate/` — kept as standalone `engram evaluate` command
- `internal/surfacinglog/` — kept, but log cleanup moves to flush

---

### Task 1: Remove evaluate from FlushRunner

**Files:**
- Modify: `internal/cli/flush.go`
- Test: `internal/cli/flush_test.go`

- [ ] **Step 1: Write the failing test — FlushRunner runs learn → context-update (no evaluate)**

In `internal/cli/flush_test.go`, update T-370 to expect 2 steps:

```go
// T-370: flush command runs learn then context-update (no evaluate).
func TestT370_FlushRunsInOrder(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	runner := cli.NewFlushRunner(
		func() error { callOrder = append(callOrder, "learn"); return nil },
		func() error { callOrder = append(callOrder, "context-update"); return nil },
	)

	err := runner.Run()
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callOrder).To(Equal([]string{"learn", "context-update"}))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestT370 ./internal/cli/`
Expected: FAIL — `NewFlushRunner` still takes 3 args

- [ ] **Step 3: Update FlushRunner to 2 steps**

In `internal/cli/flush.go`:

```go
// FlushRunner executes the end-of-turn memory management pipeline:
// learn → context-update (#309, #348).
type FlushRunner struct {
	learn         func() error
	contextUpdate func() error
}

// NewFlushRunner creates a FlushRunner with the given step functions.
func NewFlushRunner(learn, contextUpdate func() error) *FlushRunner {
	return &FlushRunner{
		learn:         learn,
		contextUpdate: contextUpdate,
	}
}

// Run executes the flush pipeline in order. Stops on first error.
func (f *FlushRunner) Run() error {
	learnErr := f.learn()
	if learnErr != nil {
		return fmt.Errorf("flush: learn: %w", learnErr)
	}

	ctxErr := f.contextUpdate()
	if ctxErr != nil {
		return fmt.Errorf("flush: context-update: %w", ctxErr)
	}

	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestT370 ./internal/cli/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/flush.go internal/cli/flush_test.go
git commit -m "refactor(flush): remove evaluate step from FlushRunner (#348)"
```

---

### Task 2: Update T-371 (stop-on-error test) for 2-step pipeline

**Files:**
- Modify: `internal/cli/flush_test.go`

- [ ] **Step 1: Update T-371 to test learn failure stops pipeline**

```go
// T-371: flush command stops on first step error.
func TestT371_FlushStopsOnError(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var callOrder []string

	learnErr := errors.New("learn failed")

	runner := cli.NewFlushRunner(
		func() error { callOrder = append(callOrder, "learn"); return learnErr },
		func() error { callOrder = append(callOrder, "context-update"); return nil },
	)

	err := runner.Run()
	g.Expect(err).To(MatchError(ContainSubstring("learn failed")))

	g.Expect(callOrder).To(Equal([]string{"learn"}))
}
```

- [ ] **Step 2: Run test to verify it passes**

Run: `targ test -- -run TestT371 ./internal/cli/`
Expected: PASS (FlushRunner already updated)

- [ ] **Step 3: Commit**

```bash
git add internal/cli/flush_test.go
git commit -m "test(flush): update T-371 for 2-step pipeline (#348)"
```

---

### Task 3: Remove evaluate wiring from runFlush + add surfacing log cleanup

**Files:**
- Modify: `internal/cli/flush.go`
- Modify: `internal/cli/flush_test.go`

The surfacing log (`surfacing-log.jsonl`) is appended to on every surface call. Currently evaluate consumes it (atomic rename + read + delete). Without evaluate, the log grows unbounded. The simplest fix: delete the surfacing log at the start of flush, since feedback now replaces evaluate's role.

Note: `runFlush` is a CLI wiring function (same boundary as the existing `os.Getenv` call on the current line 72). Direct `os.Remove` is consistent with the existing pattern — this is edge code, not `internal/` business logic.

- [ ] **Step 1: Write test — flush deletes surfacing log**

Add a new test in `flush_test.go`:

```go
func TestFlush_DeletesSurfacingLog(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Create a surfacing log file.
	logPath := filepath.Join(dataDir, "surfacing-log.jsonl")
	g.Expect(os.WriteFile(logPath, []byte(`{"mode":"prompt"}`+"\n"), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "flush", "--data-dir", dataDir},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Surfacing log should be deleted.
	_, statErr := os.Stat(logPath)
	g.Expect(os.IsNotExist(statErr)).To(BeTrue(),
		"surfacing-log.jsonl should be deleted by flush")
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `targ test -- -run TestFlush_DeletesSurfacingLog ./internal/cli/`
Expected: FAIL — log still exists

- [ ] **Step 3: Update runFlush to remove evaluate and add log cleanup**

In `internal/cli/flush.go`, replace the `runFlush` function. Remove the evaluate step, add surfacing log deletion before the pipeline:

```go
// runFlush is CLI wiring code (edge boundary) — direct os.* calls are
// consistent with the existing os.Getenv pattern here.
func runFlush(args []string, _, stderr io.Writer, stdin io.Reader) error {
	fs := flag.NewFlagSet("flush", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	transcriptPath := fs.String("transcript-path", "", "path to session transcript")
	sessionID := fs.String("session-id", "", "session identifier")
	dataDir := fs.String("data-dir", "", "path to data directory")
	contextPath := fs.String("context-path", "", "path to session-context.md")

	parseErr := fs.Parse(args)
	if parseErr != nil {
		return fmt.Errorf("flush: %w", parseErr)
	}

	if *dataDir == "" {
		return errFlushMissingDataDir
	}

	// Clean up surfacing log — evaluate no longer consumes it (#348).
	_ = os.Remove(filepath.Join(*dataDir, "surfacing-log.jsonl"))

	token := os.Getenv("ENGRAM_API_TOKEN")

	runner := NewFlushRunner(
		func() error {
			if *transcriptPath == "" || *sessionID == "" {
				return nil
			}

			learnArgs := []string{
				"--transcript-path", *transcriptPath,
				"--session-id", *sessionID,
				"--data-dir", *dataDir,
			}

			return RunLearn(learnArgs, token, stderr, stdin, nil)
		},
		func() error {
			if *transcriptPath == "" || *sessionID == "" {
				return nil
			}

			ctxArgs := []string{
				"--transcript-path", *transcriptPath,
				"--session-id", *sessionID,
				"--data-dir", *dataDir,
			}
			if *contextPath != "" {
				ctxArgs = append(ctxArgs, "--context-path", *contextPath)
			}

			return runContextUpdate(ctxArgs)
		},
	)

	return runner.Run()
}
```

Note: this requires adding `"path/filepath"` to the imports in flush.go. The `stdout` parameter changes to `_` since evaluate was the only consumer.

- [ ] **Step 4: Run test to verify it passes**

Run: `targ test -- -run TestFlush_DeletesSurfacingLog ./internal/cli/`
Expected: PASS

- [ ] **Step 5: Run all flush tests**

Run: `targ test -- -run TestFlush ./internal/cli/`
Expected: Some tests may fail — `TestFlush_BadTranscriptPath_ReturnsError` expected evaluate to fail on bad path, but now evaluate is gone. `TestT337_FlushIntegration_PipelineOrdering` references evaluate. Fix in next task.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/flush.go internal/cli/flush_test.go
git commit -m "refactor(flush): remove evaluate wiring, add surfacing log cleanup (#348)"
```

---

### Task 4: Fix remaining flush tests

**Files:**
- Modify: `internal/cli/flush_test.go`

- [ ] **Step 1: Update TestFlush_BadTranscriptPath_ReturnsError**

The original test passed only `--transcript-path` without `--session-id`. Previously, evaluate would try to open the bad path and fail. Now without evaluate, both learn and context-update skip (they require both `--transcript-path` AND `--session-id`), so this becomes a no-error case:

```go
func TestFlush_BadTranscriptPath_SkipsGracefully(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "flush",
			"--data-dir", dataDir,
			"--transcript-path", "/nonexistent/transcript.jsonl",
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// No session-id → learn and context-update both skip → no error.
	g.Expect(err).NotTo(HaveOccurred())
}
```

- [ ] **Step 2: Rename and update TestFlush_WithTranscript_RunsEvaluate**

This test (line 90-117 in the current file) references evaluate in its name and comment. Rename to `TestFlush_WithTranscript_RunsPipeline` and update the comment:

```go
func TestFlush_WithTranscript_RunsPipeline(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Create a minimal transcript file.
	transcriptPath := filepath.Join(dataDir, "transcript.jsonl")
	writeErr := os.WriteFile(transcriptPath, []byte(`{"type":"human","text":"hello"}`+"\n"), 0o644)
	g.Expect(writeErr).NotTo(HaveOccurred())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "flush",
			"--data-dir", dataDir,
			"--transcript-path", transcriptPath,
			"--session-id", "test-session",
			"--context-path", filepath.Join(dataDir, "session-context.md"),
		},
		&stdout, &stderr,
		strings.NewReader(""),
	)
	// May fail due to missing API token for learn/context-update, but exercises the wiring.
	_ = err
}
```

- [ ] **Step 3: Update T-337 integration test**

Remove evaluate-specific assertions (surfacing log consumed, LLM call count). Focus on: learn runs, context-update writes session-context.md.

```go
// T-337: flush integration test — verifies learn runs and
// context-update writes session-context.md when run through the real CLI.
// Not parallel — mutates AnthropicAPIURL global and ENGRAM_API_TOKEN env.
func TestT337_FlushIntegration_PipelineOrdering(t *testing.T) {
	g := NewWithT(t)

	dataDir := t.TempDir()

	// Create a transcript file.
	transcriptPath := filepath.Join(dataDir, "transcript.jsonl")
	g.Expect(os.WriteFile(
		transcriptPath,
		[]byte("user asked about targ check-full\n"), 0o644,
	)).To(Succeed())

	// Mock LLM: learn expects extraction JSON, context-update expects summary text.
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, _ *http.Request) {
			requestCount++

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(
				`{"content":[{"type":"text","text":"[]"}]}`,
			))
		},
	))
	defer server.Close()

	original := cli.AnthropicAPIURL
	cli.AnthropicAPIURL = server.URL

	defer func() { cli.AnthropicAPIURL = original }()

	t.Setenv("ENGRAM_API_TOKEN", "fake-token")

	contextPath := filepath.Join(dataDir, "session-context.md")

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{
			"engram", "flush",
			"--data-dir", dataDir,
			"--transcript-path", transcriptPath,
			"--session-id", "test-session-337",
			"--context-path", contextPath,
		},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Verify context-update ran: session-context.md should exist.
	_, statErr := os.Stat(contextPath)
	g.Expect(statErr).NotTo(HaveOccurred(),
		"session-context.md should be written by context-update")

	// Verify LLM was called (at least learn + context-update).
	g.Expect(requestCount).To(BeNumerically(">=", 2),
		"LLM should be called by multiple pipeline steps")
}
```

- [ ] **Step 3: Remove unused imports**

Remove `"fmt"` and `"time"` from flush_test.go if no longer needed after removing the surfacing log setup code. Keep `"net/http"`, `"net/http/httptest"` for T-337.

- [ ] **Step 4: Run all flush tests**

Run: `targ test -- -run "TestFlush|TestT337|TestT370|TestT371" ./internal/cli/`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/flush_test.go
git commit -m "test(flush): update tests for 2-step pipeline (#348)"
```

---

### Task 5: Remove flush from PreCompact hook

**Files:**
- Modify: `hooks/pre-compact.sh`

PreCompact currently runs the full flush pipeline (learn → evaluate → context-update), which is redundant with the Stop hook that runs flush at the end of every turn. Running flush at PreCompact wastes API tokens on duplicate learn/context-update calls right before context trim.

- [ ] **Step 1: Replace pre-compact.sh with a no-op**

```bash
#!/usr/bin/env bash
# PreCompact hook — no-op (#350).
# Flush runs at end-of-turn via Stop hook. Running it again at
# PreCompact was redundant and wasted API tokens.
exit 0
```

- [ ] **Step 2: Verify hook is executable**

Run: `test -x hooks/pre-compact.sh && echo "OK" || echo "MISSING +x"`
Expected: OK

- [ ] **Step 3: Commit**

```bash
git add hooks/pre-compact.sh
git commit -m "fix(hooks): remove flush from PreCompact — redundant with Stop hook (#350)"
```

---

### Task 6: Update stop.sh comment

**Files:**
- Modify: `hooks/stop.sh`

- [ ] **Step 1: Update the pipeline comment**

Change line 32 from:
```bash
# Unified flush pipeline: learn → evaluate → context-update (#309)
```
to:
```bash
# Unified flush pipeline: learn → context-update (#309, #348)
```

- [ ] **Step 2: Commit**

```bash
git add hooks/stop.sh
git commit -m "docs(hooks): update flush pipeline comment (#348)"
```

---

### Task 7: Run full check

- [ ] **Step 1: Run targ check-full**

Run: `targ check-full`
Expected: ALL PASS — no lint errors, no test failures, coverage OK

- [ ] **Step 2: Fix any issues found**

If check-full finds issues, fix them all in one pass (don't play whack-a-mole).

- [ ] **Step 3: Final commit if needed**

```bash
git add -A
git commit -m "fix: address lint/test issues from flush simplification (#348, #350)"
```
