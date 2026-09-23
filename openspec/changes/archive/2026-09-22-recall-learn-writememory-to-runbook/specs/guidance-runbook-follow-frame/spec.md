## ADDED Requirements

### Requirement: A runbook named by a skill's own instructions SHALL receive the same follow-frame treatment as a matched runbook

When a skill's own instructions (not a query-matched runbook) name a specific runbook by basename or `[[wikilink]]` as the agent's next action, the agent SHALL fetch it (`engram show <basename>`) and apply the same follow-frame obligations as a query-matched runbook: announce it by name, restate its steps as the plan, treat its `done_when` as the completion bar, and read and react to its `red_flags`. This generalizes the existing wikilinked-transitively requirement (runbook body → runbook) to the case where the referring text is still a skill, not a runbook.

#### Scenario: A skill names a worker runbook as its next action

- **WHEN** a skill's instructions read "fetch and follow `[[<basename>]]` with this handoff" at a write site
- **THEN** the transcript shows `engram show <basename>` and the runbook's steps restated before the write is composed

#### Scenario: The named runbook's red_flags still apply

- **WHEN** the runbook fetched by name carries `red_flags`
- **THEN** the agent reads them and stops if one fires, exactly as it would for a runbook matched by `engram query`
