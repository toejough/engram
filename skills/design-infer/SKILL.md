---
name: design-infer
description: Infer design decisions from interface analysis
user-invocable: false
---

# Design Infer

Infer design decisions from user-facing interfaces for adoption.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | project dir | mode | requirements summary |
| Focus | User interaction - not implementation |
| Output | docs/design.md with DES-NNN IDs |

## Modes

| Mode | Action |
|------|--------|
| infer | Create design.md from scratch |
| update | Add new decisions, preserve existing IDs |
| create | Create project-level when only feature-specific exist |
| normalize | Convert table → header format |

## Analysis Sources

| Source | Extract |
|--------|---------|
| CLI --help | Help text layout, structure |
| Error messages | How errors are presented |
| Output formatting | Tables, JSON, colors, progress |
| Interactive prompts | Questions, confirmations |

## DES Format

```markdown
### DES-001: Decision Title
Description of design decision.
**Traces to:** REQ-001, REQ-002
```

## Rules

- Preserve existing DES-NNN IDs
- Every DES must trace to REQ
- Focus on user-facing, not implementation

## Result Format

`result.toml`: `[status]`, design.md path, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname design-infer` or see SKILL-full.md
