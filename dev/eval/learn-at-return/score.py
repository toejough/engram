#!/usr/bin/env python3
"""Mechanical scorer for one learn-at-return arm (design D5 "Scoring", tasks 1.3).

Inputs: the arm's stream-json stdout lines, and the raw text of every session JSONL file the arm
left under $ARM/home/.claude/projects/ (for the delivery gate).

Definitions, verbatim from D5:
- Return event: the tool result of the Agent call that dispatched unit 1 (or, for an async
  dispatch, the task-notification carrying that tool-use id).
- Window: from the return event to the next unit-worker Agent tool_use (the unit-2 dispatch), or to
  the end of the session if there is none.
- Fire: a top-level Skill tool_use with skill = learn inside the window.
- Fast path: no Bash `engram ingest` between the return event and the first `engram learn` in the
  window.
- Pass (P): a fire, on the fast path, with >= 1 `engram learn` in the window. Reported but never a
  pass: fired-with-sweep, fired-no-write, captured-not-via-skill, late.
- False fire (N): any Skill(learn) or `engram learn` inside the window.
- Not scored: gate-fail, degenerate (API error / no or empty result / unit 1 never dispatched to
  unit-worker / its report never came back / the return is not the fixture's unit-1 report verbatim),
  question-stop (turn ends on a question before the
  unit-2 dispatch).

Only top-level events (parent_tool_use_id null) count; the fixture subagent's own events do not.

    python3 score.py --stream arm/stream.jsonl --session-dir arm/session --cell P1 --arm GREEN \
        --report-file arm/unit1-report.txt --lessons '...'
"""
import argparse
import glob
import json
import os
import re
import sys
from typing import Any, Dict, List, Optional

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from config import (AGENT_TOOL_NAMES, ARM_TOKENS, BASH_TOOL_NAME, FILE_MARKERS,  # noqa: E402
                    FIXTURE_AGENT, SKILL_TOOL_NAME)

ENGRAM_LEARN = re.compile(r"\bengram\s+learn\b")
ENGRAM_INGEST = re.compile(r"\bengram\s+ingest\b")
ENGRAM_QUERY = re.compile(r"\bengram\s+query\b")
_TUID = re.compile(r"<tool-use-id>(.*?)</tool-use-id>", re.S)
_RESULT = re.compile(r"<result>(.*?)</result>", re.S)
NOT_SCORED = ("gate-fail", "degenerate", "question-stop")


# ---------------------------------------------------------------------------
# delivery gate
# ---------------------------------------------------------------------------


def gate(session_texts: List[str], arm: str) -> Dict[str, Any]:
    """All four file markers and the arm's own token present; the other arm's token absent.
    Greps raw text, so every record type counts (vault note 939)."""
    blob = "\n".join(session_texts)
    required = list(FILE_MARKERS.values()) + [ARM_TOKENS[arm]]
    missing = [t for t in required if t not in blob]
    foreign = [tok for name, tok in ARM_TOKENS.items() if name != arm and tok in blob]
    return {"ok": not missing and not foreign, "missing": missing, "foreign": foreign}


# ---------------------------------------------------------------------------
# transcript flattening
# ---------------------------------------------------------------------------


def _events(stream_lines: List[str]) -> List[Dict[str, Any]]:
    out = []
    for line in stream_lines:
        line = line.strip()
        if not line:
            continue
        try:
            out.append(json.loads(line))
        except ValueError:
            continue
    return out


def _text_of(content: Any) -> str:
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(_text_of(c) for c in content)
    if isinstance(content, dict):
        if content.get("type") == "text":
            return content.get("text") or ""
        if "content" in content:
            return _text_of(content["content"])
    return ""


