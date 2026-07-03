"""sweep_s2.py — floor × K sweep for vocab nomination (Slice 2 tuning procedure).

Loads note and term body vectors from a bootstrapped vault copy, then for each
(floor, K-mode) config computes mechanical assignments and runs the TAG nomination
probe on the P1+P2 miss population.

Usage:
    python dev/eval/links/sweep_s2.py --vault /path/to/bootstrapped/vault/copy

Sweep: floor ∈ {0.25, 0.30, 0.35, 0.40} × K ∈ {K2+rider, K3}
Selection rule: max recovery@5 subject to median pool ≤ 40; tie → higher floor.
Baselines (from probe.py L6×TAG): recovery@5=54.2%, median_pool≈30.
"""
from __future__ import annotations

import argparse
import json
import math
import os
import statistics
from collections import defaultdict
from typing import Any

HERE = os.path.dirname(os.path.abspath(__file__))
MISSES_P1_PATH = os.path.join(HERE, "misses_p1.json")
BRIDGES_P2_PATH = os.path.join(HERE, "bridges_p2.json")
REPLAYS_PATH = os.path.join(HERE, "replays.json")

# ---------------------------------------------------------------------------
# Constants (mirror vocab.go)
# ---------------------------------------------------------------------------

TOP_VOCAB_TERM_COUNT = 2         # top-K before close-3rd rider
CLOSE_THIRD_MARGIN = 0.02        # rider fires if 3rd is within this of 2nd

# ---------------------------------------------------------------------------
# I/O helpers
# ---------------------------------------------------------------------------


def load_json(path: str) -> Any:
    with open(path) as fh:
        return json.load(fh)


def cosine(a: list[float], b: list[float]) -> float:
    """Cosine similarity between two equal-length vectors."""
    dot = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x * x for x in a))
    norm_b = math.sqrt(sum(x * x for x in b))
    if norm_a == 0.0 or norm_b == 0.0:
        return 0.0
    return dot / (norm_a * norm_b)


# ---------------------------------------------------------------------------
# Load vectors from vault copy
# ---------------------------------------------------------------------------


def load_term_vectors(vault: str) -> dict[str, list[float]]:
    """Return {term_name: body_vector} for all vocab.<term>.vec.json files."""
    result: dict[str, list[float]] = {}
    for fname in os.listdir(vault):
        if not (fname.startswith("vocab.") and fname.endswith(".vec.json")):
            continue
        term = fname[len("vocab."):-len(".vec.json")]
        if term == "index":
            continue
        path = os.path.join(vault, fname)
        with open(path) as fh:
            s = json.load(fh)
        bv = s.get("body_vector", [])
        if bv:
            result[term] = bv
    return result


def load_note_vectors(vault: str) -> dict[str, list[float]]:
    """Return {basename_no_md: body_vector} for all non-vocab note sidecars."""
    result: dict[str, list[float]] = {}
    for fname in os.listdir(vault):
        if not fname.endswith(".vec.json"):
            continue
        if fname.startswith("vocab."):
            continue
        # Derive note basename without .md
        note_name = fname[:-len(".vec.json")]
        path = os.path.join(vault, fname)
        with open(path) as fh:
            s = json.load(fh)
        bv = s.get("body_vector", [])
        if bv:
            result[note_name] = bv
    return result


# ---------------------------------------------------------------------------
# Assignment computation
# ---------------------------------------------------------------------------


def assign_terms(
    note_vec: list[float],
    term_vecs: dict[str, list[float]],
    floor: float,
    k_mode: str,
) -> list[str]:
    """Compute vocab assignment for one note under the given config.

    k_mode: "K2+rider" = top-2 + close-3rd rider (mirrors AssignVocabTerms)
            "K3"       = plain top-3, no rider
    """
    scores = [(term, cosine(note_vec, tvec)) for term, tvec in term_vecs.items()]
    candidates = sorted(
        [(term, sc) for term, sc in scores if sc >= floor],
        key=lambda x: x[1],
        reverse=True,
    )
    if not candidates:
        return []

    if k_mode == "K2+rider":
        selected = candidates[:TOP_VOCAB_TERM_COUNT]
        # Close-3rd rider
        if (
            len(candidates) > TOP_VOCAB_TERM_COUNT
            and len(selected) >= TOP_VOCAB_TERM_COUNT
        ):
            second_score = selected[-1][1]
            third_score = candidates[TOP_VOCAB_TERM_COUNT][1]
            if second_score - third_score <= CLOSE_THIRD_MARGIN:
                selected.append(candidates[TOP_VOCAB_TERM_COUNT])
    else:  # K3: plain top-3
        selected = candidates[:3]

    return [term for term, _ in selected]


