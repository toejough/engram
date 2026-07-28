# Delta: vault-vocab-lifecycle — derivational refit

## RENAMED Requirements

- FROM: `### Requirement: Two-pass refit with centroid derivation`
- TO: `### Requirement: Derivational refit with centroid derivation`

## MODIFIED Requirements

### Requirement: Derivational refit with centroid derivation
A refit operation SHALL derive the vocabulary from the vault's geometry: it clusters all non-definition note vectors using k-means with silhouette-based auto-K (reusing the recall clustering machinery), and the resulting clusters become the vocabulary's term set. Each derived cluster SHALL be matched to an existing term by greedy centroid cosine similarity above a match threshold; matched clusters keep the existing term's name and definition note. Unmatched clusters SHALL be surfaced as naming requests (structured output for agent-side LLM naming) and minted as new definition notes per the existing definition-note conventions. Existing derived terms left unmatched SHALL be retired: their definition notes superseded and their `vocab/<term>` tags stripped during the post-derivation re-tag pass. The derived centroids (mean of cluster member vectors) MUST be persisted to `vocab.centroids.json` with per-term member counts, an `origin: derived|proposed` provenance field, and the derivation timestamp. Refit MUST NOT accept an externally authored term plan.

#### Scenario: Term count follows silhouette-optimal K
- **WHEN** `engram vocab refit` runs over a vault of non-definition notes
- **THEN** the resulting derived term count equals the K selected by silhouette-based auto-K over the note vectors, independent of the prior term count

#### Scenario: Name stability across derivations
- **WHEN** a derivation produces a cluster whose centroid cosine similarity to an existing term's centroid exceeds the match threshold
- **THEN** the cluster keeps that term's name and definition note, and no new definition note is minted for it

#### Scenario: Unmatched cluster is named, unmatched term is retired
- **WHEN** a derivation produces a cluster matching no existing term, and an existing derived term matches no cluster
- **THEN** the new cluster is emitted as a naming request and minted on answer with `tags: [vocab, vocab/<term>]`, and the unmatched term's definition note is superseded with its member tags stripped in the re-tag pass

#### Scenario: Centroids are means of derived cluster members
- **WHEN** derivation completes and `writeCentroidsFile` persists the result
- **THEN** `vocab.centroids.json` contains one entry per term with `member_count`, `origin: derived` (for derivation-produced terms), and `vector` equal to the mean of that cluster's member vectors, plus `last_refit: {note_count, date}`

#### Scenario: Dry run reports the derivation diff without writing
- **WHEN** `engram vocab refit --dry-run` runs
- **THEN** the matched/new/retired term sets, selected K, and silhouette score are printed and no vault file or centroids file is modified

### Requirement: Autonomous refit triggers
The vault SHALL check its own tag health on every write (via `checkAndPersistVocabRefitTrigger`), setting `refit_pending: true` in `vocab.centroids.json` when the growth trigger fires, without initiating refit itself. The sole refit trigger is growth: notes added since last derivation ≥ 40 AND days elapsed ≥ 14. Untagged rate and hub concentration SHALL be reported by `vocab stats` as diagnostics only and MUST NOT set `refit_pending` or force a refit verdict. All trigger math SHALL derive counts from non-definition notes only: a definition note's `vocab/<term>` self-tag MUST NOT count toward its term's membership, and definition notes MUST NOT count as untagged.

#### Scenario: Growth trigger armed by both constraints
- **WHEN** `evaluateVocabTriggers()` finds `totalNotes - lastRefit.NoteCount ≥ 40` AND `daysSince ≥ 14`
- **THEN** trigger fires with reason "growth: N notes, D days"

#### Scenario: Untagged rate is diagnostic only
- **WHEN** the fraction of non-definition notes without any vocab term exceeds 8% and the growth trigger has not fired
- **THEN** `refit_pending` remains false and `vocab stats` reports the untagged rate without a REFIT_PENDING verdict

#### Scenario: Hub concentration is diagnostic only
- **WHEN** a single term claims more than 25% of non-definition notes and the growth trigger has not fired
- **THEN** `refit_pending` remains false and `vocab stats` reports the hub concentration without a REFIT_PENDING verdict

## ADDED Requirements

### Requirement: Proposed terms survive derivation
Terms minted via `engram vocab propose` SHALL carry `origin: proposed` in `vocab.centroids.json` and MUST NOT be retired by a derivation that produces no matching cluster — they exist precisely to represent concepts the clustering cannot see. Proposed terms participate in write-time assignment normally.

#### Scenario: Unmatched proposed term is kept
- **WHEN** a derivation completes and a term with `origin: proposed` matches no derived cluster
- **THEN** the term's definition note and centroid entry remain, marked `origin: proposed`, and member tags assigned to it are not stripped