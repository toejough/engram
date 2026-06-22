# Engram validation — experiment log

Living tracker for the cumulative cross-app memory eval (`dev/eval/cumulative/`). One row per
**run**; runs are grouped by **variant** (the memory-activation mechanism under test). Update this
file as runs complete — record the data path, the headline amortized result, and the retro verdict.

> **Verify, don't guess.** Numbers here are copied from `aggregate.py --root <data path>` output or
> direct per-op JSON reads — never from memory. If a row says "running", it has no trusted result yet.

---

## Target axes (what every run measures)

| Axis | Question | Primary metric (unit) | Exercised by current chain? |
|---|---|---|---|
| **C1** | faster? | active time = recall+build+learn (s); build time (s) | ✅ |
| **C2** | cheaper? | cost (USD); tokens (Mtok) | ✅ |
| **C3** | fewer interactions? | convention restatements (count); feedback rounds (rounds) | ✅ |
| **C4** | adapts as standards change over time? | (separate probe — not in the 3-app chain) | ❌ not yet |
| **C5** | uses recent history? | recency probe (separate) | ❌ not yet |
| **C6** | emergent synthesis / compounding lessons? | crystallize-new-note + adversarial judge (separate fixtures) | ❌ not yet |

**Reporting rule (learned 2026-06-21):** report the **amortized** view — separate the app1 **seed**
(one-time memory write, no prior memory to recall) from the **payback** apps (2..N). Chain totals
smear the seed cost across every app and misread it as a per-app penalty. Memory's economics are
"pay once to seed, win on every subsequent app."

---

## Variant registry (the memory-activation mechanism under test)

What differs between variants is **how the agent is prompted to use memory** — the recall/learn
machinery and the binary are identical. Cold (no-memory) is the shared baseline in every run.

| ID | Variant | Manipulation | Hypothesis |
|---|---|---|---|
| **V0** | **Baseline** (current) | Build prompt embeds an explicit per-op directive: "before writing code, INVOKE /recall"; after convergence, "INVOKE /learn". | Memory cuts C3 and (amortized) C1/C2 on payback apps. |
| **V1** | **please-driven** | Every instruction/correction is issued via **/please <ask>** — the please workflow orchestrates recall→plan→build→learn with its review gates, instead of a hand-embedded recall/learn directive. | Please's structured recall + gates lift reliability of memory use and first-pass quality vs. a bare directive. |
| **V2** | **CLAUDE.md advice** | Remove the explicit per-op recall/learn directive; instead place ambient recall/learn guidance in **CLAUDE.md** and rely on the agent to self-trigger. | Ambient advice is weaker than an explicit directive — tests whether memory still gets used without per-prompt nudging. |
| **V3** | **Combination** | please-driven **and** CLAUDE.md advice together. | Best-case activation: structured workflow + ambient reinforcement. |

> Assumptions to confirm with Joe: (1) V1 wraps both the initial build instruction and each
> feedback correction in /please; (2) V2 strips the build-prompt recall/learn lines entirely (pure
> ambient); (3) all variants keep cold as the no-memory control. Adjust the manipulations above if
> these don't match intent.

---

## Run log

| # | Variant | Model | n | stall patience | engram SHA | data path | status | headline (payback: conv / cost / build time) | retro |
|---|---|---|--:|--:|---|---|---|---|---|
| 1 | V0 | haiku | 1 | 2 | ce80225c | `/tmp/cummatrix-n1` | done (pilot) | conv −38%* / — / — (n=1, high variance) | clean; advance to n=5 |
| 2 | V0 | haiku | 5 | 2 | ce80225c | `/tmp/cummatrix-n5` | done | conv −17% / cost −11% / build −23% | real & significant (p<0.001); seed-vs-payback fix found |
| 3 | V0 | haiku | 5 | 3 | (current) | `/tmp/cummatrix-n5p3` | done | conv payback −10% (within noise); cost sign-FLIPPED vs #2 | n=5 underpowered; cost noise-dominated |
| 4 | V0 | haiku | 10 | 3 | (current) | `/tmp/cummatrix-n10more` | **CONTAMINATED — discard** | — | server outage produced 5/20 degraded warm builds (1-round, <$0.40 no-ops scoring phantom conv=0); inflated effect to −5/−6, not trustworthy |

| 5 | V0 | haiku | 10 | 3 | (current) | `/tmp/cummatrix-n10b` | done (clean) | conv payback −33% (gap −4.5); 0 degenerate | replaces contaminated #4 |

