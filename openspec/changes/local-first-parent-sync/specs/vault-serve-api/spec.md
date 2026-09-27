## ADDED Requirements

### Requirement: Served learn SHALL return an offer receipt naming the pending note
A served `learn` response SHALL be a JSON object carrying:
- `status`: `offer received`;
- `luhmann`: the pending note's Luhmann ID;
- `basename`: the pending note's basename, without `.md`;
- `pending`: `true`.

It SHALL NOT report any curation outcome (capability `vault-offer-curation`).

#### Scenario: Receipt names the pending note
- **WHEN** a served `learn` request creates pending note `1100.2026-09-27.x.md`
- **THEN** the response is `{"status":"offer received","luhmann":"1100","basename":"1100.2026-09-27.x","pending":true}`

### Requirement: Served learn SHALL accept an offer target and an idempotency key
A served `learn` request MAY carry `offer.for` (a basename in the served vault) and `offer.key` (an idempotency key). The server SHALL resolve `offer.for` against the basenames of live notes, then against their `aliases`. It SHALL record the resolved basename as `offer.for` on the pending note. When `offer.for` does not resolve, the server SHALL drop it and treat the offer as a new note. When a pending note already carries the same `offer.key`, the server SHALL return that note's receipt and SHALL write nothing.

#### Scenario: An amend-offer names its target
- **WHEN** a served `learn` arrives with `offer.for` naming live note E
- **THEN** the pending note carries `offer.for: E`, and E itself is unchanged and still live

#### Scenario: An unresolvable target is dropped
- **WHEN** `offer.for` names no live note and no alias
- **THEN** the pending note carries no `offer.for`

#### Scenario: A duplicate key returns the existing receipt
- **WHEN** a second served `learn` arrives with an `offer.key` equal to an existing pending note's key
- **THEN** the response is that note's receipt, and no new note is written

### Requirement: Served learn SHALL NOT honor caller placement or origin fields
A served `learn` SHALL place its pending note at top level, ignoring any caller-supplied `target` or `position`. The request decoding SHALL NOT set a note's `parent` link, `aliases`, `skill_hash`, `skill_key`, or `skill_source` (these are tagged so JSON decoding cannot populate them), regardless of what the request body contains.

#### Scenario: Placement is ignored
- **WHEN** a served `learn` arrives with `target: "12"` and `position: "child"`
- **THEN** the pending note receives a new top-level Luhmann ID

#### Scenario: Remote-set origin fields are ignored
- **WHEN** a served `learn` request body includes `parent`, `aliases`, `skillHash`, or `SkillKey` keys (camelCase or PascalCase)
- **THEN** the written pending note carries none of those fields

### Requirement: Served show SHALL offer a raw mode
The served `show` route SHALL accept a `raw` parameter. With `raw=1`, it SHALL return the note file's bytes exactly as stored, with no `red_flags` preview cap and no appended links section. A missing note SHALL produce a not-found error response, not a 500.

#### Scenario: Raw show is byte-exact
- **WHEN** `GET /show?note=<basename>&raw=1` is served for an existing note
- **THEN** the response body is byte-identical to the note file

#### Scenario: Missing note is not found
- **WHEN** `GET /show?note=<missing>&raw=1` is served
- **THEN** the response is a 404 with an error body

### Requirement: Served query SHALL return dedupe keys on request
The served `query` route SHALL accept a `dedupe-keys` parameter. With `dedupe-keys=1`, each note item in the returned payload SHALL carry `content_hash` (the note's sidecar content hash) and, when non-empty, `aliases`. Without the parameter, the payload SHALL be unchanged.

#### Scenario: Keys present only on request
- **WHEN** `GET /query` is served once with `dedupe-keys=1` and once without
- **THEN** note items carry `content_hash` only in the first response, and the second response is byte-identical to a local `engram query` with the same arguments

## MODIFIED Requirements

### Requirement: Served command set is fixed and explicit
`engram serve` SHALL expose exactly `query`, `query-chunks`, `show`, `show-chunk`, `activate`, and `learn` over HTTP. `engram serve` SHALL NOT expose `amend`, `ingest`, `vocab refit`, `prune`, `check`, `update`, or `resituate` — no route SHALL exist for these commands. (`amend` left the served set with the removal of the `ENGRAM_SERVER` thin client, its only caller; child environments offer amendments through `learn` with an offer target — capability `vault-parent-offers`.)

#### Scenario: A served command is reachable over the API
- **WHEN** a request for `query`, `query-chunks`, `show`, `show-chunk`, `activate`, or `learn` is sent to a running `engram serve` instance
- **THEN** the request is handled and produces the same result the equivalent local CLI invocation would produce

#### Scenario: A host-only command has no route
- **WHEN** a request naming `amend`, `ingest`, `vocab refit`, `prune`, `check`, `update`, or `resituate` is sent to a running `engram serve` instance
- **THEN** the server returns an error and performs no vault operation

### Requirement: Served commands reuse existing code paths and locks
Every served command SHALL execute through the same implementation and the same vault locks (ADR-0013) that the local CLI already uses for that command — not a separate implementation.

#### Scenario: Concurrent local and remote writers do not lose an update
- **WHEN** a local `engram learn`/`amend` invocation and a served `learn` request race against the same vault
- **THEN** both writes are applied without either being silently lost or corrupting vault state

### Requirement: Served writes are attributed to the client-declared caller identity
For a served `learn` request, the server SHALL stamp the note's `user:` field from
the identity value the calling `engram` instance declared in the request body — the same
locally-detected value that instance would have used for a local write. The server SHALL NOT
require or consult any edge-authentication header (Cloudflare Access or otherwise) to resolve
this field. The server SHALL reject a served `learn` request whose declared
identity is empty, and SHALL perform no vault write in that case.

#### Scenario: A caller's declared identity is stamped as-is
- **WHEN** a served `learn` request arrives with a self-detected `user:` value naming identity A
- **THEN** the resulting pending-offer note's `user:` field is A

#### Scenario: An empty declared identity is rejected
- **WHEN** a served `learn` request arrives with an empty `user:` value
- **THEN** the server returns an error and performs no vault write

#### Scenario: No edge-authentication header is required
- **WHEN** a served `learn` request arrives with a non-empty declared `user:` value and no Cloudflare Access (or other edge-authentication) header at all
- **THEN** the request succeeds and the resulting note's `user:` field is the declared value

### Requirement: repo: is not server-overridden on served writes
For a served `learn` request, the server SHALL resolve `repo:` the same way local `learn` already does (client/caller-detected) — SHALL NOT substitute a server-derived value for `repo:`.

#### Scenario: repo: reflects the caller's own detection
- **WHEN** a served `learn` request arrives with a self-detected `repo:` value
- **THEN** the resulting note's `repo:` field is that self-detected value, unchanged by the server

## REMOVED Requirements

### Requirement: The CLI is a transparent HTTP client when ENGRAM_SERVER is set
**Reason**: The thin-client mode is removed by Joe's 2026-09-27 decision (vault note 784a). Every environment now keeps its own local vault, and exchanges notes with its parent through offers (capabilities `vault-local-first`, `vault-parent-offers`, `vault-parent-pulldown`). Setting `ENGRAM_SERVER` is now a hard error.
**Migration**: Set `ENGRAM_PARENT` to the same URL. Queries then merge local and parent results, and writes are made locally and offered to the parent.
