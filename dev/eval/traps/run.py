"""Cold-confirm runner: run each candidate trap N times against COLD opus (no memory, no
CLAUDE.md, clean cfg) in an isolated /tmp workdir, apply the deterministic check, and report
which traps reproducibly fire (cold falls in >= THRESHOLD/N).

Usage:
  python3 run.py [--model opus] [--n 5] [--threshold 4] [--traps name1,name2] [--workers 6]
"""
import argparse, json, os, shutil, subprocess, sys, tempfile, time
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import traps as T

MODELS = {"opus": "claude-opus-4-8", "sonnet": "claude-sonnet-4-6", "haiku": "claude-haiku-4-5-20251001"}
KEYCHAIN = 'security find-generic-password -s "Claude Code-credentials" -w'
ROOT = os.environ.get("TRAPS_ROOT", "/tmp/opus-traps")


def build_cold_cfg(dst):
    """Clean CLAUDE_CONFIG_DIR: onboarding/oauth from the local install (history dropped),
    creds injected, NO CLAUDE.md, NO skills — a true cold/no-memory room."""
    shutil.rmtree(dst, ignore_errors=True)
    os.makedirs(dst, exist_ok=True)
    user_cfg = os.path.expanduser("~/.claude/.claude.json")
    base = {}
    if os.path.exists(user_cfg):
        try:
            base = json.load(open(user_cfg))
        except Exception:
            base = {}
    base["projects"] = {}
    json.dump(base, open(os.path.join(dst, ".claude.json"), "w"))
    subprocess.run(["bash", "-c", f'{KEYCHAIN} > {dst}/.credentials.json && chmod 600 {dst}/.credentials.json'],
                   capture_output=True)


def run_one(name, spec, model, cfg, idx):
    """One cold trial of a trap. Returns (verdict, cost, sid)."""
    wd = tempfile.mkdtemp(prefix=f"{name}-{idx}-", dir=os.path.join(ROOT, "ws"))
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    args = ["claude", "-p", spec["prompt"], "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = {}
        cost = out.get("total_cost_usd", 0) or 0
        is_err = out.get("is_error") or (not out)
        # transient: cheap error → retry
        if is_err and cost < 0.02:
            continue
        break
    verdict = spec["check"](wd)
    return {"trap": name, "idx": idx, "verdict": verdict, "cost": cost,
            "turns": out.get("num_turns"), "sid": out.get("session_id"), "wd": wd}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=5)
    ap.add_argument("--threshold", type=int, default=4)
    ap.add_argument("--traps", default="")
    ap.add_argument("--workers", type=int, default=6)
    a = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "cold-cfg")
    build_cold_cfg(cfg)

    names = a.traps.split(",") if a.traps else list(T.TRAPS)
    jobs = [(n, i) for n in names for i in range(a.n)]
    print(f"running {len(names)} traps × n={a.n} = {len(jobs)} cold {a.model} trials (workers={a.workers})")

    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, n, T.TRAPS[n], a.model, cfg, i): (n, i) for n, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result()
            results.append(r)
            print(f"  [{r['trap']:16} #{r['idx']}] {r['verdict']:8} ${r['cost']:.2f} turns={r['turns']}")

    # tally
    print("\n=== REPRODUCIBILITY (cold falls in = 'trap') ===")
    confirmed = []
    by = {}
    for r in results:
        by.setdefault(r["trap"], []).append(r["verdict"])
    spent = sum(r["cost"] for r in results)
    for n in names:
        v = by.get(n, [])
        trap = v.count("trap"); applied = v.count("applied"); nob = v.count("nobuild")
        valid = trap + applied
        status = "CONFIRMED" if trap >= a.threshold else ("saturated" if applied > trap else "weak/invalid")
        if trap >= a.threshold:
            confirmed.append(n)
        print(f"  {n:16} trap={trap}/{a.n} applied={applied} nobuild={nob}  -> {status}")
    print(f"\nCONFIRMED ({len(confirmed)}): {confirmed}")
    print(f"total cold spend: ${spent:.2f}")
    json.dump(results, open(os.path.join(ROOT, "results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
