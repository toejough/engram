#!/usr/bin/env python3
"""
Parent-side LESSONS capture check (learn-rate-skill-only, feeds task 3.4).

Pre-registered in parent-capture-plan.md (committed before any measurement).
Question: when a subagent reports a LESSONS item worth capturing, does the
parent session get it into the vault?

Mechanical stages (no LLM): population via run_audit's named rules, receipt +
LESSONS-item extraction from each parent transcript, a read-only vault note set
(HEAD + deleted-in-history + transcript-only notes), exact write times from an
`engram learn|amend|resituate` call index over every transcript, per-item
candidate notes by time window, and a TF-IDF top-10 prefilter for coverage.

Judge stages (sonnet via audit_moments._run_claude_p, cost-logged the same way
as run_window_sample.install_recorders): (a) worth capturing? (b) which
candidate notes capture this item's principle? Raw responses are persisted.

Usage:
  python3 parent_capture.py --output-dir <dir> --dry-run
  python3 parent_capture.py --output-dir <dir> --run [--budget-usd 25]

Read-only on the vault (git ls-tree / log / cat-file) and ~/.claude.
"""

import argparse
import json
import math
import os
import re
import subprocess
import sys
import time
from collections import Counter, OrderedDict
from datetime import date, datetime, timedelta, timezone
from pathlib import Path
from typing import Any, Callable, Dict, Iterable, List, Optional
from zoneinfo import ZoneInfo

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import build_scorecard as bs
import run_audit

SINCE = "2026-09-07"
DEFAULT_VAULT = os.path.expanduser("~/.local/share/engram/vault")
LOCAL_TZ = ZoneInfo("America/New_York")
JUDGE_MODEL = "sonnet"
BATCH_A = 10
BATCH_B = 5
COVERAGE_TOP_K = 10
NOTE_BODY_CHARS_B = 300
NOTE_BODY_CHARS_LEX = 600
# Per-call cost model fitted to the W2 re-measure's cost-log.jsonl (272 sonnet calls).
COST_INTERCEPT_USD = 0.0464
COST_PER_KCHAR_USD = 0.00205
COST_MARGIN = 1.25
DEFAULT_BUDGET_USD = 25.0

SUBAGENT_TOOLS = ("Agent", "Task", "SendMessage")

# ---------------------------------------------------------------------------
# LESSONS block + items
# ---------------------------------------------------------------------------

_MARK = re.compile(r"^[ \t]*(?:[-*>][ \t]*)?(?:\*\*|__)?LESSONS(?:\*\*|__)?[ \t]*:(?:\*\*|__)?[ \t]*", re.M)
_LIST = re.compile(r"^\s*(?:[-*•]|\d+[.)])\s+")
_EMPH = re.compile(r"[*_`]")


def lessons_block(payload: str) -> Optional[str]:
    """Text after the LAST line-start LESSONS marker, to the end of the payload."""
    matches = list(_MARK.finditer(payload or ""))
    if not matches:
        return None
    block = payload[matches[-1].end():]
    kept = []
    for line in block.splitlines():
        s = line.strip()
        if s.startswith("agentId:") or s.startswith("<usage>"):
            break
        kept.append(line)
    return "\n".join(kept).strip()


def is_none(block: str) -> bool:
    text = _EMPH.sub("", block or "").strip().lstrip("-:—–. \t").lower()
    return re.match(r"none\b", text) is not None


def split_items(block: str) -> List[str]:
    lines = [l for l in (block or "").splitlines() if l.strip()]
    if not any(_LIST.match(l) for l in lines):
        joined = " ".join(l.strip() for l in lines)
        return [joined] if joined else []
    items: List[str] = []
    preamble: List[str] = []
    current: Optional[List[str]] = None
    for line in lines:
        m = _LIST.match(line)
        if m:
            if current is not None:
                items.append(" ".join(current))
            current = [line[m.end():].strip()]
        elif current is None:
            preamble.append(line.strip())
        else:
            current.append(line.strip())
    if current is not None:
        items.append(" ".join(current))
    pre = " ".join(preamble)
    if pre and not pre.endswith(":"):
        items.insert(0, pre)
    return [i for i in items if i]


def normalize(text: str) -> str:
    return re.sub(r"\s+", " ", _EMPH.sub("", text or "").lower()).strip()


# ---------------------------------------------------------------------------
# transcript helpers
# ---------------------------------------------------------------------------


def _records(path: str) -> Iterable[Dict[str, Any]]:
    with open(path) as f:
        for line in f:
            try:
                r = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(r, dict):
                yield r


def _text_of(content: Any) -> str:
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(c.get("text", "") for c in content if isinstance(c, dict))
    return ""


def _dt(ts: str) -> datetime:
    return datetime.fromisoformat(ts.replace("Z", "+00:00"))


def local_date(ts: str) -> str:
    return _dt(ts).astimezone(LOCAL_TZ).date().isoformat()


