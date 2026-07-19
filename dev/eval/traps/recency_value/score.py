"""Deterministic scorers for the recency-value harness (#646, cross-context org-RUNLOG scenario).

The scenario measures whether engram's recency channel resurfaces phase-1's discovery of an
idiosyncratic ORG-WIDE convention after context loss. Phase 1 (`notes add`) and phase 2 (`orders
report`) are commands in DIFFERENT tools, so phase-1's narration is topically DISTANT from
phase-2's task query — the recency channel is the carrier, not cosine.

Org convention D: EVERY command in EVERY tool at this org, on success, appends one line to its
./RUNLOG of the exact form `RUNLOG v1 :: <CMD> :: rows=<N> :: sig=QX7Z` (CMD uppercased, N =
records processed, sig a fixed idiosyncratic signature). Stated in no SPEC — established by phase
1's check.sh (in the `notes` tool) and must be RECALLED and applied by phase 2 (in the `orders`
tool, a DIFFERENT tool — so phase-1's transcript is topically distant from phase-2's task query).

Pure functions:
  notes_add_ok         - phase-1 capture gate: did ./RUNLOG in the phase-1 workspace get a valid
                         ADD line (i.e. phase 1 discovered the org convention)?
  report_revenue_ok    - phase-2 artifact check: parses `total revenue: $X` from report's stdout
                         (revenue is trivial here — plain-dollar amt — so RUNLOG is the lever).
  runlog_report_ok     - PRIMARY correctness: did phase-2's report append a valid REPORT line?

  Surfacing (note-197 instrument-fidelity fix): measured from the agent's ACTUAL in-band recall
  payload in the phase-2 transcript, NOT a post-hoc out-of-band re-query. The out-of-band probe
  ran AFTER phase 2, by which point the recall skill's own `engram ingest --auto` had polluted the
  per-trial chunk index (sweeping ancestor .claude dirs + ws2), displacing phase-1's chunk from
  the recency channel; it also used canned phrases, not the agent's real multi-phrase recall.
  The recall skill queries with --lazy-chunks, which ZEROES chunk content, so matching is on
  SOURCE/PATH (the phase-1 transcript's session id), not content markers.

  extract_query_payloads       - pull every engram query YAML payload out of a phase-2 transcript.
  surfaced_any_inband          - did ANY recalled item (any provenance) reference phase-1's chunk?
  surfaced_via_recency_inband  - the note-83 diagnostic: did an item with `recent` provenance
                                 reference phase-1's chunk (recency channel specifically)?
  recall_fired                 - did the phase-2 transcript record an `engram query` invocation?
  phase1_used_learn            - P1 pilot check: did phase-1 fire `engram learn` (would write a
                                 vault note surfacing via cosine in BOTH arms, confounding)?

The payload parsers are self-contained and block-scalar-robust (real `engram query` renders chunk
content as a YAML block scalar `content: |-` that a naive line parser mangles): the items[]
section is split into per-item raw-text blocks; path + provenances are read structurally.
"""
import json
import os
import re

REVENUE_RE = re.compile(r"total revenue:\s*\$\s*(-?[0-9]+(?:\.[0-9]+)?)")

# The idiosyncratic org RUNLOG audit line. runlog_report_ok / notes_add_ok match a per-CMD line.
RUNLOG_LINE_RE = re.compile(r"^RUNLOG v1 :: (?P<cmd>[A-Z]+) :: rows=\d+ :: sig=QX7Z$", re.MULTILINE)

# An engram query YAML payload has an items: block plus at least one of these sibling keys.
QUERY_PAYLOAD_ITEMS_RE = re.compile(r"(?m)^items:")


def _runlog_has_cmd(runlog_text, cmd):
    """True iff runlog_text has a valid house-convention line for the given uppercased CMD."""
    return any(match.group("cmd") == cmd for match in RUNLOG_LINE_RE.finditer(runlog_text))


def notes_add_ok(ws):
    """Phase-1 capture gate: True iff <ws>/RUNLOG has a valid ADD line (phase 1's `notes add`
    discovered and honored the ORG-WIDE RUNLOG convention)."""
    path = os.path.join(ws, "RUNLOG")
    if not os.path.exists(path):
        return False

    with open(path, errors="ignore") as f:
        return _runlog_has_cmd(f.read(), "ADD")


def report_revenue_ok(stdout, expected_dollar_total):
    """True iff stdout's 'total revenue: $X' is within half a cent of expected_dollar_total."""
    match = REVENUE_RE.search(stdout)
    if not match:
        return False

    return abs(float(match.group(1)) - expected_dollar_total) < 0.005


def runlog_report_ok(runlog_path):
    """PRIMARY correctness: True iff RUNLOG has a valid REPORT line (phase-2 recalled + applied the
    house convention to the report command)."""
    if not os.path.exists(runlog_path):
        return False

    with open(runlog_path, errors="ignore") as f:
        return _runlog_has_cmd(f.read(), "REPORT")


# --- in-band recall payload extraction + surfacing detection -------------------------------

def _tool_result_text(content):
    """Normalize a transcript tool_result's content (str, or a list of text blocks) to text."""
    if isinstance(content, str):
        return content

    if isinstance(content, list):
        parts = []
        for block in content:
            if isinstance(block, dict) and isinstance(block.get("text"), str):
                parts.append(block["text"])
            elif isinstance(block, str):
                parts.append(block)
        return "\n".join(parts)

    return ""


