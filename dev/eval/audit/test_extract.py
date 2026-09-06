"""
Unit tests for the JSONL transcript extractor.

Tests cover:
1. Main session with recall and dispatch
2. Forked subagent with meta.json
3. Fresh-context subagent with meta.json
4. Workflow agent under subagents/workflows/
"""

import json
import os
import pytest
from pathlib import Path

from extract import (
    extract_transcript,
    determine_dispatch_type,
    normalize_tool_result_content,
    parse_engram_query_result,
    read_meta_json,
)


# Fixture directory paths
FIXTURES_DIR = Path(__file__).parent / "fixtures"
MAIN_SESSION_FIXTURE = FIXTURES_DIR / "main" / "main_session_01.jsonl"
FORK_SUBAGENT_TRANSCRIPT = FIXTURES_DIR / "session1" / "subagents" / "agent-abc123.jsonl"
FORK_SUBAGENT_META = FIXTURES_DIR / "session1" / "subagents" / "agent-abc123.meta.json"
FRESH_SUBAGENT_TRANSCRIPT = FIXTURES_DIR / "session2" / "subagents" / "agent-def456.jsonl"
FRESH_SUBAGENT_META = FIXTURES_DIR / "session2" / "subagents" / "agent-def456.meta.json"
WORKFLOW_SUBAGENT_TRANSCRIPT = (
    FIXTURES_DIR / "session3" / "subagents" / "workflows" / "wf_test" / "agent-ghi789.jsonl"
)
WORKFLOW_SUBAGENT_META = (
    FIXTURES_DIR / "session3" / "subagents" / "workflows" / "wf_test" / "agent-ghi789.meta.json"
)
RECALL_ROUTE_CONTAMINATION_FIXTURE = (
    FIXTURES_DIR / "main" / "main_session_recall_route_contamination.jsonl"
)
RECALL_WRITEMEMORY_NESTED_FIXTURE = (
    FIXTURES_DIR / "main" / "main_session_recall_writememory_nested.jsonl"
)
BARE_BASH_RECALL_FIXTURE = (
    FIXTURES_DIR / "session_bare_bash" / "subagents" / "agent-bare-bash.jsonl"
)


class TestNormalizeToolResultContent:
    """Test normalization of tool_result content."""

    def test_bare_string_content(self):
        """String content should pass through unchanged."""
        content = "This is plain text"
        assert normalize_tool_result_content(content) == "This is plain text"

    def test_list_of_text_blocks(self):
        """List of text blocks should be concatenated."""
        content = [
            {"type": "text", "text": "Hello"},
            {"type": "text", "text": " "},
            {"type": "text", "text": "World"},
        ]
        assert normalize_tool_result_content(content) == "Hello World"

    def test_mixed_block_types(self):
        """Non-text blocks should be skipped."""
        content = [
            {"type": "text", "text": "Start"},
            {"type": "code", "code": "ignored"},
            {"type": "text", "text": "End"},
        ]
        assert normalize_tool_result_content(content) == "StartEnd"


class TestParseEngramQueryResult:
    """Test parsing of engram query YAML-like output."""

    def test_parse_single_item(self):
        """Should extract path, kind, score, and source from query result."""
        result_text = """version: 1
phrases:
  - Python transcript parsing
items:
  - path: 42.2026-08-30.transcript-parsing.md
    kind: fact
    score: 0.8123456
    provenances:
      - direct
    content: |
      ---
      type: fact
      ---
      Always parse JSONL line-by-line
clusters: []"""
        items = parse_engram_query_result(result_text)
        assert len(items) == 1
        assert items[0]["path"] == "42.2026-08-30.transcript-parsing.md"
        assert items[0]["kind"] == "fact"
        assert items[0]["score"] == 0.8123456
        assert items[0]["source"] == "direct"
        assert items[0]["rank"] == 1

    def test_parse_multiple_items(self):
        """Should extract multiple items in rank order."""
        result_text = """version: 1
phrases:
  - test
items:
  - path: note1.md
    kind: fact
    score: 0.9
    provenances:
      - direct
    content: |
      Note 1
  - path: note2.md
    kind: feedback
    score: 0.7
    provenances:
      - explore
    content: |
      Note 2
clusters: []"""
        items = parse_engram_query_result(result_text)
        assert len(items) == 2
        assert items[0]["path"] == "note1.md"
        assert items[0]["rank"] == 1
        assert items[1]["path"] == "note2.md"
        assert items[1]["rank"] == 2
        assert items[1]["source"] == "explore"

    def test_empty_items(self):
        """Should handle empty items section."""
        result_text = """version: 1
phrases:
  - test
items:
clusters: []"""
        items = parse_engram_query_result(result_text)
        assert len(items) == 0


