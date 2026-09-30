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


def tr(tid, text, is_error=False):
    return {"type": "tool_result", "tool_use_id": tid, "is_error": is_error, "content": [{"type": "text", "text": text}]}


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


NOTE_PATH = "/private/tmp/engram-arm.x/vault/1.2026-09-30.quillfeather-dates-utc.md"
NOTE_NAME = os.path.basename(NOTE_PATH)


def bash(tid, cmd, out=None, is_error=False, parent=None):
    if out is None:
        out = NOTE_PATH if "engram learn" in cmd else "ok"
    return [asst(tu(tid, "Bash", {"command": cmd}), parent=parent), user(tr(tid, out, is_error), parent=parent)]


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


def run(events, cell="P1", arm="GREEN", report=REPORT_P1, lessons=LESSONS_P1, session=None, others=None,
        vault_notes=None):
    return score.score_arm(lines(events), good_session(arm) if session is None else session,
                           cell=cell, arm=arm, unit1_report=report, lessons=lessons,
                           other_session_texts=others or [], vault_notes=vault_notes)


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


def test_p_cell_no_capture_is_no_fire():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *tail()]
    assert run(ev)["label"] == "no-fire"


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
    assert run(ev)["label"] == "no-fire"


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
    missing = [json.dumps({"type": "attachment", "attachment": {"content": MARKER_PREFIX + ARM_TOKENS["GREEN"]}})]
    g = score.gate(missing, "GREEN")
    assert g["ok"] is False
    assert set(g["missing"]) == set(FILE_MARKERS.values())


def test_gate_fails_when_other_arm_token_present():
    both = good_session("GREEN") + [json.dumps({"x": ARM_TOKENS["RED"]})]
    g = score.gate(both, "GREEN")
    assert g["ok"] is False
    g = score.gate(good_session("GREEN"), "GREEN", other_texts=[json.dumps({"x": ARM_TOKENS["RED"]})])
    assert g["ok"] is False
    assert g["foreign"] == [ARM_TOKENS["RED"]]


def test_gate_counts_only_main_session_attachment_records():
    blob = " ".join(list(FILE_MARKERS.values()) + [ARM_TOKENS["RED"]])
    assert score.gate([json.dumps({"type": "attachment", "attachment": {"content": blob}})], "RED")["ok"] is True
    # the same tokens in a non-attachment record (e.g. a Read of learn.md) do not satisfy the gate
    assert score.gate([json.dumps({"type": "user", "message": {"content": blob}})], "RED")["ok"] is False
    # tokens only in a subagent transcript do not satisfy the gate
    out = run([init(), dispatch("a1", 1), ret("a1", REPORT_P1), *tail()], session=[],
              others=[json.dumps({"type": "attachment", "attachment": {"content": blob}})], arm="RED")
    assert out["label"] == "gate-fail"


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


# ---------------------------------------------------------------------------
# fix round 1: review findings 1-7
# ---------------------------------------------------------------------------


def test_learn_fired_twice_is_still_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn("s1"), *bash("b1", "engram learn feedback --x"),
          *skill_learn("s2"), *bash("b2", "engram learn feedback --y"), *tail()]
    out = run(ev)
    assert out["label"] == "pass"
    assert out["window_fires"] == 2


def test_ingest_after_the_write_stays_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(), *bash("b1", "engram learn feedback --x"),
          *bash("i1", "engram ingest --auto"), *tail()]
    assert run(ev)["label"] == "pass"


def test_failed_write_is_not_a_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          *bash("b1", "engram learn feedback --bogus", out="Error: unknown flag --bogus", is_error=True), *tail()]
    assert run(ev)["label"] == "fired-no-write"


def test_learn_help_is_not_a_write():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          *bash("b1", "engram learn --help", out="Usage: engram learn <kind> [flags]"), *tail()]
    assert run(ev)["label"] == "fired-no-write"


def test_query_phrase_mentioning_learn_is_not_a_false_fire():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1),
          *bash("q1", 'engram query --text "x" --phrase "deciding whether to engram learn a returned lesson"'), *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "no-fire"
    assert out["window_query_calls"] == 1


