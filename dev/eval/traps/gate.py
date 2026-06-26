"""Trap regression gate — re-run the warm C3/C4-idio/C5/C6 trap harnesses and emit a single
GREEN/RED/INCONCLUSIVE verdict, so cost/usage optimization can't silently erode a verified win.

Each axis runs in its own temp TRAPS_ROOT (per-axis isolation), shells out to the existing warm
harness (no trial logic reimplemented), fails loud on a non-zero exit or a missing output JSON,
then normalizes + scores via gate_verdict. Exit 0 only on GREEN, so it works as a pre-merge check.

Usage: python3 gate.py --tier smoke|full [--workers N]
"""
import argparse
import json, os, subprocess, sys, tempfile
import seed_c3, gate_verdict as gv

TRAPS = os.path.dirname(os.path.abspath(__file__))

SMOKE = {"C3": 1, "C4i": 1, "C5": 1, "C6": 1}
FULL = {"C3": 5, "C4i": 5, "C5": 5, "C6": 4}


def run_axis(axis, reps, workers):
    root = tempfile.mkdtemp(prefix=f"gate-{axis}-")
    env = {**os.environ, "TRAPS_ROOT": root}
    if axis == "C3":
        vault = os.path.join(root, "vault")
        seed_c3.seed(vault)
        cmd = ["python3", "wrun.py", "--vault", vault, "--n", str(reps), "--workers", str(workers)]
        out_file = "warm-results.json"
    elif axis == "C4i":
        # Only warm-XXp is scored (the supersession win); skip the cold/warm-X baseline arms.
        cmd = ["python3", "c4_idio.py", "--arms", "warm-XXp", "--n", str(reps), "--workers", str(workers)]
        out_file = "c4-idio-results.json"
    elif axis == "C5":
        subprocess.run(["python3", "seed_c5.py"], cwd=TRAPS, env=env, check=True)
        # Only the warm arm is scored (cold is the baseline that should fail); skip running it.
        cmd = ["python3", "c5.py", "--arms", "warm", "--n", str(reps), "--workers", str(workers)]
        out_file = "c5-results.json"
    elif axis == "C6":
        cmd = ["python3", "c6_clean.py", "--arm", "warm", "--n", str(reps), "--workers", str(workers)]
        out_file = "c6-warm.json"
    else:
        raise ValueError(axis)
    r = subprocess.run(cmd, cwd=TRAPS, env=env)
    if r.returncode != 0:
        sys.exit(f"GATE ABORT: {axis} harness exited {r.returncode}")
    path = os.path.join(root, out_file)
    if not os.path.exists(path):
        sys.exit(f"GATE ABORT: {axis} produced no {out_file} (no silent pass)")
    rows = json.load(open(path))
    trials = gv.normalize(axis, rows)
    return gv.axis_verdict(trials, bar=len([t for t in trials if not t["contaminated"]]))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--tier", required=True, choices=["smoke", "full"])
    ap.add_argument("--workers", type=int, default=6)
    a = ap.parse_args()

    reps_by_axis = {"smoke": SMOKE, "full": FULL}[a.tier]
    print(f"trap regression gate — tier={a.tier} reps={reps_by_axis}")
    axes = {}
    for axis, reps in reps_by_axis.items():
        print(f"\n=== running {axis} (reps={reps}) ===", flush=True)
        axes[axis] = run_axis(axis, reps, a.workers)

    result = gv.gate_verdict(axes)
    print(f"\n=== GATE VERDICT (tier={a.tier}) ===")
    print(f"  {'axis':5} {'valid':>5} {'contam':>7} {'passed':>6} {'bar':>4} {'status':>12}")
    for axis, v in result["axes"].items():
        print(f"  {axis:5} {v['valid']:>5} {v['contaminated']:>7} {v['passed']:>6} "
              f"{v['bar']:>4} {v['status']:>12}")
    print(f"\noverall verdict: {result['verdict']}")

    out_path = os.path.join(TRAPS, "gate-verdict.json")
    json.dump(result, open(out_path, "w"), indent=1)
    print(f"wrote {out_path}")
    sys.exit(0 if result["verdict"] == "GREEN" else 1)


if __name__ == "__main__":
    main()
