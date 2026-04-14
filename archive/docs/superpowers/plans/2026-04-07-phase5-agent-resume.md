# Phase 5 — Agent Resume + Auto-Resume Watch Loop

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert `engram-agent` from a stateful long-running process to a stateless per-invocation worker. The binary's `runConversationLoop` extends to an outer watch loop: after a session ends (DONE: or no INTENT:), the binary writes state SILENT, watches the chat file for the next `type=intent` addressed to `engram-agent`, then starts a fresh `claude -p` session with a structured resume prompt containing `CURSOR:`, `MEMORY_FILES:`, `INTENT_FROM:`, and `INTENT_TEXT:`. `engram agent run` never exits while the watch loop is active.

**Architecture:** Two new binary responsibilities: (1) `selectMemoryFiles` ranks recent memory files by mtime and injects absolute paths into the resume prompt; (2) the outer watch loop in `runConversationLoopWith` drives stateless invocation cycles. Three state transitions: STARTING→ACTIVE (first READY: marker), ACTIVE→SILENT (session ends), SILENT→ACTIVE (watch loop fires new session). Worker queue (max 3 concurrent) prevents duplicate active sessions. Holds remain kill-gates only — orthogonal to dispatch.

**Spec:** `docs/superpowers/specs/engram-deterministic-coordination-design.md` §7 (Phase 5)

**Codesign session:** planners arch-exec-planner (binary architecture), skill-exec-planner (skill rewrite + Item 5), agent-e2e-planner (agent E2E), user-e2e-planner (user E2E) — 2026-04-07. All four perspectives converged before this plan was written. Codesign decisions are locked in the chat file (`~/.local/share/engram/chat/-Users-joe-repos-personal-engram.toml`) around line 98094–100605.

**Tech Stack:** Go, existing `internal/claude`, `internal/chat`, `internal/agent`, `internal/cli`, `internal/watch` packages. No new dependencies.

---

## Pre-Flight (issue audit — do before Task 0)

| Item | Action | Rationale |
|------|--------|-----------|
| Phase 4 E2E criteria | Verify all 5 pass: session-id UUID, speech relay live, display filter, `-p` mode, skill deletions safe | Phase 5 builds on Phase 4 pipeline. Any regression here is a Phase 4 bug, not Phase 5 work. |
| Pre-Phase-5 blockers merged | Confirm commit 97dd379 present: `git log --oneline` | Naming guidance + loop exit contract required before Phase 5 codesign began. |
| State file has real session-id | `engram agent list` shows engram-agent with non-PENDING, non-`$N` session-id | Phase 5 watch loop calls `--resume`; PENDING session-id = crash on first auto-resume. |
| `targ check-full` passes clean | Zero issues before Phase 5 code is added | Distinguish pre-existing issues from new ones introduced by Phase 5 changes. |
| Untracked issues from Phase 4 | Close or triage any open issues referencing Phase 4 work | Clean issue tracker reduces noise during Phase 5 implementation. |

---

## E2E Acceptance Criteria

Phase 5 is done when all criteria pass. Binary criteria (1–5) must pass before skill deletions (Task 7). Criterion 6 baseline must pass before skill rewrites take effect.

### Criterion 1: Auto-Resume Triggered

After `engram agent spawn engram-agent`, the agent processes an intent, outputs `DONE:`, and then — WITHOUT any user action — the binary detects the next `type=intent` addressed to `engram-agent` and starts a new `claude -p` session automatically. The new session's first JSONL event must appear in the pane.

```bash
# Post a test intent after DONE: is seen in the pane
engram chat post --from test --to engram-agent --thread test --type intent --text "Situation: test. Behavior: respond ACK."
# Expected: new claude -p session fires without manual intervention
```

### Criterion 2: STARTING→ACTIVE State Transition

After `engram agent spawn`, when the spawned agent outputs `READY:` as its first prefix marker, the state file must change from `STARTING` to `ACTIVE`.

```bash
engram agent list | jq -r '.state'
# Expected: "ACTIVE" (not "STARTING") after first READY: marker
```

### Criterion 3: Worker Queue Max 3 Concurrent

A fourth `engram agent spawn` when 3 agents are in STARTING or ACTIVE state must fail with a clear error message.

```bash
# Spawn 3 agents, then attempt a 4th:
engram agent spawn --name test-4 --prompt "..."
# Expected: error "agent spawn: worker queue full (max 3 concurrent)"
```

### Criterion 4: Resume Prompt Correctness

The resume prompt delivered to each new watch-loop invocation must contain all four fields: `CURSOR:`, `MEMORY_FILES:` (with at least one path if memory directory is non-empty), `INTENT_FROM:`, `INTENT_TEXT:`.

```bash
# Inspect pane or add debug logging to buildResumePrompt; verify field presence
```

### Criterion 5: No Background Monitor Subagent

A Phase 5 engram-agent session must NOT invoke the `Agent` tool to spawn a background monitor. The pane output should show no `Spawning background agent` or equivalent output. Verify via pane capture after a complete intent-process cycle.

### Criterion 6: Stateless Memory Load

Each new watch-loop invocation cold-loads memory from the `MEMORY_FILES:` paths in the resume prompt. The agent must NOT reference facts or memories from a prior session that are not in the injected files. Baseline: record loaded files on invocation N; verify invocation N+1 does not carry session N context.

