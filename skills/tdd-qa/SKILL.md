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

1. Read task context and acceptance criteria from tasks.md
2. Parse AC completeness:
   - Extract all acceptance criteria items (`- [ ]` or `- [x]`)
   - Track which AC items are checked vs unchecked
   - Note any deferral language in producer artifacts or commits
3. Verify RED phase discipline:
   - Tests were written first (before implementation)
   - Tests initially failed for correct reasons
   - All acceptance criteria mapped to at least one test
4. Verify GREEN phase discipline:
   - Implementation was minimal (no premature optimization)
   - All tests now pass
   - No behavior beyond acceptance criteria added
5. Verify REFACTOR phase discipline:
   - Tests stayed green throughout
   - No behavior changes during refactoring
   - Code quality improved (linter issues fixed)
6. Validate overall acceptance criteria:
   - Each criterion has passing test(s)
   - Implementation matches requirements
   - No regressions in existing functionality
7. Check for AC completeness violations:
   - Any `[ ]` (unchecked) AC items = incomplete work
   - Deferral language ("defer", "skip", "out of scope", "later", "future") in producer output = escalation needed
8. Compile findings

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
- [ ] All AC items are `[x]` (none remain `[ ]`)
- [ ] No deferral language in producer artifacts

#### Visual Evidence Checklist (for `[visual]` tasks)

- [ ] Screenshot or output capture provided
- [ ] Visual matches acceptance criteria
- [ ] No obvious visual defects (blank, corrupted, misaligned)

### Visual Task Validation

For tasks with `[visual]` marker in the title:

1. **Check for visual evidence in producer yield**:
   - `visual_verified = true` must be present
   - `visual_evidence` path must be provided

2. **If evidence missing**, yield `improvement-request`:
   ```toml
   [yield]
   type = "improvement-request"
   timestamp = 2026-02-02T12:05:00Z

   [payload]
   from_agent = "tdd-qa"
   to_agent = "tdd-green"
   iteration = 2
   issues = ["Visual verification required for [visual] task but no evidence provided"]
   missing_visual_evidence = true

   [context]
   phase = "tdd"
   role = "qa"
   iteration = 2
   max_iterations = 3
   ```

3. **Waiver process**: If visual verification is impractical:
   - Producer must explain why in their yield payload
   - QA escalates to user for explicit approval via `escalate-user`

   ```toml
   [yield]
   type = "escalate-user"
   timestamp = 2026-02-02T12:15:00Z

   [payload]
   reason = "Visual verification waiver requested"
   context = "Producer claims visual verification impractical: 'No browser available in CI environment'"
   waiver_reason = "No browser available in CI environment"
   question = "Approve visual verification waiver, or require evidence?"
   options = [
       "Approve waiver - accept without visual evidence",
       "Reject waiver - must provide visual evidence",
       "Defer task - wait for proper tooling"
   ]

   [context]
   phase = "tdd"
   role = "qa"
   escalating = true
   visual_waiver_requested = true
   ```

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

**Yield `improvement-request` for unchecked AC items:**

When parsing the task definition shows acceptance criteria that remain unchecked (`[ ]`), this indicates incomplete work.

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "tdd-qa"
to_agent = "tdd-green"
iteration = 2
issues = [
    "AC incomplete: '[ ] Error if task doesn't exist in tasks.md' - not implemented",
    "AC incomplete: '[ ] Unit tests cover all new methods' - missing test coverage"
]
unchecked_ac = [
    "Error if task doesn't exist in tasks.md",
    "Unit tests cover all new methods"
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

**Yield `escalate-user` for deferral claims:**

When producer output contains deferral language (defer, skip, out of scope, later, future), escalate to user for explicit approval. Do not allow silent deferral.

```toml
[yield]
type = "escalate-user"
timestamp = 2026-02-02T12:15:00Z

[payload]
reason = "Producer claimed deferral without user approval"
context = "Producer output contains: 'Error handling deferred to future task'"
deferral_detected = "Error handling deferred to future task"
question = "Approve deferral of error handling, or require completion now?"
options = [
    "Approve deferral - create follow-up issue",
    "Reject deferral - must complete in this task",
    "Modify scope - adjust acceptance criteria"
]

[context]
phase = "tdd"
role = "qa"
escalating = true
deferral_keywords_found = ["deferred", "future"]
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

## AC Completeness Validation

This skill enforces strict acceptance criteria completeness. No silent deferrals.

### Parsing AC from tasks.md

1. Locate the task definition in tasks.md by TASK-N ID
2. Extract all lines matching `- [ ]` or `- [x]` under **Acceptance Criteria:**
3. Build a checklist of AC items with their completion status

### Unchecked AC Items

If any AC item remains `[ ]` (unchecked):
- This is incomplete work, not a minor issue
- Yield `improvement-request` with explicit list of unchecked items
- Producer must complete the work or escalate

### Deferral Detection

Scan producer artifacts (code, commits, yield files) for deferral language:
- Keywords: "defer", "skip", "out of scope", "later", "future", "TODO", "FIXME"
- Context: comments, commit messages, yield payloads

If deferral language detected:
- Yield `escalate-user` immediately
- Include the exact deferral text found
- User must explicitly approve or reject deferral
- No silent deferrals are permitted

## Quality Criteria

TDD execution must demonstrate:

1. **Discipline**: Tests first, minimal implementation, clean refactor
2. **Coverage**: All acceptance criteria tested
3. **Correctness**: Implementation matches spec
4. **Cleanliness**: Linter issues addressed
5. **Traceability**: Links to TASK-N maintained
6. **Completeness**: All AC items checked, no silent deferrals
