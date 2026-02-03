---
name: doc-qa
description: Validate documentation completeness, accuracy, and traceability
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: qa
phase: doc
---

# Documentation QA

Validate documentation for completeness, accuracy, and traceability to upstream artifacts.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML with documentation artifact paths |
| Validates | README.md, API docs, user guides |
| Criteria | Completeness, accuracy, traceability |

## Workflow

Follows REVIEW -> RETURN pattern per QA-TEMPLATE.

### REVIEW

1. Read context from `[inputs]` section
2. Load documentation artifacts (README.md, API docs, user guides)
3. Load upstream artifacts:
   - REQ-N from requirements.md
   - DES-N from design.md
   - ARCH-N from architecture.md
4. Check completeness criteria:
   - All public APIs documented?
   - Installation and quick start present?
   - User guides cover key workflows?
5. Check accuracy criteria:
   - Code examples compile/run?
   - API signatures match implementation?
   - Version numbers current?
6. Check traceability:
   - Documentation traces to REQ-N, DES-N, ARCH-N?
   - No orphan traces (referencing non-existent IDs)?
7. Compile findings

### RETURN

Based on REVIEW findings:

1. If all criteria pass:
   - Yield `approved` with checklist

2. If producer can fix issues:
   - Yield `improvement-request` with specific issues

3. If upstream phase has error/gap/conflict:
   - Yield `escalate-phase` with proposed changes

4. If cannot resolve:
   - Yield `escalate-user` with question

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol.

### Yield Types

| Type | When |
|------|------|
| `approved` | Documentation passes all checks |
| `improvement-request` | Issues producer can fix |
| `escalate-phase` | Upstream phase has problems |
| `escalate-user` | Cannot resolve without user input |

### Approved Yield Example

```toml
[yield]
type = "approved"
timestamp = 2026-02-02T12:00:00Z

[payload]
reviewed_artifact = "README.md"
checklist = [
    { item = "All public APIs documented", passed = true },
    { item = "Installation instructions complete", passed = true },
    { item = "Traces to REQ-N, DES-N, ARCH-N", passed = true },
    { item = "Code examples accurate", passed = true }
]

[context]
phase = "doc"
role = "qa"
iteration = 1
```

### Improvement Request Yield Example

```toml
[yield]
type = "improvement-request"
timestamp = 2026-02-02T12:05:00Z

[payload]
from_agent = "doc-qa"
to_agent = "doc-producer"
iteration = 2
issues = [
    "README missing installation for Windows",
    "API docs outdated for parseConfig() signature",
    "User guide example at line 45 references removed function"
]

[context]
phase = "doc"
role = "qa"
iteration = 2
max_iterations = 3
```

### Escalate-Phase Yield Example

```toml
[yield]
type = "escalate-phase"
timestamp = 2026-02-02T12:10:00Z

[payload.escalation]
from_phase = "doc"
to_phase = "design"
reason = "gap"

[payload.issue]
summary = "User workflow not captured in design"
context = "Documentation reveals user need not in DES-N artifacts"

[[payload.proposed_changes.requirements]]
action = "add"
id = "DES-15"
title = "Batch Processing Workflow"
content = "Users need to process multiple files in a single command..."

[context]
phase = "doc"
role = "qa"
escalating = true
```

## Validation Criteria

### Completeness

- [ ] README has project overview
- [ ] Installation instructions for all platforms
- [ ] Quick start guide present
- [ ] All public API endpoints documented
- [ ] Error handling documented
- [ ] Configuration options listed

### Accuracy

- [ ] Code examples execute without errors
- [ ] API signatures match implementation
- [ ] Links resolve (no 404s)
- [ ] Version numbers current
- [ ] Screenshots reflect current UI

### Traceability

- [ ] README traces to high-level REQ-N
- [ ] API docs trace to ARCH-N decisions
- [ ] User guides trace to DES-N flows
- [ ] No orphan traces (referencing non-existent IDs)

## Result Format

Yield files written to path specified in context (`output.yield_path`).

## Full Documentation

`projctl skills docs --skillname doc-qa` or see SKILL-full.md
