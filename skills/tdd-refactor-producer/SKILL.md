---
name: tdd-refactor-producer
description: |
  Core: Improves code quality through refactoring while maintaining all passing tests (TDD refactor phase).
  Triggers: refactor code, clean up implementation, tdd refactor phase, improve code quality, fix lint issues.
  Domains: refactoring, code-quality, tdd, test-driven-development, linting, code-cleanup.
  Anti-patterns: NOT for adding new features, NOT for changing test behavior, NOT for breaking existing tests.
  Related: tdd-green-producer (previous phase), qa (validates refactor quality), commit (creates refactor commit).
context: inherit
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
| Input | Context from spawn prompt: green phase results, implementation files |
| Key Rule | All tests must still pass after every refactor |

---

## Workflow Context

- **Phase**: `tdd_refactor_produce` (states.tdd_refactor_produce)
- **Upstream**: Green QA approval (`tdd_green_decide`), or retry from refactor QA (`tdd_refactor_decide`)
- **Downstream**: `tdd_refactor_qa` → `tdd_refactor_decide` → retry, escalate, or commit (`tdd_commit`)
- **Model**: sonnet (default_model in workflows.toml)

This skill refactors implementation while keeping tests green (refactor) in the TDD loop.

---

## GATHER Phase

Collect information needed for refactoring:

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode):
   - Task ID and implementation files from green phase
   - Architecture notes and conventions
   - Linter configuration
2. Query memory for relevant patterns:
   - `projctl memory query "refactoring patterns for <domain>"`
   - `projctl memory query "code quality learnings for <feature-area>"`
   If memory is unavailable, proceed gracefully without blocking
3. Run tests to establish baseline (must be green)
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
6. Send a message to team-lead with results

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

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Read project docs | `Read`, `Glob`, `Grep` tools directly |
| Run tests/linter | `Bash` |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

On completion, send a message to the team lead with:
- Artifact paths (refactored files)
- Lint results (before/after counts)
- Test results (all still passing)
- Files modified
- Key decisions made

---

## Reference

- Pattern: [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md)

---

## Contract

```yaml
contract:
  outputs:
    - path: "<refactored-files>"
      id_format: "N/A"

  traces_to:
    - "docs/tasks.md"
    - "<test-file>"

  checks:
    - id: "CHECK-001"
      description: "All tests still pass after refactoring"
      severity: error

    - id: "CHECK-002"
      description: "Behavior unchanged (refactoring only changes structure)"
      severity: error

    - id: "CHECK-003"
      description: "Linter issues reduced or eliminated"
      severity: error

    - id: "CHECK-004"
      description: "No new linter issues introduced"
      severity: error

    - id: "CHECK-005"
      description: "No blanket lint suppressions added"
      severity: error

    - id: "CHECK-006"
      description: "Code readability improved"
      severity: warning

    - id: "CHECK-007"
      description: "Complexity reduced where possible"
      severity: warning

    - id: "CHECK-008"
      description: "Duplication removed where possible"
      severity: warning
```

---

## Lessons Learned

**Extract before refactoring**: COPY working code first, verify it works, THEN refactor. Never rewrite during extraction.

**Multiple code paths**: Similar operations often have multiple paths. When fixing one, ask: "Is there another code path doing something similar?"

**Accumulated boolean flags**: Flags that only turn on need phase/state checks. Use `if hasX && phase == X`, not just `if hasX`.