def test_query_phrase_mentioning_ingest_is_not_a_sweep():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          *bash("q1", 'engram query --phrase "fast path, no engram ingest --auto"'),
          *bash("b1", "engram learn feedback --x"), *tail()]
    assert run(ev)["label"] == "pass"


@pytest.mark.parametrize("cmd,sub", [
    ("engram learn feedback --x", "learn"),
    ("cd /tmp && engram learn fact --y", "learn"),
    ("ENGRAM_X=1 engram learn fact", "learn"),
    ("engram learn feedback --slug s \\\n  --source \"a; b\"", "learn"),
    ("/private/tmp/engram-arm.x/bin/engram ingest --auto", "ingest"),
    ('engram query --phrase "engram learn it" | head', "query"),
    ('echo "engram learn x"', None),
])
def test_engram_subcommands_parse_per_segment(cmd, sub):
    got = score.engram_subcommands(cmd)
    assert (got[0] if got else None) == sub


def test_n_false_fire_is_not_hidden_by_a_trailing_question():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *skill_learn(), *bash("b1", "engram learn fact --x"),
          result("I saved a note. Should I continue with unit 2?")]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "false-fire"
    assert out["ended_on_question"] is True


def test_p_capture_then_question_is_classified_on_the_capture():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(), *bash("b1", "engram learn feedback --x"),
          result("Captured the lesson. Should I continue with unit 2?")]
    out = run(ev)
    assert out["label"] == "pass"
    assert out["ended_on_question"] is True


def test_parallel_dispatch_is_unscorable_window():
    ev = [init(), dispatch("a1", 1), dispatch("a2", 2), ret("a1", REPORT_N1),
          ret("a2", "Unit 2 done.\n\nLESSONS: none"), *skill_learn(), *tail(unit2=False)]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "unscorable-window"
    assert out["scored"] is False


def test_delegated_write_counts_for_a_p_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          asst(tu("g1", "Agent", {"subagent_type": "general-purpose", "description": "write the note",
                                  "prompt": "run engram learn"})),
          *bash("b1", "engram learn feedback --x", parent="g1"),
          user(tr("g1", "Wrote " + NOTE_PATH)), *tail()]
    out = run(ev)
    assert out["label"] == "pass"
    assert out["window_delegated_writes"] == 1


def test_delegated_write_is_a_false_fire_in_n():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1),
          asst(tu("g1", "Agent", {"subagent_type": "general-purpose", "prompt": "save it"})),
          *bash("b1", "engram learn fact --x", parent="g1"), user(tr("g1", "done")), *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "false-fire"


def test_unit_worker_fixture_events_never_count():
    ev = [init(), dispatch("a1", 1), *bash("x1", "engram learn fact --x", parent="a1"),
          ret("a1", REPORT_N1), *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "no-fire"


def test_unit1_redispatch_does_not_end_the_window():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1),
          asst(tu("a1b", "Agent", {"subagent_type": "unit-worker", "description": "Unit 1 — Front-matter dates (retry)",
                                   "prompt": "Unit 1 — please show the diff"})),
          ret("a1b", REPORT_P1), *skill_learn(), *bash("b1", "engram learn feedback --x"), *tail()]
    out = run(ev)
    assert out["label"] == "pass"
    assert out["unit1_redispatched"] is True


def test_unit2_dispatch_that_recaps_unit1_still_ends_the_window():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1),
          asst(tu("a2", "Agent", {"subagent_type": "unit-worker", "description": "Unit 2 — Slug rules",
                                  "prompt": "Unit 1 is done. Now do Unit 2 — Slug rules."})),
          ret("a2", "Unit 2 done.\n\nLESSONS: none"), *skill_learn(), *bash("b1", "engram learn feedback --x"),
          *tail(unit2=False)]
    assert run(ev)["label"] == "late"


@pytest.mark.parametrize("final,expected", [
    ("Unit 1 is back. See https://example.com/x?b=1", False),
    ("Unit 1 is back; the parser uses `dateOrNil?`", False),
    ("Unit 1 is back. Should I continue?", True),
    ("Unit 1 is back.\n\n**Want me to dispatch unit 2?**", True),
])
def test_question_stop_detection(final, expected):
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), result(final)]
    assert (run(ev)["label"] == "question-stop") is expected


