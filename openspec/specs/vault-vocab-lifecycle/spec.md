# Vocabulary Lifecycle Specification

## Purpose

The controlled vocabulary's product behavior (definition and family notes, write-time tagging, bootstrap, propose, derivational refit, the refit trigger, `vocab stats`, the self-tag backfill and the old-format migration) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a6-added-controlled-vocabulary`. What remains here waits for vocabulary-engine (792c7e3: the technical mechanism residue) and memory-query (792c7f1: the supersession ride-along).
## Requirements
### Requirement: Vocabulary mechanism residue (vocabulary-engine, 792c7e3)
- The bare `vocab` marker is the sole definition discriminator (`isVocabDefinitionNote`), so `loadVocabAssignmentBodyVector()` returns nil, false for a definition note, and refit member scans skip definitions.
- `AssignVocabTerms(bodyVec, terms, floor)` computes cosine(bodyVec, term.Vector) per term, keeps scores ≥ floor, and returns at most 3, sorted descending by score with term name ascending as tie-breaker.
- `loadAssignmentTermVectors()` reads `vocab.centroids.json`; when its `embedding_model_id` matches the loaded term sidecars, terms with a centroid entry use the centroid vector and the others fall back to their description vector (side-loaded from their sidecar).
- `checkAndPersistVocabRefitTrigger` runs on every write and sets `refit_pending: true` in `vocab.centroids.json` when `evaluateVocabTriggers()` finds `totalNotes - lastRefit.NoteCount ≥ 40` AND `daysSince ≥ 14`. All trigger math derives counts from non-definition notes only.
- `engram vocab tag-definitions` derives each term via `termFromDefinitionSlug` and writes through the vault's locked write path.
- Derivation reuses the recall clustering machinery (k-means with silhouette-based auto-K). `writeCentroidsFile` persists one entry per term with `member_count`, `origin: derived|proposed` and `vector` equal to the mean of the cluster's member vectors, plus the derivation timestamp and `last_refit: {note_count, date}`.

#### Scenario: Definition note exempt from term assignment
- **WHEN** a note carries `tags: [vocab]` in its frontmatter (the bare "vocab" marker), with or without an additional `vocab/<term>` self-tag
- **THEN** `isVocabDefinitionNote()` returns true and `loadVocabAssignmentBodyVector()` returns nil, false (exempting it from write-time assignment)

#### Scenario: Minted definition carries its self-tag
- **WHEN** `mintDefinitionNote` creates the definition for term `<term>` (bootstrap seed or refit-minted)
- **THEN** the written note's `tags:` list is `[vocab, vocab/<term>]` (bare marker first, self-tag appended)

#### Scenario: Family-note mint stays bare-only
- **WHEN** `ensureVocabFamilyNote` mints or re-mints the family note (slug `vocab-definition`)
- **THEN** the written note's `tags:` list contains `vocab` and no `vocab/` prefixed entry

#### Scenario: Refit preserves definition self-tags
- **WHEN** a refit (`retagAllNotesTwoPass`) runs over a vault whose definition notes carry self-tags
- **THEN** every definition note's `tags:` list is byte-identical before and after the refit

#### Scenario: Top-3 terms above floor are retained
- **WHEN** `AssignVocabTerms(bodyVec, terms, floor)` is called with bodyVec, a list of TermWithVector entries, and floor=0.35
- **THEN** cosine(bodyVec, term.Vector) is computed for each term; only terms with score ≥ 0.35 are included; at most 3 (top-ranked by score) are returned, sorted descending by score with term name ascending as tie-breaker

#### Scenario: Centroids preferred over descriptions
- **WHEN** `loadAssignmentTermVectors()` reads vocab.centroids.json and the embedding_model_id matches the loaded term sidecars
- **THEN** terms whose centroid entry exists in the centroids file use the centroid vector; terms without a centroid entry fall back to their description vector (side-loaded from their sidecar)

#### Scenario: Growth trigger armed by both constraints
- **WHEN** `evaluateVocabTriggers()` finds `totalNotes - lastRefit.NoteCount ≥ 40` AND `daysSince ≥ 14`
- **THEN** trigger fires with reason "growth: N notes, D days"

#### Scenario: Self-tagged definitions leave trigger math unchanged
- **WHEN** the same vault is evaluated before and after definitions gain `vocab/<term>` self-tags
- **THEN** the growth trigger's note count and outcome are identical in both states (definition notes are excluded from trigger math)

#### Scenario: Centroids are means of member vectors
- **WHEN** derivation completes and `writeCentroidsFile` persists the result
- **THEN** `vocab.centroids.json` contains one entry per term with `member_count`, `origin: derived` (for derivation-produced terms), and `vector` equal to the mean of that cluster's member vectors

### Requirement: Supersession ride-along (residue for memory-query, 792c7f1)
When a note carries a `supersedes:` frontmatter block listing older notes it replaces (via `type: updates|narrows|refutes`), the newer note SHALL be inserted directly after any delivered older note in the query results, regardless of whether the newer note itself matched the query. The inserted note MUST carry provenance `ride_along` so Gate-2 analysis can detect rank shifts from insertions.

#### Scenario: Superseder follows superseded in results
- **WHEN** query results include a delivered note with a recorded superseder in `SupersedesInverse`
- **THEN** `applySupersedesRideAlong()` inserts the superseding note immediately after the superseded note, marked with provenance `ride_along`

#### Scenario: Deduplication prevents double insertion
- **WHEN** a superseding note is already present in results (as a direct hit or prior ride-along)
- **THEN** it is not inserted again
