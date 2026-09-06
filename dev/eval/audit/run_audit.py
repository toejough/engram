#!/usr/bin/env python3
"""
Thin pipeline runner for the memory-loop audit: ONE entry point wrapping the
existing stages (extract -> audit -> scorecard) for a future before/after
re-run. Orchestration only -- no new measurement logic lives here.

Modes:
  python3 run_audit.py --estimate --config <cfg.json>
      Print a cost estimate WITHOUT any API call. The basis is the real
      per-moment-call costs this audit recorded (cost_usd fields in
      results/rejudge-all-nulls.jsonl and results/relevance-cues.jsonl),
      extrapolated to the configured sample size via the frozen baseline's
      measured moments-per-transcript rate.

  python3 run_audit.py --config <cfg.json>
      Run the pipeline: enumerate the live corpus fresh (corpus.json is
      documented stale -- never trusted), sample per config seed, run
      extract.extract_transcript + audit_moments.audit_transcript per
      transcript (real instrument), then the scorecard stage, then write
      output_dir/run-manifest.json.

Resume: per-transcript output is appended incrementally (transcript-events
.jsonl doubles as the done-marker, the same convention as
measure_relevance_cues.py's resume block); on restart, transcripts already
recorded are skipped and the persisted sample.json is reused (the live corpus
may have changed between invocations, so re-enumerating would break resume).

Halt: audit_moments.ExhaustedRetriesError is an immediate halt (manifest gets
status=halted_exhausted_retries, then the error propagates; exit code 2 from
the CLI). PromptTooLongError is deterministic and per-transcript: recorded as
a skip on that transcript's done-marker, run continues (the established
driver convention).

Frozen baseline: /dev/eval/audit/results/ is the frozen 2026-09 baseline.
The runner refuses any output_dir at or under it. build_scorecard.py pins its
SRC/OUT absolute paths into that frozen dir (DO-NOT-TOUCH file), so for any
re-run output dir the scorecard stage is a LOUD skip (recorded in the
manifest), never a silent recompute of stale inputs.
"""

import argparse
import json
import os
import random
import re
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Dict, List, Optional, Set

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import audit_moments
import extract as extract_module

AUDIT_DIR = Path(__file__).resolve().parent
FROZEN_RESULTS_DIR = AUDIT_DIR / "results"
BUILD_SCORECARD_PATH = AUDIT_DIR / "build_scorecard.py"

DEFAULT_PROJECTS_ROOT = os.path.expanduser("~/.claude/projects")

# audit_moments.audit_transcript hardcodes these internally (haiku detection,
# sonnet judgment). The config carries them for the manifest record, but any
# OTHER value must fail loud -- the instrument would not honor it.
INSTRUMENT_MODEL_TIERS = {"detect": "haiku", "judge": "sonnet"}

VALID_STAGES = ("extract", "audit", "scorecard")

DEFAULT_CONFIG: Dict[str, Any] = {
    "sample_size": 47,
    "stages": list(VALID_STAGES),
    "output_dir": "results-rerun",
    "model_tiers": dict(INSTRUMENT_MODEL_TIERS),
    "seed": 739,
}

# The two files holding the real per-moment-call costs this audit recorded
# (each record's cost_usd is the total_cost_usd claude -p itself reported).
COST_BASIS_FILES = ("rejudge-all-nulls.jsonl", "relevance-cues.jsonl")

# The frozen baseline's LLM-audited transcript sets (the extract-only
# 568-transcript sweep in transcript-events.jsonl is deliberately NOT
# included -- moments-per-transcript must be derived from transcripts the
# moment auditor actually ran on).
BASELINE_EVENTS_FILES = (
    "pilot-transcript-events.jsonl",
    "full-run-transcript-events.jsonl",
    "refill-transcript-events.jsonl",
)
BASELINE_MOMENTS_FILE = "moments.jsonl"


# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------


