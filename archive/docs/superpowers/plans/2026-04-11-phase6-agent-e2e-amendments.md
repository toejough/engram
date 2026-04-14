# Phase 6 Agent E2E Amendments

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Patch six agent-perspective gaps in the Phase 6 plan (`2026-04-11-phase6-engram-dispatch.md`) — undocumented INTENT: TO defaults, missing RESUME_REASON handling in engram-agent skill, STARTING-state buffering behavior, hold-release IPC via chat, ACTIVE vs SILENT argument relay distinction, and crash-during-argument known limitation.

**Architecture:** Each amendment is a targeted addition to an existing task in the main Phase 6 plan. Amendments are additive — they add new test steps, implementation notes, or checklist items. No task structure is removed or reordered. Amendments are numbered A1–A6 corresponding to the six gaps.

**Tech Stack:** Go, `internal/cli`, `internal/chat`, `engram-agent` skill. No new packages.

---

## Pre-Flight

Apply amendments in order. Each is independent unless stated. Run `targ test && targ check-full` after each amendment.

---

## Amendment A1: INTENT: marker TO field default in speech relay

**Gap:** Speech relay has no documented default `to` field for worker INTENT: markers. Using `to=all` would flood all registered workers with every intent.

**Refinement accepted from phase6-skill:** This change must also be reflected in Task 6 (use-engram-chat-as) prefix marker catalog — add TO: subfield syntax to the INTENT: entry.

**Files:**
- Modify: `internal/cli/cli_agent.go` (speech relay / processStream or equivalent)
- Modify: `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md` (Task 3 notes, Task 6 Step 2 keep list)

**Codesign decision to add (to main plan):**

> **INTENT: TO default**: Speech relay defaults `to=engram-agent` for all worker INTENT: markers. Workers may include an optional `TO:` subfield: `INTENT: TO: lead, engram-agent. Situation: X. Behavior: Y.` If `TO:` subfield is absent, relay posts with `to=engram-agent`. This satisfies the HARD RULE (engram-agent always in TO) without flooding all workers.

- [ ] **Step 1: Write failing test for TO: subfield parsing**

  In `internal/cli/cli_agent_test.go` (or wherever speech marker parsing is tested):

  ```go
  func TestParseSpeechMarkerIntentTOSubfield(t *testing.T) {
      t.Parallel()
      tests := []struct {
          name         string
          markerText   string
          expectedTo   string
      }{
          {
              name:       "no TO subfield defaults to engram-agent",
              markerText: "Situation: about to act. Behavior: do X.",
              expectedTo: "engram-agent",
          },
          {
              name:       "TO subfield overrides default",
              markerText: "TO: lead, engram-agent. Situation: about to act. Behavior: do X.",
              expectedTo: "lead, engram-agent",
          },
          {
              name:       "TO subfield with single recipient",
              markerText: "TO: engram-agent. Situation: test. Behavior: ACK.",
              expectedTo: "engram-agent",
          },
      }
      for _, tc := range tests {
          tc := tc
          t.Run(tc.name, func(t *testing.T) {
              t.Parallel()
              g := gomega.NewWithT(t)
              to := parseIntentMarkerTO(tc.markerText)
              g.Expect(to).To(gomega.Equal(tc.expectedTo))
          })
      }
  }
  ```

  ```bash
  targ test
  # Expected: FAIL — parseIntentMarkerTO not defined
  ```

- [ ] **Step 2: Implement `parseIntentMarkerTO`**

  In `internal/cli/cli_agent.go` (or the file containing speech marker parsing):

  ```go
  // parseIntentMarkerTO extracts the TO: subfield from an INTENT: marker text.
  // Returns the trimmed TO value, or "engram-agent" if no TO: subfield is present.
  // Format: "TO: recipient1, recipient2. Situation: ..."
  func parseIntentMarkerTO(markerText string) string {
      const toPrefix = "TO: "
      if !strings.HasPrefix(markerText, toPrefix) {
          return "engram-agent"
      }
      rest := markerText[len(toPrefix):]
      // TO: value ends at first ". "
      end := strings.Index(rest, ". ")
      if end == -1 {
          return strings.TrimSpace(rest)
      }
      return strings.TrimSpace(rest[:end])
  }
  ```

  Update the speech relay's INTENT: case to call `parseIntentMarkerTO` and use the result as the `to` field when posting to chat.

