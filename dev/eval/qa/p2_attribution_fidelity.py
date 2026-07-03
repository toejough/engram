"""P2 — Attribution fidelity probe (~$3-8)

Claim: cite-derived attribution (extract [[basename]] wikilinks written in the
answer text) is more accurate and less confabulating than free-listed attribution
(agent enumerates "what notes did you use?" at close). Ground truth: opus judge.

Pre-registered pass/fail (interpret VERBATIM, no re-derivation):
  PASS: cite-derived confab rate < 20% AND free-list confab rate > 30%
        AND separation >= 15pp
  BOTH > 30%: revise D2 capture bar before any build
  BOTH < 20%: cite-derived validated; free-list not refuted (label inconclusive)
  ANY OTHER: BORDERLINE — adopt cite-derived iff its rate < 20%, caveat recorded
  RECALL-BORDERLINE: cite-derived recall (ground-truth coverage) < 50% → bar too low

Agent safety (note 160, mandatory):
  - NO bypassPermissions flags on claude -p calls
  - Transcript excerpts payload-sanitized: strip /Users/joe/repos/... paths before injection
  - No absolute paths in agent payloads

Checkpoint: each trial appended to p2_checkpoint.jsonl immediately (flush per write).
Resume: run skips case_ids already in checkpoint (note 159).

Usage: python3 dev/eval/qa/p2_attribution_fidelity.py [--max-cases N]
"""
from __future__ import annotations

import json
import os
import re
import subprocess
import sys
import textwrap
from pathlib import Path

HERE = Path(__file__).parent
PROJ_DIR = Path.home() / ".claude" / "projects" / "-Users-joe-repos-personal-engram"
CHECKPOINT_PATH = HERE / "p2_checkpoint.jsonl"
RESULTS_PATH = HERE / "p2_results.json"
VAULT_PATH = Path(os.environ.get("ENGRAM_VAULT_PATH",
    Path.home() / ".local" / "share" / "engram" / "vault"))

# Absolute-path sanitization: strip /Users/joe/repos/... → <repo-path>
_ABS_PATH_RE = re.compile(r"/Users/joe/repos/[^\s'\"`\]]+")

def sanitize_paths(text: str) -> str:
    """Replace absolute repo paths with <repo-path> placeholder."""
    return _ABS_PATH_RE.sub("<repo-path>", text)

# Wikilink pattern for cite-derived attribution
_WIKILINK_RE = re.compile(r"\[\[([\w.\-]+)\]\]")


def load_checkpoint() -> set[str]:
    """Return set of already-completed case_ids from checkpoint."""
    done = set()
    if CHECKPOINT_PATH.exists():
        with open(CHECKPOINT_PATH) as f:
            for line in f:
                try:
                    done.add(json.loads(line)["case_id"])
                except (json.JSONDecodeError, KeyError):
                    pass
    return done


def append_checkpoint(record: dict) -> None:
    """Append a trial result to the checkpoint file (immediate flush)."""
    with open(CHECKPOINT_PATH, "a") as f:
        f.write(json.dumps(record) + "\n")
        f.flush()
        os.fsync(f.fileno())


