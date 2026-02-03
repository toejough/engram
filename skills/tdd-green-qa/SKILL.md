---
name: tdd-green-qa
description: Reviews TDD green phase output - verifies all tests pass with no regressions
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: tdd-green
---

# TDD Green QA Skill

Reviews implementation produced by tdd-green-producer to verify all tests pass with no regressions.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: REVIEW -> RETURN

This skill follows the QA-TEMPLATE pattern.

### 1. REVIEW Phase

Validate the producer's implementation:

1. Read producer's implementation files from context
2. Run test suite to verify all tests pass
3. Check for regressions in existing tests
4. Verify implementation follows minimal approach
5. Compile findings

#### Green Phase Checklist

- [ ] All new tests from red phase pass
- [ ] All existing tests still pass (no regressions)
- [ ] Implementation is minimal (no over-engineering)
- [ ] Implementation follows architecture patterns
- [ ] No new tests added (that's not green phase's job)
- [ ] Build succeeds with no errors

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

All tests pass. Implementation ready for refactor phase.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "internal/foo/bar.go"
test_results = "15/15 tests passing"
regression_check = "No regressions detected"
checklist = [
    { item = "All new tests pass", passed = true },
    { item = "No regressions in existing tests", passed = true },
    { item = "Implementation is minimal", passed = true },
    { item = "Build succeeds", passed = true }
]

[context]
phase = "tdd-green"
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
from_agent = "tdd-green-qa"
to_agent = "tdd-green-producer"
iteration = 2
issues = [
    "Test 'TestFoo' still failing: expected 42, got 0",
    "Regression: TestExistingFeature now fails after changes",
    "Implementation includes unnecessary refactoring (save for refactor phase)"
]

[context]
phase = "tdd-green"
role = "qa"
iteration = 2
max_iterations = 3
```

#### `escalate-phase`

Problem discovered that requires changes to upstream artifacts. Used when:

- **error**: Tests themselves are incorrect
- **gap**: Tests missing critical case that implementation revealed
- **conflict**: Implementation cannot satisfy conflicting test expectations

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "tdd-green"
to_phase = "tdd-red"
reason = "error"

[payload.issue]
summary = "Test expects impossible state"
context = "TestConcurrency expects atomic operation but API is inherently non-atomic"

[[payload.proposed_changes.requirements]]
action = "modify"
id = "test-concurrency"
title = "Fix concurrency test"
content = "Modify test to account for non-atomic API semantics"

[context]
phase = "tdd-green"
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
reason = "Implementation requires architectural decision"
context = "To make tests pass, need to choose between two approaches"
question = "Should we use synchronous or asynchronous processing?"
options = ["Synchronous (simpler, blocking)", "Asynchronous (complex, non-blocking)"]

[context]
phase = "tdd-green"
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

Green phase output must:

1. **All tests pass**: New and existing
2. **No regressions**: Existing tests unchanged and passing
3. **Minimal implementation**: Just enough to make tests pass
4. **Follow patterns**: Respect architectural decisions from design phase
5. **Build clean**: No warnings or errors
