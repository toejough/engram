---
name: architect-audit
description: Validate implementation against architecture specification
context: fork
model: sonnet
skills: ownership-rules
user-invocable: true
---

# Architect Audit Skill

Validate implementation against the architecture specification (architecture.md).

## Purpose

Verify that implementation follows the documented layer boundaries, dependency direction, DI patterns, data models, service interfaces, and file structure.

**"Code exists" is not the same as "architecture is followed."**

## Domain Ownership

This skill audits within the **implementation solution space** (same domain as `/architect-interview`).

**Validates:**
- Does implementation follow layer boundaries?
- Are dependency injection patterns correct?
- Do data models and APIs match spec?

**Does NOT validate:**
- Requirements fulfillment → `/pm-audit`
- Visual/interaction design → `/design-audit`

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Path to architecture.md (with ARCH- IDs)
- Project directory
- Traceability references

## Audit Steps

1. **Load architecture spec** - Read architecture.md, extract all ARCH- IDs

2. **For each layer boundary (ARCH IDs):**
   - Trace imports to verify dependencies flow inward
   - Check that domain has no external dependencies
   - Verify services implement interfaces, not concrete types
   - Document any violations

3. **For each data model (ARCH IDs):**
   - Compare implementation against spec
   - Check field types, validation rules
   - Verify relationships are implemented correctly
   - Note any deviations

4. **For each service interface (ARCH IDs):**
   - Verify interface exists and matches spec
   - Check all implementations satisfy the contract
   - Trace DI wiring to composition root
   - Confirm no direct instantiation of implementations

5. **For file structure:**
   - Walk the directory tree
   - Compare against documented structure
   - Identify unexpected files/directories
   - Check naming conventions

6. **For technology decisions (ARCH IDs):**
   - Verify chosen stack is actually used
   - Check for accidental alternative dependencies
   - Confirm patterns are applied consistently

## Audit Categories

| Category | What to Verify |
|----------|----------------|
| **Layer Boundaries** | No imports crossing boundaries incorrectly |
| **Dependency Direction** | Dependencies flow inward (domain has no deps) |
| **DI Pattern** | Interfaces defined, implementations injected |
| **Data Models** | Fields, types, relationships match spec |
| **Service Contracts** | Interfaces implemented correctly |
| **File Structure** | Directory layout matches documented structure |
| **Tech Stack** | Specified technologies used, no surprises |
| **Patterns** | Documented patterns applied consistently |

## Findings Classification

| Classification | Meaning | Action |
|----------------|---------|--------|
| **DEFECT** | Implementation wrong, architecture spec is correct | Fix implementation |
| **SPEC_GAP** | Architecture spec missed something implementation handles well | Propose spec addition |
| **SPEC_REVISION** | Architecture spec was impractical, implementation found better way | Propose spec change |
| **CROSS_SKILL** | Finding affects requirements or design domain | Flag for resolution |

## Structured Result

```
Status: success | failure | blocked
Summary: Audited N architecture decisions. X pass, Y fail, Z proposals.
Findings:
  defects:
    - id: ARCH-NNN
      category: <layer_boundary|di_pattern|data_model|etc>
      description: <what's wrong>
      location: <file:line>
      severity: blocking | warning
      evidence: <import trace, code reference>
  proposals:
    - id: ARCH-NNN
      current_spec: <what arch says>
      proposed_change: <what to change>
      rationale: <why>
  cross_skill:
    - id: ARCH-NNN
      conflicts_with: <REQ-NNN or DES-NNN>
      issue: <description>
Traceability: [ARCH IDs audited]
Layer violations: N
Contract mismatches: N
Recommendation: PASS | FIX_REQUIRED | PROPOSALS_PENDING
```
