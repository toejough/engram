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


def test_assert_isolated_rejects_unset_config_dir(tmp_path):
    env = isolation.isolated_env(str(tmp_path / "cfg"), str(tmp_path / "t"), base={})
    del env["CLAUDE_CONFIG_DIR"]

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.assert_isolated(env)
    assert "CLAUDE_CONFIG_DIR" in str(exc.value)


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
