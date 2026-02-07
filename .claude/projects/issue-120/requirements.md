# Requirements: Task Parallelization in projctl step next

**Issue:** ISSUE-120
**Created:** 2026-02-06
**Status:** Draft

**Traces to:** ISSUE-120

---

## Overview

Currently, the orchestrator manually implements the Looper Pattern: checking for unblocked tasks, spawning parallel agents with worktrees, and managing the worktree lifecycle. This results in sequential execution because `projctl step next` only returns one action at a time.

This issue moves task parallelization logic into `projctl step next` itself, enabling it to detect independent tasks and manage worktree lifecycle automatically. The orchestrator will receive parallel execution instructions directly from `projctl step next` and execute them immediately.

**Core principle:** `projctl step next` orchestrates parallelization; the orchestrator executes instructions.

---

## Requirements

### REQ-1: Parallel Task Detection

As a project orchestrator, I want `projctl step next` to detect when multiple tasks are unblocked and independent, so that I can execute them in parallel without manually implementing the Looper Pattern.

**Acceptance Criteria:**
- [ ] `projctl step next` identifies all currently unblocked tasks
- [ ] When multiple unblocked tasks exist, returns parallel execution instructions
- [ ] Each task is executed immediately when unblocked (no batching delays)
- [ ] When only one task is unblocked, returns single task execution (no format change required)

**Priority:** P0

**Traces to:** ISSUE-120

---

### REQ-2: Worktree Lifecycle Management

As a project orchestrator, I want `projctl step next` to manage the complete worktree lifecycle (create, execute, merge), so that I don't need to manually orchestrate worktree operations.

**Acceptance Criteria:**
- [ ] `projctl step next` creates worktrees for parallel task execution
- [ ] Returns execution instructions that include worktree paths
- [ ] Manages worktree cleanup after task completion
- [ ] Handles worktree merge back to main branch
- [ ] Orchestrator only executes commands; does not manage worktree lifecycle

**Priority:** P0

**Traces to:** ISSUE-120

---

### REQ-3: JSON Output Format for Parallel Actions

As a project orchestrator, I want a clean JSON format for parallel task actions, so that I can easily parse and execute parallel work.

**Acceptance Criteria:**
- [ ] `projctl step next` returns JSON with clear parallel/sequential indication
- [ ] Format supports multiple parallel tasks in a single response
- [ ] Each task includes: task ID, worktree path, execution command
- [ ] Format is backward compatible (single task case still works)

**Priority:** P0

**Traces to:** ISSUE-120

**Design Note:** JSON structure design has discretion for optimal clarity and usability. Example structure:

```json
{
  "mode": "parallel",
  "tasks": [
    {
      "id": "task-1",
      "worktree": "/path/to/worktree-1",
      "command": "..."
    },
    {
      "id": "task-2",
      "worktree": "/path/to/worktree-2",
      "command": "..."
    }
  ]
}
```

or

```json
{
  "mode": "sequential",
  "task": {
    "id": "task-1",
    "command": "..."
  }
}
```

Final structure determined during design phase.

---

### REQ-4: Remove File Overlap Detection

As a project maintainer, I want to remove `projctl tasks overlap` and related code, because conflict detection is handled by git rebase and conflict resolution, not pre-execution analysis.

**Acceptance Criteria:**
- [ ] `projctl tasks overlap` command removed entirely
- [ ] All tests for `projctl tasks overlap` removed
- [ ] No file overlap detection logic in `projctl step next`
- [ ] Conflicts are handled via standard git rebase/merge process
- [ ] Documentation updated to reflect removal

**Priority:** P0

**Traces to:** ISSUE-120

