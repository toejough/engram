# C7 — lever-recheck (anti-amnesia) eval

**Issue:** #654 · **Design:** `../../../docs/design/2026-06-24-recall-miss-and-cost-round3-findings.md` §4 ·
**Run evidence:** `RESULTS.md`.

Measures whether an agent, when it recommends an action that matches a **closed** lever (one already
tried and rolled back), re-proposes it as fresh (**AMNESIA**) or acknowledges the prior outcome
(**RECONCILED**). Named after the real miss: an agent re-proposed "cheaper-tier recall" — a lever built,
measured at −14%, and rolled back the same day (vault note 80).

## Two independent metrics

- **OUTCOME** — did the final recommendation reconcile or commit amnesia? Measured by
  `lever_recheck_scorer.py` (LLM judge, default-AMNESIA).
- **MECHANISM** — did the skill issue a lever-keyed query that would surface the disproof? Measured by
  the `stub_engram` query log. The diagnostic `note_surfaced` flag separates a **retrieval** miss
  (disproof never returned to the skill) from a **synthesis** miss (returned but ignored); it is
  recorded alongside the verdict, never folded into pass/fail (per
  `eval-end-metric-conflates-retrieval-vs-synthesis`).

## Components

| File | Role |
|---|---|
| `fixture1/` | The behavioral trap: an "Orchestra" pipeline whose natural cost-cut instinct is the closed lever. |
| `stub_engram.py` | A fake `engram` on PATH that fakes "the note is buried at scale": returns the buried note only for a lever-keyed query, and logs every query (phrases + whether it returned the note). |
| `../lever_recheck_scorer.py` | The OUTCOME judge. Default-AMNESIA, majority over 3, per-run votes recorded; judges MEANING vs `closed_levers.json`, not the note's words; a deterministic guard catches "reconciliation by vocabulary." Reuses `synthesis_judge._parse_judge_json`. Stub mode = zero-cost CI. |
| `../recheck.py` | Runs the real skill with the stub on PATH (`live_recall`, via `harness.claude`), then `recheck_result` extracts the recommendation + mechanism signals and scores. Pure core is offline-tested. |
| `../test_lever_recheck_scorer.py`, `../test_recheck.py` | 15 deterministic unit tests (`python3 -m pytest test_lever_recheck_scorer.py test_recheck.py`). |

**`fixture1/` structure:** `vault_with_closed/` (note 8 records the lever tried + rolled back) ·
`vault_open/` (control, note 8 absent) · `context.md` (the −14% numbers, no verdict line) · `task.txt` ·
`closed_levers.json` (ground truth: `canonical_action` / `closure_reason` / `measured_outcome`).

## Findings from building it (the load-bearing result)

Grounded in two live runs of the real `/recall` skill (fresh opus agents) + a retrieval sweep, all
recorded in `RESULTS.md`:

1. **The miss is narrower than #654 assumed: it needs the lever conceived *strictly after* the single
   recall.** When the lever is conceivable at recall time, the recall skill's 10-angle phrasing
   (candidate-solution, prior-work, failure-mode angles) **proactively queries the lever AND its
   history** — Run 2's stub log caught it issuing *"use a cheaper retrieval model"* and *"prior
   experiment cheap retrieval model rolled back"* itself — so the disproof surfaces and the agent
   reconciles. The gap exists only when no angle queries the lever (the diagnostic-task case) and there
   is no re-recall — exactly what #655 closes.
2. **When the disproving note surfaces, opus reconciles correctly** (both runs: it excluded the lever
   and cited the prior attempt). So the miss is *not* a synthesis failure with a salient note.
3. **A single-recall toy fixture therefore cannot reproduce the behavioral miss**, and a small vault
   cannot bury the note by scale (it ranks #1 for every framing tried). The real burial came from
   thousands of chunks + phrasing-distance — `behavioral-traps-need-context`, surfaced rather than
   contorted (note 70).

**Consequence for C7:** the tractable, deterministic metric is **MECHANISM** — RED for the current
skill *by construction* (recall Step 3 walks the plan, never the recommendation — verified by reading
the skill), operationalized by the stub log. The behavioral **OUTCOME** RED is not toy-reproducible.

## Status

**`fixture1` is a regression guard, not a complete eval.** Opus reconciles correctly on it today, so it
guards against a future recall change that *stops* reconciling when the note IS present. **A GREEN on
`fixture1` alone is NOT reportable** (per `eval-end-metric-conflates-retrieval-vs-synthesis` + #654's
≥4–5-fixture bar).

- **Done:** the instrument (stub + scorer + recheck + 15 tests) and `fixture1` as the control.
- **Deferred (PREREQUISITE for any GREEN, not "additional"):** a **two-phase** fixture (recall on a
  diagnostic framing where the buried note stays buried; lever emerges in phase 2) to demonstrate the
  deterministic MECHANISM RED live and the #655 GREEN — then the ≥4 further distinct closed-lever
  fixtures #654 requires.
