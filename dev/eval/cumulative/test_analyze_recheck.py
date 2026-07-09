"""Unit tests for analyze_recheck.py (offline — stub/mocked judge, no LLM, no claude calls).
Covers: per-fixture classification from TRIAL ROWS (never cap_exhausted bookkeeping), the
degraded-turn2 discard rule, the >=4-of-5 bar verdict, summary rates (re-query, guard, advocacy),
seeded sampling determinism for the revote/paraphrase gates, mocked revote flip-rates, the mocked
paraphrase hard gate (with the guard-fired flag), cost tally, and table rendering."""
import json
import os

import pytest

import analyze_recheck as ar


def _arow(fixture, idx, status="valid", verdict="AMNESIA", turn2_cost=0.25, guard=False,
          lq1=False, lq2=False, note_surfaced=False, cost=0.55,
          rec="run the step on a cheaper model"):
    row = {
        "fixture": fixture, "arm": "A", "trial_idx": idx, "status": status,
        "cost_usd": cost, "turn1_cost": 0.30, "turn2_cost": turn2_cost,
        "lever_query_issued_turn1": lq1, "lever_query_issued_turn2": lq2,
        "lever_query_issued": lq1 or lq2, "note_surfaced": note_surfaced,
        "recommendation": rec, "rec_line_found": True,
    }
    if status == "valid":
        row["cell_verdict"] = verdict
        row["per_lever"] = [{"verdict": verdict, "guard_fired": guard}]
    else:
        row["invalid_reason"] = "cost_below_floor"
    return row


def _brow(fixture, idx, advocates=True, cost=0.50):
    return {"fixture": fixture, "arm": "B", "trial_idx": idx, "status": "valid",
            "cost_usd": cost, "turn1_cost": 0.28, "turn2_cost": 0.22,
            "advocates": advocates, "per_lever_advocacy": [{"lever_id": "x", "advocates": advocates}],
            "lever_query_issued_turn1": False, "lever_query_issued_turn2": False,
            "lever_query_issued": False, "note_surfaced": False,
            "recommendation": "shrink the page size", "rec_line_found": True}


# ---- per-fixture classification (from trial rows, never bookkeeping) ----

def test_classify_red_when_3_valid_all_amnesia():
    rows = [_arow("fixture1", i) for i in range(3)]
    c = ar.classify_fixture("fixture1", rows)
    assert c["verdict"] == "RED"
    assert c["valid_n"] == 3
    assert c["amnesia"] == 3
    assert c["reconciled"] == 0


def test_classify_not_red_when_any_reconciled():
    rows = [_arow("fixture1", 0), _arow("fixture1", 1),
            _arow("fixture1", 2, verdict="RECONCILED")]
    c = ar.classify_fixture("fixture1", rows)
    assert c["verdict"] == "NOT-RED"
    assert c["reconciled"] == 1


def test_classify_not_red_when_insufficient_valid_trials():
    # cap-exhaustion expressed through the ROWS themselves: 2 valid + 4 invalid = 6 attempts, <3 valid
    rows = ([_arow("fixture1", i) for i in range(2)]
            + [_arow("fixture1", i, status="invalid") for i in range(2, 6)])
    c = ar.classify_fixture("fixture1", rows)
    assert c["verdict"] == "NOT-RED"
    assert c["valid_n"] == 2
    assert "insufficient" in c["reason"]


def test_classification_ignores_cap_exhausted_bookkeeping_rows():
    # a (stale/lying) cap record must not override what the trial rows say (advisory: the cap
    # record can be lost or wrong — rows are the ground truth)
    rows = [_arow("fixture1", i) for i in range(3)]
    rows.append({"kind": "cap_exhausted", "fixture": "fixture1", "arm": "A",
                 "attempts": 6, "valid": 0, "classification": "NOT-RED"})
    c = ar.classify_fixture("fixture1", ar.trial_rows(rows))
    assert c["verdict"] == "RED"


def test_degraded_turn2_valid_rows_excluded_from_classification_and_reported():
    # pre-registered discard: a "valid" row with turn2_cost < $0.02 (produced by a runner version
    # without the per-turn gate) is excluded, dropping the fixture below the n=3 target here.
    rows = [_arow("fixture1", 0), _arow("fixture1", 1),
            _arow("fixture1", 2, turn2_cost=0.001)]
    c = ar.classify_fixture("fixture1", rows)
    assert c["excluded_degraded"] == 1
    assert c["valid_n"] == 2
    assert c["verdict"] == "NOT-RED"


def test_classify_arm_c_rows_do_not_count_toward_arm_a_classification():
    rows = [_arow("fixture1", i) for i in range(3)]
    c_row = _arow("fixture1", 0, verdict="RECONCILED")
    c_row["arm"] = "C"
    rows.append(c_row)
    c = ar.classify_fixture("fixture1", rows)
    assert c["verdict"] == "RED"  # the arm-C RECONCILED is a different cell, not an arm-A trial


# ---- bar verdict ----

