# Parent Sync - Product Spec

A vault can have one parent: another engram vault, served with `engram serve` ([vault-serving](./vault-serving.md)), that sits above it in a tree
(personal, then team, then organization) and never in a mesh. `ENGRAM_PARENT` names it. Every environment keeps its own
local vault whether or not it has a parent, so a command works with the parent unreachable, and the two vaults exchange
notes in both directions. A node offers what it writes upward, through an outbox that survives being offline. It reads
its parent's notes back in one merged answer to `engram query`, and pulls a parent's note down when it uses it, as an
offer for local curation to judge ([curate](./curate.md)). Notes are exchanged and transcript chunks never are. How a vault tells a
clone or a copy of itself from its parent, and what an exchange is allowed to touch, are [vault-identity](./vault-identity.md), and the
served side of the exchange is [vault-serving](./vault-serving.md). The `ENGRAM_SERVER` thin-client mode that the exchange replaced
is gone, and [792a21a ENGRAM_SERVER is refused](#792a21a-engram_server-is-refused) makes setting it an error.

## 792a21a ENGRAM_SERVER is refused

Requirement: [792a21a ENGRAM_SERVER is refused](#792a21a-engram_server-is-refused) When the `ENGRAM_SERVER` environment variable is set to a non-empty value, every `engram`
subcommand, `engram serve` included, MUST exit non-zero before it reads or writes a vault or a chunk index or contacts a
network endpoint, with an error that names `ENGRAM_SERVER` as no longer supported and tells the user to set
`ENGRAM_PARENT` to the same URL. No subcommand MUST treat `ENGRAM_SERVER` as an alias for `ENGRAM_PARENT`, and no
thin-client mode MUST remain.

Rationale: `ENGRAM_SERVER` made the CLI a thin HTTP client with no local vault, so every write landed only on the remote
and a server outage failed every command. Joe replaced it with the local-first model on 2026-09-27 and ruled out an
alias shim and an exemption for `serve`. A host that still sets it fails loudly instead of quietly
behaving as something else.

### 792a21a1 UAT Scenarios

```gherkin
Scenario: 792a21a1a A command run with ENGRAM_SERVER set fails fast
  Given ENGRAM_SERVER is set to http://host:8093
  When the user runs engram query --phrase x
  Then it exits non-zero, its error says to set ENGRAM_PARENT=http://host:8093 instead, and no vault file, chunk file or HTTP request is touched

Scenario: 792a21a1b Writes are refused too
  Given ENGRAM_SERVER is set
  When the user runs engram learn fact with its arguments
  Then it exits non-zero with the same error, and no note is written locally or remotely

Scenario: 792a21a1c serve is not exempt
  Given ENGRAM_SERVER is set
  When the user runs engram serve --addr 127.0.0.1:8093
  Then it exits non-zero with the same error, and it binds no port

Scenario: 792a21a1d ENGRAM_SERVER is not honored as an alias
  Given ENGRAM_SERVER is set and ENGRAM_PARENT is not
  When the user runs any engram command
  Then no command contacts the ENGRAM_SERVER URL
```

## 792a21b One parent per vault

Requirement: [792a21b One parent per vault](#792a21b-one-parent-per-vault) `engram query` MUST accept an `ENGRAM_PARENT` environment variable that names exactly one parent
vault's base URL, and MUST NOT offer any way to configure more than one parent at a time.

Rationale: the vaults form a single-parent tree, so a mesh and its cycles cannot be configured.

### 792a21b1 UAT Scenarios

```gherkin
Scenario: 792a21b1a ENGRAM_PARENT names the one parent
  Given ENGRAM_PARENT is set to a served vault's base URL
  When the user runs engram query
  Then that URL is this node's one parent for merged results
```

## 792a21c A merged query returns the parent's notes, never its chunks, in one list

Requirement: [792a21c A merged query returns the parent's notes, never its chunks, in one list](#792a21c-a-merged-query-returns-the-parents-notes-never-its-chunks-in-one-list) With `ENGRAM_PARENT` set, every `engram query` MUST run its local query and ask the parent for
its answer, MUST drop every item of the parent's answer whose `kind` is `chunk`, MUST remove the duplicates of [792a21j A linked or identical note appears once, as the local copy](#792a21j-a-linked-or-identical-note-appears-once-as-the-local-copy),
and MUST return one payload ranked across both sources in the order of [792a21d A merged query orders its items](#792a21d-a-merged-query-orders-its-items), not two result sets for the caller to
reconcile. This merged path is the only query path for a node that has a parent: it MUST NOT offer another remote query
mode, and no parent chunk MUST appear in a merged payload.

Rationale: a node with a parent wants one coherent view of local and inherited memory. Chunks are
transcript text that belongs to the machine that recorded it, so they never cross vaults, and the merge, its recency
channel and its chunk budgets are local-only. The merge fuses two complete payloads rather than
fusing before clustering, because the second would need the parent to expose its candidates, and it sorts by raw score
without normalizing it, because a within-batch rescale makes a weak match look as strong as a good one.

### 792a21c1 UAT Scenarios

```gherkin
Scenario: 792a21c1a Local and parent results appear in one ranked list
  Given ENGRAM_PARENT is set, and both the local vault and the parent vault have notes that match the query
  When the user runs engram query
  Then the payload's items include notes from both sources, placed by the order of 792a21d, and no parent chunk

Scenario: 792a21c1b A parent chunk never appears
  Given a parent whose answer holds chunk items
  When the user runs engram query with ENGRAM_PARENT set
  Then none of the parent's chunk items is in the payload
```

## 792a21d A merged query orders its items

Requirement: [792a21d A merged query orders its items](#792a21d-a-merged-query-orders-its-items) A merged query MUST place the trigger hits first, the local ones before the parent's, each in the
order its source returned them. It MUST then order the other relevance items in two groups, the direct items (those
whose provenances include none of `trigger`, `explore` or `recent`) from both sources by descending score, followed by
the explore items (those whose provenances include `explore`) from both sources by descending score. It MUST apply
`--limit` to that order by position, except that when the kept set holds fewer than min(5, Q) note items, where Q is
the direct items of a note kind (any kind other than `chunk`) that score at least 0.25, it MUST replace the
lowest-positioned kept direct chunk items one for one with the highest-scoring excluded members of Q until that
minimum is kept or no kept direct chunk remains, and then put the direct group back in score order.

Rationale: explore picks scoring higher than direct matches crowded the direct matches out of a merged answer before
this order existed (#744). The note floor keeps the minimum of qualifying notes that a local query keeps under a tight
`--limit` (the matched-note floor of memory-query, [792a7e Matched-note floor](./memory-query.md#792a7e-matched-note-floor)), now over both sources.

### 792a21d1 UAT Scenarios

```gherkin
Scenario: 792a21d1a Trigger hits lead, local before parent
  Given ENGRAM_PARENT is set, the local vault returns one trigger hit and an item scoring 0.9, and the parent returns one trigger hit and an item scoring 0.8
  When the user runs engram query --text="<msg>" --limit 1
  Then the items run local trigger hit, parent trigger hit, local 0.9 item, both trigger hits survive the limit, and the parent's 0.8 item is dropped

Scenario: 792a21d1b Explore picks do not crowd out direct matches
  Given a merged query with --limit 10, 12 explore picks scoring 0.8 to 0.9 and 15 direct items scoring 0.5 to 0.7 across both sources
  When the user runs it
  Then the 10 relevance items are the 10 highest-scoring direct items, and no explore pick is included

Scenario: 792a21d1c The note floor survives the merged cap
  Given a merged query with --limit 5, 5 direct local chunk items scoring 0.9 and 3 direct note items scoring 0.5
  When the user runs it
  Then the 5 relevance items include all 3 notes and the 2 highest-scoring chunks
```

## 792a21e The budgets apply to the merged set

Requirement: [792a21e The budgets apply to the merged set](#792a21e-the-budgets-apply-to-the-merged-set) With `ENGRAM_PARENT` set, `engram query` MUST apply `--content-budget`, `--recent-fill` and `--limit`
once, over the merged items, and MUST NOT apply them to each source before the merge. `--content-budget` MUST span both
channels, `--recent-fill` MUST govern the recency channel only, and `--limit` MUST govern the relevance channel only
and MUST NOT displace the recency channel. Trigger hits MUST be exempt from `--limit`, and `--content-budget` MUST still
apply to them. Because parent chunks are dropped, every chunk item, the whole recency channel included, is the local
vault's.

Rationale: letting each source apply its own budget and concatenating the results would let the merged answer run to
twice the requested size, which is the payload-size problem the flags exist to prevent. Each source is therefore
fetched unbounded and the real values are applied once. The recency channel's newest-first order cannot be merged into
one chronology, which the payload does not carry, and it is local-only for that reason as well ([792a7q Limit](./memory-query.md#792a7q-limit) owns
`--limit`'s base enforcement; this requirement says only where in the merge the budgets apply).

### 792a21e1 UAT Scenarios

```gherkin
Scenario: 792a21e1a content-budget caps full-content chunk items across the merged set
  Given ENGRAM_PARENT is set
  When the user runs engram query --content-budget N
  Then at most N chunk items, all local, render full content by rank, lower-ranked chunk items render as a snippet, and no parent chunk is present

Scenario: 792a21e1b recent-fill caps the recency channel across the merged set
  Given ENGRAM_PARENT is set
  When the user runs engram query --recent-fill N
  Then the recency channel holds at most N newest-by-ingest local chunk items, none from the parent, and not N from each source

Scenario: 792a21e1c limit caps the merged relevance channel
  Given ENGRAM_PARENT is set
  When the user runs engram query --limit N
  Then the relevance items total at most N, drawn from the combined score-ranked set and not N from each source, and the recency channel is not counted against N
```

## 792a21f The merge does not depend on matching models

Requirement: [792a21f The merge does not depend on matching models](#792a21f-the-merge-does-not-depend-on-matching-models) `engram query` MUST merge the local and the parent results regardless of whether their `model_id`
values match, and MUST NOT refuse to run, fail or drop either source's results solely because they differ. Each item of
a merged payload MUST carry the `model_id` of the node that produced it.

Rationale: refusing or falling back to local-only on a model mismatch would break merged recall across a fleet that
upgrades its embedding model one environment at a time, which is a stated goal. The per-item `model_id` is a visible,
non-gating signal, in the way a local stale-sidecar mismatch is a warning rather than an error.

### 792a21f1 UAT Scenarios

```gherkin
Scenario: 792a21f1a Mismatched model_id does not block the merge
  Given the local vault's model_id differs from the parent's
  When the user runs engram query
  Then it returns a single ranked payload combining both sources' items

Scenario: 792a21f1b Every merged item carries its source's model
  Given a merged payload
  When it is read
  Then every item includes the model_id of the node, local or parent, that produced it
```

## 792a21g Merged items say where they came from

Requirement: [792a21g Merged items say where they came from](#792a21g-merged-items-say-where-they-came-from) Each item of a merged payload MUST carry a boolean `from_parent` that is true for an item the
parent produced and false, written explicitly, for an item produced locally, and an item of a payload that no merge
produced MUST carry no `from_parent` key.

Rationale: a caller that follows an item up with `engram show` needs to know whether to look in the parent. A bare
Luhmann ID is minted per vault, so two vaults routinely hold unrelated notes under the same ID, and a guess made
without the origin could return the wrong note with no error at all.

### 792a21g1 UAT Scenarios

```gherkin
Scenario: 792a21g1a Parent-origin and local-origin items are distinguishable
  Given a merged payload
  When it is read
  Then each parent item has from_parent true and each local item has from_parent false

Scenario: 792a21g1b A payload with no merge has no from_parent
  Given ENGRAM_PARENT is not set
  When the user runs engram query
  Then no item in the payload carries a from_parent key
```

## 792a21h A merged answer does not re-cluster

Requirement: [792a21h A merged answer does not re-cluster](#792a21h-a-merged-answer-does-not-re-cluster) A merged payload MUST present its combined items as one ranked list, with the clusters the local
query found, and MUST NOT compute a new grouping that mixes local-origin and parent-origin items in one cluster. The
payload's `pending_offers` flag and its hint, and its refit flag, MUST reflect the local vault only.

Rationale: re-clustering across two nodes' items is a different experiment (#731) that waits on a baseline of this
one. A child cannot curate its parent's offers or refit its parent's vocabulary, so a parent-side flag would tell the
child to do something it cannot.

### 792a21h1 UAT Scenarios

```gherkin
Scenario: 792a21h1a No cross-node cluster grouping
  Given a merged payload
  When it is read
  Then no cluster in it mixes local-origin and parent-origin items

Scenario: 792a21h1b The pending-offer flag is the local vault's
  Given a parent that holds a pending offer and a local vault that holds none
  When the user runs engram query
  Then the merged payload has neither pending_offers nor its hint

Scenario: 792a21h1c A local pending offer shows in a merged payload
  Given a local vault that holds a pending offer
  When the user runs engram query with ENGRAM_PARENT set
  Then the merged payload has pending_offers true and its hint
```

## 792a21i The merged payload reports the budget it applied

Requirement: [792a21i The merged payload reports the budget it applied](#792a21i-the-merged-payload-reports-the-budget-it-applied) A merged payload's `budget` block MUST report the values the merge applied and not zeros: the
`limit`, `content_budget` and `lazy_chunks` it used, the `chunks_snippeted` its content pass counted, and
`explore_allocated` summed per term over both sources.

Rationale: the first merge dropped the block to all zeros (#743), which hid whether a cap had applied.

### 792a21i1 UAT Scenarios

```gherkin
Scenario: 792a21i1a The budget block is populated
  Given ENGRAM_PARENT is set
  When the user runs engram query --limit 5 --content-budget 2 --lazy-chunks
  Then the payload's budget shows limit 5, content_budget 2, lazy_chunks true and the snippeted-chunk count the merge produced
```

## 792a21j A linked or identical note appears once, as the local copy

Requirement: [792a21j A linked or identical note appears once, as the local copy](#792a21j-a-linked-or-identical-note-appears-once-as-the-local-copy) Before it orders and caps, a merged query MUST treat a parent note P as the same note as a live
(not pending) local note L when L's `parent.vault` equals the vault ID the parent reported in this query and any of L's
parent links (`offered`, `pulled` or `covered`) names P's basename or one of P's `aliases`, or when L's exchange hash
equals P's `exchange_hash` and both are non-empty. It MUST keep L and not P: when L is already among the local items P
MUST be dropped, and otherwise P MUST be replaced by an item for L that carries L's basename plus `.md` as `path`, L's
note kind, L's content as a local query renders it, P's `score` and `provenances`, P's `source_term` only when P's
provenances include `explore`, the local vault's `model_id`, and `from_parent` false. A pending local note MUST NOT
suppress a parent item, a near-duplicate (different content, no link) MUST NOT be merged by the query, and the dedupe
keys (`exchange_hash`, `aliases`, and the parent's `vault_id`) MUST NOT appear in the merged payload.

Rationale: a note exchanged between the vaults would otherwise show twice, and the local copy is the one the child can
amend and curate. A link names a basename or an alias because the parent may have renamed or folded the note since. A
pending copy is unjudged, so it must not hide the parent's note, and near-duplicates are left to curation, which heals
them through the link graph; an unlinked near-duplicate may show twice until a later pull-down or fold links the pair.

### 792a21j1 UAT Scenarios

```gherkin
Scenario: 792a21j1a A linked note appears once, as the local copy
  Given local note L carries a primary link to parent note P, and both match the query
  When the user runs engram query
  Then the merged items contain L with from_parent false and do not contain P

Scenario: 792a21j1b A covered link dedupes
  Given L carries a covered link to parent note P2 in addition to its primary link
  When the user runs engram query
  Then P2 does not appear and L appears once

Scenario: 792a21j1c A link resolved through an alias
  Given L's link names basename O, and the parent's note E lists O in its aliases
  When the user runs engram query
  Then the merged payload keeps L and drops E

Scenario: 792a21j1d Identical content without a link
  Given an unlinked live local note and a parent note with equal exchange hashes
  When the user runs engram query
  Then only the local note appears

Scenario: 792a21j1e A pending pulled copy does not hide the parent note
  Given the only local copy of parent note P is a pending pull-down that curation has not judged
  When the user runs engram query
  Then P still appears, with from_parent true

Scenario: 792a21j1f The local copy is substituted when it did not rank locally
  Given P matches linked local note L, and L is not among the local query's items
  When the user runs engram query
  Then an item for L with P's score and provenances, L's content, the local model_id and from_parent false appears at P's position

Scenario: 792a21j1g Near-duplicates are left alone
  Given a local note and a parent note with different content and no link
  When the user runs engram query
  Then both may appear

Scenario: 792a21j1h A link under another parent vault is ignored
  Given a local note linked to a parent vault other than the one that answered
  When the user runs engram query
  Then the local note is treated as unlinked and does not hide the parent's note

Scenario: 792a21j1i The dedupe keys are not in the payload
  Given a merged payload
  When it is read
  Then it carries no exchange_hash, aliases or parent vault_id
```

## 792a21k An unreachable parent leaves the local answer

Requirement: [792a21k An unreachable parent leaves the local answer](#792a21k-an-unreachable-parent-leaves-the-local-answer) When the parent is unreachable or answers with an error, `engram query` MUST still return the
local vault's results as in local-only mode, and MUST print a non-fatal warning that the parent was unavailable,
instead of failing. While the parent is backed off ([792a21s An unreachable parent is backed off](#792a21s-an-unreachable-parent-is-backed-off)), a query MUST make no parent request and MUST return the
local results with one warning.

Rationale: a node's local recall should not depend on its parent's network availability, or one host outage takes down
every environment's recall.

### 792a21k1 UAT Scenarios

```gherkin
Scenario: 792a21k1a The parent does not respond
  Given ENGRAM_PARENT is set and the parent does not respond (a network error, a timeout or a non-2xx answer)
  When the user runs engram query
  Then it returns the local vault's results and a non-fatal warning, and does not fail

Scenario: 792a21k1b A query inside the backoff window skips the parent
  Given a query failed to reach the parent a moment ago
  When the user runs engram query again inside the backoff window
  Then no parent request is made, the local results are returned, and one warning is printed

Scenario: 792a21k1c A client error is not an outage
  Given a parent that answers a query with a 4xx
  When the user runs engram query and then again
  Then the second query still contacts the parent
```

## 792a21l A note can be shown from the parent

Requirement: [792a21l A note can be shown from the parent](#792a21l-a-note-can-be-shown-from-the-parent) `engram show` MUST accept a `--parent` flag. With `--parent` set and `ENGRAM_PARENT` configured it
MUST resolve the ref against the parent vault instead of the local one, and with `--parent` set and `ENGRAM_PARENT`
unset it MUST fail without a lookup. Without `--parent`, when `ENGRAM_PARENT` is configured and the ref is not found
locally, it MUST fall back to the parent and label the output as parent-sourced, with the label ahead of any exchange-hash header the parent's note carries, and a ref found locally MUST be
returned as before this capability, without contacting the parent. A miss with no parent configured MUST be the same
not-found error as before. `engram show-chunk` MUST NOT accept `--parent` and MUST NOT fall back to the parent.

Rationale: chunks never cross vaults, so there is no parent chunk lookup. A fallback that fires only on a local miss is
safe by construction, and the origin tag of [792a21g Merged items say where they came from](#792a21g-merged-items-say-where-they-came-from) tells a caller when to ask for the parent explicitly.

### 792a21l1 UAT Scenarios

```gherkin
Scenario: 792a21l1a --parent routes to the configured parent
  Given ENGRAM_PARENT is set
  When the user runs engram show <ref> --parent
  Then the ref is resolved against the parent vault and not the local vault

Scenario: 792a21l1b show-chunk has no parent route
  Given ENGRAM_PARENT is set
  When the user runs engram show-chunk <id> --parent
  Then the command is rejected as an unknown flag, and a plain show-chunk of a missing chunk returns the local not-found error without contacting the parent

Scenario: 792a21l1c Without --parent a local note is returned as before
  Given ENGRAM_PARENT is set and the ref exists locally
  When the user runs engram show <ref>
  Then the local note is returned without contacting the parent

Scenario: 792a21l1d A local miss falls back to the parent
  Given ENGRAM_PARENT is set and the ref is not found locally
  When the user runs engram show <ref>
  Then the ref is resolved against the parent and the output is labeled as parent-sourced

Scenario: 792a21l1e A local miss with no parent is still an error
  Given ENGRAM_PARENT is not set and the ref is not found locally
  When the user runs engram show <ref>
  Then the command returns the same not-found error as before

Scenario: 792a21l1f --parent without ENGRAM_PARENT is an error
  Given ENGRAM_PARENT is not set
  When the user runs engram show <ref> --parent
  Then the command returns an error and performs no lookup

Scenario: 792a21l1g The parent-sourced label comes before the parent's header
  Given ENGRAM_PARENT is set and the parent holds a note that carries an xid, which the child does not
  When the user runs engram show <basename>
  Then the first line is the parent-sourced label and the second is the parent note's exchange-hash header
```

## 792a21m A vault with no parent behaves as it always did

Requirement: [792a21m A vault with no parent behaves as it always did](#792a21m-a-vault-with-no-parent-behaves-as-it-always-did) With `ENGRAM_PARENT` unset, `engram query` MUST make no network request, and its output and the
output shape of `learn`, `amend` and `resituate` MUST be unchanged by the merge, the dedupe and the offer machinery,
and no exchange state MUST be written. The one difference from before the merge is that `--limit` caps the item count
([792a7q Limit](./memory-query.md#792a7q-limit)).

Rationale: the exchange is additive. A set `ENGRAM_PARENT` adds a step to local behavior and never replaces it.

### 792a21m1 UAT Scenarios

```gherkin
Scenario: 792a21m1a No parent configured
  Given ENGRAM_PARENT is not set
  When the user runs engram query
  Then no network request is made and the output matches the single-source pipeline, including the --limit cap

Scenario: 792a21m1b A write with no parent exchanges nothing
  Given ENGRAM_PARENT is not set
  When the user runs engram learn, amend or resituate
  Then each behaves as before this capability and no exchange state is written
```

## 792a21n A local write is offered to the parent

Requirement: [792a21n A local write is offered to the parent](#792a21n-a-local-write-is-offered-to-the-parent) With `ENGRAM_PARENT` set, `engram learn fact`, `engram learn feedback` and `engram learn runbook`,
`engram amend` with any content flag (`--situation`, `--subject`, `--predicate`, `--object`, `--behavior`, `--impact`,
`--action`, `--done-when`, `--body`, `--red-flag`, `--trigger`), `engram resituate`, and `engram amend --clear-pending`
of a note that carries `offer.origin` (an accepted served
offer) MUST complete locally exactly as they do without a parent and MUST then queue the note as an offer. The last case MUST NOT be queued when the origin vault ID in
`offer.origin` is the configured parent's vault ID. A queued offer MUST be an amend-offer that targets the note's
primary parent link when the note has one for the configured parent's vault ID, and a learn-offer otherwise.
The following MUST NOT be offered: the bookkeeping amends (`--activate`, `--clear-pending` of any other note,
`--discard` with or without `--into`, and amends whose only flags are `--supersedes` or `--chunk-source`), identity
backfill, `engram learn qa` notes, notes that carry `skill_hash`, pending notes, and any write whose resulting
exchange hash equals the note's primary link hash. Offers MUST carry notes only and never transcript chunks.

Rationale: the write completes locally first, so it never fails because of the parent. Accepting a served offer counts as
a local learn at that level and is offered upward, which is what lets knowledge reach every level of the tree, and it is
skipped when the offer's own origin is the parent so that the single-parent tree has no loop. A pulled note that local
curation folds into a note of its own goes up once more, as an amend of that note, because the local addition is the
point of the fold, and the parent's curated result comes back down only on the next use. Stopping propagation at the first
level would defeat the personal, team and organization tree, and a fold marker that suppressed the bounce would lose the
local addition upstream.

### 792a21n1 UAT Scenarios

```gherkin
Scenario: 792a21n1a learn is written locally and offered
  Given ENGRAM_PARENT is set and the parent is reachable
  When the user runs engram learn fact with its arguments
  Then the note exists in the local vault as a live note, and the parent receives one learn-offer for it

Scenario: 792a21n1b A content amend of a linked note is an amend-offer
  Given local note L whose primary link names parent note P
  When the user runs engram amend --target L --object="..."
  Then the parent receives an offer whose offer.for is P

Scenario: 792a21n1c A runbook-field amend is offered
  Given a linked runbook R
  When the user runs engram amend --target R --red-flag="..."
  Then an amend-offer is queued, because R's exchange hash changed

Scenario: 792a21n1d A content amend of an unlinked note is a learn-offer
  Given local note L with no parent link
  When the user runs engram amend --target L --object="..."
  Then the parent receives a learn-offer carrying L's current content

Scenario: 792a21n1e resituate is offered
  Given a local note
  When the user runs engram resituate on it
  Then the parent receives an offer carrying the note's new situation

Scenario: 792a21n1f Bookkeeping amends stay local
  Given a local note L
  When the user runs engram amend --target L --activate, or --discard, or an amend with only --supersedes, or --clear-pending on a note without offer.origin
  Then no offer is queued or sent

Scenario: 792a21n1g Accepting a served offer propagates upward
  Given a vault that serves children and has its own ENGRAM_PARENT
  When it clears the pending marker on a served offer whose origin is a child vault
  Then a learn-offer for that note is queued to its own parent, carrying the accepted note's offer.path with this vault's ID appended

Scenario: 792a21n1h An accepted offer never returns to its origin vault
  Given an accepted note whose offer.origin names the configured parent's vault ID
  When its pending marker is cleared
  Then no offer is queued

Scenario: 792a21n1i Identity backfill stays local
  Given notes missing identity fields
  When the user runs engram update --backfill-identity
  Then no offer is queued for any of them

Scenario: 792a21n1j Registration notes and qa notes are never offered
  Given a note that carries skill_hash written by registration, or a note written by engram learn qa
  When it is written
  Then no offer is queued for it

Scenario: 792a21n1k A pending note is never offered
  Given a note that carries the pending marker
  When it is written or amended
  Then no offer is queued for it

Scenario: 792a21n1l An unchanged round trip is suppressed
  Given a note whose exchange hash equals its primary link hash
  When an amend leaves it so
  Then no offer is queued

Scenario: 792a21n1m A near-fold of a pulled note bounces up once
  Given local curation folds a pulled parent note's additional claim into local note L with a content amend
  When the amend completes
  Then exactly one offer for L is queued, and the resulting parent note is pulled down again only on a later activate, and only if its exchange hash differs

Scenario: 792a21n1n A link under another parent is not the parent's
  Given ENGRAM_PARENT was changed to another parent, and a note whose content equals the old parent's link hash
  When the user amends its content
  Then the offer is still queued, as a learn-offer with no offer.for, and no supersedes entry is translated through the old links
```

## 792a21o Only a change to offered content is a change

Requirement: [792a21o Only a change to offered content is a change](#792a21o-only-a-change-to-offered-content-is-a-change) The exchange hash of a note (the comparison that decides whether an offer is queued, whether a rejected
offer is sent again, whether a parent note is pulled again and whether two notes are the same) MUST change when
`type`, `situation`, `subject`, `predicate`, `object`, `behavior`, `impact`, `action`, `done_when`, `red_flags`,
`triggers` or the body text changes, and MUST NOT change when only identity, placement, `pending`, `tags`, `sources`,
`supersedes`, `xid`, `parent`, `aliases`, `offer` or the skill fields change. A comparison in which either hash has a
different or a missing version MUST be unknown and not changed: loop suppression and dedupe MUST fire only on equal,
and the pull-down skip, decline matching, the rejected-offer re-arm and the send check MUST treat unknown as not changed.

Rationale: placement, provenance and the exchange's own fields differ between a note and its counterpart without the
content having changed, so hashing them would report a change that is none. Treating a version change as unknown spares
the fleet a storm of spurious offers and pulls when the hash is revised. The hash itself, its comparison
and the classification of every frontmatter key are exchange-versioning's ([exchange-versioning](../technical/exchange-versioning.md)).

### 792a21o1 UAT Scenarios

```gherkin
Scenario: 792a21o1a Only offered fields move the hash
  Given a fact, feedback or runbook note
  When only a feedback note's impact, or only a runbook note's done_when, red_flags or triggers, is changed
  Then the note's exchange hash changes

Scenario: 792a21o1b Non-content fields do not move the hash
  Given a note
  When only its repo, user, pending, tags or parent link changes
  Then its exchange hash is unchanged

Scenario: 792a21o1c A hash version change is unknown, not changed
  Given a link that records a hash with an older version prefix
  When the parent note is activated again
  Then the pull-down treats it as not changed and writes nothing, and a later exchange of that link records the current-version hash
```

## 792a21p An offer is built for the parent vault

Requirement: [792a21p An offer is built for the parent vault](#792a21p-an-offer-is-built-for-the-parent-vault) An offer MUST be built from the note's current content at send time, and MUST NOT carry `target`,
`position` or `chunkSources`. Each `supersedes` entry that names a local note MUST be rewritten to that note's
primary-link parent basename, and MUST be omitted when there is none. The offer MUST declare the note's own `user` and
`repo` values, falling back to the sending process's detection only when they are empty, and MUST NOT carry `tags`. It
MUST carry `offer.origin` (the local vault ID and the note's `xid`), `offer.key` (derived from `offer.origin` and the
offered exchange hash), `offer.path` (the vault IDs the offer has passed through: the local ID for a direct offer, and
the accepted note's `offer.path` plus the local ID for a propagated one) and, for an amend-offer, `offer.for`.

Rationale: placement is each vault's own (Luhmann IDs are minted per vault), tags are the sending vault's own
vocabulary, and a note's declared author must survive the trip, so the drainer's identity is only a fallback. Building at
send time sends the latest content and never a stale snapshot.

### 792a21p1 UAT Scenarios

```gherkin
Scenario: 792a21p1a Placement is not sent
  Given a note learned with --target 12 --position child
  When it is offered
  Then the offer request carries no target and no position

Scenario: 792a21p1b Supersedes is translated or dropped
  Given an offered note that supersedes local note A, which is linked to parent note PA, and local note B, which is unlinked
  When it is offered
  Then the offer's supersedes names PA and does not name A or B

Scenario: 792a21p1c The note's author is declared, not the drainer's
  Given a note whose user: is alice, drained by a command run as bob
  When it is offered
  Then the offer declares user alice

Scenario: 792a21p1d The offer names its origin, key and path
  Given a direct offer, and an offer propagated from an accepted served offer
  When each is built
  Then the direct offer's offer.path is this vault's ID, the propagated one's is the accepted note's offer.path with this vault's ID appended, and each carries offer.origin and offer.key

Scenario: 792a21p1e An amend-offer names its target
  Given a note linked to parent note P under the configured parent
  When its amend is offered
  Then the offer carries offer.for P, and a note whose only link is covered or under another parent is offered as a learn-offer

Scenario: 792a21p1f Tags are not offered
  Given a note that carries vocabulary tags
  When it is offered
  Then the offer carries no tags
```

## 792a21q An offer is queued and never lost offline

Requirement: [792a21q An offer is queued and never lost offline](#792a21q-an-offer-is-queued-and-never-lost-offline) Every offerable write MUST stamp the note's `xid` when it has none and MUST add or refresh the
note's outbox entry, keyed by `xid`, under the vault lock and in the same critical section as the note write, so the
outbox holds at most one entry per note. The local write MUST succeed whether or not the parent is reachable. An entry
whose note no longer exists, or now carries the pending-offer marker, MUST be dropped when it is sent, and renaming a
note MUST NOT orphan its entry.

Rationale: the outbox is `<vault>/.engram/outbox.json` ([792a2k Exchange state outside tracked files](./vault-identity.md#792a2k-exchange-state-outside-tracked-files)). Keying it by `xid` makes a rename harmless, where a
basename key would lose the entry. An offer is built when it is sent, so offline learn and amend coalesce into one offer.
A snapshot directory per offer would send stale content and duplicates, a line-per-offer file could not coalesce without
rewriting, and failing the local write when the parent is down would contradict "local writes always succeed".

### 792a21q1 UAT Scenarios

```gherkin
Scenario: 792a21q1a The parent is unreachable during learn
  Given the parent is unreachable
  When the user runs engram learn fact with its arguments
  Then the command exits zero, the note is live locally, the outbox has one entry for it, and stderr carries one warning naming the queued-offer count

Scenario: 792a21q1b Offline learn then amend coalesce
  Given a note learned and then amended twice while the parent is unreachable
  When the parent becomes reachable and the outbox drains
  Then exactly one offer is sent for that note and it carries the note's latest content

Scenario: 792a21q1c A renamed note keeps its entry
  Given a queued note renamed by Luhmann reparenting before the outbox drains
  When the outbox drains
  Then the drain sends the renamed note's offer and records the link on the renamed note

Scenario: 792a21q1d A discarded note's entry is dropped
  Given a queued note discarded locally before the outbox drains
  When the outbox drains
  Then it sends nothing for it and removes its entry

Scenario: 792a21q1e A queued note is stamped with an exchange ID
  Given a note with no xid
  When it is first queued
  Then it gains an xid that it keeps afterwards

Scenario: 792a21q1f A pending note's entry is dropped
  Given a queued note that has since become pending
  When the outbox drains
  Then it sends nothing for it and removes its entry

Scenario: 792a21q1g The first offer creates the exchange state in the right order
  Given a vault with no vault ID and no .engram directory
  When the first offer is queued
  Then the vault ID and its location record are stamped before .engram exists

Scenario: 792a21q1h A note is written even when no exchange ID can be made
  Given a host where no xid can be minted
  When the user runs engram learn fact with its arguments
  Then the note is written and nothing is queued
```

## 792a21r The outbox drains after the parent was reached, without the lock

Requirement: [792a21r The outbox drains after the parent was reached, without the lock](#792a21r-the-outbox-drains-after-the-parent-was-reached-without-the-lock) After a command has successfully contacted the parent (`learn`, a content `amend`, `resituate`,
`query` after the parent answers, `activate` after a pull-down, or `update`), it MUST send the queued entries in
first-queued order while holding no vault lock, and MUST then take the lock again, re-read the outbox from disk and merge
the outcomes, keeping entries queued meanwhile. A receipt for a note that no longer exists MUST be discarded, and a note
amended while its offer was in flight MUST stay queued. A transport error, a timeout or a 5xx MUST stop the drain,
keep the remaining entries and record the attempt count and last error on the failed entry. A 4xx MUST mark that entry
`rejected` together with its exchange hash and let the drain continue, and a rejected entry MUST NOT be sent again
until its note's exchange hash changes. A payload that cannot be built locally MUST keep its entry and record the error
without contacting the parent or backing off. A drain with an empty outbox MUST do nothing. When the parent's vault ID is not yet cached, the drain MUST learn it
from the parent before it sends, and MUST keep the offers queued and warn once when that probe is answered with a 4xx,
reports no vault ID (the parent is too old), or is followed by an unreachable parent.

Rationale: a drain that held the lock while it waited on the network would block every local write for as long as the
parent took to answer. Re-reading the outbox before merging is what lets another process queue an entry mid-drain
without losing it.

### 792a21r1 UAT Scenarios

```gherkin
Scenario: 792a21r1a Query drains a backlog
  Given the outbox holds three entries
  When engram query succeeds against the parent
  Then the three offers are sent in first-queued order and the outbox is empty afterwards

Scenario: 792a21r1b A transport failure stops the drain
  Given an outbox of three entries and a second send that fails with a connection error
  When the outbox drains
  Then the first entry is removed, the second and third remain in order, and the second records attempts and last_error

Scenario: 792a21r1c A rejected offer does not block the queue
  Given the parent answers the first entry with a 4xx
  When the outbox drains
  Then that entry is marked rejected, the next entries are still sent, and the rejected entry is not sent again until its note's exchange hash changes

Scenario: 792a21r1d A concurrent enqueue survives the drain
  Given another process queues a new entry while a drain is sending
  When the drain merges its results
  Then that entry is still in the outbox

Scenario: 792a21r1e A note amended in flight stays queued
  Given a note amended while its offer was being sent
  When the receipt is recorded
  Then the note keeps its entry for the newer content

Scenario: 792a21r1f A note deleted in flight drops its receipt
  Given a note deleted while its offer was being sent
  When the drain merges
  Then the receipt is discarded and the entry is dropped

Scenario: 792a21r1g An unbuildable payload does not stop the drain
  Given an entry whose offer cannot be built locally
  When the outbox drains
  Then the error is recorded on the entry, the entry stays queued, the later entries are still sent, and the parent is not backed off for it

Scenario: 792a21r1h An empty outbox touches nothing
  Given an empty outbox
  When a command that drains runs
  Then no send and no write happens

Scenario: 792a21r1i Sends hold no lock
  Given a drain in progress
  When an offer is being sent
  Then the vault lock is not held

Scenario: 792a21r1j An offer withdrawn at send time is dropped
  Given a queued offer whose origin vault is the parent, or whose content the parent already holds
  When the drain builds it
  Then it is not sent, its entry is dropped, and the entry stays if the note changed meanwhile

Scenario: 792a21r1k A vault that fails its location check sends nothing
  Given a cloned vault whose location record is missing
  When a command drains
  Then nothing is sent and the location warning is printed

Scenario: 792a21r1l A failed vault-ID probe keeps the offers queued
  Given an uncached parent vault ID and a parent that answers the probe with a 4xx, reports no vault ID, or accepts the probe and then cannot be reached for the offer
  When the outbox drains
  Then the offers stay queued and one warning is printed
```

## 792a21s An unreachable parent is backed off

Requirement: [792a21s An unreachable parent is backed off](#792a21s-an-unreachable-parent-is-backed-off) After a transport failure, a timeout or a 5xx against the parent, the vault MUST record a retry time that
grows exponentially with the consecutive failures, from 30 seconds and capped at 15 minutes. Until that time every
command except `engram update` MUST skip parent contact, so that queries return local results and offers are only queued,
and MUST print exactly one warning. A request to the parent MUST use a connect timeout of at most 3 seconds. Any answer
from the parent, a 4xx or a loop refusal included, counts as a successful contact and MUST reset the failure count. A
2xx answer to an offer whose body cannot be decoded as a receipt is not an answer and counts as a failure. A parent
other than the one backed off MUST NOT be affected.

Rationale: without a backoff every command would wait up to the full connect timeout while a parent is down.
`update` always tries so that a cleared outage is noticed. A 4xx or a refusal is an answer from a reachable parent.

### 792a21s1 UAT Scenarios

```gherkin
Scenario: 792a21s1a Commands inside the backoff window make no request
  Given a query that failed to reach the parent
  When a second query runs 10 seconds later
  Then it makes no parent request, returns local results, and prints one warning

Scenario: 792a21s1b update always tries
  Given a backoff window is active
  When the user runs engram update
  Then it attempts the parent and drains on success

Scenario: 792a21s1c The delay grows and is capped
  Given consecutive failures against the parent
  When each retry time is recorded
  Then the delay is 30 seconds after the first failure, doubles with each, and never exceeds 15 minutes

Scenario: 792a21s1d A success resets the count
  Given a parent that has failed twice
  When a contact succeeds
  Then the failure count is zero and the next failure backs off for 30 seconds

Scenario: 792a21s1e A different parent URL is not backed off
  Given a backoff recorded for one parent URL
  When a command contacts a different parent URL
  Then it is not skipped

Scenario: 792a21s1f The window ends
  Given a backoff whose retry time has passed
  When a command contacts the parent
  Then the contact is allowed
```

## 792a21t A retried offer is idempotent

Requirement: [792a21t A retried offer is idempotent](#792a21t-a-retried-offer-is-idempotent) A retry of an offer that the parent already holds MUST NOT create a second pending note on the
parent ([792a20g A served learn finds the note an offer belongs to](./vault-serving.md#792a20g-a-served-learn-finds-the-note-an-offer-belongs-to)): the offer's `offer.key` MUST be derived from `offer.origin` and the offered exchange hash, so that the
same content from the same note always carries the same key.

Rationale: an offer whose response is lost is sent again, and the parent must answer with the receipt of the note it
already has.

### 792a21t1 UAT Scenarios

```gherkin
Scenario: 792a21t1a Lost response, then retry
  Given the parent stored an offer but the response was lost
  When the entry is sent again unchanged
  Then the parent returns the receipt of the pending note it already has, and holds one pending note for that offer
```

## 792a21u A receipt links the local note to its counterpart

Requirement: [792a21u A receipt links the local note to its counterpart](#792a21u-a-receipt-links-the-local-note-to-its-counterpart) When the parent returns an offer receipt, the local note MUST record the parent's vault ID and
MUST set its primary link to `{note, via: offered, hash}`, where `note` is the resolved target basename when the receipt
names one and otherwise the receipt's basename, and `hash` is the receipt's `stored_hash`, which is the hash the parent
stored and not the one the child computed. Recording the link MUST be a frontmatter-only rewrite under the vault lock, and
MUST NOT re-embed the note or re-stamp `repo`, `user` or `vault`, so the note keeps its declared author.

Rationale: the link is what later lets a query show the note once and a pull-down skip it. The parent's stored hash is
what a future comparison sees from the parent's side, and a rewrite that re-embedded would change nothing the vectors
depend on.

### 792a21u1 UAT Scenarios

```gherkin
Scenario: 792a21u1a Receipt links the note
  Given the parent answers a learn-offer for local note L with basename 1100.2026-09-27.x
  When the receipt is recorded
  Then L's primary link is {note: 1100.2026-09-27.x, via: offered, hash: <stored_hash>} under the parent's vault ID, and L's sidecar vector is unchanged

Scenario: 792a21u1b Re-link to the resolved target
  Given an amend-offer's receipt names a resolved target E different from L's current primary
  When the receipt is recorded
  Then L's primary link becomes E

Scenario: 792a21u1c A receipt keeps the note's author
  Given a note whose user: is alice and a command run as bob
  When the receipt is recorded on the note
  Then its repo:, user: and vault: are unchanged

Scenario: 792a21u1d A receipt under another parent starts over
  Given a note linked under a parent vault other than the one that answered
  When the receipt is recorded
  Then the note's links are those of the answering parent only
```

## 792a21v A reply that is not a valid receipt is not recorded

Requirement: [792a21v A reply that is not a valid receipt is not recorded](#792a21v-a-reply-that-is-not-a-valid-receipt-is-not-recorded) A receipt that lacks a vault ID or a basename MUST stop exchange with an error saying the parent is
too old, and MUST leave the entry queued. A receipt whose `vault_id` is not 32 lowercase hexadecimal characters, or
whose `basename` or `for` is not a Luhmann basename free of `/`, `\` and `|`, MUST be treated as an undecodable reply:
nothing is recorded on the note, the entry stays queued with the failure recorded, and the parent is backed off as for an
outage ([792a21s An unreachable parent is backed off](#792a21s-an-unreachable-parent-is-backed-off)). A 409 loop refusal whose
`vault_id` is malformed MUST count as a refusal without a vault ID, and a malformed vault ID MUST never be cached as the
parent's. A receipt that fails to record for another reason, a write error for example, MUST keep the entry queued with
the failure recorded.

Rationale: the parent is another machine's input. A basename containing a separator would become a path, and a vault ID
that is not the expected shape would poison the cache that decides which links count. A receipt without a vault ID comes
from a parent that predates the exchange, and the same check detects it at pull-down. A merged query is not refused for
this reason, it only degrades: the merge keeps the old parent's results and its dedupe finds no keys to match.

### 792a21v1 UAT Scenarios

```gherkin
Scenario: 792a21v1a A pre-change parent is detected
  Given a receipt that carries no vault_id
  When the receipt is classified
  Then the command reports that the parent is too old, and the entry stays queued

Scenario: 792a21v1b A malformed receipt is not recorded
  Given a receipt whose basename contains | or /, or whose vault_id is not 32 lowercase hexadecimal characters
  When the receipt is classified
  Then the local note gains no parent link, the entry stays queued with the failure recorded, and the parent is backed off

Scenario: 792a21v1c A failed write keeps the entry
  Given recording a valid receipt on the note fails
  When the drain reports it
  Then the entry stays queued with the error and is sent again later, idempotently
```

## 792a21w A receipt the note refuses needs attention

Requirement: [792a21w A receipt the note refuses needs attention](#792a21w-a-receipt-the-note-refuses-needs-attention) When the local note refuses a receipt because its frontmatter cannot take the edit (its `parent:`
carries a YAML anchor, or its frontmatter does not decode), the note MUST be left untouched and the entry MUST move to the
`attention` state with the receipt and the exchange hash that was sent kept. The drain MUST print one warning that names
the note and the reason, at that transition only, and an `attention` entry MUST NOT be sent to the parent again. Each
later command that drains MUST retry recording the kept receipt on the note before its backoff and reachability checks,
without contacting the parent and without a warning, so a fixed note takes its receipt even while the parent is backed
off or unreachable. A kept receipt MUST be recorded only when its vault ID is the configured parent's cached vault ID,
and otherwise MUST be discarded with one warning and the entry queued for the current parent. When the retry succeeds the
entry MUST finish as an accepted entry does: removed, or queued again when the note's exchange hash changed since it was
sent. An entry whose note is gone or pending MUST be dropped, as for any other state.

Rationale: the parent accepted the offer, so sending it again would be wrong, and the note cannot take its link until
its owner fixes the frontmatter. Keeping the receipt, rather than the offer, is what makes the repair automatic.

### 792a21w1 UAT Scenarios

```gherkin
Scenario: 792a21w1a A refused receipt needs attention
  Given the parent accepts the offer of local note L, and L's parent: carries an anchor that another key aliases
  When the receipt is applied
  Then L is byte-identical, the entry is in the attention state with the reason and the receipt recorded, and stderr carries one warning naming L and parent

Scenario: 792a21w1b An attention entry is not sent again
  Given an attention entry for L, L unchanged
  When the outbox drains twice more
  Then nothing is sent for L, no further warning is printed, and the entry stays in the attention state

Scenario: 792a21w1c A kept receipt is recorded while the parent is backed off
  Given the anchor is removed from L's parent: while the parent is inside its backoff window
  When another command drains
  Then the kept receipt is recorded on L, the entry is done, and the parent receives no request

Scenario: 792a21w1d A kept receipt from another parent is discarded
  Given an attention entry that keeps a receipt whose vault ID is not the configured parent's
  When a command drains
  Then L is not written, one warning says the kept receipt is discarded, and the entry is queued for the current parent with no kept receipt

Scenario: 792a21w1e An attention entry resumes once the note is fixed
  Given the anchor is removed from L's parent:
  When the outbox drains
  Then the kept receipt is recorded on L without contacting the parent, and the entry is removed, or queued again when L's exchange hash changed since it was sent

Scenario: 792a21w1f An attention entry without a receipt is sent
  Given an attention entry that holds no receipt
  When the outbox drains
  Then its offer is built and sent as a queued one

Scenario: 792a21w1g A retry failure is reported
  Given an attention entry whose kept receipt still cannot be recorded for a reason other than the frontmatter
  When the outbox drains
  Then the failure is reported and the entry stays in attention

Scenario: 792a21w1h An attention entry of a deleted note is dropped
  Given an attention entry whose note was deleted
  When the outbox drains
  Then the entry is dropped
```

## 792a21x An offer with no author waits

Requirement: [792a21x An offer with no author waits](#792a21x-an-offer-with-no-author-waits) An offer that cannot be built because the note has no `user:` and no user identity is detected MUST
NOT be sent, because the parent would reject `user: ""`. Its entry MUST move to the `attention` state with the reason
"cannot offer: no user identity detected; set git user.email", and the drain MUST print one warning naming the note, at
that transition only. Each later drain MUST try to build the offer again, without contacting the parent and without a
warning while it still cannot, and as soon as the offer can be built the entry MUST return to `queued`, so that a failed
send leaves it queued and counted and not listed as needing attention.

Rationale: an offer nobody can be asked about is refused by the parent ([792a20e A served learn is attributed to the identity the caller declares](./vault-serving.md#792a20e-a-served-learn-is-attributed-to-the-identity-the-caller-declares)), and holding it back with a reason is
kinder than a 4xx that marks it rejected.

### 792a21x1 UAT Scenarios

```gherkin
Scenario: 792a21x1a No user identity holds the offer
  Given an entry whose note has no user: on a host where user detection resolves empty
  When the outbox drains and then drains twice more
  Then nothing is sent, the entry is in attention with the no-identity reason, and one warning was printed; after user detection starts to resolve, the next drain sends the offer
```

## 792a21y A looping offer is refused

Requirement: [792a21y A looping offer is refused](#792a21y-a-looping-offer-is-refused) When a parent refuses an offer with a 409 that marks a loop, because its own vault ID is already on
the offer's `offer.path`, and the refusing vault is not this vault, the entry MUST be marked `rejected`, the drain MUST
continue, one warning MUST name the refusing vault, and the refusal MUST NOT back the parent off: it is an answer, so it
resets the failure count ([792a21s An unreachable parent is backed off](#792a21s-an-unreachable-parent-is-backed-off)). A 409 without that mark
MUST be an ordinary rejection. When the refusing vault is this vault itself, or a receipt reports this vault's own ID,
no entry MUST be sent or marked rejected, the parent's vault ID MUST be cached, the drain MUST stop, and one
warning MUST name `engram vault-id --regenerate`. A cached parent vault ID equal to
this vault's own MUST make the drain send nothing.

Rationale: a cycle in a misconfigured tree (A to B to C to A) is caught by the path an offer carries, and it is a real
fault that rejecting every retry would only repeat. A vault that is its own parent is a copy mistake that the user can
repair, and it must not act as its own parent, so nothing is sent or rejected ([792a2q A vault does not exchange with itself](./vault-identity.md#792a2q-a-vault-does-not-exchange-with-itself)).

### 792a21y1 UAT Scenarios

```gherkin
Scenario: 792a21y1a A cycle rejects the entry and continues
  Given a parent that refuses an offer as a loop and is not this vault
  When the outbox drains
  Then that entry is rejected, the drain continues, one warning names the refusing vault and not --regenerate, and the parent is not backed off

Scenario: 792a21y1b An unmarked conflict is an ordinary rejection
  Given a parent that answers 409 without the loop mark
  When the outbox drains
  Then the entry is rejected and the drain continues

Scenario: 792a21y1c A self-parent refusal waits
  Given a parent that refuses an offer as a loop and reports this vault's own ID
  When the outbox drains
  Then no entry is sent or rejected, the parent's ID is cached, the drain stops and one warning names engram vault-id --regenerate

Scenario: 792a21y1d A self-parent receipt is not recorded
  Given a receipt that reports this vault's own ID
  When the outbox drains
  Then nothing is recorded, the drain stops, the entry is neither sent nor rejected and one warning names engram vault-id --regenerate

Scenario: 792a21y1e A cached self-parent sends nothing
  Given the cached parent vault ID equals this vault's own
  When the outbox drains
  Then nothing is sent and one warning names engram vault-id --regenerate
```

## 792a21za The update report shows the outbox

Requirement: [792a21za The update report shows the outbox](#792a21za-the-update-report-shows-the-outbox) `engram update` MUST include a notify-only notice when the outbox is non-empty or a backoff is
active. The notice MUST give the queued-entry count, the age of the oldest entry, each rejected entry with its error, each
entry in the `attention` state with its note and reason, and the backoff's retry time. The queued-entry count, in the
notice and in the backoff and unreachable-parent warnings, MUST NOT count `rejected` or `attention` entries. An idle
vault MUST have no notice.

Rationale: a stuck outbox is silent otherwise, and `update` is where the user already looks for notices.

### 792a21za1 UAT Scenarios

```gherkin
Scenario: 792a21za1a Update reports a stuck outbox
  Given the outbox holds two queued entries and one rejected entry
  When the user runs engram update
  Then its report includes one notice with the count, the oldest age and the rejected entry's error

Scenario: 792a21za1b Update reports an entry that needs attention
  Given the outbox holds one queued entry and one attention entry for note L
  When the user runs engram update
  Then its notice counts one queued offer and one that needs attention, and names L with the reason

Scenario: 792a21za1c An idle vault has no notice
  Given an empty outbox and no backoff
  When the user runs engram update
  Then no outbox notice is printed

Scenario: 792a21za1d Update drains through a backoff
  Given a backlog and an active backoff window
  When the user runs engram update and the parent answers
  Then the backlog drains and the next update reports an idle outbox
```

## 792a21zb A note with CRLF line endings exchanges as its LF form

Requirement: [792a21zb A note with CRLF line endings exchanges as its LF form](#792a21zb-a-note-with-crlf-line-endings-exchanges-as-its-lf-form) Every reader that gates an exchange operation on a note's frontmatter MUST read the note as its LF
form (each `\r\n` converted to `\n`), so a note with CRLF line endings takes part in exchange exactly as its LF form
does. These readers are the exchange hash, the exchange-field decode behind the header of `engram show`, the
pending-offer marker check behind query exclusion and the served raw `show`, the outbox and served-learn note scan, offer
classification at write time, the offer payload, the offer receipt, the pull-down envelope parse and the decline record,
and activate's parent-link re-check. None of them MUST write a note only to convert it, and a receipt recorded on a CRLF
note MUST write it as LF in that same write and rebuild its sidecar.

Rationale: a vault edited on Windows, or by an editor that rewrites line endings, must not fall out of the exchange or
make every note look changed (the curate side of this rule is [curate](./curate.md)).

### 792a21zb1 UAT Scenarios

```gherkin
Scenario: 792a21zb1a A CRLF note hashes as its LF form
  Given a note whose frontmatter and body end in \r\n
  When its exchange hash is computed
  Then it equals the exchange hash of its LF form

Scenario: 792a21zb1b A queued CRLF note is sent
  Given an outbox entry whose note has CRLF line endings
  When the outbox drains
  Then the offer is sent, the receipt is recorded, and the note is written as LF with a fresh sidecar

Scenario: 792a21zb1c A CRLF pending offer stays pending
  Given a note that carries pending: true and has CRLF line endings
  When engram query runs and the note is requested over the served raw show
  Then query excludes it and the served raw show refuses it as a pending offer

Scenario: 792a21zb1d A CRLF parent note pulls down
  Given a parent that serves a note whose envelope content has CRLF line endings
  When the user activates it
  Then it is pulled down as an LF pending copy linked at the parent's hash
```

## 792a21zc Activate resolves a ref locally first, then against the parent

Requirement: [792a21zc Activate resolves a ref locally first, then against the parent](#792a21zc-activate-resolves-a-ref-locally-first-then-against-the-parent) `engram activate --note <ref>` MUST treat a ref as a local hit when the note's `.md` file exists in
the local vault, whether or not it has a sidecar, and on a local hit MUST bump the note's sidecar `LastUsed` as [792a4n Activation](./note-revision.md#792a4n-activation)
says. On a local miss with `ENGRAM_PARENT` set, or for any ref when `--parent` is given, it MUST pull the ref down from the
parent ([792a21zd A pulled-down note is an exact-content local pending offer](#792a21zd-a-pulled-down-note-is-an-exact-content-local-pending-offer)), but only when the ref is a basename or `<basename>.md`: a bare Luhmann ID MUST NOT be resolved against
the parent, and with `--parent` a path ref MUST be refused as not sent and never as not found. `--parent` without
`ENGRAM_PARENT` configured MUST be an error and no lookup MUST be performed. `activate` MUST report each ref it could not
activate on stderr and MUST exit non-zero if any ref failed.

Rationale: Luhmann IDs are minted per vault, so the same ID in the parent names an unrelated note, and resolving a bare
one there could pull down the wrong note. A local hit never contacts the parent, so a note the vault already holds costs
no network.

### 792a21zc1 UAT Scenarios

```gherkin
Scenario: 792a21zc1a A parent-only ref pulls down
  Given ENGRAM_PARENT is set and the note exists only in the parent
  When the user runs engram activate --note <basename>.md
  Then the parent note is pulled down as a local pending offer

Scenario: 792a21zc1b A local hit never contacts the parent
  Given the ref's note file exists locally and has no parent link
  When the user runs engram activate --note <ref>
  Then the local sidecar's LastUsed is bumped, a note without a sidecar still counts, and no parent request is made

Scenario: 792a21zc1c Bare Luhmann IDs stay local
  Given no local note has the ID 1100
  When the user runs engram activate --note 1100, with or without .md, with or without --parent
  Then no parent request is made and the ref is reported as failed

Scenario: 792a21zc1d Partial failure is visible
  Given two refs, one of which cannot be resolved locally or on the parent
  When the user runs engram activate with both
  Then stderr names the failed ref and the command exits non-zero

Scenario: 792a21zc1e --parent without a parent configured
  Given ENGRAM_PARENT is unset
  When the user runs engram activate --parent --note x
  Then the command errors and no note is read or written

Scenario: 792a21zc1f --parent pulls even on a local hit
  Given ENGRAM_PARENT is set and a local file of the ref's name exists
  When the user runs engram activate --parent --note <ref>
  Then the ref is resolved against the parent

Scenario: 792a21zc1g --parent refuses a path ref
  Given ENGRAM_PARENT is set
  When the user runs engram activate --parent --note with a ref that is a path
  Then the ref is reported as not sent and nothing is fetched

Scenario: 792a21zc1h A miss with no parent fails the ref
  Given ENGRAM_PARENT is unset and the ref is not local
  When the user runs engram activate --note <ref>
  Then the ref is reported as failed and nothing is fetched
```

## 792a21zd A pulled-down note is an exact-content local pending offer

Requirement: [792a21zd A pulled-down note is an exact-content local pending offer](#792a21zd-a-pulled-down-note-is-an-exact-content-local-pending-offer) A pull-down MUST fetch the parent note as the raw envelope of [792a20k A served show returns a note, and a raw envelope on request](./vault-serving.md#792a20k-a-served-show-returns-a-note-and-a-raw-envelope-on-request) while holding no vault lock,
and MUST reject a response that is not a valid envelope (the parent is too old). It MUST also reject, writing nothing, an
envelope whose `vault_id` is not 32 lowercase hexadecimal characters, whose `basename` is not a Luhmann basename free
of `/`, `\` and `|`, or whose `exchange_hash` does not match its content, and a fetched note whose type is not `fact`,
`feedback` or `runbook`. Under the vault lock it MUST write a new local note with a fresh top-level Luhmann ID and its
own `xid`, the parent's body unchanged so that its exchange hash equals the envelope's, the pending-offer marker,
`parent.vault` set to the envelope's vault ID with a primary link `{note: <envelope basename>, via: pulled, hash: <envelope
exchange hash>}`, and `parent.author` set to the parent note's `repo`, `user` and `vault`. Its top-level `repo`, `user` and
`vault` MUST be stamped locally. It MUST NOT carry the parent's own `parent`, `aliases`, `offer`, `xid`, `skill_hash`,
`skill_key`, `skill_source`, `sources`, `supersedes`, `tags` or `vocab_version`. It MUST be embedded on write and tagged
with the local vocabulary, and the command MUST print its path.

Rationale: a pulled note is never written live: curation judges it before it counts as knowledge, so a pull-down is an
offer in the other direction ([curate](./curate.md)). The exact content keeps its exchange hash equal to the parent's, which is what
lets the skip rule and the dedupe recognize the pair. The parent's identity fields, tags and chunk provenance mean
nothing in this vault, and local vocabulary reassigns the tags. A `parent:` path prefix in payloads would break consumers
such as a bare `engram show <basename>`, and the rendered `show` output cannot carry a raw envelope because it is altered.

### 792a21zd1 UAT Scenarios

```gherkin
Scenario: 792a21zd1a Pulled note is pending and linked
  Given parent note P
  When P is pulled down
  Then a new local note exists carrying pending true, a via pulled primary link to P with P's exchange hash, and parent.author equal to P's authorship; it is excluded from local query results, and the next query payload has pending_offers true

Scenario: 792a21zd1b Remote identity and registration fields are stripped
  Given a fetched parent note that carries skill_hash, skill_key, aliases, xid or its own parent link
  When it is pulled down
  Then the local pending note carries none of those values and its parent link is the one this pull-down set

Scenario: 792a21zd1c Chunk provenance does not travel
  Given a fetched parent note that carries sources chunk IDs
  When it is pulled down
  Then the local note carries no sources

Scenario: 792a21zd1d A pre-change parent is detected
  Given a parent that answers the raw request with a rendered note instead of an envelope
  When the user activates a parent note
  Then the pull-down fails with an error saying the parent is too old and nothing is written

Scenario: 792a21zd1e A malformed envelope is refused
  Given an envelope whose vault_id is not 32 lowercase hexadecimal characters or whose basename holds a separator or |
  When the user activates a parent note
  Then nothing is written and the bad vault ID is never cached

Scenario: 792a21zd1f A hash that does not match is refused
  Given an envelope whose exchange_hash does not match its content
  When the user activates a parent note
  Then nothing is written

Scenario: 792a21zd1g A note of another type is refused
  Given a fetched note whose type is not fact, feedback or runbook
  When the user activates it
  Then nothing is written

Scenario: 792a21zd1h A pending parent note is never pulled
  Given a parent whose note is pending
  When the user activates it
  Then nothing is pulled down, and a live parent note is pulled

Scenario: 792a21zd1i The pulled note is embedded and does not keep the parent's tags
  Given a fetched parent note that carries vocabulary tags
  When it is pulled down
  Then its sidecar exists and the parent's tags are not on the local note
```

## 792a21ze A pull-down is idempotent, remembers declines and cannot loop

Requirement: [792a21ze A pull-down is idempotent, remembers declines and cannot loop](#792a21ze-a-pull-down-is-idempotent-remembers-declines-and-cannot-loop) Under the vault lock a pull-down MUST re-check its skip rule and write nothing when a local note,
live or pending, has any link (`offered`, `pulled` or `covered`) to the envelope's basename or one of its aliases and that
link's hash equals the envelope's exchange hash (bumping the sidecar `LastUsed` of that note instead when it is live),
or when the vault's declined-pull record holds that exchange hash, under the envelope's vault ID, for the envelope's
basename or any alias in the fetched content. A bare `engram amend --discard` of a note whose primary link is `via:
pulled` MUST add that basename and hash, keyed by the link's parent vault ID, to the declined-pull record, and MUST fail
and keep the note when the decline cannot be recorded, except in a vault that has no parent configured and no vault ID,
where it MUST record nothing and create neither a vault ID nor `.engram/`. A parent note whose exchange hash changed since
the last pull, or since it was declined, MUST arrive as another pending offer. A pulled-down note MUST NOT be offered back
to the parent unless its content is later changed locally. Two activates of the same parent note racing on one vault MUST
write one copy, and any sequence of activate, clear-pending, discard and fold on an unchanged parent note MUST NOT leave
more than one local note linked to it.

Rationale: re-pulling an unchanged note on every use would fill the vault with copies. A decline is keyed by the parent's
vault ID and the hash, so a decline under one parent never suppresses a note of another, and it covers the aliases so that
a rename on the parent does not bring a declined note back.

### 792a21ze1 UAT Scenarios

```gherkin
Scenario: 792a21ze1a Re-activating an unchanged parent note
  Given parent note P activated twice with no change to P in between
  When the second activate completes
  Then the local vault gained exactly one note for P

Scenario: 792a21ze1b A covered parent note is not pulled again
  Given local note L has a primary link and a pulled copy of P was folded into L with --discard --into L
  When P is activated again
  Then L has a covered link to P, nothing is written, and L's LastUsed is bumped

Scenario: 792a21ze1c A declined pull is remembered
  Given a pulled copy of P discarded outright
  When P is activated again unchanged
  Then nothing is written

Scenario: 792a21ze1d A decline survives a parent rename
  Given a declined parent note renamed on the parent, with its old basename now in its aliases
  When it is activated again unchanged
  Then nothing is written

Scenario: 792a21ze1e A changed parent note arrives as a new offer
  Given P pulled down and accepted, and then changed on the parent
  When P is activated again
  Then a second local pending offer for P exists, carrying the new exchange hash

Scenario: 792a21ze1f Accepting a pulled note sends nothing up
  Given a pulled note
  When curation clears its pending marker
  Then no offer is queued for it

Scenario: 792a21ze1g A decline under another parent does not apply
  Given a decline recorded under another parent vault for the same basename and hash
  When the note is activated under the current parent
  Then it is pulled down

Scenario: 792a21ze1h A decline that cannot be recorded keeps the note
  Given a bare discard of a pulled note whose decline cannot be written
  When the discard runs
  Then it fails and the note is kept

Scenario: 792a21ze1i Discarding a pulled note with no vault ID records nothing
  Given a vault with no parent and no vault ID
  When a pulled note is discarded
  Then it is deleted, no decline is recorded, and no vault ID or .engram directory is created

Scenario: 792a21ze1j A local edit of an accepted pulled note is offered
  Given a pulled note accepted by clearing its pending marker
  When its content is later edited locally
  Then the edit is queued as an amend-offer naming the parent note

Scenario: 792a21ze1k Racing activates write one copy
  Given two activates of the same parent note racing on one vault
  When both complete
  Then one copy is written

Scenario: 792a21ze1l No sequence leaves two linked copies
  Given any sequence of activate, clear-pending, discard and fold on an unchanged parent note
  When the sequence completes
  Then at most one local note is linked to the parent note
```

## 792a21zf Activating a linked note re-checks its parent note

Requirement: [792a21zf Activating a linked note re-checks its parent note](#792a21zf-activating-a-linked-note-re-checks-its-parent-note) When `engram activate` resolves a ref to a local note that has parent links under the configured
parent's vault ID, and `ENGRAM_PARENT` is set, it MUST, after the local bump and while holding no vault lock, fetch each
linked parent note's raw envelope through the same gated, backoff-aware parent contact a pull-down uses, and then apply
the pull-down's write step under the lock: when the parent note's current exchange hash equals the link's hash, or is
declined, nothing MUST be written, and otherwise the parent note MUST arrive as a new local pending offer exactly as a
pull-down writes one. A link under another parent vault MUST NOT be written: when that parent's vault ID is already
cached the link MUST NOT be contacted at all, and otherwise the envelope MUST still be fetched and its reply discarded
once its vault ID differs from the link's. The re-check MUST NOT send the best-effort `activate` request to the parent,
and its failure (the parent unreachable, backed off or the note gone there) MUST NOT fail the local activate.

Rationale: the merged query shows a linked local note in place of its parent note ([792a21j A linked or identical note appears once, as the local copy](#792a21j-a-linked-or-identical-note-appears-once-as-the-local-copy)), so a parent-side change
reaches the child only through the local note's use.

### 792a21zf1 UAT Scenarios

```gherkin
Scenario: 792a21zf1a A changed parent note arrives through its local copy
  Given P pulled down and accepted as local note L, then changed on the parent, and the merged query returns L in P's place
  When the user activates L
  Then a new local pending offer for P exists, carrying P's new exchange hash, and activating L again writes nothing more

Scenario: 792a21zf1b An unchanged parent note writes nothing
  Given a linked local note L whose parent note's exchange hash equals L's link hash
  When the user activates L
  Then nothing is written and only L's LastUsed changes

Scenario: 792a21zf1c A declined update is not pulled again
  Given the pending offer a re-check pulled down was discarded outright
  When L is activated again with the parent note unchanged since
  Then nothing is written

Scenario: 792a21zf1d An unreachable parent does not fail the local activate
  Given a linked local note and an unreachable parent
  When the user activates the note
  Then the command succeeds, the local LastUsed is bumped, and the failure is recorded for backoff

Scenario: 792a21zf1e A link under another parent is never checked
  Given a linked local note whose link is under another parent vault whose ID is cached
  When the user activates the note
  Then that link is not contacted

Scenario: 792a21zf1f A served parent works end to end
  Given a real served parent with a pending note and a live note
  When the child activates both, accepts its copy, and the parent's note changes and is activated through the local copy
  Then the pending parent note was never pulled, the live one was, and the change arrives through the raw route
```

## 792a21zg A pull-down tells the parent the note was used

Requirement: [792a21zg A pull-down tells the parent the note was used](#792a21zg-a-pull-down-tells-the-parent-the-note-was-used) After a pull-down that wrote the note or a skip, once the vault lock is released, `activate` MUST send a best-effort
`activate` request for the parent note to the parent, and MUST NOT send it when the local write failed. A failure of that request MUST NOT fail the command and MUST NOT be
queued. A pull-down MUST be a parent contact: it MUST be refused with one warning, writing nothing, when the vault's
location check fails, the backoff window is active, the parent reports this vault's own ID, or the exchange state cannot be
read or stamped, and an outage on the first fetch MUST record one failure, print the one backoff warning with the queued
count, and make no further request for the later refs of the same command, all of which fail. A successful pull-down MUST
drain the outbox.

Rationale: the parent's recency ranks what its children use, so a note that was not written locally was not used and sends no signal. A signal that failed is only a lost hint, so it neither
fails the command nor waits in a queue.

### 792a21zg1 UAT Scenarios

```gherkin
Scenario: 792a21zg1a Parent recency is bumped
  Given parent note P activated from a child and the parent reachable
  When the pull-down completes
  Then P's sidecar LastUsed on the parent is updated

Scenario: 792a21zg1b A failed signal is not fatal
  Given the parent's activate request fails
  When the pull-down completes
  Then the command succeeds and nothing is queued

Scenario: 792a21zg1c A paused vault pulls nothing
  Given a failed location check, an active backoff window, a self-parent or unusable exchange state
  When the user activates a parent-only note
  Then each stops the pull-down with its one warning and nothing is fetched or written

Scenario: 792a21zg1d An outage pauses the later refs
  Given an unreachable parent
  When the user activates several parent-only notes
  Then the first fetch records one failure and prints one backoff warning with the queued-offer count, the later refs make no request, and every ref fails

Scenario: 792a21zg1e A pull-down drains the outbox
  Given queued offers
  When a pull-down succeeds
  Then the queued offers drain

Scenario: 792a21zg1f The fetch holds no lock
  Given a pull-down in progress
  When the raw fetch and the activate request are made
  Then the vault lock is not held, and the write happens under it

Scenario: 792a21zg1g A failed local write sends no signal
  Given a parent-only note whose local write fails
  When the user activates it
  Then the command fails, and the parent receives no activate request for the note
```

## 792a21zh A pull-down stamps the first-write identity

Requirement: [792a21zh A pull-down stamps the first-write identity](#792a21zh-a-pull-down-stamps-the-first-write-identity) A pull-down MUST stamp the new note with the same detection of `repo:`, `user:` and `vault:` that
stamps every first write, and MUST NOT keep the parent note's own `user:` when user detection resolves empty.

Rationale: the pulled note is this vault's note, and the parent's authorship is kept apart in `parent.author` ([792a21zd A pulled-down note is an exact-content local pending offer](#792a21zd-a-pulled-down-note-is-an-exact-content-local-pending-offer)).

### 792a21zh1 UAT Scenarios

```gherkin
Scenario: 792a21zh1a No user detectable on a pull-down
  Given a host where neither git config user.email nor the operating-system username lookup resolves
  When a parent note is pulled down
  Then the note has no user: key, vault: is stamped as usual, and stderr carries one warning that user detection resolved empty
```

## 792a21zi An exchanged note carries a stable exchange ID

Requirement: [792a21zi An exchanged note carries a stable exchange ID](#792a21zi-an-exchanged-note-carries-a-stable-exchange-id) A note MUST gain an `xid` frontmatter field, a random identifier, lazily, the first time it takes
part in exchange (queued as an offer, pulled down, or received as a served offer). No migration, backfill or `update` step
MUST stamp `xid` on existing notes, an `xid` MUST never change once written, a rename included, and a note never involved
in exchange MUST carry no `xid`. Every frontmatter field that exchange adds (`xid`, `parent` and its members, `aliases`,
`offer` and its members) MUST be omitted when empty, so that a note that never takes part in exchange serializes exactly as
it did before.

Rationale: a rename-stable identifier lets the outbox and the idempotency keys survive a Luhmann re-parenting. Stamping
lazily means no vault pays for exchange it never uses.

### 792a21zi1 UAT Scenarios

```gherkin
Scenario: 792a21zi1a No backfill
  Given a vault whose notes have never taken part in exchange
  When the user runs engram update
  Then no note gains an xid and no note file changes

Scenario: 792a21zi1b The xid survives a rename
  Given a note carrying xid
  When Luhmann reparenting renames it
  Then the renamed note carries the same xid
```

## 792a21zj An exchanged note carries vault-qualified parent links

Requirement: [792a21zj An exchanged note carries vault-qualified parent links](#792a21zj-an-exchanged-note-carries-vault-qualified-parent-links) A note linked to parent notes MUST carry a nested `parent` key with `vault` (the parent's vault ID),
`links` (a list of `{note, via, hash}` entries, where `note` is a parent basename, `via` is `offered`, `pulled` or
`covered`, and `hash` is the exchange hash last exchanged) and, for pulled notes only, `author` (the parent note's `repo`,
`user` and `vault`). At most one link MUST have `via` `offered` or `pulled`, and that link is the note's primary link.
Links MUST count only while `parent.vault` equals the vault ID that the configured parent reports, so that a hostname or
address change of the same parent keeps them valid. Only an offer receipt, a pull-down, and `engram amend --discard --into`
MUST set or change `parent`, and a note never exchanged MUST carry no `parent` key.

Rationale: keying links by URL would break on every hostname or address change, so they key on the parent's stable vault
ID, and a second parent vault simply ignores the old links.

### 792a21zj1 UAT Scenarios

```gherkin
Scenario: 792a21zj1a Offered note is linked
  Given a learned note whose offer receipt names parent basename B from vault 9a1e
  When the receipt is recorded
  Then the note carries parent.vault 9a1e and a primary link {note: B, via: offered, hash: <offered hash>}

Scenario: 792a21zj1b A changed parent address keeps links valid
  Given linked notes
  When ENGRAM_PARENT changes to a new hostname for the same parent vault
  Then existing links still count, because the reported vault ID is unchanged

Scenario: 792a21zj1c A different parent vault ignores old links
  Given linked notes
  When ENGRAM_PARENT points at a parent that reports a different vault ID
  Then notes linked to the old vault ID are treated as unlinked
```

## 792a21zk Renames and folds record aliases

Requirement: [792a21zk Renames and folds record aliases](#792a21zk-renames-and-folds-record-aliases) A note's `aliases` list MUST record the basenames the note answers to in its own vault. A rename by
Luhmann reparenting, or by skill adoption, MUST append the note's old basename to its `aliases` in the same write.
`engram amend --target O --discard --into E` MUST delete O and its sidecar and MUST add O's basename and all of O's
`aliases` to E's `aliases` without duplicates, MUST merge O's parent links into E (when E already has a primary link, O's
primary link becomes a `covered` link on E, and otherwise it becomes E's primary link), and MUST NOT re-embed E or queue
an offer.

Rationale: a parent's or a child's reference to a renamed or folded note keeps resolving through the alias, and the
merged query and the pull-down skip both follow aliases.

### 792a21zk1 UAT Scenarios

```gherkin
Scenario: 792a21zk1a A rename records the old name
  Given note 1100.2026-09-27.x
  When reparenting renames it to 12a.2026-09-27.x
  Then the renamed note's aliases includes 1100.2026-09-27.x

Scenario: 792a21zk1b A fold unions aliases
  Given offer O whose aliases lists A
  When the user runs engram amend --target O --discard --into E
  Then O and its sidecar are gone and E's aliases includes both O's basename and A

Scenario: 792a21zk1c A fold keeps E's primary link
  Given E has a primary link to P1 and O has a primary link via pulled to P2
  When O is folded into E
  Then E keeps P1 as its primary link and gains {note: P2, via: covered}
```

## 792a21zl Exchange fields survive every other frontmatter rewrite

Requirement: [792a21zl Exchange fields survive every other frontmatter rewrite](#792a21zl-exchange-fields-survive-every-other-frontmatter-rewrite) `xid`, `parent`, `aliases` and `offer` MUST be preserved unchanged by every path that rewrites a
note's frontmatter and is not an exchange path (the offer receipt, the pull-down and the curation fold, and the rename's
append to `aliases`): `engram amend` with every flag, `engram resituate`, identity backfill, Luhmann reparenting and the
rewrite of wikilinks, the vocabulary tag rewrites (assignment, clear, legacy cleanup, self-tag and version stamp), the
scrubbing of references to deleted notes, and skill registration refresh and adopt. The fold and the receipt MUST edit
`aliases:` and `parent:` as YAML nodes, keep unknown keys under `parent:` and inside every link they keep, refuse
untouched and name the key (`parent` or `aliases`) when that key or anything beneath it carries a YAML anchor and the edit
would change it, and decode the rewritten frontmatter again before writing. The fold MUST do both before it writes the
existing note or deletes the offer, so that a refusal leaves both files untouched. For a note without unknown keys or
anchors both MUST write the same bytes as before.

Rationale: a link or an exchange ID lost to an unrelated rewrite would silently disconnect a note from its counterpart, and
the next exchange would treat the note as new. Each rewrite site has its own survival test.

### 792a21zl1 UAT Scenarios

```gherkin
Scenario: 792a21zl1a Amend preserves the exchange fields
  Given a note carrying xid, parent, aliases and offer
  When engram amend --target L --clear-pending, or any content amend, rewrites it
  Then all four are unchanged in the written note

Scenario: 792a21zl1b Identity backfill preserves the link
  Given a note carrying parent
  When engram update --backfill-identity rewrites it
  Then parent is unchanged

Scenario: 792a21zl1c Fold refuses an anchored parent
  Given a note E whose parent: carries an anchor that another key aliases, and a fold that would change E's parent links
  When the user runs engram amend --discard --into E
  Then the command fails naming parent, E is byte-identical, and the offer is not deleted

Scenario: 792a21zl1d Fold keeps unknown keys under parent
  Given E's parent: and one of its links carry keys the exchange model does not define
  When the fold rewrites E's parent:
  Then the written E still carries both keys with their values, alongside the folded links

Scenario: 792a21zl1e Receipt refuses an anchored parent
  Given a note whose parent: carries an anchor that another key aliases
  When an offer receipt would rewrite it
  Then the receipt is refused naming parent and the note is not written

Scenario: 792a21zl1f Receipt keeps unknown keys under parent
  Given a note whose parent: and one of its kept links carry keys the exchange model does not define
  When an offer receipt rewrites it
  Then the written note still carries both keys with their values

Scenario: 792a21zl1g Every other rewrite path preserves the fields
  Given a note carrying xid, parent, aliases and offer
  When resituate, reparenting, a vocabulary rewrite, a reference scrub or a skill registration refresh or adopt rewrites it
  Then all four are unchanged
```
