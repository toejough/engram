"""Unit tests for s3_score.split_tally (pure, no LLM, no I/O).

Run: python3 -m pytest test_s3_score.py -q  (or python3 test_s3_score.py)
"""
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from s3_score import split_tally, tally, verdict


def _row(arm, hit, pop="P1-n3"):
    return {"arm": arm, "hit": hit, "population": pop, "case_id": "X", "idx": 0}


def test_split_tally_groups_by_population():
    rows = (
        [_row("A", True, "P1-n3")] * 4 + [_row("A", False, "P1-n3")] * 1 +
        [_row("B", True, "P1-n3")] * 5 +
        [_row("A", True, "P2")] * 2 + [_row("A", False, "P2")] * 3 +
        [_row("B", False, "P2")] * 5
    )
    result = split_tally(rows)
    assert set(result.keys()) == {"P1-n3", "P2"}
    assert result["P1-n3"]["A"]["n"] == 5
    assert result["P1-n3"]["A"]["hits"] == 4
    assert result["P1-n3"]["B"]["n"] == 5
    assert result["P1-n3"]["B"]["hits"] == 5
    assert result["P2"]["A"]["hits"] == 2
    assert result["P2"]["B"]["hits"] == 0


def test_split_tally_empty_rows():
    assert split_tally([]) == {}


def test_split_tally_single_population():
    rows = [_row("A", True, "P3")] * 3 + [_row("B", False, "P3")] * 3
    result = split_tally(rows)
    assert list(result.keys()) == ["P3"]
    assert result["P3"]["A"]["rate"] == 1.0
    assert result["P3"]["B"]["rate"] == 0.0


def test_split_tally_unknown_population():
    rows = [{"arm": "A", "hit": True}]  # no 'population' key
    result = split_tally(rows)
    assert "unknown" in result


# Sanity-check re-exports from qanchor_score
def test_tally_reexport():
    rows = [{"arm": "A", "hit": True}] * 3 + [{"arm": "A", "hit": False}] * 1
    t = tally(rows)
    assert t["A"]["n"] == 4 and t["A"]["hits"] == 3


def test_verdict_reexport():
    rows = (
        [{"arm": "none", "hit": False}] * 10 +
        [{"arm": "A", "hit": True}] * 3 + [{"arm": "A", "hit": False}] * 7 +
        [{"arm": "B", "hit": True}] * 9 + [{"arm": "B", "hit": False}] * 1
    )
    v = verdict(tally(rows))
    assert v["headroom"] is True


if __name__ == "__main__":
    fns = [f for name, f in sorted(globals().items()) if name.startswith("test_")]
    for f in fns:
        f()
        print(f"PASS {f.__name__}")
    print(f"\n{len(fns)} passed")
