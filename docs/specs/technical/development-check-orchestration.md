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

Requirement: 794b4b Development-check orchestration MUST obtain the arguments for the selected mode
from the invocation policy, then send one request containing only the supplied context and those
arguments to the NilAway subprocess.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios) and
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b4b1 Contract Scenarios

```gherkin
Scenario Outline: 794b4b1a One request carries the context and the policy's arguments
  Given the invocation mode is <mode>
  When orchestration executes the configured check
  Then the NilAway subprocess receives exactly one request
  And the request contains only the supplied context and the policy's arguments for <mode>
  Examples:
    | mode  |
    | check |
    | fix   |
```

## 794b4d Successful result

Requirement: 794b4d When the NilAway subprocess returns success, development-check orchestration
MUST return success to the invoking entry point.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b4d1 Contract Scenarios

```gherkin
Scenario: 794b4d1a A successful run returns success
  Given the NilAway subprocess returns success
  When orchestration executes the configured check
  Then orchestration returns success
```

## 794b4e Failed result

Requirement: 794b4e When the NilAway subprocess returns a failure, development-check orchestration
MUST return that failure to the invoking entry point and MUST NOT retry.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b4e1 Contract Scenarios

```gherkin
Scenario: 794b4e1a A failed run returns the failure without retrying
  Given the NilAway subprocess returns an analyzer failure
  When orchestration executes the configured check
  Then orchestration returns that failure
  And the NilAway subprocess receives no second request
```
