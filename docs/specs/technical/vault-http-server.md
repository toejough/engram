# Vault HTTP Server - Technical Spec

## 792b35a Orientation

- Ownership: Repository-owned dependency-boundary interaction (the `net/http` server, and the bounded client that
  reaches a parent).
- Responsibility: answers a child's HTTP requests for `query`, `show`, `activate` and `learn` against one fixed vault.
  It reduces each request to primitives, runs the command's own implementation under the command's own locks, maps the
  outcome to a status and a JSON or text body, refuses requests that are too large, malformed, looping or aimed
  outside the vault, and writes a served `learn` as a pending offer under the declared identity, answering with a
  receipt.
- Current realization (informational): `internal/cli/serve.go`, `serve_learn.go` and `serve_exchange_read.go`, with the
  `net/http` edge in `cmd/engram/serve.go` (the mux, the route registration, the listener and the bounded fetch client).
- Qualification: responsibility, serving the four commands over HTTP is stated without naming a function or a handler;
  ownership, it owns the routes, statuses, size caps and the served-learn lookup, while the commands own their results
  and [exchange-versioning](./exchange-versioning.md) and [pending-offer-bookkeeping](./pending-offer-bookkeeping.md) own the hash and the marker it calls; independent verification,
  every handler runs over an in-memory request and a temporary vault, and a real mux serves two round trips over TCP;
  stability, the route table and receipt are the contract children depend on; necessity, a child cannot offer or pull
  without it.
- Supported capabilities: [vault-serving](../product/vault-serving.md).

Rationale: a child needs to reach its parent without a second implementation of any command, so each handler calls the
command's own code, sharing its locks. The server takes its vault, vault name and chunk index once at startup, so no
request can aim it elsewhere. A served write is an offer to be judged, never a live note, and the caller declares who it
is, because the server has no authentication of its own and its trust boundary is network reachability. The net/http
types stay in `cmd/engram`, so every other part of the server runs over primitives.

## 792b35c Behavior Requirements

### 792b35c1 The route table

