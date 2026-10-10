# Note Authoring - Technical Spec

## 792b12a Orientation

- Ownership: Repository component (caller of [note-document-format](./note-document-format.md), [luhmann-addressing](./luhmann-addressing.md),
  [vault-write-lock](./vault-write-lock.md), [embedding-sidecar](./embedding-sidecar.md), [note-provenance](./note-provenance.md) and [durable-file-store](./durable-file-store.md)).
- Responsibility: carries out `engram learn fact`, `feedback`, `runbook` and `qa`. It validates the
  arguments, turns them into a note's fields, and orders the write: lock the vault, allocate the
  Luhmann ID, stamp provenance, render and create the note, write its sidecar and vocabulary tags,
  release the lock and report the path. For a question-and-answer pair it writes both notes and
  undoes the first when the second fails.
- Current realization (informational): `RunLearn`, `writeLearnUnderLock`, `writeLearnLocked`,
  `assembleLearnContent`, the `validate*` functions and the `learnArgsFrom*` mappings in
  `internal/cli/learn.go`; `RunLearnQA`, `writeQANotesUnderLock` and the QA validators in
  `internal/cli/qa.go`; the learn argument types in `internal/cli/targets.go` and `qa.go`;
  `newLearnDeps` and `newQaDeps`. Each `learn` subcommand is its own process; nothing here calls
  another subcommand.
- Qualification: responsibility, capturing a note is stated without naming any function or seam;
  ownership, it owns validation, field mapping and the order of the write, while the components it
  calls own the format, the IDs, the lock, the sidecar and provenance; independent verification,
  it is tested directly over injected dependencies; stability, the capture flow survives any
  refactoring of the components it calls; necessity, every new note is written through it.
