"""
probe.py — S2 matrix runner for the link-value PoC probe.

Runs 19 scored cells (L×T combinations), computes recovery@5/@10/@20 on
the P1/P2/P3 miss population, checks collateral regression on the zero-miss
set, applies pre-registered prune rules, and writes s2_results.json.

Run from the repo root or directly:
    python dev/eval/links/probe.py
"""
from __future__ import annotations

import json
import os
import re
import statistics
import subprocess
import sys
from collections import defaultdict
from typing import Any

# ---------------------------------------------------------------------------
# Paths
# ---------------------------------------------------------------------------
HERE = os.path.dirname(os.path.abspath(__file__))
FABRICS_DIR = os.path.join(HERE, "fabrics")

REPLAYS_PATH = os.path.join(HERE, "replays.json")
MISSES_P1_PATH = os.path.join(HERE, "misses_p1.json")
BRIDGES_P2_PATH = os.path.join(HERE, "bridges_p2.json")
SUPERSESSION_P3_PATH = os.path.join(HERE, "supersession_p3.json")
OUT_PATH = os.path.join(HERE, "s2_results.json")

# ---------------------------------------------------------------------------
# Import traversal functions
# ---------------------------------------------------------------------------
sys.path.insert(0, HERE)
from traversal import (
    _strip_md,
    _add_md,
    expand_one_hop,
    ppr_rank,
    ppr_blend,
    rank_boost,
    supersession_ride_along,
    tag_filter_candidates,
)

# ---------------------------------------------------------------------------
# I/O helpers
# ---------------------------------------------------------------------------


def load_json(path: str) -> Any:
    with open(path) as fh:
        return json.load(fh)


def load_fabrics() -> dict[str, Any]:
    result = {}
    for name in ["l1", "l2", "l3", "l4", "l5", "l6", "l7"]:
        result[name] = load_json(os.path.join(FABRICS_DIR, f"{name}.json"))
    return result


# ---------------------------------------------------------------------------
# engram query for P3 baselines (cases without a stored delivered list)
# ---------------------------------------------------------------------------

def run_engram_query(phrases: list[str]) -> list[dict]:
    """Run engram query --lazy-chunks with all given phrases; return ranked note items."""
    cmd = ["engram", "query", "--lazy-chunks"]
    for phrase in phrases:
        cmd += ["--phrase", phrase]

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        print(
            f"  WARN: engram query failed (exit {result.returncode}): "
            f"{result.stderr.strip()[:120]}",
            file=sys.stderr,
        )
        return []

    items: list[dict] = []
    rank = 0
    current_path: str | None = None
    current_score: float | None = None
    current_kind: str | None = None
    in_items = False

    for line in result.stdout.splitlines():
        if line == "items:":
            in_items = True
            continue
        if in_items and re.match(r"^[a-z_]+:", line) and not line.startswith("  "):
            if "clusters:" in line or "metadata:" in line:
                in_items = False
                if current_path is not None:
                    rank += 1
                    if current_path.endswith(".md"):
                        items.append({
                            "basename": current_path,
                            "score": current_score,
                            "kind": current_kind,
                            "rank": rank,
                        })
                    current_path = current_score = current_kind = None
            continue

        if not in_items:
            continue

        path_m = re.match(r"^  - path:\s*(.+?)\s*$", line)
        if path_m:
            if current_path is not None:
                rank += 1
                if current_path.endswith(".md"):
                    items.append({
                        "basename": current_path,
                        "score": current_score,
                        "kind": current_kind,
                        "rank": rank,
                    })
            raw = path_m.group(1).strip('"').split("#")[0]
            current_path = os.path.basename(raw)
            current_score = None
            current_kind = None
            continue

        if current_path is not None:
            score_m = re.match(r"^\s+score:\s*([\d.]+)\s*$", line)
            if score_m:
                current_score = float(score_m.group(1))
                continue
            kind_m = re.match(r"^\s+kind:\s*(\S+)\s*$", line)
            if kind_m:
                current_kind = kind_m.group(1)

    if in_items and current_path is not None:
        rank += 1
        if current_path.endswith(".md"):
            items.append({
                "basename": current_path,
                "score": current_score,
                "kind": current_kind,
                "rank": rank,
            })

    return items


# ---------------------------------------------------------------------------
# Case building
# ---------------------------------------------------------------------------

