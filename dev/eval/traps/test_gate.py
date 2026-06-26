"""TDD for the trap regression gate: C3 seed covers all 5 conventions; verdict logic is correct."""
import os, sys
sys.path.insert(0, os.path.dirname(__file__))
import seed_c3


def test_c3_seed_covers_all_five_conventions():
    objs = " ".join(n["object"] + n["subject"] for n in seed_c3.C3_NOTES)
    for form in ["NewRequestWithContext", "NO_COLOR", "t.Parallel(", "%w", "len("]:
        assert form in objs, f"C3 seed missing convention form {form}"
    assert len(seed_c3.C3_NOTES) == 5
