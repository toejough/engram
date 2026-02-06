# Design: Per-Phase QA in TDD Loop

**Issue:** ISSUE-92
**Project:** Per-Phase QA in TDD Loop
**Date:** 2026-02-06

## Overview

This design specifies the state machine behavior and action sequences for integrating QA verification within TDD phases. The orchestrator consumes JSON actions from `projctl step next`, which must return the correct sequence of producer → QA → commit actions for each phase.

This is a **backend state machine design**, not a user interface design. The "user" is the orchestrator agent reading JSON responses.

## Design Decisions

### DES-001: Sub-Step State Model

Each phase that requires QA follows a three-sub-step internal state model:

```
Phase: tdd-red
  Sub-step 1: producer (pending → complete)
  Sub-step 2: qa (pending → complete)
  Sub-step 3: commit (pending → complete)
Phase complete → transition to next phase
```

**State tracking:**
- Phase tracks current sub-step: `producer_pending`, `producer_complete`, `qa_pending`, `qa_complete`, `commit_pending`, `commit_complete`
- `projctl step next` determines which sub-step action to return based on current state
- Sub-steps execute sequentially within the phase (no parallelism)

**Phase completion:**
- Phase only transitions to next phase after all sub-steps complete
- No new top-level phases created (e.g., no `tdd-red-qa` phase)

**Traces to:** REQ-001, REQ-006

---

### DES-002: Action Sequence for TDD Phases

When `projctl step next` is called during TDD phases (`tdd-red`, `tdd-green`, `tdd-refactor`), it returns actions in this sequence:

**First call (producer pending):**
```json
{
  "action": "spawn-producer",
  "skill": "tdd-red-producer",
  "phase": "tdd-red",
  "sub_step": "producer"
}
```

**Second call (producer complete, QA pending):**
```json
{
  "action": "spawn-qa",
  "skill": "tdd-red-qa",
  "phase": "tdd-red",
  "sub_step": "qa"
}
```

**Third call (QA approved, commit pending):**
```json
{
  "action": "commit",
  "skill": "commit",
  "phase": "tdd-red",
  "sub_step": "commit"
}
```

**Fourth call (all sub-steps complete):**
```json
{
  "action": "transition",
  "next_phase": "commit-red"
}
```

This sequence applies to all TDD phases:
- `tdd-red` → spawn tdd-red-producer → spawn tdd-red-qa → commit
- `tdd-green` → spawn tdd-green-producer → spawn tdd-green-qa → commit
- `tdd-refactor` → spawn tdd-refactor-producer → spawn tdd-refactor-qa → commit

**Traces to:** REQ-001, REQ-005, REQ-008

---

### DES-003: Action Sequence for Commit Phases

Commit phases (`commit-red`, `commit-green`, `commit-refactor`) also follow the sub-step pattern:

**First call (producer pending):**
```json
{
  "action": "spawn-producer",
  "skill": "commit-producer",
  "phase": "commit-red",
  "sub_step": "producer"
}
```

**Second call (producer complete, QA pending):**
```json
{
  "action": "spawn-qa",
  "skill": "commit-qa",
  "phase": "commit-red",
  "sub_step": "qa"
}
```

**Third call (QA approved, finalize pending):**
```json
{
  "action": "finalize",
  "phase": "commit-red",
  "sub_step": "finalize"
}
```

**Fourth call (all sub-steps complete):**
```json
{
  "action": "transition",
  "next_phase": "tdd-green"
}
```

**Traces to:** REQ-002, REQ-010

---

### DES-004: QA Verdict Handling - Improvement Loop

When QA completes, it returns a verdict via its final message to the orchestrator. The verdict determines the next action:

**QA returns `approved`:**
- Phase proceeds to commit sub-step
- Example: `{"verdict": "approved", "feedback": ""}`

**QA returns `improvement-request`:**
- Phase returns to producer sub-step (retry)
- Iteration counter increments
- Producer receives QA feedback for corrections
- Maximum 3 iterations allowed
- Example: `{"verdict": "improvement-request", "feedback": "Tests contain syntax errors in test_login.go:15"}`

**QA returns `escalate-user` (after 3 failed iterations):**
- Phase pauses, orchestrator surfaces issue to user
- User must manually resolve before continuing
- Example: `{"verdict": "escalate-user", "feedback": "Unable to fix test failures after 3 attempts. Manual intervention required."}`

**Iteration tracking:**
- Phase state includes `qa_iteration_count: 0-3`
- Counter resets when QA approves or when phase transitions
- Counter persists across `step next` calls within the same phase

**Traces to:** REQ-004

---

### DES-005: QA Immediate Feedback

QA verdicts are returned immediately to the orchestrator with no batching or deferral:

**Timing:**
- QA completes its analysis
- QA sends verdict message to orchestrator
- Orchestrator's next `step next` call reflects QA verdict in action sequence

**No user confirmation:**
- Improvement loops are automatic (no "retry?" prompts)
- Only escalation (after 3 failures) involves user
- Orchestrator processes verdicts programmatically

**Traces to:** REQ-004

---

### DES-006: Commit-QA Auto-Retry

When commit-QA detects staging issues, it provides feedback for the commit-producer to retry automatically:

**Scenario:** commit-producer staged wrong files

**Flow:**
1. commit-producer stages files
2. commit-qa detects issue: `{"verdict": "improvement-request", "feedback": "File test_integration.go should not be staged (test belongs in green phase)"}`
3. `step next` returns spawn-producer action again (retry)
4. commit-producer receives feedback, adjusts staging
5. commit-qa re-verifies
6. If approved, proceed to finalize

**No user interaction:**
- Staging corrections happen automatically within iteration limit
- Only escalates to user after 3 failed attempts

**Traces to:** REQ-002, REQ-004

---

