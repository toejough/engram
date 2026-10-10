# Note Amendment - Technical Spec

## 792b14a Orientation

- Ownership: Repository component (caller of [wikilink-graph](./wikilink-graph.md), [note-document-format](./note-document-format.md),
  [note-provenance](./note-provenance.md), [embedding-sidecar](./embedding-sidecar.md), [durable-file-store](./durable-file-store.md) and [vault-write-lock](./vault-write-lock.md)).
- Responsibility: revises a note that already exists. Under the vault lock it finds the note,
  reads it as LF, checks the revision's arguments, applies the field, supersession, chunk-source and
  identity changes, writes the note once, and rebuilds its sidecar when what is embedded changed. It
  rewrites a fact's or feedback note's situation in both of its places, and records a note's use
  in its sidecar.
- Current realization (informational): `internal/cli/amend.go` (`RunAmend`), `resituate.go`
  (`RunResituate`), `supersedes.go` (`replaceSupersedes`, `parseAllSupersedes`), `activate.go`
  (`RunActivate`, `bumpLastUsed`); the amend and resituate targets in `internal/cli/targets.go`.
  The curation flags in `amend_fold.go` are curate's ([curate](../product/curate.md)), the offer hooks are offer-outbox's ([offer-outbox](./offer-outbox.md)) and
  the pull-down is parent-vault-client's ([parent-vault-client](./parent-vault-client.md)).
- Qualification: responsibility, revising a stored note is stated without naming any function;
  ownership, it owns the ordering of lookup, validation, edit, write and re-embed, while the format,
  provenance, sidecar and lock contracts belong to the components it calls; independent
  verification, `RunAmend`, `RunResituate` and `RunActivate` are tested directly over injected
  lookup, read, write, lock and embedder fakes; stability, the read-modify-write contract survives any
  change to the flags that drive it; necessity, every change an agent makes to an existing note
  goes through it.
