# Context Commands

The `projctl context` commands manage skill dispatch context files and result collection.

## Commands

| Command | Purpose |
|---------|---------|
| `projctl context write` | Write context file for skill dispatch |
| `projctl context read` | Read context or result file |
| `projctl context write-parallel` | Create context files for multiple tasks |
| `projctl context check` | Check context budget against thresholds |

---

## context write

Creates a context file for skill dispatch. Copies a source TOML file into the context directory with routing information and optional memory injection.

### Usage

```bash
projctl context write -d <project-dir> -t <task-id> -s <skill> -f <source-file> [options]
```

### Flags

| Flag | Description |
|------|-------------|
| `-d, --dir` | Project directory (required) |
| `-t, --task` | Task ID, e.g., TASK-004 (required) |
| `-s, --skill` | Skill name, e.g., tdd-red (required) |
| `-f, --file` | Path to source TOML file (required) |
| `--no-routing` | Skip adding routing section |
| `--inject-memory` | Query memory and inject top results |

### Output

```
Context written: <path-to-context-file>
```

---

## The output.result_path Field

When skills are invoked through the orchestration system, context files include an `output.result_path` field. This field specifies an absolute path where the skill must write its result.

### Purpose

The result_path pattern enables:

1. **Parallel execution**: Multiple skill invocations can run simultaneously without file conflicts
2. **Unique results**: Each invocation writes to a distinct location via UUID
3. **Orchestrator collection**: The orchestrator knows exactly where to find each result

### Context File Structure

A context file with result_path looks like:

```toml
[dispatch]
skill = "tdd-red"

[task]
id = "TASK-004"
description = "Implement user authentication"

[territory]
root = "/project"
languages = ["go"]

[output]
result_path = "/project/.claude/context/2026-02-04-myproject-abc123/2026-02-04.14-30-45-impl-TASK-004-def456.toml"
```

The `output.result_path` field is always an absolute path.

### Path Format

Result paths follow one of two patterns depending on execution mode:

**Parallel execution (with task ID):**

```
.claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{taskID}-{fileUUID}.toml
```

Example:
```
.claude/context/2026-02-04-myproject-abc123def4/2026-02-04.14-30-45-impl-TASK-004-789abc12.toml
```

**Sequential execution (no task ID):**

```
.claude/context/{date}-{project}-{projectUUID}/{datetime}-{phase}-{fileUUID}.toml
```

Example:
```
.claude/context/2026-02-04-myproject-abc123def4/2026-02-04.14-30-45-pm-789abc12.toml
```

### Path Components

| Component | Format | Description |
|-----------|--------|-------------|
| `date` | YYYY-MM-DD | Date of context creation |
| `project` | string | Project name from state.toml |
| `projectUUID` | UUID | Stable project identifier |
| `datetime` | YYYY-MM-DD.HH-mm-SS | Timestamp of invocation |
| `phase` | string | Workflow phase (pm, design, impl, etc.) |
| `taskID` | TASK-NNN | Task identifier (parallel only) |
| `fileUUID` | UUID | Unique per-invocation identifier |

### Uniqueness Guarantees

The fileUUID component ensures uniqueness even when:

- Same task is invoked multiple times (retries)
- Multiple tasks run in parallel with same timestamp
- Multiple invocations occur within the same second

This is critical for parallel execution where race conditions would otherwise cause file conflicts.

---

## Skill Usage Pattern

Skills follow this pattern when using result_path:

### 1. Read Context and Extract result_path

```go
// Parse context file
type OutputSection struct {
    ResultPath string `toml:"result_path"`
}
type ContextFile struct {
    Dispatch DispatchSection `toml:"dispatch"`
    Task     TaskSection     `toml:"task"`
    Output   OutputSection   `toml:"output"`
}

var ctx ContextFile
_, err := toml.DecodeFile(contextPath, &ctx)
if err != nil {
    return err
}

resultPath := ctx.Output.ResultPath
```

### 2. Perform Skill Work

The skill does its work (writes tests, produces designs, etc.).

### 3. Write Result to result_path

```go
result := `[result]
status = "success"
phase = "impl"
subphase = "tdd-red"

[payload]
summary = "Tests written for user authentication"
tests_created = 5

[[payload.decisions]]
context = "Test organization"
choice = "Table-driven tests"
reason = "Cleaner test structure"
`

err := os.WriteFile(resultPath, []byte(result), 0o644)
```

### 4. Orchestrator Reads Result

The orchestrator reads from the known result_path location:

```go
resultData, err := os.ReadFile(resultPath)
if err != nil {
    return fmt.Errorf("skill did not produce result at %s: %w", resultPath, err)
}
```

---

## Sequential vs. Parallel Execution

### Sequential Execution

In sequential mode, phases run one after another. The result_path omits taskID:

```
.claude/context/2026-02-04-myproject-uuid/2026-02-04.10-00-00-pm-fileUUID.toml
.claude/context/2026-02-04-myproject-uuid/2026-02-04.10-05-00-design-fileUUID.toml
.claude/context/2026-02-04-myproject-uuid/2026-02-04.10-10-00-architect-fileUUID.toml
```

### Parallel Execution

In parallel mode, multiple tasks run simultaneously. The result_path includes taskID:

```
.claude/context/2026-02-04-myproject-uuid/2026-02-04.10-00-00-impl-TASK-001-uuid1.toml
.claude/context/2026-02-04-myproject-uuid/2026-02-04.10-00-00-impl-TASK-002-uuid2.toml
.claude/context/2026-02-04-myproject-uuid/2026-02-04.10-00-00-impl-TASK-003-uuid3.toml
```

Even with identical timestamps, the fileUUID ensures uniqueness.

### Why Both Patterns?

| Pattern | Use Case | Benefits |
|---------|----------|----------|
| Sequential | Phase transitions (PM -> Design -> Architect) | Simpler paths, clear phase ordering |
| Parallel | Multiple independent tasks | Concurrent execution, no conflicts |

---

## Result File Format

Skills write results to result_path as TOML files:

```toml
[result]
status = "success"       # success | failure | escalation
phase = "impl"           # Workflow phase
subphase = "tdd-red"     # Skill within phase

[payload]
summary = "Completed task description"
tests_created = 3        # Domain-specific fields

[[payload.decisions]]
context = "What was being decided"
choice = "What was chosen"
reason = "Why it was chosen"
alternatives = ["other", "options"]
```

The orchestrator parses this result to:
- Determine next steps based on status
- Aggregate decisions for documentation
- Track progress through workflow phases

---

## context read

Reads a context or result file for a given task and skill.

### Usage

```bash
projctl context read -d <project-dir> -t <task-id> -s <skill> [-r] [--format <format>]
```

### Flags

| Flag | Description |
|------|-------------|
| `-d, --dir` | Project directory (required) |
| `-t, --task` | Task ID (required) |
| `-s, --skill` | Skill name (required) |
| `-r, --result` | Read result file instead of context file |
| `--format` | Output format: toml (default) or json |

---

## context write-parallel

Creates context files for multiple tasks using a shared template.

### Usage

```bash
projctl context write-parallel -d <project-dir> -t <task-ids> -f <template> [-s <skill>]
```

### Flags

| Flag | Description |
|------|-------------|
| `-d, --dir` | Project directory (required) |
| `-t, --tasks` | Comma-separated task IDs (required) |
| `-f, --template` | Path to template TOML file (required) |
| `-s, --skill` | Skill name (default: tdd-red) |

### Example

```bash
projctl context write-parallel -d /project -t "TASK-001,TASK-002,TASK-003" -f template.toml
```

Output:
```
Created 3 context files:
  /project/context/TASK-001-tdd-red.toml
  /project/context/TASK-002-tdd-red.toml
  /project/context/TASK-003-tdd-red.toml
```

---

## context check

Checks context budget against configured thresholds.

### Usage

```bash
projctl context check -d <project-dir>
```

### Flags

| Flag | Description |
|------|-------------|
| `-d, --dir` | Project directory (required) |

### Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Under warning threshold |
| 1 | Warning threshold exceeded |
| 2 | Limit threshold exceeded |

---

## Reference

See [orchestration-system.md](../orchestration-system.md) for the complete orchestration specification, including the yield protocol design.
