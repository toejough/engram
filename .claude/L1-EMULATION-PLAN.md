# Layer -1 Emulation Plan

**Purpose:** Fix projctl issues blocking Layer -1 skill execution. Use projctl where it works, emulate correct behavior where it doesn't.

**RELOAD THIS FILE AFTER EVERY COMPACTION EVENT**

---

## Current Status

**Phase:** L-1 Unblocked (partial AC completion)
**Current Issue:** None active - remaining AC items documented in issues

---

## Issues to Resolve (in dependency order)

| # | Issue | Status | What to Fix |
|---|-------|--------|-------------|
| 1 | ISSUE-009 | ✅ DONE | State machine transitions - add missing phases, fix adopt order |
| 2 | ISSUE-010 | ⚠️ PARTIAL | Workflow/Issue fields done; Pairs map and Yield struct NOT done |
| 3 | ISSUE-013 | ⚠️ PARTIAL | `territory map` done; `territory show` NOT done |
| 4 | ISSUE-017 | ✅ DONE | Add `projctl state set` command |
| 5 | ISSUE-016 | ✅ DONE | Add `projctl issue create/update` commands |
| 6 | ISSUE-018 | ✅ DONE | Add `projctl yield validate` command |

---

## ISSUE-009: State Machine Transitions

### Problem
`internal/state/transitions.go` has wrong phases:
- Missing: `documentation`, `retro`, `summary`, `issue-update`, `next-steps`
- Missing: `adopt-explore`, `adopt-infer-tests`, `adopt-documentation`
- Wrong order: adopt is top-down, should be bottom-up
- Obsolete: `audit`, `audit-fix`, `audit-complete`, `adopt-map-tests`, `adopt-generate`, `integrate-*`

### Fix Plan
1. Update `LegalTransitions` map in `internal/state/transitions.go`
2. Update tests in `internal/state/transition_test.go`
3. Verify with `go test ./internal/state/...`

### New Workflow Phases

**New Project (top-down):**
```
init → pm → pm-complete → design → design-complete → architect → architect-complete →
breakdown → breakdown-complete → implementation → [task loop] → implementation-complete →
documentation → documentation-complete → alignment → alignment-complete →
retro → retro-complete → summary → summary-complete → issue-update → next-steps → complete
```

**Adopt (bottom-up):**
```
init → adopt-explore → adopt-infer-tests → adopt-infer-arch → adopt-infer-design →
adopt-infer-reqs → adopt-escalations → adopt-documentation → adopt-complete →
alignment → alignment-complete → retro → retro-complete → summary → summary-complete →
issue-update → next-steps → complete
```

**Task (single task):**
```
init → task-implementation → [tdd loop] → task-documentation →
alignment → alignment-complete → retro → retro-complete → summary → summary-complete →
issue-update → next-steps → complete
```

---

## Emulation Strategy

Until projctl is fixed, emulate by:
1. **State transitions**: Call `projctl state transition` for phases that exist, skip/log for phases that don't
2. **Workflow type**: Track in memory or temp file until ISSUE-010 is fixed
3. **Territory command**: Call `projctl map generate` instead of `projctl territory map`
4. **Issue commands**: Use direct file editing of `docs/issues.md` until ISSUE-016 is fixed
5. **State set**: Modify state.toml directly until ISSUE-017 is fixed
6. **Yield validate**: Skip validation until ISSUE-018 is fixed

---

## Files to Modify

| File | Changes |
|------|---------|
| `internal/state/transitions.go` | Update LegalTransitions map |
| `internal/state/transition_test.go` | Update tests for new phases |
| `internal/state/state.go` | Add Workflow field to Project struct |
| `cmd/projctl/state.go` | Add --mode flag to init, add set subcommand |
| `cmd/projctl/territory.go` | New file - rename from map command |
| `cmd/projctl/issue.go` | New file - issue commands |
| `cmd/projctl/yield.go` | New file - yield validate command |

---

## Progress Log

- 2026-02-03: Created emulation plan
- 2026-02-03: Starting ISSUE-009
- 2026-02-03: ISSUE-009 COMPLETE - transitions.go and tests updated, all tests pass
- 2026-02-03: Starting ISSUE-010
- 2026-02-03: ISSUE-010 COMPLETE - workflow/issue fields added to Project struct, InitOpts, CLI updated
- 2026-02-03: Starting ISSUE-013
- 2026-02-03: ISSUE-013 COMPLETE - renamed map.go to territory.go, `projctl territory map`
- 2026-02-03: Starting ISSUE-017
- 2026-02-03: ISSUE-017 COMPLETE - added Set() function and CLI command
- 2026-02-03: Starting ISSUE-016
- 2026-02-03: ISSUE-016 COMPLETE - added issue package and CLI commands
- 2026-02-03: Starting ISSUE-018
- 2026-02-03: ISSUE-018 COMPLETE - added yield package and CLI commands
- 2026-02-03: CORRECTION - ISSUE-010 and ISSUE-013 have incomplete AC items
  - ISSUE-010: Missing Pairs map and Yield struct
  - ISSUE-013: Missing `territory show` command
- 2026-02-03: L-1 unblocked but issues reopened with remaining AC documented

---

## After Compaction

If you've just reloaded after compaction:
1. Read this file to restore context
2. Check "Current Status" section for where we are
3. Check "Progress Log" for recent work
4. Continue with the current issue
