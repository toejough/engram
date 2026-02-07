# Tasks - ISSUE-105: Remove Composite Skill Redundancy

**Status:** Draft
**Created:** 2026-02-06
**Issue:** ISSUE-105

**Traces to:** ISSUE-105, REQ-001, REQ-002, REQ-003, REQ-004, REQ-005, REQ-006

---

## Simplicity Rationale

This task breakdown represents the simplest approach to eliminate composite skill redundancy because:

1. **Leverages existing patterns**: Uses established `projctl step next/complete` commands rather than creating new orchestration mechanisms
2. **Stateless design**: Orchestrator becomes a thin execution layer with no internal state tracking
3. **Incremental implementation**: Tasks are ordered to enable testing at each phase, reducing risk
4. **Backward compatibility**: Auto-migration prevents breaking in-flight projects
5. **Minimal component changes**: Only touches state machine, registry, and composite skill files

**Alternatives considered:**
- Creating new orchestration commands: Rejected (increases complexity, duplicate patterns)
- Stateful orchestrator with internal iteration tracking: Rejected (duplicates state.toml responsibility)
- Hard cutoff without migration: Rejected (breaks user workflows)

The current approach achieves the goal (eliminate redundant nesting) with minimal code changes and maximum reuse of existing infrastructure.

---

## Task Breakdown

### TASK-1: Audit composite skills for internal Task tool usage

**Description:** Identify all skills in the `skills/` directory that spawn sub-agents internally via Task tool calls, classifying them as composite orchestrators (to be removed) or leaf skills (to be preserved).

**Status:** Ready

**Acceptance Criteria:**
- [ ] All `skills/*/SKILL.md` files searched for Task tool usage patterns
- [ ] Skills classified as "Composite orchestrator" or "Leaf skill" with evidence (file:line references)
- [ ] Audit report produced listing composite skills: tdd-producer, parallel-looper (deprecated)
- [ ] For each composite skill, document which sub-skills it spawns and in what order
- [ ] Verification: grep confirms no additional composite skills beyond identified ones

**Files:** `docs/composite-skill-audit.md` (new audit report)

**Dependencies:** None

**Traces to:** REQ-001, DES-007, DES-014, ARCH-013, ISSUE-105

---

### TASK-2: Define state machine transitions for TDD sub-phases

**Description:** Define the complete state transition table that replaces `tdd-producer` composite skill orchestration logic, including TDD sub-phases (red, green, refactor) with their corresponding QA and commit phases.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Transition table documented for TDD workflow: tdd-red → tdd-red-qa → commit-red → commit-red-qa → tdd-green → ...
- [ ] All legal transitions specified in `internal/state/transitions.go`
- [ ] Sequential ordering preserved (no phase skipping)
- [ ] QA iteration loops defined (improvement-request returns to same producer)
- [ ] Error recovery paths specified (escalate-user handling)
- [ ] Documentation includes phase sequence diagram

**Files:** `internal/state/transitions.go`

**Dependencies:** TASK-1

**Traces to:** REQ-002, DES-008, ARCH-006, ARCH-008, ISSUE-105

---

### TASK-3: Implement phase registry with TDD sub-phases

**Description:** Implement the phase registry mapping phase identifiers to producer/QA skills, models, and SKILL.md file paths for all TDD sub-phases.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Phase registry implemented in `internal/step/registry.go`
- [ ] All TDD sub-phases have registry entries: tdd-red, tdd-red-qa, commit-red, commit-red-qa, tdd-green, tdd-green-qa, commit-green, commit-green-qa, tdd-refactor, tdd-refactor-qa, commit-refactor, commit-refactor-qa
- [ ] Each phase entry specifies Producer, ProducerPath, QA, QAPath, Model, QAModel
- [ ] All skill paths validated to exist
- [ ] Registry follows naming conventions: `<artifact>-<type>` pattern

**Files:** `internal/step/registry.go`

**Dependencies:** TASK-2

**Traces to:** REQ-002, DES-008, ARCH-005, ISSUE-105

---

### TASK-4: Implement state storage schema and validation

**Description:** Implement state.toml loading, saving, and validation logic with schema for workflow state persistence including iteration tracking.

**Status:** Ready

**Acceptance Criteria:**
- [ ] State schema implemented in `internal/state/state.go` with fields: workflow, phase, qa, artifacts, team
- [ ] TOML parsing/serialization works correctly (using go-toml/v2)
- [ ] State validation checks required fields, valid phase names, non-negative iterations, valid QA verdicts
- [ ] Validation runs on load, save, and transition
- [ ] Error messages provide actionable guidance for invalid state

**Files:** `internal/state/state.go`

**Dependencies:** TASK-3

**Traces to:** REQ-003, DES-006, ARCH-004, ARCH-018, ISSUE-105

---

### TASK-5: Implement iteration enforcement logic

**Description:** Implement max iteration limit enforcement in state machine to prevent infinite producer/QA retry loops, returning escalate-user action when limits are reached.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Iteration counter increments on QA improvement-request verdict
- [ ] Iteration counter resets to 0 on phase transition (QA approved)
- [ ] Max iteration limit enforced (default: 3)
- [ ] State machine returns escalate-user action when iteration >= max_iterations
- [ ] Escalation includes detailed message with phase, iteration count, and QA feedback
- [ ] QA feedback propagates to next producer spawn attempt

**Files:** `internal/step/next.go`

**Dependencies:** TASK-4

**Traces to:** REQ-003, DES-006, DES-010, ARCH-007, ARCH-009, ISSUE-105

---

### TASK-6: Implement projctl step next command logic

**Description:** Implement `projctl step next` state machine logic that reads state.toml, applies transition rules, enforces iteration limits, and returns structured JSON action for orchestrator.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Command reads state from `.claude/projects/<issue>/state.toml`
- [ ] Returns correct action type for each phase: spawn-producer, spawn-qa, commit, transition, escalate-user, all-complete
- [ ] Action JSON includes all required fields: action, skill, skill_path, model, phase, iteration, context, task_params
- [ ] Context includes issue, prior_artifacts, qa_feedback
- [ ] TaskParams structured correctly for Task tool invocation
- [ ] Handles QA verdicts: approved (advance phase), improvement-request (increment iteration + re-spawn)
- [ ] Enforces legal transitions (rejects illegal jumps)

**Files:** `internal/step/next.go`, `cmd/projctl/step_next.go`

**Dependencies:** TASK-5

**Traces to:** REQ-003, DES-009, ARCH-003, ISSUE-105

---

### TASK-7: Implement projctl step complete command logic

**Description:** Implement `projctl step complete` command that updates state.toml based on completed actions (producer done, QA verdict, escalation resolution).

**Status:** Ready

**Acceptance Criteria:**
- [ ] Command accepts flags: --action, --status, --qa-verdict, --qa-feedback, --user-decision
- [ ] Updates state.toml phase, iteration, qa_verdict, qa_feedback based on action type
- [ ] Validates state transitions before persisting changes
- [ ] Handles producer completion: updates artifacts.current list
- [ ] Handles QA completion: records verdict and feedback
- [ ] Handles escalation resolution: applies user decision
- [ ] Returns success/error status

**Files:** `internal/step/complete.go`, `cmd/projctl/step_complete.go`

**Dependencies:** TASK-6

**Traces to:** REQ-003, ARCH-003, ARCH-004, ISSUE-105

---

### TASK-8: Implement backward compatibility auto-migration

**Description:** Implement auto-migration logic in state loading to convert legacy `phase=tdd` to `phase=tdd-red`, ensuring in-flight projects continue working after composite skill deletion.

**Status:** Ready

**Acceptance Criteria:**
- [ ] State loading detects legacy phase `tdd`
- [ ] Auto-migrates to `tdd-red` with iteration=0
- [ ] Migration logged for debugging
- [ ] Migrated state persisted to disk
- [ ] Migration is one-time per project (idempotent)
- [ ] No user intervention required

**Files:** `internal/state/state.go`

**Dependencies:** TASK-7

**Traces to:** REQ-004, DES-008, ARCH-012, ISSUE-105

