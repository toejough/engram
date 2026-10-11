# Development Check Orchestration - Technical Spec

Preserves the existing nil-check entry points under their shared target identity. Routing a check
run through the invocation policy and the analyzer subprocess is the
[Nil Check Run Flow](./nil-check-run-flow.md), whose interaction requirement is
[794b4b Nil check run](./nil-check-run-flow.md#794b4b-nil-check-run).
Starting the analyzer process is [NilAway Subprocess](./nilaway-subprocess.md).

## 794b4a Preserve entry points

Requirement: 794b4a Development-check orchestration MUST deregister Targ's shared development targets
and register Engram's own nil check and fix targets under the existing names `check-nils-fix` and
`check-nils-for-fail`. It MUST register its own `check-nils`, `check`, `check-for-fail` and
`check-full` aggregates under the existing names, each with the same dependencies and dependency
mode as Targ's aggregate of that name, except that Engram's nil targets replace Targ's. It MUST
re-register Targ's other shared targets unchanged.

Rationale: Targ's aggregates hold direct references to its own nil targets, so replacing the nil
targets alone would leave the aggregates running Targ's versions; deregistering and re-registering
is Targ's supported way to customize shared targets.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios) and
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b4a1 Contract Scenarios

```gherkin
Scenario: 794b4a1a Direct and aggregate target routing
  Given Targ has registered its shared development targets
  When Engram initializes its development targets
  Then check-nils-fix and check-nils-for-fail run Engram's configured invocation
  And check-nils, check, check-for-fail and check-full each reach Engram's nil targets
  And none of them reaches Targ's nil targets

Scenario: 794b4a1c Engram's aggregates match Targ's apart from the nil targets
  Given Targ's check-nils, check, check-for-fail and check-full aggregates
  When Engram initializes its development targets
  Then each Engram aggregate of the same name has the same dependencies in the same order and mode
  And the only differences are Engram's nil targets in place of Targ's

Scenario: 794b4a1d Targ's other shared targets stay available
  Given Targ has registered its shared development targets
  When Engram initializes its development targets
  Then every Targ shared target other than the nil and aggregate targets is registered unchanged
```
