## Purpose

The behavior of `engram update --reparent-luhmann` as the user sees it (opt-in, derive, answers, apply,
chunk-index reconciliation, further candidates, dry-run, and the refusal and CRLF rules of the rename) is in
`docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a10-added-vault-migration`
(792a10b to 792a10i). The component contract below waits for note-reparenting (792c7i2).

## Requirements

### Requirement: Derive and apply parameters
The derive phase SHALL consider, for each top-level note, its top-3 nearest other top-level notes by
embedding-sidecar cosine similarity above a floor of 0.75, and emit a fingerprint of the vault's current
state that the answers file echoes. The apply phase SHALL process answers in ascending original-ID order,
via the `nextLuhmannID` machinery, so renumbering within one run cannot collide, and SHALL re-index the
renamed notes (ingest) and then detach their old-path chunk-manifest entries (prune), in process.