def build_cases(
    replays: list[dict],
    misses_p1: list[dict],
    bridges_p2: list[dict],
    supersession_p3: list[dict],
) -> tuple[list[dict], list[dict]]:
    """Build miss cases and zero-miss regression cases.

    Returns (miss_cases, zero_miss_cases).

    Miss case schema:
        {case_id, kind (P1/P2/P3), n (int or None), baseline (list[dict]),
         needed_note (str with .md)}
    """
    replay_index = {
        (r["query_id"], r["n"]): r["ranked_notes"]
        for r in replays
    }

    # --- P1: real-query misses ---
    p1_cases: list[dict] = []
    for miss in misses_p1:
        qid = miss["query_id"]
        n = miss["n"]
        baseline = replay_index.get((qid, n), [])
        p1_cases.append({
            "case_id": f"P1-{qid}-n{n}",
            "kind": "P1",
            "n": n,
            "baseline": baseline,
            "needed_note": miss["missed_note"],
        })

    # --- P2: bridge cases (synthetic descending scores by rank) ---
    p2_cases: list[dict] = []
    for bridge in bridges_p2:
        top10 = bridge["delivered_top10"]
        total = max(len(top10), 1)
        baseline = [
            {
                "basename": bn,
                "score": 1.0 - i / total,
                "kind": "fact",
                "rank": i + 1,
            }
            for i, bn in enumerate(top10)
        ]
        p2_cases.append({
            "case_id": f"P2-{bridge['case_id']}",
            "kind": "P2",
            "n": None,
            "baseline": baseline,
            "needed_note": bridge["needed_note"],
            "bridge_note": bridge.get("bridge_note"),
        })

    # --- P3: supersession-miss cases (4 entries with supersession_miss=True) ---
    p3_cases: list[dict] = []
    for p3 in supersession_p3:
        if not p3.get("supersession_miss"):
            continue
        print(f"  Running engram query for {p3['pair_id']} ({len(p3['phrases'])} phrases)…",
              flush=True)
        baseline = run_engram_query(p3["phrases"])
        p3_cases.append({
            "case_id": f"P3-{p3['pair_id']}",
            "kind": "P3",
            "n": None,
            "baseline": baseline,
            "needed_note": p3["new_note"],
            "old_note": p3["old_note"],
        })

    all_miss_cases = p1_cases + p2_cases + p3_cases

    # --- Zero-miss regression set ---
    miss_pairs = {(m["query_id"], m["n"]) for m in misses_p1}
    zero_miss_cases = [
        {
            "case_id": f"REG-{r['query_id']}-n{r['n']}",
            "n": r["n"],
            "baseline": r["ranked_notes"],
        }
        for r in replays
        if (r["query_id"], r["n"]) not in miss_pairs
    ]

    print(
        f"Cases: {len(p1_cases)} P1, {len(p2_cases)} P2, {len(p3_cases)} P3 "
        f"= {len(all_miss_cases)} total  |  "
        f"Zero-miss regression set: {len(zero_miss_cases)}",
        flush=True,
    )
    return all_miss_cases, zero_miss_cases


# ---------------------------------------------------------------------------
# Traversal application
# ---------------------------------------------------------------------------

def apply_traversal(
    traversal_id: str,
    case_kind: str,
    case_n: int | None,
    baseline: list[dict],
    fabric: list[dict],
    l5_fabric: list[dict],
) -> list[dict]:
    """Apply a traversal to a baseline; return the (possibly expanded) result."""
    if traversal_id == "T1":
        return expand_one_hop(baseline, fabric, top_m=5)

    elif traversal_id == "T2ppr":
        result, _ = ppr_rank(baseline, fabric)
        return result

    elif traversal_id == "T2blend":
        result, _ = ppr_blend(baseline, fabric)
        return result

    elif traversal_id == "T3":
        result, _ = rank_boost(baseline, fabric, w=0.1)
        return result

    elif traversal_id == "T5":
        return supersession_ride_along(baseline, l5_fabric)

    elif traversal_id == "T6":
        # Apply only to n=3 P1 cases; baseline unchanged for all others
        if case_kind == "P1" and case_n == 3:
            return expand_one_hop(baseline, fabric, top_m=5)
        return list(baseline)

    elif traversal_id == "TAG":
        # TAG is a candidate-pool traversal, handled separately in run_cell
        return list(baseline)

    return list(baseline)


# ---------------------------------------------------------------------------
# Metrics
# ---------------------------------------------------------------------------