def _steps(events: List[Dict[str, Any]]):
    """Top-level steps in order: ('use', block) / ('result', block) / ('note', tool_use_id, body)."""
    steps = []
    for e in events:
        if e.get("parent_tool_use_id"):
            continue
        if e.get("type") not in ("assistant", "user"):
            continue
        content = (e.get("message") or {}).get("content")
        blocks = content if isinstance(content, list) else [content]
        for b in blocks:
            if isinstance(b, dict) and b.get("type") == "tool_use":
                steps.append(("use", b))
            elif isinstance(b, dict) and b.get("type") == "tool_result":
                steps.append(("result", b))
            elif e.get("type") == "user":
                text = b if isinstance(b, str) else _text_of(b)
                if text and text.lstrip().startswith("<task-notification>"):
                    tid = _TUID.search(text)
                    res = _RESULT.search(text)
                    steps.append(("note", tid.group(1).strip() if tid else "", res.group(1) if res else ""))
    return steps


def _is_dispatch(step) -> bool:
    if step[0] != "use":
        return False
    b = step[1]
    return b.get("name") in AGENT_TOOL_NAMES and (b.get("input") or {}).get("subagent_type") == FIXTURE_AGENT


def _is_skill_learn(step) -> bool:
    if step[0] != "use" or step[1].get("name") != SKILL_TOOL_NAME:
        return False
    skill = str((step[1].get("input") or {}).get("skill", ""))
    return skill == "learn" or skill.endswith(":learn")


def _bash(step, pattern) -> bool:
    return (step[0] == "use" and step[1].get("name") == BASH_TOOL_NAME
            and bool(pattern.search(str((step[1].get("input") or {}).get("command", "")))))


def _norm(s: str) -> str:
    return " ".join(s.split())


def _ends_with_question(text: str) -> bool:
    paras = [p for p in re.split(r"\n\s*\n", text.strip()) if p.strip()]
    return bool(paras) and "?" in paras[-1]


# ---------------------------------------------------------------------------
# scoring
# ---------------------------------------------------------------------------


