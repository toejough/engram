#!/usr/bin/env python3
"""One operation of the cumulative-accumulation eval (v2).

Two modes, so build and learn decouple (app1 is built cold ONCE, then fanned out
to 4 write-tier learns — §1.3):

  build  recall (read-subset) -> build -> score -> feed back ALL gaps -> resume ->
         loop to the bar.  Records per-round convention/feature intervention counts
         (split on the scorer's ARCH: prefix), the round-1 per-detector ARCH
         snapshot (the say-once discriminator), stated_conventions (for the learn
         prompt), rounds_to_converge, recall_fired (+ recall_ok flag), link-following,
         per-round cost/turns, wall_min.  NO learn.

  learn  over an already-built workdir, write at the regime's write-tier
         (cold=nothing; L1=episode; L2=episode+facts; L3=episode+facts+§6b synthesis),
         capturing the STATED conventions the build fed back (so "say it once"
         persists), per the tiered-capture-discipline ADR.  Verifies output populated.

Recall encoding (read-subset, §1.4):
  none           -> no recall (cold)
  blended        -> engram query (no --tier): full vault returned
  tier [T ...]   -> engram query --tier T [--tier T2 ...]; surfaced notes carry
                    outbound_links and the build is told to follow them with
                    `engram show <basename>` (direct-vs-follow-on-demand, not a blinding)

Usage:
  harness.py build --app feeds --model sonnet --regime l2.l2 --trial 1 \
      --vault-in <dir|none> --cfg <cfgdir> --workdir <dir> --spec <spec.json> \
      --out <build.json> [--max-rounds 6] [--stub good|naive]
  harness.py learn --app notes --model sonnet --regime l2.l2 --trial 1 \
      --write-tier L2 --workdir <built-dir> --vault-in <dir|none> --vault-out <dir> \
      --build-result <build.json> --cfg <cfgdir> --out <learn.json> [--stub good|naive]
"""
import argparse, glob as _glob, json, os, subprocess, sys, time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import score as scoremod

# Single editable source of truth for the model registry — a new model is a one-line add (§1.5).
MODELS = {"haiku": "claude-haiku-4-5-20251001", "sonnet": "claude-sonnet-4-6", "opus": "claude-opus-4-8"}
ENGRAM_BIN_DIR = os.environ.get("ENGRAM_BIN_DIR", os.path.expanduser("~/go/bin"))
SCHEMA_VERSION = 2
CONVERGE_ARCH_BAR = 8  # arch_pass >= 8 (matches converged())

# The 7 regimes: write-tier (learn ceiling) x read-subset (recall surface). §1.2.
# read_mode: none | blended | tier ; read_tiers used only when read_mode == tier.
REGIMES = {
    "cold":      {"write": "none", "read_mode": "none",    "read_tiers": []},
    "l1":        {"write": "L1",   "read_mode": "tier",    "read_tiers": ["L1"]},
    "l2.l1l2":   {"write": "L2",   "read_mode": "blended", "read_tiers": []},
    "l2.l2":     {"write": "L2",   "read_mode": "tier",    "read_tiers": ["L2"]},
    "l3.l1l2l3": {"write": "L3",   "read_mode": "blended", "read_tiers": []},
    "l3.l2l3":   {"write": "L3",   "read_mode": "tier",    "read_tiers": ["L2", "L3"]},
    "l3.l3":     {"write": "L3",   "read_mode": "tier",    "read_tiers": ["L3"]},
}


def engram_sha():
    try:
        here = os.path.dirname(os.path.abspath(__file__))
        r = subprocess.run(["git", "-C", here, "rev-parse", "HEAD"], capture_output=True, text=True, timeout=10)
        return r.stdout.strip()[:12] or "unknown"
    except Exception:
        return "unknown"


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
    try:
        return json.loads(r.stdout)
    except Exception:
        return loadj_str(r.stdout)


