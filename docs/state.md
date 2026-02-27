# Rebuild State

## Current Phase
VERTICAL IMPLEMENTATION — Group A (Core Pipeline: UC-1, UC-2, UC-3). UC-level refactor and REQ dirty-flag resolution COMPLETE. Ready to descend to design layer.

## UC Groupings (Validated)

- **Group A: Core Pipeline** (UC-1, UC-2, UC-3) — Write → Read → Correct. Foundation for everything else. **ACTIVE — going first.**
- **Group B: Evaluation & Quadrant Actions** (UC-7, UC-8, UC-9, UC-10) — Importance × impact matrix. Depends on Group A signals.
- **Group C: Promotion & Lifecycle** (UC-4, UC-5, UC-6, UC-12) — Tier movement (memory → skill, memory → CLAUDE.md, skill maintenance, guidance → hook). Depends on Group B evaluation data.
- **Group D: Infrastructure** (UC-11, UC-13, UC-14) — Plugin installation, sharing/portability, session continuity. Mostly independent.

## Process (Depth-First Tree Traversal with Bidirectional Propagation)

The rebuild follows a depth-first tree traversal with grouping and prioritization at every layer.

### Layer Model

Three spaces, six layers. The top four form a diamond (not a linear chain); the bottom two are linear.

```
        UC                   Problem space: user goals
       ↙  ↘
    REQ ⟷ DES              REQ: invariants / DES: interaction model
       ↘  ↙                 (peers derived from UC, consistency-checked)
       ARCH                  Solution space: system structure
        ↓
       TEST                  Implementation space: verification
        ↓
       IMPL                  Implementation space: code
```

**Diamond structure:** UC fans out to REQ and DES — both derive from UC, neither derives from the other. REQ and DES converge at ARCH, which must satisfy requirements AND support designed interactions. Below ARCH is a linear chain.

**Layer descriptions:**

- **UC** (Use Cases): User goals and interaction flows. Discovers scope. The common ancestor.
- **REQ** (Requirements): Atomic, testable invariants derived from UCs. Traces to UC.
- **DES** (Design — UX + Behavioral Specification): Traces to UC. Two passes. First, study the UCs as a whole and design interaction primitives that satisfy all of them coherently — unified feedback formats, proposal patterns, communication channels. This is genuine UX work: ensuring the plugin feels like one product, not N independent features. Second, walk each UC through those primitives as concrete scenarios (case studies, hook flows, mock output, edge cases), verifying the primitives satisfy the UCs. Directionality matters: if a primitive can't satisfy a UC, fix the primitive.
- **ARCH** (Architecture): Component boundaries, data model, tech decisions. Traces to REQ (primary derivation) with correspondence to DES (must support designed interactions).
- **TEST** (Tests): Verification spanning multiple layers. Property-based tests verify REQ invariants. Example-based tests verify DES scenarios. Integration tests verify ARCH boundaries.
- **IMPL** (Implementation): Code that satisfies tests.

**Derivation and consistency:**