Requirement: [792b35c1 The route table](#792b35c1-the-route-table) The vault-http-server MUST serve exactly `GET /query`, `GET /show`, `POST /activate` and `POST /learn`,
and MUST answer a request for any other path with 404 and a request for a served path with another method with 405,
performing no vault operation for either. It MUST NOT route `amend`, `query-chunks`, `show-chunk`, `ingest`, `vocab
refit`, `prune`, `check`, `update` or `resituate`.

Supports: [792a20a1a A served command is reachable](../product/vault-serving.md#792a20a1-uat-scenarios), [792a20a1b A host-only command has no route](../product/vault-serving.md#792a20a1-uat-scenarios), [792a20a1c A served command refuses the wrong method](../product/vault-serving.md#792a20a1-uat-scenarios).

#### 792b35c1a Scenarios

```gherkin
Scenario: 792b35c1a1 The route table is the four routes
  Given the served routes
  When their methods and patterns are listed
  Then they are GET /query, GET /show, POST /activate and POST /learn and nothing else

Scenario: 792b35c1a2 An unrouted path or method is refused
  Given a running server
  When a request names /amend or /query-chunks, or sends GET to /learn
  Then the answer is 404 for the path and 405 for the method, and no vault operation runs
```

### 792b35c2 Starting and stopping

Requirement: [792b35c2 Starting and stopping](#792b35c2-starting-and-stopping) The vault-http-server MUST register every route once on a new mux and then listen on the given
address, MUST wrap a listening error as `serve: <cause>` when the context was not canceled, and MUST return no error
when the context was canceled, because that error is the expected close at shutdown. It MUST stamp a missing vault ID
and check the vault's location before it serves, warning and serving anyway when either fails ([vault-identity-registry](./vault-identity-registry.md)).

Supports: [792a20b1b A given address is bound](../product/vault-serving.md#792a20b1-uat-scenarios).

Rationale: a corrupt or empty ID file would crash-loop under a service manager if serving were refused, and a location
false positive would turn into a host outage.

#### 792b35c2a Scenarios

```gherkin
Scenario: 792b35c2a1 Every route is registered and a listening failure is wrapped
  Given a server whose listener fails while the context is live
  When it starts
  Then each of the four routes was registered exactly once on the mux, the listener got the given address and the error names the cause

Scenario: 792b35c2a2 A close after cancellation is not an error
  Given the same listener failure and a canceled context
  When it starts
  Then it returns no error

Scenario: 792b35c2a3 A corrupt ID or a mismatched location warns and serves
  Given a vault whose ID file is corrupt, or whose location record is for another path
  When the server starts
  Then it warns, naming the remedy, and serves anyway
```

### 792b35c3 Request bodies

Requirement: [792b35c3 Request bodies](#792b35c3-request-bodies) The vault-http-server MUST read at most 4 MiB of a request body, MUST answer 413 on every route,
writing nothing, when the body itself is over that cap, and MUST answer 400 on every route, writing nothing, for any other
failure to read the body in full. No route MUST act on a body it did not read in full. The edge MUST bound the time it
waits to read a request's headers to 10 seconds.

Supports: [792a20n1a An oversized request is refused](../product/vault-serving.md#792a20n1-uat-scenarios), [792a20n1b Another unreadable body is a bad request](../product/vault-serving.md#792a20n1-uat-scenarios).

Rationale: the server is unauthenticated, so what a caller can make it buffer is bounded. An offer or an activate body is a
few KB, so the cap leaves ample room, and a client disconnect is not an oversized body.

#### 792b35c3a Scenarios

```gherkin
Scenario: 792b35c3a1 An oversized body is 413 on every route
  Given a body marked as over the cap, sent to learn, activate, query and show
  When the server handles each
  Then each answers 413 and the vault is unchanged

Scenario: 792b35c3a2 Another read failure is 400 on every route
  Given a body that failed to read for another reason, sent to each route
  When the server handles each
  Then each answers 400 and the vault is unchanged

Scenario: 792b35c3a3 The request cap is a few MiB
  Given the request cap
  When it is read
  Then it is at least 1 MiB and at most 16 MiB

Scenario: 792b35c3a4 A slow header read is cut off
  Given a client that sends request headers slowly
  When the wait exceeds 10 seconds
  Then the server closes the connection
```

### 792b35c4 The query route

Requirement: [792b35c4 The query route](#792b35c4-the-query-route) The vault-http-server MUST answer `GET /query` with the payload the local `engram query` prints
for the same arguments, built from the `phrase` (repeatable), `limit`, `project`, `text`, `content-budget`, `recent-fill`,
`lazy-chunks` and `timings` parameters against the server's own vault and chunk index, MUST cut `text` to at most 2048
bytes on a UTF-8 rune boundary without an error or a warning, and MUST answer a failing query with 500 and an error body.
A pending offer MUST NOT appear in the items, while the payload's `pending_offers` flag and hint MUST reflect the
vault.

Supports: [792a20c1a A served query matches the local query](../product/vault-serving.md#792a20c1-uat-scenarios), [792a20c1d A request cannot redirect the server](../product/vault-serving.md#792a20c1-uat-scenarios), [792a20d1b The offered note is pending, not live](../product/vault-serving.md#792a20d1-uat-scenarios), [792a20m1a Short text round-trips unchanged](../product/vault-serving.md#792a20m1-uat-scenarios), [792a20m1b Long text is truncated on a rune boundary](../product/vault-serving.md#792a20m1-uat-scenarios).

#### 792b35c4a Scenarios

```gherkin
Scenario: 792b35c4a1 Text is cut at 2 KB
  Given a runbook whose trigger is /zz-boundary and a text that ends the cue exactly at byte 2048, one that straddles it and one with the cue at the start of a long text
  When each is queried
  Then the first and the third hit the trigger and the second does not, with no error

Scenario: 792b35c4a2 A pending offer is excluded and the flag is set
  Given a normal note and a pending offer both about coverage, and an embedder with model test-model@4
  When GET /query runs for coverage
  Then the payload names model_id test-model@4, carries pending_offers true and the curate hint, lists the normal note and not the pending one

Scenario: 792b35c4a3 The parameters reach the query
  Given a request with two phrases, a limit, a project, a content budget, a recent fill, lazy-chunks and timings
  When the server handles it
  Then the query ran with those arguments against the server's own vault

Scenario: 792b35c4a4 A failing query is a 500
  Given a query that fails
  When the server handles it
  Then the answer is 500 with an error body
```

### 792b35c5 Dedupe keys

Requirement: [792b35c5 Dedupe keys](#792b35c5-dedupe-keys) The vault-http-server MUST, only when `dedupe-keys=1` is given, add the served vault's ID as a
top-level `vault_id` and, to each note item, its `exchange_hash` and its non-empty `aliases`, adding nothing to chunk
items, and MUST leave a note whose keys cannot be computed without them and log a warning naming it, so that the query
still succeeds. Without the parameter the payload MUST be byte-identical to the local payload.

Supports: [792a20m1c Keys are present only on request](../product/vault-serving.md#792a20m1-uat-scenarios), [792a20m1d A note whose keys cannot be computed loses only its own](../product/vault-serving.md#792a20m1-uat-scenarios).

#### 792b35c5a Scenarios

```gherkin
Scenario: 792b35c5a1 Keys are present only on request
  Given two notes, one with an alias, and a query served with dedupe-keys=1 and without
  When both responses are read
  Then the first carries vault_id and each note's exchange_hash, and the aliased note's aliases; the second equals the local payload and holds none of them

Scenario: 792b35c5a2 A note whose hash cannot be computed is skipped
  Given a good note and a note whose frontmatter cannot be hashed
  When a dedupe-keys query is served
  Then the good note has an xh1: hash, the odd note has none, the response is 200 and stderr names the odd note
```

### 792b35c6 The show route

Requirement: [792b35c6 The show route](#792b35c6-the-show-route) The vault-http-server MUST answer `GET /show` without `raw` with exactly the local `engram show`
output, exchange-hash header included. With `raw=1` it MUST resolve `note` against basenames, with or without `.md`,
and then against the `aliases` of live notes, and answer 200 with `vault_id`, `basename` (current), `content` (the file's
bytes verbatim, with no preview cap and no links section) and `exchange_hash`. It MUST answer 404 with an error body for
a note that is not found, raw or not, and for a pending note named by basename or by alias, 400 for an empty `note`, and
500, never 404 or a partial envelope, for a listed note that cannot be read or whose frontmatter cannot be parsed for its
hash.

Supports: [792a20c1b A served show matches the local show](../product/vault-serving.md#792a20c1-uat-scenarios), [792a20k1a Show without raw matches the local output](../product/vault-serving.md#792a20k1-uat-scenarios), [792a20k1b Raw show is exact](../product/vault-serving.md#792a20k1-uat-scenarios), [792a20k1c Raw show follows a rename](../product/vault-serving.md#792a20k1-uat-scenarios), [792a20k1d A missing note is not found](../product/vault-serving.md#792a20k1-uat-scenarios), [792a20k1e A pending note is not served raw](../product/vault-serving.md#792a20k1-uat-scenarios), [792a20k1f A missing ref is a bad request](../product/vault-serving.md#792a20k1-uat-scenarios).

Rationale: the envelope is JSON with a `vault_id` and so cannot be mistaken for an old parent's rendered output. A ref is
only matched against listed names and never joined into a path, so it cannot reach outside the vault. A pending note is an
unvetted offer, so it is never pulled down as the vault's note.

#### 792b35c6a Scenarios

```gherkin
Scenario: 792b35c6a1 Show matches the local output
  Given a vault holding a note
  When GET /show?note=1 is served
  Then the response is 200 and has the note's content as the local show prints it

Scenario: 792b35c6a2 The exchange-hash header matches the local one
  Given a pending offer that carries xid
  When it is shown locally and as GET /show without raw
  Then the served body equals the local output and begins with "# exchange_hash: xh1:"

Scenario: 792b35c6a3 Error statuses
  Given an empty ref, a listed note that cannot be read, and a note whose frontmatter does not parse, each shown rendered or raw
  When the server handles each
  Then an empty ref is 400 and the unreadable and unparseable notes are 500

Scenario: 792b35c6a4 A missing note is 404
  Given no note of the given name
  When it is served rendered and raw
  Then both answer 404 with an error body that says not found

Scenario: 792b35c6a5 The raw envelope is exact
  Given a runbook with ten red flags, a wikilink and an alias, referred to by basename, by basename with .md and by its alias
  When GET /show?raw=1 is served for each
  Then each envelope carries the vault ID, the current basename, content byte-identical to the file and the file's exchange hash

Scenario: 792b35c6a6 A pending note is not served raw
  Given a pending offer with an alias and a live note
  When the pending offer is requested raw by basename and by alias, and the live note by basename
  Then the pending requests are 404 and the live one is 200
```

### 792b35c7 The activate route

Requirement: [792b35c7 The activate route](#792b35c7-the-activate-route) The vault-http-server MUST answer `POST /activate` by resolving each ref in `notes` only against
the vault's listed note names, with or without `.md` and case-sensitively, never joining a ref into a path, MUST list the
vault directory once per request, and MUST answer 400, reading and writing nothing, when any ref is an absolute path,
contains `/` or `\` or contains `..`. It MUST bump the `LastUsed` of each activated note's sidecar in the same request
with no pending marker and no curation. It MUST answer 200 `{"status":"ok"}` when every ref was activated, 404 with the
per-ref result when none was found, 200 with the per-ref result when some were, and 500 only when a ref failed while
being activated.

Supports: [792a20l1a A path ref is rejected](../product/vault-serving.md#792a20l1-uat-scenarios), [792a20l1b A listed note name is activated](../product/vault-serving.md#792a20l1-uat-scenarios), [792a20l1c No ref found is a 404](../product/vault-serving.md#792a20l1-uat-scenarios), [792a20l1d A partial result is a 200](../product/vault-serving.md#792a20l1-uat-scenarios), [792a20l1e A failed activation is a 500](../product/vault-serving.md#792a20l1-uat-scenarios).

Rationale: the one listing keeps a request with many short refs from costing a directory read per ref while the vault
lock is held, and a 5xx therefore only ever means a server-side failure.

#### 792b35c7a Scenarios

```gherkin
Scenario: 792b35c7a1 Activate commits directly
  Given a note with a sidecar
  When POST /activate names it
  Then the response is 200 and the sidecar's last-used date is today

Scenario: 792b35c7a2 One listing serves many refs
  Given three notes and a request naming all three
  When the server handles it
  Then the vault directory was listed exactly once

Scenario: 792b35c7a3 Refs outside the vault are rejected
  Given a request naming a valid note and an absolute path, an /etc path, a dot-dot path, a bare "..", a subdirectory path, a backslash path or a name containing ".."
  When the server handles it
  Then the answer is 400 with an error body, no sidecar inside or outside the vault is changed and not even the valid note is bumped

Scenario: 792b35c7a4 Only listed names resolve
  Given a vault note 1.2026-01-01.a-note.md
  When the refs 1.2026-01-01.a-note.md, 1.2026-01-01.a-note, 1.2026-01-01.a-note.MD and 1 are served
  Then the first two answer 200 and the last two 404

Scenario: 792b35c7a5 The status follows the outcome
  Given a request naming only a missing note, one naming an existing and a missing note, and one naming a note whose sidecar is unreadable
  When the server handles each
  Then they answer 404, 200 and 500, and the first two bodies list each ref's result
```

### 792b35c8 Decoding a served learn

Requirement: [792b35c8 Decoding a served learn](#792b35c8-decoding-a-served-learn) The vault-http-server MUST decode a `POST /learn` body as a learn request whose only JSON keys are
`action`, `behavior`, `body`, `chunkSources`, `doneWhen`, `impact`, `issue`, `object`, `offer`, `pending`, `position`,
`predicate`, `project`, `redFlags`, `repo`, `situation`, `slug`, `source`, `subject`, `supersedes`, `tags`, `target`,
`tier`, `triggers`, `type`, `user`, `vault` and `vaultName`, and MUST NOT populate `xid`, `parent`, `aliases`,
`offer.prior_keys`, `skill_hash`, `skill_key` or `skill_source` from the body under any spelling. It MUST answer 400 when
the declared `user` is empty, MUST use the server's own vault whatever the body says, MUST take `vaultName` from the body
and otherwise from the server, MUST take `user` and `repo` from the body and not detect them itself, MUST drop `tags`,
MUST write the note as pending with the offer's own placement ignored ([792b35c14 A new pending offer](#792b35c14-a-new-pending-offer)), and MUST answer with the receipt and
never with note content.

Supports: [792a20c1d A request cannot redirect the server](../product/vault-serving.md#792a20c1-uat-scenarios), [792a20d1a The receipt names the pending note](../product/vault-serving.md#792a20d1-uat-scenarios), [792a20d1c The receipt carries no content](../product/vault-serving.md#792a20d1-uat-scenarios), [792a20e1a The declared identity is stamped as given](../product/vault-serving.md#792a20e1-uat-scenarios), [792a20e1b An empty identity is rejected](../product/vault-serving.md#792a20e1-uat-scenarios), [792a20e1c No edge header is needed](../product/vault-serving.md#792a20e1-uat-scenarios), [792a20f1a repo reflects the caller's detection](../product/vault-serving.md#792a20f1-uat-scenarios), [792a20f1b vault falls back to the server's name](../product/vault-serving.md#792a20f1-uat-scenarios), [792a20i1b Wire tags are dropped](../product/vault-serving.md#792a20i1-uat-scenarios), [792a20j1a Remote-set link and registration fields are ignored](../product/vault-serving.md#792a20j1-uat-scenarios),
[792a20j1b A caller cannot forge a link](../product/vault-serving.md#792a20j1-uat-scenarios).

Rationale: the body is a remote client's, so every field a caller must not control is not decodable at all, and a new
field on the learn request that is not marked non-decodable fails the key-set test. A served learn must claim some
identity, and the server trusts what is declared because it has nothing else to consult.

#### 792b35c8a Scenarios

```gherkin
Scenario: 792b35c8a1 An empty declared identity is rejected
  Given a body with no user
  When it is posted to learn
  Then the answer is 400 and no note is written

Scenario: 792b35c8a2 The declared identity, repo and pending marker are stamped
  Given a fact body declaring user declared-user@example.com and repo git@github.com:example/remote.git, with no vault name
  When it is posted to learn
  Then the note has user, repo and pending true as declared, vault personal, and the response is an offer-received receipt that holds none of the note's content

Scenario: 792b35c8a3 Skill and exchange fields in the body are ignored
  Given a runbook body carrying xid, parent, aliases and skill_hash, skill_key and skill_source keys in camelCase and PascalCase
  When it is posted to learn
  Then the written runbook holds none of those fields or their forged values

Scenario: 792b35c8a4 Wire tags are dropped
  Given a body with tags, posted first as a new offer and then as a same-origin rewrite
  When the notes are read
  Then neither carries the tags

Scenario: 792b35c8a5 The wire keys are the known set
  Given the learn request's JSON tags
  When they are listed
  Then they are the pre-exchange keys plus offer and nothing else

Scenario: 792b35c8a6 The offer decodes and the exchange and skill fields do not
  Given a body with xid, parent, aliases and skill fields in both spellings and an offer with origin, key, for and path
  When it is decoded
  Then the exchange and skill fields are empty and the offer holds origin, key, for and path

Scenario: 792b35c8a7 The server's vault wins
  Given a body that names another vault
  When it is posted to learn
  Then the note is written in the server's vault

Scenario: 792b35c8a8 A served offer is tagged by the vault's vocabulary
  Given a vault with a vocabulary and a body with no tags
  When it is posted to learn
  Then the pending note carries the vocab/ tags that the vault assigns
```

### 792b35c9 Refusing an invalid offer

Requirement: [792b35c9 Refusing an invalid offer](#792b35c9-refusing-an-invalid-offer) The vault-http-server MUST, before taking the vault lock, answer 400 and write nothing when
`offer.path` has more than 16 entries or an entry that is not exactly 32 lowercase hexadecimal characters, when
`offer.origin` is not two such identifiers joined by `:`, when `offer.key` or `offer.path` is present without
`offer.origin`, or when the note content fails validation, and MUST answer 409 and write nothing when `offer.path`
already holds its own vault ID, with a body of `error`, `reason` `loop` and `vault_id`.

Supports: [792a20h1a A key without an origin is rejected](../product/vault-serving.md#792a20h1-uat-scenarios), [792a20h1b A malformed or oversized path is rejected](../product/vault-serving.md#792a20h1-uat-scenarios), [792a20h1c A malformed origin is rejected](../product/vault-serving.md#792a20h1-uat-scenarios), [792a20h1d A cycle is refused](../product/vault-serving.md#792a20h1-uat-scenarios), [792a20h1e Invalid note content is a client error](../product/vault-serving.md#792a20h1-uat-scenarios).

#### 792b35c9a Scenarios

```gherkin
Scenario: 792b35c9a1 A malformed offer is a 400 and a cycle is a 409
  Given an offer.path of 17 entries, a path entry that is not 32 hex characters, a malformed origin, and a path holding the server's own vault ID
  When each is posted to learn
  Then the first three answer 400 and the last answers 409, and nothing is written in any case

Scenario: 792b35c9a2 The path is valid exactly when it is short and hex
  Given any offer.path
  When it is posted to learn
  Then the answer is 400 exactly when the path has more than 16 entries or an entry that is not 32 lowercase hex characters, and nothing is written for a 400

Scenario: 792b35c9a3 The loop refusal reports the vault ID
  Given an offer whose path holds the server's own vault ID
  When it is posted to learn
  Then the 409 body carries the server's vault_id and reason loop

Scenario: 792b35c9a4 A key without an origin is a 400
  Given an offer that carries offer.key and no offer.origin
  When it is posted to learn
  Then the answer is 400 and nothing is written

Scenario: 792b35c9a5 Invalid note content is a 400
  Given a body whose note type is unknown
  When it is posted to learn
  Then the answer is 400 before any lock is taken and nothing is written
```

### 792b35c10 A recorded key is a retry

Requirement: [792b35c10 A recorded key is a retry](#792b35c10-a-recorded-key-is-a-retry) The vault-http-server MUST, when any note, live or pending, records a non-empty `offer.key` as its
current key or among its `offer.prior_keys`, write nothing and return that note's current receipt, and MUST check this
before every other case, so that a late retry of an accepted or superseded key never rewrites a newer same-origin
pending offer back to older content.

Supports: [792a20g1b A duplicate key writes nothing](../product/vault-serving.md#792a20g1-uat-scenarios), [792a20g1c A retry after acceptance writes nothing](../product/vault-serving.md#792a20g1-uat-scenarios), [792a20g1d A late retry never reverts a newer amend](../product/vault-serving.md#792a20g1-uat-scenarios), [792a20g1e A late retry of a superseded key never reverts the note](../product/vault-serving.md#792a20g1-uat-scenarios).

#### 792b35c10a Scenarios

```gherkin
Scenario: 792b35c10a1 A duplicate key writes nothing
  Given a pending note with offer.origin O and key K
  When a served learn arrives with O and K
  Then the response is that note's receipt and no vault file changes

Scenario: 792b35c10a2 A retry after acceptance writes nothing
  Given an accepted live note that still carries offer.origin O and key K
  When a served learn arrives with O and K
  Then the response is the live note's receipt with pending false and no pending note is created

Scenario: 792b35c10a3 A late retry never reverts a newer amend
  Given a live note with origin O and key K1, a newer pending amend from O with key K2, and a retry of O with K1
  When it is posted
  Then the response is the live note's receipt and the pending amend is unchanged

Scenario: 792b35c10a4 A late retry of a superseded key never reverts the note
  Given a pending note rewritten from key K1 to K2, with K1 in its prior keys, and a late retry of K1, before and after the note is accepted
  When it is posted
  Then the response is the note's current receipt, the note still holds the K2 content and no file changes
```

### 792b35c11 A same-origin pending offer is rewritten in place

Requirement: [792b35c11 A same-origin pending offer is rewritten in place](#792b35c11-a-same-origin-pending-offer-is-rewritten-in-place) The vault-http-server MUST, when no note records the key and a pending note carries the same
`offer.origin`, rewrite that note in place under its basename, still pending and keeping its `xid`, links, aliases, origin
and `offer.for`, with the new content, key and path, the superseded key appended to `offer.prior_keys` (the most recent
8 kept, oldest dropped), the identity stamped from the new request, vocabulary tags assigned, and its sidecar re-embedded
unless the exchange hash is provably unchanged. Same-origin offers racing each other MUST leave exactly one pending note.

Supports: [792a20f1c An in-place rewrite is stamped from its own request](../product/vault-serving.md#792a20f1-uat-scenarios), [792a20g1a An amend before curation updates the pending offer](../product/vault-serving.md#792a20g1-uat-scenarios), [792a20g1k Prior keys are bounded](../product/vault-serving.md#792a20g1-uat-scenarios), [792a20o1b A rewrite is re-tagged](../product/vault-serving.md#792a20o1-uat-scenarios).

#### 792b35c11a Scenarios

```gherkin
Scenario: 792b35c11a1 An amend before curation updates the pending offer
  Given a pending offer N1 and a second offer from the same origin with a new key and new content
  When it is posted
  Then N1 keeps its basename and gains the new content, key and a re-embedded sidecar, and exactly one pending note exists

Scenario: 792b35c11a2 Prior keys are bounded
  Given a pending offer rewritten by more than eight same-origin offers
  When its prior keys are read
  Then they are the eight most recent superseded keys, and a body that sets offer.prior_keys has no effect

Scenario: 792b35c11a3 Racing same-origin offers leave one pending note
  Given several same-origin offers posted at once
  When they have all completed
  Then exactly one pending note exists
```

### 792b35c12 A same-origin live offer makes a new pending note

Requirement: [792b35c12 A same-origin live offer makes a new pending note](#792b35c12-a-same-origin-live-offer-makes-a-new-pending-note) The vault-http-server MUST, when no note records the key, no pending note carries the origin and a
live note carries the same `offer.origin`, write a new pending note whose `offer.for` names that live note, and report
`for` in the receipt.

Supports: [792a20g1f A new key from an accepted origin makes a new pending note](../product/vault-serving.md#792a20g1-uat-scenarios).

#### 792b35c12a Scenarios

```gherkin
Scenario: 792b35c12a1 A new key from an accepted origin makes a new pending note
  Given an accepted live note with origin O and a served learn from O with a new key
  When it is posted
  Then a new pending note is written with offer.for naming the live note and the receipt carries for

Scenario: 792b35c12a2 The live note is untouched
  Given the same request
  When it is handled
  Then the live note is byte-identical to before
```

### 792b35c13 Resolving where an offer points

Requirement: [792b35c13 Resolving where an offer points](#792b35c13-resolving-where-an-offer-points) The vault-http-server MUST, when no earlier case applies, resolve `offer.for` against live notes'
basenames, then live notes' `aliases`, then pending notes' basenames, among fact, feedback and runbook notes that parse
only. A resolved live note MUST be left unchanged and live, with a new pending note written naming it and the receipt
carrying `for`. A resolved pending note of another origin MUST be left unmodified, with a new pending note written naming
it and no `for` in the receipt. A target that does not resolve, including a vocabulary note, a file with no frontmatter
or a note whose frontmatter does not parse, MUST be dropped, with the offer written as a new pending note.

Supports: [792a20g1g No cross-origin overwrite](../product/vault-serving.md#792a20g1-uat-scenarios), [792a20g1h An amend-offer names a live target](../product/vault-serving.md#792a20g1-uat-scenarios), [792a20g1i A renamed target resolves through its aliases](../product/vault-serving.md#792a20g1-uat-scenarios), [792a20g1j An unresolvable target is dropped](../product/vault-serving.md#792a20g1-uat-scenarios).

#### 792b35c13a Scenarios

```gherkin
Scenario: 792b35c13a1 A live target is left unchanged and reported
  Given a live note and an offer whose for names it
  When it is posted
  Then the new pending note carries offer.for, the live note is byte-identical, and the receipt carries for

Scenario: 792b35c13a2 A renamed target resolves through aliases
  Given a live note listing a basename in its aliases and an offer whose for names that basename
  When it is posted
  Then the receipt carries the live note's current basename as for

Scenario: 792b35c13a3 There is no cross-origin overwrite
  Given a pending note from origin A and an offer from origin B whose for names it
  When it is posted
  Then A's note is unchanged, a new pending note names it in offer.for and the receipt has no for

Scenario: 792b35c13a4 An unresolvable target is dropped
  Given an offer whose for names no note, alias or pending note
  When it is posted
  Then the new pending note has no offer.for and the response has no for

Scenario: 792b35c13a5 Only exchange notes resolve
  Given offer.for naming a vocabulary note, a file with no frontmatter and a note whose frontmatter does not parse
  When each is posted
  Then each target is dropped
```

### 792b35c14 A new pending offer

Requirement: [792b35c14 A new pending offer](#792b35c14-a-new-pending-offer) The vault-http-server MUST write a new pending note at top level, whatever `target` or `position` the
caller sent, with `pending: true`, a fresh `xid` it mints, an `offer` record of the offer's `origin`, `key`, `for` and
`path`, the identity declared by the caller, vocabulary tags assigned by the vault, and an embedded sidecar, and MUST
return a receipt with the note's Luhmann ID and basename, `pending` true, its vault's ID, the stored exchange hash and
`for` only when the target is live.

Supports: [792a20d1a The receipt names the pending note](../product/vault-serving.md#792a20d1-uat-scenarios), [792a20i1a Placement is ignored](../product/vault-serving.md#792a20i1-uat-scenarios), [792a20j1b A caller cannot forge a link](../product/vault-serving.md#792a20j1-uat-scenarios), [792a20o1a A served offer is tagged by the receiving vault](../product/vault-serving.md#792a20o1-uat-scenarios).

#### 792b35c14a Scenarios

```gherkin
Scenario: 792b35c14a1 A new offer records its offer and xid
  Given an offer carrying origin, key, for and path, and target "12" with position "child"
  When it is posted
  Then the note has a server-minted xid, offer.origin, key, for and path, a new top-level Luhmann ID, and the receipt names the note, the vault and the stored exchange hash
```

### 792b35c15 The locked section

Requirement: [792b35c15 The locked section](#792b35c15-the-locked-section) The vault-http-server MUST do a served learn's lookup of every note, its write, its re-embed and the
building of its receipt in one locked section of the vault lock that the local commands use, and MUST fail the request
with a server error, writing nothing, when a listed note cannot be read for any reason other than having been removed since
it was listed, so that the child retries.

Supports: [792a20c1c Concurrent local and served writers lose nothing](../product/vault-serving.md#792a20c1-uat-scenarios), [792a20g1l An unreadable note fails the request](../product/vault-serving.md#792a20g1-uat-scenarios).

Rationale: skipping an unreadable note could miss a same-origin pending offer and write a second one.

#### 792b35c15a Scenarios

```gherkin
Scenario: 792b35c15a1 One lock spans lookup, write, embed and receipt
  Given a new offer and an in-place update
  When each is handled
  Then every note read, the note write and the sidecar embed happen between taking and releasing the one vault lock

Scenario: 792b35c15a2 An unreadable note fails the request
  Given a note the lookup cannot read
  When a served learn is handled
  Then the response is a 5xx and nothing is written

Scenario: 792b35c15a3 Local and served writers do not lose an update
  Given five local learns and five served learns racing against one vault
  When all complete
  Then ten notes exist with ten distinct Luhmann IDs
```

### 792b35c16 The stored hash

Requirement: [792b35c16 The stored hash](#792b35c16-the-stored-hash) The vault-http-server MUST return as `stored_hash` the exchange hash of the note as stored ([792b33c1 The exchange hash](./exchange-versioning.md#792b33c1-the-exchange-hash)),
so that for any offered content it equals the hash a child computes over its own note with the same content.

Supports: [792a20d1a The receipt names the pending note](../product/vault-serving.md#792a20d1-uat-scenarios).

#### 792b35c16a Scenarios

```gherkin
Scenario: 792b35c16a1 The stored hash equals the child's hash
  Given any offered content
  When a served learn stores it
  Then the stored_hash equals the exchange hash a child computes over its own note with the same content, whatever the identity, Luhmann ID, pending marker and exchange fields
```

### 792b35c17 The bounded fetch client

Requirement: [792b35c17 The bounded fetch client](#792b35c17-the-bounded-fetch-client) The vault-http-server's net/http edge MUST make the requests of a child to its parent with a client whose
connect step is bounded to at most 3 seconds and whose total time is bounded to 30 seconds, built on the default transport's
proxy, TLS-handshake and idle-connection settings, and MUST read at most one byte more than the largest response a child
accepts (16 MiB), so that an oversized response is detected and not truncated.

Supports: none. The exchange that uses the client is parent-sync's ([parent-sync](../product/parent-sync.md)), which maps its scenarios.

Rationale: an unreachable parent costs at most 3 seconds once per backoff window, and a malicious parent cannot exhaust a
child's memory.

#### 792b35c17a Scenarios

```gherkin
Scenario: 792b35c17a1 The connect step is bounded
  Given the fetch client
  When its timeouts and transport are read
  Then the connect timeout is above zero and at most 3 seconds, the total timeout is 30 seconds and the transport is a clone of the default with its proxy, TLS-handshake and idle-connection settings

Scenario: 792b35c17a2 The read limit is one byte over the accepted maximum
  Given the largest accepted response
  When the read limit is read
  Then it is one byte more, and the accepted maximum is at least the request cap
```
