---
name: plan-producer
description: Produces structured project plan from issue context
context: inherit
model: opus
user-invocable: true
role: producer
phase: plan
---

# Plan Producer

Produce a structured project plan from issue context, identifying approach, tasks, risks, and open questions.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Issue description, codebase context, memory patterns |
| Pattern | GATHER -> SYNTHESIZE -> PRODUCE |
| Output | plan.md with problem, approach, tasks, risks |

---

## Workflow Context

- **Phase**: `plan_produce` (states.plan_produce)
- **Upstream**: Issue context, project initialization (`tasklist_create`, `init`)
- **Downstream**: `plan_approve` → user approval → parallel artifact production fork or retry
- **Model**: opus (default_model in workflows.toml)

This skill runs in the plan phase to produce the initial project plan before artifact creation begins.

---

## GATHER Phase

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode):
   - Issue description and acceptance criteria
   - Related codebase files and structure

2. Query semantic memory for relevant patterns (best-effort, non-blocking):
   - `projctl memory query "project planning patterns for <domain>"`
   - `projctl memory query "past approaches for similar features"`
   - If memory is unavailable or queries fail, continue without them (graceful degradation)

3. Scan existing codebase for relevant code:
   - Entry points and public API
   - Internal package structure
   - Test patterns in use

4. If missing information, send context request to team-lead with needed queries

---

## SYNTHESIZE Phase

1. **Define problem space**: Articulate the core problem from the issue
   - What exists today
   - What needs to change
   - What constraints apply

2. **Assess simplicity**: Before planning, ask:
   - "Is this the simplest approach that solves the problem?"
   - "Could this be done with fewer changes?"
   - Document alternatives considered and why current approach is appropriate

3. **Identify approach options**: Consider 2-3 approaches where applicable
   - Trade-offs for each
   - Recommended approach with rationale

4. **Plan task breakdown**: Structure work into logical units
   - Dependencies between tasks
   - Parallel opportunities
   - Risk areas

5. If blocked, send blocker message to team-lead with details

---

## PRODUCE Phase

1. Generate plan.md with:
   - **Problem**: What the issue addresses, current state, desired state
   - **Approach**: Recommended approach with rationale, alternatives considered
   - **Tasks**: High-level task list with dependencies
   - **Risks**: Known risks and mitigation strategies
   - **Open Questions**: Items needing user input before proceeding

2. Enter plan mode for interactive user review:
   - Write plan to `.claude/projects/<issue>/plan.md`
   - Call `EnterPlanMode` to enable user review and modification
   - User can ask questions, request changes to the plan
   - On approval (via `ExitPlanMode`): send completion message to orchestrator with plan path
   - On rejection: incorporate feedback, revise plan, re-enter plan mode

3. Send results to team lead via `SendMessage`:
   - Artifact path
   - Summary of approach
   - Open questions requiring user input

---

## Interactive Review

After producing plan.md, enter plan mode for interactive user review:

1. **Write plan**: Save structured plan to `.claude/projects/<issue>/plan.md`
2. **Enter plan mode**: Call `EnterPlanMode` — user sees and can modify the plan interactively
3. **On approval**: Send completion message to orchestrator with plan path
4. **On rejection**: Incorporate feedback, revise plan content, and re-enter plan mode

This workflow ensures the user can review, question, and refine the plan before implementation begins.

---

## Yield Protocol

### Yield Types

| Type | When |
|------|------|
| `complete` | Plan generated successfully |
| `need-context` | Need codebase files or issue details |
| `blocked` | Cannot proceed (insufficient information) |
| `error` | Something failed |

### Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-08T10:00:00Z

[payload]
artifact = "docs/plan.md"
files_modified = ["docs/plan.md"]

[payload.approach]
summary = "Add memory indexing via SQLite FTS5"
alternatives_considered = 2

[[payload.tasks]]
title = "Add FTS5 schema migration"
dependencies = "None"

[[payload.risks]]
description = "FTS5 may not be available on all platforms"
mitigation = "Fallback to LIKE queries"

[[payload.open_questions]]
question = "Should search results be ranked by recency or relevance?"

[context]
phase = "plan"
subphase = "complete"
```

---

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Read existing docs | `Read`, `Glob`, `Grep` tools directly |
| Run projctl commands | `Bash` tool directly |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

---

## Contract

```yaml
contract:
  outputs:
    - path: "docs/plan.md"
      id_format: "N/A"

  traces_to:
    - "issue description"

  checks:
    - id: "CHECK-001"
      description: "Problem section clearly states current and desired state"
      severity: error

    - id: "CHECK-002"
      description: "Approach section includes rationale"
      severity: error

    - id: "CHECK-003"
      description: "Alternatives considered and documented"
      severity: error

    - id: "CHECK-004"
      description: "Tasks section has logical breakdown"
      severity: error

    - id: "CHECK-005"
      description: "Risks identified with mitigation strategies"
      severity: error

    - id: "CHECK-006"
      description: "Open questions listed for user review"
      severity: warning

    - id: "CHECK-007"
      description: "Plan is appropriately scoped (not over-engineered)"
      severity: warning
```