_TUID = re.compile(r"<tool-use-id>([^<]+)</tool-use-id>")
_SUMMARY = re.compile(r"<summary>([^<]*)</summary>")
_RESULT = re.compile(r"<result>(.*?)</result>", re.S)
_OUTPUT = re.compile(r"<output>(.*?)</output>", re.S)
_AGENT_ID = re.compile(r"agentId:\s*([A-Za-z0-9]+)")


def extract_parent(path: str) -> Dict[str, Any]:
    """Receipts and deduped LESSONS items a parent transcript received."""
    uses: Dict[str, Any] = {}
    agent_desc: Dict[str, str] = {}
    items: "OrderedDict[str, Dict[str, Any]]" = OrderedDict()
    n_lessons = n_none = 0
    t_end = None
    session_id = Path(path).name[: -len(".jsonl")]

    def receive(payload: str, ts: str, dispatch: Optional[str]) -> None:
        nonlocal n_lessons, n_none
        block = lessons_block(payload)
        if block is None:
            return
        n_lessons += 1
        if is_none(block):
            n_none += 1
            return
        for text in split_items(block):
            key = normalize(text)
            if key and key not in items:
                items[key] = {"text": text, "norm": key, "t_ret": ts, "dispatch": dispatch}

    for r in _records(path):
        ts = r.get("timestamp")
        if ts:
            t_end = ts
        content = (r.get("message") or {}).get("content")
        blocks = content if isinstance(content, list) else [content]
        for b in blocks:
            if isinstance(b, dict) and b.get("type") == "tool_use":
                uses[b.get("id")] = (b.get("name"), b.get("input") or {})
                continue
            if r.get("type") != "user":
                continue
            text = b if isinstance(b, str) else (b.get("text") if isinstance(b, dict) and b.get("type") == "text" else None)
            if text and text.lstrip().startswith("<task-notification>"):
                tu = _TUID.search(text)
                name, inp = uses.get(tu.group(1).strip(), (None, {})) if tu else (None, {})
                summary = _SUMMARY.search(text)
                is_agent = name in SUBAGENT_TOOLS or (summary and summary.group(1).startswith('Agent "'))
                res = _RESULT.search(text)
                if is_agent and res:
                    desc = inp.get("description") or agent_desc.get(inp.get("to", ""))
                    receive(res.group(1), ts, desc)
                continue
            if isinstance(b, dict) and b.get("type") == "tool_result":
                name, inp = uses.get(b.get("tool_use_id"), (None, {}))
                body = _text_of(b.get("content"))
                if name in ("Agent", "Task"):
                    if body.startswith("Async agent launched"):
                        aid = _AGENT_ID.search(body)
                        if aid:
                            agent_desc[aid.group(1)] = inp.get("description") or ""
                    else:
                        receive(body, ts, inp.get("description"))
                elif name == "TaskOutput":
                    out = _OUTPUT.search(body)
                    if out:
                        receive(out.group(1), ts, agent_desc.get(inp.get("task_id", "")))

    listed = []
    for n, it in enumerate(items.values(), start=1):
        it.update({"id": f"{session_id[:8]}-{n:03d}", "session_id": session_id, "transcript": path})
        listed.append(it)
    return {"session_id": session_id, "transcript": path, "t_end": t_end, "items": listed,
            "receipts_with_lessons": n_lessons, "receipts_none": n_none}


def learn_invocations(path: str) -> List[Dict[str, Any]]:
    out = []
    for r in _records(path):
        content = (r.get("message") or {}).get("content")
        if r.get("type") == "user" and isinstance(content, str) and "<command-name>/learn</command-name>" in content:
            args = re.search(r"<command-args>(.*?)</command-args>", content, re.S)
            a = args.group(1) if args else ""
            out.append({"ts": r.get("timestamp"), "via": "slash", "mid_cycle": "mid-cycle" in a.lower()})
        if isinstance(content, list):
            for b in content:
                if isinstance(b, dict) and b.get("type") == "tool_use" and b.get("name") == "Skill":
                    skill = str((b.get("input") or {}).get("skill", ""))
                    if skill == "learn" or skill.endswith(":learn"):
                        a = str((b.get("input") or {}).get("args", ""))
                        out.append({"ts": r.get("timestamp"), "via": "skill", "mid_cycle": "mid-cycle" in a.lower()})
    return out


def closing_learn(invocations: List[Dict[str, Any]], t_last_lessons: str) -> Dict[str, bool]:
    after = [i for i in invocations if i.get("ts") and _dt(i["ts"]) > _dt(t_last_lessons)]
    return {"any": bool(after), "non_mid_cycle": any(not i["mid_cycle"] for i in after)}


_WRITE_CMD = re.compile(r"(?:^|[;&|(\n]|\$\()\s*(?:\S*/)?engram\s+(?:learn|amend|resituate)\b")