### Criterion 7: Fresh Session Per Watch-Loop Invocation

The session-id in the state file must CHANGE between watch-loop invocations (each fires a new `claude -p` without `--resume`).

```bash
engram agent list | jq -r '.["session-id"]'
# Before first auto-resume: UUID-A
# After first auto-resume: UUID-B (different from UUID-A)
```

### Criterion 8: WAIT: Exits Session Cleanly

When engram-agent outputs `WAIT:`, the session exits (watch loop continues). The initiating agent receives `WAIT from engram-agent: [text]` via its ack-wait. The watch loop then fires on the initiating agent's follow-up `type=intent`.

### Criterion 9: Hold Orthogonal to Auto-Resume

When engram-agent is on hold and SILENT, the watch loop fires normally on an incoming intent. Hold status is NOT checked by the dispatch path. Verify: held SILENT engram-agent auto-resumes when an intent arrives.

### Criterion 10: Argument Delivery via Revised Intent

After engram-agent posts `WAIT:` (objection), the initiating agent reads the WAIT: content, adjusts its approach, and re-posts a revised `type=intent` to the chat file. The watch loop fires on this revised intent, starting a new engram-agent session. The watch loop watches `type=intent` only — no `type=wait` filter needed. Verify: revised intent after WAIT: triggers new session without any binary changes to the watch type filter.

---

## Codesign Decisions

These decisions were argued and resolved during the 2026-04-07 codesign session. Do NOT revisit without reading the `codesign-phase5-exec` thread in the chat file from line ~98094.

| Decision | Resolved | Rationale |
|----------|----------|-----------|
| Stateless per-invocation model for engram-agent | Yes | No persistent context window between watch-loop invocations. Each cold-loads from MEMORY_FILES:. State is injected, not carried. |
| Watch loop: Option A — extend runConversationLoop | Yes | Binary stays in watch loop after DONE/no-INTENT. Outer loop wraps existing inner loop. agentName/chatFilePath/stateFilePath parameterized for Phase 6 extraction. |
| Fresh session per watch-loop invocation | Yes | Each watch-loop fire calls `claude -p --prompt` (no `--resume`). New session-id written to state file on first JSONL event. |
| WriteState callback on claude.Runner | Yes | Same DI pattern as WriteSessionID. WriteState("ACTIVE") called on first READY: marker. WriteState("SILENT") called by outer loop after session ends. |
| STARTING→ACTIVE on first READY: only | Yes | Idempotent: subsequent READY: markers are no-ops. ReadyDetected bool in StreamResult tracks whether transition has fired. |
| WaitDetected bool in StreamResult | Yes | Informational — all outer loop exit conditions (DONE/WAIT/!INTENT) lead to same behavior: write SILENT, watch for next intent. |
| selectMemoryFiles in internal/cli | Yes | Requires injected readDir/stat funcs — belongs at CLI boundary layer. internal/agent stays pure domain. |
| Top-20 by mtime, absolute paths, no TOML parsing | Yes | Binary reads DirEntry mtime only. No TOML interpretation in Phase 5. Skill receives absolute paths (no ~ expansion needed). |
| buildResumePrompt as private cli_agent.go function | Yes | Single call site. Exported via export_test.go for unit testing. |
| Resume prompt format: CURSOR/MEMORY_FILES/INTENT_FROM/INTENT_TEXT | Yes | Matches locked Phase 4 Post-Phase-4 spec (Item 6). No INTENT_TYPE: field — watch fires on type=intent only, so INTENT_TYPE is always "intent" (no-op). |
| activeWorkerCount pure function in internal/agent | Yes | < 10 lines, trivially testable. Check inside readModifyWriteStateFile lock in runAgentSpawn. |
| Hold orthogonal to dispatch | Yes | Holds are kill-gates, not dispatch-gates. Watch loop is hold-blind. Worker queue is the sole duplicate-session guard. |
| SILENT state written BEFORE entering watch | Yes | State file is source of truth. SILENT = no active claude -p. Written before watch loop blocks. |
| Watch loop starts from EOF at startup | Yes | Binary restart starts from current chat file end. Old intents not replayed. Intents missed during binary downtime are not re-delivered (acceptable; Phase 6 adds persistent cursor to state file). |
| WAIT: argument continuation is stateless | Yes | Executor receives WAIT response, adjusts, re-posts as new type=intent. Each argument round is a fresh invocation. No prior argument context in Phase 5 (Phase 6 adds RECENT_INTENTS:). |
| last-resumed-at updated per watch-loop fire | Yes | New state file write inside outer loop when intent is received. |
| maxTurns cap: inner loop only | Yes | Outer watch loop has no turn cap — runs until ctx cancelled or shutdown received. Inner within-session loop keeps maxTurns = 50. |
| Phase 6 extraction target: parameterized signature | Yes | runConversationLoopWith gains agentName, chatFilePath, stateFilePath params even though Phase 5 only calls it with "engram-agent". Documents Phase 6 dispatcher extraction point. |

---

## File Structure

