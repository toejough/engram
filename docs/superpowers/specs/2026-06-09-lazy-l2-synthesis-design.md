# Design — Lazy, compositional L2 synthesis at recall

Date: 2026-06-09 · Branch target: a fresh branch off `main` · Status: design, pending review

## 1. Context & motivation

**Today (eager L2).** During `/learn`, the agent reads a build and *eagerly* writes one
generic-actionable **L2 fact (or feedback)** per distinct convention — whether or not that
convention ever recurs or is ever queried. L3 ADRs are then synthesized from clusters of L2s via
`engram query --synthesis`, which already uses the lazy/semantic logic we want: each cluster
carries `nearest_l3 {path, cosine}`, and `/learn` §6b **updates** the nearest L3 at cosine ≥ 0.9
else **creates** a new one.

**Hypothesis.** L2 is only worth generating once it *proves relevant* — i.e. a real recall demands
it. Eager L2 generation spends time and tokens writing facts that may never be queried. If we defer
L2 creation until a recall actually surfaces matching evidence with no covering L2, we generate
fewer, only-relevant L2s → **faster and cheaper, without losing the say-once benefit**. And because
new L2s can be synthesized from clusters that include *existing* L2s, the L2 layer becomes
**recursive/compositional** — it forms "more complex thoughts and connections over time," not just
a flat pile of leaf facts.

This is **"pull the L3 update-or-create logic down to L2"** — with two deliberate refinements: a
third "do nothing" band (the growth bound), and recursion (L2s are both inputs to and outputs of
synthesis).

## 2. The mechanism (locked)

Lazy compositional L2 synthesis, triggered at **recall** (demand-driven):

1. **Match.** Recall queries `engram` with its task phrases. A note is *relevant* by
   `max(cos(query, situation_vector), cos(query, body_vector))` ≥ a match threshold. (Body finds
   notes by what they *say*; situation finds them by what they're *about*. Union of both — exactly
   like unioning multiple `--phrase` searches today.)
2. **Gather.** Take the matched **L1 episodes + L2 facts/feedback** (the relevant neighborhood).
3. **Cluster.** Run the existing AutoK clustering over the matched notes. **Each note's coordinate =
   the vector that matched it** (situation or body; if both matched, the *stronger* — higher cosine
   to the query). Both vectors live in one shared 384-dim MiniLM space, so mixed-coordinate
   clustering is well-defined; clusters group notes relevant *for the same reason*.
4. **Gate, per cluster.** Find the nearest **existing L2** to the cluster centroid by
   `max(situation, body)` cosine (the "either axis" rule, applied at the gate). Then:
   - **cosine ≥ 0.95 → no-op.** Some L2 already represents this cluster; leave it. *(Common case in
     a mature vault; this is the bound on runaway growth.)*
   - **0.80 ≤ cosine < 0.95 → update/elaborate** that L2, folding in the cluster's members.
   - **cosine < 0.80 → create** new L2(s) synthesizing the cluster. Because the cluster can contain
     existing L2s, the new L2 *connects and abstracts* them (composition). Each synthesis emits a
     **Fact and/or Feedback** (L2-tier), `--relation`-linked down to the cluster's L1s and L2s.
5. **Use.** The recalling agent applies the surfaced + freshly-minted L2s to its task immediately.

Thresholds (0.95 / 0.80 / match-min) are **tunable parameters** so the experiment can sweep them.

**Why no member-exclusion and no minimum cluster size.** The ≥0.95 no-op band *is* the dedup
mechanism: a cluster centered on a strong L2 self-matches at ≥0.95 → no-op (nothing new to say),
while a cluster spanning *distinct* L2s lands in the "empty middle," matches no single L2 ≥0.95 →
creates the connector. Demand (a real recall matched it) is the relevance proof, so no recurrence
floor is needed; the no-op band bounds growth over time.

## 3. Architecture & components

Four chunks. (1) is a prerequisite for (2)–(4). Each is independently testable.

### 3.1 Dual-vector sidecars  *(prerequisite — touches the whole vault)*

- **What:** every note's `.vec.json` sidecar carries **two** vectors — a **situation vector** and a
  **body vector** — instead of one.
