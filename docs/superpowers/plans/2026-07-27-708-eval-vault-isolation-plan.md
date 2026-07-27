# #708 Eval-Harness Vault Isolation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every `dev/eval` harness point engram's vault, chunk index, and transcript dir at a per-trial directory, so no trial can read or write the operator's real memory.

**Architecture:** One dependency-free module, `dev/eval/isolation.py`, owns the isolated environment and three assertions. Twenty spawn sites adopt it. The Go harness gains the chunks field its Python sibling already has. Three mechanisms that fail differently: the module makes isolation the default, a preflight raises when a site skips it, and a real-vault fingerprint asserted at process exit catches a resolution path neither anticipated.

**Tech Stack:** Python 3 stdlib only (`os`, `re`, `hashlib`), `pytest 9.0.2` for tests, Go 1.x behind the `targ` build tag for `dev/eval/*.go`.

Design doc: `docs/superpowers/plans/2026-07-27-708-eval-vault-isolation.md`.

## Global Constraints

- **Stdlib only.** `isolation.py` imports nothing outside the Python standard library and does not import `harness.py`. `dev/eval/traps/run.py` must be able to use it without pulling in the cumulative harness's model tables.
- **No property-based tests, no new `check-full` leg.** Property rigor is scoped to production code (vault note 556); `dev/eval` earns gate coverage under #710/#711.
- **Everything raises.** No warnings, no silent fallbacks, no default-to-safe. A missing or misdirected variable is an `IsolationError` naming the variable and the path it resolved to.
- **The operator data dir resolves the way the binary resolves it:** `$XDG_DATA_HOME/engram`, else `~/.local/share/engram`. Never hardcode a path.
- **Behavior-preserving at every call site.** These edits change where engram state lives, nothing else. No harness's measurement semantics, output schema, prompt text, model choice, or `bypassPermissions` mode changes.
- **Commit trailer is `AI-Used: [claude]`.**

---

### Task 1: The isolation module

**Files:**
- Create: `dev/eval/isolation.py`
- Test: `dev/eval/test_isolation.py`

**Interfaces:**
- Consumes: nothing (leaf module).
- Produces, for every later task:
  - `IsolationError(RuntimeError)`
  - `operator_data_dir() -> str`, `operator_vault() -> str`
  - `isolated_env(cfg: str, trial_dir: str, cwd: str | None = None, base: dict | None = None) -> dict`
  - `assert_isolated(env: dict, cwd: str | None = None) -> None`
  - `vault_fingerprint(vault: str | None = None) -> tuple[int, str]`
  - `assert_vault_unchanged(before: tuple[int, str], vault: str | None = None) -> None`

- [ ] **Step 1: Write the failing tests**

Create `dev/eval/test_isolation.py`:

