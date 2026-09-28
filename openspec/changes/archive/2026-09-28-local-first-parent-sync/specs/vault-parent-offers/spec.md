## ADDED Requirements

### Requirement: The exchange hash SHALL cover every offered content field
The exchange hash of a note SHALL be a hash over its `type`, `situation`, `subject`, `predicate`, `object`, `behavior`, `impact`, `action`, `done_when`, `red_flags`, `triggers`, and body text, in a canonical serialization. It SHALL NOT depend on any other frontmatter field: identity, `pending`, `tags`, `sources`, `supersedes`, `xid`, `parent`, `aliases`, `offer`, or skill fields. Every YAML key of the fact, feedback, and runbook frontmatter structs SHALL be classified, in one explicit table, as offered (hashed) or not offered. A test SHALL fail when a key is unclassified. Hashes SHALL carry a version prefix. A comparison in which either side has a different or missing version SHALL be *unknown*, not *changed*:
- loop suppression SHALL fire only on *equal*;
- the pull-down skip, decline matching, rejected-entry re-arm, and the send/apply change check SHALL treat *unknown* as *not changed*;
- dedupe SHALL require *equal*.

Exchange SHALL use the exchange hash, not the sidecar content hash, for every one of these:
- link hashes;
- loop suppression;
- idempotency keys;
- rejected-entry re-arm;
- send/apply change checks;
- the pull-down skip and declines;
- merged-query dedupe.

#### Scenario: Every offered field moves the hash
- **WHEN** only a feedback note's `impact`, or only a runbook note's `done_when`, `red_flags`, or `triggers`, is changed
- **THEN** the note's exchange hash changes

#### Scenario: Every frontmatter key is classified
- **WHEN** a YAML key is added to the fact, feedback, or runbook frontmatter struct without being classified as offered or not offered
- **THEN** the classification test fails

#### Scenario: Non-content fields do not move the hash
- **WHEN** only a note's `repo`, `user`, `pending`, `tags`, or `parent` link changes
- **THEN** its exchange hash is unchanged

#### Scenario: A hash version change is unknown, not changed
- **WHEN** a link records a hash with an older version prefix, and the parent note is activated again
- **THEN** the pull-down treats it as not changed and writes nothing, and a later exchange of that link records the current-version hash

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
- **WHEN** `engram amend --target L --object "..."` runs on a local note L whose primary link names parent note P
- **THEN** the parent receives an offer whose `offer.for` is P

#### Scenario: A runbook-field amend is offered
- **WHEN** `engram amend --target R --red-flag "..."` runs on a linked runbook R
- **THEN** an amend-offer is queued, because R's exchange hash changed

#### Scenario: A content amend of an unlinked note is a learn-offer
- **WHEN** `engram amend --target L --object "..."` runs on a local note L with no parent link
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

### Requirement: Offer payloads SHALL be translated for the parent vault
An offer SHALL be built from the note's current content at send time. It SHALL NOT carry `target`, `position`, or `chunkSources`. Each `supersedes` entry naming a local note SHALL be rewritten to that note's primary-link parent basename, and SHALL be omitted when there is none. The offer SHALL declare the note's own `user`/`repo` frontmatter values, falling back to the sending process's detection only when they are empty. It SHALL carry `offer.origin` (the local vault ID and the note's `xid`), `offer.key`, `offer.path` (the vault IDs the offer has passed through: the local ID for a direct offer, and the accepted note's `offer.path` plus the local ID for a propagated one), and `offer.for` for amend-offers.

#### Scenario: Placement is not sent
- **WHEN** a note learned with `--target 12 --position child` is offered
- **THEN** the offer request carries no `target` and no `position`

#### Scenario: Supersedes is translated or dropped
- **WHEN** an offered note supersedes local note A, which is linked to parent note PA, and local note B, which is unlinked
- **THEN** the offer's `supersedes` names PA and does not name A or B

#### Scenario: The note's author is declared, not the drainer's
- **WHEN** a note whose `user:` is alice is drained by a command run as bob
- **THEN** the offer declares `user` alice

### Requirement: Offers SHALL be queued in a local outbox and never lost offline
Every offerable write SHALL stamp the note's `xid` (if it has none) and add or refresh the note's outbox entry, keyed by `xid`, under the vault lock and in the same critical section as the note write. There SHALL be at most one entry per note. The local write SHALL succeed whether or not the parent is reachable. An entry whose note no longer exists, or now carries the pending-offer marker, SHALL be dropped when it is sent. Renaming a note SHALL NOT orphan its entry.

#### Scenario: Parent unreachable during learn
- **WHEN** `engram learn fact ...` runs while the parent is unreachable
- **THEN** the command exits zero, the note is live locally, the outbox has one entry for it, and stderr carries one warning naming the queued-offer count

#### Scenario: Offline learn then amend coalesce
- **WHEN** a note is learned and then amended twice while the parent is unreachable, and the parent then becomes reachable
- **THEN** exactly one offer is sent for that note, and it carries the note's latest content

#### Scenario: A renamed note keeps its entry
- **WHEN** a queued note is renamed by Luhmann reparenting before the outbox drains
- **THEN** the drain sends the renamed note's offer, and records the link on the renamed note

#### Scenario: A discarded note's entry is dropped
- **WHEN** a queued note is discarded locally before the outbox drains
- **THEN** the drain sends nothing for it and removes its entry

### Requirement: The outbox SHALL drain after successful parent contact, without holding the lock
After a command successfully contacts the parent (`learn`, content `amend`, `resituate`, `query` after the parent `/query` succeeds, `activate` after a pull-down, or `update`), the command SHALL send the queued entries in first-queued order, holding no vault lock while it does. It SHALL then re-take the lock, re-read the outbox from disk, and merge the outcomes, keeping entries enqueued concurrently. A receipt for a note that no longer exists SHALL be discarded.

A transport error, timeout, or 5xx SHALL stop the drain, keep the remaining entries, and record the attempt count and last error on the failed entry. A 4xx SHALL mark that entry `rejected` along with its exchange hash, and the drain SHALL continue. A rejected entry SHALL NOT be resent until the note's exchange hash changes.

#### Scenario: Query drains a backlog
- **WHEN** the outbox holds three entries and `engram query` succeeds against the parent
- **THEN** the three offers are sent in first-queued order, and the outbox is empty afterwards

#### Scenario: A transport failure stops the drain
- **WHEN** the second of three sends fails with a connection error
- **THEN** the first entry is removed, the second and third remain in order, and the second records `attempts` and `last_error`

#### Scenario: A rejected offer does not block the queue
- **WHEN** the parent answers the first entry with a 4xx
- **THEN** that entry is marked `rejected`, the next entries are still sent, and the rejected entry is not resent until its note's exchange hash changes

#### Scenario: A concurrent enqueue survives the drain
- **WHEN** another process queues a new entry while a drain is sending
- **THEN** that entry is still in the outbox after the drain merges its results

### Requirement: An unreachable parent SHALL be backed off
After a transport failure or timeout against the parent, the vault's parent cache SHALL record a retry time that grows exponentially with consecutive failures, starting at 30 seconds and capped at 15 minutes. Until that time, every command except `engram update` SHALL skip parent contact: queries return local results, and offers are only queued. Each such command SHALL print exactly one warning. Parent requests SHALL use a connect timeout of at most 3 seconds. A successful contact SHALL reset the failure count.

#### Scenario: Commands inside the backoff window make no request
- **WHEN** a query fails to reach the parent, and a second query runs 10 seconds later
- **THEN** the second query makes no parent request, returns local results, and prints one warning

#### Scenario: update always tries
- **WHEN** `engram update` runs inside a backoff window
- **THEN** it attempts the parent and drains on success

### Requirement: Offers SHALL be idempotent across retries
`offer.key` SHALL be derived from `offer.origin` and the offered exchange hash. A retry of an offer the parent already holds SHALL NOT create a second pending note (capability `vault-serve-api`).

#### Scenario: Lost response, then retry
- **WHEN** the parent stores an offer but the response is lost, and the entry is resent unchanged
- **THEN** the parent returns the receipt of the pending note it already has, and the parent vault holds one pending note for that offer

### Requirement: A receipt SHALL record the parent counterpart on the local note
When the parent returns an offer receipt, the local note SHALL record the parent's vault ID. It SHALL set its primary link to `{note: <resolved target basename when the receipt names one, else the receipt's basename>, via: offered, hash: <the receipt's stored_hash>}`. The link records the hash the parent stored, not the hash the child computed. Recording the link SHALL be a frontmatter-only rewrite under the vault lock. It SHALL NOT re-embed the note, and SHALL NOT re-stamp `repo`/`user`/`vault`. A receipt that lacks a vault ID or a basename SHALL stop exchange with an error saying the parent is too old, and SHALL leave the entry queued. A receipt whose `vault_id` is not 32 lowercase hex characters, or whose `basename` or `for` is not a Luhmann basename free of `/`, `\` and `|`, SHALL be treated as an undecodable reply: nothing is recorded on the note, and the entry stays queued with the failure recorded. A 409 loop refusal whose `vault_id` is malformed SHALL count as a refusal without a vault ID. A malformed vault ID SHALL never be cached as the parent's.

#### Scenario: Receipt links the note
- **WHEN** the parent answers a learn-offer for local note L with basename `1100.2026-09-27.x`
- **THEN** L's primary link is `{note: 1100.2026-09-27.x, via: offered, hash: <stored_hash>}` under the parent's vault ID, and L's sidecar vector is unchanged

#### Scenario: Re-link to the resolved target
- **WHEN** an amend-offer's receipt names a resolved target E different from L's current primary
- **THEN** L's primary link becomes E

#### Scenario: A pre-change parent is detected
- **WHEN** a receipt carries no `vault_id`
- **THEN** the command reports that the parent is too old, and the entry stays queued

#### Scenario: A malformed receipt is not recorded
- **WHEN** a receipt's basename contains `|` or `/`, or its `vault_id` is not 32 lowercase hex characters
- **THEN** the local note gains no parent link, and the entry stays queued with the failure recorded

### Requirement: Outbox state SHALL be reported
`engram update` SHALL include a notify-only notice when the outbox is non-empty or a backoff is active. The notice SHALL give the queued-entry count, the age of the oldest entry, each rejected entry with its error, and the backoff retry time.

#### Scenario: Update reports a stuck outbox
- **WHEN** `engram update` runs while the outbox holds two queued entries and one rejected entry
- **THEN** its report includes one notice with the count, the oldest age, and the rejected entry's error
