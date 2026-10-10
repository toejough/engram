# Chunk-Index Dedup by Content Hash Specification

## Purpose

The product behavior of ingest-time dedup (grouping by content and kind, canonical precedence, eviction gated on record coverage) and of `engram prune --duplicates` is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a5-added-chunk-indexing` (792a5l, 792a5m, 792a5r). The technical clauses below wait for chunk-index-store (792c7d5), and the `engram update` backlog notice for install-and-update (792c7j1). Validation: `dev/eval/LEDGER.md#chunk-index-dedup`.

## Requirements

### Requirement: Dedup mechanics (residue)

Sources SHALL be partitioned by `(FileHash, chunkingClass)`. A non-canonical member SHALL be recorded in the manifest as `DuplicateOf` the canonical and never indexed. The eviction gate SHALL compare the duplicate's index records by content hash against the canonical's current index, which is built before any duplicate of its group is reconciled; since `mergeChunkRecords` is append-only within a source, byte-identical sources can hold disjoint historical records. `prune --duplicates` SHALL group live manifest entries, skipping entries tagged `DuplicateOf`, and evict through the same helper as ingest.

### Requirement: `engram update` SHALL detect backlog and notify with exact command

`engram update` SHALL detect a live duplicate backlog (manifest entries with multiple canonical members per hash group) and notify the user with the exact `engram prune --duplicates` command when cleanup would remove something, rather than removing anything on its own.

#### Scenario: Update reports prune command when duplicates exist
- **WHEN** a chunk index has multiple canonical entries per hash group
- **THEN** `engram update` prints a notice with the exact `engram prune --duplicates` command and its expected impact
