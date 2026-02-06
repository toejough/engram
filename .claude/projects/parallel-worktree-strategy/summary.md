# Summary: parallel-worktree-strategy

## Problem Solved

When TDD tasks run in parallel, commits are batched at the end, losing granular git history (red→commit→green→commit→refactor→commit). This project implemented git worktree-based isolation so parallel agents can each maintain proper TDD commit discipline.

## Solution Implemented

### New Package: `internal/worktree/`

**Manager** with methods:
- `Path(taskID)` - Returns canonical worktree path: `<repo>/../<repo>-worktrees/<task>/`
- `Create(taskID)` - Creates branch `task/<taskID>` and worktree directory
- `List()` - Returns active task worktrees (filters to `task/*` branches)
- `Merge(taskID, onto)` - Removes worktree, rebases branch onto target, ff-merges
- `Cleanup(taskID)` - Removes worktree directory and deletes branch
- `CleanupAll()` - Cleans up all task worktrees

**MergeConflictError** - Returned when rebase encounters conflicts

### CLI Commands: `projctl worktree`

```bash
projctl worktree create -t TASK-001   # Create worktree + branch
projctl worktree list                  # Show active worktrees
projctl worktree merge -t TASK-001    # Rebase + ff-merge + cleanup
projctl worktree cleanup -t TASK-001  # Remove worktree + branch
projctl worktree cleanup-all          # Remove all task worktrees
```

### State Tracking

Extended `state.toml` with `WorktreeState` struct tracking path, branch, created time, and status (active/merged/failed).

## Parallel Execution Demo

Successfully executed 4 tasks in parallel using worktrees:
1. Created worktrees for TASK-003, TASK-005, TASK-006, TASK-007
2. Spawned agents with `cd <worktree>` instructions
3. Each agent made TDD commits on their branch
4. Merged branches back with rebase + ff-merge

## Key Files

- `internal/worktree/worktree.go` - Manager implementation
- `internal/worktree/worktree_test.go` - Comprehensive tests
- `cmd/projctl/worktree.go` - CLI commands
- `internal/state/state.go` - WorktreeState struct

## Closes

- ISSUE-27: Parallel TDD agents bypass commit-per-phase discipline
- ISSUE-31: Define parallel commit strategy for task execution
- ISSUE-33: Decision needed: Should parallel tasks use separate branches?