**POOLED n=20 (clean: #2 n5 + #3 n5p3 + #5 n10b; #4 discarded), C3 convention metric (round-1,
patience-invariant):** payback gap **−3.15 (−23%), z≈−4.41, permutation p=0.0001** — CONFIRMED
significant, direction unanimous (all 3 apps negative, warm < cold every app). **Magnitude
unstable batch-to-batch** (per-run gaps −2.4 / −1.2 / −4.5) — warm has high variance (±2.89 vs
cold ±1.36) because recall is probabilistic (sometimes eliminates a convention entirely, sometimes
misses). Quote as "~15–25%, significant", not a precise point. **C1/C2 (cost/time) remain
noise-dominated at n=20 — not a defensible finding. C4/C5/C6 not exercised.**

\* n=1 magnitude was a high-variance draw; n=5 settled C3 at ~−17% per-app (−12% cold-anchored chain).

### Planned next
- [ ] V0 sonnet n=5 (after run #3 retro passes) — ~$110
- [ ] V0 opus n=5 — ~$400
- [ ] V1 (please-driven) — model/n TBD
- [ ] V2 (CLAUDE.md advice) — model/n TBD
- [ ] V3 (combination) — model/n TBD
- [ ] Wire C4 / C5 / C6 probes into a run (currently separate/unexercised)

---

## Results detail

### Run #2 — V0 · haiku · n=5 · patience=2 (`/tmp/cummatrix-n5`)

Amortized economics (warm vs cold; negative = memory better):

| segment | conv (count) | rounds | build (s) | cost (USD) | tokens (Mtok) | total active time (s) |
|---|--:|--:|--:|--:|--:|--:|
| seed (app1) | −17% | +0% | −20% | **+81%** | **+136%** | +4% |
| payback (2–3) | **−17%** | **−6%** | **−23%** | **−11%** | **−13%** | **−5%** |

Read: app1 is a one-time investment (warm pays +81% cost / +136% tokens to *write* memory); apps
2–3 are net wins on every axis. C3 cut significant (permutation p=0.0004 all / p=0.0067
converged-only); cold convention count is a near-point-mass at 7.0/app.

### Run #3 — V0 · haiku · n=5 · patience=3 (`/tmp/cummatrix-n5p3`)

_Running. C3 is stall-invariant (won't move from #2); C1/C2 + convergence refresh under the
recalibrated stall. Fill in from `aggregate.py --root /tmp/cummatrix-n5p3` when complete._

---

## Decisions & lessons (chronological)

- **2026-06-21** — Recalibrated stall early-stop patience 2→3 (it cut 14/30 builds with budget to
  spare; original motivation was the cmd/-layout build bug, since fixed). Reporting fixes: per-build
  convergence (not the all-3-product), and the amortized seed-vs-payback table. validate.py 57/57.
- **2026-06-21** — Headline must be the **cold-anchored chain (~−12%)** or the **amortized payback
  view**, not the per-op 3× number (~−17%) which credits memory for the un-seeded app1.
- **2026-06-21** — Two independent n=5 draws (#2 p=2, #3 p=3) showed **C1/C2 (cost/time) are
  noise-dominated** — cost gap flipped sign (#2 −0.32 within noise → #3 +1.35) because build cost is
  round-count-dominated (2–8 rounds/build). Retracted run #2's "warm cheaper −11%" — single-draw
  over-confidence. Only C3 (conventions) has signal, and even it needs pooling/more n.
- **2026-06-21** — `convention_statements` confirmed **round-1-based** (harness.py:814 = round-1
  convention failures, the say-once metric) — NOT round-count-inflated. Its variance is genuine
  first-draft variance (cold isn't a fixed 7/app point-mass — the model sometimes gets a convention
  right unaided). Patience-invariant, so poolable across runs.
- **2026-06-21** — Attribution note: warm seed (app1, empty memory) still beats cold (6.2 vs 7.0),
  so part of the benefit is the recall-instruction *priming*, not memory content. Joe's call: the
  product-level cold-vs-warm comparison is the right question (the recall step IS part of engram) —
  no warm-empty control arm needed; bump n instead.
- **2026-06-21** — Parallelism: dropped the cold path's artificial DAG dependency (cold writes no
  vault — its 3 apps share nothing, so chaining them was needless serialization). Peak parallelism
  rises from 10 (regime×trial chains) to ~20 (15 independent cold ops + 5 warm chain-frontiers).
  Warm stays sequential (app2 recalls app1's learned notes — a real data dependency). Run with
  `--workers 20`; >20 buys nothing (only ~20 ops ever ready at once).
- **2026-06-21** — Op flow confirmed: per app, **instruct(+recall-first) → build/iterate → learn**;
  across the chain `learn(N)+ingest(N) → recall(N+1)`. `recall_s` is round-1 wall (recall + first
  draft), not isolated recall latency.
