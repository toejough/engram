## ADDED Requirements

### Requirement: ENGRAM_SERVER SHALL be a hard error
When the `ENGRAM_SERVER` environment variable is set to a non-empty value, every `engram` subcommand SHALL exit non-zero before reading or writing any vault, chunk index, or network endpoint. The error message SHALL name `ENGRAM_SERVER` as no longer supported and SHALL tell the user to set `ENGRAM_PARENT` to the same URL instead. No subcommand SHALL treat `ENGRAM_SERVER` as an alias for `ENGRAM_PARENT`, and no thin-client mode SHALL remain.

#### Scenario: A command run with ENGRAM_SERVER set fails fast
- **WHEN** `engram query --phrase x` runs with `ENGRAM_SERVER=http://host:8093`
- **THEN** it exits non-zero, its stderr says to set `ENGRAM_PARENT=http://host:8093` instead, and no vault file, chunk file, or HTTP request is touched

#### Scenario: Writes are refused too
- **WHEN** `engram learn fact ...` runs with `ENGRAM_SERVER` set
- **THEN** it exits non-zero with the same error, and no note is written locally or remotely

#### Scenario: ENGRAM_SERVER is not honored as an alias
- **WHEN** `ENGRAM_SERVER` is set and `ENGRAM_PARENT` is not
- **THEN** no command contacts the `ENGRAM_SERVER` URL

### Requirement: Every environment SHALL have its own local vault, created on first use
Every subcommand that resolves a vault path SHALL ensure that an initialized vault exists at that path before it runs. If none exists, it SHALL create one (the same `.obsidian/` directory and starter files that `engram learn` creates today) and SHALL print exactly one stderr line naming the created path. This applies regardless of whether `ENGRAM_PARENT` is set. An existing vault SHALL NOT be modified by this check, except that the vault's `.gitignore` SHALL gain the outbox file entry (capability `vault-parent-offers`) when that entry is missing.

#### Scenario: A read command creates the vault
- **WHEN** `engram query --phrase x` runs against a vault path that does not exist
- **THEN** an initialized vault exists at that path afterwards, stderr carries one line naming it, and the query returns an empty result without error

#### Scenario: amend and activate no longer fail on a missing vault
- **WHEN** `engram amend` or `engram activate` runs against a vault path that does not exist
- **THEN** the vault is created first, and the command then fails or succeeds on its own merits (for example, target not found), never on a vault lock error

#### Scenario: A child host gets its own vault
- **WHEN** a host with `ENGRAM_PARENT` set runs its first `engram query`
- **THEN** a local vault is created at the resolved local path, separate from the parent's vault

#### Scenario: An existing vault is untouched
- **WHEN** any command runs against an existing initialized vault whose `.gitignore` already names the outbox file
- **THEN** no vault file is created or modified by the check, and no creation line is printed
