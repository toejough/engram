# TDD Red Producer - TASK-4 Iteration 2 Result

## Summary

Successfully produced failing tests for TASK-4 (Context gathering phase) that verify **instruction completeness and actionability** rather than keyword presence. All QA feedback from iteration 1 has been addressed.

## Deliverables

### Primary Artifact
**File:** `/Users/joe/.claude/skills/arch-interview-producer/SKILL_test_gather_v2.sh`
- **Size:** 376 lines
- **Tests:** 13 test functions
- **State:** Red (11 failing, 2 passing)
- **Executable:** Yes (chmod +x applied)

### Supporting Documents
- **Yield:** `.claude/projects/ISSUE-61/yields/tdd-task4-producer-iteration2.toml`
- **This Result:** `.claude/projects/ISSUE-61/yields/tdd-task4-producer-iteration2-result.md`

## Test Execution Results

### Passing Tests (2)
These tests pass because the GATHER section structure already exists:

1. **test_gather_phase_structure** ✓
   - Verifies: GATHER has numbered steps (found 5)
   - Status: PASS (existing structure from original SKILL.md)

2. **test_gather_precedes_interview** ✓
   - Verifies: GATHER precedes SYNTHESIZE in workflow
   - Status: PASS (correct section ordering)

### Failing Tests (11) - RED STATE CONFIRMED ✓

All failures are expected and correct:

1. **test_territory_map_instruction_completeness** ✗
   - Error: "Territory map command not documented in GATHER"
   - Verifies: Command + purpose documented

2. **test_memory_query_instruction_completeness** ✗
   - Error: "Memory query command not documented in GATHER"
   - Verifies: Command + domain examples + usage

3. **test_result_parsing_instructions** ✗
   - Error: "No parsing step documented in GATHER"
   - Verifies: Instructions for parsing results exist

4. **test_error_handling_territory_completeness** ✗
   - Error: "Territory failure → blocked not documented"
   - Verifies: Error handling with WHEN and WHY

5. **test_error_handling_memory_completeness** ✗
   - Error: "Memory timeout handling not documented"
   - Verifies: Degraded mode handling documented

6. **test_error_handling_references_yield_protocol** ✗
   - Error: "Error handling doesn't reference yield types"
   - Verifies: YIELD.md or yield types mentioned

7. **test_context_logging_instructions** ✗
   - Error: "No context source logging instruction found"
   - Verifies: WHAT to log and WHERE

8. **test_yield_metadata_includes_sources** ✗
   - Error: "Yield examples lack context sources in [context] section"
   - Verifies: Yield examples include sources metadata

9. **test_gather_execution_order** ✗
   - Error: "Missing required GATHER steps (territory/memory/parse)"
   - Verifies: Correct execution order: territory < memory < parse

10. **test_gather_workflow_completeness** ✗
    - Error: "GATHER workflow incomplete"
    - Verifies: All required components present (integration test)

11. **test_references_interview_pattern** ✗
    - Error: "SKILL.md should reference INTERVIEW-PATTERN.md"
    - Verifies: Traceability to pattern documentation

## Why These Failures Are Correct

**Current GATHER Phase** (arch-interview-producer/SKILL.md lines 30-35):
```markdown
### GATHER Phase

1. Read context file for requirements.md and design.md paths
2. Yield `need-context` if files not provided in context
3. Extract technical implications from requirements
4. Identify decision categories (language, framework, database, etc.)
5. Yield `need-user-input` for each architecture decision
```

This represents the OLD workflow (direct user interview without context gathering from territory/memory).

**TASK-4 Requirements** (what needs to be ADDED):
- Territory map execution instructions
- Memory query execution instructions
- Result parsing instructions
- Error handling instructions (territory → blocked, memory → continue)
- Context source logging instructions
- Yield metadata examples with sources
- Reference to INTERVIEW-PATTERN.md

Tests correctly fail because the NEW instructions don't exist yet.

## Acceptance Criteria Coverage

