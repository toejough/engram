#!/usr/bin/env python3
"""Cumulative-accumulation matrix orchestrator (v3 — 2 regimes: cold, real.full).

Per (model, trial) chain — TWO regimes, each running its own 3-app chain:
  cold:      app1(notes) → app2(links) → app3(feeds), no memory, no learn
  real.full: app1 → app2 → app3, each app builds with /recall, then /learn in-session.
             Between apps the vault accumulates (seeded vault_in + /learn fact+feedback notes).

  => 2 regimes × 3 apps = 6 build ops, 0 separate learn ops (learn is in-session for real.full).
Matrix = that × (models × trials). Pilot: 1 model × 1 trial. Full: 3 × 5 = 90 cells.

Operations form a DAG (app2 build waits on app1; app3 waits on app2). Resumable (skips
ops whose result JSON exists), budget-capped, parallel across a pool of ISOLATED cfg dirs.

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
    injected at runtime. Carries NOTHING that injects conventions ambiently (clean room).

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
        # real.full: both /recall and /learn skills from the repo (the shipped skills).
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



REAL_REGIMES = ("cold", "real.full", "real.checklist")  # real.checklist = Lever 4 (gating handoff)


def real_cells_for(model, trial, date, stub, max_rounds, regimes):
    """Two-regime chains (cold, real.full). Each app is ONE one-session cell — the build op runs
    recall → build → /learn IN-SESSION for real.full; cold = build only, no recall, no learn.
    Each regime runs its OWN 3-app chain from app1. app1 for real.full recalls an empty seed vault
    (a no-op that still fires the skill, seeding recall behavior from the start)."""
    pfx = f"{model}-t{trial}"
    stub_args = ["--stub", stub] if stub else []
    apps = [("notes", "app1"), ("links", "app2"), ("feeds", "app3")]
    sel = [r for r in REAL_REGIMES if (regimes is None or r in regimes)]
    ops = []
    for regime in sel:
        rc = harness.REGIMES[regime]
        needs_vault = rc["write"] == "skill"
        read_cfg = "cold" if rc["read_mode"] == "none" else "warm"
        prev_vault, prev_dep = "none", []
        for i, (app, tag) in enumerate(apps):
            terminal = (i == len(apps) - 1)
            ws = f"{WS}/{pfx}-{tag}-{regime}"
            out = f"{RESULTS}/{pfx}-{tag}-{regime}-build.json"
            vault_out = f"{VAULTS}/v-{pfx}-{tag}-{regime}" if (needs_vault and not terminal) else ""
            cmd = ["build", "--app", app, "--model", model, "--regime", regime, "--trial", str(trial),
                   "--date", date, "--vault-in", prev_vault, "--workdir", ws,
                   "--spec", f"{CUM}/{app}_spec.json", "--out", out, "--max-rounds", str(max_rounds)]
            if vault_out:
                cmd += ["--vault-out", vault_out]
            ops.append(_op("build", f"{pfx}-{tag}-{regime}-build", prev_dep, read_cfg, out, cmd + stub_args))
            # Sequential dependency ONLY for regimes that carry memory forward (warm): app2 recalls
            # the notes app1 LEARNED, so it must wait. Cold writes no vault (--vault-in none for every
            # app), so its 3 apps share nothing and are independent — chaining them was artificial and
            # serialized 15 independent cold ops behind 5 lanes. Dropping it lifts peak parallelism
            # from 10 (chains) to ~20 (15 cold + 5 warm frontier).
            prev_dep = [f"{pfx}-{tag}-{regime}-build"] if needs_vault else []
            prev_vault = vault_out or prev_vault
    return ops


def ops_for(model, trial, date, stub, max_rounds, regimes):
    """All operations for one (model, trial) chain: real-skill regimes only (recall-v2)."""
    return real_cells_for(model, trial, date, stub, max_rounds, regimes)


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
    if op["kind"] == "learn" and d.get("learned") is False and d.get("write_mode") != "none":
        return False  # didn't engage engram at all (empty seed) — transient, re-run
    # NOTE: learn output quality (notes_written / crystallizations) is TRACKED and reported, not a
    # re-run trigger — a thin capture is a measured finding, not a failure to retry.
    return True


def op_cost(out):
    try:
        d = json.load(open(out))
        # real.* cells store the in-session learn cost nested under d["learn"]["cost"]; legacy learn
        # ops use the flat learn_cost key. Count both so the live "spent" tally is honest.
        learn_nested = (d.get("learn") or {}).get("cost") or 0
        return (d.get("build_cost") or 0) + (d.get("learn_cost") or 0) + (d.get("total_cost") or 0) + learn_nested
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


def write_manifest(models, trials, date, stub, regimes):
    os.makedirs(RESULTS, exist_ok=True)
    json.dump({
        "schema_version": harness.SCHEMA_VERSION, "engram_sha": harness.engram_sha(),
        "date": date, "models": models, "model_ids": {m: harness.MODELS[m] for m in models},
        "trials": trials, "regimes": regimes, "stub": stub or None,
        "price_sheet_date": PRICE_SHEET_DATE,
    }, open(f"{RESULTS}/run-manifest.json", "w"), indent=2)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--models", default=",".join(ALL_MODELS))
    ap.add_argument("--trials", default="1,2,3,4,5")
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--budget", type=float, default=0.0)  # 0 = NO spend cap (Joe's preference); >0 caps
    ap.add_argument("--timeout-min", type=int, default=45)
    ap.add_argument("--date", default=datetime.date.today().isoformat())
    ap.add_argument("--max-rounds", type=int, default=8)  # lowered from 15; escalation drives completion
    ap.add_argument("--stub", default="", choices=["", "good", "naive"])
    ap.add_argument("--regimes", default="",
                    help="comma-separated regime keys to restrict the run to (e.g. cold,real.full); "
                         "default empty = all regimes")
    args = ap.parse_args()

    models = [m for m in args.models.split(",") if m in ALL_MODELS]
    trials = [int(t) for t in args.trials.split(",")]
    regimes = None
    if args.regimes:
        regimes = [r for r in args.regimes.split(",") if r in harness.REGIMES]
        unknown = [r for r in args.regimes.split(",") if r not in harness.REGIMES]
        if unknown:
            log(f"!! unknown regime(s) ignored: {unknown}")
        if not regimes:
            log(f"!! no valid regimes in --regimes={args.regimes!r}; nothing to run.")
            return
    manifest_regimes = regimes if regimes is not None else list(harness.REGIMES.keys())
    for d in (VAULTS, RESULTS, WS):
        os.makedirs(d, exist_ok=True)
    write_manifest(models, trials, args.date, args.stub, manifest_regimes)

    regime_set = set(regimes) if regimes is not None else None
    pools = make_pools(nwarm=args.workers, ncold=args.workers)
    all_ops = [op for m in models for t in trials
               for op in ops_for(m, t, args.date, args.stub, args.max_rounds, regime_set)]
    by_id = {op["id"]: op for op in all_ops}
    log(f"matrix: {len(all_ops)} ops | models={models} trials={trials} regimes={manifest_regimes} "
        f"workers={args.workers} budget={'none' if not args.budget else '$'+str(args.budget)} stub={args.stub or 'no'}")
    log(f"already done: {sum(1 for op in all_ops if op_done(op))}/{len(all_ops)} | spent ${spent_so_far():.2f}")

    done = set(oid for oid in by_id if op_done(by_id[oid]))
    submitted = set(done)
    timeout_s = args.timeout_min * 60

    def ready(op):
        return op["id"] not in submitted and all(d in done for d in op["dep"])

    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {}
        while len(done) < len(all_ops):
            if args.budget and spent_so_far() >= args.budget:
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
