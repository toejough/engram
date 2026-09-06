"""
Semantic auditor for memory-loop moments.

Detects failures, corrections, rework, successes, and dispatches in transcript windows,
judges each moment on the scorecard from design.md D-D, and outputs JSONL records.

This tool splits transcripts into windows, detects moments using an LLM (haiku for speed,
sonnet for re-checking low-yield windows), judges the moments on the full scorecard,
and joins the results with structural events from extract.py.
"""

import contextlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
import time
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, List, Optional, Callable, Iterator

# Default low-yield threshold: minimum candidates from haiku before re-running sonnet.
# This is a PLACEHOLDER pending task 3.3's calibration.
LOW_YIELD_THRESHOLD = 2

# Instrument version, stamped into every moment record as instrument_version.
# Comparability guard (vault note 282's deploy-byte-identical lesson): before/after
# comparisons must declare which instrument produced each side.
# - Version 1: the original run -- bare nulls permitted on conditional judged fields,
#   no per-question rationales, existence check was a thresholdless "any items
#   returned" gate.
# - Version 2: folds the gate-week patch-learnings (rejudge_followed_nulls.py,
#   rejudge_all_nulls.py, measure_relevance_cues.py) into the judgment instrument:
#   "not_applicable" replaces bare nulls on every conditional judged field (with a
#   one-sentence rationale per answered question), and the point-in-time existence
#   check captures its top-5 items and grades their relevance against the moment.
INSTRUMENT_VERSION = 2

# The stamp written into <field>_null_reason when the judge answers "not_applicable":
# the field itself stays null and the reason explains why -- the merged corpus's
# convention (merge_followed_nulls.py / merge_all_nulls.py), so instrument-v2 records
# join the corpus without a new enum value in the field itself.
JUDGED_NOT_APPLICABLE = "judged_not_applicable"

# Conditional judged fields that may honestly be "not_applicable", grouped by where
# they live on the record. Defined at the SCHEMA level -- all judged fields of the
# same kind, never scoped to one exemplar field (vault note 900a).
_NOT_APPLICABLE_FINDING_FIELDS = (
    "search_targeted_right_thing",
    "outdated_outranked_replacement",
    "followed",
)
_NOT_APPLICABLE_WRITING_FIELDS = (
    "note_well_targeted",
    "note_superseded_correctly",
    "note_not_duplicate",
    "strength_mismatch_flagged",
)

# Point-in-time existence check: how many top items to capture onto the record, and
# the only relevance grades the follow-up judgment may answer (same enum as
# measure_relevance_cues.py).
EXISTENCE_TOP_K = 5
RELEVANCE_GRADES = ("relevant_top1", "relevant_in_top5", "nothing_relevant")

# Keychain command to retrieve Claude Code credentials (same as dev/eval/traps/run.py)
KEYCHAIN = 'security find-generic-password -s "Claude Code-credentials" -w'

# Flat-vault migration commit (2026-06-12): commits before this have a different vault structure.
# The auditor can only check commits at or after this point via engram query.
# Commits before it require ESTIMATE mode (filtering by creation date).
FLAT_VAULT_MIGRATION_COMMIT = "4d8860e"


class ExhaustedRetriesError(RuntimeError):
    """
    Raised when _run_claude_p's retry-with-backoff budget is genuinely exhausted --
    every attempt in the backoff sequence failed (transient error, or unparseable
    response treated as transient) -- as opposed to a real "the model found nothing"
    result.

    This is a hard-stop signal, not a soft failure: detect_moments/judge_moment raise
    this instead of silently returning "0 candidates"/"default judgment" so a sustained
    auth/API outage can never masquerade as 40 windows of clean zero-moment results
    (the 2026-09-02 incident this class exists to prevent). Callers running a
    multi-window loop MUST catch this specifically and halt the entire run on first
    occurrence, rather than logging-and-continuing the way a single window's
    extract/audit failure otherwise would.
    """


class PromptTooLongError(RuntimeError):
    """
    Raised when _run_claude_p's subprocess call returns a deterministic
    terminal_reason="prompt_too_long" response -- the window's content, once wrapped
    with line-number prefixes and the detection/judgment instruction text, exceeds
    claude -p's input limit.

    This is an INSTANT ($0 cost, 0ms duration), deterministic rejection -- never a
    transient failure -- so _run_claude_p does not retry it, and detect_moments/
    judge_moment raise this distinct exception rather than either (a) silently
    returning an empty candidate list/default judgment (indistinguishable from a
    genuine zero-yield result), or (b) raising ExhaustedRetriesError, which
    specifically means "we retried and gave up" -- misleading for a failure that
    never should have been retried in the first place.

    A caller catching this should treat it as "this window's content is too large for
    a single claude -p call" (i.e. split_into_windows()'s max_kb is still too generous
    for this content, or the window needs further splitting), not as a transient/auth
    problem to retry or wait out (the 2026-09-02 incident this class exists to prevent:
    a 431.8 KB single-window prompt burned the full ~3-minute retry backoff sequence,
    twice, before being correctly diagnosed as this deterministic failure mode).
    """


def _find_commit_at_or_before(vault_repo_path: str, timestamp: str) -> Optional[str]:
    """
    Find the last git commit in vault_repo_path at or before the given ISO 8601 timestamp.

    Returns the commit SHA, or None if no commit exists before that timestamp.

    Args:
        vault_repo_path: path to the vault git repository (~/.local/share/engram/vault)
        timestamp: ISO 8601 string (e.g., "2026-08-30T14:30:00+00:00")

    Returns:
        commit SHA (short form) or None
    """
    try:
        # git log --before filters to commits strictly before the timestamp.
        # We need --before plus 1 second, or use -1 to get just the last one matching.
        result = subprocess.run(
            [
                "git",
                "-C",
                vault_repo_path,
                "log",
                f"--before={timestamp}",
                "-1",
                "--format=%h",  # short SHA
            ],
            capture_output=True,
            text=True,
            timeout=10,
        )

        sha = result.stdout.strip()
        return sha if sha else None
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return None


def _is_commit_ancestor_of(vault_repo_path: str, ancestor_sha: str, descendant_sha: str) -> bool:
    """
    Check if ancestor_sha is an ancestor of descendant_sha in the vault repo.
    Uses git merge-base --is-ancestor.

    Returns True if ancestor_sha is an ancestor of (or equal to) descendant_sha.
    """
    try:
        # `git merge-base --is-ancestor` is read-only/non-mutating (confirmed by prior
        # review) -- it only inspects commit ancestry, no working-tree or ref changes.
        result = subprocess.run(
            [
                "git",
                "-C",
                vault_repo_path,
                "merge-base",
                "--is-ancestor",
                ancestor_sha,
                descendant_sha,
            ],
            capture_output=True,
            timeout=10,
        )
        return result.returncode == 0
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return False


@contextlib.contextmanager
def _checkout_vault_at(vault_repo_path: str, sha: str, scratch_root: str) -> Iterator[str]:
    """
    Create a detached worktree checkout of the vault at the given commit.

    A context manager that yields the worktree path, and cleans up afterward.
    The real vault's working tree is never modified (worktree add/remove are read-only
    operations on the .git registry, not the working tree).

    Args:
        vault_repo_path: path to the vault git repository
        sha: commit SHA to check out
        scratch_root: directory under which to create the worktree

    Yields:
        path to the detached worktree checkout

    Raises:
        subprocess.CalledProcessError if git commands fail
    """
    worktree_dir = os.path.join(scratch_root, f"vault-worktree-{sha[:8]}")

    # Defensive sweep: reconcile any dangling worktree registration leaked by a prior
    # run/process before creating a new one. Cheap insurance -- `git worktree prune`
    # is read-only on the working tree and only reclaims registrations whose backing
    # directory is already gone from disk (a no-op otherwise).
    try:
        subprocess.run(
            ["git", "-C", vault_repo_path, "worktree", "prune"],
            capture_output=True,
            timeout=30,
        )
    except Exception:
        pass

    def _force_delete_and_prune() -> None:
        # `git worktree prune` ONLY reclaims a registration once its backing directory
        # is already gone from disk -- it is a no-op while the directory still
        # physically exists, which it does immediately after a failed `remove` (nothing
        # else deleted it). So on a remove failure we must delete the directory
        # ourselves FIRST, then prune to reconcile the now-dangling registration. Order
        # matters: prune-before-delete leaves both the directory and the registration
        # behind (this was the bug -- confirmed by direct reproduction against the
        # real vault, leaking a live worktree registration).
        shutil.rmtree(worktree_dir, ignore_errors=True)
        try:
            subprocess.run(
                ["git", "-C", vault_repo_path, "worktree", "prune"],
                capture_output=True,
                timeout=30,
            )
        except Exception:
            pass

    try:
        # Create detached worktree
        subprocess.run(
            ["git", "-C", vault_repo_path, "worktree", "add", "--detach", worktree_dir, sha],
            capture_output=True,
            check=True,
            timeout=30,
        )

        yield worktree_dir

    finally:
        # Always clean up the worktree, even on error
        try:
            result = subprocess.run(
                ["git", "-C", vault_repo_path, "worktree", "remove", worktree_dir, "--force"],
                capture_output=True,
                timeout=30,
            )
            # If remove failed (non-zero exit), force-delete the directory ourselves
            # and THEN prune -- prune alone cannot reconcile a still-present directory.
            if result.returncode != 0:
                _force_delete_and_prune()
        except Exception:
            # If remove command itself raised an exception, same fallback applies.
            _force_delete_and_prune()


