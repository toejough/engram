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

The traversal algorithm uses two flags (`dirty` and `unsatisfiable`) to propagate signals through the diamond. This section explains the rationale and diamond-specific behavior.

### Why Two Signals

| Signal | Direction | Flag | Purpose |
|--------|-----------|------|---------|
| **Derivation staleness** | Downward | `dirty` on children | Parent revised → children may be stale |
| **Constraint discovery** | Upward | `unsatisfiable` on parent | Child can't satisfy parent → parent must adapt |

Both flags are written on the impacted node by the discoverer. The traversal algorithm (see "Depth-First Traversal") defines what happens when the cursor arrives at a flagged node.

### Diamond-Specific Propagation

UC fans out: a UC change dirties both REQ and DES nodes. REQ or DES changes dirty ARCH nodes. ARCH dirties TEST. TEST dirties IMPL.

ARCH has two parents (REQ and DES groups). If ARCH can't satisfy a REQ, it marks the REQ group node `unsatisfiable`. If ARCH can't support a DES decision, it marks the DES group node `unsatisfiable`. These are independent upward paths.

### Absorption-First Rule

Each node tries to absorb constraints locally before escalating. Most constraints resolve one layer up. Only fundamental impossibilities cascade. If propagation reaches UC and the use case is unsatisfiable, that's a scope cut — remove or revise the UC.

The escalation path: rise until something absorbs. Only refactor at the absorption point. Everything below gets dirtied from there. Everything above is untouched.

**Examples:**
- IMPL: "Hook model is synchronous" → marks ARCH node `unsatisfiable`. ARCH absorbs (revises to sync pipeline), dirties TEST and IMPL. Done.
- IMPL: "Can't inject multiple reminders per hook" → ARCH can't absorb → marks DES node `unsatisfiable`. DES absorbs (feedback at next hook point), dirties ARCH. Done.
- TEST: "Property X is computationally infeasible" → ARCH can't absorb → marks REQ node `unsatisfiable`. REQ absorbs (weakens invariant), dirties ARCH. Done.

## Depth-First Traversal

The traversal walks a tree of nodes depth-first, top-to-bottom, left-to-right. Each node is a group of items within a layer. The flags on each node determine what happens when the cursor arrives.

### Node States and Flags

Each node has a `status`: `pending | in_progress | refactoring | complete`.

Two flags can be set on a node by other nodes:

- **`dirty`** — set by a parent (or ancestor) when it revises. Tells the node: "your derivation basis changed, re-validate when the cursor arrives." Includes a source reference (e.g., `"L1A revised UC-2 ranking specification"`).
- **`unsatisfiable`** — set by a child that discovers it can't satisfy this node's spec. Tells the node: "absorb this constraint or escalate." Includes the constraint (e.g., `"ARCH could not satisfy REQ-4 AC(3) with pure-Go local similarity"`).

Flags are written ON the impacted node BY whatever discovers the issue. No node reads another node's state to decide what to do — when the cursor arrives, everything it needs is on the node itself.

### Algorithm

The cursor visits nodes depth-first. On arrival at any node, check its state:

