# Consolidation Signal Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace silent keyword-overlap auto-merge with consolidation proposals the user can approve/reject.

**Architecture:** Delete the mechanical merge path from `consolidate.go`, keep cluster detection (`Plan`/`buildClusters`). In `RunMaintain`, call `Plan()` instead of `Consolidate()` and convert clusters to proposals. Add consolidation to the hook triage rendering.

**Tech Stack:** Go, `targ` build system, bash (hooks)

**Spec:** `docs/superpowers/specs/2026-03-25-consolidation-signal-design.md`

---

### Task 1: Delete mechanical merge path from consolidate.go

Remove the auto-merge functions, dead struct fields, dead `With*` options, and the `ConsolidateResult` type. Keep `Plan()`, `buildClusters()`, `clusterConfidence()`, `selectSurvivor()` (used by `Plan`), and shared `With*` options.

**Files:**
- Modify: `internal/signal/consolidate.go`
- Delete or heavily modify: `internal/signal/consolidate_test.go`
- Delete or heavily modify: `internal/signal/consolidate_migrate_test.go`

- [ ] **Step 1: Identify dead functions**

Read `consolidate.go`. Delete these functions (merge-path only):
- `Consolidate()` (~line 57)
- `mergeCluster()` (~line 187)
- `processAbsorbed()` (~line 244)
- `recomputeLinks()` (~line 285)
- `collectPrinciples()` (helper for mergeCluster)
- `countNewKeywords()` (helper for mergeCluster)
- `logStderrf()` (~line 180) — verify no other caller
- `applySynthesizedPrinciple()` (~line 145) — verify no other caller

- [ ] **Step 2: Delete `ConsolidateResult` struct**

Delete the struct (lines 19-24):
```go
type ConsolidateResult struct {
    ClustersFound  int
    MemoriesMerged int
    Errors         []error
}
```

- [ ] **Step 3: Remove dead struct fields from `Consolidator`**

Delete these fields (merge-path only, not used by `Plan()` or semantic path):
- `merger`
- `synthesizer`
- `fileWriter`
- `backupWriter` + `backupDir`
- `fileDeleter`
- `entryRemover`
- `stderr`

Keep (used by `Plan()`):
- `lister`
- `effectiveness` (selectSurvivor)
- `similarity` (clusterConfidence)

Keep (used by semantic path in `consolidate_semantic.go`):
- `scorer` (FindSimilar)
- `confirmer` (ConfirmClusters)
- `extractor` (ExtractPrinciple)
- `archiver` (archive originals)
- `linkRecomputer` (RecomputeAfterMerge)

- [ ] **Step 4: Delete dead `With*` options**

Delete (merge-path only, not used by Plan or semantic path):
- `WithBackupWriter` (merge-only)
- `WithEntryRemover` (merge-only)
- `WithFileDeleter` (merge-only)
- `WithFileWriter` (merge-only)
- `WithMerger` (merge-only)
- `WithPrincipleSynthesizer` (merge-only)
- `WithStderr` (merge-only)

Keep (used by Plan):
- `WithEffectiveness`
- `WithLister`
- `WithTextSimilarityScorer`

Keep (used by semantic path):
- `WithArchiver`
- `WithConfirmer`
- `WithExtractor`
- `WithLinkRecomputer`
- `WithScorer`

- [ ] **Step 5: Delete dead tests**

Delete all test functions in `consolidate_test.go` that test `Consolidate()` or merge internals (~34 tests). If all tests in the file are dead, delete the entire file.

Delete all test functions in `consolidate_migrate_test.go` (~14 tests). This file tests `ConsolidateBatch` which wraps `Consolidate()`.

Keep any tests that test `Plan()` or `buildClusters()` directly (check first — there may be none).

- [ ] **Step 6: Run `targ check-full`**

Run: `targ check-full`
Expected: Pass (pre-existing hooks failures OK). If compilation errors appear, a caller of a deleted function was missed — check error messages and fix.

- [ ] **Step 7: Commit**

```bash
git add -A internal/signal/
git commit -m "refactor: delete mechanical keyword-overlap auto-merge (#373)

Keep Plan() for cluster detection. Delete Consolidate(),
mergeCluster, dead With* options, ~48 dead tests.

AI-Used: [claude]"
```

### Task 2: Wire consolidation proposals in RunMaintain

Replace the `consolidator.Consolidate(ctx)` call with `Plan(ctx)` and convert clusters to proposals.

**Files:**
- Modify: `internal/cli/cli.go` (~lines 269-289)
- Modify: `internal/maintain/maintain.go` (add action constant)

- [ ] **Step 1: Add `"consolidate"` action constant**

In `internal/maintain/maintain.go`, add alongside existing action constants:
```go
const ActionConsolidate = "consolidate"
```

Check existing pattern for action constants — follow it.

- [ ] **Step 2: Update consolidator construction in `RunMaintain`**

In `cli.go` `RunMaintain` (~line 269-285), the consolidator is currently constructed with many merge-path options. Trim to only the options `Plan()` needs:

```go
consolidator := signal.NewConsolidator(
    signal.WithLister(lister),
    signal.WithEffectiveness(effReader),
    signal.WithTextSimilarityScorer(tfidfScorer),
)
```

