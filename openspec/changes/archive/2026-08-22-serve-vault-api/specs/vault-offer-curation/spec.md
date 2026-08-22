## Purpose

Gives served writes a curated acceptance step instead of an immediate commit: a note arriving over the API is evaluated — discarded, folded into an existing note, or accepted — using the same agent-judged covered/near/absent reasoning `recall` already performs, before it becomes a normal, live vault note.

## ADDED Requirements

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
Curation of a pending offer SHALL use the same covered/near/absent judgment `recall`'s Step 2.5 already performs against query candidates, applied instead to the pending offer against the host vault's existing notes. A **covered** offer SHALL be discarded. A **near** offer SHALL be folded into the existing note it overlaps with, via `engram amend`. An **absent** offer SHALL be accepted as a normal note. In every case the pending-offer marker SHALL be cleared once curated. Curation SHALL run asynchronously, never synchronously within the HTTP request that created the offer.

#### Scenario: A covered offer is discarded
- **WHEN** curation judges a pending offer as covered by an existing note
- **THEN** the offer is discarded and no new or modified note results from it

#### Scenario: A near offer is folded into an existing note
- **WHEN** curation judges a pending offer as near an existing note
- **THEN** the existing note is amended to incorporate the offer, and the pending offer's marker is cleared

#### Scenario: An absent offer is accepted
- **WHEN** curation judges a pending offer as absent from the vault
- **THEN** the offer's pending-offer marker is cleared and it becomes a normal, live note

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
`engram query` SHALL include a pending-offer-exists flag in its payload on every call. `engram update` SHALL include a notify-only notice when pending offers exist, following the same detect-and-notify convention as its other vault-condition detectors. `engram learn`, `engram amend`, and `engram resituate` SHALL log a warning-level nudge, at the same point each already checks the `vocab refit` trigger, when pending offers exist.

#### Scenario: Query payload reflects current state
- **WHEN** `engram query` runs
- **THEN** its payload's pending-offer-exists flag matches whether any pending offer currently exists

#### Scenario: Update surfaces a notice
- **WHEN** `engram update` runs while pending offers exist
- **THEN** its report includes a notify-only notice naming the pending offers

#### Scenario: Write-path nudge fires
- **WHEN** `engram learn`, `engram amend`, or `engram resituate` completes while pending offers exist
- **THEN** a warning-level log line notes their presence
