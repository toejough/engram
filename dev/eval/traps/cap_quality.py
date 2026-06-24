"""Option 1 Task 3 — warm eval-arm quality under the content cap.

Injects ENGRAM_CONTENT_BUDGET into a warm harness (env propagates through the harness into the
claude subprocess that runs /recall -> engram query). Runs one criterion at one budget, parses the
harness's own summary line, appends a JSON record. C5 is the canary (R is a recency chunk); C3/C4/C6
are note-driven (notes are never capped) so they should be cap-invariant — confirmed, not assumed.

  python3 cap_quality.py --crit c5 --budget 8 --n 3
"""
import argparse
import json
import os
import re
import subprocess

# crit -> (argv, TRAPS_ROOT, summary regex -> (numerator, denominator) for the warm pass rate)
CRIT = {
    "c5": (["python3", "c5.py", "--arms", "warm"], "/tmp/c5",
           r"warm\s+valid=\d+/\d+\s+C5a surfaced=\d+/\d+\s+C5b honored=(\d+)/(\d+)"),
    "c4": (["python3", "c4_idio.py", "--arms", "warm-XXp"], "/tmp/c4-idio",
           r"warm-XXp\s+valid=\d+/\d+\s+follows_x=\d+\s+supersession_correct=(\d+)"),
    "c6": (["python3", "c6_clean.py", "--arm", "warm"], "/tmp/c6-clean",
           r"abduction-badge\s+warm:\s+(\d+)/(\d+)"),
}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--crit", required=True, choices=list(CRIT))
    ap.add_argument("--budget", type=int, required=True)
    ap.add_argument("--n", type=int, default=3)
    a = ap.parse_args()

    argv, root, rx = CRIT[a.crit]
    env = dict(os.environ)
    env["TRAPS_ROOT"] = root
    # Pass the cap through unconditionally: -1 = unlimited baseline (0 now resolves to the default 15).
    env["ENGRAM_CONTENT_BUDGET"] = str(a.budget)
    cmd = argv + ["--n", str(a.n)]
    print(f"[{a.crit} budget={a.budget} n={a.n}] {' '.join(cmd)}")
    r = subprocess.run(cmd, env=env, capture_output=True, text=True, cwd=os.path.dirname(__file__) or ".")
    print(r.stdout[-1500:])
    if r.returncode != 0:
        print("STDERR:", r.stderr[-800:])
    m = re.search(rx, r.stdout)
    rec = {"crit": a.crit, "budget": a.budget, "n": a.n,
           "pass": int(m.group(1)) if m else None,
           "denom": int(m.group(2)) if (m and m.lastindex and m.lastindex >= 2) else a.n,
           "raw": m.group(0) if m else "PARSE-FAIL"}
    with open("/tmp/cap_quality.jsonl", "a") as f:
        f.write(json.dumps(rec) + "\n")
    print("RECORD:", json.dumps(rec))


if __name__ == "__main__":
    main()