def _filter_notes_by_creation_date(vault_dir: str, timestamp: str) -> List[str]:
    """
    Filter vault notes by their creation date, returning note paths created at or before timestamp.

    This is used for ESTIMATE mode when git history doesn't reach back to the moment's timestamp.

    Args:
        vault_dir: path to the vault directory
        timestamp: ISO 8601 string (e.g., "2026-08-30T14:30:00+00:00")

    Returns:
        list of note paths (absolute paths) that were created at or before the timestamp
    """
    from datetime import datetime as dt_class, timezone as tz_class

    def parse_iso_timestamp(ts_str):
        """
        Parse ISO 8601 timestamp strings, preserving timezone offset.

        Normalizes 'Z' to '+00:00' so Python's stdlib fromisoformat can parse both
        'Z'-suffixed and explicit-offset ('-04:00') inputs. Does NOT strip the offset
        before parsing -- doing so silently compares raw clock digits across different
        timezones (e.g. a note's '-04:00' created date vs a UTC moment timestamp),
        which is wrong by up to the offset's magnitude. Naive (offset-less) inputs
        (vault notes' `created:` field is often just a date, e.g. "2026-06-26") are
        assumed UTC so every returned datetime stays tz-aware and comparable.
        """
        try:
            parsed = dt_class.fromisoformat(ts_str.replace("Z", "+00:00"))
        except Exception:
            return None
        if parsed.tzinfo is None:
            parsed = parsed.replace(tzinfo=tz_class.utc)
        return parsed

    moment_dt = parse_iso_timestamp(timestamp)
    if moment_dt is None:
        # If parsing fails, use current time (tz-aware) as a safe fallback -- a naive
        # fallback would raise TypeError when compared against tz-aware datetimes above.
        moment_dt = dt_class.now(tz_class.utc)

    filtered_notes = []

    try:
        for filename in os.listdir(vault_dir):
            if not filename.endswith(".md"):
                continue

            note_path = os.path.join(vault_dir, filename)
            try:
                with open(note_path, "r") as f:
                    # Read frontmatter to look for created field
                    content = f.read()

                    # Try to extract created/timestamp from frontmatter (YAML-style)
                    created_str = None
                    if content.startswith("---"):
                        # Split on the closing --- to get frontmatter
                        parts = content.split("---", 2)
                        if len(parts) >= 2:
                            frontmatter = parts[1]
                            # Look for created: or timestamp: fields
                            for line in frontmatter.split("\n"):
                                if line.startswith("created:"):
                                    created_str = line.split(":", 1)[1].strip().strip('"\'')
                                    break
                                elif line.startswith("timestamp:"):
                                    created_str = line.split(":", 1)[1].strip().strip('"\'')
                                    break

                    # Parse the created date
                    if created_str:
                        created_dt = parse_iso_timestamp(created_str)
                        if created_dt and created_dt <= moment_dt:
                            filtered_notes.append(note_path)
                    else:
                        # If no creation date found, include the note as a fallback
                        # (conservative assumption: might have existed)
                        filtered_notes.append(note_path)

            except (IOError, OSError):
                # Skip files we can't read
                pass

    except (IOError, OSError):
        # If we can't list the vault, return empty
        pass

    return filtered_notes


# Real vault notes carry a `type:` frontmatter field with these values (confirmed by
# inspecting the live vault: fact/feedback/runbook are used as-is; qa-question and
# qa-answer both collapse to the D-D scorecard's "qa" kind).
_NOTE_TYPE_TO_MEMORY_KIND = {
    "fact": "fact",
    "feedback": "feedback",
    "runbook": "runbook",
    "qa-question": "qa",
    "qa-answer": "qa",
}


def _parse_note_type(note_content: str) -> Optional[str]:
    """
    Parse the `type:` frontmatter field from a vault note's raw content and map it to
    the D-D scorecard's memory_kind vocabulary (chunk/fact/feedback/runbook/qa).

    Used by the ESTIMATE existence-check path, which already opens the note file to
    read its `created:` field -- this reads the same file for its `type:` field instead
    of hardcoding a constant.

    Returns the mapped kind, the raw type string if unrecognized, or None if no `type:`
    field is present.
    """
    if not note_content.startswith("---"):
        return None

    parts = note_content.split("---", 2)
    if len(parts) < 2:
        return None

    frontmatter = parts[1]
    for line in frontmatter.split("\n"):
        stripped = line.strip()
        if stripped.startswith("type:"):
            raw_type = stripped.split(":", 1)[1].strip().strip('"\'')
            return _NOTE_TYPE_TO_MEMORY_KIND.get(raw_type, raw_type or None)

    return None


def _filter_chunks_by_ingestion_date(chunks_dir: str, timestamp: str, scratch_dir: str) -> str:
    """
    Filter chunk JSONL files, keeping only records with ingested_at <= timestamp.

    This creates a scratch copy of the chunk index, containing only chunks that were
    ingested at or before the given timestamp. Legacy records (with no ingested_at field)
    are excluded, since their existence at a particular moment in time is unknown.

    Args:
        chunks_dir: path to the real chunks directory (~/.local/share/engram/chunks)
        timestamp: ISO 8601 string (e.g., "2026-08-30T14:30:00+00:00")
        scratch_dir: where to write the filtered chunks directory

    Returns:
        path to the filtered chunks directory
    """
    from datetime import datetime as dt_class, timezone as tz_class

    filtered_chunks_dir = os.path.join(scratch_dir, "chunks-filtered")
    os.makedirs(filtered_chunks_dir, exist_ok=True)

    # Parse the moment's timestamp using ISO format parsing
    def parse_iso_timestamp(ts_str):
        """
        Parse ISO 8601 timestamp strings (both with and without timezone), preserving
        the timezone offset.

        Handles formats like "2026-08-30T14:30:00Z" or "2026-07-26T21:14:45.075526-04:00".
        Does NOT strip the offset before parsing -- real chunk records split roughly
        50/50 between 'Z'-suffixed (UTC) and explicit-offset timestamps, and comparing
        raw clock digits across differing offsets (as the previous version did) is wrong
        by up to the offset's magnitude. 'Z' is normalized to '+00:00' so Python's stdlib
        fromisoformat can parse it directly. Naive (offset-less) inputs are assumed UTC
        so every returned datetime stays tz-aware and comparable.
        """
        try:
            parsed = dt_class.fromisoformat(ts_str.replace("Z", "+00:00"))
        except Exception:
            return None
        if parsed.tzinfo is None:
            parsed = parsed.replace(tzinfo=tz_class.utc)
        return parsed

    moment_dt = parse_iso_timestamp(timestamp)
    if moment_dt is None:
        # If parsing fails, use current time (tz-aware) as a safe fallback -- a naive
        # fallback would raise TypeError when compared against tz-aware datetimes above.
        moment_dt = dt_class.now(tz_class.utc)

    # Walk the real chunks dir and filter records
    for filename in os.listdir(chunks_dir):
        if not filename.endswith(".jsonl"):
            continue

        input_path = os.path.join(chunks_dir, filename)
        output_path = os.path.join(filtered_chunks_dir, filename)

        try:
            with open(input_path, "r") as infile:
                lines = infile.readlines()
        except (IOError, OSError):
            # Skip files we can't read
            continue

        filtered_lines = []
        for line in lines:
            line = line.strip()
            if not line:
                continue

            try:
                record = json.loads(line)
                ingested_at_str = record.get("ingested_at")

                # Skip legacy records (no ingested_at field)
                if not ingested_at_str:
                    continue

                # Parse ingested_at and compare
                ingested_dt = parse_iso_timestamp(ingested_at_str)
                if ingested_dt is not None and ingested_dt <= moment_dt:
                    filtered_lines.append(line)
            except (json.JSONDecodeError, KeyError):
                # Skip malformed records
                pass

        # Write filtered records
        if filtered_lines:
            with open(output_path, "w") as outfile:
                for line in filtered_lines:
                    outfile.write(line + "\n")

    return filtered_chunks_dir


