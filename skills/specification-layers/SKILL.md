---
name: specification-layers
description: |
  Core: Diamond-topology specification model for taking a project from use cases to working implementation with bidirectional signal propagation and alignment guarantees.
  Triggers: plan specification layers, set up project layers, define spec structure, organize requirements and design, UC to implementation, specification process, layer model.
  Domains: specification, requirements, design, architecture, project-structure, traceability.
  Anti-patterns: NOT for single-file changes, NOT for quick fixes with known files/lines, NOT for research-only tasks.
---

# Specification Layers

A diamond-topology model for organizing work from use cases to implementation. Designed for projects complex enough to need multiple specification layers, where coherence across layers prevents costly rework.

## The Diamond Model

```
        UC                   Problem space: user goals
       / \
    REQ   DES              Peers: invariants / interaction model
       \ /
       ARCH                  Solution space: system structure
        |
       TEST                  Implementation: verification (TDD red)
        |
       IMPL                  Implementation: code (TDD green + refactor)
```

Six layers. The top four form a diamond; the bottom two are linear.

### Layer Descriptions

**UC (Use Cases)** — User goals and interaction flows. The common ancestor from which REQ and DES independently derive. Discovers scope. Format: numbered UC-N entries with actor, starting state, end state, key interactions.

**REQ (Requirements)** — Atomic, testable invariants derived from UCs. Problem space: what must be true regardless of implementation. Traces to UC. Format: numbered REQ-N entries with acceptance criteria and verification tier.

**DES (Design)** — Interaction model and behavioral specification. Traces to UC (not to REQ — they are peers). Two passes:
1. *Horizontal (UX coherence):* Study all UCs as a whole. Design interaction primitives that satisfy all of them coherently — unified feedback formats, proposal patterns, communication channels. The goal: one product, not N independent features.
2. *Vertical (behavioral specification):* Walk each UC through those primitives as concrete scenarios — case studies, mock output, edge cases. Verify the primitives satisfy the UCs. If a primitive can't satisfy a UC, fix the primitive.

**ARCH (Architecture)** — System structure. The convergence point: traces to REQ (primary derivation) with correspondence to DES (must support designed interactions). Component boundaries, data model, tech decisions, behavioral contracts, interaction protocols. Must be comprehensive enough to be the sole source for tests.

**TEST (Tests)** — TDD red phase. Three test types, all derived from ARCH (which reflects REQ + DES):
- *Property-based tests* verify invariants (trace through ARCH back to REQ)
- *Example-based tests* verify scenarios (trace through ARCH back to DES)
- *Integration tests* verify component boundaries (directly from ARCH)

**IMPL (Implementation)** — TDD green + refactor. Code that makes tests pass.

### Why a Diamond, Not a Chain

REQ and DES are parallel derivations from UC — neither derives from the other.

- REQ decomposes user goals into testable invariants (system-facing)
- DES designs a coherent interaction model satisfying user goals (user-facing)
- Both trace to UC independently; both are needed before ARCH

