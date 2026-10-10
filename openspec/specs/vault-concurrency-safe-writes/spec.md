# Concurrency-Safe Vault Writes Specification

## Purpose

The product guarantee that concurrent `engram learn` runs never collide is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a3-added-note-writing` (792a3m). Every vault writer (learn, amend, resituate, activate, ingest, prune) acquires an exclusive file lock at its entry point and holds it through all read-modify-write operations. Separate locks protect manifest changes (ingest, prune) and note+sidecar changes (amend, resituate, activate). The lock itself (creation, exclusive hold, release and unlock errors) and its rationale are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b8-added-vault-write-lock`; `engram amend`, `resituate` and `activate`'s locking and the rule that their shared helpers never lock are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b14-added-note-amendment` (792b14c1); `engram ingest` and `engram prune`'s manifest locking and the lock-free manifest write are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b18-added-chunk-index-store` (792b18c2); the requirement below waits for its caller's node (792d63). Atomic temp-rename writes are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b3-added-durable-file-store`. Validation: concurrent-writers regression test + `targ check-full` (commit `f7f6b389`); correctness is test-locked, not eval-measured.

## Requirements

### Requirement: Exclusive flock on note+sidecar updates
Every `engram vocab bootstrap`/`refit` SHALL acquire (for `engram learn` and `engram learn qa`, see 792b12c3 and 792b12c5; for `amend`, `resituate` and `activate`, 792b14c1) an exclusive advisory flock on `.luhmann.lock` before reading or modifying any vault note or sidecar (dual-vector .vec.json file), holding the lock through all operations. This covers note content rewrites and sidecar rewrites (e.g., re-embedding or bumping `last_used`). All are wired through the shared `vaultLockFromLocker` helper (internal/cli/deps_compose.go:137). Only one writer MUST hold the lock at a time; readers (query's file access) work lock-free.

#### Scenario: Sidecar activate under lock
- **WHEN** vocab commands bump a note's `last_used` sidecar field
- **THEN** this happens within the `.luhmann.lock` critical section held by the entry point
