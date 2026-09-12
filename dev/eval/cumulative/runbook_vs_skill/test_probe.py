"""Unit tests for probe.py's transcript scoring, repo-state scoring, and decision-frame logic.

Never calls claude. Transcript fixtures are hand-built synthetic JSONL lines matching the real
Claude Code session-transcript shape (verified against a real ~/.claude/projects/*.jsonl in
this session: assistant lines carry message.content[] blocks with type "tool_use"/name/input/id;
user lines carry message.content[] blocks with type "tool_result"/tool_use_id/content; both carry
a "timestamp" field). END-STATE tests run the REAL fixtures/done_when_checks.sh against a
hand-completed clone of fixture-repo-template (built via the real init_fixture_repo.sh), not a
stub — per the harness's own note-505 "no keyword-on-final-text scoring" discipline, done_when is
the mechanical ground truth.
"""
import json
import os
import subprocess

import probe as p

HERE = os.path.dirname(os.path.abspath(__file__))


# ----- synthetic transcript helpers -----

def _tool_use_line(name, input_, tool_id="tu1", ts="2026-09-09T00:00:00.000Z"):
    return json.dumps({
        "type": "assistant", "timestamp": ts,
        "message": {"content": [{"type": "tool_use", "id": tool_id, "name": name, "input": input_}]},
    })


def _tool_result_line(tool_id, content, ts="2026-09-09T00:00:01.000Z"):
    return json.dumps({
        "type": "user", "timestamp": ts,
        "message": {"content": [{"type": "tool_result", "tool_use_id": tool_id, "content": content}]},
    })


def _write_transcript(tmp_path, lines):
    path = tmp_path / "session.jsonl"
    path.write_text("\n".join(lines) + "\n")
    return str(path)


# ----- marker gate -----

def test_marker_seen_true_when_marker_substring_present(tmp_path):
    marker = "RUNBOOK-VS-SKILL-PROBE-abc12345"
    text = f"some assistant text\nPROBE-TOKEN: {marker}\nmore text"
    assert p.is_marker_seen(text, marker) is True


def test_marker_seen_false_when_marker_absent(tmp_path):
    marker = "RUNBOOK-VS-SKILL-PROBE-abc12345"
    text = "assistant text with no token at all"
    assert p.is_marker_seen(text, marker) is False


# ----- FOUND: Arm S -----

def test_found_arm_s_true_when_skill_before_procedure_step(tmp_path):
    lines = [
        _tool_use_line("Skill", {"skill": "sensors-add-skill", "args": "registering pressure_v2"},
                        tool_id="tu1", ts="2026-09-09T00:00:00.000Z"),
        _tool_use_line("Edit", {"file_path": "/repo/lib/sensors/registry.txt", "old_string": "a", "new_string": "b"},
                        tool_id="tu2", ts="2026-09-09T00:00:05.000Z"),
    ]
    path = _write_transcript(tmp_path, lines)
    events = p.parse_transcript_events([path])
    found, idx = p.score_found("S", events, note_basename=None)
    assert found is True
    assert idx == 0


def test_found_arm_s_false_when_skill_after_procedure_step(tmp_path):
    lines = [
        _tool_use_line("Edit", {"file_path": "/repo/lib/sensors/registry.txt", "old_string": "a", "new_string": "b"},
                        tool_id="tu1", ts="2026-09-09T00:00:00.000Z"),
        _tool_use_line("Skill", {"skill": "sensors-add-skill", "args": "late invoke"},
                        tool_id="tu2", ts="2026-09-09T00:00:05.000Z"),
    ]
    path = _write_transcript(tmp_path, lines)
    events = p.parse_transcript_events([path])
    found, idx = p.score_found("S", events, note_basename=None)
    assert found is False
    assert idx is None


# ----- FOUND: Arm R/F -----

