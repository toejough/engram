# Durable File Store - Technical Spec

## 792b3a Orientation

- Ownership: Repository-owned dependency-boundary interaction (the operating-system filesystem).
- Responsibility: every engram read and write of vault, sidecar, state and chunk-index files. It
  replaces a file atomically, creates a file exclusively, reports failures with the path and
  operation while keeping the underlying error identity, and lists the Markdown files that make up
  a vault directory.
- Current realization (informational): `primFS` in `internal/cli/edgefs.go` (the `EdgeFS`
  interface in `internal/cli/deps.go`), `vaultFS` in `internal/cli/vault_fs.go`, `listMDFromFS` in
  `internal/cli/deps_compose.go`, over the raw `FSPrims` that `cmd/engram/main.go` supplies.
- Qualification: responsibility, atomic replace, exclusive create and vault listing are stated
  without naming any caller; ownership, callers decide what to write and when, while this
  component decides how a write reaches the disk; independent verification, it is tested directly
  over fake primitives and over the real filesystem; stability, the contract survives any change to
  the injection seam; necessity, every [vault-identity](../product/vault-identity.md), note and index write depends on it.
- Supported capabilities: [vault-identity](../product/vault-identity.md), [note-writing](../product/note-writing.md), [note-revision](../product/note-revision.md), [chunk-indexing](../product/chunk-indexing.md)
  chunk-indexing.

Rationale: `os.WriteFile` is not atomic, so unlocked concurrent writers could corrupt state. A same-directory rename
is atomic on POSIX, so a reader sees either the old file or the new one, never a partial write.

## 792b3c Behavior Requirements

### 792b3c1 Atomic replace

