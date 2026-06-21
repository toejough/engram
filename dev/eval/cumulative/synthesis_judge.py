#!/usr/bin/env python3
"""Synthesis judge: evaluate emergent-synthesis (C6 axis) probe results.

Two modes:
  stub=True  — deterministic, zero-cost, used by validate.py and CI.
  stub=False — real adversarial LLM judge: majority over ≥3 runs, refute-by-default.

The real judge (stub=False) is invoked only in the live C6 probe (Phase 3).
No paid LLM calls are made in this module itself — the caller controls when
stub=False is used.

Fixture layout (≥3 designed-gap fixtures under synthesis_fixtures/):
  fixture{N}/
    vault_with_cluster/   — flat vault, facet-notes only, no note states Z
    vault_covered/        — flat vault, Z is already explicitly stated
    task.txt              — the task needing Z to solve correctly
    expected_synthesis_z.txt — the integrative lesson Z (one line)
"""
import json
import os
import subprocess
import textwrap

STUB_MODE = True

# Default judge model — override with --judge-model at CLI or judge_model arg.
DEFAULT_JUDGE_MODEL = "claude-sonnet-4-6"

# Number of judge runs for majority vote.
JUDGE_RUNS = 3

# The adversarial rubric.  Default verdict is "not synthesis"; LLM must REFUTE
# the default to flip to "synthesis".
_JUDGE_SYSTEM = textwrap.dedent("""\
    You are an adversarial synthesis judge.  Your default verdict is
    NOT_SYNTHESIS.  You flip to SYNTHESIS only when the evidence is
    unambiguous: the crystallized note must state an integrative lesson Z
    that no single facet-note stated, and must not merely restate or
    paraphrase one of the input notes.

    You will be given:
      FACET_NOTES — the individual facts in the vault (the cluster).
      TASK        — the task the agent was asked to solve.
      EXPECTED_Z  — the integrative lesson Z the probe expects.
      CRYSTALLIZED_NOTE — the note the /recall skill wrote (or "NONE").
      BUILD_USED_Z — whether the downstream build acted on Z (true/false/unknown).

    Respond with a single JSON object:
      {
        "verdict": "SYNTHESIS" | "NOT_SYNTHESIS",
        "reason": "<one sentence>",
        "z_present": true | false,
        "z_integrative": true | false,
        "build_used_z": true | false | null
      }

    Rules:
    1. z_present = the crystallized note explicitly states Z (not just
       restates a single facet).
    2. z_integrative = Z combines ≥2 facet claims into a joint principle
       that no individual facet stated alone.
    3. verdict = SYNTHESIS iff z_present AND z_integrative.
    4. build_used_z = true iff the build artifact demonstrates downstream
       use of Z (not a facet alone); null when unknown.
    5. Never output text outside the JSON object.
""")

_JUDGE_USER_TMPL = textwrap.dedent("""\
    FACET_NOTES:
    {facet_notes}

    TASK:
    {task}

    EXPECTED_Z:
    {expected_z}

    CRYSTALLIZED_NOTE:
    {crystallized_note}

    BUILD_USED_Z: {build_used_z}
""")


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------

def judge_crystallization(
    vault_path,
    task_txt,
    expected_z_txt,
    stub=True,
    crystallized_note_content=None,
    build_used_z=None,
    judge_model=DEFAULT_JUDGE_MODEL,
):
    """Evaluate whether a new note was written capturing emergent synthesis Z.

    Parameters
    ----------
    vault_path : str
        Path to the fixture vault (vault_with_cluster or vault_covered).
    task_txt : str
        Task description.
    expected_z_txt : str
        The integrative lesson Z the probe expects.
    stub : bool
        If True: deterministic per vault path pattern (zero cost).
        If False: real adversarial LLM judge via `claude` CLI (paid).
    crystallized_note_content : str | None
        Content of the newly crystallized note.  None = not crystallized.
    build_used_z : bool | None
        Whether the downstream build acted on Z.
    judge_model : str
        Model ID for the real judge (ignored in stub mode).

    Returns
    -------
    dict with keys:
      verdict         : "SIGNAL" | "MUST_NOT_FIRE" | "NULL" | "SYNTHESIS" | "NOT_SYNTHESIS"
      crystallized    : bool
      z_correct       : bool
      note_path       : str | None
      stub_mode       : bool
      judge_runs      : list[dict] | None   (real mode only)
    """
    if stub:
        return _stub_judge(vault_path)

    return _real_judge(
        vault_path=vault_path,
        task_txt=task_txt,
        expected_z_txt=expected_z_txt,
        crystallized_note_content=crystallized_note_content,
        build_used_z=build_used_z,
        judge_model=judge_model,
    )


def parse_synthesis_payload(payload_str):
    """Parse engram query output for clusters and candidate_l2s fields."""
    try:
        data = json.loads(payload_str)
    except Exception:
        data = {}
    return {
        "clusters": data.get("clusters", []),
        "candidate_l2s": data.get("candidate_l2s", []),
    }


