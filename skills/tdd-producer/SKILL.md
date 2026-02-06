---
name: tdd-producer
description: Composite producer that orchestrates the full TDD RED/GREEN/REFACTOR cycle
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: tdd
---

# TDD Producer (Composite)

Orchestrates the complete TDD cycle by running nested pair loops for RED, GREEN, and REFACTOR phases. This is a composite producer that coordinates other skills rather than producing artifacts directly.

## Overview

| Aspect | Details |
|--------|---------|
| Pattern | GATHER -> SYNTHESIZE -> PRODUCE (composite variant) |
| Input | Context from spawn prompt: task ID, acceptance criteria, architecture notes |
| Nature | Composite orchestrator - runs nested pair loops |

---

## Nested Pair Loops

This skill runs three sequential pair loops, each with producer/QA iteration:

```
TASK-N
  |
  v
+------------------+
|  RED PAIR LOOP   |  tdd-red-producer <-> tdd-red-qa
+------------------+
  |
  v (tests written, failing correctly)
+------------------+
| GREEN PAIR LOOP  |  tdd-green-producer <-> tdd-green-qa
+------------------+
  |
  v (tests passing, minimal implementation)
+------------------+
| REFACTOR PAIR LOOP | tdd-refactor-producer <-> tdd-refactor-qa
+------------------+
  |
  v
yield complete
```

Each pair loop allows iteration with improvement requests until QA approves.

---

## Workflow: GATHER -> SYNTHESIZE -> PRODUCE

### GATHER Phase

Collect information needed to orchestrate the TDD cycle:

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode):
   - Task ID and acceptance criteria
   - Architecture notes and constraints
   - Test file locations and conventions
2. Check for `[query_results]` (resuming after need-context)
3. If missing critical information:
   - Yield `need-context` with queries for missing data
4. Proceed to SYNTHESIZE when ready to orchestrate

### SYNTHESIZE Phase

Prepare for nested loop execution:

1. Verify acceptance criteria are testable
2. Identify test file and implementation locations
3. Plan iteration limits for each nested loop
4. If blocked (ambiguous requirements), yield `blocked`
5. Prepare loop execution order

### PRODUCE Phase

Execute the three nested pair loops in sequence:

#### 1. RED Pair Loop

Run failing test creation with QA validation:

```
tdd-red-producer -> creates failing tests
       |
       v
tdd-red-qa -> reviews
       |
       +-> approved: proceed to GREEN
       +-> improvement-request: loop back to red-producer (max 3 iterations)
       +-> escalate-phase: propagate escalation
       +-> escalate-user: propagate escalation
```

**Exit criteria**: Tests exist for all acceptance criteria and fail correctly.

#### 2. GREEN Pair Loop

Run minimal implementation with QA validation:

```
tdd-green-producer -> writes minimal implementation
       |
       v
tdd-green-qa -> reviews
       |
       +-> approved: proceed to REFACTOR
       +-> improvement-request: loop back to green-producer (max 3 iterations)
       +-> escalate-phase: propagate escalation
       +-> escalate-user: propagate escalation
```

**Exit criteria**: All tests pass with minimal implementation.

#### 3. REFACTOR Pair Loop

Run code quality improvement with QA validation:

```
tdd-refactor-producer -> improves code quality
       |
       v
tdd-refactor-qa -> reviews
       |
       +-> approved: TDD cycle complete
       +-> improvement-request: loop back to refactor-producer (max 3 iterations)
       +-> escalate-phase: propagate escalation
       +-> escalate-user: propagate escalation
```

**Exit criteria**: Tests still pass, linter clean, code quality improved.

---

## Iteration Handling

Each nested pair loop handles improvement requests internally:

```toml
[context]
nested_loop = "red"  # red | green | refactor
iteration = 1
max_iterations = 3
```

If a nested loop exhausts iterations:
1. Escalate to user if issues remain unresolved
2. Or proceed with caveats noted

If any nested loop yields `escalate-phase` or `escalate-user`:
1. Propagate the escalation immediately
2. Do not proceed to next loop until resolved

---

## Yield Protocol

### Complete Yield

When all three pair loops complete successfully:

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
task = "TASK-5"
artifact = "TDD cycle complete"
files_modified = ["internal/foo/foo_test.go", "internal/foo/foo.go"]

[payload.cycle_summary]
red = { tests_created = 5, iterations = 1, status = "approved" }
green = { tests_passing = 5, iterations = 2, status = "approved" }
refactor = { lint_before = 8, lint_after = 0, iterations = 1, status = "approved" }

