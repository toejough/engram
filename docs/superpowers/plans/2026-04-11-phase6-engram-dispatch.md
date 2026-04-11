# Phase 6 — Full Binary Dispatcher: engram dispatch

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace skill-driven bash watch loops with a single `engram dispatch` binary command that owns the full multi-agent event loop. The lead skill drops from ~1,225 lines to ~100 lines of pure judgment. Zero bash code survives in any skill after Phase 6. Binary manages all mechanics: worker lifecycle, intent routing, auto-resume, concurrency limits, and observability.

**Architecture:** `engram dispatch` generalizes Phase 5's per-agent `runConversationLoopWith` outer loop to N agents under one coordinator. A `dispatchLoop` goroutine watches the chat file for ALL messages, routes each `type=intent` to the correct worker goroutine via an in-process channel, and posts routing metadata to chat for observability. Workers continue to express coordination via speech-to-chat prefix markers (READY:, INTENT:, ACK:, WAIT:, DONE:). Five new subcommands expose the coordinator: `start`, `assign`, `drain`, `stop`, `status`. `engram agent run` (single-agent outer loop, Phase 5) is preserved for standalone engram-agent usage; the new channel-based path is additive.

**Spec:** `docs/superpowers/specs/engram-deterministic-coordination-design.md` §7 (Phase 6)

**Codesign session (initial):** planners planner-arch (binary architecture), planner-skill (skill rewrite and reduction), planner-e2e (agent end-to-end behavior), planner-user-e2e (user end-to-end experience) — 2026-04-11. All four perspectives converged before this plan was written. Codesign decisions are locked in the chat file (`~/.local/share/engram/chat/-Users-joe-repos-personal-engram.toml`) around line 125195–127088.

**Codesign session (delivery guarantees):** Same four planners — 2026-04-11. Filed issue toejough/engram#552. Agreed on 9 amendments for guaranteed message delivery, subprocess model (tmux-optional), and per-worker delivery tracking. Decisions locked in chat file around line 127934–130100. See "Delivery Guarantee" codesign decisions in table below.

**Tech Stack:** Go, existing `internal/claude`, `internal/chat`, `internal/agent`, `internal/cli`, `internal/watch` packages. No new dependencies.

---

## Pre-Flight (do before Task 0)

| Item | Action | Rationale |
|------|--------|-----------|
| Phase 5 E2E criteria | Verify all pass: auto-resume fires, STARTING→ACTIVE, queue max-3, resume prompt fields, no background monitor, stateless memory load, fresh session per loop | Phase 6 dispatch generalizes Phase 5 outer loop. Regressions in Phase 5 are Phase 5 bugs, not Phase 6 work. |
| `engram agent run` functional | `engram agent run --name engram-agent --prompt "..."` completes a full intent-process cycle | Phase 6 uses same `runConversationLoopWith` internals; must be clean before adding channel path. |
| `targ check-full` passes clean | Zero issues before Phase 6 code is added | Distinguish pre-existing issues from Phase 6 introductions. |
| State file schema stable | `engram agent list` outputs valid NDJSON with all required fields | Phase 6 extends the state file (argument state fields); schema must be at Phase 5 final shape first. |

---

## E2E Acceptance Criteria

Phase 6 is done when all binary criteria (1–13) pass before skill rewrites (Tasks 5–9), and all skill criteria (14–19) pass after.

### Criterion 1: Dispatch Start Spawns Workers

`engram dispatch start --agent engram-agent` spawns engram-agent as a subprocess, writes it to the state file as STARTING, and returns worker name(s) on stdout confirming the subprocess started.

```bash
OUTPUT=$(engram dispatch start --agent engram-agent)
echo "$OUTPUT"
# Expected: one worker-name line per agent, e.g. "engram-agent"
engram agent list | jq -r 'select(.name=="engram-agent") | .state'
# Expected: "STARTING" → "ACTIVE" after READY: marker
```

Note: `engram dispatch start` manages agents as subprocesses, not tmux panes. Tmux is not required for agent correctness. If tmux is available, the lead skill may open an optional chat-tail monitoring pane for user observability — but this is convenience, not a functional requirement.

### Criterion 2: Dispatch Routes Intent to Correct Worker

After `engram dispatch start`, posting an intent addressed to `engram-agent` causes the binary to resume engram-agent without any lead action.

```bash
engram chat post --from test --to engram-agent --thread test --type intent \
  --text "Situation: test. Behavior: respond ACK."
# Expected: binary auto-resumes engram-agent, pane shows new session output
```

### Criterion 3: FROM-Filter Prevents Self-Resume Loop

When engram-agent posts a nested intent addressed to itself (`from=engram-agent, to=engram-agent`), `dispatchLoop` does NOT route it back to the engram-agent worker channel. No second auto-resume fires.

```bash
# Post an intent that appears self-addressed:
engram chat post --from engram-agent --to engram-agent --thread test --type intent \
  --text "Situation: test nested. Behavior: this should not trigger self-resume."
# Expected: no new claude -p session starts; intent is silently dropped by dispatchLoop
```

### Criterion 4: Semaphore Limits Concurrent Workers

When 3 workers are ACTIVE or STARTING, a 4th intent queued to a new worker is buffered. The 4th worker does not resume until a slot opens.

```bash
# Spawn 3 workers, saturate concurrency
# Expected: 4th intent queued; "engram dispatch status" shows queue-depth=1
engram dispatch status | jq '.queue_depth'
# Expected: 1 (not 0)
```

### Criterion 5: Dispatch Assign Delivers Task

`engram dispatch assign --agent executor --task "..."` posts an intent to chat on the lead's behalf and delivers it to the executor worker.

```bash
engram dispatch assign --agent executor --task "Fix the auth handler in cmd/server/auth.go"
# Expected: cursor printed to stdout; executor resumes with task in prompt; chat file shows intent from lead
```

### Criterion 6: Observability Info Messages

All routing decisions appear in chat as `type=info` from `dispatch`:

```bash
engram chat post --from test --to engram-agent --thread test --type intent --text "Situation: test. Behavior: ACK."
# Expected chat messages (in order):
#   dispatch: "Resuming worker engram-agent for intent from test (cursor N)."
#   engram-agent: READY: → ACK: → DONE:
#   dispatch: "Worker engram-agent completed intent. State: SILENT."
```

### Criterion 7: Drain Waits for In-Flight Work

