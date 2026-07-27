# Embed-on-Write + Dual-Vector Sidecars Specification

## Purpose

Every note written to the vault gets a sibling `.vec.json` sidecar file stamped with dual embedding vectors (one for the note's `situation:` field, one for the body), embedding model ID, dimensions, and content hash. The sidecar is self-contained—no separate index to build or maintain—and its dual vectors let retrieval match against whichever angle of the note fits best. Why: `docs/architecture/adr.md` ADR-0003. Validation: correctness rests on embed/sidecar invariants (`docs/architecture/memory-invariants.md`) and unit tests.

## Requirements

### Requirement: Dual-vector sidecar structure
Every sidecar `.vec.json` file SHALL contain a schema-versioned JSON object with `situation_vector` and `body_vector` (both 384-dimensional float arrays for MiniLM-L6-v2), plus metadata: `embedding_model_id`, `dims`, `content_hash`, and optional `last_used` (the date this note last surfaced as a useful recall hit, omitted when never used).

#### Scenario: Sidecar fields match spec contract
- **WHEN** `Sidecar` is marshaled via `MarshalSidecar()`
- **THEN** JSON contains exactly these top-level fields: `schema_version` (int, value 1), `embedding_model_id` (string), `dims` (int), `situation_vector` (float32 array), `body_vector` (float32 array), `content_hash` (string), `last_used` (string, omitted when empty)

#### Scenario: Dims consistency enforced
- **WHEN** `UnmarshalSidecar()` decodes a sidecar
- **THEN** it verifies `len(situation_vector) == dims` and `len(body_vector) == dims`; if either check fails, `ErrDimsMismatch` is returned with the mismatch details

#### Scenario: Schema version validated
- **WHEN** `UnmarshalSidecar()` decodes a sidecar
- **THEN** it checks `schema_version == SidecarSchemaVersion` (currently 1); if they differ (e.g., an old single-vector sidecar), `ErrSchemaVersion` is returned

### Requirement: Situation vector defaults to body when absent
When a note has no `situation:` frontmatter field, the body text SHALL be embedded as the situation vector, so every note still carries a meaningful situation vector for retrieval. The two vectors may be identical in this case, but both fields MUST always be present.

#### Scenario: Situation falls back to body
- **WHEN** `BuildSidecar()` is called on a note with no `situation:` field
- **THEN** `SituationText()` returns empty bytes, and `situationInput` is set to `BodyText()` before embedding; both `situation_vector` and `body_vector` are computed (both from the body text)

#### Scenario: Situation explicitly set
- **WHEN** `BuildSidecar()` is called on a note with a non-empty `situation:` frontmatter field
- **THEN** the situation vector is computed from the situation field; the body vector is computed from the body text independently

### Requirement: Sidecar written on every note write
Every `engram learn`, `engram amend`, and `engram resituate` SHALL write both the note content and its sidecar. The sidecar path MUST be derived as `<note-path>.vec.json` (e.g., `1aa.note.md` → `1aa.note.vec.json`).

#### Scenario: Learn triggers embed
- **WHEN** `RunLearn` writes a new fact/feedback note
- **THEN** `BuildSidecar()` is called on the note bytes, and the returned sidecar is written via `WriteFileAtomic` to `<notePath>.vec.json`

#### Scenario: Amend re-embeds
- **WHEN** `RunAmend` rewrites an existing note's content
- **THEN** the note's sidecar is re-embedded and rewritten under the `.luhmann.lock` critical section

#### Scenario: Sidecar path derived consistently
- **WHEN** `SidecarPath("<notePath>")` is called
- **THEN** it returns `<notePath>` with `.md` suffix stripped and `.vec.json` appended (e.g., `1aa.note.md` → `1aa.note.vec.json`)

### Requirement: Sidecar state classification SHALL derive from content-hash and model-ID comparison, excluding additive metadata

Each sidecar's state SHALL be computed by comparing the sidecar's content hash and embedding model id against the note's current content and the active model (internal/embed/state.go ComputeState → stale on hash mismatch, incompatible on model mismatch). The `last_used` field is additive metadata deliberately excluded from the content hash.

#### Scenario: Hash staleness detection
- **WHEN** `embed.ComputeState()` compares a sidecar's stored `content_hash` against a freshly-computed `ContentHash()` of the note's raw bytes
- **THEN** if they differ, the state is `StateStale` (the sidecar is out of date)

#### Scenario: Hash computed once per write
- **WHEN** `BuildSidecar()` is called
- **THEN** `ContentHash(raw)` is called exactly once and stored in the sidecar; the hash is never recomputed on subsequent reads

#### Scenario: Model mismatch sets state
- **WHEN** `embed.ComputeState()` compares sidecar's `embedding_model_id` against `embedder.ModelID()`
- **THEN** if they differ, the state is `StateIncompatible`

#### Scenario: Force flag re-embeds mismatched
- **WHEN** `RunEmbedApply` is called with `--force`
- **THEN** notes with `StateIncompatible` are included in the selection and their sidecars are re-embedded under the current model

#### Scenario: Last-used update is lossless
- **WHEN** `activateNote()` bumps a sidecar's `last_used` field
- **THEN** all other sidecar fields (including `content_hash`) remain unchanged; the new sidecar is still valid (not marked stale) because last-used is excluded from the staleness check

#### Scenario: Backward compatibility
- **WHEN** an old sidecar without a `last_used` field is unmarshaled
- **THEN** `Sidecar.LastUsed` decodes as `""` (empty string); the sidecar still validates (no schema mismatch)

### Requirement: Sidecar survives embed operations
The sidecar is a **derived file**, not part of the note's git-tracked content. When `engram embed apply` re-embeds notes, it SHALL only modify sidecars; existing content-based metadata (like `content_hash`) stays honest because the sidecar MUST always be recomputed from the current note content. Sidecars MUST never be stored in or versioned with the note file itself.

#### Scenario: Embed apply rewrites sidecars only
- **WHEN** `RunEmbedApply` is called
- **THEN** note files are read (not modified), and only `.vec.json` sidecars are written; note content remains unchanged

#### Scenario: Sidecar not part of note frontmatter
- **WHEN** a note is edited and re-written
- **THEN** the sidecar is regenerated on the next write operation (learn/amend/resituate); the note's `.md` file itself never contains embedding vectors or model IDs
