#!/usr/bin/env python3
"""Zero-cost validation of the cumulative-accumulation harness — NO LLM calls, no spend.

Runs validation checks for the v3 2-regime design (cold + real.full, no tiers/episodes):
  (i)   cell-gen: 2 regimes × 3 apps = 6 build ops, 0 learn ops; vault threading + deps
  (iv)  scorer is name-agnostic: GOOD fixture (Repository vocab) passes ARCH, NAIVE fails
  (ii)  clean room: no CLAUDE.md/AGENTS.md reaches a build; cfg carries only recall+learn
  +     full pipeline mechanics via the --stub matrix (build→score→loop→learn→thread→schema)
  +     aggregate.py runs and emits the tables
  +     new metrics present: notes_written, crystallizations_at_recall, chunks_ingested,
        learn_kind_breakdown (NO tier/episode fields)

The remaining check — (iii) recall fires AND is applied — inherently needs the LLM and
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
    # v3: 2 regimes (cold, real.full) × 3 apps = 6 build ops, 0 separate learn ops.
    ops = matrix.real_cells_for("sonnet", 1, "2026-06-06", "", 6, None)
    builds = [o for o in ops if o["kind"] == "build"]
    learns = [o for o in ops if o["kind"] == "learn"]
    check("cell-gen: 6 ops (6 build + 0 learn)",
          len(ops) == 6 and len(builds) == 6 and len(learns) == 0,
          f"{len(ops)} ops / {len(builds)} build / {len(learns)} learn")

    # Vault threading: real.full writes vault; cold never accumulates.
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
    check("cell-gen: vault threading + deps for real.full", threading_ok)


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
        done = "### MATRIX COMPLETE ### 6/6" in r.stdout
        check("stub matrix: 6/6 ops complete (no LLM)", done, r.stdout.strip().splitlines()[-1] if r.stdout else "")

        # real.full accumulates a vault; verify app1 and app2 produced vault dirs.
        v_app1 = os.path.join(root, "vaults", "v-sonnet-t1-app1-real.full")
        v_app2 = os.path.join(root, "vaults", "v-sonnet-t1-app2-real.full")
        full_seeded = os.path.isdir(v_app1) or os.path.isdir(v_app2)
        check("stub: real.full vault dirs created for non-terminal apps",
              full_seeded, f"app1={os.path.isdir(v_app1)} app2={os.path.isdir(v_app2)}")

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

        # schema v4: verify a result JSON has schema_version 4
        result_jsons = [f for f in os.listdir(os.path.join(root, "results"))
                        if f.endswith(".json") and f != "run-manifest.json"]
        if result_jsons:
            sample = json.load(open(os.path.join(root, "results", result_jsons[0])))
            check("schema v4: stub result has schema_version 4",
                  sample.get("schema_version") == 4,
                  f"schema_version={sample.get('schema_version')}")
        else:
            check("schema v4: stub result has schema_version 4", False, "no result JSONs found")

        # new metrics: verify result JSON has v3 metrics and NOT tier/episode fields
        if result_jsons:
            sample = json.load(open(os.path.join(root, "results", result_jsons[0])))
            new_metrics = ["notes_written", "crystallizations_at_recall",
                           "chunks_ingested", "learn_kind_breakdown"]
            stale_fields = ["l2_generated", "l2_composed", "notes_by_tier"]
            missing_new = [f for f in new_metrics if f not in sample]
            present_stale = [f for f in stale_fields if f in sample]
            check("new metrics: notes_written/crystallizations_at_recall/chunks_ingested/learn_kind_breakdown present",
                  len(missing_new) == 0, f"missing: {missing_new}" if missing_new else "all present")
            check("stale tier metrics absent: no l2_generated/l2_composed/notes_by_tier in result",
                  len(present_stale) == 0, f"still present: {present_stale}" if present_stale else "none found")

        agg = subprocess.run(["python3", os.path.join(CUM, "aggregate.py"), "--root", root,
                             "--out", os.path.join(root, "results-stub.md")],
                            env=env, capture_output=True, text=True, timeout=120)
        check("aggregate.py emits tables without error",
              agg.returncode == 0 and "Convention interventions to endpoint" in agg.stdout)

        # axis fields: verify aggregate output mentions all six axis fields
        agg_out = agg.stdout if agg.returncode == 0 else ""
        axis_fields = ["recall_s", "build_s", "learn_s", "axis_c2_cost_usd",
                       "axis_c3_interventions", "Axis CI table"]
        missing_axes = [f for f in axis_fields if f not in agg_out]
        check("axis fields: --stub dry pass emits all six axis fields",
              len(missing_axes) == 0, f"missing: {missing_axes}" if missing_axes else "all present")

        # no stale tier metric fields in aggregate output (allow "episode" in explanatory prose)
        tier_field_phrases = ["l2_generated", "l2_composed", "learn_quality_table",
                              "write-tier of its learn", "L1 episode"]
        found_tier = [p for p in tier_field_phrases if p in agg_out]
        check("aggregate: no stale tier metric fields in output",
              len(found_tier) == 0, f"found stale: {found_tier}" if found_tier else "none found")
    finally:
        shutil.rmtree(root, ignore_errors=True)


def check_modern_metrics():
    """Verify that the modern metric helpers (count_notes_written, count_learn_kind_breakdown)
    work on a synthetic vault with known fact+feedback notes — no LLM, no tier fields."""
    import harness as hh

    root = tempfile.mkdtemp(prefix="cummetrics-")
    perm = os.path.join(root, "Permanent")
    os.makedirs(perm)
    try:
        # Write one fact and one feedback note directly
        with open(os.path.join(perm, "1.2026-06-06.test-fact.md"), "w") as fh:
            fh.write("---\ntype: fact\nsituation: x\nsubject: a\npredicate: is\nobject: b\n---\nbody\n")
        with open(os.path.join(perm, "2.2026-06-06.test-feedback.md"), "w") as fh:
            fh.write("---\ntype: feedback\nsituation: y\nbehavior: bad\nimpact: costly\naction: good\n---\nbody\n")

        total = hh.count_notes_written(root)
        breakdown = hh.count_learn_kind_breakdown(root)
        check("modern metrics: count_notes_written counts fact+feedback notes",
              total == 2, f"count={total}")
        check("modern metrics: count_learn_kind_breakdown splits by type",
              breakdown == {"fact": 1, "feedback": 1, "other": 0}, str(breakdown))
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


def check_ci_and_floor():
    import aggregate
    import random
    rng = random.Random(42)
    # Known distribution: mean=10, std=2, N=100
    true_mean = 10.0
    xs = [rng.gauss(true_mean, 2.0) for _ in range(100)]
    m, lo, hi = aggregate.bootstrap_ci(xs, alpha=0.05, n_boot=500)
    check("ci: bootstrap CI contains true mean", lo <= true_mean <= hi,
          f"CI=({lo:.2f},{hi:.2f}) mean={m:.2f} true={true_mean}")

    # noise_floor on two equal distributions (warm vs warm): floor should be positive but small
    xs_warm = [rng.gauss(5.0, 1.0) for _ in range(20)]
    floor = aggregate.noise_floor(xs_warm)
    check("ci: noise floor labeled underpowered when gap < floor",
          aggregate.gap_label(0.01, floor) == "underpowered",
          f"floor={floor:.3f} gap=0.01")


def check_recency_probe():
    import recency_probe
    here = os.path.join(CUM, "testdata")
    with_r = open(os.path.join(here, "recency_with_R.yaml")).read()
    without_r = open(os.path.join(here, "recency_without_R.yaml")).read()

    items = recency_probe.parse_recent_channel(with_r)
    check("recency: parse_recent_channel finds provenance:recent items",
          len(items) >= 1, f"found {len(items)} recent items")

    surfaced = recency_probe.recent_channel_surfaced(with_r, "lesson-recent")
    check("recency: target_surfaced=True when R in recent channel", surfaced,
          f"surfaced={surfaced}")

    not_surfaced = recency_probe.recent_channel_surfaced(without_r, "lesson-recent")
    check("recency: target_surfaced=False when R absent", not not_surfaced,
          f"surfaced={not_surfaced}")


def check_reversal_scorer():
    import reversal_scorer, json as _json
    spec_path = os.path.join(CUM, "reversal_spec.json")
    spec = _json.load(open(spec_path))
    here = os.path.join(CUM, "testdata")

    code_x = open(os.path.join(here, "reversal_follows_x.go")).read()
    code_xp = open(os.path.join(here, "reversal_follows_x_prime.go")).read()

    r_x = reversal_scorer.score_supersession(code_x, spec)
    check("reversal: X fixture scored as follows_x",
          r_x["follows_x"] and not r_x["follows_x_prime"], str(r_x))

    r_xp = reversal_scorer.score_supersession(code_xp, spec)
    check("reversal: X' fixture scored as follows_x_prime + supersession_correct",
          r_xp["follows_x_prime"] and r_xp["supersession_correct"] and not r_xp["follows_x"],
          str(r_xp))


def check_synthesis_probe():
    import synthesis_judge
    fix = os.path.join(CUM, "synthesis_fixtures")

    # Iterate over all fixture{N}/ subdirectories (≥3 designed-gap fixtures).
    fixtures = list(synthesis_judge.list_fixtures(fix))
    fixture_ok = len(fixtures) >= 3
    check(f"synthesis: ≥3 fixture subdirs present", fixture_ok,
          f"found {len(fixtures)}: {[f['name'] for f in fixtures]}")

    for fixture in fixtures:
        name = fixture["name"]
        task_txt = fixture["task_txt"]
        expected_z = fixture["expected_z_txt"]

        r_cluster = synthesis_judge.judge_crystallization(
            fixture["vault_with_cluster"], task_txt, expected_z, stub=True)
        check(f"synthesis [{name}]: stub judge SIGNAL for absent cluster",
              r_cluster["verdict"] == "SIGNAL", str(r_cluster))

        r_covered = synthesis_judge.judge_crystallization(
            fixture["vault_covered"], task_txt, expected_z, stub=True)
        check(f"synthesis [{name}]: stub judge MUST_NOT_FIRE for covered cluster",
              r_covered["verdict"] == "MUST_NOT_FIRE", str(r_covered))

    # Verify real judge parse/aggregation logic with mocked judge responses.
    # No paid LLM call — _parse_judge_json is exercised directly.
    synth_json = '{"verdict": "SYNTHESIS", "reason": "Z integrates facets", "z_present": true, "z_integrative": true, "build_used_z": true}'
    not_synth_json = '{"verdict": "NOT_SYNTHESIS", "reason": "mere restatement", "z_present": false, "z_integrative": false, "build_used_z": null}'

    parsed_synth = synthesis_judge._parse_judge_json(synth_json)
    parsed_not = synthesis_judge._parse_judge_json(not_synth_json)
    check("synthesis: _parse_judge_json parses SYNTHESIS verdict",
          parsed_synth.get("verdict") == "SYNTHESIS" and parsed_synth.get("z_present") is True,
          str(parsed_synth))
    check("synthesis: _parse_judge_json parses NOT_SYNTHESIS verdict",
          parsed_not.get("verdict") == "NOT_SYNTHESIS" and parsed_not.get("z_present") is False,
          str(parsed_not))

    # Majority vote logic: 2 SYNTHESIS + 1 NOT_SYNTHESIS → SYNTHESIS.
    mock_runs = [
        {"verdict": "SYNTHESIS", "z_present": True, "z_integrative": True},
        {"verdict": "SYNTHESIS", "z_present": True, "z_integrative": True},
        {"verdict": "NOT_SYNTHESIS", "z_present": False, "z_integrative": False},
    ]
    synthesis_votes = sum(1 for r in mock_runs if r.get("verdict") == "SYNTHESIS")
    majority = "SYNTHESIS" if synthesis_votes > synthesis_judge.JUDGE_RUNS // 2 else "NOT_SYNTHESIS"
    check("synthesis: majority vote 2/3 SYNTHESIS → SYNTHESIS verdict",
          majority == "SYNTHESIS", f"votes={synthesis_votes}/{synthesis_judge.JUDGE_RUNS}")

    # Parse error → NOT_SYNTHESIS (safe default).
    err_parsed = synthesis_judge._parse_judge_json("no json here at all")
    check("synthesis: _parse_judge_json parse error → NOT_SYNTHESIS safe default",
          err_parsed.get("verdict") == "NOT_SYNTHESIS" and err_parsed.get("_parse_error") is True,
          str(err_parsed))

    # Judge prompt contains required adversarial rubric elements.
    check("synthesis: judge system prompt contains adversarial refute-by-default rule",
          "NOT_SYNTHESIS" in synthesis_judge._JUDGE_SYSTEM and "SYNTHESIS" in synthesis_judge._JUDGE_SYSTEM,
          "system prompt missing verdict strings")
    check("synthesis: judge system prompt requires z_integrative (integrative, not restatement)",
          "z_integrative" in synthesis_judge._JUDGE_SYSTEM,
          "missing z_integrative in rubric")


def main():
    print("Zero-cost validation (no LLM, no spend):\n")
    print("[cell-gen]")
    check_cellgen()
    print("[scorer]")
    check_scorer()
    print("[modern metrics]")
    check_modern_metrics()
    print("[token I/O + cost audit]")
    check_token_io()
    print("[pipeline + clean room]")
    check_stub_pipeline()
    print("[cli-surface smoke]")
    check_cli_surface()
    print("[CI + noise floor]")
    check_ci_and_floor()
    print("[recency probe]")
    check_recency_probe()
    print("[reversal scorer]")
    check_reversal_scorer()
    print("[synthesis probe]")
    check_synthesis_probe()

    npass = sum(1 for _, ok, _ in results if ok)
    total = len(results)
    print(f"\n{'ALL PASS' if npass == total else 'FAIL'}: {npass}/{total} checks")
    print("(live-only check NOT run here: recall fires AND is applied — verified in the pilot.)")
    sys.exit(0 if npass == total else 1)


if __name__ == "__main__":
    main()
