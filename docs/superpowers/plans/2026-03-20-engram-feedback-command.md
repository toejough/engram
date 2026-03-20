# Engram Feedback Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `engram feedback <slug> --relevant|--irrelevant --used|--notused --data-dir DIR` command that records explicit agent-reported effectiveness signal, and update surfacing output headers to instruct the LLM to call `feedback` and `show` for surfaced memories.

**Architecture:** Two changes: (1) new `feedback` CLI command that reads a memory TOML, increments the appropriate counter (followed/ignored based on flags), and writes it back; (2) update surfacing output headers in `internal/surface/surface.go` to include LLM instructions for calling `feedback` and `show`.

**Tech Stack:** Go, gomega, targ, BurntSushi/toml

---

## File Structure

| File | Responsibility |
|------|---------------|
| `internal/cli/feedback.go` | `runFeedback`: parse flags, resolve slug, read TOML, update counter, write back |
| `internal/cli/feedback_test.go` | Tests for feedback command via `cli.Run` |
| `internal/cli/cli.go` | Add `case "feedback"` to dispatcher |
| `internal/surface/surface.go` | Update output headers to include feedback/show instructions |
| `internal/surface/surface_test.go` | Update tests for new header format |

---

### Task 1: Add feedback CLI command

**Files:**
- Create: `internal/cli/feedback.go`
- Create: `internal/cli/feedback_test.go`
- Modify: `internal/cli/cli.go` (add `case "feedback"`)

- [ ] **Step 1: Write the failing test — relevant+used increments followed_count**

In `internal/cli/feedback_test.go`:

```go
package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"engram/internal/cli"
)

func TestFeedback_RelevantUsed_IncrementsFollowed(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dataDir := t.TempDir()
	memoriesDir := filepath.Join(dataDir, "memories")
	g.Expect(os.MkdirAll(memoriesDir, 0o750)).To(Succeed())

	memContent := "title = \"Test\"\nprinciple = \"Do it\"\nfollowed_count = 3\nignored_count = 1\n"
	memPath := filepath.Join(memoriesDir, "test-mem.toml")
	g.Expect(os.WriteFile(memPath, []byte(memContent), 0o644)).To(Succeed())

	var stdout, stderr bytes.Buffer

	err := cli.Run(
		[]string{"engram", "feedback", "test-mem", "--relevant", "--used", "--data-dir", dataDir},
		&stdout, &stderr, strings.NewReader(""),
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	// Re-read the file and check followed_count incremented.
	updated, readErr := os.ReadFile(memPath)
	g.Expect(readErr).NotTo(HaveOccurred())

	if readErr != nil {
		return
	}

	g.Expect(string(updated)).To(ContainSubstring("followed_count = 4"))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestFeedback_RelevantUsed -v`
Expected: FAIL — `feedback` command not recognized.

- [ ] **Step 3: Write implementation**

Add to `internal/cli/cli.go` dispatcher:
```go
case "feedback":
    return runFeedback(subArgs, stdout)
```

Create `internal/cli/feedback.go`. Key design decisions:

- Reuse `extractSlug` from `show.go` for slug-before-flags support
- Read the full TOML file, decode into a `map[string]any` (preserves all fields including ones we don't know about), update the counter, re-encode and write back. This avoids losing fields that `showTOMLRecord` doesn't include.
- Flag mapping: `--relevant --used` → increment `followed_count`; `--relevant --notused` → increment `ignored_count`; `--irrelevant` → no counter change (memory wasn't applicable)
- Print confirmation to stdout: `[engram] Feedback recorded: test-mem (relevant, used)`

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestFeedback_RelevantUsed -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/cli/feedback.go internal/cli/feedback_test.go internal/cli/cli.go
git commit -m "feat(cli): add engram feedback command (#341)"
```

---

### Task 2: Feedback error paths and all flag combinations

**Files:**
- Modify: `internal/cli/feedback_test.go`

- [ ] **Step 1: Add tests for all flag combinations and error paths**

```go
func TestFeedback_RelevantNotused_IncrementsIgnored(t *testing.T) {
	// Same setup as Task 1 test, but use --relevant --notused
	// Assert ignored_count incremented, followed_count unchanged
}

func TestFeedback_Irrelevant_NoCounterChange(t *testing.T) {
	// Use --irrelevant (--used/--notused not required when irrelevant)
	// Assert no counters changed
}

func TestFeedback_MissingDataDir(t *testing.T) {
	// No --data-dir flag
	// Assert error containing "--data-dir"
}

func TestFeedback_MissingSlug(t *testing.T) {
	// No slug argument
	// Assert error containing "slug"
}

func TestFeedback_MemoryNotFound(t *testing.T) {
	// Slug doesn't match any file
	// Assert error
}

func TestFeedback_MissingRelevanceFlag(t *testing.T) {
	// Neither --relevant nor --irrelevant
	// Assert error containing "relevant" or "irrelevant"
}

func TestFeedback_SlugBeforeFlags(t *testing.T) {
	// engram feedback my-mem --relevant --used --data-dir /path
	// Assert works correctly (slug before flags)
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/cli/ -run TestFeedback -v`
Expected: all PASS

- [ ] **Step 3: Commit**

```bash
git add internal/cli/feedback_test.go
git commit -m "test(cli): add feedback error path and flag combination tests (#341)"
```

---

### Task 3: Update surfacing output headers with LLM instructions

**Files:**
- Modify: `internal/surface/surface.go` (renderToolAdvisories, runPrompt, runSessionStart)
- Modify: `internal/surface/surface_test.go`

- [ ] **Step 1: Find and update all surfacing headers**

Three locations output surfacing headers. Update each:

**Tool mode** (`renderToolAdvisories`, around line 355-357):
```go
// Old:
_, _ = fmt.Fprintf(&summaryBuf, "[engram] %d tool advisories:\n", len(candidates))
_, _ = fmt.Fprintf(&contextBuf, "[engram] Tool call advisory:\n")

// New:
_, _ = fmt.Fprintf(&summaryBuf, "[engram] %d tool advisories:\n", len(candidates))
_, _ = fmt.Fprintf(&contextBuf, "[engram] Memories — for any relevant memory, call `engram show <name> --data-dir ~/.claude/engram/data` for full details. After your turn, call `engram feedback <name> --relevant|--irrelevant --used|--notused --data-dir ~/.claude/engram/data` for each:\n")
```

**Prompt mode** (`runPrompt`, around line 574-577):
```go
// Old:
_, _ = fmt.Fprintf(&buf, "[engram] Relevant memories:\n")

// New:
_, _ = fmt.Fprintf(&buf, "[engram] Memories — for any relevant memory, call `engram show <name> --data-dir ~/.claude/engram/data` for full details. After your turn, call `engram feedback <name> --relevant|--irrelevant --used|--notused --data-dir ~/.claude/engram/data` for each:\n")
```

**Summary lines** stay short (terminal display). Only `Context` (additionalContext) gets the full instruction.

- [ ] **Step 2: Update existing tests that assert the old header text**

Search for tests asserting "Relevant memories" or "Tool call advisory" and update expected strings.

Run: `grep -rn "Relevant memories\|Tool call advisory" internal/surface/surface_test.go`

- [ ] **Step 3: Run tests**

Run: `targ test`
Expected: all PASS

- [ ] **Step 4: Run targ check-full**

Run: `targ check-full`
Expected: all checks pass

- [ ] **Step 5: Commit**

```bash
git add internal/surface/surface.go internal/surface/surface_test.go
git commit -m "feat(surface): add feedback and show instructions to surfacing output (#341)"
```
