## Purpose

Lets child environments, each with its own local vault, exchange notes with this parent over HTTP (offers up, notes-only reads and raw pull-down), reusing the CLI's code paths and locks rather than a parallel implementation, with writes attributed to the calling instance's own client-declared identity.
## Requirements
### Requirement: Served command set is fixed and explicit
`engram serve` SHALL expose exactly `query`, `show`, `activate`, and `learn` over HTTP. `engram serve` SHALL NOT expose `amend`, `query-chunks`, `show-chunk`, `ingest`, `vocab refit`, `prune`, `check`, `update`, or `resituate`, and no route SHALL exist for them. (`amend` left the served set along with the `ENGRAM_SERVER` thin client, its only caller: children offer amendments through `learn` with an offer target, per capability `vault-parent-offers`. `query-chunks` and `show-chunk` left because chunks never cross vaults.)

#### Scenario: A served command is reachable over the API
- **WHEN** a request for `query`, `show`, `activate`, or `learn` is sent to a running `engram serve` instance
- **THEN** the request is handled and produces the same result the equivalent local CLI invocation would produce

#### Scenario: A host-only command has no route
- **WHEN** a request naming `amend`, `query-chunks`, `show-chunk`, `ingest`, `vocab refit`, `prune`, `check`, `update`, or `resituate` is sent to a running `engram serve` instance
- **THEN** the server returns an error and performs no vault operation

### Requirement: Bind address must be explicit
`engram serve` SHALL require an explicit, operator-configured bind address and SHALL NOT default to `0.0.0.0`.

#### Scenario: No bind address supplied
- **WHEN** `engram serve` is started without an explicit bind address
- **THEN** the server refuses to start rather than defaulting to a wildcard address

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

### Requirement: Served query caps the `text` parameter at 2 KB
The served `query` route SHALL accept the client's `--text` value as the `text` query-string parameter, and SHALL truncate it to at most 2048 bytes on a UTF-8 rune boundary (never splitting a multi-byte character) before running the query. Truncation SHALL be silent — no error and no warning — because trigger cues occur early in a message. A `text` value of 2048 bytes or fewer SHALL be passed through unchanged.

#### Scenario: Short text round-trips unchanged
- **WHEN** a client sends `text` of 2048 bytes or fewer, including quotes, newlines, `&`, and non-ASCII characters
- **THEN** the server queries with the identical string

#### Scenario: Long text is truncated on a rune boundary
- **WHEN** a client sends `text` longer than 2048 bytes and byte 2048 falls inside a multi-byte character
- **THEN** the server queries with the text cut back to the last complete character at or before byte 2048, and returns no error

### Requirement: Served learn SHALL return an offer receipt naming the note and the vault
A served `learn` response SHALL be a JSON object carrying:
- `status`: `offer received`;
- `luhmann`: the Luhmann ID of the note the offer landed on;
- `basename`: that note's basename, without `.md`;
- `pending`: that note's true pending state — `true` for a pending offer, `false` when a retry matched an already-accepted live note;
- `vault_id`: the served vault's ID (capability `vault-local-first`);
- `stored_hash`: the exchange hash of the note as stored;
- `for`: the resolved live target's basename, present only when the offer resolved to a live note.

It SHALL NOT report any curation outcome (capability `vault-offer-curation`).

#### Scenario: Receipt names the pending note
- **WHEN** a served `learn` request creates pending note `1100.2026-09-27.x.md` on the vault with ID `9a1e`
- **THEN** the response carries `status: offer received`, `luhmann: 1100`, `basename: 1100.2026-09-27.x`, `pending: true`, and `vault_id: 9a1e`

### Requirement: Served learn SHALL update a pending offer in place only for the same origin
A served `learn` request MAY carry `offer.origin`, `offer.key`, `offer.for`, and `offer.path`. The server SHALL answer 400, and SHALL write nothing, in any of these cases:
- `offer.path` has more than 16 entries;
- any entry of `offer.path` is not exactly 32 lowercase hexadecimal characters;
- `offer.origin` is not two such 32-character identifiers joined by `:`;
- `offer.key` or `offer.path` is present without `offer.origin`.

