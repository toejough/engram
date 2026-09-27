## ADDED Requirements

### Requirement: Served learn SHALL return an offer receipt naming the pending note and the vault
A served `learn` response SHALL be a JSON object carrying:
- `status`: `offer received`;
- `luhmann`: the pending note's Luhmann ID;
- `basename`: the pending note's basename, without `.md`;
- `pending`: `true`;
- `vault_id`: the served vault's ID (capability `vault-local-first`);
- `for`: the resolved live target's basename, present only when the offer resolved to a live note.

It SHALL NOT report any curation outcome (capability `vault-offer-curation`).

#### Scenario: Receipt names the pending note
- **WHEN** a served `learn` request creates pending note `1100.2026-09-27.x.md` on the vault with ID `9a1e`
- **THEN** the response carries `status: offer received`, `luhmann: 1100`, `basename: 1100.2026-09-27.x`, `pending: true`, and `vault_id: 9a1e`

### Requirement: Served learn SHALL update a pending offer from the same origin in place
A served `learn` request MAY carry `offer.origin`, `offer.key`, and `offer.for`. The server SHALL handle them in this order:
1. When a pending note already carries the same `offer.origin`, the server SHALL rewrite that pending note's content and `offer.key` in place (it stays pending and keeps its basename), and SHALL return its receipt. When its `offer.key` already matches, the server SHALL write nothing and SHALL return the receipt.
2. Otherwise, `offer.for` SHALL be resolved against live notes' basenames, then against live notes' `aliases`, then against pending notes. A resolved pending note SHALL be updated in place, as in step 1. A resolved live note E SHALL leave E unchanged and live, SHALL record `offer.for: E` on a new pending note, and SHALL report `for: E` in the receipt.
3. A target that does not resolve SHALL be dropped, and the offer SHALL be written as a new pending note.

The server SHALL place every new pending note at top level, ignoring any caller-supplied `target` or `position`.

#### Scenario: An amend before curation updates the pending offer
- **WHEN** a child offers note L (creating pending N1), amends L, and the amend-offer arrives before the host has curated N1
- **THEN** N1 now carries the amended content, and no second pending note exists

#### Scenario: A duplicate key writes nothing
- **WHEN** a second served `learn` arrives with the same `offer.origin` and `offer.key` as an existing pending note
- **THEN** the response is that note's receipt, and no vault file changes

#### Scenario: An amend-offer names a live target
- **WHEN** a served `learn` arrives with `offer.for` naming live note E, and no pending note shares its origin
- **THEN** a new pending note carries `offer.for: E`, E itself is unchanged and still live, and the receipt carries `for: E`

#### Scenario: A renamed target resolves through aliases
- **WHEN** `offer.for` names a basename that a live note lists in its `aliases`
- **THEN** the offer resolves to that note

#### Scenario: An unresolvable target is dropped
- **WHEN** `offer.for` names no live note, no alias, and no pending note
- **THEN** the new pending note carries no `offer.for`

#### Scenario: Placement is ignored
- **WHEN** a served `learn` arrives with `target: "12"` and `position: "child"`
- **THEN** the pending note receives a new top-level Luhmann ID

### Requirement: Served learn SHALL NOT let callers set link, identity, or registration fields
The request decoding SHALL NOT populate a note's `parent`, `aliases`, `xid`, `skill_hash`, `skill_key`, or `skill_source`, whatever the key spelling in the request body. The only caller-settable exchange fields SHALL be `offer.origin`, `offer.key`, and `offer.for`, and they SHALL land only on pending notes.

#### Scenario: Remote-set origin fields are ignored
- **WHEN** a served `learn` request body includes `parent`, `aliases`, `xid`, `skillHash`, or `SkillKey` keys (camelCase or PascalCase)
- **THEN** the written pending note carries none of those fields

### Requirement: Served show SHALL offer a raw JSON envelope
The served `show` route SHALL accept a `raw` parameter. With `raw=1`, it SHALL resolve the `note` parameter against basenames and then against `aliases`, and SHALL return a JSON object carrying:
- `vault_id`;
- `basename`: the note's current basename;
- `content`: the note file's bytes, unmodified, with no `red_flags` preview cap and no appended links section;
- `exchange_hash` (capability `vault-parent-offers`).

A missing note SHALL produce a 404 with an error body, not a 500. Without `raw`, the route's output SHALL be unchanged.

#### Scenario: Raw show is exact
- **WHEN** `GET /show?note=<basename>&raw=1` is served for an existing note
- **THEN** the response is a JSON envelope whose `content` is byte-identical to the note file

#### Scenario: Raw show follows a rename
- **WHEN** `GET /show?note=<old basename>&raw=1` is served for a note renamed since, which lists the old basename in `aliases`
- **THEN** the envelope carries the note's current basename and content

#### Scenario: Missing note is not found
- **WHEN** `GET /show?note=<missing>&raw=1` is served
- **THEN** the response is a 404 with an error body

### Requirement: Served query SHALL return dedupe keys on request
The served `query` route SHALL accept a `dedupe-keys` parameter. With `dedupe-keys=1`, the returned payload SHALL carry a top-level `vault_id`, and each note item SHALL carry its `exchange_hash` and, when non-empty, its `aliases`. Without the parameter, the payload SHALL be unchanged.

#### Scenario: Keys present only on request
- **WHEN** `GET /query` is served once with `dedupe-keys=1` and once without
- **THEN** only the first response carries `vault_id` and per-note `exchange_hash`, and the second is byte-identical to a local `engram query` with the same arguments

## MODIFIED Requirements

### Requirement: Served command set is fixed and explicit
`engram serve` SHALL expose exactly `query`, `show`, `activate`, and `learn` over HTTP. `engram serve` SHALL NOT expose `amend`, `query-chunks`, `show-chunk`, `ingest`, `vocab refit`, `prune`, `check`, `update`, or `resituate`, and no route SHALL exist for them. (`amend` left the served set along with the `ENGRAM_SERVER` thin client, its only caller: children offer amendments through `learn` with an offer target, per capability `vault-parent-offers`. `query-chunks` and `show-chunk` left because chunks never cross vaults.)

#### Scenario: A served command is reachable over the API
- **WHEN** a request for `query`, `show`, `activate`, or `learn` is sent to a running `engram serve` instance
- **THEN** the request is handled and produces the same result the equivalent local CLI invocation would produce

#### Scenario: A host-only command has no route
- **WHEN** a request naming `amend`, `query-chunks`, `show-chunk`, `ingest`, `vocab refit`, `prune`, `check`, `update`, or `resituate` is sent to a running `engram serve` instance
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
identity is empty, and SHALL perform no vault write in that case. The declared
identity SHALL survive the offer's acceptance: bookkeeping amends on the host
(`--clear-pending`, `--activate`, `--discard --into`) do not re-stamp it
(capability `vault-note-identity`).

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
**Reason**: The thin-client mode is removed by Joe's 2026-09-27 decision (vault note 784a). Every environment now keeps its own local vault and exchanges notes with its parent through offers (capabilities `vault-local-first`, `vault-parent-offers`, `vault-parent-pulldown`). Setting `ENGRAM_SERVER` is now a hard error.
**Migration**: Set `ENGRAM_PARENT` to the same URL. Queries then merge local results with the parent's notes, and writes are made locally and offered to the parent.
