# Question-anchored distillation — align what recall crystallizes with what was investigated

**Ask (Joe, 2026-07-01):** recall clusters the matched set around the *semantic centroid of existing notes*,
then distills a note pitched at the cluster's topic — but the valuable thing to build on later is the note that
answers the **question** the agent was investigating (the query phrases). Re-evaluate how we cluster / evaluate /
distill so what's distilled aligns with what was being investigated. **Scope (Joe):** design + a **cheap
skill-level prototype** + a **delivery eval**; only invest in binary re-clustering if the eval moves delivery.

## Verified reality (the misalignment is real)

- **Clustering is 100% content-centroid.** `clusterMatchedSet` (query.go) runs k-means over the matched items'
  *content* vectors; the query-phrase vectors are used only as a PRNG seed (`seedFromQuery`) and to rank
  candidate notes — **never as cluster centroids or objective**. Clusters form around what existing items share.
- **The questions are discarded at the union.** `mergePhraseIntoUnion` dedups to the max-scoring copy and keeps
  **no phrase field** on `matchedSetItem`; the payload echoes `phrases[]` but every cluster's `Phrase` is empty.
  So at crystallization the agent cannot map a cluster back to a question. **(Recoverable pre-union — the phrase
  hits exist per-phrase before the union collapses them.)**
- **Crystallization is strictly per-cluster, pitched at the cluster's topic.** recall Step 2.5: "one note per
  cluster… covering the cluster's principle"; the agent has members + candidate_l2s, **not** the phrases; the
  `--situation` field is a bare `"..."` placeholder with zero question-shaping guidance.

## Why this is a NEW lever (not a re-litigation)

The prior deflations scoped two *different* levers (see `2026-06-28-question-shaped-crystallization-proposals.md`,
`crystallization-audit.md`, note 72):
- **Handle *wording* rule** → SETTLED-REJECTED (fresh agents already write good handles; surfacing recall@5 0.99).
- **Graph-expanded retrieval** → SETTLED-REJECTED (built + reverted; 0 marginal value; 10-phrase recall already
  surfaces the bridge; "the bottleneck is *synthesis*, not retrieval").

**Question-anchored *clustering/distillation* — grouping and pitching by the question the evidence answers,
rather than the content centroid — was never proposed. It is un-measured/open** (confirmed in the ledger). Note 68
supports the depth: k-means-on-cosine natively does aggregation-by-similarity and is blind to grouping by *what
question the evidence answers* — a different axis.

### Why the single-line patch fails (the original proposal)

The rejected first instinct — add one line to recall Step 2.5 telling the agent to pitch the situation at the
question — is **inert, not just shallow**: the payload carries **no phrase→member association** (stripped at the
union), so the agent has no data to anchor to, whatever the skill says. It also sits against Step 2.5's explicit
**one-note-per-cluster** rule and the covered/near/absent coverage judgment, which never reference the questions.
And pointing the agent at the `learn` skill's phrasing (as the line implied) is doubly inert — the `learn` path has
no retrieved-phrase context to borrow. **The plumbing that exposes the phrase→member association is the
load-bearing prerequisite; the skill instruction is worthless without it** — which is why the mechanism below is
data-flow-first.

## The challenge this design must honor (base rate)

Every crystallization-**quality** lever to date deflated to **Δ≈0 on delivery** (handle-wording; graph-retrieval;
the emergent-synthesis step 18/18=18/18; synthesis-note persistence Δ=0 shallow). Note 99: memory's only verified
value is idiosyncratic **capability**; crystallization quality has never moved a real axis. Note 119: this class
of change is warranted **only** by a 3-condition blind knowledge-**delivery** test, never by architectural
rightness. **But unlike those deflated levers, this one has a *measured prior gap*** — note 120: cluster-driven
notes 40% question-useful vs correction-driven 79% — so there is a documented delta to close, which is why it earns
a cheap test rather than outright dismissal. **Therefore the eval is the load-bearing deliverable, and the
mechanism is a cheap prototype to feed it — not a substrate rebuild on faith.**

## The mechanism (cheap prototype — skill-level, no k-means change)

Two small changes produce **question-anchored notes** without re-architecting clustering:

