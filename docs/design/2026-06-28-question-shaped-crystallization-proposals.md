# Question-shaped crystallization + reasoning-from-memory — fixes to evaluate

> **Ask (Joe, 2026-06-28).** Propose fixes to evaluate for question-shaped crystallization, considering note
> quality + the situations we want notes surfaced in + the reasoning research — ideally so we can *reason from
> memories toward better problem-understanding and correct outcomes in fewer passes.* This is the diverse,
> evaluable set: **9 proposals + a scoped PARK**, panel-scored, with a recommendation. Data trail:
> `2026-06-28-crystallization-proposals-data/`. Plan + gate trail: `docs/superpowers/plans/2026-06-28-question-shaped-crystallization-proposals.md`.

## The frame (what the reasoning research scopes)

The floor (shipped) makes a good note *surface*; these fixes make notes *worth* surfacing and capture
reasoning *into* memory. The reasoning research is **scoped**: notes 72/73 measured ~0 headroom for
*emergent-synthesis (A+B→C)* + *graph-expansion* retrieval scaffolds (opus reasons fine over what surfaces) —
so those are **PARK**. But they did **not** measure *diagnostic/abductive surfacing* (symptom→cause = the
"understand problems" goal), so that thread is **open** (measure it: angle 8). The live lever is the **write
side** (note 74: persist validated synthesis; note 76: a persisted conclusion is reusable by a *weaker* model
— a real fewer-passes/cost path).

**Terms used below.** *Diagnostic surfacing* — does the right cause/lesson note surface when you query with a
*symptom* ("X is failing / slow / wrong"), not a topic? (untested; angle 8 measures it). *`engram resituate`*
— an existing command that rewrites a note's `situation` handle + body and re-embeds its vector.
*`inference_mode`* — a proposed additive frontmatter tag (`abduction`/`deduction`/`induction`) marking the
reasoning a synthesis note supports. The audit's *40% → ≥65%* is the share of cluster-driven notes that are
*question-shaped AND useful* (correction-driven notes hit 79%, the ceiling).

## First-wave results (2026-06-28) — measurement-first paid off

Ran the recommended first wave. The measurements *redirected* the work — two deflations, one strong win:

- **#8 diagnostic-surfacing probe — NO gap.** 75 symptom→cause probes: real-path note recall@5 = **0.99**.
  Diagnostic retrieval is healthy (77% of notes are already diagnostic-shaped + the floor surfaces them). **#4
  dropped** (no build needed). Data: `…-data/diagnostic_surfacing_results.json`.
