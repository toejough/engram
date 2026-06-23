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

1. **Is "synthesize-across-groups" the missing step?** It is a named step in the literature
   (RAPTOR / GraphRAG global search / classic extractive merge) that engram lacks — **but the
   evidence says do NOT build it.** It is GraphRAG *global search*, which independent reference-based
   evaluation found *underperforms* naive RAG on detail-centric retrieval (Han et al.). It also only
   reshapes the OUTPUT (one answer vs N cluster blobs) — it cannot surface a bridge note that was
   never retrieved, which is the actual gap. And engram's LLM-agent consumer re-synthesizes across
   cluster summaries at use-time anyway. **Conclusion: across-groups reduce is the named step, but
   not the beneficial one — skip it (or restrict to explicit global/summary queries behind an
   eval).**
2. **What IS the beneficial change?** **Graph-expanded retrieval** (GraphRAG *local* search /
   spreading activation): seed from the query, traverse `internal/vaultgraph` wikilinks before
   clustering, so bridge notes that cosine missed enter the set. This is a *different* mechanism from
   global search — it changes *what gets retrieved*, not the output format — and it is NOT the thing
   that underperformed. It is the only change that can fix the cake (T3) / transitive (T6) gap,
   because the gap is a RETRIEVAL miss.
3. **Weighted/relational grouping — different architecture or adapt k-means?** Part-weighting
   **adapts k-means** (cheap) but will NOT fix compositional join (a representation problem — the
   relation is gone at encode time). Bridging complementarity needs the **relational graph**, which
   is the same `internal/vaultgraph` graph the answer to (2) uses. So (2) and (3) are one change.

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
result). Its k-means clusters map most naturally onto GraphRAG's "communities," so the *named* next
step would be a **reduce** (one LLM pass synthesizing the per-cluster summaries into a single
answer). **But this is the path the evidence says NOT to take** (see the contested finding below and
§4): it only reshapes the output and can't surface an un-retrieved bridge note. The beneficial use
of the same GraphRAG lineage is its *local* search — graph-expanded retrieval (§4 Stage 1) — not its
global reduce.

**⚠ The contested part (independently verified, matters):** GraphRAG's positive global-QA result
rests *solely* on its own LLM-as-judge eval with no ground truth. Independent work (Han et al.,
arXiv 2502.11371, distinct authors) using **reference-based** metrics (ROUGE/BERTScore) found:
*"Community-based GraphRAG, particularly with global search, generally underperforms RAG"* — and
attributes the original positive result to the reference-free LLM-judge protocol. This **directly
matches engram's own recorded eval lesson** (LLM-judge win-rates flatter the structure/memory arm).
Nuance: it's *task-dependent* — global synthesis helps comparison/temporal multi-hop queries, hurts
detail-centric/single-hop QA. **Verdict: the across-groups reduce is a known template but NOT a win
for engram's detail-centric recall — do not build it (see §4); the beneficial GraphRAG mechanism is
*local* search / graph-expanded retrieval, not the global reduce.**

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

**Stage 1 — the lead change: graph-expanded retrieval (reuse the existing wikilink graph).** Before
clustering, **expand the cosine-matched seed set by traversing `internal/vaultgraph` wikilinks 1–2
hops**, then cluster/summarize the expanded set. This is GraphRAG *local* search / spreading
activation — it surfaces bridge notes cosine misses, enabling compositional join (T3) and transitive
chain (T6) *where the links exist*. It lives in recall's retrieval stage; no embedding-store change.
This is the only change that touches the actual gap (a retrieval miss). **Success criterion (gate to
Stage 2):** on a cake-style (T3) eval, graph-expanded retrieval must surface the bridge notes and
lift the cross-domain-join score materially over cosine-only — reference-based check, not LLM-judge.
If link density is too low to surface bridges, Stage 1 fails and the answer is Stage 2 (or denser
linking), not more graph tuning.

**Stage 2 — higher-ceiling, genuinely new mechanism (only if Stage 1 proves the value):** an
**iterative retrieve→reason→retrieve loop** (IRCoT-style) — let the synthesis step name the bridge
entity it now needs and re-query on it. This catches transitive hops with **no pre-existing link**
(what Stage 1 can't). It's a real architectural addition (a loop + intermediate query generation).

**NOT a stage — the across-groups reduce (GraphRAG global search):** initially considered as a cheap
"Stage 0," but **dropped**. It only reshapes the output (one answer vs N cluster blobs), cannot
surface an un-retrieved bridge note, underperforms naive RAG on detail-centric retrieval under
reference-based metrics (Han et al.), and duplicates synthesis the LLM-agent consumer already does at
use-time. Revisit ONLY for explicit global/summary recall queries ("what's my overall stance on X"),
and only behind a reference-based eval.

## 4b. Recorded decision (2026-06-23): the foundational primitive is LLM-judged cross-cluster LINKING

A circular dependency in §4 surfaced under challenge and must be recorded. Graph-expanded retrieval
(Stage 1) *reads* edges — but engram's **write path today is search → cluster → link WITHIN cluster
only**. No mechanism ever creates cross-cluster edges. The links the read side needs are exactly the
ones the write side never makes: complementary/bridge notes are cross-cluster *by construction*
(cosine split them). So graph traversal over today's vault surfaces more of the same cluster, never
the join — the read-side fix presupposes a graph the system cannot grow.

**Therefore the foundational primitive is a cross-cluster link-creation step at recall/learn time:**
the LLM — which already receives ALL clusters in the recall payload — **judiciously reasons across
clusters and PERSISTS connecting links/notes, defaulting to no link.** This is the actual content of
"synthesis": not an output reduce (dropped, §4), but the **write step that grows the graph**.
Graph-expanded retrieval (Stage 1) only becomes useful *after* this exists. The current Step 2.5
processes each cluster *independently* ("one write per cluster") — it is never asked to look across
clusters; that instruction is the change.

**Central risk = precision, not possibility.** A cross-cluster pass weighs O(clusters²) candidate
relationships; an eager LLM links everything to everything and pollutes the graph into uselessness.
The mechanism needs the same adversarial gate as C6 (persist a cross-cluster edge only on a genuine
relationship; default to none). Getting precision right is the whole game.

**Empirical check to run first (Joe's prompt):** run real `/recall` against the cake vault and
confirm it forms ZERO cross-cluster links today (predicted), then test whether a cross-cluster
instruction creates the right links *without flooding the graph*.

## 4c. Open question (flag for a research pass): which relationships should the LLM scan for?

The cross-cluster pass needs a catalog of relationship/inference types to look for. Composition /
complement / transitive were named ad hoc; they should be grounded in formal frameworks, and the set
is almost certainly larger. Candidate backbone (to verify, not yet sourced like §3):
- **Reasoning-mode axis — Peirce's trichotomy:** deduction (forward/transitive chain), induction
  (generalize instances → schema), abduction (infer a cause/mechanism that satisfies a need — the
  cake's "what *provides* the sweetness this needs" is abductive).
- **Relation-type catalog (lexical semantics / ontology / planning):** part-whole/meronymy
  (composition), is-a/hypernymy (abstraction), cause-effect, **requires-provides / means-ends** (the
  cake join; cf. STRIPS preconditions-effects), contradiction/supersession, analogy (same relation,
  different domain — Gentner).
- **Beyond the original three:** abstraction/generalization, abductive/causal inference, analogical
  transfer, contradiction-detection are all plausible cross-cluster link types worth encouraging.
Research this (short multi-source pass, same rigor as §3) before fixing the prompt's relationship menu.

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
