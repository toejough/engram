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

def test_deploy_skill_task_a_places_flat_commit_md(tmp_path):
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path)
    pp.deploy_skill(repo_path, "A")
    dst = os.path.join(repo_path, ".claude", "skills", "commit.md")
    assert os.path.isfile(dst)
    assert open(dst).read() == open(pp.TASKS["A"]["skill_src"]).read()


def test_deploy_skill_task_b_places_skill_dir(tmp_path):
    repo_path = str(tmp_path / "repo")
    os.makedirs(repo_path)
    pp.deploy_skill(repo_path, "B")
    dst = os.path.join(repo_path, ".claude", "skills", "gitignore-narrowing", "SKILL.md")
    assert os.path.isfile(dst)
    assert "name: gitignore-narrowing" in open(dst).read()


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


def test_evaluate_steps_repo_state_signal_uses_provided_checker():
    steps = [{"n": 1, "name": "custom repo check", "signal": "repo_state", "pattern": "always_true"}]

    def checker(pattern_name, repo_path):
        assert pattern_name == "always_true"
        assert repo_path == "/some/repo"
        return True

    results, k, all_ = pp.evaluate_steps(steps, [], repo_path="/some/repo", repo_checker=checker)
    assert results["1"] is True
    assert k == 1


def test_evaluate_steps_commit_message_format_checker_true_on_conforming_message(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m",
                     "feat(pkg): bump version\n\nwhy we did this\n\nAI-Used: [claude]"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format", repo) is True


def test_evaluate_steps_commit_message_format_checker_false_wrong_trailer(tmp_path):
    import subprocess
    repo = str(tmp_path / "repo")
    os.makedirs(repo)
    subprocess.run(["git", "init"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.email", "t@example.com"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "config", "user.name", "Trial"], cwd=repo, check=True, capture_output=True)
    subprocess.run(["git", "commit", "--allow-empty", "-m", "feat(pkg): bump version\n\nCo-Authored-By: x"],
                    cwd=repo, check=True, capture_output=True)
    assert pp.default_repo_checker("commit_message_format", repo) is False


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

def test_both_real_steps_json_files_are_valid_json_and_loadable():
    for task in ("A", "B"):
        steps = pp.load_steps(task)
        assert isinstance(steps, list)
        assert len(steps) > 0
        for step in steps:
            assert "n" in step and "signal" in step and "pattern" in step


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


def test_every_repo_state_pattern_in_real_steps_json_is_registered():
    for task in ("A", "B"):
        for step in pp.load_steps(task):
            if step["signal"] == "repo_state":
                assert step["pattern"] in pp.REPO_STATE_CHECKERS, (
                    f"Task {task} step {step['n']} names repo_state pattern "
                    f"{step['pattern']!r}, which has no registered checker"
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
    assert os.path.isfile(os.path.join(repo_path, ".claude", "skills", "commit.md"))
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
    end_state_given_found_n(0) != end_state_n(3), by construction."""
    agg = {
        "S": _agg(5, 5, 4, 4, 4),
        "R": _agg(5, 5, 3, 3, 3, end_state_given_found_n=0, followed_all_given_found_n=0),
        "F": _agg(5, 5, 4, 4, 4, end_state_given_found_n=2, followed_all_given_found_n=2),
        "Rdirect": _agg(5, 5, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    assert frame["shim_loss_total"] == 4 - 3
    assert frame["shim_loss_given_found"] == 4 - 0
    assert frame["shim_loss_total"] != frame["shim_loss_given_found"]


def test_decomposition_note_quality_f_reports_both_populations_when_they_diverge():
    agg = {
        "R": _agg(5, 5, 3, 3, 3, end_state_given_found_n=0, followed_all_given_found_n=0),
        "F": _agg(5, 5, 4, 4, 4, end_state_given_found_n=2, followed_all_given_found_n=2),
        "Rdirect": _agg(5, 5, None, 4, 4),
    }
    frame = pp.decomposition(agg)
    assert frame["note_quality_F_total"] == 4 - 4
    assert frame["note_quality_F_given_found"] == 2 - 4
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
