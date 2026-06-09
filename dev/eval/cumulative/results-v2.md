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
| haiku | warm | 12.7 | 13.3 | 50% | 36 | 32.4M | 4.79 |
| sonnet | cold | 18.8 | 3.2 | 100% | 31 | 3.2M | 3.67 |
| sonnet | warm | 8.5 | 3.0 | 100% | 48 | 16.6M | 10.12 |
| opus | cold | 17.0 | 4.4 | 100% | 21 | 4.0M | 5.89 |
| opus | warm | 7.1 | 3.2 | 100% | 24 | 8.4M | 9.63 |

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
| `l1` | L1 | 1.86 | 3.88 | 5.74 | 39 | 39.7M | 20% |
| `l2.l1l2` | L2 | 0.97 | 3.00 | 3.98 | 33 | 25.4M | 60% |
| `l2.l2` | L2 | 1.49 | 2.67 | 4.16 | 31 | 27.7M | 80% |
| `l3.l1l2l3` | L3 | 1.71 | 3.78 | 5.49 | 40 | 38.4M | 20% |
| `l3.l2l3` | L3 | 1.16 | 3.01 | 4.17 | 37 | 27.5M | 80% |
| `l3.l3` | L3 | 1.54 | 3.65 | 5.19 | 37 | 35.4M | 40% |

**sonnet**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `cold` | none | 0.00 | 3.67 | 3.67 | 31 | 3.2M | 100% |
| `l1` | L1 | 2.58 | 6.34 | 8.92 | 46 | 14.5M | 100% |
| `l2.l1l2` | L2 | 3.46 | 6.09 | 9.55 | 44 | 16.7M | 100% |
| `l2.l2` | L2 | 4.41 | 7.52 | 11.93 | 51 | 20.6M | 100% |
| `l3.l1l2l3` | L3 | 3.90 | 5.98 | 9.87 | 46 | 16.1M | 100% |
| `l3.l2l3` | L3 | 3.29 | 6.24 | 9.53 | 49 | 14.3M | 100% |
| `l3.l3` | L3 | 3.97 | 6.96 | 10.93 | 52 | 17.5M | 100% |

**opus**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `cold` | none | 0.00 | 5.89 | 5.89 | 21 | 4.0M | 100% |
| `l1` | L1 | 3.25 | 6.39 | 9.64 | 24 | 8.4M | 100% |
| `l2.l1l2` | L2 | 2.95 | 6.12 | 9.08 | 21 | 7.9M | 100% |
| `l2.l2` | L2 | 3.00 | 6.20 | 9.20 | 23 | 7.9M | 100% |
| `l3.l1l2l3` | L3 | 4.03 | 7.28 | 11.31 | 26 | 10.3M | 100% |
| `l3.l2l3` | L3 | 3.34 | 5.97 | 9.31 | 25 | 8.0M | 100% |
| `l3.l3` | L3 | 3.42 | 5.80 | 9.22 | 24 | 7.6M | 100% |



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


## Full matrix (model × regime × app, medians)

### Full matrix — app1 · notes (cold build shared per model; row = write-tier of its learn)

Medians. **Bold** = best (lowest) per model per metric. app1 build is identical across rows; only learn cost/tokens/time differ by tier.

| model | write-tier | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| haiku | `none` | **3** | **2** | **4** | **0.88** | **5.9M** | **8** |
| haiku | `L1` | 3 | 2 | 4 | 1.05 | 6.5M | 9 |
| haiku | `L2` | 3 | 2 | 4 | 1.10 | 6.6M | 10 |
| haiku | `L3` | 3 | 2 | 4 | 1.09 | 6.7M | 10 |
| sonnet | `none` | **1** | **1** | **2** | **0.91** | **0.7M** | **7** |
| sonnet | `L1` | 1 | 1 | 2 | 1.55 | 2.0M | 11 |
| sonnet | `L2` | 1 | 1 | 2 | 2.03 | 3.4M | 14 |
| sonnet | `L3` | 1 | 1 | 2 | 1.95 | 3.0M | 12 |
| opus | `none` | **1** | **1** | **2** | **1.38** | **1.1M** | **6** |
| opus | `L1` | 1 | 1 | 2 | 2.22 | 1.4M | 8 |
| opus | `L2` | 1 | 1 | 2 | 2.18 | 1.6M | 8 |
| opus | `L3` | 1 | 1 | 2 | 3.21 | 2.5M | 10 |

