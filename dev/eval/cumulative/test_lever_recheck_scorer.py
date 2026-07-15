"""Unit tests for lever_recheck_scorer (deterministic — stub mode + guard, no LLM calls)."""
import json
import os

import pytest

import lever_recheck_scorer as s

HERE = os.path.dirname(os.path.abspath(__file__))
FIXTURE1 = os.path.join(HERE, "lever_recheck", "fixture1")

LEVER = {
    "id": "cheap-retrieval-model",
    "canonical_action": "run the retrieval / context-fetch step on a cheaper or smaller model to cut cost",
    "closure_reason": "retrieval is a small share of per-document cost, so cheapening it barely moves the total",
    "measured_outcome": "only ~-14% total cost when tried; rolled back",
    "lever_terms": "cheap,retrieval,model;retrieval,small,model",  # fixture1's real AND-groups
}


def test_stub_amnesia_when_recommendation_advocates_the_closed_lever():
    rec = "RECOMMENDATION: run the retrieval step on a cheaper, smaller model to cut per-doc cost."
    r = s.score_recommendation(rec, LEVER, note_surfaced=False, stub=True)
    assert r["verdict"] == "AMNESIA"
    assert r["proposed"] is True


def test_stub_reconciled_when_recommendation_targets_a_different_lever():
    rec = "RECOMMENDATION: trim the generation step's prompt/context length, the dominant cost."
    r = s.score_recommendation(rec, LEVER, note_surfaced=True, stub=True)
    assert r["verdict"] == "RECONCILED"
    assert r["proposed"] is False


def test_deterministic_guard_catches_reconciliation_by_vocabulary():
    # Adversarial paraphrase: name-drops the closure markers but STILL advocates the lever as the pick.
    rec = ("RECOMMENDATION: run retrieval on a cheaper model. Yes a prior test showed only -14% and it "
           "was rolled back, but it is still the highest-leverage cut.")
    guard = s.deterministic_guard(rec, LEVER)
    assert guard == "AMNESIA"
    r = s.score_recommendation(rec, LEVER, note_surfaced=True, stub=True)
    assert r["verdict"] == "AMNESIA"
    assert r["guard_fired"] is True


def test_guard_does_not_fire_when_lever_not_advocated():
    rec = "RECOMMENDATION: batch the generation calls; we tried other things but this is untried."
    assert s.deterministic_guard(rec, LEVER) is None


def test_note_surfaced_is_recorded_not_folded_into_verdict():
    rec = "RECOMMENDATION: trim generation context length."
    surfaced = s.score_recommendation(rec, LEVER, note_surfaced=True, stub=True)
    buried = s.score_recommendation(rec, LEVER, note_surfaced=False, stub=True)
    # same verdict regardless of note_surfaced; the flag is recorded, not part of pass/fail
    assert surfaced["verdict"] == buried["verdict"] == "RECONCILED"
    assert surfaced["note_surfaced"] is True
    assert buried["note_surfaced"] is False


def test_load_closed_levers_fails_loud_on_missing_file(tmp_path):
    with pytest.raises(FileNotFoundError):
        s.load_closed_levers(str(tmp_path))


def test_load_closed_levers_reads_fixture1():
    levers = s.load_closed_levers(FIXTURE1)
    assert any(l["id"] == "cheap-retrieval-model" for l in levers)


def test_score_fixture_amnesia_if_any_lever_unreconciled():
    rec = "RECOMMENDATION: run retrieval on a cheaper smaller model."
    out = s.score_fixture(rec, FIXTURE1, note_surfaced=False, stub=True)
    assert out["cell_verdict"] == "AMNESIA"


def test_score_fixture_reconciled_when_lever_avoided():
    rec = "RECOMMENDATION: trim the generation prompt/context length."
    out = s.score_fixture(rec, FIXTURE1, note_surfaced=True, stub=True)
    assert out["cell_verdict"] == "RECONCILED"


# ---- advocacy needles derive from lever_terms AND-groups (never a hardcoded vocabulary) ----
#
# Natural-language probes per fixture domain (the Gate-B reviewer's style): prose breaks the lever id
# with articles/possessives and uses non-ML-cost vocabulary, so needles must come from the fixture's
# own lever_terms — semantics identical to stub_engram._phrase_is_lever_keyed (ANY ';'-group with ALL
# its ','-terms present, case-insensitive substring).

def _lever(lever_id, terms):
    return {"id": lever_id, "canonical_action": "", "closure_reason": "", "measured_outcome": "",
            "lever_terms": terms}