def _run_engram_query_at_moment(
    vault_path: str,
    chunks_path: str,
    search_phrases: List[str],
    trial_dir_base: Optional[str] = None,
) -> Dict[str, Any]:
    """
    Run engram query with the given phrases, isolated to the given vault and chunks.

    Args:
        vault_path: path to a (possibly temporary) vault directory
        chunks_path: path to a (possibly filtered) chunks directory
        search_phrases: list of 2-4 search phrases to use
        trial_dir_base: optional base for isolation trial dir (defaults to /tmp)

    Returns:
        dict with:
        - "items": list of query results (parsed from YAML), or None if the check errored
        - "raw_output": the raw stdout from engram query (if successful)
        - "error": error message (if the check failed), or None if successful

        The caller can distinguish between:
        - Successful check finding results: "items" is a non-empty list, "error" is None
        - Successful check finding nothing: "items" is an empty list, "error" is None
        - Failed check: "items" is None, "error" contains the error message
    """
    # Import isolation module and parse_engram_query_result
    sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
    import isolation
    from extract import parse_engram_query_result

    # Build isolated environment
    env = isolation.engram_env(vault=vault_path, chunks=chunks_path)

    # Confirm isolation before running
    try:
        isolation.assert_engram_isolated(env)
    except isolation.IsolationError as e:
        error_msg = f"isolation check failed: {e}"
        sys.stderr.write(f"[audit_moments] {error_msg}\n")
        return {"items": None, "error": error_msg}

    # Build engram query command
    args = ["engram", "query", "--lazy-chunks"]
    for phrase in search_phrases:
        args.extend(["--phrase", phrase])

    try:
        result = subprocess.run(
            args,
            env=env,
            capture_output=True,
            text=True,
            timeout=60,
        )

        if result.returncode != 0:
            error_msg = f"engram query failed with rc {result.returncode}: {result.stderr}"
            sys.stderr.write(f"[audit_moments] {error_msg}\n")
            return {"items": None, "error": error_msg}

        # Parse YAML output using the real parser from extract.py
        items = parse_engram_query_result(result.stdout)

        return {"items": items, "raw_output": result.stdout, "error": None}

    except subprocess.TimeoutExpired:
        error_msg = "engram query timed out"
        sys.stderr.write(f"[audit_moments] {error_msg}\n")
        return {"items": None, "error": error_msg}
    except FileNotFoundError:
        error_msg = "engram binary not found on PATH"
        sys.stderr.write(f"[audit_moments] {error_msg}\n")
        return {"items": None, "error": error_msg}


def _resolve_fork_parent_context(fork_transcript_path: str) -> Dict[str, Any]:
    """
    Resolve the memory context a forked subagent inherited from its parent conversation.

    Per design.md's D-E rule, a fork moment is structurally marked surfaced=True and
    search_ran="injected" in judge_moment because the fork inherits its parent's full
    conversation -- but the LLM judgment call is never shown the actual CONTENT of what
    was recalled there, only this fork's own window text, which contains none of it
    (a fork transcript starts fresh with a "fork-context-ref" header line, not the
    inherited conversation itself). This resolves that gap: read the fork transcript's
    own fork-context-ref line to find its parent transcript and the exact point in it
    the fork branched from, then look up whichever recall_call happened last before
    that point and read back the vault notes it matched, so judge_moment can inject
    that content into its prompt.

    Args:
        fork_transcript_path: path to a fork subagent's own transcript JSONL file.

    Returns:
        dict with:
        - "available": bool -- whether resolution got far enough to look for notes
          (False only means resolution itself failed -- not a fork transcript, its
          parent transcript is gone, or the branch point couldn't be located; True
          means resolution succeeded even if recalled_notes ends up empty)
        - "reason": str or None -- why resolution stopped short, or why recalled_notes
          is empty despite available=True; None only on full, unremarkable success
        - "recalled_notes": list of {"path", "kind", "content"} dicts, best-effort --
          may be empty even when available=True (no recall before the fork point, or
          every matched note's file is gone from the live vault today)
    """
    try:
        with open(fork_transcript_path, "r") as f:
            first_line = f.readline()
    except (IOError, OSError):
        return {"available": False, "reason": "not a fork transcript", "recalled_notes": []}

    try:
        first_event = json.loads(first_line)
    except json.JSONDecodeError:
        return {"available": False, "reason": "not a fork transcript", "recalled_notes": []}

    if first_event.get("type") != "fork-context-ref":
        return {"available": False, "reason": "not a fork transcript", "recalled_notes": []}

    parent_session_id = first_event.get("parentSessionId")
    parent_last_uuid = first_event.get("parentLastUuid")

    # Fork path convention: "<project-dir>/<session-id>/subagents/agent-X.jsonl". The
    # parent transcript is a sibling of the session-id directory, named by
    # parentSessionId (read from the JSON field above, not assumed from the directory
    # name -- retention/reorg could in principle decouple the two).
    session_dir = os.path.dirname(os.path.dirname(fork_transcript_path))
    parent_transcript_path = os.path.join(
        os.path.dirname(session_dir), f"{parent_session_id}.jsonl"
    )

    if not os.path.exists(parent_transcript_path):
        # Retention can delete the parent transcript independently of the fork's own
        # transcript -- a real, observed case, not hypothetical.
        return {"available": False, "reason": "parent transcript not found", "recalled_notes": []}

    try:
        with open(parent_transcript_path, "r") as f:
            parent_lines = f.readlines()
    except (IOError, OSError):
        return {"available": False, "reason": "parent transcript not found", "recalled_notes": []}

    bound_line_number = None
    for idx, raw_line in enumerate(parent_lines, start=1):
        try:
            event = json.loads(raw_line)
        except json.JSONDecodeError:
            continue
        if event.get("uuid") == parent_last_uuid:
            bound_line_number = idx
            break

    if bound_line_number is None:
        return {
            "available": False,
            "reason": "parentLastUuid not found in parent transcript",
            "recalled_notes": [],
        }

    from extract import extract_transcript

    parent_extracted = extract_transcript(parent_transcript_path)
    recall_calls = parent_extracted.get("recall_calls", [])
    qualifying = [rc for rc in recall_calls if rc.get("location", 0) <= bound_line_number]

    if not qualifying:
        return {"available": True, "reason": "no recall before fork point", "recalled_notes": []}

    # The LAST qualifying recall_call is the one closest to the fork point.
    last_recall = max(qualifying, key=lambda rc: rc.get("location", 0))

    vault_dir = os.path.expanduser("~/.local/share/engram/vault")
    recalled_notes = []
    for item in last_recall.get("matched_items", []):
        note_relative_path = item.get("path")
        if not note_relative_path:
            continue
        note_path = os.path.join(vault_dir, note_relative_path)
        try:
            with open(note_path, "r") as f:
                content = f.read()
        except (IOError, OSError):
            # Best-effort: the note may have been renamed/superseded/deleted since
            # this recall happened. Skip it silently, keep any other matched items.
            continue
        recalled_notes.append({"path": note_relative_path, "kind": item.get("kind"), "content": content})

    return {"available": True, "reason": None, "recalled_notes": recalled_notes}


