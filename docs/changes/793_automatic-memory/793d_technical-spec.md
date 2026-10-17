# Automatic Memory - Technical Spec

Filled depth first ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)): step 3 and step 4 run for one node at a time, in the order of
[793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan). Until the last node has been through step 4, this spec is incomplete and not approved. The
"Pending" lines below are the departure recorded in [793b10 Departure: depth-first fill](./793b_decision-log.md#793b10-departure-depth-first-fill). Its
sections follow the [technical spec guide](../../TECHNICAL-SPEC-GUIDE.md#default-structure). The solution-space
analysis, relative option impact and alternative options are omitted by decision ([793b12 Departure: no option analysis](./793b_decision-log.md#793b12-departure-no-option-analysis)): step 3 takes
the minimal components a scenario needs.

New technical requirements take the next free letter under [793d2 Recommended sequence diffs](#793d2-recommended-sequence-diffs) for an interaction requirement (one
per flow) or under [793d4 Recommended component-behavior diffs](#793d4-recommended-component-behavior-diffs) for a behavior requirement (one per component behavior).

## 793d1 Orientation

**The product problem** is [793a1 Problem Statement](./793a_product-spec.md#793a1-problem-statement): the
[agent](./793a_product-spec.md#793a1a-terms) consults the [vault](../../GLOSSARY.md#vault) only when it remembers to,
and the instruction to look first was refused in 6 of 6 headless trials.

**Current behavior.** The agent reaches the vault through its own commands, `engram query` and `engram show`, when a
[skill](../../GLOSSARY.md#skill) or its instructions tell it to. [`engram update`](../../GLOSSARY.md#engram-update)
deploys the skills of `agent-instructions/skills/` into each [harness](../../GLOSSARY.md#harness) it finds, through an
engram-owned root per harness that the harness's skills directory links into
([harness-deploy-sync](../../specs/technical/harness-deploy-sync.md)). `engram query` searches the notes, and the
chunk index when one is configured, for the phrases it is given
([query-engine](../../specs/technical/query-engine.md)).

**Why it is not enough.** Nothing in a [session](../../GLOSSARY.md#session) acts at a
[prompt](./793a_product-spec.md#793a1a-terms) without the agent's help, and `engram update` can install nothing but
skills, which only the agent runs.

**The recommended solution** is three parts. A **mod** is a folder that Claude Code loads as a plugin of function
hooks when it sits in `~/.claude/skills/<name>` (also as a link): its hooks module registers functions for events of
the session, and the `prompt.submit` event runs for each prompt before the prompt enters the session, where a
function may add context that the model reads beside the prompt and the user does not see. (Verified with Claude
Code 2.1.302: a mod in a linked skills folder loads in a headless `claude -p` session; the context it adds is
recorded in the session's [transcript](../../GLOSSARY.md#transcript) as a `hook_additional_context` record before the
agent's first reply.) (1) `engram update` deploys one mod, `engram-memory`, from `agent-instructions/mods/` through
the machinery that deploys the skills. (2) The mod runs a new command, `engram auto deliver`, for each prompt. (3) The
command looks up the [notes](../../GLOSSARY.md#permanent-note) that bear on the prompt text, as `engram query` would,
and prints them as one compact block, which the mod puts beside the prompt. The command, not the mod, holds the logic.

## 793d2 Recommended sequence diffs

### 793d2a Prompt delivery flow

Added interactions are drawn solid; the retained interaction is drawn inside a grey block.

```mermaid
sequenceDiagram
    actor User
    participant Session as Claude Code session
    participant Mod as automatic-memory-mod
    participant Delivery as prompt-delivery
    participant Engine as query-engine
    participant Context as session-context

    User->>Session: sends a prompt
    Session->>Mod: 793d2a prompt.submit with the prompt text
    Mod->>Delivery: 793d2a, 793d4a engram auto deliver, the text on standard input
    rect rgb(235, 235, 235)
        Delivery->>Engine: 792b24c1, 792b24c5, 792b24c20 the notes for the text
        Engine-->>Delivery: 792b24c5 the notes that match
    end
    Delivery->>Context: 793d4d the text the session holds
    Context-->>Delivery: 793d4d the text from the session's record
    Delivery-->>Mod: 793d4a the block
    Mod-->>Session: 793d2a the prompt unchanged, the block as context
    Session-->>User: the agent acts with the notes in its context
```

Requirement: 793d2a For each [prompt](./793a_product-spec.md#793a1a-terms) a Claude Code
[session](../../GLOSSARY.md#session) raises, the automatic-memory-mod MUST run `engram auto deliver` with the prompt
text and the session's ID, as one JSON object, on standard input, MUST add the command's output to the context of the prompt as one block when the command
exited with 0 and the output begins with `[engram memory]`, and nothing otherwise, and MUST leave the prompt text as it
received it. It MUST wait for the command no longer
than 7 seconds, so that the prompt is held for less than the 8 seconds of
[793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way), and MUST, when the command does not finish
within them, end it and let the prompt continue as it was, with nothing added.

Owner: automatic-memory-mod (a new component; its canonical file at promotion is `automatic-memory-mod.md`)

Supports: [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2d1a A slow lookup does not hold a prompt for more than 8 seconds](./793a_product-spec.md#793a2d1-uat-scenarios).

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
  Then the session holds the prompt exactly as sent and no context beside it

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
  Then the session holds the prompt exactly as sent and no context beside it, and the mod's hook did not fail
```

## 793d3 Recommended component diffs

- **automatic-memory-mod** (added): the mod `engram-memory`, a hooks module in `agent-instructions/mods/engram-memory/`
  (realization, informational). Qualification: responsibility, bringing the notes that bear on a prompt to the agent
  at the prompt is stated without naming a function; ownership, the routing and the placement of the block are its
  own, the lookup is the command's; independent verification, it is tested in a real Claude Code session against an
  `engram` that prints a chosen block; stability, it survives any change to how the command finds notes; necessity,
  without it nothing runs at a prompt.
- **prompt-delivery** (added): the command `engram auto deliver` (`internal/cli/auto.go`, `internal/auto`,
  informational). Qualification: responsibility, turning a prompt into the block of notes that bear on it is stated
  without naming a function; ownership, the choice and rendering of notes are its own, the hook is the mod's;
  independent verification, it is tested through its standard input and output over an in-memory vault; stability,
  it survives any change to the mod; necessity, the mod holds no logic.
- **session-context** (added): what a session's agent already holds of the vault, read from the session's record
  (`internal/auto`, `internal/cli/auto.go`, informational). Qualification: responsibility, telling what the agent holds
  is stated without naming a function; ownership, the format of the record and the rule for what counts as held are its
  own, the choice of notes to deliver is the delivery's; independent verification, it is tested over records of a
  session; stability, it survives any change to how the delivery uses it; necessity, a note the agent holds must not be
  delivered again ([793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)).
- **harness-deploy-sync** (updated): plans, links, reports and removes the mod as it does a skill directory, and
  reports it as a mod ([793b47 A dry run reports the mod as a mod](./793b_decision-log.md#793b47-a-dry-run-reports-the-mod-as-a-mod)).
- **query-engine** (retained): the lookup of notes for text, used as `engram query` uses it.
- **Claude Code's mod runtime** (an external boundary, retained): loads a mod from `~/.claude/skills/<name>`, runs its
  `prompt.submit` function for each prompt, runs the commands it asks for, and carries the context it returns to the
  model. The mod relies on that contract; nothing here tests it.

## 793d4 Recommended component-behavior diffs

### 793d4a Delivering the notes that bear on a prompt

Requirement: 793d4a The prompt-delivery MUST read one JSON object with the prompt's `text` from standard input and
MUST print one block on standard output. The block opens with a line that begins `[engram memory]` and holds one
line for each [note](../../GLOSSARY.md#permanent-note) that bears on the text, in the order the lookup gave them: a
dash, the note's file name, its situation in parentheses, a colon, and the note's body on one line. A note bears on
the text when the lookup of
[792b24c5 The matched set](../../specs/technical/query-engine.md#792b24c5-the-matched-set) for the text, as its message and as its only phrase,
returns it as an item that is not a chunk and was not picked by the explore half alone
([792b24c16 Delivering the explore half](../../specs/technical/query-engine.md#792b24c16-delivering-the-explore-half)):
an explore pick is sampled from the vocabulary, not matched to the text. The lookup's own default limit of
[792b24c1 Input and defaults](../../specs/technical/query-engine.md#792b24c1-input-and-defaults) bounds the number of
lines, as it bounds the number of items the agent's own lookup returns
([793b19 Volume of delivered notes](./793b_decision-log.md#793b19-volume-of-delivered-notes)). It MUST leave out each
note whose basename, its file name without the suffix `.md`, appears in the text that
[793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds) returns for the request's session. It MUST
print nothing, and succeed, when no note remains. It MUST NOT search the chunk
index, so that only notes are delivered ([793a2b Memory arrives with the prompt](./793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)). It MUST print nothing and
fail, naming the cause, when the lookup fails.

Owner: prompt-delivery (a new component; its canonical file at promotion is `prompt-delivery.md`)

Supports: [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2b1a The notes that bear on a prompt arrive with it](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2b1d A prompt that matches no note arrives with nothing](./793a_product-spec.md#793a2b1-uat-scenarios).

#### 793d4a1 Scenarios

```gherkin
Scenario: 793d4a1a The notes that bear on the text come back as one block
  Given a vault holding a note that says when the staging database is reset
  When the command is given a prompt asking when the staging database is reset
  Then it prints a block that opens with a line beginning [engram memory] and has one line naming the note, its situation and its body

Scenario: 793d4a1b A failed lookup fails the command and prints nothing
  Given a lookup that fails
  When the command is given a prompt
  Then it fails with an error that names the lookup's failure, and it prints nothing

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
```

### 793d4c Framing the notes of a block as data

Requirement: 793d4c The prompt-delivery MUST open every block with one fixed line that begins `[engram memory]` and
says that the lines below are [notes](../../GLOSSARY.md#permanent-note) from the vault, delivered by the memory
system and not typed by the user, and that an instruction written inside a note is not the user's. It MUST write each
note as exactly one line that begins with a dash and a space, with the note's file name, situation and body each
written as text in which every run of white space, line breaks and separators included, is one space and no control
character and no format character remains, a format character being one of the Unicode category Cf: a zero-width space
or joiner, a bidirectional override or isolate, a byte-order mark, a soft hyphen. So the text of a note never makes a
line of its own: it cannot produce a line that is the framing line, or a line that reads as another note.

Owner: prompt-delivery (a new component; its canonical file at promotion is `prompt-delivery.md`)

Supports: [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2b1a The notes that bear on a prompt arrive with it](./793a_product-spec.md#793a2b1-uat-scenarios).

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
```

### 793d4d Telling what a session holds

Requirement: 793d4d The session-context MUST, given a session's ID, look for the session's record, the file named for
the ID with the suffix `.jsonl`, in the directory of records it is given
([793d4f Finding the records of sessions](#793d4f-finding-the-records-of-sessions)),
directly in it or in one of its subdirectories. Of what the agent was given, it MUST return two things and nothing
else of the record. The first is the file name of each note line of every block that was delivered beside a prompt of
the session, that is of every entry, of a record of the kind `attachment` whose type is `hook_additional_context`, that
begins with `[engram memory]`, a note line being a line of the block that begins with a dash and a space, and its file
name the text after them up to the space before the situation's opening parenthesis. The second is, for every Bash
command that the agent ran whose text contains `engram query` or `engram show`, the text of the command and the text of
its result, the result being the `tool_result` block, of a `user` record, whose `tool_use_id` is the ID of the
command's `tool_use` block. It MUST return nothing of the situations and bodies of the notes of a block, of the other
entries beside a prompt, of the result of any other tool, of a prompt or of a reply. It MUST return nothing, without an
error, when the session has no record, when the record cannot be read, and when the ID is not a plain file name.

Owner: session-context (a new component; its canonical file at promotion is `session-context.md`)

Supports: [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios).

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

Requirement: 792b2c3 For every vault-resolving command other than `auto deliver`, the command-line-surface MUST ask the
[vault-identity-registry](../../specs/technical/vault-identity-registry.md) to ensure the vault at the resolved path before running the command, and
MUST NOT run the command when that step fails. For `auto deliver` it MUST NOT ask: the command runs against the path
as it is, and creates nothing when the path holds no vault.

Owner: [command-line-surface](../../specs/technical/command-line-surface.md)

Supports: [793a2a1e A vault that does not exist is not created](./793a_product-spec.md#793a2a1-uat-scenarios).

#### 793d4e1 Scenarios

```gherkin
Scenario: 793d4e1a The delivery command creates no vault
  Given no vault exists at the resolved path
  When engram auto deliver is dispatched
  Then no vault exists at the path afterwards, nothing is printed to standard output and no creation line is printed
```

### 793d4b Deploying the mods of the source

Requirement: 793d4b The harness-deploy-sync MUST plan one copy for every file under `agent-instructions/mods/` of the
source for Claude Code, keeping the file's path relative to that directory and naming the file's top-level directory
as the skill directory it is deployed in, so that a mod is deployed, linked, reported and removed as a skill directory
is. It MUST plan none for [Pi](../../GLOSSARY.md#harness), and none, without an error, when that directory does not
exist. It MUST report each mod directory it deploys, in the section of Claude Code in the report of
[`engram update`](../../GLOSSARY.md#engram-update), as a mod, with `agent-instructions/mods/<name>/` as its source,
where a skill directory is reported with `agent-instructions/skills/<name>/`. A dry run MUST plan and report the mod
and write none of its files. A second `engram update` MUST leave the deployed mod as the first left it: the same files
and the same link, with no second copy of the mod, and the report lists the mod once.

Owner: [harness-deploy-sync](../../specs/technical/harness-deploy-sync.md)

Supports: [793a2a1a The next session has memory after one engram update](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2a1d Pi gets no automatic memory](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2a1c A dry run turns nothing on](./793a_product-spec.md#793a2a1-uat-scenarios), [793a2a1b A second update still delivers each note once](./793a_product-spec.md#793a2a1-uat-scenarios).

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

Requirement: 793d4f The prompt-delivery MUST read the records of sessions from the directory that its `--sessions` flag
names, or, when the flag is not given, from the directory that `ENGRAM_TRANSCRIPT_DIR` names when it is set, else from
the `projects` directory of the directory that `CLAUDE_CONFIG_DIR` names when it is set, else from `<home>/.claude/projects`,
and from no directory when the home directory cannot be found. It MUST NOT change the session-log directory of
[792b17c6 Session-log directory](../../specs/technical/ingestion-source-discovery.md#792b17c6-session-log-directory),
which `engram ingest` sweeps.

Owner: prompt-delivery (a new component; its canonical file at promotion is `prompt-delivery.md`)

Supports: [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios).

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

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source) (the update puts the mod where Claude Code loads it), [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the mod runs the command for the prompt and puts its output beside the prompt) and [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) (the command finds the note and prints it).

### 793d5b Mapping of 793a2b1b

The product scenario [793a2b1b A note delivered earlier in the session is not delivered again](./793a_product-spec.md#793a2b1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2b1b A note delivered earlier in the session is not delivered again
  Given a session in which a prompt asking when the staging database is reset arrived with the staging note
  When the user sends another prompt about the staging database reset
  Then the second prompt arrives with nothing from the vault, and the staging note is in the agent's context once
```

Realized by [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the mod gives the command the session's ID), [793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds) (the session's record shows the block delivered with the first prompt) and [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) (the command leaves that note out). The second prompt arrives with nothing from the vault because the command prints nothing when the only note that bears on the prompt is held ([793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt)), and the mod then adds nothing ([793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow)).

### 793d5c Mapping of 793a2b1c

The product scenario [793a2b1c A note the agent looked up itself is not delivered](./793a_product-spec.md#793a2b1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2b1c A note the agent looked up itself is not delivered
  Given a session in which, before the user ran engram update, the agent looked up the staging note itself with engram query
  When the user runs engram update, resumes that session, and sends a prompt asking when the staging database is reset
  Then the prompt arrives with nothing from the vault
```

Realized by [793d4d Telling what a session holds](#793d4d-telling-what-a-session-holds) (the result of the agent's own `engram query` is in the session's record, and names the note) with the flow and the delivery of [793d5b Mapping of 793a2b1b](#793d5b-mapping-of-793a2b1b).

### 793d5d Mapping of 793a2b1d

The product scenario [793a2b1d A prompt that matches no note arrives with nothing](./793a_product-spec.md#793a2b1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2b1d A prompt that matches no note arrives with nothing
  Given the user has run engram update, a vault whose notes have nothing to do with the prompt, and a searchable conversation that does
  When the user sends the prompt
  Then the prompt arrives with nothing from the vault, and the agent answers it
```

Realized by [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) (the lookup finds no note, and does not search the chunk index, so the conversation is not delivered; the command prints nothing) and [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the mod adds nothing when the command prints nothing).

### 793d5e Mapping of 793a2d1a

The product scenario [793a2d1a A slow lookup does not hold a prompt for more than 8 seconds](./793a_product-spec.md#793a2d1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2d1a A slow lookup does not hold a prompt for more than 8 seconds
  Given the user has run engram update, and a lookup that takes 20 seconds
  When the user sends a prompt
  Then the prompt reaches the agent within 8 seconds, as the user typed it and with nothing from the vault, and the agent answers it
```

Realized by [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow): the mod ends the command at 7 seconds and lets the prompt continue unchanged.

### 793d5f Mapping of 793a2a1e

The product scenario [793a2a1e A vault that does not exist is not created](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1e A vault that does not exist is not created
  Given a machine with Claude Code on which the user has run engram update, and whose configured vault no longer exists
  When the user starts a session and sends a prompt
  Then the agent answers the prompt, and the vault still does not exist
```

Realized by [793d4e Modified: First-use gating](#793d4e-modified-first-use-gating) (the command runs without the first-use step, so it creates nothing) with [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt) (the lookup of a missing vault fails and the command prints nothing) and [793d2a Prompt delivery flow](#793d2a-prompt-delivery-flow) (the mod adds nothing, and the prompt goes on).

### 793d5g Mapping of 793a2a1d

The product scenario [793a2a1d Pi gets no automatic memory](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1d Pi gets no automatic memory
  Given a machine with both Claude Code and Pi
  When the user runs engram update
  Then the report lists the automatic memory under Claude Code and not under Pi, and nothing of it is installed under ~/.pi
```

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source): the mods are planned for Claude Code only, so the report of Pi lists the skills alone and Pi's directories hold none of the mod.

### 793d5h Mapping of 793a2a1c

The product scenario [793a2a1c A dry run turns nothing on](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1c A dry run turns nothing on
  Given a machine with Claude Code on which the user has never run engram update, and a vault holding a note that says when the staging database is reset
  When the user runs engram update --dry-run
  And starts a session and sends a prompt asking when the staging database is reset
  Then the report lists the automatic memory as planned, and the agent's context holds no note
```

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source): the dry run plans the mod, reports it as the mod it is, and writes none of it, so the session that follows loads no mod and the prompt arrives with no note.

### 793d5i Mapping of 793a2a1b

The product scenario [793a2a1b A second update still delivers each note once](./793a_product-spec.md#793a2a1-uat-scenarios), restated in full at its first mapping location:

```gherkin
Scenario: 793a2a1b A second update still delivers each note once
  Given a machine on which the user has run engram update once, and a vault holding a note that says when the staging database is reset
  When the user runs engram update again
  And starts a session and sends a prompt asking when the staging database is reset
  Then the note is in the agent's context once
```

Realized by [793d4b Deploying the mods of the source](#793d4b-deploying-the-mods-of-the-source): the second update leaves one deployed mod, so the session loads one hook and the prompt runs the command once; with [793d4a Delivering the notes that bear on a prompt](#793d4a-delivering-the-notes-that-bear-on-a-prompt), which prints the block of the note once.

## 793d6 Recommended impact

- **Repository:** a new directory `agent-instructions/mods/`; a new package `internal/auto` and a new command group
  `engram auto` with the command `deliver`; the update's deploy set gains the mods.
- **Dependencies:** none new in Go. A Claude Code release that loads mods: 2.1.302 is the build this was verified
  with, and the earlier attempt found 2.1.287 the first.
- **Toolchain:** a `targ acceptance` target runs the tests that drive a real Claude Code session; they carry the
  `network` or `integration` build tag, so `targ test` does not run them.
- **Operational:** one process start and one lookup for each prompt, held for at most the bound of
  [793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way) once the hold is built.
- **Risk and complexity:** every prompt of every Claude Code session runs the command; a failure of the command must
  not reach the session ([793a2d Never in the way](./793a_product-spec.md#793a2d-never-in-the-way)), which later nodes build.
