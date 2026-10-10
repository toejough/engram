# Vocabulary Engine - Technical Spec

## 792b20a Orientation

- Ownership: Repository component (caller of [vault-write-lock](./vault-write-lock.md), [durable-file-store](./durable-file-store.md),
  [luhmann-addressing](./luhmann-addressing.md), [note-document-format](./note-document-format.md), [text-embedder](./text-embedder.md), [embedding-sidecar](./embedding-sidecar.md) and
  [similarity-clustering](./similarity-clustering.md)).
- Responsibility: keeps the vault's controlled vocabulary. It recognizes definition notes and their
  terms, chooses each term's vector, tags a note with its closest terms, records the refit baseline and
  marks a refit due, reports vocabulary statistics, and carries out bootstrap, propose, refit, the
  self-tag backfill and the old-format migration, including writing definition notes, the family
  note's version and the vault's centroid record. It also tells `engram update` whether a vault needs
  either of those two repairs.
- Current realization (informational): `internal/cli/vocab.go` (tagging), `vocab_centroids.go`
  (`vocab.centroids.json`, term vectors, the two-pass retag), `vocab_trigger.go`,
  `vocab_commands.go` (the `Run*` commands, statistics, definition and family notes),
  `vocab_derive.go` (derivation and matching), `vocab_apply.go` (naming, applying, retiring),
  `vocab_regen.go`; `collectVaultStats`'s question helpers `countQAPairs` and `isQAQuestionFilename`
  in `qa.go`; `oldVocabFilesPresent` and `vocabDefinitionsMissingSelfTags` in `update.go`; and the
  `vocab` group in `targets.go`.
- Qualification: responsibility, keeping a vocabulary is stated without naming any function or seam;
  ownership, writers decide when a note is tagged and `engram update` decides when to report or
  migrate, while this component owns what a term is, which terms a note gets and how the term set is
  derived; independent verification, it is tested directly over injected dependencies; stability, the
  contract survives any refactoring of the commands that call it; necessity, every `vocab/` tag and
  definition note is written through it.
- Supported capabilities: [controlled-vocabulary](../product/controlled-vocabulary.md), note-writing's [792a3u Vocabulary tagging by kind](../product/note-writing.md#792a3u-vocabulary-tagging-by-kind) and [memory-query](../product/memory-query.md).

Rationale: tags are assigned at write time by cosine alone, so the write path needs no LLM and the
vocabulary's cost stays off retrieval. The `tags:` list is the only channel for terms ([792a6c Categorical tags share one list](../product/controlled-vocabulary.md#792a6c-categorical-tags-share-one-list)), and the
term set is derived from the vault's geometry ([792a6j Deriving the vocabulary](../product/controlled-vocabulary.md#792a6j-deriving-the-vocabulary)). The centroids live in a separate `vocab.centroids.json` rather than in
the definitions' sidecars, because a sidecar must stay a pure function of its note's content:
`engram embed apply` rebuilds sidecars from content and would overwrite a centroid stored there,
while a separate file survives it. The file carries the embedding model's ID so that centroids from
another model are ignored rather than mixed in.

## 792b20c Behavior Requirements

### 792b20c1 Definition recognition

Requirement: [792b20c1 Definition recognition](#792b20c1-definition-recognition) The vocabulary-engine MUST treat a note as a definition exactly when its
frontmatter `tags:` list, read as its LF form, contains the bare entry `vocab`, whatever the note's
filename or `type:`; MUST take a definition's term from its slug `vocab-<term>-definition` (the slug
being everything after the second `.` of `<id>.<date>.<slug>.md`), and none from the family slug
`vocab-definition`, a slug without that prefix and suffix, or an empty term; and MUST take a note's
terms from its `vocab/<term>` entries in order, block or inline, ignoring the bare entry and every
other tag.

