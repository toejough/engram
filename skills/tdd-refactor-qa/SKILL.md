---
name: tdd-refactor-qa
description: Reviews TDD refactor phase output - verifies tests pass and code quality improved
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: tdd-refactor
---

# TDD Refactor QA Skill

Reviews refactored code from tdd-refactor-producer to verify tests still pass and code quality has improved.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: REVIEW -> RETURN

This skill follows the QA-TEMPLATE pattern.

### 1. REVIEW Phase

Validate the producer's refactored code:

1. Read producer's refactored files from context
2. Run test suite to verify tests still pass (green)
3. Run linter to verify quality improvements
4. Compare before/after for quality metrics
5. Compile findings

#### Refactor Phase Checklist

- [ ] All tests still pass after refactoring
- [ ] Linter issues reduced or eliminated
- [ ] No new linter issues introduced
- [ ] Behavior unchanged (tests prove this)
- [ ] Code readability improved
- [ ] No blanket lint suppressions added

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

Tests still green and quality improved. Task complete.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "internal/foo/bar.go"
test_results = "15/15 tests passing"
lint_before = 8
lint_after = 0
quality_improvement = "Reduced cyclomatic complexity from 12 to 6"
checklist = [
    { item = "All tests still pass", passed = true },
    { item = "Linter issues resolved", passed = true },
    { item = "No new lint issues", passed = true },
    { item = "Behavior unchanged", passed = true }
]

[context]
phase = "tdd-refactor"
role = "qa"
iteration = 1
```

#### `improvement-request`

Issues found that the producer can fix.

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "tdd-refactor-qa"
to_agent = "tdd-refactor-producer"
iteration = 2
issues = [
    "TestFoo now fails after refactoring - behavior changed",
    "New lint issue: G101 potential hardcoded credential",
    "Added nolint:dupl without justification"
]

[context]
phase = "tdd-refactor"
role = "qa"
iteration = 2
max_iterations = 3
```

#### `escalate-phase`

Problem discovered that requires changes to upstream artifacts. Used when:

- **error**: Refactoring revealed implementation bug
- **gap**: Tests don't cover behavior that refactoring changed
- **conflict**: Lint rules conflict with architectural patterns

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "tdd-refactor"
to_phase = "tdd-red"
reason = "gap"

[payload.issue]
summary = "Refactoring revealed untested edge case"
context = "Simplifying the loop exposed that empty input case was never tested"

[[payload.proposed_changes.requirements]]
action = "add"
id = "test-empty-input"
title = "Add test for empty input"
content = "Add property test verifying behavior with empty input"

[context]
phase = "tdd-refactor"
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
reason = "Lint rule conflicts with project style"
context = "funcorder wants specific function ordering but project uses different convention"
question = "Should we follow lint rule or project convention?"
options = ["Follow lint rule (reorder functions)", "Add exclusion for this rule", "Modify lint config project-wide"]

[context]
phase = "tdd-refactor"
role = "qa"
escalating = true
```

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

## Quality Criteria

Refactor phase output must:

1. **Tests stay green**: All tests pass before and after
2. **Quality improves**: Lint issues reduced, complexity lowered
3. **Behavior unchanged**: Tests prove no functional changes
4. **No suppressions**: Don't hide problems with nolint/blanket excludes
5. **Readability**: Code is clearer than before