The server SHALL refuse the request with a 409, and SHALL write nothing, when its own vault ID is already in `offer.path`. Otherwise it SHALL do all of the following in one locked section: the lookup, any rewrite, any re-embed, and building the receipt. It SHALL check these cases in order:
0. **Any note, live or pending, has already recorded the same non-empty `offer.key`**, as its current `offer.key` or in `offer.prior_keys` (a retry). The server SHALL write nothing and SHALL return that note's current receipt. This check comes first, so a late retry of an already-accepted or superseded key never rewrites the note back to older content.
1. **A pending note carries the same `offer.origin`.** The server SHALL rewrite that pending note's content, `offer.key`, and `offer.path` in place, SHALL move the superseded `offer.key` into `offer.prior_keys` (keeping the most recent 8, oldest dropped) (the note stays pending and keeps its basename), SHALL re-embed it when its exchange hash changed, and SHALL return its receipt.
2. **A live note carries the same `offer.origin`** (an accepted offer, with a new key). The server SHALL write a new pending note whose `offer.for` names that live note.
3. **Otherwise**, `offer.for` SHALL be resolved against live notes' basenames, then their `aliases`, then pending notes.
   - A resolved **pending note of a different origin** SHALL NOT be modified. The server SHALL write a new pending note whose `offer.for` names it.
   - A resolved live note E SHALL be left unchanged and live. The server SHALL write a new pending note with `offer.for: E`, and SHALL report `for: E` in the receipt.
   - A target that does not resolve SHALL be dropped, and the offer SHALL be written as a new pending note.

The server SHALL place every new pending note at top level, ignoring any caller-supplied `target` or `position`. The receipt SHALL carry `stored_hash`, the exchange hash of the note as stored.

#### Scenario: An amend before curation updates the pending offer
- **WHEN** a child offers note L (creating pending N1), amends L, and the amend-offer arrives before the host has curated N1
- **THEN** N1 now carries the amended content and a re-embedded sidecar, and no second pending note exists

#### Scenario: A duplicate key writes nothing
- **WHEN** a second served `learn` arrives with the same `offer.origin` and `offer.key` as an existing pending note
- **THEN** the response is that note's receipt, and no vault file changes

#### Scenario: A retry after acceptance writes nothing
- **WHEN** an offer was accepted (its note is live and still carries `offer.origin`), and the same offer is retried with the same key
- **THEN** the response is the live note's receipt with `pending: false`, and no pending note is created

#### Scenario: A late retry never reverts a newer amend
- **WHEN** an accepted live note carries `offer.origin` O and `offer.key` K1, a newer pending amend from O carries `offer.key` K2, and a retry of O with K1 arrives late
- **THEN** the response is the live note's receipt, and the pending amend is unchanged — no vault file changes

#### Scenario: A late retry of a superseded key never reverts the note
- **WHEN** a child offers note L with key K1 (creating pending N1), amends L so N1 is rewritten in place with key K2, and the K1 offer is then retried late — before or after N1 is accepted
- **THEN** the response is N1's current receipt, N1 still carries the K2 content, and no vault file changes

#### Scenario: A key without an origin is rejected
- **WHEN** a served `learn` arrives whose offer carries `offer.key` or `offer.path` but no `offer.origin`
- **THEN** the response is a 400, and nothing is written

#### Scenario: No cross-origin overwrite
- **WHEN** an offer from origin B names, in `offer.for`, a pending note that came from origin A
- **THEN** A's pending note is unchanged, and a new pending note with `offer.for` naming A's note is written

#### Scenario: A malformed or oversized path is rejected
- **WHEN** a served `learn` arrives whose `offer.path` has 17 entries, or has an entry that is not 32 hex characters
- **THEN** the response is a 400, and nothing is written

#### Scenario: A cycle is refused
- **WHEN** a served `learn` arrives whose `offer.path` already contains the server's own vault ID
- **THEN** the response is a 409, and nothing is written

#### Scenario: An amend-offer names a live target
- **WHEN** a served `learn` arrives with `offer.for` naming live note E, and no note shares its origin
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
The request decoding SHALL NOT populate a note's `parent`, `aliases`, `xid`, `offer.prior_keys`, `skill_hash`, `skill_key`, or `skill_source`, whatever the key spelling in the request body. The only caller-settable exchange fields SHALL be `offer.origin`, `offer.key`, `offer.for`, and `offer.path`, and they SHALL land only on pending notes. Served learn SHALL drop any `tags` in the request: offers carry no tags, and the receiving vault's own vocab assigns them (design D5).

