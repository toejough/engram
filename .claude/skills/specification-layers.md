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
        UC                   L1: user goals
       / \
    REQ   DES              L2: invariants + interaction model (same layer)
       \ /
       ARCH                  L3: system structure (converges L2)
        |
    TEST LIST                L4: behavioral analysis (prose, PBT default)
        |
       IMPL                  L5: one-test-at-a-time TDD
```

Five layers. REQ and DES are peer item types at the same layer (L2), both derived from UC. ARCH converges both. The bottom two are linear.

### Layer Descriptions

**L1: UC (Use Cases)** — User goals and interaction flows. The common ancestor from which REQ and DES independently derive. Discovers scope. Format: numbered UC-N entries with actor, starting state, end state, key interactions.

**L2: REQ + DES (Requirements and Design)** — Two peer item types at the same layer, both derived from UC. Grouped together — a single L2 group can contain both REQ-N and DES-N items. Consistency between REQ and DES items is handled by normal layer refactoring, not a special mechanism.

- **REQ (Requirements)** — Atomic, testable invariants. Problem space: what must be true regardless of implementation. Traces to UC. Format: numbered REQ-N entries with acceptance criteria and verification tier.
- **DES (Design)** — Interaction model and UX specification. Traces to UC (not to REQ — they are peers). Two passes:
  1. *Horizontal (UX coherence):* Study all UCs as a whole. Design interaction primitives that satisfy all of them coherently — unified feedback formats, proposal patterns, communication channels. The goal: one product, not N independent features.
  2. *Vertical (behavioral specification):* Walk each UC through those primitives as concrete scenarios — case studies, mock output, edge cases. Verify the primitives satisfy the UCs. If a primitive can't satisfy a UC, fix the primitive.

**L3: ARCH (Architecture)** — System structure. The convergence point: must satisfy REQ invariants and support DES interactions. Component boundaries, data model, tech decisions, behavioral contracts, interaction protocols. Must be comprehensive enough to be the sole source for the test list.

#### Verification Types

Every item that needs verification specifies its tier. Everything is testable — tiers differ in cost and run frequency:

| Type | What it validates | Speed | When it runs |
|------|------------------|-------|-------------|
| **Unit** | Behavior — pure data in, data out | Fast | Always, no tags |
| **Integration** | Wiring — data/action flows through actual production connections correctly. Not behavior of parts. | Slow | Tagged `integration`, only when explicitly asked |
| **Linter** | Structure/content — code follows rules/standards. Not behavior. | Fast | During REFACTOR, always |
| **LLM** | LLM behavior — validates that the LLM itself behaves as expected given inputs/outputs/tools. Needs a live LLM to execute. | Slow, expensive | Tagged `llm`, only when explicitly asked |

Interview the user about what verification types exist in their project. Don't assume all four are needed.

**L4: TEST LIST (Behavioral Analysis)** — Prose test list derived from ARCH. For each ARCH item, decompose into behavioral variants using Beck's "behavioral composition": break big behavior into pieces such that implementing and verifying those pieces implies the whole.

**Format: BDD adapted for property-based testing.** Each test list entry is a series of Given/When/Then triplets. Each triplet represents one interaction boundary:

- **Given** = always about setting up property-generated inputs (function args or mock response values)
- **When** = always an external actor pushing those inputs into the function (test calling function, or mock responding)
- **Then** = always validating the response from the function under test (function calling a mock, or function returning/panicking)

**Actors are explicit** in When/Then — name who calls what. "test calls ReconcileRun," "ReconcileRun calls store.FindSimilar," "store.FindSimilar responds."

**"any" means property-generated** (rapid generates it). Criteria after "any" constrain the generator or the assertion.

**Default is property-based.** Example-based entries are explicitly justified (edge case, specific error condition, etc.).

**Property type annotations** (optional, for clarity):
- Round-trip, Ordering, Conservation, Idempotence, Invariant, Boundary, Equivalence

**Each entry also carries:**
- Verification type: `[unit | integration | linter | llm]`
- ARCH trace: which ARCH item it verifies
- I/O boundaries: which dependencies are mocked vs. wired real (relevant for mock triplets)

**Example — property-based with mocks (T-27):**
```
T-27: Empty store creates new memory [unit, ARCH-3a]