Supports: [792a6a1b A definition is never assigned a term](../product/controlled-vocabulary.md#792a6a1-uat-scenarios), [792a6t1c Questions and definitions are not member notes](../product/controlled-vocabulary.md#792a6t1-uat-scenarios), [792a6u1a Backfill adds missing self-tags](../product/controlled-vocabulary.md#792a6u1-uat-scenarios).

Rationale: The bare `vocab` tag stays the single definition discriminator rather than a per-term slug. The
`vocab/<term>` self-tag is an Obsidian-graph affordance only. Tag nominations exclude definitions on both the
nominee side and the seed side, because the seed side derives terms from a cluster's top-3 delivered notes.
The backfill is an explicit subcommand, never a side effect of `engram update`.

#### 792b20c1a Scenarios

```gherkin
Scenario: 792b20c1a1 The bare tag marks a definition
  Given notes tagged [vocab], [vocab, vocab/cooking], [vocab/cooking] and [fam/x]
  When each is checked
  Then the first two are definitions and the others are not, whatever their filenames or types

Scenario: 792b20c1a2 The term comes from the slug
  Given the slugs vocab-retrieval-design-definition, vocab-definition, vocab--definition and notes-definition
  When each is parsed
  Then only the first yields a term, retrieval-design

Scenario: 792b20c1a3 A note's terms are its vocab/ entries
  Given a note tagged [fam/x, vocab, vocab/a, vocab/b], in block or inline form
  When its terms are read
  Then they are a and b, in that order
```

### 792b20c2 Term vectors

Requirement: [792b20c2 Term vectors](#792b20c2-term-vectors) The vocabulary-engine MUST take the vocabulary to be the terms of the
definition notes whose sidecar has a body vector, keeping the first definition of a term in the
vault's listing order, with that body vector as the term's vector; and, for tagging, MUST replace a
term's vector with its centroid from `vocab.centroids.json` when the file parses, its embedding model
ID is empty or matches the definitions' sidecars, and it holds a non-empty vector for the term. It
MUST fail when the vault cannot be listed.

Supports: [792a6d1a A learned note is tagged with its closest terms](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6q1a An unmatched proposed term is kept](../product/controlled-vocabulary.md#792a6q1-uat-scenarios).

#### 792b20c2a Scenarios

```gherkin
Scenario: 792b20c2a1 Each definition's sidecar gives its term's vector
  Given two definition notes with sidecars, the family note, and a definition without a sidecar
  When the term vectors are loaded
  Then there are two terms, each with its sidecar's body vector

Scenario: 792b20c2a2 The first definition of a term wins
  Given two definition notes for one term
  When the term vectors are loaded
  Then the term appears once, with the first note's vector

Scenario: 792b20c2a3 A stored centroid replaces the description vector
  Given terms x and y, and a centroid record from the same model holding only x
  When the tagging vectors are loaded
  Then x uses its centroid and y its sidecar vector; with a record from another model, or a malformed one, both use their sidecars
```

### 792b20c3 Term selection

Requirement: [792b20c3 Term selection](#792b20c3-term-selection) The vocabulary-engine MUST select, for a note's body vector, the terms whose
vector has a cosine similarity at or above the floor, at most three, ordered by similarity
descending and then by term name, and none when the vector or the term list is empty.

Supports: [792a3u1a A runbook is tagged](../product/note-writing.md#792a3u1-uat-scenarios), [792a3u1b The answer is tagged and the question is not](../product/note-writing.md#792a3u1-uat-scenarios), [792a6d1a A learned note is tagged with its closest terms](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6d1b A note close to no term gets no tag](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6d1c At most three terms are kept](../product/controlled-vocabulary.md#792a6d1-uat-scenarios).

Rationale: the cap of three and the 0.35 floor are set by [792a6d Write-time vocabulary tagging](../product/controlled-vocabulary.md#792a6d-write-time-vocabulary-tagging).

#### 792b20c3a Scenarios

```gherkin
Scenario: 792b20c3a1 Terms below the floor are dropped
  Given terms scoring 0.9, 0.5 and 0.2 against a note and a floor of 0.35
  When terms are selected
  Then the first two are returned; with every term below the floor, none is

Scenario: 792b20c3a2 At most three are kept, best first
  Given four terms all above the floor
  When terms are selected
  Then the three highest are returned in descending order

Scenario: 792b20c3a3 No vector or no terms gives no terms
  Given an empty body vector, or an empty term list
  When terms are selected
  Then none is returned
```

### 792b20c4 Rewriting a note's vocabulary tags

Requirement: [792b20c4 Rewriting a note's vocabulary tags](#792b20c4-rewriting-a-notes-vocabulary-tags) The vocabulary-engine MUST rewrite a note's `tags:` list so that its
`vocab/<term>` entries are exactly the given terms, after every other tag in its original order and
without any bare `vocab` entry; MUST keep the `tags:` key where it was, rendered as a block list with
four-space items, and remove the key when the list ends up empty; MUST leave every other frontmatter
line and the body unchanged, writing the note's LF form; and MUST return a note without frontmatter
unchanged. Rewriting the same terms again MUST change nothing, and the rewrite MUST NOT make the
note's sidecar stale.

Supports: [792a6c1a A categorical tag survives tagging](../product/controlled-vocabulary.md#792a6c1-uat-scenarios), [792a6c1b A categorical tag survives a refit](../product/controlled-vocabulary.md#792a6c1-uat-scenarios), [792a6d1a A learned note is tagged with its closest terms](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6d1b A note close to no term gets no tag](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6d1d An amend re-tags the note](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6u1a Backfill adds missing self-tags](../product/controlled-vocabulary.md#792a6u1-uat-scenarios).

#### 792b20c4a Scenarios

```gherkin
Scenario: 792b20c4a1 Vocab entries are replaced and other tags kept
  Given a note tagged [fam/x, vocab/old]
  When its terms are rewritten to [new]
  Then its tags are [fam/x, vocab/new], and rewriting [new] again changes nothing

Scenario: 792b20c4a2 An emptied list is removed
  Given a note tagged only [vocab/old]
  When its terms are rewritten to none
  Then the note has no tags: key; with another tag present, only the vocab/ entries go

Scenario: 792b20c4a3 The rest of the note is untouched
  Given a note whose tags: key sits between other keys, in block or inline form, and whose body mentions vocab/
  When its terms are rewritten
  Then the key stays in place as a block list and every other line, and the body, is byte-identical

Scenario: 792b20c4a4 A note without frontmatter is unchanged
  Given a note with no frontmatter block
  When its terms are rewritten
  Then it is returned unchanged

Scenario: 792b20c4a5 A tags-only rewrite keeps the sidecar current
  Given a note embedded before its tags are rewritten
  When its terms are rewritten
  Then its sidecar is still classified current
```

### 792b20c5 Tagging a written note

Requirement: [792b20c5 Tagging a written note](#792b20c5-tagging-a-written-note) When a writer asks it to tag a written note, the vocabulary-engine MUST load the
tagging vectors ([792b20c2 Term vectors](#792b20c2-term-vectors)) and the note's sidecar body vector, select terms with the 0.35 floor
([792b20c3 Term selection](#792b20c3-term-selection)), rewrite the tags ([792b20c4 Rewriting a note's vocabulary tags](#792b20c4-rewriting-a-notes-vocabulary-tags)) and write the note; it MUST write nothing when it is given no
way to read or write, the vault has no terms, the sidecar is unreadable or has no body vector, the
note is a definition, or the rewrite changes nothing or only the note's line endings; and a failed
write MUST be logged as a warning, not returned.

Supports: [792a3u1a A runbook is tagged](../product/note-writing.md#792a3u1-uat-scenarios), [792a3u1b The answer is tagged and the question is not](../product/note-writing.md#792a3u1-uat-scenarios), [792a6d1a A learned note is tagged with its closest terms](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6d1d An amend re-tags the note](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6d1e A resituated note is re-tagged](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6e1a A vault without a vocabulary writes plain notes](../product/controlled-vocabulary.md#792a6e1-uat-scenarios), [792a6e1b A note whose embedding failed is not tagged](../product/controlled-vocabulary.md#792a6e1-uat-scenarios), [792a6f1a Vocab tagging reads a CRLF note](../product/controlled-vocabulary.md#792a6f1-uat-scenarios).

#### 792b20c5a Scenarios

```gherkin
Scenario: 792b20c5a1 A written note is tagged
  Given a vault with a term close to a written note that has a sidecar
  When the note is tagged
  Then it is written with the term's vocab/ tag

Scenario: 792b20c5a2 Nothing to tag with writes nothing
  Given a writer with no term source wired, or a vault with no terms
  When a note is tagged
  Then the note is not written

Scenario: 792b20c5a3 A note without a sidecar is not tagged
  Given a written note whose sidecar is missing
  When it is tagged
  Then the note is not written

Scenario: 792b20c5a4 A definition is never tagged
  Given a definition note whose vector is close to a term
  When it is tagged
  Then it is not written

Scenario: 792b20c5a5 A CRLF note is tagged as LF and only when its tags change
  Given a note whose lines end in \r\n
  When it gains a term, or is tagged with the terms it already has
  Then the first is written with LF endings and the second is not written
```

### 792b20c6 Refit check

Requirement: [792b20c6 Refit check](#792b20c6-refit-check) When a writer asks whether a refit is due, the vocabulary-engine MUST do
nothing when it is given no way to list, read or write, or when `vocab.centroids.json` already marks a
refit pending; MUST otherwise count the vault's Markdown files that are not question notes and not
definitions (counting a file it cannot read); when no baseline is recorded (including when the file is
absent) MUST record that count and today's date as `last_refit` and stop; and when the count exceeds
the baseline's by at least 40 and at least 14 days have passed since its date MUST set
`refit_pending` with the reason `growth: <N> notes, <D> days`, keeping every other field. A baseline
whose date does not parse MUST never trigger, and neither the untagged rate nor a term's share of
notes MUST affect the check. A failed write MUST be logged as a warning.

Supports: [792a6s1a Growth marks a refit pending](../product/controlled-vocabulary.md#792a6s1-uat-scenarios), [792a6s1b Growth without enough time does not trigger](../product/controlled-vocabulary.md#792a6s1-uat-scenarios), [792a6s1c A first write records the baseline](../product/controlled-vocabulary.md#792a6s1-uat-scenarios), [792a6s1d A high untagged rate alone does not trigger](../product/controlled-vocabulary.md#792a6s1-uat-scenarios).

#### 792b20c6a Scenarios

```gherkin
Scenario: 792b20c6a1 The first check records the baseline
  Given a vault with no centroid record, or a record without a baseline
  When the refit check runs
  Then the note count and today's date are recorded as the baseline and no refit is pending; a vault that cannot be listed records 0

Scenario: 792b20c6a2 Growth past both thresholds marks a refit pending
  Given a baseline of 100 notes 20 days ago and 150 notes now
  When the refit check runs
  Then refit_pending is set with the reason "growth: 50 notes, 20 days"

Scenario: 792b20c6a3 Growth without the days does not trigger
  Given a baseline of 100 notes 5 days ago and 150 notes now
  When the refit check runs
  Then no refit is pending

Scenario: 792b20c6a4 Untagged rate and hubs never trigger
  Given a vault with a high untagged rate, or one term tagging most notes, and growth below 40
  When the refit check runs
  Then no refit is pending

Scenario: 792b20c6a5 A pending refit is left alone
  Given a centroid record already marking a refit pending
  When the refit check runs
  Then nothing is read beyond the record and nothing is written

Scenario: 792b20c6a6 Definitions and questions are not counted
  Given a vault whose definitions do or do not carry self-tags, and which holds question notes
  When the refit check runs, and when bootstrap records a baseline for the same vault
  Then both counts leave out the definitions and the questions and agree

Scenario: 792b20c6a7 A failed write is a warning
  Given a centroid record that cannot be written
  When the refit check needs to write it
  Then a warning is logged and the writer still succeeds

Scenario: 792b20c6a8 Nothing wired does nothing
  Given a writer with no way to list or write the vault
  When the refit check runs
  Then nothing is read or written
```

### 792b20c7 Statistics

Requirement: [792b20c7 Statistics](#792b20c7-statistics) For `engram vocab stats`, the vocabulary-engine MUST list the vault, take its
terms from the definition notes ([792b20c1 Definition recognition](#792b20c1-definition-recognition)), count as member notes the readable files with
frontmatter that are neither definitions nor question notes, count a member without `vocab/` tags as
untagged and every `vocab/<term>` entry toward its term, read the version from the family note
([792b20c9 The family note and the version](#792b20c9-the-family-note-and-the-version)) and the pending refit from `vocab.centroids.json` (none when it is absent), count the
question notes whose answer note exists, and print the report of [792a6t Vocabulary statistics](../product/controlled-vocabulary.md#792a6t-vocabulary-statistics). It MUST write nothing.

Supports: [792a6b1b The version follows propose and refit](../product/controlled-vocabulary.md#792a6b1-uat-scenarios), [792a6s1a Growth marks a refit pending](../product/controlled-vocabulary.md#792a6s1-uat-scenarios), [792a6s1b Growth without enough time does not trigger](../product/controlled-vocabulary.md#792a6s1-uat-scenarios), [792a6s1c A first write records the baseline](../product/controlled-vocabulary.md#792a6s1-uat-scenarios), [792a6s1d A high untagged rate alone does not trigger](../product/controlled-vocabulary.md#792a6s1-uat-scenarios), [792a6t1a Stats on an empty vault](../product/controlled-vocabulary.md#792a6t1-uat-scenarios), [792a6t1b Hubs and orphans are flagged](../product/controlled-vocabulary.md#792a6t1-uat-scenarios), [792a6t1c Questions and definitions are not member notes](../product/controlled-vocabulary.md#792a6t1-uat-scenarios), [792a6t1d A pending refit is the verdict](../product/controlled-vocabulary.md#792a6t1-uat-scenarios),
[792a6q1a An unmatched proposed term is kept](../product/controlled-vocabulary.md#792a6q1-uat-scenarios).

Rationale: question notes are recognized by their filename, `qa.<…>.q.md`, and their answer by the
same name ending `.a.md`.

#### 792b20c7a Scenarios

```gherkin
Scenario: 792b20c7a1 An empty vault reports zero
  Given an empty vault
  When the statistics are printed
  Then they show version 1.0, 0 terms, 0 member notes, 0 untagged, verdict: OK

Scenario: 792b20c7a2 Hubs, orphans and a high untagged rate are flagged
  Given terms tagging more than a quarter of the member notes, fewer than two notes, and a vault more than 8% untagged
  When the statistics are printed
  Then the terms carry [hub] and [orphan] and the rate carries [high], with verdict: OK

Scenario: 792b20c7a3 Definitions, questions and plain files are not members
  Given a vault with definitions, a question-and-answer pair and a file without frontmatter
  When the statistics are printed
  Then only the answer and the other notes count as members

Scenario: 792b20c7a4 The verdict follows the pending flag
  Given a centroid record marking a refit pending with a reason
  When the statistics are printed
  Then the verdict line is "verdict: REFIT_PENDING (<reason>)"

Scenario: 792b20c7a5 Question-and-answer pairs are counted
  Given 1 complete pair, a question without an answer, and then 20 complete pairs
  When the statistics are printed
  Then the first shows "qa pairs: 1" and "qa round-2 gate: accumulating (1/20)", the second "qa round-2 gate: READY (20>=20)"
```

### 792b20c8 Writing a definition note

Requirement: [792b20c8 Writing a definition note](#792b20c8-writing-a-definition-note) The vocabulary-engine MUST write a definition note at the next free top-level
Luhmann ID of the vault (counting the notes it has written in the same run), with today's date and
the fields of [792a6a Definition notes](../product/controlled-vocabulary.md#792a6a-definition-notes), rendered as a fact note whose body formula drops one trailing period from the
description so it ends in a single period, followed by an `Exemplars:` list when there are exemplars,
and MUST then build and write its sidecar. A failure to embed or to write the sidecar MUST be logged
as a warning, leaving the note; a failure to write the note MUST be logged as a warning and the
command MUST continue.

Supports: [792a6a1a A definition note carries its self-tag](../product/controlled-vocabulary.md#792a6a1-uat-scenarios), [792a6g1a Bootstrap seeds terms and tags existing notes](../product/controlled-vocabulary.md#792a6g1-uat-scenarios), [792a6i1a A proposed term is created](../product/controlled-vocabulary.md#792a6i1-uat-scenarios), [792a6o1a A named cluster becomes a term](../product/controlled-vocabulary.md#792a6o1-uat-scenarios).

#### 792b20c8a Scenarios

```gherkin
Scenario: 792b20c8a1 A definition takes the next top-level ID and its self-tag
  Given a vault whose highest top-level ID is 4
  When a definition for cooking is written
  Then it is 5.<date>.vocab-cooking-definition.md tagged [vocab, vocab/cooking], the bare tag first

Scenario: 792b20c8a2 The body ends in one period and carries the exemplars
  Given the description "recipes." and two exemplars
  When the definition is rendered
  Then its body sentence ends "recipes." once, its object: keeps "recipes.", and an Exemplars: list holds both exemplars

Scenario: 792b20c8a3 A failed embedding keeps the note
  Given an embedder that fails, or a sidecar that cannot be written
  When a definition is written
  Then the note exists without a sidecar and the command succeeds

Scenario: 792b20c8a4 A failed note write is a warning
  Given a definition note that cannot be written
  When it is written
  Then a warning is logged and the command continues
```

### 792b20c9 The family note and the version

Requirement: [792b20c9 The family note and the version](#792b20c9-the-family-note-and-the-version) The vocabulary-engine MUST find the family note as the definition whose slug
is `vocab-definition`; when it is absent and the command creates the vocabulary, MUST write it
([792b20c8 Writing a definition note](#792b20c8-writing-a-definition-note)) with `tags: [vocab]`, `vocab_version: "1.0"` and an object that names no term, reporting
whether it was written; MUST read the version from it, `1.0` when the vault cannot be listed, the
note is absent or the key is empty; MUST raise the minor or major part as [792a6b The family note](../product/controlled-vocabulary.md#792a6b-the-family-note) says, leaving a
version that is not `<major>.<minor>` unchanged; and MUST rewrite only that note's `vocab_version`
line, in place. When there is no family note, writing the version MUST fail, and the command MUST
log that as a warning and continue.

Supports: [792a6b1a Bootstrap creates the family note](../product/controlled-vocabulary.md#792a6b1-uat-scenarios), [792a6b1b The version follows propose and refit](../product/controlled-vocabulary.md#792a6b1-uat-scenarios), [792a6i1a A proposed term is created](../product/controlled-vocabulary.md#792a6i1-uat-scenarios), [792a6i1b Propose in a vault without a family note](../product/controlled-vocabulary.md#792a6i1-uat-scenarios), [792a6o1a A named cluster becomes a term](../product/controlled-vocabulary.md#792a6o1-uat-scenarios).

#### 792b20c9a Scenarios

```gherkin
Scenario: 792b20c9a1 The family note is created bare and names no term
  Given a vault without a family note and a vocabulary of several terms
  When the family note is ensured
  Then it is written tagged only vocab, and its text names none of the terms

Scenario: 792b20c9a2 A failed family write is reported as not written
  Given a family note that cannot be written
  When it is ensured
  Then it reports that no family note was written

Scenario: 792b20c9a3 The version is read with a default
  Given a family note carrying vocab_version "6.1", no family note, or a vault that cannot be listed
  When the version is read
  Then it is 6.1, 1.0 and 1.0

Scenario: 792b20c9a4 The version is raised in place
  Given a family note at "6.1" among other definitions, and a version that is not <major>.<minor>
  When the minor and the major versions are raised and written
  Then the family note alone reads "6.2" and then "7.0" with every other line unchanged, and the malformed version is returned unchanged

Scenario: 792b20c9a5 Writing a version without a family note fails
  Given a vault without a family note, or one that cannot be listed
  When the version is written
  Then it fails saying the family note was not found, or the listing failed
```

### 792b20c10 Bootstrap

Requirement: [792b20c10 Bootstrap](#792b20c10-bootstrap) For `engram vocab bootstrap`, the vocabulary-engine MUST read and parse the
seed file before taking the lock, failing with an error naming the read or parse failure; MUST then
hold the vault lock while it ensures the family note ([792b20c9 The family note and the version](#792b20c9-the-family-note-and-the-version)), writes a definition ([792b20c8 Writing a definition note](#792b20c8-writing-a-definition-note)) for
each seed term the vault has no definition for, loads the term vectors ([792b20c2 Term vectors](#792b20c2-term-vectors)) and, when there is
at least one, re-tags the vault ([792b20c11 Two-pass re-tagging](#792b20c11-two-pass-re-tagging)); MUST NOT write a `vocab.index.md`; and MUST print
`vocab bootstrap: <N> terms, <M> notes assigned`. A note it cannot read, or whose sidecar is missing,
MUST be skipped. `--floor` 0 MUST mean 0.35.

Supports: [792a6g1a Bootstrap seeds terms and tags existing notes](../product/controlled-vocabulary.md#792a6g1-uat-scenarios), [792a6g1b A second bootstrap creates nothing new](../product/controlled-vocabulary.md#792a6g1-uat-scenarios), [792a6h1a A bad seed is refused](../product/controlled-vocabulary.md#792a6h1-uat-scenarios).

#### 792b20c10a Scenarios

```gherkin
Scenario: 792b20c10a1 Bootstrap seeds the terms and tags the vault
  Given a vault holding a note close to a seed term
  When bootstrap runs with that seed
  Then the term's definition note and sidecar are written and the note is tagged with the term

Scenario: 792b20c10a2 A second bootstrap writes no new definition
  Given a vault already holding the seed's definitions and the family note
  When bootstrap runs again with the same seed
  Then no definition or family note is written

Scenario: 792b20c10a3 No index file is written
  Given any vault and seed
  When bootstrap runs
  Then no vocab.index.md is written

Scenario: 792b20c10a4 Unreadable notes are skipped
  Given a note without a sidecar and a sidecar without a note
  When bootstrap runs
  Then neither is written and bootstrap succeeds

Scenario: 792b20c10a5 A bad seed fails before the lock
  Given a seed file that cannot be read, or is not a YAML list of entries
  When bootstrap runs
  Then it fails naming the problem and writes nothing
```

### 792b20c11 Two-pass re-tagging

Requirement: [792b20c11 Two-pass re-tagging](#792b20c11-two-pass-re-tagging) To re-tag a vault from description vectors, the vocabulary-engine MUST read the
body vector of every note that is not a definition and has a readable sidecar, select each one's
terms against the description vectors ([792b20c3 Term selection](#792b20c3-term-selection)), set each term with members to the mean of its
members' vectors (keeping the description vector for a term with none), tag every note that is not a
question against those vectors ([792b20c4 Rewriting a note's vocabulary tags](#792b20c4-rewriting-a-notes-vocabulary-tags)), and write `vocab.centroids.json` with the means, their
member counts, the definitions' embedding model ID and dimensions, and the given baseline, with no
refit pending. Definitions MUST NOT contribute to any mean or be rewritten. A failure to list the
vault MUST be logged as a warning.

Supports: [792a6a1b A definition is never assigned a term](../product/controlled-vocabulary.md#792a6a1-uat-scenarios), [792a6g1a Bootstrap seeds terms and tags existing notes](../product/controlled-vocabulary.md#792a6g1-uat-scenarios), [792a6w1a An old-format vocabulary is migrated](../product/controlled-vocabulary.md#792a6w1-uat-scenarios).

#### 792b20c11a Scenarios

```gherkin
Scenario: 792b20c11a1 The second pass tags against member centroids
  Given a note just below the floor against a term's description but close to its members' mean
  When the vault is re-tagged
  Then the note is tagged with the term

Scenario: 792b20c11a2 Definitions are neither inputs nor targets
  Given definitions with self-tags whose vectors are close to their own terms, and an unreadable note
  When the vault is re-tagged
  Then no definition's vector enters a centroid, every definition's tags are byte-identical, and the unreadable note is skipped

Scenario: 792b20c11a3 The baseline is recorded
  Given a baseline of 3 notes dated today
  When the vault is re-tagged
  Then vocab.centroids.json records last_refit {note_count: 3, date: today}

Scenario: 792b20c11a4 A listing failure is a warning
  Given a vault that cannot be listed
  When the vault is re-tagged
  Then a warning is logged
```

### 792b20c12 Propose

Requirement: [792b20c12 Propose](#792b20c12-propose) For `engram vocab propose`, the vocabulary-engine MUST, holding the vault lock,
raise the minor version ([792b20c9 The family note and the version](#792b20c9-the-family-note-and-the-version)), write the term's definition ([792b20c8 Writing a definition note](#792b20c8-writing-a-definition-note)) with the source
`vocab lifecycle (propose v<version>)`, and record the term in `vocab.centroids.json` with
`origin: proposed`, keeping every other entry and field, keeping the term's existing vector and
member count or else taking its definition's sidecar vector (none when it is not embedded), and
creating the file with the definitions' model ID and dimensions when it is absent; and MUST print
`vocab propose: created <term> (version → <v>)`. The origin MUST be recorded even when writing the
definition fails.

Supports: [792a6i1a A proposed term is created](../product/controlled-vocabulary.md#792a6i1-uat-scenarios), [792a6i1b Propose in a vault without a family note](../product/controlled-vocabulary.md#792a6i1-uat-scenarios), [792a6q1a An unmatched proposed term is kept](../product/controlled-vocabulary.md#792a6q1-uat-scenarios).

#### 792b20c12a Scenarios

```gherkin
Scenario: 792b20c12a1 A proposed term is written and the minor version raised
  Given a vault at vocab_version "1.0"
  When gardening is proposed
  Then its definition note exists and the family note reads "1.1"

Scenario: 792b20c12a2 The proposed origin is recorded
  Given a centroid record holding other terms, or no record
  When gardening is proposed
  Then the record holds gardening with origin proposed and its sidecar vector, and every other entry unchanged

Scenario: 792b20c12a3 The origin survives a failed definition write
  Given a definition note that cannot be written
  When gardening is proposed
  Then the record still holds gardening with origin proposed
```

### 792b20c13 Deriving clusters

Requirement: [792b20c13 Deriving clusters](#792b20c13-deriving-clusters) For a refit, the vocabulary-engine MUST cluster the body vectors of every note
that is not a definition and has a readable sidecar, in name order, with the [similarity-clustering](./similarity-clustering.md)
([792b19c4 Choosing the number of clusters](./similarity-clustering.md#792b19c4-choosing-the-number-of-clusters)) over K from 2 to 40 with floor 0.02 and seed 1; when the previous derivation's K (the
number of existing terms not marked proposed, an absent origin counting as derived) is at least 2
and differs from the result, it MUST also cluster at exactly that K and keep that result when it has
clusters and its silhouette is within 0.02 of the first. Each cluster MUST carry its members' names
and their mean vector. It MUST also fingerprint the input as `n<count>-<SHA-256 over each name and its
vector's bits>`.

Supports: [792a6j1a The term count follows the vault's clusters](../product/controlled-vocabulary.md#792a6j1-uat-scenarios), [792a6j1b A vault without structure keeps its vocabulary](../product/controlled-vocabulary.md#792a6j1-uat-scenarios), [792a6m1a New clusters are emitted as naming requests](../product/controlled-vocabulary.md#792a6m1-uat-scenarios), [792a6n1a A bad naming answer is refused](../product/controlled-vocabulary.md#792a6n1-uat-scenarios).

#### 792b20c13a Scenarios

```gherkin
Scenario: 792b20c13a1 Clusters hold their members and mean vectors
  Given note vectors in separated groups
  When clusters are derived
  Then every note is in exactly one cluster, each cluster's vector is its members' mean, and K is above 0

Scenario: 792b20c13a2 No structure gives no clusters
  Given identical vectors, or fewer than two notes
  When clusters are derived
  Then K is 0

Scenario: 792b20c13a3 The previous K is kept on a near tie
  Given a previous K whose silhouette is within 0.02 of the best, or equal to the best K
  When clusters are derived
  Then the previous K's clustering is used

Scenario: 792b20c13a4 A clearly better K replaces the previous one
  Given a previous K more than 0.02 worse than the best, or one that finds no clusters
  When clusters are derived
  Then the best K's clustering is used
```

### 792b20c14 Matching clusters to terms

Requirement: [792b20c14 Matching clusters to terms](#792b20c14-matching-clusters-to-terms) The vocabulary-engine MUST pair clusters with existing terms by the cosine
similarity of a cluster's mean and a term's tagging vector, taking pairs at or above 0.80 from the
highest down (ties by term name, then cluster), each cluster and each term at most once; a cluster
left unpaired MUST be new, and an unpaired term MUST be retired unless its origin is proposed.

Supports: [792a6k1a A cluster keeps its term's name](../product/controlled-vocabulary.md#792a6k1-uat-scenarios), [792a6k1b Unmatched clusters and terms are reported](../product/controlled-vocabulary.md#792a6k1-uat-scenarios), [792a6q1a An unmatched proposed term is kept](../product/controlled-vocabulary.md#792a6q1-uat-scenarios).

#### 792b20c14a Scenarios

```gherkin
Scenario: 792b20c14a1 A close cluster keeps the term's name
  Given a cluster whose mean has cosine 0.9 with term a
  When clusters are matched
  Then the cluster is paired with a

Scenario: 792b20c14a2 The best pair is taken first
  Given two clusters both above 0.80 with term a, one closer
  When clusters are matched
  Then the closer cluster gets a and the other is new

Scenario: 792b20c14a3 Below the threshold, both sides are left
  Given a cluster and a derived term with cosine below 0.80
  When clusters are matched
  Then the cluster is new and the term retired, and every cluster is either paired or new and every term either paired, retired or proposed

Scenario: 792b20c14a4 A proposed term is never retired
  Given an unpaired term whose origin is proposed
  When clusters are matched
  Then it is not retired
```

### 792b20c15 Refit routing

Requirement: [792b20c15 Refit routing](#792b20c15-refit-routing) For `engram vocab refit`, the vocabulary-engine MUST hold the vault lock for the
whole run; after deriving ([792b20c13 Deriving clusters](#792b20c13-deriving-clusters)) and matching ([792b20c14 Matching clusters to terms](#792b20c14-matching-clusters-to-terms)), it MUST print the no-structure line
when K is 0, print the dry-run diff of [792a6l Dry run](../product/controlled-vocabulary.md#792a6l-dry-run) when `--dry-run` is given, print the naming requests
([792b20c16 Naming requests](#792b20c16-naming-requests)) when there are new clusters and no `--names`, fail when the `--names` file cannot be
read or its answer is refused ([792b20c17 Naming answers](#792b20c17-naming-answers)), and otherwise apply the derivation ([792b20c18 Applying a derivation](#792b20c18-applying-a-derivation)), writing
nothing on every path but the last.

Supports: [792a6j1a The term count follows the vault's clusters](../product/controlled-vocabulary.md#792a6j1-uat-scenarios), [792a6j1b A vault without structure keeps its vocabulary](../product/controlled-vocabulary.md#792a6j1-uat-scenarios), [792a6k1a A cluster keeps its term's name](../product/controlled-vocabulary.md#792a6k1-uat-scenarios), [792a6k1b Unmatched clusters and terms are reported](../product/controlled-vocabulary.md#792a6k1-uat-scenarios), [792a6l1a Dry run reports the derivation diff without writing](../product/controlled-vocabulary.md#792a6l1-uat-scenarios), [792a6m1a New clusters are emitted as naming requests](../product/controlled-vocabulary.md#792a6m1-uat-scenarios), [792a6n1a A bad naming answer is refused](../product/controlled-vocabulary.md#792a6n1-uat-scenarios), [792a6o1a A named cluster becomes a term](../product/controlled-vocabulary.md#792a6o1-uat-scenarios), [792a6o1b A refit with nothing to name applies at once](../product/controlled-vocabulary.md#792a6o1-uat-scenarios).

Rationale: refit rewrites tags across the whole vault, so no other writer may interleave with it.
The two-run naming protocol is sound only while the vault is unchanged between the runs, which the
fingerprint checks.

#### 792b20c15a Scenarios

```gherkin
Scenario: 792b20c15a1 No structure writes nothing
  Given a vault whose derivation finds no clusters
  When refit runs
  Then it prints the no-structure line and writes nothing

Scenario: 792b20c15a2 A dry run writes nothing
  Given a vault with a derivation to report
  When refit runs with --dry-run
  Then it prints the K, silhouette, matched, new and retired lines and writes nothing

Scenario: 792b20c15a3 New clusters without names emit requests
  Given a derivation with a new cluster
  When refit runs without --names
  Then it prints the naming requests and writes nothing

Scenario: 792b20c15a4 A bad or stale answer writes nothing
  Given a --names file that cannot be read, or one whose fingerprint predates a change to the vault
  When refit runs with it
  Then it fails and writes nothing

Scenario: 792b20c15a5 An accepted answer applies the derivation
  Given a derivation with a new cluster and a valid answer naming it
  When refit runs with --names
  Then the derivation is applied and the applied line printed
```

### 792b20c16 Naming requests

Requirement: [792b20c16 Naming requests](#792b20c16-naming-requests) The vocabulary-engine MUST build one naming request per new cluster with its
index, member count and, as exemplars, its three members nearest the cluster's mean, nearest first,
each with its basename, its slug as title (the basename when it has none) and the first 200 bytes of
its trimmed body as snippet, and MUST print them as indented JSON with the instruction and the
fingerprint ([792b20c13 Deriving clusters](#792b20c13-deriving-clusters)).

Supports: [792a6m1a New clusters are emitted as naming requests](../product/controlled-vocabulary.md#792a6m1-uat-scenarios).

#### 792b20c16a Scenarios

```gherkin
Scenario: 792b20c16a1 Exemplars are the members nearest the centroid
  Given a new cluster of five members at different distances from its mean
  When its naming request is built
  Then it carries the three nearest, nearest first, each with its title and snippet
```

### 792b20c17 Naming answers

Requirement: [792b20c17 Naming answers](#792b20c17-naming-answers) The vocabulary-engine MUST accept a naming answer only when it is JSON whose
fingerprint equals the current derivation's and whose entries each name a new cluster, none twice,
with a non-empty term and description, covering every new cluster, refusing it otherwise with the
error of [792a6n Naming answers](../product/controlled-vocabulary.md#792a6n-naming-answers); and each named cluster MUST become a term whose exemplars are its request's
snippets.

Supports: [792a6n1a A bad naming answer is refused](../product/controlled-vocabulary.md#792a6n1-uat-scenarios), [792a6o1a A named cluster becomes a term](../product/controlled-vocabulary.md#792a6o1-uat-scenarios).

#### 792b20c17a Scenarios

```gherkin
Scenario: 792b20c17a1 A complete answer is accepted
  Given new clusters 1 and 3 and an answer naming both with the right fingerprint
  When the answer is parsed
  Then both names are returned, for any such clusters and names

Scenario: 792b20c17a2 An incomplete, unknown or stale answer is refused
  Given an answer that omits a cluster, names an unknown cluster, or carries another fingerprint
  When the answer is parsed
  Then it is refused with the matching error

Scenario: 792b20c17a3 A named cluster becomes a term with its snippets
  Given a naming request with exemplar snippets and an answer naming its cluster
  When the named terms are built
  Then the term carries the answer's name and description and the snippets as exemplars
```

### 792b20c18 Applying a derivation

Requirement: [792b20c18 Applying a derivation](#792b20c18-applying-a-derivation) To apply a derivation, the vocabulary-engine MUST raise the major version
([792b20c9 The family note and the version](#792b20c9-the-family-note-and-the-version)), retire the retired terms ([792b20c19 Retiring terms](#792b20c19-retiring-terms)), write a definition for each named cluster
([792b20c8 Writing a definition note](#792b20c8-writing-a-definition-note)) with the source `vocab lifecycle (refit v<version>)`, re-tag every note that is not a
question ([792b20c4 Rewriting a note's vocabulary tags](#792b20c4-rewriting-a-notes-vocabulary-tags)) against the paired and named clusters' means and the unpaired proposed terms'
vectors with the 0.35 floor, and write `vocab.centroids.json` with each paired or named term's mean,
member count and `origin: derived`, each unpaired proposed term's vector with `origin: proposed`, the
derivation's K, silhouette and date, a new baseline, and no refit pending; and MUST print the applied
line of [792a6o Applying a refit](../product/controlled-vocabulary.md#792a6o-applying-a-refit). A field left unset MUST be omitted from the file.

Supports: [792a6c1b A categorical tag survives a refit](../product/controlled-vocabulary.md#792a6c1-uat-scenarios), [792a6o1a A named cluster becomes a term](../product/controlled-vocabulary.md#792a6o1-uat-scenarios), [792a6o1b A refit with nothing to name applies at once](../product/controlled-vocabulary.md#792a6o1-uat-scenarios), [792a6p1a A retired term is deleted with its references](../product/controlled-vocabulary.md#792a6p1-uat-scenarios), [792a6q1a An unmatched proposed term is kept](../product/controlled-vocabulary.md#792a6q1-uat-scenarios).

#### 792b20c18a Scenarios

```gherkin
Scenario: 792b20c18a1 The centroid record carries origins and the derivation
  Given an applied derivation with a derived and a proposed term
  When the centroid record is written and read back
  Then each term has its origin, member count and vector, the derivation's K, silhouette and date are recorded, and an absent origin or derivation is omitted from the file
```

### 792b20c19 Retiring terms

Requirement: [792b20c19 Retiring terms](#792b20c19-retiring-terms) To retire terms, the vocabulary-engine MUST delete each retired term's
definition note and sidecar; then, in every other note, MUST drop `supersedes:` entries whose `note:`
names a deleted file (and the key with its last entry), drop body lines starting `Supersedes: ` that
link a deleted note, and replace every other `[[<deleted>]]` or `[[<deleted>.md]]` with the plain
basename, leaving unchanged notes byte-identical; and MUST remove the retired `vocab/<term>` tags from
every note but definitions and questions. It MUST record no supersession, and a failure to list,
delete or write MUST be logged as a warning without stopping the rest.

Supports: [792a6p1a A retired term is deleted with its references](../product/controlled-vocabulary.md#792a6p1-uat-scenarios), [792a6p1b Retirement leaves no dangling body link](../product/controlled-vocabulary.md#792a6p1-uat-scenarios).

#### 792b20c19a Scenarios

```gherkin
Scenario: 792b20c19a1 The definition and its sidecar are deleted
  Given a retired term with a definition note and sidecar
  When it is retired
  Then both files are deleted and no note records a supersession of it

Scenario: 792b20c19a2 References to the deleted note are scrubbed
  Given notes whose supersedes: entry, Supersedes: line and body link name the deleted definition, one with no other supersedes: entry, and a note without frontmatter linking it
  When the term is retired
  Then the entries and lines are gone, the emptied supersedes: key is removed, each link reads as the plain basename, and untouched notes are byte-identical

Scenario: 792b20c19a3 Member tags are stripped, definitions untouched
  Given a member note tagged with the retired term and a definition note mentioning it
  When the term is retired
  Then the member loses the tag, the definition is unchanged, and a note not mentioning the term is not read for rewriting

Scenario: 792b20c19a4 Failures are warnings
  Given a vault that cannot be listed, or files that cannot be deleted or written
  When terms are retired
  Then a warning is logged for each failure and the rest continues
```

### 792b20c20 Self-tag backfill

Requirement: [792b20c20 Self-tag backfill](#792b20c20-self-tag-backfill) For `engram vocab tag-definitions`, the vocabulary-engine MUST list the vault,
then hold the vault lock while, for every definition note ([792b20c1 Definition recognition](#792b20c1-definition-recognition)), it appends the missing
`vocab/<term>` tag to the note's tags, keeping the rest of the note, and prints the per-note lines of
[792a6u Self-tag backfill](../product/controlled-vocabulary.md#792a6u-self-tag-backfill); a definition whose frontmatter cannot be split MUST print `<basename>: unparseable frontmatter
(skipped)`, and one that cannot be written MUST print `<basename>: write failed (<error>)` and
continue. It MUST NOT touch any sidecar.

Supports: [792a6u1a Backfill adds missing self-tags](../product/controlled-vocabulary.md#792a6u1-uat-scenarios), [792a6u1b Backfill is idempotent](../product/controlled-vocabulary.md#792a6u1-uat-scenarios).

#### 792b20c20a Scenarios

```gherkin
Scenario: 792b20c20a1 Missing self-tags are added once
  Given definitions with and without their self-tag, and the family note
  When the backfill runs twice
  Then the first run adds each missing tag and prints "added", the second changes nothing and prints "already present", and the family note is skipped both times

Scenario: 792b20c20a2 A failed write is reported
  Given a definition note that cannot be written
  When the backfill runs
  Then it prints "<basename>: write failed (<error>)" and continues
```

### 792b20c21 Migrating an old-format vocabulary

Requirement: [792b20c21 Migrating an old-format vocabulary](#792b20c21-migrating-an-old-format-vocabulary) For `engram update --regen-vocab`, the vocabulary-engine MUST list the vault;
harvest each `vocab.<term>.md` file other than `vocab.index.md` whose frontmatter has a `term:` as a
seed term with its `description:` and the bullets of its `Exemplars:` section, skipping files it
cannot read or parse; count the `vocab.<name>.md` files and the notes holding a `vocab:` frontmatter
line or a `Vocab:` body line; and report those counts and the number of seeds. When there is nothing
to migrate, or on a dry run, it MUST write nothing. Otherwise it MUST ensure the family note
([792b20c9 The family note and the version](#792b20c9-the-family-note-and-the-version)), write the seed terms' definitions ([792b20c8 Writing a definition note](#792b20c8-writing-a-definition-note)), remove the first `vocab:` and `Vocab:` lines
from those notes, delete the old files and, best effort, their sidecars (counting only the files
deleted and warning about the others), and re-tag the vault ([792b20c11 Two-pass re-tagging](#792b20c11-two-pass-re-tagging)), reporting the number of
assignments. A listing failure MUST be returned as an error.

Supports: [792a6w1a An old-format vocabulary is migrated](../product/controlled-vocabulary.md#792a6w1-uat-scenarios), [792a6w1b A dry run changes nothing](../product/controlled-vocabulary.md#792a6w1-uat-scenarios), [792a6w1c Nothing to migrate](../product/controlled-vocabulary.md#792a6w1-uat-scenarios).

#### 792b20c21a Scenarios

```gherkin
Scenario: 792b20c21a1 An old-format vocabulary is migrated
  Given a vault with vocab.recall.md, vocab.index.md and a note carrying a vocab: line and a Vocab: line
  When the migration runs
  Then the recall definition is written, both old files are deleted, the note keeps everything but those two lines, and the counts are 2 files, 1 note and 1 seed

Scenario: 792b20c21a2 A dry run writes nothing
  Given the same vault
  When the migration runs as a dry run
  Then the same counts are reported and nothing is written or deleted

Scenario: 792b20c21a3 Nothing to migrate writes nothing
  Given a vault with no old files and no legacy lines
  When the migration runs
  Then every count is 0 and nothing is written

Scenario: 792b20c21a4 An unparseable term file is skipped
  Given old term files that cannot be read, have no frontmatter or have an empty term:, beside one good one
  When the migration is previewed
  Then it seeds only the good term, and still counts all four files as old files to delete

Scenario: 792b20c21a5 A listing failure is an error
  Given a vault that cannot be listed
  When the migration runs
  Then it fails

Scenario: 792b20c21a6 A file that cannot be deleted is warned about and not counted
  Given an old file whose deletion fails
  When the old files are removed
  Then a warning names it and it is not counted as removed
```

### 792b20c22 Repairs a vault needs

Requirement: [792b20c22 Repairs a vault needs](#792b20c22-repairs-a-vault-needs) For `engram update`'s notices, the vocabulary-engine MUST report that a vault
holds old-format files when its directory holds a regular file named `vocab.<name>.md` (never
`vocab.centroids.json`), and that a definition lacks its self-tag when a file whose slug names a term
([792b20c1 Definition recognition](#792b20c1-definition-recognition)) is a definition whose tags lack `vocab/<term>`; and MUST report neither, without failing,
when the vault cannot be read.

Supports: [792a6v1a An old-format vault is flagged](../product/controlled-vocabulary.md#792a6v1-uat-scenarios), [792a6v1b A missing self-tag is flagged](../product/controlled-vocabulary.md#792a6v1-uat-scenarios), [792a6v1c A current vault gets no vocab notice](../product/controlled-vocabulary.md#792a6v1-uat-scenarios), [792a6w1a An old-format vocabulary is migrated](../product/controlled-vocabulary.md#792a6w1-uat-scenarios).

#### 792b20c22a Scenarios

```gherkin
Scenario: 792b20c22a1 Old-format files are detected
  Given vaults holding vocab.index.md, vocab.recall.md, only ordinary notes, or no vault at all
  When the vault is checked for old-format files
  Then only the first two report them

Scenario: 792b20c22a2 A missing self-tag is detected
  Given vaults with an untagged definition, only tagged definitions, only the family note, a member note, a definition slug without the bare tag, a mix with one untagged definition, and no vault at all
  When each vault is checked for missing self-tags
  Then only the vaults holding an untagged definition report one
```
