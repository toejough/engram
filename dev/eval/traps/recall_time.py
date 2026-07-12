"""Recall-only wall-time re-measure (#657): how long does the /recall procedure take TODAY,
post lazy-chunks/recent-fill/O2/L2 cuts? The prior (LEDGER `recall-time-isolated`) is ~190 s,
n=2 directional, pre-cuts vintage — never re-measured since.

One trial = one warm `claude -p` run (wrun.py plumbing: RECALL_PREFIX, warm cfg with the real
recall/learn skills, seed_c3-seeded sandbox vault, small fixture task). The recall span is read
from the trial's session transcript (under the SANDBOX cfg dir, walked exactly as wrun.py does)
by a pinned, mechanical delimiter procedure — no eyeballing:

  START = timestamp of the session's first TIMESTAMPED record (leading metadata records —
          `mode`, `last-prompt`, etc. — carry none; the trial is recall-first by construction).
  END   = scanning `type: assistant` records BACKWARD from the last: the first whose content
          matches r"Query surfaced [0-9]+ items" (the recall synthesis opener); if none, the
          first (still backward, still assistant-type) containing "Re-entry:"; if neither,
          the trial FAILS with a transcript-shape report — never estimated, never pooled.

Host-sleep guard (pilot finding, 2026-07-11): macOS idle-sleeps mid-trial when unattended;
transcript timestamps are wall-clock but monotonic clocks (claude's duration_ms, our wall_s)
stop during sleep, so a napping host silently inflates the span. Each trial therefore runs
under `caffeinate -is`, and a trial whose span exceeds its monotonic wall time (+2 s tolerance)
is marked sleep-contaminated and NOT pooled (span > awake-process wall is physically impossible).

Vault modes (Task 1b): the default fixture seeds 5 seed_c3 notes — a small-payload floor that
never exercises the paging regime the ~190 s prior ran against (real-vault copy, ~141–237 KB
payloads, ~370–410 chunks; method: git show 51ca6723:docs/design/2026-06-25-recall-cost-isolation.md).
`--vault-copy` instead COPIES the real vault + chunk index into each per-trial sandbox and points
ENGRAM_VAULT_PATH/ENGRAM_CHUNKS_DIR at the copies — the production paths are read-only sources;
every write (learn/activate/ingest) lands in the sandbox. Vault-copy rows live under the
artifact's `vault_copy` key, summarized separately — the two modes are never pooled.

Phase segmentation + payload census (#684, `--segment`): splits the recall_span into four
mechanical phases from the SAME transcript — (a) pre_query: START to the first `engram query`
tool_use; (b) query_call: SUM of each query call's own tool_use->tool_result span; (c)
payload_consumption: first query tool_result -> the Step-2.5-start marker, minus any subsequent
query calls' in-flight spans inside that window; (d) remainder: marker -> END. The marker is
min(Arm 1: first post-query `engram amend`/`activate`/`learn` tool_use, Arm 2: first assistant
record matching END_PRIMARY) — see compute_phases(). A per-trial validity gate requires every
phase >= 0 and the four phases to sum to the START->END span (+-1s); gate failures are reported,
never pooled (same discard-never-pool contract as the sleep gate). The payload census (see
compute_census()) bytes-profiles the trial's own `engram query` output — items vs clusters
sections, note-content vs YAML markup, and content duplicated between items[] and candidate_l2s
— sliced from the RAW captured YAML (never re-dumped structs, which would carry different
markup bytes) via `yq` (no PyYAML dependency in this environment). Batch mode ALSO copies each
trial's transcript (and, under --segment, its captured query payload) into a durable
`<out>-transcripts/` directory next to the artifact, fixing the ephemerality of the sandbox
paths recorded in `transcript_path` (ROOT gets rmtree'd by the NEXT invocation's first trial).

Usage:
  python3 recall_time.py --mode pilot [--model opus]          # n=1, validate the instrument
  python3 recall_time.py --mode batch --n 3 --out <artifact>  # fresh sandbox per trial
  python3 recall_time.py --mode batch --n 3 --vault-copy --out <artifact>  # production-scale
"""
import argparse
import datetime
import json
import os
import re
import shlex
import shutil
import statistics
import subprocess
import sys
import time

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import seed_c3
import traps as T
from run import MODELS
from wrun import RECALL_PREFIX, _slug, build_warm_cfg

