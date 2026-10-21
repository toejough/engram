# Judged Automatic Memory - Product Spec

## 855a1 Problem Statement

Anyone who works with an [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) in Claude Code and keeps a [vault](../../GLOSSARY.md#vault) gets [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) from [793a Automatic Memory - Product Spec](../793_automatic-memory/793a_product-spec.md#automatic-memory---product-spec) after [`engram update`](../../GLOSSARY.md#engram-update). It looks up [notes](../../GLOSSARY.md#permanent-note) with the raw text of each [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) and delivers every match, without asking whether the prompt calls for memory or whether a match applies, and the owner judges that this pollutes the agent's context. The lessons it writes sit at the top of the vault and are tied to no note they correct, where the [learn skill](../../GLOSSARY.md#learn-skill) places them and ties them. The needed outcome is the North Star ([855f1 North Star](./855f_algorithm.md#855f1-north-star)): when a moment is a reasonable moment for recall, what arrives is what the [recall skill](../../GLOSSARY.md#recall-skill) would have surfaced, and when a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) is a reasonable moment for learn, what is written is what the learn skill would have written. The terms are those of [793a1a Terms](../793_automatic-memory/793a_product-spec.md#793a1a-terms) as [855a3a Modified: 793a1a Terms](#855a3a-modified-793a1a-terms) changes them.

## 855a2 Added: Judged automatic memory

### 855a2a Delivery writes what its depth writes

Requirement: 855a2a When [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) delivers for a [recall moment](#855a3a-modified-793a1a-terms) of glance depth, it MUST NOT write, amend or crystallize a [note](../../GLOSSARY.md#permanent-note) in the [vault](../../GLOSSARY.md#vault); marking delivered notes as used, and pulling a used parent note down as a pending copy, is not a write. When the moment is of deep depth, it MUST write what the [recall skill](../../GLOSSARY.md#recall-skill)'s deep mode writes, a note crystallized for a lesson that the vault lacks and the amending of a note it draws on, and every note it crystallizes MUST say it was written automatically ([793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)).

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

Requirement: 855a2b A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) writes for a lesson ([793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)) MUST state the lesson that the [learn skill](../../GLOSSARY.md#learn-skill) would write for the [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms), in the form that skill writes it: the general rule or fact and not the story of the turn, the situation in which it applies as a cue that a later task would be described by, and the exact names, commands, flags and values it depends on. When the turn confirms a reusable, ordered, multi-step procedure for a recurring task, the note MUST be a runbook as that skill writes it: the situation in which it applies, the numbered steps with their exact commands, and what is true once the procedure is complete. The question-and-answer pairs that the skill also captures are not written ([855b15 Lessons are the three kinds that carry a note](./855b_decision-log.md#855b15-lessons-are-the-three-kinds-that-carry-a-note)).

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

Requirement: 855a2c A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) writes MUST be placed in the [vault](../../GLOSSARY.md#vault)'s order by the ordered test of the [learn skill](../../GLOSSARY.md#learn-skill) ([792a15h Each note is placed by an ordered test](../../specs/product/learn.md#792a15h-each-note-is-placed-by-an-ordered-test)), which considers only notes written, delivered or looked up earlier in the same [session](../../GLOSSARY.md#session). When the note develops one specific sub-point of such a note, its [Luhmann ID](../../GLOSSARY.md#luhmann-id) continues that note's; otherwise, when it extends the same overall thought at the same level, its ID is that note's sibling; otherwise it is placed at the top, even when the vault holds a related note that the session has not seen.

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

Requirement: 855a2d When the lesson of a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) corrects, narrows or refutes a [note](../../GLOSSARY.md#permanent-note) of the [vault](../../GLOSSARY.md#vault) that was written, delivered or looked up earlier in the same [session](../../GLOSSARY.md#session), the note that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) writes MUST say which note it updates, narrows or refutes, as the [learn skill](../../GLOSSARY.md#learn-skill) does, and the note it supersedes MUST show that it was superseded.

#### 855a2d1 UAT Scenarios

```gherkin
Scenario: 855a2d1a A correcting note names the note it corrects
  Given the user has run engram update, a vault holding a note that says the staging database is reset every Sunday night, and a session in which that note arrived with a prompt and the agent may not run engram commands
  When the user tells the agent in a prompt that since last week the staging database is reset every Monday night, and the turn ends
  Then within 2 minutes the vault holds a note that says it is reset every Monday night and that it updates the Sunday note, and the Sunday note shows that the Monday note supersedes it
```

### 855a2e Recall moments within a turn

Requirement: 855a2e When, within a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms), the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) is about to declare its work done, has just failed in a way it cannot immediately explain, or is about to start building a new approach, and the [recall skill](../../GLOSSARY.md#recall-skill)'s rule says to run it, [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) MUST put what [applies to](#855a3a-modified-793a1a-terms) that moment, less what the agent already has, in the agent's context before the agent goes on, as [793a2b Memory arrives with the prompt](../793_automatic-memory/793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) does for a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms). The fourth cue of the recall guidance, before endorsing a proposed approach in conversation, is not promised ([855b23 Recall runs at the points the host offers](./855b_decision-log.md#855b23-recall-runs-at-the-points-the-host-offers)). The [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s own use of the memory is not such a moment: [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) MUST NOT put anything in the agent's context for a call of the agent to `engram query`, `show`, `show-chunk`, `recall`, `learn` or `amend`, or to the recall, learn or curate skill ([855b122](./855b_decision-log.md#855b122-a-memory-command-is-not-looked-up-for)).

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

Scenario: 855a2e1d A call of the agent to the memory is not followed by a delivery
  Given the user has run engram update, and a vault holding a note about the staging database that applies to the prompt below
  When the user asks the agent in a prompt to run engram query for the staging database reset and report what it found
  Then no note arrives after the agent's call to engram query
```

The third scenario is replaced by [855a2f 855a2e1c A note arrives before the agent starts building a new approach](#855a2f-modified-855a2e1c-a-note-arrives-before-the-agent-starts-building-a-new-approach).

### 855a2f Modified: 855a2e1c A note arrives before the agent starts building a new approach

Target: the third scenario of [855a2e1 UAT Scenarios](#855a2e1-uat-scenarios). A [note](../../GLOSSARY.md#permanent-note) put in front of a call reaches the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) with the call's result, so it arrives before the agent's next action and not before the call ([855b37 The before-tool cue promises the next action](./855b_decision-log.md#855b37-the-before-tool-cue-promises-the-next-action)).

```gherkin
Scenario: 855a2e1c A note arrives before the agent starts building a new approach
  Given the user has run engram update, and a vault holding a note that says schema changes are written as migrations with the migrate tool and never by editing schema.sql
  When the user asks the agent in a prompt to add an index on orders.created_at, the prompt arrives with nothing from the vault, and the agent is about to edit schema.sql
  Then before the agent's next action its context holds the migration note
```

### 855a2g Lessons are durable and atomic

Requirement: 855a2g A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) writes or amends MUST be durable (still true and useful to a future task that has nothing to do with the [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms): not status, a plan, a decision about the work in hand, a branch, worktree or ticket name, a timestamp or an episode) and atomic (one claim; an amend only sharpens the note's own claim, and a different claim is a new lesson that must clear the same bar), its situation MUST name no ticket or change number unless the lesson is about it, and a lesson that a note written in the same [session](../../GLOSSARY.md#session) states MUST NOT be written again ([855b117 A note is durable and atomic](./855b_decision-log.md#855b117-a-note-is-durable-and-atomic)).

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

The scenario [793a2c1b](../793_automatic-memory/793a_product-spec.md#793a2c1-uat-scenarios) is narrowed by this requirement and replaced by [855a2h 793a2c1b A request to remember becomes a note](#855a2h-modified-793a2c1b-a-request-to-remember-becomes-a-note).

### 855a2h Modified: 793a2c1b A request to remember becomes a note

Target: [793a2c1b A request to remember becomes a note](../793_automatic-memory/793a_product-spec.md#793a2c1-uat-scenarios). A request to remember something that is not durable, such as the status of the work, writes no [note](../../GLOSSARY.md#permanent-note) ([855a2g Lessons are durable and atomic](#855a2g-lessons-are-durable-and-atomic), [855b121 A request to save status writes no note](./855b_decision-log.md#855b121-a-request-to-save-status-writes-no-note)).

```gherkin
Scenario: 793a2c1b A request to remember becomes a note
  Given the user has run engram update, and a session in which the agent may not run engram commands
  When the user asks the agent in a prompt to remember something durable, and the turn ends
  Then within 2 minutes the vault holds a note that states it and says it was written automatically, and the agent wrote nothing to the vault
```

## 855a3 Modified: Automatic memory of 793

### 855a3a Modified: 793a1a Terms

Target: [793a1a Terms](../793_automatic-memory/793a_product-spec.md#793a1a-terms)

- **Agent:** the AI the user works with in a Claude Code [session](../../GLOSSARY.md#session), not a [subagent](../../GLOSSARY.md#subagent).
- **Prompt:** a message the user sends to the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms).
- **Turn:** a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) and everything the agent does in answer, until it hands control back to the user.
- **Recall moment:** a point at which the [recall skill](../../GLOSSARY.md#recall-skill)'s own rule says to run it: a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) for which the request might entail more than a single tool call or anything more than quick, shallow thinking, or one of three points in a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) that the recall guidance for the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) names: the agent is about to declare its work done, has just failed in a way it cannot immediately explain, or is about to start building a new approach. The user can read that rule and those cues in the skill's description and the guidance.
- **Depth:** how much of the recall skill a recall moment calls for: none, which is not a recall moment, glance, which reads, or deep, which reads and also writes as the skill's deep mode writes. The recall skill says when each fits: deep when the decision is weighty or irreversible.
- **Passage:** a stretch of an earlier conversation that the [sweep](../793_automatic-memory/793a_product-spec.md#793a1a-terms) made searchable.
- **Applies to a recall moment:** a [note](../../GLOSSARY.md#permanent-note) or a passage applies to a recall moment when the recall skill, run at the depth the moment calls for, for that moment in a [session](../../GLOSSARY.md#session) on the same [vault](../../GLOSSARY.md#vault) in which [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) is not on, draws on it; of two notes that conflict because the newer reverses the older, only the newer applies. The user can run the skill at that depth, ask the agent to name what it drew on, and read it.
- **Automatic memory:** what this change adds, done without the user or the agent asking: notes and passages that apply to a recall moment arrive in the agent's context ([793a2b Memory arrives with the prompt](../793_automatic-memory/793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)), lessons are written as notes ([793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)) and the conversation becomes searchable ([793a2f The conversation becomes searchable memory](../793_automatic-memory/793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory)).
- **Recognised credential:** one of these, as a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) holds it: a private key block; a token of three parts joined by dots that begins `eyJ`, each part of 8 or more letters, digits, `_` or `-` (the first counted after `eyJ`); `sk`, `pk`, `rk`, `ghp`, `gho`, `ghs`, `ghu`, `ghr`, `github_pat`, `npm` or `glpat`, then `-` or `_` and 16 or more letters, digits, `_` or `-`; `AKIA` or `ASIA` and exactly 16 capital letters or digits; `AIza` and 30 or more letters, digits, `_` or `-`; `xoxa-`, `xoxb-`, `xoxp-`, `xoxr-` or `xoxs-` and 10 or more letters, digits or `-`; `Bearer` or `Basic`, in any case, white space and 12 or more letters, digits or `._~+/=-`; the user and password of a web address; the rest of the line of an `Authorization` header; the value that `=`, `:` or `is` gives to a name that holds `key`, `token`, `secret`, `password`, `passwd`, `pwd`, `credential` or `authorization`, or is `pass` or ends in `_pass` or `-pass`, in any case, with up to six words of the sentence between the name and `:` or `is`; a run of 32 or more letters, digits, `+`, `_` or `=` that holds a letter and a digit, other than the 40 lowercase hexadecimal digits of a commit.
- **Sweep:** the indexing of a [session](../../GLOSSARY.md#session)'s conversation that makes it findable by `engram query` ([793a2f The conversation becomes searchable memory](../793_automatic-memory/793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory)).

### 855a3b Modified: 793a2b Memory arrives with the prompt

Target: [793a2b Memory arrives with the prompt](../793_automatic-memory/793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)

Requirement: 793a2b When the user sends a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) that is a [recall moment](#855a3a-modified-793a1a-terms), [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) MUST put the [notes](../../GLOSSARY.md#permanent-note) and the passages that [apply to](#855a3a-modified-793a1a-terms) it, less those the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) already has, in the agent's context before the agent acts, unasked. It MUST NOT put anything in the agent's context for a prompt that is not a recall moment, and MUST NOT put a note or passage that does not apply to the prompt because its words match. The agent has a note or passage when automatic memory delivered it earlier in the same [session](../../GLOSSARY.md#session) or the agent looked it up in that session; such a note or passage MUST NOT be put there again. What arrives is notes, which hold what an event meant, and passages of earlier conversations ([793a2f The conversation becomes searchable memory](../793_automatic-memory/793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory)), which hold what happened ([855b11 A note or a passage applies when the recall skill draws on it](./855b_decision-log.md#855b11-a-note-or-a-passage-applies-when-the-recall-skill-draws-on-it)). They arrive beside the prompt, which stays exactly as the user typed it ([793a2d Never in the way](../793_automatic-memory/793a_product-spec.md#793a2d-never-in-the-way)).

The scenarios [855a3q 793a2b1d A prompt that matches no note arrives with nothing](#855a3q-modified-793a2b1d-a-prompt-that-matches-no-note-arrives-with-nothing) and [855a3k 793a2b1a The notes that bear on a prompt arrive with it](#855a3k-modified-793a2b1a-the-notes-that-bear-on-a-prompt-arrive-with-it) to [855a3m 793a2b1c A note the agent looked up itself is not delivered](#855a3m-modified-793a2b1c-a-note-the-agent-looked-up-itself-is-not-delivered) are replaced, and this is added.

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

### 855a3c Modified: 793a2d Never in the way

Target: [793a2d Never in the way](../793_automatic-memory/793a_product-spec.md#793a2d-never-in-the-way)

Requirement: 793a2d [Automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) MUST NOT hold a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms), or the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) at a recall moment within a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms), for anything but its own judgement and delivery for that prompt or moment: the work that follows an earlier [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms), the writing of lessons and the sweep, MUST NOT delay it. A judgement or delivery that is only slow MUST NOT make it step aside; the prompt or the agent waits for it ([855b4 No time bound before a prompt reaches the agent](./855b_decision-log.md#855b4-no-time-bound-before-a-prompt-reaches-the-agent)). A judgement or delivery that has not answered after 2 minutes has stalled: it MUST step aside, deliver nothing for the prompt and be shown as a failure. The 2 minutes are a liveness bound and not a latency target ([855b14 A slow step is waited for, a stalled one is not](./855b_decision-log.md#855b14-a-slow-step-is-waited-for-a-stalled-one-is-not)). It MUST NOT edit the user's prompt or the text of the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s reply, MUST NOT break or stall a [session](../../GLOSSARY.md#session), and MUST NOT deliver to or write from its own sessions, the sessions it starts itself to do its work. When it cannot do its work, for any reason, including a judgement or delivery that fails or stalls and a write that fails, it MUST step aside, and the session MUST continue as if automatic memory were not installed.

The scenarios [793a2d1b](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios) and [793a2d1e](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios) of the target stay, [855a3n 793a2d1d The prompt and the reply are as they were](#855a3n-modified-793a2d1d-the-prompt-and-the-reply-are-as-they-were) is replaced, and this is added.

#### 855a3c1 UAT Scenarios

```gherkin
Scenario: 855a3c1a A judgement that never answers is stepped aside after 2 minutes
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a judgement that never answers
  When the user sends a prompt to prepare a test run against the staging database tonight
  Then the prompt reaches the agent within 3 minutes, as the user typed it and with nothing from the vault, the agent answers it, and the session shows with the next prompt a line that says the delivery could not be done, with the reason
```

### 855a3d Modified: 793a2d1a A slow lookup does not hold a prompt for more than 8 seconds

Target: [793a2d1a A slow lookup does not hold a prompt for more than 8 seconds](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios)

```gherkin
Scenario: 793a2d1a A slow judgement holds the prompt until it has the notes
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a judgement that takes 20 seconds
  When the user sends a prompt to prepare a test run against the staging database tonight
  Then the prompt reaches the agent after the judgement finished, as the user typed it and with the staging note, and the agent answers it
```

### 855a3e Modified: 793a2d1c The work after a turn does not hold the next prompt

Target: [793a2d1c The work after a turn does not hold the next prompt](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios)

```gherkin
Scenario: 793a2d1c The work after a turn does not hold the next prompt
  Given the user has run engram update, and a moment, a judgement, a placement, a write and a sweep after a turn that each take 60 seconds
  When the user sends a prompt, and then another while that work goes on
  Then the second prompt reaches the agent within 8 seconds, as the user typed it, and the agent answers it within 20 seconds
```

### 855a3f Modified: 793a3a What is written is delivered later

Target: [793a3a What is written is delivered later](../793_automatic-memory/793a_product-spec.md#793a3a-what-is-written-is-delivered-later)

Requirement: 793a3a A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) writes ([793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)) MUST be put in the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s context ([793a2b Memory arrives with the prompt](../793_automatic-memory/793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)) with every [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) sent after it is written, in the same [session](../../GLOSSARY.md#session) and in later sessions, that is a [recall moment](#855a3a-modified-793a1a-terms) and that the note [applies to](#855a3a-modified-793a1a-terms).

### 855a3g Modified: 793a3a1a A written note is delivered with later prompts

Target: [793a3a1a A written note is delivered with later prompts](../793_automatic-memory/793a_product-spec.md#793a3a1-uat-scenarios)

```gherkin
Scenario: 793a3a1a A written note is delivered with later prompts
  Given the user has run engram update, and a session in which the user corrected the agent and the correction became a note
  When the user sends a prompt that is a recall moment and that the note applies to, in that session and then in a new session
  Then each of the two prompts arrives with the note
```

### 855a3h Modified: 793a2e1g The notes delivered with a prompt are one line

Target: [793a2e1g The notes delivered with a prompt are one line](../793_automatic-memory/793a_product-spec.md#793a2e1-uat-scenarios)

```gherkin
Scenario: 793a2e1g The notes delivered with a prompt are one line
  Given the user has run engram update, and a vault holding three notes that apply to a prompt about the staging database and a note about formatting Python files
  When the user sends a prompt to prepare a test run against the staging database tonight
  Then the session shows one line that names the three notes, and no other line about the delivery
```

### 855a3i Modified: 793a2a1a The next session has memory after one engram update

Target: [793a2a1a The next session has memory after one engram update](../793_automatic-memory/793a_product-spec.md#793a2a1-uat-scenarios)

```gherkin
Scenario: 793a2a1a The next session has memory after one engram update
  Given a machine with Claude Code on which the user has never run engram update, and a vault holding a note that says when the staging database is reset
  When the user runs engram update
  And starts a Claude Code session in a project other than the one engram update was run in, and sends a prompt to prepare a test run against the staging database tonight
  Then the note is in the agent's context before the agent answers
```

### 855a3j Modified: 793a2a1b A second update still delivers each note once

Target: [793a2a1b A second update still delivers each note once](../793_automatic-memory/793a_product-spec.md#793a2a1-uat-scenarios)

```gherkin
Scenario: 793a2a1b A second update still delivers each note once
  Given a machine on which the user has run engram update once, and a vault holding a note that says when the staging database is reset
  When the user runs engram update again
  And starts a session and sends a prompt to prepare a test run against the staging database tonight
  Then the note is in the agent's context once
```

### 855a3k Modified: 793a2b1a The notes that bear on a prompt arrive with it

Target: [793a2b1a The notes that bear on a prompt arrive with it](../793_automatic-memory/793a_product-spec.md#793a2b1-uat-scenarios)

```gherkin
Scenario: 793a2b1a The notes that apply to a prompt arrive with it
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset and a note about formatting Python files
  When the user sends a prompt to prepare a test run against the staging database tonight
  Then before the agent acts its context holds the staging note and not the formatting note, and the prompt in the conversation is exactly as the user typed it
```

### 855a3l Modified: 793a2b1b A note delivered earlier in the session is not delivered again

Target: [793a2b1b A note delivered earlier in the session is not delivered again](../793_automatic-memory/793a_product-spec.md#793a2b1-uat-scenarios)

```gherkin
Scenario: 793a2b1b A note delivered earlier in the session is not delivered again
  Given a session in which a prompt to prepare a test run against the staging database tonight arrived with the staging note
  When the user sends another prompt to prepare a second test run against the staging database tonight
  Then the second prompt arrives with nothing from the vault, and the staging note is in the agent's context once
```

### 855a3m Modified: 793a2b1c A note the agent looked up itself is not delivered

Target: [793a2b1c A note the agent looked up itself is not delivered](../793_automatic-memory/793a_product-spec.md#793a2b1-uat-scenarios)

```gherkin
Scenario: 793a2b1c A note the agent looked up itself is not delivered
  Given a session in which, before the user ran engram update, the agent looked up the staging note itself with engram query, and a vault that has gained another note since, which says the reset is skipped on public holidays
  When the user runs engram update, resumes that session, and sends a prompt to prepare a test run against the staging database tonight
  Then the prompt arrives with the other note and not the staging note
```

### 855a3n Modified: 793a2d1d The prompt and the reply are as they were

Target: [793a2d1d The prompt and the reply are as they were](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios)

```gherkin
Scenario: 793a2d1d The prompt and the reply are as they were
  Given the user has run engram update, and a session in which the user's correction became a note
  When the user sends a prompt that is a recall moment and that the note applies to, typed with extra spaces and a line break
  Then the prompt in the conversation is exactly as the user typed it, and the reply is exactly the agent's text
```

### 855a3o Modified: 793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt

Target: [793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt](../793_automatic-memory/793a_product-spec.md#793a2e1-uat-scenarios)

```gherkin
Scenario: 793a2e1a What was delivered and what was written for the previous turn is in the session by the next prompt
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a session in which the user asked in one prompt to prepare a test run against the staging database tonight and to remember that it is backed up every Monday at 05:00 UTC
  When the user sends the next prompt
  Then the session shows a line that names the staging note delivered with the earlier prompt, and, with the next prompt, a line that names the note written from it
```

### 855a3p Modified: 793a2e1c A delivery that could not be done is shown with its reason

Target: [793a2e1c A delivery that could not be done is shown with its reason](../793_automatic-memory/793a_product-spec.md#793a2e1-uat-scenarios)

```gherkin
Scenario: 793a2e1c A delivery that could not be done is shown with its reason
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset, which cannot be read
  When the user sends a prompt to prepare a test run against the staging database tonight, the vault becomes readable again, and the user sends another prompt
  Then the session shows, with the second prompt, a line that says the delivery for the first could not be done, with the reason
```

### 855a3q Modified: 793a2b1d A prompt that matches no note arrives with nothing

Target: [793a2b1d A prompt that matches no note arrives with nothing](../793_automatic-memory/793a_product-spec.md#793a2b1-uat-scenarios)

```gherkin
Scenario: 793a2b1d A prompt that matches no note arrives with nothing
  Given the user has run engram update, a vault whose notes have nothing to do with the prompt, and a searchable conversation that has nothing to do with it either
  When the user sends the prompt
  Then the prompt arrives with nothing from the vault, and the agent answers it
```

### 855a3r Modified: 792a15e The four kinds of explicit lesson

Target: [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson)

Requirement: 792a15e Learn MUST scan the session for exactly four kinds of moment and MUST hand each to write-memory, only when its note is durable (still true and useful to a future task that has nothing to do with the one in hand: not status or progress of the work, a plan, a decision about the work in hand, a branch, worktree, ticket or change name, a timestamp, a finding about the session's own incident or an episode) and atomic (one claim; a second claim is its own lesson that clears both tests, or is dropped, and is never folded into the note beside it), and its situation MUST name no ticket, issue or change number or branch unless the lesson is about it. A correction (the user corrected the agent's approach or behavior) MUST be handed off as a feedback note carrying the situation, the behavior, its cost as the impact and what to do instead as the action. An explicit save-request ("remember this", "save that", "note for next time", "write this down") for a durable lesson MUST be handed off immediately as a fact note carrying the situation, the subject, the predicate and the object; an explicit save-request for status MUST NOT be handed off, and learn MUST tell the user that it was not saved to the vault and that it belongs in the change's docs or the project's memory. A reversal MUST be handed off as a feedback note whose situation says when the failure mode applies, whose behavior is what the original reasoning did wrong, whose impact is what the reversal cost and whose action is the guard that would have prevented it. A confirmed approach MUST be handed off as a feedback note, or as a runbook note when its shape calls for one ([792a15g A confirmed approach needs a bet or specific praise, and takes the kind its shape calls for](../../specs/product/learn.md#792a15g-a-confirmed-approach-needs-a-bet-or-specific-praise-and-takes-the-kind-its-shape-calls-for)). Every handoff MUST carry a slug and a source of the form "session <date>, context: <one-line what-was-happening>", and the supersession details when the lesson corrects an existing vault note.

The scenarios of the target stay, and this is added.

#### 855a3r1 UAT Scenarios

```gherkin
Scenario: 855a3r1a A save-request for status writes no note
  Given the user said "remember the status of this branch and its worktree"
  When learn runs
  Then write-memory receives nothing for it, and the user is told that the status was not saved to the vault and that it belongs in the change's docs or the project's memory
```

### 855a3s Modified: 792a14j Coverage is judged by the agent and written once per cluster

Target: [792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster)

Requirement: 792a14j In deep mode, recall MUST judge, for each cluster and then for the explore items, whether a candidate note covers the cluster's principle, from the content read and not from cosine alone, and MUST act on the outcome. When a candidate states the principle with no material omission against the recency-weighted view, the outcome is covered and recall MUST run `engram amend --target <candidate> --activate --chunk-source <ids>` and MUST NOT rewrite its content. When a candidate addresses the same situation and omits at least one substantive part of its own claim (the same subject and the same principle) that the members evidence, the outcome is near and recall MUST run `engram amend` with the chunk sources and content flags that sharpen the note's own claim from all members, recency-weighted, and MUST NOT add a different claim to it. When no candidate states the cluster's principle (a candidate on the same situation with a different claim does not), the outcome is absent: recall MUST then hand off to the write-memory skill one note of kind fact or feedback with the cluster's chunk-source IDs and a descriptive source only when the principle is durable (still true and useful to a future task that has nothing to do with this one: not status, a plan, a branch, worktree or ticket name, a timestamp, a finding about this session's incident or an episode) and atomic (one claim), and otherwise MUST write nothing, and the situation of the note MUST name no ticket or change number unless the lesson is about it. When the write corrects, narrows or refutes a surfaced note, recall MUST also pass `--supersedes="<basename>|<type>|<claim>"` with type `updates`, `narrows` or `refutes`. Recall MUST write at most one note per cluster, the representative being a note and never a chunk, MUST wait for each write before the next cluster, and MUST NOT hand-author vocabulary tags.

Rationale: coverage is judged by the agent, not by a cosine threshold, because cosine only nominates the within-cluster top-five candidates. An empty candidate list is a reason to create when the principle is durable and atomic, not to skip the cluster. Near sharpens a note's own claim and never adds another, so that a note does not collect unrelated claims. A write is blocking because the note created or updated for one cluster may be a candidate for another. Updates use `engram amend` and `engram learn` creates only, so the amend path is the one that keeps both copies of a note in step.

The scenarios 792a14j1b and 792a14j1c are replaced by [855a3u](#855a3u-modified-792a14j1b-near-re-synthesizes) and [855a3v](#855a3v-modified-792a14j1c-absent-creates-through-the-worker), the others stay, and these are added.

#### 855a3s1 UAT Scenarios

```gherkin
Scenario: 855a3s1a A different claim on the same situation is absent and not near
  Given a candidate that addresses the cluster's situation but states a different claim than the cluster's principle
  When the agent judges coverage
  Then it judges the cluster absent, runs no engram amend on that candidate, and hands off the cluster's own claim when it is durable and atomic

Scenario: 855a3s1b A cluster that is not durable writes nothing
  Given a cluster whose content is the status of the work, a plan, or a branch, worktree or ticket name
  When the agent judges it absent
  Then it writes no note for it
```

### 855a3t Modified: 792a19j The first action on every request is an engram query

Target: [792a19j The first action on every request is an engram query](../../specs/product/ambient-guidance.md#792a19j-the-first-action-on-every-request-is-an-engram-query)

Requirement: 792a19j An agent that has the shim MUST make its first tool call on every user request an `engram query` that carries at least two `--phrase` values (the request in the agent's own words, and the kind of situation it is in) and a `--text` value holding the user's message verbatim (its first 300 characters or so when long), before it reads a file, answers or asks a clarifying question, and whether or not it has decided to delegate. The one exception is a user message that arrived with a block opening `[engram memory]`: the mod already looked up memory for that message, so the agent MUST NOT run this query again for that message, and every other rule of the shim, including its re-entry moments, stands. It MUST NOT gate the query on a judgment that the task is multi-step, recurring or done before. It MUST write every `--text` and `--phrase` value as `--flag="value"`, never `--flag "value"`, and MUST NOT rewrite or summarize the message in `--text`, whose paraphrase belongs only in `--phrase`.

Rationale: whether the task is one a runbook covers is what the query answers, so the judgment cannot be a gate in front of it. A one-line request still gets a query, and the agent proceeds without a runbook when none matches. The over-fire on one-shot requests is accepted: each fire costs one query with no crystallization. The command is named literally so it is executable when no engram skill is installed. A user's message can begin with `-` (a markdown bullet) with no control over its shape, and the space-separated form reads it as a flag and exits; the attached form survives it ([792b32c2 Free-text flag values use the attached form](../../specs/technical/agent-instruction-bundle.md#792b32c2-free-text-flag-values-use-the-attached-form)). The raw text, not a paraphrase, is what matches a literal trigger ([792a7y Trigger text](../../specs/product/memory-query.md#792a7y-trigger-text), [792a7za A trigger hit comes first](../../specs/product/memory-query.md#792a7za-a-trigger-hit-comes-first)). A block that opens `[engram memory]` shows that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) already looked up memory for that message.

The scenarios of the target stay, and this is added.

#### 855a3t1 UAT Scenarios

```gherkin
Scenario: 855a3t1a A message that arrived with a delivered block is not queried for again
  Given an agent with the shim in its config, and a user message that arrived with a block opening [engram memory]
  When the agent receives it
  Then it runs no engram query for that message, and its re-entry moments still run the query
```

### 855a3u Modified: 792a14j1b Near re-synthesizes

Target: [792a14j1b Near re-synthesizes](../../specs/product/recall.md#792a14j1-uat-scenarios)

```gherkin
Scenario: 792a14j1b Near re-synthesizes
  Given a candidate that addresses the cluster's situation and omits part of its own claim that the members evidence
  When the agent judges it near
  Then it runs engram amend with the chunk sources and content flags that sharpen that claim, and adds no different claim
```

### 855a3v Modified: 792a14j1c Absent creates through the worker

Target: [792a14j1c Absent creates through the worker](../../specs/product/recall.md#792a14j1-uat-scenarios)

```gherkin
Scenario: 792a14j1c Absent creates through the worker
  Given a cluster whose principle is durable and atomic and that no candidate states
  When the agent judges it absent
  Then it hands one fact or feedback note, with the chunk sources and a source string, to the write-memory skill
```

### 855a3w Modified: 793a2c Lessons are written without the agent

Target: [793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)

Requirement: 793a2c When a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) holds a confirmed lesson that the [vault](../../GLOSSARY.md#vault) does not already state ([793a3c Nothing the vault already says is written again](../793_automatic-memory/793a_product-spec.md#793a3c-nothing-the-vault-already-says-is-written-again)), a [note](../../GLOSSARY.md#permanent-note) stating it MUST be in the vault within 2 minutes ([793c2 How soon after a turn must automatic memory have finished writing?](../793_automatic-memory/793c_open-questions.md#793c2-how-soon-after-a-turn-must-automatic-memory-have-finished-writing)) after the turn ends, while the [session](../../GLOSSARY.md#session) stays open for those 2 minutes ([793c14 Is a turn that is followed by the end of the session covered?](../793_automatic-memory/793c_open-questions.md#793c14-is-a-turn-that-is-followed-by-the-end-of-the-session-covered)), written by [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) and not by the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms), and marked as written automatically. For a headless session (`claude -p`) this applies only when `ENGRAM_AUTO_CAPTURE_SDK` is set to `1` ([793c39 Which origins count as the prompts the user sends?](../793_automatic-memory/793c_open-questions.md#793c39-which-origins-count-as-the-prompts-the-user-sends)). A turn holds a confirmed lesson when the user corrected the agent, asked it to remember something durable (even if the agent says it did not save it) or confirmed that an approach worked, when the agent plainly corrected its own earlier claim, or when an uncertain approach succeeded with evidence in the turn. A turn that holds only guesses, plans or chatter MUST NOT produce a note; a thanks or "it worked" that names no approach is chatter. A note MUST NOT contain a [recognised credential](../793_automatic-memory/793a_product-spec.md#793a1a-terms) that appeared in the turn ([793c49 Should the promise that a note holds no credential stay absolute?](../793_automatic-memory/793c_open-questions.md#793c49-should-the-promise-that-a-note-holds-no-credential-stay-absolute)). A credential in any other form, such as a password a sentence gives without `is` or `:`, a token split over two lines, a run of 40 hexadecimal digits, or a run of letters only or of digits only, is not promised away ([793c41 Is the risk of a secret of another shape acceptable?](../793_automatic-memory/793c_open-questions.md#793c41-is-the-risk-of-a-secret-of-another-shape-acceptable)).

The scenarios of the target stay (793a2c1b is already replaced by 855a2h).
