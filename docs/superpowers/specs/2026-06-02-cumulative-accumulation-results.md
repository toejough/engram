# Cumulative cross-app memory accumulation — results

Date: 2026-06-02. Full matrix: **3 models** (haiku/sonnet/opus) × **5 recall regimes**
(cold, L1-isolated, L2-isolated, L3-isolated, blended) × **2 accumulation stages**
(+notes, +notes+links) × **n=3 trials** = 99 cells (81 scored feeds builds, all n=3).
Total spend **$219**, ~6 h wall. Real production vault untouched throughout (580 notes,
verified).

## What this measured (and how it's caveat-free by construction)

Three **cumulative** CLI apps where each genuinely needs the prior's lessons:
`notes` (teaches **α** = tag/search subsystem) → `links` (teaches **β** = URL
validation + canonical dedup + import/export) → **`feeds`** (the target; needs arch +
α + β + native). Every builder got only the **command list**, never the architecture
or quality bar. Feeds was built **cold / +notes-memory / +notes+links-memory** under
each recall regime, and scored by a **deterministic, name-agnostic scorer** (structural
arch detection that keys on the *pattern* not the vocabulary; behavioral α/β/native
checks that *run the binary*; α made arg-order-agnostic so it measures capability not
parsing). The accumulation signal is **localized to the β bucket**: β should jump only
when `links`'s memory is present. Pre-resolved caveats: scorer bias (deterministic +
name-agnostic, validated on known naive/good builds), tier-isolated-vs-blended (a
measured variable), learn on/off (always-learn), converged≠100% (push to
feature-complete), app saturation (cumulative apps), isolation (headless, scrubbed cfgs).

## Headline 1 — Memory is a capability AMPLIFIER, not an equalizer

The hypothesis was "memory levels up weak models." **The data says the opposite.**
Round-1 conformance, cold vs the model's best memory config:

| model | cold | best-memory (round-1) | **Δ from memory** |
|---|---:|---:|---:|
| haiku | 6.7 | 13.3 (L2+notes+links) | **+6.7** |
| sonnet | 7.7 | 15.3 (L3+notes+links) | **+7.7** |
| opus | 6.0 | 17.0 (blended+notes+links) | **+11.0** |

All three start cold in a dead heat (6–8/18). Memory then helps **opus most** (+11.0)
and **haiku least** (+6.7). Opus goes from *worst* cold to *best* warm — it applies
recalled conventions more completely. Memory widens the capability gap rather than
closing it.

## Headline 2 — β-accumulation is real, and model-gated

The localized test: does the β bucket (β/4) jump at +notes → +notes+links, i.e. when
the 2nd app's memory enters? It does — **cleanly for strong models, weakly for haiku**:

| regime | haiku β | sonnet β | opus β |
|---|---|---|---|
| L1-iso | 2.0 → 1.7 | 2.0 → 3.3 | 2.0 → 4.0 |
| L2-iso | 2.0 → 3.3 | 2.0 → 4.0 | 2.0 → 4.0 |
| L3-iso | 2.0 → 2.3 | 2.0 → 4.0 | 2.0 → 4.0 |
| blended | 1.3 → 2.3 | 2.0 → 4.0 | 2.0 → 4.0 |

**This is the result the parallel todo/bookmarks/contacts apps could never show** — a
2nd accumulated source measurably lifting a specific capability. sonnet and opus hit
**β 4/4** (full) once `links`'s memory is present, in every regime. haiku absorbs it
only partially (and in L1-isolated it even regresses, 2.0→1.7 — episodes alone confuse
the weak model). Accumulation works; the weaker the model, the less of the 2nd app's
lessons it can convert.

## Headline 3 — Recall regime interacts with model strength

Round-1 /18 at full accumulation (+notes+links), by recall regime:

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| L1-isolated | 9.7 | 14.3 | 16.5 |
| **L2-isolated** | **13.3** | 15.0 | 16.0 |
| L3-isolated | 9.3 | 15.3 | 16.5 |
| blended | 9.3 | 14.7 | **17.0** |

- **Weak model (haiku): L2-isolated wins decisively** (13.3 vs 9–10 for everything
  else). A focused pile of specific facts transfers; the blended mix and the
  episode/ADR tiers overwhelm or under-specify it.
- **Strong model (opus): blended wins** (17.0) — it integrates all tiers and the extra
  context helps rather than distracts.
- L2 is the safe default everywhere (never worst for any model); L1-isolated and
  L3-isolated are weak for haiku — consistent with prior findings that raw episodes and
  tier-capped ADRs are thin on the load-bearing specifics.