ROOT = os.environ.get("RECALL_TIME_ROOT", "/tmp/recall-time-657")
_XDG_DATA = os.environ.get("XDG_DATA_HOME", os.path.expanduser("~/.local/share"))
REAL_VAULT = os.path.join(_XDG_DATA, "engram", "vault")  # read-only copy SOURCE, never written
REAL_CHUNKS = os.path.join(_XDG_DATA, "engram", "chunks")  # read-only copy SOURCE, never written
# Count token is tolerant (`6`, `8+`, `~20+`): on the real vault the agent hedges the count
# ("Query surfaced ~20+ items"); the strict [0-9]+ form STOP-failed 2/2 vault-copy trials on
# 2026-07-12 (transcripts preserved) while the END-record identity — the recall-synthesis
# opener — was unambiguous in both. Skill-body template text ("Query surfaced N items") never
# matches: N is not a digit, and templates ride user/attachment records, not assistant ones.
END_PRIMARY = re.compile(r"Query surfaced ~?[0-9]+\+? items")
END_FALLBACK = "Re-entry:"
# Small fixture task (post-recall work stays trivial); its convention is one of the seeded notes.
FIXTURE = T.TRAPS["nocolor"]["prompt"]
MAX_ATTEMPTS = 2  # one retry on a transient flake (is_error + near-zero cost), noted per row
DEGRADED_COST_RATIO = 0.5  # cost < ratio x median-of-siblings => flag as possibly degraded
SLEEP_TOLERANCE_S = 2.0  # span may exceed monotonic wall by at most this before we call it sleep


def parse_ts(ts):
    return datetime.datetime.fromisoformat(ts.replace("Z", "+00:00"))


def record_text(rec):
    """All text carried by a transcript record's message content, concatenated."""
    content = (rec.get("message") or {}).get("content")
    if isinstance(content, str):
        return content
    parts = []
    if isinstance(content, list):
        for block in content:
            if isinstance(block, dict) and isinstance(block.get("text"), str):
                parts.append(block["text"])
    return "\n".join(parts)


def load_records(transcript_path):
    """Parse a transcript's JSONL records (shared by find_span and the phase/census logic
    below, so a trial's transcript is read from disk exactly once)."""
    records = []
    with open(transcript_path, errors="ignore") as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                records.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    return records


def find_span(records):
    """Apply the pinned delimiter procedure. Returns (span_dict, None) or (None, shape_report)."""
    start_rec = next((r for r in records if r.get("timestamp")), None)
    end_rec, end_rule, end_match = None, None, None
    assistants = [r for r in records if r.get("type") == "assistant"]
    for rec in reversed(assistants):
        m = END_PRIMARY.search(record_text(rec))
        if m:
            end_rec, end_rule, end_match = rec, "query-surfaced", m.group(0)
            break
    if end_rec is None:
        for rec in reversed(assistants):
            text = record_text(rec)
            if END_FALLBACK in text:
                idx = text.index(END_FALLBACK)
                end_rec, end_rule = rec, "re-entry"
                end_match = text[idx:idx + 80].splitlines()[0]
                break

    if start_rec is None or end_rec is None or not end_rec.get("timestamp"):
        by_type = {}
        for rec in records:
            by_type[rec.get("type", "?")] = by_type.get(rec.get("type", "?"), 0) + 1
        shape = {
            "records": len(records), "by_type": by_type,
            "assistant_records": len(assistants),
            "first_timestamped": (start_rec or {}).get("type"),
            "last_assistant_texts": [record_text(r)[:200] for r in assistants[-3:]],
        }
        return None, shape

    start_ts, end_ts = parse_ts(start_rec["timestamp"]), parse_ts(end_rec["timestamp"])
    return {
        "start_ts": start_rec["timestamp"], "start_record_type": start_rec.get("type"),
        "end_ts": end_rec["timestamp"], "end_rule": end_rule, "end_match": end_match,
        "recall_span_s": round((end_ts - start_ts).total_seconds(), 1),
    }, None


def find_transcript(cfg, session_id):
    """Walk the sandbox cfg's projects dir for {session_id}.jsonl, exactly as wrun.py does."""
    for root, _, files in os.walk(os.path.join(cfg, "projects")):
        if f"{session_id}.jsonl" in files:
            return os.path.join(root, f"{session_id}.jsonl")
    return None


# ---------------------------------------------------------------------------------------
# #684 phase segmentation: pinned phase model (plan Global Constraints — not redesigned
# here, only implemented). Verified against the preserved #657 vault-copy transcripts
# (vc-0, vc-2) before being wired into the batch — see task-1-report.md Step 1.
# ---------------------------------------------------------------------------------------
WRITE_MARKERS = ("engram amend", "engram activate", "engram learn")


