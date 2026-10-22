# Test Suite Classification - Technical Spec

This change adds three development behaviors to one component and modifies the component block, one requirement and one scenario of it: it selects no other design, so it has no impact statement, no alternatives and no hypotheses; the options it faced are the rows of [Questions and decisions](./862d_questions-and-decisions.md#questions-and-decisions).

## 862b1 Existing-system orientation

The product problem is [862a1 Problem statement](./862a_product-spec.md#862a1-problem-statement). The software in scope is the development tooling of the `dev` package: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration) owns the registration of Engram's targets in `dev/targs.go` (no spec names `acceptance` today, [862d4 Owner of the new requirements](./862d_questions-and-decisions.md#questions-and-decisions)). Today [Targ](../../GLOSSARY.md#targ)'s shared leg `test-integration`, inside `check-for-fail` and `check-full`, runs `go test -tags=integration -run TestIntegration ./...` and no test has that name; `test-dev-integration` selects by tag but only for `./dev/...`; the `parity` tag of `internal/embed/parity_test.go` has no approved command; and `claudeToken` in `dev/acceptance/session_test.go` ends in `t.Fatalf` when no login exists. Insufficiency: an `integration`-tagged test outside `dev/` would run in no check, the parity test runs in none, and the nine untagged `*_integration_test.go` files cannot be tagged without leaving every gate. Recommendation: one tag-selected command over the whole module in the aggregate checks ([862b2a One command runs every integration-tagged test](#862b2a-added-one-command-runs-every-integration-tagged-test), [862b3a Preserve entry points](#862b3a-modified-preserve-entry-points)), a skip without a login ([862b2b A live-session test without a login is skipped](#862b2b-added-a-live-session-test-without-a-login-is-skipped)) and a parity command outside the aggregates ([862b2c A parity run has its own command](#862b2c-added-a-parity-run-has-its-own-command)). The nine untagged files stay a deviation with a recommendation in [862d10 Item 3: nine integration test files carry no tag](./862d_questions-and-decisions.md#questions-and-decisions).

## 862b2 Added: Integration run, login skip and parity command

Operations are applied to `docs/specs` at promotion ([862d8 862 merges without promotion](./862d_questions-and-decisions.md#questions-and-decisions), [861d28 Promotion order of 861, 862 and 858](../861_scenario-coverage/861d_questions-and-decisions.md#questions-and-decisions)).

### 862b2a Added: One command runs every integration-tagged test

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration), after [794b4a Preserve entry points](../../specs/technical/development-check-orchestration.md#794b4a-preserve-entry-points).

Requirement: 862b2a Development-check orchestration MUST register a target `test-integration-all` that runs `go test -count=1 -timeout=10m -race -tags=targ,integration ./...`. It MUST select tests by build tag and package pattern only, never by test name, and it MUST fail with the failure of `go test` when `go test` fails.

Rationale: Targ's shared integration leg selects tests by name, so this command selects them by build tag across the whole module, with the timeout of Targ's whole-module legs. A build tag only adds files, so the untagged tests run a second time, as they do in `test-dev-integration`.

Supports [862a3a1a Every integration-tagged test runs in the full check](./862a_product-spec.md#862a3a1-uat-scenarios).

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

### 862b2b Added: A live-session test without a login is skipped

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration), after [862b2a One command runs every integration-tagged test](#862b2a-added-one-command-runs-every-integration-tagged-test); the component registers `acceptance`, whose tests need the login.

Requirement: 862b2b A live-service test MUST be skipped, not failed, when the login it needs does not exist, and its skip message MUST name the environment variable and the credential file that supply the login.

Rationale: The Conventions Guide asks for a clear skip when a required endpoint or credential is missing; a failure reads as a product failure.

Supports [862a3b1a A check without a login skips](./862a_product-spec.md#862a3b1-uat-scenarios).

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

### 862b2c Added: A parity run has its own command

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration), after [862b2b A live-session test without a login is skipped](#862b2b-added-a-live-session-test-without-a-login-is-skipped).

Requirement: 862b2c Development-check orchestration MUST register a target `test-parity` that runs `go test -count=1 -tags=parity ./internal/embed`, and `check-for-fail` and `check-full` MUST NOT depend on it. It MUST fail with the failure of `go test` when `go test` fails.

Rationale: The parity test loads the bundled model and would slow every full check, so it has a command of its own that a developer runs when the embedder changes.

Supports [862a3c1a A developer runs the parity check](./862a_product-spec.md#862a3c1-uat-scenarios).

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

## 862b3 Modified: Aggregate checks

Each operation replaces one canonical node completely. The scenarios of [794b4a1 Contract Scenarios](../../specs/technical/development-check-orchestration.md#794b4a1-contract-scenarios) other than [794b4a1c Engram's aggregates match Targ's apart from the nil targets](../../specs/technical/development-check-orchestration.md#794b4a1-contract-scenarios) stay as they are.

### 862b3a Modified: Preserve entry points

Target: [794b4a Preserve entry points](../../specs/technical/development-check-orchestration.md#794b4a-preserve-entry-points). Differs from the current text in the clause on the final dependency group, the last sentence of the Rationale and the Supports list.

```markdown
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
```

### 862b3b Modified: Engram's aggregates match Targ's apart from the nil targets

Target: [794b4a1c Engram's aggregates match Targ's apart from the nil targets](../../specs/technical/development-check-orchestration.md#794b4a1-contract-scenarios). Differs from the current text in the last line.

```gherkin
Scenario: 794b4a1c Engram's aggregates match Targ's apart from the nil targets
  Given Targ's check-nils, check, check-for-fail, check-full and mutate targets
  When Engram initializes its development targets
  Then each Engram aggregate of the same name has the same dependencies in the same order and mode
  And the only differences are Engram's nil and aggregate targets in place of Targ's
  And Engram's dev-package checks appended to the final dependency group
  And check-for-fail and check-full also depend on test-integration-all in the final dependency group
```

### 862b3c Modified: Development Check Orchestration

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration). Differs from the block as [861b3a Development Check Orchestration](../861_scenario-coverage/861b_technical-spec.md#861b3a-modified-development-check-orchestration) leaves it in Ownership, Responsibility, Current realization, the Go toolchain bullet of Required behavior and Supported capabilities ([861d28 Promotion order of 861, 862 and 858](../861_scenario-coverage/861d_questions-and-decisions.md#questions-and-decisions)).

```markdown
## 794b4 Development Check Orchestration

- Ownership: repository-owned component (the `dev` package); it owns
  [794b4a Preserve entry points](#794b4a-preserve-entry-points),
  [861b2a A scoped test run](#861b2a-a-scoped-test-run),
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
  ([862b2b A live-session test without a login is skipped](#862b2b-a-live-session-test-without-a-login-is-skipped)). As
  the caller of [794b4b Nil check run](./nil-check-run-workflow.md#794b4b-nil-check-run) it obtains the selected mode's
  arguments from the invocation policy, sends one request to the analyzer subprocess and returns its success or its
  failure without retrying.
- Current realization (informational): `registerTargets`, the target variables and `watch` in `dev/targs.go`;
  `testPkg`, `testIntegrationAll`, `testParity` and their target variables in `dev/targs.go`; the token lookup of the
  live-session tests in `dev/acceptance/token_logic_test.go`; `checkNilsFix`, `checkNilsForFail`, `runNilaway` and
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
```
