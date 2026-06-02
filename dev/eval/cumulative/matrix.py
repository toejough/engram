#!/usr/bin/env python3
"""Full cumulative-accumulation matrix orchestrator.

3 models × {cold, L1-iso, L2-iso, L3-iso, blended} recall × {+notes, +notes+links}
accumulation stages × n=3 trials. Each (model,trial) is a chain: notes -> links ->
9 feeds cells. Priors build the accumulating vault; feeds cells recall from it and
are scored per-bucket (ARCH/alpha/beta/native) at round-1 and converged.

Parallel across chains/cells with a pool of ISOLATED config dirs (cold-isolation is
why this is real headless builds, not Workflow-tool subagents). Resumable (skips
cells whose result JSON already exists) and budget-capped.

Usage: python3 matrix.py [--models m1,m2] [--trials 1,2,3] [--workers N] [--budget USD]
"""
import argparse, concurrent.futures as cf, json, os, queue, shutil, subprocess, sys, threading, time

CUM = "/Users/joe/repos/personal/engram/dev/eval/cumulative"
ROOT = "/tmp/cummatrix"
VAULTS = ROOT + "/vaults"
RESULTS = ROOT + "/results"
WS = ROOT + "/ws"
CFGPOOL = ROOT + "/cfgpool"
SRC_WARM = "/tmp/todo-coldwarm/cfg-warm"   # has recall+learn skills
SRC_COLD = "/tmp/todo-coldwarm/cfg-cold"   # no skills
KEYCHAIN = 'security find-generic-password -s "Claude Code-credentials" -w'

ALL_MODELS = ["haiku", "sonnet", "opus"]
REGIMES = ["L1", "L2", "L3", "blended"]   # tier-isolated single tiers + blended
STAGES = [("notes", "+notes"), ("noteslinks", "+notes+links")]

_print_lock = threading.Lock()
def log(msg):
    with _print_lock:
        print(msg, flush=True)

# ---- cfg pool (thread-safe) ----
def build_cfg_template(dst, src, with_skills):
    shutil.rmtree(dst, ignore_errors=True)
    os.makedirs(dst, exist_ok=True)
    for f in (".claude.json", "settings.json"):
        if os.path.exists(os.path.join(src, f)):
            shutil.copy(os.path.join(src, f), os.path.join(dst, f))
    if with_skills and os.path.isdir(os.path.join(src, "skills")):
        shutil.copytree(os.path.join(src, "skills"), os.path.join(dst, "skills"))

def make_pools(nwarm, ncold):
    os.makedirs(CFGPOOL, exist_ok=True)
    warm_q, cold_q = queue.Queue(), queue.Queue()
    for i in range(nwarm):
        d = f"{CFGPOOL}/warm{i}"; build_cfg_template(d, SRC_WARM, True); warm_q.put(d)
    for i in range(ncold):
        d = f"{CFGPOOL}/cold{i}"; build_cfg_template(d, SRC_COLD, False); cold_q.put(d)
    return {"warm": warm_q, "cold": cold_q}

def refresh_creds(cfg):
    subprocess.run(["bash", "-c", f'{KEYCHAIN} > {cfg}/.credentials.json && chmod 600 {cfg}/.credentials.json'],
                   capture_output=True)

# ---- matrix definition ----
def cells_for(model, trial):
    vn = f"{VAULTS}/vn-{model}-t{trial}"
    vnl = f"{VAULTS}/vnl-{model}-t{trial}"
    pfx = f"{model}-t{trial}"
    cells = []
    cells.append(dict(id=f"{pfx}-notes", app="notes", model=model, regime="cold",
                      vin="none", vout=vn, learn="yes", cfg="warm", spec="notes_spec.json", dep=None))
    cells.append(dict(id=f"{pfx}-links", app="links", model=model, regime="blended",
                      vin=vn, vout=vnl, learn="yes", cfg="warm", spec="links_spec.json", dep=f"{pfx}-notes"))
    cells.append(dict(id=f"{pfx}-feeds-cold", app="feeds", model=model, regime="cold",
                      vin="none", vout="none", learn="no", cfg="cold", spec="feeds_spec.json", dep=None))
    for r in REGIMES:
        cells.append(dict(id=f"{pfx}-feeds-{r}-notes", app="feeds", model=model, regime=r,
                          vin=vn, vout="none", learn="no", cfg="warm", spec="feeds_spec.json", dep=f"{pfx}-notes"))
        cells.append(dict(id=f"{pfx}-feeds-{r}-noteslinks", app="feeds", model=model, regime=r,
                          vin=vnl, vout="none", learn="no", cfg="warm", spec="feeds_spec.json", dep=f"{pfx}-links"))
    return cells

def result_path(cid):
    return f"{RESULTS}/{cid}.json"

def cell_done(cid):
    p = result_path(cid)
    if not os.path.exists(p):
        return False
    try:
        json.load(open(p)); return True
    except Exception:
        return False

# ---- run one cell ----
def run_cell(cell, pools, timeout_s):
    cid = cell["id"]
    if cell_done(cid):
        log(f"  [skip] {cid} (already done)")
        return cid, 0.0
    q = pools[cell["cfg"]]
    cfg = q.get()
    try:
        refresh_creds(cfg)
        wd = f"{WS}/{cid}"
        cmd = ["python3", f"{CUM}/harness.py", "--app", cell["app"], "--model", cell["model"],
               "--regime", cell["regime"], "--vault-in", cell["vin"], "--vault-out", cell["vout"],
               "--cfg", cfg, "--workdir", wd, "--spec", f"{CUM}/{cell['spec']}",
               "--out", result_path(cid), "--max-rounds", "5", "--learn", cell["learn"]]
        t0 = time.time()
        log(f"  [run ] {cid} (cfg={os.path.basename(cfg)})")
        try:
            subprocess.run(cmd, capture_output=True, text=True, timeout=timeout_s)
        except subprocess.TimeoutExpired:
            log(f"  [TIMEOUT] {cid} after {timeout_s}s")
            json.dump({"id": cid, "timeout": True, "total_cost": 0}, open(result_path(cid), "w"))
        cost = 0.0
        if os.path.exists(result_path(cid)):
            try: cost = json.load(open(result_path(cid))).get("total_cost", 0) or 0
            except Exception: pass
        log(f"  [done] {cid} ${cost:.2f} {(time.time()-t0)/60:.1f}min")
        return cid, cost
    finally:
        q.put(cfg)

def spent_so_far():
    tot = 0.0
    for f in os.listdir(RESULTS) if os.path.isdir(RESULTS) else []:
        if f.endswith(".json"):
            try: tot += json.load(open(f"{RESULTS}/{f}")).get("total_cost", 0) or 0
            except Exception: pass
    return tot

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--models", default="haiku,sonnet,opus")
    ap.add_argument("--trials", default="1,2,3")
    ap.add_argument("--workers", type=int, default=4)
    ap.add_argument("--budget", type=float, default=600.0)
    ap.add_argument("--timeout-min", type=int, default=45)
    args = ap.parse_args()

    models = [m for m in args.models.split(",") if m in ALL_MODELS]
    trials = [int(t) for t in args.trials.split(",")]
    for d in (VAULTS, RESULTS, WS):
        os.makedirs(d, exist_ok=True)

    pools = make_pools(nwarm=args.workers, ncold=args.workers)
    all_cells = [c for m in models for t in trials for c in cells_for(m, t)]
    by_id = {c["id"]: c for c in all_cells}
    log(f"matrix: {len(all_cells)} cells | models={models} trials={trials} workers={args.workers} budget=${args.budget}")
    log(f"already done: {sum(1 for c in all_cells if cell_done(c['id']))}/{len(all_cells)} | spent so far ${spent_so_far():.2f}")

    done = set(cid for cid in by_id if cell_done(cid))
    submitted = set(done)
    timeout_s = args.timeout_min * 60

    def ready(c):
        return c["id"] not in submitted and (c["dep"] is None or c["dep"] in done)

    with cf.ThreadPoolExecutor(max_workers=args.workers) as ex:
        futs = {}
        while len(done) < len(all_cells):
            if spent_so_far() >= args.budget:
                log(f"!! BUDGET ${args.budget} reached (spent ${spent_so_far():.2f}); stopping new launches.")
                break
            for c in all_cells:
                if ready(c):
                    submitted.add(c["id"])
                    futs[ex.submit(run_cell, c, pools, timeout_s)] = c["id"]
            if not futs:
                log("!! no ready cells and not all done — dependency deadlock or all in-flight failed."); break
            fut = next(cf.as_completed(futs))
            cid = futs.pop(fut)
            try:
                fut.result()
            except Exception as e:
                log(f"  [error] {cid}: {e}")
            done.add(cid)  # mark resolved (done or failed) so deps can proceed/skip
            log(f"  progress: {len([c for c in done if cell_done(c)])} done / {len(all_cells)} | spent ${spent_so_far():.2f}")
        for fut in cf.as_completed(list(futs)):
            cid = futs[fut]
            try: fut.result()
            except Exception as e: log(f"  [error] {cid}: {e}")
            done.add(cid)

    log(f"### MATRIX COMPLETE ### {sum(1 for c in all_cells if cell_done(c['id']))}/{len(all_cells)} cells | total ${spent_so_far():.2f}")

if __name__ == "__main__":
    main()
