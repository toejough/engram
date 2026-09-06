#!/usr/bin/env python3
"""
Tier 1 of the memory-loop-audit eval-framework completion (Joe-approved
2026-09-05): two measurements over the 98 no-search moments, batched into ONE
sonnet judgment per moment. Real, billed claude -p calls -- no stubs.

Why: F1's existence check fires on ANY non-empty query result (no relevance
bar), so "98/150 moments didn't search despite memory existing" has a soft
denominator. This run hardens it:

  Measurement A (relevance): for each no-search moment, re-run a REAL
  point-in-time retrieval (vault worktree at the at-or-before commit,
  ingestion-date-filtered chunks, isolated engram query) with fresh
  haiku-generated phrases, then have sonnet grade whether the top-5 retrieved
  items contain something that GENUINELY bears on the moment
  (relevant_top1 / relevant_in_top5 / nothing_relevant). Judged honestly, not
  charitably: a generic lesson that technically overlaps but wouldn't have
  changed anything is NOT relevant.

  Measurement B (fire-cue decomposition, vault note 824's discipline): the
  same sonnet call classifies WHY no search fired -- cue_present_missed (a
  current recall-guidance cue plainly applied; which one), no_cue_exists
  (propose a one-line candidate cue), or not_a_recall_moment (searching would
  not have been a sensible act at all).

Target population, derived fresh from results/moments.jsonl (read-only): all
moments with finding_side.search_ran == "none" -- expected exactly 98. The
run refuses to start against an unexpected population.

Context per moment: the covering window via split_into_windows when the
transcript still exists (judged_from="transcript"), else the full chunk-index
reconstruction (judged_from="chunks"; the established fallback), capped with
an honest, marked truncation.

Per-target handling (same conventions as rejudge_all_nulls.py):
- prompt_too_long from claude -p: skipped (deterministic, never retried).
- malformed/invalid-enum/inconsistent response: ONE retry, then recorded with
  skipped_reason="unparseable response" -- never invented.
- exhausted_retries from claude -p: the ENTIRE run halts immediately (raise
  ExhaustedRetriesError; the 2026-09-02 silent-auth-outage lesson).
- any other per-moment error: caught, recorded as that moment's
  skipped_reason (one bad moment must not kill the sweep).

Isolation is mandatory: ONLY the sanctioned worktree commands touch the live
vault (_checkout_vault_at is self-cleaning), engram query runs through
isolation.assert_engram_isolated inside _run_engram_query_at_moment, and
scratch dirs are removed in finally.

Output (the ONLY files this script writes under results/):
- results/relevance-cues.jsonl -- one JSON line per target:
  {"moment_id","moment_type","judged_from","search_phrases",
   "top5":[{path,kind,score}],"relevance_grade","relevant_path",
   "relevance_rationale","fire_cue","cue_name","candidate_cue",
   "cue_rationale","skipped_reason","cost_usd"}
  cost_usd is the real total_cost_usd claude -p reported for that moment's
  call(s) -- phrase generation, judgment, and retries included -- or null
  when no call was made.
- results/fire-unit-estimate.json -- the MECHANICAL (no-LLM) fire-unit
  estimate, written once all targets are recorded: fires-per-unit counts for
  the three candidate fire-units (task-init, per-failure-event,
  per-dispatch-event), hit rates from measurement B's tallies, and over-fire
  ratios (fires / hits), every number labeled DERIVED or ESTIMATE with the
  sample-extrapolation caveat stated.

The script is RESUMABLE: on start it reads any moment_ids already present in
the output file and skips them (append mode). The final summary + fire-unit
estimate print/write only once all targets are present.

Usage: python3 measure_relevance_cues.py [max_seconds]
  max_seconds (optional): stop cleanly BETWEEN targets once this much wall
  time has elapsed (prints the incomplete-resume notice). Lets a driver with
  a hard tool timeout run the script in foreground chunks without ever
  killing an in-flight claude -p call; omitted = run to completion.
"""

import json
import os
import shutil
import sys
import tempfile
import time
from collections import Counter
from pathlib import Path
from typing import Any, Dict, List, Optional, Tuple

_SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, _SCRIPT_DIR)

from audit_moments import (  # noqa: E402  (path insert must precede this import)
    FLAT_VAULT_MIGRATION_COMMIT,
    ExhaustedRetriesError,
    PromptTooLongError,  # noqa: F401  (documented contract; prompt_too_long arrives as a response flag)
    _build_cfg_dir,
    _checkout_vault_at,
    _filter_chunks_by_ingestion_date,
    _find_commit_at_or_before,
    _is_commit_ancestor_of,
    _run_claude_p,
    _run_engram_query_at_moment,
    split_into_windows,
)
from rejudge_from_chunks import (  # noqa: E402
    _find_chunk_file,
    _reconstruct_from_chunks,
)

MOMENTS_PATH = Path(_SCRIPT_DIR) / "results" / "moments.jsonl"
EVENTS_PATH = Path(_SCRIPT_DIR) / "results" / "transcript-events.jsonl"
OUTPUT_PATH = Path(_SCRIPT_DIR) / "results" / "relevance-cues.jsonl"
FIRE_UNIT_PATH = Path(_SCRIPT_DIR) / "results" / "fire-unit-estimate.json"
VAULT_DIR = os.path.expanduser("~/.local/share/engram/vault")
CHUNKS_DIR = os.path.expanduser("~/.local/share/engram/chunks")

