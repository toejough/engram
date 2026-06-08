#!/usr/bin/env python3
"""Cumulative-accumulation matrix orchestrator (v2 — 7 regimes, write x read decoupled).

Per (model, trial) chain (§1.3):
  app1 = notes built COLD ONCE, then 4 write-tier learns -> v1[none|L1|L2|L3]
  then 7 regimes branch: each app2 = links recalls v1[write] under its read-subset,
  builds, learns -> v2[regime]; each app3 = feeds recalls v2[regime] (terminal, no learn).
  => 1 app1 build + 4 app1 learns + 7x(app2 build+learn) + 7x app3 build = 26 ops; 18 "cells".
Matrix = that x (models x trials). Pilot: 1 model x 1 trial. Full: 3 x 5 = 270 cells.

Operations form a DAG (a learn waits on its build; an app2 build waits on the app1
learn that seeds its vault). Resumable (skips ops whose result JSON exists), budget-
capped, parallel across a pool of ISOLATED cfg dirs (real headless builds, not Workflow
subagents — cold-isolation is the point). Durable: cfgs are built from the repo's own
skills + keychain creds, so the benchmark stands up from a clean checkout (no /tmp source).

Usage:
  python3 matrix.py [--models haiku,sonnet,opus] [--trials 1,2,3,4,5] [--workers N]
                    [--budget USD] [--timeout-min M] [--date YYYY-MM-DD] [--stub good|naive]
"""
import argparse, concurrent.futures as cf, datetime, json, os, queue, shutil, subprocess, sys, threading, time

CUM = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(os.path.dirname(os.path.dirname(CUM)))  # dev/eval/cumulative -> repo root
sys.path.insert(0, CUM)
import harness  # REGIMES, MODELS, engram_sha

ROOT = os.environ.get("CUMMATRIX_ROOT", "/tmp/cummatrix")
VAULTS = ROOT + "/vaults"
RESULTS = ROOT + "/results"
WS = ROOT + "/ws"
CFGPOOL = ROOT + "/cfgpool"
KEYCHAIN = 'security find-generic-password -s "Claude Code-credentials" -w'

ALL_MODELS = list(harness.MODELS.keys())
WRITE_TIERS = ["none", "L1", "L2", "L3"]
APP_SPEC = {"notes": "notes_spec.json", "links": "links_spec.json", "feeds": "feeds_spec.json"}

# Price sheet ($/Mtok), verified 2026-06-02 (carried forward for provenance; see results doc).
PRICE_SHEET_DATE = "2026-06-02"

_print_lock = threading.Lock()


def log(msg):
    with _print_lock:
        print(msg, flush=True)


# ---- durable cfg pool (built from repo skills + keychain creds; no /tmp source) ----

def build_cfg_template(dst, warm):
    """A self-contained CLAUDE_CONFIG_DIR: onboarding/oauth state from the local Claude
    install (history stripped), the repo's recall+learn skills (warm only), and creds
    injected at runtime. Carries NOTHING that injects conventions ambiently (clean room §4).

    Skip if it already exists with skills wired — so a RESUME launch does not rmtree the pool
    and destroy completed cells' transcripts (cost provenance). Token I/O is captured into each
    result JSON at run time regardless, but preserving transcripts keeps the independent
    verify_cost2 cross-check usable across resumes."""
    if os.path.exists(os.path.join(dst, ".claude.json")) and (not warm or os.path.isdir(os.path.join(dst, "skills"))):
        return
    shutil.rmtree(dst, ignore_errors=True)
    os.makedirs(dst, exist_ok=True)

    user_cfg = os.path.expanduser("~/.claude/.claude.json")
    base = {}
    if os.path.exists(user_cfg):
        try:
            base = json.load(open(user_cfg))
        except Exception:
            base = {}
    base["projects"] = {}  # drop per-project history; keep auth/onboarding flags
    json.dump(base, open(os.path.join(dst, ".claude.json"), "w"))

    if warm:
        for skill in ("recall", "learn"):
            src = os.path.join(REPO, "skills", skill)
            if os.path.isdir(src):
                shutil.copytree(src, os.path.join(dst, "skills", skill))


def refresh_creds(cfg):
    subprocess.run(["bash", "-c", f'{KEYCHAIN} > {cfg}/.credentials.json && chmod 600 {cfg}/.credentials.json'],
                   capture_output=True)


def make_pools(nwarm, ncold):
    os.makedirs(CFGPOOL, exist_ok=True)
    warm_q, cold_q = queue.Queue(), queue.Queue()
    for i in range(nwarm):
        d = f"{CFGPOOL}/warm{i}"
        build_cfg_template(d, warm=True)
        warm_q.put(d)
    for i in range(ncold):
        d = f"{CFGPOOL}/cold{i}"
        build_cfg_template(d, warm=False)
        cold_q.put(d)
    return {"warm": warm_q, "cold": cold_q}


# ---- operation DAG ----

