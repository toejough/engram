#!/usr/bin/env python3
"""Deterministic scorer for the engram retrieval probe.
For each probe (target note + nuanced query + lexical query), run the REAL engram binary in two paths:
  - isolation: ENGRAM_CHUNKS_DIR=<empty>  -> notes-only ranking (MiniLM cosine, no chunk drowning)
  - realpath : full chunk index           -> ranking among ALL items (what the agent actually sees)
Record the target note's rank + score in each. Compute recall@k, MRR, median rank.
"""
import json, os, re, subprocess, sys, tempfile, statistics

VAULT = "/Users/joe/.local/share/engram/vault"
EMPTY = tempfile.mkdtemp(prefix="engram-empty-chunks-")

def run_query(phrase, isolation):
    env = dict(os.environ)
    args = ["engram", "query", "--phrase", phrase]
    if isolation:
        env["ENGRAM_CHUNKS_DIR"] = EMPTY
    else:
        args.insert(2, "--lazy-chunks")
    try:
        out = subprocess.run(args, env=env, capture_output=True, text=True, timeout=120).stdout
    except Exception as e:
        return []
    # parse item blocks: "- path: X \n kind: Y \n score: Z"
    items = re.findall(r'-\s*path:\s*(\S+)\s*\n\s*kind:\s*(\w+)\s*\n\s*score:\s*([\d.]+)', out)
    return [{"path": p, "kind": k, "score": float(s)} for p, k, s in items]

def rank_of(items, basename, notes_only=False):
    """1-based rank of the target note; None if absent. notes_only restricts to fact/feedback items."""
    seq = [it for it in items if it["kind"] in ("fact", "feedback")] if notes_only else items
    for i, it in enumerate(seq):
        if basename in it["path"]:
            return i + 1, it["score"], len(seq)
    return None, None, len(seq)

def score_probe(p):
    bn = p["basename"]
    r = {"basename": bn, "abstract": p.get("abstract")}
    for qkind in ("nuanced", "lexical"):
        phrase = p[f"{qkind}_query"]
        iso = run_query(phrase, isolation=True)
        real = run_query(phrase, isolation=False)
        ir, iscore, itot = rank_of(iso, bn)                  # notes-only by construction
        ar, ascore, atot = rank_of(real, bn, notes_only=False)  # rank among ALL items (chunks+notes)
        nr, nscore, ntot = rank_of(real, bn, notes_only=True)   # rank among notes within the real payload
        r[qkind] = {
            "phrase": phrase,
            "iso_rank": ir, "iso_score": iscore, "iso_total": itot,
            "real_rank_all": ar, "real_total_all": atot,
            "real_rank_notes": nr,
        }
    return r

def metrics(rows, qkind, field, ks=(1,3,5,10)):
    ranks = [row[qkind][field] for row in rows]
    present = [x for x in ranks if x is not None]
    m = {f"recall@{k}": round(sum(1 for x in present if x <= k)/len(ranks), 3) for k in ks}
    m["MRR"] = round(sum((1.0/x) for x in present)/len(ranks), 3)
    m["median_rank"] = (statistics.median(present) if present else None)
    m["miss_rate"] = round(sum(1 for x in ranks if x is None)/len(ranks), 3)
    m["n"] = len(ranks)
    return m

if __name__ == "__main__":
    probes = json.load(open(sys.argv[1]))
    rows = []
    for i, p in enumerate(probes):
        rows.append(score_probe(p))
        print(f"  scored {i+1}/{len(probes)}: {p['basename'][:45]}", file=sys.stderr)
    out = {"rows": rows}
    for qkind in ("nuanced", "lexical"):
        out[qkind] = {
            "isolation_notes": metrics(rows, qkind, "iso_rank"),
            "realpath_all_items": metrics(rows, qkind, "real_rank_all"),
            "realpath_notes_only": metrics(rows, qkind, "real_rank_notes"),
        }
    json.dump(out, open(sys.argv[2], "w"), indent=1)
    print(json.dumps({k: out[k] for k in ("nuanced","lexical")}, indent=1))
