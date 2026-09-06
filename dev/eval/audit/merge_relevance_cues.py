#!/usr/bin/env python3
"""
Merge Tier-1 relevance measurement results into moments.jsonl.

Task 1 (memory-loop-audit): For each of the 98 relevance-cues records, find
its moment in moments.jsonl and add to finding_side:
  - relevance_grade (from the record)
  - relevant_path, fire_cue, cue_name (null unless cue_present_missed)
  - candidate_cue (null unless no_cue_exists)
  - relevant_after_arbitration: true ONLY for agent-afdea369b5bac50d9.jsonl#4;
    false for the other 5 initially-relevant records (with arbitration_note);
    null for nothing_relevant/skipped
  - tier1_provenance: 'tier1-relevance-cues'

For the 2 skipped records, add tier1_skipped_reason instead of grades.

Fail-loud guards: unknown moment_id -> raise; field already present -> raise.
Write moments.jsonl.new, print per-moment summary. Then verify the diff
(150 records, only the 98 changed, only the new keys added), then promote:
mv results/moments.jsonl.new results/moments.jsonl
"""
import json
import sys
from pathlib import Path
from collections import defaultdict

CUES = Path("/Users/joe/repos/personal/engram/dev/eval/audit/results/relevance-cues.jsonl")
ARB = Path("/Users/joe/repos/personal/engram/dev/eval/audit/results/relevance-arbitration.json")
MOMENTS = Path("/Users/joe/repos/personal/engram/dev/eval/audit/results/moments.jsonl")
OUT = Path("/Users/joe/repos/personal/engram/dev/eval/audit/results/moments.jsonl.new")

# Read moments into a dict by moment_id
print("Reading moments.jsonl...")
moments_by_id = {}
with MOMENTS.open() as f:
    for line in f:
        if line.strip():
            rec = json.loads(line)
            moments_by_id[rec["moment_id"]] = rec

N_MOMENTS = len(moments_by_id)
assert N_MOMENTS == 150, f"Expected 150 moments, got {N_MOMENTS}"
print(f"  Loaded {N_MOMENTS} moments")

# Read arbitration verdicts
print("Reading relevance-arbitration.json...")
with ARB.open() as f:
    arb = json.load(f)

# Build map of moment_id -> arbitration verdict
# "undisputed_top1" -> relevant_after_arbitration = true
# "disputed" (when existed==true & would_have_changed==false) -> relevant_after_arbitration = false
relevant_after_arbitration = {}
for item in arb.get("undisputed_top1", []):
    relevant_after_arbitration[item] = True

for item in arb.get("disputed", []):
    relevant_after_arbitration[item["moment_id"]] = False

for item in arb.get("in_top5_arbitration", []):
    relevant_after_arbitration[item["moment_id"]] = False

print(f"  Loaded arbitration data: {len(relevant_after_arbitration)} disputed/undisputed")
print(f"    - undisputed_top1: {arb['final_relevant_top1_count']}")
print(f"    - initially-relevant but overturned: {len(arb.get('disputed', [])) + len(arb.get('in_top5_arbitration', []))}")

# Read relevance-cues
print("Reading relevance-cues.jsonl...")
cues = []
skipped_ids = []
with CUES.open() as f:
    for line in f:
        if line.strip():
            rec = json.loads(line)
            if rec.get("skipped_reason"):
                skipped_ids.append(rec["moment_id"])
            cues.append(rec)

N_CUES = len(cues)
N_SKIPPED = len(skipped_ids)
N_JUDGED = N_CUES - N_SKIPPED
print(f"  Loaded {N_CUES} cues ({N_JUDGED} judged, {N_SKIPPED} skipped)")

# Merge
print("\nMerging...")
changes_by_grade = defaultdict(int)
merged_moment_ids = set()

for cue in cues:
    moment_id = cue["moment_id"]
    merged_moment_ids.add(moment_id)

    # Fail-loud guard 1: unknown moment_id
    if moment_id not in moments_by_id:
        raise ValueError(f"FAIL-LOUD: moment_id {moment_id} not found in moments.jsonl")

    m = moments_by_id[moment_id]
    fs = m["finding_side"]

    # Fail-loud guard 2: field already present
    if "tier1_provenance" in fs:
        raise ValueError(f"FAIL-LOUD: moment {moment_id} already has tier1_provenance")

    # Handle skipped vs judged
    if cue["skipped_reason"]:
        # Skipped: add tier1_skipped_reason instead of grades
        fs["tier1_skipped_reason"] = cue["skipped_reason"]
        fs["tier1_provenance"] = "tier1-relevance-cues"
        changes_by_grade["SKIPPED"] += 1
    else:
        # Judged: add all the grade fields
        relevance_grade = cue["relevance_grade"]

        # Determine relevant_after_arbitration
        relevant_after_arb = None
        arbitration_note = None
        if relevance_grade == "nothing_relevant":
            relevant_after_arb = None  # null per spec
        else:
            # It was initially relevant (relevant_top1 or relevant_in_top5)
            if moment_id in relevant_after_arbitration:
                relevant_after_arb = relevant_after_arbitration[moment_id]
                if relevant_after_arb is False:
                    arbitration_note = "overturned by strict counterfactual arbitration 2026-09-05"
            else:
                # Should not happen for initial relevant records
                raise ValueError(
                    f"FAIL-LOUD: moment {moment_id} has relevance_grade={relevance_grade} "
                    f"but is not in arbitration data"
                )

        # Add fields to finding_side
        fs["relevance_grade"] = relevance_grade
        fs["relevant_path"] = cue.get("relevant_path")
        fs["fire_cue"] = cue.get("fire_cue")

        # cue_name is only non-null if fire_cue == "cue_present_missed"
        if cue.get("fire_cue") == "cue_present_missed":
            fs["cue_name"] = cue.get("cue_name")
        else:
            fs["cue_name"] = None

        # candidate_cue is only non-null if fire_cue == "no_cue_exists"
        if cue.get("fire_cue") == "no_cue_exists":
            fs["candidate_cue"] = cue.get("candidate_cue")
        else:
            fs["candidate_cue"] = None

        fs["relevant_after_arbitration"] = relevant_after_arb
        if arbitration_note:
            fs["arbitration_note"] = arbitration_note

        fs["tier1_provenance"] = "tier1-relevance-cues"

        changes_by_grade[relevance_grade] += 1

