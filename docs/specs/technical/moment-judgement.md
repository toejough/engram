# Moment Judgement - Technical Spec

## 855d3a Moment Judgement

The command `engram auto moment` (`internal/cli/auto.go`, `internal/auto`, informational). Turns a cue into the request of the cheap-tier judgement of whether it is a [recall moment](../../GLOSSARY.md#recall-moment) and at what [depth](../../GLOSSARY.md#depth-recall-moment), or a learn moment ([855d4a Judging the moment](#855d4a-judging-the-moment)); tested through standard input and output over [transcripts](../../GLOSSARY.md#transcript) of a session. Called by [793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow), [793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow) and [855d2c Mid-turn recall flow](./mid-turn-recall-workflow.md#855d2c-mid-turn-recall-flow).

- Ownership: repository-owned component (the command `engram auto moment`).
- Qualification: Purpose: the judgement, from a window of the conversation and the [cue](../../GLOSSARY.md#cue), of whether it is a recall moment and at what depth, or a learn moment, which the promises that a [note](../../GLOSSARY.md#permanent-note) arrives only at a recall moment ([793a2b Memory arrives with the prompt](../product/automatic-memory.md#793a2b-memory-arrives-with-the-prompt), [855a2e Recall moments within a turn](../product/automatic-memory.md#855a2e-recall-moments-within-a-turn)) and that a lesson is written only for a [turn](../../GLOSSARY.md#turn-automatic-memory) that holds one ([793a2c Lessons are written without the agent](../product/automatic-memory.md#793a2c-lessons-are-written-without-the-agent)) need. Contract: [855d4a Judging the moment](#855d4a-judging-the-moment) is the command's own, apart from the flows that call it, and its scenarios run the command directly through standard input and output; [793d4j Redacting credentials](#793d4j-redacting-credentials) and [793d4m Telling the prompts the user sends](#793d4m-telling-the-prompts-the-user-sends) are rules it shares with the other components that their Owner lines name, stated once, here. Identity: the requirements state only what the command reads, writes, prints and fails with, and the files that realize it are informational, so a refactoring or a change of injection leaves the boundary as it is.
- Required behavior: [Turn Judgement](./turn-judgement.md) is assumed to build the window of conversation lines that a judgement is given, with its redaction and its cuts, as [793d4g Judging a turn](./turn-judgement.md#793d4g-judging-a-turn) states; compatibility verification: the scenarios of [855d4a1 Scenarios](#855d4a1-scenarios) run the moment over a transcript with the window built by the provider's own code (`internal/auto/moment_test.go`).
- Required behavior: [Lesson Writing](./lesson-writing.md) is assumed to keep the records of what [automatic memory](../../GLOSSARY.md#automatic-memory) did as [793d4k Recording what automatic memory did](./lesson-writing.md#793d4k-recording-what-automatic-memory-did) states, which the moment follows for the record of a headless turn it leaves out; compatibility verification: [793d4m1e A headless turn that is not judged is recorded as skipped with the reason](#793d4m1-scenarios) reads the record the moment makes (`internal/cli/auto_headless_test.go`), and [793a2e1f A turn that is not captured is shown with the reason](../product/automatic-memory.md#793a2e1-uat-scenarios) shows it in a real [session](../../GLOSSARY.md#session) (`dev/acceptance/activity_test.go`).
- Supported capabilities: [Automatic Memory](../product/automatic-memory.md).

## 855d4a Judging the moment

Requirement: 855d4a The `moment-judgement` MUST read one JSON object from standard input: the `session`'s ID, the `cue` (`prompt`, `before-tool`, `failure`, `batch-done` or `turn-end`), the `origin` of the [prompt](../../GLOSSARY.md#prompt-automatic-memory), the `text` the [turn](../../GLOSSARY.md#turn-automatic-memory) began with, and the cue's own: the `event` (the tool's name and input, the error, or the calls with their responses) for a tool cue, the turn's ID and final `answer` for `turn-end`. It MUST print nothing and succeed, and look nothing up, when the origin is not one the user sends or is the headless host's own and its capture is not switched on (as [793d4m Telling the prompts the user sends](#793d4m-telling-the-prompts-the-user-sends) states it). Otherwise it MUST print one object `{"request":{...},"carry":{...}}`. The request has `model` of the cheap tier, `effort` `low`, `system`, `prompt` and `maxTokens`, and, for `turn-end` only, the `timeoutMs` of the row of the model call of the capture ([793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow); a recall cue has none, [793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow)). The `system` MUST hold the instructions and nothing of the [session](../../GLOSSARY.md#session): for a recall cue, the rule of the [recall skill](../../GLOSSARY.md#recall-skill)'s description and the cues of the recall guidance, the choice of depth (none when the moment is not one, glance to read, deep when the decision is weighty or irreversible) and the reply shape `{"depth":...}`, with no phrase; for `turn-end`, the description of the [learn skill](../../GLOSSARY.md#learn-skill), which counts the [agent](../../GLOSSARY.md#agent-automatic-memory)'s plain correction of its own earlier claim, even a guess it was asked for, and the reply shape `{"learn":true}` or `{"learn":false}`. The `prompt` MUST hold, between `<window>` and `</window>`, the conversation lines and the line of the cue, built as the window of [793d4g Judging a turn](./turn-judgement.md#793d4g-judging-a-turn) is built (redacted, [793d4j Redacting credentials](#793d4j-redacting-credentials); `<` written `&lt;`), the cue's line being the prompt text, `Agent used <tool>` with its input, `Agent used <tool> failed:` with the error, or `Agent used <tool> returned:` with each response, and none of the instructions. A session with no [transcript](../../GLOSSARY.md#transcript) has no conversation lines and is not an error. The carry holds the cue and the window. It MUST fail, printing nothing and naming the cause, when the input is not a JSON object. For the cue `turn-end`, when it prints nothing and succeeds, it MUST log one line to the debug log, `auto.moment: session=<the session's ID> nothing`, which tells that the capture ended there ([793d2b Turn capture flow](./turn-capture-workflow.md#793d2b-turn-capture-flow)).

The instructions of the [recall moment](../../GLOSSARY.md#recall-moment) MUST say that a question about the status or progress of the work or about what comes next, with no decision in it, is never deep.

Owner: [Moment Judgement](./moment-judgement.md).

Supports: [855a3b1a A prompt that is not a recall moment arrives with nothing](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1b A note whose words match but that does not apply is not delivered](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a2e1a A note arrives before the agent declares its work done](../product/automatic-memory.md#855a2e1-uat-scenarios), [855a2e1b A note arrives after a failure the agent cannot explain](../product/automatic-memory.md#855a2e1-uat-scenarios), [855a2e1c A note arrives before the agent starts building a new approach](../product/automatic-memory.md#855a2e1-uat-scenarios), [855a2b1a A note states the rule and not the story](../product/automatic-memory.md#855a2b1-uat-scenarios), [793a2c1f Guesses, plans and chatter produce no note](../product/automatic-memory.md#793a2c1-uat-scenarios), [793a2c1h A thanks that names no approach produces no note](../product/automatic-memory.md#793a2c1-uat-scenarios).

### 855d4a1 Scenarios

```gherkin
Scenario: 855d4a1a A prompt becomes a request for the cheap tier
  Given the transcript of a session with an earlier turn
  When the moment is asked for a prompt, with the cue prompt
  Then it prints a request for the model haiku at effort low with no time limit, whose system holds the rule of the recall skill, the three depths and the reply shape with no phrase, and whose prompt holds the earlier lines, the prompt's line, and none of the instructions, and a carry that holds the cue and the window

Scenario: 855d4a1b A turn that ended becomes a learn-moment request
  Given the transcript of a session whose turn ended
  When the moment is asked for it, with the cue turn-end
  Then it prints a request whose time limit is 10 seconds and whose system holds the description of the learn skill and the reply shape with learn true or false

Scenario: 855d4a1c Each tool cue puts its event in the window
  Given a session, and the cues before-tool with a tool's name and input, failure with an error, and batch-done with two calls and their responses
  When the moment is asked for each
  Then the window of each ends with the line of its event, one line for each call of the batch, and the text the turn began with is in the window

Scenario: 855d4a1d A prompt, a message or an event cannot close the window or hold a credential
  Given a prompt, a conversation and an event whose texts hold the tag that closes the window, line breaks and a credential of every shape, an event whose input or response holds the value of a key that names a secret and that value holds a quote, and one whose response is an object or a list of JSON written inside a string
  When the moment is asked for each
  Then the prompt of the request holds each tag of the window once, no line holds the character that opens a tag, none of the credentials is in it, and no part of the value of such a key is in it, in the response written inside a string too

Scenario: 855d4a1e A prompt of an origin the user does not send is judged nothing
  Given the origins task-notification, peer and the empty origin
  When the moment is asked for a prompt with each
  Then each prints nothing, succeeds and looks nothing up

Scenario: 855d4a1f A session with no transcript is judged without conversation lines
  Given a session that no transcript holds
  When the moment is asked for a prompt
  Then it prints a request whose window holds only the prompt's line

Scenario: 855d4a1g An input that is not a JSON object fails the command
  Given an input that is not a JSON object
  When the moment is asked
  Then it prints nothing and fails with an error that names the input

Scenario: 855d4a1h A moment that prints nothing for the end of a turn logs it
  Given a debug log, and the cue turn-end with an origin the user does not send
  When the moment is asked for the turn
  Then it prints nothing and succeeds, and the log holds a line for the moment with the session's ID

Scenario: 855d4a1i A question of status with no decision in it is never deep
  Given the prompt what's next?
  When the moment is asked for the prompt
  Then the instructions say that a question about the status or progress of the work or about what comes next, with no decision in it, is never deep
```

## 793d4j Redacting credentials

Requirement: 793d4j The [Moment Judgement](./moment-judgement.md), the [Recall Lookup](./recall-lookup.md), the [Turn Judgement](./turn-judgement.md) and the [Lesson Placement](./lesson-placement.md) MUST redact every text of the [turn](../../GLOSSARY.md#turn-automatic-memory), of a [note](../../GLOSSARY.md#permanent-note) and of a passage shaped as a [recognised credential](../../GLOSSARY.md#recognised-credential) before cutting or joining it and before the model sees it, and the [Lesson Writing](./lesson-writing.md) every field of a lesson before writing it, so that no note holds a recognised credential that appeared in the turn. Each text is first written as one line (control and format characters dropped), then redacted, then cut, so that a credential split by characters that are not shown is whole when redacted. A credential becomes `[redacted]`, the name of an assignment kept. A secret of another shape (a password a sentence gives without `is` or `:`, a token split over two lines, a run of 40 hexadecimal digits, a run of letters only or of digits only, a token of three parts joined by dots that does not begin `eyJ`) is not redacted.

Owner: [Moment Judgement](./moment-judgement.md), [Recall Lookup](./recall-lookup.md), [Turn Judgement](./turn-judgement.md), [Lesson Placement](./lesson-placement.md) and [Lesson Writing](./lesson-writing.md).

Supports: [793a2c1g A credential in the turn is not in the note](../product/automatic-memory.md#793a2c1-uat-scenarios).

### 793d4j1 Scenarios

```gherkin
Scenario: 793d4j1a No text shaped as a credential survives
  Given, for each shape of a recognised credential, texts of the shape with each part at its lower bound and above, among any other text
  When the text is redacted
  Then the credential is not in the text, and the text around it is as it was

Scenario: 793d4j1b The name of an assignment is kept, and a text that is not a credential is not touched
  Given the names password, api_key and secret_token given values, a commit ID of 40 hexadecimal digits, a run of 36 letters, a run of 36 digits, a token of three parts joined by dots that does not begin eyJ, each shape one character short of its lower bound, and a sentence
  When the text is redacted
  Then each name is kept and its value is redacted, and the commit ID, the runs, the token, the shapes one character short and the sentence are as they were

Scenario: 793d4j1c The request holds no credential of the turn
  Given a transcript whose prompts and replies hold credentials of every shape
  When the judgement is given the session
  Then the prompt of the request holds none of them

Scenario: 793d4j1d A note holds no credential of the lesson
  Given a reply whose lessons hold credentials of every shape in every field
  When the lessons are written
  Then no note holds any of them

Scenario: 793d4j1e The value of every name that holds a word of a secret is redacted, whatever the name looks like
  Given names that hold key, token, secret, password, passwd, pwd, credential or private-key, with a word before it, a word after it, or none, in snake, kebab, camel and upper case, each given a value by =, : or is
  When the text is redacted
  Then each name is kept and its value is gone

Scenario: 793d4j1f A sentence gives the value of a name to redact
  Given a name that holds a word of a secret, up to six words of the same sentence, and a value given by is or :
  When the text is redacted
  Then the name and the words are kept and the value is gone, and a sentence with more than six words between is left as it was

Scenario: 793d4j1g A Bearer credential is redacted from 12 characters
  Given the word Bearer followed by 11 characters, and by 12
  When the text is redacted
  Then the first is as it was and the second is redacted

Scenario: 793d4j1h A credential that characters that are not shown split is not in the request
  Given a message, an answer and a note that hold a credential of each shape, a zero-width character, a soft hyphen, a byte-order mark, a bidirectional override or a control character inside it
  When the judgement is given the session
  Then the prompt of the request holds none of the credentials

Scenario: 793d4j1i A credential that characters that are not shown split is not in the note
  Given a reply whose lessons hold a credential of each shape in every field, a zero-width character, a soft hyphen, a byte-order mark, a bidirectional override or a control character inside it
  When the lessons are written
  Then no note holds any of them
```

## 793d4m Telling the prompts the user sends

Requirement: 793d4m The [Moment Judgement](./moment-judgement.md), the [Prompt Delivery](./prompt-delivery.md) and the [Conversation Sweep](./conversation-sweep.md) MUST each read the `origin` of the JSON object they are given, the kind Claude Code stamps on a [prompt](../../GLOSSARY.md#prompt-automatic-memory), and act only for a prompt the user sends, one whose origin is `composer`, `bridge`, `slack-ping` or `sdk`; for any other origin, including none, each MUST print nothing, succeed and look nothing up. The `moment-judgement` for the cue `turn-end` and the `conversation-sweep` MUST do the same for the origin `sdk`, the headless host's own [turn](../../GLOSSARY.md#turn-automatic-memory) (`claude -p`, the SDK), unless the user switched on capturing such turns with the flag `--capture-sdk` or the environment variable `ENGRAM_AUTO_CAPTURE_SDK` set to a true value (`1`), so that a script run with `claude -p` under the user's own home writes and indexes nothing by default. When the `moment-judgement` leaves out a turn for that reason it MUST record that as [793d4k Recording what automatic memory did](./lesson-writing.md#793d4k-recording-what-automatic-memory-did) records a reply that is not a list, a record of the kind `skipped` with no slug and a reason that says the turn was not captured because headless prompts are captured only when `ENGRAM_AUTO_CAPTURE_SDK` is set, in the directory of its `--records` flag, else `auto` in the engram data directory, and still print nothing and succeed; it MUST record nothing for any other origin it leaves out. The definition is theirs: the [mod](../../GLOSSARY.md#mod) passes the origin it received and decides nothing.

Owner: [Moment Judgement](./moment-judgement.md), [Prompt Delivery](./prompt-delivery.md) and [Conversation Sweep](./conversation-sweep.md).

Supports: [793a2d1b Automatic memory's work starts no session of its own](../product/automatic-memory.md#793a2d1-uat-scenarios), [793a2e1f A turn that is not captured is shown with the reason](../product/automatic-memory.md#793a2e1-uat-scenarios). [793a2c Lessons are written without the agent](../product/automatic-memory.md#793a2c-lessons-are-written-without-the-agent) and [793a2f The conversation becomes searchable memory](../product/automatic-memory.md#793a2f-the-conversation-becomes-searchable-memory) name the headless [session](../../GLOSSARY.md#session), and the scenario above is the one that leaves it out; every other scenario is raised by a prompt the user sends, which this requirement lets through.

### 793d4m1 Scenarios

```gherkin
Scenario: 793d4m1a A prompt of an origin the user sends is judged and delivered for
  Given a vault holding a note that applies to a prompt, the transcript of a session whose turn began with that prompt, and each of the origins composer, bridge, slack-ping and sdk, the moment asked to judge the turns of headless prompts
  When the moment is given the prompt with the cue prompt and then the turn with the cue turn-end, and the delivery is given the prompt, with each origin
  Then the moment prints a request for each cue, and the delivery prints its lines, with each

Scenario: 793d4m1b A prompt of any other origin is judged and delivered nothing
  Given the same vault and transcript, and any origin but those four, including none: task-notification, scheduled-trigger, peer, channel, plugin, unclassified and the empty origin
  When the moment is given the prompt with each cue, and the delivery is given the prompt, with it
  Then each prints nothing, succeeds, and looks nothing up

Scenario: 793d4m1c The turn of a headless prompt is judged only when the user asks
  Given the transcript of a session whose turn began with a prompt of the origin sdk
  When the moment is given the turn with the cue turn-end, without the flag --capture-sdk and with it
  Then without it the moment prints nothing and succeeds, and with it the moment prints its request, and the prompt of the origin sdk is judged with the cue prompt either way

Scenario: 793d4m1d A turn that a background task's notification starts is not judged
  Given the user has run engram update, and a session in which the agent may run a sleep command but may not run engram commands
  When the agent starts a task in the background in a turn the user began, and the task's notification starts a turn of its own that the agent answers
  Then automatic memory has judged one turn, that of the user's, and the vault holds no note

Scenario: 793d4m1e A headless turn that is not judged is recorded as skipped with the reason
  Given the transcript of a session whose turn began with a prompt of the origin sdk, a place for the records, and, in turn, an origin the user does not send
  When the moment is given the turn with the cue turn-end, without the flag --capture-sdk and with it
  Then without it the moment prints nothing and succeeds, and one record of the kind skipped, with the session's ID and the turn's ID and a reason that says the turn was not captured and names ENGRAM_AUTO_CAPTURE_SDK, is made; with it no record is made, and for the other origin no record is made
```
