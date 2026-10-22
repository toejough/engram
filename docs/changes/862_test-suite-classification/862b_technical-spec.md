# Test Suite Classification - Technical Spec

This change adds two development behaviors to one component and modifies one requirement and one scenario of it: it selects no other design, so it has no impact statement, no alternatives and no hypotheses; the options it faced are the rows of [Questions and decisions](./862d_questions-and-decisions.md#questions-and-decisions).

## 862b1 Existing-system orientation

The product problem is [862a1 Problem statement](./862a_product-spec.md#862a1-problem-statement). The software in scope is the development tooling of the `dev` package: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration) owns the registration of Engram's targets in `dev/targs.go`, including `acceptance`, whose tests are in `dev/acceptance`. Today Targ's shared leg `test-integration`, inside `check-for-fail` and `check-full`, runs `go test -tags=integration -run TestIntegration ./...` and no test has that name; `test-dev-integration` selects by tag but only for `./dev/...`; and `claudeToken` in `dev/acceptance/session_test.go` ends in `t.Fatalf` when no login exists. Insufficiency: an `integration`-tagged test outside `dev/` would run in no check, and the nine untagged `*_integration_test.go` files cannot be tagged without leaving every gate. Recommendation: one tag-selected command over the whole module in the aggregate checks ([862b2a One command runs every integration-tagged test](#862b2a-added-one-command-runs-every-integration-tagged-test), [862b3a Preserve entry points](#862b3a-modified-preserve-entry-points)) and a skip without a login ([862b2b A live-session test without a login is skipped](#862b2b-added-a-live-session-test-without-a-login-is-skipped)). The `parity` tag and the nine untagged files stay deviations, each with a recommendation in [862d9 Item 2: the parity tag has no approved command](./862d_questions-and-decisions.md#questions-and-decisions) and [862d10 Item 3: nine integration test files carry no tag](./862d_questions-and-decisions.md#questions-and-decisions).

## 862b2 Added: Integration run and login skip

Operations are applied to `docs/specs` at promotion ([862d8 862 merges without promotion](./862d_questions-and-decisions.md#questions-and-decisions)).

### 862b2a Added: One command runs every integration-tagged test

Target: [794b4 Development Check Orchestration](../../specs/technical/development-check-orchestration.md#794b4-development-check-orchestration), after [794b4a Preserve entry points](../../specs/technical/development-check-orchestration.md#794b4a-preserve-entry-points).

Requirement: 862b2a Development-check orchestration MUST register a target `test-integration-all` that runs `go test -count=1 -timeout=2m -race -tags=targ,integration ./...`. It MUST select tests by build tag and package pattern only, never by test name, and it MUST fail with the failure of `go test` when `go test` fails.

Rationale: The shared integration leg selects by test name and no test has the name, so an integration-tagged test outside `dev/` runs in no check; `test-dev-integration` selects by tag but only for `./dev/...`. A build tag only adds files, so the untagged tests run a second time, as they already do in `test-dev-integration`.

Supports: none ([862d11 Product purpose of the new requirements](./862d_questions-and-decisions.md#questions-and-decisions)).

#### 862b2a1 Contract Scenarios

```gherkin
Scenario: 862b2a1a The command selects by tag over the whole module
  Given the development targets
  When the developer runs test-integration-all
  Then go test is run with -count=1, -timeout=2m, -race, -tags=targ,integration and ./..., and with no -run

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

Supports: none ([862d11 Product purpose of the new requirements](./862d_questions-and-decisions.md#questions-and-decisions)).

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

## 862b3 Modified: Aggregate checks

Each operation replaces one canonical target completely. The scenarios of [794b4a1 Contract Scenarios](../../specs/technical/development-check-orchestration.md#794b4a1-contract-scenarios) other than [794b4a1c Engram's aggregates match Targ's apart from the nil targets](../../specs/technical/development-check-orchestration.md#794b4a1-contract-scenarios) stay as they are.

### 862b3a Modified: Preserve entry points

Target: [794b4a Preserve entry points](../../specs/technical/development-check-orchestration.md#794b4a-preserve-entry-points). Differs from the current text in the clause on the final dependency group and the sentence of the Rationale on `test-integration-all`.

```markdown
Requirement: 794b4a Development-check orchestration MUST deregister Targ's shared development targets
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
dev-package checks cover Engram's build-tagged tooling code, which Targ's checks never compile.
`test-integration-all` is in the aggregates so that an integration-tagged test outside `dev/` runs in
the checks, which Targ's shared integration leg, selecting by test name, does not do.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../../specs/product/development-checks.md#794a2a1-uat-scenarios) and
[794a2a1c The fix command acts only on Engram's module packages](../../specs/product/development-checks.md#794a2a1-uat-scenarios).
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