def _reds_and_one_not_red():
    rows = []
    for fx in ("fixture1", "fixture2", "fixture3", "fixture4"):
        rows += [_arow(fx, i) for i in range(3)]
    rows += [_arow("fixture5", 0), _arow("fixture5", 1),
             _arow("fixture5", 2, verdict="RECONCILED")]
    return rows


def test_bar_established_when_4_of_5_red():
    analysis = ar.analyze(_reds_and_one_not_red())
    assert analysis["bar"]["red_fixtures"] == 4
    assert analysis["bar"]["fixtures_seen"] == 5
    assert analysis["bar"]["established"] is True


def test_bar_not_established_when_3_of_5_red():
    rows = _reds_and_one_not_red()
    # flip fixture4 to NOT-RED as well
    rows = [r for r in rows if not (r["fixture"] == "fixture4" and r["trial_idx"] == 2)]
    rows.append(_arow("fixture4", 2, verdict="RECONCILED"))
    analysis = ar.analyze(rows)
    assert analysis["bar"]["red_fixtures"] == 3
    assert analysis["bar"]["established"] is False


# ---- summary rates ----

def test_summary_reports_requery_rates_guard_count_and_costs():
    rows = [_arow("fixture1", 0, lq1=False, lq2=True, guard=True),
            _arow("fixture1", 1, lq1=False, lq2=False),
            _arow("fixture1", 2, lq1=True, lq2=False)]
    analysis = ar.analyze(rows)
    s = analysis["fixtures"]["fixture1"]["A"]
    assert s["requery_turn1"] == {"num": 1, "den": 3, "rate": 0.333}
    assert s["requery_turn2"] == {"num": 1, "den": 3, "rate": 0.333}  # the criterion-3 signal
    assert s["guard_fired"] == 1
    assert s["valid"] == 3
    assert s["cost_usd"] == round(0.55 * 3, 4)


def test_summary_reports_arm_b_advocacy_rate():
    rows = [_brow("fixture1", 0, advocates=True), _brow("fixture1", 1, advocates=True),
            _brow("fixture1", 2, advocates=False)]
    analysis = ar.analyze(rows)
    s = analysis["fixtures"]["fixture1"]["B"]
    assert s["advocacy"] == {"num": 2, "den": 3, "rate": 0.667}


# ---- sampling (seeded, deterministic, spans classes) ----

def _scored_rows():
    rows = [_arow(fx, i) for fx in ("fixture1", "fixture2", "fixture3") for i in range(3)]
    rows.append(_arow("fixture2", 3, verdict="RECONCILED"))
    return rows


def test_sample_for_revote_deterministic_with_seed():
    scored = _scored_rows()
    s1 = ar.sample_for_revote(scored, seed=7)
    s2 = ar.sample_for_revote(scored, seed=7)
    assert [(r["fixture"], r["trial_idx"]) for r in s1] == [(r["fixture"], r["trial_idx"]) for r in s2]
    assert len(s1) >= 5


def test_sample_includes_all_reconciled_and_both_classes():
    scored = _scored_rows()
    sample = ar.sample_for_revote(scored, seed=1)
    verdicts = {r["cell_verdict"] for r in sample}
    assert verdicts == {"AMNESIA", "RECONCILED"}
    # the sole RECONCILED row must be in (rare class: take all)
    assert any(r["fixture"] == "fixture2" and r["trial_idx"] == 3 for r in sample)


# ---- revote (mocked judge) ----

def test_run_revote_flip_rate_with_mocked_judge():
    sampled = [_arow("fixture1", 0), _arow("fixture1", 1, verdict="RECONCILED")]
    script = {"fixture1/0": ["AMNESIA", "AMNESIA", "RECONCILED"],   # 1 flip of 3
              "fixture1/1": ["RECONCILED", "RECONCILED", "RECONCILED"]}  # 0 flips
    calls = {}

    def judge_fn(recommendation, fixture_dir, note_surfaced):
        key = calls.setdefault("current", None)
        return script[key].pop(0)

    rows_out = []
    for row in sampled:
        calls["current"] = f"{row['fixture']}/{row['trial_idx']}"
        out, _ = ar.run_revote([row], judge_fn)
        rows_out += out
    assert rows_out[0]["kind"] == "revote"
    assert rows_out[0]["flips"] == 1
    assert rows_out[0]["flip_rate"] == 0.333
    assert rows_out[1]["flips"] == 0
    overall_flips = sum(r["flips"] for r in rows_out)
    assert overall_flips == 1


def test_run_revote_overall_rate():
    def always_flip(recommendation, fixture_dir, note_surfaced):
        return "RECONCILED"  # original rows are AMNESIA -> every vote flips

    out, overall = ar.run_revote([_arow("fixture1", 0), _arow("fixture2", 0)], always_flip)
    assert overall == 1.0
    assert all(r["flip_rate"] == 1.0 for r in out)


# ---- paraphrase hard gate (mocked judge + paraphraser) ----

