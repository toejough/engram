---
name: intake-evaluator
description: |
  Core: Analyzes user requests and classifies them into appropriate workflow types: new, align, scoped, or quick-fix.
  Triggers: classify request, determine workflow, evaluate task scope, route request, workflow selection.
  Domains: intake, classification, workflow-routing, scope-assessment, tier-selection.
  Anti-patterns: NOT for implementation, NOT for planning details, only classifies and routes to appropriate workflow.
  Related: project (receives classification to select workflow), plan-producer (executes after classification).
context: inherit
model: haiku
user-invocable: true
role: evaluator
---

# Intake Evaluator

Standalone skill that evaluates incoming user requests and classifies them into the appropriate workflow type. Sends classification results to team-lead. Escalates to user if classification is uncertain.

---

## Workflow Tiers

| Tier | Workflow | Description | When to use |
|------|----------|-------------|-------------|
| Full Project | `new` | New feature from scratch | "build X", no existing code, needs PM/design/arch |
| Full Project | `align` | Adopt existing code or check alignment | "adopt", "verify", "drift", "align", code exists but lacks artifacts |
| Scoped | `scoped` | Well-defined multi-file change | Clear acceptance criteria, needs task breakdown, existing project context |
| Quick Fix | `quick-fix` | Exact files/lines known | Specific file/line, known root cause, single commit |

### Selection Guide

```
Is the request about existing code that needs adoption or alignment checking?
  → YES: align (Full Project)

Does the request need PM interview, design, architecture from scratch?
  → YES: new (Full Project)

Are the exact files/lines and fix already known?
  → YES: quick-fix (Quick Fix — no state machine, just do it)

Is it a well-defined change spanning multiple files/tasks?
  Check ALL of these for scoped:
  ✓ Problem AND solution both clearly defined in issue
  ✓ No exploratory or discovery work needed
  ✓ No design trade-offs to evaluate
  ✓ Existing codebase context is sufficient
  ✓ Clear acceptance criteria already exist
  → ALL YES: scoped (Scoped — TDD loop with task breakdown)
  → ANY NO: new (Full Project — needs PM/design/arch phases)

Uncertain?
  → Escalate to user with options
```

---

## Classification Types

| Type | Tier | Description | Signals |
|------|------|-------------|---------|
| `new` | Full Project | New feature from scratch | "build X", "create Y", "new feature", no existing code mentioned |
| `align` | Full Project | Adopt existing code or check alignment | "adopt", "take over", "verify", "check if", "drift", "align", references existing code/artifacts |
| `scoped` | Scoped | Well-defined multi-file change | Clear AC, references existing project, needs breakdown but not full PM/design |
| `quick-fix` | Quick Fix | Exact files/lines known | "fix this bug", "update this line", specific file/line, known root cause, single commit scope |

---

## Classification Criteria

Evaluate the request against these indicators:

### New Project Signals
- Request mentions building something from scratch
- No references to existing implementation
- Describes a problem that needs a solution designed
- Uses words like "create", "build", "implement", "new"
- Needs requirements gathering, design, and architecture

### Align Project Signals
- References existing code or codebase
- Mentions "take over", "adopt", "inherit", "verify", "align"
- Wants to add project management to existing work
- Code exists but lacks requirements/design artifacts
- Asks if implementation matches requirements
- References drift between code and documentation

### Scoped Signals
- Well-defined change with clear acceptance criteria
- Spans multiple files or requires task breakdown
- Doesn't need full PM interview or architecture design
- Existing project context is sufficient
- "Add feature X to existing system Y"

**Scoped vs Full Decision Criteria:**

Use **scoped** when ALL of these are true:
1. Problem statement is clear (no ambiguity about what's wrong/needed)
2. Solution approach is known (no need to explore alternatives)
3. No design trade-offs require user input
4. Codebase context exists (not greenfield)
5. Acceptance criteria can be written without discovery

Use **full (new)** when ANY of these are true:
1. Problem is vague or needs investigation
2. Multiple solution approaches need evaluation
3. Design decisions need user input (UI, API shape, etc.)
4. No existing codebase to build on
5. Requirements need gathering from user

**Examples from past issues:**
- ISSUE-170 (scoped ✓): "Fix trace validator ID normalization" — clear problem, clear solution, existing code
- ISSUE-152 (full ✓): "Memory tiering lifecycle" — new feature, needed PM/design/arch from scratch
- ISSUE-202 (scoped ✓): "Restructure hook system" — well-defined AC, existing code, no design trade-offs

### Quick Fix Signals
- Scoped to a specific file or function
- Bug fix with known root cause
- Exact file path and line number mentioned or obvious
- Single commit scope
- "Fix the typo in X", "Update this config value"
- No task breakdown needed — just do it

---

## Confidence Handling

When classification is ambiguous or uncertain:

1. **High confidence (>80%)**: Send classification to team-lead
2. **Medium confidence (50-80%)**: Send classification to team-lead with confidence note
3. **Low confidence (<50%)**: Use `AskUserQuestion` with options for user

Escalate to user when:
- Request contains signals for multiple categories
- Key context is missing to make determination
- Request is vague or could be interpreted multiple ways

---

## Workflow

1. Read the user's request from context
2. Analyze against classification criteria
3. Determine confidence level
4. If confident: Send classification to team-lead via `SendMessage`
5. If uncertain: Use `AskUserQuestion` with options

---

## Boundaries

| In Scope | Out of Scope |
|----------|--------------|
| Classify request type | Execute any workflow |
| Analyze request signals | Gather requirements |
| Escalate uncertainty | Make design decisions |
| Output classification | Modify any files |

This skill only classifies - it does not start any workflow. The orchestrator uses the classification to determine which workflow to invoke.
