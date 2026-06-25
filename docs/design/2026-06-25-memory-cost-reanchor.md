# Re-anchor — does engram memory make builds cheaper/faster?

**Date:** 2026-06-25 · **Status:** north-star re-anchor (re-statement from fresh data, no new run) ·
**Source data:** `2026-06-24-recall-miss-and-cost-round3-findings.md` §2 (capped opus n=5); vault notes
77, 84, 80, 90. Gate-A reviewed (4 angles).

> **Answered (2026-06-25, measured + verified — see `2026-06-25-warm-vs-cold-clean-measurement.md`).**
> The clean warm-vs-cold run settles this doc's open "plausibly achievable, to be measured": on these
> 3 small CRUD builds (opus, n=8/arm), memory makes the op **slower (+182 s) and costlier (+$3.08)** —
> both beyond noise. The build phase is *not* accelerated (time indistinguishable; build **$ ~$1.00
> higher** warm — recalled context re-read each turn), and the amortization "consuming apps build faster"
> hope was an **artifact**. BUT the apps are **too easy** (6/8 cold builds converge in 2 rounds — no
> rebuild waste for memory to remove), so this is **"underpowered → re-test on harder, multi-round
> builds,"** not "memory can't pay off." The fork/levers below stand; the cheaper/faster *aspiration* is
> now bounded by measured reality.
>
> **Correction (2026-06-25, measured — see `2026-06-25-recall-cost-isolation.md`).** The "~350 s recall"
> used throughout this doc is a **mislabel**: the harness `recall_s` timed round-1 = recall **+ the first
> full code build**. **Recall-only is ~190 s** — *below* the cold build (~288 s). So this doc's "recall's
> 350 s exceeds the cold build ⇒ faster is structurally impossible/hard" is **RETRACTED**: faster is *not*
> structurally blocked. ~50 % of recall is **Step 2 — paging a ~200 KB query payload — and it is trimmable**
> (cap / clusters-first view). **Bottom line, corrected: a memory-assisted build being *both cheaper and
> faster* is plausibly achievable** (not yet proven): recall ~190 s is below the cold build and ~half of it
> is trimmable, so the *structural wall is gone* — but net-faster still needs the build-round saving to beat
> the trimmed recall+learn overhead, which must be measured. The concrete first lever is trimming the Step-2
> payload paging (a *time* cut; dollar-light). The time-axis pessimism below is superseded by the isolation
> doc's measured numbers.

**Why this doc.** Work drifted several steps into *correctness* (recall-miss → C7 harness #654 → recall
fixes #655 → task-displacement reproduction). That work is real but, by its own classification,
**cost-neutral** — it does not make builds cheaper or faster. This re-anchors on the actual goal, grounded
in evidence that is recent (2026-06-24) and unaffected by the cost-neutral edits shipped since.

## The goal

The memory system must make builds **cheaper and faster, net.** Quality wins (better first-pass,
anti-amnesia, no task-displacement) are secondary unless they move that needle.

## Validated current status (capped opus, n=5; directional)

> **⚠ This table is the ORIGINAL, mislabeled accounting — superseded (2026-06-25).** The harness
> `recall_s`/`build_s` fields are **round-1 vs rounds-2+**, NOT recall vs build (`recall_s` = recall + the
> first code build). So the "recall ~350 s" row is round-1, not recall — measured **recall-only ≈ 190 s**
> (isolation doc) — and the warm-vs-cold *build* comparison ("204 < 288") below is **not clean** (it pits
> warm rounds-2+ against cold's whole op). The only solidly-measured figure is **recall-only ≈ 190 s**; a
> clean, comparable warm-vs-cold re-measurement is still owed. Rows below are kept as the superseded
> original view; corrected conclusions are in the Correction banner + the isolation doc.

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

- **Memory *appears* to pay off on the build** (more first-pass conventions; "204 s < 288 s" — **but that
  is the unclean comparison flagged in the table caveat:** warm rounds-2+ vs cold's whole op, not like for
  like; owed a clean re-measurement).
- **But recall+learn add ~411 s of overhead** (350 s of it recall), so the **warm op is ~2.1× slower and
  ~$1.72 costlier** end-to-end.

**Net today — but the two axes diverge sharply (note 84), and conflating them was an earlier error:**

- **TIME (corrected 2026-06-25 — supersedes the −327 s below it):** that −327 s used the **mislabeled**
  350 s (= round-1 = recall + first build). **Measured recall-only is ~190 s — *below* the cold build
  (~288 s)** — so faster is **NOT structurally blocked.** Whether the warm op nets faster turns on the
  build-round time saving vs the recall+learn overhead (~190 + 61 ≈ 251 s), and ~half of recall (~80–115 s
  of Step-2 payload paging) is **trimmable** — which would cut that overhead materially. **Plausibly
  achievable, to be measured** — not a wall. (The original −327 s text is retained below only as the
  superseded reasoning that the mislabel produced.)
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

So the safe cuts O2/L2 recover only **~15–40 s** — small. (They were sized against the mislabeled
"411 s / −327 s gap" — see the table caveat; the corrected gap is smaller and the structural blocker is
retracted. The surviving point holds regardless: **O2/L2 are too small to be "the lever."**) The *real*
dominant slice — confirmed by the isolation measurement — is **Step 2's ~200 KB payload paging (~half of
recall's ~190 s)**; that, not O2/L2, is the lever. Selling the tiny safe cuts as "the lever" would be the
exact "we once thought it'd help" failure this re-anchor exists to stop.

## The axes don't share a lever (note 84)

There is **no single change that moves both axes** — round-3 already found this. Keep them separate:

- **FASTER (op wall-time) — corrected 2026-06-25:** recall is **~190 s** (not 350 s), *below* the cold
  build, so faster is **not structurally blocked.** Its dominant slice is **Step 2 — the agent paging a
  ~200 KB query payload (~half of recall)** — *not* diffuse reasoning. The lever is therefore **trim the
  Step-2 payload** (cap / clusters-first view so it reads in one pass) — **lower-risk** than a
  reasoning-trim. (The earlier "trim the ~350 s of per-invocation reasoning" framing was the mislabel.)
- **CHEAPER (op dollars):** recall is a $ sub-share (note 77); the **build loop is the $ sink.** Memory's
  cost leverage is **reducing build rounds** (plausibly fewer rounds — the "build already faster"
  comparison is unclean per the table caveat; to be measured). Cutting
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
- **FASTER — plausibly achievable (corrected, not a wall).** Recall is **~190 s** (not 350 s), *below* the
  cold build — no structural wall. Concrete lever = trim the **Step-2 query-payload paging** (~80–115 s,
  ~half of recall: the agent re-reads a ~200 KB payload that won't fit one tool read): cap it / return a
  clusters-first view so it's consumable in one pass. A **time** cut (dollar-light — payload is cheap
  `cache_read`). Then net-faster turns on the build-round saving beating the trimmed recall+learn overhead
  — to be measured. Gate on recall quality.

**Recommendation: pursue CHEAPER.** It is the goal, it is not blocked, and it is exactly the
fewer-mistakes/rebuilds mechanism. Concretely: (1) **unbundle recall's $** to confirm net-cheaper is real,
then (2) drive recall *quality* → fewer build rounds, measuring the build-round $ delta. **FASTER is now
also in reach** (corrected): the concrete lever is trimming the **Step-2 payload paging** (~half of
recall's ~190 s), which is lower-risk than a deep reasoning trim — pursue it alongside cheaper if
op-wall-time matters. Measure each lever's delta in isolation; keep the axes separate (note 84).

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
- **Faster** end-to-end is **no longer structurally blocked** (corrected): measured recall (~190 s) is
  *below* the cold build, and ~half of it is the trimmable Step-2 payload paging. Net-faster is *plausibly
  achievable* but unproven — it still needs the build-round saving to beat the trimmed recall+learn
  overhead; measure it.
