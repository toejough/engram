# Vocabulary Lifecycle Specification

## Purpose

Engram maintains a controlled vocabulary of definition notes and assigns them to new notes via tag-based term nomination during write operations. The vault autonomously monitors tag health (growth, concentration, untagged rate) and prompts refit when thresholds trigger, preventing vocabulary drift without manual intervention. Why: `docs/architecture/adr.md` ADR-0011. Validation: `dev/eval/LEDGER.md#vocab-tag-nomination-l6xtag` (nomination coverage); `dev/eval/LEDGER.md#vocab-refit-cost` (refit cost).
## Requirements
### Requirement: Definition notes carry bare-vocab marker
A vocabulary term's own definition SHALL be authored as a note carrying BOTH the bare `vocab` marker tag (no slash) AND its own term tag `vocab/<term>` in its `tags:` frontmatter list — the bare marker remains the sole definition discriminator (`isVocabDefinitionNote`), while the self-tag connects the definition to its member cluster in tag-based views (Obsidian graph/tag pane). Definition minting (`vocab bootstrap` and refit-minted definitions) SHALL write both tags at creation time. The FAMILY note (slug `vocab-definition` — the tag family's own record, which has no term) SHALL remain bare-`vocab`-only. The definition's body IS the term's description and MUST never be auto-assigned a vocab term itself, preventing definition vectors from skewing their own term's centroid. When loaded for refit, definition notes are automatically excluded from member-note scans, and a refit MUST NOT strip or rewrite a definition note's self-tag.

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

### Requirement: Write-time term assignment by cosine similarity
Every note written to the vault (via learn, amend, resituate) SHALL have its vocab terms assigned by computing cosine similarity between the note's body vector and each term's vector, retaining terms whose similarity score exceeds a floor (0.35), capped at top-3 terms. When a centroid version exists (from a prior refit), it MUST be used in place of the description embedding for assignment, letting term membership reflect actual member vectors rather than descriptions alone.

#### Scenario: Top-3 terms above floor are retained
- **WHEN** `AssignVocabTerms(bodyVec, terms, floor)` is called with bodyVec, a list of TermWithVector entries, and floor=0.35
- **THEN** cosine(bodyVec, term.Vector) is computed for each term; only terms with score ≥ 0.35 are included; at most 3 (top-ranked by score) are returned, sorted descending by score with term name ascending as tie-breaker

#### Scenario: Centroids preferred over descriptions
- **WHEN** `loadAssignmentTermVectors()` reads vocab.centroids.json and the embedding_model_id matches the loaded term sidecars
- **THEN** terms whose centroid entry exists in the centroids file use the centroid vector; terms without a centroid entry fall back to their description vector (side-loaded from their sidecar)

### Requirement: Autonomous refit triggers
The vault SHALL check its own tag health on every write (via `checkAndPersistVocabRefitTrigger`), setting `refit_pending: true` in `vocab.centroids.json` when the growth trigger fires, without initiating refit itself. The sole refit trigger is growth: notes added since last derivation ≥ 40 AND days elapsed ≥ 14. Untagged rate and hub concentration SHALL be reported by `vocab stats` as diagnostics only and MUST NOT set `refit_pending` or force a refit verdict. All trigger math SHALL derive counts from non-definition notes only: a definition note's `vocab/<term>` self-tag MUST NOT count toward its term's membership, and definition notes MUST NOT count as untagged.

#### Scenario: Growth trigger armed by both constraints
- **WHEN** `evaluateVocabTriggers()` finds `totalNotes - lastRefit.NoteCount ≥ 40` AND `daysSince ≥ 14`
- **THEN** trigger fires with reason "growth: N notes, D days"

#### Scenario: Untagged rate trigger
- **WHEN** the fraction of non-definition notes without any vocab term exceeds 8% and the growth trigger has not fired
- **THEN** `refit_pending` remains false and `vocab stats` reports the untagged rate (with a `[high]` diagnostic flag) without a REFIT_PENDING verdict

#### Scenario: Hub concentration trigger
- **WHEN** a single term claims more than 25% of non-definition notes and the growth trigger has not fired
- **THEN** `refit_pending` remains false and `vocab stats` reports the hub concentration without a REFIT_PENDING verdict

#### Scenario: Self-tagged definitions leave trigger math unchanged
- **WHEN** the same vault is evaluated before and after definitions gain `vocab/<term>` self-tags
- **THEN** the growth trigger's note count and outcome are identical in both states (definition notes are excluded from trigger math)

### Requirement: Supersession ride-along
When a note carries a `supersedes:` frontmatter block listing older notes it replaces (via `type: updates|narrows|refutes`), the newer note SHALL be inserted directly after any delivered older note in the query results, regardless of whether the newer note itself matched the query. The inserted note MUST carry provenance `ride_along` so Gate-2 analysis can detect rank shifts from insertions.

#### Scenario: Superseder follows superseded in results
- **WHEN** query results include a delivered note with a recorded superseder in `SupersedesInverse`
- **THEN** `applySupersedesRideAlong()` inserts the superseding note immediately after the superseded note, marked with provenance `ride_along`

#### Scenario: Deduplication prevents double insertion
- **WHEN** a superseding note is already present in results (as a direct hit or prior ride-along)
- **THEN** it is not inserted again

### Requirement: Definition self-tag backfill subcommand
`engram vocab tag-definitions` SHALL be an explicit, idempotent one-shot subcommand that adds the missing `vocab/<term>` self-tag to every existing definition note, deriving each term via `termFromDefinitionSlug`. It MUST write through the vault's locked write path and MUST leave every touched note's sidecar valid without any refresh step — vocab tags are not content-hash inputs (the hash covers situation and body text only), so a tags-only rewrite cannot stale a sidecar. It MUST leave already-self-tagged definitions untouched, MUST skip the family note (slug `vocab-definition`, empty term) reporting it as such, and MUST report per-note results (added vs already present vs skipped). The backfill MUST NOT run as a side effect of `engram update` or any other command.

#### Scenario: Backfill adds missing self-tags
- **WHEN** `engram vocab tag-definitions` runs against a vault whose definitions carry only the bare `vocab` marker
- **THEN** each per-term definition note's `tags:` list becomes `[vocab, vocab/<term>]`, its embed state remains clean (no `StateStale` — no re-embed occurred or was needed), and the command reports one "added" line per touched note

#### Scenario: Backfill is idempotent
- **WHEN** `engram vocab tag-definitions` runs a second time over the same vault
- **THEN** no note content changes and every definition is reported as already present

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

### Requirement: Proposed terms survive derivation
Terms minted via `engram vocab propose` SHALL carry `origin: proposed` in `vocab.centroids.json` and MUST NOT be retired by a derivation that produces no matching cluster — they exist precisely to represent concepts the clustering cannot see. Proposed terms participate in write-time assignment normally.

#### Scenario: Unmatched proposed term is kept
- **WHEN** a derivation completes and a term with `origin: proposed` matches no derived cluster
- **THEN** the term's definition note and centroid entry remain, marked `origin: proposed`, and member tags assigned to it are not stripped