---

### TASK-9: Write unit tests for state machine transitions

**Description:** Write comprehensive unit tests for state transition logic, validating legal transitions are accepted and illegal transitions are rejected.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Tests for all legal TDD sub-phase transitions pass
- [ ] Tests for illegal transition rejection pass (e.g., tdd-red → commit-red, tdd-red → tdd-green)
- [ ] Test full phase chain: tdd-red → ... → commit-refactor-qa → task-audit
- [ ] Test transition validation function with valid/invalid phase pairs
- [ ] Test coverage for transitions.go ≥ 90%

**Files:** `internal/state/transitions_test.go`

**Dependencies:** TASK-2

**Traces to:** REQ-003, DES-015, ARCH-014, ISSUE-105

---

### TASK-10: Write unit tests for phase registry

**Description:** Write unit tests validating phase registry entries have correct producer/QA skills, valid file paths, and appropriate model selections.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Tests verify all TDD sub-phases have registry entries
- [ ] Tests verify producer/QA skill paths exist on filesystem
- [ ] Tests verify model selections are valid (sonnet, haiku, opus)
- [ ] Tests verify registry naming convention consistency
- [ ] Test coverage for registry.go ≥ 90%

**Files:** `internal/step/registry_test.go`

**Dependencies:** TASK-3

**Traces to:** REQ-003, DES-015, ARCH-014, ISSUE-105

---

### TASK-11: Write unit tests for state persistence

**Description:** Write unit tests for state.toml loading, saving, TOML parsing, field validation, and error handling.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Tests for valid TOML parsing pass
- [ ] Tests for invalid TOML (malformed syntax) return appropriate errors
- [ ] Tests for state validation (required fields, valid phases, iteration bounds) pass
- [ ] Tests for state load/save roundtrip preserve all fields
- [ ] Tests for missing state file return actionable error
- [ ] Test coverage for state.go ≥ 90%

**Files:** `internal/state/state_test.go`

**Dependencies:** TASK-4

**Traces to:** REQ-003, DES-015, ARCH-014, ISSUE-105

---

### TASK-12: Write unit tests for iteration enforcement

**Description:** Write unit tests for iteration increment logic, max iteration limit enforcement, and escalate-user action generation.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Test iteration increments on improvement-request verdict
- [ ] Test iteration resets to 0 on approved verdict (phase transition)
- [ ] Test escalate-user action returned when iteration >= max_iterations
- [ ] Test escalation message includes phase, iteration count, and QA feedback
- [ ] Test QA feedback propagates to next producer spawn context
- [ ] Test coverage for iteration logic in next.go ≥ 90%

**Files:** `internal/step/next_test.go`

**Dependencies:** TASK-5

**Traces to:** REQ-003, DES-015, ARCH-014, ISSUE-105

---

### TASK-13: Write unit tests for step next action generation

**Description:** Write unit tests validating `projctl step next` returns correct action JSON for each phase, handles QA verdicts correctly, and enforces transition rules.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Tests for spawn-producer action generation pass (correct skill, model, context, task_params)
- [ ] Tests for spawn-qa action generation pass
- [ ] Tests for QA verdict handling pass (approved → advance phase, improvement-request → increment iteration)
- [ ] Tests for illegal transition rejection pass
- [ ] Tests for all-complete action when workflow finishes pass
- [ ] Test coverage for next.go ≥ 90%

**Files:** `internal/step/next_test.go`

**Dependencies:** TASK-6

**Traces to:** REQ-003, DES-015, ARCH-014, ISSUE-105

---

### TASK-14: Write unit tests for step complete state updates

**Description:** Write unit tests validating `projctl step complete` correctly updates state.toml for producer completion, QA verdicts, and escalation resolutions.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Tests for producer completion state update pass (status=done)
- [ ] Tests for QA verdict state update pass (verdict, feedback)
- [ ] Tests for escalation resolution state update pass (user-decision)
- [ ] Tests for invalid action types return errors
- [ ] Tests verify state validation runs before persisting changes
- [ ] Test coverage for complete.go ≥ 90%

