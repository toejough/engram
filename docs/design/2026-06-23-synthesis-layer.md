> **Update 2026-07-03:** Step 2.6 (cross-cluster linking), referenced below as shipped, was
> subsequently REMOVED in the vocab-notes build (see `2026-07-02-link-value-exploration.md` results
> + `2026-07-03-vocab-notes-build-results.md`). Historical record.

# Design — the synthesis layer (emergent composition over recalled notes)

> **Status: RED RUN 2026-06-23 → synthesis STEP NOT BUILT (redundant). But the eval IS the C6 proof.**
> The 3-arm RED (`dev/eval/traps/synth_eval.py` + `synth_fixtures.py`) measured, across join/chain/
> transfer fixtures: **cold opus 1/9, warm `/recall` 18/18.** Memory beats cold opus decisively (the C6
> result). But warm-only is at the **100% ceiling** — no headroom for a synthesis step — so per the §5
> RED rule the explicit Step 2.8 is **redundant and NOT built**: opus composes A+B→C spontaneously once
> recall surfaces A and B. Third straight "the agent already does it" negative (after slice 2). The spec
> below is retained as the design + the eval rationale; the harness is the C6 proof artifact.

## 1. Problem & the corrected premise

**The C6 target:** memory that produces an *emergent* conclusion C by composing notes A and B that no
single note states — and does so better than cold opus (which lacks A and B entirely).

**What we now know (three corrections to the original roadmap):**
1. **Aggregation ≠ emergent synthesis** (note 68). Summarizing co-topical notes is what cosine+k-means
   already do; it is not A+B→C across domains.
2. **Engram's *architecture* is blind to relational synthesis, but the *agent* is not.** Vector+k-means
   can't compose; opus can — given A and B in front of it.
3. **Retrieval is not the bottleneck** (note 72 / slice 2). The 10-phrase recall already surfaces the
   relevant notes; graph-expanded retrieval added 0 marginal value because the agent reaches them via
   cosine anyway.

**Therefore the synthesis layer is a SKILL/REASONING step, not a binary or retrieval change.** The
notes are already co-surfaced; the gap (if any) is getting the agent to *compose them into the
emergent conclusion and assert it*, rather than aggregate or stop at coverage.

**Relationship to the built Step 2.6 (cross-cluster linking).** Step 2.6 already runs agent-judged
composition over surfaced notes — but its *output is a typed EDGE* (graph structure: "A relates to B
via means-ends"), precision-gated, and it deliberately *excludes analogy* and persists nothing but
links. The synthesis layer's *output is an emergent CONCLUSION C* (new knowledge the user acts on,
optionally crystallized as a synthesis note Z) — what those related notes *imply together*. 2.6 grows
the graph; the synthesis layer reasons over it to produce conclusions. They share the generate→justify
discipline but differ in output and in what GENERATE may use (2.6 forbids analogy for edges; synthesis
allows analogy in GENERATE because it asserts conclusions, not persists links — note 69 still bars
analogy from JUSTIFY). The synthesis step does NOT re-do 2.6's linking.

## 1b. The central challenge (read before building anything)

**The synthesis step may be redundant.** Opus is strong; given A and B co-surfaced it may produce C
*spontaneously* — exactly as it reached every bridge without graph expansion (slice 2). If warm
recall-only already yields C, an explicit synthesis step adds nothing and must NOT be built. This is
the single biggest risk and the eval (§5) is designed to expose it **before** any skill edit. The
build's RED is "warm recall-only fails to compose"; if that RED doesn't hold, we stop and record the
negative, as with slice 2.

## 2. The synthesis layer (the mechanism, if the eval justifies it)

A new reasoning step in `recall` — **Step 2.8 — Synthesize across the surfaced set** — inserted
**after Step 2.7 (activation) and before Step 3 (closing synthesis)**. (The skill's steps today are
0, 0.5, 1, 2, 2.5, 2.6, 2.7, 3; there is no room at "3.5" — that would fall after the final step.) It
is cross-note composition, distinct from Step 2.5 (within-cluster coverage = aggregation), Step 2.6
(cross-cluster *edges*), and Step 3 (plan-impact framing). Three sub-steps, reusing note 69's
generate→justify discipline:

**A. GENERATE (loose, high recall, asserts nothing).** Scan notes from *different* clusters/domains for
candidate emergent conclusions of three relational kinds (the types engram's architecture can't do but
the agent can):
- **compositional join** — A needs/wants X; B provides/supplies X → "use B to satisfy A."
- **transitive chain** — A entails B, B entails C → "A entails C."
- **analogical transfer** — A's structure in domain 1 maps onto domain 2 → "the same move applies to B."
Use analogy and "what here combines with what" freely. Proposes only.

