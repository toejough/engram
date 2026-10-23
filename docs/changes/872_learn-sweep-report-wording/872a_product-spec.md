# Learn Sweep Report Wording - Product Spec

## 872a1 Problem statement

The owner reading the product spec of the [learn skill](../../GLOSSARY.md#learn-skill) finds [792a15a1a A session with no lesson writes nothing](../../specs/product/learn.md#792a15a1-uat-scenarios) promising that the agent "reports its tally" after the [sweep](../../GLOSSARY.md#sweep) ([#872](https://github.com/toejough/engram/issues/872)). The skill, reworded by [#871](https://github.com/toejough/engram/issues/871) (row 861d382), tells the agent to report the per-source lines the sweep prints, or "memory index up to date" when it prints nothing, and [792a15b1a The sweep runs and is reported](../../specs/product/learn.md#792a15b1-uat-scenarios) owns that sentence and its test. The owner needs the first scenario to promise the report without a second wording of it, so that the spec and the skill agree and the report has one home.

## 872a2 References

This change adds no outcome and no requirement and changes no behavior, so it has no technical spec. It modifies one scenario, listed in [872c1 Scenario rows](./872c_implementation-plan.md#872c1-scenario-rows). The operation is applied in place in `docs/specs/product`, and this directory is deleted when the change is promoted.

## 872a3 Modified: Learn scenarios

### 872a3a Modified: 792a15a1a A session with no lesson writes nothing

Target: [792a15a1a A session with no lesson writes nothing](../../specs/product/learn.md#792a15a1-uat-scenarios). Differs from the current text in the Then line: it reports the [sweep](../../GLOSSARY.md#sweep), not "its tally". The wording of that report stays with [792a15b1a The sweep runs and is reported](../../specs/product/learn.md#792a15b1-uat-scenarios), which has its own test.

```gherkin
Scenario: 792a15a1a A session with no lesson writes nothing
  Given a session with no correction, save-request, reversal or confirmed approach
  When the agent runs learn
  Then it runs the sweep, reports it and writes no note
```