def test_found_arm_r_true_when_query_result_contains_basename(tmp_path):
    lines = [
        _tool_use_line("Bash", {"command": "engram query --lazy-chunks --phrase \"sensor registration\""},
                        tool_id="tu1", ts="2026-09-09T00:00:00.000Z"),
        _tool_result_line("tu1", "items:\n  - path: 1.2026-09-09.sensor-registration-procedure.md\n    kind: runbook",
                           ts="2026-09-09T00:00:01.000Z"),
        _tool_use_line("Edit", {"file_path": "/repo/lib/sensors/registry.txt", "old_string": "a", "new_string": "b"},
                        tool_id="tu2", ts="2026-09-09T00:00:05.000Z"),
    ]
    path = _write_transcript(tmp_path, lines)
    events = p.parse_transcript_events([path])
    found, idx = p.score_found("R", events, note_basename="1.2026-09-09.sensor-registration-procedure")
    assert found is True
    assert idx == 0


def test_found_arm_r_false_when_query_result_lacks_basename(tmp_path):
    lines = [
        _tool_use_line("Bash", {"command": "engram query --lazy-chunks --phrase \"sensor registration\""},
                        tool_id="tu1", ts="2026-09-09T00:00:00.000Z"),
        _tool_result_line("tu1", "items: []\n", ts="2026-09-09T00:00:01.000Z"),
        _tool_use_line("Edit", {"file_path": "/repo/lib/sensors/registry.txt", "old_string": "a", "new_string": "b"},
                        tool_id="tu2", ts="2026-09-09T00:00:05.000Z"),
    ]
    path = _write_transcript(tmp_path, lines)
    events = p.parse_transcript_events([path])
    found, idx = p.score_found("R", events, note_basename="1.2026-09-09.sensor-registration-procedure")
    assert found is False
    assert idx is None


def test_found_arm_r_false_when_query_after_procedure_step(tmp_path):
    lines = [
        _tool_use_line("Edit", {"file_path": "/repo/lib/sensors/registry.txt", "old_string": "a", "new_string": "b"},
                        tool_id="tu1", ts="2026-09-09T00:00:00.000Z"),
        _tool_use_line("Bash", {"command": "engram query --phrase x"},
                        tool_id="tu2", ts="2026-09-09T00:00:05.000Z"),
        _tool_result_line("tu2", "1.2026-09-09.sensor-registration-procedure.md", ts="2026-09-09T00:00:06.000Z"),
    ]
    path = _write_transcript(tmp_path, lines)
    events = p.parse_transcript_events([path])
    found, idx = p.score_found("R", events, note_basename="1.2026-09-09.sensor-registration-procedure")
    assert found is False


# ----- recall_fired -----

def test_recall_fired_true_when_recall_skill_invoked(tmp_path):
    lines = [_tool_use_line("Skill", {"skill": "recall", "args": "x"}, tool_id="tu1")]
    path = _write_transcript(tmp_path, lines)
    events = p.parse_transcript_events([path])
    assert p.score_recall_fired(events) is True


def test_recall_fired_false_when_absent(tmp_path):
    lines = [_tool_use_line("Bash", {"command": "ls"}, tool_id="tu1")]
    path = _write_transcript(tmp_path, lines)
    events = p.parse_transcript_events([path])
    assert p.score_recall_fired(events) is False


# ----- per-worker cfg pool (round-1 review: no shared CLAUDE_CONFIG_DIR across workers) -----

def test_build_cfg_pool_creates_distinct_dirs_from_one_template(tmp_path):
    dirs = p.build_cfg_pool(str(tmp_path), 3)
    assert len(dirs) == 3
    assert len(set(dirs)) == 3  # three DISTINCT paths, not the same dir repeated
    for d in dirs:
        assert os.path.isdir(os.path.join(d, "skills", "recall"))
        assert os.path.isdir(os.path.join(d, "skills", "learn"))
        assert os.path.exists(os.path.join(d, ".claude.json"))
    # independent copies, not the same underlying directory or a symlink to the template
    template = os.path.join(str(tmp_path), "cfg_template")
    for d in dirs:
        assert os.path.realpath(d) != os.path.realpath(template)


# ----- procedure-step detection: mutation-only (round-1 review) -----

def test_procedure_step_ls_read_only_not_a_step():
    ev = {"idx": 0, "kind": "tool_use", "name": "Bash", "input": {"command": "ls migrations/"}, "id": "tu1"}
    assert p.is_procedure_step(ev) is False


