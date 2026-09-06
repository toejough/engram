#!/usr/bin/env python3
"""
Re-judge failure_category for the 67 target moments in
results/rejudge-targets.json whose failure_category is null due to a
now-fixed prompt-scope bug, using real, billed claude -p sonnet calls.

Unlike rescore_fork_followed.py (which re-runs the full judge_moment and
had to deal with merging its side effects), this script asks ONE targeted
question per moment -- it deliberately does NOT re-derive any other
scorecard field. The moment's identity (moment_type, description,
writing_side.worth_learning_from) is read straight from the existing
results/moments.jsonl record; only failure_category (plus a required
one-sentence rationale, the auditable record the original run lacked) is
newly produced.

Per-target handling:
- transcript_exists=false (retention-deleted): skipped, recorded with
  skipped_reason="transcript deleted by retention". No call is made.
- no window of split_into_windows() covers the moment's location: skipped.
- prompt_too_long from claude -p: skipped (deterministic, never retried).
- malformed/invalid-enum/missing-rationale response: ONE retry, then
  recorded with skipped_reason="unparseable response" -- never invented.
- exhausted_retries from claude -p: the ENTIRE run halts immediately
  (raise ExhaustedRetriesError; the 2026-09-02 silent-auth-outage lesson:
  never log-and-continue past a possible sustained outage).

Output: one JSON line per target (all 67, including skips) appended to
results/rejudge-failure-category.jsonl:
  {"moment_id", "moment_type", "new_failure_category", "rationale",
   "skipped_reason" (null if judged), "cost_usd"}
cost_usd is the real total_cost_usd claude -p reported for that target's
call(s) -- including a retried call's spend, and including the spend on an
ultimately-unparseable target (real money was still spent; the tally stays
honest) -- or null when no call was made at all.

The script is RESUMABLE: on start it reads any moment_ids already present
in the output file and skips them (append mode), so an interrupted run
(tool timeout, exhausted-retries halt after auth recovery) picks up
exactly where it stopped without re-paying for completed judgments. The
final summary (tallies by moment_type + total real cost, summed from the
per-line cost_usd values across the WHOLE file, not just this invocation)
prints only once all 67 targets are present.

Never writes to moments.jsonl, scorecard-main.json, or anything else
under results/ besides results/rejudge-failure-category.jsonl.

This makes ~57 real `claude -p --model sonnet` calls -- real money, no stub.

Usage: python3 rejudge_failure_category.py [max_seconds]
  max_seconds (optional): stop cleanly BETWEEN targets once this much wall
  time has elapsed (prints the incomplete-resume notice). Lets a driver
  with a hard tool timeout run the script in foreground chunks without
  ever killing an in-flight claude -p call (which would waste its spend);
  omitted = run to completion.
"""

import json
import os
import shutil
import sys
import tempfile
import time
from collections import Counter
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

_SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _SCRIPT_DIR)

from audit_moments import (  # noqa: E402  (path insert must precede this import)
    ExhaustedRetriesError,
    PromptTooLongError,
    _build_cfg_dir,
    _run_claude_p,
    split_into_windows,
)

TARGETS_PATH = Path(_SCRIPT_DIR) / "results" / "rejudge-targets.json"
MOMENTS_PATH = Path(_SCRIPT_DIR) / "results" / "moments.jsonl"
OUTPUT_PATH = Path(_SCRIPT_DIR) / "results" / "rejudge-failure-category.jsonl"

SKIP_RETENTION = "transcript deleted by retention"
SKIP_NO_WINDOW = "no window covers moment location"
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


def _find_covering_window(windows: List[Dict[str, Any]], location: int) -> Optional[Dict[str, Any]]:
    """Find the single window whose location_start <= location <= location_end."""
    for w in windows:
        if w["location_start"] <= location <= w["location_end"]:
            return w
    return None


def _numbered_window_text(window: Dict[str, Any]) -> str:
    """
    Window text with absolute line numbers -- the SAME numbering scheme
    judge_moment uses: each line prefixed with location_start + offset.
    """
    numbered = []
    for offset, line in enumerate(window["lines"]):
        absolute_line_number = window["location_start"] + offset
        numbered.append(f"{absolute_line_number}: {line}")
    return "".join(numbered)


def _build_prompt(
    window: Dict[str, Any],
    location: int,
    moment_type: str,
    description: str,
    worth_learning_from: Any,
) -> str:
    """
    Build the targeted single-question prompt. NOT the full judge_moment
    prompt -- this deliberately avoids re-deriving other fields (and the
    side-effect-merge problem the fork rescore hit).
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

WINDOW (lines {window['location_start']}-{window['location_end']}; each line is prefixed with its absolute line number):
{_numbered_window_text(window)}

MOMENT UNDER JUDGMENT:
- location: line {location}
- moment_type: {moment_type}
- description: {description}

QUESTION: {question}

Respond with STRICT JSON only -- no prose before or after:
{{"failure_category": "fixable" | "nothing_could_have_caught_it" | "found_but_not_followed" | null, "rationale": "<one sentence>"}}
The "rationale" field is REQUIRED: one sentence justifying the answer."""