def find_rank(result: list[dict], needed_note: str) -> int | None:
    """Return 1-based rank of needed_note in result, or None if absent."""
    target = _strip_md(needed_note)
    for i, item in enumerate(result):
        if _strip_md(item["basename"]) == target:
            return i + 1
    return None


def compute_metrics(result_rows: list[dict]) -> dict:
    """Aggregate recovery@5/@10/@20, median_rank, payload_delta_pct."""
    total = len(result_rows)
    if total == 0:
        return {
            "applicable": 0, "r5": 0.0, "r10": 0.0, "r20": 0.0,
            "median_rank": None, "payload_delta_pct": 0.0,
        }

    r5 = r10 = r20 = 0
    ranks: list[float] = []
    deltas: list[float] = []

    for row in result_rows:
        rank = row.get("rank")
        if rank is not None:
            if rank <= 5:
                r5 += 1
            if rank <= 10:
                r10 += 1
            if rank <= 20:
                r20 += 1
            ranks.append(rank)

        baseline_len = row.get("baseline_len", 0)
        result_len = row.get("result_len", baseline_len)
        if baseline_len > 0:
            deltas.append((result_len - baseline_len) / baseline_len * 100.0)

    return {
        "applicable": total,
        "r5": round(r5 / total * 100, 1),
        "r10": round(r10 / total * 100, 1),
        "r20": round(r20 / total * 100, 1),
        "median_rank": statistics.median(ranks) if ranks else None,
        "payload_delta_pct": round(statistics.mean(deltas), 1) if deltas else 0.0,
    }


# ---------------------------------------------------------------------------
# Density gate
# ---------------------------------------------------------------------------

def density_metrics(fabric: list[dict], cases: list[dict], fabric_id: str) -> dict:
    """Fabric connectivity stats for the T2 density gate."""
    nodes: set[str] = set()
    for edge in fabric:
        src = _strip_md(edge.get("src") or edge.get("old") or "")
        dst = _strip_md(edge.get("dst") or edge.get("new") or "")
        if src:
            nodes.add(src)
        if dst:
            nodes.add(dst)

    total_vault = 135
    in_graph = len(nodes)
    isolated_pct = round((total_vault - in_graph) / total_vault * 100.0, 1)

    needed_in_graph = sum(
        1 for c in cases
        if _strip_md(c["needed_note"]) in nodes
    )
    return {
        "fabric": fabric_id,
        "nodes_in_graph": in_graph,
        "edges": len(fabric),
        "isolated_pct": isolated_pct,
        "needed_in_graph": needed_in_graph,
        "n_cases": len(cases),
    }


# ---------------------------------------------------------------------------
# Collateral regression check
# ---------------------------------------------------------------------------

def check_regression(result: list[dict], baseline: list[dict], top_k: int = 5) -> bool:
    """True if any baseline top-K fact/feedback note falls out of result top-K."""
    baseline_top = {
        _strip_md(item["basename"])
        for item in baseline[:top_k]
        if item.get("kind") in ("fact", "feedback")
    }
    if not baseline_top:
        return False
    result_top = {_strip_md(item["basename"]) for item in result[:top_k]}
    return bool(baseline_top - result_top)


# ---------------------------------------------------------------------------
# Cell runner
# ---------------------------------------------------------------------------

def run_cell(
    cell_id: str,
    traversal_id: str,
    fabric: list[dict],
    cases: list[dict],
    zero_miss_cases: list[dict],
    l5_fabric: list[dict],
    l6: dict,
) -> dict:
    """Run one L×T cell; return metrics dict."""
    result_rows: list[dict] = []

    for case in cases:
        baseline = case["baseline"]
        needed_note = case["needed_note"]
        case_kind = case["kind"]
        case_n = case.get("n")

        if traversal_id == "TAG":
            pool, pool_size = tag_filter_candidates(baseline, l6, top_m=3)
            target = _strip_md(needed_note)
            in_pool = any(_strip_md(b) == target for b in pool)
            rank = 1 if in_pool else None
            t4_ride_along = in_pool
            result_len = pool_size
        else:
            result = apply_traversal(
                traversal_id, case_kind, case_n, baseline, fabric, l5_fabric
            )
            rank = find_rank(result, needed_note)
            # T4 ride-along: was needed_note anywhere in the expanded/result set?
            target = _strip_md(needed_note)
            t4_ride_along = any(_strip_md(r["basename"]) == target for r in result)
            result_len = len(result)

        result_rows.append({
            "case_id": case["case_id"],
            "rank": rank,
            "baseline_len": len(baseline),
            "result_len": result_len,
            "t4_ride_along": t4_ride_along,
        })

    metrics = compute_metrics(result_rows)

    # Collateral regression on zero-miss set
    regression_count = 0
    if traversal_id != "TAG":
        for reg_case in zero_miss_cases:
            baseline = reg_case["baseline"]
            result = apply_traversal(
                traversal_id,
                case_kind="REG",
                case_n=reg_case.get("n"),
                baseline=baseline,
                fabric=fabric,
                l5_fabric=l5_fabric,
            )
            if check_regression(result, baseline, top_k=5):
                regression_count += 1

    metrics["regression_count"] = regression_count
    metrics["t4_ride_along_count"] = sum(
        1 for r in result_rows if r.get("t4_ride_along")
    )
    metrics["case_results"] = result_rows
    return metrics


