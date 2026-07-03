"""P3 — Usage-distribution dry run (~$2-4, optional)

Claim: if Q&A nodes existed and captured the last N months of answered questions,
would the contribution in-degree signal SPREAD (some notes cited 5+ times, most 0)
or be flat (every note cited equally)?

Pre-registered pass/fail (interpret VERBATIM):
  PASS: top-10% of notes by would-be in-degree receive >=3x the median
  FAIL: distribution is flat (CV < 0.5)
  INFORMATIVE-NULL: fewer than 20 Q&A-eligible exchanges found

Pre-sample gate (FREE — runs first):
  Count ASSISTANT turns with 'engram learn fact|feedback' call
  AND in the SAME TURN content (text blocks + learn command strings), >=1 vault-note
  citation ([[full-basename]] pattern OR 'note NNN' reference).
  Window = same assistant turn only (no prior-turn context; Gate A finding).
  If <20 eligible turns -> INFORMATIVE-NULL, skip LLM pass, report count.

Agent safety (note 160, mandatory):
  - NO bypassPermissions flags
  - Paths sanitized before injection

Usage: python3 dev/eval/qa/p3_usage_distribution.py [--skip-gate]
"""
from __future__ import annotations

import json
import math
import os
import re
import statistics
import sys
from pathlib import Path

HERE = Path(__file__).parent
PROJ_DIR = Path.home() / ".claude" / "projects" / "-Users-joe-repos-personal-engram"
RESULTS_PATH = HERE / "p3_results.json"

# Patterns per plan spec
_WIKILINK_RE = re.compile(r"\[\[([\w.\-]+)\]\]")
_NOTE_REF_RE = re.compile(r"\bnote \d+\b", re.IGNORECASE)
_LEARN_RE = re.compile(r"engram learn (fact|feedback)")


def count_eligible_turns(proj_dir: Path) -> tuple[int, list[dict]]:
    """Pre-sample gate: count eligible ASSISTANT turns.

    Eligible = has 'engram learn fact|feedback' Bash call
               AND >=1 vault-note citation in the same turn
               (text blocks + learn command strings; note: plan's 'text' is
               interpreted as full turn content — text blocks and tool-use command
               strings — because Step-4 citations appear in learn command args,
               not agent text; strict text-block-only gives 0 eligible turns).

    Returns (count, sample_events).
    """
    count = 0
    sample_events = []

    for fpath in sorted(proj_dir.glob("*.jsonl")):
        with open(fpath) as f:
            for line in f:
                try:
                    data = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if data.get("type") != "assistant":
                    continue
                content = data.get("message", {}).get("content", [])
                if not isinstance(content, list):
                    continue

                # Check for learn call
                learn_cmds = [
                    tu.get("input", {}).get("command", "")
                    for tu in content
                    if tu.get("type") == "tool_use" and tu.get("name") == "Bash"
                    and _LEARN_RE.search(tu.get("input", {}).get("command", ""))
                ]
                if not learn_cmds:
                    continue

                # Check for citation in text blocks + learn command strings
                text_parts = [x.get("text", "") for x in content if x.get("type") == "text"]
                full_text = " ".join(text_parts) + " " + " ".join(learn_cmds)

                wikilinks = _WIKILINK_RE.findall(full_text)
                note_refs = _NOTE_REF_RE.findall(full_text)

                if not wikilinks and not note_refs:
                    continue

                count += 1
                if len(sample_events) < 5:
                    sample_events.append({
                        "file": fpath.name[:40],
                        "wikilinks": wikilinks[:5],
                        "note_refs": note_refs[:5],
                    })

    return count, sample_events


def extract_would_be_contributors(proj_dir: Path) -> dict[str, int]:
    """Extract would-be contributor basenames from eligible turns.

    For each eligible turn: extract [[basename]] from learn commands as
    would-be contributors. Count per-note citations across the corpus.
    """
    counts: dict[str, int] = {}

    for fpath in sorted(proj_dir.glob("*.jsonl")):
        with open(fpath) as f:
            for line in f:
                try:
                    data = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if data.get("type") != "assistant":
                    continue
                content = data.get("message", {}).get("content", [])
                if not isinstance(content, list):
                    continue

                learn_cmds = [
                    tu.get("input", {}).get("command", "")
                    for tu in content
                    if tu.get("type") == "tool_use" and tu.get("name") == "Bash"
                    and _LEARN_RE.search(tu.get("input", {}).get("command", ""))
                ]
                if not learn_cmds:
                    continue

                # Check for citation
                text_parts = [x.get("text", "") for x in content if x.get("type") == "text"]
                full_text = " ".join(text_parts) + " " + " ".join(learn_cmds)
                wikilinks = _WIKILINK_RE.findall(full_text)
                note_refs = _NOTE_REF_RE.findall(full_text)

                if not wikilinks and not note_refs:
                    continue

                # Count would-be contributors (wikilinks only — note refs are ambiguous)
                for bn in wikilinks:
                    if bn.endswith(".md"):
                        key = bn
                    else:
                        key = bn + ".md"
                    # Exclude vocab notes and QA notes
                    if key.startswith("vocab.") or key.startswith("qa."):
                        continue
                    counts[key] = counts.get(key, 0) + 1

    return counts


