# Hand Verification of extract_transcript() — Task 2.2

**Date:** 2026-08-31  
**Verified against:** 4 real Claude Code transcripts (main session + 3 dispatch types)  
**Verification scope:** Independent field-by-field comparison against raw JSONL

## Summary

All 4 dispatch types verified against real transcripts. **One bug found and fixed:** completion_report extraction failed when the last assistant message contained only tool_use blocks (no text). Fix: iterate backwards to find the last message with text blocks.

---

## Files Verified

### File 1: Main Session
**Path:** `/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/95d68770-85ed-489f-874f-f576f32d0630.jsonl`  
**Dispatch type:** main (verified ✓)

| Field | Extracted | Manual Count | Match |
|-------|-----------|--------------|-------|
| `dispatch_type` | main | main | ✓ |
| `recall_calls` | 2 | 2 | ✓ |
| `dispatch_events` | 28 | 28 | ✓ |
| `learn_calls` | 5 | 5 | ✓ |
| `engram_commands` | 22 | 22 | ✓ |
| `completion_report` | None | N/A (main) | ✓ |

**Spot checks:**
- **dispatch_events[0]:** description="Bump targ pin, run gates...", subagent_type="general-purpose", model="haiku", prompt_text=3087 chars (full, not truncated) ✓
- **recall_calls[0]:** location=36, skill_args="resume #700 cleanup...", query_phrases count=10, matched_items count=10 ✓
- **recall_calls[0] query parsing:** Manually traced Bash tool_use (line 108) to tool_result (line 109); phrase text matches exactly; matched_items ranks 1-10 match extracted values ✓

---

### File 2: Fork Subagent
**Path:** `/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/3b5def4d-a702-417e-afc7-40547be72d43/subagents/agent-a16b99dfdb4969c0d.jsonl`  
**Dispatch type:** fork (verified via meta.json isFork=true ✓)

| Field | Extracted | Manual Count | Match |
|-------|-----------|--------------|-------|
| `dispatch_type` | fork | fork | ✓ |
| `recall_calls` | 0 | 0 | ✓ |
| `dispatch_events` | 1 | 1 | ✓ |
| `learn_calls` | 0 | 0 | ✓ |
| `engram_commands` | 0 | 0 | ✓ |
| `completion_report` | 2358 chars | 2358 chars | ✓ |

**Spot checks:**
- **completion_report:** Last assistant message (line 115) — text matches extracted verbatim (full message, not truncated) ✓
- **dispatch_events[0]:** Correctly extracted (1 Agent call) ✓

---

### File 3: Fresh-Context Subagent
**Path:** `/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/95d68770-85ed-489f-874f-f576f32d0630/subagents/agent-a0200fba0bc96b198.jsonl`  
**Dispatch type:** fresh (verified via path `/subagents/` and meta.json agentType="general-purpose" ✓)

| Field | Extracted | Manual Count | Match |
|-------|-----------|--------------|-------|
| `dispatch_type` | fresh | fresh | ✓ |
| `recall_calls` | 0 | 0 | ✓ |
| `dispatch_events` | 0 | 0 | ✓ |
| `learn_calls` | 0 | 0 | ✓ |
| `engram_commands` | 0 | 0 | ✓ |
| `completion_report` | 490 chars | 490 chars | ✓ |

**Spot checks:**
- **completion_report:** Last assistant message (line 127) — text matches extracted verbatim ✓
- This file confirms **fresh-context subagents exist in real corpus** — resolves task 1.2 open question ("may not exist") ✓

---

### File 4: Workflow Subagent
**Path:** `/Users/joe/.claude/projects/-Users-joe-repos-personal-engram/8a544d16-5d1c-4d29-bfea-7ade7cf98f81/subagents/workflows/wf_2400b04e-5bb/agent-a033d745bcd5f7119.jsonl`  
**Dispatch type:** workflow (verified via path `/subagents/workflows/` ✓)

