# Payload cuts (--lazy-chunks, --recent-fill) Specification

## Purpose

Recall's underlying query defers matched chunk full-text delivery and limits raw recent-activity volume — so a recall pass carries less to read without losing reach. The `--lazy-chunks` flag (QueryArgs.LazyChunks, internal/cli/query.go) renders matched chunk items with path/score only (no content); the agent fetches evidence on-demand via `engram show-chunk`. The `--recent-fill` flag (QueryArgs.RecentFill, defaults to 25, internal/cli/query.go) caps newest-by-ingest chunks in the recency channel (Channel 2). Why: docs/architecture/adr.md ADR-0004. Validation: dev/eval/LEDGER.md#payload-cut-lazy-chunks; dev/eval/LEDGER.md#payload-cut-recent-fill.

## Requirements

### Requirement: Lazy-chunks defers chunk content delivery

The query SHALL render matched chunk items (kind: chunk) with path and score fields only when `--lazy-chunks` is set, while note items (kind: fact/feedback) keep full content. The agent fetches a chunk's real content on-demand via `engram show-chunk`.

#### Scenario: Lazy-chunks shrinks payload for uninstructed recalls
- **WHEN** running `engram query --lazy-chunks` against a vault with matched chunks
- **THEN** chunk items carry no content field; payload shrinks −33.7% (146→97 KB measured real-vault)
- **AND** trap gate GREEN across 13 realistic uninstructed recalls with 0 chunk fetches

#### Scenario: Notes keep full content in lazy-chunks mode
- **WHEN** running `engram query --lazy-chunks` against a vault with matched notes
- **THEN** note items (kind: fact/feedback) retain full content field

### Requirement: Recent-fill caps recency channel volume

The query SHALL limit newest-by-ingest chunks in the recency channel (Channel 2, provenanceRecent role) to the `--recent-fill` limit (default 25, env ENGRAM_RECENT_FILL), reducing payload size by trimming unrelated recent activity.

#### Scenario: Recent-fill default shrinks payload
- **WHEN** running `engram query --recent-fill 25` (or omitting the flag for default)
- **THEN** payload shrinks −28% (230→165 KB measured real-vault)

#### Scenario: Recent-fill zero disables channel
- **WHEN** running `engram query --recent-fill 0`
- **THEN** the recency channel (Channel 2) is disabled; recent activity does not appear

### Requirement: Limit caps the relevance-ranked item count
The query SHALL cap the number of Channel 1 (relevance-ranked) items in the
returned `items[]` array at the resolved `--limit` value (flag only, no
environment override; default 20 via `defaultQueryLimit`), applied after
recency-channel assembly and before content-budget capping. Before this
requirement, `--limit` was report-only metadata (`Budget.Limit`); item
count was governed entirely by clustering/candidate-nomination sizing
(`matchPhraseLimit`, `matchSetCap`, `candidateNoteK`), with no hard
ceiling. This applies identically to local, `ENGRAM_SERVER`-exclusive, and
`ENGRAM_PARENT`-merged query modes.

`--limit` SHALL NOT cap Channel 2 (the recency channel, `provenance:
recent`) — that channel has its own dedicated budget (`--recent-fill`,
capability `recall-payload-cuts`) and SHALL survive intact even when
Channel 1 alone already reaches `--limit`. Capping the combined channels
under one budget would silently displace the entire recency channel in
any vault whose Channel 1 result count meets or exceeds `--limit` — not a
rare edge case; a routine one in a non-trivial vault.

#### Scenario: Default limit caps a large Channel 1 result set
- **WHEN** `engram query` runs against a vault whose Channel 1 (matched)
  items exceed 20, with no `--limit` flag given
- **THEN** the returned `items[]` array contains exactly 20 Channel 1
  items, the highest-ranked ones in existing rank order

#### Scenario: Explicit --limit caps Channel 1 to the requested count
- **WHEN** `engram query` runs with `--limit N`
- **THEN** the returned `items[]` array contains at most N Channel 1 items

#### Scenario: Fewer Channel 1 items than the limit are all returned
- **WHEN** `engram query` runs against a vault whose Channel 1 items total
  fewer than the resolved `--limit`
- **THEN** every Channel 1 item is returned, unchanged from before this
  requirement

#### Scenario: The recency channel survives a full Channel 1
- **WHEN** `engram query` runs against a vault whose Channel 1 (matched)
  items alone already meet or exceed the resolved `--limit`, and the
  recency channel (governed by `--recent-fill`) has items to contribute
- **THEN** the returned `items[]` array still includes the recency
  channel's items, in addition to the `--limit`-capped Channel 1 items —
  `--limit` never displaces Channel 2

### Requirement: Agent reads chunk content before judging cluster relevance, even in zero-note clusters

When Step 2.5 of the recall procedure (`agent-instructions/skills/recall/SKILL.md`) processes
a cluster, the agent SHALL fetch every chunk member's content via `engram show-chunk` before
stating any relevance or coverage judgment about that cluster — including clusters whose
`candidate_l2s` list is empty (no note candidates, chunks-only membership). The absence of a
note candidate to write does not exempt a cluster's chunk members from the read-before-judge
requirement.

#### Scenario: Zero-note cluster still gets its chunks read
- **WHEN** Step 2.5 processes a cluster whose `candidate_l2s` list is empty
- **THEN** the agent invokes `engram show-chunk` on every chunk member of that cluster before
  stating any relevance or coverage judgment about it

#### Scenario: Metadata-only judgment is a violation
- **WHEN** a chunk member's `path` or anchor text (title) alone — without its fetched content
  — is used as the basis for judging it "unrelated," "not applicable," or otherwise irrelevant
- **THEN** the judgment fails this requirement, regardless of how many other members of the
  same cluster were genuinely read and correctly judged

#### Scenario: A load-bearing chunk surfaces in a zero-note cluster
- **WHEN** a chunk carrying a hard-requirement convention (e.g. a recency-channel standard
  planted topically distant from the task) matches into a cluster with no note candidates,
  alongside an unrelated distractor chunk
- **THEN** reading both chunks' content (not just their titles) is required before either is
  judged relevant or irrelevant — this is the exact configuration in which #733's C5b honoring
  miss occurred (`dev/eval/traps/c5.py`, trial idx=3, `gate-C5-6l_mjvl1`)
