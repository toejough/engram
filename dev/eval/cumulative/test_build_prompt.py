"""TDD for Lever 4: the checklist flag adds a gating self-verification block to the build prompt,
while leaving the soft handoff (checklist=False) unchanged. The recall directive is present in both."""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import harness

APP, IFACE = "todo", "add <text>; list; done <id>"
GATING = "verify your code satisfies"  # distinctive of the checklist self-verification block


def test_checklist_adds_gating():
    p = harness.build_prompt(APP, IFACE, "skill", checklist=True)
    assert GATING.lower() in p.lower(), "checklist=True must add the gating self-verification block"
    assert "/recall" in p, "checklist arm must still carry the recall directive"
    assert "explicit checklist" in p.lower()


def test_soft_lacks_gating():
    p = harness.build_prompt(APP, IFACE, "skill", checklist=False)
    assert GATING.lower() not in p.lower(), "soft handoff must NOT contain the gating block (contrast)"
    assert "/recall" in p, "soft arm still carries the recall directive"


def test_cold_unchanged():
    p = harness.build_prompt(APP, IFACE, "none", checklist=False)
    assert "/recall" not in p and GATING.lower() not in p.lower()