Given any Learning l, any context ctx
When test calls ReconcileRun with (store, gate, l, ctx)
Then ReconcileRun calls store.FindSimilar with (any ctx, matching l.Content, K > 0)

Given empty results, nil error
When store.FindSimilar responds with (empty results, nil error)
Then ReconcileRun calls store.Create with (any ctx, non-nil Memory)

Given nil error
When store.Create responds with nil error
Then ReconcileRun returns nil error
```

**Example — property-based, simple (T-10):**
```
T-10: Empty transcript produces no memories [unit, ARCH-2]

Given any transcript t
When test calls ExtractRun with (enricher, classifier, store, gate, ctx, t)
Then ExtractRun calls enricher.Enrich with (any ctx, equal to t)

Given empty learnings, nil error
When enricher.Enrich responds with (empty, nil)
Then ExtractRun returns (nil error, any string)
  And ExtractRun never calls classifier, store, or gate
```

**Example — example-based edge case (T-2):**
```
T-2: Create rejects empty confidence [unit, ARCH-1a]

Given a Memory m with confidence = ""
When test calls store.Create with (any ctx, m)
Then store.Create returns non-nil error
```

**Example — pure implementation, no mocks (T-21):**
```
T-21: All 15 initial patterns match expected input [unit, ARCH-4a]

