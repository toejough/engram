# Cumulative cross-app memory accumulation — results (v2)

Engram SHA: `933353a87591` · date: 2026-06-10 · models: opus · trials: [1, 2, 3, 4, 5] · price sheet: 2026-06-02

> A NEW clean baseline (re-metric'd say-once + 7 vs 5 regimes); NOT comparable cell-for-cell to the 2026-06-02 run.

---

## VERDICT — lazy (`l2.lazy`) vs eager/proactive (`l2.l1l2`), opus, n=5 (2026-06-10)

**Lazy is ~25–32% cheaper and not worse on any measured axis** — same completion (100% both), same
say-once (lazy 6.8 vs eager 7.0 = a tie at n=5). Mean/trial: total **$5.83 vs $7.80** (−25%); tokens
**4.4M vs 5.8M**; wall **19 vs 24 min**; app2+app3 chain **$3.70 vs $5.45** (−32%). Learn-capture
coverage 1.00 / episode 100% for both. Validity confirmed: arm B crystallized **1 L2/app on demand**
and persisted it forward (`l2.lazy` vaults carry L2=1).

**But this run does NOT show lazy's on-demand L2 is as *valuable* as eager's** — only that
deferring/skipping L2 costs less and loses nothing the say-once metric can see. **The confound:**
say-once was ~6.8–7.0 whether the vault held **0 L2s** (the first, invalid L1-only run), **1 L2**
(lazy here), or **~10 L2s** (eager, ~one fact per convention). For opus on this thin chain **the L2
layer is near-optional** — L1 episodes alone already carry the conventions. So the arms aren't matched
on output (eager wrote ~10 L2s; lazy crystallized 1), and "lazy cheaper" largely means "lazy made far
fewer L2s and the metric couldn't tell." Two readings the data can't separate: **(a)** lazy smartly
avoids L2s nothing ever recalls (waste eager pays); **(b)** L2 was optional anyway here. The
L1-only=6.8 datapoint leans toward (b).

**Opus-first cuts against this question.** Opus is the model most able to re-derive conventions from
raw episodes — i.e. the one where L2 looks *most* optional (the 2026-06-08 round's "opus thrives on
less memory"). The discriminating follow-up, when wanted: a setting where **L1-only measurably FAILS**
— a longer/broader chain hitting many distinct conventions, or a weaker model — because only there can
lazy demonstrate it *recovers* L2's value on demand rather than just skipping it cheaply.

**Scope:** n=5, single model (opus), thin 3-app chain (lazy crystallized 1 L2/app; `l2_composed`=0 —
the compositional/recursion behavior was not exercised, deliberately deferred). The cost margin partly
reflects lazy reading leaner memory (1 L2 vs ~10); at vault scale it would narrow. Directional, not
definitive.

> ⚠️ The generic "Recommendation" / "Convergence guard" prose lower in this file is boilerplate
> emitted by the full-matrix aggregator (it references `l2.l2`/`cold`/sonnet/haiku/7 regimes and
> "25/25 builds" from the 2026-06-08 run); it is **NOT** specific to this 2-regime opus A/B. Trust the
> VERDICT above and the data tables; ignore that boilerplate.

---

## Primary — repeated-convention interventions (say-once vs every-app)

Chain-summed conventions the human had to STATE (app1+app2+app3). Prediction: memory ≈ |conv| once; no-memory (`cold`) ≈ |conv| × 3. The delta on app2/app3 — conventions memory carried so they did not recur — is memory's value.

### Convention interventions to endpoint (mean/trial)

| regime | opus |
|---|---:|
| `l2.l1l2` | 7.0 |
| `l2.lazy` | 6.8 |

### Feature interventions — CONTROL (app-specific; nobody carries these)

| regime | opus |
|---|---:|
| `l2.l1l2` | 4.6 |
| `l2.lazy` | 4.2 |

### Headline — memory cuts CONVENTION restatement far more than FEATURE restatement

- **opus**: insufficient data.

The transferable-vs-app-specific GAP is the signal. The feature side is not a pure control — feeds shares α/β with the priors, so memory transfer leaks in (and for haiku the noisy feature side even moves the wrong way); the leak-free check is the native-only control below.

**Cross-model: memory is a capability AMPLIFIER, not an equalizer.** The convention reduction grows with model strength (see per-model % above) — memory helps the stronger model more, widening the capability gap, reproducing the 2026-06-02 finding.

### Headline stats — to the endpoint (notes→links→feeds chain, mean per trial)

`conv-restate` = convention restatements the human made (the say-once metric, lower=better). `review` = feedback rounds. **Memory's win is conv-restate; it does NOT reduce time/tokens/$ — recall + richer learn cost more.**

| model | arm | conv-restate | review | converged | wall min | tokens | $ |
|---|---|--:|--:|--:|--:|--:|--:|
| opus | warm | 6.9 | 2.3 | 100% | 21 | 5.1M | 6.81 |

## Secondary

### β-bucket on feeds, ROUND 1 /4 (front-loading: does links' memory lift β in the first draft? — measured at round 1; β saturates to 4/4 at convergence)

| regime | opus |
|---|---:|
| `l2.l1l2` | 4.00 |
| `l2.lazy` | 3.80 |

### Direct-vs-followed on tier-read regimes (mean link-following rate, feeds)

| regime | opus |
|---|---:|
| `l2.l1l2` | 0.00 |
| `l2.lazy` | 0.00 |


feeds round-1 NATIVE-bucket pass count (the feed-specific features no prior app teaches). If memory is a clean say-once mechanism this should NOT rise with memory; if it does, memory is also lifting first-draft quality generally (a real effect, but it means 'feature interventions' is not a pure untouched control).
### Native-only control on feeds (leak-free: no shared α/β)

| regime | opus |
|---|---:|
| `l2.l1l2` | 2.00 |
| `l2.lazy` | 2.00 |

### Cost & convergence by regime (mean per trial) — learn$ vs build$ split

`learn$` rises with write-tier (L1 episode < L2 +facts < L3 +synthesis); `build$` is dominated by feedback round-count (convergence), which is tier-insensitive — so total $ does not cleanly follow tier simplicity.

**opus**

| regime | write | learn$ | build$ | total$ | wall | tokens | conv% |
|---|---|--:|--:|--:|--:|--:|--:|
| `l2.l1l2` | L2 | 1.86 | 5.94 | 7.80 | 24 | 5.8M | 100% |
| `l2.lazy` | L1 | 1.36 | 4.47 | 5.83 | 19 | 4.4M | 100% |



### Learn-capture quality (did the agent persist what matters, per tier)

Cell = mean convention-coverage (captured/stated) · episode-extraction%. The agent runs its own /learn skill; an L1 episode must ALWAYS be extracted (the foundation every tier links down to), so episode% < 100 is a real learn failure.

| write-tier | opus |
|---|---:|
| `L1` | 1.00 · ep 100% |
| `L2` | 1.00 · ep 100% |
| `L3` | — · — |


### Feedback escalation depth — how granular before convergence (completed builds)

`conv-depth` = median max times a *convention* was restated before it stuck (1 = fixed on the symptom; ≥2 = needed the literal code-level prescription). `#presc` = mean conventions per build that needed the prescriptive fix. Higher = more hand-holding — expected to fall as model strength rises.

| model | app | conv-depth (median) | #presc (mean) |
|---|---|--:|--:|
| opus | notes | 1.0 | 0.0 |
| opus | links | 0.0 | 0.0 |
| opus | feeds | 0.0 | 0.0 |


## Full matrix (model × regime × app, medians)

### Full matrix — app1 · notes (cold build shared per model; row = write-tier of its learn)

Medians. **Bold** = best (lowest) per model per metric. app1 build is identical across rows; only learn cost/tokens/time differ by tier.

| model | write-tier | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| opus | `none` | **1** | **1** | **2** | **1.38** | **0.8M** | **5** |
| opus | `L1` | 1 | 1 | 2 | 2.08 | 1.3M | 7 |
| opus | `L2` | 1 | 1 | 2 | 2.19 | 1.3M | 8 |
| opus | `L3` | 1 | 1 | 2 | 1.38 | 0.8M | 5 |

### Full matrix — app2 · links (recall under regime)

Medians. **Bold** = best (lowest) per model per metric. † = <60% of this cell's builds completed (resource figures include capped runs).

| model | regime | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| opus | `l2.l1l2` | **1** | **0** | **2** | 3.33 | 2.7M | 10 |
| opus | `l2.lazy` | 1 | 0 | 2 | **2.31** | **2.0M** | **7** |

### Full matrix — app3 · feeds (recall under regime; terminal, no learn)

Medians. **Bold** = best (lowest) per model per metric. † = <60% of this cell's builds completed (resource figures include capped runs).

| model | regime | human turns | prescript | →converge | cost $ | tokens | time min |
|---|---|--:|--:|--:|--:|--:|--:|
| opus | `l2.l1l2` | **0** | **0** | **1** | 1.92 | 1.3M | 5 |
| opus | `l2.lazy` | 0 | 0 | 1 | **1.32** | **1.1M** | **5** |



### Token I/O + cost audit (per model, over covered cells)  ·  45/45 LLM-using cells captured (0 cold no-op learns excluded)

Reconstructing $ from token counts × the price sheet reproduces the CLI's reported cost (ratio ≈ 1.00× over MATCHED cells — the §6 provenance check). Cost is cache-dominated.

| model | cells | input | output | cache-write | cache-read | reported $ | recomputed $ | ratio |
|---|--:|--:|--:|--:|--:|--:|--:|--:|
| opus | 45 | 175,616 | 906,321 | 2,472,011 | 42,628,537 | 60.69 | 60.30 | 0.99× |


### Cost calibration (per-operation; grounds the full-run estimate)

| op | model | app | n | mean $ | mean rounds |
|---|---|---|--:|--:|--:|
| build | opus | feeds | 10 | 1.67 | 1.1 |
| build | opus | links | 10 | 2.05 | 2.2 |
| build | opus | notes | 5 | 1.49 | 2.0 |
| learn | opus | links | 10 | 0.85 | — |
| learn | opus | notes | 10 | 0.76 | — |


## Convergence guard + honest caveats

- **Converged within the 15-round budget: 25/25 builds.** The primary metric is the round-1 intervention count, not a stall rate; low convergence means some builds plateau below the full bar — investigate feedback-symptom effectiveness / stale-break, separately from say-once.
- **n=5 trial(s).** Models: opus (single model — cross-model still open).
- **Regime axis: insufficient complete chains to compare.**
- Learn is agent-driven; learn-capture coverage + episode-extraction above are measured outputs (a poor capture is recorded, not engineered away).
- Re-derive cleanly each time a model ships or engram gains a feature; `compare.py` vs this baseline.

## Recommendation — if you could pick one model + regime

_Derived from the baseline below (engram `933353a87591` · 2026-06-10 · opus · n=[1, 2, 3, 4, 5]). A point-in-time judgement on this data; revisit when re-deriving._

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