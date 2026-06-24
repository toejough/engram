# Engram cost — round 2: the real bottleneck is the recall+learn *procedure*

**Date:** 2026-06-24 · **Status:** brainstorm (5 levers to evaluate, not decisions)

## What round 1 taught us

Option 1 (content cap, shipped default 15) cut the recall **payload −61%** — and end-to-end
C1/C2 (time/$) **did not move** within noise. The C1–C6 re-run on the capped binary confirmed the
value axes (C3–C6) are untouched, but the cost axes stayed negative. So **recall payload is not the
bottleneck.** This doc finds where the cost actually is.

## Where the time/$ actually goes (capped opus, n=5, per warm op)

| phase | warm (real.full) | cold | note |
|-------|------------------|------|------|
| recall | **350 s** | — | the overhead — runs ingest + 10-phrase query + read + Step 2.5 show calls + writes |
| build | **204 s** | 288 s | **memory makes the build FASTER** (front-loaded conventions → fewer rounds) |
| learn | **61 s** | — | crystallize + ingest, on the critical path |
| **total** | **615 s / $3.78** | 288 s / $2.06 | warm is slower/costlier *despite a faster build* |

**The reframe:** the build itself is *helped* by memory (204s < 288s; ~−29% build time). The memory
tax is the **recall + learn procedure overhead (411 s ≈ 67% of the warm op)** stacked on top. Every
lever below attacks that procedure — NOT payload (round 1, done) and NOT the build (memory already
wins there).

> Each lever: mechanism, the phase it attacks, expected savings on **both axes** ($ and time), risk,
> effort, rating. Savings are estimates to validate.

---

### Lever 1 — Run recall + learn on a cheaper/faster model than the build · **CONTENDER**

**Mechanism.** Execute the recall (read payload, judge coverage, write notes) and learn steps via a
haiku/sonnet subagent; return only the Step-3 synthesis + the build to opus. The 411 s of overhead is
mechanical curation, not the reasoning the opus build needs.

- **Phase:** recall (350 s) + learn (61 s) — the whole tax.
- **$:** large — the bulk of overhead tokens move to a ~10× cheaper model.
- **time:** medium — haiku is faster per token, but the step count is unchanged; latency drops less
  than $.
- **Risk:** medium — weak coverage/recency judgment (C4/C5) could degrade note quality. Validate on
  C4 supersession + C5 recency, where a weak judge mis-ranks.
- **Effort:** medium (orchestrate recall/learn as a model-overridden subagent).

### Lever 2 — Trim the recall *procedure* (steps, not payload) · **CONTENDER**

**Mechanism.** The 350 s is dominated by step count, not bytes: 10-phrase query, Step 2.5 per-cluster
`engram show` loop, per-cluster writes. Cut it: fewer phrases (round-1 O4, now better-motivated by
time not tokens), and **skip Step 2.5 synthesis entirely when the payload is chunk-only** (no note
clusters — the common case; it produced 0 writes in 3 of this session's recalls).

- **Phase:** recall (350 s) — attacks wall-clock directly.
- **$ / time:** medium on both — fewer query/show/write round-trips per recall.
- **Risk:** medium — fewer phrases lowers recall coverage (C5 recency is phrase-sensitive); the
  chunk-only skip is safe (it already writes nothing).
- **Effort:** low–medium (recall SKILL.md edit; writing-skills TDD).

### Lever 3 — Move learn off the critical path (async / batched) · **CONTENDER**

**Mechanism.** Learn (61 s) crystallizes + ingests *synchronously* after each build. Make it
fire-and-forget (the agent proceeds; ingest runs detached) and/or **batch crystallization to once per
session** instead of per-build. Raw chunks are already captured by the automatic sweep, so per-build
learn is mostly redundant within a session.

- **Phase:** learn (61 s × every build).
- **$ / time:** small–medium $ (fewer learn-LLM calls if batched); time win is removing 61 s from
  each build's critical path.
- **Risk:** low–medium — a later build in the same session won't recall a lesson not yet crystallized;
  acceptable if conventions are seeded once up front.
- **Effort:** medium (skill + harness change; detached ingest).

### Lever 4 — Memoize recall across a session/chain (compiled convention digest) · **PARK**

**Mechanism.** The 3-app chain re-runs full recall (350 s) every app for largely the same conventions.
Cache a small "active conventions" digest after app 1 and have apps 2–N read it cheaply instead of a
full recall — amortize the 350 s across the chain (seed-vs-payback, applied to *recall itself*).

- **Phase:** recall (350 s on apps 2..N).
- **$ / time:** large on a chain (apps 2–N skip most of recall); zero on a one-shot task.
- **Risk:** higher — a stale digest misses a newly-learned lesson; needs an invalidation rule. Cuts
  against "always recall fresh."
- **Effort:** medium–high (new caching layer + invalidation).

### Lever 5 — Tighter recall→build handoff to one-shot more builds · **PARK**

**Mechanism.** Builds still take 2–3 rounds. Memory already cuts build time; push further by feeding
recalled conventions into the build as **hard requirements** (not advisory prose) so the first draft
passes more often — fewer feedback rounds = less build time/$.

- **Phase:** build rounds (the part memory already helps).
- **$ / time:** medium — each saved round is ~one build-LLM turn.
- **Risk:** medium — over-constraining the first draft can cause churn; measure rounds-to-converge,
  not just pass-rate.
- **Effort:** medium (prompt/handoff change; harder to attribute).

---

## Ratings & sequence

| # | Lever | $ | time | risk | effort | rating |
|---|-------|---|------|------|--------|--------|
| 1 | Cheaper model for recall+learn | ↓↓↓ | ↓↓ | med | med | **CONTENDER** |
| 2 | Trim recall procedure (steps) | ↓↓ | ↓↓ | med | low–med | **CONTENDER** |
| 3 | Async/batched learn | ↓ | ↓↓ | low–med | med | **CONTENDER** |
| 4 | Memoize recall across chain | ↓↓ | ↓↓ | higher | med–high | PARK |
| 5 | Tighter handoff, fewer rounds | ↓ | ↓ | med | med | PARK |

*↓ small · ↓↓ medium · ↓↓↓ large (expected, unvalidated). `~` = no change.*

**Recommendation (next step, not part of the brainstorm):** **L1 + L2 together** — they attack the
same 411 s overhead from two angles (cheaper tokens × fewer steps) and are the lowest-risk. L3
(async learn) is orthogonal and stacks. L4/L5 are parked: L4 needs an invalidation story; L5's saving
is hard to attribute against build variance.

## How to validate (non-negotiable)

- Report **both axes** ($ and wall-time) per lever; the round-1 lesson is that a payload/token win need
  not be a time/$ win — measure end-to-end, not the proxy.
- For any "cheaper variant is as good" claim (L1, L2, L4), **test where it bites** — C4 supersession +
  C5 recency (a weak judge or thin phrasing fails there) — not a quality-blind metric.
- A/B each lever against the current capped recall on C3–C6 (quality must hold) AND the cumulative
  chain (cost must actually drop end-to-end) — but note the cumulative chain is ~$100/run on opus, so
  gate that spend or use a cheaper model for the cost-delta measurement.
