---
name: tdd-refactor
description: Refactor implementation while keeping tests green (TDD refactor phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# TDD Refactor Phase

Improve code quality while keeping all tests green.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML via $ARGUMENTS | green phase summary | arch notes |
| Process | Run linter | Fix issues by priority | Refactor for clarity | Run tests after each change | Linter again |
| Rules | Tests STAY GREEN | NO behavior changes | NO new features | Fix ALL linter issues | NO blanket overrides |

## Linter Priority

| Priority | Categories |
|----------|------------|
| HIGH | Complexity (cyclop, gocognit, funlen, nestif) | Security (gosec) | Duplication (dupl) |
| MEDIUM | Unused code | Error handling | Correctness |
| LOW | Ordering/formatting (funcorder) - fix last or skip |

## Failure Hints

| Symptom | Fix |
|---------|-----|
| Tests break after change | REVERT immediately - refactoring doesn't change behavior |
| Linter unclear | Note in findings, don't suppress |
| Spec mismatch found | Report as finding - that's a blocker |

## Output Format

`result.toml` (see shared/RESULT.md):
- `[status]` success=bool
- `[outputs]` files_modified=[]
- `[[decisions]]` context, choice, reason
- `[[learnings]]` content

## Full Documentation

`projctl skills docs --skillname tdd-refactor` or see SKILL-full.md
