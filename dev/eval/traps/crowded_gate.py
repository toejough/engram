"""Crowded-vault capability orchestrator.

Two tiers:
  * Tier-1 (free, local): for each cosine axis (C3/C4i/C6) sweep crowd size, run the real
    multi-phrase `engram query`, and find the break point — the smallest crowd where a load-bearing
    target stops surfacing OR drops below the top-RANK_THRESHOLD. That chosen crowd `B` guides where
    Tier-2 spends. C5 is recency-invariant (its target surfaces by being newest, not by cosine), so
    it skips the sweep and is checked only in Tier-2 at a fixed heavy crowd.
  * Tier-2 (LLM spend): run each axis's warm harness at `--crowd B` and a heavier stress level,
    normalize + score via gate_verdict, and compare the crowded-warm pass rate against the
    un-crowded toy baseline (and against cold, which is ~0 for these traps).

Pure helpers (break_point, degradation) are unit-tested. Fails loud on any harness error or missing
output. Run `--tier1-only` for the free sweep; bare for the full LLM run.
"""
import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import crowd
import c4_idio
import gate_verdict as gv
import reasoning_recall_eval as rr
import retrieval_probe
import seed_c3

TRAPS = os.path.dirname(os.path.abspath(__file__))

RANK_THRESHOLD = 10                       # a target worse than top-10 counts as "buried"
LEVELS = [0, 10, 30, 50, 100, 200, 400]   # crowd sizes swept in Tier-1
HEAVY_FALLBACK = 200                       # used when a cosine axis never breaks
C5_HEAVY = 200                             # recency axis: single heavy Tier-2 crowd

# Crowd vocab bias per axis (mirrors each harness's --crowd injection terms) so the crowd is a
# realistic on-topic competitor, not off-topic noise.
AXIS_VOCAB = {
    "C3": ["http", "error", "test", "color", "Go"],
    "C4i": ["error", "cfgload", "marker", "Go"],
    "C6": ["error", "reasoning", "memory"],
}

REPS = {"C3": 5, "C4i": 5, "C5": 5, "C6": 4}


def break_point(sweep, rank_threshold=RANK_THRESHOLD):
    """Smallest n in sweep where a target stops surfacing OR worst_rank exceeds rank_threshold.
    None if the axis never breaks (caller falls back to HEAVY_FALLBACK)."""
    for row in sweep:
        worst = row.get("worst_rank")
        if not row["all_surfaced"] or (worst is not None and worst > rank_threshold):
            return row["n"]
    return None


def degradation(crowded_axis, toy_axis):
    """passed-count delta crowded-minus-toy. A |delta| <= 1 is within the n=5 warm-vs-warm noise
    floor (underpowered), not a real change."""
    delta = crowded_axis["passed"] - toy_axis["passed"]
    if abs(delta) <= 1:
        note = "within noise — underpowered"
    elif delta < 0:
        note = "degraded beyond noise"
    else:
        note = "improved beyond noise"
    return {"delta": delta, "note": note}


def _seed_tier1_base(axis, temp):
    """Seed an axis's load-bearing base note(s) into a temp vault (no harness side effects)."""
    if axis == "C3":
        seed_c3.seed(temp)
    elif axis == "C4i":
        # Seed the warm-XXp base (X marker + superseding X') into the temp vault. seed_into is the
        # single source of that note content — NOT seed_vaults, which writes to hardcoded VAULTS.
        c4_idio.seed_into(temp)
    elif axis == "C6":
        # Seed all 4 premise notes (both abduction cases) into one combined probe vault.
        for case in ("abduction-diag", "abduction-badge"):
            for note in rr.CASES[case]["notes"]:
                rr._learn(temp, *note)
    else:
        raise ValueError(f"no Tier-1 base seeding for axis {axis!r}")


def tier1_sweep(axis):
    """Crowd-size sweep of retrieval precision for a cosine axis. C5 is recency-invariant."""
    if axis == "C5":
        return [{"n": C5_HEAVY, "note": "recency-invariant"}]
    notes = crowd.load_real_notes(crowd.real_vault())
    curve = []
    for n in LEVELS:
        temp = tempfile.mkdtemp(prefix=f"tier1-{axis}-{n}-")
        try:
            _seed_tier1_base(axis, temp)
            if n > 0:
                variants = crowd.make_variants(notes, n, seed=crowd.SEED,
                                               vocab_terms=AXIS_VOCAB[axis])
                crowd.seed_into(temp, variants)
            res = retrieval_probe.probe(temp, axis)
            curve.append({"n": n, "all_surfaced": res["all_surfaced"], "worst_rank": res["worst_rank"]})
        finally:
            shutil.rmtree(temp, ignore_errors=True)
    return curve


