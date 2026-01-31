---
name: task-breakdown
description: Decompose architecture into implementation tasks with traceability IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Task Breakdown Skill

Transform architecture specifications into executable implementation tasks for TDD agents.

## Purpose

Read the architecture and requirements specs, then produce a structured implementation plan where each task:
- Is small enough for a single context window (Sonnet/Haiku)
- Has clear acceptance criteria traceable to requirements
- Can be implemented with TDD (test first, then code)
- Has no ambiguity requiring design decisions

Every task gets a `TASK-NNN` traceability ID linked to upstream REQ, DES, and ARCH IDs.

## Input Documents

The skill reads these files (paths from orchestrator context):
- **Requirements:** requirements.md (contains REQ- IDs)
- **Architecture:** architecture.md (contains ARCH- IDs)
- **Design:** design.md (contains DES- IDs, if design phase was not skipped)

## Phases

### 1. VALIDATE - Check Alignment

**Goal:** Ensure architecture fulfills requirements before planning.

**Steps:**
1. Read all spec documents
2. Trace each requirement to its architectural component
3. Identify gaps (requirements without architecture)
4. Identify scope creep (architecture beyond requirements)
5. Surface ambiguities or contradictions

**Output:**
- Coverage matrix (REQ → ARCH mapping)
- List of gaps and issues
- Questions for user resolution

**Do NOT proceed until alignment is confirmed or issues are resolved.**

### 2. DECOMPOSE - Break Down Work

**Goal:** Split architecture into implementation tasks sized for a single TDD session.

**Task sizing guidelines:**
- **Pure functions first** - algorithms, parsers, calculators (no dependencies)
- **Types next** - data models, interfaces (enable other work)
- **Storage layer** - database/persistence (foundation for services)
- **Services** - business logic combining pure functions + storage
- **Components** - UI elements (depend on services)
- **Pages/Integration** - wiring everything together

**Each task must have:**
- Unique TASK-NNN ID (sequential from TASK-001)
- Clear title
- Description (1-2 sentences)
- Acceptance criteria (checkboxes, verifiable)
- Files to create/modify
- Dependencies (explicit TASK-NNN IDs only)
- Traceability (REQ, DES, ARCH IDs this task addresses)
- Test properties (for property-based testing)

**Dependency format (CRITICAL):**
- Use explicit task IDs: `TASK-001, TASK-002, TASK-003`
- Use `None` for tasks with no dependencies
- **NEVER** use prose like "All previous tasks", "All component tasks", or "Phase 1"
- Dependencies must form a DAG (no cycles)

**Task size heuristics:**
- One function/method = one task (for pure functions)
- One component = one task (for UI)
- < 100 lines of implementation code
- < 200 lines of test code

### 3. ORDER - Sequence Tasks

**Goal:** Create a dependency-respecting execution order.

**Rules:**
1. Tasks with no dependencies come first
2. Pure domain logic before services
3. Types before implementations that use them
4. Services before UI that calls them
5. Parallel tasks grouped together

### 4. DOCUMENT - Write the Plan

**Goal:** Produce structured implementation plan document with traceability IDs.

**Output file:** `tasks.md` in the project directory.

**Document structure:**

```markdown
# Implementation Tasks

## Phase 1: <Phase Name>

### TASK-001: <Title>
**Status:** pending | **Attempts:** 0

**Description:** <1-2 sentences>

**Acceptance Criteria:**
- [ ] Criterion 1
- [ ] Criterion 2

**Files:** Create: `path/to/file`, Modify: `path/to/other`
**Dependencies:** None | TASK-XXX, TASK-YYY
**Traceability:** REQ-NNN, ARCH-NNN, DES-NNN

---

### TASK-002: <Title>
...

## Dependency Graph

<ASCII dependency tree>

## Parallelism Opportunities

<Groups that can execute in parallel>
```

## Traceability

### TASK ID Assignment

- Assign sequential `TASK-NNN` IDs starting from `TASK-001`
- Every implementation task gets a TASK ID
- Each TASK ID must reference at least one upstream REQ, DES, or ARCH ID

### Upstream References

When reading requirements.md, design.md, and architecture.md:
- Note all REQ-, DES-, and ARCH- IDs
- Map each task to the upstream items it implements
- Flag any upstream items that have no corresponding TASK

## Quality Checks

Before finalizing, verify:

1. **Traceability** - Every requirement maps to at least one task
2. **Independence** - Tasks can be completed without design decisions
3. **Testability** - Each task has verifiable acceptance criteria
4. **Size** - No task exceeds single-context-window scope
5. **Order** - Dependencies form a DAG (no cycles)
6. **Completeness** - Executing all tasks delivers full scope
7. **Parseable Dependencies** - Every dependency is an explicit `TASK-NNN` ID

## Interview Rules

1. **Surface ambiguities early** - Don't guess, ask
2. **Trace requirements explicitly** - Every task links to upstream IDs
3. **Respect scope** - Don't add tasks beyond what specs call for
4. **Prefer small tasks** - When in doubt, split further
5. **Document assumptions** - Capture decisions made

## Error Handling

**Architecture doesn't cover a requirement:**
- Flag as gap
- Ask user: implement now or defer?

**Requirement is ambiguous:**
- List interpretations
- Ask user to clarify
- Do NOT proceed with assumptions

**Task too large:**
- Split by: function, data type, or user flow
- Each split must be independently testable

**Circular dependency detected:**
- Identify the cycle
- Propose resolution (extract shared code, reorder, etc.)
- Ask user to confirm

## Structured Result

When the task breakdown is complete, produce a result summary for the orchestrator:

```
Status: success
Summary: Produced tasks.md with N tasks across M phases.
Files created: tasks.md
Traceability: TASK-001 through TASK-NNN assigned, all linked to REQ/DES/ARCH IDs
Findings: (any gaps, ambiguities, or cross-skill issues)
Context for next phase: (task count, dependency graph summary, parallelism opportunities, recommended execution order)
```

## Output Expectations

The plan should be:
- **Aligned** - Every task traces to upstream requirements
- **Executable** - TDD skills can pick up any unblocked task
- **Complete** - All requirements covered
- **Ordered** - Clear execution sequence with dependency graph
- **Sized** - Each task fits one context window
- **Traceable** - Every task has a TASK-NNN ID linked to REQ/DES/ARCH IDs
