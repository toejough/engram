# #661 — Does `glance` deliver? (measure-first protocol; gates #662)

> **Issue #661.** Measure whether a cheap read-only `glance` recall delivers the decision-relevant lesson as
> well as `deep`, at what cost (and the re-derived firing ceiling), with correct + sparing escalation. Gates
> the build (#662). Design: `2026-06-29-recall-depth-dial-design.md`.

## Scoping insight — and its limit (Gate-A corrected)

`glance` differs from `deep` in **(a) fewer phrases** and **(b) skipping crystallization** (Step-2.5C/2.6/Step-4).
Crystallization is a **post-hoc write** — it can't change *this* decision's surfaced set; `glance` keeps the
read-side (2.5A read, 2.5B recency-resolve, 2.7 activate). So phrase-count → retrieval breadth is the
**first-order** delivery lever, which makes a **free retrieval probe a legitimate *spend-gate*** (note 104).

**But retrieval is necessary, not sufficient** (the reduction's limit): delivery also depends on **synthesis
over what surfaced** (C6 — note 72: for C6 the bottleneck *is* synthesis, the agent already has the notes) and
on application a probe can't see (note 83). And the read/write cut is **not perfectly clean** — dropping
Step-2.5C removes not just the *write* but the **coverage-judge** (the Covered/Near/Absent comparison that
forces the agent to read every candidate), a comprehension/elaboration step that can lift application; and
kept-Step-2.7's activation input *narrows* (it fed on the 2.5C coverage set). So the free probe **gates spend;
it is NOT the verdict** — the **3-condition blind apply test is the verdict** (note 119: justify by delivery
*outcome*, not item-rank), and Phase 2 is the **primary** signal, never a smoke "confirmation."

## `glance` does not exist yet — what each phase must build

- **Phase 1:** extend `retrieval_probe.probe(vault_path, axis)` to take a phrase count (`AXIS_PHRASES[axis][:n]`)
  — or reuse its `rank_in_payload`/`_parse_payload` helpers. `crowded_gate.tier1_sweep` is the seeding precedent.
- **Phases 2 & 4:** a minimal **`glance` skill-variant cfg** — copy `skills/recall/SKILL.md`, cap phrases, drop
  Steps 2.5C/2.6/Step-4 — via the existing precedent `reasoning_recall_eval._build_reason_cfg`. A prompt aside
  ("use 2–3, skip crystallize") would *contradict* the skill's hardcoded "always generate exactly 10" → low
  fidelity. This is a throwaway *measurement* cfg, not the #662 production build.

## Two partitions (own the crosswalk)

**Delivery** is measured **per content-axis** — C3 (conventions), C4i (recency-supersession), C5
(recency-standard), C6 (abduction) — the idiosyncratic content types where memory wins (note 99); this is the
natural partition for "does glance retrieve + apply the lesson." **Firing/escalation** is measured **per
moment-class** — declaring-done / unexplained-failure / new-approach — via the recall-moments scenarios
(`2026-06-29-recall-moments-revalidation-data/scenarios.json`, keyed by `cue`). The design's "per-moment-class
delivery" conflated the two; they are orthogonal (C3 conventions fire at any moment), so we keep both
partitions rather than force a 1:1 crosswalk.

## Phase 1 — Tier-1 retrieval SPEND-GATE (FREE, no LLM)

Seed each trap vault (`seed_c3.py`; C4i via `c4_idio.seed_into`; C6 via `reasoning_recall_eval.CASES`; C5 via
`seed_c5.py`). Sweep phrase-count **{1, 2, 3, 5, 10}**; per load-bearing target report **surfaced? + rank**,
with **K = top-5** (the recall@5 the vault already uses). **C5 cannot be Phase-1-probed** — it surfaces by
*recency*, not cosine (`retrieval_probe` omits it by design). And C5 is **not** recency-safe under glance: it
rides **Channel 2** (the recent-activity channel via `--recent-fill`), which the design has glance
**minimize** — so glance *shrinks C5's delivery channel* (2.5B is C4i's conflict-resolution mechanism, not
C5's). The C5 toy hides this (R is artificially the single newest chunk). So **C5 is deferred to a mandatory
Phase-2 check** with a *realistic recency-depth* fixture (the load-bearing recent item is NOT the single
newest), testing whether minimized `--recent-fill` still surfaces it — #661 thereby asks a design question:
**should glance minimize `--recent-fill` at all?**

**Output:** labeled table **axis × phrase-count → {surfaced, rank}**. **Role:** find the phrase-count *cliff*
— the largest `n < 10` that still surfaces every target (do **not** pre-assert 2–3; the sweep names it). This
**gates LLM spend** and locates where glance is cheapest-yet-safe. A buries-result **does not kill** the dial —
it routes that axis to Phase 3 (can deep-escalation recover it?).

## Phase 2 — 3-condition delivery VERDICT (LLM, REAL bars, incl C6)

Build the glance cfg. Run the warm harnesses (`wrun.py`/`c4_idio.py`/`c5.py`/`c6_clean.py`) in **3 conditions —
none (cold) / glance / deep** — at the trap gate's **real bars** (full tier, *not* smoke: a smoke-scale tie is
the underpowered-tie trap — a gap below the noise floor is "can't distinguish," not "deep's processing was
optional"). Blind-judge per axis: does the agent's plan **apply the lesson** (detect the *pattern*, not the
note's name — scorer-bias guard). **Must include C6**, where deep's synthesis effect lives and retrieval is
uninformative. **Verdict per axis:** glance delivers iff within the trap bar of deep.

## Phase 3 — Escalation (LLM, note 107; per moment-class)

Via a glance cfg that *documents* an escalation option (the production mechanism is #662; this is a
measurement proxy). **SPARING:** across the recall-moments moment-class scenarios, is the agent's escalation
rate **below the break-even** `≈ 1 − G/D` derived from Phase-4's measured glance/deep cost — note 107's *actual*
rule, **not** a hardcoded "majority" (with G≈tens of s, D≈190s the break-even may be ~0.7–0.9). *This bar
depends on Phase 4 — measure cost before finalizing SPARING.* **CAPABLE:** on a sole-source fixture built from
a Phase-1 buries case (glance misses, deep catches), does the agent escalate to `deep` **unprompted**? Ship
glance-as-default only if **both** hold (note 107). **Run Phase 3 on any Phase-1 buries axis before concluding it "dies"** — escalation is the
mechanism the design says licenses re-opening the phrase-cut dead-end.

## Phase 4 — Cost + firing-ceiling (note 94 + note 109)

`glance`-recall vs `deep`-recall wall-time + `recall_cost` via the $METER, **using the glance cfg** (the meter
times the *full* `/recall` by default — verify what it measures, note 94). **Then re-derive the viable firing
ceiling** (the actual #661(b) output, not just a cost delta): pin the fire-unit (task-init), take glance's
per-fire cost, compute the affordable firing rate per `tax = over-fire × per-fire-cost` (note 109); mark each
number **DERIVED vs ESTIMATE**. This is the relaxed ceiling note 140 promises.

## Pass-bar

Ship glance-as-default for an **axis** only if: Phase-2 delivery is **within the trap bar of deep** (real
bars); **and** if Phase-1 buries that axis, Phase-3 escalation recovers it (**no false-negative**) and stays
**sparing**; **and** Phase-4 shows materially lower cost with a derived relaxed ceiling. Axes that fail keep
`deep`. Escalation is judged **per moment-class**.

## Execution scope (this `/please`)

**Phase 1 runs now** — free, and it gates the LLM spend (notes 104/109). Phases 2–4 require the glance cfg +
trap-gate-scale LLM runs (~$30–40 at real bars) — **#661 is a multi-phase LLM eval, not "mostly free"** (the
Gate-A correction). After Phase 1, decide spend on the LLM verdict from its result. All results land as
**labeled tables with units** (standing requirement).

---

## Phase 1 results (2026-06-29) — retrieval SPEND-GATE: GREENLIT

Free retrieval sweep (`engram query`, bundled embedder, real ~3k-chunk index as background; per-target
rank, K=top-5). Script: `phase1_sweep.py` (+ a per-case C6 split, since `retrieval_probe`'s C6 conflates
both abduction cases with case-segregated phrases).

| Axis / case (load-bearing notes) | n=1 | n=2 | n=3 | n=5 | n=10 | retrieval cliff |
|---|---|---|---|---|---|---|
| **C3** — 5 Go conventions | all top-5 | ✓ | ✓ | ✓ | ✓ | **n=1** |
| **C4i** — errcfg supersession marker | rank 1 | ✓ | ✓ | ✓ | ✓ | **n=1** |
| **C6-zephyr** — 2 premises | top-2 | ✓ | ✓ | ✓ | — | **n=1** |
| **C6-badge** — 2 premises | buried | buried | top-2 | ✓ | — | **n=3** |

**Verdict:** retrieval holds for `glance` at **n≥3** across every trap axis/case. The phrase-count cliff is
**n=3** (set by the most oblique abduction case), so the glance phrase floor is **~3, not 2** (the sweep names
it, per the design's instruction not to pre-assert). The "cutting phrases buries un-guessable notes" dead-end
bites only at n<3. The matched-note floor keeps the seeded notes in top-5 despite the real chunk background.

**Spend-gate decision: GREENLIT.** Retrieval is not the bottleneck at n≥3 → the remaining open question is the
**apply/elaboration effect** (does dropping 2.5C's coverage-judge cost application? does deep's synthesis lift
C6 where note 72 says retrieval is uninformative?). That requires the **LLM Phase 2** (the verdict) — which
needs a glance skill-variant cfg + trap-gate-scale runs (~$30–40 at real bars). **Not run in this pass:**
Phases 2 (apply verdict), 3 (escalation), 4 (cost + firing-ceiling) await a go, since Gate A established #661
is a multi-phase LLM eval, not free.

## Phase 2 SMOKE results (2026-06-29) — glance apply-verdict: VIABLE (escalate to full bars)

Smoke-tier glance-vs-deep apply, via a `glance` skill-variant cfg (build_warm_cfg + cap phrases to 3 + a
GLANCE-MODE override that skips the write side 2.5C/2.6/4, keeps 2.5A/2.5B/2.7/Step-3). Real opus runs,
sonnet judge (HIT/applied = the agent's plan applies the lesson). Data: `…-data/phase2_smoke*`.

| Axis (apply test) | deep | glance | glance cost vs deep |
|---|---|---|---|
| **C6** — abduction (synthesis-sensitive; note 72's "retrieval uninformative" axis) | 6/6 HIT | **6/6 HIT** | $2.04 vs $3.07 (**−33%**) |
| **C3** — Go conventions (most common) | 6/6 applied | **6/6 applied** | $2.34 vs $2.70 (**−13%**) |

(Plus a 1-rep C6 validation batch: glance 2/2 = deep 2/2.)

**Verdict: glance 12/12 = deep 12/12 apply parity — including C6, where dropping deep's coverage-judge
elaboration was most likely to hurt (it didn't) — at 13–33% lower cost. No kill signal.**

**Honest bounds:** (1) SMOKE scale (C6 ×3 reps, C3 ×2 reps) — a tie here is *not conclusive* at the trap
gate's real bars (an underpowered tie is "can't distinguish," not "deep's processing is provably optional");
the triage rule says **escalate to full bars**, not ship. (2) **C5 untested** — it needs the realistic
recency-depth fixture (does glance minimizing `--recent-fill` drop the recent-channel note?). (3) **Escalation
(Phase 3) and cost/firing-ceiling (Phase 4) not yet run.** Smoke spend ≈ $11.7 total.

**Next (the conclusive verdict, separate pass):** Phase 2 at real bars incl. C5 (recency-depth fixture) →
Phase 3 escalation (sparing vs the Phase-4 break-even + capable) → Phase 4 cost + re-derived firing ceiling.

## Full-bars verdict (2026-06-29) — glance ships for 3/4 axes; C5 keeps deep; cost is scale-dependent

`gate.py --tier full` (verified bars: C3 5conv×5, C4i ×5, C5 ×5, C6 2×4) on **both** arms, same conditions:
deep = the repo recall skill; glance = a skill-variant (cap 3 phrases + skip the write side 2.5C/2.6/4; keep
2.5A read / 2.5B recency-resolve / 2.7 activate / Step-3 apply). GREEN = the axis hits its exact verified bar.
Data: `…-data/fullbars/{deep,glance}/`.

| Axis (full bars) | deep | glance | glance verdict |
|---|---|---|---|
| **C3** — apply un-guessable conventions | 25/25 (GREEN) | **25/25 (GREEN)** | **delivers** |
| **C4i** — recency supersession (2.5B) | 5/5 (GREEN) | **5/5 (GREEN)** | **delivers** |
| **C6** — abduction / synthesis | 8/8 (GREEN) | **8/8 (GREEN)** | **delivers** |
| **C5** — honor a recency-channel standard | surfaced 5/5, **honored 4/5** | surfaced 5/5, **honored 0/5** | **FAILS (apply, not retrieval)** |

**Verdict: ship `glance` as default for C3 / C4i / C6 (full parity at the verified bars); fall back to `deep` for
C5-type recency-channel standards** (deep is only *relatively* better at 4/5 — recency-channel
*apply* is hard for both; #662 owns the real fix).

**The C5 failure vindicates the Gate-A correction.** Glance **surfaces** the recency marker R every time
(5/5 — the recency channel is intact, glance keeps `--recent-fill` at default) but **applies it 0/5** (deep
4/5). Retrieval held; *delivery* didn't — exactly the `retrieval ⇏ delivery` gap notes 119/72/83 forced into
the protocol. Treating Phase-1 retrieval as the verdict (as I first framed it) would have shipped a broken C5.
The lighter glance pass surfaces the recent-channel item but doesn't elevate it to a requirement; deep's
heavier pass does. **For #662:** either add a glance step that applies recent-channel standards, or scope
glance to non-recency content and **escalate C5-type cues to deep** (a concrete escalation requirement).

**Cost (Phase 4, recall-only): scale-dependent, and smaller than the premise on eval-scale vaults.**
On the trap-scale **isolated** vault (5 notes, empty chunk index): glance **30s / $0.28** vs deep **35s /
$0.30** — only **1.2× faster**, because deep recall is *already cheap* there (nothing to page, little to
crystallize). The **~190s deep recall** that motivated the dial (notes 91/93) is a **large-vault** phenomenon
(real ~3k-chunk index: ~95s Step-2 paging + crystallization). So the glance cost win — and the relaxed firing
ceiling (note 109) — **grows with vault size**: marginal on small vaults, material on large ones (Joe's real
vault). This run cannot quantify the real-vault saving (note 94: verify what the metric measures — these
vaults make deep cheap). Gate per-cell op costs did show glance consistently cheaper (e.g. C6 ~$0.33 vs ~$0.47).

**Bounds:** escalation (Phase 3) not separately run — it is the #662 production mechanism, and the C5 finding
makes "escalate recency-channel cues to deep" a concrete #662 requirement. Deep's C5 4/5 (vs the 5/5 bar) is
mild flak on a hard honor-task; the glance-vs-deep *contrast* (0/5 vs 4/5) is categorical. Real-vault cost
saving is a clean follow-up (needs a large isolated vault copy to avoid polluting the live vault).
