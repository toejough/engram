# Requirements - ISSUE-105: Remove Composite Skill Redundancy

**Status:** Draft
**Created:** 2026-02-06
**Issue:** ISSUE-105

---

## Problem Statement

Skills were originally designed to spawn sub-agents internally (via Task tool) because they ran in the main conversation context. Now that the `/project` orchestrator spawns dedicated teammates for each skill invocation, the internal sub-agent spawning creates redundant nesting that adds unnecessary indirection without value.

### Evidence

From `skills/tdd-producer/SKILL.md`:
- Line 279: "The tdd-producer runs the full RED -> GREEN -> REFACTOR cycle internally (it does not spawn sub-teammates for each phase)"
- Lines 27-49: Describes "nested pair loops" suggesting sub-agent spawning via Task tool
- This contradiction reveals the architectural mismatch

### Root Cause

**Composite skills are treating themselves as orchestrators** when they should be **state machine transitions**. The `/project` orchestrator already provides the orchestration layer - composite skills that spawn sub-agents duplicate this responsibility.

---

## Solution Overview

Replace composite skills with `projctl state` transitions. Composite "skills" are state machine transitions, not skills that need their own agent context.

**Before (Current):**
```
User -> /project orchestrator
  -> spawns tdd-producer teammate
    -> tdd-producer spawns tdd-red-producer (redundant!)
      -> tdd-red-producer spawns qa (redundant!)
    -> tdd-producer spawns tdd-green-producer (redundant!)
      -> tdd-green-producer spawns qa (redundant!)
```

**After (Desired):**
```
User -> /project orchestrator
  -> projctl step next -> "spawn tdd-red-producer"
  -> spawns tdd-red-producer teammate
  -> projctl step next -> "spawn qa"
  -> spawns qa teammate
  -> projctl step next -> "spawn tdd-green-producer"
  -> spawns tdd-green-producer teammate
  -> projctl step next -> "spawn qa"
  -> spawns qa teammate
```

---

## Requirements

### REQ-105-001: Identify All Composite Skills

**Priority:** High
**Description:** Audit all skills in `skills/` directory to identify which spawn sub-agents internally via Task tool calls.

**Acceptance Criteria:**
1. Search all `skills/*/SKILL.md` files for Task tool usage
2. Identify skills that describe "nested loops", "composite", or "orchestrator" patterns
3. Produce list of composite skills with evidence (file:line references)
4. Classify each as:
   - **Composite orchestrator** (spawns multiple sub-skills in sequence)
   - **Leaf skill** (performs work directly, no spawning)

**Traces to:** ISSUE-105

---

### REQ-105-002: Define State Machine Transitions

**Priority:** High
**Description:** For each composite skill, define the equivalent state machine transitions that the `/project` orchestrator should drive via `projctl step next`.

**Acceptance Criteria:**
1. For each composite skill identified in REQ-105-001:
   - Document current spawning pattern (which sub-skills, in what order)
   - Define equivalent state transitions
   - Specify `projctl step next` JSON output format for each transition
2. Ensure state transitions preserve:
   - Sequential ordering (where required)
   - QA iteration loops (producer -> qa -> improvement-request loop)
   - Error recovery paths
3. Document in `docs/state-machine-transitions.md`

**Example for tdd-producer:**
```
Current: tdd-producer spawns tdd-red-producer, qa, tdd-green-producer, qa, tdd-refactor-producer, qa
New: State transitions:
  - phase=tdd, sub_phase=red -> spawn tdd-red-producer
  - phase=tdd, sub_phase=red-qa -> spawn qa
  - phase=tdd, sub_phase=green -> spawn tdd-green-producer
  - phase=tdd, sub_phase=green-qa -> spawn qa
  - phase=tdd, sub_phase=refactor -> spawn tdd-refactor-producer
  - phase=tdd, sub_phase=refactor-qa -> spawn qa
```

**Traces to:** ISSUE-105, REQ-105-001

---

### REQ-105-003: Update State Machine Implementation

**Priority:** High
**Description:** Implement state transitions in `projctl step next` command to replace composite skill orchestration.

