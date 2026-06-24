"""C4 (idiosyncratic) — supersession with a marker opus has NO prior on. The original c4.py used
wrap-%w → sentinel, but wrap-%w is opus's DEFAULT (cold follows_x 5/5), so the test was confounded.
Here X and X' are arbitrary error-marker TOKENS opus would never emit on its own, so cold lands on
'neither' and the supersession is cleanly isolatable.

  X  : every returned error must be prefixed with the marker  E7:
  X' : E7: is superseded — use the marker  ERR-CFG/  instead

Arms:
  cold     : no memory                 -> expect neither (opus invents no marker)
  warm-X   : vault has only X          -> expect follows_x      (code contains E7:, not ERR-CFG/)
  warm-XXp : vault has X + superseding X' -> expect supersession_correct (ERR-CFG/, not E7:)

warm-XXp exercises the recall skill's recency-weighting (Step 2.5-B: on conflict, recent wins).
Scored by substring on the generated Go — no LLM judge needed (the tokens are unambiguous).

Usage: python3 c4_idio.py [--model opus] [--n 5] [--workers 6] [--arms cold,warm-X,warm-XXp]
"""
import argparse, glob, json, os, subprocess, sys, tempfile, time
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from run import build_cold_cfg, MODELS
from wrun import build_warm_cfg, RECALL_PREFIX, _slug

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/c4-idio")
VAULTS = {"warm-X": os.path.join(ROOT, "vault_X"), "warm-XXp": os.path.join(ROOT, "vault_XXp")}
X_MARK, XP_MARK = "E7:", "ERR-CFG/"

TASK = ("Implement an exported function `Load(path string) ([]byte, error)` in a Go module "
        "`cfgload` (run `go mod init cfgload` first) that reads the file at `path` and returns its "
        "bytes, returning an error if the read fails. Add a `main`. Make `go build` succeed.")


def _learn(vault, slug, subj, pred, obj, source):
    env = dict(os.environ); env["ENGRAM_VAULT_PATH"] = vault
    subprocess.run(["engram", "learn", "fact", "--slug", slug, "--position", "top",
                    "--source", source, "--situation",
                    "returning an error from an exported Go function in the cfgload codebase",
                    "--subject", subj, "--predicate", pred, "--object", obj],
                   env=env, check=True, capture_output=True, text=True)


def seed_vaults():
    for v in VAULTS.values():
        os.makedirs(v, exist_ok=True)
    # X — the original marker convention (goes into both warm vaults)
    for v in (VAULTS["warm-X"], VAULTS["warm-XXp"]):
        _learn(v, "e7-error-marker",
               "every error returned from an exported function in cfgload",
               "must be prefixed with the marker token",
               'E7: — e.g. fmt.Errorf("E7: reading file: %w", err)',
               "cfgload codebase convention")
    # X' — supersedes X (newer; only in the XXp vault)
    _learn(VAULTS["warm-XXp"], "errcfg-supersedes-e7",
           "the E7: error-marker prefix convention",
           "is superseded and must no longer be used; replace it with",
           'the marker ERR-CFG/ — e.g. fmt.Errorf("ERR-CFG/ reading file: %w", err)',
           "cfgload codebase convention update 2026-06")


def _gocode(wd):
    return "\n".join(open(f, errors="ignore").read()
                     for f in glob.glob(os.path.join(wd, "**", "*.go"), recursive=True))


def score(code):
    has_x, has_xp = X_MARK in code, XP_MARK in code
    return {"follows_x": has_x and not has_xp,
            "supersession_correct": has_xp and not has_x,
            "both": has_x and has_xp,
            "neither": not has_x and not has_xp}


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
    return {"arm": arm, "idx": idx, "built": bool(code), "score": score(code) if code else None,
            "cost": out.get("total_cost_usd", 0) or 0, "turns": out.get("num_turns"), "wd": wd}


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
    seed_vaults()

    arms = a.arms.split(",")
    jobs = [(arm, i) for arm in arms for i in range(a.n)]
    print(f"C4-idio supersession: {arms} × n={a.n} = {len(jobs)} {a.model} trials "
          f"(X={X_MARK} X'={XP_MARK})")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, arm, a.model, cold_cfg, warm_cfg, i): (arm, i) for arm, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            s = r["score"] or {}
            tag = next((k for k in ("supersession_correct", "follows_x", "both", "neither")
                        if s.get(k)), "no-build")
            print(f"  [{r['arm']:9} #{r['idx']}] {tag:21} ${r['cost']:.2f} turns={r['turns']}")

    print(f"\n=== C4-idio RESULTS (n={a.n}) ===")
    by = {}
    for r in results:
        by.setdefault(r["arm"], []).append(r)
    for arm in arms:
        v = [x for x in by.get(arm, []) if x["built"]]
        fx = sum(1 for x in v if x["score"]["follows_x"])
        sc = sum(1 for x in v if x["score"]["supersession_correct"])
        nt = sum(1 for x in v if x["score"]["neither"])
        bt = sum(1 for x in v if x["score"]["both"])
        print(f"  {arm:9} valid={len(v)}/{a.n}  follows_x={fx}  supersession_correct={sc}  "
              f"neither={nt}  both={bt}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    json.dump(results, open(os.path.join(ROOT, "c4-idio-results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
