"""TDD for the recency-value scorers (#646 cross-context org-RUNLOG scenario) + runner helpers.

The scenario measures whether engram's recency channel resurfaces phase-1's discovery of an
idiosyncratic ORG-WIDE convention (the RUNLOG audit line, `sig=QX7Z`) after context loss. Phase 1
(`notes add`) and phase 2 (`orders report`) are commands in DIFFERENT tools.

Surfacing (note-197 instrument-fidelity fix) is measured from the agent's ACTUAL in-band recall
payloads in the phase-2 transcript, path-matched to phase-1's chunk by session id (robust to the
recall skill's --lazy-chunks zeroing content) — NOT a post-hoc out-of-band re-query that read a
post-`ingest --auto`-polluted index. Validated against the two real re-pilot transcripts on disk
(skipped when that job-scratch dir is gone) AND synthetic payloads that prove the matcher genuinely
detects recency-channel delivery when it is present.

build_trial_env / _build_ws / _ingest_padding are runner helpers; per-trial chunk-index isolation
is mandatory (LEDGER harder-regime-op-cost-unmeasurable, the contamination bug), each workspace is
a strict whitelist (Gate B FIX 1), and the claude cwd is isolation-clean (Task 1) so the recall
sweep can't pollute the recency channel with operator-global memory.
"""
import glob
import json
import os
import sys

import pytest

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), "..")))
import score
import recency_value as rv

GOLDEN = os.path.abspath(os.path.join(
    os.path.dirname(__file__), "..", "..", "cumulative", "testdata", "recency_with_R.yaml"))

# The two real re-pilot transcripts (job-scratch — skipped when absent). Globbed because the
# projects/<slug> subdir is derived from the workspace path.
_REPILOT = "/Users/joe/.claude/jobs/ac0c61e1/tmp/recency-value2/trials"


def _one(pattern):
    hits = glob.glob(pattern)
    return hits[0] if hits else None


ON_P2 = _one(f"{_REPILOT}/on-0-d4mcw71e/cfg-phase2/projects/*/*.jsonl")
ON_P1 = _one(f"{_REPILOT}/on-0-d4mcw71e/cfg-phase1/projects/*/*.jsonl")
OFF_P2 = _one(f"{_REPILOT}/off-0-u6y_cuso/cfg-phase2/projects/*/*.jsonl")
OFF_P1 = _one(f"{_REPILOT}/off-0-u6y_cuso/cfg-phase1/projects/*/*.jsonl")
_HAVE_REPILOT = all([ON_P2, ON_P1, OFF_P2, OFF_P1])

# Synthetic in-band payloads (real `engram query --lazy-chunks` shape: no content, path + prov).
PHASE1_TX = "/tmp/x/cfg-phase1/projects/slug/abc12345-dead-beef-0000-111122223333.jsonl"
_SID = "abc12345-dead-beef-0000-111122223333"

RECENT_INBAND = (
    "items:\n"
    f"  - path: /tmp/x/cfg-phase1/projects/slug/{_SID}.jsonl#turn-1\n"
    "    kind: chunk\n"
    "    score: 0.09\n"
    "    provenances:\n"
    "      - recent\n"
)
DIRECT_INBAND = (
    "items:\n"
    f"  - path: /tmp/x/cfg-phase1/projects/slug/{_SID}.jsonl#turn-1\n"
    "    kind: chunk\n"
    "    score: 0.66\n"
    "    provenances:\n"
    "      - direct\n"
)
NONREF_INBAND = (
    "items:\n"
    "  - path: /Users/joe/.claude/CLAUDE.md#Directory Structure\n"
    "    kind: chunk\n"
    "    score: 0.5\n"
    "    provenances:\n"
    "      - direct\n"
)


# --- report_revenue_ok (kept) ------------------------------------------------------------

def test_report_revenue_ok_accepts_total():
    assert score.report_revenue_ok("total revenue: $483.00\n", 483.00) is True


def test_report_revenue_ok_rejects_wrong_total():
    assert score.report_revenue_ok("total revenue: $99.00\n", 483.00) is False


def test_report_revenue_ok_no_match_is_false():
    assert score.report_revenue_ok("build succeeded\n", 483.00) is False


