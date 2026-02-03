---
name: tdd-qa
description: Validates overall TDD cycle compliance and acceptance criteria
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: tdd
---

# TDD QA Skill

Validates overall acceptance criteria compliance after the RED/GREEN/REFACTOR cycle completes. Ensures TDD discipline was followed throughout.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: REVIEW -> RETURN

This skill follows the QA-TEMPLATE pattern.

### 1. REVIEW Phase

Validate that TDD was executed correctly and acceptance criteria are satisfied:

1. Read task context and acceptance criteria
2. Verify RED phase discipline:
   - Tests were written first (before implementation)
   - Tests initially failed for correct reasons
   - All acceptance criteria mapped to at least one test
3. Verify GREEN phase discipline:
   - Implementation was minimal (no premature optimization)
   - All tests now pass
   - No behavior beyond acceptance criteria added
4. Verify REFACTOR phase discipline:
   - Tests stayed green throughout
   - No behavior changes during refactoring
   - Code quality improved (linter issues fixed)
5. Validate overall acceptance criteria:
   - Each criterion has passing test(s)
   - Implementation matches requirements
   - No regressions in existing functionality
6. Compile findings

#### TDD Discipline Checklist

- [ ] Tests written before implementation (RED first)
- [ ] Tests failed initially for correct reasons
- [ ] Minimal implementation (GREEN minimal)
- [ ] All tests pass after implementation
- [ ] Refactoring preserved behavior (tests stayed GREEN)
- [ ] Linter issues addressed during REFACTOR

#### Acceptance Criteria Checklist

- [ ] Each AC has corresponding test(s)
- [ ] All AC tests pass
- [ ] Implementation matches requirements spec
- [ ] No unrelated changes included
- [ ] Traceability maintained (TASK-N links)

### 2. RETURN Phase

Based on REVIEW findings, yield one of:

#### `approved`

TDD cycle complete. All acceptance criteria validated.

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
task = "TASK-5"
reviewed_artifacts = ["tests/foo_test.go", "internal/foo/foo.go"]
checklist = [
    { item = "Tests written first", passed = true },
    { item = "All AC have tests", passed = true },
    { item = "All tests pass", passed = true },
    { item = "Refactoring preserved behavior", passed = true },
    { item = "Linter issues fixed", passed = true }
]

[context]
phase = "tdd"
role = "qa"
iteration = 1
```

#### `improvement-request`

Issues found that the TDD producer(s) can fix.

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "tdd-qa"
to_agent = "tdd-red"  # or tdd-green, tdd-refactor
iteration = 2
issues = [
    "AC-3 missing test coverage",
    "Implementation added behavior not in acceptance criteria",
    "Linter issues remain unfixed"
]

[context]
phase = "tdd"
role = "qa"
iteration = 2
max_iterations = 3
```

#### `escalate-phase`

Problem discovered that requires changes to upstream artifacts. Used when:

- **error**: TDD phase violated design/architecture constraints
- **gap**: Acceptance criteria insufficient to implement feature
- **conflict**: Implementation reveals contradictions in design

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "tdd"
to_phase = "breakdown"  # or design, arch
reason = "gap"  # error | gap | conflict

[payload.issue]
summary = "Acceptance criteria incomplete for edge case"
context = "During TDD, discovered unspecified behavior for empty input"

[[payload.proposed_changes.tasks]]
action = "modify"
id = "TASK-5"
change = "Add acceptance criterion for empty input handling"

[context]
phase = "tdd"
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
context = "AC-2 says 'fast response' but no measurable threshold defined"
question = "What is the acceptable response time threshold?"
options = ["<100ms", "<500ms", "<1s", "User configurable"]

[context]
phase = "tdd"
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

TDD execution must demonstrate:

1. **Discipline**: Tests first, minimal implementation, clean refactor
2. **Coverage**: All acceptance criteria tested
3. **Correctness**: Implementation matches spec
4. **Cleanliness**: Linter issues addressed
5. **Traceability**: Links to TASK-N maintained
