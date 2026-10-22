# Development-Tooling Withdrawal - Technical Spec

This change removes two components and selects no design: it has no impact statement, no alternatives and no hypotheses; the options it faced are the rows of [Questions and decisions](./868d_questions-and-decisions.md#questions-and-decisions). The product problem is [868a1 Problem statement](./868a_product-spec.md#868a1-problem-statement).

## 868b1 Removed: Eval Trial Harness

Canonical target: `docs/specs/technical/eval-trial-harness.md`. Target: the whole component 792b38 Eval Trial Harness (ownership: repository component, dev tooling), with its requirements 792b38c1 to 792b38c11 and their 73 scenarios (792b38c1a1 to 792b38c11a7). The IDs MUST NOT be reused.

Rationale: the component is the maintainer's eval tooling under `dev/eval`, not shipped behavior ([868d1 Owner decision: option (e)](./868d_questions-and-decisions.md#questions-and-decisions)).

## 868b2 Removed: Memory Loop Auditor

Canonical target: `docs/specs/technical/memory-loop-auditor.md`. Target: the whole component 792b39 Memory Loop Auditor (ownership: repository component, dev tooling), with its requirements 792b39c1 to 792b39c8 and their 54 scenarios (792b39c1a1 to 792b39c8a5). The IDs MUST NOT be reused.

Rationale: the component is the maintainer's audit tooling under `dev/eval/audit`, not shipped behavior ([868d1 Owner decision: option (e)](./868d_questions-and-decisions.md#questions-and-decisions)).
