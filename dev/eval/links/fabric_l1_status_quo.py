"""Fabric L1: status-quo wikilink edges.

Extracts resolved wikilink edges from the live vault using the same
exact-basename resolver as internal/vaultgraph (Go). A wikilink [[target]]
resolves if `target` == some note's basename (filename without .md).

The plan notes 77 resolving edges (measured 2026-07-02). Writes:
  dev/eval/links/fabrics/l1.json: [{src, dst}]

The 4 links pointing at auto-memory files outside the vault are excluded
(broken under any resolver). The 3 slug-only prefix-drift links are
excluded here (they don't match the exact-basename resolver).
"""
import glob
import json
import os
import re
import sys

VAULT_PATH = os.path.expanduser("~/.local/share/engram/vault/")
OUT_PATH = os.path.join(os.path.dirname(__file__), "fabrics", "l1.json")

WIKILINK_RE = re.compile(r"\[\[([^\]\n]+)\]\]")
FENCE_RE = re.compile(r"^(```|~~~)")


def parse_wikilinks(body: str) -> list[str]:
    """Extract deduped wikilink targets, skipping fenced code blocks.

    Matches internal/vaultgraph.ParseWikilinks behavior exactly:
    - Whitespace-only or empty link bodies are dropped.
    - Wikilinks inside fenced code blocks (``` or ~~~) are skipped.
    - Self-links dropped later in graph building.
    """
    seen: set[str] = set()
    out: list[str] = []
    in_fence = False
    fence_char: str = ""
    fence_len: int = 0

    for line in body.splitlines():
        stripped = line.strip()
        # Fence open/close detection
        if stripped.startswith("```") or stripped.startswith("~~~"):
            char = stripped[0]
            run = len(stripped) - len(stripped.lstrip(char))
            if not in_fence:
                in_fence = True
                fence_char = char
                fence_len = run
                continue
            elif char == fence_char and run >= fence_len:
                in_fence = False
                continue
            # Closer doesn't match — still inside
        if in_fence:
            continue

        for m in WIKILINK_RE.finditer(line):
            target = m.group(1)
            if not target or not target.strip():
                continue
            if target not in seen:
                seen.add(target)
                out.append(target)

    return out


def main() -> None:
    md_files = sorted(glob.glob(os.path.join(VAULT_PATH, "*.md")))
    if not md_files:
        sys.exit(f"ERROR: no .md files found at {VAULT_PATH}")

    print(f"Found {len(md_files)} notes in vault", flush=True)

    # Build basename → filename map (exact-basename resolver)
    basename_map: dict[str, str] = {}
    for fpath in md_files:
        filename = os.path.basename(fpath)
        basename = filename[:-3]  # strip .md
        basename_map[basename] = filename

    # Parse wikilinks per note
    edges: list[dict] = []
    broken_count = 0
    broken_examples: list[str] = []

    for fpath in md_files:
        src_basename = os.path.basename(fpath)[:-3]
        with open(fpath) as fh:
            content = fh.read()

        targets = parse_wikilinks(content)
        for target in targets:
            # Self-links dropped
            if target == src_basename:
                continue
            # Exact-basename resolution
            if target in basename_map:
                edges.append({"src": src_basename, "dst": target})
            else:
                broken_count += 1
                if len(broken_examples) < 10:
                    broken_examples.append(f"{src_basename} → [[{target}]]")

    print(f"Resolved edges: {len(edges)}", flush=True)
    print(f"Broken (unresolved) links: {broken_count}", flush=True)
    if broken_examples:
        print("Broken examples:", flush=True)
        for ex in broken_examples:
            print(f"  {ex}", flush=True)

    os.makedirs(os.path.dirname(OUT_PATH), exist_ok=True)
    with open(OUT_PATH, "w") as fh:
        json.dump(edges, fh, indent=2)

    print(f"Wrote {len(edges)} edges → {OUT_PATH}", flush=True)


if __name__ == "__main__":
    main()
