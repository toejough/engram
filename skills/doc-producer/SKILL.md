---
name: doc-producer
description: Produce and update project documentation
context: fork
model: sonnet
user-invocable: false
role: producer
phase: doc
---

# Documentation Producer

Produce and update README, API docs, and user guides based on project artifacts.

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | Context TOML with artifact paths |
| Analysis | Requirements, design, architecture, code |
| Output | README.md, API docs, user guides |

## Workflow

Follows GATHER -> SYNTHESIZE -> PRODUCE pattern.

### GATHER

1. Read context from `[inputs]` section
2. Load REQ-N from requirements.md
3. Load DES-N from design.md
4. Load ARCH-N from architecture.md
5. Scan codebase for public API
6. If missing information, yield `need-context` with queries

### SYNTHESIZE

1. Map requirements to user-facing features
2. Extract API signatures and types
3. Identify usage patterns from tests/examples
4. Structure content by audience (users, developers, contributors)

### PRODUCE

1. Generate/update documentation files:
   - README.md: Overview, installation, quick start
   - API docs: Types, functions, interfaces
   - User guides: Tutorials, recipes, FAQs
2. Add traceability comments linking to REQ-N, DES-N, ARCH-N
3. Yield `complete` with artifact paths

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol.

### Yield Types

| Type | When |
|------|------|
| `complete` | Documentation generated successfully |
| `need-context` | Need files, code, or examples |
| `blocked` | Cannot proceed (missing artifacts) |
| `error` | Something failed |

### Complete Yield Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T10:30:00Z

[payload]
artifact = "README.md"
files_modified = ["README.md", "docs/api.md", "docs/user-guide.md"]

[context]
phase = "doc"
subphase = "complete"
```

## Traceability

Documentation traces to upstream artifacts:

- **REQ-N**: Features described map to requirements
- **DES-N**: UX flows and design decisions documented
- **ARCH-N**: Technical architecture explained

Example traceability in generated docs:

```markdown
## Authentication
<!-- Traces: REQ-3, DES-2, ARCH-5 -->
Users can authenticate via OAuth 2.0...
```

## Result Format

`result.toml`: `[status]`, files modified, `[[decisions]]`

## Full Documentation

`projctl skills docs --skillname doc-producer` or see SKILL-full.md
