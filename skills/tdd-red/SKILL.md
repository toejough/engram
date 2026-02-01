---
name: tdd-red
description: Write failing tests for a task (TDD red phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# TDD Red Phase

Write failing tests that specify expected behavior before implementation.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML via $ARGUMENTS | task ID | acceptance criteria |
| Process | Read context | Map criteria to tests | Write tests | Run (must FAIL) | Verify correct failures |
| Rules | Tests ONLY (no impl) | MUST fail | Cover ALL criteria | Property + example tests | Test BEHAVIOR not structure |
| Go | `package foo_test` | gomega + rapid | blackbox only |
| TS | `.test.ts` | vitest + fast-check | behavior tests |

## Failure Hints

| Symptom | Fix |
|---------|-----|
| Tests pass unexpectedly | Feature exists or tests are wrong - stop and report |
| Build fails | Check imports; minimal stubs that panic/throw only |
| Missing coverage | Map each acceptance criterion to at least one test |
| Structural only | Test behavior chain: action -> event -> handler -> state -> UI |

## Result Format

`result.toml` (see shared/RESULT.md):
- `[status]` success=bool
- `[outputs]` files_modified=[]
- `[[decisions]]` context, choice, reason
- `[[learnings]]` content

## Full Documentation

`projctl skills docs --skillname tdd-red` or see SKILL-full.md
