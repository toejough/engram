#!/usr/bin/env python3
"""Mechanical scorer for one learn-at-return arm (design D5 "Scoring", tasks 1.3).

Inputs: the arm's stream-json stdout lines; the raw text of the MAIN session JSONL (the delivery gate
reads its attachment records only) and of the other session files (subagent transcripts, checked only
for the foreign arm token); and the note files left in $ARM/vault.

Definitions (D5, as tightened by the U1 review, re-review and ruling T5):
- Return event: the tool result of the Agent call that dispatched unit 1 (or, for an async dispatch,
  the task-notification carrying that tool-use id).
- Window: from the return event to the dispatch of UNIT 2. A dispatch's unit is its own identity: the
  unit number (`Unit 2`, `Unit-2`, `unit_2`, `Units 2`) or fixture unit title in its description,
  else in its prompt's first line; mentions of other units in the prompt body are ignored. A
  re-dispatch of unit 1 does not end the window. A unit-worker dispatch with no identity ends it
  conservatively (`window_end_unit: null`). If unit 2 went out before unit 1 came back, the arm is
  `unscorable-window` and replaced.
- engram invocations are parsed per shell segment, through the wrappers timeout/env/command/nice/
  nohup/time, `bash -c`/`sh -c` scripts and backtick substitutions (outside single quotes), with
  env-assignment prefixes skipped and --help/-h excluded. An in-window Bash whose text mentions engram
  and learn but yields no learn/query invocation is listed in `window_unparsed_learn_mentions`.
- Lesson capture (ruling T5): a Skill(learn), or an `engram learn` invocation of kind feedback, fact
  or runbook that is not a route record. Route vs lesson is decided per invocation: its own `--slug`
  (`route-dispatch-*`/`route-evidence-*`, route SKILL.md), else its own created-note path (whole
  output lines, in invocation order). Route records go to `window_route_records`, `engram learn qa`
  writes to `window_qa_writes`, other learn kinds to `window_other_learn_commands`; none of them is a
  pass or a false fire.
- Fire: a top-level Skill tool_use with skill = learn inside the window.
- Write: a lesson-capture `engram learn` whose tool_result is not an error and names a created note
  path, run by the orchestrator or by a non-unit-worker subagent it dispatched inside the window. A
  subagent's tool calls are read from its session file, linked to its dispatch by
  toolUseResult.agentId, because stream-json does not reliably carry them.
- Fast path: no `engram ingest` before the first write, ordered by (top-level step, sequence of a
  nested step within its dispatch, position within the command), so `engram ingest --auto && engram
  learn ...`, or a delegated subagent's ingest call followed by its later learn call, is a sweep.
- Pass (P): a fire, on the fast path, with >= 1 write in the window, and (when the $ARM vault list is
  given) >= 1 vault note created by those writes (ruling T9: notes the unit-worker wrote never
  confirm a pass or make a false fire; every vault note not created by an in-window lesson write,
  whether the worker's or the orchestrator's own late note, is listed in `out_of_window_vault_notes`
  for audit; ruling T11 renamed it from `worker_vault_notes`). Reported, never a pass: fired-with-sweep, fired-no-write,
  write-unconfirmed, captured-not-via-skill, delegated-capture, late, no-fire.
- False fire (N): any lesson capture inside the window (Skill(learn), or a lesson-kind `engram learn`
  invocation, successful or not), by the orchestrator or a non-unit-worker subagent it dispatched.
- Classification on a capture in the window happens BEFORE the question-stop check; a question after
  a capture is only flagged (`ended_on_question`). A trailing (y/n) / [y/N] counts as a question.
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
_CREATED_LINE = re.compile(r"^/\S*/vault/[^\s/]+\.\d{4}-\d{2}-\d{2}\.[^\s/]+\.md$")
_NOTE_PATH = re.compile(r"/vault/[^\s/]+\.\d{4}-\d{2}-\d{2}\.[^\s/]+\.md")
_UNIT = re.compile(r"\bunits?[\s_\-]*([123])\b", re.I)
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


ROUTE_SLUG = re.compile(r"^route-(dispatch|evidence)-")
ROUTE_NOTE = re.compile(r"\.\d{4}-\d{2}-\d{2}\.route-(dispatch|evidence)-[^/\s]*\.md")
LESSON_KINDS = ("feedback", "fact", "runbook")


_WRAPPERS = {"command", "nohup", "time", "exec", "builtin"}
_SHELLS = {"bash", "sh", "zsh", "dash"}


def _backtick_spans(cmd: str):
    """Backtick substitutions bash would execute: outside single quotes. Returns (inner scripts, cmd with
    the spans blanked)."""
    inner, out, i, in_single = [], [], 0, False
    while i < len(cmd):
        c = cmd[i]
        if c == "'" and not in_single:
            in_single = True
        elif c == "'" and in_single:
            in_single = False
        elif c == "`" and not in_single:
            j = cmd.find("`", i + 1)
            if j > i:
                inner.append(cmd[i + 1:j])
                out.append(" ")
                i = j + 1
                continue
        out.append(c)
        i += 1
    return inner, "".join(out)


def _unwrap(words: List[str]) -> List[str]:
    """Strip env assignments and the common command wrappers."""
    while words:
        w0 = os.path.basename(words[0])
        if _ENV_ASSIGN.match(words[0]):
            words = words[1:]
        elif w0 in _WRAPPERS:
            words = words[1:]
        elif w0 == "env":
            words = words[1:]
            while words and (words[0].startswith("-") or _ENV_ASSIGN.match(words[0])):
                words = words[2:] if words[0] in ("-u", "--unset", "-C", "--chdir", "-S") else words[1:]
        elif w0 == "timeout":
            words = words[1:]
            while words and words[0].startswith("-"):
                words = words[2:] if words[0] in ("-s", "--signal", "-k", "--kill-after") else words[1:]
            words = words[1:]  # the duration
        elif w0 == "nice":
            words = words[1:]
            while words and words[0].startswith("-"):
                words = words[2:] if words[0] in ("-n", "--adjustment") else words[1:]
        else:
            return words
    return words


def engram_invocations(cmd: str, _depth: int = 0) -> List[List[str]]:
    """The words after `engram` for each engram invocation in a shell command, in execution order,
    looking through wrappers (timeout, env, command, nice, nohup, time), `bash -c`/`sh -c` scripts and
    backtick substitutions. --help/-h calls are excluded."""
    if _depth > 4:
        return []
    ticks, cmd = _backtick_spans(cmd)
    out = []
    for t in ticks:
        out += engram_invocations(t, _depth + 1)
    for seg in _segments(cmd):
        words = _unwrap(list(seg))
        if not words:
            continue
        exe = os.path.basename(words[0])
        if exe in _SHELLS and "-c" in words[1:]:
            k = words.index("-c")
            if k + 1 < len(words):
                out += engram_invocations(words[k + 1], _depth + 1)
            continue
        if exe != "engram":
            continue
        args = words[1:]
        if any(w in ("--help", "-h") for w in args) or not [w for w in args if not w.startswith("-")]:
            continue
        out.append(args)
    return out


def _slug_of(args: List[str]) -> Optional[str]:
    for i, w in enumerate(args):
        if w == "--slug" and i + 1 < len(args):
            return args[i + 1]
        if w.startswith("--slug="):
            return w.split("=", 1)[1]
    return None


def _learn_kind(args: List[str]) -> Optional[str]:
    rest = [w for w in args if not w.startswith("-")]
    return rest[1] if len(rest) > 1 and rest[0] == "learn" else None


def engram_subcommands(cmd: str) -> List[str]:
    """The engram subcommand of each engram invocation in a shell command, in order."""
    return [[w for w in args if not w.startswith("-")][0] for args in engram_invocations(cmd)]


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


def _records(texts: List[str]):
    for text in texts:
        for line in text.splitlines():
            try:
                rec = json.loads(line)
            except ValueError:
                continue
            if isinstance(rec, dict):
                yield rec


def _subagent_links(texts: List[str]) -> Dict[str, Dict[str, Any]]:
    """agentId -> {tool_use_id, agentType, resolvedModel}, from the tool_result records that carry
    `toolUseResult.agentId` (main session, and subagent files for nested dispatches)."""
    links = {}
    for rec in _records(texts):
        tur = rec.get("toolUseResult")
        content = (rec.get("message") or {}).get("content")
        if not isinstance(tur, dict) or not tur.get("agentId") or not isinstance(content, list):
            continue
        tid = next((b.get("tool_use_id") for b in content if isinstance(b, dict) and b.get("type") == "tool_result"),
                   None)
        if tid:
            links[tur["agentId"]] = {"tool_use_id": tid, "agentType": tur.get("agentType"),
                                     "resolvedModel": tur.get("resolvedModel")}
    return links


def _subagent_steps(texts: List[str], links: Dict[str, Dict[str, Any]]):
    """Tool calls recorded in subagent session files, as nested steps whose parent is the dispatching
    tool_use id. stream-json does not reliably carry a subagent's own tool calls; these files do."""
    steps = []
    for rec in _records(texts):
        link = links.get(rec.get("agentId") or "")
        if not link or rec.get("type") not in ("assistant", "user"):
            continue
        content = (rec.get("message") or {}).get("content")
        for b in content if isinstance(content, list) else []:
            if isinstance(b, dict) and b.get("type") == "tool_use":
                steps.append(("use", b, link["tool_use_id"]))
            elif isinstance(b, dict) and b.get("type") == "tool_result":
                steps.append(("result", b, link["tool_use_id"]))
    return steps


