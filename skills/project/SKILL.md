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

## Correction Detection

Before each loop iteration, scan user messages for correction patterns:
- "that's wrong", "no, do X", "I said X not Y", "remember that"
- "actually", "not X, Y", "you forgot", "I already told you"

When detected: `projctl corrections log --dir . --message "PATTERN" --context "TASK/PHASE"`

## Control Loop

| Step | Type | Action |
|------|------|--------|
| 0 | [A] | Detect corrections in user input |
| 1 | [D] | `projctl state get` |
| 2 | [D] | `projctl state transition` |
| 3 | [D] | `projctl map --cached` |
| 4 | [A] | Dispatch skill |
| 5 | [D] | `projctl context read --result` |
| 6 | [D] | `projctl corrections count --session` |
| 7 | [D] | `projctl state next` |
| 8 | [A] | If corrections >= 2: dispatch /meta-audit |
| 9 | [A] | If continue: loop |

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
