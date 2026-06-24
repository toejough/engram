# Engram cost — round 2: time sink ≠ dollar sink

**Date:** 2026-06-24 · **Status:** brainstorm (5 levers to evaluate, not decisions)
**Builds on:** `2026-06-24-engram-cost-reduction-options.md` (round 1 — Option 1 content cap, shipped).
Round-1 **O3** (cheaper model for recall/learn) is re-motivated here as **L1**. Round-1 **O2** (inline
candidate notes to drop `engram show`) is a *distinct* time lever — related to **L2** (it also cuts
recall round-trips) but not the same change; it remains available and is not superseded.

## What round 1 taught us

Option 1 (content cap, shipped default 15) cut the recall **payload −61%** and end-to-end C1/C2
(time/$) **did not move** within noise; C3–C6 held. So **recall payload is not the bottleneck.** This
doc asks: then where *is* the cost — and splits the question by axis, because **the time sink and the
dollar sink turn out to be different things.**

## Where time and dollars actually go (capped opus, n=5)

**Time — measured per phase** (warm op, recomputed from the per-op JSON):

| phase | warm | cold | note |
|-------|------|------|------|
| recall | **350 s** | — | blocking round-trips: ingest + 10-phrase query + read + Step 2.5 `show` loop + writes |
| build | **204 s** | 288 s | memory makes the build FASTER (pooled −29%; but app1 gap −8% is within noise) |
| learn | **61 s** | — | synchronous crystallize + ingest, on the critical path |
| **total** | **615 s / $3.78** | 288 s / $2.06 | warm slower despite a faster build |

**Dollars — NOT split per phase.** The harness logs the *entire* warm-op cost as one number
($3.78); `recall_cost`/`learn_cost` are not separately metered. So:

- **Time bottleneck (measured):** the recall+learn *procedure* — **411 s ≈ 67% of the warm op** —
  stacked on top of a build that memory already speeds up. This is wall-clock from many blocking
  round-trips, not big token counts.
- **Dollar bottleneck (inferred, not yet metered):** the **build** — it's where the code-gen tokens
  are, and it's logged as `build_cost`. A capped recall is ~49K tokens; the build runs 2–3 rounds of
  generation. **We do not have a per-phase $ split — instrumenting one is itself a prerequisite
  finding** (see validation).

**So the levers split by axis:** L1–L3 attack the **time** sink (recall+learn procedure); L4–L5 attack
the **dollar** sink (the build loop) — the cost driver note 77 named. None re-attacks payload (round 1).

> Each lever: mechanism, the axis/phase it attacks, expected effect on **both** axes (honestly marking
> what is unmeasured), risk + an operational trigger, effort, rating. Effects are estimates to validate.

---

### Lever 1 — Run recall + learn on a cheaper/faster model (round-1 Option 3, now time-motivated) · **CONTENDER**

**Mechanism.** Execute recall (read payload, judge coverage, write notes) + learn on a haiku/sonnet
subagent; return only the Step-3 synthesis + build to opus. Round 1 sketched this as a $ play (O3);
the new wall-clock data re-motivates it as primarily a **time** play on the 411 s overhead.

- **time:** medium — haiku is faster per token, but the *step count* (query → show loop → writes) is
  unchanged, so latency from round-trips remains; the win is per-step token speed, not fewer steps.
- **$:** **unknown** — depends on recall's unmeasured $ share. If recall is a small-$/large-time slice
  (likely — few tokens, many blocking calls), the $ win is small even though the time slice is large.
  Do not claim a large $ win until the per-phase $ split exists.
- **Risk:** medium. **Trigger:** if the cheaper model mis-ranks > 1 recency/supersession conflict per
  5-note sample (C4/C5), revert — a weak judge corrupts notes.
- **Effort:** medium (recall/learn as a model-overridden subagent).
- **TRIED 2026-06-24 (haiku quality probe, n=5):** the doc's named risk — cheaper model *mis-ranks*
  recency/supersession — **did not materialize**: haiku had 0 mis-rankings (C4 warm-XXp 4/5, all 4
  correct + 1 no-apply), recall_fired 25/25 (C3), C5 surfaced 5/5. Recall *curation* holds on haiku at
  ~5× lower cost. The misses cluster in *applying* conventions (C3 18/25) and *reasoning* (C6 3/10) —
  the build half L1 keeps on opus. So the whole-op probe conflates the halves; **L1 is GREEN to build
  as the real split (haiku recall → opus build)** + confirm n=10 on C4/C5. See EXPERIMENT-LOG 2026-06-24.

### Lever 2 — Trim the recall *procedure* steps (round-1 Option 4, now time-motivated) · **CONTENDER**

**Mechanism.** The 350 s is step-bound, not byte-bound. Cut steps: fewer phrases (round-1 O4, parked
then as token-dominated-by-O1; now justified by **wall-clock**), and **skip Step 2.5 synthesis when the
payload has no note-kind members** (the chunk-only case — Step 2.5 already writes nothing there, so the
skip is provably safe; it is NOT a heuristic guess).

