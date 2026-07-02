"""Extract real recorded `engram query` invocations from main session transcripts.

Scans ~/.claude/projects/-Users-joe-repos-personal-engram/*.jsonl (main sessions only;
subagent sub-directories are skipped). For each assistant Bash tool call that contains
`engram query`, extracts the ordered list of --phrase arguments. Dedupes identical phrase
sets (order-sensitive tuple); keeps only sets with ≥3 phrases; caps at the 25 most
recent distinct sets (recency = earliest timestamp in the session file).

Writes dev/eval/links/queries.json:
  [{id, phrases: [...], source_file, approx_date}]
"""
import json
import glob
import os
import re
import sys

TRANSCRIPT_GLOB = os.path.expanduser(
    "~/.claude/projects/-Users-joe-repos-personal-engram/*.jsonl"
)
OUT_PATH = os.path.join(os.path.dirname(__file__), "queries.json")
MIN_PHRASES = 3
MAX_SETS = 25

# Matches --phrase "..." (double-quoted) or --phrase '...' (single-quoted)
PHRASE_DOUBLE = re.compile(r'--phrase\s+"([^"]+)"')
PHRASE_SINGLE = re.compile(r"--phrase\s+'([^']+)'")


def extract_phrases(cmd: str) -> list[str]:
    """Return ordered list of --phrase values from a bash command string."""
    phrases = PHRASE_DOUBLE.findall(cmd)
    if not phrases:
        phrases = PHRASE_SINGLE.findall(cmd)
    return phrases


def session_first_timestamp(path: str) -> str:
    """Return the earliest ISO timestamp found in the JSONL file (or empty string)."""
    with open(path) as fh:
        for raw in fh:
            try:
                obj = json.loads(raw)
                ts = obj.get("timestamp", "")
                if ts:
                    return ts
            except (json.JSONDecodeError, AttributeError):
                pass
    return ""


def main() -> None:
    files = sorted(glob.glob(TRANSCRIPT_GLOB))
    if not files:
        sys.exit(f"ERROR: no JSONL files found at {TRANSCRIPT_GLOB}")

    print(f"Scanning {len(files)} session files …", flush=True)

    # Collect: list of (approx_date, source_file, phrases_tuple) in file-order
    # (glob sorts alphabetically by UUID → not temporal; we sort by first-timestamp)
    raw_hits: list[tuple[str, str, tuple[str, ...]]] = []

    for fpath in files:
        first_ts = session_first_timestamp(fpath)
        with open(fpath) as fh:
            for raw in fh:
                try:
                    obj = json.loads(raw)
                except json.JSONDecodeError:
                    continue
                if obj.get("type") != "assistant":
                    continue
                msg = obj.get("message", {})
                content = msg.get("content", [])
                for block in content:
                    if not isinstance(block, dict):
                        continue
                    if block.get("type") != "tool_use" or block.get("name") != "Bash":
                        continue
                    cmd = block.get("input", {}).get("command", "")
                    if "engram query" not in cmd:
                        continue
                    phrases = extract_phrases(cmd)
                    if len(phrases) >= MIN_PHRASES:
                        raw_hits.append((first_ts, fpath, tuple(phrases)))

    print(f"Raw hits (≥{MIN_PHRASES} phrases): {len(raw_hits)}", flush=True)

    # Sort by timestamp (ascending) so that capping at MAX_SETS takes the MOST RECENT
    raw_hits.sort(key=lambda x: x[0])

    # Dedupe: keep the LAST occurrence of each distinct phrase tuple
    seen: dict[tuple[str, ...], tuple[str, str]] = {}  # phrases_tuple → (ts, fpath)
    for ts, fpath, phrases in raw_hits:
        seen[phrases] = (ts, fpath)

    # Convert to list sorted descending (most recent first), then take first MAX_SETS
    deduped = sorted(seen.items(), key=lambda kv: kv[1][0], reverse=True)
    selected = deduped[:MAX_SETS]

    print(f"Distinct sets: {len(seen)}  →  capped at {len(selected)}", flush=True)

    records = []
    for idx, (phrases_tuple, (ts, fpath)) in enumerate(selected):
        approx_date = ts[:10] if ts else "unknown"
        records.append({
            "id": f"Q{idx:02d}",
            "phrases": list(phrases_tuple),
            "source_file": fpath,
            "approx_date": approx_date,
        })

    with open(OUT_PATH, "w") as fh:
        json.dump(records, fh, indent=2)

    print(f"Wrote {len(records)} query sets → {OUT_PATH}", flush=True)

    if len(records) < 10:
        print(
            f"\nWARNING: only {len(records)} usable query sets — below the 10-set gate "
            "threshold. This gates the miss population (Gate S1).",
            flush=True,
        )


if __name__ == "__main__":
    main()
