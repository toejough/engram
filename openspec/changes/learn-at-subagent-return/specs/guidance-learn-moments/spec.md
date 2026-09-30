## ADDED Requirements

### Requirement: Agent SHALL fire /learn on the fast path when a subagent returns a worth-keeping LESSONS line

When a dispatched subagent's completion report returns and its `LESSONS:` line is not `none`, the orchestrating agent SHALL judge each lesson in that line at that moment against the learn skill's Step-2 bar: it maps to one of the four kinds (correction, explicit save-request, reversal, confirmed approach), it is confirmed rather than hypothesized, and it states a general reusable principle rather than a session-specific narrative. For each lesson that clears the bar, the agent SHALL invoke the engram `/learn` skill (the `Skill` tool with `skill=learn`) on its mid-cycle fast path — one note per lesson, no `engram ingest --auto` sweep — and SHALL do so before it dispatches the next subagent or ends its turn. The cue SHALL be stated in the ambient learn guidance (`agent-instructions/guidance/learn.md`), so it applies to every dispatch whether or not a route or please skill mediated it.

#### Scenario: Confirmed correction returned

- **WHEN** a subagent returns with `LESSONS: the reviewer rejected local-time timestamps; the convention here is UTC ISO-8601 in front matter`, and the orchestrator has another unit left to dispatch
- **THEN** the orchestrator invokes `/learn` for that lesson, with no `engram ingest --auto` between the return and the note write, before it dispatches the next unit

#### Scenario: Confirmed approach returned

- **WHEN** a subagent returns with a LESSONS lesson stating that a targeted reproduction test confirmed a suspected defect before the fix was written
- **THEN** the orchestrator captures it with `/learn` on the fast path before the next dispatch, as a confirmed approach (kind 4)

#### Scenario: Several subagents return together

- **WHEN** two parallel subagents return and each report carries one lesson that clears the bar
- **THEN** the orchestrator captures both lessons, one note each, before its next dispatch

#### Scenario: Ad hoc dispatch without route or please

- **WHEN** the orchestrator dispatched a subagent directly with the Agent tool, without invoking route or please, and the report carries a lesson that clears the bar
- **THEN** the cue applies all the same, because it lives in the ambient learn guidance

### Requirement: Agent SHALL NOT fire /learn for a returned LESSONS line that fails the bar

The return cue SHALL NOT fire when the returned `LESSONS:` line is `none`, when a lesson is a bare success or routine narrative ("finished the unit, tests pass"), or when a lesson is an unconfirmed hypothesis ("maybe the timeout is a race"). Such lessons are left for the closing learn, which discards them silently under the same bar.

#### Scenario: LESSONS none

- **WHEN** a subagent returns with `LESSONS: none`
- **THEN** the orchestrator does not invoke `/learn` before the next dispatch

#### Scenario: Trivial lesson

- **WHEN** a subagent returns with `LESSONS: completed the unit, all tests pass`
- **THEN** the orchestrator does not invoke `/learn` before the next dispatch

#### Scenario: Unconfirmed hunch

- **WHEN** a subagent returns with `LESSONS: the build flake might be a race in the cache warmer`, with nothing in the report confirming it
- **THEN** the orchestrator does not invoke `/learn` before the next dispatch
