# Project Summary: ISSUE-90 -- Simplify Orchestrator SKILL.md for Step-Driven Execution

## Executive Overview

ISSUE-90 completed the architectural pivot started in ISSUE-89. Where ISSUE-89 made `projctl` the planner (via `projctl step next` / `projctl step complete`), ISSUE-90 made the orchestrator SKILL.md a mechanical executor of that plan. The orchestrator no longer contains phase dispatch tables, skill registries, or process knowledge -- it runs a step loop: call `projctl step next`, execute the returned action, call `projctl step complete`, repeat.

**Scope:** Single task workflow, single session (2026-02-05, ~10 minutes).

**Deliverables:**
- Rewritten `skills/project/SKILL.md` (262 lines, down from ~207)
- Deleted `skills/project/SKILL-full.md` (559 lines eliminated)
- 25 tests in `skills/project/SKILL_test.sh`

**Outcome:** All acceptance criteria met. QA approved after 2 iterations (2 CLI flag bugs found in iteration 1, fixed in iteration 2). Model downgraded from opus to haiku.

**Traces to:** REQ-001, ARCH-012, ARCH-013

---

## Key Decisions

### 1. Model Downgrade from Opus to Haiku

**Context:** The orchestrator previously ran on opus because it needed to interpret complex phase dispatch tables, manage resume logic, and make judgment calls about skill routing.

**Choice:** With process knowledge moved into `projctl step next`, the orchestrator's job is mechanical: parse JSON, spawn a teammate, report completion. This is simple enough for the cheapest model (haiku).

**Evidence:** The frontmatter `model: haiku` was set and the orchestrator executed the full TDD cycle (init through summary) without issue.

**Traces to:** ARCH-012

---

### 2. Eliminate SKILL-full.md Entirely

**Context:** SKILL-full.md (559 lines) served as the extended reference containing the full phase registry, resume map, and looper pattern details. SKILL.md was the compact version that referenced it.

**Options considered:**
1. Keep SKILL-full.md as a reference document
2. Delete it -- `projctl step next` is now the source of truth

**Choice:** Delete. The phase registry, resume map, and skill dispatch tables now live in Go code (`internal/step/registry.go`). Keeping a prose copy would create drift risk with zero benefit.

**Outcome:** 559 lines removed. Zero information loss -- all process data is in the `projctl` codebase.

**Traces to:** ARCH-012

---

### 3. Retain Looper Pattern and Context-Only Contract in SKILL.md

**Context:** Some SKILL.md content is judgment-based rather than mechanical: the looper pattern (parallel task execution with worktrees) and the context-only contract (ISSUE-53 lesson about not overriding skill behavior).

**Choice:** Keep these sections. They encode coordination patterns that `projctl step next` does not yet manage -- looper parallelism and teammate prompt construction are still orchestrator responsibilities.

**Traces to:** ARCH-013

---

## Outcomes and Deliverables

### What Was Removed

| Removed Content | Replacement |
|----------------|-------------|
| Phase dispatch tables | `projctl step next` returns skill name and path |
| PAIR LOOP pattern | `projctl` enforces producer -> QA -> commit sequence |
| Skill dispatch tables | `projctl step next` returns skill path |
| Resume map | `projctl` tracks sub-phase state via PairState |
| SKILL-full.md (559 lines) | Phase registry in `internal/step/registry.go` |

### What Was Added

| New Content | Purpose |
|-------------|---------|
| Step-driven control loop | Mechanical loop: `step next` -> execute -> `step complete` |
| Step next JSON output example | Shows orchestrator what to expect from `projctl step next` |
| Action handler sections | Exact `projctl step complete` invocations per action type |

### Quality Metrics

| Metric | Value |
|--------|-------|
| Total tests | 25 |
| QA iterations | 2 (2 findings in iteration 1, 0 in iteration 2) |
| Net lines removed | ~211 (542 added, 753 removed across all files) |
| Files modified | 2 (SKILL.md rewritten, SKILL_test.sh expanded) |
| Files deleted | 1 (SKILL-full.md) |
| Commits | 5 (failing tests, implementation, flag tests, flag fix, retro) |

### QA Findings and Resolution

| Finding | Root Cause | Fix |
|---------|-----------|-----|
| `--status retry` used in SKILL.md | Not a valid `step complete` status value | Changed to valid flow (re-enter loop) |
| `--qa-verdict approved` missing from QA approved flow | Incomplete flag specification | Added `--qa-verdict approved` flag |

Both were caught by QA iteration 1 and fixed with additional tests (commit `883ff77`) and implementation fix (commit `2eaef8a`).

---

## Superseded Issues

| Issue | Relationship | Reason |
|-------|-------------|--------|
| ISSUE-93 | Closed | Step-driven loop returns one action at a time; no opportunity for duplicate teammates |
| ISSUE-94 | Closed | `projctl step next` includes teammate name, enforcing naming convention |

---

## Follow-Up Issues Created

| Issue | Title | Priority |
|-------|-------|----------|
| ISSUE-97 | Add CLI flag validation to documentation TDD pattern | Medium |

---

## Lessons Learned

### L1: Test CLI Flag Correctness in Documentation

When SKILL.md references CLI commands with specific flags, tests must verify the exact flag names and values against the actual CLI interface. Structural tests ("does the file mention `step complete`?") are necessary but insufficient -- behavioral tests ("does it use `--status done` not `--status retry`?") catch integration bugs at the documentation layer.

### L2: Process Knowledge Belongs in Code, Not Prose

Moving phase dispatch, skill routing, and resume logic from SKILL.md prose into Go code (`internal/step/`) made the orchestrator dramatically simpler. Prose is ambiguous and unenforceable; code is deterministic and testable. The model downgrade from opus to haiku is direct evidence of reduced complexity.

### L3: Single-Task Workflow Is Efficient for Focused Scope

The full TDD cycle (init through summary) completed in ~10 minutes with zero blocked tasks. For well-scoped changes to a single artifact, the single-task workflow avoids the overhead of parallel coordination.

---

## Traceability

**Traces to:**
- REQ-001 (Dependable Agent Orchestrator)
- ARCH-012 (Deterministic Workflow Enforcement)
- ARCH-013 (Relentless Continuation)
- ISSUE-90 (parent issue)
- ISSUE-89 (prerequisite -- provided `projctl step next/complete`)

**Implementation artifacts:**
- `skills/project/SKILL.md` -- Rewritten orchestrator (262 lines)
- `skills/project/SKILL-full.md` -- Deleted (559 lines removed)
- `skills/project/SKILL_test.sh` -- Test suite (25 tests)

**Related documents:**
- `docs/retro-issue90.md` -- Project retrospective (in `.claude/projects/skill-simplification/retro.md`)
- `docs/summary-issue89.md` -- Predecessor project summary
- Commits: d538a36, a49df9e, 883ff77, 2eaef8a, 1b51b04
