# TASK-6 TDD Red Phase Summary

**Task:** TASK-6 - Implement adaptive interview logic
**Phase:** TDD Red (Test-First)
**Status:** Complete
**Date:** 2026-02-05

## Overview

Created failing tests for the adaptive interview logic that determines which questions to ask users based on gap assessment results. The `SelectQuestions` function will adjust question count and priority based on gap size (small/medium/large).

## Files Created

1. **internal/interview/interview.go** - Stub implementation with `SelectQuestions` function and `InterviewQuestion` type
2. **internal/interview/interview_test.go** - Comprehensive test suite covering all acceptance criteria

## Test Results

- **Total tests:** 10
- **Passing:** 1 (edge case: all questions answered)
- **Failing:** 9 (expected - red state verified)
- **Red state:** ✓ Confirmed

## Test Coverage by Acceptance Criteria

### AC-1: Code added after gap assessment, before yield
- TestSelectQuestions_SmallGap_OneCriticalUnanswered
- TestSelectQuestions_AllAnswered_EmptyResult

### AC-2: Small gap (≥80%) yields 1-2 confirmation questions for critical unanswered only
- TestSelectQuestions_SmallGap_OneCriticalUnanswered
- TestSelectQuestions_SmallGap_NoCriticalUnanswered
- TestSelectQuestions_MediumGap_PriorityOrdering

### AC-3: Medium gap (50-79%) yields 3-5 clarification questions prioritizing critical then important
- TestSelectQuestions_MediumGap
- TestSelectQuestions_MediumGap_PriorityOrdering

### AC-4: Large gap (<50%) yields full interview sequence (6+ questions)
- TestSelectQuestions_LargeGap
- TestSelectQuestions_LargeGap_AllPriorities

### AC-5: Question text references gathered context where relevant
- TestSelectQuestions_ReferencesGatheredContext
- TestSelectQuestions_IncludesContextInformation

### AC-6: Questions skip topics fully answered by context
- TestSelectQuestions_SkipsAnsweredTopics
- TestSelectQuestions_AllAnswered_EmptyResult

## API Design

### Types

```go
type InterviewQuestion struct {
    ID       string   // Question identifier from KeyQuestion
    Text     string   // Question text
    Priority Priority // Priority level
    Context  string   // Optional context from gathered information
}
```

### Function Signature

```go
func SelectQuestions(
    keyQuestions []KeyQuestion,
    gapAnalysis GapAnalysis,
    gathered map[string]string,
) []InterviewQuestion
```

**Parameters:**
- `keyQuestions`: All key questions that could be asked
- `gapAnalysis`: Results from CalculateGap showing coverage and unanswered questions
- `gathered`: Map of question IDs to context strings for answered questions

**Returns:**
- Ordered list of questions to ask the user

## Test Strategy

### Example-Based Tests
Cover specific scenarios:
- Small gap with one critical unanswered
- Small gap with no critical unanswered
- Medium gap with mixed priorities
- Large gap requiring full interview
- Edge cases (all answered, priority ordering)

### Behavioral Verification
Tests verify:
- Question count matches gap size tier
- Priority ordering (critical → important → optional)
- Answered questions are skipped
- Context is available for questions
- Unanswered questions are included

## Design Decisions

### Decision 1: InterviewQuestion Type
**Chosen:** Create new `InterviewQuestion` type with context field
**Rationale:** Questions need to carry gathered context to ask informed questions
**Alternatives:** Reuse KeyQuestion, use map[string]string

### Decision 2: Gathered Context Parameter
**Chosen:** Accept `map[string]string` mapping question ID to context
**Rationale:** Simple, flexible, allows easy lookup of available context
**Alternatives:** Structured context type, pass GapAnalysis only

### Decision 3: Test Coverage Approach
**Chosen:** Multiple examples per tier with specific scenarios
**Rationale:** Each gap size has different requirements, need concrete verification
**Alternatives:** Property-based tests only, single example per tier

### Decision 4: Stub Implementation
**Chosen:** Return empty list to force test failures
**Rationale:** Violates all non-empty expectations, ensures proper red state
**Alternatives:** Return random questions, return all questions

## Failure Verification

Sample failure output showing tests fail for correct reasons:

```
TestSelectQuestions_SmallGap_OneCriticalUnanswered:
  Expected <int>: 0 to be >= <int>: 1
  (Stub returns no questions when it should return 1-2)

TestSelectQuestions_SkipsAnsweredTopics:
  Should ask tech-stack (unanswered)
  Expected <bool>: false to be true
  (Stub doesn't include unanswered questions)
```

One test correctly passes: `TestSelectQuestions_AllAnswered_EmptyResult` verifies that when all questions are answered (100% coverage), the function should return an empty list.

## Traceability

**Traces to:**
- ARCH-002: Gap-Based Depth Calculation
- ARCH-004: Consistent Interview Protocol
- REQ-003: Adaptive Interview Depth

**Task:** TASK-6
**Upstream artifacts:** internal/interview/gap.go (TASK-3), SKILL.md key questions (TASK-2)

## Next Steps

1. **TDD Green Phase:** Implement `SelectQuestions` to make tests pass
2. **TDD Refactor Phase:** Optimize implementation if needed
3. **Integration:** Wire into arch-interview-producer SKILL.md workflow
4. **TASK-7:** Add yield context enrichment with gap analysis metadata

## Notes

- Tests use blackbox pattern (`package interview_test`)
- Tests use gomega assertions for readability
- No property-based tests in this phase (concrete scenarios are sufficient)
- Implementation should handle edge cases: empty unanswered list, all questions critical, etc.
- Context field population logic belongs in implementation (tests verify structure exists)