def list_fixtures(synthesis_fixtures_dir):
    """Return a sorted list of (fixture_dir, vault_cluster, vault_covered, task, expected_z)
    for every fixture{N}/ subdirectory found under synthesis_fixtures_dir.

    Yields dicts with keys: name, vault_with_cluster, vault_covered, task_txt, expected_z_txt.
    """
    fix_dir = synthesis_fixtures_dir
    fixture_names = sorted(
        name for name in os.listdir(fix_dir)
        if name.startswith("fixture") and os.path.isdir(os.path.join(fix_dir, name))
    )
    for name in fixture_names:
        base = os.path.join(fix_dir, name)
        yield {
            "name": name,
            "vault_with_cluster": os.path.join(base, "vault_with_cluster"),
            "vault_covered": os.path.join(base, "vault_covered"),
            "task_txt": open(os.path.join(base, "task.txt")).read().strip(),
            "expected_z_txt": open(os.path.join(base, "expected_synthesis_z.txt")).read().strip(),
        }


# ---------------------------------------------------------------------------
# Stub implementation
# ---------------------------------------------------------------------------

def _stub_judge(vault_path):
    if "vault_with_cluster" in vault_path:
        return {
            "crystallized": True,
            "z_correct": True,
            "verdict": "SIGNAL",
            "note_path": None,
            "stub_mode": True,
            "judge_runs": None,
        }
    if "vault_covered" in vault_path:
        return {
            "crystallized": False,
            "z_correct": False,
            "verdict": "MUST_NOT_FIRE",
            "note_path": None,
            "stub_mode": True,
            "judge_runs": None,
        }
    return {
        "crystallized": False,
        "z_correct": False,
        "verdict": "NULL",
        "note_path": None,
        "stub_mode": True,
        "judge_runs": None,
    }


# ---------------------------------------------------------------------------
# Real judge implementation
# ---------------------------------------------------------------------------

def _real_judge(vault_path, task_txt, expected_z_txt, crystallized_note_content,
                build_used_z, judge_model):
    """Run JUDGE_RUNS adversarial judge calls and return majority verdict."""
    facet_notes = _read_vault_notes(vault_path)
    crystallized_str = crystallized_note_content if crystallized_note_content else "NONE"
    build_used_str = "true" if build_used_z is True else ("false" if build_used_z is False else "unknown")

    user_prompt = _JUDGE_USER_TMPL.format(
        facet_notes=facet_notes,
        task=task_txt,
        expected_z=expected_z_txt,
        crystallized_note=crystallized_str,
        build_used_z=build_used_str,
    )

    runs = []
    for _ in range(JUDGE_RUNS):
        result = _call_claude_judge(user_prompt, judge_model)
        runs.append(result)

    synthesis_votes = sum(1 for r in runs if r.get("verdict") == "SYNTHESIS")
    majority_verdict = "SYNTHESIS" if synthesis_votes > JUDGE_RUNS // 2 else "NOT_SYNTHESIS"

    z_correct = majority_verdict == "SYNTHESIS"
    crystallized = crystallized_note_content is not None

    return {
        "crystallized": crystallized,
        "z_correct": z_correct,
        "verdict": majority_verdict,
        "note_path": None,
        "stub_mode": False,
        "judge_runs": runs,
        "synthesis_votes": synthesis_votes,
        "total_runs": JUDGE_RUNS,
    }


def _read_vault_notes(vault_path):
    """Read all .md files from the flat vault and return concatenated content."""
    parts = []
    if not os.path.isdir(vault_path):
        return "(vault not found)"
    for fname in sorted(os.listdir(vault_path)):
        if fname.endswith(".md"):
            fpath = os.path.join(vault_path, fname)
            try:
                content = open(fpath).read().strip()
                parts.append(f"--- {fname} ---\n{content}")
            except OSError:
                pass
    return "\n\n".join(parts) if parts else "(no notes found)"


def _call_claude_judge(user_prompt, judge_model):
    """Invoke the claude CLI with the adversarial judge prompt.

    Returns a parsed dict from the judge's JSON response.
    Raises on CLI failure.
    """
    full_prompt = _JUDGE_SYSTEM + "\n\n" + user_prompt
    cmd = ["claude", "--model", judge_model, "--print", full_prompt]
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=120)
    if result.returncode != 0:
        raise RuntimeError(
            f"claude judge CLI failed (exit {result.returncode}): {result.stderr[:200]}"
        )
    raw = result.stdout.strip()
    # Extract JSON from the response (may have surrounding text)
    return _parse_judge_json(raw)


def _parse_judge_json(raw):
    """Extract and parse the JSON object from a judge response string.

    Tolerates leading/trailing prose by searching for the first '{' and
    last '}' pair.  Returns a dict; on parse failure returns an error dict.
    """
    start = raw.find("{")
    end = raw.rfind("}") + 1
    if start < 0 or end <= start:
        return {
            "verdict": "NOT_SYNTHESIS",
            "reason": f"parse error: no JSON object found in response",
            "z_present": False,
            "z_integrative": False,
            "build_used_z": None,
            "_parse_error": True,
            "_raw": raw[:200],
        }
    try:
        return json.loads(raw[start:end])
    except json.JSONDecodeError as exc:
        return {
            "verdict": "NOT_SYNTHESIS",
            "reason": f"parse error: {exc}",
            "z_present": False,
            "z_integrative": False,
            "build_used_z": None,
            "_parse_error": True,
            "_raw": raw[start:start + 200],
        }
