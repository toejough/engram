# Instruction Scenario Wording - Product Spec

## 871a1 Problem statement

Two scenarios promise more than the shipped instructions state, and the scenario coverage change left them without a test ([#871](https://github.com/toejough/engram/issues/871), rows [871d1 The tag format of the route evidence note is validated by the binary, not stated by the skill (792a17k1c)](./871d_questions-and-decisions.md#questions-and-decisions) and [871d2 No shipped instruction says a session with only the write-memory runbook note follows it (792a13b1b)](./871d_questions-and-decisions.md#questions-and-decisions)). In both, the promise belongs to another owner: the binary validates the tag, and the follow-frame of the ambient guidance carries a runbook note. The owner needs each scenario to promise only what its owner keeps, and a test where it is kept, without changing what the software does.

## 871a2 References

This change adds no outcome and no requirement. It modifies two scenarios, listed in [871c1 Scenario rows](./871c_implementation-plan.md#871c1-scenario-rows). The operations are applied in place in `docs/specs/product`; the integrator deletes this directory when it promotes the change.

## 871a3 Modified: Route scenarios

### 871a3a Modified: 792a17k1c The tag format is validated

Target: [792a17k1 UAT Scenarios](../../specs/product/route.md#792a17k1-uat-scenarios), Scenario 792a17k1c. Differs from the current text in the Then line: it states the route skill's own tag rule (three families, an open kebab-case set of work-kinds, closed sets of tiers and outcomes). The character set and the refusal of a malformed tag before the write stay with [792a3g Argument validation](../../specs/product/note-writing.md#792a3g-argument-validation), the binary's rule, which has its own scenario.

```gherkin
Scenario: 792a17k1c The tag format is validated
  Given a dispatch record
  When its tags are written
  Then each is work-kind/<k> with a kebab-case value from an open set, tier/<t> or outcome/<o> from closed sets
```

## 871a4 Modified: Write Memory scenarios

### 871a4a Modified: 792a13b1b A session without the skill follows the runbook note

Target: [792a13b1 UAT Scenarios](../../specs/product/write-memory.md#792a13b1-uat-scenarios), Scenario 792a13b1b. Differs from the current text in the Then line: the note is followed through the wikilinked-runbook rule of the ambient guidance ([792a19q Linked and truncated runbooks get the same frame](../../specs/product/ambient-guidance.md#792a19q-linked-and-truncated-runbooks-get-the-same-frame)), and the contract it carries is the skill's body, which the registered note mirrors ([792a12e Accepting a refresh replaces the body, keeps the fields, and marks the note pending](../../specs/product/skill-runbook-registration.md#792a12e-accepting-a-refresh-replaces-the-body-keeps-the-fields-and-marks-the-note-pending)).

```gherkin
Scenario: 792a13b1b A session without the skill follows the runbook note
  Given a session that received only the write-memory runbook note over a vault-graph edge
  When a parent reaches a write site
  Then the note is followed through the wikilinked-runbook rule of the ambient guidance, and the contract is the note's body, the skill's handoff contract
```