```python
"""Guards for dev/eval/isolation.py — the per-trial engram isolation contract.

Every dev/eval harness spawns `claude -p --permission-mode bypassPermissions` with engram on
PATH. Without these vars pointed at a per-trial dir, a trial's `engram learn` writes to the
operator's real vault (#708) and its `engram query` reads the operator's real chunk index (#642).
"""
import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import isolation


def test_isolated_env_sets_all_four_vars(tmp_path):
    cfg = str(tmp_path / "cfg")
    trial = str(tmp_path / "trial")
    cwd = str(tmp_path / "ws")
    os.makedirs(cwd)

    env = isolation.isolated_env(cfg, trial, cwd=cwd, base={})

    assert env["CLAUDE_CONFIG_DIR"] == cfg
    assert env["ENGRAM_VAULT_PATH"] == os.path.join(trial, "vault")
    assert env["ENGRAM_CHUNKS_DIR"] == os.path.join(trial, "chunks")
    assert env["ENGRAM_TRANSCRIPT_DIR"].startswith(os.path.join(cfg, "projects"))


def test_isolated_env_creates_the_dirs(tmp_path):
    trial = str(tmp_path / "trial")
    env = isolation.isolated_env(str(tmp_path / "cfg"), trial, base={})

    assert os.path.isdir(env["ENGRAM_VAULT_PATH"])
    assert os.path.isdir(env["ENGRAM_CHUNKS_DIR"])


def test_isolated_env_preserves_base_entries(tmp_path):
    env = isolation.isolated_env(str(tmp_path / "cfg"), str(tmp_path / "t"),
                                 base={"PATH": "/usr/bin", "FOO": "bar"})

    assert env["PATH"] == "/usr/bin"
    assert env["FOO"] == "bar"


@pytest.mark.parametrize("missing", ["ENGRAM_VAULT_PATH", "ENGRAM_CHUNKS_DIR",
                                     "ENGRAM_TRANSCRIPT_DIR"])
def test_assert_isolated_rejects_unset_var(tmp_path, missing):
    env = isolation.isolated_env(str(tmp_path / "cfg"), str(tmp_path / "t"), base={})
    del env[missing]

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.assert_isolated(env)
    assert missing in str(exc.value)


@pytest.mark.parametrize("var", ["ENGRAM_VAULT_PATH", "ENGRAM_CHUNKS_DIR",
                                 "ENGRAM_TRANSCRIPT_DIR"])
def test_assert_isolated_rejects_path_inside_operator_data_dir(tmp_path, monkeypatch, var):
    fake_home = tmp_path / "xdg"
    monkeypatch.setenv("XDG_DATA_HOME", str(fake_home))
    env = isolation.isolated_env(str(tmp_path / "cfg"), str(tmp_path / "t"), base={})
    env[var] = str(fake_home / "engram" / "vault")

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.assert_isolated(env)
    assert var in str(exc.value)
    assert "engram" in str(exc.value)


def test_assert_isolated_rejects_cwd_with_dotclaude_ancestor(tmp_path):
    # `ingest --auto` walks the cwd's ANCESTORS collecting .claude dirs — correct in production,
    # but in an eval it swept ~48 operator-global files into a supposedly-isolated index and
    # confounded the measurement (vault note 296). Env isolation alone does not prevent it.
    os.makedirs(tmp_path / "outer" / ".claude")
    cwd = tmp_path / "outer" / "inner" / "ws"
    os.makedirs(cwd)
    env = isolation.isolated_env(str(tmp_path / "cfg"), str(tmp_path / "t"), base={})

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.assert_isolated(env, cwd=str(cwd))
    assert ".claude" in str(exc.value)


def test_assert_isolated_rejects_cwd_with_git_ancestor(tmp_path):
    os.makedirs(tmp_path / "repo" / ".git")
    cwd = tmp_path / "repo" / "sub"
    os.makedirs(cwd)
    env = isolation.isolated_env(str(tmp_path / "cfg"), str(tmp_path / "t"), base={})

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.assert_isolated(env, cwd=str(cwd))
    assert ".git" in str(exc.value)


def test_assert_isolated_accepts_a_clean_trial(tmp_path):
    cwd = tmp_path / "ws"
    os.makedirs(cwd)
    env = isolation.isolated_env(str(tmp_path / "cfg"), str(tmp_path / "t"),
                                 cwd=str(cwd), base={})

    isolation.assert_isolated(env, cwd=str(cwd))  # must not raise


def test_vault_fingerprint_counts_notes_and_ignores_sidecars(tmp_path):
    vault = tmp_path / "vault"
    os.makedirs(vault)
    (vault / "1.note.md").write_text("a")
    (vault / "2.note.md").write_text("b")
    (vault / "2.note.vec.json").write_text("[]")

    count, digest = isolation.vault_fingerprint(str(vault))

    assert count == 2
    assert digest


def test_vault_fingerprint_of_missing_vault_is_empty(tmp_path):
    count, _ = isolation.vault_fingerprint(str(tmp_path / "nope"))
    assert count == 0


def test_assert_vault_unchanged_passes_when_untouched(tmp_path):
    vault = tmp_path / "vault"
    os.makedirs(vault)
    (vault / "1.note.md").write_text("a")

    before = isolation.vault_fingerprint(str(vault))
    isolation.assert_vault_unchanged(before, str(vault))  # must not raise


def test_assert_vault_unchanged_raises_when_a_note_appears(tmp_path):
    vault = tmp_path / "vault"
    os.makedirs(vault)
    before = isolation.vault_fingerprint(str(vault))
    (vault / "533.leaked.md").write_text("a trial wrote this")

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.assert_vault_unchanged(before, str(vault))
    assert "533.leaked.md" in str(exc.value)


def test_operator_data_dir_follows_xdg(tmp_path, monkeypatch):
    monkeypatch.setenv("XDG_DATA_HOME", str(tmp_path))
    assert isolation.operator_data_dir() == os.path.realpath(str(tmp_path / "engram"))


def test_operator_data_dir_falls_back_to_local_share(tmp_path, monkeypatch):
    monkeypatch.delenv("XDG_DATA_HOME", raising=False)
    monkeypatch.setenv("HOME", str(tmp_path))
    expected = os.path.realpath(str(tmp_path / ".local" / "share" / "engram"))
    assert isolation.operator_data_dir() == expected
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/test_isolation.py -v
```

