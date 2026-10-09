# Product Specification Authoring Guide

This governance guide supplements [`docs/AGENTS.md`](./AGENTS.md). Follow the shared documentation and Luhmann rules there.

## Purpose

A product specification defines externally observable application behavior. It MUST NOT prescribe architecture, code structure, libraries, algorithms, storage, APIs, rollout steps, or test implementation unless the user explicitly expands the scope.

The title MUST end with ` - Product Spec`.

Unless the user explicitly requests additional sections, a product specification MUST contain only:

1. A problem statement.
2. Product requirements.
3. User acceptance testing scenarios.

## Authoring Workflow

Before drafting:

1. Interview the user until the problem, intended outcome, happy path, boundaries, and relevant edge cases are understood.
2. Ask one focused question at a time.
3. Confirm ambiguous behavior rather than choosing a technically preferred interpretation.
4. Do not exceed the behavior the user specified.
5. Present the proposed scope and structure for approval.

Continue interviewing when a requirement permits multiple interpretations, when failure behavior is unclear, or when a happy path implies uncovered boundaries.

## Problem Statement

The problem statement MUST identify:

- Who experiences the problem.
- What is difficult, missing, or risky today.
- What outcome is needed.

Keep it focused on the user need. Do not include a proposed implementation.

## Product Requirements

- Use RFC 2119 keywords such as **MUST**, **MUST NOT**, **SHOULD**, **SHOULD NOT**, and **MAY** deliberately.
- State requirements as observable product behavior.
- Separate materially different behaviors into distinct requirements.
- Include boundary and failure behavior only when confirmed with the user.
- Avoid vague qualifiers that cannot be verified by an end user.

## Product Lifecycle Rules

Follow [`docs/SPECIFICATION-LIFECYCLE-GUIDE.md`](./SPECIFICATION-LIFECYCLE-GUIDE.md) for shared
delta, stable-identity, canonical-specification, and promotion rules.

- When absence is an enduring product contract, preserve it as an explicit **MUST NOT** requirement
  rather than relying only on deletion.
- Each canonical product specification file MUST represent exactly one user-facing capability.
- Its filename MUST be the semantic kebab-case capability name without an issue or Luhmann prefix.
- Its title MUST be the semantic capability name without an issue or Luhmann prefix and MUST end
  with ` - Product Spec`.
- A capability file MAY contain requirements and scenarios introduced by multiple changes.
- Do not promote the problem statement or other change context.

## User Acceptance Testing

- Every **MUST** and **MUST NOT** product requirement MUST have at least one attached UAT scenario.
- Every distinct mandatory behavior branch within a requirement MUST have end-user-verifiable UAT coverage; one scenario for the overall requirement is insufficient when it contains multiple mandatory branches.
- **MAY** behavior documents permitted product freedom and MUST NOT require UAT scenarios.
- Do not add UAT solely to preserve one currently observed outcome of optional behavior.
- Write scenarios in Gherkin-style `Given`/`When`/`Then` form.
- Every scenario MUST be verifiable by an end user through observable application behavior.
- Number every scenario within the document's Luhmann hierarchy.
- Cover the confirmed happy path and relevant edge cases, including boundaries and failures.
- Use `Scenario Outline` and `Examples` only when multiple inputs follow the same behavior.
- Split cases into separate scenarios when their interactions or observable outcomes differ.
- Do not use implementation state, internal functions, source structure, developer-only logs, or other non-user-observable details as acceptance criteria.

## Completion Checklist

Verify:

- The document contains only the requested problem statement, product requirements, and UAT unless the user approved more.
- Every requirement uses deliberate RFC 2119 language.
- Every distinct **MUST** and **MUST NOT** branch has end-user-verifiable coverage.
- No scenario exists solely to validate **MAY** behavior.
- Happy paths, relevant boundaries, and confirmed failures are covered.
- No implementation details have leaked into behavior-only requirements.
- Every scenario can be verified by an end user.
- Promoted product behavior is placed in the owning capability file without changing stable IDs.
- The specification lifecycle checklist passes.
- The shared checklist in [`docs/AGENTS.md`](./AGENTS.md) passes.