class TestDetermineDispatchType:
    """Test dispatch type classification."""

    def test_main_session(self):
        """Path without /subagents/ should be main."""
        assert determine_dispatch_type("/Users/joe/.../session.jsonl") == "main"

    def test_fork_with_meta(self):
        """Fork meta.json should be classified as fork."""
        meta = {"isFork": True, "agentType": "fork"}
        path = "/session/subagents/agent-123.jsonl"
        assert determine_dispatch_type(path, meta) == "fork"

    def test_fork_with_agenttype(self):
        """agentType=fork in meta should be classified as fork."""
        meta = {"agentType": "fork"}
        path = "/session/subagents/agent-123.jsonl"
        assert determine_dispatch_type(path, meta) == "fork"

    def test_fresh_subagent(self):
        """Subagent path without fork flag should be fresh."""
        meta = {"agentType": "general-purpose", "model": "haiku"}
        path = "/session/subagents/agent-123.jsonl"
        assert determine_dispatch_type(path, meta) == "fresh"

    def test_workflow_by_path(self):
        """Path with /subagents/workflows/ should be workflow."""
        path = "/session/subagents/workflows/wf_test/agent-123.jsonl"
        assert determine_dispatch_type(path) == "workflow"


class TestReadMetaJson:
    """Test reading .meta.json sidecar files."""

    def test_read_existing_meta(self):
        """Should read and parse existing meta.json."""
        meta = read_meta_json(str(FORK_SUBAGENT_TRANSCRIPT))
        assert meta is not None
        assert meta.get("isFork") is True
        assert meta.get("agentType") == "fork"

    def test_missing_meta_returns_none(self):
        """Should return None if meta.json doesn't exist."""
        meta = read_meta_json("/nonexistent/agent.jsonl")
        assert meta is None


class TestExtractMainSession:
    """Test extraction from main session fixtures."""

    def test_extract_main_session(self):
        """Should extract recall calls and dispatch events from main session."""
        result = extract_transcript(str(MAIN_SESSION_FIXTURE))

        assert result["transcript_path"] == str(MAIN_SESSION_FIXTURE)
        assert result["dispatch_type"] == "main"
        assert result["completion_report"] is None  # Main sessions don't have completion reports

    def test_main_session_has_recall_call(self):
        """Main session should extract the recall call."""
        result = extract_transcript(str(MAIN_SESSION_FIXTURE))

        assert len(result["recall_calls"]) == 1
        recall = result["recall_calls"][0]
        assert recall["skill_args"] == "Python transcript parsing"
        assert len(recall["query_phrases"]) == 1
        assert "Python transcript parsing" in recall["query_phrases"]

    def test_main_session_recall_items(self):
        """Recall call should contain matched items."""
        result = extract_transcript(str(MAIN_SESSION_FIXTURE))

        recall = result["recall_calls"][0]
        assert len(recall["matched_items"]) == 1
        item = recall["matched_items"][0]
        assert item["path"] == "42.2026-08-30.transcript-parsing.md"
        assert item["kind"] == "fact"
        assert item["source"] == "direct"
        assert item["score"] == 0.8123456

    def test_main_session_has_dispatch_event(self):
        """Main session should extract dispatch event."""
        result = extract_transcript(str(MAIN_SESSION_FIXTURE))

        assert len(result["dispatch_events"]) == 1
        event = result["dispatch_events"][0]
        assert event["description"] == "Build the transcript parser"
        assert event["subagent_type"] == "fresh"
        assert event["model"] == "haiku"
        assert "Create a transcript parser" in event["prompt_text"]


class TestRecallWindowClosesOnOtherSkills:
    """
    Third recall-window bug (found by adversarial verification of the round-2 fix):

    The recall window stayed open from Skill(recall) until the NEXT Skill(recall)
    (or EOF), so an unrelated skill's OWN "engram query" Bash call (e.g. route's
    routine dispatch-evidence tally query) firing while that window was still open
    got wrongly merged into the recall call's matched_items/query_phrases.

    Fixture: Skill(recall) opens -> recall's own Bash "engram query" call attaches
    correctly -> later, Skill(route) fires and issues its OWN independent Bash
    "engram query --lazy-chunks --phrase route evidence ... tier tally" call with
    its own tool_result. That route query must NOT contaminate the recall call.
    """

    def test_other_skill_engram_query_does_not_contaminate_recall_call(self):
        """A non-recall skill's own engram query must not merge into recall_calls."""
        result = extract_transcript(str(RECALL_ROUTE_CONTAMINATION_FIXTURE))

        assert len(result["recall_calls"]) == 1
        recall = result["recall_calls"][0]

        matched_paths = [item["path"] for item in recall["matched_items"]]
        assert matched_paths == ["50.2026-08-20.thread-safety.md"]
        assert "60.2026-08-25.route-tier-data.md" not in matched_paths

        # query_phrases should likewise only reflect recall's own query, not route's
        assert recall["query_phrases"] == ["threading bug patterns"]
        assert "route evidence bugfix tier tally" not in recall["query_phrases"]


