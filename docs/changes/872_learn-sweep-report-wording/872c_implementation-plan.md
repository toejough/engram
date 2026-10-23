# Learn Sweep Report Wording - Implementation Plan

Scope. The row gives the scenario of [872a Learn Sweep Report Wording - Product Spec](./872a_product-spec.md#learn-sweep-report-wording---product-spec) its test evidence and applies its Modified operation in place. The row changes no shipped instruction and no code.

## 872c1 Scenario rows

| ID | Scenario | Kind | Why | Status |
| --- | --- | --- | --- | --- |
| 872c1a | [792a15a1a A session with no lesson writes nothing](../../specs/product/learn.md#792a15a1-uat-scenarios) | Example | Text-level: Test792a15a1a_ASessionWithNoLessonWritesNothing (dev/skillrules_learn_part1_integration_internal_test.go) asserts the [learn skill](../../GLOSSARY.md#learn-skill)'s "No moments of any kind → write nothing." and its two-command learn sentence, and scans for a write-without-moment instruction. It asserts no wording of the report, which [792a15b1a The sweep runs and is reported](../../specs/product/learn.md#792a15b1-uat-scenarios) owns with Test792a15b1a_TheSweepRunsAndIsReported. Baseline pass; the operation changes no sentence either test asserts. Operation [872a3a 792a15a1a A session with no lesson writes nothing](./872a_product-spec.md#872a3a-modified-792a15a1a-a-session-with-no-lesson-writes-nothing), decision [872d1 The sweep report is worded in one scenario only](./872d_questions-and-decisions.md#questions-and-decisions) | Green-single |
