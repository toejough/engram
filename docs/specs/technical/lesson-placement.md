# Lesson Placement - Technical Spec

## 855d3d Lesson Placement

The command `engram auto place` ([855d4h Placing the lessons](#855d4h-placing-the-lessons)). Turns the lessons into the request of the mid-tier judgement of their placement and supersession against the [seen notes](../../GLOSSARY.md#seen-note). Called by [793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow).

- Ownership: repository-owned component (the command `engram auto place`).
- Qualification: Purpose: the [request](../../GLOSSARY.md#request-automatic-memory) that has a model place each lesson against the seen [notes](../../GLOSSARY.md#permanent-note) of the [session](../../GLOSSARY.md#session) and name the notes it corrects, narrows or refutes, which [855a2c Lessons are placed as the learn skill places them](../product/automatic-memory.md#855a2c-lessons-are-placed-as-the-learn-skill-places-them) and [855a2d Lessons that correct a note say so](../product/automatic-memory.md#855a2d-lessons-that-correct-a-note-say-so) need. Contract: [855d4h Placing the lessons](#855d4h-placing-the-lessons) is the command's own, apart from the capture flow that calls it ([793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow)), and its scenarios run the command directly through standard input and output. Identity: the requirement states only what the command reads, prints and fails with, not a function or an injected collaborator, so a refactoring or a change of injection leaves the boundary as it is.
- Required behavior: [Session Context](./session-context.md) is assumed to return a text that names each note delivered to the session or looked up in it ([793d4d Telling what a session holds](./session-context.md#793d4d-telling-what-a-session-holds)); compatibility verification: [855d4h1a The lessons and the seen notes are in the request](#855d4h1-scenarios) runs the placement over a [transcript](../../GLOSSARY.md#transcript) that holds a delivered block, read by the provider's own code (`internal/cli/auto_place_test.go`).
- Required behavior: [Lesson Writing](./lesson-writing.md) is assumed to read a [reply](../../GLOSSARY.md#reply-automatic-memory) of lessons as [793d4h Writing the lessons of a reply](./lesson-writing.md#793d4h-writing-the-lessons-of-a-reply) states, with the file name a lesson can give as covering it, and to give a note it writes the source `written automatically (session <the session's ID>)`; compatibility verification: none beyond [855d4h1b A note written automatically in the session is a seen note](#855d4h1-scenarios) and [858b2a1a A lesson the vault already states is not placed](#855d4h1-scenarios), which plant the marked note and the covering name.
- Supported capabilities: [Automatic Memory](../product/automatic-memory.md).

## 855d4h Placing the lessons

Requirement: 855d4h The `lesson-placement` MUST read one JSON object from standard input: the `session`'s ID, the `turn`'s ID, the model's `reply` to the lessons request (read as [793d4h Writing the lessons of a reply](./lesson-writing.md#793d4h-writing-the-lessons-of-a-reply) reads it) and the `carry`. It MUST find the seen [notes](../../GLOSSARY.md#permanent-note) of the [session](../../GLOSSARY.md#session): the [notes](../../GLOSSARY.md#permanent-note) of the [vault](../../GLOSSARY.md#vault) whose basename appears in the text that [793d4d Telling what a session holds](./session-context.md#793d4d-telling-what-a-session-holds) returns for it, and those whose source says they were written automatically in that session ([793d4h Writing the lessons of a reply](./lesson-writing.md#793d4h-writing-the-lessons-of-a-reply)). It MUST print nothing and succeed when the reply holds no whole lesson or the session has no [seen note](../../GLOSSARY.md#seen-note), so that every lesson is placed at the top and ties no note, and it MUST NOT search the rest of the vault. Otherwise it MUST print one object `{"request":{...},"carry":{...}}`. The request has `model` of the mid tier, `system`, `prompt`, `maxTokens` and `timeoutMs` of the row of the placement judgement ([793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow)). The `system` MUST hold the instructions and nothing of the session: for each lesson, by its slug, the placement by the ordered test of [792a15h Each note is placed by an ordered test](../product/learn.md#792a15h-each-note-is-placed-by-an-ordered-test) (`continuation` with a seen note when the lesson develops one specific sub-point of it, else `sibling` with a seen note when it extends the same overall thought at the same level, else `top`), the notes it corrects, narrows or refutes with the type `updates`, `narrows` or `refutes` and the claim, and the reply shape, a JSON array of `slug`, `position`, `target` and `supersedes`. The `prompt` MUST hold the lessons (slug, kind, situation) between `<lessons>` and `</lessons>` and the seen notes between `<notes>` and `</notes>`, one line each, `- <file name> (<situation>): <body>`, written as one line, redacted ([793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials)), the body cut to 400 characters and `<` written `&lt;`. The carry names the file names of the seen notes. It MUST fail, printing nothing and naming the cause, when the input is not a JSON object or the reply is not a list of lessons.

The instructions of the placement MUST say that a lesson supersedes a seen note only when it is about that note's own claim, and that a lesson is never merged into a seen note.

Owner: [Lesson Placement](./lesson-placement.md).

Supports: [855a2c1a A sub-point of a note the session holds continues it](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2c1b The same thought at the same level is a sibling](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2c1c A note the session has not seen is not a target](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2d1a A correcting note names the note it corrects](../product/automatic-memory.md#855a2d1-uat-scenarios).

### 855d4h1 Scenarios

```gherkin
Scenario: 855d4h1a The lessons and the seen notes are in the request
  Given a vault holding a note that the session's transcript shows was delivered and another that it does not, and a reply of two whole lessons
  When the placement is given the reply
  Then the request is for the model sonnet whose time limit is 15 seconds, whose system holds the ordered test, the supersedes types and the reply shape, whose prompt lists the two lessons and the delivered note only, and whose carry names the delivered note

Scenario: 855d4h1b A note written automatically in the session is a seen note
  Given a vault holding a note whose source says it was written automatically in the session, and one written automatically in another session
  When the placement is given a reply of a lesson
  Then the request lists the first note and not the second

Scenario: 855d4h1c A session that has seen no note, and a reply with no whole lesson, print nothing
  Given a vault holding a note, a session whose transcript names none and a reply of a lesson, and, in turn, a reply that is an empty list and one whose only lesson lacks a field
  When the placement is given each
  Then it prints nothing and succeeds each time, and looks nothing up

Scenario: 855d4h1e A reply that is not a list fails the command
  Given a reply that is not JSON, and an input that is not a JSON object
  When the placement is given each
  Then it prints nothing and fails each time with an error that names the reply or the input

Scenario: 855d4h1f A seen note cannot reshape the request or hold a credential
  Given seen notes whose situations and bodies hold line breaks, the tag that closes the notes, a credential and more than 400 characters
  When the request is made
  Then each note is one line of the section, the character that opens a tag is written &lt;, no credential is in the request, and each body is cut to 400 characters

Scenario: 855d4h1h A lesson supersedes a note only when it is about the note's own claim
  Given a request of the placement
  When the instructions are read
  Then a lesson about another claim supersedes nothing, even when it is near the note, and a lesson is never merged into a seen note

Scenario: 858b2a1a A lesson the vault already states is not placed
  Given a vault holding a note that states a lesson, a session that was delivered that note, and a reply of two whole lessons of which one, or both, give that note's basename as the note that covers them
  When the placement is given the reply
  Then the lesson the note covers is not asked about, and when every lesson is covered it prints nothing and succeeds
```

## Shared requirements

- [793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials): stated once, in [Moment Judgement](./moment-judgement.md); lesson-placement is one of its owners.
