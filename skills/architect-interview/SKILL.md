---
name: architect-interview
description: Technology and architecture interview producing architecture spec with traceability IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Architect Interview

Interview for tech choices, produce architecture.md with ARCH-NNN IDs.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Domain | Implementation space - technology, layers, data models, APIs |
| Phases | UNDERSTAND reqs → RESEARCH options → INTERVIEW preferences → SPECIFY decisions |
| Output | architecture.md (ARCH-NNN) with REQ/DES traceability |
| Does NOT own | What to build (PM) | How users interact (Design) |

## Core Principles

| Principle | Details |
|-----------|---------|
| Progressive disclosure | High-level overview → detailed sections |
| Pure business logic | DI for testability |
| Clean separation | Domain, storage, UI, infrastructure |

## Interview Topics

| Topic | Questions |
|-------|-----------|
| Technology | Language preferences, framework experience |
| Constraints | Team skills, timeline, existing systems |
| Scale | Users, data volume, performance needs |
| Deployment | Cloud, self-hosted, platforms |

## ARCH Entry Format

```markdown
### ARCH-001: Decision Title
- Rationale
- Implementation notes
**Traces to:** REQ-001, DES-002
```

## Rules

- ALWAYS conduct interactive interview
- Document alternatives considered
- Every ARCH traces to REQ/DES

## Result Format

`result.toml`: `[status]`, architecture.md path, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname architect-interview` or see SKILL-full.md
