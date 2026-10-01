"""Tests for digest.py: the per-arm session digest that replaces the pruned session JSONLs
(final-review finding 9) while keeping every input the scorer reads."""

import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import digest  # noqa: E402
import score  # noqa: E402
from config import ARM_TOKENS, FILE_MARKERS, MARKER_PREFIX  # noqa: E402
from test_score import (LESSONS_P1, NOTE_PATH, REPORT_N1, REPORT_P1, _main_link, _sub_file, asst, init,  # noqa: E402
                        lines, ret, dispatch, skill_learn, tail, tr, tu, user)


def write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(text)


def arm_dir(tmp_path, main_records, sub_text):
    d = tmp_path / "00-P1-GREEN"
    write(str(d / "session" / "-w" / "s1.jsonl"), "\n".join(json.dumps(r) for r in main_records) + "\n")
    write(str(d / "session" / "-w" / "s1" / "subagents" / "agent-x.jsonl"), sub_text)
    return d


def test_digest_keeps_gate_markers_links_and_subagent_calls_and_rescoring_matches(tmp_path):
    guidance = ("lots of guidance prose\n" + "\n".join(MARKER_PREFIX + m for m in FILE_MARKERS.values()) + "\n"
                + MARKER_PREFIX + ARM_TOKENS["GREEN"] + "\nmore prose")
    main = [{"type": "attachment", "attachment": {"type": "instructions", "files": [{"content": guidance}]}},
            {"type": "user", "message": {"content": "skill body\n" + MARKER_PREFIX + "LAR-LSK-GREEN-4T9X"}},
            json.loads(_main_link("g1", "agx"))]
    sub = _sub_file("agx", "sb1", "engram learn feedback --slug x", NOTE_PATH)
    d = arm_dir(tmp_path, main, sub)
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_N1),
          asst(tu("g1", "Agent", {"subagent_type": "general-purpose", "prompt": "x"})), user(tr("g1", "done")), *tail()]
    full_main = [open(d / "session" / "-w" / "s1.jsonl").read()]
    full_sub = [sub]
    before = score.score_arm(lines(ev), full_main, "N1", "GREEN", REPORT_N1, "none", full_sub)

    digest.write_digest(str(d))
    main_d, sub_d = digest.read_digest(str(d))
    after = score.score_arm(lines(ev), main_d, "N1", "GREEN", REPORT_N1, "none", sub_d)
    assert before["label"] == after["label"] == "false-fire"  # the delegated write survives the digest
    assert after["gate"]["ok"] is True
    assert "LAR-LSK-GREEN-4T9X" in "\n".join(main_d)
    assert "lots of guidance prose" not in "\n".join(main_d)  # prose is dropped, markers kept
    total = sum(os.path.getsize(os.path.join(str(d), f)) for f in os.listdir(str(d)) if f.startswith("session-digest"))
    assert total < len(full_main[0]) + len(sub)


def test_gate_still_fails_from_a_digest_when_markers_were_missing(tmp_path):
    main = [{"type": "attachment", "attachment": {"type": "instructions", "files": [{"content": "no markers"}]}}]
    d = arm_dir(tmp_path, main, "")
    digest.write_digest(str(d))
    main_d, sub_d = digest.read_digest(str(d))
    ev = [init(), dispatch("a1", 1), ret("a1", REPORT_P1), *skill_learn(), *tail()]
    assert score.score_arm(lines(ev), main_d, "P1", "GREEN", REPORT_P1, LESSONS_P1, sub_d)["label"] == "gate-fail"
