## ADDED Requirements

### Requirement: Merged query SHALL dedupe linked and identical notes, keeping the local note
Before ordering and capping, a merged query SHALL treat a parent note item P as the same note as a live (non-pending) local note L when either of these holds:
- L's `parent.vault` equals the vault ID the parent reported in this query, and any of L's parent links (`offered`, `pulled`, or `covered`) names P's basename or one of P's `aliases`;
- L's exchange hash equals P's `exchange_hash`, and both are non-empty.

The merged payload SHALL keep L and not P. When L is already among the local items, P SHALL be dropped. Otherwise P SHALL be replaced by a substituted item for L, which carries:
- `path`: L's basename plus `.md`;
- `kind: note`;
- `content`: L's content, rendered as a local query renders a note;
- `score` and `provenances`: P's;
- `source_term`: P's, only when P's provenances include `explore`;
- `model_id`: the local vault's model;
- `from_parent: false`.

Pending local notes SHALL NOT suppress parent items. Near-duplicates (different content, no link) SHALL NOT be merged by the query; they are left to curation. The dedupe-key fields (`exchange_hash`, `aliases`, and the parent's `vault_id`) SHALL NOT appear in the merged payload.

#### Scenario: A linked note appears once, as the local copy
- **WHEN** local note L carries a primary link to parent note P, and both match the query
- **THEN** the merged `items[]` contains L with `from_parent: false`, and does not contain P

#### Scenario: A covered link dedupes
- **WHEN** L carries a `covered` link to parent note P2 in addition to its primary link
- **THEN** P2 does not appear, and L appears once

#### Scenario: A link resolved through an alias
- **WHEN** L's link names basename O, and the parent's note E lists O in its `aliases` (O was folded into E, or renamed to E)
- **THEN** the merged payload keeps L and drops E

#### Scenario: Identical content without a link
- **WHEN** an unlinked live local note and a parent note have equal exchange hashes
- **THEN** only the local note appears

#### Scenario: A pending pulled copy does not hide the parent note
- **WHEN** the only local copy of parent note P is a pending pull-down that curation has not yet judged
- **THEN** P still appears in the merged payload, tagged `from_parent: true`

#### Scenario: The local copy is substituted when it did not rank locally
- **WHEN** P matches linked local note L, and L is not among the local query's items
- **THEN** an item for L with P's score and provenances, L's content, the local `model_id`, and `from_parent: false` appears at P's position

#### Scenario: Near-duplicates are left alone
- **WHEN** a local note and a parent note have different content and no link
- **THEN** both may appear in the merged payload

### Requirement: The merged Channel 1 SHALL order direct items before explore picks and keep the note floor
A merged query SHALL order its non-trigger Channel 1 items in two groups:
1. **direct items**: items whose `provenances` include none of `trigger`, `explore`, or `recent`, from both sources, by descending `score`;
2. **explore items**: items whose `provenances` include `explore`, from both sources, by descending `score`.

It SHALL then apply `--limit` by position. Let Q be the direct items whose `kind` is `note` and whose `score` is at least 0.25. When the capped set keeps fewer than min(5, |Q|) members of Q, the lowest-positioned kept direct items whose `kind` is `chunk` SHALL be replaced, one for one, by the highest-scoring excluded members of Q, until that minimum is kept or no kept direct chunk item remains. The direct group SHALL then be restored to score order.

#### Scenario: High-scoring explore picks do not crowd out direct matches
- **WHEN** a merged query with `--limit 10` has 12 explore picks scoring 0.8–0.9 and 15 direct items scoring 0.5–0.7 across both sources
- **THEN** the 10 Channel 1 items are the 10 highest-scoring direct items, and no explore pick is included

#### Scenario: The note floor survives the merged cap
- **WHEN** a merged query with `--limit 5` has 5 direct local chunk items scoring 0.9 and 3 direct note items scoring 0.5
- **THEN** the 5 Channel 1 items include all 3 notes and the 2 highest-scoring chunks

### Requirement: The merged payload SHALL report the merge-applied budget
A merged query's `budget` block SHALL report the values the merge actually applied, not a zero value:
- `limit`, `content_budget`, and `lazy_chunks`: the values the merge applied;
- `chunks_snippeted`: counted by the merged content-budget pass;
- `explore_allocated`: the term-to-count map, summed per term over both sources.

#### Scenario: Budget block is populated
- **WHEN** `engram query --limit 5 --content-budget 2 --lazy-chunks` runs merged
- **THEN** the payload's `budget` shows `limit: 5`, `content_budget: 2`, `lazy_chunks: true`, and the snippeted-chunk count the merge produced

### Requirement: Local-only query mode is otherwise unaffected
`engram query`'s behavior and output shape when `ENGRAM_PARENT` is unset SHALL be unchanged by the merge, the dedupe, and the offer machinery. The one exception is `--limit` capping `items[]` count, which capability `recall-payload-cuts` owns.

#### Scenario: No parent configured
- **WHEN** `ENGRAM_PARENT` is not set
- **THEN** `engram query` makes no network request, and its output matches the single-source pipeline, including `--limit` capping per `recall-payload-cuts`

## MODIFIED Requirements

### Requirement: Merged query combines local and parent results into one ranked list
When `ENGRAM_PARENT` is set, every `engram query` SHALL run its local query
pipeline and separately query the parent's `/query` endpoint (requesting
dedupe keys). It SHALL drop every parent item whose `kind` is `chunk`, SHALL
remove duplicates per the dedupe requirement below, and SHALL then return
one payload ranked across both result sets in the order that the ordering
requirement below defines (trigger hits, then direct items by score, then
explore items by score) — not two separate per-source result sets the
caller must reconcile. This merged path is the standard query path for any
environment with a parent. There is no other remote query mode, and no
parent chunk ever appears in a merged payload.