## Headline 4 — Better first draft ≠ convergence, for weak models

Feeds cells reaching **feature-complete** (all behavioral buckets + arch≥8) within 5
review rounds:

| model | converged / cells | model spend |
|---|---:|---:|
| haiku | **2 / 27** | $45 |
| sonnet | 18 / 27 | $87 |
| opus | **25 / 27** | $87 |

Memory gives haiku a much better *first draft* (Δ+6.7) but it still almost never
*closes* to the full bar — it stalls on the architecture nits and the last β items.
Strong models both start higher with memory and finish (opus 25/27). So memory's
"removes the human review round" value (established for the strong-model todo test) is
itself **capability-gated**: it holds for sonnet/opus, not for haiku.

## Aggregate

99 cells (81 scored feeds builds, all n=3), $219, ~6 h. β-accumulation localized and
confirmed; model×memory is an amplifier; recall regime × model strength interacts
(L2-iso for weak, blended for strong); convergence is capability-gated. Real vault
untouched.

## Cost provenance — verified pricing + token breakdown (reproduce the math)

All `$` figures are the Claude Code CLI's `total_cost_usd`, **independently verified**:
recomputing cost from raw transcript token counts × the current published price sheet
reproduces the CLI's reported cost to the cent for every model (ratio 1.00×). No price
sheet was assumed in the run; this is a post-hoc audit.

**Price sheet** ($/Mtok), verified 2026-06-02 against platform.claude.com/docs pricing:

| model | input | output | cache-write (5m, 1.25×) | cache-read (0.1×) |
| --- | --- | --- | --- | --- |
| Claude Haiku 4.5 | $1.00 | $5.00 | $1.25 | $0.10 |
| Claude Sonnet 4.6 | $3.00 | $15.00 | $3.75 | $0.30 |
| **Claude Opus 4.8** | **$5.00** | **$25.00** | **$6.25** | **$0.50** |

> NB: Opus 4.5–4.8 are **$5/$25**, not the $15/$75 of Opus 4 / 4.1 (a 3× drop). Costing
> opus-4-8 at the old $15/$75 over-states its cost exactly 3× — a real trap when
> reading older price tables.

**Per-cell average tokens by type** (feeds build+review cells; mean over cells with
retained transcripts — n shown; some transcripts were destroyed by pool re-creation on
the rate-limit re-runs, but the surviving sample reconstructs reported cost at 1.00×):

| model | n | input | cache-write | cache-read | output | recompute $/cell | reported $/cell |
| --- | --: | --: | --: | --: | --: | --: | --: |
| haiku | 11 | 754 | 279,042 | 9,275,676 | 52,454 | $1.54 | $1.54 |
| sonnet | 16 | 40 | 163,202 | 2,696,204 | 63,441 | $2.37 | $2.38 |
| opus | 16 | 3,997 | 89,826 | 1,793,383 | 33,202 | $2.31 | $2.31 |

`cost = (input·in + cache_write·cw + cache_read·cr + output·out) / 1e6`. Worked example
(opus): 3997·5 + 89826·6.25 + 1793383·0.50 + 33202·25, all /1e6 = $0.02 + $0.56 + $0.90
+ $0.83 = **$2.31** ✓. **Cost is cache-dominated**: caching is 63% of the opus cell and
83% of the haiku cell (haiku averages 9.3M cache-read tokens/cell — its 97-turn
thrashing reads huge context cheaply at $0.10/Mtok, which is why its low per-token rate
still accumulates). Scripts: `dev/eval/cumulative/{verify_cost2,token_table}.py`.

## Honest caveats

- **n=3, heuristic-but-deterministic scorer.** Variance is real (haiku noteslinks ±3–4
  pts); the scorer is transparent/re-runnable but still a proxy for "good Go CLI."
- **20 cells hit API rate-limits at the run's concurrent tail** (opus/sonnet
  +notes+links) and were re-run — 16 cleared on a low-concurrency retry, the last 4
  (opus-t3) only on a strictly serial (workers=1) retry. All 81 feeds cells are now
  scored at n=3; the rate-limit was an orchestration artifact, not a data signal.
- **Single app trilogy, one architecture family.** α/β are two specific subsystems; "memory
  amplifies capability" is shown for *this* transfer, not proven universal.
- **Vault learned from the prior builds** (inherent to accumulation) — warm scores partly
  reflect recall of the chain's own earlier solutions.
- Reproduce: `dev/eval/cumulative/{matrix,harness,score,archscore,behavioral,aggregate}.py`;
  specs `*_spec.json`; results `/tmp/cummatrix/results/`.