def _is_agent(step) -> bool:
    return step[0] == "use" and step[1].get("name") in AGENT_TOOL_NAMES


def _is_dispatch(step) -> bool:
    return _is_agent(step) and (step[1].get("input") or {}).get("subagent_type") == FIXTURE_AGENT


def _unit_titles() -> Dict[str, int]:
    with open(os.path.join(os.path.dirname(os.path.abspath(__file__)), "fixtures", "reports.json")) as f:
        fx = json.load(f)
    return {t.lower(): i + 1 for d in fx["domains"].values() for i, t in enumerate(d["unit_titles"])}


UNIT_TITLES = _unit_titles()


def _identity(text: str) -> Optional[int]:
    m = _UNIT.search(text)
    if m:
        return int(m.group(1))
    low = text.lower()
    hits = {n for t, n in UNIT_TITLES.items() if t in low}
    return hits.pop() if len(hits) == 1 else None


def _unit_of(step) -> Optional[int]:
    """A dispatch's own unit identity: its description, else its prompt's first line (the title).
    Mentions of other units in the prompt body are ignored."""
    inp = step[1].get("input") or {}
    unit = _identity(str(inp.get("description", "")))
    if unit is None:
        first = next((ln for ln in str(inp.get("prompt", "")).splitlines() if ln.strip()), "")
        unit = _identity(first)
    return unit


