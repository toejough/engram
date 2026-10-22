# Scenario Coverage - Product Spec

## 861a1 Problem statement

The owner and every agent that relies on the specifications cannot tell which documented behavior is verified. Of the 3,562 scenarios in `docs/specs`, 1,655 have no test traced to them, and 19 test functions name an ID that no specification holds. A regression in an untraced scenario goes unseen, and a verified promise cannot be told from an unverified one. The owner needs every documented scenario traced to a test that fails when the promise breaks, or to a recorded reason why none exists, without changing what the software promises.

## 861a2 References

This change adds no outcome. The scenarios in scope belong to the capabilities and components of `docs/specs`; the change serves no outcome of its own, and its one new promise is [861a3 Scoped test runs](#861a3-added-scoped-test-runs). They are listed by owner file in [861c2 Scenario tests by owner file and package](./861c_implementation-plan.md#861c2-scenario-tests-by-owner-file-and-package), with the blocked kinds in [861c3 Blocked nodes](./861c_implementation-plan.md#861c3-blocked-nodes). The gap record this change works from is [858c2a Scenarios without a test, by owner file](../858_documentation-framework-adoption/858c_implementation-plan.md#858c2a-scenarios-without-a-test-by-owner-file) and [858c2b Test functions whose ID is in no spec](../858_documentation-framework-adoption/858c_implementation-plan.md#858c2b-test-functions-whose-id-is-in-no-spec); the issue is [#861](https://github.com/toejough/engram/issues/861).

## 861a3 Added: Scoped test runs

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](./861d_questions-and-decisions.md#questions-and-decisions)); the operation below is applied at promotion ([861d15 861 merges without promotion](./861d_questions-and-decisions.md#questions-and-decisions)).

### 861a3a Added: A package's tests run alone

Target: [794a2 Development Checks](../../specs/product/development-checks.md#794a2-development-checks), after [794a2c Resource settings pass-through](../../specs/product/development-checks.md#794a2c-resource-settings-pass-through).

Requirement: 861a3a A developer MUST be able to run the tests of one package, with an optional filter on test names, through a development command, without running the tests of the rest of the module.

Rationale: A developer needs the result for the package they touched without waiting for the whole module.

#### 861a3a1 UAT Scenarios

```gherkin
Scenario: 861a3a1a A developer runs the tests of one package
  Given a module with several packages
  When the developer runs the scoped test command on one of them
  Then only the tests of that package run and the command reports their result
```
