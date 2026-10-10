# Command Line Surface - Technical Spec

## 792b2a Orientation

- Ownership: Repository component (caller), crossing the process argv, environment and stdio
  boundary through the targ library.
- Responsibility: turns one process invocation into one command run. It parses arguments and
  environment into a typed request, resolves the vault path, the vault name and the chunk index
  directory, runs the first-use vault step before every vault-resolving command, routes to the
  command, and maps the result to the output streams and the exit status.
- Commands: `learn` (`fact`, `feedback`, `qa`, `runbook`), `query`, `query-chunks`, `show`,
  `show-chunk`, `amend`, `activate`, `resituate`, `ingest`, `prune`, `count`, `check`, `embed`
  (`apply`, `status`), `vocab` (`bootstrap`, `tag-definitions`, `propose`, `stats`, `refit`),
  `update`, `register-skills`, `serve` and `vault-id`. Every command except `prune`, `query-chunks`
  and `show-chunk` resolves a vault. Each command's behavior is specified by its capability.
- Current realization (informational): `internal/cli/targets.go` (`Targets`, `runInVault`,
  `newErrHandler`), `internal/cli/cli.go`, `resolveVault` in `internal/cli/learn.go`,
  `resolveVaultName` in `internal/cli/identity.go`, `DataDirFromHome` in `internal/cli/targets.go`,
  `ResolveChunksDir` in `internal/cli/ingest.go`;
  `cmd/engram/main.go` wires the production primitives.
- Qualification: responsibility, it owns parsing, precedence and routing for every command without
  depending on how any command works; ownership, its routing, gating and exit mapping are distinct
  from each command's own behavior; independent verification, its precedence and gating are tested
  through dispatch with injected environment and filesystem; stability, the surface survives any
  refactoring of command internals; necessity, every product capability is reached through it.
