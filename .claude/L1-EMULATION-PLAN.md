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
| 1 | ISSUE-9 | ✅ DONE | State machine transitions - add missing phases, fix adopt order |
| 2 | ISSUE-10 | ⚠️ PARTIAL | Workflow/Issue fields done; Pairs map and Yield struct NOT done |
| 3 | ISSUE-13 | ⚠️ PARTIAL | `territory map` done; `territory show` NOT done |
| 4 | ISSUE-17 | ✅ DONE | Add `projctl state set` command |
| 5 | ISSUE-16 | ✅ DONE | Add `projctl issue create/update` commands |
| 6 | ISSUE-18 | ✅ DONE | Add `projctl yield validate` command |

---

## ISSUE-9: State Machine Transitions

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
2. **Workflow type**: Track in memory or temp file until ISSUE-10 is fixed
3. **Territory command**: Call `projctl map generate` instead of `projctl territory map`
4. **Issue commands**: Use direct file editing of `docs/issues.md` until ISSUE-16 is fixed
5. **State set**: Modify state.toml directly until ISSUE-17 is fixed
6. **Yield validate**: Skip validation until ISSUE-18 is fixed

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
- 2026-02-03: Starting ISSUE-9
- 2026-02-03: ISSUE-9 COMPLETE - transitions.go and tests updated, all tests pass
- 2026-02-03: Starting ISSUE-10
- 2026-02-03: ISSUE-10 COMPLETE - workflow/issue fields added to Project struct, InitOpts, CLI updated
- 2026-02-03: Starting ISSUE-13
- 2026-02-03: ISSUE-13 COMPLETE - renamed map.go to territory.go, `projctl territory map`
- 2026-02-03: Starting ISSUE-17
- 2026-02-03: ISSUE-17 COMPLETE - added Set() function and CLI command
- 2026-02-03: Starting ISSUE-16
- 2026-02-03: ISSUE-16 COMPLETE - added issue package and CLI commands
- 2026-02-03: Starting ISSUE-18
- 2026-02-03: ISSUE-18 COMPLETE - added yield package and CLI commands
- 2026-02-03: CORRECTION - ISSUE-10 and ISSUE-13 have incomplete AC items
  - ISSUE-10: Missing Pairs map and Yield struct
  - ISSUE-13: Missing `territory show` command
- 2026-02-03: L-1 unblocked but issues reopened with remaining AC documented

---

## After Compaction

If you've just reloaded after compaction:
1. Read this file to restore context
2. Check "Current Status" section for where we are
3. Check "Progress Log" for recent work
4. Continue with the current issue
