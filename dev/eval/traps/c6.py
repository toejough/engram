"""C6 — emergent synthesis. Run /recall against a CLUSTER vault (related facet notes, no
integrative note) and measure whether recall crystallizes a NEW note Z that synthesizes them —
judged by the adversarial separate-model judge. Control: a COVERED vault (principle already noted)
must NOT crystallize.

Per fixture (synthesis_fixtures/fixtureN/):
  warm-cluster : ENGRAM_VAULT_PATH = copy of vault_with_cluster -> expect crystallize + SYNTHESIS
  covered      : ENGRAM_VAULT_PATH = copy of vault_covered      -> expect NO crystallize (MUST_NOT_FIRE)

Check:
  C6a fired   : vault note count after > before (a new note appeared)
  C6b quality : judge_crystallization(stub=False, crystallized_note_content=<new note>) -> SYNTHESIS

Usage: python3 c6.py [--model opus] [--n 5] [--fixtures fixture1,..] [--arms warm-cluster,covered] [--workers 6]
"""
import argparse, glob, json, os, shutil, subprocess, sys, tempfile, time
import concurrent.futures as cf

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "cumulative")))
import synthesis_judge
from run import build_cold_cfg, MODELS
from wrun import build_warm_cfg, _slug

CUM = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "cumulative"))
FIXROOT = os.path.join(CUM, "synthesis_fixtures")
ROOT = os.environ.get("TRAPS_ROOT", "/tmp/c6")
ARM_DIR = {"warm-cluster": "vault_with_cluster", "covered": "vault_covered"}

# Prompt: invoke /recall on the situation; the skill's lazy-synthesis path should crystallize Z if
# the cluster is uncovered. No mention of "synthesize" — the skill decides.
PROMPT_TMPL = (
    "Invoke your /recall skill for the following situation, processing every cluster it returns and "
    "crystallizing any lesson that is not already covered by an existing note, exactly as the skill "
    "directs. Situation: {task}\n\nAfter recall, briefly state what you would do — no code needed.")


def _notes(vault):
    return set(os.path.basename(f) for f in glob.glob(os.path.join(vault, "*.md")))


def run_one(fixture, arm, model, warm_cfg, idx):
    src_vault = os.path.join(FIXROOT, fixture, ARM_DIR[arm])
    task = open(os.path.join(FIXROOT, fixture, "task.txt")).read().strip()
    expected_z = open(os.path.join(FIXROOT, fixture, "expected_synthesis_z.txt")).read().strip()
    # writable copy of the fixture vault (recall's crystallization writes here)
    wd = tempfile.mkdtemp(prefix=f"{fixture}-{arm}-{idx}-", dir=os.path.join(ROOT, "ws"))
    vault = os.path.join(wd, "vault"); shutil.copytree(src_vault, vault)
    before = _notes(vault)

    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = warm_cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    env["ENGRAM_VAULT_PATH"] = vault
    chunks = os.path.join(wd, "chunks"); os.makedirs(chunks, exist_ok=True)
    env["ENGRAM_CHUNKS_DIR"] = chunks
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(warm_cfg, "projects", _slug(wd))
    args = ["claude", "-p", PROMPT_TMPL.format(task=task), "--output-format", "json",
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

    after = _notes(vault)
    new_notes = after - before
    crystallized = bool(new_notes)
    new_content = ""
    if new_notes:
        # judge the largest new note (the synthesized Z)
        paths = sorted((os.path.join(vault, n) for n in new_notes),
                       key=lambda p: os.path.getsize(p), reverse=True)
        new_content = open(paths[0], errors="ignore").read()

    verdict = None
    if arm == "warm-cluster":
        # quality judge only matters when something was crystallized
        if crystallized:
            try:
                jr = synthesis_judge.judge_crystallization(
                    src_vault, task, expected_z, stub=False,
                    crystallized_note_content=new_content, build_used_z=None)
                verdict = jr.get("verdict")
            except Exception as e:                       # judge CLI/auth failure — don't kill the run
                verdict = f"JUDGE_ERROR:{str(e)[:40]}"
        else:
            verdict = "NO_CRYSTALLIZATION"
    else:  # covered control
        verdict = "FIRED_WRONGLY" if crystallized else "MUST_NOT_FIRE_OK"

    return {"fixture": fixture, "arm": arm, "idx": idx, "crystallized": crystallized,
            "verdict": verdict, "n_new": len(new_notes),
            "cost": out.get("total_cost_usd", 0) or 0, "turns": out.get("num_turns"), "wd": wd}


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=5)
    ap.add_argument("--fixtures", default="fixture1,fixture2,fixture3")
    ap.add_argument("--arms", default="warm-cluster,covered")
    ap.add_argument("--workers", type=int, default=6)
    a = ap.parse_args()

    os.makedirs(os.path.join(ROOT, "ws"), exist_ok=True)
    warm_cfg = os.path.join(ROOT, "warm-cfg"); build_warm_cfg(warm_cfg)

    fixtures = a.fixtures.split(","); arms = a.arms.split(",")
    jobs = [(fx, arm, i) for fx in fixtures for arm in arms for i in range(a.n)]
    print(f"C6 synthesis: {fixtures} × {arms} × n={a.n} = {len(jobs)} {a.model} trials")
    results = []
    with cf.ThreadPoolExecutor(max_workers=a.workers) as ex:
        futs = {ex.submit(run_one, fx, arm, a.model, warm_cfg, i): (fx, arm, i) for fx, arm, i in jobs}
        for fut in cf.as_completed(futs):
            r = fut.result(); results.append(r)
            print(f"  [{r['fixture']} {r['arm']:12} #{r['idx']}] crystallized={r['crystallized']} "
                  f"verdict={r['verdict']:18} ${r['cost']:.2f}")

    print(f"\n=== C6 RESULTS (n={a.n}) ===")
    by = {}
    for r in results:
        by.setdefault((r["fixture"], r["arm"]), []).append(r)
    for fx in fixtures:
        for arm in arms:
            v = by.get((fx, arm), [])
            if arm == "warm-cluster":
                cr = sum(1 for x in v if x["crystallized"])
                syn = sum(1 for x in v if x["verdict"] == "SYNTHESIS")
                print(f"  {fx} {arm:12} crystallized={cr}/{a.n}  SYNTHESIS={syn}/{a.n}")
            else:
                ok = sum(1 for x in v if x["verdict"] == "MUST_NOT_FIRE_OK")
                print(f"  {fx} {arm:12} did-not-fire(correct)={ok}/{a.n}")
    print(f"\ntotal spend: ${sum(r['cost'] for r in results):.2f}")
    json.dump(results, open(os.path.join(ROOT, "c6-results.json"), "w"), indent=1)


if __name__ == "__main__":
    main()
