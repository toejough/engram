# Retrospective: ISSUE-98 — Model Validation for Teammate Spawning

## Project Summary

**Issue:** ISSUE-98 — How do we launch teammates with the right models?
**Duration:** ~4.5 hours (07:18 - 11:31)
**Tasks:** 6 tasks, all completed
**Workflow:** Full lifecycle — PM, Design, Architecture, Breakdown, Implementation (TDD), Documentation, Alignment, Retro

The project added model validation to the teammate spawning pipeline: literal `TaskParams` in `step next` output, a model handshake protocol, retry/escalation logic, and orchestrator SKILL.md instructions. This directly addresses wrong-model spawns observed in ISSUE-89.

---

## What Went Well

### S1: Clean task decomposition and dependency graph
The 6-task breakdown had clear dependencies (TASK-1/TASK-2 independent, TASK-3 depends on TASK-1, TASK-4 depends on both, etc.) and the rationale for why not fewer/more tasks was documented. Implementation proceeded smoothly through the dependency order.

### S2: Multiple QA cycles caught real issues
QA iteration caught concrete problems: substring vs exact match semantics, missing ARCH entry, missing ISSUE-98 documentation in README, and h3 header format mismatch for trace validation. These would have been defects if not caught before merge.

### S3: Backward compatibility via zero values (ADR-2)
Using Go's zero-value semantics for new `PairState` fields (`SpawnAttempts`, `FailedModels`) eliminated the need for migration. Existing `state.toml` files work unchanged. This was a good architectural decision that reduced scope.

### S4: All 6 tasks completed in implementation phase
No tasks were escalated or deferred. The scope was well-calibrated — each task was small enough to complete in one pass while being large enough to represent a meaningful unit of work.

---

## Challenges

### C1: TASK-2 worktree work was lost
A teammate working on TASK-2 in a worktree did not commit their work before the worktree was cleaned up. The work had to be redone manually. This is a recurring risk with worktree-based parallelism.

**Impact:** Medium — TASK-2 was small (~5 lines), so rework was fast, but this could be costly for larger tasks.

### C2: Model mismatch during project execution
QA agents were sometimes spawned on opus despite haiku being requested — the exact problem ISSUE-98 was designed to fix. This is ironic but expected since the fix wasn't deployed yet during its own development.

**Impact:** Low — wasted some compute but didn't block progress.

### C3: tdd-producer runs all TDD phases in one agent
ISSUE-99 was filed: the `tdd-producer` composite skill runs RED+GREEN+REFACTOR in a single agent instead of orchestrating separate phase agents. This means a single agent does all the work, which defeats the purpose of phase separation and model-appropriate assignment.

**Impact:** Medium — affects all implementation tasks, not just ISSUE-98. Filed as separate issue.

### C4: Teammates didn't always commit in worktrees
Beyond TASK-2, there was a general pattern of teammates not committing their work in worktrees before completion. This risks data loss when worktrees are cleaned up.

**Impact:** Medium — systematic risk affecting all worktree-based parallel work.

---

## Recommendations

### R1: Enforce commit-before-completion in worktree teammates (ISSUE-100)
**Priority:** High
**Action:** Add explicit instructions in teammate spawn prompts requiring a commit before sending completion message. Validate that the worktree has no uncommitted changes before accepting completion.
**Rationale:** C1 and C4 show this is a recurring data loss risk. The fix is procedural (prompt instructions) plus validation (check `git status` on worktree before cleanup).

### R2: Fix tdd-producer to orchestrate separate TDD phase agents (ISSUE-99)
**Priority:** Medium
**Action:** Address ISSUE-99. The tdd-producer should spawn separate agents for RED, GREEN, and REFACTOR phases rather than running all three in a single agent.
**Rationale:** C3 — running all phases in one agent undermines model-appropriate assignment (e.g., haiku for simple green phase, opus for complex refactoring).

### R3: Add worktree commit verification to projctl (ISSUE-101)
**Priority:** Medium
**Action:** Add a `projctl worktree verify` command (or integrate into `step complete`) that checks for uncommitted changes in a teammate's worktree before accepting task completion.
**Rationale:** C1 and C4 — procedural instructions alone aren't sufficient; automated verification catches what instructions miss.

---

## Open Questions

### Q1: Should model validation be enforced for all teammate spawns or only for specific phases? (ISSUE-102)
**Context:** ISSUE-98 adds validation for all spawn actions. But some phases (e.g., doc-producer) may not be model-sensitive. Enforcing validation everywhere adds latency (handshake round-trip) for potentially low-risk spawns. Is blanket enforcement the right default, or should phases opt in/out?

---

## Metrics

| Metric | Value |
|--------|-------|
| Total tasks | 6 |
| Tasks completed | 6 |
| Tasks escalated | 0 |
| QA iteration rounds (observed) | Multiple (caught 4+ issues) |
| Commits | 12 (1 issue setup + 4 artifact docs + 6 implementation + 1 alignment fix) |
| Phase duration: PM | ~1.5h (07:18 - 08:56) |
| Phase duration: Design | ~49min (08:56 - 09:45) |
| Phase duration: Architecture | ~6min (09:45 - 09:51) |
| Phase duration: Breakdown | ~5min (09:51 - 09:56) |
| Phase duration: Implementation | ~1.25h (09:56 - 11:12) |
| Phase duration: Documentation | ~10min (11:13 - 11:22) |
| Phase duration: Alignment | ~8min (11:22 - 11:31) |

---

## Decisions Made

- **ADR-1:** Validation logic in SKILL.md instructions, not Go code — keeps Go focused on state management
- **ADR-2:** Backward compatibility via Go zero-value semantics — no migration needed
- **ADR-3:** Status branching on existing actions rather than new action types — minimizes CLI surface changes