def split_into_windows(transcript_path: str, max_kb: int = 150) -> List[Dict[str, Any]]:
    """
    Split a transcript file into windows at line boundaries.

    Files at or under max_kb (default 150 KB) are returned as a single window. Files
    over max_kb are split into ≤max_kb windows at line boundaries (never mid-line).

    The single-window threshold is deliberately pinned to max_kb itself, not a larger,
    separate cutoff. A prior version used a 500 KB cutoff, leaving a 150-500 KB gap
    where a file was "not quite big enough to auto-split" but still large enough to
    blow the prompt-size limit once wrapped with line-number prefixes and the
    detection/judgment instruction wrapper. That gap was confirmed real: a 431.8 KB
    transcript (`agent-aa9bb69295f0f3e48.jsonl`) qualified for the old single-window
    path and produced one ~441,630-character prompt, which `claude -p` rejected
    instantly (`terminal_reason="prompt_too_long"`, $0 cost, 0ms -- a deterministic
    rejection, not a transient failure). Pinning the threshold to max_kb closes the
    gap: any file not already comfortably within the per-window cap now goes through
    the same multi-window splitting path as any other oversized file.

    Note: If a single JSONL line exceeds max_kb, it is allowed as a window by itself
    (documented as an exception in the window's metadata), since lines cannot be split.

    Each window dict contains:
    - lines: list of complete JSON-line strings
    - jsonl_lines: the same (for compatibility)
    - location_start: 1-indexed line number of first line
    - location_end: 1-indexed line number of last line (inclusive)
    - transcript_path: input path
    - window_index: 0-based ordinal
    - total_windows: total number of windows
    - oversized_line: true if this window's sole content exceeds max_kb
    """
    with open(transcript_path, "r") as f:
        all_lines = f.readlines()

    # Calculate file size in KB
    file_size_kb = sum(len(line.encode("utf-8")) for line in all_lines) / 1024

    if file_size_kb <= max_kb:
        # Single window: the whole file already fits within the per-window cap.
        return [
            {
                "lines": all_lines,
                "jsonl_lines": all_lines,  # same thing
                "location_start": 1,
                "location_end": len(all_lines),
                "transcript_path": transcript_path,
                "window_index": 0,
                "total_windows": 1,
                "oversized_line": False,
            }
        ]

    # Multi-window: split at line boundaries, max 150 KB per window
    windows = []
    current_window = []
    current_size_kb = 0
    window_start_line = 1

    for line_idx, line in enumerate(all_lines, start=1):
        line_size_kb = len(line.encode("utf-8")) / 1024

        # If adding this line would exceed max and we have content, flush the current window
        if current_window and current_size_kb + line_size_kb > max_kb:
            windows.append(
                {
                    "lines": current_window,
                    "jsonl_lines": current_window,
                    "location_start": window_start_line,
                    "location_end": line_idx - 1,
                    "transcript_path": transcript_path,
                    "window_index": len(windows),
                    "total_windows": -1,  # Placeholder; filled in after
                    "oversized_line": current_size_kb > max_kb,
                }
            )
            current_window = []
            current_size_kb = 0
            window_start_line = line_idx

        current_window.append(line)
        current_size_kb += line_size_kb

    # Flush final window
    if current_window:
        windows.append(
            {
                "lines": current_window,
                "jsonl_lines": current_window,
                "location_start": window_start_line,
                "location_end": len(all_lines),
                "transcript_path": transcript_path,
                "window_index": len(windows),
                "total_windows": -1,  # Placeholder; filled in after
                "oversized_line": current_size_kb > max_kb,
            }
        )

    # Fill in total_windows count
    total = len(windows)
    for w in windows:
        w["total_windows"] = total

    return windows


def _moment_id(transcript_path: str, location: int) -> str:
    """Generate a stable, reproducible moment ID."""
    basename = os.path.basename(transcript_path)
    return f"{basename}#{location}"


def _build_cfg_dir(cfg_dir: str) -> None:
    """
    Build an isolated config directory with credentials from macOS keychain.

    Follows the same pattern as dev/eval/traps/run.py's build_cold_cfg.
    """
    os.makedirs(os.path.join(cfg_dir, "projects"), exist_ok=True)

    # Minimal .claude.json for isolated config
    cfg_json = os.path.join(cfg_dir, ".claude.json")
    with open(cfg_json, "w") as f:
        json.dump({"projects": {}}, f)

    # Seed credentials from macOS keychain (same pattern as dev/eval/traps/run.py)
    subprocess.run(
        ["bash", "-c", f'{KEYCHAIN} > {cfg_dir}/.credentials.json && chmod 600 {cfg_dir}/.credentials.json'],
        capture_output=True
    )


def _run_claude_p(
    prompt: str,
    model: str,
    stub: Optional[Callable] = None,
    cfg_dir: Optional[str] = None,
    trial_dir_base: Optional[str] = None,
) -> Dict[str, Any]:
    """
    Run claude -p with the given prompt and model, or use a stub.

    Args:
        prompt: the text to send to the model
        model: "haiku" or "sonnet" (CLI short alias, not dated model ID)
        stub: optional callable(prompt, model) -> dict with "result" and "total_cost_usd" keys.
              If provided, uses stub instead of real subprocess call.
        cfg_dir: optional persistent CLAUDE_CONFIG_DIR. If not provided, a temp one is created.
        trial_dir_base: optional base directory for trial dirs. Defaults to temp.

    Returns:
        dict with "result" (the model's text), "total_cost_usd", "exhausted_retries"
        (bool -- True only when every attempt in the backoff sequence failed; this is
        the distinguishable hard-failure signal callers must check for and never treat
        as a genuine "found nothing" result), and "prompt_too_long" (bool -- True only
        when claude -p returned terminal_reason="prompt_too_long", a deterministic,
        non-retryable rejection that short-circuits the backoff loop on the FIRST
        attempt rather than retrying; False in every other case, including
        exhausted_retries).
    """
    if stub is not None:
        return stub(prompt, model)

    # Build or reuse isolated config directory
    own_cfg_dir = cfg_dir is None
    if own_cfg_dir:
        cfg_dir = tempfile.mkdtemp(prefix="audit-moments-cfg-")
        _build_cfg_dir(cfg_dir)

    try:
        # Set up isolated env via isolation module
        sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
        sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
        import isolation

        trial_dir = tempfile.mkdtemp(prefix="audit-moments-trial-", dir=trial_dir_base)
        env = isolation.isolated_env(cfg_dir, trial_dir, cwd=trial_dir)

        # Run claude -p with retry on transient failures
        # Follow the pattern from dev/eval/traps/run.py
        args = [
            "claude",
            "-p",
            prompt,
            "--output-format",
            "json",
            "--model",
            model,
            "--permission-mode",
            "bypassPermissions",
        ]

        for backoff in (0, 15, 45, 120):
            if backoff:
                time.sleep(backoff)

            result = subprocess.run(
                args, cwd=trial_dir, env=env, capture_output=True, text=True, timeout=300
            )

            # Parse response
            try:
                out = json.loads(result.stdout)
            except json.JSONDecodeError:
                out = {}

            cost = out.get("total_cost_usd", 0) or 0
            is_err = out.get("is_error") or (not out)

            # Deterministic, non-retryable failure: claude -p's own prompt-size
            # rejection. This is an INSTANT ($0 cost, 0ms duration) response, not a
            # transient network/auth condition -- retrying can NEVER succeed, since the
            # prompt's size doesn't change between attempts. Checked BEFORE the
            # transient-retry check below, and returns immediately (no `continue`),
            # so this never burns the ~3-minute backoff sequence (the 2026-09-02
            # incident: a 431.8 KB single-window prompt did exactly that, twice).
            if out.get("terminal_reason") == "prompt_too_long":
                sys.stderr.write(
                    "[audit_moments] prompt_too_long: window content exceeds claude -p's "
                    "input limit -- not retrying (deterministic, would never succeed)\n"
                )
                return {
                    "result": "",
                    "total_cost_usd": cost,
                    "exhausted_retries": False,
                    "prompt_too_long": True,
                }

            # Retry on transient (cheap) errors only
            if is_err and cost < 0.02:
                continue

            # Either succeeded or hit a non-transient error; break and return what we have
            result_text = out.get("result", "")
            if is_err and cost >= 0.02:
                # Non-transient error: log it (don't raise, for forward compatibility with detection)
                sys.stderr.write(
                    f"[audit_moments] non-transient error: is_error=true, cost={cost}\n"
                )

            return {
                "result": result_text,
                "total_cost_usd": cost,
                "exhausted_retries": False,
                "prompt_too_long": False,
            }

        # Exhausted retries: every attempt in the backoff sequence (0/15/45/120s) failed.
        # This is NOT "the model found nothing" -- it's "we never got a real response" --
        # so the signal must be explicit and distinguishable, not folded into an empty
        # result string. Callers (detect_moments/judge_moment) must check this and raise
        # ExhaustedRetriesError rather than silently treating it as zero candidates/a
        # default judgment (2026-09-02 incident: a stale-credential outage silently
        # logged as "0 moments" for 40 straight windows over ~4 hours).
        sys.stderr.write(f"[audit_moments] exhausted retries on transient errors\n")
        return {"result": "", "total_cost_usd": 0, "exhausted_retries": True, "prompt_too_long": False}

    finally:
        # Clean up trial dir (reuse cfg_dir if it was passed in)
        if trial_dir:
            shutil.rmtree(trial_dir, ignore_errors=True)
        if own_cfg_dir:
            shutil.rmtree(cfg_dir, ignore_errors=True)


