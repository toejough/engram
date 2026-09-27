## ADDED Requirements

### Requirement: activate SHALL resolve refs locally first, then against the parent
`engram activate --note <ref>` SHALL treat a ref as a local hit when the note's `.md` file exists in the local vault, whether or not it has a sidecar. On a local hit it SHALL bump the note's sidecar `LastUsed`, as it does today.

On a local miss with `ENGRAM_PARENT` set, or for any ref when `--parent` is given, `activate` SHALL pull the ref down from the parent, but only when the ref is a basename or `<basename>.md`. A bare Luhmann ID SHALL NOT be resolved against the parent. `--parent` without `ENGRAM_PARENT` configured SHALL be an error, and no lookup SHALL be performed.

`activate` SHALL report each ref it could not activate on stderr, and SHALL exit non-zero if any ref failed.

#### Scenario: A parent-only ref pulls down
- **WHEN** `engram activate --note <basename>.md` runs with `ENGRAM_PARENT` set, and the note exists only in the parent
- **THEN** the parent note is pulled down as a local pending offer

#### Scenario: A local hit never contacts the parent
- **WHEN** `engram activate --note <ref>` runs and the ref's note file exists locally
- **THEN** the local sidecar's `LastUsed` is bumped, and no parent request is made

#### Scenario: Bare Luhmann IDs stay local
- **WHEN** `engram activate --note 1100` runs, and no local note has that ID
- **THEN** no parent request is made, and the ref is reported as failed

#### Scenario: Partial failure is visible
- **WHEN** `engram activate` is given two refs and one cannot be resolved locally or on the parent
- **THEN** stderr names the failed ref, and the command exits non-zero

#### Scenario: --parent without a parent configured
- **WHEN** `engram activate --parent --note x` runs with `ENGRAM_PARENT` unset
- **THEN** the command errors, and no note is read or written

### Requirement: A pulled-down note SHALL be an exact-content local pending offer
A pull-down SHALL fetch the parent note through the served `show` raw envelope (capability `vault-serve-api`), holding no vault lock while it does. It SHALL reject a response that is not a valid envelope; that means the parent is too old. Under the vault lock, it SHALL write a new local note with:
- a fresh local top-level Luhmann ID and its own `xid`;
- the parent's body unchanged, so its exchange hash equals the envelope's;
- the pending-offer marker;
- `parent.vault` set to the envelope's vault ID, and a primary link `{note: <envelope basename>, via: pulled, hash: <envelope exchange hash>}`;
- `parent.author` set to the parent note's `repo`/`user`/`vault`.

Its top-level `repo`/`user`/`vault` SHALL be stamped locally. It SHALL NOT carry the parent's own `parent`, `aliases`, `offer`, `xid`, `skill_hash`, `skill_key`, `skill_source`, `sources`, `supersedes`, or `tags`. It SHALL be embedded on write. A fetched note whose type is not `fact`, `feedback`, or `runbook` SHALL be refused, and nothing SHALL be written.

#### Scenario: Pulled note is pending and linked
- **WHEN** parent note P is pulled down
- **THEN** a new local note exists carrying `pending: true`, a `via: pulled` primary link to P with P's exchange hash, and `parent.author` equal to P's authorship; it is excluded from local query results, and the next query payload has `pending_offers: true`

#### Scenario: Remote identity and registration fields are stripped
- **WHEN** the fetched parent note carries `skill_hash`, `skill_key`, `aliases`, `xid`, or its own `parent` link
- **THEN** the local pending note carries none of those values, and its parent link is the one this pull-down set

#### Scenario: Chunk provenance does not travel
- **WHEN** the fetched parent note carries `sources` chunk IDs
- **THEN** the local note carries no `sources`

#### Scenario: A pre-change parent is detected
- **WHEN** the parent answers the raw request with a rendered note instead of an envelope
- **THEN** the pull-down fails with an error saying the parent is too old, and nothing is written

### Requirement: Pull-down SHALL be idempotent, remember declines, and be loop-free
Under the write lock, a pull-down SHALL re-check its skip rule and SHALL write nothing when either of these holds:
- a local note (live or pending) has any link (`offered`, `pulled`, or `covered`) to the envelope's basename or to one of its aliases, and that link's hash equals the envelope's exchange hash. When that local note is live, the pull-down SHALL bump its sidecar `LastUsed` instead;
- the vault's declined-pull record holds that basename with that exchange hash.

A bare `engram amend --discard` of a note whose primary link is `via: pulled` SHALL add that basename and hash to the declined-pull record. A parent note whose exchange hash changed since the last pull, or since it was declined, SHALL arrive as another pending offer. A pulled-down note SHALL never be offered back to the parent unless its content is later changed locally (capability `vault-parent-offers`).

#### Scenario: Re-activating an unchanged parent note
- **WHEN** parent note P is activated twice with no change to P in between
- **THEN** the local vault gains exactly one note for P

#### Scenario: A covered parent note is not pulled again
- **WHEN** local note L already has a primary link and a pulled copy of P is folded into L with `--discard --into L`
- **THEN** L gains a `covered` link to P, and activating P again writes nothing and bumps L's `LastUsed`

#### Scenario: A declined pull is remembered
- **WHEN** a pulled copy of P is discarded outright, and P is activated again unchanged
- **THEN** nothing is written

#### Scenario: A changed parent note arrives as a new offer
- **WHEN** P is pulled down and accepted, then P changes on the parent, and P is activated again
- **THEN** a second local pending offer for P exists, carrying the new exchange hash

#### Scenario: Accepting a pulled note sends nothing up
- **WHEN** curation clears the pending marker on a pulled note
- **THEN** no offer is queued for it

### Requirement: Pull-down SHALL also signal use to the parent
After a pull-down or a skip, once the vault lock is released, `activate` SHALL send a best-effort `activate` request for the parent note to the parent. A failure of that request SHALL NOT fail the command, and SHALL NOT be queued.

#### Scenario: Parent recency is bumped
- **WHEN** parent note P is activated from a child and the parent is reachable
- **THEN** P's sidecar `LastUsed` on the parent is updated