EXPECTED_TARGET_COUNT = 98  # verified census: search_ran == "none" moments

SKIP_NO_WINDOW = "no window covers moment location"
SKIP_TOO_LONG = "prompt too long for claude -p"
SKIP_UNPARSEABLE = "unparseable response"

TOP_K = 5
NOTE_CONTENT_CAP = 1200
CHUNK_TEXT_CAP = 250_000  # same measured claude -p input-limit headroom as rejudge_all_nulls
PHRASE_GEN_CONTEXT_CAP = 30_000

FAILURE_ISH_TYPES = {"failure", "rework", "correction"}

RELEVANCE_GRADES = ("relevant_top1", "relevant_in_top5", "nothing_relevant")
FIRE_CUES = ("cue_present_missed", "no_cue_exists", "not_a_recall_moment")

# The current recall-guidance firing cues (from ~/.claude/engram/recall.md),
# keyed by the enum value the judge must answer in cue_name.
CUE_NAMES = {
    "task_init_recall": (
        "recall at task initiation -- a fresh user request/task began and recall should "
        "have run before starting the work"
    ),
    "approach_discussion": (
        "before endorsing/ranking/even just chatting about a proposed approach"
    ),
    "pre_completion": "before declaring work done",
    "unexplained_failure": (
        "after a failure the agent could not immediately explain, before guessing at causes"
    ),
    "pre_build": (
        "before starting to build a new approach, while the path was still cheap to change"
    ),
}

OUTPUT_FIELD_DEFAULTS = {
    "search_phrases": None,
    "top5": None,
    "relevance_grade": None,
    "relevant_path": None,
    "relevance_rationale": None,
    "fire_cue": None,
    "cue_name": None,
    "candidate_cue": None,
    "cue_rationale": None,
    "skipped_reason": None,
    "cost_usd": None,
}

JUDGMENT_KEYS = {
    "relevance_grade",
    "relevant_path",
    "relevance_rationale",
    "fire_cue",
    "cue_name",
    "candidate_cue",
    "cue_rationale",
}
# The model may legitimately OMIT a key whose value would be null (observed on
# the first real call: "candidate_cue" dropped when fire_cue was
# cue_present_missed) -- an omitted optional key means null. Required keys must
# be present; keys outside JUDGMENT_KEYS are still rejected.
JUDGMENT_REQUIRED_KEYS = {
    "relevance_grade",
    "relevance_rationale",
    "fire_cue",
    "cue_rationale",
}


# ---------------------------------------------------------------------------
# Target derivation + preflight
# ---------------------------------------------------------------------------


def _load_jsonl(path: Path) -> List[Dict[str, Any]]:
    records: List[Dict[str, Any]] = []
    with open(path, "r") as f:
        for line in f:
            line = line.strip()
            if line:
                records.append(json.loads(line))
    return records


def _load_targets() -> List[Dict[str, Any]]:
    """All moments with finding_side.search_ran == 'none', in moments.jsonl order."""
    return [
        m
        for m in _load_jsonl(MOMENTS_PATH)
        if (m.get("finding_side") or {}).get("search_ran") == "none"
    ]


def _preflight(targets: List[Dict[str, Any]]) -> Dict[str, Dict[str, Any]]:
    """
    Verify -- BEFORE any money is spent -- that the derived population is
    exactly the expected census and that every target's context AND
    point-in-time DERIVED retrieval are actually resolvable (fail loud, never
    silently fall back). Returns per-moment_id prep: judged_from + the
    resolved at-or-before vault commit.
    """
    if len(targets) != EXPECTED_TARGET_COUNT:
        raise ValueError(
            f"expected exactly {EXPECTED_TARGET_COUNT} search_ran=='none' moments, "
            f"found {len(targets)} -- refusing to run against an unexpected population"
        )

    commit_cache: Dict[str, str] = {}
    prep: Dict[str, Dict[str, Any]] = {}

    for target in targets:
        moment_id = target["moment_id"]
        transcript_path = target["transcript_path"]

        if os.path.exists(transcript_path):
            judged_from = "transcript"
        else:
            judged_from = "chunks"
            _find_chunk_file(transcript_path)  # fail loud now, not mid-run

        timestamp = target.get("timestamp")
        if not timestamp:
            raise ValueError(f"{moment_id}: no timestamp -- point-in-time check impossible")
        if timestamp not in commit_cache:
            sha = _find_commit_at_or_before(VAULT_DIR, timestamp)
            if not sha:
                raise ValueError(f"{moment_id}: no vault commit at or before {timestamp}")
            if not _is_commit_ancestor_of(VAULT_DIR, FLAT_VAULT_MIGRATION_COMMIT, sha):
                raise ValueError(
                    f"{moment_id}: commit {sha} at {timestamp} predates the flat-vault "
                    "migration -- DERIVED-mode retrieval unavailable"
                )
            commit_cache[timestamp] = sha

        prep[moment_id] = {"judged_from": judged_from, "commit_sha": commit_cache[timestamp]}
        print(
            f"preflight: {moment_id} judged_from={judged_from} "
            f"commit={commit_cache[timestamp]}",
            flush=True,
        )

    return prep


# ---------------------------------------------------------------------------
# Context resolution
# ---------------------------------------------------------------------------


def _find_covering_window(windows: List[Dict[str, Any]], location: int) -> Optional[Dict[str, Any]]:
    for w in windows:
        if w["location_start"] <= location <= w["location_end"]:
            return w
    return None


