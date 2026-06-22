"""Cold-confirm runner for BEHAVIORAL traps. Like run.py, but: runs each trap's `setup` to
recreate the triggering condition, locates the session transcript (to inspect what opus DID),
and supplies an LLM judge for the conceptual checks.

Usage:
  python3 brun.py [--model opus] [--n 5] [--threshold 4] [--traps name1,name2] [--workers 6]
"""
import argparse, json, os, shutil, subprocess, sys, tempfile, time
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import behavioral as B
from run import build_cold_cfg, MODELS  # reuse the clean cold-cfg builder

ROOT = os.environ.get("TRAPS_ROOT", "/tmp/opus-btraps")
JUDGE_MODEL = "claude-haiku-4-5-20251001"


def find_transcript(cfg, sid):
    if not sid:
        return ""
    proj = os.path.join(cfg, "projects")
    for root, _, files in os.walk(proj):
        if f"{sid}.jsonl" in files:
            return open(os.path.join(root, f"{sid}.jsonl"), errors="ignore").read()
    return ""


def make_judge(cfg):
    """Return a judge(code_text, rubric) -> bool. True iff the rubric's FIRST verdict word
    (the 'bad' verdict, e.g. OVERENGINEERED/SCOPECREEP) is returned by a cheap model."""
    def judge(code_text, rubric):
        # the rubric says "Answer X if ... else Y." — X is the trap verdict.
        import re
        m = re.search(r"Answer\s+([A-Z]+)\b.*?else\s+([A-Z]+)", rubric, re.S)
        trap_word = m.group(1) if m else "YES"
        prompt = (f"{rubric}\n\nRespond with EXACTLY ONE WORD.\n\n--- code ---\n{code_text[:6000]}")
        env = dict(os.environ); env["CLAUDE_CONFIG_DIR"] = cfg
        for backoff in (0, 15, 45):
            if backoff:
                time.sleep(backoff)
            r = subprocess.run(["claude", "-p", prompt, "--output-format", "json",
                                "--model", JUDGE_MODEL, "--permission-mode", "bypassPermissions"],
                               env=env, capture_output=True, text=True)
            try:
                out = json.loads(r.stdout)
            except Exception:
                out = {}
            txt = (out.get("result") or "").upper()
            if txt:
                return trap_word in txt
        return False  # judge failed → default to NOT-trap (conservative)
    return judge


def run_one(name, spec, model, cfg, judge, idx):
    wd = tempfile.mkdtemp(prefix=f"{name}-{idx}-", dir=os.path.join(ROOT, "ws"))
    if spec.get("setup"):
        spec["setup"](wd)
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    args = ["claude", "-p", spec["prompt"], "--output-format", "json",
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
    sid = out.get("session_id")
    transcript = find_transcript(cfg, sid)
    verdict = spec["check"](wd, transcript, judge)
    return {"trap": name, "idx": idx, "verdict": verdict,
            "cost": out.get("total_cost_usd", 0) or 0, "turns": out.get("num_turns"), "wd": wd}


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
    judge = make_judge(cfg)

    names = a.traps.split(",") if a.traps else list(B.TRAPS)
    jobs = [(n, i) for n in names for i in range(a.n)]
    print(f"running {len(names)} behavioral traps × n={a.n} = {len(jobs)} cold {a.model} trials")

    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, n, B.TRAPS[n], a.model, cfg, judge, i): (n, i) for n, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result()
            results.append(r)
            print(f"  [{r['trap']:20} #{r['idx']}] {r['verdict']:8} ${r['cost']:.2f} turns={r['turns']}")

    print("\n=== BEHAVIORAL REPRODUCIBILITY (cold falls in = 'trap') ===")
    confirmed = []
    by = {}
    for r in results:
        by.setdefault(r["trap"], []).append(r["verdict"])
    spent = sum(r["cost"] for r in results)
    for n in names:
        v = by.get(n, [])
        trap = v.count("trap"); applied = v.count("applied"); nob = v.count("nobuild")
        status = "CONFIRMED" if trap >= a.threshold else ("saturated" if applied > trap else "weak/invalid")
        if trap >= a.threshold:
            confirmed.append(n)
        print(f"  {n:20} trap={trap}/{a.n} applied={applied} nobuild={nob}  -> {status}")
    print(f"\nCONFIRMED ({len(confirmed)}): {confirmed}")
    print(f"total cold spend (incl. judge): ${spent:.2f}")
    json.dump(results, open(os.path.join(ROOT, "results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