**Files:** `internal/step/complete_test.go`

**Dependencies:** TASK-7

**Traces to:** REQ-003, DES-015, ARCH-014, ISSUE-105

---

### TASK-15: Write unit tests for backward compatibility migration

**Description:** Write unit tests validating auto-migration from legacy `phase=tdd` to `phase=tdd-red` works correctly and is idempotent.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Test legacy `phase=tdd` migrates to `phase=tdd-red` with iteration=0
- [ ] Test migration is logged for debugging
- [ ] Test migrated state is persisted to disk
- [ ] Test migration is idempotent (running twice has same effect as once)
- [ ] Test non-legacy phases are not affected by migration logic

**Files:** `internal/state/state_test.go`

**Dependencies:** TASK-8

**Traces to:** REQ-004, DES-008, ARCH-012, ISSUE-105

---

### TASK-16: Write property-based tests for state machine invariants

**Description:** Write property-based tests using rapid to verify state machine never gets stuck, all legal transition sequences reach "complete", and iteration counter invariants hold.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Property test: state machine always returns valid action (never stuck)
- [ ] Property test: all legal transition sequences eventually reach "complete" phase
- [ ] Property test: iteration counter never decreases
- [ ] Property test: max iteration limit always enforced across all phases
- [ ] Tests use pgregory.net/rapid for randomized property exploration

**Files:** `internal/step/next_test.go`

**Dependencies:** TASK-13

**Traces to:** REQ-003, DES-015, ARCH-014, ISSUE-105

---

### TASK-17: Write integration test for full TDD workflow

**Description:** Write end-to-end integration test validating state-driven TDD workflow executes correctly from tdd-red through task-audit without composite skills.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Test sets up sample issue with acceptance criteria
- [ ] Test initializes state machine (projctl state init)
- [ ] Test simulates orchestrator loop: step next → spawn → step complete
- [ ] Test verifies all TDD sub-phases execute in sequence
- [ ] Test verifies expected artifacts are created (tests, implementation, refactored code)
- [ ] Test verifies git commits created after each TDD stage
- [ ] Test verifies no composite skill (tdd-producer) is referenced in orchestrator execution

**Files:** `internal/step/integration_test.go`

**Dependencies:** TASK-14

**Traces to:** REQ-003, REQ-004, DES-016, ARCH-014, ISSUE-105

---

### TASK-18: Write integration test for QA iteration with feedback

**Description:** Write integration test validating producer/QA iteration loop works correctly with improvement-request verdicts and feedback propagation.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Test injects QA improvement-request verdict
- [ ] Test verifies producer re-spawns with QA feedback in context
- [ ] Test verifies iteration counter increments correctly
- [ ] Test verifies max iteration triggers escalate-user action
- [ ] Test verifies approved verdict advances to next phase with iteration reset

**Files:** `internal/step/integration_test.go`

**Dependencies:** TASK-17

**Traces to:** REQ-003, DES-016, ARCH-014, ISSUE-105

---

### TASK-19: Write integration test for backward compatibility migration

**Description:** Write integration test validating legacy state with `phase=tdd` auto-migrates to `phase=tdd-red` and workflow continues correctly.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Test creates state.toml with legacy `phase=tdd`
- [ ] Test runs projctl step next
- [ ] Test verifies state auto-migrates to `phase=tdd-red`
- [ ] Test verifies workflow continues normally from tdd-red
- [ ] Test verifies migration is logged

**Files:** `internal/step/integration_test.go`

**Dependencies:** TASK-18

**Traces to:** REQ-004, DES-008, ARCH-012, ISSUE-105

---

### TASK-20: Delete composite skill directories

**Description:** Delete `skills/tdd-producer/` and `skills/parallel-looper/` directories and remove corresponding symlinks from `~/.claude/skills/` after verifying all tests pass.

**Status:** Ready

**Acceptance Criteria:**
- [ ] All unit tests pass (TASK-9 through TASK-16)
- [ ] All integration tests pass (TASK-17 through TASK-19)
- [ ] Directory `skills/tdd-producer/` deleted
- [ ] Directory `skills/parallel-looper/` deleted
- [ ] Symlink `~/.claude/skills/tdd-producer` removed
- [ ] Symlink `~/.claude/skills/parallel-looper` removed
- [ ] Grep verification confirms no remaining references to deleted skills in skills/ or docs/