Expected: collection error — `ModuleNotFoundError: No module named 'isolation'`.

- [ ] **Step 3: Write the module**

Create `dev/eval/isolation.py`:

```python
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
    root = vault if vault is not None else operator_vault()
    try:
        names = sorted(n for n in os.listdir(root) if n.endswith(".md"))
    except FileNotFoundError:
        names = []
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

    try:
        current = sorted(n for n in os.listdir(root) if n.endswith(".md"))
    except FileNotFoundError:
        current = []
    raise IsolationError(
        f"vault {root} changed during the run: {before[0]} notes before, {after[0]} after. "
        f"A trial reached real memory. Current notes: {', '.join(current[-10:]) or '(none)'}"
    )
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/test_isolation.py -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add dev/eval/isolation.py dev/eval/test_isolation.py
git commit -m "feat(eval): add per-trial engram isolation module

AI-Used: [claude]"
```

---

### Task 2: Tier A adoption — the seven un-isolated spawn sites

**Files:**
- Modify: `dev/eval/cumulative/please_step7_probe/run_probe.py:341`
- Modify: `dev/eval/cumulative/please_step3_probe/run_probe.py:328`
- Modify: `dev/eval/cumulative/endorse_cue/probe.py:149-150`
- Modify: `dev/eval/traps/brun.py:38` and `:61`
- Modify: `dev/eval/traps/run.py:41`
- Modify: `dev/eval/traps/qanchor_eval.py:85`
- Modify: `dev/eval/traps/reasoning_eval.py:93`
- Test: `dev/eval/test_tier_a_sites.py`

**Interfaces:**
- Consumes: `isolation.isolated_env`, `isolation.IsolationError` from Task 1.
- Produces: nothing new. Each `run_one`-style function gains a per-trial directory derived from
  its existing trial working dir.

- [ ] **Step 1: Write the failing test**

Create `dev/eval/test_tier_a_sites.py`:

```python
"""Every tier-A spawn site must build its env through isolation.isolated_env.

These seven sites hand-rolled `env = dict(os.environ)` and set only CLAUDE_CONFIG_DIR, which is
how #687's measurement wrote seven real notes into the operator's vault. A regression to a
hand-rolled env reopens the leak, so this asserts on the source text — the sites are in three
directories with no shared entry point to instrument.
"""
import os
import re

HERE = os.path.dirname(os.path.abspath(__file__))

TIER_A = [
    "cumulative/please_step7_probe/run_probe.py",
    "cumulative/please_step3_probe/run_probe.py",
    "cumulative/endorse_cue/probe.py",
    "traps/brun.py",
    "traps/run.py",
    "traps/qanchor_eval.py",
    "traps/reasoning_eval.py",
]

HANDROLLED = re.compile(r"^\s*env\s*=\s*dict\(os\.environ\)", re.M)


def _src(rel):
    with open(os.path.join(HERE, rel)) as f:
        return f.read()


def test_tier_a_sites_import_isolation():
    for rel in TIER_A:
        assert "import isolation" in _src(rel), f"{rel} does not import the isolation module"


def test_tier_a_sites_call_isolated_env():
    for rel in TIER_A:
        assert "isolation.isolated_env(" in _src(rel), f"{rel} does not call isolated_env"


def test_tier_a_sites_have_no_handrolled_env():
    for rel in TIER_A:
        assert not HANDROLLED.search(_src(rel)), (
            f"{rel} still builds a subprocess env by hand — route it through isolated_env"
        )
```

