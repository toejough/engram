"""Offline tests for ccells.py (task 2, ruling V4): the confined curate-skill-wording validation
launcher. Pure-function tests plus a crash-path wiring test in the style of test_lar.py's
test_batch_writes_isolation_and_manifest_even_when_an_arm_crashes. No API calls, no reads outside
tmp_path or this repo's own git history (RED_PIN), no writes outside tmp_path."""

import json
import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import ccells  # noqa: E402
import lar  # noqa: E402


# ---------------------------------------------------------------------------
# fixture flags
# ---------------------------------------------------------------------------


def test_fixture_flags_green_is_the_base_list():
    assert ccells.fixture_flags("GREEN") == ccells.FIXTURE_FLAGS
    assert ccells.fixture_flags("RED") == ccells.FIXTURE_FLAGS
    assert len(ccells.FIXTURE_FLAGS) == 22


def test_fixture_flags_pressure_adds_the_extra_list():
    pressure = ccells.fixture_flags("PRESSURE")
    assert pressure == ccells.FIXTURE_FLAGS + ccells.PRESSURE_EXTRA_FLAGS
    assert len(pressure) == 40


def test_fixture_flags_green_and_pressure_do_not_share_mutable_state():
    got = ccells.fixture_flags("GREEN")
    got.append("mutated")
    assert "mutated" not in ccells.fixture_flags("GREEN")
    assert "mutated" not in ccells.FIXTURE_FLAGS


# ---------------------------------------------------------------------------
# fixture note
# ---------------------------------------------------------------------------


@pytest.mark.parametrize("cell", ["RED", "GREEN", "PRESSURE"])
def test_fixture_note_carries_every_flag_quoted_in_frontmatter_and_numbered_in_body(cell):
    note = ccells.fixture_note(cell)
    flags = ccells.fixture_flags(cell)
    frontmatter, body = note.split("---\n\n", 1)
    for i, flag in enumerate(flags, start=1):
        assert f'    - "{flag}"' in frontmatter
        assert f"{i}. Watch for: {flag}." in body


def test_fixture_note_is_a_pending_skill_runbook_note():
    note = ccells.fixture_note("GREEN")
    assert "type: runbook\n" in note
    assert 'skill_key: "claude:ccells-fixture-skill"\n' in note
    assert "pending: true\n" in note
    assert note.startswith("---\n")


def test_fixture_note_pressure_has_more_red_flags_entries_than_green():
    green_count = ccells.fixture_note("GREEN").count("Watch for:")
    pressure_count = ccells.fixture_note("PRESSURE").count("Watch for:")
    assert pressure_count > green_count
    assert pressure_count == 40


# ---------------------------------------------------------------------------
# curate text sources
# ---------------------------------------------------------------------------


def test_curate_text_red_reads_the_pinned_pre_fix_commit():
    text = ccells.curate_text("RED")
    assert "1200" in text  # the pre-#772-fix wording still used the bare byte cap
    assert "preview budget" not in text


def test_curate_text_green_reads_the_current_worktree_file():
    with open(os.path.join(ccells.REPO, ccells.CURATE_SKILL_REL)) as f:
        want = f.read()
    assert ccells.curate_text("GREEN") == want
    assert ccells.curate_text("PRESSURE") == want


def test_marked_curate_text_appends_exactly_one_marker_for_its_own_cell():
    for cell in ("RED", "GREEN", "PRESSURE"):
        text = ccells.marked_curate_text(cell)
        assert text.endswith(ccells.MARKER_PREFIX + ccells.MARKERS[cell] + "\n")
        assert text.startswith(ccells.curate_text(cell))
        for other, token in ccells.MARKERS.items():
            if other != cell:
                assert token not in text


# ---------------------------------------------------------------------------
# scoring: entries kept/dropped, delivery gate
# ---------------------------------------------------------------------------


def test_entries_kept_all_present_drops_nothing():
    note = ccells.fixture_note("GREEN")
    kept, dropped = ccells.entries_kept(note, "GREEN")
    assert kept == ccells.FIXTURE_FLAGS
    assert dropped == []


def test_entries_kept_reports_a_missing_entry_as_dropped():
    flags = ccells.fixture_flags("GREEN")
    written = "\n".join(f for f in flags if f != flags[3])
    kept, dropped = ccells.entries_kept(written, "GREEN")
    assert dropped == [flags[3]]
    assert flags[3] not in kept
    assert len(kept) == len(flags) - 1


def test_entries_kept_pressure_uses_the_longer_list():
    written = "\n".join(ccells.fixture_flags("PRESSURE"))
    kept, dropped = ccells.entries_kept(written, "PRESSURE")
    assert len(kept) == 40
    assert dropped == []


