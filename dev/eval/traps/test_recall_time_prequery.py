"""TDD for #690 Task 1: the pre-query inner-split instrument, purely additive over the
FROZEN recall_time.py segmenter (#684 pinned 4-phase model). Golden values are measured from
the committed dev/eval/traps/testdata/prequery_trial0.jsonl fixture (a copy of the #689
after-measure trial-0 transcript) — see the plan/brief for the exact record timestamps."""
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import recall_time as rt

FIX = os.path.join(os.path.dirname(__file__), "testdata", "prequery_trial0.jsonl")


def _load(path):
    return [__import__("json").loads(l) for l in open(path) if l.strip()]


def test_pre_query_split_golden_trial0():
    records = _load(FIX)
    start_ts = rt.find_span(records)[0]["start_ts"]
    first_q = rt.find_query_calls(records)[0]["tool_use_ts"]
    split, err = rt.compute_pre_query_split(records, start_ts, first_q)
    assert err is None
    assert split["split_gate_ok"] is True
    assert split["ttft_invoke_s"] == 4.1
    assert split["skill_read_step0_s"] == 7.1
    assert split["sweep_s"] == 1.9
    assert split["compose_s"] == 5.3
    assert abs(split["unattributed_s"]) <= 1.0


def _assistant(ts, blocks):
    return {"type": "assistant", "timestamp": ts, "message": {"content": blocks}}


def _tool_use(uid, name, input_):
    return {"type": "tool_use", "id": uid, "name": name, "input": input_}


def _tool_result(uid, text):
    return {"type": "tool_result", "tool_use_id": uid, "content": [{"type": "text", "text": text}]}


def test_stop_when_no_ingest_sweep_before_query():
    """The engram ingest call is removed from the sequence -> STOP, byte-identical reason."""
    records = [
        _assistant("2026-01-01T00:00:00.000Z", [_tool_use("u1", "Skill", {"skill": "recall", "args": "x"})]),
        _assistant("2026-01-01T00:00:10.000Z", [_tool_use("u2", "Bash", {"command": "engram query --phrase x"})]),
        _assistant("2026-01-01T00:00:11.000Z", [_tool_result("u2", "some query output")]),
    ]
    start_ts = "2026-01-01T00:00:00.000Z"
    first_q = "2026-01-01T00:00:10.000Z"
    split, err = rt.compute_pre_query_split(records, start_ts, first_q)
    assert split is None
    assert err == "no engram ingest sweep before query — cannot separate step0 from compose"


def _split(ttft, skill_read, sweep, compose, unattributed):
    return {"ttft_invoke_s": ttft, "skill_read_step0_s": skill_read, "sweep_s": sweep,
            "compose_s": compose, "unattributed_s": unattributed}


def test_split_gate_ok_when_residual_slightly_negative():
    """A -0.1s residual from independent per-field rounding of the four measured spans is
    within the intended +/-1.0s tolerance and must NOT fail the gate (#690 Task 1 bug: the
    old gate required unattributed_s >= 0, spuriously failing valid trials)."""
    split = _split(4.0, 3.0, 2.0, 1.0, -0.1)
    gate_ok, _detail = rt._pre_query_split_gate(split)
    assert gate_ok is True


def test_split_gate_fails_on_negative_measured_span():
    """A negative MEASURED span (e.g. markers out of order) is a genuine failure, not a
    rounding artifact — the gate must still catch it."""
    split = _split(-0.2, 3.0, 2.0, 1.0, 0.0)
    gate_ok, _detail = rt._pre_query_split_gate(split)
    assert gate_ok is False


def test_split_gate_fails_on_residual_beyond_tolerance():
    """A residual outside +/-1.0s (either direction) is a genuine failure, not bounded
    rounding noise."""
    gate_ok_low, _ = rt._pre_query_split_gate(_split(4.0, 3.0, 2.0, 1.0, -1.2))
    gate_ok_high, _ = rt._pre_query_split_gate(_split(4.0, 3.0, 2.0, 1.0, 1.2))
    assert gate_ok_low is False
    assert gate_ok_high is False


def test_compute_phases_unaffected_by_compute_pre_query_split():
    """Mutation tripwire: compute_phases must return identical values whether or not
    compute_pre_query_split was called first on the same records — guards against a
    FUTURE in-place mutation of the shared records/helpers (not a proof of independence;
    the two functions are already independent pure functions)."""
    records = _load(FIX)
    start_ts = rt.find_span(records)[0]["start_ts"]
    end_ts = rt.find_span(records)[0]["end_ts"]

    phases_before, err_before = rt.compute_phases(records, start_ts, end_ts)

    first_q = rt.find_query_calls(records)[0]["tool_use_ts"]
    rt.compute_pre_query_split(records, start_ts, first_q)

    phases_after, err_after = rt.compute_phases(records, start_ts, end_ts)
    assert phases_before == phases_after
    assert err_before == err_after
