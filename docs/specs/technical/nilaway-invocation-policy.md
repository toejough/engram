# NilAway Invocation Policy - Technical Spec

Constructs first-party analyzer arguments and resolves subprocess resource settings.

## 794b3a Mode and scope

Requirement: 794b3a The invocation policy MUST produce
`-include-pkgs=github.com/toejough/engram` and `./...` for both modes, adding `-fix` only in fix mode.

Supports [794a2a1a Existing entry points use project analysis](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b3a1 Contract Scenarios

```gherkin
Scenario Outline: 794b3a1a Scope and mode arguments
  Given the invocation mode is <mode>
  When the invocation policy constructs analyzer arguments
  Then the arguments are <arguments>
  Examples:
    | mode  | arguments                                                 |
    | check | -include-pkgs=github.com/toejough/engram ./...               |
    | fix   | -include-pkgs=github.com/toejough/engram -fix ./...          |
```

## 794b3b Environment precedence

Requirement: 794b3b The invocation policy MUST resolve each resource setting independently:
use a present environment value verbatim, including empty values; otherwise use
`GOMEMLIMIT=2GiB`, `GOGC=50`, or `GOMAXPROCS=2` as appropriate.

Supports [794a2b1a Absent settings use subprocess defaults](../product/development-checks.md#794a2b1-uat-scenarios)
and [794a2c1a Supplied settings take precedence](../product/development-checks.md#794a2c1-uat-scenarios).

### 794b3b1 Contract Scenarios

```gherkin
Scenario Outline: 794b3b1a Resolve present and absent settings
  Given the supplied resource settings are <supplied>
  When the policy resolves the NilAway environment
  Then the effective values are <effective>
  Examples:
    | supplied                             | effective                                |
    | none                                 | GOMEMLIMIT=2GiB, GOGC=50, GOMAXPROCS=2    |
    | GOMEMLIMIT=4GiB                       | GOMEMLIMIT=4GiB, GOGC=50, GOMAXPROCS=2     |
    | GOMEMLIMIT=off, GOGC=off, GOMAXPROCS=8 | GOMEMLIMIT=off, GOGC=off, GOMAXPROCS=8     |
    | all three explicitly empty           | all three explicitly empty               |
```
