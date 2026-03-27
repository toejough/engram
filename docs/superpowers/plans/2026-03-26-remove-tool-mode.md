# Remove Dead ModeTool Code Path — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove all dead tool-mode surfacing code after hook consolidation (#384) eliminated the hooks that invoked it.

**Architecture:** The `ModeTool` code path in the surface package (matching, scoring, rendering, budget, gating) is entirely dead — no production code invokes `surface --mode tool`. Remove it along with the `toolgate` package, CLI flag wiring, and all tool-mode tests. Shared helpers used by both prompt and tool mode stay.

**Tech Stack:** Go, gomega test framework, targ build system

---

## Scope

### What Gets Removed (tool-only)

| Area | Items |
|------|-------|
| **surface.go** | `ModeTool` constant, `runTool()`, `toolGateAllows()`, `ToolGater` interface, `WithToolGate()`, `matchToolMemories()`, `toolMatch` struct, `bm25FloorForTool()`, `filterToolMatchesByEffectivenessGate()`, `sortToolMatchesByActivation()`, `applyColdStartBudgetTool()`, `renderToolAdvisories()`, `concatenateToolFields()`, tool-only constants (`toolLimit`, `unprovenBM25FloorTool`, `minEffectivenessFloor`) |
| **budget.go** | `DefaultToolBudget`, `BudgetConfig.Tool` field, `ForMode(ModeTool)` case, `EstimateToolMemoryTokens()`, `applyToolBudget()` |
| **Options struct** | `ToolName`, `ToolInput`, `ToolOutput`, `ToolErrored`, `Budget` fields |
| **cli.go** | `--tool-name/--tool-input/--tool-output/--tool-errored` flags, `fileCounterStore` type, `newFileCounterStore()`, tool gate wiring |
| **targets.go** | `ToolName`, `ToolInput`, `ToolOutput`, `ToolErrored` struct tags |
| **toolgate/** | Entire package (2 files) |
| **Tests** | ~20 tool-mode test functions across 8 test files |

### What Stays (shared with prompt mode)

`effectivenessScoreFor`, `isUnproven`, `isEmphasized`, `formatMemoryLine`, `irrelevancePenalty`, `irrelevancePenaltyHalfLife`, `coldStartBudget`, `applyColdStartBudgetPrompt`, `ErrUnknownMode`, `slugFromPath`

---

## Task 1: Delete tool-mode tests

Remove all test functions that exercise tool mode. Do this first so subsequent code deletions don't break tests.

**Files:**
- Modify: `internal/surface/surface_test.go`
- Modify: `internal/surface/p4e_test.go`
- Modify: `internal/surface/p6e_enforcement_test.go`
- Modify: `internal/surface/budget_test.go`
- Modify: `internal/surface/cold_start_budget_test.go`
- Modify: `internal/surface/unproven_bm25_test.go`
- Modify: `internal/cli/cli_test.go`
- Modify: `internal/cli/counter_store_test.go`
- Modify: `internal/cli/export_test.go`

- [ ] **Step 1: Delete tool-mode tests from surface_test.go**

Delete these test functions (and any test helpers only used by them):
- `TestQW1_ToolModeLimitsToTop2`
- `TestQW2_ToolRelevanceFloorFiltersLowScoring`
- `TestRunTool_NonBashSkips`
- `TestRunTool_ToolGateSkips`
- `TestT100_ToolModeNoMatchProducesEmpty`
- `TestT33_PreFilterMatchesKeywordsInToolInput`
- `TestT42_ToolModeEmitsAdvisoryReminder`
- `TestT71_ToolJSONFormat`
- `TestToolErrored_LowersUnprovenFloor`
- `TestToolOutput_EnrichesBM25Query`

Also delete any test helpers only used by tool tests: `stubToolGate`, `fakeToolGater`, or similar.

After deleting, check for unused imports (`surface.ModeTool` references) and remove them.

- [ ] **Step 2: Delete tool-mode tests from p4e_test.go**

Delete:
- `TestTP4e6_ToolModeLimitsToTop2`
- `TestTP4e7_ToolModeEffectivenessGating`

Update the file-level comment block to remove REQ-P4e-4 line.

If the file is left with only the `TestTP4e8_InvocationTokenLoggerCalled` test and its helpers, keep the file.

- [ ] **Step 3: Delete tool-mode tests from p6e_enforcement_test.go**

Delete:
- `TestP6e11_EmphasizedAdvisoryToolMode`
- `TestP6e12_ReminderToolMode`
- `TestP6e13_AdvisoryToolModeNormalFormat`

If the file is left empty (no prompt-mode enforcement tests), delete the entire file.

- [ ] **Step 4: Delete tool-mode test from budget_test.go**

Delete:
- `TestT192_ToolBudgetEnforcement`
- The `ForMode(surface.ModeTool)` assertion line in `TestT193_BudgetConfigCustomValues`

Update the `BudgetConfig` literal in `TestT193_BudgetConfigCustomValues` to remove the `Tool` field.

- [ ] **Step 5: Delete tool-mode test from cold_start_budget_test.go**

Delete:
- `TestColdStartBudgetLimitsUnprovenToolMemories`

If only prompt-mode cold-start tests remain, keep the file.

- [ ] **Step 6: Delete tool-mode test from unproven_bm25_test.go**

Delete:
- `TestUnprovenToolMemoryFilteredByHigherBM25Floor`

If only prompt-mode unproven tests remain, keep the file.

- [ ] **Step 7: Delete tool-mode CLI test and counter_store tests**

In `internal/cli/cli_test.go`, delete:
- `TestT42_SurfaceToolRouting`

Delete entirely:
- `internal/cli/counter_store_test.go`
- `internal/cli/export_test.go`

- [ ] **Step 8: Run tests**

Run: `targ test`
Expected: All tests pass. Some files may have unused import warnings — fix those.

- [ ] **Step 9: Commit**

```
refactor: delete tool-mode tests (#388)

All tool-mode test functions removed across 9 files. Preparing for
tool-mode code removal.
```

---

## Task 2: Delete tool-mode production code from surface package

**Files:**
- Modify: `internal/surface/surface.go`
- Modify: `internal/surface/budget.go`

- [ ] **Step 1: Delete tool-mode code from surface.go**

Remove these items (in order to avoid leaving dangling references):

Constants to delete:
- `toolLimit`
- `unprovenBM25FloorTool`
- `minEffectivenessFloor`

Delete the `ModeTool` constant and the `ModeTool` case from the `Run()` switch statement (lines 191-192). The `default` case already returns `ErrUnknownMode`.

Types to delete:
- `toolMatch` struct
- `ToolGater` interface

Functions/methods to delete:
- `runTool()`
- `toolGateAllows()`
- `matchToolMemories()`
- `bm25FloorForTool()`
- `filterToolMatchesByEffectivenessGate()`
- `sortToolMatchesByActivation()`
- `applyColdStartBudgetTool()`
- `renderToolAdvisories()`
- `concatenateToolFields()`
- `WithToolGate()` option

Fields to remove from Surfacer struct:
- `toolGate ToolGater` (or similar field name)

Fields to remove from Options struct:
- `ToolName`
- `ToolInput`
- `ToolOutput`
- `ToolErrored`
- `Budget` (dead since precompact removal)

Update package doc comment (line 2) if it still references tool mode.

Remove the `"engram/internal/toolgate"` import.

- [ ] **Step 2: Delete tool-mode code from budget.go**

Remove:
- `DefaultToolBudget` constant
- `Tool` field from `BudgetConfig` struct
- `ModeTool` case from `ForMode()` switch
- `EstimateToolMemoryTokens()` function
- `applyToolBudget()` function

The `ForMode` method should only have the `ModePrompt` case returning `c.UserPromptSubmit`, then `default: return 0`.

Update `DefaultBudgetConfig()` to not set `Tool`.

- [ ] **Step 3: Run tests**

Run: `targ test`
Expected: All pass (tool tests were already deleted in Task 1).

- [ ] **Step 4: Commit**

```
refactor: remove tool-mode surfacing code from surface package (#388)

Delete runTool(), matchToolMemories(), renderToolAdvisories(),
toolMatch type, ToolGater interface, budget config for tool mode,
and all related constants/helpers. ModeTool is no longer a valid
surface mode.
```

---

## Task 3: Delete tool-mode CLI code and toolgate package

**Files:**
- Modify: `internal/cli/cli.go`
- Modify: `internal/cli/targets.go`
- Delete: `internal/toolgate/toolgate.go`
- Delete: `internal/toolgate/toolgate_test.go`

- [ ] **Step 1: Delete tool flags and wiring from cli.go**

In `runSurface()`:
- Delete flag definitions: `toolName`, `toolInput`, `toolOutput`, `toolErrored`
- Delete the Options field assignments for those flags (lines 1596-1599)
- Delete tool frecency gate wiring block (lines 1638-1641: `newFileCounterStore`, `toolgate.NewGate`, `WithToolGate`)
- Update mode flag description from `"surface mode: prompt, tool, stop"` to `"surface mode: prompt, stop"`

Delete the `fileCounterStore` type and its methods:
- `fileCounterStore` struct
- `Load()` method
- `Save()` method
- `newFileCounterStore()` function

Remove the `"engram/internal/toolgate"` import.

- [ ] **Step 2: Delete tool fields from targets.go**

In `SurfaceArgs` struct, remove:
- `ToolName` field and its struct tags
- `ToolInput` field and its struct tags
- `ToolOutput` field and its struct tags
- `ToolErrored` field and its struct tags

- [ ] **Step 3: Delete the toolgate package**

Delete both files:
- `internal/toolgate/toolgate.go`
- `internal/toolgate/toolgate_test.go`

- [ ] **Step 4: Run tests and full checks**

Run: `targ test`
Expected: All pass.

Run: `targ check-full`
Expected: Only pre-existing failures (coverage on pre-existing functions). No new failures. Fix any reorder-decls issues with `targ reorder-decls`.

- [ ] **Step 5: Commit**

```
refactor: remove tool-mode CLI flags and toolgate package (#388)

Delete --tool-name/--tool-input/--tool-output/--tool-errored flags,
fileCounterStore, tool frecency gate wiring, and the entire
internal/toolgate package. Closes #388.
```

---

## Task 4: Clean up stale spec references

**Files:**
- Modify: `docs/specs/architecture.toml` (remove PreToolUse/PostToolUse/PostToolUseFailure references)
- Modify: `docs/specs/design.toml` (remove tool-mode hook references)
- Modify: `docs/specs/requirements.toml` (remove REQ-P4e-4 and other tool-hook requirements)
- Modify: `docs/specs/tests.toml` (remove tool-mode test specs)

- [ ] **Step 1: Audit spec files for stale tool-mode references**

Search each spec file for: `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `PreCompact`, `--mode tool`, `tool_name`, `tool_input`, `tool-frecency`, `toolgate`.

For each match: if the section describes behavior that no longer exists, delete the section. If it's a passing reference in a broader section, update the text to reflect current behavior.

- [ ] **Step 2: Run tests**

Run: `targ test`
Expected: All pass (spec files are documentation, no test impact).

- [ ] **Step 3: Commit**

```
docs: remove stale tool-mode references from spec files (#388)

Update architecture, design, requirements, and test specs to reflect
hook consolidation. Tool-mode surfacing sections removed.
```
