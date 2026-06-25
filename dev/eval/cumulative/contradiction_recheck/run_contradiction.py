#!/usr/bin/env python3
"""contradiction_recheck runner — run a fresh agent against a cell whose ASKED task and DEFERRED work are
both in plain context, then measure whether the recommendation does the asked task (RECONCILED) or
displaces it onto the deferred "more rigorous" work (CONTRADICTED).

This is the SYNTHESIS-displacement failure (vault note 87), distinct from lever_recheck's RETRIEVAL miss
(note 85): nothing is buried — the agent re-weights OLD reasoning mid-synthesis. Every live CONTRADICTED
in the validated run had new_evidence=false (an in-context synthesis failure, not retrieval).

The scoring core (`recheck_result`) is pure and unit-tested offline (stub=True => no LLM). The live runs
(`live_single`, `live_treadmill`) are the only paid/IO part; they reuse harness.claude() (`claude -p`),
captured exactly as recheck.py does its live_recall. (harness imported lazily so the offline core needs
no harness deps.)
"""
import json
import os
import re
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import contradiction_scorer as scorer  # noqa: E402

HERE = os.path.dirname(os.path.abspath(__file__))
PARENT = os.path.dirname(HERE)

_REC_RE = re.compile(r"(?:my\s+)?recommendation[:\s]", re.IGNORECASE)


def read_cell_prompt(cell_dir):
    """Build the live prompt for a cell from its context.md + task.txt. Fails LOUD if either is missing —
    a cell with no prompt is a broken cell, not a silent strawman."""
    ctx_path = os.path.join(cell_dir, "context.md")
    task_path = os.path.join(cell_dir, "task.txt")
    for path in (ctx_path, task_path):
        if not os.path.isfile(path):
            raise FileNotFoundError(f"cell prompt input missing: {path!r}")
    with open(ctx_path) as fh:
        context = fh.read().strip()
    with open(task_path) as fh:
        task = fh.read().strip()
    # task.txt is prefix(framing) + shared suffix; the materials go between them so the agent sees the
    # framing, then the materials, then the asked task.
    head, _, tail = task.partition("\n\n")
    return f"{head}\n\n{context}\n\n{tail}".strip()


def extract_recommendation(agent_text):
    """Pull the recommendation out of the agent's final text. Falls back to the whole text so the scorer
    still sees the displacement (fail-open on parsing, not on inputs)."""
    text = (agent_text or "").strip()
    m = _REC_RE.search(text)
    return text[m.start():].strip() if m else text


def recheck_result(cell_dir, agent_text, note_displaced=None, stub=True, judge_model=None):
    """Pure core: given a cell dir + the agent's final text, produce the contradiction result.
    Unit-testable offline (no LLM when stub=True)."""
    rec = extract_recommendation(agent_text)
    kwargs = {"stub": stub, "note_displaced": note_displaced}
    if judge_model:
        kwargs["judge_model"] = judge_model
    scored = scorer.score_cell(rec, cell_dir, **kwargs)
    scored["recommendation"] = rec
    return scored


def _agent_text(out):
    """Normalize harness.claude()'s return (dict with 'result'/'text', or a bare string) to text."""
    if isinstance(out, dict):
        return out.get("result", "") or out.get("text", "") or json.dumps(out)
    return out or ""


def live_single(cfg, model, cell_dir, stub=False, judge_model=None):
    """One-shot live run: build the cell prompt, call the agent once via harness.claude(), score the
    final text. Returns the recheck_result dict augmented with the raw agent text + session_id."""
    import harness  # lazy: offline core needs no harness deps
    prompt = read_cell_prompt(cell_dir)
    out = harness.claude(cfg=cfg, model=model, vault="none", cwd=cell_dir, prompt=prompt)
    text = _agent_text(out)
    result = recheck_result(cell_dir, text, stub=stub, judge_model=judge_model)
    result["agent_text"] = text
    result["session_id"] = out.get("session_id") if isinstance(out, dict) else None
    return result


def live_treadmill(cfg, model, cell_dir, turns, stub=False, judge_model=None):
    """Multi-turn live run: issue the cell prompt, then re-issue the same asked-task suffix for `turns`
    extra turns on the SAME session (resume_sid threaded from each result's session_id), scoring the
    FINAL turn's text. Models the 'forgot recent history' drift across a longer working session.

    Returns the final-turn recheck_result dict augmented with agent_text, session_id, and the per-turn
    session_id trail."""
    import harness  # lazy
    prompt = read_cell_prompt(cell_dir)
    # the re-issue is the shared asked-task suffix only (the tail after the framing/materials).
    suffix = prompt.rpartition("\n\n")[2].strip() or prompt
    sid = None
    text = ""
    trail = []
    total = max(1, turns)
    for turn in range(total):
        turn_prompt = prompt if turn == 0 else suffix
        out = harness.claude(cfg=cfg, model=model, vault="none", cwd=cell_dir,
                             prompt=turn_prompt, resume_sid=sid)
        text = _agent_text(out)
        sid = out.get("session_id") if isinstance(out, dict) else sid
        trail.append(sid)
    result = recheck_result(cell_dir, text, stub=stub, judge_model=judge_model)
    result["agent_text"] = text
    result["session_id"] = sid
    result["session_trail"] = trail
    result["turns"] = total
    return result


def read_canned(cell_dir):
    """Read a cell's hand-authored canned recommendation (pos_control). Fails LOUD if absent."""
    path = os.path.join(cell_dir, "canned_recommendation.txt")
    if not os.path.isfile(path):
        raise FileNotFoundError(f"canned_recommendation.txt missing in cell {cell_dir!r}")
    with open(path) as fh:
        return fh.read().strip()
