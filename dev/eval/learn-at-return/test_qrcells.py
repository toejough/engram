"""Offline tests for qrcells.py: the please (Q1) and route (R1) SKILL.md cells (design D6; tasks 5.1-5.2)."""

import json
import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import lar  # noqa: E402
import qrcells  # noqa: E402
from config import ARM_TOKENS, FILE_MARKERS, MARKER_PREFIX, PIN  # noqa: E402
from test_score import (LESSONS_P1, NOTE_NAME, asst, bash, dispatch, init, ret, result,  # noqa: E402
                        skill_learn, tail, tu, user, tr)

FX = lar.load_fixtures()
REPORT = lar.unit1_report(FX, "P1", "quillfeather")


def main_session(skill, arm):
    guidance = "\n".join(MARKER_PREFIX + m for m in FILE_MARKERS.values()) + "\n" + MARKER_PREFIX + ARM_TOKENS["GREEN"]
    return [json.dumps({"type": "attachment", "attachment": {"content": guidance}}),
            json.dumps({"type": "user", "message": {"content": MARKER_PREFIX + qrcells.SKILL_TOKENS[skill][arm]}})]


def run(cell, arm, events, notes=(NOTE_NAME,), main=None):
    skill = qrcells.CELL_SKILL[cell]
    return qrcells.score_qr([json.dumps(e) for e in events], main_session(skill, arm) if main is None else main, [],
                            cell=cell, skill_arm=arm, vault_notes=list(notes))


def load(name):
    return [asst(tu("k0", "Skill", {"skill": name})), user(tr("k0", "Launching skill: " + name))]


def test_q1_pass_with_captured_mark():
    ev = [init(), *load("please"), dispatch("a1", 1), ret("a1", REPORT), *skill_learn(),
          *bash("b1", "engram learn feedback --x"),
          asst({"type": "text", "text": "Running list: unit 1 — reviewer rejected… [captured → 1.utc]"}), *tail()]
    out = run("Q1", "GREEN", ev)
    assert out["label"] == "pass"
    assert out["captured_mark"] is True


def test_q1_fire_without_captured_mark_is_reported():
    ev = [init(), *load("please"), dispatch("a1", 1), ret("a1", REPORT), *skill_learn(),
          *bash("b1", "engram learn feedback --x"), *tail()]
    out = run("Q1", "GREEN", ev)
    assert out["label"] == "pass" and out["captured_mark"] is False
    # ruling T14 (Joe, 2026-10-01): the captured mark is no longer part of the Q bar; still recorded
    assert out["meets_q1_bar"] is True


def test_r1_no_fire_before_next_dispatch():
    ev = [init(), *load("route"), dispatch("a1", 1), ret("a1", REPORT),
          *bash("r1", "engram learn fact --slug route-dispatch-single-unit --position top",
                out="/private/tmp/engram-arm.x/vault/2.2026-09-30.route-dispatch-single-unit.md"), *tail()]
    out = run("R1", "RED", ev, notes=["2.2026-09-30.route-dispatch-single-unit.md"])
    assert out["label"] == "no-fire"
    assert len(out["window_route_records"]) == 1


def test_gate_requires_the_skill_token_of_the_arm():
    ev = [init(), *load("route"), dispatch("a1", 1), ret("a1", REPORT), *tail()]
    assert run("R1", "RED", ev, main=main_session("route", "GREEN"))["label"] == "gate-fail"
    assert run("R1", "RED", ev, main=main_session("please", "RED"))["label"] == "gate-fail"


@pytest.mark.parametrize("cell,skill,pressure", [("Q1", "please", False), ("Q1P", "please", True),
                                                   ("R1", "route", False)])
def test_prompts(cell, skill, pressure):
    p = qrcells.prompt(cell)
    assert f"`{skill}` skill" in p and "unit-worker" in p
    assert ("Keep moving, we're behind." in p) is pressure
    assert "LESSONS" not in p and "learn" not in p.lower()


def test_build_installs_the_skill_under_test_pinned_elsewhere(tmp_path):
    from test_lar import FakeSource
    eng = tmp_path / "e"
    eng.write_text("")
    root = tmp_path / "engram-arm.Q"
    root.mkdir()
    info = qrcells.build_qr_arm(str(root), "Q1", "RED", FakeSource(), str(eng), [], "/h")
    please = (root / "home/.claude/skills/please/SKILL.md").read_text()
    assert please.endswith(MARKER_PREFIX + qrcells.SKILL_TOKENS["please"]["RED"] + "\n")
    learn = (root / "home/.claude/skills/learn/SKILL.md").read_text()
    assert "LAR-" not in learn  # learn SKILL.md stays pinned and untokened in Q/R cells
    assert ARM_TOKENS["GREEN"] in (root / "home/.claude/engram/learn.md").read_text()
    meta = next(t for t in info["texts"] if t["path"] == "skills/please/SKILL.md")
    assert meta["commit"] == PIN
    assert info["prompt"] == qrcells.prompt("Q1")


def test_parse_specs():
    s = lar.parse_arm_spec("Q1P:GREEN:worktree", 0)
    assert (s.cell, s.arm, s.domain) == ("Q1P", "GREEN", "quillfeather")
    with pytest.raises(lar.HarnessError):
        lar.parse_arm_spec("R1:GREEN:pin", 0)


def test_please_prompt_is_cost_bounded():
    p = qrcells.prompt("Q1")
    assert "gate B result" in p and "ends once Unit 2's report is in" in p and "do not dispatch Unit 3" in p


def test_not_captured_status_is_not_a_captured_mark():
    ev = [init(), *load("please"), dispatch("a1", 1), ret("a1", REPORT), *skill_learn(),
          *bash("b1", "engram learn feedback --x"),
          asst({"type": "text", "text": "List: unit 1 — reviewer rejected… — not captured"}), *tail()]
    assert run("Q1", "GREEN", ev)["captured_mark"] is False
    ev2 = ev[:-4] + [asst({"type": "text", "text": "List: unit 1 — status: captured → 1.utc"})] + ev[-4:]
    assert run("Q1", "GREEN", ev2)["captured_mark"] is True