def learn_call_index(paths: Iterable[str], vault: str) -> Dict[str, Dict[str, Any]]:
    """note file -> earliest write evidence {t_w, transcript, command} across transcripts."""
    home = os.path.expanduser("~")
    prefixes = {vault.rstrip("/")}
    if vault.startswith(home):
        prefixes.add("~" + vault[len(home):].rstrip("/"))
    path_re = re.compile("(?:" + "|".join(re.escape(p) for p in prefixes) + r")/([0-9a-z]+\.\d{4}-\d{2}-\d{2}\.[^\s/'\"`]+?\.md)(?![\w.-])")
    index: Dict[str, Dict[str, Any]] = {}
    for path in paths:
        try:
            with open(path) as f:
                if "engram" not in f.read():
                    continue
        except OSError:
            continue
        cmds: Dict[str, str] = {}
        for r in _records(path):
            content = (r.get("message") or {}).get("content")
            if not isinstance(content, list):
                continue
            for b in content:
                if not isinstance(b, dict):
                    continue
                if b.get("type") == "tool_use" and b.get("name") == "Bash":
                    cmd = str((b.get("input") or {}).get("command", ""))
                    if _WRITE_CMD.search(cmd):
                        cmds[b.get("id")] = cmd
                elif b.get("type") == "tool_result" and b.get("tool_use_id") in cmds:
                    ts = r.get("timestamp")
                    for m in path_re.finditer(_text_of(b.get("content"))):
                        f_ = m.group(1)
                        prev = index.get(f_)
                        if ts and (prev is None or _dt(ts) < _dt(prev["t_w"])):
                            index[f_] = {"t_w": ts, "transcript": path, "command": cmds[b["tool_use_id"]][:4000]}
    return index


# ---------------------------------------------------------------------------
# vault note set (read-only git)
# ---------------------------------------------------------------------------

_NOTE_FILE = re.compile(r"^([0-9a-z]+)\.(\d{4}-\d{2}-\d{2})\.(.+)\.md$")
_FIELDS = ("situation", "action", "subject", "predicate", "object", "luhmann", "created", "type", "source")


def _git(repo: str, *args: str, stdin: Optional[bytes] = None) -> bytes:
    return subprocess.run(["git", "-C", repo, *args], input=stdin, capture_output=True, check=True).stdout


def _cat_files(repo: str, specs: List[str]) -> Dict[str, str]:
    if not specs:
        return {}
    out = _git(repo, "cat-file", "--batch", stdin=("\n".join(specs) + "\n").encode())
    result, pos = {}, 0
    for spec in specs:
        nl = out.index(b"\n", pos)
        header = out[pos:nl].decode()
        pos = nl + 1
        if header.endswith("missing"):
            continue
        size = int(header.split()[2])
        result[spec] = out[pos:pos + size].decode("utf-8", "replace")
        pos += size + 1
    return result


def parse_note(text: str) -> Dict[str, str]:
    fields: Dict[str, str] = {}
    body = text
    if text.startswith("---"):
        end = text.find("\n---", 3)
        if end != -1:
            for line in text[3:end].splitlines():
                m = re.match(r"^([a-z_]+):\s*(.*)$", line)
                if m and m.group(1) in _FIELDS:
                    v = m.group(2).strip()
                    if len(v) >= 2 and v[0] == v[-1] and v[0] in "'\"":
                        v = v[1:-1]
                    fields[m.group(1)] = v
            body = text[end + 4:].strip()
    fields["body"] = body
    return fields


def _flag(cmd: str, name: str) -> str:
    m = re.search(r"--" + name + r"[= ](?:\"((?:[^\"\\]|\\.)*)\"|'([^']*)'|(\S+))", cmd)
    return next((g for g in m.groups() if g), "") if m else ""


def _mk_note(file: str, fields: Dict[str, str], source: str, idx: Dict[str, Any]) -> Dict[str, Any]:
    m = _NOTE_FILE.match(file)
    created = fields.get("created") or m.group(2)
    ev = idx.get(file)
    exact = None
    # an index hit counts as the note's write time only on its created date (an amend
    # of an older note is a modification, not the note's write)
    if ev and local_date(ev["t_w"]) == created:
        exact = ev["t_w"]
    return {
        "file": file, "note_id": fields.get("luhmann") or m.group(1), "slug": m.group(3),
        "created": created, "t_w_exact": exact, "writer_transcript": ev["transcript"] if exact else None,
        "source": source, "deleted_at": None, "situation": fields.get("situation", ""),
        "action": fields.get("action") or " ".join(
            fields.get(k, "") for k in ("subject", "predicate", "object")).strip(),
        "body": fields.get("body", ""),
    }


