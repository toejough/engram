#!/usr/bin/env python3
"""Build value-test contexts for the drowned situations.
For each drowned-and-relevant situation: run the REAL real-path engram query, take the top-K items,
fetch their content (show-chunk for chunks; note .md for notes), assemble the 'chunks' condition context
(EXCLUDING the target note if it surfaced), and grab the target note's content separately.
"""
import json, subprocess, re, os, glob

SC = "/private/tmp/claude-501/-Users-joe-repos-personal-engram/95570838-0d05-483c-95e7-fe004909b499/scratchpad"
VAULT = "/Users/joe/.local/share/engram/vault"
TOPK = 6
CHUNK_CHARS = 700

res = {r["basename"]: r for r in json.load(open(f"{SC}/probe_results.json"))["rows"]}
probes = {p["basename"]: p for p in json.load(open(f"{SC}/probes.json"))}

drowned = [bn for bn, r in res.items()
           if r["nuanced"]["real_rank_all"] is None
           and r["nuanced"]["iso_rank"] is not None and r["nuanced"]["iso_rank"] <= 5]

def note_content(basename):
    f = os.path.join(VAULT, basename + ".md")
    if not os.path.exists(f):
        g = glob.glob(os.path.join(VAULT, basename.split('.')[0] + ".*.md"))
        f = g[0] if g else None
    if not f: return ""
    txt = open(f).read()
    parts = txt.split("---")
    body = parts[2].strip() if len(parts) >= 3 else txt
    return re.sub(r"\n{3,}", "\n\n", body).strip()[:900]

def fetch_chunk(cid):
    try:
        out = subprocess.run(["engram", "show-chunk", cid], capture_output=True, text=True, timeout=60).stdout
        return re.sub(r"\s+", " ", out).strip()[:CHUNK_CHARS]
    except Exception:
        return ""

contexts = []
for bn in drowned:
    p = probes[bn]
    situation = p["nuanced_query"]
    lesson = note_content(bn)
    out = subprocess.run(["engram", "query", "--lazy-chunks", "--phrase", situation],
                         capture_output=True, text=True, timeout=120).stdout
    items = re.findall(r"-\s*path:\s*(\S+)\s*\n\s*kind:\s*(\w+)\s*\n\s*score:\s*([\d.]+)", out)
    chunk_blocks = []
    for path, kind, score in items:
        if len(chunk_blocks) >= TOPK: break
        if kind in ("fact", "feedback"):
            if bn in path:   # never include the target note in the 'chunks' condition
                continue
            nc = note_content(path.split("/")[-1].replace(".md", ""))
            chunk_blocks.append(f"[note] {nc[:400]}")
        else:
            c = fetch_chunk(path)
            if c: chunk_blocks.append(f"[chunk {path.split('/')[-1]}] {c}")
    contexts.append({
        "basename": bn, "abstract": res[bn]["abstract"],
        "situation": situation, "lesson": lesson,
        "chunks_context": "\n\n".join(chunk_blocks),
        "n_chunks": len(chunk_blocks),
    })
    print(f"  built {bn[:40]}: {len(chunk_blocks)} surfaced items", flush=True)

json.dump(contexts, open(f"{SC}/value_contexts.json", "w"), indent=1)
print(f"\n{len(contexts)} contexts saved; mean surfaced items {sum(c['n_chunks'] for c in contexts)/len(contexts):.1f}")
