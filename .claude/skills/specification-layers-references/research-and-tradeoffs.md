# Research and Tradeoffs: Specification Layer Model

This document captures the research, alternative models, and key decisions behind the diamond-topology specification layer model. Read this to understand not just what the model is, but why it's shaped this way and what alternatives were rejected.

## The Core Question

How should specification layers (use cases, requirements, design, architecture, tests, implementation) relate to each other? Specifically: when two solution-space layers (interaction design and system architecture) both derive from a common problem-space ancestor (use cases), what topology captures the actual dependencies?

## Framework Analysis

Eight established frameworks were evaluated for how they handle parallel derivations from a common ancestor.

### Frameworks Supporting Parallel/Peer Views

**4+1 Architectural View Model (Kruchten).** Star topology. Four peer views (Logical, Process, Development, Physical) are independent of each other. Use cases sit at the center as "+1", connecting and validating all four. No view derives from another. *Key insight:* Multiple architectural concerns can be peer views derived from a common ancestor without derivation relationships between them.

**IEEE 42010 (Architecture Description).** Graph with typed edges. Views are peers connected by *correspondence rules* — typed consistency constraints (consistency, refinement, traceability, dependency). No required ordering between views. The 2022 revision adds "Architecture Aspects" for cross-cutting characteristics. *Key insight:* Correspondence rules formalize consistency checks between parallel layers without imposing sequential ordering. This is the most formally rigorous framework.

**Zachman Framework.** 6x6 matrix. Rows are perspectives (Planner -> Worker); columns are interrogatives (What, How, Where, Who, When, Why). Columns are explicitly independent — no column derives from another. All cells in a row must be horizontally aligned. *Key insight:* Independent dimensions can exist at the same abstraction level. A classification scheme, not a process model.

**OOUX (Object-Oriented UX).** Diamond topology. Research produces an Object Map (shared vocabulary), which fans out into parallel interaction design and system architecture tracks. Developers can start architecture before wireframes exist. *Key insight:* Explicitly models the fan-out from a common artifact into parallel design and architecture tracks. Closest practical match to the diamond model.

**Problem Frames (Michael Jackson).** Flat graph of peer subproblems. Explicitly rejects hierarchical decomposition — subproblems are intentionally independent, sharing interfaces but not deriving from each other. *Key insight:* The problem space should be decomposed in parallel, not hierarchically.

**Twin Peaks (Nuseibeh 2001).** Two interleaved spirals. Requirements and architecture are co-evolving peers, not sequential. Development zigzags between them at progressively deeper levels. *Key insight:* Two specification activities can be parallel peers that inform each other. Only models two peaks — doesn't explicitly handle fan-out from a common ancestor.

### Frameworks That Don't Support Parallel Derivation

**V-Model.** Linear chain with mirrored verification. Does not natively distinguish parallel design concerns. The Multi-V variant (VDI 2206) adds discipline-specific branches but those are hardware/software/mechanical, not interaction/architecture.

**RUP (Rational Unified Process).** Linear chain of models connected by use-case realizations. Architecture-centric — does not give interaction design the same status as system architecture.

### Synthesis

No framework argues for strictly sequential "first interaction design, then architecture" or vice versa. Every framework that handles the parallel-derivation case models it as peer views, not a chain.

The strongest foundation: **IEEE 42010's correspondence model** (formalized consistency rules between peer views) applied to a **4+1-style topology** (use cases at the center, peer views derived from them). OOUX provides the practical fan-out mechanism.

## Alternative Topologies Evaluated

### Linear Chain (UC -> REQ -> DES -> ARCH -> TEST -> IMPL)

The initial model. Each layer derives from the one above.

**Rejected because:** DES traces to UC (designs interaction model satisfying user goals), not to REQ. ARCH traces to REQ (enables invariants), not to DES. The linear chain creates false dependencies — it implies DES derives from REQ and ARCH derives from DES, which misrepresents the actual relationships.

### Pure DAG (Every Layer Traces to Every Relevant Ancestor)