- **time:** medium — fewer query/show/write round-trips directly cut the 350 s.
- **$:** small — slightly fewer tokens; the gain is time.
- **Risk:** medium for fewer phrases (C5 recency is phrase-sensitive); **none** for the chunk-only skip
  (it changes nothing on payloads that have notes). **Trigger:** if dropping to 5 phrases lowers C5
  surfaced below the n≥3 warm baseline beyond noise, keep 10.
- **Effort:** low–medium (recall SKILL.md edit; writing-skills TDD).

### Lever 3 — Move learn off the critical path (async / batched) · **CONTENDER (with a correctness gate)**

**Mechanism.** Learn (61 s) crystallizes + ingests synchronously after each build. Two variants:
**(a) batch to once per session** (raw chunks are already auto-swept, so per-build crystallization is
mostly redundant within a session) — removes 61 s from every build, **no race**; or **(b) detached
per-build ingest** — faster but introduces a **vault-staleness race**: the next build's recall could
query before the prior lesson is indexed.

- **time:** medium — removes 61 s/build from the critical path.
- **$:** small (batching = fewer learn-LLM calls).
- **Risk:** **(a) low** (session-end batch — the recommended variant); **(b) high** — breaks the
  synchronous-write guarantee. **Gate:** variant (b) needs a completion signal before the next recall,
  or explicit eventual-consistency semantics. Default to (a).
- **Effort:** medium.

### Lever 4 — Cut build rounds via a hard-requirement handoff (the $ lever) · **CONTENDER**

**Mechanism.** The build is the **dollar** sink (2–3 rounds of generation = the $3.78). Memory already
trims rounds (warm build < cold); push further by feeding recalled conventions into the build as
**hard requirements / a checklist** the first draft must satisfy, not advisory prose — fewer feedback
rounds = fewer code-gen turns = less $.

- **$:** **medium–large, but inferred** — each saved round is a whole generation turn, and the build
  is where the tokens concentrate; this is aimed at the *likely* cost driver, but "build = the $ sink"
  is itself an inference until the per-phase $ split is metered (validation).
- **time:** medium — fewer build rounds.
- **Risk:** medium — over-constraining can cause churn. **Trigger:** measure rounds-to-converge AND
  pass-rate; if rounds drop but pass-rate falls, the handoff is too rigid.
- **Effort:** medium (recall→build prompt/handoff change).

### Lever 5 — Tier the build itself (cheap scaffolding, opus for convergence) · **PARK**

**Mechanism.** Run the build's early scaffolding rounds on a cheaper model, escalating to opus only for
the rounds that fail to converge. Directly attacks the dominant $ (build generation) by moving the
easy turns to a cheap model.

- **$:** large *if* early rounds are cheap-model-sufficient; **time:** medium.
- **Risk:** higher — a weak scaffold can produce churn opus must then untangle, erasing the saving;
  hard to attribute against build variance. **Trigger:** net $ must beat single-model opus at equal
  pass-rate, or park.
- **Effort:** high (multi-model build loop + escalation policy).

---

## Ratings & sequence

| # | Lever | axis | $ | time | risk | effort | rating |
|---|-------|------|---|------|------|--------|--------|
| 1 | Cheaper model for recall+learn | time | ? | ↓↓ | med | med | **CONTENDER** |
| 2 | Trim recall procedure steps | time | ? | ↓↓ | med | low–med | **CONTENDER** |
| 3 | Async/batched learn (variant a) | time | ? | ↓↓ | low | med | **CONTENDER** |
| 4 | Hard-requirement handoff (fewer build rounds) | **$** | ↓↓? | ↓↓ | med | med | **CONTENDER** |
| 5 | Tier the build model | **$** | ↓↓↓? | ↓↓ | higher | high | DEFER (pending L4 + the $ meter) |

*↓ small · ↓↓ medium · ↓↓↓ large (expected, unvalidated). `?` = effect depends on the unmeasured
per-phase $ split — every $ cell here is an inference from token distribution, not metered. DEFER =
high-potential but blocked on prior work, not ruled out.*

**Recommendation (next step, not part of the brainstorm):** **instrument the per-phase $ split first**
(cheap, deterministic — meters recall vs build vs learn cost) so the $ levers can be aimed with data,
not inference. Then: **L4** for the $ axis (the build is the measured cost driver) and **L2 + L3a** for
the time axis (lowest-risk procedure trims). L1 is a contender but its $ payoff is unknown until the
split exists; L5 is parked on effort + attribution risk.

## How to validate (non-negotiable)

- **First, meter the per-phase $ split** (recall vs build vs learn) — every $ claim in this doc is an
  inference until then, and it's the cheapest, highest-value next step. Then report **both axes** ($
  and wall-time) end-to-end per lever; round 1's lesson is that a proxy win (payload tokens) need not
  be an end-to-end win.
- For any "cheaper variant is as good" claim (L1, L5), **test where it bites** — C4 supersession + C5
  recency — not a quality-blind metric; honor each lever's operational trigger above.
- A/B against the current capped recall on C3–C6 (quality holds) AND the cumulative chain (cost drops).
  The chain is ~$100/run on opus — gate that spend or measure the cost-delta on a cheaper model.
