# Development Check Orchestration - Technical Spec

Routes existing nil-check entry points through the configured analyzer invocation while
preserving shared target identity and isolating child environment settings.

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
the command boundary with the supplied context, selected arguments, and effective child settings;
it MUST return the boundary's result without retrying or modifying the parent environment.
The command adapter MUST apply settings only to the child command while retaining inherited
environment values for unrelated keys.

Supports [794a2a1a Existing entry points use project analysis](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2b1a Absent settings use subprocess defaults](../product/development-checks.md#794a2b1-uat-scenarios), and
[794a2c1a Supplied settings take precedence](../product/development-checks.md#794a2c1-uat-scenarios).

### 794b4b1 Contract Scenarios

```gherkin
Scenario Outline: 794b4b1a Successful configured invocation
  Given the invocation mode is <mode>
  And the environment has a partial explicit resource override
  When orchestration executes the request
  Then the command boundary receives one request with the supplied context and policy output
  And the boundary succeeds
  And orchestration returns success without modifying the parent environment
  Examples:
    | mode  |
    | check |
    | fix   |

Scenario: 794b4b1b Failed configured invocation
  Given the command boundary returns an analyzer failure
  When orchestration executes the configured check
  Then it returns that failure without retrying

Scenario: 794b4b1c Child environment isolation
  Given the parent environment has an unrelated marker and no resource overrides
  When the command adapter executes a configured analyzer request
  Then the child receives the marker and all three resource defaults
  And the parent resource settings remain absent
```