- **#1 question-shaped prose rule — RED baseline PASSED, not shipped.** A clean micro-test (6 clusters) showed
  fresh agents already write question-shaped handles **5/6 without the rule**. Per the writing-skills Iron Law
  (don't author against a passing baseline), the rule wasn't shipped; the audit's 40% gap is a session-load /
  old-notes artifact, so the lever (if pursued) is the deterministic retroactive `resituate`, not a buried
  prose rule. Data: `…-data/q-shaped-handle-microtest.json`.
- **#7 weaker-model reuse — VALIDATED and generalized.** 3-arm eval (opus-warm / sonnet-warm / sonnet-cold):
  **sonnet + memory fully matches opus + memory** on C3 (15/15), C4i (3/3), C6 (6/6); sonnet *without* memory
  fails (C4i/C6 cold 0/N); C5 inconclusive (opus baseline flaked, honored 0/3). Sonnet ran ~25–30% cheaper
  per trial. **Finding: memory democratizes reasoning across model tiers** (vault note 135) — route
  memory-backed reasoning to a cheaper tier for a cost + fewer-passes win. **Next: wire into the `route` skill.**

## The set — panel-scored (3 judges: impact / evaluability / evidence-discipline)

| # | proposal | serves | mechanism (1-line) | eval | effort | score | rating |
|---|---|---|---|---|---|--:|---|
| **8** | **Diagnostic-surfacing adequacy probe** | understand-problems (surfacing) | run `score_probe.py` on a *symptom-first* query set; measure if causal notes surface when you describe what's wrong | cheap | S | **4.7** | **CONTENDER** |
| **1** | **Question-shaped handles + retroactive `resituate`** | quality · fewer-passes | add learn-Step-2's "phrase as a future task" rule to recall Step 2.5; one-time `engram resituate` pass over topic-shaped notes | cheap | S | **4.3** | **CONTENDER** |
| **3** | **Write-time quality gate** | quality | a 3-criterion self-check (question-shaped? actionable? distinct?) in recall Step 2.5's Absent branch — rewrite-or-discard before `engram learn` | cheap | S | 4.0 | CONTENDER |
| **2** | **Step-4 mandatory trigger + chain body** | reason-from-memory · fewer-passes | make recall Step 4 mandatory-evaluate (silent-skip forbidden) + a Problem/Evidence/Reasoning/Conclusion/Falsifier body template | cheap | S | 3.7 | CONTENDER |
| **5** | **Failure-mining capture in learn** | quality · coverage | a learn Step-2.0 scan that crystallizes session self-corrections into question-shaped notes (closes the 68% uncovered) | moderate | M | 3.3 | CONTENDER |
| **6** | **Abstraction-instance pairing** | quality | every note body states the generalizable *principle* + a labeled *concrete instance* (matches abstract + specific queries) | cheap | S | 3.0 | CONTENDER |
| **7** | **Weaker-model reuse (stamp + 3-arm eval)** | fewer-passes · cost | machine-readable `inference_mode`/`derived_by` frontmatter on Step-4 notes; opus-vs-sonnet+persisted-conclusion A/B on diagnostic tasks | moderate | M | 3.0 | CONTENDER |
| **4** | **Diagnostic note *shaping*** (symptom-trigger + `inference_mode`) | understand-problems | force "when [symptom] appears" situation phrasing for diagnosis notes + an `--inference-mode abduction` flag | moderate | M | 2.7 | **CONDITIONAL** |
| **99** | **PARK — emergent-synthesis / graph-expanded retrieval** | (parked) | A/B-gated only; record the exact warm-vs-warm+scaffold A/B that must show headroom to un-park | — | S | 3.7 | **PARK** |

**Legend:** *score* = mean over the 3 judges on a 0–5 scale (impact × evaluability ÷ risk); *rating* = panel
majority (CONTENDER ≥ ~3.0 with respects-evidence; CONDITIONAL = gated on another proposal; PARK = evidence
says wait). *serves* = the **primary** goal (full mappings in "Ask → goal coverage" below). Full mechanisms +
per-proposal eval recipes in the data trail's `proposals_raw.json`.

## Recommendation — start with TWO (a fix + a measurement); #3 fast-follows #1

The panel's strongest signal: **angle 4 (shaping notes for diagnosis) is premature until angle 8 *measures*
whether a diagnostic-surfacing gap even exists** — so the right order leads with measurement + the confirmed
fix, not with building on an unmeasured assumption. **Two genuinely independent first moves, both cheap, run
in parallel:**

1. **#1 — question-shaped handles (skill-only; optional retroactive `resituate`).** The direct fix for the
   *confirmed* gap (audit's cluster-driven 40% → target ≥65%) and the #1 ROADMAP lever. **The fix is pure
   recall-SKILL.md prose** — add learn Step-2's "phrase the situation as the future task/failure" rule to
   recall Step 2.5; that shapes all *future* notes, **zero binary code.** *Optional* one-time cleanup of the
   ~24 existing topic-shaped notes reuses the **existing** `engram resituate` command (rewrites situation +
   re-embeds in one shot — needed only because changing the situation text without re-embedding leaves the old
   vector). Skip or defer the resituate batch and #1 is skill-only; if run, gate it with a 5-note pilot probe.
2. **#8 — diagnostic-surfacing probe.** A pure *measurement* (reuse `score_probe.py` on a symptom→cause query
   set) that settles the one explicitly-open goal-A question and **gates #4** — if diagnostic notes already
   surface, #4 is moot; if not, that's a big new finding.

**Fast-follow to #1 (not a parallel third):** **#3 — write-time quality gate** enforces #1's rule with a
rewrite-or-discard self-check before `engram learn` (also skill-prose). Ship #1's rule first; add #3 only if
new-note quality still lags. It builds on #1, so it sequences *after*, not alongside.

**Second tier (reason-from-memory / fewer-passes):** **#2** (capture the reasoning *chain*, not just a fact)
and **#7** (weaker-model reuse — the one proposal that also moves the *cost* track: if a persisted conclusion
lets sonnet match opus on diagnostic tasks, that's fewer passes *and* cheaper). **Continuous (no wave — fold
into crystallization whenever it runs):** #5 (coverage capture) and #6 (abstraction pairing). **Conditional:**
#4 (only if #8 shows a gap). **Parked:** #99 (behind its A/B). (#5 scores 3.3 > #7 3.0 but is *continuous*, not
*sequenced after* — the tiers are priority/dependency groupings, not a strict order.)

## Ask → goal coverage (all three threads represented)
- **note quality** → 1, 3, 5, 6
- **understand problems** → 8 (measure) + 4 (shape, conditional)
- **reason from memories / fewer passes** → 2 (capture the chain), 7 (weaker-model reuse), 1 (right note first time)
- **the reasoning research, honored** → 99 parked (scoped to what 72/73 measured); diagnostic surfacing measured, not assumed

## Honest notes
- **#1 and #3 overlap** (a rule vs a stricter gate) — sequence them, don't double-build.
- **#4 and #7 share** the `inference_mode` frontmatter addition (note 46: additive, no schema bump) — if both
  advance, build the field once.
- **#8 is measurement, not a fix** — it produces a number, not a change; its value is settling the open thread
  cheaply before investing in #4.
- This is *propose, don't pick*: the recommendation is a *position inside* the set, not a pruning — every angle
  is here with an honest rating so you can evaluate them yourself.
