# Development Check Orchestration - Technical Spec

Routes existing nil-check entry points through the configured analyzer invocation while
preserving shared target identity. Starting the analyzer process is
[NilAway Subprocess](./nilaway-subprocess.md).

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

Requirement: 794b4b Development-check orchestration MUST send one NilAway request containing only
the supplied context and the policy's arguments to the command boundary, and MUST return the
boundary's result without retrying.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios) and
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b4b1 Contract Scenarios

```gherkin
Scenario Outline: 794b4b1a Successful configured invocation
  Given the invocation mode is <mode>
  When orchestration executes the request
  Then the command boundary receives one request with the supplied context and policy output
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
```