Tests trace directly to REQ, DES, and ARCH. ARCH traces to REQ and DES independently.

**Rejected because:** Too many trace links to manage. "Trace to immediate parent only" is a simpler rule that catches gaps (if ARCH only makes sense by referencing UC directly, there's a missing REQ). The diamond provides the right structure with cleaner traceability.

### Merged SPEC Layer (UC -> SPEC -> TEST -> IMPL)

Collapse REQ, DES, and ARCH into a single "specification" layer.

**Rejected because:**
- *Loses the problem/solution boundary.* REQ is problem space (what must be true, independent of how). DES and ARCH are solution space. Merging blurs a fundamental distinction.
- *Loses independent derivation.* REQ and DES have different processes (REQ: per-UC extraction; DES: horizontal-first UX coherence). Merging forces simultaneous work on different concerns.
- *Different stability profiles.* REQ stabilizes early (invariants are stable). DES evolves with better interactions. ARCH evolves with technical constraints. Merging means a late DES change dirties stable REQs.
- *Less precise dirty flags.* Any change to any aspect dirties the whole SPEC layer.
- *Conflated upward propagation.* If SPEC discovers a UC is unsatisfiable, unclear whether it's an interaction or invariant problem.

### Tests as a Separate Layer per Source (Three TEST Layers)

Property tests alongside REQ, example tests alongside DES, integration tests alongside ARCH.

**Rejected because:** Before ARCH exists, the API surface is undefined — test code can't be written. Test specifications exist in each layer (REQ has acceptance criteria, DES has scenarios, ARCH has contracts), but executable tests require ARCH's convergence. The resolution: ARCH must be comprehensive enough to be the sole test source, with each test type traceable through ARCH back to its origin layer.

### TEST Absorbed into IMPL (Five Layers)

Eliminate TEST as a separate layer — it's just the TDD red phase inside IMPL.

**Considered viable but rejected for document-first projects** where spec layers produce documents (not code). TEST as a separate layer marks the transition from documents to Go code. It also allows validating test completeness ("do my tests cover all of ARCH?") before implementation begins. For code-first projects, merging TEST into IMPL may be appropriate.

## Key Design Decisions

### Decision 1: DES Traces to UC, Not REQ

**Context:** REQ and DES both derive from UC. In the linear chain, DES would trace to REQ.

**Decision:** DES traces directly to UC. REQ and DES are peers in the diamond.

**Rationale:** If DES traced to REQ, the interaction model would be constrained by the invariants — backwards directionality. Requirements should be implementable through any coherent interaction model. DES answers "how do users experience this?" independently of REQ's "what must be true?"

**Correspondence rule:** REQ and DES are consistency-checked against each other (neither can contradict the other), but this is a peer check, not a derivation link.

### Decision 2: ARCH as the Convergence Point (Tests Derive from ARCH)

**Context:** Tests verify REQ invariants, DES scenarios, and ARCH boundaries. Should they trace to all three?

**Decision:** Tests derive from ARCH only. ARCH must be comprehensive enough to reflect both REQ invariants (as behavioral contracts) and DES scenarios (as interaction protocols).

**Rationale:** If a REQ invariant isn't reflected in ARCH, that's an ARCH gap — caught by the ARCH<->REQ consistency check before tests are written. Same for DES scenarios. This gives tests a clean single parent while forcing ARCH to be complete.

**Consequence:** ARCH must be more than box-and-arrow diagrams. It specifies component boundaries (structural), behavioral contracts between components (from REQ), and external interaction protocols (from DES). This is what IEEE 42010 describes as a multi-viewpoint architecture description.

### Decision 3: Bidirectional Dirty Flags (Not Feasibility Checkpoints)

**Context:** The Twin Peaks model identified that problem-space and solution-space co-evolve. Initial approach: add a 30-minute feasibility checkpoint between DES and ARCH.

**Decision:** Replace the checkpoint with bidirectional dirty flags — a uniform mechanism at every layer boundary.

**Rationale:** The feasibility check was an ad-hoc band-aid at one boundary. Constraints can be discovered at any boundary. The upward propagation signal ("I can't satisfy your specification") is the same pattern at every layer. Making it a native capability of the tree model eliminates special-case checkpoints.

**Key principle:** Absorption-first. Each layer absorbs constraints locally before escalating. Most resolve one layer up. Cascading across multiple layers is exceptional — just like arc consistency in constraint satisfaction problems, where full propagation is rare after initial setup.

**Two flags on nodes:**
- `dirty` (downward): set by parent when it revises; tells child "your derivation basis changed, re-validate"
- `unsatisfiable` (upward): set by child that discovers it cannot satisfy parent's spec; tells parent "absorb this constraint or escalate"

Flags are written ON the impacted node BY whatever discovers the issue. No node reads another node's state.

### Decision 4: Horizontal-First DES (UX Coherence Before Per-UC Verification)

**Context:** DES could be done per-UC (walk each use case through a scenario) or horizontally-first (design interaction primitives across all UCs, then verify per-UC).

**Decision:** Horizontal first. Design the interaction model across all UCs before verifying individual scenarios.

**Rationale:** Without horizontal coherence, individually correct UC scenarios can produce an incoherent product — different feedback formats, inconsistent proposal patterns, divergent communication channels. The horizontal pass establishes the vocabulary the product speaks in. The vertical pass then verifies each UC works within that vocabulary.

**Directionality:** Primitives serve UCs, not the reverse. If a primitive can't satisfy a UC, fix the primitive. UCs that are fundamentally incoherent are the exceptional case, handled by upward constraint propagation.

### Decision 5: Ubiquitous Language as Cross-Cutting Alignment

**Context:** How to ensure concepts flow coherently from use cases to code?

**Decision:** Enforce the same terminology across all layers. If UC says "reconciliation," REQ says "reconciliation," DES shows reconciliation happening, ARCH defines a `Reconciler`, tests call `TestReconciliation`, code has `reconcile()`.

**Rationale:** The cheapest alignment mechanism. Changes at any layer are grep-able across all layers. A terminology change that doesn't propagate is a traceability gap — visible and fixable. From DDD: ubiquitous language reduces translation errors between stakeholders, specifications, and code.

### Decision 6: Layer Standards (Lateral Injection)

**Context:** Technology choices ("use imptest," "DI everywhere," "Pure Go, no CGO") don't trace to UC. The model originally required every item to trace to its parent layer, forcing artificial traceability.

**Decision:** Items at any layer can enter laterally from professional judgment as "standards" rather than being derived from the parent layer. Items tagged `standard` (vs. implicit `derived`) with a rationale field.

**Rationale:** Not all items trace upward. "Pure Go, no CGO" is an architecture standard from organizational/technical constraints, not derived from use cases. The tag distinguishes purposeful injections from superfluous specs.

**Rules during REFACTOR:**
- Derived items must trace to parent.
- Standards do not trace to parent, but MUST be consistent with derived items and each other. Conflicts must be resolved.
- Items neither derived NOR marked standard = candidates for cutting (superfluous drift).

**Examples by layer:**
- L2: "Design language uses 4px corners" (DES standard), "All data ops auditable" (REQ standard from org policy)
- L3: "Pure Go, no CGO" (tech constraint), "DI everywhere" (architecture standard), "imptest for mocking" (tooling standard)
- L4: "Default to property-based testing" (methodology standard)
- L5: "gofmt on all files" (coding standard)

### Decision 7: L4 as TEST LIST (Prose Behavioral Analysis, Not Code)

**Context:** What does L4 produce — executable test code or prose specifications?

**Evolution:** This went through four phases:
1. Initial: L4 produced Go test files with `t.Skip()` stubs. User corrected — "the tests are the spec, not stubs."
2. L4 produced real Go test files with failing assertions. User corrected — "tests should fail to compile, not just fail assertions."
3. Real Go test code referencing nonexistent types caused massive linter noise (`BrokenImport`, `UndeclaredName`, `go mod tidy` warnings). This is unsolvable — the tests ARE supposed to fail to compile.
4. Research resolution: every TDD methodology recognizes an intermediate artifact between architecture and test code.

**Research basis:**
- **Beck's Canon TDD (2023):** The test list ("behavioral analysis") is the FIRST step, before any code.
- **Freeman/Pryce (GOOS):** Double-loop TDD — outer acceptance test, inner unit TDD. The outer test is defined before inner tests are written.
- **BDD (Behavior-Driven Development):** Given/When/Then scenarios are a specification artifact that precedes test code.
- **DDD tactical patterns:** Deliberate specification gap between aggregates and function-level behavior.

**Decision:** L4 produces a prose test list (behavioral analysis) using a BDD format adapted for property-based testing. L5 does standard TDD: RED (write actual test code, it fails to compile), GREEN (implement), REFACTOR (clean up).

**Key insight on phase confusion:** Phases mean different things depending on which layer's perspective you're in. L4 GREEN = writing tests that won't compile = L5 RED. The skill needed explicit clarification: "the tests can/should be failing, and even failing to compile, because there is no supporting implementation yet. That's RED from the implementation perspective, but GREEN from the test layer perspective."

### Decision 8: BDD-for-PBT Format (Test List Notation)

**Context:** What format should L4 test list entries use?

**Decision:** Given/When/Then triplets adapted for property-based testing, where each triplet represents one interaction boundary mirroring imptest's coroutine model.

**Format rules:**
- **Given** = always about setting up property-generated inputs (function args or mock response values). "any" means rapid generates it.
- **When** = always an external actor pushing inputs into the function. Actors are explicit: "test calls ReconcileRun," "ReconcileRun calls store.FindSimilar."
- **Then** = always validating the response from the function under test.

**Key insight:** Mock responses are an additional input vector that needs property generation. The second triplet's "Given" generates mock return values. With imptest mocks, you control responses dynamically — those should be treated as an additional input vector for validating business logic.

**Defaults:** Property-based by default ("for all" quantification). Example-based entries require explicit justification (specific edge case, error condition, DES scenario walkthrough). Property type annotations (Round-trip, Ordering, Conservation, Idempotence, Invariant, Boundary, Equivalence) are available for clarity.

### Decision 9: Verification Types (Multi-Tier Testing)

**Context:** "No CGO" is verified by a build constraint, not a unit test. "Use SQLite" is verified by dependency inspection. Should everything still be "tested"?

**Research:** Systems engineering IADT taxonomy (Inspection, Analysis, Demonstration, Test). User rejected the idea that some items aren't tested: "there should always be a test of some kind, even if that test is just inspection by an LLM." Reframed as: different things are tested differently, with different cost/automation tiers.

**Also from "Building Evolutionary Architecture":** Every architectural characteristic should have at least one fitness function, but fitness functions vary in mechanism. Tests are ONE kind. Linters, build constraints, and architecture tests are others.

**Decision:** Four verification types:

| Type | What it validates | Speed | When it runs |
|------|------------------|-------|-------------|
| Unit | Behavior — pure data in, data out | Fast | Always, no tags |
| Integration | Wiring — data flows through actual production connections | Slow | Tagged `integration`, only when explicitly asked |
| Linter | Structure/content — code follows rules/standards | Fast | During REFACTOR, always |
| LLM | LLM behavior — validates the LLM itself behaves as expected | Slow, expensive | Tagged `llm`, only when explicitly asked |

**Process rule:** Interview the user about what verification types exist in their project. Do not assume all four are needed.

### Decision 10: User Tools at Every Layer

**Context:** Should deterministic tooling (linters, test runners, build checks) only run at L5?

**Decision:** At each layer's GREEN and REFACTOR steps, the user may specify a command to run. If specified, run it FIRST — before any manual/semantic analysis. This applies at all layers, not just IMPL.

**Discovery protocol (when arriving at a new layer for the first time):**
1. Research what deterministic tools might be relevant for this layer and project stack
2. Suggest options to the user
3. Ask what tool(s) they want at GREEN and REFACTOR for this layer
4. Record the answer in `state.toml` — "Don't assume. Don't skip the question. Record the answer clearly even if it's 'none.'"

### Decision 11: Per-Layer Grouping (Groups Don't Flow Through)

**Context:** When items are derived at a new layer, how should they be grouped?

**Decision:** Grouping is independent per-layer. There is no "Group A" that flows from L1 through L5. Each layer makes its own grouping decisions based on dependency, domain, complexity, or risk at that layer's abstraction level.

**Node naming:** `L{layer}{letter}` (e.g., L1A, L2A, L3B). Layer number in the name prevents false equivalence across layers.

**Process:** Present 2-3 clustering options with tradeoffs. Recommend one. Never silently pick a grouping or priority. After grouping is chosen, recommend priority ordering for depth-first traversal.

**Orphaned children on regroup:** When regrouping changes a group's membership, dirty-mark the children of disrupted groups. At the next layer refactor, orphaned children must be assigned new parents or cut.

### Decision 12: Reconciliation vs. Deduplication (Engram-Specific)

**Context:** Two mechanisms prevent duplicate memories. Are they the same?

**Decision:** Keep separate.

- **Reconciliation** (REQ-5, REQ-14): happens when a new learning enters the system. Asks "does a similar memory already exist?" If yes, enrich that memory instead of creating a duplicate. Uses local similarity + haiku overlap gate. The general merge-or-create decision.
- **Deduplication** (REQ-18): narrower — checks that session-end extraction (UC-1) doesn't re-extract corrections already captured mid-session (UC-3). A pre-filter ("skip this learning, we already handled it mid-session") that runs BEFORE reconciliation.

**Rationale:** Dedup is an optimization — skip the haiku call entirely for learnings known to be already handled because there's a session log of mid-session corrections. Reconciliation would catch the overlap anyway, but the explicit dedup step is cheaper.

## Engram Architecture Decisions

These are project-specific decisions that emerged during L3 (ARCH) work.

### SQLite with FTS5 (ARCH-1)

Single database file. Pure Go driver (`modernc.org/sqlite`, no CGO). FTS5 provides BM25 ranking for free, covering local similarity needs without a separate TF-IDF index.

**Alternatives rejected:** file-per-memory (no query capability), separate TF-IDF index + flat files (FTS5 subsumes TF-IDF), bbolt/badger (no full-text search).

**Tradeoff noted:** Lesson 24 in `docs/lessons.md` argues for transparent file storage over databases when access patterns allow. SQLite won because FTS5 eliminated an entire subsystem (custom TF-IDF), and the project needed query capability.

### Single Binary with Subcommands (ARCH-2)

`engram extract`, `engram correct`, `engram catchup`. Each hook invocation is a short-lived process. No daemon.

**Tradeoff:** Daemon would allow persistent in-memory indices but adds process management complexity. Short-lived binary is simpler; write path latency budget is generous. Can revisit if latency becomes a problem.

### Extraction Pipeline as Injected Stages (ARCH-3)

Pipeline: Enricher (sonnet) -> QualityGate (deterministic) -> Classifier (haiku) -> Reconciler (local similarity + haiku) -> Store. Each stage independently testable via DI.

### Pattern Corpus for Correction Detection (ARCH-4)

JSON file of 15 initial regex patterns. New patterns from catch-up are added as candidates (promoted after matching in N future sessions). Deterministic first — LLM-based detection rejected for hook-time latency.

### Shared Reconciler (ARCH-5)

Single reconciler used by both extraction and correction detection. Uses FTS5 BM25 for candidate retrieval, then haiku LLM for overlap gating. K is a budget (number of candidates to evaluate), not a threshold.

### Audit Log as Structured Key-Value (ARCH-7)

Append-only file, one line per entry. Format: `TIMESTAMP OPERATION ACTION key1=value1 key2=value2`. Not SQLite (heavier than needed). Not JSON lines (harder to scan visually).

### DI Wiring Validation (ARCH-8)

Constructors verify all required dependencies are non-nil. No silent nil degradation — the single most pervasive failure in the predecessor system (projctl).

### Model Hierarchy

Deterministic (hash) -> TF-IDF/pattern matching -> haiku -> sonnet -> opus. Each step up costs more and should only be used when the previous step cannot answer. Enricher uses sonnet, Classifier and OverlapGate use haiku. Pattern corpus uses regex (deterministic).

## Test Structure Decisions

### Testing Stack

- **imptest** (`github.com/toejough/imptest`) — generated mocks from interfaces via `//go:generate impgen`. Only mock I/O dependencies (LLM, database, persistent state). Wire real pure-functional deps together.
- **rapid** (`pgregory.net/rapid`) — property-based testing. Generates both function inputs AND mock return values (mock responses are an input vector).
- **gomega** (`github.com/onsi/gomega`) — readable assertions via `NewWithT(t)`.

### Phase Discipline

- No production code at L4. Tests should fail to compile because there is no code that implements them.
- No imptest references in production code — it's a testing utility.
- Integration tests (wiring, store) use real deps directly, tagged with `//go:build integration`.

### Implementation Ordering (L5)

Bottom-up from simplest/fewest dependencies to most complex: audit -> store -> reconcile -> corpus -> extract -> correct -> catchup -> wiring -> hooks. Each package's tests become green before moving to packages that depend on it.

## Requirements Lessons (Surfaced During L2 Work)

These emerged from specific mistakes during requirements derivation and apply to any specification process:

1. **Definitions need observable conditions, not labels.** A/B/C confidence tiers were labeled "uncorrected" vs "unvalidated" (near-synonyms). Fix: describe the observable mechanism.

2. **Requirements must demand wiring, not just capabilities.** "When a hook fires, surface memories" is satisfiable by an unconnected function. Rewrite to demand end-to-end wiring.

3. **Fix ambiguity at the source.** When UC is ambiguous, fix the UC first, then derive requirements. Do not derive requirements from architectural preferences.

4. **Do not import constraints the UC did not state.** Performance numbers pulled from global design rules and attributed to specific UCs create false traceability.

5. **Do not re-derive what validated artifacts explicitly state.** When a validated UC says "Go binary," the requirement should reflect it, not second-guess it from first principles.

6. **Requirements must be more specific than their source use cases.** Downstream artifacts refine, they do not summarize.

7. **Unknown thresholds need a decision mechanism and a data plan, not a placeholder.** "Above a similarity threshold" is a non-requirement.

8. **Evaluation criteria must trace to system purpose.** Requirements that optimize for internal metrics without connecting to user outcomes are architecturally ungrounded.

## Process Lessons Learned

These emerged during development of the model and apply to any multi-layer specification process:

1. **Specifications co-evolve bidirectionally.** Lower layers discover constraints upper layers didn't account for. Build upward propagation into the process as a native capability, not as checkpoints at specific boundaries.

2. **The problem/solution boundary is load-bearing.** Merging problem-space layers (what must be true) with solution-space layers (how to achieve it) loses the ability to change solutions without re-questioning requirements.

3. **Peer layers need consistency checks, not derivation links.** REQ and DES don't derive from each other, but they can contradict each other. Consistency checking is the standard layer check applied between peers — not a new mechanism.

4. **Each layer has a different stability profile.** Requirements stabilize early. Design evolves. Architecture adapts to technical constraints. Keeping them separate allows precise dirty-flagging.

5. **Tests verify the convergence point, not individual source layers.** If the convergence layer (ARCH) properly reflects all source layers (REQ, DES), tests need only one parent. Gaps in ARCH are caught by consistency checks before tests are written.

6. **The diamond pattern recurs.** Whenever two concerns independently derive from a common ancestor and must later converge, the diamond applies. This can happen at any scale — project-level (REQ/DES from UC) or component-level (API design/data model from component requirements).

7. **REFACTOR is whole-layer, not just the active group.** When a change is absorbed, refactor the ENTIRE layer because changes to one group's items can affect consistency with other groups. Escalation does NOT trigger refactoring — rise until something absorbs, THEN refactor at the absorption point.

8. **Linter noise during expected failure states is a real problem.** Go test code at L4 that references nonexistent types causes massive unsolvable linter noise. Prose test lists at L4 eliminate it. Code-as-spec creates unsolvable tension with tooling.

9. **Mock return values are an input vector.** Property-based testing should generate both function inputs AND mock return values with rapid. The mock response is not just setup — it's a dimension of the property being tested.

10. **Group and prioritize at every layer, not just UC.** Present 2-3 options with tradeoffs. Never silently choose. Priority ordering matters — write path before read path enables immediate UAT.

11. **Phase discipline is critical.** No production code during the test layer. No test utilities referenced in production code. The boundary between "what layer owns this artifact" must be respected.

12. **State persistence must be write-ahead.** Update `docs/state.toml` after every substantive interaction. Sessions can die at any time (/exit, /clear, crash). The cursor's `next_action` must be specific enough that a fresh session with NO context can start immediately from "continue."

## Research Citations

| Source | Finding | How Applied |
|--------|---------|-------------|
| 4+1 Architectural View Model (Kruchten) | Peer views from common ancestor | REQ/DES as peer views |
| IEEE 42010 (2022) | Correspondence rules between peer views | REQ/DES consistency checks |
| OOUX | Diamond fan-out from shared vocabulary | Direct model inspiration |
| Twin Peaks (Nuseibeh 2001) | Problem/solution co-evolve as peers | Bidirectional dirty flags |
| Problem Frames (Michael Jackson) | Peer subproblems, not hierarchy | L2 grouping philosophy |
| Beck's Canon TDD (2023) | Test list ("behavioral analysis") precedes code | L4 as prose behavioral analysis |
| Freeman/Pryce (GOOS) | Double-loop TDD: outer acceptance, inner unit | L4/L5 relationship |
| Building Evolutionary Architecture | Fitness functions for architectural characteristics | Multi-tier verification types |
| DDD (Evans) | Ubiquitous language, tactical patterns | Cross-layer terminology, L4/L5 gap |
| Arc consistency (CSP theory) | Full propagation rare after initial setup | Absorption-first rule for dirty flags |

## Tradeoffs Summary

| Tradeoff | Resolution | Rationale |
|----------|-----------|-----------|
| Linear chain vs diamond topology | Diamond | False dependency if DES derives from REQ |
| Pure DAG vs diamond | Diamond | Too many trace links; diamond provides structure |
| Merged SPEC layer vs separate REQ/DES/ARCH | Separate | Loses problem/solution boundary, stability profiles |
| Three TEST layers vs one from ARCH | One from ARCH | API surface undefined before ARCH; forces ARCH completeness |
| TEST as separate layer vs absorbed into IMPL | Separate layer | Document-first projects need completeness validation |
| Feasibility checkpoints vs bidirectional dirty flags | Dirty flags | Uniform mechanism at every boundary |
| Per-UC design vs horizontal-first DES | Horizontal-first | UX coherence prevents incoherent product |
| Test code at L4 vs prose test list | Prose test list | Unsolvable linter noise; every methodology recognizes this artifact |
| Example-based vs property-based tests default | Property-based default | Better coverage; example-based must be explicitly justified |
| ONNX embeddings vs TF-IDF | TF-IDF (pure Go) | CGO complexity, platform deps, hard to test |
| SQLite+FTS5 vs flat files | SQLite+FTS5 | FTS5 subsumes TF-IDF; query capability |
| Long-running daemon vs short-lived binary | Short-lived | No process management; revisit if latency demands |
| LLM vs deterministic correction detection | Deterministic (regex) | Model hierarchy: deterministic first, cheaper, lower latency |
| Single reconciler vs per-use-case | Single shared | Same logic, avoid duplication |