def refresh_creds_path(cfg):
    # refresh the cfg's keychain cred between retries, in case a long run aged it out
    try:
        subprocess.run(["bash", "-c",
            f'security find-generic-password -s "Claude Code-credentials" -w '
            f'> {cfg}/.credentials.json && chmod 600 {cfg}/.credentials.json'],
            capture_output=True, timeout=15)
    except Exception:
        pass


def build_prompt(app, interface, read_mode, read_tiers):
    """Build prompt with read-subset-appropriate recall. Tier-capped reads are told
    EXPLICITLY they can follow each surfaced note's outbound_links with `engram show
    <basename>` — this is what makes a tier-read regime a test of
    direct-provision-vs-follow-on-demand rather than a blinding (§1.4)."""
    phrases = (f'--phrase "building a command-line {app} in Go" '
               f'--phrase "architecture and conventions for a Go CLI tool" '
               f'--phrase "{app} storage, data handling, and features"')
    if read_mode == "none":
        recall = ""
    elif read_mode == "blended":
        recall = ("\nBefore writing any code, consult your memory — run exactly this, read every "
                  "surfaced note, and APPLY every convention and decision it surfaces:\n"
                  f"  engram query {phrases}\n")
    else:  # tier
        tier_flags = " ".join(f"--tier {t}" for t in read_tiers)
        recall = (
            "\nBefore writing any code, consult your memory. Run exactly this, read every surfaced "
            "note, and APPLY every convention and decision it surfaces:\n"
            f"  engram query {tier_flags} {phrases}\n"
            "This recall is TIER-CAPPED: engram returns only the listed tier(s). Each surfaced item's "
            "YAML carries `outbound_links` — the basenames of related notes one hop away (often the "
            "lower-tier notes a distilled standard was built from). When a surfaced note cites a "
            "constituent whose content you need, FETCH IT on demand with:\n"
            "  engram show <basename>\n"
            "(`engram show` prints the note's full content plus its own outbound_links, so one fetch "
            "reveals the next hop.) Follow whatever links you need to apply the conventions fully — the "
            "tier cap limits only what engram volunteers, never what you can open.\n")
    return (f"Build a command-line {app} manager in Go, from scratch, in the current directory "
            f"(run `go mod init {app}` first).\n\nImplement these subcommands:\n{interface}\n{recall}\n"
            "Make `go test ./...` pass before you finish. Work fully autonomously: never stop to ask "
            "questions; keep going until it compiles and tests pass. Make changes by editing files "
            "directly with your tools; work across several steps; no need to reprint whole files. "
            "When done give a one-line summary.")


def feedback_prompt(failed):
    """States ALL gaps — convention and feature alike (it is fair to tell the model what you
    want; §4). The harness LABELS and counts them separately; it never strips convention gaps."""
    lines = "\n".join(f"- {sym}" for _, sym in failed)
    return ("Thanks — it builds, but a few things aren't right yet. Here's what I'm seeing as a user:\n"
            f"{lines}\n\nPlease address these. Keep `go test ./...` and `go vet ./...` passing. Edit the "
            "files directly; short summary when done.")


def is_convention(label):
    """The convention/feature split key: the scorer prefixes architecture detectors with
    ARCH: (name-agnostic, transferable conventions); behavioral checks carry an
    alpha:/beta:/native: bucket prefix (app-specific features). §4/§5."""
    return label.startswith("ARCH:")


def split_failed(failed):
    conv = [(lbl, sym) for lbl, sym in failed if is_convention(lbl)]
    feat = [(lbl, sym) for lbl, sym in failed if not is_convention(lbl)]
    return conv, feat


def conv_labels(failed):
    return [lbl[len("ARCH:"):] for lbl, _ in failed if is_convention(lbl)]


LEARN_INTRO = (
    "Now use your /learn skill to capture durable memory from the build in this directory. "
    "The source is the code here — you do not need to scan transcripts; skip `engram transcript "
    "--mark` and derive lessons from the code. Frame every note so a future agent building a "
    "DIFFERENT Go CLI surfaces and applies it (transferable conventions, not this app's features).")

