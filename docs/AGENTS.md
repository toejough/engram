# Documentation Guidance

## Purpose and Scope

This file defines the documentation conventions for files under `docs/`. Apply these rules
prospectively. Do not reorganize or renumber existing documentation unless the user explicitly
requests a migration.

## Required Implementation Workflow

These instructions govern every repository behavior change, including code, tooling,
configuration, and agent instructions, even when no documentation files were initially planned.
The root [AGENTS.md](../AGENTS.md#core-principles) requires this workflow repository-wide.

Before editing implementation files:

1. Read [SPECIFICATION-LIFECYCLE-GUIDE.md](./SPECIFICATION-LIFECYCLE-GUIDE.md).
2. Read the [product](./PRODUCT-SPEC-GUIDE.md),
   [technical](./TECHNICAL-SPEC-GUIDE.md), and
   [implementation-plan](./IMPLEMENTATION-PLAN-GUIDE.md) authoring guides applicable to the change.
3. Create or update the required issue-rooted artifacts under `docs/changes/`, following the
   lifecycle guide.
4. Identify the requirements, acceptance scenarios, and implementation-plan nodes the change
   addresses.

Before reporting completion, reconcile those artifacts with the implementation and record
validation results. Code and passing tests alone do not establish completion. Follow the
lifecycle guide's atomic promotion and change-directory removal requirements at integration.

Do not infer an exemption from the change's size or type. Any exemption must be explicitly
provided by the lifecycle guide or the user.

The following repository-governance files are unnumbered exceptions. They do not participate in the Luhmann hierarchy that they define:

- `docs/AGENTS.md`.
- [`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md).
- [`docs/PRODUCT-SPEC-GUIDE.md`](./PRODUCT-SPEC-GUIDE.md).
- [`docs/TECHNICAL-SPEC-GUIDE.md`](./TECHNICAL-SPEC-GUIDE.md).
- [`docs/IMPLEMENTATION-PLAN-GUIDE.md`](./IMPLEMENTATION-PLAN-GUIDE.md).
- [`docs/ADOPTION-GUIDE.md`](./ADOPTION-GUIDE.md).
- [`docs/GLOSSARY.md`](./GLOSSARY.md) — the central glossary of project terms.
- [`docs/ROADMAP.md`](./ROADMAP.md) — the central record of planned and parked work.

## General Documentation Rules

- Optimize for concision and clarity.
- Document only confirmed scope. Do not introduce plausible assumptions, features, constraints, or decisions that the user did not specify.
- Resolve material ambiguity with the user before writing. Ask focused questions, recorded as
  described in [Decisions and Open Questions](#decisions-and-open-questions).
- Use concrete language and examples where they remove ambiguity.
- Add newly confirmed shared conventions to this file. Add workflow-specific conventions to the applicable authoring guide.

## Decisions and Open Questions

Every change directory MUST keep two artifacts beside its specifications:

- A **decision log** (`<key><letter>_decision-log.md`): one ID'd entry per decision made while
  producing the change, each stating the choice, its rationale, the alternatives considered, and
  **who decided it**: the owner (the person who owns the change), the author, or a reviewer. An
  author's or reviewer's interpretation of an owner's answer is its own entry, marked as such, so
  the owner can validate it.
- An **open-questions** file (`<key><letter>_open-questions.md`): one ID'd entry per decision that
  needs the owner, each with context, lettered options, a recommendation, and what it blocks. When
  the owner answers, the entry records the answer and the date and stays in place.

Record decisions and questions in these artifacts as they arise, not afterwards:

- Each review round MUST record its outcome before anything is asked of the owner: findings
  accepted without asking go in the decision log, and questions go in the open-questions file.
- Questions MUST reach the owner only from the open-questions file, numbered by their IDs, never
  only in conversation.
- An owner's answer MUST be recorded in the open-questions file in the same commit that applies it.

## Documentation Structure

Use semantic organizational directories:

```text
docs/
  changes/
    <issue-key>_<change-slug>/
  specs/
    product/
      <capability-slug>
    technical/
      <component-slug>
      <flow-name>-flow
  images/
```

- `docs/changes/` contains active change artifacts.
- `docs/specs/product/` contains exactly one canonical specification file per user-facing
  capability.
- `docs/specs/technical/` contains exactly one canonical specification file per stable component
  (`<component-slug>`) and per flow (`<flow-name>-flow`).
- `docs/images/` contains assets (diagrams, screenshots) that documents and the root `README.md` embed. It holds no specification content.
- Organizational directory names do not establish identity or Luhmann ancestry.
- Repository-governance files remain directly under `docs/`.

## Stable Luhmann IDs

Every addressable change artifact, requirement, scenario, implementation-plan node, test entry,
and delta operation MUST have a stable Luhmann-style ID.
Canonical product capability files and technical component files are semantic organizational
containers, not Luhmann-addressable nodes. Their filenames and titles MUST NOT use issue or
Luhmann prefixes.

### Issue-Rooted Change IDs

- Every tracked change MUST use its external issue key as its immutable root.
- The issue tracker is not prescribed. Numeric keys, project-style keys such as `CAP-1434`, and
  other configured filesystem-safe tracker keys are valid.
- An issue URL MAY be recorded as metadata, but IDs MUST NOT depend on a tracker vendor or URL.
- Change artifacts MUST extend the issue key with alternating lowercase letters and numbers.
- Use descriptive kebab-case text after each file or directory ID.
- Concatenate hierarchy tokens without punctuation. The change between numbers and letters is the
  separator.

```text
docs/
  changes/
    CAP-1434_automatic-reporting-period-selection/
      CAP-1434a_product-spec.md
      CAP-1434b_technical-spec.md
      CAP-1434c_implementation-plan.md
```

### Canonical and Enduring IDs

- A requirement or scenario introduced by a change MUST receive an ID beneath the introducing
  change artifact.
- A canonical product filename MUST be a semantic kebab-case capability name. A canonical
  technical filename MUST be a semantic kebab-case component name or `<flow-name>-flow`.
- IDs MUST remain stable through promotion, revision, rename, and relocation.
- A later change that modifies an existing addressable node MUST reference and preserve the target
  node's ID.
- A materially new addressable node MUST receive a new ID beneath the introducing change.
- Removed IDs MUST NOT be reused.
- A canonical specification MAY contain addressable nodes from multiple issue roots.

```text
docs/specs/product/reporting-period-selection.md
docs/specs/technical/terminal-argument-adapter.md
docs/specs/technical/reporting-period-policy.md
```

### Internal Structure

- A canonical product title MUST use the semantic capability name, and a canonical technical title
  MUST use the semantic component name.
- Canonical titles MUST NOT begin with an issue or Luhmann ID.
- Every addressable heading and numbered item MUST begin with its stable ID.
- New child nodes MUST extend their introducing parent or change-operation ID.
- Do not restart numbering or omit ancestors for brevity.

Example:

```text
# Reporting Period Selection - Product Spec
## CAP-1434a1 Product Requirements
### CAP-1434a1a Default Reporting Period
Requirement: CAP-1434a1a The application MUST select the prior month by default.
#### CAP-1434a1a1 UAT Scenarios
Scenario: CAP-1434a1a1a Select the prior month during the year
```

### References to Addressable Content

- Every reference to an addressable node MUST cite the node's stable ID and MUST be a Markdown
  link to that node. Do not refer to addressable content by name or description alone.
- The link label MUST be the ID followed by the node's title: the text after the ID on the node's
  own heading or `Scenario:` line, without a leading delta-operation word such as `Added:`. Keep
  the title in the label so human readers see what the ID names.
- A reference to a canonical capability or component file MUST link to that file.
- Link to the node's own heading. When the node is not itself a heading, such as a scenario line
  or a table row, link to the nearest heading that contains it.
- Use a relative path, plus the heading anchor: `#<anchor>` within the same file, or
  `<relative-path>#<anchor>` across files.
- The node's own definition (its heading, `Requirement:` line, or `Scenario:` line) is not a
  reference and is not linked.
- Text inside code blocks and diagrams, where links do not render, MUST still cite the ID.

```text
Bad:  This reuses the parent-sync capability.
Good: This reuses [parent-sync](../../specs/product/parent-sync.md).
Bad:  A refused variable fails as in the ENGRAM_SERVER requirement.
Bad:  A refused variable fails as in [792a21a](../../specs/product/parent-sync.md#792a21a-engram_server-is-refused).
Good: A refused variable fails as in [792a21a ENGRAM_SERVER is refused](../../specs/product/parent-sync.md#792a21a-engram_server-is-refused).
```

## Specification Lifecycle

Follow [`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md) when describing
change deltas, maintaining canonical specifications, promoting completed behavior, or deleting
completed change artifacts.

## Authoring Guides

- Follow [`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md) for change
  deltas, canonical specifications, and completed-change promotion.
- Follow [`docs/PRODUCT-SPEC-GUIDE.md`](./PRODUCT-SPEC-GUIDE.md) when creating or revising a product specification.
- Follow [`docs/TECHNICAL-SPEC-GUIDE.md`](./TECHNICAL-SPEC-GUIDE.md) when creating or revising a
  technical specification; it defines technical component boundaries and direct contract-test
  ownership.
- Follow [`docs/IMPLEMENTATION-PLAN-GUIDE.md`](./IMPLEMENTATION-PLAN-GUIDE.md) when creating or revising an implementation plan.
- Follow [`docs/ADOPTION-GUIDE.md`](./ADOPTION-GUIDE.md) when migrating an existing repository's
  documentation into this framework.

## Shared Author Checklist

Before considering a document complete, verify:

- The document contains only confirmed scope.
- New change artifacts use the external issue key as their root.
- Every addressable change artifact, requirement, scenario, implementation-plan node, test entry, and delta operation has one stable, unreused ID.
- Canonical capability and component filenames and titles are semantic and have no issue or
  Luhmann prefix.
- Organizational paths do not create a competing identity hierarchy.
- Every reference to addressable content cites its ID and links to it; every reference to a
  canonical capability or component file links to the file.
- Existing documentation was not renumbered without an explicit migration request.
- The specification lifecycle guide has been followed when the work changes canonical behavior.
- The applicable authoring guide has been followed.
- Every decision is in the change's decision log with who decided it, and every question for the
  owner is in its open-questions file, answered or marked as blocking.
- No placeholders, contradictions, duplicate rules, or ambiguous statements remain.
- The final document is as concise as clarity permits.
