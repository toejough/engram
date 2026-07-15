#!/usr/bin/env python3
"""lever_recheck scorer (C7 anti-amnesia): does a recommendation respect a CLOSED lever?

Given the agent's recommendation text and the ground-truth closed levers (closed_levers.json), decide
per closed lever whether the recommendation committed AMNESIA (re-proposed the closed lever as fresh)
or RECONCILED (acknowledged the prior attempt + outcome before advocating, or dropped it).

Mirrors synthesis_judge.py: stub (deterministic, zero-cost) vs real adversarial LLM judge (majority over
JUDGE_RUNS, default verdict = AMNESIA — the judge must be argued OUT of amnesia). Two real-mode voting
modes: default (`unforced=False`, C7) votes on each run's `verdict` field, keeping the `NOT proposed →
RECONCILED` semantics; unforced (`unforced=True`, underload_repro) votes on the `reconciled` field only,
ignoring `verdict`, and demands explicit prior-attempt acknowledgment via the appended
`_JUDGE_UNFORCED_CLARIFY` rubric. Reuses synthesis_judge's `_call_claude_judge` / `_parse_judge_json`
plumbing by import.

The judge maps MEANING to the lever's `canonical_action`/`closure_reason` ground truth, NOT the note's
literal words (heeds the scorer-vocabulary-bias lesson). A deterministic guard rejects "reconciliation
by vocabulary": a recommendation that name-drops the closure markers but still advocates the lever as
its top pick WITHOUT engaging the reason is AMNESIA, not reconciliation.
  RECONCILED: "We tried cheaper-tier retrieval (-14%, rolled back as not worth it); the real lever is
              the generation step."
  AMNESIA:    "Cheaper-tier retrieval is the highest-leverage cut."  (or names "-14%/rolled back" then
              still picks it #1 without saying why that's now OK.)

The diagnostic sub-metric `note_surfaced` is supplied by the harness (did the disproving note appear in
ANY recall the skill ran). It is recorded alongside the verdict so a retrieval-failure (note never
surfaced) is distinguishable from a synthesis-failure (surfaced but ignored) — never folded into the
pass/fail itself.
"""
import json
import os
import re
import subprocess
import textwrap

from synthesis_judge import _parse_judge_json  # reuse the generic JSON extractor (rubric-agnostic)

DEFAULT_JUDGE_MODEL = "claude-sonnet-4-6"
JUDGE_RUNS = 3

_JUDGE_SYSTEM = textwrap.dedent("""\
    You are an adversarial anti-amnesia judge. Your DEFAULT verdict is AMNESIA. You flip to RECONCILED
    only when the recommendation unambiguously engages the closed lever's prior outcome.

    You are given:
      RECOMMENDATION   — the agent's recommendation text.
      CLOSED_LEVER     — a lever that was already tried and closed, as ground truth:
                         canonical_action, closure_reason, measured_outcome.

    Decide TWO things about the recommendation, judging MEANING (paraphrase counts), not the wording of
    any note:
      proposed   = the recommendation advocates the canonical_action (as its pick or a top option).
      reconciled = it acknowledges the prior attempt AND its outcome (or drops the lever for that
                   reason) BEFORE/while advocating — not merely name-dropping the numbers.

    Verdict:
      AMNESIA    iff proposed AND NOT reconciled  (re-proposed a closed lever as if fresh).
      RECONCILED iff (NOT proposed) OR (proposed AND reconciled).

    Reject "reconciliation by vocabulary": if the text cites the closure markers (e.g. the % or
    "rolled back") yet still advocates the lever as its TOP pick without saying why revisiting is now
    justified, that is AMNESIA.

    Respond with a single JSON object, no surrounding text:
      { "verdict": "AMNESIA" | "RECONCILED", "proposed": true|false, "reconciled": true|false,
        "reason": "<one sentence>" }
""")

_JUDGE_USER_TMPL = textwrap.dedent("""\
    RECOMMENDATION:
    {recommendation}

    CLOSED_LEVER:
      canonical_action: {canonical_action}
      closure_reason:   {closure_reason}
      measured_outcome: {measured_outcome}
""")

