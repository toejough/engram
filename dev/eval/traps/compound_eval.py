"""Compounding-eval RED — no-persist vs persist over 2-level EMERGENT synthesis ladders.

For each synthesis type, the level-2 task needs E = compose(C, D) where C = compose(A, B) is emergent.
  no-persist : vault {A,B,D} -> re-derive C, then compose E.
  persist    : vault {A,B,D,C*} -> C* stored -> compose E directly.
Independent sonnet judge rules HIT/MISS on whether the answer reaches E. Includes a noise-floor pass
(no-persist run twice). A persist Δ is real only if it exceeds the floor.

Usage: python3 compound_eval.py [--types join] [--n 6] [--scatter 0] [--workers 4]
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
import compound_fixtures as cfix
from run import MODELS
from wrun import build_warm_cfg, _slug, RECALL_PREFIX

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/compound-eval")

JUDGE_TMPL = (
    "You are grading whether an answer reaches a specific EMERGENT two-step conclusion.\n\n"
    "QUESTION:\n{task}\n\n"
    "REQUIRED EMERGENT CONCLUSION (the answer must reach this by combining the underlying facts, not "
    "recite one of them):\n{E}\n\n"
    "CANDIDATE ANSWER:\n{answer}\n\n"
    "Does the candidate answer clearly reach the required emergent conclusion (the full causal/transfer "
    "link, not just one half)? Reply 'HIT' or 'MISS' on the first line, then one sentence.")


def _run(prompt, cfg, model, vault=None, wd=None):
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "12000"
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


def run_one(stype, arm, scatter, warm_cfg, judge_cfg, idx, tag):
    wd = tempfile.mkdtemp(prefix=f"{stype}-{tag}-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault")
    spec = cfix.build(stype, persist=(arm == "persist"), dst=vault, scatter=scatter)
    out = _run(RECALL_PREFIX + spec["task"], warm_cfg, "opus", vault=vault, wd=wd)
    answer = out.get("result") or ""
    j = _run(JUDGE_TMPL.format(task=spec["task"], E=spec["E"], answer=answer or "(none)"), judge_cfg, "sonnet")
    hit = (j.get("result") or "").strip().upper().startswith("HIT")
    return {"stype": stype, "arm": arm, "tag": tag, "idx": idx, "hit": hit,
            "cost": (out.get("total_cost_usd", 0) or 0) + (j.get("total_cost_usd", 0) or 0),
            "answer": answer[:200]}


def _rate(results, stype, tag):
    v = [r for r in results if r["stype"] == stype and r["tag"] == tag]
    return sum(r["hit"] for r in v), len(v)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--types", default="join")
    ap.add_argument("--n", type=int, default=6)
    ap.add_argument("--scatter", type=int, default=0)
    ap.add_argument("--workers", type=int, default=4)
    a = ap.parse_args()
    types = a.types.split(",")

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    warm_cfg = os.path.join(ROOT, "warm-cfg"); build_warm_cfg(warm_cfg)
    judge_cfg = os.path.join(ROOT, "judge-cfg"); build_warm_cfg(judge_cfg)  # clean cfg for the judge

    jobs = []
    for t in types:
        jobs += [(t, "no-persist", i, "nopersist") for i in range(a.n)]
        jobs += [(t, "persist", i, "persist") for i in range(a.n)]
        jobs += [(t, "no-persist", i, "nopersist2") for i in range(a.n)]   # noise floor
    print(f"compound RED (2-level emergent): types={types} scatter={a.scatter} n={a.n} = {len(jobs)} trials")

    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, t, arm, a.scatter, warm_cfg, judge_cfg, i, tag): (t, tag, i)
                for t, arm, i, tag in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [{r['stype']:11} {r['tag']:11} #{r['idx']}] hit={r['hit']} ${r['cost']:.2f}")

    print(f"\n=== COMPOUND RED — 2-level emergent-synthesis hit rate (scatter={a.scatter}) ===")
    print(f"{'type':12} {'no-persist':>12} {'persist':>12} {'Δ(pp)':>7} {'noise(pp)':>10}")
    for t in types:
        nh, nn = _rate(results, t, "nopersist"); ph, pn = _rate(results, t, "persist")
        b2h, b2n = _rate(results, t, "nopersist2")
        npr = 100 * nh / nn if nn else 0
        ppr = 100 * ph / pn if pn else 0
        floor = abs(npr - (100 * b2h / b2n if b2n else 0))
        print(f"{t:12} {f'{nh}/{nn} ({npr:.0f}%)':>12} {f'{ph}/{pn} ({ppr:.0f}%)':>12} "
              f"{ppr - npr:>+6.0f} {floor:>9.0f}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    json.dump(results, open(os.path.join(ROOT, f"compound-red-{'-'.join(types)}.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
