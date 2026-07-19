"""TDD for the recency-value aggregator (#646 Task 3, Gate B fixes): efficiency judged only over
correct, unassisted trials (note 292); raw_cost reported over ALL valid trials (the issue's
raw-cost requirement); correct_rate_fired + BOTH surfacing rates condition on recall_fired (note
83 - separates the firing gap from the delivery/ranking gap); dual surfacing (FIX 2) reports
surfaced_any_rate (P2 vacuous-contrast gate) and surfaced_via_recency_rate (recency-channel-only
diagnostic); rates are None-safe when no trial fired (EDGE); n_valid excludes phase1_ok==False.
"""
import os
import sys

sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
import recency_value_agg as agg


def _trial(phase1_ok=True, phase1_learned=False, recall_fired=True, correct=True,
           runlog_ok=True, revenue_ok=True, surfaced_any=True, surfaced_via_recency=True,
           phase2_cost=1.0, phase2_turns=5, phase2_dur_ms=1000):
    return {"phase1_ok": phase1_ok, "phase1_learned": phase1_learned, "recall_fired": recall_fired,
            "correct": correct, "runlog_ok": runlog_ok, "revenue_ok": revenue_ok,
            "surfaced_any": surfaced_any, "surfaced_via_recency": surfaced_via_recency,
            "phase2_cost": phase2_cost, "phase2_turns": phase2_turns, "phase2_dur_ms": phase2_dur_ms}


def test_n_valid_excludes_phase1_failures():
    trials = [_trial(phase1_ok=True), _trial(phase1_ok=False), _trial(phase1_ok=True)]
    assert agg.aggregate(trials)["n_valid"] == 2


def test_raw_cost_includes_all_valid_trials_regardless_of_correctness():
    trials = [_trial(correct=True, phase2_cost=2.0), _trial(correct=False, phase2_cost=4.0)]
    assert agg.aggregate(trials)["raw_cost"]["cost_usd"] == 3.0


def test_raw_cost_excludes_phase1_invalid_trials():
    trials = [_trial(phase1_ok=True, phase2_cost=2.0), _trial(phase1_ok=False, phase2_cost=999.0)]
    assert agg.aggregate(trials)["raw_cost"]["cost_usd"] == 2.0


def test_efficiency_ignores_incorrect_trials():
    trials = [_trial(correct=True, phase2_cost=2.0), _trial(correct=False, phase2_cost=100.0)]
    assert agg.aggregate(trials)["efficiency"]["cost_usd"] == 2.0


def test_efficiency_none_when_no_correct_trials():
    trials = [_trial(correct=False), _trial(correct=False)]
    assert agg.aggregate(trials)["efficiency"] is None


def test_correct_rate_fired_conditions_on_recall_fired():
    trials = [_trial(recall_fired=True, correct=True),
              _trial(recall_fired=True, correct=False),
              _trial(recall_fired=False, correct=True)]      # not fired -> excluded from fired view
    assert agg.aggregate(trials)["correct_rate_fired"] == 0.5


def test_surfaced_any_rate_over_fired_trials_only():
    trials = [_trial(recall_fired=True, surfaced_any=True),
              _trial(recall_fired=True, surfaced_any=False),
              _trial(recall_fired=False, surfaced_any=True)]  # not fired -> excluded
    assert agg.aggregate(trials)["surfaced_any_rate"] == 0.5


def test_surfaced_via_recency_rate_distinct_from_surfaced_any():
    # A re-rank leak: recall fired, the chunk surfaced (surfaced_any) but NOT via the recency
    # channel — the two rates must diverge (surfaced_any=1.0, via_recency=0.0).
    trials = [_trial(recall_fired=True, surfaced_any=True, surfaced_via_recency=False)]
    out = agg.aggregate(trials)
    assert out["surfaced_any_rate"] == 1.0
    assert out["surfaced_via_recency_rate"] == 0.0


def test_surfaced_rates_none_safe_when_no_trial_fired():
    trials = [_trial(recall_fired=False), _trial(recall_fired=False)]
    out = agg.aggregate(trials)
    assert out["surfaced_any_rate"] is None
    assert out["surfaced_via_recency_rate"] is None
    assert out["correct_rate_fired"] is None


def test_surfaced_rates_none_safe_when_keys_missing():
    # A trial that fired recall but whose surfacing keys are absent (empty payload) must not crash.
    trials = [{"phase1_ok": True, "recall_fired": True, "correct": False,
               "phase2_cost": 1.0, "phase2_turns": 3, "phase2_dur_ms": 500}]
    out = agg.aggregate(trials)
    assert out["surfaced_any_rate"] == 0.0
    assert out["surfaced_via_recency_rate"] == 0.0


def test_correct_rate_all_over_n_valid():
    trials = [_trial(correct=True), _trial(correct=False), _trial(phase1_ok=False, correct=True)]
    assert agg.aggregate(trials)["correct_rate_all"] == 0.5


def test_recall_fired_rate_over_n_valid():
    trials = [_trial(recall_fired=True), _trial(recall_fired=False),
              _trial(phase1_ok=False, recall_fired=True)]
    assert agg.aggregate(trials)["recall_fired_rate"] == 0.5


def test_runlog_and_revenue_rates_over_n_valid_and_distinct():
    # The RUNLOG convention is the lever; revenue is the easy floor. When revenue is always right
    # but only half honor the RUNLOG convention, the two rates must diverge — over n_valid, not
    # conditioned on firing.
    trials = [_trial(runlog_ok=True, revenue_ok=True, correct=True),
              _trial(runlog_ok=False, revenue_ok=True, correct=False),
              _trial(phase1_ok=False, runlog_ok=True, revenue_ok=True)]  # invalid -> excluded
    out = agg.aggregate(trials)
    assert out["revenue_ok_rate"] == 1.0
    assert out["runlog_ok_rate"] == 0.5


def test_verdict_inputs_present_with_learn_count():
    trials = [_trial(correct=True, phase1_learned=True), _trial(correct=False, phase1_learned=False)]
    out = agg.aggregate(trials)
    assert "verdict_inputs" in out
    assert out["verdict_inputs"]["n_correct"] == 1
    assert out["verdict_inputs"]["n_phase1_learned"] == 1