- [ ] **Step 2: Run it to verify it fails**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/test_tier_a_sites.py -v
```

Expected: all three tests FAIL, listing the seven files.

- [ ] **Step 3: Edit each site**

At the top of each file, after the existing imports, add the shim. The relative depth differs —
`traps/*.py` and `cumulative/*/probe.py` are one and two levels below `dev/eval` respectively:

For `dev/eval/traps/*.py` (one level down):

```python
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import isolation
```

For `dev/eval/cumulative/<probe>/run_probe.py` and `endorse_cue/probe.py` (two levels down):

```python
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", ".."))
import isolation
```

Then in each spawn function replace the hand-rolled env. Example for
`please_step7_probe/run_probe.py`'s `run_one` — the others follow the same shape, substituting
their own `wd` variable and any extra vars they already set:

```python
def run_one(cfg, role, skill_text, marker, model, pressure, idx):
    wd = build_trial_project(role, skill_text, marker)
    trial_dir = tempfile.mkdtemp(prefix="please-step7-probe-state-")
    env = isolation.isolated_env(cfg, trial_dir, cwd=wd)
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "8000"
```

Preserve every extra variable the site already set (`CLAUDE_CODE_MAX_OUTPUT_TOKENS`, `PATH`
prefixes, stub env vars) by assigning it after the `isolated_env` call.

- [ ] **Step 4: Run the test to verify it passes**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/test_tier_a_sites.py dev/eval/test_isolation.py -v
```

Expected: all pass.

- [ ] **Step 5: Verify each edited file still parses and its own tests pass**

```bash
cd /Users/joe/repos/personal/engram
for f in dev/eval/cumulative/please_step7_probe/run_probe.py \
         dev/eval/cumulative/please_step3_probe/run_probe.py \
         dev/eval/cumulative/endorse_cue/probe.py \
         dev/eval/traps/brun.py dev/eval/traps/run.py \
         dev/eval/traps/qanchor_eval.py dev/eval/traps/reasoning_eval.py; do
  python3 -m py_compile "$f" && echo "OK $f"
done
python3 -m pytest dev/eval/cumulative/please_step7_probe/test_scorer_cases.py -q
```

Expected: seven `OK` lines, scorer suite passes.

- [ ] **Step 6: Commit**

```bash
git add dev/eval/cumulative/please_step7_probe/run_probe.py \
        dev/eval/cumulative/please_step3_probe/run_probe.py \
        dev/eval/cumulative/endorse_cue/probe.py \
        dev/eval/traps/brun.py dev/eval/traps/run.py \
        dev/eval/traps/qanchor_eval.py dev/eval/traps/reasoning_eval.py \
        dev/eval/test_tier_a_sites.py
git commit -m "fix(eval): isolate engram state at the seven un-isolated spawn sites

AI-Used: [claude]"
```

---

### Task 3: harness.claude delegation and the tier-B Python sites

**Files:**
- Modify: `dev/eval/cumulative/harness.py:78-96` (`claude`), `:319`, `:828`
- Modify: `dev/eval/traps/retrieval_probe.py:97`, `seed_c3.py:47`, `crowd.py:194`,
  `synth_fixtures.py:73`, `compound_fixtures.py:169`, `cake_fixtures.py:29` and `:49`,
  `qanchor_retrieval_probe.py:36`
- Test: `dev/eval/cumulative/test_chunks_isolation.py` (extend), `dev/eval/test_tier_b_sites.py`

**Interfaces:**
- Consumes: `isolation.isolated_env`, `isolation.assert_isolated` from Task 1.
- Produces: `harness.claude()` keeps its exact signature —
  `claude(cfg, model, vault, cwd, prompt, resume_sid=None, chunks=None, extra_env=None)` — so no
  caller changes. Its env construction moves behind `isolated_env`.

- [ ] **Step 1: Write the failing test**

Create `dev/eval/test_tier_b_sites.py`:

```python
"""Tier B: sites that set ENGRAM_VAULT_PATH but leave ENGRAM_CHUNKS_DIR unset.

An unset ENGRAM_CHUNKS_DIR silently resolves to the operator's global index. That is #642: a warm
arm read the operator's live sessions — the eval answer key — and scored round-1 8/8 off an
otherwise empty vault. Setting the vault alone is the half-fix that looks done.
"""
import os
import re

HERE = os.path.dirname(os.path.abspath(__file__))

TIER_B = [
    "traps/retrieval_probe.py",
    "traps/seed_c3.py",
    "traps/crowd.py",
    "traps/synth_fixtures.py",
    "traps/compound_fixtures.py",
    "traps/cake_fixtures.py",
    "traps/qanchor_retrieval_probe.py",
]

SETS_VAULT = re.compile(r"ENGRAM_VAULT_PATH")
SETS_CHUNKS = re.compile(r"ENGRAM_CHUNKS_DIR")


def _src(rel):
    with open(os.path.join(HERE, rel)) as f:
        return f.read()


def test_no_site_sets_a_vault_without_a_chunks_dir():
    for rel in TIER_B:
        src = _src(rel)
        if SETS_VAULT.search(src):
            assert SETS_CHUNKS.search(src), (
                f"{rel} sets ENGRAM_VAULT_PATH but never ENGRAM_CHUNKS_DIR — reads fall through "
                "to the operator's global chunk index (#642)"
            )
```

Append to `dev/eval/cumulative/test_chunks_isolation.py`:

```python
def test_claude_env_goes_through_the_isolation_module(monkeypatch, tmp_path):
    """harness.claude must not hand-roll its env — one implementation, not two."""
    import harness
    src = open(harness.__file__).read()
    assert "isolation.isolated_env(" in src, (
        "harness.claude should delegate env construction to isolation.isolated_env"
    )
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/test_tier_b_sites.py dev/eval/cumulative/test_chunks_isolation.py -v
```

Expected: `test_no_site_sets_a_vault_without_a_chunks_dir` FAILS naming the seven files;
`test_claude_env_goes_through_the_isolation_module` FAILS.

- [ ] **Step 3: Rewrite `harness.claude`'s env block**

Replace `dev/eval/cumulative/harness.py:79-92` with:

```python
def claude(cfg, model, vault, cwd, prompt, resume_sid=None, chunks=None, extra_env=None):
    # Env construction lives in isolation.isolated_env so there is ONE implementation of the
    # isolation contract, not two. vault/chunks stay caller-supplied here because the cumulative
    # harness promotes them between apps in a chain; isolated_env's defaults apply elsewhere.
    trial_dir = chunks_parent(vault, chunks)
    env = isolation.isolated_env(cfg, trial_dir, cwd=cwd)
    if vault and vault != "none":
        env["ENGRAM_VAULT_PATH"] = vault
    if chunks:
        env["ENGRAM_CHUNKS_DIR"] = chunks
    env["CLAUDE_CODE_MAX_OUTPUT_TOKENS"] = "64000"
    env["PATH"] = ENGRAM_BIN_DIR + ":" + env.get("PATH", "")
    if extra_env:
        env.update(extra_env)  # caller overrides last (e.g. the C7 recheck stub PATH + stub vars)
    isolation.assert_isolated(env)
```

Add the helper immediately above `claude`:

```python
def chunks_parent(vault, chunks):
    """Scratch root for isolated_env's defaults. The cumulative harness supplies its own vault and
    chunks paths (it promotes them between apps in a chain), so this only needs to be a writable
    per-cell directory that is not inside the operator's data dir."""
    for candidate in (chunks, vault):
        if candidate and candidate != "none":
            return os.path.dirname(os.path.abspath(candidate))
    return tempfile.mkdtemp(prefix="engram-eval-state-")
```

Add `import tempfile` to harness.py's import line and the isolation shim:

```python
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
import isolation
```

Note the deliberate ordering: `isolated_env` sets defaults and asserts, then the caller's explicit
vault/chunks override them, then `assert_isolated` runs again so an override cannot smuggle a real
path back in.

- [ ] **Step 4: Add the missing chunks var at each tier-B Python site**

Each of the seven sets a vault beside a `wd` or `ROOT`. Add the chunks line next to it, e.g. in
`dev/eval/traps/retrieval_probe.py:97`:

```python
    env["ENGRAM_VAULT_PATH"] = vault_path
    env["ENGRAM_CHUNKS_DIR"] = os.path.join(os.path.dirname(vault_path), "chunks")
    os.makedirs(env["ENGRAM_CHUNKS_DIR"], exist_ok=True)
```

Repeat with each file's own vault variable name (`vault_path` in `retrieval_probe.py`,
`seed_c3.py`, `crowd.py`; `vault` in `synth_fixtures.py`, `compound_fixtures.py`,
`cake_fixtures.py` both sites, `qanchor_retrieval_probe.py`).

- [ ] **Step 5: Run the tests**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/test_tier_b_sites.py dev/eval/cumulative/test_chunks_isolation.py dev/eval/test_isolation.py dev/eval/test_tier_a_sites.py -v
```

Expected: all pass.

- [ ] **Step 6: Commit**

```bash
git add dev/eval/cumulative/harness.py dev/eval/cumulative/test_chunks_isolation.py \
        dev/eval/test_tier_b_sites.py dev/eval/traps/retrieval_probe.py \
        dev/eval/traps/seed_c3.py dev/eval/traps/crowd.py dev/eval/traps/synth_fixtures.py \
        dev/eval/traps/compound_fixtures.py dev/eval/traps/cake_fixtures.py \
        dev/eval/traps/qanchor_retrieval_probe.py
git commit -m "fix(eval): route harness.claude through isolation, close the tier-B chunks gap

AI-Used: [claude]"
```

---

### Task 4: The shell harnesses

**Files:**
- Modify: `dev/eval/run-chain-stage.sh:51`, `dev/eval/run-layer-arm.sh:47`,
  `dev/eval/run-layer-resume.sh:33`
- Test: `dev/eval/test_shell_sites.py`

**Interfaces:**
- Consumes: nothing — shell cannot import the Python module.
- Produces: nothing.

- [ ] **Step 1: Write the failing test**

Create `dev/eval/test_shell_sites.py`:

```python
"""The shell harnesses cannot import isolation.py, so they carry the contract inline."""
import os

HERE = os.path.dirname(os.path.abspath(__file__))
SHELL = ["run-chain-stage.sh", "run-layer-arm.sh", "run-layer-resume.sh"]


def test_shell_harnesses_set_a_chunks_dir():
    for name in SHELL:
        with open(os.path.join(HERE, name)) as f:
            src = f.read()
        assert "ENGRAM_CHUNKS_DIR" in src, (
            f"{name} sets ENGRAM_VAULT_PATH but not ENGRAM_CHUNKS_DIR — reads fall through to "
            "the operator's global chunk index (#642)"
        )
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/test_shell_sites.py -v
```

Expected: FAIL naming all three scripts.

- [ ] **Step 3: Add the chunks dir to each script**

In each, beside the existing `VAULT` definition, add:

```bash
CHUNKS="${VAULT}.chunks"
mkdir -p "$CHUNKS"
```

and add `ENGRAM_CHUNKS_DIR="$CHUNKS"` to the `env ...` invocation next to the existing
`ENGRAM_VAULT_PATH="$VAULT"`. The `<vault>.chunks` sibling convention matches what
`harness.py` already uses when it promotes a chain's chunk index between apps.

- [ ] **Step 4: Run the test and shellcheck the scripts**

```bash
cd /Users/joe/repos/personal/engram
python3 -m pytest dev/eval/test_shell_sites.py -v
for f in dev/eval/run-chain-stage.sh dev/eval/run-layer-arm.sh dev/eval/run-layer-resume.sh; do
  bash -n "$f" && echo "OK $f"
done
```

Expected: test passes, three `OK` lines.

- [ ] **Step 5: Commit**

```bash
git add dev/eval/run-chain-stage.sh dev/eval/run-layer-arm.sh dev/eval/run-layer-resume.sh \
        dev/eval/test_shell_sites.py
git commit -m "fix(eval): give the shell harnesses an isolated chunk index

AI-Used: [claude]"
```

---

### Task 5: The Go harness gains a chunks dir

**Files:**
- Modify: `dev/eval/deps.go:8-15` (`AgentInvocation`)
- Modify: `dev/eval/adapters.go:167-180` (`agentEnv`)
- Modify: `dev/targs.go:44-79` (`runEval`)
- Test: `dev/eval/adapters_test.go` (create if absent)

**Interfaces:**
- Consumes: nothing from earlier tasks — this is the Go half.
- Produces: `AgentInvocation.ChunksDir string`, emitted by `agentEnv` as `ENGRAM_CHUNKS_DIR`.

- [ ] **Step 1: Write the failing test**

Create or extend `dev/eval/adapters_test.go`:

```go
//go:build targ

package eval

import (
	"slices"
	"testing"
)

// A vault without a chunks dir is the #642 half-fix: reads fall through to the operator's global
// chunk index. AgentInvocation must carry both, and agentEnv must emit both.
func TestT708_AgentEnvEmitsChunksDir(t *testing.T) {
	t.Parallel()

	env := agentEnv(AgentInvocation{
		ConfigDir: "/iso/cfg",
		VaultPath: "/iso/vault",
		ChunksDir: "/iso/chunks",
	})

	if !slices.Contains(env, "ENGRAM_CHUNKS_DIR=/iso/chunks") {
		t.Fatalf("agentEnv did not emit ENGRAM_CHUNKS_DIR; got %v", env)
	}
	if !slices.Contains(env, "ENGRAM_VAULT_PATH=/iso/vault") {
		t.Fatalf("agentEnv did not emit ENGRAM_VAULT_PATH; got %v", env)
	}
}

func TestT708_AgentEnvOmitsEmptyChunksDir(t *testing.T) {
	t.Parallel()

	env := agentEnv(AgentInvocation{ConfigDir: "/iso/cfg", VaultPath: "/iso/vault"})

	for _, entry := range env {
		if entry == "ENGRAM_CHUNKS_DIR=" {
			t.Fatalf("agentEnv emitted an empty ENGRAM_CHUNKS_DIR; got %v", env)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd /Users/joe/repos/personal/engram && go test -tags=targ ./dev/eval/ -run TestT708 -v
```

Expected: compile error — `unknown field ChunksDir in struct literal`.

- [ ] **Step 3: Add the field and emit it**

In `dev/eval/deps.go`, add to `AgentInvocation` after `VaultPath`:

```go
	ChunksDir  string // exported as ENGRAM_CHUNKS_DIR (the per-run chunk index)
```

In `dev/eval/adapters.go`'s `agentEnv`, after the `VaultPath` block:

```go
	if inv.ChunksDir != "" {
		env = append(env, "ENGRAM_CHUNKS_DIR="+inv.ChunksDir)
	}
```

In `dev/targs.go`'s `runEval`, give the run its own chunk index rather than letting resolution
fall through to the operator's global one. Add to the `eval.RunConfig` literal a per-run chunks
path and thread it to wherever `AgentInvocation` is constructed — mirror how `VaultSrc`/the vault
clone already flows.

- [ ] **Step 4: Run the tests**

```bash
cd /Users/joe/repos/personal/engram && go test -tags=targ ./dev/... -run TestT708 -v && targ test-dev
```

Expected: both new tests pass; `test-dev` green.

- [ ] **Step 5: Commit**

```bash
git add dev/eval/deps.go dev/eval/adapters.go dev/eval/adapters_test.go dev/targs.go
git commit -m "fix(eval): give the Go harness an isolated chunk index

AI-Used: [claude]"
```

---

### Task 6: Post-run vault guard in every harness main()

**Files:**
- Modify: the `main()` of each of the seven tier-A harnesses from Task 2
- Test: `dev/eval/test_vault_guard.py`

**Interfaces:**
- Consumes: `isolation.vault_fingerprint`, `isolation.assert_vault_unchanged` from Task 1.
- Produces: nothing.

- [ ] **Step 1: Write the failing test**

Create `dev/eval/test_vault_guard.py`:

```python
"""Every harness that spawns trials brackets its run with the real-vault fingerprint.

This is the backstop for a path the env vars do not cover — a subcommand resolving its own way,
or a future spawn site that skips isolated_env. It catches contamination at exit instead of days
later as unexplained real notes (#708).
"""
import os

HERE = os.path.dirname(os.path.abspath(__file__))

HARNESSES = [
    "cumulative/please_step7_probe/run_probe.py",
    "cumulative/please_step3_probe/run_probe.py",
    "cumulative/endorse_cue/probe.py",
    "traps/brun.py",
    "traps/run.py",
    "traps/qanchor_eval.py",
    "traps/reasoning_eval.py",
]


def _src(rel):
    with open(os.path.join(HERE, rel)) as f:
        return f.read()


def test_every_harness_fingerprints_the_real_vault():
    for rel in HARNESSES:
        src = _src(rel)
        assert "vault_fingerprint(" in src, f"{rel} never snapshots the real vault"
        assert "assert_vault_unchanged(" in src, f"{rel} never asserts the vault is unchanged"
```

- [ ] **Step 2: Run to verify it fails**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/test_vault_guard.py -v
```

Expected: FAIL naming all seven.

- [ ] **Step 3: Bracket each main()**

In each harness's `main()`, immediately before the trial loop:

```python
    vault_before = isolation.vault_fingerprint()
```

and at the end of `main()`, before any `shutil.rmtree` cleanup:

```python
    isolation.assert_vault_unchanged(vault_before)
```

- [ ] **Step 4: Run the tests**

```bash
cd /Users/joe/repos/personal/engram && python3 -m pytest dev/eval/ -v
```

Expected: every test in `dev/eval/` passes.

- [ ] **Step 5: Commit**

```bash
git add dev/eval/cumulative/please_step7_probe/run_probe.py \
        dev/eval/cumulative/please_step3_probe/run_probe.py \
        dev/eval/cumulative/endorse_cue/probe.py \
        dev/eval/traps/brun.py dev/eval/traps/run.py \
        dev/eval/traps/qanchor_eval.py dev/eval/traps/reasoning_eval.py \
        dev/eval/test_vault_guard.py
git commit -m "feat(eval): assert the operator's vault is unchanged after every run

AI-Used: [claude]"
```

---

### Task 7: Manual end-to-end verification

**Files:** none modified. This task produces evidence, not code.

**Interfaces:**
- Consumes: everything from Tasks 1–6.
- Produces: the verification record pasted into #708's closing comment.

Passing tests do not establish that the feature works. The bar is a real trial, against the
installed binary, leaving the real vault untouched.

- [ ] **Step 1: Record the real vault's state**

```bash
cd /Users/joe/repos/personal/engram
V="${XDG_DATA_HOME:-$HOME/.local/share}/engram/vault"
ls "$V"/*.md | wc -l
python3 -c "
import sys; sys.path.insert(0, 'dev/eval'); import isolation
print(isolation.vault_fingerprint())"
```

Record both numbers.

- [ ] **Step 2: Confirm the preflight actually raises**

```bash
cd /Users/joe/repos/personal/engram && python3 -c "
import sys, os; sys.path.insert(0, 'dev/eval'); import isolation
env = {'CLAUDE_CONFIG_DIR': '/tmp/cfg'}
try:
    isolation.assert_isolated(env)
    print('FAIL: no raise')
except isolation.IsolationError as e:
    print('OK raises:', e)
"
```

Expected: `OK raises: ENGRAM_VAULT_PATH is unset ...`.

- [ ] **Step 3: Run one real probe trial**

```bash
cd /Users/joe/repos/personal/engram
python3 dev/eval/cumulative/please_step7_probe/run_probe.py \
  --role loaded_auditor --n 1 --model sonnet \
  --out /tmp/708-verify.jsonl
```

Expected: the trial completes and `main()` exits without an `IsolationError`.

- [ ] **Step 4: Confirm the real vault did not move, and the trial vault did**

```bash
cd /Users/joe/repos/personal/engram
python3 -c "
import sys; sys.path.insert(0, 'dev/eval'); import isolation
print('real vault now:', isolation.vault_fingerprint())"
ls -t /tmp/please-step7-probe-state-*/vault/ 2>/dev/null | head
```

Expected: the real-vault fingerprint matches Step 1 exactly. If the trial's skill wrote a note,
it is in the scratch vault. Both halves matter — an unchanged real vault with an empty scratch
vault means the trial never exercised the write path, which is not a pass.

- [ ] **Step 5: Record the evidence**

Paste the Step 1 and Step 4 fingerprints, the Step 2 raise message, and the Step 3 exit status
into #708's closing comment.

---

### Task 8: Docs, roadmap, close

**Files:**
- Modify: `docs/ROADMAP.md`
- Modify: `dev/eval/LEDGER.md`

**Interfaces:**
- Consumes: Task 7's evidence.
- Produces: nothing.

- [ ] **Step 1: Fix the roadmap drift found during the what's-next briefing**

In `docs/ROADMAP.md`:
- Move NOW rank 1 (#687) out of the NOW band — it closed 2026-07-27. Its Provenance row already
  exists; delete the NOW row and renumber the band.
- Correct the #687 Provenance row's stale "not deployed via `engram update`" claim — the residue
  is present in both `~/.claude/skills/please/SKILL.md` and `~/.pi/agent/skills/please/SKILL.md`.
- Reconcile #696: the Provenance table records it Shipped 2026-07-24 while the issue is open.
- Score and place the unplaced issues: #708 (this work), #709, #702, #701, #710, #711.

- [ ] **Step 2: Add the LEDGER row**

Add a `#708-eval-vault-isolation` row recording the site counts (7 tier A, 13 tier B), the three
mechanisms, and Task 7's before/after fingerprints.

