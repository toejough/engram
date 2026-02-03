---
name: context-qa
description: Validates gathered context for completeness, relevance, and consistency
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: context
---

# Context QA Skill

Validates context gathered by context-explorer for quality and usefulness before returning to the requesting producer.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: REVIEW -> RETURN

This skill follows the QA-TEMPLATE pattern.

### 1. REVIEW Phase

Validate the context-explorer's gathered results:

1. Read context results from `[query_results]` in context file
2. Compare against original queries from producer's `need-context` yield
3. Check each validation criterion:
   - All queries answered? (no missing results)
   - Results relevant to original request?
   - Any contradictions between sources?
   - Any stale or outdated information?
4. Compile findings

#### Context Quality Checklist

- [ ] All requested queries have results (no unanswered queries)
- [ ] Results are relevant to the requesting producer's task
- [ ] No contradictions detected between multiple sources
- [ ] No stale or outdated information (check dates, versions)
- [ ] File contents exist (not "file not found" errors)
- [ ] Semantic queries answered the actual question asked
- [ ] Memory results are from relevant projects/sessions

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

All context is valid and useful. Ready to return to producer.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "query_results"
queries_validated = 3
checklist = [
    { item = "All queries answered", passed = true },
    { item = "Results are relevant", passed = true },
    { item = "No contradictions", passed = true },
    { item = "No stale information", passed = true }
]

[context]
phase = "context"
role = "qa"
iteration = 1
```

#### `improvement-request`

Issues found that context-explorer can fix (e.g., retry failed queries).

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "context-qa"
to_agent = "context-explorer"
iteration = 2
issues = [
    "Query 1 (file: docs/requirements.md) returned 'file not found' - verify path",
    "Query 3 (semantic: authentication) answer does not address the question asked"
]

[context]
phase = "context"
role = "qa"
iteration = 2
max_iterations = 3
```

#### `escalate-phase`

Problem discovered that requires changes to the original need-context yield. Used when:

- **error**: Explorer made a mistake (wrong file path, malformed query)
- **gap**: Missing information that should have been requested
- **conflict**: Contradictory information between sources

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "context"
to_phase = "context"
reason = "conflict"  # error | gap | conflict

[payload.issue]
summary = "Contradictory information in gathered context"
context = "docs/architecture.md says REST API but docs/design.md says GraphQL"

[[payload.proposed_changes.queries]]
action = "add"
type = "semantic"
question = "Which API style is actually implemented in the codebase?"

[context]
phase = "context"
role = "qa"
escalating = true
```

#### `escalate-user`

Cannot resolve issue without user input.

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Cannot determine which source is authoritative"
context = "Found contradicting information between design doc and implementation"
question = "Which is correct: the design doc (REST) or the implementation (GraphQL)?"
options = ["Design doc is correct - fix implementation", "Implementation is correct - update design doc"]

[context]
phase = "context"
role = "qa"
escalating = true
```

## Validation Criteria

### 1. Query Completeness

Check that all requested queries have results:

```markdown
Original request:
- file: docs/requirements.md
- memory: "caching patterns"
- semantic: "How does authentication work?"

Results received:
- file: docs/requirements.md -> [content]
- memory: "caching patterns" -> [results]
- semantic: "How does authentication work?" -> [answer]

Status: All queries answered
```

### 2. Relevance Assessment

Verify results actually address what was asked:

| Query Type | Relevance Check |
|------------|-----------------|
| `file` | File exists and contains expected content type |
| `memory` | Results relate to the query topic |
| `semantic` | Answer addresses the specific question asked |
| `web` | Content matches the prompt's extraction goal |
| `territory` | Map covers the requested scope |

### 3. Contradiction Detection

Look for conflicting information across sources:

```markdown
## Contradiction Found

Source 1 (docs/design.md):
> "API uses REST with JSON payloads"

Source 2 (semantic query result):
> "The codebase implements GraphQL with Apollo"

Action: Flag for resolution before returning to producer
```

### 4. Staleness Detection

Check for outdated or stale information:

| Indicator | Action |
|-----------|--------|
| File modified date > 6 months | Flag as potentially stale |
| References deprecated APIs | Flag for verification |
| Memory from old session | Check if still applicable |
| Version numbers outdated | Note version mismatch |

## Iteration Limits

QA tracks iterations to prevent infinite loops:

```toml
[context]
iteration = 2
max_iterations = 3
```

After max iterations:
1. Yield `escalate-user` if issues remain unresolved
2. Or yield `approved` with caveats noted in payload

## Context Reading

On invocation:

```markdown
1. Read context file at `<project>/.claude/context/context-qa-context.toml`
2. Extract original queries from `[inputs.original_queries]`
3. Extract results from `[query_results]`
4. Compare and validate
5. Write yield to `[output].yield_path`
```