def load_config(path: str) -> Dict[str, Any]:
    """
    Read a JSON config file, merge over DEFAULT_CONFIG, validate, and resolve
    output_dir to an absolute path (relative paths resolve against AUDIT_DIR).

    Fails loud on: missing file, unknown keys, invalid stages, model_tiers
    differing from the instrument's pinned tiers, and any output_dir at or
    under the frozen results/ baseline.
    """
    with open(path) as f:
        raw = json.load(f)

    unknown = set(raw) - set(DEFAULT_CONFIG)
    if unknown:
        raise ValueError(
            f"unknown config key(s): {sorted(unknown)} "
            f"(valid: {sorted(DEFAULT_CONFIG)})"
        )

    cfg = {**DEFAULT_CONFIG, **raw}
    # never share mutable defaults with callers
    cfg["stages"] = list(cfg["stages"])
    cfg["model_tiers"] = dict(cfg["model_tiers"])

    if not isinstance(cfg["sample_size"], int) or cfg["sample_size"] <= 0:
        raise ValueError(f"sample_size must be a positive int, got {cfg['sample_size']!r}")
    if not isinstance(cfg["seed"], int):
        raise ValueError(f"seed must be an int, got {cfg['seed']!r}")
    if not cfg["stages"]:
        raise ValueError("stages must be a non-empty list")
    for stage in cfg["stages"]:
        if stage not in VALID_STAGES:
            raise ValueError(f"invalid stage {stage!r} (valid: {list(VALID_STAGES)})")
    if cfg["model_tiers"] != INSTRUMENT_MODEL_TIERS:
        raise ValueError(
            f"model_tiers {cfg['model_tiers']} differ from the instrument's pinned "
            f"tiers {INSTRUMENT_MODEL_TIERS}: audit_moments.audit_transcript "
            "hardcodes haiku detection / sonnet judgment internally, so any other "
            "config would silently run with tiers the instrument won't use"
        )

    out_dir = Path(os.path.expanduser(str(cfg["output_dir"])))
    if not out_dir.is_absolute():
        out_dir = AUDIT_DIR / out_dir
    resolved = out_dir.resolve()
    frozen = FROZEN_RESULTS_DIR.resolve()
    if resolved == frozen or frozen in resolved.parents:
        raise ValueError(
            f"output_dir {resolved} is the frozen 2026-09 baseline "
            f"({frozen}) or inside it -- refusing to overwrite it"
        )
    cfg["output_dir"] = str(out_dir)

    return cfg


# ---------------------------------------------------------------------------
# Corpus enumeration + sampling
# ---------------------------------------------------------------------------


def enumerate_corpus(projects_root: Optional[str] = None) -> List[str]:
    """
    Enumerate the live transcript corpus fresh from the ~/.claude/projects
    layout (corpus.json is documented stale and never read here).

    Conventions (the same ones extract.py's parsing implies and corpus.json's
    inventory used):
    - main sessions:      <projects_root>/<project>/<session-id>.jsonl
    - subagents (fork/fresh): <project>/<session-id>/subagents/agent-*.jsonl
    - workflows:          <project>/<session-id>/subagents/workflows/**.jsonl
    - .meta.json sidecars are not transcripts (not matched -- *.jsonl only)
    - project dirs named -private-tmp* are ephemeral eval dirs (corpus.json's
      excluded_directories rationale) and are skipped

    Returns a sorted list of absolute paths (sorted for deterministic
    sampling regardless of filesystem order).
    """
    root = Path(projects_root or DEFAULT_PROJECTS_ROOT)
    paths: List[str] = []
    if not root.is_dir():
        return paths
    for project_dir in root.iterdir():
        if not project_dir.is_dir():
            continue
        if project_dir.name.startswith("-private-tmp"):
            continue
        for transcript in project_dir.rglob("*.jsonl"):
            if transcript.is_file():
                paths.append(str(transcript))
    return sorted(paths)


def sample_corpus(paths: List[str], sample_size: int, seed: int) -> List[str]:
    """
    Deterministically sample sample_size transcripts from the corpus under the
    given seed. Input order never matters (sorted first); a sample_size larger
    than the corpus takes the whole corpus.
    """
    ordered = sorted(paths)
    if sample_size >= len(ordered):
        return ordered
    return random.Random(seed).sample(ordered, sample_size)


# ---------------------------------------------------------------------------
# Estimate mode (no API calls, ever)
# ---------------------------------------------------------------------------


def read_cost_records(path: Path) -> List[float]:
    """Read the non-null cost_usd fields from a JSONL results file."""
    costs: List[float] = []
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            record = json.loads(line)
            cost = record.get("cost_usd")
            if cost is not None:
                costs.append(float(cost))
    return costs


