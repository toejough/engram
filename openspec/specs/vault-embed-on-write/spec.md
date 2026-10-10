# Embed-on-Write + Dual-Vector Sidecars Specification

## Purpose

Residue awaiting later #792 nodes. The sidecar format, path, embedding sources, content hash, build,
the learn-time write and state classification are in
`docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b10-added-embedding-sidecar`,
and the `learn` guarantee in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a3-added-note-writing`
(792a3n). The `amend` and `resituate` re-embedding and the lossless `last_used` bump are in
`docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a4-added-note-revision`
(792a4g, 792a4l, 792a4n). `embed apply` waits for vault-health (792c7h1).

## Requirements

### Requirement: Sidecar state classification residue
#### Scenario: Force flag re-embeds mismatched
- **WHEN** `RunEmbedApply` is called with `--force`
- **THEN** notes with `StateIncompatible` are included in the selection and their sidecars are re-embedded under the current model

### Requirement: Sidecar survives embed operations
The sidecar is a **derived file**, not part of the note's git-tracked content. When `engram embed apply` re-embeds notes, it SHALL only modify sidecars; existing content-based metadata (like `content_hash`) stays honest because the sidecar MUST always be recomputed from the current note content. Sidecars MUST never be stored in or versioned with the note file itself.

#### Scenario: Embed apply rewrites sidecars only
- **WHEN** `RunEmbedApply` is called
- **THEN** note files are read (not modified), and only `.vec.json` sidecars are written; note content remains unchanged

#### Scenario: Sidecar not part of note frontmatter
- **WHEN** a note is edited and re-written
- **THEN** the sidecar is regenerated on the next write operation (learn/amend/resituate); the note's `.md` file itself never contains embedding vectors or model IDs
