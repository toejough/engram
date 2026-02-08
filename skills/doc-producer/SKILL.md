---
name: doc-producer
description: Produce and update project documentation
context: inherit
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
| Input | Context from spawn prompt: artifact paths |
| Analysis | Requirements, design, architecture, code |
| Output | README.md, API docs, user guides |

## Workflow

Follows GATHER -> SYNTHESIZE -> PRODUCE pattern.

### GATHER

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode)
2. Load REQ-N from requirements.md
3. Load DES-N from design.md
4. Load ARCH-N from architecture.md
5. Scan codebase for public API
6. Query memory for past learnings:
   - `projctl memory query "documentation patterns for <project-domain>"`
   - `projctl memory query "known failures in documentation validation"`
   - Memory queries are non-blocking - if unavailable, continue without them
7. If missing information, send a message to team-lead requesting context

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
3. **Enforce ID format conventions**: Convert zero-padded IDs to non-zero-padded format
   - Search for zero-padded IDs: `REQ-001`, `DES-002`, `ARCH-003`, etc.
   - Convert to non-zero-padded: `REQ-1`, `DES-2`, `ARCH-3`, etc.
   - Ensure trace syntax follows current format: `**Traces to:** ID-N` (not `<!-- Traces: ... -->`)
   - Validate all section IDs use format `### ID-N: Title` (not `### ID-NNN: Title`)
   - Apply to README.md and all documentation files being updated
4. **Re-point test traces**: Replace `// traces: TASK-NNN` with permanent artifact IDs
   - Look up each task's `**Traces to:**` field in tasks.md
   - Replace with the lowest-level permanent artifact (prefer ARCH-N, then DES-N, then REQ-N)
   - Run `projctl trace validate` to verify no orphan TASK references remain
5. Send results to team lead via `SendMessage`:
   - Artifact paths
   - Files modified
   - Key decisions made

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

---

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Read existing docs | `Read`, `Glob`, `Grep` tools directly |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

---

## Contract

```yaml
contract:
  outputs:
    - path: "README.md"
      id_format: "N/A"
    - path: "docs/api.md"
      id_format: "N/A"
    - path: "docs/user-guide.md"
      id_format: "N/A"

  traces_to:
    - "docs/requirements.md"
    - "docs/design.md"
    - "docs/architecture.md"

  checks:
    - id: "CHECK-001"
      description: "All public APIs documented"
      severity: error

    - id: "CHECK-002"
      description: "Installation and quick start present"
      severity: error

    - id: "CHECK-003"
      description: "Traces to REQ-N, DES-N, ARCH-N included"
      severity: error

    - id: "CHECK-004"
      description: "Code examples compile and run (accuracy)"
      severity: error

    - id: "CHECK-005"
      description: "API signatures match implementation"
      severity: error

    - id: "CHECK-006"
      description: "No orphan traces (referencing non-existent IDs)"
      severity: error

    - id: "CHECK-007"
      description: "projctl trace validate passes"
      severity: error

    - id: "CHECK-008"
      description: "Version numbers current"
      severity: warning

    - id: "CHECK-009"
      description: "User guides cover key workflows"
      severity: warning
```
