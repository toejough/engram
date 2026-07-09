"""Unit tests for run_recheck.py's pure core (offline — no LLM, no claude -p, no harness/matrix
import). Covers arm-matrix expansion, fixture-prompt construction (context.md + task), the arm-C
gate, checkpoint/resume, the per-trial validity gate + retry cap (with the NOT-RED cap-exhausted
record), the cost tally, and fail-loud stub config. Arm-B advocacy scoring lives in
lever_recheck_scorer (tested in test_lever_recheck_scorer.py)."""
import json
import os

import pytest

import run_recheck as rr


def _write(path, text):
    with open(path, "w") as fh:
        fh.write(text)


def _make_fixture(tmp_path, name, diagnostic=False, lever_terms="cheaper,retrieval",
                  note_basename="8.2026-06-20.cheap-retrieval-model-rolled-back",
                  lever_id="cheap-retrieval-model"):
    fdir = tmp_path / name
    fdir.mkdir()
    _write(str(fdir / "task.txt"), "consult-memory framing task")
    if diagnostic:
        _write(str(fdir / "task_diagnostic.txt"), "neutral diagnose-and-recommend framing task")
    levers = [{
        "id": lever_id,
        "canonical_action": "run the retrieval / context-fetch step on a cheaper model",
        "closure_reason": "retrieval is a small share of cost",
        "measured_outcome": "-14%, rolled back",
        "note_basename": note_basename,
    }]
    if lever_terms is not None:
        levers[0]["lever_terms"] = lever_terms
    _write(str(fdir / "closed_levers.json"), json.dumps(levers))
    return str(fdir)


# ---- arm-matrix expansion ----

def test_arm_vault_mapping():
    assert rr.ARM_VAULT["A"] == "vault_with_closed"
    assert rr.ARM_VAULT["B"] == "vault_open"
    assert rr.ARM_VAULT["C"] == "vault_with_closed"


