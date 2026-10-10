# NilAway Invocation Policy - Technical Spec

Constructs first-party analyzer arguments that limit analysis to Engram's module packages.

## 794b3a Mode and scope

Requirement: 794b3a The invocation policy MUST produce `-include-pkgs=github.com/toejough/engram/`
and `./...` for both modes, adding `-fix` only in fix mode. The scope value MUST be Engram's module
path followed by `/` and MUST NOT be the bare module path.

Rationale: NilAway treats each `-include-pkgs` entry as a package-path prefix, so the bare module
path would also match `github.com/toejough/engram-eval`. The trailing `/` matches only Engram's
subpackages, and `./...` loads only the root module, which leaves nested modules out.

Supports
[794a2a1a Existing check entry points analyze only Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2a1b Modules sharing only the path prefix are not analyzed](../product/development-checks.md#794a2a1-uat-scenarios),
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios), and
[794a2a1d A nested module is not analyzed](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b3a1 Contract Scenarios

```gherkin
Scenario Outline: 794b3a1a Scope and mode arguments
  Given the invocation mode is <mode>
  When the invocation policy constructs analyzer arguments
  Then the arguments are <arguments>
  Examples:
    | mode  | arguments                                           |
    | check | -include-pkgs=github.com/toejough/engram/ ./...      |
    | fix   | -include-pkgs=github.com/toejough/engram/ -fix ./... |

Scenario: 794b3a1b The scope excludes prefix siblings
  Given the invocation mode is check
  When the invocation policy constructs analyzer arguments
  Then the scope value is a package-path prefix of "github.com/toejough/engram/internal/cli"
  And the scope value is not a package-path prefix of "github.com/toejough/engram-eval/pkg"
```
