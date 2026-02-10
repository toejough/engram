# Evaluation: ISSUE-202

**Date:** 2026-02-10
**Issue:** ISSUE-202: Memory: Make generated skills discoverable by Claude Code natively
**Team:** issue-202
**Duration:** Single session

---

## Project Summary

Restructured generated memory skills from nested `memory-gen/{slug}/` to flat `mem-{slug}/` layout under `~/.claude/skills/` for native Claude Code discovery. Removed the hook-based skill injection path (`searchSkills()`) entirely, replacing eager context injection with Claude Code's lazy-loaded native skill discovery.

**Scope:** 19 files changed (237 additions, 867 deletions — net reduction of 630 lines)

**Key Deliverables:**
1. Flat `mem-{slug}/SKILL.md` skill structure with `mem:` prefix naming
2. Migration function for legacy `memory-gen/` skills
3. Removal of hook-based skill injection (skill_retrieve.go deleted)
4. Updated prune/merge/split operations for new path convention
5. Updated CLI `skillsDir` to point to `~/.claude/skills/` root

**Team Roles:** 9 agents across 2 parallel streams (red, red-b, green, green-a2, green-b, doc, traceability, watchdog, evaluation)

---

## Key Decisions

### D1: Replace hook injection with native discovery
Chose to remove `searchSkills()` entirely rather than maintaining both paths. Native Claude Code discovery provides lazy-loaded context (~100 tokens description always, full content on-demand) versus hook injection's eager loading on every message.

**Traces to:** R6, TASK-6, TASK-8, TASK-9

### D2: Flat `mem-{slug}` directory naming
Used `mem-` prefix for directories (colons invalid in directory names) with `mem:` prefix in SKILL.md `name` field. This provides visual distinction from hand-crafted skills while maintaining Claude Code's one-level-deep discovery pattern.

**Traces to:** R1, R2, TASK-1, TASK-2

### D3: User-invocable by default
Set `user-invocable: true` on generated skills, enabling explicit `/mem-{slug}` invocation. This is a shift from the previous `user-invocable: false` which only allowed implicit auto-invocation.

**Traces to:** R3, TASK-3

### D4: Remove `context: inherit` from frontmatter
Dropped `context: inherit` from generated skill frontmatter, reducing context overhead and aligning with native skill conventions.

**Traces to:** R2, TASK-2

### D5: In-place migration with idempotency
Migration function (`migrateMemoryGenSkills`) handles: no-op when no legacy directory exists, skip when destination already exists, cleanup of empty `memory-gen/` directory. Idempotent by design.

**Traces to:** R4, TASK-4

---

## Outcomes vs Goals

| Requirement | Status | Evidence |
|---|---|---|
| R1: Flat skill directory structure | Met | `writeSkillFile()` generates `mem-{slug}/SKILL.md`; prune/merge/split use `mem-` prefix |
| R2: Prefix naming convention | Met | Frontmatter: `name: mem:{slug}`, directory: `mem-{slug}`, no `context: inherit` |
| R3: User-invocable by default | Met | `user-invocable: true` in frontmatter; safety: prune only touches `mem-*` dirs |
| R4: Migration of existing skills | Met | `migrateMemoryGenSkills()` with 3 test cases (basic, idempotent, no-op) |
| R5: Update optimize pipeline | Met | CLI `skillsDir` points to `~/.claude/skills/`; all prune/merge/split paths updated |
| R6: Remove hook-based injection | Met | `skill_retrieve.go` deleted; `Skills` field removed from `QueryResults` and `FormatMarkdownOpts`; `FormatSkillContext` removed |

**Requirements coverage:** 6/6 (100%)

**Quality Metrics:**
- Net code reduction: 630 lines removed (cleaner architecture)
- Deleted files: 3 (`skill_retrieve.go`, `skill_retrieve_test.go`, `skill_retrieve_testhelpers.go`)
- New test files: 2 (`hook_removal_test.go`, `migrate_test.go`)
- Also removed dead code in `consolidate.go` (~93 lines of `mergeDuplicates` and `truncateLine`)

---

## Process Findings

### High Priority

**H1: Parallel stream execution effective for independent requirements**
Splitting R1-R5 (restructuring) and R6 (hook removal) into parallel streams with separate red/green agent pairs eliminated inter-stream blocking. Both streams completed without merge conflicts.

**Traces to:** R1, R2, R3, R4, R5, R6

### Medium Priority

**M1: Agent handoff latency from hub-and-spoke routing**
All agent communication routed through team lead, creating bottlenecks. Agents waited idle during handoffs when peer-to-peer messaging would have been more efficient (red → green directly).

**M2: Agent nudging overhead**
Some agents required explicit prompts to continue after receiving task assignments. Pre-briefing agents with context about upcoming work before formal handoff could reduce startup latency.

### Low Priority

**L1: Task complexity estimation**
TASK-6 (single-line `skillsDir` change) took longer than expected due to full TDD cycle overhead. One-line changes still require test → implement → verify, which shouldn't be underestimated in planning.

---

## Recommendations

### R-HIGH-1: Enable peer-to-peer agent messaging in TDD chains
Agents in sequential chains (red → green → refactor) should message the next agent directly rather than routing through team lead. This reduces handoff latency and improves watchdog visibility.

### R-MED-1: Pre-brief agents on upcoming tasks
Send context about the next task to agents before formal handoff to reduce startup discovery time. Include: task description, relevant files, and expected approach.

### R-MED-2: Batch minimal changes into single TDD cycle
When multiple one-line changes share the same acceptance criteria pattern, consider batching them into a single TDD red/green cycle rather than individual agent spawns.

---

## Open Questions

**Q1:** Should generated skills include a `---` separator between frontmatter metadata fields and the compiled skill content to improve Claude Code's parsing?

**Q2:** As the number of `mem-*` skills grows, should there be a cap on total generated skills to prevent `~/.claude/skills/` directory from becoming cluttered?

---

**Traceability:** R1-R6 from ISSUE-202 requirements; TASK-1 through TASK-13 from project breakdown; D1-D5 decisions documented above.
