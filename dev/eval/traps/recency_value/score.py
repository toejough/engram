"""Deterministic scorers for the recency-value harness (#646).

Pure functions:
  import_ok            - phase-1 artifact check: every orders.db.json record's amt equals
                         round(dollars*1000) for its orders.csv row (the milli-dollar convention).
  report_revenue_ok    - phase-2 artifact check: parses `total revenue: $X` from the built CLI's
                         own stdout and compares to the known dollar total.
  surfaced_any         - did phase-1's milli-dollar narration surface ANYWHERE in a captured
                         `engram query` payload (recency channel OR the matched set)? Serves the
                         P2 vacuous-contrast gate: catches BOTH the recent-fill channel and the
                         re-rank bias leaking the chunk into the matched set as `direct`.
  surfaced_via_recency - the note-83 diagnostic: did the RECENCY CHANNEL specifically deliver it —
                         an item whose provenances contains "recent" AND whose content mentions
                         the milli-dollar unit? Mirrors recency_probe.parse_recent_channel's
                         provenance-filtering semantics (items whose provenances include "recent").
  recall_fired         - did the phase-2 transcript record an `engram query` invocation? Mirrors
                         wrun.py:65-73's transcript grep.
  phase1_used_learn    - P1 pilot check: did phase-1 fire `engram learn` (would write a vault note
                         that surfaces via cosine in BOTH arms, confounding the contrast)?

The surfacing parsers are self-contained and block-scalar-robust: real `engram query` renders
chunk content as a YAML block scalar (`content: |-` ... indented lines), which
recency_probe._parse_yaml_items mangles (it only captures inline `key: value`). recency_probe's
provenance detection is faithful for inline payloads, but content lives in a block scalar, so we
split the items[] section into per-item raw-text blocks and match markers against the raw block
text while reading provenances structurally.
"""
import csv
import json
import re

REVENUE_RE = re.compile(r"total revenue:\s*\$\s*(-?[0-9]+(?:\.[0-9]+)?)")

# Markers a phase-1 narration of the milli-dollar convention plausibly uses. Broad enough to
# catch the agent's own wording (not scripted), narrow enough not to false-positive on
# unrelated "1000"/"dollar" mentions.
MILLI_DOLLAR_MARKERS = (
    "milli-dollar",
    "milli dollar",
    "millidollar",
    "tenths-of-a-cent",
    "tenths of a cent",
    "dollars*1000",
    "dollars * 1000",
    "dollars * 1_000",
    "* 1000",
    "*1000",
)


def import_ok(db_path, csv_path):
    """True iff every orders.db.json record's amt equals round(dollars*1000) for its CSV row."""
    with open(db_path) as f:
        records = json.load(f)

    dollars = {}
    with open(csv_path, newline="") as f:
        for row in csv.DictReader(f):
            dollars[row["id"]] = float(row["dollars"])

    if len(records) != len(dollars):
        return False

    for rec in records:
        row_id = str(rec.get("id"))
        if row_id not in dollars:
            return False

        amt = rec.get("amt")
        if not isinstance(amt, int) or isinstance(amt, bool):
            return False

        if amt != round(dollars[row_id] * 1000):
            return False

    return True


def report_revenue_ok(stdout, expected_dollar_total):
    """True iff stdout's 'total revenue: $X' is within half a cent of expected_dollar_total."""
    match = REVENUE_RE.search(stdout)
    if not match:
        return False

    return abs(float(match.group(1)) - expected_dollar_total) < 0.005


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


def _has_milli_marker(text):
    lowered = text.lower()
    return any(marker in lowered for marker in MILLI_DOLLAR_MARKERS)


def _recent_block_has(payload_yaml, markers):
    """True iff a recency-channel item (provenances includes "recent") has raw block text
    matching any marker. Generalized core of surfaced_via_recency (parametrized on markers so it
    can be validated against real golden payloads whose content differs)."""
    lowered = [m.lower() for m in markers]
    for block in _item_blocks(payload_yaml):
        if "recent" in _block_provenances(block) and any(m in block.lower() for m in lowered):
            return True

    return False


def surfaced_any(payload_yaml):
    """True iff the milli-dollar unit appears in ANY item (recency channel OR matched set). The
    P2 vacuous-contrast gate: catches the re-rank bias (provenanceDirect) leaking the chunk, not
    just the recent-fill channel."""
    return any(_has_milli_marker(block) for block in _item_blocks(payload_yaml))


def surfaced_via_recency(payload_yaml):
    """Note-83 diagnostic: True iff the milli-dollar unit was delivered by the RECENCY CHANNEL
    specifically (an item whose provenances contains "recent")."""
    return _recent_block_has(payload_yaml, MILLI_DOLLAR_MARKERS)


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
