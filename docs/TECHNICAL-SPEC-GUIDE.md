# Technical Specification Authoring Guide

This governance guide supplements [`docs/AGENTS.md`](./AGENTS.md). Follow the shared documentation and Luhmann rules there.

## Purpose

A technical specification defines the solution space and the selected implementation contract. It MAY require architecture, components, dependencies, interactions, algorithms, ownership boundaries, and other implementation details, but it MUST state their semantic purpose clearly.

The title MUST end with ` - Technical Spec`.

The specification MUST stand alone as the first authoritative introduction to technical decisions that are absent from the product specification. Do not depend on discarded analyses or unexplained prior decisions.

## Authoring Workflow

Before drafting:

1. Start from an approved product specification or otherwise confirmed product behavior.
2. Inspect the existing system rather than inferring it from memory.
3. For each product requirement and UAT scenario, identify the relevant existing components, behaviors, interactions, and aggregate behavior.
4. Determine whether the existing system is sufficient and explain why.
5. Enumerate confirmed gaps without introducing unrequested product behavior.
6. Interview the user until the solution space, decision objectives, acceptable dependencies, and material tradeoffs are reasonably contained.
7. Define evaluation measures before selecting or optimizing an option.
8. Present the proposed technical scope and document structure for approval.

Ask one focused question at a time. Separate the user's goal from a proposed mechanism, and confirm consequential design choices rather than resolving them through technical preference.

## Default Structure

Use this progressive-disclosure sequence by default:

1. **Orientation:** product problem, relevant current behavior, insufficiency, and recommended solution.
2. **Recommended sequence diffs:** normative flows, branches, ordering, continuation, and side-effect gating.
3. **Recommended component diffs:** added, removed, updated, retained, and dependency-boundary components.
4. **Recommended component-behavior diffs:** normative behavior owned by each component.
5. **Product-scenario mapping:** each product UAT scenario restated in full at first reference and mapped to technical requirements.
6. **Recommended impact:** repository, dependency, toolchain, operational, risk, and complexity consequences.
7. **Relative option impact:** comparable holistic alternatives evaluated against the agreed objectives.
8. **Alternative options:** concise, similarly structured non-normative summaries.

A section MAY be omitted when it genuinely does not apply. Do not add empty placeholders. If an expected section is omitted and the reason is not obvious, state the reason where a reviewer first needs it.

## Sequence and Interaction Requirements

- In a change specification, use Mermaid sequence diagrams to show relevant final-system interactions and changed or removed behavior.
- Use consistent visual treatment for added, removed, and retained interactions in a change diagram.
- A removal arrow MUST retain the semantic label of the original interaction; do not add redundant prose such as `Remove` when the visual treatment already communicates removal.
- Explanatory removal arrows are non-normative and do not require IDs.
- Each normative flow MUST have one RFC 2119 interaction requirement that states its routing,
  ordering, continuation, gating, value propagation, and every `alt` or `else` branch. Do not split a
  flow's branches into separate requirements.
- Each branch and each materially distinct terminal path MUST have its own Gherkin scenario under the
  flow's requirement. These are the flow's aggregate-flow scenarios.
- A flow requirement MUST NOT duplicate a called component's behavior contract: its scenarios assert
  what the caller propagates and decides, not what the called component returns.
- Canonical diagrams and interaction requirements live only in a flow spec (see Technical Lifecycle
  Rules). A flow spec's diagram shows the current system only, with no added or removed treatment.
- In a flow spec, IDs label arrows only: each arrow carries the ID of the behavior requirement it
  realizes, plus the flow's interaction requirement ID where that requirement constrains the arrow.
  Notes and branch labels carry no IDs. Every ID in a diagram MUST exist, and every interaction
  requirement MUST appear on at least one arrow of its flow's diagram.
- A temporary gating condition MUST be an intermediate assertion, not the final outcome of a valid path.

## Component Qualification

A technical component is a stable area of functionality or meaningful external boundary that owns
an independently stateable contract. It MUST satisfy all of these criteria:

- **Responsibility:** Its behavior can be described without naming its current function, value,
  file, package, or injection mechanism.
- **Ownership:** Its contract is distinct from a caller's routing, ordering, continuation, gating,
  and value-propagation responsibilities.
- **Independent verification:** At least one direct test can exercise its contract without using a
  broader caller merely to prove that the caller invoked it.
- **Stability:** The concept is expected to survive ordinary refactoring of functions, packages,
  files, values, and dependency-injection seams.
- **Necessity:** The contract is required by product behavior or necessary to implement it rather
  than incidental behavior.

Implementation size does not decide component status. A one-function, one-caller unit MAY qualify
when it owns an independent stable contract. A function type, value, file, direct dependency, or
mockable seam MUST NOT qualify solely because it can be named or injected.