A linear chain (UC → REQ → DES → ARCH) would imply DES derives from REQ, creating a false dependency and wrong directionality (invariants don't dictate the interaction model). The diamond captures the actual relationships.

### Derivation and Consistency

| Layer | Derives from | Consistency check against |
|-------|-------------|--------------------------|
| REQ | UC | DES (can't contradict interaction commitments) |
| DES | UC | REQ (can't contradict invariants) |
| ARCH | REQ (primary) + DES (correspondence) | — |
| TEST | ARCH (which reflects REQ + DES) | — |
| IMPL | TEST | — |

REQ and DES consistency is checked after both are complete, before descending to ARCH. This is the standard layer consistency check applied between diamond peers — not a separate mechanism.

## Bidirectional Signal Propagation

Two signal types propagate through the diamond:

| Signal | Direction | Trigger | Action |
|--------|-----------|---------|--------|
| **Derivation staleness** | Downward | Parent artifact revised | Mark children dirty; resolve when visited |
| **Constraint discovery** | Upward | Child can't satisfy parent | Annotate parent; backtrack immediately; parent revises (triggers downward) |

### Downward Propagation

When a layer revises an artifact, mark descendant layers dirty. UC fans out: a UC change dirties both REQ and DES. REQ or DES changes dirty ARCH. ARCH dirties TEST. TEST dirties IMPL.

### Upward Propagation

When a layer discovers its parent is unsatisfiable (impossible, over-constrained, under-specified):

1. Annotate the parent with the discovered constraint and why
2. Immediately backtrack to the parent
3. Parent absorbs the constraint by revising — triggering normal downward propagation
4. Resume descent

ARCH has two parents. If ARCH can't satisfy a REQ, propagate to REQ. If ARCH can't support a DES decision, propagate to DES. Independent upward paths.

### Absorption-First Rule

Each layer absorbs constraints locally before escalating. Most constraints resolve one layer up. Only fundamental impossibilities cascade. If propagation reaches UC and the use case is unsatisfiable, that's a scope cut.

**Examples:**
- IMPL: "Hook model is synchronous" → ARCH absorbs (revises to sync pipeline). Done.
- IMPL: "Can't inject multiple reminders per hook" → ARCH can't absorb → DES absorbs (feedback at next hook point). Done.
- TEST: "Property X is computationally infeasible" → ARCH can't absorb → REQ absorbs (weakens invariant). Downward dirty-flags ARCH.

## Depth-First Traversal

The traversal is a cycle of group → refactor → descend → backtrack applied recursively at every layer.

### Algorithm

1. **Group and prioritize** at the current layer. Present to the user:
   - **Grouping options:** 2-3 ways to cluster items at this layer (by dependency, domain, complexity, risk). Explain the tradeoff of each grouping.
   - **Recommended grouping** with rationale (e.g., "Group by dependency — Group A is foundational for B and C").
   - **Priority ordering** within the chosen grouping. Recommend which group to take deep first and why. The user chooses.

   This decision point recurs at every layer for every set of items. Never silently pick a grouping or priority — always present options.

2. **Refactor the ENTIRE current layer** (not just the active group) before descending:
   - Validate dirty nodes against refactored parents (clear if still valid, re-derive if not)
   - Resolve any upward constraints annotated from prior descents
   - Whole-layer consistency check across all nodes
   - Bidirectional satisfaction: does this layer satisfy exactly the layer above?
   - For diamond peers (REQ ⟷ DES): cross-check consistency
   - Ubiquitous language: same terms for same concepts across all layers
   - Surface lessons learned
   - Dirty-mark descendants of anything changed

3. **Descend** to the next layer on the chosen group. Repeat from step 1 at the new layer.

4. **Backtrack** when a group reaches IMPL. Return to the parent layer and pick the next unfinished sibling group. Repeat from step 3.

5. **Final sweep** after the entire tree is complete. Walk depth-first to resolve any orphaned dirty flags. Expected: all clean (safety net only).

### Process Ordering Through the Diamond

REQ and DES can be done in either order after UC (both derive from UC). ARCH requires both. Typical flow per group:

```
UC (discover) → REQ (decompose) → DES (design) → REQ⟷DES check →
ARCH (converge) → TEST (TDD red) → IMPL (TDD green + refactor)
```

## Alignment Mechanisms

1. **Ubiquitous language.** Same terms across all layers. If UC says "reconciliation," code has `reconcile()`. Grep-able when terminology changes at any layer.

2. **Bidirectional traceability.** Each layer traces to its immediate parent(s). Parent covered by at least one child. In the diamond: REQ and DES trace to UC; ARCH traces to REQ and DES. If ARCH only makes sense by referencing UC directly, there's a missing REQ or DES entry.

3. **Peer consistency.** REQ and DES checked against each other after both complete (standard consistency check applied between diamond peers).

4. **Dirty flag propagation.** Downward staleness and upward constraints both follow the diamond structure. UC fans out; ARCH has two independent upward paths.

## Applying to a Project

### When to Use

Use this model when a project has enough complexity that:
- Multiple use cases need coherent implementation
- Interaction design and system architecture are distinct concerns
- Traceability from goals to code prevents costly rework
- The project spans multiple sessions (state must persist)

Skip for: single-file fixes, clear requirements with known implementation, prototypes.

### Getting Started

1. **Write UCs.** Interview or discover use cases.
2. **Group and prioritize.** Present 2-3 grouping options with tradeoffs. Recommend one. User chooses grouping and priority order.
3. **Pick a group.** Take it through all layers before backtracking for the next.
3. **Derive REQ and DES from UC** (either order). Check peer consistency.
4. **Converge at ARCH.** Must satisfy REQ + support DES.
5. **TDD.** Tests from ARCH, code to pass them.
6. **Backtrack** for the next group.

### State Persistence

After every substantive interaction, update a state file with:
- Current layer and group
- Specific next action (concrete enough for a fresh session)
- Context files to read
- Completed layers with summary
- Open questions

### Reference

For the research behind this model (8 framework analyses, alternative topologies evaluated, key design decisions with rationale), see `references/research-and-tradeoffs.md`.
