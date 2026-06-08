# Cumulative cross-app memory accumulation — results (v2)

Engram SHA: `fd8cc13c1b07` · date: 2026-06-07 · models: sonnet, haiku, opus · trials: [1, 2, 3, 4, 5] · price sheet: 2026-06-02

> A NEW clean baseline (re-metric'd say-once + 7 vs 5 regimes); NOT comparable cell-for-cell to the 2026-06-02 run.

## Primary — repeated-convention interventions (say-once vs every-app)

Chain-summed conventions the human had to STATE (app1+app2+app3). Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.

### Convention interventions to endpoint (mean/trial)

| regime | sonnet | haiku | opus |
|---|---:|---:|---:|
| `cold` | 18.8 | 18.2 | 19.8 |
| `l1` | 9.6 | 13.8 | 8.8 |
| `l2.l1l2` | 9.6 | 12.8 | 8.6 |
| `l2.l2` | 9.8 | 12.2 | 10.4 |
| `l3.l1l2l3` | 9.8 | 12.6 | 9.4 |
| `l3.l2l3` | 9.4 | 12.0 | 8.8 |
| `l3.l3` | 10.0 | 13.8 | 9.2 |

### Feature interventions — CONTROL (app-specific; nobody carries these)

| regime | sonnet | haiku | opus |
|---|---:|---:|---:|
| `cold` | 7.4 | 10.0 | 8.8 |
| `l1` | 5.6 | 10.2 | 6.0 |
| `l2.l1l2` | 7.6 | 10.2 | 6.0 |
| `l2.l2` | 7.8 | 11.0 | 6.0 |
| `l3.l1l2l3` | 6.4 | 9.0 | 6.0 |
| `l3.l2l3` | 6.6 | 11.8 | 8.2 |
| `l3.l3` | 7.4 | 10.0 | 7.6 |

### Headline — memory cuts CONVENTION restatement far more than FEATURE restatement

- **sonnet**: memory removes **48%** of the cold convention-restatement burden vs **7%** of the feature burden (a **42 pp** convention–feature gap).
- **haiku**: memory removes **29%** of the cold convention-restatement burden vs **-4%** of the feature burden (a **33 pp** convention–feature gap).
- **opus**: memory removes **54%** of the cold convention-restatement burden vs **25%** of the feature burden (a **29 pp** convention–feature gap) — it cuts convention restatement **2.2×** as deeply.

The transferable-vs-app-specific GAP is the signal. The feature side is not a pure control — feeds shares α/β with the priors, so memory transfer leaks in (and for haiku the noisy feature side even moves the wrong way); the leak-free check is the native-only control below.

**Cross-model: memory is a capability AMPLIFIER, not an equalizer.** The convention reduction grows with model strength (see per-model % above) — memory helps the stronger model more, widening the capability gap, reproducing the 2026-06-02 finding.

### Headline stats — to the endpoint (notes→links→feeds chain, mean per trial)

`conv-restate` = convention restatements the human made (the say-once metric, lower=better). `review` = feedback rounds. **Memory's win is conv-restate; it does NOT reduce time/tokens/$ — recall + richer learn cost more.**

| model | arm | conv-restate | review | converged | wall min | tokens | $ |
|---|---|--:|--:|--:|--:|--:|--:|
| sonnet | cold | 18.8 | 3.2 | 80% | 33 | 3.8M | 4.05 |
| sonnet | warm | 9.7 | 3.3 | 77% | 57 | 15.1M | 9.13 |
| haiku | cold | 18.2 | 7.4 | 0% | 20 | 13.4M | 2.24 |
| haiku | warm | 12.9 | 7.6 | 7% | 24 | 17.8M | 2.79 |
| opus | cold | 19.8 | 3.8 | 60% | 19 | 2.9M | 4.71 |
| opus | warm | 9.2 | 3.1 | 97% | 24 | 6.2M | 7.44 |

## Secondary

### β-bucket on feeds, ROUND 1 /4 (front-loading: does links' memory lift β in the first draft? — measured at round 1; β saturates to 4/4 at convergence)

| regime | sonnet | haiku | opus |
|---|---:|---:|---:|
| `cold` | 2.40 | 1.80 | 1.80 |
| `l1` | 3.40 | 1.20 | 2.00 |
| `l2.l1l2` | 2.40 | 2.20 | 2.00 |
| `l2.l2` | 1.80 | 1.80 | 2.00 |
| `l3.l1l2l3` | 2.80 | 2.00 | 2.40 |
| `l3.l2l3` | 2.60 | 1.80 | 1.80 |
| `l3.l3` | 2.40 | 1.80 | 2.20 |

### Direct-vs-followed on tier-read regimes (mean link-following rate, feeds)

| regime | sonnet | haiku | opus |
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

| regime | sonnet | haiku | opus |
|---|---:|---:|---:|
| `cold` | 1.80 | 1.00 | 1.60 |
| `l1` | 1.60 | 1.20 | 2.00 |
| `l2.l1l2` | 1.80 | 0.80 | 2.00 |
| `l2.l2` | 1.80 | 0.60 | 2.00 |
| `l3.l1l2l3` | 2.00 | 1.60 | 2.00 |
| `l3.l2l3` | 2.00 | 0.60 | 1.60 |
| `l3.l3` | 1.60 | 1.00 | 2.00 |

### Cost & convergence by regime (mean per trial) — learn$ vs build$ split

`learn$` rises with write-tier (L1 episode < L2 +facts < L3 +synthesis); `build$` is dominated by feedback round-count (convergence), which is tier-insensitive — so total $ does not cleanly follow tier simplicity.

**sonnet**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `cold` | none | 0.00 | 4.05 | 4.05 | 33 | 3.8M | 80% |
| `l1` | L1 | 2.16 | 7.23 | 9.39 | 58 | 16.5M | 60% |
| `l2.l1l2` | L2 | 2.24 | 6.19 | 8.44 | 52 | 13.8M | 80% |
| `l2.l2` | L2 | 1.74 | 7.31 | 9.05 | 56 | 15.0M | 80% |
| `l3.l1l2l3` | L3 | 2.64 | 5.71 | 8.35 | 54 | 13.0M | 100% |
| `l3.l2l3` | L3 | 2.61 | 6.08 | 8.69 | 53 | 14.3M | 100% |
| `l3.l3` | L3 | 2.31 | 8.52 | 10.83 | 68 | 17.9M | 40% |

**haiku**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `cold` | none | 0.00 | 2.24 | 2.24 | 20 | 13.4M | 0% |
| `l1` | L1 | 0.11 | 2.41 | 2.52 | 21 | 15.9M | 20% |
| `l2.l1l2` | L2 | 0.16 | 2.70 | 2.85 | 24 | 18.4M | 0% |
| `l2.l2` | L2 | 0.16 | 2.94 | 3.09 | 25 | 20.2M | 0% |
| `l3.l1l2l3` | L3 | 0.19 | 2.57 | 2.76 | 23 | 17.5M | 20% |
| `l3.l2l3` | L3 | 0.19 | 2.55 | 2.74 | 26 | 17.3M | 0% |
| `l3.l3` | L3 | 0.19 | 2.55 | 2.74 | 24 | 17.2M | 0% |

**opus**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `cold` | none | 0.00 | 4.71 | 4.71 | 19 | 2.9M | 60% |
| `l1` | L1 | 0.73 | 6.29 | 7.02 | 24 | 5.7M | 100% |
| `l2.l1l2` | L2 | 0.86 | 6.09 | 6.94 | 22 | 5.6M | 100% |
| `l2.l2` | L2 | 0.86 | 5.72 | 6.58 | 22 | 5.3M | 100% |
| `l3.l1l2l3` | L3 | 1.76 | 7.56 | 9.32 | 28 | 8.5M | 100% |
| `l3.l2l3` | L3 | 1.79 | 5.64 | 7.43 | 26 | 5.9M | 80% |
| `l3.l3` | L3 | 1.84 | 5.50 | 7.34 | 25 | 5.9M | 100% |



### Learn-capture quality (did the agent persist what matters, per tier)

Cell = mean convention-coverage (captured/stated) · episode-extraction%. The agent runs its own /learn skill; an L1 episode must ALWAYS be extracted (the foundation every tier links down to), so episode% < 100 is a real learn failure.

| write-tier | sonnet | haiku | opus |
|---|---:|---:|---:|
| `L1` | 0.99 · ep 100% | 1.00 · ep 100% | 1.00 · ep 100% |
| `L2` | 1.00 · ep 100% | 1.00 · ep 100% | 1.00 · ep 100% |
| `L3` | 1.00 · ep 100% | 1.00 · ep 100% | 1.00 · ep 100% |


### Token I/O + cost audit (per model, over covered cells)  ·  **303/390 cells covered** (the rest lost their transcripts to cfg-pool re-creation across resumes — run-time token capture in the result JSON fixes this going forward)

Reconstructing $ from token counts × the price sheet reproduces the CLI's reported cost (ratio ≈ 1.00× over MATCHED cells — the §6 provenance check). Cost is cache-dominated.

| model | cells | input | output | cache-write | cache-read | reported $ | recomputed $ | ratio |
|---|--:|--:|--:|--:|--:|--:|--:|--:|
| sonnet | 120 | 6,616 | 5,891,689 | 11,299,415 | 397,917,081 | 248.66 | 250.14 | 1.01× |
| haiku | 90 | 48,676 | 3,438,752 | 6,514,437 | 443,332,130 | 69.80 | 69.72 | 1.00× |
| opus | 93 | 319,363 | 2,885,887 | 5,892,808 | 149,002,455 | 185.16 | 185.08 | 1.00× |


### Cost calibration (per-operation; grounds the full-run estimate)

| op | model | app | n | mean $ | mean rounds |
|---|---|---|--:|--:|--:|
| build | haiku | feeds | 35 | 1.01 | 3.9 |
| build | haiku | links | 35 | 0.81 | 2.7 |
| build | haiku | notes | 5 | 0.74 | 4.0 |
| build | opus | feeds | 35 | 2.35 | 2.1 |
| build | opus | links | 35 | 2.13 | 2.1 |
| build | opus | notes | 5 | 1.44 | 2.0 |
| build | sonnet | feeds | 35 | 3.12 | 2.2 |
| build | sonnet | links | 35 | 2.31 | 2.1 |
| build | sonnet | notes | 5 | 1.01 | 2.0 |
| learn | haiku | links | 30 | 0.00 | — |
| learn | haiku | notes | 15 | 0.15 | — |
| learn | opus | links | 30 | 0.20 | — |
| learn | opus | notes | 15 | 0.99 | — |
| learn | sonnet | links | 30 | 1.33 | — |
| learn | sonnet | notes | 15 | 0.91 | — |


## Convergence guard + honest caveats

- **Converged within the 6-round budget: 168/225 builds.** The primary metric is the round-1 intervention count, not a stall rate; low convergence means some builds plateau below the full bar — investigate feedback-symptom effectiveness / stale-break, separately from say-once.
- **n=5 trial(s).** Models: sonnet, haiku, opus.
- **Regime axis (the v2 question): tier is FLAT — does not matter at n=5, every model.** Per model: sonnet 9.4–10.0 band vs cold 18.8 (best: l3.l2l3); haiku 12.0–13.8 band vs cold 18.2 (best: l3.l2l3); opus 8.6–10.4 band vs cold 19.8 (best: l2.l1l2). Within each model the warm regimes cluster well inside the cold→warm gap — writing L3 syntheses does not beat L1 episodes, reading only the distilled L3 does not beat blended, and raw L1 episodes capture the full effect. β-accumulation (round-1 feeds β) saturates to 4/4 by convergence and is noisy in the first draft, so H2 stays inconclusive at this β-difficulty.
- Learn is agent-driven; learn-capture coverage + episode-extraction above are measured outputs (a poor capture is recorded, not engineered away).
- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline.