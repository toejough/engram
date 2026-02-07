# Design - ISSUE-104: Orchestrator as Haiku Teammate

**Status:** Draft
**Created:** 2026-02-07
**Issue:** ISSUE-104

---

## Overview

This design splits the `/project` orchestrator into two distinct roles: a team lead (opus) that handles team management and user interaction, and an orchestrator teammate (haiku) that runs the mechanical state machine loop. This separation optimizes model usage by reserving opus for high-value coordination while haiku handles the routine step execution.

---

## User Experience

### DES-001: Silent Orchestration with Brief Updates

The orchestrator-team lead coordination happens behind the scenes with brief user-visible status updates. Users see high-level progress notifications without being overwhelmed by internal message traffic.

**User sees:**
- "Orchestrator requesting tdd-red-producer spawn"
- "Spawned: tdd-red-producer"
- "Retry 2/3: spawning qa teammate"

**User does not see:**
- Raw JSON message payloads
- Internal acknowledgments between orchestrator and team lead
- Low-level state transitions

**Traces to:** ISSUE-104 acceptance criteria (implicit user experience requirement)

### DES-002: Transparent Team Structure

From the user's perspective, the team structure remains simple:
- User interacts with `/project`
- `/project` spawns team lead (opus)
- Team lead spawns orchestrator (haiku) + other teammates as needed
- Work progresses with status updates
- Team lead reports completion and cleans up

The two-role split is an implementation detail. Users continue to invoke `/project` exactly as before.

**Traces to:** ISSUE-104 acceptance criteria

---

## Interaction Patterns

### DES-003: Spawn Request Flow

```
Orchestrator (haiku):
  1. projctl step next returns spawn-producer action
  2. Compose spawn request message
  3. SendMessage to team-lead with type="message"
  4. Wait for spawn confirmation

Team Lead (opus):
  5. Receive spawn request message
  6. Extract task_params from message
  7. Call Task tool with task_params
  8. On success: SendMessage confirmation to orchestrator
  9. On failure: Retry up to 3 times with exponential backoff
  10. After 3 failures: Escalate to user
```

**Traces to:** ISSUE-104 acceptance criteria: "Orchestrator sends spawn requests to team lead via SendMessage", "Team lead spawns teammates on behalf of orchestrator and confirms"

### DES-004: Spawn Request Message Format

Orchestrator sends spawn requests as structured messages containing the full task_params JSON:

```json
{
  "type": "spawn-request",
  "summary": "Request spawn: tdd-red-producer",
  "task_params": {
    "subagent_type": "general-purpose",
    "name": "tdd-red-producer",
    "model": "opus",
    "team_name": "issue-104",
    "prompt": "..."
  }
}
```

Team lead extracts `task_params` and passes it directly to the Task tool without modification.

**Traces to:** ISSUE-104 acceptance criteria: "Orchestrator sends spawn requests to team lead via SendMessage (includes full task_params)"

### DES-005: Spawn Confirmation Message

Team lead confirms successful spawn by sending a message back to orchestrator:

```json
{
  "type": "spawn-confirmation",
  "summary": "Spawned: tdd-red-producer",
  "teammate_name": "tdd-red-producer"
}
```

Orchestrator adds the teammate_name to its internal registry.

**Traces to:** ISSUE-104 acceptance criteria: "Team lead spawns teammates on behalf of orchestrator and confirms"

### DES-006: Spawn Failure Handling

When Task tool fails to spawn a teammate:

1. Team lead catches the error
2. Retry with exponential backoff (1s, 2s, 4s intervals)
3. Show brief status to user: "Retry N/3: spawning {teammate}"
4. After 3 failures, escalate to user via AskUserQuestion:
   - "Failed to spawn {teammate} after 3 attempts. Continue?"
   - Options: Retry again, Skip this step, Abort project

**Traces to:** User design decision: "Team lead auto-retries and escalates to user after 3 failures"

### DES-007: Individual Shutdown Requests

When orchestrator completes all work (receives `all-complete` from `projctl step next`):

1. For each teammate in registry, send individual shutdown request:
   ```json
   {
     "type": "shutdown_request",
     "recipient": "{teammate_name}",
     "content": "Project complete, shutting down team"
   }
   ```
2. Wait for each teammate's shutdown response
3. Teammates can approve (exit) or reject (need more time)
4. After all shutdowns confirmed, orchestrator notifies team lead
5. Team lead runs end-of-command sequence and TeamDelete

**Traces to:** ISSUE-104 acceptance criteria: "Orchestrator sends shutdown requests to team lead", "Team lead handles end-of-command sequence after orchestrator reports all-complete"

---

## Data & State

### DES-008: Orchestrator Teammate Registry

Orchestrator maintains an in-memory registry of spawned teammates:

```javascript
{
  "tdd-red-producer": { spawned_at: "2026-02-07T01:00:00Z", status: "active" },
  "qa": { spawned_at: "2026-02-07T01:05:00Z", status: "active" }
}
```

**Properties:**
- Stored in memory (not persisted to disk)
- Keys are teammate names (not UUIDs)
- Used for tracking who to shut down at project end
- Rebuilt from `projctl state` if orchestrator crashes/restarts

