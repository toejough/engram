# Plan: Parallelize consolidation + adapt in maintain (#451)

## Problem

`runSonnetAnalyses()` in `internal/maintain/maintain.go:74-100` runs `FindMerges()` and `Analyze()` sequentially. Both are independent Sonnet LLM calls that can run concurrently, saving ~1 LLM call of latency.

## Analysis

### Independence verification

- **FindMerges** reads `records` (read-only), calls `c.caller` with consolidation prompt, returns `[]Proposal`
- **Analyze** reads `records` (read-only), `cfg.Policy`, `cfg.ChangeHistory` (all read-only), calls `a.caller` with adapt prompt, returns `[]Proposal`
- No shared mutable state between them
- Both use the same `cfg.Caller` function, but `anthropic.CallerFunc` is stateless — concurrent calls are safe
- Each writes to its own local variables — no write contention

### Test impact

`TestRun_WithCaller_IncludesConsolidationAndAdapt` (maintain_test.go:344) uses a **counter-based mock**:

```go
callCount++
if callCount == 1 { return consolidation response }
return adapt response
```

This has two problems under concurrency:
1. **Data race**: `callCount++` is not atomic
2. **Order assumption**: first call is no longer guaranteed to be consolidation

**Fix**: Switch to prompt-based dispatch (matching the pattern already used in `TestRun_RewritesUpdateProposalValues` at line 310-316). Check the `systemPrompt` parameter to determine which response to return — consolidation uses `cfg.Policy.MaintainConsolidatePrompt`, adapt uses `cfg.Policy.AdaptSonnetPrompt`.

## Tasks

### Task 1: Parallelize runSonnetAnalyses and fix test mock

**Files to modify:**
- `internal/maintain/maintain.go` (lines 74-100)
- `internal/maintain/maintain_test.go` (lines 374-387)

**Step 1 (prerequisite — make mock concurrency-safe):** Update the mock in `TestRun_WithCaller_IncludesConsolidationAndAdapt` to use prompt-based dispatch instead of counter-based ordering. Remove `callCount` variable. Use `systemPrompt` parameter to distinguish consolidation vs adapt calls (same pattern as `TestRun_RewritesUpdateProposalValues`). Run `targ test` — should still pass (this is a prerequisite, not a TDD phase).

**Step 2 (RED):** Add `TestRunSonnetAnalyses_RunsConcurrently` that proves parallelism:

```go
func TestRunSonnetAnalyses_RunsConcurrently(t *testing.T) {
    t.Parallel()
    g := NewGomegaWithT(t)

    // ... setup dataDir with 2 memories ...

    defaults := policy.Defaults()
    var barrier sync.WaitGroup
    barrier.Add(2)

    mockCaller := anthropic.CallerFunc(
        func(_ context.Context, _, systemPrompt, _ string) (string, error) {
            barrier.Done()  // signal arrival
            barrier.Wait()  // block until both goroutines have arrived

            if systemPrompt == defaults.MaintainConsolidatePrompt {
                return `[{"survivor":"mem-a.toml","members":["mem-a.toml","mem-b.toml"],"rationale":"similar"}]`, nil
            }
            return `[{"field":"maintain_min_surfaced","value":"10","rationale":"increase"}]`, nil
        },
    )

    cfg := maintain.Config{
        Policy:        defaults,
        DataDir:       dataDir,
        Caller:        mockCaller,
        ChangeHistory: []policy.ChangeEntry{},
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    proposals, err := maintain.Run(ctx, cfg)
    g.Expect(err).NotTo(HaveOccurred())
    // ... assert both merge and update proposals present ...
}
```

**How this works:** The mock blocks each caller until both have arrived at the barrier. With sequential execution, the first call blocks forever waiting for the second (timeout after 5s → context deadline exceeded → test fails). With concurrent execution, both arrive at the barrier, unblock each other, and return their responses.

Run `targ test` — test times out with context deadline exceeded error. This is the RED state.

**Step 3 (GREEN):** Modify `runSonnetAnalyses` to run FindMerges and Analyze concurrently:

```go
func runSonnetAnalyses(
    ctx context.Context, cfg Config, records []memory.StoredRecord,
) ([]Proposal, error) {
    type result struct {
        proposals []Proposal
        err       error
    }

    mergeCh := make(chan result, 1)
    adaptCh := make(chan result, 1)

    consolidator := NewConsolidator(cfg.Caller, cfg.Policy.MaintainConsolidatePrompt)
    go func() {
        proposals, err := consolidator.FindMerges(ctx, records)
        mergeCh <- result{proposals, err}
    }()

    adapter := NewAdapter(cfg.Caller, cfg.Policy.AdaptSonnetPrompt)
    go func() {
        proposals, err := adapter.Analyze(ctx, records, cfg.Policy, cfg.ChangeHistory)
        adaptCh <- result{proposals, err}
    }()

    mergeResult := <-mergeCh
    adaptResult := <-adaptCh

    var proposals []Proposal
    var errs []error

    if mergeResult.err != nil {
        errs = append(errs, fmt.Errorf("finding merges: %w", mergeResult.err))
    } else {
        proposals = append(proposals, mergeResult.proposals...)
    }

    if adaptResult.err != nil {
        errs = append(errs, fmt.Errorf("running adapt analysis: %w", adaptResult.err))
    } else {
        proposals = append(proposals, adaptResult.proposals...)
    }

    return proposals, errors.Join(errs...)
}
```

No new imports needed in maintain.go — channels and goroutines are builtins/keywords.

Run `targ test` — concurrency test now passes (barrier unblocks because both goroutines arrive). All existing tests still pass.

**Step 4 (REFACTOR):** Review for any cleanup. Run `targ check-full`. Fix any issues. Commit.

**Why channels over sync.WaitGroup or errgroup:**
- Two goroutines, each returns (proposals, error) — channels are the natural fit
- No need for `sync` import or external dependency in production code
- errgroup cancels on first error by default — we want both to always complete
- Buffered channels (size 1) mean goroutines never block on send

**Note on the barrier test and rewriter:** `maintain.Run()` calls `rewriter.RewriteProposals()` before `runSonnetAnalyses()`. The rewriter also uses `cfg.Caller`. If the test memories trigger a rewrite proposal, the rewriter call would hit the barrier before either sonnet analysis goroutine, causing a hang. The test memories use `workingMemory` which is healthy — `DiagnoseAll` produces 0 proposals, so the rewriter has nothing to rewrite and does NOT call the LLM. The barrier is hit exactly twice (consolidation + adapt).

### Verification

- `targ check-full` must pass (PASS:8 FAIL:0)
- All existing tests continue to pass with same assertions
- Race detection: targ does not currently have a race detection target. The barrier-based concurrency test provides structural proof of parallelism. Race safety is guaranteed by design (no shared mutable state between goroutines). Flag to coordinator: consider adding a `targ race` target for future concurrency work.
