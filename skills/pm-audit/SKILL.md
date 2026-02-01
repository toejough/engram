---
name: pm-audit
description: Validate implementation against requirements specification
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# PM Audit

Validate implementation against requirements by testing actual workflows.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | requirements.md | project dir | how to access app |
| Process | Load REQ IDs | Start app | Test each workflow | Verify success criteria | Check edge cases |
| Domain | Problem space | Does NOT validate: design quality or architecture |

## Critical Rule

**"Feature exists" ≠ "Feature works"**

For EVERY interactive element:
- Click it / interact with it
- Verify expected outcome occurs
- If nothing happens = DEFECT

Evidence that matters: **you did the action and saw the result**

## Workflow Testing

| Step | Action |
|------|--------|
| User stories | Attempt workflow as user would, document each step |
| Success criteria | Test explicitly, document PASS/FAIL with evidence |
| Edge cases | Reproduce scenario, verify expected behavior |
| Constraints | Verify "approaches to avoid" were avoided |

## Findings Classification

| Type | Meaning |
|------|---------|
| Critical | Workflow broken, requirement unmet |
| Major | Friction, partial failure |
| Minor | Polish issues |

## Output Format

`result.toml`: `[status]`, findings by REQ ID, `[[decisions]]`, `[[learnings]]`

## Full Documentation

`projctl skills docs --skillname pm-audit` or see SKILL-full.md
