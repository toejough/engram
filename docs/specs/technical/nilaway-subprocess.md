# NilAway Subprocess - Technical Spec

Starts the NilAway analyzer process for a configured request, inheriting the developer's
environment. Choosing the arguments is [NilAway Invocation Policy](./nilaway-invocation-policy.md),
and routing entry points to the request is the
[Nil Check Run Technical Workflow](./nil-check-run-workflow.md).

Its contract is the boundary with a real NilAway process, so its direct contract tests are
integration tests (`//go:build integration`). They start a stand-in `nilaway` on PATH that reports
its arguments and environment, rather than the real analyzer.

- Ownership: repository-owned dependency-boundary interaction (the NilAway process, started by the `dev` package); it
  owns [794b4c Run NilAway](#794b4c-run-nilaway).
- Responsibility: starts NilAway with the request's arguments and the parent environment unchanged, and returns
  NilAway's success or failure, adding, changing or removing none of `GOMEMLIMIT`, `GOGC` and `GOMAXPROCS`.
- Current realization (informational): `executeNilaway` in `dev/nilaway.go`.
- Qualification: Purpose: NilAway runs with the request's arguments
  ([794a2a Project analysis](../product/development-checks.md#794a2a-project-analysis)) and the developer's
  `GOMEMLIMIT`, `GOGC` and `GOMAXPROCS`, unchanged
  ([794a2c Resource settings pass-through](../product/development-checks.md#794a2c-resource-settings-pass-through)).
  Contract: the boundary with a real NilAway process, tested directly by integration tests that start a stand-in
  `nilaway` on PATH.
- Required behavior: NilAway (external) is assumed to act on the arguments and the environment it is started with, to
  exit with success when its run succeeds and with a failure when it reports a finding
  ([794b4c Run NilAway](#794b4c-run-nilaway)); authoritative contract: not recorded in this repository; compatibility
  verification: `dev/nilaway_integration_internal_test.go` (tags `targ`, `integration`) runs the real `nilaway` through
  the entry points, while `dev/nilaway_subprocess_integration_internal_test.go` starts a stand-in, not the real
  analyzer.
- Supported capabilities: [Development Checks](../product/development-checks.md) (the developer's resource settings
  reach NilAway unchanged).

## 794b4c Run NilAway

Requirement: 794b4c The NilAway subprocess MUST start NilAway with the request's arguments and the
parent environment unchanged, and MUST return NilAway's result: success when NilAway succeeds, and
NilAway's failure otherwise. It MUST NOT add, change or remove `GOMEMLIMIT`, `GOGC` or
`GOMAXPROCS`, and MUST NOT modify the parent environment.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2c1a Supplied settings are passed through](../product/development-checks.md#794a2c1-uat-scenarios) and
[794a2c1b Absent settings stay absent](../product/development-checks.md#794a2c1-uat-scenarios).

### 794b4c1 Contract Scenarios

```gherkin
Scenario Outline: 794b4c1a Inherited environment pass-through
  Given the parent environment has an unrelated marker and <resources>
  When the NilAway subprocess executes a configured analyzer request
  Then the child environment has the marker and <resources>
  And the child environment has no other resource variable
  And the parent environment is unchanged
  Examples:
    | resources                                  |
    | none of the resource variables             |
    | GOMEMLIMIT=4GiB                            |
    | GOMEMLIMIT=off, GOGC=off, GOMAXPROCS=8     |
    | all three resource variables set to empty  |

Scenario: 794b4c1b NilAway receives the request's arguments
  Given a request with arguments "-include-pkgs=github.com/toejough/engram ./..."
  When the NilAway subprocess executes the request
  Then NilAway is started with exactly those arguments

Scenario: 794b4c1c A successful run returns success
  Given NilAway exits successfully
  When the NilAway subprocess executes a request
  Then it returns success

Scenario: 794b4c1d A failing run returns NilAway's failure
  Given NilAway exits with a failure
  When the NilAway subprocess executes a request
  Then it returns that failure
```
