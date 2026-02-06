---
name: parallel-looper
description: "[DEPRECATED] Runs N PAIR LOOPs in parallel for independent items"
context: fork
model: sonnet
skills: ownership-rules
user-invocable: false
role: standalone
deprecated: true
---

# Parallel Looper

> **DEPRECATED (ISSUE-83):** This skill is deprecated and replaced by native Claude Code team parallelism (ISSUE-79). The orchestrator now spawns concurrent teammates directly for independent tasks, using git worktrees for isolation and TaskList for coordination. Do not use this skill for new work. It is retained temporarily for rollback purposes.

Orchestrates parallel execution of PAIR LOOPs for independent items.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | List of independent items from LOOPER context |
| Pattern | SPAWN -> WAIT -> AGGREGATE -> VALIDATE |
| Output | Aggregated results yield or partial failure report |

---

## Input Context

Receives context from LOOPER with independent items:

```toml
[invocation]
skill = "parallel-looper"

[inputs]
# Items verified as independent by LOOPER
[[inputs.items]]
id = "TASK-5"
type = "task"
producer = "tdd-producer"
qa = "qa"

[[inputs.items]]
id = "TASK-6"
type = "task"
producer = "tdd-producer"
qa = "qa"

[[inputs.items]]
id = "TASK-7"
type = "task"
producer = "tdd-producer"
qa = "qa"

[inputs.consistency]
domain = "tdd"
rules = ["no shared file conflicts", "test isolation"]

[output]
```

---

## Workflow

### 1. SPAWN Phase

Launch PAIR LOOP for each item via Task tool in parallel:

```
FOR EACH item IN inputs.items (in parallel via Task tool):
    1. Build context for item's PAIR LOOP
    2. Invoke: Task(pair-loop, context={item, producer, qa})
    3. Track Task handle
```

**Parallel Execution:**
- Use Task tool to spawn all PAIR LOOPs simultaneously
- Each PAIR LOOP runs independently (producer + QA iterations)
- Do not wait between spawns

### 2. WAIT Phase

Collect yields from all spawned PAIR LOOPs:

```
WAIT for all Task handles to complete
COLLECT yields from each PAIR LOOP:
    - Success yields (approved results)
    - Failure yields (blocked, error, or max iterations)
```

### 3. AGGREGATE Phase

Combine results from all parallel executions:

```
results = {
    succeeded: [],   # Items that passed PAIR LOOP QA
    failed: [],      # Items that failed (blocked/error/iterations)
    artifacts: [],   # All artifacts produced
    decisions: [],   # All decisions made
    learnings: []    # All learnings captured
}

FOR EACH yield IN collected_yields:
    IF yield.type == "approved":
        results.succeeded.append(yield)
        results.artifacts.extend(yield.payload.files_modified)
    ELSE:
        results.failed.append(yield)
```

### 4. VALIDATE Phase

Dispatch to consistency-checker for batch QA:

```
IF results.succeeded.length > 0:
    1. Build batch context with all successful results
    2. Invoke: consistency-checker with domain rules
    3. Handle consistency-checker yield:
       - approved: proceed to complete yield
       - improvement-request: return items for rework
```

---

## Handling Partial Failures

When some items fail while others succeed:

```toml
[yield]
type = "partial-complete"
timestamp = 2026-02-02T12:00:00Z

[payload]
# Successfully completed items
[[payload.succeeded]]
id = "TASK-5"
artifact = "internal/foo/foo.go"
tests_passed = true

[[payload.succeeded]]
id = "TASK-6"
artifact = "internal/bar/bar.go"
tests_passed = true

# Failed items with reasons
[[payload.failed]]
id = "TASK-7"
reason = "blocked"
details = "Missing dependency: TASK-8 must complete first"

[[payload.failed]]
id = "TASK-8"
reason = "error"
details = "Test framework not installed"
retry_count = 3

[payload.consistency]
checker_verdict = "approved"
remediations_applied = []

[context]
total_items = 4
succeeded_count = 2
failed_count = 2
```

---

## Yield Types

### Complete (all succeed)

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T12:00:00Z

[payload]
items_processed = 3
all_succeeded = true
artifacts = ["internal/foo/foo.go", "internal/bar/bar.go", "internal/baz/baz.go"]
files_modified = ["internal/foo/foo.go", "internal/foo/foo_test.go", ...]

[payload.consistency]
checker_verdict = "approved"
remediations_applied = []

[[payload.item_results]]
id = "TASK-5"
status = "approved"
artifact = "internal/foo/foo.go"

[[payload.item_results]]
id = "TASK-6"
status = "approved"
artifact = "internal/bar/bar.go"

[context]
phase = "parallel-execution"
subphase = "complete"
```

### Partial Complete (some fail)

```toml
[yield]
type = "partial-complete"
timestamp = 2026-02-02T12:00:00Z

