# Project Retrospective: ISSUE-89

**Project:** Deterministic Orchestration via `projctl step next`
**Duration:** Single session (2026-02-05)
**Deliverables:** `internal/step/` package, `projctl step next` and `projctl step complete` CLI commands
**Approach:** Single task workflow with TDD, 2 QA iterations

---

## Project Summary

Implemented `projctl step next` -- deterministic orchestration that moves process logic from SKILL.md (soft, LLM may skip) into projctl code (hard, deterministic). Instead of the LLM deciding what to do next, `projctl step next` returns exactly one structured JSON action: spawn-producer, spawn-qa, commit, or transition. `projctl step complete` records results and advances sub-phase state.

This is the architectural pivot where the LLM becomes executor and projctl becomes planner.

### Key Metrics

- **Commits:** 2 (implementation + follow-up issues)
- **Files Created:** 7 (5 Go source + 2 test files)
- **Lines Added:** 1,407
- **Tests:** 49 (including 2 property-based via rapid), all passing
- **QA Iterations:** 2 (3 findings in iteration 1, all fixed in iteration 2)
- **Phase Registry:** 22 phases across 4 workflows (new, adopt, align, task)
- **Follow-up Issues Created:** 7 (ISSUE-90, ISSUE-91, ISSUE-92, ISSUE-93, ISSUE-94, ISSUE-95, ISSUE-96)

---

## What Went Well (Successes)

### S1: Clean Build on Existing Infrastructure

**Area:** Architecture

The implementation reused the existing `PairState` infrastructure from `internal/state/` rather than inventing new state tracking. This kept the change additive -- no modifications to existing packages were needed, only new files in `internal/step/` and a thin CLI wrapper in `cmd/projctl/step.go`.

### S2: Comprehensive Phase Registry

**Area:** Implementation

The phase registry covers all 22 producer/QA phases across 4 workflows (new, adopt, align, task). Each entry captures producer name, QA name, skill paths, models, artifact filename, and completion phase. This is static Go code, not runtime config, making it easy to validate at compile time.

### S3: All QA Findings Resolved in Second Iteration

**Area:** Quality

QA iteration 1 found 3 issues:
1. Dead code (`subphase.go` created then unused)
2. Missing test for `getPair` helper
3. Incomplete registry (missing `align-documentation` and `task-documentation` phases)

All three were fixed in iteration 2, which received unconditional approval. The QA process caught real issues -- the missing registry entries would have caused runtime failures for those workflows.

### S4: Property-Based Tests for Core Logic

**Area:** Testing

Two property-based tests (via rapid) verify invariants across random inputs:
- Registry property: every registered phase has non-empty Producer, QA, and CompletionPhase
- Next property: Next() never returns an error for valid states with known phases

These catch edge cases that hand-picked examples miss.

---

## What Could Improve (Challenges)

### C1: QA Teammate Spawning Errors

**Area:** Process Discipline

Multiple QA-related process mistakes occurred:
- First QA teammate was spawned with ad-hoc instructions instead of invoking the /qa skill
- First QA teammate was named "qa-auditor" instead of following the `<phase>-qa` naming convention
- Had to respawn with correct skill invocation

**Impact:** Medium. Wasted a teammate spawn and introduced delay.

**Root Cause:** The orchestrator did not consult the skill's frontmatter or naming conventions before spawning. This is the same issue identified in C2 of the Phase 2 retrospective -- the lesson was documented but not internalized.

### C2: Duplicate QA Teammates Running Simultaneously

**Area:** Team Coordination

Two QA teammates ran simultaneously due to failed shutdown timing. The old QA teammate completed its review while the tdd-producer-v2 was still writing files concurrently, producing an unreliable approval.

**Impact:** Medium. Unreliable QA approval means the quality gate was not trustworthy for that iteration.

**Root Cause:** Shutdown requests are asynchronous. When a teammate is mid-turn, the shutdown may not take effect before a new teammate is spawned for the same role. No guard exists to prevent duplicate role assignments.

### C3: Manual State Machine Walking for TDD Phases

**Area:** Tooling

The state machine had to be manually walked through TDD sub-phases because the composite `tdd-producer` skill does the full red-green-refactor cycle internally, but the state machine tracks each sub-phase individually. This created friction between the skill's execution model and the state tracking.

**Impact:** Low. Added manual commands but did not block progress.

**Root Cause:** `projctl step next` was designed to orchestrate individual sub-phases, but during this project, the higher-level composite TDD skill was used instead. The two models are mismatched -- this is exactly what ISSUE-92 (per-phase QA in TDD loop) is meant to resolve.

### C4: Dead Code Created Then Removed

**Area:** Implementation Discipline

A `subphase.go` file was created during implementation then identified as dead code by QA -- the `Next()` function uses `PairState` fields directly rather than going through a separate sub-phase abstraction. The file had to be deleted in the second iteration.

