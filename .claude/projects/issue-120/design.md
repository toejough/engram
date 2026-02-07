# Design: Task Parallelization in projctl step next

**Issue:** ISSUE-120
**Created:** 2026-02-06
**Status:** Draft

**Traces to:** ISSUE-120, requirements.md

---

## Overview

This design implements task parallelization by moving the Looper Pattern logic into `projctl step next`. The command will detect all unblocked tasks and return execution instructions for immediate parallel execution. The orchestrator becomes a simple executor that calls `projctl step next` after each task completion and runs whatever tasks are returned.

**Key Design Principle:** Every time a task completes, the orchestrator calls `projctl step next`, which responds with an array of all currently unblocked tasks. The orchestrator immediately executes all returned tasks in parallel.

---

## Design Decisions

### DES-1: Unified Array-Based Response Format

**Traces to:** REQ-3, REQ-5

`projctl step next` will always return a JSON response with a `tasks` array containing all currently unblocked tasks. The array length naturally indicates the execution mode:

- **Empty array** (`[]`): No tasks are unblocked; orchestrator waits
- **Single-element array**: One task is unblocked; sequential execution
- **Multi-element array**: Multiple tasks are unblocked; parallel execution

**Response Schema:**

```json
{
  "tasks": [
    {
      "id": "task-1",
      "command": "projctl run task-1",
      "worktree": "/path/to/worktree-1"
    },
    {
      "id": "task-2",
      "command": "projctl run task-2",
      "worktree": "/path/to/worktree-2"
    }
  ]
}
```

**Field Definitions:**

- `tasks` (array, required): Array of all currently unblocked tasks
  - `id` (string, required): Task identifier
  - `command` (string, required): Shell command to execute for this task
  - `worktree` (string, nullable): Path to worktree for parallel execution; `null` for tasks executing on main branch

**Rationale:**

- Consistent structure regardless of execution mode
- No explicit `mode` field needed; array length is self-documenting
- Simple to parse and iterate over
- Empty array naturally represents "nothing to do" state

**Breaking Change Note:**

This is a breaking change from the current single-task return format. All orchestrators must be updated to handle the array-based response. This aligns with the user's decision on backward compatibility: "Unified format, orchestrators must update."

---

### DES-2: Worktree Path as Task Object Field

**Traces to:** REQ-2

Each task object includes a `worktree` field that specifies the path to the worktree where the task should execute. This field is `null` for tasks that execute on the main branch (typically when only one task is unblocked).

**Implementation Details:**

- **Parallel tasks** (multiple unblocked): Each task receives a unique worktree path (e.g., `/path/to/repo/.git/worktrees/task-1`)
- **Sequential task** (single unblocked): The task's `worktree` field is `null`, indicating execution on the main branch
- **Worktree naming convention:** Use task ID as worktree name for easy identification

**Example - Parallel Execution:**

```json
{
  "tasks": [
    {"id": "task-1", "command": "...", "worktree": "/repo/.git/worktrees/task-1"},
    {"id": "task-2", "command": "...", "worktree": "/repo/.git/worktrees/task-2"}
  ]
}
```

**Example - Sequential Execution:**

```json
{
  "tasks": [
    {"id": "task-3", "command": "...", "worktree": null}
  ]
}
```

**Rationale:**

- Colocates worktree path with task data for easy parsing
- Orchestrator knows exactly where to execute each command
- No separate lookup required (vs. a top-level `worktrees` map)

---

### DES-3: Status Visibility via `projctl step status`

**Traces to:** REQ-2

A new command `projctl step status` provides visibility into the current state of parallel task execution.

**Command Output:**

```json
{
  "active_tasks": [
    {
      "id": "task-1",
      "worktree": "/repo/.git/worktrees/task-1",
      "status": "running",
      "started_at": "2026-02-06T10:30:00Z"
    },
    {
      "id": "task-2",
      "worktree": "/repo/.git/worktrees/task-2",
      "status": "running",
      "started_at": "2026-02-06T10:30:05Z"
    }
  ],
  "completed_tasks": [
    {
      "id": "task-0",
      "status": "completed",
      "completed_at": "2026-02-06T10:29:58Z"
    }
  ],
  "blocked_tasks": [
    {
      "id": "task-3",
      "blocked_by": ["task-1", "task-2"]
    }
  ]
}
```

