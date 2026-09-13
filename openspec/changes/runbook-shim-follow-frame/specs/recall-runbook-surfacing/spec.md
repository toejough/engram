## ADDED Requirements

### Requirement: A surfaced runbook SHALL render its `red_flags` and be retrievable in full

When a `runbook` note appears in a query payload (`items[]` or `candidate_l2s`), its rendered content SHALL include the `red_flags` field when present. `engram show <basename>` SHALL return the full note (frontmatter and body) so an agent can restate every step when the payload's inline content is truncated.

#### Scenario: Red flags visible in the query payload
- **WHEN** a runbook note with `red_flags` is returned by `engram query`
- **THEN** the item's content includes the `red_flags` entries

#### Scenario: Full runbook via show
- **WHEN** an agent runs `engram show <runbook basename>`
- **THEN** the output contains the complete frontmatter (situation, done_when, red_flags if any) and the full step body
