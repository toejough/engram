"""Unit tests for probe_phase2.py — the phase-2 conversion-parity eval harness.

Never calls claude. Covering-note removal / carrier-add tests use a synthetic vault dir (never the
operator's real vault) but read the real, committed encodings/ fixtures (safe — static repo files).
Transcript-scoring tests build synthetic event lists directly (mirroring phase-1's test_probe.py
style), matching the real Claude Code session-transcript shape probe.py's parse_transcript_events
already verified against a real transcript.
"""
import json
import os

import probe_phase2 as pp

HERE = os.path.dirname(os.path.abspath(__file__))


# ----- synthetic transcript helpers (mirrors phase-1's test_probe.py) -----

def _tool_use(name, input_, idx, tool_id="tu"):
    return {"idx": idx, "kind": "tool_use", "name": name, "input": input_, "id": f"{tool_id}{idx}"}


def _tool_result(idx, tool_use_idx, content, tool_id="tu"):
    return {"idx": idx, "kind": "tool_result", "tool_use_id": f"{tool_id}{tool_use_idx}", "content": content}


# ----- covering-note removal + carrier add (synthetic vault dir, never the real one) -----

def _make_synthetic_vault(tmp_path, extra_notes=()):
    vault = tmp_path / "vault"
    vault.mkdir()
    for base in pp.TASK_A_REMOVAL:
        (vault / f"{base}.md").write_text(f"removal note {base}")
        (vault / f"{base}.vec.json").write_text("{}")
    for base in pp.TASK_B_REMOVAL:
        (vault / f"{base}.md").write_text(f"removal note {base}")
        (vault / f"{base}.vec.json").write_text("{}")
    (vault / f"{pp.NOTE_830_BASENAME}.md").write_text("note 830 body")
    (vault / f"{pp.NOTE_830_BASENAME}.vec.json").write_text("{}")
    for base, content in extra_notes:
        (vault / f"{base}.md").write_text(content)
        (vault / f"{base}.vec.json").write_text("{}")
    return str(vault)


def test_remove_covering_notes_task_a_deletes_only_task_a_list(tmp_path):
    keep_base = "999.2026-01-01.unrelated-note"
    vault = _make_synthetic_vault(tmp_path, extra_notes=[(keep_base, "unrelated content")])
    pp.remove_covering_notes(vault, "A", "S")
    for base in pp.TASK_A_REMOVAL:
        assert not os.path.exists(os.path.join(vault, f"{base}.md"))
        assert not os.path.exists(os.path.join(vault, f"{base}.vec.json"))
    # Task B's list and note 830 are untouched by a Task A removal
    for base in pp.TASK_B_REMOVAL:
        assert os.path.exists(os.path.join(vault, f"{base}.md"))
    assert os.path.exists(os.path.join(vault, f"{pp.NOTE_830_BASENAME}.md"))
    # unrelated note survives
    assert os.path.exists(os.path.join(vault, f"{keep_base}.md"))


def test_remove_covering_notes_task_b_removes_830_except_for_arm_r(tmp_path):
    vault_non_r = _make_synthetic_vault(tmp_path)
    pp.remove_covering_notes(vault_non_r, "B", "F")
    for base in pp.TASK_B_REMOVAL:
        assert not os.path.exists(os.path.join(vault_non_r, f"{base}.md"))
    assert not os.path.exists(os.path.join(vault_non_r, f"{pp.NOTE_830_BASENAME}.md"))
    assert not os.path.exists(os.path.join(vault_non_r, f"{pp.NOTE_830_BASENAME}.vec.json"))


def test_remove_covering_notes_task_b_keeps_830_for_arm_r(tmp_path):
    vault_r = _make_synthetic_vault(tmp_path)
    pp.remove_covering_notes(vault_r, "B", "R")
    for base in pp.TASK_B_REMOVAL:
        assert not os.path.exists(os.path.join(vault_r, f"{base}.md"))
    # 830 survives for arm R — it IS the carrier
    assert os.path.exists(os.path.join(vault_r, f"{pp.NOTE_830_BASENAME}.md"))
    assert os.path.exists(os.path.join(vault_r, f"{pp.NOTE_830_BASENAME}.vec.json"))


def test_add_carrier_task_a_r_copies_encoding_note_and_returns_basename(tmp_path):
    vault = _make_synthetic_vault(tmp_path)
    basename = pp.add_carrier(vault, "A", "R")
    assert basename == "1.2026-09-11.commit-conventional-message"
    assert os.path.exists(os.path.join(vault, basename + ".md"))
    assert os.path.exists(os.path.join(vault, basename + ".vec.json"))


def test_add_carrier_task_a_f_copies_encoding_note_and_returns_basename(tmp_path):
    vault = _make_synthetic_vault(tmp_path)
    basename = pp.add_carrier(vault, "A", "F")
    assert basename == "1.2026-09-11.git-commit-skill-procedure"
    assert os.path.exists(os.path.join(vault, basename + ".md"))


def test_add_carrier_task_b_r_does_not_copy_but_returns_830_basename(tmp_path):
    vault = _make_synthetic_vault(tmp_path)
    before_listing = sorted(os.listdir(vault))
    basename = pp.add_carrier(vault, "B", "R")
    assert basename == pp.NOTE_830_BASENAME
    # nothing was copied in — the listing is unchanged
    assert sorted(os.listdir(vault)) == before_listing


def test_add_carrier_s_and_rdirect_add_nothing(tmp_path):
    vault = _make_synthetic_vault(tmp_path)
    before_listing = sorted(os.listdir(vault))
    assert pp.add_carrier(vault, "A", "S") is None
    assert pp.add_carrier(vault, "A", "Rdirect") is None
    assert sorted(os.listdir(vault)) == before_listing


# ----- eval-session note exclusion (luhmann >= EXCLUDE_LUHMANN_MIN), every arm and task -----

def _write_note(vault, basename, content="body"):
    with open(os.path.join(vault, basename + ".md"), "w") as f:
        f.write(content)
    with open(os.path.join(vault, basename + ".vec.json"), "w") as f:
        f.write("{}")


def test_remove_eval_session_notes_keeps_below_floor_removes_at_or_above_floor(tmp_path):
    vault = str(tmp_path / "vault")
    os.makedirs(vault)
    _write_note(vault, "100.2026-01-01.old-fact")
    _write_note(vault, "954.2026-09-08.just-below-floor")
    _write_note(vault, "955.2026-09-10.at-floor")
    _write_note(vault, "1200.2026-09-11.well-above-floor")
    _write_note(vault, "qa.2026-09-11.some-question")

    removed = pp.remove_eval_session_notes(vault, min_luhmann=955)

    assert removed == 2
    remaining_md = sorted(n for n in os.listdir(vault) if n.endswith(".md"))
    assert remaining_md == [
        "100.2026-01-01.old-fact.md",
        "954.2026-09-08.just-below-floor.md",
        "qa.2026-09-11.some-question.md",
    ]
    # sidecars removed alongside their notes
    assert not os.path.exists(os.path.join(vault, "955.2026-09-10.at-floor.vec.json"))
    assert not os.path.exists(os.path.join(vault, "1200.2026-09-11.well-above-floor.vec.json"))
    assert os.path.exists(os.path.join(vault, "954.2026-09-08.just-below-floor.vec.json"))
    assert os.path.exists(os.path.join(vault, "qa.2026-09-11.some-question.vec.json"))


