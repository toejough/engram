"""Unit tests for recheck.py's pure core (offline — no LLM, no claude -p)."""
import json
import os

import pytest

import recheck

HERE = os.path.dirname(os.path.abspath(__file__))
FIXTURE1 = os.path.join(HERE, "lever_recheck", "fixture1")


def _write_log(tmp_path, rows):
    p = tmp_path / "stub_log.jsonl"
    p.write_text("".join(json.dumps(r) + "\n" for r in rows))
    return str(p)


def test_read_stub_log_buried_when_no_lever_query(tmp_path):
    log = _write_log(tmp_path, [{"phrases": ["cut cost"], "lever_keyed": False, "returned_buried": False}])
    m = recheck.read_stub_log(log)
    assert m["note_surfaced"] is False
    assert m["lever_query_issued"] is False
    assert m["n_queries"] == 1


def test_read_stub_log_surfaced_when_lever_query(tmp_path):
    log = _write_log(tmp_path, [
        {"phrases": ["cut cost"], "lever_keyed": False, "returned_buried": False},
        {"phrases": ["cheaper retrieval model"], "lever_keyed": True, "returned_buried": True},
    ])
    m = recheck.read_stub_log(log)
    assert m["note_surfaced"] is True
    assert m["lever_query_issued"] is True


def test_read_stub_log_fails_loud_when_missing(tmp_path):
    with pytest.raises(FileNotFoundError):
        recheck.read_stub_log(str(tmp_path / "nope.jsonl"))


def test_extract_recommendation_pulls_the_line():
    txt = "blah\nRECOMMENDATION: run retrieval on a cheaper model.\nWHY: ...\n"
    assert recheck.extract_recommendation(txt) == "run retrieval on a cheaper model."


def test_recheck_result_amnesia_when_lever_proposed_and_note_buried(tmp_path):
    # the RED case: skill issued no lever-keyed query (note buried), recommendation re-proposes the lever
    log = _write_log(tmp_path, [{"phrases": ["where does Orchestra cost go"],
                                 "lever_keyed": False, "returned_buried": False}])
    agent_text = "RECOMMENDATION: run the retrieval step on a cheaper, smaller model to cut per-doc cost."
    out = recheck.recheck_result(FIXTURE1, agent_text, log, stub=True)
    assert out["cell_verdict"] == "AMNESIA"
    assert out["note_surfaced"] is False        # retrieval miss: disproof never surfaced
    assert out["lever_query_issued"] is False


def test_recheck_result_reconciled_when_note_surfaced_and_lever_avoided(tmp_path):
    log = _write_log(tmp_path, [{"phrases": ["cheaper retrieval model rolled back"],
                                 "lever_keyed": True, "returned_buried": True}])
    agent_text = "RECOMMENDATION: trim the generation step's prompt/context length."
    out = recheck.recheck_result(FIXTURE1, agent_text, log, stub=True)
    assert out["cell_verdict"] == "RECONCILED"
    assert out["note_surfaced"] is True          # surfaced and respected
