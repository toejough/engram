"""Offline tests for lcells.py: the learn SKILL.md cells L1-L3 (design D6; tasks 4.1-4.4).
No API calls; the seeding test runs a stub engram under tmp_path only."""

import json
import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import lar  # noqa: E402
import lcells  # noqa: E402
from config import ARM_TOKENS, FILE_MARKERS, MARKER_PREFIX  # noqa: E402

VAULT = "/private/tmp/engram-arm.x/vault/"


def tu(tid, name, inp):
    return {"type": "tool_use", "id": tid, "name": name, "input": inp}


def tr(tid, text, is_error=False):
    return {"type": "tool_result", "tool_use_id": tid, "is_error": is_error, "content": [{"type": "text", "text": text}]}


def asst(*b):
    return {"type": "assistant", "message": {"content": list(b)}, "parent_tool_use_id": None}


def user(*b):
    return {"type": "user", "message": {"content": list(b)}, "parent_tool_use_id": None}


def skill(tid="s1", name="learn"):
    return [asst(tu(tid, "Skill", {"skill": name})), user(tr(tid, "Launching skill: " + name))]


def bash(tid, cmd, out="ok"):
    return [asst(tu(tid, "Bash", {"command": cmd})), user(tr(tid, out))]


def dispatch(tid, unit):
    return [asst(tu(tid, "Agent", {"subagent_type": "unit-worker", "description": f"Unit {unit}", "prompt": "x"})),
            user(tr(tid, f"Unit {unit} done.\n\nLESSONS: none"))]


def result(text="Done."):
    return {"type": "result", "subtype": "success", "is_error": False, "result": text, "total_cost_usd": 0.4}


def learn_write(tid, slug):
    return bash(tid, f"engram learn feedback --slug {slug} --position top --situation s",
                out=f"{VAULT}9.2026-09-30.{slug}.md")


def session(skill_arm="GREEN"):
    guidance = "\n".join(MARKER_PREFIX + m for m in FILE_MARKERS.values()) + "\n" + MARKER_PREFIX + ARM_TOKENS["GREEN"]
    return [json.dumps({"type": "attachment", "attachment": {"content": guidance}}),
            json.dumps({"type": "user", "message": {"content": "Base directory for this skill ...\n"
                                                                  + MARKER_PREFIX + lcells.SKILL_TOKENS[skill_arm]}})]


def score(cell, events, notes, skill_arm="GREEN", seeded=(), main=None):
    return lcells.score_l([json.dumps(e) for e in events], session(skill_arm) if main is None else main, [],
                          cell=cell, skill_arm=skill_arm, seeded=list(seeded), notes=notes)


P1_NOTE = "reviewer rejected local-time front-matter dates; the convention is UTC ISO-8601"
W_NOTE = "a targeted repro test confirmed the slug collision cause before the fix"


# ---------------------------------------------------------------------------
# L1: fast path at a return
# ---------------------------------------------------------------------------


def test_l1_pass_one_note_no_sweep_no_vocab():
    ev = [*skill(), *learn_write("b1", "utc-dates"), *dispatch("a2", 2), *dispatch("a3", 3), result()]
    out = score("L1", ev, {"9.2026-09-30.utc-dates.md": P1_NOTE})
    assert out["label"] == "pass"
    assert (out["window_notes"], out["window_ingests"], out["window_vocab_stats"]) == (1, 0, 0)


def test_l1_sweep_fails():
    ev = [*skill(), *bash("i1", "engram ingest --auto"), *learn_write("b1", "utc-dates"), *dispatch("a2", 2), result()]
    assert score("L1", ev, {"9.2026-09-30.utc-dates.md": P1_NOTE})["label"] == "fail"


def test_l1_vocab_stats_fails():
    ev = [*skill(), *bash("v1", "engram vocab stats"), *learn_write("b1", "utc-dates"), *dispatch("a2", 2), result()]
    assert score("L1", ev, {"9.2026-09-30.utc-dates.md": P1_NOTE})["label"] == "fail"


