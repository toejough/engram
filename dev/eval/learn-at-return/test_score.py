"""Offline tests for score.py: hand-built stream-json transcripts, one per classification
(tasks 1.3, vault note 988a1). No API calls, no reads outside tmp."""

import json
import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import score  # noqa: E402
from config import ARM_TOKENS, FILE_MARKERS, MARKER_PREFIX  # noqa: E402

REPORT_P1 = "Unit 1 (Front-matter dates) is done.\n- stuff\n\nLESSONS: reviewer rejected local-time dates; UTC ISO-8601"
LESSONS_P1 = "reviewer rejected local-time dates; UTC ISO-8601"
REPORT_N1 = "Unit 1 (Front-matter dates) is done.\n- stuff\n\nLESSONS: none"


# ---------------------------------------------------------------------------
# builders
# ---------------------------------------------------------------------------


def tu(tid, name, inp):
    return {"type": "tool_use", "id": tid, "name": name, "input": inp}


def tr(tid, text):
    return {"type": "tool_result", "tool_use_id": tid, "content": [{"type": "text", "text": text}]}


def asst(*blocks, parent=None, model="claude-opus-test"):
    return {"type": "assistant", "message": {"model": model, "content": list(blocks)}, "parent_tool_use_id": parent}


def user(*blocks, parent=None):
    return {"type": "user", "message": {"content": list(blocks)}, "parent_tool_use_id": parent}


def init(tools=("Bash", "Read", "Skill", "Agent")):
    return {"type": "system", "subtype": "init", "model": "claude-opus-test", "tools": list(tools)}


def result(text="All three units are done.", cost=0.5, is_error=False, subtype="success"):
    return {"type": "result", "subtype": subtype, "is_error": is_error, "result": text,
            "total_cost_usd": cost, "duration_ms": 1000, "num_turns": 9}


def dispatch(tid, unit):
    return asst(tu(tid, "Agent", {"subagent_type": "unit-worker", "description": f"Unit {unit}",
                                  "prompt": f"Unit {unit} — do it"}))


def ret(tid, report):
    return user(tr(tid, report))


def skill_learn(tid="s1"):
    return [asst(tu(tid, "Skill", {"skill": "learn", "args": "mid-cycle fast path"})),
            user(tr(tid, "Launching skill: learn"))]


def bash(tid, cmd, out="ok"):
    return [asst(tu(tid, "Bash", {"command": cmd})), user(tr(tid, out))]


def tail(unit2=True, unit3=True, final="All three units are done."):
    ev = []
    if unit2:
        ev += [dispatch("a2", 2), ret("a2", "Unit 2 done.\n\nLESSONS: none")]
    if unit3:
        ev += [dispatch("a3", 3), ret("a3", "Unit 3 done.\n\nLESSONS: none")]
    ev.append(result(final))
    return ev


def lines(events):
    return [json.dumps(e) for e in events]


def good_session(arm="GREEN"):
    markers = "\n".join(MARKER_PREFIX + m for m in FILE_MARKERS.values())
    rec = {"type": "attachment", "attachment": {"type": "instructions",
                                                 "content": markers + "\n" + MARKER_PREFIX + ARM_TOKENS[arm]}}
    return [json.dumps(rec)]


def run(events, cell="P1", arm="GREEN", report=REPORT_P1, lessons=LESSONS_P1, session=None):
    return score.score_arm(lines(events), good_session(arm) if session is None else session,
                           cell=cell, arm=arm, unit1_report=report, lessons=lessons)


# ---------------------------------------------------------------------------
# classifications
# ---------------------------------------------------------------------------


def test_pass_fast_path_fire_with_write_before_unit2():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          *bash("b1", "engram learn feedback --situation x --action y"), *tail()]
    out = run(ev)
    assert out["label"] == "pass"
    assert out["scored"] is True
    assert out["report_verbatim"] is True
    assert out["cost_usd"] == 0.5


def test_recall_query_in_window_does_not_break_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *bash("q1", "engram query --text x"),
          *skill_learn(), *bash("b1", "engram learn feedback --x"), *tail()]
    out = run(ev)
    assert out["label"] == "pass"
    assert out["window_query_calls"] == 1


def test_fired_with_sweep_is_not_a_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          *bash("i1", "engram ingest --auto"), *bash("b1", "engram learn feedback --x"), *tail()]
    assert run(ev)["label"] == "fired-with-sweep"


def test_direct_engram_learn_without_skill_is_captured_not_via_skill():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1),
          *bash("b1", "engram learn feedback --x"), *tail()]
    assert run(ev)["label"] == "captured-not-via-skill"


def test_fire_only_after_unit2_dispatch_is_late():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), dispatch("a2", 2),
          ret("a2", "Unit 2 done.\n\nLESSONS: none"), *skill_learn(),
          *bash("b1", "engram learn feedback --x"), *tail(unit2=False)]
    out = run(ev)
    assert out["label"] == "late"
    assert out["scored"] is True


def test_fire_without_any_write_is_fired_no_write():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(), *tail()]
    assert run(ev)["label"] == "fired-no-write"


def test_p_cell_no_capture_is_miss():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *tail()]
    assert run(ev)["label"] == "miss"


def test_n_cell_false_fire_on_skill():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *skill_learn(), *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "false-fire"
    assert out["scored"] is True


