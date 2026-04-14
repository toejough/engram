# Design: Issue 548 — Outer Watch Loop No-Marker Path Test

**Date:** 2026-04-11
**Issue:** #548
**Status:** Draft

---

## Problem

`runConversationLoopWith` has two nested loops:

- **Inner loop** (`runWithinSessionLoop`): drives INTENT→ack-wait→resume cycles within a single claude session.
- **Outer loop** (`runConversationLoopWith` body): after a session ends, if `watchForIntent != nil`, calls `watchAndResume` to wait for the next intent and re-enter the inner loop.

There is no test that exercises the outer loop when the inner session emits **no markers** (no `INTENT:`, no `DONE:`, no `WAIT:`). The existing `TestRunAgentRun_FakeClaude_NoMarkers_ExitsClean` uses `nil` for `watchForIntent`, which exits at the nil-guard before the outer loop is entered. The comment in `TestRunConversationLoopWith_IntentThenDone` incorrectly claims it covers the outer watch loop — it also passes `nil` for `watchForIntent`.

### Untested Code Path

```
runConversationLoopWith (watchForIntent != nil):
  1. runWithinSessionLoop → result: no markers → returns cleanly
  2. watchForIntent != nil                        ← always true in production Phase 5
  3. watchAndResume:
     a. writeAgentState(SILENT)
     b. watchForIntent(ctx, name, chatFile, cursor) ← never called by any test
     c. writeAgentLastResumedAt
     d. buildResumePrompt → new prompt
  4. Loop again → runWithinSessionLoop → DONE → exit
```

---

## Approach

Use `ExportRunConversationLoopWith` (already exported for testing) with:

- A **flag-file fake claude binary** that emits plain text (no markers) on the first call, then `DONE:` on the second call.
- A **stub `watchForIntent`** function literal that returns a `chat.Message` immediately without blocking.
- A **pre-populated state file** with the agent in `STARTING` state (required by `writeAgentState`).

This is the same pattern used by `TestRunConversationLoopWith_IntentThenDone`.

---

## Test Specification

### Test: `TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches`

**File:** `internal/cli/cli_test.go`

**Location:** Near `TestRunConversationLoopWith_IntentThenDone` (around line 2729).

#### Setup

- Temp dir with `chat.toml` (empty) and `state.toml` with `worker-1` in `STARTING` state.
- Fake claude binary (flag-file pattern):
  - First call: emits plain assistant JSON with no markers (`"Here is your answer."`)
  - Second call: emits DONE JSON (`"DONE: All done."`)
- Stub `watchForIntent`: increments a call counter, returns a `chat.Message{From: "lead", Text: "Resume now."}` and cursor `0` with no error.
- Nil `memFileSelector` (no memory files needed).

#### Execution

```go
err := cli.ExportRunConversationLoopWith(
    ctx,
    "worker-1", "hello", chatFile, stateFile, fakeClaude,
    io.Discard,
    stubPromptBuilder,          // returns "Proceed." immediately
    stubWatchForIntent,         // returns fake intent immediately
    nil,                        // no mem file selector
)
```

#### Assertions

1. `err` is nil (clean exit)
2. `watchForIntentCallCount == 1` — called exactly once after the no-marker session
3. State file contains `state = "SILENT"` for `worker-1` (written before `watchForIntent`)

#### Why These Assertions

- (1) Confirms the outer loop exits cleanly after a second session with DONE.
- (2) Confirms `watchAndResume` called `watchForIntent` — the previously untested branch.
- (3) Confirms the state transition to SILENT was written (a key side effect of `watchAndResume`).

---

## Fix for Misleading Comment

The comment in `TestRunAgentRun_FakeClaude_NoMarkers_ExitsClean` says:
> "the outer watch loop is covered by TestRunConversationLoopWith_IntentThenDone"

This is incorrect — that test also passes `nil` for `watchForIntent`. Update the comment to reference the new test `TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches`.

Similarly, update the comment in `TestRunConversationLoopWith_IntentThenDone` to remove the incorrect outer-loop coverage claim.

---

## TDD Phases

### Phase 1 — RED

Write `TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches` as described above. Run `targ test`. The test is expected to **pass immediately** (the production code already handles this path). This confirms the gap was a missing test, not a missing implementation. If it fails, a bug exists and must be fixed in Phase 2.

### Phase 2 — GREEN

If Phase 1 test passes: no implementation change needed.
If Phase 1 test fails: diagnose and fix `watchAndResume` or `runConversationLoopWith`.

### Phase 3 — Simplify/Refactor

1. Fix the two misleading comments referencing the outer loop coverage.
2. Run `targ check-full` to confirm clean lint and coverage.
3. No production code changes unless Phase 2 required a fix.

---

## Files Changed

| File | Change |
|------|--------|
| `internal/cli/cli_test.go` | Add `TestRunConversationLoopWith_NoMarkers_OuterLoopRewatches`; fix two misleading comments |

No changes to `export_test.go`, production code, or other files unless Phase 2 finds a bug.
