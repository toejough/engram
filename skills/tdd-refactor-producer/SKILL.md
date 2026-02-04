---
name: tdd-refactor-producer
description: Refactor implementation while keeping tests green (TDD refactor phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: tdd-refactor
---

# TDD Refactor Producer

Improve code quality while keeping all tests green. This is the "refactor" phase of TDD.

## Overview

| Aspect | Details |
|--------|---------|
| Pattern | GATHER -> SYNTHESIZE -> PRODUCE |
| Input | Context TOML with green phase results, implementation files |
| Output | Refactored code with yield to [YIELD.md](../shared/YIELD.md) |
| Key Rule | All tests must still pass after every refactor |

---

## GATHER Phase

Collect information needed for refactoring:

1. Read context from `[inputs]` section:
   - Task ID and implementation files from green phase
   - Architecture notes and conventions
   - Linter configuration
2. Run tests to establish baseline (must be green)
3. Run linter to identify issues
4. If missing context (e.g., project conventions), yield `need-context`

```toml
[yield]
type = "need-context"

[[payload.queries]]
type = "file"
path = ".golangci.yml"

[[payload.queries]]
type = "semantic"
question = "What are the project's naming conventions?"

[context]
phase = "tdd-refactor"
subphase = "GATHER"
```

---

## SYNTHESIZE Phase

Analyze gathered information to plan refactoring:

1. Categorize linter issues by priority:
   - **HIGH**: Complexity (cyclop, gocognit, funlen, nestif), Security (gosec), Duplication (dupl)
   - **MEDIUM**: Unused code, error handling, correctness
   - **LOW**: Ordering/formatting (funcorder)
2. Identify code quality improvements:
   - Naming clarity
   - Code organization
   - Duplication removal
   - Complexity reduction
3. Check for spec mismatches - report as blockers

If blocked (e.g., spec mismatch found), yield `blocked`:

```toml
[yield]
type = "blocked"

[payload]
blocker = "Implementation differs from architecture spec"
details = "Function X uses pattern Y but ARCH-3 specifies pattern Z"
suggested_resolution = "Clarify with architecture phase"

[context]
phase = "tdd-refactor"
subphase = "SYNTHESIZE"
```

---

## PRODUCE Phase

Execute refactoring while maintaining tests green:

### Process

1. Fix linter issues by priority order
2. After EACH change, run tests - they must pass
3. If tests break, REVERT immediately (refactoring doesn't change behavior)
4. Improve naming, extract functions, reduce duplication
5. Run linter again to verify clean
6. Yield `complete` with results

### Rules

- **Tests must stay green** - Run after every change
- **No behavior changes** - Refactoring changes structure only
- **No new features** - Don't add functionality
- **No blanket lint overrides** - Fix the code, don't suppress rules
- **Extract, don't rewrite** - COPY first, verify, THEN remove original

### Complete Yield

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
artifact = "refactored implementation"
files_modified = ["internal/foo/foo.go", "internal/foo/bar.go"]

[[payload.decisions]]
context = "Complexity reduction"
choice = "Extracted helper function"
reason = "Reduced cyclomatic complexity from 15 to 8"
alternatives = ["Inline conditionals", "Table-driven approach"]

[[payload.learnings]]
content = "Project prefers explicit error handling over wrapped errors"

[context]
phase = "tdd-refactor"
subphase = "complete"
tests_passing = true
linter_clean = true
```

---

## Failure Hints

| Symptom | Fix |
|---------|-----|
| Tests break after change | REVERT immediately - behavior changed |
| Linter issue unclear | Note in findings, don't suppress |
| Spec mismatch found | Report as blocker, don't proceed |

---

## Refactoring Documentation

After doc tests pass, refactor for clarity and organization while keeping tests green.

### Documentation Best Practices

| Practice | Description |
|----------|-------------|
| Progressive disclosure | Most important info first, details later |
| Clarity and conciseness | Remove filler words, be direct |
| Consistent structure | Same heading hierarchy, same patterns |
| Remove redundancy | Don't repeat information across sections |
| Doc-type-specific | READMEs need quick start; API docs need exhaustive detail |

### Refactoring Checklist

- [ ] Tests still pass after each change
- [ ] Most important content is near the top
- [ ] No redundant sections saying the same thing
- [ ] Consistent heading levels (H2 for main sections, H3 for subsections)
- [ ] Code examples are minimal and runnable
- [ ] Links work and point to correct locations

### Key Rule

**Tests must still pass after refactoring.** Re-run your doc tests after every structural change. If a test breaks, you've lost essential content - revert and try again.

---

## Reference

- Pattern: [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md)
- Yield protocol: [YIELD.md](../shared/YIELD.md)
