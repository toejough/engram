# Implementation Plan Authoring Guide

This governance guide supplements [`docs/AGENTS.md`](./AGENTS.md). Follow the shared documentation
and Luhmann rules there, and follow
[`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md) for canonical
promotion and completed-change removal.

## Purpose

An implementation plan translates approved product and technical contracts into an executable,
status-bearing sequence of test-driven work.

The title MUST end with ` - Implementation Plan`.

An implementation plan MUST:

- Preserve the approved product behavior and technical design.
- Identify implementation dependencies, safe parallel work, and convergence points.
- Point to existing Gherkin contracts instead of restating them.
- Classify each test and explain why that test method fits.
- Track test and work-package progress through verified statuses.
- Guide implementation order without prescribing exact production or test code.

It MUST NOT assume an agent will execute it, prescribe particular skills, duplicate implementation
code already left to the implementer, or add behavior absent from the approved specifications.

## Authoring Workflow

Before drafting:

1. Start from approved product and technical specifications or otherwise confirmed equivalent
   contracts.
2. Inspect the current implementation, tests, dependencies, toolchain, and relevant local changes.
3. Identify the capability boundaries that the selected technical design adds, changes, or retains.
4. Audit the Gherkin scenarios for implementation-plan consistency.
5. Classify each independently valuable test.
6. Group sequential tests into cohesive work-package nodes.
7. Separate independently executable work into distinct nodes.
8. Draw the dependency graph and confirm every edge represents a real prerequisite.
9. Initialize test and node statuses.
10. Present the plan structure, test strategy, dependency graph, and progress model for approval.

Ask one focused question at a time when test ownership, sequencing, parallelism, or a selected test
method remains ambiguous.

## Default Structure

Use this sequence by default:

1. **Purpose:** authoritative source specifications and implementation-plan scope.
2. **Implementation shape:** approved capability boundaries, dependencies, toolchain changes, and
   other confirmed constraints.
3. **Test methods:** classification rules and the shared test-driven implementation cycle.
4. **Test status:** verified test checkpoints and non-test action statuses.
5. **Dependency tree:** Mermaid DAG showing work-package prerequisites and current node status.
6. **Parallel worktree method:** reusable rules for branching, convergence, validation, and cleanup.
7. **Node status:** node-state definitions and transition rules.
8. **Implementation-ordered test catalog:** one section per dependency node.
9. **Completion gates:** measurable conditions for finishing the change.

A section MAY be omitted when it genuinely does not apply. Do not add empty placeholders.

## Contract and Traceability Audit

Before assigning tests or statuses:

- Pull every relevant product and technical Gherkin scenario into the plan by stable ID.
- Give every independently valuable Gherkin scenario its own test entry and status.
- Do not place multiple Gherkin IDs in one test row.
- When multiple scenarios appear to require one test, inspect their ownership and purpose:
  - Consolidate the technical specification if they assert the same independently testable
    contract.
  - Keep both requirements and create separate tests when they protect distinct component,
    dependency-boundary, interaction-branch, aggregate-flow, or acceptance contracts.
- A single Gherkin scenario MAY justify multiple independently valuable tests at different
  boundaries. Name and track those tests separately rather than hiding them behind one status.
- Treat one `Scenario Outline` and its examples as one contract when every example follows the same
  flow and outcome.
- Prefix every corresponding executable test name with the Gherkin scenario's stable ID, such as
  `TestCAP_1434b1a1SelectPriorMonth`. Replace punctuation that is invalid in the programming
  language's identifier syntax with an underscore without changing the documented ID.
  - A scenario outline's table cases MAY keep descriptive names because the complete test path
    inherits the ID from its parent test.
  - Multiple tests MAY share one ID when they verify the same scenario at distinct boundaries.
  - Split a test that verifies multiple scenarios into ID-specific tests or ID-bearing subtests so
    each scenario remains independently traceable.
  - Do not assign a Gherkin ID to a test that does not verify that contract.
- Preserve explicit traceability from every remaining Gherkin contract to at least one
  implementation-plan test.

The implementation plan MUST NOT become a second behavioral specification. Reference the approved
Gherkin by ID and explain test strategy, not scenario prose.

## Test Classification

### Dedicated Examples

Use dedicated named or table-driven examples when the contract is finite or exact, including:

- Accepted and rejected syntax.
- Distinct routing branches.
- Exact output content or destination.
- Named boundary examples whose values document intent.
- Product acceptance scenarios.
- Real dependency behavior tied to a pinned dependency version.

### Property-Based Tests

Property-based tests MUST use [`rapid`](https://github.com/flyingmutant/rapid)
(`pgregory.net/rapid`).

Use them for pure behavior spanning a meaningful input domain, such as calendar relationships,
token expansion, range validity, or result-shape invariants.

Every `rapid` test entry MUST include a **Properties** column that enumerates the guarantees to test.
Properties MUST:

- State externally meaningful invariants or relations.
- Avoid reproducing the production algorithm.
- Define valid generated-input boundaries.
- State failure shape as well as success shape when relevant.
- Supplement rather than replace named Gherkin examples.

### Injected and Impure Interaction Tests

Injection and impure interaction tests MUST use
[`imptest`](https://github.com/toejough/imptest) (`github.com/toejough/imptest`).

Use them when correctness depends on:

- Call count.
- Arguments crossing a component boundary.
- Ordering.
- Returned values.
- Mutually exclusive routing.
- Forbidden calls or side effects.
- Process, filesystem, network, credential, clock, or other impure boundaries.

Use real dependencies in dependency-contract tests. Use `imptest` to isolate repository-owned
coordination around those dependencies. Use ordinary buffers for standard output and standard error.

Do not create broad interfaces around pure functions or stable standard-library values solely to
justify mocking.

## Test-Driven Implementation Cycle

Define the cycle once for the whole plan. For every test:

1. Write the test.
2. Run it and verify that it fails for the expected reason.
3. Implement the minimum change needed to make that test pass.
4. Make any additional corrections needed to keep the complete test suite passing.
5. Refactor while all tests remain green:
   - Reduce complexity.
   - Remove inconsistencies.
   - Structure the code with clear and minimal boundaries.
   - Improve performance where appropriate.

Do not add a separate red-green-refactor objective to every catalog row. The shared cycle governs all
tests; each row should describe the contract, classification, rationale, properties when applicable,
and current status.

## Test Status

Each test MUST have one status recording its latest verified checkpoint:

`Not started` → `Red` → `Green-single` → `Green-all` → `Refactor-complexity` →
`Refactor-consistency` → `Refactor-boundaries` → `Refactor-performance` → `Done`

- **Red:** The test was observed failing for the expected reason.
- **Green-single:** The minimum implementation change made the focused test pass.
- **Green-all:** The complete test suite passes after any required corrections.
- **Refactor-complexity:** Complexity has been reduced where possible and the suite passes.
- **Refactor-consistency:** Inconsistencies have been removed and the suite passes.
- **Refactor-boundaries:** Boundaries are clear and minimal and the suite passes.
- **Refactor-performance:** Performance has been assessed and improved where appropriate and the
  suite passes.
- **Done:** Every checkpoint is complete and the suite passes.

Non-test actions use:

`Not started` → `In progress` → `Done`

Update the status in the same commit that establishes each checkpoint. Do not proceed past an
uncommitted status transition.

## Dependency Tree

- Use a Mermaid directed acyclic graph.
- Use the catalog section's full Luhmann ID as both the rendered node ID and the basis of the Mermaid
  source identifier.
- Give every work-package node one cohesive responsibility.
- Draw an edge only for a real prerequisite.
- Keep top-to-bottom table order as the only sequence within a node.
- If work inside a node can proceed independently, split it into separate dependency-graph nodes
  rather than defining a second parallelization model in the table.
- Show the current node status inside each rendered node.
- Treat the graph as the source of truth for sequencing, parallelism, and convergence.

## Parallel Worktree Method

Use worktrees to materialize independent ready nodes:

- A node is **complete** only when every assigned catalog entry is `Done`, the full suite passes,
  refactoring is complete, and the result is committed.
- A node is **ready** when it is a root node or every direct dependency is complete and their commits
  are present in one base commit.
- Use the development-branch commit as a root node's base.
- A ready node may continue in its dependency's existing branch and worktree when it is that
  dependency's only downstream node.
- When a dependency has multiple downstream nodes, create a separate branch and worktree for each
  downstream node as it becomes ready, even if its siblings remain blocked.
- For a node with one direct dependency, use that dependency's committed result as its base.
- For a node with multiple direct dependencies:
  1. Choose one completed dependency branch as the continuing branch.
  2. Merge the other direct dependency branches into it.
  3. Resolve inconsistencies, run the full suite, and commit the integrated result.
  4. Remove merged worktrees and delete merged branches.
  5. Use the integration commit as the downstream node's base.
- When no incomplete nodes remain, merge each active terminal branch into the development branch,
  validate and commit the integrated result, and clean up intermediate branches and worktrees.

This is execution topology, not generic branch-preparation guidance. Do not duplicate the dependency
graph as a second plan-specific branch tree.

## Node Status

Every dependency-graph node MUST show one status:

- **Blocked by: `<node IDs>`:** A direct dependency is incomplete or the required commits are not yet
  present in one base commit.
- **Ready:** The base commit exists and work has not started.
- **In progress: `n/x entries complete`:** Work has started; `n` catalog entries are `Done` out of
  `x` assigned entries.
- **Complete:** Every entry is `Done`, the complete suite passes, and the node result is committed.
- **Merged:** The result is present in every required downstream base—or, for a terminal node, the
  development branch—and its dedicated branch and worktree have been removed.

Update the graph status in the same commit that establishes the transition.

## Test Catalog

Create one catalog section for each dependency-graph node, using the same Luhmann ID and title.

Within each node:

- Implement and validate entries from top to bottom.
- Use a table with **Gherkin**, **Classification**, **Why**, and **Status** columns for tests.
- Add **Properties** only to tables containing `rapid` tests.
- Use **Work**, **Classification**, **Why**, and **Status** for non-test setup actions.
- Initialize every entry to `Not started`.
- Explain why the assigned test method is necessary; do not describe exact test code.
- Require each corresponding executable test name to begin with the row's Gherkin ID.
- Keep dependency and parallelization information in the graph rather than repeating it in catalog
  prose.

## Completion Gates

Define measurable gates for:

- Complete product and technical Gherkin coverage.
- Dedicated examples for every finite or acceptance contract.
- Explicit properties for every `rapid` test.
- Required call, argument, ordering, and side-effect evidence from `imptest`.
- Real dependency-contract coverage.
- Full-suite, static-analysis, and build success using the approved toolchain.
- Required valid-path behavior.
- Required invalid-path gating and preservation behavior.
- Every catalog entry reaching `Done`.
- Every graph node reaching `Merged`.
- Product and technical deltas applied to their canonical specifications without changing stable
  target IDs.
- Every canonical technical requirement retaining valid product-scenario references.
- The completed change directory deleted in the same merge as its implementation, tests, and
  canonical specification updates.

Do not repeat a generic regression-suite catalog entry when `Green-all` and every refactoring status
already require the full suite to pass.

## Completion Checklist

Verify:

- The plan starts from approved product and technical contracts.
- Every remaining Gherkin scenario has at least one independently tracked test.
- Every corresponding executable test name begins with its Gherkin scenario ID.
- No test row groups multiple Gherkin IDs.
- Apparent requirement duplication was audited and resolved.
- Test classifications match the contract being protected.
- Every `rapid` entry enumerates its properties.
- Every impure collaboration uses `imptest`.
- The shared test cycle appears once and is not duplicated per row.
- Test and non-test statuses use the approved vocabulary.
- Every status transition is committed with its evidence.
- Graph node IDs match catalog Luhmann IDs.
- Every graph edge is a real prerequisite.
- Independent work is represented as separate nodes.
- The worktree method handles fan-out, convergence, delivery, and cleanup.
- The plan does not assume an agent, prescribe skills, duplicate Gherkin prose, or prescribe exact
  implementation code.
- Canonical product and technical specifications describe behavior present in the completed
  repository revision.
- No completed change archive or duplicate normative copy remains.
- The specification lifecycle checklist passes.
- The shared checklist in [`docs/AGENTS.md`](./AGENTS.md) passes.