def find_synthesis_events(max_cases: int = 10) -> list[dict]:
    """Scan transcripts for deep recall Step-4 synthesis events.

    A synthesis event is an ASSISTANT turn that:
      (1) Has a Bash tool_use calling 'engram learn fact|feedback'
      (2) The engram learn command includes --chunk-source (Step 4 provenance)

    Returns up to max_cases events with:
      - case_id: unique identifier
      - learn_cmd: the full engram learn command
      - turn_text: assistant's text content (sanitized)
      - note_body: extracted from the learn command --object or similar
    """
    events = []
    seen_cmds: set[str] = set()  # deduplicate

    for fpath in sorted(PROJ_DIR.glob("*.jsonl")):
        if len(events) >= max_cases:
            break
        with open(fpath) as f:
            for line in f:
                if len(events) >= max_cases:
                    break
                try:
                    data = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if data.get("type") != "assistant":
                    continue
                content = data.get("message", {}).get("content", [])
                if not isinstance(content, list):
                    continue

                # Find Bash tool_use with engram learn fact|feedback + --chunk-source
                learn_cmd = None
                for item in content:
                    if item.get("type") != "tool_use" or item.get("name") != "Bash":
                        continue
                    cmd = item.get("input", {}).get("command", "")
                    if re.search(r"engram learn (fact|feedback)", cmd) and "--chunk-source" in cmd:
                        learn_cmd = cmd
                        break

                if not learn_cmd:
                    continue

                # Deduplicate by command string
                cmd_key = learn_cmd[:200]
                if cmd_key in seen_cmds:
                    continue
                seen_cmds.add(cmd_key)

                # Extract text content (sanitized)
                text_parts = [sanitize_paths(x.get("text", ""))
                              for x in content if x.get("type") == "text"]
                turn_text = "\n".join(text_parts)[:3000]  # cap to reduce cost

                # Extract what the note says (--object, --behavior, etc.)
                note_content = _extract_note_args(learn_cmd)
                if not note_content:
                    continue

                # Extract cite-derived basenames from the learn command itself
                # (some Step 4 commands include --supersedes with basenames)
                cited = _WIKILINK_RE.findall(learn_cmd)

                events.append({
                    "case_id": f"P2-{len(events):02d}",
                    "session_file": fpath.name,
                    "learn_cmd": sanitize_paths(learn_cmd),
                    "note_content": sanitize_paths(note_content),
                    "cited_in_cmd": cited,
                    "turn_text": turn_text,
                })

    print(f"Found {len(events)} synthesis events")
    return events


def _extract_note_args(cmd: str) -> str:
    """Extract the note substance from an engram learn command."""
    # Try to get --object, --behavior, --action, --situation concatenated
    parts = []
    for flag in ("--situation", "--subject", "--predicate", "--object",
                 "--behavior", "--impact", "--action"):
        m = re.search(rf"{re.escape(flag)}\s+['\"]([^'\"]+)['\"]", cmd)
        if m:
            parts.append(f"{flag}: {m.group(1)[:300]}")
    # Also try unquoted (multiline) by taking text after the flag up to next flag
    return "\n".join(parts) if parts else cmd[:500]


def run_sonnet_free_list(case: dict) -> list[str]:
    """Run a fresh sonnet agent to free-list which notes contributed.

    Returns list of note basenames the agent attributes.
    """
    prompt = textwrap.dedent(f"""\
    You are reviewing a recall synthesis note that was written by an agent.
    The note content is:

    {case["note_content"]}

    The agent's turn text (context) was:

    {case["turn_text"][:1500]}

    Task: List which vault notes (by their full basename, e.g. '159.2026-07-02.slug.md')
    you believe CONTRIBUTED to this synthesis. A note contributes if its content
    directly influenced or was cited in the synthesis.

    Respond with ONLY a JSON array of basenames, e.g.:
    ["159.2026-07-02.eval-runs-checkpoint-per-trial-and-survive-orchestrator.md"]

    If you cannot identify any specific notes, return [].
    """)

    result = subprocess.run(
        ["claude", "-p", prompt, "--model", "claude-sonnet-4-5"],
        capture_output=True, text=True, timeout=120,
    )
    if result.returncode != 0:
        print(f"  WARN: sonnet call failed: {result.stderr[:200]}")
        return []

    # Parse JSON array from response
    output = result.stdout.strip()
    try:
        # Find JSON array in output
        m = re.search(r"\[([^\]]*)\]", output, re.DOTALL)
        if m:
            parsed = json.loads("[" + m.group(1) + "]")
            return [str(x).strip() for x in parsed if isinstance(x, str) and x.strip()]
    except (json.JSONDecodeError, ValueError):
        pass
    return []


