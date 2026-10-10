# Note Provenance - Technical Spec

## 792b11a Orientation

- Ownership: Repository-owned dependency-boundary interaction (`git`, through the
  [subprocess-runner](./subprocess-runner.md), and the operating system's user lookup).
- Responsibility: works out who and where a note write comes from: the repository of the working
  directory and the writing user. It stamps them, with the vault name, on a note's first write and
  re-stamps them on a revision, and omits, with a warning, any value that does not resolve, except
  that a re-stamp keeps a user the note already has.
- Current realization (informational): `detectRepo`, `detectUser`, `firstWriteIdentity` and
  `identityStamp` in `internal/cli/identity.go`; `newLearnDeps` wires them for `learn`. The amend
  re-stamp is `identityStamp.stampPreservingUser`; identity backfill
  (`internal/cli/identity_backfill.go`) is [792b11c5 Repository for a backfill](#792b11c5-repository-for-a-backfill) to [792b11c7 Identity backfill](#792b11c7-identity-backfill).
- Qualification: responsibility, provenance detection is stated without naming any command;
  ownership, writers decide when to stamp, while this component decides what the values are;
  independent verification, detection and stamping are tested directly over a fake `git` and user
  lookup; stability, the fields and their sources survive any change to the writers; necessity,
  every note's `repo:`, `user:` and `vault:` come from it.
- Supported capabilities: [vault-identity](../product/vault-identity.md), [note-writing](../product/note-writing.md), [note-revision](../product/note-revision.md), [vault-migration](../product/vault-migration.md)
  [vault-migration](../product/vault-migration.md). Vault-serving ([vault-serving](../product/vault-serving.md)) and [parent-sync](../product/parent-sync.md) add their requirements.

Rationale: detection shells out to `git` rather than reading `.git/config`, because reproducing
git's own configuration precedence (worktrees, `includeIf` and so on) would bring no benefit.
Identity is self-reported: it is detected from the writer's own environment at the moment of the
write and never verified, as `source:` already was; a verified identity could later replace it
without changing the fields. Detection never fails a write. A working directory outside git, like
an absent `project:`, simply leaves the field out.

## 792b11c Behavior Requirements

### 792b11c1 Repository detection

Requirement: [792b11c1 Repository detection](#792b11c1-repository-detection) The note-provenance MUST resolve the repository as the output of
`git remote get-url origin` in the working directory, trimmed; when that fails or prints nothing,
as the basename of the path printed by `git rev-parse --show-toplevel`; and otherwise, including
when the working directory cannot be resolved, as empty, without failing.

Supports: [792a3o1a The origin remote names the repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3o1b Without an origin the top-level directory names the repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3o1c Outside a git repository there is no repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a4h1a Amend from another environment re-stamps](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1b Amend backfills missing identity](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1c The project field does not stand in for the repo](../product/note-revision.md#792a4h1-uat-scenarios).

#### 792b11c1a Scenarios

```gherkin
Scenario: 792b11c1a1 The origin remote names the repository
  Given git remote get-url origin prints git@github.com:example/repo.git
  When the repository is resolved
  Then it is git@github.com:example/repo.git

Scenario: 792b11c1a2 Without an origin the top-level basename names it
  Given no origin remote, and git rev-parse --show-toplevel prints /home/joe/src/engram
  When the repository is resolved
  Then it is engram

Scenario: 792b11c1a3 Outside git there is no repository
  Given both git commands fail
  When the repository is resolved
  Then it is empty

Scenario: 792b11c1a4 An unresolvable working directory has no repository
  Given the working directory cannot be resolved
  When the repository is resolved
  Then it is empty, and git is never run
```

### 792b11c2 User detection

Requirement: [792b11c2 User detection](#792b11c2-user-detection) The note-provenance MUST resolve the user as the output of
`git config user.email`, trimmed; when that fails or prints nothing, as the OS username; and when
that also fails or is unavailable, as empty, without failing.

Supports: [792a3p1a The git email names the user](../product/note-writing.md#792a3p1-uat-scenarios), [792a3p1b The OS username is the fallback](../product/note-writing.md#792a3p1-uat-scenarios), [792a3p1c No detectable user](../product/note-writing.md#792a3p1-uat-scenarios), [792a4h1a Amend from another environment re-stamps](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1b Amend backfills missing identity](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1d Empty user detection keeps the prior user](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1e Empty user detection never writes an empty user](../product/note-revision.md#792a4h1-uat-scenarios).

#### 792b11c2a Scenarios

```gherkin
Scenario: 792b11c2a1 The git email names the user
  Given git config user.email prints agent@example.com
  When the user is resolved
  Then it is agent@example.com, and the OS username is not used

Scenario: 792b11c2a2 The OS username is the fallback
  Given git config user.email prints nothing or fails, and the OS username is joe
  When the user is resolved
  Then it is joe

Scenario: 792b11c2a3 No source resolves
  Given git config user.email fails and the OS username lookup fails or is unavailable
  When the user is resolved
  Then it is empty
```

### 792b11c3 First-write stamp

Requirement: [792b11c3 First-write stamp](#792b11c3-first-write-stamp) On a note's first write, the note-provenance MUST stamp the resolved
repository, user and vault name; a stamp whose user is empty MUST leave `user:` out and log one
warning that user detection resolved empty, naming the command, and a stamp whose vault name is
empty MUST leave `vault:` out and log one warning.

Supports: [792a2b1a The default vault name is personal](../product/vault-identity.md#792a2b1-uat-scenarios), [792a2b1b The environment names the vault](../product/vault-identity.md#792a2b1-uat-scenarios), [792a2b1c The flag wins over the environment](../product/vault-identity.md#792a2b1-uat-scenarios), [792a3o1a The origin remote names the repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3o1b Without an origin the top-level directory names the repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3o1c Outside a git repository there is no repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3p1a The git email names the user](../product/note-writing.md#792a3p1-uat-scenarios), [792a3p1b The OS username is the fallback](../product/note-writing.md#792a3p1-uat-scenarios),
[792a3p1c No detectable user](../product/note-writing.md#792a3p1-uat-scenarios).

Rationale: an empty `user: ""` or `vault: ""` would look like a real value to every later reader.
Identity backfill, or a later amend that re-stamps, fills a missing user in.

#### 792b11c3a Scenarios

```gherkin
Scenario: 792b11c3a1 All three fields are stamped
  Given repository git@github.com:example/vault.git, user agent@example.com and vault name work
  When a fact or feedback note is first written
  Then it carries repo, user and vault with those values, and no warning is logged

Scenario: 792b11c3a2 An empty user is left out
  Given user detection resolves empty and the vault name is personal
  When a note is first written
  Then it has no user: key, carries vault: personal, and one warning says user detection resolved empty

Scenario: 792b11c3a3 An empty vault name is left out
  Given no vault name was resolved
  When a note is first written
  Then it has no vault: key, and a warning names the vault
```

### 792b11c4 Re-stamp

Requirement: [792b11c4 Re-stamp](#792b11c4-re-stamp) When a revision re-stamps a note, the note-provenance MUST overwrite the
note's repository, user and vault with the resolved repository, user and vault name; when the
resolved user is empty it MUST keep a non-empty user the note already has, logging one warning that
names the note and says user detection resolved empty, and MUST leave the user out when the note has
none. With no re-stamp to apply, it MUST change nothing.

Supports: [792a4h1a Amend from another environment re-stamps](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1b Amend backfills missing identity](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1c The project field does not stand in for the repo](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1d Empty user detection keeps the prior user](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1e Empty user detection never writes an empty user](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1f Without a vault name the default is stamped](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1g Recording a use keeps the declared identity](../product/note-revision.md#792a4h1-uat-scenarios).

Rationale: the alternatives for an empty user were to refuse the revision, which blocks content
edits on a machine with broken identity, or to keep the old value silently, which hides the broken
environment (#776). The repository keeps fresh detection even when empty, because a working
directory outside git is legitimate.

#### 792b11c4a Scenarios

```gherkin
Scenario: 792b11c4a1 All three are overwritten
  Given a note with repo old-repo, user old-user and vault old-vault, and resolved repository new-repo, user bob and vault name new-vault
  When the note is re-stamped
  Then its repo is new-repo, its user bob and its vault new-vault

Scenario: 792b11c4a2 An empty user keeps the note's user
  Given a fact, feedback or runbook note whose user is alice, and user detection resolves empty
  When the note is re-stamped
  Then its user is still alice, and exactly one warning names the note and says user detection resolved empty

Scenario: 792b11c4a3 An empty user with none on the note is left out
  Given a fact, feedback or runbook note with no user, and user detection resolves empty
  When the note is re-stamped
  Then it has no user: key, and no warning is logged

Scenario: 792b11c4a4 No re-stamp changes nothing
  Given a note with repo r, user u and vault v
  When a revision applies no re-stamp
  Then its repo, user and vault are still r, u and v
```

### 792b11c5 Repository for a backfill

Requirement: [792b11c5 Repository for a backfill](#792b11c5-repository-for-a-backfill) The note-provenance MUST, when backfilling a note's identity, resolve the note's
repository as the note's `project:` when it is non-empty, and otherwise as the repository detection of the
backfill's working directory ([792b11c1 Repository detection](#792b11c1-repository-detection)), which may itself be empty.

Supports: [792a10k1b The project field is preferred](../product/vault-migration.md#792a10k1-uat-scenarios), [792a10k1c Without a project field the repo is detected](../product/vault-migration.md#792a10k1-uat-scenarios).

Rationale: one backfill run has one working directory but may stamp notes from several repositories, so the
note's own `project:` is the better per-note signal. An amend always uses fresh detection.

#### 792b11c5a Scenarios

```gherkin
Scenario: 792b11c5a1 The project field is preferred
  Given a note whose project: is non-empty and a detected repository
  When the repository is resolved for a backfill
  Then it is the project: value

Scenario: 792b11c5a2 Without a project the detection is used
  Given a note with no project: and a detected repository
  When the repository is resolved for a backfill
  Then it is the detected repository

Scenario: 792b11c5a3 Neither resolves
  Given a note with no project: and no detected repository
  When the repository is resolved for a backfill
  Then it is empty
```

### 792b11c6 Notes a backfill could stamp

Requirement: [792b11c6 Notes a backfill could stamp](#792b11c6-notes-a-backfill-could-stamp) The note-provenance MUST judge a note stampable when it is a `fact` or `feedback` note
whose frontmatter parses, read as its LF form, that has no `user:` and either has no `vault:` or a user
resolves now ([792b11c2 User detection](#792b11c2-user-detection)). It MUST judge a vault to hold such a note when any file at its root is stampable, and
MUST judge it not to when its directory or a file cannot be read.

Supports: [792a10j1a Notes missing identity are noticed](../product/vault-migration.md#792a10j1-uat-scenarios), [792a10j1b No notice when nothing is missing](../product/vault-migration.md#792a10j1-uat-scenarios), [792a10j1c No notice for a note that cannot be stamped](../product/vault-migration.md#792a10j1-uat-scenarios), [792a10j1d A CRLF note is noticed](../product/vault-migration.md#792a10j1-uat-scenarios).

#### 792b11c6a Scenarios

```gherkin
Scenario: 792b11c6a1 A note missing identity is found
  Given a fact note with no repo:, user: or vault:
  When the vault is tested
  Then it holds a note a backfill could stamp

Scenario: 792b11c6a2 A CRLF note is found
  Given a fact note missing identity whose lines end in \r\n
  When the vault is tested
  Then it holds a note a backfill could stamp

Scenario: 792b11c6a3 A stamped note is not found
  Given a note that has user:
  When the vault is tested
  Then it does not

Scenario: 792b11c6a4 Other note types are not found
  Given a note that is neither fact nor feedback and has no user:
  When the vault is tested
  Then it does not

Scenario: 792b11c6a5 An unreadable vault is not found
  Given a vault directory that cannot be read
  When the vault is tested
  Then it does not

Scenario: 792b11c6a6 A note with vault: and no resolvable user is not found
  Given a note with vault: but no user:, and no user resolves
  When the vault is tested
  Then it does not
```

### 792b11c7 Identity backfill

Requirement: [792b11c7 Identity backfill](#792b11c7-identity-backfill) The note-provenance MUST, under the vault write lock, stamp each stampable note
([792b11c6 Notes a backfill could stamp](#792b11c6-notes-a-backfill-could-stamp)) as YAML-node edits of its parsed frontmatter in one atomic write: a note with no `vault:` gets
`repo:` ([792b11c5 Repository for a backfill](#792b11c5-repository-for-a-backfill)), `user:` and `vault:` from the current environment, and a note with `vault:` and no `user:`
gets only `user:`. It MUST re-emit `created:` in its quoted form, keep every other key and every anchor on a
key it does not edit, leave a note neither fact nor feedback and a note with unparseable frontmatter unchanged,
and leave a note it would not write unconverted. It MUST refuse a note when a key it would set or its value or
anything beneath it carries a YAML anchor, or the rewritten frontmatter does not decode, leave that note
untouched, stamp the others, and then return one error joining each refusal that names the note. It MUST write
nothing under dry run, and return the number of notes stamped or that would be. It MUST write a stamped CRLF
note as LF and rebuild its sidecar ([792b10c5 Building a sidecar](./embedding-sidecar.md#792b10c5-building-a-sidecar)); and MUST return a failure to lock, list, read, write or
rebuild.

Supports: [792a10k1a The current environment is stamped](../product/vault-migration.md#792a10k1-uat-scenarios), [792a10k1d A backfill twice stamps once](../product/vault-migration.md#792a10k1-uat-scenarios), [792a10k1e An omitted user is filled in](../product/vault-migration.md#792a10k1-uat-scenarios), [792a10k1f A dry run writes nothing](../product/vault-migration.md#792a10k1-uat-scenarios), [792a10k1g Other note types are left alone](../product/vault-migration.md#792a10k1-uat-scenarios), [792a10l1a An unmodeled key is kept](../product/vault-migration.md#792a10l1-uat-scenarios), [792a10l1b An anchor on an unedited key is kept](../product/vault-migration.md#792a10l1-uat-scenarios), [792a10l1c An anchored identity key is refused](../product/vault-migration.md#792a10l1-uat-scenarios), [792a10l1d A CRLF note is converted](../product/vault-migration.md#792a10l1-uat-scenarios),
[792a10l1e An ordinary note is written as before](../product/vault-migration.md#792a10l1-uat-scenarios).

Rationale: re-rendering the whole frontmatter from the typed model dropped every key the model does not
define (#780, #789). Editing the parsed mapping keeps them, and keeps the bytes of an ordinary note as they
were. The lock stops a backfill racing a concurrent `learn` or `amend`.

#### 792b11c7a Scenarios

```gherkin
Scenario: 792b11c7a1 A note with project is stamped from it
  Given a note with project: set and a note with none, both missing identity, and a stamped note
  When the vault is backfilled
  Then both gain repo:, user: and vault:, the first with repo: from its project:, and the stamped note is unchanged

Scenario: 792b11c7a2 A second backfill writes nothing
  Given a vault that has been backfilled
  When it is backfilled again
  Then no note is written and the count is 0

Scenario: 792b11c7a3 An omitted user is filled in
  Given a note with repo: and vault: work and no user:, and a user that resolves
  When the vault is backfilled
  Then the note gains user:, and repo: and vault: are unchanged; and where no user resolves it is not written

Scenario: 792b11c7a4 A dry run writes nothing
  Given notes missing identity
  When the vault is backfilled with dry run
  Then nothing is written and the count is the number that would be stamped

Scenario: 792b11c7a5 Unrelated content is skipped
  Given a note without frontmatter and a vocab definition note
  When the vault is backfilled
  Then neither fails nor changes, and a read error is returned

Scenario: 792b11c7a6 An unmodeled key is kept
  Given a note with a key the typed model does not define, or a nested map
  When the vault is backfilled
  Then the note is stamped and the key keeps its value

Scenario: 792b11c7a7 An anchor on an unedited key is kept
  Given a note whose source: &src test is aliased by another key
  When the vault is backfilled
  Then both decode to test

Scenario: 792b11c7a8 An anchored identity key is refused
  Given a flagged note whose user: carries an anchor that another key aliases, and a second flagged note
  When the vault is backfilled
  Then the second is stamped, the first is byte-identical, and the error names the first and user

Scenario: 792b11c7a9 A CRLF note is written as LF
  Given a flagged CRLF note
  When the vault is backfilled
  Then it is written without \r\n, byte-identical to the backfill of its LF form, and its sidecar is rebuilt

Scenario: 792b11c7a10 An ordinary note is written as before
  Given a note holding only keys the typed model defines and no anchors
  When the vault is backfilled
  Then the bytes are what the earlier typed re-render wrote

Scenario: 792b11c7a11 A write or rebuild failure is returned
  Given a write or a sidecar rebuild that fails
  When the vault is backfilled
  Then the error is returned

Scenario: 792b11c7a12 The report records the run
  Given a backfill that stamped notes
  When it finishes
  Then the report records the count and the missing-identity flag is re-checked, and under dry run it is left as it was
```