**Files:** `skills/tdd-producer/` (deleted), `skills/parallel-looper/` (deleted)

**Dependencies:** TASK-19

**Traces to:** REQ-004, DES-011, ARCH-013, ISSUE-105

---

### TASK-21: Update orchestrator SKILL.md documentation

**Description:** Update `skills/project/SKILL.md` to remove composite skill references, clarify state-machine-driven orchestration, add producer/QA loop examples, and update error recovery section.

**Status:** Ready

**Acceptance Criteria:**
- [ ] All mentions of `tdd-producer` removed from SKILL.md
- [ ] Explicit statement added: "Skills MUST NOT spawn sub-agents via Task tool - orchestration is the orchestrator's job"
- [ ] Producer/QA iteration pattern documented with state machine loop example
- [ ] Error recovery section updated to document escalate-user handling
- [ ] Max iteration behavior documented
- [ ] State transition failure troubleshooting steps added

**Files:** `skills/project/SKILL.md`

**Dependencies:** TASK-20

**Traces to:** REQ-005, DES-012, ISSUE-105

---

### TASK-22: Update orchestrator SKILL-full.md documentation

**Description:** Update `skills/project/SKILL-full.md` phase detail tables to replace `tdd` phase with TDD sub-phases and update resume map for new phases.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Phase detail table updated with TDD sub-phases: tdd-red, tdd-red-qa, commit-red, commit-red-qa, tdd-green, tdd-green-qa, commit-green, commit-green-qa, tdd-refactor, tdd-refactor-qa, commit-refactor, commit-refactor-qa
- [ ] Resume map updated with instructions for new TDD sub-phases
- [ ] Iteration state recovery documented
- [ ] All composite skill references removed

**Files:** `skills/project/SKILL-full.md`

**Dependencies:** TASK-21

**Traces to:** REQ-005, DES-012, ISSUE-105

---

### TASK-23: Create skill convention documentation

**Description:** Create or update `docs/skill-conventions.md` to document architectural rule prohibiting internal Task tool usage for orchestration in skills.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Section "Orchestration Prohibition" added with rule statement
- [ ] Rationale explained (orchestrator responsibility, redundant nesting)
- [ ] Allowed vs. prohibited Task tool usage clearly differentiated
- [ ] Example of prohibited composite skill pattern provided
- [ ] Example of allowed leaf skill pattern provided
- [ ] Enforcement mechanisms documented (manual audit, grep check, future linter)
- [ ] Traces to ISSUE-105, REQ-004, REQ-005

**Files:** `docs/skill-conventions.md`

**Dependencies:** TASK-22

**Traces to:** REQ-006, DES-013, ISSUE-105

---

### TASK-24: Validate no remaining Task tool orchestration usage

**Description:** Grep all remaining `skills/*/SKILL.md` files for Task tool usage and verify none are used for orchestration (only allowed utility usage remains).

**Status:** Ready

**Acceptance Criteria:**
- [ ] All `skills/*/SKILL.md` files searched for Task tool references
- [ ] Each remaining Task tool usage verified as utility (not orchestration)
- [ ] Any orchestration usage flagged as architectural violation and escalated
- [ ] Validation report generated with list of remaining Task tool usages and classifications
- [ ] Grep check added to documentation: `grep -r "Task(" skills/*/SKILL.md`

**Files:** `docs/task-tool-validation-report.md` (new validation report)

**Dependencies:** TASK-23

**Traces to:** REQ-006, DES-013, ISSUE-105

---

## Dependency Graph

