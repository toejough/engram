#!/usr/bin/env python3
import json
import os

# Mapping from s1 record name to fixture name
FIXTURE_MAP = {
    "baseline2_sonnet5_bisect-before-fix.s1.jsonl": "bisect-before-fix",
    "baseline2_sonnet5_history-rewrite.s1.jsonl": "history-rewrite",
    "baseline2_sonnet5_opsx-archive.s1.jsonl": "opsx-archive",
    "baseline2_sonnet5_route.s1.jsonl": "route",
    "baseline2_sonnet5_tdd-order.s1.jsonl": "tdd-order",
}

RESULTS_DIR = "/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/phase2/results"
FIXTURES_DIR = "/Users/joe/repos/personal/engram/.claude/worktrees/runbook-vs-skill/dev/eval/cumulative/runbook_vs_skill/phase2/fixtures"

for s1_name, fixture_name in FIXTURE_MAP.items():
    print(f"\n{'='*80}")
    print(f"TASK: {fixture_name}")
    print(f"{'='*80}")

    # Read fixture task.json to get removal list
    fixture_task_json = os.path.join(FIXTURES_DIR, fixture_name, "task.json")
    if not os.path.exists(fixture_task_json):
        print(f"  No task.json found at {fixture_task_json}")
        continue

    with open(fixture_task_json) as f:
        fixture = json.load(f)

    removal_basenames = fixture.get("removal_basenames", [])
    print(f"Removal basenames: {len(removal_basenames)} items")

    # Read the s1 record to get transcript path
    s1_path = os.path.join(RESULTS_DIR, s1_name)
    if not os.path.exists(s1_path):
        print(f"  No s1 record found at {s1_path}")
        continue

    with open(s1_path) as f:
        s1_record = json.load(f)

    transcript_path = s1_record.get("transcript_path")
    if not transcript_path:
        print(f"  No transcript_path in record")
        continue

    if not os.path.exists(transcript_path):
        print(f"  Transcript not found at {transcript_path}")
        continue

    print(f"Transcript: {os.path.basename(transcript_path)}")

    # Count occurrences of each removal basename in tool_result entries
    counts = {bn: 0 for bn in removal_basenames}

    with open(transcript_path) as f:
        for line in f:
            try:
                record = json.loads(line)
            except json.JSONDecodeError:
                continue

            if record.get("type") == "user" and "message" in record:
                msg = record["message"]
                if isinstance(msg, dict) and "content" in msg:
                    content = msg["content"]
                    if isinstance(content, list):
                        for item in content:
                            if isinstance(item, dict) and item.get("type") == "tool_result":
                                tool_content = str(item.get("content", ""))
                                for bn in removal_basenames:
                                    counts[bn] += tool_content.count(bn)

    # Report nonzero counts
    nonzero = {bn: c for bn, c in counts.items() if c > 0}
    if nonzero:
        print(f"Found {len(nonzero)} removal basenames with nonzero counts:")
        for bn in sorted(nonzero.keys(), key=lambda x: nonzero[x], reverse=True):
            print(f"  {bn}: {nonzero[bn]}")
    else:
        print("No removal basenames found in transcript tool_results")
