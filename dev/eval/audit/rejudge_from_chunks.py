#!/usr/bin/env python3
"""
Re-judge failure_category for the 10 rejudge targets whose raw transcripts
were deleted by Claude Code session retention (transcript_exists=false in
results/rejudge-targets.json), using engram's own append-only chunk index
(~/.local/share/engram/chunks) as the surviving conversational record.
Real, billed claude -p sonnet calls.

Joe directed this (2026-09-03): the chunk index ingests every session
transcript BEFORE retention deletes it and never deletes chunks within a
source -- the chunks retain what's valuable, so "transcript deleted by
retention" is not "unjudgeable". Chunk files are named
<transcript-basename-without-.jsonl>-<hash>.jsonl; each line carries real
stripped conversational turn text ("USER: ..." / "ASSISTANT: ...") under a
"turn-N" anchor. Original JSONL line numbers do NOT map to chunk anchors,
so the judge gets whole-transcript turn text, honestly labeled as a
reconstruction, not the exact production window.

Same conventions as rejudge_failure_category.py (which handles the
transcript_exists=true targets), but deliberately independent of it --
imports come straight from audit_moments. Two honesty requirements
distinguish chunk-derived judgments from raw-transcript ones (the audit's
DERIVED/ESTIMATE pattern):

1. Every output record carries "judged_from": "chunks".
2. The judge must self-report chunk_context_sufficient (true/false). If it
   says false, the category is recorded as null with the rationale
   explaining what was missing -- an honest insufficient-context null
   beats a forced guess.

Per-target handling:
- prompt_too_long from claude -p: skipped (deterministic, never retried).
- malformed/invalid-enum/missing-field response: ONE retry, then recorded
  with skipped_reason="unparseable response" -- never invented.
- exhausted_retries from claude -p: the ENTIRE run halts immediately
  (raise ExhaustedRetriesError; the 2026-09-02 silent-auth-outage lesson:
  never log-and-continue past a possible sustained outage).

Output: one JSON line per target (all 10, including skips) appended to
results/rejudge-failure-category-from-chunks.jsonl:
  {"moment_id", "moment_type", "new_failure_category", "rationale",
   "chunk_context_sufficient", "judged_from": "chunks",
   "skipped_reason" (null if judged), "cost_usd"}
cost_usd is the real total_cost_usd claude -p reported for that target's
call(s) -- including a retried call's spend, and including the spend on an
ultimately-unparseable target -- or null when no call was made at all.

The script is RESUMABLE: on start it reads any moment_ids already present
in the output file and skips them (append mode). The final summary prints
only once all 10 targets are present.

Never writes to moments.jsonl, rejudge-failure-category.jsonl,
scorecard-main.json, or anything else under results/ besides
results/rejudge-failure-category-from-chunks.jsonl.

This makes ~10 real `claude -p --model sonnet` calls -- real money, no stub.

Usage: python3 rejudge_from_chunks.py
"""

import glob
import json
import os
import shutil
import sys
import tempfile
from collections import Counter
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

_SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _SCRIPT_DIR)

from audit_moments import (  # noqa: E402  (path insert must precede this import)
    ExhaustedRetriesError,
    PromptTooLongError,  # noqa: F401  (documented contract; prompt_too_long arrives as a response flag)
    _build_cfg_dir,
    _run_claude_p,
)

TARGETS_PATH = Path(_SCRIPT_DIR) / "results" / "rejudge-targets.json"
MOMENTS_PATH = Path(_SCRIPT_DIR) / "results" / "moments.jsonl"
OUTPUT_PATH = Path(_SCRIPT_DIR) / "results" / "rejudge-failure-category-from-chunks.jsonl"
CHUNKS_DIR = Path.home() / ".local" / "share" / "engram" / "chunks"

EXPECTED_TARGET_COUNT = 10

SKIP_TOO_LONG = "prompt too long for claude -p"
SKIP_UNPARSEABLE = "unparseable response"

# The only values the judge may answer with. Anything else (after one
# retry) is recorded as unparseable -- never coerced or invented.
VALID_CATEGORIES = {"fixable", "nothing_could_have_caught_it", "found_but_not_followed", None}

FAILURE_TYPES = {"failure", "correction", "rework"}
SUCCESS_TYPES = {"success", "dispatch"}


