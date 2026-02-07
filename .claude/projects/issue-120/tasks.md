# Tasks: Task Parallelization in projctl step next

**Issue:** ISSUE-120
**Created:** 2026-02-06
**Status:** Draft

**Traces to:** requirements.md, design.md, architecture.md

---

## Simplicity Rationale

This task breakdown is as simple as possible because:

1. **Maximal Code Reuse:** The implementation leverages three existing, proven subsystems:
   - `internal/task/deps.go` - Already has `Parallel()` function for detecting unblocked tasks
   - `internal/worktree` - Already has complete worktree lifecycle management
   - `internal/state` - Already has task state persistence patterns

2. **Minimal Structural Changes:** The core change is adding a `Tasks []TaskInfo` field to the existing `NextResult` struct. No new data structures, no new architectural patterns.

3. **Deletion Over Addition:** One entire task (TASK-5) is pure deletion - removing the non-existent overlap detection code from documentation references only.

4. **No External Dependencies:** Uses only Go standard library (`encoding/json`, `os/exec`, existing internal packages). No new libraries or frameworks.

5. **Explicitly Scoped:** Out-of-scope items (scheduling algorithms, resource limits, orchestrator changes) are deferred, keeping this focused on the minimal core: "return array of unblocked tasks."

**Alternatives Considered:**
- Building a new parallel execution subsystem from scratch: Rejected (ignores working code)
- Adding smart scheduling/prioritization: Rejected (adds complexity without clear value)
- Creating new state tracking mechanisms: Rejected (existing state files are sufficient)

The simplicity emerges from the architecture's decision to **orchestrate existing pieces** rather than build new ones.

---

## Dependency Graph

```
TASK-1 (NextResult struct modification)
    |
    +--- TASK-2 (Next() array response)
    |         |
    |         +--- TASK-3 (Worktree path assignment)
    |         |
    |         +--- TASK-6 (Integration tests)
    |
    +--- TASK-4 (Status command)
    |
TASK-5 (Remove overlap detection - independent)
```

**Parallel Opportunities:**
- TASK-1 and TASK-5 can run in parallel (completely independent)
- After TASK-2 completes, TASK-3 and TASK-4 can run in parallel
- TASK-6 must wait for TASK-2 and TASK-3

---

## Tasks

### TASK-1: Add Tasks Array to NextResult Struct

**Description:** Extend the `NextResult` struct in `internal/step/next.go` to include a `Tasks []TaskInfo` field for array-based responses. Define the `TaskInfo` struct with `ID`, `Command`, and `Worktree` fields.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `NextResult` struct has `Tasks []TaskInfo` field with JSON tag `"tasks"`
- [ ] `TaskInfo` struct defined with fields: `ID string`, `Command string`, `Worktree *string`
- [ ] `Worktree` field is a pointer to string (nullable: null for sequential, path for parallel)
- [ ] All existing tests continue to pass
- [ ] Unit test verifies JSON marshaling of empty array, single task, multiple tasks
- [ ] Unit test verifies worktree field marshals to `null` when pointer is nil

**Files:**
- `/Users/joe/repos/personal/projctl/internal/step/next.go`
- `/Users/joe/repos/personal/projctl/internal/step/next_test.go` (new tests)

**Dependencies:** None

**Traces to:** ARCH-1, DES-1, REQ-3

---

### TASK-2: Modify Next() to Return Array of Unblocked Tasks

**Description:** Update the `Next()` function in `internal/step/next.go` to detect all unblocked tasks using `task.Parallel()` and populate the `Tasks` array in the response. When multiple tasks are unblocked, populate the array; when one task is unblocked, return single-element array; when none are unblocked, return empty array.

**Status:** Ready

**Acceptance Criteria:**
- [ ] `Next()` calls `task.Parallel(dir)` to get list of unblocked task IDs
- [ ] For each unblocked task, construct `TaskInfo{ID: taskID, Command: "projctl run " + taskID, Worktree: nil}`
- [ ] Populate `NextResult.Tasks` with array of `TaskInfo` structs
- [ ] Empty array returned when no tasks are unblocked (edge case EC-1)
- [ ] Single-element array returned when one task unblocked (edge case EC-2)
- [ ] Multi-element array returned when multiple tasks unblocked
- [ ] Unit tests verify array population for 0, 1, N unblocked tasks
- [ ] Unit tests verify correct command string format
- [ ] Property test verifies command format holds for random task IDs

