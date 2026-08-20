## Purpose

Every note gets structured, auto-detected provenance — which repo, which person, and which vault instance produced it — so origin survives beyond the existing free-text `source:` description and can be relied on by downstream consumers (filtering, attribution, future multi-vault exchange).

## ADDED Requirements

### Requirement: Repo field auto-detected at note creation
`engram learn` SHALL stamp the note's `repo:` frontmatter field from the git remote `origin` URL of the working directory, falling back to the git root directory's basename when no `origin` remote is configured, and omitting the field entirely when the working directory is not inside a git repository.

#### Scenario: Origin remote configured
- **WHEN** `engram learn` runs in a working directory inside a git repo with an `origin` remote configured
- **THEN** the note's `repo:` frontmatter field is set to the `origin` remote's URL

#### Scenario: No origin remote
- **WHEN** `engram learn` runs in a working directory inside a git repo with no `origin` remote configured
- **THEN** the note's `repo:` frontmatter field is set to the basename of the git repo's top-level directory

#### Scenario: Not inside a git repo
- **WHEN** `engram learn` runs in a working directory that is not inside any git repository
- **THEN** the note's frontmatter has no `repo:` field

### Requirement: User field auto-detected at note creation
`engram learn` SHALL stamp the note's `user:` frontmatter field from `git config user.email` resolved at the working directory, falling back to the machine's OS username when no `user.email` is configured.

#### Scenario: Git user.email configured
- **WHEN** `engram learn` runs where `git config user.email` resolves to a non-empty value (repo-local or global config)
- **THEN** the note's `user:` frontmatter field is set to that email address

#### Scenario: No git user.email configured
- **WHEN** `engram learn` runs where `git config user.email` resolves to nothing
- **THEN** the note's `user:` frontmatter field is set to the OS username of the process running `engram learn`

### Requirement: Vault field resolved from explicit configuration
`engram learn` SHALL stamp the note's `vault:` frontmatter field by resolving, in order: a `--vault-name` flag, then an `ENGRAM_VAULT_NAME` environment variable, then the default value `"personal"` — the same flag-then-env-then-default order the existing `--vault`/`ENGRAM_VAULT_PATH` resolution already uses.

#### Scenario: Vault name flag supplied
- **WHEN** `engram learn --vault-name <name>` is run
- **THEN** the note's `vault:` frontmatter field is set to `<name>`

#### Scenario: Vault name from environment
- **WHEN** `engram learn` runs with `ENGRAM_VAULT_NAME` set in the environment and no `--vault-name` flag supplied
- **THEN** the note's `vault:` frontmatter field is set to the environment variable's value

#### Scenario: Vault name defaulted
- **WHEN** `engram learn` runs with neither `--vault-name` nor `ENGRAM_VAULT_NAME` set
- **THEN** the note's `vault:` frontmatter field is set to `"personal"`

### Requirement: Amend re-stamps identity fields on every write
`engram amend` SHALL re-detect and overwrite a note's `repo:`, `user:`, and `vault:` frontmatter fields on every call, using the same detection as `engram learn` (see the three requirements above), regardless of what the note previously held — unlike `source:`, `project:`, `issue:`, and `tier:`, which continue to be preserved verbatim through amend.

#### Scenario: Amend from a different environment than the note's origin
- **WHEN** `engram amend` rewrites a note whose `repo:`, `user:`, or `vault:` values differ from the environment `amend` is currently running in
- **THEN** the amended note's `repo:`, `user:`, and `vault:` values are overwritten with the current environment's freshly detected values

#### Scenario: Amend backfills missing identity fields
- **WHEN** `engram amend` rewrites a note written before this capability existed (no `repo:`, `user:`, or `vault:` frontmatter fields present)
- **THEN** the amended note gains freshly detected `repo:`, `user:`, and `vault:` fields, same as any other amend

#### Scenario: Amend does not trigger re-embed for identity-only changes
- **WHEN** `engram amend` is invoked with no content-changing flags (situation/subject/predicate/object/behavior/impact/action all unspecified), and `repo:`/`user:`/`vault:` are the only fields that change
- **THEN** the note's vector sidecar is not re-embedded — identity re-stamping is a provenance-only change

### Requirement: Repo detection prefers the note's own project field during backfill only
`engram update --backfill-identity` SHALL prefer the note's existing `project:` field over a fresh git-remote read when `project:` is non-empty. `engram amend` SHALL NOT use this fallback — its `repo:` re-stamp always uses fresh detection from the current working directory, same as `learn`, regardless of the note's `project:` value.

#### Scenario: Backfilling a note with a project field set
- **WHEN** `engram update --backfill-identity` stamps `repo:` on a note whose `project:` field is non-empty
- **THEN** the note's `repo:` field is set to the `project:` value, not the current working directory's git remote

#### Scenario: Backfilling a note with no project field
- **WHEN** `engram update --backfill-identity` stamps `repo:` on a note whose `project:` field is empty or absent
- **THEN** the note's `repo:` field is freshly detected from the current working directory, same as `learn`

#### Scenario: Amend ignores the project field
- **WHEN** `engram amend` re-stamps `repo:` on a note whose `project:` field differs from the current working directory's repo
- **THEN** the note's `repo:` field is set to the current working directory's freshly detected repo, not the `project:` value

### Requirement: Backfill for pre-existing notes missing identity fields
`engram update` SHALL detect notes missing `repo:`, `user:`, or `vault:` frontmatter fields and surface a notify-only notice naming the `--backfill-identity` flag, following the same detect-and-notify convention as the vocab-migration, Luhmann-branching, and chunk-pruning notices. `engram update --backfill-identity` SHALL rewrite each flagged note's `repo:`, `user:`, and `vault:` fields using the same `user:`/`vault:` detection as `learn`/`amend`, and the project-field-preferred `repo:` fallback described above, leaving all other note fields unchanged.

#### Scenario: Update detects notes missing identity fields
- **WHEN** `engram update` runs and the vault contains one or more notes with no `repo:`, `user:`, or `vault:` frontmatter fields
- **THEN** the update report includes a notify-only notice naming `engram update --backfill-identity`

#### Scenario: No notice when nothing is missing
- **WHEN** `engram update` runs and every note already has `repo:`, `user:`, and `vault:` fields
- **THEN** no identity-backfill notice appears in the update report

#### Scenario: Backfill stamps current environment onto flagged notes
- **WHEN** `engram update --backfill-identity` runs
- **THEN** every note previously missing `repo:`/`user:`/`vault:` gains those fields, `user:`/`vault:` from the current environment's detection and `repo:` per the project-field-preferred fallback above

#### Scenario: Backfill is idempotent
- **WHEN** `engram update --backfill-identity` runs a second time with no newly-missing notes
- **THEN** no notes are modified
