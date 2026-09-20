## ADDED Requirements

### Requirement: Write-memory worker SHALL pass runbook triggers through as `--trigger` flags
When a kind=runbook handoff carries an optional `triggers` list, the worker SHALL append one `--trigger "<cue>"` per entry, in order, to the `engram learn runbook` command; when the handoff carries none, no `--trigger` flag is emitted. The template SHALL state that a trigger is a distinctive cue (a slash form or multi-word phrase), never a lone common word.

#### Scenario: Runbook handoff with triggers
- **WHEN** the parent hands off kind=runbook with `triggers: ["/please", "take this end-to-end"]`
- **THEN** the composed command ends with `--trigger "/please" --trigger "take this end-to-end"` (after any `--red-flag` flags)

#### Scenario: Runbook handoff without triggers
- **WHEN** the parent hands off kind=runbook with no `triggers`
- **THEN** the composed command contains no `--trigger` flag
