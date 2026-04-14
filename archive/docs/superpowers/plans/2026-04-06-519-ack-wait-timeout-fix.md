# Fix AckWait Timeout (Issue #519) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix `engram chat ack-wait` so it returns `TIMEOUT` within `--max-wait` seconds when a recipient is online but silent, instead of blocking indefinitely.

**Architecture:** `AckWait` in `ackwaiter.go` currently passes the bare caller context to `Watch`, which has no deadline — so `WaitForChange` blocks forever. The fix captures `waitStart := w.NowFunc()` once (replacing the existing NowFunc call in `buildRecipientStates`), then derives `watchDeadline := waitStart.Add(maxWait)`. Each `Watch` call is wrapped with `context.WithDeadline(ctx, watchDeadline)`. When the deadline fires, `Watch` returns `context.DeadlineExceeded`, `AckWait` loops back, and the timeout check fires. No interface changes; NowFunc call count is preserved exactly so fake-time tests remain correct.

**Tech Stack:** Go 1.24, `context.WithDeadline`, `github.com/onsi/gomega`, `targ` build tool

---

## File Map

| File | Change |
|------|--------|
| `internal/chat/ackwaiter_test.go` | Add `TestFileAckWaiter_OnlineSilentTIMEOUT_ViaWatchDeadline` |
| `internal/chat/ackwaiter.go` | Capture `waitStart` once (replacing existing NowFunc call); derive `watchDeadline`; wrap Watch call with `context.WithDeadline` |

---

### Task 1: Write the Failing Test (RED)

**Files:**
- Modify: `internal/chat/ackwaiter_test.go`

- [ ] **Step 1.1: Add the new test to `ackwaiter_test.go`**

Add this function at the end of `internal/chat/ackwaiter_test.go`, before the closing of the file (after `TestFileAckWaiter_WatchInternalTimeoutThenACK`):

```go
func TestFileAckWaiter_OnlineSilentTIMEOUT_ViaWatchDeadline(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Recipient posted recently (online) but never responds.
	// Uses real NowFunc and short maxWait to prove Watch deadline fires the TIMEOUT path.
	// Without the fix: blocks until safety ctx (5s) fires → returns "ack wait cancelled" error → test fails.
	// With the fix: watchCtx deadline fires at ~100ms → Watch returns DeadlineExceeded →
	//   AckWait loops → real NowFunc sees elapsed >= maxWait → TIMEOUT returned → test passes.
	const maxWait = 100 * time.Millisecond

	recentMsg := chat.Message{
		From:   "engram-agent",
		To:     "all",
		Thread: "heartbeat",
		Type:   "info",
		TS:     time.Now().Add(-5 * time.Minute),
		Text:   "alive",
	}

	fakeRead := func(_ string) ([]byte, error) {
		return buildChatTOML([]chat.Message{recentMsg}), nil
	}

	// fakeWatch blocks until its context is cancelled (simulates no ack/wait arriving).
	fakeWatch := watcherFunc(func(ctx context.Context, _ string, cursor int, _ []string) (chat.Message, int, error) {
		<-ctx.Done()

		return chat.Message{}, cursor, ctx.Err()
	})

	waiter := &chat.FileAckWaiter{
		FilePath: "/fake/chat.toml",
		Watcher:  fakeWatch,
		ReadFile: fakeRead,
		NowFunc:  time.Now,
		MaxWait:  maxWait,
	}

	// Safety ctx: only fires if the fix is absent and the watch deadline doesn't work.
	// 5s >> 100ms maxWait so it never interferes with correct behaviour.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := waiter.AckWait(ctx, "caller", 0, []string{"engram-agent"})
	g.Expect(err).NotTo(HaveOccurred())

	if err != nil {
		return
	}

	g.Expect(result.Result).To(Equal("TIMEOUT"))
	g.Expect(result.Timeout).NotTo(BeNil())

	if result.Timeout == nil {
		return
	}

	g.Expect(result.Timeout.Recipient).To(Equal("engram-agent"))
}
```

- [ ] **Step 1.2: Run the test to confirm it fails**

```bash
cd /Users/joe/repos/personal/engram && targ test
```

Expected outcome: the new test blocks for ~5 seconds (safety ctx fires), then fails with something like:

```
FAIL: TestFileAckWaiter_OnlineSilentTIMEOUT_ViaWatchDeadline
Expected no error, but got: ack wait cancelled: context deadline exceeded
```

All other tests should still pass.

---

### Task 2: Apply the Fix (GREEN)

**Files:**
- Modify: `internal/chat/ackwaiter.go`

- [ ] **Step 2.1: Introduce `waitStart` and `watchDeadline`, wrap the Watch call**

In `internal/chat/ackwaiter.go`, the `AckWait` function currently reads (lines 31–79):

