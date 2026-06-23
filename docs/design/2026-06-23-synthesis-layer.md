# Design — the synthesis layer (emergent composition over recalled notes)

> **Status: design spec (spec-only).** The build edits `skills/recall/SKILL.md` (a synthesis reasoning
> step) and is gated on an eval that may KILL it before any skill change — see §5. Grounded in vault
> notes 68 (aggregation ≠ synthesis), 69 (analogy generates, doesn't prove), 72 (multi-phrase recall
> subsumes graph expansion), and the slice-2 negative (`EXPERIMENT-LOG` 2026-06-23).

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

## 1b. The central challenge (read before building anything)

**The synthesis step may be redundant.** Opus is strong; given A and B co-surfaced it may produce C
*spontaneously* — exactly as it reached every bridge without graph expansion (slice 2). If warm
recall-only already yields C, an explicit synthesis step adds nothing and must NOT be built. This is
the single biggest risk and the eval (§5) is designed to expose it **before** any skill edit. The
build's RED is "warm recall-only fails to compose"; if that RED doesn't hold, we stop and record the
negative, as with slice 2.

## 2. The synthesis layer (the mechanism, if the eval justifies it)

A new reasoning step in `recall` — call it **Step 3.5 — Synthesize across the surfaced set** — that
runs after the clustered notes are in context (after Step 2.5 coverage, before/with Step 3
plan-impact). It is cross-note composition, distinct from Step 2.5 (within-cluster coverage =
aggregation) and Step 3 (plan-impact framing). Three sub-steps, reusing note 69's generate→justify
discipline:

**A. GENERATE (loose, high recall, asserts nothing).** Scan notes from *different* clusters/domains for
candidate emergent conclusions of three relational kinds (the types engram's architecture can't do but
the agent can):
- **compositional join** — A needs/wants X; B provides/supplies X → "use B to satisfy A."
- **transitive chain** — A entails B, B entails C → "A entails C."
- **analogical transfer** — A's structure in domain 1 maps onto domain 2 → "the same move applies to B."
Use analogy and "what here combines with what" freely. Proposes only.

**B. JUSTIFY (strict — default: do not assert).** For each candidate C, validate via a truth-preserving
or satisfaction mode — **deduction** (does C actually follow from A∧B?), **abduction** (is C the best
means-end / explanation linking them?), **composition** (do the parts genuinely compose — same shared
key, no equivocation?). **Analogy alone never asserts** (note 69). Emit an audit line per candidate so
the drop is observable. Default is to NOT assert.

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

(Build ≥3 such fixtures spanning compositional-join, transitive-chain, analogical-transfer.)

## 5. The C6 eval — 3 arms (the value isolation) and the build's RED

Reusing the warm-harness pattern (`dev/eval/traps/graphexpand_warm.py`):

| arm | setup | measures |
|---|---|---|
| **cold** | opus, no memory, no skills | baseline: cannot produce C (lacks A, B) → expect 0 |
| **warm recall-only** | warm `/recall` (current skill), seeded vault | does the agent compose C *spontaneously*? |
| **warm recall+synthesis** | warm `/recall` + the Step-3.5 synthesis step | does the explicit step produce C reliably? |

**Metric:** reference-based — does the final answer state the specific emergent C (the join/chain
conclusion), checked against the fixture's C, not keywords. Run n≥5 per arm per fixture.

- **Value of MEMORY** = warm − cold (expected large; cold lacks the facts).
- **Value of the SYNTHESIS STEP** = (warm+synthesis) − (warm recall-only). **This is the number that
  decides the slice.**

**Build discipline (writing-skills TDD, gated on the eval):**
- **RED (must hold to proceed):** warm recall-only **fails to compose C** (low rate). If warm-only
  already produces C, **STOP — the step is redundant**; record the negative (as slice 2) and do not
  edit the skill.
- **GREEN:** add Step 3.5 → warm+synthesis produces C reliably AND beats warm-only by a margin above
  the noise floor (size the noise from warm-only-vs-warm-only, per the gap-below-noise memory).
- **REFACTOR / pressure tests:** an aggregation control (all-one-cluster vault → the step must NOT
  manufacture a false "emergent" C); a recitation control (C present in a note → not a synthesis win);
  the cross-domain clustering check (A, B really separate).

## 6. Deliverables of the build (separate effort)

1. `dev/eval/traps/synth_fixtures.py` — the cross-domain compositional fixtures (+ clustering check).
2. `dev/eval/traps/synth_eval.py` — the 3-arm warm harness + reference-based C-detection.
3. Run the eval RED first (cold / warm-only). Decision gate: only if warm-only fails to compose.
4. If justified: `skills/recall/SKILL.md` Step 3.5 via `superpowers:writing-skills` TDD; GREEN re-run.
5. Update note 68 (its prescribed fix — graph-expanded retrieval — is superseded; the lever is agent
   reasoning) and the roadmap.

## Self-review checklist
- Synthesis defined as emergent composition (A+B→C cross-domain), explicitly NOT aggregation (note 68)?
- Analogy confined to GENERATE; assertion requires a truth-preserving mode (note 69)?
- The redundancy risk (agent composes spontaneously) front-and-center, with a 3-arm eval that can kill
  the slice — warm-only-vs-warm+synthesis as the deciding number?
- Fixtures cross-domain + emergent-C-in-no-note + idiosyncratic (anti-cold, anti-recitation)?
- Build gated on RED (warm-only fails) — no skill edit unless the gap is proven, per the slice-2 lesson?
