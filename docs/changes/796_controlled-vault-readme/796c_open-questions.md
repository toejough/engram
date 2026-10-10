# Controlled Vault README - Open Questions

Each needs Joe's input before [796a Controlled Vault README - Product Spec](./796a_product-spec.md) is final. Choices that need only validation are in
[`796b_decision-log.md`](./796b_decision-log.md).

## 796c1 Should a missing README be restored?

- **Context:** [796a2 vault-identity (controlled vault README)](./796a_product-spec.md#796a2-modified-vault-identity-controlled-vault-readme) restores a README that is present and differs from the starter.
  A deleted README is neither. Creating the README is vault creation, which is #795's concern.
- **Options:**
  1. Leave it missing.
  2. Restore it on the first write, as for an edited README.
- **Recommendation:** 1, since vault creation belongs to #795.
- **Blocks:** the "present" condition in [796a2 vault-identity (controlled vault README)](./796a_product-spec.md#796a2-modified-vault-identity-controlled-vault-readme). Default: option 1.

## 796c2 What happens to README text in existing vaults?

- **Context:** an existing vault may hold a README with user text. The first writing command after this
  change overwrites it.
- **Options:**
  1. Overwrite it with the warning; a git-backed vault recovers the text from history.
  2. Save the old text to a note before overwriting.
- **Recommendation:** 1; option 2 is behavior beyond the issue's decisions.
- **Blocks:** [796a2a3 A write restores an edited README and warns](./796a_product-spec.md#796a2a-uat-scenarios). Default: option 1.