def note_set(repo: str, idx: Dict[str, Dict[str, Any]]) -> List[Dict[str, Any]]:
    head_files = [f for f in _git(repo, "ls-tree", "-r", "--name-only", "HEAD").decode().splitlines()
                  if _NOTE_FILE.match(os.path.basename(f))]
    head = _cat_files(repo, [f"HEAD:{f}" for f in head_files])
    notes: Dict[str, Dict[str, Any]] = {}
    for f in head_files:
        base = os.path.basename(f)
        notes[_NOTE_FILE.match(base).group(3)] = _mk_note(base, parse_note(head.get(f"HEAD:{f}", "")), "head", idx)
    log = _git(repo, "log", "--all", "--no-renames", "--diff-filter=D", "--name-only", "--format=@%H %cI").decode()
    deleted: List[tuple] = []
    commit = when = None
    for line in log.splitlines():
        if line.startswith("@"):
            commit, when = line[1:].split(" ", 1)
        elif line.strip() and _NOTE_FILE.match(os.path.basename(line.strip())):
            deleted.append((commit, line.strip(), datetime.fromisoformat(when).astimezone(timezone.utc)
                            .strftime("%Y-%m-%dT%H:%M:%SZ")))
    blobs = _cat_files(repo, [f"{c}^:{p}" for c, p, _ in deleted])
    for c, p, when in deleted:  # newest deletion first
        base = os.path.basename(p)
        slug = _NOTE_FILE.match(base).group(3)
        if slug not in notes and f"{c}^:{p}" in blobs:
            notes[slug] = _mk_note(base, parse_note(blobs[f"{c}^:{p}"]), "deleted", idx)
            notes[slug]["deleted_at"] = when
    for f, ev in sorted(idx.items()):
        m = _NOTE_FILE.match(f)
        if m and m.group(3) not in notes:
            cmd = ev["command"]
            fields = {"situation": _flag(cmd, "situation"), "action": _flag(cmd, "action") or _flag(cmd, "object"),
                      "body": cmd}
            notes[m.group(3)] = _mk_note(f, fields, "transcript_only", idx)
    return sorted(notes.values(), key=lambda n: n["file"])


# ---------------------------------------------------------------------------
# candidates
# ---------------------------------------------------------------------------


def classify_timing(note: Dict[str, Any], t_ret: str, t_end: str) -> Optional[str]:
    t = _classify_written(note, t_ret, t_end)
    # coverage needs the note to exist at the return; the deleting commit's time is an
    # upper bound on the deletion, so a note whose deletion commit predates t_ret is gone
    if t == "pre" and note.get("deleted_at") and _dt(note["deleted_at"]) <= _dt(t_ret):
        return None
    return t


def _classify_written(note: Dict[str, Any], t_ret: str, t_end: str) -> Optional[str]:
    if note.get("t_w_exact"):
        tw = _dt(note["t_w_exact"])
        if tw <= _dt(t_ret):
            return "pre"
        return "post" if tw <= _dt(t_end) + timedelta(hours=24) else None
    created = note["created"]
    rd = local_date(t_ret)
    ed = (date.fromisoformat(local_date(t_end)) + timedelta(days=1)).isoformat()
    if created < rd:
        return "pre"
    if created == rd:
        return "post_same_day_ambiguous"
    return "post" if created <= ed else None


_STOP = set("a an the and or of to in on for with is are be it this that as by at from not no into when "
            "than then its was were has have had do does did can must should so if any all each".split())


def _tokens(text: str) -> List[str]:
    return [t for t in re.findall(r"[a-z0-9]+", (text or "").lower()) if t not in _STOP and len(t) > 1]


def _lex_doc(n: Dict[str, Any]) -> str:
    return " ".join([n.get("slug", "").replace("-", " "), n.get("situation", ""), n.get("action", ""),
                     (n.get("body") or "")[:NOTE_BODY_CHARS_LEX]])


class LexicalIndex:
    def __init__(self, notes: List[Dict[str, Any]]):
        self.notes = notes
        self.tfs = [Counter(_tokens(_lex_doc(n))) for n in notes]
        df = Counter(t for tf in self.tfs for t in tf)
        self.idf = {t: math.log((1 + len(notes)) / (1 + d)) + 1 for t, d in df.items()}
        self.vecs = [self._vec(tf) for tf in self.tfs]

    def _vec(self, tf: Counter) -> Dict[str, float]:
        v = {t: c * self.idf.get(t, 0.0) for t, c in tf.items()}
        norm = math.sqrt(sum(x * x for x in v.values())) or 1.0
        return {t: x / norm for t, x in v.items()}

    def top_k(self, text: str, k: int, allow: Optional[Callable[[Dict[str, Any]], bool]] = None) -> List[Dict[str, Any]]:
        q = self._vec(Counter(_tokens(text)))
        scored = []
        for n, v in zip(self.notes, self.vecs):
            if allow and not allow(n):
                continue
            s = sum(w * v.get(t, 0.0) for t, w in q.items())
            scored.append((-s, n["file"], n))
        scored.sort(key=lambda x: (x[0], x[1]))
        return [n for _, _, n in scored[:k]]