*Generation-recall caveat (multi-axis is deferred):* GENERATE works over the cosine-surfaced set, so
the *non-topical* structural pairs that compositional/analogical synthesis most needs may not be
present — multi-axis grouping / LLM-as-axis-selector (research §7, roadmap slice 3, deferred) is the
lever that raises candidate recall. The eval must therefore distinguish "the agent can't *generate* the
right pair" (a generation/recall gap, slice 3) from "the agent generates but doesn't *compose/assert*"
(the gap this layer targets) — don't read a generation miss as a synthesis-step failure.

**B. JUSTIFY (strict — default: do not assert).** For each candidate C, validate via a truth-preserving
or satisfaction mode — **deduction** (does C actually follow from A∧B?), **abduction** (is C the best
means-end / explanation linking them?), **composition** (do the parts genuinely compose — same shared
key, no equivocation?). **Analogy alone never asserts** (note 69). Emit an audit line per candidate so
the drop is observable. Default is to NOT assert. (These are validation *modes* — orthogonal to Step
2.6's relation-*type* menu. Note 2.6's `contradiction` is intentionally absent here: a contradiction
between A and B composes into no emergent C — it is a conflict to flag, which is 2.6's job, not a
synthesis.)

**C. ASSERT (and optionally persist).** State each surviving C as an explicit conclusion the user can
act on — flagged as *emergent* (no single note says it; it comes from A∧B). Optionally crystallize C as
a **synthesis note Z** with provenance links to A and B (the deferred "synthesis-note Z" roadmap item;
decide after the eval proves the reasoning adds value — do not persist a note the eval shows the agent
would produce anyway).

**Guard against aggregation (note 68):** a "synthesis" whose inputs are all one cluster / co-topical is
aggregation — not emergent. The step only fires across notes the clustering placed in *different*
groups (or that are demonstrably cross-domain).

## 3. Why this is not slice 2 in disguise

Slice 2 changed *retrieval* (what enters the payload) and the agent reached bridges without it. This
changes *reasoning over what's already in the payload* (compose vs aggregate vs stop). The two are
orthogonal: slice 2's negative ("the agent already retrieves") does not imply this negative ("the agent
already composes and asserts"). But it does mean the SAME validation rigor applies — the eval must
prove warm+synthesis beats warm-only, or this dies too.

## 4. The C6 eval — fixtures (cross-domain compositional, anti-recitation)

A fixture seeds a vault with notes A and B such that:
- **Cross-domain:** A and B are in *different* topics, so they cluster separately (verify with the
  cake-check clustering, like slice 1) — guarantees genuine composition, not aggregation (note 68).
