# Re-anchor — does engram memory make builds cheaper/faster?

**Date:** 2026-06-25 · **Status:** north-star re-anchor (re-statement from fresh data, no new run) ·
**Source data:** `2026-06-24-recall-miss-and-cost-round3-findings.md` §2 (capped opus n=5); vault notes
77, 84, 80, 90. Gate-A reviewed (4 angles).

**Why this doc.** Work drifted several steps into *correctness* (recall-miss → C7 harness #654 → recall
fixes #655 → task-displacement reproduction). That work is real but, by its own classification,
**cost-neutral** — it does not make builds cheaper or faster. This re-anchors on the actual goal, grounded
in evidence that is recent (2026-06-24) and unaffected by the cost-neutral edits shipped since.

## The goal

The memory system must make builds **cheaper and faster, net.** Quality wins (better first-pass,
anti-amnesia, no task-displacement) are secondary unless they move that needle.

## Validated current status (capped opus, n=5; directional)

| Phase | Warm time (with memory) | Cold time (no memory) |
|---|---|---|
| recall | ~350 s | — |
| build | **~204 s** | **~288 s** |
| learn | ~61 s | — |
| **op total — time** | **~615 s** | **~288 s** |
| **op total — dollars** | **~$3.78** | **~$2.06** |
| **net — time (warm − cold)** | **+327 s slower** | — |
| **net — dollars (warm − cold)** | **+$1.72 costlier** | — |

*(Dollars are op-total only: recall's $ is bundled into `build_cost` and not separately metered — note 84.)*

- **Memory pays off ON THE BUILD:** the warm build is *faster and better* (204 s < 288 s; more first-pass
  conventions) — memory works where it touches the build.
- **But recall+learn add ~411 s of overhead** (350 s of it recall), so the **warm op is ~2.1× slower and
  ~$1.72 costlier** end-to-end.

**Net today (the uncomfortable truth):** `build_saving(~84 s) − overhead(~411 s) ≈ −327 s.`
**Memory makes the build better/faster, but the recall+learn overhead swamps that gain — so end-to-end,
memory does NOT yet make builds cheaper or faster. It makes them *better, slower, and costlier.***

## The honest magnitude: the safe cuts cannot close the gap

The earlier draft named **O2 + L2** (trim the recall Step 2.5 per-cluster loop) as "the single
highest-leverage lever." Gate A refuted that, and it's right:

- **O2** (inline candidate-note content → drop the blocking `engram show` round-trips) is real and
  code-verified (`skills/recall/SKILL.md` Step 2.5.A mandates the round-trips), but its measured ceiling
  is **~15–40 s** (round-3 §2).
- **L2** (skip chunk-only clusters) is **already implemented** — the skill already skips them
  (`skills/recall/SKILL.md:112`). Its residual is loop-iteration overhead only, not a Step-2.5-sized cut.

So the safe cuts recover **~15–40 s of a 411 s overhead / −327 s gap (~5–12%)**. They are worth doing as
free housekeeping, but they **do not make memory net cheaper/faster** — net stays deeply negative. Selling
them as "the lever" would be the exact "we once thought it'd help" failure this re-anchor exists to stop.

## The axes don't share a lever (note 84)

There is **no single change that moves both axes** — round-3 already found this. Keep them separate:

- **FASTER (op wall-time):** the gap is dominated by the **~350 s recall**, which is mostly *LLM reasoning*
  across recall's steps (Step-0 plan, 10-phrase query, Step 2.5 coverage-judging, Step-3 synthesis), not
  I/O. Closing it means **trimming the recall procedure's per-invocation reasoning** (fewer/lighter steps)
  — **higher variance, recall-quality risk** (recall quality is the whole point of memory).
- **CHEAPER (op dollars):** recall is a $ sub-share (note 77); the **build loop is the $ sink.** Memory's
  cost leverage is **reducing build rounds** (the warm build is already faster → fewer rounds). Cutting
  the build loop directly lowers cold too, so it doesn't shrink the premium (note 84).

## What is NOT the lever (closed / sub-share — do not re-open)

- **Recall payload size** — capping it −61% moved end-to-end cost by ~nothing (note 77).
- **Recall model-split** (opus host + cheap retrieval) — built, −14%, **rolled back** (note 80). Closed.
- **Build-loop $ in isolation** — lowers warm *and* cold equally; doesn't shrink the premium (note 84).

*These are sub-problems or prerequisite moves, not levers; naming them keeps us from re-walking them.*

## The real choice (a fork, not a single safe lever)

Because the overhead is mostly inherent recall reasoning and the safe cuts can't dent it, the honest
re-anchor is a strategic fork:

- **(A) Chase net-faster ops:** deeply trim the recall procedure's reasoning (~350 s) — the only thing big
  enough. Higher variance; **must gate on recall quality** (the C-sweep quality is a measured output).
- **(B) Reframe memory's value as *better/cheaper builds, not faster ops* (recommended):** accept that
  recall reasoning is inherent overhead the op can't beat cold on wall-time, and put the eye on memory's
  real leverage — **reducing build rounds in the $-sink build loop** (note 77). Judge memory on
  build-cost/quality per task, not op-speed-vs-cold.

**Recommendation: (B).** Memory's leverage is the expensive build loop, not making a reasoning-heavy
recall faster than *no* recall. **Be explicit about the trade: (B) gives up "faster ops" as a target** —
it accepts the warm op stays slower than cold (recall reasoning is inherent) and wins instead on build
**cost + quality**; the "faster" half of the stated goal is judged not achievable via a reasoning-heavy
recall. (A) is the path only if op-speed-vs-cold is a hard requirement, and only at quality risk. Do
**O2** as free housekeeping regardless; don't pretend it closes the gap.

## Next step

This is a **decision**, not an implementation: pick the fork. Under **(B)**, the next real lever to attack
is **build-round reduction** — measure whether (and how much) warm reduces build rounds vs cold, then push
recall *quality* (not speed) to widen that gap. Under **(A)**, scope a recall-procedure-reasoning trim with
a quality gate. Either way, **measure only the chosen lever's delta** — do not bundle.

## Honest caveats

- Source numbers are n=5 capped opus — directional, **not** high-power.
- The deepest open question, now explicit: **if recall reasoning is inherent, the warm op cannot beat cold
  on wall-time** — so "faster ops via memory" may be the wrong target, and the honest win is *better,
  cheaper builds*. Re-anchor here again if the chosen lever doesn't move `net` toward positive.