```
internal/
  agent/
    agent.go          MODIFY — add activeWorkerCount, MaxConcurrentWorkers const
    agent_test.go     MODIFY — add activeWorkerCount tests

  claude/
    claude.go         MODIFY — add WriteState to Runner, ReadyDetected+WaitDetected to StreamResult
    claude_test.go    MODIFY — add WriteState + WaitDetected tests

  cli/
    cli_agent.go      MODIFY — buildResumePrompt, selectMemoryFiles, outer watch loop,
                               WriteState wiring, last-resumed-at update, errWorkerQueueFull
    cli_test.go       MODIFY — tests for new functions + watch loop behavior
    export_test.go    MODIFY — export BuildResumePrompt, SelectMemoryFiles

skills/
  engram-agent/SKILL.md         MODIFY — Task 7 (stateless rewrite)
  use-engram-chat-as/SKILL.md   MODIFY — Task 8 (Background Monitor Pattern note cleanup)
```

---

## Task 0: Pre-flight verification

**Files:** None modified. Verification only.

- [ ] **Step 1: Verify Phase 4 E2E criteria 1–5 still pass**

```bash
# Criterion 1: session-id is a UUID
engram agent list | jq -r '.["session-id"]'
# Expected: UUID format (not PENDING, not $N)

# Criterion 2: post an intent, check chat file
tail -n 20 ~/.local/share/engram/chat/-Users-joe-repos-personal-engram.toml
# Expected: type = "intent" with from = agent-name (not "system")

# Criterion 3: visual — spawn agent, check pane for raw JSONL
# Expected: no lines starting with {"type":"tool_use" etc.

# Criterion 4: manual — engram agent run launches without permission errors
```

- [ ] **Step 2: Verify Pre-Phase-5 blockers merged**

```bash
git log --oneline | grep -E "97dd379|naming guidance|loop exit"
# Expected: commit 97dd379 present
```

---

## Task 1: StreamResult expansion + WriteState callback on claude.Runner

**Files:**
- Modify: `internal/claude/claude.go`
- Modify: `internal/claude/claude_test.go`

- [ ] **Step 1: Write failing tests**

```go
// Test: WriteState called with "ACTIVE" on first READY: marker
// Test: WriteState NOT called again on second READY: marker (idempotent)
// Test: WaitDetected=true when WAIT: marker in stream
// Test: ReadyDetected=true when READY: marker in stream
// Test: WriteState=nil (no panic when not injected)
```

- [ ] **Step 2: Add ReadyDetected and WaitDetected to StreamResult**

```go
type StreamResult struct {
    IntentDetected bool
    DoneDetected   bool
    WaitDetected   bool   // WAIT: prefix marker detected
    ReadyDetected  bool   // READY: prefix marker detected (triggers STARTING→ACTIVE)
    SessionID      string
}
```

- [ ] **Step 3: Add WriteState to Runner**

```go
// WriteState is called with "ACTIVE" when the first READY: marker is detected.
// Nil = skip (tests that don't need state transitions).
WriteState func(state string) error
```

- [ ] **Step 4: Update handleEvent for READY: and WAIT: cases**

In the READY: case (already in markerToMsgType switch), after relay:
```go
if marker.Prefix == "READY" && !result.ReadyDetected {
    result.ReadyDetected = true
    if r.WriteState != nil {
        if stateErr := r.WriteState("ACTIVE"); stateErr != nil {
            _, _ = fmt.Fprintf(r.Pane, "[engram] warning: state transition failed: %v\n", stateErr)
        }
    }
}
if marker.Prefix == "WAIT" {
    result.WaitDetected = true
}
```

- [ ] **Step 5: Run tests green**

```bash
targ test
```

---

## Task 2: activeWorkerCount + worker queue guard in runAgentSpawn

**Files:**
- Modify: `internal/agent/agent.go`
- Modify: `internal/agent/agent_test.go`
- Modify: `internal/cli/cli_agent.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Write failing tests**

```go
// Test: activeWorkerCount returns 0 for empty StateFile
// Test: activeWorkerCount counts STARTING agents
// Test: activeWorkerCount counts ACTIVE agents
// Test: activeWorkerCount ignores SILENT, DEAD, UNKNOWN agents
// Test: runAgentSpawn errors with errWorkerQueueFull when 3 workers active
```

- [ ] **Step 2: Add activeWorkerCount and MaxConcurrentWorkers to internal/agent**

```go
// MaxConcurrentWorkers is the maximum number of agents in STARTING or ACTIVE state.
const MaxConcurrentWorkers = 3

// activeWorkerCount returns the number of agents in STARTING or ACTIVE state.
// Pure function — no I/O.
func activeWorkerCount(sf StateFile) int {
    count := 0
    for _, rec := range sf.Agents {
        if rec.State == "STARTING" || rec.State == "ACTIVE" {
            count++
        }
    }
    return count
}
```

- [ ] **Step 3: Add errWorkerQueueFull sentinel and queue guard to runAgentSpawn**

```go
var errWorkerQueueFull = errors.New("agent spawn: worker queue full (max 3 concurrent)")
```

Inside `readModifyWriteStateFile` modify func in `runAgentSpawn`, before `AddAgent`:
```go
if agentpkg.activeWorkerCount(sf) >= agentpkg.MaxConcurrentWorkers {
    // Return sf unmodified; caller checks err separately
}
```

Note: readModifyWriteStateFile's modify func doesn't return an error. The queue guard must be implemented by checking the count BEFORE the RMW call, reading the state file first. See implementation note in cli_agent.go.

- [ ] **Step 4: Run tests green**

```bash
targ test
```

---

## Task 3: selectMemoryFiles + buildResumePrompt

**Files:**
- Modify: `internal/cli/cli_agent.go`
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Write failing tests**

```go
// Test: selectMemoryFiles returns empty slice when both dirs empty
// Test: selectMemoryFiles returns top N by mtime from feedback/ and facts/
// Test: selectMemoryFiles returns absolute paths
// Test: selectMemoryFiles caps at resumeMemoryFileLimit (20)
// Test: buildResumePrompt includes CURSOR: field
// Test: buildResumePrompt includes MEMORY_FILES: section with all paths
// Test: buildResumePrompt includes INTENT_FROM: and INTENT_TEXT:
// Test: buildResumePrompt with empty memFiles produces empty MEMORY_FILES: section
```

- [ ] **Step 2: Implement selectMemoryFiles**

```go
const resumeMemoryFileLimit = 20

