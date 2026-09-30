#!/usr/bin/env python3
"""Mechanical scorer for one learn-at-return arm (design D5 "Scoring", tasks 1.3).

Inputs: the arm's stream-json stdout lines; the raw text of the MAIN session JSONL (the delivery gate
reads its attachment records only) and of the other session files (subagent transcripts, checked only
for the foreign arm token); and the note files left in $ARM/vault.

Definitions (D5, as tightened by the U1 review):
- Return event: the tool result of the Agent call that dispatched unit 1 (or, for an async dispatch,
  the task-notification carrying that tool-use id).
- Window: from the return event to the dispatch of UNIT 2 (a unit-worker dispatch whose description
  or prompt names unit 2 or 3). A re-dispatch of unit 1 does not end it. If unit 2 went out before
  unit 1 came back, the arm is `unscorable-window` and replaced.
- Fire: a top-level Skill tool_use with skill = learn inside the window.
- Write: a successful `engram learn` (parsed per shell segment: the executable is engram and the
  subcommand is learn; not --help/-h; tool_result not an error and naming a created note path), run
  by the orchestrator or by a non-unit-worker subagent it dispatched inside the window.
- Fast path: no `engram ingest` (parsed the same way) between the return event and the first write.
- Pass (P): a fire, on the fast path, with >= 1 write in the window, and (when the $ARM vault list is
  given) >= 1 note in the vault. Reported, never a pass: fired-with-sweep, fired-no-write,
  write-unconfirmed, captured-not-via-skill, delegated-capture, late, miss.
- False fire (N): any Skill(learn) or `engram learn` invocation inside the window, by the orchestrator
  or a non-unit-worker subagent it dispatched.
- Classification on a fire/write in the window happens BEFORE the question-stop check; a question
  after a capture is only flagged (`ended_on_question`).
- Not scored: gate-fail, degenerate (API error / no or empty result / unit 1 never dispatched to
  unit-worker / its report never came back / the return is not the fixture's unit-1 report verbatim),
  unscorable-window, question-stop (the turn ends on a question before the unit-2 dispatch with
  nothing captured in the window).

    python3 score.py --stream arm/stream.jsonl --session-dir arm/session --cell P1 --arm GREEN \
        --report-file arm/unit1-report.txt --lessons '...' [--vault-notes arm/vault-notes.json]
"""
import argparse
import glob
import json
import os
import re
import shlex
import sys
from typing import Any, Dict, List, Optional

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from config import (AGENT_TOOL_NAMES, ARM_TOKENS, BASH_TOOL_NAME, FILE_MARKERS,  # noqa: E402
                    FIXTURE_AGENT, SKILL_TOOL_NAME)

_TUID = re.compile(r"<tool-use-id>(.*?)</tool-use-id>", re.S)
_RESULT = re.compile(r"<result>(.*?)</result>", re.S)
_NOTE_PATH = re.compile(r"/vault/[^\s/]+\.\d{4}-\d{2}-\d{2}\.[^\s/]+\.md")
_UNIT = re.compile(r"\bunit\s*([123])\b", re.I)
_OPERATORS = {"&&", "||", ";", "|", "&", "\n", "(", ")"}
_ENV_ASSIGN = re.compile(r"^[A-Za-z_][A-Za-z0-9_]*=")
NOT_SCORED = ("gate-fail", "degenerate", "unscorable-window", "question-stop")


# ---------------------------------------------------------------------------
# delivery gate
# ---------------------------------------------------------------------------


def _attachment_text(texts: List[str]) -> str:
    out = []
    for text in texts:
        for line in text.splitlines():
            try:
                rec = json.loads(line)
            except ValueError:
                continue
            if isinstance(rec, dict) and rec.get("type") == "attachment":
                out.append(json.dumps(rec.get("attachment"), ensure_ascii=False))
    return "\n".join(out)


def gate(main_session_texts: List[str], arm: str, other_texts: Optional[List[str]] = None) -> Dict[str, Any]:
    """All four file markers and the arm's own token present in the MAIN session's attachment records
    (where claude -p records CLAUDE.md and its imports; vault note 939); the other arm's token absent
    from every session file."""
    attached = _attachment_text(main_session_texts)
    everything = "\n".join(list(main_session_texts) + list(other_texts or []))
    required = list(FILE_MARKERS.values()) + [ARM_TOKENS[arm]]
    missing = [t for t in required if t not in attached]
    foreign = [tok for name, tok in ARM_TOKENS.items() if name != arm and tok in everything]
    return {"ok": not missing and not foreign, "missing": missing, "foreign": foreign}


# ---------------------------------------------------------------------------
# shell parsing
# ---------------------------------------------------------------------------


def _segments(cmd: str) -> List[List[str]]:
    cmd = cmd.replace("\\\n", " ")
    try:
        lex = shlex.shlex(cmd, posix=True, punctuation_chars="();<>|&\n")
        lex.whitespace = " \t\r"
        lex.whitespace_split = True
        tokens = list(lex)
    except ValueError:
        tokens = cmd.split()
    segs, cur = [], []
    for t in tokens:
        if t in _OPERATORS or set(t) <= set("&|;\n()"):
            if cur:
                segs.append(cur)
            cur = []
        else:
            cur.append(t)
    if cur:
        segs.append(cur)
    return segs


