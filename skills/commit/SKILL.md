---
name: commit
description: Create a well-formatted git commit following project conventions
context: fork
model: haiku
skills: ownership-rules
user-invocable: true
---

# Commit Skill

Stage and commit changes with properly formatted message.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML via $ARGUMENTS | phase | task ID | summary |
| Process | Check VCS type | Check state | Review style | Stage specific files | Compose message | Commit | Verify |
| Rules | Trailer is `AI-Used: [claude]` | NEVER amend pushed | Stage specific files | NO dangerous commands |

## Message Templates

| Phase | Format |
|-------|--------|
| TDD Red | `test(scope): add tests for <feature>` |
| TDD Green | `feat(scope): implement <feature>` |
| TDD Refactor | `refactor(scope): <cleanup>` |
| Other | `<type>(scope): <description>` |

## Failure Hints

| Symptom | Fix |
|---------|-----|
| Need to amend | Check `git status` for "ahead of" - if NOT ahead, CREATE NEW commit |
| Mixed concerns | Separate functional changes from lint/style fixes |
| VCS error | Check for `.jj` dir - use jj commands if present |

## Result Format

`result.toml` (see shared/RESULT.md):
- `[status]` success=bool
- `[outputs]` files_modified=[]
- `[[decisions]]` context, choice, reason

## Full Documentation

`projctl skills docs --skillname commit` or see SKILL-full.md