def _op(kind, oid, dep, cfg_kind, out, cmd_tail):
    return {"kind": kind, "id": oid, "dep": dep, "cfg_kind": cfg_kind, "out": out, "cmd_tail": cmd_tail}


def cells_for(model, trial, date, stub, max_rounds):
    """All operations for one (model, trial) chain, with dependencies."""
    pfx = f"{model}-t{trial}"
    ws1 = f"{WS}/{pfx}-app1"
    stub_args = ["--stub", stub] if stub else []
    ops = []

    app1_build_out = f"{RESULTS}/{pfx}-app1-build.json"
    ops.append(_op("build", f"{pfx}-app1-build", [], "cold", app1_build_out, [
        "build", "--app", "notes", "--model", model, "--regime", "cold", "--trial", str(trial),
        "--date", date, "--vault-in", "none", "--workdir", ws1, "--spec", f"{CUM}/notes_spec.json",
        "--out", app1_build_out, "--max-rounds", str(max_rounds)] + stub_args))

    for tier in WRITE_TIERS:
        v1 = f"{VAULTS}/v1-{pfx}-{tier}"
        out = f"{RESULTS}/{pfx}-app1-learn-{tier}.json"
        ops.append(_op("learn", f"{pfx}-app1-learn-{tier}", [f"{pfx}-app1-build"],
                       "none" if tier == "none" else "warm", out, [
            "learn", "--app", "notes", "--model", model, "--regime", f"app1-{tier}", "--trial", str(trial),
            "--date", date, "--write-tier", tier, "--workdir", ws1, "--vault-in", "none",
            "--vault-out", v1, "--build-result", app1_build_out, "--out", out] + stub_args))

    for regime, rc in harness.REGIMES.items():
        write = rc["write"]
        v1 = f"{VAULTS}/v1-{pfx}-{write}"
        v2 = f"{VAULTS}/v2-{pfx}-{regime}"
        ws2 = f"{WS}/{pfx}-app2-{regime}"
        ws3 = f"{WS}/{pfx}-app3-{regime}"
        read_cfg = "cold" if rc["read_mode"] == "none" else "warm"

        a2b = f"{RESULTS}/{pfx}-app2-{regime}-build.json"
        ops.append(_op("build", f"{pfx}-app2-{regime}-build", [f"{pfx}-app1-learn-{write}"], read_cfg, a2b, [
            "build", "--app", "links", "--model", model, "--regime", regime, "--trial", str(trial),
            "--date", date, "--vault-in", v1, "--workdir", ws2, "--spec", f"{CUM}/links_spec.json",
            "--out", a2b, "--max-rounds", str(max_rounds)] + stub_args))

        a2l = f"{RESULTS}/{pfx}-app2-{regime}-learn.json"
        ops.append(_op("learn", f"{pfx}-app2-{regime}-learn", [f"{pfx}-app2-{regime}-build"],
                       "none" if write == "none" else "warm", a2l, [
            "learn", "--app", "links", "--model", model, "--regime", regime, "--trial", str(trial),
            "--date", date, "--write-tier", write, "--workdir", ws2, "--vault-in", v1,
            "--vault-out", v2, "--build-result", a2b, "--out", a2l] + stub_args))

        a3b = f"{RESULTS}/{pfx}-app3-{regime}-build.json"
        ops.append(_op("build", f"{pfx}-app3-{regime}-build", [f"{pfx}-app2-{regime}-learn"], read_cfg, a3b, [
            "build", "--app", "feeds", "--model", model, "--regime", regime, "--trial", str(trial),
            "--date", date, "--vault-in", v2, "--workdir", ws3, "--spec", f"{CUM}/feeds_spec.json",
            "--out", a3b, "--max-rounds", str(max_rounds)] + stub_args))

    return ops


def op_done(op):
    """An op counts as done only if it produced a VALID result — so a resume re-runs cells that
    timed out, rate-limited (incomplete build), or whose learn never engaged engram (empty seed),
    instead of skipping them with poisoned data."""
    if not os.path.exists(op["out"]):
        return False
    try:
        d = json.load(open(op["out"]))
    except Exception:
        return False
    if d.get("timeout"):
        return False
    if op["kind"] == "build" and d.get("rate_limited"):
        return False
    if op["kind"] == "learn" and d.get("learned") is False and d.get("write_tier") != "none":
        return False  # didn't engage engram at all (empty seed) — transient, re-run
    # NOTE: a missing L1 episode is a real failure, but it's TRACKED (learn_quality.episode_extracted
    # + aggregate report), not re-run — it's been systematic for L2, so re-running would never
    # complete. The prompt now requires the episode; persistent misses are a measured finding.
    return True


def op_cost(out):
    try:
        d = json.load(open(out))
        return (d.get("build_cost") or 0) + (d.get("learn_cost") or 0) + (d.get("total_cost") or 0)
    except Exception:
        return 0.0