def run_opus_judge(case: dict) -> dict:
    """Run opus judge to establish ground truth for one case.

    Returns: {load_bearing: [basenames], claim_supplied: {basename: claim}}
    """
    # Get list of notes from vault to help the judge
    vault_notes = []
    if VAULT_PATH.exists():
        vault_notes = [f.name for f in VAULT_PATH.glob("*.md")
                       if not f.name.startswith("vocab.")][:50]

    note_list_str = "\n".join(f"- {n}" for n in vault_notes[:30])

    prompt = textwrap.dedent(f"""\
    You are an expert judge evaluating which vault notes were load-bearing sources
    for a synthesis note.

    SYNTHESIS NOTE CONTENT:
    {case["note_content"]}

    AGENT CONTEXT (turn text):
    {case["turn_text"][:2000]}

    DEFINITION: A note is load-bearing if removing its content would change the
    answer's substance. For each load-bearing note, identify the specific claim
    it supplied.

    SAMPLE VAULT NOTES (for reference — basenames like NNN.YYYY-MM-DD.slug.md):
    {note_list_str}

    Respond with ONLY this JSON schema (no other text):
    {{
      "load_bearing": ["<full-basename-1>", "<full-basename-2>"],
      "claim_supplied": {{
        "<full-basename-1>": "<specific claim this note supplied>",
        "<full-basename-2>": "<specific claim this note supplied>"
      }}
    }}

    If none are identifiable from the context, return:
    {{"load_bearing": [], "claim_supplied": {{}}}}
    """)

    result = subprocess.run(
        ["claude", "-p", prompt, "--model", "claude-opus-4-5"],
        capture_output=True, text=True, timeout=180,
    )
    if result.returncode != 0:
        print(f"  WARN: opus call failed: {result.stderr[:200]}")
        return {"load_bearing": [], "claim_supplied": {}}

    output = result.stdout.strip()
    try:
        # Find JSON object in output
        m = re.search(r"\{.*\}", output, re.DOTALL)
        if m:
            parsed = json.loads(m.group(0))
            return {
                "load_bearing": parsed.get("load_bearing", []),
                "claim_supplied": parsed.get("claim_supplied", {}),
            }
    except (json.JSONDecodeError, ValueError) as e:
        print(f"  WARN: could not parse opus response: {e}")
    return {"load_bearing": [], "claim_supplied": {}}


def compute_rates(trials: list[dict]) -> dict:
    """Compute confabulation rates and recall rates from completed trials."""
    cite_confab_count = 0
    cite_recall_count = 0
    free_confab_count = 0
    free_recall_count = 0
    total_trials = 0

    for t in trials:
        gt = set(t.get("ground_truth", {}).get("load_bearing", []))
        cited = set(t.get("cite_derived", []))
        free = set(t.get("free_listed", []))

        # Confabulation: attributed but not in ground truth
        if cited:
            cite_confab_count += len(cited - gt) / max(len(cited), 1)
        if free:
            free_confab_count += len(free - gt) / max(len(free), 1)

        # Recall: ground truth notes captured
        if gt:
            cite_recall_count += len(cited & gt) / len(gt)
            free_recall_count += len(free & gt) / len(gt)

        total_trials += 1

    if total_trials == 0:
        return {}

    cite_confab_rate = round(cite_confab_count / total_trials * 100, 1)
    free_confab_rate = round(free_confab_count / total_trials * 100, 1)
    cite_recall_rate = round(cite_recall_count / total_trials * 100, 1)
    free_recall_rate = round(free_recall_count / total_trials * 100, 1)

    return {
        "n_trials": total_trials,
        "cite_confab_rate_pct": cite_confab_rate,
        "free_confab_rate_pct": free_confab_rate,
        "cite_recall_rate_pct": cite_recall_rate,
        "free_recall_rate_pct": free_recall_rate,
        "separation_pp": round(free_confab_rate - cite_confab_rate, 1),
    }


def apply_pass_fail(rates: dict) -> str:
    """Apply pre-registered pass/fail criteria VERBATIM. Returns the branch that fired."""
    if not rates:
        return "NO-DATA: no trials completed"

    cite_cfb = rates["cite_confab_rate_pct"]
    free_cfb = rates["free_confab_rate_pct"]
    sep = rates["separation_pp"]
    cite_rcl = rates["cite_recall_rate_pct"]

    # Check RECALL-BORDERLINE first (independent criterion)
    recall_border = cite_rcl < 50.0

    # Main branches (exhaustive by construction per plan)
    if cite_cfb < 20.0 and free_cfb > 30.0 and sep >= 15.0:
        branch = "PASS: cite-derived confab <20%, free-list >30%, separation >=15pp"
    elif cite_cfb > 30.0 and free_cfb > 30.0:
        branch = "BOTH>30%: even cite-derived confabulates — revise D2 capture bar"
    elif cite_cfb < 20.0 and free_cfb < 20.0:
        branch = ("BOTH<20%: cite-derived validated-accurate; free-list not refuted "
                  "(inconclusive, tier-specific)")
    else:
        # ANY OTHER RESULT (catch-all)
        notes = []
        if 20.0 <= free_cfb <= 30.0:
            notes.append("middle-band free-list")
        if cite_cfb >= free_cfb:
            notes.append("inverted ordering")
        if sep < 15.0 and free_cfb > cite_cfb:
            notes.append("weak separation")
        note_str = "; ".join(notes) if notes else "mixed levels"
        adopt = "adopt cite-derived" if cite_cfb < 20.0 else "do not adopt cite-derived"
        branch = f"BORDERLINE ({note_str}): {adopt} if rate <20%"

    if recall_border:
        branch += f" | RECALL-BORDERLINE: cite recall={cite_rcl:.1f}% <50%"

    return branch


