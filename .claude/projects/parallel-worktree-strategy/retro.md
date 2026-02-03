# Retrospective: parallel-worktree-strategy

## What Went Well

**W1: Parallel execution actually worked**
Successfully spawned 4 agents working in separate worktrees. Each made proper TDD commits on their own branches. This proves the architecture is sound.

**W2: Bootstrap approach was satisfying**
Built TASK-001, TASK-002, TASK-004 sequentially, then used the worktree infrastructure to parallelize TASK-003, TASK-005, TASK-006, TASK-007. Dogfooding FTW.

**W3: Conflict detection worked**
The MergeConflictError mechanism correctly detected when TASK-005's rebase hit a conflict with TASK-003's changes in the test file.

## What Could Be Improved

**I1: Merge as branches complete, not at the end**
Waited for all 4 agents to finish before merging. Should have merged each branch as soon as its agent completed. This would have:
- Reduced conflict complexity (later branches rebase onto merged work)
- Caught duplicate method issue earlier (TASK-006 wouldn't have added List/CleanupAll)

**I2: Agents didn't know about each other's work**
TASK-006 agent added List() and CleanupAll() methods that TASK-003 and TASK-005 had already implemented. Without visibility into the shared codebase state, agents duplicated work.

**I3: No coordination on shared files**
Multiple agents modified the same files (worktree.go, worktree_test.go) leading to merge conflicts.

## Learnings for Orchestrator Design

**L1: Merge-on-complete pattern**
When agent N completes, immediately: remove worktree → rebase → ff-merge → delete branch. Don't batch merges.

**L2: Task assignment should consider file overlap**
Tasks modifying the same files should not run in parallel, OR should be assigned to the same worktree.

**L3: Agents need base branch awareness**
Spawn agents with instruction to periodically `git fetch origin <base>` and check for upstream changes, especially before committing.

## Action Items

- [x] Update orchestrator to merge branches as agents complete → ISSUE-039
- [x] Add file-overlap detection to task scheduler → ISSUE-040
- [x] Document parallel execution best practices → ISSUE-041

## Process Observation

**Retro findings not auto-converted to issues**

ISSUE-021 mandated that retro findings become issues, but this didn't happen automatically. The orchestrator completed the project without prompting for issue creation from retro items.

**Why it didn't happen:**
1. No automated tooling exists to extract action items from retro.md
2. The issue-update phase only closes the *linked* issue, doesn't process retro
3. I (Claude) forgot to do it manually despite knowing ISSUE-021 exists
4. The skill/workflow doesn't enforce this step

**This is itself a retro finding** - the retro-to-issues pipeline remains broken despite ISSUE-021 being "closed".

**Implication:** ISSUE-021 was prematurely closed, or its "fix" was incomplete. Need either:
- Reopen ISSUE-021 with concrete implementation requirements
- Create new issue for retro-to-issues automation