def _tool_use_blocks(records):
    """Yield (record_index, record, block) for every assistant tool_use content block."""
    for i, rec in enumerate(records):
        if rec.get("type") != "assistant":
            continue
        content = (rec.get("message") or {}).get("content")
        if not isinstance(content, list):
            continue
        for block in content:
            if isinstance(block, dict) and block.get("type") == "tool_use":
                yield i, rec, block


def _bash_command(block):
    return (block.get("input") or {}).get("command") or ""


def _tool_result_text(block):
    content = (block or {}).get("content")
    if isinstance(content, str):
        return content
    parts = []
    if isinstance(content, list):
        for b in content:
            if isinstance(b, dict) and isinstance(b.get("text"), str):
                parts.append(b["text"])
    return "\n".join(parts)


def _find_tool_result(records, tool_use_id, after_index):
    """First tool_result (any record type) carrying tool_use_id, scanning forward."""
    for rec in records[after_index:]:
        content = (rec.get("message") or {}).get("content")
        if not isinstance(content, list):
            continue
        for block in content:
            if (isinstance(block, dict) and block.get("type") == "tool_result"
                    and block.get("tool_use_id") == tool_use_id):
                return rec.get("timestamp"), block
    return None, None


def find_query_calls(records):
    """Every `engram query` Bash tool_use -> tool_result span, in transcript order. A
    compound piped call (`engram query ... | head -200`, observed on every preserved #657
    query call) is still ONE call and counts wholly under the per-call-span rule — never
    split, per the plan's compound-call caveat."""
    calls = []
    for i, rec, block in _tool_use_blocks(records):
        if block.get("name") != "Bash" or "engram query" not in _bash_command(block):
            continue
        use_ts = rec.get("timestamp")
        result_ts, result_block = _find_tool_result(records, block.get("id"), i + 1)
        if not use_ts or not result_ts:
            continue
        calls.append({
            "tool_use_ts": use_ts, "tool_result_ts": result_ts,
            "command": _bash_command(block), "result_text": _tool_result_text(result_block),
        })
    return calls


def find_write_marker(records, after_ts):
    """Arm 1: first assistant Bash tool_use invoking engram amend/activate/learn whose OWN
    timestamp is after `after_ts` (the first query tool_result)."""
    after = parse_ts(after_ts)
    for _, rec, block in _tool_use_blocks(records):
        if block.get("name") != "Bash":
            continue
        ts = rec.get("timestamp")
        if not ts or parse_ts(ts) <= after:
            continue
        if any(marker in _bash_command(block) for marker in WRITE_MARKERS):
            return ts
    return None


def find_synthesis_marker(records):
    """Arm 2: FORWARD scan — first assistant record matching END_PRIMARY (contrast with
    find_span's END, which is a BACKWARD scan for the LAST match; the two coincide exactly
    when a transcript has exactly one END_PRIMARY match, the observed shape on 4/4
    preserved fixtures)."""
    for rec in records:
        if rec.get("type") != "assistant":
            continue
        if END_PRIMARY.search(record_text(rec)):
            return rec.get("timestamp")
    return None


def compute_phases(records, start_ts, end_ts):
    """Segment [start_ts, end_ts] into the four #684 phases. Returns (phases_dict, None) on
    success or (None, reason) if a query call or BOTH marker arms cannot be located
    mechanically — STOP, never estimated (plan Global Constraints)."""
    query_calls = find_query_calls(records)
    if not query_calls:
        return None, "no `engram query` tool_use found — cannot segment"

    first_use_ts = query_calls[0]["tool_use_ts"]
    first_result_ts = query_calls[0]["tool_result_ts"]
    arm1_ts = find_write_marker(records, first_result_ts)
    arm2_ts = find_synthesis_marker(records)
    candidates = [t for t in (arm1_ts, arm2_ts) if t]
    if not candidates:
        return None, "neither Arm 1 (write call) nor Arm 2 (synthesis text) found — cannot mark Step 2.5"
    marker_ts = min(candidates, key=parse_ts)
    marker_arm = ("arm1-write" if marker_ts == arm1_ts and marker_ts != arm2_ts else
                  "arm2-synthesis" if marker_ts == arm2_ts and marker_ts != arm1_ts else "arm1-arm2-tie")

    start_dt, end_dt, marker_dt = parse_ts(start_ts), parse_ts(end_ts), parse_ts(marker_ts)
    first_use_dt, first_result_dt = parse_ts(first_use_ts), parse_ts(first_result_ts)

    pre_query_s = (first_use_dt - start_dt).total_seconds()
    query_call_s = sum(
        (parse_ts(c["tool_result_ts"]) - parse_ts(c["tool_use_ts"])).total_seconds() for c in query_calls)
    subsequent_in_window_s = sum(
        (parse_ts(c["tool_result_ts"]) - parse_ts(c["tool_use_ts"])).total_seconds()
        for c in query_calls[1:]
        if first_result_dt <= parse_ts(c["tool_use_ts"]) and parse_ts(c["tool_result_ts"]) <= marker_dt)
    payload_consumption_s = (marker_dt - first_result_dt).total_seconds() - subsequent_in_window_s
    remainder_s = (end_dt - marker_dt).total_seconds()

    phases = {
        "pre_query_s": round(pre_query_s, 1), "query_call_s": round(query_call_s, 1),
        "payload_consumption_s": round(payload_consumption_s, 1), "remainder_s": round(remainder_s, 1),
        "marker_ts": marker_ts, "marker_arm": marker_arm,
        "arm1_write_ts": arm1_ts, "arm2_synthesis_ts": arm2_ts, "n_query_calls": len(query_calls),
    }
    span_s = round((end_dt - start_dt).total_seconds(), 1)
    total_s = (phases["pre_query_s"] + phases["query_call_s"]
               + phases["payload_consumption_s"] + phases["remainder_s"])
    all_nonneg = all(phases[k] >= 0 for k in
                      ("pre_query_s", "query_call_s", "payload_consumption_s", "remainder_s"))
    sum_ok = abs(total_s - span_s) <= 1.0
    phases["gate_ok"] = bool(all_nonneg and sum_ok)
    phases["gate_detail"] = ("PASS" if phases["gate_ok"] else
                              f"FAIL: all_nonneg={all_nonneg} phase_sum={total_s} span={span_s}")
    return phases, None