// selectMemoryFiles returns up to maxFiles memory file paths sorted by mtime descending.
// Reads feedbackDir and factsDir. Returns absolute paths.
// Pure at the boundary: readDir and statFile are injected for testability.
func selectMemoryFiles(
    feedbackDir, factsDir string,
    readDir func(string) ([]fs.DirEntry, error),
    statFile func(string) (fs.FileInfo, error),
    maxFiles int,
) ([]string, error)
```

- [ ] **Step 3: Implement buildResumePrompt**

```go
// buildResumePrompt constructs the resume prompt for a stateless engram-agent invocation.
// Format is defined by Phase 5 codesign (Item 6 from Phase 4 Post-Phase-4 doc).
func buildResumePrompt(cursor int, memFiles []string, intentFrom, intentText string) string
```

Format:
```
CURSOR: <N>
MEMORY_FILES:
<path1>
<path2>
INTENT_FROM: <agent-name>
INTENT_TEXT: <full text of intent message>
Instruction: Load the files listed under MEMORY_FILES. Use the CURSOR value when calling engram chat ack-wait. Respond to the intent above with ACK:, WAIT:, or INTENT:.
```

- [ ] **Step 4: Export via export_test.go**

```go
var SelectMemoryFiles = selectMemoryFiles
var BuildResumePrompt = buildResumePrompt
```

- [ ] **Step 5: Run tests green**

```bash
targ test
```

---

## Task 4: Outer watch loop — extend runConversationLoopWith

**Files:**
- Modify: `internal/cli/cli_agent.go`
- Modify: `internal/cli/cli_test.go`

This is the core Phase 5 binary change. Restructures `runConversationLoopWith` into two nested loops.

- [ ] **Step 1: Write failing tests**

```go
// Test: after DoneDetected, loop watches for next intent (does not return immediately)
// Test: after WaitDetected, loop watches for next intent (same behavior as DONE)
// Test: after !IntentDetected, loop watches for next intent
// Test: ctx cancellation during watch returns nil (clean exit)
// Test: WriteState("SILENT") called after session ends (before watch)
// Test: WriteState("ACTIVE") called when new session fires READY:
// Test: fresh sessionID="" on each watch-loop invocation (not --resume)
// Test: last-resumed-at updated when intent received
// Test: watch loop uses cursor from end of prior turn (not reset to 0)
```

- [ ] **Step 2: Extract inner session loop as runWithinSessionLoop**

```go
// runWithinSessionLoop drives the within-session INTENT→ack-wait→resume cycles (Phase 4 logic).
// Returns the final StreamResult and updated sessionID.
func runWithinSessionLoop(
    ctx context.Context,
    runner claudepkg.Runner,
    initialPrompt, initialSessionID string,
    chatFilePath, claudeBinary string,
    stdout io.Writer,
    promptBuilder promptBuilderFunc,
    cursor int,
) (claudepkg.StreamResult, string, int, error)
```

- [ ] **Step 3: Add new injectable seam types**

```go
// watchForIntentFunc watches for the next intent addressed to agentName.
// Phase 5: hardcoded to "engram-agent". Phase 6: routing table calls same type.
type watchForIntentFunc func(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error)

// memFileSelectorFunc selects memory files for resume prompt injection.
type memFileSelectorFunc func(homeDir string, maxFiles int) ([]string, error)
```

- [ ] **Step 4: Update runConversationLoopWith signature (HARD CONSTRAINT)**

```go
func runConversationLoopWith(
    ctx context.Context,
    runner claudepkg.Runner,
    flags agentRunFlags,
    agentName string,        // parameterized — Phase 6 extraction target
    chatFilePath string,
    stateFilePath string,    // NEW: required for WriteState calls in outer loop
    claudeBinary string,
    stdout io.Writer,
    promptBuilder promptBuilderFunc,
    watchForIntent watchForIntentFunc,   // NEW: injectable for tests
    memFileSelector memFileSelectorFunc, // NEW: injectable for tests
) error
```

- [ ] **Step 5: Implement outer watch loop**

```
Start: cursor = current chat file EOF
Loop (outer):
    Run runWithinSessionLoop(ctx, runner, prompt, sessionID, ...)
    → result, sessionID, cursor, err

    If err: return err
    If ctx.Done(): return nil (clean exit)

    WriteState("SILENT")

    intentMsg, cursor, watchErr = watchForIntent(ctx, agentName, chatFilePath, cursor)
    If watchErr (ctx cancelled): return nil

    Update last-resumed-at in state file
    sessionID = ""  (fresh session — no --resume)

    memFiles = memFileSelector(homeDir, resumeMemoryFileLimit)
    prompt = buildResumePrompt(cursor, memFiles, intentMsg.From, intentMsg.Text)

    Go to Loop (outer)
