---
name: tdd-red-qa
description: Reviews TDD red phase output - verifies tests cover acceptance criteria and fail correctly
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: tdd-red
---

# TDD Red QA Skill

Reviews tests produced by tdd-red-producer to verify they cover acceptance criteria and fail for the correct reasons.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: REVIEW -> RETURN

This skill follows the QA-TEMPLATE pattern.

### 1. REVIEW Phase

Validate the producer's test output:

1. Read producer's test files from context
2. Load task acceptance criteria from task breakdown
3. Check each criterion:
   - Is there at least one test covering this criterion?
   - Do tests fail for the right reason (not compilation, import, or unrelated errors)?
   - Are tests testing behavior, not just structure?
   - Do tests use property-based testing where appropriate?
4. Compile findings

#### Red Phase Checklist

- [ ] Each acceptance criterion has at least one corresponding test
- [ ] Tests fail because implementation is missing (correct failure)
- [ ] Tests don't fail due to syntax, import, or compilation errors
- [ ] Tests describe expected behavior clearly
- [ ] Property tests used for invariants and edge cases
- [ ] Tests are blackbox (test public API, not internals)
- [ ] No implementation code beyond minimal stubs

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

All criteria pass. Tests are ready for green phase.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "internal/foo/bar_test.go"
test_count = 5
criteria_coverage = "3/3 acceptance criteria covered"
checklist = [
    { item = "Each acceptance criterion has tests", passed = true },
    { item = "Tests fail for correct reasons", passed = true },
    { item = "No compilation errors", passed = true },
    { item = "Tests describe expected behavior", passed = true }
]

[context]
phase = "tdd-red"
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
from_agent = "tdd-red-qa"
to_agent = "tdd-red-producer"
iteration = 2
issues = [
    "AC-2 'handles empty input' has no corresponding test",
    "Test for AC-1 fails due to import error, not missing implementation",
    "Tests are whitebox - testing unexported function directly"
]

[context]
phase = "tdd-red"
role = "qa"
iteration = 2
max_iterations = 3
```

#### `escalate-phase`

Problem discovered that requires changes to upstream artifacts. Used when:

- **error**: Tests reveal specification inconsistency
- **gap**: Acceptance criteria are untestable or incomplete
- **conflict**: Tests cannot satisfy conflicting requirements

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "tdd-red"
to_phase = "breakdown"
reason = "gap"

[payload.issue]
summary = "Acceptance criteria AC-3 is not testable"
context = "AC-3 states 'should be fast' but provides no measurable threshold"

[[payload.proposed_changes.requirements]]
action = "modify"
id = "AC-3"
title = "Performance requirement"
content = "Response time under 100ms for typical input"

[context]
phase = "tdd-red"
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
reason = "Ambiguous acceptance criteria"
context = "AC-4 can be interpreted two different ways"
question = "Should 'unique output' mean unique per run or unique globally?"
options = ["Unique per execution", "Globally unique (UUID)", "User configurable"]

[context]
phase = "tdd-red"
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

Red phase tests must:

1. **Cover all acceptance criteria**: Every AC has at least one test
2. **Fail correctly**: Tests fail because implementation doesn't exist, not due to errors
3. **Test behavior**: Focus on what, not how
4. **Use appropriate testing patterns**: Property tests for invariants, example tests for specific cases
5. **Be blackbox**: Test public API only