def top_k_lexical(text: str, notes: List[Dict[str, Any]], k: int) -> List[Dict[str, Any]]:
    return LexicalIndex(notes).top_k(text, k)


# ---------------------------------------------------------------------------
# judge prompts, parsing, batching
# ---------------------------------------------------------------------------

PROMPT_A = """You are applying the engram `learn` skill's capture bar to lessons that subagents reported at the end of their completion reports. For each ITEM decide whether it is worth capturing as a vault note. An item is worth capturing only if ALL hold:
1. It maps to one of four kinds: (1) a correction of an approach or behavior; (2) an explicit save-request; (3) a reversal — a presented conclusion, design, or verdict later overturned, by anyone or by an instrument; (4) a confirmed approach — a specific approach validated by an explicit specific confirmation or by an observable outcome that resolved a real uncertainty (never bare success, never "it worked").
2. It is confirmed, not hypothesized: the item reports something that happened or was verified, not a guess, suggestion, or open question.
3. It states a general, reusable principle that a future agent in a similar situation could act on — not a session-specific narrative, status report, or task summary.

An item that only reports that a lesson was already captured or recalled, without stating the lesson, is not worth capturing. Judge only the item text; do not reward length.

ITEMS:
{items}

Reply with ONLY a JSON array, one object per item, in order:
[{{"id": "<id>", "worth_capturing": true|false, "kind": 1|2|3|4|null, "reason": "<one sentence>"}}]"""

PROMPT_B = """Each ITEM below states a lesson a subagent reported. Each CANDIDATE is an engram vault note. For every item, list every candidate note that captures the item's principle: the same situation-to-action rule, so that a future agent recalling the note would act on the item's lesson. Shared topic or vocabulary alone is NOT a match. A note that captures the item's central principle matches even if it is broader or worded differently; a note that captures only a side detail of the item does not.

ITEMS:
{items}

CANDIDATES:
{candidates}

Reply with ONLY a JSON object mapping each item id to a list (possibly empty) of matches:
{{"<id>": [{{"note_id": "<note_id>", "reason": "<one sentence>"}}], ...}}"""


def _one_line(s: str) -> str:
    return re.sub(r"\s+", " ", s or "").strip()


def render_items(items: List[Dict[str, Any]]) -> str:
    return "\n".join(f"{i['id']}: {_one_line(i['text'])}" for i in items)


def render_candidates(notes: List[Dict[str, Any]], keys: List[str]) -> str:
    return "\n".join(
        f"[{k}] {n['slug']}\nsituation: {_one_line(n['situation'])} | action: {_one_line(n['action'])} | "
        f"body: {_one_line(n['body'])[:NOTE_BODY_CHARS_B]}"
        for k, n in zip(keys, notes))


def candidate_keys(notes: List[Dict[str, Any]]) -> List[str]:
    seen: Counter = Counter()
    keys = []
    for n in notes:
        seen[n["note_id"]] += 1
        keys.append(n["note_id"] if seen[n["note_id"]] == 1 else f"{n['note_id']}~{seen[n['note_id']]}")
    return keys


def parse_json_reply(text: str) -> Any:
    s = re.sub(r"```(?:json)?", "", text or "")
    decoder = json.JSONDecoder()
    for i, ch in enumerate(s):
        if ch in "[{":
            try:
                return decoder.raw_decode(s[i:])[0]
            except json.JSONDecodeError:
                continue
    raise ValueError("no JSON value in reply")


def batch(items: List[Any], size: int, key: Callable[[Any], Any]) -> List[List[Any]]:
    groups: "OrderedDict[Any, List[Any]]" = OrderedDict()
    for it in items:
        groups.setdefault(key(it), []).append(it)
    return [g[i:i + size] for g in groups.values() for i in range(0, len(g), size)]


def outcome(matched_pre: bool, matched_post: bool) -> str:
    if matched_pre:
        return "covered"
    return "captured" if matched_post else "lost"


def _rate(flags: List[bool]) -> Dict[str, Any]:
    return {"numerator": sum(flags), "denominator": len(flags), "rate_pct": bs.pct(sum(flags), len(flags)),
            "ci95": bs.bootstrap_ci(flags)}


def compute_metrics(items: List[Dict[str, Any]]) -> Dict[str, Any]:
    judged = [i for i in items if i.get("worth") is not None]
    worth = [i for i in judged if i["worth"]]
    return {
        "worth": _rate([i["worth"] for i in judged]),
        "C1": _rate([i["outcome"] in ("captured", "covered") for i in worth]),
        "C1_new": _rate([i["outcome"] == "captured" for i in worth if i["outcome"] != "covered"]),
        "C1_strict": _rate([i["outcome"] == "covered" or (i["outcome"] == "captured" and not i.get("ambiguous_only"))
                            for i in worth]),
        "outcomes": dict(Counter(i["outcome"] for i in worth)),
    }


