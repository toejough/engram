# Vault Init Command - Product Spec

## 795a1 Problem Statement

Joe, and anyone who runs engram on a new machine, gets a vault created as a side effect of whichever
engram command runs first against a missing path ([792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation)). A mistyped `--vault` flag or
`ENGRAM_VAULT_PATH` therefore silently builds a second, empty vault, and the user sees only one stderr
line before the command carries on against it. New vaults also get an `.obsidian/` starter that not
every user wants. Interrupted creation is repaired just as automatically by the next command ([792a2g Interrupted creation is repaired](../../specs/product/vault-identity.md#792a2g-interrupted-creation-is-repaired)).
Creation should be a deliberate act, and every other command should refuse to run without a vault and
say how to get one.

This change, decided with Joe on 2026-10-10 in issue #795, supersedes the #766 decision of creation by
any command:

- `engram init` creates the vault, repairs a vault whose creation was interrupted, and on an existing
  vault reports that nothing changed.
- Every other command that needs a vault fails, naming `engram init`, `--vault` and `ENGRAM_VAULT_PATH`,
  when the vault is missing or half-created. `engram ingest`, which writes only to the chunk index, is
  unaffected.
- New vaults get no `.obsidian/` starter. Existing vaults are untouched.
- `engram update` with no vault skips its vault steps and prints a notice pointing to `engram init`. It
  never creates a vault.
- A child host with `ENGRAM_PARENT` set needs `engram init` once, because every environment keeps its own
  local vault ([parent-sync](../../specs/product/parent-sync.md)).

The starter README's content and recognition are a separate change (#796) and are not covered here. The
technical requirements that implement first-use creation, such as [792b2c3 First-use gating](../../specs/technical/command-line-surface.md#792b2c3-first-use-gating), need a later technical-spec
change; this change specifies product behavior only.

Terms used below:

- **No vault:** the resolved vault path ([792a2a Vault location](../../specs/product/vault-identity.md#792a2a-vault-location)) is absent or is an empty directory.
- **Half-created vault:** the path holds a `.engram/` directory but no vault ID ([792a2h Stable vault ID](../../specs/product/vault-identity.md#792a2h-stable-vault-id)).
- **Existing vault:** any other non-empty directory at the path, whether or not it has `.engram/`, a vault
  ID, or notes. This includes vaults created before this change.

Operations in this change, with the node each one changes:

| Operation | Type | Target |
| --- | --- | --- |
| [795a2 Init creates the vault](#795a2-added-init-creates-the-vault) | Added | none |
| [795a3 Init on an existing vault changes nothing](#795a3-added-init-on-an-existing-vault-changes-nothing) | Added | none |
| [795a4 Init repairs a half-created vault](#795a4-added-init-repairs-a-half-created-vault) | Added | none |
| [795a5 Commands that need a vault fail without one](#795a5-added-commands-that-need-a-vault-fail-without-one) | Added | none |
| [795a6 Update without a vault skips the vault steps](#795a6-added-update-without-a-vault-skips-the-vault-steps) | Added | none |
| [795a7 Vault creation happens only through engram init](#795a7-modified-vault-creation-happens-only-through-engram-init) | Modified | [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation) |
| [795a8 Interrupted creation is repaired only by engram init](#795a8-modified-interrupted-creation-is-repaired-only-by-engram-init) | Modified | [792a2g Interrupted creation is repaired](../../specs/product/vault-identity.md#792a2g-interrupted-creation-is-repaired) |
| [795a9 The memory-query intro no longer creates a missing vault](#795a9-modified-the-memory-query-intro-no-longer-creates-a-missing-vault) | Modified | [memory-query](../../specs/product/memory-query.md) |
| [795a10 The vault-counting intro no longer creates a missing vault](#795a10-modified-the-vault-counting-intro-no-longer-creates-a-missing-vault) | Modified | [vault-counting](../../specs/product/vault-counting.md) |
| [795a11 The vault-health intro no longer creates a missing vault](#795a11-modified-the-vault-health-intro-no-longer-creates-a-missing-vault) | Modified | [vault-health](../../specs/product/vault-health.md) |
| [795a12 Parent-sync vaults are created once with engram init](#795a12-modified-parent-sync-vaults-are-created-once-with-engram-init) | Modified | [parent-sync](../../specs/product/parent-sync.md) |
| [795a13 A read command creates the vault](#795a13-removed-a-read-command-creates-the-vault) | Removed | [792a2e1a A read command creates the vault](../../specs/product/vault-identity.md#792a2e1-uat-scenarios) |
| [795a14 amend and activate fail only on their own merits](#795a14-removed-amend-and-activate-fail-only-on-their-own-merits) | Removed | [792a2e1b amend and activate fail only on their own merits](../../specs/product/vault-identity.md#792a2e1-uat-scenarios) |
| [795a15 A child host gets its own vault](#795a15-removed-a-child-host-gets-its-own-vault) | Removed | [792a2e1c A child host gets its own vault](../../specs/product/vault-identity.md#792a2e1-uat-scenarios) |
| [795a16 A second run creates nothing](#795a16-removed-a-second-run-creates-nothing) | Removed | [792a2e1d A second run creates nothing](../../specs/product/vault-identity.md#792a2e1-uat-scenarios) |
| [795a17 A half-created vault gets its ID](#795a17-removed-a-half-created-vault-gets-its-id) | Removed | [792a2g1a A half-created vault gets its ID](../../specs/product/vault-identity.md#792a2g1-uat-scenarios) |

Nodes checked and left unchanged:

| Node | Why it needs no change |
| --- | --- |
| [792a2d Starter README](../../specs/product/vault-identity.md#792a2d-starter-readme) | The starter README's content and recognition belong to #796. |
| [792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone) | Still true: a read-only command on an existing vault creates and modifies nothing, and no command other than `engram init` prints a creation line. |
| [792a2h Stable vault ID](../../specs/product/vault-identity.md#792a2h-stable-vault-id) | A new vault still gets its ID when it is created; an existing vault without one is still stamped lazily. |
| [792a2h1b The first parent contact stamps an existing vault](../../specs/product/vault-identity.md#792a2h1-uat-scenarios) | Its "existing vault with no vault ID" is a vault without `.engram/`, which is an existing vault here. |
| [792a2k Exchange state outside tracked files](../../specs/product/vault-identity.md#792a2k-exchange-state-outside-tracked-files) | Where exchange state lives is unchanged. |
| [792a2l Inspecting the vault ID](../../specs/product/vault-identity.md#792a2l-inspecting-the-vault-id) | `engram vault-id` still reads an existing vault, including one with no vault ID and no `.engram/`. |
| [792a2n Claiming a moved vault](../../specs/product/vault-identity.md#792a2n-claiming-a-moved-vault) | `engram vault-id --claim` still works on an existing vault. |
| [792a21q1g The first offer creates the exchange state in the right order](../../specs/product/parent-sync.md#792a21q1-uat-scenarios) | Its "vault with no vault ID and no .engram directory" is an existing vault. |
| [chunk-indexing](../../specs/product/chunk-indexing.md) | `engram ingest` still resolves no vault to write to; [795a5c Ingest is unaffected](#795a5c-ingest-is-unaffected) makes that explicit. |
| [install-and-update](../../specs/product/install-and-update.md) | The install and deploy are unchanged; [795a6 Update without a vault skips the vault steps](#795a6-added-update-without-a-vault-skips-the-vault-steps) adds the no-vault case. Its scenario [792a11o1c Re-parenting skips the install and the deploy](../../specs/product/install-and-update.md#792a11o1-uat-scenarios) does not state a vault and holds on a machine with one. |
| [792a12d Accepting a registration creates a pending note](../../specs/product/skill-runbook-registration.md#792a12d-accepting-a-registration-creates-a-pending-note) | Registration through `engram register-skills` still creates a pending note in an existing vault; without a vault it fails under [795a5a A missing vault is an error](#795a5a-a-missing-vault-is-an-error). |
| [792a12t Update runs registration after deploying the skills](../../specs/product/skill-runbook-registration.md#792a12t-update-runs-registration-after-deploying-the-skills) | Update still runs registration against an existing vault; with none, [795a6a Update skips its vault steps](#795a6a-update-skips-its-vault-steps) skips it. |
| [write-memory](../../specs/product/write-memory.md), [learn](../../specs/product/learn.md) | The write-memory and learn skills call `engram learn` against an existing vault; without one they fail under [795a5a A missing vault is an error](#795a5a-a-missing-vault-is-an-error). |

## 795a2 Added: Init creates the vault

### 795a2a Init creates a new vault

Requirement: 795a2a When no vault exists at the resolved vault path ([792a2a Vault location](../../specs/product/vault-identity.md#792a2a-vault-location)), `engram init` MUST create
the vault described in [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation) there and print exactly one stdout line, ``engram: created new vault at <path>``,
whether or not `ENGRAM_PARENT` is set.

#### 795a2a1 UAT Scenarios

```gherkin
Scenario: 795a2a1a Init creates a vault at an absent path
  Given no file or directory exists at /v
  When the user runs engram init --vault /v
  Then stdout is exactly "engram: created new vault at /v", the exit status is zero, and /v holds .gitignore, README.md, .engram-vault-id and .engram/ and no .obsidian/

Scenario: 795a2a1b Init uses the resolved location
  Given ENGRAM_VAULT_PATH is /env/vault, no --vault flag is given, and no file or directory exists at /env/vault
  When the user runs engram init
  Then a vault is created at /env/vault

Scenario: 795a2a1c A child host creates its own local vault
  Given ENGRAM_PARENT is set and no vault exists at the resolved local path
  When the user runs engram init
  Then a vault is created at the resolved local path and stdout is the created line naming it
```

### 795a2b Absent or empty means no vault

Requirement: 795a2b `engram init` MUST treat a resolved path that is an empty directory as holding no vault,
and create the vault in it.

#### 795a2b1 UAT Scenarios

```gherkin
Scenario: 795a2b1a Init creates a vault in an empty directory
  Given /v is an existing empty directory
  When the user runs engram init --vault /v
  Then stdout is exactly "engram: created new vault at /v" and /v holds .gitignore, README.md, .engram-vault-id and .engram/
```

Rationale: [792a2a Vault location](../../specs/product/vault-identity.md#792a2a-vault-location) already lets a user point engram at a directory they made themselves, and an empty
directory has nothing to protect, so refusing it would only force a `rmdir`.

## 795a3 Added: Init on an existing vault changes nothing

### 795a3a Existing vaults are left alone by init

Requirement: 795a3a When a vault already exists at the resolved path, `engram init` MUST change nothing in
it and MUST print exactly one stdout line, ``engram: vault at <path> already exists; nothing changed``. It
MUST NOT add or remove an `.obsidian/` directory.

#### 795a3a1 UAT Scenarios

```gherkin
Scenario: 795a3a1a A second init changes nothing
  Given a vault that engram init has just created
  When the user runs engram init again
  Then stdout is exactly "engram: vault at <path> already exists; nothing changed", the exit status is zero, and every file in the vault, including .engram-vault-id, is byte-identical

Scenario: 795a3a1b A vault made before this change is not stamped
  Given a vault with notes, no .engram/ and no .engram-vault-id
  When the user runs engram init
  Then stdout is the already-exists line, and no .engram/, .engram-vault-id, .gitignore or README.md is created or modified

Scenario: 795a3a1c An existing .obsidian directory stays
  Given an existing vault that holds .obsidian/app.json
  When the user runs engram init
  Then .obsidian/app.json is byte-identical and no other file in the vault changed
```

Rationale: a deliberate command run against a vault that is already there should be harmless to repeat, so
a user can run it as a check. A directory with notes but no `.engram/` is a vault from before explicit
creation ([792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone)), so init treats it as existing rather than stamping tracked files into a git-backed
vault.

## 795a4 Added: Init repairs a half-created vault

### 795a4a Init stamps the missing vault ID

Requirement: 795a4a When a vault has the `.engram/` state directory but no vault ID, `engram init` MUST give
it a vault ID and print exactly one stdout line, ``engram: stamped the missing vault ID of <path>``.

#### 795a4a1 UAT Scenarios

```gherkin
Scenario: 795a4a1a Init gives a half-created vault its ID
  Given a vault with a .engram/ directory and no .engram-vault-id
  When the user runs engram init
  Then .engram-vault-id is created, stdout is exactly "engram: stamped the missing vault ID of <path>", and the exit status is zero

Scenario: 795a4a1b A repaired vault is usable
  Given a half-created vault that engram init has repaired
  When the user runs engram query --phrase x
  Then the query returns an empty result without error
```

Rationale: this moves the repair of [792a2g Interrupted creation is repaired](../../specs/product/vault-identity.md#792a2g-interrupted-creation-is-repaired) from every command to the one command whose job is to fix
the vault, with the same line and the same trigger.

## 795a5 Added: Commands that need a vault fail without one

### 795a5a A missing vault is an error

Requirement: 795a5a Every engram command other than `engram init` that resolves a vault path MUST, when no
vault exists at that path, exit non-zero before doing its work, create nothing, and print to stderr
``engram: no vault at <path>; run `engram init` to create one, or use --vault or ENGRAM_VAULT_PATH to point at an existing vault``.

#### 795a5a1 UAT Scenarios

```gherkin
Scenario: 795a5a1a A read command no longer creates the vault
  Given no file or directory exists at /v
  When the user runs engram query --phrase x --vault /v
  Then it exits non-zero, stderr is exactly "engram: no vault at /v; run `engram init` to create one, or use --vault or ENGRAM_VAULT_PATH to point at an existing vault", and /v still does not exist

Scenario: 795a5a1b A write command fails the same way
  Given no vault exists at the resolved path
  When the user runs engram amend or engram activate on any note
  Then it exits non-zero with the no-vault error, never a vault lock error

Scenario: 795a5a1c A child host without a local vault fails
  Given ENGRAM_PARENT is set and no vault exists at the resolved local path
  When the user runs engram query --phrase x
  Then it exits non-zero with the no-vault error and creates no local vault

Scenario: 795a5a1d An empty directory is left empty
  Given /v is an existing empty directory
  When the user runs engram count --vault /v
  Then it exits non-zero with the no-vault error and /v is still empty
```

### 795a5b A half-created vault is an error

Requirement: 795a5b Every engram command other than `engram init` that resolves a vault path MUST, when the
vault there is half-created, exit non-zero before doing its work, change nothing, and print to stderr
``engram: the vault at <path> is incomplete; run `engram init` to repair it, or use --vault or ENGRAM_VAULT_PATH to point at another vault``.

#### 795a5b1 UAT Scenarios

```gherkin
Scenario: 795a5b1a Another command does not repair
  Given a vault with a .engram/ directory and no .engram-vault-id
  When the user runs engram query --phrase x
  Then it exits non-zero, stderr is exactly "engram: the vault at <path> is incomplete; run `engram init` to repair it, or use --vault or ENGRAM_VAULT_PATH to point at another vault", and .engram-vault-id is not created
```

### 795a5c Ingest is unaffected

Requirement: 795a5c `engram ingest` MUST behave as before when no vault exists or the vault is
half-created: it MUST NOT create or repair a vault, and MUST NOT fail or print a vault error because of the
vault's state.

#### 795a5c1 UAT Scenarios

```gherkin
Scenario: 795a5c1a Ingest without a vault
  Given no vault exists at the resolved path and a Markdown file notes.md
  When the user runs engram ingest --markdown notes.md
  Then the chunks of notes.md are indexed, the exit status is zero, no vault error is printed, and the vault path still does not exist

Scenario: 795a5c1b Ingest with a half-created vault
  Given a vault with a .engram/ directory and no .engram-vault-id, and a Markdown file notes.md
  When the user runs engram ingest --markdown notes.md
  Then the chunks of notes.md are indexed, the exit status is zero, and .engram-vault-id is not created
```

Rationale: the first-use creation of [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation) ran before every vault-resolving command, so each of them
would now turn a missing vault into an error. Ingest needs no vault note, and the issue keeps it working
before a vault exists. The error names all three ways out (`engram init`, `--vault`, `ENGRAM_VAULT_PATH`),
so a mistyped path is recognized as such instead of being answered with a new vault.

## 795a6 Added: Update without a vault skips the vault steps

### 795a6a Update skips its vault steps

Requirement: 795a6a When no vault exists at the resolved path, or the vault there is half-created,
`engram update` MUST skip every step that reads or writes the vault, namely the vault notices, skill
registration ([792a12t Update runs registration after deploying the skills](../../specs/product/skill-runbook-registration.md#792a12t-update-runs-registration-after-deploying-the-skills)), `--regen-vocab`, `--backfill-identity` and `--reparent-luhmann`, and MUST print
in its report ``engram update: no vault at <path>; skipped the vault steps. Run `engram init` to create one.``
when there is no vault, or ``engram update: the vault at <path> is incomplete; skipped the vault steps. Run `engram init` to repair it.``
when it is half-created.

#### 795a6a1 UAT Scenarios

```gherkin
Scenario: 795a6a1a Update with no vault
  Given no vault exists at the resolved path
  When the user runs engram update
  Then the report contains the line "engram update: no vault at <path>; skipped the vault steps. Run `engram init` to create one." and no registration offer or vault notice

Scenario: 795a6a1b Update with a half-created vault
  Given a vault with a .engram/ directory and no .engram-vault-id
  When the user runs engram update
  Then the report contains the line "engram update: the vault at <path> is incomplete; skipped the vault steps. Run `engram init` to repair it."

Scenario: 795a6a1c A vault option is skipped too
  Given no vault exists at the resolved path
  When the user runs engram update --backfill-identity
  Then no note is stamped, and the report contains the no-vault line

Scenario: 795a6a1d Re-parenting is skipped
  Given no vault exists at the resolved path
  When the user runs engram update --reparent-luhmann
  Then nothing is installed or deployed, no candidate is derived, and the report contains the no-vault line
```

### 795a6b Update never creates or repairs a vault

Requirement: 795a6b `engram update` MUST NOT create or repair a vault.

#### 795a6b1 UAT Scenarios

```gherkin
Scenario: 795a6b1a Update leaves the vault path as it found it
  Given no vault exists at the resolved path, or a half-created vault does
  When the user runs engram update
  Then afterwards the path is still absent, or still has no .engram-vault-id
```

### 795a6c The rest of update still runs

Requirement: 795a6c When it skips the vault steps, `engram update` MUST still install, deploy and report as
it does with a vault, and its exit status MUST NOT be affected by the skip.

#### 795a6c1 UAT Scenarios

```gherkin
Scenario: 795a6c1a The deploy proceeds without a vault
  Given no vault exists at the resolved path and a harness with skills to deploy
  When the user runs engram update
  Then the skills are deployed, the report holds the no-vault line, and the exit status is zero
```

Rationale: a fresh machine usually runs `engram update` before it ever runs `engram init`, so update must
keep working there, and it tells the user the next step instead of failing or creating the vault itself.
Its notices join the report the other update notices already use ([install-and-update](../../specs/product/install-and-update.md)).

## 795a7 Modified: Vault creation happens only through engram init

Target: [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation). Complete replacement text, keeping the target ID. The nodes below it are the UAT
scenarios that verify it.

> Requirement: 792a2e A vault MUST be created only by `engram init` ([795a2a Init creates a new vault](#795a2a-init-creates-a-new-vault)). A new vault contains a
> `.gitignore`, a starter `README.md`, a vault ID ([792a2h Stable vault ID](../../specs/product/vault-identity.md#792a2h-stable-vault-id)) and the `.engram/` state directory
> ([792a2k Exchange state outside tracked files](../../specs/product/vault-identity.md#792a2k-exchange-state-outside-tracked-files)), and MUST NOT contain an `.obsidian/` directory. No other command MUST create a vault,
> whether or not `ENGRAM_PARENT` is set.

UAT scenarios: [795a2a1a Init creates a vault at an absent path](#795a2a1-uat-scenarios) (contents), [795a2a1c A child host creates its own local vault](#795a2a1-uat-scenarios) (child host), [795a5a1a A read command no longer creates the vault](#795a5a1-uat-scenarios) (no other command creates).
[795a2a1a Init creates a vault at an absent path](#795a2a1-uat-scenarios) also verifies that a new vault has no `.obsidian/`, and [795a3a1c An existing .obsidian directory stays](#795a3a1-uat-scenarios) that an existing one is left untouched.

Rationale: Joe replaced creation by any command (#766) with explicit creation, so a mistyped
`--vault` or `ENGRAM_VAULT_PATH` is an error and not a second vault. The `.obsidian/` starter is dropped
because not every vault is opened in Obsidian; a user who wants one adds it.

## 795a8 Modified: Interrupted creation is repaired only by engram init

Target: [792a2g Interrupted creation is repaired](../../specs/product/vault-identity.md#792a2g-interrupted-creation-is-repaired). Complete replacement text, keeping the target ID.

> Requirement: 792a2g When a vault has the `.engram/` state directory but no vault ID, because its creation
> stopped part-way, only `engram init` MUST repair it, by giving it a vault ID ([795a4a Init stamps the missing vault ID](#795a4a-init-stamps-the-missing-vault-id)). Every other
> command that resolves it MUST fail with the error of [795a5b A half-created vault is an error](#795a5b-a-half-created-vault-is-an-error) and MUST NOT change it.
>
> Rationale: creation writes `.engram/` before the ID, so `.engram/` marks a vault engram created, and a
> pre-existing vault with neither stays untouched ([792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone)).

UAT scenarios: [795a4a1a Init gives a half-created vault its ID](#795a4a1-uat-scenarios) (init repairs), [795a5b1a Another command does not repair](#795a5b1-uat-scenarios) (another command does not).

Rationale: the repair moves from every command to `engram init` as decided in #795;
the trigger and the stamped line are kept, so only the command that runs them changes.

## 795a9 Modified: The memory-query intro no longer creates a missing vault

Target: [memory-query](../../specs/product/memory-query.md). Complete replacement of the section's introductory paragraph; the heading, the canonical
target line and every child node are unchanged.

> `engram query` searches the vault's notes and the chunk index for what is relevant to a set of
> phrases and prints one YAML payload: the matched notes and chunks, the clusters they fall into, the
> newest chunks, and a budget of counts. `engram show` prints one note in full, `engram show-chunk`
> prints one chunk's text, and `engram query-chunks` searches the chunk index alone. All four only
> read. The `recall` skill is the main caller and decides what to do with the payload. A query on a
> vault that does not exist fails with the no-vault error and creates nothing ([795a5a A missing vault is an error](#795a5a-a-missing-vault-is-an-error)). The payload's
> `pending_offers` flag and the exclusion of pending offers from results belong to curate ([curate](../../specs/product/curate.md)), results
> merged in from a parent vault belong to parent-sync ([parent-sync](../../specs/product/parent-sync.md)), and the served form of a query belongs to
> vault-serving ([792a20m A served query takes the message and can return dedupe keys](../../specs/product/vault-serving.md#792a20m-a-served-query-takes-the-message-and-can-return-dedupe-keys)). Which kinds of note a query returns, and that it never returns a question note, is
> note-writing's [792a3t Question-and-answer retrieval](../../specs/product/note-writing.md#792a3t-question-and-answer-retrieval).

Rationale: the intro stated first-use creation as a fact about `engram query`; with creation gone, it must
state the failure instead. [795a5a1a A read command no longer creates the vault](#795a5a1-uat-scenarios) verifies it.

## 795a10 Modified: The vault-counting intro no longer creates a missing vault

Target: [vault-counting](../../specs/product/vault-counting.md). Complete replacement of the section's introductory paragraph; everything else is
unchanged.

> `engram count` answers "how many?" over the whole vault: how many notes carry each value of a
> frontmatter attribute, and how many notes link to a given note. It reads only the Markdown notes at
> the vault's root and changes nothing. It runs against the vault located by [792a2a Vault location](../../specs/product/vault-identity.md#792a2a-vault-location), and on a vault that
> does not exist it fails with the no-vault error ([795a5a A missing vault is an error](#795a5a-a-missing-vault-is-an-error)). `engram query` answers a different question, "what is
> relevant to this phrase", and returns a ranked, bounded sample of the vault ([memory-query](../../specs/product/memory-query.md)). The `route`
> skill uses `count` to audit the evidence tallies it reports.

Rationale: same reason as [795a9 The memory-query intro no longer creates a missing vault](#795a9-modified-the-memory-query-intro-no-longer-creates-a-missing-vault). [795a5a1d An empty directory is left empty](#795a5a1-uat-scenarios) verifies it for `engram count`.

## 795a11 Modified: The vault-health intro no longer creates a missing vault

Target: [vault-health](../../specs/product/vault-health.md). Complete replacement of the section's introductory paragraph; everything else is
unchanged.

> A vault is a directory of notes and a sidecar beside each note holding its embedding ([792a3n Embed on write](../../specs/product/note-writing.md#792a3n-embed-on-write)). Two
> commands keep it healthy. `engram check` reads the vault and reports where it breaks its
> invariants: links that cannot resolve, notes recall cannot match, and sidecars that cannot be
> read. `engram embed status` and `engram embed apply` count and repair the sidecars. All three read
> only the Markdown notes at the vault's root, run against the vault located by [792a2a Vault location](../../specs/product/vault-identity.md#792a2a-vault-location), and fail with the
> no-vault error on a vault that does not exist ([795a5a A missing vault is an error](#795a5a-a-missing-vault-is-an-error)). The model change that makes an embed run necessary is the
> embedding-sidecar's [792b10c7 State classification](../../specs/technical/embedding-sidecar.md#792b10c7-state-classification).

Rationale: same reason as [795a9 The memory-query intro no longer creates a missing vault](#795a9-modified-the-memory-query-intro-no-longer-creates-a-missing-vault). [795a5a1a A read command no longer creates the vault](#795a5a1-uat-scenarios) and [795a5a1d An empty directory is left empty](#795a5a1-uat-scenarios) verify the failure.

## 795a12 Modified: Parent-sync vaults are created once with engram init

Target: [parent-sync](../../specs/product/parent-sync.md). Complete replacement of the section's introductory paragraph; the heading, the canonical
target line and every child node are unchanged.

> A vault can have one parent: another engram vault, served with `engram serve` ([vault-serving](../../specs/product/vault-serving.md)), that sits above it in a tree
> (personal, then team, then organization) and never in a mesh. `ENGRAM_PARENT` names it. Every environment keeps its own
> local vault whether or not it has a parent, created once with `engram init` ([795a2a Init creates a new vault](#795a2a-init-creates-a-new-vault)), so a command works with the parent unreachable, and the two vaults exchange
> notes in both directions. A node offers what it writes upward, through an outbox that survives being offline. It reads
> its parent's notes back in one merged answer to `engram query`, and pulls a parent's note down when it uses it, as an
> offer for local curation to judge ([curate](../../specs/product/curate.md)). Notes are exchanged and transcript chunks never are. How a vault tells a
> clone or a copy of itself from its parent, and what an exchange is allowed to touch, are vault-identity ([vault-identity](../../specs/product/vault-identity.md)), and the
> served side of the exchange is vault-serving ([vault-serving](../../specs/product/vault-serving.md)). The `ENGRAM_SERVER` thin-client mode that the exchange replaced
> is gone, and [792a21a ENGRAM_SERVER is refused](../../specs/product/parent-sync.md#792a21a-engram_server-is-refused) makes setting it an error.

Rationale: a child host no longer gets its local vault as a side effect of its first command, so the intro
must say where it comes from. [795a2a1c A child host creates its own local vault](#795a2a1-uat-scenarios) and [795a5a1c A child host without a local vault fails](#795a5a1-uat-scenarios) verify both sides.

## 795a13 Removed: A read command creates the vault

Target: [792a2e1a A read command creates the vault](../../specs/product/vault-identity.md#792a2e1-uat-scenarios).

Replacement coverage: [795a5a1a A read command no longer creates the vault](#795a5a1-uat-scenarios) (a query on a missing vault now fails and creates nothing) and
[795a2a1a Init creates a vault at an absent path](#795a2a1-uat-scenarios) (init creates the vault).

Rationale: the scenario asserts creation by a read command, which [795a7 Vault creation happens only through engram init](#795a7-modified-vault-creation-happens-only-through-engram-init) forbids.

## 795a14 Removed: amend and activate fail only on their own merits

Target: [792a2e1b amend and activate fail only on their own merits](../../specs/product/vault-identity.md#792a2e1-uat-scenarios).

Replacement coverage: [795a5a1b A write command fails the same way](#795a5a1-uat-scenarios) (amend and activate on a missing vault fail with the no-vault error,
not a vault lock error).

Rationale: its premise, that the vault is created first so the command can fail on its target, no longer
holds.

## 795a15 Removed: A child host gets its own vault

Target: [792a2e1c A child host gets its own vault](../../specs/product/vault-identity.md#792a2e1-uat-scenarios).

Replacement coverage: [795a2a1c A child host creates its own local vault](#795a2a1-uat-scenarios) (init creates the child host's local vault) and [795a5a1c A child host without a local vault fails](#795a5a1-uat-scenarios) (a query
without it fails).

Rationale: a child host now creates its local vault with `engram init`, not with its first query.

## 795a16 Removed: A second run creates nothing

Target: [792a2e1d A second run creates nothing](../../specs/product/vault-identity.md#792a2e1-uat-scenarios).

Replacement coverage: [795a3a1a A second init changes nothing](#795a3a1-uat-scenarios) (a second `engram init` changes nothing and says so).

Rationale: "a second run" now means a second `engram init`; any other command never creates anything.

## 795a17 Removed: A half-created vault gets its ID

Target: [792a2g1a A half-created vault gets its ID](../../specs/product/vault-identity.md#792a2g1-uat-scenarios).

Replacement coverage: [795a4a1a Init gives a half-created vault its ID](#795a4a1-uat-scenarios) (init stamps the ID) and [795a5b1a Another command does not repair](#795a5b1-uat-scenarios) (another command does not).

Rationale: the scenario asserts repair by any command, which [795a8 Interrupted creation is repaired only by engram init](#795a8-modified-interrupted-creation-is-repaired-only-by-engram-init) restricts to `engram init`.
