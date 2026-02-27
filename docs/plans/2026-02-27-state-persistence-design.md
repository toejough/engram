# State Persistence Design

## Problem

The specification-layers skill needs persistent state that captures a tree traversal with grouping at every layer, dirty flags, unsatisfiability propagation, and cursor position. The current `docs/state.md` tracks linear phases and can't represent this.

## Format

TOML at `docs/state.toml`. Committed to git. Structured, parseable, clear headers.

## Data Model

### Project metadata

```toml
[project]
name = "engram"
skill = "specification-layers"
```

### Layers: flat registry of all discovered items

Items are added to a layer as they're derived. This is the complete inventory — grouping is separate.

```toml
[layers.L1]  # UC
items = ["UC-1", "UC-2", "UC-3", "UC-4", "UC-5", "UC-6", "UC-7", "UC-8", "UC-9", "UC-10", "UC-11", "UC-12", "UC-13", "UC-14"]

[layers.L2]  # REQ
items = ["REQ-1", "REQ-2", "REQ-3", "REQ-4", "REQ-5", "REQ-6", "REQ-7", "REQ-8", "REQ-9", "REQ-10", "REQ-12", "REQ-13", "REQ-14", "REQ-15", "REQ-16", "REQ-17", "REQ-18", "REQ-19", "REQ-20", "REQ-21"]

[layers.L3]  # DES
items = []

[layers.L4]  # ARCH
items = []

[layers.L5]  # TEST
items = []

[layers.L6]  # IMPL
items = []
```

### Tree: nodes are groups within layers

Each node is a group created during the "group and prioritize" step at its layer. Nodes form a tree via `parent`. Root nodes (L1 groups) have no parent.

```toml
[tree.L1A]
layer = "L1"
items = ["UC-1", "UC-2", "UC-3"]
status = "complete"       # pending | in_progress | refactoring | complete
history = "Core pipeline: write, read, correct. Foundation for all other groups."

[tree.L1B]
layer = "L1"
items = ["UC-7", "UC-8", "UC-9", "UC-10"]
status = "pending"
history = "Evaluation & quadrant actions."

[tree.L1C]
layer = "L1"
items = ["UC-4", "UC-5", "UC-6", "UC-12"]
status = "pending"
history = "Promotion & lifecycle."

[tree.L1D]
layer = "L1"
items = ["UC-11", "UC-13", "UC-14"]
status = "pending"
history = "Infrastructure."

[tree.L2A]
layer = "L2"
parent = "L1A"
items = ["REQ-1", "REQ-2", "REQ-3", "REQ-4", "REQ-5", "REQ-6", "REQ-7", "REQ-8", "REQ-9", "REQ-10", "REQ-12", "REQ-13", "REQ-14", "REQ-15", "REQ-16", "REQ-17", "REQ-18"]
status = "complete"
history = "REQ-1 through REQ-18. Validated, refactored. TF-IDF replaced with local similarity. REQ-4 expanded to ranking behavior."

[tree.L3A]
layer = "L3"
parent = "L1A"
items = []                # DES items not yet derived
status = "in_progress"
history = "Starting horizontal pass for Group A UCs."
```

### Node flags

Flags are written ON the impacted node BY whatever discovers the issue. No node reads another node's state.

**dirty** — written by a parent (or ancestor) when it revises. Tells the node: "your derivation basis changed, re-validate when the cursor arrives."

```toml
[tree.L2A]
dirty = "L1A revised UC-2 ranking specification"
```

**unsatisfiable** — written by a child that discovers it can't satisfy this node's spec. Tells the node: "absorb this constraint or escalate to your parent."

```toml
[tree.L1A]
unsatisfiable = "ARCH could not satisfy REQ-4 AC(3) with pure-Go local similarity"
```

When neither flag is set, omit them (TOML absence = clean).

### Cursor: current position in the traversal

```toml
[cursor]
node = "L3A"
mode = "work"             # descend | work | refactor | backtrack
next_action = "Horizontal pass: design interaction primitives across UC-1, UC-2, UC-3"
context_files = ["docs/use-cases.md", "docs/requirements.md"]
```

## Traversal Rules

The cursor walks depth-first, top-to-bottom, left-to-right. The flags on each node determine what happens when the cursor arrives.

### On arrival at a node

| Node state | Action |
|-----------|--------|
| `unsatisfiable` set | Try to absorb locally. **If absorbed:** clear flag, refactor whole layer, dirty affected children. **If can't absorb:** mark this node's parent `unsatisfiable`, rise (don't refactor — this layer may change). |
| `dirty` set | Re-validate against parent. **If still valid:** clear flag. **If changed:** clear flag, apply changes, refactor whole layer, dirty children. |
| `pending` | Normal work: derive items, group, prioritize. |
| `complete` + clean | Skip. Move to next node in depth-first order. |

### On node completion

1. If node has pending siblings → cursor moves to parent (backtrack mode)
2. At parent: if parent layer unchanged → move to next sibling
3. At parent: if parent revised (absorbed constraint) → refactor whole layer first, then move to next sibling
4. If no pending siblings → pop up again (parent completes, check grandparent)

### Refactoring

Triggered by **any change** to a layer (absorption, re-derivation, or initial completion of work). Scope: the ENTIRE layer, not just the active group. Steps:
- Validate dirty nodes against refactored parents
- Resolve upward constraints
- Whole-layer consistency check
- Bidirectional satisfaction with layer above
- Diamond peer consistency (L2 and L3)
- Ubiquitous language check
- Surface lessons
- Dirty-mark descendants of anything changed

### Escalation path

Rise until something absorbs. Only refactor at the absorption point. Everything below gets dirtied from there. Everything above is untouched.

```
L6 discovers problem → marks L5 unsatisfiable
L5 can't absorb → marks L4 unsatisfiable (no refactor at L5)
L4 absorbs → clears flag, refactors L4, dirties L5 and L6
```

## Scope Boundaries

**In scope (state.toml):**
- Traversal tree structure, cursor position, dirty/unsatisfiable flags
- Per-node history (what happened at this node and why)
- Layer item registries
- Context file references

**Out of scope (separate concerns):**
- Session logs → local to developer, not committed
- Session-end hooks → separate design (command hooks, not prompt hooks)
- The specification-layers skill text → updated separately to match this model

## Migration

1. Replace `docs/state.md` with `docs/state.toml`
2. Update CLAUDE.md resume instructions to reference state.toml
3. Update docs/prompt.md persistence rules to reference state.toml
4. Update specification-layers skill "State Persistence" section with this format
5. Replace prompt hooks with command hook for state validation (or remove — write-ahead is primary)
6. Remove session-log prompt hooks (out of scope for committed state)

## Skill Algorithm Rewrite Needed

The specification-layers skill algorithm (SKILL.md lines 108-131) describes the traversal as five steps (group, refactor, descend, backtrack, final sweep) but never mentions dirty flags or unsatisfiability. Those concepts exist only in the "Bidirectional Signal Propagation" section (lines 71-103), disconnected from the traversal steps the LLM actually follows.

**The problem:** The algorithm and the propagation model are described as separate concerns, but they're the same thing. Dirty flags and unsatisfiability aren't a side mechanism — they're how the cursor decides what to do at each node. An algorithm that doesn't integrate them will be followed literally by the LLM, which means propagation gets skipped.

**What needs to change:** The algorithm section should be rewritten as a single unified traversal that handles all node states:

1. **Arrive at a node.** Check its flags:
   - `unsatisfiable`: absorb locally or escalate (mark parent, rise without refactoring)
   - `dirty`: re-validate against parent, clear or re-derive
   - `pending`: normal work (derive items, group, prioritize)
   - `complete` + clean: skip, move to next node depth-first

2. **On any change to a layer** (absorption, re-derivation, initial work): refactor the ENTIRE layer, dirty affected children.

3. **On child completion:** child marks parent `unsatisfiable` if it discovered a constraint. Cursor moves to parent. Parent processes its own flags per step 1.

4. **Descend** into the next child. **Backtrack** when all children complete.

5. **Final sweep** after full tree: depth-first walk to resolve orphaned flags.

The "Bidirectional Signal Propagation" section should become supporting explanation for why the algorithm works this way, not a parallel description of a separate mechanism.

## What This Replaces

- `docs/state.md` (freeform markdown with linear phase tracking)
- The two failing Stop/PreCompact prompt hooks that tried to infer phase from file existence
- The vague "update a state file with..." instruction in the specification-layers skill
