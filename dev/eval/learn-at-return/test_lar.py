"""Offline tests for lar.py (tasks 1.1, 1.4): arm tree, settings, env allowlist, fixture wiring,
guards. No API calls, no keychain, no reads of the real vault or ~/.claude; writes only under tmp_path."""

import json
import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import lar  # noqa: E402
from config import (ALLOWED_TOOLS, ARM_TOKENS, DISALLOWED_TOOLS, FILE_MARKERS, GUIDANCE,  # noqa: E402
                    IMPORT_LINES, MARKER_PREFIX, PIN, PROBE_EXTRA_TOOL, SKILLS)

FAKE_TOKEN = "tok-FAKE-0000"
LEARN_PIN = ("intro\n\n**You do NOT need the full learn sweep mid-cycle.** This is a focused, single-note capture: crystallize\n"
             "the one confirmed correction now (the `/learn` skill's mid-cycle fast path — skip its Step 1 sweep, go\n"
             "straight to the crystallize step).\n\nFire at these cues:\n\n"
             "- **A review or the user rejected your approach** — x.\n"
             "- **You caught your own approach being wrong mid-task** — a self-discovered reversal is the same cue;\n"
             "  `/learn` the root cause at the moment you catch it.\n\n**Even a one-line fix earns the note.**\n")


class FakeSource:
    def __init__(self):
        self.files = {f"agent-instructions/guidance/{g}.md": f"{g} body\n" for g in GUIDANCE}
        self.files["agent-instructions/guidance/learn.md"] = LEARN_PIN
        for s in SKILLS:
            self.files[f"agent-instructions/skills/{s}/SKILL.md"] = f"---\nname: {s}\n---\n{s} skill\n"
        self.files["agent-instructions/skills/route/price-table.md"] = "prices\n"

    def pinned(self, path):
        return self.files[path]

    def pinned_tree(self, prefix):
        return {p[len(prefix):].lstrip("/"): v for p, v in self.files.items() if p.startswith(prefix + "/")}

    def worktree(self, path):
        return self.files[path] + "EDITED\n" if path.endswith("learn.md") else self.files[path]

    def head(self):
        return "deadbeef"


@pytest.fixture
def arm(tmp_path):
    fake_engram = tmp_path / "engram-bin"
    fake_engram.write_text("#!/bin/sh\n")
    fake_engram.chmod(0o755)
    root = tmp_path / "engram-arm.TEST"
    root.mkdir()
    spec = lar.ArmSpec(cell="P1", arm="GREEN", learn_source="provisional", domain="quillfeather")
    info = lar.build_arm(str(root), spec, FakeSource(), engram_bin=str(fake_engram),
                         deny_entries=["/private/tmp/claude-501/a", "/private/tmp/claude-501/b"],
                         home="/Users/tester")
    return root, spec, info


# ---------------------------------------------------------------------------
# arm tree
# ---------------------------------------------------------------------------


def test_arm_tree_has_every_d5_dir_and_an_empty_vault(arm):
    root, _, _ = arm
    for d in ("home", "work", "tmp", "xdg", "vault", "bin"):
        assert (root / d).is_dir()
    assert os.listdir(root / "vault") == []
    assert os.access(root / "bin" / "engram", os.X_OK)


def test_claude_md_imports_exactly_the_four_files_in_production_order(arm):
    root, _, _ = arm
    text = (root / "home/.claude/CLAUDE.md").read_text()
    assert [ln for ln in text.splitlines() if ln.strip()] == list(IMPORT_LINES)
    assert IMPORT_LINES == ("@~/.claude/engram/recall.md", "@~/.claude/engram/delegate.md",
                            "@~/.claude/engram/learn.md", "@~/.claude/engram/shim.md")


def test_each_guidance_file_carries_its_marker_and_only_learn_carries_the_arm_token(arm):
    root, _, _ = arm
    for g in GUIDANCE:
        text = (root / f"home/.claude/engram/{g}.md").read_text()
        assert MARKER_PREFIX + FILE_MARKERS[g] in text
        assert (MARKER_PREFIX + ARM_TOKENS["GREEN"] in text) == (g == "learn")
        assert ARM_TOKENS["RED"] not in text
        for other, tok in FILE_MARKERS.items():
            if other != g:
                assert tok not in text


def test_non_learn_guidance_is_the_pinned_text_plus_marker_only(arm):
    root, _, _ = arm
    src = FakeSource()
    for g in ("recall", "delegate", "shim"):
        text = (root / f"home/.claude/engram/{g}.md").read_text()
        assert text == src.pinned(f"agent-instructions/guidance/{g}.md") + "\n" + MARKER_PREFIX + FILE_MARKERS[g] + "\n"


def test_learn_source_selection(tmp_path):
    src = FakeSource()
    assert lar.learn_text("pin", src) == LEARN_PIN
    assert "EDITED" in lar.learn_text("worktree", src)
    prov = lar.learn_text("provisional", src)
    assert "crystallize\nthe one confirmed lesson now" in prov
    assert "the one confirmed correction now" not in prov
    assert "**A subagent just returned, and its report's `LESSONS:` line isn't `none`**" in prov
    # the new bullet is the fourth cue: after the third bullet, before the next paragraph
    assert prov.index("A subagent just returned") > prov.index("at the moment you catch it.")
    assert prov.index("A subagent just returned") < prov.index("**Even a one-line fix")