**Files:**
- `/Users/joe/repos/personal/projctl/internal/step/next.go`
- `/Users/joe/repos/personal/projctl/internal/step/next_test.go`

**Dependencies:** TASK-1

**Traces to:** ARCH-2, DES-1, DES-6, REQ-1, REQ-5

---

### TASK-3: Assign Worktree Paths to Parallel Tasks

**Description:** Integrate `worktree.Manager.Path()` into the `Next()` function to assign worktree paths when multiple tasks are unblocked. Sequential execution (single task) keeps `worktree: null`; parallel execution (multiple tasks) assigns unique worktree paths.

**Status:** Ready

**Acceptance Criteria:**
- [ ] When `Tasks` array has length > 1, assign worktree paths to all tasks
- [ ] Use `worktree.Manager.Path(taskID)` for each task's worktree path
- [ ] When `Tasks` array has length <= 1, keep worktree as `nil` (sequential execution)
- [ ] Unit test verifies single task has `worktree: null`
- [ ] Unit test verifies multiple tasks have non-null, unique worktree paths
- [ ] Unit test verifies worktree path format matches `<repo>/../<repo-name>-worktrees/<task-id>/`
- [ ] Property test verifies path uniqueness across random task sets

**Files:**
- `/Users/joe/repos/personal/projctl/internal/step/next.go`
- `/Users/joe/repos/personal/projctl/internal/step/next_test.go`

**Dependencies:** TASK-2

**Traces to:** ARCH-3, DES-2, DES-5, REQ-2

---

### TASK-4: Implement projctl step status Command

**Description:** Create a new `Status()` function in `internal/step` and add the `projctl step status` CLI command. The command returns JSON with active tasks (from worktree list), completed tasks (from state file), and blocked tasks (from dependency graph).

**Status:** Ready

**Acceptance Criteria:**
- [ ] `Status(dir string)` function implemented in `internal/step/status.go`
- [ ] `StatusResult` struct defined with `ActiveTasks`, `CompletedTasks`, `BlockedTasks` fields
- [ ] `ActiveTasks` populated by calling `worktree.Manager.List()`
- [ ] `CompletedTasks` populated by reading state file via `state.Get(dir)`
- [ ] `BlockedTasks` populated by parsing dependency graph via `task.ParseDependencies(dir)`
- [ ] CLI command `projctl step status` added to `cmd/projctl/step.go`
- [ ] Unit test verifies correct JSON output format
- [ ] Unit test verifies active tasks detection from worktree list
- [ ] Unit test verifies completed tasks detection from state file
- [ ] Unit test verifies blocked tasks detection from dependency graph
- [ ] Integration test verifies status output with mixed task states

**Files:**
- `/Users/joe/repos/personal/projctl/internal/step/status.go` (new file)
- `/Users/joe/repos/personal/projctl/internal/step/status_test.go` (new file)
- `/Users/joe/repos/personal/projctl/cmd/projctl/step.go` (modify to add status subcommand)

**Dependencies:** TASK-1

**Traces to:** ARCH-4, DES-3

---

### TASK-5: Remove File Overlap Detection References

