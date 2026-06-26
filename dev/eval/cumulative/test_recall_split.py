"""TDD for the $METER session-split: build_prompt can omit the recall directive while keeping
the checklist gating, and a recall-only prompt exists for the separate recall call."""
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import harness


def test_build_prompt_include_recall_false_omits_recall_directive():
    p = harness.build_prompt("todo", "add/list", "skill", include_recall=False)
    assert "/recall" not in p
    assert "go mod init todo" in p  # still a real build prompt


def test_build_prompt_include_recall_false_keeps_checklist_gating():
    p = harness.build_prompt("todo", "add/list", "skill", checklist=True, include_recall=False)
    assert "checklist" in p.lower()           # gating block survives
    assert "/recall" not in p                 # but no recall directive


def test_recall_only_prompt_invokes_recall_and_stops_without_building():
    p = harness.recall_only_prompt("todo")
    assert "/recall" in p
    assert "go mod init" not in p             # must NOT build
    assert "STOP" in p.upper() or "do not write" in p.lower()
