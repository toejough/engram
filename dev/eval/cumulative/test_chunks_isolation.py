"""Regression guard for the warm-arm chunk-index contamination bug.

The warm build subprocess must resolve engram's chunk index to the cell's ISOLATED dir, never the
operator's global $XDG_DATA_HOME/engram/chunks. When ENGRAM_CHUNKS_DIR is unset, recall's
`engram query` (and its Step 0.5 `engram ingest --auto`) fall through to that global index, which
holds the operator's live sessions = the eval answer key. Measured: warm app1 scored house 8/8 off
an "empty" vault by reading the operator's design session out of the global chunk index.
"""
import json
import types

import harness


def _capture_env(monkeypatch):
    captured = {}

    def fake_run(args, cwd=None, env=None, **_kw):
        captured["args"] = args
        captured["cwd"] = cwd
        captured["env"] = env
        return types.SimpleNamespace(stdout=json.dumps({"session_id": "s", "total_cost_usd": 0}))

    monkeypatch.setattr(harness.subprocess, "run", fake_run)
    return captured


def _paths(tmp_path):
    """Real dirs, because claude() now creates the state dirs it hands the trial."""
    cfg = tmp_path / "cfg"
    ws = tmp_path / "ws"
    cfg.mkdir()
    ws.mkdir()
    return str(cfg), str(tmp_path / "vault"), str(ws), str(tmp_path / "ws.buildchunks")


def test_claude_isolates_chunks_dir_env(monkeypatch, tmp_path):
    captured = _capture_env(monkeypatch)
    cfg, vault, ws, chunks = _paths(tmp_path)

    harness.claude(cfg, "sonnet", vault, ws, "build it", chunks=chunks)

    env = captured["env"]
    # The load-bearing guard: the isolated chunk index wins, so recall never reads the global default.
    assert env["ENGRAM_CHUNKS_DIR"] == chunks
    assert env["ENGRAM_VAULT_PATH"] == vault
    assert env["CLAUDE_CONFIG_DIR"] == cfg


def test_claude_omitting_chunks_still_isolates(monkeypatch, tmp_path):
    """Omitting chunks= must NOT fall through to the operator's global index.

    This test previously asserted the opposite — that ENGRAM_CHUNKS_DIR stays UNSET when the
    caller omits chunks=, documenting the #642 mechanism. That assertion encoded the leak as
    expected behavior: unset resolves to $XDG_DATA_HOME/engram/chunks, the operator's real index.
    Under #708 the default is a per-trial dir, so a no-chunks call is now safe by construction
    rather than safe only when every caller remembers the argument.
    """
    monkeypatch.delenv("ENGRAM_CHUNKS_DIR", raising=False)
    captured = _capture_env(monkeypatch)
    cfg, vault, ws, _ = _paths(tmp_path)

    harness.claude(cfg, "sonnet", vault, ws, "build it")

    chunks_dir = captured["env"]["ENGRAM_CHUNKS_DIR"]
    assert chunks_dir, "a no-chunks call must still get an isolated chunk index"
    assert not chunks_dir.startswith(harness.isolation.operator_data_dir()), (
        f"{chunks_dir} is inside the operator's engram data dir"
    )


def test_claude_cold_arm_gets_an_isolated_vault(monkeypatch, tmp_path):
    """vault="none" used to leave ENGRAM_VAULT_PATH unset — i.e. the operator's real vault.

    ENGRAM_BIN_DIR is prepended to PATH for every arm including cold, so engram was reachable from
    a cold cell the whole time; only the absence of engram guidance kept it from being used.
    """
    captured = _capture_env(monkeypatch)
    cfg, _, ws, _ = _paths(tmp_path)

    harness.claude(cfg, "sonnet", "none", ws, "build it")

    vault_dir = captured["env"]["ENGRAM_VAULT_PATH"]
    assert vault_dir
    assert not vault_dir.startswith(harness.isolation.operator_data_dir()), (
        f"cold arm resolved its vault to {vault_dir}, inside the operator's data dir"
    )


def test_claude_rejects_a_caller_supplied_real_vault(monkeypatch, tmp_path):
    """extra_env and the vault/chunks overrides run after isolated_env's own check, so the
    re-assert is what stops a real path from slipping through."""
    _capture_env(monkeypatch)
    cfg, _, ws, _ = _paths(tmp_path)
    real_vault = harness.isolation.operator_vault()

    try:
        harness.claude(cfg, "sonnet", real_vault, ws, "build it")
    except harness.isolation.IsolationError as exc:
        assert "ENGRAM_VAULT_PATH" in str(exc)
    else:
        raise AssertionError("claude() accepted the operator's real vault as the cell vault")
