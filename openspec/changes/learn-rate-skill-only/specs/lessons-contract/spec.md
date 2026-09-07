# Lessons Contract Capability

## ADDED Requirements

### Requirement: Dispatched subagent completion report carries LESSONS line

Every subagent dispatched via the route skill (fork, fresh-context, or workflow) MUST include a `LESSONS:` line at the end of its completion report. The line format is one of:
- `LESSONS: none` (when no learnable lessons exist from the subagent's work)
- `LESSONS: <lesson1>` (single lesson)
- `LESSONS: <lesson1>, <lesson2>, <lesson3>` (up to three lessons, comma-separated)

Each lesson is a one-line summary capturing confirmed corrections, surprising findings, or validated approaches. The LESSONS line is a disposable offer tier — intentionally low bar, not a committed vault note.

#### Scenario: Subagent with learnable lessons
- **WHEN** a subagent completes work that contains confirmed corrections or surprising findings
- **THEN** its completion report ends with `LESSONS: <lesson1>, <lesson2>` (the specific findings)

#### Scenario: Subagent with no learnable lessons
- **WHEN** a subagent completes work that was either straightforward execution or testing with no novel findings
- **THEN** its completion report ends with `LESSONS: none`

#### Scenario: LESSONS line is missing
- **WHEN** a subagent's final report does not include a LESSONS line
- **THEN** the orchestrator re-asks for the LESSONS line exactly once (either as a follow-up or in re-dispatch) and collects the result

### Requirement: Orchestrator collects and passes LESSONS lines to closing learn

The orchestrator (please skill) MUST collect all `LESSONS:` lines from every dispatched subagent across the session and pass them as input to the closing learn step. Collection happens regardless of whether any individual line contains "none".

#### Scenario: Multiple subagents dispatch during session
- **WHEN** the session contains three dispatched subagents, each with a LESSONS line
- **THEN** the orchestrator collects all three lines and includes them in the context passed to the closing learn

#### Scenario: No subagents dispatched in session
- **WHEN** the session has no dispatched work (e.g., a direct user question answered without delegation)
- **THEN** the orchestrator still runs the closing learn, receiving an empty LESSONS collection (and closing learn handles gracefully with no action)

#### Scenario: Re-ask recovers missing LESSONS line
- **WHEN** the orchestrator detects a missing LESSONS line and re-asks the subagent
- **THEN** the collected line from the re-ask is included in the closing learn's input (treated the same as an on-time line)
