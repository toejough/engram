"""Regression guard: harness.claude() must never pass the prompt as a positional CLI argument.

Confirmed live, $0 cost (Route A follow-up, 7.17): `claude -p "- Fix the typo..."` makes claude's
own CLI parser read a dash-leading prompt as an unknown OPTION, not a positional prompt --
`error: unknown option '- Fix the typo...'`, exit 1 in under 4s, no session ever created. Same
flag-misparse defect class as #787/#754/#749/#750/#755, just hitting this harness's own outer
`claude` invocation instead of an `engram` subcommand. Confirmed fix, also $0: pipe the prompt via
stdin (no positional prompt arg) -- verified it reaches real argument validation (a deliberately
invalid --model value produced "unrecognized_model", not a parse error).
"""
import json
import types

import harness


def _capture_run(monkeypatch):
    captured = {}

    def fake_run(args, cwd=None, env=None, **kwargs):
        captured["args"] = args
        captured["input"] = kwargs.get("input")
        return types.SimpleNamespace(stdout=json.dumps({"session_id": "s", "total_cost_usd": 0}))

    monkeypatch.setattr(harness.subprocess, "run", fake_run)
    return captured


def _paths(tmp_path):
    cfg = tmp_path / "cfg"
    ws = tmp_path / "ws"
    cfg.mkdir()
    ws.mkdir()
    return str(cfg), str(tmp_path / "vault"), str(ws), str(tmp_path / "ws.buildchunks")


def test_claude_pipes_a_dash_leading_prompt_via_stdin_not_positionally(monkeypatch, tmp_path):
    captured = _capture_run(monkeypatch)
    cfg, vault, ws, chunks = _paths(tmp_path)
    dash_prompt = "- Fix the typo in the login error message."

    harness.claude(cfg, "sonnet", vault, ws, dash_prompt, chunks=chunks)

    assert dash_prompt not in captured["args"], (
        "the prompt must never appear as a positional argv element -- a dash-leading value "
        "there is misread as an unknown CLI option by claude's own parser"
    )
    assert captured["input"] == dash_prompt, "the prompt must be piped via stdin instead"


def test_claude_resume_also_pipes_the_prompt_via_stdin(monkeypatch, tmp_path):
    """The resume branch rebuilds args from args[1:], so it shares the same bug if the base
    args still carry the prompt positionally -- covered separately since it's a distinct
    code path (`if resume_sid:`)."""
    captured = _capture_run(monkeypatch)
    cfg, vault, ws, chunks = _paths(tmp_path)
    dash_prompt = "- Fix the typo in the login error message."

    harness.claude(cfg, "sonnet", vault, ws, dash_prompt, resume_sid="abc123", chunks=chunks)

    assert dash_prompt not in captured["args"]
    assert captured["input"] == dash_prompt
    assert "--resume" in captured["args"] and "abc123" in captured["args"]


def test_claude_still_requests_json_output_and_the_right_model(monkeypatch, tmp_path):
    captured = _capture_run(monkeypatch)
    cfg, vault, ws, chunks = _paths(tmp_path)

    harness.claude(cfg, "sonnet", vault, ws, "ordinary prompt, no leading dash", chunks=chunks)

    args = captured["args"]
    assert "-p" in args
    assert "--output-format" in args and "json" in args
    assert "--model" in args and harness.MODELS["sonnet"] in args
    assert "--permission-mode" in args and "bypassPermissions" in args
