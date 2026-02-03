# QA Skill Template

Template for creating QA skills that follow the REVIEW → RETURN pattern.

## Frontmatter

```yaml
---
name: <phase>-qa
description: <Brief description of what this QA validates>
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: <pm | design | arch | breakdown | doc | tdd-red | tdd-green | tdd-refactor | tdd | alignment | retro | summary>
---
```

## Workflow Pattern

QA skills follow the REVIEW → RETURN pattern:

### 1. REVIEW

Validate the producer's output:
- Read the artifact from context
- Check against acceptance criteria
- Verify traceability links
- Identify issues or gaps

```markdown
## REVIEW Phase

1. Read producer's artifact from context
2. Load relevant acceptance criteria
3. Check each criterion:
   - IDs properly formatted?
   - Traces to upstream artifacts?
   - Content complete and accurate?
   - No conflicts with other artifacts?
4. Compile findings
```

### 2. RETURN

Yield appropriate result:
- `approved`: All criteria met
- `improvement-request`: Issues found, producer can fix
- `escalate-phase`: Problem in upstream phase
- `escalate-user`: Cannot resolve without user

```markdown
## RETURN Phase

Based on REVIEW findings:

1. If all criteria pass:
   - Yield `approved`

2. If producer can fix issues:
   - Yield `improvement-request` with specific issues

3. If upstream phase has error/gap/conflict:
   - Yield `escalate-phase` with proposed changes

4. If cannot resolve:
   - Yield `escalate-user` with question
```

## Yield Format

See [YIELD.md](./YIELD.md) for full protocol specification.

QA skills can yield:

| Type | When to Use |
|------|-------------|
| `approved` | Work passes all checks |
| `improvement-request` | Issues producer can fix |
| `escalate-phase` | Upstream phase has problems |
| `escalate-user` | Cannot resolve without user input |

### Approved Yield Example

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "docs/requirements.md"
checklist = [
    { item = "All requirements have IDs", passed = true },
    { item = "Acceptance criteria are testable", passed = true },
    { item = "Traces to issue", passed = true }
]

[context]
phase = "pm"
role = "qa"
iteration = 1
```

### Improvement Request Yield Example

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "pm-qa"
to_agent = "pm-interview-producer"
iteration = 2
issues = [
    "REQ-3 acceptance criteria are not measurable",
    "REQ-5 missing edge case for empty input"
]

[context]
phase = "pm"
role = "qa"
iteration = 2
max_iterations = 3
```

## Escalation Responsibilities

QA skills are responsible for escalating when they discover:

### Error
Producer made a mistake that violates constraints:
```toml
[payload.escalation]
reason = "error"
```
Example: Architectural decision conflicts with a stated requirement.

### Gap
Missing content that should exist based on upstream artifacts:
```toml
[payload.escalation]
reason = "gap"
```
Example: Design phase discovered need not captured in requirements.

### Conflict
Contradictory statements across artifacts:
```toml
[payload.escalation]
reason = "conflict"
```
Example: REQ-3 says "offline first" but ARCH-2 assumes always-online.

## Escalate-Phase Yield

When escalating to a prior phase, QA must include proposed changes:

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "design"
to_phase = "pm"
reason = "gap"  # error | gap | conflict

[payload.issue]
summary = "Parallelism not addressed in requirements"
context = "Design phase discovered need for context exploration"

[[payload.proposed_changes.requirements]]
action = "add"
id = "REQ-10"
title = "Context Exploration via Yield"
content = "Producer skills can yield need-context with queries..."

[[payload.proposed_changes.source_docs]]
file = "docs/orchestration-system.md"
section = "3.2 Yield Types"
change = "Add need-context yield type"

[context]
phase = "design"
role = "qa"
escalating = true
```

The `proposed_changes` section contains:
- `requirements`: New or modified requirements to add
- `source_docs`: Supporting documentation changes

The upstream phase producer receives these proposed changes and can accept, modify, or reject them.

## Escalate-User Yield

When QA cannot resolve an issue:

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Cannot resolve conflict between requirements"
context = "REQ-3 and REQ-7 appear contradictory"
question = "Should offline mode take priority over real-time sync?"
options = ["Offline first", "Real-time first", "User configurable"]

[context]
phase = "design"
role = "qa"
escalating = true
```

## Context Reading

On invocation:

```markdown
1. Read context file at `<project>/.claude/context/<skill>-context.toml`
2. Read producer's artifact from path in context
3. Check `invocation.task` for scope:
   - "PHASE" = reviewing full phase output
   - "TASK-N" = reviewing specific task output
4. Write yield to `[output].yield_path`
```

## Iteration Limits

QA tracks iterations to prevent infinite loops:

```toml
[context]
iteration = 2
max_iterations = 3
```

After max iterations, QA should:
1. Yield `escalate-user` if issues remain
2. Or yield `approved` with caveats noted in payload

## Checklist Format

Use consistent checklist format in `approved` yields:

```toml
[[payload.checklist]]
item = "All IDs follow format"
passed = true

[[payload.checklist]]
item = "Traces link to upstream"
passed = true
note = "REQ-3 traces to ISSUE-001"
```