`engram dispatch drain --timeout 30` blocks until all ACTIVE/STARTING workers complete (state SILENT or DEAD), then outputs a JSON summary.

```bash
# Trigger an intent, immediately drain:
engram chat post --from test --to engram-agent --thread test --type intent \
  --text "Situation: test drain. Behavior: ACK."
RESULT=$(engram dispatch drain --timeout 30)
echo "$RESULT" | jq .
# Expected: {"completed":1,"failed":0,"queued":0}
```

### Criterion 8: Stop Sends Shutdown and Exits

`engram dispatch stop` sends `type=shutdown` to all registered workers, waits for their `done` messages, then exits. Binary exits with code 0. Workers post `done` messages in chat.

```bash
engram dispatch stop
# Expected: exit 0; chat file shows shutdown + done messages from all workers
```

### Criterion 9: Hold Blocks Auto-Resume

When a worker is on hold (`engram hold acquire --target <name>`), `dispatchLoop` queues intents for that worker rather than resuming. On hold release (`engram hold release`), the first queued intent resumes the worker.

```bash
engram hold acquire --target engram-agent --condition test-hold
engram chat post --from test --to engram-agent --thread test --type intent --text "..."
sleep 2
engram agent list | jq -r 'select(.name=="engram-agent") | .state'
# Expected: "SILENT" — not resumed while held
engram hold release <hold-id>
# Expected: worker resumes within 2s
```

### Criterion 10: Queue Depth Observable

`engram dispatch status` outputs queue depth and oldest queued intent age when intents are pending.

```bash
engram dispatch status | jq '{workers: [.workers[] | {name, state}], queue_depth, oldest_queued_age_s}'
# Expected: valid JSON; queue_depth >= 0
```

### Criterion 11: Crashed Worker Freed and Announced

When a worker exits with non-zero status, binary marks it `DEAD` in the state file, posts an info message to chat, and frees the concurrency slot.

```bash
# No easy way to crash a worker in tests; verify via state file + chat after
# deliberate kill: engram agent kill <name>
engram agent list | jq -r 'select(.name=="test-worker") | .state'
# Expected: "DEAD" after kill
# Chat: info message "Worker test-worker exited non-zero. Slot freed."
```

### Criterion 12: Dispatch Recover on Start (State File Repair)

`engram dispatch start` after a binary crash detects workers in the state file that are STARTING or ACTIVE (stale), marks them DEAD, posts announcement to chat, and spawns fresh workers.

```bash
# Manually write a stale ACTIVE entry to state file, then run:
engram dispatch start --agent engram-agent
# Expected: stale entry cleaned up; fresh worker spawned; no error
```

### Criterion 13: Resume Prompt Includes Agent Name and Recent Intent Summaries

The resume prompt delivered to each worker includes `AGENT_NAME: <name>` and `RECENT_INTENTS: <summary1> | <summary2> | ...` fields in addition to existing Phase 5 fields (CURSOR:, MEMORY_FILES:, INTENT_FROM:, INTENT_TEXT:). The RECENT_INTENTS field contains the last 5 intent-type messages from the chat file (from any agent, any thread), summarized as `from→to: text-truncated-at-80-chars`.

```bash
# Inspect pane after resume or enable debug logging in buildResumePrompt
# Expected: AGENT_NAME: engram-agent present in prompt
# Expected: RECENT_INTENTS: <N> entries present in prompt (N = min(5, total intents in chat))
# Expected: RESUME_REASON: intent (for normal intent-triggered resume)
# Expected: LEARNED_MESSAGES: (none) when no learned messages since last delivery
```

### Criterion 14: Delivery Guarantee — WAIT to SILENT Worker

Post an intent from worker A addressed to engram-agent. After ACK, worker A posts `done` and goes SILENT. Then post a `type=wait` from engram-agent addressed to worker A. Within 2 seconds, worker A resumes. Worker A's resume prompt contains `RESUME_REASON: wait`, `WAIT_FROM: engram-agent`, `WAIT_TEXT: <the wait message text>`, `ARGUMENT_TURN: 1`.

```bash
engram chat post --from engram-agent --to worker-a --thread test --type wait \
  --text "Situation: conflict found. Behavior: requesting pause."
sleep 2
engram agent list | jq -r 'select(.name=="worker-a") | .state'
# Expected: "ACTIVE" (resumed for the WAIT)
```

### Criterion 15: Delivery Guarantee — Shutdown to SILENT Worker

Send `type=shutdown` to a SILENT registered worker. Within 2 seconds, the worker resumes, posts `done`, and exits. Binary marks it DEAD in state file.

```bash
engram chat post --from lead --to engram-agent --thread lifecycle --type shutdown \
  --text "Session complete."
sleep 2
engram agent list | jq -r 'select(.name=="engram-agent") | .state'
# Expected: "DEAD" (resumed, posted done, exited)
```

### Criterion 16: Delivery Guarantee — to=all Broadcast

Post a message with `to=all` of type `wait`. ALL registered workers (not just `engram-agent`) receive it and resume within 2 seconds.

```bash
engram chat post --from test --to all --thread test --type wait \
  --text "Situation: broadcast test. Hold all workers."
sleep 2
engram agent list | jq '[.[].state]'
# Expected: all registered workers show "ACTIVE"
```

### Criterion 17: Delivery Guarantee — Crash Recovery Delivers Missed Intents

Start dispatch with worker engram-agent. Post intent X. Before engram-agent resumes: kill the dispatch process. Restart dispatch. Intent X is delivered to engram-agent.

```bash
engram dispatch start --agent engram-agent
engram chat post --from test --to engram-agent --thread test --type intent \
  --text "Situation: crash test. Behavior: respond ACK."
kill <dispatch-pid>
engram dispatch start --agent engram-agent   # restart
sleep 2
# Expected: engram-agent resumes and processes intent X; no intent lost
```

### Criterion 18: Delivery Guarantee — Hold-Then-Release Crash-Safe

Place hold on engram-agent. Post 20 intents to engram-agent (exceeds deferred queue of 100; well within limit). Kill and restart dispatch while held. Release hold. Assert all 20 intents delivered in FIFO order. No intents dropped.

```bash
HOLD_ID=$(engram hold acquire --target engram-agent --condition test)
for i in $(seq 1 20); do
  engram chat post --from test --to engram-agent --thread test --type intent \
    --text "Situation: hold test $i. Behavior: ACK."
done
kill <dispatch-pid>
engram dispatch start --agent engram-agent   # restart with engram-agent still held
engram hold release "$HOLD_ID"
# Expected: engram-agent receives and processes all 20 intents in order
```

