# Design: Hold Flag Validation (Issue #518)

**Date:** 2026-04-06  
**Scope:** Bug fix — `runHoldAcquire` and `runHoldRelease` in `internal/cli/cli.go`

## Problem

`engram hold acquire` and `engram hold release` silently accept missing required flags, exiting 0 with invalid/empty state in the hold registry. No user feedback; incorrect behavior.

## Design

**Approach:** Inline validation after flag parse, before any I/O.

### `runHoldAcquire` (line 717–724)

After the existing `if parseErr != nil` block, add:

```go
if *holder == "" {
    return fmt.Errorf("hold acquire: --holder is required")
}
if *target == "" {
    return fmt.Errorf("hold acquire: --target is required")
}
```

### `runHoldRelease` (line 892–898)

After the existing `if parseErr != nil` block, add:

```go
if *holdID == "" {
    return fmt.Errorf("hold release: --hold-id is required")
}
```

## Error Messages

Consistent with existing prefix pattern (`"hold acquire: ..."`, `"hold release: ..."`):
- `hold acquire: --holder is required`
- `hold acquire: --target is required`
- `hold release: --hold-id is required`

## Testing

Three new test functions in `internal/cli/cli_test.go`:
- `TestRun_HoldAcquire_EmptyHolder_ReturnsError`
- `TestRun_HoldAcquire_EmptyTarget_ReturnsError`
- `TestRun_HoldRelease_EmptyHoldID_ReturnsError`

Pattern: follow `TestRun_HoldAcquire_ParseError_ReturnsError` — call `cli.Run` with a `--chat-file` temp dir override, omit the required flag, assert `HaveOccurred()`.

## Constraints

- No new abstractions. Inline only.
- Error messages must match the issue's expected format.
- All tests must use `t.Parallel()`.
- `targ check-full` must pass.
