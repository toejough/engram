# Development-Tooling Withdrawal - Product Spec

## 868a1 Problem statement

The behavior record holds a product capability for the tools the maintainer uses to measure engram. Its own text says those tools are dev tooling in this repository and not part of the shipped command line, so the record promises engram users something engram does not ship, and its 43 scenarios count as gaps that no gate can close. The owner answered [861d21 Scenarios covered only by pytest](../861_scenario-coverage/861d_questions-and-decisions.md#questions-and-decisions) (the same question as [858d100 Scenarios covered only by pytest](../858_documentation-framework-adoption/858d_questions-and-decisions.md#questions-and-decisions)) with option (e): the development-tooling components are not product behavior and leave the behavior record. The issue is [#868](https://github.com/toejough/engram/issues/868), part of #858 and #861.

## 868a2 Removed: Memory Effectiveness Evaluation

Canonical target: `docs/specs/product/memory-effectiveness-evaluation.md`. Target: the whole capability 792a23 Memory Effectiveness Evaluation, with its requirements 792a23a to 792a23q and their 43 scenarios (792a23a1a to 792a23q1a). The IDs MUST NOT be reused.

Rationale: the maintainer's measurement tools are repository tooling, not behavior engram promises a user ([868d1 Owner decision: option (e)](./868d_questions-and-decisions.md#questions-and-decisions)).
