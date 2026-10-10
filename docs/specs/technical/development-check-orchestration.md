# Development Check Orchestration - Technical Spec

Routes existing nil-check entry points through the configured analyzer invocation while
preserving shared target identity and the inherited environment.

## 794b4a Preserve entry points

Requirement: 794b4a Development-check orchestration MUST replace the behavior of the existing
shared check/fix target objects in place, preserving the target names and all aggregate references.

Supports [794a2a1a Existing entry points use project analysis](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b4a1 Contract Scenarios

```gherkin
Scenario: 794b4a1a Direct and aggregate target routing
  Given Targ has registered its shared development targets
  When Engram initializes its development targets
  Then the existing check and fix target objects execute Engram's configured invocation
  And check-nils retains both configured targets
  And check retains check-nils
  And check-for-fail and check-full retain the configured checking target
  And the direct target names remain check-nils-fix and check-nils-for-fail
```

## 794b4b Execute configured request

Requirement: 794b4b Development-check orchestration MUST send one configured NilAway request to
the command boundary with the supplied context and selected arguments; it MUST return the
boundary's result without retrying. The request MUST carry no resource settings, and the command
adapter MUST run NilAway with the inherited environment unchanged: it MUST NOT add, change or
remove `GOMEMLIMIT`, `GOGC` or `GOMAXPROCS`, and MUST NOT modify the parent environment.

Supports
[794a2a1a Existing check entry points analyze only Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2c1a Supplied settings are passed through](../product/development-checks.md#794a2c1-uat-scenarios), and
[794a2c1b Absent settings stay absent](../product/development-checks.md#794a2c1-uat-scenarios).

### 794b4b1 Contract Scenarios

```gherkin
Scenario Outline: 794b4b1a Successful configured invocation
  Given the invocation mode is <mode>
  When orchestration executes the request
  Then the command boundary receives one request with the supplied context and policy output
  And the request carries no resource settings
  And the boundary succeeds
  And orchestration returns success
  Examples:
    | mode  |
    | check |
    | fix   |

Scenario: 794b4b1b Failed configured invocation
  Given the command boundary returns an analyzer failure
  When orchestration executes the configured check
  Then it returns that failure without retrying

Scenario Outline: 794b4b1c Inherited environment pass-through
  Given the parent environment has an unrelated marker and <resources>
  When the command adapter executes a configured analyzer request
  Then the child environment has the marker and <resources>
  And the child environment has no other resource variable
  And the parent environment is unchanged
  Examples:
    | resources                                  |
    | none of the resource variables             |
    | GOMEMLIMIT=4GiB                            |
    | GOMEMLIMIT=off, GOGC=off, GOMAXPROCS=8     |
    | all three resource variables set to empty  |