def test_advocates_fixture2_domain_natural_prose():
    lever = _lever("shrink-archive-page-size", "archive,page,size;shrink,page")
    rec = "I recommend shrinking the archive's page size to cut storage-read cost."
    assert s._advocates(rec, lever) is True


def test_advocates_fixture3_domain_natural_prose():
    lever = _lever("move-flaky-suite-to-background", "flaky,suite,background;suite,async,queue")
    rec = "Move the flaky integration suite off the blocking pipeline into a background queue."
    assert s._advocates(rec, lever) is True


def test_advocates_fixture4_domain_natural_prose():
    lever = _lever("index-by-ticket-question", "index,ticket,question;key,article,originating")
    rec = "Re-index the KB articles keyed by the original ticket question phrasing."
    assert s._advocates(rec, lever) is True


def test_advocates_fixture5_domain_natural_prose():
    lever = _lever("per-shard-subprocess-isolation", "subprocess,isolation,shard;spawn,process,shard")
    rec = "Spawn each shard's sync in its own subprocess so the OS reclaims memory on exit."
    assert s._advocates(rec, lever) is True


def test_advocates_false_when_recommendation_avoids_the_lever_domain():
    lever = _lever("shrink-archive-page-size", "archive,page,size;shrink,page")
    rec = "Batch adjacent reads and cache hot chunks to amortize the metered cost."
    assert s._advocates(rec, lever) is False


def test_advocates_requires_all_terms_of_a_group_not_just_some():
    lever = _lever("move-flaky-suite-to-background", "flaky,suite,background;suite,async,queue")
    rec = "Quarantine the three worst tests in the suite and stabilize their retries."  # suite alone
    assert s._advocates(rec, lever) is False


def test_advocates_fails_loud_when_lever_terms_missing():
    lever = {"id": "x", "canonical_action": "do x", "closure_reason": "", "measured_outcome": ""}
    with pytest.raises(KeyError):
        s._advocates("any recommendation", lever)


def test_score_recommendation_fails_loud_when_lever_terms_missing():
    lever = {"id": "x", "canonical_action": "do x", "closure_reason": "", "measured_outcome": ""}
    with pytest.raises(KeyError):
        s.score_recommendation("any recommendation", lever, note_surfaced=False, stub=True)


def test_guard_still_catches_vocabulary_reconciliation_on_derived_needles():
    # fixture3-domain adversarial paraphrase: cites the closure marker, still advocates the lever.
    lever = _lever("move-flaky-suite-to-background", "flaky,suite,background;suite,async,queue")
    rec = ("Move the flaky suite to a background queue. We tried something like this before and "
           "rolled back, but it is still the best cut.")
    assert s.deterministic_guard(rec, lever) == "AMNESIA"


# ---- negation-fix (#655 Unit 0): a recommendation that explicitly DECLINES the closed lever must
# defer to the judge, not auto-AMNESIA on vocabulary alone ----

def test_guard_defers_on_negated_advocacy_arm_c_t0_fixture1():
    # Verbatim RED fixture (plan Unit 0, results_round2_C.jsonl t0): the recommendation explicitly
    # declines the closed lever ("not downgrading the retrieval model, which was already tried and
    # rolled back...") — pre-fix this false-AMNESIA'd on vocabulary; the guard must defer (None) so
    # the LLM judge can rule (expected RECONCILED).
    t0_text = (
        "Move the retrieval/context-fetch step's standard-model calls onto the batch-discount API "
        "pricing path (keeping the standard model, so retrieval quality is unchanged) — not "
        "downgrading the retrieval model, which was already tried and rolled back for raising net "
        "cost ~9%."
    )
    levers = s.load_closed_levers(FIXTURE1)
    lever = next(lev for lev in levers if lev["id"] == "cheap-retrieval-model")
    assert s.deterministic_guard(t0_text, lever) is None


def test_guard_still_fires_amnesia_for_natural_unnegated_advocacy_fixture2_domain():
    # Regression: un-negated advocacy + closure cues in the SAME sentence still auto-AMNESIA — the
    # negation-fix must not neuter the existing reconciliation-by-vocabulary guard.
    lever = _lever("increase-archive-page-size",
                    "archive,page,increas;larger,page,archive;page,granularity")
    rec = ("Increase the archive page size; a one-tier trial already tried this and the discount "
           "held, so it is still the best cut.")
    assert s.deterministic_guard(rec, lever) == "AMNESIA"


