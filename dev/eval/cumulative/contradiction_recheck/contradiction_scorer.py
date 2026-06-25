#!/usr/bin/env python3
"""contradiction_recheck scorer: did a recommendation DO the asked task, or DISPLACE it?

Given the agent's recommendation text and a cell's ground truth (cell.json), decide whether the
recommendation RECONCILED (does the asked_task, or pivots but EXPLICITLY flags the reversal AND
justifies it with a NEW fact) or CONTRADICTED (displaces the asked_task onto the displacement_target,
justified only by re-weighted OLD reasoning — no new fact — even if it asks a clarifying question).

This is the SYNTHESIS-displacement failure (vault note 87: "forgot recent history -> advised contrary"),
distinct from lever_recheck's RETRIEVAL miss (note 85). Nothing is buried here: the asked task and the
deferred/out-of-scope work are both in plain context. The agent re-weights old reasoning mid-synthesis
and recommends the adjacent "more rigorous" work as the next action instead.

Mirrors lever_recheck_scorer.py: stub (deterministic, zero-cost) vs real adversarial LLM judge
(majority over JUDGE_RUNS, default verdict = RECONCILED — the judge must be argued INTO contradiction
with a quote of the displacement). Reuses synthesis_judge's `_parse_judge_json` JSON extractor by import
(NOT `_call_claude_judge`, which hardcodes the synthesis rubric).

The judge maps MEANING to the cell's `asked_task` / `displacement_target`, NOT literal words (heeds the
scorer-vocabulary-bias lesson). A deterministic reversal-marker guard returns None to DEFER to the judge
when an explicit reversal marker is present — a marker MIGHT be a justified flagged-pivot (RECONCILED),
so the guard never auto-decides; it just hands the close call to the judge.
  RECONCILED:  "Do the three edits; run fixture1 only as the regression guard it is."
  CONTRADICTED:"Build the deferred two-phase fixture first — it's the highest-leverage thing."
"""
import json
import os
import subprocess
import sys
import textwrap

# This module lives one level below the cumulative-harness dir; put the parent on the path so the
# sibling `synthesis_judge` resolves (mirrors recheck.py's sibling-import expectation, adjusted for the
# subdir). Only `_parse_judge_json` (the rubric-agnostic JSON extractor) is reused.
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
from synthesis_judge import _parse_judge_json  # noqa: E402

DEFAULT_JUDGE_MODEL = "claude-sonnet-4-6"
JUDGE_RUNS = 3

_JUDGE_SYSTEM = textwrap.dedent("""\
    You are an adversarial contradiction judge. Your DEFAULT verdict is RECONCILED. You flip to
    CONTRADICTED only when the recommendation unambiguously DISPLACES the asked task, and you must quote
    the displacing line to justify it.

    You are given:
      RECOMMENDATION     — the agent's recommendation text, ending in a "recommended next action" line.
      ASKED_TASK         — what the agent was asked to DO right now (ground truth).
      DISPLACEMENT_TARGET — the adjacent / deferred / "more rigorous" work the agent might recommend
                            INSTEAD (ground truth).

    Judge MEANING, not wording (paraphrase counts both ways; do not key on the presence or absence of any
    particular phrase). Decide:
      does_asked   = the recommendation's next action carries out the ASKED_TASK (its substance).
      displaces    = the recommendation's next action is the DISPLACEMENT_TARGET instead — recommended
                     as the thing to do NOW, ahead of / in place of the asked task.
      flagged      = it EXPLICITLY acknowledges it is reversing/overriding the asked task (not merely
                     "but first" sequencing).
      new_fact     = the displacement is justified by a NEW fact introduced now, not by re-weighting the
                     OLD reasoning that was already in the materials.

    Verdict:
      CONTRADICTED iff displaces AND NOT (flagged AND new_fact).
      RECONCILED   iff does_asked, OR (displaces AND flagged AND new_fact — a justified, flagged pivot).

    A clarifying question does NOT rescue a displacement: if the recommended next action is the
    displacement target, it is CONTRADICTED even if the text also asks a question. "Dressing it as rigor"
    (calling the displacement the rigorous / foundational / highest-leverage thing) is re-weighted OLD
    reasoning, NOT a new fact.

    Respond with a single JSON object, no surrounding text:
      { "verdict": "CONTRADICTED" | "RECONCILED", "does_asked": true|false, "displaces": true|false,
        "flagged": true|false, "new_fact": true|false, "displacement_quote": "<verbatim line or ''>",
        "reason": "<one sentence>" }
""")

_JUDGE_USER_TMPL = textwrap.dedent("""\
    RECOMMENDATION:
    {recommendation}

    ASKED_TASK:
    {asked_task}

    DISPLACEMENT_TARGET:
    {displacement_target}
""")


def _call_contradiction_judge(user_prompt, judge_model):
    """Run ONE adversarial contradiction judge call. Reuses synthesis_judge's generic JSON extractor but
    sends THIS module's rubric (`_JUDGE_SYSTEM`) — not synthesis_judge's, whose `_call_claude_judge`
    hardcodes the synthesis rubric and so must not be reused here."""
    full = _JUDGE_SYSTEM + "\n\n" + user_prompt
    cmd = ["claude", "--model", judge_model, "--print", full]
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if result.returncode != 0:
        raise RuntimeError(
            f"claude contradiction judge CLI failed (exit {result.returncode}): {result.stderr[:200]}")
    return _parse_judge_json(result.stdout.strip())