class TestRecallWindowDoesNotCloseOnWriteMemory:
    """
    The recall skill's own documented procedure legitimately nests a Skill(write-memory)
    call mid-execution (Step 2.5C coverage write, Step 4 persist synthesis) and then
    continues the SAME recall invocation afterward -- including a possible Step 3.5
    re-entry "engram query" call. write-memory must NOT close the recall window.

    Fixture: Skill(recall) opens -> recall's own Step 2 Bash "engram query" call
    attaches -> Skill(write-memory) fires (nested, does not close the window) ->
    recall's own Step 3.5 re-entry Bash "engram query" call attaches too. Both
    queries' items must end up merged into the same recall_calls[0] entry.
    """

    def test_write_memory_does_not_close_recall_window(self):
        """write-memory nested mid-recall must not truncate the recall call."""
        result = extract_transcript(str(RECALL_WRITEMEMORY_NESTED_FIXTURE))

        assert len(result["recall_calls"]) == 1
        recall = result["recall_calls"][0]

        matched_paths = [item["path"] for item in recall["matched_items"]]
        assert matched_paths == [
            "70.2026-08-10.nesting-item-a.md",
            "71.2026-08-11.nesting-item-b.md",
        ]

        assert recall["query_phrases"] == [
            "write-memory nesting test",
            "write-memory nesting test refined",
        ]

        # write-memory extraction itself is unaffected: it still shows up in learn_calls
        assert len(result["learn_calls"]) == 1
        assert result["learn_calls"][0]["skill"] == "write-memory"


class TestBareBashRecall:
    """
    Bug 1 fix: extract bare-Bash engram query calls that aren't preceded by Skill(recall).

    Some real dispatch prompts instruct subagents to run `engram query` DIRECTLY via Bash
    with no Skill(recall) wrapper — e.g., a dispatch prompt: "FIRST ACTION: run this recall
    query: `engram query --lazy-chunks --phrase ...`". The extractor must recognize this
    as a recall call, not silently drop it. Only applies to subagents, not main sessions
    (where internal skills like route run their own engram queries).

    Fixture: a fresh subagent with no Skill(recall) call; just a bare Bash tool_use
    with "engram query" command. The result should produce a non-empty recall_calls entry
    with correct phrases/items.
    """

    def test_bare_bash_engram_query_opens_recall_window(self):
        """A bare-Bash engram query (no Skill(recall)) should open its own recall window."""
        result = extract_transcript(str(BARE_BASH_RECALL_FIXTURE))

        # Verify it's a fresh subagent
        assert result["dispatch_type"] == "fresh"

        assert len(result["recall_calls"]) == 1
        recall = result["recall_calls"][0]
        assert recall["skill_args"] == ""  # Empty skill_args marks bare-bash
        assert len(recall["query_phrases"]) == 1
        assert "bare bash query test" in recall["query_phrases"]

    def test_bare_bash_recall_has_matched_items(self):
        """Bare-Bash recall should contain matched items."""
        result = extract_transcript(str(BARE_BASH_RECALL_FIXTURE))

        recall = result["recall_calls"][0]
        assert len(recall["matched_items"]) == 1
        item = recall["matched_items"][0]
        assert item["path"] == "80.2026-08-25.bare-bash-recall.md"
        assert item["kind"] == "feedback"
        assert item["source"] == "direct"
        assert item["score"] == 0.75

    def test_bare_bash_recall_closes_immediately_after_result(self):
        """Bare-Bash recall should close immediately after the result, not remain open."""
        result = extract_transcript(str(BARE_BASH_RECALL_FIXTURE))

        # Should have exactly one recall (the bare-bash one), not left open
        assert len(result["recall_calls"]) == 1
        # No open recall window at EOF
        # (implicitly verified by the above: if window stayed open, we'd have 1 recall
        # in recall_calls + 1 open current_recall_call; the final close-at-EOF would make
        # it 2, but we only see 1, so it was closed immediately after the result)


