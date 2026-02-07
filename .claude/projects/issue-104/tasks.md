# Tasks - ISSUE-104: Orchestrator as Haiku Teammate

**Project:** issue-104
**Issue:** ISSUE-104
**Created:** 2026-02-07

---

## Simplicity Rationale

This is the minimal viable refactoring to achieve 30x cost savings by moving mechanical orchestration work from opus to haiku. The two-role split (team lead + orchestrator teammate) is imposed by the team ownership constraint: only team owners can spawn teammates, so we need a thin opus layer for spawning and a haiku worker for the step loop.

**Alternatives considered:**
1. **Non-LLM orchestrator (ISSUE-1):** More elegant long-term, but requires external API integration - deferred
2. **Plain text spawn protocol:** Simpler message format, but structured JSON prevents parsing errors and ensures all spawn params are transmitted correctly
3. **No model handshake:** Reduces validation code, but risks running wrong model without early detection

**Why current approach is appropriate:**
- Reuses existing patterns (SendMessage for coordination, Task tool for spawning)
- Adds only necessary components: spawn request relay, shutdown coordination, model handshake
- Preserves existing orchestrator logic (step loop, state management) - just moves it to haiku context
- Clear role boundaries prevent confusion and maintain separation of concerns

---

## Dependency Graph

```
TASK-1 (project SKILL.md split docs)
    |
TASK-2 (orchestrator spawn logic)
    |
    +------ TASK-3 (spawn request protocol) ----+
    |                                           |
    +------ TASK-4 (model handshake)           |
                                                |
                                            TASK-5 (shutdown protocol)
                                                |
                                            TASK-6 (error handling & retry)
                                                |
                                            TASK-7 (state persistence)
                                                |
                                            TASK-8 (resumption logic)
                                                |
                                            TASK-9 (delegation-only enforcement)
                                                |
                                           TASK-10 (integration test)
```

**Parallel opportunities:**
- TASK-3 and TASK-4 can be implemented in parallel after TASK-2
- TASK-6, TASK-7, TASK-9 are independent and can be parallelized

---

### TASK-1: Update project SKILL.md with two-role architecture documentation

**Description:** Split the project SKILL.md documentation to clearly define team lead (opus) and orchestrator teammate (haiku) roles. Document the delegation-only mode for team lead and add orchestrator spawn sequence.

**Status:** Ready

**Acceptance Criteria:**
- [ ] "Two-Role Architecture" section added documenting team lead vs orchestrator responsibilities
- [ ] "Team Lead Mode" section documents delegation-only behavior (no Write/Edit/Bash)
- [ ] "Orchestrator Spawn" section documents startup sequence with TeamCreate + Task spawn
- [ ] Spawn request/confirmation protocol documented
- [ ] Model handshake validation documented
- [ ] SKILL-full.md updated with detailed orchestrator behavior, state persistence, resumption flow

**Files:** `skills/project/SKILL.md`, `skills/project/SKILL-full.md`

**Dependencies:** None

**Traces to:** ARCH-051, ARCH-042, ARCH-050, REQ-016

---

### TASK-2: Implement team lead orchestrator spawn on /project invocation

**Description:** Add logic to team lead to spawn orchestrator teammate immediately after TeamCreate when /project is invoked. Pass project name and issue context in spawn prompt.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Team lead calls TeamCreate with project name on /project invocation
- [ ] Team lead calls Task tool to spawn orchestrator teammate with model="haiku"
- [ ] Spawn prompt includes project name, issue number, and instruction to run step loop
- [ ] Team lead enters idle state after spawn, waiting for orchestrator messages
- [ ] Orchestrator teammate starts with `projctl state init` and enters step loop

**Files:** `skills/project/SKILL.md` (team lead startup logic)

**Dependencies:** TASK-1

**Traces to:** ARCH-048, ARCH-042, REQ-016, REQ-017

---

### TASK-3: Implement spawn request protocol

**Description:** Add orchestrator logic to detect `spawn-producer` and `spawn-qa` actions from `projctl step next` and send structured spawn requests to team lead via SendMessage. Team lead extracts task_params and calls Task tool.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Orchestrator detects `spawn-producer` action from `projctl step next` output
- [ ] Orchestrator detects `spawn-qa` action from `projctl step next` output
- [ ] Orchestrator composes SendMessage with `spawn_request` field containing full task_params JSON
- [ ] Message includes `expected_model`, `action`, `phase` fields
- [ ] Team lead receives spawn request message and extracts task_params
- [ ] Team lead calls Task tool with extracted subagent_type, name, model, prompt, team_name
- [ ] Team lead sends confirmation message after successful spawn