def compute_estimate(
    sample_size: int,
    cost_records_by_file: Dict[str, List[float]],
    baseline_moments: int,
    baseline_transcripts: int,
) -> Dict[str, Any]:
    """
    Pure estimate arithmetic from real recorded numbers:
    - per-file and pooled mean cost per moment-level call (cost_usd fields)
    - moments-per-transcript from the frozen baseline's audited run
    - extrapolate to the configured sample size

    Raises on an empty cost basis (never silently estimate $0).
    """
    total_records = sum(len(v) for v in cost_records_by_file.values())
    if total_records == 0:
        raise ValueError(
            f"no cost_usd records found in {sorted(cost_records_by_file)} -- "
            "refusing to estimate from an empty basis"
        )
    if baseline_transcripts <= 0 or baseline_moments <= 0:
        raise ValueError(
            f"baseline counts must be positive (moments={baseline_moments}, "
            f"transcripts={baseline_transcripts})"
        )

    cost_files = {}
    total_cost = 0.0
    for name, costs in cost_records_by_file.items():
        file_total = sum(costs)
        total_cost += file_total
        cost_files[name] = {
            "n_records": len(costs),
            "total_usd": file_total,
            "mean_usd": (file_total / len(costs)) if costs else None,
        }

    pooled_mean = total_cost / total_records
    moments_per_transcript = baseline_moments / baseline_transcripts
    expected_moments = sample_size * moments_per_transcript
    estimated_total = expected_moments * pooled_mean

    return {
        "sample_size": sample_size,
        "baseline": {
            "moments": baseline_moments,
            "audited_transcripts": baseline_transcripts,
            "moments_per_transcript": moments_per_transcript,
        },
        "cost_files": cost_files,
        "pooled_mean_usd_per_moment_call": pooled_mean,
        "expected_moments": expected_moments,
        "estimated_total_usd": estimated_total,
    }


def estimate_from_results_dir(
    sample_size: int, results_dir: Path = FROZEN_RESULTS_DIR
) -> Dict[str, Any]:
    """
    Build the estimate from a results directory's real recorded files:
    cost basis from COST_BASIS_FILES' cost_usd fields, baseline
    moments-per-transcript from BASELINE_MOMENTS_FILE over the
    BASELINE_EVENTS_FILES (the LLM-audited transcript sets).
    """
    cost_records = {
        name: read_cost_records(results_dir / name) for name in COST_BASIS_FILES
    }

    with open(results_dir / BASELINE_MOMENTS_FILE) as f:
        baseline_moments = sum(1 for line in f if line.strip())

    baseline_transcripts = 0
    for name in BASELINE_EVENTS_FILES:
        with open(results_dir / name) as f:
            baseline_transcripts += sum(1 for line in f if line.strip())

    return compute_estimate(
        sample_size, cost_records, baseline_moments, baseline_transcripts
    )


def format_estimate(est: Dict[str, Any]) -> str:
    """Render the estimate with its full basis -- labeled numbers, units stated."""
    lines = []
    lines.append("=== COST ESTIMATE (no API calls made) ===")
    lines.append("")
    lines.append("Basis: real per-moment-call costs recorded by this audit")
    lines.append("(each record's cost_usd is the total_cost_usd claude -p itself")
    lines.append("reported for that moment's real call):")
    for name, stats in est["cost_files"].items():
        lines.append(
            f"  {name}: {stats['n_records']} records, "
            f"total ${stats['total_usd']:.4f} USD, "
            f"mean ${stats['mean_usd']:.4f} USD/call"
        )
    lines.append(
        f"  pooled mean: ${est['pooled_mean_usd_per_moment_call']:.4f} USD per "
        "moment-level call"
    )
    lines.append("")
    baseline = est["baseline"]
    lines.append(
        f"Baseline yield: {baseline['moments']} moments from "
        f"{baseline['audited_transcripts']} LLM-audited transcripts "
        f"= {baseline['moments_per_transcript']:.2f} moments/transcript"
    )
    lines.append("")
    lines.append(
        f"Extrapolation: {est['sample_size']} transcripts x "
        f"{baseline['moments_per_transcript']:.2f} moments/transcript "
        f"= {est['expected_moments']:.1f} expected moments"
    )
    lines.append(
        f"Estimated spend: {est['expected_moments']:.1f} moments x "
        f"${est['pooled_mean_usd_per_moment_call']:.4f} USD "
        f"= ${est['estimated_total_usd']:.2f} USD"
    )
    lines.append("")
    lines.append(
        "Caveat: the basis files record per-moment judgment-call spend (the only "
        "per-call costs this audit kept); per-window haiku detection calls are "
        "not in these files, so treat this as the moment-judgment floor, not the "
        "all-in ceiling."
    )
    return "\n".join(lines)


# ---------------------------------------------------------------------------
# Stage wrappers (import/invoke only -- no measurement logic)
# ---------------------------------------------------------------------------


def extract_stage(transcript_path: str) -> Dict[str, Any]:
    """Run the real extractor over one transcript."""
    return extract_module.extract_transcript(transcript_path)


def audit_stage(transcript_path: str) -> List[Dict[str, Any]]:
    """
    Run the real auditor (LLM instrument) over one transcript.
    ExhaustedRetriesError/PromptTooLongError propagate to the caller.
    """
    return audit_moments.audit_transcript(transcript_path)


