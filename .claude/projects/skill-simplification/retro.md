# Project Retrospective: ISSUE-90

**Project:** Simplify orchestrator SKILL.md for step-driven execution
**Duration:** Single session (~10 minutes, 22:56-23:05 on 2026-02-05)
**Deliverables:** Rewritten `skills/project/SKILL.md`, deleted `skills/project/SKILL-full.md`, 25 tests
**Approach:** Single task workflow with TDD, 2 QA iterations

---

## Project Summary

Rewrote the project orchestrator SKILL.md to use a step-driven control loop powered by `projctl step next` / `projctl step complete` (implemented in ISSUE-89). The orchestrator is no longer the source of process truth -- `projctl` is. The SKILL.md became a mechanical executor: call `projctl step next`, do what it says, call `projctl step complete`, repeat.

### Key Metrics

- **Commits:** 4 (issue prep, failing tests, implementation, flag fix)
- **Files Modified:** 3 (`SKILL.md`, `SKILL-full.md` deleted, `SKILL_test.sh`)
- **Net Lines Removed:** ~350 (403 added, 753 removed)
- **Tests:** 25 (structure, required content, step-driven loop, removed content, CLI flags)
- **QA Iterations:** 2 (iteration 1 found 2 real bugs, iteration 2 approved)
- **SKILL-full.md eliminated:** 559 lines deleted entirely
- **Model downgrade:** Orchestrator frontmatter changed from opus to haiku

---

## What Went Well (Successes)

### S1: TDD Caught Real Integration Bugs

**Area:** Quality

The QA iteration 1 found two genuine CLI flag mismatches between SKILL.md and the actual `projctl step complete` implementation:
1. `--status retry` was used in the SKILL.md but is not a valid status value
2. `--qa-verdict approved` flag was missing from the QA approved flow

These would have caused runtime failures when the orchestrator tried to execute `projctl step complete` with invalid flags. The TDD cycle caught them: tests were added for the correct flags (commit `883ff77`), then the SKILL.md was fixed (commit `2eaef8a`).

### S2: Massive Simplification Achieved

**Area:** Architecture

The project achieved its primary goal: eliminating the orchestrator as a process knowledge holder. Concrete removals:
- **Phase dispatch tables** -- `projctl step next` returns skill names
- **PAIR LOOP pattern** -- `projctl` enforces the producer/QA/commit sequence
- **Skill dispatch tables** -- `projctl step next` returns skill paths
- **Resume map** -- `projctl` tracks sub-phase state
- **SKILL-full.md** -- 559 lines eliminated entirely

The orchestrator went from "process expert requiring opus" to "mechanical executor suitable for haiku."

### S3: Clean Single-Task Execution

**Area:** Process

The single-task workflow (`/project task`) was efficient for this scope. No parallel execution needed, no worktree management, no task dependency graphs. State machine walked cleanly through init -> tdd-red -> commit-red -> tdd-green -> commit-green -> tdd-refactor -> commit-refactor -> task-audit -> (retry once) -> task-complete. Total wall clock ~10 minutes.

### S4: Related Issues Closed Proactively

**Area:** Scope Management

ISSUE-93 (guard against duplicate role assignments) and ISSUE-94 (enforce naming convention for teammates) were both closed as addressed by ISSUE-90. The step-driven loop naturally prevents these issues: `projctl step next` returns one action at a time with the expected teammate name, eliminating the opportunity for the orchestrator to spawn duplicates or use wrong names.

---

## What Could Improve (Challenges)

### C1: QA Found Flag Mismatches That Should Have Been Caught During TDD-Red

**Area:** Test Coverage

The initial 22 tests (TDD-red phase) did not include tests for correct CLI flag usage. Tests verified that `projctl step complete` was referenced, but not that the flags matched the actual CLI interface. This gap meant the implementation (TDD-green) could pass all tests while containing invalid CLI invocations.

**Impact:** Medium. Required a QA retry cycle (adding 2 more tests and fixing the SKILL.md), adding ~2 minutes to the process.

**Root Cause:** The test author focused on structural presence ("does the file mention spawn-qa?") rather than behavioral correctness ("does the file use the right flags for step complete?"). This is the "test behavior, not just presence" lesson from CLAUDE.md applied to documentation testing.

---

## Process Improvement Recommendations

### R1: Add CLI Flag Validation to Documentation TDD Pattern

**Priority:** Medium

**Action:** When writing tests for skill documentation that references CLI commands, include tests that verify the exact flag names and values match the CLI implementation. For example, grep for `--status` values and verify they are in the set of valid values (`done`, `failed`), not invented ones (`retry`).

**Rationale:** Would have caught both QA findings (C1) during TDD-red, avoiding the retry cycle. Documentation that references CLI commands is an interface boundary -- flag mismatches are integration bugs.

**Measurable Outcome:** Zero QA findings related to CLI flag mismatches in SKILL.md files.

**Area:** Testing

### R2: Consider Generating SKILL.md Step Examples from CLI Source

**Priority:** Low

**Action:** Explore whether `projctl` could generate example `step complete` invocations that SKILL.md references, ensuring they stay in sync with the CLI implementation.

**Rationale:** Manual synchronization between SKILL.md examples and CLI flags is error-prone. Auto-generation would make mismatches impossible.

**Measurable Outcome:** SKILL.md examples always match CLI interface.

**Area:** Tooling

---

## Open Questions

None. The scope was well-defined and execution was clean.

---

## Traceability

**Traces to:**
- ISSUE-90 (parent issue)
- ISSUE-89 (prerequisite -- provided `projctl step next/complete`)
- ISSUE-93 (closed -- addressed by step-driven loop)
- ISSUE-94 (closed -- addressed by step-driven loop)
- Commits: `d538a36` (failing tests), `a49df9e` (implementation), `883ff77` (flag tests), `2eaef8a` (flag fix)
- Files: `skills/project/SKILL.md`, `skills/project/SKILL-full.md` (deleted), `skills/project/SKILL_test.sh`

---

## Conclusion

ISSUE-90 completed the architectural pivot started in ISSUE-89. The orchestrator SKILL.md is now a thin mechanical loop -- call `projctl step next`, execute, call `projctl step complete`, repeat. Process knowledge lives in Go code (the phase registry), not in prompt text. The model downgrade from opus to haiku validates that the orchestrator's job is now simple enough for a smaller model.

The one process gap was insufficient TDD coverage for CLI flag correctness in documentation, caught by QA in iteration 1 and immediately fixed. This is a learnable pattern: when docs reference CLI commands, test the exact flags.
