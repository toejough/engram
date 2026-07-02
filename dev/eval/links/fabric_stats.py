"""Per-fabric stats: edge count, nodes touched, isolated notes, degree distribution, top-5 hubs.

Reads all fabrics/l*.json files and prints a labeled table.
"""
import glob
import json
import os
from collections import defaultdict

HERE = os.path.dirname(os.path.abspath(__file__))
FABRICS_DIR = os.path.join(HERE, "fabrics")
VAULT_PATH = os.path.expanduser("~/.local/share/engram/vault/")

FABRIC_DESCRIPTIONS = {
    "l1": "status-quo wikilinks (control; 77 resolved edges)",
    "l2": "LLM link-on-write (body cosine top-8 + JUSTIFY gate + hub drop)",
    "l3": "shared rare tokens (df ≤ 3)",
    "l4": "situation cosine ≥ 0.50 (2.5% of pairs)",
    "l5": "supersession/temporal edges (LLM-typed: updates/narrows/refutes)",
    "l6": "tag/category taxonomy (controlled vocab 15–25 terms)",
    "l7": "provenance / same-session edges",
}


def fabric_stats(edges: list[dict], total_notes: int) -> dict:
    """Compute stats for a fabric's edge list.

    Handles both {src, dst} (L1/L2/L3/L4/L7) and {old, new} (L5) edge shapes.
    """
    degree: dict[str, int] = defaultdict(int)
    for e in edges:
        # L5 uses {old, new}; all others use {src, dst}
        src = e.get("src") or e.get("old", "")
        dst = e.get("dst") or e.get("new", "")
        if src:
            degree[src] += 1
        if dst:
            degree[dst] += 1

    nodes_touched = len(degree)
    isolated = total_notes - nodes_touched
    degrees = sorted(degree.values())

    top5 = sorted(degree.items(), key=lambda kv: kv[1], reverse=True)[:5]

    return {
        "edge_count": len(edges),
        "nodes_touched": nodes_touched,
        "isolated": isolated,
        "degree_min": degrees[0] if degrees else 0,
        "degree_median": degrees[len(degrees) // 2] if degrees else 0,
        "degree_max": degrees[-1] if degrees else 0,
        "top5": top5,
    }


def main() -> None:
    # Count total notes
    total_notes = len(glob.glob(os.path.join(VAULT_PATH, "*.md")))
    print(f"Vault note count: {total_notes}", flush=True)
    print()

    # Load all built fabrics (edge-list fabrics only — l6 has a different structure)
    fabric_files = sorted(glob.glob(os.path.join(FABRICS_DIR, "l*.json")))
    fabrics: dict[str, dict] = {}

    for fpath in fabric_files:
        name = os.path.basename(fpath)[:-5]  # strip .json
        if name == "l6":
            continue  # l6 is tag-based — handled separately below
        with open(fpath) as fh:
            edges = json.load(fh)
        if not isinstance(edges, list):
            continue  # skip malformed or non-edge-list fabrics
        fabrics[name] = {"edges": edges, "stats": fabric_stats(edges, total_notes)}

    # L6 is structured differently — load it separately
    l6_path = os.path.join(FABRICS_DIR, "l6.json")
    l6_data: dict = {}
    if os.path.exists(l6_path):
        with open(l6_path) as fh:
            l6_data = json.load(fh)

    print("=" * 90)
    print(f"{'Fabric':<6} | {'Description':<45} | {'Edges':>6} | {'Nodes':>5} | {'Isolated':>8} | {'Deg min/med/max':<16}")
    print("=" * 90)

    for fabric_id in ["l1", "l2", "l3", "l4", "l5", "l7"]:
        if fabric_id not in fabrics:
            desc = FABRIC_DESCRIPTIONS.get(fabric_id, "")[:44]
            print(f"{fabric_id:<6} | {desc:<45} | {'(not built)':>6} | {'—':>5} | {'—':>8} | {'—':<16}")
            continue
        stats = fabrics[fabric_id]["stats"]
        desc = FABRIC_DESCRIPTIONS.get(fabric_id, "")[:44]
        deg_str = f"{stats['degree_min']}/{stats['degree_median']}/{stats['degree_max']}"
        print(
            f"{fabric_id:<6} | {desc:<45} | {stats['edge_count']:>6} | "
            f"{stats['nodes_touched']:>5} | {stats['isolated']:>8} | {deg_str:<16}"
        )

    # L6 is tag-based — show vocab/assignment counts instead of edges
    if l6_data:
        vocab_size = len(l6_data.get("vocab", []))
        assignments = l6_data.get("assignments", [])
        tagged = sum(1 for a in assignments if a.get("tags"))
        desc = FABRIC_DESCRIPTIONS.get("l6", "")[:44]
        print(
            f"{'l6':<6} | {desc:<45} | {vocab_size:>3} v | "
            f"{tagged:>5} | {total_notes - tagged:>8} | {'(tag-based)':<16}"
        )
    else:
        desc = FABRIC_DESCRIPTIONS.get("l6", "")[:44]
        print(f"{'l6':<6} | {desc:<45} | {'(not built)':>6} | {'—':>5} | {'—':>8} | {'—':<16}")

    print("=" * 90)
    print()

    # Per-fabric top-5 degree notes
    for fabric_id in ["l1", "l2", "l3", "l4", "l5", "l7"]:
        if fabric_id not in fabrics:
            continue
        stats = fabrics[fabric_id]["stats"]
        print(f"Top-5 highest-degree notes — {fabric_id.upper()} ({FABRIC_DESCRIPTIONS.get(fabric_id, '')}):")
        for basename, deg in stats["top5"]:
            # Shorten basename for display
            display = basename if len(basename) <= 70 else basename[:67] + "..."
            print(f"  [{deg:>3}] {display}")
        print()

    # L6 top-5 tags by assignment count
    if l6_data:
        assignments_l6 = l6_data.get("assignments", [])
        tag_freq: dict[str, int] = {}
        for a in assignments_l6:
            for tag in a.get("tags", []):
                tag_freq[tag] = tag_freq.get(tag, 0) + 1
        print(f"Top-5 most-assigned tags — L6 ({FABRIC_DESCRIPTIONS.get('l6', '')}):")
        for tag, count in sorted(tag_freq.items(), key=lambda kv: kv[1], reverse=True)[:5]:
            print(f"  [{count:>3}] {tag}")
        print()

    # Summary counts from replays
    replays_path = os.path.join(HERE, "replays.json")
    queries_path = os.path.join(HERE, "queries.json")
    if os.path.exists(replays_path) and os.path.exists(queries_path):
        with open(queries_path) as fh:
            queries = json.load(fh)
        with open(replays_path) as fh:
            replays = json.load(fh)
        n3 = sum(1 for r in replays if r.get("n") == 3 and "error" not in r)
        n10 = sum(1 for r in replays if r.get("n") == 10 and "error" not in r)
        total_notes_replayed = sum(
            len(r.get("ranked_notes", [])) for r in replays if r.get("n") == 10 and "error" not in r
        )
        print(f"Replay summary (2026-07-02):")
        print(f"  Query sets extracted: {len(queries)}")
        print(f"  Replays n=3 (glance): {n3}")
        print(f"  Replays n=10 (deep):  {n10}")
        print(f"  Total note items in n=10 replays: {total_notes_replayed}")
        print()


if __name__ == "__main__":
    main()
