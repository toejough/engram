# Design: engram-tmux-lead Phase 5 Compatibility (#543)

## Problem

The Phase 5 stateless model changes engram-agent's lifecycle in ways the lead skill doesn't model. Four conflicts result.

---

## The Four Conflicts

### Conflict 1: Spurious SILENT Nudge

**Lead skill (§3.2):** When an agent has sent no chat messages for longer than `silence_threshold` (6 min for engram-agent), the lead transitions it to SILENT and sends a nudge.

**Phase 5 reality:** The binary writes `state = "SILENT"` to the state file when a watch-loop invocation ends (session completes or WAIT: exits). Between intents, engram-agent is legitimately SILENT for hours — this is its normal resting state. The lead's nudge fires against a healthy, watch-loop-idle agent.

---

### Conflict 2: False Heartbeat Threshold

**Lead skill (§6.2):** "If engram-agent missed heartbeat (>6 min since last), nudge immediately."

**Phase 5 reality:** A stateless engram-agent has no running process between invocations that could send heartbeats. The 6-minute threshold was calibrated for a persistent always-running agent. It fires false positives constantly in the stateless model.

---

### Conflict 3: Missing Binary-Managed ACTIVE→SILENT Transition

**Lead skill (§3 state machine):**
```
ACTIVE → DONE   (agent posts done, no holds)
ACTIVE → SILENT (no message for silence_threshold)
```

**Phase 5 reality:** The binary writes SILENT directly (via `watchAndResume`) after each session ends, before entering the file-watch loop. The ACTIVE→SILENT transition is driven by the binary, not by a timeout. This transition is absent from the lead's state machine.

---

### Conflict 4: Indistinguishable SILENT States

**Lead skill:** SILENT → nudge → (no response) → DEAD.

**Phase 5 reality:** Two distinct SILENT states exist:
- **Binary-managed SILENT:** `engram agent run` process is alive, watch loop is blocking on intent.
- **Truly-dead SILENT:** Process crashed or pane gone; no recovery possible without respawn.

Both appear identical to the lead: no chat messages, `state = "SILENT"` in the state file. The current skill cannot distinguish them.

---

## Root Cause

The lead skill was designed for a persistent stateful engram-agent. Phase 5 replaces that model with a stateless per-invocation worker where SILENT is a healthy resting state, not an anomaly.

---

## Design

### Binary Change: `last-silent-at` Timestamp

Add a `LastSilentAt time.Time` field to `AgentRecord`. The binary writes it atomically with `state = "SILENT"` every time `watchAndResume` enters the watch loop. This gives the lead a reliable audit trail: "the binary intentionally transitioned this agent to SILENT at time T."

**Why not pane-existence-only?** Pane existence requires a tmux call and is fragile (shell prompts vary, `remain-on-exit` differs by setup). A state-file field is cheaper and gives diagnostic visibility. Both checks are used together.

**Why not a new "WATCHING" state?** "WATCHING" would require updating every consumer of the state file (agent list tests, reconstruction logic, skill documentation). The binary-managed SILENT is semantically correct — the agent is silent and waiting. Adding a timestamp is a narrower, additive change.

### New function: `writeAgentSilentState`

Replace `writeAgentState(stateFilePath, agentName, "SILENT")` in `watchAndResume` with a new `writeAgentSilentState` that writes `state = "SILENT"` AND `last-silent-at = <now>` in a single read-modify-write operation.

```go
// writeAgentSilentState writes state=SILENT and last-silent-at atomically.
func writeAgentSilentState(stateFilePath, agentName string) error {
    return readModifyWriteStateFile(stateFilePath, func(sf agentpkg.StateFile) agentpkg.StateFile {
        for i, rec := range sf.Agents {
            if rec.Name == agentName {
                sf.Agents[i].State = "SILENT"
                sf.Agents[i].LastSilentAt = time.Now().UTC()
                return sf
            }
        }
        return sf
    })
}
```

### AgentRecord field addition

```go
type AgentRecord struct {
    // ... existing fields ...
    LastSilentAt  time.Time `json:"last-silent-at,omitzero"  toml:"last-silent-at,omitzero"`
}
```

`omitzero` matches the `LastResumedAt` convention — zero timestamps are omitted from TOML and JSON output.

---

## Lead Skill Changes

### §3 State Machine

Add the binary-managed transition (BEFORE the timeout-driven one):

```
ACTIVE ──(binary writes SILENT on session end)──> SILENT  [binary-managed]
ACTIVE ──(no message for silence_threshold)──> SILENT      [timeout-driven]
```

Annotate: engram-agent only experiences the binary-managed transition in Phase 5. Task agents only experience the timeout-driven one.

### §3.1 SILENT Definition Update

Add a note that SILENT has two sub-types for engram-agent:
- **Binary-managed:** `last-silent-at` is set in state file AND pane exists. Healthy.
- **Timeout-driven / truly dead:** No `last-silent-at`, OR pane is gone. Requires nudge or respawn.

### §3.2 Nudge Logic for engram-agent

Replace the current `SILENT → nudge` rule for engram-agent with a pane-existence + `last-silent-at` check:

```
For engram-agent in SILENT state:
  AGENT_INFO=$(engram agent list | jq -r 'select(.name=="engram-agent")')
  PANE_ID=$(echo "$AGENT_INFO" | jq -r '.["pane-id"]')
  LAST_SILENT=$(echo "$AGENT_INFO" | jq -r '.["last-silent-at"]')

  if tmux list-panes -F '#{pane_id}' | grep -q "$PANE_ID"; then
    # Pane alive → binary watch loop healthy → skip nudge
    continue
  else
    # Pane gone → truly dead → respawn immediately (no nudge cycle)
    transition to DEAD → respawn
  fi
```

`last-silent-at` is used for diagnostic logging ("healthy SILENT since T") but pane existence is the liveness gate.

### §6.2 Health Check Update

Remove "If engram-agent missed heartbeat (>6 min since last), nudge immediately."

Replace with:

```
For engram-agent: health = pane exists.
  - If pane exists → healthy (regardless of SILENT duration)
  - If pane gone → DEAD → respawn
For task agents: health = no message for silence_threshold → SILENT → nudge
```

---

## Test Strategy

### Phase 1 (RED) — Property Tests

Three tests, all failing because `LastSilentAt` doesn't exist in `AgentRecord`:

**1. `TestAgentRecord_LastSilentAt_RoundTrips`** (`internal/agent/agent_test.go`)
- Creates `AgentRecord` with `LastSilentAt` set
- Marshal → unmarshal → verify field preserved
- Fails: field doesn't exist

**2. `TestOuterWatchLoop_WritesSilentAtWithSilentState`** (`internal/cli/cli_test.go`)
- Runs `ExportRunConversationLoopWith` with fake DONE: claude binary and watchForIntent that cancels ctx
- After call, reads state file
- Asserts state file contains BOTH `state = "SILENT"` AND `last-silent-at =` (via `ContainSubstring`)
- Fails: `last-silent-at` never written by current code

**3. `TestRunAgentList_LastSilentAtIncludedInOutput`** (`internal/cli/cli_test.go`)
- Writes state file with agent record that has `LastSilentAt` set
- Runs `engram agent list`
- Parses NDJSON output, asserts `last-silent-at` key present and non-zero
- Fails: field doesn't exist in struct

### Phase 2 (GREEN) — Implementation

1. Add `LastSilentAt` to `AgentRecord` in `internal/agent/agent.go`
2. Add `writeAgentSilentState` in `internal/cli/cli_agent.go`
3. Replace `writeAgentState(..., "SILENT")` call in `watchAndResume` with `writeAgentSilentState`

### Phase 3 — Simplify + Skill Update

1. Run `simplify` skill on changed code
2. Run `targ check-full`
3. Update `skills/engram-tmux-lead/SKILL.md` with the §3, §3.1, §3.2, and §6.2 changes described above
4. Commit

---

## Acceptance Criteria

- [ ] `TestAgentRecord_LastSilentAt_RoundTrips` passes
- [ ] `TestOuterWatchLoop_WritesSilentAtWithSilentState` passes (state file has both SILENT and last-silent-at)
- [ ] `TestRunAgentList_LastSilentAtIncludedInOutput` passes
- [ ] Existing `TestOuterWatchLoop_WriteStateSilentAfterSession` still passes (no regression)
- [ ] `targ check-full` passes with zero issues
- [ ] Lead skill §3 state machine documents binary-managed ACTIVE→SILENT transition
- [ ] Lead skill §3.2 no longer nudges engram-agent for SILENT; uses pane-existence check
- [ ] Lead skill §6.2 removes heartbeat concept; replaces with pane-existence check

---

## Files Changed

```
internal/agent/
  agent.go          — add LastSilentAt field to AgentRecord
  agent_test.go     — add TestAgentRecord_LastSilentAt_RoundTrips

internal/cli/
  cli_agent.go      — add writeAgentSilentState; update watchAndResume
  cli_test.go       — add 2 new tests

skills/engram-tmux-lead/
  SKILL.md          — update §3, §3.1, §3.2, §6.2
```

No new dependencies. No export_test.go changes needed (existing `ExportRunConversationLoopWith` is sufficient).
