# Parent Vault Client - Technical Spec

## 792b37a Orientation

- Ownership: Repository-owned dependency-boundary interaction (the HTTP client of a parent vault's served routes).
- Responsibility: asks the configured parent vault for what a child reads from it and brings the answer into the child's
  vault. It builds and sends the parent requests and classifies their failures, fetches the parent's query answer and
  merges it into the local one (dropping chunks, tagging origin, deduping linked notes, ordering and budgeting the combined
  items), shows a note from the parent, and pulls a parent note down as a local pending offer, re-checking a linked note's
  parent copy, skipping what it already holds or has declined, and telling the parent a note was used.
- Current realization (informational): `internal/cli/serve_client.go` (`parentBase`, `fetchRaw`, `fetchQueryPayload`,
  `dispatchShow`, `fetchShowFallback`, `writeParentSourced`), `merged_query.go` (`runMergedQuery`, `mergeQueryPayloads`),
  `merged_dedupe.go` (`matchLocalNote`, `dedupeParentItems`, `capMergedChannelOne`) and `pulldown.go` (`pullSession`,
  `buildPulledNote`, `parsePulledSource`).
- Qualification: responsibility, the parent requests, the merge and the pull-down are stated without naming a function;
  ownership, it owns what is asked of the parent and how the answer enters the vault, while the parent's side of each route
  is vault-http-server's ([vault-http-server](./vault-http-server.md)), the cache, the backoff gate and the declined-pull record are
  vault-identity-registry's ([792b5c14 The parent cache and the contact backoff](./vault-identity-registry.md#792b5c14-the-parent-cache-and-the-contact-backoff) to [792b5c16 The declined-pull record](./vault-identity-registry.md#792b5c16-the-declined-pull-record)), the routing of `show` and `activate` is the command-line-surface's ([command-line-surface](./command-line-surface.md)) and
  note-amendment's ([792b2c10 Parent routing of note lookup](./command-line-surface.md#792b2c10-parent-routing-of-note-lookup), [792b14c12 Activation against the parent](./note-amendment.md#792b14c12-activation-against-the-parent)), and the outbox drain it triggers is offer-outbox's ([offer-outbox](./offer-outbox.md)); independent
  verification, the merge is tested as pure functions over payloads and notes, and the requests through an injected fetch
  capability over scripted answers; stability, the contract survives a new parent route or a change to the merge's
  ordering; necessity, without it a child could neither read its parent's notes nor use them.
- Supported capabilities: [parent-sync](../product/parent-sync.md).

Rationale: a node with a parent wants one coherent answer, so the merge fuses two complete payloads and
leaves the clusters to the local node, and it fuses notes only, because chunks are the machine's own transcript text. A
pull-down never writes a note live: it lands as a pending offer with the parent's exact content, so curation judges
it before it counts as knowledge, and its exchange hash equals the parent's, which is what lets the skip rule and the
dedupe see the pair. Every request to the parent goes through the same gated, backoff-aware contact, so a down parent costs
one connect timeout per window and not one per command.

## 792b37c Behavior Requirements

### 792b37c1 The parent address

Requirement: [792b37c1 The parent address](#792b37c1-the-parent-address) The parent-vault-client MUST take the parent's base URL from the `ENGRAM_PARENT` environment value
and MUST have no parent when the value is empty or the environment is not available.

Supports: [792a21b1a ENGRAM_PARENT names the one parent](../product/parent-sync.md#792a21b1-uat-scenarios).

Rationale: the parent is an additive setting: a set `ENGRAM_PARENT` adds a step to local behavior and never replaces it.

#### 792b37c1a Scenarios

```gherkin
Scenario: 792b37c1a1 The base URL comes from the environment
  Given ENGRAM_PARENT is set to a URL
  When the parent address is read
  Then it is that URL

Scenario: 792b37c1a2 No environment, no parent
  Given no environment function
  When the parent address is read
  Then there is no parent
```

### 792b37c2 A request to the parent

Requirement: [792b37c2 A request to the parent](#792b37c2-a-request-to-the-parent) The parent-vault-client MUST send each request to the base URL joined with the route and a query
string of percent-encoded keys and values in sorted key order, repeating a key once per value, so that any text, quotes,
newlines and non-ASCII characters included, survives the encoding unchanged. It MUST report a transport error, a timeout
or a 5xx as the parent being unreachable, and any other status outside 200 to 299 as an error that carries the message
of the answer's `{"error": "..."}` body, or the raw body text when it is not that. It MUST return a 2xx answer as it is.

Supports: [792a21k1a The parent does not respond](../product/parent-sync.md#792a21k1-uat-scenarios), [792a21k1c A client error is not an outage](../product/parent-sync.md#792a21k1-uat-scenarios).

Rationale: `internal/` may not import `net/url`, so the encoder is hand-written and held to a round-trip test. An outage backs
the parent off and a 4xx does not, which is why the two errors differ ([792b5c14 The parent cache and the contact backoff](./vault-identity-registry.md#792b5c14-the-parent-cache-and-the-contact-backoff)).

#### 792b37c2a Scenarios

```gherkin
Scenario: 792b37c2a1 A note reference is percent-encoded
  Given a note reference with characters that need escaping
  When it is requested from the parent
  Then the URL carries the escaped value, and a decoder reads the reference back unchanged

Scenario: 792b37c2a2 A long text survives the encoder
  Given a 1 KB text with quotes, newlines, ampersands, equals signs, percent signs and non-ASCII characters
  When it is sent as the text parameter
  Then a standard parser decodes it byte for byte

Scenario: 792b37c2a3 A transport error is unreachable
  Given a connection that fails
  When a request is sent
  Then the error says the parent is unreachable

Scenario: 792b37c2a4 A non-2xx answer carries the server's message
  Given a parent answering 404 with {"error": "no such note"}, and another answering with a body that is not JSON
  When each request fails
  Then the first error says "no such note" and the second carries the raw text
```

### 792b37c3 The parent's query answer

Requirement: [792b37c3 The parent's query answer](#792b37c3-the-parents-query-answer) To fetch the parent's answer to a query, the parent-vault-client MUST send a `GET /query` carrying
each query input that is set (every phrase, `limit`, `project`, `text`, `content-budget`, `recent-fill`, `lazy-chunks` and
`timings`) and `dedupe-keys=1`, and MUST decode a 2xx YAML body into the payload type a local query renders. It MUST
fail for a transport error, a non-2xx answer or a body that is not valid YAML.

Supports: [792a21c1a Local and parent results appear in one ranked list](../product/parent-sync.md#792a21c1-uat-scenarios).

Rationale: a payload decoded into the type the local query renders can be merged with it before a single render, where
copying the parent's bytes to the output could not.

#### 792b37c3a Scenarios

```gherkin
Scenario: 792b37c3a1 Every set input is sent
  Given a query with phrases, limit, project, text, content budget, recent fill, lazy chunks and timings
  When the request is built
  Then each appears as a parameter, with dedupe-keys=1

Scenario: 792b37c3a2 The answer decodes
  Given a parent's YAML answer
  When it is fetched
  Then it decodes into the query payload type

Scenario: 792b37c3a3 A malformed body fails
  Given an answer whose body is not YAML
  When it is fetched
  Then it fails

Scenario: 792b37c3a4 A non-2xx or a transport failure fails
  Given a non-2xx answer, and a failed connection
  When each is fetched
  Then each fails, the second as unreachable
```

### 792b37c4 The merged query

Requirement: [792b37c4 The merged query](#792b37c4-the-merged-query) With `ENGRAM_PARENT` set, `engram query` MUST run the merged path and with it unset MUST run the local
query with no request. The merged path MUST first make the parent contact: stamp a missing vault ID and check the vault's
location (a failed stamp skipping the contact's bookkeeping but not the request), and ask the backoff gate ([792b5c14 The parent cache and the contact backoff](./vault-identity-registry.md#792b5c14-the-parent-cache-and-the-contact-backoff)); it
MUST return the local answer alone, as an ordinary local query, when the gate refuses, when the cached parent vault ID is
this vault's own, or after the request when the parent reports this vault's own ID. It MUST fetch the parent's answer and
run the local query, each with the content budget off and `--recent-fill` and `--limit` unbounded, so that the real values
are applied once, to the merged set. It MUST record the request's outcome for backoff and cache the vault ID the parent
reports; when the parent fails it MUST print one warning and return the local answer; when the local query fails it MUST
return that error. After a merged answer it MUST drain the outbox ([792b36c5 Before a drain sends](./offer-outbox.md#792b36c5-before-a-drain-sends)), unless the vault ID could not be stamped.

Supports: [792a21c1a Local and parent results appear in one ranked list](../product/parent-sync.md#792a21c1-uat-scenarios), [792a21k1a The parent does not respond](../product/parent-sync.md#792a21k1-uat-scenarios), [792a21k1b A query inside the backoff window skips the parent](../product/parent-sync.md#792a21k1-uat-scenarios), [792a21m1a No parent configured](../product/parent-sync.md#792a21m1-uat-scenarios), [792a2q1a A copied vault served back to itself](../product/vault-identity.md#792a2q1-uat-scenarios).

Rationale: a node's recall must not depend on its parent's availability, so every failure falls back to the local
answer. Fetching unbounded and capping once is what keeps the merged size at the requested budget and not at twice it.

#### 792b37c4a Scenarios

```gherkin
Scenario: 792b37c4a1 The request asks for dedupe keys and the answer's vault ID is cached
  Given ENGRAM_PARENT is set
  When engram query runs
  Then the request carries dedupe-keys=1 and the vault ID the parent reports is cached

Scenario: 792b37c4a2 A failed stamp skips only the bookkeeping
  Given a vault whose ID cannot be stamped
  When engram query runs with a parent
  Then the parent is still asked, and no backoff is recorded and no drain creates .engram

Scenario: 792b37c4a3 A self-parent answer returns the local result
  Given a parent that reports this vault's own ID
  When engram query runs
  Then the local results are returned with the self-parent warning

Scenario: 792b37c4a4 A local failure surfaces
  Given a parent that answers and a local query that fails
  When engram query runs
  Then the local error is returned

Scenario: 792b37c4a5 A failing parent leaves the local answer
  Given a parent that fails
  When engram query runs
  Then the local results and one warning are returned

Scenario: 792b37c4a6 A backed-off parent is not asked
  Given a backoff window
  When engram query runs
  Then the local results are returned and no request is made

Scenario: 792b37c4a7 A parent-only runbook is returned and then shown
  Given a runbook only the parent holds
  When engram query runs and the runbook is then shown
  Then the query returns it tagged from_parent and show finds it through the fallback

Scenario: 792b37c4a8 A merged answer drains the outbox
  Given queued offers
  When a merged query succeeds
  Then the offers are sent after the answer
```

### 792b37c5 Dropping and tagging the parent's items

Requirement: [792b37c5 Dropping and tagging the parent's items](#792b37c5-dropping-and-tagging-the-parents-items) The parent-vault-client MUST drop every item of the parent's answer whose kind is `chunk` or whose
provenances include `recent`, before any other step of the merge. It MUST tag every item with the `model_id` of the node
that produced it and a `from_parent` of its own, true for a parent item and false for a local one, and MUST merge whether or
not the two `model_id` values match.

Supports: [792a21c1a Local and parent results appear in one ranked list](../product/parent-sync.md#792a21c1-uat-scenarios), [792a21c1b A parent chunk never appears](../product/parent-sync.md#792a21c1-uat-scenarios), [792a21f1a Mismatched model_id does not block the merge](../product/parent-sync.md#792a21f1-uat-scenarios), [792a21f1b Every merged item carries its source's model](../product/parent-sync.md#792a21f1-uat-scenarios), [792a21g1a Parent-origin and local-origin items are distinguishable](../product/parent-sync.md#792a21g1-uat-scenarios).

Rationale: chunks never cross vaults. A refusal on a model mismatch would break merged recall across a fleet
that upgrades its embedding model one environment at a time. Each item gets its own pointer so that a later write
through one never aliases another (ruling S29).

#### 792b37c5a Scenarios

```gherkin
Scenario: 792b37c5a1 Parent chunks and its recency channel are dropped
  Given a parent answer with chunks and recency items, and notes
  When the answers are merged
  Then the parent's notes remain and local chunks remain, and none of the parent's chunks or recency items does

Scenario: 792b37c5a2 Items carry their source's model and origin
  Given a local and a parent item
  When they are merged
  Then each has its source's model_id and from_parent

Scenario: 792b37c5a3 A mismatched model still merges
  Given the local model_id differs from the parent's
  When the answers are merged
  Then one payload holds both sources' items

Scenario: 792b37c5a4 Each tagged item has its own tag
  Given a list of items to tag
  When tagging runs
  Then every item holds its own from_parent value and none shares a pointer
```

### 792b37c6 Deduping against the local notes

Requirement: [792b37c6 Deduping against the local notes](#792b37c6-deduping-against-the-local-notes) The parent-vault-client MUST treat a parent item P as the same note as a live local note L when L's
`parent.vault` is the vault ID the parent reported in this query and any of L's links names P's basename or one of its
aliases, or when L's exchange hash equals P's `exchange_hash` and both are known. It MUST then drop P when L is already
among the local items or was already substituted, and otherwise replace P at its position with an item for L: L's
basename plus `.md` as `path`, L's kind and content rendered as a local query renders a note (a wikilink-stripped body,
with the `red_flags` preview cap unless the item is an explore pick), P's score and provenances, P's `source_term` only for
an explore pick, the local `model_id` and `from_parent` false. A pending local note MUST NOT suppress a parent item, a
link under another vault ID or while the parent's ID is unknown MUST NOT match, a near-duplicate with no link MUST NOT be
merged, and the dedupe keys MUST NOT remain on any item. A local note whose hash cannot be computed MUST match by link
only, and a failed scan of the local notes MUST be warned and merge without dedupe and not fail the query.

Supports: [792a21j1a A linked note appears once, as the local copy](../product/parent-sync.md#792a21j1-uat-scenarios), [792a21j1b A covered link dedupes](../product/parent-sync.md#792a21j1-uat-scenarios), [792a21j1c A link resolved through an alias](../product/parent-sync.md#792a21j1-uat-scenarios), [792a21j1d Identical content without a link](../product/parent-sync.md#792a21j1-uat-scenarios), [792a21j1e A pending pulled copy does not hide the parent note](../product/parent-sync.md#792a21j1-uat-scenarios), [792a21j1f The local copy is substituted when it did not rank locally](../product/parent-sync.md#792a21j1-uat-scenarios), [792a21j1g Near-duplicates are left alone](../product/parent-sync.md#792a21j1-uat-scenarios), [792a21j1h A link under another parent vault is ignored](../product/parent-sync.md#792a21j1-uat-scenarios), [792a21j1i The dedupe keys are not in the payload](../product/parent-sync.md#792a21j1-uat-scenarios).

Rationale: an exchanged note must not show twice, and the local copy is the one the child can curate. Matching also by
alias covers a parent that renamed or folded the note. An unlinked near-duplicate is left to curation and heals through
the link graph.

#### 792b37c6a Scenarios

```gherkin
Scenario: 792b37c6a1 A primary link dedupes
  Given a local note linked to a parent note that both ranked
  When the answers are merged
  Then the item appears once, as the local copy

Scenario: 792b37c6a2 A covered link dedupes
  Given a local note with a covered link to a second parent note
  When the answers are merged
  Then the second parent note is dropped

Scenario: 792b37c6a3 An alias link dedupes
  Given a local link naming O and a parent note that lists O among its aliases
  When the answers are merged
  Then the local note is kept and the parent note dropped

Scenario: 792b37c6a4 An equal hash dedupes
  Given an unlinked live local note and a parent item with an equal exchange hash
  When the answers are merged
  Then only the local note appears

Scenario: 792b37c6a5 A pending copy suppresses nothing
  Given a pending local copy of a parent note
  When the answers are merged
  Then the parent note remains, from_parent true

Scenario: 792b37c6a6 An unranked local copy takes the parent's place
  Given a parent item that matches a local note which did not rank locally
  When the answers are merged
  Then the local note takes the parent item's position with the parent's score and provenances, the local content and model, and from_parent false

Scenario: 792b37c6a7 A substituted explore pick keeps its term
  Given a parent explore pick that matches a local note
  When the answers are merged
  Then the substituted item keeps the pick's source_term and renders as an explore pick does

Scenario: 792b37c6a8 Near-duplicates are left alone
  Given a local note and a parent note with different content and no link
  When the answers are merged
  Then both appear

Scenario: 792b37c6a9 A link under another vault is ignored
  Given a link recorded under a vault ID other than the one the parent reported
  When the answers are merged
  Then it never dedupes

Scenario: 792b37c6a10 The dedupe keys are not in the payload
  Given a merged payload
  When it is read
  Then no item holds exchange_hash or aliases, and the parent's vault_id is absent

Scenario: 792b37c6a11 A live local note is never shown with a parent item it matches
  Given any local notes and parent items
  When the answers are merged
  Then no live local note appears with a parent item it matches

Scenario: 792b37c6a12 A failed scan merges without dedupe
  Given a scan of the local notes that fails
  When the answers are merged
  Then a warning is printed, the query does not fail, and no item is deduped
```

### 792b37c7 The order of the merged relevance items

Requirement: [792b37c7 The order of the merged relevance items](#792b37c7-the-order-of-the-merged-relevance-items) The parent-vault-client MUST put the trigger hits first, the local ones before the parent's, each in the
order its source gave them, and exempt from `--limit`. It MUST then order the other non-recency items in two groups, the
direct items (those whose provenances do not include `explore`) and then the explore items, each by descending score and
stably, and cap them to `--limit` by position, with `--limit` of zero or less unbounded. When the direct items alone reach
the limit, it MUST keep the first `limit` of them and, when they hold fewer than min(5, |Q|) members of Q (the direct items
of a note kind scoring at least 0.25), replace the lowest-positioned kept chunks one for one with the highest-scoring
excluded members of Q, then restore score order.

Supports: [792a21c1a Local and parent results appear in one ranked list](../product/parent-sync.md#792a21c1-uat-scenarios), [792a21d1a Trigger hits lead, local before parent](../product/parent-sync.md#792a21d1-uat-scenarios), [792a21d1b Explore picks do not crowd out direct matches](../product/parent-sync.md#792a21d1-uat-scenarios), [792a21d1c The note floor survives the merged cap](../product/parent-sync.md#792a21d1-uat-scenarios).

Rationale: the first merge sorted every item by raw score, so explore picks crowded out direct matches (#744); and a cap by
position must still keep the notes the matched-note floor keeps in a local query.

#### 792b37c7a Scenarios

```gherkin
Scenario: 792b37c7a1 Trigger hits lead and bypass the limit
  Given trigger hits from both sources and --limit 1
  When the answers are merged
  Then the local trigger hit and the parent's come first, in that order, and neither is counted against the limit

Scenario: 792b37c7a2 Items from both sources are one list by score
  Given items from both sources
  When the answers are merged
  Then they appear in one list ordered by descending score

Scenario: 792b37c7a3 Direct items come before explore picks
  Given --limit 10, 12 explore picks scoring 0.8 to 0.9 and 15 direct items scoring 0.5 to 0.7
  When the answers are merged
  Then the 10 items are the highest-scoring direct items

Scenario: 792b37c7a4 The limit caps the merged set, not each source
  Given --limit N and items from both sources
  When the answers are merged
  Then the relevance items total at most N, drawn from the combined set

Scenario: 792b37c7a5 The note floor survives the cap
  Given --limit 5, 5 direct chunks at 0.9 and 3 direct notes at 0.5
  When the answers are merged
  Then the three notes and the two highest-scoring chunks are kept, in score order

Scenario: 792b37c7a6 A note below the relevance floor is not promoted
  Given a direct note scoring below 0.25
  When the cap applies
  Then it is not promoted over a chunk

Scenario: 792b37c7a7 The reported crowding shape keeps its notes
  Given ten phrases' direct notes and chunks and explore picks at 0.75 to 0.89 under the default limit
  When the answers are merged
  Then direct items come first and the note floor holds
```

### 792b37c8 The budgets of the merged set

Requirement: [792b37c8 The budgets of the merged set](#792b37c8-the-budgets-of-the-merged-set) The parent-vault-client MUST append the local recency items after the relevance items, cap them to
`--recent-fill` as a group, and not count them against `--limit`. It MUST apply `--content-budget`, or `--lazy-chunks` when
that is set (clearing every chunk's content and snippeting none), once over the merged items, so that at most the budgeted
number of chunk items, all local, render full content by rank.

Supports: [792a21e1a content-budget caps full-content chunk items across the merged set](../product/parent-sync.md#792a21e1-uat-scenarios), [792a21e1b recent-fill caps the recency channel across the merged set](../product/parent-sync.md#792a21e1-uat-scenarios), [792a21e1c limit caps the merged relevance channel](../product/parent-sync.md#792a21e1-uat-scenarios).

Rationale: letting each source apply its own budget would let the merged answer run to twice the requested size. The
recency channel's newest-first order cannot be merged into one chronology, since an item carries no timestamp, so it is
the local channel only, capped as a group.

#### 792b37c8a Scenarios

```gherkin
Scenario: 792b37c8a1 The content budget spans the merged set
  Given --content-budget N and chunk items
  When the answers are merged
  Then at most N chunk items render full content, and the rest render as snippets

Scenario: 792b37c8a2 The recency channel is capped as a group
  Given --recent-fill N and recency items
  When the answers are merged
  Then at most N local recency items remain

Scenario: 792b37c8a3 The limit never starves the recency channel
  Given relevance items that reach --limit by themselves and a recency item
  When the answers are merged
  Then the recency item survives

Scenario: 792b37c8a4 Recency items follow the relevance items
  Given recency items with score zero
  When the answers are merged
  Then they come after the score-ranked items and count against none of the relevance ranking
```

### 792b37c9 The merged payload's own fields

Requirement: [792b37c9 The merged payload's own fields](#792b37c9-the-merged-payloads-own-fields) The merged payload MUST be version 1 with the local query's phrases and clusters, so that no cluster mixes
the two sources; its `pending_offers` flag and hint and its refit flag MUST be the local query's, the hint present exactly
when the flag is, and the parent's flags ignored; and its `model_id` MUST be the querying node's own. Its `budget` block MUST
report the local totals, the number of direct hits and of items with full content, the `limit`, `content_budget` and
`lazy_chunks` applied, the number of chunks snippeted, and `explore_allocated` summed per term over both sources, never
absent.

Supports: [792a21h1a No cross-node cluster grouping](../product/parent-sync.md#792a21h1-uat-scenarios), [792a21h1b The pending-offer flag is the local vault's](../product/parent-sync.md#792a21h1-uat-scenarios), [792a21h1c A local pending offer shows in a merged payload](../product/parent-sync.md#792a21h1-uat-scenarios), [792a21i1a The budget block is populated](../product/parent-sync.md#792a21i1-uat-scenarios).

Rationale: a child cannot curate its parent's offers or refit its parent's vocabulary, so a parent-side flag would tell the
child to do something it cannot; and the first merge reported an all-zero budget block (#743).

#### 792b37c9a Scenarios

```gherkin
Scenario: 792b37c9a1 Clusters stay local
  Given clusters on both sources
  When the answers are merged
  Then the payload holds the local clusters only

Scenario: 792b37c9a2 The advisory flags are the local query's
  Given a parent that holds a pending offer and a refit pending
  When the answers are merged
  Then the payload's flags follow the local query alone

Scenario: 792b37c9a3 The hint follows the flag
  Given a local pending-offer flag, true and then false
  When the answers are merged
  Then the hint is present exactly when the flag is

Scenario: 792b37c9a4 A parent's pending flag is ignored
  Given a pending parent and a clean local vault
  When the answers are merged
  Then neither pending_offers nor its hint appears

Scenario: 792b37c9a5 The payload's model is the local node's
  Given a parent on another model
  When the answers are merged
  Then the payload-level model_id is the local one

Scenario: 792b37c9a6 The budget block reports the merge
  Given --limit 5 and --content-budget 2
  When the answers are merged
  Then the block shows those values, the snippeted count, and explore_allocated summed over both sources

Scenario: 792b37c9a7 Lazy chunks are reported
  Given --limit 5 --content-budget 2 --lazy-chunks
  When the answers are merged
  Then chunk content is cleared, nothing is snippeted, and the block says lazy_chunks true
```

### 792b37c10 The parent-sourced label

Requirement: [792b37c10 The parent-sourced label](#792b37c10-the-parent-sourced-label) When the parent-vault-client prints a note it fetched as the fallback of `show`, it MUST write the line `#
from_parent: true` and then the body verbatim, and MUST fail with an error that wraps the cause when either write fails,
so that no label is left ahead of an error.

Supports: [792a21l1d A local miss falls back to the parent](../product/parent-sync.md#792a21l1-uat-scenarios).

Rationale: the fallback leaves the reader unsure where the text came from, so only it is labeled; the answer is buffered
first so that a failed fetch never leaves a stray label.

#### 792b37c10a Scenarios

```gherkin
Scenario: 792b37c10a1 A failed label write is reported
  Given an output that fails on its first write
  When the parent-sourced note is written
  Then the error wraps the cause

Scenario: 792b37c10a2 A failed body write is reported
  Given an output that fails on its second write
  When the parent-sourced note is written
  Then the error wraps the cause
```

### 792b37c11 A pull-down is a gated parent contact

Requirement: [792b37c11 A pull-down is a gated parent contact](#792b37c11-a-pull-down-is-a-gated-parent-contact) Before its first fetch in a command, the parent-vault-client MUST stamp a missing vault ID, check the
vault's location, apply the cached self-parent guard and ask the backoff gate, and MUST refuse every pull and re-check of
the command, fetching and writing nothing, with the one warning of whichever check failed, when a stamp fails, the
location check fails, the cached parent is this vault's own, the gate refuses, or the exchange state cannot be read. An
outage on a fetch MUST record one failure, print the one backoff warning with the queued count (naming the error instead
when the failure cannot be recorded) and refuse the later refs without a request. A successful contact MUST drain the
outbox once after the last ref ([792b36c5 Before a drain sends](./offer-outbox.md#792b36c5-before-a-drain-sends)), behind the same gate.

Supports: [792a21zg1c A paused vault pulls nothing](../product/parent-sync.md#792a21zg1-uat-scenarios), [792a21zg1d An outage pauses the later refs](../product/parent-sync.md#792a21zg1-uat-scenarios), [792a21zg1e A pull-down drains the outbox](../product/parent-sync.md#792a21zg1-uat-scenarios).

Rationale: a pull-down writes to the vault and contacts a remote vault, so it pauses for the same reasons an offer does
([792a2p Copied and cloned vaults pause exchange](../product/vault-identity.md#792a2p-copied-and-cloned-vaults-pause-exchange), [792a2q A vault does not exchange with itself](../product/vault-identity.md#792a2q-a-vault-does-not-exchange-with-itself)). A parent that is down would otherwise be tried again for every ref.

#### 792b37c11a Scenarios

```gherkin
Scenario: 792b37c11a1 A failed check pauses the pull-down
  Given a failed location check, an active backoff window, a self-parent or unusable exchange state
  When refs are pulled
  Then each stops the pull-down with its one warning and nothing is fetched or written

Scenario: 792b37c11a2 An outage pauses the later refs
  Given an unreachable parent
  When several refs are pulled
  Then the first fetch records one failure and prints one backoff warning with the queued count, and the rest make no request

Scenario: 792b37c11a3 An unrecordable failure names the error
  Given an outage whose failure cannot be recorded
  When the warning is printed
  Then it names that error and not a zero retry time

Scenario: 792b37c11a4 A pull-down drains the outbox
  Given queued offers
  When a pull-down succeeds
  Then the offers drain once, after the last ref
```

### 792b37c12 The raw envelope

Requirement: [792b37c12 The raw envelope](#792b37c12-the-raw-envelope) The parent-vault-client MUST fetch a note with `GET /show?note=<name>&raw=1`, holding no vault lock, and
MUST accept the answer only as a JSON object with a `vault_id`, a `basename`, an `exchange_hash` and `content`. It MUST
fail as the parent being too old for an answer that is not such an object, and write nothing for an envelope whose
`vault_id` is not an exchange identifier or whose `basename` is not an exchange basename ([792b5c15 Exchange identifiers](./vault-identity-registry.md#792b5c15-exchange-identifiers)), whose content has no
frontmatter, whose type is not `fact`, `feedback` or `runbook`, or whose built copy's exchange hash differs from the
envelope's. It MUST fail as not found for a non-2xx answer. Any answer from the parent MUST reset the backoff and cache a
well formed reported vault ID, a malformed one never being cached, and a CRLF envelope MUST be read as its LF form.

Supports: [792a21zb1d A CRLF parent note pulls down](../product/parent-sync.md#792a21zb1-uat-scenarios), [792a21zd1d A pre-change parent is detected](../product/parent-sync.md#792a21zd1-uat-scenarios), [792a21zd1e A malformed envelope is refused](../product/parent-sync.md#792a21zd1-uat-scenarios), [792a21zd1f A hash that does not match is refused](../product/parent-sync.md#792a21zd1-uat-scenarios), [792a21zd1g A note of another type is refused](../product/parent-sync.md#792a21zd1-uat-scenarios), [792a21zd1h A pending parent note is never pulled](../product/parent-sync.md#792a21zd1-uat-scenarios).

Rationale: the envelope is JSON with a `vault_id`, so it cannot be mistaken for the rendered output of an older parent, and
a parent is another machine's input ([792b5c15 Exchange identifiers](./vault-identity-registry.md#792b5c15-exchange-identifiers)).

#### 792b37c12a Scenarios

```gherkin
Scenario: 792b37c12a1 An older parent is detected
  Given an answer that is a rendered note, or JSON without a vault_id
  When the note is pulled
  Then it fails as the parent being too old and nothing is written

Scenario: 792b37c12a2 A malformed envelope writes nothing
  Given an envelope with a malformed vault_id or a basename holding a separator or |
  When the note is pulled
  Then nothing is written and the malformed ID is never cached

Scenario: 792b37c12a3 A mismatched hash writes nothing
  Given an envelope whose exchange_hash does not match its content
  When the note is pulled
  Then nothing is written

Scenario: 792b37c12a4 A note of another type is refused
  Given a fetched note that is not a fact, feedback or runbook
  When it is pulled
  Then nothing is written

Scenario: 792b37c12a5 A CRLF envelope pulls as LF
  Given an envelope whose content has CRLF line endings
  When the note is pulled
  Then the pending copy is LF and linked at the parent's hash

Scenario: 792b37c12a6 The fetch holds no lock and an answer resets the backoff
  Given a pull in progress
  When the raw fetch is made and answered
  Then the vault lock is not held, the failure count is zero and the vault ID is cached

Scenario: 792b37c12a7 A pending parent note is not pulled
  Given a parent whose note is pending
  When the note is pulled
  Then the parent answers not found and nothing is written
```

### 792b37c13 The pulled copy

Requirement: [792b37c13 The pulled copy](#792b37c13-the-pulled-copy) Under the vault lock, the parent-vault-client MUST write the pulled note as a new top-level note, named
from the parent's slug, with a fresh Luhmann ID and `xid`, the creation date of the pull, `pending: true`, the parent's
frontmatter without `parent`, `aliases`, `offer`, `xid`, `skill_hash`, `skill_key`, `skill_source`, `sources`,
`supersedes`, `tags`, `vocab_version` or `pending`, `repo`, `user` and `vault` stamped by the first-write identity (an
undetectable `repo`, `user` or `vault` omitted and never the parent's), a `parent` block with `vault` the envelope's
vault ID, one `via: pulled` link `{note, hash}` to the envelope's basename and hash, and `author` the parent note's
`repo`, `user` and `vault`, and the parent's body verbatim. Its exchange hash MUST equal the envelope's. It MUST be
embedded on write and tagged with the local vocabulary, and its path MUST be printed.

Supports: [792a21zd1a Pulled note is pending and linked](../product/parent-sync.md#792a21zd1-uat-scenarios), [792a21zd1b Remote identity and registration fields are stripped](../product/parent-sync.md#792a21zd1-uat-scenarios), [792a21zd1c Chunk provenance does not travel](../product/parent-sync.md#792a21zd1-uat-scenarios), [792a21zd1i The pulled note is embedded and does not keep the parent's tags](../product/parent-sync.md#792a21zd1-uat-scenarios), [792a21zh1a No user detectable on a pull-down](../product/parent-sync.md#792a21zh1-uat-scenarios).

Rationale: the parent's identity fields, tags and chunk provenance mean nothing in this vault, and a pulled note is never
live, so curation judges it before it counts as knowledge. Keeping the exact content is what keeps the hashes equal.

#### 792b37c13a Scenarios

```gherkin
Scenario: 792b37c13a1 The copy is pending, linked and stripped
  Given a parent note carrying its own aliases, xid, parent link, skill fields, sources, supersedes and tags
  When it is pulled
  Then the copy carries none of them, and holds pending true, a via pulled primary link with the envelope's hash and parent.author

Scenario: 792b37c13a2 The identity is stamped locally
  Given a parent note by another author
  When it is pulled
  Then repo, user and vault are this vault's, and the author is kept in parent.author

Scenario: 792b37c13a3 An undetectable user is omitted
  Given a host where user detection resolves empty
  When a parent note is pulled
  Then the copy has no user key, vault is stamped, and one warning says detection resolved empty

Scenario: 792b37c13a4 The copy hashes like the parent's note
  Given any pulled note
  When it is written
  Then its exchange hash equals the envelope's, and it is embedded

Scenario: 792b37c13a5 The parent's tags are not kept
  Given a parent note carrying vocabulary tags
  When it is pulled
  Then the copy does not keep them
```

### 792b37c14 What a pull-down skips

Requirement: [792b37c14 What a pull-down skips](#792b37c14-what-a-pull-down-skips) Under the vault lock the parent-vault-client MUST apply the skip rule again and write nothing when a local note,
live or pending, has any link to the envelope's basename or one of its aliases under the envelope's vault ID whose hash is
not changed from the envelope's, bumping the sidecar of each live such note instead, or when the declined-pull record
holds a decline for that vault ID, one of those names and a hash not changed from the envelope's ([792b5c16 The declined-pull record](./vault-identity-registry.md#792b5c16-the-declined-pull-record)). A parent
note whose hash changed MUST be written as another pending copy. Two pulls of one note racing on a vault MUST write one
copy, and any sequence of pull, clear-pending, discard and fold on an unchanged parent note MUST leave at most one local
note linked to it.

Supports: [792a21ze1a Re-activating an unchanged parent note](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1b A covered parent note is not pulled again](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1c A declined pull is remembered](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1d A decline survives a parent rename](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1e A changed parent note arrives as a new offer](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1g A decline under another parent does not apply](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1k Racing activates write one copy](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1l No sequence leaves two linked copies](../product/parent-sync.md#792a21ze1-uat-scenarios).

Rationale: re-pulling an unchanged note on every use would fill the vault with copies; the check under the lock is what
makes two racing activates safe, since the first check ran before the fetch.

#### 792b37c14a Scenarios

```gherkin
Scenario: 792b37c14a1 A linked live note is bumped, not copied
  Given a live local note with a covered or primary link to the note at its current hash
  When the note is pulled again
  Then nothing is written and the live note's last-used date is today

Scenario: 792b37c14a2 A decline suppresses an unchanged note
  Given a recorded decline of the note, including under an old name now in its aliases
  When the note is pulled again unchanged
  Then nothing is written

Scenario: 792b37c14a3 A decline under another parent does not apply
  Given a decline recorded under another parent vault
  When the note is pulled
  Then it is written

Scenario: 792b37c14a4 A changed note arrives again
  Given a note pulled and accepted and then changed on the parent
  When it is pulled again
  Then a second pending copy is written with the new hash

Scenario: 792b37c14a5 A copy that lands mid-pull is seen
  Given a copy written between the fetch and the lock
  When the pull takes the lock
  Then the re-check finds it and writes nothing

Scenario: 792b37c14a6 Racing pulls write one copy
  Given two pulls of one note on one vault
  When both complete
  Then one copy exists

Scenario: 792b37c14a7 No sequence duplicates
  Given any sequence of pull, clear-pending, discard and fold on an unchanged note
  When the sequence ends
  Then at most one local note is linked to the parent note
```

### 792b37c15 Telling the parent the note was used

Requirement: [792b37c15 Telling the parent the note was used](#792b37c15-telling-the-parent-the-note-was-used) After a pull-down that wrote the note or a skip, once the vault lock is released, the parent-vault-client MUST send `POST /activate` for the
parent note's name, MUST NOT send it when the local write failed, and a failure of that request MUST NOT fail the command, MUST NOT be queued and MUST NOT back the
parent off. The re-check of a linked note ([792b37c16 Re-checking a linked note's parent copy](#792b37c16-re-checking-a-linked-notes-parent-copy)) MUST NOT send it.

Supports: [792a21zg1a Parent recency is bumped](../product/parent-sync.md#792a21zg1-uat-scenarios), [792a21zg1b A failed signal is not fatal](../product/parent-sync.md#792a21zg1-uat-scenarios), [792a21zg1f The fetch holds no lock](../product/parent-sync.md#792a21zg1-uat-scenarios), [792a21zg1g A failed local write sends no signal](../product/parent-sync.md#792a21zg1-uat-scenarios).

Rationale: the parent ranks what its children use, and a lost hint costs nothing.

#### 792b37c15a Scenarios

```gherkin
Scenario: 792b37c15a1 The parent's recency is bumped
  Given a pulled note and a reachable parent
  When the pull-down completes
  Then the parent received one POST /activate for the note, sent with the vault lock released

Scenario: 792b37c15a2 A failed signal is harmless
  Given the parent's activate request fails
  When the pull-down completes
  Then the command succeeds, nothing is queued and the parent is not backed off

Scenario: 792b37c15a3 A failed local write sends no signal
  Given a pulled note whose local write fails
  When the pull-down runs
  Then the command fails and the parent received no POST /activate for the note
```

### 792b37c16 Re-checking a linked note's parent copy

Requirement: [792b37c16 Re-checking a linked note's parent copy](#792b37c16-re-checking-a-linked-notes-parent-copy) For a linked local note that has been activated, the parent-vault-client MUST skip a link under another
parent vault, without contacting the parent when the configured parent's vault ID is cached and differs, then fetch the link's
raw envelope through the same gated contact as a pull ([792b37c11 A pull-down is a gated parent contact](#792b37c11-a-pull-down-is-a-gated-parent-contact)) and write under the lock exactly as a pull does
([792b37c14 What a pull-down skips](#792b37c14-what-a-pull-down-skips)): nothing when the parent note's hash equals the link's or is declined, and otherwise a new pending copy. When the
envelope's vault ID differs from the link's it MUST discard the answer. It MUST NOT signal use to the parent, and its failure MUST NOT
fail the local activate.

Supports: [792a21zf1a A changed parent note arrives through its local copy](../product/parent-sync.md#792a21zf1-uat-scenarios), [792a21zf1b An unchanged parent note writes nothing](../product/parent-sync.md#792a21zf1-uat-scenarios), [792a21zf1c A declined update is not pulled again](../product/parent-sync.md#792a21zf1-uat-scenarios), [792a21zf1d An unreachable parent does not fail the local activate](../product/parent-sync.md#792a21zf1-uat-scenarios), [792a21zf1e A link under another parent is never checked](../product/parent-sync.md#792a21zf1-uat-scenarios), [792a21zf1f A served parent works end to end](../product/parent-sync.md#792a21zf1-uat-scenarios).

Rationale: the merged query shows a linked local note in place of its parent note, so a parent-side change reaches the child
only through a use of the local copy.

#### 792b37c16a Scenarios

```gherkin
Scenario: 792b37c16a1 A changed parent note arrives through the local copy
  Given an accepted pulled copy and a later change on the parent
  When the local copy is activated, and again
  Then a pending copy with the new hash is written once, and the second activation writes nothing more

Scenario: 792b37c16a2 An unchanged or declined note writes nothing
  Given a linked note whose parent hash equals the link's, and a changed one that was declined
  When each is activated
  Then nothing is written

Scenario: 792b37c16a3 A link under another parent is never written
  Given a link whose vault differs from the configured parent's cached ID, and another whose vault differs from the answer's
  When each is activated
  Then the first is not contacted and the second's answer is discarded

Scenario: 792b37c16a4 An unreachable parent does not fail the activate
  Given an unreachable parent
  When a linked note is activated
  Then the activate succeeds and the failure is recorded for backoff
```