def engram_subcommands(cmd: str) -> List[str]:
    """The engram subcommand of each shell segment whose executable is engram (help calls excluded)."""
    subs = []
    for seg in _segments(cmd):
        words = [w for w in seg]
        while words and _ENV_ASSIGN.match(words[0]):
            words = words[1:]
        if not words or os.path.basename(words[0]) != "engram":
            continue
        rest = [w for w in words[1:] if not w.startswith("-")]
        if not rest or any(w in ("--help", "-h") for w in words[1:]):
            continue
        subs.append(rest[0])
    return subs


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


def _steps(events: List[Dict[str, Any]], include_nested: bool = False):
    """Steps in order: ('use', block, parent) / ('result', block, parent) / ('note', tool_use_id, body, None).
    Top-level only unless include_nested."""
    steps = []
    for e in events:
        parent = e.get("parent_tool_use_id")
        if parent and not include_nested:
            continue
        if e.get("type") not in ("assistant", "user"):
            continue
        content = (e.get("message") or {}).get("content")
        blocks = content if isinstance(content, list) else [content]
        for b in blocks:
            if isinstance(b, dict) and b.get("type") == "tool_use":
                steps.append(("use", b, parent))
            elif isinstance(b, dict) and b.get("type") == "tool_result":
                steps.append(("result", b, parent))
            elif e.get("type") == "user" and not parent:
                text = b if isinstance(b, str) else _text_of(b)
                if text and text.lstrip().startswith("<task-notification>"):
                    tid = _TUID.search(text)
                    res = _RESULT.search(text)
                    steps.append(("note", tid.group(1).strip() if tid else "", res.group(1) if res else "", None))
    return steps


def _is_agent(step) -> bool:
    return step[0] == "use" and step[1].get("name") in AGENT_TOOL_NAMES


def _is_dispatch(step) -> bool:
    return _is_agent(step) and (step[1].get("input") or {}).get("subagent_type") == FIXTURE_AGENT


def _unit_of(step) -> Optional[int]:
    inp = step[1].get("input") or {}
    for field in ("description", "prompt"):
        m = _UNIT.search(str(inp.get(field, "")))
        if m:
            if field == "prompt":
                # a unit-2 prompt may recap unit 1; the highest unit named is the one being assigned
                return max(int(x) for x in _UNIT.findall(str(inp.get(field, ""))))
            return int(m.group(1))
    return None


def _is_skill_learn(step) -> bool:
    if step[0] != "use" or step[1].get("name") != SKILL_TOOL_NAME:
        return False
    skill = str((step[1].get("input") or {}).get("skill", ""))
    return skill == "learn" or skill.endswith(":learn")


def _bash_subs(step) -> List[str]:
    if step[0] != "use" or step[1].get("name") != BASH_TOOL_NAME:
        return []
    return engram_subcommands(str((step[1].get("input") or {}).get("command", "")))


def _norm(s: str) -> str:
    return " ".join(s.split())


def _ends_with_question(text: str) -> bool:
    lines = [ln for ln in text.strip().splitlines() if ln.strip()]
    if not lines:
        return False
    last = re.sub(r"`[^`]*`", "", lines[-1])
    last = re.sub(r"https?://\S+", "", last)
    return last.rstrip(" *_)\"'”").endswith("?")


# ---------------------------------------------------------------------------
# scoring
# ---------------------------------------------------------------------------


