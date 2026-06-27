"""Tier-1 retrieval probe (free, local): run the real multi-phrase `engram query` — the exact
retrieval call `/recall` makes — and report, per load-bearing target, whether it surfaced and at
what rank. Crowding a vault can bury a target on cosine; this is the cheap signal that guides where
the expensive Tier-2 applied check spends.

Multi-target by design (C3 and C6 have several premise notes): a break fires if ANY target drops
(Gate-A finding A). C5 is omitted — its target surfaces by recency, not cosine, so Tier-1 is
invariant for it.
"""
import os
import re
import subprocess

# The phrases /recall would generate for each axis's task — Tier-1 runs the real multi-phrase
# `engram query` with these (10 --phrase each). C3/C4i are the cfgload/Go convention framings;
# C6 is derived from the two reasoning_recall_eval abduction-case task statements.
AXIS_PHRASES = {
    "C3": ["building a command-line app in Go", "making an HTTP request in Go", "wrapping errors in Go",
           "writing parallel Go tests", "terminal color output and NO_COLOR", "guarding a slice index in Go",
           "idiomatic Go error handling", "Go CLI architecture conventions", "Go context cancellation",
           "Go code quality standards"],
    "C4i": ["error marker convention in cfgload", "prefixing returned errors in Go", "cfgload codebase conventions",
            "error wrapping standard update", "ERR-CFG error prefix", "fmt.Errorf marker token",
            "exported function error format", "Go error return convention", "config loader error handling",
            "superseded error-marker convention"],
    # Derived from reasoning_recall_eval.CASES["abduction-diag"]["task"] (reactor symptoms) and
    # CASES["abduction-badge"]["task"] (building-access symptoms) — the open-ended framings /recall
    # reflects on to retrieve each case's two premise notes.
    "C6": ["zephyr-3 reactor behaving oddly", "what to investigate first in the zephyr-3 reactor",
           "diagnosing a reactor fault from its symptoms", "rising chamber pressure and an audible hiss",
           "falling coolant level in the reactor", "longer-tenured employees cannot get into the building",
           "some employees locked out of the building while others are fine",
           "lobby badge readers rejecting access badges", "RX-9 badge reader rejecting older badges",
           "building access failing for pre-2021 badges"],
}

# The load-bearing note slug(s) per axis. A target matches a payload item when its slug is a
# substring of an items[].path basename. Multi-target for C3 (the 5 convention notes) and C6 (the 4
# premise notes across both abduction cases). C5 omitted — Tier-1-invariant (recency, not cosine).
AXIS_TARGETS = {
    "C3": ["req-with-context", "nocolor", "t-parallel", "nil-guard-split", "wrapped-error"],
    "C4i": ["errcfg-supersedes-e7"],
    "C6": ["zephyr-leak-signature", "zephyr-current-state", "badge-reader-swap", "rx9-rejects-old"],
}

_PATH_LINE = re.compile(r"^  - path:\s*(.+?)\s*$")


def _parse_payload(text):
    """Extract items[].path in order from an `engram query` payload (YAML, occasionally JSON).

    The YAML payload renders each item's first field as `  - path: <basename>` at a 2-space indent;
    nested fields and the `content: |+` block scalar are at >=4-space indent, so a 2-space `- path:`
    line unambiguously marks an item (content lines never match)."""
    import json
    try:
        data = json.loads(text)
        if isinstance(data, dict) and "items" in data:
            return data
    except (ValueError, TypeError):
        pass
    items = []
    for line in text.splitlines():
        match = _PATH_LINE.match(line)
        if match:
            items.append({"path": match.group(1).strip().strip('"')})
    return {"items": items}


def rank_in_payload(payload, target_basename):
    """Pure: 1-based rank of the first item whose path basename CONTAINS target_basename.

    Chunk paths can carry a '#anchor' suffix — stripped before matching the basename."""
    items = payload.get("items", []) if isinstance(payload, dict) else []
    for rank, item in enumerate(items, start=1):
        path = item.get("path", "") if isinstance(item, dict) else ""
        basename = os.path.basename(str(path).split("#", 1)[0])
        if target_basename in basename:
            return {"surfaced": True, "rank": rank}
    return {"surfaced": False, "rank": None}


def probe(vault_path, axis):
    """Run the real multi-phrase `engram query` for an axis against vault_path and rank each target.

    Returns {"targets": {slug: {surfaced, rank}}, "all_surfaced": bool, "worst_rank": int|None}.
    worst_rank is the max rank when every target surfaced, else None (any absent target buries the
    axis). Raises on a non-zero query exit (fail loud)."""
    phrases = AXIS_PHRASES[axis]
    targets = AXIS_TARGETS[axis]
    cmd = ["engram", "query"]
    for phrase in phrases:
        cmd += ["--phrase", phrase]
    env = dict(os.environ)
    env["ENGRAM_VAULT_PATH"] = vault_path
    result = subprocess.run(cmd, env=env, capture_output=True, text=True)
    if result.returncode != 0:
        raise RuntimeError(
            f"engram query failed for axis {axis!r} (exit {result.returncode}): {result.stderr.strip()}")
    payload = _parse_payload(result.stdout)
    per_target = {target: rank_in_payload(payload, target) for target in targets}
    all_surfaced = bool(per_target) and all(r["surfaced"] for r in per_target.values())
    worst_rank = max(r["rank"] for r in per_target.values()) if all_surfaced else None
    return {"targets": per_target, "all_surfaced": all_surfaced, "worst_rank": worst_rank}