def test_vault_note_list_confirms_the_write():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(), *bash("b1", "engram learn feedback --x"),
          *tail()]
    assert run(ev, vault_notes=["1.2026-09-30.quillfeather-dates-utc.md"])["label"] == "pass"
    out = run(ev, vault_notes=[])
    assert out["label"] == "write-unconfirmed"
    assert out["scored"] is True


def _main_link(tool_use_id, agent_id, agent_type="general-purpose", model="claude-sonnet-test"):
    return json.dumps({"type": "user", "message": {"content": [{"type": "tool_result", "tool_use_id": tool_use_id}]},
                       "toolUseResult": {"agentId": agent_id, "agentType": agent_type, "resolvedModel": model}})


def _sub_file(agent_id, tid, cmd, out):
    return "\n".join([
        json.dumps({"type": "user", "agentId": agent_id, "message": {"content": "do it"}}),
        json.dumps({"type": "assistant", "agentId": agent_id,
                    "message": {"content": [{"type": "tool_use", "id": tid, "name": "Bash", "input": {"command": cmd}}]}}),
        json.dumps({"type": "user", "agentId": agent_id,
                    "message": {"content": [{"type": "tool_result", "tool_use_id": tid, "content": out}]}}),
    ])


def test_delegated_write_seen_only_in_the_subagent_transcript_counts():
    """stream-json does not reliably carry subagent tool calls; the subagent session file does."""
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          asst(tu("g1", "Agent", {"subagent_type": "general-purpose", "prompt": "write the note"})),
          user(tr("g1", "done")), *tail()]
    main = good_session("GREEN") + [_main_link("g1", "agx")]
    sub = _sub_file("agx", "sb1", "engram learn feedback --x", NOTE_PATH)
    out = score.score_arm(lines(ev), main, cell="P1", arm="GREEN", unit1_report=REPORT_P1, lessons=LESSONS_P1,
                          other_session_texts=[sub], vault_notes=[NOTE_NAME])
    assert out["label"] == "pass"
    assert out["window_delegated_writes"] == 1
    n = score.score_arm(lines([init(), dispatch("a1", 1), ret("a1", REPORT_N1),
                               asst(tu("g1", "Agent", {"subagent_type": "general-purpose", "prompt": "x"})),
                               user(tr("g1", "done")), *tail()]),
                        main, cell="N1", arm="GREEN", unit1_report=REPORT_N1, lessons="none",
                        other_session_texts=[sub])
    assert n["label"] == "false-fire"


def test_fixture_subagent_transcript_writes_never_count_and_give_the_fixture_model():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *tail()]
    main = good_session("RED") + [_main_link("a1", "fx1", agent_type="unit-worker", model="claude-haiku-x")]
    sub = _sub_file("fx1", "sb1", "engram learn fact --x", NOTE_PATH)
    out = score.score_arm(lines(ev), main, cell="N1", arm="RED", unit1_report=REPORT_N1, lessons="none",
                          other_session_texts=[sub])
    assert out["label"] == "no-fire"
    assert "claude-haiku-x" in out["fixture_models"]


# ---------------------------------------------------------------------------
# ruling T5: route evidence / dispatch records are not lesson captures
# ---------------------------------------------------------------------------

ROUTE_NOTE = "/private/tmp/engram-arm.x/vault/2.2026-09-30.route-dispatch-single-unit-dispatch.md"
ROUTE_CMD = ('engram learn fact --slug route-dispatch-single-unit-dispatch --position top '
             '--situation "routing single-unit-dispatch work" --subject s --predicate p --object o')


def test_route_record_in_n_arm_is_no_fire():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *bash("r1", ROUTE_CMD, out=ROUTE_NOTE), *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none", vault_notes=["2.x.md"])
    assert out["label"] == "no-fire"
    assert out["window_route_records"] == [{"command": ROUTE_CMD, "note": ROUTE_NOTE}]


