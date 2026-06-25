# Re-anchor — does engram memory make builds cheaper/faster?

**Date:** 2026-06-25 · **Status:** north-star re-anchor (re-statement from fresh data, no new run) ·
**Source data:** `2026-06-24-recall-miss-and-cost-round3-findings.md` §2 (capped opus n=5); vault notes
77, 84, 80, 90.

**Why this doc.** Work drifted several steps into *correctness* (the recall-miss → C7 harness #654 →
recall fixes #655 → task-displacement reproduction). That work is real but, by its own classification,
**cost-neutral** — it does not make builds cheaper or faster. This re-anchors on the actual goal and names
the one lever, grounded in evidence that is recent (2026-06-24) and unaffected by the cost-neutral edits
shipped since.

## The goal

The memory system must make builds **cheaper and faster, net.** Quality wins (better first-pass,
anti-amnesia, no task-displacement) are secondary unless they move that needle.

## Validated current status (capped opus, n=5; directional)

| Phase | Warm (with memory) | Cold (no memory) |
|---|---|---|
| recall | ~350 s | — |
| build | **~204 s** | **~288 s** |
| learn | ~61 s | — |
| **op total** | **~615 s / ~$3.78** | **~288 s / ~$2.06** |

- **Memory pays off ON THE BUILD:** the warm build is *faster and better* (204 s < 288 s; more
  first-pass conventions) — memory works as intended where it touches the build.
- **But recall+learn add ~411 s of overhead**, so the **warm op is ~2.1× slower and ~$1.72 costlier**
  end-to-end.

**Net today (the uncomfortable truth):**
`net(time) = build_saving(~84 s) − overhead(~411 s) ≈ −327 s` · `net($) ≈ −$1.72`.
**Memory makes the build better/faster, but the recall+learn overhead swamps that gain — so end-to-end,
memory does NOT yet make builds cheaper or faster. It makes them *better, slower, and costlier.***

## What is NOT the lever (closed / sub-share — do not re-open)

- **Recall payload size** — capping it −61% moved end-to-end cost by ~nothing; recall is a $ *sub-share*
  (note 77). Shrinking what recall *returns* is not where the cost is.
- **Recall model-split** (opus host + cheap retrieval) — built, measured −14%, **rolled back** (note 80).
  Closed.
- **Build-loop $ in isolation** — it's the largest *absolute* spend, but cutting it lowers warm *and*
  cold equally, so it doesn't shrink the warm-over-cold premium that makes memory fail to pay off
  (note 84). And recall's $ is *bundled* into `build_cost` — unbundling is a prerequisite, not a lever.

## The single highest-leverage lever

**Cut the recall+learn procedure overhead — specifically the recall Step 2.5 per-cluster loop**, the
dominant slice of the ~350 s recall time (blocking `engram show` round-trips + coverage-judge reasoning +
blocking writes). Concretely:

- **O2** — inline candidate-note content into the query payload → eliminate the blocking `engram show`
  round-trips.
- **L2** — skip Step 2.5 entirely on chunk-only clusters (no note members → provably a no-op there).
- **L3a** — batch the learn *ingest sweep* once per session (sweep only; never defer the note-write).

**Why this one.** The overhead is *exactly* what makes warm worse than cold on both axes, and Step 2.5's
round-trips are its largest, lowest-risk slice (round-3 risk: low). Cutting it pushes `net(time)` toward
positive and trims the $ premium **without touching the build-quality gain memory already delivers.** It
is mainly a **time (faster)** lever; it helps **cheaper** modestly.

*The bigger but riskier $ alternative* — making memory good enough to cut build *rounds* (the absolute $
sink) — stays on the table but is higher-variance and indirect; the build is already faster warm, so the
clear, safe problem to attack first is the overhead, not the build.

## Next step (one lever, measured — per the chosen approach)

Implement **O2 + L2** in recall Step 2.5 and **measure only this lever's delta** (recall wall-time
warm-vs-baseline on the current system), confirming no quality regression (recall/learn quality is itself
a measured output). Do not bundle in other changes; do not claim the win before the measurement.

## Honest caveats

- Source numbers are n=5 capped opus — directionally solid, **not** high-power. The lever's *impact* is
  to be measured when pulled, not assumed (note 90: validate fresh; don't claim the win in advance).
- If overhead cannot be cut below the build saving, the harder question reopens: **is the recall+learn
  procedure worth running at all** for the build-quality gain? Re-anchor here again if O2/L2 don't move
  `net` toward positive.
