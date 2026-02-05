# TDD Red QA Review - TASK-4 Iteration 2

## Decision: APPROVED ✓

The test suite produced by tdd-red-producer iteration 2 is **approved** and ready for the green phase. All acceptance criteria are covered with appropriate tests that fail for the correct reasons.

## Executive Summary

**Artifact Reviewed:** `/Users/joe/.claude/skills/arch-interview-producer/SKILL_test_gather_v2.sh`

**Test Suite Status:**
- **Total Tests:** 13
- **Passing:** 2 (structure validation - acceptable)
- **Failing:** 11 (red state confirmed - correct)
- **Coverage:** 7/7 acceptance criteria covered

**Key Finding:** Tests successfully distinguish between existing structure (which validates correctly) and new content requirements (which fail correctly). This is the expected red state.

## Red Phase Checklist - All Items Passed

✓ **Each acceptance criterion has at least one test**
  - All 7 ACs covered with 1-3 dedicated tests each

✓ **Tests fail for correct reasons (missing implementation)**
  - All 11 failures are due to missing instructions in SKILL.md
  - No compilation, syntax, or import errors

✓ **Tests don't fail due to syntax/import/compilation errors**
  - Test suite executes cleanly with proper bash error handling

✓ **Tests describe expected behavior clearly**
  - Failure messages explain WHAT is missing and WHY it matters
  - Example: "FAIL: Territory map purpose not explained. Instruction should say WHAT territory map provides (file structure, artifacts, etc.)"

✓ **Property tests used for invariants**
  - `test_gather_execution_order` verifies the property: territory < memory < parse ordering must hold

✓ **Tests are blackbox (test public API)**
  - Tests examine SKILL.md documentation (the Claude instruction API)
  - No internal implementation testing

✓ **No implementation code beyond minimal stubs**
  - Tests only validate documentation structure and content
  - Target SKILL.md unchanged (correct red state)

## Acceptance Criteria Coverage

| AC | Description | Tests | Status |
|----|-------------|-------|--------|
| AC-1 | Code runs before yielding need-user-input | test_gather_phase_structure<br>test_gather_precedes_interview | ✓ Covered (partial pass on structure, fail on content) |
| AC-2 | Executes projctl territory map | test_territory_map_instruction_completeness<br>test_gather_execution_order | ✓ Covered (fails correctly) |
| AC-3 | Executes projctl memory query | test_memory_query_instruction_completeness<br>test_gather_execution_order | ✓ Covered (fails correctly) |
| AC-4 | Parses results into structured data | test_result_parsing_instructions | ✓ Covered (fails correctly) |
| AC-5 | Handles errors appropriately | test_error_handling_territory_completeness<br>test_error_handling_memory_completeness<br>test_error_handling_references_yield_protocol | ✓ Covered (fails correctly) |
| AC-6 | Logs context sources used | test_context_logging_instructions | ✓ Covered (fails correctly) |
| AC-7 | Context stored in yield metadata | test_yield_metadata_includes_sources | ✓ Covered (fails correctly) |

**Additional Coverage:**
- Integration test: `test_gather_workflow_completeness`
- Traceability test: `test_references_interview_pattern`

## Test Execution Results

### Passing Tests (2) - Acceptable Structure Validation

These tests pass because the GATHER section framework already exists from the original SKILL.md. This is the correct baseline.

1. **test_gather_phase_structure** ✓
   - Validates: GATHER has numbered steps (found 5)
   - Acceptable: Confirms existing framework is correct
   - Next: Content tests verify new requirements

2. **test_gather_precedes_interview** ✓
   - Validates: GATHER comes before SYNTHESIZE in workflow
   - Acceptable: Confirms correct section ordering
   - Next: Content tests verify what GATHER should do

### Failing Tests (11) - Correct Red State

All failures are expected and indicate missing TASK-4 implementation:

1. **test_territory_map_instruction_completeness** ✗
   - Missing: Command and purpose for territory map
   - Needs: `projctl territory map` + explanation of file structure output

2. **test_memory_query_instruction_completeness** ✗
   - Missing: Command and domain-specific query examples
   - Needs: `projctl memory query` + architecture-related examples

3. **test_result_parsing_instructions** ✗
   - Missing: Parsing step in workflow
   - Needs: Instructions for parsing territory/memory results

