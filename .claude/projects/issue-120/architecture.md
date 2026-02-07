# Architecture: Task Parallelization in projctl step next

**Issue:** ISSUE-120
**Created:** 2026-02-06
**Status:** Draft

**Traces to:** ISSUE-120, requirements.md, design.md

---

## Overview

This architecture implements task parallelization by extending `projctl step next` to detect and return multiple unblocked tasks for parallel execution. The implementation modifies existing Go packages (`internal/step`, `internal/task`, `internal/worktree`) to support the unified array-based response format and immediate execution model defined in the design.

**Key Architectural Principle:** Minimize changes to existing codebase; reuse established patterns for JSON serialization, git operations, and state management.

---

## Architecture Decisions

### ARCH-1: Array-Based JSON Response Structure

**Traces to:** DES-1, REQ-3

Extend `NextResult` struct in `internal/step/next.go` to support array-based task responses.

**Current Structure:**

```go
type NextResult struct {
    Action    string      `json:"action"`
    Skill     string      `json:"skill,omitempty"`
    // ... other fields
}
```

**New Structure:**

```go
type NextResult struct {
    Action    string      `json:"action"`
    Tasks     []TaskInfo  `json:"tasks"`           // New: array of unblocked tasks
    // ... backward compatibility fields (deprecated)
}

type TaskInfo struct {
    ID       string  `json:"id"`
    Command  string  `json:"command"`
    Worktree *string `json:"worktree"` // nullable: null for sequential, path for parallel
}
```

**Implementation Details:**

- `Tasks` array replaces single-task response pattern
- Empty array (`[]`) indicates no unblocked tasks
- Single-element array indicates sequential execution (worktree: null)
- Multi-element array indicates parallel execution (worktree: non-null paths)
- Use Go's `encoding/json` standard library for marshaling

**Rationale:** Minimal struct changes; array length naturally indicates execution mode. Standard library JSON marshaling is sufficient for this format.

**Alternatives considered:**
- Separate response types for sequential vs parallel: Rejected (adds complexity)
- Top-level `mode` field: Rejected (redundant with array length)

---

### ARCH-2: Task Detection via Dependency Graph Analysis

**Traces to:** DES-6, REQ-1, REQ-5

Reuse existing `internal/task/deps.go` `Parallel()` function to detect unblocked tasks.

**Existing Implementation:**

```go
// Parallel returns tasks that can run in parallel (pending with no pending blockers).
func Parallel(dir string) ([]string, error) {
    graph, err := ParseDependencies(dir)
    // ... returns list of unblocked task IDs
}
```

**Integration Point:**

Modify `Next()` in `internal/step/next.go` to:
1. Call `task.Parallel(dir)` to get unblocked task IDs
2. For each task ID, construct `TaskInfo` with command and worktree path
3. Return `NextResult{Tasks: []TaskInfo{...}}`

**Rationale:** Dependency graph parsing already exists and works. No need to rewrite task detection logic.

---

### ARCH-3: Worktree Path Assignment Using Existing Manager

**Traces to:** DES-2, DES-5, REQ-2

Reuse `internal/worktree.Manager` for worktree path generation and lifecycle management.

**Existing Implementation:**

```go
type Manager struct {
    repoDir string
}

func (m *Manager) Path(taskID string) string {
    // Pattern: <repo>/../<repo-name>-worktrees/<task-id>/
}

func (m *Manager) Create(taskID string) (string, error)
func (m *Manager) Merge(taskID, onto string) error
func (m *Manager) Cleanup(taskID string) error
```

**Integration:**

When `Next()` returns multiple tasks:
- Set `worktree` field to `manager.Path(taskID)` for each task
- Orchestrator calls `manager.Create(taskID)` for each worktree
- Orchestrator executes task commands in worktree directories
- Orchestrator calls `manager.Merge(taskID, "main")` after completion
- Orchestrator calls `manager.Cleanup(taskID)` after merge

**Rationale:** Worktree manager already implements the full lifecycle. No architectural changes needed; orchestrator drives the workflow using existing APIs.

---