**Files:** `skills/project/SKILL.md` (orchestrator spawn request logic, team lead spawn handling)

**Dependencies:** TASK-2

**Traces to:** ARCH-043, REQ-017, DES-003, DES-004, DES-005

---

### TASK-4: [visual] Implement model handshake validation

**Description:** Add team lead logic to validate that spawned teammate's first message contains the expected model name. On handshake failure, call `projctl step complete --status failed` and send failure message to orchestrator.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Team lead reads spawned teammate's first message after Task tool returns
- [ ] Team lead performs case-insensitive substring match for expected_model in message content
- [ ] On handshake success: team lead sends "spawn-confirmed: {name}" message to orchestrator
- [ ] On handshake failure: team lead calls `projctl step complete --status failed --reported-model "<model>"`
- [ ] On handshake failure: team lead sends "spawn-failed: wrong model" message to orchestrator with details
- [ ] User sees brief status update: "Spawned: {name}" or "Spawn failed: wrong model"

**Files:** `skills/project/SKILL.md` (team lead handshake validation logic)

**Dependencies:** TASK-2

**Traces to:** ARCH-043, REQ-021, DES-006

---

### TASK-5: Implement shutdown protocol

**Description:** Add orchestrator logic to send "all-complete" message to team lead when `projctl step next` returns `all-complete` action. Team lead handles end-of-command sequence, sends shutdown requests to teammates, and calls TeamDelete.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Orchestrator detects `all-complete` action from `projctl step next`
- [ ] Orchestrator sends "all-complete" message to team lead with project summary
- [ ] Team lead receives all-complete message and runs end-of-command sequence
- [ ] Team lead sends shutdown_request to all active teammates (including orchestrator)
- [ ] Team lead waits for shutdown confirmations from teammates
- [ ] Team lead calls TeamDelete after all confirmations received
- [ ] Team lead reports completion status to user

**Files:** `skills/project/SKILL.md` (orchestrator completion logic, team lead shutdown handling)

**Dependencies:** TASK-3

**Traces to:** ARCH-044, REQ-018, DES-007

---

### TASK-6: Implement error handling with retry-backoff

**Description:** Add orchestrator retry logic with exponential backoff (1s, 2s, 4s) for failed operations. After 3 attempts, escalate to team lead with error details.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Orchestrator wraps `projctl step next` calls with retry logic
- [ ] Orchestrator wraps `projctl step complete` calls with retry logic
- [ ] Orchestrator wraps spawn confirmation waits with timeout retry
- [ ] Backoff delays: 1s after attempt 1, 2s after attempt 2, 4s after attempt 3
- [ ] After 3 failed attempts, orchestrator sends error message to team lead
- [ ] Error message includes action, phase, error output, and retry history
- [ ] Team lead escalates errors to user via AskUserQuestion
- [ ] Orchestrator logs retry attempts for debugging

**Files:** `skills/project/SKILL.md` (orchestrator retry logic)

**Dependencies:** TASK-3

**Traces to:** ARCH-046, REQ-019, REQ-023, DES-006

---

### TASK-7: Implement state persistence ownership

**Description:** Document and enforce that orchestrator teammate owns all state persistence via `projctl state` commands. Team lead never touches state files.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Orchestrator calls `projctl state init` on first run
- [ ] Orchestrator calls `projctl state set --workflow <type>` after workflow classification
- [ ] Orchestrator calls `projctl state set` after each `projctl step complete` to persist progress
- [ ] State includes: current phase, sub-phase, workflow type, active issue, pair loop iteration
- [ ] Team lead does not read or write state files (documented prohibition)
- [ ] State file location: `.claude/projects/<project-name>/state.toml`
- [ ] Atomic state writes (temp file + rename) ensure no partial states

**Files:** `skills/project/SKILL.md` (orchestrator state management, team lead prohibition docs)

**Dependencies:** TASK-3

**Traces to:** ARCH-045, REQ-020, REQ-022

---

### TASK-8: Implement resumption after orchestrator termination