### DES-007: Phase Registry Structure

The phase registry must map each phase to its producer and QA skills:

**Registry format:**
```json
{
  "tdd-red": {
    "producer": "tdd-red-producer",
    "qa": "tdd-red-qa",
    "commit": "commit"
  },
  "tdd-green": {
    "producer": "tdd-green-producer",
    "qa": "tdd-green-qa",
    "commit": "commit"
  },
  "tdd-refactor": {
    "producer": "tdd-refactor-producer",
    "qa": "tdd-refactor-qa",
    "commit": "commit"
  },
  "commit-red": {
    "producer": "commit-producer",
    "qa": "commit-qa"
  },
  "commit-green": {
    "producer": "commit-producer",
    "qa": "commit-qa"
  },
  "commit-refactor": {
    "producer": "commit-producer",
    "qa": "commit-qa"
  }
}
```

**Usage:**
- `projctl step next` reads registry to determine which skill to spawn
- Registry is internal to projctl (not exposed via CLI flags)
- QA skills are optional per phase (phases without QA proceed directly to commit)

**Traces to:** REQ-005

---

### DES-008: TDD-QA Phase Removal

The final `tdd-qa` phase is removed from the state machine:

**Before:**
```
tdd-red → commit-red → tdd-green → commit-green → tdd-refactor → commit-refactor → tdd-qa → complete
```

**After:**
```
tdd-red → commit-red → tdd-green → commit-green → tdd-refactor → commit-refactor → complete
```

**Implications:**
- `tdd-qa` phase deleted from state machine definitions
- `tdd-qa` skill deprecated or removed
- TDD loop transitions directly from final commit to completion
- No end-of-loop meta-verification needed (per-phase QA is sufficient)

**Traces to:** REQ-003

---

### DES-009: Backward Compatibility - Version Gating

New tasks use per-phase QA; existing tasks continue with old flow:

**Detection mechanism:**
- Task state includes `created_at` timestamp or `schema_version` field
- Tasks created before per-phase QA deployment use old flow (no QA sub-steps)
- Tasks created after deployment use new flow (QA sub-steps enabled)

**Implementation:**
- `projctl step next` checks task schema version
- If version < 2 (or created_at < deployment date), skip QA sub-steps
- If version >= 2, use sub-step model

**No migration:**
- In-flight tasks complete with old flow
- No state transformation required

**Traces to:** REQ-007

---

### DES-010: QA Scope Boundaries

Each QA skill validates only its phase's work, not cross-phase concerns:

**tdd-red-qa scope:**
- Tests are well-formed (syntax, structure)
- Tests fail when run (red phase requirement)
- Tests fail for the correct reasons (not syntax errors or missing imports)
- Tests match requirements coverage
- Does NOT check: implementation code, refactoring quality, commit messages

**tdd-green-qa scope:**
- Implementation makes tests pass
- No over-implementation (only what tests require)
- Code follows project conventions
- Does NOT check: test quality, refactoring structure, commit staging

**tdd-refactor-qa scope:**
- Refactoring preserved behavior (tests still pass)
- Structure improved (readability, modularity)
- No new functionality added
- Does NOT check: test quality, implementation correctness beyond behavior preservation

**commit-qa scope:**
- Right files staged (no missing, no extra)
- No secrets staged (.env, credentials, keys)
- Does NOT check: commit message quality, hooks, branch state

**Traces to:** REQ-009, REQ-010

---

## State Machine Diagrams

### TDD Phase Sub-Step Flow

```
[Phase: tdd-red]
    |
    v
[producer_pending] --spawn-producer--> [producer_complete]
    |                                       |
    |                                       v
    |                              [qa_pending] --spawn-qa--> [qa_complete]
    |                                                             |
    |                                                             v
    |                                                      (verdict?)
    |                                                             |
    +------------------improvement-request (iter < 3)------------+
    |                                                             |
    |                                            approved          |
    |                                                             v
    |                                                    [commit_pending] --commit--> [commit_complete]
    |                                                                                      |
    |                                                                                      v
    +------------------escalate-user (iter >= 3)--> [user_intervention]         [transition to next phase]
```

### Commit Phase Sub-Step Flow

```
[Phase: commit-red]
    |
    v
[producer_pending] --spawn-producer--> [producer_complete]
    |                                       |
    |                                       v
    |                              [qa_pending] --spawn-qa--> [qa_complete]
    |                                                             |
    |                                                             v
    |                                                      (verdict?)
    |                                                             |
    +------------------improvement-request (iter < 3)------------+
    |                                                             |
    |                                            approved          |
    |                                                             v
    +------------------escalate-user (iter >= 3)--> [user_intervention]    [finalize] --> [transition to next phase]
```

---

## Verification Strategy

Per REQ-008, implementation correctness is verified through structural output testing:

**Test approach:**
1. Call `projctl step next` on a task in `tdd-red` phase
2. Verify action sequence: spawn-producer → spawn-qa → commit → transition
3. Verify skill names: tdd-red-producer, tdd-red-qa, commit
4. Repeat for tdd-green, tdd-refactor, commit phases

**No behavioral testing:**
- Don't run actual TDD loops end-to-end
- Don't verify QA skill correctness (separate skill verification)
- Only verify step-next returns correct action structure

**Traces to:** REQ-008

---

## Open Questions

None - user clarified this is a CLI/backend state machine design with no UI components.

---

## Design Summary

- **10 design decisions** (DES-001 through DES-010)
- **Focus:** State machine action sequences, sub-step model, QA verdict handling
- **No UI:** Orchestrator consumes JSON from `projctl step next`
- **QA timing:** Immediate feedback, automatic retry up to 3 iterations
- **Backward compatibility:** Version gating for new vs. existing tasks
- **Scope boundaries:** Each QA skill limited to its phase's concerns
