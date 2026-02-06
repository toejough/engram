# Project Summary: Phase 2 Team Migration

**Project:** Phase 2 Team Migration
**Issues:** ISSUE-73 through ISSUE-80
**Date:** 2026-02-05
**Branch:** project-skill-simplification
**Workflow:** task

---

## Executive Overview

Phase 2 completed the migration of all projctl skills from the legacy yield TOML protocol to Claude Code's native team mode. This involved updating 15 skill SKILL.md files with SendMessage/AskUserQuestion communication, adding TaskList-based coordination and parallel execution documentation to the orchestrator, and removing ~2,600 lines of legacy yield Go infrastructure. The project closed 8 issues with zero rework cycles and a net reduction of ~2,100 lines of code.

Phase 2 built on Phase 1 (ISSUE-69 through ISSUE-72), which established the pattern by migrating the orchestrator, PM interview producer, QA skill, and shared templates. Phase 2 applied that pattern to the remaining 15 skills, added runtime coordination features, and removed the now-unused legacy infrastructure.

---

## Key Decisions

### D1: Backward-Compatible Migration

All skills retain legacy yield as a fallback alongside the new team-mode Communication sections. This allows incremental adoption and prevents breakage if any skill is invoked outside the team context.

**Traces to:** ISSUE-73, ISSUE-74, ISSUE-75, ISSUE-76

### D2: TaskList as Runtime Coordination Only

Claude Code's native TaskList is used for runtime task coordination during the implementation phase, but docs/tasks.md remains the canonical traced artifact. TaskList entries carry TASK-NNN metadata for cross-reference but do not replace the traced task definitions.

**Traces to:** ISSUE-78

### D3: Native Team Parallelism Replaces parallel-looper

Instead of maintaining a custom parallel-looper skill, the orchestrator now uses Claude Code's native team spawning to run independent tasks concurrently. Each teammate gets a git worktree for isolation, with merge-on-complete to preserve work.

**Traces to:** ISSUE-79

### D4: Memory Package Yield Parsing Preserved

The memory package's yield parsing code (`internal/memory/parse.go`) was preserved during the yield infrastructure removal. It serves a different purpose (extracting structured data from historical yield files) than the protocol infrastructure that was removed.

**Traces to:** ISSUE-80

### D5: ISSUE-77 Closed Without Changes

The orchestrator wiring (ISSUE-77) was identified as already complete from ISSUE-69 in Phase 1. Rather than inventing unnecessary work, the issue was correctly scoped and closed with a comment explaining why no changes were needed.

**Traces to:** ISSUE-77

---

## Outcomes

### Features Delivered

| Issue | Deliverable | Files Changed |
|-------|-------------|---------------|
| ISSUE-73 | Design + arch interview producers migrated to AskUserQuestion + SendMessage | 2 skill files |
| ISSUE-74 | Inference producers (pm, design, arch) migrated to direct tool + SendMessage | 3 skill files |
| ISSUE-75 | Remaining producers (breakdown, doc, alignment, retro, summary) migrated | 5 skill files |
| ISSUE-76 | TDD skills (producer, red, green, refactor, red-infer) migrated | 5 skill files |
| ISSUE-77 | Orchestrator wiring confirmed complete (no changes needed) | 1 issue update |
| ISSUE-78 | TaskList-based implementation coordination added to orchestrator | 2 skill files |
| ISSUE-79 | Native parallel task execution documented in orchestrator | 2 skill files |
| ISSUE-80 | Legacy yield infrastructure removed (~2,600 lines of Go code) | 13 files deleted/modified |

### Metrics

- **31 files changed** across 10 commits
- **568 insertions, 2,681 deletions** (~2,100 lines net removed)
- **15 skill files** updated with team-mode Communication sections
- **13 Go files** deleted or modified to remove yield infrastructure
- **0 QA failures** -- all skills passed QA on first iteration
- **4 follow-up issues** created (ISSUE-85 through ISSUE-88)

### Quality

- All QA checks passed first iteration across all 8 issues
- `go test ./...` passes after yield removal
- No dangling dependencies from removed yield packages
- Backward compatibility maintained via retained legacy yield sections in skills

### Known Limitations

- State machine phase transitions still require manual `projctl state transition` commands; they don't auto-detect team-mode completion signals (see ISSUE-85, R3 in retro)
- Legacy yield sections remain in skill files as fallback; full removal deferred pending team-mode stability confirmation
- Stale yield references may exist in non-code documentation (see ISSUE-88)

---

## Lessons Learned

### Process

- **Parallel execution validated:** Four independent skill migration issues (ISSUE-73 through ISSUE-76) ran simultaneously without conflicts, validating the parallel execution model that the migration itself was enabling.
- **Established patterns reduce rework:** The Phase 1 migration pattern was consistently applied across all Phase 2 skills with zero QA failures, demonstrating that investing in pattern establishment pays off.
- **Process enforcement needs automation:** The orchestrator skipped QA and end-of-project phases until reminded by the user. Process steps that depend on agent memory are unreliable; they need explicit checklists or automated enforcement (ISSUE-85).

### Technical

- **Clean separation enables clean removal:** The yield infrastructure was fully excised without affecting remaining functionality, confirming good separation of concerns in the original design.
- **Skill frontmatter must drive spawning:** QA teammates were spawned on the wrong model (opus instead of haiku) because the orchestrator didn't read the skill's SKILL.md frontmatter. The model field exists precisely for this purpose (ISSUE-86).
- **State machine needs team-mode adaptation:** The state machine was designed for sequential yield-based workflows. Team mode's parallel execution and message-based completion signals create friction with manual phase transitions.

---

## Follow-Up Issues

| Issue | Priority | Description |
|-------|----------|-------------|
| ISSUE-85 | High | Enforce process checklist in orchestrator (from R1) |
| ISSUE-86 | Medium | Auto-read skill model from frontmatter before spawning (from R2) |
| ISSUE-87 | Medium | Consolidate Phase 1/2 migration memory notes (from Q1) |
| ISSUE-88 | Medium | Clean up remaining yield references in docs (from Q2) |

---

## Traceability

**Traces to:**
- ISSUE-73, ISSUE-74, ISSUE-75, ISSUE-76, ISSUE-77, ISSUE-78, ISSUE-79, ISSUE-80
- ISSUE-85, ISSUE-86, ISSUE-87, ISSUE-88 (follow-up issues)
- Phase 1 migration (ISSUE-69 through ISSUE-72) as prerequisite
- docs/retro-phase2.md (detailed retrospective)
- skills/project/SKILL.md, skills/project/SKILL-full.md (orchestrator updates)
- Commits: 60e9f95, 1b9628c, b8f2cc2, 0f0a749, 6635947, 9d8145c, d985c89, 8b5b987, 0dd6365, ee5fa15, aa32a91
