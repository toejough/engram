"""Aggregation + metrics report for the recency-value harness (#646).

Per-arm metrics from a list of trial dicts (recency_value.run_trial output). Diagnostic
sub-metrics (note 83) are conditioned on recall_fired so a retrieval miss is never scored as a
synthesis success; efficiency is judged only over correct, unassisted trials (note 292); raw
cost is reported over ALL valid trials (the issue's raw-cost requirement) alongside the judged
efficiency view, never collapsed together.

Usage:
  python3 recency_value_agg.py <on-trials.json> <off-trials.json> [...]
"""
import argparse
import json
import statistics
import sys


def _mean(values):
    values = [v for v in values if v is not None]
    return sum(values) / len(values) if values else None


def _spread(values):
    """Population stdev of correct-only trials, for sizing a gap against the within-arm spread
    (note 292 / R4 - a gap below this is underpowered, not a tie). 0.0 when <2 samples (a
    single-trial pilot has no spread to size against)."""
    values = [v for v in values if v is not None]
    return statistics.pstdev(values) if len(values) >= 2 else 0.0


def aggregate(trials):
    """Per-arm metrics dict for one arm's list of trial dicts."""
    valid = [t for t in trials if t.get("phase1_ok")]
    n_valid = len(valid)

    fired = [t for t in valid if t.get("recall_fired")]
    correct_valid = [t for t in valid if t.get("correct")]

    recall_fired_rate = len(fired) / n_valid if n_valid else None
    correct_rate_all = len(correct_valid) / n_valid if n_valid else None
    correct_rate_fired = (
        sum(1 for t in fired if t.get("correct")) / len(fired) if fired else None
    )
    # Dual surfacing rates (FIX 2), over fired trials (plan Task 3). None-safe when no trial
    # fired / payload was empty (EDGE): a missing key is falsy, and an empty `fired` yields None.
    surfaced_any_rate = (
        sum(1 for t in fired if t.get("surfaced_any")) / len(fired) if fired else None
    )
    surfaced_via_recency_rate = (
        sum(1 for t in fired if t.get("surfaced_via_recency")) / len(fired) if fired else None
    )

    raw_cost = {
        "cost_usd": _mean([t.get("phase2_cost") for t in valid]),
        "turns": _mean([t.get("phase2_turns") for t in valid]),
        "dur_ms": _mean([t.get("phase2_dur_ms") for t in valid]),
    }

    if correct_valid:
        efficiency = {
            "cost_usd": _mean([t.get("phase2_cost") for t in correct_valid]),
            "turns": _mean([t.get("phase2_turns") for t in correct_valid]),
            "dur_ms": _mean([t.get("phase2_dur_ms") for t in correct_valid]),
        }
        efficiency_spread = {
            "cost_usd": _spread([t.get("phase2_cost") for t in correct_valid]),
            "turns": _spread([t.get("phase2_turns") for t in correct_valid]),
            "dur_ms": _spread([t.get("phase2_dur_ms") for t in correct_valid]),
        }
    else:
        efficiency = None
        efficiency_spread = None

    return {
        "n_valid": n_valid,
        "recall_fired_rate": recall_fired_rate,
        "correct_rate_all": correct_rate_all,
        "correct_rate_fired": correct_rate_fired,
        "surfaced_any_rate": surfaced_any_rate,
        "surfaced_via_recency_rate": surfaced_via_recency_rate,
        "raw_cost": raw_cost,
        "efficiency": efficiency,
        "verdict_inputs": {
            "correct_rate_all": correct_rate_all,
            "n_correct": len(correct_valid),
            "n_phase1_learned": sum(1 for t in valid if t.get("phase1_learned")),
            "efficiency_spread": efficiency_spread,
        },
    }


def _pct(value):
    return f"{value * 100:.0f}%" if value is not None else "n/a"


def _fmt_money(value):
    return f"${value:.2f}" if value is not None else "n/a"


def _fmt_num(value, digits=1):
    return f"{value:.{digits}f}" if value is not None else "n/a"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("files", nargs="+", help="per-arm trial JSON files (recency_value.py --out)")
    args = parser.parse_args()

    header = (f"{'arm':5} {'n_valid':>7} {'fired%':>7} {'correct_all%':>13} "
              f"{'correct_fired%':>15} {'surf_any%':>10} {'surf_recency%':>14} "
              f"{'$raw(all)':>10} {'$(correct)':>11} {'turns(correct)':>15} {'ms(correct)':>12}")
    print(header)

    for path in args.files:
        with open(path) as f:
            trials = json.load(f)

        arm = trials[0]["arm"] if trials else "?"
        result = aggregate(trials)
        efficiency = result["efficiency"] or {}

        row = (f"{arm:5} {result['n_valid']:>7} {_pct(result['recall_fired_rate']):>7} "
               f"{_pct(result['correct_rate_all']):>13} {_pct(result['correct_rate_fired']):>15} "
               f"{_pct(result['surfaced_any_rate']):>10} "
               f"{_pct(result['surfaced_via_recency_rate']):>14} "
               f"{_fmt_money(result['raw_cost']['cost_usd']):>10} "
               f"{_fmt_money(efficiency.get('cost_usd')):>11} "
               f"{_fmt_num(efficiency.get('turns')):>15} "
               f"{_fmt_num(efficiency.get('dur_ms'), 0):>12}")
        print(row)


if __name__ == "__main__":
    main()
