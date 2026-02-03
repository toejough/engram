# Tasks: parallel-worktree-strategy

## TASK-001: Create worktree package with path resolution

Create `internal/worktree/` package with `Manager` struct and `Path()` function.

**Files:** `internal/worktree/worktree.go`

**Acceptance Criteria:**
- [x] `Manager` struct with `RepoDir` field
- [x] `Path(taskID)` returns canonical worktree path
- [x] Tests verify path pattern: `<repo>/../<repo>-worktrees/<task>/`

**Traces to:** ARCH-001, DES-001

---

## TASK-002: Implement worktree Create

Add `Create(taskID)` method that creates branch and worktree.

**Files:** `internal/worktree/worktree.go`

**Acceptance Criteria:**
- [x] Creates parent directory if needed
- [x] Creates branch `task/<taskID>`
- [x] Creates worktree at canonical path
- [x] Returns worktree path
- [x] Tests verify worktree exists after create

**Dependencies:** TASK-001

**Traces to:** ARCH-001, ARCH-002, DES-002

---

## TASK-003: Implement worktree List

Add `List()` method returning active worktrees.

**Files:** `internal/worktree/worktree.go`

**Acceptance Criteria:**
- [x] Parses `git worktree list` output
- [x] Filters to task/* branches only
- [x] Returns slice of Worktree structs
- [x] Tests verify list after create

**Dependencies:** TASK-001

**Traces to:** ARCH-001

---

## TASK-004: Implement worktree Merge

Add `Merge(taskID, onto)` with rebase and FF merge.

**Files:** `internal/worktree/worktree.go`

**Acceptance Criteria:**
- [x] Rebases task branch onto target
- [x] Fast-forward merges to target
- [x] Returns `MergeConflictError` on conflict
- [x] Tests verify commits appear on target after merge

**Dependencies:** TASK-002

**Traces to:** ARCH-001, ARCH-002, ARCH-005, DES-003

---

## TASK-005: Implement worktree Cleanup

Add `Cleanup(taskID)` and `CleanupAll()` methods.

**Files:** `internal/worktree/worktree.go`

**Acceptance Criteria:**
- [x] Removes worktree directory
- [x] Deletes task branch
- [x] Removes parent dir if empty
- [x] `CleanupAll` removes all task worktrees
- [x] Tests verify cleanup leaves no artifacts

**Dependencies:** TASK-002

**Traces to:** ARCH-001, DES-004

---

## TASK-006: Add CLI commands

Add `projctl worktree` command group.

**Files:** `cmd/projctl/worktree.go`, `cmd/projctl/main.go`

**Acceptance Criteria:**
- [x] `projctl worktree create -t TASK-001` works
- [x] `projctl worktree list` works
- [x] `projctl worktree merge -t TASK-001` works
- [x] `projctl worktree cleanup -t TASK-001` works
- [x] `projctl worktree cleanup-all` works

**Dependencies:** TASK-002, TASK-003, TASK-004, TASK-005

**Traces to:** ARCH-003, DES-005

---

## TASK-007: Add state tracking for worktrees

Extend state.toml to track active worktrees.

**Files:** `internal/state/state.go`

**Acceptance Criteria:**
- [x] `Worktrees` map added to State struct
- [x] `WorktreeState` struct with path, branch, created, status
- [x] Tests verify worktree state persists

**Dependencies:** TASK-001

**Traces to:** ARCH-004, DES-007

---

## Dependency Graph

```
TASK-001 ──┬──► TASK-002 ──┬──► TASK-004 ──┬──► TASK-006
           │               │               │
           │               └──► TASK-005 ──┘
           │
           ├──► TASK-003 ─────────────────────► TASK-006
           │
           └──► TASK-007
```
