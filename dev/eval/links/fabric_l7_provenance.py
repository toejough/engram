"""Fabric L7: provenance/episode edges (same-origin session/source).

Parses each note's frontmatter `source:` line and extracts a "session date"
family key. Notes sharing the same session date are linked. The family key is
the ISO date extracted from "session YYYY-MM-DD" patterns, or from the first
YYYY-MM-DD occurrence in the source string.

Notes created from synthesis (multiple sessions or non-session sources) are
assigned a synthetic family based on their creation date if no session date is
present — these can still share an origin date with each other.

Writes dev/eval/links/fabrics/l7.json: [{src, dst, key: source-family}]
"""
import glob
import json
import os
import re
import sys
from collections import defaultdict

VAULT_PATH = os.path.expanduser("~/.local/share/engram/vault/")
OUT_PATH = os.path.join(os.path.dirname(__file__), "fabrics", "l7.json")

# Matches "session YYYY-MM-DD" (primary pattern for session-origin notes)
SESSION_DATE_RE = re.compile(r"session\s+(\d{4}-\d{2}-\d{2})")
# Fallback: any YYYY-MM-DD in the source field
ANY_DATE_RE = re.compile(r"(\d{4}-\d{2}-\d{2})")
# Frontmatter block
FRONTMATTER_RE = re.compile(r"^---\n(.*?)\n---", re.DOTALL)
SOURCE_LINE_RE = re.compile(r"^source:\s*[\"']?(.+?)[\"']?\s*$", re.MULTILINE)
CREATED_LINE_RE = re.compile(r'^created:\s*["\']?(\d{4}-\d{2}-\d{2})["\']?\s*$', re.MULTILINE)


def extract_source_family(content: str) -> str | None:
    """Return the source-family key for a note, or None if unresolvable."""
    source_match = SOURCE_LINE_RE.search(content)
    source = source_match.group(1).strip() if source_match else ""

    # Primary: session date
    session_match = SESSION_DATE_RE.search(source)
    if session_match:
        return f"session:{session_match.group(1)}"

    # Fallback 1: any date in source (synthesis notes often have a date)
    date_match = ANY_DATE_RE.search(source)
    if date_match:
        return f"date:{date_match.group(1)}"

    # Fallback 2: created date from frontmatter
    created_match = CREATED_LINE_RE.search(content)
    if created_match:
        return f"created:{created_match.group(1)}"

    return None


def main() -> None:
    md_files = sorted(glob.glob(os.path.join(VAULT_PATH, "*.md")))
    if not md_files:
        sys.exit(f"ERROR: no .md files found at {VAULT_PATH}")

    print(f"Parsing source fields in {len(md_files)} notes …", flush=True)

    # Build family → list of note basenames
    family_notes: dict[str, list[str]] = defaultdict(list)
    unresolved = 0

    for fpath in md_files:
        basename = os.path.basename(fpath)[:-3]
        with open(fpath) as fh:
            content = fh.read()
        family = extract_source_family(content)
        if family:
            family_notes[family].append(basename)
        else:
            unresolved += 1
            print(f"  WARNING: no source family for {basename}", flush=True)

    print(f"\nSource families: {len(family_notes)}", flush=True)
    print(f"Unresolved (no family): {unresolved}", flush=True)

    # Print family sizes
    family_sizes = sorted(
        ((fam, len(notes)) for fam, notes in family_notes.items()),
        key=lambda kv: kv[1],
        reverse=True,
    )
    print("\nTop families by size:", flush=True)
    for fam, size in family_sizes[:15]:
        print(f"  {fam}: {size} notes", flush=True)

    # Build edges: for each family with ≥2 notes, link all pairs
    edge_set: dict[tuple[str, str], str] = {}  # (src, dst) → family_key

    for family, notes in family_notes.items():
        if len(notes) < 2:
            continue
        notes_sorted = sorted(notes)
        for i in range(len(notes_sorted)):
            for j in range(i + 1, len(notes_sorted)):
                src, dst = notes_sorted[i], notes_sorted[j]
                key = (src, dst)
                if key not in edge_set:
                    edge_set[key] = family

    edges = [
        {"src": src, "dst": dst, "key": family}
        for (src, dst), family in sorted(edge_set.items())
    ]

    # Degree distribution
    degree: dict[str, int] = defaultdict(int)
    for e in edges:
        degree[e["src"]] += 1
        degree[e["dst"]] += 1

    nodes_touched = len(degree)
    degrees = sorted(degree.values(), reverse=True)
    isolated = len(md_files) - nodes_touched

    print(f"\nDegree distribution (L7):", flush=True)
    print(f"  Nodes touched: {nodes_touched} / {len(md_files)}", flush=True)
    print(f"  Isolated (no edges): {isolated}", flush=True)
    if degrees:
        print(f"  Min degree: {degrees[-1]}", flush=True)
        print(f"  Median degree: {degrees[len(degrees)//2]}", flush=True)
        print(f"  Max degree: {degrees[0]}", flush=True)

    print("\nTop-5 highest-degree notes (L7):", flush=True)
    degree_sorted = sorted(degree.items(), key=lambda kv: kv[1], reverse=True)
    for basename, deg in degree_sorted[:5]:
        print(f"  {basename}: degree {deg}", flush=True)

    print(f"\nEdge count: {len(edges)}", flush=True)

    os.makedirs(os.path.dirname(OUT_PATH), exist_ok=True)
    with open(OUT_PATH, "w") as fh:
        json.dump(edges, fh, indent=2)

    print(f"Wrote {len(edges)} edges → {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
