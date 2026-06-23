"""Synthesis-layer C6 eval — RED arms (cold / warm-only) per the 2026-06-23-synthesis-layer.md spec.

The deciding question for the synthesis STEP comes later (warm+synth vs warm-only); THIS run answers
the gate-on-build question: does warm recall-only already compose the emergent C spontaneously?
  - If warm-only composes C at-or-above (100% - noise), the synthesis step is REDUNDANT -> STOP.
  - If warm-only leaves real headroom, there is something for a synthesis step to add.

Arms:
  cold      : opus, no memory, no skills, bare task           -> expect ~0 (lacks A,B by design)
  warm-only : warm /recall (current skill) over the seeded vault, RECALL_PREFIX + task

A separate judge (sonnet) rules HIT/MISS: does the answer state the fixture's emergent conclusion C?

Usage: ENGRAM_BIN unused (no binary change). python3 synth_eval.py [--n 6] [--model opus] [--workers 4]
"""
import argparse
import concurrent.futures as cf
import json
import os
import re
import subprocess
import sys
import tempfile
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import synth_fixtures
from run import build_cold_cfg, MODELS
from wrun import build_warm_cfg, _slug, RECALL_PREFIX

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/synth-eval")
KINDS = ["synth-join", "synth-chain", "synth-transfer"]

JUDGE_TMPL = (
    "You are grading whether an answer reaches a specific emergent conclusion.\n\n"
    "QUESTION:\n{task}\n\n"
    "REQUIRED EMERGENT CONCLUSION (the answer must state this causal/transitive/transfer link, "
    "not merely recite one fact):\n{C}\n\n"
    "CANDIDATE ANSWER:\n{answer}\n\n"
    "Does the candidate answer clearly state the required emergent conclusion? It must connect the "
    "pieces, not just mention one half. Reply with exactly 'HIT' or 'MISS' on the first line, then one "
    "sentence of justification.")


def _run_claude(prompt, cfg, model, vault=None, wd=None):
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "8000"
    if vault:
        env["ENGRAM_VAULT_PATH"] = vault
        env["ENGRAM_CHUNKS_DIR"] = os.path.join(wd, "chunks")
        os.makedirs(env["ENGRAM_CHUNKS_DIR"], exist_ok=True)
        env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", _slug(wd))
    args = ["claude", "-p", prompt, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=(wd or cfg), env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = {}
        if (out.get("is_error") or not out) and (out.get("total_cost_usd", 0) or 0) < 0.02:
            continue
        break
    return out


def _judge(task, expected_c, answer, judge_cfg):
    out = _run_claude(JUDGE_TMPL.format(task=task, C=expected_c, answer=answer or "(no answer)"),
                      judge_cfg, "sonnet")
    verdict = (out.get("result") or "").strip().upper()
    return verdict.startswith("HIT"), out.get("total_cost_usd", 0) or 0


def run_one(kind, arm, cold_cfg, warm_cfg, idx):
    fx = synth_fixtures.FIXTURES[kind]
    wd = tempfile.mkdtemp(prefix=f"{kind}-{arm}-{idx}-", dir=os.path.join(ROOT, "ws"))
    if arm == "cold":
        prompt = fx["task"].replace("Using ONLY your recalled memory, ", "").replace(
            "Using ONLY your recalled memory of how we solved a similar problem elsewhere, ", "")
        out = _run_claude(prompt, cold_cfg, "opus", wd=wd)
    else:  # warm-only
        vault = os.path.join(wd, "vault")
        synth_fixtures.build(kind, vault)
        out = _run_claude(RECALL_PREFIX + fx["task"], warm_cfg, "opus", vault=vault, wd=wd)
    answer = out.get("result") or ""
    hit, jcost = _judge(fx["task"], fx["C"], answer, cold_cfg)
    return {"kind": kind, "arm": arm, "idx": idx, "hit": hit,
            "cost": (out.get("total_cost_usd", 0) or 0) + jcost, "answer": answer[:200]}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--n", type=int, default=6)        # warm-only n per fixture
    ap.add_argument("--cold-n", type=int, default=3)
    ap.add_argument("--workers", type=int, default=4)
    a = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cold_cfg = os.path.join(ROOT, "cold-cfg"); build_cold_cfg(cold_cfg)
    warm_cfg = os.path.join(ROOT, "warm-cfg"); build_warm_cfg(warm_cfg)

    jobs = []
    for kind in KINDS:
        jobs += [(kind, "cold", i) for i in range(a.cold_n)]
        jobs += [(kind, "warm-only", i) for i in range(a.n)]
    print(f"synth RED: {KINDS} | cold n={a.cold_n}, warm-only n={a.n} = {len(jobs)} trials")

    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, k, arm, cold_cfg, warm_cfg, i): (k, arm, i) for k, arm, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [{r['kind']:15} {r['arm']:10} #{r['idx']}] hit={r['hit']} ${r['cost']:.2f}")

    print("\n=== SYNTH RED — C6 emergent-synthesis hit rate (per fixture) ===")
    print(f"{'fixture':16} {'cold':>10} {'warm-only':>12}")
    for kind in KINDS:
        cold = [r for r in results if r["kind"] == kind and r["arm"] == "cold"]
        warm = [r for r in results if r["kind"] == kind and r["arm"] == "warm-only"]
        ch = sum(r["hit"] for r in cold); wh = sum(r["hit"] for r in warm)
        print(f"{kind:16} {f'{ch}/{len(cold)}':>10} {f'{wh}/{len(warm)}':>12}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    print("READ: if warm-only ~= full, the synthesis step is REDUNDANT (stop). If warm-only leaves "
          "headroom (and cold ~0), there is something for a synthesis step to add.")
    json.dump(results, open(os.path.join(ROOT, "synth-red-results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