- Supported capabilities: [note-revision](../product/note-revision.md), [controlled-vocabulary](../product/controlled-vocabulary.md), [parent-sync](../product/parent-sync.md) ([792b14c12 Activation against the parent](#792b14c12-activation-against-the-parent),
  [792b14c13 A discard of a pulled note records its decline](#792b14c13-a-discard-of-a-pulled-note-records-its-decline)), [curate](../product/curate.md) ([792b14c14 Settling a pending offer](#792b14c14-settling-a-pending-offer)), [write-memory](../product/write-memory.md), [recall](../product/recall.md).

Rationale: a revision is a read-modify-write of a note and its sidecar, so two revisions, or a
revision and a learn, that interleave could each overwrite the other's change. Every revision takes
the vault's lock at its command's entry point and holds it through the note write, the sidecar
rebuild and the last-used bump. The helpers it shares with other writers never take the lock
themselves, because the lock is held per file descriptor and a nested acquire would wait on itself.

## 792b14c Behavior Requirements

### 792b14c1 Locked revision

Requirement: [792b14c1 Locked revision](#792b14c1-locked-revision) The note-amendment MUST take the vault lock ([792b8c4 The vault's lock](./vault-write-lock.md#792b8c4-the-vaults-lock)) before amend, resituate
or activate reads any note or sidecar, MUST hold it through every read and write of the note and
its sidecar, including the sidecar rebuild and the last-used bump, MUST release it before
returning, and MUST NOT take it again in the helpers it calls while holding it. A failure to take the
lock MUST be returned with nothing read or written.

Supports: [792a4o1a Concurrent amends both take effect](../product/note-revision.md#792a4o1-uat-scenarios), [792a4o1b An activation does not undo a re-embed](../product/note-revision.md#792a4o1-uat-scenarios).

#### 792b14c1a Scenarios

```gherkin
Scenario: 792b14c1a1 An amend holds the lock across its read-modify-write
  Given a lock that records when it is taken and released
  When an amend reads, writes and re-embeds a note
  Then the lock is taken before the scan and the read, and released after the note and sidecar writes

Scenario: 792b14c1a2 A resituate holds the lock across its read-modify-write
  Given a lock that records when it is taken and released
  When a resituate rewrites a note and its sidecar
  Then the lock is taken before the read and released after both writes

Scenario: 792b14c1a3 An activate holds the lock around its bumps
  Given a lock that records when it is taken and released
  When an activate bumps two notes' sidecars
  Then the lock is taken once before the first bump and released after the last
```

### 792b14c2 Amend refusals

Requirement: [792b14c2 Amend refusals](#792b14c2-amend-refusals) The note-amendment MUST refuse an amend, writing nothing, when no note
matches the reference, when the note has no frontmatter, is not a fact, feedback or runbook note, or
has a `created:` that is not a `YYYY-MM-DD` date, when a `--chunk-source` ID is not in the chunk
index, when a `--supersedes` value does not parse ([792b12c1 Argument validation](./note-authoring.md#792b12c1-argument-validation)), or when a `--trigger` is shorter than
three characters after trimming.

Supports: [792a4a1b An unknown reference is refused](../product/note-revision.md#792a4a1-uat-scenarios), [792a4c1a A question-and-answer note is refused](../product/note-revision.md#792a4c1-uat-scenarios), [792a4f1a An invalid amend argument is refused before any write](../product/note-revision.md#792a4f1-uat-scenarios).

#### 792b14c2a Scenarios

```gherkin
Scenario: 792b14c2a1 No matching note
  Given a vault with no note matching the target
  When an amend runs
  Then it fails as not found, and nothing is written

Scenario: 792b14c2a2 A note without frontmatter
  Given a target note with no frontmatter
  When an amend runs
  Then it fails saying the note has no parseable frontmatter, and nothing is written

Scenario: 792b14c2a3 A note of another kind
  Given a target note whose type is not fact, feedback or runbook
  When an amend runs
  Then it fails saying the note type is unknown, and nothing is written

Scenario: 792b14c2a4 A malformed creation date
  Given a target note whose created: is not a YYYY-MM-DD date
  When an amend runs
  Then it fails, and nothing is written

Scenario: 792b14c2a5 An unknown chunk source
  Given a chunk index that does not hold nope#x
  When an amend runs with --chunk-source nope#x
  Then it fails naming nope#x as unresolved, and nothing is written

Scenario: 792b14c2a6 A trigger that is too short
  Given a runbook note
  When an amend runs with --trigger ab
  Then it fails saying a trigger must be at least 3 characters, and nothing is written
```

### 792b14c3 Field replacement

Requirement: [792b14c3 Field replacement](#792b14c3-field-replacement) The note-amendment MUST overwrite each content field of the note's kind that
the amend supplies with a value different from the note's, MUST replace a runbook's whole
`red_flags:` or `triggers:` list when one is supplied and differs, and MUST treat the amend as
changing content exactly when one of these values changed. When content changed it MUST rebuild a
fact or feedback body from its formula ([792b6c4 Body formulas](./note-document-format.md#792b6c4-body-formulas)) and a runbook body from the supplied steps, or the
note's own steps when none are supplied, followed by the supersession lines; otherwise it MUST keep
the body and only replace its supersession lines. It MUST write the note once and print its path.

Supports: [792a4b1a A fact field is replaced](../product/note-revision.md#792a4b1-uat-scenarios), [792a4b1b A feedback field is replaced](../product/note-revision.md#792a4b1-uat-scenarios), [792a4b1c A runbook's steps and red flags are replaced](../product/note-revision.md#792a4b1-uat-scenarios), [792a4g1a A content amend re-embeds](../product/note-revision.md#792a4g1-uat-scenarios), [792a4g1b A provenance-only amend does not re-embed](../product/note-revision.md#792a4g1-uat-scenarios).

#### 792b14c3a Scenarios

```gherkin
Scenario: 792b14c3a1 One fact field is replaced
  Given a fact note
  When an amend supplies only --subject with a new value
  Then only subject: changes in the frontmatter, the body formula is rebuilt with it, and the note's path is printed

Scenario: 792b14c3a2 Feedback fields are replaced
  Given a feedback note
  When an amend supplies --action with a new value
  Then action: and the body formula carry the new action

Scenario: 792b14c3a3 A runbook's body, done-when and red flags are replaced
  Given a runbook note
  When an amend supplies --body, --done-when and --red-flag
  Then the body is the new steps, done_when: is new, and red_flags: holds only the given flags

Scenario: 792b14c3a4 A runbook's situation alone keeps its steps
  Given a runbook note with steps
  When an amend supplies only --situation
  Then situation: changes and the body keeps the note's steps

Scenario: 792b14c3a5 Triggers are replaced whole
  Given a runbook note whose triggers: lists two triggers
  When an amend supplies one --trigger
  Then triggers: lists only that trigger

Scenario: 792b14c3a6 An unchanged value is not a content change
  Given a note
  When an amend supplies only values equal to the note's own
  Then the amend is not a content change, and the sidecar is not rebuilt
```

### 792b14c4 Supersessions and chunk sources

Requirement: [792b14c4 Supersessions and chunk sources](#792b14c4-supersessions-and-chunk-sources) The note-amendment MUST replace a note's whole `supersedes:` list with the
entries an amend supplies, rewriting the body's `Supersedes:` lines to match ([792b6c4 Body formulas](./note-document-format.md#792b6c4-body-formulas)), and MUST
merge each supplied `--chunk-source` ID into `sources:` after the IDs already there, once.

Supports: [792a4d1a A supersession is written in canonical form](../product/note-revision.md#792a4d1-uat-scenarios), [792a4d1b The list is replaced, not appended to](../product/note-revision.md#792a4d1-uat-scenarios), [792a4e1a Chunk sources are merged without duplicates](../product/note-revision.md#792a4e1-uat-scenarios).

#### 792b14c4a Scenarios

```gherkin
Scenario: 792b14c4a1 A supersession is written to the frontmatter and the body
  Given a fact or feedback note
  When an amend supplies --supersedes "<note>.md|refutes|claim"
  Then supersedes: holds the entry with note <note>.md, and the body ends with "Supersedes: [[<note>]] — refutes: claim"

Scenario: 792b14c4a2 Chunk sources are merged
  Given a note whose sources: lists one ID, and a chunk index holding it and another
  When an amend supplies both IDs
  Then sources: lists the existing ID and then the new one, each once
```

### 792b14c5 Identity re-stamp gating

Requirement: [792b14c5 Identity re-stamp gating](#792b14c5-identity-re-stamp-gating) The note-amendment MUST have [note-provenance](./note-provenance.md) re-stamp a note ([792b11c4 Re-stamp](./note-provenance.md#792b11c4-re-stamp)) exactly
when the amend supplies a content flag of any kind, a `--red-flag`, a `--trigger`, a `--supersedes`
or a `--chunk-source`, detecting the repository from the working directory whatever the note's
`project:`, and MUST NOT re-stamp it otherwise. On every amend it writes, it MUST fill an absent or
empty `vault:` with the resolved vault name, keep a declared one, and, when no vault name was
resolved, leave `vault:` out and log a warning.

Supports: [792a4h1a Amend from another environment re-stamps](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1b Amend backfills missing identity](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1c The project field does not stand in for the repo](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1f Without a vault name the default is stamped](../product/note-revision.md#792a4h1-uat-scenarios), [792a4h1g Recording a use keeps the declared identity](../product/note-revision.md#792a4h1-uat-scenarios), [792a4i1a A use recorded on a note without vault fills it in](../product/note-revision.md#792a4i1-uat-scenarios).

#### 792b14c5a Scenarios

```gherkin
Scenario: 792b14c5a1 A content, supersession or chunk-source amend re-stamps
  Given a note whose repo:, user: and vault: differ from the detected ones
  When an amend supplies a content flag, --supersedes or --chunk-source
  Then repo:, user: and vault: are the detected ones

Scenario: 792b14c5a2 A note without identity gains it
  Given a note with no repo:, user: or vault:
  When an amend supplies --chunk-source
  Then it gains the detected repo:, user: and vault:

Scenario: 792b14c5a3 The project field is not the repository
  Given a note whose project: differs from the detected repository
  When an amend re-stamps it
  Then repo: is the detected repository

Scenario: 792b14c5a4 Re-stamping follows the flags exactly
  Given any combination of amend flags on any kind of note
  When the amend runs
  Then identity is re-stamped exactly when a content flag of that kind, --supersedes or --chunk-source is present, and is otherwise as declared

Scenario: 792b14c5a5 A bookkeeping amend keeps the declared identity
  Given a fact, feedback or runbook note with a declared identity
  When an amend supplies only --activate
  Then repo:, user: and vault: are unchanged, and the note was rewritten

Scenario: 792b14c5a6 An empty vault is filled, never written empty
  Given a note with no vault:
  When an amend that does not re-stamp runs with the vault name personal, and again with no vault name
  Then the first writes vault: personal with no warning, and the second writes no vault: key and warns naming vault
```

### 792b14c6 Sidecar rebuild after an amend

Requirement: [792b14c6 Sidecar rebuild after an amend](#792b14c6-sidecar-rebuild-after-an-amend) After writing the note, the note-amendment MUST build the note's sidecar from
the written text ([792b10c5 Building a sidecar](./embedding-sidecar.md#792b10c5-building-a-sidecar)) and write it to the sidecar path ([792b3c1 Atomic replace](./durable-file-store.md#792b3c1-atomic-replace)) when the amend changed content
or converted the note from CRLF, and MUST NOT rebuild it otherwise. When building or writing fails,
it MUST log a warning naming the note and complete the amend.

Supports: [792a4g1a A content amend re-embeds](../product/note-revision.md#792a4g1-uat-scenarios), [792a4g1b A provenance-only amend does not re-embed](../product/note-revision.md#792a4g1-uat-scenarios), [792a4g1c A failed re-embed keeps the amend](../product/note-revision.md#792a4g1-uat-scenarios), [792a4m1a Amend converts a CRLF note](../product/note-revision.md#792a4m1-uat-scenarios).

#### 792b14c6a Scenarios

```gherkin
Scenario: 792b14c6a1 A provenance-only amend does not re-embed
  Given a note
  When an amend supplies only --supersedes, or only values equal to the note's own
  Then the embedder is not called and no sidecar is written

Scenario: 792b14c6a2 A failed re-embed only warns
  Given an embedder that fails
  When an amend changes a content field
  Then the note is written, a warning naming the note is logged, and the amend succeeds
```

### 792b14c7 CRLF revision

Requirement: [792b14c7 CRLF revision](#792b14c7-crlf-revision) The note-amendment MUST read the note an amend or resituate edits as its LF
form ([792b6c8 CRLF reading](./note-document-format.md#792b6c8-crlf-reading)), MUST write it as LF in the single write it makes, byte-identical to the same
revision of the note's LF form, and MUST count the conversion as a reason to rebuild the sidecar.

Supports: [792a4m1a Amend converts a CRLF note](../product/note-revision.md#792a4m1-uat-scenarios), [792a4m1b Resituate converts a CRLF note](../product/note-revision.md#792a4m1-uat-scenarios).

#### 792b14c7a Scenarios

```gherkin
Scenario: 792b14c7a1 An amend converts a CRLF note
  Given a fact, feedback or runbook note whose lines end in \r\n
  When an amend of any kind that writes the note runs
  Then the note is written once with no \r\n, and its sidecar is rebuilt

Scenario: 792b14c7a2 The output equals the LF note's
  Given any note and its CRLF form
  When the same amend runs on each
  Then the two written notes are byte-identical

Scenario: 792b14c7a3 A resituate converts a CRLF note
  Given a fact or feedback note whose lines end in \r\n
  When a resituate runs
  Then it is written once with no \r, keeping its other keys
```

### 792b14c8 Resituation

Requirement: [792b14c8 Resituation](#792b14c8-resituation) The note-amendment MUST resituate only a fact or feedback note with a
`created:` that is a `YYYY-MM-DD` date, by setting `situation:` and re-emitting `created:` as the
note's own string ([792b6c7 Frontmatter node edits](./note-document-format.md#792b6c7-frontmatter-node-edits)), replacing the body's first line with the formula's first line for the
new situation and keeping the rest of the body, writing the note once and printing its path. It
MUST then build and write the note's sidecar, and MUST return a failure to scan, read, write, embed
or write the sidecar.

Supports: [792a4j1d Resituate keeps every untouched field](../product/note-revision.md#792a4j1-uat-scenarios), [792a4l1a A fact is resituated](../product/note-revision.md#792a4l1-uat-scenarios), [792a4l1b A runbook cannot be resituated](../product/note-revision.md#792a4l1-uat-scenarios), [792a4l1c A failed sidecar rebuild fails the command](../product/note-revision.md#792a4l1-uat-scenarios).

#### 792b14c8a Scenarios

```gherkin
Scenario: 792b14c8a1 A fact is resituated
  Given a fact note
  When it is resituated
  Then the written note equals the original with the old situation replaced in situation: and the body formula, and the sidecar's content hash tracks the new text

Scenario: 792b14c8a2 A feedback note is resituated
  Given a feedback note
  When it is resituated
  Then situation: and the body's first line carry the new situation, and the rest is unchanged

Scenario: 792b14c8a3 Other notes are refused
  Given a note with no frontmatter, malformed frontmatter, an unparseable created:, or another kind
  When it is resituated
  Then it fails and nothing is written

Scenario: 792b14c8a4 Every untouched key and the created string are kept
  Given a note carrying any other keys, and a created: written in its quoted form
  When it is resituated
  Then every key but situation: is unchanged, and created: keeps the note's own string

Scenario: 792b14c8a5 Every failure is returned
  Given a scan, read, note write, embedding or sidecar write that fails
  When a note is resituated
  Then that failure is returned
```

### 792b14c9 Recording a use

Requirement: [792b14c9 Recording a use](#792b14c9-recording-a-use) The note-amendment MUST record a use by reading the note's sidecar, setting
its last-used date to today and writing it back atomically with every other field unchanged, and
MUST skip the write when the date is already today. A sidecar that cannot be read or decoded MUST
fail the bump with an error naming the sidecar. An amend with `--activate` MUST bump after any
rebuild, so the rebuilt sidecar carries today's date, and MUST warn and complete the amend when the
bump fails.

Supports: [792a4n1a A note's use is recorded](../product/note-revision.md#792a4n1-uat-scenarios), [792a4n1d Amend records a use](../product/note-revision.md#792a4n1-uat-scenarios).

Rationale: the last-used date is metadata outside the content hash, so the bump never makes the
sidecar stale and never triggers a re-embed.

#### 792b14c9a Scenarios

```gherkin
Scenario: 792b14c9a1 The date is set and the vectors kept
  Given a sidecar with vectors and a content hash
  When a use is recorded on 2026-06-17
  Then the sidecar's last_used is 2026-06-17 and its vectors and content hash are unchanged

Scenario: 792b14c9a2 The same date is not rewritten
  Given a sidecar whose last_used is already today
  When a use is recorded
  Then nothing is written

Scenario: 792b14c9a3 An unreadable sidecar fails the bump
  Given a sidecar that cannot be read
  When a use is recorded
  Then the error names the sidecar's path

Scenario: 792b14c9a4 Amend records a use, after a rebuild
  Given a note whose amend changes a content field
  When the amend supplies --activate
  Then the sidecar written last carries today's last_used

Scenario: 792b14c9a5 The production wiring writes the real sidecar
  Given a note with a sidecar on disk
  When a use is recorded through the production dependencies
  Then the sidecar file on disk carries today's last_used
```

### 792b14c10 Activation refs

Requirement: [792b14c10 Activation refs](#792b14c10-activation-refs) With no parent configured, the note-amendment MUST resolve each activate ref
to a note file, as given when absolute and otherwise under the vault, adding `.md` when missing; MUST
record a use for each ref whose note file exists, counting one without a sidecar as activated; and
MUST log a warning naming each ref whose note does not exist, continue with the others, and fail
with an error saying how many refs failed.

Supports: [792a4n1a A note's use is recorded](../product/note-revision.md#792a4n1-uat-scenarios), [792a4n1b A note without a sidecar still counts](../product/note-revision.md#792a4n1-uat-scenarios), [792a4n1c A missing note fails only its own ref](../product/note-revision.md#792a4n1-uat-scenarios).

#### 792b14c10a Scenarios

```gherkin
Scenario: 792b14c10a1 Every existing note is activated
  Given two notes with sidecars
  When both are activated
  Then each sidecar's last_used is today, and the command succeeds

Scenario: 792b14c10a2 Absolute and vault-relative refs resolve
  Given a note named by its absolute path, or by its name under the vault
  When it is activated
  Then its sidecar is bumped

Scenario: 792b14c10a3 A missing note is warned and the rest continue
  Given one existing note and one ref naming no note
  When both are activated
  Then a warning names the missing ref, the existing note is bumped, and the command fails

Scenario: 792b14c10a4 All refs failing fails the command
  Given refs none of whose notes exist
  When they are activated
  Then the command fails
```

### 792b14c11 Vocabulary tagging after a revision

Requirement: [792b14c11 Vocabulary tagging after a revision](#792b14c11-vocabulary-tagging-after-a-revision) After an amend or a resituate writes the note (and, for a resituate, after its
sidecar is rewritten), the note-amendment MUST have the [vocabulary-engine](./vocabulary-engine.md) tag the note's written
content and then check whether a refit is due.

Supports: [792a6d1d An amend re-tags the note](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6d1e A resituated note is re-tagged](../product/controlled-vocabulary.md#792a6d1-uat-scenarios).

#### 792b14c11a Scenarios

```gherkin
Scenario: 792b14c11a1 An amended note is tagged
  Given a vault with a term close to the amended note and a readable sidecar for it
  When the note is amended
  Then the note is rewritten with the term's vocab/ tag

Scenario: 792b14c11a2 The refit check runs after an amend
  Given a vault whose refit baseline is 100 notes 20 days ago, now holding 150 notes, and no term vectors wired
  When a note is amended
  Then the vault's refit is marked pending

Scenario: 792b14c11a3 A resituated note is tagged
  Given a vault with a term close to the resituated note
  When the note is resituated
  Then the term vectors are loaded after the note is written, and the note gains the term's vocab/ tag

Scenario: 792b14c11a4 The refit check runs after a resituate
  Given a vault whose refit baseline is 100 notes 20 days ago, now holding 150 notes
  When a note is resituated
  Then the vault's refit is marked pending
```

### 792b14c12 Activation against the parent

Requirement: [792b14c12 Activation against the parent](#792b14c12-activation-against-the-parent) With a parent configured, or with `--parent`, the note-amendment MUST extend [792b14c10 Activation refs](#792b14c10-activation-refs) as follows.
A ref whose note file exists MUST be a local hit and be bumped, except that with `--parent` no ref is looked up locally.
A ref that is not a local hit MUST be handed to the pull-down, with no vault lock held, only when it is a basename or
`<basename>.md`; a bare Luhmann ID, and with `--parent` a path, MUST fail as not sent to the parent (a miss also saying
the note was not found), and a miss with no pull-down configured MUST fail as not found. `--parent` with no parent
configured MUST fail with `--parent requires ENGRAM_PARENT to be configured` before any lookup, read or write. After the
local pass it MUST re-check, with no lock held, each parent note that a local hit links to under its parent vault ID, and
a failed re-check MUST NOT fail the activate. When any pull or re-check was attempted it MUST run the finish step, which
drains the outbox, once after all of them. It MUST warn for and count every failed ref and fail with how many of the refs
failed.

Supports: [792a21zc1a A parent-only ref pulls down](../product/parent-sync.md#792a21zc1-uat-scenarios), [792a21zc1b A local hit never contacts the parent](../product/parent-sync.md#792a21zc1-uat-scenarios), [792a21zc1c Bare Luhmann IDs stay local](../product/parent-sync.md#792a21zc1-uat-scenarios), [792a21zc1d Partial failure is visible](../product/parent-sync.md#792a21zc1-uat-scenarios), [792a21zc1e --parent without a parent configured](../product/parent-sync.md#792a21zc1-uat-scenarios), [792a21zc1f --parent pulls even on a local hit](../product/parent-sync.md#792a21zc1-uat-scenarios), [792a21zc1g --parent refuses a path ref](../product/parent-sync.md#792a21zc1-uat-scenarios), [792a21zc1h A miss with no parent fails the ref](../product/parent-sync.md#792a21zc1-uat-scenarios), [792a21zf1d An unreachable parent does not fail the local activate](../product/parent-sync.md#792a21zf1-uat-scenarios).

Rationale: a bare Luhmann ID names an unrelated note in another vault, so only a basename may leave this vault. The local
pass holds the vault lock and each pull takes it only for its own write, because a fetch under the lock would stall every
other writer for as long as the parent took to answer.

#### 792b14c12a Scenarios

```gherkin
Scenario: 792b14c12a1 A local hit is bumped and the parent is not contacted
  Given a ref whose note file exists, with or without a sidecar
  When it is activated with a parent configured
  Then its use is recorded and no pull is requested

Scenario: 792b14c12a2 A basename that misses is pulled down
  Given a parent is configured and a basename that has no local note
  When it is activated
  Then the pull-down receives the ref with no vault lock held

Scenario: 792b14c12a3 An ID or a path is not sent
  Given a bare Luhmann ID that has no local note, and, with --parent, a path ref
  When each is activated
  Then each fails as not sent to the parent and nothing is requested

Scenario: 792b14c12a4 --parent resolves every ref against the parent
  Given a local file of the ref's name exists
  When it is activated with --parent
  Then the pull-down receives it and the local file is not bumped

Scenario: 792b14c12a5 --parent with no parent fails before any lookup
  Given no pull-down is configured
  When it is activated with --parent
  Then it fails with the not-configured error and nothing is read or written

Scenario: 792b14c12a6 A linked hit is re-checked and a failure is not fatal
  Given a local hit linked to a parent note under its parent vault, and a re-check that fails
  When it is activated
  Then the re-check is attempted after the local pass and the activate succeeds

Scenario: 792b14c12a7 The finish step runs once
  Given several pulls and re-checks
  When they have all run
  Then the finish step runs exactly once, and not at all when none was attempted

Scenario: 792b14c12a8 Failed refs are counted
  Given two refs of which one fails
  When they are activated
  Then a warning names the failed ref and the command fails saying 1 of 2 failed
```

### 792b14c13 A discard of a pulled note records its decline

Requirement: [792b14c13 A discard of a pulled note records its decline](#792b14c13-a-discard-of-a-pulled-note-records-its-decline) A bare `--discard` of a note whose primary link is `via: pulled` MUST record the link's parent
basename and hash, keyed by the link's parent vault ID, in the vault's declined-pull record before it deletes the note, and
MUST fail without deleting the note when the decline cannot be recorded. It MUST record a repeated decline once, and MUST
record nothing for a note that is not pulled. In a vault with no parent configured and no vault ID it MUST record nothing
and create neither a vault ID nor `.engram/`. `--discard --into` MUST NOT record a decline.

Supports: [792a21ze1c A declined pull is remembered](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1h A decline that cannot be recorded keeps the note](../product/parent-sync.md#792a21ze1-uat-scenarios), [792a21ze1i Discarding a pulled note with no vault ID records nothing](../product/parent-sync.md#792a21ze1-uat-scenarios).

Rationale: the decline is what stops an unchanged parent note that curation rejected from coming back on the next
activate. A decline that failed to record must not delete the note, or the note could return unrecorded. Recording a
decline in a vault with no parent would stamp a vault ID for no purpose.

#### 792b14c13a Scenarios

```gherkin
Scenario: 792b14c13a1 A decline that cannot be recorded keeps the note
  Given a pulled note and a declined-pull record that cannot be written
  When the note is discarded
  Then the discard fails and the note and its sidecar remain

Scenario: 792b14c13a2 A repeated decline is recorded once
  Given a pulled note whose decline is already recorded
  When another pulled note with the same basename and hash is discarded
  Then the record holds the entry once

Scenario: 792b14c13a3 A note that is not pulled records nothing
  Given a note whose primary link is offered, or that has no link
  When it is discarded
  Then it is deleted and nothing is recorded

Scenario: 792b14c13a4 An unconfigured vault records nothing
  Given a vault with no parent configured and no vault ID
  When a pulled note is discarded
  Then it is deleted, no decline is recorded, and no vault ID or .engram directory is created

Scenario: 792b14c13a5 A CRLF pulled note's decline is recorded
  Given a pulled note whose lines end in CRLF
  When it is discarded outright
  Then its decline is recorded from the LF form and the note is deleted
```

### 792b14c14 Settling a pending offer

Requirement: [792b14c14 Settling a pending offer](#792b14c14-settling-a-pending-offer) The note-amendment MUST clear the pending marker of its target for `--clear-pending` (and for the amend arguments' own pending field set false, which is not a command-line flag: `--pending` is listed by `--help` only by mistake, #797), on a
fact, feedback or runbook note, a runbook note that carries `skill_hash` included, changing no other field of the note and not
re-stamping its identity ([792b14c5 Identity re-stamp gating](#792b14c5-identity-re-stamp-gating)), and MUST NOT set the marker in any local amend. It MUST, for `--discard`, delete the note
and then its sidecar under the vault lock the amend holds, print the note's path only when both were deleted, treat a missing
sidecar as success, and fail with the cause when either removal fails, a failed removal of the note never attempting the
sidecar. The judged-version check ([792b34c5 The judged-version check](./pending-offer-bookkeeping.md#792b34c5-the-judged-version-check)) MUST run before either.

Supports: [792a22b1b A local amend never sets the marker](../product/curate.md#792a22b1-uat-scenarios), [792a22h1a An absent offer is accepted](../product/curate.md#792a22h1-uat-scenarios), [792a22h1b Accepting an offer keeps its author](../product/curate.md#792a22h1-uat-scenarios), [792a22k1d Clearing a skill note's marker makes it live](../product/curate.md#792a22k1-uat-scenarios).

Rationale: `--clear-pending` is the only way a curator accepts an offer, and the offer keeps its declared author because the
bookkeeping amends do not re-stamp it. A pending note is a real file, so discarding it is a deletion, not a
revision.

#### 792b14c14a Scenarios

```gherkin
Scenario: 792b14c14a1 Clearing the marker makes the note live
  Given a pending fact note
  When engram amend --target <note> --clear-pending runs
  Then the note no longer carries pending: true

Scenario: 792b14c14a2 A skill note keeps its skill_hash when cleared
  Given a pending runbook note that carries skill_hash
  When engram amend --target <note> --clear-pending runs
  Then the marker is gone, skill_hash is unchanged and the note is no longer a pending offer

Scenario: 792b14c14a3 A local amend never sets the marker
  Given a live note
  When engram amend --target <note> --object="amended" runs
  Then the amended note carries no pending line

Scenario: 792b14c14a4 A discard deletes the note and its sidecar
  Given a note with a sidecar
  When engram amend --target <note> --discard runs
  Then both files are gone and the note's path is printed

Scenario: 792b14c14a5 A note removal error is returned and nothing is printed
  Given a note whose removal fails
  When it is discarded
  Then the discard fails with that cause, no path is printed and the sidecar was not attempted

Scenario: 792b14c14a6 A sidecar removal error is returned
  Given a sidecar whose removal fails for a reason other than being absent
  When the note is discarded
  Then the discard fails with that cause and no path is printed

Scenario: 792b14c14a7 A missing sidecar is not an error
  Given a note whose sidecar is already gone
  When it is discarded
  Then the discard succeeds and the note's path is printed

Scenario: 792b14c14a8 An accepted offer keeps its author
  Given a pending offer whose user: is alice and a detected user bob
  When the marker is cleared
  Then user: is still alice and the pending line is gone
```
