#!/usr/bin/env python3
"""Per-trial engram isolation for dev/eval harnesses.

Every harness here spawns `claude -p --permission-mode bypassPermissions` with the engram binary
reachable on the inherited PATH. Unless engram's state vars point at a per-trial directory, a
trial's `engram learn` writes to the operator's REAL vault and its `engram query` reads the
operator's REAL chunk index. Both have happened: #708 (seven real notes written during #687's
measurement) and #642 (a warm arm scoring 8/8 off the operator's global index — the answer key).

Import from any depth under dev/eval:

    import os, sys
    sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", ".."))
    import isolation

Stdlib only, and deliberately no import of harness.py, so dev/eval/traps/ can use it without
pulling in the cumulative harness's model tables.
"""
import hashlib
import os
import re

# The three vars that decide whether a trial reaches real memory. CLAUDE_CONFIG_DIR is checked
# separately: it is required to be set, but it is not an engram path so it is not compared against
# the operator's data dir.
ENGRAM_STATE_VARS = ("ENGRAM_VAULT_PATH", "ENGRAM_CHUNKS_DIR", "ENGRAM_TRANSCRIPT_DIR")

# `engram ingest --auto` walks the cwd's ancestors collecting .claude dirs and the nearest VCS
# repo for markdown. That is correct in production and catastrophic in an eval — vault note 296
# measured ~48 operator-global files swept into a per-trial index, displacing the planted chunk.
CWD_ANCESTOR_MARKERS = (".claude", ".git", ".hg", ".jj")

# How many note names to name in an assert_vault_unchanged failure. A bare count delta costs a
# diagnosis round-trip; the whole list would bury the signal on a large vault.
_NAMED_ON_FAILURE = 10


class IsolationError(RuntimeError):
    """A harness would let a trial reach the operator's real engram state."""


def operator_data_dir():
    """The engram data dir the binary would resolve to: $XDG_DATA_HOME/engram, else
    ~/.local/share/engram (internal/cli/ingest.go DataDirFromHome)."""
    xdg = os.environ.get("XDG_DATA_HOME")
    base = xdg if xdg else os.path.join(os.path.expanduser("~"), ".local", "share")
    return os.path.realpath(os.path.join(base, "engram"))


def operator_vault():
    """The operator's real vault directory."""
    return os.path.join(operator_data_dir(), "vault")


def project_slug(cwd):
    """Claude Code's project-dir name for a cwd: the realpath with every non-alphanumeric
    character mapped to '-'. Mirrors harness.py's _project_slug and traps' _slug — '.' becomes
    '-' too, so a '/'-only replace MISSES the session dir."""
    return re.sub(r"[^A-Za-z0-9-]", "-", os.path.realpath(cwd))


def _is_within(path, parent):
    resolved = os.path.realpath(path)
    return resolved == parent or resolved.startswith(parent + os.sep)


def _ancestor_marker(cwd):
    """The first CWD_ANCESTOR_MARKERS path found at or above cwd, or None."""
    current = os.path.realpath(cwd)
    while True:
        for marker in CWD_ANCESTOR_MARKERS:
            candidate = os.path.join(current, marker)
            if os.path.exists(candidate):
                return candidate
        parent = os.path.dirname(current)
        if parent == current:
            return None
        current = parent


def _note_names(vault):
    try:
        return sorted(n for n in os.listdir(vault) if n.endswith(".md"))
    except FileNotFoundError:
        return []


def assert_isolated(env, cwd=None):
    """Raise IsolationError unless env (and cwd, when given) keeps a trial off real memory."""
    data_dir = operator_data_dir()

    if not env.get("CLAUDE_CONFIG_DIR"):
        raise IsolationError(
            "CLAUDE_CONFIG_DIR is unset — the trial would use the operator's real Claude config"
        )

    for var in ENGRAM_STATE_VARS:
        value = env.get(var, "")
        if not value:
            raise IsolationError(
                f"{var} is unset — engram would resolve it to the operator's real state under "
                f"{data_dir}. Build the env with isolation.isolated_env()."
            )
        if _is_within(value, data_dir):
            raise IsolationError(
                f"{var}={value} resolves to {os.path.realpath(value)}, inside the operator's "
                f"engram data dir {data_dir}. Point it at a per-trial directory."
            )

    if cwd is not None:
        marker = _ancestor_marker(cwd)
        if marker is not None:
            raise IsolationError(
                f"trial cwd {cwd} has the ancestor {marker} — `engram ingest --auto` walks "
                "ancestors and would sweep operator-global sources into the per-trial index "
                "(vault note 296). Use a cwd under /tmp with no .claude or VCS ancestor."
            )


def isolated_env(cfg, trial_dir, cwd=None, base=None):
    """Build a subprocess env with every engram state path inside trial_dir.

    cfg       CLAUDE_CONFIG_DIR for the trial.
    trial_dir per-trial scratch root; vault/ and chunks/ are created inside it.
    cwd       the trial's working directory, when known — sets ENGRAM_TRANSCRIPT_DIR to the
              matching session dir, and is checked for forbidden ancestors.
    base      the env to build on (defaults to os.environ).
    """
    env = dict(os.environ if base is None else base)
    env["CLAUDE_CONFIG_DIR"] = cfg

    vault = os.path.join(trial_dir, "vault")
    chunks = os.path.join(trial_dir, "chunks")
    os.makedirs(vault, exist_ok=True)
    os.makedirs(chunks, exist_ok=True)
    env["ENGRAM_VAULT_PATH"] = vault
    env["ENGRAM_CHUNKS_DIR"] = chunks

    # `engram transcript` defaults to ~/.claude/projects/<slug> and IGNORES CLAUDE_CONFIG_DIR,
    # so a headless cell never finds its own session unless this points at THIS cfg's dir.
    projects = os.path.join(cfg, "projects")
    env["ENGRAM_TRANSCRIPT_DIR"] = os.path.join(projects, project_slug(cwd)) if cwd else projects

    assert_isolated(env, cwd)
    return env


def vault_fingerprint(vault=None):
    """(note count, sha256 over the sorted note basenames) for a vault. Defaults to the
    operator's real vault — the thing no trial may modify."""
    names = _note_names(vault if vault is not None else operator_vault())
    return len(names), hashlib.sha256("\n".join(names).encode()).hexdigest()


def assert_vault_unchanged(before, vault=None):
    """Raise IsolationError if the vault's note set moved since `before`.

    The backstop for a path the env vars do not cover — a subcommand resolving its own way, or a
    spawn site that skipped isolated_env entirely. Names the notes that appeared or vanished,
    because a bare count delta costs a diagnosis round-trip.
    """
    root = vault if vault is not None else operator_vault()
    after = vault_fingerprint(root)
    if after == before:
        return

    now = set(_note_names(root))
    # `before` is a fingerprint, not a name list, so the exact appeared/vanished split is only
    # recoverable when notes were ADDED — the common case, and the one #708 produced.
    appeared = sorted(now)[-_NAMED_ON_FAILURE:] if after[0] > before[0] else []
    detail = f" Newest notes: {', '.join(appeared)}." if appeared else ""
    raise IsolationError(
        f"vault {root} changed during the run: {before[0]} notes before, {after[0]} after. "
        f"A trial reached real memory.{detail}"
    )