- **Emergent C:** the task's correct answer C = compose(A, B), and **C is stated in NO note**.
- **Anti-recitation / anti-cold:** A and B are *idiosyncratic invented facts* (not in opus's training),
  so cold opus cannot produce C and the warm agent cannot recite C from a single note — it must combine.

**Worked fixture (compositional join):**
- A (deploy domain): "our promote-gate blocks any release still carrying the `canary` label."
- B (webhooks domain): "the legacy Friday webhook auto-strips the `canary` label from open releases."
- Task (framed as Friday): "my release won't promote and I don't know why — what happened?"
- Emergent C: "the Friday webhook stripped your `canary` label, so the promote-gate no longer sees it
  blocking → re-add the label or disable the webhook." Requires A∧B; neither note states it; cold opus
  cannot know A or B. Shared key = `canary` label (the join).

**One fixture per relational kind** (≥3 total), each labeled with its kind so the eval reports
per-kind, not just aggregate: **fixture-join** (compositional, the `canary` example above),
**fixture-chain** (transitive: A⇒B, B⇒C, ask for the A⇒C consequence), **fixture-transfer**
(analogical: a fix/structure in domain 1, ask whether it applies to an analogous domain-2 problem).
The metric is computed and reported per fixture.

## 5. The C6 eval — 3 arms (the value isolation)

Reusing the warm-harness pattern (`dev/eval/traps/graphexpand_warm.py`). **Note (framing):** the cold
arm cannot produce C *by fixture construction* (idiosyncratic facts), so warm−cold is rigged-large and
only validates the memory premise — it is NOT the result. **The deciding result is the SYNTHESIS Δ
(warm+synthesis − warm-only).** Run **n≥5 per arm per fixture**; report **per fixture** (heterogeneous
relational kinds can mask each other in an aggregate).

**Metric:** C6 emergent-synthesis hit rate — reference-based: does the final answer state the fixture's
specific emergent C (the exact join/chain/transfer conclusion), judged against the fixture's stated C,
not keywords.

**Results table (labeled, per fixture; this is the artifact the eval emits):**

| Metric (units) | cold | warm-only | warm+synth | Δ = warm+synth − warm-only |
|---|---|---|---|---|
| **C6 synthesis hit rate (% of n)** | ~0 (lacks A,B) | _measured_ | _measured_ | _the deciding number_ |
| **memory premise check** = warm-only − cold (pp) | — | — | — | (large by design; sanity only) |
| **noise floor (pp)** = warm-only vs warm-only spread | — | _measured first_ | — | Δ must exceed this |

**Decision rule (self-contained — same noise discipline on BOTH gates):** first run **warm-only twice**
(or split n) to size the **noise floor** = the spread between identical warm-only replicates.
- **RED to proceed:** warm-only hit rate must be **below (100% − noise floor)** — i.e. there is real
  headroom for a step to add. If warm-only already composes C at-or-above (100% − noise), **STOP — the
  step is redundant** (record the negative, as slice 2; do not edit the skill). "Low rate" alone is not
  enough — gate against the measured noise, not a guessed threshold.
- **GREEN to ship:** Δ (warm+synth − warm-only) must **exceed the noise floor**. A Δ below noise = "can't
  distinguish," not a win (the gap-below-noise lesson) → do not ship.

**Pressure tests (REFACTOR):** aggregation control (all-one-cluster vault → the step must NOT
manufacture a false "emergent" C); recitation control (C present verbatim in a note → not a synthesis
win, it's recall); cross-domain clustering check (A, B land in *separate* clusters — reuse
`cake.py`'s `classify_cross`/cluster inspection functions, not a CLI).

## 6. Deliverables of the build (separate effort)

1. `dev/eval/traps/synth_fixtures.py` — the per-kind cross-domain fixtures (join/chain/transfer) +
   the cross-domain clustering check (reuse `cake.py`'s `classify_cross` / cluster inspection
   functions — internal Python, not a CLI; add new fixture kinds `synth-join`/`synth-chain`/`synth-transfer`).
2. `dev/eval/traps/synth_eval.py` — the 3-arm warm harness + reference-based C-detection. **Harness gap
   to close:** `build_warm_cfg` copies the repo's *current* `skills/recall` wholesale, so warm-only vs
   warm+synthesis can't be A/B'd as-is (slice 2's A/B swapped *binaries* via `ENGRAM_BIN`, not skills).
   Add a `build_warm_cfg(dst, recall_skill_path=...)` parameter (or build two cfg dirs) so one arm gets
   the edited `SKILL.md` and the other the current one.
3. **Run the eval RED first (cold / warm-only, with the noise floor sized per §5).** Decision gate:
   proceed only if warm-only leaves real headroom (§5 RED rule) — else STOP and record the negative.
4. **If and only if RED holds:** add Step 2.8 to `skills/recall/SKILL.md` via `superpowers:writing-skills`
   TDD; GREEN re-run; ship only if Δ exceeds the noise floor (§5 GREEN rule).
5. **If the eval ships the step:** update vault note 68 — its body prescribes the fix "via
   graph-expanded retrieval (spreading activation / GraphRAG local search)," which slice 2 reverted (0
   value); the corrected prescription is *agent reasoning over already-surfaced notes* (this layer),
   and the diagnosis (aggregation ≠ synthesis) stands. Also add a **synthesis-layer row to the
   `cross-cluster-linking.md` §1b roadmap** (a new *reasoning* track, dependency: none on slices 2-4;
   it is the upstream reasoning that the deferred "synthesis-note Z" would persist).

## Self-review checklist
- Synthesis defined as emergent composition (A+B→C cross-domain), explicitly NOT aggregation (note 68)?
- Analogy confined to GENERATE; assertion requires a truth-preserving mode (note 69)?
- The redundancy risk (agent composes spontaneously) front-and-center, with a 3-arm eval that can kill
  the slice — warm-only-vs-warm+synthesis as the deciding number?
- Fixtures cross-domain + emergent-C-in-no-note + idiosyncratic (anti-cold, anti-recitation)?
- Build gated on RED (warm-only fails) — no skill edit unless the gap is proven, per the slice-2 lesson?
