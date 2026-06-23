# What synthesis engram should build toward — a literature-grounded case

**Question (Joe):** what kinds of synthesis *should* engram support? Is "search → group →
distill-per-group → **synthesize-across-groups**" the missing step? Is weighted/relational grouping
a different architecture than k-means, or can k-means adapt? Researched across 4 angles, every
load-bearing claim cross-source pressure-tested (10/10 citations verified to exist; the one
consequential contested claim independently re-confirmed).

**Grounding fact (verified vs code this session):** a "make a cake" query surfaced all 6 notes
across two domains (cake-needs-{sweetness,texture,fluffiness} and {sugar,flour,soda}-provides-…),
but AutoK k-means split them into the two domains, and recall synthesizes **within-cluster only**
(`internal/cli/query.go` within-cluster `candidate_l2s`; SKILL.md Step 2.5 "one write per cluster";
"cross-cluster … not handled"). So the cross-domain join (sugar→sweetness→cake) is structurally
foreclosed. Joe's instinct was right.

## The three answers up front

1. **Is "synthesize-across-groups" the missing step?** **Yes** — it's a named, implemented step in
   the literature (RAPTOR / GraphRAG / classic extractive merge) that engram lacks. Add it (**Stage
   0**) — but its benefit is *contested* under honest metrics, so gate it behind a reference-based
   eval, never LLM-as-judge.
2. **Weighted/relational grouping — different architecture or adapt k-means?** Part-weighting
   **adapts k-means** (cheap), **but it will NOT fix compositional join** — that's a representation
   problem. Bridging complementarity needs a **relational graph**, which engram already has and
   ignores (the `internal/vaultgraph` wikilink graph) — **Stage 1**.
3. **What synthesis to build toward?** The genuinely emergent types (**T3 compositional join, T6
   transitive chain**) — reached by **graph expansion of the retrieval seed, not better vectors**.
   Staged sequence in §4.

Below is the literature behind each.

---

## 1. A taxonomy of synthesis (what "synthesis" even means)

From cognitive science / AI theory (Gentner structure-mapping; Fauconnier & Turner conceptual
blending; systems-sense "emergence" per Anderson / Krakauer — explicitly **not** the LLM-scale
"emergent abilities" sense, which independent work (Schaeffer 2023; Krakauer 2025) shows is largely
a metric artifact):

| type | definition | emergent? (C ∉ A,B) | needs… | engram does it? |
|---|---|---|---|---|
| **T1 Aggregation** | collect co-topical items, no new relations | no | similarity | ✅ natively |
| **T2 Abstraction** | extract the shared schema of several instances | weakly | similarity | ✅ natively |
| **T3 Compositional join** | combine parts via their *interface* into a novel whole | partly | **complementarity** | ❌ |
| **T4 Analogical transfer** | map relational structure base→target, project inferences | **yes** | **relational** | ❌ |
| **T5 Conceptual blend** | two inputs → blend with structure in neither | **yes (strongest)** | **complementary** | ❌ |
| **T6 Transitive chaining** | A→B, B→C ⊢ A→C | **yes** | **relational (edge composition)** | ❌ |

**The load-bearing line:** a vector + cosine + k-means + per-cluster-summary pipeline natively
performs **T1/T2 (similarity-driven)** and is **structurally blind to T3–T6**, because those depend
on *relational/complementary* structure that cosine similarity discards. Embeddings cluster *what is
alike*; real synthesis needs *what fits together or maps across*.

**Direct consequence for our eval:** the existing C6 "synthesis" fixtures (validate/dedup/check-
format, all "…on URL import") are **T1 aggregation of co-topical facets** — exactly what the
architecture already does. They do *not* test emergent synthesis. The cake structure (T3) does, and
the architecture can't do it. That explains the live C6 result (opus crystallized only 1/5, mostly
link-enriched) — it was the right call on a non-emergent cluster.

## 2. Two distinct synthesis problems (don't conflate them)

Joe's two examples are **different mechanisms** with **different fixes**:

**(A) Compositional join (the cake):** "cake needs sweetness" ⋈ "sugar provides sweetness" on the
shared property. The two notes share a *word* but their embeddings are dominated by framing (*needs*
vs *provides*) and domain (dessert vs ingredient), so they cluster apart.