```
TASK-1 (Audit composite skills)
    |
TASK-2 (Define state transitions) ----+---- TASK-9 (Unit tests: transitions)
    |                                  |
TASK-3 (Implement phase registry) ----+---- TASK-10 (Unit tests: registry)
    |                                  |
TASK-4 (Implement state storage) -----+---- TASK-11 (Unit tests: state)
    |                                  |
TASK-5 (Implement iteration logic) ---+---- TASK-12 (Unit tests: iteration)
    |                                  |
TASK-6 (Implement step next) ---------+---- TASK-13 (Unit tests: step next)
    |                                           |
TASK-7 (Implement step complete) ----+---- TASK-14 (Unit tests: step complete)
    |                                  |
TASK-8 (Auto-migration) --------------+---- TASK-15 (Unit tests: migration)
                                       |
                                  TASK-16 (Property tests)
                                       |
                                  TASK-17 (Integration: full TDD)
                                       |
                                  TASK-18 (Integration: QA iteration)
                                       |
                                  TASK-19 (Integration: migration)
                                       |
                                  TASK-20 (Delete composite skills)
                                       |
                                  TASK-21 (Update SKILL.md)
                                       |
                                  TASK-22 (Update SKILL-full.md)
                                       |
                                  TASK-23 (Create skill conventions)
                                       |
                                  TASK-24 (Validate no Task orchestration)
```

**Parallel Opportunities:**
- TASK-9, TASK-10, TASK-11 can run in parallel (independent test files)
- TASK-12, TASK-13, TASK-14, TASK-15 can run in parallel after their respective implementation tasks
- TASK-17, TASK-18 can run in parallel (independent integration test scenarios)

**Critical Path:**
TASK-1 → TASK-2 → TASK-3 → TASK-4 → TASK-5 → TASK-6 → TASK-7 → TASK-8 → [Tests] → TASK-20 → TASK-21 → TASK-22 → TASK-23 → TASK-24

---

## Task Summary

**Total Tasks:** 24

**Breakdown by Type:**
- Audit/Analysis: 1 task (TASK-1)
- Design/Definition: 1 task (TASK-2)
- Implementation: 6 tasks (TASK-3 to TASK-8)
- Unit Tests: 7 tasks (TASK-9 to TASK-15)
- Property Tests: 1 task (TASK-16)
- Integration Tests: 3 tasks (TASK-17 to TASK-19)
- Deletion: 1 task (TASK-20)
- Documentation: 4 tasks (TASK-21 to TASK-24)

**Visual Tasks:** None (all tasks are backend state machine and documentation changes)

**Estimated Complexity:**
- High: TASK-6 (step next logic), TASK-7 (step complete logic), TASK-17 (integration test)
- Medium: TASK-2, TASK-3, TASK-4, TASK-5, TASK-13, TASK-14, TASK-21, TASK-22
- Low: TASK-1, TASK-8, TASK-9, TASK-10, TASK-11, TASK-12, TASK-15, TASK-16, TASK-18, TASK-19, TASK-20, TASK-23, TASK-24

---

## Implementation Notes

### TDD Discipline

All tasks follow strict TDD:
1. **Red**: Write tests first (TASK-9 to TASK-19)
2. **Green**: Implement to pass tests (TASK-3 to TASK-8)
3. **Refactor**: Cleanup after tests pass

### Test-First Ordering

Tests are intentionally sequenced immediately after their corresponding implementation tasks to enforce TDD discipline and enable rapid feedback loops.

### Backward Compatibility

TASK-8 (auto-migration) is critical for zero-disruption deployment. Must be implemented and tested before TASK-20 (deletion) executes.

### Documentation Last

Documentation tasks (TASK-21 to TASK-24) are sequenced after deletion to ensure docs reflect the final architecture state without composite skills.

---

## Traceability

**Traces to:** ISSUE-105

**Satisfies Requirements:**
- REQ-001: TASK-1
- REQ-002: TASK-2, TASK-3
- REQ-003: TASK-4, TASK-5, TASK-6, TASK-7, TASK-9, TASK-10, TASK-11, TASK-12, TASK-13, TASK-14, TASK-16, TASK-17, TASK-18
- REQ-004: TASK-8, TASK-15, TASK-19, TASK-20
- REQ-005: TASK-21, TASK-22
- REQ-006: TASK-23, TASK-24

**Referenced by:** TBD (TDD artifacts, test results)
