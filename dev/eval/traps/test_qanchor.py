"""Unit tests for the pure scoring/verdict logic of the question-anchored delivery eval.

Mirrors the gate_verdict/test_gate split: the I/O orchestration lives in qanchor_eval.py; the
tally + verdict decision logic lives in qanchor_score.py and is exercised here with fixed rows (no
LLM, no subprocess). Run: python3 -m pytest test_qanchor.py -q  (or python3 test_qanchor.py)
"""
import math
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from qanchor_score import tally, verdict


def _rows(arm, hits, misses):
    return [{"pair": "p", "arm": arm, "idx": i, "hit": True} for i in range(hits)] + \
           [{"pair": "p", "arm": arm, "idx": hits + i, "hit": False} for i in range(misses)]


def test_tally_rate_and_sigma():
    t = tally(_rows("B", 8, 2))
    assert t["B"]["n"] == 10
    assert t["B"]["hits"] == 8
    assert abs(t["B"]["rate"] - 0.8) < 1e-9
    assert abs(t["B"]["sigma"] - math.sqrt(0.8 * 0.2 / 10)) < 1e-9


def test_verdict_b_wins_when_gap_exceeds_two_sigma():
    rows = _rows("none", 3, 27) + _rows("A", 6, 24) + _rows("B", 24, 6)
    v = verdict(tally(rows), none_ceiling=0.5)
    assert v["headroom"] is True
    assert v["status"].startswith("B_WINS")


def test_verdict_park_when_gap_below_two_sigma():
    rows = _rows("none", 3, 27) + _rows("A", 15, 15) + _rows("B", 17, 13)
    v = verdict(tally(rows), none_ceiling=0.5)
    assert v["headroom"] is True
    assert v["status"].startswith("PARK")


def test_verdict_underpowered_when_none_floor_too_high():
    # cold already applies the principle -> no headroom -> a null result is NOT a tie.
    rows = _rows("none", 20, 10) + _rows("A", 12, 18) + _rows("B", 26, 4)
    v = verdict(tally(rows), none_ceiling=0.5)
    assert v["headroom"] is False
    assert v["status"].startswith("UNDERPOWERED")


if __name__ == "__main__":
    fns = [f for name, f in sorted(globals().items()) if name.startswith("test_")]
    for f in fns:
        f()
        print(f"PASS {f.__name__}")
    print(f"\n{len(fns)} passed")