def test_arm_task_file_prefers_diagnostic_for_a_and_b(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", diagnostic=True)
    assert rr.arm_task_file(fdir, "A") == "task_diagnostic.txt"
    assert rr.arm_task_file(fdir, "B") == "task_diagnostic.txt"


def test_arm_task_file_falls_back_to_task_when_no_diagnostic(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", diagnostic=False)
    assert rr.arm_task_file(fdir, "A") == "task.txt"
    assert rr.arm_task_file(fdir, "B") == "task.txt"


def test_arm_task_file_c_always_uses_task_even_if_diagnostic_exists(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", diagnostic=True)
    assert rr.arm_task_file(fdir, "C") == "task.txt"


def test_expand_matrix_produces_fixture_arm_trial_tuples(tmp_path):
    f1 = _make_fixture(tmp_path, "fixture1", diagnostic=True)
    f2 = _make_fixture(tmp_path, "fixture2", diagnostic=False)
    plan = rr.expand_matrix([("fixture1", f1), ("fixture2", f2)], ["A", "B"], n=2)
    keys = [(c["fixture"], c["arm"], c["trial_idx"]) for c in plan]
    assert keys == [
        ("fixture1", "A", 0), ("fixture1", "A", 1),
        ("fixture1", "B", 0), ("fixture1", "B", 1),
        ("fixture2", "A", 0), ("fixture2", "A", 1),
        ("fixture2", "B", 0), ("fixture2", "B", 1),
    ]
    f1a = next(c for c in plan if c["fixture"] == "fixture1" and c["arm"] == "A")
    assert f1a["vault_subdir"] == "vault_with_closed"
    assert f1a["task_file"] == "task_diagnostic.txt"
    f2b = next(c for c in plan if c["fixture"] == "fixture2" and c["arm"] == "B")
    assert f2b["vault_subdir"] == "vault_open"
    assert f2b["task_file"] == "task.txt"


def test_parse_arms_dedupes_and_splits_commas():
    assert rr.parse_arms(["A,B", "b"]) == ["A", "B"]


def test_parse_arms_defaults_to_a_and_b_only():
    # arm C is opt-in (only meaningful where a DISTINCT consult-memory task exists) — never a default.
    assert rr.DEFAULT_ARMS == ("A", "B")
    assert rr.parse_arms(None) == ["A", "B"]


def test_parse_arms_accepts_explicit_c():
    assert rr.parse_arms(["C"]) == ["C"]


def test_parse_arms_rejects_unknown_arm():
    with pytest.raises(ValueError):
        rr.parse_arms(["D"])


def test_discover_fixtures_matches_fixture_star_dirs(tmp_path):
    (tmp_path / "fixture1").mkdir()
    (tmp_path / "fixture2").mkdir()
    (tmp_path / "not_a_fixture").mkdir()
    (tmp_path / "fixture_stray_file.txt").write_text("x")
    assert rr.discover_fixtures(str(tmp_path)) == ["fixture1", "fixture2"]


def test_resolve_fixtures_all(tmp_path):
    (tmp_path / "fixture3").mkdir()
    (tmp_path / "fixture1").mkdir()
    out = rr.resolve_fixtures("all", str(tmp_path))
    assert [n for n, _ in out] == ["fixture1", "fixture3"]


def test_resolve_fixtures_comma_list(tmp_path):
    (tmp_path / "fixture1").mkdir()
    (tmp_path / "fixture2").mkdir()
    out = rr.resolve_fixtures("fixture2,fixture1", str(tmp_path))
    assert [n for n, _ in out] == ["fixture2", "fixture1"]


# ---- fixture prompt construction (context.md is load-bearing; fail loud when absent) ----

def test_read_fixture_prompt_orders_prefix_then_context_then_task(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", diagnostic=True)
    _write(os.path.join(fdir, "context.md"), "# scratch data\nretrieval slice is small")
    prompt = rr.read_fixture_prompt(fdir, "task_diagnostic.txt")
    assert prompt.startswith(rr.RECALL_PREFIX)  # identical for ALL arms, ahead of everything
    assert "retrieval slice is small" in prompt
    assert "neutral diagnose-and-recommend framing task" in prompt
    assert (prompt.index(rr.RECALL_PREFIX)
            < prompt.index("retrieval slice is small")
            < prompt.index("neutral diagnose"))


def test_recall_prefix_forces_the_skill_but_stays_content_neutral():
    # note-138 discipline: the prefix may force the /recall invocation and generic apply-what-
    # surfaces, but must never hint at lever-checking or prior attempts (that would spotlight the
    # moment the RED cell exists to leave unspotlighted).
    assert "/recall" in rr.RECALL_PREFIX
    assert "engram" in rr.RECALL_PREFIX  # forbids hand-running the binary in the skill's place
    low = rr.RECALL_PREFIX.lower()
    for hint in ("lever", "prior attempt", "already tried", "rolled back", "closed", "history"):
        assert hint not in low
    assert rr.RECALL_PREFIX.endswith("\n\n")  # clean seam ahead of context.md


def test_read_fixture_prompt_fails_loud_when_context_missing(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1")
    with pytest.raises(FileNotFoundError):
        rr.read_fixture_prompt(fdir, "task.txt")


def test_read_fixture_prompt_fails_loud_when_task_file_missing(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1")
    _write(os.path.join(fdir, "context.md"), "data")
    with pytest.raises(FileNotFoundError):
        rr.read_fixture_prompt(fdir, "task_diagnostic.txt")


# ---- arm-C gate: only meaningful where a DISTINCT consult-memory task exists ----

def test_arm_c_skip_reason_when_no_diagnostic_task(tmp_path):
    # no task_diagnostic.txt: arm A already runs task.txt, so arm C would be a silent duplicate.
    fdir = _make_fixture(tmp_path, "fixture2", diagnostic=False)
    reason = rr.arm_c_skip_reason(fdir)
    assert reason is not None
    assert "task_diagnostic" in reason


def test_arm_c_skip_reason_when_diagnostic_identical_to_task(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture2", diagnostic=False)
    _write(os.path.join(fdir, "task_diagnostic.txt"), "consult-memory framing task")
    reason = rr.arm_c_skip_reason(fdir)
    assert reason is not None
    assert "identical" in reason


def test_arm_c_skip_reason_none_when_distinct_consult_task(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", diagnostic=True)
    assert rr.arm_c_skip_reason(fdir) is None


def test_run_batch_arm_c_skips_with_explicit_record_when_unsupported(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture2", diagnostic=False)
    out = str(tmp_path / "results.jsonl")
    calls = []

    def maker(cell):
        def attempt(trial_idx):
            calls.append((cell["fixture"], cell["arm"], trial_idx))
            return {"status": "valid", "cost_usd": 0.1}
        return attempt

    rr.run_batch([("fixture2", fdir)], ["C"], n=1, retry_cap=2, out_path=out, attempt_maker=maker)
    assert calls == []  # never a silent duplicate run
    rows = rr.read_jsonl(out)
    assert len(rows) == 1
    assert rows[0]["arm"] == "C"
    assert rows[0]["status"] == "skipped"
    assert "task_diagnostic" in rows[0]["skip_reason"]


def test_run_batch_arm_c_runs_task_txt_when_distinct_consult_task(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", diagnostic=True)
    out = str(tmp_path / "results.jsonl")
    seen_cells = []

    def maker(cell):
        seen_cells.append(cell)
        def attempt(trial_idx):
            return {"status": "valid", "cost_usd": 0.1}
        return attempt

    rr.run_batch([("fixture1", fdir)], ["C"], n=1, retry_cap=2, out_path=out, attempt_maker=maker)
    assert len(seen_cells) == 1
    assert seen_cells[0]["task_file"] == "task.txt"
    assert seen_cells[0]["vault_subdir"] == "vault_with_closed"
    rows = rr.read_jsonl(out)
    assert len(rows) == 1
    assert rows[0]["status"] == "valid"


def test_run_batch_resume_does_not_duplicate_skip_record(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture2", diagnostic=False)
    out = str(tmp_path / "results.jsonl")

    def maker(cell):
        def attempt(trial_idx):
            return {"status": "valid", "cost_usd": 0.1}
        return attempt

    rr.run_batch([("fixture2", fdir)], ["C"], n=1, retry_cap=2, out_path=out, attempt_maker=maker)
    completed = rr.load_completed(out)
    rr.run_batch([("fixture2", fdir)], ["C"], n=1, retry_cap=2, out_path=out, attempt_maker=maker,
                 completed=completed)
    rows = rr.read_jsonl(out)
    assert len(rows) == 1  # the resume did not append a second skipped record


# ---- checkpoint / resume ----

def test_append_jsonl_appends_lines(tmp_path):
    out = str(tmp_path / "results.jsonl")
    rr.append_jsonl(out, {"a": 1})
    rr.append_jsonl(out, {"a": 2})
    rows = rr.read_jsonl(out)
    assert rows == [{"a": 1}, {"a": 2}]


def test_load_completed_skips_summary_and_cap_rows(tmp_path):
    out = str(tmp_path / "results.jsonl")
    rr.append_jsonl(out, {"fixture": "fixture1", "arm": "A", "trial_idx": 0, "status": "valid"})
    rr.append_jsonl(out, {"kind": "summary", "total_cost_usd": 1.0})
    rr.append_jsonl(out, {"kind": "cap_exhausted", "fixture": "fixture1", "arm": "A",
                          "attempts": 6, "valid": 0, "classification": "NOT-RED"})
    completed = rr.load_completed(out)
    assert completed == {("fixture1", "A", 0): "valid"}


def test_run_fixture_arm_appends_each_attempt_immediately_before_the_next_call(tmp_path):
    out = str(tmp_path / "results.jsonl")
    seen_len_at_call = []

    def attempt(trial_idx):
        seen_len_at_call.append(len(rr.read_jsonl(out)))
        return {"status": "valid", "cost_usd": 0.05}

    rr.run_fixture_arm("fixture1", "A", n=3, retry_cap=6, out_path=out, attempt_fn=attempt)
    # each call sees exactly the rows appended by prior calls (immediate flush before next attempt)
    assert seen_len_at_call == [0, 1, 2]
    rows = rr.read_jsonl(out)
    assert len(rows) == 3
    assert all(r["status"] == "valid" for r in rows)
    assert [r["trial_idx"] for r in rows] == [0, 1, 2]


def test_run_fixture_arm_resume_skips_already_done_keys(tmp_path):
    out = str(tmp_path / "results.jsonl")
    calls = []

    def attempt(trial_idx):
        calls.append(trial_idx)
        return {"status": "valid", "cost_usd": 0.05}

    already = {0: "valid", 1: "valid"}
    rr.run_fixture_arm("fixture1", "A", n=3, retry_cap=6, out_path=out, attempt_fn=attempt,
                       already_done=already)
    # only 1 more attempt needed to reach n=3 valid; new trial_idx continues after the resumed ones
    assert calls == [2]
    rows = rr.read_jsonl(out)
    assert len(rows) == 1
    assert rows[0]["trial_idx"] == 2


# ---- validity gate + retry cap ----

def test_trial_validity_invalid_when_log_missing(tmp_path):
    ok, reason = rr.trial_validity(str(tmp_path / "nope.jsonl"), 0.05, "some text")
    assert ok is False
    assert reason == "empty_or_missing_stub_log"


def test_trial_validity_invalid_when_log_empty(tmp_path):
    log = tmp_path / "log.jsonl"
    log.write_text("")
    ok, reason = rr.trial_validity(str(log), 0.05, "some text")
    assert ok is False
    assert reason == "empty_or_missing_stub_log"


def test_trial_validity_invalid_when_cost_below_floor(tmp_path):
    log = tmp_path / "log.jsonl"
    log.write_text('{"phrases": ["x"], "lever_keyed": false, "returned_buried": false}\n')
    ok, reason = rr.trial_validity(str(log), 0.01, "some text")
    assert ok is False
    assert reason == "cost_below_floor"


def test_trial_validity_invalid_when_agent_text_empty(tmp_path):
    log = tmp_path / "log.jsonl"
    log.write_text('{"phrases": ["x"], "lever_keyed": false, "returned_buried": false}\n')
    ok, reason = rr.trial_validity(str(log), 0.05, "   ")
    assert ok is False
    assert reason == "empty_agent_text"


def test_trial_validity_valid_when_all_conditions_met(tmp_path):
    log = tmp_path / "log.jsonl"
    log.write_text('{"phrases": ["x"], "lever_keyed": false, "returned_buried": false}\n')
    ok, reason = rr.trial_validity(str(log), 0.30, "RECOMMENDATION: do the thing.")
    assert ok is True
    assert reason is None


def test_run_fixture_arm_retries_invalid_up_to_cap_and_records_not_red(tmp_path):
    out = str(tmp_path / "results.jsonl")

    def always_invalid(trial_idx):
        return {"status": "invalid", "invalid_reason": "cost_below_floor", "cost_usd": 0.0}

    rr.run_fixture_arm("fixtureX", "A", n=3, retry_cap=6, out_path=out, attempt_fn=always_invalid)
    rows = rr.read_jsonl(out)
    trials = [r for r in rows if not r.get("kind")]
    assert len(trials) == 6  # stops at the retry cap, never reaches 3 valid
    assert all(r["status"] == "invalid" for r in trials)
    caps = [r for r in rows if r.get("kind") == "cap_exhausted"]
    assert len(caps) == 1  # <3 valid at the cap -> the pre-registered NOT-RED classification, recorded
    assert caps[0]["classification"] == "NOT-RED"
    assert caps[0]["valid"] == 0
    assert caps[0]["attempts"] == 6


def test_run_fixture_arm_resume_near_cap_leaves_only_remaining_attempts(tmp_path):
    out = str(tmp_path / "results.jsonl")
    calls = []

    def always_invalid(trial_idx):
        calls.append(trial_idx)
        return {"status": "invalid", "invalid_reason": "cost_below_floor", "cost_usd": 0.0}

    # 4 invalid + 0 valid already in the checkpoint -> only 2 of the 6 total attempts remain.
    already = {0: "invalid", 1: "invalid", 2: "invalid", 3: "invalid"}
    rr.run_fixture_arm("fixtureX", "A", n=3, retry_cap=6, out_path=out, attempt_fn=always_invalid,
                       already_done=already)
    assert calls == [4, 5]
    rows = rr.read_jsonl(out)
    trials = [r for r in rows if not r.get("kind")]
    assert len(trials) == 2
    caps = [r for r in rows if r.get("kind") == "cap_exhausted"]
    assert len(caps) == 1
    assert caps[0]["classification"] == "NOT-RED"
    assert caps[0]["attempts"] == 6 and caps[0]["valid"] == 0


def test_cap_classification_is_not_red_only_for_arm_a(tmp_path):
    out = str(tmp_path / "results.jsonl")

    def always_invalid(trial_idx):
        return {"status": "invalid", "invalid_reason": "cost_below_floor", "cost_usd": 0.0}

    rr.run_fixture_arm("fixtureX", "B", n=2, retry_cap=4, out_path=out, attempt_fn=always_invalid)
    caps = [r for r in rr.read_jsonl(out) if r.get("kind") == "cap_exhausted"]
    assert len(caps) == 1
    # NOT-RED is the plan's arm-A decision-procedure term; other arms record a neutral label.
    assert caps[0]["classification"] == "insufficient_valid_trials"


def test_run_fixture_arm_stops_once_n_valid_reached(tmp_path):
    out = str(tmp_path / "results.jsonl")

    def always_valid(trial_idx):
        return {"status": "valid", "cost_usd": 0.10}

    rr.run_fixture_arm("fixtureX", "A", n=3, retry_cap=6, out_path=out, attempt_fn=always_valid)
    rows = rr.read_jsonl(out)
    assert len(rows) == 3  # no cap_exhausted record when the target was reached


def test_run_fixture_arm_mixed_invalid_then_valid_within_cap(tmp_path):
    out = str(tmp_path / "results.jsonl")
    scripted = ["invalid", "invalid", "valid", "valid", "valid"]

    def scripted_attempt(trial_idx):
        status = scripted[trial_idx]
        return {"status": status, "cost_usd": 0.10 if status == "valid" else 0.0}

    rr.run_fixture_arm("fixtureX", "A", n=3, retry_cap=6, out_path=out, attempt_fn=scripted_attempt)
    rows = rr.read_jsonl(out)
    assert len(rows) == 5  # 2 invalid + 3 valid, under the cap of 6; no cap record
    assert [r["status"] for r in rows] == scripted


def test_retry_cap_through_cli_args_is_the_plans_six_at_default_n():
    # pins the pre-registered bar: <=6 total attempts per (fixture, arm) at the default --n 3.
    args = rr.build_argparser().parse_args([])
    assert args.n * rr.RETRY_CAP_MULTIPLIER == 6


def test_retry_cap_through_cli_args_scales_with_n():
    args = rr.build_argparser().parse_args(["--n", "5"])
    assert args.n * rr.RETRY_CAP_MULTIPLIER == 10


# ---- cost tally ----

def test_summarize_aggregates_per_fixture_arm_and_total():
    records = [
        {"fixture": "fixture1", "arm": "A", "status": "valid", "cost_usd": 0.30},
        {"fixture": "fixture1", "arm": "A", "status": "invalid", "cost_usd": 0.0},
        {"fixture": "fixture1", "arm": "B", "status": "valid", "cost_usd": 0.20},
    ]
    summary = rr.summarize(records)
    assert summary["kind"] == "summary"
    assert summary["per_fixture_arm"]["fixture1/A"] == {
        "attempts": 2, "valid": 1, "invalid": 1, "cost_usd": 0.30,
    }
    assert summary["per_fixture_arm"]["fixture1/B"] == {
        "attempts": 1, "valid": 1, "invalid": 0, "cost_usd": 0.20,
    }
    assert summary["total_cost_usd"] == 0.50


def test_summarize_ignores_existing_summary_and_cap_rows():
    records = [
        {"fixture": "fixture1", "arm": "A", "status": "valid", "cost_usd": 0.30},
        {"kind": "summary", "total_cost_usd": 999},
        {"kind": "cap_exhausted", "fixture": "fixture1", "arm": "A", "attempts": 6, "valid": 0,
         "classification": "NOT-RED"},
    ]
    summary = rr.summarize(records)
    assert summary["total_cost_usd"] == 0.30
    assert summary["per_fixture_arm"]["fixture1/A"]["attempts"] == 1


def test_summarize_counts_skipped_rows_separately():
    records = [
        {"fixture": "fixture2", "arm": "C", "trial_idx": -1, "status": "skipped",
         "skip_reason": "no distinct consult task"},
        {"fixture": "fixture2", "arm": "A", "status": "valid", "cost_usd": 0.20},
    ]
    summary = rr.summarize(records)
    assert summary["skipped"] == 1
    assert "fixture2/C" not in summary["per_fixture_arm"]
    assert summary["total_cost_usd"] == 0.20


# ---- fail-loud stub config ----

def test_stub_config_reads_buried_basename_and_lever_terms(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", lever_terms="cheaper,retrieval;cheap,model")
    basename, lever_terms = rr.stub_config(fdir)
    assert basename == "8.2026-06-20.cheap-retrieval-model-rolled-back"
    assert lever_terms == "cheaper,retrieval;cheap,model"


def test_stub_config_fails_loud_when_lever_terms_missing(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", lever_terms=None)
    with pytest.raises(KeyError):
        rr.stub_config(fdir)


def test_stub_config_fails_loud_when_lever_terms_empty_string(tmp_path):
    fdir = _make_fixture(tmp_path, "fixture1", lever_terms="")
    with pytest.raises(KeyError):
        rr.stub_config(fdir)


# ---- CLI plumbing ----

def test_build_argparser_defaults():
    args = rr.build_argparser().parse_args([])
    assert args.fixtures == "all"
    assert args.arm is None
    assert args.n == 3
    assert args.model == "opus"
    assert args.judge == "stub"
    assert args.resume is False


def test_build_argparser_accepts_fable_model():
    args = rr.build_argparser().parse_args(["--model", "fable"])
    assert args.model == "fable"


def test_build_argparser_accepts_repeated_and_comma_arms():
    args = rr.build_argparser().parse_args(["--arm", "A", "--arm", "B,C"])
    assert rr.parse_arms(args.arm) == ["A", "B", "C"]
