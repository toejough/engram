#!/usr/bin/env python3
"""
Re-score 18 previously-identified fork-role moments with real, billed
claude -p sonnet calls, now that judge_moment() resolves real
inherited-memory context for fork-role moments before judging "was it
followed" (the D-E gap fix in audit_moments.py).

Each of the 18 moment_ids below was already identified by the full audit
run in results/moments.jsonl -- this script does NOT re-detect moments or
re-derive moment_type/description; it reads those two fields straight from
the existing record and re-runs ONLY the judgment step (judge_moment) for
real, so the fork-context fix's effect on finding_side.followed and
failure_category can be measured against the original (pre-fix) values
already sitting in moments.jsonl.

Writes one JSON line per successfully-judged moment to
results/rescore-fork-followed.jsonl. Never reads moments.jsonl for
anything but the two source fields (moment_type/description) it is
explicitly allowed to reuse, and never writes to moments.jsonl,
scorecard-main.json, or any other existing file under results/.

This makes ~18 real `claude -p --model sonnet` calls -- real money, no stub.

Usage: python3 rescore_fork_followed.py
"""

import json
import os
import shutil
import sys
import tempfile
from pathlib import Path
from typing import Any, Dict, List, Optional

_SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _SCRIPT_DIR)

import audit_moments  # noqa: E402  (path insert must precede this import)
from audit_moments import _build_cfg_dir, judge_moment, split_into_windows  # noqa: E402
from extract import extract_transcript  # noqa: E402

MOMENTS_PATH = Path(_SCRIPT_DIR) / "results" / "moments.jsonl"
OUTPUT_PATH = Path(_SCRIPT_DIR) / "results" / "rescore-fork-followed.jsonl"

# The 18 already-identified fork-role moments to re-judge for real. moment_type
# and description are NOT hardcoded here -- they are read from the existing
# moments.jsonl record for each moment_id (task requirement: don't re-invent them).
TARGET_MOMENTS: List[Dict[str, Any]] = [
    {
        "moment_id": "agent-a5e3cc9e58c463d77.jsonl#5",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/a19cc352-75cc-4a0d-9dcc-a5504357df38/subagents/agent-a5e3cc9e58c463d77.jsonl",
        "location": 5,
    },
    {
        "moment_id": "agent-ad3f5d4aa55408741.jsonl#32",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-ad3f5d4aa55408741.jsonl",
        "location": 32,
    },
    {
        "moment_id": "agent-a6493eb381d2089cc.jsonl#18",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/d6e0c7ea-0f20-4554-ae54-0d4f5c175473/subagents/agent-a6493eb381d2089cc.jsonl",
        "location": 18,
    },
    {
        "moment_id": "agent-a6493eb381d2089cc.jsonl#22",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/d6e0c7ea-0f20-4554-ae54-0d4f5c175473/subagents/agent-a6493eb381d2089cc.jsonl",
        "location": 22,
    },
    {
        "moment_id": "agent-a6493eb381d2089cc.jsonl#24",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/d6e0c7ea-0f20-4554-ae54-0d4f5c175473/subagents/agent-a6493eb381d2089cc.jsonl",
        "location": 24,
    },
    {
        "moment_id": "agent-a6493eb381d2089cc.jsonl#26",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/d6e0c7ea-0f20-4554-ae54-0d4f5c175473/subagents/agent-a6493eb381d2089cc.jsonl",
        "location": 26,
    },
    {
        "moment_id": "agent-a6493eb381d2089cc.jsonl#28",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/d6e0c7ea-0f20-4554-ae54-0d4f5c175473/subagents/agent-a6493eb381d2089cc.jsonl",
        "location": 28,
    },
    {
        "moment_id": "agent-a8ce598d27fa736b9.jsonl#7",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a8ce598d27fa736b9.jsonl",
        "location": 7,
    },
    {
        "moment_id": "agent-a8ce598d27fa736b9.jsonl#9",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a8ce598d27fa736b9.jsonl",
        "location": 9,
    },
    {
        "moment_id": "agent-a8ce598d27fa736b9.jsonl#12",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a8ce598d27fa736b9.jsonl",
        "location": 12,
    },
    {
        "moment_id": "agent-a8ce598d27fa736b9.jsonl#23",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a8ce598d27fa736b9.jsonl",
        "location": 23,
    },
    {
        "moment_id": "agent-a8ce598d27fa736b9.jsonl#24",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a8ce598d27fa736b9.jsonl",
        "location": 24,
    },
    {
        "moment_id": "agent-a81b80335e1cdd38e.jsonl#6",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a81b80335e1cdd38e.jsonl",
        "location": 6,
    },
    {
        "moment_id": "agent-a81b80335e1cdd38e.jsonl#9",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a81b80335e1cdd38e.jsonl",
        "location": 9,
    },
    {
        "moment_id": "agent-a81b80335e1cdd38e.jsonl#13",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a81b80335e1cdd38e.jsonl",
        "location": 13,
    },
    {
        "moment_id": "agent-a81b80335e1cdd38e.jsonl#14",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a81b80335e1cdd38e.jsonl",
        "location": 14,
    },
    {
        "moment_id": "agent-a81b80335e1cdd38e.jsonl#15",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a81b80335e1cdd38e.jsonl",
        "location": 15,
    },
    {
        "moment_id": "agent-a81b80335e1cdd38e.jsonl#17",
        "transcript_path": "/Users/joe/.claude/projects/-Users-joe-repos-personal-phone-llm/57207951-9497-464c-9230-4c7f263390cd/subagents/agent-a81b80335e1cdd38e.jsonl",
        "location": 17,
    },
]


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