| AC | Description | Tests | Status |
|----|-------------|-------|--------|
| AC-1 | Code runs before yielding need-user-input | test_gather_phase_structure<br>test_gather_precedes_interview | Partial (structure exists, content missing) |
| AC-2 | Executes projctl territory map | test_territory_map_instruction_completeness<br>test_gather_execution_order | Red ✓ |
| AC-3 | Executes projctl memory query | test_memory_query_instruction_completeness<br>test_gather_execution_order | Red ✓ |
| AC-4 | Parses results into structured data | test_result_parsing_instructions | Red ✓ |
| AC-5 | Handles errors | test_error_handling_territory_completeness<br>test_error_handling_memory_completeness<br>test_error_handling_references_yield_protocol | Red ✓ |
| AC-6 | Logs context sources | test_context_logging_instructions | Red ✓ |
| AC-7 | Context stored in yield metadata | test_yield_metadata_includes_sources | Red ✓ |

**Additional Coverage:**
- Integration: test_gather_workflow_completeness
- Traceability: test_references_interview_pattern

All 7 acceptance criteria have dedicated test coverage.

## QA Feedback Resolution

### Issue TEST-1: Instruction Completeness

**Problem:** v1 tests checked for keywords, not instruction quality.

**Solution:** v2 tests verify each instruction includes:
- WHAT to execute (command)
- WHAT to do with output (purpose/usage)
- WHEN to apply (workflow position)

**Example:**
```bash
# v1: Just check keyword
grep -q "projctl territory map" "$SKILL_FILE"

# v2: Check command + purpose
if ! echo "$gather_section" | grep -q "projctl territory map"; then
    echo "FAIL: Territory map command not documented"
    exit 1
fi

if ! echo "$gather_section" | grep -qE "(file structure|artifact|project structure)"; then
    echo "FAIL: Territory map purpose not explained"
    echo "Instruction should say WHAT territory map provides"
    exit 1
fi
```

### Issue TEST-2: Brittle Text Matching

**Problem:** v1 tests required exact phrases, breaking with synonyms.

**Solution:** Use semantic categories with OR logic:

```bash
# v1: Exact phrase
grep -qE "(architecture decisions|technology stack|system design)"

# v2: Semantic category with synonyms
grep -qE "(architecture|technology|system design|technical decision)"
```

### Issue TEST-3: Actionable Instructions

**Problem:** v1 tests passed if keywords existed, not if instructions were actionable.

**Solution:** Two-level verification: existence + rationale:

```bash
# v1: Just check "blocked" appears near "territory fail"
if grep -A10 "territory.*fail" | grep -qi "blocked"; then
    echo "PASS"
fi

# v2: Check WHEN + WHY
if grep -A5 "territory.*fail" | grep -qi "blocked"; then
    # Found mapping, now verify rationale
    if grep -B10 -A10 "territory.*fail" | grep -qiE "(infrastructure|cannot proceed|critical)"; then
        echo "PASS: Territory failure → blocked with rationale"
    else
        echo "FAIL: Lacks rationale (WHY blocked?)"
        exit 1
    fi
fi
```

### Issue TEST-4: AC-4 Interpretation

**Problem:** AC-4 "Parses results into structured data" is behavioral, but SKILL.md is documentation.

**Solution:** Test that parsing INSTRUCTIONS exist (prescriptive documentation):

```bash
if ! echo "$gather_section" | grep -qE "(parse|structured|extract)"; then
    echo "FAIL: No parsing step documented in GATHER"
    echo "Instruction should tell Claude to 'Parse results into structured data'"
    exit 1
fi
```

This addresses the AC in the context of a documentation artifact.

### Issue TEST-5: Execution Order

**Problem:** v1 test checked presence but not ordering.

**Solution:** Added explicit ordering verification:

```bash
territory_line=$(echo "$gather_section" | grep -n "territory map" | head -1 | cut -d: -f1)
memory_line=$(echo "$gather_section" | grep -n "memory query" | head -1 | cut -d: -f1)
parse_line=$(echo "$gather_section" | grep -n -E "(parse|structured)" | head -1 | cut -d: -f1)

# Validate: territory < memory < parse
if [[ "$territory_line" -gt "$memory_line" ]]; then
    echo "FAIL: Territory map should come BEFORE memory query"
    exit 1
fi
```

