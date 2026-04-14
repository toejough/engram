# Phase 6 Binary Architecture Amendments

> **Context:** These amendments are additions to the main Phase 6 plan at
> `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md`.
> Written by phase6-arch planner. All 6 amendments ACKed by phase6-agent-e2e,
> phase6-skill, and phase6-user-e2e on the phase6-exec thread (2026-04-11).

---

## ARCH-A1: AgentRecord.LastDeliveredCursor field (agent.go)

**Gap:** The plan says "state file gains `last_delivered_cursor` per worker" (Codesign
Decisions table) but no task explicitly adds this field to `internal/agent/agent.go`.
Without it, `updateLastDeliveredCursor()` and crash recovery tests fail at compile time.

**Add to Task 3 Step 2 preamble** (before `dispatchLoop` implementation):

```
Files: Modify internal/agent/agent.go
```

- [ ] **Step 0: Add LastDeliveredCursor to AgentRecord**

  In `internal/agent/agent.go`, add to `AgentRecord` struct:

  ```go
  LastDeliveredCursor int `json:"last-delivered-cursor,omitzero" toml:"last-delivered-cursor,omitzero"`
  ```

  Place after `ArgumentThread` field to maintain logical grouping.

- [ ] **Step 0b: Write failing test**

  In `internal/agent/agent_test.go`, add `TestAgentRecordLastDeliveredCursorRoundTrip`:

  ```go
  func TestAgentRecordLastDeliveredCursorRoundTrip(t *testing.T) {
      t.Parallel()
      g := gomega.NewWithT(t)
      original := agent.StateFile{
          Agents: []agent.AgentRecord{{
              Name:                "engram-agent",
              State:               "SILENT",
              LastDeliveredCursor: 12345,
          }},
      }
      data, err := agent.MarshalStateFile(original)
      g.Expect(err).NotTo(HaveOccurred())
      if err != nil {
          return
      }
      parsed, parseErr := agent.ParseStateFile(data)
      g.Expect(parseErr).NotTo(HaveOccurred())
      if parseErr != nil {
          return
      }
      g.Expect(parsed.Agents).To(HaveLen(1))
      g.Expect(parsed.Agents[0].LastDeliveredCursor).To(Equal(12345))
  }
  ```

  ```bash
  targ test
  # Expected: fails (field not in struct)
  ```

- [ ] **Step 0c: Run tests green**

  ```bash
  targ test && targ check-full
  # Expected: pass
  ```

---

## ARCH-A2: matchesAgent empty-agent wildcard

**Gap:** Plan pseudocode shows `watcher.Next(ctx, cursor)` (no agent filter) for
`dispatchLoop`. This API does not exist. `chat.FileWatcher.Watch(ctx, agent, cursor, types)`
always filters by agent. Passing `agent=""` currently only matches messages with an
empty-string `to` field — not all messages.

**Fix:** A 3-line backward-compatible change to `internal/chat/watcher.go`.

**Add to Task 3 Step 1** (failing tests for `dispatchLoop`):

- [ ] **Step A2a: Write failing test for matchesAgent wildcard**

  In `internal/chat/watcher_test.go` (or `cli_dispatch_test.go`), add
  `TestMatchesAgentEmptyWildcard`:

  ```go
  func TestMatchesAgentEmptyWildcard(t *testing.T) {
      t.Parallel()
      g := gomega.NewWithT(t)
      // empty agent should match messages addressed to anyone
      g.Expect(chat.MatchesAgent("engram-agent", "")).To(BeTrue())
      g.Expect(chat.MatchesAgent("all", "")).To(BeTrue())
      g.Expect(chat.MatchesAgent("lead, engram-agent", "")).To(BeTrue())
      // non-empty agent: existing behavior unchanged
      g.Expect(chat.MatchesAgent("engram-agent", "lead")).To(BeFalse())
      g.Expect(chat.MatchesAgent("all", "lead")).To(BeTrue())
  }
  ```

  Note: `matchesAgent` must be exported as `MatchesAgent` for testing, OR tested
  via the `FileWatcher.Watch` observable behavior. Prefer the latter to avoid
  exporting an internal helper:

  ```go
  // TestFileWatcherWatchAllAgents: Watch with agent="" returns messages for any recipient
  func TestFileWatcherWatchAllAgents(t *testing.T) {
      t.Parallel()
      // post a message to "lead", watch with agent="" — should be returned
  }
  ```

  ```bash
  targ test
  # Expected: new test fails (agent="" treated as empty-string match, not wildcard)
  ```