#### Scenario: Local and parent results appear in one ranked list
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and both the local
  vault and the parent vault have matching items for the query
- **THEN** the returned payload's items include note results from both
  sources, placed per the merged ordering requirement (score order within
  the direct group and within the explore group), with no parent chunk item

(Item-count capping for the merged set is governed by `--limit` — see the
budgets requirement below, and `recall-payload-cuts` for `--limit`'s base
enforcement.)

### Requirement: Content, recency, and item-count budgets apply to the merged set, not per-source
When `ENGRAM_PARENT` is set, `engram query` SHALL apply `--content-budget`,
`--recent-fill`, and `--limit` as a single final pass over the merged
item set, not independently to each source before merging. Because parent
chunk items are dropped, every chunk item — including all of Channel 2
(recency, which holds chunks only) — comes from the local vault.
`--content-budget` applies across both channels combined; `--recent-fill`
governs Channel 2 (recency) only; `--limit` governs Channel 1 (relevance)
only and never displaces Channel 2 — the two budgets are independent, not
stacked into one combined cap. Trigger hits (items whose `provenances` include `trigger`,
capability `runbook-lexical-triggers`) SHALL be split out of each source
before the `--limit` cap and placed first in the merged `items[]`: local
trigger hits first, then parent trigger hits, each in the order its source
returned them, ahead of the other Channel 1 items. They are exempt
from `--limit` (it caps only the non-trigger Channel 1 items) and are not
re-sorted by score across sources. `--content-budget` still applies to them
as part of the final pass. (`recall-payload-cuts` owns `--limit`'s base
enforcement; this requirement governs only where in the merge pipeline
these budgets apply).

#### Scenario: content-budget caps full-content chunk items across the merged set
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and
  `--content-budget N`
- **THEN** at most N chunk items in the merged payload, all of them local,
  render full content by rank; lower-ranked chunk items render as a snippet,
  and no parent chunk item is present

#### Scenario: recent-fill caps the recency channel across the merged set
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and
  `--recent-fill N`
- **THEN** the merged payload's recency channel contains at most N
  newest-by-ingest local chunk items, and none from the parent, not N from each
  source independently

#### Scenario: limit caps the merged Channel 1, not each source
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and `--limit N`
- **THEN** the returned payload's Channel 1 (relevance-ranked) items total
  at most N, drawn from the combined, score-ranked local+parent set — not
  N from each source independently before merging, and not counting the
  merged recency channel (which `--limit` never displaces — see
  `recall-payload-cuts`)