def main() -> None:
    max_cases = 10
    if "--max-cases" in sys.argv:
        idx = sys.argv.index("--max-cases")
        max_cases = int(sys.argv[idx + 1])

    print("=== P2 Attribution Fidelity Probe ===")
    print(f"Checkpoint: {CHECKPOINT_PATH}")
    print(f"Max cases: {max_cases}")
    print()

    done = load_checkpoint()
    if done:
        print(f"Resuming: {len(done)} cases already in checkpoint: {sorted(done)}")

    events = find_synthesis_events(max_cases=max_cases * 2)  # fetch extra for filtering
    events = events[:max_cases]

    if not events:
        print("No synthesis events found in transcripts.")
        print("P2 result: NO-DATA (no eligible Step-4 synthesis events found)")
        return

    trials = []
    for i, event in enumerate(events):
        case_id = event["case_id"]
        if case_id in done:
            print(f"  [{i+1}/{len(events)}] {case_id}: skipped (already done)")
            continue

        print(f"  [{i+1}/{len(events)}] {case_id}: {event['session_file'][:30]}...")

        # a) Cite-derived attribution from the learn command
        cite_derived = list(set(event.get("cited_in_cmd", [])))
        print(f"    cite-derived: {cite_derived}")

        # b) Fresh sonnet free-list
        print(f"    running sonnet free-list...")
        free_listed = run_sonnet_free_list(event)
        print(f"    free-listed: {free_listed}")

        # c) Opus ground truth judge
        print(f"    running opus judge...")
        ground_truth = run_opus_judge(event)
        print(f"    ground truth load-bearing: {ground_truth.get('load_bearing', [])}")

        record = {
            "case_id": case_id,
            "session_file": event["session_file"],
            "note_content": event["note_content"][:500],
            "cite_derived": cite_derived,
            "free_listed": free_listed,
            "ground_truth": ground_truth,
            "ground_truth_source": "opus-judge (LLM-labeled)",
        }
        append_checkpoint(record)
        trials.append(record)
        done.add(case_id)

    # Load all trials from checkpoint (including from previous runs)
    all_trials = []
    if CHECKPOINT_PATH.exists():
        with open(CHECKPOINT_PATH) as f:
            for line in f:
                try:
                    all_trials.append(json.loads(line))
                except json.JSONDecodeError:
                    pass

    print(f"\n=== Results ({len(all_trials)} trials) ===")

    rates = compute_rates(all_trials)
    print("\nRates table:")
    print(f"  {'Metric':<35} {'Cite-Derived':>12} {'Free-List':>10}")
    print(f"  {'-'*57}")
    print(f"  {'Confabulation rate (%)':35} {rates.get('cite_confab_rate_pct', 'N/A'):>12} "
          f"{rates.get('free_confab_rate_pct', 'N/A'):>10}")
    print(f"  {'Recall rate (% of GT notes captured)':35} "
          f"{rates.get('cite_recall_rate_pct', 'N/A'):>12} "
          f"{rates.get('free_recall_rate_pct', 'N/A'):>10}")
    print(f"  {'Separation (pp, free - cite confab)':35} "
          f"{rates.get('separation_pp', 'N/A'):>22}")
    print(f"  NOTE: free-list arm uses sonnet (tier-specific finding)")

    verdict = apply_pass_fail(rates)
    print(f"\nPre-registered verdict: {verdict}")

    # Write final results
    results = {
        "rates": rates,
        "verdict": verdict,
        "n_trials": len(all_trials),
        "trials": all_trials,
        "cost_note": "estimated (not metered); sonnet ~$0.30/call, opus ~$0.30-0.60/call",
    }
    with open(RESULTS_PATH, "w") as f:
        json.dump(results, f, indent=2)
    print(f"\nResults written to {RESULTS_PATH}")


if __name__ == "__main__":
    main()
