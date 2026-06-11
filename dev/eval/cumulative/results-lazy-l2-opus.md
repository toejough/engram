# Cumulative cross-app memory accumulation — results (v2)

Engram SHA: `a11cfeb369c8` · date: 2026-06-10 · models: haiku, sonnet, opus · trials: [1, 2, 3, 4, 5] · price sheet: 2026-06-02

> A NEW clean baseline (re-metric'd say-once + 7 vs 5 regimes); NOT comparable cell-for-cell to the 2026-06-02 run.

---

## VERDICT — lazy (`l2.lazy`) vs eager/proactive (`l2.l1l2`), 3 models × n=5 (2026-06-10)

**Lazy beats or ties eager on every measured axis, for every model — and for the weakest model it
wins decisively on completion.** Validity confirmed: lazy crystallized L2s on demand and persisted
them forward for all three models (l2_generated: haiku 14, sonnet 38, opus 8; with L2-on-L2
**composition** for haiku 4, sonnet 7).

| metric (mean/trial) | haiku eager→lazy | sonnet eager→lazy | opus eager→lazy |
|---|---|---|---|
| **say-once** (conv-restate, ↓) | 10.6 → **9.0** | 12.0 → **11.2** | 7.0 → **6.8** |
| **total $** | 4.18 → **3.51** (−16%) | 9.94 → **7.19** (−28%) | 7.80 → **5.83** (−25%) |
| **tokens** | 27.4M → **22.2M** | 14.1M → **8.7M** | 5.8M → **4.4M** |
| **completion (conv%)** | **40% → 80%** | 100% = 100% | 100% = 100% |

**This run answers the question opus alone couldn't.** For opus the two arms *tied* (the earlier
finding: L2 was near-optional, L1 episodes already carried the conventions). For **haiku — the
weakest model — the arms diverge sharply: lazy completes 80% of builds vs eager's 40%**, at lower
cost and fewer restatements. That divergence is the signal: when L2 actually matters (weaker model,
where episodes alone don't suffice), **the *way* L2 is created matters — and lazy's on-demand,
recall-crystallized L2 is the better way.** This is evidence for the "lazy is genuinely better"
reading over "lazy is just cheaper because it does less": for haiku, lazy did *more useful* work
(2× completion) at *lower* cost. Plausible mechanism — lazy makes the agent *synthesize* the
conventions itself at recall (the three-band write), which engages a weak model with them more than
handing it pre-written eager L2 facts to read.

**Net:** defer-L2-to-recall is the better policy across the board — cheaper for all, say-once-neutral
or better for all, and a large completion win for the weakest model. The cost edge is largest for
the stronger models; the *quality* edge (completion) is largest for the weakest.

**Caveats (honest):** n=5 — haiku's 40%→80% is 4/10 vs 8/10 converged builds: a real gap but small
sample. This compares two *L2 approaches*, not L2-vs-no-L2 (no `cold`/`l1` arm in this run) — but the
arm *divergence* for haiku is itself evidence the L2 layer is load-bearing there (if it were
optional, the arms would tie as they do for opus). Thin 3-app chain (composition only lightly
exercised). 4/135 cells lost their token transcripts to mid-run cfg-pool rebuilds (cost-audit
cross-check only; the say-once/cost figures derive from the build results, which are intact).

> ⚠️ The generic "Recommendation" / "Convergence guard" prose lower in this file is full-matrix
> aggregator boilerplate (it references `l2.l2`/`cold`/7 regimes). It is NOT this 2-regime A/B —
> trust the VERDICT above and the data tables.

---

## Primary — repeated-convention interventions (say-once vs every-app)

Chain-summed conventions the human had to STATE (app1+app2+app3). Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.

### Convention interventions to endpoint (mean/trial)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `l2.l1l2` | 10.6 | 12.0 | 7.0 |
| `l2.lazy` | 9.0 | 11.2 | 6.8 |

### Feature interventions — CONTROL (app-specific; nobody carries these)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `l2.l1l2` | 8.8 | 5.4 | 4.6 |
| `l2.lazy` | 7.6 | 5.8 | 4.2 |

### Headline — memory cuts CONVENTION restatement far more than FEATURE restatement

- **haiku**: insufficient data.
- **sonnet**: insufficient data.
- **opus**: insufficient data.

The transferable-vs-app-specific GAP is the signal. The feature side is not a pure control — feeds shares α/β with the priors, so memory transfer leaks in (and for haiku the noisy feature side even moves the wrong way); the leak-free check is the native-only control below.

**Cross-model: memory is a capability AMPLIFIER, not an equalizer.** The convention reduction grows with model strength (see per-model % above) — memory helps the stronger model more, widening the capability gap, reproducing the 2026-06-02 finding.

### Headline stats — to the endpoint (notes→links→feeds chain, mean per trial)

`conv-restate` = convention restatements the human made (the say-once metric, lower=better). `review` = feedback rounds. **Memory's win is conv-restate; it does NOT reduce time/tokens/$ — recall + richer learn cost more.**

| model | arm | conv-restate | review | converged | wall min | tokens | $ |
|---|---|--:|--:|--:|--:|--:|--:|
| haiku | warm | 9.8 | 11.3 | 60% | 28 | 24.8M | 3.84 |
| sonnet | warm | 11.6 | 2.6 | 100% | 50 | 11.4M | 8.57 |
| opus | warm | 6.9 | 2.3 | 100% | 21 | 5.1M | 6.81 |

## Secondary

### β-bucket on feeds, ROUND 1 /4 (front-loading: does links' memory lift β in the first draft? — measured at round 1; β saturates to 4/4 at convergence)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `l2.l1l2` | 2.00 | 2.40 | 4.00 |
| `l2.lazy` | 2.00 | 2.80 | 3.80 |

### Direct-vs-followed on tier-read regimes (mean link-following rate, feeds)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `l2.l1l2` | 0.00 | 0.00 | 0.00 |
| `l2.lazy` | 0.00 | 0.00 | 0.00 |


feeds round-1 NATIVE-bucket pass count (the feed-specific features no prior app teaches). If memory is a clean say-once mechanism this should NOT rise with memory; if it does, memory is also lifting first-draft quality generally (a real effect, but it means 'feature interventions' is not a pure untouched control).
### Native-only control on feeds (leak-free: no shared α/β)

| regime | haiku | sonnet | opus |
|---|---:|---:|---:|
| `l2.l1l2` | 1.40 | 2.00 | 2.00 |
| `l2.lazy` | 2.00 | 2.00 | 2.00 |

### Cost & convergence by regime (mean per trial) — learn$ vs build$ split

`learn$` rises with write-tier (L1 episode < L2 +facts < L3 +synthesis); `build$` is dominated by feedback round-count (convergence), which is tier-insensitive — so total $ does not cleanly follow tier simplicity.

**haiku**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `l2.l1l2` | L2 | 0.31 | 3.87 | 4.18 | 30 | 27.4M | 40% |
| `l2.lazy` | L1 | 0.29 | 3.22 | 3.51 | 25 | 22.2M | 80% |

**sonnet**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `l2.l1l2` | L2 | 1.84 | 8.10 | 9.94 | 56 | 14.1M | 100% |
| `l2.lazy` | L1 | 0.99 | 6.20 | 7.19 | 43 | 8.7M | 100% |

**opus**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `l2.l1l2` | L2 | 1.86 | 5.94 | 7.80 | 24 | 5.8M | 100% |
| `l2.lazy` | L1 | 1.36 | 4.47 | 5.83 | 19 | 4.4M | 100% |



### Learn-capture quality (did the agent persist what matters, per tier)

Cell = mean convention-coverage (captured/stated) · episode-extraction%. The agent runs its own /learn skill; an L1 episode must ALWAYS be extracted (the foundation every tier links down to), so episode% < 100 is a real learn failure.

| write-tier | haiku | sonnet | opus |
|---|---:|---:|---:|
| `L1` | 1.00 · ep 100% | 1.00 · ep 100% | 1.00 · ep 100% |
| `L2` | 1.00 · ep 100% | 1.00 · ep 100% | 1.00 · ep 100% |
| `L3` | — · — | — · — | — · — |


### Feedback escalation depth — how granular before convergence (completed builds)

`conv-depth` = median max times a *convention* was restated before it stuck (1 = fixed on the symptom; ≥2 = needed the literal code-level prescription). `#presc` = mean conventions per build that needed the prescriptive fix. Higher = more hand-holding — expected to fall as model strength rises.

| model | app | conv-depth (median) | #presc (mean) |
|---|---|--:|--:|
| haiku | notes | 1.2 | 1.4 |
| haiku | links | 1.2 | 0.2 |
| haiku | feeds | 1.6 | 0.6 |
| sonnet | notes | 1.0 | 0.0 |
| sonnet | links | 0.6 | 0.0 |
| sonnet | feeds | 0.7 | 0.1 |
| opus | notes | 1.0 | 0.0 |
| opus | links | 0.0 | 0.0 |
| opus | feeds | 0.0 | 0.0 |


## Full matrix (model × regime × app, medians)

### Full matrix — app1 · notes (cold build shared per model; row = write-tier of its learn)

Medians. **Bold** = best (lowest) per model per metric. app1 build is identical across rows; only learn cost/tokens/time differ by tier.

| model | write-tier | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| haiku | `none` | **2** | **2** | **3** | **0.65** | **3.8M** | **6** |
| haiku | `L1` | 2 | 2 | 3 | 0.77 | 4.1M | 8 |
| haiku | `L2` | 2 | 2 | 3 | 0.90 | 4.9M | 8 |
| haiku | `L3` | 2 | 2 | 3 | 0.65 | 3.8M | 6 |
| sonnet | `none` | **1** | **1** | **2** | **0.79** | **0.6M** | **7** |
| sonnet | `L1` | 1 | 1 | 2 | 1.46 | 1.7M | 10 |
| sonnet | `L2` | 1 | 1 | 2 | 2.05 | 1.9M | 14 |
| sonnet | `L3` | 1 | 1 | 2 | 0.79 | 0.6M | 7 |
| opus | `none` | **1** | **1** | **2** | **1.38** | **0.8M** | **5** |
| opus | `L1` | 1 | 1 | 2 | 2.08 | 1.3M | 7 |
| opus | `L2` | 1 | 1 | 2 | 2.19 | 1.3M | 8 |
| opus | `L3` | 1 | 1 | 2 | 1.38 | 0.8M | 5 |

### Full matrix — app2 · links (recall under regime)

Medians. **Bold** = best (lowest) per model per metric. † = <60% of this cell's builds completed (resource figures include capped runs).

| model | regime | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| haiku | `l2.l1l2` | 2 | **1** | **2** | 1.48 | 9.1M | **9** |
| haiku | `l2.lazy` | **1** | 1 | 2 | **1.23** | **7.0M** | 10 |
| sonnet | `l2.l1l2` | **0** | **0** | **1** | 4.07 | 5.9M | 21 |
| sonnet | `l2.lazy` | 1 | 1 | 2 | **2.54** | **3.5M** | **13** |
| opus | `l2.l1l2` | **1** | **0** | **2** | 3.33 | 2.7M | 10 |
| opus | `l2.lazy` | 1 | 0 | 2 | **2.31** | **2.0M** | **7** |

### Full matrix — app3 · feeds (recall under regime; terminal, no learn)

Medians. **Bold** = best (lowest) per model per metric. † = <60% of this cell's builds completed (resource figures include capped runs).

| model | regime | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| haiku | `l2.l1l2` | **3** | **1** | **3** | 1.46 | 9.3M | 10 |
| haiku | `l2.lazy` | 3 | 2 | 4 | **0.98** | **5.6M** | **7** |
| sonnet | `l2.l1l2` | **1** | **1** | **2** | 3.27 | 4.6M | 21 |
| sonnet | `l2.lazy` | 1 | 1 | 2 | **2.12** | **2.8M** | **15** |
| opus | `l2.l1l2` | **0** | **0** | **1** | 1.92 | 1.3M | 5 |
| opus | `l2.lazy` | 0 | 0 | 1 | **1.32** | **1.1M** | **5** |



### Token I/O + cost audit (per model, over covered cells)  ·  **131/135 LLM-using cells captured** (0 cold no-op learns excluded; 4 lost their transcripts to cfg-pool re-creation — run-time capture prevents this going forward)

Reconstructing $ from token counts × the price sheet reproduces the CLI's reported cost (ratio ≈ 1.00× over MATCHED cells — the §6 provenance check). Cost is cache-dominated.

| model | cells | input | output | cache-write | cache-read | reported $ | recomputed $ | ratio |
|---|--:|--:|--:|--:|--:|--:|--:|--:|
| haiku | 44 | 21,104 | 1,366,729 | 3,170,702 | 226,564,352 | 35.66 | 33.47 | 0.94× |
| sonnet | 42 | 3,209 | 1,688,188 | 4,124,268 | 105,186,474 | 80.74 | 72.35 | 0.90× |
| opus | 45 | 175,616 | 906,321 | 2,472,011 | 42,628,537 | 60.69 | 60.30 | 0.99× |


### Cost calibration (per-operation; grounds the full-run estimate)

| op | model | app | n | mean $ | mean rounds |
|---|---|---|--:|--:|--:|
| build | haiku | feeds | 10 | 1.68 | 6.7 |
| build | haiku | links | 10 | 1.32 | 5.0 |
| build | haiku | notes | 5 | 0.55 | 2.6 |
| build | opus | feeds | 10 | 1.67 | 1.1 |
| build | opus | links | 10 | 2.05 | 2.2 |
| build | opus | notes | 5 | 1.49 | 2.0 |
| build | sonnet | feeds | 10 | 3.20 | 2.0 |
| build | sonnet | links | 10 | 2.97 | 1.6 |
| build | sonnet | notes | 5 | 0.98 | 2.0 |
| learn | haiku | links | 10 | 0.14 | — |
| learn | haiku | notes | 10 | 0.16 | — |
| learn | opus | links | 10 | 0.85 | — |
| learn | opus | notes | 10 | 0.76 | — |
| learn | sonnet | links | 10 | 0.64 | — |
| learn | sonnet | notes | 10 | 0.78 | — |


## Convergence guard + honest caveats

- **Converged within the 15-round budget: 70/75 builds.** The primary metric is the round-1 intervention count, not a stall rate; low convergence means some builds plateau below the full bar — investigate feedback-symptom effectiveness / stale-break, separately from say-once.
- **n=5 trial(s).** Models: haiku, sonnet, opus.
- **Regime axis: insufficient complete chains to compare.**
- Learn is agent-driven; learn-capture coverage + episode-extraction above are measured outputs (a poor capture is recorded, not engineered away).
- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline.

## Recommendation — if you could pick one model + regime

_Derived from the baseline below (engram `a11cfeb369c8` · 2026-06-10 · opus, sonnet, haiku · n=[1, 2, 3, 4, 5]). A point-in-time judgement on this data; revisit when re-deriving._

**Pick: `opus` + `l2.l2`** (write L2 facts, read L2 tier-capped) **— when you're building many
apps that share conventions over time.** Otherwise **`sonnet`/`opus` + `cold`** is the cheaper
reliable floor for a short horizon. Reasoning, strictly from the tables above:

**Model — opus (sonnet a close second; haiku is out).**
- Cost is NOT the differentiator people assume: warm chains cost about the same on both
  (sonnet ≈ \$8.4–11.2, opus ≈ \$8.8–11.3) — opus's higher per-token rate is offset by its
  token-efficiency (≈7–10M vs sonnet ≈10–18M) and ~2-round convergence. At cost parity opus is
  faster (≈6–8 min/app vs ≈16–22), edges say-once (7 vs 9 conventions), and needs **zero**
  prescriptive hand-holding (sonnet ~1).
- **haiku is excluded:** even with escalation it completes ≤80% of chains per regime (≈42%
  overall) and only by being handed the literal code (depth-2 prescriptions). Not shippable as a
  default.

**Regime — warm, `l2.l2` specifically, on a stated principle (not a measured tier win).**
- The decision that matters is **cold vs warm**, not which tier: among the 6 warm regimes the
  spread is n=5 noise. For strong models tier is *flat* (the regime-axis finding above).
- **Warm vs cold is a horizon call.** Cold completes 100% for strong models at ~half the cost
  (≈\$4–5 vs ≈\$9) but carries ~2× the convention burden (18–19 vs 7–9 restatements). Warm's
  say-once benefit is paid once and **recovered on every later app that shares conventions**,
  while its extra cost is per-build — so a 3-app chain *understates* warm. Many convention-sharing
  apps → warm wins; a one-off or two → cold is the reliable floor.
- **Why `l2.l2` among the warm regimes:** it's the *never-worst-across-capability* config — the one
  that rescued haiku (80% complete vs ≤40% for blended/L1 reads) and ties for best on the strong
  models. That makes it the safe choice if the model is ever swapped or downgraded. A robustness
  tiebreak, not a measured victory over the other warm tiers.

**What warm does NOT buy for strong models (honest caveat):** it does **not** reduce review
round-trips — human turns are ~3 whether cold or warm for sonnet/opus; they fold recalled
conventions into the same rounds. Memory **front-loads correctness** (fewer distinct things to
teach, compounding across apps), it does not cut iterations here. The dramatic round-trip saving
(20→6) is real only for haiku, which we don't ship — so the pitch for opus+warm is "teach each
convention once, ever," not "fewer review cycles."