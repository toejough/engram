# Final memory evaluation — cold/warm, layers, recall strategy, accumulation

Date: 2026-06-01. Synthesis of a full session of agent-memory experiments run
against a name-agnostic 10-item architecture-convention rubric (and, for the
prior program, 17- and 18-item spec rubrics). Variables swept: memory
configuration {cold, L1 episodes, L2 facts, L3 ADRs, L1+L2, L1+L2+L3}; source
{single-app curated, +2nd-source accumulated, fresh-learn 3-app chain}; recall
{tier-isolated vs blended/kind-agnostic}; app-order {3 cyclic orders}. Every cost
is reported in three dimensions: **TIME** (build wall-clock), **TOKEN COST**
($ and turns), and **HUMAN-IN-THE-LOOP** (review rounds).

## 1. Executive summary

Memory's measurable value in these experiments is **autonomous convergence**, not
raw quality: with the same opinionated build prompt, a fact-bearing vault (L2, or
any stack containing it) reaches a passing architecture solution **in round 1 with
zero human review rounds**, while a cold agent reaches the *same* bar but only
after a human review round (1 round to the arch-only bar; up to 4 rounds to the
full feature+arch spec). On the arch-only bar that means cold costs ~2x the turns
and dollars of L2 to land in the same place; on the full-spec bar the no-memory
review loop costs ~4.3x ($3.53 vs $0.82) because each resumed round re-sends a
growing transcript. **L2 specific-facts is the strongest single tier** (autonomous,
cheapest, highest on the fresh-learn chain). Two things did **not** help:
accumulating a 2nd/3rd source (flat-to-slightly-negative in both the +bookmarks
sweep and the fresh-learn chain), and tier-isolated recall (2-5/10) which is much
weaker than blended kind-agnostic recall over a curated vault (8-10/10). All
results are small-n (1-3 per cell) with heavy variance and a heuristic scorer, and
because the rubric is drawn from the notes' own content this measures a
**memory-vs-review channel**, not emergent transfer.

## 2. Master table — memory configurations, all three cost dimensions

Single-sourced from the **layer-isolation convergence run** (the only experiment
spanning all six configs *and* carrying a varying human-review-rounds column).
Round-1 arch is from that same run's layer-isolation table; the $ figures match to
3 decimals, so the columns are coherent. Test build = contacts; memory derived
from a curated todo vault; recall = blended/kind-agnostic.

**Bar = architecture-only, arch ≥ 9/10 (name-agnostic scorer).** All six configs
*converge* to ~9/10 — the differentiator is how many rounds and whether a human is
in the loop.

| memory config | round-1 arch /10 | converged round | HUMAN review rounds | TOKEN turns | TOKEN $ | TIME build (min) |
|---|---:|---:|---:|---:|---:|---:|
| cold (empty) | ~4 | 2 | **1** | 72 | $1.910 | 10.06 |
| L1 (episodes) | 8 | 2 | **1** | 28 | $0.928 | 5.60 |
| **L2 (facts)** | **9** | **1** | **0** | **18** | **$0.548** | **3.07** |
| L3 (ADRs/distilled) | 8 | 2 | **1** | 38 | $1.389 | 9.01 |
| L1+L2 | 9 | 1 | **0** | 17 | $0.672 | 3.59 |
| L1+L2+L3 | 9 | 1 | **0** | 22 | $0.862 | 3.78 |

Reading: the three configs that contain L2 (L2, L1+L2, L1+L2+L3) all converge in
**round 1 with 0 human rounds**. Cold and the two partial tiers (L1 episodes, L3
distilled) each need **1 human review round** to polish in the `--json`/color
specifics that L2 carries verbatim. L2 alone is the cheapest path on every axis
(18 turns, $0.548, 3.07 min). Stacking tiers stays autonomous but *adds* cost
without buying earlier convergence (L1+L2+L3 = $0.862 vs L2's $0.548).

**Two distinct convergence bars — do not conflate the human-round counts:**
- **Arch-only bar (≥9/10), table above:** cold needs **1** human round.
- **Full feature+arch spec (17-18 items), prior program:** no-memory needs **4**
  human rounds to drip-feed every requirement (memory: 1, autonomous one-shot).
  Evidence — exact memory-vs-no-memory on contacts to ~15-16/17:

  | path | HUMAN rounds | TOKEN turns | TOKEN $ | TIME | autonomous |
  |---|---:|---:|---:|---|---|
  | memory (actionable learn + as-reqs recall) | 1 | 27 | $0.82 | — | yes (one shot) |
  | no memory (review loop) | 4 | 56 | $3.53 | — | no (human feeds reqs each round) |

  No-memory rounds escalate ($0.53 / 0.53 / 0.82 / 1.65) because each `--resume`
  re-sends the growing transcript; memory avoids that with a single primed build.
  Corroborated by the isolated episode-only todo run: COLD = 4 rounds / 61t /
  $4.34 vs WARM (3 episodes) = 1 round / 29t / $0.95 (~4.5x cheaper).

## 3. Headline findings (each with its evidence)

### (a) Memory's core value is AUTONOMOUS convergence — it removes human review rounds

Across both bars, memory collapses the human-in-the-loop dimension to 0-1 rounds
while the cold/no-memory path needs 1-4.
- **Arch-only bar (master table):** L2 / L1+L2 / L1+L2+L3 converge **round 1, 0
  human rounds**; cold + L1 + L3 each need **1**. Cold pays ~2x L2's turns (72 vs
  18) and dollars ($1.91 vs $0.55) to reach the identical ~9/10.
- **Full-spec bar:** no-memory = **4 human rounds**, 56 turns, $3.53; memory = **1
  round** (autonomous one-shot), 27 turns, $0.82 — **~4.3x cheaper, ~2x fewer
  turns**, and it clears the bar without a human feeding requirements.
- Isolated episode-only todo run corroborates: COLD 4 rounds → WARM 1 round, ~4.5x
  cheaper.

The point is not that cold *can't* build clean architecture — name-agnostic
re-scoring shows it does (9/10) — it's that cold needs a human in the loop to get
there, and memory removes that human.

### (b) L2 facts are the strongest single tier

- **Master table:** L2 is the only single tier that is round-1 autonomous (round
  1, 0 human rounds) and it is the cheapest config on every axis (18 turns,
  $0.548, 3.07 min). L1 episodes and L3 distilled each drop the `--json`/color
  specifics and need a polish round; L3 is the priciest converged-in-2 arm ($1.39).
- **Fresh-learn chain (tier-isolated, 9 builds/layer):** L2 is highest in **every
  cell** and by row-mean arch (**5.11/10** vs L1 2.56, L3 3.44) **at the lowest
  turn count** (165 vs L1 202, L3 255). L2 wins on quality and cost simultaneously.
- **Mechanism:** facts retain the load-bearing specifics (exact interfaces, flags,
  edge cases) that episodes bury in narrative and that distillation compresses away.

### (c) Accumulating a 2nd/3rd source shows NO clear benefit

Two independent experiments, both flat-to-declining:
- **+bookmarks sweep (round-1 autonomous, n=2, contacts):** adding a tests+bookmarks
  layer never *helped* arch; the verdict (help if Δ≥+1.0, dilute if ≤−1.0) lands
  two pairs neutral and two exactly on the −1.0 dilute boundary — the weakest
  possible non-neutral signal at n=2:

  | layer | todo-only arch | +bookmarks arch | Δ | verdict |
  |---|---:|---:|---:|---|
  | L1 | 10.0 | 9.0 | −1.0 | dilute (boundary; one trial −1pt) |
  | L2 | 10.0 | 10.0 | 0.0 | neutral |
  | L3 | 9.0 | 8.0 | −1.0 | dilute (boundary; one outlier build) |
  | all | 9.0 | 8.5 | −0.5 | neutral |

  Single-tier arms already saturate arch at 10/10 — no headroom to add, only
  downside from dilution. The bookmarks deltas are *off-domain* for contacts
  (browser-open/side-effect injection contacts never needs), so the enriched note
  is longer, not more useful.
- **Fresh-learn chain (accumulation stage s = prior apps in memory):** the curve is
  **mixed, not uniformly rising** — accumulation helps L1 and L2 by +0.33 each
  (cold→+2, monotonic non-decreasing) but *hurts* L3 by −0.33 (cold is L3's peak):

  | layer | cold (s0) | +1 (s1) | +2 (s2) | row-mean |
  |---|---:|---:|---:|---:|
  | L1 | 2.33 | 2.67 | 2.67 | 2.56 |
  | L2 | 5.00 | 5.00 | 5.33 | 5.11 |
  | L3 | 3.67 | 3.33 | 3.33 | 3.44 |

  The gains are within scorer noise; the robust signal is "memory helps a lot;
  +2nd-source adds nothing." Quality/curation of memory beats quantity.

### (d) Tier-ISOLATED recall is weaker than blended/kind-agnostic recall

- **Tier-isolated fresh-learn chain:** best layer (L2) tops out at row-mean
  **5.11/10**, with L1 at 2.56 and L3 at 3.44 — the whole chain lives in the 2-5/10
  band.
- **Blended kind-agnostic recall over a curated vault** (master-table / generalization
  runs): the same conventions land at **8-10/10** (e.g. contacts +todo = 14/17 ≈
  8/10 arch; L2/L1+L2/L1+L2+L3 = 9/10 on the arch rubric).
- Blending tiers and curating the vault roughly **doubles** the architecture score
  versus reading a single tier in isolation from a fresh-learned vault.
- **Confound (honest):** the chain varies *two* things at once — tier-isolated-vs-
  blended recall AND fresh-learn-from-cold-vs-hand-curated-vault. So this is a
  directional comparison of recall *regimes*, not a clean isolation of recall
  strategy alone.

### (e) Distillation/L3 consolidates correctly, but a tier-capped L3 recall is starved

- **Consolidation works:** distilling the bookmarks build into the existing todo L2/L3
  produced **100% elaboration, 0 new notes** — the architecture recurs and the only
  delta is added actionable detail (the DI principle generalizing from
  `Store`/`Clock`/`Writer` to "inject ANY side-effect"). The L3 layer correctly
  captures the cross-cutting pattern rather than duplicating it.
- **But capped L3 recall is starved:** on the master table L3-alone needs a polish
  round (drops `--json`/color specifics that L2 carries) and is the priciest
  converged-in-2 arm ($1.39); on the fresh-learn chain L3 is the *only* layer where
  accumulation hurts (−0.33). Distillation trades away the load-bearing specifics —
  so an L3-only recall budget retrieves the principle but not the details needed to
  satisfy a specific-item rubric. L3 earns its keep *stacked under* L2, not as a
  standalone recall tier.

## 4. Aggregate session totals

Summed over the **three fresh runs only** (convergence, +bookmarks sweep,
fresh-learn chain). The prior-program tables (specMatch/18, conventions/17) are
reported above as corroboration at *different bars* and are **not** folded in — and
the +bookmarks table in the prior-program section is the same data as the sweep, so
it is counted once. Build basis for the convergence run is **builds-to-converge**
(matching the cost-to-converge figures it reports): cold=2, L1=2, L2=1, L3=2,
L1+L2=1, L1+L2+L3=1 = 9 builds.

| run | builds | TOKEN turns | TOKEN $ | TIME (min) | HUMAN rounds |
|---|---:|---:|---:|---:|---:|
| Convergence (6 configs, to-converge) | 9 | 195 | $6.31 | 35.1 | 3 |
| +bookmarks sweep (9 arms × n=2) | 18 | 345 | $12.37 | 71.0 | 0 |
| Fresh-learn chain (3 layers × 3 orders × 3 stages) | 27 | 622 | $19.02 | 95.5 | 0 |
| **GRAND TOTAL** | **54** | **1162** | **$37.70** | **201.6 (3.36 h)** | **3** |

All 3 human review rounds came from the convergence run's cold / L1 / L3 arms (1
each); the sweep and chain were fully autonomous.

## 5. Honest caveats

- **Small n, heavy variance.** n=1 for the convergence run and the prior-program
  cells; n=2 for the +bookmarks sweep; n=3 per cell (the 3 cyclic orders) for the
  fresh-learn chain. The 0-vs-1 human-round splits and the layer cost-ordering hinge
  on 1-2 rubric items (`--json` / color). The accumulation "dilute" verdicts sit
  exactly on the −1.0 boundary (one trial differing by one point) — directional,
  not significant. ±1-point swings (e.g. l3bm's [9,7]) are within scorer noise.
- **Heuristic scorer, with one bug caught + fixed mid-session.** The first scorer was
  **name-biased**: its DI check required the literal token `Store`, which is the
  vocabulary the L2/L3 notes prescribe — so memory arms (which copy `Store`) passed
  and cold (which chose `Repository`, a synonym) was scored "no DI", producing a
  false "cold never converges" headline. Name-agnostic re-scoring (detect the
  *pattern*: any persistence interface + injection) put cold at 9/10. A scorer that
  keys on vocabulary drawn from the thing-under-test systematically inflates the
  memory arms; all results here use the corrected name-agnostic scorer, but the
  episode predicts other latent name/vocab biases are possible.
- **Rubric = notes circularity → memory-vs-review-channel, not emergent transfer.**
  The conventions rubric is drawn from the same opinionated content the notes encode,
  so a passing memory arm shows that the requirement reached the agent through the
  recall channel instead of the human-review channel — it is **not** evidence the
  agent generalized a convention it had never been told. Real transfer (a different,
  conventions-sharing app whose *needs* match the accumulated lessons) remains the
  next experiment.
- **Accumulation test is structurally unable to show a win.** By design the 2nd
  source (bookmarks) is off-domain for the targets (contacts), so the accumulation
  sweep can at best show "no harm." A fair accumulation test needs a target whose
  needs match the accumulated lessons (e.g. an app requiring side-effect injection,
  where the bookmarks-enriched L3 would be on-domain).
- **Finding (d) confound.** The tier-isolated-vs-blended gap also varies vault
  provenance (fresh-learned-from-cold vs hand-curated). Treat it as a comparison of
  recall *regimes*, not an isolated recall-strategy result.

---

Durable artifacts (scripts reproduce every table above):
- Convergence: `/Users/joe/repos/personal/engram/dev/eval/.layer-run/extract_metrics.py`
- +bookmarks sweep: `/tmp/accum_sweep.py` over `/Users/joe/repos/personal/engram/dev/eval/.layer-run/` data
- Fresh-learn chain: `/Users/joe/repos/personal/engram/dev/eval/.layer-run/extract_chain_metrics.py`
  over `chain/<L*>-s<0|1|2>-<app>.{json,workspace}`
- Prior program: `/Users/joe/repos/personal/engram/docs/superpowers/specs/2026-05-30-cold-warm-todo-test.md`
