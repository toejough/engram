# Documentation Framework Adoption - Technical Spec

## 792b1 Orientation

- **Product problem:** [`792a1`](./792a_product-spec.md#792a1-problem-statement). Engram's shipped
  behavior has no specifications in the framework's format, and its tests can't be traced to any
  specification.
- **Current behavior:** The system already exists and is unchanged by this change. Its technical
  contracts are described only by the legacy C4 views, the ADRs, the invariants document, and the
  technical parts of the OpenSpec specs.
- **Insufficiency:** Those sources do not follow the framework's component qualification, ID, or
  product-traceability rules.
- **Recommended solution:** Document each qualifying component of the existing system as one
  **Added** component section. Map each section to the product scenarios it supports, and rename the
  existing tests that verify its scenarios.

This change describes the existing system and selects no new design. Its technical sections
therefore omit the guide's recommended-impact, relative-option-impact, and alternative-options
sections, because no options are being compared. They also omit component diffs: every component is
retained and documented as Added. Each component section instead uses the per-component structure in
[`792c3`](./792c_implementation-plan.md#792c3-spec-section-templates): orientation, sequences,
behavior requirements with Gherkin, and product-scenario mapping. Rationale appears under the
requirement or orientation it justifies.

Component sections `792b2` through `792b39` are added by the nodes of
[`792c_implementation-plan.md`](./792c_implementation-plan.md).

## 792b2 Added: command-line-surface

Canonical target: `docs/specs/technical/command-line-surface.md` (title `Command Line Surface - Technical Spec`).

### 792b2a Orientation

- Ownership: Repository component (caller), crossing the process argv, environment and stdio
  boundary through the targ library.
- Responsibility: turns one process invocation into one command run. It parses arguments and
  environment into a typed request, resolves the vault path and vault name, runs the first-use vault
  step before every vault-resolving command, routes to the command, and maps the result to the
  output streams and the exit status.
- Commands: `learn` (`fact`, `feedback`, `qa`, `runbook`), `query`, `query-chunks`, `show`,
  `show-chunk`, `amend`, `activate`, `resituate`, `ingest`, `prune`, `count`, `check`, `embed`
  (`apply`, `status`), `vocab` (`bootstrap`, `tag-definitions`, `propose`, `stats`, `refit`),
  `update`, `register-skills`, `serve` and `vault-id`. Every command except `prune`, `query-chunks`
  and `show-chunk` resolves a vault. Each command's behavior is specified by its capability.
- Current realization (informational): `internal/cli/targets.go` (`Targets`, `runInVault`,
  `newErrHandler`), `internal/cli/cli.go`, `resolveVault` in `internal/cli/learn.go`,
  `resolveVaultName` in `internal/cli/identity.go`, `DataDirFromHome` in `internal/cli/targets.go`;
  `cmd/engram/main.go` wires the production primitives.
- Qualification: responsibility, it owns parsing, precedence and routing for every command without
  depending on how any command works; ownership, its routing, gating and exit mapping are distinct
  from each command's own behavior; independent verification, its precedence and gating are tested
  through dispatch with injected environment and filesystem; stability, the surface survives any
  refactoring of command internals; necessity, every product capability is reached through it.
- Supported capabilities: 792a2 vault-identity, 792a3 note-writing, 792a4 note-revision. Later capabilities add their own routing
  requirements here (792d30, 792d51).

Rationale: each `engram <subcommand>` is its own process. Commands never call each other in
process; a skill reads one command's stdout and runs the next. (An earlier C3 draft drew an
in-process `query`→`learn` edge that does not exist.) Each skill step runs the CLI as a fresh
subprocess, and the skill layer reaches the vault only through it: it reads notes through
`engram show`, `engram show-chunk` and the query payload, and writes through `engram amend` and
`engram learn`, never by editing vault files directly.

### 792b2b Sequences

```mermaid
sequenceDiagram
  participant Caller as Skill or operator
  participant CLI as command-line-surface
  participant Registry as vault-identity-registry
  participant Command as Selected command
  Caller->>CLI: engram subcommand with flags and environment
  alt undefined flag or missing required flag
    CLI-->>Caller: error naming the flag on stderr, non-zero exit, command not run (792b2c5)
  else arguments parsed
    CLI->>CLI: resolve vault path (792b2c1) and vault name (792b2c2)
    CLI->>Registry: ensure the vault exists (792b2c3)
    alt vault ready
      Registry-->>CLI: ready, created or already present
      CLI->>Command: run the command
      Command-->>CLI: result
      alt command succeeded
        CLI-->>Caller: output on stdout, exit 0 (792b2c4)
      else command failed
        CLI-->>Caller: error on stderr, non-zero exit (792b2c4)
      end
    else first-use step failed
      Registry-->>CLI: error
      CLI-->>Caller: error on stderr, non-zero exit, command not run (792b2c3)
    end
  end
```

### 792b2c Behavior Requirements

#### 792b2c1 Vault path resolution

Requirement: 792b2c1 The command-line-surface MUST resolve the vault path to the `--vault` flag
when it is non-empty, else to `ENGRAM_VAULT_PATH` when it is non-empty, else to
`$XDG_DATA_HOME/engram/vault` when `XDG_DATA_HOME` is non-empty, else to
`$HOME/.local/share/engram/vault`, and MUST return that path without checking that it exists.

Supports: 792a2a1a, 792a2a1b, 792a2a1c, 792a2a1d.

##### 792b2c1a Scenarios

```gherkin
Scenario: 792b2c1a1 The flag wins
  Given the --vault flag is /explicit and ENGRAM_VAULT_PATH is /env
  When the vault path is resolved
  Then the path is /explicit

Scenario: 792b2c1a2 The environment is used without a flag
  Given the --vault flag is empty and ENGRAM_VAULT_PATH is /from/env
  When the vault path is resolved
  Then the path is /from/env

Scenario: 792b2c1a3 XDG_DATA_HOME sets the default
  Given no flag, no ENGRAM_VAULT_PATH, and XDG_DATA_HOME is /xdg/data
  When the vault path is resolved
  Then the path is /xdg/data/engram/vault

Scenario: 792b2c1a4 The home directory is the last fallback
  Given no flag, no ENGRAM_VAULT_PATH, no XDG_DATA_HOME, and home /home/u
  When the vault path is resolved
  Then the path is /home/u/.local/share/engram/vault
```

#### 792b2c2 Vault name resolution

Requirement: 792b2c2 The command-line-surface MUST resolve the vault name to the `--vault-name`
flag when it is non-empty, else to `ENGRAM_VAULT_NAME` when it is non-empty, else to `personal`.

Supports: 792a2b1a, 792a2b1b, 792a2b1c, 792a4h1f.

##### 792b2c2a Scenarios

```gherkin
Scenario: 792b2c2a1 The default is personal
  Given neither --vault-name nor ENGRAM_VAULT_NAME is set
  When the vault name is resolved
  Then the name is personal

Scenario: 792b2c2a2 The environment names the vault
  Given --vault-name is empty and ENGRAM_VAULT_NAME is team
  When the vault name is resolved
  Then the name is team

Scenario: 792b2c2a3 The flag wins
  Given --vault-name is work and ENGRAM_VAULT_NAME is personal-env
  When the vault name is resolved
  Then the name is work
```

#### 792b2c3 First-use gating

Requirement: 792b2c3 For every vault-resolving command, the command-line-surface MUST ask the
vault-identity-registry to ensure the vault at the resolved path before running the command, and
MUST NOT run the command when that step fails.

Supports: 792a2e1a, 792a2e1b, 792a2e1c, 792a2e1d, 792a2f1a, 792a2g1a.

Rationale: one shared first-use step in dispatch, rather than one per command, makes every command
create a missing vault the same way (#766).

##### 792b2c3a Scenarios

```gherkin
Scenario: 792b2c3a1 Every vault-resolving command ensures the vault first
  Given no vault exists at the resolved path
  When any vault-resolving command is dispatched
  Then the vault is created before the command runs, and the command then runs against it

Scenario: 792b2c3a2 A failed first-use step stops the command
  Given the resolved vault path names a regular file
  When a vault-resolving command is dispatched
  Then the error is reported and the command does not run
```

#### 792b2c4 Result reporting

Requirement: 792b2c4 The command-line-surface MUST write a failed command's error to stderr and exit
with a non-zero status, and MUST neither exit nor write to stderr on the command's behalf when the
command succeeds.

Supports: 792a2e1b, 792a2j1c, 792a2n1b, 792a2o1a, 792a3f1b, 792a3g1a, 792a3k1a, 792a3k1b, 792a3k1c,
792a3k1d, 792a3r1a, 792a3s1a, 792a4a1b, 792a4c1a, 792a4f1a, 792a4k1a, 792a4k1b, 792a4l1b, 792a4l1c,
792a4n1c.

##### 792b2c4a Scenarios

```gherkin
Scenario: 792b2c4a1 A failure is reported and exits non-zero
  Given a command that fails with the error "boom"
  When its result is reported
  Then stderr contains "boom" and the exit status is non-zero

Scenario: 792b2c4a2 A success neither exits nor writes stderr
  Given a command that succeeds
  When its result is reported
  Then nothing is written to stderr and no exit is requested
```

#### 792b2c5 Argument refusal

Requirement: 792b2c5 The command-line-surface MUST refuse an invocation that passes a flag the
command does not define, or omits a flag the command marks required, by writing an error naming the
flag to stderr and exiting non-zero without running the command.

Supports: 792a3e1a, 792a3f1a, 792a3f1c, 792a3g1a, 792a3r1a.

##### 792b2c5a Scenarios

```gherkin
Scenario: 792b2c5a1 An undefined flag is refused
  Given a command that defines no --skill-hash flag
  When it is dispatched with --skill-hash
  Then the exit status is non-zero and the command does not run

Scenario: 792b2c5a2 A missing required flag is refused
  Given a command that marks --done-when required
  When it is dispatched without --done-when
  Then stderr names --done-when, the exit status is non-zero, and the command does not run
```

### 792b2d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a2a1a | 792b2c1 |
| 792a2a1b | 792b2c1 |
| 792a2a1c | 792b2c1 |
| 792a2a1d | 792b2c1 |
| 792a2b1a | 792b2c2 |
| 792a2b1b | 792b2c2 |
| 792a2b1c | 792b2c2 |
| 792a2e1a | 792b2c3 |
| 792a2e1b | 792b2c3, 792b2c4 |
| 792a2e1c | 792b2c3 |
| 792a2e1d | 792b2c3 |
| 792a2f1a | 792b2c3 |
| 792a2g1a | 792b2c3 |
| 792a2j1c | 792b2c4 |
| 792a2n1b | 792b2c4 |
| 792a2o1a | 792b2c4 |
| 792a3e1a | 792b2c5 |
| 792a3f1a | 792b2c5 |
| 792a3f1b | 792b2c4 |
| 792a3f1c | 792b2c5 |
| 792a3g1a | 792b2c4, 792b2c5 |
| 792a3k1a | 792b2c4 |
| 792a3k1b | 792b2c4 |
| 792a3k1c | 792b2c4 |
| 792a3k1d | 792b2c4 |
| 792a3r1a | 792b2c4, 792b2c5 |
| 792a3s1a | 792b2c4 |
| 792a4a1b | 792b2c4 |
| 792a4c1a | 792b2c4 |
| 792a4f1a | 792b2c4 |
| 792a4h1f | 792b2c2 |
| 792a4k1a | 792b2c4 |
| 792a4k1b | 792b2c4 |
| 792a4l1b | 792b2c4 |
| 792a4l1c | 792b2c4 |
| 792a4n1c | 792b2c4 |

```gherkin
Scenario: 792a2a1a The flag wins over the environment
  Given ENGRAM_VAULT_PATH is set to /env/vault
  When the user runs an engram command with --vault /flag/vault
  Then the command uses /flag/vault

Scenario: 792a2a1b The environment variable is used without a flag
  Given ENGRAM_VAULT_PATH is set to /env/vault
  When the user runs an engram command without --vault
  Then the command uses /env/vault

Scenario: 792a2a1c The XDG data directory is the default
  Given neither --vault nor ENGRAM_VAULT_PATH is set and XDG_DATA_HOME is /data
  When the user runs an engram command
  Then the command uses /data/engram/vault

Scenario: 792a2a1d The home directory is the last fallback
  Given neither --vault, ENGRAM_VAULT_PATH nor XDG_DATA_HOME is set
  When the user runs an engram command
  Then the command uses ~/.local/share/engram/vault

Scenario: 792a2b1a The default vault name is personal
  Given neither --vault-name nor ENGRAM_VAULT_NAME is set
  When the user writes a note with engram learn
  Then the note's vault: field is personal

Scenario: 792a2b1b The environment names the vault
  Given ENGRAM_VAULT_NAME is set to team
  When the user writes a note with engram learn without --vault-name
  Then the note's vault: field is team

Scenario: 792a2b1c The flag wins over the environment
  Given ENGRAM_VAULT_NAME is set to team
  When the user writes a note with engram learn --vault-name org
  Then the note's vault: field is org

Scenario: 792a2e1a A read command creates the vault
  Given no vault exists at the resolved path
  When the user runs engram query --phrase x
  Then an initialized vault exists at that path, stderr carries one line naming it, and the query returns an empty result without error

Scenario: 792a2e1b amend and activate fail only on their own merits
  Given no vault exists at the resolved path
  When the user runs engram amend or engram activate on a target that does not exist
  Then the vault is created first, and the command fails because the target is not found, never on a vault lock error

Scenario: 792a2e1c A child host gets its own vault
  Given a host with ENGRAM_PARENT set and no vault at its resolved local path
  When the user runs engram query for the first time
  Then a local vault is created at the resolved local path, separate from the parent's vault

Scenario: 792a2e1d A second run creates nothing
  Given a vault that an engram command has just created
  When the user runs the same command again
  Then no creation line is printed and the vault ID is unchanged

Scenario: 792a2f1a An existing vault is untouched by read-only use
  Given an existing vault with no vault ID and no parent configured
  When the user runs a read-only command against it
  Then no vault file is created or modified, and no creation line is printed

Scenario: 792a2g1a A half-created vault gets its ID
  Given a vault with a .engram/ directory and no .engram-vault-id
  When the user runs any vault-resolving command
  Then .engram-vault-id is created and stderr carries one line saying the missing vault ID was stamped

Scenario: 792a2j1c vault-id reports the corrupt file
  Given a vault whose .engram-vault-id holds junk
  When the user runs engram vault-id
  Then it fails with an error saying the vault ID is invalid

Scenario: 792a2n1b Claim needs an ID
  Given an existing vault with no vault ID
  When the user runs engram vault-id --claim
  Then it fails with an error saying the vault has no vault ID

Scenario: 792a2o1a Both flags are refused
  Given a vault with a vault ID
  When the user runs engram vault-id --regenerate --claim
  Then it fails naming --regenerate, and the vault ID is unchanged
```

```gherkin
Scenario: 792a3e1a A task type is refused
  Given an existing vault
  When the user runs engram learn runbook with --task-type foo
  Then it fails naming --task-type as an undefined flag, and no note is written

Scenario: 792a3f1a A missing situation is refused
  Given an existing vault
  When the user runs engram learn fact without --situation
  Then it fails naming --situation, and no note is written

Scenario: 792a3f1b A blank situation is refused
  Given an existing vault
  When the user runs engram learn feedback with --situation "  "
  Then it fails saying --situation is required, and no note is written

Scenario: 792a3f1c A runbook without done-when is refused
  Given an existing vault
  When the user runs engram learn runbook without --done-when
  Then it fails naming --done-when, and no note is written

Scenario Outline: 792a3g1a An invalid argument is refused before any write
  Given an existing vault
  When the user runs engram learn <kind> with <argument>
  Then it fails with an error saying <problem>, and no note is written

  Examples:
    | kind    | argument               | problem                                         |
    | fact    | no --source            | --source is a missing required flag             |
    | fact    | no --slug              | slug is required                                |
    | fact    | --slug X_y             | slug must match [a-z0-9-]+                      |
    | fact    | --project A            | project slug must match [a-z0-9-]+              |
    | fact    | --issue "a b"          | issue must be non-empty with no whitespace      |
    | fact    | --tag "A B"            | tag must be <family> or <family>/<value>        |
    | runbook | --trigger ab           | trigger must be at least 3 characters           |
    | fact    | --tier L9              | tier must be L2, or L1 or L3                    |
    | fact    | --supersedes "a\|bogus\|c" | supersedes type must be updates, narrows or refutes |

Scenario: 792a3k1a A missing position is refused
  Given an existing vault
  When the user runs engram learn fact without --position
  Then it fails saying the position must be top, continuation or sibling, and no note is written

Scenario: 792a3k1b A continuation without a target is refused
  Given an existing vault
  When the user runs engram learn fact --position continuation without --target
  Then it fails saying a target is required, and no note is written

Scenario: 792a3k1c A sibling of a top-level note is refused
  Given a vault whose note has ID 1
  When the user runs engram learn fact --position sibling --target 1
  Then it fails saying a sibling of a top-level note requires --position top, and no note is written

Scenario: 792a3k1d A malformed target is refused
  Given an existing vault
  When the user runs engram learn fact --position continuation --target abc
  Then it fails saying the ID must start with a digit, and no note is written

Scenario Outline: 792a3r1a An invalid pair is refused before any write
  Given an existing vault
  When the user runs engram learn qa with <argument>
  Then it fails with an error saying <problem>, and neither note is written

  Examples:
    | argument                       | problem                                      |
    | no --question                  | --question is required                       |
    | both --answer and --answer-file | exactly one of --answer or --answer-file is required |
    | neither --answer nor --answer-file | exactly one of --answer or --answer-file is required |
    | --certainty huge               | certainty must be high, medium, or low       |
    | --contributors nope            | contributor not found in vault               |
    | --slug X_y                     | slug must match [a-z0-9-]+                   |
    | no --source                    | --source is a missing required flag          |

Scenario: 792a3s1a A failed answer removes the question
  Given a vault that already holds a file named like today's answer note for slug why-x
  When the user runs engram learn qa --slug why-x
  Then it fails saying the answer note could not be written, and no question note for why-x remains
```

```gherkin
Scenario: 792a4a1b An unknown reference is refused
  Given a vault with no note 9
  When the user runs engram amend --target 9, or engram resituate --note 9 --situation s
  Then it fails saying note "9" was not found, and no note is written

Scenario: 792a4c1a A question-and-answer note is refused
  Given a vault holding an answer note qa.2026-10-09.why.a
  When the user runs engram amend --target qa.2026-10-09.why.a --situation s
  Then it fails saying the note type is unknown, and the note is unchanged

Scenario Outline: 792a4f1a An invalid amend argument is refused before any write
  Given a vault holding fact note 1
  When the user runs engram amend --target 1 with <argument>
  Then it fails with an error saying <problem>, and the note is unchanged

  Examples:
    | argument                     | problem                                        |
    | --supersedes "a\|bogus\|c"   | supersedes type must be updates, narrows or refutes |
    | --supersedes "a\|updates"    | supersedes format must be <basename>\|<type>\|<claim> |
    | --trigger ab                 | trigger must be at least 3 characters          |
    | --chunk-source nope#x        | unresolved chunk-source id "nope#x"            |

Scenario: 792a4h1f Without a vault name the default is stamped
  Given neither --vault-name nor ENGRAM_VAULT_NAME is set
  When the user runs engram amend --target <note> --object new
  Then the note's vault: is personal, never an empty string

Scenario: 792a4k1a Amend refuses an anchored edited key
  Given a fact note whose object: value carries an anchor that another key aliases
  When the user runs engram amend --target <note> --object new
  Then it fails naming object, and the note is unchanged

Scenario: 792a4k1b Resituate refuses an anchored situation
  Given a fact note whose situation: or created: carries an anchor
  When the user runs engram resituate --note <note> --situation new
  Then it fails naming the key, and the note is unchanged

Scenario: 792a4l1b A runbook cannot be resituated
  Given a vault holding runbook note 2
  When the user runs engram resituate --note 2 --situation new
  Then it fails saying the note type runbook is unknown, and the note is unchanged

Scenario: 792a4l1c A failed sidecar rebuild fails the command
  Given embedding fails for the note being resituated
  When the user runs engram resituate --note <note> --situation new
  Then the command fails naming the note, and the note's situation is already new

Scenario: 792a4n1c A missing note fails only its own ref
  Given no parent is configured, a note 3.2026-09-20.local, and no note 7.2026-09-01.gone
  When the user runs engram activate --note 3.2026-09-20.local --note 7.2026-09-01.gone.md
  Then 3.2026-09-20.local's last_used is today, stderr names 7.2026-09-01.gone.md as not found, and the command fails saying 1 of 2 refs failed
```

## 792b3 Added: durable-file-store

Canonical target: `docs/specs/technical/durable-file-store.md` (title `Durable File Store - Technical Spec`).

### 792b3a Orientation

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
  the injection seam; necessity, every vault-identity, note and index write depends on it.
- Supported capabilities: 792a2 vault-identity, 792a3 note-writing, 792a4 note-revision.

Rationale: `os.WriteFile` is not atomic. Before ADR-0013, unlocked non-atomic writes let two
concurrent `engram ingest` or `engram amend` runs corrupt state in production. A same-directory
rename is atomic on POSIX, so a reader sees either the old file or the new one, never a partial
write.

### 792b3b Sequences

```mermaid
sequenceDiagram
  participant Caller as Writer
  participant Store as durable-file-store
  participant OS as Filesystem
  Caller->>Store: replace path atomically with data at perm (792b3c1)
  loop up to 10 candidate temp names beside the target
    Store->>OS: create the temp exclusively and write data (792b3c2)
    alt temp name already exists
      OS-->>Store: exists, try the next candidate
    end
  end
  alt temp created
    Store->>OS: set the temp's exact permissions
    alt permissions set
      Store->>OS: rename the temp onto the target
      alt rename succeeded
        Store-->>Caller: success
      else rename failed
        Store->>OS: remove the temp
        Store-->>Caller: error naming the path, original untouched
      end
    else permissions failed
      Store->>OS: remove the temp
      Store-->>Caller: error naming the path, original untouched
    end
  else creation failed or names exhausted
    Store-->>Caller: error naming the path, nothing created
  end
```

### 792b3c Behavior Requirements

#### 792b3c1 Atomic replace

Requirement: 792b3c1 The durable-file-store MUST replace a file's content by writing the new data
to a uniquely named temp file in the target's directory, setting the temp's permissions to exactly
the requested mode regardless of umask, and renaming the temp onto the target, so that a concurrent
reader sees either the old or the new content. On any failure after the temp is created it MUST
remove the temp, leave the original unchanged, and return an error naming the target path.

Supports: 792a2h1a, 792a2m1a, 792a2n1a, 792a3n1a, 792a3u1a, 792a3u1b, 792a4b1a, 792a4l1a, 792a4m1a,
792a4n1a.

Rationale: the unique temp name (target base, clock nanoseconds, attempt counter) is retried a
bounded number of times on a name collision, so two writers of the same file never share a temp.

##### 792b3c1a Scenarios

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

#### 792b3c2 Exclusive create

Requirement: 792b3c2 The durable-file-store MUST create a file only when no file exists at the path,
writing the data at the requested mode, and MUST fail with an error that names the path and
satisfies `errors.Is(err, fs.ErrExist)` when one does. When writing the data fails, that error
takes precedence over a close error; when the write succeeds and the close fails, the close error
is returned.

Supports: 792a2h1c, 792a3a1a, 792a3b1a, 792a3c1a, 792a3m1a, 792a3q1a, 792a3s1a.

Rationale: exclusive create is the collision backstop for concurrent creators, such as two
processes stamping the same vault ID or racing for the same Luhmann ID.

##### 792b3c2a Scenarios

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

#### 792b3c3 Error identity

Requirement: 792b3c3 The durable-file-store MUST wrap every failed filesystem operation in an error
that names the operation and path and still satisfies `errors.Is` for the underlying error, such as
`fs.ErrNotExist`.

Supports: 792a2e1a, 792a2f1a.

Rationale: callers decide "missing" versus "failed" by `errors.Is`. First-use creation, for
example, creates a vault only when the path is missing.

##### 792b3c3a Scenarios

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

#### 792b3c4 Vault Markdown listing

Requirement: 792b3c4 The durable-file-store MUST list a vault directory as the names of the `.md`
files directly inside it, leaving out subdirectories, other files and the untouched starter
`README.md`, MUST return an empty list when the directory does not exist, and MUST return any other
read error.

Supports: 792a2c1a, 792a2d1a, 792a2d1b, 792a3r1a.

Rationale: this listing is the one shared point where every vault scanner (query, check, embed,
count) stops seeing subdirectories and the starter README.

##### 792b3c4a Scenarios

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

### 792b3d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a2c1a | 792b3c4 |
| 792a2d1a | 792b3c4 |
| 792a2d1b | 792b3c4 |
| 792a2e1a | 792b3c3 |
| 792a2f1a | 792b3c3 |
| 792a2h1a | 792b3c1 |
| 792a2h1c | 792b3c2 |
| 792a2m1a | 792b3c1 |
| 792a2n1a | 792b3c1 |
| 792a3a1a | 792b3c2 |
| 792a3b1a | 792b3c2 |
| 792a3c1a | 792b3c2 |
| 792a3m1a | 792b3c2 |
| 792a3n1a | 792b3c1 |
| 792a3q1a | 792b3c2 |
| 792a3r1a | 792b3c4 |
| 792a3s1a | 792b3c2 |
| 792a3u1a | 792b3c1 |
| 792a3u1b | 792b3c1 |
| 792a4b1a | 792b3c1 |
| 792a4l1a | 792b3c1 |
| 792a4m1a | 792b3c1 |
| 792a4n1a | 792b3c1 |

```gherkin
Scenario: 792a2c1a A Markdown file in a subdirectory is not a note
  Given a vault with one note at its root and one Markdown file in a subdirectory
  When the user runs engram embed status on the vault
  Then it reports a total of 1 note

Scenario: 792a2d1a A fresh vault has no notes
  Given a vault that engram has just created
  When the user runs engram embed status and engram check on it
  Then embed status reports a total of 0 notes and check reports no note missing a sidecar

Scenario: 792a2d1b A user's README is a note
  Given a vault whose README.md the user has replaced with their own text
  When the user runs engram embed status and engram check on it
  Then embed status reports a total of 1 note and check reports 1 note missing a sidecar

Scenario: 792a2h1a serve stamps an existing vault
  Given an existing vault that has no .engram-vault-id
  When engram serve starts on it
  Then the file is created with a new ID, and later starts reuse it unchanged

Scenario: 792a2h1c Concurrent creation converges
  Given an existing vault that has no .engram-vault-id
  When two processes create the ID for it at the same moment
  Then exactly one file is written, and both processes use its ID

Scenario: 792a2m1a Regenerate gives a copy its own identity
  Given a copied vault
  When the user runs engram vault-id --regenerate in it
  Then its ID differs from the original's, exchange resumes, and no note file changed

Scenario: 792a2n1a Claim accepts a moved vault
  Given a vault that was moved to a new path
  When the user runs engram vault-id --claim
  Then the ID is unchanged, engram vault-id then reports location: ok, and exchange resumes
```

```gherkin
Scenario: 792a3a1a A fact note is written
  Given an empty vault
  When the user runs engram learn fact --slug ctx-fact --position top --source "smoke test" --situation "concurrent Go code" --subject goroutines --predicate "leak when" --object "ctx is ignored"
  Then stdout prints the path of 1.<today>.ctx-fact.md at the vault root, whose frontmatter has type: fact and whose body reads "Information learned: when in concurrent Go code, goroutines leak when ctx is ignored."

Scenario: 792a3b1a A feedback note is written
  Given an empty vault
  When the user runs engram learn feedback --slug ctx-rule --position top --source "smoke test" --situation "writing concurrent Go code" --behavior "ignoring ctx" --impact "leaks goroutines" --action "check ctx.Done()"
  Then stdout prints the path of 1.<today>.ctx-rule.md, whose frontmatter has type: feedback and whose body reads "Lesson learned: when writing concurrent Go code, check ctx.Done()."

Scenario: 792a3c1a A runbook note is written
  Given an empty vault
  When the user runs engram learn runbook --slug rb --position top --source src --situation "releasing a build" --done-when "the tag is pushed" --body "1. Run the checks"
  Then stdout prints the path of 1.<today>.rb.md, whose frontmatter has type: runbook, situation and done_when, and whose body is "1. Run the checks"

Scenario: 792a3m1a Concurrent learns get distinct IDs
  Given an existing vault
  When two engram learn fact --position top runs start at the same time
  Then two notes exist with different Luhmann IDs, each holding its own content

Scenario: 792a3n1a A learned note has its sidecar
  Given an empty vault
  When the user runs engram learn fact, feedback, runbook or qa
  Then every note written has a .vec.json sidecar beside it, and engram check reports no note missing a sidecar

Scenario: 792a3q1a A pair is captured
  Given a vault holding note 1.2026-10-09.a-fact
  When the user runs engram learn qa --slug why-x --question "Why X?" --answer "Because." --source src --contributors 1.2026-10-09.a-fact
  Then stdout prints the paths of qa.<today>.why-x.q.md and then qa.<today>.why-x.a.md, the question note ends "Answered by: [[qa.<today>.why-x.a]]", and the answer note ends "Answers: [[qa.<today>.why-x.q]]" and "Contributors: [[1.2026-10-09.a-fact]]"

Scenario: 792a3u1a A runbook is tagged
  Given a vault with a vocabulary term close to a runbook's text
  When the user runs engram learn runbook with that text
  Then the runbook carries the term's vocab tag

Scenario: 792a3u1b The answer is tagged and the question is not
  Given a vault with a vocabulary term close to both a question and its answer
  When the user runs engram learn qa with them
  Then the answer note carries the term's vocab tag, and the question note has no tags: field
```

```gherkin
Scenario: 792a4b1a A fact field is replaced
  Given a fact note whose situation is "testing", subject s, predicate p and object o
  When the user runs engram amend --target <note> --object o2
  Then the note's object: is o2, its other fields are unchanged, its body reads "Information learned: when in testing, s p o2.", and its path is printed

Scenario: 792a4l1a A fact is resituated
  Given a fact note whose situation is "old" and whose body reads "Information learned: when in old, s p o." followed by a related-notes section
  When the user runs engram resituate --note <note> --situation new
  Then situation: is new, the body's first line reads "Information learned: when in new, s p o.", the related-notes section is unchanged, the sidecar is rebuilt, and the note's path is printed

Scenario: 792a4m1a Amend converts a CRLF note
  Given a fact, feedback or runbook note whose lines end in \r\n
  When the user runs engram amend --target <note> with a content flag, --supersedes, --chunk-source or --activate
  Then the amend succeeds, the written note contains no \r\n, it is byte-identical to what the same amend writes for the note's LF form, and its sidecar is fresh

Scenario: 792a4n1a A note's use is recorded
  Given a note with a current sidecar
  When the user runs engram activate --note <basename>
  Then the sidecar's last_used is today, its vectors and content hash are unchanged, the note file is unchanged, and the command succeeds
```

## 792b4 Added: subprocess-runner

Canonical target: `docs/specs/technical/subprocess-runner.md` (title `Subprocess Runner - Technical Spec`).

### 792b4a Orientation

- Ownership: Repository-owned dependency-boundary interaction (operating-system processes).
- Responsibility: runs an external program, such as `git` or `go`, in a chosen directory and hands
  back what it printed, or reports why it failed, telling "program not installed" apart from other
  failures.
- Current realization (informational): `primCommander` in `internal/cli/commander.go` (the
  `update.Commander` interface), over the raw `ExecPrims` that `cmd/engram/main.go` supplies.
  `primSpawner` in `internal/cli/spawner.go` runs a child with inherited stdio; its requirements are
  added by install-and-update (792c7j1, 792d51).
- Qualification: responsibility, running a program and collecting its output is stated without
  naming any caller; ownership, callers choose the program and interpret its output, while this
  component owns collection, error context and not-found translation; independent verification, it
  is tested directly over a fake run primitive and over real processes; stability, the contract
  survives any change to the injection seam; necessity, the uncommitted vault-ID notice and note
  provenance depend on running `git`.
- Supported capabilities: 792a2 vault-identity, 792a3 note-writing.

No legacy document describes this component. It is derived from its realization and tests.

### 792b4b Sequences

```mermaid
sequenceDiagram
  participant Caller as Command
  participant Runner as subprocess-runner
  participant OS as Operating system
  Caller->>Runner: run program with args in dir (792b4c1)
  Runner->>OS: start the program and capture stdout and stderr
  alt program exits 0
    OS-->>Runner: output
    Runner-->>Caller: stdout and stderr, no error (792b4c1)
  else program not found
    OS-->>Runner: not-found error
    Runner-->>Caller: output so far and a command-not-found error (792b4c3)
  else program fails
    OS-->>Runner: failure
    Runner-->>Caller: output so far and an error naming program and args (792b4c2)
  end
```

### 792b4c Behavior Requirements

#### 792b4c1 Run and collect output

Requirement: 792b4c1 The subprocess-runner MUST run the named program with the given arguments in
the given directory, or in the process's working directory when none is given, and MUST return
everything the program wrote to stdout and stderr with no error when it exits successfully.

Supports: 792a2i1a, 792a2i1b, 792a2i1c, 792a3o1a, 792a3o1b, 792a3o1c, 792a3p1a, 792a3p1b.

##### 792b4c1a Scenarios

```gherkin
Scenario: 792b4c1a1 Output is collected on success
  Given a program that writes to stdout and stderr and exits 0
  When it is run
  Then its stdout and stderr are returned and there is no error

Scenario: 792b4c1a2 The program runs in the given directory
  Given a directory
  When pwd -P is run in it
  Then the output is that directory's resolved path, and the program, arguments and directory reach the run primitive unchanged
```

#### 792b4c2 Failure reporting

Requirement: 792b4c2 When the program cannot run or exits unsuccessfully, the subprocess-runner MUST
return an error that names the program and its arguments and wraps the underlying failure, along
with whatever output was captured.

Supports: 792a2i1d, 792a3o1c, 792a3p1b, 792a3p1c.

##### 792b4c2a Scenarios

```gherkin
Scenario: 792b4c2a1 A failing program is reported with its output
  Given a program that writes output and then fails
  When it is run
  Then an error naming the program and arguments is returned, and the captured output is kept
```

#### 792b4c3 Not-found translation

Requirement: 792b4c3 When the failure is the platform's "program not found" error, the
subprocess-runner MUST also make the error satisfy `errors.Is(err, update.ErrCommandNotFound)`, and
it MUST NOT do so for any other failure, including when no platform not-found error is configured.

Supports: 792a2i1d.

Rationale: callers such as the self-installer report a missing `git` or `go` differently from a
failed one, without knowing the platform's error value.

##### 792b4c3a Scenarios

```gherkin
Scenario: 792b4c3a1 A missing program is reported as not found
  Given no program by the given name is installed
  When it is run
  Then the error satisfies errors.Is for the command-not-found error and still wraps the platform error

Scenario: 792b4c3a2 Without a configured not-found error nothing is translated
  Given no platform not-found error is configured and the program fails to start
  When it is run
  Then the error wraps the failure and does not satisfy errors.Is for the command-not-found error
```

### 792b4d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a2i1a | 792b4c1 |
| 792a2i1b | 792b4c1 |
| 792a2i1c | 792b4c1 |
| 792a2i1d | 792b4c2, 792b4c3 |
| 792a3o1a | 792b4c1 |
| 792a3o1b | 792b4c1 |
| 792a3o1c | 792b4c1, 792b4c2 |
| 792a3p1a | 792b4c1 |
| 792a3p1b | 792b4c1, 792b4c2 |
| 792a3p1c | 792b4c2 |

```gherkin
Scenario: 792a2i1a Uncommitted ID is flagged
  Given a git-backed vault whose .engram-vault-id is untracked
  When the user runs engram update
  Then its report includes a notice asking for a one-time vault commit

Scenario: 792a2i1b An ignored ID is flagged
  Given a git-backed vault whose .engram-vault-id is ignored by its .gitignore
  When the user runs engram update
  Then its report includes the commit notice

Scenario: 792a2i1c A committed ID is not flagged
  Given a git-backed vault whose .engram-vault-id is committed and unmodified
  When the user runs engram update
  Then its report includes no commit notice

Scenario: 792a2i1d A vault outside git is not flagged
  Given a vault that is not a git work tree and has a .engram-vault-id
  When the user runs engram update
  Then its report includes no commit notice
```

```gherkin
Scenario: 792a3o1a The origin remote names the repo
  Given a working directory inside a git repository whose origin is git@example.com:o/r.git
  When the user runs engram learn fact
  Then the note's repo: is git@example.com:o/r.git

Scenario: 792a3o1b Without an origin the top-level directory names the repo
  Given a working directory inside a git repository named proj with no origin remote
  When the user runs engram learn fact
  Then the note's repo: is proj

Scenario: 792a3o1c Outside a git repository there is no repo
  Given a working directory outside any git repository
  When the user runs engram learn fact
  Then the note has no repo: field

Scenario: 792a3p1a The git email names the user
  Given git config user.email resolves to a@example.com
  When the user runs engram learn fact
  Then the note's user: is a@example.com

Scenario: 792a3p1b The OS username is the fallback
  Given git config user.email resolves to nothing and the OS username is joe
  When the user runs engram learn fact
  Then the note's user: is joe

Scenario: 792a3p1c No detectable user
  Given neither git config user.email nor the OS username resolves
  When the user runs engram learn fact
  Then the note has no user: field, vault: is stamped as usual, and stderr carries one warning that user detection resolved empty
```

## 792b5 Added: vault-identity-registry

Canonical target: `docs/specs/technical/vault-identity-registry.md` (title `Vault Identity Registry - Technical Spec`).

### 792b5a Orientation

- Ownership: Repository component.
- Responsibility: brings a vault into existence on first use and owns its identity: the tracked
  vault ID, the untracked `.engram/` state directory, and the location record that tells a copied,
  moved or cloned vault from the original. It stamps, validates, regenerates and claims the ID,
  checks and reports the location, refuses exchange with a parent that reports the vault's own ID,
  and detects an uncommitted ID file.
- Current realization (informational): `internal/cli/exchangestate.go` (`ensureVault`,
  `repairVaultID`, `stampVaultID`, `checkVaultLocation`, `warnVaultLocation`,
  `regenerateVaultID`, `claimVaultLocation`, `selfParentGuard`, `vaultIDUncommitted`),
  `internal/cli/vaultid.go` (`RunVaultID`), `internal/cli/vault_init.go` (`initializeVault`). The
  parent cache, declined pulls and exchange-identifier checks in `exchangestate.go` are added by
  parent-sync (792c7t1).
- Qualification: responsibility, the identity contract is stated without naming any command;
  ownership, commands decide when to exchange, while this component decides whether the vault's
  identity allows it; independent verification, every requirement is tested directly over a
  temp-dir or scripted filesystem and an injected random source; stability, the ID file, state
  directory and location record are on-disk contracts that survive refactoring; necessity,
  vault-identity's creation, ID and refusal requirements all depend on it.
- Supported capabilities: 792a2 vault-identity.

Rationale (ADR-0029 D2): every environment has its own local vault, created by the first command
that resolves its path, and a stable random vault ID in the tracked `.engram-vault-id`. Exchange
links key on that ID, never on URL or hostname, so DNS or IP churn breaks nothing. The untracked
`.engram/home.json` location record catches a `cp -R` copy or a `git clone` before it exchanges.
Transient exchange state lives in `.engram/`, which ignores itself, so the vault's tracked root
`.gitignore` is never touched. Losing `.engram/`, for example through a fresh clone, pauses
child-side exchange until `--claim` or `--regenerate` runs; links and IDs live in tracked files, so
no note content is lost.

### 792b5b Sequences

```mermaid
sequenceDiagram
  participant CLI as command-line-surface
  participant Registry as vault-identity-registry
  participant Store as durable-file-store
  CLI->>Registry: ensure the vault at path (792b5c1)
  Registry->>Store: stat the path
  alt path is missing
    Registry->>Store: write the starter files (792b5c3)
    Registry->>Store: create .engram/ and its .gitignore (792b5c6)
    Registry->>Registry: stamp the vault ID (792b5c4)
    Registry-->>CLI: ready, one creation line on stderr (792b5c1)
  else path is a directory
    alt .engram/ exists and the ID file is missing
      Registry->>Registry: stamp the vault ID (792b5c4)
      Registry-->>CLI: ready, one repair line on stderr (792b5c2)
    else any other existing vault
      Registry-->>CLI: ready, nothing written (792b5c1)
    end
  else path is not a directory, or stat fails
    Registry-->>CLI: error (792b5c1)
  end
```

```mermaid
sequenceDiagram
  participant Caller as Command
  participant Registry as vault-identity-registry
  participant Store as durable-file-store
  Caller->>Registry: stamp the vault ID (792b5c4)
  Registry->>Store: read .engram-vault-id
  alt the file exists
    Registry->>Store: re-read until it holds a valid ID (792b5c5)
    Registry-->>Caller: the existing ID
  else the file is missing
    Registry->>Registry: mint 16 random bytes as 32 lowercase hex
    Registry->>Store: create .engram-vault-id exclusively
    alt this process created it
      Registry->>Store: re-read the winning ID
      Registry->>Store: write the location record (792b5c7)
      Registry-->>Caller: the new ID
    else another process created it first
      Registry->>Store: re-read the winning ID
      Registry-->>Caller: the winner's ID, no location record written
    end
  end
```

### 792b5c Behavior Requirements

#### 792b5c1 First-use ensure

Requirement: 792b5c1 Given a vault path, the vault-identity-registry MUST, when nothing exists at
the path, write the starter files, create `.engram/`, stamp a vault ID with its location record,
and print exactly one stderr line `engram: created new vault at <path>`. It MUST leave an existing
directory untouched (except the repair in 792b5c2), MUST fail when the path exists but is not a
directory, and MUST fail when the path can't be inspected or the random source fails.

Supports: 792a2e1a, 792a2e1b, 792a2e1c, 792a2e1d, 792a2f1a.

##### 792b5c1a Scenarios

```gherkin
Scenario: 792b5c1a1 A missing vault is created with its ID and state directory
  Given nothing exists at the vault path
  When the vault is ensured with a fixed random source
  Then the starters, a 32-hex ID from that source, .engram/.gitignore holding *, and a location record of exactly the ID and the canonical path exist, and stderr has exactly one creation line

Scenario: 792b5c1a2 An existing vault without an ID is left alone
  Given an existing vault directory with no ID file and no .engram/
  When the vault is ensured
  Then nothing is created or changed and nothing is printed

Scenario: 792b5c1a3 A second ensure creates and prints nothing
  Given a vault that was just created
  When the vault is ensured again
  Then nothing is printed and the ID is unchanged

Scenario: 792b5c1a4 A regular file is not a vault
  Given a regular file at the vault path
  When the vault is ensured
  Then it fails saying the path is not a directory

Scenario: 792b5c1a5 An inspection failure is an error
  Given a vault path whose stat fails with an error other than not-exist
  When the vault is ensured
  Then that error is returned

Scenario: 792b5c1a6 A random-source failure is an error
  Given nothing exists at the vault path and a failing random source
  When the vault is ensured
  Then an error naming the random source is returned
```

#### 792b5c2 Half-created vault repair

Requirement: 792b5c2 When an existing vault has `.engram/` but no `.engram-vault-id`, the
vault-identity-registry MUST stamp the vault ID and print exactly one stderr line
`engram: stamped the missing vault ID of <path>`.

Supports: 792a2g1a.

Rationale: first-use creation writes `.engram/` before the ID, so `.engram/` marks a vault engram
created, and a stamp that failed part-way is finished by the next run.

##### 792b5c2a Scenarios

```gherkin
Scenario: 792b5c2a1 A half-created vault is repaired
  Given an existing vault with .engram/ and no ID file
  When the vault is ensured
  Then the ID file and location record are written and stderr has exactly one repair line
```

#### 792b5c3 Starter files

Requirement: 792b5c3 The vault-identity-registry MUST initialize a vault by creating `.obsidian/`
and writing `.obsidian/app.json` (`{}`), a `.gitignore` listing `.luhmann.lock`,
`.obsidian/workspace*` and `.obsidian/cache`, and the starter `README.md`, writing each only when it
is missing, and MUST return any directory or write failure.

Supports: 792a2d1a, 792a2e1a.

Rationale: the minimal `.obsidian/app.json` lets Obsidian open the directory as a vault without
engram shipping an Obsidian configuration. The `.gitignore` keeps lock and workspace files out of
git for users who track their vault.

##### 792b5c3a Scenarios

```gherkin
Scenario: 792b5c3a1 The starter set is written
  Given an empty vault directory
  When the vault is initialized
  Then .obsidian/app.json, .gitignore and README.md exist with the starter content

Scenario: 792b5c3a2 Initialization is idempotent
  Given a vault whose starter files were edited
  When the vault is initialized again
  Then no existing file is changed

Scenario: 792b5c3a3 Directory and write failures are returned
  Given a filesystem whose directory creation or file write fails
  When the vault is initialized
  Then an error naming the failed step is returned
```

#### 792b5c4 Vault ID stamping

Requirement: 792b5c4 The vault-identity-registry MUST return the vault's existing ID when
`.engram-vault-id` exists, and otherwise MUST mint 16 bytes from the injected random source as 32
lowercase hex characters, create the file exclusively, re-read it, and use whatever ID the file
holds. Only the process whose exclusive create succeeded writes the location record. An existing
vault is stamped only by `engram serve` at startup and by a command's first parent contact.

Supports: 792a2h1a, 792a2h1b, 792a2h1c.

Rationale: exclusive create followed by a re-read makes concurrent creators converge on one ID. The
random source is injected so tests are deterministic.

##### 792b5c4a Scenarios

```gherkin
Scenario: 792b5c4a1 Concurrent creators converge
  Given two processes stamping the same vault at once
  When both finish
  Then the file holds one ID and both processes returned it

Scenario: 792b5c4a2 The loser adopts the winner's ID
  Given the exclusive create reports the file already exists and its first read is empty
  When the vault ID is stamped
  Then the loser retries the read, returns the winner's ID, and writes no location record

Scenario: 792b5c4a3 The ID comes from the injected random source
  Given a fixed random source
  When a new vault ID is stamped
  Then the ID is the hex encoding of that source's bytes

Scenario: 792b5c4a4 An existing ID is reused
  Given a vault with a valid ID file
  When the vault ID is stamped
  Then the existing ID is returned and the file is unchanged
```

#### 792b5c5 Vault ID validation

Requirement: 792b5c5 The vault-identity-registry MUST accept the ID file's content, ignoring
trailing whitespace, only when it is 32 lowercase hex characters, and MUST otherwise fail with an
error saying the vault ID is invalid and naming `engram vault-id --regenerate`. A re-read after a
lost race retries an invalid read a bounded number of times before failing.

Supports: 792a2j1a, 792a2j1b, 792a2j1c.

##### 792b5c5a Scenarios

```gherkin
Scenario: 792b5c5a1 Invalid content is refused
  Given an ID file holding something other than 32 lowercase hex characters
  When the vault ID is read
  Then it fails with the invalid-vault-ID error naming engram vault-id --regenerate

Scenario: 792b5c5a2 A read failure is returned
  Given an ID file that cannot be read
  When the vault ID is read
  Then the read error is returned
```

#### 792b5c6 State directory

Requirement: 792b5c6 The vault-identity-registry MUST create `<vault>/.engram/` with a `.gitignore`
holding only `*`, writing that `.gitignore` only when it is missing, and MUST NOT write the vault's
root `.gitignore`.

Supports: 792a2k1a, 792a2k1b.

##### 792b5c6a Scenarios

```gherkin
Scenario: 792b5c6a1 The root .gitignore is untouched
  Given an existing vault with its own root .gitignore and no ID
  When its ID is stamped
  Then .engram/.gitignore holds *, the root .gitignore is unchanged, and a second stamp reuses the ID

Scenario: 792b5c6a2 State-directory failures are returned
  Given a filesystem where .engram/ or its .gitignore cannot be created
  When the state directory is ensured
  Then the failure is returned
```

#### 792b5c7 Location record and check

Requirement: 792b5c7 The vault-identity-registry MUST record a vault's location in
`.engram/home.json` as exactly its vault ID and its canonical path (symlinks evaluated, made
absolute against the working directory, cleaned), with no hostname. Checking the location MUST
report `no vault id` when there is no ID, `missing` when there is no record, `mismatch` when the
record is unparseable or names another ID or path, and `ok` otherwise.

Supports: 792a2l1a, 792a2l1b, 792a2l1c, 792a2p1a, 792a2p1b, 792a2p1c, 792a2p1d, 792a2p1e.

Rationale: a hostname would give false positives, because DHCP, `.local` renames and containers
change it without the vault moving. Canonicalizing the path keeps a symlinked route to the same
vault from failing.

##### 792b5c7a Scenarios

```gherkin
Scenario: 792b5c7a1 A copy is a mismatch
  Given a vault copied to another path
  When the copy's location is checked
  Then the result is mismatch

Scenario: 792b5c7a2 A missing record is missing
  Given a vault with an ID and no location record
  When its location is checked
  Then the result is missing

Scenario: 792b5c7a3 No ID is reported
  Given a vault with no ID file
  When its location is checked
  Then the result is no vault id

Scenario: 792b5c7a4 A record for another ID is a mismatch
  Given a location record naming a different vault ID
  When the location is checked
  Then the result is mismatch

Scenario: 792b5c7a5 A symlinked path passes
  Given a vault reached through a symlink to its recorded path
  When its location is checked
  Then the result is ok
```

#### 792b5c8 Location warning

Requirement: 792b5c8 When the location check does not pass, or cannot run, the
vault-identity-registry MUST print exactly one warning naming the vault, the check result, and both
`engram vault-id --regenerate` and `engram vault-id --claim`, and report the failure to the caller;
a passing check prints nothing.

Supports: 792a2p1a, 792a2p1b, 792a2p1c.

##### 792b5c8a Scenarios

```gherkin
Scenario: 792b5c8a1 A failed check warns once with both remedies
  Given a vault whose location check fails
  When the location warning runs
  Then exactly one warning naming both remedies is printed and the failure is reported, while a passing check prints nothing
```

#### 792b5c9 Regenerate

Requirement: 792b5c9 Regenerating MUST mint a new vault ID, atomically rewrite `.engram-vault-id`
and the location record with it, and touch no note.

Supports: 792a2m1a.

##### 792b5c9a Scenarios

```gherkin
Scenario: 792b5c9a1 A copy gets its own identity
  Given a copied vault with notes
  When its ID is regenerated
  Then the ID differs from the original's, the location check passes, and no note file changed
```

#### 792b5c10 Claim

Requirement: 792b5c10 Claiming MUST rewrite only the location record for the vault's current ID and
path, and MUST fail with an error saying the vault has no vault ID when the ID file is absent.

Supports: 792a2n1a, 792a2n1b.

##### 792b5c10a Scenarios

```gherkin
Scenario: 792b5c10a1 A moved vault keeps its ID and passes
  Given a vault moved to a new path
  When its location is claimed
  Then the ID is unchanged and the location check passes

Scenario: 792b5c10a2 Claim without an ID fails
  Given a vault with no ID file
  When its location is claimed
  Then it fails saying the vault has no vault ID
```

#### 792b5c11 vault-id report

Requirement: 792b5c11 The `vault-id` command MUST refuse `--regenerate` and `--claim` together,
print `vault_id: <id> (regenerated)` or `vault_id: <id> (location claimed)` after those actions, and
with no flag print `vault_id:` (the ID or `none`) and `location:` (the check result), adding one
remedy line naming both commands when the result is `missing` or `mismatch`, without writing
anything.

Supports: 792a2l1a, 792a2l1b, 792a2l1c, 792a2o1a.

##### 792b5c11a Scenarios

```gherkin
Scenario: 792b5c11a1 The report prints the ID and the check
  Given a vault with an ID at its recorded path
  When the vault-id report runs with no flag
  Then it prints vault_id: with the ID and location: ok, and writes nothing

Scenario: 792b5c11a2 Both flags are refused
  Given both --regenerate and --claim
  When the vault-id command runs
  Then it fails as mutually exclusive and changes nothing
```

#### 792b5c12 Self-parent guard

Requirement: 792b5c12 Given the local vault ID and a parent's reported vault ID, the
vault-identity-registry MUST report a self-parent, and print exactly one warning naming
`engram vault-id --regenerate`, exactly when the local ID is non-empty and equal to the parent's,
and MUST print nothing otherwise.

Supports: 792a2q1a.

##### 792b5c12a Scenarios

```gherkin
Scenario: 792b5c12a1 Only an equal, non-empty ID is a self-parent
  Given any local vault ID, possibly empty, and any parent vault ID
  When the self-parent guard runs
  Then it fires with one warning naming engram vault-id --regenerate exactly when the local ID is non-empty and equals the parent's
```

#### 792b5c13 Uncommitted ID detection

Requirement: 792b5c13 The vault-identity-registry MUST report the vault ID as uncommitted when
`.engram-vault-id` exists in a git work tree and is either not in the index (untracked or ignored)
or staged or modified relative to the last commit, and MUST report it as committed when the file is
absent, the vault is not a git work tree, the file is committed and unchanged, or git status fails.

Supports: 792a2i1a, 792a2i1b, 792a2i1c, 792a2i1d.

Rationale: `git status` alone hides an ignored file, so index membership is checked first.

##### 792b5c13a Scenarios

```gherkin
Scenario Outline: 792b5c13a1 Uncommitted detection
  Given a vault where <case>
  When uncommitted detection runs
  Then it reports <result>

  Examples:
    | case                                  | result      |
    | the ID file is untracked              | uncommitted |
    | the ID file is ignored                | uncommitted |
    | the ID file is staged, not committed  | uncommitted |
    | the ID file is committed              | committed   |
    | git status fails                      | committed   |
    | the vault is not a git work tree      | committed   |
    | there is no ID file                   | committed   |

Scenario: 792b5c13a2 The update report carries the notice only when flagged
  Given the uncommitted flag is set or clear
  When the update report is written
  Then it names .engram-vault-id and asks for a commit only when the flag is set
```

### 792b5d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a2d1a | 792b5c3 |
| 792a2e1a | 792b5c1, 792b5c3 |
| 792a2e1b | 792b5c1 |
| 792a2e1c | 792b5c1 |
| 792a2e1d | 792b5c1 |
| 792a2f1a | 792b5c1 |
| 792a2g1a | 792b5c2 |
| 792a2h1a | 792b5c4 |
| 792a2h1b | 792b5c4 |
| 792a2h1c | 792b5c4 |
| 792a2i1a | 792b5c13 |
| 792a2i1b | 792b5c13 |
| 792a2i1c | 792b5c13 |
| 792a2i1d | 792b5c13 |
| 792a2j1a | 792b5c5 |
| 792a2j1b | 792b5c5 |
| 792a2j1c | 792b5c5 |
| 792a2k1a | 792b5c6 |
| 792a2k1b | 792b5c6 |
| 792a2l1a | 792b5c7, 792b5c11 |
| 792a2l1b | 792b5c7, 792b5c11 |
| 792a2l1c | 792b5c7, 792b5c11 |
| 792a2m1a | 792b5c9 |
| 792a2n1a | 792b5c10 |
| 792a2n1b | 792b5c10 |
| 792a2o1a | 792b5c11 |
| 792a2p1a | 792b5c7, 792b5c8 |
| 792a2p1b | 792b5c7, 792b5c8 |
| 792a2p1c | 792b5c7, 792b5c8 |
| 792a2p1d | 792b5c7 |
| 792a2p1e | 792b5c7 |
| 792a2q1a | 792b5c12 |

```gherkin
Scenario: 792a2h1b The first parent contact stamps an existing vault
  Given an existing vault that has no .engram-vault-id and ENGRAM_PARENT set
  When the user runs engram query
  Then .engram-vault-id is created and no location warning is printed

Scenario: 792a2j1a serve starts despite a corrupt ID
  Given a vault whose .engram-vault-id is empty or holds a git conflict marker
  When engram serve starts on it
  Then it warns naming engram vault-id --regenerate, serves, and leaves the file unchanged

Scenario: 792a2j1b A merged query warns and still runs
  Given a vault whose .engram-vault-id holds junk and ENGRAM_PARENT set
  When the user runs engram query
  Then stderr says the vault ID could not be stamped and names engram vault-id --regenerate, and the local results are returned

Scenario: 792a2k1a State does not dirty the vault's git
  Given a git-backed vault
  When an offer is queued in it
  Then git status --porcelain in the vault shows no change caused by exchange state

Scenario: 792a2k1b The root .gitignore is untouched
  Given an existing vault with its own root .gitignore
  When engram stamps the vault's ID and creates .engram/
  Then the root .gitignore is byte-for-byte unchanged

Scenario: 792a2l1a A healthy vault reports its ID
  Given a vault that engram created at its current path
  When the user runs engram vault-id
  Then stdout shows its vault ID and location: ok, and stderr is empty

Scenario: 792a2l1b A moved vault reports a mismatch with both remedies
  Given a vault that was moved after engram created it
  When the user runs engram vault-id
  Then stdout shows location: mismatch and names both engram vault-id --claim and engram vault-id --regenerate

Scenario: 792a2l1c A vault without an ID is reported without writing
  Given an existing vault with no vault ID
  When the user runs engram vault-id
  Then stdout shows location: no vault id, and no file in the vault is created or changed

Scenario: 792a2p1a A cloned child is caught before it offers
  Given vault A is cloned with git to a new host with ENGRAM_PARENT set
  When the clone runs engram learn
  Then the note is written locally and queued, no offer is sent, and the warning names both engram vault-id remedies

Scenario: 792a2p1b A cloned child's merged query still runs
  Given a vault with ENGRAM_PARENT set whose location record is missing
  When the user runs engram query
  Then the query returns its results, and stderr carries one warning naming both engram vault-id remedies

Scenario: 792a2p1c serve starts despite a mismatched record
  Given a vault whose location record is missing or differs
  When engram serve starts on it
  Then it prints the warning and serves normally

Scenario: 792a2p1d A symlinked path is not a false positive
  Given a vault reached through a symlink to its recorded path
  When the location check runs
  Then the check passes

Scenario: 792a2p1e A hostname change is not a false positive
  Given a host whose name changed while the vault path did not
  When the location check runs
  Then the check passes

Scenario: 792a2q1a A copied vault served back to itself
  Given a copy of vault V is served and V's environment points ENGRAM_PARENT at it
  When the user runs engram query in V
  Then it returns local-only results with a warning naming engram vault-id --regenerate, and no offer is sent
```

## 792b6 Added: note-document-format

Canonical target: `docs/specs/technical/note-document-format.md` (title `Note Document Format - Technical Spec`).

### 792b6a Orientation

- Ownership: Repository component, crossing the YAML library boundary.
- Responsibility: the on-disk shape of a vault note. It names the note's file, renders its
  frontmatter block with each kind's keys in a fixed order, quotes the scalars that must stay
  strings, renders the body formulas and the machine-written link lines, splits a note's text
  into frontmatter and body, reads a CRLF note as its LF form, and edits an existing note's
  frontmatter in place.
- Current realization (informational): in `internal/cli/learn.go`, `learnPath`,
  `marshalFrontmatter`, `quotedString`, the `factFrontmatterDoc`, `feedbackFrontmatterDoc` and
  `runbookFrontmatterDoc` types and their `render*` functions; in `internal/cli/qa.go`,
  `qaQuestionPath`, `qaAnswerPath`, `renderQAQuestionNote` and `renderQAAnswerNote`;
  `renderSupersedes` in `internal/cli/supersedes.go`; the exchange fields in
  `internal/cli/exchangefields.go`; `SplitFrontmatter` in `internal/embed/hash.go`; the YAML-node
  edits in `internal/cli/frontmatter_node.go`; `toLF` in `internal/cli/luhmann_reparent.go`.
- Qualification: responsibility, the file and document shape is stated without naming any writer or
  reader; ownership, writers decide what a note says and when it is written, while this component
  decides how it is laid out; independent verification, rendering, naming and splitting are tested
  directly on strings; stability, the shape survives any change to the commands that write it, and
  hand-edited and Obsidian-edited notes depend on it; necessity, every note engram writes or reads
  passes through it.
- Supported capabilities: 792a3 note-writing, 792a4 note-revision. Vault-counting,
  skill-runbook-registration and parent-sync add their requirements here (792d30, 792d61).

Rationale: the vault is flat. Notes sit at the vault root, and a note's place in the hierarchy is
carried by the Luhmann ID that leads its filename, not by a folder (792a2c). A fact or feedback note
states its situation twice: in the `situation:` field, which recall's situation vector is built
from, and again in the body formula. A legacy invariant asked for a single representation of any
datum that affects retrieval; the two copies are instead kept in step by the writers that change a
situation (792d62).

### 792b6b Sequences

The note-document-format owns no flow. Note-authoring calls it to name and render a new note, and
every reader of a note calls it to split the note's text.

### 792b6c Behavior Requirements

#### 792b6c1 Note filenames

Requirement: 792b6c1 The note-document-format MUST name a fact, feedback or runbook note
`<luhmann>.<YYYY-MM-DD>.<slug>.md` in the vault root, using its creation date, and MUST name a
question-and-answer pair `qa.<YYYY-MM-DD>.<slug>.q.md` and `qa.<YYYY-MM-DD>.<slug>.a.md`.

Supports: 792a3a1a, 792a3b1a, 792a3c1a, 792a3q1a.

##### 792b6c1a Scenarios

```gherkin
Scenario: 792b6c1a1 A Luhmann note's filename
  Given vault /vault, Luhmann ID 1a3, creation date 2026-05-09 and slug subagent-driven-recovery
  When the note's path is formed
  Then it is /vault/1a3.2026-05-09.subagent-driven-recovery.md

Scenario: 792b6c1a2 A pair's filenames
  Given vault /vault, creation date 2026-07-03 and slug my-slug
  When the pair's paths are formed
  Then they are /vault/qa.2026-07-03.my-slug.q.md and /vault/qa.2026-07-03.my-slug.a.md
```

#### 792b6c2 Frontmatter block

Requirement: 792b6c2 The note-document-format MUST render a note's frontmatter as a line `---`, the
fields as YAML, a line `---` and one blank line, and MUST encode every value so that it reads back
unchanged, whatever newlines, colons, leading dashes or other YAML-significant characters it holds.

Supports: 792a3a1a, 792a3b1a, 792a3c1a, 792a3q1a.

Rationale: frontmatter was once built by concatenating strings, and a multi-line value ended the
block early. Encoding through the YAML library fixed that for every value.

##### 792b6c2a Scenarios

```gherkin
Scenario: 792b6c2a1 The block is delimited
  Given the fields k: v
  When they are rendered as frontmatter
  Then the result is "---", "k: v", "---" and a blank line

Scenario: 792b6c2a2 Tricky values read back unchanged
  Given field values holding a newline, a colon, a leading dash and a leading asterisk, or any printable text
  When a fact or feedback frontmatter is rendered and parsed again
  Then every field reads back exactly as given
```

#### 792b6c3 Kind schemas

Requirement: 792b6c3 The note-document-format MUST render each kind's keys in this order, leaving
out any optional key whose value is empty:

- fact: `type`, `tier`, `situation`, `subject`, `predicate`, `object`, `luhmann`, `created`,
  `source`, `project`, `repo`, `user`, `vault`, `pending`, `issue`, `sources`, `vocab_version`,
  `tags`, `supersedes`, then the exchange fields;
- feedback: `type`, `tier`, `situation`, `behavior`, `impact`, `action`, `luhmann`, `created`,
  `source`, `project`, `repo`, `user`, `vault`, `pending`, `issue`, `sources`, `tags`,
  `supersedes`, then the exchange fields;
- runbook: `type`, `tier`, `situation`, `done_when`, `red_flags`, `triggers`, `luhmann`, `created`,
  `source`, `project`, `repo`, `user`, `vault`, `skill_hash`, `skill_key`, `skill_source`,
  `pending`, `issue`, `sources`, `tags`, `supersedes`, then the exchange fields.

`type`, `luhmann`, `created` and `source` are always written, and so are a fact's or feedback note's
`situation` and content fields. `luhmann` and `issue` MUST be double-quoted, `created` MUST be the
date as `YYYY-MM-DD`, and list values MUST keep their order.

Supports: 792a3a1a, 792a3b1a, 792a3c1a, 792a3d1a, 792a3d1b, 792a3d1c, 792a3h1a, 792a3h1b, 792a3i1a,
792a3i1b, 792a3o1c, 792a3p1c, 792a4b1c.

Rationale: a Luhmann ID is quoted so that an ID such as `1e1` is not read back as the number 10, and
so that notes written by engram, migrated and written by hand all read the same way. A fact or
feedback note always carries `situation:`, because recall finds a note by its situation.

##### 792b6c3a Scenarios

```gherkin
Scenario: 792b6c3a1 A fact's frontmatter
  Given a fact with situation, subject, predicate, object, Luhmann ID 11, creation date 2026-05-09 and a source
  When its frontmatter is rendered
  Then it has type fact, the given fields, luhmann 11 and created 2026-05-09

Scenario: 792b6c3a2 A feedback note's frontmatter holds exactly its keys
  Given a feedback note with situation, behavior, impact, action, Luhmann ID 9z, creation date 2026-05-09, a source and nothing else
  When its frontmatter is rendered and parsed
  Then it holds exactly type, situation, behavior, impact, action, luhmann, created and source

Scenario: 792b6c3a3 A runbook's frontmatter holds exactly its keys
  Given a runbook with situation, done_when, Luhmann ID 7a, creation date 2026-05-09, a source and nothing else
  When its frontmatter is rendered and parsed
  Then it holds exactly type, situation, done_when, luhmann, created and source

Scenario: 792b6c3a4 The Luhmann ID is double-quoted
  Given a feedback note with Luhmann ID 9aa
  When its frontmatter is rendered
  Then it contains luhmann: "9aa"

Scenario: 792b6c3a5 Project and issue follow the source
  Given a fact or feedback note with project engram and issue 636
  When its frontmatter is rendered
  Then project: engram and issue: "636" follow source:, in that order

Scenario: 792b6c3a6 Empty optional keys are left out
  Given a fact, feedback or runbook note with no project, issue or red flags
  When its frontmatter is rendered
  Then it has no project:, issue: or red_flags: key

Scenario: 792b6c3a7 Lists keep their order
  Given a runbook with red flags A then B, or a fact with any list of valid tags
  When its frontmatter is rendered and parsed
  Then the list reads back with the same entries in the same order
```

#### 792b6c4 Body formulas

Requirement: 792b6c4 The note-document-format MUST render a fact's body as
`Information learned: when in <situation>, <subject> <predicate> <object>.`, a feedback note's as
`Lesson learned: when <situation>, <action>.`, dropping one case-insensitive leading `when ` from the
situation, and a runbook's as its steps with trailing newlines reduced to one, each followed by a
blank line and then one line `Supersedes: [[<note>]] — <type>: <claim>` per supersession, with any
`.md` dropped from the linked note.

Supports: 792a3a1a, 792a3a1b, 792a3b1a, 792a3b1b, 792a3c1a, 792a3h1a, 792a4b1a, 792a4b1b, 792a4d1a,
792a4d1b, 792a4l1a.

Rationale: situations are usually phrased "When …", while the formulas already say "when"; without
the strip, a body would read "when When". A supersession link drops `.md` because links resolve by
extension-less basename.

##### 792b6c4a Scenarios

```gherkin
Scenario: 792b6c4a1 A fact's body
  Given a fact whose situation is "reasoning about agent coordination", subject "subagent dispatch", predicate "is fundamentally" and object "a verification problem dressed as coordination"
  When its body is rendered
  Then it is "Information learned: when in reasoning about agent coordination, subagent dispatch is fundamentally a verification problem dressed as coordination." and a blank line

Scenario: 792b6c4a2 A feedback note's body
  Given a feedback note whose situation is "orchestrating multi-step work" and action "shrink the task"
  When its body is rendered
  Then it is "Lesson learned: when orchestrating multi-step work, shrink the task." and a blank line

Scenario: 792b6c4a3 A runbook's body is its steps
  Given a runbook whose steps are three numbered lines
  When its body is rendered
  Then it begins with those three lines unchanged

Scenario: 792b6c4a4 A leading When is dropped
  Given a fact or feedback situation beginning "When "
  When its body is rendered
  Then the body reads "when <rest of situation>" and never "when When"

Scenario: 792b6c4a5 A supersession line drops .md
  Given a supersession of 9e.2026-01-01.old-note.md of type updates with claim "old note superseded"
  When its body line is rendered
  Then it is "Supersedes: [[9e.2026-01-01.old-note]] — updates: old note superseded"
```

#### 792b6c5 Question-and-answer documents

Requirement: 792b6c5 The note-document-format MUST render a question note with frontmatter `type:
qa-question`, `date`, `answered_by` (the answer's basename) and `source`, and a body of the question
followed by a blank line and `Answered by: [[<answer basename>]]`; and an answer note with
frontmatter `type: qa-answer`, `date`, `answers` (the question's basename), `certainty`,
`contributors` when there are any, and `source`, and a body of the answer followed by a blank line,
`Answers: [[<question basename>]]` and, when there are contributors, `Contributors:` and a
comma-separated link to each.

Supports: 792a3q1a, 792a3q1b, 792a3q1c.

##### 792b6c5a Scenarios

```gherkin
Scenario: 792b6c5a1 A question note
  Given the question "What is the question?", slug my-slug and date 2026-07-03
  When the question note is rendered
  Then its frontmatter has type: qa-question and answered_by: qa.2026-07-03.my-slug.a, and its body holds the question and "Answered by: [[qa.2026-07-03.my-slug.a]]"

Scenario: 792b6c5a2 An answer note with contributors
  Given an answer, slug my-slug, date 2026-07-03, certainty medium and two contributors
  When the answer note is rendered
  Then its frontmatter has type: qa-answer, answers: qa.2026-07-03.my-slug.q and certainty: medium, and its body holds the answer, "Answers: [[qa.2026-07-03.my-slug.q]]" and "Contributors: [[<first>]], [[<second>]]"

Scenario: 792b6c5a3 An answer note without contributors
  Given an answer with no contributors
  When the answer note is rendered
  Then it has no Contributors: line
```

#### 792b6c6 Frontmatter splitting

Requirement: 792b6c6 The note-document-format MUST split a note's text into the frontmatter between
a leading `---` line and the next line that is exactly `---`, and the text after that line. A value
that ends in `---`, or an indented `---` inside a block value, MUST NOT close the frontmatter, an
empty frontmatter MUST split cleanly, and a text that has no leading `---` line or no closing line
MUST be treated as having no frontmatter.

Supports: 792a3n1a, 792a3u1a, 792a3u1b.

Rationale: a skill key, for example, can end in `---`; matching only a whole line keeps such a value
from cutting the frontmatter short.

##### 792b6c6a Scenarios

```gherkin
Scenario: 792b6c6a1 Only a whole --- line closes the frontmatter
  Given frontmatter holding a value that ends in ---, or a block value with an indented --- line, followed by a body that may itself start with ---
  When the note is split
  Then the frontmatter runs to the first line that is exactly ---, and the body is everything after it

Scenario: 792b6c6a2 An empty frontmatter splits cleanly
  Given a note whose --- line is immediately followed by the closing --- line
  When the note is split
  Then the frontmatter is empty and the body follows

Scenario: 792b6c6a3 A note without frontmatter
  Given a note with no leading --- line, or with no closing --- line
  When the note is split
  Then it has no frontmatter, and its whole text is the body
```

#### 792b6c7 Frontmatter node edits

Requirement: 792b6c7 The note-document-format MUST write an edit to an existing note's frontmatter
as edits to its parsed YAML mapping: a key whose value the edit changes MUST be replaced in place, a
key the edit adds MUST be inserted after the nearest key that precedes it in its kind's order
(before the nearest following one when none precedes, at the end when neither is present), a key
the edit clears MUST be deleted, and every other key, including keys no kind defines, MUST be left
exactly as parsed, with its anchors and style. A replaced `supersedes:` list MUST keep, for each new
entry whose `note:` names the same note as an old entry (compared without a trailing `.md`), the
old entry's keys that no kind defines. It MUST refuse the edit, naming the key, when a key it would
set or delete, its value, or anything beneath the value carries a YAML anchor; MUST refuse a
frontmatter block that is not a mapping; and MUST refuse a result whose frontmatter does not decode.
For a note whose frontmatter holds only keys its kind defines and no anchors, the edited text MUST
be byte-identical to the note rendered afresh with the edit.

Supports: 792a4j1a, 792a4j1b, 792a4j1c, 792a4j1d, 792a4k1a, 792a4k1b.

Rationale: re-rendering the whole frontmatter from the typed note model dropped every key the model
does not define (#780); editing the parsed mapping keeps them. An anchored value cannot be replaced
without leaving its aliases dangling, and decoding the result again before it is written keeps a
note that no reader could parse from ever reaching the disk. Byte-identical output for ordinary
notes keeps their exchange hash unchanged, so the offer paths that compare these bytes behave as
they did before the node edits.

##### 792b6c7a Scenarios

```gherkin
Scenario: 792b6c7a1 Keys the edit does not change are kept
  Given a note's frontmatter carrying keys no kind defines, a nested unknown map, and an anchor on a key the edit does not change
  When an edit replaces one key its kind defines
  Then every other key keeps its value and its anchor

Scenario: 792b6c7a2 A set key keeps or takes its kind's position
  Given the key order a, b, c, d
  When b is set in "a: 1\nb: 2\n", in "a: 1\nd: 4\n", in "x: 0\nd: 4\n", and c is set in "x: 0\n"
  Then the results are "a: 1\nb: new\n", "a: 1\nb: new\nd: 4\n", "x: 0\nb: new\nd: 4\n" and "x: 0\nc: new\n"

Scenario: 792b6c7a3 A kept supersession entry keeps its unknown keys
  Given a supersedes: list naming A with an extra key x_reason: kept, and naming B
  When the list is replaced by entries naming A and C
  Then the A entry carries the new type and claim and x_reason: kept, B is gone, and C carries only note, type and claim

Scenario: 792b6c7a4 An anchored edited key is refused
  Given frontmatter whose key to be replaced carries an anchor that another key aliases
  When the edit is made
  Then it fails naming the key, and nothing is written

Scenario: 792b6c7a5 A result that does not decode is refused
  Given rendered note text whose frontmatter holds a dangling alias, is not a mapping, or is missing
  When it is checked before a write
  Then it is refused as undecodable; valid frontmatter, an anchor with its alias, and an empty block pass

Scenario: 792b6c7a6 A frontmatter block must be a mapping
  Given a frontmatter block
  When it is parsed for editing
  Then a mapping is returned as one, an empty block is an empty mapping, a list is refused as not a mapping, and malformed YAML is refused

Scenario: 792b6c7a7 An ordinary note is written as a fresh render
  Given a fact, feedback or runbook note holding only keys its kind defines and no anchors
  When any amend kind edits it
  Then the result is byte-identical to the note rendered afresh with the edit, and its exchange hash is unchanged
```

#### 792b6c8 CRLF reading

Requirement: 792b6c8 The note-document-format MUST read a note for splitting and editing as its LF
form, converting each `\r\n` to `\n` and keeping a lone `\r`, while frontmatter splitting itself
stays LF-only (792b6c6).

Supports: 792a4m1a, 792a4m1b.

Rationale: a CRLF-aware split would hand the writers that splice its slices back into a note CRLF
lines, and they would write mixed line endings; converting at the reader leaves every writer working
on the LF form, and a writer converts a note only inside a write it makes anyway.

##### 792b6c8a Scenarios

```gherkin
Scenario: 792b6c8a1 CRLF pairs become LF
  Given the text "a\r\nb\rc\r\n\r\n"
  When it is read as LF
  Then it is "a\nb\rc\n\n", and text without CRLF is unchanged
```

### 792b6d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a3a1a | 792b6c1, 792b6c2, 792b6c3, 792b6c4 |
| 792a3a1b | 792b6c4 |
| 792a3b1a | 792b6c1, 792b6c2, 792b6c3, 792b6c4 |
| 792a3b1b | 792b6c4 |
| 792a3c1a | 792b6c1, 792b6c2, 792b6c3, 792b6c4 |
| 792a3d1a | 792b6c3 |
| 792a3d1b | 792b6c3 |
| 792a3d1c | 792b6c3 |
| 792a3h1a | 792b6c3, 792b6c4 |
| 792a3h1b | 792b6c3 |
| 792a3i1a | 792b6c3 |
| 792a3i1b | 792b6c3 |
| 792a3n1a | 792b6c6 |
| 792a3o1c | 792b6c3 |
| 792a3p1c | 792b6c3 |
| 792a3q1a | 792b6c1, 792b6c2, 792b6c5 |
| 792a3q1b | 792b6c5 |
| 792a3q1c | 792b6c5 |
| 792a3u1a | 792b6c6 |
| 792a3u1b | 792b6c6 |
| 792a4b1a | 792b6c4 |
| 792a4b1b | 792b6c4 |
| 792a4b1c | 792b6c3 |
| 792a4d1a | 792b6c4 |
| 792a4d1b | 792b6c4 |
| 792a4j1a | 792b6c7 |
| 792a4j1b | 792b6c7 |
| 792a4j1c | 792b6c7 |
| 792a4j1d | 792b6c7 |
| 792a4k1a | 792b6c7 |
| 792a4k1b | 792b6c7 |
| 792a4l1a | 792b6c4 |
| 792a4m1a | 792b6c8 |
| 792a4m1b | 792b6c8 |

```gherkin
Scenario: 792a3a1b A leading When is not doubled
  Given an empty vault
  When the user runs engram learn fact with --situation "When testing" --subject s --predicate p --object o
  Then the note's body reads "Information learned: when in testing, s p o."

Scenario: 792a3b1b A leading When is not doubled
  Given an empty vault
  When the user runs engram learn feedback with --situation "When x" --action a
  Then the note's body reads "Lesson learned: when x, a."

Scenario: 792a3d1a Red flags are kept in order
  Given an empty vault
  When the user runs engram learn runbook with --red-flag "A" --red-flag "B"
  Then the note's red_flags: lists A then B

Scenario: 792a3d1b Triggers follow the red flags in order
  Given an empty vault
  When the user runs engram learn runbook with --red-flag "A" --trigger "cue one" --trigger "cue two"
  Then the note's triggers: lists "cue one" then "cue two", after red_flags:

Scenario: 792a3d1c Neither flag means neither field
  Given an empty vault
  When the user runs engram learn runbook with no --red-flag and no --trigger
  Then the note is written with no red_flags: and no triggers: field

Scenario: 792a3h1a Optional metadata is written
  Given an empty vault
  When the user runs engram learn runbook with --project proj --issue 12 --tag fam/val --chunk-source "a#b" --supersedes "1.x|updates|claim"
  Then the note carries project: proj, issue: "12", tags: [fam/val], sources: [a#b], a supersedes: entry for 1.x of type updates, and the body line "Supersedes: [[1.x]] — updates: claim"

Scenario: 792a3h1b Absent metadata writes no field
  Given an empty vault
  When the user runs engram learn fact with none of those flags
  Then the note has no project:, issue:, tags:, sources: or supersedes: field

Scenario: 792a3i1a The tier defaults to L2
  Given an empty vault
  When the user runs engram learn fact or feedback without --tier
  Then the note carries tier: L2

Scenario: 792a3i1b A legacy tier is written as given
  Given an empty vault
  When the user runs engram learn fact with --tier L3
  Then the note carries tier: L3

Scenario: 792a3q1b A pair without contributors
  Given an existing vault
  When the user runs engram learn qa without --contributors
  Then the answer note has no contributors: field and no Contributors: line

Scenario: 792a3q1c The answer can come from a file
  Given a file holding the answer text
  When the user runs engram learn qa with --answer-file naming it
  Then the answer note's body is that file's text
```

```gherkin
Scenario: 792a4b1b A feedback field is replaced
  Given a feedback note whose situation is x and action a
  When the user runs engram amend --target <note> --action b
  Then the note's action: is b and its body reads "Lesson learned: when x, b."

Scenario: 792a4b1c A runbook's steps and red flags are replaced
  Given a runbook note with body "1. old" and red flags A and B
  When the user runs engram amend --target <note> --body "1. new" --red-flag C
  Then the note's body is "1. new" and its red_flags: lists only C

Scenario: 792a4d1a A supersession is written in canonical form
  Given a vault holding note 107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior
  When the user runs engram amend --target <note> --supersedes "107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior.md|refutes|claim"
  Then the body line is "Supersedes: [[107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior]] — refutes: claim", and the supersedes: entry's note: is 107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior.md

Scenario: 792a4d1b The list is replaced, not appended to
  Given a note whose supersedes: names A and B, with a Supersedes: line for each
  When the user runs engram amend --target <note> --supersedes "C|narrows|c"
  Then the note's supersedes: names only C, and its body has one Supersedes: line, for C

Scenario: 792a4j1a Amend keeps an unmodeled key
  Given a fact, feedback or runbook note whose frontmatter carries luhmann_old: "12" and a nested map that engram does not define
  When the user runs engram amend --target <note> with a content flag, --supersedes or --chunk-source
  Then the written note still carries luhmann_old: "12" and the nested map with the same values

Scenario: 792a4j1b Amend keeps an anchor on a key it does not edit
  Given a fact note whose source: &src test is aliased by another key
  When the user runs engram amend --target <note> --object new
  Then the amend succeeds, and source: and the aliasing key both still read test

Scenario: 792a4j1c Amend keeps the extra keys of a kept supersession
  Given a note whose supersedes: names A with an extra key x_reason: kept, and names B
  When the user runs engram amend --target <note> --supersedes "A|refutes|new" --supersedes "C|narrows|c"
  Then the A entry carries type: refutes, claim: new and x_reason: kept; B is gone; and the C entry carries only note, type and claim

Scenario: 792a4j1d Resituate keeps every untouched field
  Given a fact note carrying tags:, sources:, supersedes:, pending: true, parent:, aliases: and luhmann_old: "12"
  When the user runs engram resituate --note <note> --situation new
  Then all of those are unchanged, and only situation: and the body's first line differ

Scenario: 792a4m1b Resituate converts a CRLF note
  Given a fact or feedback note whose lines end in \r\n
  When the user runs engram resituate --note <note> --situation new
  Then the rewrite succeeds, the written note contains no \r, and only situation: and the body's first line changed
```

## 792b7 Added: luhmann-addressing

Canonical target: `docs/specs/technical/luhmann-addressing.md` (title `Luhmann Addressing - Technical Spec`).

### 792b7a Orientation

- Ownership: Repository component.
- Responsibility: the Luhmann ID scheme. It parses an ID into its alternating number and letter
  segments, reads the ID that leads a note's filename, orders letter segments, and allocates the
  next unused ID for a new note from the existing IDs, a target and a position.
- Current realization (informational): `ParseID`, `FromBasename` and `LetterLess` in
  `internal/luhmann/luhmann.go`; `nextLuhmannID` and its helpers in `internal/cli/luhmann.go`;
  `extractLuhmannFromFilename` in `internal/cli/learn.go`. The parse and order functions are a
  shared kernel: note-authoring, the vault scanner, Luhmann reparenting and the exchange code all
  use them.
- Qualification: responsibility, the scheme is stated without naming any caller; ownership, callers
  choose the target and position, while this component decides what ID they yield; independent
  verification, parsing, ordering and allocation are pure functions tested directly; stability, the
  scheme is the vault's addressing convention and outlives any refactoring; necessity, every fact,
  feedback and runbook note is placed by it.
- Supported capabilities: 792a3 note-writing. Vault-migration adds reparenting's needs (792d30).

Rationale: Luhmann IDs must be unique across the vault and well formed, a digit-led run of digits
and letters. The allocator only ever returns an ID that no existing note holds; two concurrent
writers are kept from allocating the same one by the vault-write-lock.

### 792b7b Sequences

The luhmann-addressing owns no flow. Note-authoring calls it, under the vault write lock, with the
vault's existing IDs.

### 792b7c Behavior Requirements

#### 792b7c1 ID parsing

Requirement: 792b7c1 The luhmann-addressing MUST split an ID into its alternating runs of digits and
letters, MUST refuse an empty ID, and MUST refuse an ID that does not start with a digit.

Supports: 792a3j1a, 792a3j1b, 792a3j1c, 792a3j1d, 792a3j1e, 792a3k1d.

##### 792b7c1a Scenarios

```gherkin
Scenario: 792b7c1a1 Segments alternate
  Given the IDs 1a3b and 12ab3
  When they are parsed
  Then they split into 1, a, 3, b and into 12, ab, 3

Scenario: 792b7c1a2 A top-level ID is one segment
  Given the ID 1
  When it is parsed
  Then it is the single segment 1

Scenario: 792b7c1a3 An empty ID is refused
  Given an empty ID
  When it is parsed
  Then the empty-ID error is returned

Scenario: 792b7c1a4 A leading letter is refused
  Given the ID a1
  When it is parsed
  Then the leading-letter error is returned
```

#### 792b7c2 ID from a filename

Requirement: 792b7c2 The luhmann-addressing MUST read a note's ID as the part of its `.md`
filename before the first dot, when that part starts with a digit and holds only digits and
letters, and MUST report no ID for any other filename.

Supports: 792a3j1a, 792a3j1b, 792a3j1c, 792a3j1d, 792a3j1e.

##### 792b7c2a Scenarios

```gherkin
Scenario: 792b7c2a1 The leading ID is read
  Given the filenames 1a3.2026-05-09.subagent-recovery.md and 9o1.2026-05-10.slug
  When their IDs are read
  Then they are 1a3 and 9o1

Scenario: 792b7c2a2 A filename without a leading ID has none
  Given README.md, a name starting with a dot, a name with no dot, or a name not starting with a digit
  When its ID is read
  Then no ID is reported

Scenario: 792b7c2a3 A file that is not Markdown has no ID
  Given 1a3.2026-05-09.subagent-recovery.txt
  When its ID is read
  Then no ID is reported
```

#### 792b7c3 Letter order

Requirement: 792b7c3 The luhmann-addressing MUST order letter segments shorter first, then
alphabetically, so that `z` comes before `aa`, `az` before `ba`, and `zz` before `aaa`.

Supports: 792a3j1e.

Rationale: an allocator once compared letter segments lexically, so with children `a` to `z`, `aa`
and `ab`, it took `z` as the largest and allocated `aa` again.

##### 792b7c3a Scenarios

```gherkin
Scenario: 792b7c3a1 Shorter letter segments come first
  Given the letter segments z, aa, ab, az, ba, zz and aaa
  When they are compared
  Then z < aa < ab, az < ba, zz < aaa, and no segment is less than itself
```

#### 792b7c4 Next-ID allocation

Requirement: 792b7c4 Given the existing IDs, the luhmann-addressing MUST allocate, for `top`, one
more than the largest top-level number, or `1` when there is none; for `continuation`, the next
child of the target, counting only the target's direct children: after an ID ending in a number,
the letter after its largest letter child in Luhmann order, or `a`, and after an ID ending in a
letter, the number after its largest numeric child, or `1`; and for `sibling`, the next child of
the target's parent.

Supports: 792a3j1a, 792a3j1b, 792a3j1c, 792a3j1d, 792a3j1e.

##### 792b7c4a Scenarios

```gherkin
Scenario: 792b7c4a1 Top takes the next top-level number
  Given the IDs 1, 1a, 2 and 2a, or no IDs at all
  When an ID is allocated at top
  Then it is 3, or 1 for no IDs

Scenario: 792b7c4a2 A continuation of a number takes the next letter
  Given the IDs 1 and 2, the IDs 1, 1a and 1b, or the IDs 1, 2, 2a and 1a3
  When an ID is allocated as a continuation of 1
  Then it is 1a, 1c and 1a respectively, ignoring IDs that are not direct children of 1

Scenario: 792b7c4a3 A continuation of a letter takes the next number
  Given the IDs 1 and 1a, or the IDs 1, 1a, 1a1 and 1a2
  When an ID is allocated as a continuation of 1a
  Then it is 1a1 or 1a3 respectively

Scenario: 792b7c4a4 Letters roll over
  Given the IDs 1 and 1a through 1z, optionally with 1aa and 1ab
  When an ID is allocated as a continuation of 1
  Then it is 1aa, or 1ac when 1aa and 1ab exist

Scenario: 792b7c4a5 A sibling takes the next child of the parent
  Given the IDs 1 and 1a
  When an ID is allocated as a sibling of 1a
  Then it is 1b
```

#### 792b7c5 Allocation refusals

Requirement: 792b7c5 The luhmann-addressing MUST refuse to allocate for a position other than `top`,
`continuation` or `sibling`, for a `continuation` or `sibling` with no target or with a target that
is not a valid ID, and for a `sibling` of a top-level ID.

Supports: 792a3k1a, 792a3k1b, 792a3k1c, 792a3k1d.

##### 792b7c5a Scenarios

```gherkin
Scenario: 792b7c5a1 An unknown position is refused
  Given the position bogus
  When an ID is allocated
  Then an error is returned

Scenario: 792b7c5a2 A missing target is refused
  Given a continuation or a sibling with no target
  When an ID is allocated
  Then an error is returned

Scenario: 792b7c5a3 A malformed target is refused
  Given a continuation or a sibling whose target is abc
  When an ID is allocated
  Then an error is returned

Scenario: 792b7c5a4 A sibling of a top-level ID is refused
  Given the IDs 1 and 2
  When an ID is allocated as a sibling of 1
  Then an error is returned
```

### 792b7d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a3j1a | 792b7c1, 792b7c2, 792b7c4 |
| 792a3j1b | 792b7c1, 792b7c2, 792b7c4 |
| 792a3j1c | 792b7c1, 792b7c2, 792b7c4 |
| 792a3j1d | 792b7c1, 792b7c2, 792b7c4 |
| 792a3j1e | 792b7c1, 792b7c2, 792b7c3, 792b7c4 |
| 792a3k1a | 792b7c5 |
| 792a3k1b | 792b7c5 |
| 792a3k1c | 792b7c5 |
| 792a3k1d | 792b7c1, 792b7c5 |

```gherkin
Scenario: 792a3j1a Top takes the next top-level number
  Given a vault whose notes have IDs 1, 2 and 2a
  When the user runs engram learn fact --position top
  Then the new note's ID is 3

Scenario: 792a3j1b A continuation of a number takes the next letter
  Given a vault whose notes have IDs 1 and 1a
  When the user runs engram learn fact --position continuation --target 1
  Then the new note's ID is 1b

Scenario: 792a3j1c A continuation of a letter takes the next number
  Given a vault whose note has ID 1a
  When the user runs engram learn runbook --position continuation --target 1a
  Then the new note's ID is 1a1

Scenario: 792a3j1d A sibling takes the next child of the target's parent
  Given a vault whose notes have IDs 1, 1a and 1a1
  When the user runs engram learn feedback --position sibling --target 1a1
  Then the new note's ID is 1a2

Scenario: 792a3j1e Letters roll over from z to aa
  Given a vault whose notes have IDs 1 and 1a through 1z
  When the user runs engram learn fact --position continuation --target 1
  Then the new note's ID is 1aa
```

## 792b8 Added: vault-write-lock

Canonical target: `docs/specs/technical/vault-write-lock.md` (title `Vault Write Lock - Technical Spec`).

### 792b8a Orientation

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
  requirement (792d63).
- Qualification: responsibility, exclusive hold of a lock file is stated without naming any writer;
  ownership, callers decide what to do under the lock and when to take it, while this component
  owns acquiring, waiting and releasing; independent verification, it is tested directly over fake
  primitives and over a real lock file; stability, the contract survives any change to the
  injection seam; necessity, concurrent note writes depend on it.
- Supported capabilities: 792a3 note-writing, 792a4 note-revision. Chunk-indexing and
  controlled-vocabulary add their callers' requirements (792d30, 792d63).

Rationale: before every read-modify-write writer took a lock, only `learn`'s Luhmann-ID allocation
was locked, and two concurrent `ingest`, `amend`, `resituate` or `activate` runs corrupted the vault
or the chunk index in production; many parallel sub-recalls make that common. Every such writer
now takes the lock at its command's entry point: `.luhmann.lock` for notes and sidecars, and
`.manifest.lock` for the chunk index. The helpers they share never take a lock themselves, so a
command never waits on a lock it already holds. That rule is a convention, not a checked
invariant. The lock works on a raw file descriptor rather than a Go file object, because a dropped
file object's finalizer would close the descriptor and silently release the lock while it was still
meant to be held.

### 792b8b Sequences

```mermaid
sequenceDiagram
  participant Caller as Writer command
  participant Lock as vault-write-lock
  participant OS as Operating system
  Caller->>Lock: lock the vault (792b8c4)
  Lock->>OS: open or create <vault>/.luhmann.lock with mode 0600 (792b8c1)
  alt open fails
    OS-->>Lock: error
    Lock-->>Caller: error naming the lock path, no lock taken (792b8c3)
  else opened
    Lock->>OS: take an exclusive lock, waiting while another holder has it (792b8c1)
    alt lock fails
      Lock->>OS: close the file
      Lock-->>Caller: error naming the lock path (792b8c3)
    else lock held
      Lock-->>Caller: release function (792b8c1)
      Caller->>Lock: release (792b8c2)
      Lock->>OS: unlock, then close the file
      Lock-->>Caller: unlock error, else close error, else success (792b8c2)
    end
  end
```

### 792b8c Behavior Requirements

#### 792b8c1 Exclusive hold

Requirement: 792b8c1 The vault-write-lock MUST open the lock file at the given path, creating it
with mode `0600` when absent, MUST take an exclusive advisory lock on it, waiting while another
holder has it, and MUST then return a release function.

Supports: 792a3m1a, 792a3q1a, 792a4o1a, 792a4o1b.

##### 792b8c1a Scenarios

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

#### 792b8c2 Release

Requirement: 792b8c2 The vault-write-lock's release MUST unlock and then close the lock file, MUST
close the file even when unlocking fails, and MUST return the unlock error when unlocking fails,
else the close error when closing fails, each naming the lock path, else nothing.

Supports: 792a3m1a.

##### 792b8c2a Scenarios

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

#### 792b8c3 Acquire failures

Requirement: 792b8c3 When the lock file cannot be opened, the vault-write-lock MUST return an error
naming the lock path without trying to lock it; when the lock cannot be taken, it MUST close the
file and return an error naming the lock path.

Supports: 792a3m1a.

##### 792b8c3a Scenarios

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

#### 792b8c4 The vault's lock

Requirement: 792b8c4 The vault-write-lock MUST lock a vault by locking the file `.luhmann.lock` at
the vault root, and releasing that lock MUST release the file's lock.

Supports: 792a3m1a, 792a3q1a, 792a4o1a, 792a4o1b.

##### 792b8c4a Scenarios

```gherkin
Scenario: 792b8c4a1 A vault is locked through .luhmann.lock
  Given the vault /vault
  When the vault is locked and then released
  Then /vault/.luhmann.lock was locked, and released
```

### 792b8d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a3m1a | 792b8c1, 792b8c2, 792b8c4 |
| 792a3q1a | 792b8c1, 792b8c4 |
| 792a4o1a | 792b8c1, 792b8c4 |
| 792a4o1b | 792b8c1, 792b8c4 |

```gherkin
Scenario: 792a4o1a Concurrent amends both take effect
  Given a fact note
  When engram amend --target <note> --subject s2 and engram amend --target <note> --chunk-source a#1 start at the same time
  Then the note carries both subject: s2 and a#1 in sources:

Scenario: 792a4o1b An activation does not undo a re-embed
  Given a fact note with a current sidecar
  When engram amend --target <note> --object new and engram activate --note <note> start at the same time
  Then the sidecar holds the vectors of the amended text
```

## 792b9 Added: text-embedder

Canonical target: `docs/specs/technical/text-embedder.md` (title `Text Embedder - Technical Spec`).

### 792b9a Orientation

- Ownership: Repository-owned dependency-boundary interaction (the Hugot and GoMLX inference
  libraries, the bundled ONNX model, and the model cache directory).
- Responsibility: turns text into a fixed-length vector with the bundled model, in process and
  offline. It unpacks the model into a cache directory once, builds the model only when first
  needed, fits long input to the model's window, and reports a model ID and dimension for the
  vectors it makes.
- Current realization (informational): `internal/embed/hugot.go` (`HugotEmbedder`,
  `LazyEmbedder`, the bundled `assets/model`, `BundledModelID` `minilm-l6-v2@384`),
  `internal/embed/runtime.go` (the runtime backend), `internal/embed/cache.go` and
  `internal/embed/cachefs.go` (model cache), `CacheDirFromHome` in `internal/cli/targets.go`, and
  `cmd/engram/hugot.go`, the thin Hugot runtime.
- Qualification: responsibility, embedding text is stated without naming any caller; ownership,
  callers decide what text to embed and what to do with the vector, while this component owns the
  model, its cache and its window; independent verification, it is tested directly over fake
  backends and cache filesystems and over the real bundled model; stability, the contract survives
  a change of runtime library; necessity, every sidecar and every query phrase depends on it.
- Supported capabilities: 792a3 note-writing. Chunk-indexing, controlled-vocabulary,
  memory-query and vault-health add their mappings (792d30).

Rationale: an early design embedded through an external API, which brought a network dependency,
per-call cost and latency, and sent vault content off the machine. Instead the
`sentence-transformers/all-MiniLM-L6-v2` model (384 dimensions) is bundled into the binary with
`go:embed` and run in pure Go through Hugot and GoMLX's `simplego` backend, with no CGO, no daemon
and no network. The only external API in the system is the LLM that runs the agent. Embeddings are
therefore deterministic, offline and free per call, and the embedder is part of the engram binary
rather than an external system. Engram embeds note text when it writes a note and query text when
it reads.

### 792b9b Sequences

```mermaid
sequenceDiagram
  participant Caller as Writer or query
  participant Embedder as text-embedder
  participant Cache as Model cache directory
  participant Runtime as Hugot runtime
  Caller->>Embedder: embed text (792b9c1)
  opt first use in this process (792b9c4)
    Embedder->>Cache: unpack the bundled model unless already complete (792b9c5)
    Cache-->>Embedder: model directory
    Embedder->>Runtime: open a session and pipeline, probe once for the dimension (792b9c3, 792b9c7)
    Runtime-->>Embedder: pipeline, or a construction error
  end
  alt embedder built
    Embedder->>Runtime: run the text, cut to the window, halving on failure (792b9c2)
    Runtime-->>Embedder: vector, or error
    Embedder-->>Caller: vector, or error (792b9c1, 792b9c2)
  else construction failed
    Embedder-->>Caller: embedder-unavailable error (792b9c4)
  end
```

### 792b9c Behavior Requirements

#### 792b9c1 Bundled-model embedding

Requirement: 792b9c1 The text-embedder MUST embed text with the bundled model, model ID
`minilm-l6-v2@384`, returning one 384-value vector per text, and MUST match the reference
implementation's similarities for that model.

Supports: 792a3n1a.

##### 792b9c1a Scenarios

```gherkin
Scenario: 792b9c1a1 The bundled model embeds text
  Given the bundled embedder
  When it embeds "hello world" and a 4000-character text
  Then its model ID is minilm-l6-v2@384, its dimension is 384, and each vector has 384 values

Scenario: 792b9c1a2 The binary carries the model
  Given the binary's bundled assets
  When the model directory is listed
  Then it holds the model files

Scenario: 792b9c1a3 Similarities match the reference
  Given five text pairs with cosine similarities computed by the Python reference implementation
  When the bundled model embeds them
  Then each pair's cosine matches the reference

Scenario: 792b9c1a4 A run returns the pipeline's vector
  Given a pipeline that returns a vector for the text
  When the text is embedded and the embedder closed twice
  Then that vector is returned, and the pipeline is released once
```

#### 792b9c2 Input window

Requirement: 792b9c2 The text-embedder MUST cut text longer than 1500 characters to its first 1500,
and when the model run fails MUST halve the text and retry until a run succeeds or the text is
shorter than 100 characters, then return the error; a run that yields no vector MUST be an error.

Supports: 792a3n1a.

Rationale: 1500 characters fit the model's 512-token window for prose, but dense text such as code
can tokenize past it. Halving degrades a long chunk to a shorter prefix instead of failing a whole
ingest.

##### 792b9c2a Scenarios

```gherkin
Scenario: 792b9c2a1 An over-long failure retries shorter
  Given a model run that fails on long input and succeeds on shorter input
  When a long text is embedded
  Then a shorter prefix is embedded and its vector returned

Scenario: 792b9c2a2 A persistent failure surfaces
  Given a model run that always fails
  When a text is embedded
  Then the error is returned once the text is shorter than 100 characters

Scenario: 792b9c2a3 An empty result is an error
  Given a model run that returns no vector
  When a text is embedded
  Then the empty-result error is returned
```

#### 792b9c3 Construction

Requirement: 792b9c3 The text-embedder MUST build itself by opening the model's pipeline and
embedding one probe text to learn the dimension. It MUST return the open error when the pipeline
cannot open, and when the probe fails or yields an empty vector it MUST release the pipeline and
return the error, or the empty-probe error. It MUST refuse a model directory the runtime cannot
load, and closing it MUST release the pipeline and return any release error.

Supports: 792a3n1a, 792a3n1b.

##### 792b9c3a Scenarios

```gherkin
Scenario: 792b9c3a1 A probed embedder is built
  Given a pipeline whose probe returns a 2-value vector
  When the embedder is built
  Then its dimension is 2, and closing it releases the pipeline

Scenario: 792b9c3a2 An open failure is returned
  Given a pipeline that cannot open
  When the embedder is built
  Then the open error is returned

Scenario: 792b9c3a3 A failed probe releases the pipeline
  Given a pipeline whose probe fails
  When the embedder is built
  Then the pipeline is released and the probe error returned

Scenario: 792b9c3a4 An empty probe releases the pipeline
  Given a pipeline whose probe returns no vector, or a zero-length one
  When the embedder is built
  Then the pipeline is released and the empty-probe error returned

Scenario: 792b9c3a5 An invalid model directory is refused
  Given a model directory without a valid model.onnx
  When the real runtime builds an embedder from it
  Then an error is returned

Scenario: 792b9c3a6 A release error on close is returned
  Given a built embedder whose pipeline release fails
  When it is closed
  Then the release error is returned
```

#### 792b9c4 Lazy construction

Requirement: 792b9c4 The text-embedder MUST build the model at most once per process, on the first
embedding or dimension request, MUST report the bundled model ID without building it, and when
building fails MUST return an embedder-unavailable error from every embedding request and a
dimension of 0.

Supports: 792a3n1a, 792a3n1b.

Rationale: commands that never embed, such as `--help` or `update`, neither pay to unpack the model
nor fail when it cannot load.

##### 792b9c4a Scenarios

```gherkin
Scenario: 792b9c4a1 The model ID needs no build
  Given an embedder that has not been built
  When its model ID is asked
  Then it is the bundled model ID, and nothing is built

Scenario: 792b9c4a2 A failed build makes the embedder unavailable
  Given an embedder whose build fails
  When it is asked to embed and for its dimension
  Then embedding returns an embedder-unavailable error wrapping the build error, the dimension is 0, and the model ID is still the bundled one

Scenario: 792b9c4a3 A successful build is reused
  Given an embedder whose build succeeds
  When it is asked its dimension and model ID and to embed
  Then all three are served by the built embedder
```

#### 792b9c5 Model cache

Requirement: 792b9c5 The text-embedder MUST unpack the bundled model into the cache directory
`$XDG_CACHE_HOME/engram/models/<model ID>`, or `~/.cache/engram/models/<model ID>` without
`XDG_CACHE_HOME`, at most once: it MUST reuse a cache that carries its `.complete` marker, and
otherwise MUST unpack into a new sibling directory, write the marker, and rename that directory into
place. When the rename finds the cache already complete, it MUST discard its own copy and use the
cache; any other failure MUST remove its copy and be returned. A binary without the bundled model
MUST fail with an error saying the model is missing. Closing the embedder MUST NOT remove the cache.

Supports: 792a3n1a, 792a3n1b.

Rationale: the model is about 90 MB, so it is unpacked once and shared by every later run. The
rename makes the unpacked cache appear all at once, so a concurrent process never sees a half-written
model.

##### 792b9c5a Scenarios

```gherkin
Scenario: 792b9c5a1 The first use unpacks the model
  Given no cache directory
  When the embedder is built
  Then the model is unpacked into a sibling directory, marked complete, and renamed into place

Scenario: 792b9c5a2 A complete cache is reused
  Given a cache directory carrying its .complete marker
  When the embedder is built
  Then nothing is unpacked

Scenario: 792b9c5a3 A lost race uses the winner's cache
  Given another process completes the cache first
  When this process's rename finds the cache present and complete
  Then its own copy is removed and the cache is used

Scenario: 792b9c5a4 A race without a complete cache fails
  Given the rename finds the cache present but not complete
  When the embedder is built
  Then its own copy is removed and the rename error returned

Scenario: 792b9c5a5 Unpacking failures clean up
  Given creating the sibling directory, writing a model file, writing the marker, or renaming fails
  When the embedder is built
  Then any copy is removed and the error returned

Scenario: 792b9c5a6 A missing bundled model is a clear error
  Given a binary built without the model files
  When the embedder is built
  Then it fails with an error saying the bundled model is missing

Scenario: 792b9c5a7 Closing keeps the cache
  Given a built embedder
  When it is closed
  Then the cache directory still exists

Scenario: 792b9c5a8 The cache directory follows XDG
  Given home /Users/joe, model ID minilm-l6-v2@384, and XDG_CACHE_HOME unset or set to /custom/cache
  When the cache directory is resolved
  Then it is /Users/joe/.cache/engram/models/minilm-l6-v2@384, or /custom/cache/engram/models/minilm-l6-v2@384
```

#### 792b9c6 Cache filesystem

Requirement: 792b9c6 The text-embedder's cache filesystem MUST report a rename onto an existing,
populated directory as an "already exists" error, whatever error the platform gives, and MUST
report whether a directory carries the `.complete` marker it writes.

Supports: 792a3n1a.

Rationale: macOS reports a directory renamed over a non-empty one as "directory not empty", not
"exists"; translating it lets the race in 792b9c5 be recognized on every platform.

##### 792b9c6a Scenarios

```gherkin
Scenario: 792b9c6a1 A rename onto a populated directory is "exists"
  Given a destination directory that already holds files, on the real filesystem or behind either platform error
  When a directory is renamed onto it
  Then the error satisfies errors.Is for fs.ErrExist

Scenario: 792b9c6a2 The marker round-trips
  Given a directory
  When the .complete marker is written into it
  Then the directory is reported complete, and a directory without it is not
```

#### 792b9c7 Runtime lifecycle

Requirement: 792b9c7 The text-embedder's runtime backend MUST open a session and then a pipeline
named `engram-embed` over the model's `model.onnx`, MUST release the session when the pipeline
cannot open, and MUST return session, pipeline, run and release failures with their context. A
backend with no runtime wired MUST fail with the runtime-missing error rather than crash.

Supports: 792a3n1a.

##### 792b9c7a Scenarios

```gherkin
Scenario: 792b9c7a1 The pipeline is configured internally
  Given a runtime
  When a pipeline is opened for /models/m1
  Then the runtime gets the model path /models/m1, the name engram-embed and the file model.onnx

Scenario: 792b9c7a2 A pipeline failure releases the session
  Given a runtime whose pipeline cannot open
  When a pipeline is opened
  Then the session is released and the error names the pipeline

Scenario: 792b9c7a3 Session, run and release failures carry context
  Given a runtime whose session, run or release fails
  When the pipeline is opened, run or released
  Then the error names the failing step and wraps the cause

Scenario: 792b9c7a4 A run maps the vectors
  Given a pipeline run that returns vectors
  When it is run
  Then the same vectors are returned

Scenario: 792b9c7a5 A missing runtime fails loudly
  Given a backend with no runtime wired
  When a pipeline is opened
  Then the runtime-missing error is returned
```

### 792b9d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a3n1a | 792b9c1, 792b9c2, 792b9c3, 792b9c4, 792b9c5, 792b9c6, 792b9c7 |
| 792a3n1b | 792b9c3, 792b9c4, 792b9c5 |

```gherkin
Scenario: 792a3n1b A failed embedding keeps the note
  Given embedding fails for the note being learned
  When the user runs engram learn fact
  Then the note is written, stderr carries a warning naming it, and the command succeeds
```

## 792b10 Added: embedding-sidecar

Canonical target: `docs/specs/technical/embedding-sidecar.md` (title `Embedding Sidecar - Technical Spec`).

### 792b10a Orientation

- Ownership: Repository component.
- Responsibility: each note's `.vec.json` sidecar. It decides which text of a note is embedded and
  hashed, builds and writes the sidecar with two vectors, a model ID and a content hash, decodes
  and validates one, classifies a note's sidecar against the current model, and measures cosine
  similarity between vectors.
- Current realization (informational): `internal/embed/sidecar.go` (`Sidecar` format,
  `SidecarPath`, `MarshalSidecar`, `UnmarshalSidecar`), `internal/embed/embedder.go`
  (`BuildSidecar`, `State`), `internal/embed/hash.go` (`SituationText`, `BodyText`, `ContentHash`),
  `internal/embed/state.go` (`ComputeState`), `internal/embed/cosine.go` (`Cosine`), and
  `autoEmbedNote` in `internal/cli/learn.go`. The `engram embed apply` and `status` selection in
  `internal/cli/embed.go` is added by vault-health (792d66).
- Qualification: responsibility, the sidecar contract is stated without naming any writer or
  reader; ownership, callers decide when to build, read or classify a sidecar, while this component
  decides its content and what makes it current; independent verification, building, decoding,
  hashing and classification are tested directly over fake embedders and filesystems; stability,
  the on-disk format is versioned and outlives any refactoring; necessity, every retrieval over notes
  reads it.
- Supported capabilities: 792a2 vault-identity, 792a3 note-writing, 792a4 note-revision.
  Controlled-vocabulary, memory-query, vault-health and vault-migration add their mappings and
  requirements (792d30).

Rationale: semantic search needs vectors, but a vector database or a separate rebuild step is a
moving part that can drift from the notes. Instead every note carries its own sidecar, written as
part of the same command: there is no index to maintain or invalidate, and a copied vault is
self-contained. Each sidecar holds two vectors, one for the note's situation and one for its body,
so retrieval can match a query against whichever fits better. Notes of every kind embed this way;
the retired episode kind once embedded only its situation.

### 792b10b Sequences

```mermaid
sequenceDiagram
  participant Writer as Note writer
  participant Sidecar as embedding-sidecar
  participant Embedder as text-embedder
  participant Store as durable-file-store
  Writer->>Sidecar: write the sidecar for a just-written note (792b10c6)
  Sidecar->>Sidecar: take the situation text, or the body when there is none, and the body text (792b10c3)
  Sidecar->>Embedder: embed the situation text, then the body text (792b10c5)
  alt both embedded
    Embedder-->>Sidecar: two vectors
    Sidecar->>Store: atomically write <note>.vec.json with the vectors, model ID, dimension and content hash (792b10c1, 792b10c2, 792b10c4)
    alt write succeeded
      Sidecar-->>Writer: done
    else write failed
      Sidecar-->>Writer: warning naming the note, note kept (792b10c6)
    end
  else embedding failed
    Embedder-->>Sidecar: error
    Sidecar-->>Writer: warning naming the note, note kept (792b10c6)
  end
```

### 792b10c Behavior Requirements

#### 792b10c1 Sidecar format

Requirement: 792b10c1 The embedding-sidecar MUST store a sidecar as JSON with `schema_version` 1,
`embedding_model_id`, `dims`, `situation_vector`, `body_vector`, `content_hash`, and `last_used`
only when set. Decoding MUST refuse malformed JSON, MUST refuse any other schema version, such as an
old single-vector sidecar, before checking lengths, MUST refuse a sidecar whose either vector's
length differs from `dims`, and MUST read an absent `last_used` as empty.

Supports: 792a3n1a, 792a3u1a, 792a3u1b, 792a4n1a, 792a4n1d.

Rationale: `last_used` is added metadata. It does not bump the schema version, and it is kept out of
the content hash, so recording a use never makes a note look stale.

##### 792b10c1a Scenarios

```gherkin
Scenario: 792b10c1a1 A dual-vector sidecar round-trips
  Given a sidecar with both vectors, a model ID, a dimension and a content hash
  When it is encoded and decoded
  Then every field reads back unchanged

Scenario: 792b10c1a2 last_used is optional
  Given a sidecar with and without last_used
  When each is encoded and decoded
  Then last_used round-trips when set, and a sidecar without it decodes with last_used empty

Scenario: 792b10c1a3 A vector of the wrong length is refused
  Given a sidecar whose situation or body vector length differs from dims
  When it is decoded
  Then the dims-mismatch error is returned

Scenario: 792b10c1a4 Malformed JSON is refused
  Given text that is not valid sidecar JSON
  When it is decoded
  Then the malformed-sidecar error is returned

Scenario: 792b10c1a5 An old single-vector sidecar is a schema error
  Given a sidecar written by the earlier single-vector schema
  When it is decoded
  Then the schema-version error is returned
```

#### 792b10c2 Sidecar path

Requirement: 792b10c2 The embedding-sidecar MUST place a note's sidecar beside it, named after the
note with `.md` replaced by `.vec.json`, and MUST append `.vec.json` to any other path.

Supports: 792a3n1a.

##### 792b10c2a Scenarios

```gherkin
Scenario: 792b10c2a1 A note's sidecar path
  Given the note path 1aa.note.md
  When its sidecar path is formed
  Then it is 1aa.note.vec.json

Scenario: 792b10c2a2 A non-Markdown path gets the suffix appended
  Given a path that does not end in .md
  When its sidecar path is formed
  Then .vec.json is appended to it
```

#### 792b10c3 Embedding sources

Requirement: 792b10c3 The embedding-sidecar MUST take a note's situation text from its
`situation:` frontmatter value, trimmed, for any kind of note, and its body text from the text after
the frontmatter, less one leading blank line, with every machine-written line removed (lines
starting `Supersedes:`, `Contributors:`, `Answered by:` or `Answers:`) and trailing blank lines
reduced to one newline. Other body lines MUST be kept. A note with no frontmatter is all body.

Supports: 792a3n1a, 792a3q1a, 792a4g1a, 792a4g1b.

Rationale: the machine-written lines are link channels rewritten in place by learn, amend and
curation. Leaving them out means a link-only change neither moves the body vector nor makes the
sidecar stale. Trailing blanks are normalized because the writers end bodies differently.

##### 792b10c3a Scenarios

```gherkin
Scenario: 792b10c3a1 The situation is read from the frontmatter
  Given notes of any kind with a situation: value, including after a value ending in --- and with surrounding whitespace
  When the situation text is taken
  Then it is the value, trimmed

Scenario: 792b10c3a2 The body is the text after the frontmatter
  Given a note with frontmatter followed by a blank line and a body, or a note without frontmatter
  When the body text is taken
  Then it is the body alone, or the whole text

Scenario: 792b10c3a3 Machine-written lines are left out
  Given a body with Supersedes:, Contributors:, Answered by: or Answers: lines
  When the body text is taken
  Then those lines are gone and the rest is unchanged

Scenario: 792b10c3a4 Ordinary lines are kept
  Given a body with a "Related to" section or a vocabulary line
  When the body text is taken
  Then those lines are kept

Scenario: 792b10c3a5 Trailing blank lines are normalized
  Given bodies that differ only in their trailing blank lines
  When their body text is taken
  Then each ends in exactly one newline
```

#### 792b10c4 Content hash

Requirement: 792b10c4 The embedding-sidecar MUST hash a note as `sha256:` followed by the hex
SHA-256 of its situation text, a zero byte, and its body text, so the hash changes when either
changes, does not change for any other frontmatter edit, and is the same for the same note.

Supports: 792a3n1a, 792a4g1a, 792a4g1b.

##### 792b10c4a Scenarios

```gherkin
Scenario: 792b10c4a1 The hash covers situation and body
  Given a note with a situation and a body
  When it is hashed
  Then the hash is sha256: and the hex SHA-256 of the situation, a zero byte and the body

Scenario: 792b10c4a2 Changing either source changes the hash
  Given a note whose situation or body text changes, for any such texts
  When it is hashed again
  Then the hash differs

Scenario: 792b10c4a3 Other frontmatter does not change the hash
  Given a note whose triggers, tags or other non-situation frontmatter changes
  When it is hashed again
  Then the hash is the same

Scenario: 792b10c4a4 Hashing is repeatable
  Given any note
  When it is hashed twice
  Then both hashes are equal
```

#### 792b10c5 Building a sidecar

Requirement: 792b10c5 The embedding-sidecar MUST build a sidecar by embedding the situation text,
or the body text when the note has no situation, and the body text, and stamping schema version 1,
the embedder's model ID and dimension, and the note's content hash; an embedding failure MUST be
returned.

Supports: 792a3n1a, 792a3n1b, 792a4g1a, 792a4g1c, 792a4l1a, 792a4l1c, 792a4o1b.

##### 792b10c5a Scenarios

```gherkin
Scenario: 792b10c5a1 Both vectors are embedded and stamped
  Given a note with a situation and a body, and an embedder
  When its sidecar is built
  Then the situation vector embeds the situation, the body vector the body, and the model ID, dimension and hash are stamped

Scenario: 792b10c5a2 A note without a situation embeds its body twice
  Given a note with no situation: field
  When its sidecar is built
  Then both vectors embed the body text
```

#### 792b10c6 Writing a learned note's sidecar

Requirement: 792b10c6 After `engram learn` writes a note, the embedding-sidecar MUST build its
sidecar and atomically write it to the note's sidecar path. When building or writing fails, it MUST
log a warning naming the note and return without failing the command. With no embedder or no
sidecar writer wired, it MUST do nothing.

Supports: 792a3n1a, 792a3n1b.

Rationale: the note itself is already written in one atomic step, so a missing sidecar is
recoverable later by `engram embed apply --missing`.

##### 792b10c6a Scenarios

```gherkin
Scenario: 792b10c6a1 A learned note gets a valid sidecar
  Given a just-written note and an embedder
  When its sidecar is written
  Then a sidecar that decodes, with the embedder's model ID, is at its sidecar path

Scenario: 792b10c6a2 An embedding failure only warns
  Given an embedder that fails
  When the sidecar is written
  Then a warning naming the note is logged, and no error is returned

Scenario: 792b10c6a3 A write failure only warns
  Given a sidecar writer that fails
  When the sidecar is written
  Then a warning naming the note is logged, and no error is returned

Scenario: 792b10c6a4 Nothing is done without an embedder or writer
  Given no embedder, or no sidecar writer
  When the sidecar is written
  Then nothing is embedded or written
```

#### 792b10c7 State classification

Requirement: 792b10c7 The embedding-sidecar MUST classify a note against the current model as
`broken` when the note cannot be read; `missing` when its sidecar does not exist; `broken` when the
sidecar cannot be read, is malformed or has a dims mismatch; `incompatible` when it has another
schema version or another model ID; `stale` when its content hash differs from the note's; and `ok`
otherwise, checking the model before the hash.

Supports: 792a2c1a, 792a2d1a, 792a2d1b.

Rationale: a re-embed under a new model also picks up any content change, so a model mismatch takes
precedence. Every failure is a state rather than an error, so a pass over the whole vault never
stops early. All sidecars are expected to share one model ID. A query skips sidecars made by another
model with only an advisory, and fails only when none remains, so after a partial model change its
results can be silently thinner (792d66).

##### 792b10c7a Scenarios

```gherkin
Scenario: 792b10c7a1 A current sidecar is ok
  Given a note whose sidecar matches its content and the current model
  When it is classified
  Then it is ok

Scenario: 792b10c7a2 An absent sidecar is missing
  Given a note with no sidecar, whether the read error is fs.ErrNotExist, wraps it, or reports IsNotExist
  When it is classified
  Then it is missing, and an unrelated or nil error is not treated as absent

Scenario: 792b10c7a3 Changed content is stale
  Given a note whose body changed after its sidecar was written, including by adding a related-notes section
  When it is classified
  Then it is stale

Scenario: 792b10c7a4 Another model or schema is incompatible
  Given a sidecar made by another model, or by the old single-vector schema
  When it is classified
  Then it is incompatible

Scenario: 792b10c7a5 An unreadable sidecar is broken
  Given a sidecar that is malformed JSON or has a vector of the wrong length
  When it is classified
  Then it is broken

Scenario: 792b10c7a6 States have labels
  Given each state
  When it is printed
  Then it reads ok, missing, stale, incompatible or broken
```

#### 792b10c8 Cosine similarity

Requirement: 792b10c8 The embedding-sidecar MUST measure two vectors' cosine similarity, and MUST
return 0 when their lengths differ, when they are empty, or when either has zero magnitude.

Supports: 792a3t1a, 792a3u1a, 792a3u1b.

##### 792b10c8a Scenarios

```gherkin
Scenario: 792b10c8a1 A vector is fully similar to itself
  Given any non-zero vector
  When its cosine with itself is measured
  Then it is 1

Scenario: 792b10c8a2 Orthogonal vectors are not similar
  Given two orthogonal vectors
  When their cosine is measured
  Then it is 0

Scenario: 792b10c8a3 Vectors without signal score 0
  Given vectors of different lengths, or a zero vector
  When their cosine is measured
  Then it is 0
```

#### 792b10c9 CRLF notes

Requirement: 792b10c9 The embedding-sidecar MUST take the situation text, the body text and the
content hash of a note whose lines end in `\r\n` from its LF form, so that each equals the value
for the note's LF form.

Supports: 792a4m1a.

Rationale: a CRLF note used to be embedded from the wrong text, and a conversion to LF then changed
its hash. Reading the LF form makes the conversion that a revision makes leave the hash as it
would be for the LF note.

##### 792b10c9a Scenarios

```gherkin
Scenario: 792b10c9a1 A CRLF note embeds and hashes as its LF form
  Given a note whose lines end in \r\n, and the same note with LF line endings
  When their situation text, body text and content hash are taken
  Then each equals the LF note's
```

### 792b10d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a2c1a | 792b10c7 |
| 792a2d1a | 792b10c7 |
| 792a2d1b | 792b10c7 |
| 792a3n1a | 792b10c1, 792b10c2, 792b10c3, 792b10c4, 792b10c5, 792b10c6 |
| 792a3n1b | 792b10c5, 792b10c6 |
| 792a3q1a | 792b10c3 |
| 792a3t1a | 792b10c8 |
| 792a3u1a | 792b10c1, 792b10c8 |
| 792a3u1b | 792b10c1, 792b10c8 |
| 792a4g1a | 792b10c3, 792b10c4, 792b10c5 |
| 792a4g1b | 792b10c3, 792b10c4 |
| 792a4g1c | 792b10c5 |
| 792a4l1a | 792b10c5 |
| 792a4l1c | 792b10c5 |
| 792a4m1a | 792b10c9 |
| 792a4n1a | 792b10c1 |
| 792a4n1d | 792b10c1 |
| 792a4o1b | 792b10c5 |

```gherkin
Scenario: 792a3t1a The answer is found and the question is not
  Given a vault holding a pair whose question and answer both match a phrase
  When the user runs engram query with that phrase
  Then the answer note can appear in the results, and the question note never does
```

```gherkin
Scenario: 792a4g1a A content amend re-embeds
  Given a fact note with a current sidecar
  When the user runs engram amend --target <note> --object new
  Then the note's sidecar is rebuilt from the amended text, and engram embed status reports it ok

Scenario: 792a4g1b A provenance-only amend does not re-embed
  Given a fact note with a current sidecar
  When the user runs engram amend --target <note> with only --supersedes or --chunk-source
  Then the note's sidecar file is unchanged

Scenario: 792a4g1c A failed re-embed keeps the amend
  Given embedding fails for the note being amended
  When the user runs engram amend --target <note> --object new
  Then the note is amended, stderr carries a warning naming it, and the command succeeds

Scenario: 792a4n1d Amend records a use
  Given a note with a current sidecar
  When the user runs engram amend --target <note> --activate
  Then the sidecar's last_used is today
```

## 792b11 Added: note-provenance

Canonical target: `docs/specs/technical/note-provenance.md` (title `Note Provenance - Technical Spec`).

### 792b11a Orientation

- Ownership: Repository-owned dependency-boundary interaction (`git`, through the
  subprocess-runner, and the operating system's user lookup).
- Responsibility: works out who and where a note write comes from: the repository of the working
  directory and the writing user. It stamps them, with the vault name, on a note's first write and
  re-stamps them on a revision, and omits, with a warning, any value that does not resolve, except
  that a re-stamp keeps a user the note already has.
- Current realization (informational): `detectRepo`, `detectUser`, `firstWriteIdentity` and
  `identityStamp` in `internal/cli/identity.go`; `newLearnDeps` wires them for `learn`. The amend
  re-stamp is `identityStamp.stampPreservingUser`; identity backfill
  (`internal/cli/identity_backfill.go`) is specified by vault-migration (792d67).
- Qualification: responsibility, provenance detection is stated without naming any command;
  ownership, writers decide when to stamp, while this component decides what the values are;
  independent verification, detection and stamping are tested directly over a fake `git` and user
  lookup; stability, the fields and their sources survive any change to the writers; necessity,
  every note's `repo:`, `user:` and `vault:` come from it.
- Supported capabilities: 792a2 vault-identity, 792a3 note-writing, 792a4 note-revision. Vault-migration,
  vault-serving and parent-sync add their requirements (792d30, 792d67).

Rationale: detection shells out to `git` rather than reading `.git/config`, because reproducing
git's own configuration precedence (worktrees, `includeIf` and so on) would bring no benefit.
Identity is self-reported: it is detected from the writer's own environment at the moment of the
write and never verified, as `source:` already was; a verified identity could later replace it
without changing the fields. Detection never fails a write. A working directory outside git, like
an absent `project:`, simply leaves the field out.

### 792b11b Sequences

```mermaid
sequenceDiagram
  participant Writer as Note writer
  participant Provenance as note-provenance
  participant Runner as subprocess-runner
  Writer->>Provenance: stamp a first write with the resolved vault name (792b11c3)
  Provenance->>Runner: git remote get-url origin in the working directory (792b11c1)
  alt origin found
    Runner-->>Provenance: origin URL
  else no origin
    Provenance->>Runner: git rev-parse --show-toplevel
    Runner-->>Provenance: top-level path, or failure outside git
  end
  Provenance->>Runner: git config user.email (792b11c2)
  alt email found
    Runner-->>Provenance: email
  else no email
    Provenance->>Provenance: OS username, or nothing
  end
  opt user or vault name empty
    Provenance-->>Writer: one warning per empty field (792b11c3)
  end
  Provenance-->>Writer: repo, user and vault, empty ones omitted from the note (792b11c3)
```

```mermaid
sequenceDiagram
  participant Reviser as Note reviser
  participant Provenance as note-provenance
  Reviser->>Provenance: re-stamp a revised note with the resolved vault name (792b11c4)
  Provenance->>Provenance: resolve the repository and the user (792b11c1, 792b11c2)
  alt user resolved
    Provenance-->>Reviser: the note's repo, user and vault overwritten
  else user empty and the note has a user
    Provenance-->>Reviser: repo and vault overwritten, the note's user kept, one warning naming the note
  else user empty and the note has none
    Provenance-->>Reviser: repo and vault overwritten, no user
  end
```

### 792b11c Behavior Requirements

#### 792b11c1 Repository detection

Requirement: 792b11c1 The note-provenance MUST resolve the repository as the output of
`git remote get-url origin` in the working directory, trimmed; when that fails or prints nothing,
as the basename of the path printed by `git rev-parse --show-toplevel`; and otherwise, including
when the working directory cannot be resolved, as empty, without failing.

Supports: 792a3o1a, 792a3o1b, 792a3o1c, 792a4h1a, 792a4h1b, 792a4h1c.

##### 792b11c1a Scenarios

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

#### 792b11c2 User detection

Requirement: 792b11c2 The note-provenance MUST resolve the user as the output of
`git config user.email`, trimmed; when that fails or prints nothing, as the OS username; and when
that also fails or is unavailable, as empty, without failing.

Supports: 792a3p1a, 792a3p1b, 792a3p1c, 792a4h1a, 792a4h1b, 792a4h1d, 792a4h1e.

##### 792b11c2a Scenarios

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

#### 792b11c3 First-write stamp

Requirement: 792b11c3 On a note's first write, the note-provenance MUST stamp the resolved
repository, user and vault name; a stamp whose user is empty MUST leave `user:` out and log one
warning that user detection resolved empty, naming the command, and a stamp whose vault name is
empty MUST leave `vault:` out and log one warning.

Supports: 792a2b1a, 792a2b1b, 792a2b1c, 792a3o1a, 792a3o1b, 792a3o1c, 792a3p1a, 792a3p1b,
792a3p1c.

Rationale: an empty `user: ""` or `vault: ""` would look like a real value to every later reader.
Identity backfill, or a later amend that re-stamps, fills a missing user in.

##### 792b11c3a Scenarios

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

#### 792b11c4 Re-stamp

Requirement: 792b11c4 When a revision re-stamps a note, the note-provenance MUST overwrite the
note's repository, user and vault with the resolved repository, user and vault name; when the
resolved user is empty it MUST keep a non-empty user the note already has, logging one warning that
names the note and says user detection resolved empty, and MUST leave the user out when the note has
none. With no re-stamp to apply, it MUST change nothing.

Supports: 792a4h1a, 792a4h1b, 792a4h1c, 792a4h1d, 792a4h1e, 792a4h1f, 792a4h1g.

Rationale: the alternatives for an empty user were to refuse the revision, which blocks content
edits on a machine with broken identity, or to keep the old value silently, which hides the broken
environment (#776). The repository keeps fresh detection even when empty, because a working
directory outside git is legitimate.

##### 792b11c4a Scenarios

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

### 792b11d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a2b1a | 792b11c3 |
| 792a2b1b | 792b11c3 |
| 792a2b1c | 792b11c3 |
| 792a3o1a | 792b11c1, 792b11c3 |
| 792a3o1b | 792b11c1, 792b11c3 |
| 792a3o1c | 792b11c1, 792b11c3 |
| 792a3p1a | 792b11c2, 792b11c3 |
| 792a3p1b | 792b11c2, 792b11c3 |
| 792a3p1c | 792b11c2, 792b11c3 |
| 792a4h1a | 792b11c1, 792b11c2, 792b11c4 |
| 792a4h1b | 792b11c1, 792b11c2, 792b11c4 |
| 792a4h1c | 792b11c1, 792b11c4 |
| 792a4h1d | 792b11c2, 792b11c4 |
| 792a4h1e | 792b11c2, 792b11c4 |
| 792a4h1f | 792b11c4 |
| 792a4h1g | 792b11c4 |

```gherkin
Scenario: 792a4h1a Amend from another environment re-stamps
  Given a fact note whose repo:, user: and vault: differ from the current environment's
  When the user runs engram amend --target <note> with a content flag, --supersedes or --chunk-source
  Then the note's repo:, user: and vault: are the current environment's detected values

Scenario: 792a4h1b Amend backfills missing identity
  Given a fact note with no repo:, user: or vault:
  When the user runs engram amend --target <note> --object new
  Then the note gains the current environment's repo:, user: and vault:

Scenario: 792a4h1c The project field does not stand in for the repo
  Given a note whose project: differs from the current working directory's repository
  When the user runs engram amend --target <note> --object new
  Then the note's repo: is the current working directory's repository, not the project: value

Scenario: 792a4h1d Empty user detection keeps the prior user
  Given a note whose user: is alice, and neither git config user.email nor the OS username resolves
  When the user runs engram amend --target <note> --object new
  Then the note's user: is still alice, stderr carries a warning naming the note, and the amend succeeds

Scenario: 792a4h1e Empty user detection never writes an empty user
  Given a note with no user:, and neither git config user.email nor the OS username resolves
  When the user runs engram amend --target <note> --object new
  Then the note has no user: field

Scenario: 792a4h1g Recording a use keeps the declared identity
  Given a note whose repo:, user: and vault: differ from the current environment's
  When the user runs engram amend --target <note> --activate
  Then the note's repo:, user: and vault: are unchanged
```

## 792b12 Added: note-authoring

Canonical target: `docs/specs/technical/note-authoring.md` (title `Note Authoring - Technical Spec`).

### 792b12a Orientation

- Ownership: Repository component (caller of note-document-format, luhmann-addressing,
  vault-write-lock, embedding-sidecar, note-provenance and durable-file-store).
- Responsibility: carries out `engram learn fact`, `feedback`, `runbook` and `qa`. It validates the
  arguments, turns them into a note's fields, and orders the write: lock the vault, allocate the
  Luhmann ID, stamp provenance, render and create the note, write its sidecar and vocabulary tags,
  release the lock and report the path. For a question-and-answer pair it writes both notes and
  undoes the first when the second fails.
- Current realization (informational): `RunLearn`, `writeLearnUnderLock`, `writeLearnLocked`,
  `assembleLearnContent`, the `validate*` functions and the `learnArgsFrom*` mappings in
  `internal/cli/learn.go`; `RunLearnQA`, `writeQANotesUnderLock` and the QA validators in
  `internal/cli/qa.go`; the learn argument types in `internal/cli/targets.go` and `qa.go`;
  `newLearnDeps` and `newQaDeps`. Each `learn` subcommand is its own process; nothing here calls
  another subcommand.
- Qualification: responsibility, capturing a note is stated without naming any function or seam;
  ownership, it owns validation, field mapping and the order of the write, while the components it
  calls own the format, the IDs, the lock, the sidecar and provenance; independent verification,
  it is tested directly over injected dependencies; stability, the capture flow survives any
  refactoring of the components it calls; necessity, every new note is written through it.
- Supported capabilities: 792a3 note-writing. Controlled-vocabulary, skill-runbook-registration,
  write-memory, learn, vault-serving and parent-sync add their requirements (792d30).

Rationale: the vault lock spans everything from listing the existing IDs to creating the note, so
two concurrent writers never allocate the same ID; creating the note exclusively is a backstop that
fails rather than overwrites. A question-and-answer pair has no Luhmann ID, so it takes the lock only
around its two writes and embeds after releasing it: sidecar writes are atomic per file and need no
lock, and holding it through embedding would stall other writers.

### 792b12b Sequences

```mermaid
sequenceDiagram
  participant CLI as command-line-surface
  participant Author as note-authoring
  participant Lock as vault-write-lock
  participant Ids as luhmann-addressing
  participant Prov as note-provenance
  participant Format as note-document-format
  participant Store as durable-file-store
  participant Sidecar as embedding-sidecar
  CLI->>Author: learn fact, feedback or runbook with its arguments
  Author->>Author: validate slug, project, issue, tags and triggers (792b12c1)
  alt an argument is invalid
    Author-->>CLI: error, nothing written (792b12c1)
  else arguments valid
    Author->>Lock: lock the vault (792b12c3)
    alt lock failed
      Lock-->>Author: error
      Author-->>CLI: error, nothing written (792b12c3)
    else lock held
      Author->>Store: list the vault's note IDs
      Author->>Ids: next ID for the target and position
      Author->>Prov: stamp repo, user and vault
      Author->>Author: check tier, supersessions and required content (792b12c1, 792b12c2)
      alt listing, placement or content refused
        Author-->>CLI: error, nothing written (792b12c3)
      else note ready
        Author->>Format: name and render the note (792b12c4)
        Author->>Store: create the note exclusively
        Author->>Sidecar: write its sidecar
        Author->>Author: apply vocabulary tagging
        Author->>Lock: release
        Author-->>CLI: the note's path on stdout (792b12c3)
      end
    end
  end
```

```mermaid
sequenceDiagram
  participant CLI as command-line-surface
  participant Author as note-authoring
  participant Lock as vault-write-lock
  participant Store as durable-file-store
  participant Sidecar as embedding-sidecar
  CLI->>Author: learn qa with its arguments
  Author->>Author: validate slug, question, answer source, source and certainty (792b12c5)
  opt --answer-file given
    Author->>Store: read the answer file
  end
  Author->>Store: list the vault and check every contributor exists (792b12c5)
  alt invalid argument, unreadable answer file or unknown contributor
    Author-->>CLI: error, nothing written (792b12c5)
  else valid
    Author->>Lock: lock the vault
    Author->>Store: create the question note exclusively
    Author->>Store: create the answer note exclusively
    alt answer write failed
      Author->>Store: remove the question note
      Author->>Lock: release
      Author-->>CLI: error naming the answer failure, and any removal failure (792b12c5)
    else both written
      Author->>Lock: release
      Author->>Sidecar: write both sidecars
      Author->>Author: apply vocabulary tagging to the answer only
      Author-->>CLI: the question's path, then the answer's (792b12c5)
    end
  end
```

### 792b12c Behavior Requirements

#### 792b12c1 Argument validation

Requirement: 792b12c1 The note-authoring MUST refuse, before writing anything, a slug that is empty
or does not match `[a-z0-9-]+`, a project that is not kebab-case, an issue containing whitespace, a
tag that is not `<family>` or `<family>/<value>` with kebab-case segments, a trigger shorter than
three characters after trimming, a tier other than `L1`, `L2` or `L3`, a supersession that is not
`<note>|<type>|<claim>` with a non-empty note and claim and a type of `updates`, `narrows` or
`refutes`, and a note type other than fact, feedback or runbook. A claim MAY itself contain `|`.

Supports: 792a3g1a.

##### 792b12c1a Scenarios

```gherkin
Scenario: 792b12c1a1 An invalid slug is refused
  Given a learn with slug "Bad Slug"
  When it runs
  Then it fails with the invalid-slug error and nothing is written

Scenario: 792b12c1a2 Project, issue and tag shapes are checked
  Given a project, issue or tag value, empty or not
  When it is validated
  Then an empty project, issue or tag list and well-shaped values pass, and a badly shaped project, an issue with whitespace or a malformed tag is refused before any write

Scenario: 792b12c1a3 A short trigger is refused
  Given a runbook learn with a trigger shorter than three characters
  When it runs
  Then it fails and nothing is written

Scenario: 792b12c1a4 An unknown tier is refused
  Given a learn with tier L9
  When it runs
  Then it fails with the bad-tier error and nothing is written

Scenario: 792b12c1a5 A malformed supersession is refused
  Given a supersession with too few fields, an unknown type, an empty note or an empty claim
  When it is parsed
  Then it is refused with the matching error

Scenario: 792b12c1a6 A well-formed supersession is accepted
  Given a supersession of type updates, narrows or refutes, whose claim may contain |
  When it is parsed
  Then its note, type and claim are read, with the claim's | kept

Scenario: 792b12c1a7 An unknown note type is refused
  Given a learn of an unknown type such as principle
  When it runs
  Then it fails with the unknown-type error
```

#### 792b12c2 Required content

Requirement: 792b12c2 The note-authoring MUST refuse a fact, feedback or runbook whose situation is
empty or whitespace, and a runbook whose done-when is empty or whitespace, with an error naming the
missing flag and nothing written.

Supports: 792a3f1a, 792a3f1b, 792a3f1c.

##### 792b12c2a Scenarios

```gherkin
Scenario: 792b12c2a1 A fact or feedback without a situation is refused
  Given a fact or feedback learn whose situation is empty or whitespace
  When it runs
  Then it fails naming --situation and nothing is written

Scenario: 792b12c2a2 A runbook without a situation is refused
  Given a runbook learn whose situation is empty
  When it runs
  Then it fails naming --situation and nothing is written

Scenario: 792b12c2a3 A runbook without done-when is refused
  Given a runbook learn whose done-when is empty
  When it runs
  Then it fails naming --done-when and nothing is written
```

#### 792b12c3 Locked write

Requirement: 792b12c3 The note-authoring MUST, for a fact, feedback or runbook, take the vault lock;
list the vault's note IDs, which are the IDs of the Markdown files at its root; allocate the next ID
for the target and position; stamp provenance; render and exclusively create the note; write its
sidecar and apply vocabulary tagging; release the lock; and print the note's path. A lock failure,
a listing failure, a refused placement or refused content MUST return an error with nothing
written.

Supports: 792a3a1a, 792a3b1a, 792a3c1a, 792a3j1a, 792a3j1b, 792a3j1c, 792a3j1d, 792a3j1e,
792a3k1a, 792a3k1b, 792a3k1c, 792a3k1d, 792a3m1a, 792a3n1a, 792a3n1b, 792a3o1a, 792a3p1a, 792a3u1a.

##### 792b12c3a Scenarios

```gherkin
Scenario: 792b12c3a1 A note is written at its allocated path
  Given an empty vault and a fact, feedback or runbook learn with slug fact-slug at top on 2026-05-09
  When it runs
  Then the note is created at /vault/1.2026-05-09.fact-slug.md with its kind's frontmatter and body, and its path is printed

Scenario: 792b12c3a2 A lock failure writes nothing
  Given a vault lock that cannot be taken
  When a learn runs
  Then it fails with the lock error and nothing is written

Scenario: 792b12c3a3 A listing failure writes nothing
  Given a vault whose note IDs cannot be listed
  When a learn runs
  Then it fails with the listing error and nothing is written

Scenario: 792b12c3a4 Only root notes count as IDs
  Given a vault with notes 1 and 1a at its root, a note in a subdirectory and a README.md
  When its note IDs are listed
  Then they are exactly 1 and 1a
```

#### 792b12c4 Arguments to fields

Requirement: 792b12c4 The note-authoring MUST give a note `tier: L2` when no tier is given and the
given tier otherwise, and MUST carry the project, issue, tags, chunk sources, supersessions, and a
runbook's red flags and triggers in order, into the note's fields, with none written when none is
given.

Supports: 792a3d1a, 792a3d1b, 792a3d1c, 792a3h1a, 792a3h1b, 792a3i1a, 792a3i1b.

##### 792b12c4a Scenarios

```gherkin
Scenario: 792b12c4a1 The tier defaults to L2
  Given a fact or feedback learn without a tier, or with tier L3
  When it runs
  Then the note carries tier: L2, or tier: L3

Scenario: 792b12c4a2 Project and issue are carried
  Given a fact learn with --project engram and --issue 636
  When it runs
  Then the note carries project: engram and issue: "636"

Scenario: 792b12c4a3 Tags are carried
  Given a fact or feedback learn with tags, or with none
  When it runs
  Then the note carries the tags in order, or no tags: key

Scenario: 792b12c4a4 Chunk sources are carried
  Given a fact learn with chunk sources, or with none
  When it runs
  Then the note carries them under sources:, or no sources: key

Scenario: 792b12c4a5 Supersessions are carried
  Given a feedback learn with a supersession
  When it runs
  Then the note carries it under supersedes: and as a Supersedes: body line

Scenario: 792b12c4a6 Red flags are carried
  Given a runbook learn with repeated --red-flag values, or with none
  When it runs
  Then the note carries them in order under red_flags:, or no red_flags: key and no error

Scenario: 792b12c4a7 Triggers are carried
  Given a runbook learn with repeated --trigger values, or with none
  When it runs
  Then the note carries them in order under triggers: after red_flags:, or no triggers: key, for any valid triggers
```

#### 792b12c5 Question-and-answer write

Requirement: 792b12c5 The note-authoring MUST, for a question-and-answer pair, validate the
arguments, defaulting certainty to `medium`; read the answer from `--answer-file` when given; check
every contributor against the vault's Markdown files; and only then take the vault lock, create the
question note and then the answer note, both exclusively, and release the lock. When the answer note
cannot be created it MUST remove the question note and return an error saying so, also naming the
failure when the question cannot be removed. After a successful write it MUST write both sidecars,
apply vocabulary tagging to the answer only, and print the question's path and then the answer's.
A lock failure MUST return an error with nothing written.

Supports: 792a3n1a, 792a3q1a, 792a3q1b, 792a3q1c, 792a3r1a, 792a3r1b, 792a3s1a, 792a3u1b.

##### 792b12c5a Scenarios

```gherkin
Scenario: 792b12c5a1 A pair is written question first
  Given a valid pair
  When it is learned
  Then two notes are created, the .q.md first and the .a.md second

Scenario: 792b12c5a2 The answer comes from a file
  Given a pair whose answer is given by --answer-file
  When it is learned
  Then both notes are created with the file's text as the answer

Scenario: 792b12c5a3 Invalid arguments are refused
  Given a pair with an invalid slug, no question, both or neither answer source, no source, or an unknown certainty
  When it is validated
  Then it is refused with the matching error, and a fully valid pair passes

Scenario: 792b12c5a4 An unknown contributor is refused before any write
  Given a pair whose contributor names no note in the vault
  When it is learned
  Then it fails saying the contributor was not found, and nothing is written

Scenario: 792b12c5a5 A lock failure writes nothing
  Given a vault lock that cannot be taken
  When a pair is learned
  Then it fails with the lock error

Scenario: 792b12c5a6 A failed answer removes the question
  Given an answer note that cannot be created
  When a pair is learned
  Then the question note is removed and an error is returned

Scenario: 792b12c5a7 A failed removal is reported too
  Given an answer note that cannot be created and a question note that cannot be removed
  When a pair is learned
  Then the error names the orphaned question note
```

### 792b12d Product-Scenario Mapping

| Product scenario | Technical requirements |
| --- | --- |
| 792a3a1a | 792b12c3 |
| 792a3b1a | 792b12c3 |
| 792a3c1a | 792b12c3 |
| 792a3d1a | 792b12c4 |
| 792a3d1b | 792b12c4 |
| 792a3d1c | 792b12c4 |
| 792a3f1a | 792b12c2 |
| 792a3f1b | 792b12c2 |
| 792a3f1c | 792b12c2 |
| 792a3g1a | 792b12c1 |
| 792a3h1a | 792b12c4 |
| 792a3h1b | 792b12c4 |
| 792a3i1a | 792b12c4 |
| 792a3i1b | 792b12c4 |
| 792a3j1a | 792b12c3 |
| 792a3j1b | 792b12c3 |
| 792a3j1c | 792b12c3 |
| 792a3j1d | 792b12c3 |
| 792a3j1e | 792b12c3 |
| 792a3k1a | 792b12c3 |
| 792a3k1b | 792b12c3 |
| 792a3k1c | 792b12c3 |
| 792a3k1d | 792b12c3 |
| 792a3m1a | 792b12c3 |
| 792a3n1a | 792b12c3, 792b12c5 |
| 792a3n1b | 792b12c3 |
| 792a3o1a | 792b12c3 |
| 792a3p1a | 792b12c3 |
| 792a3q1a | 792b12c5 |
| 792a3q1b | 792b12c5 |
| 792a3q1c | 792b12c5 |
| 792a3r1a | 792b12c5 |
| 792a3r1b | 792b12c5 |
| 792a3s1a | 792b12c5 |
| 792a3u1a | 792b12c3 |
| 792a3u1b | 792b12c5 |

```gherkin
Scenario: 792a3r1b Certainty defaults to medium
  Given an existing vault
  When the user runs engram learn qa without --certainty
  Then the answer note carries certainty: medium
```
