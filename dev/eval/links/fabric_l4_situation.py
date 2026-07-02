"""Fabric L4: situation-cosine edges.

Reads each note's .vec.json sidecar `situation_vector` (384-dim). Computes
cosine similarity between all note pairs. Links pairs with situation cosine ≥
threshold, chosen to capture the top ~2-5% of pairs (meaningful-not-flood).

Threshold justification: reported from the score distribution histogram.

Writes dev/eval/links/fabrics/l4.json: [{src, dst, cosine}]
"""
import glob
import json
import math
import os
import sys
from collections import defaultdict

VAULT_PATH = os.path.expanduser("~/.local/share/engram/vault/")
OUT_PATH = os.path.join(os.path.dirname(__file__), "fabrics", "l4.json")


def load_situation_vectors(vault_path: str) -> dict[str, list[float]]:
    """Load situation_vector from each .vec.json sidecar. Returns basename → vector."""
    vec_files = sorted(glob.glob(os.path.join(vault_path, "*.vec.json")))
    vectors: dict[str, list[float]] = {}

    for fpath in vec_files:
        # Derive basename: strip .vec.json → get the note basename
        filename = os.path.basename(fpath)
        if not filename.endswith(".vec.json"):
            continue
        basename = filename[: -len(".vec.json")]

        with open(fpath) as fh:
            data = json.load(fh)

        vec = data.get("situation_vector")
        if not vec or not isinstance(vec, list):
            print(f"  WARNING: no situation_vector in {filename}", flush=True)
            continue

        vectors[basename] = vec

    return vectors


def cosine_similarity(a: list[float], b: list[float]) -> float:
    """Compute cosine similarity between two float vectors."""
    dot = sum(x * y for x, y in zip(a, b))
    norm_a = math.sqrt(sum(x * x for x in a))
    norm_b = math.sqrt(sum(x * x for x in b))
    if norm_a == 0 or norm_b == 0:
        return 0.0
    return dot / (norm_a * norm_b)


def main() -> None:
    print("Loading situation vectors …", flush=True)
    vectors = load_situation_vectors(VAULT_PATH)
    notes = sorted(vectors.keys())
    n = len(notes)

    if n < 2:
        sys.exit(f"ERROR: only {n} notes with situation vectors — need ≥2")

    print(f"Loaded {n} situation vectors (384-dim)", flush=True)

    # Compute all pairwise cosines (upper triangle)
    total_pairs = n * (n - 1) // 2
    print(f"Computing {total_pairs:,} pairwise cosines …", flush=True)

    all_cosines: list[tuple[float, str, str]] = []
    for i in range(n):
        for j in range(i + 1, n):
            cos = cosine_similarity(vectors[notes[i]], vectors[notes[j]])
            all_cosines.append((cos, notes[i], notes[j]))

    all_cosines.sort(reverse=True)

    # Build score distribution histogram (buckets of 0.05)
    print("\nSituation-cosine distribution (all pairs):", flush=True)
    buckets: dict[float, int] = defaultdict(int)
    for cos, _, _ in all_cosines:
        bucket = round(int(cos * 20) / 20, 2)  # floor to nearest 0.05
        buckets[bucket] += 1

    cumulative = 0
    for bucket in sorted(buckets.keys(), reverse=True):
        cumulative += buckets[bucket]
        pct = cumulative / total_pairs * 100
        print(f"  cosine ≥ {bucket:.2f}: {buckets[bucket]:>5} pairs this bucket  "
              f"(cumulative: {cumulative:>6} = {pct:.1f}%)", flush=True)
        if bucket < 0.50:
            break

    # Target top 2-5% of pairs → pick threshold where cumulative ≈ 2-5% of total_pairs
    target_min = int(0.02 * total_pairs)
    target_max = int(0.05 * total_pairs)
    print(f"\nTarget range: {target_min}–{target_max} pairs (2–5% of {total_pairs:,})", flush=True)

    # Walk cosines in descending order to find a natural threshold.
    # Start high, relax in 0.05 steps until we enter the target range.
    # Floor at 0.45 to avoid connecting everything.
    chosen_threshold = 0.90
    threshold_count = sum(1 for cos, _, _ in all_cosines if cos >= chosen_threshold)

    while threshold_count < target_min and chosen_threshold > 0.45:
        chosen_threshold -= 0.05
        chosen_threshold = round(chosen_threshold, 2)
        threshold_count = sum(1 for cos, _, _ in all_cosines if cos >= chosen_threshold)

    # If above target_max, tighten by 0.05
    while threshold_count > target_max and chosen_threshold <= 0.95:
        chosen_threshold += 0.05
        chosen_threshold = round(chosen_threshold, 2)
        threshold_count = sum(1 for cos, _, _ in all_cosines if cos >= chosen_threshold)

    print(f"Settled threshold: cosine ≥ {chosen_threshold:.2f}  ({threshold_count} pairs = "
          f"{threshold_count/total_pairs*100:.1f}% of all pairs)", flush=True)

    # Build edges
    edges = [
        {"src": src, "dst": dst, "cosine": round(cos, 6)}
        for cos, src, dst in all_cosines
        if cos >= chosen_threshold
    ]

    # Degree distribution
    degree: dict[str, int] = defaultdict(int)
    for e in edges:
        degree[e["src"]] += 1
        degree[e["dst"]] += 1

    nodes_touched = len(degree)
    degrees = sorted(degree.values(), reverse=True)
    isolated = n - nodes_touched

    print(f"\nDegree distribution (L4):", flush=True)
    print(f"  Nodes touched: {nodes_touched} / {n}", flush=True)
    print(f"  Isolated (no edges): {isolated}", flush=True)
    if degrees:
        print(f"  Min degree: {degrees[-1]}", flush=True)
        print(f"  Median degree: {degrees[len(degrees)//2]}", flush=True)
        print(f"  Max degree: {degrees[0]}", flush=True)

    print("\nTop-5 highest-degree notes (L4):", flush=True)
    degree_sorted = sorted(degree.items(), key=lambda kv: kv[1], reverse=True)
    for basename, deg in degree_sorted[:5]:
        print(f"  {basename}: degree {deg}", flush=True)

    os.makedirs(os.path.dirname(OUT_PATH), exist_ok=True)
    with open(OUT_PATH, "w") as fh:
        json.dump(edges, fh, indent=2)

    print(f"\nWrote {len(edges)} edges → {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
