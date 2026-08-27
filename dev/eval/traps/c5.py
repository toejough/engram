"""C5 — recent-history surfaced and honored. A recent decision R (an arbitrary marker convention)
is planted in the chunk index as the newest chunk, TOPICALLY DISTANT from the build task (so it
surfaces by recency, not cosine — validated free: R has score 0, lives only in the recent channel).

Arms:
  cold : no chunks            -> opus writes a normal comment (no marker)
  warm : R in recent channel  -> recall surfaces R; opus must prefix comments with the marker

Two-stage check:
  C5a surfaced : recency_probe.score_recency_hit(payload, 'R-decision.md').target_surfaced
  C5b honored  : generated code comment begins with the marker token

Usage: python3 c5.py [--model opus] [--n 5] [--workers 6] [--arms cold,warm]
"""
import argparse, glob, json, os, subprocess, sys, tempfile, time
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "cumulative")))
import recency_probe
from run import build_cold_cfg, MODELS
from wrun import build_warm_cfg, RECALL_PREFIX, _slug

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/c5")
SEED_CHUNKS = os.path.join(ROOT, "chunks")          # pre-seeded: distractors + R (newest)
MARKER = "ZÖRBAX"

TASK = ("Add an exported Go function `Timestamp()` to a new module `clock` (run `go mod init "
        "clock` first) that returns the current Unix time as a string. Put a short comment line "
        "above the function. Add a `main` that prints it. Make `go build` succeed.")


def _gocode(wd):
    return "\n".join(open(f, errors="ignore").read()
                     for f in glob.glob(os.path.join(wd, "**", "*.go"), recursive=True))


def honored(wd):
    """C5b: does any code comment begin with the marker token?"""
    for f in glob.glob(os.path.join(wd, "**", "*.go"), recursive=True):
        for line in open(f, errors="ignore"):
            s = line.strip()
            if s.startswith("//") and MARKER in s.split("//", 1)[1][:20]:
                return True
    return False


def run_one(arm, model, cold_cfg, warm_cfg, idx):
    wd = tempfile.mkdtemp(prefix=f"{arm}-{idx}-", dir=os.path.join(ROOT, "ws"))
    env = dict(os.environ)
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    payload_surfaced = None
    if arm == "cold":
        env["CLAUDE_CONFIG_DIR"] = cold_cfg
        prompt = TASK
    else:
        env["CLAUDE_CONFIG_DIR"] = warm_cfg
        env["ENGRAM_VAULT_PATH"] = os.path.join(ROOT, "ev")              # empty vault
        env["ENGRAM_CHUNKS_DIR"] = SEED_CHUNKS                            # the seeded recent index
        env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(warm_cfg, "projects", _slug(wd))
        prompt = RECALL_PREFIX + TASK
    args = ["claude", "-p", prompt, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    out = {}
    for backoff in (0, 15, 45, 120):
        if backoff:
            time.sleep(backoff)
        r = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        try:
            out = json.loads(r.stdout)
        except Exception:
            out = {}
        if (out.get("is_error") or not out) and (out.get("total_cost_usd", 0) or 0) < 0.02:
            continue
        break
    # C5a: did R surface in the recent channel of the recall this agent ran? Grep its transcript
    # for the engram query payload. Simpler + robust: re-run the same query against the seed index.
    if arm != "cold":
        q = subprocess.run(["engram", "query",
                            "--phrase", "add a Go function returning the current unix timestamp as a string",
                            "--phrase", "small helper function in a Go file"],
                           env={**env, "ENGRAM_CHUNKS_DIR": SEED_CHUNKS,
                                "ENGRAM_VAULT_PATH": os.path.join(ROOT, "ev")},
                           capture_output=True, text=True)
        payload_surfaced = recency_probe.score_recency_hit(q.stdout, "R-decision.md")["target_surfaced"]
    code = _gocode(wd)
    return {"arm": arm, "idx": idx, "built": bool(code), "surfaced": payload_surfaced,
            "honored": honored(wd) if code else None,
            "cost": out.get("total_cost_usd", 0) or 0, "turns": out.get("num_turns"), "wd": wd}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=5)
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--arms", default="cold,warm")
    ap.add_argument("--crowd", type=int, default=0,
                    help="rebuild the seed with N real-vault variant chunks before R (R stays newest)")
    a = ap.parse_args()

    if a.crowd > 0:
        import seed_c5
        seed_c5.build_seed(a.crowd)
    if not os.path.isdir(SEED_CHUNKS):
        sys.exit(f"seed chunks missing at {SEED_CHUNKS} — run the C5 seed first")
    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cold_cfg = os.path.join(ROOT, "cold-cfg"); build_cold_cfg(cold_cfg)
    warm_cfg = os.path.join(ROOT, "warm-cfg"); build_warm_cfg(warm_cfg)

    arms = a.arms.split(",")
    jobs = [(arm, i) for arm in arms for i in range(a.n)]
    print(f"C5 recency: {arms} × n={a.n} = {len(jobs)} {a.model} trials (marker={MARKER})")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, arm, a.model, cold_cfg, warm_cfg, i): (arm, i) for arm, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [{r['arm']:5} #{r['idx']}] built={r['built']} surfaced={r['surfaced']} "
                  f"honored={r['honored']} ${r['cost']:.2f} turns={r['turns']}")

    print(f"\n=== C5 RESULTS (n={a.n}) ===")
    by = {}
    for r in results:
        by.setdefault(r["arm"], []).append(r)
    for arm in arms:
        v = [x for x in by.get(arm, []) if x["built"]]            # valid (non-degenerate) trials
        surf = sum(1 for x in v if x["surfaced"])
        hon = sum(1 for x in v if x["honored"])
        print(f"  {arm:5} valid={len(v)}/{a.n}  C5a surfaced={surf}/{len(v)}  C5b honored={hon}/{len(v)}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    json.dump(results, open(os.path.join(ROOT, "c5-results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
