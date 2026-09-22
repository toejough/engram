## Purpose

Gives served writes a curated acceptance step instead of an immediate commit: a note arriving over the API is evaluated — discarded, folded into an existing note, or accepted — using the same agent-judged covered/near/absent reasoning `recall` already performs, before it becomes a normal, live vault note.

## Requirements

### Requirement: Served writes land as pending offers, not immediate notes
A `learn` or `amend` request handled by `engram serve` SHALL be written with a pending-offer marker rather than being immediately treated as a normal, live note.

#### Scenario: A served learn request creates a pending offer
- **WHEN** a `learn` request is handled over the served API
- **THEN** the resulting note carries the pending-offer marker

### Requirement: activate commits directly and never becomes a pending offer
`engram activate`, though part of the served write set, SHALL commit directly (bump the target note's sidecar `LastUsed`) without creating a pending offer and without going through curation.

#### Scenario: A served activate request commits immediately
- **WHEN** an `activate` request is handled over the served API
- **THEN** the named note's sidecar `LastUsed` is updated immediately, with no pending-offer marker created and no curation step involved

### Requirement: Pending offers are excluded from normal query results
`engram query` SHALL NOT return a pending-offer note as part of its normal candidate/result set.

#### Scenario: A pending offer does not surface in recall
- **WHEN** `engram query` runs while a pending-offer note exists in the vault
- **THEN** that note is absent from the returned results

### Requirement: Curation judges pending offers the same way recall judges candidates
Curation of a pending offer SHALL use the same covered/near/absent judgment `recall`'s Step 2.5 already performs against query candidates, applied instead to the pending offer against the host vault's existing notes. The curation procedure SHALL be carried by the curate runbook (a vault note of type `runbook`, surfaced by `engram query`), not by a skill. A **covered** offer SHALL be discarded (the existing note is reinforced with `engram amend --activate`, then the offer is deleted with `engram amend --discard`). A **near** offer SHALL be folded into the existing note it overlaps with via `engram amend`, and the now-redundant offer SHALL then be discarded. An **absent** offer SHALL be accepted as a normal note by clearing its own pending-offer marker with `engram amend --clear-pending`. The pending-offer marker is cleared only on an offer that is kept (the absent case); a discarded offer carries no marker afterwards. Curation SHALL run asynchronously, never synchronously within the HTTP request that created the offer.

#### Scenario: A covered offer is discarded
- **WHEN** curation judges a pending offer as covered by an existing note
- **THEN** the existing note is reinforced, the offer is discarded, and no new or modified note content results from it

#### Scenario: A near offer is folded into an existing note
- **WHEN** curation judges a pending offer as near an existing note
- **THEN** the existing note is amended to incorporate the offer's additional claim, and the pending offer is discarded so no second live note remains

#### Scenario: An absent offer is accepted
- **WHEN** curation judges a pending offer as absent from the vault
- **THEN** the offer's pending-offer marker is cleared and it becomes a normal, live note

#### Scenario: Curation never composes a new learn
- **WHEN** curation handles any offer
- **THEN** every action is an `engram amend` call on existing files, with no `engram learn` and no handoff to `write-memory`

### Requirement: Curation outcome is not reported back to the offering caller
The served API's response to a `learn`/`amend` request that creates a pending offer SHALL confirm only that the offer was received, not its eventual curation outcome. No later notification of the outcome SHALL be sent to the offering caller.

#### Scenario: Server responds before curation happens
- **WHEN** a served `learn` request creates a pending offer
- **THEN** the server's response is returned before curation has run, and no subsequent message reports the offer's disposition

### Requirement: Pending-offer detection is a stateless, unbatched scan
The system SHALL determine whether any pending offers exist by scanning current vault state each time it is checked. It SHALL NOT persist a pending-offer flag across checks, and SHALL NOT withhold surfacing while accumulating a growth or time threshold (unlike the `vocab refit` trigger).

#### Scenario: A single pending offer surfaces immediately
- **WHEN** exactly one pending offer exists in the vault
- **THEN** every surfacing point (see below) reflects its presence on the very next check, with no accumulation delay

### Requirement: Pending offers are surfaced at three points
`engram query` SHALL include a pending-offer-exists flag in its payload on every call, and when the flag is true SHALL also include a `pending_offers_hint` string carrying the same curate instruction the notices carry (omitted when the flag is false, so payloads without pending offers are unchanged). When present, `pending_offers` and `pending_offers_hint` SHALL be the first keys after `version` in the YAML payload, before `items` and every other large field, so a truncated preview of a large payload still shows them. The curate instruction SHALL state that reviewing pending offers is expected vault upkeep, not an extra, that the agent is to curate them after it finishes the user's request without asking, and SHALL carry the copy-pasteable command; the update notice, the write-path nudge and the payload hint SHALL each embed this same instruction from one shared definition, and the notice and nudge SHALL each remain a single line. `engram update` SHALL include a notify-only notice when pending offers exist, following the same detect-and-notify convention as its other vault-condition detectors. `engram learn`, `engram amend`, and `engram resituate` SHALL log a warning-level nudge, at the same point each already checks the `vocab refit` trigger, when pending offers exist. The update notice and the write-path nudge SHALL each carry a copy-pasteable `engram query` command whose `--text` contains the curate runbook's trigger words, so that following the notice surfaces the curate runbook without naming a skill.

#### Scenario: Query payload reflects current state
- **WHEN** `engram query` runs
- **THEN** its payload's pending-offer-exists flag matches whether any pending offer currently exists

#### Scenario: Query payload carries the hint when offers are pending
- **WHEN** `engram query` runs while pending offers exist
- **THEN** its payload has `pending_offers: true` and a `pending_offers_hint` whose text is the same instruction (the `engram query --text "curate pending offers" ...` command and "follow the curate runbook it returns") that the update notice and write-path nudge embed, taken from one shared definition

#### Scenario: Flag and hint lead the payload
- **WHEN** `engram query` renders a payload larger than 100 KB while pending offers exist
- **THEN** `pending_offers: true` and `pending_offers_hint` appear before `items:`, and the first 1500 bytes of the payload contain the hint

#### Scenario: The instruction says curation is expected upkeep
- **WHEN** the shared curate instruction is read on any of the three surfaces
- **THEN** it says reviewing pending offers is expected vault upkeep, not an extra, says to curate them after finishing the user's request without asking, and gives the `engram query --text "curate pending offers" ...` command

#### Scenario: Query payload has no hint without offers
- **WHEN** `engram query` runs while no pending offer exists
- **THEN** its payload contains neither `pending_offers` nor `pending_offers_hint`, and is byte-identical to a payload produced before the hint existed

#### Scenario: Merged query carries the hint
- **WHEN** a merged query (local plus parent) runs and either source reports pending offers
- **THEN** the merged payload has `pending_offers: true` and the hint, and with neither source pending it has neither

#### Scenario: Served query carries the hint
- **WHEN** `GET /query` is served while pending offers exist on the host
- **THEN** the response body is the same YAML a local query would print, including `pending_offers: true` and `pending_offers_hint`

#### Scenario: Update surfaces a notice
- **WHEN** `engram update` runs while pending offers exist
- **THEN** its report includes a notify-only notice naming the pending offers and the `engram query --text "curate pending offers" ...` command that surfaces the curate runbook

#### Scenario: Write-path nudge fires
- **WHEN** `engram learn`, `engram amend`, or `engram resituate` completes while pending offers exist
- **THEN** a warning-level log line notes their presence and carries the same trigger query

#### Scenario: The notice's command surfaces the runbook
- **WHEN** an agent runs the command a notice prints against a vault that contains the curate runbook
- **THEN** the curate runbook appears in the query result with `trigger` provenance
