# Embedding Sidecar - Technical Spec

## 792b10a Orientation

- Ownership: Repository component.
- Responsibility: each note's `.vec.json` sidecar. It decides which text of a note is embedded and
  hashed, builds and writes the sidecar with two vectors, a model ID and a content hash, decodes
  and validates one, classifies a note's sidecar against the current model, and measures cosine
  similarity between vectors. It also runs the operator's census of a vault's sidecars and the
  re-embed of the notes whose sidecar is missing, stale, incompatible or broken.
- Current realization (informational): `internal/embed/sidecar.go` (`Sidecar` format,
  `SidecarPath`, `MarshalSidecar`, `UnmarshalSidecar`), `internal/embed/embedder.go`
  (`BuildSidecar`, `State`), `internal/embed/hash.go` (`SituationText`, `BodyText`, `ContentHash`),
  `internal/embed/state.go` (`ComputeState`), `internal/embed/cosine.go` (`Cosine`), and
  `autoEmbedNote` in `internal/cli/learn.go`, and `RunEmbedApply`, `RunEmbedStatus` and
  `selectStates` in `internal/cli/embed.go` (the operator's census and re-embed).
- Qualification: responsibility, the sidecar contract is stated without naming any writer or
  reader; ownership, callers decide when to build, read or classify a sidecar, while this component
  decides its content and what makes it current; independent verification, building, decoding,
  hashing and classification are tested directly over fake embedders and filesystems; stability,
  the on-disk format is versioned and outlives any refactoring; necessity, every retrieval over notes
  reads it.
- Supported capabilities: [vault-identity](../product/vault-identity.md), [note-writing](../product/note-writing.md), [note-revision](../product/note-revision.md), [chunk-indexing](../product/chunk-indexing.md)
  chunk-indexing, [controlled-vocabulary](../product/controlled-vocabulary.md), [memory-query](../product/memory-query.md), [vault-health](../product/vault-health.md).
  Vault-migration ([vault-migration](../product/vault-migration.md)) adds its mappings.

Rationale: semantic search needs vectors, but a vector database or a separate rebuild step is a
moving part that can drift from the notes. Instead every note carries its own sidecar, written as
part of the same command: there is no index to maintain or invalidate, and a copied vault is
self-contained. Each sidecar holds two vectors, one for the note's situation and one for its body,
so retrieval can match a query against whichever fits better. Notes of every kind embed this way;
the retired episode kind once embedded only its situation.

## 792b10c Behavior Requirements

### 792b10c1 Sidecar format

Requirement: [792b10c1 Sidecar format](#792b10c1-sidecar-format) The embedding-sidecar MUST store a sidecar as JSON with `schema_version` 1,
`embedding_model_id`, `dims`, `situation_vector`, `body_vector`, `content_hash`, and `last_used`
only when set. Decoding MUST refuse malformed JSON, MUST refuse any other schema version, such as an
old single-vector sidecar, before checking lengths, MUST refuse a sidecar whose either vector's
length differs from `dims`, and MUST read an absent `last_used` as empty.

Supports: [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a3u1a A runbook is tagged](../product/note-writing.md#792a3u1-uat-scenarios), [792a3u1b The answer is tagged and the question is not](../product/note-writing.md#792a3u1-uat-scenarios), [792a4n1a A note's use is recorded](../product/note-revision.md#792a4n1-uat-scenarios), [792a4n1d Amend records a use](../product/note-revision.md#792a4n1-uat-scenarios), [792a10c1a Candidates are emitted for review](../product/vault-migration.md#792a10c1-uat-scenarios).

Rationale: `last_used` is added metadata. It does not bump the schema version, and it is kept out of
the content hash, so recording a use never makes a note look stale.

#### 792b10c1a Scenarios

```gherkin
Scenario: 792b10c1a1 A dual-vector sidecar round-trips
  Given a sidecar with both vectors, a model ID, a dimension and a content hash
  When it is encoded and decoded
  Then every field reads back unchanged

Scenario: 792b10c1a2 last_used is optional
  Given a sidecar with and without last_used
  When each is encoded and decoded
  Then last_used round-trips when set, and a sidecar without it decodes with last_used empty

Scenario: 792b10c1a3 A vector of the wrong length is refused
  Given a sidecar whose situation or body vector length differs from dims
  When it is decoded
  Then the dims-mismatch error is returned

Scenario: 792b10c1a4 Malformed JSON is refused
  Given text that is not valid sidecar JSON
  When it is decoded
  Then the malformed-sidecar error is returned

Scenario: 792b10c1a5 An old single-vector sidecar is a schema error
  Given a sidecar written by the earlier single-vector schema
  When it is decoded
  Then the schema-version error is returned
```

### 792b10c2 Sidecar path

Requirement: [792b10c2 Sidecar path](#792b10c2-sidecar-path) The embedding-sidecar MUST place a note's sidecar beside it, named after the
note with `.md` replaced by `.vec.json`, and MUST append `.vec.json` to any other path.

Supports: [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a6a1a A definition note carries its self-tag](../product/controlled-vocabulary.md#792a6a1-uat-scenarios).

#### 792b10c2a Scenarios

```gherkin
Scenario: 792b10c2a1 A note's sidecar path
  Given the note path 1aa.note.md
  When its sidecar path is formed
  Then it is 1aa.note.vec.json

Scenario: 792b10c2a2 A non-Markdown path gets the suffix appended
  Given a path that does not end in .md
  When its sidecar path is formed
  Then .vec.json is appended to it
```

### 792b10c3 Embedding sources

Requirement: [792b10c3 Embedding sources](#792b10c3-embedding-sources) The embedding-sidecar MUST take a note's situation text from its
`situation:` frontmatter value, trimmed, for any kind of note, and its body text from the text after
the frontmatter, less one leading blank line, with every machine-written line removed (lines
starting `Supersedes:`, `Contributors:`, `Answered by:` or `Answers:`) and trailing blank lines
reduced to one newline. Other body lines MUST be kept. A note with no frontmatter is all body.

Supports: [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a3q1a A pair is captured](../product/note-writing.md#792a3q1-uat-scenarios), [792a4g1a A content amend re-embeds](../product/note-revision.md#792a4g1-uat-scenarios), [792a4g1b A provenance-only amend does not re-embed](../product/note-revision.md#792a4g1-uat-scenarios), [792a6d1a A learned note is tagged with its closest terms](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6j1a The term count follows the vault's clusters](../product/controlled-vocabulary.md#792a6j1-uat-scenarios).

Rationale: the machine-written lines are link channels rewritten in place by learn, amend and
curation. Leaving them out means a link-only change neither moves the body vector nor makes the
sidecar stale. Trailing blanks are normalized because the writers end bodies differently.

#### 792b10c3a Scenarios

```gherkin
Scenario: 792b10c3a1 The situation is read from the frontmatter
  Given notes of any kind with a situation: value, including after a value ending in --- and with surrounding whitespace
  When the situation text is taken
  Then it is the value, trimmed

Scenario: 792b10c3a2 The body is the text after the frontmatter
  Given a note with frontmatter followed by a blank line and a body, or a note without frontmatter
  When the body text is taken
  Then it is the body alone, or the whole text

Scenario: 792b10c3a3 Machine-written lines are left out
  Given a body with Supersedes:, Contributors:, Answered by: or Answers: lines
  When the body text is taken
  Then those lines are gone and the rest is unchanged

Scenario: 792b10c3a4 Ordinary lines are kept
  Given a body with a "Related to" section or a vocabulary line
  When the body text is taken
  Then those lines are kept

Scenario: 792b10c3a5 Trailing blank lines are normalized
  Given bodies that differ only in their trailing blank lines
  When their body text is taken
  Then each ends in exactly one newline
```

### 792b10c4 Content hash

Requirement: [792b10c4 Content hash](#792b10c4-content-hash) The embedding-sidecar MUST hash a note as `sha256:` followed by the hex
SHA-256 of its situation text, a zero byte, and its body text, so the hash changes when either
changes, does not change for any other frontmatter edit, and is the same for the same note.

Supports: [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a4g1a A content amend re-embeds](../product/note-revision.md#792a4g1-uat-scenarios), [792a4g1b A provenance-only amend does not re-embed](../product/note-revision.md#792a4g1-uat-scenarios), [792a6u1a Backfill adds missing self-tags](../product/controlled-vocabulary.md#792a6u1-uat-scenarios), [792a10i1c A CRLF note that is renamed becomes LF](../product/vault-migration.md#792a10i1-uat-scenarios).

#### 792b10c4a Scenarios

```gherkin
Scenario: 792b10c4a1 The hash covers situation and body
  Given a note with a situation and a body
  When it is hashed
  Then the hash is sha256: and the hex SHA-256 of the situation, a zero byte and the body

Scenario: 792b10c4a2 Changing either source changes the hash
  Given a note whose situation or body text changes, for any such texts
  When it is hashed again
  Then the hash differs

Scenario: 792b10c4a3 Other frontmatter does not change the hash
  Given a note whose triggers, tags or other non-situation frontmatter changes
  When it is hashed again
  Then the hash is the same

Scenario: 792b10c4a4 Hashing is repeatable
  Given any note
  When it is hashed twice
  Then both hashes are equal
```

### 792b10c5 Building a sidecar

Requirement: [792b10c5 Building a sidecar](#792b10c5-building-a-sidecar) The embedding-sidecar MUST build a sidecar by embedding the situation text,
or the body text when the note has no situation, and the body text, and stamping schema version 1,
the embedder's model ID and dimension, and the note's content hash; an embedding failure MUST be
returned.

Supports: [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a3n1b A failed embedding keeps the note](../product/note-writing.md#792a3n1-uat-scenarios), [792a4g1a A content amend re-embeds](../product/note-revision.md#792a4g1-uat-scenarios), [792a4g1c A failed re-embed keeps the amend](../product/note-revision.md#792a4g1-uat-scenarios), [792a4l1a A fact is resituated](../product/note-revision.md#792a4l1-uat-scenarios), [792a4l1c A failed sidecar rebuild fails the command](../product/note-revision.md#792a4l1-uat-scenarios), [792a4o1b An activation does not undo a re-embed](../product/note-revision.md#792a4o1-uat-scenarios), [792a6a1a A definition note carries its self-tag](../product/controlled-vocabulary.md#792a6a1-uat-scenarios), [792a10e1f Rewritten notes are not left stale](../product/vault-migration.md#792a10e1-uat-scenarios),
[792a10i1c A CRLF note that is renamed becomes LF](../product/vault-migration.md#792a10i1-uat-scenarios), [792a10l1d A CRLF note is converted](../product/vault-migration.md#792a10l1-uat-scenarios).

#### 792b10c5a Scenarios

```gherkin
Scenario: 792b10c5a1 Both vectors are embedded and stamped
  Given a note with a situation and a body, and an embedder
  When its sidecar is built
  Then the situation vector embeds the situation, the body vector the body, and the model ID, dimension and hash are stamped

Scenario: 792b10c5a2 A note without a situation embeds its body twice
  Given a note with no situation: field
  When its sidecar is built
  Then both vectors embed the body text
```

### 792b10c6 Writing a learned note's sidecar

Requirement: [792b10c6 Writing a learned note's sidecar](#792b10c6-writing-a-learned-notes-sidecar) After `engram learn` writes a note, the embedding-sidecar MUST build its
sidecar and atomically write it to the note's sidecar path. When building or writing fails, it MUST
log a warning naming the note and return without failing the command. With no embedder or no
sidecar writer wired, it MUST do nothing.

Supports: [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a3n1b A failed embedding keeps the note](../product/note-writing.md#792a3n1-uat-scenarios), [792a6e1b A note whose embedding failed is not tagged](../product/controlled-vocabulary.md#792a6e1-uat-scenarios).

Rationale: the note itself is already written in one atomic step, so a missing sidecar is
recoverable later by `engram embed apply --missing`.

#### 792b10c6a Scenarios

```gherkin
Scenario: 792b10c6a1 A learned note gets a valid sidecar
  Given a just-written note and an embedder
  When its sidecar is written
  Then a sidecar that decodes, with the embedder's model ID, is at its sidecar path

Scenario: 792b10c6a2 An embedding failure only warns
  Given an embedder that fails
  When the sidecar is written
  Then a warning naming the note is logged, and no error is returned

Scenario: 792b10c6a3 A write failure only warns
  Given a sidecar writer that fails
  When the sidecar is written
  Then a warning naming the note is logged, and no error is returned

Scenario: 792b10c6a4 Nothing is done without an embedder or writer
  Given no embedder, or no sidecar writer
  When the sidecar is written
  Then nothing is embedded or written
```

### 792b10c7 State classification

Requirement: [792b10c7 State classification](#792b10c7-state-classification) The embedding-sidecar MUST classify a note against the current model as
`broken` when the note cannot be read; `missing` when its sidecar does not exist; `broken` when the
sidecar cannot be read, is malformed or has a dims mismatch; `incompatible` when it has another
schema version or another model ID; `stale` when its content hash differs from the note's; and `ok`
otherwise, checking the model before the hash.

Supports: [792a2c1a A Markdown file in a subdirectory is not a note](../product/vault-identity.md#792a2c1-uat-scenarios), [792a2d1a A fresh vault has no notes](../product/vault-identity.md#792a2d1-uat-scenarios), [792a2d1b A user's README is a note](../product/vault-identity.md#792a2d1-uat-scenarios), [792a6u1a Backfill adds missing self-tags](../product/controlled-vocabulary.md#792a6u1-uat-scenarios), [792a9g1a Notes without sidecars are counted as without](../product/vault-health.md#792a9g1-uat-scenarios), [792a9g1b Every state is counted](../product/vault-health.md#792a9g1-uat-scenarios), [792a9h1a The default embeds only notes without a sidecar](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1b --stale embeds changed notes and not another model's sidecars](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1c --force replaces another model's sidecar](../product/vault-health.md#792a9h1-uat-scenarios),
[792a9h1f A broken sidecar is always replaced](../product/vault-health.md#792a9h1-uat-scenarios).

Rationale: a re-embed under a new model also picks up any content change, so a model mismatch takes
precedence. Every failure is a state rather than an error, so a pass over the whole vault never
stops early. All sidecars are expected to share one model ID. A query skips sidecars made by another
model with only an advisory, and fails only when none remains, so after a partial model change its
results can be silently thinner.

#### 792b10c7a Scenarios

```gherkin
Scenario: 792b10c7a1 A current sidecar is ok
  Given a note whose sidecar matches its content and the current model
  When it is classified
  Then it is ok

Scenario: 792b10c7a2 An absent sidecar is missing
  Given a note with no sidecar, whether the read error is fs.ErrNotExist, wraps it, or reports IsNotExist
  When it is classified
  Then it is missing, and an unrelated or nil error is not treated as absent

Scenario: 792b10c7a3 Changed content is stale
  Given a note whose body changed after its sidecar was written, including by adding a related-notes section
  When it is classified
  Then it is stale

Scenario: 792b10c7a4 Another model or schema is incompatible
  Given a sidecar made by another model, or by the old single-vector schema
  When it is classified
  Then it is incompatible

Scenario: 792b10c7a5 An unreadable sidecar is broken
  Given a sidecar that is malformed JSON or has a vector of the wrong length
  When it is classified
  Then it is broken

Scenario: 792b10c7a6 States have labels
  Given each state
  When it is printed
  Then it reads ok, missing, stale, incompatible or broken
```

### 792b10c8 Cosine similarity

Requirement: [792b10c8 Cosine similarity](#792b10c8-cosine-similarity) The embedding-sidecar MUST measure two vectors' cosine similarity, and MUST
return 0 when their lengths differ, when they are empty, or when either has zero magnitude.

Supports: [792a3t1a The answer is found and the question is not](../product/note-writing.md#792a3t1-uat-scenarios), [792a3u1a A runbook is tagged](../product/note-writing.md#792a3u1-uat-scenarios), [792a3u1b The answer is tagged and the question is not](../product/note-writing.md#792a3u1-uat-scenarios), [792a6d1a A learned note is tagged with its closest terms](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6d1c At most three terms are kept](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a10c1a Candidates are emitted for review](../product/vault-migration.md#792a10c1-uat-scenarios), [792a10c1b No pair is similar enough](../product/vault-migration.md#792a10c1-uat-scenarios).

#### 792b10c8a Scenarios

```gherkin
Scenario: 792b10c8a1 A vector is fully similar to itself
  Given any non-zero vector
  When its cosine with itself is measured
  Then it is 1

Scenario: 792b10c8a2 Orthogonal vectors are not similar
  Given two orthogonal vectors
  When their cosine is measured
  Then it is 0

Scenario: 792b10c8a3 Vectors without signal score 0
  Given vectors of different lengths, or a zero vector
  When their cosine is measured
  Then it is 0
```

### 792b10c9 CRLF notes

Requirement: [792b10c9 CRLF notes](#792b10c9-crlf-notes) The embedding-sidecar MUST take the situation text, the body text and the
content hash of a note whose lines end in `\r\n` from its LF form, so that each equals the value
for the note's LF form.

Supports: [792a4m1a Amend converts a CRLF note](../product/note-revision.md#792a4m1-uat-scenarios), [792a9j1a A CRLF note is not stale](../product/vault-health.md#792a9j1-uat-scenarios), [792a9j1b A CRLF note gets its LF form's sidecar](../product/vault-health.md#792a9j1-uat-scenarios).

Rationale: a CRLF note used to be embedded from the wrong text, and a conversion to LF then changed
its hash. Reading the LF form makes the conversion that a revision makes leave the hash as it
would be for the LF note.

#### 792b10c9a Scenarios

```gherkin
Scenario: 792b10c9a1 A CRLF note embeds and hashes as its LF form
  Given a note whose lines end in \r\n, and the same note with LF line endings
  When their situation text, body text and content hash are taken
  Then each equals the LF note's
```

### 792b10c10 Embedding census

Requirement: [792b10c10 Embedding census](#792b10c10-embedding-census) The embedding-sidecar MUST, for a census, scan the vault, classify each scanned
note ([792b10c7 State classification](#792b10c7-state-classification)) and print six lines: `total:` and the number of notes, then `with-embeddings:` (ok),
`without:` (missing), `stale:`, `incompatible:` and `broken:` with their numbers, each label padded
so the numbers align. It MUST return a scan failure as `embed status: scan: <error>`.

Supports: [792a9g1a Notes without sidecars are counted as without](../product/vault-health.md#792a9g1-uat-scenarios), [792a9g1b Every state is counted](../product/vault-health.md#792a9g1-uat-scenarios), [792a9g1c An unreadable note stops the census](../product/vault-health.md#792a9g1-uat-scenarios), [792a9j1a A CRLF note is not stale](../product/vault-health.md#792a9j1-uat-scenarios).

#### 792b10c10a Scenarios

```gherkin
Scenario: 792b10c10a1 Notes without sidecars are counted as without
  Given a vault of two notes and no sidecar
  When the census is taken
  Then the output has "total:           2", "with-embeddings: 0" and "without:         2"

Scenario: 792b10c10a2 Every state is counted
  Given notes that are ok, stale, incompatible and broken
  When the census is taken
  Then each of those counts is 1 and the total is 4

Scenario: 792b10c10a3 A scan failure is returned
  Given a vault whose scan fails
  When the census is taken
  Then it fails with an error beginning "embed status: scan:"
```

### 792b10c11 Embedding selection

Requirement: [792b10c11 Embedding selection](#792b10c11-embedding-selection) The embedding-sidecar MUST select, with no mode flag or with the missing flag,
the notes whose state is missing; with the stale flag, the stale notes; with the force flag, the
incompatible notes; and with the all flag, every note. Mode flags other than all MUST combine, and all
MUST select every state. It MUST select a note whose state is broken under every selection.

Supports: [792a9h1a The default embeds only notes without a sidecar](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1b --stale embeds changed notes and not another model's sidecars](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1c --force replaces another model's sidecar](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1d --all embeds every note](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1e Mode flags combine](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1f A broken sidecar is always replaced](../product/vault-health.md#792a9h1-uat-scenarios).

Rationale: a broken sidecar cannot be classified against the content, and no selection should leave
it, so every selection takes it. The force flag is separate from the stale flag because replacing
another model's sidecars changes the model a vault runs under ([792b10c7 State classification](#792b10c7-state-classification)).

#### 792b10c11a Scenarios

```gherkin
Scenario: 792b10c11a1 No flag selects the missing notes
  Given no mode flag
  When each state is tested for selection
  Then missing and broken are selected and ok, stale and incompatible are not

Scenario: 792b10c11a2 The all flag selects every state
  Given the all flag
  When each state is tested for selection
  Then every state is selected

Scenario: 792b10c11a3 The stale flag selects stale and broken
  Given the stale flag
  When each state is tested for selection
  Then stale and broken are selected and the others are not

Scenario: 792b10c11a4 The force flag selects incompatible and broken
  Given the force flag
  When each state is tested for selection
  Then incompatible and broken are selected and the others are not

Scenario: 792b10c11a5 The missing flag is the explicit default
  Given the missing flag
  When each state is tested for selection
  Then missing and broken are selected and the others are not

Scenario: 792b10c11a6 Mode flags combine
  Given the stale and force flags
  When each state is tested for selection
  Then stale, incompatible and broken are selected and missing and ok are not
```

### 792b10c12 Embedding run

Requirement: [792b10c12 Embedding run](#792b10c12-embedding-run) The embedding-sidecar MUST, for each selected note in ascending basename order,
read the note, build its sidecar ([792b10c5 Building a sidecar](#792b10c5-building-a-sidecar)), write it atomically beside the note and print
`embedded  <note>.md (<state>)`. With the dry-run flag it MUST print `would-embed <note>.md (<state>)`
and write nothing. It MUST print `skip      <note>.md: read error: <error>` for a note it
cannot read, and `fail      <note>.md: embed: <error>` or `fail      <note>.md: write: <error>` for an
embedding or a write that fails, and MUST continue with the remaining notes and return no error
for them. It MUST write nothing but sidecars, and MUST return a scan failure as `embed apply: scan:
<error>`.

Supports: [792a9h1a The default embeds only notes without a sidecar](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1b --stale embeds changed notes and not another model's sidecars](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1c --force replaces another model's sidecar](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1d --all embeds every note](../product/vault-health.md#792a9h1-uat-scenarios), [792a9h1f A broken sidecar is always replaced](../product/vault-health.md#792a9h1-uat-scenarios), [792a9i1a A missing sidecar is written](../product/vault-health.md#792a9i1-uat-scenarios), [792a9i1b A dry run writes nothing](../product/vault-health.md#792a9i1-uat-scenarios), [792a9i1c Only sidecars are written](../product/vault-health.md#792a9i1-uat-scenarios), [792a9i1d A failed write is reported and the run goes on](../product/vault-health.md#792a9i1-uat-scenarios),
[792a9i1e Notes are embedded in order](../product/vault-health.md#792a9i1-uat-scenarios), [792a9i1f An unreadable note stops the run](../product/vault-health.md#792a9i1-uat-scenarios), [792a9j1b A CRLF note gets its LF form's sidecar](../product/vault-health.md#792a9j1-uat-scenarios).

Rationale: a sidecar is a derived file, not part of the note's git-tracked content. It is recomputed
from the current note, so its content hash stays honest, and it is never stored in or versioned with
the note file. No lock is taken: a sidecar write does not collide with Luhmann ID assignment.

#### 792b10c12a Scenarios

```gherkin
Scenario: 792b10c12a1 A missing sidecar is written
  Given a note with no sidecar
  When the run embeds the missing notes
  Then the output has "embedded  <note>.md (missing)" and the sidecar holds the schema version, model ID, dims, both vectors and a sha256 content hash

Scenario: 792b10c12a2 A dry run writes nothing
  Given a note with no sidecar
  When the run is a dry run
  Then the output has "would-embed <note>.md (missing)" and no sidecar is written

Scenario: 792b10c12a3 The stale flag leaves another model's sidecar and the force flag replaces it
  Given a note whose sidecar names another model
  When the run uses the stale flag, then the force flag
  Then the first leaves the sidecar unchanged and the second rewrites it under the current model

Scenario: 792b10c12a4 A failed write is reported
  Given a note whose sidecar cannot be written
  When the run embeds it
  Then the output has a line beginning "fail      " for the note and containing the write error, and no error is returned

Scenario: 792b10c12a5 A failed embedding is reported
  Given an embedder that fails
  When the run embeds a note
  Then the output has a line beginning "fail      " for the note, and no error is returned

Scenario: 792b10c12a6 A stale and a broken note are both embedded
  Given a stale note and a note with a malformed sidecar
  When the run uses the stale flag
  Then both are embedded, printing their states

Scenario: 792b10c12a7 A note that cannot be read is skipped
  Given a selected note that cannot be read
  When the run embeds it
  Then the output has a line beginning "skip      " for the note and the run goes on

Scenario: 792b10c12a8 A scan failure is returned
  Given a vault whose scan fails
  When the run starts
  Then it fails with an error beginning "embed apply: scan:" and writes nothing

Scenario: 792b10c12a9 Notes are embedded in ascending order
  Given several selected notes
  When the run embeds them
  Then their lines are in ascending basename order

Scenario: 792b10c12a10 Only sidecars are written
  Given selected notes
  When the run embeds them
  Then no note file changes and every file written is a sidecar
```
