# Recall Output Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make recall output parsable by agent tooling, reduce budget to 15KB, and wire real memory surfacing.

**Architecture:** Three changes to the recall pipeline: (1) replace JSON output with plain text + `=== MEMORIES ===` separator, (2) add `DefaultModeABudget` constant and use it in `recallModeA`, (3) build a surfacer adapter in `internal/cli/` that wraps `surface.Surfacer` behind the `recall.MemorySurfacer` interface, wire it into `runRecall`.

**Tech Stack:** Go, engram internal packages (`recall`, `surface`, `retrieve`, `effectiveness`, `memory`)

**Spec:** `docs/superpowers/specs/2026-03-22-recall-output-fix-design.md`

---

### Task 1: Reduce mode A budget to 15KB

**Files:**
- Modify: `internal/recall/orchestrate.go:10-13` (constants), `internal/recall/orchestrate.go:80,88` (call sites)
- Modify: `internal/recall/orchestrate_test.go:113-140` (budget test)

- [ ] **Step 1: Update the budget test to expect 15KB**

In `internal/recall/orchestrate_test.go`, the "budget exceeded stops reading sessions" test (line 113) uses `recall.DefaultStripBudget` for the fake reader's size. Add a new constant reference and update this test to use `recall.DefaultModeABudget`:

```go
t.Run("budget exceeded stops reading sessions", func(t *testing.T) {
    t.Parallel()
    g := NewWithT(t)

    finder := &fakeFinder{paths: []string{"/a.jsonl", "/b.jsonl"}}
    reader := &fakeReader{
        contents: map[string]string{
            "/a.jsonl": "big content",
            "/b.jsonl": "should not read",
        },
        sizes: map[string]int{
            "/a.jsonl": recall.DefaultModeABudget, // Exactly at budget
            "/b.jsonl": 100,
        },
    }

    orch := recall.NewOrchestrator(finder, reader, nil, nil)

    result, err := orch.Recall(context.Background(), "/proj", "")
    g.Expect(err).NotTo(HaveOccurred())

    if err != nil {
        return
    }

    // Only first session's content should be returned.
    g.Expect(result.Summary).To(Equal("big content"))
})
```

- [ ] **Step 2: Run the test — expect FAIL**

Run: `targ test`
Expected: compilation error — `recall.DefaultModeABudget` undefined.

- [ ] **Step 3: Add the constant and update recallModeA**

In `internal/recall/orchestrate.go`, add the new constant and update `recallModeA`:

```go
const (
    DefaultExtractCap  = 1500      // 1500 bytes of extracted content
    DefaultModeABudget = 15 * 1024 // 15KB for mode A raw transcript
    DefaultStripBudget = 50 * 1024 // 50KB per-session read budget (mode B)
)
```

In `recallModeA`, change both references from `DefaultStripBudget` to `DefaultModeABudget`:

```go
func (o *Orchestrator) recallModeA(
    _ context.Context,
    sessions []string,
) (*Result, error) {
    var builder strings.Builder

    bytesRead := 0

    for _, path := range sessions {
        content, size, readErr := o.reader.Read(path, DefaultModeABudget-bytesRead)
        if readErr != nil {
            continue
        }

        builder.WriteString(content)

        bytesRead += size
        if bytesRead >= DefaultModeABudget {
            break
        }
    }

    accumulated := builder.String()
    memories := o.surfaceMemories(accumulated)

    return &Result{Summary: accumulated, Memories: memories}, nil
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `targ test`
Expected: all tests pass.

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: all checks pass.

- [ ] **Step 6: Commit**

```bash
git add internal/recall/orchestrate.go internal/recall/orchestrate_test.go
git commit -m "feat(recall): reduce mode A budget from 50KB to 15KB

AI-Used: [claude]"
```

---

### Task 2: Plain text output format

**Files:**
- Modify: `internal/cli/cli.go:1351-1359` (runRecall output)
- Modify: `internal/cli/cli_test.go` (if recall CLI tests exist — check first)

- [ ] **Step 1: Find and read existing CLI tests for runRecall**

Search for tests that assert JSON output from `runRecall`. These tests need to be updated to expect plain text. Look in `internal/cli/cli_test.go` or similar files.

Run: grep for `runRecall\|recall.*json\|recall.*Encode\|RecallIntegration` in `internal/cli/`

Note: `TestRecallIntegrationSummaryMode` (around line 786) has a comment referencing "JSON output" that must be updated to reflect plain text format.

- [ ] **Step 2: Update or write tests for plain text output**

Update existing tests to expect plain text. Also add test cases for:
- Summary content appears as plain text lines (not wrapped in JSON)
- When memories are present, `=== MEMORIES ===` separator appears on its own line, followed by memory content
- When memories are empty, no separator appears
- When summary is empty but memories are present, output starts cleanly (no leading blank line)

- [ ] **Step 3: Run the test — expect FAIL**

Run: `targ test`
Expected: FAIL — output is still JSON.

- [ ] **Step 4: Replace JSON encoding with plain text in runRecall**

In `internal/cli/cli.go`, replace the JSON encoder (lines 1358-1359) with plain text output:

```go
// Replace this:
//nolint:wrapcheck // thin JSON encoding at CLI boundary
return json.NewEncoder(stdout).Encode(result)