def test_remove_eval_session_notes_respects_custom_floor(tmp_path):
    vault = str(tmp_path / "vault")
    os.makedirs(vault)
    _write_note(vault, "100.2026-01-01.old-fact")
    _write_note(vault, "200.2026-01-02.also-old")
    removed = pp.remove_eval_session_notes(vault, min_luhmann=150)
    assert removed == 1
    assert os.path.exists(os.path.join(vault, "100.2026-01-01.old-fact.md"))
    assert not os.path.exists(os.path.join(vault, "200.2026-01-02.also-old.md"))


def test_leading_luhmann_number_parses_integer_prefix_and_ignores_qa():
    assert pp._leading_luhmann_number("955.2026-09-10.at-floor") == 955
    assert pp._leading_luhmann_number("1.2026-09-11.commit-conventional-message") == 1
    assert pp._leading_luhmann_number("qa.2026-09-11.some-question") is None


def test_setup_trial_vault_applies_eval_session_note_exclusion_for_every_arm_and_task(tmp_path, monkeypatch):
    """The exclusion must run inside setup_trial_vault (every arm, every task) — not just be a
    standalone function nobody calls. verify_vault_health is stubbed out: this test's fake
    REAL_VAULT has placeholder (non-real) .vec.json sidecars, so it is not meant to pass a real
    `engram embed status` check — that plumbing is covered separately by the dry-run against the
    real vault (see task-3-report.md)."""
    fake_real_vault = str(tmp_path / "real_vault")
    os.makedirs(fake_real_vault)
    _write_note(fake_real_vault, "100.2026-01-01.old-fact")
    _write_note(fake_real_vault, "960.2026-09-11.this-session-note")
    monkeypatch.setattr(pp, "REAL_VAULT", fake_real_vault)
    monkeypatch.setattr(pp, "verify_vault_health", lambda vault: {})

    for task, arm in (("A", "S"), ("A", "R"), ("B", "F"), ("B", "Rdirect")):
        trial_vault = str(tmp_path / f"trial_{task}_{arm}")
        env = {"ENGRAM_VAULT_PATH": trial_vault}
        pp.setup_trial_vault(env, task, arm)
        remaining_md = [n for n in os.listdir(trial_vault) if n.endswith(".md")]
        assert "960.2026-09-11.this-session-note.md" not in remaining_md, (task, arm)


# ----- Rdirect CLAUDE.md: procedure section + marker -----

def test_build_claude_md_phase2_no_extra_matches_phase1_shape():
    marker = "RUNBOOK-VS-SKILL-PROBE2-abc12345"
    claude_md = pp.build_claude_md_phase2(marker, extra=None)
    assert f"PROBE-TOKEN: {marker}" in claude_md
    assert "## Project procedure" not in claude_md


def test_build_claude_md_phase2_rdirect_appends_procedure_section_verbatim():
    marker = "RUNBOOK-VS-SKILL-PROBE2-def67890"
    extra_text = "1. Do the thing.\n2. Verify the thing."
    claude_md = pp.build_claude_md_phase2(marker, extra=extra_text)
    assert f"PROBE-TOKEN: {marker}" in claude_md
    assert "## Project procedure" in claude_md
    assert extra_text in claude_md
    # the procedure section comes after the marker line (appended, not interleaved)
    assert claude_md.index(f"PROBE-TOKEN: {marker}") < claude_md.index("## Project procedure")


def test_rdirect_procedure_text_task_a_reads_a_r_note_body_verbatim():
    text = pp.rdirect_procedure_text("A")
    assert "Check VCS type" in text
    assert "AI-Used: [claude]" in text
    # frontmatter fields are stripped
    assert "situation:" not in text


def test_rdirect_procedure_text_task_b_reads_real_830_note_body_verbatim():
    text = pp.rdirect_procedure_text("B")
    assert "git check-ignore" in text
    assert "situation:" not in text


# ----- S skill deployment path per task -----

def test_deploy_skill_task_a_places_directory_form_skill(tmp_path):
    """Round-3 fix: Claude Code's skill listing did not discover the flat .claude/skills/commit.md
    file the way it discovered the directory-form gitignore-narrowing skill (smoke run 1
    finding), so Task A's skill is deployed as .claude/skills/commit/SKILL.md instead. Body must
    still be byte-identical to the live repo skill file (the frontmatter already carries `name:
    commit`, so no injection is needed here — see the dedicated frontmatter-injection test)."""
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path)
    pp.deploy_skill(repo_path, "A")
    dst = os.path.join(repo_path, ".claude", "skills", "commit", "SKILL.md")
    assert os.path.isfile(dst)
    assert open(dst).read() == open(pp.TASKS["A"]["skill_src"]).read()
    # the old flat-file path must NOT exist — this harness deploys the directory form only
    assert not os.path.exists(os.path.join(repo_path, ".claude", "skills", "commit.md"))


def test_skill_body_with_name_is_noop_when_name_already_present():
    text = "---\nname: commit\ndescription: x\n---\n\n# Commit\n\nbody text\n"
    assert pp._skill_body_with_name(text, "commit") == text


def test_skill_body_with_name_injects_when_absent():
    text = "---\ndescription: x\n---\n\n# Commit\n\nbody text\n"
    result = pp._skill_body_with_name(text, "commit")
    assert "name: commit" in result
    # body content after the frontmatter is unchanged
    assert result.endswith("\n\n# Commit\n\nbody text\n")
    # only one name key was added, not a duplicate
    assert result.count("name:") == 1


def test_skill_body_with_name_leaves_non_frontmatter_text_unchanged():
    text = "no frontmatter here, just body text\n"
    assert pp._skill_body_with_name(text, "commit") == text


def test_deploy_skill_task_b_places_skill_dir(tmp_path):
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path)
    pp.deploy_skill(repo_path, "B")
    dst = os.path.join(repo_path, ".claude", "skills", "gitignore-narrowing", "SKILL.md")
    assert os.path.isfile(dst)
    assert "name: gitignore-narrowing" in open(dst).read()


# ----- mutating-step regex (smoke-run-1 finding: bare '>' matched '2>&1') -----

def _mut_ev(command, idx=0):
    return {"idx": idx, "kind": "tool_use", "name": "Bash", "input": {"command": command}, "id": "tu"}


def test_mutating_step_stderr_redirect_pipe_is_not_mutating():
    """engram ingest --auto 2>&1 | tail -5 — the recall skill's own diagnostic call — must NOT be
    treated as a mutation. Smoke run 1: this false-positive fired on turn 1 of every trial,
    masking FOUND for R/F even when the carrier was genuinely in the query result."""
    ev = _mut_ev("engram ingest --auto 2>&1 | tail -5")
    assert pp.is_first_mutating_step(ev, "A") is False


def test_mutating_step_echo_redirect_to_file_is_mutating():
    ev = _mut_ev("echo x > file")
    assert pp.is_first_mutating_step(ev, "A") is True


