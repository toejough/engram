## Purpose

The `skill_hash`, `skill_key` and `skill_source` requirements are in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a12u-skill-identity-fields-belong-to-registration` and `792b_technical-spec.md#792b6-added-note-document-format` (792b6c9, 792b6c10). The repo/user provenance stamped by `engram learn` is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a3-added-note-writing`, and its detection in `792b_technical-spec.md#792b11-added-note-provenance`. Revision by `engram amend` and `engram resituate` (identity re-stamp, vault fill, key preservation, anchor refusal and CRLF conversion) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a4-added-note-revision`. The exchange identity of a note (`xid`, vault-qualified `parent` links, `aliases`, their survival of every other rewrite, the first-write stamp of a pull-down and the offer receipt's keeping the declared author) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a21-added-parent-sync` (792a21zh to 792a21zl, 792a21u). What remains here waits for curate (792c7u1).
## Requirements
### Requirement: Curation bookkeeping keeps the declared identity (residue)
The amend re-stamp and the `--activate` bookkeeping amend are in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a4-added-note-revision` (792a4h, 792a4i). These bookkeeping amends also SHALL NOT re-stamp `repo:`, `user:` and `vault:`, so an accepted offer keeps its declared author; they wait for curate (792c7u1):
- `--clear-pending`;
- `--discard --into` on the target note.

#### Scenario: Accepting an offer keeps its author
- **WHEN** `engram amend --target O --clear-pending` runs on a pending offer whose `user:` is alice, from an environment whose detected user is bob
- **THEN** O's `user:` is still alice

### Requirement: Curation folds and the judged-version check read a note as LF (residue)
Waiting for curate (792c7u1): a fold (`--discard --into`) SHALL read its existing note as LF and write it as LF only when the fold changes it, and the judged-version check (`--expect-hash`) SHALL run on the LF form, the form amend edits; for a note whose frontmatter is LF this is the hash `engram show` prints.

#### Scenario: Fold converts a CRLF existing note
- **WHEN** `engram amend --target O --discard --into E` folds an offer into a note E whose lines end in `\r\n`, and the fold changes E's aliases or parent links
- **THEN** the fold succeeds, E is written with no `\r\n`, and O and its sidecar are gone