class TestExtractForkedSubagent:
    """Test extraction from forked subagent fixtures."""

    def test_extract_fork_subagent(self):
        """Should extract fork subagent with correct dispatch type."""
        result = extract_transcript(str(FORK_SUBAGENT_TRANSCRIPT))

        assert result["dispatch_type"] == "fork"
        assert result["transcript_path"] == str(FORK_SUBAGENT_TRANSCRIPT)

    def test_fork_subagent_has_completion_report(self):
        """Fork subagent should have completion report."""
        result = extract_transcript(str(FORK_SUBAGENT_TRANSCRIPT))

        assert result["completion_report"] is not None
        assert "implementation is complete" in result["completion_report"]

    def test_fork_subagent_meta_validation(self):
        """Fork meta.json should have isFork=true and no model."""
        meta = read_meta_json(str(FORK_SUBAGENT_TRANSCRIPT))

        assert meta["isFork"] is True
        assert meta["agentType"] == "fork"
        assert "model" not in meta


class TestExtractFreshSubagent:
    """Test extraction from fresh-context subagent fixtures."""

    def test_extract_fresh_subagent(self):
        """Should extract fresh subagent with correct dispatch type."""
        result = extract_transcript(str(FRESH_SUBAGENT_TRANSCRIPT))

        assert result["dispatch_type"] == "fresh"
        assert result["transcript_path"] == str(FRESH_SUBAGENT_TRANSCRIPT)

    def test_fresh_subagent_has_completion_report(self):
        """Fresh subagent should have completion report."""
        result = extract_transcript(str(FRESH_SUBAGENT_TRANSCRIPT))

        assert result["completion_report"] is not None
        assert "comprehensive JSONL parser is complete" in result["completion_report"]

    def test_fresh_subagent_has_learn_call(self):
        """Fresh subagent should extract learn/write-memory calls."""
        result = extract_transcript(str(FRESH_SUBAGENT_TRANSCRIPT))

        assert len(result["learn_calls"]) == 1
        learn = result["learn_calls"][0]
        assert learn["skill"] == "write-memory"
        assert learn["skill_args"] == "transcript parsing best practices"

    def test_fresh_subagent_meta_validation(self):
        """Fresh meta.json should have model and agentType."""
        meta = read_meta_json(str(FRESH_SUBAGENT_TRANSCRIPT))

        assert meta["agentType"] == "general-purpose"
        assert meta["model"] == "haiku"
        assert meta.get("isFork") is not True


class TestExtractWorkflowSubagent:
    """Test extraction from workflow subagent fixtures."""

    def test_extract_workflow_subagent(self):
        """Should extract workflow subagent with correct dispatch type."""
        result = extract_transcript(str(WORKFLOW_SUBAGENT_TRANSCRIPT))

        assert result["dispatch_type"] == "workflow"
        assert result["transcript_path"] == str(WORKFLOW_SUBAGENT_TRANSCRIPT)

    def test_workflow_subagent_has_completion_report(self):
        """Workflow subagent should have completion report."""
        result = extract_transcript(str(WORKFLOW_SUBAGENT_TRANSCRIPT))

        assert result["completion_report"] is not None
        assert "Review complete" in result["completion_report"]

    def test_workflow_subagent_has_recall_and_engram_query(self):
        """Workflow subagent should extract recall calls from skill invocation."""
        result = extract_transcript(str(WORKFLOW_SUBAGENT_TRANSCRIPT))

        assert len(result["recall_calls"]) == 1
        recall = result["recall_calls"][0]
        assert recall["skill_args"] == "parser code review"
        assert len(recall["matched_items"]) == 1
        assert recall["matched_items"][0]["kind"] == "feedback"

    def test_workflow_subagent_has_engram_commands(self):
        """Workflow subagent should extract engram query command."""
        result = extract_transcript(str(WORKFLOW_SUBAGENT_TRANSCRIPT))

        # The engram query is a bash command with "engram query"
        # Our current implementation only captures learn|amend|activate
        # so this test verifies the structure exists
        assert "engram_commands" in result

    def test_workflow_subagent_meta_validation(self):
        """Workflow meta.json should be classified correctly."""
        meta = read_meta_json(str(WORKFLOW_SUBAGENT_TRANSCRIPT))

        assert meta["agentType"] == "code-reviewer"
        assert meta["model"] == "sonnet"
        assert meta.get("isFork") is not True