# ---------------------------------------------------------------------------------------
# #684 payload census: byte-profile the trial's own `engram query` output (plan Step 2
# formulas — not redesigned here, only implemented). No PyYAML in this environment (pip is
# broken locally); `yq` (kislyuk python-yq, wrapping PyYAML, confirmed on PATH) parses the
# payload into JSON for content-field access, while section/item BYTE spans are always
# sliced from the RAW captured text per the plan's explicit "slice, don't re-dump" method.
# ---------------------------------------------------------------------------------------
_TOP_LEVEL_KEYS = ("version", "phrases", "items", "clusters", "budget", "refit_pending")
_TOP_LEVEL_RE = re.compile(r"^(" + "|".join(_TOP_LEVEL_KEYS) + r"):", re.MULTILINE)
_ITEM_START_RE = re.compile(r"^  - path: ", re.MULTILINE)


def _yq_flavor():
    try:
        out = subprocess.run(["yq", "--version"], capture_output=True, text=True, timeout=10).stdout
    except (FileNotFoundError, subprocess.TimeoutExpired):
        return None
    return "go" if "mikefarah" in out.lower() else "python"


def yaml_to_json(raw_text):
    """Shell out to `yq` to parse the payload YAML into JSON. Fails LOUD (raises) if `yq`
    is missing or the parse fails — a payload census is never silently approximated."""
    flavor = _yq_flavor()
    if flavor is None:
        raise RuntimeError("`yq` not found on PATH — required to parse the query payload for census")
    args = ["yq", "-o=json", "eval", "."] if flavor == "go" else ["yq", "."]
    proc = subprocess.run(args, input=raw_text, capture_output=True, text=True, timeout=30)
    if proc.returncode != 0:
        raise RuntimeError(f"yq failed to parse payload YAML: {proc.stderr.strip()[:300]}")
    return json.loads(proc.stdout)


def slice_top_level_sections(raw_text):
    """Byte-exact spans of each top-level payload key (its line to the next top-level
    key's line, or EOF), sliced from the RAW text — never re-dumped structs, whose markup
    bytes differ from what the binary actually emitted."""
    matches = list(_TOP_LEVEL_RE.finditer(raw_text))
    sections = {}
    for idx, m in enumerate(matches):
        start = m.start()
        end = matches[idx + 1].start() if idx + 1 < len(matches) else len(raw_text)
        sections[m.group(1)] = raw_text[start:end]
    return sections


def payload_is_complete(raw_text):
    """A capture is complete iff it parses AND carries all 5 top-level keys — `budget` is
    the LAST-rendered struct field (query.go's queryPayload field order), so its presence
    proves the capture wasn't cut off by an agent's own `| head` / `| sed` pipe (observed
    on every preserved #657 query call — see task-1-report.md)."""
    try:
        doc = yaml_to_json(raw_text)
    except (RuntimeError, json.JSONDecodeError):
        return False
    return isinstance(doc, dict) and all(k in doc for k in ("version", "phrases", "items", "clusters", "budget"))


