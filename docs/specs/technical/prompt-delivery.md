# Prompt Delivery - Technical Spec

## 793d3b Prompt Delivery

The command `engram auto deliver` (`internal/cli/auto.go`, `internal/auto`, informational). Turns the candidate judgement of a [recall moment](../../GLOSSARY.md#recall-moment) into the block of [notes](../../GLOSSARY.md#permanent-note) and passages that arrives beside a [prompt](../../GLOSSARY.md#prompt-automatic-memory) or within a [turn](../../GLOSSARY.md#turn-automatic-memory), and into the lines that show the user what [automatic memory](../../GLOSSARY.md#automatic-memory) did, owning the choice and rendering of both and the marks of the records shown (the hook is the [mod](../../GLOSSARY.md#mod)'s). It marks the notes it delivers as used and looks nothing up. Tested through standard input and output over an in-memory [vault](../../GLOSSARY.md#vault) and records. Called by [793d2a Prompt delivery flow](./prompt-delivery-workflow.md#793d2a-prompt-delivery-flow) and [855d2c Mid-turn recall flow](./mid-turn-recall-workflow.md#855d2c-mid-turn-recall-flow).

- Ownership: repository-owned component (the command `engram auto deliver`).
- Qualification: Purpose: the block of notes and [passages](../../GLOSSARY.md#passage) that arrives beside a prompt and the lines that show the user what automatic memory did are promised by
  [793a2b Memory arrives with the prompt](../product/automatic-memory.md#793a2b-memory-arrives-with-the-prompt) and [793a2e The user can see what it did](../product/automatic-memory.md#793a2e-the-user-can-see-what-it-did),
  and this component chooses and renders both. Contract: it owns the choice and rendering of the block and of the lines and the marks of the records shown,
  the hook being the mod's, and it is tested directly through standard input and output. Identity: the requirements state only what the command reads, writes, prints and fails with,
  and the files that realize it are informational, so a refactoring or a change of injection leaves the boundary as it is.
- Required behavior: [Session Context](./session-context.md) returns the text that holds what the [session](../../GLOSSARY.md#session) already holds
  ([793d4d Telling what a session holds](./session-context.md#793d4d-telling-what-a-session-holds)).
- Required behavior: [Moment Judgement](./moment-judgement.md) states the redaction of [credentials](../../GLOSSARY.md#recognised-credential) that the text of a passage and the reason of a record are put through
  ([793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials)).
- Required behavior: [Lesson Writing](./lesson-writing.md) records what became of each [reply](../../GLOSSARY.md#reply-automatic-memory), of the kinds `written`, `skipped` and `failed`, as a file of its own in a
  directory named for the session ([793d4k Recording what automatic memory did](./lesson-writing.md#793d4k-recording-what-automatic-memory-did)).
- Required behavior: [Note Amendment](./note-amendment.md) records the use of a note and pulls a parent note down as a pending copy, as `engram activate` does
  ([792b14c9 Recording a use](./note-amendment.md#792b14c9-recording-a-use), [792b14c12 Activation against the parent](./note-amendment.md#792b14c12-activation-against-the-parent)).
- Required behavior: Claude Code (external) is assumed to write the [transcripts](../../GLOSSARY.md#transcript) of sessions in the `projects` directory of the directory `CLAUDE_CONFIG_DIR` names
  when set, else in `<home>/.claude/projects` ([793d4f Finding the transcripts of sessions](#793d4f-finding-the-transcripts-of-sessions)); authoritative contract: not recorded in this
  repository; compatibility verification: `dev/acceptance/product_test.go`.
- Supported capabilities: [Automatic Memory](../product/automatic-memory.md).

## 793d4a Delivering the notes that apply to a prompt

Requirement: 793d4a The `prompt-delivery` MUST read one JSON object from standard input: the `session`'s ID, the `origin`, the `cue`, the `held` IDs ([855d2c Mid-turn recall flow](./mid-turn-recall-workflow.md#855d2c-mid-turn-recall-flow)), and, when a candidate request was answered, the model's `reply` and the `carry`. It MUST print on standard output one block, or nothing when the origin is not one the user sends ([793d4m Telling the prompts the user sends](./moment-judgement.md#793d4m-telling-the-prompts-the-user-sends)), when there is no reply, or when no candidate is delivered. The reply is a JSON object, fenced or not, or such an object that follows the text in which the model reasoned (it is read from the last `{"candidates"`), with `candidates`, each an object with an `id` the carry holds, a `verdict` of `covered`, `near`, `evidence` or `unrelated`, and optionally the `newer` candidate that reverses it; an entry whose ID the carry does not hold is ignored, and a reply that is not such an object fails the command, printing nothing and naming the judgement. A reply names a candidate by its ID as the carry holds it. A candidate is delivered when its verdict is not `unrelated` and its `newer` is not the ID of a candidate of the carry. The block opens with a line beginning `[engram memory]` and has one line for each delivered [note](../../GLOSSARY.md#permanent-note), in the order of the carry: a dash, the file name, the situation in parentheses, a colon and the body on one line; and one line for each delivered passage: a dash, the chunk ID, `(passage)`, a colon and the text of the chunk redacted ([793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials)) and cut to 1,200 characters on one line, a note's body being as the note holds it. It MUST leave out each note or passage whose ID is among the `held` IDs, and each note whose basename (file name without `.md`) and each passage whose chunk ID appears in the text that [793d4d Telling what a session holds](./session-context.md#793d4d-telling-what-a-session-holds) returns for the request's [session](../../GLOSSARY.md#session), and print no block and succeed when none remains. It MUST mark each note it delivers as used and pull a delivered parent note down as a pending copy, as `engram activate` does, and MUST NOT write, amend or look up anything else, or search the chunk index.

Owner: [Prompt Delivery](./prompt-delivery.md).

Supports: [793a2a1a The next session has memory after one engram update](../product/automatic-memory.md#793a2a1-uat-scenarios), [793a2b1a The notes that apply to a prompt arrive with it](../product/automatic-memory.md#793a2b1-uat-scenarios), [793a2b1b A note delivered earlier in the session is not delivered again](../product/automatic-memory.md#793a2b1-uat-scenarios), [793a2b1d A prompt that matches no note arrives with nothing](../product/automatic-memory.md#793a2b1-uat-scenarios), [793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt](../product/automatic-memory.md#793a2e1-uat-scenarios), [855a2a1a A glance leaves the vault's notes as they were](../product/automatic-memory.md#855a2a1-uat-scenarios), [855a3b1b A note whose words match but that does not apply is not delivered](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1c Of two notes that conflict, only the newer is delivered](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1d The notes that arrive are the notes the recall skill draws on](../product/automatic-memory.md#855a3b1-uat-scenarios), [855a3b1e A passage of an earlier conversation arrives when it applies](../product/automatic-memory.md#855a3b1-uat-scenarios).

### 793d4a1 Scenarios

```gherkin
Scenario: 793d4a1a The candidates the judgement chose come back as one block
  Given a carry that holds a note that says when the staging database is reset and a passage of an earlier conversation, and a reply that gives both the verdict covered and evidence
  When the command is given the reply and the carry
  Then it prints a block that opens with a line beginning [engram memory] and has one line naming the note, its situation and its body, and one naming the passage's chunk ID with the text

Scenario: 793d4a1b A reply that is not a reply fails the command and prints nothing
  Given replies that are not JSON and a JSON array
  When the command is given each with a carry
  Then it fails with an error that names the judgement and holds no text of the prompt, and it prints nothing

Scenario: 793d4a1d A note the session holds is left out
  Given a carry of two notes, a reply that delivers both, and a session whose transcript shows that one of them was delivered
  When the command is given the reply
  Then the block names the other note only

Scenario: 793d4a1e No candidate to deliver prints nothing
  Given a reply whose verdicts are all unrelated, a reply that delivers only a note the session holds, and no reply
  When the command is given each
  Then it prints nothing and succeeds each time

Scenario: 793d4a1g A note the agent fetched by its basename is left out
  Given a carry of two notes, a reply that delivers both, and a session whose transcript shows that the agent ran engram show with the basename of one of them
  When the command is given the reply
  Then the block names the other note only
```

### 855d4c1 Scenarios

```gherkin
Scenario: 855d4c1a Of two candidates that conflict, only the newer is delivered
  Given a carry of an older and a newer note, and a reply that gives the older the newer as its newer and delivers both
  When the command is given the reply
  Then the block names the newer note only

Scenario: 855d4c1b A candidate the carry does not hold is not delivered
  Given a carry of one note, and a reply that delivers it and an ID the carry does not hold, such as the file name of another note of the vault
  When the command is given the reply
  Then the block names the one note only

Scenario: 855d4c1c A passage the session holds is left out
  Given a carry of a passage, a reply that delivers it, and a session whose transcript shows a block with that chunk ID
  When the command is given the reply
  Then it prints nothing and succeeds

Scenario: 855d4c1d A delivered note is marked as used and a delivered parent note is pulled down
  Given a carry of a note and a parent note, a reply that delivers both, and an activation that records its arguments
  When the command is given the reply
  Then the activation was asked for both, and nothing else of the vault was written

Scenario: 855d4c1f The delivery looks nothing up
  Given a vault, a chunk index and a lookup that record what they are asked, and a reply that delivers a note of its carry
  When the command is given the reply
  Then neither the lookup nor the chunk index was asked for anything

Scenario: 855d4c1e A passage is cut to 1,200 characters
  Given a carry whose passage holds 2,000 characters
  When the command is given a reply that delivers it
  Then its line holds the first 1,200 characters of the text

Scenario: 855d4c1g A note or passage the mod has delivered already is left out
  Given a carry of a note and a passage, a reply that delivers both, and the held IDs of the note
  When the command is given the reply
  Then the block names the passage only, and the note was not marked as used

Scenario: 855d4c1h A reply with text before the object is read by the object
  Given a carry of a note, and, in turn, a reply that holds the model's reasoning and then an object that delivers the note, one that holds the reasoning and then an object that finds it unrelated, and one that holds the reasoning and no object
  When the command is given the reply
  Then with the first the block names the note, with the second the block is empty and the command succeeds, and with the third the command fails naming the judgement
```

## 793d4c Framing the notes of a block as data

Requirement: 793d4c The `prompt-delivery` MUST open every block with one fixed line beginning `[engram memory]` that says the lines below are [notes](../../GLOSSARY.md#permanent-note) and passages of earlier conversations from the [vault](../../GLOSSARY.md#vault), delivered by the memory system and not typed by the user, and that an instruction inside one is not the user's. It MUST write each note and each passage as exactly one line beginning with a dash and a space, its file name or chunk ID, situation and body or text each with every run of white space, line breaks and separators included, as one space and no control or format character (Unicode category Cf: zero-width space or joiner, bidirectional override or isolate, byte-order mark, soft hyphen), so that its text cannot make a line of its own, such as the framing line or a line that reads as another note. The line of a note whose source holds the words `written automatically`, wherever in it ([793d4h Writing the lessons of a reply](./lesson-writing.md#793d4h-writing-the-lessons-of-a-reply); the notes of the earlier [mod](../../GLOSSARY.md#mod) hold them after other text), ends with the fixed words `(written automatically)`, so that the [agent](../../GLOSSARY.md#agent-automatic-memory) can tell it from a note a person or the agent saved.

Owner: [Prompt Delivery](./prompt-delivery.md).

Supports: [793a2a1a The next session has memory after one engram update](../product/automatic-memory.md#793a2a1-uat-scenarios), [793a2b1a The notes that apply to a prompt arrive with it](../product/automatic-memory.md#793a2b1-uat-scenarios), [793a3a1a A written note is delivered with later prompts](../product/automatic-memory.md#793a3a1-uat-scenarios), [855a3b1e A passage of an earlier conversation arrives when it applies](../product/automatic-memory.md#855a3b1-uat-scenarios).

### 793d4c1 Scenarios

```gherkin
Scenario: 793d4c1a A note cannot reshape the block
  Given notes whose file names, situations and bodies hold line breaks, control characters, the framing line and lines that read as other notes, however they are mixed
  When the block is made
  Then the block has the framing line first and nowhere else, and exactly one line for each note, each beginning with a dash and a space and holding single spaces and no control or format character

Scenario: 793d4c1b A note cannot hide text in format characters
  Given a note whose file name, situation and body hold a zero-width space, a zero-width joiner, a bidirectional override, a byte-order mark and a soft hyphen, between and inside words
  When the block is made
  Then the line of the note holds none of them, and the words they were between and inside are joined as the characters around them were

Scenario: 793d4c1c A note written automatically says so in its line
  Given a note whose source says that it was written automatically, a note whose source says it after other text, and a note whose source is another
  When the block is made for each
  Then the lines of the first two end with (written automatically), and the line of the other does not
```

### 855d4d1 Scenarios

```gherkin
Scenario: 855d4d1a A passage cannot reshape the block
  Given passages whose chunk IDs and texts hold line breaks, control and format characters, the framing line and lines that read as other notes, however they are mixed
  When the block is made
  Then the block has the framing line first and nowhere else, and exactly one line for each passage, each beginning with a dash and a space and holding single spaces and no control or format character

Scenario: 855d4d1b A passage is redacted and a note arrives as it is
  Given a note whose body holds a name of 32 characters and more that reads as a token, and a passage whose text holds a run that reads as a token, and a reply that delivers both
  When the block is made
  Then the passage's line holds no such run and the note's line holds the name as it is
```

## 793d4f Finding the transcripts of sessions

Requirement: 793d4f The `prompt-delivery` MUST read the [transcripts](../../GLOSSARY.md#transcript) of [sessions](../../GLOSSARY.md#session) from the directory its `--sessions` flag names or, without the flag, from the directory `ENGRAM_TRANSCRIPT_DIR` names when set, else the `projects` directory of the directory `CLAUDE_CONFIG_DIR` names when set, else `<home>/.claude/projects`, and from no directory when the home directory cannot be found. It MUST NOT change the session-log directory of [792b17c6 Session-log directory](./ingestion-source-discovery.md#792b17c6-session-log-directory) that [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand) sweeps.

Owner: [Prompt Delivery](./prompt-delivery.md).

Supports: [793a2b1b A note delivered earlier in the session is not delivered again](../product/automatic-memory.md#793a2b1-uat-scenarios), [793a2b1c A note the agent looked up itself is not delivered](../product/automatic-memory.md#793a2b1-uat-scenarios).

### 793d4f1 Scenarios

```gherkin
Scenario: 793d4f1a ENGRAM_TRANSCRIPT_DIR names the transcripts first
  Given ENGRAM_TRANSCRIPT_DIR and CLAUDE_CONFIG_DIR are both set, and the transcript of a session in each, the first showing one note delivered and the second another
  When engram auto deliver is dispatched for a prompt of the session, without --sessions
  Then the block names the second note only

Scenario: 793d4f1b CLAUDE_CONFIG_DIR names them when ENGRAM_TRANSCRIPT_DIR does not
  Given only CLAUDE_CONFIG_DIR is set, a transcript of the session in its projects directory showing one note delivered, and a transcript of the session in the home's .claude/projects showing another
  When engram auto deliver is dispatched for a prompt of the session, without --sessions
  Then the block names the second note only

Scenario: 793d4f1c The home's .claude/projects holds them by default
  Given neither variable is set, and a transcript of the session in the home's .claude/projects showing one note delivered
  When engram auto deliver is dispatched for a prompt of the session, without --sessions
  Then the block names the other note only

Scenario: 793d4f1d No transcript is read when no directory can be found
  Given neither variable is set, and a home directory that cannot be found
  When engram auto deliver is dispatched for a prompt of a session, without --sessions
  Then no transcript is read, and the block names the notes that apply to the prompt
```

## 793d4p Telling what automatic memory did

Requirement: 793d4p The `prompt-delivery` MUST print on standard error the lines that show the user what [automatic memory](../../GLOSSARY.md#automatic-memory) did, one to a line, at most four, one for each of four kinds in the order below and none for a kind with nothing to show, and nothing else, in the formats of [855d8a Formats of the lines of 793d4p](#855d8a-formats-of-the-lines-of-793d4p): the line of what it delivered, the [notes](../../GLOSSARY.md#permanent-note) and the passages in the order of the block ([793d4a Delivering the notes that apply to a prompt](#793d4a-delivering-the-notes-that-apply-to-a-prompt)), and the lines of the records of the kinds `written`, `skipped` and `failed` that [793d4k Recording what automatic memory did](./lesson-writing.md#793d4k-recording-what-automatic-memory-did) makes for the [session](../../GLOSSARY.md#session) of the request, read from the directory its `--records` flag names, else `auto` in the engram data directory, that it has not shown, oldest first, each line beginning with the time (`15:04:05`) of the oldest of its records. It prints these lines also when it is given no reply, which is the case of a prompt that is not a [recall moment](../../GLOSSARY.md#recall-moment). A reason is written as one line and redacted ([793d4j Redacting credentials](./moment-judgement.md#793d4j-redacting-credentials)); a line is cut in its middle, so that both ends stay, when it is longer than 240 characters; every line is one line of single spaces with no control or format character. After printing the lines of the records it MUST mark each record that a line it printed tells, also one that the cut left out of the text of the line, by an empty file beside the record, named for it with the suffix `.told`, so that a line is shown once; a record with a mark makes no line, nor does a record of another session, of a kind that tells nothing, or that cannot be read, nor a directory that cannot be read, and a mark that cannot be written is the end of it. Then it MUST remove each record that has a mark, and the mark, and the directory of the session once nothing is left in it, so that the records do not pile up; what it cannot remove is the end of it. For a cue other than `prompt` it MUST print the line of what it delivered only, and read, mark and remove no record. The lines are no part of the block ([793d4c Framing the notes of a block as data](#793d4c-framing-the-notes-of-a-block-as-data)).

Owner: [Prompt Delivery](./prompt-delivery.md).

Supports: [793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt](../product/automatic-memory.md#793a2e1-uat-scenarios), [793a2e1b A write that finishes after the next prompt is shown by the prompt after it finishes](../product/automatic-memory.md#793a2e1-uat-scenarios), [793a2e1c A delivery that could not be done is shown with its reason](../product/automatic-memory.md#793a2e1-uat-scenarios), [793a2e1d A write that failed is shown with its reason](../product/automatic-memory.md#793a2e1-uat-scenarios), [793a2e1e A sweep that failed is shown with its reason](../product/automatic-memory.md#793a2e1-uat-scenarios), [793a2e1f A turn that is not captured is shown with the reason](../product/automatic-memory.md#793a2e1-uat-scenarios), [793a2e1g The notes delivered with a prompt are one line](../product/automatic-memory.md#793a2e1-uat-scenarios).

### 793d4p1 Scenarios

```gherkin
Scenario: 793d4p1a A record that was not shown becomes a line
  Given the records of a session of the kinds written, skipped with a slug, skipped without one, failed without a slug and failed with one, and none of them shown
  When the delivery is given a prompt of that session
  Then it prints on standard error one line for the written, one for the left out and one for the failed, each beginning with the time of the oldest of its records and saying what became of them

Scenario: 793d4p1b A record of a kind that tells nothing makes no line
  Given the records of a session of a kind other than written, skipped and failed, as an earlier version made them, only
  When the delivery is given a prompt of that session, and no note applies to it
  Then it prints nothing and succeeds

Scenario: 793d4p1c A line that was shown is not shown again
  Given the records of a session, and a mark beside the first
  When the delivery is given a prompt of that session, and then another
  Then the first prompt prints the line of the second record only, and the second prompt prints no line

Scenario: 793d4p1d A reason is one line, redacted and cut in its middle
  Given a failed record whose reason holds line breaks, a credential and more than 240 characters
  When the delivery is given a prompt of its session
  Then its line is one line without the credential, at most 240 characters, and holds both ends of the text

Scenario: 793d4p1e One line is shown for each kind of record
  Given the records of a session of seven written, two left out and two failed, none of them shown
  When the delivery is given a prompt of that session, and then another
  Then the first prompt prints three lines, one for each kind, the first of at most 240 characters with the first and the last of the seven file names, and the second prompt prints no line

Scenario: 793d4p1f The records of another session, a record that cannot be read and a missing directory make no line
  Given a directory of records that holds the records of another session and a file that is not a record, and a session with no directory of its own
  When the delivery is given a prompt of that session and of the other
  Then it prints what the notes make for each and succeeds, and the lines of the other session are shown to the other only

Scenario: 793d4p1g The records are read from the directory the flag names, else from the engram data directory
  Given records in the directory that --records names and others in the auto directory of the engram data directory
  When engram auto deliver is dispatched with the flag, and then without it
  Then the first prints the line of the first records and the second the line of the others

Scenario: 793d4p1h A note delivered is shown as a line
  Given a carry of a note, and a reply that delivers it
  When the delivery is given the reply
  Then it prints the block on standard output, and on standard error the line delivered the note and the file name of the note

Scenario: 793d4p1i A line shown is one line whatever the texts hold
  Given notes in a carry, a reply that delivers them, and records whose texts hold line breaks, control and format characters and lines that read as other lines, however they are mixed
  When the delivery is given the reply and a prompt of that session
  Then each line printed on standard error is one line of single spaces with no control or format character, of at most 240 characters, and there is a line for each of the four kinds that has something to tell and no other

Scenario: 793d4p1j A mark that cannot be written does not fail the command
  Given the records of a session, and a place for the marks that cannot be written
  When the delivery is given a prompt of that session
  Then it prints the lines and succeeds

Scenario: 793d4p1l The notes delivered are one line
  Given a carry of two notes, and a reply that delivers both
  When the delivery is given the reply
  Then it prints the block with a line for each note on standard output, and on standard error one line that names the two notes

Scenario: 793d4p1m The records that were shown are removed with their marks
  Given the records of a session, a record with a mark among them, and a file in the directory that is no record
  When the delivery is given a prompt of that session
  Then the records and their marks are gone, the file that is no record is still there, and the next prompt prints no line

Scenario: 793d4p1n The directory of a session whose records were shown is removed
  Given the records of a session and nothing else in its directory
  When the delivery is given a prompt of that session
  Then the directory of the session is gone, and the next prompt prints no line
```

### 855d4p1 Scenarios

```gherkin
Scenario: 855d4p1a The passages delivered are named in the line
  Given a carry of a note and two passages, and a reply that delivers all three
  When the delivery is given the reply
  Then the line holds the note's file name and both chunk IDs, as delivered the note and the passages

Scenario: 855d4p1b A delivery within a turn shows its line only and touches no record
  Given the records of a session not yet shown, a carry of a note, and a reply that delivers it
  When the delivery is given the reply with the cue before-tool
  Then it prints on standard error the line of the note only, and no record has a mark or is removed

Scenario: 855d4p1c A prompt that is not a recall moment shows the records
  Given the records of a session not yet shown, and no reply
  When the delivery is given a prompt of that session
  Then it prints the lines of the records, prints no block, and marks them
```

### 855d8a Formats of the lines of 793d4p

- Delivered: `delivered`, then for the [notes](../../GLOSSARY.md#permanent-note) `the note` and its file name, or `the notes` and the file names separated by commas, and for the passages `the passage` and its chunk ID, or `the passages` and the chunk IDs separated by commas, the two parts joined by `and`.
- Written: `wrote the note` and its file name, or `wrote the notes` and the file names separated by commas.
- Skipped: `left out:` and, for each record, `the lesson` and its slug and a colon when it has a slug, then the reason, the records separated by semicolons.
- Failed: `failed:` and, for each record, `writing` and its slug and a colon when it has a slug, then the reason, the records separated by semicolons.

## Shared requirements

- [793d4m Telling the prompts the user sends](./moment-judgement.md#793d4m-telling-the-prompts-the-user-sends): stated once, in [Moment Judgement](./moment-judgement.md); prompt-delivery is one of its owners.
