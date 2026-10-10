# Exchange Versioning - Technical Spec

## 792b33a Orientation

- Ownership: Repository component.
- Responsibility: gives a note the identity that two vaults compare and the links that tie it to a note in another
  vault: a versioned hash over exactly the content that a note offers, a three-way comparison of two such hashes, the
  optional exchange fields of a note (its own exchange ID, its links to parent notes, the names it answers to, and
  the record of the offer it came from), and the edits that record an offer receipt on a note's links without
  disturbing anything else in its frontmatter.
- Current realization (informational): `internal/cli/exchangehash.go`, `exchangefields.go`, `exchange_blocks.go`
  and `offer_receipt.go` (`exchangeHash`, `compareExchangeHashes`, `exchangeKeyClassification`,
  `exchangeFrontmatter`, `editExchangeBlocks`, `applyReceiptToContent`, `setXIDField`).
- Qualification: responsibility, the hash, its comparison and the link edit are stated without naming a function;
  ownership, it owns what is hashed and how two hashes compare, while [note-document-format](./note-document-format.md) owns the frontmatter
  edit machinery it reuses and the exchange paths that call it (served learn, the outbox, pull-down, curation) own
  when to hash and edit; independent verification, a golden note fixes the hash and a property test moves each
  field in turn; stability, the contract survives a new frontmatter key, because a reflection test refuses an
  unclassified one; necessity, two vaults cannot tell whether a note changed, nor link it to its counterpart,
  without it.
- Supported capabilities: [vault-serving](../product/vault-serving.md), [parent-sync](../product/parent-sync.md). Curate ([curate](../product/curate.md)) extends it later.

Rationale: a vault that exchanges notes needs a hash that changes when and only when an offered note's content
changes. The sidecar's content hash (`embed.ContentHash`) covers other fields and exists to detect a stale embedding,
so exchange has its own. Hashing the whole file would make identity and link fields move the hash, and loop
suppression would never fire. The hash is versioned (`xh1:`) so a later change to what is hashed makes old and new
hashes compare as unknown instead of different. Both sides of an exchange call the same function, so the same file
hashes the same on a child and on its parent.

## 792b33c Behavior Requirements

### 792b33c1 The exchange hash

Requirement: [792b33c1 The exchange hash](#792b33c1-the-exchange-hash) The exchange-versioning MUST give a note the hash `xh1:` followed by the lowercase hex SHA-256
of the sorted-key JSON object of `type`, `situation`, `subject`, `predicate`, `object`, `behavior`, `impact`,
`action`, `done_when`, `red_flags`, `triggers` and `body`, in which an absent text field is the empty string, an
absent list is the empty list and `body` is the note's canonical body ([792b33c3 The canonical body](#792b33c3-the-canonical-body)).

Supports: [792a20d1a The receipt names the pending note](../product/vault-serving.md#792a20d1-uat-scenarios), [792a20k1a Show without raw matches the local output](../product/vault-serving.md#792a20k1-uat-scenarios), [792a20k1b Raw show is exact](../product/vault-serving.md#792a20k1-uat-scenarios), [792a20m1c Keys are present only on request](../product/vault-serving.md#792a20m1-uat-scenarios).

Rationale: the one function is used by the server for a receipt's `stored_hash`, for the raw envelope, for the
exchange-hash header of `engram show` and for the query payload's dedupe keys, and by the child for what it
offers, so a golden test pins what both sides compute.

#### 792b33c1a Scenarios

```gherkin
Scenario: 792b33c1a1 A feedback note hashes to the canonical JSON
  Given a feedback note with situation, behavior, impact and action, identity fields, tags and a supersedes line
  When its exchange hash is computed
  Then it is xh1: followed by the SHA-256 of the sorted-key JSON of the offered fields, with absent fields empty, and the body Lesson learned line followed by one newline
```

### 792b33c2 What the hash covers

Requirement: [792b33c2 What the hash covers](#792b33c2-what-the-hash-covers) The exchange hash MUST change when any offered field or the body of a note changes, on a fact,
feedback or runbook note, and MUST NOT change when any other frontmatter key changes: `tier`, `luhmann`, `created`,
`source`, `project`, `repo`, `user`, `vault`, `pending`, `issue`, `sources`, `vocab_version`, `tags`, `supersedes`,
`skill_hash`, `skill_key`, `skill_source`, `xid`, `parent`, `aliases` and `offer`. Every key of the fact, feedback and
runbook frontmatter MUST be classified as offered or not offered, and a key on no frontmatter MUST NOT be classified.

Supports: none. The hash is consumed by [parent-sync](../product/parent-sync.md) and [curate](../product/curate.md), whose scenarios map it.

Rationale: placement, provenance, identity, bookkeeping and the exchange's own fields differ between a note and its
counterpart in another vault without the content having changed, so hashing them would report a change that is none.
The classification is one explicit table, with a reflection test that fails when a frontmatter key is missing from
it, so a new field cannot silently fall outside the hash.

#### 792b33c2a Scenarios

```gherkin
Scenario: 792b33c2a1 Changing an offered field moves the hash
  Given a note of any type
  When one offered field or the body is changed
  Then the exchange hash changes

Scenario: 792b33c2a2 Every offered field of each type moves the hash
  Given a feedback note, and a runbook note with done_when, red_flags and triggers
  When the feedback note's impact, or the runbook's done_when, red_flags or triggers, is changed
  Then each change moves the hash

Scenario: 792b33c2a3 Changing any other key leaves the hash alone
  Given a note of any type
  When one non-offered frontmatter key is changed
  Then the exchange hash is unchanged

Scenario: 792b33c2a4 Every frontmatter key is classified
  Given the fact, feedback and runbook frontmatter structures
  When each of their YAML keys is looked up in the classification
  Then every key is classified, and the classification names no key that no structure has

Scenario: 792b33c2a5 The classification is the design's split
  Given the classification
  When its offered keys are listed
  Then they are exactly the hashed content fields, and repo, user, vault, pending, tags, sources, supersedes, xid, parent, aliases, offer and the skill fields are not offered
```

### 792b33c3 The canonical body

Requirement: [792b33c3 The canonical body](#792b33c3-the-canonical-body) The exchange-versioning MUST hash a note as its LF form, with the body being the text after the
frontmatter with CRLF line endings turned into LF and ending in exactly one newline (none for an empty body), so that
the LF, CRLF and missing-final-newline spellings of one body, and a CRLF blank separator line, hash alike. A note
whose leading `---` block never closes MUST have no frontmatter, with its whole text as the body and every offered
frontmatter field empty, and MUST NOT be an error.

Supports: none. See [792b33c2 What the hash covers](#792b33c2-what-the-hash-covers).

Rationale: one body in two spellings must not look like two edits. A note with LF frontmatter and a CRLF body hashes
as its LF conversion, and a note converted from CRLF frontmatter matches the same note authored in LF. An unterminated
block reads as body everywhere else in engram, and an error would let one malformed note stall an outbox drain or a
dedupe pass. The prefix stays `xh1:` because a bump would turn every note's comparison into unknown across versions.

#### 792b33c3a Scenarios

```gherkin
Scenario: 792b33c3a1 Line endings do not move the hash
  Given a note body in LF, in CRLF with a final newline and in CRLF without one, with either separator line
  When each is hashed
  Then all hashes are equal

Scenario: 792b33c3a2 A CRLF frontmatter hashes as its LF form
  Given a note whose frontmatter is CRLF and the same note in LF
  When both are hashed
  Then the hashes are equal

Scenario: 792b33c3a3 Converting a CRLF body leaves the hash alone
  Given a note with LF frontmatter and a CRLF body, with an LF or a CRLF separator line
  When it is converted to LF
  Then its hash is unchanged

Scenario: 792b33c3a4 A converted note matches the one authored in LF
  Given a note converted from CRLF frontmatter and the same note authored in LF
  When both are hashed
  Then the hashes are equal

Scenario: 792b33c3a5 An unterminated frontmatter is the whole text as body
  Given a note whose leading --- block never closes
  When it is hashed
  Then there is no error, the body is the whole text and every offered field is empty
```

### 792b33c4 An unreadable frontmatter has no hash

Requirement: [792b33c4 An unreadable frontmatter has no hash](#792b33c4-an-unreadable-frontmatter-has-no-hash) The exchange-versioning MUST fail with an error wrapping `exchange hash: parsing frontmatter`
when a note's closed frontmatter is not YAML, and MUST NOT give such a note a hash.

Supports: none. See [792b33c2 What the hash covers](#792b33c2-what-the-hash-covers).

#### 792b33c4a Scenarios

```gherkin
Scenario: 792b33c4a1 Invalid YAML has no hash
  Given a note whose frontmatter is "situation: [unclosed"
  When its exchange hash is computed
  Then it fails
```

### 792b33c5 Comparing two hashes

Requirement: [792b33c5 Comparing two hashes](#792b33c5-comparing-two-hashes) The exchange-versioning MUST compare two hashes three ways: unknown when either lacks the
current `xh1:` prefix, whatever the other is, otherwise equal when the strings are equal and changed when they
differ, and the comparison MUST be symmetric. A check for equality, which loop suppression and deduplication use,
MUST be true only for equal. A check for change, which the pull-down skip, the decline match, the re-arm of a rejected
offer and the send-and-apply check use, MUST be true only for changed, so unknown counts as not changed.

Supports: none. See [792b33c2 What the hash covers](#792b33c2-what-the-hash-covers).

Rationale: a hash from an older or newer version, a sidecar's content hash, a missing hash and an empty one say
nothing about whether the content changed, so a consumer that acts only on equal never acts on a guess, and a
consumer that acts only on changed never re-sends or re-pulls on one.

#### 792b33c5a Scenarios

```gherkin
Scenario: 792b33c5a1 Hashes compare equal, changed or unknown
  Given two current hashes, an older-version hash, a newer-version hash, a hash with no prefix, a sidecar content hash and an empty string
  When each pair is compared in both orders
  Then equal and changed are reported only when both carry the current prefix, and every other pair is unknown

Scenario: 792b33c5a2 Unknown is not a match and not a change
  Given a recorded hash of an older version and a current hash, and two empty hashes
  When the equality check and the change check run
  Then the equality check is false and the change check is false

Scenario: 792b33c5a3 Equal matches and changed changes
  Given two equal current hashes and two different current hashes
  When the equality check and the change check run
  Then the equality check is true only for the equal pair and the change check is true only for the different pair
```

### 792b33c6 The exchange fields

Requirement: [792b33c6 The exchange fields](#792b33c6-the-exchange-fields) The exchange-versioning MUST give the fact, feedback and runbook frontmatter the optional keys
`xid` (the note's own exchange ID), `parent` (`vault`, `links` of `note`, `via` and `hash`, and `author` of `repo`,
`user` and `vault`), `aliases` (the basenames the note answers to in its own vault) and `offer` (`origin`, `key`,
`prior_keys`, `for` and `path`), each omitted when empty, so that a note that never takes part in exchange renders
exactly as it did before exchange existed. JSON decoding into a frontmatter MUST NOT populate any of them, under any
spelling of its key. It MUST read the exchange fields of a note with CRLF frontmatter from its LF form. No identity backfill MUST stamp an `xid` or any other exchange field on a note.

Supports: [792a20j1a Remote-set link and registration fields are ignored](../product/vault-serving.md#792a20j1-uat-scenarios), [792a20j1b A caller cannot forge a link](../product/vault-serving.md#792a20j1-uat-scenarios).

Rationale: `xid` is stamped the first time a note takes part in exchange (queued as an offer, pulled, or received as
a served offer) and never changed, so a note that never exchanges carries none. `parent` holds at most one primary
link (`offered` or `pulled`) and any number of `covered` links, because one link per note could not record that the
note covers a second parent note once it has a counterpart. `aliases` lets a parent's or a child's
reference to a renamed note keep resolving. The keys are not settable from the wire, which is what stops a remote
client forging a link ([792a20j A caller cannot set link, identity or registration fields](../product/vault-serving.md#792a20j-a-caller-cannot-set-link-identity-or-registration-fields)).

#### 792b33c6a Scenarios

```gherkin
Scenario: 792b33c6a1 Every frontmatter structure declares the exchange fields
  Given the fact, feedback and runbook frontmatter structures
  When their YAML keys are listed
  Then each holds xid, parent, aliases and offer

Scenario: 792b33c6a2 A note without exchange fields renders as before
  Given a fact, feedback and runbook note with every pre-exchange field set and no exchange field
  When its frontmatter is rendered
  Then it is byte-identical to the pre-exchange rendering

Scenario: 792b33c6a3 JSON never populates an exchange field
  Given a JSON document carrying xid, parent, aliases and offer under several spellings
  When it is decoded into each frontmatter structure and the structure is rendered as YAML
  Then the rendering holds none of the four keys

Scenario: 792b33c6a4 Identity backfill adds no exchange field
  Given a vault of fact, feedback and runbook notes that never took part in exchange
  When the identity backfill stamps the notes it can
  Then no note gains xid, parent, aliases or offer

Scenario: 792b33c6a5 A CRLF frontmatter reads as LF
  Given a note with CRLF frontmatter carrying an xid
  When its exchange fields are decoded
  Then they equal those decoded from its LF form, and the xid is not empty
```

### 792b33c7 The lazy exchange ID

Requirement: [792b33c7 The lazy exchange ID](#792b33c7-the-lazy-exchange-id) The exchange-versioning MUST stamp `xid` into a note's frontmatter, before `parent`, `aliases`
and `offer` where they are present, only when the note has none, MUST return a note that already has an `xid`
unchanged, and MUST fail for a note with no frontmatter.

Supports: [792a20j1b A caller cannot forge a link](../product/vault-serving.md#792a20j1-uat-scenarios).

#### 792b33c7a Scenarios

```gherkin
Scenario: 792b33c7a1 The stamp goes before the exchange blocks
  Given a note with aliases and no xid
  When an xid is stamped
  Then the xid line precedes the aliases block and every other line is unchanged

Scenario: 792b33c7a2 An existing xid is kept
  Given a note that already has an xid
  When another xid is stamped
  Then the note is unchanged

Scenario: 792b33c7a3 A note with no frontmatter cannot be stamped
  Given a note with no frontmatter
  When an xid is stamped
  Then it fails
```

### 792b33c8 Editing the exchange blocks

Requirement: [792b33c8 Editing the exchange blocks](#792b33c8-editing-the-exchange-blocks) The exchange-versioning MUST edit a note's `aliases:` list and `parent:` block as YAML nodes: the
`aliases:` list replaced whole, `parent:` edited in place with its `vault` and `author` set and each link's node
reused when the edit keeps a link to the same note, so that unknown keys under `parent:` and inside its links, and every
other key, survive. It MUST refuse with `errFrontmatterAnchoredKey` and write nothing when a key it edits carries a
YAML anchor, MUST decode the result again before returning it and refuse with `errFrontmatterUndecodable` when it does
not decode, MUST return a note it leaves unchanged byte for byte, and MUST place a new block where the frontmatter
writer does (`parent`, `aliases`, `offer`). A note without unknown keys or anchors MUST be written with the same
bytes, and the same exchange hash, as the line-based rewrite it replaced.

Supports: none. The edit is used by curate's ([curate](../product/curate.md)) fold and by the offer receipt.

Rationale: an anchored value cannot be replaced without leaving its aliases dangling, and a rewrite that does not
decode would corrupt a note an agent depends on, so the edit refuses before it writes. The fold refuses before it
writes the existing note or deletes the offer, so a refusal leaves both files untouched.

#### 792b33c8a Scenarios

```gherkin
Scenario: 792b33c8a1 Unknown keys under parent survive
  Given a note whose parent: and one of its links carry keys the exchange model does not define
  When a receipt or a fold rewrites its parent links
  Then the written note still carries both keys with their values, alongside the new links

Scenario: 792b33c8a2 An anchored parent is refused
  Given a note whose parent: carries an anchor that another key aliases
  When a receipt would rewrite it
  Then the edit is refused with the anchored-key error and no content is returned

Scenario: 792b33c8a3 An anchored edited key is refused by a fold
  Given a note whose parent: or aliases: carries an anchor that another key aliases, and an offer to fold into it
  When the fold would change that key
  Then it fails naming the key, the note is byte-identical and the offer is not deleted

Scenario: 792b33c8a4 A node edit either refuses or keeps every unknown key
  Given drawn exchange notes with optional unknown keys under parent: and an optional aliased anchor
  When a receipt or a fold edits them
  Then each either refuses the anchored edit with every file untouched, or writes frontmatter that decodes and keeps every unknown key and the anchored value

Scenario: 792b33c8a5 A fold keeps unknown keys under parent
  Given an existing note whose parent: and a link carry unknown keys, and an offer with parent links
  When the offer is folded into it
  Then the existing note still carries both unknown keys with its links merged

Scenario: 792b33c8a6 A note without unknown keys is written as before
  Given notes without unknown keys or anchors
  When a fold or a receipt edits them
  Then the written bytes and the exchange hash equal those the line-based rewrite wrote

Scenario: 792b33c8a7 Held link nodes are reused only when they line up
  Given a parent: block whose links line up one to one with the decoded links as mappings, and blocks that do not
  When the held link nodes are taken
  Then they are returned only for the aligned block
```

### 792b33c9 Recording an offer receipt

Requirement: [792b33c9 Recording an offer receipt](#792b33c9-recording-an-offer-receipt) The exchange-versioning MUST record an offer receipt on a note by setting `parent.vault` to the
receipt's vault ID and making the primary link `{note, via: offered, hash: stored_hash}`, where `note` is the receipt's
resolved target when it names one and otherwise its basename. It MUST replace the previous primary link (`offered` or
`pulled`) and any link to the same note, MUST keep `covered` links under the same vault, and MUST start over with only
the new link when the note's links belong to another vault. It MUST edit only `parent:`, so that the exchange hash, the
identity fields, `pending`, `xid`, `aliases` and `offer` are untouched and nothing re-embeds, MUST read a CRLF note as LF
and write it as LF in that write, MUST be idempotent, and MUST fail, writing nothing, for a note with no frontmatter or
whose `parent:` does not decode.

Supports: none. The receipt is parent-sync's.

Rationale: the receipt is the parent's answer to an offer, and linking the child's note to the note it became (or to the
live note it was folded onto) is what lets a later query dedupe the pair and a retry find it.

#### 792b33c9a Scenarios

```gherkin
Scenario: 792b33c9a1 The receipt links the note
  Given a note with no parent block and a receipt for 1100.2026-09-27.x
  When the receipt is recorded
  Then parent.vault is the receipt's vault, the one link is {1100.2026-09-27.x, offered, the stored hash}, and nothing outside parent: changed

Scenario: 792b33c9a2 Another vault's links start over
  Given a note whose links are under a different parent vault
  When a receipt is recorded
  Then parent.vault is the receipt's vault and the note holds only the new link

Scenario: 792b33c9a3 A receipt that names a target re-links to it
  Given a note linked to a note offered earlier and a receipt whose for names a live note
  When the receipt is recorded
  Then the primary link is to the live note

Scenario: 792b33c9a4 The primary is replaced and covered links are kept
  Given a note with a pulled primary link and a covered link under the same vault
  When a receipt is recorded
  Then the primary is the new offered link and the covered link remains

Scenario: 792b33c9a5 A receipt touches only the parent block
  Given any note and any receipt
  When the receipt is recorded, and recorded again
  Then the exchange hash, identity, pending, xid, aliases and offer are unchanged, there is one primary link, and the second recording changes nothing

Scenario: 792b33c9a6 A note with no frontmatter cannot take a receipt
  Given a note with no frontmatter
  When a receipt is recorded
  Then it fails

Scenario: 792b33c9a7 A CRLF note takes the receipt as LF
  Given a CRLF note queued for an offer
  When its receipt is recorded after the offer is sent
  Then the note is written with no CRLF and the outbox entry is removed
```

### 792b33c10 The exchange fields survive every other rewrite

Requirement: [792b33c10 The exchange fields survive every other rewrite](#792b33c10-the-exchange-fields-survive-every-other-rewrite) Every path that rewrites a note's frontmatter and is not an exchange path MUST leave `xid`,
`parent`, `aliases` and `offer` unchanged, byte for byte: `amend` with every flag family on every note type, `resituate`,
identity backfill, Luhmann reparenting (both the rename of the note itself and the rewrite of the references inside
another note), the vocabulary rewrites (term assignment, removal of a term, legacy cleanup, the self-tag of a definition
note and the version stamp), the scrubbing of references to deleted notes, and skill registration refresh and adopt. The
exchange paths are the offer receipt, the pull-down, the curation fold and the rename's append to `aliases`.

Supports: [792a21zi1b The xid survives a rename](../product/parent-sync.md#792a21zi1-uat-scenarios), [792a21zl1a Amend preserves the exchange fields](../product/parent-sync.md#792a21zl1-uat-scenarios), [792a21zl1b Identity backfill preserves the link](../product/parent-sync.md#792a21zl1-uat-scenarios), [792a21zl1g Every other rewrite path preserves the fields](../product/parent-sync.md#792a21zl1-uat-scenarios).

Rationale: a link or an exchange ID lost to an unrelated rewrite would silently disconnect a note from its counterpart,
and the next exchange would treat the note as new. Each rewrite site has its own survival test because each rewrites
the frontmatter in its own way, some by re-marshalling the typed document and some line by line.

#### 792b33c10a Scenarios

```gherkin
Scenario: 792b33c10a1 Amend keeps the exchange fields
  Given notes of every type carrying xid, parent, aliases and offer, including randomly drawn values
  When engram amend runs with each flag family
  Then all four fields are byte-for-byte unchanged

Scenario: 792b33c10a2 Resituate keeps the exchange fields
  Given a note carrying all four fields, including randomly drawn values
  When engram resituate replaces its situation
  Then all four fields are byte-for-byte unchanged

Scenario: 792b33c10a3 Identity backfill keeps the exchange fields
  Given a note carrying all four fields and missing user: or vault:
  When engram update --backfill-identity rewrites it
  Then all four fields are unchanged

Scenario: 792b33c10a4 Reparenting keeps the exchange fields
  Given a note carrying all four fields
  When Luhmann reparenting renames it, and when it only rewrites the references inside another note
  Then xid, parent and offer are unchanged and the renamed note's aliases only gained its old basename

Scenario: 792b33c10a5 Vocabulary rewrites keep the exchange fields
  Given a note carrying all four fields
  When a term is assigned, removed, cleaned up as legacy, self-tagged or version-stamped
  Then all four fields are unchanged

Scenario: 792b33c10a6 Reference scrubs keep the exchange fields
  Given a note carrying all four fields and references to a deleted note
  When the references are scrubbed, for one note or for the whole vault
  Then all four fields are unchanged

Scenario: 792b33c10a7 Skill registration keeps the exchange fields
  Given a skill note carrying all four fields
  When registration refreshes it, or adopts it with or without a rename
  Then all four fields are unchanged
```
