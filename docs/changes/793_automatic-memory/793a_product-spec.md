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
consulted or growing, and cannot tell when it was not.

The needed outcome is the North Star: [automatic memory](#793a1a-terms). In every Claude Code session on the
user's machine the vault is consulted and grows whether or not the agent thinks to, without getting in the
user's way, and the user can see what it did.

Both operations below are Added. Any existing canonical node a requirement changes is found and recorded as a
Modified or Removed operation with that requirement's scenarios
([793f5 How this change applies it](./793f_algorithm.md#793f5-how-this-change-applies-it)). Scenarios are
written depth first, so each requirement's scenario section stays pending until the requirement is reached in
[793e Automatic Memory - Implementation Plan](./793e_implementation-plan.md#automatic-memory---implementation-plan).

### 793a1a Terms

- **Agent:** the AI the user works with in a Claude Code [session](../../GLOSSARY.md#session). It is not a
  [subagent](../../GLOSSARY.md#subagent).
- **Prompt:** a message the user sends to the [agent](#793a1a-terms).
- **Turn:** one [prompt](#793a1a-terms) and everything the agent does in answer to it, until it hands control
  back to the user.
- **Automatic memory:** what this change adds, done without the user or the agent asking: it puts the
  [notes](../../GLOSSARY.md#permanent-note) that bear on a prompt in the agent's context
  ([793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt)) and writes notes for
  lessons ([793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent)).

## 793a2 Added: Automatic memory

### 793a2a On after `engram update`

Requirement: 793a2a When the user runs [`engram update`](../../GLOSSARY.md#engram-update) on a machine that has
Claude Code, [automatic memory](#793a1a-terms) MUST be on for every Claude Code
[session](../../GLOSSARY.md#session) the user starts on that machine afterwards, in any project. Turning it on
MUST NOT need any step other than running `engram update`: no new credential or configuration for the user to
supply. Running `engram update` again MUST leave automatic memory exactly as the first run left it. A session
in any other [harness](../../GLOSSARY.md#harness) MUST behave exactly as before.

#### 793a2a1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a2b Memory arrives with the prompt

Requirement: 793a2b When the user sends a [prompt](#793a1a-terms), [automatic memory](#793a1a-terms) MUST put in
the [agent](#793a1a-terms)'s context, before the agent acts on the prompt, the
[notes](../../GLOSSARY.md#permanent-note) in the [vault](../../GLOSSARY.md#vault) that bear on it and that the
agent does not already have
([793a3b Nothing the agent already has is delivered again](#793a3b-nothing-the-agent-already-has-is-delivered-again)),
without the user or the agent asking for them. The notes arrive beside the prompt, which stays exactly as the
user typed it ([793a2d Never in the way](#793a2d-never-in-the-way)).

#### 793a2b1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a2c Lessons are written without the agent

Requirement: 793a2c When a [turn](#793a1a-terms) holds a confirmed lesson that the
[vault](../../GLOSSARY.md#vault) does not already state
([793a3c Nothing the vault already says is written again](#793a3c-nothing-the-vault-already-says-is-written-again)),
a [note](../../GLOSSARY.md#permanent-note) stating it MUST be in the vault within 2 minutes
([793c2 How soon after a turn must automatic memory have finished writing?](./793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing)) after the turn ends, written by
[automatic memory](#793a1a-terms) and not by the [agent](#793a1a-terms), and marked as written automatically.
A turn holds a confirmed
lesson when the user corrected the agent, asked it to remember something, or confirmed that an approach
worked, or when the agent plainly corrected its own earlier claim. A turn that holds only guesses, plans or
chatter MUST NOT produce a note.

#### 793a2c1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a2d Never in the way

Requirement: 793a2d [Automatic memory](#793a1a-terms) MUST NOT hold a [prompt](#793a1a-terms) for longer than
8 seconds ([793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held)), MUST NOT change what the user typed or what the [agent](#793a1a-terms)
answered, and MUST NOT break or stall a [session](../../GLOSSARY.md#session). When it cannot do its work, for
any reason, it MUST step aside, and the session MUST continue as if automatic memory were not installed.

#### 793a2d1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a2e The user can see what it did

Requirement: 793a2e After each [turn](#793a1a-terms) on which [automatic memory](#793a1a-terms) delivered a
[note](../../GLOSSARY.md#permanent-note), wrote a note or skipped a step, it MUST show the user, in the
[session](../../GLOSSARY.md#session) and in one line, each note it delivered for the turn's
[prompt](#793a1a-terms), each note it wrote for the turn, and each step it skipped with the reason, so the
user can see its work without going to the [vault](../../GLOSSARY.md#vault). The line MUST appear once
automatic memory has finished its work for the turn.

#### 793a2e1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

## 793a3 Added: Interactions

### 793a3a What is written is delivered later

Requirement: 793a3a A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](#793a1a-terms) writes
under [793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent) MUST be put
in the [agent](#793a1a-terms)'s context under
[793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt) with later
[prompts](#793a1a-terms), in the same [session](../../GLOSSARY.md#session) and in later sessions, whenever it
bears on them.

#### 793a3a1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a3b Nothing the agent already has is delivered again

Requirement: 793a3b [Automatic memory](#793a1a-terms) MUST NOT put a [note](../../GLOSSARY.md#permanent-note) in
the [agent](#793a1a-terms)'s context when the agent already has it in the
[session](../../GLOSSARY.md#session), whether automatic memory delivered it earlier or the agent looked it up
itself.

#### 793a3b1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a3c Nothing the vault already says is written again

Requirement: 793a3c [Automatic memory](#793a1a-terms) MUST NOT write a [note](../../GLOSSARY.md#permanent-note)
for a lesson that a note in the [vault](../../GLOSSARY.md#vault) already states, whoever wrote that note and
whenever it was written.

#### 793a3c1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a3d Failures never break a prompt and always show

Requirement: 793a3d A delivery under
[793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt) that does not finish within
8 seconds ([793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held)), and any other failure of that delivery or of the writing under
[793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent), MUST be treated
under [793a2d Never in the way](#793a2d-never-in-the-way) as [automatic memory](#793a1a-terms) being unable to
do its work, and MUST be shown as a skipped step in the line of
[793a2e The user can see what it did](#793a2e-the-user-can-see-what-it-did).

#### 793a3d1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a3e Explicit memory use keeps working

Requirement: 793a3e The [agent](#793a1a-terms)'s own explicit use of memory, such as `engram query`,
[`engram learn`](../../GLOSSARY.md#engram-learn-subcommand), the [recall skill](../../GLOSSARY.md#recall-skill)
and the [learn skill](../../GLOSSARY.md#learn-skill), MUST keep working unchanged alongside
[automatic memory](#793a1a-terms).

#### 793a3e1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

### 793a3f Everything runs on the existing login and vault

Requirement: 793a3f [Automatic memory](#793a1a-terms) MUST use the user's existing Claude Code login and the
user's existing [vault](../../GLOSSARY.md#vault), and no other account, key or vault.

#### 793a3f1 UAT Scenarios

Pending step 2 ([793f3 Depth first](./793f_algorithm.md#793f3-depth-first)).

## 793a4 Not in the minimal set

Each capability below is left out because it is not needed to deliver the North Star. Any of them can be a
separate change later ([793b8 Exclusions](./793b_decision-log.md#793b8-exclusions)).

- Memory at [subagent](../../GLOSSARY.md#subagent) start and capture at subagent stop: the North Star is met in
  the user's own [sessions](../../GLOSSARY.md#session), and a subagent works for the
  [agent](#793a1a-terms), not for the user.
- Memory at route results: a [prompt](#793a1a-terms) already brings the
  [notes](../../GLOSSARY.md#permanent-note) that bear on it, and a route result is not a prompt.
- The chunk-index deep lookup and the [transcript](../../GLOSSARY.md#transcript) sweep
  ([`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand)): notes alone carry what the
  [vault](../../GLOSSARY.md#vault) has learned.
- Backlog capture of old transcripts: the lessons of new [turns](#793a1a-terms) are written by
  [793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent), and old ones
  can be mined later.
- Curation of [pending offers](../../GLOSSARY.md#offer): lessons are written straight into the vault
  ([793c5 Are lessons written straight into the vault or held for review?](./793c_open-questions.md#793c5-are-lessons-written-straight-into-the-vault-or-held-for-review)).
- A model-made relevance judgement before delivery: the agent judges what it is given, as it did when it ran
  the query itself.
- Reinforcing an already-covered note: a lesson the vault already states is simply not written again
  ([793a3c Nothing the vault already says is written again](#793a3c-nothing-the-vault-already-says-is-written-again)).
- A status line: the one line after a turn
  ([793a2e The user can see what it did](#793a2e-the-user-can-see-what-it-did)) already shows what happened.
- The Pi [harness](../../GLOSSARY.md#harness): sessions in other harnesses stay as before
  ([793a2a On after `engram update`](#793a2a-on-after-engram-update)).
