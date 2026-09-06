#!/usr/bin/env python3
"""
Merge the rejudged followed verdicts (results/rejudge-followed-nulls.jsonl)
into a NEW copy of results/moments.jsonl, written to results/moments.jsonl.new.

Never modifies moments.jsonl in place -- the caller inspects the printed
old->new diff and swaps the file in deliberately.

Rules (per the verification hand-off):
- For each NON-skipped rejudge record:
  - new_followed == "not_applicable": set
    finding_side.followed_null_reason = "judged_not_applicable" and leave
    finding_side.followed null -- the scorecard spec's followed enum has no
    not_applicable value, so N/A stays null but explained.
  - any other verdict: set finding_side.followed = new_followed and
    finding_side.followed_provenance = "rejudge-followed-nulls".
- ALSO (mechanical, no LLM): every dispatch-type moment with surfaced==true
  and followed==null gets
  finding_side.followed_null_reason = "not_applicable_dispatch_moment"
  (followed is structurally not-applicable for dispatch moments by design).

Fail-loud invariants (never merge into an unexpected state):
- every non-skipped rejudge moment_id must exist in moments.jsonl;
- every rejudge target must still have surfaced==true, followed==null,
  and moment_type != "dispatch" in moments.jsonl (a non-null followed
  means this merge already ran or the file changed under us);
- no target may already carry followed_null_reason or followed_provenance.

Usage: python3 merge_followed_nulls.py
"""

import json
import sys
from pathlib import Path

_SCRIPT_DIR = Path(__file__).resolve().parent
MOMENTS_PATH = _SCRIPT_DIR / "results" / "moments.jsonl"
REJUDGE_PATH = _SCRIPT_DIR / "results" / "rejudge-followed-nulls.jsonl"
OUTPUT_PATH = _SCRIPT_DIR / "results" / "moments.jsonl.new"

NA_REASON = "judged_not_applicable"
DISPATCH_REASON = "not_applicable_dispatch_moment"
PROVENANCE = "rejudge-followed-nulls"


def _load_jsonl(path: Path) -> list:
    records = []
    with open(path, "r") as f:
        for line_number, line in enumerate(f, start=1):
            stripped = line.strip()
            if not stripped:
                continue
            try:
                records.append(json.loads(stripped))
            except json.JSONDecodeError as exc:
                raise ValueError(f"{path}:{line_number}: invalid JSON: {exc}") from exc
    return records


def main() -> None:
    moments = _load_jsonl(MOMENTS_PATH)
    rejudged = _load_jsonl(REJUDGE_PATH)

    moments_by_id = {}
    for rec in moments:
        moment_id = rec["moment_id"]
        if moment_id in moments_by_id:
            raise ValueError(f"duplicate moment_id in {MOMENTS_PATH}: {moment_id}")
        moments_by_id[moment_id] = rec

    # --- Validate the rejudge records against the current moments state. ---
    verdicts = {}  # moment_id -> new_followed
    skipped = []
    for rec in rejudged:
        moment_id = rec["moment_id"]
        if rec.get("skipped_reason") is not None:
            skipped.append(moment_id)
            continue
        if moment_id in verdicts:
            raise ValueError(f"duplicate rejudge verdict for {moment_id}")
        if moment_id not in moments_by_id:
            raise KeyError(f"rejudge target {moment_id} not found in {MOMENTS_PATH}")
        target = moments_by_id[moment_id]
        finding = target.get("finding_side") or {}
        if target.get("moment_type") == "dispatch":
            raise ValueError(f"{moment_id}: rejudge target is dispatch-type -- not a valid target")
        if finding.get("surfaced") is not True:
            raise ValueError(f"{moment_id}: surfaced is not true -- not a valid target")
        if finding.get("followed") is not None:
            raise ValueError(
                f"{moment_id}: followed is already {finding['followed']!r} -- "
                "this merge appears to have already run (refusing to double-merge)"
            )
        for key in ("followed_null_reason", "followed_provenance"):
            if key in finding:
                raise ValueError(
                    f"{moment_id}: finding_side already has {key}={finding[key]!r} -- "
                    "refusing to double-merge"
                )
        new_followed = rec.get("new_followed")
        if not isinstance(new_followed, str) or not new_followed:
            raise ValueError(f"{moment_id}: non-skipped record has no new_followed verdict")
        verdicts[moment_id] = new_followed

    # --- Identify the dispatch-type surfaced-followed-null moments. ---
    dispatch_ids = []
    for rec in moments:
        finding = rec.get("finding_side") or {}
        if (
            rec.get("moment_type") == "dispatch"
            and finding.get("surfaced") is True
            and finding.get("followed") is None
        ):
            for key in ("followed_null_reason", "followed_provenance"):
                if key in finding:
                    raise ValueError(
                        f"{rec['moment_id']}: dispatch moment already has {key}="
                        f"{finding[key]!r} -- refusing to double-merge"
                    )
            dispatch_ids.append(rec["moment_id"])

    # --- Apply, preserving moments.jsonl's record order. ---
    changed = 0
    for rec in moments:
        moment_id = rec["moment_id"]
        if moment_id in verdicts:
            finding = rec["finding_side"]
            new_followed = verdicts[moment_id]
            if new_followed == "not_applicable":
                finding["followed_null_reason"] = NA_REASON
                print(
                    f"{moment_id}: followed null -> null "
                    f"(followed_null_reason={NA_REASON!r})"
                )
            else:
                finding["followed"] = new_followed
                finding["followed_provenance"] = PROVENANCE
                print(
                    f"{moment_id}: followed null -> {new_followed!r} "
                    f"(followed_provenance={PROVENANCE!r})"
                )
            changed += 1
        elif moment_id in dispatch_ids:
            rec["finding_side"]["followed_null_reason"] = DISPATCH_REASON
            print(
                f"{moment_id}: followed null -> null "
                f"(followed_null_reason={DISPATCH_REASON!r}) [dispatch, mechanical]"
            )
            changed += 1

    with open(OUTPUT_PATH, "w") as f:
        for rec in moments:
            f.write(json.dumps(rec) + "\n")

    print()
    print(f"Merged {len(verdicts)} rejudged verdict(s) + {len(dispatch_ids)} dispatch null-reason(s)")
    if skipped:
        print(f"Skipped (rejudge skipped_reason set, left untouched): {skipped}")
    print(f"{changed} of {len(moments)} moment records changed")
    print(f"Wrote {OUTPUT_PATH} ({MOMENTS_PATH.name} NOT modified)")


if __name__ == "__main__":
    sys.exit(main())
