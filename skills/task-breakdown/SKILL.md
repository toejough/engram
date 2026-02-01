---
name: task-breakdown
description: Decompose architecture into implementation tasks with traceability IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Task Breakdown

Transform architecture specs into executable TDD tasks.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | requirements.md (REQ-) | architecture.md (ARCH-) | design.md (DES-) |
| Phases | 1. VALIDATE alignment | 2. DECOMPOSE into tasks | 3. SEQUENCE dependencies |
| Output | tasks.md with TASK-NNN IDs, acceptance criteria, traceability |

## Task Requirements

Each task must have:
- TASK-NNN ID (sequential)
- Clear title + description
- Acceptance criteria (checkboxes)
- Files to create/modify
- Dependencies (explicit TASK-NNN IDs or `None`)
- Traceability (REQ/DES/ARCH IDs)

## Sizing Order

| Priority | Type | Reason |
|----------|------|--------|
| 1 | Pure functions | No dependencies, algorithms |
| 2 | Types/interfaces | Enable other work |
| 3 | Storage layer | Foundation for services |
| 4 | Services | Business logic |
| 5 | Components | UI elements |
| 6 | Integration | Wiring together |

## Critical Rules

| Rule | Details |
|------|---------|
| Dependencies | Explicit TASK-NNN only - NEVER "All previous" or prose |
| DAG | No cycles allowed |
| Alignment | Validate REQ→ARCH coverage BEFORE decomposing |
| Size | One function/method = one task (for pure functions) |

## Result Format

`result.toml`: `[status]`, tasks list, coverage matrix, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname task-breakdown` or see SKILL-full.md