```

- [ ] **Step 6: Wire production implementations**

```go
// Production watchForIntent using existing chat.FileWatcher
func defaultWatchForIntent(ctx context.Context, agentName, chatFilePath string, cursor int) (chat.Message, int, error) {
    watcher := newFileWatcher(chatFilePath)
    return watcher.Watch(ctx, agentName, cursor, []string{"intent"})
}

// Production memFileSelector using os.ReadDir and os.Stat
func defaultMemFileSelector(homeDir string, maxFiles int) ([]string, error) {
    feedbackDir := filepath.Join(homeDir, ".local/share/engram/memory/feedback")
    factsDir := filepath.Join(homeDir, ".local/share/engram/memory/facts")
    return selectMemoryFiles(feedbackDir, factsDir, os.ReadDir, os.Stat, maxFiles)
}
```

- [ ] **Step 7: Run tests green**

```bash
targ check-full
```

---

## Task 5: Wire WriteState + last-resumed-at in buildAgentRunner

**Files:**
- Modify: `internal/cli/cli_agent.go`
- Modify: `internal/cli/cli_test.go`

- [ ] **Step 1: Add WriteState to buildAgentRunner**

```go
WriteState: func(state string) error {
    return readModifyWriteStateFile(stateFilePath, func(sf agentpkg.StateFile) agentpkg.StateFile {
        for i, rec := range sf.Agents {
            if rec.Name == flags.name {
                sf.Agents[i].State = state
                return sf
            }
        }
        return sf
    })
},
```

- [ ] **Step 2: Run full quality check**

```bash
targ check-full
```

---

## Task 6: `engram-agent` Skill Rewrite (Stateless Model)

**Files:**
- Modify: `skills/engram-agent/SKILL.md`

**REQUIRED SUB-SKILL:** Use `superpowers:writing-skills` for ALL edits. No exceptions.

**Scope:** Major rewrite. Deletes the stateful watch-loop model, subagent spawning, and tiered startup loading. Replaces with a stateless resume-context model. Retains all memory matching, learning, and locking logic unchanged.

**Deletions (~110 lines):**
- Main Loop section — Steps 1–4 (Background Monitor Pattern, background Agent spawning, wait loop): ~35 lines
- Setup steps 3–5 (`Initialize recent_intents = []`, "Enter the watch loop"): ~4 lines
- Tiered Loading section (startup loading strategy, auto-promotion/demotion/cap): ~20 lines
- Subagent Management subsection (max 3, queue, naming, routing, timeout): ~15 lines
- Performance Tracking table rows for "Subagent queue depth" and "Intents seen vs checked": ~4 lines
- "(currently via the Background Monitor Pattern)" parenthetical in Responding via Prefix Markers: ~2 lines
- Feedback Surfacing steps c–d (subagent spawn block): ~20 lines (replaced with direct response)

**Additions (~35 lines):**
- Resume Context section (new): CURSOR:/MEMORY_FILES:/INTENT_FROM:/INTENT_TEXT: parsing guidance
- Stateless startup sequence (replaces Setup steps 3–5): parse context → load files → respond
- Edge case: no MEMORY_FILES: in context → ACK: with warning (binary bug; not expected in production)
- Direct feedback judgment in Feedback Surfacing (replaces subagent spawn)

**Rewrites:**
- Setup section (keep steps 1–2, replace 3–5 with stateless startup)
- Tiered Loading → Memory Loading (load from MEMORY_FILES: list; no directory scan)
- Feedback Surfacing steps c–d → direct judgment + WAIT:/ACK: response (no subagent)
- Common Mistakes table (3 deletions, 2 additions)

---

- [ ] **Step 1: Invoke writing-skills skill**

Before any edits: invoke `superpowers:writing-skills`. Run the RED baseline behavior test first.

- [ ] **Step 2: Audit deletion targets**

Locate exact line ranges for each deletion before cutting:

```bash
grep -n "## Main Loop\|## Tiered Loading\|### Subagent Management\|recent_intents\|Background Monitor Pattern\|If fewer than 3 subagents\|Give the subagent" skills/engram-agent/SKILL.md
```

Document each line range in working notes before editing.

- [ ] **Step 3: Delete stateful watch-loop sections**

Delete in order (work bottom-to-top to avoid line number drift):
1. Main Loop section (Steps 1–4 and loop diagram): from `## Main Loop` through the diagram, ending before `## Responding via Prefix Markers`
2. Setup steps 3–5 (`Initialize recent_intents = []` and "Enter the watch loop"): keep steps 1–2
3. "(currently via the Background Monitor Pattern)" parenthetical in Responding via Prefix Markers

- [ ] **Step 4: Delete Subagent Management subsection**

Delete the entire `### Subagent Management` subsection. Also delete Performance Tracking rows for "Subagent queue depth" and "Intents seen vs checked / Agent overwhelm".

- [ ] **Step 5: Rewrite Tiered Loading → Memory Loading**

Replace the `## Tiered Loading` section with:

