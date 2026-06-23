"""Compounding DEPTH eval — does a DEEP emergent-synthesis ladder break no-persist?

The JOIN ladder (compound_fixtures.build_ladder) chains genuine emergent compositions:
  L1: A+B -> C1 ; L2: C1+D2 -> C2 ; L3: C2+D3 -> C3 ; L4: C3+D4 -> C4.
The depth-k question asks for Ck. no-persist must derive C1..Ck from raw; persist stores C1..C{k-1}
(oracle) so it does only the last hop. If re-deriving the deep stack drops a step, no-persist falls
while persist holds -> persistence headroom.

Usage: python3 compound_depth_eval.py [--depths 3,4] [--n 6] [--scatter 0] [--workers 5]
"""
import argparse
import concurrent.futures as cf
import json
import os
import sys
import tempfile
import time
import subprocess

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import compound_fixtures as cfix
from compound_eval import _run, JUDGE_TMPL
from run import MODELS
from wrun import build_warm_cfg, _slug, RECALL_PREFIX

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/compound-depth")


def run_one(depth, arm, scatter, warm_cfg, judge_cfg, idx, tag):
    wd = tempfile.mkdtemp(prefix=f"d{depth}-{tag}-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault")
    spec = cfix.build_ladder(depth, persist=(arm == "persist"), dst=vault, scatter=scatter)
    out = _run(RECALL_PREFIX + spec["task"], warm_cfg, "opus", vault=vault, wd=wd)
    answer = out.get("result") or ""
    j = _run(JUDGE_TMPL.format(task=spec["task"], E=spec["E"], answer=answer or "(none)"), judge_cfg, "sonnet")
    hit = (j.get("result") or "").strip().upper().startswith("HIT")
    return {"depth": depth, "arm": arm, "tag": tag, "idx": idx, "hit": hit,
            "cost": (out.get("total_cost_usd", 0) or 0) + (j.get("total_cost_usd", 0) or 0),
            "answer": answer[:220]}


def _rate(results, depth, tag):
    v = [r for r in results if r["depth"] == depth and r["tag"] == tag]
    return sum(r["hit"] for r in v), len(v)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--depths", default="3,4")
    ap.add_argument("--n", type=int, default=6)
    ap.add_argument("--scatter", type=int, default=0)
    ap.add_argument("--workers", type=int, default=5)
    a = ap.parse_args()
    depths = [int(x) for x in a.depths.split(",")]

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    warm_cfg = os.path.join(ROOT, "warm-cfg"); build_warm_cfg(warm_cfg)
    judge_cfg = os.path.join(ROOT, "judge-cfg"); build_warm_cfg(judge_cfg)

    jobs = []
    for d in depths:
        jobs += [(d, "no-persist", i, "nopersist") for i in range(a.n)]
        jobs += [(d, "persist", i, "persist") for i in range(a.n)]
        jobs += [(d, "no-persist", i, "nopersist2") for i in range(a.n)]   # noise floor
    print(f"compound DEPTH RED (join ladder): depths={depths} scatter={a.scatter} n={a.n} = {len(jobs)} trials")

    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, d, arm, a.scatter, warm_cfg, judge_cfg, i, tag): (d, tag, i)
                for d, arm, i, tag in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [d{r['depth']} {r['tag']:11} #{r['idx']}] hit={r['hit']} ${r['cost']:.2f}")

    print(f"\n=== COMPOUND DEPTH RED — join ladder hit rate (scatter={a.scatter}) ===")
    print(f"{'depth':>6} {'no-persist':>12} {'persist':>12} {'Δ(pp)':>7} {'noise(pp)':>10}")
    for d in depths:
        nh, nn = _rate(results, d, "nopersist"); ph, pn = _rate(results, d, "persist")
        b2h, b2n = _rate(results, d, "nopersist2")
        npr = 100 * nh / nn if nn else 0
        ppr = 100 * ph / pn if pn else 0
        floor = abs(npr - (100 * b2h / b2n if b2n else 0))
        print(f"{d:>6} {f'{nh}/{nn} ({npr:.0f}%)':>12} {f'{ph}/{pn} ({ppr:.0f}%)':>12} "
              f"{ppr - npr:>+6.0f} {floor:>9.0f}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    json.dump(results, open(os.path.join(ROOT, "compound-depth-results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