def compute_census(raw_text):
    """Payload byte census per the plan's pinned formulas (Step 2). Raises RuntimeError
    (yq) or ValueError (shape mismatch) if the payload can't be measured — never
    estimated."""
    doc = yaml_to_json(raw_text)
    if not isinstance(doc, dict):
        raise ValueError("payload did not parse to a YAML mapping")
    sections = slice_top_level_sections(raw_text)
    items = doc.get("items") or []
    clusters = doc.get("clusters") or []

    items_section = sections.get("items", "")
    starts = [m.start() for m in _ITEM_START_RE.finditer(items_section)]
    item_blocks = [items_section[s:(starts[i + 1] if i + 1 < len(starts) else len(items_section))]
                   for i, s in enumerate(starts)]
    if len(item_blocks) != len(items):
        raise ValueError(
            f"item block count mismatch: {len(item_blocks)} raw blocks vs {len(items)} parsed items "
            "— the 2-space '- path:' boundary assumption broke; escalate rather than guess")

    def content_bytes(entry):
        return len((entry.get("content") or "").encode("utf-8"))

    items_bytes = len(items_section.encode("utf-8"))
    clusters_bytes = len(sections.get("clusters", "").encode("utf-8"))
    items_all_content_bytes = sum(content_bytes(it) for it in items)
    items_notes_content_bytes = sum(content_bytes(it) for it in items if it.get("kind") != "chunk")
    items_meta_bytes = items_bytes - items_all_content_bytes

    candidate_content_bytes = 0
    candidate_paths = set()
    for cluster in clusters:
        for cand in cluster.get("candidate_l2s") or []:
            candidate_content_bytes += content_bytes(cand)
            candidate_paths.add(cand.get("path"))
    clusters_meta_bytes = clusters_bytes - candidate_content_bytes

    # item_blocks[i] is item[i]'s RAW block by construction (both in document order); zip
    # positionally rather than by path — items[] carries genuine path duplicates (the same
    # note surfacing via >1 provenance channel), which would silently collide in a dict.
    recent_bytes = sum(len(block.encode("utf-8")) for it, block in zip(items, item_blocks)
                       if "recent" in (it.get("provenances") or []))

    duplicated_note_content_bytes = sum(
        content_bytes(it) for it in items
        if it.get("kind") != "chunk" and it.get("path") in candidate_paths)

    return {
        "total_bytes": len(raw_text.encode("utf-8")),
        "items_notes_content_bytes": items_notes_content_bytes,
        "items_meta_bytes": items_meta_bytes,
        "clusters_candidate_content_bytes": candidate_content_bytes,
        "clusters_meta_bytes": clusters_meta_bytes,
        "recent_bytes": recent_bytes,
        "duplicated_note_content_bytes": duplicated_note_content_bytes,
    }


def _clean_query_command(cmd):
    """Reduce a captured Bash command to the bare `engram query ...` argv, dropping any
    redirection/pipe suffix an agent added (e.g. `2>&1 | head -200`) — those pipes are why
    a transcript-captured payload can be truncated — and un-escaping the `\\\\\\n` line
    continuations agents wrap long --phrase chains in. Returns an argv list (never a shell
    string): the reissue then runs shell=False, so no shell-metacharacter risk."""
    if " 2>&1" in cmd:
        cmd = cmd.split(" 2>&1", 1)[0]
    elif " | " in cmd:
        cmd = cmd.split(" | ", 1)[0]
    cmd = cmd.replace("\\\n", " ")
    return shlex.split(cmd)


def capture_query_payload(query_calls, env, wd):
    """The trial's raw query payload YAML for the census: prefers the transcript's own
    first-query-call capture (what the agent actually saw); if that capture was truncated
    by the agent's own pipe, reissues the SAME query (pipe stripped) directly against the
    trial's own sandbox (still on disk — sandboxes are rmtree'd at the START of the NEXT
    run_trial call, not on exit) so the census still measures a complete payload. Returns
    (raw_text, source_label) or (None, failure_reason)."""
    first = query_calls[0]
    raw = first["result_text"]
    if payload_is_complete(raw):
        return raw, "transcript (first query tool_result)"

    clean_argv = _clean_query_command(first["command"])
    try:
        proc = subprocess.run(clean_argv, cwd=wd, env=env, capture_output=True, text=True, timeout=120)
    except subprocess.TimeoutExpired:
        return None, "reissue timed out"
    except FileNotFoundError as exc:
        return None, f"reissue command not found: {exc}"
    if payload_is_complete(proc.stdout):
        return proc.stdout, "reissued (transcript capture truncated by agent's own pipe)"
    return None, f"reissued capture also incomplete (rc={proc.returncode}): {proc.stderr[:200]}"