### Full matrix — app2 · links (recall under regime)

Medians. **Bold** = best (lowest) per model per metric. † = <60% of this cell's builds completed (resource figures include capped runs).

| model | regime | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| haiku | `cold`† | 14 | 2 | 3 | 1.94 | 14.1M | 15 |
| haiku | `l1`† | 14 | 2 | 2 | 4.37 | 31.3M | 17 |
| haiku | `l2.l1l2` | **1** | **1** | **2** | 1.37 | **8.1M** | **6** |
| haiku | `l2.l2` | 2 | 1 | 2 | 1.49 | 9.9M | 6 |
| haiku | `l3.l1l2l3`† | 14 | 1 | 2 | 3.48 | 24.6M | 14 |
| haiku | `l3.l2l3` | 2 | 2 | 2 | **1.36** | 9.4M | 10 |
| haiku | `l3.l3` | 2 | 2 | 2 | 1.48 | 9.3M | 7 |
| sonnet | `cold` | **1** | **1** | **2** | **1.14** | **0.8M** | **9** |
| sonnet | `l1` | 1 | 1 | 2 | 3.27 | 5.5M | 14 |
| sonnet | `l2.l1l2` | 1 | 1 | 2 | 4.27 | 7.5M | 12 |
| sonnet | `l2.l2` | 1 | 1 | 2 | 6.63 | 11.7M | 19 |
| sonnet | `l3.l1l2l3` | 1 | 1 | 2 | 5.17 | 8.5M | 15 |
| sonnet | `l3.l2l3` | 1 | 1 | 2 | 3.35 | 3.4M | 13 |
| sonnet | `l3.l3` | 1 | 1 | 2 | 4.77 | 8.1M | 14 |
| opus | `cold` | **1** | 1 | **2** | **2.04** | **1.5M** | 7 |
| opus | `l1` | 1 | **0** | 2 | 4.93 | 4.8M | 9 |
| opus | `l2.l1l2` | 1 | 0 | 2 | 4.76 | 4.7M | 7 |
| opus | `l2.l2` | 1 | 0 | 2 | 4.58 | 4.1M | 7 |
| opus | `l3.l1l2l3` | 1 | 0 | 2 | 5.44 | 5.3M | 8 |
| opus | `l3.l2l3` | 1 | 0 | 2 | 3.84 | 3.4M | **7** |
| opus | `l3.l3` | 1 | 0 | 2 | 4.10 | 3.7M | 7 |

### Full matrix — app3 · feeds (recall under regime; terminal, no learn)

Medians. **Bold** = best (lowest) per model per metric. † = <60% of this cell's builds completed (resource figures include capped runs).

