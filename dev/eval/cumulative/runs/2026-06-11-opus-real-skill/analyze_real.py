#!/usr/bin/env python3
"""Per-(regime, app, trial) breakdown of the opus real-skill run + chain integrity check."""
import json, os, sys
from statistics import mean

RESULTS = "/tmp/cummatrix-real/results"
REGIMES = ["cold", "real.lazy", "real.eager"]
APPS = ["app1", "app2", "app3"]
TRIALS = [1, 2, 3, 4, 5]


def load(model, trial, app, regime):
    f = f"{RESULTS}/{model}-t{trial}-{app}-{regime}-build.json"
    if not os.path.exists(f):
        return None
    try:
        return json.load(open(f))
    except Exception:
        return None


def vault_ok(d):
    vi = d.get("vault_in", "")
    if not vi or vi == "none":
        return "seed-empty"   # app1 always recalls empty seed — expected
    return "OK" if os.path.isdir(vi) else "MISSING"


def main():
    model = "opus"
    print(f"=== {model} real-skill run: per-(regime, app, trial) ===\n")
    print(f"{'regime':<12}{'app':<6}" + "".join(f"t{t:<5}" for t in TRIALS) + "  mean  notes")
    integrity = []
    per_app_means = {}
    for regime in REGIMES:
        for app in APPS:
            row, fails, flags = [], [], []
            for t in TRIALS:
                d = load(model, t, app, regime)
                if d is None:
                    row.append("  --")
                    flags.append(f"t{t}:MISSING-CELL")
                    continue
                cf = d.get("round1_convention_fails")
                row.append(f"{cf:>4}" if cf is not None else "  ?")
                if cf is not None:
                    fails.append(cf)
                vk = vault_ok(d)
                if vk == "MISSING":
                    flags.append(f"t{t}:VAULT-MISSING")
                rf = d.get("recall_fired")
                if regime != "cold" and rf != 1:
                    flags.append(f"t{t}:recall_fired={rf}")
            m = f"{mean(fails):>5.1f}" if fails else "   --"
            per_app_means[(regime, app)] = mean(fails) if fails else None
            note = " ".join(flags)
            print(f"{regime:<12}{app:<6}" + "".join(f"{c:<6}" for c in row) + f" {m}  {note}")
            if flags:
                integrity.append((regime, app, flags))
        print()

    print("=== Memory-active decomposition (cold vs lazy vs eager) ===")
    print("app1 = empty seed (memory OFF by construction); app2/app3 = memory ACTIVE\n")
    for regime in REGIMES:
        a1 = per_app_means.get((regime, "app1"))
        a2 = per_app_means.get((regime, "app2"))
        a3 = per_app_means.get((regime, "app3"))
        active = [x for x in (a2, a3) if x is not None]
        active_sum = sum(active) if active else None
        print(f"  {regime:<12} app1={a1}  app2={a2}  app3={a3}  | mem-active(app2+app3)={active_sum}")

    print("\n=== Chain integrity flags ===")
    if not integrity:
        print("  (clean — no missing cells or broken vaults)")
    for regime, app, flags in integrity:
        print(f"  {regime} {app}: {flags}")


if __name__ == "__main__":
    main()
