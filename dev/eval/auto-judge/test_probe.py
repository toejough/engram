"""Tests of the pure parts of the probe of the turn judgement (dev/eval/auto-judge/probe.py)."""
import json
import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import probe


def test_record_of_writes_a_line_for_each_row_and_a_result_for_a_tool():
    rows = [{"user": "run it"}, {"tool": "Bash", "result": "done"}, {"agent": "it ran"}]

    lines = [json.loads(line) for line in probe.record_of(rows).splitlines()]

    assert [line["type"] for line in lines] == ["user", "assistant", "user", "assistant"]
    assert lines[0]["message"]["content"] == "run it"
    used = lines[1]["message"]["content"][0]
    assert (used["type"], used["name"]) == ("tool_use", "Bash")
    result = lines[2]["message"]["content"][0]
    assert (result["type"], result["tool_use_id"], result["content"]) == ("tool_result", used["id"], "done")
    assert lines[3]["message"]["content"] == [{"type": "text", "text": "it ran"}]


def test_record_of_refuses_a_row_that_is_none_of_the_three():
    with pytest.raises(SystemExit):
        probe.record_of([{"system": "reminder"}])


@pytest.mark.parametrize("reply, expect, kind", [
    ("[]", {}, "none"),
    ("[]", {"lesson": ["flag"]}, "none (a miss)"),
    ("[]", {"covered": True}, "none (a miss)"),
    ('[{"slug": "x", "covered": null, "action": "pass the flag"}]', {"lesson": ["flag"]}, "lesson"),
    ('[{"slug": "x", "covered": null, "action": "pass it"}]', {"lesson": ["flag"]}, "lesson without the words (a miss)"),
    ('[{"slug": "x", "covered": null}]', {}, "lesson (a miss)"),
    ('[{"slug": "x", "covered": "1.note.md"}]', {"covered": True}, "covered"),
    ('[{"slug": "x", "covered": null}]', {"covered": True}, "written (a miss)"),
    ('```json\n[{"action": "FLAG"}]\n```', {"lesson": ["flag"]}, "lesson"),
    ("not json", {}, "invalid"),
    ('{"a": 1}', {}, "invalid"),
])
def test_verdict_names_what_a_reply_holds(reply, expect, kind):
    assert probe.verdict(reply, expect) == kind


def test_the_fixtures_hold_no_secret_whole_and_no_placeholder_once_loaded():
    with open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures.json"), encoding="utf-8") as handle:
        text = handle.read()
    for secret in probe.SECRETS.values():
        assert secret not in text

    loaded = probe.load_fixtures()

    assert "{{" not in json.dumps(loaded)
    assert any(secret in json.dumps(loaded) for secret in probe.SECRETS.values())


def test_every_fixture_has_rows_a_turn_and_an_expectation():
    for name, fixture in probe.load_fixtures().items():
        assert fixture["rows"], name
        assert fixture["turn"]["text"] and fixture["turn"]["answer"], name
        assert isinstance(fixture["expect"], dict), name
        assert fixture["rows"][-1].keys() == {"agent"}, name


def test_drop_once_removes_one_occurrence_and_refuses_any_other_count():
    assert probe.drop_once("a b c", "b") == "a  c"
    with pytest.raises(SystemExit):
        probe.drop_once("a b b", "b")
    with pytest.raises(SystemExit):
        probe.drop_once("a b", "z")
