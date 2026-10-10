# Ingest Auto-Sweep with Non-Persistent-Workspace Skip Specification

## Purpose

The product behavior of `engram ingest --auto` (its roots, the `.claude/jobs` exclusion, the throwaway-workspace skip, and the rule that a source with no chunks writes no index file) and of `engram prune --empty` is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a5-added-chunk-indexing` (792a5d, 792a5f, 792a5k, 792a5q). The technical clauses below wait for ingestion-source-discovery (792c7d4).

## Requirements

### Requirement: Sweep-root resolution mechanics (residue)

The session-log skip SHALL match each walked directory's NAME against `non_persistent_prefixes`, attached only to the session-log root. The root drop SHALL compare each `--auto` root's own resolved path against `non_persistent_path_prefixes` at root-derivation time, before `extra_roots` are appended, respecting path-segment boundaries. The `.claude` root SHALL carry `claude_exclude_dirs` (including `jobs`) on top of `exclude_dirs`. A source yielding zero chunk records SHALL skip the index-file write while the manifest and dedup state are recorded independently.
