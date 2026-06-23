"""Slice-2 REAL-SKILL validation: run the warm /recall skill (not a bare `engram query`)
over the transitive fixture and measure whether the AGENT surfaces AND USES the graph-expanded
bridge note in its answer.

The transitive vault: joe-wants-cake -> cake-needs-sweetness -> sugar-provides-sweetness (chain
edges written by slice-1 `engram amend`). The task asks what to buy. Cosine matches the cake/Joe
notes but misses `sugar-provides-sweetness` (the answer) — only graph expansion surfaces it. A
useful system requires the recall SKILL to read that bridge and name sugar.

Measures per run:
  surfaced : the agent's `engram query` payload contained the bridge (graph expansion worked)
  shown    : the agent ran `engram show` on the bridge (read its content)
  used     : the agent's final answer names the bridge's subject ("sugar") as what to buy

Uses ENGRAM_BIN's directory on PATH so the agent's `engram` is the slice-2 build.

Usage: ENGRAM_BIN=/tmp/s2bin/engram python3 graphexpand_warm.py [--n 3] [--model opus]
"""
import argparse
import concurrent.futures as cf
import json
import os
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import cake_fixtures
from run import MODELS
from wrun import build_warm_cfg, _slug, RECALL_PREFIX

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/graphexpand-warm")
BIN = os.environ.get("ENGRAM_BIN", "engram")
KIND = os.environ.get("FIXTURE_KIND", "transitive")
# BRIDGE = the bridge note's basename slug (for surfaced/shown detection);
# BRIDGE_SUBJECT = the token the answer must contain to count as "used".
BRIDGE = os.environ.get("BRIDGE", "sugar-provides-sweetness")
BRIDGE_SUBJECT = os.environ.get("BRIDGE_SUBJECT", "sugar")
TASK = os.environ.get("TASK",
                      "Joe wants to bake a cake. Using ONLY your recalled memory about what a cake needs and "
                      "which ingredient provides it, tell me the one grocery item to buy. Name the item.")


def run_one(model, cfg, idx):
    wd = tempfile.mkdtemp(prefix=f"tr-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault")
    cake_fixtures.build(KIND, vault)
    chunks = os.path.join(wd, "chunks")
    os.makedirs(chunks, exist_ok=True)

    env = dict(os.environ)
    env["PATH"] = os.path.dirname(os.path.abspath(BIN)) + os.pathsep + env.get("PATH", "")
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = chunks
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", _slug(wd))

    args = ["claude", "-p", RECALL_PREFIX + TASK, "--output-format", "json",
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

    answer = (out.get("result") or "").lower()
    used = BRIDGE_SUBJECT in answer

    surfaced = shown = False
    sid = out.get("session_id")
    if sid:
        for rt, _, fs in os.walk(os.path.join(cfg, "projects")):
            if f"{sid}.jsonl" in fs:
                tx = open(os.path.join(rt, f"{sid}.jsonl"), errors="ignore").read()
                surfaced = ("engram query" in tx) and (BRIDGE in tx)
                shown = f"engram show" in tx and BRIDGE in tx.split("engram show", 1)[-1][:400]
                break

    return {"idx": idx, "surfaced": surfaced, "shown": shown, "used": used,
            "cost": out.get("total_cost_usd", 0) or 0, "answer": answer[:160], "wd": wd}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=3)
    ap.add_argument("--workers", type=int, default=3)
    a = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "warm-cfg")
    build_warm_cfg(cfg)

    print(f"transitive WARM /recall: n={a.n} ({a.model}); bin={BIN}")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, a.model, cfg, i): i for i in range(a.n)}
        for fut in cf.as_completed(futs):
            r = fut.result()
            results.append(r)
            print(f"  [#{r['idx']}] surfaced={r['surfaced']} shown={r['shown']} used={r['used']} "
                  f"${r['cost']:.2f}  ans={r['answer']!r}")

    s = sum(x["surfaced"] for x in results)
    sh = sum(x["shown"] for x in results)
    u = sum(x["used"] for x in results)
    print(f"\n=== transitive WARM (n={a.n}): surfaced={s}/{a.n} shown={sh}/{a.n} used={u}/{a.n} ===")
    json.dump(results, open(os.path.join(ROOT, "results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