def build_assignments(
    note_vecs: dict[str, list[float]],
    term_vecs: dict[str, list[float]],
    floor: float,
    k_mode: str,
) -> dict:
    """Build l6-equivalent assignments dict for the given (floor, k_mode) config."""
    assignments = []
    for note_name, nvec in note_vecs.items():
        tags = assign_terms(nvec, term_vecs, floor, k_mode)
        if tags:
            assignments.append({"note": note_name, "tags": tags})
    return {
        "vocab": sorted(term_vecs.keys()),
        "assignments": assignments,
    }


# ---------------------------------------------------------------------------
# Nomination probe (mirrors tag_filter_candidates from traversal.py)
# ---------------------------------------------------------------------------


def tag_filter_candidates(
    ranked: list[dict],
    l6_sweep: dict,
    top_m: int = 3,
) -> tuple[list[str], int]:
    """Return (candidate_basenames, pool_size) for the nomination probe."""
    assignments = l6_sweep.get("assignments", [])

    note_tags: dict[str, set[str]] = {}
    tag_notes: dict[str, set[str]] = defaultdict(set)
    for entry in assignments:
        note = _strip_md(entry["note"])
        tags = set(entry.get("tags", []))
        note_tags[note] = tags
        for tag in tags:
            tag_notes[tag].add(note)

    top_tags: set[str] = set()
    for item in ranked[:top_m]:
        node = _strip_md(item["basename"])
        top_tags |= note_tags.get(node, set())

    delivered: set[str] = {_strip_md(r["basename"]) for r in ranked}
    candidates: set[str] = set()
    for tag in top_tags:
        candidates |= tag_notes[tag]
    candidates -= delivered

    return sorted(_add_md(c) for c in candidates), len(candidates)


def _strip_md(s: str) -> str:
    return s[:-3] if s.endswith(".md") else s


def _add_md(s: str) -> str:
    return s if s.endswith(".md") else s + ".md"


# ---------------------------------------------------------------------------
# Cases (mirrors build_cases from probe.py, P1+P2 only — no live engram calls)
# ---------------------------------------------------------------------------


def build_cases(
    replays: list[dict],
    misses_p1: list[dict],
    bridges_p2: list[dict],
) -> list[dict]:
    """Build P1+P2 miss cases. Skips P3 (needs live engram query)."""
    replay_index = {
        (r["query_id"], r["n"]): r["ranked_notes"] for r in replays
    }

    cases: list[dict] = []

    for miss in misses_p1:
        qid, n = miss["query_id"], miss["n"]
        baseline = replay_index.get((qid, n), [])
        cases.append({
            "case_id": f"P1-{qid}-n{n}",
            "kind": "P1",
            "baseline": baseline,
            "needed_note": miss["missed_note"],
        })

    for bridge in bridges_p2:
        top10 = bridge["delivered_top10"]
        total = max(len(top10), 1)
        baseline = [
            {"basename": bn, "score": 1.0 - i / total, "kind": "fact", "rank": i + 1}
            for i, bn in enumerate(top10)
        ]
        cases.append({
            "case_id": f"P2-{bridge['case_id']}",
            "kind": "P2",
            "baseline": baseline,
            "needed_note": bridge["needed_note"],
        })

    return cases


# ---------------------------------------------------------------------------
# Metrics
# ---------------------------------------------------------------------------