def detect_moments(
    window: Dict[str, Any],
    model: str = "haiku",
    stub: Optional[Callable] = None,
    cfg_dir: Optional[str] = None,
) -> List[Dict[str, Any]]:
    """
    Detect moment candidates (failures, corrections, etc.) in a window.

    Args:
        window: window dict from split_into_windows()
        model: "haiku" or "sonnet"
        stub: optional callable for testing
        cfg_dir: optional persistent CLAUDE_CONFIG_DIR

    Returns:
        list of candidate dicts with keys: location, moment_type, description
    """
    # Build window text with explicit line numbers to prevent model self-counting errors.
    # Use absolute line numbers from the transcript (window["location_start"] + offset).
    window_lines_numbered = []
    for offset, line in enumerate(window["lines"]):
        absolute_line_number = window["location_start"] + offset
        # Prefix each line with its absolute line number (1-indexed)
        # Format: "NNN: <line content>" where NNN is the absolute line number
        numbered_line = f"{absolute_line_number}: {line}"
        window_lines_numbered.append(numbered_line)

    window_text = "".join(window_lines_numbered)

    prompt = f"""You are analyzing a Claude Code agent session transcript for moments:
- Failures: the agent encounters an error, gets stuck, or produces wrong results
- Corrections: user or reviewer corrects the agent's approach
- Rework: agent re-does work based on feedback or its own realization
- Success: a memory the agent had was clearly used and helped; a procedure worked well
- Dispatch: agent sends work to a subagent

CRITICAL: Most corrections are SUBTLE (no blunt keyword "fix" or "change"). Detect by READING IN CONTEXT.
READ THE WINDOW FOR SEMANTIC MEANING, NOT KEYWORDS. Over-flag generously — missing real moments is worse than false positives.

IMPORTANT: Each line below is prefixed with its real line number (e.g., "47: {{"type":"user",...}}").
Report the EXACT line number shown as the location for each moment. Do NOT count lines yourself.

Respond with STRICT JSON ONLY (no prose wrapper), matching this schema exactly:
{{
  "candidates": [
    {{
      "location": <line number shown at the start of the line, 1-indexed>,
      "moment_type": "failure" | "correction" | "rework" | "success" | "dispatch",
      "description": "<one-sentence description of the moment>"
    }}
  ]
}}

WINDOW (lines {window['location_start']}-{window['location_end']}):
{window_text}"""

    response = _run_claude_p(prompt, model, stub=stub, cfg_dir=cfg_dir)

    # Deterministic, non-retryable failure: this window's content (once wrapped) is
    # too large for a single claude -p call. Distinct from exhausted_retries (which
    # means "possibly-transient failure, gave up after retrying") -- checked first so
    # it's never mistaken for that, or for a genuine zero-candidate result.
    if response.get("prompt_too_long"):
        raise PromptTooLongError(
            f"detect_moments: window "
            f"{window.get('location_start')}-{window.get('location_end')} "
            f"(transcript={window.get('transcript_path')}, model={model}) exceeds "
            f"claude -p's prompt size limit -- split into smaller windows, do not retry"
        )

    # Hard-stop signal: retries were genuinely exhausted (not a real "found nothing"
    # result). Must never fall through to the JSON-parse-failure path below, which
    # returns an indistinguishable empty candidate list.
    if response.get("exhausted_retries"):
        raise ExhaustedRetriesError(
            f"detect_moments: claude -p exhausted retries for window "
            f"{window.get('location_start')}-{window.get('location_end')} "
            f"(transcript={window.get('transcript_path')}, model={model})"
        )

    result_text = response.get("result", "")

    # Parse JSON from response (model may wrap it in prose or code fence)
    json_text = result_text
    if "```json" in json_text:
        json_text = json_text.split("```json")[1].split("```")[0]
    elif "```" in json_text:
        json_text = json_text.split("```")[1].split("```")[0]

    try:
        parsed = json.loads(json_text)
        candidates = parsed.get("candidates", [])
    except json.JSONDecodeError:
        candidates = []

    return candidates


def _extract_timestamp_from_window(window: Dict[str, Any], location: int) -> Optional[str]:
    """
    Extract the timestamp from a specific line in the window.

    Args:
        window: window dict from split_into_windows()
        location: 1-indexed line number within the window

    Returns:
        timestamp string (ISO 8601 format) or None if not found
    """
    try:
        # Adjust location to index within the window
        line_idx = location - window["location_start"]
        if 0 <= line_idx < len(window["lines"]):
            line_str = window["lines"][line_idx]
            line_obj = json.loads(line_str)
            return line_obj.get("timestamp")
    except (json.JSONDecodeError, IndexError, KeyError):
        pass
    return None


def _stamp_not_applicable(record_side: Dict[str, Any], field: str) -> None:
    """
    Apply the merged corpus's not_applicable convention to one judged field in place:
    a "not_applicable" verdict becomes a null field value plus
    <field>_null_reason = "judged_not_applicable". Any other value (including a
    degenerate bare null from a non-compliant response) is left untouched, and no
    null_reason key is added -- the stamp appears only where a verdict earned it.
    """
    if record_side.get(field) == "not_applicable":
        record_side[field] = None
        record_side[f"{field}_null_reason"] = JUDGED_NOT_APPLICABLE


def _parse_judgment_rationales(judgment: Dict[str, Any]) -> Dict[str, str]:
    """
    Extract the judgment response's "rationales" object: one-sentence rationale per
    answered question, keyed by field name. Malformed shapes (missing, non-dict,
    non-string entries) degrade to an empty dict -- never invented, never a crash.
    """
    raw_rationales = judgment.get("rationales")
    if not isinstance(raw_rationales, dict):
        return {}
    return {
        key: value.strip()
        for key, value in raw_rationales.items()
        if isinstance(key, str) and isinstance(value, str) and value.strip()
    }


def _build_relevance_grade_prompt(
    moment_type: str,
    description: str,
    location: int,
    window: Dict[str, Any],
    window_text: str,
    existence_top5: List[Dict[str, Any]],
    search_phrases: List[str],
) -> str:
    """
    The follow-up judgment prompt grading the point-in-time existence check's top-5
    against the moment. A separate, second call by necessity: the top-5 only exist
    after the existence check runs, and that check needs the main judgment call's
    search_phrases -- so the relevance question cannot ride on the first prompt.
    Uses measure_relevance_cues.py's strict framing (a technically-overlapping
    generic lesson is NOT relevant).
    """
    items_block = "\n".join(
        f"- rank {rank}: {item['path']} (kind={item['kind']}, score={item['score']})"
        for rank, item in enumerate(existence_top5, start=1)
    )
    return f"""You are auditing one moment from a real agent-session transcript for a memory-loop audit. \
A point-in-time memory search was re-run against the vault exactly as it existed at this moment's \
timestamp. Answer ONE question: grade the relevance of what it returned against this moment.

Window context (lines {window['location_start']}-{window['location_end']}; each line is prefixed \
with its real line number):

{window_text}

MOMENT UNDER JUDGMENT:
- location: line {location}
- moment_type: {moment_type}
- description: {description}

Search phrases used: {"; ".join(search_phrases)}
Top-{len(existence_top5)} items returned (ranked):
{items_block}

QUESTION -- "relevance_grade": would the retrieved memory have genuinely helped at THIS moment?
  - "relevant_top1": the rank-1 item genuinely bears on this moment -- it would have informed or \
changed what the agent did. (If rank 1 bears, answer this even if lower items also bear.)
  - "relevant_in_top5": rank 1 does not bear, but a lower-ranked item does.
  - "nothing_relevant": none of the retrieved items would have helped here.
Judge honestly against the moment, not charitably -- a generic lesson that technically overlaps \
but would not have changed anything is NOT relevant.

Respond with STRICT JSON only -- no prose before or after:
{{"relevance_grade": "relevant_top1" | "relevant_in_top5" | "nothing_relevant", "rationale": "<one sentence>"}}"""


def _parse_relevance_grade(result_text: str) -> tuple:
    """
    Parse the relevance-grade follow-up response. Returns (grade, rationale).
    An out-of-enum grade, bad JSON, or non-dict shape yields (None, None) -- never
    coerced or invented. A missing/empty rationale degrades to None while keeping a
    valid grade.
    """
    json_text = result_text
    if "```json" in json_text:
        json_text = json_text.split("```json")[1].split("```")[0]
    elif "```" in json_text:
        json_text = json_text.split("```")[1].split("```")[0]

    try:
        parsed = json.loads(json_text.strip())
    except json.JSONDecodeError:
        return None, None
    if not isinstance(parsed, dict):
        return None, None

    grade = parsed.get("relevance_grade")
    if grade not in RELEVANCE_GRADES:
        return None, None

    rationale = parsed.get("rationale")
    if not isinstance(rationale, str) or not rationale.strip():
        return grade, None
    return grade, rationale.strip()