- [ ] **Step A2b: Modify matchesAgent**

  In `internal/chat/watcher.go`, update `matchesAgent`:

  ```go
  // matchesAgent reports whether the To field targets the given agent.
  // The To field may be "all", a single agent name, or comma-separated names.
  // An empty agent string matches any To field (wildcard — used by dispatchLoop).
  func matchesAgent(to, agent string) bool {
      if agent == "" {
          return true // empty = match all recipients
      }
      for part := range strings.SplitSeq(to, ",") {
          trimmed := strings.TrimSpace(part)
          if trimmed == "all" || trimmed == agent {
              return true
          }
      }
      return false
  }
  ```

  `dispatchLoop` now uses `FileWatcher.Watch(ctx, "", cursor, []string{"intent","wait","shutdown"})`
  to receive all routable messages.

- [ ] **Step A2c: Run tests green**

  ```bash
  targ test && targ check-full
  ```

---

## ARCH-A3: watchAndResume intents + silentCh parameters

**Gap:** Task 1 says "in `watchAndResume`, when `intents != nil`, read from channel."
But `watchAndResume`'s current signature does not include `intents` or `silentCh`.
These must be threaded through.

**Add to Task 1 Step 2** (after the `runConversationLoopWith` signature update):

- [ ] **Step A3: Modify watchAndResume signature**

  In `internal/cli/cli_agent.go`, update `watchAndResume`:

  ```go
  func watchAndResume(
      ctx context.Context,
      agentName, chatFilePath, stateFilePath string,
      cursor int,
      _ claudepkg.StreamResult,
      stdout io.Writer,
      watchForIntent watchForIntentFunc,
      intents <-chan chat.Message,   // nil = use watchForIntent (standalone mode)
      silentCh chan<- string,        // nil when not in dispatch mode
      memFileSelector memFileSelectorFunc,
  ) (string, error) {
      // Write SILENT state first
      silentErr := writeAgentSilentState(stateFilePath, agentName)
      if silentErr != nil {
          _, _ = fmt.Fprintf(stdout, "[engram] warning: failed to write SILENT state: %v\n", silentErr)
      }
      // Signal dispatchLoop AFTER state file committed (drain only after SILENT is durable)
      if silentCh != nil {
          silentCh <- agentName
      }
      // ... ctx.Err() check ...
      var intentMsg chat.Message
      var newCursor int
      if intents != nil {
          select {
          case <-ctx.Done():
              return "", ctx.Err() //nolint:wrapcheck
          case msg := <-intents:
              intentMsg = msg
              newCursor = cursor // cursor advancing is dispatch's responsibility
          }
      } else {
          var watchErr error
          intentMsg, newCursor, watchErr = watchForIntent(ctx, agentName, chatFilePath, cursor)
          if watchErr != nil {
              if ctx.Err() != nil {
                  return "", ctx.Err() //nolint:wrapcheck
              }
              return "", fmt.Errorf("agent run: watch: %w", watchErr)
          }
      }
      // ... last-resumed-at, memFiles, buildResumePrompt ...
  }
  ```

  Update `runConversationLoopWith` to accept and pass `intents` and `silentCh`:

  ```go
  func runConversationLoopWith(
      ctx context.Context,
      runner claudepkg.Runner,
      agentName, initialPrompt string,
      chatFilePath, stateFilePath, claudeBinary string,
      stdout io.Writer,
      promptBuilder promptBuilderFunc,
      watchForIntent watchForIntentFunc,
      intents <-chan chat.Message,   // nil = standalone watch mode
      silentCh chan<- string,        // nil = standalone watch mode
      memFileSelector memFileSelectorFunc,
  ) error
  ```

  Existing callers (`runAgentRunWith`, `runAgentSpawnWith`) pass `nil, nil` for
  `intents` and `silentCh` — no behavior change.

---

## ARCH-A4: resumePromptArgs struct wrapper

**Gap:** Task 2 expands `buildResumePrompt` from 4 to 11 parameters. Positional
11-parameter functions are error-prone and produce unreadable tests.

**Add to Task 2 Step 3** (replace the 11-parameter signature with a struct):

