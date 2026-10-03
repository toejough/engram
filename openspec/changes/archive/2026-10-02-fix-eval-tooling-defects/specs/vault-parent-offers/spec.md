## MODIFIED Requirements

### Requirement: Local writes SHALL be offered to the parent
When `ENGRAM_PARENT` is set (and the self-parent guard of capability `vault-local-first` does not apply), the writes listed below SHALL first complete locally, exactly as they do without a parent, and SHALL then be queued as offers:
- `engram learn fact`, `engram learn feedback`, and `engram learn runbook`;
- `engram amend` with any content flag (`--situation`, `--subject`, `--predicate`, `--object`, `--behavior`, `--impact`, `--action`, `--done-when`, `--body`, `--red-flag`, `--trigger`);
- `engram resituate`;
- `engram amend --clear-pending` of a note that carries `offer.origin`, which is an accepted served offer. This is multi-level propagation. It SHALL NOT be queued when the origin vault ID in `offer.origin` equals the configured parent's vault ID.

A queued offer SHALL be an **amend-offer** targeting the note's primary parent link when the note has one for the configured parent's vault ID. Otherwise it SHALL be a **learn-offer**.

The following SHALL NOT be offered:
- bookkeeping amends: `--activate`; `--clear-pending` of any other note; `--discard` with or without `--into`; amends whose only flags are `--supersedes` or `--chunk-source`;
- identity backfill;
- `engram learn qa` notes;
- notes carrying `skill_hash`;
- pending notes;
- any write whose resulting exchange hash equals the note's primary link hash.

Offers SHALL carry notes only, never transcript chunks.

#### Scenario: learn is written locally and offered
- **WHEN** `engram learn fact ...` runs with `ENGRAM_PARENT` set and the parent reachable
- **THEN** the note exists in the local vault as a live note, and the parent receives one learn-offer for it

#### Scenario: A content amend of a linked note is an amend-offer
- **WHEN** `engram amend --target L --object="..."` runs on a local note L whose primary link names parent note P
- **THEN** the parent receives an offer whose `offer.for` is P

#### Scenario: A runbook-field amend is offered
- **WHEN** `engram amend --target R --red-flag="..."` runs on a linked runbook R
- **THEN** an amend-offer is queued, because R's exchange hash changed

#### Scenario: A content amend of an unlinked note is a learn-offer
- **WHEN** `engram amend --target L --object="..."` runs on a local note L with no parent link
- **THEN** the parent receives a learn-offer carrying L's current content

#### Scenario: Bookkeeping amends stay local
- **WHEN** `engram amend --target L --activate`, `--discard`, or an amend with only `--supersedes` runs, or `--clear-pending` runs on a note without `offer.origin`
- **THEN** no offer is queued or sent

#### Scenario: Accepting a served offer propagates upward
- **WHEN** a vault that serves children and has its own `ENGRAM_PARENT` clears the pending marker on a served offer whose origin is a child vault
- **THEN** a learn-offer for that note is queued to its own parent, carrying the accepted note's `offer.path` with this vault's ID appended

#### Scenario: An accepted offer never returns to its origin vault
- **WHEN** the accepted note's `offer.origin` names the configured parent's vault ID
- **THEN** no offer is queued

#### Scenario: Identity backfill stays local
- **WHEN** `engram update --backfill-identity` rewrites notes
- **THEN** no offer is queued for any of them

#### Scenario: Registration notes and qa notes are never offered
- **WHEN** a note carrying `skill_hash` is written by registration, or `engram learn qa` writes a note
- **THEN** no offer is queued for it

#### Scenario: An unchanged round-trip is suppressed
- **WHEN** an amend leaves a note whose exchange hash equals its primary link hash
- **THEN** no offer is queued

#### Scenario: A near-fold of a pulled note bounces up once
- **WHEN** local curation folds a pulled parent note's additional claim into local note L with a content amend
- **THEN** exactly one offer for L is queued, and the resulting parent note is pulled down again only on a later activate, and only if its exchange hash differs

#### Scenario: No parent configured
- **WHEN** `ENGRAM_PARENT` is not set
- **THEN** learn, amend, and resituate behave exactly as before this capability, and no exchange state is written

