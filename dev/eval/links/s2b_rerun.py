"""
s2b_rerun.py — S2b harness correction: re-run T2ppr/T2blend/T3 cells only.

The S2 rule-(c) kills on T2ppr/T2blend/T3 measured a harness artifact: re-ranking
the MIXED note+chunk list by raw cosine reintroduced the note-vs-chunk drowning
that engram's native ranking (the note-floor) already solved. The plan's T3
definition is "re-rank buried/below-floor NOTES upward" — notes, not chunks.

traversal.py is now chunk-pinned (note-only re-ranking). This script re-runs
EXACTLY the affected cells — {L2,L3,L4,L7} × {T2ppr,T2blend,T3} (L4 with its
{0.5,0.6,0.8} sweep) — with the SAME pre-registered prune rules, updates
s2_results.json (marking corrected rows), and prints the updated table.

All other cells' results are untouched.
"""
from __future__ import annotations

import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)

from probe import (
    OUT_PATH,
    REPLAYS_PATH,
    MISSES_P1_PATH,
    BRIDGES_P2_PATH,
    SUPERSESSION_P3_PATH,
    load_json,
    load_fabrics,
    build_cases,
    run_cell,
    run_l4_cell,
    prune_verdict,
    print_table,
)

AFFECTED_TRAVERSALS = ("T2ppr", "T2blend", "T3")
AFFECTED_FABRICS = ("L2", "L3", "L4", "L7")
S2B_MARK = "S2b corrected — chunk-pinned"


def main() -> None:
    print("=== S2b — harness correction re-run (chunk-pinned T2/T3) ===\n", flush=True)

    # Load prior results (all untouched cells keep these numbers)
    prior = load_json(OUT_PATH)

    replays         = load_json(REPLAYS_PATH)
    misses_p1       = load_json(MISSES_P1_PATH)
    bridges_p2      = load_json(BRIDGES_P2_PATH)
    supersession_p3 = load_json(SUPERSESSION_P3_PATH)
    fabrics         = load_fabrics()

    print("Rebuilding cases (P3 baselines re-run via engram query)…", flush=True)
    cases, zero_miss_cases = build_cases(
        replays, misses_p1, bridges_p2, supersession_p3
    )

    # Sanity: P3 baselines must match the S2 run's lengths (no vault drift)
    prior_control = prior["cells"]["L1×T1 [CONTROL]"]["case_results"]
    prior_p3_lens = {
        r["case_id"]: r["baseline_len"]
        for r in prior_control
        if r["case_id"].startswith("P3")
    }
    for case in cases:
        if case["kind"] == "P3":
            prior_len = prior_p3_lens.get(case["case_id"])
            if prior_len is not None and prior_len != len(case["baseline"]):
                print(
                    f"  WARN: {case['case_id']} baseline drifted "
                    f"({prior_len} → {len(case['baseline'])} items) — "
                    f"vault changed since S2; comparability reduced.",
                    flush=True,
                )

    l5 = fabrics["l5"]
    l6 = fabrics["l6"]

    # -----------------------------------------------------------------------
    # Re-run the 12 affected cells
    # -----------------------------------------------------------------------
    corrected: dict[str, dict] = {}

    print("\nRe-running affected cells (chunk-pinned)…\n", flush=True)
    for trav_id in AFFECTED_TRAVERSALS:
        for fab in AFFECTED_FABRICS:
            cell_id = f"{fab}×{trav_id}"
            if fab == "L4":
                m = run_l4_cell(trav_id, fabrics["l4"], cases, zero_miss_cases, l5, l6)
                tau_note = f"  (best τ={m.get('best_threshold', '?')})"
            else:
                m = run_cell(
                    cell_id, trav_id, fabrics[fab.lower()],
                    cases, zero_miss_cases, l5, l6,
                )
                tau_note = ""
            m["traversal_id"] = trav_id
            m["s2b_note"] = S2B_MARK
            corrected[cell_id] = m
            print(
                f"  {cell_id:<32} r@10={m['r10']:5.1f}%  "
                f"regress={m['regression_count']}  "
                f"Δpld={m['payload_delta_pct']:.1f}%{tau_note}",
                flush=True,
            )

    # -----------------------------------------------------------------------
    # Merge: prior metrics for untouched cells + corrected metrics
    # -----------------------------------------------------------------------
    all_metrics: dict[str, dict] = {}
    for cell_id, cell_data in prior["cells"].items():
        if cell_id in corrected:
            all_metrics[cell_id] = corrected[cell_id]
        else:
            m = dict(cell_data["metrics"])
            m["case_results"] = cell_data.get("case_results", [])
            if cell_data.get("threshold_sweep"):
                m["threshold_sweep"] = cell_data["threshold_sweep"]
            all_metrics[cell_id] = m

    control_r10 = prior["control_r10"]

    # -----------------------------------------------------------------------
    # Verdicts: recompute for corrected cells with the SAME pre-registered
    # rules; untouched cells keep their S2 verdicts (rule-d cross-references
    # use the merged metrics map, so corrected r@10 values participate).
    # -----------------------------------------------------------------------
    verdicts: dict[str, str] = dict(prior["verdicts"])
    flips: list[str] = []
    stands: list[str] = []

    for cell_id in corrected:
        old_verdict = prior["verdicts"].get(cell_id, "?")
        new_verdict = prune_verdict(
            cell_id, corrected[cell_id], control_r10,
            corrected[cell_id]["traversal_id"], all_metrics,
        )
        verdicts[cell_id] = new_verdict
        old_kind = old_verdict.split("(")[0]
        new_kind = new_verdict.split("(")[0]
        if old_kind != new_kind:
            flips.append(f"  {cell_id}: {old_verdict}  →  {new_verdict}")
        else:
            stands.append(f"  {cell_id}: {new_verdict}  (was: {old_verdict})")

    # -----------------------------------------------------------------------
    # Print updated table (same columns)
    # -----------------------------------------------------------------------
    print_table(all_metrics, verdicts)

    print("\n--- S2b verdict changes (kills that FLIP) ---")
    print("\n".join(flips) if flips else "  (none)")
    print("\n--- S2b verdicts that STAND (possibly different rule) ---")
    print("\n".join(stands) if stands else "  (none)")

    survivors = [c for c, v in verdicts.items() if v == "SURVIVOR"]
    print(f"\n--- Updated survivor list ({len(survivors)}) ---")
    for s in survivors:
        mark = "  [S2b corrected]" if s in corrected else ""
        print(f"  {s}{mark}")

    # -----------------------------------------------------------------------
    # Update s2_results.json in place — only the corrected rows change
    # -----------------------------------------------------------------------
    for cell_id, m in corrected.items():
        prior["cells"][cell_id] = {
            "metrics": {
                k: v for k, v in m.items()
                if k not in ("case_results", "threshold_sweep")
            },
            "verdict": verdicts[cell_id],
            "case_results": m.get("case_results", []),
            "threshold_sweep": m.get("threshold_sweep"),
        }
    prior["verdicts"] = verdicts
    prior["s2b_correction"] = {
        "date": "2026-07-02",
        "note": (
            "T2ppr/T2blend/T3 cells re-run with chunk-pinned (note-only) "
            "re-ranking. Harness-spec fix per the plan's T3 definition "
            "('re-rank buried/below-floor NOTES upward'); prune rules unchanged."
        ),
        "affected_cells": sorted(corrected.keys()),
    }

    with open(OUT_PATH, "w") as fh:
        json.dump(prior, fh, indent=2)
    print(f"\nUpdated {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
