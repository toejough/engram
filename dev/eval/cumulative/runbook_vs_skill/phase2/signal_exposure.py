#!/usr/bin/env python3
"""Signal-exposure report for the curate-signal task (fixtures/curate-signal): per valid trial in a
results .jsonl, did the agent (1) see engram's pending-offers signal in its first `engram query` result
(`pending_offers: true` + `pending_offers_hint`), (2) run the hint's command, (3) get the curate
procedure surfaced (arm R: runbook text in a tool result; arm S: Skill{curate} invoked), (4) write the
requested note, (5) curate the offers. (4)/(5) come from the end-state checker's two-half output.

Usage: signal_exposure.py <results.jsonl> [<results.jsonl> ...]   (prints a table + per-arm k/n)
"""
import json
import os
import re
import sys

import probe_phase2 as pp  # puts phase-1 probe.py on sys.path

p1 = pp.p1  # phase-1 transcript parsing

QUERY_RE = re.compile(r"\bengram\s+query\b")
HINT_CMD_RE = re.compile(r"pending\s+offers", re.IGNORECASE)
SAVED_RE = re.compile(r"saved to: (\S+)")


def _bash_cmd(ev):
    return (ev.get("input") or {}).get("command") or "" if ev.get("name") == "Bash" else ""


def exposure(events, arm, end_state_output):
    """saw_signal = any tool result carried engram's pending-offers signal (a query payload's
    `pending_offers: true`, or the write-path nudge `awaiting curation`); q_signal/q_hint = specifically
    the FIRST `engram query` result (the shim's mid-turn moment) had `pending_offers: true` /
    `pending_offers_hint`; nudge = a write-path warning was printed; ran_hint = the agent then ran an
    `engram query` naming pending offers (the hint's command)."""
    results = {e["tool_use_id"]: e.get("content") or "" for e in events if e["kind"] == "tool_result"}
    uses = [e for e in events if e["kind"] == "tool_use"]
    queries = [e for e in uses if QUERY_RE.search(_bash_cmd(e))]
    first = results.get(queries[0]["id"], "") if queries else ""
    all_results = list(results.values())
    # A big query payload is persisted to a file and the agent only sees a preview: read the file so
    # q_emitted/q_hint_emitted say what the BINARY printed, while q_visible says what the agent was shown.
    full = first
    m = SAVED_RE.search(first)
    if m and os.path.exists(m.group(1)):
        full = open(m.group(1), errors="replace").read()
    q_emitted = "pending_offers: true" in full
    q_hint_emitted = "pending_offers_hint" in full
    q_visible = "pending_offers: true" in first
    nudge = any("awaiting curation" in c for c in all_results)
    saw_signal = q_visible or nudge or any("pending_offers: true" in c for c in all_results)
    ran_hint = any(HINT_CMD_RE.search(_bash_cmd(q)) for q in queries)
    surfaced = False
    if arm == "S":
        surfaced = any(e.get("name") == "Skill" and "curate" in json.dumps(e.get("input")) for e in uses)
    elif arm == "R":
        surfaced = any("curate-review-pending-offers" in c for c in all_results)
    runbook_read = any("Judge Pending Offers" in c for c in all_results) or any(
        e.get("name") == "Skill" and "curate" in json.dumps(e.get("input")) for e in uses)
    out = end_state_output or ""
    return {
        "queried": bool(queries), "q_emitted": q_emitted, "q_hint_emitted": q_hint_emitted,
        "q_visible": q_visible, "nudge": nudge,
        "saw_signal": saw_signal, "ran_hint": ran_hint, "surfaced": surfaced, "runbook_read": runbook_read,
        "requested_note_done": "[requested-note: ok]" in out,
        "offers_curated": "[offers: ok]" in out,
    }


def main(paths):
    for path in paths:
        rows = [json.loads(line) for line in open(path) if line.strip()]
        print(f"== {path}")
        tally = {}
        for rec in rows:
            if not rec.get("valid"):
                print(f"  {rec.get('arm')}#{rec.get('trial')}: INVALID ({rec.get('invalid_reason')})")
                continue
            events = p1.parse_transcript_events([rec["transcript_path"]]) if rec.get("transcript_path") else []
            ex = exposure(events, rec["arm"], rec.get("end_state_output"))
            ex["end_state"] = bool(rec.get("end_state"))
            print(f"  {rec['arm']}#{rec['trial']}: " + " ".join(f"{k}={int(v)}" for k, v in ex.items())
                  + f" cost=${rec.get('total_cost_usd')}")
            t = tally.setdefault(rec["arm"], {"n": 0, "cost": 0.0})
            t["n"] += 1
            t["cost"] += rec.get("total_cost_usd") or 0
            for k, v in ex.items():
                t[k] = t.get(k, 0) + int(v)
        for arm, t in tally.items():
            print(f"  ARM {arm}: n={t['n']} " + " ".join(f"{k}={v}/{t['n']}" for k, v in t.items()
                                                       if k not in ("n", "cost")) + f" cost=${t['cost']:.2f}")


if __name__ == "__main__":
    main(sys.argv[1:])
