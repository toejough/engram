# Rebuild State

## Current Phase
VERTICAL IMPLEMENTATION — Process redesigned. Moving from horizontal (layer-by-layer) to depth-first vertical (group-and-descend at every layer). REQ-1 through REQ-21 exist covering UC-1 through UC-4, pending user inline review. UC-5 through UC-14 requirements not yet extracted.

## Process (Depth-First Tree Traversal)

The rebuild now follows a depth-first tree traversal with grouping and prioritization at every layer:

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
1. Discuss clearing out current memory implementation (user requested)
2. Then: present 3-5 UC grouping options with rationale for user selection
3. Then: present 3-5 priority options for which group goes first
4. Then: begin depth-first vertical work on the chosen group

## Context Files
- `docs/rebuild/prompt.md` — Full rebuild process instructions
- `docs/rebuild/lessons.md` — Validated Phase 1 output + 5 process lessons from Phase 3
- `docs/rebuild/use-cases.md` — Validated Phase 2 output (14 use cases + 7 cross-cutting design decisions)
- `docs/rebuild/requirements.md` — REQ-1 through REQ-21 (UC-1 through UC-4), pending inline review

## Completed Phases
- **LESSONS (Phase 1)** — Validated. 16 successes, 13 failures, 9 design constraints.
- **USE CASES (Phase 2)** — Validated. 14 use cases (UC-1 through UC-14) + 7 cross-cutting design decisions.
- **REQUIREMENTS (Phase 3, partial)** — REQ-1 through REQ-21 written covering UC-1 through UC-4. Not yet reviewed inline. UC-5-14 not yet extracted. Process changed to vertical before completing this phase horizontally.

## Last Session Summary
Redesigned the rebuild process from horizontal (complete each layer across all UCs before moving to the next layer) to depth-first vertical (group UCs, pick a group, work top-to-bottom through all layers for that group, backtrack to next group). Key decisions:
- Group and prioritize at EVERY layer, not just the UC layer
- Refactor the ENTIRE current layer at each step (not just the current group) to catch cross-group inconsistencies early
- Progressive refactoring at every layer rather than deferring to green-then-refactor
- Dirty-marking for invalidated descendants: propagate down, resolve on visit, don't chase cascades
- Final depth-first sweep as safety net for orphaned dirty flags

## Open Questions
- How to clear out current memory implementation to start fresh (user wants to discuss)

## Artifacts Produced
- `docs/rebuild/lessons.md` — Validated
- `docs/rebuild/use-cases.md` — Validated
- `docs/rebuild/requirements.md` — Partial (REQ-1 through REQ-21, UC-1 through UC-4 only)
