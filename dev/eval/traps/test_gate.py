"""TDD for the trap regression gate: C3 seed covers all 5 conventions; verdict logic is correct."""
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import seed_c3
import gate_verdict as gv


def test_c3_seed_covers_all_five_conventions():
    objs = " ".join(n["object"] + n["subject"] for n in seed_c3.C3_NOTES)
    for form in ["NewRequestWithContext", "NO_COLOR", "t.Parallel(", "%w", "len("]:
        assert form in objs, f"C3 seed missing convention form {form}"
    assert len(seed_c3.C3_NOTES) == 5


def test_axis_all_pass_is_green():
    t = [{"pass": True, "contaminated": False}] * 5
    assert gv.axis_verdict(t, bar=5)["status"] == "GREEN"


def test_axis_one_valid_miss_is_red():
    t = [{"pass": True, "contaminated": False}] * 4 + [{"pass": False, "contaminated": False}]
    assert gv.axis_verdict(t, bar=5)["status"] == "RED"


def test_axis_high_contamination_is_inconclusive():
    t = [{"pass": True, "contaminated": True}] * 3 + [{"pass": True, "contaminated": False}] * 2
    assert gv.axis_verdict(t, bar=5)["status"] == "INCONCLUSIVE"   # 3/5 = 60% > 20%


def test_axis_contaminated_excluded_rest_pass_green():
    # 1 contaminated of 5 (20%, not over), remaining 4 all pass
    t = [{"pass": True, "contaminated": True}] + [{"pass": True, "contaminated": False}] * 4
    v = gv.axis_verdict(t, bar=5)
    assert v["valid"] == 4 and v["passed"] == 4 and v["status"] == "GREEN"


def test_gate_red_if_any_axis_red():
    axes = {"C3": {"status": "GREEN"}, "C6": {"status": "RED"}}
    assert gv.gate_verdict(axes)["verdict"] == "RED"


def test_gate_inconclusive_if_any_inconclusive_and_none_red():
    axes = {"C3": {"status": "GREEN"}, "C5": {"status": "INCONCLUSIVE"}}
    assert gv.gate_verdict(axes)["verdict"] == "INCONCLUSIVE"