**Field Definitions:**

- `active_tasks`: Tasks currently executing in worktrees
- `completed_tasks`: Tasks that have finished and been merged
- `blocked_tasks`: Tasks waiting on dependencies

**Implementation Notes:**

- `projctl step status` reads project state to determine task states
- No persistent process tracking needed; state is inferred from project files and worktree existence
- Orchestrator can call this command to display progress to users

**Rationale:**

- Clear separation of concerns: `projctl step next` returns work; `projctl step status` reports state
- Provides visibility without mixing decision-making with status reporting
- Useful for debugging and monitoring parallel execution

---

### DES-4: Error Handling with Retry Logic

**Traces to:** REQ-2, EC-4

Error handling for parallel tasks follows the same pattern as sequential execution: retry up to 3 times, then escalate to the user.

**Error Scenarios:**

1. **Worktree Creation Fails**
   - Retry worktree creation up to 3 times
   - If still failing, escalate to user with error details

2. **Task Execution Fails**
   - Retry task execution up to 3 times (orchestrator responsibility)
   - If still failing, escalate to user with error details

3. **Worktree Merge Conflict**
   - No retry; immediately escalate to user for manual conflict resolution
   - Provide clear error message with conflicting files
   - User resolves conflict, then orchestrator retries merge

**Implementation Details:**

- Retry logic is implemented by the orchestrator, not by `projctl step next`
- `projctl step next` returns errors via standard exit codes and stderr
- On error, orchestrator:
  1. Retries the failed operation (up to 3 times)
  2. If still failing, presents error to user
  3. User can choose to: retry manually, skip task, or abort

**Error Response Format:**

When an error occurs, `projctl step next` exits with non-zero code and writes error details to stderr:

```
Error: Failed to create worktree for task-1
Details: worktree '/repo/.git/worktrees/task-1' already exists
```

**Rationale:**

- Consistent error handling between parallel and sequential execution
- Orchestrator controls retry logic for flexibility
- Clear escalation path for errors that can't be auto-resolved
- Uses standard Unix conventions (exit codes, stderr)

---

### DES-5: Worktree Lifecycle Management

**Traces to:** REQ-2

`projctl step next` does NOT directly manage worktree creation, execution, or merging. Instead, it returns instructions that the orchestrator executes. The worktree lifecycle is managed through orchestrator actions:

**Lifecycle Phases:**

1. **Detection Phase** (handled by `projctl step next`):
   - Identify all unblocked tasks
   - Determine if parallel execution is needed (multiple tasks)
   - Return task list with worktree paths

2. **Creation Phase** (handled by orchestrator):
   - For each task with a non-null `worktree` field:
   - Execute: `git worktree add <worktree-path> <base-branch>`
   - Verify worktree creation succeeded

3. **Execution Phase** (handled by orchestrator):
   - For each task:
     - `cd <worktree-path>` (if worktree is non-null)
     - Execute the task's `command`
     - Capture exit code and output

4. **Merge Phase** (handled by orchestrator):
   - After task completes successfully:
     - Switch to main branch
     - Execute: `git merge <worktree-branch>` or `git rebase <worktree-branch>`
     - Handle conflicts if they arise (see DES-4)

5. **Cleanup Phase** (handled by orchestrator):
   - After successful merge:
     - Execute: `git worktree remove <worktree-path>`
     - Verify cleanup succeeded

**Example Orchestrator Flow:**

```
1. Call: projctl step next
2. Receive: {"tasks": [{"id": "A", "worktree": "/wt/A"}, {"id": "B", "worktree": "/wt/B"}]}
3. Create worktrees: git worktree add /wt/A main && git worktree add /wt/B main
4. Execute in parallel:
   - (cd /wt/A && <task A command>)
   - (cd /wt/B && <task B command>)
5. Wait for both to complete
6. Merge: git merge A-branch && git merge B-branch
7. Cleanup: git worktree remove /wt/A && git worktree remove /wt/B
8. Call: projctl step next (repeat)
```

**Rationale:**

