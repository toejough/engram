# Automatic Memory - Technical Spec

Filled depth first ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)), so incomplete until the last node is through step 4. The solution-space analysis, option impact and alternatives are omitted ([793b12 Departure: no option analysis](./793b_decision-log.md#793b12-departure-no-option-analysis)). A new interaction requirement (one per flow) takes the next free letter under [793d2 Recommended sequence diffs](#793d2-recommended-sequence-diffs), a behavior requirement (one per component behavior) under [793d4 Recommended component-behavior diffs](#793d4-recommended-component-behavior-diffs).

## 793d1 Orientation

The product problem, [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement): the [agent](./793a_product-spec.md#793a1a-terms) consults the [vault](../../GLOSSARY.md#vault) only when it remembers to, and the instruction to look first was refused in 6 of 6 headless trials. Today it reaches the vault only through its own commands (`engram query`, `engram show`), when a [skill](../../GLOSSARY.md#skill) or its instructions tell it to. [`engram update`](../../GLOSSARY.md#engram-update) deploys the skills of `agent-instructions/skills/` into each [harness](../../GLOSSARY.md#harness) through an engram-owned root per harness that the harness's skills directory links into ([harness-deploy-sync](../../specs/technical/harness-deploy-sync.md)); `engram query` searches the [notes](../../GLOSSARY.md#permanent-note), and the chunk index when one is configured ([query-engine](../../specs/technical/query-engine.md)). Nothing in a [session](../../GLOSSARY.md#session) acts at a [prompt](./793a_product-spec.md#793a1a-terms) without the agent's help, and `engram update` installs nothing but skills.

A **mod** is a folder Claude Code loads as a plugin of function hooks when it sits in `~/.claude/skills/<name>` (also as a link): its hooks module registers functions for session events, and `prompt.submit` runs for each prompt before it enters the session, where a function may add context that the model reads beside the prompt and the user does not see. The solution: `engram update` deploys one mod, `engram-memory`, from `agent-instructions/mods/` through the skills machinery. For each prompt the mod runs a new command, `engram auto deliver`, which prints the notes that bear on the prompt text, as `engram query` would, as one block that the mod puts beside the prompt ([793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow)). After the [turn](./793a_product-spec.md#793a1a-terms) of a prompt the user sent, the mod runs `engram auto judge`, which prints the request of one model call, makes that call with the user's own login and runs `engram auto write` with the reply, which writes each lesson as a note through the learn path ([793d2b Turn capture flow](#793d2b-turn-capture-flow)), and `engram auto sweep`, which indexes the session's record in the chunk index so that `engram query` finds the conversation ([793d2c Conversation sweep flow](#793d2c-conversation-sweep-flow)). The delivery also prints, apart from the block, the lines that show the user what [automatic memory](./793a_product-spec.md#793a1a-terms) did, from the records of what became of each reply, failure and skip, and the mod shows each as a log line of the host, which the user sees and the model does not read ([793d4p Telling what automatic memory did](#793d4p-telling-what-automatic-memory-did)). The commands, not the mod, hold the logic.

The call raises neither a prompt nor a turn, so [automatic memory](./793a_product-spec.md#793a1a-terms) starts no session of its own ([793b63 A turn is judged by one model call, and no session is started](./793b_decision-log.md#793b63-a-turn-is-judged-by-one-model-call-and-no-session-is-started)).

Verified with Claude Code 2.1.302: a mod in a linked skills folder loads in a headless `claude -p` session, and the context it adds is recorded in the session's [transcript](../../GLOSSARY.md#transcript) as a `hook_additional_context` record before the agent's first reply; `turn.complete` carries the turn's final text and ID and fires when the record already holds the turn's last row; `$.model.complete` answers `haiku` within about two seconds, also from work left running after the hook that started it returned. `$.ui.log`, called in a hook or from a timer, appends one line to the conversation as the user sees it, "drawn like a system notice (dim; not sent to the model)" in the host's type file, and to no record of the session; a host that streams JSON receives it as a `system` event of the subtype `ui_log` with the mod's name, and plain text output of `claude -p` shows none.

## 793d2 Recommended sequence diffs

### 793d2a Prompt delivery flow

Added interactions are solid; the retained one is in a grey block.

```mermaid
sequenceDiagram
    actor User
    participant Session as Claude Code session
    participant Mod as automatic-memory-mod
    participant Delivery as prompt-delivery
    participant Engine as query-engine
    participant Context as session-context
    participant Records as the record of what it did

    User->>Session: sends a prompt
    Session->>Mod: 793d2a prompt.submit with the prompt text
    Mod->>Delivery: 793d2a, 793d4a engram auto deliver, the text, the session's ID and the origin on standard input
    rect rgb(235, 235, 235)
        Delivery->>Engine: 792b24c1, 792b24c5, 792b24c20 the notes for the text
        Engine-->>Delivery: 792b24c5 the notes that match
    end
    Delivery->>Context: 793d4d the text the session holds
    Context-->>Delivery: 793d4d the text from the session's record
    Delivery->>Records: 793d4p the records of the session not yet shown
    Records-->>Delivery: 793d4p what automatic memory did
    Delivery-->>Mod: 793d4a, 793d4p the block on standard output, the lines for the user on standard error
    Delivery->>Records: 793d4p a mark beside each record shown
    Mod-->>Session: 793d2a the prompt unchanged, the block as context
    Mod-->>User: 793d2a each line for the user, as a log line of the host
    Mod->>Records: 793d2a, 793d4k a delivery that failed or was ended, as a failure, from a timer
    Session-->>User: the agent acts with the notes in its context
```

Requirement: 793d2a For each [prompt](./793a_product-spec.md#793a1a-terms) a Claude Code [session](../../GLOSSARY.md#session) raises, the `automatic-memory-mod` MUST run `engram auto deliver` with the prompt text, the session's ID and the prompt's origin (the kind Claude Code stamps on it: `composer`, `sdk`, `task-notification` and the rest) as one JSON object on standard input; MUST add the command's standard output to the prompt's context as one block when the command exited with 0 and the output begins with `[engram memory]`, and nothing otherwise; MUST show the user each line the command printed on standard error when it exited with 0, as one log line of the host each (`$.ui.log`), which the model does not read, and add none of them to the prompt's context; and MUST leave the prompt text as received. It MUST hold the prompt for its own work, the session-ID call and the wait for the command together, at most 7 seconds, so the prompt is held under the 8 seconds of [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), and when the work does not finish within them MUST end the command and let the prompt continue as it was, with nothing added. When the command fails (it exits with other than 0 or cannot be run) or the work does not finish within the bound, and the session's ID is known, it MUST also run `engram auto write` with the session's ID, no [turn](./793a_product-spec.md#793a1a-terms) and a `failure`, a sentence that names the delivery and how it failed, as [793d2b Turn capture flow](#793d2b-turn-capture-flow) does for a step that fails, and MUST start that run from a timer (`$.clock.after`), so that the prompt is held no longer and no call of the host is in flight when the hook settles; a command that prints nothing, or text that is not a block, has not failed, and no failure is reported for it.

Owner: `automatic-memory-mod` (new; `automatic-memory-mod.md` at promotion). Supports: [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2d1a A slow lookup does not hold a prompt for more than 8 seconds](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2d1d The prompt and the reply are as they were](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2d1e A write that fails does not break the session](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1b A write that finishes after the next prompt is shown when it finishes](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c A delivery that could not be done is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1d A write that failed is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1e A sweep that failed is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1f A turn that is not captured is shown with the reason](./793a_product-spec.md#793a2e1-uat-scenarios).

#### 793d2a1 Scenarios

```gherkin
Scenario: 793d2a1a A block is added beside the prompt
  Given a session with the mod installed, and an engram whose auto deliver prints a block
  When a prompt is sent
  Then the command received the prompt text, and the session holds the prompt exactly as sent and then the block as context before the agent's first reply

Scenario: 793d2a1b The command receives the session's ID
  Given a session with the mod installed, and an engram whose auto deliver prints a block
  When a prompt is sent
  Then the command received the ID of the session the prompt was sent in

Scenario: 793d2a1c An empty output adds nothing
  Given a session with the mod installed, and an engram whose auto deliver prints nothing
  When a prompt is sent
  Then the session holds the prompt exactly as sent and no context beside it, and no failure was reported

Scenario: 793d2a1d A slow command is ended and the prompt continues
  Given a session with the mod installed, and an engram whose auto deliver takes 20 seconds
  When a prompt is sent
  Then the model is asked within 8 seconds of the prompt, with the prompt exactly as sent and no context beside it, and the mod's hook did not fail

Scenario: 793d2a1e A failed command adds nothing
  Given a session with the mod installed, and an engram whose auto deliver prints a block and exits with 1
  When a prompt is sent
  Then the session holds the prompt exactly as sent and no context beside it, and the mod's hook did not fail

Scenario: 793d2a1f Output that is not a block adds nothing
  Given a session with the mod installed, and an engram whose auto deliver prints text that does not begin with [engram memory] and exits with 0
  When a prompt is sent
  Then the session holds the prompt exactly as sent and no context beside it, the mod's hook did not fail, and no failure was reported

Scenario: 793d2a1g A session that does not give its ID in time does not hold the prompt
  Given the mod, and a session that never gives its ID
  When a prompt is sent
  Then the mod lets the prompt continue as it was after the 7 seconds, with nothing added, and the command was never run

Scenario: 793d2a1h The command receives the prompt's origin
  Given a session with the mod installed, and an engram whose auto deliver prints a block
  When a prompt is sent from a headless session, with claude -p
  Then the command received the origin sdk

Scenario: 793d2a1i A command that fails is reported to the writing
  Given the mod, a session whose ID is known, and an engram whose auto deliver exits with 1 and prints a cause on standard error
  When a prompt is submitted
  Then the prompt goes on as it was with nothing added, no call of the host is in flight when the hook settles, and, when the timers run, auto write received the session's ID, no turn and a failure that names the delivery and the cause, and the user was shown no line

Scenario: 793d2a1j A delivery that does not finish within its bound is reported to the writing
  Given the mod, a session whose ID is known, and an engram whose auto deliver runs until its time limit
  When a prompt is submitted
  Then the prompt goes on as it was with nothing added, auto deliver was given the time limit of 7 seconds, and, when the timers run, auto write received the session's ID, no turn and a failure that names the delivery and the 7 seconds, and the user was shown no line

Scenario: 793d2a1k The lines the command prints for the user are shown through the host's log and are not added to the context
  Given the mod, a session whose ID is known, and an engram whose auto deliver prints a block and two lines on standard error and exits with 0
  When a prompt is submitted
  Then the host was asked to log the two lines, one call each, in order, and the prompt goes on with the block as context, which holds neither line
```

### 793d2b Turn capture flow

Added interactions are solid.

```mermaid
sequenceDiagram
    actor User
    participant Session as Claude Code session
    participant Mod as automatic-memory-mod
    participant Judgement as turn-judgement
    participant Model as the host's model call
    participant Writing as lesson-writing
    participant Vault as the vault
    participant Record as the record of what it did

    User->>Session: the turn ends
    Session->>Mod: 793d2b turn.complete with the turn's final text
    Mod-->>Session: 793d2b the event's result, at once
    Mod->>Judgement: 793d2b, 793d4g engram auto judge, the session's ID, the turn's ID, the final text and the origin on standard input
    Judgement-->>Mod: 793d4g the request for the model
    Mod->>Model: 793d2b the request
    Model-->>Mod: 793d2b the reply
    Mod->>Writing: 793d2b, 793d4h engram auto write, the session's ID, the turn's ID and the reply on standard input
    Writing->>Vault: 793d4h a note for each lesson
    Writing->>Record: 793d4k what became of the reply
```

Requirement: 793d2b For each [turn](./793a_product-spec.md#793a1a-terms) a [session](../../GLOSSARY.md#session) completes, the `automatic-memory-mod` MUST give the event's result back at once, with no call of the host in flight, and then, from a timer (`$.clock.after`), run `engram auto judge` with one JSON object on standard input: the session's ID, the turn's ID, the turn's final text, the text the turn began with and the origin of the [prompt](./793a_product-spec.md#793a1a-terms) it began with. It MUST do nothing for a turn that a [subagent](../../GLOSSARY.md#subagent) ran, which the event's `agentId` says. When the command exits with 0 and prints a request, the mod MUST ask the model for it through the host's model call, exactly as printed, and when the model answers run `engram auto write` with the session's ID, the turn's ID and the reply; when it exits with 0 and prints nothing, nothing more. When a step fails (the command exits with other than 0 or prints what is not a request, a request that is not a JSON object being a failure of `engram auto judge`; the model does not answer; a call is ended or rejected) it MUST run `engram auto write` with the session's ID, the turn's ID and a `failure`, a sentence that names the step and how it failed, in place of the reply, whole, for the writing cuts it ([793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did)). It MUST bound each stage so that together they take under the 2 minutes of [793a2c Lessons are written without the agent](./793a_product-spec.md#793a2c-lessons-are-written-without-the-agent): the session-ID call 5 seconds (then it does nothing), `engram auto judge` 15, the model call the 60 of the request, `engram auto write` 20 and its failure report 15, 115 in all; a stage that overruns has failed. The mod keeps the text and origin of each prompt the session raises: a turn that starts with a text takes the oldest kept prompt with that text and drops the prompts kept before it, which began no turn; a turn whose text no kept prompt has takes the origin of the newest; a turn that starts with no text has no origin.

Owner: `automatic-memory-mod`. Supports: [793a2c1a A correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1b A request to remember becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1c An approach the user confirmed becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1d The agent's own correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1e An approach that succeeded with evidence becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1f Guesses, plans and chatter produce no note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1g A credential in the turn is not in the note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1h A thanks that names no approach produces no note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1i A request to remember is written when the agent says it did not save it](./793a_product-spec.md#793a2c1-uat-scenarios), [793a3c1a A lesson an earlier note states is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), [793a3c1b A lesson the agent saved itself is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), [793a3c1c A lesson no earlier note states is written](./793a_product-spec.md#793a3c1-uat-scenarios), [793a2d1c The work after a turn does not hold the next prompt](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2d1d The prompt and the reply are as they were](./793a_product-spec.md#793a2d1-uat-scenarios), [793a2d1e A write that fails does not break the session](./793a_product-spec.md#793a2d1-uat-scenarios).

#### 793d2b1 Scenarios

```gherkin
Scenario: 793d2b1a The mod judges a turn and gives the reply to the writing
  Given the mod, a session whose ID is known, an engram whose auto judge prints a request, and a model that answers
  When a prompt is raised, the turn starts, and it completes with a final text
  Then the event's result is given back at once, auto judge received the session's ID, the turn's ID, the final text and the origin of the prompt, the model was asked the request exactly as printed, and auto write received the session's ID, the turn's ID and the model's reply

Scenario: 793d2b1b A judgement that prints nothing ends the capture
  Given the mod, and an engram whose auto judge exits with 0 and prints nothing
  When a turn completes
  Then the model was never asked, and auto write was never run

Scenario: 793d2b1c A step that fails is reported to the writing
  Given the mod, and an engram whose auto judge exits with 1, prints text that is not a request, or is ended before it answers
  When a turn completes
  Then the model was never asked, and auto write received the session's ID, the turn's ID and a failure that names the judgement

Scenario: 793d2b1d A model that does not answer is reported to the writing
  Given the mod, an engram whose auto judge prints a request, and a model that answers with an API error, with no text, or by rejecting the call
  When a turn completes
  Then auto write received the session's ID, the turn's ID and a failure that names the model call

Scenario: 793d2b1e The judgement receives the origin of the prompt the turn began with
  Given the mod, and an engram whose auto judge prints a request
  When a prompt of the origin task-notification is raised, a turn starts, and it completes
  Then auto judge received the origin task-notification, beside the session's ID, the turn's ID and the final text

Scenario: 793d2b1f A turn of a subagent is not captured
  Given the mod, and an engram whose auto judge prints a request
  When a turn that a subagent ran completes
  Then auto judge was never run, and the model was never asked

Scenario: 793d2b1g The judgement receives the text the turn began with
  Given the mod, and an engram whose auto judge prints a request
  When a prompt is raised, the turn starts with its text, and it completes
  Then auto judge received the text, beside the session's ID, the turn's ID, the final text and the origin

Scenario: 793d2b1h A turn takes the origin and the text of its own prompt when others are waiting
  Given the mod, and an engram whose auto judge prints a request
  When a prompt of the user starts a turn, a second prompt of the user is raised while it runs, a task notification is raised after the second, the first turn completes, and the second prompt starts a turn that completes
  Then auto judge received for the second turn the origin and the text of the second prompt, and not those of the notification

Scenario: 793d2b1i A turn that starts with no text has no origin
  Given the mod, and an engram whose auto judge prints a request
  When a prompt of the user starts a turn that completes, and a turn then starts with no text and completes
  Then auto judge received no origin for the second turn

Scenario: 793d2b1j A stage that stalls is ended at its bound and reported
  Given the mod, and a host in which, in turn, the call for the session's ID never answers, auto judge runs until its time limit, the model's call is cut at its time limit, and auto write runs until its time limit
  When a turn completes
  Then the mod waited for the ID 5 seconds and then did nothing, auto judge was given the time limit of 15 seconds, the model's request held 60 seconds, auto write was given 20 seconds, and the report of the failure to auto write was given 15 seconds, and those limits together are less than 120 seconds

Scenario: 793d2b1k A request that is not JSON is a failure of the judgement
  Given the mod, and an engram whose auto judge prints text that begins as a request and is not a JSON object
  When a turn completes
  Then the model was never asked, and auto write received a failure that names the judgement

Scenario: 793d2b1l The work after a turn is started from a timer, with no call in flight when the event settles
  Given the mod, a host that counts the calls it has in flight when the hook of an event settles and runs the mod's timers only when the driver lets them, and an engram whose auto judge and auto sweep do not answer
  When a turn completes
  Then no call of the host was made or in flight when the event's hook settled, and auto judge and auto sweep were run when the timers were let go

Scenario: 793d2b1m A long failure is reported whole
  Given the mod, and an engram whose auto judge exits with 1 and prints on standard error a line of 1000 characters that begins by naming the step and ends with a cause
  When a turn completes
  Then auto write received a failure that holds the whole line, from the name of the judgement and the status to the cause
```

### 793d2c Conversation sweep flow

Added interactions are solid.

```mermaid
sequenceDiagram
    actor User
    participant Session as Claude Code session
    participant Mod as automatic-memory-mod
    participant Sweep as conversation-sweep
    participant Index as chunk-index-store
    participant Writing as lesson-writing
    participant Record as the record of what it did

    User->>Session: the turn ends
    Session->>Mod: 793d2c turn.complete
    Mod-->>Session: 793d2c the event's result, at once
    Mod->>Sweep: 793d2c, 793d4n engram auto sweep, the session's ID and the origin on standard input
    Sweep->>Index: 793d4n the session's record, as engram ingest --transcript indexes it
    alt the sweep fails or is ended
        Mod->>Writing: 793d2c, 793d4h engram auto write, the session's ID, the turn's ID and a failure
        Writing->>Record: 793d4k what became of the failure
    end
```

Requirement: 793d2c For each [turn](./793a_product-spec.md#793a1a-terms) a [session](../../GLOSSARY.md#session) completes, the `automatic-memory-mod` MUST give the event's result back at once and then, from a timer and apart from the capture of [793d2b Turn capture flow](#793d2b-turn-capture-flow), run `engram auto sweep` with one JSON object on standard input: the session's ID and the origin of the [prompt](./793a_product-spec.md#793a1a-terms) the turn began with. It MUST do nothing for a turn that a [subagent](../../GLOSSARY.md#subagent) ran, and for a session whose ID does not come within 5 seconds. It MUST end the command after 90 seconds, so that the call for the ID (5), the sweep (90) and the report of its failure (15) take 110 seconds, under the 2 minutes of [793a2f The conversation becomes searchable memory](./793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory); the bound of the sweep neither extends nor waits for those of the capture. When the command exits with other than 0, or is ended or cannot run, it MUST run `engram auto write` with the session's ID, the turn's ID and a `failure`, a sentence that names the sweep and how it failed.

Owner: `automatic-memory-mod`. Supports: [793a2f1a The conversation so far is findable after a turn](./793a_product-spec.md#793a2f1-uat-scenarios), [793a2d1c The work after a turn does not hold the next prompt](./793a_product-spec.md#793a2d1-uat-scenarios).

#### 793d2c1 Scenarios

```gherkin
Scenario: 793d2c1a The mod sweeps the conversation after a turn, apart from the capture
  Given the mod, a session whose ID is known, an engram whose auto judge does not answer, and an engram whose auto sweep exits with 0
  When a prompt is raised, the turn starts, and it completes
  Then the event's result is given back at once, and auto sweep received the session's ID and the origin of the prompt while the judgement was still running

Scenario: 793d2c1b A sweep that fails is reported to the writing
  Given the mod, and an engram whose auto sweep exits with 1, or is ended before it answers
  When a turn completes
  Then auto write received the session's ID, the turn's ID and a failure that names the sweep

Scenario: 793d2c1c A sweep that stalls is ended at its bound and reported
  Given the mod, and a host in which, in turn, the call for the session's ID never answers, and auto sweep runs until its time limit
  When a turn completes
  Then with the first auto sweep was never run after the mod waited 5 seconds, and with the second auto sweep was given the time limit of 90 seconds and the report of its failure 15, which with the 5 seconds for the ID are less than 120 seconds

Scenario: 793d2c1d A turn of a subagent is not swept
  Given the mod
  When a turn that a subagent ran completes
  Then auto sweep was never run
```

## 793d3 Recommended component diffs

- **automatic-memory-mod** (added): the mod `engram-memory`, a hooks module in `agent-instructions/mods/engram-memory/` (informational). Owns the routing and the placement of the block (the lookup is the command's), the showing of the lines for the user and the routing of the judgement, of the write and of the sweep after a [turn](./793a_product-spec.md#793a1a-terms) ([793d2b Turn capture flow](#793d2b-turn-capture-flow), [793d2c Conversation sweep flow](#793d2c-conversation-sweep-flow)); tested in a real Claude Code [session](../../GLOSSARY.md#session) against an `engram` that prints a chosen block.
- **prompt-delivery** (added): the command `engram auto deliver` (`internal/cli/auto.go`, `internal/auto`, informational). Turns a [prompt](./793a_product-spec.md#793a1a-terms) into the block of [notes](../../GLOSSARY.md#permanent-note) that bear on it and the lines that show the user what [automatic memory](./793a_product-spec.md#793a1a-terms) did, owning the choice and rendering of both and the marks of the records shown (the hook is the mod's); tested through standard input and output over an in-memory [vault](../../GLOSSARY.md#vault) and records.
- **session-context** (added): what a session's [agent](./793a_product-spec.md#793a1a-terms) already holds of the vault, read from the session's record (`internal/auto`, `internal/cli/auto.go`, informational), owning the record's format, the rule for what counts as held (which notes to deliver is the delivery's); tested over records of a session.
- **turn-judgement** (added): the command `engram auto judge` (`internal/cli/auto.go`, `internal/auto`, informational). Turns a turn into the request of the call that judges it, redacting credentials first ([793d4j Redacting credentials](#793d4j-redacting-credentials)), owning the choice of the turn's lines and the instructions (the call is the mod's); tested through standard input and output over records of a session.
- **lesson-writing** (added): the command `engram auto write` (`internal/cli/auto.go`, `internal/auto`, informational). Turns the model's reply into notes of the vault, owning the reading of the reply and the mark of a note (the write itself is the learn path's), and keeps the record of what became of each reply ([793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did)); tested through standard input and output over an in-memory vault.
- **conversation-sweep** (added): the command `engram auto sweep` (`internal/cli/auto.go`, `internal/auto`, informational). Turns a session's record into chunks of the chunk index, owning the choice of the record and of the turns it acts for (the indexing is [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand)'s); tested through standard input and output over an injected index, and with the real `engram` for the end of a sweep.
- **chunk-index-store** (retained): `engram ingest`, which indexes a source under the lock of the manifest and leaves the index usable when it is ended ([792b18c2 Locked reconciliation](../../specs/technical/chunk-index-store.md#792b18c2-locked-reconciliation), [792b18c3 Change detection](../../specs/technical/chunk-index-store.md#792b18c3-change-detection)).
- **learn path** (retained): the write of a note, as [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand) does it (`RunLearn`).
- **harness-deploy-sync** (updated): plans, links, reports and removes the mod as it does a [skill](../../GLOSSARY.md#skill) directory, and reports it as a mod ([793b47 A dry run reports the mod as a mod](./793b_decision-log.md#793b47-a-dry-run-reports-the-mod-as-a-mod)).
- **query-engine** (retained): the lookup of notes for text, used by the delivery and the judgement as `engram query` uses it.
- **Claude Code's mod runtime** (external boundary, retained): loads a mod from `~/.claude/skills/<name>`, runs its `prompt.submit`, `turn.start` and `turn.complete` functions, runs the commands it asks for, makes the model calls it asks for with the user's own login, and carries the context it returns to the model. Relied on, not tested here.

## 793d4 Recommended component-behavior diffs

### 793d4a Delivering the notes that bear on a prompt

Requirement: 793d4a The `prompt-delivery` MUST read one JSON object (`text`, the `session`'s ID, `origin`) from standard input and print on standard output one block, or nothing when the origin is not one the user sends ([793d4m Telling the prompts the user sends](#793d4m-telling-the-prompts-the-user-sends)). The block opens with a line beginning `[engram memory]` and has one line per [note](../../GLOSSARY.md#permanent-note) that bears on the text, in the lookup's order: a dash, the file name, the situation in parentheses, a colon and the body on one line. A note bears on the text when the lookup of [792b24c5 The matched set](../../specs/technical/query-engine.md#792b24c5-the-matched-set) for the text, as its message and only phrase, returns it as an item that is not a chunk and was not picked by the explore half alone ([792b24c16 Delivering the explore half](../../specs/technical/query-engine.md#792b24c16-delivering-the-explore-half): an explore pick is sampled from the vocabulary, not matched to the text); the lookup's default limit ([792b24c1 Input and defaults](../../specs/technical/query-engine.md#792b24c1-input-and-defaults)) bounds the lines ([793b19 Volume of delivered notes](./793b_decision-log.md#793b19-volume-of-delivered-notes)). It MUST leave out each note whose basename (file name without `.md`) appears in the text that [793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds) returns for the request's [session](../../GLOSSARY.md#session), print no block and succeed when no note remains, never search the chunk index, and print nothing and fail, naming the cause, when the lookup fails; the cause holds no text of the [prompt](./793a_product-spec.md#793a1a-terms) ([793b107 A delivery that could not be done is reported without the prompt's text](./793b_decision-log.md#793b107-a-delivery-that-could-not-be-done-is-reported-without-the-prompts-text)).

Owner: `prompt-delivery` (new; `prompt-delivery.md` at promotion). Supports: [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2b1a The notes that bear on a prompt arrive with it](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2b1d A prompt that matches no note arrives with nothing](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt](./793a_product-spec.md#793a2e1-uat-scenarios).

#### 793d4a1 Scenarios

```gherkin
Scenario: 793d4a1a The notes that bear on the text come back as one block
  Given a vault holding a note that says when the staging database is reset
  When the command is given a prompt asking when the staging database is reset
  Then it prints a block that opens with a line beginning [engram memory] and has one line naming the note, its situation and its body

Scenario: 793d4a1b A failed lookup fails the command and prints nothing
  Given a lookup that fails
  When the command is given a prompt
  Then it fails with an error that names the lookup's failure and holds no text of the prompt, and it prints nothing

Scenario: 793d4a1c A note that only the explore half picked does not bear on the prompt
  Given a lookup whose payload holds a matched note and a note picked by the explore half alone
  When the command is given a prompt
  Then the block names the matched note and not the explore pick

Scenario: 793d4a1d A note the session holds is left out
  Given a lookup that returns two notes, and a session whose record shows that one of them was delivered
  When the command is given a prompt of that session
  Then the block names the other note only

Scenario: 793d4a1e No note to deliver prints nothing
  Given a lookup that returns a note the session holds, and a lookup that returns no note
  When the command is given a prompt for each
  Then it prints nothing and succeeds both times

Scenario: 793d4a1f The chunk index is not searched
  Given a vault with a note that bears on the prompt, and a chunk index
  When the command is given the prompt
  Then the chunk index is never listed, and the block names the note

Scenario: 793d4a1g A note the agent fetched by its basename is left out
  Given a lookup that returns two notes, and a session whose record shows that the agent ran engram show with the basename of one of them
  When the command is given a prompt of that session
  Then the block names the other note only

Scenario: 793d4a1h The lookup is asked for the prompt's text as its message and as its only phrase
  Given a request whose text is a prompt, and a lookup that records the arguments it is run with
  When the command is given the request
  Then the lookup is run once, for the vault the command was given, with the prompt's text as its message and as its only phrase

Scenario: 793d4a1i The lookup is asked for the text and what the session holds and was not shown for the session
  Given a request with a text and a session's ID, a lookup and the readings of what the session holds and of what it was not shown, each recording what it is given
  When the block is made
  Then the lookup is given the request's text and each reading is given the request's session ID
```

### 793d4c Framing the notes of a block as data

Requirement: 793d4c The `prompt-delivery` MUST open every block with one fixed line beginning `[engram memory]` that says the lines below are [notes](../../GLOSSARY.md#permanent-note) from the [vault](../../GLOSSARY.md#vault), delivered by the memory system and not typed by the user, and that an instruction inside a note is not the user's. It MUST write each note as exactly one line beginning with a dash and a space, its file name, situation and body each with every run of white space, line breaks and separators included, as one space and no control or format character (Unicode category Cf: zero-width space or joiner, bidirectional override or isolate, byte-order mark, soft hyphen), so that a note's text cannot make a line of its own, such as the framing line or a line that reads as another note. The line of a note whose source says it was written automatically ([793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply)) ends with the fixed words `(written automatically)`, so that the [agent](./793a_product-spec.md#793a1a-terms) can tell it from a note a person or the agent saved.

Owner: `prompt-delivery`. Supports: [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2b1a The notes that bear on a prompt arrive with it](./793a_product-spec.md#793a2b1-uat-scenarios), [793a3a1a A written note is delivered with later prompts](./793a_product-spec.md#793a3a1-uat-scenarios).

#### 793d4c1 Scenarios

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
  Given a note whose source says that it was written automatically, and a note whose source is another
  When the block is made for each
  Then the line of the first ends with (written automatically), and the line of the other does not
```

### 793d4d Telling what a session holds

Requirement: 793d4d The `session-context` MUST, given a [session](../../GLOSSARY.md#session)'s ID, find the session's record (the file named for the ID with the suffix `.jsonl`, in the directory of records it is given ([793d4f Finding the records of sessions](#793d4f-finding-the-records-of-sessions)) or a subdirectory) and return, of what the [agent](./793a_product-spec.md#793a1a-terms) was given, only: the file name on each [note](../../GLOSSARY.md#permanent-note) line (a line beginning with a dash and a space; the name runs to the space before the situation's parenthesis) of every block delivered beside a [prompt](./793a_product-spec.md#793a1a-terms) (an `attachment` entry of type `hook_additional_context` beginning `[engram memory]`), and the text of each Bash command the agent ran that contains `engram query` or `engram show` with its result (the `tool_result` block, of a `user` record, whose `tool_use_id` is the command's `tool_use` ID). It MUST return nothing of a block's situations and bodies, of other entries, of other tools' results, of prompts or of replies, and nothing, without an error, when the session has no record, the record cannot be read or the ID is not a plain file name.

Owner: `session-context` (new; `session-context.md` at promotion). Supports: [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios).

#### 793d4d1 Scenarios

```gherkin
Scenario: 793d4d1a The note lines of the blocks delivered beside prompts are what a session holds
  Given the record of a session with a block delivered beside a prompt, whose note lines name two notes and whose first note's body names a third, an entry of another hook beside a prompt, a prompt, a reply and other attachments
  When what the session holds is read
  Then the text holds the file names of the two notes, and nothing of the situations and bodies, the other hook's entry, the prompt, the reply or the other attachments

Scenario: 793d4d1d What the agent's own lookup printed is held
  Given the record of a session in which the agent ran engram query as a Bash command, whose result names a note, as a string and as text blocks
  When what the session holds is read
  Then the text holds the command and the result, and nothing of the reply

Scenario: 793d4d1e A listing of the vault is not held
  Given the record of a session in which the agent ran ls over the vault, whose result names a note
  When what the session holds is read
  Then the text does not hold the note's name

Scenario: 793d4d1f A note fetched with engram show is held
  Given the record of a session in which the agent ran engram show with the basename of a note, whose result does not name the note
  When what the session holds is read
  Then the text holds the command, which names the note

Scenario: 793d4d1b The record is found under its project directory
  Given a session-log directory whose subdirectory for a project holds the record of the session
  When what the session holds is read
  Then the text is that of the blocks of that record

Scenario: 793d4d1c A session with no record, or an ID that is not a file name, holds nothing
  Given a session-log directory with no record of the session, and an ID that holds a path separator
  When what the session holds is read
  Then it is empty and there is no error
```

### 793d4e Modified: First-use gating

Target: [792b2c3 First-use gating](../../specs/technical/command-line-surface.md#792b2c3-first-use-gating)

Requirement: 792b2c3 For every [vault](../../GLOSSARY.md#vault)-resolving command other than the commands of the group `auto` (`auto deliver`, `auto judge`, `auto write` and `auto sweep`), the command-line-surface MUST ask the [vault-identity-registry](../../specs/technical/vault-identity-registry.md) to ensure the vault at the resolved path before running the command, and MUST NOT run the command when that step fails. For a command of the group `auto` it MUST NOT ask: the command runs against the path as it is, and creates nothing when the path holds no vault.

Owner: [command-line-surface](../../specs/technical/command-line-surface.md)

Supports: [792a2e1a A read command creates the vault](../../specs/product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1b amend and activate fail only on their own merits](../../specs/product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1c A child host gets its own vault](../../specs/product/vault-identity.md#792a2e1-uat-scenarios), [792a2e1d A second run creates nothing](../../specs/product/vault-identity.md#792a2e1-uat-scenarios), [792a2f1a An existing vault is untouched by read-only use](../../specs/product/vault-identity.md#792a2f1-uat-scenarios), [792a2g1a A half-created vault gets its ID](../../specs/product/vault-identity.md#792a2g1-uat-scenarios), [792a6t1a Stats on an empty vault](../../specs/product/controlled-vocabulary.md#792a6t1-uat-scenarios), [793a2a1e A vault that does not exist is not created](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2a1g A lesson is not written to a vault that does not exist](./793a_product-spec.md#793a2a1-uat-scenarios).

Rationale: one shared first-use step in dispatch, rather than one per command, makes every command create a missing vault the same way; the commands of the group `auto` are run by the mod, not by the user, and a missing vault is the user's to create.

#### 793d4e1 Scenarios

```gherkin
Scenario: 793d4e1a The delivery command creates no vault
  Given no vault exists at the resolved path
  When engram auto deliver is dispatched
  Then no vault exists at the path afterwards, nothing is printed to standard output and no creation line is printed

Scenario: 793d4e1b The judgement command creates no vault
  Given no vault exists at the resolved path, and a session whose record holds a prompt
  When engram auto judge is dispatched
  Then no vault exists at the path afterwards and no creation line is printed

Scenario: 793d4e1c The writing command creates no vault and writes no note
  Given no vault exists at the resolved path, and a reply that holds one lesson
  When engram auto write is dispatched
  Then no vault exists at the path afterwards, no note was written, nothing is printed to standard output and no creation line is printed

Scenario: 793d4e1d The sweep command creates no vault
  Given no vault exists at the resolved path, and the record of a session
  When engram auto sweep is dispatched
  Then no vault exists at the path afterwards, nothing is printed to standard output and no creation line is printed
```

### 793d4b Deploying the mods of the source

Requirement: 793d4b The `harness-deploy-sync` MUST plan one copy for every file under `agent-instructions/mods/` of the source for Claude Code, keeping the file's relative path and naming its top-level directory as the [skill](../../GLOSSARY.md#skill) directory it is deployed in, so that a mod is deployed, linked, reported and removed as a skill directory is; plan none for Pi, and none without an error when the directory does not exist; and report each mod directory it deploys, in the Claude Code section of the report of [`engram update`](../../GLOSSARY.md#engram-update), as a mod, with `agent-instructions/mods/<name>/` as its source (a skill has `agent-instructions/skills/<name>/`). A dry run MUST plan and report the mod and write none of its files; a second `engram update` MUST leave the deployed mod as the first left it (the same files and link, no second copy, listed once).

Owner: [harness-deploy-sync](../../specs/technical/harness-deploy-sync.md). Supports: [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2a1b A second update still delivers each note once](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2a1c A dry run turns nothing on](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2a1d Pi gets no automatic memory](./793a_product-spec.md#793a2a1-uat-scenarios).

#### 793d4b1 Scenarios

```gherkin
Scenario: 793d4b1a Every file of a mod is planned for Claude Code
  Given a source whose agent-instructions/mods holds the mod engram-memory with three files
  When the mod copies are planned for Claude Code
  Then each file is planned once under Claude Code's skills directory, in the engram-memory directory

Scenario: 793d4b1b A source with no mods plans none
  Given a source with no agent-instructions/mods directory
  When the mod copies are planned
  Then none is planned and there is no error

Scenario: 793d4b1c An update deploys and links the mod
  Given a checkout whose agent-instructions/mods holds the mod engram-memory, and a machine with Claude Code
  When engram update runs
  Then the engram-owned root holds the mod's files, and Claude Code's skills directory links engram-memory to them

Scenario: 793d4b1d Pi is planned no mod
  Given a source whose agent-instructions/mods holds the mod engram-memory, and the harnesses Claude Code and Pi
  When the mod copies are planned
  Then they are planned for Claude Code only

Scenario: 793d4b1e A deployed mod is reported as a mod
  Given a checkout whose agent-instructions/mods holds the mod engram-memory and whose agent-instructions/skills holds the skill learn, and a machine with Claude Code
  When engram update runs
  Then the report of Claude Code lists learn as a skill directory and engram-memory as a mod directory

Scenario: 793d4b1f The report names the mods directory as the source of a mod
  Given a report of Claude Code with the skill directory learn and the mod directory engram-memory
  When the report is written
  Then the skill is listed from agent-instructions/skills/learn/ and the mod from agent-instructions/mods/engram-memory/

Scenario: 793d4b1g A dry run plans the mod and writes none of it
  Given a checkout whose agent-instructions/mods holds the mod engram-memory, and a machine with Claude Code
  When engram update --dry-run runs
  Then the report lists the mod, and no file of it and no link to it has been written

Scenario: 793d4b1h A second update leaves the mod as the first left it
  Given a checkout whose agent-instructions/mods holds the mod engram-memory, a machine with Claude Code, and engram update run once
  When engram update runs again
  Then the files of the mod and the link to it are those the first run left, and the report lists the mod once
```

### 793d4f Finding the records of sessions

Requirement: 793d4f The `prompt-delivery` MUST read the records of [sessions](../../GLOSSARY.md#session) from the directory its `--sessions` flag names or, without the flag, from the directory `ENGRAM_TRANSCRIPT_DIR` names when set, else the `projects` directory of the directory `CLAUDE_CONFIG_DIR` names when set, else `<home>/.claude/projects`, and from no directory when the home directory cannot be found. It MUST NOT change the session-log directory of [792b17c6 Session-log directory](../../specs/technical/ingestion-source-discovery.md#792b17c6-session-log-directory) that [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand) sweeps.

Owner: `prompt-delivery`. Supports: [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios).

#### 793d4f1 Scenarios

```gherkin
Scenario: 793d4f1a ENGRAM_TRANSCRIPT_DIR names the records first
  Given ENGRAM_TRANSCRIPT_DIR and CLAUDE_CONFIG_DIR are both set, and the record of a session in each, the first showing one note delivered and the second another
  When engram auto deliver is dispatched for a prompt of the session, without --sessions
  Then the block names the second note only

Scenario: 793d4f1b CLAUDE_CONFIG_DIR names them when ENGRAM_TRANSCRIPT_DIR does not
  Given only CLAUDE_CONFIG_DIR is set, a record of the session in its projects directory showing one note delivered, and a record of the session in the home's .claude/projects showing another
  When engram auto deliver is dispatched for a prompt of the session, without --sessions
  Then the block names the second note only

Scenario: 793d4f1c The home's .claude/projects holds them by default
  Given neither variable is set, and a record of the session in the home's .claude/projects showing one note delivered
  When engram auto deliver is dispatched for a prompt of the session, without --sessions
  Then the block names the other note only

Scenario: 793d4f1d No record is read when no directory can be found
  Given neither variable is set, and a home directory that cannot be found
  When engram auto deliver is dispatched for a prompt of a session, without --sessions
  Then no record is read, and the block names the notes that bear on the prompt
```

### 793d4g Judging a turn

Requirement: 793d4g The `turn-judgement` MUST read one JSON object from standard input (the `session`'s ID, the final `answer`, the `text` the [turn](./793a_product-spec.md#793a1a-terms) began with, the `origin` of that [prompt](./793a_product-spec.md#793a1a-terms)), print nothing when the origin is not one the user sends ([793d4m Telling the prompts the user sends](#793d4m-telling-the-prompts-the-user-sends)), find the [session](../../GLOSSARY.md#session)'s record as [793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds) does in the directory of [793d4f Finding the records of sessions](#793d4f-finding-the-records-of-sessions), and print the request for the model as one JSON object: `model` (`haiku`), `effort` (`low`), `system`, `prompt`, `maxTokens` and `timeoutMs`. The `system` MUST hold the instructions of the judgement and nothing of the record, and the `prompt` none of the instructions, so that the model reads them as instructions. The prompt holds, between `<window>` and `</window>`, the context lines, the line `--- the turn ---` and the turn's lines, and, between `<notes>` and `</notes>`, the [notes](../../GLOSSARY.md#permanent-note) nearest to the turn (what the lookup of [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) returns for the text of the turn's prompts, each as a block line, cut to 400 characters) or `(none)`.

A prompt is a row of the record that the user sent; a row the [harness](../../GLOSSARY.md#harness) wrote is none and makes no line: a row marked as meta (the body of a [skill](../../GLOSSARY.md#skill) the [agent](./793a_product-spec.md#793a1a-terms) invoked), a row stamped with an origin the user does not send ([793d4m Telling the prompts the user sends](#793d4m-telling-the-prompts-the-user-sends)) and a row whose text begins as [792b15c3 Harness noise](../../specs/technical/transcript-reader.md#792b15c3-harness-noise) (harness noise) (a system reminder, a skill's base directory, the caveat or output of a local command, a task notification); a block of a prompt that begins so is left out, and a slash command's prompt is its arguments, none when it has none. The turn begins at the last prompt with the input `text` (else the last prompt) and ends before the next prompt and after the last line of the agent, which is the final text, added as the last line when no line of the turn holds its first 80 characters; the conversation before it is the context. A line is one message (`User: `, `Agent: ` or `Agent used ` and the tool's name; a break written ` ⏎ `), redacted ([793d4j Redacting credentials](#793d4j-redacting-credentials)) before it is cut or joined, then cut to its first 1,200 characters; a `<` in a line of the window or of a note MUST be written `&lt;`, which the instructions say, so that no message or note can close the window or open a section; the context is its last 6 lines; the window is cut to 8,000 characters, the oldest context first and then the turn's middle, replaced by the line `--- some of the turn is left out ---`. The command MUST read the record once and redact only the lines of the window, so that its time does not grow with the session: 1 MB within 2 seconds.

It MUST print nothing and succeed when the record holds no prompt, and print nothing and fail, naming the cause, when the input is not a JSON object, no [vault](../../GLOSSARY.md#vault) exists at the vault's path, the session has no record in the directory of records, or the lookup fails, the cause holding no text of the turn's prompts ([793b107 A delivery that could not be done is reported without the prompt's text](./793b_decision-log.md#793b107-a-delivery-that-could-not-be-done-is-reported-without-the-prompts-text)): the capture is then recorded once, with its reason, and the model is not asked.

Owner: `turn-judgement` (new; `turn-judgement.md` at promotion). Supports: [793a2c1a A correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1b A request to remember becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1c An approach the user confirmed becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1d The agent's own correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1e An approach that succeeded with evidence becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1f Guesses, plans and chatter produce no note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1g A credential in the turn is not in the note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1h A thanks that names no approach produces no note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1i A request to remember is written when the agent says it did not save it](./793a_product-spec.md#793a2c1-uat-scenarios), [793a3c1a A lesson an earlier note states is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), [793a3c1b A lesson the agent saved itself is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), [793a3c1c A lesson no earlier note states is written](./793a_product-spec.md#793a3c1-uat-scenarios).

#### 793d4g1 Scenarios

```gherkin
Scenario: 793d4g1a The request holds the instructions, the context and the turn
  Given the record of a session with a first turn and a second turn in which the user corrected the agent
  When the judgement is given the session and the second turn's final text
  Then it prints a request for the model haiku at effort low whose prompt holds the instructions, the first turn's lines, the line --- the turn ---, and the second turn's lines, each opening with its label

Scenario: 793d4g1b The final text is added when the record lacks it
  Given the record of a session whose last row is the user's prompt
  When the judgement is given the session and a final text
  Then the last line of the turn is the final text, labelled as the agent's

Scenario: 793d4g1c A session with no prompt in its record is judged nothing
  Given the record of a session with no prompt
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
  Given a vault holding a note that says when the staging database is reset, and the record of a session whose turn asks about the staging database
  When the judgement is given the session
  Then the request lists the staging note, with its file name, its situation and its body on one line

Scenario: 793d4g1i A failed lookup fails the command and prints nothing
  Given a lookup that fails
  When the judgement is given the record of a session with a prompt
  Then it fails with an error that names the lookup's failure and holds no text of the turn's prompts, and it prints nothing

Scenario: 793d4g1j A note cannot reshape the window
  Given a note whose situation and body hold line breaks, a credential and more than 400 characters
  When the request is made
  Then the note is one line of the section of notes, the credential is redacted, and the body is cut to 400 characters

Scenario: 793d4g1g A long conversation is cut to the bound
  Given a record with more context than the window holds, and a turn with more lines than it holds
  When the judgement is given the session
  Then the context is its last 6 lines or fewer, the window is no longer than 8,000 characters, the first line of the turn and its last lines are kept, and the line that says that some of the turn is left out stands where they were

Scenario: 793d4g1k A prompt of text blocks is one prompt, and the result of a tool is none
  Given a record whose last prompt is made of two text blocks and an image, followed by a row that holds only the result of a tool
  When the judgement is given the session
  Then the turn begins with one line for the prompt, holding the text of both blocks, and the result of the tool makes no line

Scenario: 793d4g1l The body of a skill is no prompt
  Given the record of a session whose turn began with a prompt of the user, in which the agent invoked a skill and the harness wrote the skill's body as a row marked as meta
  When the judgement is given the session
  Then the request has no line for the skill's body, and the turn begins with the user's prompt

Scenario: 793d4g1m A system reminder in the row of a prompt is not part of the prompt
  Given a record whose last prompt is a row of two text blocks, the user's words and a system reminder, followed by a row that holds only a system reminder
  When the judgement is given the session
  Then the prompt is one line of the user's words alone, and the row of the system reminder makes no line

Scenario: 793d4g1n A task notification delivered into a turn is no prompt
  Given the record of a session whose turn began with a prompt of the user, and in which a task notification was delivered as a row of the user's before the agent's last text
  When the judgement is given the session
  Then the request has no line for the notification, and the turn begins with the user's prompt

Scenario: 793d4g1o A command that ran locally is no prompt, and a slash command is its arguments
  Given the record of a session whose turn began with a prompt of the user and holds, after it, the caveat, the name with no arguments and the output of a command that ran locally, and a record whose prompt is a slash command given arguments
  When the judgement is given each
  Then none of the rows of the command that ran locally is a line of the first request, and the prompt of the second is the arguments

Scenario: 793d4g1p A row of an origin the user does not send is no prompt
  Given a record whose turn began with a prompt of the user, and a row of the user's stamped with the origin peer whose text opens with none of the marks of the harness, delivered before the agent's last text
  When the judgement is given the session
  Then the request has no line for that row, and the turn begins with the user's prompt

Scenario: 793d4g1w A session with no record fails the command
  Given a session that no record holds, and no session at all
  When the judgement is given each
  Then it prints nothing and fails each time, with an error that names the session and the directory of the records

Scenario: 793d4g1x A vault that does not exist fails the command
  Given a path that holds no vault, and the record of a session with a prompt
  When the judgement is given the session
  Then it prints nothing, fails with an error that names the path, and looks nothing up

Scenario: 793d4g1q A long record is judged in time
  Given the record of a session of 1 MB, of which the turn is the last prompt
  When the judgement is given the session
  Then it prints the request within 2 seconds

Scenario: 793d4g1r A prompt typed ahead does not begin the turn of the one before it
  Given the record of a session whose first turn ended with the agent saying that its guess was wrong and what the file says, and whose second prompt, typed ahead, is already in the record
  When the judgement is given the session, the text of the first prompt and the first turn's final text
  Then the turn of the request is the first turn's lines, and the second prompt is no line of the request

Scenario: 793d4g1s A turn that a notification started after it is not part of it
  Given the record of a session whose turn began with a prompt of the user and ended with the agent's text, followed by a task notification and the lines of the turn it started
  When the judgement is given the session, the text of the prompt and the final text
  Then the turn of the request ends with the final text, and holds nothing of the turn the notification started

Scenario: 793d4g1t A text that no prompt has begins the turn at the last prompt
  Given the record of a session with two prompts, and a text that neither has
  When the judgement is given the session and that text
  Then the turn of the request begins at the last prompt

Scenario: 793d4g1u The instructions are the system of the request
  Given the record of a session with a prompt
  When the judgement is given the session
  Then the system of the request holds the instructions of the judgement and nothing of the record, and its prompt holds the window and the notes and none of the instructions

Scenario: 793d4g1v A message or a note cannot close the window or open a section
  Given messages and notes whose texts hold the tags that close and open the window and the section of notes, and other uses of the character that opens a tag
  When the request is made
  Then no line of the window or of the section of notes holds that character, each holds &lt; in its place, and the prompt holds each tag of the window and of the notes once
```

### 793d4h Writing the lessons of a reply

Requirement: 793d4h The `lesson-writing` MUST read one JSON object from standard input: the `session`'s ID, the `turn`'s ID, and the model's `reply` or, when it gave none, the `failure`. The reply is a JSON array of lessons, fenced or not; a lesson has a `kind` (`feedback` with `behavior`, `impact` and `action`, or `fact` with `subject`, `predicate` and `object`), a `slug` of 1 to 64 lower-case letters, digits and dashes, and a `situation`, and may name as `covered` the file name of a [note](../../GLOSSARY.md#permanent-note) of the [vault](../../GLOSSARY.md#vault) that already states it. A lesson whose `covered` names a file of the vault MUST NOT be written but recorded as skipped with that name (a name that is not a file of the vault, or not a file name, covers nothing). Each other lesson MUST be written as a note of its kind through the learn path of [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand), at the top of the vault's order, with the source `written automatically (session <the session's ID>)`, and its path printed. Every field is written as one line (its line breaks and white space become single spaces and its control and format characters are dropped), so that a field cannot make a line of its own, such as one opening with `Supersedes:`, which the vault reads as a link. A reply that is not a list, and a lesson that lacks a field or has an unknown kind, write nothing and are recorded with the reason ([793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did)); a reply cut before its list ends keeps the lessons that are whole, and the cut is recorded as skipped.

Owner: `lesson-writing` (new; `lesson-writing.md` at promotion). Supports: [793a2c1a A correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1b A request to remember becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1c An approach the user confirmed becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1d The agent's own correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1e An approach that succeeded with evidence becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1f Guesses, plans and chatter produce no note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1g A credential in the turn is not in the note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1h A thanks that names no approach produces no note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1i A request to remember is written when the agent says it did not save it](./793a_product-spec.md#793a2c1-uat-scenarios), [793a3c1a A lesson an earlier note states is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), [793a3c1b A lesson the agent saved itself is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), [793a3c1c A lesson no earlier note states is written](./793a_product-spec.md#793a3c1-uat-scenarios).

#### 793d4h1 Scenarios

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

### 793d4j Redacting credentials

Requirement: 793d4j The `turn-judgement` MUST redact every text of the [turn](./793a_product-spec.md#793a1a-terms) shaped as a credential before cutting or joining it and before the model sees it, and the `lesson-writing` every field of a lesson before writing it, so that no [note](../../GLOSSARY.md#permanent-note) holds a credential that appeared in the turn. Each text is first written as one line (control and format characters dropped), then redacted, then cut, so that a credential split by characters that are not shown is whole when redacted. The shapes: a private key block; a JSON web token; a key or token that opens with its issuer's prefix (`sk-`, `pk-`, `rk-`, `ghp_`, `gho_`, `ghs_`, `ghu_`, `ghr_`, `github_pat_`, `npm_`, `glpat-`, `AKIA`, `ASIA`, `AIza`, `xoxa-`, `xoxb-`, `xoxp-`, `xoxr-`, `xoxs-`); a `Bearer` or `Basic` credential of 12 characters or more; the user and password of a URL; the value of an `Authorization` header; the value given by `=`, `:` or `is`, with up to six words of the sentence between, to a name holding `key`, `token`, `secret`, `password`, `passwd`, `pwd`, `credential` or `private-key` in any case with anything around the word (`accessKey`, `x-api-key`, `DB_PASSWORD`); a run of 32 or more letters, digits, `+`, `_` and `=` that holds both a letter and a digit and is not a commit's 40 hexadecimal digits. It becomes `[redacted]`, the name of an assignment kept. A secret of another shape (a password a sentence gives without `is` or `:`, a token split over two lines) is not redacted.

Owner: `turn-judgement` and `lesson-writing`. Supports: [793a2c1g A credential in the turn is not in the note](./793a_product-spec.md#793a2c1-uat-scenarios).

#### 793d4j1 Scenarios

```gherkin
Scenario: 793d4j1a No text shaped as a credential survives
  Given any text with a credential of each of the shapes, among any other text
  When the text is redacted
  Then the credential is not in the text, and the text around it is as it was

Scenario: 793d4j1b The name of an assignment is kept, and a text that is not a credential is not touched
  Given the names password, api_key and secret_token given values, a commit ID of 40 hexadecimal digits, a long word of letters only, and a sentence
  When the text is redacted
  Then each name is kept and its value is redacted, and the commit ID, the word and the sentence are as they were

Scenario: 793d4j1c The request holds no credential of the turn
  Given a record whose prompts and replies hold credentials of every shape
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

### 793d4k Recording what automatic memory did

Requirement: 793d4k The `lesson-writing` MUST record what became of each reply or failure it reads, where the display of [793a2e The user can see what it did](./793a_product-spec.md#793a2e-the-user-can-see-what-it-did) can read it: per lesson written, kind `written` with the [note](../../GLOSSARY.md#permanent-note)'s file name; per lesson left out, `skipped` with its slug, when it is one ([793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply)), and the reason; per reply that is not a list, `skipped` with the reason; per reply with no lesson, `nothing`; per failure, `failed` with the failure. A lesson that cannot be written is recorded `failed` and the command then succeeds, so the mod does not report it a second time. A record is a JSON file of its own in a directory named for the [session](../../GLOSSARY.md#session)'s ID under `auto` in the engram data directory, its name opening with the time it was made; it holds the time, the session's ID, the [turn](./793a_product-spec.md#793a1a-terms)'s ID, the kind, the note, the slug and the reason. The reason is written as one line, redacted ([793d4j Redacting credentials](#793d4j-redacting-credentials)) and only then cut in its middle to 300 characters, so that the step a failure names at its beginning and the cause at its end stay. A record that cannot be written does not fail the command unless it records a failure, when the command fails with the reason.

Owner: `lesson-writing`. Supports: [793a2c1f Guesses, plans and chatter produce no note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2c1h A thanks that names no approach produces no note](./793a_product-spec.md#793a2c1-uat-scenarios), [793a2d1e A write that fails does not break the session](./793a_product-spec.md#793a2d1-uat-scenarios).

#### 793d4k1 Scenarios

```gherkin
Scenario: 793d4k1a A lesson that is written is recorded with its note
  Given a reply that holds one lesson
  When the lessons are written
  Then one record, of the kind written, holds the file name of the note, the session's ID and the turn's ID

Scenario: 793d4k1b A reply with no lesson is recorded as nothing
  Given a reply that is an empty list
  When the lessons are written
  Then one record, of the kind nothing, holds the session's ID and the turn's ID

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
```

### 793d4m Telling the prompts the user sends

Requirement: 793d4m The `prompt-delivery`, the `turn-judgement` and the `conversation-sweep` MUST each read the `origin` of the JSON object they are given, the kind Claude Code stamps on a [prompt](./793a_product-spec.md#793a1a-terms), and act only for a prompt the user sends, one whose origin is `composer`, `bridge`, `slack-ping` or `sdk`; for any other origin, including none, each MUST print nothing, succeed and look nothing up. The `turn-judgement` and the `conversation-sweep` MUST do the same for the origin `sdk`, the headless host's own [turn](./793a_product-spec.md#793a1a-terms) (`claude -p`, the SDK), unless the user switched on capturing such turns with the flag `--capture-sdk` or the environment variable `ENGRAM_AUTO_CAPTURE_SDK` set to a true value (`1`), so that a script run with `claude -p` under the user's own home writes and indexes nothing by default. When the `turn-judgement` leaves out a turn for that reason it MUST record that as [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did) records a reply that is not a list, a record of the kind `skipped` with no slug and a reason that says the turn was not captured because headless prompts are captured only when `ENGRAM_AUTO_CAPTURE_SDK` is set, in the directory of its `--records` flag, else `auto` in the engram data directory, and still print nothing and succeed; it MUST record nothing for any other origin it leaves out. The definition is theirs: the mod passes the origin it received and decides nothing ([793b64 The prompts the user sends](./793b_decision-log.md#793b64-the-prompts-the-user-sends), [793b84 A headless prompt is judged only when the user asks](./793b_decision-log.md#793b84-a-headless-prompt-is-judged-only-when-the-user-asks)).

Owner: `prompt-delivery`, `turn-judgement` and `conversation-sweep`. Supports: [793a2d1b Automatic memory's work starts no session of its own](./793a_product-spec.md#793a2d1-uat-scenarios). No clause of a product requirement names an origin, so none of its scenarios does: all are raised by prompts the user sends, which this requirement lets through.

#### 793d4m1 Scenarios

```gherkin
Scenario: 793d4m1a A prompt of an origin the user sends is delivered for and its turn is judged
  Given a vault holding a note that bears on a prompt, the record of a session whose turn began with that prompt, and each of the origins composer, bridge, slack-ping and sdk, the judgement asked to judge the turns of headless prompts
  When the delivery is given the prompt, and the judgement is given the turn, with each origin
  Then the delivery prints its block and the judgement prints its request, with each

Scenario: 793d4m1b A prompt of any other origin is delivered nothing and its turn is judged nothing
  Given the same vault and record, and any origin but those four, including none: task-notification, scheduled-trigger, peer, channel, plugin, unclassified and the empty origin
  When the delivery is given the prompt, and the judgement is given the turn, with it
  Then each prints nothing, succeeds, and looks nothing up

Scenario: 793d4m1c The turn of a headless prompt is judged only when the user asks
  Given the record of a session whose turn began with a prompt of the origin sdk
  When the judgement is given the turn, without the flag --capture-sdk and with it
  Then without it the judgement prints nothing and succeeds, and with it the judgement prints its request

Scenario: 793d4m1d A turn that a background task's notification starts is not judged
  Given the user has run engram update, and a session in which the agent may run a sleep command but may not run engram commands
  When the agent starts a task in the background in a turn the user began, and the task's notification starts a turn of its own that the agent answers
  Then automatic memory has made one record, that of the user's turn, and the vault holds no note

Scenario: 793d4m1e A headless turn that is not judged is recorded as skipped with the reason
  Given the record of a session whose turn began with a prompt of the origin sdk, a place for the records, and, in turn, an origin the user does not send
  When the judgement is given the turn, without the flag --capture-sdk and with it
  Then without it the judgement prints nothing and succeeds, and one record of the kind skipped, with the session's ID and the turn's ID and a reason that says the turn was not captured and names ENGRAM_AUTO_CAPTURE_SDK, is made; with it no record is made, and for the other origin no record is made
```

### 793d4n Sweeping the conversation

Requirement: 793d4n The `conversation-sweep` MUST read one JSON object from standard input (the `session`'s ID and the `origin`), act only for the [prompts](./793a_product-spec.md#793a1a-terms) that [793d4m Telling the prompts the user sends](#793d4m-telling-the-prompts-the-user-sends) lets the judgement act for, and for them find the [session](../../GLOSSARY.md#session)'s record as [793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds) does in the directory of [793d4f Finding the records of sessions](#793d4f-finding-the-records-of-sessions), and index it in the chunk index as `--transcript` of [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand) does, as the one explicit source, printing nothing. It MUST print nothing, succeed and index nothing for any other origin; print nothing and fail, naming the cause, when the input is not a JSON object, the session has no record in the directory of records or the index fails; wait for the lock that `engram ingest` takes on the manifest when another ingest holds it; and leave the chunk index usable when it is ended at any moment, so that a later sweep indexes the record whole. It MUST NOT create a [vault](../../GLOSSARY.md#vault) ([793d4e First-use gating](#793d4e-modified-first-use-gating)).

Owner: `conversation-sweep` (new; `conversation-sweep.md` at promotion). Supports: [793a2f1a The conversation so far is findable after a turn](./793a_product-spec.md#793a2f1-uat-scenarios).

#### 793d4n1 Scenarios

```gherkin
Scenario: 793d4n1a The record of the session is indexed as the one explicit source
  Given the record of a session in the directory of records, and an index
  When the sweep is given the session
  Then the index is run for the record's path alone, as an explicit transcript, and nothing is printed

Scenario: 793d4n1b A session with no record fails the command
  Given a session that no record holds, and no session at all
  When the sweep is given each
  Then it prints nothing and fails each time, with an error that names the session and the directory of the records, and indexes nothing

Scenario: 793d4n1c A prompt of an origin that is not swept is swept nothing
  Given the record of a session, and each of the origins task-notification, peer, the empty origin, and sdk without the flag --capture-sdk
  When the sweep is given the session with each, and then with sdk and the flag
  Then with each of the first it prints nothing, succeeds and indexes nothing, and with the last it indexes the record

Scenario: 793d4n1d A failed index fails the command, naming the cause
  Given an index that fails, and an input that is not a JSON object
  When the sweep is given the record of a session, and the input
  Then it prints nothing and fails each time, with an error that names the index's failure and the input

Scenario: 793d4n1e A sweep that is ended leaves the chunk index usable
  Given the record of a session of several turns, and the real engram
  When the sweep is ended while it works, and then runs again
  Then the manifest is not corrupt after the first, and the second indexes the whole record

Scenario: 793d4n1f Two sweeps at once index both records
  Given the records of two sessions, and the real engram
  When both sweeps run at once
  Then the manifest records both sources, and each record is indexed whole
```

### 793d4p Telling what automatic memory did

Requirement: 793d4p The `prompt-delivery` MUST print on standard error the lines that show the user what [automatic memory](./793a_product-spec.md#793a1a-terms) did, one to a line and none when it has nothing to show: first a line for each [note](../../GLOSSARY.md#permanent-note) it delivered, `delivered the note` and its file name, in the order of the block ([793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt)), then a line for each record of the kind `written`, `skipped` or `failed` that [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did) makes for the [session](../../GLOSSARY.md#session) of the request, read from the directory its `--records` flag names, else `auto` in the engram data directory, that it has not shown, oldest first and at most 5; the others wait for the next [prompt](./793a_product-spec.md#793a1a-terms). A line of a record is the time of the record (`15:04:05`) and: for `written`, `wrote the note` and its file name; for `skipped`, `left out the lesson` and its slug, or `left out`, then a colon and the reason; for `failed`, `failed:`, then `writing` and its slug and a colon when it has a slug, and the reason. A reason is written as one line, redacted ([793d4j Redacting credentials](#793d4j-redacting-credentials)) and, when it is longer than 240 characters, cut in its middle so that both ends stay; every line is one line of single spaces with no control or format character. After printing the lines it MUST mark each record whose line it printed by an empty file beside the record, named for it with the suffix `.told`, so that a line is shown once; a record with a mark makes no line, nor does a record of another session, of the kind `nothing`, or that cannot be read, nor a directory that cannot be read, and a mark that cannot be written is the end of it. The lines are no part of the block ([793d4c Framing the notes of a block as data](#793d4c-framing-the-notes-of-a-block-as-data)).

Owner: `prompt-delivery`. Supports: [793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1b A write that finishes after the next prompt is shown when it finishes](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1c A delivery that could not be done is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1d A write that failed is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1e A sweep that failed is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), [793a2e1f A turn that is not captured is shown with the reason](./793a_product-spec.md#793a2e1-uat-scenarios).

#### 793d4p1 Scenarios

```gherkin
Scenario: 793d4p1a A record that was not shown becomes a line
  Given the records of a session of the kinds written, skipped with a slug, skipped without one, failed without a slug and failed with one, and none of them shown
  When the delivery is given a prompt of that session
  Then it prints on standard error a line for each, oldest first, each beginning with the time of its record and saying what became of it

Scenario: 793d4p1b A record of the kind nothing makes no line
  Given the records of a session of the kind nothing only
  When the delivery is given a prompt of that session, and no note bears on it
  Then it prints nothing and succeeds

Scenario: 793d4p1c A line that was shown is not shown again
  Given the records of a session, and a mark beside the first
  When the delivery is given a prompt of that session, and then another
  Then the first prompt prints the line of the second record only, and the second prompt prints no line

Scenario: 793d4p1d A reason is one line, redacted and cut in its middle
  Given a failed record whose reason holds line breaks, a credential and more than 240 characters
  When the delivery is given a prompt of its session
  Then its line holds the reason as one line without the credential, at most 240 characters of it, and both ends of it

Scenario: 793d4p1e At most five lines are shown at once
  Given the records of a session that make seven lines
  When the delivery is given a prompt of that session, and then another
  Then the first prompt prints five lines and the second the other two

Scenario: 793d4p1f The records of another session, a record that cannot be read and a missing directory make no line
  Given a directory of records that holds the records of another session and a file that is not a record, and a session with no directory of its own
  When the delivery is given a prompt of that session and of the other
  Then it prints what the notes make for each and succeeds, and the lines of the other session are shown to the other only

Scenario: 793d4p1g The records are read from the directory the flag names, else from the engram data directory
  Given records in the directory that --records names and others in the auto directory of the engram data directory
  When engram auto deliver is dispatched with the flag, and then without it
  Then the first prints the line of the first records and the second the line of the others

Scenario: 793d4p1h A note delivered is shown as a line
  Given a vault holding a note that bears on a prompt
  When the delivery is given the prompt
  Then it prints the block on standard output, and on standard error the line delivered the note and the file name of the note

Scenario: 793d4p1i A line shown is one line whatever the texts hold
  Given notes and records whose texts hold line breaks, control and format characters and lines that read as other lines, however they are mixed
  When the delivery is given a prompt of that session
  Then each line printed on standard error is one line of single spaces with no control or format character, and there is a line for each note delivered and for each record shown
```

## 793d5 Product-scenario mapping

### 793d5a Mapping of 793a2a1a

The product scenario [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1a The next session has memory after one engram update
  Given a machine with Claude Code on which the user has never run engram update, and a vault holding a note that says when the staging database is reset
  When the user runs engram update
  And starts a Claude Code session in a project other than the one engram update was run in, and sends a prompt asking when the staging database is reset
  Then the note is in the agent's context before the agent answers
```

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source), [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) and [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt).

### 793d5b Mapping of 793a2b1b

The product scenario [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2b1b A note delivered earlier in the session is not delivered again
  Given a session in which a prompt asking when the staging database is reset arrived with the staging note
  When the user sends another prompt about the staging database reset
  Then the second prompt arrives with nothing from the vault, and the staging note is in the agent's context once
```

Realized by [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow), [793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds) and [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt).

### 793d5c Mapping of 793a2b1c

The product scenario [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2b1c A note the agent looked up itself is not delivered
  Given a session in which, before the user ran engram update, the agent looked up the staging note itself with engram query, and a vault that has gained another note about the staging database since
  When the user runs engram update, resumes that session, and sends a prompt asking when the staging database is reset
  Then the prompt arrives with the other note and not the staging note
```

Realized by [793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds), with the flow and the delivery of [793d5b Mapping of 793a2b1b](#793d5b-mapping-of-793a2b1b).

### 793d5d Mapping of 793a2b1d

The product scenario [793a2b1d A prompt that matches no note arrives with nothing](./793a_product-spec.md#793a2b1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2b1d A prompt that matches no note arrives with nothing
  Given the user has run engram update, a vault whose notes have nothing to do with the prompt, and a searchable conversation that does
  When the user sends the prompt
  Then the prompt arrives with nothing from the vault, and the agent answers it
```

Realized by [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) and [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow).

### 793d5e Mapping of 793a2d1a

The product scenario [793a2d1a A slow lookup does not hold a prompt for more than 8 seconds](./793a_product-spec.md#793a2d1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2d1a A slow lookup does not hold a prompt for more than 8 seconds
  Given the user has run engram update, and a lookup that takes 20 seconds
  When the user sends a prompt
  Then the prompt reaches the agent within 8 seconds, as the user typed it and with nothing from the vault, and the agent answers it
```

Realized by [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the 7-second bound).

### 793d5f Mapping of 793a2a1e

The product scenario [793a2a1e A vault that does not exist is not created](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1e A vault that does not exist is not created
  Given a machine with Claude Code on which the user has run engram update, and whose configured vault no longer exists
  When the user starts a session and sends a prompt
  Then the agent answers the prompt, and the vault still does not exist
```

Realized by [793d4e First-use gating](#793d4e-modified-first-use-gating) (the command creates no [vault](../../GLOSSARY.md#vault)), [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) and [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow).

### 793d5g Mapping of 793a2a1d

The product scenario [793a2a1d Pi gets no automatic memory](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1d Pi gets no automatic memory
  Given a machine with both Claude Code and Pi
  When the user runs engram update
  Then the report lists the automatic memory under Claude Code and not under Pi, and nothing of it is installed under ~/.pi
```

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source) (mods are planned for Claude Code only).

### 793d5h Mapping of 793a2a1c

The product scenario [793a2a1c A dry run turns nothing on](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1c A dry run turns nothing on
  Given a machine with Claude Code on which the user has never run engram update, and a vault holding a note that says when the staging database is reset
  When the user runs engram update --dry-run
  And starts a session and sends a prompt asking when the staging database is reset
  Then the report lists the automatic memory as planned, and the agent's context holds no note
```

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source) (a dry run plans and reports the mod and writes none of it).

### 793d5i Mapping of 793a2a1b

The product scenario [793a2a1b A second update still delivers each note once](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1b A second update still delivers each note once
  Given a machine on which the user has run engram update once, and a vault holding a note that says when the staging database is reset
  When the user runs engram update again
  And starts a session and sends a prompt asking when the staging database is reset
  Then the note is in the agent's context once
```

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source) (one deployed mod, so one hook) and [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt).

### 793d5j Mapping of 793a2c1a

The product scenario [793a2c1a A correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1a A correction becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands and has answered a question about how to build a project
  When the user corrects the agent's answer in a prompt, and the turn ends
  Then within 2 minutes the vault holds a note that states the correction and says it was written automatically, and the agent wrote nothing to the vault
```

Realized by [793d2b Turn capture flow](#793d2b-turn-capture-flow), [793d4g Judging a turn](#793d4g-judging-a-turn), [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply) and [793d4f Finding the records of sessions](#793d4f-finding-the-records-of-sessions); the [agent](./793a_product-spec.md#793a1a-terms) may not run `engram` commands, so the [note](../../GLOSSARY.md#permanent-note) is the writing's.

### 793d5k Mapping of 793a2c1b

The product scenario [793a2c1b A request to remember becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1b A request to remember becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember something, and the turn ends
  Then within 2 minutes the vault holds a note that states it and says it was written automatically, and the agent wrote nothing to the vault
```

Realized by the components of [793d5j Mapping of 793a2c1a](#793d5j-mapping-of-793a2c1a); the instructions of the judgement name a request to remember as a lesson, and [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply) writes the kind `fact`.

### 793d5l Mapping of 793a2c1c

The product scenario [793a2c1c An approach the user confirmed becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1c An approach the user confirmed becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands and has done a task one way
  When the user tells the agent in a prompt that the way worked and to keep doing it, and the turn ends
  Then within 2 minutes the vault holds a note that states the approach and says it was written automatically
```

Realized by [793d4g Judging a turn](#793d4g-judging-a-turn) (the task is in the context of the request and the confirmation in the [turn](./793a_product-spec.md#793a1a-terms)) with the components of [793d5j Mapping of 793a2c1a](#793d5j-mapping-of-793a2c1a); the instructions name an approach the user confirmed as a lesson.

### 793d5m Mapping of 793a2c1d

The product scenario [793a2c1d The agent's own correction becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1d The agent's own correction becomes a note
  Given the user has run engram update, and a session in which the agent may read files but may not run engram commands
  When the user asks the agent in a prompt to guess what a file says and then to read it, and the agent says its guess was wrong and states what the file says, and the turn ends
  Then within 2 minutes the vault holds a note that states what the file says and says it was written automatically
```

Realized by [793d4g Judging a turn](#793d4g-judging-a-turn) (the lines of the [turn](./793a_product-spec.md#793a1a-terms) hold the guess and its correction, as the [agent](./793a_product-spec.md#793a1a-terms) states both in text) with the components of [793d5j Mapping of 793a2c1a](#793d5j-mapping-of-793a2c1a); the instructions name the agent's own correction as a lesson.

### 793d5n Mapping of 793a2c1e

The product scenario [793a2c1e An approach that succeeded with evidence becomes a note](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1e An approach that succeeded with evidence becomes a note
  Given the user has run engram update, and a session in which the agent may run a check script but may not run engram commands
  When the agent, unsure whether an approach will make the check pass, tries it in a turn and the turn shows that the check passed
  Then within 2 minutes the vault holds a note that states the approach and says it was written automatically
```

Realized by [793d4g Judging a turn](#793d4g-judging-a-turn) (the lines of the [turn](./793a_product-spec.md#793a1a-terms) hold what the [agent](./793a_product-spec.md#793a1a-terms) was unsure of and what it saw) with the components of [793d5j Mapping of 793a2c1a](#793d5j-mapping-of-793a2c1a); the instructions name an approach that succeeded with evidence as a lesson.

### 793d5o Mapping of 793a2c1f

The product scenario [793a2c1f Guesses, plans and chatter produce no note](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1f Guesses, plans and chatter produce no note
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user sends a prompt that the agent answers with a guess, a plan and chatter, and the turn ends
  Then 2 minutes after the turn ends the vault holds no note
```

Realized by the components of [793d5j Mapping of 793a2c1a](#793d5j-mapping-of-793a2c1a), whose instructions name a guess, a plan and chatter as no lesson, and by [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did): a reply with no lesson is recorded, so an empty result is a finished judgement.

### 793d5p Mapping of 793a2c1g

The product scenario [793a2c1g A credential in the turn is not in the note](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1g A credential in the turn is not in the note
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember how to deploy, with the token and the password that the deploy needs, each in a recognised form, and the turn ends
  Then within 2 minutes the vault holds a note that states how to deploy, and no note holds the token or the password
```

Realized by [793d4j Redacting credentials](#793d4j-redacting-credentials), [793d4g Judging a turn](#793d4g-judging-a-turn) (redaction comes before the cut) and [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply), with the components of [793d5j Mapping of 793a2c1a](#793d5j-mapping-of-793a2c1a).

### 793d5q Mapping of 793a3c1a

The product scenario [793a3c1a A lesson an earlier note states is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a3c1a A lesson an earlier note states is not written again
  Given the user has run engram update, a vault holding a note that the user saved earlier and that states a lesson, and a session in which the agent may not run engram commands
  When the user states the same lesson to the agent in a prompt, and the turn ends
  Then 2 minutes after the turn ends the vault holds the earlier note only
```

Realized by [793d4g Judging a turn](#793d4g-judging-a-turn) (the request lists the earlier [note](../../GLOSSARY.md#permanent-note) among the notes nearest to the [turn](./793a_product-spec.md#793a1a-terms)) and [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply) (a lesson the earlier note covers is not written and the record names that note), with the lookup of [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt).

### 793d5r Mapping of 793a3c1b

The product scenario [793a3c1b A lesson the agent saved itself is not written again](./793a_product-spec.md#793a3c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a3c1b A lesson the agent saved itself is not written again
  Given the user has run engram update, and a session in which the agent may run engram commands
  When the user asks the agent in a prompt to remember something and to save it with engram learn, and the turn ends
  Then 2 minutes after the turn ends the vault holds one note for it, the one the agent saved
```

Realized by the components of [793d5q Mapping of 793a3c1a](#793d5q-mapping-of-793a3c1a) and nothing new: the [note](../../GLOSSARY.md#permanent-note) the [agent](./793a_product-spec.md#793a1a-terms) saves with [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand) is in the [vault](../../GLOSSARY.md#vault) when the [turn](./793a_product-spec.md#793a1a-terms) ends, so the request lists it and [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply) writes nothing for the lesson it covers.

### 793d5s Mapping of 793a3e1a

The product scenario [793a3e1a The agent's own lookups and saves keep working](./793a_product-spec.md#793a3e1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a3e1a The agent's own lookups and saves keep working
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a session in which the agent may run engram commands
  When the user asks the agent in a prompt to look the staging note up with engram query and to save another note with engram learn
  Then the agent's lookup shows the staging note, and the vault holds the note the agent saved, as the agent wrote it
```

Realized by no component of its own: the [agent](./793a_product-spec.md#793a1a-terms)'s commands are the binary's, which [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) and [793d2b Turn capture flow](#793d2b-turn-capture-flow) neither wrap nor replace, and the writing changes the [vault](../../GLOSSARY.md#vault) only through the learn path, as new [notes](../../GLOSSARY.md#permanent-note).

### 793d5t Mapping of 793a3a1a

The product scenario [793a3a1a A written note is delivered with later prompts](./793a_product-spec.md#793a3a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a3a1a A written note is delivered with later prompts
  Given the user has run engram update, and a session in which the user corrected the agent and the correction became a note
  When the user sends a prompt that the note bears on, in that session and then in a new session
  Then each of the two prompts arrives with the note
```

Realized by no component of its own: [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply) writes through the learn path, which embeds the [note](../../GLOSSARY.md#permanent-note), so [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) finds it in a new [session](../../GLOSSARY.md#session); in the session where it was written [793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds) does not count it as held, because it was neither delivered nor looked up.

### 793d5u Mapping of 793a2a1f

The product scenario [793a2a1f A second update still writes each lesson as one note](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1f A second update still writes each lesson as one note
  Given a machine on which the user has run engram update once
  When the user runs engram update again, and corrects the agent in a prompt of a session
  Then the vault holds one note for the correction
```

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source) (one deployed mod, so one hook), with the components of [793d5j Mapping of 793a2c1a](#793d5j-mapping-of-793a2c1a).

### 793d5v Mapping of 793a2a1g

The product scenario [793a2a1g A lesson is not written to a vault that does not exist](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1g A lesson is not written to a vault that does not exist
  Given a machine with Claude Code on which the user has run engram update, and whose configured vault no longer exists
  When the user corrects the agent in a prompt of a session, and the turn ends
  Then the vault still does not exist 2 minutes after the turn ends
```

Realized by [793d4e First-use gating](#793d4e-modified-first-use-gating), [793d4g Judging a turn](#793d4g-judging-a-turn) (the judgement fails at its start, naming the path, so the model is not asked) and [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did) (the failure is recorded once).

### 793d5w Mapping of 793a2d1b

The product scenario [793a2d1b Automatic memory's work starts no session of its own](./793a_product-spec.md#793a2d1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2d1b Automatic memory's work starts no session of its own
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user corrects the agent in a prompt, and the turn ends
  Then within 2 minutes the vault holds a note for the correction, and no session but the user's exists, so nothing was delivered to or written from another
```

Realized by [793d2b Turn capture flow](#793d2b-turn-capture-flow) (the model call raises no [prompt](./793a_product-spec.md#793a1a-terms) or [turn](./793a_product-spec.md#793a1a-terms) and starts no [session](../../GLOSSARY.md#session)) and [793d4m Telling the prompts the user sends](#793d4m-telling-the-prompts-the-user-sends).

### 793d5x Mapping of 793a2c1h

The product scenario [793a2c1h A thanks that names no approach produces no note](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1h A thanks that names no approach produces no note
  Given the user has run engram update, and a session in which the agent may not run engram commands and has done a task one way
  When the user thanks the agent in a prompt, saying only that it worked, and the turn ends
  Then 2 minutes after the turn ends the vault holds no note
```

Realized by the components of [793d5o Mapping of 793a2c1f](#793d5o-mapping-of-793a2c1f), with instructions that name chatter, a thanks or "it worked" that names no approach, and a success no uncertainty preceded as no lesson ([793b87 The instructions carry the bar of the learn skill](./793b_decision-log.md#793b87-the-instructions-carry-the-bar-of-the-learn-skill)).

### 793d5y Mapping of 793a3c1c

The product scenario [793a3c1c A lesson no earlier note states is written](./793a_product-spec.md#793a3c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a3c1c A lesson no earlier note states is written
  Given the user has run engram update, a vault holding a note that the user saved earlier and that states a lesson, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember a different lesson about the same subject, and the turn ends
  Then within 2 minutes the vault holds the earlier note, unchanged, and a new note for the different lesson that says it was written automatically
```

Realized by the components of [793d5q Mapping of 793a3c1a](#793d5q-mapping-of-793a3c1a) the other way round: the request lists the earlier [note](../../GLOSSARY.md#permanent-note) as it does for a lesson it states, and the model names it only for a lesson it states, so [793d4h Writing the lessons of a reply](#793d4h-writing-the-lessons-of-a-reply) writes the lesson it does not name.

### 793d5z Mapping of 793a2c1i

The product scenario [793a2c1i A request to remember is written when the agent says it did not save it](./793a_product-spec.md#793a2c1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2c1i A request to remember is written when the agent says it did not save it
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember something and not to save it itself, the agent says that it did not save it, and the turn ends
  Then within 2 minutes the vault holds a note that states it and says it was written automatically
```

Realized by the components of [793d5k Mapping of 793a2c1b](#793d5k-mapping-of-793a2c1b), with instructions that say a request to remember is a lesson whatever the [agent](./793a_product-spec.md#793a1a-terms) says about saving it, even when it holds a secret ([793b90 A request to remember is a lesson whatever the agent says about saving it](./793b_decision-log.md#793b90-a-request-to-remember-is-a-lesson-whatever-the-agent-says-about-saving-it)).

### 793d5za Mapping of 793a2b1a

The product scenario [793a2b1a The notes that bear on a prompt arrive with it](./793a_product-spec.md#793a2b1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2b1a The notes that bear on a prompt arrive with it
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset and a note about formatting Python files
  When the user sends a prompt asking when the staging database is reset
  Then before the agent acts its context holds the staging note and not the formatting note, and the prompt in the conversation is exactly as the user typed it
```

Realized by [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the block arrives beside the [prompt](./793a_product-spec.md#793a1a-terms), which stays as typed), [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) (the [note](../../GLOSSARY.md#permanent-note) that bears on the prompt and not the other) and [793d4c Framing the notes of a block as data](#793d4c-framing-the-notes-of-a-block-as-data) (the form of the block).

### 793d5zb Mapping of 793a2f1a

The product scenario [793a2f1a The conversation so far is findable after a turn](./793a_product-spec.md#793a2f1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2f1a The conversation so far is findable after a turn
  Given the user has run engram update, and a session in which the agent may not run engram commands and the user has told the agent a first thing that only this conversation holds
  When the user tells it a second such thing in a prompt, and the turn ends
  Then within 2 minutes engram query returns a chunk of the conversation for each of the two things, and neither the user nor the agent ran engram ingest
```

Realized by [793d2c Conversation sweep flow](#793d2c-conversation-sweep-flow), [793d4n Sweeping the conversation](#793d4n-sweeping-the-conversation) (the record is found as [793d4f Finding the records of sessions](#793d4f-finding-the-records-of-sessions) finds it) and [793d4e First-use gating](#793d4e-modified-first-use-gating); the [agent](./793a_product-spec.md#793a1a-terms) may not run `engram` commands, so the indexing is the sweep's.

### 793d5zc Mapping of 793a2d1c

The product scenario [793a2d1c The work after a turn does not hold the next prompt](./793a_product-spec.md#793a2d1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2d1c The work after a turn does not hold the next prompt
  Given the user has run engram update, and a judgement, a write and a sweep that each take 60 seconds
  When the user sends a prompt, and then another while that work goes on
  Then the second prompt reaches the agent within 8 seconds, as the user typed it, and the agent answers it within 20 seconds
```

Realized by [793d2b Turn capture flow](#793d2b-turn-capture-flow) and [793d2c Conversation sweep flow](#793d2c-conversation-sweep-flow): the work is started from a timer, so no call of the host is in flight when the event settles, and the host, which holds the next event of the mod until the calls in flight settle, holds nothing.

### 793d5zd Mapping of 793a2d1d

The product scenario [793a2d1d The prompt and the reply are as they were](./793a_product-spec.md#793a2d1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2d1d The prompt and the reply are as they were
  Given the user has run engram update, and a session in which the user's correction became a note
  When the user sends a prompt that the note bears on, typed with extra spaces and a line break
  Then the prompt in the conversation is exactly as the user typed it, and the reply is exactly the agent's text
```

Realized by [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the [prompt](./793a_product-spec.md#793a1a-terms) stays as received, the block beside it) and [793d2b Turn capture flow](#793d2b-turn-capture-flow) (the event's result is given back as it was); nothing else of the mod edits a prompt or a reply.

### 793d5ze Mapping of 793a2d1e

The product scenario [793a2d1e A write that fails does not break the session](./793a_product-spec.md#793a2d1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2d1e A write that fails does not break the session
  Given the user has run engram update, and a vault that cannot be written
  When the user corrects the agent in a prompt, the turn ends, and the user sends another prompt
  Then the agent answers the second prompt as typed, with nothing about the failed write in its context, and the vault holds no new note
```

Realized by [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did) (a lesson that cannot be written is recorded once and the command succeeds), [793d2b Turn capture flow](#793d2b-turn-capture-flow) (the capture steps aside; a failed step is reported to the writing) and [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the lines of the failure are shown to the user and not added to the context).

### 793d5zf Mapping of 793a2e1a

The product scenario [793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt](./793a_product-spec.md#793a2e1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a session in which the user asked in one prompt when it is reset and to remember that it is backed up every Monday at 05:00 UTC
  When the user sends the next prompt
  Then the session shows a line that names the staging note delivered with the earlier prompt, and, with the next prompt, a line that names the note written from it
```

Realized by [793d4p Telling what automatic memory did](#793d4p-telling-what-automatic-memory-did) (the line for the [note](../../GLOSSARY.md#permanent-note) delivered, and the line of the record of the write, shown once), [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the mod shows each line to the user and adds none to the context) and [793d4c Framing the notes of a block as data](#793d4c-framing-the-notes-of-a-block-as-data) (the form of the block), with the delivery of [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt).

### 793d5zg Mapping of 793a2e1b

The product scenario [793a2e1b A write that finishes after the next prompt is shown when it finishes](./793a_product-spec.md#793a2e1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2e1b A write that finishes after the next prompt is shown when it finishes
  Given the user has run engram update, and a write that takes 12 seconds
  When the user corrects the agent in a prompt, sends another before the write finishes, and a third after it finished
  Then the session shows no line about the write with the second prompt, and with the third a line that names the note written
```

Realized by [793d4p Telling what automatic memory did](#793d4p-telling-what-automatic-memory-did) (the records are read at every [prompt](./793a_product-spec.md#793a1a-terms) and a write makes its records last, so the line is shown at the first prompt after the write finished), [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the mod shows the line) and [793d2b Turn capture flow](#793d2b-turn-capture-flow) (the write runs apart from the prompts, so the second prompt is not held for it).

### 793d5zh Mapping of 793a2e1c

The product scenario [793a2e1c A delivery that could not be done is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2e1c A delivery that could not be done is shown with its reason
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a lookup that takes 20 seconds
  When the user sends a prompt asking when the staging database is reset, the lookup becomes quick again, and the user sends another prompt
  Then the session shows, with the second prompt, a line that says the delivery for the first could not be done, with the reason
```

Realized by [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the mod ends the delivery at its bound, lets the [prompt](./793a_product-spec.md#793a1a-terms) go on and reports the failure to the writing, from a timer, and shows the line of the next prompt), [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did) (the writing records it as a failure) and [793d4p Telling what automatic memory did](#793d4p-telling-what-automatic-memory-did) (the delivery of the next prompt prints its line).

### 793d5zi Mapping of 793a2e1d

The product scenario [793a2e1d A write that failed is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2e1d A write that failed is shown with its reason
  Given the user has run engram update, and a vault that cannot be written
  When the user corrects the agent in a prompt, the turn ends, and the user sends another prompt
  Then the session shows, with the second prompt, a line that says the write failed, with the reason
```

Realized by [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did) (a lesson that cannot be written is recorded as failed, with its slug and the cause) and [793d4p Telling what automatic memory did](#793d4p-telling-what-automatic-memory-did) (the line says `writing` and the slug, and the reason is cut in its middle, so that the cause at its end stays), shown by [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow).

### 793d5zj Mapping of 793a2e1e

The product scenario [793a2e1e A sweep that failed is shown with its reason](./793a_product-spec.md#793a2e1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2e1e A sweep that failed is shown with its reason
  Given the user has run engram update, and a chunk index that cannot be written
  When the user says something in a prompt, the turn ends, and the user sends another prompt
  Then the session shows, with the second prompt, a line that says the sweep failed, with the reason
```

Realized by [793d2c Conversation sweep flow](#793d2c-conversation-sweep-flow) (the mod reports a sweep that fails to the writing, as a failure that names the sweep and the cause), [793d4k Recording what automatic memory did](#793d4k-recording-what-automatic-memory-did) (the writing records it as a failure) and [793d4p Telling what automatic memory did](#793d4p-telling-what-automatic-memory-did) (the delivery of the next [prompt](./793a_product-spec.md#793a1a-terms) prints its line, which [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) shows).

### 793d5zk Mapping of 793a2e1f

The product scenario [793a2e1f A turn that is not captured is shown with the reason](./793a_product-spec.md#793a2e1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2e1f A turn that is not captured is shown with the reason
  Given the user has run engram update, and headless prompts are captured only when the user asks
  When the user corrects the agent in a headless session, and sends another prompt
  Then the session shows, with the second prompt, a line that says the turn was not captured, with the reason
```

Realized by [793d4m Telling the prompts the user sends](#793d4m-telling-the-prompts-the-user-sends) (the judgement records a headless [turn](./793a_product-spec.md#793a1a-terms) it leaves out as skipped, with the reason) and [793d4p Telling what automatic memory did](#793d4p-telling-what-automatic-memory-did) (the delivery of the next [prompt](./793a_product-spec.md#793a1a-terms) prints its line, which [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) shows).

## 793d6 Recommended impact

- **Repository:** a new directory `agent-instructions/mods/`, a new package `internal/auto` and a new command group `engram auto` (`deliver`, `judge`, `write`, `sweep`); the update's deploy set gains the mods.
- **Dependencies:** none new in Go; a Claude Code release that loads mods (2.1.302 verified, 2.1.287 the first the earlier attempt found); `node`, which `targ check-full` needs to parse the mod's JavaScript and run its tests ([793b55 The mod's JavaScript is parsed by a test of the dev integration suite](./793b_decision-log.md#793b55-the-mods-javascript-is-parsed-by-a-test-of-the-dev-integration-suite)).
- **Toolchain:** a `targ acceptance` target runs the tests that drive a real Claude Code [session](../../GLOSSARY.md#session); they carry the `network` build tag, so `targ test` does not run them.
- **Operational:** one process start and one lookup per [prompt](./793a_product-spec.md#793a1a-terms), held at most by the mod's one timer of 7 seconds ([793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow)); after each [turn](./793a_product-spec.md#793a1a-terms), three process starts and one model call of the user's own login, apart from the turn ([793d2b Turn capture flow](#793d2b-turn-capture-flow), [793d2c Conversation sweep flow](#793d2c-conversation-sweep-flow)).
- **Risk:** every prompt of every Claude Code session runs the command ([793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow)). Open risk [793c32 Is the stall between the spawn of the command and its first instruction a risk to the product?](./793c_open-questions.md#793c32-is-the-stall-between-the-spawn-of-the-command-and-its-first-instruction-a-risk-to-the-product): on a loaded machine the time between the spawn of the command and its first instruction has reached the whole bound, so the prompt is held about 7 seconds with nothing delivered or shown; not optimised here.
