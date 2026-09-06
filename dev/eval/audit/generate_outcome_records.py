#!/usr/bin/env python3
"""Task 5.3 (memory-loop-audit): derive draft outcome records from the finalized
dev/eval/audit/results/moments.jsonl (150 real audited moments) and
dev/eval/audit/results/transcript-events.jsonl (568 real mechanical-extractor
records), per design.md D-F and spec.md's outcome-record schema:

    {moment_id, note_ref, situation_text,
     outcome ∈ {applied-helped, applied-hurt, surfaced-ignored,
                injected-unused, absent-needed},
     ts, evidence{transcript, anchor}}

This script is purely derivational: it reads the two already-finalized JSONL
files and writes dev/eval/audit/results/outcome-records.jsonl. It makes no
LLM calls, and it makes no vault writes of any kind (`engram learn`/`amend`/
`activate`/... are never invoked) -- the one place it touches vault content
is read-only, inherited from `_resolve_fork_parent_context` (imported from
audit_moments.py, not reimplemented), which opens vault note files under
~/.local/share/engram/vault to read their text back for a fork-context
resolution. Per spec.md: "The audit SHALL NOT write, amend, activate,
supersede, or remove any vault note."

Derivation rules (see dev/eval/audit/CREDIT_SCHEMA.md for the full writeup):

  RULE 1 (applied-helped / applied-hurt / surfaced-ignored / injected-unused)
    Applies when moment_type in (success, failure, rework) AND
    finding_side.followed is not null. Resolves note_ref candidates from
    either the fork-parent-context (search_ran == "injected") or the last
    qualifying recall_call in transcript-events.jsonl (search_ran in
    (quick_glance, full_recall)); emits one record per distinct candidate
    note path, or one record with note_ref: null if resolution fails.
    search_ran == "none" means Rule 1 does not apply at all (no record).

  RULE 2 (absent-needed)
    Applies when moment_type in (failure, rework). Emits a record when
    finding_side.memory_existed is false, or when memory_existed is true but
    surfaced is false. A moment can independently produce records from BOTH
    rules.

Re-run this script against the same moments.jsonl / transcript-events.jsonl
to reproduce the same outcome-records.jsonl byte-for-byte (deterministic;
no randomness, no network calls).
"""

from __future__ import annotations

import json
import sys
from collections import Counter
from pathlib import Path
from typing import Any, Dict, List, Optional

AUDIT_DIR = Path(__file__).resolve().parent
RESULTS_DIR = AUDIT_DIR / "results"
MOMENTS_PATH = RESULTS_DIR / "moments.jsonl"
TRANSCRIPT_EVENTS_PATH = RESULTS_DIR / "transcript-events.jsonl"
OUTPUT_PATH = RESULTS_DIR / "outcome-records.jsonl"

# audit_moments.py lives alongside this script. _resolve_fork_parent_context
# already exists there and is imported, not reimplemented, per task 5.3's
# instructions.
sys.path.insert(0, str(AUDIT_DIR))
from audit_moments import _resolve_fork_parent_context  # noqa: E402  (import after sys.path setup)

RULE1_MOMENT_TYPES = ("success", "failure", "rework")
RULE2_MOMENT_TYPES = ("failure", "rework")

# finding_side.followed values that count as "surfaced but not applied as-is".
IGNORED_LIKE_FOLLOWED = ("ignored", "skipped_step", "out_of_order", "partial")

UNRESOLVED_NOTE_REF_TEXT = "note path could not be resolved (source data missing or resolution failed)"

OUTCOME_TAGS = (
    "applied-helped",
    "applied-hurt",
    "surfaced-ignored",
    "injected-unused",
    "absent-needed",
)