```go
func (w *FileAckWaiter) AckWait(
	ctx context.Context, callerAgent string, cursor int, recipients []string,
) (AckResult, error) {
	maxWait := w.MaxWait
	if maxWait == 0 {
		maxWait = defaultMaxWait
	}

	data, err := readFileOptional(w.ReadFile, w.FilePath)
	if err != nil {
		return AckResult{}, err
	}

	states := buildRecipientStates(data, recipients, w.NowFunc())
	currentCursor := cursor

	for {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			return AckResult{}, fmt.Errorf("ack wait cancelled: %w", ctxErr)
		}

		nowCheck := w.NowFunc()
		applyOfflineImplicit(states, nowCheck)

		if result, timedOut := checkOnlineSilentTimeout(states, nowCheck, maxWait, currentCursor); timedOut {
			return result, nil
		}

		if allResponded(states) {
			return AckResult{Result: "ACK", NewCursor: currentCursor}, nil
		}

		msg, newCursor, watchErr := w.Watcher.Watch(ctx, callerAgent, currentCursor, []string{"ack", "wait"})
		if watchErr != nil {
			if !errors.Is(watchErr, context.DeadlineExceeded) {
				return AckResult{}, fmt.Errorf("watching for ack: %w", watchErr)
			}
			// Watch's internal deadline exceeded — loop back to re-check offline/online timeouts.
			continue
		}

		currentCursor = newCursor

		if result, done := w.applyMsg(msg, states, currentCursor); done {
			return result, nil
		}
	}
}
```

Replace the entire function body with:

```go
func (w *FileAckWaiter) AckWait(
	ctx context.Context, callerAgent string, cursor int, recipients []string,
) (AckResult, error) {
	maxWait := w.MaxWait
	if maxWait == 0 {
		maxWait = defaultMaxWait
	}

	// waitStart is the reference point for both per-recipient offline detection and the
	// watch deadline. Captured once so NowFunc call count is unchanged from the original,
	// preserving existing fake-time test behaviour.
	waitStart := w.NowFunc()

	// watchDeadline is the fixed point at which Watch must unblock so AckWait can
	// re-evaluate the online-silent timeout. Fixed so unrelated file-change events
	// (health-checker heartbeats etc.) cannot reset the clock.
	watchDeadline := waitStart.Add(maxWait)

	data, err := readFileOptional(w.ReadFile, w.FilePath)
	if err != nil {
		return AckResult{}, err
	}

	states := buildRecipientStates(data, recipients, waitStart)
	currentCursor := cursor

	for {
		ctxErr := ctx.Err()
		if ctxErr != nil {
			return AckResult{}, fmt.Errorf("ack wait cancelled: %w", ctxErr)
		}

		nowCheck := w.NowFunc()
		applyOfflineImplicit(states, nowCheck)

		if result, timedOut := checkOnlineSilentTimeout(states, nowCheck, maxWait, currentCursor); timedOut {
			return result, nil
		}

		if allResponded(states) {
			return AckResult{Result: "ACK", NewCursor: currentCursor}, nil
		}

		watchCtx, watchCancel := context.WithDeadline(ctx, watchDeadline)
		msg, newCursor, watchErr := w.Watcher.Watch(watchCtx, callerAgent, currentCursor, []string{"ack", "wait"})
		watchCancel()

		if watchErr != nil {
			if !errors.Is(watchErr, context.DeadlineExceeded) {
				return AckResult{}, fmt.Errorf("watching for ack: %w", watchErr)
			}
			// Watch's internal deadline exceeded — loop back to re-check offline/online timeouts.
			continue
		}

		currentCursor = newCursor

		if result, done := w.applyMsg(msg, states, currentCursor); done {
			return result, nil
		}
	}
}
```

Key changes from the original:
1. `waitStart := w.NowFunc()` — single NowFunc call replaces the inline `w.NowFunc()` arg to `buildRecipientStates`; total call count preserved
2. `watchDeadline := waitStart.Add(maxWait)` — fixed deadline, derived from same `waitStart`
3. `buildRecipientStates(data, recipients, waitStart)` — uses captured `waitStart` instead of a second `w.NowFunc()` call
4. `watchCtx, watchCancel := context.WithDeadline(ctx, watchDeadline)` — wraps ctx per iteration
5. `w.Watcher.Watch(watchCtx, ...)` — passes bounded context instead of bare `ctx`
6. `watchCancel()` — called inline (not `defer`), releases resources each iteration

- [ ] **Step 2.2: Run all tests to confirm green**

```bash
cd /Users/joe/repos/personal/engram && targ test
```

Expected outcome: all tests pass, including `TestFileAckWaiter_OnlineSilentTIMEOUT_ViaWatchDeadline`, and that new test completes in ~100ms (not 5s).

---

### Task 3: Full Quality Check (REFACTOR)

**Files:** none modified

- [ ] **Step 3.1: Run full lint + coverage check**

```bash
cd /Users/joe/repos/personal/engram && targ check-full
```

Expected outcome: no lint errors, coverage thresholds pass. If `targ check-full` reports issues, fix them before proceeding. Do not suppress or ignore reported issues.

---

### Task 4: Commit

**Files:** all changes from Tasks 1–2

- [ ] **Step 4.1: Stage and commit**

```bash
cd /Users/joe/repos/personal/engram && git add internal/chat/ackwaiter.go internal/chat/ackwaiter_test.go
git commit -m "$(cat <<'EOF'
fix(chat): pass maxWait deadline into Watch so ack-wait timeout fires

AckWait's timeout check ran at the top of the loop, but Watch blocked
indefinitely on WaitForChange — the check never re-ran. Fix: compute a
fixed watchDeadline = NowFunc().Add(maxWait) once before the loop, then
wrap ctx with context.WithDeadline on each Watch call. When the deadline
fires, Watch returns DeadlineExceeded, the loop re-runs, and
checkOnlineSilentTimeout fires as designed.

Closes #519

AI-Used: [claude]
EOF
)"
```