def _is_skill_learn(step) -> bool:
    if step[0] != "use" or step[1].get("name") != SKILL_TOOL_NAME:
        return False
    skill = str((step[1].get("input") or {}).get("skill", ""))
    return skill == "learn" or skill.endswith(":learn")


def _norm(s: str) -> str:
    return " ".join(s.split())


_LIST_ITEM = re.compile(r"^\s*(?:[-*+]|\d+[.)])\s+")


def _ends_with_question(text: str) -> bool:
    """The last non-empty line ends with '?' (URLs, inline code and a trailing (y/n)/[y/N] ignored),
    or the text ends with an option list whose introducing line ends with '?'."""
    lines = [ln for ln in text.strip().splitlines() if ln.strip()]
    if lines and _LIST_ITEM.match(lines[-1]):
        while lines and _LIST_ITEM.match(lines[-1]):
            lines.pop()
    return bool(lines) and _is_question_line(lines[-1])


def _is_question_line(line: str) -> bool:
    last = re.sub(r"`[^`]*`", "", line)
    last = re.sub(r"https?://\S+", "", last)
    last = re.sub(r"\s*[\(\[]\s*y(es)?\s*/\s*n(o)?\s*[\)\]]\W*$", "", last, flags=re.I)
    return last.rstrip(" *_)\"'”").endswith("?")


