## ADDED Requirements

### Requirement: Merged query SHALL dedupe linked and identical notes, keeping the local note
Before ranking and capping, a merged query SHALL treat a parent note item P as the same note as a live (non-pending) local note L when either of the following holds:
- L's parent link names the currently configured parent, and its `note` equals P's basename or is listed in P's `aliases`;
- L's content hash equals P's `content_hash`, and both are non-empty.

The merged payload SHALL then keep L and not P. P SHALL be dropped when L is already among the local items. Otherwise L's item SHALL take P's place, with `from_parent: false`. Pending local notes SHALL NOT suppress parent items. Parent chunk items SHALL NOT be deduped. Near-duplicates (different content, no link) SHALL NOT be merged by the query. They are left to curation. The dedupe-key fields (`content_hash`, `aliases`) SHALL NOT appear in the merged payload.

#### Scenario: A linked note appears once, as the local copy
- **WHEN** local note L carries a parent link to parent note P, and both match the query
- **THEN** the merged `items[]` contains L with `from_parent: false`, and does not contain P

#### Scenario: A link resolved through an alias
- **WHEN** L's parent link names offer basename O, and parent curation folded O into parent note E (E's `aliases` lists O)
- **THEN** the merged payload keeps L and drops E

#### Scenario: Identical content without a link
- **WHEN** an unlinked live local note and a parent note have equal content hashes
- **THEN** only the local note appears

#### Scenario: A pending pulled copy does not hide the parent note
- **WHEN** the only local copy of parent note P is a pending pull-down that curation has not yet judged
- **THEN** P still appears in the merged payload, tagged `from_parent: true`

#### Scenario: The local copy is substituted when it did not rank locally
- **WHEN** P matches linked local note L, and L is not among the local query's items
- **THEN** L's item appears at P's position with `from_parent: false`

#### Scenario: Near-duplicates are left alone
- **WHEN** a local note and a parent note have different content and no link
- **THEN** both may appear in the merged payload

### Requirement: Explore picks SHALL NOT displace direct matches in the merged set
A merged query SHALL order non-trigger Channel 1 items the way the single-source path does: direct items from both sources, merged by score, first; then explore picks from both sources, merged by score. It SHALL then apply `--limit` by position. The note floor (`noteFloorK`, capability `recall-matched-note-floor`) SHALL be re-applied over the merged direct items.

#### Scenario: High-scoring explore picks do not crowd out direct matches
- **WHEN** a merged query with `--limit 10` has 12 explore picks scoring 0.8–0.9 and 15 direct items scoring 0.5–0.7 across both sources
- **THEN** the 10 Channel 1 items are the 10 highest-scoring direct items, and no explore pick is included

### Requirement: The merged payload SHALL report the merge-applied budget
A merged query's `budget` block SHALL report the values the merge actually applied, not a zero value:
- `limit`, `content_budget`, and `lazy_chunks`: the values the merge applied;
- `chunks_snippeted`: counted by the merged content-budget pass;
- `explore_allocated`: the term-to-count map summed per term over both sources.

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
dedupe keys), SHALL remove duplicates per the dedupe requirement below, and
then SHALL return one payload whose items are ranked across both result
sets (score order within each channel, per the ordering requirement
below) — not two separate per-source result sets the caller must
reconcile. This merged path is the standard query path for any
environment with a parent; there is no other remote query mode.

#### Scenario: Local and parent results appear in one ranked list
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and both the local
  vault and the parent vault have matching items for the query
- **THEN** the returned payload's items include results from both sources,
  ordered by descending score

(Item-count capping for the merged set is governed by `--limit` — see the
budgets requirement below, and `recall-payload-cuts` for `--limit`'s base
enforcement.)

### Requirement: show and show-chunk can route a lookup to the parent
`engram show` and `engram show-chunk` SHALL accept a `--parent` flag. When
`--parent` is set and `ENGRAM_PARENT` is configured, the command SHALL
resolve the given ref against the parent vault's existing `/show` or
`/show-chunk` endpoint instead of the local vault. Additionally, when
`--parent` is NOT set, `ENGRAM_PARENT` is configured, and the ref is not
found locally, `engram show` and `engram show-chunk`
SHALL fall back to resolving the ref against the parent and SHALL label the
output as parent-sourced.

#### Scenario: --parent routes to the configured parent
- **WHEN** `engram show <ref> --parent` (or `engram show-chunk <id>
  --parent`) runs with `ENGRAM_PARENT` set
- **THEN** the ref is resolved against the parent vault, not the local
  vault

#### Scenario: Without --parent, behavior is unchanged
- **WHEN** `engram show` or `engram show-chunk` runs without `--parent` and the ref exists locally
- **THEN** the local note is returned exactly as before this capability existed; the parent is not contacted (only a local miss changes, per the fallback scenario below)

#### Scenario: Local miss falls back to the parent
- **WHEN** `engram show <ref>` runs without `--parent`, `ENGRAM_PARENT` is set, and the ref is not found locally
- **THEN** the ref is resolved against the parent and the output is labeled as parent-sourced

#### Scenario: Local miss with no parent configured is still an error
- **WHEN** `engram show <ref>` runs without `--parent`, `ENGRAM_PARENT` is not set, and the ref is not found locally
- **THEN** the command returns the same not-found error as before this capability existed

#### Scenario: --parent without ENGRAM_PARENT configured is an error
- **WHEN** `--parent` is passed but `ENGRAM_PARENT` is not set
- **THEN** the command returns an error and performs no lookup

## REMOVED Requirements

### Requirement: ENGRAM_SERVER takes precedence over ENGRAM_PARENT
**Reason**: The thin-client `ENGRAM_SERVER` mode is removed (capability `vault-local-first`). Setting it is now a hard error, so there is nothing for `ENGRAM_PARENT` to yield to.
**Migration**: Set `ENGRAM_PARENT` to the URL formerly in `ENGRAM_SERVER`. Queries then merge local and parent results, and writes are made locally and offered to the parent.

### Requirement: Local-only and server-exclusive query modes are otherwise unaffected
**Reason**: The server-exclusive mode no longer exists. The local-only guarantee moves, unchanged in substance, to the ADDED requirement "Local-only query mode is otherwise unaffected".
**Migration**: None needed. The local-only behavior is preserved.
