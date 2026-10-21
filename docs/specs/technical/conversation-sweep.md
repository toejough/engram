# Conversation Sweep - Technical Spec

## 793d3f Orientation

The command `engram auto sweep` (`internal/cli/auto.go`, `internal/auto`, informational). Turns a [session](../../GLOSSARY.md#session)'s [transcript](../../GLOSSARY.md#transcript) into chunks of the chunk index, owning the choice of the transcript and of the turns it acts for (the indexing is [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand)'s, which indexes a source under the lock of the manifest and leaves the index usable when it is ended: [792b18c2 Locked reconciliation](./chunk-index-store.md#792b18c2-locked-reconciliation), [792b18c3 Change detection](./chunk-index-store.md#792b18c3-change-detection)); tested through standard input and output over an injected index, and with the real `engram` for the end of a [sweep](../../GLOSSARY.md#sweep). Called by [793d2c Conversation sweep flow](./conversation-sweep-workflow.md#793d2c-conversation-sweep-flow).

## 793d4n Sweeping the conversation

Requirement: 793d4n The `conversation-sweep` MUST read one JSON object from standard input (the `session`'s ID and the `origin`), act only for the [prompts](../../GLOSSARY.md#prompt-automatic-memory) that [793d4m Telling the prompts the user sends](./moment-judgement.md#793d4m-telling-the-prompts-the-user-sends) lets the judgement act for, and for them find the [session](../../GLOSSARY.md#session)'s [transcript](../../GLOSSARY.md#transcript) as [793d4d Telling what a session holds](./session-context.md#793d4d-telling-what-a-session-holds) does in the directory of [793d4f Finding the transcripts of sessions](./prompt-delivery.md#793d4f-finding-the-transcripts-of-sessions), and index it in the chunk index as `--transcript` of [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand) does, as the one explicit source, printing nothing. It MUST print nothing, succeed and index nothing for any other origin; print nothing and fail, naming the cause, when the input is not a JSON object, the session has no transcript in the directory of transcripts or the index fails; wait for the lock that `engram ingest` takes on the manifest when another ingest holds it; and leave the chunk index usable when it is ended at any moment, so that a later [sweep](../../GLOSSARY.md#sweep) indexes the transcript whole. It MUST also print nothing, succeed and index nothing for a session whose directory in the records of [793d4k Recording what automatic memory did](./lesson-writing.md#793d4k-recording-what-automatic-memory-did) (the directory of its `--records` flag, else `auto` in the engram data directory) holds the file `sweep.ended`; and when the object it reads has `ended` set to true, the [mod](../../GLOSSARY.md#mod) having ended the sweep at its bound, MUST write that file, print nothing and fail with an error that says the sweep was ended at its time limit, that it is not run again in this session and that `engram ingest --auto` indexes the history, except that it succeeds when the file is there already, so that the user is told once. It MUST NOT create a [vault](../../GLOSSARY.md#vault) ([792b2c3 First-use gating](./command-line-surface.md#792b2c3-first-use-gating)).

Owner: [Conversation Sweep](./conversation-sweep.md).

Supports: [793a2f1a The conversation so far is findable after a turn](../product/automatic-memory.md#793a2f1-uat-scenarios), [793a2f1b A conversation too long to index in time is left alone and the user is told once what to run](../product/automatic-memory.md#793a2f1-uat-scenarios).

### 793d4n1 Scenarios

```gherkin
Scenario: 793d4n1a The transcript of the session is indexed as the one explicit source
  Given the transcript of a session in the directory of transcripts, and an index
  When the sweep is given the session
  Then the index is run for the transcript's path alone, as an explicit transcript, and nothing is printed

Scenario: 793d4n1b A session with no transcript fails the command
  Given a session that no transcript holds, and no session at all
  When the sweep is given each
  Then it prints nothing and fails each time, with an error that names the session and the directory of the transcripts, and indexes nothing

Scenario: 793d4n1c A prompt of an origin that is not swept is swept nothing
  Given the transcript of a session, and each of the origins task-notification, peer, the empty origin, and sdk without the flag --capture-sdk
  When the sweep is given the session with each, and then with sdk and the flag
  Then with each of the first it prints nothing, succeeds and indexes nothing, and with the last it indexes the transcript

Scenario: 793d4n1d A failed index fails the command, naming the cause
  Given an index that fails, and an input that is not a JSON object
  When the sweep is given the transcript of a session, and the input
  Then it prints nothing and fails each time, with an error that names the index's failure and the input

Scenario: 793d4n1e A sweep that is ended leaves the chunk index usable
  Given the transcript of a session of several turns, and the real engram
  When the sweep is ended while it works, and then runs again
  Then the manifest is not corrupt after the first, and the second indexes the whole transcript

Scenario: 793d4n1f Two sweeps at once index both transcripts
  Given the transcripts of two sessions, and the real engram
  When both sweeps run at once
  Then the manifest records both sources, and each transcript is indexed whole

Scenario: 793d4n1g A sweep that was ended gives up for the session and fails, naming what to run
  Given the transcript of a session, a place for the records of what automatic memory did, and an index
  When the sweep is given the session as ended, and then again
  Then the first writes the file sweep.ended in the directory of the session and fails with an error that says the sweep was ended, that it is not run again and that engram ingest --auto indexes the history, the second succeeds, and neither runs the index

Scenario: 793d4n1h A session that gave up is not swept
  Given the transcript of a session and the file sweep.ended in the directory of the session
  When the sweep is given the session
  Then it prints nothing, succeeds and runs no index
```

## Shared requirements

- [793d4m Telling the prompts the user sends](./moment-judgement.md#793d4m-telling-the-prompts-the-user-sends): stated once, in [Moment Judgement](./moment-judgement.md); conversation-sweep is one of its owners.
