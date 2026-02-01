---
name: task-audit
description: Validate task completion against acceptance criteria with TDD discipline checks
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Task Audit

Validate completed task meets acceptance criteria and followed TDD discipline.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML via $ARGUMENTS | task ID | TDD phase summaries | project dir |
| Steps | Load task | Verify files exist | Run tests (ALL PASS) | TDD discipline check | AC verification |
| TDD Checks | Test weakening | Linter gaming | Test quality | Property tests implemented |

## TDD Red Flags

| Violation | Examples |
|-----------|----------|
| Test Weakening | Removed tests | Weakened assertions | Added .skip | Changed expected values |
| Linter Gaming | New nolint | Config changes | Threshold changes | Exclusion rules |
| Quality Standards | Go: use `package foo_test`, rapid | TS: .test.ts, property tests |

## AC Verification

| Check | Action |
|-------|--------|
| Create files | Verify all exist |
| Modify files | Check git diff shows changes |
| Tests pass | Run ALL tests - no exceptions |
| Properties | Property-based test for each specified property |

## Failure Hints

| Symptom | Fix |
|---------|-----|
| Tests fail during audit | FAIL - fix tests, don't dismiss as "pre-existing" |
| Test modified to pass | FAIL - test weakening detected |
| Linter suppressed | FAIL - linter gaming detected |

## Result Format

`result.toml` (see shared/RESULT.md):
- `[status]` success=bool
- `[[decisions]]` context, choice, reason
- `[[learnings]]` content

## Full Documentation

`projctl skills docs --skillname task-audit` or see SKILL-full.md