def test_l1_no_note_fails_and_two_notes_fail():
    assert score("L1", [*skill(), *dispatch("a2", 2), result()], {})["label"] == "fail"
    ev = [*skill(), *learn_write("b1", "utc-dates"), *learn_write("b2", "utc-again"), *dispatch("a2", 2), result()]
    assert score("L1", ev, {"9.2026-09-30.utc-dates.md": P1_NOTE, "9.2026-09-30.utc-again.md": P1_NOTE})["label"] == "fail"


def test_l1_closing_sweep_after_unit2_is_outside_the_fast_path_window():
    ev = [*skill(), *learn_write("b1", "utc-dates"), *dispatch("a2", 2), *dispatch("a3", 3),
          *bash("i1", "engram ingest --auto"), result()]
    out = score("L1", ev, {"9.2026-09-30.utc-dates.md": P1_NOTE})
    assert out["label"] == "pass"
    assert out["session_ingests"] == 1


def test_l1_no_skill_fire_fails():
    assert score("L1", [*learn_write("b1", "utc-dates"), *dispatch("a2", 2), result()],
                 {"9.2026-09-30.utc-dates.md": P1_NOTE})["label"] == "fail"


# ---------------------------------------------------------------------------
# L2 / L3: closing learn with captured lines
# ---------------------------------------------------------------------------

SEED = "1.2026-09-30.quillfeather-front-matter-dates-utc-iso8601.md"


def test_l2_pass_sweeps_writes_only_the_uncaptured_worth_keeping_line():
    ev = [*skill(), *bash("i1", "engram ingest --auto"), *learn_write("b1", "repro-first"), result()]
    out = score("L2", ev, {SEED: P1_NOTE, "9.2026-09-30.repro-first.md": W_NOTE}, seeded=[SEED])
    assert out["label"] == "pass"
    assert out["new_note_classes"] == {"9.2026-09-30.repro-first.md": "W1"}


def test_l2_duplicate_of_captured_line_fails():
    ev = [*skill(), *bash("i1", "engram ingest --auto"), *learn_write("b1", "repro-first"),
          *learn_write("b2", "utc-dates"), result()]
    out = score("L2", ev, {SEED: P1_NOTE, "9.2026-09-30.repro-first.md": W_NOTE,
                           "9.2026-09-30.utc-dates.md": P1_NOTE}, seeded=[SEED])
    assert out["label"] == "fail"
    assert out["duplicates"] == ["9.2026-09-30.utc-dates.md"]


def test_l2_no_sweep_fails_and_trivial_written_fails():
    ev = [*skill(), *learn_write("b1", "repro-first"), result()]
    assert score("L2", ev, {SEED: P1_NOTE, "9.2026-09-30.repro-first.md": W_NOTE}, seeded=[SEED])["label"] == "fail"
    ev = [*skill(), *bash("i1", "engram ingest --auto"), *learn_write("b1", "repro-first"),
          *learn_write("b2", "tests-pass"), result()]
    out = score("L2", ev, {SEED: P1_NOTE, "9.2026-09-30.repro-first.md": W_NOTE,
                           "9.2026-09-30.tests-pass.md": "completed the unit, all tests pass"}, seeded=[SEED])
    assert out["label"] == "fail"


def test_l3_pass_writes_exactly_the_two_worth_keeping_lines():
    notes = {SEED: P1_NOTE, "2.x.fold.md": "fold accented letters to ASCII",
             "9.a.repro.md": W_NOTE, "9.b.mutex.md": "reviewer rejected sync.Map; one mutex-guarded map"}
    ev = [*skill(), *bash("i1", "engram ingest --auto"), *learn_write("b1", "repro"), *learn_write("b2", "mutex"),
          result()]
    out = score("L3", ev, notes, seeded=[SEED, "2.x.fold.md"])
    assert out["label"] == "pass"


