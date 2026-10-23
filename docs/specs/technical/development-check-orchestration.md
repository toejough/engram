# Development Check Orchestration - Technical Spec

Preserves the existing nil-check entry points under their shared target identity. Routing a check
run through the invocation policy and the analyzer subprocess is the
[Nil Check Run Technical Workflow](./nil-check-run-workflow.md), whose interaction requirement is
[794b4b Nil check run](./nil-check-run-workflow.md#794b4b-nil-check-run).
Starting the analyzer process is [NilAway Subprocess](./nilaway-subprocess.md).

## 794b4 Development Check Orchestration

- Ownership: repository-owned component (the `dev` package); it owns
  [794b4a Preserve entry points](#794b4a-preserve-entry-points),
  [861b2a A scoped test run](#861b2a-a-scoped-test-run),
  [861b2b The conformance gate covers every root](#861b2b-the-conformance-gate-covers-every-root),
  [862b2a One command runs every integration-tagged test](#862b2a-one-command-runs-every-integration-tagged-test),
  [862b2b A live-session test without a login is skipped](#862b2b-a-live-session-test-without-a-login-is-skipped) and
  [862b2c A parity run has its own command](#862b2c-a-parity-run-has-its-own-command).
- Responsibility: registers Engram's own nil check and fix targets, its aggregate targets and `watch` in place of
  [Targ](../../GLOSSARY.md#targ)'s shared development targets, under the existing names and with Targ's dependencies,
  order and dependency mode, and registers Targ's other shared targets unchanged. It also registers `test-pkg`, which
  runs the tests of one package ([861b2a A scoped test run](#861b2a-a-scoped-test-run)), `test-integration-all`, which
  runs every integration-tagged test of the module
  ([862b2a One command runs every integration-tagged test](#862b2a-one-command-runs-every-integration-tagged-test)),
  and `test-parity`, which runs the real-model parity tests outside the aggregates
  ([862b2c A parity run has its own command](#862b2c-a-parity-run-has-its-own-command)); the live-session tests it
  registers under `acceptance` skip when no login exists
  ([862b2b A live-session test without a login is skipped](#862b2b-a-live-session-test-without-a-login-is-skipped)).
  Its dev integration tests gate the scenarios of every root
  ([861b2b The conformance gate covers every root](#861b2b-the-conformance-gate-covers-every-root)). As the caller of
  [794b4b Nil check run](./nil-check-run-workflow.md#794b4b-nil-check-run) it obtains the selected mode's arguments from
  the invocation policy, sends one request to the analyzer subprocess and returns its success or its failure without
  retrying.
- Current realization (informational): `registerTargets`, the target variables and `watch` in `dev/targs.go`;
  `testPkg`, `testIntegrationAll`, `testParity` and their target variables in `dev/targs.go`; `scenarioIDs`,
  `scenariosNeedingATest` and the allowlist in `dev/conformance.go` and `dev/conformance_allowlist.go`; the token lookup
  of the live-session tests in `dev/acceptance/token_logic_test.go`; `checkNilsFix`, `checkNilsForFail`, `runNilaway`
  and `runConfiguredNilaway` in `dev/nilaway.go`.
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
- Required behavior: the Go toolchain (external) is assumed to accept `-count`, `-timeout`, `-race`, `-run` and `-tags`
  on `go test`, to select tests by build tag and package pattern, and to exit non-zero when a test fails; authoritative
  contract: `go help test`, not recorded in this repository; compatibility verification: `Test861b2a1d_` runs
  `test-pkg` on a fixture package whose test fails, and `Test862b2a1b_` and `Test862b2a1c_` run
  `test-integration-all` on a fixture module.
- Required behavior: [NilAway Invocation Policy](./nilaway-invocation-policy.md) returns the arguments for the selected
  mode ([794b3a Mode and scope](./nilaway-invocation-policy.md#794b3a-mode-and-scope)).
- Required behavior: [NilAway Subprocess](./nilaway-subprocess.md) starts NilAway with the request's arguments and the
  parent environment unchanged and returns NilAway's success or failure
  ([794b4c Run NilAway](./nilaway-subprocess.md#794b4c-run-nilaway)).
- Supported capabilities: [Development Checks](../product/development-checks.md) (the existing entry points reach
  Engram's nil targets, never Targ's; a developer runs the tests of one package
  ([861a3a1a A developer runs the tests of one package](../product/development-checks.md#861a3a1-uat-scenarios)), the
  full check runs every integration-tagged test
  ([862a3a1a Every integration-tagged test runs in the full check](../product/development-checks.md#862a3a1-uat-scenarios)),
  a check without a login skips
  ([862a3b1a A check without a login skips](../product/development-checks.md#862a3b1-uat-scenarios)), and the parity check
  has a command
  ([862a3c1a A developer runs the parity check](../product/development-checks.md#862a3c1-uat-scenarios))).

### 794b4a Preserve entry points

Requirement: 794b4a Development-check orchestration MUST deregister [Targ](../../GLOSSARY.md#targ)'s shared development targets
and register Engram's own nil check and fix targets under the existing names `check-nils-fix` and
`check-nils-for-fail`. It MUST register its own `check-nils`, `check`, `check-for-fail`,
`check-full`, `mutate` and `watch` under the existing names. Each MUST have the same dependencies,
order and dependency mode as Targ's target of that name, except that Engram's nil and aggregate
targets replace Targ's, Engram's own dev-package checks (`test-dev`, `test-dev-integration`,
`lint-dev`) MAY be appended to the final dependency group, and `check-for-fail` and `check-full`
MUST also depend on `test-integration-all` in that group. `watch` MUST run Engram's `check`. It MUST
re-register Targ's other shared targets unchanged.

Rationale: Targ's aggregates hold direct references to its own nil targets, and its `watch` calls
its own `check`, so replacing the nil targets alone would leave them running Targ's versions;
deregistering and re-registering is Targ's supported way to customize shared targets. The appended
dev-package checks cover Engram's build-tagged tooling code, which Targ's checks never compile, and
`test-integration-all` runs the integration-tagged tests outside `dev/`, which Targ's shared
integration leg, selecting by test name, never runs.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios) and
[862a3a1a Every integration-tagged test runs in the full check](../product/development-checks.md#862a3a1-uat-scenarios).

#### 794b4a1 Contract Scenarios

```gherkin
Scenario: 794b4a1a Direct and aggregate target routing
  Given Targ has registered its shared development targets
  When Engram initializes its development targets
  Then check-nils-fix and check-nils-for-fail run Engram's configured invocation
  And check-nils, check, check-for-fail, check-full, mutate and watch each reach Engram's nil targets
  And none of them reaches Targ's nil targets

Scenario: 794b4a1c Engram's aggregates match Targ's apart from the nil targets
  Given Targ's check-nils, check, check-for-fail, check-full and mutate targets
  When Engram initializes its development targets
  Then each Engram aggregate of the same name has the same dependencies in the same order and mode
  And the only differences are Engram's nil and aggregate targets in place of Targ's
  And Engram's dev-package checks appended to the final dependency group
  And check-for-fail and check-full also depend on test-integration-all in the final dependency group

Scenario: 794b4a1d Targ's other shared targets stay available
  Given Targ has registered its shared development targets
  When Engram initializes its development targets
  Then every Targ shared target other than those Engram replaces is registered unchanged

Scenario: 794b4a1e watch runs Engram's check
  Given Engram has initialized its development targets
  When watch detects a change
  Then it runs Engram's check, not Targ's

Scenario: 794b4a1f No re-registered Targ target runs Targ's nil targets
  Given Engram has initialized its development targets
  When any re-registered Targ shared target runs
  Then it does not reach Targ's nil check or fix targets
```

### 861b2a A scoped test run

Requirement: 861b2a Development-check orchestration MUST register a target `test-pkg` that takes one package path, an optional `--run` regular expression, `--no-race` and an optional `--tags` value, and runs `go test -count=1 -timeout=2m -race` on that package, omitting `-race` with `--no-race`, adding `-tags=<value>` when a value is given and `-run <expression>` when an expression is given. It MUST add no build tag unless `--tags` is given, and it MUST fail with the failure of `go test` when `go test` fails.

Rationale: A developer needs the tests of the package they touched without running the whole module. A per-unit run may skip the race detector and reach the tagged tests of one package, while the aggregate checks keep `-race` and the integration legs.

Supports [861a3a1a A developer runs the tests of one package](../product/development-checks.md#861a3a1-uat-scenarios).

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

Scenario: 861b2a1e --no-race leaves out the race detector
  Given the development targets
  When the developer runs test-pkg with a package path and --no-race
  Then go test is run without -race and with the same flags and the package path otherwise

Scenario: 861b2a1f --tags runs the package under the build tags
  Given the development targets
  When the developer runs test-pkg with a package path and --tags with a value
  Then go test receives -tags=<value> before the filter and the package path
```

### 861b2b The conformance gate covers every root

Requirement: 861b2b Development-check orchestration MUST fail `targ check-full` when a scenario of `docs/specs/product` or `docs/specs/technical`, of any root (an ID of three digits, a letter, then letters and digits), has no top-level test function named `Test<ID>_...` and is not on the allowlist. The allowlist MUST be a list of scenario IDs sorted by ID, each with a reason that names a decision row or an issue. The gate MUST fail on an allowlist entry whose scenario no spec holds or whose scenario has a test, and on an entry whose reason names neither a decision row nor an issue or that is out of order or listed twice. It MUST pin no scenario count, and it MUST fail when a retired scenario ID is in a spec again.

Rationale: A scenario held out stays on the allowlist with its reason until it has a test, and a stale entry fails so the list cannot outlive its reason. The gate checks the form of a reason (a decision-row ID or an issue number), not that the row or issue exists.

#### 861b2b1 Contract Scenarios

```gherkin
Scenario: 861b2b1a A scenario of any root needs a test
  Given scenarios of the roots 792, 793, 855 and 861, some with a test function named for the ID and some without
  When the gate reads them with an empty allowlist
  Then it names each scenario that has no test, whatever its root, in the order of the specs

Scenario: 861b2b1b An allowlisted scenario is not named
  Given scenarios with no test, some of them on the allowlist
  When the gate reads them
  Then it names only the scenarios with no test that are not on the allowlist

Scenario: 861b2b1c An allowlist entry names its reason
  Given an allowlist
  When the gate checks its entries
  Then it names each entry whose reason names neither a decision row nor an issue and each entry that is out of order or listed twice

Scenario: 861b2b1d A stale entry is named
  Given an allowlist and the scenarios of the specs
  When an entry's scenario has a test or is in no spec
  Then the gate names that entry's ID

Scenario: 861b2b1e No count is pinned
  Given specs that gain or lose scenarios, each with a test
  When the gate reads them
  Then it names none, whatever the number of scenarios
```

### 862b2a One command runs every integration-tagged test

Requirement: 862b2a Development-check orchestration MUST register a target `test-integration-all` that runs `go test -count=1 -timeout=10m -race -tags=targ,integration ./...`. It MUST select tests by build tag and package pattern only, never by test name, and it MUST fail with the failure of `go test` when `go test` fails.

Rationale: [Targ](../../GLOSSARY.md#targ)'s shared integration leg selects tests by name, so this command selects them by build tag across the whole module, with the timeout of Targ's whole-module legs. A build tag only adds files, so the untagged tests run a second time, as they do in `test-dev-integration`.

Supports [862a3a1a Every integration-tagged test runs in the full check](../product/development-checks.md#862a3a1-uat-scenarios).

#### 862b2a1 Contract Scenarios

```gherkin
Scenario: 862b2a1a The command selects by tag over the whole module
  Given the development targets
  When the developer runs test-integration-all
  Then go test is run with -count=1, -timeout=10m, -race, -tags=targ,integration and ./..., and with no -run

Scenario: 862b2a1b A tagged test is run whatever its package and name
  Given a module with a test tagged integration that is outside dev and is not named TestIntegration
  When the developer runs test-integration-all on it
  Then that test is run

Scenario: 862b2a1c A failing tagged test fails the command
  Given a module with a tagged test that fails
  When the developer runs test-integration-all on it
  Then the command fails with the failure of go test
```

### 862b2b A live-session test without a login is skipped

Requirement: 862b2b A live-service test MUST be skipped, not failed, when the login it needs does not exist, and its skip message MUST name the environment variable and the credential file that supply the login.

Rationale: The [Conventions Guide](../../CONVENTIONS-GUIDE.md) asks for a clear skip when a required endpoint or credential is missing; a failure reads as a product failure.

Supports [862a3b1a A check without a login skips](../product/development-checks.md#862a3b1-uat-scenarios).

#### 862b2b1 Contract Scenarios

```gherkin
Scenario: 862b2b1a A missing login skips the test
  Given neither the token variable CLAUDE_CODE_OAUTH_TOKEN nor the credential file ~/.config/claude-key.env supplies a login
  When a live-session test asks for the login
  Then the test is skipped with a message that names the variable and the file, and it does not fail

Scenario: 862b2b1b A login that exists is used
  Given the token variable CLAUDE_CODE_OAUTH_TOKEN is set
  When a live-session test asks for the login
  Then the test continues with that token and is not skipped
```

### 862b2c A parity run has its own command

Requirement: 862b2c Development-check orchestration MUST register a target `test-parity` that runs `go test -count=1 -tags=parity ./internal/embed`, and `check-for-fail` and `check-full` MUST NOT depend on it. It MUST fail with the failure of `go test` when `go test` fails.

Rationale: The parity test loads the bundled model and would slow every full check, so it has a command of its own that a developer runs when the embedder changes.

Supports [862a3c1a A developer runs the parity check](../product/development-checks.md#862a3c1-uat-scenarios).

#### 862b2c1 Contract Scenarios

```gherkin
Scenario: 862b2c1a The parity command line is pinned
  Given the development targets
  When the developer runs test-parity
  Then go test is run with -count=1, -tags=parity and ./internal/embed, and with no other flag

Scenario: 862b2c1b The aggregates do not run it
  Given the development targets
  When the dependencies of check-for-fail and check-full are listed
  Then test-parity is in neither
```
