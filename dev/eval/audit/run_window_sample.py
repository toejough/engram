#!/usr/bin/env python3
"""
Window-sampled re-measure driver (learn-rate-skill-only task 3.2, 2026-09-28).

Joe's option 1 (2026-09-28): audit ONE seeded window per sampled transcript
instead of every window, to hold the run near the pre-registered ~$35.

DEVIATION NOTE: the 2026-09 baseline audited every window of each sampled
transcript (its moments.jsonl has moments from several windows of one
transcript). It recorded no one-window selection rule, so the window here is
chosen uniformly: one random.Random(seed) stream consumed in sample order,
index = rng.randrange(total_windows).

Sampler-level only -- the instrument is untouched:
- corpus: run_audit.enumerate_corpus (named dir exclusions) -> filter_since ->
  filter_exclusions (named re-measure rules) -> sample_corpus(seed);
- per item: audit_moments.audit_transcript runs UNCHANGED, with its module-level
  split_into_windows temporarily bound to return only the chosen window, so the
  instrument's own detect(haiku) -> low-yield re-check(sonnet) -> judge(sonnet)
  loop runs on that window;
- moments are appended after every window (one window per transcript), then the
  per-transcript done-marker (resume-safe, run_audit.py's convention).

Recording wrappers (runtime only; prompts/results pass through unchanged):
- audit_moments._run_claude_p: appends each call's model + total_cost_usd to
  cost-log.jsonl (the instrument discards per-call costs);
- audit_moments._run_engram_query_at_moment: when the point-in-time existence
  query fails (items is None), re-runs the same `engram query` against the same
  isolated vault/chunks (local, no LLM) and logs rc/stdout/stderr to
  engram-query-failures.jsonl, for diagnosis.

Usage:
  python3 run_window_sample.py --output-dir <dir> [--since 2026-09-07]
      [--sample-size 47] [--seed 739] [--limit N]
"""

import argparse
import json
import os
import random
import subprocess
import sys
import time
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import audit_moments
import extract as extract_module
import run_audit

FROZEN_RESULTS_DIR = run_audit.FROZEN_RESULTS_DIR


def _now() -> str:
    return datetime.now(timezone.utc).isoformat()


def choose_windows(sample: List[str], seed: int, split=None) -> List[Dict[str, Any]]:
    """One uniform window per sampled transcript, from one seeded stream in sample order."""
    split = split or audit_moments.split_into_windows
    rng = random.Random(seed)
    items = []
    for path in sample:
        windows = split(path)
        index = rng.randrange(len(windows))
        chosen = windows[index]
        items.append(
            {
                "transcript_path": path,
                "total_windows": len(windows),
                "window_index": index,
                "location_start": chosen["location_start"],
                "location_end": chosen["location_end"],
            }
        )
    return items


def build_sample(output_dir: Path, since: Optional[str], sample_size: int, seed: int,
                 projects_root: Optional[str] = None) -> Dict[str, Any]:
    """Enumerate -> since -> exclusions -> sample -> choose windows; persist sample.json."""
    corpus = run_audit.enumerate_corpus(projects_root)
    enumerated = len(corpus)
    dropped_since = None
    if since:
        corpus, dropped_since = run_audit.filter_since(corpus, since)
    after_since = len(corpus)
    corpus, exclusion_counts = run_audit.filter_exclusions(corpus)
    if not corpus:
        raise ValueError("no transcripts remain after since + exclusions -- refusing to run")
    sample = run_audit.sample_corpus(corpus, sample_size, seed)
    record = {
        "metadata": {
            "generated_at": _now(),
            "seed": seed,
            "since": since,
            "sample_size_requested": sample_size,
            "sample_size": len(sample),
            "corpus_enumerated": enumerated,
            "dropped_by_since": dropped_since,
            "corpus_after_since": after_since,
            "excluded_by_rule": exclusion_counts,
            "corpus_eligible": len(corpus),
            "excluded_project_dir_prefixes": list(run_audit.EXCLUDED_PROJECT_DIR_PREFIXES),
            "excluded_project_dir_substrings": list(run_audit.EXCLUDED_PROJECT_DIR_SUBSTRINGS),
            "excluded_session_ids": list(run_audit.EXCLUDED_SESSION_IDS),
            "window_rule": "uniform: random.Random(seed).randrange(total_windows), in sample order",
            "instrument_version": audit_moments.INSTRUMENT_VERSION,
        },
        "items": choose_windows(sample, seed),
    }
    with open(output_dir / "sample.json", "w") as f:
        json.dump(record, f, indent=2)
    return record


