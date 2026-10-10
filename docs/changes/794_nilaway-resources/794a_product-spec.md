# 794a NilAway Resource Configuration - Product Spec

## 794a1 Problem Statement

Developers running Engram's checks experience excessive analyzer memory use that interferes with
other applications. They need analysis focused on their project, conservative resource defaults,
and control over those defaults. This change covers the existing check and fix commands; it does
not promise a hard memory ceiling or any particular runtime.

Scope is confirmed by the request to implement recommendations 1 and 3 on issue
[#794](https://github.com/toejough/engram/issues/794), and the subsequent request to apply the
documentation workflow.

## 794a2 Added: development-checks

Canonical destination at integration: `docs/specs/product/development-checks.md`.

### 794a2a Project analysis

Requirement: 794a2a Engram's nil check and fix commands MUST limit substantive nil analysis to
Engram's first-party packages while retaining their existing command names and aggregate-check
entry points.

#### 794a2a1 UAT Scenarios

```gherkin
Scenario Outline: 794a2a1a Existing entry points use project analysis
  Given an Engram checkout with NilAway installed
  When the developer invokes <entry>
  Then each invoked NilAway process analyzes Engram's first-party packages
  And it does not analyze third-party or standard-library package bodies
  Examples:
    | entry               |
    | check-nils-for-fail  |
    | check-nils-fix       |
    | check-nils          |
    | check               |
    | check-for-fail      |
    | check-full          |
```

### 794a2b Resource defaults

Requirement: 794a2b Each NilAway invocation MUST supply a 2 GiB soft Go memory limit, garbage
collection target of 50, and maximum Go execution parallelism of 2 when those settings are absent
from the developer's environment. These defaults MUST apply to the NilAway subprocess without
changing the settings of other checks.

#### 794a2b1 UAT Scenarios

```gherkin
Scenario Outline: 794a2b1a Absent settings use subprocess defaults
  Given none of GOMEMLIMIT, GOGC, or GOMAXPROCS is set
  When the developer invokes <entry>
  Then NilAway receives GOMEMLIMIT=2GiB, GOGC=50, and GOMAXPROCS=2
  And other checks retain their inherited resource settings
  Examples:
    | entry               |
    | check-nils-for-fail  |
    | check-nils-fix       |
```

### 794a2c Developer overrides

Requirement: 794a2c Each NilAway invocation MUST preserve explicitly supplied resource settings,
including empty values, and default only absent settings.

#### 794a2c1 UAT Scenarios

```gherkin
Scenario Outline: 794a2c1a Supplied settings take precedence
  Given the developer supplies <settings>
  When the developer invokes the nil check or fix command
  Then NilAway receives <effective>
  Examples:
    | settings                             | effective                                |
    | GOMEMLIMIT=4GiB                       | GOMEMLIMIT=4GiB, GOGC=50, GOMAXPROCS=2     |
    | GOMEMLIMIT=off, GOGC=off, GOMAXPROCS=8 | GOMEMLIMIT=off, GOGC=off, GOMAXPROCS=8     |
    | all three explicitly empty           | all three explicitly empty               |
```
