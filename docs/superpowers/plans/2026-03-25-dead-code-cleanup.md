# Dead Code Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all dead session-start and precompact surface mode code (#375).

**Architecture:** Bottom-up deletion: tests first, then production code, then budget cleanup, then CLI wiring. Each commit verified with `targ check-full`.

**Tech Stack:** Go, `targ` build system

**Spec:** `docs/superpowers/specs/2026-03-25-dead-code-cleanup-design.md`

---

### Task 1: Delete dead test functions

Remove test functions that exercise only dead session-start/precompact paths. Leave live tests untouched. Delete entire files only when ALL functions are dead.

**Files:**
- Delete: `internal/surface/contradict_test.go` (entire file — all tests exercise dead `suppressContradictions`)
- Modify: `internal/surface/surface_test.go` (remove 21 dead test functions)
- Modify: `internal/surface/p3_test.go` (remove ~10 dead test functions, keep ~3 live)
- Modify: `internal/surface/p4e_test.go` (remove 5 dead test functions, keep 3 live)
- Modify: `internal/surface/p4f_test.go` (remove 3 dead test functions, keep 7 live)
- Modify: `internal/surface/cold_start_budget_test.go` (remove 1 dead test function, keep 3 live)
- Modify: `internal/surface/budget_test.go` (remove dead budget field assertions from 2 tests)
- Modify: `internal/surface/export_test.go` (remove `ExportSessionStartDefaultEffectiveness` after renaming the constant)

- [ ] **Step 1: Delete `contradict_test.go`**

Delete the entire file — all tests exercise `suppressContradictions` which is dead.

- [ ] **Step 2: Remove dead functions from `surface_test.go`**

Delete these 21 test functions (all use `ModeSessionStart` or `ModePreCompact`):

```
TestT115_SurfacingOutputIncludesPrinciple
TestT116_OutputUsesSlugPrincipleFormat
TestT163_ToolModeFrecencyReRanking
TestT199_RegistryRecordSurfacingCalled
TestT199b_RegistryErrorDoesNotAffectSurfacing
TestT199c_NilRegistryNoPanic
TestT27_SessionStartSurfacesTop7
TestT28_SessionStartSurfacesAll
TestT29_SessionStartNoMemories
TestT323_PreCompactRanksByEffectiveness
TestT324_PreCompactSkipsLowEffectiveness
TestT325_PreCompactLimitsToTop5
TestT326_PreCompactOutputFormat
TestT346_SurfacerSwallowsSurfacingLogError
TestT69_SessionStartJSONFormat
TestT72_NoMatchJSONFormat
TestT79_TrackerReceivesMatchedMemories
TestT80_TrackerErrorDoesNotAffectOutput
TestT92_SessionStartIncludesCreationReport
TestT93_SessionStartNoCreationLogReturnsRecencyOnly
TestT94_SessionStartCreationLogNoMemoriesProducesCreationOnly
```

- [ ] **Step 3: Remove dead functions from `p3_test.go`**

Delete these test functions (session-start-only — test dead `appendClusterNotes`, `updateCoSurfacingLinks`, dead `With*` options):

```
TestAppendClusterNotes_CalledDuringSessionStart
TestClusterNotes_AbsentWithoutTitleFetcher
TestClusterNotes_Format
TestClusterNotes_SkipsNoTitle
TestClusterNotes_Top2ByWeight
TestCoSurfacingUpdate_ErrorDoesNotAbort
TestUpdateCoSurfacingLinks_CalledDuringSessionStart
TestUpdateCoSurfacingLinks_IncrementsExistingLink
TestWithLinkUpdater_SetsOption
TestWithTitleFetcher_SetsOption
```

Keep these (live — test prompt-mode spreading activation and shared `WithLinkReader`):
```
TestPromptActivationSorting
TestSpreadingActivation_LinkedTargetNotInCandidateSet
TestSpreadingActivation_NoLinkReader
TestSpreadingActivation_ReranksMemories
TestWithLinkReader_SetsOption
```

Verify: check the mode used in each `Spreading*` test. If any use `ModeSessionStart`, they're dead. If they use `ModePrompt` or test the function directly, they're live.

- [ ] **Step 4: Remove dead functions from `p4e_test.go`**

Delete these (session-start-only):
```
TestTP4e1_SessionStartLimitsToTop7
TestTP4e2_SessionStartEffectivenessGating
TestTP4e3_SessionStartFewEvaluationsUsesRecordedScore
TestTP4e4_SessionStartRanksByEffectiveness
TestTP4e5_DefaultBudgetsUpdated
```

