---
name: alignment-producer
description: Validates traceability chain across project artifacts
context: inherit
model: sonnet
skills: ownership-rules
user-invocable: true
role: producer
phase: alignment
---

# Alignment Producer Skill

Validates the traceability chain (REQ -> DES -> ARCH -> TASK) across all project artifacts.

## Workflow Context

- **Phase**: `alignment_produce` (states.alignment_produce)
- **Upstream**: Documentation commit (`documentation_commit`)
- **Downstream**: `alignment_qa` → `alignment_decide` → retry or `alignment_commit` → evaluation phase
- **Model**: sonnet (default_model in workflows.toml)

This skill validates traceability after documentation is complete, before final evaluation.

---

## Workflow: GATHER -> SYNTHESIZE -> PRODUCE

This skill follows the PRODUCER-TEMPLATE pattern.

### 1. GATHER Phase

Collect all artifact files for validation:

1. Read project context (from spawn prompt in team mode, or `[inputs]` in legacy mode) for project directory
2. Query memory for relevant patterns:
   - `projctl memory query "traceability patterns for <domain>"`
   - `projctl memory query "known failures in alignment validation"`
   If memory is unavailable, proceed gracefully without blocking
3. If missing artifacts, send context request to team-lead with queries for:
   - `docs/requirements.md` (REQ-NNN IDs)
   - `docs/design.md` (DES-NNN IDs with `**Traces to:**`)
   - `docs/architecture.md` (ARCH-NNN IDs with `**Traces to:**`)
   - `docs/tasks.md` (TASK-NNN IDs with `**Traceability:**`)
3. Proceed to SYNTHESIZE when all artifacts gathered

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

Generate validation report and send results to team lead via `SendMessage`:
- Artifact path
- Summary statistics (total, linked, orphan, unlinked counts)
- Chain coverage percentage
- Specific issues found

#### Validation Report Structure

The report includes:
- Summary statistics (total IDs, linked, orphan, unlinked counts)
- Chain coverage analysis (which chains are complete)
- Specific issues with file locations and suggested fixes
- Recommendations for repair

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

---

## Communication

### Team Mode (preferred)

| Action | Tool |
|--------|------|
| Read existing docs | `Read`, `Glob`, `Grep` tools directly |
| Run projctl commands | `Bash` tool directly |
| Report completion | `SendMessage` to team lead |
| Report blocker | `SendMessage` to team lead |

---

## Contract

```yaml
contract:
  outputs:
    - path: ".claude/context/alignment-report.toml"
      id_format: "N/A"

  traces_to:
    - "docs/requirements.md"
    - "docs/design.md"
    - "docs/architecture.md"
    - "docs/tasks.md"

  checks:
    - id: "CHECK-001"
      description: "All expected artifact files were analyzed"
      severity: error

    - id: "CHECK-002"
      description: "ID extraction used correct patterns (REQ-N, DES-N, ARCH-N, TASK-N)"
      severity: error

    - id: "CHECK-003"
      description: "Orphan IDs detection accurate (cross-file validation)"
      severity: error

    - id: "CHECK-004"
      description: "Unlinked IDs detection accurate"
      severity: error

    - id: "CHECK-005"
      description: "Chain direction is correct (downstream traces to upstream)"
      severity: error

    - id: "CHECK-006"
      description: "Suggested fixes are actionable and specific"
      severity: error

    - id: "CHECK-007"
      description: "Domain boundary violations correctly identified"
      severity: warning

    - id: "CHECK-008"
      description: "Coverage metrics included"
      severity: warning
```