| Layer | Derives from | Consistency check against |
|-------|-------------|--------------------------|
| REQ | UC | DES (can't contradict interaction commitments) |
| DES | UC | REQ (can't contradict invariants) |
| ARCH | REQ (primary) + DES (correspondence) | — |
| TEST | REQ + DES + ARCH | — |
| IMPL | TEST | — |

REQ ⟷ DES consistency is the existing layer consistency check applied between peers in the diamond — not a new concept. After completing both, verify they don't contradict each other before descending to ARCH.

**Process ordering:** REQ and DES can be done in either order after UC (both derive from UC independently). We do REQ first, then DES — but neither depends on the other. ARCH requires both to be complete.

### Traversal

1. **At each layer:** Group items, prioritize, pick a group to go deep on.
2. **Before descending:** Refactor the ENTIRE current layer (not just the current group):
   - Validate dirty nodes against their refactored parents (clear flag if still valid, re-derive if not)
   - Resolve any upward constraints annotated on this layer from prior descents
   - Whole-layer consistency check across all nodes at this layer
   - Bidirectional satisfaction: does this layer satisfy exactly the layer above?
   - Semantic review: simple, deduplicated, progressive disclosure, standardized?
   - Ubiquitous language check: same terms across all layers for same concepts
   - Surface lessons for recording/adjusting/dismissal
   - Dirty-mark descendants of anything that changed
3. **Descend** to next layer on the chosen group.
4. **Backtrack** to next unfinished sibling group, repeat.
5. **Final sweep:** After entire tree is complete, depth-first walk to resolve any orphaned dirty flags (expected: all clean, safety net only).

### Bidirectional Dirty Flags

Two signal types propagate through the tree:

| Signal | Direction | Trigger | Action |
|--------|-----------|---------|--------|
| **Derivation staleness** | Downward | Parent artifact revised | Mark children dirty; resolve when visited |
| **Constraint discovery** | Upward | Child can't satisfy parent spec | Annotate parent with constraint; backtrack immediately; parent revises (triggers downward propagation) |

**Downward propagation:** When a layer revises an artifact, mark all descendant layers dirty. In the diamond, UC fans out: UC changes dirty both REQ and DES. REQ or DES changes dirty ARCH. ARCH changes dirty TEST. TEST changes dirty IMPL.

**Upward propagation:** When a layer discovers its parent is unsatisfiable (impossible, over-constrained, or under-specified):

1. Annotate the parent artifact with the discovered constraint and why it can't be satisfied
2. Immediately backtrack to the parent to resolve
3. Parent absorbs the constraint by revising itself — which triggers normal downward dirty-flagging
4. Resume descent

In the diamond, ARCH has two parents (REQ and DES). If ARCH can't satisfy a REQ, propagate to REQ. If ARCH can't support a DES decision, propagate to DES. These are independent upward paths.

**Absorption-first rule:** Each layer tries to absorb the constraint locally before escalating. If the parent can resolve within its own scope, propagation stops. If it can't, it escalates to its own parent. Most constraints are absorbed one layer up. Only fundamental impossibilities cascade multiple layers.

**Termination:** If upward propagation reaches UC and the use case is unsatisfiable, that's a scope cut — remove or revise the UC.

**Examples:**
- IMPL: "Hook model is synchronous" → ARCH absorbs: revises to sync pipeline. Done.
- IMPL: "Can't inject multiple reminders per hook" → ARCH can't absorb (platform constraint) → DES absorbs: feedback at next hook point. Done.
- ARCH: "Can't unify storage for REQ-5 and DES interaction model" → REQ absorbs: relax storage constraint. DES unaffected.
- TEST: "Reconciliation isn't idempotent" → ARCH can't absorb → REQ absorbs: weaken to "convergent within K ops" → downward dirty-flags ARCH (and DES if affected).

### Alignment Mechanisms

1. **Ubiquitous language:** Same terms across all layers for same concepts. If UC says "reconciliation," code has `reconcile()`. Enforced at every layer transition.
2. **Bidirectional traceability:** Each layer traces to its immediate parent(s). Parent is covered by at least one child artifact. In the diamond: REQ and DES both trace to UC. ARCH traces to REQ (derivation) and DES (correspondence). If ARCH only makes sense by referencing UC directly, there's a missing REQ or DES entry.
3. **Peer consistency:** REQ ⟷ DES are checked against each other after both are complete (before descending to ARCH). This is the existing layer consistency check applied between diamond peers — not a new mechanism.
4. **Change propagation via dirty flags:** Downward (derivation staleness) and upward (constraint discovery) signals propagate through the diamond structure. UC fans out to both REQ and DES; both converge at ARCH.

## Next Action
1. Descend to DES layer for Group A (UC-1, UC-2, UC-3). First: horizontal pass — design interaction primitives across all Group A UCs (unified feedback formats, proposal patterns, communication channels). Second: vertical pass — walk each UC through those primitives as concrete scenarios (case studies, hook flows, mock output, edge cases).
2. Present each case study to user for validation before moving to the next.
3. After DES complete: REQ ⟷ DES peer consistency check before descending to ARCH.

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
Refined the layer model and process before descending to DES. Key decisions:

**Layer model — diamond topology:** UC fans out to REQ and DES (both derive from UC, neither from each other). REQ and DES converge at ARCH (must satisfy REQ invariants AND support DES interactions). ARCH → TEST → IMPL is linear below. Research across 8 frameworks (4+1, IEEE 42010, Zachman, OOUX, Problem Frames, Twin Peaks, V-Model, RUP) confirmed: parallel derivations from a common ancestor, not a linear chain.

**DES framing:** UX + behavioral specification. Horizontal first (interaction primitives satisfying all UCs coherently), vertical second (each UC walked through primitives). Primitives serve UCs, not the reverse.

**Bidirectional dirty flags:** Upward constraint propagation added. Absorption-first rule. In the diamond: UC fans out downward to both REQ and DES; ARCH propagates upward to REQ or DES independently.

**Peer consistency:** REQ ⟷ DES checked against each other after both complete — existing consistency mechanism applied between diamond peers, not a new concept.

**Tests trace to multiple layers:** Property-based → REQ invariants. Example-based → DES scenarios. Integration → ARCH boundaries.

### Previous Session Summary
Retroactive UC-level refactoring + full REQ dirty-flag resolution + verified refactoring criteria at both layers. Key changes:

**UC layer (8 changes):** Removed premature TF-IDF constraint → "local retrieval" with algorithm deferred to architecture. Added quality gate to UC-1. Unified "reconciliation" terminology across UC-1/UC-3. Removed deferred backlog note from UC-2. Added ranking bullet to UC-2 (frecency primary, confidence tiebreaker, cold start behavior) — this closed a gap where REQ-4 traced to UC-2 but UC-2 never mentioned confidence or ranking.

**REQ layer (10 REQs touched):** All TF-IDF → local similarity. REQ-5 renamed Deduplication → Reconciliation. REQ-2 got quality gate AC(4). REQ-4 expanded from "confidence tiebreaker" to full ranking behavior REQ (absorbed REQ-7's misplaced ACs 4-5 about cold start and impact-over-frequency). REQ-7/8/9 all got consistent (REQ-4) cross-references. Budget table unified to single `reconciliation.candidate_count` for REQ-5 and REQ-14.

**Verification:** Both UC and REQ layers passed all 5 refactoring criteria (parent validation, consistency, bidirectional satisfaction, semantic review, lessons). Each pass done against actual file content, not assumptions. UC re-verified after adding ranking bullet. REQ re-verified after adding cross-references.

## Open Questions
None.

## Artifacts Produced
- `docs/lessons.md` — Validated (24 process lessons, 16 successes, 14 failures, 9 constraints)
- `docs/use-cases.md` — Validated (14 UCs + 7 cross-cutting decisions)
- `docs/requirements.md` — Group A validated, Group C (UC-4) unchanged, UC-5-14 not yet extracted
- `skills/specification-layers/` — Reusable skill: diamond model, bidirectional propagation, traversal process, alignment mechanisms. Research notes in `references/research-and-tradeoffs.md`.
