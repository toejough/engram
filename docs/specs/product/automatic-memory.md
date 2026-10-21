# Automatic Memory - Product Spec

[Automatic memory](../../GLOSSARY.md#automatic-memory) puts the [notes](../../GLOSSARY.md#permanent-note) and passages that apply to a [recall moment](../../GLOSSARY.md#recall-moment) in the [agent](../../GLOSSARY.md#agent-automatic-memory)'s context, writes the lessons of a [turn](../../GLOSSARY.md#turn-automatic-memory) as notes, and makes the conversation searchable, without the user or the agent asking, once [engram update](../../GLOSSARY.md#engram-update) has turned it on for Claude Code. Its terms are defined in the [glossary](../../GLOSSARY.md#automatic-memory-terms).

## 793a2 Automatic memory

### 793a2a On after `engram update`

Requirement: 793a2a When the user runs [`engram update`](../../GLOSSARY.md#engram-update), other than with `--dry-run` ([792a11f Previewing an update](./install-and-update.md#792a11f-previewing-an-update)), on a machine that has Claude Code, [automatic memory](../../GLOSSARY.md#automatic-memory) MUST be on for every Claude Code [session](../../GLOSSARY.md#session) the user starts on that machine afterwards, in any project. Turning it on MUST NOT need a step other than `engram update`, and automatic memory MUST NOT use an account, key or [vault](../../GLOSSARY.md#vault) the user has not already configured. After a second `engram update`, each [prompt](../../GLOSSARY.md#prompt-automatic-memory) MUST still get its [notes](../../GLOSSARY.md#permanent-note) delivered once and each lesson MUST still become one note. `engram update` MUST NOT install anything into Pi.

#### 793a2a1 UAT Scenarios

```gherkin
Scenario: 793a2a1a The next session has memory after one engram update
  Given a machine with Claude Code on which the user has never run engram update, and a vault holding a note that says when the staging database is reset
  When the user runs engram update
  And starts a Claude Code session in a project other than the one engram update was run in, and sends a prompt to prepare a test run against the staging database tonight
  Then the note is in the agent's context before the agent answers

Scenario: 793a2a1b A second update still delivers each note once
  Given a machine on which the user has run engram update once, and a vault holding a note that says when the staging database is reset
  When the user runs engram update again
  And starts a session and sends a prompt to prepare a test run against the staging database tonight
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

Requirement: 793a2b When the user sends a [prompt](../../GLOSSARY.md#prompt-automatic-memory) that is a [recall moment](../../GLOSSARY.md#recall-moment), [automatic memory](../../GLOSSARY.md#automatic-memory) MUST put the [notes](../../GLOSSARY.md#permanent-note) and the passages that [apply to](../../GLOSSARY.md#applies-to-a-recall-moment) it, less those the [agent](../../GLOSSARY.md#agent-automatic-memory) already has, in the agent's context before the agent acts, unasked. It MUST NOT put anything in the agent's context for a prompt that is not a recall moment, and MUST NOT put a note or passage that does not apply to the prompt because its words match. The agent has a note or passage when automatic memory delivered it earlier in the same [session](../../GLOSSARY.md#session) or the agent looked it up in that session; such a note or passage MUST NOT be put there again. What arrives is notes, which hold what an event meant, and passages of earlier conversations ([793a2f The conversation becomes searchable memory](#793a2f-the-conversation-becomes-searchable-memory)), which hold what happened. They arrive beside the prompt, which stays exactly as the user typed it ([793a2d Never in the way](#793a2d-never-in-the-way)).

#### 793a2b1 UAT Scenarios

```gherkin
Scenario: 793a2b1a The notes that apply to a prompt arrive with it
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset and a note about formatting Python files
  When the user sends a prompt to prepare a test run against the staging database tonight
  Then before the agent acts its context holds the staging note and not the formatting note, and the prompt in the conversation is exactly as the user typed it

Scenario: 793a2b1b A note delivered earlier in the session is not delivered again
  Given a session in which a prompt to prepare a test run against the staging database tonight arrived with the staging note
  When the user sends another prompt to prepare a second test run against the staging database tonight
  Then the second prompt arrives with nothing from the vault, and the staging note is in the agent's context once

Scenario: 793a2b1c A note the agent looked up itself is not delivered
  Given a session in which, before the user ran engram update, the agent looked up the staging note itself with engram query, and a vault that has gained another note since, which says the reset is skipped on public holidays
  When the user runs engram update, resumes that session, and sends a prompt to prepare a test run against the staging database tonight
  Then the prompt arrives with the other note and not the staging note

Scenario: 793a2b1d A prompt that matches no note arrives with nothing
  Given the user has run engram update, a vault whose notes have nothing to do with the prompt, and a searchable conversation that has nothing to do with it either
  When the user sends the prompt
  Then the prompt arrives with nothing from the vault, and the agent answers it
```

#### 855a3b1 UAT Scenarios

```gherkin
Scenario: 855a3b1a A prompt that is not a recall moment arrives with nothing
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset
  When the user sends a prompt that says only thanks, that settles when the staging database is reset
  Then the prompt arrives with nothing from the vault, and the agent answers it

Scenario: 855a3b1b A note whose words match but that does not apply is not delivered
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset
  When the user asks the agent in a prompt to write a function that parses the staging database connection string from a config file
  Then the prompt arrives with nothing from the vault, and the agent answers it

Scenario: 855a3b1c Of two notes that conflict, only the newer is delivered
  Given the user has run engram update, and a vault holding an older note that says the staging database is reset every Sunday night and a newer note, linked to it by nothing, that says it is reset every Monday night and not on Sunday
  When the user asks the agent in a prompt to prepare a test run against the staging database tonight
  Then before the agent acts its context holds the newer note and not the older

Scenario: 855a3b1d The notes that arrive are the notes the recall skill draws on
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset, a note that says to take a backup before a database migration, and a note about formatting Python files, of which the first two plainly apply to the prompt below and the third plainly does not
  When the user asks the agent in a prompt to plan a migration of the staging database for tonight, and in a second session on the same vault in which automatic memory is not on asks the agent to run the recall skill in its glance mode for the same prompt and to name the notes it drew on
  Then the notes that arrived with the prompt in the first session are the staging note and the backup note, and the notes that the agent names in the second are the same two

Scenario: 855a3b1e A passage of an earlier conversation arrives when it applies
  Given the user has run engram update, a vault holding no note about the production cutover, and a searchable earlier conversation in which the user said the production cutover is frozen on the last Friday of every month
  When the user asks the agent in a prompt to plan the staging deploys of next month so that they avoid the production cutover freeze
  Then before the agent acts its context holds the passage of the earlier conversation that states the freeze, and the prompt in the conversation is exactly as the user typed it
```

### 793a2c Lessons are written without the agent

Requirement: 793a2c When a [turn](../../GLOSSARY.md#turn-automatic-memory) holds a confirmed lesson that the [vault](../../GLOSSARY.md#vault) does not already state ([793a3c Nothing the vault already says is written again](#793a3c-nothing-the-vault-already-says-is-written-again)), a [note](../../GLOSSARY.md#permanent-note) stating it MUST be in the vault within 2 minutes after the turn ends, while the [session](../../GLOSSARY.md#session) stays open for those 2 minutes, written by [automatic memory](../../GLOSSARY.md#automatic-memory) and not by the [agent](../../GLOSSARY.md#agent-automatic-memory), and marked as written automatically. For a headless session (`claude -p`) this applies only when `ENGRAM_AUTO_CAPTURE_SDK` is set to `1`. A turn holds a confirmed lesson when the user corrected the agent, asked it to remember something durable (even if the agent says it did not save it) or confirmed that an approach worked, when the agent plainly corrected its own earlier claim, or when an uncertain approach succeeded with evidence in the turn. A turn that holds only guesses, plans or chatter MUST NOT produce a note; a thanks or "it worked" that names no approach is chatter. A note MUST NOT contain a [recognised credential](../../GLOSSARY.md#recognised-credential) that appeared in the turn. A credential in any other form, such as a password a sentence gives without `is` or `:`, a token split over two lines, a run of 40 hexadecimal digits, or a run of letters only or of digits only, is not promised away.

#### 793a2c1 UAT Scenarios

```gherkin
Scenario: 793a2c1a A correction becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands and has answered a question about how to build a project
  When the user corrects the agent's answer in a prompt, and the turn ends
  Then within 2 minutes the vault holds a note that states the correction and says it was written automatically, and the agent wrote nothing to the vault

Scenario: 793a2c1b A request to remember becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember something durable, and the turn ends
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

Requirement: 793a2f Within 2 minutes after a [turn](../../GLOSSARY.md#turn-automatic-memory) ends, while the [session](../../GLOSSARY.md#session) stays open for those 2 minutes, [automatic memory](../../GLOSSARY.md#automatic-memory) MUST make the session's conversation so far findable by the [agent](../../GLOSSARY.md#agent-automatic-memory)'s own memory lookup (`engram query`), without the user or the agent running [`engram ingest`](../../GLOSSARY.md#engram-ingest-subcommand). For a headless session (`claude -p`) this applies only when `ENGRAM_AUTO_CAPTURE_SDK` is set to `1`. When the conversation is too long to be made findable within those 2 minutes, [automatic memory](../../GLOSSARY.md#automatic-memory) MUST say so once, in the session, naming the command `engram ingest --auto`, which makes it findable, and MUST NOT try again for the rest of that session.

#### 793a2f1 UAT Scenarios

```gherkin
Scenario: 793a2f1a The conversation so far is findable after a turn
  Given the user has run engram update, and a session in which the agent may not run engram commands and the user has told the agent a first thing that only this conversation holds
  When the user tells it a second such thing in a prompt, and the turn ends
  Then within 2 minutes engram query returns a chunk of the conversation for each of the two things, and neither the user nor the agent ran engram ingest

Scenario: 793a2f1b A conversation too long to index in time is left alone and the user is told once what to run
  Given the user has run engram update, and a conversation that is too long to be indexed within 2 minutes
  When the user sends a prompt, and 2 minutes after its turn ends sends a second, and 2 minutes after the second turn ends sends a third
  Then the session shows, with the second prompt, one line that says the conversation could not be indexed in time and that engram ingest --auto indexes it, and shows no line about it with the third
```

### 793a2d Never in the way

Requirement: 793a2d [Automatic memory](../../GLOSSARY.md#automatic-memory) MUST NOT hold a [prompt](../../GLOSSARY.md#prompt-automatic-memory), or the [agent](../../GLOSSARY.md#agent-automatic-memory) at a [recall moment](../../GLOSSARY.md#recall-moment) within a [turn](../../GLOSSARY.md#turn-automatic-memory), for anything but its own judgement and delivery for that prompt or moment: the work that follows an earlier [turn](../../GLOSSARY.md#turn-automatic-memory), the writing of lessons and the [sweep](../../GLOSSARY.md#sweep), MUST NOT delay it. A judgement or delivery that is only slow MUST NOT make it step aside; the prompt or the agent waits for it. A judgement or delivery that has not answered after 2 minutes has stalled: it MUST step aside, deliver nothing for the prompt and be shown as a failure. The 2 minutes are a liveness bound and not a latency target. It MUST NOT edit the user's prompt or the text of the [agent](../../GLOSSARY.md#agent-automatic-memory)'s reply, MUST NOT break or stall a [session](../../GLOSSARY.md#session), and MUST NOT deliver to or write from its own sessions, the sessions it starts itself to do its work. When it cannot do its work, for any reason, including a judgement or delivery that fails or stalls and a write that fails, it MUST step aside, and the session MUST continue as if automatic memory were not installed.

#### 793a2d1 UAT Scenarios

```gherkin
Scenario: 793a2d1a A slow judgement holds the prompt until it has the notes
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a judgement that takes 20 seconds
  When the user sends a prompt to prepare a test run against the staging database tonight
  Then the prompt reaches the agent after the judgement finished, as the user typed it and with the staging note, and the agent answers it

Scenario: 793a2d1b Automatic memory's work starts no session of its own
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user corrects the agent in a prompt, and the turn ends
  Then within 2 minutes the vault holds a note for the correction, and no session but the user's exists, so nothing was delivered to or written from another

Scenario: 793a2d1c The work after a turn does not hold the next prompt
  Given the user has run engram update, and a moment, a judgement, a placement, a write and a sweep after a turn that each take 60 seconds
  When the user sends a prompt, and then another while that work goes on
  Then the second prompt reaches the agent within 8 seconds, as the user typed it, and the agent answers it within 20 seconds

Scenario: 793a2d1d The prompt and the reply are as they were
  Given the user has run engram update, and a session in which the user's correction became a note
  When the user sends a prompt that is a recall moment and that the note applies to, typed with extra spaces and a line break
  Then the prompt in the conversation is exactly as the user typed it, and the reply is exactly the agent's text

Scenario: 793a2d1e A write that fails does not break the session
  Given the user has run engram update, and a vault that cannot be written
  When the user corrects the agent in a prompt, the turn ends, and the user sends another prompt
  Then the agent answers the second prompt as typed, with nothing about the failed write in its context, and the vault holds no new note
```

#### 855a3c1 UAT Scenarios

```gherkin
Scenario: 855a3c1a A judgement that never answers is stepped aside after 2 minutes
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a judgement that never answers
  When the user sends a prompt to prepare a test run against the staging database tonight
  Then the prompt reaches the agent within 3 minutes, as the user typed it and with nothing from the vault, the agent answers it, and the session shows with the next prompt a line that says the delivery could not be done, with the reason
```

### 793a2e The user can see what it did

Requirement: 793a2e When [automatic memory](../../GLOSSARY.md#automatic-memory) delivered a [note](../../GLOSSARY.md#permanent-note), wrote one, left out the [turn](../../GLOSSARY.md#turn-automatic-memory) of a headless [session](../../GLOSSARY.md#session) ([793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent)) or could not do a delivery, a write or a [sweep](../../GLOSSARY.md#sweep), the user MUST be able to see in the [session](../../GLOSSARY.md#session), by the time the user sends the next [prompt](../../GLOSSARY.md#prompt-automatic-memory), without going to the [vault](../../GLOSSARY.md#vault), what it delivered for the previous prompt, what it wrote from the previous turn, each turn it left out and each delivery, write or sweep it could not do, with the reason. What a write did, and what a write or a sweep could not do, when it finishes after that prompt is sent, MUST be shown by the next prompt after it finishes. At a prompt the session MUST show at most four lines about what automatic memory did: one for the notes delivered, one for the notes written, one for what was left out and one for what failed, each cut where it is long.

#### 793a2e1 UAT Scenarios

```gherkin
Scenario: 793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a session in which the user asked in one prompt to prepare a test run against the staging database tonight and to remember that it is backed up every Monday at 05:00 UTC
  When the user sends the next prompt
  Then the session shows a line that names the staging note delivered with the earlier prompt, and, with the next prompt, a line that names the note written from it

Scenario: 793a2e1b A write that finishes after the next prompt is shown by the prompt after it finishes
  Given the user has run engram update, and a write that takes 12 seconds
  When the user corrects the agent in a prompt, sends another before the write finishes, and a third after it finished
  Then the session shows no line about the write with the second prompt, and with the third a line that names the note written

Scenario: 793a2e1c A delivery that could not be done is shown with its reason
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset, which cannot be read
  When the user sends a prompt to prepare a test run against the staging database tonight, the vault becomes readable again, and the user sends another prompt
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

Scenario: 793a2e1g The notes delivered with a prompt are one line
  Given the user has run engram update, and a vault holding three notes that apply to a prompt about the staging database and a note about formatting Python files
  When the user sends a prompt to prepare a test run against the staging database tonight
  Then the session shows one line that names the three notes, and no other line about the delivery
```

## 793a3 Interactions

### 793a3a What is written is delivered later

Requirement: 793a3a A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../../GLOSSARY.md#automatic-memory) writes ([793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent)) MUST be put in the [agent](../../GLOSSARY.md#agent-automatic-memory)'s context ([793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt)) with every [prompt](../../GLOSSARY.md#prompt-automatic-memory) sent after it is written, in the same [session](../../GLOSSARY.md#session) and in later sessions, that is a [recall moment](../../GLOSSARY.md#recall-moment) and that the note [applies to](../../GLOSSARY.md#applies-to-a-recall-moment).

#### 793a3a1 UAT Scenarios

```gherkin
Scenario: 793a3a1a A written note is delivered with later prompts
  Given the user has run engram update, and a session in which the user corrected the agent and the correction became a note
  When the user sends a prompt that is a recall moment and that the note applies to, in that session and then in a new session
  Then each of the two prompts arrives with the note
```

### 793a3c Nothing the vault already says is written again

Requirement: 793a3c [Automatic memory](../../GLOSSARY.md#automatic-memory) MUST NOT write a [note](../../GLOSSARY.md#permanent-note) for a lesson that a note in the [vault](../../GLOSSARY.md#vault) already states, whoever wrote that note and whenever. A lesson that no note states is written even when a note states another lesson about the same subject.

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

Requirement: 793a3e The [agent](../../GLOSSARY.md#agent-automatic-memory)'s own explicit use of memory, such as `engram query`, [`engram learn`](../../GLOSSARY.md#engram-learn-subcommand), the [recall skill](../../GLOSSARY.md#recall-skill) and the [learn skill](../../GLOSSARY.md#learn-skill), MUST keep working unchanged alongside [automatic memory](../../GLOSSARY.md#automatic-memory).

#### 793a3e1 UAT Scenarios

```gherkin
Scenario: 793a3e1a The agent's own lookups and saves keep working
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a session in which the agent may run engram commands
  When the user asks the agent in a prompt to look the staging note up with engram query and to save another note with engram learn
  Then the agent's lookup shows the staging note, and the vault holds the note the agent saved, as the agent wrote it
```

## 855a2 Judged automatic memory

### 855a2a Delivery writes what its depth writes

Requirement: 855a2a When [automatic memory](../../GLOSSARY.md#automatic-memory) delivers for a [recall moment](../../GLOSSARY.md#recall-moment) of glance depth, it MUST NOT write, amend or crystallize a [note](../../GLOSSARY.md#permanent-note) in the [vault](../../GLOSSARY.md#vault); marking delivered notes as used, and pulling a used parent note down as a pending copy, is not a write. When the moment is of deep depth, it MUST write what the [recall skill](../../GLOSSARY.md#recall-skill)'s deep mode writes, a note crystallized for a lesson that the vault lacks and the amending of a note it draws on, and every note it crystallizes MUST say it was written automatically ([793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent)).

#### 855a2a1 UAT Scenarios

```gherkin
Scenario: 855a2a1a A glance leaves the vault's notes as they were
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset and a note about formatting Python files
  When the user asks the agent in a prompt to prepare a test run against the staging database tonight, and the agent answers
  Then the staging note was in the agent's context, no note was added to the vault, and every note states what it stated before

Scenario: 855a2a1b A deep delivery writes the lesson the vault lacks
  Given the user has run engram update, a vault holding no note about the production cutover, and a searchable earlier conversation in which the user said the production cutover is frozen on the last Friday of every month
  When the user asks the agent in a prompt to plan the irreversible production cutover for the last Friday of next month, and the agent answers
  Then the passage of the earlier conversation was in the agent's context, and the vault holds a note that states the freeze and says it was written automatically
```

### 855a2b Lessons are written as the learn skill writes them

Requirement: 855a2b A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../../GLOSSARY.md#automatic-memory) writes for a lesson ([793a2c Lessons are written without the agent](#793a2c-lessons-are-written-without-the-agent)) MUST state the lesson that the [learn skill](../../GLOSSARY.md#learn-skill) would write for the [turn](../../GLOSSARY.md#turn-automatic-memory), in the form that skill writes it: the general rule or fact and not the story of the turn, the situation in which it applies as a cue that a later task would be described by, and the exact names, commands, flags and values it depends on. When the turn confirms a reusable, ordered, multi-step procedure for a recurring task, the note MUST be a runbook as that skill writes it: the situation in which it applies, the numbered steps with their exact commands, and what is true once the procedure is complete. The question-and-answer pairs that the skill also captures are not written.

#### 855a2b1 UAT Scenarios

```gherkin
Scenario: 855a2b1a A note states the rule and not the story
  Given the user has run engram update, and a session in which the agent may not run engram commands and has force-pushed a branch with git push --force
  When the user tells the agent in a prompt to push with --force-with-lease and never with --force, and the turn ends
  Then within 2 minutes the vault holds a note that states the rule, holds the flag --force-with-lease, names pushing a branch as the situation in which it applies, and does not tell what happened in the turn

Scenario: 855a2b1b The note states the lesson the learn skill writes
  Given the user has run engram update, and two sessions with the same history, in which the agent has force-pushed a branch with git push --force, the first where the agent may not run engram commands and the second where it may and where automatic memory is not on
  When in each the user tells the agent in a prompt to push with --force-with-lease and never with --force, and in the second asks it to run the learn skill, and the turn ends
  Then within 2 minutes the note that automatic memory wrote in the first and the note that the learn skill wrote in the second state the same rule, hold the same flag and name the same situation

Scenario: 855a2b1c A confirmed procedure is written as a runbook
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user tells the agent in a prompt that releasing this project is always: run make ci, then tag with git tag vX.Y.Z, then push the tag with git push origin vX.Y.Z, and the agent says these three commands in that order and runs none of them, and the user confirms it is the way, and the turn ends
  Then within 2 minutes the vault holds a runbook note that names releasing the project as its situation, lists the three commands in that order, and says what is true once they are done
```

### 855a2c Lessons are placed as the learn skill places them

Requirement: 855a2c A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../../GLOSSARY.md#automatic-memory) writes MUST be placed in the [vault](../../GLOSSARY.md#vault)'s order by the ordered test of the [learn skill](../../GLOSSARY.md#learn-skill) ([792a15h Each note is placed by an ordered test](./learn.md#792a15h-each-note-is-placed-by-an-ordered-test)), which considers only notes written, delivered or looked up earlier in the same [session](../../GLOSSARY.md#session). When the note develops one specific sub-point of such a note, its [Luhmann ID](../../GLOSSARY.md#luhmann-id) continues that note's; otherwise, when it extends the same overall thought at the same level, its ID is that note's sibling; otherwise it is placed at the top, even when the vault holds a related note that the session has not seen.

#### 855a2c1 UAT Scenarios

```gherkin
Scenario: 855a2c1a A sub-point of a note the session holds continues it
  Given the user has run engram update, a vault holding a note that says the project is built with make ci, which runs the lint, test and package steps, and a session in which that note arrived with a prompt
  When the user corrects the agent in a prompt: make ci alone fails, because the lint step of make ci needs LINT_STRICT=1, and the turn ends
  Then within 2 minutes the vault holds a note for it whose ID continues the ID of the make ci note

Scenario: 855a2c1b The same thought at the same level is a sibling
  Given the user has run engram update, a vault holding a note that says to format Python files with black before committing, which continues a note about something else, and a session in which only the black note arrived with a prompt
  When the user asks the agent in a prompt to remember to format Go files with gofmt before committing, and the turn ends
  Then within 2 minutes the vault holds a note for it whose ID is a sibling of the ID of the black note

Scenario: 855a2c1c A note the session has not seen is not a target
  Given the user has run engram update, a vault holding a note that says to format Python files with black before committing, which continues a note about something else, and a session in which no note arrived or was looked up, the prompt below included
  When the user asks the agent in a prompt to remember to format Go files with gofmt before committing, and the turn ends
  Then within 2 minutes the vault holds a note for it placed at the top, and its ID continues no other note's and is no other note's sibling
```

### 855a2d Lessons that correct a note say so

Requirement: 855a2d When the lesson of a [turn](../../GLOSSARY.md#turn-automatic-memory) corrects, narrows or refutes a [note](../../GLOSSARY.md#permanent-note) of the [vault](../../GLOSSARY.md#vault) that was written, delivered or looked up earlier in the same [session](../../GLOSSARY.md#session), the note that [automatic memory](../../GLOSSARY.md#automatic-memory) writes MUST say which note it updates, narrows or refutes, as the [learn skill](../../GLOSSARY.md#learn-skill) does, and the note it supersedes MUST show that it was superseded.

#### 855a2d1 UAT Scenarios

```gherkin
Scenario: 855a2d1a A correcting note names the note it corrects
  Given the user has run engram update, a vault holding a note that says the staging database is reset every Sunday night, and a session in which that note arrived with a prompt and the agent may not run engram commands
  When the user tells the agent in a prompt that since last week the staging database is reset every Monday night, and the turn ends
  Then within 2 minutes the vault holds a note that says it is reset every Monday night and that it updates the Sunday note, and the Sunday note shows that the Monday note supersedes it
```

### 855a2e Recall moments within a turn

Requirement: 855a2e When, within a [turn](../../GLOSSARY.md#turn-automatic-memory), the [agent](../../GLOSSARY.md#agent-automatic-memory) is about to declare its work done, has just failed in a way it cannot immediately explain, or is about to start building a new approach, and the [recall skill](../../GLOSSARY.md#recall-skill)'s rule says to run it, [automatic memory](../../GLOSSARY.md#automatic-memory) MUST put what [applies to](../../GLOSSARY.md#applies-to-a-recall-moment) that moment, less what the agent already has, in the agent's context before the agent goes on, as [793a2b Memory arrives with the prompt](#793a2b-memory-arrives-with-the-prompt) does for a [prompt](../../GLOSSARY.md#prompt-automatic-memory). The fourth cue of the recall guidance, before endorsing a proposed approach in conversation, is not promised. The [agent](../../GLOSSARY.md#agent-automatic-memory)'s own use of the memory is not such a moment: [automatic memory](../../GLOSSARY.md#automatic-memory) MUST NOT put anything in the agent's context for a call of the agent to `engram query`, `show`, `show-chunk`, `recall`, `learn` or `amend`, or to the recall, learn, curate or write-memory skill.

Rationale: a [note](../../GLOSSARY.md#permanent-note) put in front of a call reaches the [agent](../../GLOSSARY.md#agent-automatic-memory) with the call's result, so it arrives before the agent's next action and not before the call.

#### 855a2e1 UAT Scenarios

```gherkin
Scenario: 855a2e1a A note arrives before the agent declares its work done
  Given the user has run engram update, and a vault holding a note that says a release is done only when CHANGELOG.md has the version's entry
  When the user asks the agent in a prompt to tag v1.4.0, the prompt arrives with nothing from the vault, and the agent has tagged and is about to declare the release done
  Then before the agent declares it done its context holds the changelog note

Scenario: 855a2e1b A note arrives after a failure the agent cannot explain
  Given the user has run engram update, and a vault holding a note that says make ci fails with the message LINT_STRICT is unset unless LINT_STRICT=1 is set
  When the user asks the agent in a prompt to get the branch ready for review, the prompt arrives with nothing from the vault, and the agent runs make ci, which fails with that message
  Then before the agent's next action its context holds the make ci note

Scenario: 855a2e1c A note arrives before the agent starts building a new approach
  Given the user has run engram update, and a vault holding a note that says schema changes are written as migrations with the migrate tool and never by editing schema.sql
  When the user asks the agent in a prompt to add an index on orders.created_at, the prompt arrives with nothing from the vault, and the agent is about to edit schema.sql
  Then before the agent's next action its context holds the migration note

Scenario: 855a2e1d A call of the agent to the memory is not followed by a delivery
  Given the user has run engram update, and a vault holding a note about the staging database that applies to the prompt below
  When the user asks the agent in a prompt to run engram query for the staging database reset and report what it found
  Then no note arrives after the agent's call to engram query
```

### 855a2g Lessons are durable and atomic

Requirement: 855a2g A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../../GLOSSARY.md#automatic-memory) writes or amends MUST be durable (still true and useful to a future task that has nothing to do with the [turn](../../GLOSSARY.md#turn-automatic-memory): not status, a plan, a decision about the work in hand, a branch, worktree or ticket name, a timestamp or an episode) and atomic (one claim; an amend only sharpens the note's own claim, and a different claim is a new lesson that must clear the same bar), its situation MUST name no ticket or change number unless the lesson is about it, and a lesson that a note written in the same [session](../../GLOSSARY.md#session) states MUST NOT be written again.

#### 855a2g1 UAT Scenarios

```gherkin
Scenario: 855a2g1a A status of the work is not written
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember the status of the work, a branch and a worktree, and the turn ends
  Then 2 minutes after the turn ends the vault holds no note

Scenario: 855a2g1b A lesson the session wrote is not written twice
  Given the user has run engram update, a vault crowded with notes about a user repeating an instruction, and a session whose first turn became a note
  When the user corrects the agent again in a prompt with none of the lesson's words, the agent's answer states the lesson again, and the turn ends
  Then 2 minutes after the turn ends the vault holds the one note that states the correction

Scenario: 855a2g1c A correction made in work on a ticket is written with no ticket in its situation
  Given the user has run engram update, and a session in which the agent has answered a question about work on a ticket
  When the user corrects the agent in a prompt with a rule that holds beyond the ticket, and the turn ends
  Then within 2 minutes the vault holds a note that states the rule and whose situation names no ticket
```