def test_mutating_step_heredoc_redirect_to_gitignore_is_mutating():
    ev = _mut_ev("cat <<EOF > .gitignore")
    assert pp.is_first_mutating_step(ev, "B") is True


def test_mutating_step_git_commit_heredoc_is_mutating():
    ev = _mut_ev("git commit -m \"$(cat <<'EOF'")
    assert pp.is_first_mutating_step(ev, "A") is True


def test_mutating_step_plain_read_only_bash_not_mutating():
    for command in ("git status", "git diff --staged", "git log --oneline -5", "ls -la",
                     "cat pkg/version.go", "2>/dev/null"):
        ev = _mut_ev(command)
        assert pp.is_first_mutating_step(ev, "A") is False, command


def test_mutating_step_append_redirect_after_fd_dup_is_still_mutating():
    """A real append AFTER an fd-duplication (e.g. `2>&1 >> out.log`) must still count — only the
    fd-dup/duplication redirect itself is excluded, not every redirect in the command."""
    ev = _mut_ev("some_cmd 2>&1 >> out.log")
    assert pp.is_first_mutating_step(ev, "A") is True


def test_mutating_step_engram_query_phrase_containing_git_commit_words_not_mutating():
    """Smoke-run-1 finding #2: the recall skill's own retrieval phrasing can legitimately contain
    the words 'git commit' inside a quoted --phrase argument. This exact A-R smoke transcript
    command must not be treated as a mutation."""
    ev = _mut_ev(
        'engram query --lazy-chunks \\\n'
        '  --phrase "commit a version bump following project conventions" \\\n'
        '  --phrase "git commit message conventions for this repo" \\\n'
        '  --phrase "staging files for a version release commit"'
    )
    assert pp.is_first_mutating_step(ev, "A") is False


def test_mutating_step_git_commit_heredoc_still_mutating_despite_quote_stripping():
    """The real invocation verb ('git commit') sits BEFORE the first quote character, so quote-
    stripping must not blind the detector to a genuine commit."""
    ev = _mut_ev("git commit -m \"$(cat <<'EOF'")
    assert pp.is_first_mutating_step(ev, "A") is True


def test_mutating_step_git_add_inside_echoed_quotes_not_mutating():
    ev = _mut_ev('echo "git add"')
    assert pp.is_first_mutating_step(ev, "A") is False


def test_mutating_step_engram_readonly_verbs_never_mutating_even_with_redirect_looking_text():
    for command in (
        'engram query --lazy-chunks --phrase "some > thing"',
        "engram show-chunk source#anchor",
        "engram activate --note 1.2026-01-01.some-note.md",
        "engram ingest --auto",
    ):
        ev = _mut_ev(command)
        assert pp.is_first_mutating_step(ev, "A") is False, command


# ----- FOUND: Arm S (skill: commit) -----

def test_found_s_true_when_commit_skill_before_first_mutation():
    events = [
        _tool_use("Skill", {"skill": "commit", "args": "commit the version bump"}, idx=0),
        _tool_use("Bash", {"command": "git status"}, idx=1),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    found, idx = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is True
    assert idx == 0


def test_found_s_false_when_commit_skill_after_first_mutation():
    events = [
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0),
        _tool_use("Skill", {"skill": "commit", "args": "late invoke"}, idx=1),
    ]
    found, idx = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is False
    assert idx is None


def test_found_s_false_when_wrong_skill_name():
    events = [_tool_use("Skill", {"skill": "recall", "args": "x"}, idx=0)]
    found, idx = pp.score_found_phase2("A", "S", events, carrier_basename=None)
    assert found is False


def test_found_s_task_b_uses_gitignore_narrowing_skill_name():
    events = [
        _tool_use("Skill", {"skill": "gitignore-narrowing", "args": "narrow it"}, idx=0),
        _tool_use("Bash", {"command": "git add .gitignore"}, idx=1),
    ]
    found, idx = pp.score_found_phase2("B", "S", events, carrier_basename=None)
    assert found is True


# ----- FOUND: Arm R/F requires carrier basename in the query result -----

def test_found_r_true_when_query_result_contains_carrier_basename():
    carrier = "1.2026-09-11.commit-conventional-message"
    events = [
        _tool_use("Bash", {"command": "engram query --lazy-chunks --phrase \"commit conventions\""}, idx=0),
        _tool_result(1, 0, f"items:\n  - path: {carrier}.md\n    kind: runbook"),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    found, idx = pp.score_found_phase2("A", "R", events, carrier_basename=carrier)
    assert found is True
    assert idx == 0


def test_found_r_false_when_query_result_lacks_carrier_basename():
    carrier = "1.2026-09-11.commit-conventional-message"
    events = [
        _tool_use("Bash", {"command": "engram query --lazy-chunks --phrase \"commit conventions\""}, idx=0),
        _tool_result(1, 0, "items: []\n"),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    found, idx = pp.score_found_phase2("A", "R", events, carrier_basename=carrier)
    assert found is False
    assert idx is None


def test_found_f_true_when_query_result_contains_carrier_basename():
    carrier = "1.2026-09-11.git-commit-skill-procedure"
    events = [
        _tool_use("Bash", {"command": "engram query --phrase x"}, idx=0),
        _tool_result(1, 0, f"{carrier}.md"),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=2),
    ]
    found, idx = pp.score_found_phase2("A", "F", events, carrier_basename=carrier)
    assert found is True


def test_found_rdirect_is_always_na():
    events = [_tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0)]
    found, idx = pp.score_found_phase2("A", "Rdirect", events, carrier_basename=None)
    assert found is None
    assert idx is None


# ----- steps.json evaluation: not_pattern and after -----