Given each pattern from the initial corpus and its expected matching string
When test calls corpus.Match with the input string
Then corpus.Match returns a non-nil match
```

**L5: IMPL (Implementation)** — TDD with a strong preference for groups of 1 test. Rarely, a small group may make sense (e.g., tests that share setup infrastructure), but default to implementing one test at a time.

For each test list entry (or small group) in priority order:

1. **RED:** Write test(s) from the test list entry. They fail or don't compile. Expected FOR THE CURRENT TEST(S) ONLY — all previous tests still compile and pass.
2. **GREEN:** Implement until they pass. If a user tool is specified for this layer, run it first — before manual/semantic analysis. All previous tests still pass.
3. **REFACTOR:** If a user tool is specified for this layer, run it first — before manual/semantic analysis. Then do semantic consistency review. Fix everything.
4. Next test list entry.

If a test proves unsatisfiable during GREEN, mark the L4 test list node `unsatisfiable` and escalate. Don't modify the test at L5.

#### Expected Failures During One-Test-at-a-Time TDD

- **RED:** Current test(s) have compilation errors (references types that don't exist yet). Expected for the CURRENT TEST(S) ONLY. All previous tests compile and pass.
- **GREEN:** Current test(s) pass. All previous tests pass.
- **REFACTOR:** User tool passes (if specified). Semantic review complete.

First test in a group has the most compilation noise (new packages, types). Diminishes rapidly.

### Why a Diamond, Not a Chain

REQ and DES are parallel derivations from UC — neither derives from the other. They coexist at the same layer because they share the same parent (UC) and the same depth.

- REQ decomposes user goals into testable invariants (system-facing)
- DES designs a coherent interaction model satisfying user goals (user-facing)
- Both trace to UC independently; both are needed before ARCH
- A group at L2 can contain both REQ and DES items — grouping is per-layer, not per-item-type

A linear chain (UC → REQ → DES → ARCH) would imply DES derives from REQ, creating a false dependency and wrong directionality (invariants don't dictate the interaction model). The diamond captures the actual relationships.

### Derivation and Consistency

| Layer | Derives from | Standards can enter | Consistency |
|-------|-------------|-------------------|-------------|
| L1 (UC) | Project purpose | Yes (user workflow prefs) | — |
| L2 (REQ + DES) | UC | Yes (design language, org policies) | REQ/DES cross-checked in REFACTOR |
| L3 (ARCH) | L2 (both REQ and DES) | Yes (tech stack, constraints) | — |
| L4 (TEST LIST) | ARCH | Yes (testing methodology, coverage) | — |
| L5 (IMPL) | TEST LIST | Yes (coding standards, tooling) | — |

REQ/DES consistency is not a special mechanism — it's handled by normal layer refactoring at L2. When you refactor L2, all items at that layer (both REQ and DES) are checked for mutual consistency.

## Traversal Algorithm

The traversal walks a tree of group nodes. Each node is a group of items within a layer, with a single parent node at the layer above. Groups are independent per-layer — there is no "Group A" that flows from L1 through L5. Each layer makes its own grouping decisions.

### The RED → GREEN → REFACTOR Cycle

At every layer, for every group, the work follows this cycle:

**1. RED (assess gaps).** Compare the current layer's content against the parent group's specification (or the core project purpose at L1). Identify what's missing — items that need to be derived, gaps in coverage, stale content.

**2. GREEN (derive and fill).** Derive new items and interview to fill gaps. If a user tool is specified for this layer, run it first — before manual/semantic analysis. If during this work you discover the parent group is unsatisfiable — it demands something impossible or contradictory — mark the parent node `unsatisfiable` with a clear description of the constraint, and rise back up a layer immediately. Do NOT refactor — the layer may change once an ancestor absorbs.

**3. REFACTOR (whole layer).** If a user tool is specified for this layer, run it first — before manual/semantic analysis. Then refactor the ENTIRE layer, not just the active group:
- Whole-layer consistency check across all groups at this layer
- At L2: REQ and DES items checked against each other (normal consistency, not a special mechanism)
- Bidirectional satisfaction: does this layer satisfy exactly the layer above?
- Ubiquitous language: same terms for same concepts across all layers
- Surface lessons learned
- If content of any group changes, dirty-mark its child nodes

**4. (RE)GROUP.** Group items from the whole layer. Refactoring may have changed content enough to disrupt prior groupings. If regrouping changes a group's membership, dirty-mark the children of disrupted groups — they are now parentless. At the next layer refactor, orphaned children must be assigned new parents or cut.

Grouping proposals (present to the user every time):
- Present 2-3 ways to cluster items (by dependency, domain, complexity, risk). Explain the tradeoff of each grouping.
- Recommend one grouping with rationale.
- After the user chooses a grouping, recommend a priority ordering for which group to take deep first and why. The user chooses.

Never silently pick a grouping or priority. This decision recurs at every layer.

**5. DESCEND.** Pick the next incomplete group and descend to the next layer. Start the RED → GREEN → REFACTOR cycle there.

Navigation when descending:
- **Incomplete child groups exist:** Pick the next priority incomplete child and descend.
- **No incomplete children:** Move to the next priority sibling group at this layer.
- **No incomplete siblings:** Backtrack up to the parent and repeat the search.

### User Tools at Every Layer

At each layer's GREEN and REFACTOR steps, the user may specify a command to run (e.g., `go test ./...`, `go vet`, a custom script). If a user tool is specified, run it FIRST — before any manual/semantic analysis. This applies at all layers, not just IMPL.

**Discovering user tools.** When arriving at a new layer for the first time, explicitly:
1. Research what deterministic tools might be relevant for this layer and this project's stack.
2. Suggest options to the user (e.g., "For Go at IMPL, you might want `go test ./...` and `go vet`").
3. Ask the user what tool(s) they want run at GREEN and REFACTOR for this layer.
4. Record their answer in state.toml under the layer so it persists across sessions.

Don't assume. Don't skip the question. Record the answer clearly even if it's "none."

### Groups Are Per-Layer

Every time you derive items at a new layer, you make a fresh grouping decision. A UC group (L1A) spawns L2 items. Those L2 items are grouped independently — the groups at L2 are whatever makes sense for L2's content. An L2 group's parent is the L1 group it was derived from, but L2A is NOT "the L2 portion of Group A." It's its own grouping decision.

A single L2 group can contain both REQ and DES items. Grouping is by dependency, domain, complexity, or risk — not by item type.

### Layer Standards

Layer standards are items at any layer that enter laterally from professional judgment, organizational defaults, or accumulated experience. They are NOT derived from the parent layer.

**Key rules:**
- Items tagged as `standard` (vs. implicit `derived`). Distinguishes purposeful injections from superfluous specs.
- During REFACTOR: derived items must trace to parent. Standards don't — but MUST be consistent with derived items and each other. Conflicts must be resolved.
- Standards propagate downward like any other item.
- Items neither derived NOR marked standard = candidates for cutting (superfluous drift).

**Examples by layer:**
- L2: "Design language uses 4px corners" (DES standard), "All data ops auditable" (REQ standard from org policy)
- L3: "Pure Go, no CGO" (tech constraint), "DI everywhere" (architecture standard), "imptest for mocking" (tooling standard)
- L4: "Default to property-based testing" (methodology standard)
- L5: "gofmt on all files" (coding standard)

### Node States and Flags

Each node has a `status`: `pending | in_progress | refactoring | complete`.

Two flags can be set on a node by other nodes:

- **`dirty`** — set by a parent (or ancestor) when it revises. Tells the node: "your derivation basis changed, re-validate when the cursor arrives." Includes a source reference (e.g., `"L1A revised UC-2 ranking specification"`).
- **`unsatisfiable`** — set by a child that discovers it can't satisfy this node's spec. Tells the node: "absorb this constraint or escalate." Includes the constraint (e.g., `"ARCH could not satisfy REQ-4 AC(3) with pure-Go local similarity"`).

Flags are written ON the impacted node BY whatever discovers the issue. No node reads another node's state to decide what to do — when the cursor arrives, everything it needs is on the node itself.

### Handling Flags on Arrival

When the cursor arrives at a node with flags, handle before normal work:

**Node is `unsatisfiable`:**
- Try to absorb the constraint locally (revise this node's items).
- **If absorbed:** Clear the flag. Run the REFACTOR step for the whole layer. Dirty affected child nodes.
- **If can't absorb:** Mark this node's parent `unsatisfiable` with the constraint. Rise immediately. Do NOT refactor — it may change once an ancestor absorbs.

**Node is `dirty`:**
- Re-validate against the parent node.
- **If still valid:** Clear the flag.
- **If changed:** Clear the flag, apply changes. Run the REFACTOR step for the whole layer. Dirty affected child nodes.

**Node is `complete` + clean:**
- Skip. Move to the next node per navigation rules.

### Absorption-First Rule

Each node tries to absorb constraints locally before escalating. Most constraints resolve one layer up. Only fundamental impossibilities cascade. If propagation reaches UC and the use case is unsatisfiable, that's a scope cut — remove or revise the UC.

The escalation path: rise until something absorbs. Only refactor at the absorption point. Everything below gets dirtied from there. Everything above is untouched.

**Examples:**
- IMPL: "Hook model is synchronous" → marks ARCH node `unsatisfiable`. ARCH absorbs (revises to sync pipeline), dirties TEST and IMPL. Done.
- IMPL: "Can't inject multiple reminders per hook" → ARCH can't absorb → marks L2 group `unsatisfiable`. DES item revised (feedback at next hook point), L2 refactored, dirties ARCH. Done.
- TEST: "Property X is computationally infeasible" → ARCH can't absorb → marks L2 group `unsatisfiable`. REQ item revised (weakens invariant), L2 refactored, dirties ARCH. Done.

### Diamond-Specific Propagation

UC fans out: a UC change dirties L2 groups derived from it (which contain both REQ and DES items). L2 changes dirty ARCH groups. ARCH dirties TEST LIST. TEST LIST dirties IMPL.

ARCH's parent is an L2 group (which contains both REQ and DES items). If ARCH can't satisfy the group, it marks that L2 group `unsatisfiable` — the L2 group then figures out whether the issue is a REQ item, a DES item, or both.

### Final Sweep

After the entire tree is complete, walk depth-first to resolve any orphaned dirty flags. Expected: all clean (safety net only).

## Alignment Mechanisms

1. **Ubiquitous language.** Same terms across all layers. If UC says "reconciliation," code has `reconcile()`. Grep-able when terminology changes at any layer.

2. **Bidirectional traceability.** Each layer traces to its immediate parent(s). Parent covered by at least one child. REQ and DES trace to UC; ARCH traces to L2 (which contains both REQ and DES). If ARCH only makes sense by referencing UC directly, there's a missing REQ or DES entry.

3. **Layer refactoring as consistency enforcement.** REQ/DES consistency, bidirectional satisfaction, and ubiquitous language are all checked during the REFACTOR step — a single mechanism applied uniformly at every layer, not special-case checks.

4. **Flag-based propagation.** `dirty` (downward staleness) and `unsatisfiable` (upward constraints) flags are written on the impacted node. The traversal algorithm processes them on arrival. UC fans out; ARCH has one upward path to L2.

## Applying to a Project

### When to Use

Use this model when a project has enough complexity that:
- Multiple use cases need coherent implementation
- Interaction design and system architecture are distinct concerns
- Traceability from goals to code prevents costly rework
- The project spans multiple sessions (state must persist)

Skip for: single-file fixes, clear requirements with known implementation, prototypes.

### Getting Started

Follow the RED → GREEN → REFACTOR → (RE)GROUP → DESCEND cycle. At L1 (the top), the "parent" is the core project purpose, and GREEN means interviewing the user to discover use cases.

### State Persistence

State is persisted in `docs/state.toml` (committed to git). Write-ahead: update after every substantive interaction — do not defer to session end.

The file has four sections:

**`[project]`** — Project name and skill reference.

**`[layers.L1]` through `[layers.L5]`** — Flat registry of all discovered items per layer. L1=UC, L2=REQ+DES, L3=ARCH, L4=TEST LIST, L5=IMPL. Items are added as they're derived. Each item has an optional `source` field: `"derived"` (default, traces to parent) or `"standard"` (lateral injection from professional judgment). Standards include a `rationale`.

Each layer can also specify `green_tool` and `refactor_tool` — commands to run at GREEN and REFACTOR steps before manual analysis.

**`[tree.<node>]`** — Nodes are groups within layers. Each node has: `layer`, `parent` (the parent group node, omitted for root nodes), `items` (member IDs — can mix item types at L2), `status` (pending/in_progress/refactoring/complete), `history` (what happened at this node and why). Optional flags: `dirty` (source reference string) and `unsatisfiable` (constraint string). Omit flags when clean.

Node naming: `L{layer}{letter}` — e.g., L1A, L2A, L3B. Layer number prevents false equivalence across layers. Groups at L2 are independent from groups at L1. A group is a per-layer decision, not a label that flows through layers.

**`[cursor]`** — Current position: `node`, `mode` (work/refactor/backtrack), `next_action` (concrete enough for a fresh session to start immediately), `context_files`.

Example:

```toml
[project]
name = "my-project"
skill = "specification-layers"

[layers.L1]
items = ["UC-1", "UC-2", "UC-3"]

[layers.L2]
items = ["REQ-1", "REQ-2", "REQ-3", "DES-1", "DES-2"]

[layers.L3]
items = [
  { id = "ARCH-1", source = "derived" },
  { id = "ARCH-10", source = "standard", rationale = "DI prevents test coupling to I/O" },
]

[layers.L4]
items = []

[layers.L5]
items = []
green_tool = "go test -tags sqlite_fts5 ./internal/..."
refactor_tool = "go vet ./internal/... && go test -tags sqlite_fts5 ./internal/..."

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
items = ["REQ-1", "REQ-2", "DES-1"]
status = "complete"
dirty = "L1A revised UC-1 scope"
history = "Core invariants and interaction model. REQ-1/REQ-2 for storage guarantees, DES-1 for surfacing UX."

[tree.L2B]
layer = "L2"
parent = "L1A"
items = ["REQ-3", "DES-2"]
status = "pending"
history = "Evaluation requirements and feedback design."

[cursor]
node = "L2A"
mode = "work"
next_action = "Re-validate REQ-1, REQ-2, DES-1 against revised UC-1"
context_files = ["docs/use-cases.md", "docs/requirements.md"]
```

When the user says "continue" or "resume", read `docs/state.toml` and resume from the cursor's `next_action`.
