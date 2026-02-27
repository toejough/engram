# Rebuild State

## Current Phase
VERTICAL IMPLEMENTATION — Group A (Core Pipeline: UC-1, UC-2, UC-3). UC grouping validated. Next step: requirements layer refactor before descending.

## UC Groupings (Validated)

- **Group A: Core Pipeline** (UC-1, UC-2, UC-3) — Write → Read → Correct. Foundation for everything else. **ACTIVE — going first.**
- **Group B: Evaluation & Quadrant Actions** (UC-7, UC-8, UC-9, UC-10) — Importance × impact matrix. Depends on Group A signals.
- **Group C: Promotion & Lifecycle** (UC-4, UC-5, UC-6, UC-12) — Tier movement (memory → skill, memory → CLAUDE.md, skill maintenance, guidance → hook). Depends on Group B evaluation data.
- **Group D: Infrastructure** (UC-11, UC-13, UC-14) — Plugin installation, sharing/portability, session continuity. Mostly independent.

## Process (Depth-First Tree Traversal)

The rebuild follows a depth-first tree traversal with grouping and prioritization at every layer:

1. **At each layer:** Group items, prioritize, pick a group to go deep on.
2. **Before descending:** Refactor the ENTIRE current layer (not just the current group):
   - Validate dirty nodes against their refactored parents (clear flag if still valid, re-derive if not)
   - Whole-layer consistency check across all nodes at this layer
   - Bidirectional satisfaction: does this layer satisfy exactly the layer above?
   - Semantic review: simple, deduplicated, progressive disclosure, standardized?
   - Surface lessons for recording/adjusting/dismissal
   - Dirty-mark descendants of anything that changed
3. **Descend** to next layer on the chosen group.
4. **Backtrack** to next unfinished sibling group, repeat.
5. **Final sweep:** After entire tree is complete, depth-first walk to resolve any orphaned dirty flags (expected: all clean, safety net only).

Layer progression per group: UC → REQ → DES/ARCH → Tests → Implementation

Dirty-marking rules:
- When a layer-N refactor changes a node, all its descendants at layer N+1 and below are marked dirty
- Dirty flags propagate downward but are only resolved when naturally visited during traversal
- Dirty resolution: re-validate against refactored parent → still valid? clear flag. Invalid? re-derive, mark own descendants dirty.

## Next Action
1. Requirements layer refactor for Group A (UC-1, UC-2, UC-3): REQ-1 through REQ-18 exist. Apply the per-requirement checklist from prompt.md, do whole-layer consistency check, bidirectional satisfaction check against use-cases.md, and semantic review.
2. After requirements are clean: descend to design layer for Group A.

## Context Files
- `docs/prompt.md` — Full rebuild process instructions
- `docs/lessons.md` — Validated Phase 1 output + 5 process lessons from Phase 3
- `docs/use-cases.md` — Validated Phase 2 output (14 use cases + 7 cross-cutting design decisions)
- `docs/requirements.md` — REQ-1 through REQ-21 (UC-1 through UC-4), pending inline review

## Completed Phases
- **LESSONS (Phase 1)** — Validated. 16 successes, 13 failures, 9 design constraints.
- **USE CASES (Phase 2)** — Validated. 14 use cases (UC-1 through UC-14) + 7 cross-cutting design decisions.
- **REQUIREMENTS (Phase 3, partial)** — REQ-1 through REQ-21 written covering UC-1 through UC-4. Not yet reviewed inline. UC-5-14 not yet extracted. Process changed to vertical before completing this phase horizontally.

## Last Session Summary
Validated UC groupings: Group A (Core Pipeline: UC-1/2/3), Group B (Evaluation: UC-7/8/9/10), Group C (Promotion: UC-4/5/6/12), Group D (Infrastructure: UC-11/13/14). Group A goes first as the dependency root and fastest path to a working plugin. Fixed SessionEnd hook error (prompt hooks not supported on SessionEnd — removed it, Stop hook provides same coverage). Next: requirements layer refactor for Group A before descending.

## Open Questions
None.

## Artifacts Produced
- `docs/lessons.md` — Validated
- `docs/use-cases.md` — Validated
- `docs/requirements.md` — Partial (REQ-1 through REQ-21, UC-1 through UC-4 only)
