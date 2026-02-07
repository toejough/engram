---
name: commit
description: Create a well-formatted git commit following project conventions
context: inherit
model: haiku
skills: ownership-rules
user-invocable: true
---

# Commit Skill

Stage and commit changes with properly formatted message.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context from spawn prompt: phase, task ID, summary |
| Process | Check VCS type, check state, review style, stage specific files, compose message, commit, verify |
| Rules | Trailer is `AI-Used: [claude]`, NEVER amend pushed, stage specific files, NO dangerous commands |

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

## Reporting Results

On success, send completion message to lead:
```
Complete: commit created

Commit hash: abc123f
Files modified: internal/foo.go, internal/foo_test.go
Message: feat(foo): implement feature
```

On failure, send error message to lead:
```
Error: commit failed

Error: Pre-commit hook failed
Details: golangci-lint found issues
Recoverable: yes
```

## Full Documentation

`projctl skills docs --skillname commit` or see SKILL-full.md
