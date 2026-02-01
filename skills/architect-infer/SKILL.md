---
name: architect-infer
description: Infer architecture decisions from code structure
user-invocable: false
---

# Architect Infer

Infer architecture from code structure for adoption.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | project dir | mode | req/design summaries |
| Focus | Implementation structure - organization, deps, patterns |
| Output | docs/architecture.md with ARCH-NNN IDs |

## Modes

| Mode | Action |
|------|--------|
| infer | Create architecture.md from scratch |
| update | Add new decisions, preserve existing IDs |
| augment | Add ARCH-NNN IDs to existing content |
| normalize | Fix format, header levels |

## Analysis Sources

| Source | Extract |
|--------|---------|
| go.mod/package.json | Dependencies, module structure |
| Directory structure | Package organization |
| Import graph | Component dependencies |
| Build tooling | Makefile, mage, etc. |
| Config patterns | How config is loaded |

## ARCH Format

```markdown
### ARCH-001: Decision Title
Rationale and implementation notes.
**Traces to:** REQ-001, DES-002
```

## Rules

- Preserve existing ARCH-NNN IDs
- Every ARCH must trace to REQ/DES
- Document technology choices with rationale

## Result Format

`result.toml`: `[status]`, architecture.md path, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname architect-infer` or see SKILL-full.md
