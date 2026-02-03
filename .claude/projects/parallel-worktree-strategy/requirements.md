# Requirements: parallel-worktree-strategy

## REQ-001: Worktree directory structure

When parallel tasks execute, create a canonical worktree directory structure:
- Parent: `<repo>/../<repo-name>-worktrees/`
- Per-task: `<repo>/../<repo-name>-worktrees/TASK-XXX/`

**Rationale:** Matches user's existing `git workon` pattern. Keeps worktrees outside repo but clearly associated.

---

## REQ-002: Worktree creation for parallel tasks

For each parallel task, the orchestrator must:
1. Create branch `task/TASK-XXX` from current HEAD
2. Create worktree at canonical location
3. Return worktree path to agent

**Rationale:** Each agent needs isolated workspace for TDD commits.

---

## REQ-003: Normal TDD commits in worktree

Agents working in worktrees commit normally:
- `commit-red`: Failing test commit
- `commit-green`: Passing implementation commit
- `commit-refactor`: Cleanup commit

**Rationale:** Preserves granular TDD history per task.

---

## REQ-004: Rebase before merge

After task completion, rebase task branch onto target:
```
git rebase <target-branch> task/TASK-XXX
```

**Rationale:** Creates linear history, resolves any conflicts before merge.

---

## REQ-005: Fast-forward merge

After successful rebase, fast-forward merge:
```
git checkout <target-branch>
git merge --ff-only task/TASK-XXX
```

**Rationale:** FF merge preserves the rebased commit structure cleanly.

---

## REQ-006: Worktree cleanup

After merge, clean up:
1. Delete worktree directory
2. Delete task branch (local and remote if pushed)
3. Delete parent worktree directory if empty

**Rationale:** No stale worktrees or branches left behind.

---

## REQ-007: Conflict handling

If rebase fails due to conflicts:
1. Report conflicting task to orchestrator
2. Leave worktree intact for manual resolution
3. Continue with other tasks if possible

**Rationale:** Don't lose work; allow human intervention.

---

## REQ-008: CLI commands for worktree operations

Provide `projctl worktree` commands:
- `projctl worktree create --task TASK-XXX` - Create worktree for task
- `projctl worktree list` - List active worktrees
- `projctl worktree merge --task TASK-XXX` - Rebase and merge
- `projctl worktree cleanup --task TASK-XXX` - Remove worktree and branch
- `projctl worktree cleanup-all` - Remove all task worktrees

**Rationale:** Orchestrator needs programmatic control; also useful for manual recovery.