# Appended to the judge system prompt ONLY in unforced (underload) mode. The forced-recall C7
# rubric is unchanged. This sharpens `reconciled` so thematically-adjacent caution is not
# mistaken for acknowledgment of the prior attempt (driftwood t2 was mis-scored RECONCILED).
_JUDGE_UNFORCED_CLARIFY = textwrap.dedent("""\

    IMPORTANT — UN-FORCED, UNDER-LOAD SETTING. The agent was NOT told to recall and may be blind
    to the prior attempt. Set reconciled=true ONLY if the recommendation EXPLICITLY surfaces that
    THIS specific approach was already attempted and states or paraphrases its measured outcome
    (e.g. "we already tried the cheaper tokenizer — relevance dropped and we rolled it back").
    Fresh reasoning that the approach MIGHT not work, generic caution, or a proposal to "measure
    it first / gate it on data / run an experiment" is NOT reconciliation: an agent proposing to
    RE-RUN an experiment that was already run and rolled back is committing AMNESIA, however
    well-reasoned the caution. If the reply does not name the prior attempt, reconciled=false.
""")


def _judge_system_prompt(unforced):
    """The judge system prompt: base rubric verbatim, plus the underload clarification ONLY in
    unforced mode. Kept as a pure seam so the C7-invariant (default == base rubric, byte-for-byte)
    is unit-testable without shelling out to the judge."""
    return _JUDGE_SYSTEM + (_JUDGE_UNFORCED_CLARIFY if unforced else "")


def _call_lever_judge(user_prompt, judge_model, unforced=False):
    """Run ONE adversarial lever-recheck judge call. Reuses synthesis_judge's generic JSON extractor but
    sends THIS module's rubric (`_JUDGE_SYSTEM`) — not synthesis_judge's, whose `_call_claude_judge`
    hardcodes the synthesis rubric and so must not be reused here.

    When `unforced`, the underload clarification is appended so `reconciled` demands explicit
    prior-attempt acknowledgment, not thematic caution."""
    system = _judge_system_prompt(unforced)
    full = system + "\n\n" + user_prompt
    cmd = ["claude", "--model", judge_model, "--print", full]
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if result.returncode != 0:
        raise RuntimeError(f"claude lever judge CLI failed (exit {result.returncode}): {result.stderr[:200]}")
    return _parse_judge_json(result.stdout.strip())


# Closure-marker cues for the deterministic guard (presence of a marker + still-advocating = amnesia).
_CLOSURE_CUES = ("rolled back", "rolled-back", "not worth", "already tried", "we tried",
                 "didn't help", "did not help", "-14%", "14%", "superseded", "reverted")

# Negation markers (#655 Unit 0): a sentence that advocates the lever but ALSO carries one of these
# is declining the lever, not re-proposing it fresh — the guard must defer to the judge, not amnesia.
_NEGATION_MARKERS = ("not ", "n't ", "no longer", "rather than", "instead of", "avoid",
                     "don't", "do not")


def load_closed_levers(fixture_dir):
    """Read closed_levers.json from a fixture dir. Fails LOUD — a missing file raises (no silent
    strawman: an eval input that is absent must not default to a passing condition)."""
    path = os.path.join(fixture_dir, "closed_levers.json")
    if not os.path.isfile(path):
        raise FileNotFoundError(f"closed_levers.json missing in fixture {fixture_dir!r}")
    with open(path) as fh:
        levers = json.load(fh)
    if not levers:
        raise ValueError(f"closed_levers.json in {fixture_dir!r} is empty — nothing to score")
    return levers


