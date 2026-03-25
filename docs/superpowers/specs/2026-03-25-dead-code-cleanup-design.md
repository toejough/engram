# Dead Code Cleanup — Remove Unused Surface Modes (#375)

## Problem

`runSessionStart` and `runPreCompact` in `internal/surface/surface.go` are dead code. Their hook callers were removed (session-recall redesign 2026-03-21, #350 precompact no-op). 13 functions, 6 types/interfaces, 6 `With*` options, 9 constants, and ~15 test functions exist with no production callers.

## Approach

Bottom-up deletion in 4 commits. Each commit leaves the build green (`targ check-full` passes).

## Deletion Inventory

### Dead functions (13)

| Function | File | Called only from |
|----------|------|-----------------|
| `runSessionStart` | surface.go:650 | `Run()` switch |
| `runPreCompact` | surface.go:379 | `Run()` switch |
| `filterByEffectivenessGate` | surface.go:1350 | `runSessionStart` |
| `sortByEffectivenessScore` | surface.go:1559 | `runSessionStart` |
| `applySpreadingActivation` | surface.go:1199 | `runSessionStart` |
| `sortByActivatedScore` | surface.go:1551 | `runSessionStart` |
| `applyColdStartBudgetStored` | surface.go:1141 | `runSessionStart` |
| `suppressClusterDuplicates` | suppress_p4f.go:135 | `runSessionStart` |
| `writeCreationSection` | surface.go:1605 | `runSessionStart` |
| `writeRecencySection` | surface.go:1631 | `runSessionStart` |
| `updateCoSurfacingLinks` | surface.go:908 | `runSessionStart` |
| `appendClusterNotes` | surface.go:276 | `runSessionStart` |
| `suppressContradictions` | surface.go:850 | `runSessionStart` |

### Dead types/interfaces (6)

| Name | File | Used only by |
|------|------|-------------|
| `ContradictionDetector` | surface.go:41 | `suppressContradictions` |
| `CreationLogReader` | surface.go:46 | `runSessionStart` |
| `LogEntry` | surface.go:97 | `runSessionStart`, `writeCreationSection` |
| `LinkUpdater` | surface.go:91 | `updateCoSurfacingLinks` |
| `SignalEmitter` | surface.go (search) | `suppressContradictions` |
| `TitleFetcher` | surface.go (search) | `appendClusterNotes` |

### Dead `With*` options (6)

| Function | Injects field used only by dead path |
|----------|-------------------------------------|
| `WithLogReader` | `s.logReader` → `runSessionStart` |
| `WithLinkUpdater` | `s.linkUpdater` → `updateCoSurfacingLinks` |
| `WithTitleFetcher` | `s.titleFetcher` → `appendClusterNotes` |
| `WithClusterDedupReader` | `s.clusterDedupReader` → `suppressClusterDuplicates` |
| `WithContradictionDetector` | `s.contradictionDetector` → `suppressContradictions` |
| `WithSignalEmitter` | `s.signalEmitter` → `suppressContradictions` |

### Dead constants (9)

| Name | File |
|------|------|
| `ModeSessionStart` | surface.go:31 |
| `ModePreCompact` | surface.go:29 |
| `sessionStartLimit` | surface.go |
| `spreadingActivationDecay` | surface.go |
| `minPreCompactEffectiveness` | surface.go |
| `preCompactLimit` | surface.go |
| `DefaultSessionStartBudget` | budget.go |
| `DefaultPreCompactBudget` | budget.go |
| `BudgetConfig.SessionStart` / `BudgetConfig.PreCompact` fields | budget.go |

### Dead budget infrastructure

- `BudgetConfig.SessionStart` and `BudgetConfig.PreCompact` struct fields
- `ForMode` cases for `ModeSessionStart` and `ModePreCompact` (method itself stays — used by live modes)
- `DefaultBudgetConfig()` — only called in `budget_test.go`; delete along with dead budget tests
- `DefaultSessionStartBudget` and `DefaultPreCompactBudget` constants

### Dead `Surfacer` struct fields (6)

The fields injected by the dead `With*` options:
- `logReader`, `linkUpdater`, `titleFetcher`, `clusterDedupReader`, `contradictionDetector`, `signalEmitter`

### CLI wiring

- `cli.go:1516` — `surface.WithLogReader(logReader)` call and the `logReader` construction above it

### Dead test files/functions

| File | Dead content |
|------|-------------|
| `contradict_test.go` | Entire file — tests `suppressContradictions` |
| `cold_start_budget_test.go` | Tests for `applyColdStartBudgetStored` only (verify — may also test live `applyColdStartBudgetPrompt`/`Tool`) |
| `p4f_test.go` | Tests for `suppressClusterDuplicates` |
| `p3_test.go` | Session-start-specific tests (verify which tests are prompt-mode live) |
| `surface_test.go` | ~13 session-start and precompact test functions |
| `p4e_test.go` | Session-start-specific tests (verify which are shared) |
| `export_test.go` | Remove dead exports (`ExportSessionStartDefaultEffectiveness` after rename) |
| `helpers_test.go` | Update references to renamed `sessionStartDefaultEffectiveness` |
| `budget_test.go` | Remove tests for `DefaultBudgetConfig()` and dead budget fields |

### Rename (not delete)

| Name | Why | New name |
|------|-----|----------|
| `sessionStartDefaultEffectiveness` | Used by live `effectivenessScoreFor` → `filterToolMatchesByEffectivenessGate`. Value (50.0) is a neutral default, not session-start-specific. | `defaultEffectiveness` |

### Shared items — DO NOT DELETE

| Name | Why shared |
|------|-----------|
| `effectivenessScoreFor` | Used by live `filterToolMatchesByEffectivenessGate` |
| `isUnproven` | Used by live `applyColdStartBudgetPrompt`/`Tool` and `unprovenBM25Floor*` |
| `coldStartBudget` | Used by live `applyColdStartBudgetPrompt`/`Tool` |
| `CrossRefChecker` | Used by `runPrompt` |
| `LinkReader` | Used by `runPrompt` and `runTool` |
| `suppressByCrossRef` | Used by `runPrompt` and `runSessionStart` |
| `suppressByTranscript` | Used by `runPrompt` |
| `computeSuppressionStats` | Used by `Run()` for all modes |
| `SuppressionEvent`/`SuppressionStats` | Used by all modes |
| `applyPromptBudget` | Used by `runPrompt` |

## Commit ordering

1. **Delete dead tests** — removes consumers first
2. **Delete dead functions, types, interfaces, constants, struct fields** from surface.go and suppress_p4f.go
3. **Clean up budget.go** — remove dead fields, cases, constants
4. **Clean up CLI wiring** — remove dead `With*` call and associated construction in cli.go

Run `targ check-full` after each commit.

## Not in scope

- Removing the `internal/creationlog/` package (it still has a write path used by UC-1)
- Refactoring surface.go for size (separate concern)
- Removing spec references (handled by #377)
