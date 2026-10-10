# Vault Init Command - Decision Log

These are choices made while drafting the product spec of #795 ([795a Vault Init Command - Product Spec](./795a_product-spec.md#vault-init-command---product-spec)), recorded for Joe to validate.
Nothing here needs an answer before the spec is usable; each default stands unless Joe says otherwise.

## 795b1 Modify the creation and repair requirements, remove only their scenarios

- **Choice:** [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation) and [792a2g Interrupted creation is repaired](../../specs/product/vault-identity.md#792a2g-interrupted-creation-is-repaired) are Modified, keeping their IDs ([795a7 Vault creation happens only through engram init](./795a_product-spec.md#795a7-modified-vault-creation-happens-only-through-engram-init), [795a8 Interrupted creation is repaired only by engram init](./795a_product-spec.md#795a8-modified-interrupted-creation-is-repaired-only-by-engram-init)). Only the five
  scenarios that assert creation or repair by any command are Removed ([795a13 A read command creates the vault](./795a_product-spec.md#795a13-removed-a-read-command-creates-the-vault) to [795a17 A half-created vault gets its ID](./795a_product-spec.md#795a17-removed-a-half-created-vault-gets-its-id)).
- **Rationale:** the lifecycle guide treats a changed rule as a modification and a rename as a modification,
  not a removal plus an addition. The requirements still describe creation and repair; only who performs
  them changes.
- **Alternatives:** remove both requirements and add new ones. Rejected: it would retire IDs that other
  nodes link to for no gain.

## 795b2 Contents and exclusivity stay with the creation requirement; the command is separate

- **Choice:** [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation) keeps what a new vault holds and that only `engram init` creates it. [795a2 Init creates the vault](./795a_product-spec.md#795a2-added-init-creates-the-vault) holds
  what the command prints and where it runs.
- **Rationale:** each rule lives in one place, so neither requirement restates the other.
- **Alternatives:** put everything in the Modified text. Rejected: the command's messages and the
  empty-directory rule would hide inside a replaced paragraph.

## 795b3 Modified nodes cite scenarios instead of carrying their own

- **Choice:** the Modified operations for [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation) and [792a2g Interrupted creation is repaired](../../specs/product/vault-identity.md#792a2g-interrupted-creation-is-repaired) list the Added scenarios that verify
  them, and add none of their own.
- **Rationale:** the same user-visible behavior would otherwise be written twice with two IDs.
- **Alternatives:** duplicate the scenarios under the Modified operations. Rejected for the same reason.

## 795b4 An absent or empty directory is "no vault" (provisional)

- **Choice:** [795a2b Absent or empty means no vault](./795a_product-spec.md#795a2b-absent-or-empty-means-no-vault) lets `engram init` create a vault in an empty directory, and the missing-vault
  error ([795a5a A missing vault is an error](./795a_product-spec.md#795a5a-a-missing-vault-is-an-error)) applies to an empty directory too.
- **Rationale:** an empty directory has nothing to protect, and a user often makes one before pointing
  engram at it.
- **Alternatives:** refuse any existing directory. Rejected: it forces a `rmdir` first. Provisional until
  Joe confirms.

## 795b5 A non-empty directory without .engram/ is an existing vault (provisional)

- **Choice:** [795a3a Existing vaults are left alone by init](./795a_product-spec.md#795a3a-existing-vaults-are-left-alone-by-init) makes `engram init` a no-op on it. It stamps no ID and creates no `.engram/`,
  `.gitignore` or README.
- **Rationale:** it is a vault made before explicit creation, and [792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone) already keeps such vaults
  untouched so a git-backed vault is not dirtied. Reporting "nothing changed" matches the issue.
- **Alternatives:** have init adopt it by stamping the ID. Rejected: the issue says init on an existing
  vault reports that nothing changed. Provisional until Joe confirms.

## 795b6 Message texts and streams (provisional)

- **Choice:** init's results go to stdout, because they are the command's output; errors and the update
  notice follow their existing channels, stderr for errors and the update report for the notice.

| Where | Text |
| --- | --- |
| Created ([795a2a Init creates a new vault](./795a_product-spec.md#795a2a-init-creates-a-new-vault)) | ``engram: created new vault at <path>`` (the text the old creation line used) |
| Already there ([795a3a Existing vaults are left alone by init](./795a_product-spec.md#795a3a-existing-vaults-are-left-alone-by-init)) | ``engram: vault at <path> already exists; nothing changed`` |
| Repaired ([795a4a Init stamps the missing vault ID](./795a_product-spec.md#795a4a-init-stamps-the-missing-vault-id)) | ``engram: stamped the missing vault ID of <path>`` (the old repair line) |
| No vault ([795a5a A missing vault is an error](./795a_product-spec.md#795a5a-a-missing-vault-is-an-error)) | ``engram: no vault at <path>; run `engram init` to create one, or use --vault or ENGRAM_VAULT_PATH to point at an existing vault`` |
| Half-created ([795a5b A half-created vault is an error](./795a_product-spec.md#795a5b-a-half-created-vault-is-an-error)) | ``engram: the vault at <path> is incomplete; run `engram init` to repair it, or use --vault or ENGRAM_VAULT_PATH to point at another vault`` |
| Update, no vault ([795a6a Update skips its vault steps](./795a_product-spec.md#795a6a-update-skips-its-vault-steps)) | ``engram update: no vault at <path>; skipped the vault steps. Run `engram init` to create one.`` |
| Update, half-created ([795a6a Update skips its vault steps](./795a_product-spec.md#795a6a-update-skips-its-vault-steps)) | ``engram update: the vault at <path> is incomplete; skipped the vault steps. Run `engram init` to repair it.`` |

- **Rationale:** reusing the old creation and repair lines keeps what users already recognize. Both
  errors name `engram init`, `--vault` and `ENGRAM_VAULT_PATH`, as the issue requires.
- **Alternatives:** stderr for init's results, as the old creation line was. Rejected: the old line was a
  side effect of another command and init's result is the command's answer. Joe may still prefer stderr.

## 795b7 Two errors, not one

- **Choice:** a missing vault ([795a5a A missing vault is an error](./795a_product-spec.md#795a5a-a-missing-vault-is-an-error)) and a half-created vault ([795a5b A half-created vault is an error](./795a_product-spec.md#795a5b-a-half-created-vault-is-an-error)) get distinct messages.
- **Rationale:** the remedy differs ("create" versus "repair"), and a user who sees "no vault" for a
  directory full of files would be misled.
- **Alternatives:** one message for both. Rejected for that reason.

## 795b8 Repair stays limited to the missing vault ID

- **Choice:** [795a4a Init stamps the missing vault ID](./795a_product-spec.md#795a4a-init-stamps-the-missing-vault-id) repairs exactly what [792a2g Interrupted creation is repaired](../../specs/product/vault-identity.md#792a2g-interrupted-creation-is-repaired) repaired: a `.engram/` directory with no ID.
- **Rationale:** the issue says init replaces that repair. Recreating a missing `.gitignore` or README
  would add behavior it does not mention.
- **Alternatives:** have init recreate every missing creation file. Rejected as beyond the decision.

## 795b9 What "the vault steps" of update are

- **Choice:** [795a6a Update skips its vault steps](./795a_product-spec.md#795a6a-update-skips-its-vault-steps) names the vault notices, skill registration ([792a12t Update runs registration after deploying the skills](../../specs/product/skill-runbook-registration.md#792a12t-update-runs-registration-after-deploying-the-skills)) and the three vault
  options `--regen-vocab`, `--backfill-identity` and `--reparent-luhmann`. The install, the deploy and the
  report continue, and the exit status is unaffected ([795a6c The rest of update still runs](./795a_product-spec.md#795a6c-the-rest-of-update-still-runs)).
- **Rationale:** these are the update steps that read or write the vault ([install-and-update](../../specs/product/install-and-update.md)). A missing vault is a
  state to report, not a failure of an update that installed correctly.
- **Alternatives:** exit non-zero when a vault is missing. Rejected: it would fail every first-time update
  on a machine that has not run `engram init` yet.

## 795b10 Nodes checked and left unchanged

- **Choice:** the table in [795a1 Problem Statement](./795a_product-spec.md#795a1-problem-statement) lists them with reasons. [792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone) keeps its wording: "the creation
  line" is still true, since only init prints one.
- **Rationale:** editing a still-true requirement and its scenario for wording alone would add churn.
- **Alternatives:** reword [792a2f Existing vaults are left alone](../../specs/product/vault-identity.md#792a2f-existing-vaults-are-left-alone) and [792a2f1a An existing vault is untouched by read-only use](../../specs/product/vault-identity.md#792a2f1-uat-scenarios). Rejected.

## 795b11 Section intros are replaced as whole paragraphs

- **Choice:** [795a9 The memory-query intro no longer creates a missing vault](./795a_product-spec.md#795a9-modified-the-memory-query-intro-no-longer-creates-a-missing-vault) to [795a12 Parent-sync vaults are created once with engram init](./795a_product-spec.md#795a12-modified-parent-sync-vaults-are-created-once-with-engram-init) each replace one section's introductory paragraph and leave the
  heading and children alone.
- **Rationale:** the lifecycle guide asks for complete replacement text for one target ID, and the
  introduction is the only text in those sections that mentions creation.
- **Alternatives:** none considered; there is no narrower node to target.

## 795b12 Technical spec follow-up is deferred

- **Choice:** this change does not touch [792b2c3 First-use gating](../../specs/technical/command-line-surface.md#792b2c3-first-use-gating), which implements first-use gating, or the Supports
  lists that cite the Removed scenarios.
- **Rationale:** the request is for the product-spec change only.
- **Alternatives:** none; the technical change comes next.
