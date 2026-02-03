---
name: design-qa
description: Reviews design.md for completeness, requirement coverage, and DES-N ID format
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: design
---

# Design QA

Review design decisions for completeness and requirement coverage.

**Pattern:** REVIEW → RETURN (see [QA-TEMPLATE.md](../shared/QA-TEMPLATE.md))

**Yield Protocol:** See [YIELD.md](../shared/YIELD.md)

## Quick Reference

| Aspect | Details |
|--------|---------|
| Domain | User interaction space - validates design.md content |
| Pattern | REVIEW → RETURN |
| Input | design.md (DES-N IDs), requirements.md (REQ-N IDs) |
| Output | Yield with approval or issues |
| Yields | `approved`, `improvement-request`, `escalate-phase`, `escalate-user` |

## Workflow

### 1. REVIEW Phase

Validate the producer's design.md artifact:

1. Read context from `[inputs]` section
2. Load design.md from configured path
3. Load requirements.md for traceability validation
4. Check each DES-N entry:
   - ID properly formatted (DES-NNN)?
   - Traces to at least one REQ-N?
   - Content describes WHAT user sees, not HOW built?
   - No conflicts with other DES entries?
5. Check requirement coverage:
   - Every user-facing REQ-N traced by at least one DES-N?
   - Missing designs identified?
6. Compile findings

### 2. RETURN Phase

Yield appropriate result based on REVIEW findings:

1. If all criteria pass:
   - Yield `approved`

2. If producer can fix issues:
   - Yield `improvement-request` with specific issues
   - Examples: missing traces, unclear descriptions, format errors

3. If requirements have gap/error/conflict:
   - Yield `escalate-phase` with proposed changes
   - Examples: discovered need not in requirements, conflicting requirements

4. If cannot resolve:
   - Yield `escalate-user` with question

## Validation Checklist

| Check | Criterion |
|-------|-----------|
| Format | All entries use DES-NNN format |
| Traces | Every DES-N traces to at least one REQ-N |
| Coverage | All user-facing REQ-N have corresponding DES-N |
| Content | Describes visual/interaction, not implementation |
| Consistency | No conflicting design decisions |
| Completeness | All screens/flows addressed |

## Yield Examples

### Approved

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "docs/design.md"
checklist = [
    { item = "All DES entries have valid IDs", passed = true },
    { item = "All DES entries trace to requirements", passed = true },
    { item = "All user-facing requirements covered", passed = true },
    { item = "Content describes user interaction", passed = true }
]

[context]
phase = "design"
role = "qa"
iteration = 1
```

### Improvement Request

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "design-qa"
to_agent = "design-interview-producer"
iteration = 2
issues = [
    "DES-003 missing trace to requirements",
    "DES-005 describes implementation ('uses React') instead of user interaction",
    "REQ-007 (user login) has no corresponding design"
]

[context]
phase = "design"
role = "qa"
iteration = 2
max_iterations = 3
```

### Escalate Phase

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "design"
to_phase = "pm"
reason = "gap"

[payload.issue]
summary = "Error handling not addressed in requirements"
context = "Design phase discovered need for error state designs but no requirements exist"

[[payload.proposed_changes.requirements]]
action = "add"
id = "REQ-012"
title = "Error State Display"
content = "System must display user-friendly error messages when operations fail"

[context]
phase = "design"
role = "qa"
escalating = true
```

## Rules

| Rule | Action |
|------|--------|
| Missing requirements.md | Yield `escalate-phase` - cannot validate without upstream |
| DES without REQ trace | Yield `improvement-request` - producer must add trace |
| REQ without DES | Yield `improvement-request` - producer must add design |
| Conflicting requirements | Yield `escalate-phase` to PM phase |
| Ambiguous requirement | Yield `escalate-user` with clarifying question |
