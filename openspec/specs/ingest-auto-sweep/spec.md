# Ingest Auto-Sweep with Non-Persistent-Workspace Skip Specification

## Purpose

The product behavior of `engram ingest --auto` (its roots, the `.claude/jobs` exclusion, the throwaway-workspace skip, and the rule that a source with no chunks writes no index file) and of `engram prune --empty` is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a5-added-chunk-indexing` (792a5d, 792a5f, 792a5k, 792a5q), and sweep-root resolution in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b17-added-ingestion-source-discovery`. The technical clause below waits for chunk-index-store (792c7d5).

## Requirements

### Requirement: Zero-record index-file guard (residue)

A source yielding zero chunk records SHALL skip the index-file write while the manifest and dedup state are recorded independently.