[[payload.decisions]]
context = "TDD cycle orchestration"
choice = "Standard RED -> GREEN -> REFACTOR sequence"
reason = "No escalations required during cycle"

[context]
phase = "tdd"
subphase = "complete"
all_loops_passed = true
```

### Need-Context Yield

When missing information before starting loops:

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:35:00Z

[[payload.queries]]
type = "file"
path = "docs/tasks/TASK-5.md"

[[payload.queries]]
type = "semantic"
question = "What are the project's testing conventions?"

[context]
phase = "tdd"
subphase = "GATHER"
awaiting = "context-results"
```

### Blocked Yield

When cannot proceed:

```toml
[yield]
type = "blocked"
timestamp = 2026-02-02T10:40:00Z

[payload]
blocker = "Acceptance criteria not testable"
details = "AC-3 says 'should be performant' with no measurable threshold"
suggested_resolution = "Define specific performance thresholds in task breakdown"

[context]
phase = "tdd"
subphase = "SYNTHESIZE"
awaiting = "blocker-resolution"
```

### Escalation Propagation

When a nested loop escalates, propagate immediately:

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T10:45:00Z

[payload.escalation]
from_phase = "tdd"
to_phase = "breakdown"
reason = "gap"
originated_in = "red-pair-loop"

[payload.issue]
summary = "Acceptance criteria incomplete"
context = "During RED phase, discovered AC-2 is ambiguous"

[context]
phase = "tdd"
subphase = "red-loop"
escalating = true
```

---

## Error Recovery

| Situation | Action |
|-----------|--------|
| Red loop fails after max iterations | Escalate to user with findings |
| Green loop can't make tests pass | Check if tests are correct, escalate to red if needed |
| Refactor breaks tests | Revert and try different approach |
| Nested escalation received | Propagate immediately, pause cycle |

---

## Communication

### Team Mode (preferred)

In team mode, the project lead spawns the tdd-producer as a teammate. The tdd-producer runs the full RED -> GREEN -> REFACTOR cycle internally (it does not spawn sub-teammates for each phase). It coordinates all three phases itself, using the sub-producer skill docs as guidance for each phase.

| Action | Tool |
|--------|------|
| Read project docs | `Read`, `Glob`, `Grep` tools directly |
| Run tests/linter | `Bash` |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

On completion, send a message to the team lead with:
- Artifact paths (test files and implementation files)
- Cycle summary (red/green/refactor results)
- Files modified
- Key decisions made

---

## Reference

- Producer template: [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md)
- RED producer: [tdd-red-producer](../tdd-red-producer/SKILL.md)
- RED QA: [tdd-red-qa](../tdd-red-qa/SKILL.md)
- GREEN producer: [tdd-green-producer](../tdd-green-producer/SKILL.md)
- GREEN QA: [tdd-green-qa](../tdd-green-qa/SKILL.md)
- REFACTOR producer: [tdd-refactor-producer](../tdd-refactor-producer/SKILL.md)
- REFACTOR QA: [tdd-refactor-qa](../tdd-refactor-qa/SKILL.md)

---

## Contract

```yaml
contract:
  outputs:
    - path: "<test-files>"
      id_format: "N/A"
    - path: "<implementation-files>"
      id_format: "N/A"

  traces_to:
    - "docs/tasks.md"

  checks:
    - id: "CHECK-001"
      description: "Tests written before implementation (RED phase first)"
      severity: error

    - id: "CHECK-002"
      description: "Tests failed initially for correct reasons"
      severity: error

    - id: "CHECK-003"
      description: "All tests pass after implementation (GREEN phase)"
      severity: error

    - id: "CHECK-004"
      description: "Refactoring preserved behavior (tests still pass)"
      severity: error

    - id: "CHECK-005"
      description: "Each acceptance criterion has corresponding test(s)"
      severity: error

    - id: "CHECK-006"
      description: "All AC items are checked [x] (none remain unchecked)"
      severity: error

    - id: "CHECK-007"
      description: "No deferral language in producer artifacts"
      severity: error

    - id: "CHECK-008"
      description: "Linter issues addressed during REFACTOR"
      severity: error

    - id: "CHECK-009"
      description: "Minimal implementation (GREEN phase is minimal)"
      severity: warning

    - id: "CHECK-010"
      description: "Visual evidence for tasks with [visual] marker"
      severity: warning
```
