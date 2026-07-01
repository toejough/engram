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

## The challenge this design must honor (base rate)

Every crystallization-**quality** lever to date deflated to **Δ≈0 on delivery** (handle-wording; graph-retrieval;
the emergent-synthesis step 18/18=18/18; synthesis-note persistence Δ=0 shallow). Note 99: memory's only verified
value is idiosyncratic **capability**; crystallization quality has never moved a real axis. Note 119: this class
of change is warranted **only** by a 3-condition blind knowledge-**delivery** test, never by architectural
rightness. **Therefore the eval is the load-bearing deliverable, and the mechanism is a cheap prototype to feed
it — not a substrate rebuild on faith.**

## The mechanism (cheap prototype — skill-level, no k-means change)

Two small changes produce **question-anchored notes** without re-architecting clustering:

1. **Binary: carry the phrase→item association through to the payload** (modest plumbing).
   - `matchedSetItem` gains `phrases []int` (indices into the phrase list). `mergePhraseIntoUnion` **accumulates**
     the set of matching phrase indices per item instead of discarding all but the max-score copy.
   - Thread through `applyFloorAndCap` → `buildMatchedSet` → `matchedMember` → cluster members → payload.
   - Expose it: each cluster member (and/or each item) lists the phrase indices that retrieved it. Additive
     field, `omitempty`, no schema break.

2. **Skill: distill per question-group, not per content-cluster** (recall Step 2.5, prototype behind the existing
   content-cluster coverage loop).
   - The coverage loop (covered/near/absent judgment) stays on the content-clusters — it works (surfacing 0.99).
   - The **write** changes: for an `absent` verdict, the agent groups that cluster's members by the **question(s)**
     that retrieved them (now visible), and pitches the crystallized note's `--situation` at the *question the
     evidence answers* (the way the `learn` path phrases a correction), distilling from the question-aligned
     subset — not the whole content-cluster's topic.

### The cardinality rule (the hard part Joe flagged)

Content-clusters ≈ AutoK-many; the 10 phrases are **correlated** (one situation → overlapping retrieval), so naive
per-phrase distillation would mint redundant notes. The collapse rule (skill-level, agent-judged):
- **Group** a cluster's members by shared retrieving-phrase(s).
- **Collapse** question-groups whose member-sets overlap heavily (high Jaccard) or whose phrases are near-synonyms
  → one distillation unit per *distinct question-intent*, not per phrase.
- Net: roughly the same number of distillation units as content-clusters, but each anchored to a **question
  intent** rather than a content topic. This IS "clustering by question," done in the agent at prototype cost.
- **LLM-workload guard (Joe's concern):** this must not make the agent work materially harder. The grouping is
  over a cluster's already-small member set; if it adds meaningful burden or confusion in the RED baseline, that
  is itself a finding against the lever.

## The delivery eval (the gate — this is what decides everything)

Per note 119, adapted to crystallization quality. The claim under test: **a question-anchored note is retrieved +
applied better on a FUTURE related question than a topic-anchored one.**

- **Corpus:** N past investigations, each = {the query phrases (questions), the retrieved evidence, a later
  related question that should reuse the lesson}. Draw from the mined failure/correction corpus + trap fixtures.
- **Arms:** crystallize each investigation BOTH ways — (A) current per-content-cluster topic-anchored note; (B)
  prototype per-question-group question-anchored note. Same evidence, same model; differ only in the distillation.
- **Verdict (blind-judged, per note 119):** on the later question, run recall and judge — does the agent's plan
  **apply** the lesson? 3 conditions: none / +topic-note / +question-note. Metric = knowledge-**delivery**
  (retrieved AND applied), NOT "% question-shaped" (the deflated proxy). Detect the *pattern* applied, not the
  note name (scorer-bias guard, note-scorer-vocabulary-bias).
- **Pass-bar:** B must beat A on delivery **above the noise floor** (sized from a same-arm contrast). A tie below
  noise = "can't distinguish," park it (like the prior levers). Gate on a C3–C6 trap regression (no capability
  loss) + the `recall_cost` $METER (question-grouping must not blow the procedure cost).

## Decision after the eval

- **B moves delivery above noise** → invest in the **binary-level question-clustering** (the full form: cluster/
  distill around question intent in `engram query`, not just skill-level grouping). Spec that as a follow-on.
- **B ≈ A (below noise)** → **park with evidence**, alongside the other deflated crystallization-quality levers.
  The verified value stays idiosyncratic capability; we stop polishing distillation.

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