Remove all deleted `With*` calls (`WithMerger`, `WithFileWriter`, `WithFileDeleter`, `WithBackupWriter`, `WithEntryRemover`, `WithStderr`, `WithPrincipleSynthesizer`, `WithLinkRecomputer`).

- [ ] **Step 3: Replace `Consolidate()` with `Plan()` → proposals**

Replace the `consolidator.Consolidate(ctx)` call (~line 286) with:

```go
plans, planErr := consolidator.Plan(ctx)
if planErr != nil {
    // log but don't fail — same pattern as existing consolidate error handling
    fmt.Fprintf(os.Stderr, "[engram] consolidation plan: %v\n", planErr)
}

for _, plan := range plans {
    members := make([]map[string]string, 0, len(plan.Absorbed)+1)
    // Add survivor and absorbed as members
    // ... build member list with path and title from memoryMap ...

    details, _ := json.Marshal(map[string]any{
        "members":         members,
        "shared_keywords": /* extract from memoryMap */,
        "confidence":      plan.Confidence,
    })

    proposals = append(proposals, maintain.Proposal{
        MemoryPath: plan.Survivor,
        Quadrant:   "",
        Diagnosis:  fmt.Sprintf("%d memories share keywords (confidence: %.2f). Consider consolidating.", len(plan.Absorbed)+1, plan.Confidence),
        Action:     maintain.ActionConsolidate,
        Details:    details,
    })
}
```

Adapt to match existing patterns in `RunMaintain`. The `memoryMap` (built earlier in the function) maps file paths to `*memory.Stored`, so member titles and shared keywords can be extracted.

- [ ] **Step 4: Write a test**

Add a test in `internal/cli/cli_test.go` or `internal/maintain/maintain_test.go` that verifies consolidation proposals appear when `Plan()` returns clusters. Use a mock lister that returns memories with overlapping keywords.

- [ ] **Step 5: Run `targ check-full`**

Run: `targ check-full`
Expected: Pass.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/cli.go internal/maintain/maintain.go
git commit -m "feat: emit consolidation proposals from maintain (#373)

Replace Consolidate() with Plan() → proposals. Users see
consolidation candidates in triage output.

AI-Used: [claude]"
```

### Task 3: Add consolidation to hook triage rendering

Add consolidation count and details to `session-start.sh` triage output.

**Files:**
- Modify: `hooks/session-start.sh` (~lines 78-150)

- [ ] **Step 1: Add consolidation count**

After the existing count variables (~line 89), add:
```bash
CONSOLIDATE_COUNT=$(echo "$SIGNAL_OUTPUT" | jq '[.[] | select(.action == "consolidate")] | length' 2>/dev/null) || CONSOLIDATE_COUNT=0
```

- [ ] **Step 2: Add consolidation details section**

Follow the existing pattern (e.g., Noise section ~lines 92-100). Add:
```bash
CONSOLIDATE_DETAILS=""
if [[ "$CONSOLIDATE_COUNT" -gt 0 ]]; then
    CONSOLIDATE_DETAILS=$(echo "$SIGNAL_OUTPUT" | jq -r '
        [.[] | select(.action == "consolidate")] |
        to_entries |
        map("  \(.key + 1). \(.value.memory_path | split("/") | last | rtrimstr(".toml")) — \(.value.diagnosis)") |
        .[] // empty
    ' 2>/dev/null) || CONSOLIDATE_DETAILS=""
    if [[ -n "$CONSOLIDATE_DETAILS" ]]; then
        CONSOLIDATE_DETAILS="## Consolidation Candidates ($CONSOLIDATE_COUNT groups)\n$CONSOLIDATE_DETAILS"
    fi
fi
```

- [ ] **Step 3: Include in summary line**

Update the summary line construction to include consolidation count alongside existing counts (noise, hidden gems, leech, etc.).

- [ ] **Step 4: Append to triage details**

Add consolidation details to `TRIAGE_DETAILS` alongside existing sections (~line 142-149):
```bash
if [[ -n "$CONSOLIDATE_DETAILS" ]]; then
    TRIAGE_DETAILS="${TRIAGE_DETAILS:+$TRIAGE_DETAILS\n\n}$CONSOLIDATE_DETAILS"
fi
```

- [ ] **Step 5: Run `targ check-full`**

Run: `targ check-full`
Expected: Pass. Note: the pre-existing hooks test failures may change since we modified hook content — check if `TestT370_*` assertions need updating.

- [ ] **Step 6: Commit**

```bash
git add hooks/session-start.sh
git commit -m "feat: render consolidation proposals in triage output (#373)

AI-Used: [claude]"
```

### Task 4: Update hooks tests (if needed)

If Task 3 caused hooks test failures because the test assertions check exact hook script content, update the test expectations.

**Files:**
- Modify: `internal/hooks/hooks_test.go` (if assertions broke)

- [ ] **Step 1: Check test results**

Run: `targ test -- -run TestT370 ./internal/hooks/`

If tests pass (the assertions may not check the triage section), skip this task.

If tests fail, update the expected content to include the new consolidation rendering.

- [ ] **Step 2: Commit (if changes needed)**

```bash
git add internal/hooks/hooks_test.go
git commit -m "test: update hooks test expectations for consolidation triage (#373)

AI-Used: [claude]"
```
