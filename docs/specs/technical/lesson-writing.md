# Lesson Writing - Technical Spec

## 793d3e Lesson Writing

The command `engram auto write` (`internal/cli/auto.go`, `internal/auto`, informational). Turns the model's reply into notes of the [vault](../../GLOSSARY.md#vault), owning the reading of the reply and the mark of a note (the write itself is the learn path's, as [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand) does it), writes the lessons placed and tied, a runbook among them, and what a deep delivery writes, and keeps the record of what became of each reply ([793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did)); tested through standard input and output over an in-memory vault. Called by [793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow), [793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow), [793d2c Conversation sweep flow](./conversation-sweep-workflow.md#793d2c-conversation-sweep-flow) and [855d2c Mid-turn recall flow](./mid-turn-recall-workflow.md#855d2c-mid-turn-recall-flow).

- Ownership: repository-owned component (the command `engram auto write`).
- Qualification: Purpose: the reading of the model's [replies](../../GLOSSARY.md#reply-automatic-memory), the [notes](../../GLOSSARY.md#permanent-note) marked as written automatically that the lessons of a [turn](../../GLOSSARY.md#turn-automatic-memory) and the results of a deep delivery become, and the record of what became of each reply, which [793a2c Lessons are written without the agent](../product/automatic-memory.md#793a2c-lessons-are-written-without-the-agent), [855a2a Delivery writes what its depth writes](../product/automatic-memory.md#855a2a-delivery-writes-what-its-depth-writes) and [793a2e The user can see what it did](../product/automatic-memory.md#793a2e-the-user-can-see-what-it-did) need. Contract: [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply), [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did) and [855d4j Writing what a deep delivery writes](#855d4j-writing-what-a-deep-delivery-writes) are the command's own, apart from the flows that call it, and their scenarios run the command directly through standard input and output. Identity: the requirements state only what the command reads, writes, prints and fails with, and the files that realize it are informational, so a refactoring or a change of injection leaves the boundary as it is.
- Required behavior: [Note Authoring](./note-authoring.md) (the learn path of `engram learn`) is assumed to write a note of the given kind from the fields, source, place in the vault's order, supersessions and chunk sources it is given ([792b12c4 Arguments to fields](./note-authoring.md#792b12c4-arguments-to-fields) for the chunk sources and supersessions); compatibility verification: the scenarios of [793d4h1 Scenarios](#793d4h1-scenarios) write through the real learn path over an in-memory vault (`internal/cli/auto_write_test.go`).
- Required behavior: [Note Amendment](./note-amendment.md) is assumed to amend a note with the chunk sources it is given and, for a verdict of near, the fields it names ([792b14c4 Supersessions and chunk sources](./note-amendment.md#792b14c4-supersessions-and-chunk-sources), [792b14c3 Field replacement](./note-amendment.md#792b14c3-field-replacement)); compatibility verification: none, the scenarios of [855d4j1 Scenarios](#855d4j1-scenarios) call a double of the amend (`internal/cli/auto_write_deep_test.go`).
- Supported capabilities: [Automatic Memory](../product/automatic-memory.md).

## 793d4h Writing the lessons of a reply

Requirement: 793d4h The `lesson-writing` MUST read one JSON object from standard input: the `session`'s ID, the `turn`'s ID, the model's `reply` or, when it gave none, the `failure`, and, after a capture, the `carry` of the placement and the `placement` reply, or the `placementFailure` when the placement could not be had. The reply is a JSON array of lessons, fenced or not; a lesson has a `kind` (`feedback`, `fact` or `runbook`, with the fields of its kind), a `slug` of 1 to 64 lower-case letters, digits and dashes, and a `situation`, and may name as `covered` the file name of a [note](../../GLOSSARY.md#permanent-note) of the [vault](../../GLOSSARY.md#vault) that already states it. A lesson whose `covered` names a file of the vault MUST NOT be written but recorded as skipped with that name (any other name covers nothing). Each other lesson MUST be written as a note of its kind through the learn path of [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand), with the source `written automatically (session <the session's ID>)`. The placement reply is a JSON array of `slug`, `position` (`continuation`, `sibling` or `top`), `target` and `supersedes` (each a note with a `type` of `updates`, `narrows` or `refutes` and a `claim`); a lesson MUST be written as a continuation or a sibling of its target only when the carry names the target as a [seen note](../../GLOSSARY.md#seen-note), and with a supersession only for a seen note the carry names, and every other placement or supersession is ignored: the lesson is written at the top. With no placement reply every lesson is written at the top, with no supersession; a placement reply that is not a list is ignored in the same way and recorded as skipped with the reason, and so is a `placementFailure`, a sentence that says why the placement could not be had, which the reason holds. Every field is written as one line (its line breaks and white space become single spaces and its control and format characters are dropped), so that no field makes a line of its own, such as one opening with `Supersedes:`. A reply that is not a list, and a lesson that lacks a field or has an unknown kind, write nothing and are recorded with the reason ([793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did)); a reply cut before its list ends keeps the lessons that are whole, and the cut is recorded as skipped. An object that holds a `cue` is the write of a deep delivery, which [855d4j Writing what a deep delivery writes](#855d4j-writing-what-a-deep-delivery-writes) states.

Owner: [Lesson Writing](./lesson-writing.md).

Supports: [793a2c1a A correction becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1b A request to remember becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1c An approach the user confirmed becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1d The agent's own correction becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1e An approach that succeeded with evidence becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1f Guesses, plans and chatter produce no note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1g A credential in the turn is not in the note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1h A thanks that names no approach produces no note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1i A request to remember is written when the agent says it did not save it](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a3c1a A lesson an earlier note states is not written again](../product/automatic-memory.md#793a3c1-uat-scenarios), [793a3c1b A lesson the agent saved itself is not written again](../product/automatic-memory.md#793a3c1-uat-scenarios), [793a3c1c A lesson no earlier note states is written](../product/automatic-memory.md#793a3c1-uat-scenarios), [855a2b1a A note states the rule and not the story](../product/automatic-memory.md#855a2b1-uat-scenarios), [855a2b1b The note states the lesson the learn skill writes](../product/automatic-memory.md#855a2b1-uat-scenarios), [855a2b1c A confirmed procedure is written as a runbook](../product/automatic-memory.md#855a2b1-uat-scenarios), [855a2c1a A sub-point of a note the session holds continues it](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2c1b The same thought at the same level is a sibling](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2c1c A note the session has not seen is not a target](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2d1a A correcting note names the note it corrects](../product/automatic-memory.md#855a2d1-uat-scenarios).

### 793d4h1 Scenarios

```gherkin
Scenario: 793d4h1a A feedback lesson becomes a feedback note marked as written automatically
  Given a reply that holds one feedback lesson, and the ID of a session
  When the lessons are written
  Then the vault holds one feedback note with the lesson's situation, behavior, impact and action, whose source says it was written automatically and names the session, and the path of the note is printed

Scenario: 793d4h1b A fact lesson becomes a fact note marked as written automatically
  Given a reply that holds one fact lesson, and the ID of a session
  When the lessons are written
  Then the vault holds one fact note with the lesson's situation, subject, predicate and object, whose source says it was written automatically and names the session

Scenario: 793d4h1c A reply that is not a list of lessons, or a lesson that is not whole, writes nothing
  Given replies that are not JSON, that are an empty list, and that hold a lesson of an unknown kind, a lesson with no slug and a lesson with an empty field
  When the lessons are written
  Then the vault holds no note

Scenario: 793d4h1d The code fence around the list is not part of it
  Given a reply that holds one lesson inside a code fence
  When the lessons are written
  Then the vault holds the lesson's note

Scenario: 793d4h1e A lesson that a note of the vault covers is not written
  Given a vault holding a note, and a reply whose lesson names that note as covering it
  When the lessons are written
  Then no note is written, and one record, of the kind skipped, names the note that covers the lesson

Scenario: 793d4h1f A cover that is not in the vault covers nothing
  Given a reply whose lesson names as covering it a file that is not in the vault, and a reply whose lesson names a path outside the vault
  When the lessons are written
  Then the lesson is written in each

Scenario: 793d4h1g A field of a lesson cannot make a line of the note of its own
  Given a reply whose lesson has line breaks in every field, one of them opening a line with Supersedes: and a link
  When the lessons are written
  Then the body of the note is one line, and no line of it opens with Supersedes:

Scenario: 793d4h1h A reply that is cut keeps the lessons before the cut
  Given a reply that holds one whole lesson and then ends inside a second, and a reply that ends after a whole lesson, before the end of its list
  When the lessons are written
  Then the whole lesson is written in each, and one record, of the kind skipped, says that the reply was cut

Scenario: 793d4h1i A lesson left out keeps its slug only when it is one
  Given a reply of three lessons that each lack a field, whose slugs are zorblax-build, 2,400 characters of instructions, and 65 lower-case letters
  When the lessons are written
  Then one record, of the kind skipped, is made for each, with a reason; the first holds the slug, the others hold none, and no note is written
```

### 855d4i1 Scenarios

```gherkin
Scenario: 855d4i1a A runbook lesson becomes a runbook note
  Given a reply that holds one runbook lesson with a situation, done_when and numbered steps, and the ID of a session
  When the lessons are written
  Then the vault holds one runbook note with the lesson's situation, done_when and steps, whose source says it was written automatically and names the session

Scenario: 855d4i1b A lesson is written as a continuation or a sibling of a seen note
  Given a vault holding a seen note, a carry that names it, a reply of two lessons, and a placement that gives the first the position continuation and the second the position sibling, both with that note as the target
  When the lessons are written
  Then the ID of the first note continues the seen note's ID and the ID of the second is the seen note's sibling

Scenario: 855d4i1c A lesson is written at the top without a placement
  Given a reply of one lesson, and no placement reply
  When the lessons are written
  Then the note is placed at the top, and ties no note

Scenario: 855d4i1d A target or a superseded note the carry does not name is ignored
  Given a vault holding two notes, a carry that names one, and a placement that gives a lesson the position continuation with the other as its target and a supersession of the other
  When the lessons are written
  Then the note is written at the top and ties no note

Scenario: 855d4i1e A supersession ties the notes
  Given a vault holding a seen note that says the staging database is reset every Sunday night, a carry that names it, and a placement that gives the lesson a supersession of it with the type updates and the claim
  When the lessons are written
  Then the vault holds the new note with that supersession and the old note shows that the new note supersedes it

Scenario: 855d4i1f A placement reply that is not a list is ignored and recorded
  Given a reply of one lesson, and a placement reply that is not JSON
  When the lessons are written
  Then the note is written at the top, and one record, of the kind skipped, says that the placement was not used, with the reason

Scenario: 855d4i1g A placement that failed is recorded and the lesson is written at the top
  Given a reply of one lesson, and a placementFailure
  When the lessons are written
  Then the note is written at the top, and one record, of the kind skipped, says that the placement was not used, with the failure as the reason
```

## 793d4k Recording what automatic memory did

Requirement: 793d4k The `lesson-writing` MUST record what became of each reply or failure it reads, where the display of [793a2e The user can see what it did](../product/automatic-memory.md#793a2e-the-user-can-see-what-it-did) can read it: per lesson written, kind `written` with the [note](../../GLOSSARY.md#permanent-note)'s file name; per lesson left out, `skipped` with its slug, when it is one ([793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply)), and the reason; per reply that is not a list, `skipped` with the reason; per failure, `failed` with the failure; a reply with no lesson makes no record. A lesson that cannot be written is recorded `failed` and the command then succeeds, so the [mod](../../GLOSSARY.md#mod) does not report it a second time. A record is a JSON file of its own in a directory named for the [session](../../GLOSSARY.md#session)'s ID under `auto` in the engram data directory, its name opening with the time it was made; it holds the time, the session's ID, the [turn](../../GLOSSARY.md#turn-automatic-memory)'s ID, the kind, the note, the slug and the reason. The reason is written as one line, redacted ([793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials)) and only then cut in its middle to 300 characters, so that the step a failure names at its beginning and the cause at its end stay. A record that cannot be written does not fail the command unless it records a failure, when the command fails with the reason. When it ends it MUST write one line to the debug log that `ENGRAM_DEBUG_LOG` names, with the session's ID, the turn's ID and the number of notes written, lessons left out and failures recorded, so that the end of a write that made no record can be told.

Owner: [Lesson Writing](./lesson-writing.md).

Supports: [793a2c1f Guesses, plans and chatter produce no note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1h A thanks that names no approach produces no note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2d1e A write that fails does not break the session](../product/automatic-memory.md#793a2d1-uat-scenarios).

### 793d4k1 Scenarios

```gherkin
Scenario: 793d4k1a A lesson that is written is recorded with its note
  Given a reply that holds one lesson
  When the lessons are written
  Then one record, of the kind written, holds the file name of the note, the session's ID and the turn's ID

Scenario: 793d4k1b A reply with no lesson makes no record
  Given a reply that is an empty list
  When the lessons are written
  Then no record is made, and the command succeeds

Scenario: 793d4k1c A lesson left out, and a reply that is not a list, are recorded with the reason
  Given a reply that holds a lesson with no slug beside a lesson that is whole, and a reply that is not a list
  When the lessons are written
  Then the lesson that is whole is written, and the lesson with no slug and the reply that is not a list are each recorded as skipped with a reason that names what is wrong

Scenario: 793d4k1d A failure is recorded
  Given a failure in place of a reply
  When the lessons are written
  Then one record, of the kind failed, holds the failure, and no note is written

Scenario: 793d4k1e A record that cannot be written does not fail the command
  Given a reply that holds one lesson, and a place for the records that cannot be written
  When the lessons are written
  Then the note is written and the command succeeds

Scenario: 793d4k1g A lesson that cannot be written is recorded once and does not fail the command
  Given a reply that holds one lesson, and a learn path that fails
  When the lessons are written
  Then one record, of the kind failed, holds the reason, no note is written, and the command succeeds

Scenario: 793d4k1h A failure that cannot be recorded fails the command
  Given a reply that holds one lesson, a learn path that fails, and a place for the records that cannot be written
  When the lessons are written
  Then the command fails with the reason of the learn path

Scenario: 793d4k1f A directory of records that cannot be made, and a record that cannot be created, do not fail the command
  Given a vault, a reply that holds one lesson, and a file system that refuses to make the directory of the records, and then one that makes it and refuses to create the record
  When engram auto write is dispatched with each
  Then the note is written and the command succeeds with each

Scenario: 793d4k1i A failure that is reported and cannot be recorded fails the command
  Given a failure in place of a reply, and a place for the records that cannot be written
  When the lessons are written
  Then the command fails with the failure as its reason

Scenario: 793d4k1j A reason is written as one line, redacted, and cut in its middle
  Given a failure in place of a reply, of more than 300 characters, with a line break, a credential that straddles the place where it is cut, and the cause at its end
  When the lessons are written
  Then the record holds the reason as one line without the credential, at most 300 characters of it, with its beginning and its end

Scenario: 793d4k1k A write that ends logs what it did
  Given a debug log, and a reply that holds one lesson and one that is an empty list
  When the lessons are written for each
  Then the log holds a line for each, with the session's ID and the turn's ID, the first with one note written and the second with none
```

## 855d4j Writing what a deep delivery writes

Requirement: 855d4j The `lesson-writing` MUST, given a `cue`, the `session`'s ID, the model's `reply` to a candidate request and its `carry`, act on the reply's `lessons` and `candidates`, each of an ID the carry holds only. For each lesson, a `fact` or a `feedback` as [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply) reads it with the chunk IDs of the carry's passages that evidence it (`chunks`) and optionally a `supersedes` of a [note](../../GLOSSARY.md#permanent-note) of the carry, it MUST write the note through the learn path at the top of the [vault](../../GLOSSARY.md#vault)'s order with the source `written automatically (session <the session's ID>)`, those chunk IDs as its chunk sources and the supersession. For each candidate note with the verdict `covered` it MUST amend the note, as [`engram amend`](./note-amendment.md) does, with the chunk IDs it lists as chunk sources and no field changed; for `near`, with those and the `fields` it names that its kind has. It MUST NOT amend a note that came from the parent vault or is a pending copy. A lesson that cannot be read loses only itself, and a carry that is not of depth deep writes nothing and is recorded as `skipped`. It MUST record each note written or amended as `written` with its file name (with the reason `amended` for an amend) and each lesson or candidate left out as `skipped` with the reason ([793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did)), and the run with no [turn](../../GLOSSARY.md#turn-automatic-memory).

Owner: [Lesson Writing](./lesson-writing.md).

Supports: [855a2a1b A deep delivery writes the lesson the vault lacks](../product/automatic-memory.md#855a2a1-uat-scenarios), [855a2a1a A glance leaves the vault's notes as they were](../product/automatic-memory.md#855a2a1-uat-scenarios).

### 855d4j1 Scenarios

```gherkin
Scenario: 855d4j1b A lesson the vault lacks is crystallized with its evidence
  Given a carry of depth deep that holds a passage, and a reply with one fact lesson that lists the passage
  When the writing is given the reply
  Then the vault holds a note for the lesson at the top, whose source says it was written automatically and names the session and whose chunk sources are the passage, and one record, of the kind written, names it

Scenario: 855d4j1c A covered note is amended for provenance only
  Given a carry of depth deep that holds a note and a passage, and a reply that gives the note the verdict covered and lists the passage
  When the writing is given the reply
  Then the note is amended with the passage as a chunk source, no field changed, and one record, of the kind written, names it

Scenario: 855d4j1d A near note is revised with the fields the judgement names
  Given a carry of depth deep that holds a fact note, and a reply that gives it the verdict near and a new object, and a field a fact has no such field as
  When the writing is given the reply
  Then the note is amended with the new object, the field it has no such field as is left out and recorded as skipped

Scenario: 855d4j1e A note of the parent vault and a pending copy are not amended
  Given a carry of depth deep that holds a note from the parent vault and a pending copy, and a reply that gives both the verdict near
  When the writing is given the reply
  Then neither is amended, and one record, of the kind skipped, is made for each

Scenario: 855d4j1f An ID the carry does not hold is ignored
  Given a carry of depth deep that holds one note, and a reply that gives that note and another file of the vault the verdict covered, and a lesson that lists a chunk ID and supersedes a note the carry does not hold
  When the writing is given the reply
  Then only the note of the carry is amended, and the lesson is written with no chunk source and no supersession

Scenario: 855d4j1g A lesson that cannot be read loses only itself
  Given a carry of depth deep, and a reply with a lesson whose chunks are a string and a whole lesson after it
  When the writing is given the reply
  Then the whole lesson is written, and one record, of the kind skipped, names the other by its slug

Scenario: 855d4j1h A carry that is not of depth deep writes nothing
  Given a carry of depth glance that holds a note and a passage, and a reply that gives the note the verdict covered and a lesson
  When the writing is given the reply
  Then nothing is written or amended, and one record, of the kind skipped, says why

Scenario: 861b4zzze1a A supersession is read as a file name, an object or a list
  Given a carry of depth deep that holds a note, and a reply with a fact lesson whose supersession is, in turn, the note's file name beside the type and the claim of the lesson, an object that holds the note, the type and the claim, and a list of that object
  When the reply is read
  Then the lesson ties the note with the type and the claim, for each of the three

Scenario: 861b4zzze1b A lesson that cannot be read, and has no slug, is left out by its place
  Given a reply with a lesson that has no slug and whose chunks are a string, before a lesson that is whole
  When the reply is read
  Then the whole lesson is read, and the other is left out with a reason that names its place in the list

Scenario: 861b4zzze1c A list of names is the ties of those notes
  Given a carry that holds a note, and a reply with a fact lesson whose supersession is a list of the note's file name, beside a type and a claim
  When the reply is read
  Then the lesson ties the note with the type and the claim

Scenario: 861b4zzze1d A near note is revised by the fields of its kind
  Given a carry that holds a note, in turn a fact, a feedback and a runbook, and a reply that gives it the verdict near with a field its kind has and a field its kind has not
  When the reply is read
  Then the note is amended with the field its kind has, and the field its kind has not is left out and recorded as skipped

Scenario: 861b4zzze1e A reply that is not an object writes nothing and says why
  Given a reply that is a list
  When the reply is read
  Then no lesson is read and no note is amended, and one record, of the kind skipped, says why

Scenario: 861b4zzze1f A reply with no candidates, or cut short, is no deep reply
  Given a reply that is an object with no candidates, and a reply cut in the middle of an entry
  When each reply is read
  Then nothing is read from either, and one record, of the kind skipped, says why for each

Scenario: 861b4zzze1g A runbook lesson is left out and said so
  Given a reply with a lesson of the kind runbook
  When the reply is read
  Then the lesson is not read, and one record, of the kind skipped, names it by its slug and says that a deep delivery writes facts and feedback only

Scenario: 861b4zzze1h A supersession named without its extension, or with spaces, is stored under the note's ID
  Given a carry that holds a note, and a reply with a fact lesson whose supersession names the note without the extension of its file and with spaces around the name
  When the reply is read
  Then the lesson ties the note by its file name in the carry, with the type and the claim

Scenario: 861b4zzzf1a An amend and a learn path that fail are recorded as failed, and a lesson the vault states is skipped
  Given a carry of depth deep that holds a note and a passage, and a reply that gives the note the verdict covered and lists the passage, and holds a lesson the note covers and a whole lesson no note covers, and an amend and a learn path that both fail
  When the writing is given the reply
  Then the command succeeds, and the records are one of the kind skipped for the lesson the note covers and one of the kind failed for each of the amend and the learn path

Scenario: 861b4zzzf1b Nothing outside the carry is written
  Given a carry of depth deep that holds a note and a passage, and any reply: any candidate ID with any verdict, any chunk IDs and no fields, and a lesson with any chunk IDs and any superseded note
  When the writing is given the reply
  Then the one note written has as chunk sources only the passage of the carry, when the lesson lists it, and supersedes at most the note of the carry, and the note of the carry is amended exactly when the reply gives it the verdict covered or near and lists the passage, with that passage as the only chunk source
```

## Shared requirements

- [793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials): stated once, in [Moment Judgement](./moment-judgement.md); lesson-writing is one of its owners.
