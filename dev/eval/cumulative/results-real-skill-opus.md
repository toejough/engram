# Cumulative cross-app memory accumulation — REAL-SKILL run (opus)

Engram SHA: `d4111831a184` · date: 2026-06-11 · model: **opus** (`claude-opus-4-8`) · trials: [1,2,3,4,5] · price sheet: 2026-06-02

> **This is the first run that drives the REAL `/recall` and `/learn` skills end-to-end.** Every prior
> cumulative result (including `results-lazy-l2-opus.md`) measured a *proxy*: the harness inlined the
> recall procedure into the build prompt and the `/recall` `SKILL.md` never fired. Here the headless
> agent invokes its actual `/recall` skill (verified: `"skill":"recall"` in every warm transcript,
> `recall_fired=1`) and its actual `/learn` skill in-session, with **no episode/tier overrides**. The
> lazy-vs-eager difference is now the difference between the *shipped skills*, not between two prompts.
> Cells where the Skill tool did not fire are discarded, not scored.
>
> **Scope: opus only.** The haiku + sonnet arms of the same n=5 matrix are pending (full 3-model run).
> Do not generalize these numbers to weaker models — the proxy run found the lazy/eager gap was
> model-dependent (flat for opus, large for haiku), so the cross-model picture is genuinely open.

---

## VERDICT — cold vs lazy vs eager, real skills, opus n=5

**1. Memory works, and the effect is large (the robust headline).** On the two memory-active apps
(app2+app3, where a populated vault is recalled), first-draft convention failures fall from **13.0
(cold)** to **1.6 (lazy)** / **2.8 (eager)** — roughly an order of magnitude. This dwarfs the noise
floor (below) and is the finding to lead with.

**2. Lazy ≈ eager on quality — not distinguishable at this n.** Memory-active convention fails: lazy
**1.6** vs eager **2.8** per chain (diff 1.2, pooled SE 0.98 → **~1.2 SE**; needs ≥2 to claim a real
difference). The *entire* gap is a single high-variance eager trial: per-trial diffs (eager−lazy) =
**[1, 6, −1, 1, −1]** — drop the one t2=6 eager run and the arms are **identical** (mean diff 0). The
point estimate favors lazy, but at n=5 this is **"can't tell them apart," not a proven equivalence** —
a real equivalence claim would need larger n or an equivalence test against a preset margin. What the
data *does* rule out: eager's pre-crystallized L2 did **not** visibly *improve* first-attempt convention
adherence over lazy's defer-to-recall.

**3. Lazy is cheaper and far leaner.** $15.94 vs $18.70 per 3-app chain (**−15%**), and the final
accumulated vault holds **~⅓ the L2 notes** — 5.0 vs 13.2 — with **comparable L1 episodes** (6.0 vs 5.2).
Eager's ~2.6× L2 volume bought nothing measurable here and is a standing retrieval-precision and
vault-bloat liability as the vault grows.

**Net for the deployed decision:** the real skills reproduce what the hunch bet on — **defer-L2-to-recall
matches eager on first-attempt convention quality at lower cost and a third of the L2 volume.** The
proxy run's lazy conclusions were about a paraphrase; this one is about the skill, and it lands the same
way for opus. Keep lazy deployed. **But see the metric-sensitivity caveat — this is "L2 crystallization
didn't matter *for first-attempt convention-following*," not "L2 is worthless."**

### The cost inversion — surfaced loudly

**Warm costs MORE than cold, not less.** cold $7.90/chain → lazy $15.94 (**2.0×**) → eager $18.70
(**2.4×**). The real skills are multi-turn: `/recall` fires, reads the vault, synthesizes impact on the
plan; `/learn` runs in-session and writes notes. That machinery is a per-session premium. Memory here
**front-loads correctness** (an order-of-magnitude fewer conventions to teach on app2/app3), it does
**not** save money on a 3-app horizon. The cost case for memory is "teach each convention once, ever,"
amortized over many convention-sharing apps — not a per-build saving. On a one-off or two-app job, cold
is the cheaper floor.

| metric (mean per 3-app chain) | cold | real.lazy | real.eager |
|---|--:|--:|--:|
| **convention fails, memory-active apps (app2+app3)** ↓ | **13.0** | **1.6** | **2.8** |
| total $ / chain | 7.90 | 15.94 | 18.70 |
| warm/cold cost ratio | 1.0× | 2.0× | 2.4× |
| L2 notes in final vault / chain | 0 | 5.0 | 13.2 |
| L1 episodes in final vault / chain | 0 | 6.0 | 5.2 |

---

## Metric sensitivity — what a lazy=eager tie does and does NOT mean

The scored metric is `round1_convention_fails`: project conventions the agent failed to follow on its
**first-draft** submission (the simulated reviewer flags them; memory's job is to surface them so the
first draft already complies). A lazy≈eager tie on this metric means precisely:

