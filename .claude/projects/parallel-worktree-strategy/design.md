# Design: parallel-worktree-strategy

## DES-001: Worktree path resolution

**Pattern:** `<repo-parent>/<repo-name>-worktrees/<task-id>/`

Example for `/Users/joe/repos/personal/projctl` with TASK-001:
```
/Users/joe/repos/personal/projctl-worktrees/TASK-001/
```

Implementation:
```go
func WorktreePath(repoDir, taskID string) string {
    repoName := filepath.Base(repoDir)
    parent := filepath.Dir(repoDir)
    return filepath.Join(parent, repoName+"-worktrees", taskID)
}
```

**Traces to:** REQ-001

---

## DES-002: Worktree creation flow

```
1. Resolve worktree path
2. Create parent dir if needed: <repo>-worktrees/
3. Create branch: git branch task/<task-id>
4. Add worktree: git worktree add <path> task/<task-id>
5. Return path to caller
```

No remote push during creation (short-lived branches).

**Traces to:** REQ-002

---

## DES-003: Merge flow

```
1. From main worktree (not task worktree):
   git fetch . task/<task-id>:task/<task-id>  # ensure branch visible
2. git rebase <target> task/<task-id>
3. If conflict: return error with details
4. git checkout <target>
5. git merge --ff-only task/<task-id>
6. Cleanup (DES-004)
```

**Traces to:** REQ-004, REQ-005, REQ-007

---

## DES-004: Cleanup flow

```
1. git worktree remove <path>  # or rm -rf if locked
2. git branch -d task/<task-id>
3. rmdir <parent> if empty (non-fatal if fails)
```

**Traces to:** REQ-006

---

## DES-005: Command structure

```
projctl worktree create -t TASK-001 [-d <project-dir>]
  → Creates worktree, prints path

projctl worktree list [-d <project-dir>]
  → Lists task worktrees with status

projctl worktree merge -t TASK-001 [--onto <branch>] [-d <project-dir>]
  → Rebases and FF merges, cleans up on success

projctl worktree cleanup -t TASK-001 [-d <project-dir>]
  → Force removes worktree and branch

projctl worktree cleanup-all [-d <project-dir>]
  → Removes all task/* worktrees
```

**Traces to:** REQ-008

---

## DES-006: Integration with parallel orchestration

Orchestrator flow for parallel tasks:

```
1. For each task in parallel batch:
   - worktree create → get path
   - spawn agent with CWD=worktree path
   - agent does TDD cycle with normal commits

2. Wait for all agents

3. For each completed task (in dependency order):
   - worktree merge --onto <current-branch>
   - If conflict: mark task failed, continue others

4. worktree cleanup-all
```

**Traces to:** REQ-002, REQ-003, REQ-004, REQ-005, REQ-006, REQ-007

---

## DES-007: State tracking for worktrees

Add to state.toml:
```toml
[worktrees]
  [worktrees.TASK-001]
    path = "/path/to/worktree"
    branch = "task/TASK-001"
    created = 2026-02-03T12:00:00Z
    status = "active"  # active, merged, failed
```

**Traces to:** REQ-002, REQ-008