- [ ] **Step A4: Define resumePromptArgs struct and update buildResumePrompt**

  In `internal/cli/cli_agent.go`:

  ```go
  // resumePromptArgs holds all fields for buildResumePrompt.
  // Separating args into a struct prevents positional-parameter bugs when
  // new fields are added (Phase 6 adds 7 new fields vs Phase 5's 4).
  type resumePromptArgs struct {
      AgentName       string
      Cursor          int
      MemFiles        []string
      IntentFrom      string
      IntentText      string
      RecentIntents   []string // from selectRecentIntents; empty → "(none)"
      ResumeReason    string   // "intent" | "wait" | "shutdown"
      WaitFrom        string   // populated when ResumeReason == "wait"
      WaitText        string   // populated when ResumeReason == "wait"
      ArgumentTurn    int      // populated when ResumeReason == "wait"
      LearnedMessages []string // from collectLearned; empty → "(none)"
  }

  func buildResumePrompt(args resumePromptArgs) string {
      var b strings.Builder
      fmt.Fprintf(&b, "AGENT_NAME: %s\n", args.AgentName)
      fmt.Fprintf(&b, "CURSOR: %d\n", args.Cursor)
      b.WriteString("MEMORY_FILES:\n")
      for _, f := range args.MemFiles {
          b.WriteString(f)
          b.WriteByte('\n')
      }
      fmt.Fprintf(&b, "INTENT_FROM: %s\n", args.IntentFrom)
      fmt.Fprintf(&b, "INTENT_TEXT: %s\n", args.IntentText)
      if len(args.RecentIntents) > 0 {
          fmt.Fprintf(&b, "RECENT_INTENTS: %s\n", strings.Join(args.RecentIntents, " | "))
      } else {
          b.WriteString("RECENT_INTENTS: (none)\n")
      }
      fmt.Fprintf(&b, "RESUME_REASON: %s\n", args.ResumeReason)
      if args.ResumeReason == "wait" {
          fmt.Fprintf(&b, "WAIT_FROM: %s\n", args.WaitFrom)
          fmt.Fprintf(&b, "WAIT_TEXT: %s\n", args.WaitText)
          fmt.Fprintf(&b, "ARGUMENT_TURN: %d\n", args.ArgumentTurn)
      }
      if len(args.LearnedMessages) > 0 {
          fmt.Fprintf(&b, "LEARNED_MESSAGES: %s\n", strings.Join(args.LearnedMessages, " | "))
      } else {
          b.WriteString("LEARNED_MESSAGES: (none)\n")
      }
      b.WriteString("Instruction: Load the files listed under MEMORY_FILES. " +
          "Use the CURSOR value when calling engram chat ack-wait. " +
          "Respond to the intent above with ACK:, WAIT:, or INTENT:.")
      return b.String()
  }
  ```

  One call site to update: `watchAndResume` (~line 1346):

  ```go
  return buildResumePrompt(resumePromptArgs{
      AgentName:       agentName,
      Cursor:          newCursor,
      MemFiles:        memFiles,
      IntentFrom:      intentMsg.From,
      IntentText:      intentMsg.Text,
      RecentIntents:   recentIntents,   // from selectRecentIntents()
      ResumeReason:    resumeReason,    // "intent"|"wait"|"shutdown"
      WaitFrom:        waitFrom,
      WaitText:        waitText,
      ArgumentTurn:    argumentTurn,
      LearnedMessages: learnedMessages, // from collectLearned()
  }), nil
  ```

---

## ARCH-A5: silentCh drain trigger for ACTIVE→SILENT

**Gap:** Codesign decision says "On ACTIVE→SILENT transition: drain `deferredQueues[worker]`"
but the trigger mechanism is unspecified. `dispatchLoop` has no way to learn when a
session completes without either polling or a notification channel.

**Complement phase6-agent-e2e ARCH-A4** (`drainDeferredQueue` helper) with:

**Add to Task 3 Step 2** (inside `dispatchLoop` + `runDispatch` implementation):

- [ ] **Step A5: Add silentCh to dispatchLoop and runDispatch**

  `dispatchLoop` signature:

  ```go
  func dispatchLoop(
      ctx context.Context,
      workerChans map[string]chan chat.Message,
      deferredQueues map[string][]chat.Message,
      holdChecker holdCheckerFunc,
      poster *chat.FilePoster,
      stateFilePath, chatFilePath string,
      cursor int,
      silentCh <-chan string, // receives worker name when session completes
  ) error
  ```

  Inside `dispatchLoop`, replace the watcher-only `for` loop with a `select`:

  ```go
  msgCh := make(chan watchResult, 1)
  go func() {
      msg, newCursor, err := watcher.Watch(ctx, "", cursor, []string{"intent","wait","shutdown","hold-release"})
      msgCh <- watchResult{msg, newCursor, err}
  }()
  for {
      select {
      case name := <-silentCh:
          drainDeferredQueue(name, deferredQueues, workerChans, stateFilePath)
      case res := <-msgCh:
          if res.err != nil { return res.err }
          cursor = res.cursor
          // ... routing logic ...
          // restart watcher goroutine
          go func() {
              msg, c, err := watcher.Watch(ctx, "", cursor, []string{"intent","wait","shutdown","hold-release"})
              msgCh <- watchResult{msg, c, err}
          }()
      case <-ctx.Done():
          return nil //nolint:nilerr
      }
  }
  ```

  `runDispatch` creates `silentCh` and passes it to both `dispatchLoop` and
  `runConversationLoopWith` (via ARCH-A3):

  ```go
  silentCh := make(chan string, len(workers)) // buffered: avoid blocking worker goroutine
  // pass silentCh to dispatchLoop and to each runConversationLoopWith call
  ```

  **State ordering:** `watchAndResume` writes SILENT state to state file FIRST, then
  sends on `silentCh`. `drainDeferredQueue` runs after state is durable. This prevents
  a window where deferredQueue is drained but state file still says ACTIVE.

