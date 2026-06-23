# Research Plan — emergent synthesis for engram memory

> **For agentic workers:** a RESEARCH deliverable. The output is a committed case document grounded
> in external literature, multi-angle and pressure-tested against multiple sources (no single-source
> over-indexing). No engram code is changed this pass.

**Goal:** Ground engram's next synthesis capability in the literature on emergent properties,
reasoning systems, and logical progression. Answer, with cited sources:
1. What kinds of synthesis *should* a memory system aspire to (taxonomy)?
2. Is "search → group → distill-per-group → **synthesize-across-groups**" the right shape, and is
   the across-groups step the missing one?
3. Is weighted/relational grouping a fundamentally different architecture than k-means, or can
   k-means be adapted? What clustering alternatives exist and what do they buy?
4. Multi-hop / transitive / logical-progression synthesis — what's the established approach?

**Grounding fact (established empirically this session, verified against code):** a "make a cake"
query surfaced all 6 notes across two domains (requirements vs mechanisms), but the clustering split
them into the two domains, and engram's recall synthesizes within-cluster only — so cross-domain
join (sugar→sweetness→cake) is structurally foreclosed. The research must explain whether this is a
fundamental limit or an architecture choice, and what the literature says the fix is.

**Definitions (gate-A precision):**
- **engram's current substrate (the feasibility constraint):** AutoK k-means (k∈[2,7],
  silhouette-selected, k-means++ seed, cosine distance) over a bounded matched set (~300 notes+chunks)
  of MiniLM-L6 384-d embeddings; lazy synthesis runs as an LLM pass **per cluster** over that
  cluster's own members. There is NO cross-cluster pass, NO transitive/graph engine, NO re-embedding
  of arbitrary derived text at query time. "Feasible now" = runs in the per-cluster LLM pass.
  "Feasible with a bolt-on" = needs a new cross-cluster step but keeps the vector store. "Needs a new
  substrate" = needs a graph/relational store or symbolic engine.
- **independent source:** distinct authors AND venue AND a *different evidence chain* — two papers
  that both rest on the same seminal work (e.g. both cite Gentner 1983 with no other support) are
  NOT independent; one citing the other is NOT independent. Convergence via different
  experimental/theoretical paths counts.
- **vendor/hype separation:** vendor blogs (e.g. the GraphRAG product post) do NOT count as sources
  for design claims; cite peer-reviewed papers separately, and if none exist for a system, say so.

## Research angles (one subagent each, web-search + fetch, cite sources)

- **A1 — Cognitive/AI theory of synthesis:** structure-mapping theory (Gentner), conceptual
  blending (Fauconnier & Turner), analogical reasoning, "emergent properties" in the
  reasoning-systems sense (not the ML-scale sense). What *is* synthesis vs aggregation vs
  retrieval? Produce a taxonomy of synthesis types.
- **A2 — The across-groups ALGORITHMIC pattern (not a product survey):** at the *literature* level,
  how do systems combine per-group summaries into a higher synthesis? Is it tree aggregation
  (RAPTOR recursive summarization), graph/community traversal (GraphRAG community summaries), or a
  join on shared referents? Name the algorithmic template and where the across-groups step lives.
  GraphRAG/RAPTOR are starting points, but cite the *papers*, not vendor docs; the deliverable is
  "what is the principled shape of synthesize-across-groups," not "which product has it."
- **A3 — Clustering alternatives to k-means:** spectral clustering, community detection on
  graphs (Leiden/Louvain), soft/fuzzy clustering (overlapping membership), subspace/biclustering
  (cluster on *parts* of the feature vector — directly tests "weighting different parts of
  concepts"), relational clustering. Is k-means adaptable (weighted features, kernel) or is a
  graph/biclustering model fundamentally different? Trade-offs.
- **A4 — Multi-hop & transitive reasoning over knowledge:** knowledge-graph reasoning, multi-hop
  QA (HotpotQA-style), chain construction, neuro-symbolic / path-based inference. How is the
  "Joe wants cake → cake needs sugar → ∴ need sugar" transitive chain done in practice?
  *(In scope per the user's explicit naming of "logical progressions" and the transitive cake
  example this turn — A4 is the literature for that, distinct from A2's compositional join.)*

## Pressure-test protocol (non-negotiable — the user's explicit ask)

After the 4 angle reports, an **adversarial verification pass**: for each load-bearing claim, find a
SECOND independent source that confirms or contradicts it. Flag any claim resting on one source.
Distinguish established results from hype/marketing (GraphRAG vendor claims especially). Note where
the literature disagrees with itself.

## Synthesis (the case to build)

Map findings back to engram's concrete architecture (k-means over note+chunk embeddings, within-
cluster lazy synthesis). Deliver:
- A **taxonomy of synthesis types** engram could support, delivered as a **staged build-sequence**
  (the ask is "build up *to*"): what's feasible NOW (per-cluster pass) → what needs a cross-cluster
  bolt-on → what needs a new substrate — in that dependency order, not just a flat ranked list.
- A verdict on "synthesize-across-groups": is it the missing step, and what would implementing it
  concretely require (a cross-cluster pass over the matched set, a join on shared keys, ...)?
- A verdict on clustering: keep k-means + add a cross-cluster synthesis step, vs move to a graph /
  biclustering substrate. With the trade-off, not a single recommendation.
- Explicit **what we should NOT attempt** (synthesis types that need a different substrate than a
  vector store, e.g. full transitive theorem-proving).

## Self-review checklist
- All 4 angles covered with cited sources?
- Every load-bearing claim has ≥2 independent sources or is flagged single-source?
- Vendor/hype claims (GraphRAG) separated from peer-reviewed results?
- Findings mapped to engram's ACTUAL substrate, not generic advice?
- A clear taxonomy + ranked, feasibility-aware recommendations (not one-source over-index)?
