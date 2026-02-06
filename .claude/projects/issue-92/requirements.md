# Requirements: Per-Phase QA in TDD Loop

**Issue:** ISSUE-92
**Project:** Per-Phase QA in TDD Loop
**Date:** 2026-02-06

## Overview

Restructure the TDD loop so each sub-phase (red, green, refactor) has its own producer/QA pair instead of deferring all QA to the end. This change makes QA immediate and scoped, catching problems before they compound across phases.

Currently, QA happens once at the end of the TDD loop via `tdd-qa`. Problems in the red phase (bad tests) aren't caught until after green and refactor are complete, requiring rework across all three phases.

The new structure embeds QA **within** each phase as a sub-step, not as separate state machine phases. Each phase follows the pattern: producer creates artifact → QA verifies artifact → commit persists artifact. This applies to both TDD phases (red/green/refactor) and commit operations.

## Requirements

### REQ-001: Per-Phase QA Integration

**Description:** Each TDD sub-phase (red, green, refactor) must include QA verification as an internal step before proceeding to commit.

**Acceptance Criteria:**
- QA verification happens within the phase, not as a separate state machine phase
- Producer completes work first, then QA verifies before commit
- QA scope is limited to the specific phase's work (red tests only, green impl only, refactor changes only)
- Phase does not advance to commit until QA approves
- Implementation uses the existing pair-loop pattern (spawn-producer → spawn-qa → commit)

**Priority:** High

**Traces to:** ISSUE-92

---

### REQ-002: Commit QA - Staging Verification

**Description:** Commit operations must include QA verification of staging correctness before the commit is finalized.

**Acceptance Criteria:**
- Commit-QA verifies the right files were staged (nothing missing, nothing extra)
- Commit-QA verifies no secrets or sensitive data are staged
- Commit-QA does NOT verify message quality (commit skill already handles this)
- Commit-QA does NOT verify hooks, branch, or commit success (commit skill handles failures)
- Commit does not proceed until commit-QA approves staging

**Priority:** High

**Traces to:** REQ-001, ISSUE-92

---

### REQ-003: Remove Final TDD-QA Phase

**Description:** The final `tdd-qa` phase at the end of the TDD loop must be removed as it becomes redundant with per-phase QA.

**Acceptance Criteria:**
- `tdd-qa` phase removed from state machine
- `tdd-qa` skill removed or deprecated
- TDD loop transitions directly from final commit to completion
- No end-of-loop meta-verification (each phase already verified)

**Priority:** High

**Traces to:** REQ-001, ISSUE-92

**Resolves:** ISSUE-91 (tdd-qa rename becomes moot)

---

### REQ-004: QA Failure Recovery - Improvement Loop

**Description:** When QA rejects work, the producer must receive feedback and make improvements, with escalation after exhausting attempts.

**Acceptance Criteria:**
- QA can return `improvement-request` with specific feedback
- Producer receives feedback and iterates on the work
- Maximum 3 improvement-request cycles per phase
- After 3 failed iterations, escalate to user BEFORE commit
- No work is committed if QA never approves

**Priority:** High

**Traces to:** REQ-001, ISSUE-92

---

### REQ-005: Step Registry Updates

**Description:** The phase registry must be updated to include QA skills for TDD sub-phases and commit operations.

**Acceptance Criteria:**
- Phase registry includes QA skill references for `tdd-red`, `tdd-green`, `tdd-refactor`
- Phase registry includes QA skill reference for `commit`
- Registry structure supports producer + QA pairing within a single phase
- `projctl step next` uses registry to determine QA actions automatically

**Priority:** High

**Traces to:** REQ-001, ISSUE-92

**Relates to:** ISSUE-89 (phase registry)

---

### REQ-006: State Machine Internal Sub-Steps

**Description:** State machine phases must support internal sub-steps for QA without creating new top-level phases.

**Acceptance Criteria:**
- Phases can return multiple actions (spawn-producer, spawn-qa, commit) from step-next
- Actions execute sequentially within the same phase
- Phase only transitions to next phase after all internal actions complete
- No new `tdd-red-qa`, `tdd-green-qa`, `tdd-refactor-qa` phases in state machine (QA is internal)
- Phase state tracks sub-step progress (producer pending/complete, QA pending/complete, commit pending/complete)

**Priority:** High

**Traces to:** REQ-001, REQ-005, ISSUE-92

---

### REQ-007: Backward Compatibility - New Tasks Only

**Description:** Per-phase QA logic applies only to newly created tasks, not tasks already in progress.

**Acceptance Criteria:**
- Existing in-progress tasks continue with old phase flow (no QA sub-steps)
- New tasks created after deployment use new per-phase QA flow
- No state migration required for in-flight projects
- State machine version or creation timestamp determines which flow to use

**Priority:** Medium

**Traces to:** REQ-001, ISSUE-92

---

### REQ-008: Structural Verification Testing

**Description:** Implementation must be verifiable through `projctl step next` output showing correct action sequences.

**Acceptance Criteria:**
- `projctl step next` for `tdd-red` phase returns: spawn-producer, spawn-qa, commit (in order)
- `projctl step next` for `tdd-green` phase returns: spawn-producer, spawn-qa, commit (in order)
- `projctl step next` for `tdd-refactor` phase returns: spawn-producer, spawn-qa, commit (in order)
- Output includes skill names for producer and QA agents
- Verification can be automated via structural output tests

**Priority:** High

**Traces to:** REQ-001, REQ-005, REQ-006, ISSUE-92

---

### REQ-009: TDD Sub-Phase QA Skills

**Description:** QA skills must be created or identified for each TDD sub-phase to verify phase-specific correctness.

**Acceptance Criteria:**
- `tdd-red-qa` skill verifies tests are well-formed, fail for right reasons, match requirements
- `tdd-green-qa` skill verifies implementation makes tests pass, no over-implementation, follows conventions
- `tdd-refactor-qa` skill verifies refactoring preserved behavior, improved structure, no new functionality
- Each QA skill scope limited to its phase (red-qa doesn't check green implementation)
- QA skills follow standard approval/improvement-request/escalation pattern

**Priority:** High

**Traces to:** REQ-001, REQ-004, ISSUE-92

---

### REQ-010: Commit QA Skill

**Description:** A commit-QA skill must verify staging correctness before commits are finalized.

**Acceptance Criteria:**
- `commit-qa` skill verifies all changed files are staged
- `commit-qa` skill verifies no unintended files staged (build artifacts, temp files, etc.)
- `commit-qa` skill verifies no secrets staged (.env files, credentials, API keys, etc.)
- `commit-qa` skill does NOT verify commit message (commit skill responsibility)
- `commit-qa` skill does NOT verify hooks/branch/success (commit skill handles failures)

**Priority:** High

**Traces to:** REQ-002, REQ-004, ISSUE-92

---

## Out of Scope

- Changing the commit skill's message validation (already handles this)
- Retry logic for commit failures (commit skill already handles this)
- Cross-phase QA verification (each phase is independent)
- Behavioral/integration tests (structural verification is sufficient per PM interview)
- Migration of in-flight tasks to new QA flow

## Success Metrics

- `projctl step next` output shows spawn-producer → spawn-qa → commit for TDD phases
- Injecting a bad test in tdd-red triggers tdd-red-qa improvement-request
- Staging wrong files triggers commit-qa improvement-request
- `go test ./...` passes
- State machine transitions work correctly with internal sub-steps
- New tasks use per-phase QA, existing tasks continue with old flow
