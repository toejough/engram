## ADDED Requirements

### Requirement: Local writes SHALL be offered to the parent
When `ENGRAM_PARENT` is set, the following writes SHALL first complete locally, exactly as they do without a parent, and SHALL then be queued as an offer to the parent:
- `engram learn fact`, `engram learn feedback`, and `engram learn runbook`;
- `engram amend` with any content flag (`--situation`, `--subject`, `--predicate`, `--object`, `--behavior`, `--impact`, `--action`, `--done-when`, `--body`, `--red-flag`, `--trigger`);
- `engram resituate`.

A queued offer SHALL be a **learn-offer** (a new note) when the local note has no parent link for the currently configured parent. It SHALL be an **amend-offer** targeting the linked parent note (`offer.for`) when such a link exists.

The following SHALL NOT be offered:
- bookkeeping amends: `--activate`, `--clear-pending`, `--discard` (with or without `--into`), and amends whose only flags are `--supersedes` or `--chunk-source`;
- identity backfill (`engram update --backfill-identity`);
- `engram learn qa` notes;
- notes carrying `skill_hash`;
- notes carrying the pending-offer marker;
- a write whose resulting content hash equals the note's recorded `parent.hash`.

Offers SHALL carry notes only, never transcript chunks.

#### Scenario: learn is written locally and offered
- **WHEN** `engram learn fact ...` runs with `ENGRAM_PARENT` set and the parent reachable
- **THEN** the note exists in the local vault as a live note, and the parent receives one learn-offer for it

#### Scenario: A content amend of a linked note is an amend-offer
- **WHEN** `engram amend --target L --object "..."` runs on a local note L whose parent link names parent note P for the configured parent
- **THEN** the parent receives an offer whose `offer.for` is P

#### Scenario: A content amend of an unlinked note is a learn-offer
- **WHEN** `engram amend --target L --object "..."` runs on a local note L with no parent link
- **THEN** the parent receives a learn-offer carrying L's current content

#### Scenario: Bookkeeping amends stay local
- **WHEN** `engram amend --target L --clear-pending` (or `--activate`, `--discard`, or an amend with only `--supersedes`) runs
- **THEN** no offer is queued or sent

#### Scenario: Identity backfill stays local
- **WHEN** `engram update --backfill-identity` rewrites notes
- **THEN** no offer is queued for any of them

#### Scenario: Registration notes and qa notes are never offered
- **WHEN** a note carrying `skill_hash` is written by registration, or `engram learn qa` writes a note
- **THEN** no offer is queued for it

#### Scenario: An unchanged round-trip is suppressed
- **WHEN** an amend leaves a note whose content hash equals its recorded `parent.hash`
- **THEN** no offer is queued

#### Scenario: No parent configured
- **WHEN** `ENGRAM_PARENT` is not set
- **THEN** learn, amend, and resituate behave exactly as before this capability, and no outbox entry is written

### Requirement: Offer payloads SHALL be translated for the parent vault
An offer SHALL NOT carry the local `target` or `position` Luhmann placement, and SHALL NOT carry `chunkSources`. Each `supersedes` entry naming a local note SHALL be rewritten to that note's parent counterpart when the note has a parent link for the configured parent, and SHALL be omitted otherwise. An offer SHALL carry the caller's declared `user`/`repo` identity, as served writes require (capability `vault-serve-api`).

#### Scenario: Placement is not sent
- **WHEN** a note learned with `--target 12 --position child` is offered
- **THEN** the offer request carries no `target` and no `position`

#### Scenario: Supersedes is translated or dropped
- **WHEN** an offered note supersedes local note A, which is linked to parent note PA, and local note B, which is unlinked
- **THEN** the offer's `supersedes` names PA and does not name A or B

### Requirement: Offers SHALL be queued in a local outbox and never lost offline
Every offerable write SHALL add or refresh an entry for that local note in the vault's outbox file (`<vault>/.engram-outbox.json`). The update SHALL happen under the vault lock, in the same critical section as the note write. There SHALL be at most one entry per local note. The offer payload SHALL be built from the note's current content when the entry is sent, not when it is queued. The local write SHALL succeed whether or not the parent is reachable. An entry whose note no longer exists, or now carries the pending-offer marker, SHALL be dropped at send time.

