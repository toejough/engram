# Concurrency-Safe Vault Writes Specification

## Purpose

The product guarantee that concurrent `engram learn` runs never collide is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a3-added-note-writing` (792a3m). Every vault writer (learn, amend, resituate, activate, ingest, prune) acquires an exclusive file lock at its entry point and holds it through all read-modify-write operations. Separate locks protect manifest changes (ingest, prune) and note+sidecar changes (amend, resituate, activate). The lock itself (creation, exclusive hold, release and unlock errors) and its rationale are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b8-added-vault-write-lock`; the per-command requirements below wait for their callers' nodes (792d63). Atomic temp-rename writes are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b3-added-durable-file-store`. Validation: concurrent-writers regression test + `targ check-full` (commit `f7f6b389`); correctness is test-locked, not eval-measured.

## Requirements

### Requirement: Exclusive flock on manifest updates
Every `engram ingest` and `engram prune` SHALL acquire an exclusive advisory flock on `.manifest.lock` before reading or modifying the chunk-index manifest (`.jsonl` file listings), holding the lock through all read-modify-write operations. Only one writer MUST hold the lock at a time; readers (query's manifest scan) work lock-free.

#### Scenario: Manifest write under lock
- **WHEN** `RunIngest` or `RunPrune` is called (entry point)
- **THEN** `primLocker.Lock(".manifest.lock")` acquires an exclusive flock, all manifest RMW happens while the lock is held, and the lock is released before returning

### Requirement: Exclusive flock on note+sidecar updates
Every `engram amend`, `engram resituate`, `engram activate`, and `engram vocab bootstrap`/`refit` SHALL acquire (for `engram learn` and `engram learn qa`, see 792b12c3 and 792b12c5) an exclusive advisory flock on `.luhmann.lock` before reading or modifying any vault note or sidecar (dual-vector .vec.json file), holding the lock through all operations. This covers note content rewrites and sidecar rewrites (e.g., re-embedding or bumping `last_used`). All are wired through the shared `vaultLockFromLocker` helper (internal/cli/deps_compose.go:137). Only one writer MUST hold the lock at a time; readers (query's file access) work lock-free.

#### Scenario: Note write under lock
- **WHEN** `RunAmend` or `RunResituate` is called (entry point)
- **THEN** `primLocker.Lock(".luhmann.lock")` acquires an exclusive flock, all vault-note and sidecar RMW happens while the lock is held, and the lock is released before returning

#### Scenario: Sidecar activate under lock
- **WHEN** `RunActivate` or vocab commands call `reEmbedAndActivate` to bump a note's `last_used` sidecar field
- **THEN** this happens within the `.luhmann.lock` critical section held by the entry point

### Requirement: Lock-free shared write helpers
Read-modify-write operations SHALL call shared helpers (`bumpLastUsed`, `writeManifestFile`, `reEmbedAndActivate`) which assume the caller already holds the lock. These helpers MUST NOT acquire locks themselves, relying on a convention that they are called only from a `Run*` entry point (which holds the lock). Violating this convention by calling a helper from within another helper risks deadlock.

#### Scenario: Helper assumes lock is held
- **WHEN** `bumpLastUsed` is called to update a sidecar's `last_used` field
- **THEN** the caller has already acquired `.luhmann.lock`; the function writes directly without re-acquiring

#### Scenario: No nested lock acquisition
- **WHEN** a `Run*` method acquires `.luhmann.lock` and calls `reEmbedAndActivate`
- **THEN** `reEmbedAndActivate` does not attempt to re-acquire the lock (would deadlock); it assumes the lock is already held
