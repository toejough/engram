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
    ops = matrix.cells_for("sonnet", 1, "2026-06-06", "", 6)
    builds = [o for o in ops if o["kind"] == "build"]
    learns = [o for o in ops if o["kind"] == "learn"]
    check("cell-gen: 26 ops (15 build + 11 learn)",
          len(ops) == 26 and len(builds) == 15 and len(learns) == 11,
          f"{len(ops)} ops / {len(builds)} build / {len(learns)} learn")

    def arg(o, flag):
        return o["cmd_tail"][o["cmd_tail"].index(flag) + 1] if flag in o["cmd_tail"] else None

    threading_ok = True
    for r, rc in harness.REGIMES.items():
        a2b = next(o for o in ops if o["id"].endswith(f"app2-{r}-build"))
        a2l = next(o for o in ops if o["id"].endswith(f"app2-{r}-learn"))
        a3b = next(o for o in ops if o["id"].endswith(f"app3-{r}-build"))
        if not (arg(a2b, "--vault-in").endswith(f"v1-sonnet-t1-{rc['write']}")
                and arg(a2l, "--vault-out").endswith(f"v2-sonnet-t1-{r}")
                and arg(a3b, "--vault-in").endswith(f"v2-sonnet-t1-{r}")
                and a2b["dep"] == [f"sonnet-t1-app1-learn-{rc['write']}"]):
            threading_ok = False
    check("cell-gen: vault threading + deps across all 7 regimes", threading_ok)


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
        done = "### MATRIX COMPLETE ### 26/26" in r.stdout
        check("stub matrix: 26/26 ops complete (no LLM)", done, r.stdout.strip().splitlines()[-1] if r.stdout else "")

        def count(tier):
            d = os.path.join(root, "vaults", f"v1-sonnet-t1-{tier}", "Permanent")
            return len([f for f in os.listdir(d) if f.endswith(".md")]) if os.path.isdir(d) else 0

        # The good fixture converges with 0 stated conventions, so the deterministic learn
        # writes episode-only for L1/L2 and episode+ADR for L3 — monotonic, none empty. (The
        # strict fact-per-convention nesting is exercised by check_learn_tiers below.)
        counts = {t: count(t) for t in ["none", "L1", "L2", "L3"]}
        check("stub: write-tier seeds monotonic (none=0 ≤ L1 ≤ L2 ≤ L3)",
              counts["none"] == 0 and 1 <= counts["L1"] <= counts["L2"] <= counts["L3"], str(counts))

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
    """The deterministic learn must produce strictly-nested tier seeds from a stated-convention
    set: L1 = episode only, L2 = +one fact per convention, L3 = +a synthesized ADR. Zero LLM
    (the learn is harness-driven), so this is the advisor's per-tier micro-test made free."""
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
                            "--vault-out", vout, "--build-result", br,
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
    print("[pipeline + clean room]")
    check_stub_pipeline()

    npass = sum(1 for _, ok, _ in results if ok)
    total = len(results)
    print(f"\n{'ALL PASS' if npass == total else 'FAIL'}: {npass}/{total} checks")
    print("(live-only check NOT run here: recall fires AND is applied — verified in the pilot.)")
    sys.exit(0 if npass == total else 1)


if __name__ == "__main__":
    main()
