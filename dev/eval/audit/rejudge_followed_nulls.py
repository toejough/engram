#!/usr/bin/env python3
"""
Re-judge finding_side.followed for the 19 moments in results/moments.jsonl
where surfaced==true but followed is null and moment_type != 'dispatch',
using real, billed claude -p sonnet calls.

Why: the F5 followed field was left null for these moments by earlier runs
-- for the 16 fork-role (search_ran=="injected") moments the judge was
never able to render a followed verdict against the actually-inherited
memory content, and for the 3 full_recall moments the verdict simply never
got asked. Unasked/unexplained nulls in a decision-relevant field get
re-collected for real (the 2026-09-03 lesson, vault note 900) -- and this
prompt closes the null loophole explicitly: the judge must answer one of
seven enum values, where "not_applicable" (the surfaced content does not
meaningfully bear on this moment) is the honest replacement for null.
(The 5 dispatch-type followed-nulls are NOT judged here: followed is
structurally not-applicable for dispatch moments by design, handled
mechanically elsewhere.)

Same conventions as rejudge_failure_category.py / rejudge_from_chunks.py.
One targeted single-question prompt per moment -- no other scorecard field
is re-derived. Targets are derived from moments.jsonl itself (read-only)
and split three ways:

1. 16 fork-role moments (search_ran=="injected"): context is the covering
   window of the fork's own transcript, plus the REAL inherited memory
   content resolved via audit_moments._resolve_fork_parent_context()
   (imported, not reimplemented). judged_from="fork-inherited".
2. 2 full_recall moments with surviving transcripts: context is the
   covering window; the surfaced memory content is what their own recall
   actually matched, read from results/transcript-events.jsonl (the
   recall_calls entry at-or-before the moment's location) with each
   matched note's CURRENT content read best-effort from the live vault.
   judged_from="transcript".
3. 1 full_recall moment whose transcript was retention-deleted
   (agent-ac7ec7f863667110d.jsonl#7): context is reconstructed from
   engram's chunk index (same reconstruction as rejudge_from_chunks.py,
   whose helpers are imported); the surfaced memory content comes from
   transcript-events.jsonl the same way. judged_from="chunks".

Per-target handling:
- prompt_too_long from claude -p: skipped (deterministic, never retried).
- malformed/invalid-enum/missing-rationale response: ONE retry, then
  recorded with skipped_reason="unparseable response" -- never invented.
- exhausted_retries from claude -p: the ENTIRE run halts immediately
  (raise ExhaustedRetriesError; the 2026-09-02 silent-auth-outage lesson:
  never log-and-continue past a possible sustained outage).

Output: one JSON line per target (all 19, including skips) appended to
results/rejudge-followed-nulls.jsonl:
  {"moment_id", "moment_type", "role", "new_followed", "rationale",
   "judged_from": "transcript"|"chunks"|"fork-inherited",
   "skipped_reason" (null if judged), "cost_usd"}
cost_usd is the real total_cost_usd claude -p reported for that target's
call(s) -- including a retried call's spend, and including the spend on an
ultimately-unparseable target -- or null when no call was made at all.

The script is RESUMABLE: on start it reads any moment_ids already present
in the output file and skips them (append mode). The final summary prints
only once all 19 targets are present.

Never writes to moments.jsonl or anything else under results/ besides
results/rejudge-followed-nulls.jsonl.

This makes ~19 real `claude -p --model sonnet` calls -- real money, no stub.

Usage: python3 rejudge_followed_nulls.py [max_seconds]
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
    PromptTooLongError,  # noqa: F401  (documented contract; prompt_too_long arrives as a response flag)
    _build_cfg_dir,
    _resolve_fork_parent_context,
    _run_claude_p,
    split_into_windows,
)
from rejudge_from_chunks import (  # noqa: E402
    _find_chunk_file,
    _reconstruct_from_chunks,
)

MOMENTS_PATH = Path(_SCRIPT_DIR) / "results" / "moments.jsonl"
EVENTS_PATH = Path(_SCRIPT_DIR) / "results" / "transcript-events.jsonl"
OUTPUT_PATH = Path(_SCRIPT_DIR) / "results" / "rejudge-followed-nulls.jsonl"
VAULT_DIR = Path.home() / ".local" / "share" / "engram" / "vault"

EXPECTED_TARGET_COUNT = 19
EXPECTED_FORK_COUNT = 16

SKIP_NO_WINDOW = "no window covers moment location"
SKIP_TOO_LONG = "prompt too long for claude -p"
SKIP_UNPARSEABLE = "unparseable response"

# The only values the judge may answer with. null is deliberately NOT in
# this set -- the whole point of this pass is closing the null loophole;
# "not_applicable" is the honest replacement. Anything else (after one
# retry) is recorded as unparseable -- never coerced or invented.
VALID_FOLLOWED = {
    "yes",
    "ignored",
    "partial",
    "skipped_step",
    "out_of_order",
    "contradicted",
    "not_applicable",
}

QUESTION = (
    "Was this surfaced memory actually followed at this moment? Answer with exactly one of: "
    '"yes" (the agent acted in line with the memory), "ignored", "partial", "skipped_step", '
    '"out_of_order", "contradicted", or "not_applicable" (the surfaced content does not '
    "meaningfully bear on this specific moment, so followed/not-followed is not a sensible "
    'question here). Do NOT answer null -- if you cannot render any of the first six verdicts, '
    '"not_applicable" with a rationale is the honest answer.'
)

RESPONSE_FORMAT = """Respond with STRICT JSON only -- no prose before or after:
{"followed": "yes" | "ignored" | "partial" | "skipped_step" | "out_of_order" | "contradicted" | "not_applicable", \
"rationale": "<one sentence>"}
The "rationale" field is REQUIRED: one sentence justifying the answer."""

NOTE_GONE = "[content unavailable -- this note is no longer in the live vault]"


def _load_targets() -> List[Dict[str, Any]]:
    """
    Derive the target population straight from moments.jsonl (read-only):
    surfaced==true AND followed==null AND moment_type != 'dispatch'.
    """
    targets: List[Dict[str, Any]] = []
    with open(MOMENTS_PATH, "r") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rec = json.loads(line)
            finding = rec.get("finding_side") or {}
            if (
                finding.get("surfaced") is True
                and finding.get("followed") is None
                and rec.get("moment_type") != "dispatch"
            ):
                targets.append(rec)
    return targets


def _load_recall_events_by_basename(path: Path) -> Dict[str, Dict[str, Any]]:
    """transcript-events.jsonl records keyed by transcript basename. Fail loud on collisions."""
    by_basename: Dict[str, Dict[str, Any]] = {}
    with open(path, "r") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            rec = json.loads(line)
            basename = os.path.basename(rec["transcript_path"])
            if basename in by_basename:
                raise ValueError(f"duplicate transcript basename in {path}: {basename}")
            by_basename[basename] = rec
    return by_basename


def _recall_content_for(events_record: Dict[str, Any], location: int) -> Tuple[str, List[str]]:
    """
    The memory content a full_recall moment's own recall actually surfaced:
    the recall_calls entry at-or-before the moment's location (the last such
    one), its matched_items' note paths, each note's CURRENT vault content
    read best-effort. Fail loud if no qualifying recall_call exists at all
    (the surfaced=true credit would then be unexplainable).

    Returns (formatted content block, list of matched note paths).
    """
    qualifying = [
        rc for rc in events_record.get("recall_calls", []) if rc.get("location", 0) <= location
    ]
    if not qualifying:
        raise ValueError(
            f"no recall_call at-or-before location {location} in "
            f"{events_record['transcript_path']} -- surfaced=true is unexplainable"
        )
    last_recall = max(qualifying, key=lambda rc: rc.get("location", 0))

    matched_paths: List[str] = []
    sections: List[str] = []
    for item in last_recall.get("matched_items", []):
        note_path = item.get("path")
        if not note_path:
            continue
        matched_paths.append(note_path)
        try:
            with open(VAULT_DIR / note_path, "r") as f:
                content = f.read()
        except (IOError, OSError):
            # Best-effort: the note may have been renamed/superseded/deleted
            # since this recall happened. Say so honestly rather than skip
            # silently -- the judge should know the item existed.
            content = NOTE_GONE
        sections.append(f"--- {note_path} ---\n{content}")

    if not matched_paths:
        raise ValueError(
            f"recall_call at location {last_recall.get('location')} in "
            f"{events_record['transcript_path']} matched no items -- surfaced=true is unexplainable"
        )
    return "\n\n".join(sections), matched_paths


def _fork_inherited_content(fork_context: Dict[str, Any]) -> str:
    """Format _resolve_fork_parent_context's recalled_notes as one content block."""
    sections = []
    for note in fork_context["recalled_notes"]:
        sections.append(f"--- {note['path']} ---\n{note['content']}")
    return "\n\n".join(sections)


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


