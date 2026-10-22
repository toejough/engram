# Scenario Coverage - Technical Spec

This change documents unchanged behavior and adds one development target: it selects no other design and changes no contract, so it has no impact statement, no alternatives and no hypotheses; the options it faced are the rows of [Questions and decisions](./861d_questions-and-decisions.md#questions-and-decisions).

## 861b1 Existing-system orientation

The product problem is [861a1 Problem statement](./861a_product-spec.md#861a1-problem-statement). The software in scope is the development tooling of the `dev` package: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration) owns the registration of Engram's targets in `dev/targs.go`, and its Responsibility lists the targets it registers, so promotion adds `test-pkg` to that list. Today `targ test` takes no argument, runs the whole module with `-race` and a coverage profile, and the Project Profile forbids `go test` outside `targ`; the only target with a package filter, `acceptance`, runs the live-session suite. Insufficiency: a builder cannot run the tests of the package it touched. A local `targ test` run of 2026-10-20 took 24 s for `internal/auto` and 21 s for `internal/cli`, and up to four builders at once mean four concurrent race and coverage builds. Recommendation: one target, `test-pkg`, modeled on `acceptance` ([861b2a A scoped test run](#861b2a-added-a-scoped-test-run)).

## 861b2 Added: Scoped test run

Operations are applied to `docs/specs` at promotion ([861d15 861 merges without promotion](./861d_questions-and-decisions.md#questions-and-decisions)).

### 861b2a Added: A scoped test run

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration), after [794b4a Preserve entry points](../../specs/technical/development-check-orchestration.md#794b4a-preserve-entry-points).

Requirement: 861b2a Development-check orchestration MUST register a target `test-pkg` that takes one package path and an optional `--run` regular expression and runs `go test -count=1 -timeout=2m -race` on that package, adding `-run <expression>` when an expression is given. It MUST add no build tag, and it MUST fail with the failure of `go test` when `go test` fails.

Rationale: A builder needs the tests of the package it touched without the whole module; `acceptance` is the precedent for a `go test` wrapper with a filter, and `targ` is the only sanctioned way to run tests. Tagged `dev` tests keep `test-dev` and `test-dev-integration`.

Supports: none ([861d24 Product purpose of the test-pkg target](./861d_questions-and-decisions.md#questions-and-decisions)).

#### 861b2a1 Contract Scenarios

```gherkin
Scenario: 861b2a1a The package path is required
  Given the development targets
  When the developer runs test-pkg with no package path
  Then it reports the missing package path and starts no go test

Scenario: 861b2a1b A filter narrows the run
  Given the development targets
  When the developer runs test-pkg with a package path and --run with an expression
  Then go test receives -run with that expression before the package path

Scenario: 861b2a1c The command line is pinned
  Given the development targets
  When the developer runs test-pkg with a package path and no --run
  Then go test is run with -count=1, -timeout=2m, -race and that package path, and with no other flag

Scenario: 861b2a1d A failing test fails the target
  Given a package whose tests fail
  When the developer runs test-pkg on it
  Then the target fails with the failure of go test and the developer sees the output of go test
```

## 861b3 Recommended design diffs

Component changes that make a scenario testable by dependency injection are recorded here by the node that makes them, with the commit; none yet ([861d12 Incremental refactoring](./861d_questions-and-decisions.md#questions-and-decisions)).

## 861b4 Operations on the technical specs

Empty until a node finds a documentation problem in a technical spec; each Modified or Added operation is then recorded here and applied in place to `docs/specs` in the commit that adds its test.