def read_jsonl(path: Path) -> List[Dict[str, Any]]:
    """Read a JSONL file into a list of dicts, preserving line order."""
    records = []
    with path.open("r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            records.append(json.loads(line))
    return records


def index_transcript_events_by_path(events: List[Dict[str, Any]]) -> Dict[str, Dict[str, Any]]:
    """Index transcript-events.jsonl records by transcript_path for exact-string lookup."""
    return {event["transcript_path"]: event for event in events}


def dedupe_preserve_order(paths: List[str]) -> List[str]:
    """Drop exact-duplicate paths, keeping first-seen order."""
    seen = set()
    deduped = []
    for path in paths:
        if path not in seen:
            seen.add(path)
            deduped.append(path)
    return deduped


def evidence_of(moment: Dict[str, Any]) -> Dict[str, Any]:
    """Copy the moment's evidence sub-object verbatim (transcript, anchor only)."""
    evidence = moment["evidence"]
    return {"transcript": evidence["transcript"], "anchor": evidence["anchor"]}


def resolve_note_ref_candidates(
    moment: Dict[str, Any], events_by_transcript_path: Dict[str, Dict[str, Any]]
) -> List[str]:
    """Resolve the note_ref candidates (distinct vault note paths) for a Rule 1 moment.

    Only called for search_ran in ("injected", "quick_glance", "full_recall") --
    the caller is responsible for skipping search_ran == "none" moments entirely
    (Rule 1 does not apply there; see task 5.3 spec step 1a's "Else" branch).
    """
    search_ran = moment["finding_side"]["search_ran"]

    if search_ran == "injected":
        resolution = _resolve_fork_parent_context(moment["transcript_path"])
        paths = [note["path"] for note in resolution.get("recalled_notes", []) if note.get("path")]
        return dedupe_preserve_order(paths)

    if search_ran in ("quick_glance", "full_recall"):
        event = events_by_transcript_path.get(moment["transcript_path"])
        if event is None:
            return []
        moment_location = moment["location"]
        qualifying = [rc for rc in event["recall_calls"] if rc["location"] <= moment_location]
        if not qualifying:
            return []
        last_recall_call = qualifying[-1]
        paths = [item["path"] for item in last_recall_call["matched_items"] if item.get("path")]
        return dedupe_preserve_order(paths)

    return []


def rule1_outcome_tag(followed: str, search_ran: str) -> Optional[str]:
    """Map (finding_side.followed, finding_side.search_ran) to an outcome tag."""
    if followed == "yes":
        return "applied-helped"
    if followed == "contradicted":
        return "applied-hurt"
    if followed in IGNORED_LIKE_FOLLOWED:
        return "injected-unused" if search_ran == "injected" else "surfaced-ignored"
    return None


def apply_rule1(
    moment: Dict[str, Any], events_by_transcript_path: Dict[str, Dict[str, Any]]
) -> List[Dict[str, Any]]:
    """Rule 1: applied-helped / applied-hurt / surfaced-ignored / injected-unused."""
    finding_side = moment["finding_side"]
    followed = finding_side["followed"]

    if moment["moment_type"] not in RULE1_MOMENT_TYPES or followed is None:
        return []

    search_ran = finding_side["search_ran"]
    if search_ran == "none":
        # Rule 1 does not apply -- skip, no record from this branch.
        return []

    outcome = rule1_outcome_tag(followed, search_ran)
    if outcome is None:
        return []

    note_ref_candidates = resolve_note_ref_candidates(moment, events_by_transcript_path)

    if not note_ref_candidates:
        return [
            {
                "moment_id": moment["moment_id"],
                "note_ref": None,
                "situation_text": UNRESOLVED_NOTE_REF_TEXT,
                "outcome": outcome,
                "ts": moment["timestamp"],
                "evidence": evidence_of(moment),
            }
        ]

    return [
        {
            "moment_id": moment["moment_id"],
            "note_ref": note_ref,
            "situation_text": moment["description"],
            "outcome": outcome,
            "ts": moment["timestamp"],
            "evidence": evidence_of(moment),
        }
        for note_ref in note_ref_candidates
    ]


def apply_rule2(moment: Dict[str, Any]) -> List[Dict[str, Any]]:
    """Rule 2: absent-needed."""
    if moment["moment_type"] not in RULE2_MOMENT_TYPES:
        return []

    finding_side = moment["finding_side"]
    memory_existed = finding_side["memory_existed"]
    surfaced = finding_side["surfaced"]

    if memory_existed is False:
        situation_text = (
            "no relevant memory existed in the vault at this moment ("
            + finding_side["existence_derived_or_estimate"].upper()
            + ")"
        )
    elif memory_existed is True and surfaced is False:
        situation_text = (
            "relevant memory existed in the vault but was not surfaced to the agent (search_ran="
            + finding_side["search_ran"]
            + ", "
            + finding_side["existence_derived_or_estimate"].upper()
            + ")"
        )
    else:
        return []

    return [
        {
            "moment_id": moment["moment_id"],
            "note_ref": None,
            "situation_text": situation_text,
            "outcome": "absent-needed",
            "ts": moment["timestamp"],
            "evidence": evidence_of(moment),
        }
    ]


def derive_outcome_records(
    moments: List[Dict[str, Any]], events_by_transcript_path: Dict[str, Dict[str, Any]]
) -> List[Dict[str, Any]]:
    """Derive all outcome records for all moments (Rule 1 and Rule 2 are independent)."""
    records: List[Dict[str, Any]] = []
    for moment in moments:
        records.extend(apply_rule1(moment, events_by_transcript_path))
        records.extend(apply_rule2(moment))
    return records


def write_jsonl(path: Path, records: List[Dict[str, Any]]) -> None:
    """Write records as JSONL, one compact JSON object per line."""
    with path.open("w", encoding="utf-8") as f:
        for record in records:
            f.write(json.dumps(record))
            f.write("\n")


def main() -> None:
    moments = read_jsonl(MOMENTS_PATH)
    assert len(moments) == 150, f"expected 150 moments, got {len(moments)}"

    events = read_jsonl(TRANSCRIPT_EVENTS_PATH)
    assert len(events) == 568, f"expected 568 transcript-events records, got {len(events)}"

    events_by_transcript_path = index_transcript_events_by_path(events)

    records = derive_outcome_records(moments, events_by_transcript_path)
    write_jsonl(OUTPUT_PATH, records)

    counts = Counter(record["outcome"] for record in records)
    print(f"Wrote {len(records)} outcome records to {OUTPUT_PATH}")
    for outcome in OUTCOME_TAGS:
        print(f"  {outcome}: {counts.get(outcome, 0)}")


if __name__ == "__main__":
    main()
