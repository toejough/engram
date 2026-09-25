## ADDED Requirements

### Requirement: The pending-offer marker SHALL apply to skill runbook notes
A runbook note carrying `skill_hash` (capability `skill-runbook-registration`) and the pending-offer marker SHALL be treated as a pending offer exactly as a pending fact or feedback note is: excluded from normal query results, counted by pending-offer detection, and reported at every pending-offer surfacing point. A runbook note without `skill_hash` SHALL NOT be affected by this requirement.

#### Scenario: A newly registered skill note is pending
- **WHEN** a skill's registration is accepted and its note carries the pending-offer marker
- **THEN** `engram query` omits the note from its results and its payload has `pending_offers: true`

#### Scenario: Clearing the marker makes the skill note live
- **WHEN** curation clears the marker on a skill note whose runbook fields are authored
- **THEN** the note surfaces in `engram query` results like any runbook, and it no longer counts as a pending offer

## MODIFIED Requirements

### Requirement: Curation judges pending offers the same way recall judges candidates
Curation of a pending offer SHALL use the same covered/near/absent judgment `recall`'s Step 2.5 already performs against query candidates, applied instead to the pending offer against the host vault's existing notes. The curation procedure SHALL be carried by the curate skill, whose runbook note in the vault (per capability `skill-runbook-registration`) lets `engram query` surface it by situation and trigger. A **covered** offer SHALL be discarded (the existing note is reinforced with `engram amend --activate`, then the offer is deleted with `engram amend --discard`). A **near** offer SHALL be folded into the existing note it overlaps with via `engram amend`, and the now-redundant offer SHALL then be discarded. An **absent** offer SHALL be accepted as a normal note by clearing its own pending-offer marker with `engram amend --clear-pending`. The pending-offer marker is cleared only on an offer that is kept (the absent case); a discarded offer carries no marker afterwards. A pending **skill runbook note** (a runbook note carrying `skill_hash`) SHALL NOT be judged covered/near/absent and SHALL NOT be discarded by curation: curation SHALL instead author its `situation`, `done_when`, `triggers`, and `red_flags` from its body when it has none, or re-check existing ones against its body after a refresh (amending any that no longer fit, with `red_flags` within the 1200-byte rendered cap), and then clear its marker with `engram amend --clear-pending`. Curation SHALL run asynchronously, never synchronously within the HTTP request that created the offer.

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

#### Scenario: A new skill note gets its runbook fields
- **WHEN** curation handles a pending skill runbook note with no `situation`, `done_when`, `triggers`, or `red_flags`
- **THEN** it amends the note to add those fields, authored from the note's body, clears the marker, and does not discard the note

#### Scenario: A refreshed skill note is re-checked, not discarded
- **WHEN** curation handles a pending skill runbook note that already has runbook fields
- **THEN** it amends any field that no longer matches the body, clears the marker, and does not discard the note even if another note covers similar ground