**Traces to:** User design decision: "Orchestrator tracks spawned teammates"

### DES-009: Team Lead Coordination State

Team lead does NOT maintain separate teammate registry. It relies on:
- TeamCreate/TeamDelete for team lifecycle
- SendMessage for orchestrator coordination
- Task tool for actual spawning

Team lead is stateless between orchestrator requests - all state lives in either the orchestrator's memory or the team config file (`~/.claude/teams/{team-name}/config.json`).

**Traces to:** ISSUE-104 design goal: "Team lead stays thin — only does what requires team ownership (spawn/shutdown)"

---

## Communication Protocol

### DES-010: All Coordination via SendMessage

All orchestrator-team lead communication uses the SendMessage tool with structured JSON payloads. No side channels (files, environment variables, etc.).

**Message types:**
- `spawn-request`: Orchestrator → Team Lead
- `spawn-confirmation`: Team Lead → Orchestrator
- `spawn-failure`: Team Lead → Orchestrator (after retries exhausted)
- `shutdown_request`: Orchestrator → Teammates (via team lead relay)
- `all-complete`: Orchestrator → Team Lead

**Traces to:** User design decision: "All orchestrator-team lead messages use SendMessage tool"

### DES-011: Brief Status Updates to User

Team lead outputs brief status messages to user during coordination:

```
Orchestrator requesting tdd-red-producer spawn...
Spawned: tdd-red-producer
Orchestrator requesting qa spawn...
Retry 1/3: spawning qa teammate
Spawned: qa
```

These appear in the main conversation stream as regular text output, not tool results.

**Traces to:** User design decision: "Brief status updates" for user visibility

---

## Workflows

### DES-012: Initial Project Setup

```
User invokes /project
  ↓
Team Lead (opus):
  1. Parse user intent
  2. TeamCreate(name="issue-{N}")
  3. Task(spawn orchestrator teammate):
     - subagent_type: "general-purpose"
     - name: "orchestrator"
     - model: "haiku"
     - team_name: "issue-{N}"
     - prompt: Load project SKILL.md
  4. Wait for messages from orchestrator
```

**Traces to:** ISSUE-104 acceptance criteria: "Team lead spawns a haiku orchestrator teammate on `/project` invocation"

### DES-013: Orchestrator Main Loop

```
Orchestrator (haiku):
  1. projctl state init / set workflow
  2. Loop until done:
     a. projctl step next → get action JSON
     b. Parse action type:
        - spawn-producer: SendMessage spawn request to team-lead
        - spawn-qa: SendMessage spawn request to team-lead
        - commit: Handle directly (no spawn)
        - transition: Handle directly
        - all-complete: Send shutdown requests, notify team-lead
     c. Wait for confirmations
     d. projctl step complete --result (success/failure)
  3. Exit when all-complete handled
```

**Traces to:** ISSUE-104 acceptance criteria: "Orchestrator runs the full `projctl step next/complete` loop"

### DES-014: End-of-Project Sequence

```
Orchestrator completes final step:
  1. projctl step next returns {"action": "all-complete"}
  2. For each teammate in registry:
     - SendMessage shutdown_request
     - Wait for confirmation
  3. SendMessage to team-lead: {"type": "all-complete"}

Team Lead receives all-complete:
  4. Run final reporting/summary (if any)
  5. TeamDelete()
  6. Report success to user
```

**Traces to:** ISSUE-104 acceptance criteria: "Team lead handles end-of-command sequence after orchestrator reports all-complete"

---

## Design Rationale

### DES-015: Why Two Roles?

**Problem:** Opus runs the entire orchestration loop today, including mechanical JSON parsing and routing. This is expensive and doesn't leverage haiku's capabilities.

**Solution:** Split into:
- **Team lead (opus):** Thin coordination layer. Only does what requires team ownership (spawn/shutdown via Task tool, TeamDelete, user escalation).
- **Orchestrator (haiku):** Runs the mechanical loop (projctl commands, JSON parsing, routing). Sufficient for this deterministic work.

**Benefit:**
- Reduces opus usage by ~80% (only spawns, not every step)
- Preserves opus context for user interaction
- Haiku cost is 20x cheaper than opus for routine work

**Traces to:** ISSUE-104 problem statement: "This wastes an expensive model on mechanical work"

### DES-016: Why In-Memory Registry?

**Alternative considered:** Persist teammate registry to disk.

**Decision:** Keep in memory because:
1. Registry can be rebuilt from team config file if orchestrator crashes
2. Avoids file I/O overhead on every spawn/shutdown
3. Registry lifetime = orchestrator lifetime = project session
4. Simpler implementation (no file locking, no sync issues)

**Trade-off:** If orchestrator crashes mid-project, team lead must restart it and rebuild registry from team config.

**Traces to:** User design decision: "Teammate registry stored in orchestrator memory only"

---

## Open Questions

None. All design decisions finalized.

---

## Summary

This design establishes a clear separation between high-level coordination (team lead, opus) and mechanical execution (orchestrator, haiku). The orchestrator handles the deterministic state machine loop while the team lead manages team lifecycle and user interaction. Communication happens via structured SendMessage payloads with brief user-visible status updates. The orchestrator tracks spawned teammates in memory and coordinates individual shutdowns at project end.