A meaningful external-library, operating-system, standard-library, or remote-service boundary MAY
qualify when the application relies on an independent contract at that boundary.

## Component and Behavior Requirements

- Name components as stable capability-level areas of functionality. Describe packages, files, functions, and values separately as their current realization when useful.
- Model direct dependencies and operating-system or standard-library boundaries when repository behavior relies on their contracts.
- Identify ownership explicitly: repository component, repository-owned dependency-boundary interaction, or dependency-internal behavior.
- Every canonical technical requirement MUST have a direct contract test for the component that
  owns that requirement.
- For a caller interaction requirement, the owning component is the caller and the direct test
  exercises its routing, ordering, continuation, gating, and value propagation.
- For a called-component behavior requirement, the owning component is the called unit and the
  direct test exercises what it accepts, does, returns, or fails with without a broader caller
  merely proving invocation.
- Every intended final-system request/response pair or one-way arrow between components MUST have an owning component-behavior requirement and Gherkin scenario. An arrow to or from an external actor, such as the developer, carries the flow's interaction ID and needs no component-behavior requirement.
- A request and its related response MUST share one behavior requirement ID.
- A one-way interaction uses its behavior requirement ID once.
- Component-behavior requirements MUST state what the called component does and returns.
- Lead requirement and arrow wording with semantic purpose. Include framework or API mechanics only when they clarify or constrain the solution.
- Use RFC 2119 language for every normative technical requirement.
- Attach at least one Gherkin scenario to every **MUST** and **MUST NOT** technical requirement.
- Specify behavior that is neither required by a product **MUST** nor necessary to implement one as
  **MAY**, if it is useful to state at all, and do not attach validation scenarios to it.
- Do not create technical requirements, scenarios, or application-owned tests solely to preserve
  incidental dependency behavior.
- Use `Scenario Outline` only when examples follow the same technical interactions and outcome. Split examples whose flows differ.

## Test-Suite Classification

The default `go test ./...` suite MUST contain only pure and deterministic tests. Importing or
exercising a third-party package in process does not by itself make a test an integration test.

Tests that necessarily cross a local impure production boundary MUST use a file-level
`//go:build integration` constraint and run explicitly with:

```bash
go test -tags=integration ./...
```

Local impure boundaries include the real filesystem, process environment, subprocesses and
signals, wall clock, and operating-system descriptors.

Tests that necessarily call a live remote service MUST use a file-level `//go:build network`
constraint and run explicitly with:

```bash
go test -tags=network ./...
```

Network tests MUST skip with a clear reason when their required endpoint or credentials are
absent. The `integration` and `network` tags are independent. Run both optional suites with:

```bash
go test -tags='integration network' ./...
```

Classify a test by the asserted production boundary, not by the package it imports. A test whose
asserted contract requires a live remote service uses `network`, even if incidental local
operating-system plumbing is involved. `integration` is for tests whose asserted production
boundary is local and which do not require a live remote service.

## Product Traceability

- Pull every product UAT scenario through the technical specification.
- Restate each external scenario faithfully and completely at its first mapping location; use its stable ID afterward.
- Preserve the product scenario text modulo formatting whitespace.
- Map each product scenario to all interaction, component, and component-behavior requirements needed to realize it.
- Partition malformed or rejected examples by their actual parser, adapter, policy, or other owning path. Do not assign one example to multiple mutually exclusive paths.
- Preserve product-level **MAY** behavior as implementation freedom unless the user explicitly
  resolves it. A selected design MAY state its current optional behavior, but MUST NOT turn that
  behavior into a scenario-backed technical contract unless it is necessary to implement a product
  **MUST**.

## Technical Lifecycle Rules

