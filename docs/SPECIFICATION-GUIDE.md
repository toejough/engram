# Specifying Behavior and Contracts

Specify desired outcomes, observable software promises, and the contracts that realize them. Use
[Conventions: Model and Vocabulary](./CONVENTIONS-GUIDE.md#model-and-vocabulary) for shared meanings,
[Specification Files](./CONVENTIONS-GUIDE.md#specification-files) for names and placement, and
[Stable Luhmann IDs](./CONVENTIONS-GUIDE.md#stable-luhmann-ids) for identity. Follow
[Workflow: Choose the Work Route](./WORKFLOW-GUIDE.md#choose-the-work-route) for baseline checks,
authoring, agreement, execution, and completion. Unchanged behavior follows
[Workflow: Document Existing Behavior](./WORKFLOW-GUIDE.md#document-existing-behavior).

## Choosing a Specification

Use the affected portions of these recipes in a change. They are outlines, not a requirement to
create a new file of every type for every change.

| Recipe | Canonical owner |
| --- | --- |
| [Outcome Specification](#outcome-specification) | One desired business outcome and its success criteria. |
| [Capability Specification](#capability-specification) | One capability's software contribution. |
| [Business Workflow Specification](#business-workflow-specification) | One business workflow's composition contract, when composition needs its own maintained software contract. |
| [Component Specification](#component-specification) | One qualified component's responsibility and boundary contracts. |
| [Technical Workflow Specification](#technical-workflow-specification) | One technical workflow's composition within a declared scope. |

Each canonical product file MUST own exactly one outcome, capability contribution, or business
workflow contract; each canonical technical file owns exactly one qualified component or technical
workflow.
A workflow file is not a component. Keep the catalog flat by default; differently scoped map entries
need not all become files. Technical refinement creates neither matching product hierarchies nor
expanded software scope. Use [Product Traceability](#product-traceability) to connect the views.

A change records its problem and proposed differences; a canonical spec records agreed outcomes,
current promises, and necessary context. Apply [Normative Deltas](#normative-deltas) and
[Canonical Content](#canonical-content) to both specification families.

For all scenarios, use Gherkin `Scenario Outline`/`Examples` only for inputs with the same interactions
and observable outcome; otherwise split scenarios. Assign stable IDs under the conventions above.

## Product Specifications

Product specs define desired outcomes and observable software promises within capabilities and
business workflows. They MUST make participant responsibilities, interactions, and boundary assumptions
explicit. They MUST NOT prescribe architecture, code structure, libraries, algorithms, storage, APIs, rollout, or test
implementation unless the user expands scope.

A change product spec MUST contain only a problem statement, outcome definitions or references,
requirements with context and responsibility allocation, and UAT scenarios, unless the user requests
more. The problem statement MUST identify who experiences what difficulty, absence, or risk and the
outcome needed, without proposing an implementation. Use the applicable recipes below. Software promises follow
[Product Requirements](#product-requirements) and [Acceptance Scenarios](#acceptance-scenarios);
desired business success is defined in the outcome spec.

### Outcome Specification

Describe the desired business result, such as **reduce monthly report preparation time**. Give each
agreed goal in product scope its own canonical file under
[Conventions: Specification Files](./CONVENTIONS-GUIDE.md#specification-files). Record:

1. **Desired result:** beneficiary, intended change, scope, and important external assumptions.
2. **Success criteria:** observable conditions, agreed targets, or qualitative standards defining
   success. Resolve unknown intent with the owner; do not invent targets or infer goals from implementation.

Give the goal and its criteria stable IDs. Contributing capability and workflow specs reference them;
one goal may have many contributors, and each contributor may support many goals. A scenario's
terminal result does not by itself create a separate business goal.

The file defines intent; it is not a log of observations or goal achievement. Software promises still
require acceptance scenarios, but tests of the assigned software contribution do not establish
business success. Changes to agreed goals or criteria use [Normative Deltas](#normative-deltas).

### Capability Specification

Discover the business ability before allocating it: what must a capable person or team be able to
do across situations to achieve the outcomes? Ground it in business concepts. A sequence of steps,
a generic skill such as reading, or a technical mechanism alone is not a capability; this discovery
aid does not imply full automation.

Record:

1. **Ability and purpose.** Link supported outcome IDs directly or through contributing business
   workflows. Include relevant supporting capabilities even when not directly requested.
   Map only what clarifies the change; an optional overview or project glossary may show broader
   relationships.
2. **Allocation and assumptions.** State software responsibilities, including partial contributions,
   and work retained by people or other systems with assumed inputs, decisions, and conditions.
   Several participants may share a capability; allocation alone does not justify splitting it.
3. **Reusable software promises and UAT.** Label others' work as external responsibilities or
   assumptions. Specify confirmed software checks and responses to unmet assumptions or failed
   handoffs. Reference existing owners of shared rules and workflow composition.

Keep this boundary context through promotion. A capability without a software contribution needs no
standalone spec, software UAT, or implementation node; its activities can still contextualize a workflow.

**Split a capability** only for distinct business abilities with separately describable
responsibilities, rules, and standards of competent performance that survive changes in performer
or implementation. Outcomes, steps, and components alone are insufficient. **Stop decomposition**
when responsibilities, rules, and allocation are distinct enough to discuss; another split would
expose workflow or implementation detail rather than a useful business distinction.

For navigational groupings, use [Conventions: Ownership and Scope](./CONVENTIONS-GUIDE.md#ownership-and-scope).

For composition, use [Business Workflow Specification](#business-workflow-specification); both recipes
continue with [Product Requirements](#product-requirements) and [Acceptance Scenarios](#acceptance-scenarios).

### Business Workflow Specification

Use a separate workflow spec when composition needs its own maintained software contract; otherwise
keep relevant workflow context with capability promises. Do not invent a capability to hold composition.

Record in prose, a table, or a diagram:

1. **Scope:** trigger, supported outcome IDs, and relevant completion or alternate results.
2. **Activities:** link capability IDs and name responsible participants and their allocated work.
   Contextual entries can identify human-only capabilities without creating software specs.
   An activity may contribute only part of a capability:
   assessing options exercises professional judgment; clicking a button alone establishes no capability,
   though interface behavior can be a product requirement.
3. **Handoffs and decisions:** transferred information or decisions, meaning, authority, completion
   conditions, and confirmed failure, cancellation, timeout, or recovery behavior.
4. **Composition promises and UAT:** required ordering, decisions, and aggregate software results.
   Reference capability guarantees; distinguish required paths from illustrative sequences, which
   MUST NOT imply product constraints.

Activities, capabilities, and technical realizations need not map one-to-one.

### Product Requirements

Use RFC 2119 keywords deliberately. State observable conditions and results, including promised
measurable limits, without vague qualifiers; separate materially different behaviors. Include
boundaries and failures only when confirmed. Preserve enduring absence as an explicit **MUST NOT**,
not deletion alone.

Split a requirement when different conditions, rules, or outcomes need independent verification;
keep its owner if the responsibility or composition contract is unchanged. Give business rules,
information meaning, state invariants, and quality requirements one capability or workflow owner.
Reference them without copying or forcing them into steps. Workflow scenarios alone do not prove
these constraints; the implementation plan selects suitable verification.

### Acceptance Scenarios

- Every **MUST** and **MUST NOT** MUST have attached Gherkin `Given`/`When`/`Then` scenarios covering
  every mandatory branch, including confirmed happy paths, boundaries, and failures.
- Scenarios MUST be end-user-verifiable through observable application behavior; every `Then` MUST
  be checkable as written. Internal functions, implementation state, source structure, and
  developer-only logs are not criteria.
- External participants may supply prerequisites and actions in `Given` and `When`; `Then` verifies
  the software's promised response, including required handoff checks. Do not claim to verify human
  judgment, external competence, or work outside the software's assigned scope.
- Workflow scenarios cover required sequences, decisions, handoffs, and aggregate results,
  referencing reusable guarantees and shared constraints by ID.
- **MAY** permits freedom and MUST NOT require UAT; do not preserve an optional outcome with scenarios.

### Product Conformance

Check each applicable recipe, including the outcome's owner, criteria, and
[Product Traceability](#product-traceability). For software promises, check
[Product Requirements](#product-requirements), [Acceptance Scenarios](#acceptance-scenarios), and the
distinct capability and specification stopping criteria. Responsibilities and assumptions must be
explicit; mandatory conditions, results, confirmed boundaries, and failures covered; scenarios must
distinguish conformance from violation without choosing
an implementation. Required workflow ordering, decisions, handoffs, and shared constraints must be clear.
Stop specification refinement there unless further detail resolves an ambiguity in the promises.

Keep requirements and scenarios with their capability or workflow owner. Product authors establish
what must be realized without predicting components or matching hierarchies. Product and technical
authors own their respective links under [Product Traceability](#product-traceability).
Do not create a product spec for each component.

[Workflow: Review the Prepared Change](./WORKFLOW-GUIDE.md#review-the-prepared-change) supplies shared
checks and approval timing. Product authoring does not require finished implementation.

## Technical Specifications

A technical spec MAY prescribe architecture, components, dependencies, interactions, algorithms, and
ownership, but MUST explain their semantic purpose. It MUST introduce decisions authoritatively
without relying on discarded analyses or unexplained prior choices.

For a change, establish scope and investigate consequential uncertainty before selecting contracts;
record contracts as boundaries are selected. Existing applicable contracts constrain every phase;
a complete new design is not a prerequisite to building. Follow
[Workflow: Prepare the Change](./WORKFLOW-GUIDE.md#prepare-the-change) and
[Investigate a Consequential Uncertainty](./WORKFLOW-GUIDE.md#investigate-a-consequential-uncertainty)
for that work.

Add applicable content as choices are made; omit empty or irrelevant sections, explaining an unexpected
omission where first needed:

1. **Orientation:** product problem, capabilities and software scope, current behavior, insufficiency,
   and recommendation.
2. **Recommended design diffs:** sequence changes (branches, ordering, continuation, side-effect gating),
   component changes (added, removed, updated, retained, dependency boundaries), and behavior changes
   owned by each component.
3. **Product-scenario mapping:** follow [Product Traceability](#product-traceability).
4. **Impact and alternatives:** repository, dependency, toolchain, operational, risk, and complexity
   consequences of the recommendation, compared with holistic alternatives against agreed objectives.
   Summarize unselected options concisely in a comparable non-normative structure.

Use the applicable portions of the component and technical workflow recipes without inventing owners.
All normative technical requirements use RFC 2119 language; each **MUST**/**MUST NOT** has at least one
Gherkin scenario. Scenario and test mandates exclude **MAY** clauses. Unless behavior is required by
or needed to implement a product **MUST**, state it as **MAY**, if useful, without validation scenarios
or application-owned contract tests. Product **MAY** remains freedom unless the user resolves it;
selecting an option does not remove that freedom. Do not create requirements, scenarios, or
application-owned tests merely to preserve incidental dependency behavior.

Lead requirements and arrows with semantic purpose, adding API or framework mechanics only when
useful or constraining. Use [Conventions: Test-Suite Classification](./CONVENTIONS-GUIDE.md#test-suite-classification)
for verification suites and approved commands.

### Product Traceability

Capability and business workflow records own links to supported outcome IDs, directly or through a
contributing business workflow. Business workflows own activity-to-capability links and participant
allocation. Check that every agreed outcome has contributing capabilities/workflows or an explicit
external responsibility, and every software contribution has a business purpose. Outcome files own
desired results and success criteria, not copies of contributing contracts.

Each component and workflow spec MUST explain its contribution to software capability realization,
linking governing product capabilities or workflows, requirements, and UAT. Supporting components MAY
link through consumer or enclosing contracts, but the chain MUST reach product obligations; an
“infrastructure” label is insufficient. Canonical technical requirements MUST list supported product
scenario IDs. Check both directions: every software promise has a realization, and every normative
technical responsibility has a product purpose. Technical specs own these realization links. Check
the graph in both directions using the authoritative links or backlinks; no duplicated reverse links
or matrix are required. Requirements and scenario IDs connect realizations to tests under
[Workflow: Test Catalog](./WORKFLOW-GUIDE.md#test-catalog).

- Map every UAT scenario to all required interaction, component, and component-behavior requirements.
  In change specs, preserve its full text at first mapping, changing only formatting whitespace;
  use IDs afterward. Canonical specs link without copying product scenario text.
- Preserve allocation and scope. Components/capabilities and technical/business workflows are
  many-to-many; steps and depths need not match. Technical refinement neither changes capabilities
  nor absorbs external responsibilities. Do not invent business workflows for capability-owned
  promises, or technical contracts for unsupported capability-map entries.
- Realize required business ordering, handoffs, authority, and failures, distinguishing constraints
  from illustrative paths. Verify software support and responses, not human judgment or external work.
- Assign malformed or rejected examples to their actual parser, adapter, policy, or other owning
  path, not mutually exclusive paths simultaneously.

### Component Specification

Use product promises, existing contracts, consumer needs, and implementation or experiment evidence
to qualify the boundary, then record its responsibility, contracts, and optional internal view below.

#### Component Qualification

A component MAY qualify only when all four conditions hold:

1. **Purpose:** a technical responsibility required by product promises or needed to implement them.
2. **Contract:** a contract distinct from caller composition, directly testable without a broader
   caller merely proving invocation.
3. **Identity:** survives ordinary refactoring and dependency-injection changes.
4. **Value:** concrete independent reasoning, implementation, verification, change, or meaningful
   risk, ownership, or lifecycle isolation justifies the boundary.

Location, deployment, team, package, process, size, and caller count add no qualification criteria.
Naming or injection alone is insufficient.

**Split** only into qualifying children whose benefit exceeds added interfaces, assumptions,
coordination, and maintenance. Consider independent change, state or lifecycle ownership, failures,
and verification. **Stop** when ownership, assumptions, and contract are clear, the implementation
approach is credible, and further splitting adds no benefit. Keep steps as implementation details
when splitting only renames them, exposes injection, or shifts complexity into communication.

To shorten documentation, split overloaded requirements or improve presentation before creating
components. Size, depth, page count, and scenario count impose no thresholds.

**Record responsibility and ownership.** Qualifying repository-owned components MUST have separate
canonical files. Name the responsibility and capability contribution under [Product Traceability](#product-traceability),
independently of code constructs or injection mechanisms; record those as realizations.
Distinguish repository-owned components, owned dependency-boundary interactions, and dependency-internal
behavior. Reference standard-library and third-party providers through addressable entries and their
authoritative contracts; do not create local specs for their internals. Split a requirement binding
two components into one per owner; a component file MUST NOT own provider or child requirements or
workflow interaction requirements.

#### Component Contracts

Verification timing and adoption gaps follow [Technical Conformance](#technical-conformance).

**Provided behavior.** In the component's file, state its boundary and what it accepts, does, returns,
or fails with in requirements and Gherkin scenarios. Every canonical requirement MUST have a direct
test of that component's contract; a broader caller merely proving invocation is insufficient.

**Required behavior.** In the consumer's file, label dependency assumptions, give Gherkin for each
mandatory behavior (or references), and link selected provider guarantees and any adapter. A
[port](./CONVENTIONS-GUIDE.md#port) states these needs independently of provider internals, including
within one repository. Model direct dependencies and platform boundaries when behavior relies on
them. Authoritative provider requirements stay with their owner. Use a direct connection when
compatible; an extra wrapper is not a qualification rule.

Compatibility covers relied-on inputs, results, errors, ordering, state ownership, lifecycle,
concurrency, durability, and performance. Matching signatures alone is insufficient. An
[adapter](./CONVENTIONS-GUIDE.md#adapter) MUST NOT promise what it and the provider cannot deliver
together. For an unresolved mismatch, change the realization or revisit the assumption and affected
guarantees. Required behavior is an assumption, not something a consumer enforces independently;
identify unsupported assumptions rather than claiming verified guarantees.

Required assumptions MUST identify verification of the provider or adapter/provider at the
consumer-facing boundary. For every affected required contract, pair consumer mocks with real-provider
compatibility verification through any adapter. Pure boundaries may run in process; impure ones use
the appropriate integration or network suite. Conforming doubles prove coordination under assumptions,
not the provider's fulfillment of them.

#### Optional Internal Views

When it explains realization, link child and provider guarantees in text or a table, using components
in the same catalog and under the same criteria. Depth stays local; no matching depths or new product
capabilities. Interaction diagrams belong in scoped technical workflows.

Use [Conventions: Ownership and Scope](./CONVENTIONS-GUIDE.md#ownership-and-scope) to distinguish
contains from uses and keep shared providers shared. Parents own composite guarantees, children
local guarantees: link them and explain composition rather than copying requirements.

### Technical Workflow Specification

**Scope.** Use product promises, participating contracts, and interaction evidence to state the enclosing
system or component, entry point, boundary, and contribution under [Product Traceability](#product-traceability).

**Composition.** One workflow file owns composition within its declared scope, expressed through a
sequence diagram and one RFC 2119 interaction requirement naming its caller. Cover routing, ordering,
continuation, gating, propagation, and every `alt`/`else` branch; do not split branches into separate
requirements. Each branch and materially distinct terminal path MUST have its own aggregate Gherkin
scenario. Every path MUST reach a meaningful valid or invalid terminal outcome; a temporary gate is
intermediate, not the final outcome of a valid path.

**Verification.** Every canonical interaction requirement MUST have a direct caller test under
[Technical Conformance](#technical-conformance). Scenarios assert what the caller decides and
propagates, not the called component's return contract. Components retain local guarantees; reference
them and shared constraints without copying them, including applicable [Product Requirements](#product-requirements).

#### Sequence Diagrams

- Use Mermaid sequence diagrams at the declared scope. Show only necessary participants and
  interactions; components MAY remain opaque. Link detailed internal workflows instead of expanding
  recursively or copying internals into callers.
- Every final-system intercomponent request/response pair or one-way arrow MUST have an owning
  component-behavior requirement and scenario, which MAY be referenced from elsewhere. Conversely,
  every cross-component behavior requirement MUST appear in at least one workflow diagram.
- IDs label arrows only, not notes or branches. A request and response MUST share a behavior ID;
  a one-way interaction uses it once. Add the workflow interaction ID where it constrains the arrow.
  Arrows to/from external participants carry only the interaction ID, needing no component-behavior
  requirement. Every ID MUST exist; each interaction requirement MUST label at least one workflow arrow.
- Change diagrams distinguish added, removed, and retained interactions consistently. Removal arrows
  keep original semantic labels without redundant `Remove` text; they are explanatory, non-normative,
  and need no IDs.
- Canonical technical diagrams appear only in workflow specs; business workflow diagrams belong in
  product specs. Apply [Canonical Content](#canonical-content) when promoting diagrams.

### Technical Conformance

At the current authoring stage, check the applicable recipe, product traceability, component
qualification, contract and diagram rules, and evidence for unresolved choices. Apply the shared
checks in [Workflow: Review the Prepared Change](./WORKFLOW-GUIDE.md#review-the-prepared-change).
For behavior delivery, direct tests and compatibility verification must be complete by consolidation's
end under [Workflow: Consolidate the Change](./WORKFLOW-GUIDE.md#consolidate-the-change); authoring
readiness is not whole-change completion. Documentation-only adoption handles pre-existing gaps under
[Workflow: Close Out and Promote](./WORKFLOW-GUIDE.md#6-close-out-and-promote).

## Normative Deltas

After establishing the baseline, change product and technical specs MUST express every normative
effect as an addressable operation:

| Operation | Content |
| --- | --- |
| `Added` | A new addressable node's complete normative block. |
| `Modified` | One canonical target ID and its complete replacement. Renames and relocations are modifications. |
| `Removed` | One canonical target ID. |

Keep operation IDs distinct from target IDs; preserve stable targets under
[Conventions: Canonical and Enduring IDs](./CONVENTIONS-GUIDE.md#canonical-and-enduring-ids).
Refinement references existing guarantees rather than copying ownership.

## Canonical Content

Retain agreed outcomes and success criteria, current normative behavior, Gherkin, and concise necessary
context under the applicable recipe. Capability and business workflow owners retain promises,
participant allocation, assumptions, handoffs, and shared-rule links. Technical owners retain
component or workflow contracts, ownership,
scenarios, realizations, product traceability, and current provider/adapter links. Applicable boundary
context and unresolved defect/verification-gap issue references are current context, not change history.

Canonical workflows and diagrams show current required behavior, distinguishing illustrative paths.
Business workflow diagrams are optional; technical workflow sequence diagrams are required. Omit
added/removed treatment from final-state diagrams. Exclude problem statements, plans, options, impact
analysis, reviews, historical decisions, evaluations, proposed-state diagrams, and other change context.

A change MAY place `Rationale:` under a requirement. At promotion, retain one or two sentences
explaining why current behavior is specified that way. Remove dates, replaced behavior, measurements,
evaluations, decision history, and rejected alternatives; remove the paragraph if empty. Git and
external issues preserve the evidence.

Canonical placement applies when establishing or promoting the baseline. Follow
[Workflow: Promote Atomically](./WORKFLOW-GUIDE.md#promote-atomically) for validation, integration,
and removal of completed change artifacts.
