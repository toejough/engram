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
| Handoffs | `projctl context write/read` with `output.yield_path` |
| PAIR LOOP | Producer → QA → iterate (max 3x) or advance |
| TDD | Red→commit→green→commit→refactor→commit |
| Continue | If `state next`=continue, proceed immediately |
| Dispatch | ALL code work via Skill/Task tool |

## Skill Dispatch (New Names)

| Phase | Producer Skill | QA Skill |
|-------|---------------|----------|
| PM | `pm-interview-producer` or `pm-infer-producer` | `pm-qa` |
| Design | `design-interview-producer` or `design-infer-producer` | `design-qa` |
| Architecture | `arch-interview-producer` or `arch-infer-producer` | `arch-qa` |
| Breakdown | `breakdown-producer` | `breakdown-qa` |
| TDD | `tdd-producer` (composite) | `tdd-qa` |
| Documentation | `doc-producer` | `doc-qa` |
| Alignment | `alignment-producer` | `alignment-qa` |
| Retro | `retro-producer` | `retro-qa` |
| Summary | `summary-producer` | `summary-qa` |

## PAIR LOOP Pattern

```
PAIR LOOP (for each phase):
1. Write context with output.yield_path
2. Dispatch PRODUCER skill
3. Read yield (see shared/YIELD.md)
4. If yield.type = "complete":
   - Dispatch QA skill
   - Read QA yield
5. Handle QA yield:
   - "approved" → advance to next phase
   - "improvement-request" → resume producer with feedback (max 3x)
   - "escalate-phase" → return to prior phase with proposed_changes
   - "escalate-user" → present to user
```

## Yield Type Handling

| Yield Type | Orchestrator Action |
|------------|---------------------|
| `complete` | Advance to QA or next phase |
| `approved` | Phase complete, advance |
| `need-user-input` | Prompt user, resume with answer |
| `need-context` | Dispatch `context-explorer`, resume with results |
| `need-decision` | Present options, resume with choice |
| `improvement-request` | Resume producer with feedback |
| `escalate-phase` | Return to prior phase |
| `escalate-user` | Present to user |
| `blocked` | Present blocker, await resolution |
| `error` | Retry (max 3x) or escalate |

## Context Format

When writing context for skills:
```toml
[input]
phase = "pm"
task = "TASK-5"
# ... phase-specific data

[output]
yield_path = ".claude/agents/pm-interview-producer-yield.toml"
```

Skills write their yield to the provided `yield_path`.

## Support Skills

| Skill | Purpose |
|-------|---------|
| `context-explorer` | Handle need-context queries (file, memory, territory, web, semantic) |
| `intake-evaluator` | Classify request type (new, adopt, align, single-task) |
| `parallel-looper` | Run N independent PAIR LOOPs in parallel |
| `consistency-checker` | Validate parallel outputs for consistency |
| `next-steps` | Suggest follow-up work |

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

## Full Docs

`projctl skills docs --skillname project` or see SKILL-full.md
