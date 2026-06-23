"""Cake cross-cluster check. Build a fixture vault, run the real warm /recall over it
(skill Step 2.6 should write cross-cluster edges), then inspect the vault's [[wikilinks]].

Usage: python3 cake.py [--kind cake] [--model opus] [--n 3] [--workers 3]
"""
import argparse
import concurrent.futures as cf
import glob
import json
import os
import re
import sys
import tempfile
import time
import subprocess

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import cake_fixtures
from run import MODELS
from wrun import build_warm_cfg, _slug

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/cake")
WIKILINK = re.compile(r"\[\[([^\]]+)\]\]")
PROMPT = (
    "Invoke your /recall skill for this situation, processing EVERY cluster it returns exactly as "
    "the skill directs — including any cross-cluster linking step. Situation: I am planning how to "
    "bake a cake and what each ingredient contributes.\n\nAfter recall, briefly state your plan — no code.")


def _domain(basename):
    # strip a leading Luhmann prefix like "7.2026-06-23." then take the first slug token
    slug = re.sub(r"^\d+\.\d{4}-\d{2}-\d{2}\.", "", basename)
    return slug.split("-", 1)[0]                 # cake / sugar / flour / git / joe …


def inspect_edges(vault):
    edges = {}
    for f in glob.glob(os.path.join(vault, "*.md")):
        base = os.path.basename(f)
        text = open(f, errors="ignore").read()
        targets = set()
        for m in WIKILINK.findall(text):
            t = m.split("|")[0].strip()
            if not t.endswith(".md"):
                t += ".md"
            targets.add(t)
        edges[base] = sorted(targets)
    return edges


def classify_cross(vault):
    edges = inspect_edges(vault)
    cross = []
    for src, dsts in edges.items():
        for d in dsts:
            if _domain(d) != _domain(src) and os.path.exists(os.path.join(vault, d)):
                cross.append((src, d))
    return sorted(set(cross))


def _property(basename):
    # the means-ends shared key is the final slug token: cake-needs-SWEETNESS / sugar-provides-SWEETNESS
    slug = re.sub(r"^\d+\.\d{4}-\d{2}-\d{2}\.", "", basename)[:-3]   # strip prefix + .md
    return slug.rsplit("-", 1)[-1]


def classify_means_ends(vault):
    """Precision view: split every cross-note edge into property-MATCHED means-ends links
    (need-X <-> provides-X, same X) vs SPURIOUS links (property mismatch / topical flood)."""
    correct, spurious = [], []
    for src, d in classify_cross(vault):
        # the flood metric is about links BETWEEN the original need-/provides- fixture notes;
        # a link to/from a newly-crystallized synthesis note is Step 2.5 provenance, measured
        # separately by note_delta — exclude it here so it doesn't masquerade as a flood.
        if not (("-needs-" in src or "-provides-" in src) and ("-needs-" in d or "-provides-" in d)):
            continue
        sp, dp = _property(src), _property(d)
        is_needs = "needs" in src
        is_prov = "provides" in d
        if sp == dp and is_needs and is_prov:
            correct.append((src, d))
        # also accept the reverse direction provides-X <- ... ; only the property-match matters for "correct"
        elif sp == dp and "provides" in src and "needs" in d:
            correct.append((src, d))
        else:
            spurious.append((src, d))
    # dedupe directionless for the count (A->B and B->A on the same property are one means-ends join)
    seen, uniq_correct = set(), []
    for s, d in correct:
        key = tuple(sorted((_property(s), _property(d)))) + (_property(s),)
        if _property(s) not in seen:
            seen.add(_property(s))
            uniq_correct.append((s, d))
    return uniq_correct, spurious


def run_one(kind, model, cfg, idx):
    wd = tempfile.mkdtemp(prefix=f"{kind}-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault")
    cake_fixtures.build(kind, vault)
    before = len(glob.glob(os.path.join(vault, "*.md")))
    chunks = os.path.join(wd, "chunks")
    os.makedirs(chunks, exist_ok=True)
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = chunks
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", _slug(wd))
    args = ["claude", "-p", PROMPT, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = {}
        if (out.get("is_error") or not out) and (out.get("total_cost_usd", 0) or 0) < 0.02:
            continue
        break
    after = len(glob.glob(os.path.join(vault, "*.md")))
    cross = classify_cross(vault)
    correct, spurious = classify_means_ends(vault)
    sid = out.get("session_id")
    recalled = False
    if sid:
        for rt, _, fs in os.walk(os.path.join(cfg, "projects")):
            if f"{sid}.jsonl" in fs:
                tx = open(os.path.join(rt, f"{sid}.jsonl"), errors="ignore").read()
                recalled = "engram query" in tx
                break
    return {"kind": kind, "idx": idx, "cross_edges": cross,
            "correct_means_ends": correct, "spurious": spurious, "note_delta": after - before,
            "recalled": recalled, "cost": out.get("total_cost_usd", 0) or 0,
            "turns": out.get("num_turns"), "wd": wd}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--kind", default="cake")
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=3)
    ap.add_argument("--workers", type=int, default=3)
    a = ap.parse_args()
    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "warm-cfg")
    build_warm_cfg(cfg)
    print(f"cake check: kind={a.kind} × n={a.n} ({a.model})")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, a.kind, a.model, cfg, i): i for i in range(a.n)}
        for fut in cf.as_completed(futs):
            r = fut.result()
            results.append(r)
            print(f"  [{r['kind']} #{r['idx']}] correct_means_ends={len(r['correct_means_ends'])}/3 "
                  f"spurious={len(r['spurious'])} note_delta={r['note_delta']} "
                  f"recall={r['recalled']} ${r['cost']:.2f}")
            for s, d in r["spurious"]:
                print(f"        SPURIOUS {_property(s)} -> {_property(d)}")
    json.dump(results, open(os.path.join(ROOT, f"cake-{a.kind}-results.json"), "w"), indent=1)
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")


if __name__ == "__main__":
    main()
