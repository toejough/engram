# Concurrency-Safe Vault Writes Specification

## Purpose

Every vault writer (learn, amend, resituate, activate, ingest, prune) acquires an exclusive file lock at its entry point and holds it through all read-modify-write operations, with atomic rename ensuring concurrent readers see either the old or new file, never a torn write. Separate locks protect manifest changes (ingest, prune) and note+sidecar changes (amend, resituate, activate). Why: `docs/architecture/adr.md` ADR-0013. Validation: concurrent-writers regression test + `targ check-full` (commit `f7f6b389`); correctness is test-locked, not eval-measured.

## Requirements

### Requirement: Exclusive flock on manifest updates
Every `engram ingest` and `engram prune` SHALL acquire an exclusive advisory flock on `.manifest.lock` before reading or modifying the chunk-index manifest (`.jsonl` file listings), holding the lock through all read-modify-write operations. Only one writer MUST hold the lock at a time; readers (query's manifest scan) work lock-free.

#### Scenario: Manifest write under lock
- **WHEN** `RunIngest` or `RunPrune` is called (entry point)
- **THEN** `primLocker.Lock(".manifest.lock")` acquires an exclusive flock, all manifest RMW happens while the lock is held, and the lock is released before returning

#### Scenario: Lock creation on first acquisition
- **WHEN** `Lock(path)` is called and the lock file does not exist
- **THEN** `prims.Lock.OpenLockFile(path, 0o600)` creates it with mode 0o600

### Requirement: Exclusive flock on note+sidecar updates
Every `engram learn`, `engram learn qa`, `engram amend`, `engram resituate`, `engram activate`, and `engram vocab bootstrap`/`refit` SHALL acquire an exclusive advisory flock on `.luhmann.lock` before reading or modifying any vault note or sidecar (dual-vector .vec.json file), holding the lock through all operations. This covers note content rewrites and sidecar rewrites (e.g., re-embedding or bumping `last_used`). All are wired through the shared `vaultLockFromLocker` helper (internal/cli/deps_compose.go:137). Only one writer MUST hold the lock at a time; readers (query's file access) work lock-free.

#### Scenario: Note write under lock
- **WHEN** `RunLearn`, `RunLearnQA`, `RunAmend`, or `RunResituate` is called (entry point)
- **THEN** `primLocker.Lock(".luhmann.lock")` acquires an exclusive flock, all vault-note and sidecar RMW happens while the lock is held, and the lock is released before returning

#### Scenario: Sidecar activate under lock
- **WHEN** `RunActivate` or vocab commands call `reEmbedAndActivate` to bump a note's `last_used` sidecar field
- **THEN** this happens within the `.luhmann.lock` critical section held by the entry point

### Requirement: Atomic temp-rename writes
Every vault file write (note, sidecar, manifest) SHALL use atomic temp-file-then-rename semantics (not bare `os.WriteFile`). A temp file MUST be written with random suffix, then atomically renamed to the target path via `os.Rename`, ensuring concurrent readers never see a partial write.

#### Scenario: Temp-rename for sidecar writes
- **WHEN** a sidecar is rewritten (embed, activate, etc.)
- **THEN** `WriteFileAtomic` writes to a temporary file (e.g., `<sidecar>.tmp<random>`) and then calls `os.Rename(temp, sidecar)`, so readers see either the old sidecar or the new one, never an in-progress write

#### Scenario: Manifest file renamed under lock
- **WHEN** manifest changes (chunk-index entries added/removed)
- **THEN** the new manifest is written to a temp file and renamed to `.jsonl` while `.manifest.lock` is held

### Requirement: Lock-free shared write helpers
Read-modify-write operations SHALL call shared helpers (`bumpLastUsed`, `writeManifestFile`, `reEmbedAndActivate`) which assume the caller already holds the lock. These helpers MUST NOT acquire locks themselves, relying on a convention that they are called only from a `Run*` entry point (which holds the lock). Violating this convention by calling a helper from within another helper risks deadlock.

#### Scenario: Helper assumes lock is held
- **WHEN** `bumpLastUsed` is called to update a sidecar's `last_used` field
- **THEN** the caller has already acquired `.luhmann.lock`; the function writes directly without re-acquiring

#### Scenario: No nested lock acquisition
- **WHEN** a `Run*` method acquires `.luhmann.lock` and calls `reEmbedAndActivate`
- **THEN** `reEmbedAndActivate` does not attempt to re-acquire the lock (would deadlock); it assumes the lock is already held

### Requirement: Unlock error handling
If `FlockUnlock` fails, the unlock error SHALL be returned to the caller. If `CloseFD` fails after a successful unlock, the close error MUST be returned. If both fail, the unlock error MUST take precedence. Errors SHALL be wrapped with context (path, operation name) for diagnostics.

#### Scenario: Unlock error before close error
- **WHEN** `primLocker.Lock().unlock()` is called and `FlockUnlock` returns an error but `CloseFD` succeeds
- **THEN** the unlock error is returned (close error suppressed)

#### Scenario: Errors wrapped with context
- **WHEN** unlock or close fails
- **THEN** error message includes the lock path and operation name (e.g., "funlock .luhmann.lock: <error>")