def _parse_response(result_text: str) -> Tuple[bool, Any, Optional[str]]:
    """
    Parse and validate the model's response.

    Returns (ok, failure_category, rationale). ok=False means the response
    was malformed/invalid (bad JSON, out-of-enum value, or missing/empty
    rationale) and the caller should retry once, then skip.
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
        return False, None, None

    if not isinstance(parsed, dict) or "failure_category" not in parsed:
        return False, None, None

    category = parsed["failure_category"]
    if category not in VALID_CATEGORIES:
        return False, None, None

    rationale = parsed.get("rationale")
    if not isinstance(rationale, str) or not rationale.strip():
        return False, None, None

    return True, category, rationale.strip()


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
    total_cost = sum(r["cost_usd"] for r in records if r["cost_usd"] is not None)

    print()
    print("=== FINAL SUMMARY ===")
    print(f"Targets: {len(records)}/{len(targets)} recorded in {OUTPUT_PATH.name}")
    print(f"Judged: {len(judged)}; skipped: {len(skipped)}")
    for reason, count in sorted(Counter(r["skipped_reason"] for r in skipped).items()):
        print(f"  skipped ({reason}): {count}")
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
    max_seconds = float(sys.argv[1]) if len(sys.argv) > 1 else None
    started = time.monotonic()

    with open(TARGETS_PATH, "r") as f:
        targets: List[Dict[str, Any]] = json.load(f)

    moments_by_id = _load_moments_by_id(MOMENTS_PATH)

    # Pre-flight (before any money is spent): every judgeable target must have
    # a moments.jsonl record supplying description + writing_side. Fail loud
    # here rather than mid-run.
    missing = [
        t["moment_id"]
        for t in targets
        if t["transcript_exists"] and t["moment_id"] not in moments_by_id
    ]
    if missing:
        raise KeyError(f"targets missing from {MOMENTS_PATH}: {missing}")

    done_ids = set(_load_done_ids(OUTPUT_PATH))
    n_judgeable = sum(1 for t in targets if t["transcript_exists"])

    # ONE shared cfg_dir (real keychain-seeded creds) reused across all calls.
    cfg_dir = tempfile.mkdtemp(prefix="rejudge-failure-category-cfg-")
    _build_cfg_dir(cfg_dir)

    # Cache split_into_windows per transcript_path: several targets share a
    # transcript, and it is a deterministic pure read.
    windows_cache: Dict[str, List[Dict[str, Any]]] = {}

    invocation_spend = 0.0
    judgeable_seen = 0

    try:
        with open(OUTPUT_PATH, "a") as out_f:

            def emit(record: Dict[str, Any]) -> None:
                out_f.write(json.dumps(record) + "\n")
                out_f.flush()
                done_ids.add(record["moment_id"])

            for target in targets:
                moment_id = target["moment_id"]
                moment_type = target["moment_type"]
                skeleton = {
                    "moment_id": moment_id,
                    "moment_type": moment_type,
                    "new_failure_category": None,
                    "rationale": None,
                    "skipped_reason": None,
                    "cost_usd": None,
                }

                if not target["transcript_exists"]:
                    judgeable = False
                else:
                    judgeable = True
                    judgeable_seen += 1

                if moment_id in done_ids:
                    continue  # resume: already recorded by a prior invocation

                if max_seconds is not None and time.monotonic() - started > max_seconds:
                    print(f"time budget ({max_seconds:.0f}s) reached; stopping between targets", flush=True)
                    break

                if not judgeable:
                    emit({**skeleton, "skipped_reason": SKIP_RETENTION})
                    print(f"skip (retention-deleted): {moment_id}", flush=True)
                    continue

                print(f"[{judgeable_seen}/{n_judgeable}] judging {moment_id} ...", flush=True)

                transcript_path = target["transcript_path"]
                location = target["location"]
                moment = moments_by_id[moment_id]

                if transcript_path not in windows_cache:
                    windows_cache[transcript_path] = split_into_windows(transcript_path)
                window = _find_covering_window(windows_cache[transcript_path], location)
                if window is None:
                    emit({**skeleton, "skipped_reason": SKIP_NO_WINDOW})
                    print(f"  skipped: {SKIP_NO_WINDOW} (location {location})", flush=True)
                    continue

                prompt = _build_prompt(
                    window,
                    location,
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

                    ok, category, rationale = _parse_response(response.get("result", ""))
                    if ok:
                        outcome = {
                            **skeleton,
                            "new_failure_category": category,
                            "rationale": rationale,
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
                        f"  -> {outcome['new_failure_category']!r} (${target_cost:.4f}; "
                        f"invocation spend so far ${invocation_spend:.4f})",
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