def _build_window_prompt(
    window: Dict[str, Any],
    location: int,
    moment_type: str,
    description: str,
    memory_label: str,
    memory_content: str,
) -> str:
    """Prompt for a target with a surviving transcript (fork or full_recall)."""
    return f"""You are auditing one moment from a real agent-session transcript for a memory-loop audit.

WINDOW (lines {window['location_start']}-{window['location_end']}; each line is prefixed with its absolute line number):
{_numbered_window_text(window)}

{memory_label}:
{memory_content}

MOMENT UNDER JUDGMENT:
- location: line {location}
- moment_type: {moment_type}
- description: {description}

QUESTION: {QUESTION}

{RESPONSE_FORMAT}"""


def _build_chunk_prompt(
    chunk_text: str,
    location: int,
    moment_type: str,
    description: str,
    memory_label: str,
    memory_content: str,
) -> str:
    """Prompt for the retention-deleted target, over the chunk-reconstructed record."""
    return f"""You are auditing one moment from a real agent-session transcript for a memory-loop audit.

The original transcript file was deleted by session retention. The record below is reconstructed \
from engram's chunk index -- stripped conversational turns, tool-call detail reduced, the original \
line numbering is unavailable. It is the FULL surviving conversational record for this session, \
in turn order.

RECONSTRUCTED TRANSCRIPT (from engram's chunk index):
{chunk_text}

{memory_label}:
{memory_content}

MOMENT UNDER JUDGMENT:
- approximate original location: line {location} of the original transcript JSONL (that line \
numbering does not map onto the reconstruction above; use the description to find the moment)
- moment_type: {moment_type}
- description: {description}

QUESTION: {QUESTION}

{RESPONSE_FORMAT}"""


