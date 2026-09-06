#!/usr/bin/env python3
"""
Merge the all-nulls rejudge results (results/rejudge-all-nulls.jsonl) into a
NEW copy of results/moments.jsonl, written to results/moments.jsonl.new.

Never modifies moments.jsonl in place -- the caller inspects the printed
old->new lines and swaps the file in deliberately.

Rules (per the verification hand-off):
- For each judged field value in a NON-skipped rejudge record:
  - value true/false: set the field and <field>_provenance = "rejudge-all-nulls".
  - value "not_applicable": leave the field null and set
    <field>_null_reason = "judged_not_applicable".
- For each memory_existed recheck:
  - a resolved recheck (memory_existed true/false) sets finding_side's
    memory_existed / memory_kind / memory_generic_or_specific /
    existence_derived_or_estimate / existence_check_error from the recheck,
    plus memory_existed_provenance = "rejudge-all-nulls".
  - an unresolved recheck stamps memory_existed_null_reason with the honest
    recorded reason (its existence_check_error / failure reason); if no
    reason was recorded, that is an error (fail loud, never invent one).
- ALSO (mechanical, no LLM) stamp the structurally-N/A nulls corpus-wide:
  - search_targeted_right_thing null + search_ran == "none"
      -> search_targeted_right_thing_null_reason = "not_applicable_no_search_ran"
  - outdated_outranked_replacement / strength_mismatch_flagged null
    + surfaced != true -> <field>_null_reason = "not_applicable_nothing_surfaced"
  - note_well_targeted / note_not_duplicate / note_superseded_correctly null
    + note_written != true -> <field>_null_reason = "not_applicable_no_note_written"
  - worth_learning_from or memory_existed still null after the merge
      -> <field>_null_reason = the honest recorded reason (the rejudge record's
         skipped_reason / the recheck's recorded failure); fail loud if none
         was recorded.

Fail-loud invariants (never merge into an unexpected state):
- every rejudge moment_id must exist in moments.jsonl, at most once each;
- every judged field must be one this moment's current state actually targets
  (same predicates the rejudge run derived its population from);
- every judged field must still be null in moments.jsonl, with no existing
  <field>_null_reason or <field>_provenance key (a non-null value or an
  existing stamp means this merge already ran -- refuse to double-merge);
- every judged value must be true, false, or "not_applicable";
- memory_existed rechecks only apply where memory_existed is still null and
  unstamped;
- after applying + stamping, NO tracked field may remain null without a
  <field>_null_reason (nothing silently unexplained).

Writes results/moments.jsonl.new only. Usage: python3 merge_all_nulls.py
"""

import json
import sys
from pathlib import Path

_SCRIPT_DIR = Path(__file__).resolve().parent
MOMENTS_PATH = _SCRIPT_DIR / "results" / "moments.jsonl"
REJUDGE_PATH = _SCRIPT_DIR / "results" / "rejudge-all-nulls.jsonl"
OUTPUT_PATH = _SCRIPT_DIR / "results" / "moments.jsonl.new"

PROVENANCE = "rejudge-all-nulls"
NA_REASON = "judged_not_applicable"
NO_SEARCH_REASON = "not_applicable_no_search_ran"
NOTHING_SURFACED_REASON = "not_applicable_nothing_surfaced"
NO_NOTE_REASON = "not_applicable_no_note_written"

FINDING_FIELDS = frozenset(
    ["search_targeted_right_thing", "outdated_outranked_replacement"]
)
WRITING_FIELDS = frozenset(
    [
        "strength_mismatch_flagged",
        "note_well_targeted",
        "note_not_duplicate",
        "note_superseded_correctly",
        "worth_learning_from",
    ]
)
ALL_FIELDS = FINDING_FIELDS | WRITING_FIELDS
NOTE_FIELDS = frozenset(
    ["note_well_targeted", "note_not_duplicate", "note_superseded_correctly"]
)
RECHECK_COMPANION_KEYS = (
    "memory_kind",
    "memory_generic_or_specific",
    "existence_derived_or_estimate",
    "existence_check_error",
)


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


def _side(moment: dict, field: str) -> dict:
    return moment["finding_side"] if field in FINDING_FIELDS else moment["writing_side"]


def _targeted_fields(moment: dict) -> set:
    """The null fields this moment's CURRENT state targets -- the same
    predicates rejudge_all_nulls.py derived its population from."""
    finding = moment.get("finding_side") or {}
    writing = moment.get("writing_side") or {}
    fields = set()
    if (
        finding.get("search_targeted_right_thing") is None
        and finding.get("memory_existed") is True
        and finding.get("search_ran") not in (None, "none")
    ):
        fields.add("search_targeted_right_thing")
    if (
        finding.get("outdated_outranked_replacement") is None
        and finding.get("surfaced") is True
    ):
        fields.add("outdated_outranked_replacement")
    if writing.get("strength_mismatch_flagged") is None and finding.get("surfaced") is True:
        fields.add("strength_mismatch_flagged")
    for note_field in NOTE_FIELDS:
        if writing.get(note_field) is None and writing.get("note_written") is True:
            fields.add(note_field)
    if writing.get("worth_learning_from") is None:
        fields.add("worth_learning_from")
    return fields