```markdown
## Memory Loading

On each invocation, load exactly the files listed in `MEMORY_FILES:` from your resume context.

Do **not** scan `~/.local/share/engram/memory/` directly — the binary has already selected
the most relevant files by recency (top 20 by mtime across feedback/ and facts/ directories).

**Situations-only loading still applies:** Load only the `situation` field and filename slug
for each memory initially. Full records are loaded only when a situation match is found.

**Empty MEMORY_FILES: block:** If no files are listed, post `ACK:` with a note:
"No memory files loaded — MEMORY_FILES: was empty. No memories surfaced."
This is a binary-side condition (empty memory directory); it is not an error.
```

- [ ] **Step 6: Add Resume Context section**

Insert after the Setup section (or as the first new section after the preamble):

```markdown
## Resume Context

Each invocation receives a structured resume context injected by the binary:

```
CURSOR: <N>
MEMORY_FILES:
~/.local/share/engram/memory/feedback/foo.toml
~/.local/share/engram/memory/facts/bar.toml
INTENT_FROM: <agent-name>
INTENT_TEXT: <full text of the intent to respond to>
Instruction: Load the files listed under MEMORY_FILES. Use the CURSOR value
when calling engram chat ack-wait. Respond to the intent above with ACK:,
WAIT:, or INTENT:.
```

Parse these fields on every invocation:
1. `CURSOR:` — integer value; pass to `--cursor` when calling `engram chat ack-wait`
2. `MEMORY_FILES:` — one absolute path per line; read each with the Read tool
3. `INTENT_FROM:` — the agent who posted the intent you must respond to
4. `INTENT_TEXT:` — the full intent text; use as input to your matching logic

**You have NO prior conversation history.** Each invocation is a fresh session.
Your context is exactly what is listed above — nothing more.
```

- [ ] **Step 7: Rewrite Setup section startup steps**

Replace Setup steps 3–5 with:

```markdown
3. Parse resume context (see Resume Context section): CURSOR:, MEMORY_FILES:, INTENT_FROM:, INTENT_TEXT:
4. Load memory files from MEMORY_FILES: list (see Memory Loading section)
5. Run matching logic against loaded memories
6. Respond with ACK: or WAIT: — your session ends after responding
```

- [ ] **Step 8: Simplify Feedback Surfacing subagent block**

Replace Feedback Surfacing steps c–d (the subagent spawn block) with:

```markdown
c. Read the full memory TOML file (all fields).
d. Judge whether the intended behavior resembles the memory's "behavior to avoid":
   - **No behavior match** (situation matched but behavior is fine): post `ack` — false positive.
   - **Behavior matches**: post `wait` with the memory's full content and why it applies.
     You are the **reactor**: be direct and specific. State exactly which behavior concerns you.
     Do not wait for a counter-argument — your session ends after posting WAIT:.
     The initiating agent reads your concern from the chat file and adjusts their approach.
     If they post a revised `type=intent` after reading your WAIT:, the binary will resume you
     with the revised intent in a new session.
```

- [ ] **Step 9: Update Common Mistakes table**

Delete rows referencing subagents:
- "Spawning unlimited subagents — Max 3 concurrent, no two on same thread. Queue the rest."
- "Subagent using cached memory data — Always re-read the file fresh before writing."
- "Subagent being too polite — The reactor role is AGGRESSIVE. Push back hard."

Add rows:
- "Scanning memory directories directly — Load only from MEMORY_FILES: list. Binary has already selected relevant files."
- "Expecting counter-argument delivery in same session — Phase 5 WAIT: is fire-and-done. Post your concern; your session ends. The initiating agent adjusts and re-posts."

- [ ] **Step 10: Run writing-skills TDD cycle and commit**

```bash
git add skills/engram-agent/SKILL.md
git commit -m "feat(skills): rewrite engram-agent for Phase 5 stateless model

Phase 5 skill update. Deletes ~110 lines: Main Loop (Background Monitor
Pattern), Tiered Loading startup strategy, Subagent Management, and
Performance Tracking queue metrics. Adds ~35 lines: Resume Context section
(CURSOR:/MEMORY_FILES:/INTENT_FROM:/INTENT_TEXT: parsing), stateless startup
sequence, Memory Loading from injected file list.

Retains all memory matching, learning, and locking logic unchanged. Feedback
Surfacing simplified: direct WAIT:/ACK: response without subagent spawning.
Phase 5: WAIT: is fire-and-done; argument continuation is manual via
initiating-agent intent re-post."
```

---

## Task 7: `use-engram-chat-as` Cleanup

**Files:**
- Modify: `skills/use-engram-chat-as/SKILL.md`

**REQUIRED SUB-SKILL:** Use `superpowers:writing-skills`. No exceptions.

**Scope:** Targeted deletion (~2–3 lines). Remove the "survives through Phase 4" note from the Background Monitor Pattern section. This note was deferred from Phase 3 (see Phase 4 pre-flight table). The Background Monitor Pattern itself is **retained** — interactive agents (leads, planners) still use it. Only `engram-agent` no longer uses it.

**Deletion target:**
```
This pattern survives through Phase 4. Phase 5 (`engram agent resume`) eliminates it
by converting engram-agent to a stateless worker.
```

---

- [ ] **Step 1: Invoke writing-skills skill**

Before any edits: invoke `superpowers:writing-skills`. Run the RED baseline behavior test.

- [ ] **Step 2: Locate and delete the Phase 4 note**

