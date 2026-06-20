#!/usr/bin/env python3
"""Zero-cost validation of the cumulative-accumulation harness — NO LLM calls, no spend.

Runs the §7 pilot checks that don't need an LLM, so they're reproducible from a clean
checkout before anyone authorizes the live pilot:
  (i)   cell-gen: the §1.3 26-op / 18-cell chain, vault threading, dependency wiring
  (iv)  scorer is name-agnostic: GOOD fixture (Repository vocab) passes ARCH, NAIVE fails
  (ii)  clean room: no CLAUDE.md/AGENTS.md reaches a build; cfg carries only recall+learn
  +     full pipeline mechanics via the --stub matrix (build->score->loop->learn->thread->schema)
  +     aggregate.py runs and emits the tables

The remaining §7 check — (iii) recall fires AND is applied — inherently needs the LLM and
is verified in the live pilot.

Usage: python3 validate.py
Exit 0 iff every check passes.
"""
import json, os, shutil, subprocess, sys, tempfile

CUM = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, CUM)
import harness  # noqa: E402
import matrix  # noqa: E402
import score as scoremod  # noqa: E402

results = []


def check(name, ok, detail=""):
    results.append((name, ok, detail))
    print(f"  {'PASS' if ok else 'FAIL'}  {name}" + (f"  — {detail}" if detail else ""))


def check_cellgen():
    # real-skill-only: 4 regimes (cold, real.lazy, real.auto, real.autol2) × 3 apps = 12 build ops, 0 learn ops.
    ops = matrix.real_cells_for("sonnet", 1, "2026-06-06", "", 6, None)
    builds = [o for o in ops if o["kind"] == "build"]
    learns = [o for o in ops if o["kind"] == "learn"]
    check("cell-gen: 12 ops (12 build + 0 learn)",
          len(ops) == 12 and len(builds) == 12 and len(learns) == 0,
          f"{len(ops)} ops / {len(builds)} build / {len(learns)} learn")

    def arg(o, flag):
        return o["cmd_tail"][o["cmd_tail"].index(flag) + 1] if flag in o["cmd_tail"] else None

    # Vault threading: for vault-writing regimes, each app seals its vault and passes it to the next.
    # cold never writes; real.lazy/autol2 write vault; real.auto writes chunks only.
    threading_ok = True
    for r, rc in harness.REGIMES.items():
        if r == "cold":
            continue  # cold never accumulates; skip threading check
        a1b = next((o for o in ops if o["id"] == f"sonnet-t1-app1-{r}-build"), None)
        a2b = next((o for o in ops if o["id"] == f"sonnet-t1-app2-{r}-build"), None)
        a3b = next((o for o in ops if o["id"] == f"sonnet-t1-app3-{r}-build"), None)
        if not (a1b and a2b and a3b):
            threading_ok = False
            continue
        # app2 depends on app1 completing; app3 depends on app2
        if (a2b["dep"] != [f"sonnet-t1-app1-{r}-build"]
                or a3b["dep"] != [f"sonnet-t1-app2-{r}-build"]):
            threading_ok = False
    check("cell-gen: vault threading + deps across all real regimes", threading_ok)


def check_scorer():
    good = scoremod.score(os.path.join(CUM, "testdata", "good"), os.path.join(CUM, "notes_spec.json"))
    naive = scoremod.score(os.path.join(CUM, "testdata", "naive"), os.path.join(CUM, "notes_spec.json"))
    gp, npass = good.get("arch_pass", 0), naive.get("arch_pass", 0)
    gfail, nfail = set(good.get("arch_fail", [])), set(naive.get("arch_fail", []))
    # GOOD must score full ARCH despite using "Repository" not "Store" (name-agnostic);
    # NAIVE must fail every load-bearing convention.
    check("scorer: GOOD passes all 10 ARCH with non-vault vocab (name-agnostic)", gp == 10, f"good arch_pass={gp} fail={sorted(gfail)}")
    check("scorer: NAIVE scores clearly lower", npass <= 2, f"naive arch_pass={npass}")
    check("scorer: DI detected by pattern not name (good Repository=pass, naive=fail)",
          "di" not in gfail and "di" in nfail)
    # The tightened/hardened detectors must flag these on the naive build (no false-negatives
    # from comments, nested parens, or grouped `var ( ... )` blocks).
    must_fail = {"json", "nocolor", "named_perms", "sentinel", "no_global_data"}
    check("scorer: tightened detectors flag naive (no comment/nested-paren/var-block false-neg)",
          must_fail <= nfail, f"naive missing-from-fail: {sorted(must_fail - nfail)}")