def _parse_pinned_path(source: str, name: str) -> Optional[str]:
    """Read build_scorecard.py's pinned SRC/OUT Path constants from its text."""
    match = re.search(rf'^{name}\s*=\s*Path\("([^"]+)"\)', source, re.MULTILINE)
    return match.group(1) if match else None


def scorecard_stage(moments_path: Path, output_dir: Path) -> Dict[str, Any]:
    """
    Invoke build_scorecard.py -- but only when its own pinned SRC path IS this
    run's moments file. build_scorecard.py (DO-NOT-TOUCH) hardcodes absolute
    SRC/OUT paths into the frozen results/ baseline and asserts the baseline's
    moment count, so invoking it for any re-run output dir would (a) re-read
    the frozen moments.jsonl, not this run's, and (b) write into the frozen
    baseline dir. That is a loud, recorded skip -- never a silent recompute.
    """
    source = BUILD_SCORECARD_PATH.read_text()
    pinned_src = _parse_pinned_path(source, "SRC")
    pinned_out = _parse_pinned_path(source, "OUT")

    if pinned_src is None or Path(pinned_src).resolve() != Path(moments_path).resolve():
        reason = (
            f"build_scorecard.py pins SRC={pinned_src} and OUT={pinned_out} "
            f"(the frozen 2026-09 baseline); this run's moments file is "
            f"{moments_path}. Invoking it would recompute the baseline from "
            "stale inputs and write into the frozen results/ dir, so the stage "
            "is skipped loudly. A re-run scorecard needs a scorecard build "
            "parameterized on this run's moments file."
        )
        sys.stderr.write(f"[run_audit] scorecard stage SKIPPED: {reason}\n")
        return {
            "status": "skipped_pinned_baseline",
            "reason": reason,
            "pinned_src": pinned_src,
            "pinned_out": pinned_out,
        }

    result = subprocess.run(
        [sys.executable, str(BUILD_SCORECARD_PATH)],
        capture_output=True,
        text=True,
        timeout=300,
    )
    return {
        "status": "completed" if result.returncode == 0 else "failed",
        "returncode": result.returncode,
        "stdout": result.stdout[-4000:],
        "stderr": result.stderr[-4000:],
        "pinned_src": pinned_src,
        "pinned_out": pinned_out,
    }


# ---------------------------------------------------------------------------
# Run mode
# ---------------------------------------------------------------------------


def load_done_transcripts(events_path: Path) -> Set[str]:
    """
    Read the transcript paths already recorded in the per-transcript output
    file (the established resume convention: the output file is the done-set).
    """
    done: Set[str] = set()
    if not events_path.exists():
        return done
    with open(events_path) as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            done.add(json.loads(line)["transcript_path"])
    return done


def _now_iso() -> str:
    return datetime.now(timezone.utc).isoformat()


def _load_or_create_sample(
    config: Dict[str, Any], sample_path: Path, projects_root: Optional[str]
) -> List[str]:
    """
    Reuse a persisted sample.json if present (resume: the live corpus may have
    changed between invocations); otherwise enumerate the live corpus fresh,
    sample per config seed, and persist.
    """
    if sample_path.exists():
        with open(sample_path) as f:
            persisted = json.load(f)
        print(f"[run_audit] reusing persisted sample: {sample_path}", flush=True)
        return list(persisted["sample"])

    corpus = enumerate_corpus(projects_root)
    if not corpus:
        raise ValueError(
            f"live corpus enumeration found no transcripts under "
            f"{projects_root or DEFAULT_PROJECTS_ROOT} -- refusing to run on nothing"
        )
    sample = sample_corpus(corpus, config["sample_size"], config["seed"])
    sample_record = {
        "metadata": {
            "generated_at": _now_iso(),
            "seed": config["seed"],
            "sample_size": len(sample),
            "corpus_size": len(corpus),
            "projects_root": projects_root or DEFAULT_PROJECTS_ROOT,
            "note": (
                "corpus enumerated fresh from the live ~/.claude/projects layout "
                "(corpus.json is documented stale and was not read)"
            ),
        },
        "sample": sample,
    }
    with open(sample_path, "w") as f:
        json.dump(sample_record, f, indent=2)
    return sample


