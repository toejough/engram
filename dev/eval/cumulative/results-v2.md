# Cumulative cross-app memory accumulation — results (v2)

Engram SHA: `d1e745f3a32e` · date: 2026-06-07 · models: sonnet · trials: [1, 2, 3, 4, 5] · price sheet: 2026-06-02

> A NEW clean baseline (re-metric'd say-once + 7 vs 5 regimes); NOT comparable cell-for-cell to the 2026-06-02 run.

## Primary — repeated-convention interventions (say-once vs every-app)

Chain-summed conventions the human had to STATE (app1+app2+app3). Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.

### Convention interventions to endpoint (mean/trial)

| regime | sonnet |
|---|---:|
| `cold` | 18.8 |
| `l1` | 9.6 |
| `l2.l1l2` | 9.6 |
| `l2.l2` | 9.8 |
| `l3.l1l2l3` | 9.8 |
| `l3.l2l3` | 9.4 |
| `l3.l3` | 10.0 |

### Feature interventions — CONTROL (app-specific; nobody carries these)

| regime | sonnet |
|---|---:|
| `cold` | 7.4 |
| `l1` | 5.6 |
| `l2.l1l2` | 7.6 |
| `l2.l2` | 7.8 |
| `l3.l1l2l3` | 6.4 |
| `l3.l2l3` | 6.6 |
| `l3.l3` | 7.4 |

### Headline — memory cuts CONVENTION restatement more than FEATURE restatement

- **sonnet**: memory removes **48%** of the cold convention-restatement burden vs **7%** of the feature burden — it cuts convention restatement **7.2×** as deeply as feature restatement. The transferable-vs-app-specific differential is the signal. Features move at all only because feeds shares α/β with the priors (memory transfer leaking into the control) — see the native-only control below for the leak-free check.

## Secondary

### β-bucket on feeds, ROUND 1 /4 (front-loading: does links' memory lift β in the first draft? — measured at round 1; β saturates to 4/4 at convergence)

| regime | sonnet |
|---|---:|
| `cold` | 2.40 |
| `l1` | 3.40 |
| `l2.l1l2` | 2.40 |
| `l2.l2` | 1.80 |
| `l3.l1l2l3` | 2.80 |
| `l3.l2l3` | 2.60 |
| `l3.l3` | 2.40 |

### Direct-vs-followed on tier-read regimes (mean link-following rate, feeds)

| regime | sonnet |
|---|---:|
| `cold` | 0.00 |
| `l1` | 1.00 |
| `l2.l1l2` | 0.00 |
| `l2.l2` | 1.00 |
| `l3.l1l2l3` | 0.00 |
| `l3.l2l3` | 1.00 |
| `l3.l3` | 1.00 |


feeds round-1 NATIVE-bucket pass count (the feed-specific features no prior app teaches). If memory is a clean say-once mechanism this should NOT rise with memory; if it does, memory is also lifting first-draft quality generally (a real effect, but it means 'feature interventions' is not a pure untouched control).
### Native-only control on feeds (leak-free: no shared α/β)

| regime | sonnet |
|---|---:|
| `cold` | 1.80 |
| `l1` | 1.60 |
| `l2.l1l2` | 1.80 |
| `l2.l2` | 1.80 |
| `l3.l1l2l3` | 2.00 |
| `l3.l2l3` | 2.00 |
| `l3.l3` | 1.60 |

### Cost + time to endpoint (mean $/min per trial)

| regime | sonnet |
|---|---:|
| `cold` | 6.77 / 33 |
| `l1` | 11.33 / 46 |
| `l2.l1l2` | 10.27 / 38 |
| `l2.l2` | 10.88 / 44 |
| `l3.l1l2l3` | 10.03 / 38 |
| `l3.l2l3` | 10.38 / 38 |
| `l3.l3` | 12.51 / 53 |

### Learn-capture quality (did the agent persist what matters, per tier)

Cell = mean convention-coverage (captured/stated) · episode-extraction%. The agent runs its own /learn skill; an L1 episode must ALWAYS be extracted (the foundation every tier links down to), so episode% < 100 is a real learn failure.

| write-tier | sonnet |
|---|---:|
| `L1` | 0.99 · ep 100% |
| `L2` | 1.00 · ep 100% |
| `L3` | 1.00 · ep 100% |


### Token I/O + cost audit (per model, over covered cells)  ·  **120/130 cells covered** (the rest lost their transcripts to cfg-pool re-creation across resumes — run-time token capture in the result JSON fixes this going forward)

Reconstructing $ from token counts × the price sheet reproduces the CLI's reported cost (ratio ≈ 1.00× over MATCHED cells — the §6 provenance check). Cost is cache-dominated.

| model | cells | input | output | cache-write | cache-read | reported $ | recomputed $ | ratio |
|---|--:|--:|--:|--:|--:|--:|--:|--:|
| sonnet | 120 | 6,616 | 5,891,689 | 11,299,415 | 397,917,081 | 248.66 | 250.14 | 1.01× |


### Cost calibration (per-operation; grounds the full-run estimate)

| op | model | app | n | mean $ | mean rounds |
|---|---|---|--:|--:|--:|
| build | sonnet | feeds | 35 | 3.12 | 2.2 |
| build | sonnet | links | 35 | 2.31 | 2.1 |
| build | sonnet | notes | 5 | 1.01 | 2.0 |
| learn | sonnet | links | 30 | 1.33 | — |
| learn | sonnet | notes | 15 | 0.91 | — |


## Convergence guard + honest caveats

- **Converged within the 6-round budget: 66/75 builds.** The primary metric is the round-1 intervention count, not a stall rate; low convergence means some builds plateau below the full bar — investigate feedback-symptom effectiveness / stale-break, separately from say-once.
- **n=5 trial(s).** Models: sonnet (single model — cross-model still open).
- **Regime axis (the v2 question): FLAT for sonnet at n=5.** All warm regimes sit in a 9.4–10.0 band (spread 0.6) vs cold 18.8 — the cold→warm gap (~9) dwarfs any between-tier difference. Writing L3 syntheses doesn't beat L1 episodes; reading only the distilled L3 doesn't beat blended; raw L1 episodes capture the full effect. **Open: cross-model** (haiku/opus may differ — the 2026-06-02 run found weak models prefer L2). β-accumulation (round-1 feeds β/4, cold 2.4 → warm 2.6) is flat/inconclusive at this difficulty — β saturates to 4/4 by convergence, so the signal only exists in the first draft and is noisy at n=5.
- Learn is agent-driven; learn-capture coverage + episode-extraction above are measured outputs (a poor capture is recorded, not engineered away).
- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline.