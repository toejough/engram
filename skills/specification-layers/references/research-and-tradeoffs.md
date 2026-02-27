# Research and Tradeoffs: Specification Layer Model

This document captures the research, alternative models, and key decisions behind the diamond-topology specification layer model. Read this to understand not just what the model is, but why it's shaped this way and what alternatives were rejected.

## The Core Question

How should specification layers (use cases, requirements, design, architecture, tests, implementation) relate to each other? Specifically: when two solution-space layers (interaction design and system architecture) both derive from a common problem-space ancestor (use cases), what topology captures the actual dependencies?

## Framework Analysis

Eight established frameworks were evaluated for how they handle parallel derivations from a common ancestor.

### Frameworks Supporting Parallel/Peer Views

**4+1 Architectural View Model (Kruchten).** Star topology. Four peer views (Logical, Process, Development, Physical) are independent of each other. Use cases sit at the center as "+1", connecting and validating all four. No view derives from another. *Key insight:* Multiple architectural concerns can be peer views derived from a common ancestor without derivation relationships between them.

**IEEE 42010 (Architecture Description).** Graph with typed edges. Views are peers connected by *correspondence rules* — typed consistency constraints (consistency, refinement, traceability, dependency). No required ordering between views. The 2022 revision adds "Architecture Aspects" for cross-cutting characteristics. *Key insight:* Correspondence rules formalize consistency checks between parallel layers without imposing sequential ordering. This is the most formally rigorous framework.

**Zachman Framework.** 6x6 matrix. Rows are perspectives (Planner → Worker); columns are interrogatives (What, How, Where, Who, When, Why). Columns are explicitly independent — no column derives from another. All cells in a row must be horizontally aligned. *Key insight:* Independent dimensions can exist at the same abstraction level. A classification scheme, not a process model.

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

### Linear Chain (UC → REQ → DES → ARCH → TEST → IMPL)

The initial model. Each layer derives from the one above.

**Rejected because:** DES traces to UC (designs interaction model satisfying user goals), not to REQ. ARCH traces to REQ (enables invariants), not to DES. The linear chain creates false dependencies — it implies DES derives from REQ and ARCH derives from DES, which misrepresents the actual relationships.

### Pure DAG (Every Layer Traces to Every Relevant Ancestor)

Tests trace directly to REQ, DES, and ARCH. ARCH traces to REQ and DES independently.

**Rejected because:** Too many trace links to manage. "Trace to immediate parent only" is a simpler rule that catches gaps (if ARCH only makes sense by referencing UC directly, there's a missing REQ). The diamond provides the right structure with cleaner traceability.

### Merged SPEC Layer (UC → SPEC → TEST → IMPL)

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

**Rationale:** If a REQ invariant isn't reflected in ARCH, that's an ARCH gap — caught by the ARCH↔REQ consistency check before tests are written. Same for DES scenarios. This gives tests a clean single parent while forcing ARCH to be complete.

**Consequence:** ARCH must be more than box-and-arrow diagrams. It specifies component boundaries (structural), behavioral contracts between components (from REQ), and external interaction protocols (from DES). This is what IEEE 42010 describes as a multi-viewpoint architecture description.

### Decision 3: Bidirectional Dirty Flags (Not Feasibility Checkpoints)

**Context:** The Twin Peaks model identified that problem-space and solution-space co-evolve. Initial approach: add a 30-minute feasibility checkpoint between DES and ARCH.

**Decision:** Replace the checkpoint with bidirectional dirty flags — a uniform mechanism at every layer boundary.

**Rationale:** The feasibility check was an ad-hoc band-aid at one boundary. Constraints can be discovered at any boundary. The upward propagation signal ("I can't satisfy your specification") is the same pattern at every layer. Making it a native capability of the tree model eliminates special-case checkpoints.

**Key principle:** Absorption-first. Each layer absorbs constraints locally before escalating. Most resolve one layer up. Cascading across multiple layers is exceptional — just like arc consistency in constraint satisfaction problems, where full propagation is rare after initial setup.

### Decision 4: Horizontal-First DES (UX Coherence Before Per-UC Verification)

**Context:** DES could be done per-UC (walk each use case through a scenario) or horizontally-first (design interaction primitives across all UCs, then verify per-UC).

**Decision:** Horizontal first. Design the interaction model across all UCs before verifying individual scenarios.

**Rationale:** Without horizontal coherence, individually correct UC scenarios can produce an incoherent product — different feedback formats, inconsistent proposal patterns, divergent communication channels. The horizontal pass establishes the vocabulary the product speaks in. The vertical pass then verifies each UC works within that vocabulary.

**Directionality:** Primitives serve UCs, not the reverse. If a primitive can't satisfy a UC, fix the primitive. UCs that are fundamentally incoherent are the exceptional case, handled by upward constraint propagation.

### Decision 5: Ubiquitous Language as Cross-Cutting Alignment

**Context:** How to ensure concepts flow coherently from use cases to code?

**Decision:** Enforce the same terminology across all layers. If UC says "reconciliation," REQ says "reconciliation," DES shows reconciliation happening, ARCH defines a `Reconciler`, tests call `TestReconciliation`, code has `reconcile()`.

**Rationale:** The cheapest alignment mechanism. Changes at any layer are grep-able across all layers. A terminology change that doesn't propagate is a traceability gap — visible and fixable. From DDD: ubiquitous language reduces translation errors between stakeholders, specifications, and code.

## Process Lessons Learned

These emerged during development of the model and apply to any multi-layer specification process:

1. **Specifications co-evolve bidirectionally.** Lower layers discover constraints upper layers didn't account for. Build upward propagation into the process as a native capability, not as checkpoints at specific boundaries.

2. **The problem/solution boundary is load-bearing.** Merging problem-space layers (what must be true) with solution-space layers (how to achieve it) loses the ability to change solutions without re-questioning requirements.

3. **Peer layers need consistency checks, not derivation links.** REQ and DES don't derive from each other, but they can contradict each other. Consistency checking is the standard layer check applied between peers — not a new mechanism.

4. **Each layer has a different stability profile.** Requirements stabilize early. Design evolves. Architecture adapts to technical constraints. Keeping them separate allows precise dirty-flagging.

5. **Tests verify the convergence point, not individual source layers.** If the convergence layer (ARCH) properly reflects all source layers (REQ, DES), tests need only one parent. Gaps in ARCH are caught by consistency checks before tests are written.

6. **The diamond pattern recurs.** Whenever two concerns independently derive from a common ancestor and must later converge, the diamond applies. This can happen at any scale — project-level (REQ/DES from UC) or component-level (API design/data model from component requirements).
