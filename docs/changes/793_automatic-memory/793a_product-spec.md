# Automatic Memory - Product Spec

> **North Star:** I want to add automatic learning and memory retrieval.
>
> The owner, Joe, 2026-10-16 ([793b2 North Star](./793b_decision-log.md#793b2-north-star)).

## 793a1 Problem Statement

Anyone who works with an [agent](#793a1a-terms) in Claude Code and keeps a [vault](../../GLOSSARY.md#vault)
gets value from the vault only when the agent consults it before acting and saves what a
[session](../../GLOSSARY.md#session) taught. Today both depend on the agent remembering, unprompted, to do so.
Issue #793 measured that the agent saves about one moment in twenty that is worth saving, and that the
instruction to look first was refused in 6 of 6 headless trials. The user cannot rely on the vault being
consulted or growing, and once the work leaves the agent, the user no longer sees it.

The needed outcome is the North Star, in the owner's words: "I want to add automatic learning and memory
retrieval."

### 793a1a Terms

- **Agent:** the AI the user works with in a Claude Code [session](../../GLOSSARY.md#session). It is not a
  [subagent](../../GLOSSARY.md#subagent).
- **Prompt:** a message the user sends to the [agent](#793a1a-terms).
- **Turn:** one [prompt](#793a1a-terms) and everything the agent does in answer to it, until it hands control
  back to the user.
- **Bears on a prompt:** a [note](../../GLOSSARY.md#permanent-note) bears on a [prompt](#793a1a-terms) when the command
  `engram query`, run with the prompt's text as its message and as its only phrase, returns it as a match for that text:
  not as a fragment of a conversation, and not as one of the extra notes the command samples to widen what it returns.
  The user can run that command and read the notes it returns.
- **Automatic memory:** what this change adds, done without the user or the agent asking: it puts the
  [notes](../../GLOSSARY.md#permanent-note) that bear on a prompt in the agent's context
  ([793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt)), writes notes for
  lessons ([793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent)) and
  makes the conversation searchable
  ([793a2f The conversation becomes searchable memory](#793a2f-the-conversation-becomes-searchable-memory)).

## 793a2 Added: Automatic memory

### 793a2a On after `engram update`

Requirement: 793a2a When the user runs [`engram update`](../../GLOSSARY.md#engram-update), other than with
`--dry-run` ([792a11f Previewing an update](../../specs/product/install-and-update.md#792a11f-previewing-an-update)),
on a machine that has Claude Code, [automatic memory](#793a1a-terms) MUST be on for every Claude Code
[session](../../GLOSSARY.md#session) the user starts on that machine afterwards, in any project. Turning it on
MUST NOT need any step other than running `engram update`, and automatic memory MUST use no account, key or
[vault](../../GLOSSARY.md#vault) the user has not already configured. After a second `engram update`, each
[prompt](#793a1a-terms) MUST still get its [notes](../../GLOSSARY.md#permanent-note) delivered once, and each
lesson MUST still become one note. `engram update` MUST NOT install anything into
[Pi](../../GLOSSARY.md#harness).

#### 793a2a1 UAT Scenarios

```gherkin
Scenario: 793a2a1a The next session has memory after one engram update
  Given a machine with Claude Code on which the user has never run engram update, and a vault holding a note that says when the staging database is reset
  When the user runs engram update
  And starts a Claude Code session in a project other than the one engram update was run in, and sends a prompt asking when the staging database is reset
  Then the note is in the agent's context before the agent answers

Scenario: 793a2a1b A second update still delivers each note once
  Given a machine on which the user has run engram update once, and a vault holding a note that says when the staging database is reset
  When the user runs engram update again
  And starts a session and sends a prompt asking when the staging database is reset
  Then the note is in the agent's context once

Scenario: 793a2a1c A dry run turns nothing on
  Given a machine with Claude Code on which the user has never run engram update, and a vault holding a note that says when the staging database is reset
  When the user runs engram update --dry-run
  And starts a session and sends a prompt asking when the staging database is reset
  Then the report lists the automatic memory as planned, and the agent's context holds no note

Scenario: 793a2a1d Pi gets no automatic memory
  Given a machine with both Claude Code and Pi
  When the user runs engram update
  Then the report lists the automatic memory under Claude Code and not under Pi, and nothing of it is installed under ~/.pi

Scenario: 793a2a1e A vault that does not exist is not created
  Given a machine with Claude Code on which the user has run engram update, and whose configured vault no longer exists
  When the user starts a session and sends a prompt
  Then the agent answers the prompt, and the vault still does not exist
```

The clause of [793a2a On after `engram update`](#793a2a-on-after-engram-update) about lessons is proved by a scenario written with [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent) ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a2b Memory arrives with the prompt

Requirement: 793a2b When the user sends a [prompt](#793a1a-terms), [automatic memory](#793a1a-terms) MUST put in
the [agent](#793a1a-terms)'s context, before the agent acts on the prompt, the
[notes](../../GLOSSARY.md#permanent-note) that [bear on it](#793a1a-terms), less those
it already has, without the user or the agent asking for them. The agent already has a note when automatic
memory delivered it earlier in the same [session](../../GLOSSARY.md#session), or the agent looked it up in that
session; automatic memory MUST NOT put such a note in the agent's context again. Only notes are delivered: the
conversation of
[793a2f The conversation becomes searchable memory](#793a2f-the-conversation-becomes-searchable-memory) is found
when the agent asks. The notes arrive beside the prompt, which stays exactly as the user typed it
([793a2d Never in the way](#793a2d-never-in-the-way)).

#### 793a2b1 UAT Scenarios

```gherkin
Scenario: 793a2b1a The notes that bear on a prompt arrive with it
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset and a note about formatting Python files
  When the user sends a prompt asking when the staging database is reset
  Then before the agent acts its context holds the staging note and not the formatting note, and the prompt in the conversation is exactly as the user typed it

Scenario: 793a2b1b A note delivered earlier in the session is not delivered again
  Given a session in which a prompt asking when the staging database is reset arrived with the staging note
  When the user sends another prompt about the staging database reset
  Then the second prompt arrives with nothing from the vault, and the staging note is in the agent's context once

Scenario: 793a2b1c A note the agent looked up itself is not delivered
  Given a session in which, before the user ran engram update, the agent looked up the staging note itself with engram query, and a vault that has gained another note about the staging database since
  When the user runs engram update, resumes that session, and sends a prompt asking when the staging database is reset
  Then the prompt arrives with the other note and not the staging note

Scenario: 793a2b1d A prompt that matches no note arrives with nothing
  Given the user has run engram update, a vault whose notes have nothing to do with the prompt, and a searchable conversation that does
  When the user sends the prompt
  Then the prompt arrives with nothing from the vault, and the agent answers it
```

### 793a2c Lessons are written without the agent

Requirement: 793a2c When a [turn](#793a1a-terms) holds a confirmed lesson that the
[vault](../../GLOSSARY.md#vault) does not already state
([793a3c Nothing the vault already says is written again](#793a3c-nothing-the-vault-already-says-is-written-again)),
a [note](../../GLOSSARY.md#permanent-note) stating it MUST be in the vault within 2 minutes
([793c2 How soon after a turn must automatic memory have finished writing?](./793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing)) after the turn ends, written by
[automatic memory](#793a1a-terms) and not by the [agent](#793a1a-terms), and marked as written automatically.
A turn holds a confirmed lesson when the user corrected the agent, asked it to remember something, or
confirmed that an approach worked, when the agent plainly corrected its own earlier claim, or when an
uncertain approach succeeded with evidence in the turn. A turn that holds only guesses, plans or chatter MUST
NOT produce a note. A note MUST NOT contain a credential, token, key or password that appeared in the turn.

#### 793a2c1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a2f The conversation becomes searchable memory

Requirement: 793a2f Within 2 minutes ([793c2 How soon after a turn must automatic memory have finished writing?](./793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing)) after a
[turn](#793a1a-terms) ends, [automatic memory](#793a1a-terms) MUST make the
[session](../../GLOSSARY.md#session)'s conversation so far findable by the [agent](#793a1a-terms)'s own memory
lookup (`engram query`), without the user or the agent running
[`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand).

#### 793a2f1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a2d Never in the way

Requirement: 793a2d [Automatic memory](#793a1a-terms) MUST NOT hold a [prompt](#793a1a-terms) for longer than
8 seconds ([793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held)), MUST NOT edit the user's prompt or the text of the
[agent](#793a1a-terms)'s reply, MUST NOT break or stall a [session](../../GLOSSARY.md#session), and MUST NOT
deliver to or write from its own sessions, the sessions it starts itself to do its work. When it cannot do its
work, for any reason, including a delivery that does not finish within those 8 seconds and a write that fails,
it MUST step aside, and the session MUST continue as if automatic memory were not installed.

#### 793a2d1 UAT Scenarios

```gherkin
Scenario: 793a2d1a A slow lookup does not hold a prompt for more than 8 seconds
  Given the user has run engram update, and a lookup that takes 20 seconds
  When the user sends a prompt
  Then the prompt reaches the agent within 8 seconds, as the user typed it and with nothing from the vault, and the agent answers it
```

The scenario of the clause of [793a2d Never in the way](#793a2d-never-in-the-way) about its own [sessions](../../GLOSSARY.md#session) is written with [793e6b Lessons are written without the agent](./793e_implementation-plan.md#793e6b-lessons-are-written-without-the-agent), the first group that starts such sessions, and the other scenarios of [793a2d Never in the way](#793a2d-never-in-the-way) are written with [793e6c Never in the way](./793e_implementation-plan.md#793e6c-never-in-the-way) ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a2e The user can see what it did

Requirement: 793a2e When [automatic memory](#793a1a-terms) delivered a [note](../../GLOSSARY.md#permanent-note),
wrote a note, or could not do a delivery or a write, the user MUST be able to see in the
[session](../../GLOSSARY.md#session), by the time the user sends the next [prompt](#793a1a-terms), what it
delivered for the previous prompt, what it wrote from the previous [turn](#793a1a-terms), and each delivery or
write it could not do, with the reason, without going to the [vault](../../GLOSSARY.md#vault). A write that
finishes after the next prompt is sent MUST be shown when it finishes.

#### 793a2e1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

## 793a3 Added: Interactions

### 793a3a What is written is delivered later

Requirement: 793a3a A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](#793a1a-terms) writes
under [793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent) MUST be put
in the [agent](#793a1a-terms)'s context under
[793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt) with
[prompts](#793a1a-terms) sent after the note is written, in the same [session](../../GLOSSARY.md#session) and in
later sessions, whenever it bears on them.

#### 793a3a1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a3c Nothing the vault already says is written again

Requirement: 793a3c [Automatic memory](#793a1a-terms) MUST NOT write a [note](../../GLOSSARY.md#permanent-note)
for a lesson that a note in the [vault](../../GLOSSARY.md#vault) already states, whoever wrote that note and
whenever it was written.

#### 793a3c1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a3e Explicit memory use keeps working

Requirement: 793a3e The [agent](#793a1a-terms)'s own explicit use of memory, such as `engram query`,
[`engram learn`](../../GLOSSARY.md#engram-learn-subcommand), the [recall skill](../../GLOSSARY.md#recall-skill)
and the [learn skill](../../GLOSSARY.md#learn-skill), MUST keep working unchanged alongside
[automatic memory](#793a1a-terms).

#### 793a3e1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

## 793a5 Modified: First-use vault creation

### 793a5a Modified: 792a2e First-use vault creation

Target: [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation)

Requirement: 792a2e Every command that resolves a vault path, other than the commands that
[automatic memory](#793a1a-terms) runs on its own, MUST, when no [vault](../../GLOSSARY.md#vault) exists at that path,
create one before the command runs, whether or not `ENGRAM_PARENT` is set, and print exactly one stderr line,
`engram: created new vault at <path>`. The new vault contains `.obsidian/app.json`, a `.gitignore`, a starter
`README.md`, a vault ID ([792a2h Stable vault ID](../../specs/product/vault-identity.md#792a2h-stable-vault-id)) and the `.engram/` state directory
([792a2k Exchange state outside tracked files](../../specs/product/vault-identity.md#792a2k-exchange-state-outside-tracked-files)). A command that automatic memory runs on its own creates no vault
([793a2a On after `engram update`](#793a2a-on-after-engram-update)).

Rationale: a [prompt](#793a1a-terms) must not create a vault the user has not asked for, and a missing vault is the user's to create
with a command they run.

The scenario of this operation is [793a2a1e A vault that does not exist is not created](#793a2a1-uat-scenarios).

