## ADDED Requirements

### Requirement: activate SHALL resolve refs locally first, then against the parent
`engram activate --note <ref>` SHALL resolve each ref against the local vault first, and SHALL bump the local note's sidecar `LastUsed` on a hit, as it does today. On a local miss with `ENGRAM_PARENT` set, or for any ref when `--parent` is given, `activate` SHALL treat the ref as a parent note and pull it down. `--parent` without `ENGRAM_PARENT` configured SHALL be an error, with no lookup performed. `activate` SHALL report each ref it could not activate on stderr, and SHALL exit non-zero if any ref failed.

#### Scenario: A parent-only ref pulls down
- **WHEN** `engram activate --note <basename>.md` runs with `ENGRAM_PARENT` set, and the note exists only in the parent
- **THEN** the parent note is pulled down as a local pending offer

#### Scenario: A local hit never contacts the parent
- **WHEN** `engram activate --note <ref>` runs and the ref resolves locally
- **THEN** the local sidecar's `LastUsed` is bumped, and no parent request is made

#### Scenario: Partial failure is visible
- **WHEN** `engram activate` is given two refs and one cannot be resolved locally or on the parent
- **THEN** stderr names the failed ref, and the command exits non-zero

#### Scenario: --parent without a parent configured
- **WHEN** `engram activate --parent --note x` runs with `ENGRAM_PARENT` unset
- **THEN** the command errors, and no note is read or written

### Requirement: A pulled-down note SHALL be an exact-content local pending offer
A pull-down SHALL fetch the parent note's file content verbatim (the served `show` raw mode, capability `vault-serve-api`). It SHALL then write a new local note under the vault lock, with:
- a fresh local top-level Luhmann ID;
- the parent's body unchanged, so that its content hash equals the parent's;
- the pending-offer marker;
- a parent link with `url` set to the configured parent, `note` set to the fetched basename, `via` set to `pulled`, and `hash` set to the fetched content hash.

The written note SHALL NOT carry the parent's own `parent` link, `aliases`, `offer`, `skill_hash`, `skill_key`, `skill_source`, `sources`, `supersedes`, or `tags`. It SHALL keep the parent's `repo`/`user`/`vault` values as authorship provenance. It SHALL be embedded on write, like any note. A fetched note whose type is not `fact`, `feedback`, or `runbook` SHALL be refused, and nothing SHALL be written.

#### Scenario: Pulled note is pending and linked
- **WHEN** parent note P is pulled down
- **THEN** a new local note exists carrying `pending: true`, `parent.note` naming P, `parent.via: pulled`, and P's content hash; it is excluded from local query results, and the next query payload has `pending_offers: true`

#### Scenario: Remote identity and registration fields are stripped
- **WHEN** the fetched parent note carries `skill_hash`, `skill_key`, `aliases`, or its own `parent` link
- **THEN** the local pending note carries none of those values, and its parent link is the one this pull-down set

#### Scenario: Chunk provenance does not travel
- **WHEN** the fetched parent note carries `sources` chunk IDs
- **THEN** the local note carries no `sources`

### Requirement: Pull-down SHALL be idempotent and loop-free
A pull-down SHALL write nothing when a local note (live or pending) already holds a parent link to the same parent note for the configured parent, with a `parent.hash` equal to the fetched content hash. When that local note is live, the pull-down SHALL bump its sidecar `LastUsed` instead. A parent note whose content changed since the last pull SHALL arrive as another pending offer. A pulled-down note SHALL never be offered back to the parent unless its content is later changed locally (capability `vault-parent-offers`).

#### Scenario: Re-activating an unchanged parent note
- **WHEN** parent note P is activated twice with no change to P in between
- **THEN** the local vault gains exactly one note for P

#### Scenario: A changed parent note arrives as a new offer
- **WHEN** P is pulled down and accepted, then P changes on the parent, and P is activated again
- **THEN** a second local pending offer for P exists, carrying the new content hash

#### Scenario: Accepting a pulled note sends nothing up
- **WHEN** curation clears the pending marker on a pulled note
- **THEN** no offer is queued for it

### Requirement: Pull-down SHALL also signal use to the parent
After a successful pull-down, or a no-op pull-down, `activate` SHALL send a best-effort `activate` request for the parent note to the parent. A failure of that request SHALL NOT fail the command, and SHALL NOT be queued.

#### Scenario: Parent recency is bumped
- **WHEN** parent note P is activated from a child and the parent is reachable
- **THEN** P's sidecar `LastUsed` on the parent is updated
