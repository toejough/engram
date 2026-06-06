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

        counts = {t: count(t) for t in ["none", "L1", "L2", "L3"]}
        check("stub: cumulative write-tier seed vaults (none0/L1=1/L2=2/L3=3)",
              counts == {"none": 0, "L1": 1, "L2": 2, "L3": 3}, str(counts))

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


def main():
    print("Zero-cost validation (no LLM, no spend):\n")
    print("[cell-gen]")
    check_cellgen()
    print("[scorer]")
    check_scorer()
    print("[pipeline + clean room]")
    check_stub_pipeline()

    npass = sum(1 for _, ok, _ in results if ok)
    total = len(results)
    print(f"\n{'ALL PASS' if npass == total else 'FAIL'}: {npass}/{total} checks")
    print("(live-only check NOT run here: recall fires AND is applied — verified in the pilot.)")
    sys.exit(0 if npass == total else 1)


if __name__ == "__main__":
    main()
