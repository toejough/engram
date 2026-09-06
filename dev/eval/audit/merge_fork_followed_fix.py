#!/usr/bin/env python3
"""Selectively merge rescore-fork-followed.jsonl into moments.jsonl.

Reads moments.jsonl (the full moment corpus) and rescore-fork-followed.jsonl
(a rescoring pass produced by a prior step, keyed by moment_id). For every
moment_id present in the rescore file, updates ONLY two fields on the
matching moments.jsonl record:

    finding_side["followed"] = rescore record's "new_followed"
    failure_category         = rescore record's "new_failure_category"

Every other field on that record, and every field on every other record, is
left byte-for-byte unchanged. The full updated list (all records, original
order) is written to moments.jsonl.new -- moments.jsonl itself is never
modified.

This script performs no destructive action on its own: it only ever writes
to the .new output path. Promoting moments.jsonl.new over moments.jsonl is a
separate, explicit step done after independent review.
"""

from __future__ import annotations

import json
from pathlib import Path

RESULTS_DIR = Path(__file__).resolve().parent / "results"
MOMENTS_PATH = RESULTS_DIR / "moments.jsonl"
RESCORE_PATH = RESULTS_DIR / "rescore-fork-followed.jsonl"
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


def read_rescore_by_moment_id(path: Path) -> dict[str, dict]:
    """Read the rescore JSONL file into a dict keyed by moment_id."""
    rescore_by_id: dict[str, dict] = {}
    for record in read_jsonl(path):
        rescore_by_id[record["moment_id"]] = record
    return rescore_by_id


def merge(moments: list[dict], rescore_by_id: dict[str, dict]) -> list[tuple[str, object, object]]:
    """Apply rescore updates to moments in place.

    Returns a list of (moment_id, old_followed, new_followed) tuples for
    every record that was updated, in the order they were encountered.
    """
    updates: list[tuple[str, object, object]] = []
    for moment in moments:
        moment_id = moment.get("moment_id")
        rescore_record = rescore_by_id.get(moment_id)
        if rescore_record is None:
            continue

        old_followed = moment["finding_side"]["followed"]
        new_followed = rescore_record["new_followed"]
        new_failure_category = rescore_record["new_failure_category"]

        moment["finding_side"]["followed"] = new_followed
        moment["failure_category"] = new_failure_category

        updates.append((moment_id, old_followed, new_followed))
    return updates


def write_jsonl(path: Path, records: list[dict]) -> None:
    """Write records as JSONL, one compact JSON object per line."""
    with path.open("w", encoding="utf-8") as f:
        for record in records:
            f.write(json.dumps(record))
            f.write("\n")


def main() -> None:
    moments = read_jsonl(MOMENTS_PATH)
    rescore_by_id = read_rescore_by_moment_id(RESCORE_PATH)

    updates = merge(moments, rescore_by_id)

    write_jsonl(OUTPUT_PATH, moments)

    print(f"Updated {len(updates)} of {len(moments)} records.")
    for moment_id, old_followed, new_followed in updates:
        print(f"{moment_id}: followed {old_followed!r} -> {new_followed!r}")


if __name__ == "__main__":
    main()
