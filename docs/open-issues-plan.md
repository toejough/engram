# Open Issues Execution Plan

**Created:** 2026-02-07

## Dependency Graph

```
                    ┌──────────────────────────────┐
  QUICK WINS        │  137: model from frontmatter  │──┐
  (parallel,        │  139: fix integrate bugs      │──┤
  do first)         │  141: remove commit-QA phases │──┤── all parallel, do now
                    │  149: behavioral patterns     │──┤
                    │  151: workflow tiers           │──┘
                    └──────────────────────────────┘
                                 │
                                 ▼
                    ┌──────────────────────────────┐
  BIG RESTRUCTURE   │  150: declarative TOML        │ ← unblocks everything below
                    │  (informed by 151 tier design)│
                    └──────────────────────────────┘
                                 │
                    ┌────────────┼────────────┐
                    ▼            ▼            ▼
                  138          148       140+142+145
              plan mode    eval phase    small state
              + parallel   consolidated  machine tweaks
                                         (trivial TOML
                                          changes after
                                          150)
                    └────────────┬────────────┘
                                 │ parallel
```

## Phase 1: Quick Wins (parallel)

No dependencies on each other or on ISSUE-150. Do now, won't be obviated by later work.

| Issue | Summary | Why now |
|-------|---------|---------|
| ISSUE-137 | Read model from SKILL.md frontmatter instead of hardcoded registry | Skills keep frontmatter regardless of Go vs TOML registry |
| ISSUE-139 | Fix integrate command bugs (renumbering, ID format, path) | Command bugs, independent of state machine |
| ISSUE-141 | Remove commit-producer QA phases, consolidate commit skills | Fewer phases simplifies system now, fewer to migrate later |
| ISSUE-149 | Investigate idle-wait prevention (behavioral patterns) | CLAUDE.md/skill contract changes, no infrastructure dependency |
| ISSUE-151 | Define workflow tiers (full/task/quick-fix) | Defines tier criteria that inform ISSUE-150's TOML workflow definitions; updates CLAUDE.md |

## Phase 2: Big Restructure

| Issue | Summary | Blocks | Informed by |
|-------|---------|--------|-------------|
| ISSUE-150 | Declarative TOML state machine, streamlined API | ISSUE-138, ISSUE-148, ISSUE-140, ISSUE-142, ISSUE-145 | ISSUE-151 (tiers become TOML workflow defs) |

## Phase 3: Post-Restructure (parallel where noted)

Can proceed in parallel after ISSUE-150 completes.

| Issue | Summary | Depends on | Parallel with |
|-------|---------|------------|---------------|
| ISSUE-138 | Plan mode + parallel PM/Design/Arch collaboration | ISSUE-150 | ISSUE-148 |
| ISSUE-148 | Consolidated evaluation phase with interview | ISSUE-150 | ISSUE-138 |
| ISSUE-140 | Step next includes current_task in context | ISSUE-150 | ISSUE-142, ISSUE-145 |
| ISSUE-142 | Add TaskList creation step to control loop | ISSUE-150 | ISSUE-140, ISSUE-145 |
| ISSUE-145 | Definition of done checkpoint before retro | ISSUE-150 | ISSUE-140, ISSUE-142 |

## Deferred

| Issue | Summary | Reason |
|-------|---------|--------|
| ISSUE-108 | Trace validation after artifact creation | Likely superseded by ISSUE-148 evaluation redesign |
| ISSUE-130 | Visual testing workflow for CLI | Independent but low urgency, defer |
