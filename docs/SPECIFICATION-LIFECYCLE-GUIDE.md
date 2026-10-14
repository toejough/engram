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

A node that a change placed under the wrong parent, before that change is integrated, was misnumbered
rather than relocated: give it an ID beneath its correct parent. The misplaced ID was never
canonical, so it is not reserved. Relocating an integrated node keeps its ID.

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
specifications, except the current-state sequence diagram in a flow spec.

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
[`docs/TECHNICAL-SPEC-GUIDE.md`](./TECHNICAL-SPEC-GUIDE.md), or exactly one flow. A component
filename and title MUST use the semantic component name without an issue or Luhmann prefix; the
filename MUST be kebab-case and the title MUST end with ` - Technical Spec`. A flow filename MUST
be `<flow-name>-flow.md` in kebab-case, and its title `<Flow Name> Flow - Technical Spec`:

```text
docs/specs/technical/terminal-argument-adapter.md
docs/specs/technical/reporting-period-policy.md
docs/specs/technical/report-generation-flow.md
```

A file MAY contain requirements and scenarios introduced by multiple changes. A component file holds
behavior requirements only; interaction requirements MUST be placed in the flow spec, each naming
its owning (caller) component. In a component file, each requirement has exactly one owning
component, and components that qualify separately MUST have separate canonical files. A flow file
is not a component, so that rule does not apply to it. A flow spec also holds the current-state
sequence diagram and the aggregate-flow scenarios.

Every canonical technical requirement MUST identify the stable product scenario IDs it supports
and MUST NOT duplicate canonical product scenario text.

## Change Phases

A change is one feature-level TDD cycle: red, green, refactor, in that order. Each phase ends with
one agent review round and the owner walkthrough
([Decisions and Open Questions](./AGENTS.md#decisions-and-open-questions)).

### Red

- **Product spec:** user-verifiable promises and their UAT scenarios only, with no mechanism
  ([Product Spec Guide](./PRODUCT-SPEC-GUIDE.md#scope-in-the-red-phase)).
- **Technical spec:** the
  [implementation hypotheses and build rules](./TECHNICAL-SPEC-GUIDE.md#implementation-hypotheses),
  nothing else.
- **Acceptance suite:** executes every UAT scenario literally and MUST fail on every `Then`.
- **Review checks only:** each promise is user-verifiable, each hypothesis is testable, and the
  build rules are stated.

### Green

- **Work:** the minimal implementation that turns the acceptance suite green, and each hypothesis
  experiment run.
- **Results:** reported per clause, with measurements as facts and the build rule's literal output as
  one line. The owner states each conclusion.
- **Gates:** the acceptance suite is green and the existing checks do not break.
- **Review:** one consolidated review at the end of green, checking correctness, security,
  minimality, and standards. Its findings are the refactor's input list; do not review per commit.
- **Eval artifacts** are ephemeral: git history, the decision log, and the owner's conclusion are the
  durable record.

### Refactor

- Behavior is fixed. The acceptance suite MUST stay green throughout and is the only behavioral gate.
- Every standard or technical goal (minimality, performance, complexity, DI, lint, hardening, spec
  conformance) MUST first be made red by a test, an enabled linter, or a measured budget, then worked
  to green ([Refactor Checklist](./TECHNICAL-SPEC-GUIDE.md#refactor-checklist)).
- Write the technical component and flow specs from the refactored system, with each hypothesis
  conclusion as a `Rationale:` line under the requirements it justifies.
- The refactor review precedes promotion.

Across phases, record each decision with who decided it, and report multi-clause hypotheses per clause.

## Atomic Promotion

Promotion MUST occur in the same completed integration as the implementation and tests:

1. Confirm the implemented behavior and executable tests match the approved change specifications.
2. Apply all product deltas to `docs/specs/product/`.
3. Apply all technical deltas to `docs/specs/technical/`, writing each affected flow spec's
   diagram to the final state.
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

- Red produced only user-verifiable promises with UAT, the hypotheses and build rules, and an
  acceptance suite that failed on every `Then`; its review checked only those.
- Green turned the acceptance suite green with the existing checks intact, reported hypothesis
  results per clause with owner-stated conclusions, and held one consolidated review.
- Refactor kept the acceptance suite green, took each standard or technical goal red-first to green,
  wrote the component and flow specs from the refactored system, and passed its review before
  promotion.
- Every normative change is represented as an Added, Modified, or Removed operation.
- Every modification or removal names exactly one canonical target ID.
- Added addressable nodes have new, stable issue-rooted IDs.
- Modified, renamed, moved, and promoted addressable nodes preserve their existing IDs.
- Canonical capability and component filenames and titles are semantic and have no change artifact,
  issue, or Luhmann prefix.
- Each canonical product file represents one capability and contains only its current product
  requirements and UAT.
- Each canonical technical file represents one component that qualifies under
  [`docs/TECHNICAL-SPEC-GUIDE.md`](./TECHNICAL-SPEC-GUIDE.md) or one flow, and contains only its
  current technical requirements and scenarios. Component files hold behavior requirements only;
  interaction requirements and the current-state diagram are in the flow spec.
- Each affected flow spec's diagram shows the final system with no added or removed treatment.
- Every canonical technical requirement has a direct contract test for the component that owns
  that requirement: the caller for a caller interaction requirement, and the called unit for a
  called-component behavior requirement.
- Canonical technical requirements retain valid product-scenario references.
- Canonical specifications match behavior present in the completed repository revision.
- Promoted rationale states only why the behavior is the way it is; its history was removed.
- Product and technical deltas are promoted in the same integration as implementation and tests.
- The completed change directory is deleted.
- No completed-change archive or duplicate normative copy remains.
