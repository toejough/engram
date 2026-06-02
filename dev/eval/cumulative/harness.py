#!/usr/bin/env python3
"""One convergence cell of the cumulative accumulation eval.

build (revealed commands + regime-appropriate recall) -> score (deterministic,
bucketed) -> feed back failed items' user-symptoms -> resume -> loop to the bar
-> always-learn (full /learn incl L3). Records per-round bucketed score + cost.

Recall: blended -> /recall skill; L1/L2/L3 -> `engram query --tier`; cold -> none.
Learn:  full /learn skill (facts + episode + L3 synthesis) into vault-out.

Usage:
  harness.py --app feeds --model sonnet --regime blended \
      --vault-in <dir|none> --vault-out <dir|none> --cfg <cfgdir> \
      --workdir <dir> --spec <spec.json> --out <result.json> [--max-rounds 6] [--learn yes]
"""
import argparse, json, os, re, subprocess, sys, time
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import score as scoremod

MODELS = {"haiku": "claude-haiku-4-5-20251001", "sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
ENGRAM_BIN_DIR = "/Users/joe/go/bin"

def loadj(path):
    txt = open(path).read()
    try:
        return json.loads(txt)
    except Exception:
        best = None
        for line in txt.splitlines():
            line = line.strip()
            if not line:
                continue
            try:
                o = json.loads(line)
            except Exception:
                continue
            if isinstance(o, dict) and ("total_cost_usd" in o or o.get("type") == "result"):
                best = o
        return best or {}

def claude(cfg, model, vault, cwd, prompt, resume_sid=None):
    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "64000"
    env["PATH"] = ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    if vault and vault != "none":
        env["ENGRAM_VAULT_PATH"] = vault
    args = ["claude", "-p", prompt, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]
    if resume_sid:
        args = ["claude", "--resume", resume_sid] + args[1:]
    r = subprocess.run(args, cwd=cwd, env=env, capture_output=True, text=True)
    out = {}
    try:
        out = json.loads(r.stdout)
    except Exception:
        out = loadj_str(r.stdout)
    return out

def loadj_str(txt):
    best = {}
    for line in txt.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            o = json.loads(line)
        except Exception:
            continue
        if isinstance(o, dict) and ("total_cost_usd" in o or o.get("type") == "result"):
            best = o
    return best

def build_prompt(app, interface, regime):
    if regime == "blended":
        recall = ("\nBefore writing any code, consult your memory: use your /recall skill to surface "
                  "anything relevant to what you're about to build, and apply every convention and "
                  "decision it surfaces.\n")
    elif regime in ("L1", "L2", "L3"):
        recall = ("\nBefore writing any code, consult your memory. Run exactly this and read every "
                  "surfaced note, then apply every convention and decision they surface:\n"
                  f"  engram query --tier {regime} --phrase \"building a command-line {app} in Go\" "
                  f"--phrase \"architecture and conventions for a Go CLI tool\" "
                  f"--phrase \"{app} storage, data handling, and features\"\n")
    else:
        recall = ""
    return (f"Build a command-line {app} manager in Go, from scratch, in the current directory "
            f"(run `go mod init {app}` first).\n\nImplement these subcommands:\n{interface}\n{recall}\n"
            "Make `go test ./...` pass before you finish. Work fully autonomously: never stop to ask "
            "questions; keep going until it compiles and tests pass. Make changes by editing files "
            "directly with your tools; work across several steps; no need to reprint whole files. "
            "When done give a one-line summary.")

def feedback_prompt(failed):
    lines = "\n".join(f"- {sym}" for _, sym in failed)
    return ("Thanks — it builds, but a few things aren't right yet. Here's what I'm seeing as a user:\n"
            f"{lines}\n\nPlease address these. Keep `go test ./...` and `go vet ./...` passing. Edit the "
            "files directly; short summary when done.")

LEARN_PROMPT = (
    "Now use your /learn skill to capture durable memory from this build: the architecture conventions "
    "and the design/feature decisions, as facts (and at least one episode), and run the L3 synthesis "
    "step. The source is the code in this directory — you do not need to scan transcripts; skip "
    "`engram transcript --mark` and derive lessons from the code. Frame each note so a future agent "
    "building a similar Go CLI will surface and apply it. Work autonomously; one-line summary of how "
    "many notes you wrote.")

def glob_notes(vault):
    import glob as _g
    return _g.glob(os.path.join(vault, "**", "*.md"), recursive=True)

def refresh_creds_path(cfg):
    # refresh the cfg's keychain cred between retries, in case a long run aged it out
    try:
        subprocess.run(["bash", "-c",
            f'security find-generic-password -s "Claude Code-credentials" -w '
            f'> {cfg}/.credentials.json && chmod 600 {cfg}/.credentials.json'],
            capture_output=True, timeout=15)
    except Exception:
        pass

def converged(sc):
    # feature-complete (all behavioral buckets pass) + strong arch (<=2 nits)
    beh_fail = [f for f in sc.get("failed", []) if not f[0].startswith("ARCH:")]
    return len(beh_fail) == 0 and sc.get("arch_pass", 0) >= 8

def passed_of(sc):
    try:
        return int(sc.get("total", "0/18").split("/")[0])
    except Exception:
        return 0

def recall_fired(cfg, sid):
    hits = 0
    proj = os.path.join(cfg, "projects")
    for root, _, files in os.walk(proj):
        for fn in files:
            if sid and sid in fn:
                try:
                    if "engram query" in open(os.path.join(root, fn)).read():
                        hits += 1
                except Exception:
                    pass
    return hits

def main():
    ap = argparse.ArgumentParser()
    for a in ["app", "model", "regime", "vault-in", "vault-out", "cfg", "workdir", "spec", "out"]:
        ap.add_argument("--" + a, required=True)
    ap.add_argument("--max-rounds", type=int, default=6)
    ap.add_argument("--learn", default="yes")
    args = ap.parse_args()

    os.makedirs(args.workdir, exist_ok=True)
    spec = json.load(open(args.spec))
    rounds = []
    t0 = time.time()

    import shutil as _sh
    vault_in = args.__dict__["vault_in"]
    vault_out = args.__dict__["vault_out"]

    # ALWAYS bind an isolated ENGRAM_VAULT_PATH for the build session — even cold —
    # so a stray `engram learn`/recall can NEVER fall through to the real default
    # vault (~/.local/share/engram/vault). For warm cells this isolated build-vault
    # is a per-cell COPY of vault-in, so the /recall skill's in-loop synthesis writes
    # land in the throwaway copy, not the shared snapshot other cells read.
    build_vault = args.workdir + ".buildvault"
    _sh.rmtree(build_vault, ignore_errors=True)
    if vault_in != "none" and os.path.isdir(vault_in):
        _sh.copytree(vault_in, build_vault)
    else:
        if vault_in != "none":
            print(f"WARN: vault_in {vault_in} missing — running this cell COLD", file=sys.stderr)
            vault_in = "none"  # degrade to cold rather than crash on a failed prior
        os.makedirs(os.path.join(build_vault, "Permanent"), exist_ok=True)

    prompt = build_prompt(args.app, spec["interface"], args.regime)
    res = claude(args.cfg, args.model, build_vault, args.workdir, prompt)
    # retry with exponential backoff on an instant/empty failure (transient
    # rate-limit / overload / cred error: $0-ish, 1 turn). A single short retry
    # isn't enough when the limit is sustained (many expensive cells concurrent).
    for backoff in (15, 45, 120):
        if not (res.get("is_error") and (res.get("total_cost_usd", 0) or 0) < 0.02):
            break
        refresh_creds_path(args.cfg)
        time.sleep(backoff)
        res = claude(args.cfg, args.model, build_vault, args.workdir, prompt)
    sid = res.get("session_id")
    sc = scoremod.score(args.workdir, args.spec)
    rounds.append({"round": 1, "score": sc, "cost": res.get("total_cost_usd", 0),
                   "turns": res.get("num_turns", 0), "is_error": res.get("is_error")})

    rnd = 1
    stale = 0
    prev = passed_of(sc)
    while not converged(sc) and rnd < args.max_rounds and sc.get("build") == "ok":
        rnd += 1
        fb = feedback_prompt(sc["failed"])
        res = claude(args.cfg, args.model, build_vault, args.workdir, fb, resume_sid=sid)
        errored = bool(res.get("is_error"))
        sc = scoremod.score(args.workdir, args.spec)
        rounds.append({"round": rnd, "score": sc, "cost": res.get("total_cost_usd", 0),
                       "turns": res.get("num_turns", 0), "is_error": errored})
        if errored:
            break  # dead/errored resume — stop, don't burn rounds on a broken session
        cur = passed_of(sc)
        if cur <= prev:
            stale += 1
        else:
            stale, prev = 0, cur
        if stale >= 2:
            break  # plateau — feedback isn't moving the needle

    # Learn as a FRESH session (no resume): the build session's accumulated context
    # both overflows ("Prompt is too long") and doesn't reliably carry the new
    # ENGRAM_VAULT_PATH on --resume (it fell through to the real vault). A fresh
    # session in the workdir reads the code from disk (as the prompt instructs),
    # binds the vault env cleanly, and writes into a seeded copy we then promote.
    learn_info = None
    if args.learn == "yes" and vault_out != "none":
        learn_vault = args.workdir + ".learnvault"
        _sh.rmtree(learn_vault, ignore_errors=True)
        if vault_in != "none" and os.path.isdir(vault_in):
            _sh.copytree(vault_in, learn_vault)   # accumulate on top of prior memory
        else:
            os.makedirs(os.path.join(learn_vault, "Permanent"), exist_ok=True)
        lr = claude(args.cfg, args.model, learn_vault, args.workdir, LEARN_PROMPT)
        if lr.get("is_error") and (lr.get("total_cost_usd", 0) or 0) == 0:
            time.sleep(5)
            lr = claude(args.cfg, args.model, learn_vault, args.workdir, LEARN_PROMPT)
        # promote the freshly-written vault to the durable vault_out location
        _sh.rmtree(vault_out, ignore_errors=True)
        _sh.copytree(learn_vault, vault_out)
        learn_info = {"cost": lr.get("total_cost_usd", 0), "turns": lr.get("num_turns", 0),
                      "notes_written": len(glob_notes(vault_out)),
                      "result": (lr.get("result") or "")[:300]}

    out = {
        "app": args.app, "model": args.model, "regime": args.regime,
        "vault_in": args.__dict__["vault_in"], "vault_out": args.__dict__["vault_out"],
        "converged": converged(sc), "rounds_to_converge": rnd if converged(sc) else None,
        "round1_score": rounds[0]["score"].get("total"),
        "round1_buckets": rounds[0]["score"].get("feat_buckets"),
        "round1_arch": rounds[0]["score"].get("arch"),
        "final_score": sc.get("total"), "final_buckets": sc.get("feat_buckets"), "final_arch": sc.get("arch"),
        "recall_fired": recall_fired(args.cfg, sid) if args.regime != "cold" else 0,
        "build_cost": round(sum(r["cost"] for r in rounds), 4),
        "build_turns": sum(r["turns"] for r in rounds),
        "learn": learn_info,
        "total_cost": round(sum(r["cost"] for r in rounds) + (learn_info["cost"] if learn_info else 0), 4),
        "wall_min": round((time.time() - t0) / 60.0, 1),
        "rounds": rounds, "session_id": sid,
    }
    json.dump(out, open(args.out, "w"), indent=2)
    print(json.dumps({k: out[k] for k in ["app", "model", "regime", "converged", "rounds_to_converge",
          "round1_score", "round1_buckets", "final_score", "final_buckets", "recall_fired",
          "total_cost", "wall_min"]}, indent=2))

if __name__ == "__main__":
    main()
