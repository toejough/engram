"""
Tests for audit_moments.py — semantic auditor for memory-loop moments.

Uses stubs for all claude -p calls to ensure zero real network activity.
"""

import json
import os
import pytest
import shutil
import subprocess
import sys
import tempfile
import time
import unittest.mock
from pathlib import Path

# Make extract importable
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

import audit_moments
from extract import extract_transcript


@pytest.fixture
def fixture_main_session():
    """Return path to the main_session_01.jsonl fixture."""
    return os.path.join(
        os.path.dirname(os.path.abspath(__file__)),
        "fixtures",
        "main",
        "main_session_01.jsonl",
    )


@pytest.fixture
def detect_stub():
    """Stub for detect_moments — returns canned candidates."""

    def stub(prompt: str, model: str) -> dict:
        # Simulate detection finding one moment: correction at line 7 (engram query result)
        candidates = [
            {
                "location": 7,
                "moment_type": "success",
                "description": "Query returned a relevant memory that was used",
            },
            {
                "location": 8,
                "moment_type": "dispatch",
                "description": "Agent dispatched work to a fresh subagent",
            },
        ]
        # Format response as claude -p would return it
        response_json = {"candidates": candidates}
        return {
            "result": json.dumps(response_json),
            "total_cost_usd": 0.001,
        }

    return stub


@pytest.fixture
def judge_stub():
    """Stub for judge_moment's LLM call — returns canned judgment fields."""

    def stub(prompt: str, model: str) -> dict:
        # Canned judgment response covering all scorecard fields
        judgment = {
            "search_targeted_right_thing": True,
            "outdated_outranked_replacement": False,
            "followed": "yes",
            "worth_learning_from": True,
            "note_well_targeted": True,
            "note_superseded_correctly": False,
            "note_not_duplicate": True,
            "strength_mismatch_flagged": False,
            "failure_category": None,
        }
        return {
            "result": json.dumps(judgment),
            "total_cost_usd": 0.002,
        }

    return stub


class TestWindowSplitting:
    """Test transcript windowing logic."""

    def test_small_file_single_window(self, fixture_main_session):
        """Files < 500 KB should produce a single window."""
        windows = audit_moments.split_into_windows(fixture_main_session)

        assert len(windows) == 1
        assert windows[0]["location_start"] == 1
        assert windows[0]["total_windows"] == 1
        assert windows[0]["window_index"] == 0
        assert windows[0]["transcript_path"] == fixture_main_session

    def test_window_preserves_lines_as_is(self, fixture_main_session):
        """Window lines should be complete, unmodified JSONL lines."""
        windows = audit_moments.split_into_windows(fixture_main_session)
        assert len(windows) > 0

        with open(fixture_main_session) as f:
            original_lines = f.readlines()

        # Single window should have all original lines
        if len(windows) == 1:
            assert windows[0]["lines"] == original_lines

    def test_window_has_required_fields(self, fixture_main_session):
        """Every window must have all required metadata fields."""
        windows = audit_moments.split_into_windows(fixture_main_session)
        required_fields = {
            "lines",
            "jsonl_lines",
            "location_start",
            "location_end",
            "transcript_path",
            "window_index",
            "total_windows",
        }

        for window in windows:
            assert set(window.keys()) >= required_fields
            assert window["location_start"] >= 1
            assert window["location_end"] >= window["location_start"]
            assert window["window_index"] >= 0
            assert window["total_windows"] >= 1

    def test_location_ranges_are_1_indexed(self, fixture_main_session):
        """location_start and location_end should be 1-indexed line numbers."""
        windows = audit_moments.split_into_windows(fixture_main_session)

        with open(fixture_main_session) as f:
            total_lines = len(f.readlines())

        # First window starts at 1
        assert windows[0]["location_start"] == 1
        # Last window ends at total line count
        assert windows[-1]["location_end"] == total_lines


class TestMomentDetection:
    """Test moment detection and judgment."""

    def test_detect_moments_with_stub(self, fixture_main_session, detect_stub):
        """detect_moments with stub should return candidates without real LLM call."""
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidates = audit_moments.detect_moments(window, model="haiku", stub=detect_stub)

        assert len(candidates) > 0
        for cand in candidates:
            assert "location" in cand
            assert "moment_type" in cand
            assert "description" in cand
            assert cand["moment_type"] in [
                "failure",
                "correction",
                "rework",
                "success",
                "dispatch",
            ]

    def test_judge_moment_with_stub(
        self, fixture_main_session, detect_stub, judge_stub
    ):
        """judge_moment with stub should return complete record without real LLM call."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        # Get a candidate
        candidates = audit_moments.detect_moments(
            window, model="haiku", stub=detect_stub
        )
        assert len(candidates) > 0

        candidate = candidates[0]
        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        # Verify it's a dict
        assert isinstance(record, dict)


class TestMomentSchema:
    """Test that moment records conform to the required schema."""

    def test_moment_record_has_all_required_fields(
        self, fixture_main_session, detect_stub, judge_stub
    ):
        """Every moment record must have all scorecard fields from design.md D-D."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidates = audit_moments.detect_moments(
            window, model="haiku", stub=detect_stub
        )
        assert len(candidates) > 0

        candidate = candidates[0]
        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        # Top-level fields (instrument v2 adds instrument_version + judgment_rationales)
        required_top = {
            "moment_id",
            "transcript_path",
            "location",
            "timestamp",
            "repo",
            "harness",
            "role",
            "moment_type",
            "description",
            "evidence",
            "finding_side",
            "writing_side",
            "handoff",
            "failure_category",
            "instrument_version",
            "judgment_rationales",
        }
        assert set(record.keys()) == required_top

        # finding_side fields (instrument v2 adds existence_top5 + relevance_grade;
        # <field>_null_reason keys appear only when a not_applicable verdict stamps them)
        required_finding = {
            "memory_existed",  # DEFERRED
            "memory_kind",  # DEFERRED
            "memory_generic_or_specific",  # DEFERRED
            "existence_derived_or_estimate",  # DEFERRED
            "existence_check_error",  # Bug 2: error tracking
            "existence_top5",
            "relevance_grade",
            "search_ran",
            "search_targeted_right_thing",
            "surfaced",
            "surfaced_rank",
            "outdated_outranked_replacement",
            "followed",
        }
        assert set(record["finding_side"].keys()) == required_finding

        # writing_side fields
        required_writing = {
            "worth_learning_from",
            "learn_fired",
            "note_written",
            "note_well_targeted",
            "note_superseded_correctly",
            "note_not_duplicate",
            "strength_updated",
            "strength_mismatch_flagged",
        }
        assert set(record["writing_side"].keys()) == required_writing

        # handoff fields
        required_handoff = {
            "memories_in_dispatch_prompt",
            "memories_orchestrator_had_but_left_out",
        }
        assert set(record["handoff"].keys()) == required_handoff

    def test_deferred_fields_are_null_without_search_phrases(
        self, fixture_main_session, detect_stub, judge_stub
    ):
        """When no search_phrases are generated, deferred fields must be explicitly null."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidates = audit_moments.detect_moments(
            window, model="haiku", stub=detect_stub
        )
        candidate = candidates[0]
        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        # Without search_phrases (judge_stub doesn't provide them), deferred fields stay null
        assert "memory_existed" in record["finding_side"]
        assert record["finding_side"]["memory_existed"] is None
        assert "memory_kind" in record["finding_side"]
        assert record["finding_side"]["memory_kind"] is None
        assert "memory_generic_or_specific" in record["finding_side"]
        assert record["finding_side"]["memory_generic_or_specific"] is None
        assert "existence_derived_or_estimate" in record["finding_side"]
        assert record["finding_side"]["existence_derived_or_estimate"] is None

        assert "memories_orchestrator_had_but_left_out" in record["handoff"]
        assert record["handoff"]["memories_orchestrator_had_but_left_out"] is None

    def test_moment_id_is_stable(
        self, fixture_main_session, detect_stub, judge_stub
    ):
        """moment_id should be deterministic and stable."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidates = audit_moments.detect_moments(
            window, model="haiku", stub=detect_stub
        )
        candidate = candidates[0]
        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        # moment_id should be <basename>#<location>
        basename = os.path.basename(fixture_main_session)
        expected_id = f"{basename}#{candidate['location']}"
        assert record["moment_id"] == expected_id


class TestFullPipeline:
    """Test end-to-end pipeline."""

    def test_full_audit_transcript_pipeline(
        self, fixture_main_session, detect_stub, judge_stub
    ):
        """Full pipeline should produce moment records without real LLM calls.

        Note: fixture_main_session (main_session_01.jsonl) is a schema-only smoke fixture (task 2.1) —
        it proves the pipeline's MECHANICS (windowing → detect → judge → schema → join) work end-to-end,
        not that the detection/judgment PROMPTS behave well against realistic transcript content.
        Prompt-quality calibration against real transcripts is task 3.3's job.
        """
        moments = audit_moments.audit_transcript(
            fixture_main_session, detect_stub=detect_stub, judge_stub=judge_stub
        )

        # Should have at least one moment
        assert len(moments) > 0

        # Each moment should be a valid dict
        for moment in moments:
            assert isinstance(moment, dict)
            # Can serialize to JSON
            json_str = json.dumps(moment)
            assert len(json_str) > 0

    def test_full_pipeline_no_subprocess_calls(
        self, fixture_main_session, detect_stub, judge_stub
    ):
        """When stubs are provided, no real subprocess should be invoked.

        Note: fixture_main_session (main_session_01.jsonl) is a schema-only smoke fixture (task 2.1) —
        it proves the pipeline's MECHANICS (windowing → detect → judge → schema → join) work end-to-end,
        not that the detection/judgment PROMPTS behave well against realistic transcript content.
        Prompt-quality calibration against real transcripts is task 3.3's job.
        """
        # Should complete without invoking subprocess
        moments = audit_moments.audit_transcript(
            fixture_main_session, detect_stub=detect_stub, judge_stub=judge_stub
        )

        # If we got here without exception, stubs were used
        assert len(moments) > 0

    def test_moments_are_json_serializable(
        self, fixture_main_session, detect_stub, judge_stub
    ):
        """All moment records should be JSON-serializable.

        Note: fixture_main_session (main_session_01.jsonl) is a schema-only smoke fixture (task 2.1) —
        it proves the pipeline's MECHANICS (windowing → detect → judge → schema → join) work end-to-end,
        not that the detection/judgment PROMPTS behave well against realistic transcript content.
        Prompt-quality calibration against real transcripts is task 3.3's job.
        """
        moments = audit_moments.audit_transcript(
            fixture_main_session, detect_stub=detect_stub, judge_stub=judge_stub
        )

        for moment in moments:
            # Should be serializable
            json_line = json.dumps(moment)
            # And deserializable
            deserialized = json.loads(json_line)
            assert isinstance(deserialized, dict)