| model | regime | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| haiku | `cold`† | 14 | 2 | 4 | 1.83 | 11.7M | 18 |
| haiku | `l1` | 3 | 2 | 3 | 1.10 | 7.4M | 9 |
| haiku | `l2.l1l2` | 3 | **0** | **2** | 1.00 | 6.6M | 7 |
| haiku | `l2.l2` | **2** | 1 | 3 | **0.58** | **3.4M** | **6** |
| haiku | `l3.l1l2l3` | 3 | 2 | 4 | 1.12 | 7.7M | 8 |
| haiku | `l3.l2l3` | 3 | 2 | 4 | 0.88 | 5.5M | 8 |
| haiku | `l3.l3` | 3 | 2 | 4 | 1.41 | 10.2M | 11 |
| sonnet | `cold` | **1** | **1** | **2** | **1.28** | **1.3M** | **10** |
| sonnet | `l1` | 1 | 1 | 2 | 3.58 | 6.2M | 22 |
| sonnet | `l2.l1l2` | 1 | 1 | 2 | 3.02 | 5.3M | 17 |
| sonnet | `l2.l2` | 1 | 1 | 2 | 3.57 | 5.6M | 20 |
| sonnet | `l3.l1l2l3` | 1 | 1 | 2 | 2.21 | 3.1M | 16 |
| sonnet | `l3.l2l3` | 1 | 1 | 2 | 2.91 | 5.2M | 18 |
| sonnet | `l3.l3` | 2 | 1 | 3 | 2.61 | 3.5M | 19 |
| opus | `cold` | **1** | 1 | **2** | **1.43** | **0.7M** | **6** |
| opus | `l1` | 1 | **0** | 2 | 2.24 | 2.0M | 7 |
| opus | `l2.l1l2` | 1 | 0 | 2 | 2.50 | 2.2M | 7 |
| opus | `l2.l2` | 1 | 0 | 2 | 2.24 | 2.3M | 6 |
| opus | `l3.l1l2l3` | 1 | 0 | 2 | 2.76 | 2.7M | 8 |
| opus | `l3.l2l3` | 1 | 0 | 2 | 2.18 | 2.0M | 8 |
| opus | `l3.l3` | 1 | 0 | 2 | 2.02 | 1.8M | 6 |



### Token I/O + cost audit (per model, over covered cells)  ·  360/360 LLM-using cells captured (30 cold no-op learns excluded)

Reconstructing $ from token counts × the price sheet reproduces the CLI's reported cost (ratio ≈ 1.00× over MATCHED cells — the §6 provenance check). Cost is cache-dominated.

| model | cells | input | output | cache-write | cache-read | reported $ | recomputed $ | ratio |
|---|--:|--:|--:|--:|--:|--:|--:|--:|
| haiku | 120 | 87,475 | 5,994,205 | 11,188,069 | 922,557,895 | 136.62 | 136.30 | 1.00× |
| sonnet | 120 | 9,010 | 6,750,231 | 12,562,658 | 439,218,905 | 279.62 | 280.16 | 1.00× |
| opus | 120 | 497,608 | 3,700,185 | 7,959,062 | 210,975,434 | 250.30 | 250.22 | 1.00× |


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
| learn | haiku | links | 30 | 1.27 | — |
| learn | haiku | notes | 15 | 0.17 | — |
| learn | opus | links | 30 | 2.25 | — |
| learn | opus | notes | 15 | 0.97 | — |
| learn | sonnet | links | 30 | 2.45 | — |
| learn | sonnet | notes | 15 | 1.06 | — |


## Convergence guard + honest caveats

- **Converged within the 15-round budget: 201/225 builds.** The primary metric is the round-1 intervention count, not a stall rate; low convergence means some builds plateau below the full bar — investigate feedback-symptom effectiveness / stale-break, separately from say-once.
- **n=5 trial(s).** Models: haiku, sonnet, opus.
- **Regime axis (the v2 question): tier is NOT uniformly flat at n=5, every model.** Per model: haiku 11.0–14.0 band vs cold 18.0 (best: l3.l1l2l3); sonnet 7.4–10.6 band vs cold 18.8 (best: l3.l1l2l3); opus 7.0–7.4 band vs cold 17.0 (best: l1). At least one model shows a between-tier spread comparable to its cold→warm gap — see the per-model bands. β-accumulation (round-1 feeds β) saturates to 4/4 by convergence and is noisy in the first draft, so H2 stays inconclusive at this β-difficulty.
- Learn is agent-driven; learn-capture coverage + episode-extraction above are measured outputs (a poor capture is recorded, not engineered away).
- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline.