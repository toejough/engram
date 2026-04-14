# Issue 547: Log memFileSelector Error in watchAndResume

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Log a warning to stdout when `memFileSelector` returns an error in `watchAndResume`, while preserving the empty-result fallback behavior.

**Architecture:** Single three-line change inside `watchAndResume` in `cli_agent.go`, matching the existing warning pattern already used for `silentErr` and `resumeErr` in the same function. New export in `export_test.go` enables direct unit testing without standing up the full conversation loop.

**Tech Stack:** Go, gomega, pgregory.net/rapid (property-based tests)

---

## File Map

| File | Change |
|---|---|
| `internal/cli/export_test.go` | Add `ExportWatchAndResume` function |
| `internal/cli/cli_test.go` | Add property test + 3 unit tests |
| `internal/cli/cli_agent.go` | Capture and log memFileSelector error |

---

## Phase 1: RED Tests

### Task 1: Add ExportWatchAndResume to export_test.go

**Files:**
- Modify: `internal/cli/export_test.go` (after line 51, before the `// --- Factory functions` comment)

- [ ] **Step 1: Add the export function**

Open `internal/cli/export_test.go` and insert the following between the `ExportWaitAndBuildPrompt` line and the `// --- Factory functions` comment block:

```go
// ExportWatchAndResume calls watchAndResume with all injectable dependencies for testing.
func ExportWatchAndResume(
	ctx context.Context,
	agentName, chatFilePath, stateFilePath string,
	cursor int,
	result claudepkg.StreamResult,
	stdout io.Writer,
	watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
	memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) (string, error) {
	return watchAndResume(ctx, agentName, chatFilePath, stateFilePath, cursor, result, stdout, watchForIntent, memFileSelector)
}
```

The `claudepkg` and `chat` imports are already present in the file. No new imports needed.

- [ ] **Step 2: Verify the file compiles**

```bash
targ build
```

Expected: build succeeds (no errors).

---

### Task 2: Write RED tests for memFileSelector error logging

**Files:**
- Modify: `internal/cli/cli_test.go` (append after the `selectMemoryFiles` test block, near line 4287)

- [ ] **Step 1: Add imports for rapid and claudepkg**

In `internal/cli/cli_test.go`, add to the import block:

```go
claudepkg "engram/internal/claude"
"pgregory.net/rapid"
```

The full import block becomes:

```go
import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	agentpkg "engram/internal/agent"
	"engram/internal/chat"
	"engram/internal/cli"
	claudepkg "engram/internal/claude"
)
```

- [ ] **Step 2: Append the four tests to cli_test.go**

Add the following block at the end of `internal/cli/cli_test.go` (after all existing tests):

