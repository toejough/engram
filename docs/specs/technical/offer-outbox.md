# Offer Outbox - Technical Spec

## 792b36a Orientation

- Ownership: Repository component.
- Responsibility: decides which local writes are offered to the parent vault, stamps an exchange ID on the note and
  queues the offer in a durable per-vault outbox in the same critical section as the write, builds each offer from the
  note as it is when it is sent, sends the queued offers in order while holding no vault lock, folds every outcome back
  into the outbox and the notes (accepted, rejected, refused, failed, withdrawn, held), and reports the outbox's state
  to `engram update`.
- Current realization (informational): `internal/cli/offer_classify.go` (`classifyOffer`), `offer_payload.go`
  (`buildOfferPayload`), `outbox.go` (`enqueueOutbox`, `drainOutbox`, `recordKeptReceipts`, `classifyOfferResponse`) and
  `offer_wiring.go` (`offerHooks`, `drainForCommand`, `ensureParentVaultID`, `newUpdateExchange`, `outboxNotice`). The
  receipt edit it calls is exchange-versioning's ([exchange-versioning](./exchange-versioning.md)) `applyReceiptToContent` ([792b33c9 Recording an offer receipt](./exchange-versioning.md#792b33c9-recording-an-offer-receipt)).
- Qualification: responsibility, the offered writes, the outbox and the drain are stated without naming a function;
  ownership, it owns what is offered, queued and sent, while [exchange-versioning](./exchange-versioning.md) owns the hash and the receipt edit,
  [vault-identity-registry](./vault-identity-registry.md) owns the parent cache, the backoff gate and the location and self-parent checks, the
  [vault-http-server](./vault-http-server.md) owns the route that answers an offer, and the commands own when to drain; independent verification,
  every requirement is tested over an in-memory filesystem, lock, clock and fetch capability; stability, the contract
  survives a new offered command or a new outcome of a send; necessity, without it a child could not share what it
  learns and a local write would depend on its parent being up.
- Supported capabilities: [parent-sync](../product/parent-sync.md).

Rationale: a local write must never fail because of the parent, so the outbox is written under the
vault lock in the write's own critical section, and the offer is built from the note when it is sent, so offline learn and
amend coalesce into one offer carrying the latest content. The queue keys on the note's `xid`, which survives a rename,
where a basename key would lose the entry. The drain sends with no lock held, because a lock held across a slow network
call would block every local writer, and merges under the lock after re-reading the outbox from disk, which is what lets
a command queue an offer while another drains. A transport error stops the drain and backs the parent off, while a 4xx
only rejects that entry, because an answer proves the parent is up.

## 792b36c Behavior Requirements

### 792b36c1 Which writes are offered

Requirement: [792b36c1 Which writes are offered](#792b36c1-which-writes-are-offered) The offer-outbox MUST classify a write as offered exactly when the note is a fact, feedback or
runbook note whose frontmatter parses and carries neither the pending marker nor `skill_hash`, the command is `learn`
(not `learn qa`), `resituate`, or an `amend` that is not a discard and carries a content flag (`--situation`,
`--subject`, `--predicate`, `--object`, `--behavior`, `--impact`, `--action`, `--done-when`, `--body`, `--red-flag` or
`--trigger`) or clears the pending marker (`--clear-pending`) of a note that carries `offer.origin`
whose origin vault is not the configured parent's, and the note's exchange hash is not equal to the hash of its primary
link under the configured parent's vault ID. An amend whose only flags are `--supersedes` or `--chunk-source`, identity
backfill and skill registration MUST NOT be offered. A link under any other vault ID, or any link while the parent's ID
is unknown, MUST NOT count as the primary link.

Supports: [792a21n1a learn is written locally and offered](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1b A content amend of a linked note is an amend-offer](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1c A runbook-field amend is offered](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1d A content amend of an unlinked note is a learn-offer](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1e resituate is offered](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1f Bookkeeping amends stay local](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1g Accepting a served offer propagates upward](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1h An accepted offer never returns to its origin vault](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1i Identity backfill stays local](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1j Registration notes and qa notes are never offered](../product/parent-sync.md#792a21n1-uat-scenarios),
[792a21n1k A pending note is never offered](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1l An unchanged round trip is suppressed](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1m A near-fold of a pulled note bounces up once](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1n A link under another parent is not the parent's](../product/parent-sync.md#792a21n1-uat-scenarios).

Rationale: a content change is what a parent can use, and bookkeeping (a use, a discard, a relation) is local. Accepting
a served offer is itself a local learn at that level, so it is offered onward unless it came from the parent, which
keeps the single-parent tree free of loops. Suppressing an offer whose hash equals the link's hash is what stops a pulled
note from bouncing back, while a link under another parent never suppresses.

#### 792b36c1a Scenarios

```gherkin
Scenario: 792b36c1a1 Every row of the offer table
  Given writes of each kind: learn fact, feedback and runbook, learn qa, resituate, an amend with a content flag, --activate, --discard, --supersedes only, --chunk-source only, a clear-pending of a served offer, a clear-pending of another note, identity backfill and registration
  When each is classified
  Then the learns, the resituate, the content amends and the clear-pending of a served offer from another vault are offered, and every other write is not

Scenario: 792b36c1a2 Each content flag alone is an offer
  Given an amend carrying exactly one content flag
  When it is classified
  Then it is offered, for each of the eleven flags

Scenario: 792b36c1a3 Bookkeeping is never offered
  Given an amend with no content flag, drawn at random
  When it is classified
  Then it is offered only when it clears the pending marker of a served offer whose origin is not the parent

Scenario: 792b36c1a4 Pending and registration notes are never offered
  Given a pending note, or a note that carries skill_hash
  When any command and flags are classified
  Then it is not offered

Scenario: 792b36c1a5 Other notes are not offered
  Given a note with no frontmatter, of another type, or with corrupt frontmatter
  When it is classified
  Then it is not offered

Scenario: 792b36c1a6 An equal hash suppresses the offer
  Given any offered write and a primary link under the parent whose hash equals the note's exchange hash, and another whose hash is a different current one
  When each is classified
  Then the first is suppressed and the second is not
```

### 792b36c2 The offer built from a note

Requirement: [792b36c2 The offer built from a note](#792b36c2-the-offer-built-from-a-note) The offer-outbox MUST build an offer's `/learn` body from the note as it is when sent, read as its
LF form: the `type`, the `slug` of the basename, the vault name, `source`, `project`, `issue`, `tier`, the
situation, behavior, impact, action, subject, predicate, object, `done_when`, `red_flags` and `triggers`, the note's own
`repo` and `user` (the sending process's detection filling in only an empty one), the `supersedes` entries rewritten to
the primary-link parent basename of the note each names (an entry whose target has no such link, or that names no known
note, dropped) and, for a runbook, its body without `Supersedes:` lines. It MUST NOT carry `target`, `position`,
`chunkSources`, a vault key, `pending` or `tags`. It MUST carry `offer.origin` (`<local vault ID>:<xid>`), `offer.key`
(the lowercase hexadecimal SHA-256 of the origin followed by the note's exchange hash), `offer.path` (the local vault ID
for a direct offer, and for a note that carries `offer.origin` the path it already travelled, starting from its origin
vault when it recorded none, with the local vault ID appended) and, when the note has a primary link under the
configured parent's vault ID, `offer.for` set to that link's basename. It MUST fail to build, without contacting the
parent, for a note whose basename has no slug or whose frontmatter is missing or does not parse, and for an unreadable
vault ID or vault, and for a note with no `user` when none is detected ([792b36c15 An entry that waits](#792b36c15-an-entry-that-waits)).

Supports: [792a21n1b A content amend of a linked note is an amend-offer](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1d A content amend of an unlinked note is a learn-offer](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1g Accepting a served offer propagates upward](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1m A near-fold of a pulled note bounces up once](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21n1n A link under another parent is not the parent's](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21p1a Placement is not sent](../product/parent-sync.md#792a21p1-uat-scenarios), [792a21p1b Supersedes is translated or dropped](../product/parent-sync.md#792a21p1-uat-scenarios), [792a21p1c The note's author is declared, not the drainer's](../product/parent-sync.md#792a21p1-uat-scenarios), [792a21p1d The offer names its origin, key and path](../product/parent-sync.md#792a21p1-uat-scenarios), [792a21p1e An amend-offer names its target](../product/parent-sync.md#792a21p1-uat-scenarios),
[792a21p1f Tags are not offered](../product/parent-sync.md#792a21p1-uat-scenarios), [792a21t1a Lost response, then retry](../product/parent-sync.md#792a21t1-uat-scenarios).

Rationale: placement is each vault's own and chunks never travel; tags are the sending vault's vocabulary, which the
parent's own vocabulary reassigns. Declaring the note's author lets it survive the trip, since the drainer may be a
different person. Hashing the offered content into `offer.key` makes a retry of unchanged content idempotent and a
changed note a new offer.

#### 792b36c2a Scenarios

```gherkin
Scenario: 792b36c2a1 Placement and chunks are not sent
  Given a note learned with --target and --position and with chunk provenance
  When its offer is built
  Then the body holds no target, position, chunkSources or vault key

Scenario: 792b36c2a2 Supersedes is translated or dropped
  Given a note that supersedes A, which is linked to parent note PA, and B, which is unlinked
  When its offer is built
  Then supersedes names PA and does not name A or B

Scenario: 792b36c2a3 The note's own identity is declared
  Given a note with user and repo, and another with neither while detection finds both
  When each offer is built
  Then the first declares its own values and the second the detected ones

Scenario: 792b36c2a4 Origin, key and path of a direct offer
  Given a note with xid X in vault V and exchange hash H
  When its offer is built
  Then offer.origin is V:X, offer.key is the SHA-256 of V:X followed by H, and offer.path is [V]

Scenario: 792b36c2a5 A propagated offer extends its path
  Given an accepted served offer that travelled [O] and is now in vault V
  When its offer is built
  Then offer.path is [O, V] and offer.origin is this vault's, and when the note recorded no path it starts from its origin vault

Scenario: 792b36c2a6 A primary link makes an amend-offer
  Given a note whose primary link is under the configured parent's vault ID
  When its offer is built
  Then offer.for is the linked parent note

Scenario: 792b36c2a7 A covered or foreign link makes a learn-offer
  Given a note with only a covered link, or only a link under another parent vault
  When its offer is built
  Then it has no offer.for

Scenario: 792b36c2a8 A runbook sends its steps
  Given a runbook with a body, done_when, red flags, triggers and Supersedes lines
  When its offer is built
  Then the body, done_when, red flags and triggers are sent and the body holds no Supersedes line

Scenario: 792b36c2a9 Every offered field survives the wire
  Given any note, drawn at random, with no supersedes
  When its offer is built and rendered the way a served learn renders it
  Then the rendering has the note's exchange hash, and offer.key moves exactly when the hash does

Scenario: 792b36c2a10 An unbuildable note is not built
  Given a note whose basename has no slug, or whose frontmatter is missing or corrupt
  When its offer is built
  Then it fails and the parent is not contacted

Scenario: 792b36c2a11 Unreadable local state fails the build
  Given an unreadable vault ID, or an unreadable note in the supersedes scan
  When the payload builder first runs
  Then it fails

Scenario: 792b36c2a12 No user, no offer
  Given a note with no user: on a host whose user detection is empty
  When its offer is built
  Then no body is built, and the error says no user identity was detected
```

### 792b36c3 An offer withdrawn at send time

Requirement: [792b36c3 An offer withdrawn at send time](#792b36c3-an-offer-withdrawn-at-send-time) When the parent's vault ID is known at send time, the offer-outbox MUST NOT build an offer, and the
drain MUST NOT send it, when the note's `offer.origin` vault is the parent or the note's exchange hash equals the hash of
its primary link under the parent, and MUST drop such an entry unless the note changed since the snapshot was taken, in
which case it keeps the entry for the next drain.

Supports: [792a21n1h An accepted offer never returns to its origin vault](../product/parent-sync.md#792a21n1-uat-scenarios), [792a21r1j An offer withdrawn at send time is dropped](../product/parent-sync.md#792a21r1-uat-scenarios).

Rationale: the classification at write time may have run before the parent's vault ID was known, so the check is repeated
once the drain has learned it. An offer to its own origin would loop, and one equal to its link repeats what the parent
holds.

#### 792b36c3a Scenarios

```gherkin
Scenario: 792b36c3a1 An origin that is the parent, or an equal hash, is withdrawn
  Given an offer whose origin vault is the parent, and another whose content equals its primary link's hash
  When each is built with the parent's ID known
  Then neither is built, each reporting that it was withdrawn

Scenario: 792b36c3a2 A withdrawn entry is dropped unless its note changed
  Given a withdrawn offer, and another whose note changed while the offer was being built
  When the outcomes are merged
  Then the first entry is dropped and the second stays
```

### 792b36c4 Queueing an offer

Requirement: [792b36c4 Queueing an offer](#792b36c4-queueing-an-offer) With a parent configured, the offer-outbox MUST, under the vault lock and before the note is written,
classify the write ([792b36c1 Which writes are offered](#792b36c1-which-writes-are-offered)) and, when it is offered, stamp a freshly minted `xid` on a note that has none ([792b33c7 The lazy exchange ID](./exchange-versioning.md#792b33c7-the-lazy-exchange-id)),
and after the write MUST add the note's entry to `<vault>/.engram/outbox.json` in the same critical section: keyed by
`xid`, `queued`, with the current time, and leaving an existing entry as it is, with its first-queued time and state.
It MUST save the outbox atomically, after the vault ID is stamped and before `.engram/` exists. A failure to stage or to
queue MUST be warned and MUST NOT fail the write, which then proceeds unstaged and unqueued, and a note for which no
`xid` can be minted MUST be written and not queued. A corrupt outbox, a failed atomic write or a vault ID that cannot be
stamped MUST fail the enqueue. With no parent configured it MUST offer nothing, stamp nothing and write no exchange state.
When the cached parent vault ID is this vault's own, the vault MUST NOT act as its own parent ([792b5c12 Self-parent guard](./vault-identity-registry.md#792b5c12-self-parent-guard)):
it MUST stage and queue nothing, the write proceeds unstaged and unqueued, and one warning MUST name `engram vault-id --regenerate`.

Supports: [792a21m1b A write with no parent exchanges nothing](../product/parent-sync.md#792a21m1-uat-scenarios), [792a21q1a The parent is unreachable during learn](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21q1b Offline learn then amend coalesce](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21q1e A queued note is stamped with an exchange ID](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21q1g The first offer creates the exchange state in the right order](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21q1h A note is written even when no exchange ID can be made](../product/parent-sync.md#792a21q1-uat-scenarios), [792a2k1a State does not dirty the vault's git](../product/vault-identity.md#792a2k1-uat-scenarios), [792a2k1b The root .gitignore is untouched](../product/vault-identity.md#792a2k1-uat-scenarios), [792a2q1b A write in a vault that is its own parent queues nothing](../product/vault-identity.md#792a2q1-uat-scenarios).

Rationale: the write completes locally whether or not the parent is up, so the outbox lives next to the note and is
written in the write's own lock. Stamping the vault ID first keeps `.engram/` meaning created-by-engram.

#### 792b36c4a Scenarios

```gherkin
Scenario: 792b36c4a1 One entry per note, in its first slot
  Given a note already queued
  When the same note is queued again
  Then the outbox holds one entry for it, with its first-queued time

Scenario: 792b36c4a2 The outbox is saved atomically
  Given a note to queue
  When the enqueue runs
  Then the outbox is written to .engram/outbox.json only through the atomic write

Scenario: 792b36c4a3 The vault ID is stamped before the state directory
  Given a vault with no vault ID and no .engram
  When the first entry is queued
  Then the vault ID and its location record are stamped before .engram exists

Scenario: 792b36c4a4 Outbox failures fail the enqueue
  Given a corrupt outbox, a failed atomic write, or a vault ID that cannot be stamped
  When an entry is queued
  Then the enqueue fails

Scenario: 792b36c4a5 The xid is lazy and stable
  Given a note without an xid and another with one
  When each is queued
  Then the first gains an xid and the second keeps its own

Scenario: 792b36c4a6 A hook failure never fails the write
  Given a staging failure, and a queueing failure
  When a write runs with a parent configured
  Then each is warned and the write proceeds unstaged and unqueued

Scenario: 792b36c4a7 No parent, no exchange state
  Given no parent configured
  When a learn, amend or resituate runs
  Then nothing is classified, stamped, queued or written to .engram

Scenario: 792b36c4a8 Queueing leaves the vault's tracked files alone
  Given a git-backed vault with a root .gitignore
  When an offer is queued
  Then git status in the vault shows no change caused by the queue, and the root .gitignore is unchanged

Scenario: 792b36c4a9 A self-parent vault queues nothing
  Given the cached parent vault ID equals this vault's own
  When a learn, amend or resituate runs
  Then the note is written, no entry is queued, and one warning names engram vault-id --regenerate
```

### 792b36c5 Before a drain sends

Requirement: [792b36c5 Before a drain sends](#792b36c5-before-a-drain-sends) After a command has reached the parent (or in `update`), the offer-outbox MUST first record every
kept receipt ([792b36c15 An entry that waits](#792b36c15-an-entry-that-waits)), and then, unless the backoff gate refuses the contact ([792b5c14 The parent cache and the contact backoff](./vault-identity-registry.md#792b5c14-the-parent-cache-and-the-contact-backoff)), MUST do nothing when the
outbox is empty. When the outbox holds entries and the parent's vault ID is not cached, it MUST learn it with a one-item
`GET /query?dedupe-keys=1` before it builds any offer, and MUST send nothing and print one warning when that probe is
answered with a 4xx (recording the answer), reports no vault ID (the parent is too old, recording the answer), or cannot
reach the parent (recording a failure and printing the backoff warning with the queued count). It MUST send nothing when
the vault fails its location check (printing that check's warning), nor when the cached parent vault ID is the vault's own
(printing the self-parent warning), and a corrupt outbox, a corrupt parent cache or an unreadable note MUST fail the drain
before anything is sent.

Supports: [792a21r1h An empty outbox touches nothing](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21r1k A vault that fails its location check sends nothing](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21r1l A failed vault-ID probe keeps the offers queued](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21s1a Commands inside the backoff window make no request](../product/parent-sync.md#792a21s1-uat-scenarios), [792a21y1e A cached self-parent sends nothing](../product/parent-sync.md#792a21y1-uat-scenarios), [792a2p1a A cloned child is caught before it offers](../product/vault-identity.md#792a2p1-uat-scenarios).

Rationale: a clone or a copy must not offer under another vault's name ([792a2p Copied and cloned vaults pause exchange](../product/vault-identity.md#792a2p-copied-and-cloned-vaults-pause-exchange)), and an offer is built against the
parent's vault ID, so it is learned first, by the cheapest request the parent offers, or the offer would be built
wrongly. A failed location or backoff gate leaves the offers queued for the next command; a self-parent sends nothing and queues nothing new.

#### 792b36c5a Scenarios

```gherkin
Scenario: 792b36c5a1 An empty outbox touches nothing
  Given an empty outbox
  When a drain runs
  Then no location check, send or write happens

Scenario: 792b36c5a2 A failed location check sends nothing
  Given a cloned vault with no location record and a queued entry
  When a drain runs
  Then nothing is sent and the location warning is printed

Scenario: 792b36c5a3 A cached self-parent sends nothing
  Given the cached parent vault ID equals this vault's own
  When a drain runs
  Then nothing is sent, an entry queued earlier is neither sent nor rejected, and one warning names engram vault-id --regenerate

Scenario: 792b36c5a4 Unreadable local state fails the drain
  Given a corrupt outbox, a corrupt parent cache, or an unreadable note
  When a drain runs
  Then it fails and nothing is sent

Scenario: 792b36c5a5 A probe that fails keeps the offers queued
  Given an uncached parent vault ID and a probe answered with a 4xx, with no vault ID, or followed by an unreachable parent
  When a drain runs
  Then the offers stay queued and one warning is printed, and the unreachable case records a failure and prints the backoff warning

Scenario: 792b36c5a6 The probe caches the vault ID
  Given an uncached parent vault ID and a probe that answers with one
  When a drain runs
  Then the ID is cached before any offer is built

Scenario: 792b36c5a7 A backed-off parent is not drained
  Given a backoff window and a command other than update
  When a drain runs
  Then nothing is sent, and the kept receipts were still recorded
```

### 792b36c6 Which entries are sent

Requirement: [792b36c6 Which entries are sent](#792b36c6-which-entries-are-sent) The send phase MUST visit the entries oldest first (stable for ties, whatever the file order),
holding no vault lock while it sends, and MUST build each payload from the note found by `xid` at that moment, so a
rename between enqueue and drain is followed. It MUST drop, without sending, the entry of a note that is gone or now
pending, MUST skip a `rejected` entry whose note's exchange hash is not changed from the hash recorded as rejected, MUST
NOT send an `attention` entry that keeps a receipt, and MUST send an `attention` entry that keeps none as it sends a
queued one.

Supports: [792a21q1b Offline learn then amend coalesce](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21q1c A renamed note keeps its entry](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21q1d A discarded note's entry is dropped](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21q1f A pending note's entry is dropped](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21r1a Query drains a backlog](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21r1c A rejected offer does not block the queue](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21r1i Sends hold no lock](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21zb1b A queued CRLF note is sent](../product/parent-sync.md#792a21zb1-uat-scenarios).

Rationale: FIFO order is what a user expects of a queue, and sending without the lock keeps local writers moving while
the network answers. A note read by `xid` is the note as it is now, so an offer carries the latest content.

#### 792b36c6a Scenarios

```gherkin
Scenario: 792b36c6a1 Oldest first, whatever the file order
  Given three entries whose file order is not their queued order
  When the drain sends
  Then the offers go in queued order

Scenario: 792b36c6a2 Sends hold no lock
  Given a drain in progress
  When an offer is being sent
  Then the vault lock is not held, while the snapshot and the merge take it

Scenario: 792b36c6a3 A deleted or pending note is dropped unsent
  Given an entry whose note is gone and another whose note is pending
  When the drain sends
  Then neither is sent and both entries are dropped

Scenario: 792b36c6a4 The offer carries the latest content
  Given a note learned offline and then amended
  When the drain builds its offer
  Then the offer carries the amended content

Scenario: 792b36c6a5 A renamed note is followed by its xid
  Given an entry whose note was renamed after it was queued
  When the drain sends
  Then the offer is sent and its receipt lands on the new name

Scenario: 792b36c6a6 A CRLF note is found and sent
  Given a queued note with CRLF line endings
  When the drain sends
  Then the entry is not treated as gone and the offer is sent
```

### 792b36c7 The parent's answer

Requirement: [792b36c7 The parent's answer](#792b36c7-the-parents-answer) The offer-outbox MUST classify the parent's answer to a send as failed for a transport error, a
timeout, a 5xx, a status outside 200 to 299 that is not a 4xx, and a 2xx body that does not decode; as refused for a 409
whose body carries the loop discriminator (with the refusing vault's ID when it is well formed, else none); as rejected
for any other 4xx, with the parent's error as its cause; as too old for a 2xx receipt that lacks `vault_id` or `basename`;
as failed for a 2xx receipt whose `vault_id` is not an exchange identifier or whose `basename` or `for` is not an
exchange basename ([792b5c15 Exchange identifiers](./vault-identity-registry.md#792b5c15-exchange-identifiers)), however the rest of the receipt reads; and otherwise as accepted.

Supports: [792a21v1a A pre-change parent is detected](../product/parent-sync.md#792a21v1-uat-scenarios), [792a21v1b A malformed receipt is not recorded](../product/parent-sync.md#792a21v1-uat-scenarios).

Rationale: the parent is another machine's input, so what it returns is checked before it is written into frontmatter.
Only a 2xx with a whole, well formed receipt is a success, and a 5xx or a silent parent is an outage.

#### 792b36c7a Scenarios

```gherkin
Scenario: 792b36c7a1 Each answer maps to one outcome
  Given a transport error, a 5xx, a 4xx, a 409 with and without the loop mark, a receipt without vault_id or basename, and a full receipt
  When each answer is classified
  Then they are failed, failed, rejected, refused and rejected, too old, and accepted

Scenario: 792b36c7a2 Status classes always map the same way
  Given any status code
  When it is classified
  Then every 5xx is failed and every 4xx other than the loop 409 is rejected

Scenario: 792b36c7a3 A malformed identifier is never accepted
  Given a receipt whose vault_id is not 32 lowercase hexadecimal characters, or whose basename or for holds a separator or |, and a 409 with a malformed vault_id
  When each is classified
  Then the receipts are failed as undecodable and the 409 is a refusal with no vault ID

Scenario: 792b36c7a4 A forbidden basename character is never accepted
  Given a 2xx receipt whose basename holds /, \ or |, and any other field values
  When it is classified
  Then it is not accepted
```

### 792b36c8 An accepted offer

Requirement: [792b36c8 An accepted offer](#792b36c8-an-accepted-offer) For an accepted offer, the drain MUST, under the lock, cache the vault ID the receipt reports and reset
the failure count, record the receipt on the note with exchange-versioning's ([exchange-versioning](./exchange-versioning.md)) receipt edit ([792b33c9 Recording an offer receipt](./exchange-versioning.md#792b33c9-recording-an-offer-receipt)), and finish the
entry: removed when the note's exchange hash equals the hash that was sent, and queued again when it changed since. A
receipt for a note that has been deleted since the snapshot MUST be discarded and its entry dropped. A receipt that
fails to record for a reason other than the note refusing it ([792b36c15 An entry that waits](#792b36c15-an-entry-that-waits)) MUST keep the entry queued, with the attempt
counted and the error recorded, so it is sent again idempotently, and a malformed receipt MUST keep the entry queued
with the failure recorded.

Supports: [792a21q1b Offline learn then amend coalesce](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21r1e A note amended in flight stays queued](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21r1f A note deleted in flight drops its receipt](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21u1a Receipt links the note](../product/parent-sync.md#792a21u1-uat-scenarios), [792a21u1b Re-link to the resolved target](../product/parent-sync.md#792a21u1-uat-scenarios), [792a21u1c A receipt keeps the note's author](../product/parent-sync.md#792a21u1-uat-scenarios), [792a21u1d A receipt under another parent starts over](../product/parent-sync.md#792a21u1-uat-scenarios), [792a21v1b A malformed receipt is not recorded](../product/parent-sync.md#792a21v1-uat-scenarios), [792a21v1c A failed write keeps the entry](../product/parent-sync.md#792a21v1-uat-scenarios), [792a21zb1b A queued CRLF note is sent](../product/parent-sync.md#792a21zb1-uat-scenarios).

Rationale: the receipt is what links the child's note to its counterpart, and it is the parent's stored hash that is
recorded, not the child's ([792b33c9 Recording an offer receipt](./exchange-versioning.md#792b33c9-recording-an-offer-receipt)). A note amended while its offer was in flight carries content the parent has not
seen, so it keeps an entry.

#### 792b36c8a Scenarios

```gherkin
Scenario: 792b36c8a1 The receipt is recorded and the parent's ID cached
  Given an accepted offer
  When the outcome is merged
  Then the note carries the link, the parent's vault ID is cached, the failure count is zero, and the entry is removed

Scenario: 792b36c8a2 A note amended in flight stays queued
  Given a note amended while its offer was being sent
  When the receipt is recorded
  Then the entry is queued again for the newer content

Scenario: 792b36c8a3 A deleted note drops its receipt
  Given a note deleted while its offer was being sent
  When the outcome is merged
  Then the receipt is discarded and the entry is dropped

Scenario: 792b36c8a4 A receipt write failure keeps the entry
  Given recording the receipt on the note fails
  When the outcome is merged
  Then the drain reports the error and the entry stays queued with it

Scenario: 792b36c8a5 A malformed receipt keeps the entry
  Given a receipt with a malformed basename or vault ID
  When the drain handles it
  Then nothing is applied, and the entry stays queued with the failure recorded

Scenario: 792b36c8a6 A CRLF note takes its receipt as LF
  Given a queued CRLF note whose offer is accepted
  When the receipt is recorded
  Then the note is written as LF and its entry is removed
```

### 792b36c9 A rejected offer

Requirement: [792b36c9 A rejected offer](#792b36c9-a-rejected-offer) For a rejected offer (a 4xx, or a 409 without the loop mark), the drain MUST mark the entry `rejected`
with the exchange hash that was sent and the parent's error (empty when the parent gave none), count the attempt,
continue with the later entries, and reset the failure count and the backoff, because the parent answered. A rejected
entry MUST be sent again only when its note's exchange hash changes.

Supports: [792a21r1c A rejected offer does not block the queue](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21za1a Update reports a stuck outbox](../product/parent-sync.md#792a21za1-uat-scenarios).

Rationale: a 4xx says the offer, not the parent, is the problem, so retrying it unchanged would only repeat the refusal.
The next edit of the note, which moves its hash, re-arms it.

#### 792b36c9a Scenarios

```gherkin
Scenario: 792b36c9a1 A rejection re-arms only when the hash changes
  Given a 4xx answer to the first of two entries
  When the drain runs, and again after the note's hash changes
  Then the first entry is marked rejected with its hash and the second is still sent; it is not sent again until the hash changes

Scenario: 792b36c9a2 A rejection with no message is still a rejection
  Given a send that reports a rejection with no cause
  When the outcome is merged
  Then the entry is rejected with an empty error

Scenario: 792b36c9a3 An unmarked conflict is an ordinary rejection
  Given a 409 without the loop discriminator
  When the drain runs
  Then the entry is rejected and the drain continues

Scenario: 792b36c9a4 Any answer resets the backoff
  Given a rejection, a loop refusal and a too-old receipt in turn
  When each is merged
  Then each resets the failure count and the backoff window
```

### 792b36c10 An unreachable parent

Requirement: [792b36c10 An unreachable parent](#792b36c10-an-unreachable-parent) When a send fails ([792b36c7 The parent's answer](#792b36c7-the-parents-answer)), the drain MUST stop, MUST keep the failed entry and every later one in
their order, MUST record the attempt and the error on the failed entry, and MUST record a failure in the parent cache so
that the parent is backed off ([792b5c14 The parent cache and the contact backoff](./vault-identity-registry.md#792b5c14-the-parent-cache-and-the-contact-backoff)). The command MUST print the one backoff warning with the number of queued
offers, and the entries sent before the failure MUST finish as their outcomes say.

Supports: [792a21q1a The parent is unreachable during learn](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21r1b A transport failure stops the drain](../product/parent-sync.md#792a21r1-uat-scenarios), [792a21s1a Commands inside the backoff window make no request](../product/parent-sync.md#792a21s1-uat-scenarios).

Rationale: after one failure the rest would fail too, and each would wait out the connect timeout.

#### 792b36c10a Scenarios

```gherkin
Scenario: 792b36c10a1 A transport failure stops the drain
  Given three entries and a connection error on the second
  When the drain runs
  Then the first entry is removed, the second and third remain in order, the second records attempts and the error, and the parent is backed off for 30 seconds
```

### 792b36c11 A parent that is too old

Requirement: [792b36c11 A parent that is too old](#792b36c11-a-parent-that-is-too-old) When a receipt lacks a vault ID or a basename, the drain MUST stop with an error saying the parent
is too old, MUST keep the entry queued, MUST record the attempt, and MUST reset the failure count, because the parent
answered.

Supports: [792a21v1a A pre-change parent is detected](../product/parent-sync.md#792a21v1-uat-scenarios).

Rationale: an old parent cannot link the note, but it is up, so it is not backed off, and upgrading it needs no change here.

#### 792b36c11a Scenarios

```gherkin
Scenario: 792b36c11a1 A too-old parent keeps the entry and errors
  Given a receipt with no vault ID
  When the drain handles it
  Then it stops with a too-old error and the entry stays queued
```

### 792b36c12 A looping offer

Requirement: [792b36c12 A looping offer](#792b36c12-a-looping-offer) When the parent refuses an offer as a loop, the drain MUST, for a refusing vault other than this one,
mark the entry `rejected`, continue, print one warning per drain naming the refusing vault, and not back the
parent off, the refusal being an answer that resets the failure count ([792b36c9 A rejected offer](#792b36c9-a-rejected-offer)). When the refusing vault is this vault, or an accepted receipt reports this vault's own ID, it MUST send
and reject no entry, cache the parent's vault ID, record nothing on any note, stop the drain, and print
the one self-parent warning naming `engram vault-id --regenerate`.

Supports: [792a21y1a A cycle rejects the entry and continues](../product/parent-sync.md#792a21y1-uat-scenarios), [792a21y1c A self-parent refusal waits](../product/parent-sync.md#792a21y1-uat-scenarios), [792a21y1d A self-parent receipt is not recorded](../product/parent-sync.md#792a21y1-uat-scenarios).

Rationale: a real cycle (A to B to C to A) is a fault that retrying cannot mend, while a vault that is its own parent is a
copy mistake that the user repairs, and it must not act as its own parent, so nothing is sent or rejected ([792a2q A vault does not exchange with itself](../product/vault-identity.md#792a2q-a-vault-does-not-exchange-with-itself)).

#### 792b36c12a Scenarios

```gherkin
Scenario: 792b36c12a1 A cycle rejects the entry and continues
  Given a loop refusal from a vault that is not this one
  When the drain runs
  Then that entry is rejected, the drain continues, one warning names the refusing vault and not --regenerate, and the parent is not backed off

Scenario: 792b36c12a2 A self-parent refusal waits for a regenerate
  Given a loop refusal that names this vault's own ID
  When the drain runs
  Then no entry is sent or rejected, the parent's ID is cached, and the drain stops with one warning naming --regenerate

Scenario: 792b36c12a3 A self-parent receipt records nothing
  Given a receipt that reports this vault's own ID
  When the drain handles it
  Then nothing is recorded, the drain stops, the entry is neither sent nor rejected and one warning names --regenerate
```

### 792b36c13 An offer that cannot be built

Requirement: [792b36c13 An offer that cannot be built](#792b36c13-an-offer-that-cannot-be-built) When an offer cannot be built from its note, for a reason other than a withdrawal ([792b36c3 An offer withdrawn at send time](#792b36c3-an-offer-withdrawn-at-send-time)) or the lack
of a user identity ([792b36c15 An entry that waits](#792b36c15-an-entry-that-waits)), the drain MUST record the error on the entry, keep it queued, contact the parent not at
all, leave the parent cache alone and continue with the later entries.

Supports: [792a21r1g An unbuildable payload does not stop the drain](../product/parent-sync.md#792a21r1-uat-scenarios).

Rationale: a local fault in one note must not stop the others, and it says nothing about the parent.

#### 792b36c13a Scenarios

```gherkin
Scenario: 792b36c13a1 A skipped send keeps the entry and goes on
  Given an entry whose payload cannot be built
  When the drain runs
  Then the error is recorded, the entry stays queued, the later entries are sent, and the parent is not backed off

Scenario: 792b36c13a2 The production sender skips without contacting the parent
  Given a note that cannot be built
  When the production sender runs
  Then it makes no request, and for a buildable note it posts the built payload to /learn
```

### 792b36c14 Merging the outcomes

Requirement: [792b36c14 Merging the outcomes](#792b36c14-merging-the-outcomes) After the sends, the drain MUST take the vault lock, re-read the outbox, the parent cache and the notes
from disk, fold in every outcome, and save the outbox and the cache, so that an entry queued while the sends ran
survives. The outbox MUST hold at most one entry per note, and every note that exists, is not pending and whose current
content the parent has not accepted MUST keep an entry, across any sequence of writes, deletions, pending marks and drains.
A step for an entry that another drain already cleared MUST NOT revive it.

Supports: [792a21q1b Offline learn then amend coalesce](../product/parent-sync.md#792a21q1-uat-scenarios), [792a21r1d A concurrent enqueue survives the drain](../product/parent-sync.md#792a21r1-uat-scenarios).

Rationale: the sends ran without the lock, so the state they began from may be stale, and merging into a fresh read is
what keeps both a concurrent enqueue and a concurrent clear.

#### 792b36c14a Scenarios

```gherkin
Scenario: 792b36c14a1 A concurrent enqueue survives
  Given another process queues an entry while a drain is sending
  When the drain merges
  Then that entry is in the outbox

Scenario: 792b36c14a2 No offer is lost
  Given any sequence of writes, deletions, pending marks and drains, with accepted, rejected and failed sends and concurrent writes
  When the sequence ends
  Then the outbox holds at most one entry per note, and every live, non-pending note whose content the parent has not accepted keeps an entry
```

### 792b36c15 An entry that waits

Requirement: [792b36c15 An entry that waits](#792b36c15-an-entry-that-waits) When the note refuses an accepted receipt because its `parent:` carries a YAML anchor or its frontmatter
does not decode, the offer-outbox MUST leave the note untouched, move the entry to `attention` keeping the receipt, the
exchange hash that was sent and the reason, and print one warning naming the note and the reason, at that transition only.
When an offer cannot be built for want of a user identity, it MUST move the entry to `attention` with the reason `cannot
offer: no user identity detected; set git user.email` and warn once, MUST try to build the offer again on every drain
without contacting the parent and without a warning while it still cannot, and MUST return the entry to `queued` as
soon as the offer builds. Before the backoff gate, a drain MUST retry recording each kept receipt on its note, under the
vault lock and without contacting the parent: a receipt that records finishes the entry as an accepted offer does ([792b36c8 An accepted offer](#792b36c8-an-accepted-offer)),
a receipt that is still refused leaves the entry as it is and silent, a receipt whose vault ID is not the configured
parent's cached ID is discarded with one warning and the entry queued for the current parent, an entry whose note is gone
or pending is dropped, any other failure is recorded on the entry and reported, and a vault that cannot be listed fails
the pass.

Supports: [792a21w1a A refused receipt needs attention](../product/parent-sync.md#792a21w1-uat-scenarios), [792a21w1b An attention entry is not sent again](../product/parent-sync.md#792a21w1-uat-scenarios), [792a21w1c A kept receipt is recorded while the parent is backed off](../product/parent-sync.md#792a21w1-uat-scenarios), [792a21w1d A kept receipt from another parent is discarded](../product/parent-sync.md#792a21w1-uat-scenarios), [792a21w1e An attention entry resumes once the note is fixed](../product/parent-sync.md#792a21w1-uat-scenarios), [792a21w1f An attention entry without a receipt is sent](../product/parent-sync.md#792a21w1-uat-scenarios), [792a21w1g A retry failure is reported](../product/parent-sync.md#792a21w1-uat-scenarios), [792a21w1h An attention entry of a deleted note is dropped](../product/parent-sync.md#792a21w1-uat-scenarios), [792a21x1a No user identity holds the offer](../product/parent-sync.md#792a21x1-uat-scenarios), [792a21za1b Update reports an entry that needs attention](../product/parent-sync.md#792a21za1-uat-scenarios).

Rationale: the parent accepted the offer, so resending it would be wrong, and the note cannot take its link until its
owner repairs it. Keeping the receipt rather than the offer makes the repair automatic, and recording it before the
gate means a fixed note is linked even while the parent is away.

#### 792b36c15a Scenarios

```gherkin
Scenario: 792b36c15a1 A refused receipt needs attention
  Given a parent that accepts an offer whose note's parent: carries an anchor that another key aliases
  When the drain handles the receipt, then drains twice more, then the anchor is removed and it drains again
  Then the note is untouched and the entry is in attention with the receipt, the sent hash and the reason, and one warning names the note; later drains send nothing and stay silent; the next drain records the receipt without contacting the parent

Scenario: 792b36c15a2 A kept receipt is recorded inside the backoff window
  Given the anchor was removed while the parent is backed off
  When another command drains
  Then the receipt is recorded and the parent receives no request

Scenario: 792b36c15a3 A kept receipt from another parent is discarded
  Given an attention entry whose receipt names a vault other than the configured parent's
  When a command drains
  Then the note is not written, one warning says the receipt is discarded, and the entry is queued for the current parent

Scenario: 792b36c15a4 A fixed note that also changed is queued again
  Given the anchor was removed and the note's content changed since it was sent
  When the outbox drains
  Then the kept receipt is recorded without a send and the entry is queued again

Scenario: 792b36c15a5 An attention entry with no receipt is sent
  Given an attention entry that keeps no receipt
  When the outbox drains
  Then it is sent like a queued one

Scenario: 792b36c15a6 A retry failure is reported
  Given a kept receipt that cannot be recorded for a reason other than the frontmatter
  When the outbox drains
  Then the drain reports and records the error and the entry stays in attention without a send

Scenario: 792b36c15a7 A deleted note's kept receipt is dropped
  Given an attention entry whose note is gone
  When the kept receipts are recorded
  Then the entry is dropped

Scenario: 792b36c15a8 A vault that cannot be listed fails the pass
  Given a vault that cannot be listed
  When the kept receipts are recorded
  Then the pass fails

Scenario: 792b36c15a9 No user identity holds the offer, then releases it
  Given an entry whose note has no user: on a host where detection resolves empty
  When the outbox drains three times, and then detection starts to resolve
  Then nothing is sent, the entry is in attention with the no-identity reason and one warning was printed; the next drain builds the offer and returns the entry to queued

Scenario: 792b36c15a10 A built no-user entry that then fails is still counted
  Given an attention entry waiting for a user that now builds, and a network failure on its send
  When the outbox drains
  Then the entry is queued, counted, and not listed as needing attention
```

### 792b36c16 The update exchange and the outbox notice

Requirement: [792b36c16 The update exchange and the outbox notice](#792b36c16-the-update-exchange-and-the-outbox-notice) With a parent configured and outside `--dry-run`, the offer-outbox MUST drain in `engram update` ignoring
the backoff window. It MUST then return a notice that is empty when the outbox holds no entry and no backoff is active, and
otherwise gives a line with the queued count (not counting `rejected` or `attention` entries), the rejected count, the
count that need attention when it is not zero, and the age of the oldest entry; then each rejected entry and each entry
that needs attention with its note's current basename (its `xid` when the note is gone) and its error; then, while a
backoff is active, the retry time and the number of consecutive failures. An outbox that cannot be read MUST give a
notice that says it is unreadable.

Supports: [792a21s1b update always tries](../product/parent-sync.md#792a21s1-uat-scenarios), [792a21za1a Update reports a stuck outbox](../product/parent-sync.md#792a21za1-uat-scenarios), [792a21za1b Update reports an entry that needs attention](../product/parent-sync.md#792a21za1-uat-scenarios), [792a21za1c An idle vault has no notice](../product/parent-sync.md#792a21za1-uat-scenarios), [792a21za1d Update drains through a backoff](../product/parent-sync.md#792a21za1-uat-scenarios).

Rationale: a stuck outbox is silent otherwise, and `update` is where the user already looks for notices. A dry run must
change nothing, and the end of an outage must be noticed, so `update` drains through the window.

#### 792b36c16a Scenarios

```gherkin
Scenario: 792b36c16a1 A dry run only reports, a real run drains through the window
  Given queued entries and an active backoff window
  When update runs with --dry-run, and then without it
  Then the dry run drains nothing and the real run drains, after which the notice is empty

Scenario: 792b36c16a2 An idle outbox has no notice
  Given no entry and no backoff
  When the notice is built
  Then it is empty

Scenario: 792b36c16a3 Rejected and attention entries are listed
  Given a rejected entry and an attention entry
  When the notice is built
  Then each is listed with its note and error, and the count line separates them from the queued count

Scenario: 792b36c16a4 An attention entry is counted and listed through the production wiring
  Given a receipt the note refuses
  When update runs, and again after the anchor is removed
  Then the first notice counts and lists the entry and the parent is not offered it again, and the second records the receipt and goes idle
```