### Criterion 19: Delivery Guarantee — LEARNED_MESSAGES in Resume Prompt

Post `type=learned` addressed to engram-agent. Then post an intent also addressed to engram-agent. On engram-agent's resume for the intent, the resume prompt includes `LEARNED_MESSAGES:` with the learned text. After DONE:, verify memory file was updated with the extracted fact.

```bash
engram chat post --from executor --to engram-agent --thread test --type learned \
  --text "engram → uses → targ for all build/test/check operations."
engram chat post --from test --to engram-agent --thread test --type intent \
  --text "Situation: test. Behavior: respond ACK."
# Expected: resume prompt includes LEARNED_MESSAGES: engram → uses → targ for ...
# Expected: memory file updated with extracted fact after DONE:
```

---

### Criterion 20: Zero Bash in All Skills

```bash
for skill in use-engram-chat-as engram-tmux-lead engram-agent engram-down engram-up; do
  path=$(find ~/.claude/plugins -name "${skill}.md" | head -1)
  count=$(grep -cE '^\$|&&|\|\||`engram |`targ |#!/' "$path" 2>/dev/null || echo 0)
  echo "$skill: $count bash lines"
done
# Expected: all 0
```

### Criterion 21: Agent Lifecycle Section Deleted from use-engram-chat-as

```bash
grep -c "Agent Lifecycle" ~/.claude/plugins/*/skills/use-engram-chat-as.md
# Expected: 0
```

### Criterion 22: Compaction Recovery Cursor Section Deleted from use-engram-chat-as

```bash
grep -c "Compaction Recovery" ~/.claude/plugins/*/skills/use-engram-chat-as.md
# Expected: 0 (or 1 for the ~5 line lead-only recovery note in engram-tmux-lead)
```

### Criterion 23: Skill Line Count Targets

```bash
wc -l \
  ~/.claude/plugins/*/skills/use-engram-chat-as.md \
  ~/.claude/plugins/*/skills/engram-tmux-lead.md \
  ~/.claude/plugins/*/skills/engram-agent.md \
  ~/.claude/plugins/*/skills/engram-down.md
# Expected: ≤50, ≤110, ≤80, ≤55
```

### Criterion 24: Nine or Fewer Binary Command References Across All Skills

Binary commands referenced in skills must be limited to: `engram dispatch start`, `engram dispatch assign`, `engram dispatch drain`, `engram dispatch stop`, `engram dispatch status`, `engram agent spawn`, `engram hold acquire`, `engram hold release`, `engram hold check`. No other `engram *` commands appear in skill text. (`hold acquire/release/check` are three separate subcommands, counted individually — total ceiling is 9.)

### Criterion 25: Skill Quality Gates (manual behavioral tests)

Run these manually after each skill rewrite. Record pass/fail in the task checklist.

| Input | Expected lead routing |
|-------|----------------------|
| "implement OAuth for the login flow" | executor (not planner) |
| "what approach should we take for the auth architecture?" | planner (not executor) |
| "review this PR" | reviewer |
| "run the tests and fix failures" | executor with TDD skill |
| Two agents disagreeing after 3 argument turns | escalation surfaced to user |

---

## Codesign Decisions

These decisions are locked. Revisiting requires a new codesign session.