```bash
grep -n "survives through Phase 4\|converting engram-agent to a stateless" skills/use-engram-chat-as/SKILL.md
```

Expected: 1–2 lines in the Background Monitor Pattern section. Delete the sentence(s) referencing Phase 4 survival. Verify the surrounding verbatim template block and spawning instructions are unchanged.

- [ ] **Step 3: Run writing-skills TDD cycle and commit**

```bash
git add skills/use-engram-chat-as/SKILL.md
git commit -m "feat(skills): remove Phase 4 survival note from Background Monitor Pattern

Phase 5 skill cleanup. Deletes the 'This pattern survives through Phase 4'
note deferred from Phase 3. engram-agent no longer uses the Background
Monitor Pattern (stateless worker model). Pattern remains in skill for
interactive agents (leads, executors, planners)."
```

---

## Task 8: E2E verification

**Files:** None modified. Verification only.

Criteria 1–8 must pass before Category B skill deletions (Task 6). All 10 criteria must pass before Phase 5 is declared complete.

### Step 1: Run Phase 4 baseline guard (Task 0)

See Task 0. All Phase 4 criteria must still pass before proceeding.

### Step 2: Scenario A — Cold Start + ACTIVE Transition (Criteria 1, 2)

```bash
engram agent spawn engram-agent --prompt 'You are engram-agent. Post READY: then ACK: the intent in your prompt, then DONE:'
sleep 3
engram agent list | jq -r '.["engram-agent"].state'
# Expected: ACTIVE (not STARTING)

engram chat post --from lead --to engram-agent --thread test --type intent \
  --text 'Situation: startup test. Behavior: acknowledge with ACK:.'
# Wait ~5s, then:
tail -n 20 ~/.local/share/engram/chat/-Users-joe-repos-personal-engram.toml
# Expected: from = "engram-agent", type = "ack"
```

**PASS when:** state=ACTIVE after READY:; intent receives ACK: response; `engram agent run` still alive.

### Step 3: Scenario B — Sequential Auto-Resume (Criteria 1, 7)

After Scenario A DONE: is detected:

```bash
SESSION_1=$(engram agent list | jq -r '.["engram-agent"]["session-id"]')

engram chat post --from lead --to engram-agent --thread test --type intent \
  --text 'Situation: second intent. Behavior: ACK:.'
sleep 5
SESSION_2=$(engram agent list | jq -r '.["engram-agent"]["session-id"]')

[ "$SESSION_1" != "$SESSION_2" ] && echo 'PASS: session-id changed' || echo 'FAIL: session-id unchanged'
```

**PASS when:** both intents handled; session-id changed (UUID-A ≠ UUID-B); binary still alive without manual restart.

### Step 4: Scenario C — Worker Queue Enforcement (Criterion 3)

```bash
for i in 1 2 3; do
  engram agent spawn --name worker-$i --prompt 'Post READY: then wait 30 seconds before DONE:'
done
engram agent spawn --name worker-4 --prompt 'Test'
echo "Exit: $?"
# Expected: non-zero exit; message contains 'worker queue full'

engram agent list | jq '[.[] | select(.state == "STARTING" or .state == "ACTIVE")] | length'
# Expected: 3 (not 4)
```

**PASS when:** 4th spawn fails with 'worker queue full'. State shows exactly 3 STARTING/ACTIVE agents.

### Step 5: Scenario D — Resume Prompt Correctness (Criterion 4)

Unit-test first (fast feedback):

```bash
targ test
# Verify BuildResumePrompt and SelectMemoryFiles tests pass.
```

Live behavioral check: after Scenario A or B, inspect engram-agent's response in the pane. The agent must reference the intent topic from INTENT_TEXT: (not a generic fallback response), confirming INTENT_TEXT: was received.

**PASS when:** unit tests green; pane output contextually references the intent topic.

### Step 6: Skill Deletions Safety Gate (Criteria 5, 6) — Category B gate

After Criteria 1–4 pass, apply Task 6 (engram-agent skill rewrite). Re-run Scenarios A–B.

```bash
# Criterion 5: No background monitor subagent
# Watch pane — no Agent tool calls should appear
# Expected: pane shows only text output and prefix markers

# Criterion 6: Stateless memory load
# Verify engram-agent's responses are contextually relevant
# (agent uses MEMORY_FILES: content, not stale cached memories)
```

**PASS when:** no Agent tool invocations visible; agent responses are contextually relevant.

### Step 7: Scenario E — WAIT: Exits Cleanly (Criteria 8, 10)

Design an intent that conflicts with a known memory file so engram-agent posts WAIT::

```bash
# Post intent that should trigger an objection from loaded memory
engram chat post --from lead --to engram-agent --thread test --type intent \
  --text 'Situation: [describe conflict with known memory]. Behavior: proceed.'

# Expected in chat file:
# type = "wait", from = "engram-agent", to = "lead"

# Verify watch loop is still alive (no exit after WAIT:)
engram agent list | jq -r '.["engram-agent"].state'
# Expected: SILENT (session ended, watch loop active)

# Argument continuation: re-post revised intent
engram chat post --from lead --to engram-agent --thread test --type intent \
  --text 'Situation: [revised plan addressing concern]. Behavior: proceed.'
# Expected: new session fires, engram-agent responds ACK:
```

