---
name: pm-infer
description: Infer requirements from codebase analysis
user-invocable: false
---

# PM Infer

Infer requirements from existing codebase for adoption.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML | project dir | mode (infer/update/normalize) |
| Analysis | README → existing docs → CLI help → public API → tests |
| Output | docs/requirements.md with REQ-NNN IDs |

## Modes

| Mode | Action |
|------|--------|
| infer | Create requirements.md from scratch |
| update | Add new requirements, preserve existing IDs |
| normalize | Convert table format → header format |

## Analysis Sources

| Source | Extract |
|--------|---------|
| README.md | Purpose, features, usage examples |
| Existing docs | Preserve REQ-NNN items |
| CLI --help | Commands, flags, options |
| Public API | Functions, types, interfaces |
| Test names | Implied requirements |

## REQ Format

```markdown
### REQ-001: Capability Name
Description of requirement.
- [ ] Acceptance criterion 1
- [ ] Acceptance criterion 2
```

## Rules

- Preserve existing REQ-NNN IDs
- New IDs start after highest existing
- Output to docs/requirements.md (not .project/)

## Output Format

`result.toml`: `[status]`, requirements.md path, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname pm-infer` or see SKILL-full.md
