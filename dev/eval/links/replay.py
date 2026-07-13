"""HISTORICAL INSTRUMENT (pre-#684): this script belongs to the closed S1-S3 link-value
exploration (final verdicts 2026-07-02, `dev/eval/LEDGER.md#vocab-tag-nomination-l6xtag` /
`#ppr-killed` / `#supersession-edges-mechanism`; plan doc deleted 2026-07-05) and is not a live
consumer (no targ target/CI invokes it; judge_misses.py only reads its cached replays.json,
manually). The 2026-07-12 clusters-first/lazy-note-content payload restructure (#684) moved notes'
content out of items[] into candidate_l2s and reordered clusters before items — parse_items'
items->clusters end-sentinel is now unreachable in the new struct order (items[] is followed by
budget:, not clusters:) and note items never carry content in items[] anymore. Do NOT re-run
without porting this parser to the new shape; results here are vintage of the pre-#684 payload.

Replay each query set from queries.json against the live vault.

For each query set, runs `engram query --lazy-chunks` twice:
  - n=3 (first 3 phrases, simulates glance's positional-rule floor)
  - n=10 (all phrases up to 10, simulates deep recall)

Parses the YAML output for ranked NOTE items (path, score, rank) and cluster
candidate_l2s. Writes dev/eval/links/replays.json:
  [{query_id, n, ranked_notes: [{basename, score, rank}], candidates: [...]}]

Also captures each replay's full top-10 for the no-regression set.
Parsing patterns follow dev/eval/traps/retrieval_probe.py conventions.
"""
import json
import os
import re
import subprocess
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
QUERIES_PATH = os.path.join(HERE, "queries.json")
OUT_PATH = os.path.join(HERE, "replays.json")

# Patterns for the YAML payload produced by `engram query`
_ITEM_PATH_LINE = re.compile(r"^  - path:\s*(.+?)\s*$")
_ITEM_SCORE_LINE = re.compile(r"^\s+score:\s*([\d.]+)\s*$")
_ITEM_KIND_LINE = re.compile(r"^\s+kind:\s*(\S+)\s*$")
_CANDIDATE_BLOCK = re.compile(r"candidate_l2s:")
_CANDIDATE_PATH = re.compile(r"^\s+- path:\s*(.+?)\s*$")
_CANDIDATE_COSINE = re.compile(r"^\s+cosine:\s*([\d.]+)\s*$")


def parse_items(text: str) -> list[dict]:
    """Extract top-level items[] from engram query YAML output.

    Returns list of {basename, score, rank} for note items.
    Chunks are included (rank counted) but chunk items have no .md in their path
    under --lazy-chunks (they appear as path-only). We track all items for rank
    counting but only return note items (path ends with .md after stripping #anchor).
    """
    items = []
    rank = 0
    current_path: str | None = None
    current_score: float | None = None
    current_kind: str | None = None
    in_items = False

    for line in text.splitlines():
        # Detect start of top-level items block
        if line == "items:":
            in_items = True
            continue
        # Detect end of items block (clusters: or metadata block)
        if in_items and re.match(r"^[a-z_]+:", line) and not line.startswith("  "):
            if line.startswith("clusters:") or line.startswith("metadata:"):
                in_items = False
                # Flush last item
                if current_path is not None:
                    rank += 1
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

        path_match = _ITEM_PATH_LINE.match(line)
        if path_match:
            # Flush previous item
            if current_path is not None:
                rank += 1
                items.append({
                    "basename": current_path,
                    "score": current_score,
                    "kind": current_kind,
                    "rank": rank,
                })
            raw_path = path_match.group(1).strip().strip('"')
            # Strip #anchor suffix, take basename
            raw_path = raw_path.split("#")[0]
            current_path = os.path.basename(raw_path)
            current_score = None
            current_kind = None
            continue

        if current_path is not None:
            score_match = _ITEM_SCORE_LINE.match(line)
            if score_match:
                current_score = float(score_match.group(1))
                continue
            kind_match = _ITEM_KIND_LINE.match(line)
            if kind_match:
                current_kind = kind_match.group(1)

    # Flush last item if loop ended while still in items block
    if in_items and current_path is not None:
        rank += 1
        items.append({
            "basename": current_path,
            "score": current_score,
            "kind": current_kind,
            "rank": rank,
        })

    return items


