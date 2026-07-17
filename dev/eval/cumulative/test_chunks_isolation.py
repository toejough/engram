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


def test_claude_isolates_chunks_dir_env(monkeypatch):
    captured = _capture_env(monkeypatch)
    harness.claude("/iso/cfg", "sonnet", "/iso/vault", "/iso/ws", "build it",
                   chunks="/iso/ws.buildchunks")
    env = captured["env"]
    # The load-bearing guard: the isolated chunk index wins, so recall never reads the global default.
    assert env["ENGRAM_CHUNKS_DIR"] == "/iso/ws.buildchunks"
    assert env["ENGRAM_VAULT_PATH"] == "/iso/vault"
    assert env["CLAUDE_CONFIG_DIR"] == "/iso/cfg"


def test_claude_omitting_chunks_leaves_env_unset(monkeypatch):
    # Documents the bug's mechanism: the OLD call site omitted chunks=, so ENGRAM_CHUNKS_DIR was never
    # set and engram resolved the global default index. A regression to a no-chunks call reopens the leak.
    monkeypatch.delenv("ENGRAM_CHUNKS_DIR", raising=False)
    captured = _capture_env(monkeypatch)
    harness.claude("/iso/cfg", "sonnet", "/iso/vault", "/iso/ws", "build it")
    assert "ENGRAM_CHUNKS_DIR" not in captured["env"]
