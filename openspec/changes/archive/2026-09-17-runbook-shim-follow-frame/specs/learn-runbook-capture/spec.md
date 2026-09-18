## ADDED Requirements

### Requirement: Runbook notes MAY carry a `red_flags` field of task-specific failure modes

A runbook note SHALL support an optional frontmatter field `red_flags` (a list of strings), each naming a condition specific to this procedure that a general "follow the steps" rule would not catch (e.g. "filter-branch on all refs sweeps the backup branch"). `engram learn runbook` SHALL accept a repeatable `--red-flag <text>` flag that populates it. Entries SHALL be task-specific; the general behavioral floor (shim) is not restated here.

#### Scenario: Runbook captured with red flags
- **WHEN** `engram learn runbook … --red-flag "<text A>" --red-flag "<text B>"` is invoked
- **THEN** the written note's frontmatter contains `red_flags:` with the two entries in order, and the note otherwise matches the runbook schema

#### Scenario: Runbook captured without red flags
- **WHEN** `engram learn runbook` is invoked with no `--red-flag`
- **THEN** the note is written with no `red_flags` field and no error

## MODIFIED Requirements

### Requirement: `write-memory` SHALL compose and execute runbook-note writes

The `write-memory` skill SHALL compose the `engram learn runbook` command from fields handed off by `learn` (slug, situation, done_when, body/steps, source, target/position disposition, optional contributors, optional red_flags list), execute it, verify the result, and report the written note path — consistent with how it handles `fact` and `feedback` handoffs. Each handed-off red flag becomes one `--red-flag` argument.

#### Scenario: write-memory executes a runbook handoff

- **WHEN** `learn` hands off a confirmed runbook (slug, situation, done_when, body, source, position, optional target, optional red_flags) to `write-memory`
- **THEN** `write-memory` composes and runs the corresponding `engram learn runbook` command, with one `--red-flag` per handed-off entry, and reports the written note path
