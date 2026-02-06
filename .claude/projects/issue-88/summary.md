# Project Summary: Clean Up Remaining Yield References

**Issue:** ISSUE-88
**Date:** 2026-02-06

## Executive Overview

ISSUE-88 removed all references to the deprecated yield infrastructure from documentation and code. The yield system (TOML-based skill communication via file I/O) was replaced by direct teammate messaging (`SendMessage` tool) during the Phase 1/2 migration (ISSUE-69 through ISSUE-83), but documentation still contained ~40+ files referencing the old patterns.

**Outcome:** All active documentation, skill files, config files, and code comments updated to reflect the current messaging-based communication model. 7/7 verification checks pass. 4 follow-up issues created from retrospective.

## Key Decisions

### Multi-pattern grep discovery (ARCH-001)
**Traces to:** DES-001, REQ-001

Used ripgrep with multiple case-insensitive patterns (`yield`, `yield_path`, `producer_yield_path`, `yield.type`, `[yield]`) to identify all files. This caught references across different usage contexts that a single pattern would miss.

### In-place edit with read-before-edit (ARCH-002)
**Traces to:** DES-002, DES-003, REQ-002

Used Edit tool for precise replacements with Read-before-Edit pattern. When yield content was >50% of a section, rewrote the entire section rather than piecemeal edits. This produced cleaner results than incremental changes.

### Historical preservation strategy (DES-004, DES-005)
**Traces to:** REQ-003

Preserved historical yield directories in closed projects (`.claude/projects/ISSUE-*/yields/`) and retrospective documents. Only active workflow documentation was updated. This maintains project archaeology while eliminating confusion in current docs.

### Verification via TDD (ARCH-003, ARCH-005)
**Traces to:** DES-006, REQ-004

Created both a shell verification script (7 checks) and Go tests before making any changes. The tests caught the worker-docs annotation problem and provided confidence that cleanup was complete.

### Live yield code exclusion
**Traces to:** REQ-001

Correctly excluded `internal/memory/` files containing active `projctl memory extract --yield` runtime code. This is live functionality, not dead documentation — filed ISSUE-116 to decide its future.

## Outcomes

### Deliverables
- ~40 files modified: SKILL.md files, architecture docs, design docs, project docs, config files, Go comments
- 6 files deleted: 4 obsolete docs (`docs/orchestration-system.md`, `docs/unified-system-design.md`, `docs/simplification-analysis.md`, `docs/current-ideal-flows.md`) + 2 yield.toml files
- 7 context config files cleaned of `yield_path`/`producer_yield_path` fields
- Verification script at `scripts/test-yield-cleanup.sh` (7 checks, all passing)
- Go test at `internal/memory/yield_cleanup_test.go`

**Traces to:** TASK-2 through TASK-12

### Quality
- 7/7 verification checks passing
- All replacement mappings followed DES-007 consistently
- Traceability chain validated: TASK → ARCH → DES → REQ → ISSUE-88

**Traces to:** TASK-13, TASK-14, TASK-15

### Follow-up Issues Created
- ISSUE-113: QA hallucination guards (High)
- ISSUE-114: Task completion signal for state machine (Medium)
- ISSUE-115: Worker test enforcement (Medium)
- ISSUE-116: Live yield code decision (Medium)

## Lessons Learned

**Process:** The `/project` orchestration process works well for systematic cleanup tasks. The TDD discipline caught a worker producing incorrect output (annotations instead of removals).

**Parallelization:** Splitting work by file type (skills, docs, configs, tests) enabled effective parallel execution with minimal conflicts.

**QA model choice:** Haiku QA agents hallucinate findings too frequently for reliable automated QA. Either use a more capable model or add structural validation.

**Skill invocation in teammates:** Skills with `context: fork` don't work in teammate mode — this is a known limitation tracked as ISSUE-105.

## Known Limitations

- Live yield code in `internal/memory/` remains untouched (ISSUE-116)
- Historical yield references preserved in retrospective and archived project files (by design)
- State machine required manual force-transition to exit TDD loop (ISSUE-114)