def _numbered_window_text(window: Dict[str, Any]) -> str:
    """Window text with absolute line numbers (same scheme as the original audit)."""
    numbered = []
    for offset, line in enumerate(window["lines"]):
        numbered.append(f"{window['location_start'] + offset}: {line}")
    return "".join(numbered)


def _capped_chunk_text(chunk_text: str, cap: int = CHUNK_TEXT_CAP) -> str:
    """Head-keep truncation with an honest marker when the reconstruction exceeds cap."""
    if len(chunk_text) <= cap:
        return chunk_text
    return (
        chunk_text[:cap]
        + f"\n\n[TRUNCATED: showing the first {cap:,} of {len(chunk_text):,} characters of the "
        "reconstruction -- the tail was cut to fit the model's input limit]"
    )


def _centered_excerpt(window: Dict[str, Any], location: int, cap: int) -> str:
    """
    An excerpt of the window's numbered lines centered on the moment's
    location, up to ~cap characters -- used only for the cheap phrase-
    generation call, so the phrases come from text AROUND the moment rather
    than an arbitrary head-slice of a possibly 150 KB window.
    """
    lines = window["lines"]
    center = max(0, min(len(lines) - 1, location - window["location_start"]))
    chosen = {center}
    total = len(lines[center])
    step = 1
    while total < cap:
        added = False
        for idx in (center - step, center + step):
            if 0 <= idx < len(lines) and idx not in chosen:
                if total + len(lines[idx]) > cap:
                    continue
                chosen.add(idx)
                total += len(lines[idx])
                added = True
        if not added:
            break
        step += 1
    ordered = sorted(chosen)
    return "".join(
        f"{window['location_start'] + idx}: {lines[idx]}" for idx in ordered
    )


# ---------------------------------------------------------------------------
# Phrase generation (one cheap haiku call per moment)
# ---------------------------------------------------------------------------

PHRASE_GEN_PROMPT = """You are helping re-run a point-in-time memory-relevance check for a memory-loop audit.

A moment from a real agent session needs 2-4 search phrases that capture what a relevant \
memory/lesson for that moment would have been about. The phrases will query a memory vault as it \
existed at the moment's timestamp, to answer: would a search at that moment have retrieved \
something genuinely useful?

MOMENT:
- moment_type: {moment_type}
- description: {description}

SESSION CONTEXT (an excerpt around the moment; may be truncated):
{context}

Respond with STRICT JSON only -- no prose before or after:
{{"search_phrases": ["<phrase>", "<phrase>", ...]}}
Give 2-4 phrases, each at least 2 words, concrete and content-bearing (what the lesson/memory \
would be about), not generic filler."""


def _strip_fences(result_text: str) -> str:
    json_text = result_text
    if "```json" in json_text:
        json_text = json_text.split("```json")[1].split("```")[0]
    elif "```" in json_text:
        json_text = json_text.split("```")[1].split("```")[0]
    return json_text.strip()


def _generate_search_phrases(
    moment: Dict[str, Any], phrase_context: str, cfg_dir: str
) -> Tuple[Optional[List[str]], float, Optional[str]]:
    """
    One cheap claude -p haiku call generating 2-4 search phrases. One retry on
    malformed output. Returns (phrases or None, spend, failure_reason or None).
    """
    prompt = PHRASE_GEN_PROMPT.format(
        moment_type=moment["moment_type"],
        description=moment["description"],
        context=phrase_context[:PHRASE_GEN_CONTEXT_CAP],
    )
    spend = 0.0
    for attempt in (1, 2):
        response = _run_claude_p(prompt, "haiku", cfg_dir=cfg_dir)
        spend += response.get("total_cost_usd", 0) or 0
        if response.get("exhausted_retries"):
            raise ExhaustedRetriesError(
                "claude -p exhausted retries during phrase generation -- halting the run"
            )
        if response.get("prompt_too_long"):
            return None, spend, "phrase-generation prompt too long for claude -p"
        try:
            parsed = json.loads(_strip_fences(response.get("result", "")))
        except json.JSONDecodeError:
            parsed = None
        if isinstance(parsed, dict):
            raw = parsed.get("search_phrases")
            if isinstance(raw, list):
                phrases = [p for p in raw if isinstance(p, str) and len(p.split()) >= 2][:4]
                if len(phrases) >= 2:
                    return phrases, spend, None
        if attempt == 1:
            print("  phrase generation: malformed response; retrying once ...", flush=True)
    return None, spend, "phrase generation returned no usable phrases after one retry"


# ---------------------------------------------------------------------------
# Point-in-time retrieval (DERIVED mode, sanctioned helpers only)
# ---------------------------------------------------------------------------


def _point_in_time_top5(
    commit_sha: str, timestamp: str, search_phrases: List[str]
) -> Tuple[Optional[List[Dict[str, Any]]], Optional[str]]:
    """
    The DERIVED-mode point-in-time retrieval: vault worktree at commit_sha,
    ingestion-date-filtered chunks, isolated engram query. Captures the top
    TOP_K items -- path, kind, score, and (for note items) content read from
    the checked-out worktree, truncated to NOTE_CONTENT_CAP chars.

    Returns (top5 or None, error or None). Scratch dirs cleaned in finally.
    """
    scratch_dir = tempfile.mkdtemp(prefix="measure-relevance-cues-ptc-")
    try:
        with _checkout_vault_at(VAULT_DIR, commit_sha, scratch_dir) as vault_path:
            if os.path.exists(CHUNKS_DIR):
                chunks_path = _filter_chunks_by_ingestion_date(
                    CHUNKS_DIR, timestamp, scratch_dir
                )
            else:
                chunks_path = os.path.join(scratch_dir, "chunks-empty")
                os.makedirs(chunks_path, exist_ok=True)

            query_result = _run_engram_query_at_moment(vault_path, chunks_path, search_phrases)
            items = query_result.get("items")
            if items is None:
                return None, query_result.get("error", "Unknown error")

            top5: List[Dict[str, Any]] = []
            for item in items[:TOP_K]:
                entry: Dict[str, Any] = {
                    "path": item.get("path"),
                    "kind": item.get("kind"),
                    "score": item.get("score"),
                }
                # Content read INSIDE the worktree context (it is gone after).
                if entry["kind"] != "chunk" and entry["path"]:
                    try:
                        with open(os.path.join(vault_path, entry["path"]), "r") as f:
                            content = f.read()
                        if len(content) > NOTE_CONTENT_CAP:
                            content = (
                                content[:NOTE_CONTENT_CAP]
                                + f"\n[truncated at {NOTE_CONTENT_CAP} chars]"
                            )
                        entry["content"] = content
                    except (IOError, OSError) as read_error:
                        entry["content"] = f"[note content unreadable: {read_error}]"
                else:
                    entry["content"] = "(chunk item -- raw session-chunk text, content not shown)"
                top5.append(entry)
            return top5, None
    finally:
        shutil.rmtree(scratch_dir, ignore_errors=True)


# ---------------------------------------------------------------------------
# Judgment prompt + parsing
# ---------------------------------------------------------------------------


def _retrieval_block(search_phrases: List[str], top5: List[Dict[str, Any]]) -> str:
    lines = [
        "WHAT A POINT-IN-TIME MEMORY SEARCH WOULD HAVE RETURNED at this moment's timestamp "
        f"(top {len(top5)} items, ranked; the vault and chunk index were reconstructed exactly "
        "as they existed at that time):",
        "Search phrases used: " + "; ".join(search_phrases),
        "",
    ]
    if not top5:
        lines.append(
            "NO ITEMS RETURNED -- the point-in-time search found nothing for these phrases."
        )
        return "\n".join(lines)
    for rank, item in enumerate(top5, start=1):
        lines.append(
            f"--- rank {rank}: {item['path']} (kind={item['kind']}, score={item['score']}) ---"
        )
        lines.append(item["content"])
        lines.append("")
    return "\n".join(lines)


def _cue_list_block() -> str:
    return "\n".join(f'  - "{key}": {desc}' for key, desc in CUE_NAMES.items())


def _build_judgment_prompt(
    context_intro: str,
    context_body: str,
    retrieval_block: str,
    location_note: str,
    moment_type: str,
    description: str,
    has_items: bool,
) -> str:
    no_items_rule = (
        ""
        if has_items
        else (
            '\nThe retrieval returned NO items, so "relevance_grade" MUST be '
            '"nothing_relevant" with "relevant_path": null.'
        )
    )
    return f"""You are auditing one moment from a real agent-session transcript for a memory-loop audit. \
At this moment the agent ran NO memory search, even though the memory vault existed. A point-in-time \
retrieval has been re-run for you (below). Judge two things: (A) whether anything genuinely relevant \
would have been retrieved, and (B) why no search fired.

{context_intro}
{context_body}

{retrieval_block}

MOMENT UNDER JUDGMENT:
{location_note}
- moment_type: {moment_type}
- description: {description}

QUESTION A -- "relevance_grade": would the retrieved memory have genuinely helped at THIS moment?
  - "relevant_top1": the rank-1 item genuinely bears on this moment -- it would have informed or \
changed what the agent did. (If rank 1 bears, answer this even if lower items also bear.)
  - "relevant_in_top5": rank 1 does not bear, but a lower-ranked item (rank 2-{TOP_K}) does.
  - "nothing_relevant": none of the retrieved items would have helped here.
Judge honestly against the moment, not charitably -- a generic lesson that technically overlaps but \
would not have changed anything is NOT relevant. Also answer "relevant_path": the exact path of the \
item that bears (copied VERBATIM from the ranked list above), or null when nothing_relevant; and \
"relevance_rationale": one sentence.{no_items_rule}

QUESTION B -- "fire_cue": why did no memory search fire at this moment? The current recall guidance \
names these firing cues:
{_cue_list_block()}
Answer one of:
  - "cue_present_missed": one of those cues PLAINLY applied at this moment (a search should have \
fired under the current guidance) -- name which one in "cue_name" (exactly one of: \
{" | ".join(CUE_NAMES)}).
  - "no_cue_exists": no current cue covers this moment type, but searching here WOULD have been a \
sensible act -- propose a one-line "candidate_cue" describing the missing cue.
  - "not_a_recall_moment": searching here would not have been a sensible act at all (e.g. \
mid-mechanical-execution with nothing to decide).
Also answer "cue_rationale": one sentence. When fire_cue is not "cue_present_missed", "cue_name" \
must be null; when fire_cue is not "no_cue_exists", "candidate_cue" must be null.

Respond with STRICT JSON only -- no prose before or after, EXACTLY these keys:
{{"relevance_grade": "relevant_top1" | "relevant_in_top5" | "nothing_relevant", \
"relevant_path": "<path>" | null, "relevance_rationale": "<one sentence>", \
"fire_cue": "cue_present_missed" | "no_cue_exists" | "not_a_recall_moment", \
"cue_name": "<cue>" | null, "candidate_cue": "<one line>" | null, \
"cue_rationale": "<one sentence>"}}"""


