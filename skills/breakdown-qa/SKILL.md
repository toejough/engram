---
name: breakdown-qa
description: Validate task breakdown for completeness and valid dependency DAG
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: breakdown
---

# Breakdown QA

Validate task decomposition completeness and dependency graph structure.

**Yield Protocol:** See [YIELD.md](../shared/YIELD.md)

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | tasks.md from breakdown-producer |
| Pattern | REVIEW -> RETURN |
| Output | approved, improvement-request, or escalate-phase yield |

---

## REVIEW Phase

1. Read breakdown-producer's artifact (tasks.md)

2. Validate TASK-N structure:
   - All tasks have unique TASK-N IDs
   - Sequential numbering (no gaps)
   - Each task has required fields

3. **Validate traceability (REQUIRED):**
   - Every task MUST have a `**Traces to:**` field
   - Run `projctl trace validate -d <project-dir>`
   - If orphan IDs exist (referenced but undefined), reject the breakdown
   - If unlinked IDs exist (defined but not connected), flag for review

4. Validate decomposition completeness:
   - All ARCH-N IDs have at least one implementing TASK
   - No orphan tasks (all trace to ARCH/DES/REQ)
   - Appropriate granularity (not too large/small)

5. Validate dependency graph:
   - All dependencies reference valid TASK-N IDs
   - No cycle in dependency graph (must be DAG)
   - No prose dependencies ("all previous", "depends on earlier")

6. Check acceptance criteria:
   - Each task has testable criteria
   - Criteria are measurable, not vague

---

## RETURN Phase

Based on REVIEW findings:

### Yield `approved`

When all criteria pass:

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "docs/tasks.md"
checklist = [
    { item = "All tasks have TASK-N IDs", passed = true },
    { item = "Decomposition is complete", passed = true },
    { item = "Dependency graph is valid DAG", passed = true },
    { item = "All tasks have Traces-to field", passed = true },
    { item = "projctl trace validate passes", passed = true }
]

[context]
phase = "breakdown"
role = "qa"
iteration = 1
```

### Yield `improvement-request`

When producer can fix issues:

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "breakdown-qa"
to_agent = "breakdown-producer"
iteration = 2
issues = [
    "TASK-3 has cycle with TASK-5",
    "ARCH-4 has no implementing task",
    "TASK-7 missing acceptance criteria",
    "TASK-8 missing Traces-to field",
    "projctl trace validate: orphan ID ARCH-99 referenced by TASK-5"
]

[context]
phase = "breakdown"
role = "qa"
iteration = 2
max_iterations = 3
```

### Yield `escalate-phase`

When architecture or design has gaps:

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "breakdown"
to_phase = "arch"
reason = "gap"

[payload.issue]
summary = "Component interface undefined"
context = "Cannot decompose storage layer without ARCH decision"

[[payload.proposed_changes.architecture]]
action = "add"
id = "ARCH-10"
title = "Storage Layer Interface"
content = "Define interface for persistence abstraction"

[context]
phase = "breakdown"
role = "qa"
escalating = true
```

---

## Validation Checklist

| Check | Pass Criteria |
|-------|---------------|
| ID format | All tasks match TASK-N pattern |
| Sequencing | No gaps in numbering |
| Completeness | All ARCH IDs covered |
| DAG validity | No dependency cycle detected |
| Traces-to field | Every task has `**Traces to:**` field |
| projctl validate | `projctl trace validate` passes with no orphans |
| Criteria | All tasks have testable acceptance |

---

## Traceability Validation

**Every task MUST have a Traces-to field.** This is non-negotiable.

### Manual Check

Scan tasks.md for tasks missing the field:
```
**Traces to:** ARCH-NNN
```

### Automated Validation

Run `projctl trace validate` on the project directory:

```bash
projctl trace validate -d <project-dir>
```

**Pass:** Output shows "Validation passed: all IDs properly linked"

**Fail:** Output lists orphan or unlinked IDs. Example:
```
Orphan IDs (referenced but not defined):
  - ARCH-99 (referenced by TASK-5)

Unlinked IDs (defined but not connected):
  - TASK-12
```

### Rejection Criteria

Reject the breakdown (yield `improvement-request`) if:
- Any task is missing `**Traces to:**` field
- `projctl trace validate` reports orphan IDs (referenced but undefined)
- Tasks trace to non-existent upstream IDs

---

## Cycle Detection

To verify DAG (no dependency cycle):

1. Build adjacency list from Dependencies fields
2. Run DFS from each unvisited node
3. Track visiting (gray) vs visited (black) nodes
4. Back edge to gray node = cycle found

Example cycle violation:
```
TASK-1 -> TASK-2 -> TASK-3 -> TASK-1  # INVALID
```

---

## Yield Types Used

| Type | When |
|------|------|
| `approved` | Tasks complete, valid DAG, all traces valid, projctl validate passes |
| `improvement-request` | Producer can fix (cycles, gaps, missing criteria, missing Traces-to, orphan IDs) |
| `escalate-phase` | Architecture/design needs upstream change |
| `escalate-user` | Cannot resolve conflict |
