---
name: pm-infer-producer
description: Infer requirements from existing codebase analysis
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: pm
variant: infer
---

# PM Infer Producer

Analyzes existing code to infer requirements for codebase adoption.

**Template:** [PRODUCER-TEMPLATE.md](../shared/PRODUCER-TEMPLATE.md)
**Yield Protocol:** [YIELD.md](../shared/YIELD.md)

## Quick Reference

| Aspect | Details |
|--------|---------|
| Pattern | GATHER -> SYNTHESIZE -> PRODUCE |
| Input | Context TOML with project path |
| Output | requirements.md with REQ-N IDs |
| Primary Yield | `need-context` for code exploration |
| Terminal Yield | `complete` with artifact path |

## Workflow

### 1. GATHER Phase

Collect information by exploring existing codebase:

1. Read context from `[inputs]` section
2. Check `[query_results]` for resumed context
3. If code exploration needed, yield `need-context`:
   - `semantic` queries for understanding code behavior
   - `file` queries for specific source files
   - `territory` queries for codebase structure
4. Proceed to SYNTHESIZE when sufficient information gathered

**Analysis Sources:**

| Source | What to Extract |
|--------|-----------------|
| README.md | Purpose, features, usage examples |
| Existing docs | Preserve REQ-N items if present |
| CLI --help | Commands, flags, options |
| Public API | Functions, types, interfaces |
| Test names | Implied requirements |
| Error messages | Error handling requirements |

### 2. SYNTHESIZE Phase

Process gathered code analysis:

1. Identify distinct functional capabilities
2. Map behaviors to requirement categories
3. Check for conflicts with existing requirements
4. If blocked, yield `blocked` with details
5. Structure findings as requirements

**Categories:**

- Core functionality (what it does)
- Input/output (data formats, interfaces)
- Configuration (settings, options)
- Error handling (failure modes)
- Performance (if observable)

### 3. PRODUCE Phase

Create requirements artifact:

1. Generate requirements.md with REQ-N format
2. Include `**Traces to:**` links (ISSUE if available)
3. Write to path from `[config]`
4. Yield `complete` with artifact details

## REQ Format

```markdown
### REQ-001: Capability Name

Description of requirement inferred from code.

- [ ] Acceptance criterion 1
- [ ] Acceptance criterion 2

**Traces to:** ISSUE-NNN (if available)
```

## Yield Types

| Yield | When |
|-------|------|
| `need-context` | Need code exploration (semantic/file/territory queries) |
| `blocked` | Cannot proceed (missing access, unreadable code) |
| `complete` | requirements.md created successfully |
| `error` | Parse failure or other recoverable error |

### need-context Example

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:35:00Z

[[payload.queries]]
type = "semantic"
question = "What are the main commands this CLI supports?"

[[payload.queries]]
type = "territory"
scope = "public-api"

[[payload.queries]]
type = "file"
path = "README.md"

[context]
phase = "pm"
subphase = "GATHER"
awaiting = "context-results"
```

### complete Example

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T11:30:00Z

[payload]
artifact = "docs/requirements.md"
ids_created = ["REQ-001", "REQ-002", "REQ-003"]
files_modified = ["docs/requirements.md"]

[[payload.decisions]]
context = "Requirement scope"
choice = "Include only observable behaviors"
reason = "Cannot infer internal requirements from code"
alternatives = ["Include inferred internals", "Ask user"]

[context]
phase = "pm"
subphase = "complete"
```

## Rules

- Preserve existing REQ-N IDs if updating
- New IDs start after highest existing
- Output to path specified in context (default: docs/requirements.md)
- Infer only what is observable from code
- When uncertain, yield `need-context` for deeper exploration

## Comparison with Interview Variant

| Aspect | infer | interview |
|--------|-------|-----------|
| Primary source | Existing code | User conversation |
| Primary yield | `need-context` | `need-user-input` |
| Certainty | Lower (inference) | Higher (explicit) |
| Use case | Adoption, documentation | New development |

---

## Contract

```yaml
contract:
  outputs:
    - path: "docs/requirements.md"
      id_format: "REQ-N"

  traces_to:
    - "issue description"

  checks:
    - id: "CHECK-001"
      description: "Every requirement has REQ-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every REQ-N has acceptance criteria"
      severity: error

    - id: "CHECK-003"
      description: "Every REQ-N traces to the issue (if available)"
      severity: warning

    - id: "CHECK-004"
      description: "Acceptance criteria are measurable and testable"
      severity: error

    - id: "CHECK-005"
      description: "No ambiguous language (should, may, might)"
      severity: error

    - id: "CHECK-006"
      description: "Inferred requirements trace to observable code behavior"
      severity: error

    - id: "CHECK-007"
      description: "Edge cases identified where applicable"
      severity: warning

    - id: "CHECK-008"
      description: "Dependencies between requirements documented"
      severity: warning
```
