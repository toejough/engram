#!/usr/bin/env python3
"""
learn-rate-skill-only task 3.3: compute W2 + the pre-registered secondary
metrics from a re-measure run's moments.jsonl, with the SAME definitions and
bootstrap as the frozen baseline scorecard.

run_audit.py skips its scorecard stage for any output dir other than the
frozen results/ (build_scorecard.py pins SRC/OUT there and is DO-NOT-TOUCH).
So this script imports build_scorecard's own functions (bootstrap_ci,
rate_block, pct -- percentile bootstrap, n=2000, seed=739) and applies the
exact W1/W2/lost-lesson/fixable definitions build_scorecard.main() uses.
Importing build_scorecard does not run it (its main() is __main__-guarded).

Also adds, for the baseline-vs-re-measure comparison only:
- a two-sample bootstrap CI on the W2 difference (re-measure - baseline),
  resampling each side's per-moment flags independently, same n=2000 and
  seed=739 as build_scorecard.bootstrap_ci;
- the fixable-lost-lesson role breakdown (the plan's third baseline row).

Usage: python3 compute_remeasure.py <run_dir>
Writes <run_dir>/scorecard-remeasure.json and prints it.
"""
import collections
import json
import os
import random
import sys
from pathlib import Path

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import build_scorecard as bs

BASELINE_MOMENTS = Path(__file__).resolve().parent / "results" / "moments.jsonl"


def load(path):
    with open(path) as f:
        return [json.loads(line) for line in f if line.strip()]


def w2_flags(recs):
    return [bs.ws(r)["learn_fired"] is True for r in recs if bs.ws(r)["worth_learning_from"] is True]


def diff_ci(flags_a, flags_b, n_boot=2000, alpha=0.05, seed=739):
    """Percentile bootstrap CI (pp) of rate(b) - rate(a), independent resamples."""
    if not flags_a or not flags_b:
        return {"lo_pp": None, "hi_pp": None}
    rng = random.Random(seed)
    stats = []
    for _ in range(n_boot):
        ra = rng.choices(flags_a, k=len(flags_a))
        rb = rng.choices(flags_b, k=len(flags_b))
        stats.append(100.0 * sum(rb) / len(rb) - 100.0 * sum(ra) / len(ra))
    stats.sort()
    lo_idx = min(n_boot - 1, max(0, round(n_boot * (alpha / 2))))
    hi_idx = min(n_boot - 1, max(0, round(n_boot * (1 - alpha / 2)) - 1))
    return {"lo_pp": round(stats[lo_idx], 1), "hi_pp": round(stats[hi_idx], 1)}


def score(recs):
    n = len(recs)
    wl_true = [r for r in recs if bs.ws(r)["worth_learning_from"] is True]
    w1 = bs.rate_block(
        question="Was this moment worth learning from?",
        counting_unit="per-moment", label="DERIVED",
        denom_n=n, denom_def=f"all {n} audited moments",
        num_n=len(wl_true), num_def="writing_side.worth_learning_from == true",
        flags=[bs.ws(r)["worth_learning_from"] is True for r in recs],
        extra={"null_n_data_gap": sum(1 for r in recs if bs.ws(r)["worth_learning_from"] is None)},
    )
    fired = [r for r in wl_true if bs.ws(r)["learn_fired"] is True]
    w2 = bs.rate_block(
        question="Did a learn/write-memory step actually execute?",
        counting_unit="per-moment", label="DERIVED",
        denom_n=len(wl_true), denom_def=f"W1-passing moments: worth_learning_from == true ({len(wl_true)})",
        num_n=len(fired), num_def="writing_side.learn_fired == true",
        flags=[bs.ws(r)["learn_fired"] is True for r in wl_true],
    )
    lost = [r for r in recs if bs.ws(r)["worth_learning_from"] is True and bs.ws(r)["learn_fired"] is False]
    fc = collections.Counter(r.get("failure_category") for r in lost)
    fixable = [r for r in lost if r.get("failure_category") == "fixable"]
    return {
        "n_moments": n,
        "moment_type_counts": dict(collections.Counter(r["moment_type"] for r in recs)),
        "role_counts": dict(collections.Counter(r.get("role") for r in recs)),
        "repo_counts": dict(collections.Counter(r.get("repo") for r in recs)),
        "instrument_versions": dict(collections.Counter(str(r.get("instrument_version")) for r in recs)),
        "W1_worth_learning": w1,
        "W2_learn_fired": w2,
        "lost_lessons": {
            "definition": "worth_learning_from == true AND learn_fired == false",
            "n": len(lost),
            "pct_of_all_moments": bs.pct(len(lost), n),
            "failure_category_counts": {str(k): v for k, v in fc.items()},
            "fixable_n": len(fixable),
            "fixable_pct_of_lost": bs.pct(len(fixable), len(lost)),
            "fixable_ci95": bs.bootstrap_ci([r.get("failure_category") == "fixable" for r in lost]),
            "fixable_role_breakdown": dict(collections.Counter(r.get("role") for r in fixable)),
            "lost_moment_type_breakdown": dict(collections.Counter(r["moment_type"] for r in lost)),
        },
    }


def main(run_dir):
    run_dir = Path(run_dir)
    base = load(BASELINE_MOMENTS)
    rem = load(run_dir / "moments.jsonl")
    out = {
        "method": (
            "build_scorecard.py functions imported (bootstrap_ci/rate_block/pct: percentile "
            "bootstrap n=2000 seed=739, per-moment); definitions identical to build_scorecard.main(). "
            "Baseline recomputed from the frozen results/moments.jsonl as a reproduction check."
        ),
        "baseline_2026_09_02": score(base),
        "remeasure": score(rem),
        "W2_delta_pp": None,
        "W2_delta_ci95_pp": diff_ci(w2_flags(base), w2_flags(rem)),
    }
    out["sensitivity_without_automated_security_review"] = (
        "n/a -- automated security-review sessions are excluded from the corpus by "
        "run_audit.exclusion_reason ('automated_security_review') since 2026-09-28"
    )
    b, r = out["baseline_2026_09_02"]["W2_learn_fired"], out["remeasure"]["W2_learn_fired"]
    if b["rate_pct"] is not None and r["rate_pct"] is not None:
        out["W2_delta_pp"] = round(r["rate_pct"] - b["rate_pct"], 1)
    with open(run_dir / "scorecard-remeasure.json", "w") as f:
        json.dump(out, f, indent=2)
    print(json.dumps(out, indent=2))


if __name__ == "__main__":
    main(sys.argv[1])