> **Pre-crystallized L2 facts did not improve first-attempt convention adherence over what the raw L1
> episodes already provide.** For *this* task, the conventions are recoverable from episodes alone, so
> eager's synthesis step was not the deciding factor.

It does **not** establish that L2 crystallization is valueless. This eval **cannot** test what eager L2
is actually for: cross-domain transfer, queries that need a *synthesized fact* rather than a recounted
episode, longer horizons where consolidation compounds, or retrieval precision/cost as the vault scales
(where eager's ~2.6× L2 volume could help *or* hurt). The lazy bet is validated **for convention-following on
a short same-domain chain**; the broader "is eager L2 ever worth it" question is out of scope here and
needs a task whose success depends on synthesized facts, not episodes. (cf.
`feedback_check_metric_sensitivity_before_crowning_cheaper_arm`.)

---

## Per-(regime, app, trial) — full clean n=5

`round1_convention_fails` (lower = better). app1 recalls an **empty seed** (memory OFF by construction);
app2/app3 recall the accumulated vault (memory ACTIVE).

| regime | app | t1 | t2 | t3 | t4 | t5 | mean |
|---|---|--:|--:|--:|--:|--:|--:|
| cold | app1 | 7 | 6 | 7 | 7 | 6 | 6.6 |
| cold | app2 | 7 | 7 | 6 | 7 | 6 | 6.6 |
| cold | app3 | 6 | 7 | 6 | 6 | 7 | 6.4 |
| real.lazy | app1 | 6 | 4 | 4 | 5 | 4 | 4.6 |
| real.lazy | app2 | 1 | 0 | 2 | 1 | 2 | 1.2 |
| real.lazy | app3 | 0 | 0 | 0 | 0 | 2 | 0.4 |
| real.eager | app1 | 5 | 5 | 5 | 4 | 4 | 4.6 |
| real.eager | app2 | 1 | 0 | 1 | 0 | 3 | 1.0 |
| real.eager | app3 | 1 | 6 | 0 | 2 | 0 | 1.8 |

**Noise floor / scaffolding effect (app1, memory-OFF in every regime).** app1 recalls an empty vault, so
stored memory cannot act. The right warm-vs-warm noise reference for the lazy-eager contrast is
**lazy-app1 vs eager-app1 = 4.6 vs 4.6** — essentially zero gap (both arms fire recall+learn), consistent
with the warm-warm tie above. Cold-app1 (6.6) sits **2.0 above** the warm app1s; that gap is the
recall/learn *scaffolding* (invoking `/recall` even on an empty vault, plus in-session `/learn`, primes
first-draft convention attention) plus run-to-run variance — it is **not** stored-memory value, and it is
the *wrong* yardstick for a warm-vs-warm comparison. The cold→warm drop on memory-active apps (13→~2) is
an order of magnitude beyond either reference — robustly real. The eager-app3 t2=6 is a genuine
high-variance trial (memory present: 11-note vault, recall fired, converged 18/18), not an artifact to
discard — it is simply the single trial that creates the entire lazy-eager point gap.

---

## Data integrity note (the t2 eager chain)

During an earlier backfill attempt, the `opus-t2-app2-real.eager` result JSON and its accumulated vault
were deleted, leaving the chain inconsistent (app3 present, its app2 antecedent gone). Forensics on the
preserved `app3-t2-eager` cell showed it had run against a **real, populated vault** (`vault_notes_total=11`,
`recall_ok=true`, converged 18/18) — so its 6 convention-fails is a **legitimate eager trial**, not a
broken-chain artifact. Fix: rebuilt the lost `app2-t2-eager` cell fresh (converged, fails=0, built on the
intact app1 vault) and **kept the original app3-t2-eager=6** rather than resampling a real measurement
away. Final dataset is a clean n=5 with no integrity flags. (An earlier draft of this analysis wrongly
called the 6 an artifact; the recall-content check corrected that.)

---

## Honest caveats

- **opus only.** haiku + sonnet pending the full 3-model run. The proxy run found lazy-vs-eager is
  model-dependent — do not generalize.
- **n=5.** The lazy-eager tie is a not-distinguishable result at this sample, not a proven equality.
- **3-app same-domain chain.** L2 composition / cross-domain transfer is only lightly exercised; this is
  exactly where eager L2 might earn its keep and this run can't see it.
- **Scorer is keyword/label-based**, not semantic — it detects whether a stated convention recurs, by
  pattern. A semantic rescore of learn-capture quality is deferred (GitHub issue).
- **Cost is real-skill cost**, dominated by recall+learn turn counts; it reflects the current skill
  implementations, which are not cost-optimized.

Raw cells: `dev/eval/cumulative/runs/2026-06-11-opus-real-skill/` (45 build cells + manifest).
Reproduce: `python3 /tmp/analyze_real.py` and `/tmp/analyze_cost.py` (archived alongside).