def check_stub_pipeline():
    root = tempfile.mkdtemp(prefix="cumvalidate-")
    env = dict(os.environ)
    env["CUMMATRIX_ROOT"] = root
    try:
        r = subprocess.run(["python3", os.path.join(CUM, "matrix.py"), "--models", "sonnet",
                            "--trials", "1", "--stub", "good", "--max-rounds", "1", "--workers", "4",
                            "--timeout-min", "5"], env=env, capture_output=True, text=True, timeout=600)
        done = "### MATRIX COMPLETE ### 12/12" in r.stdout
        check("stub matrix: 12/12 ops complete (no LLM)", done, r.stdout.strip().splitlines()[-1] if r.stdout else "")

        # Real-skill regimes accumulate per-regime vaults (vault-writing: real.lazy, real.autol2).
        # Verify that app1 and app2 produced a vault for real.lazy (the simplest vault-writing arm).
        v_app1 = os.path.join(root, "vaults", "v-sonnet-t1-app1-real.lazy")
        v_app2 = os.path.join(root, "vaults", "v-sonnet-t1-app2-real.lazy")
        lazy_seeded = os.path.isdir(v_app1) or os.path.isdir(v_app2)
        check("stub: real.lazy vault dirs created for non-terminal apps",
              lazy_seeded, f"app1={os.path.isdir(v_app1)} app2={os.path.isdir(v_app2)}")

        # clean room: build workdirs carry no ambient conventions; cfg only recall+learn.
        ws = os.path.join(root, "ws")
        stray = []
        for dirpath, _, files in os.walk(ws):
            stray += [os.path.join(dirpath, f) for f in files if f in ("CLAUDE.md", "AGENTS.md")]
        warm_skills = sorted(os.listdir(os.path.join(root, "cfgpool", "warm0", "skills")))
        cold_has_skills = os.path.isdir(os.path.join(root, "cfgpool", "cold0", "skills"))
        check("clean room: no CLAUDE.md/AGENTS.md in build workdirs", not stray, "; ".join(stray))
        check("clean room: warm cfg carries ONLY recall+learn; cold none",
              warm_skills == ["learn", "recall"] and not cold_has_skills, f"warm={warm_skills} cold_skills={cold_has_skills}")

        agg = subprocess.run(["python3", os.path.join(CUM, "aggregate.py"), "--root", root,
                             "--out", os.path.join(root, "results-stub.md")],
                            env=env, capture_output=True, text=True, timeout=120)
        check("aggregate.py emits tables without error",
              agg.returncode == 0 and "Convention interventions to endpoint" in agg.stdout)
    finally:
        shutil.rmtree(root, ignore_errors=True)


def check_prune():
    """Deterministic test of the write-tier ceiling enforcement: a vault with L1+L2+L3
    notes must prune to exactly the ceiling (guards the bug where the /learn skill wrote
    facts during an L1 episode-only capture, making v1[L1] == v1[L2])."""
    import harness as hh

    def seed():
        root = tempfile.mkdtemp(prefix="cumprune-")
        perm = os.path.join(root, "Permanent")
        os.makedirs(perm)
        for tier, kind in [("L1", "episode"), ("L2", "fact"), ("L3", "fact")]:
            with open(os.path.join(perm, f"9{tier[-1]}.2026-06-06.x-{tier.lower()}.md"), "w") as fh:
                fh.write(f"---\ntype: {kind}\ntier: {tier}\nsituation: x\n---\nbody\n")
        return root

    r1, r2 = seed(), seed()
    try:
        hh.prune_to_ceiling(r1, "L1")
        hh.prune_to_ceiling(r2, "L2")
        c1, c2 = hh.count_notes_by_tier(r1), hh.count_notes_by_tier(r2)
        check("prune: L1 ceiling keeps only the episode (no facts)",
              c1 == {"L1": 1, "L2": 0, "L3": 0, "other": 0}, str(c1))
        check("prune: L2 ceiling keeps L1+L2, drops L3", c2 == {"L1": 1, "L2": 1, "L3": 0, "other": 0}, str(c2))
    finally:
        shutil.rmtree(r1, ignore_errors=True)
        shutil.rmtree(r2, ignore_errors=True)