### ARCH-4: Status Command Implementation

**Traces to:** DES-3

Implement `projctl step status` as a new command in `cmd/projctl/step.go`.

**Data Sources:**

1. **Active tasks:** Check for existing worktrees via `worktree.Manager.List()`
2. **Completed tasks:** Parse state file (existing pattern in `internal/state`)
3. **Blocked tasks:** Use dependency graph from `internal/task/deps.go`

**Implementation:**

```go
type StatusResult struct {
    ActiveTasks    []ActiveTaskInfo    `json:"active_tasks"`
    CompletedTasks []CompletedTaskInfo `json:"completed_tasks"`
    BlockedTasks   []BlockedTaskInfo   `json:"blocked_tasks"`
}

func Status(dir string) (StatusResult, error) {
    // 1. List worktrees -> active tasks
    // 2. Read state -> completed tasks
    // 3. Parse dependencies -> blocked tasks
}
```

**Rationale:** Combines existing data sources (worktree list, state file, dependency graph) without introducing new state tracking mechanisms.

---

### ARCH-5: Removal of File Overlap Detection

**Traces to:** DES-7, REQ-4

Delete `projctl tasks overlap` command and all related code.

**Files to Remove:**

- Any CLI command handler for `tasks overlap` in `cmd/projctl/tasks.go`
- Any overlap detection logic in `internal/task/`
- All tests for overlap detection

**Migration:**

- Update documentation to reflect removal
- Conflicts detected naturally during git rebase/merge (existing `worktree.Manager.Merge()` handles this)

**Rationale:** File overlap detection adds complexity without value. Git provides authoritative conflict detection during merge. Removal simplifies codebase.

---

### ARCH-6: Error Handling and Retry Logic

**Traces to:** DES-4

Orchestrator implements retry logic; `projctl step next` returns errors via exit codes and stderr.

**Error Categories:**

1. **Worktree creation failure:** Retry up to 3 times (orchestrator responsibility)
2. **Task execution failure:** Retry up to 3 times (orchestrator responsibility)
3. **Merge conflicts:** No retry; escalate to user immediately

**Implementation Pattern:**

```go
// In projctl step next
func Next(dir string) (NextResult, error) {
    // ... logic
    if err != nil {
        return NextResult{}, fmt.Errorf("task detection failed: %w", err)
    }
    return result, nil
}

// Exit code: 0 for success, 1 for error
// Error details written to stderr
```

**Merge Conflict Handling:**

Existing `worktree.Manager.Merge()` returns `*MergeConflictError`:

```go
type MergeConflictError struct {
    TaskID  string
    Message string
}
```

Orchestrator catches this error and escalates to user without retry.

**Rationale:** Follows existing error handling patterns in projctl. Orchestrator controls retry logic for flexibility.

---

### ARCH-7: Immediate Execution Model via Event-Driven Polling

**Traces to:** DES-6, REQ-5

Orchestrator calls `projctl step next` after every task completion (not just after all parallel tasks complete).

**Execution Flow:**

```
1. Orchestrator: projctl step next
2. Response: {"tasks": [A, B]}
3. Orchestrator: spawn A and B in parallel
4. Task A completes
5. Orchestrator: projctl step next (immediately)
6. Response: {"tasks": [C, D]} (newly unblocked by A)
7. Orchestrator: spawn C and D (B still running)
8. Task B completes
9. Orchestrator: projctl step next
10. ... repeat
```

**Implementation:**

No changes to `projctl step next` required. Logic already returns all unblocked tasks via `task.Parallel()`. Orchestrator drives the polling loop.

**Rationale:** Immediate execution is orchestrator behavior, not `projctl step next` behavior. Clean separation of concerns.

---

### ARCH-8: State Persistence Using Existing Patterns

**Traces to:** REQ-2

Reuse existing state file structure in `internal/state` for tracking task progress.

**Existing State Structure:**

```go
type State struct {
    Project struct {
        Name  string
        Phase string
        Issue string
    }
    Pairs map[string]PairState
}
```

**Extension for Task Tracking:**