def _parse_response(result_text: str) -> Tuple[bool, Optional[str], Optional[str]]:
    """
    Parse and validate the model's response.

    Returns (ok, followed, rationale). ok=False means the response was
    malformed/invalid (bad JSON, out-of-enum value -- including null --
    or missing/empty rationale) and the caller should retry once, then skip.
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

    if not isinstance(parsed, dict) or "followed" not in parsed:
        return False, None, None

    followed = parsed["followed"]
    if followed not in VALID_FOLLOWED:
        return False, None, None

    rationale = parsed.get("rationale")
    if not isinstance(rationale, str) or not rationale.strip():
        return False, None, None

    return True, followed, rationale.strip()


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
    print("new_followed tallies (judged targets):")
    for value, count in sorted(Counter(r["new_followed"] for r in judged).items()):
        print(f"  {value}: {count}")
    print("new_followed by judged_from:")
    tally = Counter((r["judged_from"], str(r["new_followed"])) for r in judged)
    for (source, value), count in sorted(tally.items()):
        print(f"  {source:>15} -> {value}: {count}")
    print(
        f"Total real $ cost: ${total_cost:.4f} (sum of per-target cost_usd -- each the "
        "total_cost_usd claude -p's own --output-format json response reported for that "
        "target's real sonnet call(s), across ALL invocations of this script; not an estimate)"
    )


def _preflight(targets: List[Dict[str, Any]]) -> Dict[str, Dict[str, Any]]:
    """
    Verify -- BEFORE any money is spent -- that the target population is
    exactly the expected 19 (16 fork/injected + 3 full_recall, of which
    exactly 1 is retention-deleted), and that every target's memory content
    is actually resolvable (fail loud, never silently fall back). Returns
    per-moment_id prep: {"judged_from", "memory_label", "memory_content"}.
    """
    if len(targets) != EXPECTED_TARGET_COUNT:
        raise ValueError(
            f"expected exactly {EXPECTED_TARGET_COUNT} targets "
            f"(surfaced==true, followed==null, moment_type != 'dispatch'), "
            f"found {len(targets)} -- refusing to run against an unexpected population"
        )

    forks = [t for t in targets if t["finding_side"].get("search_ran") == "injected"]
    non_forks = [t for t in targets if t["finding_side"].get("search_ran") != "injected"]
    if len(forks) != EXPECTED_FORK_COUNT:
        raise ValueError(
            f"expected {EXPECTED_FORK_COUNT} fork (search_ran=='injected') targets, found {len(forks)}"
        )
    for t in non_forks:
        if t["finding_side"].get("search_ran") != "full_recall":
            raise ValueError(
                f"{t['moment_id']}: expected search_ran=='full_recall' for non-fork target, "
                f"got {t['finding_side'].get('search_ran')!r}"
            )
    deleted = [t for t in non_forks if not os.path.exists(t["transcript_path"])]
    if len(deleted) != 1:
        raise ValueError(
            f"expected exactly 1 retention-deleted non-fork target, found "
            f"{[t['moment_id'] for t in deleted]}"
        )
    missing_fork_transcripts = [t["moment_id"] for t in forks if not os.path.exists(t["transcript_path"])]
    if missing_fork_transcripts:
        raise ValueError(f"fork targets with missing transcripts: {missing_fork_transcripts}")

    events_by_basename = _load_recall_events_by_basename(EVENTS_PATH)
    prep: Dict[str, Dict[str, Any]] = {}

    # Fork targets: resolve the real inherited memory content (cached per
    # transcript -- resolution is a deterministic pure read).
    fork_context_cache: Dict[str, Dict[str, Any]] = {}
    for t in forks:
        transcript_path = t["transcript_path"]
        if transcript_path not in fork_context_cache:
            fork_context_cache[transcript_path] = _resolve_fork_parent_context(transcript_path)
        fork_context = fork_context_cache[transcript_path]
        if not fork_context["available"] or not fork_context["recalled_notes"]:
            raise ValueError(
                f"{t['moment_id']}: fork parent context unresolvable "
                f"(available={fork_context['available']}, reason={fork_context['reason']!r}) "
                "-- the inherited memory content is required to judge followed"
            )
        prep[t["moment_id"]] = {
            "judged_from": "fork-inherited",
            "memory_label": (
                "MEMORY CONTENT THE AGENT INHERITED (recalled in the parent conversation "
                "before the fork point; this fork inherited the parent's full prior conversation)"
            ),
            "memory_content": _fork_inherited_content(fork_context),
        }

    # full_recall targets: resolve what their own recall actually surfaced.
    for t in non_forks:
        basename = os.path.basename(t["transcript_path"])
        if basename not in events_by_basename:
            raise KeyError(f"{t['moment_id']}: no transcript-events.jsonl record for {basename}")
        content, matched_paths = _recall_content_for(events_by_basename[basename], t["location"])
        prep[t["moment_id"]] = {
            "judged_from": "chunks" if not os.path.exists(t["transcript_path"]) else "transcript",
            "memory_label": (
                "MEMORY CONTENT THIS AGENT'S OWN RECALL SURFACED (the notes its engram recall "
                "call matched, shown with their current vault content)"
            ),
            "memory_content": content,
        }
        print(
            f"preflight: {t['moment_id']} recall matched {len(matched_paths)} note(s); "
            f"judged_from={prep[t['moment_id']]['judged_from']}",
            flush=True,
        )

    # The retention-deleted target must have exactly one chunk file (fail loud).
    for t in deleted:
        _find_chunk_file(t["transcript_path"])

    return prep


def main() -> None:
    max_seconds = float(sys.argv[1]) if len(sys.argv) > 1 else None
    started = time.monotonic()

    targets = _load_targets()
    prep = _preflight(targets)

    done_ids = set(_load_done_ids(OUTPUT_PATH))

    # ONE shared cfg_dir (real keychain-seeded creds) reused across all calls.
    cfg_dir = tempfile.mkdtemp(prefix="rejudge-followed-nulls-cfg-")
    _build_cfg_dir(cfg_dir)

    # Cache windows / chunk reconstructions per transcript: several targets
    # share a transcript, and both are deterministic pure reads.
    windows_cache: Dict[str, List[Dict[str, Any]]] = {}
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
                target_prep = prep[moment_id]
                skeleton = {
                    "moment_id": moment_id,
                    "moment_type": target["moment_type"],
                    "role": target["role"],
                    "new_followed": None,
                    "rationale": None,
                    "judged_from": target_prep["judged_from"],
                    "skipped_reason": None,
                    "cost_usd": None,
                }

                if moment_id in done_ids:
                    continue  # resume: already recorded by a prior invocation

                if max_seconds is not None and time.monotonic() - started > max_seconds:
                    print(f"time budget ({max_seconds:.0f}s) reached; stopping between targets", flush=True)
                    break

                print(
                    f"[{index}/{len(targets)}] judging {moment_id} "
                    f"(judged_from={target_prep['judged_from']}) ...",
                    flush=True,
                )

                transcript_path = target["transcript_path"]
                location = target["location"]

                if target_prep["judged_from"] == "chunks":
                    if transcript_path not in reconstruction_cache:
                        reconstruction_cache[transcript_path] = _reconstruct_from_chunks(
                            _find_chunk_file(transcript_path)
                        )
                    prompt = _build_chunk_prompt(
                        reconstruction_cache[transcript_path],
                        location,
                        target["moment_type"],
                        target["description"],
                        target_prep["memory_label"],
                        target_prep["memory_content"],
                    )
                else:
                    if transcript_path not in windows_cache:
                        windows_cache[transcript_path] = split_into_windows(transcript_path)
                    window = _find_covering_window(windows_cache[transcript_path], location)
                    if window is None:
                        emit({**skeleton, "skipped_reason": SKIP_NO_WINDOW})
                        print(f"  skipped: {SKIP_NO_WINDOW} (location {location})", flush=True)
                        continue
                    prompt = _build_window_prompt(
                        window,
                        location,
                        target["moment_type"],
                        target["description"],
                        target_prep["memory_label"],
                        target_prep["memory_content"],
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

                    ok, followed, rationale = _parse_response(response.get("result", ""))
                    if ok:
                        outcome = {
                            **skeleton,
                            "new_followed": followed,
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
                        f"  -> {outcome['new_followed']!r} (${target_cost:.4f}; "
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