def run_probe(cases: list[dict], l6_sweep: dict, top_m: int = 3) -> dict:
    """Run the TAG nomination probe; return recovery and pool metrics."""
    r5 = r10 = r20 = 0
    pool_sizes: list[int] = []

    for case in cases:
        baseline = case["baseline"]
        needed_note = case["needed_note"]

        pool, pool_size = tag_filter_candidates(baseline, l6_sweep, top_m=top_m)
        target = _strip_md(needed_note)
        in_pool = any(_strip_md(b) == target for b in pool)
        pool_sizes.append(pool_size)

        if in_pool:
            r5 += 1
            r10 += 1
            r20 += 1

    total = len(cases)
    return {
        "n": total,
        "r5": round(r5 / total * 100, 1) if total else 0.0,
        "r10": round(r10 / total * 100, 1) if total else 0.0,
        "r20": round(r20 / total * 100, 1) if total else 0.0,
        "median_pool": statistics.median(pool_sizes) if pool_sizes else 0,
    }


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main() -> None:
    parser = argparse.ArgumentParser(description="floor×K sweep for vocab nomination.")
    parser.add_argument("--vault", required=True, help="Path to bootstrapped vault copy.")
    args = parser.parse_args()

    vault = args.vault
    print(f"Loading vectors from {vault}…", flush=True)
    term_vecs = load_term_vectors(vault)
    note_vecs = load_note_vectors(vault)
    print(f"  {len(term_vecs)} term vectors, {len(note_vecs)} note vectors", flush=True)

    replays = load_json(REPLAYS_PATH)
    misses_p1 = load_json(MISSES_P1_PATH)
    bridges_p2 = load_json(BRIDGES_P2_PATH)

    cases = build_cases(replays, misses_p1, bridges_p2)
    print(f"  {len(cases)} miss cases (P1+P2)\n", flush=True)

    FLOORS = [0.25, 0.30, 0.35, 0.40]
    K_MODES = ["K2+rider", "K3"]

    results: dict[str, dict] = {}
    rows: list[tuple] = []

    for floor in FLOORS:
        for k_mode in K_MODES:
            config_id = f"floor={floor:.2f}×{k_mode}"
            l6_sweep = build_assignments(note_vecs, term_vecs, floor, k_mode)
            n_tagged = sum(1 for a in l6_sweep["assignments"] if a["tags"])
            m = run_probe(cases, l6_sweep)
            results[config_id] = {"floor": floor, "k_mode": k_mode, "n_tagged": n_tagged, **m}
            rows.append((config_id, n_tagged, m["r5"], m["median_pool"]))
            print(
                f"  {config_id:<30} n_tagged={n_tagged:>3}  "
                f"r@5={m['r5']:>5.1f}%  median_pool={m['median_pool']:>5.1f}",
                flush=True,
            )

    # -----------------------------------------------------------------------
    # Table
    # -----------------------------------------------------------------------
    print(f"\n{'='*80}")
    print("SWEEP TABLE — floor × K-mode (TAG nomination, P1+P2 miss population)")
    print(f"Baseline (L6×TAG, LLM-labeled): r@5=54.2%, median_pool≈30")
    print(f"Selection rule: max r@5 subject to median_pool ≤ 40; tie → higher floor")
    print(f"{'='*80}")
    print(f"{'Config':<30}  {'n_tagged':>8}  {'r@5%':>6}  {'r@10%':>6}  {'r@20%':>6}  {'med_pool':>9}")
    print("-" * 80)
    for config_id, m in results.items():
        print(
            f"{config_id:<30}  {m['n_tagged']:>8}  {m['r5']:>6.1f}  "
            f"{m['r10']:>6.1f}  {m['r20']:>6.1f}  {m['median_pool']:>9.1f}"
        )
    print("=" * 80)

    # -----------------------------------------------------------------------
    # Select best config
    # -----------------------------------------------------------------------
    eligible = {
        cid: m for cid, m in results.items()
        if m["median_pool"] <= 40
    }
    if eligible:
        best_id = max(eligible, key=lambda cid: (eligible[cid]["r5"], eligible[cid]["floor"]))
        best = eligible[best_id]
        print(f"\n  SELECTED: {best_id}")
        print(f"    floor={best['floor']}, k_mode={best['k_mode']}")
        print(f"    r@5={best['r5']}%, median_pool={best['median_pool']}")
    else:
        print("\n  WARNING: no config meets median_pool ≤ 40 constraint")
        best_id = max(results, key=lambda cid: results[cid]["r5"])
        print(f"  Best unconstrained: {best_id} r@5={results[best_id]['r5']}%")

    # -----------------------------------------------------------------------
    # Write results
    # -----------------------------------------------------------------------
    out_path = os.path.join(HERE, "sweep_s2_results.json")
    with open(out_path, "w") as fh:
        json.dump({"configs": results, "selected": best_id}, fh, indent=2)
    print(f"\nResults written to {out_path}")


if __name__ == "__main__":
    main()
