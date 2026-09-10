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
    events = p.parse_transcript_events([])
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


# ----- summarize's decision-frame outputs -----

def _agg(found_n, followed_trial_equiv, end_state_n, n, recall_fired_n=0, cost_mean=0.0, duration_mean=0.0):
    return {"n": n, "valid_n": n, "found_n": found_n, "followed_trial_equiv": followed_trial_equiv,
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
