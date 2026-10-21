# NilAway Invocation Policy - Technical Spec

Builds NilAway's arguments: the analysis scope for Engram's module and the check or fix mode.

## 794b3 NilAway Invocation Policy

- Ownership: repository-owned component (the `dev` package); it owns [794b3a Mode and scope](#794b3a-mode-and-scope).
- Responsibility: builds NilAway's arguments for the check or fix mode: the analysis scope for Engram's module, and
  `-fix` in fix mode only. Its caller is [Development Check Orchestration](./development-check-orchestration.md), in the
  [Nil Check Run Technical Workflow](./nil-check-run-workflow.md).
- Current realization (informational): `nilawayArgs` in `dev/nilaway.go`.
- Qualification: Purpose: the arguments that confine NilAway's findings to Engram's module packages, as the nil check
  and fix commands promise ([794a2a Project analysis](../product/development-checks.md#794a2a-project-analysis)).
  Contract: the arguments for each mode, tested directly (`Test794b3a1a_ModeArguments` in
  `dev/nilaway_internal_test.go`) and, through the entry points, against the real analyzer
  (`dev/nilaway_integration_internal_test.go`). Value: the scope has its own reasoning, about how NilAway matches
  `-include-pkgs` entries and reports findings only for roots (the Rationale of
  [794b3a Mode and scope](#794b3a-mode-and-scope)).
- Required behavior: NilAway (external) is assumed to match each `-include-pkgs` entry as a string prefix of the
  package path and to report findings only for the packages `./...` selects as roots (the Rationale of
  [794b3a Mode and scope](#794b3a-mode-and-scope)); authoritative contract: not recorded in this repository;
  compatibility verification: `dev/nilaway_integration_internal_test.go` (tags `targ`, `integration`) runs the real
  `nilaway` on fixture modules holding a third-party dependency, a prefix-sibling module and a nested module.
- Supported capabilities: [Development Checks](../product/development-checks.md) (the nil check and fix commands
  report findings only in Engram's module packages).

### 794b3a Mode and scope

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

#### 794b3a1 Contract Scenarios

```gherkin
Scenario Outline: 794b3a1a Scope and mode arguments
  Given the invocation mode is <mode>
  When the invocation policy constructs analyzer arguments
  Then the arguments are <arguments>
  Examples:
    | mode  | arguments                                           |
    | check | -include-pkgs=github.com/toejough/engram ./...      |
    | fix   | -include-pkgs=github.com/toejough/engram -fix ./... |
```