def call_cost(prompt_chars: int) -> float:
    return COST_INTERCEPT_USD + COST_PER_KCHAR_USD * prompt_chars / 1000.0


def project_cost(prompt_char_list: List[int]) -> float:
    return sum(call_cost(c) for c in prompt_char_list) * COST_MARGIN


def check_output_dir(path: Path) -> Path:
    out = Path(path).resolve()
    frozen = run_audit.FROZEN_RESULTS_DIR.resolve()
    if out == frozen or frozen in out.parents:
        raise SystemExit(f"refusing to write into the frozen baseline {frozen}")
    return out


# ---------------------------------------------------------------------------
# pipeline
# ---------------------------------------------------------------------------


def population(projects_root: Optional[str] = None) -> Dict[str, Any]:
    corpus = run_audit.enumerate_corpus(projects_root)
    mains = [p for p in corpus if "/subagents/" not in p]
    kept, dropped = run_audit.filter_since(mains, SINCE)
    eligible, excluded = run_audit.filter_exclusions(kept)
    parents = [extract_parent(p) for p in eligible]
    for p in parents:
        p["learn_invocations"] = learn_invocations(p["transcript"])
    with_items = [p for p in parents if p["items"]]
    return {"all_transcripts": corpus, "counts": {
        "main_enumerated": len(mains), "dropped_by_since": dropped, "after_since": len(kept),
        "excluded_by_rule": excluded, "eligible_main_sessions": len(eligible),
        "sessions_with_any_lessons_receipt": sum(1 for p in parents if p["receipts_with_lessons"]),
        "population_sessions": len(with_items)}, "parents": with_items}


def prepare(vault: str, projects_root: Optional[str] = None) -> Dict[str, Any]:
    pop = population(projects_root)
    idx = learn_call_index(pop["all_transcripts"], vault)
    notes = note_set(vault, idx)
    lex = LexicalIndex(notes)
    items = []
    for p in pop["parents"]:
        for it in p["items"]:
            it["t_end"] = p["t_end"]
            post = [n["file"] for n in notes if (classify_timing(n, it["t_ret"], p["t_end"]) or "").startswith("post")]
            cov = lex.top_k(it["text"], COVERAGE_TOP_K,
                            allow=lambda n: classify_timing(n, it["t_ret"], p["t_end"]) == "pre")
            it["post_candidates"] = post
            it["coverage_candidates"] = [n["file"] for n in cov]
            items.append(it)
    return {"pop": pop, "idx": idx, "notes": notes, "items": items}


def build_batches(items: List[Dict[str, Any]], notes_by_file: Dict[str, Dict[str, Any]]):
    a_batches = batch(items, BATCH_A, key=lambda i: i["session_id"])
    a_prompts = [PROMPT_A.format(items=render_items(b)) for b in a_batches]
    return a_batches, a_prompts


def b_batch_prompt(b: List[Dict[str, Any]], notes_by_file: Dict[str, Dict[str, Any]]):
    files: List[str] = []
    for it in b:
        for f in it["post_candidates"] + it["coverage_candidates"]:
            if f not in files:
                files.append(f)
    notes = [notes_by_file[f] for f in files]
    keys = candidate_keys(notes)
    prompt = PROMPT_B.format(items=render_items(b), candidates=render_candidates(notes, keys))
    return prompt, dict(zip(keys, files))


def b_batches(items: List[Dict[str, Any]]) -> List[List[Dict[str, Any]]]:
    return batch(items, BATCH_B, key=lambda i: (i["session_id"], local_date(i["t_ret"])))


def dry_run(prep: Dict[str, Any], out_dir: Path) -> Dict[str, Any]:
    items = prep["items"]
    notes_by_file = {n["file"]: n for n in prep["notes"]}
    _, a_prompts = build_batches(items, notes_by_file)
    b_prompts = [b_batch_prompt(b, notes_by_file)[0] for b in b_batches(items)]
    a_chars = [len(p) for p in a_prompts]
    b_chars = [len(p) for p in b_prompts]
    per_session = {p["session_id"]: {"items": len(p["items"]), "receipts_with_lessons": p["receipts_with_lessons"],
                                     "receipts_none": p["receipts_none"], "t_end": p["t_end"],
                                     "first_receipt": p["items"][0]["t_ret"]} for p in prep["pop"]["parents"]}
    src = Counter(n["source"] for n in prep["notes"])
    record = {
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "population": prep["pop"]["counts"], "per_session": per_session,
        "items": len(items),
        "vault": {"notes": len(prep["notes"]), "by_source": dict(src),
                  "exact_write_time": sum(1 for n in prep["notes"] if n["t_w_exact"]),
                  "learn_call_index_entries": len(prep["idx"])},
        "candidates": {
            "post_per_item_mean": round(sum(len(i["post_candidates"]) for i in items) / max(1, len(items)), 1),
            "post_per_item_max": max((len(i["post_candidates"]) for i in items), default=0),
            "coverage_per_item": COVERAGE_TOP_K},
        "judge_calls": {"a": len(a_prompts), "b_upper_bound": len(b_prompts)},
        "prompt_chars": {"a_total": sum(a_chars), "b_total": sum(b_chars), "b_max": max(b_chars, default=0)},
        "cost_model": {"intercept_usd": COST_INTERCEPT_USD, "per_kchar_usd": COST_PER_KCHAR_USD,
                       "margin": COST_MARGIN, "basis": "linear fit to results-re-measure-2026-09-28/cost-log.jsonl sonnet calls"},
        "projected_cost_usd": {"a": round(project_cost(a_chars), 2), "b_upper_bound": round(project_cost(b_chars), 2),
                               "total": round(project_cost(a_chars + b_chars), 2)},
        "budget_usd": DEFAULT_BUDGET_USD,
    }
    with open(out_dir / "dryrun.json", "w") as f:
        json.dump(record, f, indent=2)
    with open(out_dir / "items.jsonl", "w") as f:
        for it in items:
            f.write(json.dumps(it) + "\n")
    return record