# ---------------------------------------------------------------------------
# L4 threshold sweep
# ---------------------------------------------------------------------------

def run_l4_cell(
    traversal_id: str,
    l4_full: list[dict],
    cases: list[dict],
    zero_miss_cases: list[dict],
    l5_fabric: list[dict],
    l6: dict,
) -> dict:
    """Run an L4 cell sweeping cosine thresholds {0.5, 0.6, 0.8}; return best + all."""
    sub: dict[float, dict] = {}
    for tau in (0.5, 0.6, 0.8):
        fabric = [e for e in l4_full if e.get("cosine", 0.0) >= tau]
        cell_metrics = run_cell(
            f"L4(τ={tau})×{traversal_id}", traversal_id,
            fabric, cases, zero_miss_cases, l5_fabric, l6,
        )
        cell_metrics["threshold"] = tau
        cell_metrics["edges_at_threshold"] = len(fabric)
        sub[tau] = cell_metrics

    # Best = highest r10; tie-break lowest payload_delta_pct
    best_tau = max(sub, key=lambda t: (sub[t]["r10"], -sub[t]["payload_delta_pct"]))
    best = dict(sub[best_tau])
    best["best_threshold"] = best_tau
    best["threshold_sweep"] = {str(t): {k: v for k, v in m.items() if k != "case_results"}
                                for t, m in sub.items()}
    return best


# ---------------------------------------------------------------------------
# Pre-registered prune rules
# ---------------------------------------------------------------------------

# Recall-changes-required burden level (higher = heavier)
BURDEN = {
    "T1":     1,   # light: expansion pass post-union
    "T2ppr":  3,   # heaviest: graph-aware scoring stage
    "T2blend": 3,  # same burden as T2ppr
    "T3":     2,   # medium: score-adjust pass
    "T5":     1,   # small: typed lookup
    "T6":     1,   # very light: conditional on phrase-count
    "TAG":    1,   # lightest: candidate nomination only
}


def prune_verdict(
    cell_id: str,
    metrics: dict,
    control_r10: float,
    traversal_id: str,
    all_metrics: dict[str, dict],
) -> str:
    """Apply pre-registered prune rules; return CONTROL / SURVIVOR / KILLED(rule_X)."""
    if "CONTROL" in cell_id:
        return "CONTROL"

    r10 = metrics["r10"]

    # (a) recovery@10 ≤ control's
    if r10 <= control_r10:
        return f"KILLED(a: r@10={r10} ≤ control={control_r10})"

    # (b) payload growth > +20% without recovery gain (r@10 ≤ 0)
    if metrics["payload_delta_pct"] > 20.0 and r10 <= 0.0:
        return (
            f"KILLED(b: payload_Δ={metrics['payload_delta_pct']:.1f}%>20% "
            f"with r@10={r10})"
        )

    # (c) ≥1 collateral regression
    if metrics.get("regression_count", 0) >= 1:
        rcount = metrics["regression_count"]
        return f"KILLED(c: {rcount} collateral regression(s))"

    # (d) burden rule: T2 burden with no gain over a lighter variant of the same fabric
    if BURDEN.get(traversal_id, 0) >= 3:
        fabric_prefix = cell_id.rsplit("×", 1)[0]
        for cid, cm in all_metrics.items():
            other_trav = cm.get("traversal_id", "")
            if (
                cid != cell_id
                and cid.startswith(fabric_prefix)
                and BURDEN.get(other_trav, 0) < 3
                and cm.get("r10", 0.0) >= r10
            ):
                return (
                    f"KILLED(d: lighter {cid}[r@10={cm.get('r10')}] ≥ "
                    f"this[r@10={r10}] with less burden)"
                )

    return "SURVIVOR"


