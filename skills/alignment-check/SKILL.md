---
name: alignment-check
description: Verify traceability coverage and consistency across artifacts
context: fork
model: haiku
skills: ownership-rules
user-invocable: true
---

# Alignment Check Skill

Verify that all project artifacts (requirements, design, architecture, tasks) are properly linked and consistent.

## Purpose

Use `projctl trace validate` for mechanical checking of traceability coverage, then interpret results to propose specific fixes for any gaps. This is a lightweight, frequent check that runs after each artifact-producing phase.

## Input

Receives context via `$ARGUMENTS` pointing to a context file (TOML) containing:
- Project directory
- Which phase just completed (to focus interpretation)
- Traceability references

## Traceability Model

**Inline only - NO traceability.toml.** All traceability links are embedded in artifact markdown files via `**Traces to:**` fields.

Format:
```markdown
### DES-001: Design Decision

Description...

**Traces to:** REQ-001, REQ-002
```

## Process

1. **Scan artifacts for IDs and traces**

   Parse artifact files directly to find all artifact IDs and their declared traces:

   **Files to scan:**
   - `docs/requirements.md` - REQ-NNN IDs
   - `docs/design.md` - DES-NNN IDs and `**Traces to:**` fields
   - `docs/architecture.md` - ARCH-NNN IDs and `**Traces to:**` fields
   - `docs/tasks.md` - TASK-NNN IDs and `**Traceability:**` fields
   - `docs/issues.md` - ISSUE-NNN IDs
   - `*_test.go` files - TEST-NNN IDs via `// TEST-NNN` and `// traces:` comments

   **Note:** TEST tracing is in source files, NOT tests.md. Do NOT suggest creating tests.md.

   **ID extraction patterns:**
   - Header format: `### REQ-001: Title` or `## DES-001: Title`
   - Table format: `| REQ-001 |` or `| ID |...| REQ-001 |`
   - Inline references: `REQ-001`, `DES-001`, etc.

   **Traces_to extraction:**
   - Look for `**Traces to:**` or `**Traceability:**` followed by ID references
   - Parse comma-separated IDs with optional descriptions: `REQ-001 (description), REQ-002`

2. **Run mechanical validation**
   ```bash
   projctl trace validate --dir <project-dir>
   ```

3. **Interpret results** - For each gap or issue found:
   - **Orphan ID** (referenced in `**Traces to:**` but not defined): Propose adding the missing ID or fixing the reference
   - **Unlinked ID** (defined but no `**Traces to:**` pointing to it or from it): Determine upstream and add field

4. **Auto-fix gaps by editing artifacts**

   For each unlinked ID or missing coverage:
   - Analyze artifact content to determine the upstream ID
   - If **single clear upstream**: Edit the artifact file to add/update `**Traces to:**` field
   - If **ambiguous** (multiple possible upstreams): Add to escalation list

   ```markdown
   # Edit the artifact directly - example:
   # In design.md, find DES-003 section and add:
   **Traces to:** REQ-005
   ```

5. **Report results**
   - List what `**Traces to:**` fields were added/updated
   - List what needs user decision (ambiguous cases)
   - Never ask for confirmation - just fix and report

## Rules

1. **Run the tool first** - Don't guess at coverage, use `projctl trace validate`
2. **Edit artifacts directly** - Add `**Traces to:**` fields, don't create separate files
3. **Only escalate ambiguity** - If there's one clear upstream, add the link without asking
4. **Be concise** - This runs frequently; keep output focused
5. **Never create traceability.toml** - All links are inline in artifacts

## Domain Boundary Validation

When checking alignment between artifacts, validate that each artifact stays within its domain:

1. **PM artifacts (requirements.md)** should contain only problem-space content:
   - No specific UI descriptions (that's Design)
   - No technology choices (that's Architecture)

2. **Design artifacts (design.md)** should contain only user-interaction content:
   - No problem redefinition (that's PM)
   - No implementation details (that's Architecture)

3. **Architecture artifacts (architecture.md)** should contain only implementation content:
   - No problem redefinition (that's PM)
   - No user-facing decisions (that's Design)

Flag violations as: `DOMAIN_BOUNDARY: [artifact] contains [content type] which belongs in [correct domain]`

## Structured Result

```
Status: success | failure | blocked
Summary: Scanned N artifacts, extracted M IDs. Added K **Traces to:** fields. L gaps remaining.

Artifacts scanned:
  - docs/requirements.md: 12 REQ-NNN IDs
  - docs/design.md: 8 DES-NNN IDs with traces_to fields
  - docs/architecture.md: 6 ARCH-NNN IDs with traces_to fields

Traces added:
  - DES-001: Added **Traces to:** REQ-001, REQ-002
  - ARCH-005: Added **Traces to:** DES-003

Needs decision:
  - ARCH-008: Multiple possible upstreams (DES-003, DES-004)
  - REQ-015: No downstream artifact references this requirement

Validation result:
  total_ids: N
  linked_ids: M
  orphan_ids: [list]
  unlinked_ids: [list]

Traceability: [IDs checked]
```