def _load_moments_by_id(path: Path) -> Dict[str, Dict[str, Any]]:
    """Load moments.jsonl into a dict keyed by moment_id. Read-only -- never written to."""
    by_id: Dict[str, Dict[str, Any]] = {}
    with open(path, "r") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rec = json.loads(line)
            by_id[rec["moment_id"]] = rec
    return by_id


def _find_chunk_file(transcript_path: str) -> Path:
    """
    Find the single chunk file for a transcript: named
    <basename-without-.jsonl>-<hash>.jsonl in the chunk index. Verified
    2026-09-03: exactly one exists per target basename. Fail loud on
    zero or multiple matches -- never guess.
    """
    basename = os.path.basename(transcript_path)
    stem = basename[: -len(".jsonl")] if basename.endswith(".jsonl") else basename
    matches = sorted(glob.glob(str(CHUNKS_DIR / f"{stem}-*.jsonl")))
    if len(matches) != 1:
        raise FileNotFoundError(
            f"expected exactly 1 chunk file for {basename!r} in {CHUNKS_DIR}, "
            f"found {len(matches)}: {matches}"
        )
    return Path(matches[0])


def _reconstruct_from_chunks(chunk_file: Path) -> str:
    """
    Concatenate all chunk lines' text, stable-sorted by the anchor's turn
    number (anchors are "turn-N"; chunks of the same turn keep their file
    order), into the full surviving conversational record.
    """
    chunks: List[Tuple[int, str]] = []
    with open(chunk_file, "r") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rec = json.loads(line)
            anchor = rec["anchor"]  # "turn-N"
            turn_number = int(anchor.rsplit("-", 1)[1])
            chunks.append((turn_number, rec["text"]))
    chunks.sort(key=lambda pair: pair[0])  # stable: preserves within-turn chunk order
    return "\n".join(text for _, text in chunks)


def _build_prompt(
    chunk_text: str,
    location: int,
    moment_type: str,
    description: str,
    worth_learning_from: Any,
) -> str:
    """
    Build the targeted single-question prompt over the chunk-reconstructed
    record. Same per-type question as rejudge_failure_category.py, plus the
    mandatory chunk_context_sufficient self-report.
    """
    if moment_type in FAILURE_TYPES:
        question = (
            "Could a relevant memory (a lesson from prior work), recalled at the right time, "
            "have prevented or caught this problem? "
            'Answer "fixable" if yes; "nothing_could_have_caught_it" if no plausible memory '
            'could have; "found_but_not_followed" if a relevant memory WAS visible in context '
            "but not followed."
        )
    else:  # success / dispatch
        question = (
            f"This moment was judged worth-learning-from "
            f"(worth_learning_from={json.dumps(worth_learning_from)} in the audit record), "
            "but no learn step fired. "
            'Could a mandatory "record lessons learned in your completion report" step '
            'realistically have captured this lesson? Answer "fixable" if yes; '
            '"nothing_could_have_caught_it" if no plausible mechanism would have; '
            "null if genuinely not applicable."
        )

    return f"""You are auditing one moment from a real agent-session transcript for a memory-loop audit.

The original transcript file was deleted by session retention. The record below is reconstructed \
from engram's chunk index -- stripped conversational turns, tool-call detail reduced, the original \
line numbering is unavailable. It is the FULL surviving conversational record for this session, \
in turn order.

RECONSTRUCTED TRANSCRIPT (from engram's chunk index):
{chunk_text}

MOMENT UNDER JUDGMENT:
- approximate original location: line {location} of the original transcript JSONL (that line \
numbering does not map onto the reconstruction above; use the description to find the moment)
- moment_type: {moment_type}
- description: {description}

QUESTION: {question}

Respond with STRICT JSON only -- no prose before or after:
{{"failure_category": "fixable" | "nothing_could_have_caught_it" | "found_but_not_followed" | null, \
"rationale": "<one sentence>", "chunk_context_sufficient": true | false}}
The "rationale" field is REQUIRED: one sentence justifying the answer.
The "chunk_context_sufficient" field is REQUIRED and must be honest: true only if the reconstructed \
record above actually contained enough context to answer the question; false if it did not (then \
the rationale must say what was missing -- an honest "insufficient context" beats a forced guess)."""


