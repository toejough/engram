---
name: design-interview
description: Visual and interaction design interview producing design specs with traceability IDs
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Design Interview

Transform requirements into visual designs via Pencil MCP.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Domain | User interaction space - workflows, layouts, interactions |
| Phases | UNDERSTAND spec → PREFERENCES interview → DESIGN system → BUILD screens |
| Output | design.md (DES-NNN) + .pen files |
| Does NOT own | What to build (PM) | How to build (Architecture) |

## Design Outputs

| Artifact | Contents |
|----------|----------|
| design.md | DES-NNN IDs with REQ traceability |
| Design system | Colors, typography, spacing, components |
| .pen files | Key screens as Pencil designs |

## Interview Phases

| Phase | Goal |
|-------|------|
| UNDERSTAND | Read spec, identify flows, note constraints |
| PREFERENCES | Visual style, existing patterns, accessibility |
| DESIGN SYSTEM | Establish tokens, components before screens |
| BUILD | Create screens in dependency order |

## DES Entry Format

```markdown
### DES-001: Screen Name
- Layout description
- Components used
- Interactions
**Traces to:** REQ-001, REQ-002
```

## Rules

- All designs in Pencil MCP (.pen files)
- Build design system BEFORE screens
- Every DES traces to REQ

## Output Format

`result.toml`: `[status]`, design.md path, .pen paths, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname design-interview` or see SKILL-full.md
