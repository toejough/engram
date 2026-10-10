# Note Reparenting - Technical Spec

## 792b27a Orientation

- Ownership: Repository component.
- Responsibility: moves existing notes to new Luhmann IDs, and keeps the vault consistent with the move.
  It proposes which top-level notes may belong under another, takes the answers an agent gives, gives each
  answered note its new ID and basename, and rewrites every link to the old basename. It also renames and
  rewrites for any caller that needs it, as skill adoption does. It recognizes a vault that holds only
  top-level notes.
- Current realization (informational): `internal/cli/luhmann_reparent.go` (`RenameAndRewriteReferences`,
  `RebuildNoteSidecars`, the pre-flight and the link rewriting) and `internal/cli/luhmann_reparent_apply.go`
  (`RunReparentLuhmann`, derive, apply and preview); `vaultHasOnlyTopLevelNotes` in
  `internal/cli/update.go`.
- Qualification: responsibility, re-parenting is stated without naming any command; ownership, the
  `update` command decides when a flag selects it and how a notice reaches the user, while this
  component decides what is proposed, renamed and rewritten; independent verification, it is tested
  directly over fake listing, reading, writing and renaming; stability, a vault whose IDs must change
  without breaking its links is a lasting need; necessity, `engram update --reparent-luhmann` is this
  contract.