**Description:** Remove all references to file overlap detection from documentation. Note: The `projctl tasks overlap` command does not currently exist in the codebase, so this task only addresses documentation cleanup.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Search codebase for "overlap" references in documentation files
- [ ] Remove or update documentation that references `projctl tasks overlap`
- [ ] Update any design/architecture docs that mention overlap detection
- [ ] Verify no code references to overlap detection exist (command doesn't exist)
- [ ] Documentation states conflicts are handled by git rebase/merge, not pre-execution detection

**Files:**
- `/Users/joe/repos/personal/projctl/.claude/projects/parallel-execution-improvements/design.md` (if references exist)
- `/Users/joe/repos/personal/projctl/.claude/projects/parallel-execution-improvements/requirements.md` (if references exist)
- Any other documentation files that reference overlap detection

**Dependencies:** None

**Traces to:** ARCH-5, DES-7, REQ-4

---

### TASK-6: [visual] Integration Tests for Parallel Execution

**Description:** Create end-to-end integration tests verifying the full parallel execution flow, including array response format, worktree path assignment, immediate execution model, and error handling scenarios.

**Status:** Ready

**Acceptance Criteria:**
- [ ] Integration test: Create project with 3 unblocked tasks, verify `Next()` returns array of 3 tasks
- [ ] Integration test: Verify each task in array has unique worktree path (non-null)
- [ ] Integration test: Verify single unblocked task returns single-element array with `worktree: null`
- [ ] Integration test: Verify no unblocked tasks returns empty array
- [ ] Integration test: Simulate task A completing and unblocking tasks B, C; verify immediate execution (EC-3)
- [ ] Integration test: Verify status command shows correct active/completed/blocked tasks
- [ ] Integration test: Verify JSON output can be parsed by standard tools (jq, etc.)
- [ ] Property test: Verify response format holds across randomized task dependency graphs
- [ ] Visual test: Screenshot of `projctl step status` JSON output, verify formatting

**Files:**
- `/Users/joe/repos/personal/projctl/internal/step/integration_test.go` (extend existing file)
- `/Users/joe/repos/personal/projctl/internal/step/property_test.go` (extend existing file)

**Dependencies:** TASK-2, TASK-3

**Traces to:** DES-4, DES-5, DES-6, REQ-1, REQ-2, REQ-5

---

## Summary

**Total Tasks:** 6
**Independent Tasks:** 2 (TASK-1, TASK-5)
**Sequential Tasks:** 4 (TASK-2, TASK-3, TASK-4, TASK-6)
**Visual Tasks:** 1 (TASK-6)

**Critical Path:**
TASK-1 → TASK-2 → TASK-3 → TASK-6

**Parallel Execution Opportunities:**
1. Start: TASK-1 and TASK-5 in parallel
2. After TASK-2: TASK-3 and TASK-4 in parallel
3. Final: TASK-6 (waits for TASK-2 and TASK-3)

**Estimated Complexity:**
- TASK-1: Low (struct modification)
- TASK-2: Medium (logic change, integration with task.Parallel)
- TASK-3: Low (path assignment)
- TASK-4: Medium (new command, data aggregation)
- TASK-5: Low (documentation cleanup)
- TASK-6: High (comprehensive integration tests)

---

## Traceability Matrix

| Task | Requirements | Design | Architecture |
|------|--------------|--------|--------------|
| TASK-1 | REQ-3 | DES-1 | ARCH-1 |
| TASK-2 | REQ-1, REQ-5 | DES-1, DES-6 | ARCH-2 |
| TASK-3 | REQ-2 | DES-2, DES-5 | ARCH-3 |
| TASK-4 | - | DES-3 | ARCH-4 |
| TASK-5 | REQ-4 | DES-7 | ARCH-5 |
| TASK-6 | REQ-1, REQ-2, REQ-5 | DES-4, DES-5, DES-6 | - |

**Architecture Coverage:**
- ARCH-1: TASK-1
- ARCH-2: TASK-2
- ARCH-3: TASK-3
- ARCH-4: TASK-4
- ARCH-5: TASK-5
- ARCH-6: Implicit in acceptance criteria (error handling patterns)
- ARCH-7: TASK-2 (immediate execution model)
- ARCH-8: TASK-4 (state persistence)
- ARCH-9: Out of scope (orchestrator optimization)
- ARCH-10: Existing pattern (no new task needed)

**Requirements Coverage:**
- REQ-1: TASK-2, TASK-6
- REQ-2: TASK-3, TASK-6
- REQ-3: TASK-1
- REQ-4: TASK-5
- REQ-5: TASK-2, TASK-6

All requirements, design decisions, and architecture decisions are covered by the task breakdown.