Follow [`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md) for shared
delta, stable-identity, canonical-specification, and promotion rules.

- A canonical technical specification file is one of two kinds: a **component spec** for exactly
  one stable component, or a **flow spec** for exactly one flow.
- A component filename MUST be the semantic kebab-case component name, and its title that name
  ending with ` - Technical Spec`. A flow filename MUST be `<flow-name>-flow.md` in kebab-case, and
  its title `<Flow Name> Flow - Technical Spec`. Neither carries an issue or Luhmann prefix.
- A file MAY contain requirements and scenarios introduced by multiple changes.
- A component spec holds behavior requirements only: what the component accepts, does, and
  returns. It MUST NOT hold interaction requirements.
- A flow spec covers one flow, from an entry point to every materially distinct terminal outcome. It
  holds the current-state sequence diagram, the interaction requirements, and the aggregate-flow
  scenarios.
- Interaction requirements belong in the flow spec. Each names its owning (caller) component, which
  has the direct contract test.
- Every behavior requirement belongs in the file of the one component that owns it. A requirement
  that binds two components MUST be split into one requirement per owner. Components that each
  qualify under Component Qualification MUST have separate canonical files; one component file MUST
  NOT hold requirements owned by different components. A flow file is not a component, so this
  rule does not apply to it.
- Every cross-component behavior requirement MUST be drawn in at least one flow spec diagram.
- Do not promote option analysis, change diagrams, impact analysis, reviews, or other change
  context. The current-state diagram in a flow spec is the only diagram promoted.
- Canonical technical requirements MUST list the stable product scenario IDs they support and MUST
  NOT duplicate canonical product scenario text.

## Solution-Space and Option Analysis

For each product requirement:

1. Describe the relevant current components, behaviors, interactions, and aggregate behavior.
2. Decide whether they are sufficient and explain the evidence.
3. Enumerate the gaps.
4. Identify reasonable ways to fill material gaps when the solution space remains open.

Then evaluate proposed changes in aggregate:

- Merge overlapping changes into coherent holistic options.
- Define the optimization objectives before choosing representatives.
- Do not retain an option category when its objective does not distinguish any feasible design.
- Compare the same complete system scope for every option.
- Keep the recommended design normative and alternatives explicitly non-normative.
- Introduce every novel dependency, toolchain change, metric, option, and adoption cost before relying on it.

## Risk and Complexity Measures

When quantitative measures are useful:

- Define each unit and ownership boundary before counting.
- Use the same unit model for every option.
- Count repository-owned sequence units, components, and behavioral clauses consistently.
- Exclude dependency internals when measuring repository-maintained complexity, but include repository-owned interactions across dependency boundaries.
- Distinguish:
  - **Risk:** existing repository-owned units changed or removed.
  - **Change complexity:** repository-owned units added, changed, or removed.
  - **Final complexity:** repository-owned units in the complete relevant final system.
- Decompose totals by sequence units, components, and behavioral clauses so readers can see what drives the result.
- Keep counts as derived comparison evidence, not a substitute for the normative behavior contract.
- State when diagrams use different abstraction levels. Visual size MUST NOT be presented as comparable complexity evidence unless the diagrams cover comparable scopes.
- Report excluded dependency surface and operational tradeoffs separately when they matter to the decision.

## Completion Checklist

Verify:

- The specification stands alone and introduces technical decisions before referencing them.
- The default structure is present or every omission is justified by non-applicability.
- Every product UAT scenario is preserved and mapped.
- Every canonical behavior requirement is located in its owning component file, and every
  interaction requirement in its flow spec, naming its owning component.
- No requirement binds more than one component, and no component file holds requirements owned by
  different components.
- Component filenames and titles follow the component rule; flow filenames and titles follow the
  `<flow-name>-flow.md` and `<Flow Name> Flow - Technical Spec` rule.
- Each flow has one interaction requirement covering every branch, with one scenario per branch and
  terminal path.
- Each flow spec's diagram shows the current system only, carries IDs on arrows only, and uses only
  IDs that exist.
- Every interaction requirement appears in its flow's diagram, and every cross-component behavior
  requirement is drawn in at least one flow.
- Every canonical technical requirement links to its supported canonical product scenario IDs.
- The specification lifecycle checklist passes.
- Every mandatory branch is covered by its flow's interaction requirement and has its own scenario.
- Every mandatory final-system arrow set has an owning component-behavior requirement and scenario.
- Optional behavior is not protected by validation scenarios or application-owned contract tests.
- Branch and component requirements do not duplicate each other's responsibilities.
- Every aggregate flow reaches a meaningful valid or invalid terminal outcome.
- Every component satisfies the component-qualification criteria.
- Every canonical technical requirement has a direct contract test for the component that owns
  that requirement.
- For a caller interaction requirement, the owning component is the caller and the direct test
  exercises its routing, ordering, continuation, gating, and value propagation.
- For a called-component behavior requirement, the owning component is the called unit and the
  direct test exercises what it accepts, does, returns, or fails with without a broader caller
  merely proving invocation.
- Components are capability-level and ownership boundaries are explicit.
- The default `go test ./...` suite contains only pure and deterministic tests.
- Test suites are classified by the asserted production boundary: `network` when the asserted
  contract requires a live remote service, including with incidental local operating-system
  plumbing; otherwise `integration` only when the asserted production boundary is local.
- Semantic purpose is clear without hidden framework knowledge.
- Options use agreed objectives and comparable complete scopes.
- Quantitative counts follow one defined ownership and unit model.
- Abbreviated diagrams are labeled so their size is not mistaken for complexity.
- The shared checklist in [`docs/AGENTS.md`](./AGENTS.md) passes.