def test_procedure_step_cat_without_redirect_not_a_step():
    ev = {"idx": 0, "kind": "tool_use", "name": "Bash",
          "input": {"command": "cat lib/sensors/registry.txt"}, "id": "tu1"}
    assert p.is_procedure_step(ev) is False


def test_procedure_step_echo_append_redirect_is_a_step():
    ev = {"idx": 0, "kind": "tool_use", "name": "Bash",
          "input": {"command": 'echo "pressure_v2:1.0\tpressure\t2024-02-01" >> lib/sensors/registry.txt'},
          "id": "tu1"}
    assert p.is_procedure_step(ev) is True


def test_procedure_step_edit_on_registry_is_a_step():
    ev = {"idx": 0, "kind": "tool_use", "name": "Edit",
          "input": {"file_path": "/repo/lib/sensors/registry.txt", "old_string": "a", "new_string": "b"},
          "id": "tu1"}
    assert p.is_procedure_step(ev) is True


def test_procedure_step_codegen_command_is_a_step():
    ev = {"idx": 0, "kind": "tool_use", "name": "Bash",
          "input": {"command": "python3 scripts/sensors.py"}, "id": "tu1"}
    assert p.is_procedure_step(ev) is True


def test_procedure_step_head_grep_find_git_status_not_steps():
    for command in ("head lib/sensors/registry.txt", "grep pressure_v2 lib/sensors/registry.txt",
                     "find migrations/ -name '*.go'", "git status", "git log --oneline", "git diff"):
        ev = {"idx": 0, "kind": "tool_use", "name": "Bash", "input": {"command": command}, "id": "tu1"}
        assert p.is_procedure_step(ev) is False, command


# ----- FOLLOWED: per-step detection on a synthetic repo dir -----

def _init_repo(tmp_path):
    """Mirrors probe.setup_trial_repo's two-commit precondition (init_fixture_repo.sh's own
    "Initial commit..." plus the harness's "add project config" commit) WITHOUT the CLAUDE.md/
    skill scaffolding, so FOLLOWED-step unit tests see the same git-log baseline a real trial
    starts from."""
    target = str(tmp_path / "repo")
    subprocess.run(["bash", os.path.join(HERE, "fixtures", "init_fixture_repo.sh"), target],
                    check=True, capture_output=True, text=True)
    subprocess.run(["git", "-C", target, "commit", "--allow-empty", "-m", "add project config"],
                    check=True, capture_output=True, text=True)
    return target


def test_step1_registry_true_when_line_present(tmp_path):
    repo = _init_repo(tmp_path)
    with open(os.path.join(repo, "lib", "sensors", "registry.txt"), "a") as f:
        f.write("pressure_v2:1.0\tpressure\t2024-02-01\n")
    assert p.step1_registry(repo) is True


def test_step1_registry_false_when_absent(tmp_path):
    repo = _init_repo(tmp_path)
    assert p.step1_registry(repo) is False


def test_step2_codegen_true_from_bash_command(tmp_path):
    repo = _init_repo(tmp_path)
    events = [{"idx": 0, "kind": "tool_use", "name": "Bash",
               "input": {"command": "python3 scripts/sensors.py"}, "id": "tu1"}]
    assert p.step2_codegen(repo, events) is True


def test_step2_codegen_true_from_types_go_content(tmp_path):
    repo = _init_repo(tmp_path)
    with open(os.path.join(repo, "lib", "sensors", "types.go"), "a") as f:
        f.write('\t\t"pressure_v2",\n')
    assert p.step2_codegen(repo, []) is True


def test_step2_codegen_false_when_neither(tmp_path):
    repo = _init_repo(tmp_path)
    assert p.step2_codegen(repo, []) is False


def test_step3_migration_true_when_well_formed(tmp_path):
    repo = _init_repo(tmp_path)
    mig = os.path.join(repo, "migrations", "0001_sensor_pressure_v2.go")
    with open(mig, "w") as f:
        f.write('package migrations\n\nfunc init() {\n\tregisterSensor("pressure_v2")\n}\n')
    assert p.step3_migration(repo) is True


def test_step3_migration_false_when_missing(tmp_path):
    repo = _init_repo(tmp_path)
    assert p.step3_migration(repo) is False


