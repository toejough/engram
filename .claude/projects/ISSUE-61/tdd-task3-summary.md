# TASK-3 TDD Red Phase Summary

## Overview

Created failing tests for the coverage calculation function that will determine adaptive interview depth.

## Files Created

- `/Users/joe/repos/personal/projctl/internal/interview/gap.go` - Stub implementation (function signature and types defined)
- `/Users/joe/repos/personal/projctl/internal/interview/gap_test.go` - Test suite (19 tests)

## Test Results

**Total Tests:** 19
**Passing:** 3 (property tests that verify basic constraints)
**Failing:** 16 (example-based tests waiting for implementation)

### Passing Tests

1. `TestCalculateGap_Property_CoverageRange` - Coverage is always 0-100
2. `TestCalculateGap_Property_AllAnsweredMeansNoUnanswered` - All answered means no unanswered
3. `TestCalculateGap_Property_GapSizeConsistentWithCoverage` - Gap size matches coverage percent

### Failing Tests (Expected)

All example-based tests fail because stub returns 0% coverage:

**AC-1 Coverage (Function I/O):**
- TestCalculateGap_AllAnswered
- TestCalculateGap_NoneAnswered
- TestCalculateGap_EmptyKeyQuestions
- TestCalculateGap_ExtraAnsweredQuestionsIgnored

**AC-2 Coverage (Return Values):**
- TestCalculateGap_AllAnswered
- TestCalculateGap_NoneAnswered

**AC-3 Coverage (Priority Weights):**
- TestCalculateGap_CriticalUnanswered (-15% each)
- TestCalculateGap_ImportantUnanswered (-10% each)
- TestCalculateGap_OptionalUnanswered (-5% each)
- TestCalculateGap_MixedPriorities_MediumGap
- TestCalculateGap_LargeGap

**AC-4 Coverage (Gap Classification):**
- TestCalculateGap_Boundary_80Percent (≥80% = small)
- TestCalculateGap_Boundary_79Percent (50-79% = medium)
- TestCalculateGap_Boundary_50Percent (50-79% = medium)
- TestCalculateGap_Boundary_49Percent (<50% = large)

**AC-5 Coverage (Edge Case <20%):**
- TestCalculateGap_EdgeCase_VeryLowCoverage
- TestCalculateGap_EdgeCase_Exactly20Percent
- TestCalculateGap_EdgeCase_Below20Percent

## Test Philosophy Applied

- **Blackbox testing**: Package `interview_test` (not `interview`)
- **Human-readable matchers**: Using gomega (e.g., `Expect(x).To(Equal(y))`)
- **Property-based testing**: Using rapid to verify invariants across random inputs
- **Behavior testing**: Tests verify the calculation logic, not internal structure

## Coverage Mapping

Each acceptance criterion has comprehensive test coverage:

| AC | Description | Test Count |
|----|-------------|------------|
| AC-1 | Function I/O | 4 tests |
| AC-2 | Return values | 2 tests |
| AC-3 | Priority weights | 5 tests |
| AC-4 | Gap classification | 9 tests |
| AC-5 | Edge case <20% | 3 tests |
| AC-6 | Property tests | 3 tests |

## Next Steps

The TDD green phase (TASK-4) will implement the actual coverage calculation logic to make these tests pass.

## Traceability

- Tests trace to: TASK-3
- TASK-3 traces to: ARCH-002 (Gap-Based Depth Calculation), ARCH-005 (Key Questions Registry)