| Field | Extracted | Manual Count | Match |
|-------|-----------|--------------|-------|
| `dispatch_type` | workflow | workflow | ✓ |
| `recall_calls` | 0 | 0 | ✓ |
| `dispatch_events` | 0 | 0 | ✓ |
| `learn_calls` | 0 | 0 | ✓ |
| `engram_commands` | 4 | 4 | ✓ |
| `completion_report` | 1226 chars | 1226 chars | ✓ |

**Spot check (found and fixed bug here):**
- **completion_report:** The last assistant message (line 66) contained only a StructuredOutput tool_use block with NO text blocks. Prior code would extract None/missing. **After fix:** iterated backwards and found message at line 62 with text "All steps completed successfully..." — now correctly extracted ✓

---

## Bug Found and Fixed

### Issue: completion_report extraction returns None when last assistant message has only tool_use blocks

**Location:** `extract.py` lines 411-425  
**Root cause:** Code was taking the *last* assistant message regardless of content type, then looking for text blocks in it. If the last message contained only tool_use/other blocks (no text), completion_report would be None even though earlier messages had text.

**Real-world trigger:** File 4 (workflow) — last assistant message (line 66) was a StructuredOutput tool_use. The intended completion_report text was in message at line 62.

**Fix:** Changed logic to iterate **backwards** through assistant_lines to find the **last message containing text blocks**, rather than blindly taking the last message.

**Code change:**
```python
# BEFORE (lines 411-425):
if dispatch_type != "main" and assistant_lines:
    last_assistant_line_num, last_assistant_event = assistant_lines[-1]
    message = last_assistant_event.get("message", {})
    content = message.get("content", [])
    completion_parts = []
    for block in content:
        if isinstance(block, dict) and block.get("type") == "text":
            completion_parts.append(block.get("text", ""))
    if completion_parts:
        result["completion_report"] = "".join(completion_parts)

# AFTER:
if dispatch_type != "main" and assistant_lines:
    completion_report = None
    for line_num, event in reversed(assistant_lines):
        message = event.get("message", {})
        content = message.get("content", [])
        completion_parts = []
        for block in content:
            if isinstance(block, dict) and block.get("type") == "text":
                completion_parts.append(block.get("text", ""))
        if completion_parts:
            completion_report = "".join(completion_parts)
            break
    if completion_report is not None:
        result["completion_report"] = completion_report
```

**Regression test added:** `test_extract.py` line ~470 — `test_completion_report_with_trailing_tool_use_blocks()` creates a scenario where the last assistant message has only a tool_use block and verifies the fix finds the text in an earlier message.

---

## Test Results

