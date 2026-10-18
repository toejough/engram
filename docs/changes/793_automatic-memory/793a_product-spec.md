# Automatic Memory - Product Spec

> **North Star:** I want to add automatic learning and memory retrieval.
>
> The owner, Joe, 2026-10-16 ([793b2 North Star](./793b_decision-log.md#793b2-north-star)).

## 793a1 Problem Statement

Anyone who works with an [agent](#793a1a-terms) in Claude Code and keeps a [vault](../../GLOSSARY.md#vault) gets value from it only when the agent consults it before acting and saves what a [session](../../GLOSSARY.md#session) taught, and today both depend on the agent remembering to. Issue #793 measured that the agent saves about one moment in twenty worth saving, and that the instruction to look first was refused in 6 of 6 headless trials. The user cannot rely on the vault being consulted or growing, and once the work leaves the agent the user no longer sees it. The needed outcome is the North Star.

### 793a1a Terms

- **Agent:** the AI the user works with in a Claude Code [session](../../GLOSSARY.md#session), not a [subagent](../../GLOSSARY.md#subagent).
- **Prompt:** a message the user sends to the [agent](#793a1a-terms).
- **Turn:** a [prompt](#793a1a-terms) and everything the agent does in answer, until it hands control back to the user.
- **Bears on a prompt:** a [note](../../GLOSSARY.md#permanent-note) bears on a prompt when `engram query`, run with the prompt's text as its message and only phrase, returns it as a match for that text: not as a fragment of a conversation, not as one of the extra notes the command samples to widen what it returns. The user can run that command and read the notes.
- **Automatic memory:** what this change adds, done without the user or the agent asking: notes that bear on a prompt arrive in the agent's context ([793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt)), lessons are written as notes ([793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent)) and the conversation becomes searchable ([793a2f The conversation becomes searchable memory](#793a2f-the-conversation-becomes-searchable-memory)).
- **Sweep:** the indexing of a [session](../../GLOSSARY.md#session)'s conversation that makes it findable by `engram query` ([793a2f The conversation becomes searchable memory](#793a2f-the-conversation-becomes-searchable-memory)).

## 793a2 Added: Automatic memory

### 793a2a On after `engram update`

Requirement: 793a2a When the user runs [`engram update`](../../GLOSSARY.md#engram-update), other than with `--dry-run` ([792a11f Previewing an update](../../specs/product/install-and-update.md#792a11f-previewing-an-update)), on a machine that has Claude Code, [automatic memory](#793a1a-terms) MUST be on for every Claude Code [session](../../GLOSSARY.md#session) the user starts on that machine afterwards, in any project. Turning it on MUST NOT need a step other than `engram update`, and automatic memory MUST NOT use an account, key or [vault](../../GLOSSARY.md#vault) the user has not already configured. After a second `engram update`, each [prompt](#793a1a-terms) MUST still get its [notes](../../GLOSSARY.md#permanent-note) delivered once and each lesson MUST still become one note. `engram update` MUST NOT install anything into Pi.

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

Scenario: 793a2a1f A second update still writes each lesson as one note
  Given a machine on which the user has run engram update once
  When the user runs engram update again, and corrects the agent in a prompt of a session
  Then the vault holds one note for the correction

Scenario: 793a2a1g A lesson is not written to a vault that does not exist
  Given a machine with Claude Code on which the user has run engram update, and whose configured vault no longer exists
  When the user corrects the agent in a prompt of a session, and the turn ends
  Then the vault still does not exist 2 minutes after the turn ends
```

### 793a2b Memory arrives with the prompt

Requirement: 793a2b When the user sends a [prompt](#793a1a-terms), [automatic memory](#793a1a-terms) MUST put the [notes](../../GLOSSARY.md#permanent-note) that [bear on it](#793a1a-terms), less those the [agent](#793a1a-terms) already has, in the agent's context before the agent acts, unasked. The agent has a note when automatic memory delivered it earlier in the same [session](../../GLOSSARY.md#session) or the agent looked it up in that session; such a note MUST NOT be put there again. Only notes are delivered; the conversation ([793a2f The conversation becomes searchable memory](#793a2f-the-conversation-becomes-searchable-memory)) is found when the agent asks. The notes arrive beside the prompt, which stays exactly as the user typed it ([793a2d Never in the way](#793a2d-never-in-the-way)).

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

Requirement: 793a2c When a [turn](#793a1a-terms) holds a confirmed lesson that the [vault](../../GLOSSARY.md#vault) does not already state ([793a3c Nothing the vault already says is written again](#793a3c-nothing-the-vault-already-says-is-written-again)), a [note](../../GLOSSARY.md#permanent-note) stating it MUST be in the vault within 2 minutes ([793c2 How soon after a turn must automatic memory have finished writing?](./793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing)) after the turn ends, while the [session](../../GLOSSARY.md#session) stays open for those 2 minutes ([793c14 Is a turn that is followed by the end of the session covered?](./793c_open-questions.md#793c14-is-a-turn-that-is-followed-by-the-end-of-the-session-covered)), written by [automatic memory](#793a1a-terms) and not by the [agent](#793a1a-terms), and marked as written automatically. A turn holds a confirmed lesson when the user corrected the agent, asked it to remember something (even if the agent says it did not save it) or confirmed that an approach worked, when the agent plainly corrected its own earlier claim, or when an uncertain approach succeeded with evidence in the turn. A turn that holds only guesses, plans or chatter MUST NOT produce a note; a thanks or "it worked" that names no approach is chatter. A note MUST NOT contain a credential, token, key or password that appeared in the turn in a recognised form ([793c49 Should the promise that a note holds no credential stay absolute?](./793c_open-questions.md#793c49-should-the-promise-that-a-note-holds-no-credential-stay-absolute)): a private key block; a key or token that opens with the name of the service that issued it (GitHub, npm, GitLab, Slack, Google or AWS); a signed token of three parts joined by dots; a `Bearer` or `Basic` credential of 12 characters or more; the user and password of a web address; the value of an `Authorization` header; the value a name holding key, token, secret, password, credential or private key is given by `=`, `:` or `is`, with up to six words of the sentence between the name and `:` or `is`; or a run of 32 or more letters and digits. A credential in any other form, such as a password a sentence gives without `is` or `:`, or a token split over two lines, is not promised away.

#### 793a2c1 UAT Scenarios

```gherkin
Scenario: 793a2c1a A correction becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands and has answered a question about how to build a project
  When the user corrects the agent's answer in a prompt, and the turn ends
  Then within 2 minutes the vault holds a note that states the correction and says it was written automatically, and the agent wrote nothing to the vault

Scenario: 793a2c1b A request to remember becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember something, and the turn ends
  Then within 2 minutes the vault holds a note that states it and says it was written automatically, and the agent wrote nothing to the vault

Scenario: 793a2c1c An approach the user confirmed becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands and has done a task one way
  When the user tells the agent in a prompt that the way worked and to keep doing it, and the turn ends
  Then within 2 minutes the vault holds a note that states the approach and says it was written automatically

Scenario: 793a2c1d The agent's own correction becomes a note
  Given the user has run engram update, and a session in which the agent may read files but may not run engram commands
  When the user asks the agent in a prompt to guess what a file says and then to read it, and the agent says its guess was wrong and states what the file says, and the turn ends
  Then within 2 minutes the vault holds a note that states what the file says and says it was written automatically

Scenario: 793a2c1e An approach that succeeded with evidence becomes a note
  Given the user has run engram update, and a session in which the agent may run a check script but may not run engram commands
  When the agent, unsure whether an approach will make the check pass, tries it in a turn and the turn shows that the check passed
  Then within 2 minutes the vault holds a note that states the approach and says it was written automatically

Scenario: 793a2c1f Guesses, plans and chatter produce no note
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user sends a prompt that the agent answers with a guess, a plan and chatter, and the turn ends
  Then 2 minutes after the turn ends the vault holds no note

Scenario: 793a2c1g A credential in the turn is not in the note
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember how to deploy, with the token and the password that the deploy needs, each in a recognised form, and the turn ends
  Then within 2 minutes the vault holds a note that states how to deploy, and no note holds the token or the password

Scenario: 793a2c1h A thanks that names no approach produces no note
  Given the user has run engram update, and a session in which the agent may not run engram commands and has done a task one way
  When the user thanks the agent in a prompt, saying only that it worked, and the turn ends
  Then 2 minutes after the turn ends the vault holds no note

Scenario: 793a2c1i A request to remember is written when the agent says it did not save it
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember something and not to save it itself, the agent says that it did not save it, and the turn ends
  Then within 2 minutes the vault holds a note that states it and says it was written automatically
```

### 793a2f The conversation becomes searchable memory

Requirement: 793a2f Within 2 minutes ([793c2 How soon after a turn must automatic memory have finished writing?](./793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing)) after a [turn](#793a1a-terms) ends, while the [session](../../GLOSSARY.md#session) stays open for those 2 minutes ([793c14 Is a turn that is followed by the end of the session covered?](./793c_open-questions.md#793c14-is-a-turn-that-is-followed-by-the-end-of-the-session-covered)), [automatic memory](#793a1a-terms) MUST make the session's conversation so far findable by the [agent](#793a1a-terms)'s own memory lookup (`engram query`), without the user or the agent running [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand).

#### 793a2f1 UAT Scenarios

```gherkin
Scenario: 793a2f1a The conversation so far is findable after a turn
  Given the user has run engram update, and a session in which the agent may not run engram commands and the user has told the agent a first thing that only this conversation holds
  When the user tells it a second such thing in a prompt, and the turn ends
  Then within 2 minutes engram query returns a chunk of the conversation for each of the two things, and neither the user nor the agent ran engram ingest
```

### 793a2d Never in the way

Requirement: 793a2d [Automatic memory](#793a1a-terms) MUST NOT hold a [prompt](#793a1a-terms) for longer than 8 seconds ([793c1 How long may a prompt be held?](./793c_open-questions.md#793c1-how-long-may-a-prompt-be-held)), MUST NOT edit the user's prompt or the text of the [agent](#793a1a-terms)'s reply, MUST NOT break or stall a [session](../../GLOSSARY.md#session), and MUST NOT deliver to or write from its own sessions, the sessions it starts itself to do its work. When it cannot do its work, for any reason, including a delivery that does not finish within those 8 seconds and a write that fails, it MUST step aside, and the session MUST continue as if automatic memory were not installed.

#### 793a2d1 UAT Scenarios

```gherkin
Scenario: 793a2d1a A slow lookup does not hold a prompt for more than 8 seconds
  Given the user has run engram update, and a lookup that takes 20 seconds
  When the user sends a prompt
  Then the prompt reaches the agent within 8 seconds, as the user typed it and with nothing from the vault, and the agent answers it

Scenario: 793a2d1b Automatic memory's work starts no session of its own
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user corrects the agent in a prompt, and the turn ends
  Then within 2 minutes the vault holds a note for the correction, and no session but the user's exists, so nothing was delivered to or written from another

Scenario: 793a2d1c The work after a turn does not hold the next prompt
  Given the user has run engram update, and a judgement, a write and a sweep that each take 60 seconds
  When the user sends a prompt, and then another while that work goes on
  Then the second prompt reaches the agent within 8 seconds, as the user typed it, and the agent answers it

Scenario: 793a2d1d The prompt and the reply are as they were
  Given the user has run engram update, and a session in which the user's correction became a note
  When the user sends a prompt that the note bears on, typed with extra spaces and a line break
  Then the prompt in the conversation is exactly as the user typed it, and the reply is exactly the agent's text

Scenario: 793a2d1e A write that fails does not break the session
  Given the user has run engram update, and a vault that cannot be written
  When the user corrects the agent in a prompt, the turn ends, and the user sends another prompt
  Then the agent answers the second prompt as typed, with nothing about the failed write in its context, and the vault holds no new note
```

### 793a2e The user can see what it did

Requirement: 793a2e When [automatic memory](#793a1a-terms) delivered a [note](../../GLOSSARY.md#permanent-note), wrote one or could not do a delivery, a write or a [sweep](#793a1a-terms), the user MUST be able to see in the [session](../../GLOSSARY.md#session), by the time the user sends the next [prompt](#793a1a-terms), without going to the [vault](../../GLOSSARY.md#vault), what it delivered for the previous prompt, what it wrote from the previous [turn](#793a1a-terms), and each delivery, write or sweep it could not do, with the reason. What a write did, and what a write or a sweep could not do, when it finishes after that prompt is sent, MUST be shown when it finishes.

#### 793a2e1 UAT Scenarios

```gherkin
Scenario: 793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a session in which the user asked in one prompt when it is reset and to remember that it is backed up every Monday at 05:00 UTC
  When the user sends the next prompt
  Then the session shows a line that names the staging note delivered with the earlier prompt, and, with the next prompt, a line that names the note written from it

Scenario: 793a2e1b A write that finishes after the next prompt is shown when it finishes
  Given the user has run engram update, and a write that takes 12 seconds
  When the user corrects the agent in a prompt, sends another before the write finishes, and a third after it finished
  Then the session shows no line about the write with the second prompt, and with the third a line that names the note written

Scenario: 793a2e1c A delivery that could not be done is shown with its reason
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a lookup that takes 20 seconds
  When the user sends a prompt asking when the staging database is reset, the lookup becomes quick again, and the user sends another prompt
  Then the session shows, with the second prompt, a line that says the delivery for the first could not be done, with the reason

Scenario: 793a2e1d A write that failed is shown with its reason
  Given the user has run engram update, and a vault that cannot be written
  When the user corrects the agent in a prompt, the turn ends, and the user sends another prompt
  Then the session shows, with the second prompt, a line that says the write failed, with the reason

Scenario: 793a2e1e A sweep that failed is shown with its reason
  Given the user has run engram update, and a chunk index that cannot be written
  When the user says something in a prompt, the turn ends, and the user sends another prompt
  Then the session shows, with the second prompt, a line that says the sweep failed, with the reason

Scenario: 793a2e1f A turn that is not captured is shown with the reason
  Given the user has run engram update, and headless prompts are captured only when the user asks
  When the user corrects the agent in a headless session, and sends another prompt
  Then the session shows, with the second prompt, a line that says the turn was not captured, with the reason
```

## 793a3 Added: Interactions

### 793a3a What is written is delivered later

Requirement: 793a3a A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](#793a1a-terms) writes ([793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent)) MUST be put in the [agent](#793a1a-terms)'s context ([793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt)) with every [prompt](#793a1a-terms) sent after it is written, in the same [session](../../GLOSSARY.md#session) and in later sessions, that it bears on.

#### 793a3a1 UAT Scenarios

```gherkin
Scenario: 793a3a1a A written note is delivered with later prompts
  Given the user has run engram update, and a session in which the user corrected the agent and the correction became a note
  When the user sends a prompt that the note bears on, in that session and then in a new session
  Then each of the two prompts arrives with the note
```

### 793a3c Nothing the vault already says is written again

Requirement: 793a3c [Automatic memory](#793a1a-terms) MUST NOT write a [note](../../GLOSSARY.md#permanent-note) for a lesson that a note in the [vault](../../GLOSSARY.md#vault) already states, whoever wrote that note and whenever. A lesson that no note states is written even when a note states another lesson about the same subject.

#### 793a3c1 UAT Scenarios

```gherkin
Scenario: 793a3c1a A lesson an earlier note states is not written again
  Given the user has run engram update, a vault holding a note that the user saved earlier and that states a lesson, and a session in which the agent may not run engram commands
  When the user states the same lesson to the agent in a prompt, and the turn ends
  Then 2 minutes after the turn ends the vault holds the earlier note only

Scenario: 793a3c1b A lesson the agent saved itself is not written again
  Given the user has run engram update, and a session in which the agent may run engram commands
  When the user asks the agent in a prompt to remember something and to save it with engram learn, and the turn ends
  Then 2 minutes after the turn ends the vault holds one note for it, the one the agent saved

Scenario: 793a3c1c A lesson no earlier note states is written
  Given the user has run engram update, a vault holding a note that the user saved earlier and that states a lesson, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember a different lesson about the same subject, and the turn ends
  Then within 2 minutes the vault holds the earlier note, unchanged, and a new note for the different lesson that says it was written automatically
```

### 793a3e Explicit memory use keeps working

Requirement: 793a3e The [agent](#793a1a-terms)'s own explicit use of memory, such as `engram query`, [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand), the [recall skill](../../GLOSSARY.md#recall-skill) and the [learn skill](../../GLOSSARY.md#learn-skill), MUST keep working unchanged alongside [automatic memory](#793a1a-terms).

#### 793a3e1 UAT Scenarios

```gherkin
Scenario: 793a3e1a The agent's own lookups and saves keep working
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a session in which the agent may run engram commands
  When the user asks the agent in a prompt to look the staging note up with engram query and to save another note with engram learn
  Then the agent's lookup shows the staging note, and the vault holds the note the agent saved, as the agent wrote it
```

## 793a5 Modified: First-use vault creation

### 793a5a Modified: 792a2e First-use vault creation

Target: [792a2e First-use vault creation](../../specs/product/vault-identity.md#792a2e-first-use-vault-creation)

Requirement: 792a2e Every command that resolves a vault path, other than the commands that [automatic memory](#793a1a-terms) runs on its own, MUST, when no [vault](../../GLOSSARY.md#vault) exists at that path, create one before the command runs, whether or not `ENGRAM_PARENT` is set, and print exactly one stderr line, `engram: created new vault at <path>`. The new vault contains `.obsidian/app.json`, a `.gitignore`, a starter `README.md`, a vault ID ([792a2h Stable vault ID](../../specs/product/vault-identity.md#792a2h-stable-vault-id)) and the `.engram/` state directory ([792a2k Exchange state outside tracked files](../../specs/product/vault-identity.md#792a2k-exchange-state-outside-tracked-files)). A command that automatic memory runs on its own creates no vault ([793a2a On after `engram update`](#793a2a-on-after-engram-update)).

Rationale: a [prompt](#793a1a-terms) must not create a vault the user has not asked for, and a missing vault is the user's to create with a command they run (scenario [793a2a1e A vault that does not exist is not created](#793a2a1-uat-scenarios)).