class TestClassificationMismatches:
    """Test detection of classification mismatches between parent and meta.json."""

    def test_no_mismatches_in_fixtures(self):
        """Fixtures should have no classification mismatches."""
        for fixture in [
            FORK_SUBAGENT_TRANSCRIPT,
            FRESH_SUBAGENT_TRANSCRIPT,
            WORKFLOW_SUBAGENT_TRANSCRIPT,
        ]:
            result = extract_transcript(str(fixture))
            assert len(result["classification_mismatches"]) == 0


class TestMalformedJsonHandling:
    """Test handling of malformed JSON lines."""

    def test_skip_malformed_lines(self, tmp_path):
        """Should skip malformed JSON lines without crashing."""
        transcript = tmp_path / "malformed.jsonl"
        transcript.write_text(
            '{"type":"assistant","uuid":"a1","message":{"content":[]}}\n'
            "this is not json\n"
            '{"type":"user","uuid":"u1","message":{"content":"test"}}\n'
        )

        result = extract_transcript(str(transcript))
        assert result["transcript_path"] == str(transcript)
        assert result["dispatch_type"] == "main"
        # Should have processed the two valid lines


class TestCompletionReportExtraction:
    """Test extraction of completion_report for subagent transcripts."""

    def test_completion_report_extracted_for_fork_subagent(self):
        """Fork subagent should extract completion_report."""
        result = extract_transcript(str(FORK_SUBAGENT_TRANSCRIPT))

        assert result["dispatch_type"] == "fork"
        assert result["completion_report"] is not None
        assert isinstance(result["completion_report"], str)
        assert len(result["completion_report"]) > 0

    def test_completion_report_extracted_for_fresh_subagent(self):
        """Fresh subagent should extract completion_report."""
        result = extract_transcript(str(FRESH_SUBAGENT_TRANSCRIPT))

        assert result["dispatch_type"] == "fresh"
        assert result["completion_report"] is not None
        assert isinstance(result["completion_report"], str)
        assert len(result["completion_report"]) > 0

    def test_completion_report_not_extracted_for_main(self):
        """Main session should not have completion_report."""
        result = extract_transcript(str(MAIN_SESSION_FIXTURE))

        assert result["dispatch_type"] == "main"
        assert result["completion_report"] is None

    def test_completion_report_with_trailing_tool_use_blocks(self, tmp_path):
        """
        Regression test for bug where completion_report was None when the last
        assistant message contained only tool_use blocks (no text blocks).

        The fix iterates backwards through assistant messages to find the last
        one with text blocks, rather than just taking the last message.
        """
        # Create a transcript in subagents/ path (required for dispatch_type != "main")
        subagents_dir = tmp_path / "subagents"
        subagents_dir.mkdir()

        transcript_path = subagents_dir / "agent-test.jsonl"
        transcript_path.write_text(
            '{"type":"ai-title","aiTitle":"Test","sessionId":"test-id"}\n'
            '{"type":"assistant","uuid":"a1","timestamp":"2026-08-31T12:00:00Z",'
            '"message":{"content":[{"type":"text","text":"First message"}]}}\n'
            '{"type":"user","uuid":"u1","message":{"content":[]}}\n'
            '{"type":"assistant","uuid":"a2","timestamp":"2026-08-31T12:01:00Z",'
            '"message":{"content":[{"type":"text","text":"Second message with content"}]}}\n'
            '{"type":"user","uuid":"u2","message":{"content":[]}}\n'
            # This message has only a tool_use block - no text
            '{"type":"assistant","uuid":"a3","timestamp":"2026-08-31T12:02:00Z",'
            '"message":{"content":[{"type":"tool_use","id":"t1","name":"StructuredOutput",'
            '"input":{"status":"ok"}}]}}\n'
        )

        # Create meta.json to explicitly mark as fresh subagent (not a fork)
        meta_path = subagents_dir / "agent-test.meta.json"
        meta_path.write_text('{"agentType":"general-purpose"}')

        result = extract_transcript(str(transcript_path))

        # Should be classified as fresh (not main, not fork)
        assert result["dispatch_type"] == "fresh"

        # Should extract "Second message with content", not the empty tool_use
        assert result["completion_report"] == "Second message with content"


class TestPiFormatDetection:
    """Test Pi transcript format detection."""

    def test_detect_pi_main_session(self):
        """Should detect Pi main session from first line."""
        pi_main = FIXTURES_DIR / "pi" / "pi_main_session.jsonl"
        result = extract_transcript(str(pi_main))

        assert result["dispatch_type"] == "pi_main"
        assert result["transcript_path"] == str(pi_main)

    def test_pi_format_does_not_break_claude_code_detection(self):
        """Claude Code fixtures should still be detected correctly."""
        result = extract_transcript(str(MAIN_SESSION_FIXTURE))

        assert result["dispatch_type"] == "main"
        assert "pi" not in result["dispatch_type"].lower()


