"""#661 Phase 2 SMOKE — C6 (abduction, synthesis-sensitive) glance vs deep apply-verdict.

C6 is where note 72 says retrieval-holds is uninformative (the bottleneck is synthesis), so it's the
highest-value axis to check whether dropping glance's coverage-judge/crystallization hurts application.
rr.run_one seeds the case's premise notes, runs the cfg's recall + opus answer, sonnet-judges HIT/MISS.
"""
import sys, os, shutil, json
TRAPS = "/Users/joe/repos/personal/engram/dev/eval/traps"
sys.path.insert(0, TRAPS)
import reasoning_recall_eval as rr
from wrun import build_warm_cfg

SCRATCH = "/private/tmp/claude-501/-Users-joe-repos-personal-engram/95570838-0d05-483c-95e7-fe004909b499/scratchpad"
ROOT = os.path.join(SCRATCH, "c6smoke")
shutil.rmtree(ROOT, ignore_errors=True)
os.makedirs(os.path.join(ROOT, "ws"))
rr.ROOT = ROOT  # redirect rr's workspace root

DEEP = os.path.join(SCRATCH, "cfgs", "deep-cfg")
GLANCE = os.path.join(SCRATCH, "cfgs", "glance-cfg")
judge = os.path.join(ROOT, "judge-cfg"); build_warm_cfg(judge)

REPS = int(os.environ.get("REPS", "1"))
results = []
for case in ("abduction-diag", "abduction-badge"):
    for arm, cfg in (("deep", DEEP), ("glance", GLANCE)):
        for rep in range(REPS):
            r = rr.run_one(case, arm, cfg, judge, rep)
            results.append(r)
            print(f"{case:18} {arm:6} rep={rep} hit={r['hit']} ${r['cost']:.2f}", flush=True)

print("\n=== C6 smoke: HIT rate (abduction composed correctly) ===")
for case in ("abduction-diag", "abduction-badge"):
    for arm in ("deep", "glance"):
        rs = [x for x in results if x["case"] == case and x["arm"] == arm]
        hits = sum(1 for x in rs if x["hit"])
        print(f"  {case:18} {arm:6}: {hits}/{len(rs)} HIT  ${sum(x['cost'] for x in rs):.2f}")
with open(os.path.join(SCRATCH, "c6_smoke_results.json"), "w") as f:
    json.dump(results, f, indent=2)
print(f"\ntotal $ {sum(x['cost'] for x in results):.2f}")