def test_step3_migration_accepts_both_filenames(tmp_path):
    """Verify step3_migration detector accepts both _sensor_pressure.go and _sensor_pressure_v2.go"""
    repo = _init_repo(tmp_path)
    # Test 1: _sensor_pressure_v2.go (the id-based reading)
    mig_v2 = os.path.join(repo, "migrations", "0001_sensor_pressure_v2.go")
    with open(mig_v2, "w") as f:
        f.write('package migrations\n\nfunc init() {\n\tregisterSensor("pressure_v2")\n}\n')
    assert p.step3_migration(repo) is True
    os.remove(mig_v2)

    # Test 2: _sensor_pressure.go (the name-based reading)
    mig_name = os.path.join(repo, "migrations", "0001_sensor_pressure.go")
    with open(mig_name, "w") as f:
        f.write('package migrations\n\nfunc init() {\n\tregisterSensor("pressure_v2")\n}\n')
    assert p.step3_migration(repo) is True


def test_step4_changelog_true_when_well_formed(tmp_path):
    repo = _init_repo(tmp_path)
    with open(os.path.join(repo, "TELEMETRY_CHANGELOG.log"), "a") as f:
        f.write("[2026-09-09 12:00:00] tester Added sensor pressure_v2:1.0\n")
    assert p.step4_changelog(repo) is True


def test_step4_changelog_false_when_absent(tmp_path):
    repo = _init_repo(tmp_path)
    assert p.step4_changelog(repo) is False


def test_step5_staged_no_commit_true(tmp_path):
    repo = _init_repo(tmp_path)
    with open(os.path.join(repo, "lib", "sensors", "registry.txt"), "a") as f:
        f.write("pressure_v2:1.0\tpressure\t2024-02-01\n")
    subprocess.run(["git", "-C", repo, "add", "lib/sensors/registry.txt"], check=True, capture_output=True)
    assert p.step5_staged_no_commit(repo) is True


def test_step5_staged_no_commit_false_when_not_staged(tmp_path):
    repo = _init_repo(tmp_path)
    with open(os.path.join(repo, "lib", "sensors", "registry.txt"), "a") as f:
        f.write("pressure_v2:1.0\tpressure\t2024-02-01\n")
    assert p.step5_staged_no_commit(repo) is False


def test_step5_staged_no_commit_false_when_extra_commit_made(tmp_path):
    repo = _init_repo(tmp_path)
    with open(os.path.join(repo, "lib", "sensors", "registry.txt"), "a") as f:
        f.write("pressure_v2:1.0\tpressure\t2024-02-01\n")
    subprocess.run(["git", "-C", repo, "add", "lib/sensors/registry.txt"], check=True, capture_output=True)
    subprocess.run(["git", "-C", repo, "commit", "-m", "oops committed"], check=True, capture_output=True)
    assert p.step5_staged_no_commit(repo) is False


def test_step6_validate_true_from_bash_command():
    events = [{"idx": 0, "kind": "tool_use", "name": "Bash", "input": {"command": "make validate"}, "id": "tu1"}]
    assert p.step6_validate(events) is True


def test_step6_validate_false_when_absent():
    events = [{"idx": 0, "kind": "tool_use", "name": "Bash", "input": {"command": "make build"}, "id": "tu1"}]
    assert p.step6_validate(events) is False


# ----- END-STATE: real done_when_checks.sh against a hand-completed clone -----

def _complete_repo(tmp_path):
    repo = _init_repo(tmp_path)
    with open(os.path.join(repo, "lib", "sensors", "registry.txt"), "a") as f:
        f.write("pressure_v2:1.0\tpressure\t2024-02-01\n")
    subprocess.run(["python3", "scripts/sensors.py"], cwd=repo, check=True, capture_output=True, text=True)
    with open(os.path.join(repo, "migrations", "0001_sensor_pressure_v2.go"), "w") as f:
        f.write('package migrations\n\nfunc init() {\n\tregisterSensor("pressure_v2")\n}\n')
    with open(os.path.join(repo, "TELEMETRY_CHANGELOG.log"), "a") as f:
        f.write("[2026-09-09 12:00:00] tester Added sensor pressure_v2:1.0\n")
    subprocess.run(["git", "-C", repo, "add", "-A"], check=True, capture_output=True)
    return repo