// With this:
_, writeErr := fmt.Fprint(stdout, result.Summary)
if writeErr != nil {
    return fmt.Errorf("recall: writing summary: %w", writeErr)
}

if result.Memories != "" {
    _, writeErr = fmt.Fprint(stdout, "\n=== MEMORIES ===\n", result.Memories)
    if writeErr != nil {
        return fmt.Errorf("recall: writing memories: %w", writeErr)
    }
}

return nil
```

Remove `json` from the import list if it's no longer used by this function (check other usages first).

- [ ] **Step 5: Run tests — expect PASS**

Run: `targ test`
Expected: all tests pass.

- [ ] **Step 6: Run full checks**

Run: `targ check-full`
Expected: all checks pass.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/cli.go internal/cli/cli_test.go
git commit -m "feat(recall): output plain text instead of JSON

AI-Used: [claude]"
```

---

### Task 3: Wire memory surfacer adapter

This is the largest task. The adapter lives in `internal/cli/` (at the wiring boundary) because `internal/recall/` cannot import `internal/surface/` (circular dependency). The adapter defines its own `SurfaceRunnerOptions` struct instead of using `surface.Options` — this keeps the interface and tests free of the `surface` import. A thin `surfaceRunnerAdapter` in `cli.go` bridges the two types at wiring time.

The wiring uses a minimal set of surface options (effectiveness + surfacing recorder). Advanced options (tracker, log reader, contradiction detector, cross-ref checker, tool gate) are intentionally omitted — recall surfaces memories as supplementary context, not as the primary surfacing path. Those features remain in `runSurface` for the hook-based surfacing pipeline.

**Files:**
- Create: `internal/cli/recallsurfacer.go` (adapter)
- Create: `internal/cli/recallsurfacer_test.go` (adapter tests)
- Modify: `internal/cli/cli.go:1340-1351` (wire adapter into runRecall)

- [ ] **Step 1: Write the adapter test**

Create `internal/cli/recallsurfacer_test.go`. The adapter wraps a `surface.Surfacer` via the `Run` method. Test with a fake that captures the options passed:

```go
package cli_test

import (
    "context"
    "errors"
    "io"
    "testing"

    "engram/internal/cli"

    . "github.com/onsi/gomega"
)

func TestRecallSurfacer(t *testing.T) {
    t.Parallel()

    t.Run("surfaces memories in prompt mode", func(t *testing.T) {
        t.Parallel()
        g := NewWithT(t)

        runner := &fakeSurfaceRunner{
            writeOutput: "[engram] memory one\n[engram] memory two",
        }
        surfacer := cli.NewRecallSurfacer(runner, "/data", context.Background())

        result, err := surfacer.Surface("my query text")
        g.Expect(err).NotTo(HaveOccurred())

        if err != nil {
            return
        }

        g.Expect(result).To(Equal("[engram] memory one\n[engram] memory two"))
        g.Expect(runner.opts.Mode).To(Equal("prompt"))
        g.Expect(runner.opts.Message).To(Equal("my query text"))
        g.Expect(runner.opts.DataDir).To(Equal("/data"))
    })

    t.Run("empty query returns empty string", func(t *testing.T) {
        t.Parallel()
        g := NewWithT(t)

        runner := &fakeSurfaceRunner{}
        surfacer := cli.NewRecallSurfacer(runner, "/data", context.Background())

        result, err := surfacer.Surface("")
        g.Expect(err).NotTo(HaveOccurred())

        if err != nil {
            return
        }

        g.Expect(result).To(BeEmpty())
        g.Expect(runner.called).To(BeFalse())
    })

    t.Run("runner error propagates", func(t *testing.T) {
        t.Parallel()
        g := NewWithT(t)

        runner := &fakeSurfaceRunner{
            err: errors.New("surface failed"),
        }
        surfacer := cli.NewRecallSurfacer(runner, "/data", context.Background())

        _, err := surfacer.Surface("query")
        g.Expect(err).To(HaveOccurred())

        if err != nil {
            g.Expect(err.Error()).To(ContainSubstring("surface failed"))
        }
    })
}

type fakeSurfaceRunner struct {
    opts        cli.SurfaceRunnerOptions
    writeOutput string
    err         error
    called      bool
}

func (f *fakeSurfaceRunner) Run(ctx context.Context, w io.Writer, opts cli.SurfaceRunnerOptions) error {
    f.called = true
    f.opts = opts

    if f.err != nil {
        return f.err
    }

    if f.writeOutput != "" {
        _, _ = io.WriteString(w, f.writeOutput)
    }

    return nil
}
```