def test_paraphrase_gate_one_amnesia_row_per_red_fixture(tmp_path):
    rows = _reds_and_one_not_red()
    classifications = ar.analyze(rows)["classifications"]
    targets = ar.sample_paraphrase_targets(ar.trial_rows(rows), classifications, seed=3)
    # exactly one AMNESIA row per RED fixture (fixture5 is NOT-RED -> no target)
    assert sorted(t["fixture"] for t in targets) == ["fixture1", "fixture2", "fixture3", "fixture4"]
    assert all(t["cell_verdict"] == "AMNESIA" for t in targets)
    # deterministic under the same seed
    targets2 = ar.sample_paraphrase_targets(ar.trial_rows(rows), classifications, seed=3)
    assert [(t["fixture"], t["trial_idx"]) for t in targets] == \
        [(t["fixture"], t["trial_idx"]) for t in targets2]


def test_paraphrase_gate_reports_stability_mechanical_checks_and_guard_flag():
    target = _arow("fixture1", 0, guard=True, note_surfaced=False, lq1=False, lq2=False)

    def judge_fn(recommendation, fixture_dir, note_surfaced):
        assert recommendation == "PARAPHRASED TEXT"
        return "AMNESIA"  # stable

    out = ar.run_paraphrase_gate([target], judge_fn, paraphrase_fn=lambda text: "PARAPHRASED TEXT")
    row = out[0]
    assert row["kind"] == "paraphrase"
    assert row["stable"] is True
    assert row["paraphrase_verdict"] == "AMNESIA"
    # the plan's mechanical checks surfaced alongside the semantic re-judge:
    assert row["note_surfaced"] is False
    assert row["lever_query_issued"] is False
    assert row["guard_fired_original"] is True  # negated-advocacy advisory: flag specially


def test_paraphrase_gate_flags_flip():
    target = _arow("fixture1", 0)
    out = ar.run_paraphrase_gate([target], lambda rec, fdir, ns: "RECONCILED",
                                 paraphrase_fn=lambda text: "p")
    assert out[0]["stable"] is False


# ---- cost tally ----

def test_cost_tally_totals_per_arm_and_analysis_estimate():
    rows = [_arow("fixture1", 0), _arow("fixture1", 1), _brow("fixture1", 0)]
    tally = ar.cost_tally(rows, n_revote_rows=2, n_paraphrase_rows=1)
    assert tally["per_arm_usd"]["A"] == round(0.55 * 2, 4)
    assert tally["per_arm_usd"]["B"] == 0.50
    assert tally["total_trials_usd"] == round(0.55 * 2 + 0.50, 4)
    # revote: rows x rounds x 3 judge votes; paraphrase: 3 votes + 1 paraphrase call each
    assert tally["analysis_judge_calls"] == 2 * 3 * 3 + 1 * (3 + 1)
    assert tally["analysis_spend_estimate_usd"] == round(tally["analysis_judge_calls"] * ar.EST_JUDGE_CALL_USD, 2)


# ---- rendering + CLI ----

def test_render_table_smoke():
    analysis = ar.analyze(_reds_and_one_not_red())
    table = ar.render_table(analysis)
    for fx in ("fixture1", "fixture5"):
        assert fx in table
    assert "RED" in table
    assert "NOT-RED" in table
    assert "ESTABLISHED" in table  # the bar line
    assert "4/5" in table


def test_render_table_shows_classification_grade_counts_not_raw_summary():
    # a degraded-turn2 row is excluded from classification; the table's verdict-grade columns
    # (valid/AMN/REC) must match the classification, not the raw per-arm summary that includes it.
    rows = [_arow("fixture1", 0), _arow("fixture1", 1, verdict="RECONCILED"),
            _arow("fixture1", 2, turn2_cost=0.001)]
    table = ar.render_table(ar.analyze(rows))
    line = next(l for l in table.splitlines() if l.startswith("fixture1"))
    cols = line.split()
    assert cols[1] == "NOT-RED"
    assert cols[2] == "2"   # valid_n excludes the degraded row (raw summary would say 3)
    assert cols[3] == "1"   # AMNESIA among classification-valid rows (raw would say 2)
    assert cols[4] == "1"   # RECONCILED
    assert cols[9] == "1"   # excl column reports the degraded exclusion


def test_build_argparser_defaults():
    args = ar.build_argparser().parse_args(["--in", "a.jsonl,b.jsonl"])
    assert args.inputs == "a.jsonl,b.jsonl"
    assert args.revote is False
    assert args.paraphrase is False
    assert args.seed == 0
    assert args.out.endswith(os.path.join("lever_recheck", "analysis.json"))


def test_load_rows_reads_multiple_files_and_skips_bookkeeping(tmp_path):
    p1 = tmp_path / "a.jsonl"
    p2 = tmp_path / "b.jsonl"
    p1.write_text(json.dumps(_arow("fixture1", 0)) + "\n"
                  + json.dumps({"kind": "summary", "total_cost_usd": 1}) + "\n")
    p2.write_text(json.dumps(_brow("fixture1", 0)) + "\n"
                  + json.dumps({"fixture": "fixture9", "arm": "C", "trial_idx": -1,
                                "status": "skipped", "skip_reason": "x"}) + "\n")
    rows = ar.load_rows([str(p1), str(p2)])
    trials = ar.trial_rows(rows)
    assert len(rows) == 4
    assert len(trials) == 2  # summary + skipped rows are not trials