STATED_INTRO = (
    "CRITICAL — capture the conventions the reviewer STATED this build, not only patterns you "
    "re-derive from the code. The reviewer explicitly asked for these architecture conventions "
    "during review:\n{stated}\nPersist each as a durable convention (a fact, or feedback), phrased "
    "generally, so the next app's recall surfaces the instruction without the human restating it.\n")

LEARN_BY_TIER = {
    "L1": ("Capture exactly ONE episode of this build (a concrete record of what you built — files, "
           "interfaces, patterns, and the conventions you applied). Write the episode only; no facts, "
           "no L3 synthesis."),
    "L2": ("Capture ONE episode of this build, then FACTS — one fact per architecture convention "
           "(both the ones the reviewer stated and the ones you applied). Relation-link each fact to "
           "the episode it came from. Do NOT run L3 synthesis."),
    "L3": ("Capture ONE episode of this build, then FACTS (one per convention, each relation-linked "
           "to the episode), then run the §6b L3 synthesis (`engram query --synthesis` scenario-seeded "
           "over the L2 clusters, update-or-create by nearest_l3 cosine). Write episode + facts + ADRs."),
}


def learn_prompt(write_tier, stated):
    parts = [LEARN_INTRO]
    if stated:
        parts.append(STATED_INTRO.format(stated="\n".join(f"  - {s}" for s in stated)))
    parts.append(LEARN_BY_TIER[write_tier])
    parts.append("Work autonomously; one-line summary of how many notes of each tier you wrote.")
    return "\n\n".join(parts)


def glob_notes(vault):
    return _glob.glob(os.path.join(vault, "**", "*.md"), recursive=True)


def count_notes_by_tier(vault):
    """Verify the learn actually populated each tier — a tested mechanism produces nothing until
    run on real data (note-18). Reads the frontmatter tier: of every note."""
    counts = {"L1": 0, "L2": 0, "L3": 0, "other": 0}
    for path in glob_notes(vault):
        try:
            head = open(path).read(600)
        except Exception:
            continue
        tier = "other"
        for line in head.splitlines():
            s = line.strip()
            if s.startswith("tier:"):
                tier = s.split(":", 1)[1].strip().strip('"').strip("'")
                break
        counts[tier if tier in counts else "other"] += 1
    return counts


def converged(sc):
    # feature-complete (all behavioral buckets pass) + strong arch (arch_pass >= 8)
    beh_fail = [f for f in sc.get("failed", []) if not f[0].startswith("ARCH:")]
    return len(beh_fail) == 0 and sc.get("arch_pass", 0) >= CONVERGE_ARCH_BAR


def passed_of(sc):
    try:
        return int(sc.get("total", "0/18").split("/")[0])
    except Exception:
        return 0


def recall_fired(cfg, sid):
    """Count cell-transcript turns that ran `engram query` — the forced-memory-path assertion
    (M1): a headless agent does not self-fire recall, so a warm cell that fired zero used no
    memory and must be flagged/discarded (§4)."""
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


def link_followed(cfg, sid):
    """Whether the agent actually followed surfaced links — `engram show` calls or reads of
    Permanent/*.md beyond the surfaced set. Makes direct-vs-followed visible, not assumed (§1.4)."""
    proj = os.path.join(cfg, "projects")
    for root, _, files in os.walk(proj):
        for fn in files:
            if sid and sid in fn:
                try:
                    txt = open(os.path.join(root, fn)).read()
                except Exception:
                    continue
                if "engram show" in txt or "Permanent/" in txt:
                    return True
    return False


# ----- stub builders (no LLM; for the zero-cost dry-run, §7/§13) -----

def _stub_build(args):
    """Drop the chosen fixture Go app into the workdir (real, compilable Go the scorer
    builds and runs) and return a canned result. No LLM call."""
    import shutil
    fix = os.path.join(os.path.dirname(os.path.abspath(__file__)), "testdata", args.stub)
    for path in _glob.glob(os.path.join(fix, "*")):
        dst = os.path.join(args.workdir, os.path.basename(path))
        if os.path.isdir(path):
            shutil.copytree(path, dst, dirs_exist_ok=True)
        else:
            shutil.copy(path, dst)
    return {"is_error": False, "total_cost_usd": 0.0, "num_turns": 1,
            "session_id": "stub-build", "result": "stubbed build"}


def _stub_learn(args, vault_out):
    """Write canned tier-appropriate notes (cumulative ceiling) into vault_out and embed
    them, so vault threading + per-tier population are exercised without an LLM."""
    perm = os.path.join(vault_out, "Permanent")
    os.makedirs(perm, exist_ok=True)
    base = 900 + sum(ord(c) for c in (args.regime + args.app)) % 90

    def note(idx, tier, kind):
        name = f"{idx}.2026-06-06.stub-{args.app}-{args.regime}-{tier.lower()}"
        body = (f"---\ntype: {kind}\ntier: {tier}\n"
                f"situation: building a command-line {args.app} in Go\n"
                f"luhmann: \"{idx}\"\ncreated: \"2026-06-06\"\nsource: stub eval\n---\n\n"
                f"stub {tier} note for {args.app} ({args.regime}).\n")
        open(os.path.join(perm, name + ".md"), "w").write(body)

    note(base, "L1", "episode")
    if args.write_tier in ("L2", "L3"):
        note(base + 1, "L2", "fact")
    if args.write_tier == "L3":
        note(base + 2, "L3", "fact")

    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault_out
    env["PATH"] = ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    subprocess.run(["engram", "embed", "apply", "--all"], env=env, capture_output=True, text=True)
    return {"is_error": False, "total_cost_usd": 0.0, "num_turns": 1, "result": "stubbed learn"}


# ----- build mode -----

def _seed_build_vault(workdir, vault_in):
    """Per-cell isolated copy of vault_in so in-loop recall synthesis writes land in a
    throwaway, never the shared snapshot other cells read. Returns (build_vault, vault_in)."""
    import shutil
    build_vault = workdir + ".buildvault"
    shutil.rmtree(build_vault, ignore_errors=True)
    if vault_in != "none" and os.path.isdir(vault_in):
        shutil.copytree(vault_in, build_vault)
    else:
        if vault_in != "none":
            print(f"WARN: vault_in {vault_in} missing — building COLD", file=sys.stderr)
            vault_in = "none"
        os.makedirs(os.path.join(build_vault, "Permanent"), exist_ok=True)
    return build_vault, vault_in


def _round_rec(rnd, sc, res, conv, feat):
    return {"round": rnd, "score": sc.get("total"), "feat_buckets": sc.get("feat_buckets"),
            "arch": sc.get("arch"), "convention_fails": len(conv), "feature_fails": len(feat),
            "cost": res.get("total_cost_usd", 0) or 0, "turns": res.get("num_turns", 0) or 0,
            "is_error": bool(res.get("is_error"))}


def _arch_detector_names():
    import archscore
    return [n for n, _ in archscore.DETECTORS]


