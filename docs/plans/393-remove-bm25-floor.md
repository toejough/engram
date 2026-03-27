# Implementation Plan: Remove BM25 Floor Threshold (#393)

Remove the BM25 floor filtering logic and related constants from `internal/surface/surface.go`.
After #391, `minRelevanceScore` is already zeroed (floor disabled), making these constants and the floor-checking logic dead code.

## Context

Issue #391 sets `minRelevanceScore = 0`, which disables the BM25 floor entirely. This plan removes the disabled constants and dead code.

## Changes

### 1. `internal/surface/surface.go` — remove constants and logic

**Delete constants (lines 535–538):**
```go
const (
	// ...
	minRelevanceScore       = 0.05 // DES-P4e-3: raised BM25 floor for tighter filtering
	promptLimit             = 2
	unprovenBM25FloorPrompt = 0.20
)
```

Keep `promptLimit = 2` and other constants. Delete only:
- `minRelevanceScore`
- `unprovenBM25FloorPrompt`

**Simplify `matchPromptMemories` function (lines 678–728):**

Current logic (lines 713–722):
```go
floor := minRelevanceScore
if isUnproven(result.ID, effectiveness) {
    floor = unprovenBM25FloorPrompt
}

if penalizedScore < floor {
    continue
}
```

Replace with simple append (no floor check):
```go
matches = append(matches, promptMatch{mem: mem, bm25Score: penalizedScore})
```

Remove the `floor` variable and the floor-checking if-statement entirely. The penalized score becomes the match score without any threshold.

**Update docstring for `matchPromptMemories` (line 675–677):**

Change from:
```go
// matchPromptMemories returns top 10 memories ranked by BM25 relevance to message.
// Concatenates title, content, principle, keywords, and concepts for scoring.
// Unproven memories (never surfaced) require a higher BM25 floor than proven ones.
```

To:
```go
// matchPromptMemories returns top 10 memories ranked by BM25 relevance to message.
// Concatenates title, content, principle, keywords, and concepts for scoring.
```

### 2. Delete `internal/surface/unproven_bm25_test.go`

The entire file tests BM25 floor behavior:
- `TestProvenMemoryPassesAtLowerBM25Score` — tests that proven memory passes 0.05 floor
- `TestUnprovenPromptMemoryFilteredByHigherBM25Floor` — tests that unproven memory is filtered by 0.20 floor

Both tests are rendered meaningless when the floor is 0. Delete the file.

## TDD Approach

### Red phase

**No new tests needed.** The existing tests in `unproven_bm25_test.go` will fail because:
- They expect memories to be filtered by the floor thresholds
- With floor = 0, all BM25 matches (including weak ones) will surface
- Test assertions checking for substring presence/absence will fail

Let the failing tests guide the fix.

### Green phase

Delete `internal/surface/unproven_bm25_test.go`. No other test changes required — other tests
(cold-start budget, suppression, ranking) do not depend on the floor logic.

### Refactor phase

- Verify `matchPromptMemories` is now simpler (no conditional logic for floor calculation).
- Confirm `targ check-full` passes with zero errors.

## Files to Change

1. `/Users/joe/repos/personal/engram/internal/surface/surface.go` — delete constants, simplify floor-checking logic in `matchPromptMemories`
2. **Delete** `/Users/joe/repos/personal/engram/internal/surface/unproven_bm25_test.go` — all BM25 floor tests

## Verification

After green phase: `targ check-full` must pass with zero errors.
Run `targ test` and confirm all tests pass.

Manually verify: BM25 matches with weak scores (0.01–0.05) now surface in prompt mode
(they previously would have been filtered if unproven).