Keep these (live — PreToolUse path):
```
TestTP4e6_PreToolUseLimitsToTop2
TestTP4e7_PreToolUseEffectivenessGating
TestTP4e8_InvocationTokenLoggerCalled
```

Verify: check `TestTP4e5_DefaultBudgetsUpdated` — if it only checks dead budget constants, delete it. If it checks live ones too, update it.

- [ ] **Step 5: Remove dead functions from `p4f_test.go`**

Delete these (test dead `suppressClusterDuplicates`):
```
TestTP4f1_ClusterDedupKeepsHigherEffectiveness
TestTP4f2_ClusterDedupDoesNotSuppressUnlinkedMemories
TestTP4f10_ClusterDedupEventFields
```

Keep these (test live suppression infrastructure used by prompt/tool):
```
TestTP4f3_CrossRefCheckerSuppressesCoveredMemory
TestTP4f4_TranscriptSuppressionSkipsMatchingKeywords
TestTP4f5_TranscriptSuppressionIsCaseInsensitive
TestTP4f6_SuppressionLoggerReceivesEvents
TestTP4f7_SuppressionStatsInResult
TestTP4f8_EmptyTranscriptWindowNoSuppression
TestTP4f9_TranscriptSuppressionInPromptMode
```

- [ ] **Step 6: Remove dead function from `cold_start_budget_test.go`**

Delete:
```
TestColdStartBudgetLimitsUnprovenSessionStart
```

Keep (live — prompt and tool modes):
```
TestColdStartBudgetDoesNotLimitProvenMemories
TestColdStartBudgetLimitsUnprovenPromptMemories
TestColdStartBudgetLimitsUnprovenToolMemories
```

- [ ] **Step 7: Clean up `budget_test.go`**

In `TestT193_BudgetConfigCustomValues` (line 231): remove assertions for `ModePreCompact` and any dead budget fields (`PostToolUse`, `Stop`, `PreCompact` if dead).

In `TestT193_BudgetConfigDefaults` (line 252): remove assertions for dead constants (`DefaultPreCompactBudget`, etc.). If `DefaultBudgetConfig()` is only testing dead fields, delete the entire test.

Verify by reading both tests — only remove assertions for fields that will be deleted in Task 3.

- [ ] **Step 8: Run `targ check-full`**

Run: `targ check-full`
Expected: All tests pass, no lint errors. Some dead production code will show as unused — that's expected, we delete it in Task 2.

Note: If the linter flags unused production code at this stage, the build may fail. In that case, combine Task 1 and Task 2 into a single commit.

- [ ] **Step 9: Commit**

```bash
git add -A internal/surface/
git commit -m "test: remove dead session-start and precompact test functions (#375)

AI-Used: [claude]"
```

### Task 2: Delete dead production code from surface package

Remove all dead functions, types, interfaces, constants, struct fields, and `With*` options.

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/suppress_p4f.go`
- Modify: `internal/surface/export_test.go`
- Modify: `internal/surface/helpers_test.go`

- [ ] **Step 1: Remove dead mode constants and `Run()` switch cases**

In `surface.go`, delete:
- `ModeSessionStart` constant (line 31)
- `ModePreCompact` constant (line 29)
- The `case ModeSessionStart:` block in `Run()` (lines ~207-208)
- The `case ModePreCompact:` block in `Run()` (lines ~215-225)

- [ ] **Step 2: Remove dead interfaces and types**

Delete from `surface.go`:
- `ContradictionDetector` interface (~line 41)
- `CreationLogReader` interface (~line 46)
- `LogEntry` type alias (~line 97)
- `LinkUpdater` interface (~line 91)
- `SignalEmitter` interface (search for it)
- `TitleFetcher` interface (search for it)

- [ ] **Step 3: Remove dead `With*` options and struct fields**

Delete from `surface.go`:
- `WithLogReader` function and `logReader` struct field
- `WithLinkUpdater` function and `linkUpdater` struct field
- `WithTitleFetcher` function and `titleFetcher` struct field
- `WithClusterDedupReader` function and `clusterDedupReader` struct field
- `WithContradictionDetector` function and `contradictionDetector` struct field
- `WithSignalEmitter` function and `signalEmitter` struct field

- [ ] **Step 4: Remove dead functions**

Delete from `surface.go`:
- `runSessionStart` (~line 650)
- `runPreCompact` (~line 379)
- `filterByEffectivenessGate` (~line 1350)
- `sortByEffectivenessScore` (~line 1559)
- `applySpreadingActivation` (~line 1199)
- `sortByActivatedScore` (~line 1551)
- `applyColdStartBudgetStored` (~line 1141)
- `writeCreationSection` (~line 1605)
- `writeRecencySection` (~line 1631)
- `updateCoSurfacingLinks` (~line 908)
- `appendClusterNotes` (~line 276)
- `suppressContradictions` (~line 850)

Delete from `suppress_p4f.go`:
- `suppressClusterDuplicates` (~line 135)

- [ ] **Step 5: Remove dead constants**

Delete from `surface.go`:
- `sessionStartLimit`
- `spreadingActivationDecay`
- `minPreCompactEffectiveness`
- `preCompactLimit`

- [ ] **Step 6: Rename `sessionStartDefaultEffectiveness` → `defaultEffectiveness`**

This constant is shared (used by live `effectivenessScoreFor`) but has a misleading name. Rename in `surface.go` using replace-all. Also update:
- `export_test.go`: rename `ExportSessionStartDefaultEffectiveness` → `ExportDefaultEffectiveness`
- `helpers_test.go`: update any references to the old name in test descriptions/comments

- [ ] **Step 7: Run `targ check-full`**

Run: `targ check-full`
Expected: All tests pass, no lint errors, no unused code warnings.

- [ ] **Step 8: Commit**

```bash
git add -A internal/surface/
git commit -m "refactor: remove dead session-start and precompact surface code (#375)