1. **Binary: carry the phrase→item association through to the payload** (modest plumbing).
   - `matchedSetItem` gains `phrases []int` (indices into the phrase list). `mergePhraseIntoUnion` **accumulates**
     the union of matching phrase indices per item instead of discarding all but the max-score copy.
   - Thread it through the FULL path (verified against query.go — the naive path silently drops it at
     `splitMatchedSet`, query.go:1622, which splits items into notes/chunks). **FOUR structs need the field, not
     two:** `matchedSetItem` → (via `splitMatchedSet`) `scoredCandidate` (notes) + `scoredChunk` (chunks) → (via
     `buildMatchedSet` for notes, `addMatchedChunksToMatchedSet` for chunks) → `matchedMember` → payload.
   - Expose it: each cluster member lists the phrase indices that retrieved it. Additive `[]int` with `omitempty`
     on the payload structs (queryClusterMember / queryItem) — no schema break.

2. **Skill: distill per question-group, not per content-cluster** (recall Step 2.5, prototype behind the existing
   content-cluster coverage loop).
   - The coverage loop (covered/near/absent judgment) stays on the content-clusters — it works (surfacing 0.99).
   - The **write** changes: for an `absent` verdict, the agent groups that cluster's members by the **question(s)**
     that retrieved them (now visible), and pitches the crystallized note's `--situation` at the *question the
     evidence answers* (the way the `learn` path phrases a correction), distilling from the question-aligned
     subset — not the whole content-cluster's topic.

### The cardinality rule — both directions (the hard part Joe flagged)

Two mismatches between content-clusters and questions, both resolved by making the distillation unit the
**question-intent, spanning content-clusters** (not per-cluster):
- **More phrases than intents** (correlated phrases): 10 phrases from one situation retrieve overlapping items.
  **Collapse** phrase-groups whose member-sets overlap at **Jaccard ≥ 0.75**, or whose phrases embed at **cosine
  ≥ 0.85** (phrase-level embeddings), into one distinct intent. (Starting thresholds; tune in the prototype.)
- **Same intent across clusters** (the converse, flagged in Gate A): a single question's evidence often lands in
  **multiple** content-clusters. So the grouping runs **across all `absent`-verdict clusters' members**, not
  within one cluster — members sharing a retrieving-phrase (post-collapse) form ONE distillation unit and yield
  ONE note, even if they came from two clusters. This prevents the per-cluster loop's fragmentation (two notes
  about the same question-intent).
- Net: distillation units = distinct question-intents (cross-cluster), each anchored to a question rather than a
  content topic. This IS "clustering by question," done in the agent at prototype cost. The coverage judgment
  (covered/near/absent) stays per-content-cluster (it works); only the write side re-groups by intent.
