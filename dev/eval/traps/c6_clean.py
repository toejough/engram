"""C6 (clean) — does MEMORY let opus produce an emergent conclusion cold opus cannot?

Replaces the suspect synth_eval (leading prompts). Uses the vetted, non-leading abduction cases from
reasoning_recall_eval, with proper ISOLATION (cold runs in its own root with NO vaults on disk, so a
wandering cold agent can't read a sibling's fixture — the contamination bug from earlier).

  warm : premises in engram memory; invoke /recall; open-ended task -> reason to the conclusion.
  cold : NO memory, NO recall, isolated empty cwd, same open-ended task (it never saw the facts).

Independent sonnet judge on reaching the emergent conclusion. Run cold and warm SEPARATELY
(different TRAPS_ROOT) so no warm vaults exist while cold runs.

Usage: python3 c6_clean.py --arm warm --n 4   (then)   python3 c6_clean.py --arm cold --n 4
"""
import argparse, concurrent.futures as cf, json, os, sys, tempfile
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import reasoning_recall_eval as rr
from run import build_cold_cfg
from wrun import build_warm_cfg

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/c6-clean")
CASES = ["abduction-diag", "abduction-badge"]


def warm_one(case, cfg, judge_cfg, idx, model="opus"):
    spec = rr.CASES[case]
    wd = tempfile.mkdtemp(prefix=f"{case}-warm-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault"); os.makedirs(vault)
    for n in spec["notes"]:
        rr._learn(vault, *n)
    out = rr._run(rr.NEUTRAL_PREFIX + spec["task"], cfg, model, vault=vault, wd=wd)
    return _judge(case, out, judge_cfg, "warm", idx)


def cold_one(case, cfg, judge_cfg, idx, model="opus"):
    spec = rr.CASES[case]
    wd = tempfile.mkdtemp(prefix=f"{case}-cold-{idx}-", dir=os.path.join(ROOT, "ws"))  # ROOT has NO vaults
    out = rr._run(spec["task"], cfg, model, wd=wd)   # no vault, no recall prefix
    return _judge(case, out, judge_cfg, "cold", idx)


def _judge(case, out, judge_cfg, arm, idx):
    spec = rr.CASES[case]; answer = out.get("result") or ""
    j = rr._run(rr.JUDGE.format(task=spec["task"], E=spec["E"], answer=answer or "(none)"), judge_cfg, "sonnet")
    hit = (j.get("result") or "").strip().upper().startswith("HIT")
    return {"case": case, "arm": arm, "idx": idx, "hit": hit,
            "cost": (out.get("total_cost_usd", 0) or 0) + (j.get("total_cost_usd", 0) or 0), "answer": answer}


def main():
    ap = argparse.ArgumentParser(); ap.add_argument("--arm", required=True, choices=["warm", "cold"])
    ap.add_argument("--n", type=int, default=4); ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--model", default="opus")  # model under test; judge stays sonnet (see _judge)
    a = ap.parse_args()
    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    judge_cfg = os.path.join(ROOT, "judge-cfg"); build_warm_cfg(judge_cfg)
    if a.arm == "warm":
        cfg = os.path.join(ROOT, "warm-cfg"); build_warm_cfg(cfg); fn = warm_one
    else:
        cfg = os.path.join(ROOT, "cold-cfg"); build_cold_cfg(cfg); fn = cold_one
    jobs = [(c, i) for c in CASES for i in range(a.n)]
    print(f"C6-clean {a.arm}: cases={CASES} n={a.n} = {len(jobs)} trials")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(fn, c, cfg, judge_cfg, i, a.model): (c, i) for c, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [{r['case']:16} {r['arm']} #{r['idx']}] hit={r['hit']} ${r['cost']:.2f}")
    for c in CASES:
        v = [r for r in results if r["case"] == c]; h = sum(r["hit"] for r in v)
        print(f"  {c:16} {a.arm}: {h}/{len(v)}")
    json.dump(results, open(os.path.join(ROOT, f"c6-{a.arm}.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