Remove 13 dead functions, 6 dead types/interfaces, 6 dead With*
options, 6 dead struct fields, and 4 dead constants. Rename
sessionStartDefaultEffectiveness to defaultEffectiveness.

AI-Used: [claude]"
```

### Task 3: Clean up budget.go

Remove dead budget fields, constants, and `ForMode` cases.

**Files:**
- Modify: `internal/surface/budget.go`
- Modify: `internal/surface/budget_test.go` (if not fully cleaned in Task 1)

- [ ] **Step 1: Read `budget.go` and identify all dead items**

Read the file. Identify:
- `DefaultSessionStartBudget` and `DefaultPreCompactBudget` constants
- `BudgetConfig.SessionStart` and `BudgetConfig.PreCompact` struct fields
- `ForMode` cases for the deleted mode constants
- `DefaultBudgetConfig()` — if it only sets dead fields, delete it entirely
- Any other dead budget fields (check `PostToolUse`, `Stop` — verify if they're used by live code)

- [ ] **Step 2: Remove dead items**

Delete identified dead constants, struct fields, and `ForMode` cases. If `DefaultBudgetConfig()` becomes empty or only serves dead tests, delete it.

Keep `ForMode()` method itself (used by live modes).

- [ ] **Step 3: Run `targ check-full`**

Run: `targ check-full`
Expected: Pass.

- [ ] **Step 4: Commit**

```bash
git add internal/surface/budget.go internal/surface/budget_test.go
git commit -m "refactor: remove dead budget fields for session-start and precompact (#375)

AI-Used: [claude]"
```

### Task 4: Clean up CLI wiring

Remove the dead `WithLogReader` call and associated construction in the CLI.

**Files:**
- Modify: `internal/cli/cli.go`

- [ ] **Step 1: Read `runSurface` in `cli.go`**

Read the function around line 1457–1540. Find:
- The `surface.WithLogReader(logReader)` call (line ~1516)
- The `logReader` variable construction above it
- Any other dead mode references (session-start/precompact handling)

- [ ] **Step 2: Remove dead wiring**

Delete:
- The `WithLogReader` call
- The `logReader` construction (the variable and its initialization)
- Any mode-specific handling for session-start or precompact if it exists

Do NOT remove: any `With*` calls for live options (tracker, surfacing logger, effectiveness, cross-ref checker, tool gate, etc.).

- [ ] **Step 3: Run `targ check-full`**

Run: `targ check-full`
Expected: Pass. This is the final verification.

- [ ] **Step 4: Commit**

```bash
git add internal/cli/cli.go
git commit -m "refactor: remove dead WithLogReader CLI wiring (#375)

AI-Used: [claude]"
```

### Task 5: Close issue

- [ ] **Step 1: Close GitHub issue #375**

```bash
gh issue close 375 --comment "Fixed: removed all dead session-start and precompact surface mode code.

Deleted: 13 functions, 6 types/interfaces, 6 With* options, 6 struct fields, 4 constants, ~40 dead test functions. Cleaned up budget.go dead fields and CLI wiring. Renamed sessionStartDefaultEffectiveness → defaultEffectiveness."
```
