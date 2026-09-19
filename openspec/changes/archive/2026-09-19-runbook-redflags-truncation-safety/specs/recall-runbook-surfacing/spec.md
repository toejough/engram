## MODIFIED Requirements

### Requirement: A surfaced runbook SHALL render its `red_flags` and be retrievable in full

When a `runbook` note appears in a query payload (`items[]` or `candidate_l2s`), its rendered content SHALL include the `red_flags` field when present. `engram show <basename>` SHALL return the full note (frontmatter and body) so an agent can restate every step when the payload's inline content is truncated. When a runbook's `red_flags` list is large enough that the calling harness's own output truncation would otherwise drop entries silently, `engram query` and `engram show` SHALL both keep the most-recently-added entries and SHALL replace any dropped earlier entries with an explicit in-band marker naming the omission and how to retrieve the full list — `engram show <basename>` is not exempt from this guarantee merely because it is the prescribed fallback for a truncated preview.

#### Scenario: Red flags visible in the query payload
- **WHEN** a runbook note with `red_flags` is returned by `engram query`
- **THEN** the item's content includes the `red_flags` entries

#### Scenario: Full runbook via show
- **WHEN** an agent runs `engram show <runbook basename>`
- **THEN** the output contains the complete frontmatter (situation, done_when, red_flags if any) and the full step body

#### Scenario: Oversized red_flags list keeps its newest entry
- **WHEN** a runbook's `red_flags` list is large enough that rendering it in full would exceed the external output-truncation boundary the calling harness applies
- **THEN** both `engram query`'s item content and `engram show`'s output keep the most-recently-added `red_flags` entries and include an explicit marker naming how many earlier entries were omitted and how to retrieve them
