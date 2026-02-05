---
name: breakdown-producer
description: Decompose architecture into implementation tasks with dependency graph
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: breakdown
---

# Breakdown Producer

Transform architecture specs into executable TDD tasks with TASK-N IDs.

**Yield Protocol:** See [YIELD.md](../shared/YIELD.md)

## Quick Reference

| Aspect | Details |
|--------|---------|
| Input | requirements.md, design.md, architecture.md |
| Pattern | GATHER -> SYNTHESIZE -> PRODUCE |
| Output | tasks.md with TASK-N IDs, dependency graph |

---

## GATHER Phase

1. Read context for artifact paths:
   - requirements.md (REQ-N IDs)
   - design.md (DES-N IDs)
   - architecture.md (ARCH-N IDs)

2. If artifacts missing, yield `need-context`:
   ```toml
   [yield]
   type = "need-context"

   [[payload.queries]]
   type = "file"
   path = "docs/architecture.md"
   ```

3. Check for `[query_results]` if resuming

---

## SYNTHESIZE Phase

1. Validate alignment: All REQ -> ARCH coverage

2. **Assess simplicity**: Before decomposing, ask:
   - "Is there a simpler approach that achieves the same outcome?"
   - "Could this be done with fewer tasks/components/changes?"
   - Document alternatives considered and why current approach is appropriate

3. Identify decomposition units:
   - Pure functions (no dependencies)
   - Types/interfaces
   - Storage layer
   - Services
   - Components
   - Integration points

4. Build dependency graph:
   - Explicit TASK-N references only
   - No cycles (DAG requirement)
   - No prose like "All previous"

5. If blocked, yield `blocked` with details

---

## PRODUCE Phase

1. Generate tasks.md with:
   - TASK-N IDs (sequential)
   - Title + description
   - Acceptance criteria (checkboxes)
   - Files to create/modify
   - Dependencies (explicit TASK-N or `None`)
   - `**Traces to:**` links (REQ/DES/ARCH)

2. Include dependency graph visualization

3. Yield `complete`:
   ```toml
   [yield]
   type = "complete"

   [payload]
   artifact = "docs/tasks.md"
   ids_created = ["TASK-1", "TASK-2", "TASK-3"]

   [context]
   phase = "breakdown"
   subphase = "complete"
   ```

---

## Dependency Graph Format

```
TASK-1 (foundation)
    |
TASK-2 (types)
    |
TASK-3 ----+---- TASK-4
           |
        TASK-5
```

Rules:
- Arrows show dependencies (child depends on parent)
- Group related tasks visually
- Show parallel opportunities

---

## Task Format

```markdown
### TASK-N: [visual] Title

**Description:** What this task accomplishes

**Status:** Ready | In Progress | Complete

**Acceptance Criteria:**
- [ ] Criterion 1
- [ ] Criterion 2

**Files:** `path/to/file.go`

**Dependencies:** TASK-X, TASK-Y | None

**Simplicity Assessment:** Brief explanation of alternatives considered (e.g., "Considered combining with TASK-X but kept separate due to different dependencies" or "Simplest approach - no viable alternatives identified")

**Traces to:** ARCH-1, DES-2
```

---

## Simplicity Assessment

The simplicity assessment field documents whether the task breakdown is appropriately scoped or if simpler alternatives exist.

### When to Document Alternatives

Include alternatives considered when:

1. **Combining tasks**: "Considered merging TASK-X and TASK-Y but kept separate due to different dependencies"
2. **Splitting tasks**: "Split from TASK-Z to isolate integration testing"
3. **Using existing code**: "Considered reusing module X but incompatible interface"
4. **Skipping abstraction**: "Could add abstraction layer but unnecessary for single use case"

### Simplest Approach

If no viable alternatives exist, document:
- "Simplest approach - no viable alternatives identified"
- "Minimal change required - direct implementation"

### Example Assessments

**Good**: "Considered combining with TASK-3 database setup, but kept separate because TASK-3 blocks on schema design while this can proceed with mocks"

**Good**: "Simplest approach - single function addition with no dependencies"

**Poor**: "N/A" or "None" (doesn't show thinking)

**Poor**: "Best approach" (doesn't document alternatives)

---

## Visual Task Detection

Apply `[visual]` marker to tasks when:

1. **Files created/modified** include:
   - UI components (`.tsx`, `.vue`, `.svelte`)
   - CSS/styling files
   - CLI output formatting code
   - Template/view files

2. **Description mentions**:
   - "display", "show", "render", "appearance"
   - "button", "dialog", "modal", "form"
   - "output format", "table", "color"

3. **Acceptance criteria reference**:
   - Visual properties (size, color, position)
   - User-visible behavior
   - Design spec compliance

### Example

Task affects `components/Button.tsx` and AC says "button displays loading spinner":

```markdown
### TASK-7: [visual] Add loading state to submit button
```

---

## Sizing Priority

| Priority | Type | Reason |
|----------|------|--------|
| 1 | Pure functions | No dependencies, testable |
| 2 | Types/interfaces | Enable other work |
| 3 | Storage layer | Foundation for services |
| 4 | Services | Business logic |
| 5 | Components | UI elements |
| 6 | Integration | Wiring together |

---

## Yield Types Used

| Type | When |
|------|------|
| `complete` | tasks.md created with all TASK-N IDs |
| `need-context` | Need architecture/requirements files |
| `blocked` | Cannot decompose (missing info, conflicts) |
| `error` | Parse failure, invalid input |

---

## Contract

```yaml
contract:
  outputs:
    - path: "docs/tasks.md"
      id_format: "TASK-N"

  traces_to:
    - "docs/architecture.md"
    - "docs/design.md"
    - "docs/requirements.md"

  checks:
    - id: "CHECK-001"
      description: "Every task has TASK-N identifier"
      severity: error

    - id: "CHECK-002"
      description: "Every TASK-N traces to at least one ARCH-N, DES-N, or REQ-N"
      severity: error

    - id: "CHECK-003"
      description: "All ARCH-N IDs have at least one implementing TASK (architecture coverage)"
      severity: error

    - id: "CHECK-004"
      description: "Tasks have testable acceptance criteria"
      severity: error

    - id: "CHECK-005"
      description: "Acceptance criteria are measurable, not vague"
      severity: error

    - id: "CHECK-006"
      description: "No orphan tasks (all trace to ARCH/DES/REQ)"
      severity: error

    - id: "CHECK-007"
      description: "Sequential numbering (no gaps)"
      severity: error

    - id: "CHECK-008"
      description: "Dependencies reference valid TASK-N IDs"
      severity: error

    - id: "CHECK-009"
      description: "No prose dependencies (explicit TASK-N references only)"
      severity: error

    - id: "CHECK-010"
      description: "Appropriate granularity (not too large/small)"
      severity: warning

    - id: "CHECK-011"
      description: "Visual tasks marked with [visual] tag"
      severity: warning

    - id: "CHECK-012"
      description: "Each task includes simplicity assessment explaining alternatives considered"
      severity: warning
```
