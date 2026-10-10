# Specification Lifecycle Guide

This guide defines how change-scoped product and technical specifications become canonical,
long-lived repository specifications. It supplements [`docs/AGENTS.md`](./AGENTS.md) and the
product, technical, and implementation-plan authoring guides.

## Purpose

The repository separates proposed change context from current normative truth:

- `docs/changes/` contains the product specification, technical specification, implementation plan,
  decisions, and other artifacts for active changes.
- `docs/specs/product/` contains current user-facing product behavior.
- `docs/specs/technical/` contains current technical contracts.

Canonical specifications are maintained views of current behavior, not archives of completed
changes or snapshots owned by one issue.

## Change-Scoped Deltas

A change-scoped product or technical specification MUST describe its normative effect on current
canonical truth as one or more explicit operations:

- **Added:** Assign a new stable ID beneath the introducing change and provide the complete
  normative block.
- **Modified:** Assign the operation an ID beneath the current change, identify exactly one
  canonical target ID, and provide the target's complete replacement text while preserving the
  target ID.
- **Removed:** Assign the operation an ID beneath the current change and identify exactly one
  canonical target ID to remove.

Renames and relocations of addressable nodes are modifications, not removal followed by addition.

A materially new addressable requirement or scenario MUST receive a new ID. Existing addressable
IDs MUST remain stable through modification, promotion, rename, and relocation. Removed IDs MUST
NOT be reused.

Capabilities, components, and their canonical files are semantic organizational containers, not
addressable nodes. Renaming or relocating a container is a semantic reorganization; the IDs of its
contained requirements and scenarios MUST remain stable.

## Canonical Specification Content

Canonical specifications MUST contain only current normative behavior, its Gherkin scenarios, and
the concise context needed to understand that behavior.

Do not promote problem statements, implementation plans, implementation decisions, option
analysis, diagrams, reviews, impact analysis, or other change history into canonical
specifications.

### Rationale at Promotion

A change MAY carry a `Rationale:` paragraph under the requirement it justifies. At promotion:

- Keep the reason the behavior is the way it is, in one or two sentences.
- Remove history: dates, what was removed or replaced and when, measurements and evaluation
  results, how the decision was reached, and rejected alternatives. Git history and the external
  issue preserve them.
- Drop the paragraph entirely when nothing remains after that removal.

A canonical specification MAY contain addressable nodes introduced by multiple changes.
Organizational paths and canonical containers do not replace or rewrite those nodes' stable IDs.
Canonical filenames and titles MUST be semantic and MUST NOT use change artifact or Luhmann IDs.

### Product Specifications

Promote current product requirements and their UAT scenarios into `docs/specs/product/`.

Each canonical product file MUST represent exactly one user-facing capability. Its filename and
title MUST use the semantic capability name without an issue or Luhmann prefix. The filename MUST
be kebab-case, and the title MUST end with ` - Product Spec`:

```text
docs/specs/product/reporting-period-selection.md
```

A capability file MAY contain requirements and scenarios introduced by multiple changes.

When absence is an enduring product contract, preserve it as an explicit **MUST NOT** requirement
rather than relying only on removal.

### Technical Specifications

Promote current technical requirements and their Gherkin scenarios into
`docs/specs/technical/`.

Each canonical technical file MUST represent exactly one stable component that qualifies under
[`docs/TECHNICAL-SPEC-GUIDE.md`](./TECHNICAL-SPEC-GUIDE.md). Its filename and title MUST use the
semantic component name without an issue or Luhmann prefix. The filename MUST be kebab-case, and
the title MUST end with ` - Technical Spec`:

```text
docs/specs/technical/terminal-argument-adapter.md
docs/specs/technical/reporting-period-policy.md
```

A component file MAY contain requirements and scenarios introduced by multiple changes.
Interaction requirements MUST be placed in the file for the component that owns the interaction.
Each requirement has exactly one owning component, and components that qualify separately MUST have
separate canonical files.

Every canonical technical requirement MUST identify the stable product scenario IDs it supports
and MUST NOT duplicate canonical product scenario text.

## Atomic Promotion

Promotion MUST occur in the same completed integration as the implementation and tests:

1. Confirm the implemented behavior and executable tests match the approved change specifications.
2. Apply all product deltas to `docs/specs/product/`.
3. Apply all technical deltas to `docs/specs/technical/`.
4. Validate stable IDs, product-to-technical traceability, Gherkin coverage, and documentation
   links.
5. Delete the completed change directory from `docs/changes/`.

The repository revision produced by that integration MUST contain the completed implementation,
its tests, updated canonical specifications, and no duplicate completed-change copy.

## History and Conflicts

The repository MUST NOT retain a separate completed-change archive. Git history and the external
issue preserve the deleted change artifacts and their rationale.

Authors resolve overlapping edits to canonical specifications through the normal version-control
workflow, just as they resolve overlapping code changes. The documentation lifecycle does not add
a separate locking, blocking, or merge protocol.

## Completion Checklist

Before completing a change, verify:

- Every normative change is represented as an Added, Modified, or Removed operation.
- Every modification or removal names exactly one canonical target ID.
- Added addressable nodes have new, stable issue-rooted IDs.
- Modified, renamed, moved, and promoted addressable nodes preserve their existing IDs.
- Canonical capability and component filenames and titles are semantic and have no change artifact,
  issue, or Luhmann prefix.
- Each canonical product file represents one capability and contains only its current product
  requirements and UAT.
- Each canonical technical file represents one component that qualifies under
  [`docs/TECHNICAL-SPEC-GUIDE.md`](./TECHNICAL-SPEC-GUIDE.md) and contains only its current
  technical requirements and scenarios.
- Every canonical technical requirement has a direct contract test for the component that owns
  that requirement: the caller for a caller interaction requirement, and the called unit for a
  called-component behavior requirement.
- Canonical technical requirements retain valid product-scenario references.
- Canonical specifications match behavior present in the completed repository revision.
- Promoted rationale states only why the behavior is the way it is; its history was removed.
- Product and technical deltas are promoted in the same integration as implementation and tests.
- The completed change directory is deleted.
- No completed-change archive or duplicate normative copy remains.
