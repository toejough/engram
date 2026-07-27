# Prune Preserves Memory (Detach on Source Deletion) Specification

## Purpose

`engram prune` no longer garbage-collects chunks whose source file is gone — the embedded chunk with its vector is the asset; the source `.jsonl` is only provenance. Prune now detaches: it drops the stale manifest entry but keeps the per-source index file on disk, which chunk search discovers by directory scan (never via the manifest), so the memory stays fully searchable. Deleting ingested source files (e.g. a restored-transcript directory) reclaims disk without losing the recovered memory. Why: issue #659 (no dedicated ADR). Validation: `internal/cli/prune_test.go` (TestPruneDetachesDeadSources) and `prune_integration_test.go` (real-fs detach end-to-end: ingest → delete source → prune → query still finds chunk).

## Requirements

### Requirement: Manifest entry SHALL be dropped for dead sources

When `engram prune` runs, it SHALL identify every manifest entry whose source file no longer exists on disk and remove that manifest entry, marking the source as detached.

#### Scenario: Deleted source is removed from manifest
- **WHEN** a source file is deleted and `engram prune` runs
- **THEN** the manifest entry for that source is removed

#### Scenario: Existing source is kept in manifest
- **WHEN** a source file still exists and `engram prune` runs
- **THEN** the manifest entry is left unchanged

### Requirement: Per-source index file SHALL be kept on disk

When a source's manifest entry is detached, its per-source `.jsonl` index file (containing the embedded chunk vectors) SHALL be left on disk intact. This is the critical difference from deletion: the index file survives the prune operation.

#### Scenario: Index file survives after detach
- **WHEN** a source is detached (manifest entry removed) and `engram prune` completes
- **THEN** the source's `.jsonl` index file still exists on disk at its original path

#### Scenario: Detached source produces manifest report
- **WHEN** `engram prune` detaches sources
- **THEN** it reports "detached N source(s) — embedded chunks preserved (still searchable)"

### Requirement: Detached chunks SHALL remain fully searchable via directory scan

Chunk search SHALL discover .jsonl files by directory scan of the chunks directory, never by looking them up in the manifest. When a source is detached, its .jsonl index file is still discoverable by this directory scan, so the chunks within it remain available for retrieval.

#### Scenario: Query finds chunks from detached source
- **WHEN** a source is detached (manifest entry removed, index file kept), and an `engram query` is run
- **THEN** chunks from the detached source's index file are included in the search results, discoverable by the directory scan

#### Scenario: Detached chunks rank normally in results
- **WHEN** chunks from a detached source match a query
- **THEN** they compete in ranking and may surface in results like any other chunk, since they carry their embeddings intact

### Requirement: Memory SHALL be fully preserved after source file deletion

The complete workflow of ingesting a source, deleting the source file, running prune, and then querying SHALL leave all chunk content searchable. No recovery step is required; the chunks are available immediately after prune completes.

#### Scenario: End-to-end preservation after source deletion
- **WHEN** a source is ingested, the source file is deleted, `engram prune` runs, and then `engram query` searches
- **THEN** all chunks from the original source are still found in query results without any recovery or re-indexing step