- [ ] **Step 3: Run tests green**

  ```bash
  targ test
  # Expected: all pass
  ```

- [ ] **Step 4: `targ check-full`**

  ```bash
  targ check-full
  # Expected: zero issues
  ```

- [ ] **Step 5: Add notes to Task 3 and Task 6 in main plan**

  In `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md`, find the `## Task 3` heading and add under Step 2's dispatchLoop implementation note:

  ```
  Note: INTENT: marker TO field — speech relay applies parseIntentMarkerTO: defaults
  to "engram-agent" if no "TO: " subfield found. Workers that need multi-recipient
  intents use "TO: lead, engram-agent. Situation: ..." syntax.
  ```

  In Task 6 Step 2 keep list (prefix marker catalog section), add to the INTENT: entry:

  ```
  - INTENT: prefix marker catalog: add TO: subfield syntax
    Format: INTENT: [TO: name1, name2. ]Situation: X. Behavior: Y.
    If TO: absent, speech relay defaults to engram-agent.
    TO: is always optional — engram-agent is always included regardless.
  ```

- [ ] **Step 6: Commit**

  ```bash
  git add internal/cli/cli_agent.go internal/cli/cli_agent_test.go \
    docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md
  git commit -m "feat(speech-relay): default INTENT: marker to=engram-agent; support TO: subfield"
  ```

---

## Amendment A2: RESUME_REASON handling in engram-agent Task 8

**Gap:** Task 8's 10-item startup checklist omits RESUME_REASON routing. An agent resumed with `RESUME_REASON: wait` has no guidance on what to do vs `RESUME_REASON: shutdown` vs `RESUME_REASON: intent`.

**Refinements accepted from phase6-skill:**
1. engram-agent acts as INITIATOR (factual, not aggressive) when resumed with `RESUME_REASON: wait` — it is defending its own intent against a challenge from WAIT_FROM. Item 0 must make the initiator role explicit.
2. Item 0 must end with: "If intent: proceed to item 1. Process LEARNED_MESSAGES before situation matching (item 1a)."

**Files:**
- Modify: `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md` (Task 8 checklist)
- Modify: `internal/cli/cli_agent_test.go` (new prompt field tests)

- [ ] **Step 1: Write failing tests for RESUME_REASON fields**

  Note: Uses `resumePromptArgs` struct form per ARCH-A4. Update if ARCH-A4 is not yet applied.

  In `internal/cli/cli_agent_test.go`, in the existing `TestBuildResumePrompt` (or add alongside):

  ```go
  func TestBuildResumePromptShutdown(t *testing.T) {
      t.Parallel()
      g := gomega.NewWithT(t)
      prompt := buildResumePrompt(resumePromptArgs{
          AgentName:    "engram-agent",
          Cursor:       100,
          IntentFrom:   "lead",
          IntentText:   "shutdown",
          ResumeReason: "shutdown",
      })
      g.Expect(prompt).To(gomega.ContainSubstring("RESUME_REASON: shutdown"))
      // shutdown prompt must NOT contain wait fields
      g.Expect(prompt).NotTo(gomega.ContainSubstring("WAIT_FROM:"))
  }

  func TestBuildResumePromptWait(t *testing.T) {
      t.Parallel()
      g := gomega.NewWithT(t)
      prompt := buildResumePrompt(resumePromptArgs{
          AgentName:    "worker-a",
          Cursor:       200,
          IntentFrom:   "engram-agent",
          IntentText:   "you stepped on my cursor",
          ResumeReason: "wait",
          WaitFrom:     "engram-agent",
          WaitText:     "you stepped on my cursor",
          ArgumentTurn: 1,
      })
      g.Expect(prompt).To(gomega.ContainSubstring("RESUME_REASON: wait"))
      g.Expect(prompt).To(gomega.ContainSubstring("WAIT_FROM: engram-agent"))
      g.Expect(prompt).To(gomega.ContainSubstring("WAIT_TEXT: you stepped on my cursor"))
      g.Expect(prompt).To(gomega.ContainSubstring("ARGUMENT_TURN: 1"))
  }

  func TestBuildResumePromptWaitTurn3(t *testing.T) {
      t.Parallel()
      g := gomega.NewWithT(t)
      prompt := buildResumePrompt(resumePromptArgs{
          AgentName:    "worker-a",
          Cursor:       200,
          IntentFrom:   "engram-agent",
          IntentText:   "still objecting",
          ResumeReason: "wait",
          WaitFrom:     "engram-agent",
          WaitText:     "still objecting",
          ArgumentTurn: 3,
      })
      g.Expect(prompt).To(gomega.ContainSubstring("ARGUMENT_TURN: 3"))
      // ARGUMENT_TURN: 3 is the escalation threshold; prompt must include escalation hint
      g.Expect(prompt).To(gomega.ContainSubstring("ARGUMENT_ESCALATION_NOTE:"))
  }
  ```

  ```bash
  targ test
  # Expected: FAIL — ESCALATE: guidance not in prompt for turn 3
  ```

