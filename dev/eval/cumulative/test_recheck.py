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


def test_extract_recommendation_takes_the_last_match():
    # the format directive says END with the line; recall narration or earlier drafts may contain
    # stray RECOMMENDATION: matches — the LAST one is the reply's actual recommendation.
    txt = ("The note said RECOMMENDATION: something older, quoted mid-analysis.\n"
           "...analysis...\n"
           "RECOMMENDATION: batch the generation calls.\n")
    assert recheck.extract_recommendation(txt) == "batch the generation calls."


def test_rec_line_found_flags_explicit_line_vs_fallback():
    assert recheck.rec_line_found("RECOMMENDATION: do the thing.") is True
    assert recheck.rec_line_found("no explicit line here, just prose") is False
    assert recheck.rec_line_found("") is False
    assert recheck.rec_line_found(None) is False


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
    assert out["rec_line_found"] is True         # explicit line, no whole-text fallback


def test_recheck_result_records_rec_line_found_false_on_fallback(tmp_path):
    log = _write_log(tmp_path, [{"phrases": ["x"], "lever_keyed": False, "returned_buried": False}])
    out = recheck.recheck_result(FIXTURE1, "just prose, no explicit line", log, stub=True)
    assert out["rec_line_found"] is False


def test_agent_text_normalizes_harness_result_shapes():
    assert recheck.agent_text({"result": "hi"}) == "hi"
    assert recheck.agent_text({"text": "yo"}) == "yo"
    assert recheck.agent_text("plain") == "plain"
    assert recheck.agent_text(None) == ""


def _fake_harness(monkeypatch, seen):
    """Install a fake harness module in sys.modules (live_recall imports it lazily), recording the
    claude() call's kwargs into `seen` — keeps live_recall tests pure/offline."""
    import sys
    import types

    def fake_claude(cfg, model, vault, cwd, prompt, resume_sid=None, chunks=None, extra_env=None):
        seen.update({"extra_env": extra_env, "vault": vault, "prompt": prompt, "cwd": cwd,
                     "resume_sid": resume_sid})
        return {"result": "RECOMMENDATION: batch the generation calls.",
                "total_cost_usd": 0.42, "session_id": "sid-1"}

    fake = types.ModuleType("harness")
    fake.claude = fake_claude
    monkeypatch.setitem(sys.modules, "harness", fake)


def test_live_recall_returns_full_harness_result_dict(tmp_path, monkeypatch):
    """live_recall must return the FULL harness result (cost + session_id intact), not just the
    agent text — the runner's validity gate and cost tally read those fields."""
    seen = {}
    _fake_harness(monkeypatch, seen)
    out = recheck.live_recall(
        str(tmp_path), cfg="/dev/null", model="opus", task="diagnose the cost regression",
        bin_dir=str(tmp_path / "bin"), log_path=str(tmp_path / "stub_log.jsonl"),
        buried_basename="8.note", lever_terms="a,b")
    assert out["total_cost_usd"] == 0.42
    assert out["session_id"] == "sid-1"
    assert recheck.agent_text(out) == "RECOMMENDATION: batch the generation calls."
    # the stub shim + env wiring still happen inside live_recall
    assert os.path.isfile(str(tmp_path / "bin" / "engram"))
    assert seen["extra_env"]["STUB_ENGRAM_BURIED"] == "8.note"
    assert seen["extra_env"]["STUB_ENGRAM_LEVER_TERMS"] == "a,b"
    assert seen["vault"].endswith("vault_with_closed")
    assert seen["cwd"] == str(tmp_path)  # default cwd: fixture_dir (back-compat)


def test_live_recall_accepts_explicit_cwd_override(tmp_path, monkeypatch):
    """Two-phase layout (amendment 2): the runner passes an isolated trial cwd so the fixture's
    ground truth is not reachable from the agent's working directory; the vault stays keyed off
    fixture_dir via ENGRAM_VAULT_PATH regardless."""
    seen = {}
    _fake_harness(monkeypatch, seen)
    trial_cwd = str(tmp_path / "trial-cwd")
    recheck.live_recall(
        str(tmp_path), cfg="/dev/null", model="opus", task="diagnose",
        bin_dir=str(tmp_path / "bin"), log_path=str(tmp_path / "stub_log.jsonl"),
        buried_basename="8.note", lever_terms="a,b", cwd=trial_cwd)
    assert seen["cwd"] == trial_cwd
    assert seen["vault"] == os.path.join(str(tmp_path), "vault_with_closed")
    assert seen["resume_sid"] is None  # default: a fresh session


def test_live_recall_threads_resume_sid_and_preserves_log_when_told(tmp_path, monkeypatch):
    """Two-turn structure (amendment 3): turn 2 resumes turn 1's session and must NOT truncate the
    shared stub log — one log spans both turns so the mechanism metric is session-wide."""
    seen = {}
    _fake_harness(monkeypatch, seen)
    log = tmp_path / "stub_log.jsonl"
    log.write_text('{"phrases": ["turn one query"], "lever_keyed": false, "returned_buried": false}\n')
    recheck.live_recall(
        str(tmp_path), cfg="/dev/null", model="opus", task="here is the data",
        bin_dir=str(tmp_path / "bin"), log_path=str(log),
        buried_basename="8.note", lever_terms="a,b",
        resume_sid="sid-1", truncate_log=False)
    assert seen["resume_sid"] == "sid-1"
    assert "turn one query" in log.read_text()  # turn-1 rows preserved


def test_live_recall_truncates_log_by_default(tmp_path, monkeypatch):
    seen = {}
    _fake_harness(monkeypatch, seen)
    log = tmp_path / "stub_log.jsonl"
    log.write_text('{"phrases": ["stale row from a previous run"], "lever_keyed": false}\n')
    recheck.live_recall(
        str(tmp_path), cfg="/dev/null", model="opus", task="diagnose",
        bin_dir=str(tmp_path / "bin"), log_path=str(log),
        buried_basename="8.note", lever_terms="a,b")
    assert log.read_text() == ""  # fresh session starts with a clean log
