# Rebuild State

## Current Phase
VERTICAL IMPLEMENTATION — Group A (Core Pipeline: UC-1, UC-2, UC-3). UC-level refactor and REQ dirty-flag resolution COMPLETE. Ready to descend to design layer.

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
1. Descend to design layer for Group A (UC-1, UC-2, UC-3). Per prompt.md Phase 4: produce case study walkthroughs, hook interaction flows, system reminder mock output, and error/edge case interactions for each UC.
2. Present each case study to user for validation before moving to the next.

## Context Files
- `docs/prompt.md` — Full rebuild process instructions
- `docs/lessons.md` — Validated Phase 1 output + 5 process lessons from Phase 3
- `docs/use-cases.md` — Validated Phase 2 output (14 use cases + 7 cross-cutting design decisions)
- `docs/requirements.md` — REQ-1 through REQ-21 (UC-1 through UC-4). Group A (REQ-1-18) validated and refactored.

## Completed Phases
- **LESSONS (Phase 1)** — Validated. 16 successes, 13 failures, 9 design constraints.
- **USE CASES (Phase 2)** — Validated. 14 use cases (UC-1 through UC-14) + 7 cross-cutting design decisions.
- **REQUIREMENTS (Phase 3, partial)** — REQ-1 through REQ-21 written covering UC-1 through UC-4. Not yet reviewed inline. UC-5-14 not yet extracted. Process changed to vertical before completing this phase horizontally.

## Last Session Summary
Retroactive UC-level refactoring + full REQ dirty-flag resolution + verified refactoring criteria at both layers. Key changes:

**UC layer (8 changes):** Removed premature TF-IDF constraint → "local retrieval" with algorithm deferred to architecture. Added quality gate to UC-1. Unified "reconciliation" terminology across UC-1/UC-3. Removed deferred backlog note from UC-2. Added ranking bullet to UC-2 (frecency primary, confidence tiebreaker, cold start behavior) — this closed a gap where REQ-4 traced to UC-2 but UC-2 never mentioned confidence or ranking.

**REQ layer (10 REQs touched):** All TF-IDF → local similarity. REQ-5 renamed Deduplication → Reconciliation. REQ-2 got quality gate AC(4). REQ-4 expanded from "confidence tiebreaker" to full ranking behavior REQ (absorbed REQ-7's misplaced ACs 4-5 about cold start and impact-over-frequency). REQ-7/8/9 all got consistent (REQ-4) cross-references. Budget table unified to single `reconciliation.candidate_count` for REQ-5 and REQ-14.

**Verification:** Both UC and REQ layers passed all 5 refactoring criteria (parent validation, consistency, bidirectional satisfaction, semantic review, lessons). Each pass done against actual file content, not assumptions. UC re-verified after adding ranking bullet. REQ re-verified after adding cross-references.

## Open Questions
None.

## Artifacts Produced
- `docs/lessons.md` — Validated
- `docs/use-cases.md` — Validated
- `docs/requirements.md` — Group A validated, Group C (UC-4) unchanged, UC-5-14 not yet extracted