```
python3 -m pytest dev/eval/audit/test_extract.py -v

============================= test session starts ==============================
collected 37 items

dev/eval/audit/test_extract.py::TestNormalizeToolResultContent::test_bare_string_content PASSED
dev/eval/audit/test_extract.py::TestNormalizeToolResultContent::test_list_of_text_blocks PASSED
dev/eval/audit/test_extract.py::TestNormalizeToolResultContent::test_mixed_block_types PASSED
dev/eval/audit/test_extract.py::TestParseEngramQueryResult::test_parse_single_item PASSED
dev/eval/audit/test_extract.py::TestParseEngramQueryResult::test_parse_multiple_items PASSED
dev/eval/audit/test_extract.py::TestParseEngramQueryResult::test_empty_items PASSED
dev/eval/audit/test_extract.py::TestDetermineDispatchType::test_main_session PASSED
dev/eval/audit/test_extract.py::TestDetermineDispatchType::test_fork_with_meta PASSED
dev/eval/audit/test_extract.py::TestDetermineDispatchType::test_fork_with_agenttype PASSED
dev/eval/audit/test_extract.py::TestDetermineDispatchType::test_fresh_subagent PASSED
dev/eval/audit/test_extract.py::TestDetermineDispatchType::test_workflow_by_path PASSED
dev/eval/audit/test_extract.py::TestReadMetaJson::test_read_existing_meta PASSED
dev/eval/audit/test_extract.py::TestReadMetaJson::test_missing_meta_returns_none PASSED
dev/eval/audit/test_extract.py::TestExtractMainSession::test_extract_main_session PASSED
dev/eval/audit/test_extract.py::TestExtractMainSession::test_main_session_has_recall_call PASSED
dev/eval/audit/test_extract.py::TestExtractMainSession::test_main_session_recall_items PASSED
dev/eval/audit/test_extract.py::TestExtractMainSession::test_main_session_has_dispatch_event PASSED
dev/eval/audit/test_extract.py::TestRecallWindowClosesOnOtherSkills::test_other_skill_engram_query_does_not_contaminate_recall_call PASSED
dev/eval/audit/test_extract.py::TestRecallWindowDoesNotCloseOnWriteMemory::test_write_memory_does_not_close_recall_window PASSED
dev/eval/audit/test_extract.py::TestExtractForkedSubagent::test_extract_fork_subagent PASSED
dev/eval/audit/test_extract.py::TestExtractForkedSubagent::test_fork_subagent_has_completion_report PASSED
dev/eval/audit/test_extract.py::TestExtractForkedSubagent::test_fork_subagent_meta_validation PASSED
dev/eval/audit/test_extract.py::TestExtractFreshSubagent::test_extract_fresh_subagent PASSED
dev/eval/audit/test_extract.py::TestExtractFreshSubagent::test_fresh_subagent_has_completion_report PASSED
dev/eval/audit/test_extract.py::TestExtractFreshSubagent::test_fresh_subagent_has_learn_call PASSED
dev/eval/audit/test_extract.py::TestExtractFreshSubagent::test_fresh_subagent_meta_validation PASSED
dev/eval/audit/test_extract.py::TestExtractWorkflowSubagent::test_extract_workflow_subagent PASSED
dev/eval/audit/test_extract.py::TestExtractWorkflowSubagent::test_workflow_subagent_has_completion_report PASSED
dev/eval/audit/test_extract.py::TestExtractWorkflowSubagent::test_workflow_subagent_has_recall_and_engram_query PASSED
dev/eval/audit/test_extract.py::TestExtractWorkflowSubagent::test_workflow_subagent_has_engram_commands PASSED
dev/eval/audit/test_extract.py::TestExtractWorkflowSubagent::test_workflow_subagent_meta_validation PASSED
dev/eval/audit/test_extract.py::TestClassificationMismatches::test_no_mismatches_in_fixtures PASSED
dev/eval/audit/test_extract.py::TestMalformedJsonHandling::test_skip_malformed_lines PASSED
dev/eval/audit/test_extract.py::TestCompletionReportExtraction::test_completion_report_extracted_for_fork_subagent PASSED
dev/eval/audit/test_extract.py::TestCompletionReportExtraction::test_completion_report_extracted_for_fresh_subagent PASSED
dev/eval/audit/test_extract.py::TestCompletionReportExtraction::test_completion_report_not_extracted_for_main PASSED
dev/eval/audit/test_extract.py::TestCompletionReportExtraction::test_completion_report_with_trailing_tool_use_blocks PASSED

============================== 37 passed in 0.03s ===============================
```

---

## Verification Completeness

✓ **All 4 dispatch types verified** (main, fork, fresh, workflow)  
✓ **All 4 files are real transcripts** (not synthetic fixtures)  
✓ **All major extracted fields spot-checked** (dispatch_type, recall_calls, dispatch_events, learn_calls, engram_commands, completion_report)  
✓ **Recall queries manually traced** (Bash tool_use → tool_result mapping for File 1)  
✓ **Completion reports verified verbatim** (full text, not truncated)  
✓ **One bug found and fixed** (completion_report with trailing tool_use blocks)  
✓ **Regression test added** (37 total tests, all passing)  
✓ **Fresh-context subagent existence confirmed** (File 3 resolves task 1.2 open question)

