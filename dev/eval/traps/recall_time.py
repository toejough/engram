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


def find_span(transcript_path):
    """Apply the pinned delimiter procedure. Returns (span_dict, None) or (None, shape_report)."""
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


def run_trial(idx, model, vault_copy=False):
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
    span, shape = find_span(tx)
    if span is None:
        row["error"] = "delimiters unresolved; transcript shape: " + json.dumps(shape)
        return row
    row.update(span)
    if row["recall_span_s"] > row["wall_s"] + SLEEP_TOLERANCE_S:
        row["error"] = (f"sleep-contaminated: span {row['recall_span_s']}s exceeds monotonic wall "
                        f"{row['wall_s']}s — the host slept mid-trial; not poolable")
        return row
    row["ok"] = True
    return row


def flag_degraded(rows):
    """Flag any OK trial whose cost is anomalously LOW vs its siblings (degraded-trial signature)."""
    ok = [r for r in rows if r["ok"]]
    for row in ok:
        siblings = [r["cost_usd"] for r in ok if r is not row]
        row["low_cost_flag"] = bool(
            siblings and row["cost_usd"] < DEGRADED_COST_RATIO * statistics.median(siblings))
    return [r["idx"] for r in ok if r.get("low_cost_flag")]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--mode", choices=("pilot", "batch"), required=True)
    ap.add_argument("--model", default="opus")
    ap.add_argument("--n", type=int, default=3)
    ap.add_argument("--out", default=None, help="artifact JSON path (batch mode)")
    ap.add_argument("--vault-copy", action="store_true",
                    help="production-scale mode: sandbox copies of the real vault + chunk index")
    a = ap.parse_args()

    os.makedirs(ROOT, exist_ok=True)
    vault_desc = "real-vault copy" if a.vault_copy else "seed_c3 (5 notes)"

    if a.mode == "pilot":
        print(f"PILOT n=1 model={a.model} fixture=nocolor vault={vault_desc} root={ROOT}")
        row = run_trial("pilot", a.model, a.vault_copy)
        print(json.dumps(row, indent=1))
        if not row["ok"]:
            print("\nPILOT FAILED — delimiters did not resolve (or the trial errored).")
            print("Do NOT proceed to the batch; escalate the method with the shape above.")
            sys.exit(1)
        print(f"\nSTART = {row['start_ts']}  (first timestamped record, type={row['start_record_type']})")
        print(f"END   = {row['end_ts']}  (rule={row['end_rule']}, match={row['end_match']!r})")
        print(f"SPAN  = {row['recall_span_s']} s  (trial total {row['total_duration_s']} s, "
              f"${row['cost_usd']:.2f}, turns={row['turns']})")
        return

    if not a.out:
        ap.error("--out is required in batch mode")
    print(f"BATCH n={a.n} model={a.model} fixture=nocolor vault={vault_desc} root={ROOT}")
    rows = []
    for i in range(a.n):
        row = run_trial(i, a.model, a.vault_copy)
        rows.append(row)
        print(f"  [trial {i}] ok={row['ok']} span={row['recall_span_s']}s "
              f"total={row['total_duration_s']}s ${row['cost_usd']:.2f} turns={row['turns']}"
              + (f" ERROR: {row['error']}" if row["error"] else ""))
        if not row["ok"]:  # one full re-trial per slot (fresh sandbox); the failed row stays reported
            row = run_trial(f"{i}r", a.model, a.vault_copy)
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
    })
    if a.vault_copy:
        artifact["vault_copy"] = {"vault": vault_desc, "trials": rows, "summary": summary}
    else:
        artifact.update({"trials": rows, "summary": summary})
    os.makedirs(os.path.dirname(a.out), exist_ok=True)
    with open(a.out, "w") as fh:
        json.dump(artifact, fh, indent=1)
    print(f"\nmedian={summary['median_recall_span_s']}s range={summary['range_recall_span_s']} "
          f"n_ok={summary['n_ok']}/{a.n} spend=${summary['total_cost_usd']:.2f} "
          f"low_cost_flags={flagged}")
    print(f"wrote {a.out}")


if __name__ == "__main__":
    main()
