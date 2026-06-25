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

**Net today — but the two axes diverge sharply (note 84), and conflating them was an earlier error:**

- **TIME:** `build_saving(~84 s) − overhead(~411 s) ≈ −327 s.` Warm is slower, and *structurally* so:
  recall's ~350 s wall-time alone **exceeds the entire cold build (288 s)**, so even a perfect (instant)
  build leaves the warm op slower (411 s vs 288 s). Net-*faster* requires cutting recall **wall-time**
  drastically (~80%) — not the build.
- **DOLLARS:** warm is ~$1.72 costlier *today*, but this is **not** structural. Recall is
  **time-expensive but dollar-cheap** (~49 K *input* tokens; the build's *generation output* is the $
  sink). If memory's upfront pointers cut enough build **rounds** (fewer mistakes/reviews/rebuilds — the
  $ sink), the build-$ saving can exceed recall's small $ and the op gets **cheaper.** That is memory's
  real value prop and it is **not blocked.** *Caveat:* recall's $ is bundled/inferred, never measured — so
  net-cheaper is **well-founded but unproven** until recall's $ is unbundled (PREREQ-$METER, findings §5).

**Corrected bottom line (supersedes the earlier "memory does NOT make builds cheaper or faster" — that
conflated the axes and was wrong on dollars):** a memory-assisted build **can be cheaper end-to-end** by
making the *same* app with fewer rounds — that's the goal, and nothing fundamental blocks it. **Faster**
end-to-end is the genuinely hard axis, because recall spends reasoning wall-time the cold path never
spends.

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

## Two goals, two levers (not one safe lever, and not a give-up fork)

The axes split, so treat cheaper and faster as separate questions with different answers:

- **CHEAPER — the achievable aim.** Lever = **maximize memory's build-round reduction**: push recall
  *quality* so the build makes the *same* app with fewer mistakes/reviews/tests/rebuilds, cutting the
  $-sink build loop (note 77). Recall's $ is small, so the build-round $ saving can net out positive — this
  is the "fewer rebuilds" mechanism, and it is **not blocked.** Prerequisite: **unbundle recall's $**
  (PREREQ-$METER, findings §5) to confirm the asymmetry and get a real net-$ figure rather than the
  current inference.
- **FASTER — the hard axis.** Lever = deeply cut recall **wall-time** (~350 s of reasoning) — the only
  thing big enough, since even a perfect build can't beat cold while recall runs longer than the entire
  cold build. Higher variance; **must gate on recall quality.** The safe cuts (O2 ~15–40 s) don't touch it.

**Recommendation: pursue CHEAPER.** It is the goal, it is not blocked, and it is exactly the
fewer-mistakes/rebuilds mechanism. Concretely: (1) **unbundle recall's $** to confirm net-cheaper is real,
then (2) drive recall *quality* → fewer build rounds, measuring the build-round $ delta. Treat **FASTER**
as a separate, harder problem — take it on only if op-wall-time-vs-cold is a hard requirement, since it
needs a risky recall-time trim. Do **O2** as free housekeeping regardless; don't mistake it for either
lever.

## Next step

Pursue **cheaper**, in order: (1) **unbundle recall's $** from `build_cost` (PREREQ-$METER) — this is the
one measurement that turns "cheaper is well-founded" into "cheaper is proven (or not)"; (2) measure how
much warm reduces **build rounds** vs cold, and push recall *quality* to widen that gap. Measure each
delta in isolation — don't bundle. (Faster, if ever required, is a separate recall-wall-time trim.)

## Honest caveats

- Source numbers are n=5 capped opus — directional, **not** high-power.
- Recall's $ is **bundled/inferred, never measured** — so "memory can be net-cheaper" is well-founded
  (recall is dollar-cheap; the build is the $ sink) but **unproven** until unbundled. That measurement is
  the next step, not an assumption.
- **Faster** end-to-end is the genuinely hard axis (recall spends reasoning wall-time the cold path never
  spends); it is not "impossible," but it needs a large recall-wall-time cut, not a build improvement.
