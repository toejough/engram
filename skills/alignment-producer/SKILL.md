---
name: alignment-producer
description: Validates traceability chain across project artifacts
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: alignment
---

# Alignment Producer Skill

Validates the traceability chain (REQ -> DES -> ARCH -> TASK) across all project artifacts.

## Yield Protocol

See [YIELD.md](../shared/YIELD.md) for full protocol specification.

## Workflow: GATHER -> SYNTHESIZE -> PRODUCE

This skill follows the PRODUCER-TEMPLATE pattern.

### 1. GATHER Phase

Collect all artifact files for validation:

1. Read context from `[inputs]` section for project directory
2. If missing artifacts, yield `need-context` with queries for:
   - `docs/requirements.md` (REQ-NNN IDs)
   - `docs/design.md` (DES-NNN IDs with `**Traces to:**`)
   - `docs/architecture.md` (ARCH-NNN IDs with `**Traces to:**`)
   - `docs/tasks.md` (TASK-NNN IDs with `**Traceability:**`)
3. Proceed to SYNTHESIZE when all artifacts gathered

#### need-context Yield

```toml
[yield]
type = "need-context"
timestamp = 2026-02-02T10:00:00Z

[[payload.queries]]
type = "file"
path = "docs/requirements.md"

[[payload.queries]]
type = "file"
path = "docs/design.md"

[[payload.queries]]
type = "file"
path = "docs/architecture.md"

[[payload.queries]]
type = "file"
path = "docs/tasks.md"

[context]
phase = "alignment"
subphase = "GATHER"
awaiting = "context-results"
```

### 2. SYNTHESIZE Phase

Analyze traceability coverage:

1. **Extract all IDs** from each artifact:
   - REQ-NNN from requirements.md
   - DES-NNN from design.md
   - ARCH-NNN from architecture.md
   - TASK-NNN from tasks.md

2. **Extract all traces** from `**Traces to:**` and `**Traceability:**` fields

3. **Identify issues**:
   - **Orphan IDs**: Referenced in traces but not defined anywhere
   - **Unlinked IDs**: Defined but no traces point to or from them
   - **Broken traces**: Invalid ID format or wrong prefix level
   - **Chain gaps**: Missing links in REQ -> DES -> ARCH -> TASK chain

4. **Calculate coverage metrics**:
   - Total IDs per type
   - Linked vs unlinked counts
   - Orphan count
   - Chain completeness percentage

### 3. PRODUCE Phase

Generate validation report and yield complete:

#### Validation Report Structure

The report includes:
- Summary statistics (total IDs, linked, orphan, unlinked counts)
- Chain coverage analysis (which chains are complete)
- Specific issues with file locations and suggested fixes
- Recommendations for repair

#### complete Yield

```toml
[yield]
type = "complete"
timestamp = 2026-02-02T11:00:00Z

[payload]
artifact = ".claude/context/alignment-report.toml"
files_modified = [".claude/context/alignment-report.toml"]

[payload.summary]
total_ids = 45
linked_ids = 38
orphan_ids = 2
unlinked_ids = 5
chain_coverage = 0.84

[[payload.issues]]
type = "orphan"
id = "REQ-99"
location = "docs/design.md:42"
context = "Referenced in DES-005 traces but not defined"

[[payload.issues]]
type = "unlinked"
id = "ARCH-012"
location = "docs/architecture.md:156"
context = "No TASK traces to this architecture decision"

[[payload.issues]]
type = "broken"
id = "DES-003"
location = "docs/design.md:78"
context = "Traces to ARCH-001 (wrong direction - should trace to REQ)"

[context]
phase = "alignment"
subphase = "complete"
```

## Traceability Chain

The expected chain direction is:

```
REQ-NNN (requirements)
    |
    v
DES-NNN (design) -- Traces to: REQ-NNN
    |
    v
ARCH-NNN (architecture) -- Traces to: DES-NNN, REQ-NNN
    |
    v
TASK-NNN (tasks) -- Traceability: ARCH-NNN, DES-NNN
```

Each downstream artifact traces UP to its upstream sources.

## Issue Types

| Type | Description | Example |
|------|-------------|---------|
| `orphan` | ID referenced but not defined | `**Traces to:** REQ-99` but REQ-99 not in requirements.md |
| `unlinked` | ID defined but not traced to/from | ARCH-012 exists but no TASK mentions it |
| `broken` | Invalid trace (wrong direction or format) | DES traces to ARCH instead of REQ |
| `gap` | Missing intermediate link in chain | TASK traces to REQ directly, skipping DES/ARCH |

## Domain Boundary Validation

Also check that artifacts stay in their domain:

- **REQ**: Problem space only (no UI details, no tech choices)
- **DES**: User interaction only (no problem redef, no implementation)
- **ARCH**: Implementation only (no problem redef, no user-facing)

Flag domain violations in the report.