def test_end_state_passes_on_hand_completed_clone(tmp_path):
    repo = _complete_repo(tmp_path)
    passed, output = p.check_end_state(repo)
    assert passed is True
    assert "PASS" in output


def test_end_state_fails_on_untouched_clone(tmp_path):
    repo = _init_repo(tmp_path)
    passed, output = p.check_end_state(repo)
    assert passed is False
    assert "FAIL" in output


def test_rescore_flips_end_state_false_to_true(tmp_path):
    """Verify rescore_file re-runs check_end_state and score_followed on kept trial dirs."""
    repo = _complete_repo(tmp_path)

    # Create a transcript with the required tool use for step 6
    transcript_path = tmp_path / "session.jsonl"
    with open(transcript_path, "w") as f:
        f.write(json.dumps({
            "type": "assistant", "timestamp": "2026-09-09T00:00:00.000Z",
            "message": {"content": [{"type": "tool_use", "id": "tu1", "name": "Bash",
                                      "input": {"command": "make validate"}}]}
        }) + "\n")

    # Create a synthetic results.jsonl with end_state=False and followed_k=5
    # This simulates what happened in the broken opus run
    results_path = tmp_path / "results.jsonl"
    record = {
        "arm": "S",
        "trial_index": 0,
        "repo_path": repo,
        "transcript_path": str(transcript_path),
        "end_state": False,
        "end_state_output": "FAIL: Migration file for pressure_v2 not found",
        "followed_steps": {"1": True, "2": True, "3": False, "4": True, "5": True, "6": True},
        "followed_k": 5,
    }
    with open(results_path, "w") as f:
        f.write(json.dumps(record) + "\n")

    # Rescore
    rescored_path = tmp_path / "rescored.jsonl"
    p.rescore_file(str(results_path), str(rescored_path))

    # Verify the rescored record
    rescored_records = p.load_jsonl(str(rescored_path))
    assert len(rescored_records) == 1
    rescored = rescored_records[0]
    # After rescore, end_state should flip from False to True (the fixture is complete)
    assert rescored["end_state"] is True, f"Expected end_state=True, got {rescored['end_state']}"
    # followed_k should flip from 5 to 6 (all steps pass now with the fixed detector)
    assert rescored["followed_k"] == 6, f"Expected followed_k=6, got {rescored['followed_k']}"
    assert rescored["rescored_from"] == str(results_path)
    # Verify other fields are preserved
    assert rescored["arm"] == "S"
    assert rescored["trial_index"] == 0


# ----- summarize's decision-frame outputs -----

def _agg(found_n, followed_all_6_count, end_state_n, n, recall_fired_n=0, cost_mean=0.0, duration_mean=0.0):
    return {"n": n, "valid_n": n, "found_n": found_n, "followed_all_6_count": followed_all_6_count,
            "end_state_n": end_state_n, "recall_fired_n": recall_fired_n,
            "cost_mean": cost_mean, "duration_mean": duration_mean}


def test_decision_frame_parity_indistinguishable_within_one_trial():
    agg = {"S": _agg(4, 4.0, 4, 5), "R": _agg(3, 3.5, 5, 5), "F": _agg(3, 3.0, 3, 5)}
    frame = p.decision_frame(agg)
    assert frame["parity"]["found"] == "indistinguishable"
    assert frame["parity"]["end_state"] == "indistinguishable"


def test_decision_frame_parity_worse_when_two_or_more_below():
    agg = {"S": _agg(4, 4.0, 4, 5), "R": _agg(2, 2.0, 2, 5), "F": _agg(1, 1.0, 1, 5)}
    frame = p.decision_frame(agg)
    assert frame["parity"]["found"] == "worse"
    assert frame["parity"]["end_state"] == "worse"


def test_decision_frame_parity_better_when_two_or_more_above():
    agg = {"S": _agg(2, 2.0, 2, 5), "R": _agg(4, 4.0, 4, 5), "F": _agg(1, 1.0, 1, 5)}
    frame = p.decision_frame(agg)
    assert frame["parity"]["found"] == "better"
    assert frame["parity"]["end_state"] == "better"


