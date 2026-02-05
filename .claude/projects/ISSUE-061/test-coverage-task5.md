# TASK-5 Test Coverage: Gap Assessment Phase

**Task:** TASK-5 - Implement ASSESS phase that compares gathered context against key questions to determine coverage percentage and gap size.

**Issue:** ISSUE-061

**Artifact:** `skills/arch-interview-producer/SKILL_test_assess.sh`

## Test Summary

| Metric | Value |
|--------|-------|
| Total Tests | 10 |
| Passing | 0 |
| Failing | 10 |
| Status | RED (as expected) |

## Acceptance Criteria Coverage

### AC-1: Code added after context gathering, before interview

**Tests:**
- `test_assess_phase_placement` - Verifies ASSESS Phase section exists between GATHER and INTERVIEW/SYNTHESIZE
- `test_assess_phase_has_structured_instructions` - Verifies ASSESS has ≥3 numbered steps

**Current Status:** FAILING - No ASSESS Phase section exists in SKILL.md

**What tests verify:**
- ASSESS Phase section exists in SKILL.md
- Section is positioned between GATHER and next phase (SYNTHESIZE or INTERVIEW)
- Instructions are structured with numbered steps (not prose)
- Minimum 3 steps to cover: question assessment → calculation → logging

### AC-2: For each key question, determines if answerable from context sources

**Tests:**
- `test_question_assessment_instructions` - Verifies instructions for assessing questions against context

**Current Status:** FAILING - No ASSESS Phase section exists

**What tests verify:**
- Instructions mention checking each/all key questions
- Instructions specify determining answerability from gathered context
- Instructions reference context sources: issue description, territory map, memory results, context files
- At least 3 of 4 context sources are mentioned

### AC-3: Uses coverage calculation function (TASK-3) to compute gap metrics

**Tests:**
- `test_coverage_calculation_usage` - Verifies reference to coverage calculation
- `test_assess_uses_task3_function` - Verifies integration with TASK-3 CalculateGap

**Current Status:** FAILING - No ASSESS Phase section exists

**What tests verify:**
- Instructions mention coverage calculation/CalculateGap function
- Instructions reference weighted formula or priority weights
- Optional: Direct reference to internal/interview/gap.go or CalculateGap function

### AC-4: Logs assessment results: total questions, answered count, coverage %, gap size

**Tests:**
- `test_logging_instructions` - Verifies logging instructions completeness

**Current Status:** FAILING - No ASSESS Phase section exists

**What tests verify:**
- Instructions specify logging/recording assessment results
- At least 3 of 4 key metrics mentioned:
  - Total questions count
  - Answered count
  - Coverage percentage
  - Gap size classification

### AC-5: Handles contradictory context: yields need-decision with conflicts if detected

**Tests:**
- `test_contradictory_context_handling` - Verifies error handling instructions

**Current Status:** FAILING - No ASSESS Phase section exists

**What tests verify:**
- Instructions mention contradictory/conflicting/inconsistent context
- Instructions specify yielding `need-decision` for conflicts
- Instructions mention providing conflict details in the yield

### AC-6: Stores gap assessment in yield metadata for observability

**Tests:**
- `test_yield_metadata_storage` - Verifies metadata storage instructions

**Current Status:** FAILING - No ASSESS Phase section exists

**What tests verify:**
- Instructions specify storing in yield metadata/context
- Instructions mention gap assessment or assessment results
- Instructions provide rationale: observability, debugging, or traceability

### Integration Tests

**Tests:**
- `test_assess_workflow_integration` - Verifies complete workflow documented
- `test_assess_instructions_actionable` - Verifies all instructions have action verbs

**Current Status:** FAILING - No ASSESS Phase section exists

**What tests verify:**
- Complete workflow: assess questions → calculate coverage → classify gap → log results
- All numbered steps use action verbs (Execute, Calculate, Determine, etc.)
- Instructions are actionable, not just descriptive

## Test Design Philosophy

These tests follow the documentation testing approach from TASK-4:

**Three-level validation:**
1. **Existence** - Section/instructions exist
2. **Completeness** - Instructions include necessary components (what, how, when, why)
3. **Actionability** - Instructions use imperative verbs and specify clear actions

**Testing approach:**
- Structural tests (section placement, numbered steps)
- Semantic tests (key concepts mentioned, workflow completeness)
- Integration tests (references TASK-3 function, complete workflow)
- Property tests (all steps actionable)

**What tests do NOT verify:**
- Runtime behavior (this is documentation, not executable code)
- Exact wording (semantic matching with synonym support)
- Implementation details (test intent, not format)

## Red State Verification

**Expected Behavior:** All tests should FAIL because ASSESS Phase does not exist in SKILL.md.

**Actual Behavior:** ✓ Tests fail at first check (no ASSESS Phase section found)

**Verification:** Tests fail for the right reason (feature doesn't exist, not broken tests)

**Next Step:** Implement ASSESS Phase documentation to make tests pass (green phase)

## Test Execution

```bash
# Run tests
bash skills/arch-interview-producer/SKILL_test_assess.sh

# Expected output (red state)
=== TASK-5: Gap Assessment Phase Tests ===
TEST: ASSESS phase positioned between GATHER and INTERVIEW/SYNTHESIZE
FAIL: No ASSESS Phase section found
```

## Traceability

**Traces to:**
- TASK-5 Acceptance Criteria (all 6 criteria)
- ARCH-002 (Gap-Based Depth Calculation)
- ARCH-003 (Yield Context Enrichment)
- REQ-002 (Context gathering before interview)
- REQ-003 (Adaptive interview depth)

**Dependencies:**
- TASK-3: Coverage calculation function (internal/interview/gap.go)
- TASK-4: GATHER phase (context sources)
- TASK-2: Key Questions registry

## Files Modified

- **Created:** `skills/arch-interview-producer/SKILL_test_assess.sh`
- **Expected to modify:** `skills/arch-interview-producer/SKILL.md` (green phase)

## Test Characteristics

| Characteristic | Value |
|----------------|-------|
| Test Type | Documentation structure and semantic tests |
| Language | Bash (grep, sed pattern matching) |
| Test Pattern | Three-level validation (existence → completeness → actionability) |
| Integration | References TASK-3 gap.go, TASK-4 GATHER phase |
| Maintainability | Semantic matching allows wording flexibility |
| Brittleness | Low (uses OR logic, synonym support) |