def test_route_record_plus_lesson_in_p_arm_is_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *bash("r1", ROUTE_CMD, out=ROUTE_NOTE), *skill_learn(),
          *bash("b1", "engram learn feedback --slug quillfeather-dates-utc --x"), *tail()]
    out = run(ev, vault_notes=[NOTE_NAME, "2.x.md"])
    assert out["label"] == "pass"
    assert len(out["window_route_records"]) == 1
    assert out["window_learn_writes"] == 1


def test_route_record_only_in_p_arm_is_no_fire():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *bash("r1", ROUTE_CMD, out=ROUTE_NOTE), *tail()]
    out = run(ev, vault_notes=["2.x.md"])
    assert out["label"] == "no-fire"
    assert out["scored"] is True


def test_route_evidence_detected_by_note_path_without_slug_flag():
    note = "/private/tmp/engram-arm.x/vault/3.2026-09-30.route-evidence-single-unit-dispatch.md"
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *bash("r1", "engram learn fact --position top --x", out=note),
          *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "no-fire"
    assert out["window_route_records"][0]["note"] == note


def test_route_record_that_only_fires_learn_skill_is_still_a_lesson_capture_signal():
    # Skill(learn) is always a lesson capture per T5, whatever it ends up writing
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *skill_learn(), *bash("r1", ROUTE_CMD, out=ROUTE_NOTE),
          *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none")
    assert out["label"] == "false-fire"


# ---------------------------------------------------------------------------
# fix round 2: re-review residuals N1-N6
# ---------------------------------------------------------------------------

N_KW = dict(cell="N1", arm="RED", report=REPORT_N1, lessons="none")


def test_n1_ingest_before_learn_in_the_same_command_is_a_sweep():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          *bash("b1", "engram ingest --auto && engram learn feedback --slug x --y"), *tail()]
    assert run(ev, vault_notes=[NOTE_NAME])["label"] == "fired-with-sweep"


def test_n1_learn_before_ingest_in_the_same_command_stays_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          *bash("b1", "engram learn feedback --slug x --y && engram ingest --auto"), *tail()]
    assert run(ev, vault_notes=[NOTE_NAME])["label"] == "pass"


@pytest.mark.parametrize("cmd", [
    "timeout 60 engram learn feedback --slug x",
    "timeout -s KILL 60 engram learn feedback --slug x",
    "env FOO=1 engram learn feedback --slug x",
    "env -i FOO=1 engram learn feedback --slug x",
    "command engram learn feedback --slug x",
    "nice -n 5 engram learn feedback --slug x",
    "nohup engram learn feedback --slug x",
    "time engram learn feedback --slug x",
    "bash -c 'engram learn feedback --slug x'",
    'sh -c "cd /tmp && engram learn feedback --slug x"',
    "echo `engram learn feedback --slug x`",
])
def test_n2_wrappers_are_parsed(cmd):
    assert "learn" in score.engram_subcommands(cmd)
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *bash("b1", cmd), *tail()]
    assert run(ev, **N_KW)["label"] == "false-fire"


def test_n2_backticks_inside_single_quotes_are_not_executed():
    assert score.engram_subcommands("engram query --phrase 'run `engram learn feedback` now'") == ["query"]


def test_n2_unresolvable_learn_mention_is_listed_for_audit():
    cmd = 'eval "$(printf %s engram) learn feedback --slug x"'
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *bash("b1", cmd, out="ok"), *tail()]
    out = run(ev, **N_KW)
    assert out["window_unparsed_learn_mentions"] == [cmd]


def test_n3_route_record_and_lesson_in_one_command_is_still_a_lesson():
    cmd = ROUTE_CMD + " && engram learn feedback --slug quillfeather-dates-utc --x"
    out_text = ROUTE_NOTE + "\n/private/tmp/engram-arm.x/vault/3.2026-09-30.quillfeather-dates-utc.md"
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1), *bash("b1", cmd, out=out_text), *tail()]
    out = run(ev, **N_KW)
    assert out["label"] == "false-fire"
    assert len(out["window_route_records"]) == 1
    assert out["window_route_records"][0]["note"] == ROUTE_NOTE