- **LLM-workload guard (Joe's concern):** the re-grouping is over the small set of `absent`-cluster members; if it
  adds material burden or confusion in the RED baseline (measured — does the agent stumble on the grouping?), that
  is itself a finding against the lever.

## The delivery eval (the gate — this is what decides everything)

Per note 119, adapted to crystallization quality. The claim under test: **a question-anchored note is retrieved +
applied better on a FUTURE related question than a topic-anchored one.**

- **Corpus (N ≥ 10):** each entry is a PAIR — {initial query phrases + retrieved evidence} → {a later question,
  from a *different* investigation, that applies the *same principle to a new domain*}. Draw from the mined
  failure/correction corpus + trap fixtures. (N is a starting floor; report the power bound.)
- **Arms:** crystallize each investigation BOTH ways — (A) current per-content-cluster topic-anchored note; (B)
  prototype per-question-intent question-anchored note. Same evidence, same model; differ only in the distillation.
- **Verdict (blind-judged, per note 119):** on the later question, run recall and judge with an **opus LLM scorer**
  — does the agent's plan **apply** the lesson's principle *unprompted*, tracking the reasoning not the note name?
  3 conditions: none / +topic-note / +question-note. Metric = knowledge-**delivery** (retrieved AND applied), NOT
  "% question-shaped" (the deflated proxy); the scorer detects the *pattern*, never the note name (bias guard).
- **Pass-bar:** run each arm on the same evidence subset ≥2× to size per-arm variance; take the larger σ; **B must
  beat A by ≥ 2σ (95%)**. A tie below 2σ = "can't distinguish," park it (like the prior levers). Also gate on a
  C3–C6 trap regression (no capability loss) + the `recall_cost` $METER (question-grouping must not blow the
  procedure cost). **RED signal:** if delivery-with-notes == delivery-with-no-notes, the grouping adds no leverage
  → park.

## Results (2026-07-01) — PARK (no delivery benefit; clear retrieval loss)

Ran as designed: 10 idiosyncratic pairs, opus model + blind name-agnostic opus judge, harness
`dev/eval/traps/qanchor_{corpus,eval,score,retrieval_probe}.py` (+ `test_qanchor.py`). Total spend ≈ $30.

**Headroom real (not ceiling-limited):** none-condition (cold, no note) delivered the principle only
**25%** (10/40) — 73% of cases fail cold — so an A≈B null here is a genuine finding, not underpowered.

**Application channel (note injected, opus applies?):**

| Condition | Delivery rate (applied) | ±1σ | Δ vs cold |
|---|---|---|---|
| none (cold) | 25% (10/40) | 6.8 pp | — |
| A — topic-anchored (current) | **62%** (25/40) | 7.7 pp | +37 pp |
| B — question-anchored (prototype) | 52% (21/40) | 7.9 pp | +27 pp |

B − A = **−10 pp**, inside the 2σ floor (±22 pp) → the pre-registered "B beats A by ≥2σ" bar is **not met**.

**Retrieval channel (FREE probe — cosine of the concrete future question to each note):** topic-anchored
out-scored question-anchored in **10/10 pairs** (mean 0.52 vs 0.35; one B note abstracted below the 0.25
relevance floor entirely). So the retrieval channel *inverts* the design's premise — it favors topic-anchoring.

**Unified mechanism:** the **concrete idiosyncratic token is load-bearing on both channels.** Keeping it
(topic-anchoring) embeds the note nearer a concrete future question (real future questions name the system,
not the abstract principle) AND lets a downstream agent confidently apply it to the named system.
Question-abstraction strips the token and loses both ways. Anchoring interacts with lesson TYPE — abstraction
helped only the 3 transferable-*pattern* pairs (B 83% vs A 58%), lost the 7 concrete-*API* pairs (A 64% vs
B 39%) — but nets to no win. This is note 119's "proxy moves, outcome doesn't": the note-120 40-vs-79 wording
gap is real but delivery-inert.

## Decision after the eval — PARK with evidence

- **B ≈ A on delivery AND loses retrieval 10/10 → PARK.** Do not build binary-level question-clustering; do
  not add a question-shaping skill rule. The phrase-provenance plumbing prototype is **reverted** (backed up:
  `docs/design/artifacts/2026-07-01-phrase-provenance-plumbing.patch`). The verified value stays idiosyncratic
  capability (both notes deliver +27–37 pp of it over cold); we stop polishing distillation anchoring.
- **Note 120 amended, not discarded:** its wording-audit finding stands; its *prescription* ("re-anchor the
  handle to the question") is refuted on delivery+retrieval and is scoped down accordingly.
- **Honest bound:** both crystallizer and delivery agent were opus (strong distiller → abstracts topic notes
  to the principle at read time), and the fictional tokens (needed for headroom) amplify the API-call retrieval
  deficit. The safe claim is "question-anchoring does not beat topic-anchoring," not "it is strictly worse."

## Explicitly out of scope / do-not-rebuild
- The handle-**wording** prose rule (settled-rejected — baseline passes).
- Graph-expanded retrieval (settled-rejected — 0 marginal value).
- Binary-level re-clustering — **deferred behind the eval verdict** (not this round).

## Spec self-review
- **Placeholders:** none — the plumbing, the skill change, the collapse rule, and the eval arms/verdict/pass-bar
  are concrete.
- **Consistency:** the coverage loop stays on content-clusters (works); only the write side becomes
  question-anchored; the eval isolates that one change (A vs B, same evidence).
- **Scope:** prototype + eval this round; binary re-clustering gated behind a delivery move; settled-rejected
  levers named.
- **Honest bound:** the base rate says this probably deflates too; the eval is designed to catch that cheaply
  rather than ship on faith.