def judge_moment(
    candidate: Dict[str, Any],
    window: Dict[str, Any],
    extracted_events: Dict[str, Any],
    model: str = "sonnet",
    stub: Optional[Callable] = None,
    cfg_dir: Optional[str] = None,
    parent_transcript: Optional[Dict[str, Any]] = None,
) -> Dict[str, Any]:
    """
    Judge a detected moment on the full scorecard.

    Merges LLM-judged fields with structural fields computed from extracted_events.

    Args:
        candidate: from detect_moments()
        window: window dict from split_into_windows()
        extracted_events: output from extract.extract_transcript()
        model: "haiku" or "sonnet"
        stub: optional callable for testing
        cfg_dir: optional persistent CLAUDE_CONFIG_DIR
        parent_transcript: optional parent transcript data for fork/fresh/workflow/pi_subagent moments (D-E).
                          # DEFERRED to task 5.1: the full-corpus runner has cross-transcript visibility to look up
                          # and pass the actual dispatching parent's dispatch_event here; a single-transcript function
                          # cannot do this cross-transcript join itself. For now, None is always passed from audit_transcript().

    Returns:
        complete moment record dict with all scorecard fields
    """
    # Build window text with explicit line numbers to prevent model self-counting errors.
    # Use absolute line numbers from the transcript (window["location_start"] + offset).
    window_lines_numbered = []
    for offset, line in enumerate(window["lines"]):
        absolute_line_number = window["location_start"] + offset
        # Prefix each line with its absolute line number (1-indexed)
        # Format: "NNN: <line content>" where NNN is the absolute line number
        numbered_line = f"{absolute_line_number}: {line}"
        window_lines_numbered.append(numbered_line)

    window_text = "".join(window_lines_numbered)
    location = candidate["location"]
    moment_type = candidate["moment_type"]

    # Structural fields from extracted_events
    recall_calls = extracted_events.get("recall_calls", [])
    dispatch_events = extracted_events.get("dispatch_events", [])
    learn_calls = extracted_events.get("learn_calls", [])
    engram_commands = extracted_events.get("engram_commands", [])
    dispatch_type = extracted_events.get("dispatch_type", "main")
    transcript_path = extracted_events.get("transcript_path", "")
    role = extracted_events.get("role", dispatch_type)

    # Filter to this window's location range
    window_recalls = [r for r in recall_calls if window["location_start"] <= r.get("location", 0) <= window["location_end"]]
    window_learns = [l for l in learn_calls if window["location_start"] <= l.get("location", 0) <= window["location_end"]]
    window_engrams = [e for e in engram_commands if window["location_start"] <= e.get("location", 0) <= window["location_end"]]
    window_dispatches = [d for d in dispatch_events if window["location_start"] <= d.get("location", 0) <= window["location_end"]]

    # D-E: Forked subagents inherit parent conversation; fresh subagents may have injected memory
    # If role is 'fork', automatically mark surfaced=true and search_ran='injected'
    search_ran = "none"
    surfaced = False
    surfaced_rank = None

    if role == "fork":
        # Fork: entire parent conversation is available (D-E rule)
        search_ran = "injected"
        surfaced = True
    elif role in ("fresh", "workflow", "pi_subagent") and parent_transcript and parent_transcript.get("memories_in_dispatch_prompt"):
        # Fresh/workflow/pi_subagent: check if parent dispatch had memories injected
        search_ran = "injected"
        surfaced = True
    else:
        # Main agent or fresh without injected memory: check for recalls in this window
        if window_recalls:
            # Check skill_args to distinguish quick_glance from full_recall
            for recall in window_recalls:
                skill_args = recall.get("skill_args", "")
                if "glance" in skill_args.lower():
                    search_ran = "quick_glance"
                else:
                    search_ran = "full_recall"
                break

        # Check surfaced in window recalls
        for recall in window_recalls:
            matched = recall.get("matched_items", [])
            if matched:
                surfaced = True
                surfaced_rank = matched[0].get("rank") if matched else None
                break

    # Resolve the fork's inherited recall context (D-E gap fix): the branch above only
    # sets structural fields for role=="fork" -- surfaced/search_ran -- but never shows
    # the LLM judgment call WHAT was recalled in the parent conversation before this
    # fork was dispatched, since this fork's own window text contains none of it.
    fork_recalled_notes = []
    if role == "fork":
        fork_context = _resolve_fork_parent_context(transcript_path)
        fork_recalled_notes = fork_context.get("recalled_notes", [])

    learn_fired = len(window_learns) > 0
    note_written = any(e.get("command") == "learn" for e in window_engrams)
    strength_updated = any(e.get("command") in ("amend", "activate") for e in window_engrams)

    # Extract timestamp from the moment's location
    timestamp = _extract_timestamp_from_window(window, location)

    # Memories in dispatch prompt (for dispatch moments)
    memories_in_dispatch_prompt = []
    if moment_type == "dispatch":
        for dispatch in window_dispatches:
            prompt_text = dispatch.get("prompt_text", "")
            # Improved heuristic: look for references to vault notes or specific memory patterns
            # This includes both explicit patterns and paraphrased references
            has_vault_ref = (
                "vault note" in prompt_text.lower() or
                "note-" in prompt_text or
                "precedent" in prompt_text.lower() or
                "reuse" in prompt_text.lower() or
                "prior" in prompt_text.lower() or
                "existing" in prompt_text.lower()
            )
            if has_vault_ref and prompt_text:
                memories_in_dispatch_prompt.append(prompt_text[:100])

    # Derive repo from transcript path
    repo = "unknown"
    if "/.claude/projects/" in transcript_path:
        parts = transcript_path.split("/.claude/projects/")
        if len(parts) > 1:
            repo_part = parts[1].split("/")[0]
            repo = repo_part

    # Determine harness
    harness = "claude_code"
    if "pi_main" in dispatch_type or "pi_subagent" in dispatch_type:
        harness = "pi"

    # Fork-inherited recall context (D-E gap fix): when non-empty, inject a section
    # ahead of the scorecard instructions naming what the fork's parent conversation
    # had already recalled and surfaced before this fork was dispatched -- the LLM
    # otherwise judges "followed" from this fork's own window text alone, which never
    # contains that content. Stays "" (contributing nothing to the template below) in
    # every other case: non-fork roles, or a fork whose resolution found nothing --
    # zero behavior change for those.
    fork_context_section = ""
    if fork_recalled_notes:
        notes_text = "".join(
            f"--- {note['path']} ({note['kind']}) ---\n{note['content']}\n\n"
            for note in fork_recalled_notes
        )
        fork_context_section = (
            "This is a forked subagent: it inherited its parent conversation's full "
            "context, including the following memory that had already been recalled "
            "and surfaced there before this fork was dispatched:\n\n"
            + notes_text
            + "Use this inherited context (in addition to the window below, which "
            "shows only this fork's own actions) to judge whether it was followed.\n\n"
        )

    # failure_category guidance covers ALL moment types, not just "failure"
    # (the old 'if moment_type="failure"' scoping left corrections, rework,
    # successes, and dispatches uncategorized). moment_type is already known
    # here, so build the type-appropriate instruction block conditionally:
    # - failure/correction/rework: the runbook-824 prevention question.
    # - success/dispatch: the capture-reachability variant (could a mandatory
    #   "record lessons in your completion report" step have captured the
    #   lesson, given worth_learning_from and no learn step fired?).
    # Instrument v2: "not_applicable" replaces the bare-null option in both
    # variants and in the JSON response schema line below (vault note 900:
    # nulls were unasked, not unanswerable).
    if moment_type in ("failure", "correction", "rework"):
        failure_category_section = (
            'For failure_category (moment_type="failure", "correction", or "rework"):\n'
            "Could a relevant memory, recalled at the right time, have prevented "
            "or caught this problem?\n"
            '- failure_category: "fixable" (could have been caught by memory), '
            '"nothing_could_have_caught_it", "found_but_not_followed", or "not_applicable"'
        )
    else:
        failure_category_section = (
            'For failure_category (moment_type="success" or "dispatch" -- capture reachability):\n'
            "IF this moment holds a lesson worth learning (see worth_learning_from) "
            "and no learn step fired, could a mandatory "
            '"record lessons in your completion report" step '
            "realistically have captured that lesson?\n"
            '- failure_category: "fixable" (yes, realistically capturable), '
            '"nothing_could_have_caught_it" (no plausible mechanism would have), '
            'or "not_applicable" (e.g. not worth learning from, or a learn '
            "step DID fire)"
        )

    # Prepare prompt for judgment call
    judgment_prompt = f"""You are judging a moment in a Claude Code session for quality assessment.

Moment type: {moment_type}
Description: {candidate['description']}
Location: line {location}

Window context (lines {window['location_start']}-{window['location_end']}):
Each line below is prefixed with its real line number (e.g., "47: {{"type":"user",...}}").

{window_text}

{fork_context_section}Judge the following fields (true/false/"not_applicable" for the binary questions -- see the conditional-field rule below):

For FINDING SIDE:
- search_targeted_right_thing: Did any search phrases in this window target the right thing for this moment?
- surfaced: Was a relevant memory present in search results? (Answerable: {surfaced})
- outdated_outranked_replacement: If a memory surfaced, was it outdated with a replacement outranking it?
- followed: Was the surfaced memory followed? ("yes"/"ignored"/"partial"/"skipped_step"/"out_of_order"/"contradicted"/"not_applicable")
  When surfaced is true, a bare null is NOT a permitted answer: if you cannot render any of the first six verdicts, "not_applicable" with a rationale is the honest answer.

For WRITING SIDE:
- worth_learning_from: Is this a moment worth capturing as a lesson?
- note_well_targeted: If a note was written, was it well-targeted to the moment's lesson?
- note_superseded_correctly: If a note superseded another, was the supersession correct?
- note_not_duplicate: If a note was written, is it not a duplicate of existing notes?
- strength_mismatch_flagged: Was any strength mismatch flagged (e.g., over/under-confident)?

{failure_category_section}

For EVERY conditional judged field (search_targeted_right_thing, outdated_outranked_replacement,
note_well_targeted, note_superseded_correctly, note_not_duplicate, strength_mismatch_flagged,
failure_category): when the field's condition does not hold for this moment, answer "not_applicable" -- never a bare null.
Do NOT answer null anywhere -- if you cannot render a verdict, "not_applicable" with a rationale is the honest answer.

Also include a "rationales" object in your response: a one-sentence rationale per answered question, keyed by field name.

Also generate 2-4 short search phrases representing what SHOULD be searched for at this moment,
based on what actually happened (separate from whether the agent's own search targeted the right thing).
These phrases will be used to check if a relevant memory existed at the time.

Respond with STRICT JSON ONLY (no prose wrapper):
{{
  "search_targeted_right_thing": true | false | "not_applicable",
  "outdated_outranked_replacement": true | false | "not_applicable",
  "followed": "yes" | "ignored" | "partial" | "skipped_step" | "out_of_order" | "contradicted" | "not_applicable",
  "worth_learning_from": true | false,
  "note_well_targeted": true | false | "not_applicable",
  "note_superseded_correctly": true | false | "not_applicable",
  "note_not_duplicate": true | false | "not_applicable",
  "strength_mismatch_flagged": true | false | "not_applicable",
  "failure_category": "fixable" | "nothing_could_have_caught_it" | "found_but_not_followed" | "not_applicable",
  "rationales": {{"<field>": "<one sentence>", ...}},
  "search_phrases": ["phrase 1", "phrase 2", "phrase 3"]
}}
"""

    response = _run_claude_p(judgment_prompt, model, stub=stub, cfg_dir=cfg_dir)

    # Deterministic, non-retryable failure: this window's content (once wrapped) is
    # too large for a single claude -p call. Distinct from exhausted_retries (which
    # means "possibly-transient failure, gave up after retrying") -- checked first so
    # it's never mistaken for that, or for a genuine default judgment.
    if response.get("prompt_too_long"):
        raise PromptTooLongError(
            f"judge_moment: moment at location {location} "
            f"(transcript={extracted_events.get('transcript_path')}, model={model}) "
            f"exceeds claude -p's prompt size limit -- split into smaller windows, "
            f"do not retry"
        )

    # Hard-stop signal: retries were genuinely exhausted (not a real "default judgment"
    # result). Must never fall through to the JSON-parse-failure path below, which
    # returns an indistinguishable empty-judgment dict.
    if response.get("exhausted_retries"):
        raise ExhaustedRetriesError(
            f"judge_moment: claude -p exhausted retries for moment at location {location} "
            f"(transcript={extracted_events.get('transcript_path')}, model={model})"
        )

    result_text = response.get("result", "")

    # Parse JSON
    json_text = result_text
    if "```json" in json_text:
        json_text = json_text.split("```json")[1].split("```")[0]
    elif "```" in json_text:
        json_text = json_text.split("```")[1].split("```")[0]

    try:
        judgment = json.loads(json_text)
    except json.JSONDecodeError:
        judgment = {}

    # Per-question rationales (instrument v2): stored on the record as
    # judgment_rationales. The relevance-grade follow-up below may add its own.
    judgment_rationales = _parse_judgment_rationales(judgment)

    # Extract search phrases for point-in-time check
    search_phrases = judgment.get("search_phrases", [])
    if not isinstance(search_phrases, list):
        search_phrases = []
    # Filter to 2-4 phrases, minimum 2 words each
    search_phrases = [p for p in search_phrases if isinstance(p, str) and len(p.split()) >= 2][:4]

    # Point-in-time memory check: populate memory_existed, memory_kind, memory_generic_or_specific, existence_derived_or_estimate
    memory_existed = None
    memory_kind = None
    memory_generic_or_specific = None
    existence_derived_or_estimate = None
    existence_check_error = None  # Bug 2: track errors separately from "not found"
    existence_top5 = None  # instrument v2: top-5 (path/kind/score) when items returned
    relevance_grade = None  # instrument v2: judged relevance of existence_top5

    # Only run the check if we have a timestamp and search phrases
    if timestamp and search_phrases:
        # Determine DERIVED vs ESTIMATE
        vault_repo_path = os.path.expanduser("~/.local/share/engram/vault")
        existence_derived_or_estimate = "estimate"  # default

        # Try to find a commit at or before this timestamp
        commit_sha = _find_commit_at_or_before(vault_repo_path, timestamp)

        # Check if we can use DERIVED mode (git checkout at a valid commit)
        can_use_derived = False
        if commit_sha:
            if _is_commit_ancestor_of(vault_repo_path, FLAT_VAULT_MIGRATION_COMMIT, commit_sha):
                can_use_derived = True

        if can_use_derived:
            # DERIVED path: check out the vault at that moment and query it
            existence_derived_or_estimate = "derived"

            scratch_dir = tempfile.mkdtemp(prefix="audit-ptc-")
            try:
                # Check out the vault at that moment
                with _checkout_vault_at(vault_repo_path, commit_sha, scratch_dir) as vault_path:
                    # Filter chunks by ingestion date
                    operator_data_dir_path = os.path.expanduser("~/.local/share/engram/chunks")
                    if os.path.exists(operator_data_dir_path):
                        chunks_path = _filter_chunks_by_ingestion_date(
                            operator_data_dir_path, timestamp, scratch_dir
                        )
                    else:
                        chunks_path = os.path.join(scratch_dir, "chunks-empty")
                        os.makedirs(chunks_path, exist_ok=True)

                    # Run engram query against the checked-out vault
                    query_result = _run_engram_query_at_moment(
                        vault_path, chunks_path, search_phrases
                    )

                    # Bug 2: Handle explicit error case (items is None = check failed)
                    items = query_result.get("items")
                    if items is None:
                        # The check itself failed
                        memory_existed = None
                        memory_kind = None
                        memory_generic_or_specific = None
                        existence_check_error = query_result.get("error", "Unknown error")
                    elif len(items) > 0:
                        # Successful check with results
                        memory_existed = True
                        # Instrument v2: capture the top-5 (path/kind/score) so the
                        # existence check is no longer a thresholdless "any items
                        # returned" gate -- the relevance follow-up below grades them.
                        existence_top5 = [
                            {
                                "path": item.get("path"),
                                "kind": item.get("kind"),
                                "score": item.get("score"),
                            }
                            for item in items[:EXISTENCE_TOP_K]
                        ]
                        # Get the top-ranked item's kind
                        top_item = items[0]
                        memory_kind = top_item.get("kind")
                        # Classify as generic or specific based on the query's top match
                        # For now: if score is high (>0.7), mark as specific; otherwise generic
                        score = top_item.get("score", 0)
                        memory_generic_or_specific = "specific" if score > 0.7 else "generic"
                    else:
                        # Successful check finding nothing
                        memory_existed = False
                        memory_kind = None
                        memory_generic_or_specific = None

            finally:
                shutil.rmtree(scratch_dir, ignore_errors=True)

        else:
            # ESTIMATE path: filter vault notes by creation date (live vault, read-only)
            existence_derived_or_estimate = "estimate"

            if os.path.exists(vault_repo_path):
                # Use live vault as fallback for ESTIMATE
                live_vault = vault_repo_path

                try:
                    # Filter notes by creation date
                    filtered_note_paths = _filter_notes_by_creation_date(live_vault, timestamp)

                    if filtered_note_paths:
                        # For ESTIMATE mode, we do a simple heuristic check: does any filtered
                        # note contain any of the search phrases (content match)?
                        #
                        # Scan ALL filtered notes, not a fixed prefix -- measured at ~0.1s to
                        # scan the entire ~800-note vault, well within budget, so no cap is
                        # needed (a cap would silently drop matches past its cutoff, since
                        # filtered_note_paths is in os.listdir order, not relevance order).
                        found_matches = False
                        for note_path in filtered_note_paths:
                            try:
                                with open(note_path, "r") as f:
                                    note_content = f.read()
                            except (IOError, OSError):
                                continue

                            note_content_lower = note_content.lower()
                            # Count how many distinct search phrases appear in the note, to
                            # derive generic_or_specific from real content overlap with the
                            # moment's own specifics (Finding 7), rather than a hardcoded
                            # constant. Majority-of-phrases match => the note echoes most of
                            # what the moment was specifically about => "specific"; a single
                            # incidental phrase match => reads more like a general principle
                            # => "generic".
                            matched_phrase_count = sum(
                                1 for phrase in search_phrases if phrase.lower() in note_content_lower
                            )
                            if matched_phrase_count > 0:
                                found_matches = True
                                memory_kind = _parse_note_type(note_content)
                                match_ratio = matched_phrase_count / len(search_phrases)
                                memory_generic_or_specific = "specific" if match_ratio >= 0.5 else "generic"
                                break

                        memory_existed = found_matches
                    else:
                        memory_existed = False
                        memory_kind = None
                        memory_generic_or_specific = None
                except (IOError, OSError) as e:
                    # Bug 2: surface errors from ESTIMATE path too
                    memory_existed = None
                    memory_kind = None
                    memory_generic_or_specific = None
                    existence_check_error = f"ESTIMATE mode failed: {e}"
            else:
                # Vault doesn't exist
                memory_existed = None
                memory_kind = None
                memory_generic_or_specific = None
                existence_check_error = "Vault path does not exist"

    # Instrument v2: when the existence check returned items, grade their relevance
    # against this moment with ONE more judgment question (a second call by
    # necessity -- the top-5 only exist after the existence check, which itself
    # needs the first call's search_phrases). Same hard-stop conventions as the
    # main judgment call: prompt_too_long and exhausted_retries raise, never
    # degrade to a silently-null grade.
    if existence_top5:
        relevance_prompt = _build_relevance_grade_prompt(
            moment_type,
            candidate["description"],
            location,
            window,
            window_text,
            existence_top5,
            search_phrases,
        )
        relevance_response = _run_claude_p(relevance_prompt, model, stub=stub, cfg_dir=cfg_dir)

        if relevance_response.get("prompt_too_long"):
            raise PromptTooLongError(
                f"judge_moment (relevance grade): moment at location {location} "
                f"(transcript={extracted_events.get('transcript_path')}, model={model}) "
                f"exceeds claude -p's prompt size limit -- split into smaller windows, "
                f"do not retry"
            )
        if relevance_response.get("exhausted_retries"):
            raise ExhaustedRetriesError(
                f"judge_moment (relevance grade): claude -p exhausted retries for moment "
                f"at location {location} "
                f"(transcript={extracted_events.get('transcript_path')}, model={model})"
            )

        relevance_grade, relevance_rationale = _parse_relevance_grade(
            relevance_response.get("result", "")
        )
        if relevance_rationale:
            judgment_rationales["relevance_grade"] = relevance_rationale

    # Build full moment record
    moment_record = {
        "moment_id": _moment_id(transcript_path, location),
        "transcript_path": transcript_path,
        "location": location,
        "timestamp": timestamp,
        "repo": repo,
        "harness": harness,
        "role": role,
        "moment_type": moment_type,
        "description": candidate["description"],
        "instrument_version": INSTRUMENT_VERSION,
        "evidence": {
            "transcript": os.path.basename(transcript_path),
            "anchor": str(location),
        },
        "finding_side": {
            "memory_existed": memory_existed,
            "memory_kind": memory_kind,
            "memory_generic_or_specific": memory_generic_or_specific,
            "existence_derived_or_estimate": existence_derived_or_estimate,
            "existence_check_error": existence_check_error,  # Bug 2: surface errors
            "existence_top5": existence_top5,
            "relevance_grade": relevance_grade,
            "search_ran": search_ran,
            "search_targeted_right_thing": judgment.get("search_targeted_right_thing"),
            "surfaced": surfaced,
            "surfaced_rank": surfaced_rank,
            "outdated_outranked_replacement": judgment.get("outdated_outranked_replacement"),
            "followed": judgment.get("followed"),
        },
        "writing_side": {
            "worth_learning_from": judgment.get("worth_learning_from"),
            "learn_fired": learn_fired,
            "note_written": note_written,
            "note_well_targeted": judgment.get("note_well_targeted"),
            "note_superseded_correctly": judgment.get("note_superseded_correctly"),
            "note_not_duplicate": judgment.get("note_not_duplicate"),
            "strength_updated": strength_updated,
            "strength_mismatch_flagged": judgment.get("strength_mismatch_flagged"),
        },
        "handoff": {
            "memories_in_dispatch_prompt": memories_in_dispatch_prompt,
            "memories_orchestrator_had_but_left_out": None,  # DEFERRED to task 3.2
        },
        "failure_category": judgment.get("failure_category"),
        "judgment_rationales": judgment_rationales,
    }

    # Instrument v2: apply the merged corpus's not_applicable convention to every
    # conditional judged field -- the field stays null, <field>_null_reason says why
    # (stamped only where a not_applicable verdict earned it).
    for na_field in _NOT_APPLICABLE_FINDING_FIELDS:
        _stamp_not_applicable(moment_record["finding_side"], na_field)
    for na_field in _NOT_APPLICABLE_WRITING_FIELDS:
        _stamp_not_applicable(moment_record["writing_side"], na_field)
    _stamp_not_applicable(moment_record, "failure_category")

    return moment_record