Note: The adapter defines its own `SurfaceRunner` interface and `SurfaceRunnerOptions` struct to avoid importing `internal/surface` in tests. The real `surface.Surfacer` satisfies this interface via structural typing, or we use a thin wrapper in the wiring code.

- [ ] **Step 2: Run test — expect FAIL**

Run: `targ test`
Expected: compilation error — `cli.NewRecallSurfacer`, `cli.SurfaceRunner`, `cli.SurfaceRunnerOptions` undefined.

- [ ] **Step 3: Implement the adapter**

Create `internal/cli/recallsurfacer.go`:

```go
package cli

import (
    "context"
    "fmt"
    "io"
    "strings"
)

// SurfaceRunnerOptions holds the options for a surface run.
// Mirrors surface.Options fields needed for prompt mode.
type SurfaceRunnerOptions struct {
    Mode    string
    DataDir string
    Message string
}

// SurfaceRunner runs the memory surface pipeline.
type SurfaceRunner interface {
    Run(ctx context.Context, w io.Writer, opts SurfaceRunnerOptions) error
}

// RecallSurfacer adapts the surface pipeline to the recall.MemorySurfacer interface.
type RecallSurfacer struct {
    runner  SurfaceRunner
    dataDir string
    ctx     context.Context
}

// NewRecallSurfacer creates a RecallSurfacer.
func NewRecallSurfacer(runner SurfaceRunner, dataDir string, ctx context.Context) *RecallSurfacer {
    return &RecallSurfacer{runner: runner, dataDir: dataDir, ctx: ctx}
}

// Surface finds relevant memories for the given query.
func (s *RecallSurfacer) Surface(query string) (string, error) {
    if query == "" {
        return "", nil
    }

    var buf strings.Builder

    err := s.runner.Run(s.ctx, &buf, SurfaceRunnerOptions{
        Mode:    "prompt",
        DataDir: s.dataDir,
        Message: query,
    })
    if err != nil {
        return "", fmt.Errorf("recall surfacer: %w", err)
    }

    return buf.String(), nil
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `targ test`
Expected: all tests pass.

- [ ] **Step 5: Run full checks**

Run: `targ check-full`
Expected: all checks pass.

- [ ] **Step 6: Commit the adapter**

```bash
git add internal/cli/recallsurfacer.go internal/cli/recallsurfacer_test.go
git commit -m "feat(cli): add RecallSurfacer adapter for recall memory surfacing

AI-Used: [claude]"
```

- [ ] **Step 7: Wire the adapter into runRecall**

In `internal/cli/cli.go`, update `runRecall` to create and wire the adapter. Add a `surfaceRunnerAdapter` that converts `surface.Options` to `SurfaceRunnerOptions`:

```go
// Add after the summarizer block (around line 1349):

// Build memory surfacer adapter.
retriever := retrieve.New()

allMemories, memErr := retriever.ListMemories(ctx, *dataDir)
if memErr != nil {
    return fmt.Errorf("recall: listing memories: %w", memErr)
}

effAdapter := &effectivenessAdapter{stats: effectiveness.FromMemories(allMemories)}
surfacerOpts := []surface.SurfacerOption{
    surface.WithEffectiveness(effAdapter),
    surface.WithSurfacingRecorder(func(path string) error {
        return memory.ReadModifyWrite(path, func(record *memory.MemoryRecord) {
            record.SurfacedCount++
            record.LastSurfacedAt = time.Now().UTC().Format(time.RFC3339)
        })
    }),
}
realSurfacer := surface.New(retriever, surfacerOpts...)
memorySurfacer := NewRecallSurfacer(
    &surfaceRunnerAdapter{surfacer: realSurfacer},
    *dataDir,
    ctx,
)