def _looks_like_query_payload(text):
    """Heuristic for an `engram query` YAML payload. The plan suggested `items:` + `budget:`, but
    real transcripts (verified) have no `budget:` line — the recall skill's payloads carry items:
    plus phrases:/provenances:. Require an items: block AND one of those sibling keys."""
    if not QUERY_PAYLOAD_ITEMS_RE.search(text):
        return False

    return any(key in text for key in ("phrases:", "provenances:", "budget:"))


def extract_query_payloads(transcript_path):
    """Parse a phase-2 transcript JSONL; return every engram query YAML payload found in a
    tool_result (there may be several — the multi-phrase recall + re-entry queries)."""
    payloads = []
    if not transcript_path or not os.path.exists(transcript_path):
        return payloads

    with open(transcript_path, errors="ignore") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue

            try:
                obj = json.loads(line)
            except (json.JSONDecodeError, ValueError):
                continue

            message = obj.get("message") or {}
            content = message.get("content")
            if not isinstance(content, list):
                continue

            for block in content:
                if isinstance(block, dict) and block.get("type") == "tool_result":
                    text = _tool_result_text(block.get("content"))
                    if _looks_like_query_payload(text):
                        payloads.append(text)

    return payloads


def _item_blocks(payload_yaml):
    """Split a query payload's top-level items: section into per-item raw-text blocks. Robust to
    YAML block-scalar content (`content: |-` ...): each item starts at a 2-space `  - ` entry and
    runs until the next such entry or the end of the items: block (the next indent-0 key)."""
    section = []
    in_items = False
    for line in payload_yaml.splitlines():
        if not in_items:
            if line.startswith("items:"):
                in_items = True
            continue
        # A new indent-0 (non-space, non-blank) key ends the items section.
        if line[:1] not in (" ", "\t", "") and line.strip():
            break
        section.append(line)

    blocks = []
    current = None
    for line in section:
        # Only a true item entry has `- ` at exactly 2-space indent; deeper list entries
        # (provenance roles at indent 6) do not match "  - ".
        if line.startswith("  - "):
            if current is not None:
                blocks.append("\n".join(current))
            current = [line]
        elif current is not None:
            current.append(line)

    if current is not None:
        blocks.append("\n".join(current))

    return blocks


def _block_path(block):
    """Extract an item block's `path` value (the item entry's first line, `  - path: <path>`)."""
    for line in block.splitlines():
        stripped = line.strip()
        for prefix in ("- path:", "path:"):
            if stripped.startswith(prefix):
                return stripped[len(prefix):].strip().strip('"').strip("'")

    return ""


def _block_provenances(block):
    """Structurally extract an item block's provenance roles (robust to block-scalar content
    that follows the provenances list)."""
    roles = []
    in_provs = False
    for line in block.splitlines():
        stripped = line.strip()
        indent = len(line) - len(line.lstrip(" "))
        if not stripped:
            continue

        if stripped.startswith("provenances:"):
            _, _, rest = stripped.partition(":")
            rest = rest.strip()
            if rest.startswith("[") and rest.endswith("]"):
                roles += [x.strip().strip('"').strip("'") for x in rest[1:-1].split(",") if x.strip()]
            else:
                in_provs = True
            continue

        if in_provs:
            if stripped.startswith("- "):
                roles.append(stripped[2:].strip().strip('"').strip("'"))
            elif indent <= 4:
                # dedented back to a sibling field (kind/score/content) — list ended
                in_provs = False

    return roles


def _phase1_session_id(phase1_transcript_path):
    """The phase-1 session id (transcript basename without .jsonl). Pre-phase-2 the only ingested
    source IS phase-1's transcript, so a chunk path containing this id is phase-1's chunk."""
    if not phase1_transcript_path:
        return None

    base = os.path.basename(phase1_transcript_path)
    return base[:-len(".jsonl")] if base.endswith(".jsonl") else base


def _iter_phase1_blocks(payloads, phase1_transcript_path):
    """Yield item blocks (across all payloads) whose path references phase-1's chunk."""
    session_id = _phase1_session_id(phase1_transcript_path)
    if not session_id:
        return

    for payload in payloads:
        for block in _item_blocks(payload):
            if session_id in _block_path(block):
                yield block


def surfaced_any_inband(payloads, phase1_transcript_path):
    """True iff the agent's real recall surfaced phase-1's chunk at all (any provenance). Path-
    based (robust to --lazy-chunks zeroing content)."""
    return any(True for _ in _iter_phase1_blocks(payloads, phase1_transcript_path))


def surfaced_via_recency_inband(payloads, phase1_transcript_path):
    """Note-83 diagnostic: True iff phase-1's chunk was delivered by the RECENCY CHANNEL
    specifically — an item referencing phase-1 whose provenances contains "recent"."""
    for block in _iter_phase1_blocks(payloads, phase1_transcript_path):
        if "recent" in _block_provenances(block):
            return True

    return False


def recall_fired(transcript_path):
    """True iff the transcript contains an engram query invocation (wrun.py:65-73's grep)."""
    with open(transcript_path, errors="ignore") as f:
        text = f.read()

    return "engram query" in text or '"recall"' in text


def phase1_used_learn(transcript_path):
    """P1 pilot check: True iff phase-1's transcript recorded an `engram learn` invocation. Phase 1
    runs with NO engram skills to prevent /learn firing; this is the defense-in-depth detector —
    a note written here would surface via cosine in BOTH arms and confound the contrast."""
    with open(transcript_path, errors="ignore") as f:
        return "engram learn" in f.read()