Add task state tracking to existing state file (if not already present):

```go
type State struct {
    // ... existing fields
    Tasks map[string]TaskState `json:"tasks,omitempty"`
}

type TaskState struct {
    ID        string    `json:"id"`
    Status    string    `json:"status"` // pending, active, complete
    StartedAt time.Time `json:"started_at,omitempty"`
    CompletedAt time.Time `json:"completed_at,omitempty"`
}
```

**Rationale:** Extends existing state persistence pattern. No new persistence mechanisms needed.

---

### ARCH-9: Concurrent Worktree Creation Using Goroutines

**Traces to:** REQ-2

Orchestrator can parallelize worktree creation using Go goroutines for performance.

**Example Pattern:**

```go
var wg sync.WaitGroup
errs := make([]error, len(tasks))

for i, task := range tasks {
    wg.Add(1)
    go func(i int, taskID string) {
        defer wg.Done()
        _, err := manager.Create(taskID)
        errs[i] = err
    }(i, task.ID)
}

wg.Wait()
```

**Rationale:** Git worktree creation is I/O bound. Parallel creation speeds up initialization. This is an orchestrator optimization; `projctl step next` remains synchronous.

---

### ARCH-10: Git Command Execution via os/exec

**Traces to:** REQ-2

Continue using `os/exec.Command` for git operations (existing pattern in `internal/worktree`).

**Existing Pattern:**

```go
func (m *Manager) git(args ...string) (string, error) {
    cmd := exec.Command("git", args...)
    cmd.Dir = m.repoDir
    output, err := cmd.CombinedOutput()
    return string(output), err
}
```

**Rationale:** Standard library `os/exec` is sufficient for git operations. No need for external libraries or git libraries (e.g., go-git) which add complexity and dependencies.

**Alternatives considered:**
- `go-git` library: Rejected (adds dependency, more complexity than needed)
- Git over HTTP API: Rejected (local operations only)

---

## Implementation Plan Summary

**Phase 1: Response Format Changes**
- Modify `NextResult` struct to include `Tasks []TaskInfo`
- Update `Next()` to call `task.Parallel()` and populate array
- Add tests for empty, single-task, multi-task responses

**Phase 2: Worktree Path Assignment**
- Integrate `worktree.Manager.Path()` into `Next()` response
- Set `worktree` field to null for single-task case, non-null for parallel

**Phase 3: Status Command**
- Implement `Status()` function in `internal/step`
- Add CLI command `projctl step status` in `cmd/projctl/step.go`
- Combine worktree list, state file, dependency graph data

**Phase 4: Remove Overlap Detection**
- Delete overlap detection command and code
- Remove related tests
- Update documentation

**Phase 5: Integration Testing**
- End-to-end tests for parallel execution
- Merge conflict handling
- Immediate execution model verification

---

## Technology Stack Summary

| Component | Technology | Rationale |
|-----------|------------|-----------|
| Language | Go 1.25.6 | Existing codebase |
| JSON marshaling | `encoding/json` | Standard library, sufficient |
| Git operations | `os/exec` + git CLI | Existing pattern, simple |
| Concurrency | Goroutines + sync.WaitGroup | Standard library, lightweight |
| State persistence | JSON files | Existing pattern |
| Dependency management | `internal/task/deps.go` | Existing implementation |
| Worktree management | `internal/worktree.Manager` | Existing implementation |

---

## Traceability Matrix

| ARCH ID | Traces to |
|---------|-----------|
| ARCH-1 | DES-1, REQ-3 |
| ARCH-2 | DES-6, REQ-1, REQ-5 |
| ARCH-3 | DES-2, DES-5, REQ-2 |
| ARCH-4 | DES-3 |
| ARCH-5 | DES-7, REQ-4 |
| ARCH-6 | DES-4 |
| ARCH-7 | DES-6, REQ-5 |
| ARCH-8 | REQ-2 |
| ARCH-9 | REQ-2 |
| ARCH-10 | REQ-2 |

---

## Open Questions

None. All architectural decisions are based on requirements, design, and existing codebase patterns.