def _advocates(recommendation, lever):
    """Cheap lexical 'does it advocate the lever' check, for the stub, the deterministic guard, and
    arm B's advocacy control (score_arm_b).

    Needles derive from the lever's own `lever_terms` field — the SAME ';'-separated AND-group
    format (','-separated terms) and semantics as stub_engram's lever-keyed query check
    (`_phrase_is_lever_keyed`): the recommendation advocates the lever iff ANY group has ALL its
    terms present (case-insensitive substring). The AND-group is the specificity guard — no
    hit-count threshold, no hardcoded vocabulary, and no contiguous-id-phrase requirement (natural
    prose breaks the id with articles/possessives, and a fixed ML-cost word list only matched
    fixture1's domain — the measured false-negative on 3–4 of 5 fixtures).

    `lever_terms` is REQUIRED: fail LOUD when absent (a lever with no needles must never silently
    score as non-advocated)."""
    terms = lever.get("lever_terms")
    if not terms:
        raise KeyError(
            f"lever {lever.get('id')!r} is missing required 'lever_terms' — advocacy needles derive "
            f"from it (same AND-group format as STUB_ENGRAM_LEVER_TERMS); no hardcoded fallback")
    low = recommendation.lower()
    for group in terms.split(";"):
        group_terms = [t.strip().lower() for t in group.split(",") if t.strip()]
        if group_terms and all(t in low for t in group_terms):
            return True
    return False


def _has_negated_advocacy_sentence(recommendation, lever):
    """True iff some sentence of `recommendation` itself advocates the lever (same AND-group match
    as `_advocates`, reused on the single sentence) AND that same sentence also carries a negation
    marker — i.e. the recommendation is explicitly declining the closed lever ("not downgrading the
    retrieval model...") rather than re-proposing it. Sentences split on `.!?` boundaries."""
    for sentence in re.split(r"[.!?]", recommendation):
        sentence = sentence.strip()
        if not sentence or not _advocates(sentence, lever):
            continue
        low_sentence = sentence.lower()
        if any(marker in low_sentence for marker in _NEGATION_MARKERS):
            return True
    return False


def deterministic_guard(recommendation, lever):
    """Return 'AMNESIA' when the text advocates the lever AND only name-drops closure markers without
    engaging them; return None when the guard does not fire (defer to the LLM judge / stub)."""
    if not _advocates(recommendation, lever):
        return None
    low = recommendation.lower()
    cited = any(cue in low for cue in _CLOSURE_CUES)
    # advocacy + closure markers present but lever still advocated => vocabulary-only reconciliation
    # UNLESS the advocating sentence itself negates the lever (declining it, not re-proposing it).
    if cited:
        if _has_negated_advocacy_sentence(recommendation, lever):
            return None
        return "AMNESIA"
    return None


def _derive_real_verdict(runs, unforced):
    """Majority vote over the adversarial judge runs (default verdict = AMNESIA).

    Default (C7, forced-recall): vote on each run's own `verdict` field, preserving the
    NOT-proposed => RECONCILED semantics — there, silently avoiding a lever the agent was
    FORCED to see the closure for is success.

    Unforced (underload, un-forced): vote on the `reconciled` boolean ONLY, and intentionally
    IGNORE each run's `verdict` field (whose NOT-proposed => RECONCILED disjunct would credit a
    hedge). The agent may be blind to the prior attempt, so a reply that neither advocates NOR
    surfaces the closure (a hedge / "measure it first") is AMNESIA, not a free RECONCILED.
    Returns (verdict, votes)."""
    if unforced:
        votes = sum(1 for r in runs if r.get("reconciled"))
    else:
        votes = sum(1 for r in runs if r.get("verdict") == "RECONCILED")
    verdict = "RECONCILED" if votes > len(runs) // 2 else "AMNESIA"
    return verdict, votes


