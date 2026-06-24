"""C4 — standards-change-over-time (supersession) live test. Three arms scored by the existing
reversal_scorer.score_supersession:
  cold        : no memory                -> expect neither (opus default: bare return err)
  warm-X      : vault has only X (wrap %w) -> expect follows_x
  warm-XXp    : vault has X + superseding X' (sentinel) -> expect supersession_correct

Usage: python3 c4.py [--model opus] [--n 5] [--workers 6] [--arms cold,warm-X,warm-XXp]
"""
import argparse, json, os, subprocess, sys, tempfile, time
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "cumulative")))
import reversal_scorer
from run import build_cold_cfg, MODELS
from wrun import build_warm_cfg, RECALL_PREFIX, _slug

CUM = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "cumulative"))
SPEC = json.load(open(os.path.join(CUM, "reversal_spec.json")))
ROOT = os.environ.get("TRAPS_ROOT", "/tmp/c4")
VAULTS = {"warm-X": os.path.join(ROOT, "vault_X"), "warm-XXp": os.path.join(ROOT, "vault_XXp")}

TASK = ("Implement an exported function `Load(path string) ([]byte, error)` in a Go module "
        "`cfgload` (run `go mod init cfgload` first) that reads the file at `path` and returns its "
        "bytes, returning an error if the read fails. Add a `main`. Make `go build` succeed.")


def _gocode(wd):
    import glob
    return "\n".join(open(f, errors="ignore").read()
                     for f in glob.glob(os.path.join(wd, "**", "*.go"), recursive=True))


def run_one(arm, model, cold_cfg, warm_cfg, idx):
    wd = tempfile.mkdtemp(prefix=f"{arm}-{idx}-", dir=os.path.join(ROOT, "ws"))
    env = dict(os.environ)
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    if arm == "cold":
        env["CLAUDE_CONFIG_DIR"] = cold_cfg
        prompt = TASK
    else:
        env["CLAUDE_CONFIG_DIR"] = warm_cfg
        env["ENGRAM_VAULT_PATH"] = VAULTS[arm]
        chunks = os.path.join(ROOT, "chunks", f"{arm}-{idx}"); os.makedirs(chunks, exist_ok=True)
        env["ENGRAM_CHUNKS_DIR"] = chunks
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
    code = _gocode(wd)
    score = reversal_scorer.score_supersession(code, SPEC) if code else None
    return {"arm": arm, "idx": idx, "score": score, "cost": out.get("total_cost_usd", 0) or 0,
            "turns": out.get("num_turns"), "wd": wd}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=5)
    ap.add_argument("--workers", type=int, default=6)
    ap.add_argument("--arms", default="cold,warm-X,warm-XXp")
    a = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cold_cfg = os.path.join(ROOT, "cold-cfg"); build_cold_cfg(cold_cfg)
    warm_cfg = os.path.join(ROOT, "warm-cfg"); build_warm_cfg(warm_cfg)

    arms = a.arms.split(",")
    jobs = [(arm, i) for arm in arms for i in range(a.n)]
    print(f"C4 supersession: {arms} × n={a.n} = {len(jobs)} {a.model} trials")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, arm, a.model, cold_cfg, warm_cfg, i): (arm, i) for arm, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            s = r["score"] or {}
            tag = ("supersession_correct" if s.get("supersession_correct") else
                   "follows_x" if s.get("follows_x") else
                   "follows_x_prime" if s.get("follows_x_prime") else "neither")
            print(f"  [{r['arm']:9} #{r['idx']}] {tag:21} ${r['cost']:.2f} turns={r['turns']}")

    print("\n=== C4 RESULTS (by arm, n={}) ===".format(a.n))
    by = {}
    for r in results:
        by.setdefault(r["arm"], []).append(r["score"] or {})
    for arm in arms:
        v = by.get(arm, [])
        fx = sum(1 for s in v if s.get("follows_x"))
        fxp = sum(1 for s in v if s.get("follows_x_prime"))
        sc = sum(1 for s in v if s.get("supersession_correct"))
        neither = sum(1 for s in v if s and not s.get("follows_x") and not s.get("follows_x_prime"))
        print(f"  {arm:9} follows_x={fx}/{a.n}  follows_x'={fxp}/{a.n}  supersession_correct={sc}/{a.n}  neither={neither}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    json.dump(results, open(os.path.join(ROOT, "c4-results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
