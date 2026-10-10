# Note Document Format - Technical Spec

## 792b6a Orientation

- Ownership: Repository component, crossing the YAML library boundary.
- Responsibility: the on-disk shape of a vault note. It names the note's file, renders its
  frontmatter block with each kind's keys in a fixed order, quotes the scalars that must stay
  strings, renders the body formulas and the machine-written link lines, splits a note's text
  into frontmatter and body, reads a CRLF note as its LF form, and edits an existing note's
  frontmatter in place.
- Current realization (informational): in `internal/cli/learn.go`, `learnPath`,
  `marshalFrontmatter`, `quotedString`, the `factFrontmatterDoc`, `feedbackFrontmatterDoc` and
  `runbookFrontmatterDoc` types and their `render*` functions; in `internal/cli/qa.go`,
  `qaQuestionPath`, `qaAnswerPath`, `renderQAQuestionNote` and `renderQAAnswerNote`;
  `renderSupersedes` in `internal/cli/supersedes.go`; the exchange fields in
  `internal/cli/exchangefields.go`; `SplitFrontmatter` in `internal/embed/hash.go`; the YAML-node
  edits in `internal/cli/frontmatter_node.go`; `toLF` in `internal/cli/luhmann_reparent.go`.
- Qualification: responsibility, the file and document shape is stated without naming any writer or
  reader; ownership, writers decide what a note says and when it is written, while this component
  decides how it is laid out; independent verification, rendering, naming and splitting are tested
  directly on strings; stability, the shape survives any change to the commands that write it, and
  hand-edited and Obsidian-edited notes depend on it; necessity, every note engram writes or reads
  passes through it.
