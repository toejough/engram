# Prune Preserves Memory (Detach on Source Deletion) Specification

## Purpose

The product behavior of `engram prune` detaching vanished sources while their chunks stay searchable is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a5-added-chunk-indexing` (792a5p). The technical clause below waits for chunk-index-store (792c7d5).

## Requirements

### Requirement: Index discovery by directory scan (residue)

Detaching a source SHALL delete only its manifest entry and leave its per-source `.jsonl` index file on disk. Chunk search SHALL discover `.jsonl` files by a directory scan of the chunks directory, never by looking them up in the manifest, so a detached source's chunks stay searchable.