def run_build(args):
    regime = REGIMES[args.regime]
    os.makedirs(args.workdir, exist_ok=True)
    t0 = time.time()
    build_vault, vault_in = _seed_build_vault(args.workdir, args.vault_in)

    prompt = build_prompt(args.app, json.load(open(args.spec))["interface"],
                          regime["read_mode"], regime["read_tiers"])

    def do_build(resume_sid=None):
        if args.stub:
            return _stub_build(args)
        res = claude(args.cfg, args.model, build_vault, args.workdir, prompt, resume_sid=resume_sid)
        if resume_sid is None:
            for backoff in (15, 45, 120):  # transient rate-limit/overload: $0-ish, 1 turn
                if not (res.get("is_error") and (res.get("total_cost_usd", 0) or 0) < 0.02):
                    break
                refresh_creds_path(args.cfg)
                time.sleep(backoff)
                res = claude(args.cfg, args.model, build_vault, args.workdir, prompt)
        return res

    res = do_build()
    sid = res.get("session_id")
    sc = scoremod.score(args.workdir, args.spec)
    conv, feat = split_failed(sc.get("failed", []))
    rounds = [_round_rec(1, sc, res, conv, feat)]

    # Round-1 per-detector ARCH snapshot — the say-once discriminator (advisor flag 3).
    arch_fail1 = set(sc.get("arch_fail", []))
    round1_arch = {name: name not in arch_fail1 for name in _arch_detector_names()}
    round1_conv_fails = len(conv)
    round1_feat_fails = len(feat)

    stated = list(conv_labels(sc.get("failed", [])))  # cumulative, for the learn prompt

    rnd, stale, prev = 1, 0, passed_of(sc)
    while not converged(sc) and rnd < args.max_rounds and sc.get("build") == "ok":
        rnd += 1
        res = do_build(resume_sid=sid) if not args.stub else _stub_build(args)
        errored = bool(res.get("is_error"))
        sc = scoremod.score(args.workdir, args.spec)
        conv, feat = split_failed(sc.get("failed", []))
        rounds.append(_round_rec(rnd, sc, res, conv, feat))
        for lbl in conv_labels(sc.get("failed", [])):
            if lbl not in stated:
                stated.append(lbl)
        if errored:
            break
        cur = passed_of(sc)
        if cur <= prev:
            stale += 1
        else:
            stale, prev = 0, cur
        if stale >= 2:  # plateau — feedback isn't moving the needle
            break

    rf = 0 if (regime["read_mode"] == "none" or args.stub) else recall_fired(args.cfg, sid)
    recall_ok = regime["read_mode"] == "none" or bool(args.stub) or rf > 0
    followed = False if args.stub else (regime["read_mode"] == "tier" and link_followed(args.cfg, sid))

    out = {
        "schema_version": SCHEMA_VERSION, "engram_sha": engram_sha(), "kind": "build",
        "app": args.app, "model": args.model, "model_id": MODELS[args.model],
        "regime": args.regime, "trial": args.trial, "date": args.date,
        "read_mode": regime["read_mode"], "read_tiers": regime["read_tiers"],
        "vault_in": args.vault_in, "stub": args.stub or None,
        "converged": converged(sc), "rounds_to_converge": rnd if converged(sc) else None,
        "max_rounds": args.max_rounds,
        "round1_score": rounds[0]["score"], "round1_arch_detectors": round1_arch,
        "round1_convention_fails": round1_conv_fails, "round1_feature_fails": round1_feat_fails,
        "final_score": sc.get("total"), "final_buckets": sc.get("feat_buckets"),
        "final_arch": sc.get("arch"), "arch_pass": sc.get("arch_pass", 0),
        "stated_conventions": stated, "convention_statements": round1_conv_fails,
        "feature_statements": round1_feat_fails,
        "recall_fired": rf, "recall_ok": recall_ok, "link_followed": followed,
        "build_cost": round(sum(r["cost"] for r in rounds), 4),
        "build_turns": sum(r["turns"] for r in rounds),
        "wall_min": round((time.time() - t0) / 60.0, 1),
        "rounds": rounds, "session_id": sid, "workdir": args.workdir,
    }
    json.dump(out, open(args.out, "w"), indent=2)
    print(json.dumps({k: out[k] for k in ["app", "model", "regime", "converged",
          "rounds_to_converge", "round1_score", "convention_statements", "feature_statements",
          "recall_fired", "recall_ok", "link_followed", "build_cost", "wall_min"]}, indent=2))


# ----- learn mode -----

