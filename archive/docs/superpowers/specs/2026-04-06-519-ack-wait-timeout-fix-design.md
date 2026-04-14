# Design: Fix AckWait Timeout (Issue #519)

**Date:** 2026-04-06
**Issue:** [#519 — bug: ack-wait timeout does not fire while Watch inner loop blocks on fsnotify](https://github.com/toejough/engram/issues/519)

---

## Problem

`AckWait` loops calling `Watch`. The timeout check runs at the **top** of the AckWait loop, before calling Watch. Watch blocks internally on `WaitForChange` until a file change arrives or its context is cancelled. The context passed in has no deadline, so Watch never returns, and the timeout check never re-runs.

**Result:** `engram chat ack-wait --max-wait N` blocks indefinitely when the recipient is online but silent.

---

## Root Cause

```go
// AckWait loop (ackwaiter.go)
for {
    nowCheck := w.NowFunc()
    if result, timedOut := checkOnlineSilentTimeout(..., nowCheck, maxWait, ...); timedOut {
        return result, nil  // unreachable — Watch never returns
    }

    // Watch holds the goroutine here:
    msg, cursor, err := w.Watcher.Watch(ctx, ...)  // ctx has no deadline
}

// Watch (watcher.go)
for {
    if found := findMessage(...); found {
        return msg, cursor, nil
    }
    w.FSWatcher.WaitForChange(ctx, path)  // blocks here; only unblocks on file change or ctx cancel
}
```

---

## Approach Considered

### Option A — Per-iteration `context.WithTimeout(ctx, maxWait)`
Rejected: file changes from other agents (health-checker) reset the maxWait window on each iteration. Timeout never fires if any agent posts within the window.

### Option B — One-time deadline before the loop (recommended)
Compute `watchDeadline = w.NowFunc().Add(maxWait)` once. Each loop iteration wraps the context with this fixed deadline. When the deadline fires, `WaitForChange` returns `DeadlineExceeded`, `Watch` propagates it, `AckWait` loops back, and the timeout check fires.

**Rationale for `w.NowFunc()` over `time.Now()`:**
- `context.WithDeadline` uses the real clock, but in tests where `fakeNow` returns a fixed past time, `watchDeadline` lands in the past, causing the context to be already-cancelled at construction time. This makes tests fast (no real sleep needed) and preserves the existing fake-time test pattern.
- Production uses `NowFunc = time.Now` — behaves identically to `time.Now().Add(maxWait)`.

---

## Design

### Change scope
One file: `internal/chat/ackwaiter.go`. No interface changes. No changes to `watcher.go`.

### Implementation

In `AckWait`, immediately after resolving `maxWait`:

```go
watchDeadline := w.NowFunc().Add(maxWait)
```

Replace the Watch call:
```go
// Before:
msg, newCursor, watchErr := w.Watcher.Watch(ctx, callerAgent, currentCursor, []string{"ack", "wait"})

// After:
watchCtx, watchCancel := context.WithDeadline(ctx, watchDeadline)
msg, newCursor, watchErr := w.Watcher.Watch(watchCtx, callerAgent, currentCursor, []string{"ack", "wait"})
watchCancel()
```

`watchCancel()` is called inline (not deferred) because we're in a loop — resources should be released on each iteration, not only when the function returns.

### Behaviour invariants

| Scenario | Behaviour |
|----------|-----------|
| File changes (health-checker posts) | `WaitForChange` returns early. Watch re-reads, finds no ack/wait, calls `WaitForChange` again with the same already-expired-if-past deadline → immediate DeadlineExceeded → AckWait re-checks timeout. ✓ |
| ACK/WAIT arrives before deadline | Watch returns message before hitting `WaitForChange`. Deadline irrelevant. ✓ |
| Outer ctx cancelled | Top-of-loop `ctx.Err()` check fires first, returns error. Correct. ✓ |
| Deadline already past (fake-time tests) | `context.WithDeadline(ctx, pastTime)` returns already-cancelled context. Watch returns DeadlineExceeded immediately. Loop re-runs. fakeNow returns advanced time. Timeout check fires. ✓ |

---

## Testing

### Existing tests
All existing `FileAckWaiter` tests continue to pass without modification:
- Tests with `fakeNow` returning fixed past time: `watchDeadline` is in the past → context immediately cancelled → Watch returns DeadlineExceeded immediately → loop re-runs with advanced fake time → correct result fires before outer test ctx
- Tests where `fakeWatch` returns immediately (doesn't block on ctx): unaffected — message found before `WaitForChange`

### New test required (AC from issue)

`TestFileAckWaiter_OnlineSilentTIMEOUT_ViaWatchDeadline` in `ackwaiter_test.go`:

- Recipient has a recent message (online)
- `maxWait = 100ms`, `NowFunc = time.Now` (real)
- `fakeWatch` blocks on `ctx.Done()`
- Safety outer ctx: 5s (much longer than `maxWait`)
- No outer ctx timeout needed to trigger the fix — the watch deadline alone fires

**RED behaviour (without fix):** AckWait blocks until safety ctx (5s) fires → Watch returns `DeadlineExceeded` → loop → outer ctx also expired → `ctx.Err()` returns error → test fails at `Expect(err).NotTo(HaveOccurred())`.

**GREEN behaviour (with fix):** Watch unblocks after ~100ms → TIMEOUT returned → test passes.

---

## Acceptance Criteria

- [ ] `engram chat ack-wait --max-wait N` returns `TIMEOUT` JSON within ~N seconds when recipient is online but silent
- [ ] Returns `ACK` immediately when all recipients respond before the deadline
- [ ] Returns `WAIT` immediately when any recipient posts a wait message
- [ ] All existing ack-wait tests pass
- [ ] New test `TestFileAckWaiter_OnlineSilentTIMEOUT_ViaWatchDeadline` passes
