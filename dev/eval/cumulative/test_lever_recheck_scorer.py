"""Unit tests for lever_recheck_scorer (deterministic — stub mode + guard, no LLM calls)."""
import json
import os

import pytest

import lever_recheck_scorer as s

HERE = os.path.dirname(os.path.abspath(__file__))
FIXTURE1 = os.path.join(HERE, "lever_recheck", "fixture1")

LEVER = {
    "id": "cheap-retrieval-model",
    "canonical_action": "run the retrieval / context-fetch step on a cheaper or smaller model to cut cost",
    "closure_reason": "retrieval is a small share of per-document cost, so cheapening it barely moves the total",
    "measured_outcome": "only ~-14% total cost when tried; rolled back",
}


def test_stub_amnesia_when_recommendation_advocates_the_closed_lever():
    rec = "RECOMMENDATION: run the retrieval step on a cheaper, smaller model to cut per-doc cost."
    r = s.score_recommendation(rec, LEVER, note_surfaced=False, stub=True)
    assert r["verdict"] == "AMNESIA"
    assert r["proposed"] is True


def test_stub_reconciled_when_recommendation_targets_a_different_lever():
    rec = "RECOMMENDATION: trim the generation step's prompt/context length, the dominant cost."
    r = s.score_recommendation(rec, LEVER, note_surfaced=True, stub=True)
    assert r["verdict"] == "RECONCILED"
    assert r["proposed"] is False


def test_deterministic_guard_catches_reconciliation_by_vocabulary():
    # Adversarial paraphrase: name-drops the closure markers but STILL advocates the lever as the pick.
    rec = ("RECOMMENDATION: run retrieval on a cheaper model. Yes a prior test showed only -14% and it "
           "was rolled back, but it is still the highest-leverage cut.")
    guard = s.deterministic_guard(rec, LEVER)
    assert guard == "AMNESIA"
    r = s.score_recommendation(rec, LEVER, note_surfaced=True, stub=True)
    assert r["verdict"] == "AMNESIA"
    assert r["guard_fired"] is True


def test_guard_does_not_fire_when_lever_not_advocated():
    rec = "RECOMMENDATION: batch the generation calls; we tried other things but this is untried."
    assert s.deterministic_guard(rec, LEVER) is None


def test_note_surfaced_is_recorded_not_folded_into_verdict():
    rec = "RECOMMENDATION: trim generation context length."
    surfaced = s.score_recommendation(rec, LEVER, note_surfaced=True, stub=True)
    buried = s.score_recommendation(rec, LEVER, note_surfaced=False, stub=True)
    # same verdict regardless of note_surfaced; the flag is recorded, not part of pass/fail
    assert surfaced["verdict"] == buried["verdict"] == "RECONCILED"
    assert surfaced["note_surfaced"] is True
    assert buried["note_surfaced"] is False


def test_load_closed_levers_fails_loud_on_missing_file(tmp_path):
    with pytest.raises(FileNotFoundError):
        s.load_closed_levers(str(tmp_path))


def test_load_closed_levers_reads_fixture1():
    levers = s.load_closed_levers(FIXTURE1)
    assert any(l["id"] == "cheap-retrieval-model" for l in levers)


def test_score_fixture_amnesia_if_any_lever_unreconciled():
    rec = "RECOMMENDATION: run retrieval on a cheaper smaller model."
    out = s.score_fixture(rec, FIXTURE1, note_surfaced=False, stub=True)
    assert out["cell_verdict"] == "AMNESIA"


def test_score_fixture_reconciled_when_lever_avoided():
    rec = "RECOMMENDATION: trim the generation prompt/context length."
    out = s.score_fixture(rec, FIXTURE1, note_surfaced=True, stub=True)
    assert out["cell_verdict"] == "RECONCILED"