def parse_candidates(text: str) -> list[dict]:
    """Extract candidate_l2s paths + cosine scores from all clusters."""
    candidates = []
    in_cands = False
    current_path: str | None = None
    current_cosine: float | None = None

    for line in text.splitlines():
        if "candidate_l2s:" in line:
            in_cands = True
            continue
        if in_cands:
            # End of this candidate_l2s block when we hit a line that isn't indented
            # deeply enough or is a new top-level field. Cluster-level fields start
            # at 4 spaces ("    phrase:" etc).
            if line and not line.startswith("    "):
                # Flush
                if current_path is not None:
                    candidates.append({"basename": current_path, "cosine": current_cosine})
                    current_path = current_cosine = None
                in_cands = False
                continue

            cand_path_match = _CANDIDATE_PATH.match(line)
            if cand_path_match:
                if current_path is not None:
                    candidates.append({"basename": current_path, "cosine": current_cosine})
                raw = cand_path_match.group(1).strip().strip('"').split("#")[0]
                current_path = os.path.basename(raw)
                current_cosine = None
                continue

            if current_path is not None:
                cosine_match = _CANDIDATE_COSINE.match(line)
                if cosine_match:
                    current_cosine = float(cosine_match.group(1))

    if in_cands and current_path is not None:
        candidates.append({"basename": current_path, "cosine": current_cosine})

    return candidates


def run_query(phrases: list[str], n: int) -> dict:
    """Run `engram query --lazy-chunks` with the first n phrases.

    Returns {ranked_notes, candidates, raw_output} or raises on failure.
    """
    cmd = ["engram", "query", "--lazy-chunks"]
    for phrase in phrases[:n]:
        cmd += ["--phrase", phrase]

    result = subprocess.run(cmd, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(
            f"engram query exited {result.returncode}: {result.stderr.strip()}"
        )

    raw = result.stdout
    ranked_notes = parse_items(raw)
    candidates = parse_candidates(raw)
    return {"ranked_notes": ranked_notes, "candidates": candidates, "raw": raw}


def main() -> None:
    if not os.path.exists(QUERIES_PATH):
        sys.exit(f"ERROR: {QUERIES_PATH} not found — run extract_queries.py first")

    with open(QUERIES_PATH) as fh:
        query_sets = json.load(fh)

    print(f"Replaying {len(query_sets)} query sets (n=3 and n=10 each) …", flush=True)

    results = []
    for qs in query_sets:
        query_id = qs["id"]
        phrases = qs["phrases"]
        print(f"  {query_id}: {len(phrases)} phrases, date={qs['approx_date']} …", flush=True)

        for n_phrases in (3, 10):
            actual_n = min(n_phrases, len(phrases))
            try:
                out = run_query(phrases, actual_n)
            except RuntimeError as exc:
                print(f"    ERROR (n={n_phrases}): {exc}", flush=True)
                results.append({
                    "query_id": query_id,
                    "n": n_phrases,
                    "error": str(exc),
                    "ranked_notes": [],
                    "candidates": [],
                })
                continue

            ranked_notes = out["ranked_notes"]
            candidates = out["candidates"]

            # Note items only (path ends with .md)
            note_items = [item for item in ranked_notes if item["basename"].endswith(".md")]

            print(
                f"    n={n_phrases}: {len(ranked_notes)} items total, "
                f"{len(note_items)} notes, {len(candidates)} candidates",
                flush=True,
            )
            results.append({
                "query_id": query_id,
                "n": n_phrases,
                "phrases_used": phrases[:actual_n],
                "ranked_notes": note_items,
                "candidates": candidates,
                # Full top-10 items (notes+chunks) for no-regression set
                "top10_all": ranked_notes[:10],
            })

    with open(OUT_PATH, "w") as fh:
        json.dump(results, fh, indent=2)

    print(f"\nWrote {len(results)} replay records → {OUT_PATH}", flush=True)

    # Summary
    n3_count = sum(1 for r in results if r.get("n") == 3 and "error" not in r)
    n10_count = sum(1 for r in results if r.get("n") == 10 and "error" not in r)
    print(f"n=3 successful: {n3_count}  |  n=10 successful: {n10_count}", flush=True)


if __name__ == "__main__":
    main()