class TestLowYieldRecheck:
    """Test the low-yield re-check logic."""

    def test_low_yield_recheck_triggered(self):
        """If haiku returns fewer than LOW_YIELD_THRESHOLD candidates, sonnet should re-check."""
        # Create a stub that returns different counts based on model
        def low_yield_stub(prompt: str, model: str) -> dict:
            if "haiku" in str(model):
                # Haiku returns only 1 candidate (below threshold)
                candidates = [
                    {
                        "location": 5,
                        "moment_type": "correction",
                        "description": "Minor fix",
                    }
                ]
            else:
                # Sonnet returns more
                candidates = [
                    {
                        "location": 5,
                        "moment_type": "correction",
                        "description": "Minor fix",
                    },
                    {
                        "location": 8,
                        "moment_type": "success",
                        "description": "Good work",
                    },
                    {
                        "location": 10,
                        "moment_type": "dispatch",
                        "description": "Sent to subagent",
                    },
                ]

            response_json = {"candidates": candidates}
            return {"result": json.dumps(response_json), "total_cost_usd": 0.001}

        # Create a minimal test window
        window = {
            "lines": ["line 1\n", "line 2\n", "line 3\n"],
            "location_start": 1,
            "location_end": 3,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

        # Create minimal extracted events
        extracted = {
            "recall_calls": [],
            "dispatch_events": [],
            "learn_calls": [],
            "engram_commands": [],
            "dispatch_type": "main",
            "transcript_path": "test.jsonl",
            "role": "main",
        }

        # Run the low_yield re-check logic via audit_transcript-like flow
        # Simulate what audit_transcript does for a single window
        candidates = audit_moments.detect_moments(window, model="haiku", stub=low_yield_stub)

        # Should start with haiku's 1 candidate
        assert len(candidates) == 1

        # Simulate low-yield re-check
        if len(candidates) < audit_moments.LOW_YIELD_THRESHOLD:
            sonnet_candidates = audit_moments.detect_moments(window, model="sonnet", stub=low_yield_stub)
            # Merge candidates
            candidates_by_loc = {c["location"]: c for c in candidates}
            for c in sonnet_candidates:
                if c["location"] not in candidates_by_loc:
                    candidates_by_loc[c["location"]] = c
            candidates = list(candidates_by_loc.values())

        # After re-check, should have merged haiku (1) + sonnet-only (2 more)
        assert len(candidates) == 3


class TestRecordConsistency:
    """Test that records are consistent with extracted events."""

    def test_record_reflects_extracted_events(
        self, fixture_main_session, detect_stub, judge_stub
    ):
        """Moment records should reflect the structural data from extract_transcript."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidates = audit_moments.detect_moments(
            window, model="haiku", stub=detect_stub
        )
        assert len(candidates) > 0

        candidate = candidates[0]
        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        # Check that structural fields are populated from extracted data
        assert record["transcript_path"] == fixture_main_session
        # repo should be derivable (or "unknown" for fixtures)
        assert "repo" in record
        # harness should be populated
        assert record["harness"] in ["claude_code", "pi"]
        # role should be populated
        assert record["role"] in ["main", "fork", "fresh", "workflow", "pi_main", "pi_subagent"]


class TestFixtureCompatibility:
    """Test compatibility with real fixture files."""

    def test_works_with_main_session_fixture(self, fixture_main_session):
        """audit_moments should work with real main session fixture."""
        # Just verify the file is readable and has expected structure
        with open(fixture_main_session) as f:
            lines = f.readlines()
        assert len(lines) > 0

    def test_audit_produces_records_from_fixture(self, fixture_main_session):
        """Full audit on fixture should produce records."""
        # Use stubs to avoid real network calls
        def simple_detect_stub(prompt: str, model: str) -> dict:
            candidates = [
                {
                    "location": 3,
                    "moment_type": "success",
                    "description": "Recall skill invoked",
                }
            ]
            return {"result": json.dumps({"candidates": candidates}), "total_cost_usd": 0.001}

        def simple_judge_stub(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        moments = audit_moments.audit_transcript(
            fixture_main_session,
            detect_stub=simple_detect_stub,
            judge_stub=simple_judge_stub,
        )
        assert len(moments) > 0


class TestMultiWindowSplitting:
    """Test windowing for large files (>=500KB)."""

    def test_multiwindow_splitting_produces_multiple_windows(self):
        """A synthetic >=500KB file should produce multiple windows."""
        # Create a temporary large JSONL file
        with tempfile.NamedTemporaryFile(mode='w', suffix='.jsonl', delete=False) as f:
            temp_path = f.name
            # Write many lines to exceed 500KB
            for i in range(400):
                line = {
                    "type": "user",
                    "timestamp": "2026-08-31T10:00:00Z",
                    "message": "x" * 1500,  # ~1.5KB per line
                }
                f.write(json.dumps(line) + "\n")

        try:
            windows = audit_moments.split_into_windows(temp_path)
            # Should have multiple windows
            assert len(windows) > 1
            # Each window (except possibly the last) should respect the 150KB cap
            for window in windows[:-1]:  # Skip last window for this check
                window_size_kb = sum(len(line.encode('utf-8')) for line in window["lines"]) / 1024
                assert window_size_kb <= 200  # Allow some slack
            # Windows should have proper metadata
            for window in windows:
                assert window["total_windows"] == len(windows)
                assert "oversized_line" in window
        finally:
            os.unlink(temp_path)

    def test_multiwindow_lines_not_split(self):
        """Lines should never be split across windows."""
        with tempfile.NamedTemporaryFile(mode='w', suffix='.jsonl', delete=False) as f:
            temp_path = f.name
            # Write lines to exceed 500KB
            for i in range(400):
                line = {
                    "type": "user",
                    "timestamp": "2026-08-31T10:00:00Z",
                    "data": "x" * 1500,
                }
                f.write(json.dumps(line) + "\n")

        try:
            windows = audit_moments.split_into_windows(temp_path)
            # Verify no line is split
            for window in windows:
                for line in window["lines"]:
                    # Each line should be complete (end with newline)
                    assert line.endswith("\n"), "Line was split"
        finally:
            os.unlink(temp_path)


class TestSubprocessIntegration:
    """Test subprocess invocation and JSON parsing."""

    def test_subprocess_not_called_with_stub(self, monkeypatch):
        """When stub is provided, subprocess.run should not be called."""
        call_tracker = {"called": False}

        def mock_run(*args, **kwargs):
            call_tracker["called"] = True
            raise AssertionError("subprocess.run should not be called with stub")

        monkeypatch.setattr(subprocess, "run", mock_run)

        window = {
            "lines": ["line 1\n"],
            "location_start": 1,
            "location_end": 1,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

        def stub(prompt: str, model: str) -> dict:
            return {"result": json.dumps({"candidates": []}), "total_cost_usd": 0.001}

        # Should complete without calling subprocess
        candidates = audit_moments.detect_moments(window, model="haiku", stub=stub)
        assert not call_tracker["called"], "subprocess.run was called despite stub being provided"

    def test_json_parsing_from_subprocess_response(self, monkeypatch):
        """_run_claude_p should correctly parse JSON from subprocess stdout."""
        def mock_run(*args, **kwargs):
            # Simulate real claude -p JSON response format
            response_obj = {
                "type": "message",
                "is_error": False,
                "result": "Test result",
                "total_cost_usd": 0.005,
                "num_turns": 1,
                "session_id": "test-session",
            }
            return subprocess.CompletedProcess(
                args=args,
                returncode=0,
                stdout=json.dumps(response_obj),
                stderr=""
            )

        monkeypatch.setattr(subprocess, "run", mock_run)

        result = audit_moments._run_claude_p("test prompt", "haiku")
        # Should have extracted result and cost
        assert result["result"] == "Test result"
        assert result["total_cost_usd"] == 0.005
        assert result["exhausted_retries"] is False

    def test_json_parsing_with_is_error_true(self, monkeypatch):
        """_run_claude_p should handle is_error=true in response."""
        call_count = {"count": 0}

        def mock_run(*args, **kwargs):
            call_count["count"] += 1
            if call_count["count"] <= 3:
                # First 3 calls: transient error (cheap)
                response_obj = {
                    "is_error": True,
                    "result": "Error",
                    "total_cost_usd": 0.001,
                }
            else:
                # 4th call: success
                response_obj = {
                    "is_error": False,
                    "result": "Success",
                    "total_cost_usd": 0.005,
                }
            return subprocess.CompletedProcess(
                args=args,
                returncode=0,
                stdout=json.dumps(response_obj),
                stderr=""
            )

        monkeypatch.setattr(subprocess, "run", mock_run)
        monkeypatch.setattr(time, "sleep", lambda *_: None)  # Mock time.sleep to avoid 180s delay

        # Should retry on transient errors
        result = audit_moments._run_claude_p("test prompt", "haiku")
        # After retries, should get the success response
        assert result["result"] == "Success"
        assert call_count["count"] >= 4
        assert result["exhausted_retries"] is False

    def test_json_parse_error_handling(self, monkeypatch):
        """_run_claude_p should handle malformed JSON from subprocess."""
        def mock_run(*args, **kwargs):
            # Return invalid JSON
            return subprocess.CompletedProcess(
                args=args,
                returncode=0,
                stdout="not valid json {",
                stderr=""
            )

        monkeypatch.setattr(subprocess, "run", mock_run)
        monkeypatch.setattr(time, "sleep", lambda *_: None)  # Mock time.sleep to avoid delay during retries

        # Should not crash, should return empty result -- and since every attempt in the
        # backoff sequence hit the same (treated-as-transient) malformed-JSON failure,
        # this is the exhausted-retries path, not a genuine "found nothing" response.
        result = audit_moments._run_claude_p("test prompt", "haiku")
        assert result["result"] == ""
        assert result["total_cost_usd"] == 0
        assert result["exhausted_retries"] is True


class TestOversizedLineDetection:
    """Test oversized line detection across multi-window splits (Finding 3)."""

    def test_oversized_line_detected_in_middle_window(self):
        """A >150KB single line in the middle of a multi-window file should be marked oversized_line=True."""
        with tempfile.NamedTemporaryFile(mode='w', suffix='.jsonl', delete=False) as f:
            temp_path = f.name
            # Write lines to exceed 500KB total
            # Window 1: normal lines (150KB+)
            for i in range(120):
                line = {
                    "type": "user",
                    "timestamp": "2026-08-31T10:00:00Z",
                    "message": "x" * 1500,  # ~1.5KB per line, 120 * ~1.5 = ~180KB
                }
                f.write(json.dumps(line) + "\n")

            # Middle: an oversized single line (>150KB)
            oversized = {
                "type": "assistant",
                "timestamp": "2026-08-31T10:00:05Z",
                "message": "y" * 160000,  # ~160KB
            }
            f.write(json.dumps(oversized) + "\n")

            # Window 3: normal lines (150KB+)
            for i in range(120):
                line = {
                    "type": "user",
                    "timestamp": "2026-08-31T10:00:10Z",
                    "message": "z" * 1500,
                }
                f.write(json.dumps(line) + "\n")

        try:
            windows = audit_moments.split_into_windows(temp_path)
            # Should have multiple windows
            assert len(windows) > 1

            # Find the window containing the oversized line (should be at position 121)
            oversized_location = 121
            oversized_window = None
            for window in windows:
                if window["location_start"] <= oversized_location <= window["location_end"]:
                    oversized_window = window
                    break

            assert oversized_window is not None, "Oversized line not found in any window"
            assert oversized_window["oversized_line"] is True, "Oversized window not marked correctly"

            # Adjacent windows should not be marked oversized
            for window in windows:
                if window != oversized_window and not (window["location_start"] <= oversized_location <= window["location_end"]):
                    assert window["oversized_line"] is False, f"Non-oversized window {window['window_index']} incorrectly marked oversized"
        finally:
            os.unlink(temp_path)


class TestInjectedMemoryFreshRole:
    """Test D-E injected memory detection for fresh/workflow/pi_subagent roles (Finding 1)."""

    def test_judge_moment_with_fresh_role_and_parent_transcript(self, fixture_main_session):
        """judge_moment should detect injected memory for fresh role when parent_transcript is provided."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        # Create a fresh-role moment
        candidate = {
            "location": 3,
            "moment_type": "success",
            "description": "Used injected memory",
        }

        # Create a parent transcript dict simulating memory injection
        parent_transcript = {
            "memories_in_dispatch_prompt": ["vault note 42: remember X", "vault note 43: remember Y"]
        }

        # Mock judge stub
        def judge_stub(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        # Set role to "fresh" in extracted events (override)
        extracted_with_fresh = extracted.copy()
        extracted_with_fresh["role"] = "fresh"

        record = audit_moments.judge_moment(
            candidate, window, extracted_with_fresh, model="sonnet",
            stub=judge_stub, parent_transcript=parent_transcript
        )

        # Should detect injected memory
        assert record["finding_side"]["search_ran"] == "injected", "Fresh role with parent_transcript should set search_ran='injected'"
        assert record["finding_side"]["surfaced"] is True, "Fresh role with parent_transcript should set surfaced=True"


FORK_CONTEXT_FIXTURE_BASE = os.path.join(
    os.path.dirname(os.path.abspath(__file__)), "fixtures", "fork_context"
)
FORK_CONTEXT_PARENT_SESSION_ID = "e7a1c9d2-5b3e-4f8a-9c21-6d4b8f2a1e33"


def _fork_context_fixture_path(*parts):
    return os.path.join(FORK_CONTEXT_FIXTURE_BASE, *parts)


@pytest.fixture
def fork_fake_vault(monkeypatch):
    """Redirect only the hardcoded '~/.local/share/engram/vault' lookup that
    _resolve_fork_parent_context() reads notes from, to a fixture vault dir --
    leaving every other os.path.expanduser call (e.g. the point-in-time check's own
    vault_repo_path) untouched. Keeps the production code's real, unparameterized
    vault path exactly as shipped, while making the test fully deterministic and
    never touching the operator's real vault (this file's established isolation
    discipline elsewhere -- see TestPointInTimeCheck's vault_fingerprint checks)."""
    fake_vault_dir = _fork_context_fixture_path("fake-vault")
    real_expanduser = os.path.expanduser

    def patched_expanduser(path):
        if path == "~/.local/share/engram/vault":
            return fake_vault_dir
        return real_expanduser(path)

    monkeypatch.setattr(os.path, "expanduser", patched_expanduser)
    return fake_vault_dir


class TestForkParentContextResolution:
    """Unit tests for _resolve_fork_parent_context(), the fix for judge_moment's D-E
    fork gap: a fork inherits its parent conversation's full context (design.md D-E),
    but judge_moment's LLM judgment call was never shown WHAT was recalled there --
    only this fork's own window text, which contains none of it. This resolves the
    fork's parent transcript + branch point and reads back the vault notes that were
    recalled just before the fork was dispatched.

    Fixtures live under fixtures/fork_context/: a parent transcript with three real
    recall_calls (one early decoy, one right before the fork point, one after it) and
    three fork transcripts exercising the main path plus edge cases (a)-(c). Edge case
    (d) -- a matched_item whose file is gone from the vault -- is folded into the main
    fixture itself (901.md deliberately doesn't exist in fake-vault/)."""

    def test_finds_last_qualifying_recall_before_fork_point(self, fork_fake_vault):
        fork_path = _fork_context_fixture_path(
            FORK_CONTEXT_PARENT_SESSION_ID, "subagents", "agent-testfork001.jsonl"
        )
        result = audit_moments._resolve_fork_parent_context(fork_path)

        assert result["available"] is True
        assert result["reason"] is None
        # Only 900 should come back: 901 (same recall_call) has no file in fake-vault
        # (edge case d), 899 belongs to an earlier recall_call that also qualifies but
        # isn't the LAST one before the bound, and 902 belongs to a recall_call after
        # the bound entirely.
        assert len(result["recalled_notes"]) == 1, result["recalled_notes"]
        note = result["recalled_notes"][0]
        assert note["path"] == "900.2026-09-01.test-planted-fixture-note.md"
        assert note["kind"] == "feedback"
        assert "exponential backoff with jitter" in note["content"]

    def test_early_bound_has_no_qualifying_recall(self, fork_fake_vault):
        """Edge case (c): every recall_call in the parent happened AFTER the fork's
        bound line -- recalled_notes must be empty, with a distinct reason from a
        resolution failure (available stays True; nothing went wrong, there just
        wasn't a recall yet)."""
        fork_path = _fork_context_fixture_path(
            FORK_CONTEXT_PARENT_SESSION_ID, "subagents", "agent-earlybound001.jsonl"
        )
        result = audit_moments._resolve_fork_parent_context(fork_path)

        assert result["available"] is True
        assert result["reason"] == "no recall before fork point"
        assert result["recalled_notes"] == []

    def test_missing_parent_transcript(self, fork_fake_vault):
        """Edge case (a): retention deleted the parent transcript independently of the
        fork's own transcript -- a real, observed case, not hypothetical."""
        fork_path = _fork_context_fixture_path(
            FORK_CONTEXT_PARENT_SESSION_ID, "subagents", "agent-missingparent.jsonl"
        )
        result = audit_moments._resolve_fork_parent_context(fork_path)

        assert result["available"] is False
        assert result["reason"] == "parent transcript not found"
        assert result["recalled_notes"] == []

    def test_not_a_fork_transcript(self, fixture_main_session):
        """Edge case (b): the transcript's first line isn't a fork-context-ref."""
        result = audit_moments._resolve_fork_parent_context(fixture_main_session)

        assert result["available"] is False
        assert result["reason"] == "not a fork transcript"
        assert result["recalled_notes"] == []

    def test_skips_missing_vault_note_but_keeps_others(self, fork_fake_vault):
        """Edge case (d): a matched_item (901) has no corresponding file left in the
        live vault -- must be skipped silently, without crashing, while the other
        resolvable note (900) from the same recall_call still comes back."""
        fork_path = _fork_context_fixture_path(
            FORK_CONTEXT_PARENT_SESSION_ID, "subagents", "agent-testfork001.jsonl"
        )
        result = audit_moments._resolve_fork_parent_context(fork_path)

        assert result["available"] is True
        paths = [n["path"] for n in result["recalled_notes"]]
        assert "901.2026-09-01.superseded-note-not-in-vault.md" not in paths
        assert "900.2026-09-01.test-planted-fixture-note.md" in paths


class TestForkJudgmentPromptInjection:
    """judge_moment() must inject the fork's resolved parent-recall context into its
    judgment prompt for role=='fork' moments when recalled_notes resolve, and must
    leave every other case's prompt exactly as it was before this fix (zero behavior
    change) -- non-fork roles, and fork moments where resolution finds nothing."""

    def test_fork_role_prompt_includes_inherited_recall_context(self, fork_fake_vault):
        fork_path = _fork_context_fixture_path(
            FORK_CONTEXT_PARENT_SESSION_ID, "subagents", "agent-testfork001.jsonl"
        )
        windows = audit_moments.split_into_windows(fork_path)
        window = windows[0]

        extracted = {
            "recall_calls": [],
            "dispatch_events": [],
            "learn_calls": [],
            "engram_commands": [],
            "dispatch_type": "fork",
            "transcript_path": fork_path,
            "role": "fork",
        }

        candidate = {
            "location": 6,
            "moment_type": "success",
            "description": "Forked subagent updated the retry configuration.",
        }

        captured_prompts = []

        def judge_stub(prompt: str, model: str) -> dict:
            captured_prompts.append(prompt)
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "partial",
                "worth_learning_from": True,
                "note_well_targeted": None,
                "note_superseded_correctly": None,
                "note_not_duplicate": None,
                "strength_mismatch_flagged": None,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        assert len(captured_prompts) == 1
        prompt = captured_prompts[0]

        # Exact-shape check: build the expected injected section byte-for-byte from
        # the real planted note content (read independently here, not hardcoded), and
        # confirm it appears verbatim in the assembled prompt.
        with open(os.path.join(fork_fake_vault, "900.2026-09-01.test-planted-fixture-note.md")) as f:
            note_900_content = f.read()

        expected_section = (
            "This is a forked subagent: it inherited its parent conversation's full "
            "context, including the following memory that had already been recalled "
            "and surfaced there before this fork was dispatched:\n\n"
            "--- 900.2026-09-01.test-planted-fixture-note.md (feedback) ---\n"
            f"{note_900_content}\n\n"
            "Use this inherited context (in addition to the window below, which "
            "shows only this fork's own actions) to judge whether it was followed.\n\n"
        )
        assert expected_section in prompt

        # The excluded recalls' content (early decoy, post-bound) must not leak in.
        assert "899.2026-08-30.early-decoy-note.md" not in prompt
        assert "902.2026-09-01.after-bound-note-should-not-appear.md" not in prompt

        # The structural D-E fields are untouched by this fix.
        assert record["finding_side"]["search_ran"] == "injected"
        assert record["finding_side"]["surfaced"] is True

    def test_non_fork_role_prompt_has_no_fork_context_section(
        self, fixture_main_session, detect_stub
    ):
        """Regression: a main-role moment's prompt must be unchanged from before this
        fix -- no fork-context section appears when role != 'fork'."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidates = audit_moments.detect_moments(window, model="haiku", stub=detect_stub)
        assert len(candidates) > 0
        candidate = candidates[0]

        captured_prompts = []

        def judge_stub(prompt: str, model: str) -> dict:
            captured_prompts.append(prompt)
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        assert len(captured_prompts) == 1
        prompt = captured_prompts[0]
        assert "forked subagent" not in prompt
        assert "already been recalled and surfaced there" not in prompt

    def test_fork_role_without_resolvable_context_has_no_dangling_section(self):
        """When role=='fork' but _resolve_fork_parent_context finds nothing to inject
        (here: parent transcript retention-deleted), judgment_prompt must fall back to
        the exact unmodified template -- no empty/dangling section marker, and the
        structural D-E fields still populate independently of content resolution."""
        fork_path = _fork_context_fixture_path(
            FORK_CONTEXT_PARENT_SESSION_ID, "subagents", "agent-missingparent.jsonl"
        )
        windows = audit_moments.split_into_windows(fork_path)
        window = windows[0]

        extracted = {
            "recall_calls": [],
            "dispatch_events": [],
            "learn_calls": [],
            "engram_commands": [],
            "dispatch_type": "fork",
            "transcript_path": fork_path,
            "role": "fork",
        }

        candidate = {
            "location": 1,
            "moment_type": "dispatch",
            "description": "Fork dispatched with a since-deleted parent transcript.",
        }

        captured_prompts = []

        def judge_stub(prompt: str, model: str) -> dict:
            captured_prompts.append(prompt)
            judgment = {
                "search_targeted_right_thing": None,
                "outdated_outranked_replacement": None,
                "followed": None,
                "worth_learning_from": False,
                "note_well_targeted": None,
                "note_superseded_correctly": None,
                "note_not_duplicate": None,
                "strength_mismatch_flagged": None,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        assert len(captured_prompts) == 1
        prompt = captured_prompts[0]
        assert "forked subagent" not in prompt
        assert record["finding_side"]["search_ran"] == "injected"
        assert record["finding_side"]["surfaced"] is True


class TestSubprocessBypassDetection:
    """Test stub-bypass regression prevention (Finding 4)."""

    def test_subprocess_not_called_in_judge_moment(self, monkeypatch):
        """When judge_stub is provided, no real subprocess should be called from judge_moment."""
        call_tracker = {"called": False}

        def mock_run(*args, **kwargs):
            call_tracker["called"] = True
            raise AssertionError("subprocess.run should not be called in judge_moment with stub")

        monkeypatch.setattr(subprocess, "run", mock_run)

        # Prepare test data
        window = {
            "lines": ["line 1\n"],
            "location_start": 1,
            "location_end": 1,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

        candidate = {
            "location": 1,
            "moment_type": "success",
            "description": "Test moment",
        }

        extracted = {
            "recall_calls": [],
            "dispatch_events": [],
            "learn_calls": [],
            "engram_commands": [],
            "dispatch_type": "main",
            "transcript_path": "test.jsonl",
            "role": "main",
        }

        def judge_stub(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        # Should complete without calling subprocess
        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )
        assert not call_tracker["called"], "subprocess.run was called despite stub being provided"
        assert isinstance(record, dict)

    def test_subprocess_not_called_in_full_pipeline(self, monkeypatch, fixture_main_session):
        """When both detect_stub and judge_stub are provided, no real subprocess should be called in audit_transcript."""
        call_tracker = {"called": False}

        def mock_run(*args, **kwargs):
            call_tracker["called"] = True
            raise AssertionError("subprocess.run should not be called in full pipeline with stubs")

        monkeypatch.setattr(subprocess, "run", mock_run)

        def detect_stub(prompt: str, model: str) -> dict:
            candidates = [
                {
                    "location": 3,
                    "moment_type": "success",
                    "description": "Test detection",
                }
            ]
            return {"result": json.dumps({"candidates": candidates}), "total_cost_usd": 0.001}

        def judge_stub(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        # Should complete without calling subprocess
        moments = audit_moments.audit_transcript(
            fixture_main_session, detect_stub=detect_stub, judge_stub=judge_stub
        )
        assert not call_tracker["called"], "subprocess.run was called despite stubs being provided"
        assert len(moments) > 0


class TestTimestampExtraction:
    """Test timestamp extraction from transcript lines."""

    def test_timestamp_extracted_correctly(self):
        """_extract_timestamp_from_window should extract timestamp from JSON line."""
        window = {
            "lines": [
                '{"timestamp":"2026-08-31T10:00:00Z","type":"user"}\n',
                '{"timestamp":"2026-08-31T10:00:05Z","type":"assistant"}\n',
            ],
            "location_start": 1,
            "location_end": 2,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

        # Extract timestamp at location 1
        ts = audit_moments._extract_timestamp_from_window(window, 1)
        assert ts == "2026-08-31T10:00:00Z"

        # Extract timestamp at location 2
        ts = audit_moments._extract_timestamp_from_window(window, 2)
        assert ts == "2026-08-31T10:00:05Z"

    def test_timestamp_returned_in_moment_record(self, fixture_main_session):
        """Moment records should include extracted timestamp."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        def detect_stub(prompt: str, model: str) -> dict:
            candidates = [{"location": 1, "moment_type": "success", "description": "Test"}]
            return {"result": json.dumps({"candidates": candidates}), "total_cost_usd": 0.001}

        def judge_stub(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        candidates = audit_moments.detect_moments(window, model="haiku", stub=detect_stub)
        candidate = candidates[0]
        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        # Timestamp should be populated (not None)
        assert record["timestamp"] is not None


class TestPointInTimeCheck:
    """Test the point-in-time memory existence check (task 3.2)."""

    def test_sandboxing_empty_vault_returns_zero_results(self):
        """Prove sandboxing works: an empty scratch vault returns zero results from engram query."""
        sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
        import isolation
        from extract import parse_engram_query_result

        # Create an isolated environment with empty vault and chunks
        scratch_root = tempfile.mkdtemp(prefix="test-sandbox-")
        try:
            vault = os.path.join(scratch_root, "vault")
            chunks = os.path.join(scratch_root, "chunks")
            os.makedirs(vault, exist_ok=True)
            os.makedirs(chunks, exist_ok=True)

            # Build isolated env
            env = isolation.engram_env(vault=vault, chunks=chunks)

            # Confirm isolation
            isolation.assert_engram_isolated(env)

            # Run engram query with test phrases
            result = subprocess.run(
                [
                    "engram",
                    "query",
                    "--lazy-chunks",
                    "--phrase",
                    "test memory retrieval",
                ],
                env=env,
                capture_output=True,
                text=True,
                timeout=30,
            )

            # The query should complete without error
            assert result.returncode == 0, f"engram query failed: {result.stderr}"

            # Parse output and verify zero items
            output = result.stdout.strip()
            items = parse_engram_query_result(output)

            # For an empty vault, the query should return zero items
            assert len(items) == 0, f"expected 0 items from empty vault, got {len(items)}: {items}"

        finally:
            shutil.rmtree(scratch_root, ignore_errors=True)

    def test_live_vault_unchanged_after_ptc_check(self):
        """Prove the real vault is not modified by the point-in-time check.

        This is a real trial using actual vault data, but the check only runs read-only
        operations (git log, git worktree add/remove, engram query against scratch copies).
        """
        sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), ".."))
        import isolation

        vault_path = os.path.expanduser("~/.local/share/engram/vault")

        # Skip if vault doesn't exist (not on the host)
        if not os.path.exists(vault_path):
            pytest.skip("vault not found at ~/.local/share/engram/vault")

        # Capture fingerprint before
        before = isolation.vault_fingerprint(vault_path)

        try:
            # Run a minimal point-in-time check:
            # - Find a commit from 3 days ago (a safe, recent time)
            # - Check it's after the migration
            # - Query the vault at that time

            # Use a timestamp from 3 days ago
            from datetime import datetime, timedelta, timezone

            three_days_ago = (datetime.now(timezone.utc) - timedelta(days=3)).isoformat()

            # Find commit at or before that time
            commit_sha = audit_moments._find_commit_at_or_before(vault_path, three_days_ago)

            if commit_sha:
                # Check if it's after the migration
                is_after = audit_moments._is_commit_ancestor_of(
                    vault_path, audit_moments.FLAT_VAULT_MIGRATION_COMMIT, commit_sha
                )

                if is_after:
                    # Do a full point-in-time check
                    scratch_dir = tempfile.mkdtemp(prefix="test-vault-check-")
                    try:
                        with audit_moments._checkout_vault_at(vault_path, commit_sha, scratch_dir) as worktree:
                            # Verify the worktree was created
                            assert os.path.exists(worktree), "worktree not created"

                            # Query it with a test phrase
                            chunks_path = os.path.join(scratch_dir, "chunks-test")
                            os.makedirs(chunks_path, exist_ok=True)
                            result = audit_moments._run_engram_query_at_moment(
                                worktree, chunks_path, ["test query"]
                            )

                            # The query should complete (result is a dict, possibly with empty items)
                            assert isinstance(result, dict)

                    finally:
                        shutil.rmtree(scratch_dir, ignore_errors=True)

        finally:
            # Verify vault unchanged
            after = isolation.vault_fingerprint(vault_path)
            assert before == after, f"vault was modified: {before} -> {after}"

    def test_worktree_cleanup_on_engram_query_failure(self, monkeypatch):
        """Prove worktrees are cleaned up even when git worktree remove fails."""
        vault_path = os.path.expanduser("~/.local/share/engram/vault")

        # Skip if vault doesn't exist
        if not os.path.exists(vault_path):
            pytest.skip("vault not found")

        # Find a recent commit. `git rev-parse HEAD` is read-only/non-mutating
        # (confirmed by prior review) -- it only resolves a ref to a SHA.
        result = subprocess.run(
            ["git", "-C", vault_path, "rev-parse", "HEAD"],
            capture_output=True,
            text=True,
            timeout=10,
        )

        if result.returncode != 0:
            pytest.skip("cannot get HEAD commit")

        commit_sha = result.stdout.strip()[:7]
        scratch_dir = tempfile.mkdtemp(prefix="test-worktree-cleanup-")

        try:
            # Inject a failure in git worktree remove by making subprocess.run return non-zero
            # for the remove call specifically
            original_run = subprocess.run
            remove_call_count = {"count": 0}

            def mock_run_with_failure(*args, **kwargs):
                # Check if this is a worktree remove call
                if len(args) > 0 and isinstance(args[0], list):
                    if "worktree" in args[0] and "remove" in args[0]:
                        remove_call_count["count"] += 1
                        # First time (the one in _checkout_vault_at): simulate failure
                        if remove_call_count["count"] == 1:
                            return subprocess.CompletedProcess(
                                args=args[0],
                                returncode=128,  # Non-zero to trigger prune fallback
                                stdout="",
                                stderr="worktree remove failed (simulated)"
                            )
                # For everything else, use the real subprocess.run
                return original_run(*args, **kwargs)

            monkeypatch.setattr(subprocess, "run", mock_run_with_failure)

            # Create a worktree (with remove failure injection). Do NOT wrap this in a
            # bare try/except -- a swallowed AssertionError here previously hid a real
            # leaked-worktree bug (Finding 1+2+5): the assertion below must be able to
            # actually fail the test.
            with audit_moments._checkout_vault_at(vault_path, commit_sha, scratch_dir) as wt:
                assert os.path.exists(wt), "worktree not created"

            # After context manager exit, worktree should be cleaned up via the
            # force-delete + prune fallback (git worktree prune alone is a no-op while
            # the directory still exists on disk, which it does right after a failed
            # `git worktree remove`).
            assert not os.path.exists(wt), "worktree not cleaned up even with force-delete + prune fallback"

            # Verify the prune fallback was triggered
            assert remove_call_count["count"] > 0, "worktree remove was never called"

            # And verify no dangling registration was left in the real vault's
            # `.git/worktrees` -- only the main worktree should remain.
            list_result = subprocess.run(
                ["git", "-C", vault_path, "worktree", "list"],
                capture_output=True,
                text=True,
                timeout=10,
            )
            worktree_lines = [line for line in list_result.stdout.splitlines() if line.strip()]
            assert len(worktree_lines) == 1, (
                f"expected only the main worktree registered, found:\n{list_result.stdout}"
            )

        finally:
            shutil.rmtree(scratch_dir, ignore_errors=True)

    def test_filter_chunks_by_ingestion_date_excludes_legacy(self):
        """Test that chunk filtering excludes legacy records (no ingested_at)."""
        # Create a test chunk index with mixed records
        scratch_dir = tempfile.mkdtemp(prefix="test-chunk-filter-")

        try:
            chunks_dir = os.path.join(scratch_dir, "chunks-in")
            os.makedirs(chunks_dir, exist_ok=True)

            # Write a test chunk file with mixed records
            test_index = os.path.join(chunks_dir, "test.jsonl")
            with open(test_index, "w") as f:
                # Legacy record (no ingested_at)
                f.write(
                    json.dumps(
                        {
                            "source": "test.md",
                            "anchor": "1",
                            "content_hash": "abc",
                            "text": "legacy content",
                            "vector": [],
                        }
                    )
                    + "\n"
                )

                # Recent record (after cutoff)
                f.write(
                    json.dumps(
                        {
                            "source": "test.md",
                            "anchor": "2",
                            "content_hash": "def",
                            "text": "future content",
                            "vector": [],
                            "ingested_at": "2099-01-01T00:00:00Z",
                        }
                    )
                    + "\n"
                )

                # Old record (before cutoff)
                f.write(
                    json.dumps(
                        {
                            "source": "test.md",
                            "anchor": "3",
                            "content_hash": "ghi",
                            "text": "old content",
                            "vector": [],
                            "ingested_at": "2026-06-01T00:00:00Z",
                        }
                    )
                    + "\n"
                )

            # Filter with cutoff at 2026-08-01
            filtered_dir = audit_moments._filter_chunks_by_ingestion_date(
                chunks_dir, "2026-08-01T00:00:00Z", scratch_dir
            )

            # Check filtered output
            filtered_index = os.path.join(filtered_dir, "test.jsonl")
            assert os.path.exists(filtered_index)

            with open(filtered_index, "r") as f:
                lines = [l.strip() for l in f.readlines() if l.strip()]

            # Should have only the old record (2026-06-01 <= 2026-08-01)
            # Legacy and future records should be excluded
            assert len(lines) == 1

            record = json.loads(lines[0])
            assert record["text"] == "old content"

        finally:
            shutil.rmtree(scratch_dir, ignore_errors=True)

    def test_point_in_time_check_integration_with_search_phrases(self, fixture_main_session):
        """Test that judge_moment actually populates memory_existed/kind when search_phrases are present."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        # Create a candidate with the required structure
        candidate = {
            "location": 3,
            "moment_type": "success",
            "description": "Test moment for PTC",
        }

        # Create a judge_stub that DOES return search_phrases (unlike the default judge_stub)
        def judge_stub_with_phrases(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
                # KEY: Include search_phrases to trigger the point-in-time check
                "search_phrases": ["test phrase one", "test phrase two"],
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        # Get a timestamp from the window (use a recent date for ESTIMATE testing)
        timestamp = "2026-08-01T12:00:00Z"

        # Create a minimal extracted events dict with the timestamp
        extracted_with_timestamp = extracted.copy()

        # Create a window with a line that has our timestamp
        window_with_timestamp = window.copy()

        # Mock the timestamp extraction to return our test timestamp
        import unittest.mock
        with unittest.mock.patch.object(
            audit_moments, "_extract_timestamp_from_window", return_value=timestamp
        ):
            record = audit_moments.judge_moment(
                candidate, window_with_timestamp, extracted_with_timestamp,
                model="sonnet", stub=judge_stub_with_phrases
            )

        # Verify the fields are populated (not None)
        assert record["finding_side"]["memory_existed"] is not None, "memory_existed should be populated when search_phrases present"
        # memory_kind might be None for ESTIMATE if no notes matched, but existence_derived_or_estimate should be set
        assert record["finding_side"]["existence_derived_or_estimate"] is not None, "existence_derived_or_estimate should be set"
        # Should be either "derived" or "estimate"
        assert record["finding_side"]["existence_derived_or_estimate"] in ["derived", "estimate"], \
            f"unexpected existence_derived_or_estimate: {record['finding_side']['existence_derived_or_estimate']}"

    def test_derived_vs_estimate_boundary(self):
        """Test DERIVED/ESTIMATE determination at the vault migration boundary."""
        vault_path = os.path.expanduser("~/.local/share/engram/vault")

        # Skip if vault doesn't exist
        if not os.path.exists(vault_path):
            pytest.skip("vault not found")

        # Test 1: A timestamp well after migration should be DERIVED
        # Use June 15, 2026 which is well after the June 12 migration
        after_migration = "2026-06-15T12:00:00Z"
        commit_sha = audit_moments._find_commit_at_or_before(vault_path, after_migration)

        assert commit_sha, "should find at least one commit before 2026-06-15"

        is_derived = audit_moments._is_commit_ancestor_of(
            vault_path, audit_moments.FLAT_VAULT_MIGRATION_COMMIT, commit_sha
        )
        # If we found a commit after migration, it should pass the ancestry check
        assert is_derived, f"post-migration commit {commit_sha} should be DERIVED"

        # Test 2: A timestamp before migration should be ESTIMATE
        before_migration = "2026-06-01T00:00:00Z"
        commit_sha = audit_moments._find_commit_at_or_before(vault_path, before_migration)

        if commit_sha:
            is_derived = audit_moments._is_commit_ancestor_of(
                vault_path, audit_moments.FLAT_VAULT_MIGRATION_COMMIT, commit_sha
            )
            # If we found a commit before migration, it should fail the ancestry check
            assert not is_derived, "pre-migration commit incorrectly marked as DERIVED"

    def test_estimate_branch_real_pre_migration_timestamp_content_derived(
        self, fixture_main_session
    ):
        """
        Finding 8: exercise the real ESTIMATE existence-check branch end-to-end.

        The two pre-existing "ESTIMATE" tests both failed to actually exercise it: one
        used a timestamp so early that commit_sha came back None (its own `if
        commit_sha:` guard never fired, making the assertion vacuous); the other
        resolved to a commit AFTER the flat-vault migration despite being labeled
        ESTIMATE, so it silently exercised the DERIVED branch instead.

        This test uses 2026-06-04T12:00:00Z, which resolves to commit bf3dfe6
        (2026-06-03T17:23:28-04:00, confirmed present in the real vault's git history)
        -- a real, non-None commit_sha that is confirmed pre-migration
        (FLAT_VAULT_MIGRATION_COMMIT = 4d8860e, 2026-06-12), so the DERIVED/ESTIMATE
        decision correctly lands on ESTIMATE.

        It also proves Finding 6+7's fixes: with real vault content, memory_kind and
        memory_generic_or_specific come back content-derived, not the old hardcoded
        "feedback"/"generic" constants.
        """
        vault_path = os.path.expanduser("~/.local/share/engram/vault")
        if not os.path.exists(vault_path):
            pytest.skip("vault not found")

        timestamp = "2026-06-04T12:00:00Z"

        # (a) Confirm the DERIVED/ESTIMATE boundary logic actually resolves to
        # ESTIMATE for this timestamp -- i.e. commit_sha is real (not None) and
        # pre-migration.
        commit_sha = audit_moments._find_commit_at_or_before(vault_path, timestamp)
        assert commit_sha, "expected a real commit at or before 2026-06-04T12:00:00Z"
        is_derived = audit_moments._is_commit_ancestor_of(
            vault_path, audit_moments.FLAT_VAULT_MIGRATION_COMMIT, commit_sha
        )
        assert not is_derived, f"commit {commit_sha} should be pre-migration (ESTIMATE), not DERIVED"

        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidate = {
            "location": 3,
            "moment_type": "success",
            "description": "Test moment for ESTIMATE existence check",
        }

        # These phrases are verbatim substrings of a real vault note
        # (qa.2026-07-03.are-answers-returned-with-questions.q.md, type: qa-question)
        # that has NO created:/timestamp: frontmatter field, so
        # _filter_notes_by_creation_date always includes it (its documented
        # conservative fallback for notes with no parseable creation date) regardless
        # of the timestamp used here. Both phrases were confirmed (grep -ril) to be
        # unique to that single file across the whole vault, so the match is
        # deterministic regardless of os.listdir iteration order.
        def judge_stub_with_phrases(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
                "search_phrases": [
                    "returning answers with the questions",
                    "relevant content",
                ],
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        with unittest.mock.patch.object(
            audit_moments, "_extract_timestamp_from_window", return_value=timestamp
        ):
            record = audit_moments.judge_moment(
                candidate, window, extracted, model="sonnet", stub=judge_stub_with_phrases
            )

        finding = record["finding_side"]
        # (b) DERIVED/ESTIMATE label
        assert finding["existence_derived_or_estimate"] == "estimate", (
            f"expected ESTIMATE for pre-migration timestamp, got {finding['existence_derived_or_estimate']}"
        )
        assert finding["memory_existed"] is True, "expected the known QA note to be found"
        # (b) content-derived memory_kind/memory_generic_or_specific, not the old
        # hardcoded "feedback"/"generic" constants.
        assert finding["memory_kind"] == "qa", (
            f"expected content-derived kind 'qa' (from type: qa-question), got {finding['memory_kind']}"
        )
        # Both search phrases matched (match_ratio == 1.0 >= 0.5) => "specific".
        assert finding["memory_generic_or_specific"] == "specific", (
            f"expected content-derived 'specific', got {finding['memory_generic_or_specific']}"
        )


class TestLineNumberingFix:
    """Test that line-numbering prefixes prevent location drift (task 5.1 bug fix).

    The bug: detect_moments() and judge_moment() built prompts by concatenating raw
    window lines WITHOUT line numbers, asking the model to self-report a 1-indexed line
    number by self-counting. The model's count drifted as windows got longer/denser,
    landing moments on unrelated content (e.g., line 26 reported for content at line 52).

    The fix: Prefix each line with its real absolute line number in the prompt, and
    instruct the model to report the EXACT line number shown, not to count lines itself.
    """

    def test_detect_moments_with_line_numbering_synthetic_window(self):
        """Verify detect_moments builds prompts with explicit line numbers."""
        # Create a synthetic window of 45 lines (dense, JSON-like)
        lines = []
        for i in range(45):
            # Simulate transcript JSON lines
            line_obj = {
                "type": "user" if i % 2 == 0 else "assistant",
                "timestamp": f"2026-08-31T10:{i:02d}:00Z",
                "message": f"message line {i+1}",
            }
            lines.append(json.dumps(line_obj) + "\n")

        window = {
            "lines": lines,
            "location_start": 100,  # Absolute line numbers (simulating middle of a large transcript)
            "location_end": 144,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

        # Create a stub that captures the prompt to inspect it
        captured_prompts = []

        def capture_stub(prompt: str, model: str) -> dict:
            captured_prompts.append(prompt)
            # Return a moment at line 125 (absolute) which is offset 25 in this window
            candidates = [
                {
                    "location": 125,
                    "moment_type": "success",
                    "description": "Test moment",
                }
            ]
            return {"result": json.dumps({"candidates": candidates}), "total_cost_usd": 0.001}

        # Call detect_moments
        candidates = audit_moments.detect_moments(window, model="haiku", stub=capture_stub)

        # Verify the prompt contains explicit line numbers
        assert len(captured_prompts) > 0, "stub should have been called at least once"
        prompt = captured_prompts[0]

        # Prompt should contain line numbers (e.g., "100: ", "101: ", etc.)
        assert "100:" in prompt, "prompt should contain absolute line number 100"
        assert "125:" in prompt, "prompt should contain absolute line number 125"
        assert "144:" in prompt, "prompt should contain absolute line number 144"

        # Prompt should instruct NOT to self-count
        assert "Do NOT count lines yourself" in prompt or "report the EXACT line number shown" in prompt, \
            "prompt should instruct model not to self-count"

        # Verify the candidate is returned correctly
        assert len(candidates) == 1
        assert candidates[0]["location"] == 125, "location should be the absolute line number reported"

    def test_judge_moment_with_line_numbering_synthetic_window(self):
        """Verify judge_moment builds prompts with explicit line numbers."""
        # Create a synthetic window of 50 lines
        lines = []
        for i in range(50):
            line_obj = {
                "type": "user" if i % 2 == 0 else "assistant",
                "timestamp": f"2026-08-31T11:{i%60:02d}:00Z",
                "message": f"content {i+1}",
            }
            lines.append(json.dumps(line_obj) + "\n")

        window = {
            "lines": lines,
            "location_start": 200,  # Absolute line numbers
            "location_end": 249,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

        candidate = {
            "location": 225,
            "moment_type": "success",
            "description": "Test moment",
        }

        extracted = {
            "recall_calls": [],
            "dispatch_events": [],
            "learn_calls": [],
            "engram_commands": [],
            "dispatch_type": "main",
            "transcript_path": "test.jsonl",
            "role": "main",
        }

        # Capture prompts
        captured_prompts = []

        def judge_stub(prompt: str, model: str) -> dict:
            captured_prompts.append(prompt)
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        record = audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        # Verify the prompt contains explicit line numbers
        assert len(captured_prompts) > 0, "judge stub should have been called"
        prompt = captured_prompts[0]

        # Prompt should contain absolute line numbers
        assert "200:" in prompt, "prompt should contain absolute line number 200"
        assert "225:" in prompt, "prompt should contain absolute line number 225"
        assert "249:" in prompt, "prompt should contain absolute line number 249"

        # Verify record was created successfully
        assert isinstance(record, dict)
        assert record["location"] == 225

    def test_line_numbering_round_trip_with_absolute_offsets(self):
        """Verify that absolute line numbers in prompts enable correct location round-tripping.

        This is the core test: if the prompt has explicit line numbers, the model should
        report a location that directly maps to the transcript, not requiring any offset
        arithmetic or re-mapping.
        """
        # Create a window that is NOT at line 1 (i.e., location_start > 1)
        # to ensure we're testing absolute line numbering, not window-relative numbering
        lines = []
        for i in range(40):
            line_obj = {
                "type": "user" if i % 3 == 0 else "assistant",
                "timestamp": f"2026-08-31T12:{i:02d}:00Z",
                "content": f"line {i}",
            }
            lines.append(json.dumps(line_obj) + "\n")

        # Window starts at line 500 (a large offset)
        window = {
            "lines": lines,
            "location_start": 500,
            "location_end": 539,
            "transcript_path": "test.jsonl",
            "window_index": 5,
            "total_windows": 10,
            "oversized_line": False,
        }

        # Create a stub that extracts and validates the prompt
        def validate_stub(prompt: str, model: str) -> dict:
            # Parse the prompt to verify it has absolute line numbers
            # Look for "500:", "520:", "539:", etc.
            for line_num in [500, 520, 539]:
                expected_marker = f"{line_num}:"
                assert expected_marker in prompt, f"prompt should have absolute line number {line_num}"

            # Return a candidate at line 520 (absolute)
            # This is the KEY: the model returns 520, which should be interpreted as
            # absolute, not relative to window start (which would be 20)
            candidates = [
                {
                    "location": 520,
                    "moment_type": "failure",
                    "description": "error at line 520",
                }
            ]
            return {"result": json.dumps({"candidates": candidates}), "total_cost_usd": 0.001}

        candidates = audit_moments.detect_moments(window, model="haiku", stub=validate_stub)

        # The location should be 520 (absolute), not 20 (offset within window)
        assert len(candidates) == 1
        assert candidates[0]["location"] == 520, \
            f"location should be absolute line number 520, not relative offset; got {candidates[0]['location']}"


class TestEngramQueryErrorHandling:
    """
    Bug 2 fix: explicit error handling when engram query subprocess fails.

    Previously, a failed engram query (non-zero return code) was silently recorded as
    "memory_existed: False", indistinguishable from a genuine "nothing existed" result.

    The fix: _run_engram_query_at_moment() returns items=None (not []) when the check
    fails, and judge_moment() sets memory_existed=None (not False) with existence_check_error
    surfaced in the record.
    """

    def test_run_engram_query_subprocess_failure(self):
        """_run_engram_query_at_moment should return error indicator on subprocess failure."""
        # Mock subprocess.run to return failure
        with unittest.mock.patch("subprocess.run") as mock_run:
            mock_run.return_value = unittest.mock.Mock(
                returncode=1,
                stdout="",
                stderr="vault not found",
            )

            result = audit_moments._run_engram_query_at_moment(
                "/tmp/vault", "/tmp/chunks", ["test phrase"]
            )

            # Bug 2: items should be None (not []), error should be set
            assert result.get("items") is None, "items should be None on subprocess failure"
            assert result.get("error") is not None, "error should be set on subprocess failure"
            assert "rc 1" in result.get("error", ""), "error should mention return code"

    def test_run_engram_query_timeout(self):
        """_run_engram_query_at_moment should return error indicator on timeout."""
        with unittest.mock.patch("subprocess.run") as mock_run:
            mock_run.side_effect = subprocess.TimeoutExpired("engram", 60)

            result = audit_moments._run_engram_query_at_moment(
                "/tmp/vault", "/tmp/chunks", ["test phrase"]
            )

            # Bug 2: items should be None, error should indicate timeout
            assert result.get("items") is None, "items should be None on timeout"
            assert "timed out" in result.get("error", "").lower(), "error should mention timeout"

    def test_run_engram_query_success_no_results(self):
        """_run_engram_query_at_moment should return empty items list on successful check with no results."""
        with unittest.mock.patch("subprocess.run") as mock_run:
            mock_run.return_value = unittest.mock.Mock(
                returncode=0,
                stdout="version: 1\nphrases:\n  - test phrase\nitems:\nclusters: []",
                stderr="",
            )

            result = audit_moments._run_engram_query_at_moment(
                "/tmp/vault", "/tmp/chunks", ["test phrase"]
            )

            # Successful check with no results: items is [], error is None
            assert isinstance(result.get("items"), list), "items should be a list on success"
            assert len(result.get("items", [])) == 0, "items should be empty list when no results"
            assert result.get("error") is None, "error should be None on success"

    def test_judge_moment_with_engram_query_error(
        self, fixture_main_session, detect_stub
    ):
        """judge_moment should set memory_existed=None and existence_check_error when query fails."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidates = audit_moments.detect_moments(
            window, model="haiku", stub=detect_stub
        )
        assert len(candidates) > 0
        candidate = candidates[0]

        # Stub that returns search_phrases so the point-in-time check runs
        def judge_stub_with_phrases(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
                "search_phrases": ["test query"],
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        # Mock _run_engram_query_at_moment to return an error
        with unittest.mock.patch.object(
            audit_moments, "_run_engram_query_at_moment"
        ) as mock_query:
            mock_query.return_value = {"items": None, "error": "vault checkout failed"}

            with unittest.mock.patch.object(
                audit_moments, "_extract_timestamp_from_window", return_value="2026-08-31T10:00:00Z"
            ):
                record = audit_moments.judge_moment(
                    candidate, window, extracted, model="sonnet", stub=judge_stub_with_phrases
                )

        finding = record["finding_side"]
        # Bug 2: memory_existed should be None (not False), error should be surfaced
        assert finding["memory_existed"] is None, "memory_existed should be None on query error"
        assert finding["existence_check_error"] is not None, "existence_check_error should be set"
        assert "vault checkout failed" in finding["existence_check_error"], "error message should be preserved"

    def test_judge_moment_with_engram_query_success_no_results(
        self, fixture_main_session, detect_stub
    ):
        """judge_moment should distinguish error (None) from success-with-no-results (False)."""
        extracted = extract_transcript(fixture_main_session)
        windows = audit_moments.split_into_windows(fixture_main_session)
        window = windows[0]

        candidates = audit_moments.detect_moments(
            window, model="haiku", stub=detect_stub
        )
        assert len(candidates) > 0
        candidate = candidates[0]

        # Stub that returns search_phrases
        def judge_stub_with_phrases(prompt: str, model: str) -> dict:
            judgment = {
                "search_targeted_right_thing": True,
                "outdated_outranked_replacement": False,
                "followed": "yes",
                "worth_learning_from": True,
                "note_well_targeted": True,
                "note_superseded_correctly": False,
                "note_not_duplicate": True,
                "strength_mismatch_flagged": False,
                "failure_category": None,
                "search_phrases": ["test query"],
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        # Mock _run_engram_query_at_moment to return success with no results
        with unittest.mock.patch.object(
            audit_moments, "_run_engram_query_at_moment"
        ) as mock_query:
            mock_query.return_value = {"items": [], "raw_output": "...", "error": None}

            with unittest.mock.patch.object(
                audit_moments, "_extract_timestamp_from_window", return_value="2026-08-31T10:00:00Z"
            ):
                record = audit_moments.judge_moment(
                    candidate, window, extracted, model="sonnet", stub=judge_stub_with_phrases
                )

        finding = record["finding_side"]
        # Bug 2: when query succeeds but finds nothing, memory_existed should be False (not None)
        assert finding["memory_existed"] is False, "memory_existed should be False when no results found"
        assert finding["existence_check_error"] is None, "existence_check_error should be None on success"


class TestExhaustedRetriesPropagation:
    """
    Test that a genuinely exhausted retry budget (2026-09-02 incident: a sustained
    stale-credential outage made every claude -p call fail, but the driver logged each
    window as a normal "0 moments" result for ~4 hours undetected) is surfaced as an
    explicit, distinguishable signal all the way up through detect_moments/judge_moment
    -- never silently folded into "0 candidates" / "default judgment", which is what a
    genuine empty/zero-yield result also looks like.
    """

    def _minimal_window(self):
        return {
            "lines": ["line 1\n", "line 2\n", "line 3\n"],
            "location_start": 1,
            "location_end": 3,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

    def _minimal_extracted(self):
        return {
            "recall_calls": [],
            "dispatch_events": [],
            "learn_calls": [],
            "engram_commands": [],
            "dispatch_type": "main",
            "transcript_path": "test.jsonl",
            "role": "main",
        }

    def test_run_claude_p_signals_exhausted_when_subprocess_always_fails(self, monkeypatch):
        """_run_claude_p should set exhausted_retries=True when every subprocess.run
        attempt in the backoff sequence fails (simulating a sustained auth/API outage)."""
        call_count = {"count": 0}

        def always_failing_run(*args, **kwargs):
            call_count["count"] += 1
            # Every attempt returns a cheap (transient-looking) error -- exactly what a
            # stale-credential outage produces on every single call, not just the first.
            response_obj = {"is_error": True, "result": "", "total_cost_usd": 0.0}
            return subprocess.CompletedProcess(
                args=args, returncode=0, stdout=json.dumps(response_obj), stderr=""
            )

        monkeypatch.setattr(subprocess, "run", always_failing_run)
        monkeypatch.setattr(time, "sleep", lambda *_: None)

        result = audit_moments._run_claude_p("test prompt", "haiku")

        assert result["exhausted_retries"] is True
        assert result["result"] == ""
        assert result["total_cost_usd"] == 0
        # All 4 backoff slots (0, 15, 45, 120) were attempted, not just the first
        # (call_count also includes the one keychain-seeding subprocess.run call that
        # _build_cfg_dir makes before the retry loop starts, same as the existing
        # test_json_parsing_with_is_error_true pattern's ">= 4").
        assert call_count["count"] >= 4

    def test_run_claude_p_distinguishable_from_genuine_empty_result(self, monkeypatch):
        """A real (non-erroring) response with an empty result string must NOT be
        flagged as exhausted_retries -- that's a genuine "nothing" answer, distinct
        from "we never got a real answer"."""

        def genuine_empty_run(*args, **kwargs):
            response_obj = {"is_error": False, "result": "", "total_cost_usd": 0.003}
            return subprocess.CompletedProcess(
                args=args, returncode=0, stdout=json.dumps(response_obj), stderr=""
            )

        monkeypatch.setattr(subprocess, "run", genuine_empty_run)

        result = audit_moments._run_claude_p("test prompt", "haiku")

        assert result["exhausted_retries"] is False
        assert result["result"] == ""

    def test_detect_moments_raises_on_exhausted_retries(self, monkeypatch):
        """detect_moments must propagate exhausted_retries as a raised exception, not
        return an empty candidate list indistinguishable from a genuine zero-yield
        window."""

        def always_failing_run(*args, **kwargs):
            response_obj = {"is_error": True, "result": "", "total_cost_usd": 0.0}
            return subprocess.CompletedProcess(
                args=args, returncode=0, stdout=json.dumps(response_obj), stderr=""
            )

        monkeypatch.setattr(subprocess, "run", always_failing_run)
        monkeypatch.setattr(time, "sleep", lambda *_: None)

        window = self._minimal_window()

        with pytest.raises(audit_moments.ExhaustedRetriesError):
            audit_moments.detect_moments(window, model="haiku")

    def test_detect_moments_genuine_zero_candidates_does_not_raise(self):
        """A genuine, successful "no moments found" response must still return an empty
        list without raising -- proving the two cases are distinguishable."""

        def genuine_zero_yield_stub(prompt: str, model: str) -> dict:
            return {
                "result": json.dumps({"candidates": []}),
                "total_cost_usd": 0.001,
                "exhausted_retries": False,
            }

        window = self._minimal_window()
        candidates = audit_moments.detect_moments(window, model="haiku", stub=genuine_zero_yield_stub)

        assert candidates == []

    def test_judge_moment_raises_on_exhausted_retries(self, monkeypatch):
        """judge_moment must propagate exhausted_retries as a raised exception, not
        return a default/empty judgment indistinguishable from a genuine judgment."""

        def always_failing_run(*args, **kwargs):
            response_obj = {"is_error": True, "result": "", "total_cost_usd": 0.0}
            return subprocess.CompletedProcess(
                args=args, returncode=0, stdout=json.dumps(response_obj), stderr=""
            )

        monkeypatch.setattr(subprocess, "run", always_failing_run)
        monkeypatch.setattr(time, "sleep", lambda *_: None)

        window = self._minimal_window()
        extracted = self._minimal_extracted()
        candidate = {"location": 2, "moment_type": "failure", "description": "test"}

        with pytest.raises(audit_moments.ExhaustedRetriesError):
            audit_moments.judge_moment(candidate, window, extracted, model="sonnet")

    def test_judge_moment_exhausted_retries_stub_raises(self):
        """A stub simulating an exhausted-retries response (as a real _run_claude_p call
        would produce during the 2026-09-02-style outage) must also raise, not silently
        return a default judgment record -- proving detect_moments/judge_moment check
        the signal regardless of whether it came from a stub or a real subprocess call."""

        def exhausted_stub(prompt: str, model: str) -> dict:
            return {"result": "", "total_cost_usd": 0, "exhausted_retries": True}

        window = self._minimal_window()
        extracted = self._minimal_extracted()
        candidate = {"location": 2, "moment_type": "failure", "description": "test"}

        with pytest.raises(audit_moments.ExhaustedRetriesError):
            audit_moments.judge_moment(candidate, window, extracted, model="sonnet", stub=exhausted_stub)


class TestSingleWindowThresholdGap:
    """
    Test that the 150-500 KB gap (real incident: a 431.8 KB single-window prompt hit
    claude -p's terminal_reason="prompt_too_long") is closed -- split_into_windows()'s
    single-vs-multi-window threshold is now pinned to max_kb (150), not the old,
    separate 500 KB cutoff.
    """

    def test_previously_unsafe_gap_file_now_splits_into_multiple_windows(self):
        """A synthetic ~300-400 KB file (squarely inside the old 150-500 KB single-window
        gap) must now be split into multiple <=150 KB windows, not returned as one
        unsplit window."""
        with tempfile.NamedTemporaryFile(mode='w', suffix='.jsonl', delete=False) as f:
            temp_path = f.name
            # ~220 lines * ~1.58 KB/line lands in the 300-400 KB range.
            for i in range(220):
                line = {
                    "type": "user",
                    "timestamp": "2026-09-02T10:00:00Z",
                    "message": "x" * 1500,
                }
                f.write(json.dumps(line) + "\n")

        try:
            file_size_kb = os.path.getsize(temp_path) / 1024
            # Confirm the fixture actually lands in the previously-unsafe gap: big
            # enough to have qualified for the OLD 500 KB single-window cutoff, but
            # well over the new max_kb (150) threshold.
            assert 150 < file_size_kb < 500, (
                f"fixture file_size_kb={file_size_kb} is not in the 150-500 KB gap "
                f"this test exists to cover"
            )

            windows = audit_moments.split_into_windows(temp_path)

            # Under the old (500 KB) threshold this file would have produced exactly
            # ONE unsplit window. It must now produce multiple, safely-sized windows.
            assert len(windows) > 1, (
                "expected multiple windows for a file in the previously-unsafe "
                "150-500 KB gap, got a single unsplit window"
            )

            for window in windows:
                window_size_kb = sum(
                    len(line.encode("utf-8")) for line in window["lines"]
                ) / 1024
                assert window_size_kb <= 150 + 5, (
                    f"window {window['window_index']} is {window_size_kb:.1f} KB, "
                    f"over the 150 KB per-window cap"
                )

            # Line integrity: every original line is accounted for exactly once, across
            # windows, in order (the multi-window splitting path's own correctness,
            # which this fix must not disturb).
            with open(temp_path) as fh:
                original_lines = fh.readlines()
            reconstructed = [line for w in windows for line in w["lines"]]
            assert reconstructed == original_lines
        finally:
            os.unlink(temp_path)

    def test_file_just_under_max_kb_stays_single_window(self):
        """A file comfortably within max_kb (150 KB) should still be a single window --
        the fix must not over-correct into splitting small files unnecessarily."""
        with tempfile.NamedTemporaryFile(mode='w', suffix='.jsonl', delete=False) as f:
            temp_path = f.name
            for i in range(10):
                line = {"type": "user", "timestamp": "2026-09-02T10:00:00Z", "message": "x" * 100}
                f.write(json.dumps(line) + "\n")

        try:
            file_size_kb = os.path.getsize(temp_path) / 1024
            assert file_size_kb < 150

            windows = audit_moments.split_into_windows(temp_path)
            assert len(windows) == 1
            assert windows[0]["total_windows"] == 1
        finally:
            os.unlink(temp_path)


class TestPromptTooLongFastFail:
    """
    Test that a deterministic terminal_reason="prompt_too_long" response from claude -p
    short-circuits the retry loop immediately (no wasted ~3-minute backoff sequence),
    and that the distinguishing prompt_too_long signal propagates through
    detect_moments/judge_moment as a distinct PromptTooLongError -- never mistaken for
    exhausted_retries or a genuine empty result.
    """

    def _minimal_window(self):
        return {
            "lines": ["line 1\n", "line 2\n", "line 3\n"],
            "location_start": 1,
            "location_end": 3,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

    def _minimal_extracted(self):
        return {
            "recall_calls": [],
            "dispatch_events": [],
            "learn_calls": [],
            "engram_commands": [],
            "dispatch_type": "main",
            "transcript_path": "test.jsonl",
            "role": "main",
        }

    def test_run_claude_p_returns_immediately_on_prompt_too_long(self, monkeypatch):
        """_run_claude_p must call the `claude` subprocess exactly ONCE (no
        retry-loop delay) when the response is a deterministic prompt_too_long
        rejection, and must set prompt_too_long=True, exhausted_retries=False."""
        claude_call_count = {"count": 0}

        def prompt_too_long_run(*args, **kwargs):
            # subprocess.run is also used by _build_cfg_dir's one-off keychain-seeding
            # call (["bash", "-c", ...]) before the retry loop even starts -- only the
            # actual `claude` invocation is what "not retried" is asserting about, so
            # count and respond to those specifically.
            invoked = args[0] if args else kwargs.get("args", [])
            if invoked and invoked[0] == "claude":
                claude_call_count["count"] += 1
                # Mirrors the real independently-reproduced claude -p response.
                response_obj = {
                    "is_error": True,
                    "total_cost_usd": 0,
                    "duration_api_ms": 0,
                    "terminal_reason": "prompt_too_long",
                    "result": "",
                }
                return subprocess.CompletedProcess(
                    args=invoked, returncode=0, stdout=json.dumps(response_obj), stderr=""
                )
            # Keychain-seeding call (or any other non-`claude` subprocess call): no-op success.
            return subprocess.CompletedProcess(args=invoked, returncode=0, stdout="", stderr="")

        def sleep_should_not_be_called(*_):
            raise AssertionError("time.sleep should not be called -- prompt_too_long must not retry")

        monkeypatch.setattr(subprocess, "run", prompt_too_long_run)
        monkeypatch.setattr(time, "sleep", sleep_should_not_be_called)

        result = audit_moments._run_claude_p("test prompt", "haiku")

        assert claude_call_count["count"] == 1, (
            f"expected exactly 1 `claude` subprocess call, got {claude_call_count['count']} -- "
            f"prompt_too_long must not be retried"
        )
        assert result["prompt_too_long"] is True
        assert result["exhausted_retries"] is False

    def test_genuine_transient_error_still_retries_as_before(self, monkeypatch):
        """A genuinely transient is_error/low-cost response (no terminal_reason at all)
        must still go through the full retry-with-backoff sequence -- the
        prompt_too_long check must not break existing transient-failure handling."""
        call_count = {"count": 0}

        def transient_then_success_run(*args, **kwargs):
            call_count["count"] += 1
            if call_count["count"] <= 3:
                response_obj = {"is_error": True, "result": "", "total_cost_usd": 0.001}
            else:
                response_obj = {"is_error": False, "result": "Success", "total_cost_usd": 0.005}
            return subprocess.CompletedProcess(
                args=args, returncode=0, stdout=json.dumps(response_obj), stderr=""
            )

        monkeypatch.setattr(subprocess, "run", transient_then_success_run)
        monkeypatch.setattr(time, "sleep", lambda *_: None)

        result = audit_moments._run_claude_p("test prompt", "haiku")

        assert call_count["count"] >= 4
        assert result["result"] == "Success"
        assert result["exhausted_retries"] is False
        assert result["prompt_too_long"] is False

    def test_detect_moments_raises_prompt_too_long_not_exhausted_retries(self, monkeypatch):
        """detect_moments must raise the distinct PromptTooLongError, not
        ExhaustedRetriesError, when the window's content is too large -- and must not
        retry (the `claude` subprocess called exactly once)."""
        claude_call_count = {"count": 0}

        def prompt_too_long_run(*args, **kwargs):
            # See test_run_claude_p_returns_immediately_on_prompt_too_long: only count
            # the actual `claude` invocation, not _build_cfg_dir's keychain-seeding call.
            invoked = args[0] if args else kwargs.get("args", [])
            if invoked and invoked[0] == "claude":
                claude_call_count["count"] += 1
                response_obj = {
                    "is_error": True,
                    "total_cost_usd": 0,
                    "duration_api_ms": 0,
                    "terminal_reason": "prompt_too_long",
                    "result": "",
                }
                return subprocess.CompletedProcess(
                    args=invoked, returncode=0, stdout=json.dumps(response_obj), stderr=""
                )
            return subprocess.CompletedProcess(args=invoked, returncode=0, stdout="", stderr="")

        monkeypatch.setattr(subprocess, "run", prompt_too_long_run)
        monkeypatch.setattr(time, "sleep", lambda *_: (_ for _ in ()).throw(
            AssertionError("must not sleep/retry on prompt_too_long")
        ))

        window = self._minimal_window()

        with pytest.raises(audit_moments.PromptTooLongError):
            audit_moments.detect_moments(window, model="haiku")

        assert claude_call_count["count"] == 1

    def test_judge_moment_raises_prompt_too_long_not_exhausted_retries(self, monkeypatch):
        """judge_moment must raise the distinct PromptTooLongError, not
        ExhaustedRetriesError, when the moment's window content is too large."""

        def prompt_too_long_run(*args, **kwargs):
            response_obj = {
                "is_error": True,
                "total_cost_usd": 0,
                "duration_api_ms": 0,
                "terminal_reason": "prompt_too_long",
                "result": "",
            }
            return subprocess.CompletedProcess(
                args=args, returncode=0, stdout=json.dumps(response_obj), stderr=""
            )

        monkeypatch.setattr(subprocess, "run", prompt_too_long_run)
        monkeypatch.setattr(time, "sleep", lambda *_: None)

        window = self._minimal_window()
        extracted = self._minimal_extracted()
        candidate = {"location": 2, "moment_type": "failure", "description": "test"}

        with pytest.raises(audit_moments.PromptTooLongError):
            audit_moments.judge_moment(candidate, window, extracted, model="sonnet")

    def test_detect_moments_prompt_too_long_stub_raises(self):
        """A stub simulating a prompt_too_long response (as a real _run_claude_p call
        would produce) must also raise PromptTooLongError, not ExhaustedRetriesError
        and not silently return an empty candidate list."""

        def prompt_too_long_stub(prompt: str, model: str) -> dict:
            return {
                "result": "",
                "total_cost_usd": 0,
                "exhausted_retries": False,
                "prompt_too_long": True,
            }

        window = self._minimal_window()

        with pytest.raises(audit_moments.PromptTooLongError):
            audit_moments.detect_moments(window, model="haiku", stub=prompt_too_long_stub)


class TestFailureCategoryPromptScope:
    """The judgment prompt's failure_category block must cover ALL moment types,
    not just moment_type=='failure' (the old 'if moment_type="failure"' scoping left
    corrections, rework, successes, and dispatches uncategorized).

    - failure/correction/rework get the runbook-824 question: could a relevant
      memory, recalled at the right time, have prevented or caught this problem?
    - success/dispatch get the capture-reachability variant: could a mandatory
      "record lessons in your completion report" step realistically have captured
      the lesson?

    The JSON response schema line (same enum values) stays unchanged either way.
    """

    RUNBOOK_824_QUESTION = (
        "Could a relevant memory, recalled at the right time, have prevented "
        "or caught this problem?"
    )
    # Instrument v2: "not_applicable" replaces the bare-null option in the
    # failure_category enum (nulls were unasked, not unanswerable -- vault note 900).
    ORIGINAL_ENUM_LINE = (
        '- failure_category: "fixable" (could have been caught by memory), '
        '"nothing_could_have_caught_it", "found_but_not_followed", or "not_applicable"'
    )
    CAPTURE_REACHABILITY_MARKER = "record lessons in your completion report"
    SCHEMA_LINE = (
        '"failure_category": "fixable" | "nothing_could_have_caught_it" | '
        '"found_but_not_followed" | "not_applicable",'
    )

    def _capture_judgment_prompt(self, moment_type: str) -> str:
        """Run judge_moment on a minimal candidate of moment_type; return the
        assembled judgment prompt captured via a recording stub."""
        window = {
            "lines": ["line 1\n"],
            "location_start": 1,
            "location_end": 1,
            "transcript_path": "test.jsonl",
            "window_index": 0,
            "total_windows": 1,
            "oversized_line": False,
        }

        candidate = {
            "location": 1,
            "moment_type": moment_type,
            "description": f"Test {moment_type} moment",
        }

        extracted = {
            "recall_calls": [],
            "dispatch_events": [],
            "learn_calls": [],
            "engram_commands": [],
            "dispatch_type": "main",
            "transcript_path": "test.jsonl",
            "role": "main",
        }

        captured_prompts = []

        def judge_stub(prompt: str, model: str) -> dict:
            captured_prompts.append(prompt)
            judgment = {
                "search_targeted_right_thing": None,
                "outdated_outranked_replacement": None,
                "followed": None,
                "worth_learning_from": True,
                "note_well_targeted": None,
                "note_superseded_correctly": None,
                "note_not_duplicate": None,
                "strength_mismatch_flagged": None,
                "failure_category": None,
            }
            return {"result": json.dumps(judgment), "total_cost_usd": 0.002}

        audit_moments.judge_moment(
            candidate, window, extracted, model="sonnet", stub=judge_stub
        )

        assert len(captured_prompts) == 1
        return captured_prompts[0]

    def test_rework_moment_gets_categorization_guidance(self):
        """A rework moment's prompt must contain the runbook-824 categorization
        question -- not scoped away behind moment_type=='failure'."""
        prompt = self._capture_judgment_prompt("rework")

        # The old scoping text must be gone.
        assert 'if moment_type="failure"' not in prompt
        # The runbook-824 question and its enum apply to rework moments.
        assert self.RUNBOOK_824_QUESTION in prompt
        assert self.ORIGINAL_ENUM_LINE in prompt
        # The capture-reachability variant is for success/dispatch only.
        assert self.CAPTURE_REACHABILITY_MARKER not in prompt
        # The JSON response schema line stays unchanged.
        assert self.SCHEMA_LINE in prompt

    def test_success_moment_gets_capture_reachability_variant(self):
        """A success moment's prompt must ask the capture-reachability variant:
        could a mandatory record-lessons-in-completion-report step realistically
        have captured the lesson?"""
        prompt = self._capture_judgment_prompt("success")

        assert 'if moment_type="failure"' not in prompt
        assert self.CAPTURE_REACHABILITY_MARKER in prompt
        assert "realistically have captured that lesson" in prompt
        # Variant enum meanings: fixable / nothing_could_have_caught_it / null.
        assert '"fixable" (yes, realistically capturable)' in prompt
        assert '"nothing_could_have_caught_it" (no plausible mechanism would have)' in prompt
        assert "not worth learning from, or a learn step DID fire" in prompt
        # The prevention question belongs to failure/correction/rework only.
        assert self.RUNBOOK_824_QUESTION not in prompt
        # The JSON response schema line stays unchanged (same enum values).
        assert self.SCHEMA_LINE in prompt

    def test_failure_moment_regression_keeps_runbook_824_wording(self):
        """Regression: a failure moment still gets the original runbook-824
        question with the original enum line, not the capture-reachability
        variant."""
        prompt = self._capture_judgment_prompt("failure")

        assert self.RUNBOOK_824_QUESTION in prompt
        assert self.ORIGINAL_ENUM_LINE in prompt
        assert self.CAPTURE_REACHABILITY_MARKER not in prompt
        assert self.SCHEMA_LINE in prompt


def _minimal_window():
    """A one-line window for prompt/parsing tests that never hit the filesystem."""
    return {
        "lines": ['{"type":"user","timestamp":"2026-08-30T14:30:00+00:00"}\n'],
        "location_start": 1,
        "location_end": 1,
        "transcript_path": "test.jsonl",
        "window_index": 0,
        "total_windows": 1,
        "oversized_line": False,
    }


def _minimal_extracted():
    """Minimal extracted-events dict for a main-role moment."""
    return {
        "recall_calls": [],
        "dispatch_events": [],
        "learn_calls": [],
        "engram_commands": [],
        "dispatch_type": "main",
        "transcript_path": "test.jsonl",
        "role": "main",
    }


def _minimal_candidate(moment_type="failure"):
    return {
        "location": 1,
        "moment_type": moment_type,
        "description": f"Test {moment_type} moment",
    }


def _judgment_result(**overrides):
    """A canned, fully-valued judgment response body (no not_applicable, no nulls)."""
    judgment = {
        "search_targeted_right_thing": True,
        "outdated_outranked_replacement": False,
        "followed": "yes",
        "worth_learning_from": True,
        "note_well_targeted": True,
        "note_superseded_correctly": False,
        "note_not_duplicate": True,
        "strength_mismatch_flagged": False,
        "failure_category": "fixable",
    }
    judgment.update(overrides)
    return {"result": json.dumps(judgment), "total_cost_usd": 0.002}


class TestInstrumentV2PromptContent:
    """Instrument v2 folds the gate-week patch-learnings into judge_moment's own
    judgment prompt, so a future re-run does not regress to bare nulls and
    thresholdless existence checks. Each test proves one new instruction appears."""

    def _capture_judgment_prompt(self, moment_type="failure"):
        captured = []

        def judge_stub(prompt: str, model: str) -> dict:
            captured.append(prompt)
            return _judgment_result()

        audit_moments.judge_moment(
            _minimal_candidate(moment_type),
            _minimal_window(),
            _minimal_extracted(),
            model="sonnet",
            stub=judge_stub,
        )
        assert len(captured) == 1
        return captured[0]

    def test_followed_enum_includes_not_applicable_with_null_ban(self):
        """The followed instruction must include "not_applicable" in the enum and
        the exact null-ban framing that worked in rejudge_followed_nulls.py."""
        prompt = self._capture_judgment_prompt()

        # Schema line: not_applicable joins the enum, null leaves it.
        assert (
            '"followed": "yes" | "ignored" | "partial" | "skipped_step" | '
            '"out_of_order" | "contradicted" | "not_applicable"' in prompt
        )
        # The bare-null ban when surfaced is true.
        assert "a bare null is NOT a permitted answer" in prompt
        # The exact framing rejudge_followed_nulls.py used.
        assert (
            'if you cannot render any of the first six verdicts, "not_applicable" '
            "with a rationale is the honest answer" in prompt
        )

    def test_conditional_fields_get_not_applicable_instruction(self):
        """Every conditional judged field must be told to answer not_applicable
        (never a bare null) when its condition does not hold."""
        prompt = self._capture_judgment_prompt()

        # The instruction names every conditional judged field (schema-level fix,
        # vault note 900a -- never scoped to the exemplar field).
        conditional_instruction = prompt[prompt.index("For EVERY conditional judged field") :]
        for field in (
            "search_targeted_right_thing",
            "outdated_outranked_replacement",
            "note_well_targeted",
            "note_superseded_correctly",
            "note_not_duplicate",
            "strength_mismatch_flagged",
            "failure_category",
        ):
            assert field in conditional_instruction, f"{field} missing from the instruction"
        assert 'answer "not_applicable" -- never a bare null' in prompt
        assert "Do NOT answer null anywhere" in prompt

        # The response schema offers "not_applicable" (not null) on each conditional binary.
        for field in (
            "search_targeted_right_thing",
            "outdated_outranked_replacement",
            "note_well_targeted",
            "note_superseded_correctly",
            "note_not_duplicate",
            "strength_mismatch_flagged",
        ):
            assert f'"{field}": true | false | "not_applicable"' in prompt

    def test_rationales_object_requested(self):
        """The judge must be asked for a one-sentence rationale per answered
        question, delivered as a "rationales" object in the response."""
        prompt = self._capture_judgment_prompt()

        assert '"rationales"' in prompt
        assert "one-sentence rationale per answered question" in prompt
        assert '"rationales": {"<field>": "<one sentence>"' in prompt


class TestInstrumentV2NotApplicableParsing:
    """not_applicable verdicts parse to null field values plus a
    <field>_null_reason='judged_not_applicable' stamp -- the merged corpus's
    convention -- never a new enum value stored in the field itself."""

    def _judge(self, **judgment_overrides):
        def judge_stub(prompt: str, model: str) -> dict:
            return _judgment_result(**judgment_overrides)

        return audit_moments.judge_moment(
            _minimal_candidate(),
            _minimal_window(),
            _minimal_extracted(),
            model="sonnet",
            stub=judge_stub,
        )

    def test_followed_not_applicable_parses_to_null_plus_reason(self):
        record = self._judge(followed="not_applicable")

        finding = record["finding_side"]
        assert finding["followed"] is None
        assert finding["followed_null_reason"] == "judged_not_applicable"

    def test_conditional_finding_fields_not_applicable_stamp_null_reason(self):
        record = self._judge(
            search_targeted_right_thing="not_applicable",
            outdated_outranked_replacement="not_applicable",
        )

        finding = record["finding_side"]
        assert finding["search_targeted_right_thing"] is None
        assert finding["search_targeted_right_thing_null_reason"] == "judged_not_applicable"
        assert finding["outdated_outranked_replacement"] is None
        assert finding["outdated_outranked_replacement_null_reason"] == "judged_not_applicable"

    def test_conditional_writing_fields_not_applicable_stamp_null_reason(self):
        record = self._judge(
            note_well_targeted="not_applicable",
            note_superseded_correctly="not_applicable",
            note_not_duplicate="not_applicable",
            strength_mismatch_flagged="not_applicable",
        )

        writing = record["writing_side"]
        for field in (
            "note_well_targeted",
            "note_superseded_correctly",
            "note_not_duplicate",
            "strength_mismatch_flagged",
        ):
            assert writing[field] is None, field
            assert writing[f"{field}_null_reason"] == "judged_not_applicable", field

    def test_failure_category_not_applicable_stamps_null_reason(self):
        record = self._judge(failure_category="not_applicable")

        assert record["failure_category"] is None
        assert record["failure_category_null_reason"] == "judged_not_applicable"

    def test_real_verdicts_leave_no_null_reason_keys(self):
        """A fully-valued judgment must not sprout any *_null_reason keys --
        the stamp appears only where a not_applicable verdict earned it."""
        record = self._judge()

        for side in (record["finding_side"], record["writing_side"]):
            null_reason_keys = [k for k in side if k.endswith("_null_reason")]
            assert null_reason_keys == []
        assert "failure_category_null_reason" not in record
        assert record["finding_side"]["followed"] == "yes"
        assert record["failure_category"] == "fixable"

    def test_rationales_stored_as_judgment_rationales(self):
        rationales = {
            "followed": "The agent applied the memory verbatim.",
            "failure_category": "A recalled note names this exact gotcha.",
        }
        record = self._judge(rationales=rationales)

        assert record["judgment_rationales"] == rationales

    def test_malformed_rationales_default_to_empty_object(self):
        record = self._judge(rationales="not a dict")

        assert record["judgment_rationales"] == {}


class TestInstrumentVersionStamp:
    """Every record must declare which instrument produced it (comparability
    guard, vault note 282's deploy-byte-identical lesson): version 1 is the
    original run, this module is version 2."""

    def test_module_constant_is_2(self):
        assert audit_moments.INSTRUMENT_VERSION == 2

    def test_record_carries_instrument_version(self):
        def judge_stub(prompt: str, model: str) -> dict:
            return _judgment_result()

        record = audit_moments.judge_moment(
            _minimal_candidate(),
            _minimal_window(),
            _minimal_extracted(),
            model="sonnet",
            stub=judge_stub,
        )

        assert record["instrument_version"] == audit_moments.INSTRUMENT_VERSION


class TestExistenceTop5AndRelevanceGrade:
    """The point-in-time existence check must no longer be a thresholdless
    existence gate: when items come back, the top-5 (path/kind/score) land on
    the record and one more judgment question grades their relevance against
    the moment with measure_relevance_cues.py's strict framing."""

    ITEMS = [
        {
            "rank": rank,
            "path": f"note-{rank}.md",
            "kind": "feedback",
            "score": round(0.9 - rank * 0.1, 2),
            "source": "vault",
        }
        for rank in range(1, 7)
    ]

    def _run_judge(self, monkeypatch, items, relevance_result, captured=None):
        """judge_moment with the DERIVED existence-check plumbing monkeypatched
        (no real vault/git/engram), a stub answering the judgment call with
        search_phrases and the relevance call with relevance_result."""
        import contextlib as ctx

        monkeypatch.setattr(
            audit_moments,
            "_extract_timestamp_from_window",
            lambda window, location: "2026-08-30T14:30:00+00:00",
        )
        monkeypatch.setattr(
            audit_moments, "_find_commit_at_or_before", lambda repo, ts: "abc1234"
        )
        monkeypatch.setattr(
            audit_moments, "_is_commit_ancestor_of", lambda repo, a, d: True
        )

        @ctx.contextmanager
        def fake_checkout(repo, sha, scratch):
            yield scratch

        monkeypatch.setattr(audit_moments, "_checkout_vault_at", fake_checkout)
        monkeypatch.setattr(
            audit_moments,
            "_filter_chunks_by_ingestion_date",
            lambda chunks, ts, scratch: scratch,
        )
        monkeypatch.setattr(
            audit_moments,
            "_run_engram_query_at_moment",
            lambda vault, chunks, phrases, **kw: {"items": items, "error": None},
        )

        if captured is None:
            captured = []

        def two_phase_stub(prompt: str, model: str) -> dict:
            captured.append(prompt)
            if "relevance_grade" in prompt:
                return {"result": relevance_result, "total_cost_usd": 0.001}
            return _judgment_result(search_phrases=["alpha beta", "gamma delta"])

        return audit_moments.judge_moment(
            _minimal_candidate("success"),
            _minimal_window(),
            _minimal_extracted(),
            model="sonnet",
            stub=two_phase_stub,
        )

    def test_top5_captured_and_relevance_graded(self, monkeypatch):
        captured = []
        relevance_result = json.dumps(
            {"relevance_grade": "relevant_top1", "rationale": "Rank 1 names this exact trap."}
        )
        record = self._run_judge(monkeypatch, self.ITEMS, relevance_result, captured)

        finding = record["finding_side"]
        # Top-5 capped at 5, path/kind/score only (no rank/source/content).
        assert finding["existence_top5"] == [
            {"path": f"note-{rank}.md", "kind": "feedback", "score": round(0.9 - rank * 0.1, 2)}
            for rank in range(1, 6)
        ]
        assert finding["relevance_grade"] == "relevant_top1"
        assert (
            record["judgment_rationales"]["relevance_grade"]
            == "Rank 1 names this exact trap."
        )

        # Two calls: the judgment call, then the relevance-grade call.
        assert len(captured) == 2
        relevance_prompt = captured[1]
        # The strict framing from measure_relevance_cues.py.
        assert (
            "Judge honestly against the moment, not charitably -- a generic lesson "
            "that technically overlaps but would not have changed anything is NOT "
            "relevant" in relevance_prompt
        )
        # The enum and the graded items appear.
        assert '"relevant_top1" | "relevant_in_top5" | "nothing_relevant"' in relevance_prompt
        assert "note-1.md" in relevance_prompt
        assert "note-5.md" in relevance_prompt

    def test_no_items_returned_skips_relevance_call(self, monkeypatch):
        captured = []
        record = self._run_judge(monkeypatch, [], "unused", captured)

        finding = record["finding_side"]
        assert finding["memory_existed"] is False
        assert finding["existence_top5"] is None
        assert finding["relevance_grade"] is None
        assert len(captured) == 1  # judgment call only

    def test_invalid_relevance_response_leaves_grade_null(self, monkeypatch):
        record = self._run_judge(
            monkeypatch, self.ITEMS, json.dumps({"relevance_grade": "very_relevant"})
        )

        assert record["finding_side"]["relevance_grade"] is None
        assert record["finding_side"]["existence_top5"] is not None


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