**Description:** Add team lead logic to detect orchestrator termination and respawn with state resumption. Orchestrator reads state on startup and resumes from last saved phase.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Team lead detects orchestrator termination (no response, error signal)
- [ ] Team lead checks for state file existence (`.claude/projects/<name>/state.toml`)
- [ ] If state exists: team lead respawns orchestrator with same spawn params
- [ ] Respawned orchestrator reads state via `projctl state get --format json` on startup
- [ ] Orchestrator skips init if state.phase is not empty, goes straight to step loop
- [ ] Orchestrator resumes from last saved phase without repeating completed work
- [ ] If state missing: team lead reports "Cannot resume - no state file" to user

**Files:** `skills/project/SKILL.md` (team lead resumption logic, orchestrator startup logic)

**Dependencies:** TASK-7

**Traces to:** ARCH-049, REQ-020, REQ-022

---

### TASK-9: Implement delegation-only enforcement for team lead

**Description:** Document and enforce that team lead never calls Write, Edit, NotebookEdit, or Bash tools. Team lead delegates all file operations and command execution to spawned teammates.

**Status:** Ready

**Acceptance Criteria:**
- [ ] SKILL.md "DO NOT" column lists Write, Edit, NotebookEdit, Bash as prohibited for team lead
- [ ] Team lead allowed actions documented: TeamCreate, TeamDelete, Task, SendMessage, AskUserQuestion, Read
- [ ] Example violation prevention scenarios documented in SKILL.md
- [ ] Team lead self-monitors during execution and spawns teammates instead of editing directly
- [ ] Delegation discipline enforced consistently throughout orchestration

**Files:** `skills/project/SKILL.md` (team lead prohibition documentation)

**Dependencies:** TASK-3

**Traces to:** ARCH-050, REQ-016, ARCH-042

---

### TASK-10: [visual] Integration test: full project flow with two-role split

**Description:** Test end-to-end flow: /project invocation → team lead spawns orchestrator → orchestrator runs step loop → spawn requests → model handshake → shutdown → TeamDelete. Verify cost savings and correct behavior.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Start with /project command for a small test issue
- [ ] Verify team lead spawns orchestrator teammate with model="haiku"
- [ ] Verify model handshake passes (orchestrator's first message contains "haiku")
- [ ] Verify orchestrator sends spawn request for pm-interview-producer
- [ ] Verify team lead spawns pm-interview-producer and confirms to orchestrator
- [ ] Verify orchestrator completes at least one phase transition
- [ ] Verify orchestrator sends "all-complete" message when done
- [ ] Verify team lead handles shutdown and calls TeamDelete
- [ ] Compare token usage: haiku cost << opus cost for orchestration work
- [ ] User experience matches DES-001 and DES-002 (silent coordination, brief updates)

**Files:** Test project invocation (no code changes, validation only)

**Dependencies:** TASK-2, TASK-3, TASK-4, TASK-5, TASK-6, TASK-8, TASK-9

**Traces to:** ARCH-042, ARCH-047, ARCH-048, DES-001, DES-002, REQ-016

---

## Coverage Summary

**Requirements Coverage:**
- REQ-016: TASK-1, TASK-2, TASK-9, TASK-10
- REQ-017: TASK-2, TASK-3, TASK-4
- REQ-018: TASK-5
- REQ-019: TASK-6
- REQ-020: TASK-7, TASK-8
- REQ-021: TASK-3, TASK-4
- REQ-022: TASK-7, TASK-8
- REQ-023: TASK-6

**Architecture Coverage:**
- ARCH-042: TASK-1, TASK-2, TASK-9, TASK-10
- ARCH-043: TASK-3, TASK-4
- ARCH-044: TASK-5
- ARCH-045: TASK-7, TASK-8
- ARCH-046: TASK-6
- ARCH-047: TASK-10
- ARCH-048: TASK-2, TASK-10
- ARCH-049: TASK-8
- ARCH-050: TASK-1, TASK-9
- ARCH-051: TASK-1

**Design Coverage:**
- DES-001: TASK-10
- DES-002: TASK-10
- DES-003: TASK-3
- DES-004: TASK-3
- DES-005: TASK-3
- DES-006: TASK-4, TASK-6
- DES-007: TASK-5

**Total Tasks:** 10
**Visual Tasks:** 2 (TASK-4, TASK-10)
**Parallel Opportunities:** 3 groups (TASK-3/TASK-4 after TASK-2; TASK-6/TASK-7/TASK-9 independent)

---
