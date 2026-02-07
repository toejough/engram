# Retrospective - ISSUE-104: Orchestrator as Haiku Teammate

**Project:** issue-104
**Issue:** ISSUE-104
**Date:** 2026-02-07
**Duration:** Single session (~9 hours based on commit timestamps)

---

## Project Summary

### Objective

Split the `/project` orchestrator into two distinct roles to optimize model usage:
- **Team lead (opus):** Thin coordination layer handling spawn/shutdown, user escalation
- **Orchestrator (haiku):** Mechanical step loop executing projctl commands

### Deliverables Produced

1. **Documentation**
   - Two-role architecture in project SKILL.md
   - Spawn request protocol specification
   - Model handshake validation flow
   - State persistence ownership model
   - README updated with architecture overview

2. **Implementation** (TASK-1 through TASK-7 completed)
   - TASK-1: SKILL.md documentation split ✅
   - TASK-2: Orchestrator spawn logic ✅
   - TASK-3: Spawn request protocol ✅
   - TASK-4: Model handshake validation ✅
   - TASK-5: Shutdown protocol ✅
   - TASK-6: Error handling with retry-backoff ✅
   - TASK-7: State persistence ownership ✅

3. **Testing**
   - TDD red-green-refactor discipline for all tasks
   - Go tests and shell tests for documentation validation
   - Integration test planning (TASK-10)

### Team Structure

- **Team lead (opus):** User-facing, spawned orchestrator and teammates
- **Orchestrator (haiku):** [Inferred not yet implemented - project used traditional approach]
- **Producer agents:** Spawned for each phase (pm-interview, tdd-red, etc.)
- **QA agents:** Spawned for quality validation

### Key Metrics

- **Commits:** 17 commits to ISSUE-104
- **Tasks completed:** 7 of 10 tasks (70%)
- **Cost optimization target:** ~30x savings by moving mechanical work to haiku
- **Issues created:** 5 follow-up issues (ISSUE-137-141)

---

## What Went Well (Successes)

### S-1: Clear Architecture Documentation

**Area:** Requirements & Design Phase

**Description:** The two-role architecture was thoroughly documented before implementation with:
- 16 design decisions (DES-001 through DES-016)
- 10 architecture components (ARCH-042 through ARCH-051)
- 8 requirements (REQ-016 through REQ-023)

This comprehensive upfront design led to smooth implementation of TASK-1 through TASK-7 with minimal rework.

**Evidence:**
- design.md contains detailed interaction patterns, data models, and rationale
- tasks.md shows clear traceability from requirements to tasks
- Commits show linear progression through task sequence

---

### S-2: TDD Discipline Maintained Throughout

**Area:** Implementation

**Description:** All implemented tasks (TASK-1 through TASK-7) followed strict red-green-refactor discipline:
- Tests written first (red phase)
- Minimal implementation to pass tests (green phase)
- Documentation expanded/refactored as needed

**Evidence:**
- Git log shows test commits before implementation commits for each task
- task-6-tdd-red-summary.md documents 19 failing tests before green phase
- Commits like "test(project): add failing tests for TASK-6" precede "feat(project): implement retry-backoff documentation"

---

### S-3: SendMessage Protocol Worked Well

**Area:** Architecture

**Description:** Using SendMessage for orchestrator-team lead coordination proved effective:
- Structured JSON payloads prevent parsing errors
- Clear message types (spawn-request, spawn-confirmation, spawn-failure)
- Model handshake validation catches wrong model early

**Evidence:**
- DES-003, DES-004, DES-005 specify protocol cleanly
- TASK-3 and TASK-4 implemented without reported issues
- No evidence of protocol failures in retro-notes

---

### S-4: Model Handshake Validation Prevents Silent Failures

**Area:** Quality & Reliability

**Description:** TASK-4 implemented validation that spawned teammates report their model name in first message. This catches model mismatches immediately rather than discovering them after wasted work.

**Evidence:**
- Commit "feat(ISSUE-104): add handshake confirmation/failure messaging (TASK-4)"
- SKILL.md spawn-producer and spawn-qa handlers include explicit handshake validation steps
- Protocol specifies both success path (spawn-confirmed) and failure path (spawn-failed with reported model)

---

### S-5: Atomic State Writes for Crash Safety

**Area:** Reliability

**Description:** TASK-7 implemented atomic state persistence using temp file + rename pattern, ensuring no partial/corrupt state files if orchestrator crashes mid-write.

**Evidence:**
- Commit "feat(project): add atomic state writes test and docs for TASK-7"
- DES-008 and ARCH-045 document state persistence ownership

