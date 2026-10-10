# NilAway Invocation Policy - Technical Spec

Constructs first-party analyzer arguments that limit analysis to Engram's module packages.

## 794b3a Mode and scope

Requirement: 794b3a The invocation policy MUST produce `-include-pkgs=github.com/toejough/engram`
and `./...` for both modes, adding `-fix` only in fix mode.

Rationale: NilAway matches each `-include-pkgs` entry as a string prefix of the package path, so the
module path covers the root package and every subpackage. `./...` selects only main-module packages
as roots, and NilAway reports findings only for roots, so a prefix-sibling dependency or an imported
nested module may be analyzed but is never reported on.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2a1b Modules sharing only the path prefix are not reported on](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios), and
[794a2a1d A nested module is not reported on](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b3a1 Contract Scenarios

```gherkin
Scenario Outline: 794b3a1a Scope and mode arguments
  Given the invocation mode is <mode>
  When the invocation policy constructs analyzer arguments
  Then the arguments are <arguments>
  Examples:
    | mode  | arguments                                           |
    | check | -include-pkgs=github.com/toejough/engram ./...      |
    | fix   | -include-pkgs=github.com/toejough/engram -fix ./... |

Scenario: 794b3a1b The scope covers the root package and its subpackages
  Given the invocation mode is check
  When the invocation policy constructs analyzer arguments
  Then the scope value is a string prefix of "github.com/toejough/engram"
  And the scope value is a string prefix of "github.com/toejough/engram/internal/cli"
```
