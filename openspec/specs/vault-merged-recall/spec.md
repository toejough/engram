# Vault-Merged Recall Specification

## Purpose

Lets a node with its own local vault also merge results from its single
configured parent vault into one ranked `engram query` answer, so an agent
gets one coherent view of local plus inherited memory instead of two
separate queries it has to reconcile by hand.

## Requirements

### Requirement: ENGRAM_PARENT configures a single parent vault
`engram query` SHALL accept an `ENGRAM_PARENT` environment variable naming
exactly one parent vault's base URL. No mechanism SHALL exist to configure
more than one simultaneous parent.

#### Scenario: ENGRAM_PARENT set to a single URL
- **WHEN** `ENGRAM_PARENT` is set to a served vault's base URL
- **THEN** `engram query` treats that URL as this node's one parent for
  merged results

### Requirement: Merged query combines local and parent results into one ranked list
When `ENGRAM_PARENT` is set and `ENGRAM_SERVER` is not, `engram query` SHALL
run its local query pipeline and separately query the parent's `/query`
endpoint, then SHALL return one payload whose items are ordered by
descending score across both result sets — not two separate per-source
result sets the caller must reconcile.

#### Scenario: Local and parent results appear in one ranked list
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and both the local
  vault and the parent vault have matching items for the query
- **THEN** the returned payload's items include results from both sources,
  ordered by descending score

(Item-count capping for the merged set is governed by `--limit` — see the
budgets requirement below, and `recall-payload-cuts` for `--limit`'s base
enforcement.)

### Requirement: Content, recency, and item-count budgets apply to the merged set, not per-source
When `ENGRAM_PARENT` is set, `engram query` SHALL apply `--content-budget`,
`--recent-fill`, and `--limit` as a single final pass over the merged
local+parent item set, not independently to each source before merging.
`--content-budget` applies across both channels combined; `--recent-fill`
governs Channel 2 (recency) only; `--limit` governs Channel 1 (relevance)
only and never displaces Channel 2 — the two budgets are independent, not
stacked into one combined cap (`recall-payload-cuts` owns `--limit`'s base
enforcement; this requirement governs only where in the merge pipeline
these budgets apply).

#### Scenario: content-budget caps full-content chunk items across the merged set
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and
  `--content-budget N`
- **THEN** at most N chunk items in the merged payload, by rank across both
  sources, render full content; lower-ranked chunk items from either source
  render as a snippet

#### Scenario: recent-fill caps the recency channel across the merged set
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and
  `--recent-fill N`
- **THEN** the merged payload's recency channel contains at most N
  newest-by-ingest items drawn from both sources combined, not N from each
  source independently

#### Scenario: limit caps the merged Channel 1, not each source
- **WHEN** `engram query` runs with `ENGRAM_PARENT` set and `--limit N`
- **THEN** the returned payload's Channel 1 (relevance-ranked) items total
  at most N, drawn from the combined, score-ranked local+parent set — not
  N from each source independently before merging, and not counting the
  merged recency channel (which `--limit` never displaces — see
  `recall-payload-cuts`)

### Requirement: Merge does not gate on model_id
`engram query` SHALL merge local and parent results into the ranked list
regardless of whether their `model_id` values match. The merge SHALL NOT
refuse to run, error, or drop either source's results solely because
`model_id` differs.

#### Scenario: Mismatched model_id does not block the merge
- **WHEN** the local vault's `model_id` differs from the parent's `model_id`
- **THEN** `engram query` still returns a single ranked payload combining
  both sources' items

### Requirement: Merged items carry their source's model_id
Each item in a merged query payload SHALL carry the `model_id` of the node
that produced it.

#### Scenario: Per-item model_id present in a merged payload
- **WHEN** `engram query` returns a merged payload
- **THEN** every item in that payload's item list includes the `model_id`
  of whichever node (local or parent) produced it

### Requirement: Merged items are tagged with their origin
Each item in a merged query payload SHALL carry a boolean field indicating
whether it originated from the parent rather than the local vault.