def run_trial(idx, model, vault_copy=False, segment=False):
    """One warm trial in a fresh sandbox (fresh cfg + vault + workdir + chunks dir).

    vault_copy=True: production-scale contrast — copy the REAL vault + chunk index into the
    sandbox (sources read-only; all trial writes stay inside the copies). Otherwise seed_c3."""
    base = os.path.join(ROOT, f"trial-vc-{idx}" if vault_copy else f"trial-{idx}")
    shutil.rmtree(base, ignore_errors=True)
    wd = os.path.join(base, "ws")
    vault = os.path.join(base, "vault")
    chunks = os.path.join(base, "chunks")
    cfg = os.path.join(base, "cfg")
    os.makedirs(wd, exist_ok=True)
    build_warm_cfg(cfg)
    if vault_copy:
        shutil.copytree(REAL_VAULT, vault)
        shutil.copytree(REAL_CHUNKS, chunks)
    else:
        os.makedirs(chunks, exist_ok=True)
        seed_c3.seed(vault)

    env = dict(os.environ)
    env["CLAUDE_CONFIG_DIR"] = cfg
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "32000"
    env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = chunks
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(cfg, "projects", _slug(wd))
    # caffeinate -is: hold a no-idle-sleep assertion for exactly the trial's lifetime (see docstring)
    args = ["caffeinate", "-is", "claude", "-p", RECALL_PREFIX + FIXTURE, "--output-format", "json",
            "--model", MODELS[model], "--permission-mode", "bypassPermissions"]

    out, retried, wall_s = {}, False, 0.0
    for attempt in range(MAX_ATTEMPTS):
        t0 = time.monotonic()
        proc = subprocess.run(args, cwd=wd, env=env, capture_output=True, text=True)
        wall_s = round(time.monotonic() - t0, 1)
        try:
            out = json.loads(proc.stdout)
        except json.JSONDecodeError:
            out = {}
        cost = out.get("total_cost_usd", 0) or 0
        if (out.get("is_error") or not out) and cost < 0.02:
            retried = attempt + 1 < MAX_ATTEMPTS
            continue
        break

    row = {
        "idx": idx, "ok": False, "retried": retried,
        "session_id": out.get("session_id"), "transcript_path": None,
        "recall_span_s": None, "total_duration_s": round((out.get("duration_ms") or 0) / 1000, 1),
        "wall_s": wall_s, "cost_usd": round(out.get("total_cost_usd", 0) or 0, 4),
        "turns": out.get("num_turns"), "is_error": bool(out.get("is_error")), "error": None,
    }
    if out.get("is_error") or not out:
        row["error"] = f"claude -p failed after {MAX_ATTEMPTS} attempts (is_error/empty envelope)"
        return row
    if not row["session_id"]:
        row["error"] = "envelope carried no session_id"
        return row
    tx = find_transcript(cfg, row["session_id"])
    row["transcript_path"] = tx
    if not tx:
        row["error"] = f"no {row['session_id']}.jsonl under {cfg}/projects"
        return row
    records = load_records(tx)
    span, shape = find_span(records)
    if span is None:
        row["error"] = "delimiters unresolved; transcript shape: " + json.dumps(shape)
        return row
    row.update(span)
    if row["recall_span_s"] > row["wall_s"] + SLEEP_TOLERANCE_S:
        row["error"] = (f"sleep-contaminated: span {row['recall_span_s']}s exceeds monotonic wall "
                        f"{row['wall_s']}s — the host slept mid-trial; not poolable")
        return row
    row["ok"] = True

    if segment:
        row["phases"], row["phase_error"] = compute_phases(records, span["start_ts"], span["end_ts"])
        query_calls = find_query_calls(records)
        row["census"], row["census_error"], row["census_source"], row["payload_path"] = None, None, None, None
        if not query_calls:
            row["census_error"] = "no `engram query` tool_use found — cannot capture payload"
        else:
            payload_text, note = capture_query_payload(query_calls, env, wd)
            if payload_text is None:
                row["census_error"] = note
            else:
                row["census_source"] = note
                payload_path = os.path.join(base, "query_payload.yaml")
                with open(payload_path, "w") as fh:
                    fh.write(payload_text)
                row["payload_path"] = payload_path
                try:
                    row["census"] = compute_census(payload_text)
                except (RuntimeError, ValueError) as exc:
                    row["census_error"] = str(exc)
    return row


