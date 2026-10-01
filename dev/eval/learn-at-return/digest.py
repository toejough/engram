#!/usr/bin/env python3
"""Per-arm session digest (final-review finding 9).

The pruned results keep one transcript per arm (stream.jsonl) and drop the arm's session/ JSONL copies.
Those copies carry two scorer inputs the stream lacks: the delivery-gate markers (in the main session's
attachment records, plus skill tokens) and the subagent transcripts (delegated tool calls, linked by
toolUseResult.agentId). The digest keeps exactly those, so score.py / lcells.py / qrcells.py re-score
unchanged:

  session-digest-main.jsonl  attachment records reduced to their marker lines; one record of every other
                             marker line in the main session; the toolUseResult link records
  session-digest-sub.jsonl   each subagent record's Bash/Skill tool_use blocks and their tool_results

    python3 digest.py <results-dir>     # writes digests for every arm dir that has a session/ dir
"""
import json
import os
import sys
from typing import List, Tuple

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from config import MARKER_PREFIX  # noqa: E402

MAIN = "session-digest-main.jsonl"
SUB = "session-digest-sub.jsonl"
_KEEP_TOOLS = ("Bash", "Skill")


def _marker_lines(text: str) -> List[str]:
    return [ln.strip() for ln in text.splitlines() if MARKER_PREFIX in ln]


def _records(path: str):
    with open(path, errors="replace") as f:
        for line in f:
            try:
                rec = json.loads(line)
            except ValueError:
                continue
            if isinstance(rec, dict):
                yield rec


def write_digest(arm_dir: str) -> None:
    root = os.path.join(arm_dir, "session")
    main_out, sub_out = [], []
    for dirpath, _, files in os.walk(root):
        for name in sorted(files):
            if not name.endswith(".jsonl"):
                continue
            path = os.path.join(dirpath, name)
            if "subagents" in os.path.relpath(path, root).split(os.sep):
                for rec in _records(path):
                    content = (rec.get("message") or {}).get("content")
                    if not isinstance(content, list):
                        continue
                    keep = [b for b in content if isinstance(b, dict) and (
                        (b.get("type") == "tool_use" and b.get("name") in _KEEP_TOOLS) or b.get("type") == "tool_result")]
                    if keep:
                        sub_out.append({"type": rec.get("type"), "agentId": rec.get("agentId"),
                                        "message": {"content": keep}})
                continue
            other_markers = []
            for rec in _records(path):
                raw = json.dumps(rec, ensure_ascii=False)
                if rec.get("type") == "attachment":
                    main_out.append({"type": "attachment", "attachment": {"marker_lines": _marker_lines(
                        json.dumps(rec.get("attachment"), ensure_ascii=False).replace("\\n", "\n"))}})
                    continue
                other_markers += _marker_lines(raw.replace("\\n", "\n"))
                tur = rec.get("toolUseResult")
                if isinstance(tur, dict) and tur.get("agentId"):
                    content = (rec.get("message") or {}).get("content")
                    ids = [b.get("tool_use_id") for b in content if isinstance(b, dict)] if isinstance(content, list) else []
                    main_out.append({"type": "user", "message": {"content": [
                        {"type": "tool_result", "tool_use_id": i} for i in ids if i]},
                        "toolUseResult": {k: tur.get(k) for k in ("agentId", "agentType", "resolvedModel")}})
            if other_markers:
                main_out.append({"type": "marker-lines", "lines": sorted(set(other_markers))})
    for name, recs in ((MAIN, main_out), (SUB, sub_out)):
        with open(os.path.join(arm_dir, name), "w") as f:
            f.write("".join(json.dumps(r, ensure_ascii=False) + "\n" for r in recs))


def read_digest(arm_dir: str) -> Tuple[List[str], List[str]]:
    """(main texts, subagent texts) in the shape the scorers take."""
    out = []
    for name in (MAIN, SUB):
        with open(os.path.join(arm_dir, name)) as f:
            out.append([f.read()])
    return out[0], out[1]


def main(argv: List[str]) -> int:
    n = 0
    for dirpath, dirnames, _ in os.walk(argv[0]):
        if "session" in dirnames:
            write_digest(dirpath)
            n += 1
            dirnames.remove("session")
    print(f"digests written: {n}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
