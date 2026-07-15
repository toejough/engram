#!/usr/bin/env python3
"""Re-score the stored underload_repro v2 recommendations under the tightened UNFORCED scorer.

The expensive multi-turn opus harness is NOT re-run: v2 records carry the full `recommendation`
text, so only the (cheap) adversarial judge is re-run, in unforced mode. Emits red_baseline_v3
and an aggregate table split by whether recall fired (the honest failure denominator is the
no-recall trials)."""
import json
import os
import sys
import collections

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
import lever_recheck_scorer as scorer  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
FIXTURES = os.path.join(HERE, "fixtures")
V2 = os.path.join(HERE, "results", "red_baseline_v2.jsonl")
V3 = os.path.join(HERE, "results", "red_baseline_v3.jsonl")


# Human-verified ground truth (transcript read, 2026-07-15): 5 no-recall trials all blind,
# 4 recall-fired trials all reconciled. A VALIDITY signal for the tightened scorer — NOT a
# target to engineer toward; the printed numbers are reported verbatim regardless.
EXPECT_NORECALL_AMNESIA = 5
EXPECT_RECALL_RECONCILED = 4


def main():
    with open(V2) as fh:
        rows = [json.loads(line) for line in fh if line.strip()]
    rows = [r for r in rows if r.get("fixture") and r.get("marker_seen")]
    tally = collections.Counter()
    with open(V3, "w") as out:
        for r in sorted(rows, key=lambda x: (x["fixture"], x["trial_idx"])):
            fixture_dir = os.path.join(FIXTURES, r["fixture"])
            scored = scorer.score_fixture(r["recommendation"], fixture_dir,
                                          note_surfaced=r.get("recall_fired_any"),
                                          stub=False, unforced=True)
            out.write(json.dumps({**r, "v3_cell_verdict": scored["cell_verdict"],
                                  "v3_per_lever": scored["per_lever"]}) + "\n")
            channel = "recall" if r.get("recall_fired_any") else "no-recall"
            tally[(channel, scored["cell_verdict"])] += 1
            print(f'{r["fixture"]:24} t{r["trial_idx"]}  recall_any={str(r.get("recall_fired_any")):5}  '
                  f'v2={r.get("cell_verdict"):11} v3={scored["cell_verdict"]}')

    norecall_total = sum(v for (ch, _), v in tally.items() if ch == "no-recall")
    recall_total = sum(v for (ch, _), v in tally.items() if ch == "recall")
    norecall_amnesia = tally[("no-recall", "AMNESIA")]
    recall_recon = tally[("recall", "RECONCILED")]
    print("\nHonest baseline (v3, unforced):")
    print(f"  Blind-endorse (AMNESIA) | recall did NOT fire: {norecall_amnesia}/{norecall_total}")
    print(f"  Reconciled              | recall fired:        {recall_recon}/{recall_total}")

    passed = (norecall_amnesia == norecall_total == EXPECT_NORECALL_AMNESIA
              and recall_recon == recall_total == EXPECT_RECALL_RECONCILED)
    print(f"\nPRE-REGISTERED BAR ({EXPECT_NORECALL_AMNESIA}/{EXPECT_NORECALL_AMNESIA} no-recall "
          f"AMNESIA, {EXPECT_RECALL_RECONCILED}/{EXPECT_RECALL_RECONCILED} recall RECONCILED): "
          f"{'PASS' if passed else 'DEVIATION — investigate per plan Task 2 Step 3'}")


if __name__ == "__main__":
    main()
