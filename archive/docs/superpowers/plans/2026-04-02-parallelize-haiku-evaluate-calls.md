# Plan: Parallelize Sequential Haiku Calls in Evaluate (#452)

## Problem

`Evaluator.Run()` in `internal/evaluate/evaluate.go` iterates over pending memories sequentially, making one Haiku API call per memory. Each call is independent — different memory records, different file paths, independent modifier calls. Running them concurrently will reduce latency from `N × call_time` to `~1 × call_time`.

## Analysis

### Current Code (evaluate.go:39-48)

```go
func (e *Evaluator) Run(ctx context.Context, memories []PendingMemory, transcript string) []Result {
    results := make([]Result, 0, len(memories))
    for _, pending := range memories {
        result := e.evaluate(ctx, pending, transcript)
        results = append(results, result)
    }
    return results
}
```

### Independence Verification

Each `evaluate()` call:
- Reads its own `PendingMemory` (unique path, record pointer, eval entry) — no shared mutable state
- Calls `e.caller` (stateless `anthropic.CallerFunc`) — safe for concurrent calls
- Calls `e.modifier` on a unique file path — each goroutine writes to a different memory file
- Writes to a local `Result` — no shared output state

`e.promptTemplate` and `e.model` are read-only struct fields. Fully safe for concurrent access.

### Test Mocks with Data Races

Two existing tests use a shared `callCount` variable that is NOT concurrency-safe:

1. **`TestRun_HaikuErrorOnFirstDoesNotBlockSecond`** (line 39): `callCount++` in caller mock, dispatches error on `callCount == 1`. Under concurrency, both goroutines race on `callCount`, and "first" is meaningless.

2. **`TestRun_MultipleMemories_EachEvaluatedIndependently`** (line 307): Same pattern — `callCount++`, returns FOLLOWED for first call, NOT_FOLLOWED for second.

Both must be fixed to dispatch based on the prompt content (which contains unique memory fields per call) rather than call order.

### Result Ordering

The existing test `TestRun_MultipleMemories_EachEvaluatedIndependently` asserts `results[0]` and `results[1]` by index, meaning the **result order must match input order**. The implementation must preserve this.

**Approach**: Pre-allocate a `[]Result` of length `len(memories)` and have each goroutine write directly to its own index (`results[i]`). No mutex needed — each goroutine writes to a distinct index.

## Design Decision: Why Pre-Allocated Slice, Not Channels

Channels would require collecting results and re-sorting by original index. Pre-allocated slice is simpler:
- Each goroutine owns `results[i]` — no synchronization needed for writes
- `sync.WaitGroup` handles completion signaling
- Order is preserved inherently
- Matches the existing pattern: input slice index → output slice index

## Files to Modify

1. `internal/evaluate/evaluate.go` — parallelize `Run()` with goroutines + `sync.WaitGroup`
2. `internal/evaluate/evaluate_test.go` — fix 2 test mocks for concurrency safety, add barrier concurrency test

## Task 1: Parallelize Haiku evaluate calls (1 task, 1 commit)

### Step 0 — Prerequisite: Fix test mocks for concurrency safety

**File**: `internal/evaluate/evaluate_test.go`

Fix `TestRun_HaikuErrorOnFirstDoesNotBlockSecond` (line 39):
- Remove `callCount := 0` (line 55) and `callCount++` (line 57)
- Replace dispatch condition `if callCount == 1` with prompt-content dispatch
- The caller mock receives `userPrompt` which contains the memory's situation field
- Record 1 has `Situation: "s1"`, record 2 has `Situation: "s2"`
- Dispatch: if `userPrompt` contains `"s1"` → return error; otherwise → return `"FOLLOWED"`
- No callCount assertion exists in this test — only the two lines above need removal

Fix `TestRun_MultipleMemories_EachEvaluatedIndependently` (line 307):
- Remove ALL three callCount references:
  - `callCount := 0` (line 323)
  - `callCount++` (line 327)
  - `g.Expect(callCount).To(Equal(2))` (line 357)
- Replace dispatch condition `if callCount == 1` with prompt-content dispatch
- Record 1 has `Situation: "s1"`, record 2 has `Situation: "s2"`
- Dispatch: if `userPrompt` contains `"s1"` → return `"FOLLOWED"`; if `"s2"` → return `"NOT_FOLLOWED"`
- The callCount assertion is redundant because existing result-based assertions already verify
  both calls completed: results[0].Verdict, results[1].Verdict, record1.FollowedCount,
  record2.NotFollowedCount

These tests should still pass after this step (non-behavioral change to tests).

### Step 1 — RED: Add concurrency barrier test

**File**: `internal/evaluate/evaluate_test.go`

Add `TestRun_EvaluatesMemoriesConcurrently`:
- Create 2 `PendingMemory` entries with distinct records
- Create a `sync.WaitGroup` barrier with `barrier.Add(2)`
- Mock caller: each call does `barrier.Done(); barrier.Wait()` then returns a verdict
  - Under sequential execution, first call blocks at `barrier.Wait()` forever (second goroutine never starts) → context deadline exceeded → test fails
  - Under concurrent execution, both calls arrive at barrier simultaneously → both unblock → test passes
- Use `context.WithTimeout(ctx, 5*time.Second)` — sequential code times out = RED
- Assert both results have correct verdicts

This test MUST FAIL with the current sequential `Run()` implementation.

New imports needed in test file: `"sync"`, `"time"`.

### Step 2 — GREEN: Implement parallel evaluation

**File**: `internal/evaluate/evaluate.go`

Replace `Run()` body:

```go
func (e *Evaluator) Run(ctx context.Context, memories []PendingMemory, transcript string) []Result {
    results := make([]Result, len(memories))

    var waitGroup sync.WaitGroup
    waitGroup.Add(len(memories))

    for i, pending := range memories {
        go func(index int, mem PendingMemory) {
            defer waitGroup.Done()
            results[index] = e.evaluate(ctx, mem, transcript)
        }(i, pending)
    }

    waitGroup.Wait()

    return results
}
```

New import needed: `"sync"`.

Key properties:
- Pre-allocated `results` slice: each goroutine writes to its own index — no mutex needed
- `len(memories)` not `0, len(memories)` — slice is pre-sized, not appended to
- `sync.WaitGroup` signals all goroutines complete before returning
- Order preserved: `results[i]` corresponds to `memories[i]`
- Context is shared (read-only) — both goroutines get the same ctx for cancellation

### Step 3 — REFACTOR: Verify + cleanup

- Run `targ check-full` — all 8 checks must pass
- Verify barrier test now passes (GREEN)
- Verify all existing tests still pass (no regression)

## Verification

- `targ check-full`: PASS:8 FAIL:0
- Barrier test proves actual concurrency (not just "works by accident")
- All existing tests pass unchanged (except the 2 mock fixes, which are non-behavioral)
- Result ordering preserved (existing index-based assertions still pass)

## Risk Assessment

- **Write race on same memory file**: Theoretically possible if two `PendingMemory` entries share the same `Path`. In practice, the scanner produces one `PendingMemory` per `(path, eval)` pair, and modifier uses atomic `ReadModifyWrite`. Consequence if it happens: one lost counter increment, not corruption.
- **Goroutine leak**: `sync.WaitGroup.Wait()` blocks until all complete. No leak possible.
- **Context cancellation**: If ctx is cancelled, `e.caller` returns immediately with error, goroutine completes quickly, `WaitGroup.Done()` called. No hang.
