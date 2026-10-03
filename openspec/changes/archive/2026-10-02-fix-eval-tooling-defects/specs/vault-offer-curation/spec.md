## MODIFIED Requirements

### Requirement: Pending offers are surfaced at three points
`engram query` SHALL include a pending-offer-exists flag in its payload on every call, and when the flag is true SHALL also include a `pending_offers_hint` string carrying the same curate instruction the notices carry (omitted when the flag is false, so payloads without pending offers are unchanged). When present, `pending_offers` and `pending_offers_hint` SHALL be the first keys after `version` in the YAML payload, before `items` and every other large field, so a truncated preview of a large payload still shows them. The curate instruction SHALL state that reviewing pending offers is expected vault upkeep, not an extra, that the agent is to curate them after it finishes the user's request without asking, and SHALL carry the copy-pasteable command; the update notice, the write-path nudge and the payload hint SHALL each embed this same instruction from one shared definition, and the notice and nudge SHALL each remain a single line. `engram update` SHALL include a notify-only notice when pending offers exist, following the same detect-and-notify convention as its other vault-condition detectors. `engram learn`, `engram amend`, and `engram resituate` SHALL log a warning-level nudge, at the same point each already checks the `vocab refit` trigger, when pending offers exist. The update notice and the write-path nudge SHALL each carry a copy-pasteable `engram query` command whose `--text` contains the curate runbook's trigger words, so that following the notice surfaces the curate runbook without naming a skill. The command's flags SHALL use the `--flag="value"` form (the quote touching the `=`, no space), not a space-separated `--text "value"`, consistent with every other shipped `engram` command template (`fix-eval-tooling-defects` D14/D17) — the string is a fixed literal today, not a free-text value that could itself begin with `-`, but the template form stays consistent with the rest of the surface this requirement is part of.

#### Scenario: Query payload reflects current state
- **WHEN** `engram query` runs
- **THEN** its payload's pending-offer-exists flag matches whether any pending offer currently exists

#### Scenario: Query payload carries the hint when offers are pending
- **WHEN** `engram query` runs while pending offers exist
- **THEN** its payload has `pending_offers: true` and a `pending_offers_hint` whose text is the same instruction (the `engram query --text="curate pending offers" ...` command and "follow the curate runbook it returns") that the update notice and write-path nudge embed, taken from one shared definition

#### Scenario: Flag and hint lead the payload
- **WHEN** `engram query` renders a payload larger than 100 KB while pending offers exist
- **THEN** `pending_offers: true` and `pending_offers_hint` appear before `items:`, and the first 1500 bytes of the payload contain the hint

#### Scenario: The instruction says curation is expected upkeep
- **WHEN** the shared curate instruction is read on any of the three surfaces
- **THEN** it says reviewing pending offers is expected vault upkeep, not an extra, says to curate them after finishing the user's request without asking, and gives the `engram query --text="curate pending offers" ...` command

#### Scenario: Query payload has no hint without offers
- **WHEN** `engram query` runs while no pending offer exists
- **THEN** its payload contains neither `pending_offers` nor `pending_offers_hint`, and is byte-identical to a payload produced before the hint existed

#### Scenario: Merged query carries the hint
- **WHEN** a merged query (local plus parent) runs
- **THEN** the merged payload's `pending_offers` and hint reflect the LOCAL vault's pending offers only — a child cannot curate its parent, so a parent-side pending flag is ignored; with no local pending offer the merged payload has neither field

#### Scenario: Served query carries the hint
- **WHEN** `GET /query` is served while pending offers exist on the host
- **THEN** the response body is the same YAML a local query would print, including `pending_offers: true` and `pending_offers_hint`

#### Scenario: Update surfaces a notice
- **WHEN** `engram update` runs while pending offers exist
- **THEN** its report includes a notify-only notice naming the pending offers and the `engram query --text="curate pending offers" ...` command that surfaces the curate runbook

#### Scenario: Write-path nudge fires
- **WHEN** `engram learn`, `engram amend`, or `engram resituate` completes while pending offers exist
- **THEN** a warning-level log line notes their presence and carries the same trigger query

#### Scenario: The notice's command surfaces the runbook
- **WHEN** an agent runs the command a notice prints against a vault that contains the curate runbook
- **THEN** the curate runbook appears in the query result with `trigger` provenance
