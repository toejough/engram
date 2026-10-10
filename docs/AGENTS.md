# Documentation Guidance

## Purpose and Scope

This file defines the documentation conventions for files under `docs/`. Apply these rules
prospectively. Do not reorganize or renumber existing documentation unless the user explicitly
requests a migration.

The following repository-governance files are unnumbered exceptions. They do not participate in the Luhmann hierarchy that they define:

- `docs/AGENTS.md`.
- [`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md).
- [`docs/PRODUCT-SPEC-GUIDE.md`](./PRODUCT-SPEC-GUIDE.md).
- [`docs/TECHNICAL-SPEC-GUIDE.md`](./TECHNICAL-SPEC-GUIDE.md).
- [`docs/IMPLEMENTATION-PLAN-GUIDE.md`](./IMPLEMENTATION-PLAN-GUIDE.md).
- [`docs/GLOSSARY.md`](./GLOSSARY.md) — the central glossary of project terms.
- [`docs/ROADMAP.md`](./ROADMAP.md) — the central record of planned and parked work.

## General Documentation Rules

- Optimize for concision and clarity.
- Document only confirmed scope. Do not introduce plausible assumptions, features, constraints, or decisions that the user did not specify.
- Resolve material ambiguity with the user before writing. Ask one focused question at a time.
- Use concrete language and examples where they remove ambiguity.
- Add newly confirmed shared conventions to this file. Add workflow-specific conventions to the applicable authoring guide.

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
  images/
```

- `docs/changes/` contains active change artifacts.
- `docs/specs/product/` contains exactly one canonical specification file per user-facing
  capability.
- `docs/specs/technical/` contains exactly one canonical specification file per stable component.
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
  technical filename MUST be a semantic kebab-case component name.
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
Bad:  This reuses [792a21](./792a_product-spec.md#792a21-added-parent-sync).
Good: This reuses [792a21 parent-sync](./792a_product-spec.md#792a21-added-parent-sync).
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
- No placeholders, contradictions, duplicate rules, or ambiguous statements remain.
- The final document is as concise as clarity permits.