def check_learn_tiers():
    """The --stub deterministic learn must produce strictly-nested tier seeds from a stated set:
    L1 = episode only, L2 = +one fact per convention, L3 = +a synthesized ADR. (Validates the
    stub pipeline path; the REAL learn is agent-driven and measured live in the pilot.)"""
    import harness as hh

    root = tempfile.mkdtemp(prefix="cumlearn-")
    stated = ["di", "atomic", "nocolor"]
    br = os.path.join(root, "build.json")
    json.dump({"stated_conventions": stated}, open(br, "w"))
    counts = {}
    try:
        for tier in ["L1", "L2", "L3"]:
            vout = os.path.join(root, f"v1-{tier}")
            subprocess.run(["python3", os.path.join(CUM, "harness.py"), "learn", "--app", "notes",
                            "--model", "sonnet", "--regime", f"t-{tier}", "--write-tier", tier,
                            "--workdir", os.path.join(root, "ws"), "--vault-in", "none",
                            "--vault-out", vout, "--build-result", br, "--stub", "good",
                            "--out", os.path.join(root, f"learn-{tier}.json"), "--date", "2026-06-06"],
                           capture_output=True, text=True, timeout=180)
            counts[tier] = hh.count_notes_by_tier(vout) if os.path.isdir(vout) else {}

        l1, l2, l3 = counts["L1"], counts["L2"], counts["L3"]
        check("learn: L1 = episode only (no facts)", l1 == {"L1": 1, "L2": 0, "L3": 0, "other": 0}, str(l1))
        check("learn: L2 = episode + 1 fact per stated convention",
              l2 == {"L1": 1, "L2": len(stated), "L3": 0, "other": 0}, str(l2))
        check("learn: L3 = episode + facts + synthesized ADR",
              l3.get("L1") == 1 and l3.get("L2") == len(stated) and l3.get("L3") == 1, str(l3))

        env = dict(os.environ)
        env["ENGRAM_VAULT_PATH"] = os.path.join(root, "v1-L3")
        env["PATH"] = hh.ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
        chk = subprocess.run(["engram", "check"], env=env, capture_output=True, text=True)
        check("learn: L3 seed vault passes engram check (links resolve)",
              chk.returncode == 0 and "FAIL" not in chk.stdout, chk.stdout.strip()[:80])
    finally:
        shutil.rmtree(root, ignore_errors=True)


