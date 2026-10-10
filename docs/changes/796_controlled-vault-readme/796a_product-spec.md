# Controlled Vault README - Product Spec

## 796a1 Problem Statement

Engram users, and the agents that work in their vaults, can edit the root `README.md` of a vault.
Today ([792a2d Starter README](../../specs/product/vault-identity.md#792a2d-starter-readme)) engram skips only the untouched starter README and treats an edited or replaced one as a
note. A user who writes in the README therefore gets a note that is counted, embedded, checked and
recalled, although the file was never written as a note. A user who wants to remember something, or to
change how engram behaves, has no reason to use the README for it: regular notes do that.

The needed outcome (issue #796, decided with Joe on 2026-10-10): the root `README.md` is an
engram-controlled artifact and is never a note. When a command that writes to the vault finds it
edited, engram restores its own content and prints a stderr warning. Read-only commands leave it alone,
per [792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone).

Terms:

- **Starter README:** the `README.md` content engram ships and writes into a vault.
- **Writing command:** an engram invocation that changes any file in the vault. Provisionally these are
  `learn`, `amend`, `resituate`, `activate`, `embed apply`, the `vocab` subcommands that write,
  `register-skills` accept and decline, `update` when it writes vault files, `serve`, and
  `vault-id --regenerate` and `--claim` (see [796b1 Writing and read-only commands](./796b_decision-log.md#796b1-writing-and-read-only-commands)).
- **Read-only command:** every other invocation: `query`, `show`, `count`, `check`, `embed status`,
  `vault-id` without a changing flag, and any run with `--dry-run`.

Vault creation, including the explicit `engram init`, is a separate change (#795,
[795a Vault Init Command - Product Spec](../795_vault-init-command/795a_product-spec.md)). This change
covers only how the README is treated once a vault exists.

### 796a1a Requirements checked

Every requirement or scenario in the [canonical product specs](../../specs/product/) that
counts notes, reads the README or depends on it being a note was checked.

| Target | Result |
| --- | --- |
| [792a2d Starter README](../../specs/product/vault-identity.md#792a2d-starter-readme) | Modified by [796a2 vault-identity (controlled vault README)](#796a2-modified-vault-identity-controlled-vault-readme) |
| [792a2d1b A user's README is a note](../../specs/product/vault-identity.md#792a2d1-uat-scenarios) | Removed by [796a3 vault-identity (a user's README is a note)](#796a3-removed-vault-identity-a-users-readme-is-a-note) |
| [792a2d1a A fresh vault has no notes](../../specs/product/vault-identity.md#792a2d1-uat-scenarios) | Unchanged: a new vault still reports 0 notes |
| [792a2c Flat vault layout](../../specs/product/vault-identity.md#792a2c-flat-vault-layout) | Unchanged: it concerns subdirectories only |
| [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation) | Unchanged: it writes the starter README at creation; #795 changes creation |
| [792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone) | Unchanged: read-only commands already leave every file alone |
| [792a8h A count changes nothing and is exact](../../specs/product/vault-counting.md#792a8h-a-count-changes-nothing-and-is-exact) | Unchanged: "every note at the vault root" excludes the README by [792a2d Starter README](../../specs/product/vault-identity.md#792a2d-starter-readme) |
| [792a9f A check changes nothing](../../specs/product/vault-health.md#792a9f-a-check-changes-nothing) | Unchanged: a check reports on notes only and stays read-only |
| [792a9g The embedding census](../../specs/product/vault-health.md#792a9g-the-embedding-census) | Unchanged: the census counts notes, and the README is not one |
| [792a9h Which notes get embedded](../../specs/product/vault-health.md#792a9h-which-notes-get-embedded) | Unchanged: the README is never selected, as it is not a note |
| [792a10a The flat-vault notice](../../specs/product/vault-migration.md#792a10a-the-flat-vault-notice) | Unchanged: it tests notes only |
| [vault-serving](../../specs/product/vault-serving.md) and [parent-sync](../../specs/product/parent-sync.md) | Unchanged: they exchange notes, and the README is not one |

The technical spec for this change, not yet drafted, must also revise
[792b3c4 Vault Markdown listing](../../specs/technical/durable-file-store.md#792b3c4-vault-markdown-listing), which today lists an edited README as a note.

## 796a2 Modified: vault-identity (controlled vault README)

Canonical target: `docs/specs/product/vault-identity.md`.

Target: [792a2d Starter README](../../specs/product/vault-identity.md#792a2d-starter-readme). The complete replacement text follows; the ID 792a2d
is preserved.

### 792a2d Starter README

Requirement: 792a2d Starter README Engram MUST NOT treat the root `README.md` of a vault as a note, whatever its
content. A writing command MUST, when the root `README.md` is present and its content differs from the
starter README, restore the starter README just before the command's first write to the vault, and MUST
print exactly one stderr line, `engram: restored the engram-controlled README at <path>; put anything you
want remembered in a regular note`, where `<path>` is the README's path. A writing command MUST NOT
restore the README or print that line when the content already equals the starter README, or when it makes no
write. A read-only command MUST NOT create, modify or delete the README.

#### 792a2d1 UAT Scenarios

```gherkin
Scenario: 792a2d1a A fresh vault has no notes
  Given a vault that engram has just created
  When the user runs engram embed status and engram check on it
  Then embed status reports a total of 0 notes and check reports no note missing a sidecar
```

Rationale: the README is how engram introduces itself to anyone who opens the vault, so its content is
engram's to define. Anything a user wants engram to remember, or wants to change in how engram behaves,
belongs in a regular note, which recall, check and the embedder already handle. Restoring at the first
write, and only then, fixes the file at the moment engram is already changing the vault, and keeps
read-only use free of side effects ([792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone)).
The warning tells the user why their edit vanished and where to put it instead. A missing README is not
restored here ([796c1 Should a missing README be restored?](./796c_open-questions.md#796c1-should-a-missing-readme-be-restored)).

#### 796a2a UAT Scenarios

```gherkin
Scenario: 796a2a1 An edited README is not a note
  Given a vault with no notes whose README.md the user has replaced with their own text
  When the user runs engram embed status and engram check on it
  Then embed status reports a total of 0 notes and check reports no note missing a sidecar

Scenario: 796a2a2 An empty README is not a note
  Given a vault with one note whose README.md the user has emptied
  When the user runs engram embed status
  Then it reports a total of 1 note

Scenario: 796a2a3 A write restores an edited README and warns
  Given a vault whose README.md the user has replaced with their own text
  When the user runs engram learn to write a note
  Then the note is written, README.md holds the starter README again, and stderr carries exactly one line,
    "engram: restored the engram-controlled README at <path>; put anything you want remembered in a regular note"

Scenario Outline: 796a2a4 Each writing command restores the README
  Given a vault with a note whose README.md the user has replaced with their own text
  When the user runs <command> so that it changes the vault
  Then README.md holds the starter README again and stderr carries the restore line once
  Examples:
    | command                      |
    | engram amend                 |
    | engram embed apply           |
    | engram vault-id --regenerate |

Scenario: 796a2a5 An unedited README is left as it is
  Given a vault whose README.md equals the starter README
  When the user runs engram learn to write a note
  Then README.md is unchanged and stderr carries no restore line

Scenario: 796a2a6 A write that never happens does not restore
  Given a vault whose README.md the user has replaced with their own text
  When the user runs engram amend on a note that does not exist
  Then the command fails because the note is not found, README.md still holds the user's text, and stderr carries no restore line

Scenario: 796a2a7 Read-only commands leave an edited README alone
  Given a vault whose README.md the user has replaced with their own text
  When the user runs engram query, show, count, check, embed status and vault-id without a changing flag
  Then README.md still holds the user's text and no restore line is printed

Scenario: 796a2a8 A dry run leaves an edited README alone
  Given a vault whose README.md the user has replaced with their own text
  When the user runs a writing command with --dry-run
  Then README.md still holds the user's text and no restore line is printed
```

## 796a3 Removed: vault-identity (a user's README is a note)

Canonical target: `docs/specs/product/vault-identity.md`.

Target: [792a2d1b A user's README is a note](../../specs/product/vault-identity.md#792a2d1-uat-scenarios). The scenario is removed, and its ID
792a2d1b MUST NOT be reused.

Rationale: the scenario asserts that a user-edited README counts as a note, which [796a2 vault-identity (controlled vault README)](#796a2-modified-vault-identity-controlled-vault-readme)
reverses. [796a2a1 An edited README is not a note](#796a2a-uat-scenarios) covers the opposite outcome.