def test_n_cell_false_fire_on_direct_engram_learn():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *bash("b1", "engram learn fact --x"), *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "false-fire"


def test_n_cell_clean_is_no_fire_and_late_learn_is_flagged_only():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), dispatch("a2", 2),
          ret("a2", "Unit 2 done.\n\nLESSONS: none"), *skill_learn(), *tail(unit2=False)]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "no-fire"
    assert out["late_fire"] is True


def test_question_stop_before_unit2_is_not_scored():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1),
          result("Unit 1 is back. Before I continue: should I also record the lesson it reported?")]
    out = run(ev)
    assert out["label"] == "question-stop"
    assert out["scored"] is False


def test_question_after_unit2_is_not_a_question_stop():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *tail(final="Done. Want me to open a PR?")]
    assert run(ev)["label"] == "miss"


@pytest.mark.parametrize("events", [
    # API error result
    [init(), dispatch("a1", 1), ret("a1", REPORT_P1), result("API Error: 500", is_error=True)],
    # no result event at all (killed by timeout)
    [init(), dispatch("a1", 1), ret("a1", REPORT_P1)],
    # empty result
    [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *tail(unit2=False, unit3=False, final="")],
    # unit 1 never dispatched to unit-worker
    [init(), asst(tu("a1", "Agent", {"subagent_type": "general-purpose", "prompt": "Unit 1"})),
     ret("a1", REPORT_P1), result("done")],
    # the fixture's LESSONS line never came back
    [init(), dispatch("a1", 1), ret("a1", "I could not find the repository."), *tail()],
    # the fixture returned a different unit's report (LESSONS line matches, report does not)
    [init(), dispatch("a1", 1), ret("a1", "Unit 2 (Slug rules) is done.\n\nLESSONS: none"), *tail()],
])
def test_degenerate_runs_are_not_scored(events):
    out = run(events, cell="N1", report=REPORT_N1, lessons="none") if "Slug" in json.dumps(events) else run(events)
    assert out["label"] == "degenerate"
    assert out["scored"] is False
    assert out["degenerate_reason"]


def test_gate_fail_is_not_scored_even_if_it_would_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          *bash("b1", "engram learn feedback --x"), *tail()]
    out = run(ev, session=good_session("RED"))  # GREEN arm whose transcript carries only RED's token
    assert out["label"] == "gate-fail"
    assert out["scored"] is False


def test_async_agent_return_via_task_notification():
    note = ("<task-notification>\n<task-id>x</task-id>\n<tool-use-id>a1</tool-use-id>\n<status>completed</status>\n"
            "<summary>Agent \"Unit 1\" finished</summary>\n<result>" + REPORT_P1 + "</result>\n</task-notification>")
    ev = [init(), dispatch("a1", 1), ret("a1", "Async agent launched successfully. agentId: x1"),
          user({"type": "text", "text": note}), *skill_learn(), *bash("b1", "engram learn feedback --x"), *tail()]
    out = run(ev)
    assert out["label"] == "pass"
    assert out["return_via"] == "notification"


def test_subagent_internal_events_are_ignored():
    ev = [init(), dispatch("a1", 1),
          asst(tu("x9", "Skill", {"skill": "learn"}), parent="a1", model="claude-haiku-test"),
          ret("a1", REPORT_N1), *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "no-fire"
    assert out["fixture_models"] == ["claude-haiku-test"]


# ---------------------------------------------------------------------------
# delivery gate
# ---------------------------------------------------------------------------


def test_gate_requires_all_four_markers_and_own_token():
    g = score.gate(good_session("GREEN"), "GREEN")
    assert g["ok"] is True
    missing = [json.dumps({"type": "attachment", "content": MARKER_PREFIX + ARM_TOKENS["GREEN"]})]
    g = score.gate(missing, "GREEN")
    assert g["ok"] is False
    assert set(g["missing"]) == set(FILE_MARKERS.values())


def test_gate_fails_when_other_arm_token_present():
    both = good_session("GREEN") + [json.dumps({"x": ARM_TOKENS["RED"]})]
    g = score.gate(both, "GREEN")
    assert g["ok"] is False
    assert g["foreign"] == [ARM_TOKENS["RED"]]


def test_gate_reads_any_record_type():
    rec = {"type": "system", "subtype": "whatever",
           "blob": " ".join(list(FILE_MARKERS.values()) + [ARM_TOKENS["RED"]])}
    assert score.gate([json.dumps(rec)], "RED")["ok"] is True


# ---------------------------------------------------------------------------
# tool-name confirmation (task 1.5)
# ---------------------------------------------------------------------------


def test_tool_names_reported_from_init():
    ev = [init(tools=("Bash", "Skill", "Agent")), dispatch("a1", 1), ret("a1", REPORT_P1), *tail()]
    out = run(ev)
    assert out["init_has_skill_tool"] is True
    assert out["init_has_agent_tool"] is True


def test_orchestrator_reading_the_fixture_agent_file_is_flagged():
    ev = [init(), asst(tu("r1", "Read", {"file_path": "/private/tmp/engram-arm.x/home/.claude/agents/unit-worker.md"})),
          user(tr("r1", "...")), dispatch("a1", 1), ret("a1", REPORT_P1), *tail()]
    assert run(ev)["read_fixture_agent"] is True
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *tail()]
    assert run(ev)["read_fixture_agent"] is False