**1. Node is `unsatisfiable`:**
- Try to absorb the constraint locally (revise this node's items).
- **If absorbed:** Clear the flag. Refactor the whole layer. Dirty affected child nodes.
- **If can't absorb:** Mark this node's parent `unsatisfiable` with the constraint. Rise to the parent immediately. Do NOT refactor this layer — it may change once an ancestor absorbs.

**2. Node is `dirty`:**
- Re-validate against the parent node.
- **If still valid:** Clear the flag.
- **If changed:** Clear the flag, apply changes. Refactor the whole layer. Dirty affected child nodes.

**3. Node is `pending`:**
- Normal work: derive items at this layer, then group and prioritize.
- **Grouping:** Present 2-3 ways to cluster items (by dependency, domain, complexity, risk) with tradeoffs. Recommend one. This recurs at every layer — never silently pick a grouping.
- **Priority ordering:** Recommend which group to take first and why. The user chooses.

**4. Node is `complete` + clean:**
- Skip. Move to the next node in depth-first order.

### Refactoring

Triggered by **any change** to a layer (absorption, re-derivation, or initial completion of work). Scope: the ENTIRE layer, not just the active group.

- Validate dirty nodes against refactored parents (clear if still valid, re-derive if not)
- Whole-layer consistency check across all nodes at this layer
- Bidirectional satisfaction: does this layer satisfy exactly the layer above?
- For diamond peers (L2 ⟷ L3): cross-check consistency
- Ubiquitous language: same terms for same concepts across all layers
- Surface lessons learned
- Dirty-mark child nodes of anything that changed

### Descend and Backtrack

After completing work at a node (and refactoring if anything changed):
1. **Descend** into the first pending/dirty child node.
2. **On child completion:** The child marks this node `unsatisfiable` if it discovered a constraint. Cursor returns to this node, which processes its own flags per the algorithm above.
3. **After processing flags:** Move to the next pending/dirty sibling (left-to-right).
4. **When all children are complete and clean:** This node completes. Pop up to its parent.

### Final Sweep

After the entire tree is complete, walk depth-first to resolve any orphaned dirty flags. Expected: all clean (safety net only).

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

4. **Flag-based propagation.** `dirty` (downward staleness) and `unsatisfiable` (upward constraints) flags are written on the impacted node. The traversal algorithm processes them on arrival. UC fans out; ARCH has two independent upward paths.

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
4. **Derive REQ and DES from UC** (either order). Check peer consistency.
5. **Converge at ARCH.** Must satisfy REQ + support DES.
6. **TDD.** Tests from ARCH, code to pass them.
7. **Backtrack** for the next group.

### State Persistence

State is persisted in `docs/state.toml` (committed to git). Write-ahead: update after every substantive interaction — do not defer to session end.

The file has four sections:

**`[project]`** — Project name and skill reference.

**`[layers.L1]` through `[layers.L6]`** — Flat registry of all discovered items per layer. L1=UC, L2=REQ, L3=DES, L4=ARCH, L5=TEST, L6=IMPL. Items are added as they're derived.

**`[tree.<node>]`** — Nodes are groups within layers. Each node has: `layer`, `parent` (the parent group node, omitted for root nodes), `items` (member IDs), `status` (pending/in_progress/refactoring/complete), `history` (what happened at this node and why). Optional flags: `dirty` (source reference string) and `unsatisfiable` (constraint string). Omit flags when clean.

Node naming: `L{layer}{letter}` — e.g., L1A, L2A, L3B. Layer number prevents false equivalence across layers. Groups at L2 are independent from groups at L1.

**`[cursor]`** — Current position: `node`, `mode` (descend/work/refactor/backtrack), `next_action` (concrete enough for a fresh session to start immediately), `context_files`.

Example:

```toml
[project]
name = "my-project"
skill = "specification-layers"

[layers.L1]
items = ["UC-1", "UC-2", "UC-3"]

[layers.L2]
items = ["REQ-1", "REQ-2", "REQ-3"]

[layers.L3]
items = []

[tree.L1A]
layer = "L1"
items = ["UC-1", "UC-2"]
status = "complete"
history = "Core pipeline. Foundation for other groups."

[tree.L1B]
layer = "L1"
items = ["UC-3"]
status = "pending"

[tree.L2A]
layer = "L2"
parent = "L1A"
items = ["REQ-1", "REQ-2", "REQ-3"]
status = "complete"
dirty = "L1A revised UC-1 scope"
history = "Derived from UC-1 and UC-2."

[cursor]
node = "L2A"
mode = "work"
next_action = "Re-validate REQ-1 through REQ-3 against revised UC-1"
context_files = ["docs/use-cases.md", "docs/requirements.md"]
```

When the user says "continue" or "resume", read `docs/state.toml` and resume from the cursor's `next_action`.

### Reference

For the research behind this model (8 framework analyses, alternative topologies evaluated, key design decisions with rationale), see `references/research-and-tradeoffs.md`.