**Impact:** Low. Caught by QA and removed cleanly.

**Root Cause:** Premature abstraction. The sub-phase concept was designed before the implementation revealed it was unnecessary. Writing tests first (TDD) should have surfaced this earlier, but the abstraction was created alongside the initial implementation rather than driven by failing tests.

---

## Process Improvement Recommendations

### R1: Guard Against Duplicate Role Assignments (ISSUE-93)

**Priority:** High

**Action:** Before spawning a teammate for a role, verify no active teammate already holds that role. This could be a check in the orchestrator's spawning logic or a `projctl` command that lists active teammates per role.

**Rationale:** Would prevent the duplicate QA issue in C2. Two agents doing the same job wastes resources and produces unreliable results.

**Measurable Outcome:** Zero instances of duplicate teammates for the same role in a session.

**Area:** Team Coordination

### R2: Enforce Naming Convention for Teammates (ISSUE-94)

**Priority:** Medium

**Action:** Document and enforce the `<phase>-<role>` naming convention (e.g., `tdd-qa`, `retro-producer`) in the orchestrator SKILL.md. When `projctl step next` returns an action, include the expected teammate name in the output.

**Rationale:** Would prevent the naming confusion in C1. Consistent naming makes task assignment and shutdown coordination reliable.

**Measurable Outcome:** All spawned teammates follow the `<phase>-<role>` naming pattern.

**Area:** Process Discipline

### R3: Resolve TDD Sub-Phase vs. Composite Skill Mismatch

**Priority:** High

**Action:** ISSUE-92 addresses this. Either (a) `projctl step next` drives individual red/green/refactor sub-phases with per-phase QA, or (b) a composite TDD action is added to the step registry. The current hybrid (step tracks sub-phases, skill runs composite) should not persist.

**Rationale:** Would eliminate the manual state walking in C3 and align the execution model with the state tracking model.

**Measurable Outcome:** TDD phases advance automatically via `projctl step next/complete` without manual `projctl state transition` commands.

**Area:** Tooling

---

## Open Questions

### Q1: Should the Phase Registry Be Runtime Config Instead of Static Code? (ISSUE-95)

**Context:** Currently the registry is a Go map literal in `registry.go`. Adding a new skill or changing a model requires recompiling. A TOML/YAML config file would allow changes without rebuilding, but adds parsing complexity and runtime failure modes.

**Tradeoff:**
- **Static (current):** Compile-time validation, no config file to get out of sync, simple
- **Dynamic:** Easier to update, could be extended by users, but adds failure modes

**Recommendation:** Keep static for now. The registry changes infrequently and compile-time safety is worth the rebuild cost.

### Q2: How Should `step complete` Handle Failures? (ISSUE-96)

**Context:** Currently `step complete` accepts `status: failed` but doesn't do anything special with it. A failed producer or QA should probably trigger different behavior (retry? escalate? block?).

**Consideration:** The orchestrator currently handles failures ad-hoc. `projctl step next` should eventually encode failure recovery paths so the LLM doesn't have to decide how to handle failures.

---

## Traceability

**Traces to:**
- ISSUE-89 (parent issue)
- ISSUE-84 (partially superseded -- hooks approach replaced by step commands)
- ISSUE-86 (superseded -- model from frontmatter now embedded in registry)
- ISSUE-90 (follow-up: simplify orchestrator SKILL.md)
- ISSUE-91 (follow-up: rename task-audit to tdd-qa)
- ISSUE-92 (follow-up: per-phase QA in TDD loop)
- ISSUE-93 (follow-up: guard against duplicate role assignments, from R1)
- ISSUE-94 (follow-up: enforce naming convention for teammates, from R2)
- ISSUE-95 (decision: phase registry static vs runtime config, from Q1)
- ISSUE-96 (decision: step complete failure handling, from Q2)
- Commits: 732f998, c7fa14f
- Files: `internal/step/next.go`, `internal/step/registry.go`, `internal/step/step.go`, `cmd/projctl/step.go`

---

## Conclusion

ISSUE-89 delivered the foundational infrastructure for deterministic orchestration. The LLM no longer decides what to do next -- `projctl step next` returns exactly one action with all context needed to execute it. This is the architectural pivot that makes the process reliable: QA cannot be skipped (commit action checks QA approval), models are specified per-phase, and the full phase graph is encoded in Go.

The main process issues were around team coordination -- duplicate QA teammates and inconsistent naming -- rather than implementation quality. The implementation itself was clean, well-tested (49 tests including property-based), and approved after one QA rework cycle that caught genuine issues (dead code, missing registry entries).

The three follow-up issues (ISSUE-90, ISSUE-91, ISSUE-92) define the path from this foundation to a fully step-driven orchestrator where the SKILL.md becomes a simple executor rather than the source of process truth.
