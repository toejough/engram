# TDD Red Phase Improvement - TEST-305

**Issue:** ISSUE-61 - Adaptive Interview Depth
**Task:** TASK-6 - Implement adaptive interview logic
**Iteration:** 1 of 3 max
**Date:** 2026-02-05

## Problem

TEST-305 (`TestSelectQuestions_IncludesContextInformation`) had a weak assertion that didn't actually verify AC-5 requirement:

**AC-5:** Question text references gathered context where relevant

**Original assertion (line 265):**
```go
for _, q := range questions {
    // Context field should exist (even if empty)
    // The implementation will populate this based on gathered context
    g.Expect(q).NotTo(BeNil())
}
```

**Issue:** This assertion is vacuous - it only checks that the question struct exists, not that it contains gathered context information.

## Solution

Strengthened the assertion to verify that when gathered context exists:
1. At least one question has its `Context` field populated
2. The `Context` field contains meaningful gathered information (not just any string)

**New assertion (lines 263-278):**
```go
hasContextReference := false
for _, q := range questions {
    // Check if Context field is populated with relevant info
    if q.Context != "" {
        hasContextReference = true
        // Context should actually contain meaningful gathered info, not just exist
        g.Expect(q.Context).To(ContainSubstring("Context"),
            "Context field should reference gathered information, got: %s", q.Context)
    }
}

// At least one question should have context populated when gathered data exists
g.Expect(hasContextReference).To(BeTrue(),
    "When gathered context exists, at least one question should reference it in Context field")
```

## Test Data Enhancement

Enhanced the test's gathered context to be more realistic:
```go
gathered := map[string]string{
    "scale":        "Context shows 10k concurrent users expected",
    "related-tech": "Context mentions SQLite database choice in design.md",
}
```

This provides meaningful context that the implementation should reference in the Context field.

## Verification

Test now fails correctly (red state):

```
Expected
    <int>: 0
to be >=
    <int>: 2
```

The test fails because `SelectQuestions` is currently a stub returning empty slice. Once implemented, it will need to:
1. Return the appropriate number of questions based on gap size
2. Populate the `Context` field with relevant gathered information

## Test Results

**Total tests:** 30
**Passing:** 20 (CalculateGap tests - already implemented)
**Failing:** 10 (SelectQuestions tests - stub implementation)

## Traceability

- **Test ID:** TEST-305
- **Traces to:** TASK-6 AC-5
- **File:** `/Users/joe/repos/personal/projctl/internal/interview/interview_test.go`
- **Lines:** 236-278

## Next Steps

When implementing `SelectQuestions`:
1. Iterate through unanswered questions based on gap size
2. For each question, check if related context exists in `gathered` map
3. If related context found, populate `InterviewQuestion.Context` field with that information
4. Format context as human-readable reference (e.g., "Context shows 10k users expected")
