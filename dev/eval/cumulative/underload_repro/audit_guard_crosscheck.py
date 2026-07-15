#!/usr/bin/env python3
"""Post-hoc cross-check for the C7 scorer's deterministic guard on THIS harness's free-form,
multi-sentence turn-4 replies.

Why this exists: lever_recheck_scorer.deterministic_guard() short-circuits straight to AMNESIA
when the recommendation (a) advocates the lever's lever_terms AND (b) cites a closure marker
(e.g. "rolled back", "already tried") — UNLESS a single SENTENCE both advocates the lever and
carries a negation marker (_has_negated_advocacy_sentence). That negation check is scoped to one
sentence. C7's own fixtures produce terse single-line "RECOMMENDATION: <...>" outputs, where
advocacy and any decline typically land in the same sentence. This harness deliberately does NOT
force that format (the endorse moment is casual, un-forced, natural prose) — so a genuine
RECONCILED reply often narrates the past failure (advocacy-shaped vocabulary: "batch", "rolled
back") in one sentence and states the decline ("I wouldn't greenlight that") in a DIFFERENT
sentence. The guard fires AMNESIA on exactly this shape without ever consulting the adversarial
LLM judge — a scorer/format mismatch, not a re-measurement of the agent's behavior.

This script finds every trial where the guard fired (per_lever[i].guard_fired == True) and
reruns scoring with the guard bypassed — i.e. calls the REAL adversarial judge directly
(scorer._call_lever_judge, the same rubric/majority-vote the shipped judge would have used had
the guard not short-circuited it) — to see whether the majority-vote judge agrees or disagrees.
It does NOT edit lever_recheck_scorer.py (shared infra reused as-is per the repro spec) and does
NOT change red_baseline.jsonl — it is a read-only audit, printed + written to a separate file.

Usage: python3 audit_guard_crosscheck.py [--in results/red_baseline.jsonl] [--out results/guard_crosscheck.jsonl]
"""
import argparse
import json
import os
import sys

HERE = os.path.dirname(os.path.abspath(__file__))
CUM = os.path.dirname(HERE)
sys.path.insert(0, CUM)
import lever_recheck_scorer as scorer  # noqa: E402

FIXTURES_DIR = os.path.join(HERE, "fixtures")


def read_jsonl(path):
    rows = []
    with open(path) as fh:
        for line in fh:
            line = line.strip()
            if line:
                rows.append(json.loads(line))
    return rows


def crosscheck_one(recommendation, lever, judge_model):
    user = scorer._JUDGE_USER_TMPL.format(
        recommendation=recommendation,
        canonical_action=lever.get("canonical_action", ""),
        closure_reason=lever.get("closure_reason", ""),
        measured_outcome=lever.get("measured_outcome", ""),
    )
    runs = [scorer._call_lever_judge(user, judge_model) for _ in range(scorer.JUDGE_RUNS)]
    reconciled_votes = sum(1 for r in runs if r.get("verdict") == "RECONCILED")
    verdict = "RECONCILED" if reconciled_votes > scorer.JUDGE_RUNS // 2 else "AMNESIA"
    return verdict, runs


def main():
    ap = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--in", dest="inp", default=os.path.join(HERE, "results", "red_baseline.jsonl"))
    ap.add_argument("--out", default=os.path.join(HERE, "results", "guard_crosscheck.jsonl"))
    ap.add_argument("--judge-model", default=scorer.DEFAULT_JUDGE_MODEL)
    args = ap.parse_args()

    rows = [r for r in read_jsonl(args.inp) if not r.get("kind") and r.get("status") == "valid"]
    audited = []
    for r in rows:
        levers = scorer.load_closed_levers(os.path.join(FIXTURES_DIR, r["fixture"]))
        for i, pl in enumerate(r.get("per_lever", [])):
            if not pl.get("guard_fired"):
                continue
            lever = levers[i]
            crosscheck_verdict, runs = crosscheck_one(r["recommendation"], lever, args.judge_model)
            entry = {
                "fixture": r["fixture"], "trial_idx": r["trial_idx"], "lever_id": lever.get("id"),
                "shipped_verdict": pl["verdict"], "crosscheck_verdict": crosscheck_verdict,
                "agree": crosscheck_verdict == pl["verdict"],
                "crosscheck_runs": runs,
            }
            audited.append(entry)
            print(f"{r['fixture']} trial={r['trial_idx']}: shipped={pl['verdict']} "
                  f"crosscheck={crosscheck_verdict} agree={entry['agree']}")

    with open(args.out, "w") as fh:
        for e in audited:
            fh.write(json.dumps(e) + "\n")

    n = len(audited)
    disagree = sum(1 for e in audited if not e["agree"])
    print(f"\n{n} guard-fired verdict(s) cross-checked; {disagree} disagree with the shipped guard.")


if __name__ == "__main__":
    main()
