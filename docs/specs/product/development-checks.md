# Development Checks - Product Spec

## 794a2 Development Checks

**Ability and purpose.** The nil check and fix commands report on, and fix, only Engram's own module packages and
analyze no third-party or standard-library package, because dependency package bodies are most of the analysis cost
and nothing in them is Engram's to fix ([794a2a Project analysis](#794a2a-project-analysis)).

**Participants and responsibilities.**
- The developer: runs a nil check or fix command, directly or through an aggregate check, and sets `GOMEMLIMIT`,
  `GOGC` and `GOMAXPROCS` in their own environment when their machine needs limits
  ([794a2c Resource settings pass-through](#794a2c-resource-settings-pass-through)).
- The software: keeps the existing command names and aggregate-check entry points
  ([794a2a Project analysis](#794a2a-project-analysis)) and passes the developer's resource settings to NilAway
  unchanged ([794a2c Resource settings pass-through](#794a2c-resource-settings-pass-through)).
- Retained by other systems: NilAway (third party) runs as the analysis process of each command once it is installed
  in the checkout; the software assumes it ([794a2a1 UAT Scenarios](#794a2a1-uat-scenarios),
  [794a2c1 UAT Scenarios](#794a2c1-uat-scenarios)).

**Interactions.** The developer hands an entry point to the software by invoking it; the software responds with the
nil dereferences found in Engram's module packages and no finding in a third-party dependency, a module that shares
only the path prefix, or a nested module ([794a2a1 UAT Scenarios](#794a2a1-uat-scenarios)). The developer hands
resource settings to the software through the environment; the software hands them to NilAway as supplied and leaves
an absent setting absent ([794a2c Resource settings pass-through](#794a2c-resource-settings-pass-through)).

**Boundary assumptions.** The software assumes from the developer that NilAway is installed; no response is specified
when it is not ([794a2a1 UAT Scenarios](#794a2a1-uat-scenarios)). It assumes the developer sets limits that suit their
machine; whether a setting is absent, empty or set, it adds, changes and removes nothing
([794a2c Resource settings pass-through](#794a2c-resource-settings-pass-through)).

### 794a2a Project analysis

Requirement: 794a2a The nil check and fix commands MUST report findings only in Engram's module
packages: packages whose import path is Engram's module path or that path followed by `/`. They
MUST NOT analyze third-party or standard-library packages, and MUST NOT report findings in any other
package, such as a package of `github.com/toejough/engram-eval` or of a nested module that has its
own `go.mod`. A dependency whose module path begins with Engram's module path MAY be analyzed. The
commands MUST retain their existing command names and aggregate-check entry points.

Rationale: Dependency package bodies are most of the analysis cost and nothing in them is Engram's
to fix. Dependencies whose path begins with Engram's are not expected, so analyzing one is an
accepted cost of a simple scope.

#### 794a2a1 UAT Scenarios

```gherkin
Scenario Outline: 794a2a1a Existing check entry points report only on Engram's module packages
  Given an Engram checkout with NilAway installed
  And an Engram package contains a known nil dereference
  And a third-party dependency contains a known nil dereference
  When the developer invokes <entry>
  Then the check reports the nil dereference in the Engram package
  And it reports no finding in the third-party dependency
  Examples:
    | entry               |
    | check-nils-for-fail |
    | check-nils          |
    | check               |
    | check-for-fail      |
    | check-full          |

Scenario: 794a2a1b Modules sharing only the path prefix are not reported on
  Given an Engram checkout with NilAway installed
  And an Engram package contains a known nil dereference
  And Engram depends on a module "github.com/toejough/engram-eval" that contains a known nil dereference
  When the developer invokes the nil check command
  Then the check reports the nil dereference in the Engram package
  And it reports no finding in "github.com/toejough/engram-eval"

Scenario: 794a2a1c The fix command acts only on Engram's module packages
  Given an Engram checkout with NilAway installed
  And an Engram package contains a known nil dereference
  And a third-party dependency contains a known nil dereference
  When the developer invokes check-nils-fix
  Then the third-party dependency is neither modified nor reported on

Scenario: 794a2a1d A nested module is not reported on
  Given an Engram checkout with NilAway installed
  And an Engram package contains a known nil dereference
  And a nested module with its own go.mod contains a known nil dereference
  When the developer invokes the nil check command
  Then the check reports the nil dereference in the Engram package
  And it reports no finding in the nested module
```

### 794a2c Resource settings pass-through

Requirement: 794a2c Engram MUST pass the developer's `GOMEMLIMIT`, `GOGC` and `GOMAXPROCS` to
NilAway unchanged. It MUST NOT add, change or remove any of them, whether a variable is absent, set
to a value, or set to an empty value.

Rationale: suitable resource limits depend on the developer's machine, so Engram leaves them to the
developer. A developer on a memory-constrained machine can set them in their own environment.

#### 794a2c1 UAT Scenarios

```gherkin
Scenario Outline: 794a2c1a Supplied settings are passed through
  Given the developer's environment sets <settings> and none of the other resource variables
  When the developer invokes <entry>
  Then the NilAway process runs with <settings>
  And its environment contains none of <absent>
  Examples:
    | entry               | settings                               | absent           |
    | check-nils-for-fail | GOMEMLIMIT=4GiB                        | GOGC, GOMAXPROCS |
    | check-nils-fix      | GOMEMLIMIT=off, GOGC=off, GOMAXPROCS=8 | none             |
    | check-nils-for-fail | all three explicitly empty             | none             |

Scenario Outline: 794a2c1b Absent settings stay absent
  Given none of GOMEMLIMIT, GOGC, or GOMAXPROCS is set
  When the developer invokes <entry>
  Then the NilAway process's environment contains none of them
  Examples:
    | entry               |
    | check-nils-for-fail |
    | check-nils-fix      |
```

### 861a3a A package's tests run alone

Requirement: 861a3a A developer MUST be able to run the tests of one package, with an optional filter on test names, through a development command, without running the tests of the rest of the module.

Rationale: A developer needs the result for the package they touched without waiting for the whole module.

#### 861a3a1 UAT Scenarios

```gherkin
Scenario: 861a3a1a A developer runs the tests of one package
  Given a module with several packages
  When the developer runs the scoped test command on one of them
  Then only the tests of that package run and the command reports their result
```

### 862a3a Every integration-tagged test runs in the full check

Requirement: 862a3a The full development check MUST run every test tagged `integration`, wherever the test is in the module.

Rationale: A test that no check runs cannot fail when the behavior breaks.

#### 862a3a1 UAT Scenarios

```gherkin
Scenario: 862a3a1a Every integration-tagged test runs in the full check
  Given a test tagged integration outside the dev tree
  When the developer runs the full check
  Then that test runs
```

### 862a3b A check without a login skips

Requirement: 862a3b A development check that needs a login MUST be skipped, not failed, when the login does not exist, and it MUST say which login it needs.

Rationale: A failure reads as a product failure; a skip says what is missing.

#### 862a3b1 UAT Scenarios

```gherkin
Scenario: 862a3b1a A check without a login skips
  Given no login exists
  When the developer runs a check that needs a login
  Then the check is skipped and says which login it needs
```

### 862a3c The real-model parity check has a command

Requirement: 862a3c The developer MUST have a development command that runs the real-model parity tests, and the full check MUST NOT run them.

Rationale: The parity tests load the bundled model and would slow every full check; without a command they run in no check.

#### 862a3c1 UAT Scenarios

```gherkin
Scenario: 862a3c1a A developer runs the parity check
  Given the bundled model
  When the developer runs the parity command
  Then the parity tests run and the full check does not run them
```
