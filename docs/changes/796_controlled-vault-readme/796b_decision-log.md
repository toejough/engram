# Controlled Vault README - Decision Log

These are choices made while drafting [796a Controlled Vault README - Product Spec](./796a_product-spec.md) (#796), recorded for Joe to
validate. Questions that need Joe's input are in [`796c_open-questions.md`](./796c_open-questions.md).
Issue #796 holds the binding decisions; the entries below fill the gaps it leaves.

## 796b1 Writing and read-only commands

- **Choice:** a command is writing when the invocation changes any file in the vault. Writing: `learn`,
  `amend`, `resituate`, `activate`, `embed apply`, the `vocab` subcommands that write,
  `register-skills` accept and decline, `update` with vault flags or registration writes, `serve`, and
  `vault-id --regenerate` and `--claim`. Read-only: `query`, `show`, `count`, `check`, `embed status`,
  `vault-id` without a changing flag, and any `--dry-run`. Provisional.
- **Rationale:** the issue says "a command that writes to the vault" and cites [792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone)
  for read-only commands. Defining it by the effect, not by a command list alone, keeps new commands covered.
- **Alternatives:** restore on every command. Rejected: it breaks [792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone).
  Restore only on `learn`. Rejected: other writers would leave the file stale.

## 796b2 Restore just before the first write, once per invocation

- **Choice:** the restore happens just before the command's first write to the vault, and a command that
  makes no write does not restore. `serve` counts as one invocation. Provisional.
- **Rationale:** a command that fails before writing, such as an amend of a missing note, changed nothing
  and should leave the README alone.
- **Alternatives:** restore at command start. Rejected: it would restore for commands that then write nothing.

## 796b3 Exact warning text

- **Choice:** one stderr line, `engram: restored the engram-controlled README at <path>; put anything you
  want remembered in a regular note`.
- **Rationale:** it follows the `engram: ...` format of the creation lines ([792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation)), says what
  happened, and points to the issue's own guidance that anything to remember goes in a regular note.
- **Alternatives:** a bare "README restored" line. Rejected: it does not say where the edit should go.

## 796b4 792a2d keeps its title

- **Choice:** the replacement keeps the heading title "Starter README".
- **Rationale:** references to [792a2d Starter README](../../specs/product/vault-identity.md#792a2d-starter-readme) elsewhere carry that label, and a rename would need all of them to change.
- **Alternatives:** retitle it "Controlled vault README". Rejected for the churn.

## 796b5 The 792a2d1a scenario is kept

- **Choice:** [792a2d1a A fresh vault has no notes](../../specs/product/vault-identity.md#792a2d1-uat-scenarios) stays unchanged and is restated in the replacement.
- **Rationale:** a new vault still reports 0 notes.

## 796b6 Technical spec follow-up

- **Choice:** no technical spec or plan is drafted. [792b3c4 Vault Markdown listing](../../specs/technical/durable-file-store.md#792b3c4-vault-markdown-listing) lists an edited README as a note and
  must change with this behavior.
- **Rationale:** the task scopes this change to the product spec.
