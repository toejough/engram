# Issue 550: Outer-Loop Cursor Continuity Not Verified

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two tests that assert the cursor value passed to `watchForIntent` matches the chat-file line count at the end of the prior session, catching any regression that resets it to zero.

**Architecture:** Two new tests in `internal/cli/cli_test.go` using the existing `ExportRunConversationLoopWith` entry point and a capturing `watchForIntent` closure. A `rapid.Check` property-based test varies the number of initial chat-file lines and verifies the cursor in all cases; a companion unit test runs two outer-loop iterations and confirms neither cursor is reset. No production code changes are anticipated.

**Tech Stack:** Go, gomega, `pgregory.net/rapid`, `internal/cli` (`ExportRunConversationLoopWith`)

---

## File Map

| File | Change |
|------|--------|
| `internal/cli/cli_test.go` | Add `pgregory.net/rapid` import; add `TestOuterWatchLoop_WatchForIntentCursorMatchesEndOfSession` and `TestOuterWatchLoop_CursorNotResetBetweenSessions` after line 1204 |

No changes to `export_test.go`, production code, or other files unless Phase 2 uncovers a bug.

---

## Phase 1 — RED: Write the Tests

### Task 1: Add `rapid` import to `cli_test.go`

**Files:**
- Modify: `internal/cli/cli_test.go` (import block, lines 3–24)

- [ ] **Step 1.1: Add `pgregory.net/rapid` to the import block**

Open `internal/cli/cli_test.go`. The current import block (lines 3–24) is:

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

	agentpkg "engram/internal/agent"
	"engram/internal/chat"
	"engram/internal/cli"
)
```

Replace it with (adding `pgregory.net/rapid` in the third-party group):

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
)
```

---

### Task 2: Add the property-based test

**Files:**
- Modify: `internal/cli/cli_test.go` (insert after line 1204, the closing `}` of `TestOuterWatchLoop_WriteStateSilentAfterSession`)

- [ ] **Step 2.1: Insert `TestOuterWatchLoop_WatchForIntentCursorMatchesEndOfSession` after line 1204**

Insert the following test immediately after the closing `}` of `TestOuterWatchLoop_WriteStateSilentAfterSession` (line 1204) and before `func TestOutputAckResult_FailWriter_ReturnsError` (line 1206):

```go
// TestOuterWatchLoop_WatchForIntentCursorMatchesEndOfSession is a property-based test
// verifying that the cursor passed to watchForIntent equals the chat-file line count
// (chatFileCursor result) at the end of the prior session, for any initial file size.
// A regression hardcoding cursor=0 would fail this test for any non-trivial content.
func TestOuterWatchLoop_WatchForIntentCursorMatchesEndOfSession(t *testing.T) {
	t.Parallel()

	// Fake claude binary shared across all rapid cases: emits DONE immediately.
	claudeDir := t.TempDir()
	fakeClaude := filepath.Join(claudeDir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	if writeErr := os.WriteFile(fakeClaude, []byte(script), 0o700); writeErr != nil {
		t.Fatal(writeErr)
	}

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"

	rapid.Check(t, func(rt *rapid.T) {
		g := NewGomegaWithT(rt)

		// Generate a random number of content lines [1, 20].
		numLines := rapid.IntRange(1, 20).Draw(rt, "numLines")
		content := strings.Repeat("comment\n", numLines)
		// chatFileCursor = len(strings.Split(content, "\n")) — same formula as production code.
		expectedCursor := len(strings.Split(content, "\n"))

		// Each rapid case gets its own temp dir to avoid cross-case interference.
		dir := t.TempDir()
		chatFile := filepath.Join(dir, "chat.toml")
		stateFile := filepath.Join(dir, "state.toml")

		g.Expect(os.WriteFile(chatFile, []byte(content), 0o600)).To(Succeed())
		g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// watchForIntent captures the cursor it receives, then cancels ctx (clean exit).
		var capturedCursor int
		watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
			capturedCursor = cursor
			cancel()

			return chat.Message{}, 0, context.Canceled
		}

		err := cli.ExportRunConversationLoopWith(
			ctx,
			"worker-1", "hello", chatFile, stateFile, fakeClaude,
			io.Discard,
			func(_ context.Context, _, _ string, _ int) (string, error) { return "Proceed.", nil },
			watchForIntent,
			nil,
		)
		g.Expect(err).NotTo(HaveOccurred())

		if err != nil {
			return
		}

		g.Expect(capturedCursor).To(Equal(expectedCursor),
			"cursor passed to watchForIntent must equal chatFileCursor of chat file (numLines=%d)", numLines)
	})
}
```