- Supported capabilities: [vault-identity](../product/vault-identity.md), [note-writing](../product/note-writing.md), [note-revision](../product/note-revision.md), [chunk-indexing](../product/chunk-indexing.md)
  chunk-indexing, [controlled-vocabulary](../product/controlled-vocabulary.md), [memory-query](../product/memory-query.md), [parent-sync](../product/parent-sync.md) ([792b2c9 A removed environment variable is refused](#792b2c9-a-removed-environment-variable-is-refused), [792b2c10 Parent routing of note lookup](#792b2c10-parent-routing-of-note-lookup)).
  Later capabilities add their own routing requirements here.

Rationale: each `engram <subcommand>` is its own process. Commands never call each other in
process; a skill reads one command's stdout and runs the next. (An earlier C3 draft drew an
in-process `query`→`learn` edge that does not exist.) Each skill step runs the CLI as a fresh
subprocess, and the skill layer reaches the vault only through it: it reads notes through
`engram show`, `engram show-chunk` and the query payload, and writes through `engram amend` and
`engram learn`, never by editing vault files directly.

## 792b2c Behavior Requirements

### 792b2c1 Vault path resolution

Requirement: [792b2c1 Vault path resolution](#792b2c1-vault-path-resolution) The command-line-surface MUST resolve the vault path to the `--vault` flag
when it is non-empty, else to `ENGRAM_VAULT_PATH` when it is non-empty, else to
`$XDG_DATA_HOME/engram/vault` when `XDG_DATA_HOME` is non-empty, else to
`$HOME/.local/share/engram/vault`, and MUST return that path without checking that it exists.
`engram update` defines no `--vault` flag, so it resolves with an empty flag value.

Supports: [792a2a1a The flag wins over the environment](../product/vault-identity.md#792a2a1-uat-scenarios), [792a2a1b The environment variable is used without a flag](../product/vault-identity.md#792a2a1-uat-scenarios), [792a2a1c The XDG data directory is the default](../product/vault-identity.md#792a2a1-uat-scenarios), [792a2a1d The home directory is the last fallback](../product/vault-identity.md#792a2a1-uat-scenarios), [792a6t1a Stats on an empty vault](../product/controlled-vocabulary.md#792a6t1-uat-scenarios).

#### 792b2c1a Scenarios

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

### 792b2c2 Vault name resolution

Requirement: [792b2c2 Vault name resolution](#792b2c2-vault-name-resolution) The command-line-surface MUST resolve the vault name to the `--vault-name`
flag when it is non-empty, else to `ENGRAM_VAULT_NAME` when it is non-empty, else to `personal`.

Supports: [792a2b1a The default vault name is personal](../product/vault-identity.md#792a2b1-uat-scenarios), [792a2b1b The environment names the vault](../product/vault-identity.md#792a2b1-uat-scenarios), [792a2b1c The flag wins over the environment](../product/vault-identity.md#792a2b1-uat-scenarios), [792a4h1f Without a vault name the default is stamped](../product/note-revision.md#792a4h1-uat-scenarios).

#### 792b2c2a Scenarios

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

### 792b2c3 First-use gating

Requirement: [792b2c3 First-use gating](#792b2c3-first-use-gating) For every vault-resolving command, the command-line-surface MUST ask the
[vault-identity-registry](./vault-identity-registry.md) to ensure the vault at the resolved path before running the command, and
MUST NOT run the command when that step fails.

Supports: [792a2e1a A read command creates the vault](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1b amend and activate fail only on their own merits](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1c A child host gets its own vault](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1d A second run creates nothing](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2f1a An existing vault is untouched by read-only use](../product/vault-identity.md#792a2f1-uat-scenarios), [792a2g1a A half-created vault gets its ID](../product/vault-identity.md#792a2g1-uat-scenarios), [792a6t1a Stats on an empty vault](../product/controlled-vocabulary.md#792a6t1-uat-scenarios).

Rationale: one shared first-use step in dispatch, rather than one per command, makes every command
create a missing vault the same way (#766).

#### 792b2c3a Scenarios

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

### 792b2c4 Result reporting

Requirement: [792b2c4 Result reporting](#792b2c4-result-reporting) The command-line-surface MUST write a failed command's error to stderr and exit
with a non-zero status, and MUST neither exit nor write to stderr on the command's behalf when the
command succeeds.

Supports: [792a2e1b amend and activate fail only on their own merits](../product/vault-identity.md#792a2e1-uat-scenarios), [792a2j1c vault-id reports the corrupt file](../product/vault-identity.md#792a2j1-uat-scenarios), [792a2n1b Claim needs an ID](../product/vault-identity.md#792a2n1-uat-scenarios), [792a2o1a Both flags are refused](../product/vault-identity.md#792a2o1-uat-scenarios), [792a3f1b A blank situation is refused](../product/note-writing.md#792a3f1-uat-scenarios), [792a3g1a An invalid argument is refused before any write](../product/note-writing.md#792a3g1-uat-scenarios), [792a3k1a A missing position is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3k1b A continuation without a target is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3k1c A sibling of a top-level note is refused](../product/note-writing.md#792a3k1-uat-scenarios),
[792a3k1d A malformed target is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3r1a An invalid pair is refused before any write](../product/note-writing.md#792a3r1-uat-scenarios), [792a3s1a A failed answer removes the question](../product/note-writing.md#792a3s1-uat-scenarios), [792a4a1b An unknown reference is refused](../product/note-revision.md#792a4a1-uat-scenarios), [792a4c1a A question-and-answer note is refused](../product/note-revision.md#792a4c1-uat-scenarios), [792a4f1a An invalid amend argument is refused before any write](../product/note-revision.md#792a4f1-uat-scenarios), [792a4k1a Amend refuses an anchored edited key](../product/note-revision.md#792a4k1-uat-scenarios), [792a4k1b Resituate refuses an anchored situation](../product/note-revision.md#792a4k1-uat-scenarios), [792a4l1b A runbook cannot be resituated](../product/note-revision.md#792a4l1-uat-scenarios), [792a4l1c A failed sidecar rebuild fails the command](../product/note-revision.md#792a4l1-uat-scenarios),
[792a4n1c A missing note fails only its own ref](../product/note-revision.md#792a4n1-uat-scenarios), [792a6h1a A bad seed is refused](../product/controlled-vocabulary.md#792a6h1-uat-scenarios), [792a6i1c Both flags are required](../product/controlled-vocabulary.md#792a6i1-uat-scenarios), [792a6r1a The plan flow is gone](../product/controlled-vocabulary.md#792a6r1-uat-scenarios), [792a6t1a Stats on an empty vault](../product/controlled-vocabulary.md#792a6t1-uat-scenarios), [792a8c1f A filter without = is refused](../product/vault-counting.md#792a8c1-uat-scenarios), [792a8e1j An unreadable note fails the report](../product/vault-counting.md#792a8e1-uat-scenarios), [792a8f1a Both flags are refused](../product/vault-counting.md#792a8f1-uat-scenarios), [792a8f1b Neither flag is refused](../product/vault-counting.md#792a8f1-uat-scenarios), [792a8f1c A filter alone is not a mode](../product/vault-counting.md#792a8f1-uat-scenarios),
[792a9a1b A FAIL line fails the command](../product/vault-health.md#792a9a1-uat-scenarios), [792a9a1e An unreadable note stops the check](../product/vault-health.md#792a9a1-uat-scenarios), [792a9g1c An unreadable note stops the census](../product/vault-health.md#792a9g1-uat-scenarios), [792a9i1f An unreadable note stops the run](../product/vault-health.md#792a9i1-uat-scenarios), [792a10d1a A missing answers file](../product/vault-migration.md#792a10d1-uat-scenarios), [792a10d1b Malformed answers](../product/vault-migration.md#792a10d1-uat-scenarios), [792a10d1c Stale answers](../product/vault-migration.md#792a10d1-uat-scenarios), [792a10d1d An unknown note](../product/vault-migration.md#792a10d1-uat-scenarios).

#### 792b2c4a Scenarios

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

### 792b2c5 Argument refusal

Requirement: [792b2c5 Argument refusal](#792b2c5-argument-refusal) The command-line-surface MUST refuse an invocation that passes a flag the
command does not define, or omits a flag the command marks required, by writing an error naming the
flag to stderr and exiting non-zero without running the command.

Supports: [792a3e1a A task type is refused](../product/note-writing.md#792a3e1-uat-scenarios), [792a3f1a A missing situation is refused](../product/note-writing.md#792a3f1-uat-scenarios), [792a3f1c A runbook without done-when is refused](../product/note-writing.md#792a3f1-uat-scenarios), [792a3g1a An invalid argument is refused before any write](../product/note-writing.md#792a3g1-uat-scenarios), [792a3r1a An invalid pair is refused before any write](../product/note-writing.md#792a3r1-uat-scenarios), [792a6h1a A bad seed is refused](../product/controlled-vocabulary.md#792a6h1-uat-scenarios), [792a6i1c Both flags are required](../product/controlled-vocabulary.md#792a6i1-uat-scenarios), [792a6r1a The plan flow is gone](../product/controlled-vocabulary.md#792a6r1-uat-scenarios).

#### 792b2c5a Scenarios

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

### 792b2c6 Chunk index directory resolution

Requirement: [792b2c6 Chunk index directory resolution](#792b2c6-chunk-index-directory-resolution) For `ingest`, `prune`, `query`, `query-chunks`, `show-chunk`, `amend` and
`serve`, the command-line-surface MUST resolve the chunk index directory to the `--chunks-dir` flag
when it is non-empty, else to `ENGRAM_CHUNKS_DIR` when it is non-empty, else to
`$XDG_DATA_HOME/engram/chunks` when `XDG_DATA_HOME` is non-empty, else to
`$HOME/.local/share/engram/chunks`, and MUST return that path without checking that it exists.

Supports: [792a5a1a The flag wins over the environment](../product/chunk-indexing.md#792a5a1-uat-scenarios), [792a5a1b The environment variable is used without a flag](../product/chunk-indexing.md#792a5a1-uat-scenarios), [792a5a1c The XDG data directory is the default](../product/chunk-indexing.md#792a5a1-uat-scenarios), [792a5a1d The home directory is the last fallback](../product/chunk-indexing.md#792a5a1-uat-scenarios).

#### 792b2c6a Scenarios

```gherkin
Scenario: 792b2c6a1 The flag wins
  Given the --chunks-dir flag is /from/flag and ENGRAM_CHUNKS_DIR is /from/env
  When the chunk index directory is resolved
  Then the directory is /from/flag

Scenario: 792b2c6a2 The environment is used without a flag
  Given the --chunks-dir flag is empty and ENGRAM_CHUNKS_DIR is /from/env
  When the chunk index directory is resolved
  Then the directory is /from/env

Scenario: 792b2c6a3 XDG_DATA_HOME sets the default
  Given no flag, no ENGRAM_CHUNKS_DIR, and XDG_DATA_HOME is /xdg/data
  When the chunk index directory is resolved
  Then the directory is /xdg/data/engram/chunks

Scenario: 792b2c6a4 The home directory is the last fallback
  Given no flag, no ENGRAM_CHUNKS_DIR, no XDG_DATA_HOME, and home /home/dev
  When the chunk index directory is resolved
  Then the directory is /home/dev/.local/share/engram/chunks
```

### 792b2c7 Note lookup

Requirement: [792b2c7 Note lookup](#792b2c7-note-lookup) For `show`, the command-line-surface MUST read the note that the reference names,
a full basename, a Luhmann ID, or either of them in `[[ ]]` with an optional `|display` or with a
trailing `.md`, and MUST print its content exactly as stored, a newline after it when it lacks one,
a blank line, the line `# outbound links (fetch with: engram show <basename>)`, and then the target
of each link in the note's text ([792b13c3 Link targets](./wikilink-graph.md#792b13c3-link-targets)), one per line, or `(none)`. It MUST fail with
`show: empty note reference` when the reference is blank, and with `show: note not found: "<ref>"`,
quoting the reference as given, when no note matches. It MUST NOT shorten a runbook's `red_flags`,
and MUST NOT change any note.

Supports: [792a7x1a A note is printed with its links](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1b A note with no links says so](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1c Every reference form resolves](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1d Show never truncates red flags](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1e A fenced link is not listed](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1f An unknown note is an error](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1g A blank reference is an error](../product/memory-query.md#792a7x1-uat-scenarios), [792a7x1h Each link is listed once](../product/memory-query.md#792a7x1-uat-scenarios).

Rationale: `engram show` is the full-fidelity source that the guidance shim names, so the preview
budget that bounds a query payload ([792a7w Red flags in the query preview](../product/memory-query.md#792a7w-red-flags-in-the-query-preview)) is not applied here. Resolving a note that is on the
parent vault, by `--parent` or by a miss with a parent configured, is [792b2c10 Parent routing of note lookup](#792b2c10-parent-routing-of-note-lookup).

#### 792b2c7a Scenarios

```gherkin
Scenario Outline: 792b2c7a1 Every reference form resolves
  Given a vault holding the note 1.hub
  When `engram show <ref>` runs
  Then the note's content is printed

  Examples:
    | ref             |
    | 1.hub           |
    | 1               |
    | [[1.hub]]       |
    | 1.hub.md        |
    | [[1.hub\|the hub]] |

Scenario: 792b2c7a2 The outbound links skip fenced code
  Given the note 1.hub linking [[2.alpha]] and [[3.beta]] and holding [[9.fenced]] inside a code fence
  When `engram show 1.hub` runs
  Then the outbound links section lists 2.alpha and 3.beta and not 9.fenced

Scenario: 792b2c7a3 An unknown reference fails
  Given a vault without a note 99.nonexistent
  When `engram show 99.nonexistent` runs
  Then it fails with a not-found error

Scenario: 792b2c7a4 A blank reference fails
  Given any vault
  When `engram show` runs with the reference "  "
  Then it fails

Scenario: 792b2c7a5 A runbook's red flags are printed whole
  Given a runbook with 20 red flags that take more than 1200 bytes
  When `engram show` runs for it
  Then every entry is printed in file order and the output holds no OMITTED marker

Scenario: 792b2c7a6 The production filesystem is read
  Given a temporary vault on disk holding 1.real, which links [[2.other]]
  When `engram show 1.real` runs over the production filesystem wiring
  Then the note's body and 2.other are printed
```

### 792b2c8 Chunk lookup

Requirement: [792b2c8 Chunk lookup](#792b2c8-chunk-lookup) For `show-chunk`, the command-line-surface MUST read every index file in the
resolved chunk index directory ([792b2c6 Chunk index directory resolution](#792b2c6-chunk-index-directory-resolution), [792b18c12 Listing index files](./chunk-index-store.md#792b18c12-listing-index-files)) and print the text of the record whose
`<source>#<anchor>` equals the reference with surrounding whitespace trimmed, a newline after it when
it lacks one. It MUST fail with `show-chunk: empty chunk ref` when the reference is blank, and with
`chunk not found: <ref>` when no record has that ID. It MUST NOT embed anything and MUST NOT change
any file.

Supports: [792a7t1a A chunk's text is printed](../product/memory-query.md#792a7t1-uat-scenarios), [792a7t1b An unknown chunk is an error](../product/memory-query.md#792a7t1-uat-scenarios), [792a7t1c An empty ID is an error](../product/memory-query.md#792a7t1-uat-scenarios).

Rationale: with `engram query --lazy-chunks` ([792a7s Lazy chunks](../product/memory-query.md#792a7s-lazy-chunks)) a payload omits chunk text, and the agent
fetches one chunk's evidence on demand; the lookup is by ID and needs no model.

#### 792b2c8a Scenarios

```gherkin
Scenario: 792b2c8a1 A chunk's text is printed
  Given an index holding /s/a.jsonl#turn-1 and /s/a.jsonl#turn-2
  When `engram show-chunk /s/a.jsonl#turn-2` runs
  Then only the second chunk's text is printed

Scenario: 792b2c8a2 An unknown chunk fails
  Given an index holding only /s/a.jsonl#turn-1
  When `engram show-chunk /s/a.jsonl#turn-9` runs
  Then it fails with "chunk not found: /s/a.jsonl#turn-9"

Scenario: 792b2c8a3 A blank reference fails
  Given any chunk index
  When `engram show-chunk` runs with the reference "   "
  Then it fails
```

### 792b2c9 A removed environment variable is refused

Requirement: [792b2c9 A removed environment variable is refused](#792b2c9-a-removed-environment-variable-is-refused) Before it builds any command, the command-line-surface MUST refuse every invocation, `serve` included,
when the environment variable `ENGRAM_SERVER` is set to a non-empty value. The refusal MUST be an error that says
`ENGRAM_SERVER is no longer supported`, tells the user to set `ENGRAM_PARENT` to the same value, and is reported through
the usual error mapping ([792b2c4 Result reporting](#792b2c4-result-reporting)). It MUST happen before any vault, chunk index, lock, network or bind capability is
used, and `ENGRAM_SERVER` MUST NOT be read for any other purpose: its value is only echoed in the message.

Supports: [792a21a1a A command run with ENGRAM_SERVER set fails fast](../product/parent-sync.md#792a21a1-uat-scenarios), [792a21a1b Writes are refused too](../product/parent-sync.md#792a21a1-uat-scenarios), [792a21a1c serve is not exempt](../product/parent-sync.md#792a21a1-uat-scenarios), [792a21a1d ENGRAM_SERVER is not honored as an alias](../product/parent-sync.md#792a21a1-uat-scenarios).

Rationale: the thin-client mode that read this variable sent ten commands to a remote server and touched no local file.
Joe ruled out an alias shim and an exemption for `serve`, so the check sits ahead of every target and
nothing can reach a vault, a lock, a fetch or a bind first. A fleet host that still sets it fails on its first command
and is told the replacement.

#### 792b2c9a Scenarios

```gherkin
Scenario: 792b2c9a1 Every command is refused before anything is touched
  Given ENGRAM_SERVER is set to a URL
  When any subcommand, serve included, is run
  Then it exits non-zero with a message naming ENGRAM_PARENT and the same URL, and no filesystem, lock, fetch or bind capability is used

Scenario: 792b2c9a2 The value is only echoed
  Given ENGRAM_SERVER is set and ENGRAM_PARENT is not
  When any subcommand is run
  Then no fetch is issued, and the message names ENGRAM_PARENT with exactly the value ENGRAM_SERVER held
```

### 792b2c10 Parent routing of note lookup

Requirement: [792b2c10 Parent routing of note lookup](#792b2c10-parent-routing-of-note-lookup) For `show`, the command-line-surface MUST, when `--parent` is set, resolve the reference against
`ENGRAM_PARENT` alone and print the parent's answer unlabeled, and MUST fail with `--parent requires ENGRAM_PARENT to be
configured`, making no lookup, when `ENGRAM_PARENT` is unset. Without `--parent` it MUST look the reference up locally
first ([792b2c7 Note lookup](#792b2c7-note-lookup)) and return a local hit without contacting the parent. Only when the local lookup fails because the note
was not found and `ENGRAM_PARENT` is configured MUST it ask the parent, and then it MUST print the line `# from_parent:
true` followed by the parent's answer, printing the label only when the parent answered. Any other local error, and a
miss with no parent configured, MUST be returned as the local error. Each request to the parent MUST be a parent contact:
it MUST stamp a missing vault ID and check the vault's location first ([792b5c4 Vault ID stamping](./vault-identity-registry.md#792b5c4-vault-id-stamping), [792b5c7 Location record and check](./vault-identity-registry.md#792b5c7-location-record-and-check)), and a failed stamp MUST still
let the request run with its bookkeeping skipped so that no later write creates `.engram/` without a stamp; it MUST be
skipped inside a backoff window after the one warning ([792b5c14 The parent cache and the contact backoff](./vault-identity-registry.md#792b5c14-the-parent-cache-and-the-contact-backoff)), the fallback then returning the local not-found error
and `--parent` returning an error; and its outcome MUST be recorded for backoff. `show-chunk` MUST NOT accept `--parent`
and MUST NOT contact the parent.

Supports: [792a21l1a --parent routes to the configured parent](../product/parent-sync.md#792a21l1-uat-scenarios), [792a21l1b show-chunk has no parent route](../product/parent-sync.md#792a21l1-uat-scenarios), [792a21l1c Without --parent a local note is returned as before](../product/parent-sync.md#792a21l1-uat-scenarios), [792a21l1d A local miss falls back to the parent](../product/parent-sync.md#792a21l1-uat-scenarios), [792a21l1e A local miss with no parent is still an error](../product/parent-sync.md#792a21l1-uat-scenarios), [792a21l1f --parent without ENGRAM_PARENT is an error](../product/parent-sync.md#792a21l1-uat-scenarios), [792a21l1g The parent-sourced label comes before the parent's header](../product/parent-sync.md#792a21l1-uat-scenarios).

Rationale: a bare Luhmann ID is minted per vault, so a lookup that guessed local-first without being told could return the
wrong note with no error; the explicit flag and the origin tag of a merged item ([792a21g Merged items say where they came from](../product/parent-sync.md#792a21g-merged-items-say-where-they-came-from)) remove the guess, and a
fallback that fires only after a local miss cannot return a wrong local note. Chunks never cross vaults, so there is no
parent chunk lookup. The explicit `--parent` route is byte-identical to the parent's own answer, and only
the fallback is labeled, because only the fallback leaves the reader unsure where the text came from.

#### 792b2c10a Scenarios

```gherkin
Scenario: 792b2c10a1 --parent asks the parent only
  Given ENGRAM_PARENT is set
  When `engram show <ref> --parent` runs
  Then the ref is requested from the parent, whatever the local vault holds, and the output is the parent's answer without a label

Scenario: 792b2c10a2 --parent with no parent is an error
  Given ENGRAM_PARENT is unset
  When `engram show <ref> --parent` runs
  Then it fails and no request is made and no local fallback runs

Scenario: 792b2c10a3 A local hit does not contact the parent
  Given ENGRAM_PARENT is set and the note exists locally
  When `engram show <ref>` runs
  Then the local note is printed and no request is made

Scenario: 792b2c10a4 A local miss falls back, labeled
  Given ENGRAM_PARENT is set and the note is not local
  When `engram show <ref>` runs
  Then the parent's answer is printed after the line "# from_parent: true", and the label precedes the parent note's exchange-hash header

Scenario: 792b2c10a5 Without a parent a miss is the local error
  Given ENGRAM_PARENT is unset and the note is not local
  When `engram show <ref>` runs
  Then it fails with the local not-found error and the fetch capability is never invoked

Scenario: 792b2c10a6 Another local error does not reach the parent
  Given ENGRAM_PARENT is set
  When `engram show` runs with a blank reference
  Then it fails with the local error and no request is made

Scenario: 792b2c10a7 A backoff window skips the fallback
  Given ENGRAM_PARENT is set, the parent is backed off, and the note is not local
  When `engram show <ref>` runs
  Then no request is made and the local not-found error is returned after one warning

Scenario: 792b2c10a8 A backoff window skips --parent
  Given the parent is backed off
  When `engram show <ref> --parent` runs
  Then no request is made and the command fails after one warning

Scenario: 792b2c10a9 A parent contact stamps the vault ID
  Given a vault with no vault ID
  When `engram show <ref> --parent` runs, or a local miss falls back to the parent
  Then the vault ID is stamped before the request

Scenario: 792b2c10a10 A failed stamp skips only the bookkeeping
  Given a vault whose ID cannot be stamped
  When `engram show <ref> --parent` runs
  Then the request still runs, no backoff is recorded, and .engram is not created

Scenario: 792b2c10a11 show-chunk never reaches the parent
  Given ENGRAM_PARENT is set
  When `engram show-chunk <id> --parent` runs, and when `engram show-chunk <id>` runs for a missing chunk
  Then the first is rejected as an unknown flag with no request, and the second returns the local not-found error with no request
```