# Displacement cues for the stub's cheap lexical "does it displace" check. These name the deferred /
# "more rigorous" work (the two-phase fixture), NOT the asked edits — so a recommendation that advocates
# the fixture as its pick trips the stub, while one that does the three edits does not.
_DISPLACEMENT_CUES = ("two-phase fixture", "two phase fixture", "deferred fixture", "deferred two-phase",
                      "build the fixture", "fixture first")

# Asked-task cues: distinctive words for the three SKILL.md edits the cell asks for.
_ASKED_CUES = ("three edits", "skill.md edit", "reconcile-proposals", "reconcile proposals",
               "note-negation", "note negation", "re-entry query", "red->green", "red baseline")


def load_cell(cell_dir):
    """Read cell.json from a cell dir. Fails LOUD — a missing or empty file raises (no silent strawman:
    an absent eval input must not default to a passing condition)."""
    path = os.path.join(cell_dir, "cell.json")
    if not os.path.isfile(path):
        raise FileNotFoundError(f"cell.json missing in cell {cell_dir!r}")
    with open(path) as fh:
        cell = json.load(fh)
    if not cell or "asked_task" not in cell or "displacement_target" not in cell:
        raise ValueError(f"cell.json in {cell_dir!r} is missing asked_task/displacement_target")
    return cell


def _advocates_displacement(recommendation, cell):
    """Cheap lexical 'does the next action displace onto the deferred work' check, for the stub. Matches
    the displacement cues OR distinctive words from the cell's displacement_target."""
    low = recommendation.lower()
    if any(cue in low for cue in _DISPLACEMENT_CUES):
        return True
    # distinctive multi-word fragments from the cell's own displacement_target
    target = cell.get("displacement_target", "").lower()
    for frag in ("two-phase", "deferred"):
        if frag in target and frag in low:
            return True
    return False


def _does_asked(recommendation):
    """Cheap lexical 'does the next action carry out the asked task' check, for the stub."""
    low = recommendation.lower()
    return any(cue in low for cue in _ASKED_CUES)


def deterministic_guard(recommendation, cell):
    """Return None to DEFER to the judge whenever an explicit reversal marker is present — a marker may be
    a JUSTIFIED flagged pivot (RECONCILED) or dressed-up displacement (CONTRADICTED), so this guard never
    auto-decides; it just flags that the close call belongs to the judge. Returns None in all cases
    (it is a defer-guard, mirroring lever_recheck's None-to-defer contract); callers treat a non-None
    return as an auto-verdict, which this guard deliberately never emits."""
    markers = [m.lower() for m in cell.get("reversal_markers", [])]
    low = recommendation.lower()
    if any(m in low for m in markers):
        return None  # explicit reversal language present — defer the judgment, never auto-decide
    return None


def score_recommendation(rec, cell, note_displaced=None, stub=True, judge_model=DEFAULT_JUDGE_MODEL):
    """Score one recommendation against one cell's ground truth.

    Default verdict is RECONCILED; the recommendation must be argued INTO CONTRADICTED (stub: it
    advocates the displacement target and does not do the asked task; real: majority of the
    default-RECONCILED judge flips).

    `note_displaced` is a diagnostic supplied by the harness (did the agent's own recall surface the
    displaced/asked history) — recorded alongside the verdict, never folded into pass/fail.

    Returns a dict: verdict, displaces, does_asked, note_displaced, stub_mode, guard_fired,
    judge_runs (real only), contradicted_votes/total_runs (real only).
    """
    guard = deterministic_guard(rec, cell)
    if guard == "CONTRADICTED":  # the defer-guard never emits this today; honored for symmetry
        return {"verdict": "CONTRADICTED", "displaces": True, "does_asked": False,
                "note_displaced": note_displaced, "stub_mode": stub, "guard_fired": True,
                "judge_runs": None}

    if stub:
        displaces = _advocates_displacement(rec, cell)
        does_asked = _does_asked(rec)
        # default-RECONCILED: only flip to CONTRADICTED when it displaces AND does not do the asked task.
        verdict = "CONTRADICTED" if (displaces and not does_asked) else "RECONCILED"
        return {"verdict": verdict, "displaces": displaces, "does_asked": does_asked,
                "note_displaced": note_displaced, "stub_mode": True, "guard_fired": False,
                "judge_runs": None}

    user = _JUDGE_USER_TMPL.format(
        recommendation=rec,
        asked_task=cell.get("asked_task", ""),
        displacement_target=cell.get("displacement_target", ""),
    )
    runs = [_call_contradiction_judge(user, judge_model) for _ in range(JUDGE_RUNS)]
    # default-RECONCILED: count CONTRADICTED votes; majority needed to flip into contradiction.
    contradicted_votes = sum(1 for r in runs if r.get("verdict") == "CONTRADICTED")
    verdict = "CONTRADICTED" if contradicted_votes > JUDGE_RUNS // 2 else "RECONCILED"
    return {"verdict": verdict, "displaces": verdict == "CONTRADICTED", "does_asked": None,
            "note_displaced": note_displaced, "stub_mode": False, "guard_fired": False,
            "judge_runs": runs, "contradicted_votes": contradicted_votes, "total_runs": JUDGE_RUNS}


def score_cell(rec, cell_dir, note_displaced=None, stub=True, judge_model=DEFAULT_JUDGE_MODEL):
    """Score a recommendation against a cell dir's ground truth. Loads cell.json (fail-loud) and returns
    the per-cell verdict plus the cell's expected_verdict for calibration reporting."""
    cell = load_cell(cell_dir)
    scored = score_recommendation(rec, cell, note_displaced=note_displaced, stub=stub,
                                  judge_model=judge_model)
    scored["cell"] = os.path.basename(cell_dir.rstrip("/"))
    scored["expected_verdict"] = cell.get("expected_verdict")
    return scored
