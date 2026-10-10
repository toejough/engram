# Vault Health - Product Spec

A vault is a directory of notes and a sidecar beside each note holding its embedding ([792a3n Embed on write](./note-writing.md#792a3n-embed-on-write)). Two
commands keep it healthy. `engram check` reads the vault and reports where it breaks its
invariants: links that cannot resolve, notes recall cannot match, and sidecars that cannot be
read. `engram embed status` and `engram embed apply` count and repair the sidecars. All three read
only the Markdown notes at the vault's root, run against the vault located by [792a2a Vault location](./vault-identity.md#792a2a-vault-location), and create a
vault that does not exist first ([792a2e First-use vault creation](./vault-identity.md#792a2e-first-use-vault-creation)). The model change that makes an embed run necessary is the
embedding-sidecar's [792b10c7 State classification](../technical/embedding-sidecar.md#792b10c7-state-classification).

## 792a9a The check report

Requirement: [792a9a The check report](#792a9a-the-check-report) `engram check` MUST print one line for each invariant it checks, beginning
`PASS`, `WARN` or `FAIL` and naming the invariant, and under a violated invariant MUST list the
offending items, one per line, at most ten, followed by `… and <N> more` when there are more. It MUST
exit non-zero, with `check: FAIL-class invariant violations found` on stderr, when any line begins
`FAIL`, and MUST exit zero when the worst line is `WARN`. It MUST fail with `check: scan: <error>`
and a non-zero exit, printing no report, when a note at the vault root cannot be read. It MUST check
a note whose text is not valid UTF-8 like any other.

Rationale: a vault check is only useful if a violation cannot go unnoticed, so a correctness violation is a `FAIL` and
a non-zero exit, while a corpus-health finding is a `WARN` that reports without blocking. The checker is Go code over
bytes, not a `grep`, because a `grep` that stops at an invalid multibyte sequence silently skips notes.

### 792a9a1 UAT Scenarios

```gherkin
Scenario: 792a9a1a A clean vault passes
  Given a vault whose links resolve, whose fact notes all have a situation, and whose sidecars are current
  When the user runs engram check
  Then stdout has a PASS line for G0 graph-resolution, M5 situation-presence and S1 sidecar-schema, and the exit status is zero

Scenario: 792a9a1b A FAIL line fails the command
  Given a vault with a note whose link is a bare ID that should resolve
  When the user runs engram check
  Then a line beginning FAIL names G0 graph-resolution, stderr has "check: FAIL-class invariant violations found", and the exit status is non-zero

Scenario: 792a9a1c A WARN line does not fail the command
  Given a vault whose only finding is a link to a note that does not exist
  When the user runs engram check
  Then a line beginning WARN names G3 dangling, and the exit status is zero

Scenario: 792a9a1d Offenders are capped at ten
  Given a vault in which twelve links target no note
  When the user runs engram check
  Then ten of them are listed, followed by the line "… and 2 more"

Scenario: 792a9a1e An unreadable note stops the check
  Given a vault with a note the user cannot read
  When the user runs engram check
  Then it fails with an error beginning "check: scan:" and naming the file, and prints no PASS, WARN or FAIL line

Scenario: 792a9a1f An empty vault passes
  Given an existing vault with no notes
  When the user runs engram check
  Then every line begins PASS, the exit status is zero, and stderr is empty

Scenario: 792a9a1g A note that is not valid UTF-8 is still checked
  Given a note with invalid UTF-8 bytes in its body that links [[ghost]], and no such note
  When the user runs engram check
  Then the dangling link is reported, as it would be without the invalid bytes
```

## 792a9b Links that should resolve

Requirement: [792a9b Links that should resolve](#792a9b-links-that-should-resolve) `engram check` MUST report `FAIL  G0 graph-resolution: <N> bare-id links that
should resolve`, listing each as `<source> → [[<target>]]`, when a note's link target is a bare
Luhmann ID, digits followed by letters or digits, that is the leading ID of an existing note, and
MUST otherwise report `PASS  G0 graph-resolution: every authored link resolves by form`. A link to a
note's full basename, with or without a trailing `.md`, MUST count as resolving.

Rationale: a link is written in one form and resolved in another when `[[105]]` is written for a note
`105.2026-05-30.foo`. Neither engram nor Obsidian resolves a prefix: both resolve only a full basename (or alias).
Such a link is a lost edge. The invariant behind it is that for every write and read of the same datum the read key
must be a function of, or equal to, the write key, and a mismatch must be loud, not silent. Machine writers write the
extension-less form, and the reader accepts the suffixed form already in the vault ([792b13c3 Link
targets](../technical/wikilink-graph.md#792b13c3-link-targets)).

### 792a9b1 UAT Scenarios

```gherkin
Scenario: 792a9b1a A bare-ID link fails
  Given a note 1.2026-05-30.foo that links [[2]], and a note 2.2026-05-30.bar
  When the user runs engram check
  Then stdout has "FAIL  G0 graph-resolution: 1 bare-id links that should resolve" and the line "1.2026-05-30.foo → [[2]]"

Scenario: 792a9b1b A basename link passes
  Given a note A that links [[B]], and a note B
  When the user runs engram check
  Then stdout has "PASS  G0 graph-resolution: every authored link resolves by form"

Scenario: 792a9b1c A legacy suffixed link resolves
  Given a note body containing "Supersedes: [[B.md]]" and a note B
  When the user runs engram check
  Then the link is not reported, neither under G0 nor as dangling under G3
```

## 792a9c Links to nothing

Requirement: [792a9c Links to nothing](#792a9c-links-to-nothing) `engram check` MUST warn, without failing, with `WARN  G3 dangling: <N>
authored links target no note`, listing each as `<source> → [[<target>]]`, for every distinct link
in a note whose target, after a trailing `.md` is dropped, is neither a note's basename nor a G0
bare ID of an existing note. It MUST NOT report a note's link to itself, and MUST NOT report a link
inside a fenced code block. It MUST NOT print the line when there are none.

Rationale: the graph drops a dangling link without a sound, so a dangling link is a lost intended
edge. A deleted note's name, or a name such as `vocab.X`, is a legitimate finding, but a note may
still stand alone, so this is corpus health and not correctness.

### 792a9c1 UAT Scenarios

```gherkin
Scenario: 792a9c1a A link to a missing note warns
  Given a fact note that links [[999]] and no note has the ID 999
  When the user runs engram check
  Then stdout has a WARN G3 dangling line listing the link, G0 still passes, and the exit status is zero

Scenario: 792a9c1b A missing note is reported once however it is spelled
  Given a note that links [[vocab.X]] and [[vocab.X.md]], and no such note
  When the user runs engram check
  Then stdout has "WARN  G3 dangling: 1 authored links target no note"

Scenario: 792a9c1c A note's link to itself is not reported
  Given a note that links its own basename
  When the user runs engram check
  Then no G3 line is printed

Scenario: 792a9c1d A link inside a fenced code block is not reported
  Given a note whose only link to a missing note is inside a fenced code block
  When the user runs engram check
  Then no G3 line is printed

Scenario: 792a9c1e A vault with no dangling link prints no G3 line
  Given a vault whose links all resolve
  When the user runs engram check
  Then stdout has no line mentioning G3
```

## 792a9d Notes recall can match

Requirement: [792a9d Notes recall can match](#792a9d-notes-recall-can-match) `engram check` MUST report `FAIL  M5 situation-presence: <N> note(s) missing a
situation`, listing the notes, when a note of type `fact` or `feedback` has a `situation:` that is
absent, empty or only whitespace, and MUST otherwise report `PASS  M5 situation-presence: every
fact/feedback names a situation`. It MUST NOT judge a note of any other type, a note with no
frontmatter, or a note whose frontmatter is not valid YAML. It MUST judge a note whose lines end in
`\r\n` as it judges the same note with LF line endings.

Rationale: recall matches a query against each note's situation, and the embedding is shaped
around it ([792a7d1h A note is found by its own situation](./memory-query.md#792a7d1-uat-scenarios)). Every write path refuses a blank one ([792a3f Required content](./note-writing.md#792a3f-required-content)), so this check finds a note that
reached the vault another way.

### 792a9d1 UAT Scenarios

```gherkin
Scenario: 792a9d1a A fact without a situation fails
  Given a fact note whose frontmatter has no situation key
  When the user runs engram check
  Then stdout has "FAIL  M5 situation-presence: 1 note(s) missing a situation" naming the note, and the exit status is non-zero

Scenario: 792a9d1b A blank situation fails
  Given a feedback note whose situation is only spaces
  When the user runs engram check
  Then M5 fails naming the note

Scenario: 792a9d1c Every situation present passes
  Given fact and feedback notes that each name a situation
  When the user runs engram check
  Then stdout has "PASS  M5 situation-presence: every fact/feedback names a situation"

Scenario: 792a9d1d Other note types are not judged
  Given a note of type moc and a note of type runbook, neither with a situation
  When the user runs engram check
  Then M5 passes

Scenario: 792a9d1e A note without usable frontmatter is not judged
  Given a note with no frontmatter and a note whose frontmatter is malformed YAML
  When the user runs engram check
  Then M5 passes

Scenario: 792a9d1f A CRLF note is judged
  Given a fact note whose lines end in \r\n and whose situation is empty
  When the user runs engram check
  Then M5 fails naming the note
```

## 792a9e Sidecars that can be read

Requirement: [792a9e Sidecars that can be read](#792a9e-sidecars-that-can-be-read) `engram check` MUST report `FAIL  S1 sidecar-schema: <N> sidecar(s) on an
old/invalid schema (run `engram embed apply --force`)`, listing the notes, when a note's sidecar
cannot be decoded under the current schema, whether it is the old single-vector schema, malformed
JSON, or a vector of the wrong length. It MUST report `WARN  S1 sidecar-schema: <N> note(s) missing
a sidecar (run `engram embed apply`)`, without failing, for each note that has no sidecar. It MUST
report `PASS  S1 sidecar-schema: every sidecar parses under the current schema` when no sidecar is
invalid.

Rationale: a sidecar that cannot be decoded silently drops its note from retrieval. A missing sidecar
is a gap that `engram embed apply` fills, and it is no corruption.

### 792a9e1 UAT Scenarios

```gherkin
Scenario: 792a9e1a An old-schema sidecar fails
  Given a fact note whose sidecar has the old single-vector schema
  When the user runs engram check
  Then stdout has a FAIL line for S1 sidecar-schema naming the note and `engram embed apply --force`, and the exit status is non-zero

Scenario: 792a9e1b A malformed sidecar fails
  Given a note whose sidecar is not valid JSON
  When the user runs engram check
  Then S1 fails naming the note

Scenario: 792a9e1c A current sidecar passes
  Given a note with a current sidecar
  When the user runs engram check
  Then stdout has "PASS  S1 sidecar-schema: every sidecar parses under the current schema"

Scenario: 792a9e1d A missing sidecar warns
  Given notes with no sidecar and no invalid sidecar
  When the user runs engram check
  Then stdout has a WARN line for S1 sidecar-schema counting the notes and naming `engram embed apply`, and the exit status is zero
```

## 792a9f A check changes nothing

Requirement: [792a9f A check changes nothing](#792a9f-a-check-changes-nothing) `engram check` MUST NOT create, modify or delete any note or sidecar.

### 792a9f1 UAT Scenarios

```gherkin
Scenario: 792a9f1a A check leaves the vault as it was
  Given an existing vault with notes, some with sidecars, some violating invariants
  When the user runs engram check
  Then every file in the vault is byte-identical afterwards, and no file was created
```

## 792a9g The embedding census

Requirement: [792a9g The embedding census](#792a9g-the-embedding-census) `engram embed status` MUST print six lines: `total:`, `with-embeddings:`,
`without:`, `stale:`, `incompatible:` and `broken:`, each with the number of notes at the vault root in
that state, so that the five counts add up to the total. A note is with-embeddings when its sidecar
matches its content and the current model, without when it has no sidecar, stale when its content
changed after its sidecar was written, incompatible when its sidecar was made by another model or an
older schema, and broken when the note or its sidecar cannot be read or decoded. It MUST fail with
`embed status: scan: <error>` and a non-zero exit when a note at the vault root cannot be read.

### 792a9g1 UAT Scenarios

```gherkin
Scenario: 792a9g1a Notes without sidecars are counted as without
  Given a vault of two notes and no sidecar
  When the user runs engram embed status
  Then stdout has "total:           2", "with-embeddings: 0" and "without:         2"

Scenario: 792a9g1b Every state is counted
  Given a vault of four notes: one embedded and current, one whose body changed after embedding, one whose sidecar names another model, and one whose sidecar is malformed
  When the user runs engram embed status
  Then total is 4, and with-embeddings, stale, incompatible and broken are each 1

Scenario: 792a9g1c An unreadable note stops the census
  Given a vault with a note the user cannot read
  When the user runs engram embed status
  Then it fails with an error beginning "embed status: scan:" and naming the file
```

## 792a9h Which notes get embedded

Requirement: [792a9h Which notes get embedded](#792a9h-which-notes-get-embedded) `engram embed apply` MUST embed, with no mode flag or with `--missing`, the notes
that have no sidecar; with `--stale`, the notes whose content changed after their sidecar; with
`--force`, the notes whose sidecar was made by another model or an older schema; and with `--all`,
every note. Mode flags MUST combine, each adding its notes, and `--all` MUST select every note whatever
else is given. A note whose sidecar is broken MUST be embedded under every selection.

Rationale: a re-embed under a new model also picks up any content change, so `--force` is the explicit
step that replaces another model's sidecars; the default and `--stale` leave them alone, which keeps a
routine run from silently rewriting a vault under a model it did not choose.

### 792a9h1 UAT Scenarios

```gherkin
Scenario: 792a9h1a The default embeds only notes without a sidecar
  Given one note without a sidecar and one with a current sidecar
  When the user runs engram embed apply
  Then only the note without a sidecar is embedded

Scenario: 792a9h1b --stale embeds changed notes and not another model's sidecars
  Given a note whose body changed after its sidecar and a note whose sidecar names another model
  When the user runs engram embed apply --stale
  Then the changed note is re-embedded and the other model's sidecar is unchanged

Scenario: 792a9h1c --force replaces another model's sidecar
  Given a note whose sidecar names another model
  When the user runs engram embed apply --force
  Then its sidecar is rewritten under the current model

Scenario: 792a9h1d --all embeds every note
  Given a note with a current sidecar
  When the user runs engram embed apply --all
  Then it is embedded again

Scenario: 792a9h1e Mode flags combine
  Given a stale note, an incompatible note and a missing note
  When the user runs engram embed apply --stale --force
  Then the stale and the incompatible notes are embedded and the missing one is not

Scenario: 792a9h1f A broken sidecar is always replaced
  Given a note whose sidecar is malformed
  When the user runs engram embed apply, then with --stale, then with --force
  Then the note is embedded each time
```

## 792a9i Embedding the selected notes

Requirement: [792a9i Embedding the selected notes](#792a9i-embedding-the-selected-notes) `engram embed apply` MUST, for each selected note in ascending basename order,
write the note's sidecar beside it and print `embedded  <note>.md (<state>)`, where `<state>` is the
state the note was in. With `--dry-run` it MUST instead print `would-embed <note>.md (<state>)` and
write nothing. It MUST report a note it cannot read as `skip      <note>.md: read error: <error>`, an
embedding that fails as `fail      <note>.md: embed: <error>`, and a sidecar it cannot write as
`fail      <note>.md: write: <error>`, and MUST go on to the next note and exit zero. It MUST NOT
modify any note, and MUST fail with `embed apply: scan: <error>` and a non-zero exit when a note at the
vault root cannot be read.

Rationale: a sidecar is a derived file, not part of the note's content. It is recomputed from the
current note each time, so its content hash stays honest, and it is never stored in or versioned with
the note file.

### 792a9i1 UAT Scenarios

```gherkin
Scenario: 792a9i1a A missing sidecar is written
  Given a note 1.foo.md with no sidecar
  When the user runs engram embed apply
  Then stdout has "embedded  1.foo.md (missing)" and the sidecar 1.foo.vec.json exists

Scenario: 792a9i1b A dry run writes nothing
  Given a note 1.foo.md with no sidecar
  When the user runs engram embed apply --dry-run
  Then stdout has "would-embed 1.foo.md (missing)" and no sidecar exists

Scenario: 792a9i1c Only sidecars are written
  Given notes needing embedding
  When the user runs engram embed apply --all
  Then every note file is byte-identical afterwards, none of them contains a vector or a model ID, and the only files created are sidecars

Scenario: 792a9i1d A failed write is reported and the run goes on
  Given two notes, where a directory stands where the first note's sidecar would be
  When the user runs engram embed apply
  Then stdout has a line beginning "fail      " for the first note and ending with the write error, the second note is embedded, and the exit status is zero

Scenario: 792a9i1e Notes are embedded in order
  Given three notes without sidecars
  When the user runs engram embed apply
  Then the "embedded" lines are in ascending basename order

Scenario: 792a9i1f An unreadable note stops the run
  Given a vault with a note the user cannot read
  When the user runs engram embed apply
  Then it fails with an error beginning "embed apply: scan:" and nothing is embedded
```

## 792a9j CRLF notes embed as LF

Requirement: [792a9j CRLF notes embed as LF](#792a9j-crlf-notes-embed-as-lf) `engram embed status` and `engram embed apply` MUST treat a note whose lines end in
`\r\n` as the same note with LF line endings, so that its sidecar equals the one its LF form gets and
is not reported stale for its line endings.

### 792a9j1 UAT Scenarios

```gherkin
Scenario: 792a9j1a A CRLF note is not stale
  Given a note embedded with LF line endings and then converted to CRLF
  When the user runs engram embed status
  Then the note is counted with-embeddings, and stale is 0

Scenario: 792a9j1b A CRLF note gets its LF form's sidecar
  Given a note with LF line endings and the same note with CRLF line endings
  When the user runs engram embed apply --all on each
  Then the two sidecars are byte-identical
```