- [ ] **Step 3: Run the full gate**

```bash
cd /Users/joe/repos/personal/engram && targ check-full
```

Expected: `PASS:8 FAIL:0`.

- [ ] **Step 4: Commit**

```bash
git add docs/ROADMAP.md dev/eval/LEDGER.md
git commit -m "docs(708): record eval vault isolation; re-place the roadmap's stale rows

AI-Used: [claude]"
```

---

## Self-Review

**Spec coverage.** Every element of the design maps to a task: the module and its four functions
(Task 1), tier A's seven sites (Task 2), `harness.claude` delegation and tier B's Python sites
(Task 3), tier B's shell sites (Task 4), the Go `ChunksDir` (Task 5), the post-run fingerprint
(Task 6), manual E2E (Task 7), docs (Task 8). The cwd-ancestor guard Joe ratified is in Task 1's
`assert_isolated` with two dedicated tests.

**Placeholder scan.** No TBD/TODO. Every code step carries the actual code. Task 5's Step 3
`runEval` change is the one instruction phrased as "mirror how VaultSrc flows" rather than exact
code — that file's construction site is a single literal, and the exact edit depends on reading
it; flagged here as the one place the implementer reads before typing.

**Type consistency.** `isolated_env(cfg, trial_dir, cwd, base)` is called with that signature in
Tasks 2 and 3. `vault_fingerprint()` returns a tuple consumed by `assert_vault_unchanged(before)`
in Task 6 and printed in Task 7. `IsolationError` is raised in Task 1 and caught by name in Task
7. The Go field is `ChunksDir` in `deps.go`, `adapters.go`, and the test.