## Test Philosophy Applied

Tests follow the principles from TDD Red Producer SKILL.md:

### 1. Tests Must Fail ✓
11 of 13 tests fail, confirming red state.

### 2. Cover All Acceptance Criteria ✓
Each AC has 1-3 dedicated tests.

### 3. Test Behavior, Not Just Structure ✓
For SKILL.md (documentation artifact):
- **Structure tests:** Numbered steps, section ordering, references
- **Semantic tests:** Required concepts present (flexible wording)
- **Completeness tests:** Command + purpose + error handling

### 4. Human-Readable Matchers ✓
Error messages explain WHAT is missing and WHY it matters:
```
FAIL: Territory map purpose not explained
Instruction should say WHAT territory map provides (file structure, artifacts, etc.)
```

### 5. Property Exploration ✓
test_gather_execution_order verifies the property: "territory < memory < parse" must hold.

## Key Decisions

### DEC-1: Test Instruction Completeness
Each instruction must include command + purpose + usage, not just keywords.

### DEC-2: Document-Appropriate Testing
SKILL.md is prescriptive documentation. Test that INSTRUCTIONS exist and are actionable.
For runtime behavior testing, use integration tests (future work).

### DEC-3: Structural + Semantic Testing
Combine:
- **Structural:** Numbered steps, section ordering, references
- **Semantic:** Required concepts present (flexible wording for synonyms)
- **Completeness:** Each instruction complete with command + purpose + error handling

### DEC-4: Execution Order Matters
Territory map informs memory queries (file structure guides semantic search).
Both results are parsed together.
Tests explicitly verify this ordering.

## Traceability

**Tests trace to:** TASK-4 acceptance criteria

Each test function includes comments linking to specific AC:
```bash
# ============================================================================
# AC-2: Executes projctl territory map to get file structure
# Test: Instructions specify HOW to execute territory map AND WHAT to do with output
# ============================================================================
test_territory_map_instruction_completeness() {
    ...
}
```

## Next Phase: Implementation

The green phase (tdd-green-producer) will update SKILL.md to make tests pass by adding:

1. **Territory map instructions:**
   - Command: `projctl territory map`
   - Purpose: Discover artifacts, files, project structure
   - Usage: Use results to inform memory queries

2. **Memory query instructions:**
   - Command: `projctl memory query "<domain-query>"`
   - Examples: "architecture decisions", "technology stack", "system design"
   - Usage: Semantic search for domain knowledge

3. **Result parsing instructions:**
   - Parse territory map output for artifact locations
   - Parse memory query output for relevant context
   - Structure data for ASSESS phase

4. **Error handling instructions:**
   - Territory map failure → yield `blocked` (infrastructure problem, cannot proceed)
   - Memory query timeout → continue with note (degraded but functional)
   - Reference YIELD.md protocol

5. **Context logging instructions:**
   - Log which sources used (territory, memory, files)
   - Include in assessment results

6. **Yield metadata updates:**
   - Update need-user-input yield example
   - Add `[context]` section with `sources = ["territory", "memory", ...]`

7. **Traceability:**
   - Add reference to INTERVIEW-PATTERN.md

## Files Modified

- **Created:** `/Users/joe/.claude/skills/arch-interview-producer/SKILL_test_gather_v2.sh`
- **Target (unchanged):** `/Users/joe/.claude/skills/arch-interview-producer/SKILL.md`

## Verification

```bash
# Run tests to confirm red state
bash ~/.claude/skills/arch-interview-producer/SKILL_test_gather_v2.sh

# Expected output:
TEST: GATHER phase has structured, ordered instructions
PASS: GATHER phase has 5 numbered steps

TEST: GATHER phase documented before need-user-input yield
PASS: GATHER (line 30) precedes SYNTHESIZE (line 44)

TEST: Territory map instructions are complete and actionable
FAIL: Territory map command not documented in GATHER
```

Red state confirmed: tests fail because required instructions don't exist yet.
