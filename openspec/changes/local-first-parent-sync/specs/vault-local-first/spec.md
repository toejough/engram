## ADDED Requirements

### Requirement: ENGRAM_SERVER SHALL be a hard error
When the `ENGRAM_SERVER` environment variable is set to a non-empty value, every `engram` subcommand, including `engram serve`, SHALL exit non-zero before reading or writing any vault, chunk index, or network endpoint. The error message SHALL name `ENGRAM_SERVER` as no longer supported and SHALL tell the user to set `ENGRAM_PARENT` to the same URL instead. No subcommand SHALL treat `ENGRAM_SERVER` as an alias for `ENGRAM_PARENT`, and no thin-client mode SHALL remain.

#### Scenario: A command run with ENGRAM_SERVER set fails fast
- **WHEN** `engram query --phrase x` runs with `ENGRAM_SERVER=http://host:8093`
- **THEN** it exits non-zero, its stderr says to set `ENGRAM_PARENT=http://host:8093` instead, and no vault file, chunk file, or HTTP request is touched

#### Scenario: Writes are refused too
- **WHEN** `engram learn fact ...` runs with `ENGRAM_SERVER` set
- **THEN** it exits non-zero with the same error, and no note is written locally or remotely

#### Scenario: serve is not exempt
- **WHEN** `engram serve --addr 127.0.0.1:8093` runs with `ENGRAM_SERVER` set
- **THEN** it exits non-zero with the same error, and it binds no port

#### Scenario: ENGRAM_SERVER is not honored as an alias
- **WHEN** `ENGRAM_SERVER` is set and `ENGRAM_PARENT` is not
- **THEN** no command contacts the `ENGRAM_SERVER` URL

### Requirement: Every environment SHALL have its own local vault, created on first use
Every subcommand that resolves a vault path SHALL ensure that an initialized vault exists at that path before it runs. If none exists, it SHALL create one: the `.obsidian/` directory and starter files that `engram learn` creates today, plus a vault ID and the exchange state directory. It SHALL then print exactly one stderr line naming the created path. This SHALL apply whether or not `ENGRAM_PARENT` is set. The vault's root `.gitignore` SHALL NOT be modified for an existing vault.

#### Scenario: A read command creates the vault
- **WHEN** `engram query --phrase x` runs against a vault path that does not exist
- **THEN** an initialized vault exists at that path afterwards, stderr carries one line naming it, and the query returns an empty result without error

#### Scenario: amend and activate no longer fail on a missing vault
- **WHEN** `engram amend` or `engram activate` runs against a vault path that does not exist
- **THEN** the vault is created first, and the command then fails or succeeds on its own merits (for example, target not found), never on a vault lock error

#### Scenario: A child host gets its own vault
- **WHEN** a host with `ENGRAM_PARENT` set runs its first `engram query`
- **THEN** a local vault is created at the resolved local path, separate from the parent's vault

#### Scenario: An existing vault is untouched by read-only use
- **WHEN** a read-only command runs against an existing initialized vault while no parent is configured
- **THEN** no vault file is created or modified, and no creation line is printed

### Requirement: Every vault SHALL have a stable vault ID
Every vault SHALL have a random identifier stored in the tracked file `<vault>/.engram-vault-id`. A new vault SHALL get one when it is created. An existing vault without one SHALL get one only when `engram serve` starts on it, or when a command first contacts a parent from it. The file SHALL be created with exclusive create and then re-read, so that concurrent creators all end up using the single ID that won. The random source SHALL be injected, so tests are deterministic. `engram update` SHALL print a notify-only notice while the file exists but is not committed in a git-backed vault. The ID SHALL change only through `engram vault-id --regenerate`.

#### Scenario: serve stamps an existing vault
- **WHEN** `engram serve` starts on an existing vault that has no `.engram-vault-id`
- **THEN** the file is created with a new ID, and later starts reuse it unchanged

#### Scenario: Concurrent creation converges
- **WHEN** two processes create the ID for the same vault at the same moment
- **THEN** exactly one file is written, and both processes use its ID

#### Scenario: Uncommitted ID is flagged
- **WHEN** `engram update` runs on a git-backed vault whose `.engram-vault-id` is untracked
- **THEN** its report includes a notice asking for a one-time vault commit

### Requirement: Exchange state SHALL live outside the vault's tracked files
Transient exchange state SHALL live in `<vault>/.engram/`: the outbox, declined pulls, and the parent cache with its backoff. The directory SHALL contain a `.gitignore` whose only pattern is `*`, so that git ignores the directory's contents, including that `.gitignore`. No exchange state SHALL be written to the vault root or to the root `.gitignore`.

#### Scenario: State does not dirty the vault's git
- **WHEN** an offer is queued in a git-backed vault
- **THEN** `git status --porcelain` in the vault shows no change caused by exchange state

### Requirement: A copied or cloned vault SHALL NOT exchange until its identity is resolved
When the vault ID is created, the untracked exchange state SHALL record the vault's location: the ID, the hostname, and the absolute vault path. Before any exchange (serving, merging, offering, or pulling down), a command SHALL compare that record with the current vault ID, hostname and path. When the record is missing (a fresh `git clone`) or differs (a `cp -R`, or a move), the command SHALL do no exchange, `engram serve` SHALL refuse to start, and exactly one warning SHALL name both `engram vault-id --regenerate` and `engram vault-id --claim`.
- `engram vault-id --regenerate` SHALL mint a new ID, rewrite `.engram-vault-id` and the location record, and change no note.
- `engram vault-id --claim` SHALL rewrite the location record only.
- `engram vault-id` with no flag SHALL print the ID and the result of the location check.

#### Scenario: A cloned child is caught before it offers
- **WHEN** vault A is cloned with git to a new host, and the clone runs `engram learn` with `ENGRAM_PARENT` set
- **THEN** the note is written locally, no offer is sent, and the warning names both `engram vault-id` remedies

#### Scenario: Regenerate gives a copy its own identity
- **WHEN** `engram vault-id --regenerate` runs in a copied vault
- **THEN** its ID differs from the original's, exchange resumes, and no note file changed

#### Scenario: Claim accepts a moved vault
- **WHEN** a vault is moved to a new path and `engram vault-id --claim` runs
- **THEN** the ID is unchanged and exchange resumes

### Requirement: A vault SHALL NOT exchange with itself
When a parent reports the same vault ID as the local vault, the command SHALL NOT merge, offer, or pull down. It SHALL print exactly one warning naming `engram vault-id --regenerate`, and SHALL behave as if no parent were configured.

#### Scenario: A copied vault served back to itself
- **WHEN** a copy of vault V is served and V's environment points `ENGRAM_PARENT` at it
- **THEN** `engram query` returns local-only results with a warning naming `engram vault-id --regenerate`, and no offer is sent
