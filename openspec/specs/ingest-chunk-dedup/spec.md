# Chunk-Index Dedup by Content Hash Specification

## Purpose

The product behavior of ingest-time dedup and of `engram prune --duplicates` is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a5-added-chunk-indexing` (792a5l, 792a5m, 792a5r), and its mechanics in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b18-added-chunk-index-store` (792b18c6, 792b18c7, 792b18c11). The `engram update` backlog notice below waits for install-and-update (792c7j1). Validation: `dev/eval/LEDGER.md#chunk-index-dedup`.

## Requirements

### Requirement: `engram update` SHALL detect backlog and notify with exact command

`engram update` SHALL detect a live duplicate backlog (manifest entries with multiple canonical members per hash group) and notify the user with the exact `engram prune --duplicates` command when cleanup would remove something, rather than removing anything on its own.

#### Scenario: Update reports prune command when duplicates exist
- **WHEN** a chunk index has multiple canonical entries per hash group
- **THEN** `engram update` prints a notice with the exact `engram prune --duplicates` command and its expected impact