def spent_so_far():
    tot = 0.0
    for f in (os.listdir(RESULTS) if os.path.isdir(RESULTS) else []):
        if f.endswith(".json") and f != "run-manifest.json":
            tot += op_cost(f"{RESULTS}/{f}")
    return tot


# ---- run one operation ----

def run_op(op, pools, timeout_s):
    oid = op["id"]
    if op_done(op):
        log(f"  [skip] {oid}")
        return oid, 0.0

    cfg = None
    if op["cfg_kind"] != "none":
        cfg = pools[op["cfg_kind"]].get()
    try:
        cmd = ["python3", f"{CUM}/harness.py"] + op["cmd_tail"]
        if cfg:
            refresh_creds(cfg)
            cmd += ["--cfg", cfg]
        elif op["kind"] == "build":
            cmd += ["--cfg", "/dev/null"]  # build requires --cfg; cold-read stub builds never use it
        t0 = time.time()
        log(f"  [run ] {oid} ({op['cfg_kind']})")
        try:
            subprocess.run(cmd, capture_output=True, text=True, timeout=timeout_s)
        except subprocess.TimeoutExpired:
            log(f"  [TIMEOUT] {oid} after {timeout_s}s")
            json.dump({"id": oid, "timeout": True, "build_cost": 0, "learn_cost": 0},
                      open(op["out"], "w"))
        cost = op_cost(op["out"]) if os.path.exists(op["out"]) else 0.0
        log(f"  [done] {oid} ${cost:.2f} {(time.time()-t0)/60:.1f}min")
        return oid, cost
    finally:
        if cfg:
            pools[op["cfg_kind"]].put(cfg)


def write_manifest(models, trials, date, stub):
    os.makedirs(RESULTS, exist_ok=True)
    json.dump({
        "schema_version": harness.SCHEMA_VERSION, "engram_sha": harness.engram_sha(),
        "date": date, "models": models, "model_ids": {m: harness.MODELS[m] for m in models},
        "trials": trials, "regimes": list(harness.REGIMES.keys()), "stub": stub or None,
        "price_sheet_date": PRICE_SHEET_DATE,
    }, open(f"{RESULTS}/run-manifest.json", "w"), indent=2)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--models", default=",".join(ALL_MODELS))
    ap.add_argument("--trials", default="1,2,3,4,5")
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--budget", type=float, default=1500.0)
    ap.add_argument("--timeout-min", type=int, default=45)
    ap.add_argument("--date", default=datetime.date.today().isoformat())
    ap.add_argument("--max-rounds", type=int, default=15)  # high safety cap; escalation drives completion
    ap.add_argument("--stub", default="", choices=["", "good", "naive"])
    args = ap.parse_args()

    models = [m for m in args.models.split(",") if m in ALL_MODELS]
    trials = [int(t) for t in args.trials.split(",")]
    for d in (VAULTS, RESULTS, WS):
        os.makedirs(d, exist_ok=True)
    write_manifest(models, trials, args.date, args.stub)

    pools = make_pools(nwarm=args.workers, ncold=args.workers)
    all_ops = [op for m in models for t in trials for op in cells_for(m, t, args.date, args.stub, args.max_rounds)]
    by_id = {op["id"]: op for op in all_ops}
    log(f"matrix: {len(all_ops)} ops | models={models} trials={trials} regimes={len(harness.REGIMES)} "
        f"workers={args.workers} budget=${args.budget} stub={args.stub or 'no'}")
    log(f"already done: {sum(1 for op in all_ops if op_done(op))}/{len(all_ops)} | spent ${spent_so_far():.2f}")

    done = set(oid for oid in by_id if op_done(by_id[oid]))
    submitted = set(done)
    timeout_s = args.timeout_min * 60

    def ready(op):
        return op["id"] not in submitted and all(d in done for d in op["dep"])

    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {}
        while len(done) < len(all_ops):
            if spent_so_far() >= args.budget:
                log(f"!! BUDGET ${args.budget} reached (spent ${spent_so_far():.2f}); stopping launches.")
                break
            for op in all_ops:
                if ready(op):
                    submitted.add(op["id"])
                    futs[ex.submit(run_op, op, pools, timeout_s)] = op["id"]
            if not futs:
                log("!! no ready ops and not all done — dependency deadlock or all in-flight failed.")
                break
            fut = next(cf.as_completed(futs))
            oid = futs.pop(fut)
            try:
                fut.result()
            except Exception as e:
                log(f"  [error] {oid}: {e}")
            done.add(oid)
            log(f"  progress: {len([o for o in done if op_done(by_id[o])])} done / {len(all_ops)} "
                f"| spent ${spent_so_far():.2f}")
        for fut in cf.as_completed(list(futs)):
            oid = futs[fut]
            try:
                fut.result()
            except Exception as e:
                log(f"  [error] {oid}: {e}")
            done.add(oid)

    log(f"### MATRIX COMPLETE ### {sum(1 for op in all_ops if op_done(op))}/{len(all_ops)} ops "
        f"| total ${spent_so_far():.2f}")


if __name__ == "__main__":
    main()