- Supported capabilities: [note-writing](../product/note-writing.md), [controlled-vocabulary](../product/controlled-vocabulary.md), [curate](../product/curate.md) ([792b12c8 The pending marker is not set by a local learn](#792b12c8-the-pending-marker-is-not-set-by-a-local-learn)).
  Also [skill-runbook-registration](../product/skill-runbook-registration.md), [write-memory](../product/write-memory.md), [learn](../product/learn.md), [vault-serving](../product/vault-serving.md) and [parent-sync](../product/parent-sync.md).

Rationale: the vault lock spans everything from listing the existing IDs to creating the note, so
two concurrent writers never allocate the same ID; creating the note exclusively is a backstop that
fails rather than overwrites. A question-and-answer pair has no Luhmann ID, so it takes the lock only
around its two writes and embeds after releasing it: sidecar writes are atomic per file and need no
lock, and holding it through embedding would stall other writers.

## 792b12c Behavior Requirements

### 792b12c1 Argument validation

Requirement: [792b12c1 Argument validation](#792b12c1-argument-validation) The note-authoring MUST refuse, before writing anything, a slug that is empty
or does not match `[a-z0-9-]+`, a project that is not kebab-case, an issue containing whitespace, a
tag that is not `<family>` or `<family>/<value>` with kebab-case segments, a trigger shorter than
three characters after trimming, a tier other than `L1`, `L2` or `L3`, a supersession that is not
`<note>|<type>|<claim>` with a non-empty note and claim and a type of `updates`, `narrows` or
`refutes`, and a note type other than fact, feedback or runbook. A claim MAY itself contain `|`.

Supports: [792a3g1a An invalid argument is refused before any write](../product/note-writing.md#792a3g1-uat-scenarios).

#### 792b12c1a Scenarios

```gherkin
Scenario: 792b12c1a1 An invalid slug is refused
  Given a learn with slug "Bad Slug"
  When it runs
  Then it fails with the invalid-slug error and nothing is written

Scenario: 792b12c1a2 Project, issue and tag shapes are checked
  Given a project, issue or tag value, empty or not
  When it is validated
  Then an empty project, issue or tag list and well-shaped values pass, and a badly shaped project, an issue with whitespace or a malformed tag is refused before any write

Scenario: 792b12c1a3 A short trigger is refused
  Given a runbook learn with a trigger shorter than three characters
  When it runs
  Then it fails and nothing is written

Scenario: 792b12c1a4 An unknown tier is refused
  Given a learn with tier L9
  When it runs
  Then it fails with the bad-tier error and nothing is written

Scenario: 792b12c1a5 A malformed supersession is refused
  Given a supersession with too few fields, an unknown type, an empty note or an empty claim
  When it is parsed
  Then it is refused with the matching error

Scenario: 792b12c1a6 A well-formed supersession is accepted
  Given a supersession of type updates, narrows or refutes, whose claim may contain |
  When it is parsed
  Then its note, type and claim are read, with the claim's | kept

Scenario: 792b12c1a7 An unknown note type is refused
  Given a learn of an unknown type such as principle
  When it runs
  Then it fails with the unknown-type error
```

### 792b12c2 Required content

Requirement: [792b12c2 Required content](#792b12c2-required-content) The note-authoring MUST refuse a fact, feedback or runbook whose situation is
empty or whitespace, and a runbook whose done-when is empty or whitespace, with an error naming the
missing flag and nothing written.

Supports: [792a3f1a A missing situation is refused](../product/note-writing.md#792a3f1-uat-scenarios), [792a3f1b A blank situation is refused](../product/note-writing.md#792a3f1-uat-scenarios), [792a3f1c A runbook without done-when is refused](../product/note-writing.md#792a3f1-uat-scenarios).

#### 792b12c2a Scenarios

```gherkin
Scenario: 792b12c2a1 A fact or feedback without a situation is refused
  Given a fact or feedback learn whose situation is empty or whitespace
  When it runs
  Then it fails naming --situation and nothing is written

Scenario: 792b12c2a2 A runbook without a situation is refused
  Given a runbook learn whose situation is empty
  When it runs
  Then it fails naming --situation and nothing is written

Scenario: 792b12c2a3 A runbook without done-when is refused
  Given a runbook learn whose done-when is empty
  When it runs
  Then it fails naming --done-when and nothing is written
```

### 792b12c3 Locked write

Requirement: [792b12c3 Locked write](#792b12c3-locked-write) The note-authoring MUST, for a fact, feedback or runbook, take the vault lock;
list the vault's note IDs, which are the IDs of the Markdown files at its root; allocate the next ID
for the target and position; stamp provenance; render and exclusively create the note; write its
sidecar and apply vocabulary tagging; release the lock; and print the note's path. A lock failure,
a listing failure, a refused placement or refused content MUST return an error with nothing
written.

Supports: [792a3a1a A fact note is written](../product/note-writing.md#792a3a1-uat-scenarios), [792a3b1a A feedback note is written](../product/note-writing.md#792a3b1-uat-scenarios), [792a3c1a A runbook note is written](../product/note-writing.md#792a3c1-uat-scenarios), [792a3j1a Top takes the next top-level number](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1b A continuation of a number takes the next letter](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1c A continuation of a letter takes the next number](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1d A sibling takes the next child of the target's parent](../product/note-writing.md#792a3j1-uat-scenarios), [792a3j1e Letters roll over from z to aa](../product/note-writing.md#792a3j1-uat-scenarios),
[792a3k1a A missing position is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3k1b A continuation without a target is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3k1c A sibling of a top-level note is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3k1d A malformed target is refused](../product/note-writing.md#792a3k1-uat-scenarios), [792a3m1a Concurrent learns get distinct IDs](../product/note-writing.md#792a3m1-uat-scenarios), [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a3n1b A failed embedding keeps the note](../product/note-writing.md#792a3n1-uat-scenarios), [792a3o1a The origin remote names the repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3p1a The git email names the user](../product/note-writing.md#792a3p1-uat-scenarios), [792a3u1a A runbook is tagged](../product/note-writing.md#792a3u1-uat-scenarios).

#### 792b12c3a Scenarios

```gherkin
Scenario: 792b12c3a1 A note is written at its allocated path
  Given an empty vault and a fact, feedback or runbook learn with slug fact-slug at top on 2026-05-09
  When it runs
  Then the note is created at /vault/1.2026-05-09.fact-slug.md with its kind's frontmatter and body, and its path is printed

Scenario: 792b12c3a2 A lock failure writes nothing
  Given a vault lock that cannot be taken
  When a learn runs
  Then it fails with the lock error and nothing is written

Scenario: 792b12c3a3 A listing failure writes nothing
  Given a vault whose note IDs cannot be listed
  When a learn runs
  Then it fails with the listing error and nothing is written

Scenario: 792b12c3a4 Only root notes count as IDs
  Given a vault with notes 1 and 1a at its root, a note in a subdirectory and a README.md
  When its note IDs are listed
  Then they are exactly 1 and 1a
```

### 792b12c4 Arguments to fields

Requirement: [792b12c4 Arguments to fields](#792b12c4-arguments-to-fields) The note-authoring MUST give a note `tier: L2` when no tier is given and the
given tier otherwise, and MUST carry the project, issue, tags, chunk sources, supersessions, and a
runbook's red flags and triggers in order, into the note's fields, with none written when none is
given.

Supports: [792a3d1a Red flags are kept in order](../product/note-writing.md#792a3d1-uat-scenarios), [792a3d1b Triggers follow the red flags in order](../product/note-writing.md#792a3d1-uat-scenarios), [792a3d1c Neither flag means neither field](../product/note-writing.md#792a3d1-uat-scenarios), [792a3h1a Optional metadata is written](../product/note-writing.md#792a3h1-uat-scenarios), [792a3h1b Absent metadata writes no field](../product/note-writing.md#792a3h1-uat-scenarios), [792a3i1a The tier defaults to L2](../product/note-writing.md#792a3i1-uat-scenarios), [792a3i1b A legacy tier is written as given](../product/note-writing.md#792a3i1-uat-scenarios).

#### 792b12c4a Scenarios

```gherkin
Scenario: 792b12c4a1 The tier defaults to L2
  Given a fact or feedback learn without a tier, or with tier L3
  When it runs
  Then the note carries tier: L2, or tier: L3

Scenario: 792b12c4a2 Project and issue are carried
  Given a fact learn with --project engram and --issue 636
  When it runs
  Then the note carries project: engram and issue: "636"

Scenario: 792b12c4a3 Tags are carried
  Given a fact or feedback learn with tags, or with none
  When it runs
  Then the note carries the tags in order, or no tags: key

Scenario: 792b12c4a4 Chunk sources are carried
  Given a fact learn with chunk sources, or with none
  When it runs
  Then the note carries them under sources:, or no sources: key

Scenario: 792b12c4a5 Supersessions are carried
  Given a feedback learn with a supersession
  When it runs
  Then the note carries it under supersedes: and as a Supersedes: body line

Scenario: 792b12c4a6 Red flags are carried
  Given a runbook learn with repeated --red-flag values, or with none
  When it runs
  Then the note carries them in order under red_flags:, or no red_flags: key and no error

Scenario: 792b12c4a7 Triggers are carried
  Given a runbook learn with repeated --trigger values, or with none
  When it runs
  Then the note carries them in order under triggers: after red_flags:, or no triggers: key, for any valid triggers
```

### 792b12c5 Question-and-answer write

Requirement: [792b12c5 Question-and-answer write](#792b12c5-question-and-answer-write) The note-authoring MUST, for a question-and-answer pair, validate the
arguments, defaulting certainty to `medium`; read the answer from `--answer-file` when given; check
every contributor against the vault's Markdown files; and only then take the vault lock, create the
question note and then the answer note, both exclusively, and release the lock. When the answer note
cannot be created it MUST remove the question note and return an error saying so, also naming the
failure when the question cannot be removed. After a successful write it MUST write both sidecars,
apply vocabulary tagging to the answer only, and print the question's path and then the answer's.
A lock failure MUST return an error with nothing written.

Supports: [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a3q1a A pair is captured](../product/note-writing.md#792a3q1-uat-scenarios), [792a3q1b A pair without contributors](../product/note-writing.md#792a3q1-uat-scenarios), [792a3q1c The answer can come from a file](../product/note-writing.md#792a3q1-uat-scenarios), [792a3r1a An invalid pair is refused before any write](../product/note-writing.md#792a3r1-uat-scenarios), [792a3r1b Certainty defaults to medium](../product/note-writing.md#792a3r1-uat-scenarios), [792a3s1a A failed answer removes the question](../product/note-writing.md#792a3s1-uat-scenarios), [792a3u1b The answer is tagged and the question is not](../product/note-writing.md#792a3u1-uat-scenarios).

#### 792b12c5a Scenarios

```gherkin
Scenario: 792b12c5a1 A pair is written question first
  Given a valid pair
  When it is learned
  Then two notes are created, the .q.md first and the .a.md second

Scenario: 792b12c5a2 The answer comes from a file
  Given a pair whose answer is given by --answer-file
  When it is learned
  Then both notes are created with the file's text as the answer

Scenario: 792b12c5a3 Invalid arguments are refused
  Given a pair with an invalid slug, no question, both or neither answer source, no source, or an unknown certainty
  When it is validated
  Then it is refused with the matching error, and a fully valid pair passes

Scenario: 792b12c5a4 An unknown contributor is refused before any write
  Given a pair whose contributor names no note in the vault
  When it is learned
  Then it fails saying the contributor was not found, and nothing is written

Scenario: 792b12c5a5 A lock failure writes nothing
  Given a vault lock that cannot be taken
  When a pair is learned
  Then it fails with the lock error

Scenario: 792b12c5a6 A failed answer removes the question
  Given an answer note that cannot be created
  When a pair is learned
  Then the question note is removed and an error is returned

Scenario: 792b12c5a7 A failed removal is reported too
  Given an answer note that cannot be created and a question note that cannot be removed
  When a pair is learned
  Then the error names the orphaned question note
```

### 792b12c6 Vocabulary tagging after a write

Requirement: [792b12c6 Vocabulary tagging after a write](#792b12c6-vocabulary-tagging-after-a-write) After creating a fact, feedback or runbook note and writing its sidecar, and
after writing both sidecars of a question-and-answer pair, the note-authoring MUST have the
[vocabulary-engine](./vocabulary-engine.md) tag the written note (the answer only, never the question) and then check whether
a refit is due, and MUST still succeed when tagging or the check does nothing.

Supports: [792a3u1a A runbook is tagged](../product/note-writing.md#792a3u1-uat-scenarios), [792a3u1b The answer is tagged and the question is not](../product/note-writing.md#792a3u1-uat-scenarios), [792a6c1a A categorical tag survives tagging](../product/controlled-vocabulary.md#792a6c1-uat-scenarios), [792a6d1a A learned note is tagged with its closest terms](../product/controlled-vocabulary.md#792a6d1-uat-scenarios), [792a6e1a A vault without a vocabulary writes plain notes](../product/controlled-vocabulary.md#792a6e1-uat-scenarios), [792a6e1b A note whose embedding failed is not tagged](../product/controlled-vocabulary.md#792a6e1-uat-scenarios), [792a6s1a Growth marks a refit pending](../product/controlled-vocabulary.md#792a6s1-uat-scenarios), [792a6s1c A first write records the baseline](../product/controlled-vocabulary.md#792a6s1-uat-scenarios).

#### 792b12c6a Scenarios

```gherkin
Scenario: 792b12c6a1 A learned note is tagged after its write
  Given a vault with a term close to the note's text and a readable sidecar for the note
  When the note is learned
  Then the note is rewritten with the term's vocab/ tag after its sidecar is written

Scenario: 792b12c6a2 The refit check runs after a learn
  Given a vault whose refit baseline is 100 notes 20 days ago, now holding 150 notes, and no term vectors wired
  When a note is learned
  Then the vault's refit is marked pending

Scenario: 792b12c6a3 The question of a pair is not tagged
  Given a vault with a term close to both a question and its answer
  When the pair is learned
  Then only the answer note is tagged
```

### 792b12c7 Skill identity fields are not written by learn

Requirement: [792b12c7 Skill identity fields are not written by learn](#792b12c7-skill-identity-fields-are-not-written-by-learn) The note-authoring MUST NOT write `skill_hash`, `skill_key` or `skill_source` for a
note captured through `engram learn`, and the learn commands MUST NOT offer a flag for them. Only the
registration path, which calls the authoring code directly, supplies them.

Supports: [792a12u1g Ordinary capture carries none](../product/skill-runbook-registration.md#792a12u1-uat-scenarios).

#### 792b12c7a Scenarios

```gherkin
Scenario: 792b12c7a1 A captured runbook has no skill hash
  Given a runbook learn
  When it runs
  Then the note carries no skill_hash

Scenario: 792b12c7a2 A captured runbook has no skill key or source
  Given a runbook learn
  When it runs
  Then the note carries neither skill_key nor skill_source
```

### 792b12c8 The pending marker is not set by a local learn

Requirement: [792b12c8 The pending marker is not set by a local learn](#792b12c8-the-pending-marker-is-not-set-by-a-local-learn) The note-authoring MUST write `pending: true` only when the arguments it is given carry the pending marker, which
a served learn, a pull-down and a skill registration build, and the command line MUST NOT expose a flag that sets it, so that a local `engram learn`
never writes a pending note.

Supports: [792a22b1a A local learn is never pending](../product/curate.md#792a22b1-uat-scenarios).

Rationale: the marker means that the note came from another vault and has not been judged, so a note the user writes in their own
vault is never held back from recall.

#### 792b12c8a Scenarios

```gherkin
Scenario: 792b12c8a1 A local learn writes no marker
  Given the user runs engram learn fact on the host
  When the note is written
  Then it carries no pending line
```