def run(
    config: Dict[str, Any], projects_root: Optional[str] = None
) -> Dict[str, Any]:
    """
    Run the configured stages over a fresh (or resumed) sample and write the
    run manifest. Returns the manifest dict. Raises ExhaustedRetriesError
    after persisting a halted manifest (immediate halt, per audit_moments'
    contract).
    """
    output_dir = Path(config["output_dir"])
    output_dir.mkdir(parents=True, exist_ok=True)

    sample_path = output_dir / "sample.json"
    events_path = output_dir / "transcript-events.jsonl"
    moments_path = output_dir / "moments.jsonl"
    manifest_path = output_dir / "run-manifest.json"

    stages = config["stages"]
    started_at = _now_iso()
    audit_cost_usd = 0.0
    scorecard_result: Optional[Dict[str, Any]] = None
    status = "completed"
    halt_error: Optional[BaseException] = None

    sample = _load_or_create_sample(config, sample_path, projects_root)
    done = load_done_transcripts(events_path)

    try:
        for index, transcript_path in enumerate(sample, start=1):
            if transcript_path in done:
                print(
                    f"[run_audit] [{index}/{len(sample)}] {transcript_path} "
                    "already recorded -- skipping (resume)",
                    flush=True,
                )
                continue

            print(
                f"[run_audit] [{index}/{len(sample)}] {transcript_path} ...",
                flush=True,
            )

            events_record: Dict[str, Any] = {"transcript_path": transcript_path}
            if "extract" in stages:
                events_record = extract_stage(transcript_path)

            moments: List[Dict[str, Any]] = []
            if "audit" in stages:
                try:
                    moments = audit_stage(transcript_path)
                except audit_moments.PromptTooLongError as e:
                    # Deterministic, per-transcript: record the skip on the
                    # done-marker and continue (driver convention).
                    events_record["skipped_reason"] = f"prompt_too_long: {e}"
                    sys.stderr.write(
                        f"[run_audit] {transcript_path} skipped: prompt_too_long\n"
                    )

            # Append incrementally: moments first, then the done-marker line, so
            # an interrupt can never mark a transcript done with moments unwritten.
            if moments:
                with open(moments_path, "a") as f:
                    for moment in moments:
                        f.write(json.dumps(moment) + "\n")
                audit_cost_usd += sum(
                    m.get("cost_usd") or 0.0 for m in moments
                )
            with open(events_path, "a") as f:
                f.write(json.dumps(events_record) + "\n")
            done.add(transcript_path)

        if "scorecard" in stages:
            scorecard_result = scorecard_stage(
                moments_path=moments_path, output_dir=output_dir
            )

    except audit_moments.ExhaustedRetriesError as e:
        status = "halted_exhausted_retries"
        halt_error = e
        sys.stderr.write(
            f"[run_audit] IMMEDIATE HALT -- claude -p retries exhausted "
            f"(sustained auth/API outage, not a clean zero-yield): {e}\n"
        )

    manifest = {
        "config": config,
        "instrument_version": getattr(audit_moments, "INSTRUMENT_VERSION", None),
        "started_at": started_at,
        "finished_at": _now_iso(),
        "status": status,
        "per_stage_costs": {
            "extract_usd": 0.0,
            "audit_usd": audit_cost_usd,
            "scorecard_usd": 0.0,
            "note": (
                "audit_moments.audit_transcript does not report per-call costs; "
                "audit_usd sums any cost_usd fields present on moment records "
                "(0.0 means not-recorded-by-instrument, not free)"
            ),
        },
        "outputs": {
            "sample": str(sample_path),
            "transcript_events": str(events_path),
            "moments": str(moments_path),
            "manifest": str(manifest_path),
        },
        "scorecard": scorecard_result,
        "transcripts_recorded": len(done),
        "transcripts_sampled": len(sample),
    }
    with open(manifest_path, "w") as f:
        json.dump(manifest, f, indent=2)

    if halt_error is not None:
        raise halt_error

    return manifest


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def main(argv: Optional[List[str]] = None) -> int:
    parser = argparse.ArgumentParser(
        description="Thin pipeline runner for the memory-loop audit "
        "(orchestration only -- wraps extract/audit_moments/build_scorecard)"
    )
    parser.add_argument("--config", required=True, help="path to JSON config file")
    parser.add_argument(
        "--estimate",
        action="store_true",
        help="print a real-basis cost estimate (no API calls) and exit",
    )
    args = parser.parse_args(argv)

    config = load_config(args.config)

    if args.estimate:
        est = estimate_from_results_dir(config["sample_size"])
        print(format_estimate(est))
        return 0

    try:
        manifest = run(config)
    except audit_moments.ExhaustedRetriesError:
        return 2

    print(
        f"[run_audit] done: {manifest['transcripts_recorded']}/"
        f"{manifest['transcripts_sampled']} transcripts recorded; "
        f"manifest at {manifest['outputs']['manifest']}"
    )
    return 0


if __name__ == "__main__":
    sys.exit(main())
