# Automatic Memory - Technical Spec

Filled depth first ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)): step 3 and step 4 run for one node at a time, in the order of
[793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan). Until the last node has been through step 4, this spec is incomplete and not approved. The
"Pending" lines below are the departure recorded in [793b10 Departure: depth-first fill](./793b_decision-log.md#793b10-departure-depth-first-fill). Its
sections follow the [technical spec guide](../../TECHNICAL-SPEC-GUIDE.md#default-structure). The solution-space
analysis, relative option impact and alternative options are omitted by decision ([793b12 Departure: no option analysis](./793b_decision-log.md#793b12-departure-no-option-analysis)): step 3 takes
the minimal components a scenario needs.

New technical requirements take the next free letter under [793d2 Recommended sequence diffs](#793d2-recommended-sequence-diffs) for an interaction requirement (one
per flow) or under [793d4 Recommended component-behavior diffs](#793d4-recommended-component-behavior-diffs) for a behavior requirement (one per component behavior).

## 793d1 Orientation

Pending, written with the first requirement's step 3: the product problem ([793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement)), the relevant current
behavior, why it is not enough, and the recommended solution.

## 793d2 Recommended sequence diffs

Pending. Each normative flow gets one Mermaid sequence diagram, one interaction requirement stating its
routing, ordering, continuation, gating and every branch, and one scenario per branch and terminal path.

## 793d3 Recommended component diffs

Pending. The components added, removed, updated and retained, with their dependency boundaries. A component is
named only if it meets the guide's qualification criteria.

## 793d4 Recommended component-behavior diffs

Pending. One behavior requirement per component behavior, each owned by one component, with at least one
scenario and the product scenario IDs it supports.

## 793d5 Product-scenario mapping

Pending. Each product scenario restated in full at its first mapping location and mapped to the technical
requirements that realize it, then cited by its ID.

## 793d6 Recommended impact

Pending. The repository, dependency, toolchain, operational, risk and complexity consequences.