#### Scenario: Parent-origin and local-origin items are distinguishable
- **WHEN** `engram query` returns a merged payload
- **THEN** each item the parent produced has its parent-origin field set to
  true, and each item produced locally has it set to false

### Requirement: Merged results are not re-clustered across nodes
A merged query payload SHALL present its combined items as one ranked list
rather than computing new clusters spanning both nodes' items.

#### Scenario: No cross-node cluster grouping
- **WHEN** `engram query` returns a merged payload
- **THEN** the payload does not present a cluster grouping that mixes
  local-origin and parent-origin items into the same cluster

### Requirement: Parent unavailability degrades to local-only results
When the parent is unreachable or returns an error, `engram query` SHALL
still return the local vault's results (unchanged from local-only mode)
rather than failing the whole command, and SHALL emit a non-fatal warning
noting the parent was unavailable.

#### Scenario: Parent unreachable
- **WHEN** `ENGRAM_PARENT` is set but the parent does not respond (network
  error, timeout, or non-2xx response)
- **THEN** `engram query` returns the local vault's results and emits a
  non-fatal warning, rather than returning an error and no results

### Requirement: show and show-chunk can route a lookup to the parent
`engram show` and `engram show-chunk` SHALL accept a `--parent` flag. When
`--parent` is set and `ENGRAM_PARENT` is configured, the command SHALL
resolve the given ref against the parent vault's existing `/show` or
`/show-chunk` endpoint instead of the local vault.

#### Scenario: --parent routes to the configured parent
- **WHEN** `engram show <ref> --parent` (or `engram show-chunk <id>
  --parent`) runs with `ENGRAM_PARENT` set
- **THEN** the ref is resolved against the parent vault, not the local
  vault

#### Scenario: Without --parent, behavior is unchanged
- **WHEN** `engram show` or `engram show-chunk` runs without `--parent`
- **THEN** its behavior is unchanged from before this capability existed
  (local-only, or `ENGRAM_SERVER`-exclusive if that is set)

#### Scenario: --parent without ENGRAM_PARENT configured is an error
- **WHEN** `--parent` is passed but `ENGRAM_PARENT` is not set
- **THEN** the command returns an error and performs no lookup

### Requirement: ENGRAM_SERVER takes precedence over ENGRAM_PARENT
When both `ENGRAM_SERVER` and `ENGRAM_PARENT` are set, `engram query` SHALL
run in `ENGRAM_SERVER`'s existing exclusive remote-client mode, unchanged,
and SHALL NOT attempt to merge in parent results. `engram show`/`engram
show-chunk --parent` SHALL likewise be inert in that case — `ENGRAM_SERVER`
already fully determines where the command runs.

#### Scenario: Both env vars set
- **WHEN** `ENGRAM_SERVER` and `ENGRAM_PARENT` are both set
- **THEN** `engram query` behaves exactly as it does today with only
  `ENGRAM_SERVER` set, and `ENGRAM_PARENT` has no effect on the result

#### Scenario: --parent is inert when ENGRAM_SERVER is set
- **WHEN** `ENGRAM_SERVER` is set and `engram show`/`engram show-chunk
  --parent` is run
- **THEN** the command behaves exactly as it does today under
  `ENGRAM_SERVER`-exclusive mode, ignoring `--parent`

### Requirement: Local-only and server-exclusive query modes are otherwise unaffected
`engram query`'s behavior and output shape when `ENGRAM_PARENT` is unset
SHALL be unchanged from its behavior before this capability existed, except
for `--limit` now capping `items[]` count — a real-enforcement fix owned by
`recall-payload-cuts` and bundled into this same change, applying
regardless of `ENGRAM_PARENT`.

#### Scenario: ENGRAM_PARENT unset
- **WHEN** `ENGRAM_PARENT` is not set
- **THEN** `engram query`'s output is unchanged from its behavior before
  this capability existed, other than `items[]` now being capped at
  `--limit` per `recall-payload-cuts`