- [ ] **Step 2: Add ARGUMENT_TURN=3 escalation hint to `buildResumePrompt`**

  In `internal/cli/cli_agent.go`, in `buildResumePrompt`, after setting `ARGUMENT_TURN:`:

  ```go
  if argumentTurn >= 3 {
      b.WriteString("ARGUMENT_ESCALATION_NOTE: This is argument turn 3. If unresolved, " +
          "say ESCALATE: summarizing both positions for lead mediation.\n")
  }
  ```

  This gives the resumed agent a concrete cue without overriding judgment.

- [ ] **Step 3: Run tests green**

  ```bash
  targ test
  # Expected: all pass
  ```

- [ ] **Step 4: Add RESUME_REASON item 0 to Task 8 checklist in main plan**

  In `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md`, under `## Task 8`, find "Sections to KEEP (ordered init checklist form)" and prepend item 0:

  ```
  0. Read RESUME_REASON from prompt. Branch:
     - shutdown: say DONE: immediately; skip all other steps.
     - wait: you are being resumed to respond to a challenge. Read WAIT_FROM,
       WAIT_TEXT, ARGUMENT_TURN. You are the INITIATOR (your action was
       challenged) — respond factually, not aggressively. Defend or concede
       with ACK:. If ARGUMENT_TURN=3 and no resolution, say ESCALATE: with
       both positions summarized for lead mediation.
     - intent: proceed to item 1. Process LEARNED_MESSAGES before situation
       matching (item 1a).
  ```

- [ ] **Step 5: `targ check-full` and commit**

  ```bash
  targ check-full
  git add internal/cli/cli_agent.go internal/cli/cli_agent_test.go \
    docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md
  git commit -m "feat(resume-prompt): add ARGUMENT_ESCALATION_NOTE at turn 3; document RESUME_REASON routing in Task 8"
  ```

---

## Amendment A3: STARTING state intent buffering — documentation only

**Gap:** dispatchLoop behavior for STARTING workers (between spawn and READY:) is undocumented. The channel is buffered (cap 16), so messages buffer silently — but this is implicit.

**Files:**
- Modify: `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md` (Task 3 Step 2 comment)

No code change needed — buffered channel already handles this correctly.

- [ ] **Step 1: Add comment to Task 3 Step 2 dispatchLoop pseudocode**

  In `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md`, in Task 3 Step 2's dispatchLoop Go snippet, immediately after the `ACTIVE check:` comment block, add:

  ```go
  // STARTING check: workers in STARTING state (spawned but READY: not yet seen)
  // receive messages via channel immediately. Channel is buffered (cap 16) so
  // messages queue until the worker's outer loop begins after READY: fires.
  // No explicit deferral needed — channel backpressure is sufficient.
  ```

