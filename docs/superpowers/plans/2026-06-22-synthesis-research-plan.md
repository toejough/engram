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

**Grounding fact (already established empirically this session):** a "make a cake" query surfaced
all 6 notes across two domains (requirements vs mechanisms), but k-means split them into the two
domains, and engram's recall synthesizes within-cluster only — so cross-domain join (sugar→sweetness
→cake) is structurally foreclosed. The research must explain whether this is a fundamental limit or
an architecture choice, and what the literature says the fix is.

## Research angles (one subagent each, web-search + fetch, cite sources)

- **A1 — Cognitive/AI theory of synthesis:** structure-mapping theory (Gentner), conceptual
  blending (Fauconnier & Turner), analogical reasoning, "emergent properties" in the
  reasoning-systems sense (not the ML-scale sense). What *is* synthesis vs aggregation vs
  retrieval? Produce a taxonomy of synthesis types.
- **A2 — The across-groups pattern in retrieval systems:** GraphRAG (Microsoft), hierarchical /
  community-summary RAG, multi-document summarization, RAPTOR (recursive tree summarization),
  "query-focused summarization." Does the field do distill-per-group then synthesize-across-groups?
  What is the across-groups step actually called and how is it implemented?
- **A3 — Clustering alternatives to k-means:** spectral clustering, community detection on
  graphs (Leiden/Louvain), soft/fuzzy clustering (overlapping membership), subspace/biclustering
  (cluster on *parts* of the feature vector — directly tests "weighting different parts of
  concepts"), relational clustering. Is k-means adaptable (weighted features, kernel) or is a
  graph/biclustering model fundamentally different? Trade-offs.
- **A4 — Multi-hop & transitive reasoning over knowledge:** knowledge-graph reasoning, multi-hop
  QA (HotpotQA-style), chain construction, neuro-symbolic / path-based inference. How is the
  "Joe wants cake → cake needs sugar → ∴ need sugar" transitive chain done in practice?

## Pressure-test protocol (non-negotiable — the user's explicit ask)

After the 4 angle reports, an **adversarial verification pass**: for each load-bearing claim, find a
SECOND independent source that confirms or contradicts it. Flag any claim resting on one source.
Distinguish established results from hype/marketing (GraphRAG vendor claims especially). Note where
the literature disagrees with itself.

## Synthesis (the case to build)

Map findings back to engram's concrete architecture (k-means over note+chunk embeddings, within-
cluster lazy synthesis). Deliver:
- A **taxonomy of synthesis types** engram could support, ranked by (value × feasibility on the
  current vector+cluster substrate).
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