def install_recorders(output_dir: Path, current: Dict[str, Any]) -> None:
    """Runtime recording wrappers; see module docstring."""
    cost_log = output_dir / "cost-log.jsonl"
    fail_log = output_dir / "engram-query-failures.jsonl"
    orig_claude = audit_moments._run_claude_p
    orig_query = audit_moments._run_engram_query_at_moment

    def claude_wrapper(prompt, model, *args, **kwargs):
        out = orig_claude(prompt, model, *args, **kwargs)
        with open(cost_log, "a") as f:
            f.write(json.dumps({
                "t": time.time(), "transcript_path": current.get("path"), "model": model,
                "prompt_chars": len(prompt), "cost_usd": out.get("total_cost_usd"),
                "exhausted_retries": out.get("exhausted_retries"),
                "prompt_too_long": out.get("prompt_too_long"),
            }) + "\n")
        return out

    def query_wrapper(vault_path, chunks_path, search_phrases, *args, **kwargs):
        out = orig_query(vault_path, chunks_path, search_phrases, *args, **kwargs)
        if out.get("items") is None:
            diag: Dict[str, Any] = {"t": time.time(), "transcript_path": current.get("path"),
                                    "phrases": list(search_phrases), "instrument_result": out}
            try:
                import isolation
                env = isolation.engram_env(vault=vault_path, chunks=chunks_path)
                # Shared with the primary invocation this rerun is diagnosing, so the
                # two can never drift apart again the way they did for #787.
                args_q = audit_moments._build_engram_query_argv(search_phrases)
                rerun = subprocess.run(args_q, env=env, capture_output=True, text=True, timeout=60)
                diag.update({"rerun_rc": rerun.returncode, "rerun_stdout_tail": rerun.stdout[-2000:],
                             "rerun_stderr_tail": rerun.stderr[-2000:]})
            except Exception as e:  # diagnosis must never break the run
                diag["rerun_error"] = repr(e)
            with open(fail_log, "a") as f:
                f.write(json.dumps(diag) + "\n")
        return out

    audit_moments._run_claude_p = claude_wrapper
    audit_moments._run_engram_query_at_moment = query_wrapper


def projected_total(prior_spend: float, run_spend: float, windows_done: int,
                    windows_total: int) -> float:
    """All-in projected spend: prior spend + this run's mean per-window cost x all windows."""
    if windows_done <= 0:
        return prior_spend
    return prior_spend + run_spend / windows_done * windows_total


def audit_one_window(item: Dict[str, Any]) -> List[Dict[str, Any]]:
    """Run the unchanged instrument on the chosen window only (fails loud on drift)."""
    windows = audit_moments.split_into_windows(item["transcript_path"])
    chosen = windows[item["window_index"]] if item["window_index"] < len(windows) else None
    if chosen is None or (chosen["location_start"], chosen["location_end"]) != (
        item["location_start"], item["location_end"]
    ):
        raise RuntimeError(
            f"window drift for {item['transcript_path']}: recorded "
            f"{item['location_start']}-{item['location_end']} of {item['total_windows']}, "
            f"now {len(windows)} windows"
        )
    original = audit_moments.split_into_windows
    audit_moments.split_into_windows = lambda path, *a, **k: [chosen]
    try:
        return audit_moments.audit_transcript(item["transcript_path"])
    finally:
        audit_moments.split_into_windows = original