- [ ] **Step 2: Add test documenting STARTING buffering behavior**

  In `internal/cli/cli_dispatch_test.go`, add:

  ```go
  func TestDispatchLoopStartingWorkerBuffersIntent(t *testing.T) {
      t.Parallel()
      // STARTING worker: spawned but READY: not yet seen.
      // Intent should be buffered in channel, not deferred.
      g := gomega.NewWithT(t)
      ch := make(chan chat.Message, 16)
      workerChans := map[string]chan chat.Message{"engram-agent": ch}
      // State: STARTING (not ACTIVE, not HELD)
      stateFile := writeStateFile(t, agentState{name: "engram-agent", state: "STARTING"})
      msg := chat.Message{From: "test", To: "engram-agent", Type: "intent", Text: "test"}
      routeMessage(workerChans, map[string][]chat.Message{}, nil, stateFile, msg)
      g.Expect(ch).To(gomega.HaveLen(1), "STARTING worker should receive intent via channel")
  }
  ```

  Where `routeMessage` is the extracted routing logic from `dispatchLoop` (or test via the full loop with a mock watcher).

  ```bash
  targ test
  # Expected: pass (existing channel logic already handles this correctly)
  ```

- [ ] **Step 3: Commit**

  ```bash
  git add internal/cli/cli_dispatch_test.go \
    docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md
  git commit -m "docs(dispatch): document STARTING-state buffering behavior; add covering test"
  ```

---

## Amendment A4: hold-release → deferredQueue drain via chat message

**Gap:** `dispatchLoop` and `engram hold release` are separate OS processes. The plan says "drain deferredQueue on hold release" but doesn't specify the IPC mechanism. The `hold-release` chat message type is the answer — dispatchLoop must watch for it.

**Refinement accepted from phase6-skill:** Prefer state file lookup for hold-id → target resolution. The state file has hold records with target worker. dispatchLoop reads the state file to find the target from the hold-id, rather than parsing the hold-release message JSON body. This is more robust (hold records are authoritative) and requires no message format change.

**Files:**
- Modify: `internal/cli/cli_dispatch.go` (dispatchLoop type routing)
- Modify: `internal/cli/cli_dispatch_test.go` (TestDispatchLoopHeldWorker drain path)

- [ ] **Step 1: Extend TestDispatchLoopHeldWorker to verify chat-message drain path**

  In `internal/cli/cli_dispatch_test.go`, extend or add alongside `TestDispatchLoopHeldWorker`:

  ```go
  func TestDispatchLoopHoldReleaseDrainsViaChat(t *testing.T) {
      t.Parallel()
      g := gomega.NewWithT(t)
      ch := make(chan chat.Message, 16)
      workerChans := map[string]chan chat.Message{"engram-agent": ch}
      deferredQueues := map[string][]chat.Message{
          "engram-agent": {
              {From: "test", To: "engram-agent", Type: "intent", Text: "queued intent 1"},
              {From: "test", To: "engram-agent", Type: "intent", Text: "queued intent 2"},
          },
      }
      // Simulate hold-release message arriving in chat
      releaseMsg := chat.Message{
          From: "system",
          To:   "engram-agent",
          Type: "hold-release",
          Text: `{"hold-id":"abc","target":"engram-agent"}`,
      }
      routeMessage(workerChans, deferredQueues, nil, stateFilePath, releaseMsg)
      // Both deferred messages should now be in channel
      g.Expect(ch).To(gomega.HaveLen(2))
      g.Expect(deferredQueues["engram-agent"]).To(gomega.BeEmpty())
  }
  ```

  ```bash
  targ test
  # Expected: FAIL — hold-release type not handled in dispatchLoop
  ```