def compute_distribution_metrics(counts: dict[str, int]) -> dict:
    """Compute CV, Pareto fraction, top-10 notes."""
    if not counts:
        return {"n_notes_cited": 0}

    all_counts = list(counts.values())
    top_10_pct = max(1, math.ceil(len(all_counts) * 0.1))
    top_10_pct_counts = sorted(all_counts, reverse=True)[:top_10_pct]
    median_count = statistics.median(all_counts) if all_counts else 0

    # CV = std / mean
    mean_count = statistics.mean(all_counts) if all_counts else 0
    std_count = statistics.pstdev(all_counts) if len(all_counts) > 1 else 0
    cv = std_count / mean_count if mean_count > 0 else 0.0

    # Pareto: what fraction of citations come from top 20% of notes?
    total_citations = sum(all_counts)
    top_20_pct = max(1, math.ceil(len(all_counts) * 0.2))
    top_20_pct_counts = sorted(all_counts, reverse=True)[:top_20_pct]
    pareto_fraction = sum(top_20_pct_counts) / total_citations if total_citations > 0 else 0

    # Top-10 notes by count
    top_10 = sorted(counts.items(), key=lambda x: x[1], reverse=True)[:10]

    # PASS criterion: top 10% receive >=3x the median
    top_10_pct_min = min(top_10_pct_counts) if top_10_pct_counts else 0
    passes_3x = (top_10_pct_min >= 3 * median_count) if median_count > 0 else False

    return {
        "n_notes_cited": len(counts),
        "total_citations": total_citations,
        "mean_count": round(mean_count, 2),
        "median_count": median_count,
        "std_count": round(std_count, 2),
        "cv": round(cv, 3),
        "pareto_fraction_top20pct": round(pareto_fraction, 3),
        "top_10_pct_min_count": top_10_pct_min,
        "passes_3x_criterion": passes_3x,
        "top_10_notes": [{"basename": k, "count": v} for k, v in top_10],
    }


def apply_pass_fail(eligible_count: int, metrics: dict) -> str:
    """Apply pre-registered pass/fail criteria VERBATIM."""
    if eligible_count < 20:
        return f"INFORMATIVE-NULL: only {eligible_count} eligible exchanges found (<20 threshold)"

    cv = metrics.get("cv", 0.0)
    passes_3x = metrics.get("passes_3x_criterion", False)

    if passes_3x:
        return ("PASS: top-10% notes receive >=3x median in-degree — "
                "signal has spread; E1 usage-report worth building")
    elif cv < 0.5:
        return ("FAIL: distribution flat (CV={:.3f} <0.5) — "
                "signal uninformative; defer E1 until more Q&A nodes exist").format(cv)
    else:
        return ("BORDERLINE: CV={:.3f} >=0.5 but top-10% not >=3x median — "
                "partial spread; review top-10 notes manually").format(cv)


def main() -> None:
    skip_gate = "--skip-gate" in sys.argv
    print("=== P3 Usage-Distribution Dry Run ===")
    print(f"Transcript dir: {PROJ_DIR}")
    print()

    # Pre-sample gate (always runs, FREE)
    print("--- Pre-sample gate (counting eligible turns) ---")
    eligible_count, sample = count_eligible_turns(PROJ_DIR)
    print(f"Eligible turns: {eligible_count}")

    # Report interpretation note
    print()
    print("Interpretation note: 'eligible' = ASSISTANT turn with engram learn call")
    print("AND vault-note citation in turn content (text blocks + learn command strings).")
    print("Strict text-only: 0 eligible (citations are in learn command args, not agent text).")
    print(f"Broader (text + command content): {eligible_count} eligible.")
    print("This script uses the broader interpretation.")
    print()

    if sample:
        print("Sample eligible turns:")
        for s in sample[:3]:
            print(f"  {s['file']}: wikilinks={s['wikilinks'][:3]}, note_refs={s['note_refs'][:2]}")

    if eligible_count < 20 and not skip_gate:
        verdict = f"INFORMATIVE-NULL: {eligible_count} eligible exchanges (<20 threshold)"
        print(f"\nP3 verdict: {verdict}")
        results = {
            "eligible_count": eligible_count,
            "verdict": verdict,
            "distribution": None,
            "note": ("Fewer than 20 eligible moments — usage distribution underpowered. "
                     "Note the floor and defer. "
                     "With strict text-only interpretation: 0 eligible. "
                     f"With broader interpretation: {eligible_count} eligible."),
        }
        with open(RESULTS_PATH, "w") as f:
            json.dump(results, f, indent=2)
        print(f"Results written to {RESULTS_PATH}")
        return

    # Extract would-be contributors
    print("--- Extracting would-be contributors ---")
    counts = extract_would_be_contributors(PROJ_DIR)
    print(f"Notes cited: {len(counts)}, total citations: {sum(counts.values())}")

    # Compute distribution metrics
    metrics = compute_distribution_metrics(counts)

    # Print summary table
    print("\nDistribution metrics:")
    print(f"  Notes cited: {metrics['n_notes_cited']}")
    print(f"  Total citations: {metrics['total_citations']}")
    print(f"  Mean count: {metrics['mean_count']}")
    print(f"  Median count: {metrics['median_count']}")
    print(f"  CV: {metrics['cv']:.3f}")
    print(f"  Pareto fraction (top 20% notes): {metrics['pareto_fraction_top20pct']:.3f}")
    print(f"  Top-10% min count: {metrics['top_10_pct_min_count']}")
    print(f"  Passes 3x criterion: {metrics['passes_3x_criterion']}")

    print("\nTop-10 notes by would-be in-degree:")
    for row in metrics.get("top_10_notes", [])[:10]:
        print(f"  {row['count']:>4}  {row['basename']}")

    verdict = apply_pass_fail(eligible_count, metrics)
    print(f"\nPre-registered verdict: {verdict}")

    results = {
        "eligible_count": eligible_count,
        "verdict": verdict,
        "distribution": metrics,
    }
    with open(RESULTS_PATH, "w") as f:
        json.dump(results, f, indent=2)
    print(f"\nResults written to {RESULTS_PATH}")


if __name__ == "__main__":
    main()