Requirement: [792b3c1 Atomic replace](#792b3c1-atomic-replace) The durable-file-store MUST replace a file's content by writing the new data
to a uniquely named temp file in the target's directory, setting the temp's permissions to exactly
the requested mode regardless of umask, and renaming the temp onto the target, so that a concurrent
reader sees either the old or the new content. On any failure after the temp is created it MUST
remove the temp, leave the original unchanged, and return an error naming the target path.

Supports: [792a2h1a serve stamps an existing vault](../product/vault-identity.md#792a2h1-uat-scenarios), [792a2m1a Regenerate gives a copy its own identity](../product/vault-identity.md#792a2m1-uat-scenarios), [792a2n1a Claim accepts a moved vault](../product/vault-identity.md#792a2n1-uat-scenarios), [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a3u1a A runbook is tagged](../product/note-writing.md#792a3u1-uat-scenarios), [792a3u1b The answer is tagged and the question is not](../product/note-writing.md#792a3u1-uat-scenarios), [792a4b1a A fact field is replaced](../product/note-revision.md#792a4b1-uat-scenarios), [792a4l1a A fact is resituated](../product/note-revision.md#792a4l1-uat-scenarios), [792a4m1a Amend converts a CRLF note](../product/note-revision.md#792a4m1-uat-scenarios),
[792a4n1a A note's use is recorded](../product/note-revision.md#792a4n1-uat-scenarios).

Rationale: the unique temp name (target base, clock nanoseconds, attempt counter) is retried a
bounded number of times on a name collision, so two writers of the same file never share a temp.

#### 792b3c1a Scenarios

```gherkin
Scenario: 792b3c1a1 An existing file is replaced and no temp remains
  Given a file with old content
  When it is replaced atomically with new content
  Then the file holds the new content and no temp file remains in its directory

Scenario: 792b3c1a2 A new file is written
  Given no file at the target path
  When it is written atomically
  Then the file exists with the data

Scenario: 792b3c1a3 Permissions do not depend on umask
  Given a restrictive process umask
  When a file is written atomically with mode 0644
  Then the file's mode is exactly 0644

Scenario: 792b3c1a4 A rename failure removes the temp and keeps the original
  Given a file with original content and a rename that fails
  When it is replaced atomically
  Then an error naming the path is returned, the temp is removed, and the original content is unchanged

Scenario: 792b3c1a5 A permission failure removes the temp
  Given a temp file whose permissions cannot be set
  When a file is replaced atomically
  Then an error naming the path is returned and the temp is removed

Scenario: 792b3c1a6 A failed temp creation leaves the original untouched
  Given a target directory where no file can be created
  When a file in it is replaced atomically
  Then an error is returned, nothing is created, and the original content is unchanged

Scenario: 792b3c1a7 A temp-name collision tries a fresh name
  Given the first candidate temp name already exists
  When a file is replaced atomically
  Then the next candidate is used and the write succeeds

Scenario: 792b3c1a8 Exhausted temp names give a bounded error
  Given every candidate temp name already exists
  When a file is replaced atomically
  Then an error saying no unique temp name was available is returned after the bounded attempts
```

### 792b3c2 Exclusive create

Requirement: [792b3c2 Exclusive create](#792b3c2-exclusive-create) The durable-file-store MUST create a file only when no file exists at the path,
writing the data at the requested mode, and MUST fail with an error that names the path and
satisfies `errors.Is(err, fs.ErrExist)` when one does. When writing the data fails, that error
takes precedence over a close error; when the write succeeds and the close fails, the close error
is returned.

Supports: [792a2h1c Concurrent creation converges](../product/vault-identity.md#792a2h1-uat-scenarios), [792a3a1a A fact note is written](../product/note-writing.md#792a3a1-uat-scenarios), [792a3b1a A feedback note is written](../product/note-writing.md#792a3b1-uat-scenarios), [792a3c1a A runbook note is written](../product/note-writing.md#792a3c1-uat-scenarios), [792a3m1a Concurrent learns get distinct IDs](../product/note-writing.md#792a3m1-uat-scenarios), [792a3q1a A pair is captured](../product/note-writing.md#792a3q1-uat-scenarios), [792a3s1a A failed answer removes the question](../product/note-writing.md#792a3s1-uat-scenarios).

Rationale: exclusive create is the collision backstop for concurrent creators, such as two
processes stamping the same vault ID or racing for the same Luhmann ID.

#### 792b3c2a Scenarios

```gherkin
Scenario: 792b3c2a1 An existing file is refused
  Given a file already exists at the path
  When the path is created exclusively
  Then the call fails with an error satisfying fs.ErrExist that names the path, and the existing content is unchanged

Scenario: 792b3c2a2 A new file is created with the data and mode
  Given no file exists at the path
  When the path is created exclusively with data and a mode
  Then the file holds the data with that mode

Scenario: 792b3c2a3 A write error wins over a close error
  Given a created file whose write and close both fail
  When the path is created exclusively
  Then the write error is returned

Scenario: 792b3c2a4 A close error surfaces after a good write
  Given a created file whose write succeeds and whose close fails
  When the path is created exclusively
  Then the close error is returned
```

### 792b3c3 Error identity

Requirement: [792b3c3 Error identity](#792b3c3-error-identity) The durable-file-store MUST wrap every failed filesystem operation in an error
that names the operation and path and still satisfies `errors.Is` for the underlying error, such as
`fs.ErrNotExist`.

Supports: [792a2e1a A read command creates the vault](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2f1a An existing vault is untouched by read-only use](../product/vault-identity.md#792a2f1-uat-scenarios).

Rationale: callers decide "missing" versus "failed" by `errors.Is`. First-use creation, for
example, creates a vault only when the path is missing.

#### 792b3c3a Scenarios

```gherkin
Scenario: 792b3c3a1 A missing file still reads as not-exist
  Given no file at the path
  When the file is read
  Then the error names the path and satisfies fs.ErrNotExist

Scenario: 792b3c3a2 Every operation adds its context
  Given a filesystem whose every primitive fails
  When each operation is called
  Then each error names its operation and path and wraps the primitive's error
```

### 792b3c4 Vault Markdown listing

Requirement: [792b3c4 Vault Markdown listing](#792b3c4-vault-markdown-listing) The durable-file-store MUST list a vault directory as the names of the `.md`
files directly inside it, leaving out subdirectories, other files and the untouched starter
`README.md`, MUST return an empty list when the directory does not exist, and MUST return any other
read error.

Supports: [792a2c1a A Markdown file in a subdirectory is not a note](../product/vault-identity.md#792a2c1-uat-scenarios), [792a2d1a A fresh vault has no notes](../product/vault-identity.md#792a2d1-uat-scenarios), [792a2d1b A user's README is a note](../product/vault-identity.md#792a2d1-uat-scenarios), [792a3r1a An invalid pair is refused before any write](../product/note-writing.md#792a3r1-uat-scenarios), [792a8a1e Only notes at the vault root are counted](../product/vault-counting.md#792a8a1-uat-scenarios), [792a8b1c An empty vault prints nothing](../product/vault-counting.md#792a8b1-uat-scenarios).

Rationale: this listing is the one shared point where every vault scanner (query, check, embed,
count) stops seeing subdirectories and the starter README.

#### 792b3c4a Scenarios

```gherkin
Scenario: 792b3c4a1 Only root Markdown files are listed
  Given a directory with note.md, readme.txt and a subdirectory
  When the vault directory is listed
  Then the list is exactly note.md

Scenario: 792b3c4a2 A missing directory lists as empty
  Given a directory path that does not exist, even behind a wrapped not-exist error
  When the vault directory is listed
  Then the list is empty and no error is returned

Scenario: 792b3c4a3 Other read errors are returned
  Given a path that names a regular file
  When it is listed as a vault directory
  Then an error is returned

Scenario: 792b3c4a4 The starter README is left out
  Given a directory holding the untouched starter README.md and one note
  When the vault directory is listed
  Then the list holds only the note, and an edited README.md would be listed
```
