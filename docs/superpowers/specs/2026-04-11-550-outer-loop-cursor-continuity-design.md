# Design: Issue 550 — Outer-Loop Cursor Continuity Not Verified

**Date:** 2026-04-11
**Issue:** #550
**Status:** Draft

---

## Problem

`runConversationLoopWith` passes the `cursor` returned by `runWithinSessionLoop` to `watchAndResume`, which forwards it to `watchForIntent`. That cursor is the chat-file line count captured just before the last claude turn of the session — it must be a real file position, not zero.

All existing tests that inject a `watchForIntent` closure receive the `cursor int` parameter but never assert on its value. For example, `TestOuterWatchLoop_DoneThenWatchFires` (line 811 in `cli_test.go`) names the parameter `cursor int` only to use it as `cursor + 10` in the return value — no assertion that it is non-zero or that it matches the actual file position.

This means a regression where `runConversationLoopWith` hardcodes `0` (or resets the cursor between sessions) would go undetected.

### Untested Property

```
runConversationLoopWith → runWithinSessionLoop → chatFileCursor(file)
                       ↓
                  cursor returned
                       ↓
         watchAndResume(cursor) → watchForIntent(cursor)
                       ↑
         cursor must equal chatFileCursor(chatFile) at the point
         just before the last claude turn ran
```

---

## Approach

Use `ExportRunConversationLoopWith` (already exported) with a capturing `watchForIntent` closure. No new exports are needed.

**Two tests:**

1. **Property-based** (`rapid.Check`) — generates different amounts of initial chat-file content and asserts that, for every input, the cursor passed to `watchForIntent` equals the line count the file would produce at `chatFileCursor` time. Catches any hardcoding of `0` or incorrect arithmetic.

2. **Multi-session continuity** — two outer-loop iterations, each capturing the cursor passed to `watchForIntent`. Asserts both cursors are non-zero and match the file state, confirming the cursor is not reset between sessions.

Both tests write an initial content block to the chat file before the session. The fake claude script only writes to stdout — it does not modify the chat file. The state file is separate. Therefore the cursor at session end must equal `len(strings.Split(initialContent, "\n"))`.

---

## Test Specifications

### Test 1: `TestOuterWatchLoop_WatchForIntentCursorMatchesEndOfSession`

**File:** `internal/cli/cli_test.go`

**Location:** In the outer-watch-loop section, after `TestOuterWatchLoop_WriteStateSilentAfterSession`.

**Type:** Property-based (`rapid.Check`).

#### Property

For any `numLines ∈ [1, 20]`:
- Build `content = strings.Repeat("comment\n", numLines)` and write to `chatFile`.
- `expectedCursor = len(strings.Split(content, "\n"))` (same computation as `chatFileCursor`).
- Run `ExportRunConversationLoopWith` with a capturing `watchForIntent` closure.
- Assert `capturedCursor == expectedCursor`.

#### Rapid generator

```go
numLines := rapid.IntRange(1, 20).Draw(rt, "numLines")
content := strings.Repeat("comment\n", numLines)
expectedCursor := len(strings.Split(content, "\n"))
```

#### Setup (per rapid case)

- `t.TempDir()` inside the rapid callback for isolation.
- `chatFile` pre-written with `content`.
- `stateFile` with `worker-1` in `STARTING` state.
- Fake claude binary emitting DONE on first call (session ends immediately).
- `watchForIntent` closure: records `cursor int` into a local variable, then cancels ctx to end the outer loop.

#### Assertions

- `capturedCursor == expectedCursor` (the core property).
- `err == nil` (outer loop exits cleanly on ctx cancel).

---

### Test 2: `TestOuterWatchLoop_CursorNotResetBetweenSessions`

**File:** `internal/cli/cli_test.go`

**Location:** Immediately after Test 1.

**Type:** Standard unit test.

#### Goal

Verify that across two outer-loop iterations, the cursor passed to `watchForIntent` in each iteration is non-zero and matches the file state at the end of that session.

#### Setup

- Chat file pre-written with fixed content (e.g. 5 TOML comment lines).
- `expectedCursor = len(strings.Split(content, "\n"))`.
- Fake claude: DONE on every call.
- `watchForIntent`: captures cursor for calls 1 and 2; on call 2, cancels ctx.

#### Assertions

1. `cursor1 > 0` and `cursor1 == expectedCursor`
2. `cursor2 > 0` and `cursor2 == expectedCursor`
3. `err == nil`

Both cursors come from the same file state (fake claude doesn't write to chatFile), so they will be equal. The test confirms neither is zero and the outer loop doesn't reset the cursor between iterations.

---

## What "Passes Immediately" vs. "Bug Found"

The production code in `runWithinSessionLoop` already calls `chatFileCursor(chatFilePath, os.ReadFile)` correctly. Both tests are expected to **pass GREEN immediately** — this is a coverage gap, not a missing implementation. If either test fails, the cursor is being reset (a real bug to fix in Phase 2).

---

## TDD Phases

### Phase 1 — RED

Write both tests. Run `targ test`. Expected: both pass GREEN (coverage improvement, not a bug fix). If either fails, proceed to Phase 2.

### Phase 2 — GREEN

If Phase 1 passes: no changes. If Phase 1 fails: diagnose `runWithinSessionLoop` or `runConversationLoopWith` for cursor reset; fix; re-run.

### Phase 3 — Simplify

Run `superpowers:simplify` on changed code. Run `targ check-full`. Commit if any simplifications were made.

---

## Files Changed

| File | Change |
|------|--------|
| `internal/cli/cli_test.go` | Add `TestOuterWatchLoop_WatchForIntentCursorMatchesEndOfSession` and `TestOuterWatchLoop_CursorNotResetBetweenSessions` |

No changes to `export_test.go`, production code, or other files unless Phase 2 finds a bug.

---

## Import Dependencies

- `pgregory.net/rapid` — **not yet imported** in `cli_test.go`. Must be added to the import block.
- `strings` — already present
- `context`, `os`, `path/filepath`, `io` — already present
- `engram/internal/chat` — already present