def _assert_unstamped(moment_id: str, side: dict, field: str) -> None:
    if side.get(field) is not None:
        raise ValueError(
            f"{moment_id}: {field} is already {side[field]!r} -- "
            "this merge appears to have already run (refusing to double-merge)"
        )
    for key in (f"{field}_null_reason", f"{field}_provenance"):
        if key in side:
            raise ValueError(
                f"{moment_id}: already has {key}={side[key]!r} -- refusing to double-merge"
            )


def main() -> None:
    moments = _load_jsonl(MOMENTS_PATH)
    rejudged = _load_jsonl(REJUDGE_PATH)

    moments_by_id = {}
    for rec in moments:
        moment_id = rec["moment_id"]
        if moment_id in moments_by_id:
            raise ValueError(f"duplicate moment_id in {MOMENTS_PATH}: {moment_id}")
        moments_by_id[moment_id] = rec

    # --- Validate every rejudge record against the current moments state. ---
    judged = {}  # moment_id -> {field: value}
    rechecks = {}  # moment_id -> recheck dict
    skipped = {}  # moment_id -> skipped_reason
    seen_rejudge_ids = set()
    for rec in rejudged:
        moment_id = rec["moment_id"]
        if moment_id in seen_rejudge_ids:
            raise ValueError(f"duplicate rejudge record for {moment_id}")
        seen_rejudge_ids.add(moment_id)
        if moment_id not in moments_by_id:
            raise KeyError(f"rejudge target {moment_id} not found in {MOMENTS_PATH}")
        moment = moments_by_id[moment_id]

        if rec.get("skipped_reason") is not None:
            skipped[moment_id] = rec["skipped_reason"]
        else:
            targeted = _targeted_fields(moment)
            fields = rec.get("fields") or {}
            unknown = set(fields) - ALL_FIELDS
            if unknown:
                raise ValueError(f"{moment_id}: unknown judged field(s) {sorted(unknown)}")
            untargeted = set(fields) - targeted
            if untargeted:
                raise ValueError(
                    f"{moment_id}: judged field(s) {sorted(untargeted)} are not null-and-"
                    "targeted in the current moments.jsonl -- the file changed or this "
                    "merge already ran (refusing to double-merge)"
                )
            verdicts = {}
            for field, judgment in fields.items():
                value = judgment.get("value")
                if value not in (True, False, "not_applicable"):
                    raise ValueError(f"{moment_id}: {field} has invalid value {value!r}")
                _assert_unstamped(moment_id, _side(moment, field), field)
                verdicts[field] = value
            if verdicts:
                judged[moment_id] = verdicts

        recheck = rec.get("memory_existed_recheck")
        if recheck is not None:
            finding = moment["finding_side"]
            if finding.get("memory_existed") is not None:
                raise ValueError(
                    f"{moment_id}: memory_existed is already "
                    f"{finding['memory_existed']!r} -- refusing to double-merge"
                )
            for key in ("memory_existed_null_reason", "memory_existed_provenance"):
                if key in finding:
                    raise ValueError(
                        f"{moment_id}: finding_side already has {key}="
                        f"{finding[key]!r} -- refusing to double-merge"
                    )
            rechecks[moment_id] = recheck

    # --- Apply, preserving moments.jsonl's record order. ---
    applied = 0
    na_stamped = 0
    recheck_applied = 0
    mechanical_stamped = 0

    def stamp(moment_id: str, side: dict, field: str, reason: str, label: str) -> None:
        nonlocal mechanical_stamped
        key = f"{field}_null_reason"
        if key in side:
            raise ValueError(
                f"{moment_id}: already has {key}={side[key]!r} -- refusing to double-merge"
            )
        side[key] = reason
        mechanical_stamped += 1
        print(f"{moment_id}: {field} null -> null ({key}={reason!r}) [{label}]")

    for rec in moments:
        moment_id = rec["moment_id"]
        finding = rec["finding_side"]
        writing = rec["writing_side"]

        # 1. Judged field values.
        for field, value in (judged.get(moment_id) or {}).items():
            side = _side(rec, field)
            if value == "not_applicable":
                side[f"{field}_null_reason"] = NA_REASON
                na_stamped += 1
                print(
                    f"{moment_id}: {field} null -> null "
                    f"({field}_null_reason={NA_REASON!r})"
                )
            else:
                side[field] = value
                side[f"{field}_provenance"] = PROVENANCE
                applied += 1
                print(
                    f"{moment_id}: {field} null -> {value!r} "
                    f"({field}_provenance={PROVENANCE!r})"
                )

        # 2. memory_existed recheck results.
        if moment_id in rechecks:
            recheck = rechecks[moment_id]
            if recheck.get("memory_existed") is not None:
                finding["memory_existed"] = recheck["memory_existed"]
                for key in RECHECK_COMPANION_KEYS:
                    finding[key] = recheck.get(key)
                finding["memory_existed_provenance"] = PROVENANCE
                recheck_applied += 1
                print(
                    f"{moment_id}: memory_existed null -> {recheck['memory_existed']!r} "
                    f"(kind={recheck.get('memory_kind')!r}, "
                    f"{recheck.get('existence_derived_or_estimate')!r}, "
                    f"memory_existed_provenance={PROVENANCE!r})"
                )
            else:
                reason = recheck.get("existence_check_error") or recheck.get(
                    "existence_recheck_failed_reason"
                )
                if not reason:
                    raise ValueError(
                        f"{moment_id}: unresolved memory_existed recheck recorded no "
                        "honest failure reason -- refusing to invent one"
                    )
                stamp(moment_id, finding, "memory_existed", reason, "recheck unresolved")

        # 3. Mechanical structurally-N/A stamps (evaluated on post-apply state).
        if (
            finding.get("search_targeted_right_thing") is None
            and "search_targeted_right_thing_null_reason" not in finding
            and finding.get("search_ran") == "none"
        ):
            stamp(
                moment_id,
                finding,
                "search_targeted_right_thing",
                NO_SEARCH_REASON,
                "mechanical",
            )
        if (
            finding.get("outdated_outranked_replacement") is None
            and "outdated_outranked_replacement_null_reason" not in finding
            and finding.get("surfaced") is not True
        ):
            stamp(
                moment_id,
                finding,
                "outdated_outranked_replacement",
                NOTHING_SURFACED_REASON,
                "mechanical",
            )
        if (
            writing.get("strength_mismatch_flagged") is None
            and "strength_mismatch_flagged_null_reason" not in writing
            and finding.get("surfaced") is not True
        ):
            stamp(
                moment_id,
                writing,
                "strength_mismatch_flagged",
                NOTHING_SURFACED_REASON,
                "mechanical",
            )
        for note_field in sorted(NOTE_FIELDS):
            if (
                writing.get(note_field) is None
                and f"{note_field}_null_reason" not in writing
                and writing.get("note_written") is not True
            ):
                stamp(moment_id, writing, note_field, NO_NOTE_REASON, "mechanical")

        # 4. Still-null worth_learning_from / memory_existed carry the honest
        #    recorded reason (a skipped rejudge / unresolved recheck).
        if (
            writing.get("worth_learning_from") is None
            and "worth_learning_from_null_reason" not in writing
        ):
            reason = skipped.get(moment_id)
            if not reason:
                raise ValueError(
                    f"{moment_id}: worth_learning_from is still null with no recorded "
                    "reason (not judged, not skipped) -- refusing to leave it unexplained"
                )
            stamp(
                moment_id,
                writing,
                "worth_learning_from",
                f"rejudge_skipped: {reason}",
                "recorded reason",
            )
        if (
            finding.get("memory_existed") is None
            and "memory_existed_null_reason" not in finding
        ):
            reason = skipped.get(moment_id)
            if not reason:
                raise ValueError(
                    f"{moment_id}: memory_existed is still null with no recheck and no "
                    "recorded reason -- refusing to leave it unexplained"
                )
            stamp(
                moment_id,
                finding,
                "memory_existed",
                f"rejudge_skipped: {reason}",
                "recorded reason",
            )

    # --- Final sweep: nothing tracked may remain null and unexplained. ---
    for rec in moments:
        moment_id = rec["moment_id"]
        for field in sorted(ALL_FIELDS) + ["memory_existed"]:
            side = (
                rec["finding_side"]
                if field in FINDING_FIELDS or field == "memory_existed"
                else rec["writing_side"]
            )
            if side.get(field) is None and f"{field}_null_reason" not in side:
                raise ValueError(
                    f"{moment_id}: {field} is null with no {field}_null_reason after "
                    "the merge -- the stamp rules missed it (fail loud, fix the rules)"
                )

    with open(OUTPUT_PATH, "w") as f:
        for rec in moments:
            f.write(json.dumps(rec) + "\n")

    print()
    print(
        f"Applied {applied} judged value(s), {na_stamped} judged-not-applicable "
        f"stamp(s), {recheck_applied} memory_existed recheck(s), "
        f"{mechanical_stamped} mechanical/recorded-reason stamp(s)"
    )
    if skipped:
        print(f"Skipped rejudge records (reasons carried into stamps): {sorted(skipped)}")
    print(f"Wrote {OUTPUT_PATH} ({MOMENTS_PATH.name} NOT modified)")


if __name__ == "__main__":
    sys.exit(main())