def score_arm(stream_lines: List[str], session_texts: List[str], cell: str, arm: str,
              unit1_report: str, lessons: str, other_session_texts: Optional[List[str]] = None,
              vault_notes: Optional[List[str]] = None) -> Dict[str, Any]:
    events = _events(stream_lines)
    steps = _steps(events)
    all_steps = _steps(events, include_nested=True)
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
        "init_tools_agent_names": [t for t in tools if t in AGENT_TOOL_NAMES],
        "init_has_skill_tool": SKILL_TOOL_NAME in tools,
        "init_has_agent_tool": any(t in tools for t in AGENT_TOOL_NAMES),
        "cost_usd": (res or {}).get("total_cost_usd"),
        "duration_ms": (res or {}).get("duration_ms"),
        "num_turns": (res or {}).get("num_turns"),
        "report_verbatim": False, "return_via": None, "late_fire": False, "ended_on_question": False,
        "unit1_redispatched": False, "window_query_calls": 0, "degenerate_reason": None,
        "vault_notes": None if vault_notes is None else len(vault_notes),
        "gate": gate(session_texts, arm, other_session_texts),
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
    if ret_idx is None or _norm(f"LESSONS: {lessons}") not in _norm(body):
        out["degenerate_reason"] = "unit-1 report with the cell's LESSONS line never returned"
        return done("degenerate")
    out["report_verbatim"] = _norm(unit1_report) in _norm(body)
    if not out["report_verbatim"]:
        # a `LESSONS: none` substring also matches the unit-2/3 reports, so only the exact report counts
        out["degenerate_reason"] = "unit-1 return is not the fixture's unit-1 report verbatim"
        return done("degenerate")

    # --- window: ends at the dispatch of unit 2 (or 3), not at a unit-1 re-dispatch
    later = [i for i in dispatches[1:]]
    out["unit1_redispatched"] = any(_unit_of(steps[i]) == 1 for i in later)
    u2 = next((i for i in later if _unit_of(steps[i]) in (2, 3, None)), None)
    if u2 is not None and u2 < ret_idx:
        out["dispatched_before_return"] = True
        return done("unscorable-window")
    w_end = u2 if u2 is not None else len(steps)
    window = range(ret_idx + 1, w_end)

    # map every step (top-level and nested) onto the top-level position of its root dispatch
    top_pos = {s[1].get("id"): i for i, s in enumerate(steps) if s[0] == "use"}
    parent_of = {s[1].get("id"): s[2] for s in all_steps if s[0] == "use"}

    def root(tid: Optional[str]) -> Optional[str]:
        seen = set()
        while tid and tid not in top_pos and tid in parent_of and tid not in seen:
            seen.add(tid)
            tid = parent_of[tid]
        return tid

    fixture_ids = {steps[i][1].get("id") for i in dispatches}
    results = {s[1].get("tool_use_id"): s[1] for s in all_steps if s[0] == "result"}

    def placed(s) -> Optional[int]:
        """Top-level position of a step; for a nested step, its root dispatch's position. None for fixture."""
        if s[2] is None:
            return top_pos.get(s[1].get("id"))
        r = root(s[2])
        if r in fixture_ids:
            return None
        return top_pos.get(r)

    def write_ok(s) -> bool:
        r = results.get(s[1].get("id"))
        return bool(r) and not r.get("is_error") and bool(_NOTE_PATH.search(_text_of(r.get("content"))))

    in_win, after = [], []
    for s in all_steps:
        if s[0] != "use":
            continue
        pos = placed(s)
        if pos is None:
            continue
        # a nested step belongs to its root dispatch's position; order within the window uses the root
        (in_win if pos in window else after if pos >= w_end and u2 is not None else []).append((pos, s))
    in_win.sort(key=lambda x: x[0])

    fires = [p for p, s in in_win if s[2] is None and _is_skill_learn(s)]
    delegated_fires = [p for p, s in in_win if s[2] is not None and _is_skill_learn(s)]
    learn_calls = [(p, s) for p, s in in_win if "learn" in _bash_subs(s)]
    writes = [p for p, s in learn_calls if write_ok(s)]
    delegated_writes = [p for p, s in learn_calls if write_ok(s) and s[2] is not None]
    ingests = [p for p, s in in_win if "ingest" in _bash_subs(s)]
    out["window_query_calls"] = sum(1 for _, s in in_win if "query" in _bash_subs(s))
    out["late_fire"] = any(_is_skill_learn(s) or "learn" in _bash_subs(s) for _, s in after)
    out.update({"window_fires": len(fires), "window_delegated_fires": len(delegated_fires),
                "window_learn_calls": len(learn_calls), "window_learn_writes": len(writes),
                "window_delegated_writes": len(delegated_writes), "window_ingests": len(ingests),
                "window_learn_commands": [str((s[1].get("input") or {}).get("command", ""))[:200]
                                          for _, s in learn_calls]})
    out["ended_on_question"] = u2 is None and _ends_with_question(final_text)
    captured_any = bool(fires or delegated_fires or learn_calls)

    if cell.startswith("N"):
        if captured_any:
            return done("false-fire")
        return done("question-stop" if out["ended_on_question"] else "no-fire")

    if fires:
        if not writes:
            return done("fired-no-write")
        if any(i < writes[0] for i in ingests):
            return done("fired-with-sweep")
        if vault_notes is not None and not vault_notes:
            return done("write-unconfirmed")
        return done("pass")
    if delegated_fires and writes:
        return done("delegated-capture")
    if writes:
        return done("captured-not-via-skill")
    if out["ended_on_question"]:
        return done("question-stop")
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
    ap.add_argument("--vault-notes", help="JSON list of note files left in $ARM/vault")
    a = ap.parse_args(argv)
    with open(a.stream) as f:
        stream = f.readlines()
    main_texts, other_texts = [], []
    for p in sorted(glob.glob(os.path.join(a.session_dir, "**", "*.jsonl"), recursive=True)):
        with open(p, errors="replace") as f:
            (other_texts if os.sep + "subagents" + os.sep in p else main_texts).append(f.read())
    with open(a.report_file) as f:
        report = f.read()
    notes = None
    if a.vault_notes:
        with open(a.vault_notes) as f:
            notes = json.load(f)
    print(json.dumps(score_arm(stream, main_texts, a.cell, a.arm, report, a.lessons, other_texts, notes), indent=2))
    return 0


if __name__ == "__main__":
    sys.exit(main())