def _judge(prompt: str, run_claude: Callable, cost_log: Path, label: str) -> Any:
    """One judge call with a single parse retry; returns parsed JSON or None (judge_error)."""
    for attempt in (1, 2):
        out = run_claude(prompt, JUDGE_MODEL)
        with open(cost_log, "a") as f:
            f.write(json.dumps({"t": time.time(), "label": label, "attempt": attempt, "model": JUDGE_MODEL,
                                "prompt_chars": len(prompt), "cost_usd": out.get("total_cost_usd"),
                                "exhausted_retries": out.get("exhausted_retries"),
                                "prompt_too_long": out.get("prompt_too_long")}) + "\n")
        if out.get("exhausted_retries"):
            raise RuntimeError(f"judge exhausted retries at {label}")
        try:
            return {"raw": out.get("result", ""), "parsed": parse_json_reply(out.get("result", ""))}
        except ValueError:
            last = out.get("result", "")
    return {"raw": last, "parsed": None}


def _load_done(path: Path) -> Dict[str, Any]:
    if not path.exists():
        return {}
    return {r["batch_id"]: r for r in (json.loads(l) for l in open(path) if l.strip())}


def run_judges(prep: Dict[str, Any], out_dir: Path, run_claude: Callable) -> List[Dict[str, Any]]:
    items = prep["items"]
    by_id = {i["id"]: i for i in items}
    notes_by_file = {n["file"]: n for n in prep["notes"]}
    cost_log = out_dir / "cost-log.jsonl"

    a_path = out_dir / "judge-a.jsonl"
    done = _load_done(a_path)
    a_batches, a_prompts = build_batches(items, notes_by_file)
    for b, prompt in zip(a_batches, a_prompts):
        bid = "a:" + ",".join(i["id"] for i in b)
        if bid not in done:
            res = _judge(prompt, run_claude, cost_log, bid)
            done[bid] = {"batch_id": bid, **res}
            with open(a_path, "a") as f:
                f.write(json.dumps(done[bid]) + "\n")
    for i in items:
        i["worth"], i["kind"], i["worth_reason"] = None, None, None
    for r in done.values():
        for v in r["parsed"] if isinstance(r["parsed"], list) else []:
            it = by_id.get(str(v.get("id")))
            if it is not None and isinstance(v.get("worth_capturing"), bool):
                it["worth"], it["kind"], it["worth_reason"] = v["worth_capturing"], v.get("kind"), v.get("reason")

    worth = [i for i in items if i["worth"]]
    b_path = out_dir / "judge-b.jsonl"
    done_b = _load_done(b_path)
    for b in b_batches(worth):
        bid = "b:" + ",".join(i["id"] for i in b)
        prompt, keymap = b_batch_prompt(b, notes_by_file)
        if bid not in done_b:
            res = _judge(prompt, run_claude, cost_log, bid)
            done_b[bid] = {"batch_id": bid, "keymap": keymap, **res}
            with open(b_path, "a") as f:
                f.write(json.dumps(done_b[bid]) + "\n")
    for i in worth:
        i["b_error"], i["matches"] = True, []
    for r in done_b.values():
        parsed = r["parsed"] if isinstance(r["parsed"], dict) else {}
        for iid, matches in parsed.items():
            it = by_id.get(str(iid))
            if it is None or not it.get("worth"):
                continue
            it["b_error"] = False
            for m in matches if isinstance(matches, list) else []:
                f_ = r["keymap"].get(str(m.get("note_id")))
                if f_:
                    it["matches"].append({"file": f_, "reason": m.get("reason")})
    for i in items:
        classify_item(i, notes_by_file)
    return items