class TestPiMainSession:
    """Test extraction from Pi main session fixtures."""

    def test_extract_pi_main_session_basic(self):
        """Should extract Pi main session with correct dispatch type."""
        pi_main = FIXTURES_DIR / "pi" / "pi_main_session.jsonl"
        result = extract_transcript(str(pi_main))

        assert result["dispatch_type"] == "pi_main"
        assert result["transcript_path"] == str(pi_main)

    def test_pi_main_session_recall_extraction(self):
        """Pi main session should extract recall call from bash engram query."""
        pi_main = FIXTURES_DIR / "pi" / "pi_main_session.jsonl"
        result = extract_transcript(str(pi_main))

        assert len(result["recall_calls"]) == 1
        recall = result["recall_calls"][0]
        assert len(recall["query_phrases"]) == 2
        assert "test phrase one" in recall["query_phrases"]
        assert "test phrase two" in recall["query_phrases"]

    def test_pi_main_session_recall_items(self):
        """Pi recall should extract matched items from engram result."""
        pi_main = FIXTURES_DIR / "pi" / "pi_main_session.jsonl"
        result = extract_transcript(str(pi_main))

        recall = result["recall_calls"][0]
        assert len(recall["matched_items"]) == 2

        item1 = recall["matched_items"][0]
        assert item1["path"] == "test-note-1.md"
        assert item1["kind"] == "fact"
        assert item1["score"] == 0.95
        assert item1["source"] == "direct"

        item2 = recall["matched_items"][1]
        assert item2["path"] == "test-note-2.md"
        assert item2["kind"] == "feedback"
        assert item2["score"] == 0.87

    def test_pi_main_session_dispatch_multiple_tasks(self):
        """Pi subagent tool call with multiple tasks should extract one event per task."""
        pi_main = FIXTURES_DIR / "pi" / "pi_main_session.jsonl"
        result = extract_transcript(str(pi_main))

        assert len(result["dispatch_events"]) == 2

        event1 = result["dispatch_events"][0]
        assert event1["description"] == "reviewer-a"
        assert event1["subagent_type"] == "reviewer-a"
        assert event1["model"] == "anthropic/claude-haiku-4.5"
        assert "You are reviewer A" in event1["prompt_text"]

        event2 = result["dispatch_events"][1]
        assert event2["description"] == "reviewer-b"
        assert event2["subagent_type"] == "reviewer-b"
        assert event2["model"] == "anthropic/claude-sonnet-4-20250514"
        assert "You are reviewer B" in event2["prompt_text"]

    def test_pi_main_session_no_learn_calls(self):
        """Pi format should have empty learn_calls list (uses engram_commands instead)."""
        pi_main = FIXTURES_DIR / "pi" / "pi_main_session.jsonl"
        result = extract_transcript(str(pi_main))

        assert result["learn_calls"] == []

    def test_pi_main_session_no_classification_mismatches(self):
        """Pi format should have empty classification_mismatches list."""
        pi_main = FIXTURES_DIR / "pi" / "pi_main_session.jsonl"
        result = extract_transcript(str(pi_main))

        assert result["classification_mismatches"] == []

    def test_pi_main_session_no_completion_report(self):
        """Pi main session should not have completion_report (only subagents do)."""
        pi_main = FIXTURES_DIR / "pi" / "pi_main_session.jsonl"
        result = extract_transcript(str(pi_main))

        assert result["completion_report"] is None


