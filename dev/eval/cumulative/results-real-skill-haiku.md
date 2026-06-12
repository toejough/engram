# Cumulative cross-app memory accumulation — REAL-SKILL run (haiku)

Engram SHA: `d4111831a184` · date: 2026-06-11 · model: **haiku** (`claude-haiku-4-5-20251001`) · trials: [1,2,3,4,5] · price sheet: 2026-06-02

> Same real-skill harness as `results-real-skill-opus.md` (real `/recall` + `/learn` fire end-to-end,
> no inlined proxy). 45/45 cells, no rate-limit/timeout/vault-missing flags. Companion to the opus run;
> sonnet still pending.

---

## VERDICT — haiku is too weak and too noisy for this eval to separate lazy from eager

**1. Memory helps haiku, but weakly and inconsistently.** Memory-active convention fails (app2+app3):
cold **12.6** → lazy **7.8** / eager **8.6**. The benefit is real but ~⅓ the size of opus's (opus went
13→~2) and very noisy: per-cell values flip between cold-level 7 (memory ignored) and 1–2 (memory used)
within the same arm — the weak model leverages recalled conventions only some of the time.

**2. Lazy vs eager: indistinguishable, dominated by noise.** Mem-active fails lazy 7.8 vs eager 8.6,
but the per-trial diff (eager−lazy) = **[4, −11, 4, −1, 8]** — swinging ±11 with per-arm sd ~3–4 on a
mean gap of 0.8. There is **no separable difference**, and crucially **the proxy run's "lazy clearly
wins for haiku" claim does NOT reproduce on the real skills** — that earlier signal was a property of
the inlined-recall proxy, not the shipped `/recall`.

**3. The dominant haiku finding isn't lazy-vs-eager — it's that haiku barely works here.** It converged
**0/5 (cold), 0/5 (lazy), 2/5 (eager)** full 3-app chains within the round budget, and its in-session
`/learn` **failed to fire / capture an L1 episode in 5 of 10 warm chains** (`learn fired=None`, L1=0).
Unreliable convergence and unreliable episode capture are the headline weak-model problems; the L2
policy is a second-order question the data can't reach underneath that noise.

| metric (mean per 3-app chain) | cold | real.lazy | real.eager |
|---|--:|--:|--:|
| convention fails, mem-active (app2+app3) ↓ | 12.6 | 7.8 | 8.6 |
| converged full chains | 0/5 | 0/5 | 2/5 |
| $ / chain | 5.14 | 5.35 | 4.96 |
| L2 in final vault | 0 | 0.2 | 8.8 (spiky: [0,23,14,0,7]) |
| L1 episodes in final vault | 0 | 2.4 | 2.2 |

**Net:** for haiku, this eval **cannot distinguish lazy from eager** (memory helps weakly; the L2 policy
is buried under convergence + capture noise at n=5). The lazy deployment isn't *contradicted* for haiku
— lazy is no worse and far leaner (L2 0.2 vs 8.8) — but the proxy's positive haiku case for lazy is
**not** confirmed by the real skills. Note the cost picture also differs from opus: no warm>cold
inversion here (~$5/chain everywhere), because haiku's non-converging builds simply stop early rather
than running the long multi-turn recall+learn loop that drove opus's premium.

---

## Per-(regime, app, trial) — `round1_convention_fails`

| regime | app | t1 | t2 | t3 | t4 | t5 | mean |
|---|---|--:|--:|--:|--:|--:|--:|
| cold | app1 | 7 | 7 | 7 | 7 | 7 | 7.0 |
| cold | app2 | 7 | 7 | 7 | 0\* | 7 | 5.6\* |
| cold | app3 | 7 | 7 | 7 | 7 | 7 | 7.0 |
| real.lazy | app1 | 7 | 7 | 7 | 7 | 7 | 7.0 |
| real.lazy | app2 | 7 | 7 | 2 | 3 | 1 | 4.0 |
| real.lazy | app3 | 2 | 7 | 2 | 6 | 2 | 3.8 |
| real.eager | app1 | 7 | 7 | 7 | 7 | 7 | 7.0 |
| real.eager | app2 | 7 | 2 | 6 | 2 | 7 | 4.8 |
| real.eager | app3 | 6 | 1 | 2 | 6 | 4 | 3.8 |

app1 is pinned at **7.0 in every regime** — haiku fails ~all conventions on a first draft regardless of
memory (empty seed for app1), so the warm-vs-warm noise reference is 7.0 = 7.0 = 0 gap, same as opus.
\* **cold-app2-t4 = 0 is an artifact**: that build did not converge (final 10/17) and read 0 convention
fails while failing on other axes; it drags cold-app2's mean down (without it cold-app2 = 7.0). cold's
true mem-active number is therefore closer to **14**, widening the cold→warm memory effect, not narrowing it.

---

## Honest caveats

- **n=5, and far noisier than opus** — per-arm sd ~3–4 on mem-active fails; do not read the 7.8-vs-8.6
  point gap as a result.
- **Episode-capture failures are real** (`learn fired=None`/L1=0 in 5/10 warm chains) — a weak-model
  learn-reliability problem worth its own investigation, separate from the L2 policy.
- **Convergence is poor** (0–2/5) — many chains plateau below the bar, so "fails" counts partly reflect
  unfinished builds, not pure say-once behavior.
- Same keyword/label scorer caveat as the opus doc; sonnet arm still pending.

Raw cells: `dev/eval/cumulative/runs/2026-06-11-haiku-real-skill/`.
