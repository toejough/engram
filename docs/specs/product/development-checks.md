# Development Checks - Product Spec

## 794a2a Project analysis

Requirement: 794a2a The nil check and fix commands MUST report findings only in Engram's module
packages: packages whose import path is Engram's module path or that path followed by `/`. They
MUST NOT analyze third-party or standard-library packages, and MUST NOT report findings in any other
package, such as a package of `github.com/toejough/engram-eval` or of a nested module that has its
own `go.mod`. A dependency whose module path begins with Engram's module path MAY be analyzed. The
commands MUST retain their existing command names and aggregate-check entry points.

Rationale: Dependency package bodies are most of the analysis cost and nothing in them is Engram's
to fix. Dependencies whose path begins with Engram's are not expected, so analyzing one is an
accepted cost of a simple scope.

### 794a2a1 UAT Scenarios

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
  Then the nil dereference in the Engram package is reported
  And the third-party dependency is neither modified nor reported on

Scenario: 794a2a1d A nested module is not reported on
  Given an Engram checkout with NilAway installed
  And an Engram package contains a known nil dereference
  And a nested module with its own go.mod contains a known nil dereference
  When the developer invokes the nil check command
  Then the check reports the nil dereference in the Engram package
  And it reports no finding in the nested module
```

## 794a2c Resource settings pass-through

Requirement: 794a2c Engram MUST pass the developer's `GOMEMLIMIT`, `GOGC` and `GOMAXPROCS` to
NilAway unchanged. It MUST NOT add, change or remove any of them, whether a variable is absent, set
to a value, or set to an empty value.

Rationale: suitable resource limits depend on the developer's machine, so Engram leaves them to the
developer. A developer on a memory-constrained machine can set them in their own environment.

### 794a2c1 UAT Scenarios

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