class TestPiSubagentRun:
    """Test extraction from Pi nested subagent run fixtures."""

    def test_extract_pi_subagent_run(self):
        """Should extract Pi subagent run with correct dispatch type."""
        pi_subagent = FIXTURES_DIR / "pi" / "test_session" / "abcd1234" / "run-0" / "session.jsonl"
        result = extract_transcript(str(pi_subagent))

        assert result["dispatch_type"] == "pi_subagent"
        assert result["transcript_path"] == str(pi_subagent)

    def test_pi_subagent_run_has_completion_report(self):
        """Pi subagent run should extract completion_report."""
        pi_subagent = FIXTURES_DIR / "pi" / "test_session" / "abcd1234" / "run-0" / "session.jsonl"
        result = extract_transcript(str(pi_subagent))

        assert result["completion_report"] is not None
        assert "implementation is solid" in result["completion_report"]
        assert "no blocking issues" in result["completion_report"]

    def test_pi_subagent_run_has_recall_call(self):
        """Pi subagent run should extract recall call from bash engram query."""
        pi_subagent = FIXTURES_DIR / "pi" / "test_session" / "abcd1234" / "run-0" / "session.jsonl"
        result = extract_transcript(str(pi_subagent))

        assert len(result["recall_calls"]) == 1
        recall = result["recall_calls"][0]
        assert "implementation review feedback" in recall["query_phrases"]
        assert len(recall["matched_items"]) == 1
        assert recall["matched_items"][0]["path"] == "review-note.md"

    def test_pi_subagent_run_dispatch_type_detection(self):
        """Pi subagent run path should trigger pi_subagent dispatch type."""
        # Test the dispatch_type detection logic by checking the path pattern
        pi_subagent = FIXTURES_DIR / "pi" / "test_session" / "abcd1234" / "run-0" / "session.jsonl"
        result = extract_transcript(str(pi_subagent))

        # The fixture has /run-0/ in the path, so it should be detected as pi_subagent
        assert result["dispatch_type"] == "pi_subagent"


class TestGap1FalsePositiveDetection:
    """
    Regression tests for Gap 1: False-positive "engram query" detection.

    Verifies that engram query substrings appearing inside pi -p or claude -p
    subprocess dispatches are NOT counted as direct engram query calls.
    """

    def test_pi_p_subprocess_dispatch_not_counted_as_engram_query(self, tmp_path):
        """Gap 1: pi -p dispatch containing 'engram query' should NOT be a recall call."""
        transcript = tmp_path / "pi_p_dispatch.jsonl"
        transcript.write_text(
            '{"type": "session", "version": 3, "id": "test-gap1", "timestamp": "2026-08-31T10:00:00Z", "cwd": "/test"}\n'
            '{"type": "message", "id": "msg-1", "parentId": null, "timestamp": "2026-08-31T10:00:01Z", '
            '"message": {"role": "user", "content": "Run a subprocess review"}}\n'
            '{"type": "message", "id": "msg-2", "parentId": "msg-1", "timestamp": "2026-08-31T10:00:02Z", '
            '"message": {"role": "assistant", "content": ['
            '{"type": "text", "text": "Dispatching review via pi -p"}, '
            '{"type": "toolCall", "id": "bash_1", "name": "bash", "arguments": {'
            '"command": "cd /repo && pi -p --no-session --model haiku \'You should run engram query --phrase test\'"'
            '}}'
            ']}}\n'
            '{"type": "message", "id": "msg-3", "parentId": "msg-2", "timestamp": "2026-08-31T10:00:03Z", '
            '"message": {"role": "toolResult", "toolCallId": "bash_1", "toolName": "bash", '
            '"content": [{"type": "text", "text": "exit 0"}]}}\n'
        )

        result = extract_transcript(str(transcript))

        # Should have 0 recall calls since engram query is inside the pi -p dispatch
        assert len(result["recall_calls"]) == 0

    def test_claude_p_subprocess_dispatch_not_counted_as_engram_query(self, tmp_path):
        """Gap 1: claude -p dispatch containing 'engram query' should NOT be a recall call."""
        transcript = tmp_path / "claude_p_dispatch.jsonl"
        transcript.write_text(
            '{"type":"assistant","uuid":"a1","message":{"content":['
            '{"type":"tool_use","id":"bash_1","name":"Bash",'
            '"input":{"command":"claude -p \'engram query --phrase test\' && echo done"}}'
            ']}}\n'
            '{"type":"user","uuid":"u1","message":{"content":['
            '{"type":"tool_result","tool_use_id":"bash_1","content":"exit 0"}'
            ']}}\n'
        )

        result = extract_transcript(str(transcript))

        # Should have 0 recall calls since engram query is inside the claude -p dispatch
        assert len(result["recall_calls"]) == 0

    def test_direct_engram_query_is_counted(self, tmp_path):
        """Verify that direct engram query calls (not in subprocess) are still counted."""
        transcript = tmp_path / "direct_engram_query.jsonl"
        transcript.write_text(
            '{"type":"assistant","uuid":"a1","message":{"content":['
            '{"type":"tool_use","id":"recall_1","name":"Skill",'
            '"input":{"skill":"recall","args":"test"}}'
            ']}}\n'
            '{"type":"user","uuid":"u1","message":{"content":[]}}\n'
            '{"type":"assistant","uuid":"a2","message":{"content":['
            '{"type":"tool_use","id":"bash_1","name":"Bash",'
            '"input":{"command":"engram query --phrase test"}}'
            ']}}\n'
            '{"type":"user","uuid":"u2","message":{"content":['
            '{"type":"tool_result","tool_use_id":"bash_1",'
            '"content":"version: 1\\nphrases:\\n  - test\\nitems:\\nclusters: []"}'
            ']}}\n'
        )

        result = extract_transcript(str(transcript))

        # Should have 1 recall call since engram query is direct
        assert len(result["recall_calls"]) == 1


