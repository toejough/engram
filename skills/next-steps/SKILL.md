---
name: next-steps
description: Suggest follow-up work based on completed project
context: fork
model: haiku
skills: ownership-rules
user-invocable: true
---

# Next Steps Skill

Analyze completed work and suggest follow-up actions based on open issues and project state.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML via $ARGUMENTS | completed task/phase | project dir |
| Analysis | Recent commits | Open issues.md | Remaining tasks | Learnings |
| Output | Yield TOML with prioritized next steps |

## Workflow

### 1. Gather Context

1. Read completed task or phase information from input
2. Scan `docs/issues.md` for open ISSUE-NNN entries
3. Check `docs/tasks.md` for remaining TASK-NNN items
4. Review recent decisions and learnings

### 2. Analyze

1. Identify follow-up work suggested by completed task
2. Cross-reference with open issues to find related work
3. Prioritize based on:
   - Dependencies (blocked items now unblocked)
   - User value (high-impact features)
   - Technical debt (cleanup from recent changes)

### 3. Produce

Generate yield TOML with prioritized recommendations.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol.

### Yield Types

| Type | When |
|------|------|
| `complete` | Recommendations generated |
| `need-context` | Need additional project files |
| `blocked` | Cannot analyze (missing artifacts) |

### Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
summary = "3 recommended next steps based on TASK-15 completion"

[[payload.recommendations]]
priority = 1
type = "issue"
id = "ISSUE-003"
title = "Add caching layer"
reason = "Now unblocked by completed authentication work"

[[payload.recommendations]]
priority = 2
type = "task"
id = "TASK-18"
title = "Write integration tests"
reason = "New feature needs test coverage"

[[payload.recommendations]]
priority = 3
type = "improvement"
title = "Refactor error handling"
reason = "Pattern emerged during TASK-15 that could be generalized"

[[payload.learnings]]
content = "Authentication module ready for caching integration"

[context]
phase = "next-steps"
completed_task = "TASK-15"
```

## Result Format

`result.toml`: `[status]`, `[[recommendations]]`, `[[learnings]]`

## Full Documentation

`projctl skills docs --skillname next-steps` or see SKILL-full.md