def run_learn(args):
    import shutil
    t0 = time.time()
    if args.write_tier == "none":
        out = {"schema_version": SCHEMA_VERSION, "engram_sha": engram_sha(), "kind": "learn",
               "app": args.app, "model": args.model, "regime": args.regime, "trial": args.trial,
               "date": args.date, "write_tier": "none", "vault_in": args.vault_in,
               "vault_out": args.vault_out, "learned": False, "notes_by_tier": {},
               "learn_cost": 0.0, "learn_turns": 0, "wall_min": 0.0}
        json.dump(out, open(args.out, "w"), indent=2)
        print(json.dumps({"app": args.app, "regime": args.regime, "write_tier": "none", "learned": False}))
        return

    # Learn into a fresh copy of vault_in (accumulate on prior memory), then promote to vault_out.
    # Stage off vault_out (unique per op), NOT workdir: app1's 4 write-tier learns share one
    # build workdir and run in parallel, so a workdir-derived stage path would race and
    # cross-contaminate the seed vaults.
    learn_vault = args.vault_out + ".staging"
    shutil.rmtree(learn_vault, ignore_errors=True)
    if args.vault_in != "none" and os.path.isdir(args.vault_in):
        shutil.copytree(args.vault_in, learn_vault)
    else:
        os.makedirs(os.path.join(learn_vault, "Permanent"), exist_ok=True)

    stated = []
    if args.build_result and os.path.exists(args.build_result):
        try:
            stated = json.load(open(args.build_result)).get("stated_conventions", [])
        except Exception:
            stated = []

    if args.stub:
        lr = _stub_learn(args, learn_vault)
    else:
        lr = claude(args.cfg, args.model, learn_vault, args.workdir, learn_prompt(args.write_tier, stated))
        if lr.get("is_error") and (lr.get("total_cost_usd", 0) or 0) == 0:
            time.sleep(5)
            lr = claude(args.cfg, args.model, learn_vault, args.workdir, learn_prompt(args.write_tier, stated))

    shutil.rmtree(args.vault_out, ignore_errors=True)
    shutil.copytree(learn_vault, args.vault_out)
    by_tier = count_notes_by_tier(args.vault_out)

    out = {
        "schema_version": SCHEMA_VERSION, "engram_sha": engram_sha(), "kind": "learn",
        "app": args.app, "model": args.model, "model_id": MODELS[args.model],
        "regime": args.regime, "trial": args.trial, "date": args.date,
        "write_tier": args.write_tier, "vault_in": args.vault_in, "vault_out": args.vault_out,
        "stub": args.stub or None, "stated_conventions_input": stated, "learned": True,
        "notes_total": len(glob_notes(args.vault_out)), "notes_by_tier": by_tier,
        "learn_cost": round(lr.get("total_cost_usd", 0) or 0, 4), "learn_turns": lr.get("num_turns", 0) or 0,
        "result": (lr.get("result") or "")[:300], "wall_min": round((time.time() - t0) / 60.0, 1),
    }
    json.dump(out, open(args.out, "w"), indent=2)
    print(json.dumps({"app": args.app, "regime": args.regime, "write_tier": args.write_tier,
          "notes_by_tier": by_tier, "learn_cost": out["learn_cost"]}))


def main():
    ap = argparse.ArgumentParser()
    sub = ap.add_subparsers(dest="mode", required=True)

    b = sub.add_parser("build")
    for a in ["app", "model", "regime", "vault-in", "cfg", "workdir", "spec", "out"]:
        b.add_argument("--" + a, required=True)
    b.add_argument("--trial", type=int, default=1)
    b.add_argument("--date", default="")
    b.add_argument("--max-rounds", type=int, default=6)
    b.add_argument("--stub", default="", choices=["", "good", "naive"])

    le = sub.add_parser("learn")
    for a in ["app", "model", "regime", "write-tier", "workdir", "vault-in", "vault-out", "out"]:
        le.add_argument("--" + a, required=True)
    le.add_argument("--trial", type=int, default=1)
    le.add_argument("--date", default="")
    le.add_argument("--cfg", default="")
    le.add_argument("--build-result", default="")
    le.add_argument("--stub", default="", choices=["", "good", "naive"])

    args = ap.parse_args()
    # argparse stores hyphenated flags with underscores; normalize the few we read by attr.
    args.vault_in = getattr(args, "vault_in")
    if args.mode == "build":
        run_build(args)
    else:
        args.vault_out = getattr(args, "vault_out")
        args.write_tier = getattr(args, "write_tier")
        run_build_result = getattr(args, "build_result", "")
        args.build_result = run_build_result
        run_learn(args)


if __name__ == "__main__":
    main()
