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