def _parse_judgment(
    result_text: str, top5: List[Dict[str, Any]]
) -> Tuple[bool, Optional[Dict[str, Any]]]:
    """
    Parse and validate the judgment. Returns (ok, fields). ok=False means the
    response was malformed/invalid (bad JSON, wrong key set, out-of-enum
    value, missing rationale, or a relevant_path inconsistent with the ranked
    list) and the caller should retry once, then skip. Never coerced.
    """
    stripped = _strip_fences(result_text)
    try:
        parsed = json.loads(stripped)
    except json.JSONDecodeError:
        # Lenience fallback: extract the outermost {...} block (the model
        # sometimes wraps the JSON in prose). Still strictly validated below.
        start, end = stripped.find("{"), stripped.rfind("}")
        if start == -1 or end <= start:
            return False, None
        try:
            parsed = json.loads(stripped[start : end + 1])
        except json.JSONDecodeError:
            return False, None
    if not isinstance(parsed, dict):
        return False, None
    keys = set(parsed.keys())
    if not JUDGMENT_REQUIRED_KEYS <= keys or not keys <= JUDGMENT_KEYS:
        return False, None
    parsed = {**{key: None for key in JUDGMENT_KEYS}, **parsed}

    grade = parsed["relevance_grade"]
    if grade not in RELEVANCE_GRADES:
        return False, None
    top5_paths = [item["path"] for item in top5]
    relevant_path = parsed["relevant_path"]
    if grade == "nothing_relevant":
        if relevant_path is not None:
            return False, None
    elif grade == "relevant_top1":
        if not top5_paths or relevant_path != top5_paths[0]:
            return False, None
    else:  # relevant_in_top5
        if relevant_path not in top5_paths[1:]:
            return False, None
    if not top5 and grade != "nothing_relevant":
        return False, None

    fire_cue = parsed["fire_cue"]
    if fire_cue not in FIRE_CUES:
        return False, None
    cue_name = parsed["cue_name"]
    candidate_cue = parsed["candidate_cue"]
    if fire_cue == "cue_present_missed":
        if cue_name not in CUE_NAMES:
            return False, None
    elif cue_name is not None:
        return False, None
    if fire_cue == "no_cue_exists":
        if not isinstance(candidate_cue, str) or not candidate_cue.strip():
            return False, None
        candidate_cue = candidate_cue.strip()
    elif candidate_cue is not None:
        return False, None

    for rationale_key in ("relevance_rationale", "cue_rationale"):
        rationale = parsed[rationale_key]
        if not isinstance(rationale, str) or not rationale.strip():
            return False, None

    return True, {
        "relevance_grade": grade,
        "relevant_path": relevant_path,
        "relevance_rationale": parsed["relevance_rationale"].strip(),
        "fire_cue": fire_cue,
        "cue_name": cue_name,
        "candidate_cue": candidate_cue,
        "cue_rationale": parsed["cue_rationale"].strip(),
    }


# ---------------------------------------------------------------------------
# Mechanical fire-unit estimate (no LLM)
# ---------------------------------------------------------------------------


def _fire_unit_entry(
    fires_sample: float,
    fires_sample_provenance: str,
    fires_corpus: float,
    fires_corpus_provenance: str,
    hits_sample: int,
    hits_definition: str,
    extrapolation_factor: float,
) -> Dict[str, Any]:
    hits_corpus_est = hits_sample * extrapolation_factor
    return {
        "fires_sample": fires_sample,
        "fires_sample_provenance": fires_sample_provenance,
        "fires_corpus": fires_corpus,
        "fires_corpus_provenance": fires_corpus_provenance,
        "hits_sample": hits_sample,
        "hits_definition": hits_definition,
        "hit_rate_sample": (hits_sample / fires_sample) if fires_sample else None,
        "over_fire_sample": (fires_sample / hits_sample) if hits_sample else None,
        "hits_corpus_estimate": hits_corpus_est,
        "over_fire_corpus_estimate": (fires_corpus / hits_corpus_est) if hits_corpus_est else None,
    }


