## MODIFIED Requirements

### Requirement: The shim SHALL name the first action as a literal `engram query` on every user request

Before the first tool call on any user request, the shim SHALL instruct the agent to run `engram query` with at least two `--phrase` values (the request in the agent's own words; the kind of situation it is in), as a bare imperative naming the command, so that the instruction is executable with no `recall` skill present. The trigger SHALL NOT depend on the agent judging the task multi-step, recurring, or previously done: that judgment is what the returned runbook supplies, so it cannot gate the lookup. Over-fire on one-shot requests is accepted (task-init fire-unit, measured 3.4x; per-fire cost is one query with no crystallization). Adjacent to the phrase-shape instruction, the shim SHALL carry a generic re-check on the second phrase: if it contains a concrete noun from the ticket (a file, flag, feature, bug, or field name) or reads like a paraphrase of the request, the agent SHALL rewrite it as the kind of work-handling decision (how to carry out, verify, review, or land this class of work) before running the query. The shim SHALL illustrate the second-phrase shape with at least two examples of different kinds of work, and the re-check and examples SHALL NOT name any specific runbook, skill, or eval task.

#### Scenario: Shim-only session runs the first query

- **WHEN** an agent with the shim in CLAUDE.md and no engram skills installed receives a user request
- **THEN** its first tool call is an `engram query` carrying both the request phrase and the situation phrase, before any other tool call

#### Scenario: A one-shot request still fires

- **WHEN** the request is a single-step action (a one-line answer, a single-file typo fix)
- **THEN** the query still runs first; the agent proceeds without a runbook when none matches

#### Scenario: Deliverable-laden second phrase is rewritten before the query runs

- **WHEN** the agent's draft second phrase contains a ticket-specific noun (for example a flag, file, or endpoint name) or restates the request
- **THEN** the second `--phrase` actually passed to `engram query` names a class of work-handling decision and contains none of those ticket-specific nouns

#### Scenario: Re-check wording is runbook-agnostic

- **WHEN** the shim's phrase-two re-check and its examples are read
- **THEN** they mention no specific runbook, skill, or eval task, and the examples cover at least two different kinds of work

#### Scenario: Re-check does not regress other runbooks' retrieval

- **WHEN** a previously validated runbook's task (for example route's dispatch-tier task) is run shim-only with the updated shim
- **THEN** that runbook is still retrieved at the same rate as before the shim change