def test_n3_route_path_in_lesson_output_text_does_not_make_it_a_route_record():
    out_text = ("warning: similar note exists: " + ROUTE_NOTE +
                "\n/private/tmp/engram-arm.x/vault/3.2026-09-30.quillfeather-dates-utc.md")
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1),
          *bash("b1", "engram learn feedback --position top --x", out=out_text), *tail()]
    out = run(ev, **N_KW)
    assert out["label"] == "false-fire"
    assert out["window_route_records"] == []


def test_n4_qa_write_is_listed_not_counted():
    qa_path = "/private/tmp/engram-arm.x/vault/4.2026-09-30.why-utc.md"
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1),
          *bash("b1", "engram learn qa --question q --answer a", out=qa_path), *tail()]
    out = run(ev, **N_KW)
    assert out["label"] == "no-fire"
    assert out["window_qa_writes"] == [{"command": "engram learn qa --question q --answer a", "note": qa_path}]


@pytest.mark.parametrize("desc,unit", [
    ("Unit-1 retry", 1), ("unit_1: show diff", 1), ("Units 2", 2), ("Slug rules", 2), ("Lock file", 3),
    ("Unit 3 — Template cache", 3),
])
def test_n5_unit_identity_is_tolerant(desc, unit):
    step = ("use", {"name": "Agent", "input": {"subagent_type": "unit-worker", "description": desc, "prompt": "x"}})
    assert score._unit_of(step) == unit


def test_n5_prompt_body_mentioning_unit2_does_not_end_the_window():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1),
          asst(tu("a1b", "Agent", {"subagent_type": "unit-worker", "description": "Front-matter dates: show diff",
                                   "prompt": "Front-matter dates\n\nShow me the diff before I move on to Unit 2."})),
          ret("a1b", REPORT_P1), *skill_learn(), *bash("b1", "engram learn feedback --x"), *tail()]
    out = run(ev, vault_notes=[NOTE_NAME])
    assert out["label"] == "pass"
    assert out["unit1_redispatched"] is True


def test_n5_unidentified_dispatch_ends_the_window_and_is_flagged():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1),
          asst(tu("ax", "Agent", {"subagent_type": "unit-worker", "description": "next one", "prompt": "carry on"})),
          ret("ax", "Unit 2 done.\n\nLESSONS: none"), *skill_learn(), *bash("b1", "engram learn feedback --x"),
          *tail(unit2=False)]
    out = run(ev, vault_notes=[NOTE_NAME])
    assert out["label"] == "late"
    assert out["window_end_unit"] is None


@pytest.mark.parametrize("final", [
    "Unit 1 is back. Shall I go on? (y/n)",
    "Unit 1 is back. Continue? [y/N]",
    "Unit 1 is back (continue?)",
])
def test_n6_yes_no_suffixes_are_questions(final):
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), result(final)]
    assert run(ev)["label"] == "question-stop"


def test_n7_delegated_ingest_then_later_learn_call_is_a_sweep():
    """Nested steps of one delegated dispatch keep their own order: an ingest call followed by a later
    learn call inside the subagent is a sweep, not a fast-path write."""
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          asst(tu("g1", "Agent", {"subagent_type": "general-purpose", "prompt": "save the lesson"})),
          user(tr("g1", "done")), *tail()]
    main = good_session("GREEN") + [_main_link("g1", "agx")]
    sub = "\n".join([_sub_file("agx", "sb1", "engram ingest --auto", "ingested 12 files"),
                     _sub_file("agx", "sb2", "engram learn feedback --slug x", NOTE_PATH)])
    out = score.score_arm(lines(ev), main, cell="P1", arm="GREEN", unit1_report=REPORT_P1, lessons=LESSONS_P1,
                          other_session_texts=[sub], vault_notes=[NOTE_NAME])
    assert out["label"] == "fired-with-sweep"