**Rationale:** File overlap detection adds complexity without value. Git's conflict resolution during rebase already handles this case. Pre-execution analysis creates false negatives (tasks that appear to conflict but don't) and false positives (tasks that don't conflict but share files).

---

### REQ-5: Immediate Execution Model

As a project orchestrator, I want tasks to execute immediately when they become unblocked, so that work progresses as fast as possible without artificial batching delays.

**Acceptance Criteria:**
- [ ] `projctl step next` returns execution instructions for all currently unblocked tasks
- [ ] No waiting for "batches" to accumulate
- [ ] As soon as a task is unblocked (dependencies complete), it's returned for execution
- [ ] Orchestrator executes returned tasks immediately

**Priority:** P0

**Traces to:** ISSUE-120

---

## Edge Cases

### EC-1: All Tasks Blocked

**Scenario:** All remaining tasks are blocked by dependencies.

**Behavior:**
- `projctl step next` returns no executable tasks
- Orchestrator waits for current tasks to complete
- After completion, calls `projctl step next` again to get newly unblocked tasks

**Priority:** P0

---

### EC-2: Single Task Unblocked

**Scenario:** Only one task is currently unblocked.

**Behavior:**
- `projctl step next` returns single task execution instruction
- Format may differ from parallel case (design discretion)
- No worktree needed for single sequential task (uses main branch)
- Backward compatible with current orchestrator behavior

**Priority:** P0

---

### EC-3: Task Completion Unblocks New Tasks

**Scenario:** Completing task A unblocks tasks B and C.

**Behavior:**
- Orchestrator calls `projctl step next` after task A completes
- `projctl step next` detects B and C are now unblocked
- Returns parallel execution instructions for B and C
- Orchestrator spawns parallel work immediately

**Priority:** P0

---

### EC-4: Worktree Merge Conflicts

**Scenario:** Two parallel tasks modify the same file, causing merge conflict during worktree merge.

**Behavior:**
- Git reports merge conflict during rebase/merge
- Orchestrator surfaces conflict to user or conflict resolution process
- No pre-execution detection needed (REQ-4)
- Standard git conflict resolution workflow applies

**Priority:** P1

**Notes:** This validates REQ-4 - conflicts are handled by git, not by pre-execution file overlap analysis.

---

## Out of Scope

The following are explicitly out of scope for this issue:

1. **Smart scheduling algorithms** - No complex task scheduling heuristics. Simple rule: execute all unblocked tasks immediately.
2. **Resource limits** - No limits on parallel task count. Orchestrator can add limits externally if needed.
3. **Task priority** - All unblocked tasks have equal priority. No prioritization logic in `projctl step next`.
4. **Conflict prediction** - No pre-execution analysis of potential conflicts (REQ-4 explicitly removes this).
5. **Orchestrator changes** - This issue focuses on `projctl step next` behavior. Orchestrator changes to consume new format are separate work.

---

## Success Criteria

This issue is resolved when:

1. **REQ-1 verification:** Test case with multiple unblocked tasks shows `projctl step next` returns parallel execution instructions
2. **REQ-2 verification:** Test case shows worktrees are created, used, and merged automatically by `projctl step next` commands
3. **REQ-3 verification:** JSON output parses correctly and contains all required fields for parallel execution
4. **REQ-4 verification:** `projctl tasks overlap` command returns error (removed), and all related tests are deleted
5. **REQ-5 verification:** Test case shows tasks execute immediately when unblocked, not waiting for batches

**Test descriptions:**
- **Test 1 (REQ-1, REQ-5):** Create project state with 3 unblocked independent tasks, verify `projctl step next` returns all 3 for immediate execution
- **Test 2 (REQ-2):** Verify worktree creation, task execution in worktree, and merge back to main branch all handled by `projctl step next` lifecycle
- **Test 3 (REQ-3):** Parse JSON output from `projctl step next`, verify all required fields present and correctly formatted
- **Test 4 (REQ-4):** Run `projctl tasks overlap` and verify it returns "command not found" or similar error; verify no overlap-related tests exist
- **Test 5 (EC-1, EC-2, EC-3):** Test edge cases for blocked tasks, single task, and task completion unlocking new tasks

---

## Traceability

**Upstream:** ISSUE-120
**Downstream:** (Design and Architecture phases will reference these REQ-IDs)
