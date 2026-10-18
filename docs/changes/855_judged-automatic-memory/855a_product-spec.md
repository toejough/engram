# Judged Automatic Memory - Product Spec

## 855a1 Problem Statement

Anyone who works with an [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) in Claude Code and keeps a [vault](../../GLOSSARY.md#vault) gets [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) from [793a Automatic Memory - Product Spec](../793_automatic-memory/793a_product-spec.md#automatic-memory---product-spec) after [`engram update`](../../GLOSSARY.md#engram-update). It looks up [notes](../../GLOSSARY.md#permanent-note) with the raw text of each [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) and delivers every match, without asking whether the prompt calls for memory or whether a match applies, and the owner judges that this pollutes the agent's context. The lessons it writes sit at the top of the vault and are tied to no note they correct, where the [learn skill](../../GLOSSARY.md#learn-skill) places them and ties them. The needed outcome is the North Star ([855f1 North Star](./855f_algorithm.md#855f1-north-star)): when a moment is a reasonable moment for recall, what arrives is what the [recall skill](../../GLOSSARY.md#recall-skill) would have surfaced, and when a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) is a reasonable moment for learn, what is written is what the learn skill would have written. The terms are those of [793a1a Terms](../793_automatic-memory/793a_product-spec.md#793a1a-terms) as [855a3a Modified: 793a1a Terms](#855a3a-modified-793a1a-terms) changes them.

## 855a2 Added: Judged automatic memory

### 855a2a Delivery writes what its depth writes

Requirement: 855a2a When [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) delivers for a [recall moment](#855a3a-modified-793a1a-terms) of glance depth, it MUST NOT write, amend or crystallize a [note](../../GLOSSARY.md#permanent-note) in the [vault](../../GLOSSARY.md#vault); marking delivered notes as used, and pulling a used parent note down as a pending copy, is not a write. When the moment is of deep depth, it MUST write what the [recall skill](../../GLOSSARY.md#recall-skill)'s deep mode writes, a note crystallized for a lesson that the vault lacks and the amending of a note it draws on, and every note it writes MUST say it was written automatically ([793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)).

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
  When the user tells the agent in a prompt that releasing this project is always: run make ci, then tag with git tag vX.Y.Z, then push the tag with git push origin vX.Y.Z, and the agent does these in that order and the user confirms it is the way, and the turn ends
  Then within 2 minutes the vault holds a runbook note that names releasing the project as its situation, lists the three commands in that order, and says what is true once they are done
```

### 855a2c Lessons are placed as the learn skill places them

Requirement: 855a2c A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) writes MUST be placed in the [vault](../../GLOSSARY.md#vault)'s order by the ordered test of the [learn skill](../../GLOSSARY.md#learn-skill) ([792a15h Each note is placed by an ordered test](../../specs/product/learn.md#792a15h-each-note-is-placed-by-an-ordered-test)), which considers only notes written, delivered or looked up earlier in the same [session](../../GLOSSARY.md#session). When the note develops one specific sub-point of such a note, its [Luhmann ID](../../GLOSSARY.md#luhmann-id) continues that note's; otherwise, when it extends the same overall thought at the same level, its ID is that note's sibling; otherwise it is placed at the top, even when the vault holds a related note that the session has not seen.

#### 855a2c1 UAT Scenarios

```gherkin
Scenario: 855a2c1a A sub-point of a note the session holds continues it
  Given the user has run engram update, a vault holding a note that says the project is built with make ci, which runs the lint, test and package steps, and a session in which that note arrived with a prompt
  When the user tells the agent in a prompt that the lint step of make ci needs LINT_STRICT=1, and the turn ends
  Then within 2 minutes the vault holds a note for it whose ID continues the ID of the make ci note

Scenario: 855a2c1b The same thought at the same level is a sibling
  Given the user has run engram update, a vault holding a note that says to format Python files with black before committing, and a session in which that note arrived with a prompt
  When the user tells the agent in a prompt to format Go files with gofmt before committing, and the turn ends
  Then within 2 minutes the vault holds a note for it whose ID is a sibling of the ID of the black note

Scenario: 855a2c1c A note the session has not seen is not a target
  Given the user has run engram update, a vault holding a note that says to format Python files with black before committing, and a session in which no note arrived or was looked up, the prompt below included
  When the user tells the agent in a prompt to format Go files with gofmt before committing, and the turn ends
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

Requirement: 855a2e When, within a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms), the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) is about to declare its work done, has just failed in a way it cannot immediately explain, or is about to start building a new approach, and the [recall skill](../../GLOSSARY.md#recall-skill)'s rule says to run it, [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) MUST put what [applies to](#855a3a-modified-793a1a-terms) that moment, less what the agent already has, in the agent's context before the agent goes on, as [793a2b Memory arrives with the prompt](../793_automatic-memory/793a_product-spec.md#793a2b-memory-arrives-with-the-prompt) does for a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms). The fourth cue of the recall guidance, before endorsing a proposed approach in conversation, is not promised ([855b23 Recall runs at the points the host offers](./855b_decision-log.md#855b23-recall-runs-at-the-points-the-host-offers)).

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
```

The scenario 855a2e1c is replaced by [855a2f 855a2e1c A note arrives before the agent starts building a new approach](#855a2f-modified-855a2e1c-a-note-arrives-before-the-agent-starts-building-a-new-approach).

### 855a2f Modified: 855a2e1c A note arrives before the agent starts building a new approach

Target: 855a2e1c A [note](../../GLOSSARY.md#permanent-note) arrives before the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) starts building a new approach, in [855a2e1 UAT Scenarios](#855a2e1-uat-scenarios). A note put in front of a call reaches the agent with the call's result, so it arrives before the agent's next action and not before the call ([855b37 The before-tool cue promises the next action](./855b_decision-log.md#855b37-the-before-tool-cue-promises-the-next-action)).

```gherkin
Scenario: 855a2e1c A note arrives before the agent starts building a new approach
  Given the user has run engram update, and a vault holding a note that says schema changes are written as migrations with the migrate tool and never by editing schema.sql
  When the user asks the agent in a prompt to add an index on orders.created_at, the prompt arrives with nothing from the vault, and the agent is about to edit schema.sql
  Then before the agent's next action its context holds the migration note
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
  Given the user has run engram update, and a judgement, a write and a sweep after a turn that each take 10 minutes
  When the user sends a prompt, and then another while that work goes on
  Then the second prompt reaches the agent while that work goes on, as the user typed it, and the agent answers it
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