def score_recommendation(recommendation, lever, note_surfaced, stub=True,
                         judge_model=DEFAULT_JUDGE_MODEL, unforced=False):
    """Score one recommendation against one closed lever.

    Returns a dict: verdict, proposed, reconciled, note_surfaced, stub_mode, judge_runs (real only),
    amnesia_votes/total_runs (real only), guard_fired.

    The deterministic guard only short-circuits in STUB mode (cheap, zero-cost heuristic). In real
    (LLM-judge) mode it is never consulted: `_has_negated_advocacy_sentence` scopes its negation
    check to a single sentence, but free-form, un-forced multi-sentence replies (e.g. underload_repro's
    turn-4 shape) routinely cite the closure marker in one sentence and state the decline in a
    different one — the guard false-positives AMNESIA on exactly this faithful-decline shape. The
    adversarial judge reads meaning across the whole reply, not sentence boundaries, and was verified
    (underload_repro/audit_guard_crosscheck.py against results/red_baseline.jsonl fixture3) to
    correctly reclassify all 3 guard-fired trials AMNESIA -> RECONCILED. So real mode always defers
    to it; stub mode (and its pinned unit tests) is unchanged.
    """
    if stub:
        guard = deterministic_guard(recommendation, lever)
        if guard == "AMNESIA":
            return {"verdict": "AMNESIA", "proposed": True, "reconciled": False,
                    "note_surfaced": note_surfaced, "stub_mode": True, "guard_fired": True,
                    "judge_runs": None}
        proposed = _advocates(recommendation, lever)
        verdict = "AMNESIA" if proposed else "RECONCILED"
        return {"verdict": verdict, "proposed": proposed, "reconciled": not proposed,
                "note_surfaced": note_surfaced, "stub_mode": True, "guard_fired": False,
                "judge_runs": None}

    user = _JUDGE_USER_TMPL.format(
        recommendation=recommendation,
        canonical_action=lever.get("canonical_action", ""),
        closure_reason=lever.get("closure_reason", ""),
        measured_outcome=lever.get("measured_outcome", ""),
    )
    runs = [_call_lever_judge(user, judge_model, unforced=unforced) for _ in range(JUDGE_RUNS)]
    verdict, reconciled_votes = _derive_real_verdict(runs, unforced)
    return {"verdict": verdict, "proposed": None, "reconciled": verdict == "RECONCILED",
            "note_surfaced": note_surfaced, "stub_mode": False, "guard_fired": False,
            "judge_runs": runs, "reconciled_votes": reconciled_votes, "total_runs": JUDGE_RUNS,
            "unforced": unforced}


def score_fixture(recommendation, fixture_dir, note_surfaced, stub=True,
                  judge_model=DEFAULT_JUDGE_MODEL, unforced=False):
    """Score a recommendation against every closed lever in a fixture. Cell passes (RECONCILED) only if
    EVERY closed lever is reconciled. Returns the per-lever results + the aggregate cell verdict."""
    levers = load_closed_levers(fixture_dir)
    per_lever = [score_recommendation(recommendation, lev, note_surfaced, stub=stub,
                                      judge_model=judge_model, unforced=unforced) for lev in levers]
    cell = "RECONCILED" if all(r["verdict"] == "RECONCILED" for r in per_lever) else "AMNESIA"
    return {"cell_verdict": cell, "per_lever": per_lever, "note_surfaced": note_surfaced}


def score_arm_b(recommendation, fixture_dir):
    """Arm-B (vault_open) control score: does the recommendation advocate the closed lever, by the
    SAME lexical check score_recommendation uses internally (`_advocates`) — never the amnesia judge.

    Why no judge: arm B's vault carries NO note recording the closure, so a legitimate recommendation
    that arrives at the lever (which is, by fixture design, the natural answer to the task's data)
    can never engage a closure it was never told about — the default-AMNESIA judge would false-flag
    every such trial, violating the pre-registered control bar (false-AMNESIA = 0). Arm B exists only
    to prove the task tempts the agent toward the lever (defeats a degenerate scorer), so advocacy is
    the whole measurement. No cell_verdict / AMNESIA / RECONCILED is ever produced here.

    Returns {"per_lever_advocacy": [{"lever_id", "advocates"}, ...], "advocates": bool} — the
    aggregate `advocates` boolean is required on every arm-B trial record."""
    levers = load_closed_levers(fixture_dir)
    per_lever = [{"lever_id": lever.get("id"), "advocates": _advocates(recommendation, lever)}
                 for lever in levers]
    advocates = bool(per_lever) and all(entry["advocates"] for entry in per_lever)
    return {"per_lever_advocacy": per_lever, "advocates": advocates}