def test_delivery_gate_ok_with_only_the_own_marker():
    text = "some transcript text " + ccells.MARKERS["GREEN"] + " more text"
    gate = ccells.delivery_gate(text, "GREEN")
    assert gate == {"own_marker_present": True, "foreign_markers_present": [], "ok": True}


def test_delivery_gate_fails_without_the_own_marker():
    gate = ccells.delivery_gate("nothing relevant here", "GREEN")
    assert gate["own_marker_present"] is False
    assert gate["ok"] is False


def test_delivery_gate_fails_when_a_foreign_marker_leaked_in():
    text = ccells.MARKERS["GREEN"] + " ... " + ccells.MARKERS["RED"]
    gate = ccells.delivery_gate(text, "GREEN")
    assert gate["ok"] is False
    assert gate["foreign_markers_present"] == [ccells.MARKERS["RED"]]


# ---------------------------------------------------------------------------
# read_written_note
# ---------------------------------------------------------------------------


def test_read_written_note_returns_empty_string_when_the_vault_copy_has_no_fixture_note(tmp_path):
    assert ccells.read_written_note(str(tmp_path)) == ""


def test_read_written_note_reads_the_fixture_basename(tmp_path):
    path = tmp_path / f"{ccells.FIXTURE_BASENAME}.md"
    path.write_text("note body")
    assert ccells.read_written_note(str(tmp_path)) == "note body"


# ---------------------------------------------------------------------------
# score_arm
# ---------------------------------------------------------------------------


def _write_stream(out_dir, result_text, cost=0.1):
    event = {"type": "result", "subtype": "success", "result": result_text, "total_cost_usd": cost}
    with open(os.path.join(out_dir, "stream.jsonl"), "w") as f:
        f.write(json.dumps(event) + "\n")


def _write_vault_note(out_dir, cell, flags=None):
    """Writes the vault copy of the fixture note. With flags given, writes a note whose body
    mentions only those flags (entries_kept is a plain substring check, so the exact frontmatter
    shape doesn't matter for the dropped-entry tests)."""
    vault = os.path.join(out_dir, "vault")
    os.makedirs(vault, exist_ok=True)
    if flags is None:
        note = ccells.fixture_note(cell)
    else:
        note = "---\ntype: runbook\npending: true\n---\n\n" + "\n".join(f"Watch for: {f}." for f in flags) + "\n"
    with open(os.path.join(vault, f"{ccells.FIXTURE_BASENAME}.md"), "w") as f:
        f.write(note)
    return note


def test_score_arm_green_pass_when_marker_present_and_nothing_dropped(tmp_path):
    out_dir = tmp_path / "00-GREEN"
    out_dir.mkdir()
    _write_stream(str(out_dir), "Kept all 22 red_flags. " + ccells.MARKERS["GREEN"], cost=0.25)
    _write_vault_note(str(out_dir), "GREEN")
    rec = ccells.score_arm(str(out_dir), "GREEN")
    assert rec["label"] == "PASS"
    assert rec["entries_dropped"] == 0
    assert rec["entries_total"] == 22
    assert rec["cost_usd"] == 0.25
    assert rec["delivery_gate"]["ok"] is True


def test_score_arm_green_fails_when_an_entry_is_dropped(tmp_path):
    out_dir = tmp_path / "00-GREEN"
    out_dir.mkdir()
    _write_stream(str(out_dir), "done " + ccells.MARKERS["GREEN"])
    dropped_flags = ccells.fixture_flags("GREEN")[1:]
    _write_vault_note(str(out_dir), "GREEN", flags=dropped_flags)
    rec = ccells.score_arm(str(out_dir), "GREEN")
    assert rec["label"] == "FAIL"
    assert rec["entries_dropped"] == 1
    assert ccells.fixture_flags("GREEN")[0] in rec["dropped_entries"]


def test_score_arm_green_fails_when_the_delivery_marker_is_missing(tmp_path):
    out_dir = tmp_path / "00-GREEN"
    out_dir.mkdir()
    _write_stream(str(out_dir), "done, no marker here")
    _write_vault_note(str(out_dir), "GREEN")
    rec = ccells.score_arm(str(out_dir), "GREEN")
    assert rec["label"] == "FAIL"
    assert rec["delivery_gate"]["ok"] is False


def test_score_arm_red_is_observational_not_pass_fail(tmp_path):
    out_dir = tmp_path / "00-RED"
    out_dir.mkdir()
    _write_stream(str(out_dir), "done " + ccells.MARKERS["RED"])
    dropped_flags = ccells.fixture_flags("RED")[1:]
    _write_vault_note(str(out_dir), "RED", flags=dropped_flags)
    rec = ccells.score_arm(str(out_dir), "RED")
    assert rec["label"] == "drop-observed"
    assert rec["drop_observed"] is True
    assert rec["label"] not in ("PASS", "FAIL")


