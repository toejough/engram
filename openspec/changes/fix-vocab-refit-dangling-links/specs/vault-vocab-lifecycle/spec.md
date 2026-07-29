# vault-vocab-lifecycle — delta

## MODIFIED Requirements

### Requirement: Derivational refit with centroid derivation
A refit operation SHALL derive the vocabulary from the vault's geometry: it clusters all non-definition note vectors using k-means with silhouette-based auto-K (reusing the recall clustering machinery), and the resulting clusters become the vocabulary's term set. Each derived cluster SHALL be matched to an existing term by greedy centroid cosine similarity above a match threshold; matched clusters keep the existing term's name and definition note. Unmatched clusters SHALL be surfaced as naming requests (structured output for agent-side LLM naming) and minted as new definition notes per the existing definition-note conventions. Existing derived terms left unmatched SHALL be retired by deletion: the term's definition note and its `.vec.json` sidecar are deleted from the vault, its `vocab/<term>` tags are stripped during the post-derivation re-tag pass, and every remaining reference to the deleted note — wikilinks in note bodies and `supersedes:` entries naming its basename — is removed. Retirement MUST NOT demote a definition note in place and MUST NOT record supersession for it: vocab definition notes have no supersession story — they are term identity, not preserved facts. The derived centroids (mean of cluster member vectors) MUST be persisted to `vocab.centroids.json` with per-term member counts, an `origin: derived|proposed` provenance field, and the derivation timestamp. Refit MUST NOT accept an externally authored term plan.

#### Scenario: Term count follows silhouette-optimal K
- **WHEN** `engram vocab refit` runs over a vault of non-definition notes
- **THEN** the resulting derived term count equals the K selected by silhouette-based auto-K over the note vectors, independent of the prior term count

#### Scenario: Name stability across derivations
- **WHEN** a derivation produces a cluster whose centroid cosine similarity to an existing term's centroid exceeds the match threshold
- **THEN** the cluster keeps that term's name and definition note, and no new definition note is minted for it

#### Scenario: Unmatched cluster is named, unmatched term is retired by deletion
- **WHEN** a derivation produces a cluster matching no existing term, and an existing derived term matches no cluster
- **THEN** the new cluster is emitted as a naming request and minted on answer with `tags: [vocab, vocab/<term>]`, and the unmatched term's definition note and sidecar are deleted, its member tags stripped, and no supersedes entry or demoted note remains in the vault

#### Scenario: Retirement leaves no dangling references
- **WHEN** a refit retires a term whose definition note is wikilinked or named in a `supersedes:` entry elsewhere in the vault (e.g. the vocab-definition family note)
- **THEN** those references are removed in the same refit, and `engram check` reports no G3 dangling link targeting the deleted basename

#### Scenario: Centroids are means of member vectors
- **WHEN** derivation completes and `writeCentroidsFile` persists the result
- **THEN** `vocab.centroids.json` contains one entry per term with `member_count`, `origin: derived` (for derivation-produced terms), and `vector` equal to the mean of that cluster's member vectors

#### Scenario: Refit baseline seeds last_refit timestamp
- **WHEN** a derivation applies (bootstrap or scheduled refit)
- **THEN** `vocab.centroids.json` includes `last_refit: {note_count: <total non-definition notes>, date: "YYYY-MM-DD"}` recording the vault state at derivation time, and `refit_pending` is cleared

#### Scenario: Dry run reports the derivation diff without writing
- **WHEN** `engram vocab refit --dry-run` runs
- **THEN** the matched/new/retired term sets, selected K, and silhouette score are printed and no vault file or centroids file is modified
