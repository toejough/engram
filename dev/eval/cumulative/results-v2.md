# Cumulative cross-app memory accumulation — results (v2)

Engram SHA: `b57bb0c82da2` · date: 2026-06-08 · models: haiku, sonnet, opus · trials: [1, 2, 3, 4, 5] · price sheet: 2026-06-02

> A NEW clean baseline (re-metric'd say-once + 7 vs 5 regimes); NOT comparable cell-for-cell to the 2026-06-02 run.

## Primary — repeated-convention interventions (say-once vs every-app)

Chain-summed conventions the human had to STATE (app1+app2+app3). Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.

### Convention interventions to endpoint (mean/trial)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `cold` | 18.0 | 18.8 | 17.0 |
| `l1` | 14.0 | 8.6 | 7.0 |
| `l2.l1l2` | 11.2 | 8.0 | 7.0 |
| `l2.l2` | 12.4 | 7.6 | 7.0 |
| `l3.l1l2l3` | 11.0 | 7.4 | 7.0 |
| `l3.l2l3` | 13.4 | 9.0 | 7.4 |
| `l3.l3` | 14.0 | 10.6 | 7.4 |

### Feature interventions — CONTROL (app-specific; nobody carries these)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `cold` | 11.6 | 7.6 | 10.0 |
| `l1` | 12.2 | 7.2 | 7.2 |
| `l2.l1l2` | 9.0 | 7.2 | 7.2 |
| `l2.l2` | 9.6 | 7.2 | 6.8 |
| `l3.l1l2l3` | 12.2 | 7.6 | 7.4 |
| `l3.l2l3` | 9.2 | 7.4 | 7.0 |
| `l3.l3` | 11.0 | 8.2 | 7.6 |

### Headline — memory cuts CONVENTION restatement far more than FEATURE restatement

- **haiku**: memory removes **30%** of the cold convention-restatement burden vs **9%** of the feature burden (a **20 pp** convention–feature gap).
- **sonnet**: memory removes **55%** of the cold convention-restatement burden vs **2%** of the feature burden (a **53 pp** convention–feature gap).
- **opus**: memory removes **58%** of the cold convention-restatement burden vs **28%** of the feature burden (a **30 pp** convention–feature gap) — it cuts convention restatement **2.1×** as deeply.

The transferable-vs-app-specific GAP is the signal. The feature side is not a pure control — feeds shares α/β with the priors, so memory transfer leaks in (and for haiku the noisy feature side even moves the wrong way); the leak-free check is the native-only control below.

**Cross-model: memory is a capability AMPLIFIER, not an equalizer.** The convention reduction grows with model strength (see per-model % above) — memory helps the stronger model more, widening the capability gap, reproducing the 2026-06-02 finding.

### Headline stats — to the endpoint (notes→links→feeds chain, mean per trial)

`conv-restate` = convention restatements the human made (the say-once metric, lower=better). `review` = feedback rounds. **Memory's win is conv-restate; it does NOT reduce time/tokens/$ — recall + richer learn cost more.**

| model | arm | conv-restate | review | converged | wall min | tokens | $ |
|---|---|--:|--:|--:|--:|--:|--:|
| haiku | cold | 18.0 | 23.6 | 0% | 44 | 32.0M | 4.61 |
| haiku | warm | 12.7 | 13.3 | 50% | 36 | 23.5M | 3.52 |
| sonnet | cold | 18.8 | 3.2 | 100% | 31 | 3.2M | 3.67 |
| sonnet | warm | 8.5 | 3.0 | 100% | 48 | 12.6M | 7.70 |
| opus | cold | 17.0 | 4.4 | 100% | 21 | 4.0M | 5.89 |
| opus | warm | 7.1 | 3.2 | 100% | 24 | 6.3M | 7.38 |

## Secondary

### β-bucket on feeds, ROUND 1 /4 (front-loading: does links' memory lift β in the first draft? — measured at round 1; β saturates to 4/4 at convergence)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `cold` | 1.80 | 2.00 | 1.60 |
| `l1` | 1.60 | 2.20 | 2.00 |
| `l2.l1l2` | 2.40 | 1.80 | 2.00 |
| `l2.l2` | 1.80 | 2.00 | 2.00 |
| `l3.l1l2l3` | 1.60 | 2.00 | 2.00 |
| `l3.l2l3` | 2.40 | 2.00 | 2.00 |
| `l3.l3` | 1.60 | 2.00 | 2.00 |

### Direct-vs-followed on tier-read regimes (mean link-following rate, feeds)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `cold` | 0.00 | 0.00 | 0.00 |
| `l1` | 1.00 | 1.00 | 1.00 |
| `l2.l1l2` | 0.00 | 0.00 | 0.00 |
| `l2.l2` | 1.00 | 1.00 | 1.00 |
| `l3.l1l2l3` | 0.00 | 0.00 | 0.00 |
| `l3.l2l3` | 1.00 | 1.00 | 1.00 |
| `l3.l3` | 1.00 | 1.00 | 1.00 |


feeds round-1 NATIVE-bucket pass count (the feed-specific features no prior app teaches). If memory is a clean say-once mechanism this should NOT rise with memory; if it does, memory is also lifting first-draft quality generally (a real effect, but it means 'feature interventions' is not a pure untouched control).
### Native-only control on feeds (leak-free: no shared α/β)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `cold` | 1.40 | 2.00 | 2.00 |
| `l1` | 1.20 | 2.00 | 2.00 |
| `l2.l1l2` | 1.40 | 2.00 | 2.00 |
| `l2.l2` | 2.00 | 2.00 | 2.00 |
| `l3.l1l2l3` | 1.60 | 2.00 | 2.00 |
| `l3.l2l3` | 2.00 | 2.00 | 2.00 |
| `l3.l3` | 1.60 | 2.00 | 2.00 |

### Cost & convergence by regime (mean per trial) — learn$ vs build$ split

`learn$` rises with write-tier (L1 episode < L2 +facts < L3 +synthesis); `build$` is dominated by feedback round-count (convergence), which is tier-insensitive — so total $ does not cleanly follow tier simplicity.

**haiku**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `cold` | none | 0.00 | 4.61 | 4.61 | 44 | 32.0M | 0% |
| `l1` | L1 | 0.14 | 3.88 | 4.01 | 39 | 27.3M | 20% |
| `l2.l1l2` | L2 | 0.18 | 3.00 | 3.18 | 33 | 20.2M | 60% |
| `l2.l2` | L2 | 0.18 | 2.67 | 2.85 | 31 | 18.6M | 80% |
| `l3.l1l2l3` | L3 | 0.20 | 3.78 | 3.98 | 40 | 27.5M | 20% |
| `l3.l2l3` | L3 | 0.24 | 3.01 | 3.25 | 37 | 21.3M | 80% |
| `l3.l3` | L3 | 0.20 | 3.65 | 3.85 | 37 | 26.2M | 40% |

**sonnet**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `cold` | none | 0.00 | 3.67 | 3.67 | 31 | 3.2M | 100% |
| `l1` | L1 | 0.89 | 6.34 | 7.23 | 46 | 11.8M | 100% |
| `l2.l1l2` | L2 | 1.23 | 6.09 | 7.31 | 44 | 12.9M | 100% |
| `l2.l2` | L2 | 1.23 | 7.52 | 8.74 | 51 | 15.1M | 100% |
| `l3.l1l2l3` | L3 | 1.24 | 5.98 | 7.22 | 46 | 11.6M | 100% |
| `l3.l2l3` | L3 | 1.24 | 6.24 | 7.48 | 49 | 11.5M | 100% |
| `l3.l3` | L3 | 1.24 | 6.96 | 8.20 | 52 | 12.8M | 100% |

**opus**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `cold` | none | 0.00 | 5.89 | 5.89 | 21 | 4.0M | 100% |
| `l1` | L1 | 0.74 | 6.39 | 7.12 | 24 | 6.0M | 100% |
| `l2.l1l2` | L2 | 0.80 | 6.12 | 6.92 | 21 | 5.9M | 100% |
| `l2.l2` | L2 | 0.80 | 6.20 | 7.00 | 23 | 5.9M | 100% |
| `l3.l1l2l3` | L3 | 1.39 | 7.28 | 8.67 | 26 | 7.8M | 100% |
| `l3.l2l3` | L3 | 1.39 | 5.97 | 7.36 | 25 | 6.2M | 100% |
| `l3.l3` | L3 | 1.39 | 5.80 | 7.19 | 24 | 5.9M | 100% |



### Learn-capture quality (did the agent persist what matters, per tier)

Cell = mean convention-coverage (captured/stated) · episode-extraction%. The agent runs its own /learn skill; an L1 episode must ALWAYS be extracted (the foundation every tier links down to), so episode% < 100 is a real learn failure.

| write-tier | haiku | sonnet | opus |
|---|---:|---:|---:|
| `L1` | 0.99 · ep 100% | 0.83 · ep 100% | 0.92 · ep 100% |
| `L2` | 0.99 · ep 100% | 0.85 · ep 100% | 0.79 · ep 100% |
| `L3` | 1.00 · ep 100% | 0.92 · ep 100% | 1.00 · ep 100% |


### Feedback escalation depth — how granular before convergence (completed builds)

`conv-depth` = median max times a *convention* was restated before it stuck (1 = fixed on the symptom; ≥2 = needed the literal code-level prescription). `#presc` = mean conventions per build that needed the prescriptive fix. Higher = more hand-holding — expected to fall as model strength rises.

| model | app | conv-depth (median) | #presc (mean) |
|---|---|--:|--:|
| haiku | notes | 1.8 | 2.0 |
| haiku | links | 1.3 | 1.0 |
| haiku | feeds | 1.6 | 1.3 |
| sonnet | notes | 0.8 | 0.0 |
| sonnet | links | 0.8 | 0.0 |
| sonnet | feeds | 0.9 | 0.1 |
| opus | notes | 1.0 | 0.0 |
| opus | links | 0.4 | 0.0 |
| opus | feeds | 0.3 | 0.0 |


### Token I/O + cost audit (per model, over covered cells)  ·  **272/390 cells covered** (the rest lost their transcripts to cfg-pool re-creation across resumes — run-time token capture in the result JSON fixes this going forward)

Reconstructing $ from token counts × the price sheet reproduces the CLI's reported cost (ratio ≈ 1.00× over MATCHED cells — the §6 provenance check). Cost is cache-dominated.

| model | cells | input | output | cache-write | cache-read | reported $ | recomputed $ | ratio |
|---|--:|--:|--:|--:|--:|--:|--:|--:|
| haiku | 91 | 63,686 | 4,373,111 | 8,094,607 | 662,360,590 | 98.60 | 98.28 | 1.00× |
| sonnet | 91 | 6,269 | 5,023,000 | 9,318,455 | 323,958,279 | 206.96 | 207.50 | 1.00× |
| opus | 90 | 368,918 | 2,718,654 | 5,912,369 | 151,975,358 | 182.83 | 182.75 | 1.00× |


### Cost calibration (per-operation; grounds the full-run estimate)

| op | model | app | n | mean $ | mean rounds |
|---|---|---|--:|--:|--:|
| build | haiku | feeds | 35 | 1.32 | 6.7 |
| build | haiku | links | 35 | 1.29 | 7.4 |
| build | haiku | notes | 5 | 0.91 | 3.6 |
| build | opus | feeds | 35 | 2.35 | 2.1 |
| build | opus | links | 35 | 2.22 | 2.1 |
| build | opus | notes | 5 | 1.67 | 2.2 |
| build | sonnet | feeds | 35 | 3.01 | 2.3 |
| build | sonnet | links | 35 | 2.31 | 2.0 |
| build | sonnet | notes | 5 | 0.79 | 1.8 |
| learn | haiku | links | 30 | 0.01 | — |
| learn | haiku | notes | 15 | 0.17 | — |
| learn | opus | links | 30 | 0.00 | — |
| learn | opus | notes | 15 | 0.97 | — |
| learn | sonnet | links | 30 | 0.03 | — |
| learn | sonnet | notes | 15 | 1.06 | — |


## Convergence guard + honest caveats

- **Converged within the 15-round budget: 201/225 builds.** The primary metric is the round-1 intervention count, not a stall rate; low convergence means some builds plateau below the full bar — investigate feedback-symptom effectiveness / stale-break, separately from say-once.
- **n=5 trial(s).** Models: haiku, sonnet, opus.
- **Regime axis (the v2 question): tier is NOT uniformly flat at n=5, every model.** Per model: haiku 11.0–14.0 band vs cold 18.0 (best: l3.l1l2l3); sonnet 7.4–10.6 band vs cold 18.8 (best: l3.l1l2l3); opus 7.0–7.4 band vs cold 17.0 (best: l1). At least one model shows a between-tier spread comparable to its cold→warm gap — see the per-model bands. β-accumulation (round-1 feeds β) saturates to 4/4 by convergence and is noisy in the first draft, so H2 stays inconclusive at this β-difficulty.
- Learn is agent-driven; learn-capture coverage + episode-extraction above are measured outputs (a poor capture is recorded, not engineered away).
- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline.