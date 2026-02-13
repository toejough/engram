---
name: next-steps
description: |
  Core: Analyzes completed project state and suggests prioritized follow-up work based on open issues and remaining tasks.
  Triggers: suggest next steps, recommend follow-up, prioritize remaining work, post-project planning.
  Domains: project-planning, follow-up, prioritization, backlog-management, recommendations.
  Anti-patterns: NOT for implementation, NOT for current project work, runs only after project completion.
  Related: evaluation-producer (precedes next-steps), project (receives recommendations).
context: inherit
model: haiku
skills: ownership-rules
user-invocable: true
---

# Next Steps Skill

Analyze completed work and suggest follow-up actions based on open issues and project state.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context from spawn prompt: completed task/phase, project dir |
| Analysis | Recent commits, open issues.md, remaining tasks, learnings |
| Output | Prioritized next steps via `SendMessage` to lead |

## Workflow Context

- **Phase**: `next_steps` (states.next_steps)
- **Upstream**: Issue update complete (`issue_update`)
- **Downstream**: `complete` (final workflow state)
- **Model**: haiku (default_model in workflows.toml)

This skill runs at the end of the workflow to suggest follow-up work after project completion.

---

## Workflow

### 1. Gather Context

1. Read completed task or phase information from spawn prompt
2. Scan `docs/issues.md` for open ISSUE-NNN entries
3. Check `docs/tasks.md` for remaining TASK-NNN items
4. Review recent decisions and learnings
5. Query memory for relevant context:
   - `projctl memory query "past project recommendations"`
   - `projctl memory query "follow-up patterns"`

   Memory queries are optional - graceful degradation if memory is unavailable

### 2. Analyze

1. Identify follow-up work suggested by completed task
2. Cross-reference with open issues to find related work
3. Prioritize based on:
   - Dependencies (blocked items now unblocked)
   - User value (high-impact features)
   - Technical debt (cleanup from recent changes)

### 3. Produce

Send completion message to lead with prioritized recommendations.

### Example Completion Message

```
Complete: next-steps analysis

Summary: 3 recommended next steps based on TASK-15 completion

Recommendations:
1. [ISSUE-3] Add caching layer - now unblocked by completed authentication work
2. [TASK-18] Write integration tests - new feature needs test coverage
3. [improvement] Refactor error handling - pattern emerged during TASK-15

Learnings: Authentication module ready for caching integration
```

## Result Format

Completion message to lead with: summary, prioritized recommendations, learnings.

## Full Documentation

`projctl skills docs --skillname next-steps` or see SKILL-full.md
