# Clean warm-vs-cold measurement — does memory make builds cheaper/faster?

**Date:** 2026-06-25 · **Status:** measured + adversarially verified (3-angle) ·
**Supersedes:** the mislabeled phase accounting in `2026-06-25-memory-cost-reanchor.md` and round-3 §2.
**Data:** `/tmp/cummatrix-warmcold/results/` (opus, trials t1–t3 × apps notes/links/feeds × {cold,
real.full}); decomposition `scratchpad/decompose_build.py` + `aggregate_warmcold.py`.

## Method (the fix for the `recall_s` mislabel)

Phases decomposed from each transcript by **first code-write** boundary (note 93): **recall** = start →
last `engram` call before the first Edit/Write; **build** = first code-write → converged (all rounds);
**learn** = the closing `/learn`. Cold = whole build session. n = **8 usable per arm** (2 of 18 discarded:
one cold did-not-converge, one warm rate-limited). Verified: boundary is clean (the straddling message is
attributed to recall, no token leak into build); per-phase $ reconcile to billed to 4 decimals.

## Result (warm vs cold, n=8/arm; Δ sized against the same-contrast spread)

| metric | Warm (with memory) | Cold | Δ | beyond noise? |
|---|---|---|---|---|
| **NET op — time** | 659.9 s (sd 83) | 477.7 s (sd 137) | **+182 s slower** | **YES** (d/SE 3.2, CLES 0.84) |
| **NET op — $ (billed)** | $5.37 (sd 1.5) | $2.29 (sd 0.7) | **+$3.08 costlier** | **YES, robust** (d/SE 5.1, CLES 0.97; warm loses 7/7) |
| build-phase time | 473.6 s | 477.7 s | −4.2 s | **NO — indistinguishable** (d/SE −0.08, CI ±110 s) |
| build-phase $ (raw tokens) | — | — | **+$1.00 warm costlier** | **YES** (d/SE 2.9) — *real, not artifact* |
| recall+learn overhead | +186 s / +$1.80 | — | — | the fixed memory tax |
| rounds-to-converge | 2.0 (sd 0) | 2.6 (sd 1.1) | −0.6 | **NO — same median**; cold scatter = 2 outliers |

## Verdict (verified)

**On these three small CRUD-app builds with opus, memory does NOT make builds cheaper or faster
end-to-end — it is materially slower (+182 s) and costlier (+$3.08), both robustly beyond noise.** Causes:

1. A **fixed ~186 s / $1.80 recall+learn tax** layered on top of the build.
2. The build phase memory is meant to *accelerate* doesn't: build **time** is statistically
   indistinguishable (not a tie — underpowered), and build **cost** is actually **~$1.00 higher warm** —
   the resumed recall context is re-read as cache on every build turn (confirmed on raw tokens, not a
   billing artifact). So warm pays *more* per build round even when it uses no more rounds.

**The hoped-for mechanism (memory → fewer rebuilds → cheaper/faster) did NOT pay off here, and the
amortization defense fails:** the apparent "consuming apps (app2/3) build faster warm" saving is an
**artifact** — the pooled −45 s collapses to −5.5 s when paired by app+trial and flips to warm-slower once
restricted to cells where cold also converged in 2 rounds. It was driven by two long *cold* outliers.

**The one genuine pro-memory signal — round consistency:** warm converged in exactly 2 rounds in **8/8**
cells (sd 0); cold scattered to 4–5 in 2/8. In the single paired case where cold ran 4 rounds, warm held
at 2 and saved 47 s — the mechanism firing **exactly where the premise predicts** — but it rests on 1–2
cells, below the noise floor, and never large enough to repay the ~186 s overhead.

## The honest scope boundary (what this does and does NOT prove)

**These tasks are too convergent to expose memory's benefit: 6/8 cold cells need ZERO rebuild** (converge
in 2 rounds cold). The fewer-rebuild mechanism can only act on the rare hard build — so a fixed overhead
on an already-easy build can only lose. This is **"underpowered → re-test on harder builds,"** NOT "memory
is useless" and NOT "tie":

- **Beyond noise (trust):** net warm slower + costlier; build-phase cost genuinely higher warm.
- **Not beyond noise (don't over-read):** build *time* indistinguishable; "fewer rounds" is lower
  *variance*, not lower median; consuming-app speedup is noise.
- **Cannot show (concede):** whether memory pays off on **harder, genuinely multi-round builds** where
  avoided-rebuild waste is large enough to exceed the overhead. The mechanism is *suggested* but
  underpowered here.

## Implication for the cost goal

For memory to make builds **cheaper/faster net**, the avoided-rebuild saving must exceed the **fixed
~186 s / $1.80 recall+learn tax** *and* the **per-round cost premium** of carrying recalled context into
the build. On easy 2-round builds that's impossible by construction. The real next test: **harder builds
with genuine multi-round rebuild waste** (the regime where memory's value lives) — plus attacking the two
fixed costs (the Step-2 recall payload paging; the recalled-context re-read each build turn). Until then,
the honest statement is: **memory currently buys build-quality consistency at a real time+cost premium; it
has not been shown to pay for itself on these tasks, and the harder-build regime is untested.**