# ---------------------------------------------------------------------------
# Table printing
# ---------------------------------------------------------------------------

def print_table(
    all_metrics: dict[str, dict],
    verdicts: dict[str, str],
) -> None:
    col_w = 30
    print(
        f"\n{'='*108}\n"
        f"CELL TABLE — S2 PoC Probe Matrix  (date: 2026-07-02)\n"
        f"{'='*108}"
    )
    hdr = (
        f"{'Cell':<{col_w}} {'n':>5} {'r@5%':>6} {'r@10%':>6} {'r@20%':>6} "
        f"{'medRk':>7} {'Δpld%':>7} {'regrs':>6}  verdict"
    )
    print(hdr)
    print("-" * 108)
    for cell_id, m in all_metrics.items():
        med = f"{m['median_rank']:.1f}" if m.get("median_rank") is not None else "—"
        verdict = verdicts.get(cell_id, "?")
        print(
            f"{cell_id:<{col_w}} {m['applicable']:>5} {m['r5']:>6.1f} "
            f"{m['r10']:>6.1f} {m['r20']:>6.1f} {med:>7} "
            f"{m['payload_delta_pct']:>7.1f} {m['regression_count']:>6}  {verdict}"
        )
    print("=" * 108)


def print_density_table(density_report: list[dict], total_cases: int) -> None:
    print(
        f"\n{'='*60}\n"
        f"DENSITY GATE (T2 cells, pre-registered)  n_cases={total_cases}\n"
        f"{'='*60}"
    )
    print(f"{'Fabric':<8} {'nodes':>6} {'edges':>6} {'isolated%':>10} {'needed_in_graph':>16}")
    print("-" * 60)
    for dm in density_report:
        print(
            f"{dm['fabric']:<8} {dm['nodes_in_graph']:>6} {dm['edges']:>6} "
            f"{dm['isolated_pct']:>10.1f} {dm['needed_in_graph']:>6}/{dm['n_cases']:<10}"
        )
    print("=" * 60)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main() -> None:
    print("=== S2 PoC Probe Matrix (link-value exploration) ===\n", flush=True)

    replays        = load_json(REPLAYS_PATH)
    misses_p1      = load_json(MISSES_P1_PATH)
    bridges_p2     = load_json(BRIDGES_P2_PATH)
    supersession_p3 = load_json(SUPERSESSION_P3_PATH)
    fabrics        = load_fabrics()

    print("Building cases…", flush=True)
    cases, zero_miss_cases = build_cases(
        replays, misses_p1, bridges_p2, supersession_p3
    )

    # Gate S1
    if len(cases) < 8:
        print(
            f"\n!!! GATE S1 FAILED: only {len(cases)} miss cases (threshold=8). "
            f"STOPPING EARLY — thin miss population is itself the finding."
        )
        return
    print(f"Gate S1 passed: {len(cases)} cases ≥ 8\n", flush=True)

    l5 = fabrics["l5"]
    l6 = fabrics["l6"]

    # Density gate for T2-candidate fabrics
    density_report = []
    for fab_id in ("l1", "l2", "l3", "l4", "l7"):
        dm = density_metrics(
            fabrics[fab_id] if fab_id != "l4"
            else [e for e in fabrics["l4"] if e.get("cosine", 0.0) >= 0.5],
            cases, fab_id,
        )
        density_report.append(dm)

    # -----------------------------------------------------------------------
    # Cell definitions (19 scored cells per the plan)
    # L4 cells run with threshold sweep; T2 cells run both ppr and blend
    # sub-variants (labeled T2ppr / T2blend for clarity; both carry T2 burden).
    # -----------------------------------------------------------------------
    all_metrics: dict[str, dict] = {}

    def _run(cell_id: str, trav_id: str, fabric: list[dict]) -> None:
        m = run_cell(cell_id, trav_id, fabric, cases, zero_miss_cases, l5, l6)
        m["traversal_id"] = trav_id
        all_metrics[cell_id] = m
        print(
            f"  {cell_id:<32} r@10={m['r10']:5.1f}%  "
            f"regress={m['regression_count']}  "
            f"Δpld={m['payload_delta_pct']:.1f}%",
            flush=True,
        )

    def _run_l4(trav_id: str) -> None:
        cell_id = f"L4×{trav_id}"
        m = run_l4_cell(trav_id, fabrics["l4"], cases, zero_miss_cases, l5, l6)
        m["traversal_id"] = trav_id
        all_metrics[cell_id] = m
        tau = m.get("best_threshold", "?")
        print(
            f"  {cell_id:<32} r@10={m['r10']:5.1f}%  "
            f"regress={m['regression_count']}  "
            f"Δpld={m['payload_delta_pct']:.1f}%  (best τ={tau})",
            flush=True,
        )

    print("Running cells…\n", flush=True)

    # Control (must reproduce settled null)
    _run("L1×T1 [CONTROL]", "T1", fabrics["l1"])
    control_r10 = all_metrics["L1×T1 [CONTROL]"]["r10"]

    # Sanity check: control must NOT beat baseline meaningfully
    # On the miss population, L1×T1 (one-hop expansion of a sparse fabric)
    # should recover very few missed notes.  If r@10 > 20%, something is wrong.
    if control_r10 > 20.0:
        print(
            f"\n!!! HARNESS SUSPECT: L1×T1 control r@10={control_r10}% > 20% "
            f"on the miss population. This contradicts the settled null (note 73). "
            f"Check case assembly before trusting any cell results."
        )

    # L2 cells
    for trav_id in ("T1", "T2ppr", "T2blend", "T3", "T6"):
        _run(f"L2×{trav_id}", trav_id, fabrics["l2"])

    # L3 cells
    for trav_id in ("T1", "T2ppr", "T2blend", "T3", "T6"):
        _run(f"L3×{trav_id}", trav_id, fabrics["l3"])

    # L4 cells (threshold sweep)
    for trav_id in ("T1", "T2ppr", "T2blend", "T3", "T6"):
        _run_l4(trav_id)

    # L7 cells
    for trav_id in ("T1", "T2ppr", "T2blend", "T3", "T6"):
        _run(f"L7×{trav_id}", trav_id, fabrics["l7"])

    # L5×T5
    _run("L5×T5", "T5", l5)

    # L6×TAG (tag filter / candidate nomination)
    _run("L6×TAG", "TAG", [])

    # -----------------------------------------------------------------------
    # Prune verdicts
    # -----------------------------------------------------------------------
    verdicts: dict[str, str] = {}
    for cell_id, m in all_metrics.items():
        trav_id = m.get("traversal_id", "")
        verdicts[cell_id] = prune_verdict(
            cell_id, m, control_r10, trav_id, all_metrics
        )

    # -----------------------------------------------------------------------
    # Print tables
    # -----------------------------------------------------------------------
    print_table(all_metrics, verdicts)

    print("\n--- Prune verdicts ---")
    for cell_id, verdict in verdicts.items():
        if verdict != "CONTROL":
            print(f"  {cell_id}: {verdict}")

    print("\n--- T4 ride-along summary ---")
    print("(For each cell: count of cases where needed_note entered the expanded/boosted set)")
    for cell_id, m in all_metrics.items():
        t4 = m.get("t4_ride_along_count", 0)
        recovered = sum(1 for r in m.get("case_results", []) if r.get("rank") is not None)
        print(f"  {cell_id}: t4_nomination={t4}/{len(cases)}, recovered={recovered}/{len(cases)}")

    print_density_table(density_report, len(cases))

    print(
        "\n--- Pre-registered note ---\n"
        "P2 baselines have needed notes ABSENT from L1 edges by construction\n"
        "(bridge_is_linked=false for all 8 P2 cases). Expected L1×T1 r@10=0 on P2 sub-set."
    )

    # -----------------------------------------------------------------------
    # Write s2_results.json
    # -----------------------------------------------------------------------
    out = {
        "run_date": "2026-07-02",
        "n_cases": len(cases),
        "n_zero_miss": len(zero_miss_cases),
        "control_r10": control_r10,
        "density_gate": density_report,
        "verdicts": verdicts,
        "cells": {
            cell_id: {
                "metrics": {
                    k: v
                    for k, v in m.items()
                    if k not in ("case_results", "threshold_sweep")
                },
                "verdict": verdicts.get(cell_id, "?"),
                "case_results": m.get("case_results", []),
                "threshold_sweep": m.get("threshold_sweep"),
            }
            for cell_id, m in all_metrics.items()
        },
    }

    with open(OUT_PATH, "w") as fh:
        json.dump(out, fh, indent=2)
    print(f"\nWrote {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