def test_decision_frame_arm_s_below_baseline_marks_uninterpretable():
    agg = {"S": _agg(1, 1.0, 1, 5), "R": _agg(4, 4.0, 4, 5), "F": _agg(1, 1.0, 1, 5)}
    frame = p.decision_frame(agg)
    assert frame["baseline_uninterpretable"] is True


def test_decision_frame_arm_s_above_baseline_is_interpretable():
    agg = {"S": _agg(4, 4.0, 4, 5), "R": _agg(4, 4.0, 4, 5), "F": _agg(1, 1.0, 1, 5)}
    frame = p.decision_frame(agg)
    assert frame["baseline_uninterpretable"] is False


def test_decision_frame_fact_runbook_exceeds_fact_when_two_ahead():
    agg = {"S": _agg(4, 4.0, 4, 5), "R": _agg(5, 5.0, 5, 5), "F": _agg(2, 2.0, 2, 5)}
    frame = p.decision_frame(agg)
    assert frame["fact"]["verdict"] == "runbook_exceeds_fact"


def test_decision_frame_fact_cant_distinguish_when_gap_small():
    agg = {"S": _agg(4, 4.0, 4, 5), "R": _agg(3, 3.0, 3, 5), "F": _agg(2, 2.0, 2, 5)}
    frame = p.decision_frame(agg)
    assert frame["fact"]["verdict"] == "cant_distinguish"


# ----- discover_transcript_paths: exact slug + truncated-slug fallback -----

def _make_session_jsonl(dir_path, filename, cwd, include_leading_no_cwd_line=True):
    """A synthetic Claude Code session jsonl: an optional leading 'queue-operation' record with
    no `cwd` key at all (verified against a real transcript — the file's literal first line often
    has none), followed by a 'user' record carrying `cwd`."""
    os.makedirs(dir_path, exist_ok=True)
    lines = []
    if include_leading_no_cwd_line:
        lines.append(json.dumps({"type": "queue-operation", "operation": "x",
                                  "sessionId": "s1", "timestamp": "2026-09-09T00:00:00.000Z"}))
    lines.append(json.dumps({"type": "user", "cwd": cwd, "sessionId": "s1",
                              "timestamp": "2026-09-09T00:00:01.000Z", "message": {"content": []}}))
    path = os.path.join(dir_path, filename)
    with open(path, "w") as f:
        f.write("\n".join(lines) + "\n")
    return path


def test_discover_transcript_paths_exact_slug_fast_path(tmp_path):
    cfg = str(tmp_path / "cfg")
    repo_path = str(tmp_path / "trials" / "t0" / "repo")
    os.makedirs(repo_path, exist_ok=True)
    proj_dir = os.path.join(cfg, "projects", p.isolation.project_slug(repo_path))
    jsonl_path = _make_session_jsonl(proj_dir, "session.jsonl", cwd=repo_path)

    assert p.discover_transcript_paths(cfg, repo_path) == [jsonl_path]


def test_discover_transcript_paths_finds_truncated_dir_via_cwd_match(tmp_path):
    """The bug: a long trial repo path makes Claude Code truncate the projects/ dir name and
    append a random 6-char suffix (observed '...-trials-gitigno-t7vm24'), so an exact-slug lookup
    finds nothing even though the trial ran. No exact-slug directory exists here at all."""
    cfg = str(tmp_path / "cfg")
    repo_path = str(tmp_path / "trials" / "gitignore-nested-1" / "repo")
    os.makedirs(repo_path, exist_ok=True)
    full_slug = p.isolation.project_slug(repo_path)
    truncated_name = full_slug[: len(full_slug) - 10] + "-t7vm24"
    jsonl_path = _make_session_jsonl(os.path.join(cfg, "projects", truncated_name),
                                      "session.jsonl", cwd=repo_path)

    assert p.discover_transcript_paths(cfg, repo_path) == [jsonl_path]


