"""Slice-2 proof: does graph expansion surface the transitive bridge cosine misses?

Builds the transitive fixture (chain edges present via `engram amend` -> body
"Related to:" wikilinks -> Note.Outgoing), runs `engram query` cosine-only
(--graph-expand-hops -1) vs expanded (default), and checks clusters[].members[].path.

Set ENGRAM_BIN to test a freshly-built binary (defaults to `engram` on PATH).

Usage: ENGRAM_BIN=/tmp/engram-s2 python3 graphexpand.py
"""
import os
import re
import subprocess
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import cake_fixtures

BIN = os.environ.get("ENGRAM_BIN", "engram")
QUERY = ["--phrase", "what ingredient should I buy at the store for the recipe"]
BRIDGE = "sugar-provides-sweetness"   # cosine-distant from the QUERY; only a graph hop reaches it


def _member_paths(out):
    # clusters[].members[].path — parse the `path:` lines that sit under a `members:` block.
    paths, in_members = [], False
    for line in out.splitlines():
        if re.match(r"\s*members:", line):
            in_members = True
            continue
        if in_members and re.match(r"\s*candidate_l2s:|\s*- id:", line):
            in_members = False
        if in_members:
            m = re.search(r"path:\s*(\S+)", line)
            if m:
                paths.append(m.group(1).strip("'\""))
    return paths


def _bridge_in_members(vault, extra):
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = os.path.join(vault, "_chunks")
    out = subprocess.run([BIN, "query", *QUERY, *extra], env=env,
                         capture_output=True, text=True).stdout
    return any(BRIDGE in os.path.basename(p) for p in _member_paths(out))


def main():
    vault = "/tmp/graphexpand/vault"
    cake_fixtures.build("transitive", vault)
    cosine = _bridge_in_members(vault, ["--graph-expand-hops", "-1"])
    expand = _bridge_in_members(vault, [])
    print(f"bridge '{BRIDGE}' in clusters[].members:  cosine-only={cosine}  graph-expanded={expand}")
    assert not cosine and expand, "expected bridge MISSED by cosine, SURFACED by expansion"
    print("PASS: graph expansion surfaces the transitive bridge cosine missed")


if __name__ == "__main__":
    main()