def test_guard_defers_when_negation_present_but_no_closure_cues():
    # A negated-advocacy text with NO closure cues: the guard wasn't firing anyway (cited=False) —
    # still None, unaffected by the new negation check.
    lever = _lever("increase-archive-page-size",
                    "archive,page,increas;larger,page,archive;page,granularity")
    rec = "We should not increase the archive page size; that's untested territory."
    assert s.deterministic_guard(rec, lever) is None


# ---- arm-B advocacy-only control scoring (score_arm_b) ----

def test_score_arm_b_true_when_recommendation_advocates_the_lever():
    rec = "RECOMMENDATION: run the retrieval step on a cheaper, smaller model to cut per-doc cost."
    out = s.score_arm_b(rec, FIXTURE1)
    assert out["advocates"] is True
    assert out["per_lever_advocacy"][0]["advocates"] is True
    assert out["per_lever_advocacy"][0]["lever_id"] == "cheap-retrieval-model"


def test_score_arm_b_false_when_recommendation_avoids_the_lever():
    rec = "RECOMMENDATION: trim the generation step's prompt/context length."
    out = s.score_arm_b(rec, FIXTURE1)
    assert out["advocates"] is False


def test_score_arm_b_never_produces_a_cell_verdict():
    # arm B (vault_open — NO closure info anywhere) must never run the amnesia judge: default-AMNESIA
    # would false-flag every legitimate advocacy, violating the control bar (false-AMNESIA = 0).
    rec = "RECOMMENDATION: run the retrieval step on a cheaper, smaller model to cut per-doc cost."
    out = s.score_arm_b(rec, FIXTURE1)
    assert "cell_verdict" not in out
    assert "verdict" not in out


# ---- _derive_real_verdict (unforced verdict-derivation, pure/synthetic — no LLM calls) ----

def test_derive_verdict_default_credits_not_proposed():
    # C7 semantics preserved: a run that did not propose the lever counts as RECONCILED.
    runs = [{"verdict": "RECONCILED", "proposed": False, "reconciled": False}] * 3
    verdict, votes = s._derive_real_verdict(runs, unforced=False)
    assert verdict == "RECONCILED"
    assert votes == 3

def test_derive_verdict_unforced_hedge_is_amnesia():
    # The literal NOT-proposed loophole (beacon t2's run2 shape): no advocacy AND no
    # acknowledgment -> AMNESIA under unforced, not a free RECONCILED.
    runs = [{"verdict": "RECONCILED", "proposed": False, "reconciled": False}] * 3
    verdict, votes = s._derive_real_verdict(runs, unforced=True)
    assert verdict == "AMNESIA"
    assert votes == 0

def test_derive_verdict_unforced_mixed_beacon_shape():
    # beacon t2 actual stored runs: verdict-vote flips it to RECONCILED (loophole),
    # reconciled-vote (1/3) keeps it AMNESIA.
    runs = [
        {"verdict": "AMNESIA", "proposed": True, "reconciled": False},
        {"verdict": "RECONCILED", "proposed": True, "reconciled": True},
        {"verdict": "RECONCILED", "proposed": False, "reconciled": False},
    ]
    assert s._derive_real_verdict(runs, unforced=True)[0] == "AMNESIA"
    assert s._derive_real_verdict(runs, unforced=False)[0] == "RECONCILED"

def test_derive_verdict_unforced_genuine_reconcile_stays():
    # Guard against over-correction: an explicit reconciliation stays RECONCILED.
    runs = [{"verdict": "RECONCILED", "proposed": True, "reconciled": True}] * 3
    assert s._derive_real_verdict(runs, unforced=True)[0] == "RECONCILED"

def test_derive_verdict_blind_advocacy_amnesia_both_modes():
    runs = [{"verdict": "AMNESIA", "proposed": True, "reconciled": False}] * 3
    assert s._derive_real_verdict(runs, unforced=True)[0] == "AMNESIA"
    assert s._derive_real_verdict(runs, unforced=False)[0] == "AMNESIA"


def test_judge_system_prompt_default_is_base_rubric_verbatim():
    # C7-invariance: default mode sends the base rubric byte-for-byte, nothing appended.
    assert s._judge_system_prompt(False) == s._JUDGE_SYSTEM

def test_judge_system_prompt_unforced_appends_clarification():
    prompt = s._judge_system_prompt(True)
    assert prompt.startswith(s._JUDGE_SYSTEM)
    assert s._JUDGE_UNFORCED_CLARIFY in prompt
    assert prompt != s._JUDGE_SYSTEM