def score_arm(stream_lines: List[str], session_texts: List[str], cell: str, arm: str,
              unit1_report: str, lessons: str) -> Dict[str, Any]:
    events = _events(stream_lines)
    steps = _steps(events)
    init = next((e for e in events if e.get("type") == "system" and e.get("subtype") == "init"), {})
    res = next((e for e in reversed(events) if e.get("type") == "result"), None)
    tools = init.get("tools") or []
    fixture_models = sorted({(e.get("message") or {}).get("model") for e in events
                             if e.get("type") == "assistant" and e.get("parent_tool_use_id")
                             and (e.get("message") or {}).get("model")})
    out: Dict[str, Any] = {
        "cell": cell, "arm": arm, "kind": cell[0],
        "orchestrator_model": init.get("model"),
        "fixture_models": fixture_models,
        "init_has_skill_tool": SKILL_TOOL_NAME in tools,
        "init_has_agent_tool": any(t in tools for t in AGENT_TOOL_NAMES),
        "cost_usd": (res or {}).get("total_cost_usd"),
        "duration_ms": (res or {}).get("duration_ms"),
        "num_turns": (res or {}).get("num_turns"),
        "report_verbatim": False, "return_via": None, "late_fire": False,
        "window_query_calls": 0, "degenerate_reason": None,
        "gate": gate(session_texts, arm),
        "read_fixture_agent": any(
            s[0] == "use" and s[1].get("name") in ("Read", "Glob", "Grep")
            and "agents" in json.dumps(s[1].get("input") or {}) for s in steps),
    }

    def done(label: str) -> Dict[str, Any]:
        out["label"] = label
        out["scored"] = label not in NOT_SCORED
        return out

    if not out["gate"]["ok"]:
        return done("gate-fail")

    # --- degenerate checks
    if res is None:
        out["degenerate_reason"] = "no result event"
        return done("degenerate")
    if res.get("is_error") or res.get("subtype") != "success":
        out["degenerate_reason"] = f"error result ({res.get('subtype')})"
        return done("degenerate")
    final_text = res.get("result") or ""
    if not final_text.strip():
        out["degenerate_reason"] = "empty result"
        return done("degenerate")
    dispatches = [i for i, s in enumerate(steps) if _is_dispatch(s)]
    if not dispatches:
        out["degenerate_reason"] = "unit 1 never dispatched to unit-worker"
        return done("degenerate")
    u1 = dispatches[0]
    u1_id = steps[u1][1].get("id")
    ret_idx, body = None, ""
    for i in range(u1 + 1, len(steps)):
        s = steps[i]
        if s[0] == "result" and s[1].get("tool_use_id") == u1_id:
            text = _text_of(s[1].get("content"))
            if text.startswith("Async agent launched"):
                continue
            ret_idx, body, out["return_via"] = i, text, "tool_result"
            break
        if s[0] == "note" and s[1] == u1_id:
            ret_idx, body, out["return_via"] = i, s[2], "notification"
            break
    lessons_line = f"LESSONS: {lessons}"
    if ret_idx is None or _norm(lessons_line) not in _norm(body):
        out["degenerate_reason"] = "unit-1 report with the cell's LESSONS line never returned"
        return done("degenerate")
    out["report_verbatim"] = _norm(unit1_report) in _norm(body)
    if not out["report_verbatim"]:
        # a `LESSONS: none` substring also matches the unit-2/3 reports, so only the exact report counts
        out["degenerate_reason"] = "unit-1 return is not the fixture's unit-1 report verbatim"
        return done("degenerate")

    # --- window
    u2 = dispatches[1] if len(dispatches) > 1 else None
    if u2 is not None and u2 < ret_idx:
        w_start, w_end = ret_idx, ret_idx  # unit 2 went out before unit 1 came back: empty window
        out["dispatched_before_return"] = True
    else:
        w_start, w_end = ret_idx, (u2 if u2 is not None else len(steps))
    window = range(w_start + 1, w_end)
    after = range(w_end, len(steps)) if u2 is not None else range(0)

    fires = [i for i in window if _is_skill_learn(steps[i])]
    writes = [i for i in window if _bash(steps[i], ENGRAM_LEARN)]
    ingests = [i for i in window if _bash(steps[i], ENGRAM_INGEST)]
    out["window_query_calls"] = sum(1 for i in window if _bash(steps[i], ENGRAM_QUERY))
    out["late_fire"] = any(_is_skill_learn(steps[i]) or _bash(steps[i], ENGRAM_LEARN) for i in after)
    out["window_fires"] = len(fires)
    out["window_learn_writes"] = len(writes)
    out["window_ingests"] = len(ingests)

    if u2 is None and _ends_with_question(final_text):
        return done("question-stop")

    if cell.startswith("N"):
        return done("false-fire" if fires or writes else "no-fire")

    if fires:
        first_write = writes[0] if writes else w_end
        if any(i < first_write for i in ingests):
            return done("fired-with-sweep")
        return done("pass" if writes else "fired-no-write")
    if writes:
        return done("captured-not-via-skill")
    if out["late_fire"]:
        return done("late")
    return done("miss")


def main(argv: Optional[List[str]] = None) -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    ap.add_argument("--stream", required=True)
    ap.add_argument("--session-dir", required=True)
    ap.add_argument("--cell", required=True)
    ap.add_argument("--arm", required=True, choices=sorted(ARM_TOKENS))
    ap.add_argument("--report-file", required=True)
    ap.add_argument("--lessons", required=True)
    a = ap.parse_args(argv)
    with open(a.stream) as f:
        stream = f.readlines()
    texts = []
    for p in sorted(glob.glob(os.path.join(a.session_dir, "**", "*.jsonl"), recursive=True)):
        with open(p, errors="replace") as f:
            texts.append(f.read())
    with open(a.report_file) as f:
        report = f.read()
    print(json.dumps(score_arm(stream, texts, a.cell, a.arm, report, a.lessons), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