- Supported capabilities: [vault-migration](../product/vault-migration.md), [parent-sync](../product/parent-sync.md) (the alias, [792b27c6 Rename and link rewrite](#792b27c6-rename-and-link-rewrite)). Install-and-update ([install-and-update](../product/install-and-update.md))
  extends it with the command's routing and notices.

Rationale: whether one note is a sub-point of another is a judgment of content that embeddings cannot
make, so the binary proposes candidates by similarity, an agent judges them, and the binary applies the
judgments. The fingerprint stops answers from being applied to a vault that has changed since they were
derived. The rename, the link rewrite and the chunk-index update happen in one invocation so that a
successful apply never leaves a manual `engram ingest --auto` or `engram prune`. A failure of the chunk
index step leaves the already-completed rename in place, because the vault is the authority and the index
can be rebuilt from it.

## 792b27c Behavior Requirements

### 792b27c1 Flat-vault detection

Requirement: [792b27c1 Flat-vault detection](#792b27c1-flat-vault-detection) The note-reparenting MUST report that a vault holds only top-level notes when it holds
at least one note file at its root and the Luhmann ID that leads every note file's name has one segment.
It MUST report false for a vault with no note file, a vault with a note whose ID has more than one segment
or does not parse, and a vault directory it cannot read, and MUST skip a file whose name is not a note
name, such as a sidecar.

Supports: [792a10a1a Only top-level notes](../product/vault-migration.md#792a10a1-uat-scenarios), [792a10a1b A branched note silences the notice](../product/vault-migration.md#792a10a1-uat-scenarios), [792a10a1c An empty vault has no notice](../product/vault-migration.md#792a10a1-uat-scenarios).

Rationale: a detection failure must never fail `engram update`'s main job, so every uncertain case reads as
"not flat", and an ID that does not parse counts as branched so a malformed vault is never called flat.

#### 792b27c1a Scenarios

```gherkin
Scenario: 792b27c1a1 Every note is top-level
  Given note files 1.2026-07-01.first-note.md and 2.2026-07-02.second-note.md
  When the vault is tested
  Then it holds only top-level notes; and a sidecar beside a note is skipped, as is a directory

Scenario: 792b27c1a2 One branched note is enough
  Given note files 1.2026-07-01.first-note.md and 1a.2026-07-02.branch-note.md
  When the vault is tested
  Then it does not hold only top-level notes

Scenario: 792b27c1a3 An empty vault is not flat
  Given a vault with no note file
  When the vault is tested
  Then it does not hold only top-level notes

Scenario: 792b27c1a4 An unreadable vault is not flat
  Given a vault directory that cannot be read
  When the vault is tested
  Then it does not hold only top-level notes

Scenario: 792b27c1a5 An ID that does not parse is not top-level
  Given a note file whose leading ID does not parse
  When the vault is tested
  Then it does not hold only top-level notes
```

### 792b27c2 Candidate selection

Requirement: [792b27c2 Candidate selection](#792b27c2-candidate-selection) The note-reparenting MUST consider each note at the vault root whose Luhmann ID has one
segment and whose sidecar yields a non-empty body vector, in ascending filename order. For each, it MUST keep
as candidates the up to three other considered notes whose body vectors have cosine similarity ([792b10c8 Cosine similarity](./embedding-sidecar.md#792b10c8-cosine-similarity)) of at
least 0.75 with its own, nearest first, each carrying its ID, an excerpt of its body of at most 200 bytes
(empty when the note cannot be read), the target's ID and excerpt, and the similarity.

Supports: [792a10c1a Candidates are emitted for review](../product/vault-migration.md#792a10c1-uat-scenarios), [792a10c1b No pair is similar enough](../product/vault-migration.md#792a10c1-uat-scenarios), [792a10c1d Only top-level notes with sidecars are considered](../product/vault-migration.md#792a10c1-uat-scenarios).

#### 792b27c2a Scenarios

```gherkin
Scenario: 792b27c2a1 Similar top-level notes become candidates
  Given top-level notes with sidecars whose vectors are similar
  When candidates are derived
  Then each note is paired with its up to three nearest similar notes, with their excerpts and similarity

Scenario: 792b27c2a2 Nothing below the floor is a candidate
  Given top-level notes none of whose pairs has a similarity of 0.75
  When candidates are derived
  Then there are no candidates

Scenario: 792b27c2a3 An unreadable note has an empty excerpt
  Given a candidate note whose file cannot be read
  When its excerpt is taken
  Then the excerpt is empty

Scenario: 792b27c2a4 Branched notes and notes without a sidecar are left out
  Given a branched note and a note without a usable sidecar
  When candidates are derived
  Then neither is a candidate or a target

Scenario: 792b27c2a5 An excerpt is capped
  Given a note whose body is longer than 200 bytes
  When its excerpt is taken
  Then it is the first 200 bytes of the trimmed body
```

### 792b27c3 Derive output

Requirement: [792b27c3 Derive output](#792b27c3-derive-output) The note-reparenting MUST, with no answers file, write nothing to the vault and print
either `update --reparent-luhmann: no candidates found` or a JSON document with the candidates, a
`fingerprint` that is the vocabulary derivation fingerprint ([792b20c13 Deriving clusters](./vocabulary-engine.md#792b20c13-deriving-clusters)) over the considered notes' names and
vectors, a `next_command` of `engram update --reparent-luhmann --answers <path-to-answers-file>`, and an
`instruction` that names the learn skill's batch mode, states the answers-file shape
`{"reparenting":[{"note","position","target"}],"fingerprint"}`, requires one entry per distinct candidate
note, sends the finished file back to the user or orchestrating agent to preview with `next_command` and
`--dry-run` and then apply, and does not tell the answering pass to run apply.

Supports: [792a10c1a Candidates are emitted for review](../product/vault-migration.md#792a10c1-uat-scenarios), [792a10c1b No pair is similar enough](../product/vault-migration.md#792a10c1-uat-scenarios), [792a10c1c The instruction routes to the learn skill](../product/vault-migration.md#792a10c1-uat-scenarios).

#### 792b27c3a Scenarios

```gherkin
Scenario: 792b27c3a1 The payload carries candidates, fingerprint and next command
  Given candidates
  When the derive output is printed
  Then it is JSON holding the candidates, a fingerprint, a next_command and an instruction

Scenario: 792b27c3a2 No candidates is a message
  Given no candidate
  When the derive output is printed
  Then it is "update --reparent-luhmann: no candidates found"

Scenario: 792b27c3a3 The instruction routes to the learn skill's batch mode
  Given candidates
  When the derive output is printed
  Then the instruction names the batch mode, requires one entry per distinct candidate note, names --dry-run as the preview, and does not tell the answering pass to run apply

Scenario: 792b27c3a4 The next command is literal
  Given candidates
  When the derive output is printed
  Then next_command is "engram update --reparent-luhmann --answers <path-to-answers-file>"

Scenario: 792b27c3a5 Deriving writes nothing
  Given a vault
  When candidates are derived, with or without the dry-run flag
  Then no file is written
```

### 792b27c4 Answers validation

Requirement: [792b27c4 Answers validation](#792b27c4-answers-validation) The note-reparenting MUST read the answers file and return, writing nothing, an error
when it cannot be read (`reading --answers`), when it is not valid JSON (`cannot parse --answers JSON`), when
its `fingerprint` differs from the fingerprint of the vault's current top-level notes (the stale error, which
says to re-run `engram update --reparent-luhmann`), and when an answer names a note ID that no note holds
(`--answers references an unknown note`).

Supports: [792a10d1a A missing answers file](../product/vault-migration.md#792a10d1-uat-scenarios), [792a10d1b Malformed answers](../product/vault-migration.md#792a10d1-uat-scenarios), [792a10d1c Stale answers](../product/vault-migration.md#792a10d1-uat-scenarios), [792a10d1d An unknown note](../product/vault-migration.md#792a10d1-uat-scenarios).

#### 792b27c4a Scenarios

```gherkin
Scenario: 792b27c4a1 An unreadable answers file
  Given an answers path that cannot be read
  When the answers are applied
  Then the error says the answers file cannot be read

Scenario: 792b27c4a2 Malformed answers
  Given answers that are not valid JSON
  When they are applied
  Then the error says it cannot parse --answers JSON

Scenario: 792b27c4a3 Stale answers
  Given answers with a fingerprint that is not the vault's current one
  When they are applied
  Then they are refused with the stale-answers error and nothing is written

Scenario: 792b27c4a4 An unknown note
  Given an answer naming a note ID that no note holds
  When the answers are applied
  Then the error says --answers references an unknown note
```

### 792b27c5 Rename map

Requirement: [792b27c5 Rename map](#792b27c5-rename-map) The note-reparenting MUST build the old-to-new basename map from the answers, taking
them in ascending numeric order of the answered note IDs, with an ID that is not numeric after the numeric
ones in string order. It MUST skip an answer whose position is `top`, give every other answer the ID that the
Luhmann allocator ([792b7c4 Next-ID allocation](./luhmann-addressing.md#792b7c4-next-id-allocation)) allocates for its position and target against the existing IDs and the IDs
allocated earlier in the run, and keep the note's date and slug in the new basename.

Supports: [792a10e1a A continuation answer renames the note and rewrites links](../product/vault-migration.md#792a10e1-uat-scenarios), [792a10e1b Renames are ordered by original ID](../product/vault-migration.md#792a10e1-uat-scenarios), [792a10e1d A top answer changes nothing](../product/vault-migration.md#792a10e1-uat-scenarios).

#### 792b27c5a Scenarios

```gherkin
Scenario: 792b27c5a1 Answers are taken in numeric order
  Given IDs 2 and 10, and IDs 2 and x
  When they are ordered
  Then 2 comes before 10, and the non-numeric x comes after 2

Scenario: 792b27c5a2 A continuation takes the next child ID
  Given note 12 answered continuation of note 7, and note 20 linking note 12
  When the answers are applied
  Then note 12 is renamed to the next child ID under 7 with the same date and slug

Scenario: 792b27c5a3 Two children are allocated in order
  Given notes 2 and 3 both answered continuation of note 1
  When the rename map is built
  Then note 2 maps to 1a and note 3 maps to 1b

Scenario: 792b27c5a4 A top answer is skipped
  Given an answer with position top
  When the rename map is built
  Then the note is not in the map
```

### 792b27c6 Rename and link rewrite

Requirement: [792b27c6 Rename and link rewrite](#792b27c6-rename-and-link-rewrite) The note-reparenting MUST, given a rename map, do nothing when it is empty, and otherwise
rename each noted note's file and its sidecar (when it has one), write its content with `luhmann:` set to the
new ID and the old basename appended to its `aliases:` (and the new basename removed from them), and rewrite,
in every note's text, every `[[old]]` and `[[old.md]]` to `[[new]]` in the body and in every frontmatter
string, and the `note:` of each entry of the `supersedes:` list (with or without `.md`, keeping the form),
using the full map in one pass so that a link from a renamed note to another renamed note names the other's
new basename. It MUST NOT rewrite a `note:` outside the `supersedes:` list, MUST write a note that neither is
renamed nor links a renamed note not at all, MUST keep a renamed note's body when it has no frontmatter, and
leave a renamed note with no `luhmann:` key without one. It MUST return the paths of the notes whose
references it rewrote and of the renamed notes it converted, and MUST return a failure to list, read, rename
or write a note naming it.

Supports: [792a10e1a A continuation answer renames the note and rewrites links](../product/vault-migration.md#792a10e1-uat-scenarios), [792a10e1c Cascading renames resolve to the final names](../product/vault-migration.md#792a10e1-uat-scenarios), [792a10e1e Only supersedes entries are rewritten in a frontmatter list](../product/vault-migration.md#792a10e1-uat-scenarios), [792a10e1g An untouched note is not written](../product/vault-migration.md#792a10e1-uat-scenarios).

Rationale: a `parent:` link's `note:` names a note in another vault, never a local reference, so a local
rename must leave it alone. A basename uses only `[a-z0-9.-]`, so swapping one for another inside a plain,
quoted or double-quoted scalar needs no escaping and cannot break the scalar's quoting.

#### 792b27c6a Scenarios

```gherkin
Scenario: 792b27c6a1 A single reference is rewritten
  Given a note linking a renamed note once
  When the rename is applied
  Then the link names the new basename

Scenario: 792b27c6a2 References across notes are rewritten
  Given several notes linking renamed notes, in the body and in frontmatter
  When the rename is applied
  Then every one names the new basename

Scenario: 792b27c6a3 Cascading renames use the final map
  Given two renamed notes that link each other
  When the rename is applied
  Then each note's link names the other's new basename, never an intermediate one

Scenario: 792b27c6a4 Only supersedes entries are rewritten
  Given a supersedes: entry and a parent link naming a renamed basename
  When the rename is applied
  Then the supersedes: entry is rewritten and the parent link is not

Scenario: 792b27c6a5 Frontmatter links are rewritten
  Given links to a renamed note in a frontmatter string field and in a Supersedes: line
  When the rename is applied
  Then they name the new basename

Scenario: 792b27c6a6 The luhmann value is replaced whatever its shape
  Given a luhmann: value that is plain, quoted, or a block or multi-line scalar
  When the note is renamed
  Then it is one luhmann: "<new id>" line and the frontmatter still decodes

Scenario: 792b27c6a7 The old basename is recorded as an alias
  Given a renamed note, with and without an aliases: list in any YAML sequence style
  When the rename is applied
  Then its aliases: includes the old basename once and not its new basename

Scenario: 792b27c6a8 An unrelated note is not written
  Given a note that neither is renamed nor links a renamed note
  When the rename is applied
  Then it is not written

Scenario: 792b27c6a9 An empty map does nothing
  Given an empty rename map
  When the rename is applied
  Then nothing is listed, read or written

Scenario: 792b27c6a10 A note without frontmatter keeps its body
  Given a renamed note with no frontmatter
  When the rename is applied
  Then its body is kept

Scenario: 792b27c6a11 A note without a luhmann key is not given one
  Given a renamed note whose frontmatter has no luhmann: key
  When the rename is applied
  Then it gains none

Scenario: 792b27c6a12 A listing failure is returned
  Given a vault whose listing fails
  When the rename is applied
  Then the error is returned

Scenario: 792b27c6a13 A read failure is returned
  Given a note that cannot be read
  When the rename is applied
  Then the error names it

Scenario: 792b27c6a14 A rename failure is returned
  Given a rename that fails
  When the rename is applied
  Then the error names the paths

Scenario: 792b27c6a15 A write failure is returned
  Given a write that fails
  When the rename is applied
  Then the error names the path
```

### 792b27c7 Rename safety

Requirement: [792b27c7 Rename safety](#792b27c7-rename-safety) The note-reparenting MUST, before it renames or writes any file, render the new content
of every note it would rename, with CRLF converted, references rewritten, `luhmann:` set and the alias
appended, and refuse the whole invocation with one joined error naming each refused note and its reason when
that content's frontmatter does not decode as a mapping or its `luhmann:` is not the new ID. It MUST then
rename and write nothing. When it writes a note whose lines end in `\r\n`, it MUST convert them to `\n` in
that same write, and a CRLF note it does not write MUST stay byte-identical. A converted note's exchange
hash MUST equal that of the same note authored with LF, and a note with LF frontmatter MUST keep its hash.
A preview MUST run the same pre-flight, and an empty map MUST preflight nothing.

Supports: [792a10h1c A preview reports a refusal](../product/vault-migration.md#792a10h1-uat-scenarios), [792a10i1a An anchored luhmann value is refused](../product/vault-migration.md#792a10i1-uat-scenarios), [792a10i1b Every refusal is named](../product/vault-migration.md#792a10i1-uat-scenarios), [792a10i1c A CRLF note that is renamed becomes LF](../product/vault-migration.md#792a10i1-uat-scenarios), [792a10i1d A CRLF referrer is rewritten](../product/vault-migration.md#792a10i1-uat-scenarios), [792a10i1e An untouched CRLF note stays as it was](../product/vault-migration.md#792a10i1-uat-scenarios), [792a10i1f The exchange hash survives the conversion](../product/vault-migration.md#792a10i1-uat-scenarios), [792a10i1g An anchor-free note renames with every other key intact](../product/vault-migration.md#792a10i1-uat-scenarios).

Rationale: re-rendering would lose a key the note model does not define, and an anchored `luhmann:` that
another key aliases cannot be replaced without leaving that alias dangling. Converting a CRLF note only
inside a write that is made anyway keeps a rename from touching notes it has no reason to touch.

#### 792b27c7a Scenarios

```gherkin
Scenario: 792b27c7a1 An anchored luhmann value is refused untouched
  Given a renamed note with luhmann: &a "1050" and issue: *a
  When the rename is applied
  Then it fails naming the note, and no note or sidecar is renamed or written

Scenario: 792b27c7a2 Every refusal is named
  Given two renamed notes that would each be refused
  When the rename is applied
  Then one error names both

Scenario: 792b27c7a3 A CRLF renamed note is converted
  Given a renamed note whose lines end in \r\n
  When the rename is applied
  Then the written note has no \r\n, its luhmann: is its new ID, its alias is recorded, and every other key decodes as before

Scenario: 792b27c7a4 A CRLF referrer is rewritten as LF
  Given a CRLF note whose supersedes: names a renamed note
  When the rename is applied
  Then the referrer is written as LF with the new basename

Scenario: 792b27c7a5 An untouched CRLF note is never written
  Given a CRLF note that is neither renamed nor links a renamed note
  When the rename is applied
  Then it is not written

Scenario: 792b27c7a6 A preview preflights the same notes
  Given a rename map
  When the vault is preflighted without applying
  Then each refusal is reported, an empty map lists nothing, and a listing failure is returned
```

### 792b27c8 Sidecars of rewritten notes

Requirement: [792b27c8 Sidecars of rewritten notes](#792b27c8-sidecars-of-rewritten-notes) The note-reparenting MUST, after a rename, rebuild the sidecar ([792b10c5 Building a sidecar](./embedding-sidecar.md#792b10c5-building-a-sidecar)) of each note it
reports as rewritten or converted, from the note's current content, and MUST return a failure to read, embed or
write.

Supports: [792a10e1f Rewritten notes are not left stale](../product/vault-migration.md#792a10e1-uat-scenarios), [792a10i1c A CRLF note that is renamed becomes LF](../product/vault-migration.md#792a10i1-uat-scenarios).

#### 792b27c8a Scenarios

```gherkin
Scenario: 792b27c8a1 Referrers get fresh sidecars
  Given a note whose links were rewritten
  When the apply finishes
  Then its sidecar is rebuilt

Scenario: 792b27c8a2 A converted note gets a fresh sidecar
  Given a CRLF note converted by the rename
  When the apply finishes
  Then its sidecar is rebuilt from the LF content

Scenario: 792b27c8a3 A rebuild failure is returned
  Given a sidecar that cannot be rebuilt
  When the apply finishes
  Then the error is returned
```

### 792b27c9 Apply flow

Requirement: [792b27c9 Apply flow](#792b27c9-apply-flow) The note-reparenting MUST, once the answers are valid, apply the rename map ([792b27c6 Rename and link rewrite](#792b27c6-rename-and-link-rewrite),
[792b27c8 Sidecars of rewritten notes](#792b27c8-sidecars-of-rewritten-notes)) and print `update --reparent-luhmann: renamed <N> note(s)`, and, only when the map is not empty,
re-index the vault's notes with ingest and then prune the old paths from the chunk manifest. When either stage
fails it MUST print which stage failed, that the vault rename above is intact and authoritative, and which
manual commands finish the reconciliation, and return the failure without undoing the rename. It MUST finish by
deriving candidates against the resulting vault and printing either `<N> further candidate(s) found — run
`engram update --reparent-luhmann` again` or `no further candidates — vault fully evaluated`.

Supports: [792a10e1d A top answer changes nothing](../product/vault-migration.md#792a10e1-uat-scenarios), [792a10f1a The new path is indexed and the old entry detached](../product/vault-migration.md#792a10f1-uat-scenarios), [792a10f1b A failed re-index leaves the rename](../product/vault-migration.md#792a10f1-uat-scenarios), [792a10f1c Nothing renamed, nothing re-indexed](../product/vault-migration.md#792a10f1-uat-scenarios), [792a10g1a More candidates remain](../product/vault-migration.md#792a10g1-uat-scenarios), [792a10g1b The vault is fully evaluated](../product/vault-migration.md#792a10g1-uat-scenarios).

Rationale: Apply runs the rename and rewrite, then ingest, then prune, so the new paths are indexed before the
old ones are detached and no renamed note is ever unindexed. A failure in ingest or prune does not roll back
the rename: the renamed vault is the authoritative state and the chunk index is a derived cache that a later
`engram ingest --auto` or `prune` rebuilds.

#### 792b27c9a Scenarios

```gherkin
Scenario: 792b27c9a1 A continuation is renamed, rewritten and indexed
  Given an answer placing note 12 under note 7, and note 20 linking note 12
  When the answers are applied
  Then the files are renamed, note 20's link is rewritten, the new path is indexed and the old path's manifest entry is detached

Scenario: 792b27c9a2 A failed re-index keeps the rename
  Given a rename that succeeded and a re-index that fails
  When the answers are applied
  Then the stage and the manual fallback are reported and the rename is not undone

Scenario: 792b27c9a3 A failed prune keeps the rename
  Given a rename and re-index that succeeded and a prune that fails
  When the answers are applied
  Then the stage and the manual fallback are reported and the rename is not undone

Scenario: 792b27c9a4 Nothing renamed skips the index
  Given answers that rename no note
  When the answers are applied
  Then neither ingest nor prune runs

Scenario: 792b27c9a5 Further candidates are reported
  Given a resulting vault with a similar pair of top-level notes
  When the apply finishes
  Then the output gives the count and the command to run again

Scenario: 792b27c9a6 A fully evaluated vault is reported
  Given a resulting vault with no similar pair
  When the apply finishes
  Then the output says the vault is fully evaluated
```

### 792b27c10 Preview

Requirement: [792b27c10 Preview](#792b27c10-preview) The note-reparenting MUST, with the dry-run flag and an answers file, validate the
answers ([792b27c4 Answers validation](#792b27c4-answers-validation)), run the pre-flight ([792b27c7 Rename safety](#792b27c7-rename-safety)), and print `update --reparent-luhmann (dry-run): would
rename`, each `<old> -> <new>` in ascending order of old basename, and `references rewritten in: <note>` for
each other note whose references would change, or `update --reparent-luhmann (dry-run): no renames`, without
renaming, rewriting, re-indexing or pruning. It MUST return a usage error, `update --reparent-luhmann
--dry-run requires --answers`, when the dry-run flag is given without an answers file.

Supports: [792a10h1a A preview changes nothing](../product/vault-migration.md#792a10h1-uat-scenarios), [792a10h1b A preview without answers is refused](../product/vault-migration.md#792a10h1-uat-scenarios), [792a10h1c A preview reports a refusal](../product/vault-migration.md#792a10h1-uat-scenarios).

#### 792b27c10a Scenarios

```gherkin
Scenario: 792b27c10a1 A preview lists and changes nothing
  Given valid, current answers
  When they are previewed
  Then the renames and the referencing notes are printed, and no file is written, no chunk index is touched

Scenario: 792b27c10a2 A preview without answers is refused
  Given the dry-run flag and no answers file
  When the command runs
  Then it fails with the usage error and writes nothing

Scenario: 792b27c10a3 A preview reports an anchored refusal
  Given answers that would rename a note with an anchored luhmann:
  When they are previewed
  Then the refusal is returned and nothing is written

Scenario: 792b27c10a4 The apply skips the chunk pipeline on a preview
  Given a dry-run with answers
  When it finishes
  Then neither ingest nor prune ran
```
