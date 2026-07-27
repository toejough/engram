# Recall-at-decision-moments Guidance Specification

## Purpose

A deployed guidance document (@import-able into CLAUDE.md) that fires recall at specific moments — when weighing a proposed approach in conversation, before declaring work done, after unexplained failures, and before committing to a new approach — instead of relying on recall firing only at task start. Why: docs/architecture/adr.md — ADR-0001. Validation: dev/eval/LEDGER.md#recall-moments-opus48-remeasure (current-model measurement) and dev/eval/LEDGER.md#underload-firing-wording-fix (conversational endorse cue, deployed 2026-07-15).

## Requirements

### Requirement: Agent SHALL fire recall before endorsing or ranking a proposed approach
Before discussing or ranking a proposed technical direction, the agent SHALL invoke `/recall glance` to surface any prior decision that tried and killed the approach.

#### Scenario: Chatting about a proposed approach
- **WHEN** an agent is about to discuss, evaluate, or rank a proposed approach in conversation
- **THEN** the agent invokes `/recall glance` with phrases derived from the approach before offering analysis

### Requirement: Agent SHALL fire recall before declaring work done
Before marking a task complete, the agent SHALL invoke `/recall glance` to surface vault lessons and gotchas it has already learned but failed to recall.

#### Scenario: Completing a work task
- **WHEN** an agent is about to declare a task, unit, or phase done
- **THEN** the agent invokes `/recall glance` to surface gotchas before verifying and closing

### Requirement: Agent SHALL fire recall after an unexplained failure
When encountering a failure it cannot immediately explain, the agent SHALL invoke `/recall glance` (once, not on every retry) to surface the lesson that names the root cause.

#### Scenario: Failure without immediate explanation
- **WHEN** an agent encounters a failure and cannot immediately explain the root cause
- **THEN** the agent invokes `/recall glance` once with phrases derived from the failure before guessing at causes

### Requirement: Agent SHALL fire recall before committing to a new approach
Before beginning a new technical approach, the agent SHALL invoke `/recall glance` to surface prior decisions and standards while the path is still cheap to change.

#### Scenario: Starting to build a new approach
- **WHEN** an agent is about to begin implementing a new technical approach or strategy
- **THEN** the agent invokes `/recall glance` to surface prior decisions and standards before proceeding

### Requirement: Agent SHALL escalate glance to deep for weighty or irreversible decisions
When a decision is weighty or irreversible, or when glance flags a gap it can't resolve, the agent SHALL escalate to `/recall deep` to crystallize lessons into the vault.

#### Scenario: Weighty decision requires deep escalation
- **WHEN** a decision is weighty, irreversible, or glance surfaces a gap it cannot resolve
- **THEN** the agent escalates to `/recall deep` to crystallize the lesson. The recency-channel (C5) escalation rule is owned by the recall-glance-deep-dial spec
