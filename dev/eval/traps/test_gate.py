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


def test_normalize_c3_applied_pass_nobuild_contaminated():
    rows = [{"verdict": "applied"}, {"verdict": "trap"}, {"verdict": "nobuild"}]
    out = gv.normalize("C3", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1] == {"pass": False, "contaminated": False}
    assert out[2]["contaminated"] is True


def test_normalize_c4i_warmxxp_built_supersession_passes():
    rows = [{"arm": "cold", "built": True, "score": {"supersession_correct": False}},
            {"arm": "warm-XXp", "built": True, "score": {"supersession_correct": True}}]
    out = gv.normalize("C4i", rows)               # cold filtered out
    assert out == [{"pass": True, "contaminated": False}]


def test_normalize_c4i_unbuilt_is_contaminated_no_crash():
    rows = [{"arm": "warm-XXp", "built": False, "score": None}]
    out = gv.normalize("C4i", rows)
    assert out[0]["contaminated"] is True and out[0]["pass"] is False


def test_normalize_c5_unbuilt_contaminated():
    rows = [{"built": True, "honored": True}, {"built": False, "honored": None}]
    out = gv.normalize("C5", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1]["contaminated"] is True


def test_normalize_c6_empty_answer_is_contaminated():
    rows = [{"hit": True, "answer": "HIT because..."}, {"hit": False, "answer": ""}]
    out = gv.normalize("C6", rows)
    assert out[0] == {"pass": True, "contaminated": False}
    assert out[1]["contaminated"] is True


def test_normalize_unknown_axis_raises():
    import pytest
    with pytest.raises(ValueError):
        gv.normalize("C9", [])