def classify_item(i: Dict[str, Any], notes_by_file: Dict[str, Dict[str, Any]]) -> None:
    if not i.get("worth"):
        i["outcome"] = None
        return
    if i.get("b_error"):
        i["worth"], i["outcome"] = None, None  # judge_error: excluded from rates
        i["judge_error"] = "b"
        return
    parent_dir = i["transcript"][: -len(".jsonl")] + "/"
    pre, post = [], []
    for m in i["matches"]:
        n = notes_by_file[m["file"]]
        timing = classify_timing(n, i["t_ret"], i["t_end"])
        m["timing"] = timing
        if timing == "pre":
            pre.append(n)
        elif timing and timing.startswith("post"):
            post.append((n, timing))
    i["outcome"] = outcome(bool(pre), bool(post))
    i["ambiguous_only"] = bool(post) and all(t == "post_same_day_ambiguous" for _, t in post)
    attr = "unknown"
    for n, _ in post:
        w = n.get("writer_transcript")
        if w and (w == i["transcript"] or w.startswith(parent_dir)):
            attr = "parent_tree"
            break
        if w:
            attr = "elsewhere"
    i["attribution"] = attr if i["outcome"] == "captured" else None


def scorecard(prep: Dict[str, Any], items: List[Dict[str, Any]]) -> Dict[str, Any]:
    sessions = {}
    c0_any = c0_closing = 0
    for p in prep["pop"]["parents"]:
        last = max(p["items"], key=lambda i: _dt(i["t_ret"]))["t_ret"]
        cl = closing_learn(p["learn_invocations"], last)
        c0_any += cl["any"]
        c0_closing += cl["non_mid_cycle"]
        s_items = [i for i in items if i["session_id"] == p["session_id"]]
        sessions[p["session_id"]] = {"closing_learn": cl, "learn_invocations": p["learn_invocations"],
                                     "last_lessons_receipt": last, "metrics": compute_metrics(s_items)}
    n = len(prep["pop"]["parents"])
    worth = [i for i in items if i.get("worth")]
    return {
        "metrics": compute_metrics(items),
        "C0": {"any_learn_after_last_lessons": {"numerator": c0_any, "denominator": n},
               "non_mid_cycle_learn_after_last_lessons": {"numerator": c0_closing, "denominator": n}},
        "kinds": dict(Counter(str(i.get("kind")) for i in worth)),
        "attribution": dict(Counter(i["attribution"] for i in worth if i.get("outcome") == "captured")),
        "ambiguous_only_captures": sum(1 for i in worth if i.get("outcome") == "captured" and i.get("ambiguous_only")),
        "judge_errors": sum(1 for i in items if i.get("judge_error") or i.get("worth") is None),
        "per_session": sessions,
    }


def main(argv: Optional[List[str]] = None) -> int:
    ap = argparse.ArgumentParser(description="parent-side LESSONS capture check")
    ap.add_argument("--output-dir", required=True)
    ap.add_argument("--vault", default=DEFAULT_VAULT)
    mode = ap.add_mutually_exclusive_group(required=True)
    mode.add_argument("--dry-run", action="store_true")
    mode.add_argument("--run", action="store_true")
    ap.add_argument("--budget-usd", type=float, default=DEFAULT_BUDGET_USD)
    args = ap.parse_args(argv)

    out_dir = check_output_dir(Path(args.output_dir))
    out_dir.mkdir(parents=True, exist_ok=True)
    prep = prepare(args.vault)
    dr = dry_run(prep, out_dir)
    print(json.dumps({k: dr[k] for k in ("population", "per_session", "items", "vault", "candidates",
                                         "judge_calls", "projected_cost_usd")}, indent=2))
    if args.dry_run:
        return 0
    if dr["projected_cost_usd"]["total"] > args.budget_usd:
        print(f"[parent_capture] projected ${dr['projected_cost_usd']['total']} > budget ${args.budget_usd}: stopping")
        return 3

    import audit_moments
    items = run_judges(prep, out_dir, audit_moments._run_claude_p)
    sc = scorecard(prep, items)
    spent = sum((json.loads(l).get("cost_usd") or 0) for l in open(out_dir / "cost-log.jsonl") if l.strip())
    sc["cost_usd_actual"] = round(spent, 4)
    with open(out_dir / "items-judged.jsonl", "w") as f:
        for it in items:
            f.write(json.dumps(it) + "\n")
    with open(out_dir / "scorecard.json", "w") as f:
        json.dump(sc, f, indent=2)
    with open(out_dir / "run-manifest.json", "w") as f:
        json.dump({"finished_at": datetime.now(timezone.utc).isoformat(), "judge_model": JUDGE_MODEL,
                   "cost_usd_actual": round(spent, 4), "projected_cost_usd": dr["projected_cost_usd"],
                   "args": vars(args)}, f, indent=2)
    print(json.dumps({k: sc[k] for k in ("metrics", "C0", "kinds", "attribution", "judge_errors")}, indent=2))
    print(f"[parent_capture] actual cost ${spent:.2f}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
