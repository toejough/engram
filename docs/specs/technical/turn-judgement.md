# Turn Judgement - Technical Spec

## 793d3d Orientation

The command `engram auto judge` (`internal/cli/auto.go`, `internal/auto`, informational). Turns a learn moment into the request of the mid-tier judgement of the lessons of four kinds, redacting credentials first ([793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials)), owning the choice of the turn's lines and the instructions (the call is the [mod](../../GLOSSARY.md#mod)'s); tested through standard input and output over [transcripts](../../GLOSSARY.md#transcript) of a session. Called by [793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow).

## 793d4g Judging a turn

Requirement: 793d4g The `turn-judgement` MUST read one JSON object from standard input (the `session`'s ID, the final `answer`, the `text` the [turn](../../GLOSSARY.md#turn-automatic-memory) began with, the `origin` of that [prompt](../../GLOSSARY.md#prompt-automatic-memory), the `reply` of the learn-moment judgement and its `carry`), [read the reply](../../GLOSSARY.md#reply-automatic-memory), print nothing and succeed when it says the turn is not a learn moment (`{"learn":false}`) and fail, naming the judgement, when it is not `{"learn":true}` or `{"learn":false}`, the failure holding, after the words `learn-moment reply`, the first 40 and the last 40 characters of the reply, redacted, so that the line shown for the failure, which holds 240 characters at most ([793d4p Telling what automatic memory did](./prompt-delivery.md#793d4p-telling-what-automatic-memory-did)), keeps both ends. When the object holds no `reply` at all, the turn was handed over by the [mod](../../GLOSSARY.md#mod) of an older engram: it MUST print nothing, log the line below and succeed, and record that as [793d4k Recording what automatic memory did](./lesson-writing.md#793d4k-recording-what-automatic-memory-did) records a reply that is not a list, a record of the kind `skipped` with no slug and a reason that says the turn ended while [`engram update`](../../GLOSSARY.md#engram-update) replaced the memory mod, in the directory of its `--records` flag, else `auto` in the engram data directory. Otherwise it MUST find the [session](../../GLOSSARY.md#session)'s [transcript](../../GLOSSARY.md#transcript) as [793d4d Telling what a session holds](./session-context.md#793d4d-telling-what-a-session-holds) does in the directory of [793d4f Finding the transcripts of sessions](./prompt-delivery.md#793d4f-finding-the-transcripts-of-sessions), and print the request for the model as one object `{"request":{...},"carry":{}}`: `model` of the mid tier, `system`, `prompt`, `maxTokens` and `timeoutMs`, the last the row of the lessons judgement ([793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow)). The `system` MUST hold the instructions of the judgement and nothing of the transcript, and the `prompt` none of the instructions, so that the model reads them as instructions. The instructions are the [learn skill](../../GLOSSARY.md#learn-skill)'s Step 2: the four kinds of lesson (a correction by the user, an explicit request to remember, a reversal of what was presented and then overturned, which is written as a `feedback` lesson (its behavior what the earlier reasoning did wrong, its impact the cost, its action the guard, naming the corrected value) and not as a `fact`, and an approach confirmed by the user's praise of a specific behavior or by an observable outcome that resolved an uncertainty), a confirmed approach that is a reusable ordered multi-step procedure for a recurring task being a `runbook` (its `situation`, its `done_when` and its numbered `body`), which replaces the `fact` of an explicit request when the user also says to do it going forward, and any other lesson a `feedback` or a `fact`; the general rule and not the story of the turn, with the exact names, commands, flags and values it depends on; no lesson for a guess, a plan, chatter or a bare thanks; and the reply shape, a JSON array of lessons, each with `kind`, `slug`, `situation` and `covered` beside the fields of its kind. The prompt holds, between `<window>` and `</window>`, the context lines, the line `--- the turn ---` and the turn's lines, and, between `<notes>` and `</notes>`, the [notes](../../GLOSSARY.md#permanent-note) nearest to the turn (what the lookup of [792b24c5 The matched set](./query-engine.md#792b24c5-the-matched-set) returns for the text of the turn's prompts as its message and only phrase, each note as a line of a block, cut to 400 characters) or `(none)`.

A prompt is a row of the transcript that the user sent; a row the [harness](../../GLOSSARY.md#harness) wrote is none and makes no line: a row marked as meta (the body of a [skill](../../GLOSSARY.md#skill) the [agent](../../GLOSSARY.md#agent-automatic-memory) invoked), a row stamped with an origin the user does not send ([793d4m Telling the prompts the user sends](./moment-judgement.md#793d4m-telling-the-prompts-the-user-sends); the transcript stamps the kind `human` on a prompt the user typed, `task-notification` on a finished task the harness delivered and `peer` on a message of another session, and of these only `human` is the user's, beside the origins of the prompt event) and a row whose text begins as [792b15c3 Harness noise](./transcript-reader.md#792b15c3-harness-noise) (a system reminder, a skill's base directory, the caveat or output of a local command, a task notification); a block of a prompt that begins so is left out, and a slash command's prompt is its arguments, none when it has none. The turn begins at the last prompt with the input `text` (else the last prompt) and ends before the next prompt and after the last line of the agent, which is the final text, added as the last line when no line of the turn holds its first 80 characters; the conversation before it is the context. A line is one message (`User: `, `Agent: ` or `Agent used ` and the tool's name; a break written ` ⏎ `), redacted ([793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials)) before it is cut or joined, then cut to its first 1,200 characters; a `<` in a line of the window or of a note MUST be written `&lt;`, which the instructions say, so that no message or note can close the window or open a section; the context is its last 6 lines; the window is cut to 8,000 characters, the oldest context first and then the turn's middle, replaced by the line `--- some of the turn is left out ---`. The command MUST read the transcript once and redact only the lines of the window, so that its time does not grow with the session: 1 MB within 2 seconds.

It MUST print nothing and succeed when the transcript holds no prompt, and print nothing and fail, naming the cause, when the input is not a JSON object, no [vault](../../GLOSSARY.md#vault) exists at the vault's path, the session has no transcript in the directory of transcripts, or the lookup fails, the cause holding no text of the turn's prompts: the capture is then recorded once, with its reason, and the model is not asked.

When it prints nothing and succeeds it MUST log one line to the debug log, `auto.judge: session=<the session's ID> nothing`, which tells that the capture ended there ([793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow)).

The instructions MUST keep a lesson only when it is durable (still true and useful to a future task that has nothing to do with this one: not status, a plan, a decision about the work in hand, a branch, worktree, ticket or change name, a timestamp, a finding about the incident of this session or an episode, even when the user asks to remember it) and atomic (one claim; a second claim is a lesson that is durable too, or is left out), and its situation MUST name no ticket, issue or change number or branch unless the lesson is about it. The notes of the request MUST be the notes nearest to the turn and, after them, the notes whose source says they were written automatically in the session, at most the last 20 and none twice, read from the vault, so that a lesson one of them states is `covered`.

Owner: [Turn Judgement](./turn-judgement.md).

Supports: [793a2c1a A correction becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1b A request to remember becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1c An approach the user confirmed becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1d The agent's own correction becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1e An approach that succeeded with evidence becomes a note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1f Guesses, plans and chatter produce no note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1g A credential in the turn is not in the note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1h A thanks that names no approach produces no note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1i A request to remember is written when the agent says it did not save it](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a3c1a A lesson an earlier note states is not written again](../product/automatic-memory.md#793a3c1-uat-scenarios), [793a3c1b A lesson the agent saved itself is not written again](../product/automatic-memory.md#793a3c1-uat-scenarios), [793a3c1c A lesson no earlier note states is written](../product/automatic-memory.md#793a3c1-uat-scenarios), [855a2b1a A note states the rule and not the story](../product/automatic-memory.md#855a2b1-uat-scenarios), [855a2b1b The note states the lesson the learn skill writes](../product/automatic-memory.md#855a2b1-uat-scenarios), [855a2b1c A confirmed procedure is written as a runbook](../product/automatic-memory.md#855a2b1-uat-scenarios), [855a2g1a A status of the work is not written](../product/automatic-memory.md#855a2g1-uat-scenarios), [855a2g1b A lesson the session wrote is not written twice](../product/automatic-memory.md#855a2g1-uat-scenarios), [855a2g1c A correction made in work on a ticket is written with no ticket in its situation](../product/automatic-memory.md#855a2g1-uat-scenarios).

### 793d4g1 Scenarios

```gherkin
Scenario: 793d4g1a The request holds the instructions, the context and the turn
  Given the transcript of a session with a first turn and a second turn in which the user corrected the agent, and a reply that says the turn is a learn moment
  When the judgement is given the session and the second turn's final text
  Then it prints a request for the model sonnet whose time limit is 30 seconds and whose prompt holds the first turn's lines, the line --- the turn ---, and the second turn's lines, each opening with its label, and a carry

Scenario: 793d4g1b The final text is added when the transcript lacks it
  Given the transcript of a session whose last row is the user's prompt
  When the judgement is given the session and a final text
  Then the last line of the turn is the final text, labelled as the agent's

Scenario: 793d4g1c A session with no prompt in its transcript is judged nothing
  Given the transcript of a session with no prompt
  When the judgement is given the session
  Then it prints nothing and succeeds

Scenario: 793d4g1d An input that is not a JSON object fails the command
  Given an input that is not a JSON object
  When the judgement is given it
  Then it prints nothing and fails with an error that names the input

Scenario: 793d4g1e A message cannot make a line of its own
  Given messages whose text holds line breaks of every kind, control characters and text that reads as a label
  When the lines of the turn are made
  Then there is exactly one line for each message, and each opens with its label

Scenario: 793d4g1f A line is cut after its credentials are redacted
  Given a message whose text is longer than a line may be, with a token that the cut falls in
  When the lines of the turn are made
  Then no part of the token is in the request

Scenario: 793d4g1h The nearest notes are in the request
  Given a vault holding a note that says when the staging database is reset, and the transcript of a session whose turn asks about the staging database
  When the judgement is given the session
  Then the request lists the staging note, with its file name, its situation and its body on one line

Scenario: 793d4g1i A failed lookup fails the command and prints nothing
  Given a lookup that fails
  When the judgement is given the transcript of a session with a prompt
  Then it fails with an error that names the lookup's failure and holds no text of the turn's prompts, and it prints nothing

Scenario: 793d4g1j A note cannot reshape the window
  Given a note whose situation and body hold line breaks, a credential and more than 400 characters
  When the request is made
  Then the note is one line of the section of notes, the credential is redacted, and the body is cut to 400 characters

Scenario: 793d4g1g A long conversation is cut to the bound
  Given a transcript with more context than the window holds, and a turn with more lines than it holds
  When the judgement is given the session
  Then the context is its last 6 lines or fewer, the window is no longer than 8,000 characters, the first line of the turn and its last lines are kept, and the line that says that some of the turn is left out stands where they were

Scenario: 793d4g1k A prompt of text blocks is one prompt, and the result of a tool is none
  Given a transcript whose last prompt is made of two text blocks and an image, followed by a row that holds only the result of a tool
  When the judgement is given the session
  Then the turn begins with one line for the prompt, holding the text of both blocks, and the result of the tool makes no line

Scenario: 793d4g1l The body of a skill is no prompt
  Given the transcript of a session whose turn began with a prompt of the user, in which the agent invoked a skill and the harness wrote the skill's body as a row marked as meta
  When the judgement is given the session
  Then the request has no line for the skill's body, and the turn begins with the user's prompt

Scenario: 793d4g1m A system reminder in the row of a prompt is not part of the prompt
  Given a transcript whose last prompt is a row of two text blocks, the user's words and a system reminder, followed by a row that holds only a system reminder
  When the judgement is given the session
  Then the prompt is one line of the user's words alone, and the row of the system reminder makes no line

Scenario: 793d4g1n A task notification delivered into a turn is no prompt
  Given the transcript of a session whose turn began with a prompt of the user, and in which a task notification was delivered as a row of the user's before the agent's last text
  When the judgement is given the session
  Then the request has no line for the notification, and the turn begins with the user's prompt

Scenario: 793d4g1o A command that ran locally is no prompt, and a slash command is its arguments
  Given the transcript of a session whose turn began with a prompt of the user and holds, after it, the caveat, the name with no arguments and the output of a command that ran locally, and a transcript whose prompt is a slash command given arguments
  When the judgement is given each
  Then none of the rows of the command that ran locally is a line of the first request, and the prompt of the second is the arguments

Scenario: 793d4g1p A row of an origin the user does not send is no prompt
  Given a transcript whose turn began with a prompt of the user, and a row of the user's stamped with the origin peer whose text opens with none of the marks of the harness, delivered before the agent's last text
  When the judgement is given the session
  Then the request has no line for that row, and the turn begins with the user's prompt

Scenario: 793d4g1w A session with no transcript fails the command
  Given a session that no transcript holds, and no session at all
  When the judgement is given each
  Then it prints nothing and fails each time, with an error that names the session and the directory of the transcripts

Scenario: 793d4g1x A vault that does not exist fails the command
  Given a path that holds no vault, and the transcript of a session with a prompt
  When the judgement is given the session
  Then it prints nothing, fails with an error that names the path, and looks nothing up

Scenario: 793d4g1q A long transcript is judged in time
  Given the transcript of a session of 1 MB, of which the turn is the last prompt
  When the judgement is given the session
  Then it prints the request within 2 seconds

Scenario: 793d4g1r A prompt typed ahead does not begin the turn of the one before it
  Given the transcript of a session whose first turn ended with the agent saying that its guess was wrong and what the file says, and whose second prompt, typed ahead, is already in the transcript
  When the judgement is given the session, the text of the first prompt and the first turn's final text
  Then the turn of the request is the first turn's lines, and the second prompt is no line of the request

Scenario: 793d4g1s A turn that a notification started after it is not part of it
  Given the transcript of a session whose turn began with a prompt of the user and ended with the agent's text, followed by a task notification and the lines of the turn it started
  When the judgement is given the session, the text of the prompt and the final text
  Then the turn of the request ends with the final text, and holds nothing of the turn the notification started

Scenario: 793d4g1t A text that no prompt has begins the turn at the last prompt
  Given the transcript of a session with two prompts, and a text that neither has
  When the judgement is given the session and that text
  Then the turn of the request begins at the last prompt

Scenario: 793d4g1u The instructions are the system of the request
  Given the transcript of a session with a prompt
  When the judgement is given the session
  Then the system of the request holds the instructions of the judgement and nothing of the transcript, and its prompt holds the window and the notes and none of the instructions

Scenario: 793d4g1v A message or a note cannot close the window or open a section
  Given messages and notes whose texts hold the tags that close and open the window and the section of notes, and other uses of the character that opens a tag
  When the request is made
  Then no line of the window or of the section of notes holds that character, each holds &lt; in its place, and the prompt holds each tag of the window and of the notes once
```

### 855d4g1 Scenarios

```gherkin
Scenario: 855d4g1a A turn that is not a learn moment is judged nothing
  Given the transcript of a session with a prompt, and the replies learn false, and learn true
  When the judgement is given each
  Then with the first it prints nothing and succeeds, and with the second it prints a request

Scenario: 855d4g1b A fenced reply is read and a reply that is not a moment reply fails the command
  Given a reply learn true in a code fence, replies that are not JSON and that hold no learn field
  When the judgement is given each
  Then the first prints a request, and the others print nothing and fail with an error that names the judgement

Scenario: 855d4g1c The instructions name the four kinds and the runbook
  Given the transcript of a session with a prompt, and a reply that says learn
  When the judgement is given the session
  Then the system of the request names a correction, an explicit request to remember, a reversal and a confirmed approach, says that a reversal is written as a feedback lesson and not as a fact, that a confirmed multi-step procedure is a runbook with its situation, done_when and numbered body, that the runbook replaces the fact for a request to do it going forward, and holds none of the transcript

Scenario: 855d4g1d A judgement that prints nothing logs it
  Given a debug log, the transcript of a session with a prompt, and a reply learn false
  When the judgement is given the session
  Then it prints nothing and succeeds, and the log holds a line for the judgement with the session's ID

Scenario: 855d4g1e A prompt the user typed in an interactive session is in the window
  Given the transcript of an interactive session, whose rows are stamped human, with a prompt of the user, a tool call, a row stamped task-notification and a row stamped peer
  When the moment and the judgement are given the session and the turn's final text
  Then each request holds the user's prompt, and neither holds the row stamped task-notification or the row stamped peer

Scenario: 855d4g1f A reply that is no moment reply fails with its ends
  Given a reply of 232 characters that is not JSON and holds a credential
  When the judgement is given it
  Then it fails with an error that names the learn-moment reply and holds its first 40 and its last 40 characters, with the credential redacted, and the line of activity that the record of the failure makes, as the mod hands the failure over, holds both ends within 240 characters

Scenario: 855d4g1g A turn handed over without a reply is skipped and is no failure
  Given the transcript of a session with a prompt, and a turn handed over with no reply field
  When the judgement is given the session
  Then it prints nothing and succeeds, and a record of the kind skipped says the turn ended while engram update replaced the memory mod

Scenario: 855d4g1h The instructions keep only a durable and atomic lesson
  Given a request of the judgement of a turn
  When the instructions are read
  Then a lesson is kept only when it is durable and atomic, and status, a plan, a decision about the work in hand, a branch, worktree or ticket name, a timestamp and a finding about the incident of the session are not, even when the user asks to remember them

Scenario: 855d4g1i A situation names no ticket or change number
  Given a request of the judgement of a turn
  When the instructions are read
  Then the situation of a lesson names no ticket, issue or change number and no branch, unless the lesson is about that ticket

Scenario: 855d4g1j The notes the session wrote are in the request once
  Given a vault holding 22 notes written automatically in the session on one day, one written in the session on an earlier day whose ID is the highest, a note written automatically in another session, and a lookup that finds the first among its notes or does not
  When the judgement is given the session
  Then the notes of the request hold the notes nearest to the turn first, then the last 20 of the session's own by the day they were created and then by Luhmann ID, each once, and not the note of the other session
```

## Shared requirements

- [793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials): stated once, in [Moment Judgement](./moment-judgement.md); turn-judgement is one of its owners.
