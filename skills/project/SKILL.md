---
name: project
description: State-machine-driven project orchestrator
user-invocable: true
---

# Project Orchestrator

## Critical Rules

| Rule | Details |
|------|---------|
| State | `projctl state transition` - NEVER modify state.toml |
| Log | `projctl log write` for logging |
| Handoffs | `projctl context write/read` |
| TDD | Red→commit→green→commit→refactor→commit |
| Audits | Loop until zero defects |
| Continue | If `state next`=continue, proceed immediately |
| Dispatch | ALL code work via Skill/Task tool |
| Context | At 80% warn, 90% compact |

## Control Loop

| Step | Type | Action |
|------|------|--------|
| 1 | [D] | `projctl state get` |
| 2 | [D] | `projctl state transition` |
| 3 | [D] | `projctl map --cached` |
| 4 | [A] | Dispatch skill |
| 5 | [D] | `projctl context read --result` |
| 6 | [D] | `projctl state next` |
| 7 | [A] | If continue: loop |

## Stop Conditions

| Reason | Action |
|--------|--------|
| all_complete | Present summary |
| escalation_pending | Present to user |
| validation_failed | Run repair loop |

## End-of-Command

```bash
projctl integrate features --dir .
projctl trace repair --dir .
projctl trace validate --dir .
```

## Result Format

Orchestrator skill - reads results from dispatched skills via `projctl context read --result`. Does not produce its own result.toml.

## Full Docs

`projctl skills docs --skillname project`
