# Design: Log memFileSelector Error in watchAndResume (Issue 547)

**Date:** 2026-04-10
**Status:** Approved for implementation

## Problem

In `internal/cli/cli_agent.go:1335`, the error returned by `memFileSelector` is silently discarded:

```go
memFiles, _ = memFileSelector(home, resumeMemoryFileLimit)
```

When `selectMemoryFiles` fails (e.g., because the feedback or facts directory does not exist), no warning is logged. Operators have no visibility into why memory files were not injected into the resume prompt. The agent silently falls back to an empty file list.

## Scope

- **In scope:** Log the `memFileSelector` error as a warning to `stdout`. Preserve the empty-result fallback.
- **Out of scope:** The `homeErr` silent-drop at line 1334 (separate issue). Changes to `selectMemoryFiles` behavior.

## Solution

Apply the same warning pattern already used in `watchAndResume` for `silentErr` (line 1302) and `resumeErr` (line 1324):

```go
// Before:
memFiles, _ = memFileSelector(home, resumeMemoryFileLimit)

// After:
var memErr error
memFiles, memErr = memFileSelector(home, resumeMemoryFileLimit)
if memErr != nil {
    _, _ = fmt.Fprintf(stdout,
        "[engram] warning: failed to select memory files: %v\n",
        memErr)
}
```

The function does not return the error — memory file injection failure is non-fatal. The resume prompt is still built and returned.

## Architecture

No new types, interfaces, or packages are needed. The change is three lines inside `watchAndResume`.

`watchAndResume` already receives `stdout io.Writer` as a parameter, so the warning pattern is directly applicable without any signature change.

## Testing

### Export

Add `ExportWatchAndResume` to `internal/cli/export_test.go` so tests can call `watchAndResume` directly without standing up the full conversation loop.

Signature:
```go
func ExportWatchAndResume(
    ctx context.Context,
    agentName, chatFilePath, stateFilePath string,
    cursor int,
    result claudepkg.StreamResult,
    stdout io.Writer,
    watchForIntent func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error),
    memFileSelector func(homeDir string, maxFiles int) ([]string, error),
) (string, error)
```

### Phase 1 — RED (property-based + unit tests)

All tests go in `internal/cli/cli_test.go`.

**Property test** (rapid): For any arbitrary error message string, when `memFileSelector` returns that error, `stdout` contains the message in the warning line.

**Unit test 1:** When `memFileSelector` returns an error, `watchAndResume` returns no error (fallback preserved).

**Unit test 2:** When `memFileSelector` returns an error, the returned prompt string is still correctly built from the intent message (prompt content unaffected).

**Unit test 3 (baseline):** When `memFileSelector` returns `nil` error, no warning is written to `stdout`. (Ensures the log is conditional, not always-on.)

### Phase 2 — GREEN

Apply the three-line change to `watchAndResume`. All RED tests pass.

### Phase 3 — Refactor

Run `simplify` skill. No structural changes expected — the fix is minimal.

## Error Handling

| Scenario | Behavior |
|---|---|
| `memFileSelector` returns error | Warning logged to stdout; `memFiles` remains nil; prompt built normally |
| `memFileSelector` returns nil error | No warning; `memFiles` used as returned |
| `memFileSelector` is nil (already guarded) | Block skipped entirely (unchanged) |

## Acceptance Criteria

1. When `memFileSelector` returns a non-nil error, stdout receives a line matching `[engram] warning: failed to select memory files: <err>`.
2. The function still returns a valid prompt string when the selector errors.
3. No error is propagated to the caller when the selector errors.
4. When the selector succeeds, no warning is written.
5. All existing tests continue to pass.