[payload]
items_processed = 4
succeeded_count = 2
failed_count = 2

[[payload.succeeded]]
id = "TASK-5"
artifact = "internal/foo/foo.go"

[[payload.failed]]
id = "TASK-7"
reason = "blocked"
details = "Dependency not met"

[context]
phase = "parallel-execution"
subphase = "partial"
```

### Consistency Failed

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:00:00Z

[payload]
from_agent = "consistency-checker"
to_agent = "parallel-looper"
reason = "Conflicting implementations detected"

[[payload.issues]]
items = ["TASK-5", "TASK-6"]
conflict = "Both modify shared state in config.go"
resolution = "Coordinate state access pattern"

[context]
phase = "parallel-execution"
subphase = "consistency-failed"
```

### Error

```toml
[yield]
type = "error"
timestamp = 2026-02-02T12:00:00Z

[payload]
error = "All parallel executions failed"
details = "No items completed successfully"
recoverable = false

[[payload.failures]]
id = "TASK-5"
reason = "timeout"

[[payload.failures]]
id = "TASK-6"
reason = "blocked"

[context]
phase = "parallel-execution"
subphase = "all-failed"
```

---

## Task Tool Usage

Spawn PAIR LOOPs in parallel using Task tool:

```markdown
Spawn parallel PAIR LOOPs:

Task 1: Execute PAIR LOOP for TASK-5
- Producer: tdd-producer
- QA: qa
- Context: {task: TASK-5, acceptance_criteria: [...]}

Task 2: Execute PAIR LOOP for TASK-6
- Producer: tdd-producer
- QA: qa
- Context: {task: TASK-6, acceptance_criteria: [...]}

Task 3: Execute PAIR LOOP for TASK-7
- Producer: tdd-producer
- QA: qa
- Context: {task: TASK-7, acceptance_criteria: [...]}
```

Each Task invocation runs independently. The parallel-looper collects all results before proceeding to consistency check.

---

## Consistency Checker Dispatch

After aggregation, dispatch to consistency-checker:

```toml
# Context for consistency-checker
[invocation]
skill = "consistency-checker"

[inputs]
domain = "tdd"

[[inputs.results]]
id = "TASK-5"
artifact = "internal/foo/foo.go"
files_modified = ["internal/foo/foo.go", "internal/foo/foo_test.go"]

[[inputs.results]]
id = "TASK-6"
artifact = "internal/bar/bar.go"
files_modified = ["internal/bar/bar.go", "internal/bar/bar_test.go"]

[[inputs.rules]]
name = "no-shared-file-conflicts"
description = "No two items should modify the same file"

[[inputs.rules]]
name = "test-isolation"
description = "Tests should not depend on execution order"

[output]
```

---

## Git Worktrees for Isolation

When executing tasks in parallel, each task runs in an isolated git worktree to prevent file conflicts.

### Worktree Lifecycle

```
FOR EACH item IN inputs.items (in parallel):
    1. CREATE: projctl worktree create --taskid <item.id>
       - Creates branch and isolated worktree directory
       - Agent works in .worktrees/<item.id>/

    2. WORK: Execute PAIR LOOP in worktree
       - All file changes isolated
       - Normal commits on task branch

    3. MERGE (on completion): projctl worktree merge --taskid <item.id>
       - Rebase onto target branch
       - Fast-forward merge
       - Remove worktree and branch
```

### Merge-on-Complete Pattern

Merge each task immediately when it completes - don't wait for all parallel tasks:

| Pattern | Behavior | Result |
|---------|----------|--------|
| Batch merge (old) | Wait for all, merge all at end | More conflicts |
| Merge-on-complete | Merge each as it completes | Fewer conflicts |

Benefits:
- Later-completing agents rebase onto already-merged work
- Reduces conflict window
- Simplifies conflict resolution
- No batch of N-way merges at the end

### Conflict Handling

| Situation | Action |
|-----------|--------|
| Rebase conflict | Pause orchestration, prompt user to resolve |
| Agent failure | Don't merge branch, cleanup worktree, log failure |
| Cleanup failure | Log error, continue, report at end |
| Simultaneous completions | Serialize by completion timestamp |

### Commands Reference

```bash
# Create worktree for task
projctl worktree create --taskid TASK-001

# List active worktrees
projctl worktree list

# Merge completed task (auto-cleans up)
projctl worktree merge --taskid TASK-001 [--onto main]

# Manual cleanup single worktree
projctl worktree cleanup --taskid TASK-001

# Cleanup all worktrees
projctl worktree cleanup-all
```

---

## Summary

| Yield Type | When |
|------------|------|
| `complete` | All items succeeded, consistency passed |
| `partial-complete` | Some items succeeded, some failed |
| `improvement-request` | Consistency check failed, need rework |
| `error` | All items failed or critical failure |
