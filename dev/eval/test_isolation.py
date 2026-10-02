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


def test_engram_env_sets_both_paths_and_defaults_chunks_to_the_sibling(tmp_path):
    vault = str(tmp_path / "fixture-vault")

    env = isolation.engram_env(vault, base={})

    assert env["ENGRAM_VAULT_PATH"] == vault
    assert env["ENGRAM_CHUNKS_DIR"] == vault + ".chunks"
    assert os.path.isdir(env["ENGRAM_CHUNKS_DIR"])


def test_engram_env_accepts_an_explicit_chunks_dir(tmp_path):
    chunks = str(tmp_path / "elsewhere")

    env = isolation.engram_env(str(tmp_path / "v"), chunks=chunks, base={})

    assert env["ENGRAM_CHUNKS_DIR"] == chunks


def test_engram_env_rejects_the_operators_real_vault(tmp_path, monkeypatch):
    monkeypatch.setenv("XDG_DATA_HOME", str(tmp_path))
    real = os.path.join(str(tmp_path), "engram", "vault")

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.engram_env(real, base={})
    assert "ENGRAM_VAULT_PATH" in str(exc.value)


def test_assert_engram_isolated_ignores_claude_vars(tmp_path):
    """A direct `engram` call has no CLAUDE_CONFIG_DIR or transcript dir — requiring them would
    make the helper unusable for the fixture builders that only run the engram CLI."""
    isolation.assert_engram_isolated({
        "ENGRAM_VAULT_PATH": str(tmp_path / "v"),
        "ENGRAM_CHUNKS_DIR": str(tmp_path / "c"),
    })


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


# ----- install_skill: the shared warm-skill install primitive (#749 Gate-B DRY follow-up) -----
#
# wrun.py::build_warm_cfg, matrix.py::build_cfg_template, and probe.py::build_cfg_template all
# install skills through this one primitive now, instead of each hand-mirroring the
# source-exists / SKILL.md-exists / copy / destination-verify sequence.

def test_install_skill_raises_on_missing_source_dir(tmp_path):
    src_root = str(tmp_path / "skills")  # does not exist at all
    dst_skills = str(tmp_path / "cfg" / "skills")

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.install_skill(src_root, "recall", dst_skills)
    assert "recall" in str(exc.value)
    assert os.path.join(src_root, "recall") in str(exc.value)


def test_install_skill_raises_on_empty_source_dir_no_skill_md(tmp_path):
    src_root = str(tmp_path / "skills")
    os.makedirs(os.path.join(src_root, "recall"))  # exists, but no SKILL.md
    dst_skills = str(tmp_path / "cfg" / "skills")

    with pytest.raises(isolation.IsolationError) as exc:
        isolation.install_skill(src_root, "recall", dst_skills)
    assert "recall" in str(exc.value)
    assert "SKILL.md" in str(exc.value)


def test_install_skill_copies_and_verifies_destination_content(tmp_path):
    src_root = str(tmp_path / "skills")
    src_skill = os.path.join(src_root, "recall")
    os.makedirs(src_skill)
    (open(os.path.join(src_skill, "SKILL.md"), "w")).write("# Recall\n")
    dst_skills = str(tmp_path / "cfg" / "skills")
    os.makedirs(dst_skills)

    isolation.install_skill(src_root, "recall", dst_skills)

    dst_skill_md = os.path.join(dst_skills, "recall", "SKILL.md")
    assert os.path.exists(dst_skill_md)
    assert open(dst_skill_md).read() == "# Recall\n"
