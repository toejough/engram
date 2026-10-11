# Development Check Orchestration - Technical Spec

Preserves the existing nil-check entry points under their shared target identity. Routing a check
run through the invocation policy and the analyzer subprocess is the
[Nil Check Run Flow](./nil-check-run-flow.md): it holds
[794b4b Execute configured request](./nil-check-run-flow.md#794b4b-execute-configured-request),
[794b4d Successful result](./nil-check-run-flow.md#794b4d-successful-result) and
[794b4e Failed result](./nil-check-run-flow.md#794b4e-failed-result).
Starting the analyzer process is [NilAway Subprocess](./nilaway-subprocess.md).

## 794b4a Preserve entry points

Requirement: 794b4a Development-check orchestration MUST replace the behavior of the existing
shared check/fix target objects in place, preserving the target names and all aggregate references.

Supports [794a2a1a Existing entry points use project analysis](../product/development-checks.md#794a2a1-uat-scenarios).

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
```
