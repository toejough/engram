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


def test_split_costs_warm_separates_recall_from_build():
    recall_res = {"total_cost_usd": 0.5}
    rounds = [{"cost": 1.0}, {"cost": 0.5}]
    rc, bc = harness.split_costs(recall_res, rounds)
    assert rc == 0.5
    assert bc == 1.5          # build = sum(rounds), recall excluded


def test_split_costs_cold_recall_is_zero():
    rounds = [{"cost": 2.0}]
    rc, bc = harness.split_costs(None, rounds)
    assert rc == 0.0
    assert bc == 2.0