def test_score_arm_red_no_drop_is_labelled_no_drop(tmp_path):
    out_dir = tmp_path / "00-RED"
    out_dir.mkdir()
    _write_stream(str(out_dir), "done " + ccells.MARKERS["RED"])
    _write_vault_note(str(out_dir), "RED")
    rec = ccells.score_arm(str(out_dir), "RED")
    assert rec["label"] == "no-drop"
    assert rec["drop_observed"] is False


def test_score_arm_includes_session_transcript_text_in_the_delivery_gate(tmp_path):
    out_dir = tmp_path / "00-GREEN"
    session_dir = out_dir / "session" / "sub"
    session_dir.mkdir(parents=True)
    _write_stream(str(out_dir), "done, no marker in the main stream")
    (session_dir / "a.jsonl").write_text(ccells.MARKERS["GREEN"] + "\n")
    _write_vault_note(str(out_dir), "GREEN")
    rec = ccells.score_arm(str(out_dir), "GREEN")
    assert rec["delivery_gate"]["ok"] is True
    assert rec["label"] == "PASS"


def test_score_arm_written_note_present_is_false_when_the_vault_copy_is_missing(tmp_path):
    out_dir = tmp_path / "00-GREEN"
    out_dir.mkdir()
    _write_stream(str(out_dir), "done " + ccells.MARKERS["GREEN"])
    rec = ccells.score_arm(str(out_dir), "GREEN")
    assert rec["written_note_present"] is False
    assert rec["label"] == "FAIL"


# ---------------------------------------------------------------------------
# cumulative_cost
# ---------------------------------------------------------------------------


def test_cumulative_cost_is_zero_with_no_log_file(tmp_path, monkeypatch):
    monkeypatch.setattr(ccells, "RESULTS", str(tmp_path))
    assert ccells.cumulative_cost() == 0.0


def test_cumulative_cost_sums_cost_usd_across_lines(tmp_path, monkeypatch):
    monkeypatch.setattr(ccells, "RESULTS", str(tmp_path))
    log = tmp_path / "ccells-cost-log.jsonl"
    log.write_text("\n".join(json.dumps(r) for r in [
        {"cost_usd": 0.1}, {"cost_usd": 0.2}, {"cost_usd": None}, {"other": "x"},
    ]) + "\n")
    assert ccells.cumulative_cost() == pytest.approx(0.3)


def test_cumulative_cost_skips_blank_lines(tmp_path, monkeypatch):
    monkeypatch.setattr(ccells, "RESULTS", str(tmp_path))
    log = tmp_path / "ccells-cost-log.jsonl"
    log.write_text('\n{"cost_usd": 0.5}\n\n')
    assert ccells.cumulative_cost() == pytest.approx(0.5)


# ---------------------------------------------------------------------------
# build_ccell_arm
# ---------------------------------------------------------------------------


def test_build_ccell_arm_writes_marked_skill_fixture_note_and_copies_the_binary(tmp_path):
    engram_bin = tmp_path / "engram"
    engram_bin.write_text("#!/bin/sh\n")
    engram_bin.chmod(0o755)
    arm = tmp_path / "engram-arm.TEST"
    arm.mkdir()
    ccells.build_ccell_arm(str(arm), "GREEN", str(engram_bin))

    skill = (arm / "home" / ".claude" / "skills" / "curate" / "SKILL.md").read_text()
    assert skill == ccells.marked_curate_text("GREEN")
    assert not (arm / "home" / ".claude" / "CLAUDE.md").exists()
    assert not (arm / "home" / ".claude" / "skills" / "learn").exists()

    note = (arm / "vault" / f"{ccells.FIXTURE_BASENAME}.md").read_text()
    assert note == ccells.fixture_note("GREEN")

    assert os.path.exists(arm / "bin" / "engram")


# ---------------------------------------------------------------------------
# run_batch wiring: refusals and the crash path still writes isolation + manifest
# ---------------------------------------------------------------------------


def test_run_batch_refuses_an_unknown_cell(tmp_path, monkeypatch):
    monkeypatch.setattr(ccells, "RESULTS", str(tmp_path / "results"))
    with pytest.raises(ccells.CcellsError, match="unknown cell"):
        ccells.run_batch("x", "BLUE", 1, "sonnet", 10, "/x/claude", "/x/engram", ccells.COST_CAP_USD)


def test_run_batch_refuses_an_existing_batch_dir(tmp_path, monkeypatch):
    results = tmp_path / "results"
    (results / "ccells-dup").mkdir(parents=True)
    monkeypatch.setattr(ccells, "RESULTS", str(results))
    with pytest.raises(ccells.CcellsError, match="exists"):
        ccells.run_batch("dup", "GREEN", 1, "sonnet", 10, "/x/claude", "/x/engram", ccells.COST_CAP_USD)


