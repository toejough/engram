#!/usr/bin/env python3
"""Cost inversion, L2 counts, noise floor, and lazy-vs-eager significance for the opus real run."""
import json, os
from statistics import mean, pstdev

RESULTS = "/tmp/cummatrix-real/results"
REGIMES = ["cold", "real.lazy", "real.eager"]
APPS = ["app1", "app2", "app3"]
TRIALS = [1, 2, 3, 4, 5]


def load(t, app, regime):
    f = f"{RESULTS}/opus-t{t}-{app}-{regime}-build.json"
    return json.load(open(f)) if os.path.exists(f) else None


def cell_cost(d):
    learn_nested = (d.get("learn") or {}).get("cost") or 0
    return (d.get("build_cost") or 0) + (d.get("learn_cost") or 0) + learn_nested


print("=== COST per regime (build + in-session learn), summed across 3 apps, per trial ===\n")
print(f"{'regime':<12}" + "".join(f"t{t:<6}" for t in TRIALS) + "  mean/trial-chain")
regime_cost = {}
for regime in REGIMES:
    per_trial = []
    for t in TRIALS:
        c = sum(cell_cost(load(t, app, regime)) for app in APPS if load(t, app, regime))
        per_trial.append(c)
    regime_cost[regime] = mean(per_trial)
    print(f"{regime:<12}" + "".join(f"${c:<5.1f}" for c in per_trial) + f"  ${mean(per_trial):.2f}")

print(f"\n  cost inversion: cold ${regime_cost['cold']:.2f}/chain  vs  "
      f"lazy ${regime_cost['real.lazy']:.2f}  eager ${regime_cost['real.eager']:.2f}")
print(f"  warm/cold ratio: lazy {regime_cost['real.lazy']/regime_cost['cold']:.1f}x  "
      f"eager {regime_cost['real.eager']/regime_cost['cold']:.1f}x")
print(f"  lazy vs eager: lazy is {(1-regime_cost['real.lazy']/regime_cost['real.eager'])*100:.0f}% cheaper")

print("\n=== L2/L1 notes in the FINAL accumulated vault, per regime, per trial ===")
print("NOTE: notes_by_tier is CUMULATIVE vault state (carries prior apps forward), NOT per-session")
print("writes — so the chain total is the TERMINAL app3 count, never the sum across apps.\n")
for regime in ["real.lazy", "real.eager"]:
    per_trial_l2 = []
    per_trial_l1 = []
    for t in TRIALS:
        d = load(t, "app3", regime)   # terminal app's cumulative vault = the whole chain's notes
        nbt = (d.get("learn") or {}).get("notes_by_tier") or {}
        per_trial_l2.append(nbt.get("L2", 0))
        per_trial_l1.append(nbt.get("L1", 0))
    print(f"  {regime:<12} L2/chain: {per_trial_l2} mean={mean(per_trial_l2):.1f}   "
          f"L1/chain: {per_trial_l1} mean={mean(per_trial_l1):.1f}")

print("\n=== NOISE FLOOR: app1 has NO memory in any regime (empty seed) — should be equal ===\n")
for regime in REGIMES:
    fails = [load(t, "app1", regime).get("round1_convention_fails") for t in TRIALS]
    print(f"  {regime:<12} app1 fails: {fails}  mean={mean(fails):.1f} sd={pstdev(fails):.1f}")
a1 = [mean([load(t, "app1", r).get("round1_convention_fails") for t in TRIALS]) for r in REGIMES]
print(f"\n  app1 across regimes (all memory-OFF): {[round(x,1) for x in a1]} -> spread={max(a1)-min(a1):.1f}")
print("  => differences <= this spread at per-app level are within run-to-run noise.")

print("\n=== LAZY vs EAGER on memory-active apps (app2+app3), per trial ===\n")
for regime in ["real.lazy", "real.eager"]:
    per_trial = []
    for t in TRIALS:
        s = sum(load(t, app, regime).get("round1_convention_fails") for app in ["app2", "app3"])
        per_trial.append(s)
    print(f"  {regime:<12} app2+app3 fails/trial: {per_trial}  mean={mean(per_trial):.1f} sd={pstdev(per_trial):.1f}")
lazy_pt = [sum(load(t, app, "real.lazy").get("round1_convention_fails") for app in ["app2","app3"]) for t in TRIALS]
eager_pt = [sum(load(t, app, "real.eager").get("round1_convention_fails") for app in ["app2","app3"]) for t in TRIALS]
diff = mean(eager_pt) - mean(lazy_pt)
pooled_sd = (pstdev(lazy_pt)**2/len(lazy_pt) + pstdev(eager_pt)**2/len(eager_pt))**0.5
print(f"\n  diff (eager-lazy) = {diff:.1f}  pooled SE = {pooled_sd:.2f}  -> ~{diff/pooled_sd:.1f} SE")
print("  (|diff/SE| < 2 => not distinguishable at this n)")
