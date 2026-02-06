# Project Retrospective: Phase 2 Team Migration

**Project:** Phase 2 Team Migration (ISSUE-73 through ISSUE-80)
**Duration:** Single session (2026-02-05)
**Deliverables:** Full skill migration to Claude Code native team mode, legacy yield removal
**Approach:** Parallel migration of 4 skill groups, sequential cleanup and infrastructure work

---

## Project Summary

Migrated all projctl skills from the legacy yield TOML protocol to Claude Code's native team mode (SendMessage + AskUserQuestion). Added TaskList-based coordination, documented native parallel execution patterns, and removed ~2,600 lines of legacy yield infrastructure code.

### Scope

- 8 issues (ISSUE-73 through ISSUE-80) spanning 4 migration phases
- 17 skill files modified across interview, inference, general, and TDD producer categories
- 31 files changed total: 568 insertions, 2,681 deletions
- 10 commits on the branch
- Net removal of ~2,100 lines of code

### Issues Completed

| Issue | Description | Result |
|-------|-------------|--------|
| ISSUE-73 | Design + arch interview producers | Migrated to AskUserQuestion + SendMessage |
| ISSUE-74 | Inference producers (pm, design, arch) | Migrated to direct tool + SendMessage |
| ISSUE-75 | Remaining producers (breakdown, doc, alignment, retro, summary) | Migrated to SendMessage |
| ISSUE-76 | TDD skills (producer, red, green, refactor, red-infer) | Migrated to SendMessage |
| ISSUE-77 | Wire all phases into orchestrator | Already complete from ISSUE-69 (no changes) |
| ISSUE-78 | TaskList-based implementation coordination | Added to orchestrator |
| ISSUE-79 | Native parallel task execution docs | Documented in SKILL-full.md |
| ISSUE-80 | Remove legacy yield infrastructure | Deleted ~2,600 lines of Go code and tests |

---

## What Went Well (Successes)

### S1: Effective Parallel Execution

**Area:** Process Efficiency

ISSUE-73 through ISSUE-76 were independent skill migrations that could run simultaneously. Four producers executed in parallel without conflicts or merge issues. This validated the parallel execution model that the migration itself was enabling.

### S2: All QA Passed First Iteration

**Area:** Quality Assurance

Every migrated skill passed QA on the first attempt. No rework cycles were needed for any of the 8 issues. This indicates the migration pattern was well-established from Phase 1 (ISSUE-69 through ISSUE-72) and consistently applied across all skill categories.

### S3: Clean Yield Removal

**Area:** Code Quality

Removing the legacy yield infrastructure (ISSUE-80) was clean with no dangling dependencies. The Go packages (`internal/yield`, `internal/context` yield paths, `cmd/projctl/yield.go`) were fully excised without affecting any remaining functionality. This confirms good separation of concerns in the original design.

### S4: Net Code Reduction

**Area:** System Simplicity

The entire Phase 2 effort resulted in a net reduction of ~2,100 lines. The new team mode communication sections added to skills (~570 lines across 17 files) are far simpler than the yield TOML infrastructure they replaced (~2,680 lines of Go code, tests, and CLI commands). Simpler protocol, less code, fewer moving parts.

### S5: ISSUE-77 Was Already Done

**Area:** Prior Work

ISSUE-77 (wire all phases into orchestrator) turned out to already be complete from ISSUE-69 in Phase 1. Rather than inventing unnecessary work, this was correctly identified and closed without changes. This shows good scoping discipline.

---

## What Could Improve (Challenges)

### C1: Orchestrator Did Not Follow Its Own Process

**Area:** Process Discipline

The orchestrator (team lead) initially skipped parts of the project process:
- No QA was run on the first attempt until the user corrected the behavior
- The main flow ending (retro/summary/next-steps) was initially skipped until the user reminded

This pattern of needing user nudges to follow the defined process reduces the value of having a process at all.

**Impact:** Medium. The user had to intervene twice to keep the process on track.

**Root Cause:** The orchestrator's SKILL.md may not be explicit enough about mandatory QA steps and end-of-project phases. Or the orchestrator may be treating them as optional when under time/context pressure.

### C2: QA Teammates Spawned on Wrong Model

**Area:** Resource Efficiency