- `projctl step next` focuses on task detection and decision-making
- Orchestrator handles the mechanical work of worktree operations
- Clean separation of concerns: decision vs. execution
- Orchestrator can optimize (e.g., parallel worktree creation)

---

### DES-6: Immediate Execution Model

**Traces to:** REQ-1, REQ-5, EC-3

Tasks execute immediately when they become unblocked. There is no batching, queuing, or delayed execution.

**Execution Flow:**

```
1. Orchestrator calls `projctl step next`
2. `projctl step next` returns all currently unblocked tasks (0, 1, or N tasks)
3. Orchestrator immediately executes all returned tasks
4. When any task completes, orchestrator calls `projctl step next` again
5. Repeat until no tasks remain
```

**Key Properties:**

- **No artificial delays**: As soon as a task is unblocked, it's returned and executed
- **Dynamic work discovery**: Completing task A might unblock tasks B and C, which are then immediately executed in parallel
- **No batch accumulation**: Don't wait for "a few more tasks" to accumulate before returning results
- **Call frequency**: Orchestrator calls `projctl step next` after every task completion (not just after all parallel tasks complete)

**Example Scenario:**

```
State: Tasks A, B unblocked; C, D blocked by A; E blocked by B

Call 1: projctl step next → returns [A, B]
Execute A and B in parallel

A completes:
Call 2: projctl step next → returns [C, D] (newly unblocked)
Execute C and D in parallel (B still running)

B completes:
Call 3: projctl step next → returns [E] (newly unblocked)
Execute E (C and D still running)

C, D, E complete:
Call 4: projctl step next → returns [] (all done)
```

**Rationale:**

- Maximizes parallelism by starting work as soon as possible
- Avoids orchestrator needing to track "when to check for new work"
- Simple rule: always call `projctl step next` after any task completes
- No complex scheduling logic needed

---

### DES-7: Removal of File Overlap Detection

**Traces to:** REQ-4

All file overlap detection code will be removed:

**Code Removal:**

- `projctl tasks overlap` command and all related code
- All tests for overlap detection functionality
- Any internal functions that analyze file overlap between tasks

**Rationale:**

