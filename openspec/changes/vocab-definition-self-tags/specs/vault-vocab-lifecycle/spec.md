## MODIFIED Requirements

### Requirement: Definition notes carry bare-vocab marker
A vocabulary term's own definition SHALL be authored as a note carrying BOTH the bare `vocab` marker tag (no slash) AND its own term tag `vocab/<term>` in its `tags:` frontmatter list — the bare marker remains the sole definition discriminator (`isVocabDefinitionNote`), while the self-tag connects the definition to its member cluster in tag-based views (Obsidian graph/tag pane). Definition minting (`vocab bootstrap` and refit-minted definitions) SHALL write both tags at creation time. The FAMILY note (slug `vocab-definition` — the tag family's own record, which has no term) SHALL remain bare-`vocab`-only. The definition's body IS the term's description and MUST never be auto-assigned a vocab term itself, preventing definition vectors from skewing their own term's centroid. When loaded for refit, definition notes are automatically excluded from member-note scans, and a refit MUST NOT strip or rewrite a definition note's self-tag.

#### Scenario: Definition note exempt from term assignment
- **WHEN** a note carries `tags: [vocab]` in its frontmatter (the bare "vocab" marker), with or without an additional `vocab/<term>` self-tag
- **THEN** `isVocabDefinitionNote()` returns true and `loadVocabAssignmentBodyVector()` returns nil, false (exempting it from write-time assignment)

#### Scenario: Minted definition carries its self-tag
- **WHEN** `mintDefinitionNote` creates the definition for term `<term>` (bootstrap seed or refit-minted)
- **THEN** the written note's `tags:` list contains both `vocab` and `vocab/<term>`

#### Scenario: Refit preserves definition self-tags
- **WHEN** a refit (`retagAllNotesTwoPass`) runs over a vault whose definition notes carry self-tags
- **THEN** every definition note's `tags:` list is byte-identical before and after the refit

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
Recall's query path SHALL nominate cross-cluster candidate notes that share a vocabulary tag with the top-3 matched notes in a cluster. When a cluster's top-3 delivered notes carry tags in the `vocab/<term>` namespace, every non-definition vault note also tagged with any of those terms MUST be nominated into that cluster's candidate list (up to 40 per cluster, deduplicated across clusters). Definition notes SHALL be excluded from nomination pools — their `vocab/<term>` self-tag is a display affordance, not a membership claim. Nominated candidates SHALL carry cosine score 0 (tag-matched, not centroid-ranked).

#### Scenario: Tag nomination feeds candidate_l2s
- **WHEN** a query cluster's top-3 delivered notes carry tags like `vocab/retrieval-design`
- **THEN** `buildTagNominations()` collects all non-definition notes tagged `vocab/retrieval-design` and appends them to the cluster's `candidate_l2s` (up to cap 40), reported in the query budget as `tag_nominations_added`

#### Scenario: Truncation reported in budget
- **WHEN** nomination pool exceeds nominationCapPerCluster (40)
- **THEN** the count is tracked in query budget as `tag_nominations_dropped` (the overflow count)

#### Scenario: Definitions never nominated via self-tag
- **WHEN** a cluster's top-3 delivered notes carry `vocab/<term>` and that term's definition note carries the same tag as its self-tag
- **THEN** the definition note does not appear in the cluster's nominations

## ADDED Requirements

### Requirement: Definition self-tag backfill subcommand
`engram vocab tag-definitions` SHALL be an explicit, idempotent one-shot subcommand that adds the missing `vocab/<term>` self-tag to every existing definition note, deriving each term from the `vocab-<term>-definition` slug. It MUST write through the vault's locked write path (refreshing each touched note's embedding sidecar so no sidecar is left stale), MUST leave already-self-tagged definitions untouched, MUST skip the family note (slug `vocab-definition`, empty term) reporting it as such, and MUST report per-note results (added vs already present vs skipped). The backfill MUST NOT run as a side effect of `engram update` or any other command.

#### Scenario: Backfill adds missing self-tags
- **WHEN** `engram vocab tag-definitions` runs against a vault whose definitions carry only the bare `vocab` marker
- **THEN** each definition note's `tags:` list gains `vocab/<term>` for its own term, its sidecar is refreshed (embed state clean), and the command reports one "added" line per touched note

#### Scenario: Backfill is idempotent
- **WHEN** `engram vocab tag-definitions` runs a second time over the same vault
- **THEN** no note content changes and every definition is reported as already present