```go
// ============================================================
// watchAndResume memFileSelector error logging tests (issue 547)
// ============================================================

// makeWatchAndResumeFixture creates a minimal temp dir and stub functions for
// ExportWatchAndResume tests. watchForIntent returns a single intent message then
// blocks forever (ctx cancel stops it). stateFilePath is pre-populated.
func makeWatchAndResumeFixture(t *testing.T) (stateFilePath string, watchForIntent func(context.Context, string, string, int) (chat.Message, int, error)) {
	t.Helper()

	dir := t.TempDir()
	stateFilePath = filepath.Join(dir, "state.toml")

	watchForIntent = func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		return chat.Message{
			From: "sender-agent",
			Text: "Situation: test. Behavior: test.",
		}, cursor + 5, nil
	}

	return stateFilePath, watchForIntent
}

// TestWatchAndResume_MemFileSelectorError_LogsWarning is a property-based test.
// For any arbitrary error message, when memFileSelector returns that error,
// the warning written to stdout must contain the error text.
func TestWatchAndResume_MemFileSelectorError_LogsWarning(t *testing.T) {
	t.Parallel()

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

		errMsg := rapid.StringOf(
			rapid.RuneFrom([]rune(
				"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 :/-_.",
			)),
		).Filter(func(s string) bool { return len(s) > 0 }).Draw(rt, "errMsg")

		memFileSelector := func(_ string, _ int) ([]string, error) {
			return nil, errors.New(errMsg)
		}

		var stdout bytes.Buffer

		_, err := cli.ExportWatchAndResume(
			context.Background(),
			"test-agent", "chat.toml", stateFilePath, 0,
			claudepkg.StreamResult{}, &stdout,
			watchForIntent, memFileSelector,
		)
		g.Expect(err).NotTo(HaveOccurred())
		if err != nil {
			return
		}

		g.Expect(stdout.String()).To(ContainSubstring(errMsg),
			"expected warning to contain the error message")
		g.Expect(stdout.String()).To(ContainSubstring("[engram] warning: failed to select memory files:"),
			"expected warning prefix")
	})
}

// TestWatchAndResume_MemFileSelectorError_ReturnsSuccessAndPrompt verifies that
// when memFileSelector returns an error, watchAndResume still returns a non-error
// result and a prompt containing the intent sender and text.
func TestWatchAndResume_MemFileSelectorError_ReturnsSuccessAndPrompt(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, errors.New("stat /nonexistent: no such file or directory")
	}

	var stdout bytes.Buffer

	prompt, err := cli.ExportWatchAndResume(
		context.Background(),
		"test-agent", "chat.toml", stateFilePath, 0,
		claudepkg.StreamResult{}, &stdout,
		watchForIntent, memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(prompt).To(ContainSubstring("INTENT_FROM: sender-agent"))
	g.Expect(prompt).To(ContainSubstring("INTENT_TEXT: Situation: test. Behavior: test."))
}

// TestWatchAndResume_MemFileSelectorSuccess_NoWarning verifies that when
// memFileSelector returns nil error, no warning is written to stdout.
func TestWatchAndResume_MemFileSelectorSuccess_NoWarning(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return []string{"/home/user/.local/share/engram/memory/facts/fact1.toml"}, nil
	}

	var stdout bytes.Buffer

	_, err := cli.ExportWatchAndResume(
		context.Background(),
		"test-agent", "chat.toml", stateFilePath, 0,
		claudepkg.StreamResult{}, &stdout,
		watchForIntent, memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	g.Expect(stdout.String()).NotTo(ContainSubstring("failed to select memory files"),
		"expected no warning when selector succeeds")
}

// TestWatchAndResume_MemFileSelectorError_MemFilesEmpty verifies that when
// memFileSelector returns an error, the resume prompt has an empty MEMORY_FILES
// section (fallback to no files preserved).
func TestWatchAndResume_MemFileSelectorError_MemFilesEmpty(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	stateFilePath, watchForIntent := makeWatchAndResumeFixture(t)

	memFileSelector := func(_ string, _ int) ([]string, error) {
		return nil, errors.New("reading directory /nonexistent: no such file or directory")
	}

	var stdout bytes.Buffer

	prompt, err := cli.ExportWatchAndResume(
		context.Background(),
		"test-agent", "chat.toml", stateFilePath, 0,
		claudepkg.StreamResult{}, &stdout,
		watchForIntent, memFileSelector,
	)
	g.Expect(err).NotTo(HaveOccurred())
	if err != nil {
		return
	}

	// MEMORY_FILES section should be present but empty (no file paths listed).
	// buildResumePrompt writes "MEMORY_FILES:\n" followed by one path per line.
	// When memFiles is nil, the section header is followed immediately by INTENT_FROM.
	g.Expect(prompt).To(ContainSubstring("MEMORY_FILES:\nINTENT_FROM:"),
		"expected MEMORY_FILES section to be empty when selector errors")
}
```

- [ ] **Step 3: Run the new tests to confirm they are RED**

```bash
targ test 2>&1 | grep -A 3 "TestWatchAndResume_MemFileSelectorError_LogsWarning\|TestWatchAndResume_MemFileSelectorError_ReturnsSuccessAndPrompt\|TestWatchAndResume_MemFileSelectorSuccess_NoWarning\|TestWatchAndResume_MemFileSelectorError_MemFilesEmpty\|FAIL"
```

