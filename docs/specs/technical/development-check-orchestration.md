# Development Check Orchestration - Technical Spec

Preserves the existing nil-check entry points under their shared target identity. Routing a check
run through the invocation policy and the analyzer subprocess is the
[Nil Check Run Flow](./nil-check-run-flow.md), whose interaction requirement is
[794b4b Nil check run](./nil-check-run-flow.md#794b4b-nil-check-run).
Starting the analyzer process is [NilAway Subprocess](./nilaway-subprocess.md).

## 794b4a Preserve entry points

Requirement: 794b4a Development-check orchestration MUST deregister Targ's shared development targets
and register Engram's own nil check and fix targets under the existing names `check-nils-fix` and
`check-nils-for-fail`. It MUST register its own `check-nils`, `check`, `check-for-fail`,
`check-full`, `mutate` and `watch` under the existing names. Each MUST have the same dependencies,
order and dependency mode as Targ's target of that name, except that Engram's nil and aggregate
targets replace Targ's, and Engram's own dev-package checks (`test-dev`, `test-dev-integration`,
`lint-dev`) MAY be appended to the final dependency group. `watch` MUST run Engram's `check`. It MUST
re-register Targ's other shared targets unchanged.

Rationale: Targ's aggregates hold direct references to its own nil targets, and its `watch` calls
its own `check`, so replacing the nil targets alone would leave them running Targ's versions;
deregistering and re-registering is Targ's supported way to customize shared targets. The appended
dev-package checks cover Engram's build-tagged tooling code, which Targ's checks never compile.

Supports
[794a2a1a Existing check entry points report only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios) and
[794a2a1c The fix command acts only on Engram's module packages](../product/development-checks.md#794a2a1-uat-scenarios).

### 794b4a1 Contract Scenarios

```gherkin
Scenario: 794b4a1a Direct and aggregate target routing
  Given Targ has registered its shared development targets
  When Engram initializes its development targets
  Then check-nils-fix and check-nils-for-fail run Engram's configured invocation
  And check-nils, check, check-for-fail, check-full, mutate and watch each reach Engram's nil targets
  And none of them reaches Targ's nil targets

Scenario: 794b4a1c Engram's aggregates match Targ's apart from the nil targets
  Given Targ's check-nils, check, check-for-fail, check-full and mutate targets
  When Engram initializes its development targets
  Then each Engram aggregate of the same name has the same dependencies in the same order and mode
  And the only differences are Engram's nil and aggregate targets in place of Targ's
  And Engram's dev-package checks appended to the final dependency group

Scenario: 794b4a1d Targ's other shared targets stay available
  Given Targ has registered its shared development targets
  When Engram initializes its development targets
  Then every Targ shared target other than those Engram replaces is registered unchanged

Scenario: 794b4a1e watch runs Engram's check
  Given Engram has initialized its development targets
  When watch detects a change
  Then it runs Engram's check, not Targ's

Scenario: 794b4a1f No re-registered Targ target runs Targ's nil targets
  Given Engram has initialized its development targets
  When any re-registered Targ shared target runs
  Then it does not reach Targ's nil check or fix targets
```