| Decision | Resolution |
|----------|-----------|
| Dispatch vs skill event loop | Binary owns all mechanics. Skills own routing classification, prompt composition, hold pattern selection. |
| FROM-filter for self-resume | `dispatchLoop` filters intents where `msg.From == targetWorkerName`. Prevents engram-agent from triggering its own auto-resume. |
| Concurrency enforcement | Binary semaphore (`make(chan struct{}, maxConcurrent)`), not skill compliance. Skill contains no mention of the number 3. |
| Online detection in dispatch | State file (running process), NOT message timestamps. Stateless workers appear online by timestamp after DONE: — only session registry is authoritative. |
| `engram agent run` preserved | Single-agent outer loop kept for backward compatibility and standalone use. Channel-based path is additive in `runConversationLoopWith`. |
| Config file for dispatch | `~/.local/share/engram/dispatch/<slug>.toml` holds fixed persistent workers (engram-agent). Task executors are spawned ad-hoc via `engram dispatch assign`; dispatch picks them up after spawn. |
| Subprocess model (not panes) | `engram dispatch start` spawns agents as subprocesses (exec.Cmd), not tmux panes. Tmux is optional convenience — if available, lead may open a chat-tail monitoring pane. Delivery guarantees do not depend on tmux. |
| Observability | dispatchLoop posts `type=info, thread=dispatch` for every routing event (route, resume, silent, queue, crash). |
| Agent naming in lead skill | Lead skill routing table includes naming guidance with examples. Binary rejects (with warning, not error) generic sequential names matching `/^(worker|agent|executor|planner)[-_]\d+$/`. |
| Skill binary command references | ≤7 total across all skills: `dispatch start`, `dispatch assign`, `dispatch drain`, `dispatch stop`, `dispatch status`, `agent spawn`, `hold acquire/release/check`. |
| Argument relay under dispatch | Binary delivers argument turns as continued `--resume` conversation turns (same session), not as new agent runs. Argument state (count, thread, partner) tracked in state file per Phase 5 schema. |
| Hold blocks auto-resume | `dispatchLoop` checks hold status (state file) before resuming. Held workers have messages routed to per-worker in-memory `deferredQueue` (cap 100). Release drains `deferredQueue` immediately. |
| `engram dispatch drain` | Proper drain/wait replaces `sleep N` guesses in engram-down. Outputs JSON summary for lead to read and compose session report. |
| Delivery type whitelist | `dispatchLoop` routes only actionable types: `intent`, `wait`, `shutdown`. All other types (ack, info, learned, done, ready) are not routed — agents don't need to resume for them. |
| to=all routing | `resolveRecipients` expands `"all"` to all registered worker names in `intentChans`. The literal string `"all"` is never looked up directly as a worker name. |
| Per-worker last_delivered_cursor | State file gains `last_delivered_cursor` per worker. Advanced only on confirmed delivery to worker channel. On restart, dispatchLoop reads from min(all workers' cursors). Per-worker filter skips already-delivered messages on replay. |
| Per-worker deferredQueue | In-memory `deferredQueues map[string][]chat.Message` (cap 100 per worker). Used when worker is ACTIVE (can't inject into running session) or HELD. Drained on ACTIVE→SILENT transition or hold release. Crash recovery handled by `last_delivered_cursor` (not deferredQueue — in-memory queue is not durable). |
| RESUME_REASON field | `buildResumePrompt` includes `RESUME_REASON: intent\|wait\|shutdown`. When reason=wait: also includes `WAIT_FROM:`, `WAIT_TEXT:`, `ARGUMENT_TURN:` fields. |
| LEARNED_MESSAGES field | `buildResumePrompt` calls `collectLearned(chatFilePath, worker, lastDeliveredCursor, readFile)` and includes `LEARNED_MESSAGES: <text1>\|<text2>` on every resume. Learned messages are NOT in the delivery whitelist (no eager resume triggered). Facts extracted lazily per intent cycle. Known limitation: dirty kill (SIGKILL, no restart/shutdown) loses unprocessed learned messages. |

---

## File Structure

New file:
```
internal/cli/cli_dispatch.go    # runDispatch, dispatchLoop, parseDispatch* flags
```

Modified files:
```
internal/cli/cli_agent.go       # runConversationLoopWith: add intents <-chan chat.Message path
                                # buildResumePrompt: add AGENT_NAME field
internal/cli/cli.go             # Register dispatch subcommands (start/assign/drain/stop/status)
```

Skill files (rewrites — all in plugin cache):
```
skills/use-engram-chat-as.md    # ~760 → ~45 lines
skills/engram-tmux-lead.md      # ~1,225 → ~100 lines
skills/engram-agent.md          # ~370 → ~72 lines
skills/engram-down.md           # ~110 → ~45 lines
skills/engram-up.md             # ~13 → ~20 lines (Phase 6 startup guidance added)
```

---

## Task 0: Pre-flight verification

- [ ] **Step 1: Verify Phase 5 criteria still pass**

  Run Phase 5 E2E scenarios manually. All 10 criteria must pass.

  ```bash
  engram agent spawn --name engram-agent --prompt "Load engram:engram-agent skill. Watch for intents."
  engram chat post --from test --to engram-agent --thread test --type intent \
    --text "Situation: test. Behavior: respond ACK."
  # Verify: new session fires; DONE: appears; state returns SILENT
  ```

  - [ ] Criterion 1 (auto-resume) passes
  - [ ] Criterion 2 (STARTING→ACTIVE) passes
  - [ ] Criterion 3 (max-3 concurrent) passes
  - [ ] Criterion 4 (resume prompt fields) passes
  - [ ] Criterion 5 (no background monitor) passes

- [ ] **Step 2: `targ check-full` passes clean**

  ```bash
  targ check-full
  # Expected: zero issues
  ```

---

## Task 1: `runConversationLoopWith` — channel-based intent delivery path

Add `intents <-chan chat.Message` as an alternative to `watchForIntentFunc`. When the channel is non-nil, the outer loop reads from it instead of calling `watchForIntent`. Preserve the existing function signature for `engram agent run` (single-agent standalone) compatibility.

- [ ] **Step 1: Write failing tests**

  In `internal/cli/cli_agent_test.go`, add `TestRunConversationLoopWithChannel`:
  - Construct a buffered `intents` channel
  - Send one intent message to the channel
  - Verify worker session fires and produces expected output
  - Verify DONE: causes channel-wait (not watchForIntent call)

  ```bash
  targ test
  # Expected: new tests fail (channel path not implemented)
  ```

- [ ] **Step 2: Add channel path to `runConversationLoopWith`**

  In `internal/cli/cli_agent.go`, modify `runConversationLoopWith`:
  ```go
  func runConversationLoopWith(
      ctx context.Context,
      runner claudepkg.Runner,
      flags agentRunFlags,
      chatFilePath, stateFilePath, claudeBinary string,
      stdout io.Writer,
      watchForIntent watchForIntentFunc,   // nil = use intents channel
      intents <-chan chat.Message,          // nil = use watchForIntent
      memFileSelector memFileSelectorFunc,
      promptBuilder promptBuilderFunc,
  ) error {
  ```
  
  In `watchAndResume`, when `intents != nil`, read from channel instead of calling `watchForIntent`:
  ```go
  var intentMsg chat.Message
  var newCursor int
  if intents != nil {
      select {
      case <-ctx.Done():
          return ctx.Err()
      case msg := <-intents:
          intentMsg = msg
          newCursor = cursor // cursor advancing is dispatch's responsibility
      }
  } else {
      var err error
      intentMsg, newCursor, err = watchForIntent(ctx, flags.name, chatFilePath, cursor)
      if err != nil { return err }
  }
  ```

- [ ] **Step 3: Run tests green**

  ```bash
  targ test
  # Expected: all tests pass
  ```

- [ ] **Step 4: `targ check-full`**

  ```bash
  targ check-full
  # Expected: zero issues
  ```

---

## Task 2: `buildResumePrompt` — add AGENT_NAME and RECENT_INTENTS fields

Resume prompt must include `AGENT_NAME: <name>` (agent identity for stateless invocations) and `RECENT_INTENTS: ...` (last 5 intent summaries from chat, required for failure correlation across invocations per spec §7 Phase 5 item 1). Without RECENT_INTENTS, engram-agent cannot correlate repeated failures across stateless sessions.

- [ ] **Step 1: Write failing tests**

  In `TestBuildResumePrompt`, add assertions:
  - Output contains `AGENT_NAME: engram-agent` when agent name is provided
  - Output contains `RECENT_INTENTS:` with up to 5 entries when recentIntentSummaries is non-empty
  - Output contains `RECENT_INTENTS: (none)` when recentIntentSummaries is empty
  - Output contains `RESUME_REASON: intent` for normal intent-triggered resume
  - Output contains `RESUME_REASON: wait`, `WAIT_FROM:`, `WAIT_TEXT:`, `ARGUMENT_TURN:` when resumeReason is "wait"
  - Output contains `LEARNED_MESSAGES: (none)` when learnedMessages is empty
  - Output contains `LEARNED_MESSAGES:` with entries when learnedMessages is non-empty

  In `TestSelectRecentIntents`, add:
  - Given chat file with 7 intent messages, returns exactly 5 most recent
  - Returns empty slice when no intent-type messages exist
  - Truncates individual summaries at 80 chars

  In `TestCollectLearned`, add:
  - Given chat file with 3 learned messages since cursor, returns all 3 texts
  - Learned messages addressed to "all" are included for any agent
  - Learned messages before lastDeliveredCursor are excluded
  - Returns empty slice when no learned messages exist since cursor

  ```bash
  targ test
  # Expected: new tests fail (fields not present)
  ```

- [ ] **Step 2: Implement `selectRecentIntents`**

  New function in `internal/cli/cli_agent.go`:
  ```go
  // selectRecentIntents reads the last maxIntents intent-type messages from the chat file
  // and returns them as summary strings: "from→to: text (truncated at 80 chars)"
  func selectRecentIntents(chatFilePath string, readFile func(string) ([]byte, error), maxIntents int) ([]string, error)
  ```

- [ ] **Step 3: Add agentName, recentIntentSummaries, resumeReason, and learnedMessages to `buildResumePrompt`**

  ```go
  func buildResumePrompt(
      agentName string,
      cursor int,
      memFiles []string,
      intentFrom, intentText string,
      recentIntents []string,
      resumeReason string,        // "intent" | "wait" | "shutdown"
      waitFrom, waitText string,  // populated when resumeReason == "wait"
      argumentTurn int,           // populated when resumeReason == "wait"
      learnedMessages []string,   // entries from collectLearned() since last delivery
  ) string {
      // ...existing fields...
      // Add: "AGENT_NAME: " + agentName + "\n"
      // Add: "RECENT_INTENTS: " + formatIntentSummaries(recentIntents) + "\n"
      // Add: "RESUME_REASON: " + resumeReason + "\n"
      // If resumeReason == "wait":
      //   "WAIT_FROM: " + waitFrom + "\n"
      //   "WAIT_TEXT: " + waitText + "\n"
      //   "ARGUMENT_TURN: " + strconv.Itoa(argumentTurn) + "\n"
      // Add: "LEARNED_MESSAGES: " + formatLearnedMessages(learnedMessages) + "\n"
  }
  ```

  New helper in `internal/cli/cli_agent.go`:
  ```go
  // collectLearned reads learned-type messages addressed to agentName (or "all")
  // from the chat file since lastDeliveredCursor, returning their text fields.
  func collectLearned(chatFilePath string, agentName string, lastDeliveredCursor int, readFile func(string) ([]byte, error)) ([]string, error)
  ```

  Update all callers. In `dispatchLoop`/`watchAndResume`, populate all new fields before each resume.

- [ ] **Step 4: Run tests green**

  ```bash
  targ test && targ check-full
  ```

---

## Task 3: `internal/cli/cli_dispatch.go` — dispatchLoop + runDispatch

New file. Core of Phase 6.

- [ ] **Step 1: Write failing tests**

  Create `internal/cli/cli_dispatch_test.go`:

  - `TestDispatchLoopRoutes`: verify intent addressed to worker A is sent to A's channel, not B's
  - `TestDispatchLoopFromFilter`: verify intent FROM worker A to worker A is NOT sent to A's channel
  - `TestDispatchLoopHeldWorker`: verify intent for held worker goes to deferredQueue, not channel; drain on release
  - `TestDispatchLoopActiveWorker`: verify intent for ACTIVE worker goes to deferredQueue; drain on SILENT transition
  - `TestDispatchLoopDeferredQueueCap`: verify 101st message for held worker triggers overflow info message, oldest dropped
  - `TestDispatchLoopWaitDelivery`: verify type=wait message addressed to worker A is routed to A's channel
  - `TestDispatchLoopShutdownDelivery`: verify type=shutdown message addressed to worker A is routed to A's channel
  - `TestDispatchLoopToAllExpansion`: verify to=all routes to ALL registered workers, not just one
  - `TestDispatchLoopLearnedNotRouted`: verify type=learned message is NOT routed (not in whitelist)
  - `TestRunDispatchSemaphore`: verify 4th concurrent worker blocks until slot opens
  - `TestDispatchObservabilityMessages`: verify info messages posted to chat for route/resume/silent/crash/overflow events
  - `TestDispatchLastDeliveredCursorAdvances`: verify state file last_delivered_cursor advances after confirmed delivery
  - `TestDispatchCrashRecovery`: verify restart from last_delivered_cursor delivers missed messages

  ```bash
  targ test
  # Expected: new tests fail (file not implemented)
  ```

- [ ] **Step 2: Implement `dispatchLoop`**

  ```go
  // dispatchLoop watches all chat messages and routes actionable types to worker channels.
  // It runs until ctx is cancelled.
  func dispatchLoop(
      ctx context.Context,
      workerChans map[string]chan chat.Message,
      deferredQueues map[string][]chat.Message,   // in-memory, cap 100 per worker
      holdChecker holdCheckerFunc,
      poster *chat.FilePoster,
      stateFilePath, chatFilePath string,
      cursor int,
  ) error {
      watcher := chat.NewWatcher(chatFilePath)
      for {
          msg, newCursor, err := watcher.Next(ctx, cursor)
          if err != nil { return err }
          cursor = newCursor

          // Route only actionable types that require the recipient to take action.
          // intent: task assignment; wait: argument challenge; shutdown: lifecycle signal.
          if msg.Type != "intent" && msg.Type != "wait" && msg.Type != "shutdown" { continue }

          for _, recipient := range resolveRecipients(msg.To, workerChans) {
              ch, ok := workerChans[recipient]
              if !ok { continue }
              // FROM-filter: do not route messages back to sender (prevents self-resume loop).
              if msg.From == recipient { continue }
              // Hold check: defer if held; do NOT drop.
              if holdChecker != nil && holdChecker(recipient) {
                  if len(deferredQueues[recipient]) < 100 {
                      deferredQueues[recipient] = append(deferredQueues[recipient], msg)
                  } else {
                      postQueueOverflow(poster, recipient)
                  }
                  continue
              }
              // ACTIVE check: defer if worker is currently in a session.
              if isWorkerActive(stateFilePath, recipient) {
                  if len(deferredQueues[recipient]) < 100 {
                      deferredQueues[recipient] = append(deferredQueues[recipient], msg)
                  } else {
                      postQueueOverflow(poster, recipient)
                  }
                  continue
              }
              // Deliver: send to channel and advance per-worker last_delivered_cursor.
              ch <- msg
              updateLastDeliveredCursor(stateFilePath, recipient, cursor)
              postRoutingInfo(poster, recipient, msg, cursor)
          }
      }
  }

  // resolveRecipients expands "all" to all registered worker names.
  func resolveRecipients(toField string, workers map[string]chan chat.Message) []string {
      if toField == "all" {
          names := make([]string, 0, len(workers))
          for name := range workers { names = append(names, name) }
          return names
      }
      return parseRecipients(toField)
  }
  ```

  On hold release: drain `deferredQueues[worker]` by sending each queued message to the worker's channel and advancing `last_delivered_cursor`.

  On ACTIVE→SILENT transition: drain `deferredQueues[worker]` the same way.

  State file schema addition per worker:
  ```toml
  [workers.engram-agent]
  last_delivered_cursor = 12345
  ```

- [ ] **Step 3: Implement `runDispatch`**

  ```go
  type workerConfig struct {
      name   string
      prompt string
  }
  
  func runDispatch(
      ctx context.Context,
      workers []workerConfig,
      maxConcurrent int,
      chatFilePath, stateFilePath, claudeBinary string,
      stdout io.Writer,
  ) error {
      sem := make(chan struct{}, maxConcurrent)
      intentChans := make(map[string]chan chat.Message, len(workers))
      for _, w := range workers {
          intentChans[w.name] = make(chan chat.Message, 16)
      }
      
      cursor, err := chatFileCursor(chatFilePath, os.ReadFile)
      if err != nil { return err }
      
      var wg sync.WaitGroup
      for _, w := range workers {
          wg.Add(1)
          go func(cfg workerConfig) {
              defer wg.Done()
              sem <- struct{}{}
              defer func() { <-sem }()
              // run outer loop, channel-based
              _ = runConversationLoopWith(ctx, buildAgentRunner(...), /* ... */, nil, intentChans[cfg.name], ...)
          }(w)
      }
      
      go func() {
          _ = dispatchLoop(ctx, intentChans, nil, nil, chatFilePath, cursor)
      }()
      
      wg.Wait()
      return nil
  }
  ```

- [ ] **Step 4: Run tests green**

  ```bash
  targ test && targ check-full
  ```

---

## Task 4: `engram dispatch` CLI subcommands

Register `dispatch start`, `dispatch assign`, `dispatch drain`, `dispatch stop`, `dispatch status` in `internal/cli/cli.go`.

- [ ] **Step 1: Write failing tests**

  In `internal/cli/cli_dispatch_test.go`, add integration tests:
  - `TestDispatchStartOutputsPaneIDs`: verify start outputs pane IDs
  - `TestDispatchAssignPostsIntent`: verify assign posts to chat with correct from/type
  - `TestDispatchStatusOutputsJSON`: verify status outputs valid JSON with required fields
  - `TestDispatchDrainOutputsJSON`: verify drain outputs JSON summary

  ```bash
  targ test
  # Expected: new tests fail
  ```

- [ ] **Step 2: Implement `parseDispatchFlags` + subcommand handlers**

  ```go
  // engram dispatch start [--agent name]... [--max-concurrent N] [--config path]
  func runDispatchStart(args []string, stdout io.Writer) error
  
  // engram dispatch assign --agent name --task text
  func runDispatchAssign(args []string, stdout io.Writer) error
  
  // engram dispatch drain [--timeout seconds]
  func runDispatchDrain(args []string, stdout io.Writer) error
  
  // engram dispatch stop
  func runDispatchStop(args []string, stdout io.Writer) error
  
  // engram dispatch status
  func runDispatchStatus(args []string, stdout io.Writer) error
  ```

- [ ] **Step 3: Register in dispatch router**

  ```go
  func runDispatchDispatch(subArgs []string, stdout io.Writer) error {
      switch subArgs[0] {
      case "start":   return runDispatchStart(subArgs[1:], stdout)
      case "assign":  return runDispatchAssign(subArgs[1:], stdout)
      case "drain":   return runDispatchDrain(subArgs[1:], stdout)
      case "stop":    return runDispatchStop(subArgs[1:], stdout)
      case "status":  return runDispatchStatus(subArgs[1:], stdout)
      default:        return fmt.Errorf("unknown dispatch subcommand: %s", subArgs[0])
      }
  }
  ```

  Register in `internal/cli/cli.go` alongside existing `chat`, `hold`, `agent` routers.

- [ ] **Step 4: Run tests green**

  ```bash
  targ test && targ check-full
  ```

---

## Task 5: E2E verification (binary criteria 1–19)

Verify all binary criteria before any skill rewrites.

- [ ] **Criterion 1: Dispatch Start Spawns Workers** — see E2E section above
- [ ] **Criterion 2: Intent Routing** — post intent, verify auto-resume fires
- [ ] **Criterion 3: FROM-Filter** — post self-addressed intent, verify no loop
- [ ] **Criterion 4: Semaphore Limit** — saturate concurrency, verify queue
- [ ] **Criterion 5: Dispatch Assign** — verify intent posted + delivered
- [ ] **Criterion 6: Observability Info Messages** — verify all routing events in chat
- [ ] **Criterion 7: Drain** — post intent, drain, verify JSON summary
- [ ] **Criterion 8: Stop** — verify shutdown messages + done messages in chat
- [ ] **Criterion 9: Hold Blocks** — acquire hold, post intent, verify queued in deferredQueue; verify drain on release
- [ ] **Criterion 10: Status Observable** — verify JSON output with queue depth
- [ ] **Criterion 11: Crash Cleanup** — kill worker, verify DEAD state + chat info
- [ ] **Criterion 12: Recovery on Start** — simulate stale state, verify clean start
- [ ] **Criterion 13: Resume Prompt Includes Agent Name and RESUME_REASON** — inspect prompt fields
- [ ] **Criterion 14: WAIT Delivery to SILENT Worker** — see E2E section above
- [ ] **Criterion 15: Shutdown Delivery to SILENT Worker** — see E2E section above
- [ ] **Criterion 16: to=all Broadcast Delivery** — see E2E section above
- [ ] **Criterion 17: Crash Recovery Delivers Missed Intents** — see E2E section above
- [ ] **Criterion 18: Hold-Then-Release Crash-Safe** — see E2E section above
- [ ] **Criterion 19: LEARNED_MESSAGES in Resume Prompt** — see E2E section above

  ```bash
  targ check-full
  # Expected: zero issues
  ```

---

## Task 6: `use-engram-chat-as` Rewrite (~760 → ~45 lines)

- [ ] **Step 1: Invoke writing-skills skill**

  ```
  /writing-skills
  ```

- [ ] **Step 2: Audit deletion targets**

  Sections to delete entirely:
  - Agent Lifecycle (steps 1–11) — binary-owned
  - Compaction Recovery (full section) — binary-owned; ~5 lines move to engram-tmux-lead
  - Chat File Location (path derivation bash) — binary-owned
  - Message Format (TOML syntax) — binary-owned
  - Cursor Tracking (all bash cursor patterns) — binary-owned
  - Online/offline detection specifics — binary-owned
  - Common Mistakes table (all bash-pattern rows) — binary-owned
  - Background Monitor references — deleted in Phase 5
  - ACK-wait mechanics (bash) — binary-owned
  - Hold mechanics (bash) — binary-owned

  Sections to KEEP (irreducible behavioral content):
  - Message type catalog (semantics only — WHEN to use intent vs info vs learned vs done)
  - Prefix marker catalog (INTENT:/ACK:/WAIT:/DONE:/LEARNED:/INFO: + WHEN-to-end-turn rules)
  - Intent protocol behavioral rules (WHO must be in TO, WHEN intent is warranted)
  - Argument protocol (initiator=factual, reactor=aggressive; 3-input cap; escalation trigger; resolution recording)
  - Agent roles (active vs reactive — purely behavioral)
  - Shutdown honor protocol (complete in-flight, post done, exit)

  **NOTE:** Prefix marker catalog MUST include both format and when-to-end-turn rules. "Say INTENT: and end your turn" is behavioral, not delivery mechanics. Without this, agents output INTENT: and continue generating, invalidating ACK-wait.

- [ ] **Step 3: Rewrite — preserve example density**

  Compression risk: examples ARE the behavioral content. For argument reactor behavior, a single rule line "push back hard" is insufficient. At minimum two concrete examples of strong vs weak objection must survive. Apply this test to every compressed section.

- [ ] **Step 4: Run writing-skills TDD cycle**

  Baseline behavioral test (RED):
  - Agent reads current skill text, given scenario: "binary about to write to shared file; you know a conflict exists." Does agent post WAIT: with aggressive pushback? Record current behavior.

  Update skill (GREEN):
  - Apply deletions and rewrites.

  Verify behavioral change:
  - Same scenario. Does agent still post WAIT: with aggressive pushback? Does agent still include engram-agent in TO?
  - New: does agent post INTENT: and END the turn (not continue generating)?

- [ ] **Step 5: Commit**

---

## Task 7: `engram-tmux-lead` Rewrite (~1,225 → ~100 lines)

- [ ] **Step 1: Invoke writing-skills skill**

- [ ] **Step 2: Audit deletion targets**

  Sections to delete:
  - SPAWN-PANE bash definition — binary-owned (`engram agent spawn`)
  - Column split logic — binary-owned
  - Heartbeat bash loop — binary-owned
  - Watch loop (`while true; do`) patterns — binary-owned (`dispatchLoop`)
  - Manual `engram agent resume` calls in loops — binary-owned (`engram dispatch assign`)
  - Hold bash (hold acquire/release/list invocations as bash loops) — reduce to 7 command references
  - Concurrency tracking bash — binary-owned (semaphore)
  - Pane registry tracking — binary-owned (state file)

  Sections to KEEP (irreducible judgment content):
  - Routing decision table (classify request as quick-fix/scoped/feature) — LLM judgment
  - Spawn prompt template (role + initial context, ~10 lines)
  - Assign task template (task description + hold expectations, ~10 lines)
  - Hold pattern selection (pair/handoff/fan-in/barrier — WHEN each fits, ~10 lines)
  - Escalation surfacing template — user-visible output, must survive with concrete example
  - Context pressure self-management (~10 lines including `engram dispatch status` call)
  - Agent naming guidance with examples (~3 lines, user-visible: pane names in tmux)
  - Startup sequence (~10 lines: dispatch start, open chat tail pane, post ready)
  - Compaction recovery (~5 lines: `engram agent list + engram hold list`)
  - Shutdown sequence (~5 lines: dispatch drain, dispatch stop)

- [ ] **Step 3: Rewrite — preserve prompt template specificity**

  Spawn and assign prompt templates must include enough structure to produce task-appropriate worker prompts. "Write a prompt for the executor" is not actionable. Each template must specify: role identity, applicable skills, behavioral expectations (TDD, DONE: marker, worktree path if applicable).

- [ ] **Step 4: Skill Quality Gates (Criterion 19)**

  Run all 5 manual behavioral tests. Record results in checklist:
  - [ ] "implement OAuth" → routes to executor
  - [ ] "why is auth failing?" → routes to planner
  - [ ] "review this PR" → routes to reviewer
  - [ ] "run tests and fix failures" → executor with TDD skill
  - [ ] 3-turn argument → escalation surfaced to user

- [ ] **Step 5: Run writing-skills TDD cycle and commit**

---

## Task 8: `engram-agent` Rewrite (~370 → ~72 lines)

- [ ] **Step 1: Invoke writing-skills skill**

- [ ] **Step 2: Audit deletion targets**

  Sections to delete:
  - Watch loop, background monitor — deleted in Phase 5
  - Cursor tracking bash — binary-owned
  - Chat write bash (`engram chat post`) — binary-owned (speech-to-chat)
  - All I/O mechanics

  Sections to KEEP (ordered init checklist form — resists compression better than prose):
  1. Startup: read AGENT_NAME, CURSOR, MEMORY_FILES, INTENT_FROM, INTENT_TEXT, RECENT_INTENTS from resume prompt
  2. Tiered memory load: core memories always, recents on startup (list files from MEMORY_FILES)
  3. Rate limit check: if >5 new memories in last 10 min (check file mtimes), pause extraction
  4. Last 5 intent summaries: read from resume prompt for failure correlation
  5. Situation matching guidance (semantic similarity is LLM judgment)
  6. Argument reactor behavior (aggressive — minimum 2 concrete examples of strong objection)
  7. Feedback learning signal detection (user correction vs noise — examples)
  8. Fact extraction patterns (S-P-O triples — what makes a reusable fact)
  9. Conflict resolution judgment (when new memory overrides existing)
  10. Memory file locking and atomic write (survives per SPEECH-5)

  **NOTE:** Startup sequence (items 1–4) MUST be in ordered numbered form. Cold-start invocations must execute these steps in order; prose paragraphs obscure ordering. Each step should be ≤2 lines.

- [ ] **Step 3: Verify ≤80 lines**

  Line count check after rewrite. If >80 lines, identify candidates for further compression. Do not drop examples that are necessary for behavioral correctness.

- [ ] **Step 4: Run writing-skills TDD cycle and commit**

---

## Task 9: `engram-down` and `engram-up` Rewrites

### engram-down (~110 → ~45 lines)

- [ ] **Step 1: Invoke writing-skills skill**

- [ ] **Step 2: Delete bash sections** (kill-pane, hold release bash, wait loops)

- [ ] **Step 3: Rewrite shutdown sequence**

  Surviving content:
  - `engram dispatch drain --timeout 30` — wait for in-flight work
  - `engram dispatch stop` — send shutdown
  - Memory preservation check: scan chat for LEARNED messages since session start BEFORE posting summary (race risk if skipped)
  - Session summary format: decisions made, facts learned, open questions, tasks completed vs in-flight

- [ ] **Step 4: Run writing-skills TDD cycle and commit**

### engram-up (~13 → ~20 lines)

- [ ] **Step 1: Update startup sequence** for Phase 6

  Replace old spawn-based startup with:
  ```
  1. Run: engram dispatch start [--agent engram-agent] [--agent ...]
  2. Open chat tail pane (observability): tmux split-window -h 'tail -f <chat-file>'
  3. Post ready to user
  ```

  Note: The skill should say "open a monitoring pane watching the chat file" without specifying the exact tmux command — the lead can use whatever method they have available. This avoids tension with Criterion 14 (zero bash) while preserving the behavioral requirement.

- [ ] **Step 2: Commit**

---

## Task 10: Full E2E Verification (Criteria 20–25 + regression guard)

- [ ] **Step 1: Run all skill criteria (20–24)**

  ```bash
  # Criterion 20: zero bash
  for skill in use-engram-chat-as engram-tmux-lead engram-agent engram-down engram-up; do
    path=$(find ~/.claude/plugins -name "${skill}.md" | head -1)
    echo "$skill: $(grep -cE '^\$|&&|\|\||`engram |`targ |#!/' "$path" 2>/dev/null || echo 0) bash lines"
  done
  
  # Criterion 21: Agent Lifecycle deleted
  grep -c "Agent Lifecycle" ~/.claude/plugins/*/skills/use-engram-chat-as.md
  
  # Criterion 22: Compaction Recovery deleted from use-engram-chat-as
  grep -c "Compaction Recovery" ~/.claude/plugins/*/skills/use-engram-chat-as.md
  
  # Criterion 23: Line counts
  wc -l ~/.claude/plugins/*/skills/use-engram-chat-as.md \
         ~/.claude/plugins/*/skills/engram-tmux-lead.md \
         ~/.claude/plugins/*/skills/engram-agent.md \
         ~/.claude/plugins/*/skills/engram-down.md
  ```

- [ ] **Step 2: Skill Quality Gates (Criterion 25)** — run 5 manual routing tests (see Task 7)

- [ ] **Step 3: Regression guard — Phase 5 criteria still pass**

  ```bash
  engram agent spawn --name engram-agent --prompt "..."
  # Post intent, verify auto-resume fires
  # Verify session-id changes between invocations (Criterion 7 from Phase 5)
  ```

- [ ] **Step 4: `targ check-full` clean**

  ```bash
  targ check-full
  # Expected: zero issues
  ```

---

## Known Limitations

**Memory file write race (RISK-E2E-3):** Up to 3 concurrent engram-agent invocations may write memory files simultaneously. Phase 6 preserves the atomic write with lock timeout (surviving per SPEECH-5 in engram-agent skill). Stale lock detection (lock file left by crashed process) is NOT implemented in Phase 6. File: `internal/memory/readmodifywrite.go`. Future work: add mtime-based stale lock timeout.

**Crash recovery scope:** `engram dispatch start` detects stale STARTING/ACTIVE entries and marks them DEAD. It re-reads the chat file from each worker's `last_delivered_cursor` on restart, delivering messages that arrived while dispatch was down. However, messages in `deferredQueues` at crash time (held or in-flight for ACTIVE workers) are lost — the in-memory queue is not durable. Future work: persist deferredQueue to state file.

**WAIT delivery to ACTIVE worker:** If a worker is currently in an active `claude -p` session and a WAIT arrives for it, the WAIT is placed in `deferredQueue` and delivered on the next resume cycle (after the current session posts `DONE:`). The worker sees the WAIT but not instantaneously. This is architecturally unavoidable: the binary cannot inject messages into a running process.

**Dirty kill loses learned messages:** If the binary receives SIGKILL with no subsequent restart or shutdown signal, any `learned` messages posted after the last `intent` delivery to `engram-agent` will not be processed (the shutdown resume that calls `collectLearned` never runs). Clean shutdown (`engram dispatch stop`) always processes pending learned messages.

**Argument session context growth (RISK-E2E-4):** Each argument turn appends to the session context via `claude -p --resume`. For argument sessions exceeding 3 turns with large resume prompts (many MEMORY_FILES + RECENT_INTENTS), the Claude context window may fill. Phase 6 does not implement fresh-session restart with condensed history. Future work: add context-size check in `runConversationLoopWith` before each resume call, triggering a fresh session with condensed argument history when approaching limit.

**`engram dispatch` not a daemon:** Phase 6 `dispatch` runs in the foreground of the lead pane. If the lead's terminal session closes, dispatch exits. Future work: daemonize with `engram dispatch start --daemon`. Not in scope for Phase 6.

---

## Pre-Flight Checklist (run before declaring Phase 6 complete)

- [ ] `engram dispatch start --agent engram-agent` outputs worker name and returns (subprocess started)
- [ ] `engram dispatch status` outputs valid JSON with `workers` and `queue_depth` fields
- [ ] Post one intent → binary resumes engram-agent; routing info message appears in chat
- [ ] `engram dispatch drain --timeout 30` returns JSON summary `{"completed":1,"failed":0,"queued":0}`
- [ ] `engram dispatch stop` exits 0; shutdown+done messages appear in chat
- [ ] Post self-addressed intent from engram-agent; verify no loop (no second session starts)
- [ ] All 5 skill quality gate tests pass (routing classification correct)
- [ ] `wc -l` on all four rewritten skills within targets (≤50/≤110/≤80/≤55)
- [ ] `grep -cE '^\$|&&|`engram '` on all skills returns 0 (zero bash)
- [ ] `targ check-full` passes clean