**Acceptance Criteria:**
1. `projctl step next` returns correct next action for all transitions defined in REQ-105-002
2. State machine preserves QA iteration loops:
   - On `qa-verdict=improvement-request`, next action returns same producer skill
   - On `qa-verdict=approved`, advances to next phase/sub-phase
3. State machine enforces max iteration limits (default: 3 iterations per producer-qa loop)
4. All state transitions have unit tests verifying correct next action
5. `projctl step complete` updates state correctly for new sub-phases

**Traces to:** ISSUE-105, REQ-105-002

---

### REQ-105-004: Remove Composite Skill Files

**Priority:** High
**Description:** Delete composite skill directories and update references.

**Acceptance Criteria:**
1. Delete skill directories for composite skills (e.g., `skills/tdd-producer/`)
2. Remove symlinks from `~/.claude/skills/` for deleted skills
3. Update any documentation that references composite skills:
   - Update skill catalog/index
   - Update workflow diagrams
   - Update `/project` skill documentation
4. Verify no remaining references to deleted skills via grep
5. All deletions traced to this requirement in commit message

**Traces to:** ISSUE-105, REQ-105-002

---

### REQ-105-005: Update Orchestrator Skill Documentation

**Priority:** Medium
**Description:** Update `/project` (orchestrator) skill to reflect removal of composite skills.

**Acceptance Criteria:**
1. Remove any mentions of composite skills from `skills/project/SKILL.md`
2. Document that all orchestration happens via `projctl step next` state transitions
3. Add examples showing state-driven producer/QA loops
4. Clarify that skills should NEVER spawn sub-agents via Task tool
5. Update error recovery section to handle state transition failures

**Traces to:** ISSUE-105, REQ-105-004

---

### REQ-105-006: Validate No Internal Task Tool Usage

**Priority:** High
**Description:** Ensure remaining skills do NOT spawn sub-agents internally.

**Acceptance Criteria:**
1. Grep all `skills/*/SKILL.md` files for Task tool references
2. For any remaining Task tool usage:
   - Verify it's NOT spawning sub-agents for orchestration
   - If orchestration detected, escalate as architectural violation
3. Add linter rule (future) to prevent Task tool usage in skill docs
4. Document in `docs/skill-conventions.md`: "Skills MUST NOT spawn sub-agents via Task tool - orchestration is the orchestrator's job"

**Traces to:** ISSUE-105

---

## Success Criteria

### Functional Success
- [ ] All composite skills removed, orchestration happens via state machine
- [ ] `projctl step next` drives all producer/QA iteration loops
- [ ] Existing workflows (e.g., `/project new ISSUE-XXX`) work unchanged from user perspective
- [ ] No skills spawn sub-agents via Task tool

### Performance Success
- [ ] Reduced latency: one fewer agent spawn per composite skill invocation
- [ ] Reduced API costs: eliminate redundant context loading for composite skills
- [ ] Simplified debugging: single orchestrator trace instead of nested conversations

### Quality Success
- [ ] All state transitions have unit tests
- [ ] Documentation updated to reflect new architecture
- [ ] No orphaned references to deleted composite skills

---

## Out of Scope

- Changing user-facing commands (e.g., `/project`, `/commit` syntax remains same)
- Modifying leaf skill behavior (only composite skills affected)
- Parallel skill dispatch (separate from this issue)

---

## Dependencies

- ARCH-012 (Deterministic Workflow Enforcement) - state machine preconditions
- `projctl step next` command implementation (already exists)
- `projctl step complete` command implementation (already exists)

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Breaking existing workflows | High | Comprehensive testing of all workflow types (new/task/adopt/align) |
| Missing state transitions | Medium | Audit composite skills thoroughly, test all paths |
| Incorrect max iteration limits | Low | Preserve existing iteration limits from composite skills |
| Documentation drift | Medium | Update all docs atomically with code changes |

---

## Traceability

**Traces to:** ISSUE-105

**Referenced by:** TBD (design, architecture, test artifacts)