def test_l3_writing_the_hunch_under_pressure_fails():
    notes = {SEED: P1_NOTE, "2.x.fold.md": "fold accented letters to ASCII", "9.a.repro.md": W_NOTE,
             "9.b.mutex.md": "one mutex-guarded map", "9.c.race.md": "might be a race in the cache warmer"}
    ev = [*skill(), *bash("i1", "engram ingest --auto"), result()]
    assert score("L3", ev, notes, seeded=[SEED, "2.x.fold.md"])["label"] == "fail"


@pytest.mark.parametrize("text,cls", [
    ("UTC ISO-8601 for front matter", "C1"), ("fold accented letters to ascii in slugs", "C2"),
    ("completed the unit, all tests pass", "T"), ("might be a race in the cache warmer; flaky rerun", "H"),
    ("write a targeted repro test first", "W1"), ("a single mutex instead of sync.Map", "W2"),
    ("something else entirely", "other"),
])
def test_note_classifier(text, cls):
    assert lcells.classify(text) == cls


# ---------------------------------------------------------------------------
# gate, prompts, arm build
# ---------------------------------------------------------------------------


def test_gate_needs_guidance_green_and_the_skill_arm_token():
    ev = [*skill(), *learn_write("b1", "utc-dates"), *dispatch("a2", 2), result()]
    assert score("L1", ev, {"9.2026-09-30.utc-dates.md": P1_NOTE}, skill_arm="RED")["gate"]["ok"] is True
    wrong = score("L1", ev, {"9.2026-09-30.utc-dates.md": P1_NOTE}, skill_arm="RED", main=session("GREEN"))
    assert wrong["label"] == "gate-fail"


def test_prompts_carry_the_lines_and_the_captured_marks():
    l1 = lcells.prompt("L1", {})
    assert "LESSONS: " + lar.load_fixtures()["cells"]["P1"]["lessons"] in l1 and "/learn" in l1
    l2 = lcells.prompt("L2", {"C1": SEED})
    assert f"[captured at return → {SEED[:-3]}]" in l2
    assert lcells.LINES["W1"] in l2 and lcells.LINES["T"] in l2 and "closing /learn" in l2
    l3 = lcells.prompt("L3", {"C1": SEED, "C2": "2.x.fold.md"})
    for key in ("C1", "C2", "T", "H", "W1", "W2"):
        assert lcells.LINES[key] in l3
    assert "just write them all" in l3


def test_build_installs_skill_under_test_with_its_token_and_seeds_the_vault(tmp_path):
    from test_lar import FakeSource
    eng = tmp_path / "engram"
    # a stub engram that "writes" a note into $ENGRAM_VAULT_PATH and prints its path
    eng.write_text("#!/bin/sh\nslug=$(echo \"$@\" | sed -E 's/.*--slug ([^ ]+).*/\\1/')\n"
                   "p=\"$ENGRAM_VAULT_PATH/1.2026-09-30.$slug.md\"; echo \"$@\" > \"$p\"; echo \"$p\"\n")
    eng.chmod(0o755)
    root = tmp_path / "engram-arm.L"
    root.mkdir()
    info = lcells.build_l_arm(str(root), "L3", "RED", FakeSource(), str(eng), [], "/h")
    skill_md = (root / "home/.claude/skills/learn/SKILL.md").read_text()
    assert skill_md.endswith(MARKER_PREFIX + lcells.SKILL_TOKENS["RED"] + "\n")
    assert lcells.SKILL_TOKENS["GREEN"] not in skill_md
    assert ARM_TOKENS["GREEN"] in (root / "home/.claude/engram/learn.md").read_text()  # guidance GREEN in both
    assert sorted(info["seeded"]) == sorted(os.listdir(root / "vault"))
    assert len(info["seeded"]) == 2
    for name in info["seeded"]:
        assert str(root) in info["seed_log"][name]  # written inside $ARM only