#### Scenario: trigger hits lead the merged payload, local before parent
- **WHEN** `engram query --text "<msg>" --limit 1` runs with `ENGRAM_PARENT`
  set, the local vault returns one trigger hit and a score-0.9 item, and
  the parent returns one trigger hit and a score-0.8 item
- **THEN** the merged `items[]` order is: local trigger hit, parent
  trigger hit, local score-0.9 item — both trigger hits survive `--limit 1`
  and the parent's score-0.8 item is dropped by the limit

### Requirement: Parent unavailability degrades to local-only results
When the parent is unreachable or returns an error, `engram query` SHALL
still return the local vault's results (unchanged from local-only mode)
rather than failing the whole command, and SHALL emit a non-fatal warning
noting the parent was unavailable. After a failure, parent contact SHALL be
backed off (capability `vault-parent-offers`): a query inside the backoff
window SHALL make no parent request and SHALL return local results with
one warning.

#### Scenario: Parent unreachable
- **WHEN** `ENGRAM_PARENT` is set but the parent does not respond (network
  error, timeout, or non-2xx response)
- **THEN** `engram query` returns the local vault's results and emits a
  non-fatal warning, rather than returning an error and no results

#### Scenario: Backoff window skips the parent
- **WHEN** a query runs within the backoff window after a parent failure
- **THEN** no parent request is made, and local results are returned with one warning

### Requirement: show and show-chunk can route a lookup to the parent
`engram show` SHALL accept a `--parent` flag. When `--parent` is set and
`ENGRAM_PARENT` is configured, the command SHALL resolve the given ref
against the parent vault's `/show` endpoint instead of the local vault.
Additionally, when `--parent` is NOT set, `ENGRAM_PARENT` is configured,
and the ref is not found locally, `engram show` SHALL fall back to resolving
the ref against the parent and SHALL label the output as parent-sourced.
`engram show-chunk` SHALL NOT accept `--parent` and SHALL NOT fall back to
the parent: chunks never cross vaults, so there is no parent chunk lookup.

#### Scenario: --parent routes to the configured parent
- **WHEN** `engram show <ref> --parent` runs with `ENGRAM_PARENT` set
- **THEN** the ref is resolved against the parent vault, not the local
  vault; `engram show-chunk --parent` is rejected as an unknown flag

#### Scenario: Without --parent, behavior is unchanged
- **WHEN** `engram show` or `engram show-chunk` runs without `--parent` and the ref exists locally
- **THEN** the local note is returned exactly as before this capability existed; the parent is not contacted (only a local miss changes, per the fallback scenario below)

#### Scenario: Local miss falls back to the parent
- **WHEN** `engram show <ref>` runs without `--parent`, `ENGRAM_PARENT` is set, and the ref is not found locally
- **THEN** the ref is resolved against the parent and the output is labeled as parent-sourced; `engram show-chunk <id>` in the same situation returns the local not-found error without contacting the parent

#### Scenario: Local miss with no parent configured is still an error
- **WHEN** `engram show <ref>` runs without `--parent`, `ENGRAM_PARENT` is not set, and the ref is not found locally
- **THEN** the command returns the same not-found error as before this capability existed

#### Scenario: --parent without ENGRAM_PARENT configured is an error
- **WHEN** `--parent` is passed but `ENGRAM_PARENT` is not set
- **THEN** the command returns an error and performs no lookup

## REMOVED Requirements

### Requirement: ENGRAM_SERVER takes precedence over ENGRAM_PARENT
**Reason**: The thin-client `ENGRAM_SERVER` mode is removed (capability `vault-local-first`). Setting it is now a hard error, so there is nothing for `ENGRAM_PARENT` to yield to.
**Migration**: Set `ENGRAM_PARENT` to the URL formerly in `ENGRAM_SERVER`. Queries then merge local results with the parent's notes, and writes are made locally and offered to the parent.

### Requirement: Local-only and server-exclusive query modes are otherwise unaffected
**Reason**: The server-exclusive mode no longer exists. The local-only guarantee moves, unchanged in substance, to the ADDED requirement "Local-only query mode is otherwise unaffected".
**Migration**: None needed. The local-only behavior is preserved.
