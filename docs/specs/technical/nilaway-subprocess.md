# NilAway Subprocess - Technical Spec

Starts the NilAway analyzer process for a configured request, inheriting the developer's
environment. Choosing the arguments is [NilAway Invocation Policy](./nilaway-invocation-policy.md),
and routing entry points to the request is
[Development Check Orchestration](./development-check-orchestration.md).

## 794b4c Inherited environment

Requirement: 794b4c The NilAway subprocess MUST start NilAway with the parent environment
unchanged: it MUST NOT add, change or remove `GOMEMLIMIT`, `GOGC` or `GOMAXPROCS`, and MUST NOT
modify the parent environment.

Supports
[794a2c1a Supplied settings are passed through](../product/development-checks.md#794a2c1-uat-scenarios) and
[794a2c1b Absent settings stay absent](../product/development-checks.md#794a2c1-uat-scenarios).

### 794b4c1 Contract Scenarios

```gherkin
Scenario Outline: 794b4b1c Inherited environment pass-through
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
```
