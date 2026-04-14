# Design: Test WriteState(ACTIVE) on Second Outer-Loop Session (#549)

## Problem

`WriteState("ACTIVE")` is triggered when the `READY:` marker is detected in the stream
(claude.go:102 via `maybeWriteState`). The outer loop in `runConversationLoopWith`
(cli_agent.go:1063) runs multiple sessions, and each session should call
`WriteState("ACTIVE")` when the agent emits `READY:`.

Existing coverage:
- Session 1 ACTIVE: verified in `claude_test.go:539-570` (unit-level stream test)
- SILENT between sessions: verified in `cli_test.go:1149-1204`
- Two sessions running: verified in `cli_test.go:784-848`

**Gap:** No test runs the outer loop through two sessions and asserts
`WriteState("ACTIVE")` is called in both.

The problem with verifying ACTIVE via the state file: the outer loop writes SILENT to
the state file after each session ends (before watchForIntent is called). By the time
the test has a checkpoint, ACTIVE has already been overwritten. A polling goroutine
would be timing-sensitive and flaky.

## Design

### Observable WriteState via State Hook Export

Add a new test-only export `ExportRunConversationLoopWithStateHook` to `export_test.go`.
This export wraps the runner's `WriteState` with a caller-provided observer function,
recording every state value passed to WriteState across all sessions.

The observer is called **before** the real WriteState writes to the state file. If the
observer returns an error, the write is aborted. In the test, the observer records state
values to a mutex-protected slice.

This follows the existing injection pattern: `watchForIntent` and `memFileSelector` are
already injectable via `ExportRunConversationLoopWith`. The state hook extends that same
seam.

**New export signature:**

```go
func ExportRunConversationLoopWithStateHook(
    ctx context.Context,
    name, prompt, chatFile, stateFile, claudeBinary string,
    stdout io.Writer,
    promptBuilder func(ctx context.Context, agentName, chatFilePath string, turn int) (string, error),
    watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
    memFileSelector func(homeDir string, maxFiles int) ([]string, error),
    writeStateObserver func(state string) error, // called before each real WriteState write
) error
```

Implementation: builds runner via `buildAgentRunner`, then wraps `runner.WriteState`
to call the observer first, then the original write.

### New Test

`TestOuterWatchLoop_WriteStateActiveOnSecondSession` in `cli_test.go`:

- **Setup**: temp dir with chat + state files; fake claude script that emits `READY:` JSON
  then `DONE:` JSON (same format as existing outer loop tests)
- **WriteState observer**: mutex-protected `stateHistory []string` appended on each call
- **watchForIntent**: first call returns an intent (enabling session 2); second call
  cancels context
- **Assertion**: after the loop exits, count ACTIVE entries in stateHistory — expect
  exactly 2 (one per session)

The fake claude script format (matching existing tests):

```sh
#!/bin/sh
printf '%s\n' '{"type":"assistant","session_id":"sess-abc","message":{"content":[{"type":"text","text":"READY: Online."}]}}'
printf '%s\n' '{"type":"assistant","session_id":"sess-abc","message":{"content":[{"type":"text","text":"DONE: All done."}]}}'
```

### Files Changed

| File | Change |
|------|--------|
| `internal/cli/export_test.go` | Add `ExportRunConversationLoopWithStateHook` |
| `internal/cli/cli_test.go` | Add `TestOuterWatchLoop_WriteStateActiveOnSecondSession` |

No production code changes required. The behavior already works — this is a test gap.

## TDD Phases

### Phase 1 — RED (Write the failing test)

Write `TestOuterWatchLoop_WriteStateActiveOnSecondSession` calling
`ExportRunConversationLoopWithStateHook`. The export does not yet exist, so the package
fails to compile. This is the RED state.

### Phase 2 — GREEN (Add the export, verify the test passes)

Add `ExportRunConversationLoopWithStateHook` to `export_test.go`. The test should now
compile and pass, because `WriteState("ACTIVE")` is already triggered on every `READY:`
marker regardless of which session iteration it is. Run `targ test` to verify.

### Phase 3 — REFACTOR (Simplify)

Review for duplication:
- The new test shares setup structure with `TestOuterWatchLoop_DoneThenWatchFires` and
  `TestOuterWatchLoop_WriteStateSilentAfterSession`. If patterns are repetitive, extract
  a shared helper for the two-session outer loop scaffold (only if 3+ tests share it).
- Verify that no linter or nilaway issues are introduced by the new export or test.
- Run `targ check-full` to confirm zero issues.

## Acceptance Criteria

1. `TestOuterWatchLoop_WriteStateActiveOnSecondSession` exists and passes.
2. `stateHistory` contains exactly 2 entries of `"ACTIVE"` after a two-session run.
3. `targ check-full` passes with no new issues.
4. No production code modified.