def _parse_response(result_text: str) -> Tuple[bool, Any, Optional[str], Optional[bool]]:
    """
    Parse and validate the model's response.

    Returns (ok, failure_category, rationale, chunk_context_sufficient).
    ok=False means the response was malformed/invalid (bad JSON,
    out-of-enum value, missing/empty rationale, or missing/non-bool
    chunk_context_sufficient) and the caller should retry once, then skip.
    """
    json_text = result_text
    # Handle ```json fences (same scheme as judge_moment's parser).
    if "```json" in json_text:
        json_text = json_text.split("```json")[1].split("```")[0]
    elif "```" in json_text:
        json_text = json_text.split("```")[1].split("```")[0]

    try:
        parsed = json.loads(json_text.strip())
    except json.JSONDecodeError:
        return False, None, None, None

    if not isinstance(parsed, dict) or "failure_category" not in parsed:
        return False, None, None, None

    category = parsed["failure_category"]
    if category not in VALID_CATEGORIES:
        return False, None, None, None

    rationale = parsed.get("rationale")
    if not isinstance(rationale, str) or not rationale.strip():
        return False, None, None, None

    sufficient = parsed.get("chunk_context_sufficient")
    if not isinstance(sufficient, bool):
        return False, None, None, None

    return True, category, rationale.strip(), sufficient


def _load_done_ids(path: Path) -> List[str]:
    """moment_ids already written to the output file (for resume)."""
    if not path.exists():
        return []
    done: List[str] = []
    with open(path, "r") as f:
        for line in f:
            line = line.strip()
            if line:
                done.append(json.loads(line)["moment_id"])
    return done


def _print_final_summary(targets: List[Dict[str, Any]]) -> None:
    """Read the whole output file back and print the run-wide summary."""
    records = []
    with open(OUTPUT_PATH, "r") as f:
        for line in f:
            line = line.strip()
            if line:
                records.append(json.loads(line))

    judged = [r for r in records if r["skipped_reason"] is None]
    skipped = [r for r in records if r["skipped_reason"] is not None]
    insufficient = [r for r in judged if r["chunk_context_sufficient"] is False]
    total_cost = sum(r["cost_usd"] for r in records if r["cost_usd"] is not None)

    print()
    print("=== FINAL SUMMARY (judged_from=chunks) ===")
    print(f"Targets: {len(records)}/{len(targets)} recorded in {OUTPUT_PATH.name}")
    print(f"Judged: {len(judged)}; skipped: {len(skipped)}")
    for reason, count in sorted(Counter(r["skipped_reason"] for r in skipped).items()):
        print(f"  skipped ({reason}): {count}")
    print(
        f"Self-reported insufficient chunk context: {len(insufficient)} "
        "(category recorded as null for these)"
    )
    print("Category tallies by moment_type (judged targets):")
    tally = Counter((r["moment_type"], str(r["new_failure_category"])) for r in judged)
    for (mtype, category), count in sorted(tally.items()):
        print(f"  {mtype:>10} -> {category}: {count}")
    print(
        f"Total real $ cost: ${total_cost:.4f} (sum of per-target cost_usd -- each the "
        "total_cost_usd claude -p's own --output-format json response reported for that "
        "target's real sonnet call(s), across ALL invocations of this script; not an estimate)"
    )


