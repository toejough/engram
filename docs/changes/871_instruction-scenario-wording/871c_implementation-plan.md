# Instruction Scenario Wording - Implementation Plan

Scope. Each row gives one scenario of [871a Instruction Scenario Wording - Product Spec](./871a_product-spec.md#instruction-scenario-wording---product-spec) a test and applies its Modified operation in place. No row changes shipped instructions or code.

## 871c1 Scenario rows

| ID | Scenario | Kind | Why | Status |
| --- | --- | --- | --- | --- |
| 871c1a | [792a17k1c The tag format is validated](../../specs/product/route.md#792a17k1-uat-scenarios) | Example | Test792a17k1c_TheTagFormatIsValidated (internal/cli/route_evidence_tag_scenarios_test.go) drives engram learn. Baseline pass; with the tag pattern widened, three malformed-tag subtests fail. Operation [871a3a](./871a_product-spec.md#871a3a-modified-792a17k1c-the-tag-format-is-validated), decision [871d1](./871d_questions-and-decisions.md#questions-and-decisions) | Green-single |
| 871c1b | [792a13b1b A session without the skill follows the runbook note](../../specs/product/write-memory.md#792a13b1-uat-scenarios) | Example | Text-level: Test792a13b1b_ASessionWithoutTheSkillFollowsTheRunbookNote (dev/skillrules_write_memory_part1_integration_internal_test.go) asserts the opening clause of shim rule 6 and the skill's contract heading, not the frame's sentences. Baseline pass; renaming either fails it. Operation [871a4a](./871a_product-spec.md#871a4a-modified-792a13b1b-a-session-without-the-skill-follows-the-runbook-note), decision [871d2](./871d_questions-and-decisions.md#questions-and-decisions) | Green-single |