- [ ] **Step 2: Add hold-release case to dispatchLoop**

  In `internal/cli/cli_dispatch.go`, in `dispatchLoop`, extend the type routing section:

  ```go
  // Hold-release: look up the target worker via state file (authoritative), drain deferred queue.
  if msg.Type == "hold-release" {
      // hold-release message text is the hold JSON: {"hold-id":"abc",...}
      var payload struct { HoldID string `json:"hold-id"` }
      if err := json.Unmarshal([]byte(msg.Text), &payload); err == nil && payload.HoldID != "" {
          target := resolveHoldTarget(payload.HoldID, stateFilePath)
          if target != "" {
              drainDeferredQueue(target, workerChans, deferredQueues, stateFilePath, poster, &cursor)
          }
      }
      continue
  }

  // Route only actionable types that require the recipient to take action.
  if msg.Type != "intent" && msg.Type != "wait" && msg.Type != "shutdown" { continue }
  ```

  Add helper:

  ```go
  // resolveHoldTarget looks up the target worker for a hold-id via the state file.
  // The state file holds section is authoritative; no need to parse the chat message body.
  // Returns "" if the hold-id is not found.
  func resolveHoldTarget(holdID string, stateFilePath string) string {
      holds, err := agent.ReadHolds(stateFilePath)
      if err != nil {
          return ""
      }
      for _, h := range holds {
          if h.HoldID == holdID {
              return h.Target
          }
      }
      return ""
  }

  // drainDeferredQueue sends all deferred messages for a worker to its channel
  // and advances last_delivered_cursor for each. Called on hold release or ACTIVE→SILENT.
  func drainDeferredQueue(
      worker string,
      workerChans map[string]chan chat.Message,
      deferredQueues map[string][]chat.Message,
      stateFilePath string,
      poster *chat.FilePoster,
      cursor *int,
  ) {
      ch, ok := workerChans[worker]
      if !ok {
          return
      }
      queue := deferredQueues[worker]
      for _, msg := range queue {
          ch <- msg
          updateLastDeliveredCursor(stateFilePath, worker, *cursor)
          postRoutingInfo(poster, worker, msg, *cursor)
      }
      deferredQueues[worker] = deferredQueues[worker][:0]
  }
  ```

  Update all existing deferredQueue drain callsites (ACTIVE→SILENT transition) to use `drainDeferredQueue`.

- [ ] **Step 3: Run tests green**

  ```bash
  targ test
  # Expected: all pass
  ```

- [ ] **Step 4: `targ check-full` and commit**

  ```bash
  targ check-full
  git add internal/cli/cli_dispatch.go internal/cli/cli_dispatch_test.go
  git commit -m "feat(dispatch): drain deferredQueue on hold-release chat message"
  ```

---

## Amendment A5: ACTIVE vs SILENT argument relay — explicit distinction

**Gap:** Codesign says "argument turns via continued --resume conversation turns." Criterion 14 tests a SILENT worker being resumed for a WAIT. These are two distinct cases but the plan doesn't distinguish them, risking implementors handling only one.

**Refinement accepted from phase6-skill:** Also add ~2 lines to Task 6 Step 2 keep list for ACTIVE-session WAIT handling: 'If you receive a WAIT while ACTIVE (mid-session), stop at a safe point and respond in-session per argument protocol. If resumed with RESUME_REASON=wait (post-SILENT), your next turn IS the argument response.'

**Files:**
- Modify: `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md` (Task 3 Step 2 note + new test)
- Modify: `internal/cli/cli_dispatch_test.go` (new test for ACTIVE case)

- [ ] **Step 1: Add test for WAIT to ACTIVE worker (inline argument relay)**

  In `internal/cli/cli_dispatch_test.go`, add:

  ```go
  func TestDispatchLoopWaitToActiveWorkerDeferred(t *testing.T) {
      t.Parallel()
      // When a worker is ACTIVE (mid-session), a WAIT addressed to it
      // must go to deferredQueue (not channel). The argument relay happens
      // as a --resume turn in the ongoing session; dispatch injects it
      // when the session completes its current turn.
      g := gomega.NewWithT(t)
      ch := make(chan chat.Message, 16)
      workerChans := map[string]chan chat.Message{"worker-a": ch}
      deferredQueues := map[string][]chat.Message{"worker-a": {}}
      stateFile := writeStateFile(t, agentState{name: "worker-a", state: "ACTIVE"})
      waitMsg := chat.Message{
          From: "engram-agent",
          To:   "worker-a",
          Type: "wait",
          Text: "Situation: conflict detected.",
      }
      routeMessage(workerChans, deferredQueues, nil, stateFile, waitMsg)
      g.Expect(ch).To(gomega.HaveLen(0), "ACTIVE worker WAIT must be deferred, not sent to channel")
      g.Expect(deferredQueues["worker-a"]).To(gomega.HaveLen(1))
  }
  ```

  ```bash
  targ test
  # Expected: verify this matches existing behavior (may pass already if ACTIVE check
  # applies to all actionable types including wait)
  ```