# ---------------------------------------------------------------------------
# scoring
# ---------------------------------------------------------------------------


def score_arm(stream_lines: List[str], session_texts: List[str], cell: str, arm: str,
              unit1_report: str, lessons: str, other_session_texts: Optional[List[str]] = None,
              vault_notes: Optional[List[str]] = None) -> Dict[str, Any]:
    events = _events(stream_lines)
    steps = _steps(events)
    links = _subagent_links(list(session_texts) + list(other_session_texts or []))
    all_steps = _steps(events, include_nested=True)
    seen_ids = {(s[0], s[1].get("id") or s[1].get("tool_use_id")) for s in all_steps if s[0] in ("use", "result")}
    all_steps += [s for s in _subagent_steps(other_session_texts or [], links)
                  if (s[0], s[1].get("id") or s[1].get("tool_use_id")) not in seen_ids]
    init = next((e for e in events if e.get("type") == "system" and e.get("subtype") == "init"), {})
    res = next((e for e in reversed(events) if e.get("type") == "result"), None)
    tools = init.get("tools") or []
    fixture_models = sorted({(e.get("message") or {}).get("model") for e in events
                             if e.get("type") == "assistant" and e.get("parent_tool_use_id")
                             and (e.get("message") or {}).get("model")}
                            | {v["resolvedModel"] for v in links.values()
                               if v.get("agentType") == FIXTURE_AGENT and v.get("resolvedModel")})
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
    out["window_end_unit"] = _unit_of(steps[u2]) if u2 is not None else "end-of-session"
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
    # nested steps share their root dispatch's position, so order them by their own sequence index
    # (stream order, then each subagent file's order) as the tie-break (re-review N7)
    seq_of = {id(st): k for k, st in enumerate(all_steps)}
    in_win.sort(key=lambda x: (x[0], seq_of[id(x[1])]))

    fires = [p for p, s in in_win if s[2] is None and _is_skill_learn(s)]
    delegated_fires = [p for p, s in in_win if s[2] is not None and _is_skill_learn(s)]
    def result_text(s) -> str:
        r = results.get(s[1].get("id"))
        return _text_of(r.get("content")) if r else ""

    def cmd_of(s) -> str:
        return str((s[1].get("input") or {}).get("command", ""))

    def created_paths(s) -> List[str]:
        """Note paths the step's output prints as whole lines (engram learn's created-note line)."""
        return [ln.strip() for ln in result_text(s).splitlines() if _CREATED_LINE.match(ln.strip())]

    def invocations(s):
        """(order, class, args, note) per engram invocation of a Bash step. class is one of
        lesson / route / qa / other-learn / ingest / query / other. Route vs lesson is decided per
        invocation: its own --slug, else its own created-note path (whole output lines, in order)."""
        if s[1].get("name") != BASH_TOOL_NAME:
            return []
        paths = created_paths(s)
        learn_idx = 0
        rows = []
        for order, args in enumerate(engram_invocations(cmd_of(s))):
            sub = [w for w in args if not w.startswith("-")][0]
            if sub != "learn":
                rows.append((order, sub if sub in ("ingest", "query") else "other", args, None))
                continue
            slug = _slug_of(args)
            note = None
            if slug:
                note = next((p for p in paths if ("." + slug + ".md") in p), None)
            elif learn_idx < len(paths):
                note = paths[learn_idx]
            learn_idx += 1
            kind = _learn_kind(args)
            if (slug and ROUTE_SLUG.match(slug)) or (not slug and note and ROUTE_NOTE.search(note)):
                cls = "route"
            elif kind == "qa":
                cls = "qa"
            elif kind in LESSON_KINDS:
                cls = "lesson"
            else:
                cls = "other-learn"
            rows.append((order, cls, args, note))
        return rows

    inv = [(p, s, row) for p, s in in_win for row in invocations(s)]
    learn_calls = [(p, s, r) for p, s, r in inv if r[1] == "lesson"]
    out["window_route_records"] = [{"command": cmd_of(s), "note": r[3]} for _, s, r in inv if r[1] == "route"]
    out["window_qa_writes"] = [{"command": cmd_of(s), "note": r[3]} for _, s, r in inv if r[1] == "qa"]
    out["window_other_learn_commands"] = [cmd_of(s)[:200] for _, s, r in inv if r[1] == "other-learn"]
    out["window_unparsed_learn_mentions"] = [
        cmd_of(s) for _, s in in_win
        if s[1].get("name") == BASH_TOOL_NAME and re.search(r"\bengram\b.*\blearn\b", cmd_of(s), re.S)
        and not any(r[1] in ("lesson", "route", "qa", "other-learn") for r in invocations(s))
        and not any(r[1] == "query" for r in invocations(s))]
    writes = sorted({(p, seq_of[id(s)], r[0]) for p, s, r in learn_calls if write_ok(s)})
    # T9: only notes the orchestrator's own (or delegated non-worker) writes created confirm a pass
    own_notes = sorted({os.path.basename(ln) for _, s, _ in learn_calls if write_ok(s)
                        for ln in created_paths(s)})
    fixture_notes = sorted({os.path.basename(ln) for s in all_steps
                            if s[0] == "use" and s[2] is not None and root(s[2]) in fixture_ids
                            and s[1].get("name") == BASH_TOOL_NAME for ln in created_paths(s)})
    confirmed = sorted(set(vault_notes or []) & set(own_notes))
    out["confirmed_vault_notes"] = confirmed
    out["out_of_window_vault_notes"] = sorted(set(vault_notes or []) - set(own_notes)) if vault_notes is not None \
        else fixture_notes
    delegated_writes = [p for p, s, r in learn_calls if write_ok(s) and s[2] is not None]
    ingests = sorted((p, seq_of[id(s)], r[0]) for p, s, r in inv if r[1] == "ingest")
    out["window_query_calls"] = sum(1 for _, _, r in inv if r[1] == "query")
    out["late_fire"] = any(_is_skill_learn(s) or any(r[1] == "lesson" for r in invocations(s)) for _, s in after)
    out.update({"window_fires": len(fires), "window_delegated_fires": len(delegated_fires),
                "window_learn_calls": len(learn_calls), "window_learn_writes": len(writes),
                "window_delegated_writes": len(delegated_writes), "window_ingests": len(ingests),
                "window_learn_commands": [str((s[1].get("input") or {}).get("command", ""))[:200]
                                          for _, s, _ in learn_calls]})
    out["ended_on_question"] = u2 is None and _ends_with_question(final_text)
    captured_any = bool(fires or delegated_fires or learn_calls)

    if cell.startswith("N"):
        if captured_any:
            return done("false-fire")
        return done("question-stop" if out["ended_on_question"] else "no-fire")

    if fires:
        if not writes:
            return done("fired-no-write")
        if any(i < writes[0] for i in ingests):  # (step position, nested sequence, order within command)
            return done("fired-with-sweep")
        if vault_notes is not None and not confirmed:
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
    return done("no-fire")


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