QA teammates were accidentally spawned using opus instead of haiku. The QA skill explicitly specifies `model: haiku` in its frontmatter, but the team lead did not pass this model parameter when spawning QA agents.

**Impact:** Low (cost only, not quality). The QA work completed correctly but used a more expensive model than necessary.

**Root Cause:** The spawning agent didn't check the skill's frontmatter for model specification before creating the teammate.

### C3: State Machine Management Was Manual

**Area:** Tooling

The state machine had to be manually walked through TDD states to advance past the implementation phase. This friction suggests the state machine's phase transitions don't map well to the team-mode workflow.

**Impact:** Low. Added manual steps but didn't block progress.

**Root Cause:** The state machine was designed for the yield-based sequential workflow. Team mode introduces parallel execution and different completion signals that the state machine doesn't natively support.

---

## Process Improvement Recommendations

### R1: Enforce Process Checklist in Orchestrator

**Priority:** High

**Action:** Add an explicit, non-optional process checklist to the project orchestrator SKILL.md that must be completed for every issue: (1) Execute producer, (2) Run QA on output, (3) Commit changes. The orchestrator should not advance to the next issue until all three steps are confirmed.

**Rationale:** Would prevent the process skipping observed in C1. The orchestrator needs guardrails to follow its own defined process consistently.

**Measurable Outcome:** Zero user interventions needed to remind orchestrator of mandatory process steps.

**Area:** Orchestrator Process

### R2: Auto-Read Skill Model from Frontmatter Before Spawning

**Priority:** Medium

**Action:** When the orchestrator spawns a teammate for a skill, it should read the target skill's SKILL.md frontmatter and use the specified `model` field. Document this as a mandatory step in the orchestrator's teammate spawning procedure.

**Rationale:** Would prevent the wrong-model spawning observed in C2. The model information is already documented in each skill's frontmatter; it just needs to be read.

**Measurable Outcome:** All spawned teammates use the model specified in their skill's frontmatter.

**Area:** Resource Efficiency

### R3: Simplify State Machine for Team Mode

**Priority:** Low

**Action:** Evaluate whether the state machine needs simplification now that team mode replaces the yield-based workflow. Consider whether some phase transitions can be auto-detected from teammate completion messages rather than requiring manual `projctl state transition` commands.

**Rationale:** The manual state management friction in C3 suggests the state machine was designed for a different execution model. Team mode's message-based completion signals could drive transitions automatically.

**Measurable Outcome:** Fewer manual `projctl state transition` commands needed per project.

**Area:** Tooling

---

## Open Questions

### Q1: Should Phase 1 and Phase 2 Memory Notes Be Consolidated?

**Context:** Both Phase 1 and Phase 2 of the team migration added memory notes (e.g., "QA teammates must use model: haiku"). These notes are scattered across the migration sessions. A consolidation pass would ensure all lessons are properly captured in MEMORY.md.

### Q2: Are There Remaining Yield References in Non-Skill Files?

**Context:** ISSUE-80 removed the Go infrastructure, but there may be stale references to yield concepts in documentation, CLAUDE.md, or other non-code files. A grep for "yield" across the repo would identify any cleanup needed.

---

## Traceability

**Traces to:**
- ISSUE-73, ISSUE-74, ISSUE-75, ISSUE-76, ISSUE-77, ISSUE-78, ISSUE-79, ISSUE-80
- skills/project/SKILL.md (orchestrator updates)
- skills/project/SKILL-full.md (TaskList and parallel execution docs)
- Phase 1 migration (ISSUE-69 through ISSUE-72) as prerequisite
- Commits: 60e9f95, 1b9628c, b8f2cc2, 0f0a749, 6635947, 9d8145c, d985c89, 8b5b987, 0dd6365, ee5fa15

---

## Conclusion

Phase 2 completed the full migration of projctl skills from yield TOML to Claude Code's native team mode. The migration was clean, parallel execution worked well, and the net result is a simpler system with ~2,100 fewer lines of code. The main process improvement needed is for the orchestrator to consistently follow its own process without user reminders -- the defined workflow is good, but enforcement is lacking. The secondary issue of model selection for spawned teammates is a simple fix that prevents unnecessary resource usage.