- [ ] **Step 2: Add WAIT routing note to Task 3 and Task 6 in main plan**

  In `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md`, in Task 3 Step 2, after the dispatchLoop pseudocode, add:

  ```
  WAIT routing — two distinct cases:
  - ACTIVE target: WAIT is deferred to deferredQueue. On ACTIVE→SILENT transition,
    deferredQueue drains and the WAIT is delivered as the next session's intent
    with RESUME_REASON=wait. This is "argument relay" — the argument continues
    in a new session starting from where DONE: left off.
  - SILENT target: WAIT is delivered directly to the worker's channel.
    runConversationLoopWith resumes the worker immediately with RESUME_REASON=wait
    in the prompt.

  In both cases the worker sees RESUME_REASON=wait in its next resume prompt.
  The codesign decision "continued --resume conversation turns" describes argument
  continuation across multiple resume cycles, not a single persistent session.
  ```

  In Task 6 Step 2 keep list, add (per phase6-skill refinement):
  ```
  - ACTIVE-session WAIT handling (~2 lines):
    If you receive a WAIT while ACTIVE (mid-session), stop at a safe point and respond
    in-session per argument protocol. If resumed with RESUME_REASON=wait (post-SILENT),
    your entire next turn IS the argument response — read WAIT_TEXT, respond, say DONE:.
  ```

- [ ] **Step 3: `targ check-full` and commit**

  ```bash
  targ check-full
  git add internal/cli/cli_dispatch_test.go \
    docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md
  git commit -m "docs(dispatch): clarify ACTIVE vs SILENT WAIT routing; add covering test"
  ```

---

## Amendment A6: Argument crash recovery — known limitation

**Gap:** When dispatch crashes while a worker is ACTIVE in an argument relay, restart marks the worker DEAD and argument state is lost. No Criterion covers this. The counterparty's WAIT goes unresolved.

**Files:**
- Modify: `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md` (known limitations section)

No code change. Accept as known limitation with documented recovery path.

- [ ] **Step 1: Add known limitation to main plan**

  In `docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md`, after the Codesign Decisions table, add:

  ```markdown
  ## Known Limitations

  | Limitation | Behavior | Recovery path |
  |-----------|----------|---------------|
  | Crash during argument relay | Worker marked DEAD on restart. In-flight argument state (argument_turn, argument_partner) is wiped. Counterparty's WAIT goes unresolved. | Counterparty's ack-wait times out → escalate to lead. Lead surfaces to user. Argument must be restarted manually. Acceptable for Phase 6. Robustness improvement is post-Phase-6 scope. |
  | Dirty SIGKILL of worker | LEARNED: messages posted after last delivery cursor are not lost (they're in chat) but won't be extracted until next intent cycle. No memory data loss — only a one-cycle extraction delay. | Next intent resume will call collectLearned from last_delivered_cursor and include the messages. |
  ```

- [ ] **Step 2: Commit**

  ```bash
  git add docs/superpowers/plans/2026-04-11-phase6-engram-dispatch.md
  git commit -m "docs(phase6): add known limitations — argument crash recovery, dirty kill"
  ```

---

## Self-Review

**Spec coverage:**
- Gap 1 (INTENT: TO): A1 adds test + implementation + plan note ✓
- Gap 2 (RESUME_REASON): A2 adds test for ARGUMENT_ESCALATION_NOTE + Task 8 checklist item 0 ✓
- Gap 3 (STARTING): A3 adds test + comment (no code change needed) ✓
- Gap 4 (hold-release IPC): A4 adds hold-release case to dispatchLoop + test ✓
- Gap 5 (ACTIVE vs SILENT): A5 adds test + plan note distinguishing the two cases ✓
- Gap 6 (crash recovery): A6 documents known limitation ✓

**Placeholder scan:** None found.

**Type consistency:**
- `drainDeferredQueue` introduced in A4, used consistently in A4 and referenced in A5.
- `parseHoldReleaseTarget` introduced and used only in A4.
- `parseIntentMarkerTO` introduced and tested only in A1.
- All function signatures match between tests and implementations.