4. **test_error_handling_territory_completeness** ✗
   - Missing: Territory failure → blocked mapping with rationale
   - Needs: WHEN to yield blocked and WHY (infrastructure problem)

5. **test_error_handling_memory_completeness** ✗
   - Missing: Memory timeout → continue mapping with degraded mode note
   - Needs: WHEN to continue and WHY (can proceed with partial context)

6. **test_error_handling_references_yield_protocol** ✗
   - Missing: References to YIELD.md or yield types
   - Needs: Link to yield protocol or mention of blocked/need-decision

7. **test_context_logging_instructions** ✗
   - Missing: Instructions for logging context sources
   - Needs: WHAT to log (territory, memory, files) and WHERE (metadata)

8. **test_yield_metadata_includes_sources** ✗
   - Missing: Context sources in yield examples
   - Needs: `[context]` section with `sources = ["territory", "memory", ...]`

9. **test_gather_execution_order** ✗
   - Missing: Required GATHER steps (territory/memory/parse)
   - Needs: Ordered instructions showing territory < memory < parse

10. **test_gather_workflow_completeness** ✗
    - Missing: Complete integration of all GATHER components
    - Needs: All steps present and properly ordered

11. **test_references_interview_pattern** ✗
    - Missing: Reference to INTERVIEW-PATTERN.md
    - Needs: Traceability link to shared pattern documentation

## Why These Failures Are Correct

**Current GATHER Phase** (lines 30-36 of SKILL.md):
```markdown
### GATHER Phase

1. Read context file for requirements.md and design.md paths
2. Yield `need-context` if files not provided in context
3. Extract technical implications from requirements
4. Identify decision categories (language, framework, database, etc.)
5. Yield `need-user-input` for each architecture decision
```

This represents the **OLD workflow** - direct user interview without context gathering from territory/memory.

**TASK-4 Requirements** add:
- Territory map execution
- Memory query execution with domain examples
- Result parsing into structured data
- Error handling with specific yield mappings
- Context source logging
- Yield metadata with sources
- Traceability to INTERVIEW-PATTERN.md

Tests correctly fail because these new instructions don't exist yet.

## QA Feedback Resolution (Iteration 1 → 2)

### TEST-1: Instruction Completeness ✓ RESOLVED

**Issue:** v1 tests checked for keywords, not instruction quality.

**Resolution:** v2 tests verify each instruction includes:
- WHAT to execute (command)
- WHAT to do with output (purpose/usage)
- WHEN to apply (workflow position)

**Example:**
```bash
# v1: Just keyword check
grep -q "projctl territory map"

# v2: Command + purpose
if ! echo "$gather_section" | grep -q "projctl territory map"; then
    echo "FAIL: Territory map command not documented"
fi
if ! echo "$gather_section" | grep -qE "(file structure|artifact|project structure)"; then
    echo "FAIL: Territory map purpose not explained"
fi
```

### TEST-2: Brittle Text Matching ✓ RESOLVED

**Issue:** v1 tests required exact phrases, breaking with synonyms.

**Resolution:** Use semantic categories with OR logic:
```bash
# v2: Semantic category with synonyms
grep -qE "(architecture|technology|system design|technical decision)"
```

### TEST-3: Actionable Instructions ✓ RESOLVED

**Issue:** v1 tests passed if keywords existed, not if instructions were actionable.

**Resolution:** Two-level verification: existence + rationale:
```bash
# Check error → yield mapping exists
if grep -A5 "territory.*fail" | grep -qi "blocked"; then
    # Verify rationale provided
    if grep -B10 -A10 "territory.*fail" | grep -qiE "(infrastructure|cannot proceed|critical)"; then
        echo "PASS: Territory failure → blocked with rationale"
    else
        echo "FAIL: Lacks rationale (WHY blocked?)"
    fi
fi
```

### TEST-4: AC-4 Scope Interpretation ✓ RESOLVED

**Issue:** AC-4 "Parses results into structured data" is behavioral, but SKILL.md is documentation.

**Resolution:** Test that parsing INSTRUCTIONS exist (prescriptive documentation):
```bash
if ! echo "$gather_section" | grep -qE "(parse|structured|extract)"; then
    echo "FAIL: No parsing step documented in GATHER"
    echo "Instruction should tell Claude to 'Parse results into structured data'"
fi
```