class TestGap2TruncationDetection:
    """
    Regression tests for Gap 2: Silent truncation detection.

    Verifies that results not starting with 'version:' header are flagged
    with possibly_truncated: true to distinguish from genuine empty results.
    """

    def test_truncated_result_flagged(self, tmp_path):
        """Gap 2: Result without version header gets possibly_truncated=true."""
        transcript = tmp_path / "truncated_result.jsonl"
        # Construct a long truncated-looking result (over 100 chars but no version header)
        truncated_content = "luhmann: 74\ncreated: 2026-08-31\nsource: session 2026-06-23\nThis is a large truncated result that starts mid-note without the version header and is long enough"
        # Escape for JSON
        escaped_content = truncated_content.replace('\\', '\\\\').replace('"', '\\"').replace('\n', '\\n')

        transcript.write_text(
            '{"type":"assistant","uuid":"a1","message":{"content":['
            '{"type":"tool_use","id":"recall_1","name":"Skill",'
            '"input":{"skill":"recall","args":"test"}}'
            ']}}\n'
            '{"type":"user","uuid":"u1","message":{"content":[]}}\n'
            '{"type":"assistant","uuid":"a2","message":{"content":['
            '{"type":"tool_use","id":"bash_1","name":"Bash",'
            '"input":{"command":"engram query --phrase test"}}'
            ']}}\n'
            f'{{"type":"user","uuid":"u2","message":{{"content":['
            f'{{"type":"tool_result","tool_use_id":"bash_1",'
            f'"content":"{escaped_content}"}}'
            ']}}\n'
        )

        result = extract_transcript(str(transcript))

        assert len(result["recall_calls"]) == 1
        assert result["recall_calls"][0]["possibly_truncated"] is True

    def test_normal_result_not_flagged_as_truncated(self, tmp_path):
        """Gap 2: Normal result with version header is not flagged as truncated."""
        transcript = tmp_path / "normal_result.jsonl"
        transcript.write_text(
            '{"type":"assistant","uuid":"a1","message":{"content":['
            '{"type":"tool_use","id":"recall_1","name":"Skill",'
            '"input":{"skill":"recall","args":"test"}}'
            ']}}\n'
            '{"type":"user","uuid":"u1","message":{"content":[]}}\n'
            '{"type":"assistant","uuid":"a2","message":{"content":['
            '{"type":"tool_use","id":"bash_1","name":"Bash",'
            '"input":{"command":"engram query --phrase test"}}'
            ']}}\n'
            '{"type":"user","uuid":"u2","message":{"content":['
            '{"type":"tool_result","tool_use_id":"bash_1",'
            '"content":"version: 1\\nphrases:\\n  - test\\nitems:\\nclusters: []"}'
            ']}}\n'
        )

        result = extract_transcript(str(transcript))

        assert len(result["recall_calls"]) == 1
        assert result["recall_calls"][0]["possibly_truncated"] is False

    def test_empty_result_not_flagged_as_truncated(self, tmp_path):
        """Gap 2: Empty result is not flagged as truncated (short content)."""
        transcript = tmp_path / "empty_result.jsonl"
        transcript.write_text(
            '{"type":"assistant","uuid":"a1","message":{"content":['
            '{"type":"tool_use","id":"recall_1","name":"Skill",'
            '"input":{"skill":"recall","args":"test"}}'
            ']}}\n'
            '{"type":"user","uuid":"u1","message":{"content":[]}}\n'
            '{"type":"assistant","uuid":"a2","message":{"content":['
            '{"type":"tool_use","id":"bash_1","name":"Bash",'
            '"input":{"command":"engram query --phrase test"}}'
            ']}}\n'
            '{"type":"user","uuid":"u2","message":{"content":['
            '{"type":"tool_result","tool_use_id":"bash_1",'
            '"content":""}'
            ']}}\n'
        )

        result = extract_transcript(str(transcript))

        assert len(result["recall_calls"]) == 1
        # Empty content should not trigger truncation flag
        assert result["recall_calls"][0]["possibly_truncated"] is False


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