def flag_degraded(rows):
    """Flag any OK trial whose cost is anomalously LOW vs its siblings (degraded-trial signature)."""
    ok = [r for r in rows if r["ok"]]
    for row in ok:
        siblings = [r["cost_usd"] for r in ok if r is not row]
        row["low_cost_flag"] = bool(
            siblings and row["cost_usd"] < DEGRADED_COST_RATIO * statistics.median(siblings))
    return [r["idx"] for r in ok if r.get("low_cost_flag")]


PHASE_FIELDS = ("pre_query_s", "query_call_s", "payload_consumption_s", "remainder_s")
CENSUS_FIELDS = ("total_bytes", "items_notes_content_bytes", "items_meta_bytes",
                  "clusters_candidate_content_bytes", "clusters_meta_bytes",
                  "recent_bytes", "duplicated_note_content_bytes")


def summarize_phases(rows):
    """Median + range per phase across gate-PASS trials only — a validity-gate failure is
    reported (gate_failed_idx) but never pooled, same discard-never-pool contract as the
    sleep gate."""
    poolable = [r for r in rows if r.get("ok") and r.get("phases") and r["phases"]["gate_ok"]]
    summary = {
        "n_ok": sum(1 for r in rows if r.get("ok")), "n_gate_pass": len(poolable),
        "gate_failed_idx": [r["idx"] for r in rows if r.get("phases") and not r["phases"]["gate_ok"]],
        "phase_error_idx": [r["idx"] for r in rows if r.get("ok") and r.get("phase_error")],
        "note": "medians/ranges pool ONLY phase-validity-gate-PASS trials; discard-never-pool",
    }
    for field in PHASE_FIELDS:
        vals = sorted(r["phases"][field] for r in poolable)
        summary[field] = {"median": statistics.median(vals) if vals else None,
                          "range": [vals[0], vals[-1]] if vals else None}
    return summary


def summarize_census(rows):
    """Median + range per census field across trials with a successfully measured census."""
    withcensus = [r for r in rows if r.get("census")]
    summary = {
        "n_ok": sum(1 for r in rows if r.get("ok")), "n_census": len(withcensus),
        "census_error_idx": [r["idx"] for r in rows if r.get("ok") and r.get("census_error")],
    }
    for field in CENSUS_FIELDS:
        vals = sorted(r["census"][field] for r in withcensus)
        summary[field] = {"median": statistics.median(vals) if vals else None,
                          "range": [vals[0], vals[-1]] if vals else None}
    return summary