def audit_transcript(
    transcript_path: str,
    extract_func: Optional[Callable] = None,
    detect_stub: Optional[Callable] = None,
    judge_stub: Optional[Callable] = None,
) -> List[Dict[str, Any]]:
    """
    Main auditor runner: extract, window, detect, judge, and return moment records.

    Args:
        transcript_path: path to transcript JSONL file
        extract_func: optional replacement for extract.extract_transcript (for testing)
        detect_stub: optional stub for detect_moments (no real claude -p call)
        judge_stub: optional stub for judge_moment (no real claude -p call)

    Returns:
        list of moment records (dicts, ready for JSONL output)
    """
    # Import and run extractor
    if extract_func is None:
        from extract import extract_transcript
        extract_func = extract_transcript

    extracted = extract_func(transcript_path)

    # Split into windows
    windows = split_into_windows(transcript_path)

    # Build a persistent cfg_dir for real subprocess calls (reused across all windows)
    cfg_dir = None
    if detect_stub is None:  # Only build if we'll do real subprocess calls
        cfg_dir = tempfile.mkdtemp(prefix="audit-moments-cfg-")
        _build_cfg_dir(cfg_dir)

    try:
        all_moments = []

        for window in windows:
            # Detect candidates using haiku
            candidates = detect_moments(window, model="haiku", stub=detect_stub, cfg_dir=cfg_dir)

            # If low-yield, re-check with sonnet
            if len(candidates) < LOW_YIELD_THRESHOLD:
                sonnet_candidates = detect_moments(window, model="sonnet", stub=detect_stub, cfg_dir=cfg_dir)
                # Merge and dedupe by location
                candidates_by_loc = {c["location"]: c for c in candidates}
                for c in sonnet_candidates:
                    if c["location"] not in candidates_by_loc:
                        candidates_by_loc[c["location"]] = c
                candidates = list(candidates_by_loc.values())

            # Judge each candidate
            for candidate in candidates:
                # DEFERRED to task 5.1: the full-corpus runner has cross-transcript visibility to look up
                # and pass the actual dispatching parent's dispatch_event here; a single-transcript function
                # cannot do this cross-transcript join itself. For now, parent_transcript=None.
                moment = judge_moment(
                    candidate, window, extracted, model="sonnet", stub=judge_stub, cfg_dir=cfg_dir,
                    parent_transcript=None
                )
                all_moments.append(moment)

        return all_moments

    finally:
        # Clean up persistent cfg_dir
        if cfg_dir:
            shutil.rmtree(cfg_dir, ignore_errors=True)


if __name__ == "__main__":
    import sys

    if len(sys.argv) < 2:
        print("Usage: python audit_moments.py <transcript_path> [--output <output_path>]")
        sys.exit(1)

    transcript_path = sys.argv[1]
    output_path = None

    if "--output" in sys.argv:
        idx = sys.argv.index("--output")
        if idx + 1 < len(sys.argv):
            output_path = sys.argv[idx + 1]

    moments = audit_transcript(transcript_path)

    # Output as JSONL
    for moment in moments:
        line = json.dumps(moment)
        if output_path:
            with open(output_path, "a") as f:
                f.write(line + "\n")
        else:
            print(line)
