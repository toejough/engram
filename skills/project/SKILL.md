---
name: project
description: State-machine-driven project orchestrator
user-invocable: true
---

# Project Orchestrator

Manage projects through structured phases via state machine.

## Critical Rules

| Rule | Details |
|------|---------|
| State transitions | Use `projctl state transition` - NEVER modify state.toml directly |
| Logging | Use `projctl log write` for all logging |
| Skill handoffs | Use `projctl context write/read` for context/result |
| TDD commits | Red → commit → green → commit → refactor → commit |
| Never skip audits | Audit loop runs until zero defects |
| Never ask to continue | If `projctl state next` returns `continue`, proceed immediately |
| Sub-agent dispatch | ALL skill work via Task tool - orchestrator never reads/writes code |

## Sub-Agent Mandate

**NEVER** use Read/Edit/Write tools directly for code files. **ALL** skill work dispatched via Task tool.

| Orchestrator CAN | Orchestrator CANNOT |
|------------------|---------------------|
| `projctl` commands | Read source code files |
| Read state.toml, context/*.toml | Edit source code files |
| Read tasks.md, result.toml | Write source code files |
| git status | Inline implementation work |

**Dispatch:** `Skill tool` for /tdd-red, /commit, etc. `Task tool` for exploration.

## Control Loop

| Step | Type | Action |
|------|------|--------|
| 1 | [D] | `projctl state get` - read current phase |
| 2 | [D] | `projctl state transition` - validates preconditions |
| 3 | [D] | `projctl map --cached` - territory context |
| 4 | [A] | Dispatch skill via Skill tool |
| 5 | [D] | `projctl context read --result` - parse result |
| 6 | [D] | `projctl state next` - check if work remains |
| 7 | [A] | If continue: loop. If stop: check reason |

## Stop Conditions

| Reason | Action |
|--------|--------|
| all_complete | Present summary |
| escalation_pending | Present to user |
| validation_failed | Run repair loop |
| retries_exhausted | Present failure |

## End-of-Command Sequence

Always run before completing:
```bash
projctl integrate features --dir .
projctl trace repair --dir .
projctl trace validate --dir .
```

## Modes

| Mode | Purpose |
|------|---------|
| new | Full interview → task breakdown → implementation |
| adopt | Infer artifacts from existing codebase |
| resume | Continue from saved state |

## Full Documentation

`projctl skills docs --skillname project` or see SKILL-full.md