def test_provisional_green_refuses_text_it_cannot_edit():
    with pytest.raises(lar.HarnessError):
        lar.provisional_green("unrelated text\n")


def test_all_six_skills_installed_from_pin(arm):
    root, _, info = arm
    for s in SKILLS:
        assert (root / f"home/.claude/skills/{s}/SKILL.md").read_text().endswith(f"{s} skill\n")
    assert (root / "home/.claude/skills/route/price-table.md").exists()
    assert all(t["commit"] == PIN for t in info["texts"] if t["path"].startswith("skills/"))


def test_texts_manifest_records_commit_and_hash_for_every_installed_text(arm):
    _, _, info = arm
    paths = {t["path"] for t in info["texts"]}
    for g in GUIDANCE:
        assert f"engram/{g}.md" in paths
    learn = next(t for t in info["texts"] if t["path"] == "engram/learn.md")
    assert learn["source"] == "provisional"
    assert all(len(t["sha256"]) == 64 for t in info["texts"])


# ---------------------------------------------------------------------------
# fixture wiring (vault note 194: every load-bearing input reaches the arm)
# ---------------------------------------------------------------------------


def test_fixture_agent_carries_the_cells_reports_and_haiku_no_tools(arm):
    root, spec, info = arm
    text = (root / "home/.claude/agents/unit-worker.md").read_text()
    assert "name: unit-worker" in text
    assert "model: haiku" in text
    assert "disallowedTools:" in text
    fx = lar.load_fixtures()
    assert info["unit1_report"] in text
    assert info["unit1_report"].rstrip().endswith("LESSONS: " + fx["cells"]["P1"]["lessons"])
    assert fx["domains"]["quillfeather"]["unit2_report"] in text
    assert fx["domains"]["quillfeather"]["unit3_report"] in text


def test_prompt_is_the_frozen_domain_prompt_and_names_no_token_or_learning(arm):
    _, _, info = arm
    fx = lar.load_fixtures()
    with open(os.path.join(lar.HERE, "fixtures", fx["domains"]["quillfeather"]["prompt"])) as f:
        assert info["prompt"] == f.read()
    low = info["prompt"].lower()
    assert "learn" not in low and "lesson" not in low and "token" not in low
    assert "unit-worker" in info["prompt"]


@pytest.mark.parametrize("cell,index,domain", [
    ("P1", 0, "quillfeather"), ("P1", 1, "quillfeather"), ("P2", 0, "tarnbrook"),
    ("N1", 0, "quillfeather"), ("N1", 1, "tarnbrook"), ("N3", 2, "quillfeather"),
])
def test_domain_assignment(cell, index, domain):
    assert lar.domain_for(cell, index) == domain


def test_n_cell_report_is_neutral_body_plus_cell_line():
    fx = lar.load_fixtures()
    r = lar.unit1_report(fx, "N3", "tarnbrook")
    assert r.startswith(fx["domains"]["tarnbrook"]["unit1_neutral_body"])
    assert r.endswith("LESSONS: " + fx["cells"]["N3"]["lessons"])


def test_p_cell_in_wrong_domain_is_refused():
    with pytest.raises(lar.HarnessError):
        lar.unit1_report(lar.load_fixtures(), "P1", "tarnbrook")


# ---------------------------------------------------------------------------
# confinement: settings, env, argv
# ---------------------------------------------------------------------------


def test_settings_match_d5_exactly(arm):
    root, _, _ = arm
    s = json.loads((root / "home/.claude/settings.json").read_text())
    assert s == {
        "sandbox": {
            "enabled": True, "failIfUnavailable": True,
            "allowUnsandboxedCommands": False, "autoAllowBashIfSandboxed": False,
            "filesystem": {"allowWrite": [str(root)], "denyRead": ["/Users/tester"],
                           "denyWrite": ["/private/tmp/claude-501/a", "/private/tmp/claude-501/b"]},
            "network": {"allowedDomains": [], "strictAllowlist": True},
        },
        "permissions": {"deny": ["Read(//Users/tester/**)"]},
    }


def test_deny_entries_lists_each_entry_not_the_dir(tmp_path):
    d = tmp_path / "claude-501"
    d.mkdir()
    (d / "x").mkdir()
    (d / "y.txt").write_text("")
    got = lar.deny_entries(str(d))
    assert got == [str(d / "x"), str(d / "y.txt")]
    assert str(d) not in got