def main() -> None:
    moments_by_id = _load_moments_by_id(MOMENTS_PATH)

    # Shared, persistent cfg_dir (real keychain-seeded creds) reused across all 18
    # calls -- same pattern as audit_transcript()'s cfg_dir handling in this file.
    cfg_dir = tempfile.mkdtemp(prefix="rescore-fork-followed-cfg-")
    _build_cfg_dir(cfg_dir)

    # Wrap audit_moments._run_claude_p to capture the real total_cost_usd signal
    # each call reports (straight from claude -p's own --output-format json
    # response) without changing what call happens: this is a transparent
    # passthrough to the original implementation, not a stub -- stub=None is
    # still what judge_moment receives and forwards, so every call remains a
    # real, unstubbed claude -p invocation. judge_moment's own return value
    # (moment_record) carries no cost field, so this is the only way to report
    # a real, non-fabricated total cost per the task's own instructions.
    cost_tracker = {"total_usd": 0.0, "n_calls": 0}
    original_run_claude_p = audit_moments._run_claude_p

    def _cost_tracking_run_claude_p(*args, **kwargs):
        result = original_run_claude_p(*args, **kwargs)
        cost_tracker["total_usd"] += result.get("total_cost_usd", 0) or 0
        cost_tracker["n_calls"] += 1
        return result

    audit_moments._run_claude_p = _cost_tracking_run_claude_p

    # Cache extract_transcript/split_into_windows per transcript_path: several
    # target moments share the same transcript file, and both are deterministic,
    # pure-read functions over that file -- no reason to re-parse it per moment.
    extracted_cache: Dict[str, Dict[str, Any]] = {}
    windows_cache: Dict[str, List[Dict[str, Any]]] = {}

    n_total = len(TARGET_MOMENTS)
    n_succeeded = 0
    n_failed = 0
    failed_ids: List[str] = []

    try:
        with open(OUTPUT_PATH, "w") as out_f:
            for i, target in enumerate(TARGET_MOMENTS, start=1):
                moment_id = target["moment_id"]
                transcript_path = target["transcript_path"]
                location = target["location"]

                print(f"[{i}/{n_total}] judging {moment_id} ...", flush=True)

                try:
                    existing = moments_by_id.get(moment_id)
                    if existing is None:
                        raise KeyError(
                            f"{moment_id} not found in {MOMENTS_PATH} -- cannot reuse "
                            "its moment_type/description"
                        )

                    if transcript_path not in extracted_cache:
                        extracted_cache[transcript_path] = extract_transcript(transcript_path)
                    extracted_events = extracted_cache[transcript_path]

                    if transcript_path not in windows_cache:
                        windows_cache[transcript_path] = split_into_windows(transcript_path)
                    windows = windows_cache[transcript_path]

                    window = _find_covering_window(windows, location)
                    if window is None:
                        raise ValueError(
                            f"no window covers location {location} in {transcript_path} "
                            f"(windows span: "
                            f"{[(w['location_start'], w['location_end']) for w in windows]})"
                        )

                    candidate = {
                        "location": location,
                        "moment_type": existing["moment_type"],
                        "description": existing["description"],
                    }

                    result = judge_moment(
                        candidate,
                        window,
                        extracted_events,
                        model="sonnet",
                        stub=None,
                        cfg_dir=cfg_dir,
                        parent_transcript=None,
                    )

                    out_record = {
                        "moment_id": moment_id,
                        "new_followed": result["finding_side"]["followed"],
                        "new_failure_category": result["failure_category"],
                        "full_judgment": result,
                    }
                    out_f.write(json.dumps(out_record) + "\n")
                    out_f.flush()

                    n_succeeded += 1
                    print(
                        f"  moment {i}/{n_total} done "
                        f"(followed={out_record['new_followed']!r}); "
                        f"cost so far: ${cost_tracker['total_usd']:.4f} "
                        f"over {cost_tracker['n_calls']} real claude -p call(s)",
                        flush=True,
                    )

                except Exception as e:
                    n_failed += 1
                    failed_ids.append(moment_id)
                    print(f"  moment {i}/{n_total} FAILED: {moment_id}: {e}", file=sys.stderr, flush=True)
                    continue
    finally:
        audit_moments._run_claude_p = original_run_claude_p
        shutil.rmtree(cfg_dir, ignore_errors=True)

    print()
    print("=== FINAL SUMMARY ===")
    print(f"Total target moments: {n_total}")
    print(f"Succeeded: {n_succeeded}")
    print(f"Failed: {n_failed}" + (f" ({', '.join(failed_ids)})" if failed_ids else ""))
    print(
        f"Total real $ cost: ${cost_tracker['total_usd']:.4f} "
        f"(sum of total_cost_usd across {cost_tracker['n_calls']} real claude -p "
        "sonnet calls, captured by wrapping audit_moments._run_claude_p -- "
        "judge_moment's own return value carries no cost field, so this is the "
        "real internal cost signal claude -p's --output-format json response "
        "reports for each call, not a fabricated or estimated number)"
    )
    print(f"Output written to: {OUTPUT_PATH} ({n_succeeded} line(s))")


if __name__ == "__main__":
    main()