// Update the orchestrator line:
orch := recall.NewOrchestrator(finder, reader, summarizer, memorySurfacer)
```

Add the adapter type that bridges `surface.Surfacer.Run(ctx, w, surface.Options)` to `SurfaceRunner.Run(ctx, w, SurfaceRunnerOptions)`:

```go
type surfaceRunnerAdapter struct {
    surfacer *surface.Surfacer
}

func (a *surfaceRunnerAdapter) Run(ctx context.Context, w io.Writer, opts SurfaceRunnerOptions) error {
    return a.surfacer.Run(ctx, w, surface.Options{
        Mode:    opts.Mode,
        DataDir: opts.DataDir,
        Message: opts.Message,
    })
}
```

Add needed imports: `retrieve`, `effectiveness`, `memory`, `surface` (check which are already imported).

- [ ] **Step 8: Run tests — expect PASS**

Run: `targ test`
Expected: all tests pass.

- [ ] **Step 9: Run full checks**

Run: `targ check-full`
Expected: all checks pass.

- [ ] **Step 10: Commit the wiring**

```bash
git add internal/cli/cli.go internal/cli/recallsurfacer.go
git commit -m "feat(cli): wire real memory surfacer into recall pipeline (#367)

AI-Used: [claude]"
```

---

### Task 4: Update recall skill

**Files:**
- Modify: `skills/recall/SKILL.md`

- [ ] **Step 1: Update SKILL.md**

Replace the "Handling Output" section to reflect plain text format:

```markdown
## Handling Output

The command outputs plain text in two sections:

1. **Transcript content** — raw stripped session content (first section, always present)
2. **Memories** — relevant memories surfaced by similarity (after `=== MEMORIES ===` separator, only present when memories match)

**For plain `/recall`:** The transcript content is raw session history (not a summary). Read it, absorb the full context, then present a concise recap focusing on:

1. **What tradeoffs were considered** — options weighed and why
2. **What decisions were made** — what was chosen
3. **What work got done** — commits, issues filed, changes pushed
4. **What is still outstanding** — open threads, deferred items, known gaps
5. **What state things were left in** — clean/dirty tree, passing/failing tests, waiting on something

Prioritize conclusions over discussions. The user needs to know how work *ended*, not everything that was talked about.

**For `/recall <query>`:** The transcript section contains Haiku-extracted content relevant to the query.

In both cases, treat any content after `=== MEMORIES ===` as additional context from the memory system.

If the command fails or returns empty, inform the user that no previous session data was found.
```

- [ ] **Step 2: Commit**

```bash
git add skills/recall/SKILL.md
git commit -m "docs(recall): update skill for plain text output format

AI-Used: [claude]"
```

---

### Task 5: End-to-end verification

**Files:** None (verification only)

- [ ] **Step 1: Run the binary end-to-end**

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
~/.claude/engram/bin/engram recall \
  --data-dir ~/.claude/engram/data \
  --project-slug="$PROJECT_SLUG" 2>&1 | head -20
```

Verify:
- Output is multi-line plain text, NOT JSON (no `{` on first line)
- Transcript content appears as readable text

- [ ] **Step 2: Check for memories section**

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
~/.claude/engram/bin/engram recall \
  --data-dir ~/.claude/engram/data \
  --project-slug="$PROJECT_SLUG" 2>&1 | grep -A5 "=== MEMORIES ==="
```

Verify:
- `=== MEMORIES ===` separator appears
- Real memory content follows the separator (not empty)

**Before concluding wiring is broken:** verify that memory files exist under `~/.claude/engram/data/memories/` (`ls ~/.claude/engram/data/memories/*.toml | wc -l`). If no memories exist, the surfacer correctly returns empty — the test is inconclusive for #367 wiring. If memories DO exist and no separator appears, that is the #367 failure pattern — investigate the adapter wiring.

- [ ] **Step 3: Check output size**

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
~/.claude/engram/bin/engram recall \
  --data-dir ~/.claude/engram/data \
  --project-slug="$PROJECT_SLUG" 2>&1 | wc -c
```

Verify: output is ~15KB or less (not 50KB+).

- [ ] **Step 4: Test mode B still works**

```bash
PROJECT_SLUG="$(echo "$PWD" | tr '/' '-')"
~/.claude/engram/bin/engram recall \
  --data-dir ~/.claude/engram/data \
  --project-slug="$PROJECT_SLUG" \
  --query "token resolver" 2>&1 | head -20
```

Verify: output contains Haiku-extracted content relevant to "token resolver", in plain text format.
