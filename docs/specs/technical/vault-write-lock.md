# Vault Write Lock - Technical Spec

## 792b8a Orientation

- Ownership: Repository-owned dependency-boundary interaction (operating-system advisory file
  locks).
- Responsibility: gives one process at a time exclusive hold of a lock file. It creates the lock
  file when absent, waits while another holder has it, and on release lets the lock go and closes
  the file, reporting any failure with the lock's path. The vault's lock is the file
  `.luhmann.lock` at the vault root.
- Current realization (informational): `primLocker` in `internal/cli/locker.go` (the `FileLocker`
  interface), over the raw `LockPrims` that `cmd/engram/main.go` supplies; `vaultLockFromLocker`
  in `internal/cli/deps_compose.go`. Callers also lock `.manifest.lock` in the chunk index
  directory; which writer takes which lock, and what it does while holding it, is each caller's
  requirement.
- Qualification: responsibility, exclusive hold of a lock file is stated without naming any writer;
  ownership, callers decide what to do under the lock and when to take it, while this component
  owns acquiring, waiting and releasing; independent verification, it is tested directly over fake
  primitives and over a real lock file; stability, the contract survives any change to the
  injection seam; necessity, concurrent note writes depend on it.
- Supported capabilities: [note-writing](../product/note-writing.md), [note-revision](../product/note-revision.md), [chunk-indexing](../product/chunk-indexing.md).
  Controlled-vocabulary ([controlled-vocabulary](../product/controlled-vocabulary.md)) adds its callers' requirements.

Rationale: before every read-modify-write writer took a lock, only `learn`'s Luhmann-ID allocation
was locked, and two concurrent `ingest`, `amend`, `resituate` or `activate` runs corrupted the vault
or the chunk index in production; many parallel sub-recalls make that common. Every such writer
now takes the lock at its command's entry point: `.luhmann.lock` for notes and sidecars, and
`.manifest.lock` for the chunk index. The helpers they share never take a lock themselves, so a
command never waits on a lock it already holds. That rule is a convention, not a checked
invariant. The lock works on a raw file descriptor rather than a Go file object, because a dropped
file object's finalizer would close the descriptor and silently release the lock while it was still
meant to be held.

## 792b8c Behavior Requirements

### 792b8c1 Exclusive hold

Requirement: [792b8c1 Exclusive hold](#792b8c1-exclusive-hold) The vault-write-lock MUST open the lock file at the given path, creating it
with mode `0600` when absent, MUST take an exclusive advisory lock on it, waiting while another
holder has it, and MUST then return a release function.

Supports: [792a3m1a Concurrent learns get distinct IDs](../product/note-writing.md#792a3m1-uat-scenarios), [792a3q1a A pair is captured](../product/note-writing.md#792a3q1-uat-scenarios), [792a4o1a Concurrent amends both take effect](../product/note-revision.md#792a4o1-uat-scenarios), [792a4o1b An activation does not undo a re-embed](../product/note-revision.md#792a4o1-uat-scenarios).

#### 792b8c1a Scenarios

```gherkin
Scenario: 792b8c1a1 A second holder waits for the first
  Given one holder has the lock on a lock file
  When a second holder asks for the same lock
  Then the second waits until the first releases it, and then gets it

Scenario: 792b8c1a2 An absent lock file is created
  Given a lock path with no file
  When the lock is taken
  Then the lock file is created with mode 0600
```

### 792b8c2 Release

Requirement: [792b8c2 Release](#792b8c2-release) The vault-write-lock's release MUST unlock and then close the lock file, MUST
close the file even when unlocking fails, and MUST return the unlock error when unlocking fails,
else the close error when closing fails, each naming the lock path, else nothing.

Supports: [792a3m1a Concurrent learns get distinct IDs](../product/note-writing.md#792a3m1-uat-scenarios).

#### 792b8c2a Scenarios

```gherkin
Scenario: 792b8c2a1 Release unlocks, then closes
  Given a held lock
  When it is released
  Then the file is unlocked and then closed, and no error is returned

Scenario: 792b8c2a2 An unlock failure is reported and the file still closed
  Given a held lock whose unlock fails
  When it is released
  Then the unlock error is returned, naming the operation, and the file is closed

Scenario: 792b8c2a3 A close failure is reported
  Given a held lock whose unlock succeeds and whose close fails
  When it is released
  Then the close error is returned
```

### 792b8c3 Acquire failures

Requirement: [792b8c3 Acquire failures](#792b8c3-acquire-failures) When the lock file cannot be opened, the vault-write-lock MUST return an error
naming the lock path without trying to lock it; when the lock cannot be taken, it MUST close the
file and return an error naming the lock path.

Supports: [792a3m1a Concurrent learns get distinct IDs](../product/note-writing.md#792a3m1-uat-scenarios).

#### 792b8c3a Scenarios

```gherkin
Scenario: 792b8c3a1 An open failure names the path
  Given a lock file that cannot be opened
  When the lock is taken
  Then the error names the lock path, and no lock is attempted

Scenario: 792b8c3a2 A lock failure closes the file
  Given a lock file that opens but cannot be locked
  When the lock is taken
  Then the file is closed and the error names the lock path
```

### 792b8c4 The vault's lock

Requirement: [792b8c4 The vault's lock](#792b8c4-the-vaults-lock) The vault-write-lock MUST lock a vault by locking the file `.luhmann.lock` at
the vault root, and releasing that lock MUST release the file's lock.

Supports: [792a3m1a Concurrent learns get distinct IDs](../product/note-writing.md#792a3m1-uat-scenarios), [792a3q1a A pair is captured](../product/note-writing.md#792a3q1-uat-scenarios), [792a4o1a Concurrent amends both take effect](../product/note-revision.md#792a4o1-uat-scenarios), [792a4o1b An activation does not undo a re-embed](../product/note-revision.md#792a4o1-uat-scenarios).

#### 792b8c4a Scenarios

```gherkin
Scenario: 792b8c4a1 A vault is locked through .luhmann.lock
  Given the vault /vault
  When the vault is locked and then released
  Then /vault/.luhmann.lock was locked, and released
```
