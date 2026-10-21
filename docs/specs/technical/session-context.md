# Session Context - Technical Spec

## 793d3c Orientation

What a [session](../../GLOSSARY.md#session)'s [agent](../../GLOSSARY.md#agent-automatic-memory) already holds of the [vault](../../GLOSSARY.md#vault): the notes and passages delivered beside prompts or within turns, and what its own memory commands returned, read from the session's [transcript](../../GLOSSARY.md#transcript) (`internal/auto`, `internal/cli/auto.go`, informational). Owns the transcript's format and the rule for what counts as held (which notes to deliver is the delivery's); tested over transcripts of a session. Used by [prompt-delivery](./prompt-delivery.md), [turn-judgement](./turn-judgement.md), [lesson-placement](./lesson-placement.md) and [conversation-sweep](./conversation-sweep.md).

## 793d4d Telling what a session holds

Requirement: 793d4d The `session-context` MUST, given a [session](../../GLOSSARY.md#session)'s ID, find the session's [transcript](../../GLOSSARY.md#transcript) (the file named for the ID with the suffix `.jsonl`, in the directory of transcripts it is given ([793d4f Finding the transcripts of sessions](./prompt-delivery.md#793d4f-finding-the-transcripts-of-sessions)) or a subdirectory) and return, of what the [agent](../../GLOSSARY.md#agent-automatic-memory) was given or did, only: the file name on each [note](../../GLOSSARY.md#permanent-note) line and the chunk ID on each passage line (a line beginning with a dash and a space; the name or ID runs to the space before the parenthesis) of every block delivered beside a [prompt](../../GLOSSARY.md#prompt-automatic-memory) or within a [turn](../../GLOSSARY.md#turn-automatic-memory) (an `attachment` entry of type `hook_additional_context` beginning `[engram memory]`), and the text of each Bash command the agent ran that runs, at the position of a command ([855d2c Mid-turn recall flow](./mid-turn-recall-workflow.md#855d2c-mid-turn-recall-flow)), `engram query`, `engram show`, `engram show-chunk`, `engram recall`, [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand) or `engram amend` with its result (the `tool_result` block, of a `user` row, whose `tool_use_id` is the command's `tool_use` ID). It MUST return nothing of a block's situations and bodies, of other entries, of other tools' results, of prompts or of replies, and nothing, without an error, when the session has no transcript, the transcript cannot be read or the ID is not a plain file name.

Owner: `session-context`.

Supports: [793a2b1b A note delivered earlier in the session is not delivered again](../product/automatic-memory.md#793a2b1-uat-scenarios), [793a2b1c A note the agent looked up itself is not delivered](../product/automatic-memory.md#793a2b1-uat-scenarios), [855a2c1a A sub-point of a note the session holds continues it](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2c1b The same thought at the same level is a sibling](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2c1c A note the session has not seen is not a target](../product/automatic-memory.md#855a2c1-uat-scenarios), [855a2d1a A correcting note names the note it corrects](../product/automatic-memory.md#855a2d1-uat-scenarios).

### 793d4d1 Scenarios

```gherkin
Scenario: 793d4d1a The note lines of the blocks delivered beside prompts are what a session holds
  Given the transcript of a session with a block delivered beside a prompt, whose note lines name two notes and whose first note's body names a third, an entry of another hook beside a prompt, a prompt, a reply and other attachments
  When what the session holds is read
  Then the text holds the file names of the two notes, and nothing of the situations and bodies, the other hook's entry, the prompt, the reply or the other attachments

Scenario: 793d4d1d What the agent's own lookup printed is held
  Given the transcript of a session in which the agent ran engram query as a Bash command, whose result names a note, as a string and as text blocks
  When what the session holds is read
  Then the text holds the command and the result, and nothing of the reply

Scenario: 793d4d1e A listing of the vault is not held
  Given the transcript of a session in which the agent ran ls over the vault, whose result names a note
  When what the session holds is read
  Then the text does not hold the note's name

Scenario: 793d4d1f A note fetched with engram show is held
  Given the transcript of a session in which the agent ran engram show with the basename of a note, whose result does not name the note
  When what the session holds is read
  Then the text holds the command, which names the note

Scenario: 793d4d1b The transcript is found under its project directory
  Given a session-log directory whose subdirectory for a project holds the transcript of the session
  When what the session holds is read
  Then the text is that of the blocks of that transcript

Scenario: 793d4d1c A session with no transcript, or an ID that is not a file name, holds nothing
  Given a session-log directory with no transcript of the session, and an ID that holds a path separator
  When what the session holds is read
  Then it is empty and there is no error
```

### 855d4e1 Scenarios

```gherkin
Scenario: 855d4e1a The chunk IDs of the passages delivered are held
  Given the transcript of a session with a block delivered beside a prompt, whose passage line names a chunk ID and whose body names another
  When what the session holds is read
  Then the text holds the chunk ID of the passage line and nothing of its text and of the other ID

Scenario: 855d4e1b A passage fetched with engram show-chunk, and a note the agent saved, are held
  Given the transcript of a session in which the agent ran engram show-chunk with a chunk ID, and engram learn, whose result names the note it wrote
  When what the session holds is read
  Then the text holds both commands and the result of the second

Scenario: 855d4e1c A block delivered within a turn is held
  Given the transcript of a session with a block delivered by an event within a turn, recorded as an attachment of type hook_additional_context
  When what the session holds is read
  Then the text holds the file names of its note lines
```