This correctly tests the documentation artifact rather than runtime behavior.

### TEST-5: Execution Order Verification ✓ RESOLVED

**Issue:** v1 test checked presence but not ordering.

**Resolution:** Added explicit ordering verification:
```bash
territory_line=$(echo "$gather_section" | grep -n "territory map" | cut -d: -f1)
memory_line=$(echo "$gather_section" | grep -n "memory query" | cut -d: -f1)
parse_line=$(echo "$gather_section" | grep -n -E "(parse|structured)" | cut -d: -f1)

# Validate: territory < memory < parse
if [[ "$territory_line" -gt "$memory_line" ]]; then
    echo "FAIL: Territory map should come BEFORE memory query"
fi
```

## Test Quality Assessment

### Strengths

1. **Clear Separation of Structure and Content**
   - Structure tests validate existing framework (pass)
   - Content tests verify new requirements (fail correctly)

2. **Three-Level Validation Pattern**
   - Level 1: Does element exist?
   - Level 2: Is it complete (command + purpose)?
   - Level 3: Is it actionable (with rationale)?

3. **Appropriate for Documentation Artifact**
   - Tests verify instructions are present and actionable
   - Correct scope for prescriptive documentation (SKILL.md)

4. **Informative Failure Messages**
   - Each failure explains WHAT is missing
   - Provides example of what SHOULD be there
   - Makes green phase implementation clear

5. **Strong Traceability**
   - Each test function comments link to specific AC
   - Clear mapping: test → AC → requirement

## Recommendations for Green Phase

### Critical Priorities

1. **Add GATHER Phase Content**
   - Territory map execution instructions (command + purpose)
   - Memory query execution instructions (command + domain examples + usage)
   - Result parsing instructions (what to extract from each source)

2. **Document Error Handling**
   - Territory failure → yield `blocked` with rationale (infrastructure problem, cannot proceed safely)
   - Memory timeout → continue with note (degraded mode, proceed with available context)
   - Reference YIELD.md or mention yield types explicitly

### Important Priorities

3. **Update Yield Examples**
   - Add `[context.sources]` metadata to `need-user-input` yield example
   - Show structure: `sources = ["territory", "memory", "context-files"]`

4. **Add Traceability**
   - Reference INTERVIEW-PATTERN.md in GATHER phase description
   - Links implementation to shared pattern documentation (TASK-1)

### Implementation Guidance

**Execution Order:** Territory map must come before memory query because file structure informs semantic search queries. Both results are then parsed together.

**Error Handling Philosophy:**
- Territory map failure is critical (infrastructure problem) → yield `blocked`
- Memory query timeout is degraded mode (can proceed) → continue with note

**Context Logging:** Document WHAT to log (which sources were used: territory, memory, files) and WHERE to log it (yield metadata, assessment results).

## Files

**Test Artifact (Approved):** `/Users/joe/.claude/skills/arch-interview-producer/SKILL_test_gather_v2.sh`
- 13 tests, 376 lines
- Executable, ready for use
- Red state confirmed (11 failing, 2 passing)

**Target Implementation:** `/Users/joe/.claude/skills/arch-interview-producer/SKILL.md`
- GATHER phase section (lines 30-36)
- Needs update to make tests pass

**Yield File:** `.claude/projects/ISSUE-061/yields/tdd-task4-qa-iteration2.toml`

## Next Steps

1. **Proceed to Green Phase** (tdd-green-producer)
   - Update SKILL.md GATHER section with required instructions
   - Run tests iteratively until all pass
   - Ensure implementation maintains numbered step structure

2. **Verify Test Passage**
   - All 13 tests should pass after implementation
   - No new failures introduced
   - Structure tests continue passing

3. **Advance to QA Phase** (tdd-green-qa)
   - Verify implementation completeness
   - Check that instructions are actionable
   - Validate yield metadata examples

## Summary

The iteration 2 test suite successfully addresses all QA feedback from iteration 1. Tests now verify **instruction completeness and actionability** rather than keyword presence. The red state is correctly confirmed with 11 tests failing due to missing TASK-4 implementation. All acceptance criteria have appropriate coverage.

**Approval Status:** ✓ APPROVED - Ready for green phase implementation
