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

### Requirement: Two-pass refit with centroid derivation
A refit operation SHALL run two passes over the vault's non-definition notes: pass 1 assigns terms against description vectors (or prior centroids), accumulating member-sets per term; pass 2 computes the mean vector of each term's accumulated members and re-assigns all notes against the new centroids. The derived centroids MUST be persisted to `vocab.centroids.json` alongside per-term member counts and the centroid derivation's timestamp.

#### Scenario: Centroids are means of member vectors
- **WHEN** `retagAllNotesTwoPass()` completes, pass-2 centroid assignment is applied, and `writeCentroidsFile()` persists the result
- **THEN** `vocab.centroids.json` contains one entry per term with `member_count` (count of members) and `vector` (mean of all pass-1-assigned member vectors for that term)

#### Scenario: Refit baseline seeds last_refit timestamp
- **WHEN** `retagAllNotesTwoPass()` is called with a non-nil lastRefit parameter (bootstrap or scheduled refit)
- **THEN** `vocab.centroids.json` includes `last_refit: {note_count: <total>, date: "YYYY-MM-DD"}` recording the vault state at refit time

### Requirement: Autonomous refit triggers
The vault SHALL check its own tag health on every write (via `checkAndPersistVocabRefitTrigger`), evaluating three thresholds and setting `refit_pending: true` in `vocab.centroids.json` when any fires, without initiating refit itself. Triggers are: (a) growth—when notes added since last refit ≥ 40 AND days elapsed ≥ 14; (b) untagged rate—when fraction of notes without any vocab term > 8%; (c) hub concentration—when a single term claims > 25% of all notes. All trigger math SHALL derive member and untagged counts from non-definition notes only: a definition note's `vocab/<term>` self-tag MUST NOT count toward its term's membership, and definition notes MUST NOT count as untagged.

#### Scenario: Growth trigger armed by both constraints
- **WHEN** `evaluateVocabTriggers()` finds `totalNotes - lastRefit.NoteCount ≥ 40` AND `daysSince ≥ 14`
- **THEN** trigger fires with reason "growth: N notes, D days"

#### Scenario: Untagged rate trigger
- **WHEN** `evaluateVocabTriggers()` computes `untaggedCount / totalNotes > 0.08`
- **THEN** trigger fires with reason "untagged: X.X%"

#### Scenario: Hub concentration trigger
- **WHEN** `evaluateVocabTriggers()` finds a term where `memberCount / totalNotes > 0.25`
- **THEN** trigger fires with reason "hub: <term> (X%)"

#### Scenario: Self-tagged definitions leave trigger math unchanged
- **WHEN** the same vault is evaluated before and after definitions gain `vocab/<term>` self-tags
- **THEN** `evaluateVocabTriggers()` produces identical member counts, untagged counts, and trigger outcomes in both states

### Requirement: Tag nomination in recall queries
Recall's query path SHALL nominate cross-cluster candidate notes that share a vocabulary tag with the top-3 matched notes in a cluster. When a cluster's top-3 delivered notes carry tags in the `vocab/<term>` namespace, every non-definition vault note also tagged with any of those terms MUST be nominated into that cluster's candidate list (up to 40 per cluster, deduplicated across clusters). Definition notes SHALL be excluded from nomination on BOTH sides: a definition MUST never be nominated into a pool, and a definition appearing among a cluster's top-3 delivered notes MUST NOT contribute its self-tag as a nomination seed — its `vocab/<term>` self-tag is a display affordance, not a membership claim. Nominated candidates SHALL carry cosine score 0 (tag-matched, not centroid-ranked).

#### Scenario: Tag nomination feeds candidate_l2s
- **WHEN** a query cluster's top-3 delivered notes carry tags like `vocab/retrieval-design`
- **THEN** `buildTagNominations()` collects all non-definition notes tagged `vocab/retrieval-design` and appends them to the cluster's `candidate_l2s` (up to cap 40), reported in the query budget as `tag_nominations_added`

#### Scenario: Truncation reported in budget
- **WHEN** nomination pool exceeds nominationCapPerCluster (40)
- **THEN** the count is tracked in query budget as `tag_nominations_dropped` (the overflow count)

#### Scenario: Definitions never nominated via self-tag
- **WHEN** a cluster's top-3 delivered notes carry `vocab/<term>` and that term's definition note carries the same tag as its self-tag
- **THEN** the definition note does not appear in the cluster's nominations

#### Scenario: A top-3 definition seeds no nominations
- **WHEN** a self-tagged definition note is itself among a cluster's top-3 delivered notes
- **THEN** its `vocab/<term>` self-tag contributes no seed terms — the cluster's nominations are identical to what a bare-only definition in that position would produce

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

