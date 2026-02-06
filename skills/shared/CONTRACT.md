# Contract Standard

Standard format for validation contracts embedded in producer SKILL.md files.

**Design references:** DES-001 (Contract YAML Format), DES-002 (Contract Section Placement)

---

## Schema

Contracts use a flat YAML structure optimized for LLM parsing:

```yaml
contract:
  outputs:
    - path: "<artifact-path>"
      id_format: "<ID-PREFIX>-N"

  traces_to:
    - "<upstream-artifact-path>"

  checks:
    - id: "CHECK-NNN"
      description: "<what to validate>"
      severity: error | warning
```

All fields at a glance:

| Field | Parent | Type | Description |
|-------|--------|------|-------------|
| `outputs` | contract | array | What files this producer creates |
| `path` | outputs[] | string | Path to output artifact |
| `id_format` | outputs[] | string | ID format used (e.g., "REQ-N", "DES-N") |
| `traces_to` | contract | array | Upstream artifacts that must be referenced |
| `checks` | contract | array | Validation criteria |
| `id` | checks[] | string | Unique check identifier |
| `description` | checks[] | string | What the check validates |
| `severity` | checks[] | enum | `error` or `warning` |

---

## Outputs

The `outputs` section declares what artifacts the producer creates:

```yaml
outputs:
  - path: "docs/requirements.md"
    id_format: "REQ-N"
```

- `path`: Relative path from project root to the artifact
- `id_format`: Pattern for IDs in this artifact (N = sequential number)

**Common ID formats:**

| Phase | Format | Example |
|-------|--------|---------|
| pm | REQ-N | REQ-1, REQ-2 |
| design | DES-N | DES-1, DES-2 |
| arch | ARCH-N | ARCH-1, ARCH-2 |
| breakdown | TASK-N | TASK-1, TASK-2 |

---

## Traces

The `traces_to` section specifies what upstream artifacts must be referenced:

```yaml
traces_to:
  - "docs/requirements.md"
  - "issue description"
```

Each entry identifies an artifact that IDs in this producer's output must trace to.

- File paths: The actual artifact file
- "issue description": The original issue/task that started the work

---

## Checks

The `checks` section defines validation criteria:

```yaml
checks:
  - id: "CHECK-001"
    description: "Every entry has a REQ-N identifier"
    severity: error

  - id: "CHECK-002"
    description: "Every REQ-N traces to the issue"
    severity: error

  - id: "CHECK-003"
    description: "Acceptance criteria are testable"
    severity: warning
```

- `id`: Unique identifier for the check (CHECK-NNN format)
- `description`: Human-readable description of what is validated
- `severity`: How failures are handled

---

## Severity

Two severity levels control how QA handles failures:

| Severity | Meaning | QA Result |
|----------|---------|-----------|
| `error` | Must pass for QA approval | QA fails if any error check fails |
| `warning` | Should pass, but not blocking | QA passes with note if warning check fails |

**Guidelines:**

- Use `error` for structural requirements (IDs exist, traces present, format correct)
- Use `warning` for quality suggestions (clarity, completeness, style)

**Example output with warnings:**

```
QA Results: PASSED

[x] CHECK-001: Every entry has REQ-N ID
[x] CHECK-002: Every REQ-N traces to issue
[x] CHECK-003: Acceptance criteria are testable (warning: REQ-3 criteria could be more specific)
```

---

## Contract Section Placement

Contracts are embedded in producer SKILL.md files under a `## Contract` heading with a fenced YAML block:

```markdown
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
      description: "Every requirement has REQ-N ID"
      severity: error
```
```

**Placement rules (per DES-002):**

1. `## Contract` section in producer SKILL.md
2. Immediately followed by a fenced YAML code block
3. Single source of truth - no separate contract files

---

## Examples

### PM Producer Contract (Interview/Infer)

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
      description: "Every REQ-N traces to the issue"
      severity: error

    - id: "CHECK-004"
      description: "Acceptance criteria are measurable and testable"
      severity: warning
```

### Design Producer Contract

```yaml
contract:
  outputs:
    - path: "docs/design.md"
      id_format: "DES-N"

  traces_to:
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every design decision has DES-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every DES-N traces to at least one REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "No orphan REQ-N references"
      severity: warning

    - id: "CHECK-004"
      description: "Design describes user-facing behavior, not implementation"
      severity: warning
```

### Architecture Producer Contract

```yaml
contract:
  outputs:
    - path: "docs/architecture.md"
      id_format: "ARCH-N"

  traces_to:
    - "docs/design.md"
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every architecture decision has ARCH-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every ARCH-N traces to at least one DES-N or REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "Technical decisions include rationale"
      severity: warning
```

### Breakdown Producer Contract

```yaml
contract:
  outputs:
    - path: "docs/tasks.md"
      id_format: "TASK-N"

  traces_to:
    - "docs/architecture.md"
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every task has TASK-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every TASK-N traces to at least one ARCH-N or REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "Tasks have clear acceptance criteria"
      severity: error

    - id: "CHECK-004"
      description: "Task dependencies are documented"
      severity: warning
```

### TDD Red Producer Contract

```yaml
contract:
  outputs:
    - path: "<test-file>"
      id_format: "N/A"

  traces_to:
    - "docs/tasks.md"

  checks:
    - id: "CHECK-001"
      description: "Test file exists at specified path"
      severity: error

    - id: "CHECK-002"
      description: "Tests fail when run (red phase)"
      severity: error

    - id: "CHECK-003"
      description: "Tests cover task acceptance criteria"
      severity: error
```

---

## Version and Evolution

**Current version:** 1.0 (initial)

**Evolution policy:**

1. **No version field in contracts** - Format is simple enough that versioning adds unnecessary complexity
2. **Backward compatible changes only** - New optional fields allowed; removing or changing required fields requires migration
3. **Changes documented here** - This file is the source of truth for contract format
4. **QA skill handles unknowns gracefully** - Unknown fields are ignored, not errors

**Migration notes:**

- If contract format changes significantly, producers will be updated as part of the change
- The universal QA skill (per ARCH-019) extracts contracts via markdown parsing, making format changes localized

---

## Related Documents

- **DES-001**: Contract YAML Format - Design decision establishing this format
- **DES-002**: Contract Section Placement - Where contracts live in SKILL.md
- **ARCH-020**: Contract Standard Location - Why this file exists at `skills/shared/CONTRACT.md`
- **ARCH-021**: Contract Extraction Algorithm - How QA extracts contracts from SKILL.md
- **PRODUCER-TEMPLATE.md**: Template for creating producer skills