---

## ARCH-A6: dispatchLoop non-blocking channel send guard

**Gap:** Plan shows `ch <- msg` (blocking) in `dispatchLoop`. If the per-worker
goroutine's 16-slot channel buffer fills up, `dispatchLoop` blocks — freezing routing
for ALL workers.

**Replace all `ch <- msg` in dispatchLoop routing logic with:**

```go
select {
case ch <- msg:
    updateLastDeliveredCursor(stateFilePath, recipient, cursor)
    postRoutingInfo(poster, recipient, msg, cursor)
default:
    // channel full or worker not yet consuming (STARTING state) — defer to deferredQueue
    if len(deferredQueues[recipient]) < maxDeferredQueueCap {
        deferredQueues[recipient] = append(deferredQueues[recipient], msg)
    } else {
        postQueueOverflow(poster, recipient)
    }
}
```

`deferredQueue` is now the **universal safety net** for:
1. Worker HELD (detected by `holdChecker` — existing logic)
2. Worker ACTIVE (detected by `isWorkerActive` — existing logic)
3. Channel full / worker STARTING (new — `default` branch above)

All three cases drain via the single `drainDeferredQueue` helper (phase6-agent-e2e A4).

**Add to Task 3 Step 1** (failing tests):

```go
// TestDispatchLoopChannelFullFallsToDeferredQueue:
// Pre-fill the per-worker channel to capacity (16 messages).
// Post one more intent via dispatchLoop.
// Verify it appears in deferredQueues[worker], not dropped.
// Verify dispatchLoop did NOT block (completed within 100ms).
func TestDispatchLoopChannelFullFallsToDeferredQueue(t *testing.T) {
    t.Parallel()
    // setup: create workerChan with cap 16, fill it
    workerChan := make(chan chat.Message, 16)
    for i := range 16 {
        workerChan <- chat.Message{From: "fill", To: "worker", Type: "intent",
            Text: fmt.Sprintf("fill %d", i)}
    }
    deferredQueues := map[string][]chat.Message{"worker": {}}
    // ... run dispatchLoop routing for one more intent ...
    // assert: deferredQueues["worker"] has len 1
    // assert: workerChan still has len 16 (overflow went to deferred, not channel)
}
```

---

## Interaction Map

| Amendment | Depends On | Required By |
|-----------|-----------|-------------|
| ARCH-A1 | — | ARCH-A5 (drainDeferredQueue uses LastDeliveredCursor), phase6-agent-e2e A4 |
| ARCH-A2 | — | ARCH-A5 (dispatchLoop passes agent="" to Watch) |
| ARCH-A3 | ARCH-A5 (silentCh) | ARCH-A5 (threadthrough) |
| ARCH-A4 | Task 2 (resumePromptArgs) | phase6-agent-e2e A2 (struct-form tests) |
| ARCH-A5 | ARCH-A2, ARCH-A3, phase6-agent-e2e A4 | Criterion 14 (WAIT to SILENT), Criterion 9 (Hold) |
| ARCH-A6 | ARCH-A5 (deferredQueue safety net) | Criterion 4 (Semaphore/queue), Criterion 18 |

---

## Summary of Plan Changes

| Task | Change |
|------|--------|
| Task 0 | Add Step 0: `AgentRecord.LastDeliveredCursor` in agent.go with round-trip test |
| Task 1 Step 2 | Extend `watchAndResume` + `runConversationLoopWith` with `intents`, `silentCh` params |
| Task 2 Step 3 | Replace 11-param `buildResumePrompt` with `resumePromptArgs` struct |
| Task 3 Step 1 | Add `TestMatchesAgentEmptyWildcard`, `TestDispatchLoopChannelFullFallsToDeferredQueue` |
| Task 3 Step 2 | matchesAgent wildcard; dispatchLoop with silentCh select loop; non-blocking channel send |
| `internal/chat/watcher.go` | `matchesAgent`: add `if agent == "" { return true }` |
| `internal/agent/agent.go` | `AgentRecord`: add `LastDeliveredCursor int` field |