#### Scenario: Remote-set origin fields are ignored
- **WHEN** a served `learn` request body includes `parent`, `aliases`, `xid`, `skillHash`, or `SkillKey` keys (camelCase or PascalCase)
- **THEN** the written pending note carries none of those fields

#### Scenario: Wire tags are dropped
- **WHEN** a served `learn` request body carries `tags`
- **THEN** the pending note, new or updated in place, carries none of those tags

### Requirement: Served show SHALL offer a raw JSON envelope
The served `show` route SHALL accept a `raw` parameter. With `raw=1`, it SHALL resolve the `note` parameter against basenames and then against `aliases`, and SHALL return a JSON object carrying:
- `vault_id`;
- `basename`: the note's current basename;
- `content`: the note file's bytes, unmodified, with no `red_flags` preview cap and no appended links section;
- `exchange_hash` (capability `vault-parent-offers`).

A missing note SHALL produce a 404 with an error body, not a 500. A pending note SHALL be treated as missing on the raw route, whether it is named by basename or by alias: it is an unvetted offer, not the vault's note, so it is never pulled down as one. Without `raw`, the route SHALL return exactly the local `engram show` output, including the exchange-hash header line for notes carrying `xid` (capability `vault-offer-curation`).

#### Scenario: Raw show is exact
- **WHEN** `GET /show?note=<basename>&raw=1` is served for an existing note
- **THEN** the response is a JSON envelope whose `content` is byte-identical to the note file

#### Scenario: Raw show follows a rename
- **WHEN** `GET /show?note=<old basename>&raw=1` is served for a note renamed since, which lists the old basename in `aliases`
- **THEN** the envelope carries the note's current basename and content

#### Scenario: Missing note is not found
- **WHEN** `GET /show?note=<missing>&raw=1` is served
- **THEN** the response is a 404 with an error body

#### Scenario: A pending note is not served raw
- **WHEN** `GET /show?note=<basename or alias of a pending note>&raw=1` is served
- **THEN** the response is a 404 with an error body

### Requirement: Served activate SHALL resolve refs only against the vault's note names
The served `activate` route SHALL resolve each ref only against the vault's listed note names (with or without `.md`), as the raw `show` route does; a ref SHALL never be joined into a filesystem path. A request with any ref that is an absolute path, contains a path separator (`/` or `\`), or contains `..` SHALL be answered 400, and nothing SHALL be read or written. The local `engram activate` command keeps accepting absolute and vault-relative paths.

#### Scenario: A path ref is rejected
- **WHEN** `POST /activate` names `/etc/x`, `../x.md`, or an absolute path to a note outside the vault
- **THEN** the response is a 400, and no sidecar inside or outside the vault is changed

#### Scenario: A listed note name is activated
- **WHEN** `POST /activate` names a vault note's basename, with or without `.md`
- **THEN** that note's sidecar `LastUsed` is bumped

### Requirement: Served request and parent response bodies SHALL be size-capped
`engram serve` SHALL read at most 4 MiB of a request body; a body it could not read in full SHALL be answered 413 by every route, and nothing SHALL be written. A child SHALL accept at most 16 MiB of a parent response; a longer response SHALL be an error, never a truncated payload.

#### Scenario: An oversized request is refused
- **WHEN** a served request's body exceeds the cap
- **THEN** the response is a 413, and the vault is unchanged

#### Scenario: An oversized parent response is refused
- **WHEN** a parent's response body exceeds the child's cap
- **THEN** the request fails as the parent's failure, and nothing from it is stored

### Requirement: Served query SHALL return dedupe keys on request
The served `query` route SHALL accept a `dedupe-keys` parameter. With `dedupe-keys=1`, the returned payload SHALL carry a top-level `vault_id`, and each note item SHALL carry its `exchange_hash` and, when non-empty, its `aliases`. Without the parameter, the payload SHALL be unchanged.

#### Scenario: Keys present only on request
- **WHEN** `GET /query` is served once with `dedupe-keys=1` and once without
- **THEN** only the first response carries `vault_id` and per-note `exchange_hash`, and the second is byte-identical to a local `engram query` with the same arguments

