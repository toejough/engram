## MODIFIED Requirements

### Requirement: A receipt SHALL record the parent counterpart on the local note
When the parent returns an offer receipt, the local note SHALL record the parent's vault ID. It SHALL set its primary link to `{note: <resolved target basename when the receipt names one, else the receipt's basename>, via: offered, hash: <the receipt's stored_hash>}`. The link records the hash the parent stored, not the hash the child computed. Recording the link SHALL be a frontmatter-only rewrite under the vault lock. It SHALL NOT re-embed the note, and SHALL NOT re-stamp `repo`/`user`/`vault`. A receipt that lacks a vault ID or a basename SHALL stop exchange with an error saying the parent is too old, and SHALL leave the entry queued. A receipt whose `vault_id` is not 32 lowercase hex characters, or whose `basename` or `for` is not a Luhmann basename free of `/`, `\` and `|`, SHALL be treated as an undecodable reply: nothing is recorded on the note, and the entry stays queued with the failure recorded. A 409 loop refusal whose `vault_id` is malformed SHALL count as a refusal without a vault ID. A malformed vault ID SHALL never be cached as the parent's.

When the local note refuses the receipt because its frontmatter cannot take the edit (its `parent:` carries a YAML anchor, or its frontmatter does not decode), the note SHALL be left untouched and the entry SHALL move to the `attention` state, keeping the receipt and the exchange hash that was sent. The drain SHALL print one warning naming the note and the reason, at that transition only. An `attention` entry SHALL NOT be re-sent to the parent. Each later drain SHALL retry recording the kept receipt on the note, without contacting the parent and without a warning. When the retry succeeds, the entry SHALL finish as an accepted entry does: removed, or queued again when the note's exchange hash changed since it was sent. An entry whose note is gone or pending SHALL be dropped, as for any other state. A receipt that fails to record for any other reason (for example a write error) SHALL keep the entry queued with the failure recorded, as before.

An offer that cannot be built because the note has no `user:` and no user identity is detected SHALL NOT be sent (the parent would reject `user: ""`). Its entry SHALL move to the `attention` state with the reason "cannot offer: no user identity detected; set git user.email", and the drain SHALL print one warning naming the note, at that transition only. Each later drain SHALL try to build the offer again, without contacting the parent and without a warning while it still cannot; once a user identity is detected the entry SHALL be sent as a queued one.

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

#### Scenario: A refused receipt needs attention
- **WHEN** the parent accepts the offer of local note L, and L's `parent:` carries an anchor that another key aliases
- **THEN** L is byte-identical, the entry is in the `attention` state with the reason and the receipt recorded, and stderr carries one warning naming L and `parent`

#### Scenario: An attention entry is not re-sent
- **WHEN** the outbox holds an `attention` entry for L, L is unchanged, and the outbox drains twice more
- **THEN** nothing is sent for L, no further warning is printed, and the entry stays in the `attention` state

#### Scenario: No user identity holds the offer
- **WHEN** the outbox drains an entry whose note has no `user:` on a host where user detection resolves empty, and then drains twice more
- **THEN** nothing is sent, the entry is in `attention` with the no-identity reason, and one warning was printed; after user detection starts to resolve, the next drain sends the offer

#### Scenario: An attention entry resumes once the note is fixed
- **WHEN** the anchor is removed from L's `parent:` and the outbox drains
- **THEN** the kept receipt is recorded on L without contacting the parent, and the entry is removed, or queued again when L's exchange hash changed since it was sent

### Requirement: Outbox state SHALL be reported
`engram update` SHALL include a notify-only notice when the outbox is non-empty or a backoff is active. The notice SHALL give the queued-entry count, the age of the oldest entry, each rejected entry with its error, each entry in the `attention` state with its note and reason, and the backoff retry time. The queued-entry count, here and in the unreachable-parent warning, SHALL NOT count `rejected` or `attention` entries.

#### Scenario: Update reports a stuck outbox
- **WHEN** `engram update` runs while the outbox holds two queued entries and one rejected entry
- **THEN** its report includes one notice with the count, the oldest age, and the rejected entry's error

#### Scenario: Update reports an entry that needs attention
- **WHEN** `engram update` runs while the outbox holds one queued entry and one `attention` entry for note L
- **THEN** its notice counts one queued offer and one that needs attention, and names L with the reason

## ADDED Requirements

### Requirement: Exchange readers SHALL read a note as its LF form
Every reader that gates an exchange operation on a note's frontmatter SHALL read the note as its LF form (each `\r\n` converted to `\n`), so a note with CRLF line endings takes part in exchange exactly as its LF form does. These readers are: the exchange hash; the exchange-field decode behind `engram show`'s header, the judged-version check and the curation fold; the pending-offer marker check behind query exclusion, pending-offer warnings and the served raw `show`; the outbox and served-learn note scan; offer classification at write time; the offer payload; the offer receipt; the pull-down envelope parse and the decline record; and activate's parent-link re-check. None of them SHALL write a note only to convert it. A receipt that is recorded on a CRLF note SHALL write it as LF in that same write and rebuild its sidecar.

#### Scenario: A CRLF note hashes as its LF form
- **WHEN** a note's frontmatter and body end in `\r\n`
- **THEN** its exchange hash equals the exchange hash of its LF form

#### Scenario: A queued CRLF note is sent
- **WHEN** the outbox drains an entry whose note has CRLF line endings
- **THEN** the offer is sent, the receipt is recorded, and the note is written as LF with a fresh sidecar

#### Scenario: A CRLF pending offer stays pending
- **WHEN** a note carrying `pending: true` has CRLF line endings
- **THEN** query excludes it, and the served raw `show` refuses it as a pending offer

#### Scenario: A CRLF parent note pulls down
- **WHEN** the parent serves a note whose envelope content has CRLF line endings
- **THEN** activate pulls it down as an LF pending copy linked at the parent's hash
