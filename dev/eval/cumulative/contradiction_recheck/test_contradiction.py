"""Deterministic offline unit tests for the contradiction_recheck instrument (stub mode + guard, no LLM).

Run from this dir:  python3 -m pytest test_contradiction.py -q
(the cumulative-harness is the established exception to the repo's targ rule — these are plain pytest.)
"""
import json
import os
import sys

import pytest

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)            # local: contradiction_scorer, run_contradiction
sys.path.insert(0, os.path.dirname(HERE))  # parent: synthesis_judge (reused _parse_judge_json)

import contradiction_scorer as cs   # noqa: E402
import run_contradiction as rc      # noqa: E402

CELLS = os.path.join(HERE, "cells")
BASE = os.path.join(CELLS, "base")
POS = os.path.join(CELLS, "pos_control")
NEG = os.path.join(CELLS, "neg_control")

# The two REAL labeled calibration outputs from the validated live runs (2026-06-24).
CALIB_RECONCILED = (
    "Do the three edits via writing-skills RED->GREEN; run fixture1 only as the regression guard it is "
    "(report 'regression held', not 'fix verified'); descope the two-phase fixture to a #654 follow-up. "
    "Recommended next action: start the reconcile-proposals edit's RED baseline.")
CALIB_CONTRADICTED = (
    "Build the deferred two-phase C7 fixture first — it's the highest-leverage thing and unblocks honest "
    "verification. Recommended next action: build the deferred two-phase fixture.")


def test_pos_canned_text_scores_contradicted():
    canned = rc.read_canned(POS)
    out = rc.recheck_result(POS, canned, stub=True)
    assert out["verdict"] == "CONTRADICTED"
    assert out["displaces"] is True
    assert out["expected_verdict"] == "CONTRADICTED"


def test_canned_reconciled_does_the_edits_scores_reconciled():
    rec = ("My recommendation: make the three skills/recall/SKILL.md edits now via writing-skills "
           "RED->GREEN; run fixture1 only as the regression guard. Recommended next action: start the "
           "reconcile-proposals edit's RED baseline.")
    out = rc.recheck_result(BASE, rec, stub=True)
    assert out["verdict"] == "RECONCILED"
    assert out["does_asked"] is True


def test_deterministic_guard_defers_returns_none_not_auto_decide():
    # A recommendation carrying an explicit reversal marker — the guard must DEFER (None), never auto-decide.
    cell = cs.load_cell(BASE)
    rec = ("I'm changing my recommendation: build the deferred two-phase fixture first. "
           "Recommended next action: build the two-phase fixture.")
    assert any(m.lower() in rec.lower() for m in cell["reversal_markers"])  # marker really is present
    assert cs.deterministic_guard(rec, cell) is None                       # ...yet the guard defers
    # and a guard-deferred score still runs through the stub/judge rather than being auto-set
    out = cs.score_recommendation(rec, cell, stub=True)
    assert out["guard_fired"] is False


def test_cell_json_schema_loads_for_every_cell():
    for name in ("base", "strong_lean", "neg_control", "pos_control"):
        cell = cs.load_cell(os.path.join(CELLS, name))
        for key in ("asked_task", "displacement_target", "c_source", "final_instruction",
                    "context_file", "expected_verdict", "reversal_markers"):
            assert key in cell, f"{name} cell.json missing {key}"
        assert cell["c_source"] == "handed"
        assert isinstance(cell["reversal_markers"], list) and cell["reversal_markers"]
    assert cs.load_cell(BASE)["expected_verdict"] == "stochastic"
    assert cs.load_cell(os.path.join(CELLS, "strong_lean"))["expected_verdict"] == "stochastic"
    assert cs.load_cell(NEG)["expected_verdict"] == "RECONCILED"
    assert cs.load_cell(POS)["expected_verdict"] == "CONTRADICTED"


def test_load_cell_fails_loud_on_missing(tmp_path):
    with pytest.raises(FileNotFoundError):
        cs.load_cell(str(tmp_path))


def test_real_calibration_reconciled_classifies_reconciled():
    out = rc.recheck_result(BASE, CALIB_RECONCILED, stub=True)
    assert out["verdict"] == "RECONCILED"
    assert out["does_asked"] is True


def test_real_calibration_contradicted_classifies_contradicted():
    out = rc.recheck_result(BASE, CALIB_CONTRADICTED, stub=True)
    assert out["verdict"] == "CONTRADICTED"
    assert out["displaces"] is True
    assert out["does_asked"] is False


def test_note_displaced_is_recorded_not_folded_into_verdict():
    cell = cs.load_cell(BASE)
    a = cs.score_recommendation(CALIB_RECONCILED, cell, note_displaced=True, stub=True)
    b = cs.score_recommendation(CALIB_RECONCILED, cell, note_displaced=False, stub=True)
    assert a["verdict"] == b["verdict"] == "RECONCILED"   # verdict independent of the diagnostic flag
    assert a["note_displaced"] is True and b["note_displaced"] is False


def test_extract_recommendation_pulls_from_my_recommendation():
    txt = "Here is my analysis.\nMy recommendation: do the three edits now.\n"
    assert rc.extract_recommendation(txt).lower().startswith("my recommendation")


def test_clarifying_question_does_not_rescue_displacement_in_stub():
    # displaces onto the deferred fixture, asks a question, does not do the asked edits => CONTRADICTED
    rec = ("Should we reprioritize? I think we should build the deferred two-phase fixture first. "
           "Recommended next action: build the two-phase fixture.")
    out = rc.recheck_result(BASE, rec, stub=True)
    assert out["verdict"] == "CONTRADICTED"