def main() -> None:
    with open(TARGETS_PATH, "r") as f:
        all_targets: List[Dict[str, Any]] = json.load(f)

    targets = [t for t in all_targets if not t["transcript_exists"]]
    if len(targets) != EXPECTED_TARGET_COUNT:
        raise ValueError(
            f"expected exactly {EXPECTED_TARGET_COUNT} transcript_exists=false targets, "
            f"found {len(targets)} -- refusing to run against an unexpected target set"
        )

    moments_by_id = _load_moments_by_id(MOMENTS_PATH)

    # Pre-flight (before any money is spent): every target must have a
    # moments.jsonl record AND exactly one chunk file. Fail loud here
    # rather than mid-run.
    missing = [t["moment_id"] for t in targets if t["moment_id"] not in moments_by_id]
    if missing:
        raise KeyError(f"targets missing from {MOMENTS_PATH}: {missing}")
    chunk_files: Dict[str, Path] = {}  # transcript_path -> chunk file
    for t in targets:
        if t["transcript_path"] not in chunk_files:
            chunk_files[t["transcript_path"]] = _find_chunk_file(t["transcript_path"])

    done_ids = set(_load_done_ids(OUTPUT_PATH))

    # ONE shared cfg_dir (real keychain-seeded creds) reused across all calls.
    cfg_dir = tempfile.mkdtemp(prefix="rejudge-from-chunks-cfg-")
    _build_cfg_dir(cfg_dir)

    # Cache the reconstructed record per transcript: several targets share a
    # transcript, and it is a deterministic pure read.
    reconstruction_cache: Dict[str, str] = {}

    invocation_spend = 0.0

    try:
        with open(OUTPUT_PATH, "a") as out_f:

            def emit(record: Dict[str, Any]) -> None:
                out_f.write(json.dumps(record) + "\n")
                out_f.flush()
                done_ids.add(record["moment_id"])

            for index, target in enumerate(targets, start=1):
                moment_id = target["moment_id"]
                moment_type = target["moment_type"]
                skeleton = {
                    "moment_id": moment_id,
                    "moment_type": moment_type,
                    "new_failure_category": None,
                    "rationale": None,
                    "chunk_context_sufficient": None,
                    "judged_from": "chunks",
                    "skipped_reason": None,
                    "cost_usd": None,
                }

                if moment_id in done_ids:
                    continue  # resume: already recorded by a prior invocation

                print(f"[{index}/{len(targets)}] judging {moment_id} from chunks ...", flush=True)

                transcript_path = target["transcript_path"]
                if transcript_path not in reconstruction_cache:
                    reconstruction_cache[transcript_path] = _reconstruct_from_chunks(
                        chunk_files[transcript_path]
                    )
                moment = moments_by_id[moment_id]

                prompt = _build_prompt(
                    reconstruction_cache[transcript_path],
                    target["location"],
                    moment_type,
                    moment["description"],
                    moment.get("writing_side", {}).get("worth_learning_from"),
                )

                # First attempt + ONE retry on a malformed/invalid response.
                target_cost = 0.0
                outcome: Optional[Dict[str, Any]] = None
                for attempt in (1, 2):
                    response = _run_claude_p(prompt, "sonnet", cfg_dir=cfg_dir)
                    call_cost = response.get("total_cost_usd", 0) or 0
                    target_cost += call_cost
                    invocation_spend += call_cost

                    if response.get("prompt_too_long"):
                        # Deterministic rejection -- never retried (would never succeed).
                        outcome = {**skeleton, "skipped_reason": SKIP_TOO_LONG, "cost_usd": target_cost}
                        break

                    if response.get("exhausted_retries"):
                        # HALT THE ENTIRE RUN: a sustained auth/API outage must never
                        # be absorbed as per-moment skips (2026-09-02 lesson). The
                        # output file keeps everything recorded so far; re-running
                        # this script resumes from here.
                        raise ExhaustedRetriesError(
                            f"claude -p exhausted retries on {moment_id} -- halting the "
                            f"entire run (${invocation_spend:.4f} spent this invocation; "
                            "output so far is recorded; re-run to resume)"
                        )

                    ok, category, rationale, sufficient = _parse_response(response.get("result", ""))
                    if ok:
                        if sufficient is False:
                            # Honest insufficient-context null beats a forced guess:
                            # the rationale explains what was missing.
                            category = None
                        outcome = {
                            **skeleton,
                            "new_failure_category": category,
                            "rationale": rationale,
                            "chunk_context_sufficient": sufficient,
                            "cost_usd": target_cost,
                        }
                        break

                    if attempt == 1:
                        print("  malformed/invalid response; retrying once ...", flush=True)

                if outcome is None:
                    # Both attempts malformed/invalid: record honestly, never invent.
                    outcome = {**skeleton, "skipped_reason": SKIP_UNPARSEABLE, "cost_usd": target_cost}

                emit(outcome)
                if outcome["skipped_reason"] is None:
                    print(
                        f"  -> {outcome['new_failure_category']!r} "
                        f"(chunk_context_sufficient={outcome['chunk_context_sufficient']}; "
                        f"${target_cost:.4f}; invocation spend so far ${invocation_spend:.4f})",
                        flush=True,
                    )
                else:
                    print(f"  skipped: {outcome['skipped_reason']}", flush=True)
    finally:
        shutil.rmtree(cfg_dir, ignore_errors=True)

    if len(done_ids) >= len(targets):
        _print_final_summary(targets)
    else:
        print(
            f"\nIncomplete: {len(done_ids)}/{len(targets)} targets recorded "
            f"(${invocation_spend:.4f} spent this invocation). Re-run to resume.",
            flush=True,
        )


if __name__ == "__main__":
    main()
