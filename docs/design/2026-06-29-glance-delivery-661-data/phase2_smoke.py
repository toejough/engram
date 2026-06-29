"""#661 Phase 2 SMOKE (firmed) — glance vs deep apply-verdict on C6 (3 reps) + C3 (3 conventions x 2 reps)."""
import sys, os, shutil, json
TRAPS = "/Users/joe/repos/personal/engram/dev/eval/traps"
sys.path.insert(0, TRAPS)
import reasoning_recall_eval as rr
import traps as T
import seed_c3
import wrun
from wrun import build_warm_cfg

SCRATCH = "/private/tmp/claude-501/-Users-joe-repos-personal-engram/95570838-0d05-483c-95e7-fe004909b499/scratchpad"
DEEP = os.path.join(SCRATCH, "cfgs", "deep-cfg")
GLANCE = os.path.join(SCRATCH, "cfgs", "glance-cfg")
ARMS = (("deep", DEEP), ("glance", GLANCE))

# --- C6: abduction (synthesis-sensitive), 3 reps ---
ROOT6 = os.path.join(SCRATCH, "c6smoke2"); shutil.rmtree(ROOT6, ignore_errors=True); os.makedirs(os.path.join(ROOT6, "ws"))
rr.ROOT = ROOT6
judge = os.path.join(ROOT6, "judge-cfg"); build_warm_cfg(judge)
c6 = []
for case in ("abduction-diag", "abduction-badge"):
    for arm, cfg in ARMS:
        for rep in range(3):
            r = rr.run_one(case, arm, cfg, judge, rep)
            c6.append({"axis": "C6", "case": case, "arm": arm, "rep": rep, "ok": r["hit"], "cost": r["cost"]})
            print(f"C6 {case:16} {arm:6} rep={rep} hit={r['hit']} ${r['cost']:.2f}", flush=True)

# --- C3: conventions (most common), 3 traps x 2 reps ---
ROOT3 = os.path.join(SCRATCH, "c3smoke"); shutil.rmtree(ROOT3, ignore_errors=True)
os.makedirs(os.path.join(ROOT3, "ws")); os.makedirs(os.path.join(ROOT3, "chunks"))
wrun.ROOT = ROOT3
vault = os.path.join(ROOT3, "vault"); os.makedirs(vault); seed_c3.seed(vault)
c3 = []
for conv in ("req-with-context", "wrapped-error", "nil-guard-split"):
    for arm, cfg in ARMS:
        for rep in range(2):
            r = wrun.run_one(conv, T.TRAPS[conv], "opus", cfg, vault, rep)
            ok = r["verdict"] == "applied"
            c3.append({"axis": "C3", "case": conv, "arm": arm, "rep": rep, "ok": ok,
                       "verdict": r["verdict"], "recalled": r["recalled"], "cost": r["cost"]})
            print(f"C3 {conv:16} {arm:6} rep={rep} {r['verdict']:8} recall={r['recalled']} ${r['cost']:.2f}", flush=True)

allr = c6 + c3
print("\n=== SMOKE: glance vs deep apply rate ===")
for axis in ("C6", "C3"):
    for arm in ("deep", "glance"):
        rs = [x for x in allr if x["axis"] == axis and x["arm"] == arm]
        ok = sum(1 for x in rs if x["ok"])
        print(f"  {axis} {arm:6}: {ok}/{len(rs)}  ${sum(x['cost'] for x in rs):.2f}")
with open(os.path.join(SCRATCH, "smoke_combined_results.json"), "w") as f:
    json.dump(allr, f, indent=2)
print(f"\nTOTAL ${sum(x['cost'] for x in allr):.2f}")
print("SMOKE_DONE")
