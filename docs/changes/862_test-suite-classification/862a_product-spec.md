# Test Suite Classification - Product Spec

## 862a1 Problem statement

The owner and every agent that relies on the project's checks cannot rely on the suite classification that the Project Profile states. A test tagged `integration` outside `dev/` runs in no check; the `parity` tests have no approved command; an acceptance run without a login fails instead of skipping, which reads as a product failure; and the Profile states known deviations in its own text instead of citing a decision for each. The owner needs every test to run in the check its classification names, a run without a login to skip clearly, and each remaining deviation to be a recorded decision.

## 862a2 References

The issue is [#862](https://github.com/toejough/engram/issues/862) (part of #858); it was recorded at the adoption as [858d99 Test-suite classification deviations](../858_documentation-framework-adoption/858d_questions-and-decisions.md#questions-and-decisions). The rules are the [Test-Suite Classification](../../CONVENTIONS-GUIDE.md#test-suite-classification) section of the Conventions Guide and the test-suite table of the [Project Profile](../../../AGENTS.md#project-profile). The eighteen default-suite tests that use a real temporary directory stay tracked by [#860](https://github.com/toejough/engram/issues/860).

## 862a3 Added: Integration run, login skip and parity check

Operations a test finds necessary in existing specs are applied in place ([861d6 A new test that fails at baseline](../861_scenario-coverage/861d_questions-and-decisions.md#questions-and-decisions)); the operations below are applied at promotion ([862d8 862 merges without promotion](./862d_questions-and-decisions.md#questions-and-decisions)).

### 862a3a Added: Every integration-tagged test runs in the full check

Target: [794a2 Development Checks](../../specs/product/development-checks.md#794a2-development-checks), after [794a2c Resource settings pass-through](../../specs/product/development-checks.md#794a2c-resource-settings-pass-through).

Requirement: 862a3a The full development check MUST run every test tagged `integration`, wherever the test is in the module.

Rationale: A test that no check runs cannot fail when the behavior breaks.

#### 862a3a1 UAT Scenarios

```gherkin
Scenario: 862a3a1a Every integration-tagged test runs in the full check
  Given a test tagged integration outside the dev tree
  When the developer runs the full check
  Then that test runs
```

### 862a3b Added: A check without a login skips

Target: [794a2 Development Checks](../../specs/product/development-checks.md#794a2-development-checks), after [862a3a Every integration-tagged test runs in the full check](#862a3a-added-every-integration-tagged-test-runs-in-the-full-check).

Requirement: 862a3b A development check that needs a login MUST be skipped, not failed, when the login does not exist, and it MUST say which login it needs.

Rationale: A failure reads as a product failure; a skip says what is missing.

#### 862a3b1 UAT Scenarios

```gherkin
Scenario: 862a3b1a A check without a login skips
  Given no login exists
  When the developer runs a check that needs a login
  Then the check is skipped and says which login it needs
```

### 862a3c Added: The real-model parity check has a command

Target: [794a2 Development Checks](../../specs/product/development-checks.md#794a2-development-checks), after [862a3b A check without a login skips](#862a3b-added-a-check-without-a-login-skips).

Requirement: 862a3c The developer MUST have a development command that runs the real-model parity tests, and the full check MUST NOT run them.

Rationale: The parity tests load the bundled model and would slow every full check; without a command they run in no check.

#### 862a3c1 UAT Scenarios

```gherkin
Scenario: 862a3c1a A developer runs the parity check
  Given the bundled model
  When the developer runs the parity command
  Then the parity tests run and the full check does not run them
```