def _run_harness(axis, reps, crowd_n, workers):
    """Run one axis's warm harness at a given crowd size and return its scored axis_verdict.
    Fails loud on a non-zero exit or a missing output JSON."""
    root = tempfile.mkdtemp(prefix=f"crowded-{axis}-{crowd_n}-")
    env = {**os.environ, "TRAPS_ROOT": root}
    if axis == "C3":
        vault = os.path.join(root, "vault")
        seed_c3.seed(vault)
        cmd = ["python3", "wrun.py", "--vault", vault, "--n", str(reps),
               "--crowd", str(crowd_n), "--workers", str(workers)]
        out_file = "warm-results.json"
    elif axis == "C4i":
        cmd = ["python3", "c4_idio.py", "--arms", "warm-XXp", "--n", str(reps),
               "--crowd", str(crowd_n), "--workers", str(workers)]
        out_file = "c4-idio-results.json"
    elif axis == "C5":
        # Seed first: build the crowded chunk index (crowd before R, R newest), then run warm.
        seed = subprocess.run(["python3", "seed_c5.py", "--crowd", str(crowd_n)],
                              cwd=TRAPS, env=env)
        if seed.returncode != 0:
            sys.exit(f"CROWDED GATE ABORT: C5 seed (crowd={crowd_n}) exited {seed.returncode}")
        cmd = ["python3", "c5.py", "--arms", "warm", "--n", str(reps), "--workers", str(workers)]
        out_file = "c5-results.json"
    elif axis == "C6":
        cmd = ["python3", "c6_clean.py", "--arm", "warm", "--n", str(reps),
               "--crowd", str(crowd_n), "--workers", str(workers)]
        out_file = "c6-warm.json"
    else:
        raise ValueError(axis)
    result = subprocess.run(cmd, cwd=TRAPS, env=env)
    if result.returncode != 0:
        sys.exit(f"CROWDED GATE ABORT: {axis} harness (crowd={crowd_n}) exited {result.returncode}")
    path = os.path.join(root, out_file)
    if not os.path.exists(path):
        sys.exit(f"CROWDED GATE ABORT: {axis} produced no {out_file} (no silent pass)")
    rows = json.load(open(path))
    trials = gv.normalize(axis, rows)
    return gv.axis_verdict(trials, bar=len([t for t in trials if not t["contaminated"]]))


def run_tier1(cosine_axes):
    """Run the free Tier-1 sweep and choose a crowd level per axis."""
    sweeps, chosen = {}, {}
    print("=== Tier-1 retrieval-precision sweep (free) ===")
    for axis in cosine_axes:
        sweep = tier1_sweep(axis)
        sweeps[axis] = sweep
        bp = break_point(sweep)
        chosen[axis] = bp if bp is not None else HEAVY_FALLBACK
        print(f"\n{axis}:")
        for row in sweep:
            print(f"  n={row['n']:>4}  all_surfaced={row.get('all_surfaced')}  "
                  f"worst_rank={row.get('worst_rank')}")
        print(f"  break_point={bp}  chosen_crowd={chosen[axis]}")
    chosen["C5"] = C5_HEAVY
    return sweeps, chosen


def run_tier2(chosen, workers):
    """Run the Tier-2 applied check at chosen + heavier crowd and score vs the toy baseline."""
    rows = []
    for axis in ["C3", "C4i", "C5", "C6"]:
        base = chosen[axis]
        heavier = min(2 * base, 400)
        crowd_levels = [base] if axis == "C5" else sorted({base, heavier})
        toy = _run_harness(axis, REPS[axis], 0, workers)
        for crowd_n in crowd_levels:
            crowded = _run_harness(axis, REPS[axis], crowd_n, workers)
            deg = degradation(crowded, toy)
            rows.append({"axis": axis, "break_n": base, "crowd": crowd_n,
                         "crowded_pass": crowded["passed"], "crowded_valid": crowded["valid"],
                         "toy_pass": toy["passed"], "toy_valid": toy["valid"],
                         "delta": deg["delta"], "note": deg["note"],
                         "verdict": crowded["status"],
                         "beats_cold": crowded["passed"] > 0})  # cold ~0 for these traps
    return rows


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--tier1-only", action="store_true",
                    help="run the free retrieval sweep + chosen crowd, then exit before any LLM spend")
    ap.add_argument("--workers", type=int, default=6)
    a = ap.parse_args()

    cosine_axes = ["C3", "C4i", "C6"]
    sweeps, chosen = run_tier1(cosine_axes)

    if a.tier1_only:
        print(f"\nchosen crowd per axis: {chosen}")
        print("--tier1-only: stopping before any LLM spend.")
        return

    print(f"\n=== Tier-2 applied check (LLM spend ~$0.45/trial) — chosen crowd {chosen} ===")
    rows = run_tier2(chosen, a.workers)

    print("\n=== CROWDED CAPABILITY TABLE ===")
    print(f"  {'axis':5} {'break_n':>7} {'crowd':>6} {'crowded_pass':>12} {'toy_pass':>8} "
          f"{'delta':>6} {'verdict':>12}")
    for r in rows:
        print(f"  {r['axis']:5} {r['break_n']:>7} {r['crowd']:>6} "
              f"{str(r['crowded_pass']) + '/' + str(r['crowded_valid']):>12} "
              f"{str(r['toy_pass']) + '/' + str(r['toy_valid']):>8} {r['delta']:>6} "
              f"{r['verdict']:>12}  (beats_cold={r['beats_cold']}; {r['note']})")

    result = {"sweeps": sweeps, "chosen": chosen, "rows": rows}
    out = os.path.join(TRAPS, "crowded-verdict.json")
    json.dump(result, open(out, "w"), indent=1)
    print(f"\nwrote {out}")


if __name__ == "__main__":
    main()