---

## What Could Improve (Challenges)

### C-1: TaskList Tool Not Used Until Prompted

**Area:** Team Coordination

**Description:** Despite system reminders, the team lead did not create TaskList entries from tasks.md during PM, design, architect, and breakdown phases. User had no live dashboard of progress until explicitly asking "what happened to using the claude task tool?"

**Impact:**
- Reduced visibility into progress for user
- No structured tracking during early phases
- Wasted opportunity for parallel work coordination

**Root Cause:** SKILL.md control loop doesn't explicitly include "create/update TaskList entries" as a required step. It's implied by Looper Pattern section but not enforced.

**Related:** Retro-notes O-1

---

### C-2: Redundant Commit-Red QA Phases Waste Tokens

**Area:** State Machine Efficiency

**Description:** State machine has both a `commit` action within tdd-red phase AND separate `commit-red` + `commit-red-qa` phases. This results in:
1. Team lead commits during tdd-red phase
2. commit-red phase spawns commit-producer (nothing to commit)
3. QA validates the commit
4. Another commit attempt
5. commit-red-qa spawns another QA

Total: ~4 redundant agent spawns (2 haiku producers, 2 haiku QAs) for work already completed.

**Impact:**
- Wasted tokens on empty commits
- Increased latency (2 extra spawn cycles)
- Confusing user experience (repeated QA for same commit)

**Root Cause:** State machine double-counts commit work - both inline and as separate phase.

**Related:** Retro-notes O-2

---

### C-3: TDD Red Phase Not Scoped to Specific Task

**Area:** Producer Contract Clarity

**Description:** When `projctl step next` returned tdd-red phase, it included `current_task = ""`. The tdd-red-producer chose TASK-1 arbitrarily, but the state machine didn't specify which task to work on.

**Impact:**
- Unclear contract between state machine and TDD producers
- Producers must infer task scope
- Potential for wrong task selection

**Root Cause:** State machine doesn't propagate task context to phase entry.

**Related:** Retro-notes O-3

---

### C-4: Project Scope Creep: Only 70% Complete

**Area:** Project Management

**Description:** Project defined 10 tasks but only completed 7 (TASK-1 through TASK-7). Remaining tasks:
- TASK-8: Resumption after orchestrator termination
- TASK-9: Delegation-only enforcement for team lead
- TASK-10: Integration test

**Impact:**
- Two-role architecture documented but not fully implemented
- No end-to-end validation of the new architecture
- Uncertainty about whether design works in practice

**Root Cause:** [Unclear - session may have ended before completion, or tasks deferred]

---

## Process Improvement Recommendations

### R-1: Add Explicit TaskList Creation Step to Project Control Loop

**Priority:** High

**Action:** Update project SKILL.md startup sequence to require TaskList creation:
```
1. TeamCreate(team_name: "<project-name>")
2. Load tasks.md and create TaskList entries for all defined tasks
3. Spawn orchestrator
4. Enter idle state
```

**Rationale:** Addresses C-1. Making TaskList creation explicit in the control loop ensures it happens consistently, not only when user asks.

**Measurable Impact:**
- User sees task dashboard from project start
- No need to manually prompt for task tracking
- Enables better parallelization decisions based on task dependencies

**Related Challenges:** C-1

---

### R-2: Investigate Collapsing Redundant Commit QA Phases

**Priority:** High

**Action:** Audit whether commit-red and commit-red-qa phases provide value beyond the commit already performed during tdd-red. If not, collapse them or skip when commit already exists.

**Rationale:** Addresses C-2. ~4 redundant agent spawns per red-commit cycle add cost and latency without benefit.

**Measurable Impact:**
- Reduce haiku spawns by 2-4 per TDD cycle
- Faster project completion (fewer spawn cycles)
- Clearer user experience (no duplicate QA messages)

**Implementation Options:**
1. Remove commit-red/commit-red-qa phases entirely
2. Add state check: skip if commit already exists for current phase
3. Merge commit and commit QA into single phase

**Related Challenges:** C-2

---

### R-3: Propagate Task Context to TDD Phase Entry

**Priority:** Medium

**Action:** Update state machine to include `current_task` field when entering tdd-red, tdd-green, tdd-refactor phases. TDD producers should validate task is specified and reject if empty.

**Rationale:** Addresses C-3. Explicit task scoping removes ambiguity about which task the TDD producer should work on.

**Measurable Impact:**
- TDD producers receive clear task assignment
- No guessing about which task to test
- Better traceability from phase to task