- [ ] **Step 2.2: Run the new test in isolation to verify RED/GREEN state**

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -E "WatchForIntentCursorMatches|FAIL|ok"
```

Expected: test passes GREEN immediately (this is a coverage gap, not a missing implementation). If it fails, record the failure and proceed to Phase 2.

---

### Task 3: Add the multi-session continuity test

**Files:**
- Modify: `internal/cli/cli_test.go` (insert immediately after the test added in Task 2)

- [ ] **Step 3.1: Insert `TestOuterWatchLoop_CursorNotResetBetweenSessions` after the previous test**

```go
// TestOuterWatchLoop_CursorNotResetBetweenSessions verifies that across two outer-loop
// iterations, the cursor passed to watchForIntent in each iteration is non-zero and
// matches the chat-file state. Confirms the outer loop does not reset cursor to 0
// between sessions.
func TestOuterWatchLoop_CursorNotResetBetweenSessions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	dir := t.TempDir()
	chatFile := filepath.Join(dir, "chat.toml")
	stateFile := filepath.Join(dir, "state.toml")

	// Five lines of content — chatFileCursor = len(strings.Split(content, "\n")) = 6.
	const numLines = 5
	content := strings.Repeat("comment\n", numLines)
	expectedCursor := len(strings.Split(content, "\n"))

	g.Expect(os.WriteFile(chatFile, []byte(content), 0o600)).To(Succeed())

	stateToml := "[[agent]]\nname = \"worker-1\"\npane_id = \"\"\n" +
		"session_id = \"\"\nstate = \"STARTING\"\nspawned_at = 2026-04-06T00:00:00Z\n"
	g.Expect(os.WriteFile(stateFile, []byte(stateToml), 0o600)).To(Succeed())

	// Fake claude emits DONE on every call.
	fakeClaude := filepath.Join(dir, "claude")
	doneJSON := `{"type":"assistant","session_id":"sess-abc",` +
		`"message":{"content":[{"type":"text","text":"DONE: All done."}]}}`
	script := "#!/bin/sh\nprintf '%s\\n' '" + doneJSON + "'\n"
	g.Expect(os.WriteFile(fakeClaude, []byte(script), 0o700)).To(Succeed())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// watchForIntent captures cursor for calls 1 and 2; cancels ctx on call 2.
	var cursor1, cursor2 int

	callCount := 0
	watchForIntent := func(_ context.Context, _, _ string, cursor int) (chat.Message, int, error) {
		callCount++

		if callCount == 1 {
			cursor1 = cursor

			return chat.Message{
				From: "lead",
				Type: "intent",
				Text: "Resume now.",
			}, cursor + 1, nil
		}

		cursor2 = cursor
		cancel()

		return chat.Message{}, 0, context.Canceled
	}

	err := cli.ExportRunConversationLoopWith(
		ctx,
		"worker-1", "hello", chatFile, stateFile, fakeClaude,
		io.Discard,
		func(_ context.Context, _, _ string, _ int) (string, error) { return "Proceed.", nil },
		watchForIntent,
		nil,
	)
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(callCount).To(Equal(2), "watchForIntent must be called exactly twice")
	g.Expect(cursor1).To(BeNumerically(">", 0), "first watchForIntent cursor must be non-zero")
	g.Expect(cursor1).To(Equal(expectedCursor), "cursor1 must match chatFileCursor of initial content")
	g.Expect(cursor2).To(BeNumerically(">", 0), "second watchForIntent cursor must be non-zero")
	g.Expect(cursor2).To(Equal(expectedCursor), "cursor2 must equal cursor1 (chat file unchanged between sessions)")
}
```

- [ ] **Step 3.2: Run both new tests to verify GREEN state**

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -E "WatchForIntentCursorMatches|CursorNotReset|FAIL|ok"
```