# --- notes_add_ok (phase-1 capture gate) -------------------------------------------------

def test_notes_add_ok_true_when_runlog_has_valid_add_line(tmp_path):
    (tmp_path / "RUNLOG").write_text("RUNLOG v1 :: ADD :: rows=3 :: sig=QX7Z\n")
    assert score.notes_add_ok(str(tmp_path)) is True


def test_notes_add_ok_false_when_runlog_absent(tmp_path):
    assert score.notes_add_ok(str(tmp_path)) is False


def test_notes_add_ok_false_when_signature_wrong(tmp_path):
    (tmp_path / "RUNLOG").write_text("RUNLOG v1 :: ADD :: rows=3 :: sig=ZZZZ\n")
    assert score.notes_add_ok(str(tmp_path)) is False


def test_notes_add_ok_false_when_line_malformed(tmp_path):
    (tmp_path / "RUNLOG").write_text("notes added ok\n")
    assert score.notes_add_ok(str(tmp_path)) is False


# --- runlog_report_ok (PRIMARY correctness) ----------------------------------------------

def test_runlog_report_ok_true_for_valid_report_line(tmp_path):
    runlog = tmp_path / "RUNLOG"
    runlog.write_text("RUNLOG v1 :: REPORT :: rows=10 :: sig=QX7Z\n")
    assert score.runlog_report_ok(str(runlog)) is True


def test_runlog_report_ok_false_when_only_add_line(tmp_path):
    runlog = tmp_path / "RUNLOG"
    runlog.write_text("RUNLOG v1 :: ADD :: rows=3 :: sig=QX7Z\n")
    assert score.runlog_report_ok(str(runlog)) is False


def test_runlog_report_ok_false_when_absent(tmp_path):
    assert score.runlog_report_ok(str(tmp_path / "RUNLOG")) is False


def test_runlog_report_ok_false_when_signature_wrong(tmp_path):
    runlog = tmp_path / "RUNLOG"
    runlog.write_text("RUNLOG v1 :: REPORT :: rows=10 :: sig=NOPE\n")
    assert score.runlog_report_ok(str(runlog)) is False


# --- extract_query_payloads --------------------------------------------------------------

def _jsonl_line(tool_result_text):
    return json.dumps({"type": "user", "message": {"content": [
        {"type": "tool_result", "tool_use_id": "t1", "content": tool_result_text}]}})