**PASS when:** WAIT: relayed to chat; session exits; watch loop stays alive (SILENT); revised intent triggers new session; ACK: posted.

### Step 8: Scenario F — Hold Orthogonality (Criterion 9)

```bash
engram hold acquire engram-agent
engram chat post --from lead --to engram-agent --thread test --type intent \
  --text 'Situation: hold test. Behavior: ACK:.'
# Expected: new session fires DESPITE hold — hold does not block dispatch
# Verify ACK: appears in chat file

engram hold release $(engram hold list | jq -r '.[0]["hold-id"]')
```

**PASS when:** engram-agent auto-resumes and responds while on hold.

### Step 9: Scenario G — Binary Restart Limitation (documented non-criterion)

Verify the known limitation: intents posted while binary is down are NOT replayed.

```bash
# Kill binary mid-watch
kill $(pgrep -f 'engram agent run')

# Post intent while binary is dead
engram chat post --from lead --to engram-agent --thread test --type intent \
  --text 'Missed intent — should not be replayed.'

# Restart binary
engram agent run --name engram-agent --state-file ...
# Expected: binary does NOT process the missed intent (starts from EOF)

# Post new intent — this one SHOULD be processed
engram chat post --from lead --to engram-agent --thread test --type intent \
  --text 'Post-restart intent. Behavior: ACK:.'
# Expected: ACK: response arrives
```

**Expected behavior:** Missed intent not replayed — this is correct Phase 5 behavior, not a bug. Phase 6 fix: persist last-processed cursor in AgentRecord state file.

### Step 10: targ check-full

```bash
targ check-full
```

**PASS when:** no new lint errors; coverage thresholds met; all existing tests still green.

---

## Known Limitations

These are intentional Phase 5 behaviors. Do not treat as bugs.

**Intents missed on binary restart.** If `engram agent run` is not running when a `type=intent` is posted to `engram-agent`, that intent will not be processed when the binary restarts. The watch loop starts from current chat file EOF on restart. Phase 6 fix: persist `last-processed-cursor` in AgentRecord (state file). For Phase 5: re-post the intent after restarting the binary.

**Hold release requires manual re-post.** Holds are kill-gates only — they do not block dispatch. If you want to prevent engram-agent from processing a new intent, kill the agent (`engram agent kill engram-agent`) rather than placing a hold. Hold release does not replay any skipped intents.

**WAIT: argument continuation is manual.** After engram-agent posts `WAIT:`, its session ends. The initiating agent must read the concern from chat and re-post a revised `type=intent`. There is no automatic argument continuation in Phase 5. Phase 6 adds `RECENT_INTENTS:` to the resume prompt for cross-invocation context reconstruction.

**No rate limiting on auto-resume.** Phase 5 has no rate limiting. Rapid intent arrival triggers a new session per intent subject to the worker queue limit (max 3 STARTING/ACTIVE). Phase 6 adds: skip auto-resume if >5 new memories written in last 10 minutes.

**Session-id changes between invocations.** Each watch-loop fire is a fresh `claude -p` call. Session-id changes between invocations. This is intentional (stateless model). Phase 6 may add `RECENT_INTENTS:` for argument context, but full session history is not persisted.

---

## Pre-Flight Checklist (run before declaring Phase 5 complete)

Passing unit tests is not sufficient. Run these manually at the binary entry point.

- [ ] `engram agent spawn engram-agent` succeeds without error
- [ ] `READY:` appears in pane output within 5s of spawn
- [ ] `engram agent list` shows `state=ACTIVE` (not STARTING) after `READY:`
- [ ] Post one intent → new session starts; chat file receives non-empty response from `engram-agent`
- [ ] Resume prompt received by agent includes non-zero `CURSOR:` value (inspect pane or add debug logging)
- [ ] Resume prompt received by agent includes non-empty `MEMORY_FILES:` section (at least 1 path if memories exist)
- [ ] Post second intent after `DONE:` → agent auto-resumes without manual restart (binary stays alive)
- [ ] Spawn 4 workers → 4th spawn fails gracefully (non-panic; message contains 'worker queue full')
- [ ] Hold `engram-agent`, post intent → new session starts (hold did not block dispatch)
- [ ] Kill binary, restart, post new intent → new session starts; missed intent NOT replayed
- [ ] `targ check-full` passes with no new issues

---

## Post-Phase-5: Phase 6 Codesign Agenda

Phase 6 is a **rewrite-from-scratch**, not an incremental extension. It must not begin without a dedicated codesign session.

| Item | Scope |
|------|-------|
| Full binary dispatcher | Extends Phase 5 watch loop to route intents to any managed agent (routing table: agent-name → session-id from state file) |
| Thread-aware routing | INTENT[thread=build]: prefix syntax; binary extracts thread name |
| Worker-to-worker addressing | INTENT[to=lead,thread=review]: prefix syntax |
| Mechanical argument enforcement | 3-input argument protocol across turns (requires cross-turn state) |
| RECENT_INTENTS: in resume prompt | Item 1: summaries of recent intents for argument context |
| Rate limiting | Item 4: skip auto-resume if >5 new memories in last 10 min |
| Persistent cursor in state file | last-cursor field on AgentRecord; replay-safe binary restarts |
| Skill rewrite-from-scratch | ~45-line targets for all three skills (~85% reduction from current) |

**Do not write the Phase 6 plan until the Phase 6 codesign session completes.**