Git's conflict detection during merge/rebase is the authoritative source for conflicts. Pre-execution file overlap analysis:
- Creates false negatives (tasks that appear to conflict but don't)
- Creates false positives (tasks that share files but don't conflict)
- Adds complexity without providing value
- Conflicts are naturally discovered during merge (DES-5, phase 4)

**Conflict Resolution:**

When conflicts occur during merge:
1. Git reports conflict during `git merge <worktree-branch>`
2. Orchestrator surfaces conflict to user (DES-4)
3. User resolves conflict using standard git workflow
4. Orchestrator retries merge after resolution

**Migration Note:**

Any existing code or documentation that references `projctl tasks overlap` must be updated or removed.

---

## Implementation Plan

### Phase 1: Response Format Changes

**Goal:** Update `projctl step next` to return array-based response format.

**Tasks:**

1. Modify `projctl step next` output to return `{"tasks": [...]}`
2. Update task detection logic to identify ALL unblocked tasks (not just one)
3. Add tests for array responses (0, 1, N tasks)
4. Update documentation for new response format

**Acceptance:**

- `projctl step next` returns JSON array with all unblocked tasks
- Tests verify empty array, single-task array, multi-task array cases

**Traces to:** DES-1, REQ-3

---

### Phase 2: Worktree Path Assignment

**Goal:** Assign worktree paths to tasks in response.

**Tasks:**

1. Add `worktree` field to task objects in response
2. Implement worktree path generation (use task ID as worktree name)
3. Set `worktree` to `null` for single-task case (sequential execution)
4. Add tests for worktree path assignment

**Acceptance:**

- Parallel tasks receive unique worktree paths
- Sequential tasks receive `worktree: null`
- Paths follow naming convention: `.git/worktrees/<task-id>`

**Traces to:** DES-2, REQ-2

---

### Phase 3: Status Command Implementation

**Goal:** Implement `projctl step status` command.

**Tasks:**

1. Create `projctl step status` command skeleton
2. Implement state detection (active, completed, blocked tasks)
3. Format JSON output with task states
4. Add tests for status command with various project states

**Acceptance:**

- `projctl step status` returns JSON with active/completed/blocked tasks
- Status accurately reflects current project state
- Tests cover empty state, active tasks, completed tasks, blocked tasks

**Traces to:** DES-3

---

### Phase 4: Remove Overlap Detection

**Goal:** Remove all file overlap detection code.

**Tasks:**

1. Delete `projctl tasks overlap` command
2. Remove all overlap detection tests
3. Remove any internal overlap analysis functions
4. Update documentation to remove overlap references

**Acceptance:**

- `projctl tasks overlap` returns error (command not found)
- No overlap-related code remains in codebase
- All overlap-related tests deleted
- Documentation updated

**Traces to:** DES-7, REQ-4

---

### Phase 5: Integration Testing

**Goal:** Verify end-to-end parallel execution flow.

**Tasks:**

1. Create integration test with multiple unblocked tasks
2. Verify immediate execution model (no batching delays)
3. Test task completion unblocking new tasks (EC-3)
4. Test error handling with retries
5. Test merge conflict scenario

**Acceptance:**

- Integration test verifies full parallel execution lifecycle
- All edge cases (EC-1, EC-2, EC-3, EC-4) covered
- Error handling matches DES-4 specification

**Traces to:** DES-4, DES-5, DES-6, REQ-1, REQ-2, REQ-5

---

## API Specification

### Command: `projctl step next`

**Description:** Returns all currently unblocked tasks for immediate execution.

**Usage:**

```bash
projctl step next
```

**Output Format:**

```json
{
  "tasks": [
    {
      "id": "<task-id>",
      "command": "<shell-command>",
      "worktree": "<worktree-path-or-null>"
    }
  ]
}
```

**Exit Codes:**

- `0`: Success, tasks returned (array may be empty)
- `1`: Error occurred (details on stderr)

**Examples:**

```bash
# Multiple unblocked tasks (parallel execution)
$ projctl step next
{
  "tasks": [
    {"id": "task-1", "command": "projctl run task-1", "worktree": "/repo/.git/worktrees/task-1"},
    {"id": "task-2", "command": "projctl run task-2", "worktree": "/repo/.git/worktrees/task-2"}
  ]
}

# Single unblocked task (sequential execution)
$ projctl step next
{
  "tasks": [
    {"id": "task-3", "command": "projctl run task-3", "worktree": null}
  ]
}

# No unblocked tasks
$ projctl step next
{
  "tasks": []
}
```

**Traces to:** DES-1, DES-2, REQ-3

---

### Command: `projctl step status`

**Description:** Returns current state of all tasks (active, completed, blocked).

**Usage:**

```bash
projctl step status
```

**Output Format:**

```json
{
  "active_tasks": [
    {
      "id": "<task-id>",
      "worktree": "<worktree-path>",
      "status": "running",
      "started_at": "<iso-8601-timestamp>"
    }
  ],
  "completed_tasks": [
    {
      "id": "<task-id>",
      "status": "completed",
      "completed_at": "<iso-8601-timestamp>"
    }
  ],
  "blocked_tasks": [
    {
      "id": "<task-id>",
      "blocked_by": ["<task-id>", ...]
    }
  ]
}
```

**Exit Codes:**

- `0`: Success
- `1`: Error occurred (details on stderr)

**Example:**

```bash
$ projctl step status
{
  "active_tasks": [
    {"id": "task-1", "worktree": "/repo/.git/worktrees/task-1", "status": "running", "started_at": "2026-02-06T10:30:00Z"}
  ],
  "completed_tasks": [
    {"id": "task-0", "status": "completed", "completed_at": "2026-02-06T10:29:58Z"}
  ],
  "blocked_tasks": [
    {"id": "task-2", "blocked_by": ["task-1"]}
  ]
}
```

**Traces to:** DES-3

---

## Edge Cases

### Edge Case 1: All Tasks Blocked (EC-1)

**Scenario:** All remaining tasks are blocked by dependencies.

**Design Response:**

`projctl step next` returns empty array:

```json
{
  "tasks": []
}
```

Orchestrator waits for currently executing tasks to complete, then calls `projctl step next` again.

**Traces to:** EC-1, DES-1

---

### Edge Case 2: Single Task Unblocked (EC-2)

**Scenario:** Only one task is currently unblocked.

**Design Response:**

`projctl step next` returns single-element array with `worktree: null`:

```json
{
  "tasks": [
    {"id": "task-1", "command": "projctl run task-1", "worktree": null}
  ]
}
```

Orchestrator executes on main branch (no worktree creation needed).

**Traces to:** EC-2, DES-2

---

### Edge Case 3: Task Completion Unblocks New Tasks (EC-3)

**Scenario:** Completing task A unblocks tasks B and C.

**Design Response:**

After task A completes, orchestrator calls `projctl step next`:

```json
{
  "tasks": [
    {"id": "B", "command": "...", "worktree": "/repo/.git/worktrees/B"},
    {"id": "C", "command": "...", "worktree": "/repo/.git/worktrees/C"}
  ]
}
```

Orchestrator immediately spawns parallel execution of B and C.

**Traces to:** EC-3, DES-6

---

### Edge Case 4: Worktree Merge Conflicts (EC-4)

**Scenario:** Two parallel tasks modify the same file, causing merge conflict during worktree merge.

**Design Response:**

Git reports merge conflict during merge phase (DES-5, phase 4):

```
CONFLICT (content): Merge conflict in file.go
Automatic merge failed; fix conflicts and then commit the result.
```

Orchestrator:
1. Does not retry (merge conflicts require manual resolution)
2. Escalates to user with conflict details
3. User resolves conflict using standard git workflow
4. Orchestrator retries merge after resolution

**Traces to:** EC-4, DES-4

---

## Testing Strategy

### Unit Tests

**DES-1 Tests:**
- Test empty array response (no unblocked tasks)
- Test single-element array response (one unblocked task)
- Test multi-element array response (multiple unblocked tasks)

**DES-2 Tests:**
- Test worktree path assignment for parallel tasks
- Test `worktree: null` for sequential tasks
- Test worktree naming convention (task ID)

**DES-3 Tests:**
- Test `projctl step status` with no active tasks
- Test `projctl step status` with active tasks
- Test `projctl step status` with blocked tasks
- Test `projctl step status` with completed tasks

**DES-7 Tests:**
- Verify `projctl tasks overlap` returns error
- Verify no overlap detection code in codebase (static analysis)

---

### Integration Tests

**Parallel Execution Test (REQ-1, REQ-5):**

1. Create project with 3 unblocked independent tasks
2. Call `projctl step next`
3. Verify response contains all 3 tasks
4. Verify each task has unique worktree path
5. Verify tasks execute immediately (no delays)

**Worktree Lifecycle Test (REQ-2):**

1. Call `projctl step next` → receive 2 tasks with worktree paths
2. Create worktrees using returned paths
3. Execute tasks in worktrees
4. Merge worktree branches back to main
5. Cleanup worktrees
6. Verify main branch contains both task results

**Immediate Execution Test (EC-3):**

1. Create project: Task A unblocked, B and C blocked by A
2. Call `projctl step next` → receive [A]
3. Execute A
4. Call `projctl step next` → receive [B, C]
5. Verify B and C are returned immediately after A completes

**Error Handling Test (DES-4):**

1. Create scenario where worktree creation fails
2. Verify retry logic executes up to 3 times
3. Verify escalation to user after 3 failures

**Merge Conflict Test (EC-4):**

1. Create 2 tasks that modify the same file
2. Execute tasks in parallel
3. Merge first task successfully
4. Attempt merge of second task → conflict
5. Verify orchestrator escalates to user (no retry)

---

## Traceability

**Upstream:**
- ISSUE-120
- requirements.md (REQ-1, REQ-2, REQ-3, REQ-4, REQ-5)
- Edge cases (EC-1, EC-2, EC-3, EC-4)

**Downstream:**
- architecture.md (will reference DES-1 through DES-7)
- Implementation code
- Test code

---

## Open Questions

None. All design questions have been resolved through user input.

---

## Revision History

| Date | Version | Changes |
|------|---------|---------|
| 2026-02-06 | 1.0 | Initial design based on requirements and user input |
