# Vault Serving - Product Spec

`engram serve` is a node's network door for its children: other environments that each keep their own local vault. A
child offers the notes it learns to this node and reads this node's notes back over HTTP, through four commands only
(`query`, `show`, `activate` and `learn`). The local command line is the same node's local door and never runs through
the server. The server reuses the command line's code paths and locks, and a served `learn` lands as a pending offer
(curate judges it; [curate](./curate.md)), attributed to the identity the calling instance declares. The server has no
authentication of its own, so the network that can reach its address is the trust boundary. The children's side of the
exchange (the outbox, merged queries and pull-down) is [parent-sync](./parent-sync.md). The unrelated multi-agent chat server of an
earlier era and the Cloudflare dependency are gone, and nothing here depends on either.

## 792a20a The served command set is fixed

Requirement: [792a20a The served command set is fixed](#792a20a-the-served-command-set-is-fixed) `engram serve` MUST expose exactly `query` and `show` (by GET) and `activate` and `learn` (by
POST) over HTTP, and MUST NOT expose `amend`, `query-chunks`, `show-chunk`, `ingest`, `vocab refit`, `prune`, `check`,
`update` or `resituate`: a request for any of them MUST be answered with an error and perform no vault operation.

Rationale: `amend` left the served set along with the `ENGRAM_SERVER` thin client, its only caller, and children offer
amendments through `learn` with an offer target. `query-chunks` and `show-chunk` left because chunks never cross
vaults. One write route keeps one path for offers.
The route split is by mutation: the two read-only commands are GET and the two writes are POST. `learn` is a
POST because its repeatable flags and free text do not fit in a URL, and `activate` follows the split because it
is a write. The server verifies no field of a served `learn`: the `user:` that carries the attribution weight is
the one the caller declares ([792a20e A served learn is attributed to the identity the caller declares](#792a20e-a-served-learn-is-attributed-to-the-identity-the-caller-declares)). Offer and
curation judgment run off the request path, in a skill and not in server code.

### 792a20a1 UAT Scenarios

```gherkin
Scenario: 792a20a1a A served command is reachable
  Given a running engram serve
  When a request for query, show, activate or learn is sent with the method it takes
  Then the request is handled and produces the result the equivalent local command would

Scenario: 792a20a1b A host-only command has no route
  Given a running engram serve
  When a request names amend, query-chunks, show-chunk, ingest, vocab refit, prune, check, update or resituate
  Then the server answers with an error and performs no vault operation

Scenario: 792a20a1c A served command refuses the wrong method
  Given a running engram serve
  When a GET is sent to learn
  Then the server answers 405
```

## 792a20b A bind address must be given

Requirement: [792a20b A bind address must be given](#792a20b-a-bind-address-must-be-given) `engram serve` MUST require an explicit bind address in `--addr` and MUST NOT start without one,
and MUST NOT default to the wildcard address `0.0.0.0` or to any address. `engram serve --help` MUST describe the four
served commands, say that the address is required and never defaults to `0.0.0.0`, and list `--vault`, `--vault-name`
and `--chunks-dir`.

Rationale: a server that binds a default address can expose a vault to a network its operator never chose, so the
operator names the address. The address has no environment fallback either.

### 792a20b1 UAT Scenarios

```gherkin
Scenario: 792a20b1a No address, no server
  Given the user runs engram serve with no --addr
  When the command starts
  Then it refuses with "missing required flag: --addr" and no server is listening

Scenario: 792a20b1b A given address is bound
  Given the user runs engram serve --addr 127.0.0.1:8420
  When the command starts
  Then the server listens on that address and nowhere else

Scenario: 792a20b1c The help states the address rule
  Given the user runs engram serve --help
  When the help is read
  Then it names query, show, activate and learn, says --addr is required and never defaults to 0.0.0.0, and lists --vault, --vault-name and --chunks-dir
```

## 792a20c Served commands behave as the local ones

Requirement: [792a20c Served commands behave as the local ones](#792a20c-served-commands-behave-as-the-local-ones) Every served command MUST run through the same implementation and the same vault locks as the
local command, MUST give the result the local command gives for the same arguments, and MUST NOT let a served request
change which vault, vault name or chunk index the server uses. A served `query` without `dedupe-keys` MUST be
byte-identical to the local payload, and a served `show` without `raw` MUST be exactly the local output.

Rationale: a second implementation of a command would drift from the first. Sharing the vault lock means a local write
and a served write racing on one vault each apply, with neither lost. The vault, vault name and chunk index are fixed
once at startup.

### 792a20c1 UAT Scenarios

```gherkin
Scenario: 792a20c1a A served query matches the local query
  Given a vault and a query with a phrase
  When the same query runs locally and as GET /query
  Then the served response is byte-identical to the local payload

Scenario: 792a20c1b A served show matches the local show
  Given a vault holding a note
  When the note is shown locally and as GET /show without raw
  Then the served response is exactly the local output

Scenario: 792a20c1c Concurrent local and served writers lose nothing
  Given a local engram learn and a served learn racing against one vault
  When both complete
  Then both notes are written, neither is lost and the vault is not corrupted

Scenario: 792a20c1d A request cannot redirect the server
  Given a served request that carries a vault, vault name or chunks directory of its own
  When the server handles it
  Then the server's own vault and chunk index are used
```

## 792a20d A served learn lands as a pending offer and returns a receipt

Requirement: [792a20d A served learn lands as a pending offer and returns a receipt](#792a20d-a-served-learn-lands-as-a-pending-offer-and-returns-a-receipt) A served `learn` MUST write its note as a pending offer in the served vault, never as an
immediately live note, and MUST answer with a receipt that is a JSON object carrying `status` (`offer received`),
`luhmann` and `basename` of the note the offer landed on, `pending` (its true state: `true` for a pending offer,
`false` when a retry matched an already accepted live note), `vault_id` (the served vault's ID), `stored_hash` (the
exchange hash of the note as stored) and, only when the offer resolved to a live note, `for` (that note's basename).
The receipt MUST NOT carry the note's content or any curation outcome.

Rationale: curation judges an offer before it counts as knowledge ([curate](./curate.md)), so the receiving side learns nothing it
has not vetted. The receipt tells the child what the note became, so a retry or a later amend finds the same note.

### 792a20d1 UAT Scenarios

```gherkin
Scenario: 792a20d1a The receipt names the pending note
  Given a served vault with ID 9a1e
  When a served learn creates pending note 1100.2026-09-27.x.md
  Then the response carries status "offer received", luhmann "1100", basename "1100.2026-09-27.x", pending true and vault_id "9a1e"

Scenario: 792a20d1b The offered note is pending, not live
  Given a served learn that succeeded
  When the note is read
  Then it carries the pending marker and engram query does not return it

Scenario: 792a20d1c The receipt carries no content
  Given a served learn that succeeded
  When the response is read
  Then it holds no note content and no curation outcome
```

## 792a20e A served learn is attributed to the identity the caller declares

Requirement: [792a20e A served learn is attributed to the identity the caller declares](#792a20e-a-served-learn-is-attributed-to-the-identity-the-caller-declares) A served `learn` MUST stamp the note's `user:` from the identity the calling instance declares in
the request body, the same locally detected value that instance would use for a local write, and MUST NOT require or
consult any edge-authentication header to resolve it. It MUST reject a request whose declared identity is empty with
an error and write nothing. The declared identity MUST survive the offer's acceptance, because the bookkeeping amends
that accept it do not re-stamp it ([curate](./curate.md)).

Rationale: the server trusts what a caller declares, since the caller and the server share a network trust boundary and
no more. An empty declaration would write an offer nobody can be asked about, so it is refused outright. Requiring an
edge header would tie the server to a particular front door.
A credential map and a trusted-header adapter were both rejected: attribution needs a stable caller-chosen
name and not an unforgeable one, and the trust boundary is network reachability. Only the empty string is
refused, since it is the one value that cannot be a real identity. Two `engram serve` instances on one vault
are allowed, because the flock-covered writes and stateless pending-offer detection make that safe and the
remaining cost is redundant resource use.

### 792a20e1 UAT Scenarios

```gherkin
Scenario: 792a20e1a The declared identity is stamped as given
  Given a served learn whose body declares user alice@example.com
  When the offer is written
  Then the pending note's user: is alice@example.com

Scenario: 792a20e1b An empty identity is rejected
  Given a served learn whose declared user is empty
  When the server handles it
  Then it answers 400 with "serve: user: must be non-empty" and writes nothing

Scenario: 792a20e1c No edge header is needed
  Given a served learn with a declared user and no authentication header of any kind
  When the server handles it
  Then the request succeeds and the note's user: is the declared value
```

## 792a20f A served learn stamps the caller's repo, the served vault's name and the first-write identity

Requirement: [792a20f A served learn stamps the caller's repo, the served vault's name and the first-write identity](#792a20f-a-served-learn-stamps-the-callers-repo-the-served-vaults-name-and-the-first-write-identity) A served `learn` MUST take `repo:` from the value the caller declares and MUST NOT substitute a
value the server detects for itself. It MUST stamp `vault:` with the vault name the caller sent and otherwise with the
name the server was started with. It MUST apply this one stamp to every write of an offer, a new pending note and a
same-origin pending note rewritten in place, so that the rewrite takes the identity of its own request.

Rationale: the server's own repository context is the remote caller's wrong answer, so detection stays with the
caller. The first-write stamp is the one every learn applies ([792a3o Repository provenance](./note-writing.md#792a3o-repository-provenance), [792a3p User provenance](./note-writing.md#792a3p-user-provenance)), and a served learn's rewrite of a pending
offer is a first write of that content.

### 792a20f1 UAT Scenarios

```gherkin
Scenario: 792a20f1a repo reflects the caller's detection
  Given a served learn that declares repo "r"
  When the offer is written
  Then the note's repo: is "r", unchanged by the server

Scenario: 792a20f1b vault falls back to the server's name
  Given a served learn that sends no vault name and a server started with --vault-name personal
  When the offer is written
  Then the note's vault: is personal

Scenario: 792a20f1c An in-place rewrite is stamped from its own request
  Given a pending offer written with repo "r", and a same-origin served learn that declares no repo
  When the pending offer is rewritten
  Then the rewritten note carries no repo: and the user: of the new request
```

## 792a20g A served learn finds the note an offer belongs to

Requirement: [792a20g A served learn finds the note an offer belongs to](#792a20g-a-served-learn-finds-the-note-an-offer-belongs-to) A served `learn` MUST do its lookup, any rewrite, any re-embed and the building of its receipt in
one locked section, and MUST check these cases in order. An offer whose non-empty `offer.key` any note, live or
pending, already records as its current key or among `offer.prior_keys` MUST write nothing and return that note's
current receipt. Otherwise an offer whose `offer.origin` a pending note carries MUST rewrite that pending note in
place (new content, key and path, superseded key moved to `offer.prior_keys` with the most recent 8 kept, still
pending, same basename, re-embedded when the exchange hash changed). Otherwise an offer whose `offer.origin` a live
note carries MUST write a new pending note whose `offer.for` names that live note. Otherwise `offer.for` MUST be
resolved against live notes' basenames, then their `aliases`, then pending notes' basenames: a resolved pending note of
another origin MUST be left unmodified and a new pending note written with `offer.for` naming it, a resolved live note
MUST be left unchanged and live with a new pending note written with `offer.for` naming it and the receipt carrying
`for`, and a target that does not resolve MUST be dropped and the offer written as a new pending note.

Rationale: a retry must be harmless, and the key check comes first so that a late retry of an already accepted or
superseded key never rewrites a newer same-origin amend back to old content. An amend before curation updates the
pending note rather than queueing a second one. A different origin's pending note is never overwritten, because the
caller does not own it.

### 792a20g1 UAT Scenarios

```gherkin
Scenario: 792a20g1a An amend before curation updates the pending offer
  Given a child offered note L, creating pending N1, and then amended L before the host curated N1
  When the amend-offer arrives
  Then N1 carries the amended content with a re-embedded sidecar and no second pending note exists

Scenario: 792a20g1b A duplicate key writes nothing
  Given a second served learn with the same offer.origin and offer.key as an existing pending note
  When the server handles it
  Then the response is that note's receipt and no vault file changes

Scenario: 792a20g1c A retry after acceptance writes nothing
  Given an offer that was accepted, so its note is live and still carries offer.origin, and the same offer retried with the same key
  When the server handles it
  Then the response is the live note's receipt with pending false and no pending note is created

Scenario: 792a20g1d A late retry never reverts a newer amend
  Given an accepted live note with offer.origin O and key K1, a newer pending amend from O with key K2, and a late retry of O with K1
  When the server handles it
  Then the response is the live note's receipt and the pending amend is unchanged

Scenario: 792a20g1e A late retry of a superseded key never reverts the note
  Given a child offered note L with key K1, creating pending N1, amended L so that N1 was rewritten in place with key K2, and then retried K1 late
  When the server handles it
  Then the response is N1's current receipt, N1 still carries the K2 content and no vault file changes

Scenario: 792a20g1f A new key from an accepted origin makes a new pending note
  Given a live note that carries offer.origin O and a served learn from O with a new key
  When the server handles it
  Then a new pending note is written whose offer.for names the live note, and the receipt carries for

Scenario: 792a20g1g No cross-origin overwrite
  Given an offer from origin B whose offer.for names a pending note that came from origin A
  When the server handles it
  Then A's pending note is unchanged and a new pending note whose offer.for names A's note is written

Scenario: 792a20g1h An amend-offer names a live target
  Given a served learn whose offer.for names live note E and no note shares its origin
  When the server handles it
  Then a new pending note carries offer.for E, E is unchanged and live, and the receipt carries for E

Scenario: 792a20g1i A renamed target resolves through its aliases
  Given offer.for names a basename that a live note lists in its aliases
  When the server handles the offer
  Then the offer resolves to that note

Scenario: 792a20g1j An unresolvable target is dropped
  Given offer.for names no live note, no alias and no pending note
  When the server handles the offer
  Then the new pending note carries no offer.for

Scenario: 792a20g1k Prior keys are bounded
  Given a pending note rewritten in place more than eight times
  When its offer.prior_keys is read
  Then it holds at most the eight most recent superseded keys, oldest dropped

Scenario: 792a20g1l An unreadable note fails the request
  Given a note in the served vault that cannot be read for a reason other than being removed
  When a served learn runs its lookup
  Then the server answers an error and writes nothing, so that the child retries
```

## 792a20h An invalid offer is refused with nothing written

Requirement: [792a20h An invalid offer is refused with nothing written](#792a20h-an-invalid-offer-is-refused-with-nothing-written) A served `learn` MUST answer 400 and write nothing when `offer.path` has more than 16 entries,
when any `offer.path` entry is not exactly 32 lowercase hexadecimal characters, when `offer.origin` is not two such
identifiers joined by `:`, when `offer.key` or `offer.path` is present without `offer.origin`, or when the note content
fails validation (its type, tier, supersedes or a required field). It MUST answer 409 and write nothing when its own
vault ID is already in `offer.path`, with a body carrying `error`, `reason` `loop` and its `vault_id`. Each refusal MUST
come before the vault lock is taken.

Rationale: `offer.path` bounds how far an offer propagates and lets a server that sits on a misconfigured cycle refuse
it. A malformed offer is a client error the child must not retry, so it is a 400 rather than a 5xx. The `loop` reason
and the vault ID let a child that is its own parent, or sits on the cycle, recognize and cache the refusal.

### 792a20h1 UAT Scenarios

```gherkin
Scenario: 792a20h1a A key without an origin is rejected
  Given a served learn whose offer carries offer.key or offer.path and no offer.origin
  When the server handles it
  Then the response is 400 and nothing is written

Scenario: 792a20h1b A malformed or oversized path is rejected
  Given a served learn whose offer.path has 17 entries, or an entry that is not 32 hex characters
  When the server handles it
  Then the response is 400 and nothing is written

Scenario: 792a20h1c A malformed origin is rejected
  Given a served learn whose offer.origin is not two 32-character identifiers joined by a colon
  When the server handles it
  Then the response is 400 and nothing is written

Scenario: 792a20h1d A cycle is refused
  Given a served learn whose offer.path already contains the server's own vault ID
  When the server handles it
  Then the response is 409 with reason "loop" and the server's vault_id, and nothing is written

Scenario: 792a20h1e Invalid note content is a client error
  Given a served learn whose note fails validation, such as an unknown type
  When the server handles it
  Then the response is 400 and nothing is written
```

## 792a20i Placement and tags are the server's

Requirement: [792a20i Placement and tags are the server's](#792a20i-placement-and-tags-are-the-servers) A served `learn` MUST place every new pending note at top level and MUST ignore any `target` or
`position` the caller sent, and MUST drop any `tags` in the request, both on a new pending note and on a same-origin
rewrite.

Rationale: placement in the Luhmann tree is the receiving vault's, and tags are the vocabulary of the vault that holds
the note, so the receiving vault assigns its own ([792a20o The receiving vault tags a served offer](#792a20o-the-receiving-vault-tags-a-served-offer)).

### 792a20i1 UAT Scenarios

```gherkin
Scenario: 792a20i1a Placement is ignored
  Given a served learn with target "12" and position "child"
  When the offer is written
  Then the pending note receives a new top-level Luhmann ID

Scenario: 792a20i1b Wire tags are dropped
  Given a served learn that carries tags, first as a new offer and then as a same-origin rewrite
  When the pending note is read
  Then it carries none of those tags
```

## 792a20j A caller cannot set link, identity or registration fields

Requirement: [792a20j A caller cannot set link, identity or registration fields](#792a20j-a-caller-cannot-set-link-identity-or-registration-fields) A served `learn` MUST NOT let its request body populate a note's `parent`, `aliases`, `xid`,
`offer.prior_keys`, `skill_hash`, `skill_key` or `skill_source`, whatever the spelling of the key in the body. The only
exchange fields a caller MAY set are `offer.origin`, `offer.key`, `offer.for` and `offer.path`, and they MUST land only
on pending notes.

Rationale: a remote client that could set a link or an exchange ID could forge a relation to a note it does not own,
or a registration identity it was never given. An offer can only create or update a pending note, never a live one, and
curation reviews every one, so the worst a hostile caller does is queue work.

### 792a20j1 UAT Scenarios

```gherkin
Scenario: 792a20j1a Remote-set link and registration fields are ignored
  Given a served learn body that includes parent, aliases, xid, skillHash or SkillKey keys in camelCase or PascalCase
  When the pending note is written
  Then it carries none of those fields

Scenario: 792a20j1b A caller cannot forge a link
  Given a served learn body that carries parent, aliases or xid values
  When the pending note is written
  Then it carries none of those values, and its xid is one the server minted
```

## 792a20k A served show returns a note, and a raw envelope on request

Requirement: [792a20k A served show returns a note, and a raw envelope on request](#792a20k-a-served-show-returns-a-note-and-a-raw-envelope-on-request) The served `show` MUST, without `raw`, return exactly the local `engram show` output, including the
exchange-hash header line for a note that carries an `xid`. With `raw=1` it MUST resolve `note` against basenames and
then against `aliases` and MUST return a JSON object carrying `vault_id`, `basename` (the note's current basename),
`content` (the note file's bytes, unmodified, with no `red_flags` preview cap and no appended links section) and
`exchange_hash`. It MUST answer 404 with an error body for a missing note, and an empty `note` MUST be a 400. A pending
note MUST be treated as missing on the raw route, whether named by basename or by alias.

Rationale: the raw envelope is JSON with a `vault_id`, so it cannot be mistaken for the rendered output of an older
parent. A pending note is an unvetted offer and not the vault's note, so it is never pulled down as one.

### 792a20k1 UAT Scenarios

```gherkin
Scenario: 792a20k1a Show without raw matches the local output
  Given a note that carries an xid
  When it is shown locally and as GET /show without raw
  Then the served response is the local output, exchange-hash header line included

Scenario: 792a20k1b Raw show is exact
  Given an existing live note
  When GET /show?note=<basename>&raw=1 is served
  Then the response is a JSON envelope whose content is byte-identical to the note file

Scenario: 792a20k1c Raw show follows a rename
  Given a note renamed since, which lists its old basename in aliases
  When GET /show?note=<old basename>&raw=1 is served
  Then the envelope carries the note's current basename and content

Scenario: 792a20k1d A missing note is not found
  Given no note of that name
  When GET /show?note=<missing>&raw=1 is served
  Then the response is 404 with an error body

Scenario: 792a20k1e A pending note is not served raw
  Given a pending note
  When GET /show?note=<its basename or an alias>&raw=1 is served
  Then the response is 404 with an error body

Scenario: 792a20k1f A missing ref is a bad request
  Given GET /show with no note parameter
  When the server handles it
  Then the response is 400
```

## 792a20l A served activate takes note names only

Requirement: [792a20l A served activate takes note names only](#792a20l-a-served-activate-takes-note-names-only) The served `activate` MUST resolve each ref only against the vault's listed note names, with or
without `.md`, and never join a ref into a filesystem path. A request with any ref that is an absolute path, contains a
path separator (`/` or `\`) or contains `..` MUST be answered 400 and nothing MUST be read or written. A listed note
name MUST have its sidecar's last-used date bumped, committed directly with no pending marker and no curation. The
response MUST be 200 `{"status":"ok"}` when every ref was activated, 404 with the per-ref result when none was found, 200
with the per-ref result when some were, and 500 only when a ref failed while being activated. The local `engram
activate` MUST keep accepting absolute and vault-relative paths.

Rationale: a ref joined into a path would let an unauthenticated caller reach outside the vault. Activation never
changes a note's content, so there is no new claim for curation to judge, and a 5xx therefore only ever means a
server-side failure.

### 792a20l1 UAT Scenarios

```gherkin
Scenario: 792a20l1a A path ref is rejected
  Given POST /activate naming /etc/x, ../x.md or an absolute path to a note outside the vault
  When the server handles it
  Then the response is 400 and no sidecar inside or outside the vault is changed

Scenario: 792a20l1b A listed note name is activated
  Given POST /activate naming a vault note's basename, with or without .md
  When the server handles it
  Then that note's sidecar last-used date is bumped and the response is 200 with status ok

Scenario: 792a20l1c No ref found is a 404
  Given POST /activate naming only notes the vault does not hold
  When the server handles it
  Then the response is 404 and its body lists each ref as not found

Scenario: 792a20l1d A partial result is a 200
  Given POST /activate naming one note the vault holds and one it does not
  When the server handles it
  Then the response is 200 with the per-ref result

Scenario: 792a20l1e A failed activation is a 500
  Given POST /activate naming a note whose sidecar cannot be written
  When the server handles it
  Then the response is 500

Scenario: 792a20l1f Local activate keeps paths
  Given the local engram activate with an absolute path to a note
  When it runs
  Then the note is activated
```

## 792a20m A served query takes the message and can return dedupe keys

Requirement: [792a20m A served query takes the message and can return dedupe keys](#792a20m-a-served-query-takes-the-message-and-can-return-dedupe-keys) The served `query` MUST accept the client's `--text` value as the `text` query-string parameter,
and MUST truncate it to at most 2048 bytes on a UTF-8 rune boundary, never splitting a multi-byte character, before it
runs the query, silently, with no error and no warning. A `text` of 2048 bytes or fewer MUST be passed through unchanged
(quotes, newlines, `&` and non-ASCII characters included) and produce the payload the local path produces. With
`dedupe-keys=1` the payload MUST carry a top-level `vault_id`, and each note item MUST carry its `exchange_hash` and,
when non-empty, its `aliases`, and a note whose keys cannot be computed MUST lose only its own keys, with a warning.
Without the parameter the payload MUST be unchanged.

Rationale: trigger cues occur early in a message, so a silent cut at 2 KB loses none that matter, and the cap bounds the
query string a caller can make the server hold. The keys are what a child dedupes a merged result by.

### 792a20m1 UAT Scenarios

```gherkin
Scenario: 792a20m1a Short text round-trips unchanged
  Given a client sends text of 2048 bytes or fewer containing quotes, newlines, & and non-ASCII characters
  When the server queries
  Then it queries with the identical string and produces the same payload as the local path

Scenario: 792a20m1b Long text is truncated on a rune boundary
  Given a client sends text longer than 2048 bytes and byte 2048 falls inside a multi-byte character
  When the server queries
  Then it queries with the text cut back to the last complete character at or before byte 2048, and returns no error

Scenario: 792a20m1c Keys are present only on request
  Given GET /query served once with dedupe-keys=1 and once without
  When both responses are read
  Then only the first carries vault_id and a per-note exchange_hash, and the second is byte-identical to the local payload

Scenario: 792a20m1d A note whose keys cannot be computed loses only its own
  Given a dedupe-keys query whose payload has a note that cannot be read
  When the server adds keys
  Then the other notes keep theirs, the query succeeds and the server warns once for the skipped note
```

## 792a20n Request and response bodies are capped

Requirement: [792a20n Request and response bodies are capped](#792a20n-request-and-response-bodies-are-capped) `engram serve` MUST read at most 4 MiB of a request body and MUST answer 413 on every route, writing
nothing, when the body is over that cap, and MUST answer 400 for any other failure to read a body in full. A child MUST
accept at most 16 MiB of a parent's response, and a longer response MUST be an error and never a truncated payload.

Rationale: an unauthenticated caller must not be able to make the server buffer without bound, and a malicious parent
must not exhaust a child's memory. An offer or an activate body is a few KB, so the caps leave ample room. The exchange
that uses a parent's response is parent-sync's ([parent-sync](./parent-sync.md)).

### 792a20n1 UAT Scenarios

```gherkin
Scenario: 792a20n1a An oversized request is refused
  Given a served request whose body exceeds 4 MiB
  When the server handles it
  Then the response is 413 and the vault is unchanged

Scenario: 792a20n1b Another unreadable body is a bad request
  Given a served request whose body cannot be read in full for a reason other than size, such as a client disconnect
  When the server handles it
  Then the response is 400 and nothing is written

Scenario: 792a20n1c An oversized parent response is refused
  Given a parent's response body that exceeds 16 MiB
  When the child reads it
  Then the request fails as the parent's failure and nothing from it is stored
```

## 792a20o The receiving vault tags a served offer

Requirement: [792a20o The receiving vault tags a served offer](#792a20o-the-receiving-vault-tags-a-served-offer) After a served `learn` writes or rewrites a pending offer, the served vault MUST assign it
`vocab/<term>` tags from the vault's own vocabulary as it does for any note it writes, and the offer's own tags MUST NOT
be kept ([792a20i Placement and tags are the server's](#792a20i-placement-and-tags-are-the-servers)).

Rationale: tags are the vocabulary of the vault that holds the note ([controlled-vocabulary](./controlled-vocabulary.md)), so a child's tags would mean nothing here.

### 792a20o1 UAT Scenarios

```gherkin
Scenario: 792a20o1a A served offer is tagged by the receiving vault
  Given a vault with a vocabulary and a served learn that sends no tags
  When the pending offer is written
  Then it carries the vocab/ tags that the vault's vocabulary assigns to it

Scenario: 792a20o1b A rewrite is re-tagged
  Given a pending offer rewritten in place by a same-origin served learn with changed content
  When the rewrite is written
  Then its vocab/ tags are the ones the vault's vocabulary assigns to the new content
```