- **Schema:** `embed.Sidecar` gains `SituationVector []float32` and `BodyVector []float32`
  (replacing the single `Vector`), plus a sidecar `schema_version` so old single-vector sidecars
  are detected as incompatible (same mechanism as the model-id stamp). `Dims`/`EmbeddingModelID`/
  `ContentHash` unchanged.
- **Embed pipeline:** `embed.Text()` currently returns *situation for episodes, body for others* —
  a single string. Split into `embed.SituationText()` (the `situation:` frontmatter field) and
  `embed.BodyText()` (`ExtractBody`); `autoEmbedNote` (and the migrate/resituate paths) embed
  **both** and store both vectors. Every note now has a real situation vector (episodes already did)
  and a real body vector.
- **Migration:** one-time `engram embed apply --force` (or `--all`) re-embeds the corpus into the
  two-vector form. The schema_version bump makes stale single-vector sidecars surface the existing
  "run `engram embed apply`" guidance.
- **Interface unchanged for callers** except retrieval/clustering now choose which vector(s) to use
  (below).
- **Validation gate before building on it:** confirm a re-embedded vault passes `engram check` and
  that `engram query` retrieval quality is at least as good with `max(situation, body)` as the old
  single-vector ranking (no regression on the existing recall behavior).

### 3.2 Binary — L2-synthesis query mode

- **What:** an `engram query` mode (e.g. `--synthesize-l2`, mirroring `--synthesis`) that returns
  the matched **L1+L2** neighborhood, clusters it with per-note matched-vector coordinates, and
  emits **`nearest_l2 {path, cosine}`** per cluster (alongside the existing `nearest_l3`).
- **Reuses:** the entire existing union → cluster-once → nearest-tier-cosine machinery. The deltas:
  - Retrieval ranks every note by `max(situation_cos, body_cos)` to the query; record *which* vector
    won per note (for its clustering coordinate).
  - Constrain the clustered set to matched notes with `tier ∈ {L1, L2}`.
  - Compute `nearest_l2` = best `max(situation, body)` cosine from the cluster centroid to the L2
    index (the L2 analogue of `gatherL3Index` / `nearestL3For`).
- **Output:** the existing payload shape (`items[]`, `clusters[]`, `budget`) with `nearest_l2` added
  to each cluster. **The binary emits the raw `nearest_l2.cosine` only — it applies no band
  decision** (just as today it emits `nearest_l3.cosine` and the skill applies the 0.9 cut). The
  three-band policy lives in `/recall` (3.3); the threshold *values* are experiment parameters the
  harness injects into the recall step, so a sweep needs no skill or binary edit. Defaults
  0.95 / 0.80 / match-min.
- **TDD:** Go unit tests (imptest/rapid/gomega) for the new ranking, coordinate selection, and
  `nearest_l2` computation, plus a property test that a vault with a near-duplicate L2 yields
  `nearest_l2.cosine ≥ 0.95` for the matching cluster.

### 3.3 `/recall` skill — the three-band writes

- **What:** formalize what Step 3a already does ad-hoc (dispatch synthesis subagents that write) into
  the explicit three-band rule on `nearest_l2.cosine`: ≥0.95 no-op · 0.80–0.95 update · <0.80
  create, emitting Fact **and/or** Feedback per cluster, linked down to members.
- **Key behavior change:** unlike 3a's *fire-and-forget* L3 writes (for future recalls), the L2
  synthesis is **blocking** — the recalling agent must *use* the freshly-minted L2s for its current
  task, so it waits for the writes, then proceeds. This makes recall a deliberate, sometimes-writer.
- **Reuse:** the fact-vs-feedback split, situation/source/relation conventions, and Luhmann
  placement are exactly the existing `/learn` write rules.
- **TDD:** **mandatory `superpowers:writing-skills`** (baseline RED → edit → behavioral GREEN), then
  `engram update` to sync. A pressure test: given a query payload with clusters in each band, the
  agent does nothing / updates / creates correctly and uses the result.

### 3.4 Experiment — eager-L2 vs lazy-L2