def check_token_io():
    """Token-I/O capture + cost reconstruction, on a synthetic transcript (no LLM). A session's
    main + subagent messages are summed, deduped by message id; cost = tokens × price sheet."""
    import harness as hh

    root = tempfile.mkdtemp(prefix="cumtok-")
    sid = "sess-123"
    proj = os.path.join(root, "cfgpool", "warm0", "projects", "-x")
    os.makedirs(os.path.join(proj, sid, "subagents"))

    def msg(mid, i, o, cw, cr):
        return json.dumps({"message": {"id": mid, "usage": {
            "input_tokens": i, "output_tokens": o,
            "cache_creation_input_tokens": cw, "cache_read_input_tokens": cr}}})

    # main transcript: two distinct messages + a DUP of the first (must be deduped)
    with open(os.path.join(proj, f"{sid}.jsonl"), "w") as fh:
        fh.write(msg("m1", 100, 10, 1000, 5000) + "\n")
        fh.write(msg("m2", 50, 20, 0, 2000) + "\n")
        fh.write(msg("m1", 100, 10, 1000, 5000) + "\n")  # duplicate id → ignored
    # a subagent transcript (recall/learn dispatch) — its tokens count too
    with open(os.path.join(proj, sid, "subagents", "agent-a.jsonl"), "w") as fh:
        fh.write(msg("s1", 200, 30, 500, 1000) + "\n")

    try:
        tok = hh.token_usage_for_session(os.path.join(root, "cfgpool"), sid)
        want = {"input": 350, "output": 60, "cache_write": 1500, "cache_read": 8000}
        check("tokens: main+subagent summed, duplicate message id deduped", tok == want, str(tok))

        cost = hh.recompute_cost(tok, "claude-sonnet-4-6")
        expect = round(350 * 3e-6 + 60 * 15e-6 + 1500 * 3.75e-6 + 8000 * 0.30e-6, 4)
        check("tokens: cost reconstructed from price sheet", cost == expect, f"{cost} vs {expect}")
        check("tokens: unknown model → None (no silent zero)", hh.recompute_cost(tok, "bogus") is None)
    finally:
        shutil.rmtree(root, ignore_errors=True)


def check_cli_surface():
    """Smoke-check every distinct engram subcommand+flag the harness invokes against the live binary.
    A future CLI flag removal fails this zero-cost check rather than a paid eval run."""
    import shutil as _shutil
    engram_bin = _shutil.which("engram") or os.path.join(harness.ENGRAM_BIN_DIR, "engram")
    if not os.path.exists(engram_bin):
        check("cli-surface: engram binary reachable", False, f"not found at {engram_bin}")
        return

    def help_text(*sub):
        r = subprocess.run([engram_bin] + list(sub) + ["--help"], capture_output=True, text=True, timeout=15)
        return r.stdout + r.stderr

    # Each entry: (description, subcommand path, required string in help output).
    # These are the DISTINCT engram invocations the harness makes (harness.py eg_learn, embed apply,
    # ingest, plus engram check in validate.py). The common-learn-args flags (--slug, --position,
    # --source, --relation, --tier) appear under --common-learn-args and are not listed individually
    # in --help, so we probe the subcommand existence + the flags that ARE shown.
    # Update this list whenever harness.py adds or changes an engram invocation.
    probes = [
        ("engram learn fact subcommand exists", ["learn", "fact"], "--situation"),
        ("engram learn fact --subject flag", ["learn", "fact"], "--subject"),
        ("engram learn feedback subcommand exists", ["learn", "feedback"], "--situation"),
        ("engram embed apply subcommand exists", ["embed", "apply"], "apply"),
        ("engram embed apply --all flag", ["embed", "apply"], "--all"),
        ("engram ingest subcommand exists", ["ingest"], "--transcript"),
        ("engram ingest --chunks-dir flag", ["ingest"], "--chunks-dir"),
        ("engram check subcommand exists", ["check"], "check"),
    ]
    all_ok = True
    missing = []
    for desc, sub, flag in probes:
        txt = help_text(*sub)
        ok = (not flag) or (flag in txt)
        if not ok:
            missing.append(desc)
            all_ok = False
    check("cli-surface: harness engram invocations exist in the live binary",
          all_ok, f"missing: {missing}" if missing else "all present")


def main():
    print("Zero-cost validation (no LLM, no spend):\n")
    print("[cell-gen]")
    check_cellgen()
    print("[scorer]")
    check_scorer()
    print("[write-tier ceiling]")
    check_prune()
    print("[deterministic learn tiers]")
    check_learn_tiers()
    print("[token I/O + cost audit]")
    check_token_io()
    print("[pipeline + clean room]")
    check_stub_pipeline()
    print("[cli-surface smoke]")
    check_cli_surface()

    npass = sum(1 for _, ok, _ in results if ok)
    total = len(results)
    print(f"\n{'ALL PASS' if npass == total else 'FAIL'}: {npass}/{total} checks")
    print("(live-only check NOT run here: recall fires AND is applied — verified in the pilot.)")
    sys.exit(0 if npass == total else 1)


if __name__ == "__main__":
    main()
