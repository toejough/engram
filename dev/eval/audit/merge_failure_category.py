#!/usr/bin/env python3
"""Selectively merge rejudge-failure-category.jsonl into moments.jsonl.

Reads moments.jsonl (the full moment corpus) and
rejudge-failure-category.jsonl (the targeted failure_category re-judgment
pass, keyed by moment_id). For every JUDGED rejudge record -- one whose
skipped_reason is null -- updates ONLY one field on the matching
moments.jsonl record:

    failure_category = rejudge record's "new_failure_category"

Skipped rejudge records (skipped_reason non-null, e.g. "transcript deleted
by retention") are ignored: their moments keep failure_category as-is.
Every other field on every record is left unchanged. The full updated list
(all records, original order) is written to moments.jsonl.new --
moments.jsonl itself is never modified.

This script performs no destructive action on its own: it only ever writes
to the .new output path. Promoting moments.jsonl.new over moments.jsonl is
a separate, explicit step done after independent review.
"""

from __future__ import annotations

import json
from pathlib import Path

RESULTS_DIR = Path(__file__).resolve().parent / "results"
MOMENTS_PATH = RESULTS_DIR / "moments.jsonl"
REJUDGE_PATH = RESULTS_DIR / "rejudge-failure-category.jsonl"
OUTPUT_PATH = RESULTS_DIR / "moments.jsonl.new"


def read_jsonl(path: Path) -> list[dict]:
    """Read a JSONL file into a list of dicts, preserving line order."""
    records = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            records.append(json.loads(line))
    return records


def read_judged_by_moment_id(path: Path) -> dict[str, dict]:
    """Read the rejudge JSONL file into a dict keyed by moment_id.

    Only JUDGED records (skipped_reason is null) are included: a skipped
    target carries no new judgment, so its moment must keep its existing
    failure_category untouched.
    """
    judged_by_id: dict[str, dict] = {}
    for record in read_jsonl(path):
        if record["skipped_reason"] is not None:
            continue
        judged_by_id[record["moment_id"]] = record
    return judged_by_id


def merge(moments: list[dict], judged_by_id: dict[str, dict]) -> list[tuple[str, object, object]]:
    """Apply rejudged failure_category values to moments in place.

    Returns a list of (moment_id, old_failure_category, new_failure_category)
    tuples for every record that was updated, in the order encountered.
    """
    updates: list[tuple[str, object, object]] = []
    for moment in moments:
        moment_id = moment.get("moment_id")
        judged_record = judged_by_id.get(moment_id)
        if judged_record is None:
            continue

        old_failure_category = moment["failure_category"]
        new_failure_category = judged_record["new_failure_category"]

        moment["failure_category"] = new_failure_category

        updates.append((moment_id, old_failure_category, new_failure_category))
    return updates


def write_jsonl(path: Path, records: list[dict]) -> None:
    """Write records as JSONL, one compact JSON object per line."""
    with path.open("w", encoding="utf-8") as f:
        for record in records:
            f.write(json.dumps(record))
            f.write("\n")


def main() -> None:
    moments = read_jsonl(MOMENTS_PATH)
    judged_by_id = read_judged_by_moment_id(REJUDGE_PATH)

    updates = merge(moments, judged_by_id)

    write_jsonl(OUTPUT_PATH, moments)

    print(f"Updated {len(updates)} of {len(moments)} records.")
    for moment_id, old_failure_category, new_failure_category in updates:
        print(f"{moment_id}: failure_category {old_failure_category!r} -> {new_failure_category!r}")


if __name__ == "__main__":
    main()