def test_n7_delegated_learn_then_later_ingest_stays_pass():
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(),
          asst(tu("g1", "Agent", {"subagent_type": "general-purpose", "prompt": "save the lesson"})),
          user(tr("g1", "done")), *tail()]
    main = good_session("GREEN") + [_main_link("g1", "agx")]
    sub = "\n".join([_sub_file("agx", "sb2", "engram learn feedback --slug x", NOTE_PATH),
                     _sub_file("agx", "sb1", "engram ingest --auto", "ingested 12 files")])
    out = score.score_arm(lines(ev), main, cell="P1", arm="GREEN", unit1_report=REPORT_P1, lessons=LESSONS_P1,
                          other_session_texts=[sub], vault_notes=[NOTE_NAME])
    assert out["label"] == "pass"


def test_fixture_worker_real_tool_calls_never_count():
    """D5 amendment: the worker Reads/Edits/runs its check (and even an engram call) under its own
    dispatch; none of it is a capture, in the stream or in its session file."""
    fix = "/private/tmp/engram-arm.x/fixture/quillfeather"
    ev = [init(), dispatch("a1", 1),
          asst(tu("w1", "Read", {"file_path": fix + "/internal/frontmatter/date.go"}), parent="a1"),
          asst(tu("w2", "Edit", {"file_path": fix + "/internal/frontmatter/date.go"}), parent="a1"),
          *bash("w3", fix + "/run-tests", out="ok", parent="a1"),
          *bash("w4", "engram learn feedback --slug x", parent="a1"),
          ret("a1", REPORT_N1), *tail()]
    main = good_session("RED") + [_main_link("a1", "fx1", agent_type="unit-worker")]
    sub = _sub_file("fx1", "sb1", "engram learn feedback --slug y", NOTE_PATH)
    out = score.score_arm(lines(ev), main, cell="N1", arm="RED", unit1_report=REPORT_N1, lessons="none",
                          other_session_texts=[sub])
    assert out["label"] == "no-fire"
    p = score.score_arm(lines([init(), dispatch("a1", 1), *bash("w4", "engram learn feedback --x", parent="a1"),
                               ret("a1", REPORT_P1), *tail()]),
                        good_session("GREEN"), cell="P1", arm="GREEN", unit1_report=REPORT_P1, lessons=LESSONS_P1,
                        vault_notes=[NOTE_NAME])
    assert p["label"] == "no-fire"


def test_t9_worker_written_vault_notes_never_confirm_a_pass():
    """T9: only notes created by the orchestrator's own (or delegated non-worker) writes confirm a pass;
    notes the unit-worker wrote are listed for audit."""
    worker_note = "/private/tmp/engram-arm.x/vault/1.2026-09-30.worker-scribble.md"
    ev = [init(), dispatch("a1", 1), *bash("w1", "engram learn fact --slug worker-scribble", out=worker_note,
                                           parent="a1"),
          ret("a1", REPORT_P1), *skill_learn(),
          *bash("b1", "engram learn feedback --x"), *tail()]
    # the orchestrator's write claims NOTE_PATH, but only the worker's note is in the vault
    out = run(ev, vault_notes=["1.2026-09-30.worker-scribble.md"])
    assert out["label"] == "write-unconfirmed"
    assert out["worker_vault_notes"] == ["1.2026-09-30.worker-scribble.md"]
    ok = run(ev, vault_notes=["1.2026-09-30.worker-scribble.md", "1.2026-09-30.quillfeather-dates-utc.md"])
    assert ok["label"] == "pass"
    assert ok["confirmed_vault_notes"] == ["1.2026-09-30.quillfeather-dates-utc.md"]


def test_t9_worker_note_in_n_arm_is_listed_not_a_false_fire():
    worker_note = "/private/tmp/engram-arm.x/vault/1.2026-09-30.worker-scribble.md"
    ev = [init(), dispatch("a1", 1), *bash("w1", "engram learn fact --slug worker-scribble", out=worker_note,
                                           parent="a1"),
          ret("a1", REPORT_N1), *tail()]
    out = run(ev, cell="N1", arm="RED", report=REPORT_N1, lessons="none",
              vault_notes=["1.2026-09-30.worker-scribble.md"])
    assert out["label"] == "no-fire"
    assert out["worker_vault_notes"] == ["1.2026-09-30.worker-scribble.md"]