def _stub_isolation(monkeypatch, tmp_path):
    vault, cdir = tmp_path / "vault", tmp_path / "dotclaude"
    vault.mkdir()
    cdir.mkdir()
    monkeypatch.setattr(lar, "real_paths", lambda: ("/h", str(vault), str(cdir)))


def test_run_batch_aborts_before_any_arm_when_the_cost_cap_is_already_reached(tmp_path, monkeypatch):
    monkeypatch.setattr(ccells, "RESULTS", str(tmp_path / "results"))
    _stub_isolation(monkeypatch, tmp_path)
    monkeypatch.setattr(ccells, "cumulative_cost", lambda: 100.0)

    def boom(*a, **k):
        raise AssertionError("must not run the probe once the cap is already reached")
    monkeypatch.setattr(lar, "run_probe", boom)

    with pytest.raises(ccells.CcellsError, match="cost cap"):
        ccells.run_batch("capped", "GREEN", 1, "sonnet", 10, "/x/claude", "/x/engram", cost_cap=1.0)

    manifest = json.loads((tmp_path / "results" / "ccells-capped" / "run-manifest.json").read_text())
    assert manifest["cost_before_batch_usd"] == 100.0
    assert "cost cap" in manifest["error"]
    assert (tmp_path / "results" / "ccells-capped" / "isolation.json").exists()


def test_run_batch_writes_isolation_and_manifest_even_when_an_arm_crashes(tmp_path, monkeypatch):
    monkeypatch.setattr(ccells, "RESULTS", str(tmp_path / "results"))
    _stub_isolation(monkeypatch, tmp_path)
    monkeypatch.setattr(ccells, "cumulative_cost", lambda: 0.0)
    monkeypatch.setattr(lar, "read_token", lambda: "tok-FAKE")
    monkeypatch.setattr(lar, "claude_version", lambda b: "stub")
    monkeypatch.setattr(lar, "deny_entries", lambda r: [])
    monkeypatch.setattr(lar, "run_probe", lambda *a: {"ok": True, "reason": "stub", "cost_usd": 0.0})

    engram_bin = tmp_path / "engram"
    engram_bin.write_bytes(b"#!/bin/sh\n")

    def boom(*a, **k):
        raise ccells.CcellsError("arm build exploded")
    monkeypatch.setattr(ccells, "build_ccell_arm", boom)

    with pytest.raises(ccells.CcellsError, match="arm build exploded"):
        ccells.run_batch("crash", "GREEN", 1, "sonnet", 10, "/x/claude", str(engram_bin), ccells.COST_CAP_USD)

    batch_dir = tmp_path / "results" / "ccells-crash"
    assert (batch_dir / "isolation.json").exists()
    manifest = json.loads((batch_dir / "run-manifest.json").read_text())
    assert "arm build exploded" in manifest["error"]
    assert manifest["arms"] == []


def test_run_batch_isolation_failure_is_recorded_without_masking_cost_cap_success(tmp_path, monkeypatch):
    """A batch that completes its (zero) arms cleanly but whose real-vault isolation check fails
    still records the failure in the manifest, exactly as lar.run_batch does (shared helper)."""
    monkeypatch.setattr(ccells, "RESULTS", str(tmp_path / "results"))
    vault, cdir = tmp_path / "vault", tmp_path / "dotclaude"
    vault.mkdir()
    cdir.mkdir()
    calls = {"n": 0}

    def flaky_real_paths():
        calls["n"] += 1
        return "/h", str(vault), str(cdir)
    monkeypatch.setattr(lar, "real_paths", flaky_real_paths)
    monkeypatch.setattr(ccells, "cumulative_cost", lambda: 0.0)
    monkeypatch.setattr(lar, "read_token", lambda: "tok-FAKE")
    monkeypatch.setattr(lar, "claude_version", lambda b: "stub")
    monkeypatch.setattr(lar, "deny_entries", lambda r: [])

    def probe_then_dirty(*a):
        # simulate a real-vault write landing after the "before" snapshot was taken
        (vault / "stray.md").write_text("x")
        return {"ok": True, "reason": "stub", "cost_usd": 0.0}
    monkeypatch.setattr(lar, "run_probe", probe_then_dirty)

    engram_bin = tmp_path / "engram"
    engram_bin.write_bytes(b"#!/bin/sh\n")

    status = ccells.run_batch("dirty", "GREEN", 0, "sonnet", 10, "/x/claude", str(engram_bin), ccells.COST_CAP_USD)
    assert status != 0
    manifest = json.loads((tmp_path / "results" / "ccells-dirty" / "run-manifest.json").read_text())
    assert manifest["failed"] == "real-vault entry newer than batch start"