def _compute_fire_unit_estimate() -> Dict[str, Any]:
    """
    MECHANICAL fire-unit estimate (vault note 824's discipline: pin the
    fire-unit, derive the denominator from a real corpus count, label every
    number DERIVED or ESTIMATE). No LLM calls -- pure counting over
    transcript-events.jsonl, moments.jsonl, and this run's output.
    """
    moments = _load_jsonl(MOMENTS_PATH)
    events = _load_jsonl(EVENTS_PATH)
    records = _load_jsonl(OUTPUT_PATH)

    moments_by_id = {m["moment_id"]: m for m in moments}
    judged = [r for r in records if r["skipped_reason"] is None]
    cue_missed = [r for r in judged if r["fire_cue"] == "cue_present_missed"]

    audited_basenames = {os.path.basename(m["transcript_path"]) for m in moments}
    n_sample_transcripts = len(audited_basenames)  # DERIVED: 54
    n_corpus_transcripts = len(events)  # DERIVED: 568
    extrapolation_factor = n_corpus_transcripts / n_sample_transcripts

    dispatch_fires_corpus = sum(len(r.get("dispatch_events") or []) for r in events)
    dispatch_fires_sample = sum(
        len(r.get("dispatch_events") or [])
        for r in events
        if os.path.basename(r["transcript_path"]) in audited_basenames
    )
    failure_ish_sample = sum(1 for m in moments if m["moment_type"] in FAILURE_ISH_TYPES)

    cue_name_tally = dict(Counter(r["cue_name"] for r in cue_missed))
    task_init_hits = sum(1 for r in cue_missed if r["cue_name"] == "task_init_recall")
    failure_hits = sum(1 for r in cue_missed if r["cue_name"] == "unexplained_failure")
    dispatch_hits = sum(
        1
        for r in cue_missed
        if moments_by_id.get(r["moment_id"], {}).get("moment_type") == "dispatch"
    )

    estimate = {
        "label": "ESTIMATE",
        "caveat": (
            "Hit counts come from the AUDITED SAMPLE only: 150 detected moments across "
            f"{n_sample_transcripts} audited transcripts, of which {len(records)} no-search "
            f"moments were measured here ({len(judged)} judged, {len(records) - len(judged)} "
            "skipped). Corpus-wide hit numbers extrapolate the per-transcript sample rate to "
            f"the {n_corpus_transcripts}-transcript corpus (x{extrapolation_factor:.2f}) and "
            "assume the audited transcripts are representative -- they are labeled ESTIMATE, "
            "not measured. Detected moments are themselves a lower bound (the moment detector "
            "is not exhaustive). Per-failure-event fires corpus-wide are also extrapolated "
            "(moments exist only for the audited sample)."
        ),
        "corpus": {
            "transcripts": n_corpus_transcripts,
            "transcripts_provenance": "DERIVED (transcript-events.jsonl record count)",
            "dispatch_events": dispatch_fires_corpus,
            "dispatch_events_provenance": "DERIVED (sum of dispatch_events lengths)",
        },
        "audited_sample": {
            "transcripts": n_sample_transcripts,
            "moments": len(moments),
            "no_search_moments_measured": len(records),
            "judged": len(judged),
            "skipped": len(records) - len(judged),
            "cue_present_missed": len(cue_missed),
            "failure_ish_moments": failure_ish_sample,
            "dispatch_events_in_sample": dispatch_fires_sample,
        },
        "cue_name_tally_among_cue_present_missed": cue_name_tally,
        "fire_units": {
            "task_init": _fire_unit_entry(
                fires_sample=n_sample_transcripts,
                fires_sample_provenance="DERIVED (one fire per audited transcript)",
                fires_corpus=n_corpus_transcripts,
                fires_corpus_provenance="DERIVED (one fire per corpus transcript)",
                hits_sample=task_init_hits,
                hits_definition=(
                    "cue_present_missed moments with cue_name == task_init_recall"
                ),
                extrapolation_factor=extrapolation_factor,
            ),
            "per_failure_event": _fire_unit_entry(
                fires_sample=failure_ish_sample,
                fires_sample_provenance=(
                    "DERIVED (failure/rework/correction moments in the 150-moment sample; "
                    "itself a lower bound on real failure events)"
                ),
                fires_corpus=failure_ish_sample * extrapolation_factor,
                fires_corpus_provenance=(
                    f"ESTIMATE (sample rate x {extrapolation_factor:.2f} -- no corpus-wide "
                    "failure-event count exists)"
                ),
                hits_sample=failure_hits,
                hits_definition=(
                    "cue_present_missed moments with cue_name == unexplained_failure"
                ),
                extrapolation_factor=extrapolation_factor,
            ),
            "per_dispatch_event": _fire_unit_entry(
                fires_sample=dispatch_fires_sample,
                fires_sample_provenance=(
                    "DERIVED (dispatch_events in the audited transcripts, "
                    "transcript-events.jsonl)"
                ),
                fires_corpus=dispatch_fires_corpus,
                fires_corpus_provenance="DERIVED (dispatch_events corpus-wide)",
                hits_sample=dispatch_hits,
                hits_definition=(
                    "cue_present_missed moments of moment_type == dispatch (no dispatch-"
                    "specific cue exists in the current guidance; this counts missed-cue "
                    "moments a dispatch-time fire would reach)"
                ),
                extrapolation_factor=extrapolation_factor,
            ),
        },
    }
    return estimate


def _print_fire_unit_table(estimate: Dict[str, Any]) -> None:
    print()
    print("=== FIRE-UNIT ESTIMATE (mechanical; label: ESTIMATE) ===")
    print(estimate["caveat"])
    print()
    header = (
        f"{'unit':<20} {'fires(sample)':>13} {'fires(corpus)':>13} {'hits(sample)':>12} "
        f"{'hit-rate':>9} {'over-fire(sample)':>17} {'over-fire(corpus est)':>21}"
    )
    print(header)
    print("-" * len(header))
    for name, entry in estimate["fire_units"].items():
        hit_rate = entry["hit_rate_sample"]
        over_sample = entry["over_fire_sample"]
        over_corpus = entry["over_fire_corpus_estimate"]
        print(
            f"{name:<20} {entry['fires_sample']:>13.0f} {entry['fires_corpus']:>13.0f} "
            f"{entry['hits_sample']:>12d} "
            f"{(f'{hit_rate:.3f}' if hit_rate is not None else 'n/a'):>9} "
            f"{(f'{over_sample:.1f}x' if over_sample is not None else 'n/a (0 hits)'):>17} "
            f"{(f'{over_corpus:.1f}x' if over_corpus is not None else 'n/a (0 hits)'):>21}"
        )
    print()
    print(
        "cue_name tally among cue_present_missed: "
        + json.dumps(estimate["cue_name_tally_among_cue_present_missed"])
    )


# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------


def _load_done_ids(path: Path) -> set:
    if not path.exists():
        return set()
    return {r["moment_id"] for r in _load_jsonl(path)}


def _print_final_summary() -> None:
    records = _load_jsonl(OUTPUT_PATH)
    judged = [r for r in records if r["skipped_reason"] is None]
    skipped = [r for r in records if r["skipped_reason"] is not None]
    total_cost = sum(r["cost_usd"] for r in records if r["cost_usd"] is not None)

    print()
    print("=== FINAL SUMMARY ===")
    print(f"Targets: {len(records)}/{EXPECTED_TARGET_COUNT} recorded in {OUTPUT_PATH.name}")
    print(f"Judged: {len(judged)}; skipped: {len(skipped)}")
    for reason, count in sorted(Counter(r["skipped_reason"] for r in skipped).items()):
        print(f"  skipped ({reason}): {count}")

    print("relevance_grade tally (judged):")
    for value, count in sorted(Counter(r["relevance_grade"] for r in judged).items()):
        print(f"  {value}: {count}")
    print("fire_cue tally (judged):")
    for value, count in sorted(Counter(r["fire_cue"] for r in judged).items()):
        print(f"  {value}: {count}")
    print("cue_name tally (cue_present_missed):")
    for value, count in sorted(
        Counter(
            r["cue_name"] for r in judged if r["fire_cue"] == "cue_present_missed"
        ).items()
    ):
        print(f"  {value}: {count}")
    print("JOINT distribution relevance_grade x fire_cue (judged):")
    joint = Counter((r["relevance_grade"], r["fire_cue"]) for r in judged)
    for (grade, cue), count in sorted(joint.items()):
        print(f"  {grade} x {cue}: {count}")
    print(
        f"Total real $ cost: ${total_cost:.4f} (sum of per-target cost_usd -- each the "
        "total_cost_usd claude -p's own --output-format json response reported for that "
        "target's real call(s), across ALL invocations of this script; not an estimate)"
    )