def test_env_is_exactly_the_d5_allowlist(tmp_path):
    argv = lar.arm_argv(str(tmp_path), FAKE_TOKEN, model="opus", user="tester", claude_bin="/x/claude")
    assert argv[:2] == ["/usr/bin/env", "-i"]
    ci = argv.index("/x/claude")
    pairs = dict(a.split("=", 1) for a in argv[2:ci])
    assert set(pairs) == {"HOME", "USER", "PATH", "TERM", "TMPDIR", "XDG_DATA_HOME", "ENGRAM_VAULT_PATH",
                          "CLAUDE_CODE_OAUTH_TOKEN"}
    assert pairs["HOME"] == f"{tmp_path}/home"
    assert pairs["PATH"] == f"{tmp_path}/bin:/usr/bin:/bin"
    assert pairs["TERM"] == "dumb"
    assert pairs["TMPDIR"] == f"{tmp_path}/tmp"
    assert pairs["XDG_DATA_HOME"] == f"{tmp_path}/xdg"
    assert pairs["ENGRAM_VAULT_PATH"] == f"{tmp_path}/vault"
    assert "bypassPermissions" not in " ".join(argv)


def test_tool_flags_match_d5(tmp_path):
    argv = lar.arm_argv(str(tmp_path), FAKE_TOKEN, model="opus", user="t", claude_bin="/x/claude")
    a, d = argv.index("--allowedTools"), argv.index("--disallowedTools")
    assert argv[a + 1:d] == list(ALLOWED_TOOLS)
    assert argv[d + 1:d + 1 + len(DISALLOWED_TOOLS)] == list(DISALLOWED_TOOLS)
    probe = lar.arm_argv(str(tmp_path), FAKE_TOKEN, model="haiku", user="t", claude_bin="/x/claude",
                         extra_tools=(PROBE_EXTRA_TOOL,))
    a, d = probe.index("--allowedTools"), probe.index("--disallowedTools")
    assert probe[a + 1:d] == list(ALLOWED_TOOLS) + [PROBE_EXTRA_TOOL]


def test_redacted_argv_never_holds_the_token(tmp_path):
    argv = lar.arm_argv(str(tmp_path), FAKE_TOKEN, model="opus", user="t", claude_bin="/x/claude")
    red = lar.redact(argv, FAKE_TOKEN)
    assert FAKE_TOKEN not in json.dumps(red)
    assert "CLAUDE_CODE_OAUTH_TOKEN=<redacted>" in red


def test_extract_token_parses_keychain_json_without_leaking_on_error():
    assert lar.extract_token(json.dumps({"claudeAiOauth": {"accessToken": "abc"}})) == "abc"
    with pytest.raises(lar.HarnessError) as e:
        lar.extract_token('{"claudeAiOauth": {"secret": "sk-ant-leak"}}')
    assert "sk-ant" not in str(e.value)


# ---------------------------------------------------------------------------
# guards (task 1.4)
# ---------------------------------------------------------------------------


def test_probe_verdict_requires_blocked_target_and_home_plus_working_control():
    assert lar.probe_verdict(target_exists=False, home_exists=False, control_exists=True)["ok"] is True
    assert lar.probe_verdict(target_exists=True, home_exists=False, control_exists=True)["ok"] is False
    assert lar.probe_verdict(target_exists=False, home_exists=True, control_exists=True)["ok"] is False
    v = lar.probe_verdict(target_exists=False, home_exists=False, control_exists=False)
    assert v["ok"] is False and "control" in v["reason"]


def test_probe_target_is_inside_an_existing_directory_entry(tmp_path):
    d = tmp_path / "claude-501"
    d.mkdir()
    (d / "a.txt").write_text("")
    (d / "sess").mkdir()
    t = lar.probe_target(lar.deny_entries(str(d)), "lar-probe-x")
    assert t == str(d / "sess" / "lar-probe-x")


def test_isolation_snapshot_flags_new_vault_files_outside_git(tmp_path):
    vault, cdir = tmp_path / "vault", tmp_path / "dotclaude"
    (vault / ".git").mkdir(parents=True)
    cdir.mkdir()
    (vault / "old.md").write_text("")
    os.utime(vault / "old.md", (1000, 1000))
    start = 2000.0
    (vault / ".git" / "index").write_text("")
    (vault / "new.md").write_text("")
    (cdir / "history.jsonl").write_text("")
    os.utime(cdir, (1000, 1000))
    snap = lar.isolation_snapshot(str(vault), str(cdir), start)
    assert snap["vault_newer"] == ["new.md"]
    assert snap["claude_top_newer"] == ["history.jsonl"]


def test_isolation_record_fails_batch_on_new_vault_entry(tmp_path):
    ok = lar.isolation_record(1.0, "/v", "/c", {"vault_newer": [], "claude_top_newer": []},
                              {"vault_newer": [], "claude_top_newer": ["projects"]})
    assert ok["ok"] is True
    bad = lar.isolation_record(1.0, "/v", "/c", {"vault_newer": [], "claude_top_newer": []},
                               {"vault_newer": ["1.x.md"], "claude_top_newer": []})
    assert bad["ok"] is False


def test_parse_arm_spec():
    s = lar.parse_arm_spec("P1:GREEN:provisional", 0)
    assert (s.cell, s.arm, s.learn_source, s.domain) == ("P1", "GREEN", "provisional", "quillfeather")
    s = lar.parse_arm_spec("N1:RED:pin:tarnbrook", 0)
    assert s.domain == "tarnbrook"
    with pytest.raises(lar.HarnessError):
        lar.parse_arm_spec("P1:BLUE:pin", 0)