def test_extract_query_payloads_picks_query_results_only(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text("\n".join([
        _jsonl_line("items:\n  - path: /a#x\n    provenances:\n      - direct\n"),  # a query payload
        _jsonl_line("just some chunk text from show-chunk, no items block"),        # not a payload
        json.dumps({"type": "assistant", "message": {"content": [
            {"type": "tool_use", "name": "Bash", "input": {"command": "engram query --lazy-chunks"}}]}}),
    ]))
    payloads = score.extract_query_payloads(str(p))
    assert len(payloads) == 1
    assert "items:" in payloads[0]


def test_extract_query_payloads_missing_file_is_empty():
    assert score.extract_query_payloads("/no/such/transcript.jsonl") == []
    assert score.extract_query_payloads(None) == []


# --- surfaced_*_inband (synthetic — proves the matcher genuinely detects recency delivery) --

def test_surfaced_via_recency_inband_true_when_phase1_chunk_is_recent():
    assert score.surfaced_via_recency_inband([RECENT_INBAND], PHASE1_TX) is True
    assert score.surfaced_any_inband([RECENT_INBAND], PHASE1_TX) is True


def test_surfaced_via_recency_inband_false_when_phase1_chunk_is_direct_only():
    # delivered via cosine (`direct`), not the recency channel — surfaced_any True, via_recency False
    assert score.surfaced_any_inband([DIRECT_INBAND], PHASE1_TX) is True
    assert score.surfaced_via_recency_inband([DIRECT_INBAND], PHASE1_TX) is False


def test_surfaced_inband_false_when_phase1_chunk_absent():
    assert score.surfaced_any_inband([NONREF_INBAND], PHASE1_TX) is False
    assert score.surfaced_via_recency_inband([NONREF_INBAND], PHASE1_TX) is False


def test_surfaced_inband_false_for_empty_payload_list():
    assert score.surfaced_any_inband([], PHASE1_TX) is False
    assert score.surfaced_via_recency_inband([], PHASE1_TX) is False


# --- surfaced_*_inband validated against the REAL re-pilot transcripts (the key step) -----

@pytest.mark.skipif(not _HAVE_REPILOT, reason="re-pilot job-scratch transcripts not on disk")
def test_inband_surfacing_on_real_repilot_transcripts():
    on_payloads = score.extract_query_payloads(ON_P2)
    off_payloads = score.extract_query_payloads(OFF_P2)

    # ON: the agent recalled (>=1 payload); its real recall surfaced phase-1's chunk...
    assert len(on_payloads) >= 1
    assert score.surfaced_any_inband(on_payloads, ON_P1) is True
    # ...but via `direct` cosine, NOT the recency channel: the recall skill's own `ingest --auto`
    # polluted the per-trial index (swept ancestor .claude dirs), deduping the recent channel empty
    # and displacing phase-1's chunk. This is the honest instrument reading — the matcher is not
    # hardcoded; RECENT_INBAND above proves it WOULD read True on a genuine recent-channel hit.
    assert score.surfaced_via_recency_inband(on_payloads, ON_P1) is False

    # OFF: the agent never recalled at all (0 query payloads) — the outcome gap is firing-driven.
    assert off_payloads == []
    assert score.surfaced_any_inband(off_payloads, OFF_P1) is False
    assert score.surfaced_via_recency_inband(off_payloads, OFF_P1) is False


# --- structural parsers validated against the REAL golden shape --------------------------

def test_block_parsers_on_real_golden_shape():
    payload = open(GOLDEN).read()
    blocks = score._item_blocks(payload)
    assert len(blocks) == 2
    # recent-provenance item first, direct second (per the golden)
    assert "recent" in score._block_provenances(blocks[0])
    assert "direct" in score._block_provenances(blocks[1])
    assert score._block_path(blocks[0]).endswith("lesson-recent.md")


# --- recall_fired -------------------------------------------------------------------------

def test_recall_fired_detects_engram_query(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text('{"content":"... engram query --phrase ..."}')
    assert score.recall_fired(str(p)) is True


def test_recall_fired_false_when_absent(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text('{"content":"... wrote the report command ..."}')
    assert score.recall_fired(str(p)) is False


# --- phase1_used_learn (Gate B FIX 3 P1 check) -------------------------------------------

def test_phase1_used_learn_detects_engram_learn(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text('{"content":"... engram learn --subject notes ..."}')
    assert score.phase1_used_learn(str(p)) is True


def test_phase1_used_learn_false_when_absent(tmp_path):
    p = tmp_path / "t.jsonl"
    p.write_text('{"content":"... built notes add, no memory writes ..."}')
    assert score.phase1_used_learn(str(p)) is False


# --- build_trial_env (Task 2 Step 1) -------------------------------------------------------

def test_build_trial_env_per_trial_unique_chunks_dir(tmp_path):
    env_a = rv.build_trial_env("on", str(tmp_path / "trial-a"))
    env_b = rv.build_trial_env("on", str(tmp_path / "trial-b"))
    assert env_a["ENGRAM_CHUNKS_DIR"] and env_b["ENGRAM_CHUNKS_DIR"]
    assert env_a["ENGRAM_CHUNKS_DIR"] != env_b["ENGRAM_CHUNKS_DIR"]
    assert env_a["ENGRAM_VAULT_PATH"]


def test_build_trial_env_recent_fill_absent_for_arm_on(tmp_path):
    env = rv.build_trial_env("on", str(tmp_path / "trial"))
    assert "ENGRAM_RECENT_FILL" not in env


def test_build_trial_env_recent_fill_off_for_arm_off(tmp_path):
    env = rv.build_trial_env("off", str(tmp_path / "trial"))
    assert env["ENGRAM_RECENT_FILL"] == "-1"


# --- _build_ws whitelist (Gate B FIX 1) --------------------------------------------------

def test_build_ws2_is_orders_tool_whitelist(tmp_path):
    ws2 = str(tmp_path / "ws2")
    rv._build_ws(ws2, rv.WS2_FILES)
    assert sorted(os.listdir(ws2)) == ["SPEC.md", "orders.db.json"]
    assert json.load(open(os.path.join(ws2, "orders.db.json")))[0]["amt"] == 19.99
    # SPEC.md is the ORDERS tool spec, and must NOT name the org RUNLOG convention.
    spec = open(os.path.join(ws2, "SPEC.md")).read().lower()
    assert "orders" in spec
    assert "runlog" not in spec and "qx7z" not in spec and "sig=" not in spec


def test_build_ws1_is_notes_tool_whitelist_with_executable_check(tmp_path):
    ws1 = str(tmp_path / "ws1")
    rv._build_ws(ws1, rv.WS1_FILES)
    assert sorted(os.listdir(ws1)) == ["SPEC.md", "check.sh", "items.txt"]
    # SPEC.md is the NOTES tool spec (a different tool than phase-2's orders), no RUNLOG leak.
    spec = open(os.path.join(ws1, "SPEC.md")).read().lower()
    assert "notes" in spec and "runlog" not in spec and "qx7z" not in spec
    # check.sh must keep its exec bit (copy2 preserves mode) so phase 1 can run it.
    assert os.access(os.path.join(ws1, "check.sh"), os.X_OK)


def test_build_ws_raises_if_extra_file_present(tmp_path):
    ws = tmp_path / "ws"
    ws.mkdir()
    (ws / "leaked.sh").write_text("RUNLOG v1 :: <CMD> :: rows=<N> :: sig=QX7Z")
    with pytest.raises(RuntimeError):
        rv._build_ws(str(ws), rv.WS2_FILES)


# --- workspace isolation guard (Task 1: stop the --auto recall sweep pollution) ----------

def test_assert_clean_cwd_passes_for_plain_temp(tmp_path):
    ws = tmp_path / "ws"
    ws.mkdir()
    rv._assert_clean_cwd(str(ws))  # no .claude / VCS ancestor -> no raise


def test_assert_clean_cwd_raises_under_claude_ancestor(tmp_path):
    # workspace nested under a `.claude` dir — engram ingest --auto would sweep it
    (tmp_path / ".claude").mkdir()
    ws = tmp_path / ".claude" / "jobs" / "trial" / "ws2"
    ws.mkdir(parents=True)
    with pytest.raises(RuntimeError, match=r"\.claude"):
        rv._assert_clean_cwd(str(ws))


def test_assert_clean_cwd_raises_inside_vcs_repo(tmp_path):
    (tmp_path / ".git").mkdir()
    ws = tmp_path / "sub" / "ws2"
    ws.mkdir(parents=True)
    with pytest.raises(RuntimeError, match=r"repo"):
        rv._assert_clean_cwd(str(ws))


def test_new_clean_workspace_dir_is_isolation_clean():
    work = rv._new_clean_workspace_dir("on", 0)
    try:
        rv._assert_clean_cwd(work)  # must not raise — it is under a clean temp base
        assert os.path.isdir(work)
    finally:
        import shutil as _sh
        _sh.rmtree(work, ignore_errors=True)


# --- _ingest_padding (Gate B FIX 4) ------------------------------------------------------

def test_ingest_padding_builds_markdown_ingest_command(tmp_path, monkeypatch):
    calls = []

    class _FakeResult:
        returncode = 0
        stdout = ""
        stderr = ""

    def fake_run(cmd, **kwargs):
        calls.append(cmd)
        return _FakeResult()

    monkeypatch.setattr(rv.subprocess, "run", fake_run)

    chunks_dir = str(tmp_path / "chunks")
    os.makedirs(chunks_dir)
    rv._ingest_padding(chunks_dir, {"ENGRAM_CHUNKS_DIR": chunks_dir}, 3)

    assert len(calls) == 1
    cmd = calls[0]
    assert cmd[:2] == ["engram", "ingest"]
    assert "--chunks-dir" in cmd and chunks_dir in cmd
    assert cmd.count("--markdown") == 3
    md_paths = [cmd[i + 1] for i, tok in enumerate(cmd) if tok == "--markdown"]
    assert len(md_paths) == 3
    for path in md_paths:
        assert os.path.exists(path)