def main(argv=None) -> int:
    ap = argparse.ArgumentParser(description=__doc__.split("\n")[1])
    ap.add_argument("--output-dir", required=True)
    ap.add_argument("--since", default=None)
    ap.add_argument("--sample-size", type=int, default=47)
    ap.add_argument("--seed", type=int, default=739)
    ap.add_argument("--limit", type=int, default=None, help="process at most N pending items")
    ap.add_argument("--prior-spend-usd", type=float, default=0.0,
                    help="spend already incurred outside this output dir (counts toward the cap)")
    ap.add_argument("--cost-cap-usd", type=float, default=None,
                    help="stop cleanly between windows once the projected all-in total exceeds this")
    args = ap.parse_args(argv)

    output_dir = Path(args.output_dir).resolve()
    frozen = FROZEN_RESULTS_DIR.resolve()
    if output_dir == frozen or frozen in output_dir.parents:
        raise SystemExit(f"refusing to write into the frozen baseline {frozen}")
    output_dir.mkdir(parents=True, exist_ok=True)

    sample_path = output_dir / "sample.json"
    if sample_path.exists():
        sample = json.load(open(sample_path))
        print(f"[window_sample] reusing {sample_path}", flush=True)
    else:
        sample = build_sample(output_dir, args.since, args.sample_size, args.seed)
    items = sample["items"]

    events_path = output_dir / "transcript-events.jsonl"
    moments_path = output_dir / "moments.jsonl"
    done = run_audit.load_done_transcripts(events_path)
    current: Dict[str, Any] = {}
    install_recorders(output_dir, current)

    status, processed = "completed", 0
    try:
        for n, item in enumerate(items, start=1):
            path = item["transcript_path"]
            if path in done:
                continue
            if args.limit is not None and processed >= args.limit:
                status = "stopped_at_limit"
                break
            current["path"] = path
            print(f"[window_sample] [{n}/{len(items)}] window {item['window_index'] + 1}/"
                  f"{item['total_windows']} of {path}", flush=True)
            events = extract_module.extract_transcript(path)
            events["window_sample"] = {k: item[k] for k in
                                       ("window_index", "total_windows", "location_start", "location_end")}
            moments: List[Dict[str, Any]] = []
            try:
                moments = audit_one_window(item)
            except audit_moments.PromptTooLongError as e:
                events["skipped_reason"] = f"prompt_too_long: {e}"
            if moments:
                with open(moments_path, "a") as f:
                    for m in moments:
                        f.write(json.dumps(m) + "\n")
            with open(events_path, "a") as f:
                f.write(json.dumps(events) + "\n")
            done.add(path)
            processed += 1
            if args.cost_cap_usd is not None and processed >= 3:
                spent = sum((json.loads(l).get("cost_usd") or 0)
                            for l in open(output_dir / "cost-log.jsonl") if l.strip())
                proj = projected_total(args.prior_spend_usd, spent, len(done), len(items))
                print(f"[window_sample] spent ${spent:.2f} over {len(done)} windows; "
                      f"projected all-in ${proj:.2f} (cap ${args.cost_cap_usd:.2f})", flush=True)
                if proj > args.cost_cap_usd:
                    status = "stopped_projected_over_cap"
                    break
    except audit_moments.ExhaustedRetriesError as e:
        status = "halted_exhausted_retries"
        sys.stderr.write(f"[window_sample] HALT: {e}\n")

    cost_total = 0.0
    cost_log = output_dir / "cost-log.jsonl"
    if cost_log.exists():
        cost_total = sum((json.loads(l).get("cost_usd") or 0) for l in open(cost_log) if l.strip())
    manifest = {
        "status": status,
        "finished_at": _now(),
        "instrument_version": audit_moments.INSTRUMENT_VERSION,
        "model_tiers": dict(run_audit.INSTRUMENT_MODEL_TIERS),
        "items_total": len(items),
        "items_done": len(done),
        "cost_usd_recorded": round(cost_total, 4),
        "cost_method": "sum of total_cost_usd reported by each claude -p call (cost-log.jsonl)",
        "args": vars(args),
    }
    with open(output_dir / "run-manifest.json", "w") as f:
        json.dump(manifest, f, indent=2)
    print(json.dumps(manifest, indent=2))
    return 2 if status == "halted_exhausted_retries" else 0


if __name__ == "__main__":
    sys.exit(main())
