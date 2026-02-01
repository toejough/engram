---
name: tdd-green
description: Write minimal implementation to make tests pass (TDD green phase)
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# TDD Green Phase

Write minimal implementation to make failing tests pass.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML via $ARGUMENTS | test file locations | architecture notes |
| Process | Read tests | Write minimal impl | Run tests (ALL PASS) | Debug if needed | Verify no regressions |
| Rules | MINIMAL code only | NO refactoring | ALL tests pass | Follow arch patterns | NO new tests |

## Debugging Heuristics

| Issue | Check |
|-------|-------|
| Struct field changes | Grep for copy/clone logic - new fields stay zero |
| Multiple code paths | Similar operations have multiple paths - fix all |
| Accumulated flags | Flags that only turn on need phase/state checks |
| Still failing | Trace what happens AFTER the code runs |

## Failure Hints

| Symptom | Fix |
|---------|-----|
| Tests still fail | Read failure message carefully - match test expectation |
| Existing tests break | Fix them - don't dismiss as "pre-existing" |
| Stuck after 3 attempts | Report as blocked with detailed findings |

## Result Format

`result.toml` (see shared/RESULT.md):
- `[status]` success=bool
- `[outputs]` files_modified=[]
- `[[decisions]]` context, choice, reason
- `[[learnings]]` content

## Full Documentation

`projctl skills docs --skillname tdd-green` or see SKILL-full.md