Expected: both tests pass GREEN.

- [ ] **Step 3.3: Commit Phase 1**

```bash
cd /Users/joe/repos/personal/engram
git add internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
test(cli): add cursor continuity assertions at watchForIntent boundary (#550)

Property-based test verifies cursor passed to watchForIntent equals
chatFileCursor for any initial chat-file size. Multi-session test
confirms cursor is not reset to 0 between outer-loop iterations.

AI-Used: [claude]
EOF
)"
```

---

## Phase 2 — GREEN: Fix if Needed

### Task 4: Diagnose and fix if Phase 1 tests failed

- [ ] **Step 4.1: Check if Phase 1 tests both passed**

If both tests in Phase 1 passed GREEN: **skip this entire task**. Phase 2 has no work to do.

If either test failed, continue to the steps below.

- [ ] **Step 4.2 (only if a test failed): Read the failure output**

The most likely failure patterns:

| Symptom | Likely cause |
|---------|-------------|
| `capturedCursor == 0` | `runConversationLoopWith` is hardcoding or resetting cursor to 0 before calling `watchAndResume` |
| `capturedCursor != expectedCursor` | `runWithinSessionLoop` is returning a stale `lastCursor` (not refreshed before last turn) |
| `callCount == 0` | `watchForIntent` was never called — `watchAndResume` is not invoked after DONE |

- [ ] **Step 4.3 (only if a test failed): Inspect the cursor flow**

Read `internal/cli/cli_agent.go` lines 1063–1109 (`runConversationLoopWith`) and lines 1146–1197 (`runWithinSessionLoop`). Verify:

1. `runWithinSessionLoop` calls `chatFileCursor(chatFilePath, os.ReadFile)` at the top of each inner iteration and assigns to `lastCursor`.
2. `runWithinSessionLoop` returns `lastCursor` as the third return value.
3. `runConversationLoopWith` passes that `cursor` value to `watchAndResume` without modification.
4. `watchAndResume` passes `cursor` to `watchForIntent` without modification.

If any step in this chain is broken, fix it.

- [ ] **Step 4.4 (only if a test failed): Re-run after fix**

```bash
cd /Users/joe/repos/personal/engram && targ test 2>&1 | grep -E "WatchForIntentCursorMatches|CursorNotReset|FAIL|ok"
```

Expected: both tests PASS.

- [ ] **Step 4.5 (only if production code was changed): Commit the fix**

```bash
cd /Users/joe/repos/personal/engram
git add internal/cli/cli_agent.go internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
fix(cli): cursor passed to watchForIntent was reset to 0 (#550)

runWithinSessionLoop was not propagating lastCursor correctly through
the outer loop. watchForIntent now receives the real chat-file position
from the end of the prior session.

AI-Used: [claude]
EOF
)"
```

---

## Phase 3 — Simplify and Verify

### Task 5: Simplify changed code and run full quality checks

- [ ] **Step 5.1: Run the simplify skill on changed code**

Invoke `superpowers:simplify` to review the two new test functions for clarity, redundancy, and consistency with the rest of `cli_test.go`. Apply any suggestions that reduce complexity without changing behavior.

- [ ] **Step 5.2: Run full quality checks**

```bash
cd /Users/joe/repos/personal/engram && targ check-full 2>&1
```

Expected: all tests pass, lint clean, coverage thresholds met. If lint errors appear: fix them before committing.

- [ ] **Step 5.3: Commit Phase 3 (only if Step 5.1 made changes)**

If simplify made no changes, skip this commit.

```bash
cd /Users/joe/repos/personal/engram
git add internal/cli/cli_test.go
git commit -m "$(cat <<'EOF'
refactor(cli): simplify cursor continuity tests (#550)

AI-Used: [claude]
EOF
)"
```
