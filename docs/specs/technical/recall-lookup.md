# Recall Lookup - Technical Spec

## 855d3c Recall Lookup

The command `engram auto recall`. Turns the phrases of the phrases judgement into the candidate [notes](../../GLOSSARY.md#permanent-note) and passages and the request of the mid-tier judgement of them, running the lookup of [792b24c5 The matched set](./query-engine.md#792b24c5-the-matched-set) of the [query-engine](./query-engine.md); tested through standard input and output over an injected lookup and chunk reader. Called by [793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow).

- Ownership: repository-owned component (the command `engram auto recall`).
- Qualification: Purpose: [793a2b Memory arrives with the prompt](../product/automatic-memory.md#793a2b-memory-arrives-with-the-prompt) promises that the notes and [passages](../../GLOSSARY.md#passage) that apply to a [prompt](../../GLOSSARY.md#prompt-automatic-memory) arrive with it and
  [855a2a Delivery writes what its depth writes](../product/automatic-memory.md#855a2a-delivery-writes-what-its-depth-writes) that a deep delivery writes the lesson the [vault](../../GLOSSARY.md#vault) lacks; the candidates and the [request](../../GLOSSARY.md#request-automatic-memory) that judges them are this
  component's. Contract: it turns the phrases of the phrases judgement into the candidates and the request of the candidate judgement
  ([855d4b Looking up the candidates for the phrases](#855d4b-looking-up-the-candidates-for-the-phrases)), and it is tested directly through standard input and output.
  Identity: the requirement states only what the command reads, prints and fails with, and the files that realize it are informational, so a refactoring or a change of
  injection leaves the boundary as it is.
- Required behavior: [Query Engine](./query-engine.md) returns the matched set for the phrases ([792b24c5 The matched set](./query-engine.md#792b24c5-the-matched-set)) and tells the notes picked by the
  explore half alone ([792b24c16 Delivering the explore half](./query-engine.md#792b24c16-delivering-the-explore-half)) and the newest chunks ([792b24c19 The newest chunks](./query-engine.md#792b24c19-the-newest-chunks)) from it.
- Required behavior: [Chunk Index Store](./chunk-index-store.md) keeps the text of each chunk in the index file of its source ([792b18c1 Index layout](./chunk-index-store.md#792b18c1-index-layout)), the text that
  `engram show-chunk` prints for the chunk ([792b2c8 Chunk lookup](./command-line-surface.md#792b2c8-chunk-lookup)).
- Required behavior: [Lesson Writing](./lesson-writing.md) reads the lessons that a deep [reply](../../GLOSSARY.md#reply-automatic-memory) holds in the shape of [793d4h Writing the lessons of a reply](./lesson-writing.md#793d4h-writing-the-lessons-of-a-reply).
- Required behavior: [Recall](../product/recall.md): the [recall skill](../../GLOSSARY.md#recall-skill) is assumed to define, as [792a14j Coverage is judged by the agent and written once per cluster](../product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster) states, when a
  note is near, which the verdicts of [855d8b The candidates of the lookup](#855d8b-the-candidates-of-the-lookup) and the instructions of the request repeat; compatibility verification:
  none, the repeated text is `internal/auto/prompts/candidates.txt`, which no test compares with the skill.
- Supported capabilities: [Automatic Memory](../product/automatic-memory.md).

## 855d4b Looking up the candidates for the phrases

Requirement: 855d4b The `recall-lookup` MUST read one JSON object from standard input: the `session`'s ID, the `reply` of the phrases judgement and its `carry`, which holds the cue. The reply is a JSON object with `phrases`, a list of short strings; a reply that is not one, or holds no phrase, MUST fail the command, printing nothing and naming the judgement, and so MUST a carry whose depth is neither `glance` nor `deep`, naming the depth. It MUST run the lookup of [792b24c5 The matched set](./query-engine.md#792b24c5-the-matched-set) once, for the [vault](../../GLOSSARY.md#vault), with the first 3 phrases at glance and the first 10 at deep, each as a phrase of its own and with no message, and MUST keep the [notes](../../GLOSSARY.md#permanent-note) and the chunks the matched set holds, not the notes picked by the explore half alone ([792b24c16 Delivering the explore half](./query-engine.md#792b24c16-delivering-the-explore-half)) and not the newest chunks ([792b24c19 The newest chunks](./query-engine.md#792b24c19-the-newest-chunks)), and not the chunks of the [session](../../GLOSSARY.md#session)'s own [transcript](../../GLOSSARY.md#transcript). When none remain, or none fits the budget of [855d8b The candidates of the lookup](#855d8b-the-candidates-of-the-lookup), it MUST print nothing and succeed. Otherwise it MUST read the text of each chunk it kept as [`engram show-chunk`](../../GLOSSARY.md#engram-show-chunk-subcommand) does, from the index file of its source alone, and print one object `{"request":{...},"carry":{...}}`. The request has `model` of the mid tier, `system`, `prompt` and `maxTokens`, and no `timeoutMs` ([793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow)). The `system` MUST hold the instructions and nothing of the session: for each candidate a `verdict` ([855d8b The candidates of the lookup](#855d8b-the-candidates-of-the-lookup)); as `newer` the candidate that reverses another because its date is later and it explicitly reverses it; and the reply shape; and, at deep depth, also the `lessons` the evidence shows that no candidate states, in the shape of [793d4h Writing the lessons of a reply](./lesson-writing.md#793d4h-writing-the-lessons-of-a-reply), and for a near note the `fields` to revise. The `prompt` MUST hold the window of the carry, escaped again, between `<window>` and `</window>` and the candidates between `<candidates>` and `</candidates>`, one line each, redacted ([793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials)) but for the ID, which is a path or a file name and no credential, and with `<` written `&lt;`, in the format and within the budget of [855d8b The candidates of the lookup](#855d8b-the-candidates-of-the-lookup). The carry holds the cue, the depth, the window and the candidates as the lookup found them, raw and not cut, since only the request is redacted; its output holds `"write":true` at deep depth. It MUST print nothing and fail, naming the cause, when the lookup fails, the cause holding no text of the [prompt](../../GLOSSARY.md#prompt-automatic-memory), and MUST write a warning of the lookup to the debug log and not to standard error.

The instructions of the candidate judgement MUST say that a note is near only when the moment sharpens its own claim, the same subject and the same principle, and that a note that merely shares a situation or words and would need a different claim is unrelated; those of a deep request MUST say that a revision sharpens the note's own claim and never adds a different claim, a status, a plan, a branch, worktree or ticket name, a timestamp or an episode, and that a lesson must be durable and atomic with no ticket or change number in its situation.

Owner: [Recall Lookup](./recall-lookup.md).

Supports: [855a3b1a A prompt that is not a recall moment arrives with nothing](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1b A note whose words match but that does not apply is not delivered](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1c Of two notes that conflict, only the newer is delivered](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1d The notes that arrive are the notes the recall skill draws on](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1e A passage of an earlier conversation arrives when it applies](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a2a1a A glance leaves the vault's notes as they were](../product/automatic-memory.md#855a2a1-uat-scenarios), [855a2a1b A deep delivery writes the lesson the vault lacks](../product/automatic-memory.md#855a2a1-uat-scenarios), [793a2b1d A prompt that matches no note arrives with nothing](../product/automatic-memory.md#793a2b1-uat-scenarios).

### 855d4b1 Scenarios

```gherkin
Scenario: 855d4b1a The phrases are looked up once, as phrases of their own
  Given a reply of four phrases at glance, one of twelve at deep, and a lookup that records its arguments
  When the lookup is asked for each
  Then it is run once for each, with the first 3 and the first 10 phrases, each as a phrase of its own, and with no message

Scenario: 855d4b1c A reply that is not a phrases reply fails the command
  Given replies that are not JSON, that hold no phrases and that hold an empty list, a carry with no depth, and a fenced reply of one phrase
  When the lookup is asked for each
  Then the fenced one is looked up, and the others print nothing and fail with an error that names the judgement (the carry with no depth, the depth) and holds no text of the prompt

Scenario: 855d4b1d The notes and passages that matched become candidates, the explore picks and the newest chunks do not
  Given a lookup whose payload holds a matched note, a matched chunk, a note picked by the explore half alone and a newest chunk
  When the lookup is asked
  Then the request lists the matched note and the matched chunk, with the text of the chunk, and not the others

Scenario: 855d4b1e A chunk of the session's own transcript is not a candidate
  Given a lookup whose payload holds a chunk of the transcript of the session and a chunk of another
  When the lookup is asked
  Then the request lists the chunk of the other only

Scenario: 855d4b1f Nothing matched prints nothing
  Given a lookup that returns no note and no chunk
  When the lookup is asked
  Then it prints nothing and succeeds

Scenario: 855d4b1g The request is for the mid tier and holds the verdicts, the window and the dated candidates
  Given a lookup that returns two notes and a chunk with dates, and a carry with a window
  When the lookup is asked at glance
  Then the request is for the model sonnet with no time limit, whose system holds the four verdicts, the rule that a note matching only superseded content is near, the newer as the later date that reverses and the reply shape and none of the candidates, whose prompt holds the window and a line for each candidate with its date, for a note its type, and for one from the parent vault the word parent, and whose carry holds the three candidates in full with their dates and no "write":true

Scenario: 855d4b1h A deep request also asks for the lessons and the revisions
  Given the same lookup
  When the lookup is asked at deep
  Then the system also holds the lessons and the fields (those of a runbook too), that a note a newer candidate reverses is near with fields, and the output holds "write":true

Scenario: 855d4b1i A candidate cannot reshape the request or hold a credential
  Given notes and chunks whose texts hold line breaks, the tag that closes the candidates, a credential of every shape, a note of more than 400 characters and a chunk of more than 1,200
  When the lookup is asked
  Then each candidate is one line of the section, the character that opens a tag is written &lt;, no credential is in the request, an ID that reads as a token is in it as it is, the note is whole and the chunk is cut to 1,200 characters in the request, the carry holds the candidates as they were found, and and a window that holds a tag cannot close the window of the request

Scenario: 855d4b1j A failed lookup fails the command and prints nothing
  Given a lookup that fails
  When the lookup is asked for a prompt
  Then it fails with an error that names the lookup's failure and holds no text of the prompt, and it prints nothing

Scenario: 855d4b1k A warning of the lookup is no line for the user
  Given a vault holding a note whose embedding is of another model, and a debug log
  When the lookup is asked
  Then standard error holds nothing and the debug log holds the warning of the lookup

Scenario: 855d4b1l The candidates are held to a count and a budget
  Given a lookup that returns 40 notes whose bodies hold 3,000 characters each
  When the lookup is asked
  Then the request lists the first 20 of the matched set whole, and the candidates hold no more than 60,000 characters of text in all

Scenario: 855d4b1m A cap on the count keeps a floor of passages
  Given a matched set of 32 notes and 12 passages, and any matched set of notes and passages in any order
  When the lookup is asked
  Then the request lists 30 candidates, at least 10 of them passages and at least 20 notes when so many matched, in the order of the matched set

Scenario: 855d4b1n The candidates are in the order of their scores, and nothing that fits prints nothing
  Given a lookup whose payload holds a chunk that scores higher than a note, and one whose only note holds more than 60,000 characters
  When the lookup is asked
  Then the request lists the chunk first, and for the long note it prints nothing and succeeds

Scenario: 855d4b1o The limit on the reply holds a deep reply for the most candidates
  Given the reply of a deep recall for 30 candidates: 20 notes judged near with 3 chunks, 3 fields and a newer note, 10 passages, and 6 lessons that supersede a note
  When the lookup is asked at deep
  Then the limit on the reply, at 3 characters to the token, holds the reply

Scenario: 855d4b1p Only the index file of each source that matched is read
  Given a payload of matched chunks of two sources and a chunk of the session's own transcript
  When the lookup is asked
  Then the index file of each of the two sources is read once, and the session's own is not read

Scenario: 855d4b1q A note with no front matter is a candidate whole, and the index reader reports a missing or a corrupt file
  Given a note whose content holds no front matter, and a directory of index files, one of them corrupt
  When the lookup is asked and the reader reads the files
  Then the line holds the whole content, the reader returns the records of a file, and it fails on the missing file and on the corrupt one

Scenario: 855d4b1r A note is near only when the moment sharpens its own claim
  Given a request of the candidate judgement
  When the instructions are read
  Then a near note omits a substantive part of its own claim, the same subject and the same principle, and a note that merely shares a situation or words and would need a different claim is unrelated

Scenario: 855d4b1s A deep request revises only the note's own claim and writes only a durable, atomic lesson
  Given a deep request of the candidate judgement
  When the instructions are read
  Then a revision sharpens the claim a note already makes and never adds a different claim, a status, a plan, a branch, worktree or ticket name, a timestamp or an episode, and a lesson must be durable and atomic and name no ticket or change number in its situation

Scenario: 855d4b1t A deep request holds the notes the session wrote
  Given a vault holding a note written automatically in the session and one written in another session, and a lookup that finds neither
  When a deep request and a glance request are built for the session
  Then the deep request holds the first among its candidates, after the notes that matched and before the passages, and not the second, and the glance request does not hold it
```

### 855d8b The candidates of the lookup

- Verdicts: `covered`, a [note](../../GLOSSARY.md#permanent-note) that states what applies to the moment; `near`, a note that addresses the same situation and omits a substantive part of its own claim, the same subject and the same principle, or that only matches content a newer candidate reverses ([recall skill](../../GLOSSARY.md#recall-skill) Step 2.5C), and not a note that merely shares a situation or words and would need a different claim, which is `unrelated`; `evidence`, a passage that applies; `unrelated`.
- Budget: the lookup is asked for 60 items; the matched set is in the order of the scores; a note whole; a chunk cut to 1,200 characters; at most 30 candidates, at least 10 of them passages when so many matched, and 60,000 characters of text in all, the list ending at the first candidate that does not fit.
- Candidate line: `- <file name or chunk ID> (<date>; <type>; parent; <situation>): <body>` for a note, `parent` only for one from the parent [vault](../../GLOSSARY.md#vault), and `- <chunk ID> (<date>; passage): <text>` for a chunk, written as one line.
- Carry: for each candidate, its ID, kind, type, date, situation, full text (redacted and written as one line, not cut) and whether it came from the parent [vault](../../GLOSSARY.md#vault) or is pending.

## Shared requirements

- [793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials): stated once, in [Moment Judgement](./moment-judgement.md); recall-lookup is one of its owners.