**Implementation:**
- State machine sets `current_task` during phase transition
- `projctl step next` includes `current_task` in output JSON
- TDD producers verify `current_task != ""` and fail if missing

**Related Challenges:** C-3

---

### R-4: Establish "Definition of Done" Checkpoint Before Retrospective

**Priority:** Medium

**Action:** Add explicit completion check before entering retrospective:
1. Read tasks.md
2. Count completed vs total tasks
3. If incomplete:
   - Report percentage complete to user
   - Ask: "Continue with remaining tasks or proceed to retrospective?"

**Rationale:** Addresses C-4. Projects should not enter retrospective with significant incomplete work unless user explicitly approves.

**Measurable Impact:**
- Reduce scope creep (partial deliveries)
- User awareness of completion status
- Opportunity to finish vs defer decision made explicitly

**Related Challenges:** C-4

---

### R-5: Add Retro-Notes Discipline to Project Lifecycle

**Priority:** Low

**Action:** Update project SKILL.md to include retro-notes.md creation at project start. Throughout execution, producers and QA append observations to retro-notes when they encounter friction, surprises, or repeated patterns.

**Rationale:** Retro-notes captured valuable observations (O-1, O-2, O-3) that informed this retrospective. Making this a standard practice ensures insights aren't lost.

**Measurable Impact:**
- Richer retrospectives with concrete examples
- Real-time capture prevents forgetting issues
- Patterns visible during execution, not just in hindsight

**Implementation:**
- Add retro-notes.md to SKILL.md artifacts
- Teach producers/QA to append observations when they see issues
- Retro-producer reads retro-notes as primary input

---

## Open Questions

### Q-1: Should Orchestrator Run as Haiku Agent or External Process?

**Context:** ISSUE-104 designed the orchestrator as a haiku teammate spawned by team lead. However, ISSUE-1 proposes a deterministic non-LLM orchestrator (`projctl orchestrate`) that only invokes LLM for skill work.

**Trade-offs:**
- **Haiku teammate:** Simpler to implement (reuses Task tool), but still LLM cost for mechanical work
- **External process:** Eliminates LLM cost for control loop, but requires external API access or claude CLI

**Impact:** Long-term architecture decision affecting cost and reliability.

**Recommendation:** Implement ISSUE-104's haiku approach first (short-term win), then migrate to ISSUE-1's external orchestrator (medium-term optimization).

---

### Q-2: What Do ISSUE-137 through ISSUE-141 Cover?

**Context:** Spawn prompt mentions "Issues filed during this session: ISSUE-137, 138, 139, 140, 141" but details aren't in retro-notes or visible artifacts.

**Need to know:**
- Are they blockers for TASK-8, TASK-9, TASK-10?
- Are they follow-up improvements?
- Should they be linked in retrospective recommendations?

**Action Required:** Query team-lead or check issue tracker for these issue details.

---

### Q-3: Why Were TASK-8, TASK-9, TASK-10 Not Completed?

**Context:** Project defined 10 tasks but stopped after TASK-7. No explicit decision or blocker documented.

**Possible reasons:**
1. Session ended before completion
2. Blockers encountered (undocumented)
3. Deliberate scope reduction (user decision)
4. Integration test (TASK-10) deferred pending TASK-8/TASK-9

**Impact:** Uncertainty about two-role architecture readiness for production use.

**Action Required:** Clarify with user whether remaining tasks should be completed or explicitly deferred.

---

## Traceability

This retrospective traces to:
- **ISSUE-104:** Orchestrator as haiku teammate
- **Retro-notes:** .claude/projects/issue-104/retro-notes.md (O-1, O-2, O-3)
- **Tasks:** .claude/projects/issue-104/tasks.md (TASK-1 through TASK-10)
- **Design:** .claude/projects/issue-104/design.md (DES-001 through DES-016)
- **Commits:** 17 commits to ISSUE-104 (from git log)

---

## Summary

ISSUE-104 successfully designed and partially implemented (70%) a two-role architecture for cost optimization. Key wins: comprehensive design, TDD discipline, SendMessage protocol, model handshake validation, atomic state writes. Key challenges: TaskList tool not used proactively, redundant commit QA phases, unclear TDD task scoping, incomplete project scope. High-priority recommendations focus on explicit TaskList creation and eliminating redundant QA phases. Medium-priority recommendations address task scoping and completion discipline.

**Next Steps:**
1. Review created issues (R-1, R-2 high priority)
2. Decide whether to complete TASK-8, TASK-9, TASK-10 or defer
3. Run integration test once implementation complete