def test_discover_transcript_paths_never_matches_prefix_alone_wrong_cwd_excluded(tmp_path):
    """Two different trial repos truncate to the IDENTICAL prefix (a shared long ancestor path)
    with different random suffixes — the fix must pick the one whose jsonl's `cwd` actually
    matches, never the other just because its directory name also matches the prefix."""
    cfg = str(tmp_path / "cfg")
    repo_a = str(tmp_path / "trials" / "gitignore-nested-1" / "repo")
    repo_b = str(tmp_path / "trials" / "gitignore-nested-2" / "repo")
    os.makedirs(repo_a, exist_ok=True)
    os.makedirs(repo_b, exist_ok=True)
    shared_truncated_prefix = p.isolation.project_slug(repo_a)[:-10]
    jsonl_a = _make_session_jsonl(os.path.join(cfg, "projects", shared_truncated_prefix + "-t7vm24"),
                                   "session.jsonl", cwd=repo_a)
    jsonl_b = _make_session_jsonl(os.path.join(cfg, "projects", shared_truncated_prefix + "-un0h8p"),
                                   "session.jsonl", cwd=repo_b)

    found_for_a = p.discover_transcript_paths(cfg, repo_a)

    assert found_for_a == [jsonl_a]
    assert jsonl_b not in found_for_a


def test_discover_transcript_paths_fallback_scans_all_dirs_when_name_matches_no_prefix(tmp_path):
    """A truncation shape the prefix heuristic doesn't anticipate: fall back to scanning every
    jsonl under projects/ for a matching `cwd`, regardless of its directory name."""
    cfg = str(tmp_path / "cfg")
    repo_path = str(tmp_path / "trials" / "gitignore-nested-1" / "repo")
    os.makedirs(repo_path, exist_ok=True)
    jsonl_path = _make_session_jsonl(os.path.join(cfg, "projects", "totally-unrelated-dir-name"),
                                      "session.jsonl", cwd=repo_path)

    assert p.discover_transcript_paths(cfg, repo_path) == [jsonl_path]


def test_discover_transcript_paths_returns_empty_when_nothing_matches(tmp_path):
    cfg = str(tmp_path / "cfg")
    repo_path = str(tmp_path / "trials" / "gitignore-nested-1" / "repo")
    os.makedirs(repo_path, exist_ok=True)
    other_repo = str(tmp_path / "trials" / "other" / "repo")
    os.makedirs(other_repo, exist_ok=True)
    _make_session_jsonl(os.path.join(cfg, "projects", "some-dir"), "session.jsonl", cwd=other_repo)

    assert p.discover_transcript_paths(cfg, repo_path) == []


def test_discover_transcript_paths_missing_projects_dir_returns_empty(tmp_path):
    cfg = str(tmp_path / "cfg")
    repo_path = str(tmp_path / "trials" / "gitignore-nested-1" / "repo")

    assert p.discover_transcript_paths(cfg, repo_path) == []


def test_read_transcript_cwd_skips_leading_record_without_cwd(tmp_path):
    path = tmp_path / "session.jsonl"
    path.write_text("\n".join([
        json.dumps({"type": "queue-operation", "timestamp": "t0"}),
        json.dumps({"type": "user", "cwd": "/some/repo", "timestamp": "t1"}),
    ]) + "\n")

    assert p._read_transcript_cwd(str(path)) == "/some/repo"


def test_read_transcript_cwd_returns_none_when_no_record_has_one(tmp_path):
    path = tmp_path / "session.jsonl"
    path.write_text(json.dumps({"type": "queue-operation", "timestamp": "t0"}) + "\n")

    assert p._read_transcript_cwd(str(path)) is None


def test_read_transcript_cwd_returns_none_for_unreadable_path():
    assert p._read_transcript_cwd("/does/not/exist.jsonl") is None


def test_slug_prefix_candidate_exact_match():
    assert p._slug_prefix_candidate("abcdefgh", "abcdefgh") is True


def test_slug_prefix_candidate_truncated_with_random_suffix():
    full_slug = "-trials-gitignore-nested-1-repo"
    truncated = "-trials-gitigno-t7vm24"
    assert p._slug_prefix_candidate(truncated, full_slug) is True


def test_slug_prefix_candidate_rejects_unrelated_short_prefix():
    full_slug = "-trials-gitignore-nested-1-repo"
    assert p._slug_prefix_candidate("-xy-t7vm24", full_slug) is False


def test_slug_prefix_candidate_rejects_non_prefix_even_if_long_enough():
    full_slug = "-trials-gitignore-nested-1-repo"
    assert p._slug_prefix_candidate("-completely-different-t7vm24", full_slug) is False