def main() -> None:
    max_seconds = float(sys.argv[1]) if len(sys.argv) > 1 else None
    started = time.monotonic()

    targets = _load_targets()
    prep = _preflight(targets)

    done_ids = _load_done_ids(OUTPUT_PATH)

    # ONE shared cfg_dir (real keychain-seeded creds) reused across all calls.
    cfg_dir = tempfile.mkdtemp(prefix="measure-relevance-cues-cfg-")
    _build_cfg_dir(cfg_dir)

    # Cache windows / chunk reconstructions per transcript (deterministic pure reads).
    windows_cache: Dict[str, List[Dict[str, Any]]] = {}
    reconstruction_cache: Dict[str, str] = {}

    invocation_spend = 0.0

    try:
        with open(OUTPUT_PATH, "a") as out_f:

            def emit(record: Dict[str, Any]) -> None:
                out_f.write(json.dumps(record) + "\n")
                out_f.flush()
                done_ids.add(record["moment_id"])

            for index, moment in enumerate(targets, start=1):
                moment_id = moment["moment_id"]
                if moment_id in done_ids:
                    continue  # resume: already recorded by a prior invocation

                if max_seconds is not None and time.monotonic() - started > max_seconds:
                    print(
                        f"time budget ({max_seconds:.0f}s) reached; stopping between targets",
                        flush=True,
                    )
                    break

                target_prep = prep[moment_id]
                skeleton: Dict[str, Any] = {
                    "moment_id": moment_id,
                    "moment_type": moment["moment_type"],
                    "judged_from": target_prep["judged_from"],
                    **OUTPUT_FIELD_DEFAULTS,
                }
                print(
                    f"[{index}/{len(targets)}] {moment_id} "
                    f"(judged_from={target_prep['judged_from']}, "
                    f"type={moment['moment_type']}) ...",
                    flush=True,
                )

                target_cost = 0.0
                try:
                    outcome = dict(skeleton)
                    transcript_path = moment["transcript_path"]
                    location = moment["location"]

                    # -------- 1. CONTEXT --------
                    if target_prep["judged_from"] == "chunks":
                        if transcript_path not in reconstruction_cache:
                            reconstruction_cache[transcript_path] = _reconstruct_from_chunks(
                                _find_chunk_file(transcript_path)
                            )
                        context_intro = (
                            "The original transcript file was deleted by session retention. "
                            "The record below is reconstructed from engram's chunk index -- "
                            "stripped conversational turns, tool-call detail reduced, the "
                            "original line numbering is unavailable. It is the surviving "
                            "conversational record for this session, in turn order.\n\n"
                            "RECONSTRUCTED TRANSCRIPT (from engram's chunk index):"
                        )
                        context_body = _capped_chunk_text(reconstruction_cache[transcript_path])
                        location_note = (
                            f"- approximate original location: line {location} of the original "
                            "transcript JSONL (that numbering does not map onto the "
                            "reconstruction above; use the description to find the moment)"
                        )
                        phrase_context = reconstruction_cache[transcript_path][
                            :PHRASE_GEN_CONTEXT_CAP
                        ]
                    else:
                        if transcript_path not in windows_cache:
                            windows_cache[transcript_path] = split_into_windows(transcript_path)
                        window = _find_covering_window(windows_cache[transcript_path], location)
                        if window is None:
                            emit({**skeleton, "skipped_reason": SKIP_NO_WINDOW})
                            print(f"  skipped: {SKIP_NO_WINDOW} (location {location})", flush=True)
                            continue
                        context_intro = (
                            f"WINDOW (lines {window['location_start']}-{window['location_end']} "
                            "of the session transcript; each line is prefixed with its absolute "
                            "line number):"
                        )
                        context_body = _numbered_window_text(window)
                        location_note = f"- location: line {location}"
                        phrase_context = _centered_excerpt(
                            window, location, PHRASE_GEN_CONTEXT_CAP
                        )

                    # -------- 2. PHRASES (one cheap haiku call) --------
                    phrases, phrase_spend, phrase_failure = _generate_search_phrases(
                        moment, phrase_context, cfg_dir
                    )
                    target_cost += phrase_spend
                    invocation_spend += phrase_spend
                    if phrases is None:
                        emit(
                            {
                                **skeleton,
                                "skipped_reason": phrase_failure,
                                "cost_usd": target_cost if target_cost > 0 else None,
                            }
                        )
                        print(f"  skipped: {phrase_failure}", flush=True)
                        continue
                    outcome["search_phrases"] = phrases

                    # -------- 3. REAL POINT-IN-TIME RETRIEVAL (DERIVED mode) --------
                    top5, retrieval_error = _point_in_time_top5(
                        target_prep["commit_sha"], moment["timestamp"], phrases
                    )
                    if top5 is None:
                        emit(
                            {
                                **outcome,
                                "skipped_reason": (
                                    f"point-in-time query failed: {retrieval_error}"
                                ),
                                "cost_usd": target_cost if target_cost > 0 else None,
                            }
                        )
                        print(f"  skipped: point-in-time query failed: {retrieval_error}", flush=True)
                        continue
                    outcome["top5"] = [
                        {"path": item["path"], "kind": item["kind"], "score": item["score"]}
                        for item in top5
                    ]

                    # -------- 4. ONE sonnet judgment (both measurements) --------
                    prompt = _build_judgment_prompt(
                        context_intro,
                        context_body,
                        _retrieval_block(phrases, top5),
                        location_note,
                        moment["moment_type"],
                        moment["description"],
                        has_items=bool(top5),
                    )

                    judged_fields: Optional[Dict[str, Any]] = None
                    deterministic_skip: Optional[str] = None
                    for attempt in (1, 2):
                        response = _run_claude_p(prompt, "sonnet", cfg_dir=cfg_dir)
                        call_cost = response.get("total_cost_usd", 0) or 0
                        target_cost += call_cost
                        invocation_spend += call_cost

                        if response.get("prompt_too_long"):
                            deterministic_skip = SKIP_TOO_LONG  # never retried
                            break
                        if response.get("exhausted_retries"):
                            # HALT THE ENTIRE RUN: a sustained auth/API outage must
                            # never be absorbed as per-moment skips (2026-09-02 lesson).
                            raise ExhaustedRetriesError(
                                f"claude -p exhausted retries on {moment_id} -- halting the "
                                f"entire run (${invocation_spend:.4f} spent this invocation; "
                                "output so far is recorded; re-run to resume)"
                            )
                        ok, fields = _parse_judgment(response.get("result", ""), top5)
                        if ok:
                            judged_fields = fields
                            break
                        if attempt == 1:
                            print("  malformed/invalid response; retrying once ...", flush=True)

                    if deterministic_skip is not None:
                        outcome["skipped_reason"] = deterministic_skip
                    elif judged_fields is None:
                        outcome["skipped_reason"] = SKIP_UNPARSEABLE
                    else:
                        outcome.update(judged_fields)

                    outcome["cost_usd"] = target_cost if target_cost > 0 else None
                    emit(outcome)
                    if outcome["skipped_reason"] is None:
                        print(
                            f"  -> {outcome['relevance_grade']} / {outcome['fire_cue']}"
                            f"{' (' + outcome['cue_name'] + ')' if outcome['cue_name'] else ''}"
                            f" (${target_cost:.4f}; invocation spend so far "
                            f"${invocation_spend:.4f})",
                            flush=True,
                        )
                    else:
                        print(f"  skipped: {outcome['skipped_reason']}", flush=True)

                except ExhaustedRetriesError:
                    raise  # halt the run -- never absorbed as a per-moment skip
                except Exception as moment_error:  # noqa: BLE001 -- one bad moment must not kill the sweep
                    emit(
                        {
                            **skeleton,
                            "skipped_reason": (
                                f"error: {type(moment_error).__name__}: {moment_error}"
                            ),
                            "cost_usd": target_cost if target_cost > 0 else None,
                        }
                    )
                    print(
                        f"  skipped (error): {type(moment_error).__name__}: {moment_error}",
                        flush=True,
                    )
    finally:
        shutil.rmtree(cfg_dir, ignore_errors=True)

    if len(done_ids) >= len(targets):
        _print_final_summary()
        # -------- 6. MECHANICAL fire-unit estimate (no LLM) --------
        estimate = _compute_fire_unit_estimate()
        with open(FIRE_UNIT_PATH, "w") as f:
            json.dump(estimate, f, indent=2)
        _print_fire_unit_table(estimate)
        print(f"\nfire-unit estimate written to {FIRE_UNIT_PATH}", flush=True)
    else:
        print(
            f"\nIncomplete: {len(done_ids)}/{len(targets)} targets recorded "
            f"(${invocation_spend:.4f} spent this invocation). Re-run to resume.",
            flush=True,
        )


if __name__ == "__main__":
    main()
