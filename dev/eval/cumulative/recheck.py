#!/usr/bin/env python3
"""C7 lever-recheck recheck mode — run the REAL recall/please skill against a fixture whose disproving
note is buried by stub_engram, then measure whether the recommendation re-proposes the closed lever.

Two measures, kept separate (per the eval-end-metric-conflates-retrieval-vs-synthesis lesson):
  * OUTCOME  (lever_recheck_scorer): did the final recommendation reconcile or commit amnesia?
  * MECHANISM (the stub query log): did the skill ever issue a lever-keyed query that surfaced the
    disproof? `note_surfaced` separates a retrieval miss (the disproof was never returned to the skill)
    from a synthesis miss (returned but ignored). The current skill has no recall-before-recommend
    step, so when the lever is conceived AFTER its single upfront recall, the log shows no lever-keyed
    query and the disproof never surfaces — deterministic RED.

The live skill run reuses harness.claude (`claude -p`, ENGRAM_VAULT_PATH set, stub on PATH). The
extraction + scoring core (`recheck_result`) is pure and unit-tested offline; the live run is the only
paid/IO part.
"""
import json
import os
import re

import lever_recheck_scorer as scorer

HERE = os.path.dirname(os.path.abspath(__file__))
STUB = os.path.join(HERE, "lever_recheck", "stub_engram.py")


def read_stub_log(log_path):
    """Parse the stub query log into mechanism signals. Fails LOUD if the log is missing (a run that
    produced no log is a broken run, not a silent pass)."""
    if not os.path.isfile(log_path):
        raise FileNotFoundError(f"stub query log missing: {log_path!r} — the skill run wrote no queries")
    queries = [json.loads(line) for line in open(log_path) if line.strip()]
    return {
        "queries": queries,
        "n_queries": len(queries),
        "note_surfaced": any(q.get("returned_buried") for q in queries),
        "lever_query_issued": any(q.get("lever_keyed") for q in queries),
    }


_REC_RE = re.compile(r"RECOMMENDATION:\s*(.+)", re.IGNORECASE)


def extract_recommendation(agent_text):
    """Pull the recommendation out of the agent's final text. Falls back to the whole text so the
    scorer still sees the advocacy (fail-open on parsing, not on inputs)."""
    m = _REC_RE.search(agent_text or "")
    return m.group(1).strip() if m else (agent_text or "").strip()


def recheck_result(fixture_dir, agent_text, stub_log_path, stub=True, judge_model=None):
    """Pure core: given the agent's text + the stub log, produce the full C7 result. Unit-testable
    offline (no LLM when stub=True)."""
    mech = read_stub_log(stub_log_path)
    rec = extract_recommendation(agent_text)
    kwargs = {"stub": stub}
    if judge_model:
        kwargs["judge_model"] = judge_model
    scored = scorer.score_fixture(rec, fixture_dir, note_surfaced=mech["note_surfaced"], **kwargs)
    return {
        "fixture": os.path.basename(fixture_dir.rstrip("/")),
        "recommendation": rec,
        "cell_verdict": scored["cell_verdict"],
        "per_lever": scored["per_lever"],
        # mechanism diagnostics — never folded into the pass/fail
        "note_surfaced": mech["note_surfaced"],
        "lever_query_issued": mech["lever_query_issued"],
        "n_queries": mech["n_queries"],
    }


def write_stub_bin(bin_dir):
    """Write an `engram` shim into bin_dir that dispatches to stub_engram.py. Returns the bin dir."""
    os.makedirs(bin_dir, exist_ok=True)
    shim = os.path.join(bin_dir, "engram")
    with open(shim, "w") as fh:
        fh.write(f'#!/bin/bash\nexec python3 "{STUB}" "$@"\n')
    os.chmod(shim, 0o755)
    return bin_dir


def _stub_env(bin_dir, log_path, buried_basename, lever_terms):
    """Build the env overrides that put stub_engram on PATH ahead of the real binary and configure it."""
    return {
        "PATH": bin_dir + ":" + os.environ.get("PATH", ""),
        "STUB_ENGRAM_BURIED": buried_basename,
        "STUB_ENGRAM_LEVER_TERMS": lever_terms,
        "STUB_ENGRAM_LOG": log_path,
    }


def live_recall(fixture_dir, cfg, model, task, bin_dir, log_path, buried_basename, lever_terms,
                vault_subdir="vault_with_closed"):
    """Run the REAL skill via the canonical harness.claude() runner, with the stub injected onto PATH so
    the skill's `engram` calls hit it. Returns the agent's final text. (harness imported lazily so the
    offline core needs no harness deps.)"""
    import harness
    write_stub_bin(bin_dir)
    open(log_path, "w").close()  # truncate the log for this run
    extra_env = _stub_env(bin_dir, log_path, buried_basename, lever_terms)
    out = harness.claude(cfg=cfg, model=model, vault=os.path.join(fixture_dir, vault_subdir),
                         cwd=fixture_dir, prompt=task, extra_env=extra_env)
    if isinstance(out, dict):
        return out.get("result", "") or out.get("text", "") or json.dumps(out)
    return out or ""