Expected: the four new tests FAIL (because the warning is not yet logged). Existing tests still pass.

---

## Phase 2: GREEN Implementation

### Task 3: Add error logging in watchAndResume

**Files:**
- Modify: `internal/cli/cli_agent.go:1332-1337`

- [ ] **Step 1: Replace the silent error discard**

In `internal/cli/cli_agent.go`, replace lines 1332–1337:

```go
	if memFileSelector != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr == nil {
			memFiles, _ = memFileSelector(home, resumeMemoryFileLimit)
		}
	}
```

With:

```go
	if memFileSelector != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr == nil {
			var memErr error

			memFiles, memErr = memFileSelector(home, resumeMemoryFileLimit)
			if memErr != nil {
				_, _ = fmt.Fprintf(stdout,
					"[engram] warning: failed to select memory files: %v\n",
					memErr)
			}
		}
	}
```

No other changes. The function signature, return values, and all other behavior are unchanged.

- [ ] **Step 2: Run the new tests to confirm they are GREEN**

```bash
targ test 2>&1 | grep -A 3 "TestWatchAndResume_MemFileSelectorError_LogsWarning\|TestWatchAndResume_MemFileSelectorError_ReturnsSuccessAndPrompt\|TestWatchAndResume_MemFileSelectorSuccess_NoWarning\|TestWatchAndResume_MemFileSelectorError_MemFilesEmpty\|PASS\|FAIL"
```

Expected: all four new tests PASS.

- [ ] **Step 3: Run full test suite and quality checks**

```bash
targ check-full
```

Expected: all tests pass, no lint errors, coverage thresholds met.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/cli_agent.go internal/cli/export_test.go internal/cli/cli_test.go
```

```bash
git commit -m "$(cat <<'EOF'
fix(cli): log memFileSelector error in watchAndResume (#547)

Previously the error from memFileSelector was silently discarded with
'_', giving operators no visibility into why memory files were absent
from the resume prompt. Now logs a warning to stdout matching the
existing pattern for silentErr and resumeErr in the same function.

AI-Used: [claude]
EOF
)"
```

---

## Phase 3: Simplify and Refactor

### Task 4: Run simplify skill

- [ ] **Step 1: Invoke the simplify skill**

Run `/simplify` to review the changed code for quality and redundancy.

- [ ] **Step 2: Run check-full again after any simplify changes**

```bash
targ check-full
```

Expected: all tests pass, no lint errors.

- [ ] **Step 3: Commit any simplify changes (if any)**

If the simplify skill produced changes:

```bash
git add internal/cli/cli_agent.go internal/cli/export_test.go internal/cli/cli_test.go
```

```bash
git commit -m "$(cat <<'EOF'
refactor(cli): simplify watchAndResume memFileSelector error handling (#547)

AI-Used: [claude]
EOF
)"
```

---

## Self-Review

**Spec coverage:**

| Spec requirement | Task |
|---|---|
| Log warning when memFileSelector errors | Task 3 |
| Preserve empty-result fallback | Task 3 (memFiles stays nil), Test: `MemFilesEmpty` |
| Return valid prompt on error | Test: `ReturnsSuccessAndPrompt` |
| No error propagated to caller | Test: `ReturnsSuccessAndPrompt` (checks `err == nil`) |
| No warning when selector succeeds | Test: `MemFileSelectorSuccess_NoWarning` |
| All existing tests pass | Task 3, Step 3 (`targ check-full`) |

All spec requirements covered. No gaps.

**Placeholder scan:** No TBDs, TODOs, or vague steps. All code blocks are complete.

**Type consistency:** `ExportWatchAndResume` signature matches `watchAndResume` signature exactly. `claudepkg.StreamResult{}` is the zero value for the ignored `_` parameter.
