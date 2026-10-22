# Scenario Coverage - Technical Spec

This change adds tests for unchanged, documented behavior and one development target: it selects no other design and changes no contract, so it has no impact statement, no alternatives and no hypotheses; the options it faced are the rows of [Questions and decisions](./861d_questions-and-decisions.md#questions-and-decisions).

## 861b1 Existing-system orientation

The product problem is [861a1 Problem statement](./861a_product-spec.md#861a1-problem-statement). The software in scope is the development tooling of the `dev` package: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration) owns the registration of Engram's [Targ](../../GLOSSARY.md#targ) targets in `dev/targs.go`; its Responsibility names its nil, aggregate and watch targets only, so [861b3a Development Check Orchestration](#861b3a-modified-development-check-orchestration) adds the new target to it. Today `targ test` takes no argument, runs the whole module with `-race` and a coverage profile, and the Project Profile forbids `go test` outside `targ`; no target takes a package, and the only target with a test filter, `acceptance` (`--run`), runs the fixed live-session package. Insufficiency: a builder cannot run the tests of the package it touched, and up to four builders at once would run four concurrent race and coverage builds of the whole module. Recommendation: one target, `test-pkg`, modeled on `acceptance` ([861b2a A scoped test run](#861b2a-added-a-scoped-test-run)).

## 861b2 Added: Scoped test run

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the operations below are applied at promotion ([861d15 861 merges without promotion](./861d_questions-and-decisions.md#questions-and-decisions), [861d28 Promotion order of 861, 862 and 858](./861d_questions-and-decisions.md#questions-and-decisions)).

### 861b2a Added: A scoped test run

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration), after [794b4a Preserve entry points](../../specs/technical/development-check-orchestration.md#794b4a-preserve-entry-points).

Requirement: 861b2a Development-check orchestration MUST register a target `test-pkg` that takes one package path, an optional `--run` regular expression, `--no-race` and an optional `--tags` value, and runs `go test -count=1 -timeout=2m -race` on that package, omitting `-race` with `--no-race`, adding `-tags=<value>` when a value is given and `-run <expression>` when an expression is given. It MUST add no build tag unless `--tags` is given, and it MUST fail with the failure of `go test` when `go test` fails.

Rationale: A developer needs the tests of the package they touched without running the whole module. Tagged `dev` tests run through `test-dev` and `test-dev-integration`. A per-unit run skips the race detector and runs the tagged tests of one package; the batch-end gates keep `-race` and the integration legs ([861d400 Per-unit runs skip the race detector and the integration suites](./861d_questions-and-decisions.md#questions-and-decisions)).

Supports [861a3a1a A developer runs the tests of one package](./861a_product-spec.md#861a3a1-uat-scenarios).

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

Scenario: 861b2a1e --no-race leaves out the race detector
  Given the development targets
  When the developer runs test-pkg with a package path and --no-race
  Then go test is run without -race and with the same flags and the package path otherwise

Scenario: 861b2a1f --tags runs the package under the build tags
  Given the development targets
  When the developer runs test-pkg with a package path and --tags with a value
  Then go test receives -tags=<value> before the filter and the package path

Scenario: 861b2a1d A failing test fails the target
  Given a package whose tests fail
  When the developer runs test-pkg on it
  Then the target fails with the failure of go test and the developer sees the output of go test
```

## 861b3 Modified: Development-check orchestration component

The operation replaces the component block of the canonical file completely. It is written on the block as it stands at the base of this change; [862b3c Development Check Orchestration](../862_test-suite-classification/862b_technical-spec.md#862b3c-modified-development-check-orchestration) is written on the block as this operation leaves it.

### 861b3a Modified: Development Check Orchestration

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration). Differs from the current text in Ownership, Responsibility, Current realization and one added Required behavior bullet.

```markdown
## 794b4 Development Check Orchestration

- Ownership: repository-owned component (the `dev` package); it owns
  [794b4a Preserve entry points](#794b4a-preserve-entry-points) and
  [861b2a A scoped test run](#861b2a-a-scoped-test-run).
- Responsibility: registers Engram's own nil check and fix targets, its aggregate targets and `watch` in place of
  [Targ](../../GLOSSARY.md#targ)'s shared development targets, under the existing names and with Targ's dependencies,
  order and dependency mode, and registers Targ's other shared targets unchanged. It also registers `test-pkg`, which
  runs the tests of one package ([861b2a A scoped test run](#861b2a-a-scoped-test-run)). As the caller of
  [794b4b Nil check run](./nil-check-run-workflow.md#794b4b-nil-check-run) it obtains the selected mode's arguments from
  the invocation policy, sends one request to the analyzer subprocess and returns its success or its failure without
  retrying.
- Current realization (informational): `registerTargets`, the target variables and `watch` in `dev/targs.go`;
  `testPkg` and its target variable in `dev/targs.go`; `checkNilsFix`, `checkNilsForFail`, `runNilaway` and
  `runConfiguredNilaway` in `dev/nilaway.go`.
- Qualification: Purpose: the existing check entry points keep their names and reach Engram's nil targets, so that the
  nil check and fix commands report only on Engram's module packages
  ([794a2a Project analysis](../product/development-checks.md#794a2a-project-analysis)). Contract: the names,
  dependencies, order and dependency mode of the registered targets, tested directly against Targ's own targets
  (`dev/nilaway_internal_test.go`, `dev/nilaway_integration_internal_test.go`). Value: Targ's aggregates hold direct
  references to its own nil targets and its `watch` calls its own `check`, so replacing the nil targets alone would
  leave them running Targ's versions; deregistering and re-registering the shared targets together does not.
- Required behavior: Targ (external) is assumed to support customizing its shared development targets by deregistering
  them and registering others under the same names (the Rationale of
  [794b4a Preserve entry points](#794b4a-preserve-entry-points)); authoritative contract: not recorded in this
  repository; compatibility verification: `dev/nilaway_internal_test.go` compares Engram's aggregates with Targ's own
  (`Test794b4a1c`), and `dev/nilaway_integration_internal_test.go` (tags `targ`, `integration`) checks that every Targ
  shared target is kept or replaced (`Test794b4a1d`).
- Required behavior: the Go toolchain (external) is assumed to accept `-count`, `-timeout`, `-race`, `-tags` and `-run` on
  `go test` and to exit non-zero when a test fails; authoritative contract: `go help test`, not recorded in this
  repository; compatibility verification: `Test861b2a1d_` runs `test-pkg` on a fixture package whose test fails.
- Required behavior: [NilAway Invocation Policy](./nilaway-invocation-policy.md) returns the arguments for the selected
  mode ([794b3a Mode and scope](./nilaway-invocation-policy.md#794b3a-mode-and-scope)).
- Required behavior: [NilAway Subprocess](./nilaway-subprocess.md) starts NilAway with the request's arguments and the
  parent environment unchanged and returns NilAway's success or failure
  ([794b4c Run NilAway](./nilaway-subprocess.md#794b4c-run-nilaway)).
- Supported capabilities: [Development Checks](../product/development-checks.md) (the existing entry points reach
  Engram's nil targets, never Targ's; a developer runs the tests of one package, [861a3a1a A developer runs the tests of one package](../product/development-checks.md#861a3a1-uat-scenarios)).
```

## 861b4y Modified: Offer Outbox scenarios

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Offer Outbox](../../specs/technical/offer-outbox.md) in the commit that records it.

### 861b4y1 Modified: 792b36c4a3 The vault ID is stamped before the state directory

Target: [792b36c4a Scenarios](../../specs/technical/offer-outbox.md#792b36c4a-scenarios), Scenario 792b36c4a3. Differs from the current text in the Then line: the location record is the file `.engram/home.json`, so it cannot be stamped before `.engram` exists ([861a4y2 792a21q1g The first offer creates the exchange state in the right order](./861a_product-spec.md#861a4y2-modified-792a21q1g-the-first-offer-creates-the-exchange-state-in-the-right-order) is its product form).

```gherkin
Scenario: 792b36c4a3 The vault ID is stamped before the state directory
  Given a vault with no vault ID and no .engram
  When the first entry is queued
  Then the vault ID is stamped before .engram exists, and its location record is then written into .engram
```

## 861b4zzl Modified: Note Provenance scenarios

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the one below is applied in [Note Provenance](../../specs/technical/note-provenance.md) in the commit that records it.

### 861b4zzl1 Modified: 792b11c7a5 Unrelated content is skipped

Target: [792b11c7a Scenarios](../../specs/technical/note-provenance.md#792b11c7a-scenarios), Scenario 792b11c7a5. Differs from the current text in the Given line: a vocab definition note is a fact note ([vocab definition note](../../GLOSSARY.md#vocab-definition-note)) that the backfill stamps, so the note skipped is the [vocab term-note (retired)](../../GLOSSARY.md#vocab-term-note-retired), the shared stale Given of [861a4zzl1 792a10k1g Other note types are left alone](./861a_product-spec.md#861a4zzl1-modified-792a10k1g-other-note-types-are-left-alone) ([861d153 A vocab definition note is a fact, and the backfill stamps it (792a10k1g)](./861d_questions-and-decisions.md#questions-and-decisions)).

```gherkin
Scenario: 792b11c7a5 Unrelated content is skipped
  Given a note without frontmatter and a vocab term-note (retired)
  When the vault is backfilled
  Then neither fails nor changes, and a read error is returned
```