- **Arms** (reuse the cumulative-accumulation harness, `dev/eval/cumulative/`). Both must end up
  with L1+L2 available at recall — the difference is *when* the L2s are made:
  - **A · eager (today):** an L2-writing regime (e.g. `l2.l1l2` — write L2 facts at learn, read
    {L1,L2}); recall reads, no writes.
  - **B · lazy (new):** `/learn` writes **L1 episodes only**; recall runs the L2-synthesis mode and
    the three-band writes, crystallizing L2s on demand.
  - **L3 is not generated in either arm for v1** (out of scope, §6); the comparison is purely about
    eager-vs-lazy *L2*.
- **Harness change:** a `--learn-mode eager|lazy` switch that (a) swaps the learn prompt to
  episode-only for B, and (b) points the build's recall step at the `--synthesize-l2` mode + the new
  `/recall` write path. Per-cell build-vault isolation already absorbs in-loop recall writes.
- **Metrics** (the hypothesis): **#L2 notes generated** (B fewer), **learn cost/tokens** (B lower —
  L1 only), **recall cost/tokens/time** (B higher — on-demand synthesis), **net chain cost** (the
  bet), **say-once convention restatements** (must be preserved in B), **completion**, **vault
  growth**, and **L2 composition** (do B's L2s link to *other L2s*, i.e. is the recursion real).
- **Scope:** start focused — sonnet, n=5, the full notes→links→feeds chain, A vs B. Extend to
  opus/haiku only if the focused result warrants. Token I/O + cost audit (§6) as in the current
  baseline.

## 4. Data flow — one recall (lazy arm)

```
agent (building links, recalls v1 = app1's L1 episodes)
  └─ engram query --synthesize-l2 --phrase "building a Go CLI ..." ...
       binary: rank all notes by max(sit,body); keep matched L1+L2; cluster (matched-vector coords);
               per cluster → nearest_l2 {path, cosine}
       payload: items[] + clusters[]{..., nearest_l2}
  └─ /recall reads each cluster's nearest_l2.cosine:
       ≥0.95 → skip
       0.80–0.95 → engram learn (update nearest L2, fold in members)
       <0.80 → engram learn fact|feedback (new L2, --relation → cluster's L1s+L2s)   [blocking]
  └─ agent applies surfaced + new L2s, builds links
```

## 5. Decisions locked vs open

**Locked** (resolved in design dialogue):
- Lazy at recall; demand-driven trigger.
- Match by `max(situation, body)`; cluster coordinate = matched (stronger) vector; gate by
  `max(situation, body)` to centroid.
- Three bands 0.95 / 0.80 (tunable); no member-exclusion; no minimum cluster size.
- Fact **and/or** Feedback from a cluster (not facts-only).
- Dual-vector sidecars; one shared embedding space.

**Open / to validate:**
- **Does mixed L1/L2 clustering actually group same-topic notes in practice?** Resolved *in theory*
  (shared space, matched-vector coords, situation-shaped queries) — but verify empirically on the
  re-embedded vault before trusting it (gate after 3.1).
- **Threshold values.** 0.95/0.80 are guesses; the experiment sweeps them.
- **Recall latency/cost** from the blocking synthesis — measured by the experiment; it's the cost
  side of the hypothesis.
- **Runaway depth.** The no-op band bounds duplication, not abstraction depth; watch vault growth in
  the experiment and add a depth cap only if growth is pathological (YAGNI until shown).

## 6. Out of scope (v1)

- Changing **L3** synthesis (stays as-is; lazy-L2 could later feed it).
- A new clusterer (we keep centroid k-means; "max" is used for *matching/gating scores*, never as a
  clustering metric).
- Retrofitting non-eval callers of recall — the behavior change ships through `/recall` + the eval.

## 7. Sequencing & gates

1. **Dual-vector sidecars** (3.1) → `engram check` clean + retrieval-no-regression gate.
2. **Binary L2-synthesis mode** (3.2) → Go TDD green; `nearest_l2` verified on a seeded vault.
3. **`/recall` three-band writes** (3.3) → writing-skills RED→GREEN; `engram update` synced.
4. **Experiment** (3.4) → zero-cost stub validation first, then the focused sonnet n=5 A/B with the
   cost/say-once/composition metrics; aggregate into a `results` doc; `compare.py` vs the eager
   baseline.
