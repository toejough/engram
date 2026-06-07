# Cumulative cross-app memory accumulation — results (v2)

Engram SHA: `195ae2315515` · date: 2026-06-06 · models: sonnet · trials: [1] · price sheet: 2026-06-02

> A NEW clean baseline (re-metric'd say-once + 7 vs 5 regimes); NOT comparable cell-for-cell to the 2026-06-02 run.

## Primary — repeated-convention interventions (say-once vs every-app)

Chain-summed conventions the human had to STATE (app1+app2+app3). Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.

### Convention interventions to endpoint (mean/trial)

| regime | sonnet |
|---|---:|
| `cold` | 21.0 |
| `l1` | 15.0 |
| `l2.l1l2` | 18.0 |
| `l2.l2` | 17.0 |
| `l3.l1l2l3` | 15.0 |
| `l3.l2l3` | 15.0 |
| `l3.l3` | 15.0 |

### Feature interventions — CONTROL (memory should not move these)

| regime | sonnet |
|---|---:|
| `cold` | 9.0 |
| `l1` | 10.0 |
| `l2.l1l2` | 8.0 |
| `l2.l2` | 8.0 |
| `l3.l1l2l3` | 7.0 |
| `l3.l2l3` | 8.0 |
| `l3.l3` | 8.0 |

## Secondary

### β-bucket on feeds (does β transfer once links' memory is present)

| regime | sonnet |
|---|---:|
| `cold` | 2.00 |
| `l1` | 1.00 |
| `l2.l1l2` | 2.00 |
| `l2.l2` | 2.00 |
| `l3.l1l2l3` | 2.00 |
| `l3.l2l3` | 2.00 |
| `l3.l3` | 2.00 |

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

### Cost + time to endpoint (mean $/min per trial)

| regime | sonnet |
|---|---:|
| `cold` | 4.28 / 8 |
| `l1` | 8.20 / 22 |
| `l2.l1l2` | 9.70 / 26 |
| `l2.l2` | 7.40 / 17 |
| `l3.l1l2l3` | 9.59 / 32 |
| `l3.l2l3` | 9.76 / 24 |
| `l3.l3` | 10.51 / 30 |

### Learn-capture quality (did the agent persist the stated conventions per tier)

Cell = mean coverage (captured/stated) · engaged% (wrote any note). The agent runs its own /learn skill — this measures whether the memory system captured what matters.

| write-tier | sonnet |
|---|---:|
| `L1` | 1.00 · 100% |
| `L2` | 1.00 · 100% |
| `L3` | 1.00 · 100% |


### Cost calibration (per-operation; grounds the full-run estimate)

| op | model | app | n | mean $ | mean rounds |
|---|---|---|--:|--:|--:|
| build | sonnet | feeds | 7 | 2.47 | 3.0 |
| build | sonnet | links | 7 | 1.74 | 3.0 |
| build | sonnet | notes | 1 | 0.32 | 3.0 |
| learn | sonnet | links | 6 | 1.13 | — |
| learn | sonnet | notes | 3 | 1.00 | — |


## Convergence guard + honest caveats

- **Converged within the 6-round budget: 0/15 builds.** The primary metric is the round-1 intervention count, not a stall rate; but 0 (or low) convergence means builds plateau below the full bar — investigate the feedback-symptom effectiveness / stale-break, separately from say-once.
- **n=1 trial(s) — PILOT, variance unknown; the standing run is n=5.** Models: sonnet (single model — not yet cross-model).
- Learn is agent-driven; learn-capture coverage above shows whether the agent actually persisted each stated convention (a measured output, not assumed).
- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline.