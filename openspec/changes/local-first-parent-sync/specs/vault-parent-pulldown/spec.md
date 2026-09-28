## ADDED Requirements

### Requirement: activate SHALL resolve refs locally first, then against the parent
`engram activate --note <ref>` SHALL treat a ref as a local hit when the note's `.md` file exists in the local vault, whether or not it has a sidecar. On a local hit it SHALL bump the note's sidecar `LastUsed`, as it does today.

On a local miss with `ENGRAM_PARENT` set, or for any ref when `--parent` is given, `activate` SHALL pull the ref down from the parent, but only when the ref is a basename or `<basename>.md`. A bare Luhmann ID SHALL NOT be resolved against the parent. `--parent` without `ENGRAM_PARENT` configured SHALL be an error, and no lookup SHALL be performed.

`activate` SHALL report each ref it could not activate on stderr, and SHALL exit non-zero if any ref failed.

#### Scenario: A parent-only ref pulls down
- **WHEN** `engram activate --note <basename>.md` runs with `ENGRAM_PARENT` set, and the note exists only in the parent
- **THEN** the parent note is pulled down as a local pending offer

#### Scenario: A local hit never contacts the parent
- **WHEN** `engram activate --note <ref>` runs and the ref's note file exists locally, and that note has no parent link
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
A pull-down SHALL fetch the parent note through the served `show` raw envelope (capability `vault-serve-api`), holding no vault lock while it does. It SHALL reject a response that is not a valid envelope; that means the parent is too old. It SHALL also reject, writing nothing, an envelope whose `vault_id` is not 32 lowercase hex characters or whose `basename` is not a Luhmann basename free of `/`, `\` and `|` (a malformed reply). Under the vault lock, it SHALL write a new local note with:
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
- the vault's declined-pull record holds that exchange hash, under the envelope's vault ID, for the envelope's basename or any alias in the fetched content (so a parent-side rename does not bring back a declined note, and a decline under one parent never suppresses a note of another).

A bare `engram amend --discard` of a note whose primary link is `via: pulled` SHALL add that basename and hash, keyed by the link's parent vault ID, to the declined-pull record — except in a vault with no parent configured and no vault ID, where recording would stamp a vault ID (capability `vault-local-first`), so nothing is recorded. A parent note whose exchange hash changed since the last pull, or since it was declined, SHALL arrive as another pending offer. A pulled-down note SHALL never be offered back to the parent unless its content is later changed locally (capability `vault-parent-offers`).

#### Scenario: Re-activating an unchanged parent note
- **WHEN** parent note P is activated twice with no change to P in between
- **THEN** the local vault gains exactly one note for P

#### Scenario: A covered parent note is not pulled again
- **WHEN** local note L already has a primary link and a pulled copy of P is folded into L with `--discard --into L`
- **THEN** L gains a `covered` link to P, and activating P again writes nothing and bumps L's `LastUsed`

#### Scenario: A declined pull is remembered
- **WHEN** a pulled copy of P is discarded outright, and P is activated again unchanged
- **THEN** nothing is written

#### Scenario: A decline survives a parent rename
- **WHEN** a declined parent note is renamed on the parent (its old basename now in its `aliases`) and activated again unchanged
- **THEN** nothing is written

#### Scenario: A changed parent note arrives as a new offer
- **WHEN** P is pulled down and accepted, then P changes on the parent, and P is activated again
- **THEN** a second local pending offer for P exists, carrying the new exchange hash

#### Scenario: Accepting a pulled note sends nothing up
- **WHEN** curation clears the pending marker on a pulled note
- **THEN** no offer is queued for it

### Requirement: Activating a linked local note SHALL re-check its parent note
The merged query shows a linked local note in place of its parent note (capability `vault-merged-recall`), so a parent-side change reaches the child through the local note's use. When `engram activate` resolves a ref to a local note that has parent links under the configured parent's vault ID, and `ENGRAM_PARENT` is set, it SHALL, after the local bump and with no vault lock held, fetch each linked parent note's raw envelope through the same gated, backoff-aware parent contact a pull-down uses. It SHALL then apply the pull-down write step under the lock: when the parent note's current exchange hash equals the link's hash, or is declined, nothing SHALL be written; otherwise the parent note SHALL arrive as a new local pending offer, exactly as a pull-down writes one. A link under another parent vault SHALL NOT be written: when that parent's vault ID is already cached, the link SHALL NOT be contacted at all; otherwise the envelope SHALL still be fetched and the reply SHALL be discarded once its vault ID differs from the link's. The re-check SHALL NOT send the best-effort `activate` request to the parent. Its failure — the parent unreachable, backed off, or the note gone there — SHALL NOT fail the local activate.

#### Scenario: A changed parent note arrives through its local copy
- **WHEN** P is pulled down and accepted as local note L, then P changes on the parent, the merged query still returns L in P's place, and L is activated
- **THEN** a new local pending offer for P exists, carrying P's new exchange hash, and activating L again writes nothing more

#### Scenario: An unchanged parent note writes nothing
- **WHEN** a linked local note L is activated and its parent note's exchange hash equals L's link hash
- **THEN** nothing is written, and only L's `LastUsed` changes

#### Scenario: A declined update is not pulled again
- **WHEN** the pending offer a re-check pulled down is discarded outright, and L is activated again with the parent note unchanged since
- **THEN** nothing is written

#### Scenario: An unreachable parent does not fail the local activate
- **WHEN** a linked local note is activated while the parent is unreachable
- **THEN** the command succeeds, the local `LastUsed` is bumped, and the failure is recorded for backoff

### Requirement: Pull-down SHALL also signal use to the parent
After a pull-down or a skip, once the vault lock is released, `activate` SHALL send a best-effort `activate` request for the parent note to the parent. A failure of that request SHALL NOT fail the command, and SHALL NOT be queued. A served `activate` SHALL answer with a server-error status only for a failure on the server side: when no ref was found it SHALL answer 404, and when only some refs were found it SHALL answer 200; both carry the per-ref result (the activated refs, and each failed ref with its error).

#### Scenario: Parent recency is bumped
- **WHEN** parent note P is activated from a child and the parent is reachable
- **THEN** P's sidecar `LastUsed` on the parent is updated
