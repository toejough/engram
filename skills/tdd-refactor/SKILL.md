---
name: tdd-refactor
description: Refactor implementation while keeping tests green (TDD refactor phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# TDD Refactor Phase Skill

Refactor the implementation while keeping all tests green. This is the "refactor" phase of TDD.

## Purpose

Improve code quality, naming, structure, and adherence to conventions without changing behavior. Tests must stay green throughout.

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Task ID and description
- Green phase summary (implementation files, approach, warnings about complexity)
- Architecture notes (patterns, conventions, file structure)
- Traceability IDs
- Project conventions (linter config, style guide)

## Process

1. **Run linter** - Execute the project's lint command (`mage check`, `golangci-lint run`, `npm run lint`, etc.)
2. **Assess code quality** - Review implementation from green phase for:
   - Naming clarity
   - Code organization
   - Duplication
   - Complexity
   - Convention adherence
3. **Fix linter issues by priority:**
   - **HIGH**: Complexity (cyclop, gocognit, funlen, nestif), Security (gosec), Duplication (dupl)
   - **MEDIUM**: Unused code, error handling, correctness
   - **LOW**: Ordering/formatting (funcorder) - fix last or skip
4. **Refactor for clarity** - Improve naming, extract functions, reduce duplication
5. **Run tests after each change** - Tests must stay GREEN
6. **Run linter again** - Verify all issues resolved

## Rules

1. **Tests must stay green** - Run tests after every change. If a test breaks, revert.
2. **No behavior changes** - Refactoring changes structure, not behavior
3. **No new features** - Don't add functionality even if you see opportunities
4. **Fix ALL linter issues** - Don't dismiss or suppress. If a fix is unclear, note it in findings.
5. **No blanket lint overrides** - Never add exclusions, change thresholds, or disable rules. If a linter flags something, fix the code.
6. **Extract, don't rewrite** - When moving code: COPY first, verify it works, THEN remove the original
7. **Check specs** - If implementation differs from architecture/requirements, that's a finding

## Refactoring Priorities

1. **Fix linter issues** (required)
2. **Improve naming** - Variables, functions, types should reveal intent
3. **Reduce duplication** - But only real duplication, not superficial similarity
4. **Simplify complexity** - Extract complex conditions, reduce nesting
5. **Align with conventions** - File structure, patterns match project standards

## What NOT to Do

- Do not change behavior (tests are the contract)
- Do not add features or error handling beyond what's tested
- Do not add nolint comments without asking
- Do not suppress linter rules globally
- Do not skip running tests between changes
- Do not dismiss linter findings

## Structured Result

When refactoring is complete, produce:

```
Status: success | failure | blocked
Summary: Refactored [brief description]. All N tests still passing. Linter clean.
Files modified: [list]
Tests: N total, N passing, 0 failing
Linter: clean | N remaining issues (with justification)
Refactoring performed:
  - [list of changes made]
Traceability: [REQ/DES/ARCH IDs addressed]
Findings:
  - [spec mismatches, quality concerns, suggestions for future work]
Context for next phase: [final state summary, any concerns for task audit]
```