def copy_durable_artifacts(rows, out_path):
    """Copy each trial's transcript (and, if segmented, its captured query payload) into a
    durable `<out>-transcripts/` dir next to the artifact — the sandbox paths recorded in
    `transcript_path`/`payload_path` get rmtree'd by the NEXT invocation's first trial."""
    transcripts_dir = os.path.splitext(out_path)[0] + "-transcripts"
    os.makedirs(transcripts_dir, exist_ok=True)
    for row in rows:
        tx = row.get("transcript_path")
        if tx and os.path.exists(tx):
            dest = os.path.join(transcripts_dir, f"trial-{row['idx']}.jsonl")
            shutil.copy2(tx, dest)
            row["transcript_copy_path"] = dest
        payload = row.get("payload_path")
        if payload and os.path.exists(payload):
            dest = os.path.join(transcripts_dir, f"trial-{row['idx']}-payload.yaml")
            shutil.copy2(payload, dest)
            row["payload_copy_path"] = dest
    return transcripts_dir


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--mode", choices=("pilot", "batch"), required=True)
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=3)
    ap.add_argument("--out", default=None, help="artifact JSON path (batch mode)")
    ap.add_argument("--vault-copy", action="store_true",
                    help="production-scale mode: sandbox copies of the real vault + chunk index")
    ap.add_argument("--segment", action="store_true",
                    help="#684: also compute per-trial phase segmentation + payload census")
    a = ap.parse_args()

    os.makedirs(ROOT, exist_ok=True)
    vault_desc = "real-vault copy" if a.vault_copy else "seed_c3 (5 notes)"

    if a.mode == "pilot":
        print(f"PILOT n=1 model={a.model} fixture=nocolor vault={vault_desc} root={ROOT}")
        row = run_trial("pilot", a.model, a.vault_copy, a.segment)
        print(json.dumps(row, indent=1))
        if not row["ok"]:
            print("\nPILOT FAILED — delimiters did not resolve (or the trial errored).")
            print("Do NOT proceed to the batch; escalate the method with the shape above.")
            sys.exit(1)
        print(f"\nSTART = {row['start_ts']}  (first timestamped record, type={row['start_record_type']})")
        print(f"END   = {row['end_ts']}  (rule={row['end_rule']}, match={row['end_match']!r})")
        print(f"SPAN  = {row['recall_span_s']} s  (trial total {row['total_duration_s']} s, "
              f"${row['cost_usd']:.2f}, turns={row['turns']})")
        if a.segment:
            print(f"PHASES = {row.get('phases')}  phase_error={row.get('phase_error')}")
            print(f"CENSUS = {row.get('census')}  census_error={row.get('census_error')} "
                  f"source={row.get('census_source')}")
        return

    if not a.out:
        ap.error("--out is required in batch mode")
    print(f"BATCH n={a.n} model={a.model} fixture=nocolor vault={vault_desc} root={ROOT} segment={a.segment}")
    rows = []
    for i in range(a.n):
        row = run_trial(i, a.model, a.vault_copy, a.segment)
        rows.append(row)
        print(f"  [trial {i}] ok={row['ok']} span={row['recall_span_s']}s "
              f"total={row['total_duration_s']}s ${row['cost_usd']:.2f} turns={row['turns']}"
              + (f" ERROR: {row['error']}" if row["error"] else "")
              + (f" phases={row['phases']}" if a.segment and row.get("phases") else "")
              + (f" PHASE-ERROR: {row['phase_error']}" if a.segment and row.get("phase_error") else "")
              + (f" census_error={row['census_error']}" if a.segment and row.get("census_error") else ""))
        if not row["ok"]:  # one full re-trial per slot (fresh sandbox); the failed row stays reported
            row = run_trial(f"{i}r", a.model, a.vault_copy, a.segment)
            rows.append(row)
            print(f"  [trial {i}r] (retry) ok={row['ok']} span={row['recall_span_s']}s "
                  f"total={row['total_duration_s']}s ${row['cost_usd']:.2f} turns={row['turns']}"
                  + (f" ERROR: {row['error']}" if row["error"] else ""))

    flagged = flag_degraded(rows)
    spans = sorted(r["recall_span_s"] for r in rows if r["ok"])
    summary = {
        "n_requested": a.n, "n_ok": len(spans),
        "median_recall_span_s": statistics.median(spans) if spans else None,
        "range_recall_span_s": [spans[0], spans[-1]] if spans else None,
        "total_cost_usd": round(sum(r["cost_usd"] for r in rows), 4),
        "low_cost_flagged_trials": flagged,
        "note": "n=3 directional by design; failed trials are reported, never pooled",
    }
    phase_summary = summarize_phases(rows) if a.segment else None
    census_summary = summarize_census(rows) if a.segment else None
    transcripts_dir = copy_durable_artifacts(rows, a.out) if a.out else None

    # Vault-copy rows append under their own key in an existing artifact; the seed_c3 rows and
    # the vault-copy rows are summarized separately and NEVER pooled.
    artifact = {}
    if os.path.exists(a.out):
        with open(a.out) as fh:
            artifact = json.load(fh)
    artifact.setdefault("meta", {}).update({
        "purpose": "#657 recall-only wall-time re-measure (post lazy-chunks/recent-fill/O2/L2)",
        "prior": "LEDGER recall-time-isolated: ~190 s, n=2 directional, pre-cuts vintage",
        "model": MODELS[a.model], "fixture": "nocolor trap prompt (traps.py) + RECALL_PREFIX",
        "delimiters": "START=first timestamped record; END=backward assistant scan for "
                      "'Query surfaced N items', fallback 'Re-entry:'",
        "date": datetime.date.today().isoformat(),
        "transcripts_dir": transcripts_dir,
    })
    if a.vault_copy:
        artifact["vault_copy"] = {"vault": vault_desc, "trials": rows, "summary": summary}
    else:
        artifact.update({"trials": rows, "summary": summary})
    if a.segment:
        segmented_block = {"trials": rows, "phase_summary": phase_summary, "census_summary": census_summary}
        artifact.setdefault("segmented", {})[
            "vault_copy" if a.vault_copy else "trials_fixture"] = segmented_block
    os.makedirs(os.path.dirname(a.out), exist_ok=True)
    with open(a.out, "w") as fh:
        json.dump(artifact, fh, indent=1)
    print(f"\nmedian={summary['median_recall_span_s']}s range={summary['range_recall_span_s']} "
          f"n_ok={summary['n_ok']}/{a.n} spend=${summary['total_cost_usd']:.2f} "
          f"low_cost_flags={flagged}")
    if a.segment:
        print(f"PHASE SUMMARY: {json.dumps(phase_summary, indent=1)}")
        print(f"CENSUS SUMMARY: {json.dumps(census_summary, indent=1)}")
    print(f"wrote {a.out}")


if __name__ == "__main__":
    main()