**(B) Transitive chain (Joe wants cake → cake needs sugar → ∴ need sugar):** edge composition where
the endpoints (Joe's-want, sugar) are embedding-*distant*; the bridge note is dissimilar to the
query.

## 3. Findings per angle (pressure-tested)

### A2 — "synthesize-across-groups" IS a named step, in three shapes — but its benefit is CONTESTED

| shape | where the combine lives | cost | source |
|---|---|---|---|
| **Tree aggregation** (RAPTOR) | recurse: re-embed+re-cluster summaries; combine deferred to query-time **concatenation** — no cross-cluster combine call | cheap recursion | ICLR 2024 (peer-reviewed), arXiv 2401.18059 |
| **Map-reduce reduce** (GraphRAG) | per-group map → self-score → **ranked LLM reduce** into one synthesis | +1 LLM call/query | arXiv 2404.16130 (**preprint, not peer-reviewed**) |
| **Cross-cluster extractive merge** (classic QFS/MDS) | rank sentences across clusters + redundancy cutoff (MMR) | no LLM call | SIGIR/AAAI (peer-reviewed) |

**Engram maps onto RAPTOR as one layer with the recursion turned off** (our analysis, not a cited
result). Its k-means clusters map most naturally onto GraphRAG's "communities" — so the concrete
missing step is a **reduce**: take the per-cluster summaries, optionally self-score relevance,
**filter out the low-relevance clusters**, run **one LLM pass that synthesizes a single cross-cluster
answer** instead of returning N independent cluster blobs.

**⚠ The contested part (independently verified, matters):** GraphRAG's positive global-QA result
rests *solely* on its own LLM-as-judge eval with no ground truth. Independent work (Han et al.,
arXiv 2502.11371, distinct authors) using **reference-based** metrics (ROUGE/BERTScore) found:
*"Community-based GraphRAG, particularly with global search, generally underperforms RAG"* — and
attributes the original positive result to the reference-free LLM-judge protocol. This **directly
matches engram's own recorded eval lesson** (LLM-judge win-rates flatter the structure/memory arm).
Nuance: it's *task-dependent* — global synthesis helps comparison/temporal multi-hop queries, hurts
detail-centric/single-hop QA. **Verdict: the across-groups reduce is a known template worth adding
*behind a reference-based eval*, NOT a settled win.**

### A3 — part-weighting adapts k-means cheaply, but it will NOT bridge complementarity

- **"Weight different parts of a concept" = adapt k-means, not a new architecture.** Feature-
  weighted k-means and **soft subspace clustering** give each cluster its own weighted slice of the
  384 dims; **biclustering/co-clustering** is the most direct ("cluster notes over a *subset* of
  dims"). All operate on the same 300×384 matrix — a changed clustering call, no new store.
- **But none reliably puts "cake-needs-sweetness" with "sugar-provides-sweetness".** That's a
  **representation problem, not a clustering-objective problem**: MiniLM encodes the whole sentence;
  the shared "sweetness" signal isn't an isolable dimension, and the divergent framing dominates.
  Reweighting dims of a representation that already discarded the relation can't recover it.
- **The only thing that bridges complementarity is a relational graph where the edge encodes the
  relation** — then community detection (Leiden) over that graph. This is a *different architecture*
  (graph substrate), but **engram already has the substrate: the `internal/vaultgraph` wikilink
  graph**, which recall ignores for synthesis.

### A4 — transitive chaining: the bottleneck is RETRIEVAL, not reasoning

- Multi-source consensus (IRCoT ACL 2023; PRISM 2025; survey 2601.00536): LLMs chain A→B→C reliably
  *once the facts are in context*. The hard part is surfacing the **bridge fact B**, which is
  dissimilar to the query — so **one-shot cosine retrieval structurally misses it**, and clustering
  can't manufacture a hop retrieval never delivered.
- **Not a vector-store tuning change** (better embeddings / higher k / rerank won't help — B is
  genuinely dissimilar). The proven fixes: **iterative retrieve→reason→retrieve** (IRCoT) or
  **graph traversal** of existing edges. Again → engram's wikilink graph.

### Convergence (verified sound, with named precedent)

A3 and A4 independently land on the same unused asset: **engram's wikilink graph is the substrate
for both complementary-join and transitive-chain, and recall doesn't traverse it.** The mechanism —
*seed by cosine, expand by graph traversal before clustering* — is established: **spreading
activation** (SA-RAG), **GraphRAG local search**, graph-expanded seed retrieval. Caveat: those
precedents are over entity/KG graphs; engram's wikilinks are note-level, so the payoff depends on
**link density/quality** (sparse links → few bridges surfaced).

## 4. The staged build-sequence (what to build, in order)

**Stage 0 — now, no architecture change (cheap, eval-gated):** add the **across-groups reduce** to
recall — one LLM pass over the per-cluster summaries producing a single integrated answer (start
with the **extractive merge**, the peer-reviewed cheap option; the generative reduce is the
higher-ceiling, higher-risk variant). This closes the "N blobs instead of one answer" gap (T1/T2→
better T2). **Gate it behind a reference-based eval, not LLM-judge** — the literature *and* our own
lessons say LLM-judge would falsely bless it.

**Stage 1 — bolt-on, reuse the existing wikilink graph (the real unlock for T3/T6):** before
clustering, **expand the cosine-matched seed set by traversing `internal/vaultgraph` wikilinks 1–2
hops**, then cluster/summarize the expanded set. The smallest change that surfaces bridge notes
cosine misses — enabling compositional join (T3) and transitive chain (T6) *where the links exist*.
Lives entirely in recall's retrieval stage; no embedding-store change. Precedent: spreading
activation / GraphRAG local search. **Stage-1 success criterion (gate to Stage 2):** on a cake-style
(T3) eval, graph-expanded retrieval must surface the bridge notes and lift the cross-domain-join
score materially over cosine-only — measured on a reference-based check, not LLM-judge. If link
density is too low to surface bridges, Stage 1 fails and the answer is Stage 2 (or denser linking),
not more graph tuning.

**Stage 2 — higher-ceiling, genuinely new mechanism (only if Stage 1 proves the value):** an
**iterative retrieve→reason→retrieve loop** (IRCoT-style) — let the synthesis step name the bridge
entity it now needs and re-query on it. This catches transitive hops with **no pre-existing link**
(what Stage 1 can't). It's a real architectural addition (a loop + intermediate query generation).

## 5. What NOT to attempt

- **Don't try to fix complementarity by reweighting/biclustering MiniLM dimensions** — it's a
  representation problem; the relation was discarded at encode time. (A3, verified.)
- **Don't chase T4/T5 (analogical transfer / conceptual blend) on the current substrate** — those
  need explicit relational/structure-mapping machinery (SME-style) or a symbolic engine; a vector
  store can't host them. Park them.
- **Don't ship any across-groups synthesis behind an LLM-as-judge eval** — independently shown to
  flatter the structure arm (Han et al.) and it repeats our own recorded mistake.
- **Don't treat the existing C6 fixtures as a synthesis test** — they're T1 aggregation; rebuild
  them on the cake (T3) structure if we want to measure real emergence.

## 6. Sources & pressure-test record

All 10 load-bearing arXiv IDs verified to resolve (no hallucinations; the 2026-dated suspects —
2601.00536, 2603.27958 — are real). Source-strength flags:
- **Strong / multi-independent:** retrieval-is-the-multihop-bottleneck (IRCoT + PRISM + survey);
  kernel-k-means ≡ spectral/normalized-cut; emergence systems-vs-LLM distinction (Krakauer +
  Schaeffer).
- **Contested / single-origin-positive:** GraphRAG global-QA benefit (Microsoft preprint, LLM-judge)
  **vs** independent reference-based contradiction (Han et al. 2502.11371) — treat as *contested,
  task-dependent*.
- **Single-team (treat cautiously):** conceptual blending core mechanism (Fauconnier & Turner);
  structure-mapping (Gentner/SME, same lab — phenomenon corroborated independently by Hofstadter &
  Mitchell's Copycat).
- **Title-attribution flag:** arXiv 2506.02404 is *GraphRAG-Bench*, not "When to use Graphs in RAG"
  (that's 2506.05690) — both real.
- **Author analysis, not cited findings:** the engram-substrate mappings and the staged sequence are
  our synthesis; the literature establishes the mechanisms, not the engram-specific recommendations.

Key papers: RAPTOR (2401.18059, ICLR'24) · GraphRAG (2404.16130) · RAG-vs-GraphRAG (2502.11371) ·
IRCoT (2212.10509, ACL'23) · feature-weighting k-means survey (1601.03483) · soft-subspace survey
(1409.5616) · Dhillon kernel-kmeans≡spectral (KDD'04) · Gentner structure-mapping (1983) ·
Fauconnier & Turner conceptual blending · Krakauer/Mitchell emergence (2025) · SA-RAG spreading
activation.
