---
name: tdd-green-producer
description: Write minimal implementation to make tests pass (TDD green phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: tdd-green
---

# TDD Green Producer

Write minimal implementation code to make failing tests pass.

## Workflow: GATHER -> SYNTHESIZE -> PRODUCE

This skill follows the producer pattern from [PRODUCER-TEMPLATE](../shared/PRODUCER-TEMPLATE.md).

### GATHER Phase

1. Read context from `[inputs]` section for:
   - Test file locations
   - Architecture notes
   - TASK-N being implemented
2. Check for `[query_results]` (resuming after need-context)
3. If missing test files or architecture context:
   - Yield `need-context` with file queries
4. Proceed to SYNTHESIZE when test expectations are clear

### SYNTHESIZE Phase

1. Analyze failing tests to understand expected behavior
2. Identify minimal code changes needed
3. Check existing patterns in codebase
4. If blocked by missing information, yield `blocked`
5. Prepare implementation plan

### PRODUCE Phase

1. Write minimal implementation to pass tests
2. **All targeted tests must pass** - no exceptions
3. Run full test suite to verify no regressions
4. Yield `complete` with files modified

## Rules

| Rule | Rationale |
|------|-----------|
| MINIMAL code only | Don't over-engineer |
| NO refactoring | That's tdd-refactor's job |
| ALL tests must pass | Non-negotiable exit criteria |
| Follow arch patterns | Consistency with codebase |
| NO new tests | That's tdd-red's job |

## Making Documentation Tests Pass

When doc tests fail, edit the documentation minimally to make them pass.

### Principles

1. **Add only what's needed** - Don't over-document
2. **Match the test's expectation** - If test checks for "## Yield Types", add that exact heading
3. **Preserve existing content** - Don't remove working content while adding new

### Examples

**Example 1: Word matching test fails**

Test: `grep -q "## Acceptance Criteria" SKILL.md`
Failure: Section doesn't exist

Minimal fix - add exactly the section the test expects:
```markdown
## Acceptance Criteria

[Add criteria here]
```

**Example 2: Semantic test fails**

Test: README must explain "how to install the tool" (similarity >= 0.7)
Failure: Score is 0.45 - concept not conveyed strongly enough

Minimal fix - add installation section with clear, direct language:
```markdown
## Installation

To install the tool, run:
\`\`\`bash
go install github.com/example/tool@latest
\`\`\`
```

Don't add extra sections or elaborate beyond what's needed to pass the semantic match.

## Debugging Heuristics

| Issue | Check |
|-------|-------|
| Struct field changes | Grep for copy/clone logic - new fields stay zero |
| Multiple code paths | Similar operations have multiple paths - fix all |
| Accumulated flags | Flags that only turn on need phase/state checks |
| Still failing | Trace what happens AFTER the code runs |

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full specification.

### Complete Yield

When all tests pass:

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
artifact = "internal/foo/bar.go"
files_modified = ["internal/foo/bar.go", "internal/foo/baz.go"]
tests_passing = ["TestFoo", "TestBar"]

[[payload.decisions]]
context = "Implementation approach"
choice = "Used existing pattern from pkg/util"
reason = "Consistency with codebase"

[context]
phase = "tdd-green"
task = "TASK-5"
subphase = "complete"
```

### Need-Context Yield

When missing information:

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:35:00Z

[[payload.queries]]
type = "file"
path = "internal/foo/bar_test.go"

[[payload.queries]]
type = "semantic"
question = "How is error handling implemented in this package?"

[context]
phase = "tdd-green"
task = "TASK-5"
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
blocker = "Test expects behavior not defined in architecture"
details = "TestFoo expects caching but ARCH-3 doesn't mention it"
suggested_resolution = "Clarify caching requirements in architecture"

[context]
phase = "tdd-green"
task = "TASK-5"
awaiting = "blocker-resolution"
```

## Failure Recovery

| Symptom | Action |
|---------|--------|
| Tests still fail after implementation | Re-read test expectations carefully |
| Existing tests break | Fix them - never dismiss as "pre-existing" |
| Stuck after 3 attempts | Yield `blocked` with detailed findings |
| Architecture unclear | Yield `need-context` for semantic exploration |
