# Delegate-object-level-work Guidance Specification

## Purpose

A deployed guidance document (@import-able into CLAUDE.md) that fires the "orchestrator" reflex at delegation moments — plan the unit, hand it to a subagent via the route skill, review what returns, report the outcome — instead of doing object-level work solo out of habit. Sibling to the recall-firing guidance; it points to the route skill and please skill rather than restating them. Why: delegation-guidance pattern (no dedicated ADR). Validation: dev/eval/LEDGER.md#delegate-guidance-flip (headless RED→GREEN: solo→subagent-dispatch, including the trivial-rename case).

## Requirements

### Requirement: Agent SHALL plan every unit of work before delegating
For every unit of work — from a multi-file feature to a one-liner to a "quick look" — the agent SHALL draft a plan and route it to a subagent before taking any direct tool action on the work itself.

#### Scenario: Multi-step change
- **WHEN** an agent encounters a multi-step change task
- **THEN** the agent decomposes it into units and routes each to a subagent rather than executing inline

#### Scenario: Single-file rename or edit
- **WHEN** an agent is about to rename a file or edit a single file
- **THEN** the agent still routes the unit to a subagent; small size does not bypass delegation

#### Scenario: Quick look or investigation
- **WHEN** an agent is about to investigate or "just look at" files for orientation
- **THEN** the agent recognizes this as a unit of work and routes it to a subagent rather than reading directly

### Requirement: Agent SHALL use route to set tier and route evidence
When delegating object-level work, the agent SHALL use the route skill to pick the subagent's model tier based on recalled evidence, not prediction.

#### Scenario: Dispatch based on evidence
- **WHEN** a subagent is about to be dispatched for a unit of work
- **THEN** the agent invokes route to select the tier, starting at cheapest unless recalled evidence raises it

### Requirement: Agent SHALL review subagent results before reporting
After a subagent returns, the agent SHALL review the artifact with fresh context before accepting and reporting the outcome.

#### Scenario: Subagent completion
- **WHEN** a subagent completes a delegated unit and returns its work
- **THEN** the agent reviews the result using fresh-context review (never the subagent's own "done" claim)

### Requirement: Agent SHALL report outcomes with evidence trail
The agent SHALL report the outcome of delegated work, including the route evidence (work-kind, tier, model, review verdict, cost if available).

#### Scenario: Reporting delegated work
- **WHEN** an agent reports the result of delegated work to the user
- **THEN** the agent includes the route evidence table (work-kind, tier, model, outcome, escalation, cost)

### Requirement: Agent SHALL refuse solo work on unvalidated "quick" tasks
The agent SHALL not perform object-level work solo based on the reasoning "it's trivial", "it's just a rename", "exact files known", or "the overhead would exceed the work" without recalled evidence that this work-kind runs reliably below routing overhead.

#### Scenario: Resisting the solo-work reflex
- **WHEN** an agent feels the impulse to do quick work inline (a rename, a one-liner, a "trivial" fix)
- **THEN** the agent checks recalled memory for evidence that this work-kind is proven to run below routing overhead; absent evidence, the agent routes it