print(f"  Changes by grade:")
for grade in sorted(changes_by_grade.keys()):
    print(f"    {grade}: {changes_by_grade[grade]}")

# Verify 98 were merged
assert len(merged_moment_ids) == 98, f"Expected to merge 98, got {len(merged_moment_ids)}"

# Write new moments file
print(f"\nWriting {OUT}...")
with OUT.open("w") as f:
    for moment_id in moments_by_id:
        rec = moments_by_id[moment_id]
        f.write(json.dumps(rec) + "\n")

print(f"  Wrote {N_MOMENTS} moments to {OUT}")

# Print per-moment summary (sample of changed moments)
print(f"\n=== Per-moment summary (sample of {min(3, len(merged_moment_ids))} changed moments) ===")
sample_count = 0
for moment_id in sorted(merged_moment_ids):
    if sample_count >= 3:
        break
    m = moments_by_id[moment_id]
    fs = m["finding_side"]
    grade = fs.get("relevance_grade")
    after_arb = fs.get("relevant_after_arbitration")
    skipped_reason = fs.get("tier1_skipped_reason")

    if skipped_reason:
        print(f"  {moment_id}: SKIPPED ({skipped_reason})")
    else:
        arb_note = f" [overturned]" if fs.get("arbitration_note") else ""
        print(f"  {moment_id}: grade={grade}, after_arb={after_arb}{arb_note}")
    sample_count += 1

if len(merged_moment_ids) > 3:
    print(f"  ... ({len(merged_moment_ids) - 3} more)")

# Verify diff: only the 98 moments changed, only new keys added
print(f"\n=== Verifying diff ===")
with MOMENTS.open() as f_old, OUT.open() as f_new:
    old_moments = {rec["moment_id"]: rec for line in f_old if line.strip()
                   for rec in [json.loads(line)]}
    new_moments = {rec["moment_id"]: rec for line in f_new if line.strip()
                   for rec in [json.loads(line)]}

assert len(old_moments) == N_MOMENTS and len(new_moments) == N_MOMENTS
print(f"  Both files have {N_MOMENTS} moments: OK")

# Check that only 98 were changed
changed_count = 0
unchanged_count = 0
for moment_id in old_moments:
    if moment_id not in new_moments:
        raise ValueError(f"FAIL-LOUD: moment {moment_id} missing from new file")

    old_m = old_moments[moment_id]
    new_m = new_moments[moment_id]

    if old_m != new_m:
        changed_count += 1
    else:
        unchanged_count += 1

assert changed_count == 98, f"Expected 98 changed, got {changed_count}"
assert unchanged_count == 52, f"Expected 52 unchanged, got {unchanged_count}"
print(f"  Changed: {changed_count} (expected 98): OK")
print(f"  Unchanged: {unchanged_count} (expected 52): OK")

# Check that only new keys were added (no deletions, no overwrites of old keys)
new_tier1_keys = {"relevance_grade", "relevant_path", "fire_cue", "cue_name",
                  "candidate_cue", "relevant_after_arbitration", "arbitration_note",
                  "tier1_provenance", "tier1_skipped_reason"}

for moment_id in merged_moment_ids:
    old_m = old_moments[moment_id]
    new_m = new_moments[moment_id]

    # Check structure is the same
    old_keys = set(old_m.keys())
    new_keys = set(new_m.keys())

    if old_keys != new_keys:
        raise ValueError(f"FAIL-LOUD: moment {moment_id} has different top-level keys")

    # Check finding_side has only new fields
    old_fs = old_m.get("finding_side", {})
    new_fs = new_m.get("finding_side", {})

    added_keys = set(new_fs.keys()) - set(old_fs.keys())
    removed_keys = set(old_fs.keys()) - set(new_fs.keys())

    if removed_keys:
        raise ValueError(f"FAIL-LOUD: moment {moment_id} lost keys: {removed_keys}")

    # Check added keys are only tier1 keys
    for key in added_keys:
        if key not in new_tier1_keys:
            raise ValueError(f"FAIL-LOUD: moment {moment_id} added unexpected key: {key}")

    # Check old values are unchanged
    for key in old_fs:
        if old_fs[key] != new_fs[key]:
            raise ValueError(
                f"FAIL-LOUD: moment {moment_id} finding_side[{key}] changed from "
                f"{old_fs[key]} to {new_fs[key]}"
            )

print(f"  Field-level integrity: OK (only new tier1 keys added, no overwrites)")
print(f"\n✓ Merge verification passed")
print(f"\n>>> Ready to promote: mv {OUT} {MOMENTS}")