- Supported capabilities: [note-writing](../product/note-writing.md), [note-revision](../product/note-revision.md), [controlled-vocabulary](../product/controlled-vocabulary.md).
  Also [skill-runbook-registration](../product/skill-runbook-registration.md) ([792b6c9 Skill identity fields](#792b6c9-skill-identity-fields), [792b6c10 Rewrites by other writers keep the skill identity fields](#792b6c10-rewrites-by-other-writers-keep-the-skill-identity-fields)), [vault-counting](../product/vault-counting.md) and [parent-sync](../product/parent-sync.md).

Rationale: the vault is flat. Notes sit at the vault root, and a note's place in the hierarchy is
carried by the Luhmann ID that leads its filename, not by a folder ([792a2c Flat vault layout](../product/vault-identity.md#792a2c-flat-vault-layout)). A fact or feedback note
states its situation twice: in the `situation:` field, which recall's situation vector is built
from, and again in the body formula. The writers that change a
situation keep the two copies in step.

## 792b6c Behavior Requirements

### 792b6c1 Note filenames

Requirement: [792b6c1 Note filenames](#792b6c1-note-filenames) The note-document-format MUST name a fact, feedback or runbook note
`<luhmann>.<YYYY-MM-DD>.<slug>.md` in the vault root, using its creation date, and MUST name a
question-and-answer pair `qa.<YYYY-MM-DD>.<slug>.q.md` and `qa.<YYYY-MM-DD>.<slug>.a.md`.

Supports: [792a3a1a A fact note is written](../product/note-writing.md#792a3a1-uat-scenarios), [792a3b1a A feedback note is written](../product/note-writing.md#792a3b1-uat-scenarios), [792a3c1a A runbook note is written](../product/note-writing.md#792a3c1-uat-scenarios), [792a3q1a A pair is captured](../product/note-writing.md#792a3q1-uat-scenarios), [792a6a1a A definition note carries its self-tag](../product/controlled-vocabulary.md#792a6a1-uat-scenarios), [792a6b1a Bootstrap creates the family note](../product/controlled-vocabulary.md#792a6b1-uat-scenarios).

#### 792b6c1a Scenarios

```gherkin
Scenario: 792b6c1a1 A Luhmann note's filename
  Given vault /vault, Luhmann ID 1a3, creation date 2026-05-09 and slug subagent-driven-recovery
  When the note's path is formed
  Then it is /vault/1a3.2026-05-09.subagent-driven-recovery.md

Scenario: 792b6c1a2 A pair's filenames
  Given vault /vault, creation date 2026-07-03 and slug my-slug
  When the pair's paths are formed
  Then they are /vault/qa.2026-07-03.my-slug.q.md and /vault/qa.2026-07-03.my-slug.a.md
```

### 792b6c2 Frontmatter block

Requirement: [792b6c2 Frontmatter block](#792b6c2-frontmatter-block) The note-document-format MUST render a note's frontmatter as a line `---`, the
fields as YAML, a line `---` and one blank line, and MUST encode every value so that it reads back
unchanged, whatever newlines, colons, leading dashes or other YAML-significant characters it holds.

Supports: [792a3a1a A fact note is written](../product/note-writing.md#792a3a1-uat-scenarios), [792a3b1a A feedback note is written](../product/note-writing.md#792a3b1-uat-scenarios), [792a3c1a A runbook note is written](../product/note-writing.md#792a3c1-uat-scenarios), [792a3q1a A pair is captured](../product/note-writing.md#792a3q1-uat-scenarios).

Rationale: frontmatter is encoded through the YAML library, not built by concatenating strings,
so a multi-line value cannot end the block early.

#### 792b6c2a Scenarios

```gherkin
Scenario: 792b6c2a1 The block is delimited
  Given the fields k: v
  When they are rendered as frontmatter
  Then the result is "---", "k: v", "---" and a blank line

Scenario: 792b6c2a2 Tricky values read back unchanged
  Given field values holding a newline, a colon, a leading dash and a leading asterisk, or any printable text
  When a fact or feedback frontmatter is rendered and parsed again
  Then every field reads back exactly as given
```

### 792b6c3 Kind schemas

Requirement: [792b6c3 Kind schemas](#792b6c3-kind-schemas) The note-document-format MUST render each kind's keys in this order, leaving
out any optional key whose value is empty:

- fact: `type`, `tier`, `situation`, `subject`, `predicate`, `object`, `luhmann`, `created`,
  `source`, `project`, `repo`, `user`, `vault`, `pending`, `issue`, `sources`, `vocab_version`,
  `tags`, `supersedes`, then the exchange fields;
- feedback: `type`, `tier`, `situation`, `behavior`, `impact`, `action`, `luhmann`, `created`,
  `source`, `project`, `repo`, `user`, `vault`, `pending`, `issue`, `sources`, `tags`,
  `supersedes`, then the exchange fields;
- runbook: `type`, `tier`, `situation`, `done_when`, `red_flags`, `triggers`, `luhmann`, `created`,
  `source`, `project`, `repo`, `user`, `vault`, `skill_hash`, `skill_key`, `skill_source`,
  `pending`, `issue`, `sources`, `tags`, `supersedes`, then the exchange fields.

`type`, `luhmann`, `created` and `source` are always written, and so are a fact's or feedback note's
`situation` and content fields. `luhmann` and `issue` MUST be double-quoted, `created` MUST be the
date as `YYYY-MM-DD`, and list values MUST keep their order.

Supports: [792a3a1a A fact note is written](../product/note-writing.md#792a3a1-uat-scenarios), [792a3b1a A feedback note is written](../product/note-writing.md#792a3b1-uat-scenarios), [792a3c1a A runbook note is written](../product/note-writing.md#792a3c1-uat-scenarios), [792a3d1a Red flags are kept in order](../product/note-writing.md#792a3d1-uat-scenarios), [792a3d1b Triggers follow the red flags in order](../product/note-writing.md#792a3d1-uat-scenarios), [792a3d1c Neither flag means neither field](../product/note-writing.md#792a3d1-uat-scenarios), [792a3h1a Optional metadata is written](../product/note-writing.md#792a3h1-uat-scenarios), [792a3h1b Absent metadata writes no field](../product/note-writing.md#792a3h1-uat-scenarios), [792a3i1a The tier defaults to L2](../product/note-writing.md#792a3i1-uat-scenarios),
[792a3i1b A legacy tier is written as given](../product/note-writing.md#792a3i1-uat-scenarios), [792a3o1c Outside a git repository there is no repo](../product/note-writing.md#792a3o1-uat-scenarios), [792a3p1c No detectable user](../product/note-writing.md#792a3p1-uat-scenarios), [792a4b1c A runbook's steps and red flags are replaced](../product/note-revision.md#792a4b1-uat-scenarios), [792a6a1a A definition note carries its self-tag](../product/controlled-vocabulary.md#792a6a1-uat-scenarios), [792a6b1a Bootstrap creates the family note](../product/controlled-vocabulary.md#792a6b1-uat-scenarios).

Rationale: a Luhmann ID is quoted so that an ID such as `1e1` is not read back as the number 10, and
so that notes written by engram, migrated and written by hand all read the same way. A fact or
feedback note always carries `situation:`, because recall finds a note by its situation.

#### 792b6c3a Scenarios

```gherkin
Scenario: 792b6c3a1 A fact's frontmatter
  Given a fact with situation, subject, predicate, object, Luhmann ID 11, creation date 2026-05-09 and a source
  When its frontmatter is rendered
  Then it has type fact, the given fields, luhmann 11 and created 2026-05-09

Scenario: 792b6c3a2 A feedback note's frontmatter holds exactly its keys
  Given a feedback note with situation, behavior, impact, action, Luhmann ID 9z, creation date 2026-05-09, a source and nothing else
  When its frontmatter is rendered and parsed
  Then it holds exactly type, situation, behavior, impact, action, luhmann, created and source

Scenario: 792b6c3a3 A runbook's frontmatter holds exactly its keys
  Given a runbook with situation, done_when, Luhmann ID 7a, creation date 2026-05-09, a source and nothing else
  When its frontmatter is rendered and parsed
  Then it holds exactly type, situation, done_when, luhmann, created and source

Scenario: 792b6c3a4 The Luhmann ID is double-quoted
  Given a feedback note with Luhmann ID 9aa
  When its frontmatter is rendered
  Then it contains luhmann: "9aa"

Scenario: 792b6c3a5 Project and issue follow the source
  Given a fact or feedback note with project engram and issue 636
  When its frontmatter is rendered
  Then project: engram and issue: "636" follow source:, in that order

Scenario: 792b6c3a6 Empty optional keys are left out
  Given a fact, feedback or runbook note with no project, issue or red flags
  When its frontmatter is rendered
  Then it has no project:, issue: or red_flags: key

Scenario: 792b6c3a7 Lists keep their order
  Given a runbook with red flags A then B, or a fact with any list of valid tags
  When its frontmatter is rendered and parsed
  Then the list reads back with the same entries in the same order
```

### 792b6c4 Body formulas

Requirement: [792b6c4 Body formulas](#792b6c4-body-formulas) The note-document-format MUST render a fact's body as
`Information learned: when in <situation>, <subject> <predicate> <object>.`, a feedback note's as
`Lesson learned: when <situation>, <action>.`, dropping one case-insensitive leading `when ` from the
situation, and a runbook's as its steps with trailing newlines reduced to one, each followed by a
blank line and then one line `Supersedes: [[<note>]] — <type>: <claim>` per supersession, with any
`.md` dropped from the linked note.

Supports: [792a3a1a A fact note is written](../product/note-writing.md#792a3a1-uat-scenarios), [792a3a1b A leading When is not doubled](../product/note-writing.md#792a3a1-uat-scenarios), [792a3b1a A feedback note is written](../product/note-writing.md#792a3b1-uat-scenarios), [792a3b1b A leading When is not doubled](../product/note-writing.md#792a3b1-uat-scenarios), [792a3c1a A runbook note is written](../product/note-writing.md#792a3c1-uat-scenarios), [792a3h1a Optional metadata is written](../product/note-writing.md#792a3h1-uat-scenarios), [792a4b1a A fact field is replaced](../product/note-revision.md#792a4b1-uat-scenarios), [792a4b1b A feedback field is replaced](../product/note-revision.md#792a4b1-uat-scenarios), [792a4d1a A supersession is written in canonical form](../product/note-revision.md#792a4d1-uat-scenarios),
[792a4d1b The list is replaced, not appended to](../product/note-revision.md#792a4d1-uat-scenarios), [792a4l1a A fact is resituated](../product/note-revision.md#792a4l1-uat-scenarios), [792a6a1a A definition note carries its self-tag](../product/controlled-vocabulary.md#792a6a1-uat-scenarios).

Rationale: situations are usually phrased "When …", while the formulas already say "when"; without
the strip, a body would read "when When". A supersession link drops `.md` because links resolve by
extension-less basename.

#### 792b6c4a Scenarios

```gherkin
Scenario: 792b6c4a1 A fact's body
  Given a fact whose situation is "reasoning about agent coordination", subject "subagent dispatch", predicate "is fundamentally" and object "a verification problem dressed as coordination"
  When its body is rendered
  Then it is "Information learned: when in reasoning about agent coordination, subagent dispatch is fundamentally a verification problem dressed as coordination." and a blank line

Scenario: 792b6c4a2 A feedback note's body
  Given a feedback note whose situation is "orchestrating multi-step work" and action "shrink the task"
  When its body is rendered
  Then it is "Lesson learned: when orchestrating multi-step work, shrink the task." and a blank line

Scenario: 792b6c4a3 A runbook's body is its steps
  Given a runbook whose steps are three numbered lines
  When its body is rendered
  Then it begins with those three lines unchanged

Scenario: 792b6c4a4 A leading When is dropped
  Given a fact or feedback situation beginning "When "
  When its body is rendered
  Then the body reads "when <rest of situation>" and never "when When"

Scenario: 792b6c4a5 A supersession line drops .md
  Given a supersession of 9e.2026-01-01.old-note.md of type updates with claim "old note superseded"
  When its body line is rendered
  Then it is "Supersedes: [[9e.2026-01-01.old-note]] — updates: old note superseded"
```

### 792b6c5 Question-and-answer documents

Requirement: [792b6c5 Question-and-answer documents](#792b6c5-question-and-answer-documents) The note-document-format MUST render a question note with frontmatter `type:
qa-question`, `date`, `answered_by` (the answer's basename) and `source`, and a body of the question
followed by a blank line and `Answered by: [[<answer basename>]]`; and an answer note with
frontmatter `type: qa-answer`, `date`, `answers` (the question's basename), `certainty`,
`contributors` when there are any, and `source`, and a body of the answer followed by a blank line,
`Answers: [[<question basename>]]` and, when there are contributors, `Contributors:` and a
comma-separated link to each.

Supports: [792a3q1a A pair is captured](../product/note-writing.md#792a3q1-uat-scenarios), [792a3q1b A pair without contributors](../product/note-writing.md#792a3q1-uat-scenarios), [792a3q1c The answer can come from a file](../product/note-writing.md#792a3q1-uat-scenarios).

#### 792b6c5a Scenarios

```gherkin
Scenario: 792b6c5a1 A question note
  Given the question "What is the question?", slug my-slug and date 2026-07-03
  When the question note is rendered
  Then its frontmatter has type: qa-question and answered_by: qa.2026-07-03.my-slug.a, and its body holds the question and "Answered by: [[qa.2026-07-03.my-slug.a]]"

Scenario: 792b6c5a2 An answer note with contributors
  Given an answer, slug my-slug, date 2026-07-03, certainty medium and two contributors
  When the answer note is rendered
  Then its frontmatter has type: qa-answer, answers: qa.2026-07-03.my-slug.q and certainty: medium, and its body holds the answer, "Answers: [[qa.2026-07-03.my-slug.q]]" and "Contributors: [[<first>]], [[<second>]]"

Scenario: 792b6c5a3 An answer note without contributors
  Given an answer with no contributors
  When the answer note is rendered
  Then it has no Contributors: line
```

### 792b6c6 Frontmatter splitting

Requirement: [792b6c6 Frontmatter splitting](#792b6c6-frontmatter-splitting) The note-document-format MUST split a note's text into the frontmatter between
a leading `---` line and the next line that is exactly `---`, and the text after that line. A value
that ends in `---`, or an indented `---` inside a block value, MUST NOT close the frontmatter, an
empty frontmatter MUST split cleanly, and a text that has no leading `---` line or no closing line
MUST be treated as having no frontmatter.

Supports: [792a3n1a A learned note has its sidecar](../product/note-writing.md#792a3n1-uat-scenarios), [792a3u1a A runbook is tagged](../product/note-writing.md#792a3u1-uat-scenarios), [792a3u1b The answer is tagged and the question is not](../product/note-writing.md#792a3u1-uat-scenarios), [792a8d1a A note without usable frontmatter is skipped](../product/vault-counting.md#792a8d1-uat-scenarios), [792a8d1c A CRLF note is counted](../product/vault-counting.md#792a8d1-uat-scenarios), [792a9d1e A note without usable frontmatter is not judged](../product/vault-health.md#792a9d1-uat-scenarios).

Rationale: a skill key, for example, can end in `---`; matching only a whole line keeps such a value
from cutting the frontmatter short.

#### 792b6c6a Scenarios

```gherkin
Scenario: 792b6c6a1 Only a whole --- line closes the frontmatter
  Given frontmatter holding a value that ends in ---, or a block value with an indented --- line, followed by a body that may itself start with ---
  When the note is split
  Then the frontmatter runs to the first line that is exactly ---, and the body is everything after it

Scenario: 792b6c6a2 An empty frontmatter splits cleanly
  Given a note whose --- line is immediately followed by the closing --- line
  When the note is split
  Then the frontmatter is empty and the body follows

Scenario: 792b6c6a3 A note without frontmatter
  Given a note with no leading --- line, or with no closing --- line
  When the note is split
  Then it has no frontmatter, and its whole text is the body
```

### 792b6c7 Frontmatter node edits

Requirement: [792b6c7 Frontmatter node edits](#792b6c7-frontmatter-node-edits) The note-document-format MUST write an edit to an existing note's frontmatter
as edits to its parsed YAML mapping: a key whose value the edit changes MUST be replaced in place, a
key the edit adds MUST be inserted after the nearest key that precedes it in its kind's order
(before the nearest following one when none precedes, at the end when neither is present), a key
the edit clears MUST be deleted, and every other key, including keys no kind defines, MUST be left
exactly as parsed, with its anchors and style. A replaced `supersedes:` list MUST keep, for each new
entry whose `note:` names the same note as an old entry (compared without a trailing `.md`), the
old entry's keys that no kind defines. It MUST refuse the edit, naming the key, when a key it would
set or delete, its value, or anything beneath the value carries a YAML anchor; MUST refuse a
frontmatter block that is not a mapping; and MUST refuse a result whose frontmatter does not decode.
For a note whose frontmatter holds only keys its kind defines and no anchors, the edited text MUST
be byte-identical to the note rendered afresh with the edit.

Supports: [792a4j1a Amend keeps an unmodeled key](../product/note-revision.md#792a4j1-uat-scenarios), [792a4j1b Amend keeps an anchor on a key it does not edit](../product/note-revision.md#792a4j1-uat-scenarios), [792a4j1c Amend keeps the extra keys of a kept supersession](../product/note-revision.md#792a4j1-uat-scenarios), [792a4j1d Resituate keeps every untouched field](../product/note-revision.md#792a4j1-uat-scenarios), [792a4k1a Amend refuses an anchored edited key](../product/note-revision.md#792a4k1-uat-scenarios), [792a4k1b Resituate refuses an anchored situation](../product/note-revision.md#792a4k1-uat-scenarios).

Rationale: re-rendering the whole frontmatter from the typed note model would drop every key the model
does not define; editing the parsed mapping keeps them. An anchored value cannot be replaced
without leaving its aliases dangling, and decoding the result again before it is written keeps a
note that no reader could parse from ever reaching the disk. Byte-identical output for ordinary
notes keeps their exchange hash unchanged, so the offer paths that compare these bytes are unaffected.

#### 792b6c7a Scenarios

```gherkin
Scenario: 792b6c7a1 Keys the edit does not change are kept
  Given a note's frontmatter carrying keys no kind defines, a nested unknown map, and an anchor on a key the edit does not change
  When an edit replaces one key its kind defines
  Then every other key keeps its value and its anchor

Scenario: 792b6c7a2 A set key keeps or takes its kind's position
  Given the key order a, b, c, d
  When b is set in "a: 1\nb: 2\n", in "a: 1\nd: 4\n", in "x: 0\nd: 4\n", and c is set in "x: 0\n"
  Then the results are "a: 1\nb: new\n", "a: 1\nb: new\nd: 4\n", "x: 0\nb: new\nd: 4\n" and "x: 0\nc: new\n"

Scenario: 792b6c7a3 A kept supersession entry keeps its unknown keys
  Given a supersedes: list naming A with an extra key x_reason: kept, and naming B
  When the list is replaced by entries naming A and C
  Then the A entry carries the new type and claim and x_reason: kept, B is gone, and C carries only note, type and claim

Scenario: 792b6c7a4 An anchored edited key is refused
  Given frontmatter whose key to be replaced carries an anchor that another key aliases
  When the edit is made
  Then it fails naming the key, and nothing is written

Scenario: 792b6c7a5 A result that does not decode is refused
  Given rendered note text whose frontmatter holds a dangling alias, is not a mapping, or is missing
  When it is checked before a write
  Then it is refused as undecodable; valid frontmatter, an anchor with its alias, and an empty block pass

Scenario: 792b6c7a6 A frontmatter block must be a mapping
  Given a frontmatter block
  When it is parsed for editing
  Then a mapping is returned as one, an empty block is an empty mapping, a list is refused as not a mapping, and malformed YAML is refused

Scenario: 792b6c7a7 An ordinary note is written as a fresh render
  Given a fact, feedback or runbook note holding only keys its kind defines and no anchors
  When any amend kind edits it
  Then the result is byte-identical to the note rendered afresh with the edit, and its exchange hash is unchanged
```

### 792b6c8 CRLF reading

Requirement: [792b6c8 CRLF reading](#792b6c8-crlf-reading) The note-document-format MUST read a note for splitting and editing as its LF
form, converting each `\r\n` to `\n` and keeping a lone `\r`, while frontmatter splitting itself
stays LF-only ([792b6c6 Frontmatter splitting](#792b6c6-frontmatter-splitting)).

Supports: [792a4m1a Amend converts a CRLF note](../product/note-revision.md#792a4m1-uat-scenarios), [792a4m1b Resituate converts a CRLF note](../product/note-revision.md#792a4m1-uat-scenarios), [792a6f1a Vocab tagging reads a CRLF note](../product/controlled-vocabulary.md#792a6f1-uat-scenarios), [792a8d1c A CRLF note is counted](../product/vault-counting.md#792a8d1-uat-scenarios), [792a9d1f A CRLF note is judged](../product/vault-health.md#792a9d1-uat-scenarios).

Rationale: a CRLF-aware split would hand the writers that splice its slices back into a note CRLF
lines, and they would write mixed line endings; converting at the reader leaves every writer working
on the LF form, and a writer converts a note only inside a write it makes anyway.

#### 792b6c8a Scenarios

```gherkin
Scenario: 792b6c8a1 CRLF pairs become LF
  Given the text "a\r\nb\rc\r\n\r\n"
  When it is read as LF
  Then it is "a\nb\rc\n\n", and text without CRLF is unchanged
```

### 792b6c9 Skill identity fields

Requirement: [792b6c9 Skill identity fields](#792b6c9-skill-identity-fields) The note-document-format MUST write a runbook's `skill_hash`, `skill_key` and
`skill_source`, at the place [792b6c3 Kind schemas](#792b6c3-kind-schemas) gives them, only when each is non-empty, and MUST read back exactly
what it wrote for any hash, key and source.

Supports: [792a12d1a Accepted registration](../product/skill-runbook-registration.md#792a12d1-uat-scenarios), [792a12u1g Ordinary capture carries none](../product/skill-runbook-registration.md#792a12u1-uat-scenarios).

Rationale: a runbook note that mirrors a skill is identified by these three fields ([792a12b One runbook note per skill, identified by its key](../product/skill-runbook-registration.md#792a12b-one-runbook-note-per-skill-identified-by-its-key)), so a value
that does not survive a render and parse would make the note unfindable or its offer unstable.

#### 792b6c9a Scenarios

```gherkin
Scenario: 792b6c9a1 The hash is written when set
  Given a runbook with a skill_hash
  When its frontmatter is rendered
  Then it contains skill_hash with that value

Scenario: 792b6c9a2 The key and source are written when set
  Given a runbook with a skill_key and a skill_source
  When its frontmatter is rendered
  Then it contains both with their values

Scenario: 792b6c9a3 The hash is left out when absent
  Given a runbook with no skill_hash
  When its frontmatter is rendered
  Then it has no skill_hash key

Scenario: 792b6c9a4 The key and source are left out when absent
  Given a runbook with no skill_key and no skill_source
  When its frontmatter is rendered
  Then it has neither key

Scenario: 792b6c9a5 A hash reads back unchanged
  Given a runbook with a 64-character hash
  When its frontmatter is rendered and parsed
  Then the parsed hash equals the original

Scenario: 792b6c9a6 Any key and source read back unchanged
  Given any source-qualified key and any path
  When a runbook carrying them is rendered and parsed
  Then both read back unchanged
```

### 792b6c10 Rewrites by other writers keep the skill identity fields

Requirement: [792b6c10 Rewrites by other writers keep the skill identity fields](#792b6c10-rewrites-by-other-writers-keep-the-skill-identity-fields) Every writer other than registration that rewrites a note's frontmatter MUST leave
`skill_hash`, `skill_key` and `skill_source` as they were: `engram amend` (a content change, an
identity-only re-stamp, `--clear-pending` and a key that ends in dashes), `engram resituate`, the rename
of Luhmann re-parenting, the rewrite of wikilinks and the removal of references to deleted notes, the
identity backfill, and the vocabulary rewrites (assignment, clearing removed terms, definition tagging,
the version key and the strip of the legacy channel).

Supports: [792a12u1a Amend preserves the identity fields](../product/skill-runbook-registration.md#792a12u1-uat-scenarios), [792a12u1b Resituate preserves the identity fields](../product/skill-runbook-registration.md#792a12u1-uat-scenarios), [792a12u1c Reparenting preserves the identity fields](../product/skill-runbook-registration.md#792a12u1-uat-scenarios), [792a12u1d A wikilink rewrite preserves the identity fields](../product/skill-runbook-registration.md#792a12u1-uat-scenarios), [792a12u1e The identity backfill preserves the identity fields](../product/skill-runbook-registration.md#792a12u1-uat-scenarios), [792a12u1f Vocabulary rewrites preserve the identity fields](../product/skill-runbook-registration.md#792a12u1-uat-scenarios).

Rationale: only registration may change them, because registration compares a note's hash with the
source's to decide whether a skill changed.

#### 792b6c10a Scenarios

```gherkin
Scenario: 792b6c10a1 Amend keeps the fields
  Given a runbook carrying all three skill fields
  When it is amended by a content change, an identity-only re-stamp or --clear-pending, for any key spelling
  Then the three fields are unchanged in the written note

Scenario: 792b6c10a2 Resituate keeps the fields
  Given a runbook carrying all three skill fields
  When it is resituated
  Then the three fields are unchanged

Scenario: 792b6c10a3 Re-parenting keeps the fields
  Given a skill note renamed by Luhmann re-parenting
  When the rename is written
  Then the three fields are unchanged in the renamed note

Scenario: 792b6c10a4 Link rewrites keep the fields
  Given a skill note whose frontmatter links are rewritten or whose references to a deleted note are removed
  When the rewrite is written
  Then the three fields are unchanged

Scenario: 792b6c10a5 The identity backfill keeps the fields
  Given a skill note that the identity backfill stamps
  When the backfill is written
  Then the three skill fields are unchanged

Scenario: 792b6c10a6 Vocabulary rewrites keep the fields
  Given a skill note whose tags, vocabulary version or legacy vocabulary channel are rewritten
  When the rewrite is written
  Then the three fields are unchanged
```
