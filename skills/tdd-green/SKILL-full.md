---
name: tdd-green
description: Write minimal implementation to make tests pass (TDD green phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# TDD Green Phase Skill

Write the minimal implementation to make failing tests pass. This is the "green" phase of TDD.

## Purpose

Make all tests pass with the simplest code that works. Do not refactor, do not optimize, do not clean up -- just make the tests green.

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Task ID and description
- Red phase summary (test file locations, test descriptions, stub info)
- Architecture notes (patterns to follow, DI approach, file structure)
- Traceability IDs
- Project conventions

## Process

1. **Understand what tests expect** - Read all test files from the red phase
2. **Write minimal implementation** - Create/modify implementation files
3. **Run tests** - Confirm they ALL PASS
4. **Debug if needed** - Use these heuristics:
   - **Struct field changes?** Grep for copy/clone logic - new fields stay zero if not added
   - **Multiple code paths?** Similar operations often have multiple paths - fix all of them
   - **Accumulated flags?** Flags that only turn on need phase/state checks
   - **Trace the journey** - What happens AFTER this code runs?
5. **Verify no regressions** - Run the full test suite, not just new tests

## Rules

1. **Minimal implementation only** - Write the least code needed to pass tests
2. **No refactoring** - That's the next phase. Don't clean up, extract, or reorganize.
3. **All tests must pass** - Both new tests from red phase and existing tests
4. **Follow architecture patterns** - Use the DI approach, file structure, and conventions specified in context
5. **No new tests** - The red phase defined what to test. If you discover missing test coverage, note it in findings.
6. **Fix ALL failures** - If existing tests break, fix them. Don't dismiss as "pre-existing."

## What NOT to Do

- Do not refactor or reorganize code
- Do not add features beyond what tests require
- Do not write new tests (note missing coverage in findings instead)
- Do not optimize for performance
- Do not add error handling beyond what tests verify
- Do not dismiss test failures as "pre-existing" or "unrelated"

## Debugging Strategy

If tests don't pass after implementation:

1. Read the test failure message carefully
2. Check if the implementation matches what the test expects (not what you think it should do)
3. Check for off-by-one errors, nil/null handling, type mismatches
4. If stuck after 3 attempts, report as blocked with detailed findings

## Structured Result

When all tests pass, produce:

```
Status: success | failure | blocked
Summary: All N tests passing. Implemented [brief description of approach].
Files created: [list]
Files modified: [list]
Tests: N total, N passing, 0 failing
Implementation approach: [1-2 sentences]
Traceability: [REQ/DES/ARCH IDs addressed]
Findings:
  - [any issues discovered, missing test coverage, etc.]
Context for next phase: [implementation summary, files touched, areas that may benefit from refactoring, warnings about complexity]
```

## Result Format

See [shared/RESULT.md](../shared/RESULT.md) for the complete schema.

```toml
[status]
success = true

[outputs]
files_modified = ["internal/foo/foo.go"]

[[decisions]]
context = "Implementation approach"
choice = "Minimal implementation"
reason = "Only what's needed to pass tests"
alternatives = []

[[learnings]]
content = "Found existing helper function that simplified implementation"
```
