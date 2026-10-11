# Development Check Orchestration - Technical Spec

Preserves the existing nil-check entry points under their shared target identity. Routing a check
run through the invocation policy and the analyzer subprocess is the
[Nil Check Run Flow](./nil-check-run-flow.md), whose interaction requirement is
[794b4b Nil check run](./nil-check-run-flow.md#794b4b-nil-check-run).
Starting the analyzer process is [NilAway Subprocess](./nilaway-subprocess.md).

## 794b4a Preserve entry points

Requirement: 794b4a Development-check orchestration MUST replace the action of the existing shared
check/fix targets, preserving their names, their registration metadata, and all aggregate
references to them.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios) and
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios).

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

Scenario: 794b4a1b Registration metadata is preserved
  Given Targ has registered its shared check and fix targets with their registration metadata
  When Engram initializes its development targets
  Then each target keeps the registration metadata Targ gave it, including its source package
  And deregistering Targ's shared targets still removes them
```