#### Scenario: Parent unreachable during learn
- **WHEN** `engram learn fact ...` runs while the parent is unreachable
- **THEN** the command exits zero, the note is live locally, the outbox has one entry for it, and stderr carries one warning naming the queued-offer count

#### Scenario: Offline learn then amend coalesce
- **WHEN** a note is learned and then amended twice while the parent is unreachable, and the parent then becomes reachable
- **THEN** exactly one offer is sent for that note, and it carries the note's latest content

#### Scenario: A discarded note's entry is dropped
- **WHEN** a queued note is discarded locally before the outbox drains
- **THEN** the drain sends nothing for it and removes its entry

### Requirement: The outbox SHALL drain after any successful parent contact
After any command successfully contacts the parent (`learn`, content `amend`, `resituate`, `query` after the parent `/query` succeeds, `activate` after a pull-down succeeds, `update`), the command SHALL send the queued entries in first-queued order. A transport error, timeout, or 5xx response SHALL stop the drain, keep the remaining entries, and record the attempt count and last error on the failed entry. A 4xx response SHALL mark that entry `rejected` with its error and content hash, and the drain SHALL continue with the next entry. A rejected entry SHALL NOT be resent until the note's content hash changes. HTTP requests SHALL NOT be made while the vault lock is held.

#### Scenario: Query drains a backlog
- **WHEN** the outbox holds three entries and `engram query` succeeds against the parent
- **THEN** the three offers are sent in first-queued order, and the outbox is empty afterwards

#### Scenario: A transport failure stops the drain
- **WHEN** the second of three sends fails with a connection error
- **THEN** the first entry is removed, the second and third remain in order, and the second records `attempts` and `last_error`

#### Scenario: A rejected offer does not block the queue
- **WHEN** the parent answers the first entry with a 4xx
- **THEN** that entry is marked `rejected`, the next entries are still sent, and the rejected entry is not resent until its note's content changes

### Requirement: Offers SHALL be idempotent across retries
Each offer SHALL carry an idempotency key derived from the declaring user, the local basename, and the offered content hash. A retry of an offer the parent already stored as a pending note SHALL NOT create a second pending note.

#### Scenario: Lost response, then retry
- **WHEN** the parent stores an offer but the response is lost, and the entry is resent unchanged
- **THEN** the parent returns the receipt of the pending note it already has, and the parent vault holds one pending note for that offer

### Requirement: A receipt SHALL record the parent counterpart on the local note
When the parent returns an offer receipt that names the pending note's basename, the local note SHALL gain a parent link with these values:
- `url`: the normalized configured parent;
- `note`: the receipt's basename;
- `via`: `offered`;
- `hash`: the content hash that was offered.

For an amend-offer, `note` SHALL keep its existing value. Recording the link SHALL be a frontmatter-only rewrite under the vault lock. It SHALL NOT re-embed the note, and SHALL NOT re-stamp `repo`/`user`/`vault`. If the note's content hash changed between send and receipt, the link SHALL still be recorded, and the entry SHALL stay queued. A receipt without a basename (an older parent) SHALL record no link, and SHALL produce one warning.

#### Scenario: Receipt links the note
- **WHEN** the parent answers a learn-offer for local note L with basename `1100.2026-09-27.x`
- **THEN** L's frontmatter carries `parent.note: 1100.2026-09-27.x`, `parent.via: offered`, the configured URL, and the offered hash; its sidecar vector is unchanged

### Requirement: Outbox state SHALL be reported
`engram update` SHALL include a notify-only notice when the outbox is non-empty. The notice SHALL give the queued-entry count, the age of the oldest entry, and each rejected entry with its error. Any command that queues an entry it cannot send SHALL print exactly one stderr warning giving the queued count.

#### Scenario: Update reports a stuck outbox
- **WHEN** `engram update` runs while the outbox holds two queued entries and one rejected entry
- **THEN** its report includes one notice with the count, the oldest age, and the rejected entry's error
