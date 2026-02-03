---
name: summary-qa
description: Validate project summary accuracy and completeness
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: summary
---

# Summary QA

Validate project summary for accuracy, completeness, and proper traceability to upstream artifacts.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML with summary artifact path |
| Validates | Project summary document |
| Criteria | Accuracy, completeness, traceability |

## Workflow

Follows REVIEW -> RETURN pattern per QA-TEMPLATE.

### REVIEW

1. Read context from `[inputs]` section
2. Load project summary from artifact path
3. Load upstream artifacts:
   - REQ-N from requirements.md
   - DES-N from design.md
   - ARCH-N from architecture.md
   - TASK-N from tasks.md
4. Check accuracy criteria:
   - Decisions accurately described?
   - Outcomes match actual results?
   - Traces reference valid IDs?
5. Check completeness criteria:
   - All major decisions captured?
   - All outcomes documented?
   - Lessons learned included?
6. Compile findings

### RETURN

Based on REVIEW findings:

1. If all criteria pass:
   - Yield `approved` with checklist

2. If producer can fix issues:
   - Yield `improvement-request` with specific issues

3. If upstream phase has error/gap/conflict:
   - Yield `escalate-phase` with proposed changes

4. If cannot resolve:
   - Yield `escalate-user` with question

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol.

### Yield Types

| Type | When |
|------|------|
| `approved` | Summary passes all checks |
| `improvement-request` | Issues producer can fix |
| `escalate-phase` | Upstream phase has problems |
| `escalate-user` | Cannot resolve without user input |

### Approved Yield Example

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "docs/project-summary.md"
checklist = [
    { item = "All major decisions documented", passed = true },
    { item = "Outcomes match implementation", passed = true },
    { item = "Traces to REQ-N, DES-N, ARCH-N, TASK-N valid", passed = true },
    { item = "Lessons learned section present", passed = true }
]

[context]
phase = "summary"
role = "qa"
iteration = 1
```

### Improvement Request Yield Example

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "summary-qa"
to_agent = "summary-producer"
iteration = 2
issues = [
    "Missing decision about database selection (ARCH-2)",
    "Outcome claims 95% coverage but actual is 87%",
    "TASK-7 not referenced in completed tasks"
]

[context]
phase = "summary"
role = "qa"
iteration = 2
max_iterations = 3
```

### Escalate-Phase Yield Example

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "summary"
to_phase = "breakdown"
reason = "gap"

[payload.issue]
summary = "Task completion status unclear"
context = "TASK-5 marked complete but no evidence of implementation"

[[payload.proposed_changes.requirements]]
action = "update"
id = "TASK-5"
title = "Clarify completion status"
content = "Add evidence of completion or mark as incomplete..."

[context]
phase = "summary"
role = "qa"
escalating = true
```

### Escalate-User Yield Example

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Cannot verify outcome claim"
context = "Summary claims 'zero production bugs' but no monitoring data available"
question = "Should this claim be removed or is there evidence to support it?"
options = ["Remove claim", "Add evidence source", "Mark as unverified"]

[context]
phase = "summary"
role = "qa"
escalating = true
```

## Validation Criteria

### Accuracy

- [ ] Decision descriptions match actual choices made
- [ ] Outcomes reflect actual implementation results
- [ ] Metrics are verifiable (coverage %, performance numbers)
- [ ] Timeline and milestones are accurate
- [ ] No contradictions with upstream artifacts

### Completeness

- [ ] Executive overview present
- [ ] All major decisions documented
- [ ] All project outcomes captured
- [ ] Lessons learned section included
- [ ] Key trade-offs explained
- [ ] Known limitations documented

### Traceability

- [ ] Decisions trace to REQ-N, DES-N, ARCH-N
- [ ] Outcomes trace to TASK-N
- [ ] No orphan traces (referencing non-existent IDs)
- [ ] All significant artifacts referenced

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

## Result Format

Yield files written to path specified in context (`output.yield_path`).

## Full Documentation

`projctl skills docs --skillname summary-qa` or see SKILL-full.md
