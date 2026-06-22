"""WARM runner: re-run the confirmed tactical traps with engram memory ON. A warm cfg carries the
real /recall + /learn skills; a seeded vault holds the convention notes. Each trap prompt is
prefixed with a recall-first instruction. The SAME deterministic check then measures whether the
behavior flipped trap -> applied (the memory payoff).

Usage:
  python3 wrun.py --vault /tmp/opus-traps-warm/vault [--model opus] [--n 5] [--traps ...] [--workers 6]
"""
import argparse, json, os, shutil, subprocess, sys, tempfile, time
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import traps as T
from run import build_cold_cfg, MODELS

REPO = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))
REPO = "/Users/joe/repos/personal/engram"
ROOT = os.environ.get("TRAPS_ROOT", "/tmp/opus-traps-warm")

RECALL_PREFIX = (
    "Before writing ANY code, consult your memory: actually INVOKE YOUR /recall skill (do not skip "
    "it, do not hand-run engram yourself in its place). Read every note it surfaces and APPLY every "
    "convention it states as a hard requirement for the code you write. Then complete this task:\n\n")


def build_warm_cfg(dst):
    build_cold_cfg(dst)  # clean base + creds
    for skill in ("recall", "learn"):
        src = os.path.join(REPO, "skills", skill)
        if os.path.isdir(src):
            shutil.copytree(src, os.path.join(dst, "skills", skill), dirs_exist_ok=True)


def _slug(cwd):
    import re
    return re.sub(r"[^A-Za-z0-9-]", "-", os.path.realpath(cwd))


def run_one(name, spec, model, cfg, vault, idx):
    wd = tempfile.mkdtemp(prefix=f"{name}-{idx}-", dir=os.path.join(ROOT, "ws"))
    chunks = os.path.join(ROOT, "chunks", f"{name}-{idx}")
    os.makedirs(chunks, exist_ok=True)
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = chunks
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", _slug(wd))
    args = ["claude", "-p", RECALL_PREFIX + spec["prompt"], "--output-format", "json",
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
        cost = out.get("total_cost_usd", 0) or 0
        if (out.get("is_error") or not out) and cost < 0.02:
            continue
        break
    # did recall actually fire? grep the session transcript for an engram query
    sid = out.get("session_id")
    recalled = False
    if sid:
        for root, _, files in os.walk(os.path.join(cfg, "projects")):
            if f"{sid}.jsonl" in files:
                tx = open(os.path.join(root, f"{sid}.jsonl"), errors="ignore").read()
                recalled = "engram query" in tx or '"recall"' in tx
                break
    verdict = spec["check"](wd)
    return {"trap": name, "idx": idx, "verdict": verdict, "recalled": recalled,
            "cost": out.get("total_cost_usd", 0) or 0, "turns": out.get("num_turns"), "wd": wd}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--vault", required=True)
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=5)
    ap.add_argument("--traps", default="req-with-context,nocolor,t-parallel,nil-guard-split,wrapped-error")
    ap.add_argument("--workers", type=int, default=6)
    a = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    cfg = os.path.join(ROOT, "warm-cfg")
    build_warm_cfg(cfg)

    names = a.traps.split(",")
    jobs = [(n, i) for n in names for i in range(a.n)]
    print(f"running {len(names)} WARM traps × n={a.n} = {len(jobs)} {a.model} trials (vault={a.vault})")

    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, n, T.TRAPS[n], a.model, cfg, a.vault, i): (n, i) for n, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result()
            results.append(r)
            print(f"  [{r['trap']:16} #{r['idx']}] {r['verdict']:8} recall={r['recalled']} ${r['cost']:.2f} turns={r['turns']}")

    print("\n=== WARM FLIP (applied = memory flipped the cold trap) ===")
    by = {}
    for r in results:
        by.setdefault(r["trap"], []).append(r)
    spent = sum(r["cost"] for r in results)
    for n in names:
        v = by.get(n, [])
        applied = sum(1 for r in v if r["verdict"] == "applied")
        trap = sum(1 for r in v if r["verdict"] == "trap")
        rec = sum(1 for r in v if r["recalled"])
        print(f"  {n:16} applied={applied}/{a.n} (trap={trap}) recall_fired={rec}/{a.n}  "
              f"-> {'FLIPPED' if applied == a.n else 'partial' if applied else 'no flip'}")
    print(f"\ntotal warm spend: ${spent:.2f}")
    json.dump(results, open(os.path.join(ROOT, "warm-results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
