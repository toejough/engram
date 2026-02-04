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
2. Identify decomposition units:
   - Pure functions (no dependencies)
   - Types/interfaces
   - Storage layer
   - Services
   - Components
   - Integration points

3. Build dependency graph:
   - Explicit TASK-N references only
   - No cycles (DAG requirement)
   - No prose like "All previous"

4. If blocked, yield `blocked` with details

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

**Traces to:** ARCH-1, DES-2
```

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