def test_evaluate_steps_not_pattern_rejects_git_add_dash_a():
    steps = [
        {"n": 1, "name": "stage explicit paths, not -A", "signal": "bash_regex",
         "pattern": r"git\s+add", "not_pattern": r"-A|git\s+add\s+\.$"},
    ]
    events = [
        _tool_use("Bash", {"command": "git add -A"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert k == 1
    assert all_ is True


def test_evaluate_steps_not_pattern_all_matches_disqualified_stays_false():
    steps = [
        {"n": 1, "name": "stage explicit paths, not -A", "signal": "bash_regex",
         "pattern": r"git\s+add", "not_pattern": r"-A"},
    ]
    events = [_tool_use("Bash", {"command": "git add -A"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False
    assert k == 0
    assert all_ is False


def test_evaluate_steps_after_requires_match_strictly_after_referenced_step():
    steps = [
        {"n": 1, "name": "check state", "signal": "bash_regex", "pattern": r"git\s+status"},
        {"n": 2, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add", "after": 1},
    ]
    # git add occurs BEFORE git status — step 2 must not count despite matching the pattern
    events = [
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=0),
        _tool_use("Bash", {"command": "git status"}, idx=1),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is False
    assert k == 1


def test_evaluate_steps_after_satisfied_when_match_follows_referenced_step():
    steps = [
        {"n": 1, "name": "check state", "signal": "bash_regex", "pattern": r"git\s+status"},
        {"n": 2, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add", "after": 1},
    ]
    events = [
        _tool_use("Bash", {"command": "git status"}, idx=0),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=1),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is True
    assert k == 2
    assert all_ is True


def test_evaluate_steps_after_same_event_later_match_position_is_satisfied():
    """Final-review finding (coordinator correction): a single compound Bash command can
    legitimately satisfy an earlier step and a later step at once (e.g.
    `git add X && git diff --cached --name-only`). Requiring a STRICTLY LATER EVENT index for
    `after` wrongly fails the later step here — the fix accepts a match in the SAME Bash event
    when this step's own match starts strictly after the referenced step's match position within
    that command string."""
    steps = [
        {"n": 1, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add"},
        {"n": 2, "name": "verify", "signal": "bash_regex", "pattern": r"git\s+diff\s+--cached",
         "after": 1},
    ]
    events = [
        _tool_use("Bash", {"command": "git add file.txt && git diff --cached --name-only"}, idx=0),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is True
    assert k == 2
    assert all_ is True


def test_evaluate_steps_after_same_event_earlier_match_position_not_satisfied():
    """Mirror case: when the 'verify' pattern's match position comes BEFORE the referenced step's
    match position in the SAME command, the after constraint must NOT be satisfied — same
    command, wrong order."""
    steps = [
        {"n": 1, "name": "stage", "signal": "bash_regex", "pattern": r"git\s+add"},
        {"n": 2, "name": "verify", "signal": "bash_regex", "pattern": r"git\s+diff\s+--cached",
         "after": 1},
    ]
    events = [
        _tool_use("Bash", {"command": "git diff --cached --name-only && git add file.txt"}, idx=0),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert results["2"] is False
    assert k == 1


def test_evaluate_steps_repo_state_signal_uses_provided_checker():
    steps = [{"n": 1, "name": "custom repo check", "signal": "repo_state", "pattern": "always_true"}]

    def checker(pattern_name, repo_path):
        assert pattern_name == "always_true"
        assert repo_path == "/some/repo"
        return True

    results, k, all_ = pp.evaluate_steps(steps, [], repo_path="/some/repo", repo_checker=checker)
    assert results["1"] is True
    assert k == 1


# ----- tool_path signal + any_of (round 5: native Read/Edit/Write credit) -----

def test_evaluate_steps_tool_path_credits_native_read_on_matching_path():
    steps = [{"n": 1, "name": "inspect via native tool", "signal": "tool_path",
              "tools": ["Read", "Edit", "Write"], "pattern": r"\.gitignore$"}]
    events = [_tool_use("Read", {"file_path": "/repo/.gitignore"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True
    assert k == 1


def test_evaluate_steps_tool_path_false_when_wrong_tool():
    steps = [{"n": 1, "name": "inspect via native tool", "signal": "tool_path",
              "tools": ["Read", "Edit", "Write"], "pattern": r"\.gitignore$"}]
    events = [_tool_use("Bash", {"command": "cat .gitignore"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_evaluate_steps_tool_path_false_when_path_does_not_match():
    steps = [{"n": 1, "name": "inspect via native tool", "signal": "tool_path",
              "tools": ["Read", "Edit", "Write"], "pattern": r"\.gitignore$"}]
    events = [_tool_use("Read", {"file_path": "/repo/README.md"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_evaluate_steps_tool_path_matches_edit_and_write_too():
    steps = [{"n": 1, "name": "inspect via native tool", "signal": "tool_path",
              "tools": ["Read", "Edit", "Write"], "pattern": r"\.gitignore$"}]
    for tool_name in ("Edit", "Write"):
        events = [_tool_use(tool_name, {"file_path": "/repo/.gitignore"}, idx=0)]
        results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
        assert results["1"] is True, tool_name


def test_evaluate_steps_any_of_satisfied_by_either_alternative():
    steps = [{
        "n": 1, "name": "inspect .gitignore",
        "any_of": [
            {"signal": "bash_regex", "pattern": r"cat\s+\.gitignore"},
            {"signal": "tool_path", "tools": ["Read"], "pattern": r"\.gitignore$"},
        ],
    }]
    # only the tool_path alternative is satisfied
    events = [_tool_use("Read", {"file_path": "/repo/.gitignore"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True

    # only the bash_regex alternative is satisfied
    events2 = [_tool_use("Bash", {"command": "cat .gitignore"}, idx=0)]
    results2, k2, all2_ = pp.evaluate_steps(steps, events2, repo_path="/does/not/matter")
    assert results2["1"] is True


def test_evaluate_steps_any_of_false_when_neither_alternative_matches():
    steps = [{
        "n": 1, "name": "inspect .gitignore",
        "any_of": [
            {"signal": "bash_regex", "pattern": r"cat\s+\.gitignore"},
            {"signal": "tool_path", "tools": ["Read"], "pattern": r"\.gitignore$"},
        ],
    }]
    events = [_tool_use("Bash", {"command": "git status"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is False


def test_real_gitignore_steps_json_step_1_credits_synthetic_native_read():
    """Round-5 finding: hand-verifying B-R showed an agent that inspects .gitignore via the
    native Read tool (not `cat .gitignore` in Bash) got no credit for step 1. The real fixture's
    step 1 is now any_of(bash_regex, tool_path) — a synthetic Read-only transcript must credit
    it."""
    steps = pp.load_steps("B")
    step1 = next(s for s in steps if s["n"] == 1)
    assert "any_of" in step1
    events = [_tool_use("Read", {"file_path": "/repo/.gitignore"}, idx=0)]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/does/not/matter")
    assert results["1"] is True


def test_commit_message_format_and_body_true_on_conforming_message(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m",
                     "feat(pkg): bump version\n\nwhy we did this\n\nAI-Used: [claude]"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format_and_body", repo) is True


def test_commit_message_format_and_body_true_even_with_only_co_authored_trailer(tmp_path):
    """Round-4 ruling: the checker no longer scores the trailer at all — a body line's CONTENT
    is never inspected, only that one exists. A message whose only 'body' line is a
    Co-Authored-By trailer (the harness's own injected convention, not AI-Used) must still PASS
    this checker; trailer classification is reported separately via classify_trailer."""
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", "feat(pkg): bump version\n\nCo-Authored-By: x"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format_and_body", repo) is True


def test_commit_message_format_and_body_false_when_no_body_at_all(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", "feat(pkg): bump version"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format_and_body", repo) is False


def test_commit_message_format_and_body_false_when_subject_form_missing(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", "bumped the version\n\nsome body text"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format_and_body", repo) is False


# ----- classify_trailer: reported, not scored (round 4) -----

def _commit_with_message(repo, message):
    import subprocess
    os.makedirs(repo, exist_ok=True)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", message], cwd=repo, check=True, capture_output=True)


def test_classify_trailer_ai_used(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(repo, "feat(pkg): bump version\n\nwhy\n\nAI-Used: [claude]")
    assert pp.classify_trailer(repo) == "ai_used"


def test_classify_trailer_co_authored(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(repo, "feat(pkg): bump version\n\nwhy\n\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>")
    assert pp.classify_trailer(repo) == "co_authored"


def test_classify_trailer_both(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(
        repo,
        "feat(pkg): bump version\n\nwhy\n\nAI-Used: [claude]\nCo-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>",
    )
    assert pp.classify_trailer(repo) == "both"


def test_classify_trailer_none(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(repo, "feat(pkg): bump version\n\nwhy we made this change")
    assert pp.classify_trailer(repo) == "none"


# ----- gitignore_narrowed_to_generated checker (Task B step 3) -----

def _write_gitignore(repo, lines):
    os.makedirs(repo, exist_ok=True)
    with open(os.path.join(repo, ".gitignore"), "w") as f:
        f.write("\n".join(lines) + "\n")


def test_gitignore_narrowed_to_generated_true_when_narrowed_correctly(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["testdata/generated/", "*.o"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is True


def test_gitignore_narrowed_to_generated_true_with_leading_globstar_form(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["**/testdata/generated", "*.o"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is True


def test_gitignore_narrowed_to_generated_false_when_bare_testdata_still_present(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["testdata/", "testdata/generated/"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is False


def test_gitignore_narrowed_to_generated_false_when_bare_scripts_still_present(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["testdata/generated/", "scripts/"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is False


def test_gitignore_narrowed_to_generated_false_when_narrowed_target_missing(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["*.o"])
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is False


def test_gitignore_narrowed_to_generated_false_when_gitignore_missing(tmp_path):
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    assert pp.default_repo_checker("gitignore_narrowed_to_generated", repo) is False


# ----- real steps.json files: valid JSON, evaluable, every repo_state pattern registered -----

def _step_signals(step):
    """A step's list of signal definitions — its own top-level signal, or every entry in
    `any_of`."""
    return step["any_of"] if "any_of" in step else [step]


def test_both_real_steps_json_files_are_valid_json_and_loadable():
    for task in ("A", "B"):
        steps = pp.load_steps(task)
        assert isinstance(steps, list)
        assert len(steps) > 0
        for step in steps:
            assert "n" in step
            for sig in _step_signals(step):
                assert "signal" in sig and "pattern" in sig


def test_real_steps_json_task_a_evaluates_against_a_synthetic_transcript_and_repo(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m",
                     "feat(pkg): bump version\n\nwhy\n\nAI-Used: [claude]"],
                    cwd=repo, check=True, capture_output=True)
    events = [
        _tool_use("Bash", {"command": "ls -la .jj 2>/dev/null || true"}, idx=0),
        _tool_use("Bash", {"command": "git status"}, idx=1),
        _tool_use("Bash", {"command": "git diff --staged"}, idx=2),
        _tool_use("Bash", {"command": "git log --oneline -5"}, idx=3),
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=4),
        _tool_use("Bash", {"command": "git commit -m 'feat: bump'"}, idx=5),
        _tool_use("Bash", {"command": "git log -1"}, idx=6),
    ]
    steps = pp.load_steps("A")
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path=repo)
    assert k == len(steps)
    assert all_ is True


def test_real_steps_json_task_b_evaluates_against_a_synthetic_transcript_and_repo(tmp_path):
    repo = str(tmp_path / "repo")
    _write_gitignore(repo, ["testdata/generated/", "*.o"])
    events = [
        _tool_use("Bash", {"command": "cat .gitignore"}, idx=0),
        _tool_use("Bash", {"command": "git check-ignore -q testdata/generated/big.bin"}, idx=1),
        _tool_use("Bash", {"command": "git status --porcelain"}, idx=2),
        _tool_use("Bash", {"command": "git add scripts/build.sh"}, idx=3),
        _tool_use("Bash", {"command": "git diff --cached"}, idx=4),
    ]
    steps = pp.load_steps("B")
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path=repo)
    assert k == len(steps)
    assert all_ is True


def test_real_commit_steps_json_step_7_verify_requires_order_after_commit():
    """Final-review finding: Task A step 7 ('git log -1 or git status after commit') shares its
    pattern with step 2's 'git status' check. Without an 'after' constraint, an EARLY git status
    call (satisfying step 2, run before any real work) would ALSO satisfy step 7, even though no
    verification ever happened after the commit. The real fixture must carry after=6 (the commit
    step) so this can't happen — this proves the ordering constraint changes the verdict, not just
    that the field is present."""
    steps = pp.load_steps("A")
    step7 = next(s for s in steps if s["n"] == 7)
    assert step7.get("after") == 6

    def always_true_checker(_pattern, _repo):
        return True

    events_no_post_commit_verify = [
        _tool_use("Bash", {"command": "ls -la .jj 2>/dev/null || true"}, idx=0),  # step 1
        _tool_use("Bash", {"command": "git status"}, idx=1),                     # step 2
        _tool_use("Bash", {"command": "git log --oneline -5"}, idx=2),           # step 3
        _tool_use("Bash", {"command": "git add pkg/version.go"}, idx=3),         # step 4
        _tool_use("Bash", {"command": "git commit -m 'feat: bump'"}, idx=4),     # step 6
    ]
    results, k, all_ = pp.evaluate_steps(steps, events_no_post_commit_verify, repo_path="/x",
                                          repo_checker=always_true_checker)
    assert results["2"] is True  # the early git status still credits step 2
    assert results["7"] is False  # but must NOT also credit step 7 — no call happened after commit
    assert all_ is False

    events_with_post_commit_verify = events_no_post_commit_verify + [
        _tool_use("Bash", {"command": "git log -1"}, idx=5),
    ]
    results2, k2, all2_ = pp.evaluate_steps(steps, events_with_post_commit_verify, repo_path="/x",
                                             repo_checker=always_true_checker)
    assert results2["7"] is True
    assert all2_ is True


def test_real_gitignore_steps_json_step_6_verify_requires_order_after_staging():
    """Final-review finding: Task B step 6 ('git diff --cached or git status') shares its
    pattern with step 4's 'git status' check. Without an 'after' constraint, an EARLY git status
    call (satisfying step 4, run before staging) would ALSO satisfy step 6, even though nothing
    was ever verified after staging. The real fixture must carry after=5 (the staging step)."""
    steps = pp.load_steps("B")
    step6 = next(s for s in steps if s["n"] == 6)
    assert step6.get("after") == 5

    def always_true_checker(_pattern, _repo):
        return True

    events_no_post_stage_verify = [
        _tool_use("Bash", {"command": "cat .gitignore"}, idx=0),                                   # step 1
        _tool_use("Bash", {"command": "git check-ignore -q testdata/generated/big.bin"}, idx=1),   # step 2
        _tool_use("Bash", {"command": "git status --porcelain"}, idx=2),                           # step 4
        _tool_use("Bash", {"command": "git add scripts/build.sh"}, idx=3),                         # step 5
    ]
    results, k, all_ = pp.evaluate_steps(steps, events_no_post_stage_verify, repo_path="/x",
                                          repo_checker=always_true_checker)
    assert results["4"] is True  # the early git status still credits step 4
    assert results["6"] is False  # but must NOT also credit step 6 — nothing verified after staging
    assert all_ is False

    events_with_post_stage_verify = events_no_post_stage_verify + [
        _tool_use("Bash", {"command": "git diff --cached"}, idx=4),
    ]
    results2, k2, all2_ = pp.evaluate_steps(steps, events_with_post_stage_verify, repo_path="/x",
                                             repo_checker=always_true_checker)
    assert results2["6"] is True
    assert all2_ is True


def test_real_gitignore_steps_json_step_6_verify_satisfied_by_same_compound_staging_command():
    """Regression test for the reported real trial (hand-verified B-R, idx 16): a single compound
    Bash command stages AND verifies in one call —
    `git add .gitignore scripts/build.sh testdata/fixture.json && git diff --cached --name-only
    && git status --short`. Step 5 (staging) and step 6 (verify, after=5) must BOTH be credited
    from this one event — the after-constraint fix must not regress this legitimate case back to
    false."""
    steps = pp.load_steps("B")

    def always_true_checker(_pattern, _repo):
        return True

    events = [
        _tool_use("Bash", {"command": "cat .gitignore"}, idx=0),
        _tool_use("Bash", {"command": "git check-ignore -q testdata/generated/big.bin"}, idx=1),
        _tool_use("Bash", {"command": "git status --porcelain"}, idx=2),
        _tool_use("Bash", {
            "command": ("git add .gitignore scripts/build.sh testdata/fixture.json "
                        "&& git diff --cached --name-only && git status --short"),
        }, idx=3),
    ]
    results, k, all_ = pp.evaluate_steps(steps, events, repo_path="/x",
                                          repo_checker=always_true_checker)
    assert results["5"] is True
    assert results["6"] is True
    assert all_ is True


def test_every_repo_state_pattern_in_real_steps_json_is_registered():
    for task in ("A", "B"):
        for step in pp.load_steps(task):
            for sig in _step_signals(step):
                if sig["signal"] == "repo_state":
                    assert sig["pattern"] in pp.REPO_STATE_CHECKERS, (
                        f"Task {task} step {step['n']} names repo_state pattern "
                        f"{sig['pattern']!r}, which has no registered checker"
                    )


# ----- non-fatal scoring: one trial's scoring exception must not lose sibling results -----

def test_score_trial_scoring_exception_is_captured_not_raised(monkeypatch, tmp_path):
    def _boom(*args, **kwargs):
        raise ValueError("synthetic scoring failure")

    monkeypatch.setattr(pp, "score_found_phase2", _boom)
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    scored = pp._score_trial("A", "R", events=[], repo_path=repo, carrier_basename="x")
    assert scored["scoring_error"] is not None
    assert "synthetic scoring failure" in scored["scoring_error"]
    # safe defaults, not a raised exception
    assert scored["found"] is None
    assert scored["end_state"] is False
    assert scored["followed_k"] == 0


def test_score_trial_no_exception_leaves_scoring_error_none():
    scored = pp._score_trial("A", "S", events=[], repo_path="/does/not/exist", carrier_basename=None)
    assert scored["scoring_error"] is None


def test_score_trial_populates_trailer_for_task_a(tmp_path):
    repo = str(tmp_path / "repo")
    _commit_with_message(repo, "feat(pkg): bump version\n\nwhy\n\nAI-Used: [claude]")
    scored = pp._score_trial("A", "S", events=[], repo_path=repo, carrier_basename=None)
    assert scored["trailer"] == "ai_used"


def test_score_trial_trailer_is_na_for_task_b():
    scored = pp._score_trial("B", "S", events=[], repo_path="/does/not/exist", carrier_basename=None)
    assert scored["trailer"] == "n/a"


# ----- live scoring vs --rescore: same field set (round 6) -----

def test_live_scoring_and_rescore_emit_the_same_scored_field_set(tmp_path):
    """The live scoring path (_score_trial, feeding run_one_trial_phase2's persisted record) and
    --rescore must emit the SAME set of scored fields — round-6 finding: first_procedure_step_index
    was computed by --rescore but never by the live path, so base result files lacked it.
    `rescored_from` is the one expected rescore-only addition (provenance marking that a record
    was rescored) — everything else must match exactly."""
    repo_path = pp.setup_trial_repo(str(tmp_path / "trial"), "A", "R",
                                     marker="RUNBOOK-VS-SKILL-PROBE2-parity")
    carrier = "1.2026-09-11.commit-conventional-message"

    transcript_path = tmp_path / "session.jsonl"
    lines = [
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:00.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu1", "name": "Bash",
                                      "input": {"command": "engram query --phrase x"}}]}
        }),
        json.dumps({
            "type": "user", "timestamp": "2026-09-11T00:00:01.000Z",
            "message": {"content": [{"type": "tool_result", "tool_use_id": "tu1",
                                      "content": f"{carrier}.md"}]}
        }),
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:02.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu2", "name": "Bash",
                                      "input": {"command": "git add pkg/version.go"}}]}
        }),
    ]
    with open(transcript_path, "w") as f:
        f.write("\n".join(lines) + "\n")

    # --- live path: _score_trial's scored keys (minus scoring_error, a live-only diagnostic
    # never persisted as a scored field name in the record schema itself) ---
    events = pp.p1.parse_transcript_events([str(transcript_path)])
    scored = pp._score_trial("A", "R", events, repo_path, carrier)
    live_scored_keys = set(scored.keys()) - {"scoring_error"}

    # --- rescore path: whatever keys rescore_file adds/overwrites onto a minimal provenance-only
    # record ---
    minimal_record = {"task": "A", "arm": "R", "repo_path": repo_path,
                       "transcript_path": str(transcript_path), "carrier_basename": carrier}
    results_path = tmp_path / "minimal.jsonl"
    with open(results_path, "w") as f:
        f.write(json.dumps(minimal_record) + "\n")
    rescored_path = tmp_path / "minimal.rescored.jsonl"
    pp.rescore_file(str(results_path), str(rescored_path))
    rescored_record = pp.load_jsonl(str(rescored_path))[0]
    rescore_added_keys = set(rescored_record.keys()) - set(minimal_record.keys())

    assert rescore_added_keys - {"rescored_from"} == live_scored_keys


def test_score_trial_load_steps_exception_is_captured_not_raised(monkeypatch, tmp_path):
    """Round-2 review: load_steps() itself must be inside the try — a missing/invalid steps.json
    must not break the 'never raises' contract either. n_steps stays at its 0 default since it
    is set only after a successful load_steps() call."""
    def _boom(task_key):
        raise FileNotFoundError("synthetic missing steps.json")

    monkeypatch.setattr(pp, "load_steps", _boom)
    repo = str(tmp_path / "repo")
    os.makedirs(repo, exist_ok=True)
    scored = pp._score_trial("A", "R", events=[], repo_path=repo, carrier_basename="x")
    assert scored["scoring_error"] is not None
    assert "synthetic missing steps.json" in scored["scoring_error"]
    assert scored["n_steps"] == 0
    assert scored["found"] is None
    assert scored["end_state"] is False


# ----- starting-state assertion (real fixture init scripts, no claude calls) -----

def test_setup_trial_repo_task_a_preserves_unstaged_and_decoy(tmp_path):
    repo_path = pp.setup_trial_repo(str(tmp_path), "A", "R", marker="RUNBOOK-VS-SKILL-PROBE2-x")
    import subprocess
    status = subprocess.run(["git", "-C", repo_path, "status", "--porcelain"],
                             capture_output=True, text=True, check=True).stdout
    assert " M pkg/version.go" in status
    assert "?? notes/" in status


def test_setup_trial_repo_task_b_preserves_decoy_and_ignored_dirs(tmp_path):
    """Task B's fixture (init_fixture_repo.sh, fixed at commit b098afa1) commits only .gitignore
    + src/main.go; scripts/build.sh and testdata/fixture.json start ignored AND untracked (not
    in the index), and tmp.log is created as an untracked decoy after the fixture commit."""
    repo_path = pp.setup_trial_repo(str(tmp_path), "B", "F", marker="RUNBOOK-VS-SKILL-PROBE2-y")
    import subprocess
    status = subprocess.run(["git", "-C", repo_path, "status", "--porcelain"],
                             capture_output=True, text=True, check=True).stdout
    assert "?? tmp.log" in status

    for rel_path in ("testdata/generated/big.bin", "scripts/build.sh", "testdata/fixture.json"):
        ignored = subprocess.run(["git", "-C", repo_path, "check-ignore", "-q", rel_path])
        assert ignored.returncode == 0, f"{rel_path} should be ignored"

    ls_files = subprocess.run(["git", "-C", repo_path, "ls-files"],
                               capture_output=True, text=True, check=True).stdout
    assert "scripts/build.sh" not in ls_files
    assert "testdata/fixture.json" not in ls_files


def test_setup_trial_repo_task_a_arm_s_deploys_skill_and_commits_it(tmp_path):
    repo_path = pp.setup_trial_repo(str(tmp_path), "A", "S", marker="RUNBOOK-VS-SKILL-PROBE2-z")
    assert os.path.isfile(os.path.join(repo_path, ".claude", "skills", "commit", "SKILL.md"))
    import subprocess
    log = subprocess.run(["git", "-C", repo_path, "log", "--oneline"], capture_output=True, text=True).stdout
    assert "add project config" in log


# ----- decision frame outputs: baseline_uninterpretable and shim loss -----

def _agg(n, valid_n, found_n, end_state_n, followed_all_n, end_state_given_found_n=None,
         followed_all_given_found_n=None, n_steps=6):
    return {
        "n": n, "valid_n": valid_n, "found_n": found_n, "found_given_n": found_n,
        "end_state_n": end_state_n,
        "end_state_given_found_n": end_state_given_found_n if end_state_given_found_n is not None else end_state_n,
        "followed_all_n": followed_all_n,
        "followed_all_given_found_n": (followed_all_given_found_n if followed_all_given_found_n is not None
                                        else followed_all_n),
        "recall_fired_n": 0, "followed_mean_k": float(followed_all_n), "n_steps": n_steps,
        "cost_mean": 0.10, "duration_mean": 30.0,
    }


def test_decomposition_baseline_uninterpretable_true_when_s_end_state_below_3():
    agg = {"S": _agg(5, 5, 2, 2, 2), "R": _agg(5, 5, 4, 4, 4), "F": _agg(5, 5, 3, 3, 3),
           "Rdirect": _agg(5, 5, None, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["baseline_uninterpretable"] is True


def test_decomposition_baseline_uninterpretable_false_when_s_end_state_at_least_3():
    agg = {"S": _agg(5, 5, 3, 3, 3), "R": _agg(5, 5, 4, 4, 4), "F": _agg(5, 5, 3, 3, 3),
           "Rdirect": _agg(5, 5, None, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["baseline_uninterpretable"] is False


def test_decomposition_shim_loss_reports_both_populations_when_they_diverge():
    """Controller ruling (final): shim_loss must report BOTH the total population (all valid R
    trials, incl. not-found) and the given_found population (only R's found=true subset) — these
    must be observably different, not aliased. R's found_n=3 < valid_n=5, and
    end_state_given_found_n(0) != end_state_n(3), by construction. Final-review correction:
    shim_loss is a percentage-point RATE difference (dict with both fractions), never a raw-count
    subtraction — see test_decomposition_shim_loss_unequal_n_equal_rates_is_zero_pp for why."""
    agg = {
        "S": _agg(5, 5, 4, 4, 4),
        "R": _agg(5, 5, 3, 3, 3, end_state_given_found_n=0, followed_all_given_found_n=0),
        "F": _agg(5, 5, 4, 4, 4, end_state_given_found_n=2, followed_all_given_found_n=2),
        "Rdirect": _agg(5, 5, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    # Rdirect 4/5 (80%) vs R 3/5 (60%) -> +20.0pp
    assert frame["shim_loss_total"] == {"pp_diff": 20.0, "rdirect": "4/5", "r": "3/5"}
    # Rdirect 4/5 (80%) vs R's found=true subset 0/3 (0%) -> +80.0pp
    assert frame["shim_loss_given_found"] == {"pp_diff": 80.0, "rdirect": "4/5", "r_given_found": "0/3"}
    assert frame["shim_loss_total"] != frame["shim_loss_given_found"]


def test_decomposition_shim_loss_unequal_n_equal_rates_is_zero_pp():
    """Final-review finding: Task B's real numbers have Rdirect at valid_n=4 (one trial
    invalidated by a mid-session rate limit) at 4/4 (100%) vs R at valid_n=5 at 5/5 (100%) — raw
    counts differ (4 - 5 = -1) but the RATES are identical. The reported figure must be 0
    percentage points, not a negative count implying Rdirect lost ground."""
    agg = {
        "R": _agg(5, 5, 5, 5, 5, end_state_given_found_n=5, followed_all_given_found_n=5),
        "F": _agg(5, 5, 5, 5, 5, end_state_given_found_n=5, followed_all_given_found_n=5),
        "Rdirect": _agg(4, 4, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    assert frame["shim_loss_total"] == {"pp_diff": 0.0, "rdirect": "4/4", "r": "5/5"}


def test_decomposition_note_quality_f_reports_both_populations_when_they_diverge():
    agg = {
        "R": _agg(5, 5, 3, 3, 3, end_state_given_found_n=0, followed_all_given_found_n=0),
        "F": _agg(5, 5, 4, 4, 4, end_state_given_found_n=2, followed_all_given_found_n=2),
        "Rdirect": _agg(5, 5, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    # F 4/5 (80%) vs Rdirect 4/5 (80%) -> 0.0pp
    assert frame["note_quality_F_total"] == {"pp_diff": 0.0, "f": "4/5", "rdirect": "4/5"}
    # F's found=true subset 2/4 (50%) vs Rdirect 4/5 (80%) -> -30.0pp
    assert frame["note_quality_F_given_found"] == {
        "pp_diff": -30.0, "f_given_found": "2/4", "rdirect": "4/5"}
    assert frame["note_quality_F_total"] != frame["note_quality_F_given_found"]


def test_decomposition_type_effect_reports_both_populations_when_they_diverge():
    agg = {
        "R": _agg(5, 5, 3, 3, 3, end_state_given_found_n=0, followed_all_given_found_n=0),
        "F": _agg(5, 5, 4, 4, 4, end_state_given_found_n=2, followed_all_given_found_n=2),
        "Rdirect": _agg(5, 5, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    assert frame["type_effect"]["end_state_total"] == 3 - 4
    assert frame["type_effect"]["end_state_given_found"] == 0 - 2
    assert frame["type_effect"]["followed_all_total"] == 3 - 4
    assert frame["type_effect"]["followed_all_given_found"] == 0 - 2
    assert frame["type_effect"]["end_state_total"] != frame["type_effect"]["end_state_given_found"]
    assert (frame["type_effect"]["followed_all_total"]
            != frame["type_effect"]["followed_all_given_found"])


def test_decomposition_parity_cant_distinguish_within_one_trial():
    agg = {"S": _agg(5, 5, 4, 4, 4), "R": _agg(5, 5, 3, 3, 3), "F": _agg(5, 5, 3, 3, 3),
           "Rdirect": _agg(5, 5, None, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["parity"]["S_vs_R"]["end_state"] == "cant_distinguish"
    assert frame["parity"]["S_vs_F"]["end_state"] == "cant_distinguish"


def test_decomposition_parity_worse_and_better_at_two_or_more():
    agg = {"S": _agg(5, 5, 4, 4, 4), "R": _agg(5, 5, 1, 1, 1), "F": _agg(5, 5, 1, 1, 1),
           "Rdirect": _agg(5, 5, None, 3, 3)}
    frame = pp.decomposition(agg)
    assert frame["parity"]["S_vs_R"]["end_state"] == "worse"
    agg2 = {"S": _agg(5, 5, 1, 1, 1), "R": _agg(5, 5, 5, 5, 5), "F": _agg(5, 5, 1, 1, 1),
            "Rdirect": _agg(5, 5, None, 3, 3)}
    frame2 = pp.decomposition(agg2)
    assert frame2["parity"]["S_vs_R"]["end_state"] == "better"


def test_decomposition_note_quality_given_delivery_format():
    agg = {"R": _agg(5, 5, 4, 3, 3, end_state_given_found_n=3, followed_all_given_found_n=2)}
    frame = pp.decomposition(agg)
    assert frame["note_quality_given_delivery_R"]["end_state"] == "3 of 4"
    assert frame["note_quality_given_delivery_R"]["followed_all"] == "2 of 4"


# ----- aggregate() -----

def test_aggregate_filters_by_task_and_arm_and_validity():
    records = [
        {"task": "A", "arm": "S", "valid": True, "found": True, "end_state": True, "followed_all": True,
         "followed_k": 7, "n_steps": 7, "recall_fired": False, "total_cost_usd": 0.5, "duration_ms": 1000},
        {"task": "A", "arm": "S", "valid": False, "found": True, "end_state": True, "followed_all": True,
         "followed_k": 7, "n_steps": 7, "recall_fired": False, "total_cost_usd": 0.5, "duration_ms": 1000},
        {"task": "B", "arm": "S", "valid": True, "found": True, "end_state": True, "followed_all": True,
         "followed_k": 6, "n_steps": 6, "recall_fired": False, "total_cost_usd": 0.3, "duration_ms": 500},
        {"task": "A", "arm": "R", "valid": True, "found": True, "end_state": True, "followed_all": True,
         "followed_k": 7, "n_steps": 7, "recall_fired": True, "total_cost_usd": 0.4, "duration_ms": 900},
    ]
    agg = pp.aggregate(records, "A", "S")
    assert agg["n"] == 2
    assert agg["valid_n"] == 1
    assert agg["found_n"] == 1
    assert agg["end_state_n"] == 1


def test_aggregate_rdirect_found_n_is_zero_when_found_is_none():
    records = [
        {"task": "A", "arm": "Rdirect", "valid": True, "found": None, "end_state": True,
         "followed_all": True, "followed_k": 7, "n_steps": 7, "recall_fired": False,
         "total_cost_usd": 0.2, "duration_ms": 800},
    ]
    agg = pp.aggregate(records, "A", "Rdirect")
    assert agg["found_n"] == 0
    assert agg["end_state_n"] == 1


# ----- --rescore also recomputes FOUND (round 3) -----

def test_rescore_recomputes_found_from_kept_transcript(tmp_path):
    """Simulates the smoke-run-1 scenario: an original record was scored found=False by the
    pre-fix mutating regex (a bare '>' falsely matched '2>&1' in the recall skill's own
    diagnostic call, making that turn 0 count as 'the first mutation'), even though the carrier
    basename genuinely appeared in the engram query result before any real mutation. Rescoring
    with the fixed detector must flip found to True."""
    repo = str(tmp_path / "repo")
    os.makedirs(repo)

    transcript_path = tmp_path / "session.jsonl"
    lines = [
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:00.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu1", "name": "Bash",
                                      "input": {"command": "engram ingest --auto 2>&1 | tail -5"}}]}
        }),
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:01.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu2", "name": "Bash",
                                      "input": {"command": "engram query --phrase x"}}]}
        }),
        json.dumps({
            "type": "user", "timestamp": "2026-09-11T00:00:02.000Z",
            "message": {"content": [{"type": "tool_result", "tool_use_id": "tu2",
                                      "content": "1.2026-09-11.commit-conventional-message.md"}]}
        }),
        json.dumps({
            "type": "assistant", "timestamp": "2026-09-11T00:00:03.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu3", "name": "Bash",
                                      "input": {"command": "git add pkg/version.go"}}]}
        }),
    ]
    with open(transcript_path, "w") as f:
        f.write("\n".join(lines) + "\n")

    record = {
        "task": "A", "arm": "R", "repo_path": repo, "transcript_path": str(transcript_path),
        "carrier_basename": "1.2026-09-11.commit-conventional-message",
        "found": False, "found_index": None,  # the pre-fix (buggy) scoring
        "followed_steps": {}, "followed_k": 0, "followed_all": False,
        "end_state": False, "end_state_output": "",
    }
    results_path = tmp_path / "results.jsonl"
    with open(results_path, "w") as f:
        f.write(json.dumps(record) + "\n")

    rescored_path = tmp_path / "rescored.jsonl"
    pp.rescore_file(str(results_path), str(rescored_path))

    rescored = pp.load_jsonl(str(rescored_path))[0]
    assert rescored["found"] is True
    assert rescored["found_index"] == 1
    assert rescored["found_method"] == "engram query"
    assert rescored["first_procedure_step_index"] == 3  # the git add, not the 2>&1 diagnostic call
    assert rescored["rescored_from"] == str(results_path)


def test_rescore_preserves_provenance_fields_from_kept_record():
    """arm/carrier_basename/task provenance must be READ from the kept record, never re-derived —
    a missing repo_path is the only thing that should short-circuit to an error."""
    record = {"task": "A", "arm": "S", "repo_path": "/does/not/exist", "transcript_path": None,
              "carrier_basename": None}
    import tempfile
    with tempfile.TemporaryDirectory() as td:
        results_path = os.path.join(td, "results.jsonl")
        with open(results_path, "w") as f:
            f.write(json.dumps(record) + "\n")
        rescored_path = os.path.join(td, "rescored.jsonl")
        pp.rescore_file(results_path, rescored_path)
        rescored = pp.load_jsonl(rescored_path)[0]
    assert rescored["error"] == "rescore: repo missing"
    assert rescored["arm"] == "S"
