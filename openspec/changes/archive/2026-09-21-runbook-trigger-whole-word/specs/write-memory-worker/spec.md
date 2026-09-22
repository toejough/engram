## MODIFIED Requirements

### Requirement: Write-memory worker SHALL pass runbook triggers through as `--trigger` flags
When a kind=runbook handoff carries an optional `triggers` list, the worker SHALL append one `--trigger "<cue>"` per entry, in order, to the `engram learn runbook` command; when the handoff carries none, no `--trigger` flag is emitted. The template SHALL state that a trigger is a cue word or phrase the user (or an engram notice) would literally write, that a single word must be distinctive enough that whole-word matching will not fire on ordinary prose (a very common word is questioned unless the handoff states over-firing is deliberate), and that matching is case-insensitive, whitespace-collapsed, and whole-word at letter/digit edges.

#### Scenario: Runbook handoff with triggers
- **WHEN** the parent hands off kind=runbook with `triggers: ["/please", "take this end-to-end"]`
- **THEN** the composed command ends with `--trigger "/please" --trigger "take this end-to-end"` (after any `--red-flag` flags)

#### Scenario: Runbook handoff without triggers
- **WHEN** the parent hands off kind=runbook with no `triggers`
- **THEN** the composed command contains no `--trigger` flag
