---
name: consistency-checker
description: "[DEPRECATED] Reviews parallel producer outputs for consistency across batch results"
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
deprecated: true
---

# Consistency Checker Skill

> **DEPRECATED (ISSUE-83):** This skill is deprecated. Its sole consumer was the parallel-looper skill, which has been replaced by native Claude Code team parallelism (ISSUE-79). The orchestrator now handles batch validation inline via QA teammates during parallel task execution. Do not use this skill for new work. It is retained temporarily for rollback purposes.

Reviews outputs from parallel producers to ensure consistency across all batch results. Applies domain-specific consistency rules passed as input.

## Workflow: REVIEW -> RETURN

This skill follows the QA pattern for batch QA operations.

### 1. REVIEW Phase

Validate consistency across all parallel results:

1. Read all producer artifacts from context (batch of results)
2. Load consistency rules from input context
3. Cross-validate each artifact against all others:
   - Are IDs unique across all results?
   - Do shared references resolve correctly?
   - Are conventions applied consistently?
   - Do related items agree on shared facts?
4. Compile inconsistencies by category

#### Consistency Checks

Cross-result validation:

- [ ] No duplicate IDs across parallel outputs
- [ ] Shared terminology used consistently
- [ ] Cross-references between items are valid
- [ ] Formatting conventions match across all items
- [ ] Related items do not contradict each other
- [ ] Domain-specific rules from input are satisfied

Per-item validation:

- [ ] Each item follows schema requirements
- [ ] Required fields are present
- [ ] Traces are valid

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

All parallel results are consistent. Batch is ready to proceed.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_items = 5
checklist = [
    { item = "No duplicate IDs", passed = true },
    { item = "Consistent terminology", passed = true },
    { item = "Valid cross-references", passed = true },
    { item = "Domain rules satisfied", passed = true }
]

[context]
role = "qa"
iteration = 1
batch_size = 5
```

#### `improvement-request`

Inconsistencies found that producers can fix. Uses batch format to address multiple items.

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "consistency-checker"
to_agent = "parallel-looper"
iteration = 2

[[payload.items]]
item_id = "TASK-3"
producer = "breakdown-producer-1"
issues = [
    "Uses 'user' but TASK-1 uses 'customer' - standardize terminology",
    "References TASK-2 but TASK-2 outputs don't exist"
]
suggested_resolution = "Align terminology with TASK-1, verify TASK-2 reference"

[[payload.items]]
item_id = "TASK-5"
producer = "breakdown-producer-2"
issues = [
    "Duplicate ID - TASK-5 already produced by breakdown-producer-1"
]
suggested_resolution = "Renumber to TASK-6"

[context]
role = "qa"
iteration = 2
max_iterations = 3
batch_size = 5
```

#### `escalate-phase`

Problem discovered that requires changes to upstream artifacts. Used when:

- **error**: Producer made a mistake that violates constraints
- **gap**: Missing content that should exist based on upstream context
- **conflict**: Contradictory statements across artifacts that cannot be resolved by individual fixes

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "breakdown"
to_phase = "arch"
reason = "conflict"  # error | gap | conflict

[payload.issue]
summary = "Parallel breakdown revealed architectural ambiguity"
context = "Multiple producers interpreted API boundary differently"

[[payload.proposed_changes.architecture]]
action = "clarify"
id = "ARCH-3"
title = "API Boundary Definition"
change = "Explicitly define which components own user authentication"

[context]
role = "qa"
escalating = true
```

#### `escalate-user`

Cannot resolve inconsistency without user input.

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Irreconcilable approach differences"
context = "Producer-1 used event sourcing, Producer-2 used CRUD - both valid per requirements"
question = "Which architectural approach should we standardize on?"
options = ["Event sourcing", "CRUD", "Hybrid by domain"]

[context]
role = "qa"
escalating = true
```

## Domain-Specific Consistency Rules

Consistency rules are passed as input context. The checker applies these rules across all parallel results.

### Rule Format

```toml
# context/consistency-rules.toml

[[rules]]
name = "unique_ids"
description = "Each ID must be unique across all parallel outputs"
scope = "cross-item"
severity = "error"

[[rules]]
name = "terminology"
description = "Use consistent terms for same concepts"
scope = "cross-item"
severity = "warning"
terms = [
    { canonical = "customer", alternatives = ["user", "client"] },
    { canonical = "repository", alternatives = ["repo", "storage"] }
]

[[rules]]
name = "trace_format"
description = "Traces must follow ARTIFACT-NNN format"
scope = "per-item"
severity = "error"
pattern = "^[A-Z]+-\\d+$"
```

### Scope Types

| Scope | Description |
|-------|-------------|
| `cross-item` | Rule validates relationships between items |
| `per-item` | Rule validates each item independently |

### Severity Levels

| Severity | Meaning | Action |
|----------|---------|--------|
| `error` | Must fix before proceeding | Yields improvement-request |
| `warning` | Should fix, but can proceed with caveats | Notes in approved yield |

## Inconsistency Documentation

When documenting inconsistencies, include:

1. **What**: Specific description of the inconsistency
2. **Where**: Which items are affected (item IDs, producer names)
3. **Impact**: Why this inconsistency matters
4. **Resolution**: Suggested fix

### Example Inconsistency Report

```markdown
## Inconsistency: Duplicate Task ID

**What:** TASK-5 appears in outputs from both breakdown-producer-1 and breakdown-producer-2

**Where:**
- breakdown-producer-1: TASK-5 "Implement user authentication"
- breakdown-producer-2: TASK-5 "Add caching layer"

**Impact:** Task tracking will fail - cannot have duplicate IDs in task list

**Resolution:** Renumber breakdown-producer-2 output to TASK-6+
```

## Iteration Limits

Consistency checker tracks iterations to prevent infinite loops:

```toml
[context]
iteration = 2
max_iterations = 3
```

After max iterations:
1. Yield `escalate-user` if critical inconsistencies remain
2. Or yield `approved` with warnings noted in payload

## Partial Failure Handling

When some parallel producers failed:

1. Mark failed items in report
2. Validate only successful outputs
3. Note failed items in yield payload

```toml
[payload]
successful_items = 4
failed_items = ["TASK-7"]
failed_reason = "Producer timeout"
```

## Integration with Parallel Looper

The consistency-checker is invoked by parallel-looper after all parallel PAIR LOOPs complete:

```
parallel-looper spawns N PAIR LOOPs
    ↓
All PAIR LOOPs complete (or timeout)
    ↓
parallel-looper aggregates results
    ↓
parallel-looper invokes consistency-checker
    ↓
consistency-checker reviews batch
    ↓
If approved: parallel-looper yields complete
If improvement-request: parallel-looper re-runs affected items
If escalate: parallel-looper forwards escalation
```
