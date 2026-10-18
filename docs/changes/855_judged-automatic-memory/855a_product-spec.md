# Judged Automatic Memory - Product Spec

## 855a1 Problem Statement

Anyone who works with an [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) in Claude Code and keeps a [vault](../../GLOSSARY.md#vault) gets [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) from [793a Automatic Memory - Product Spec](../793_automatic-memory/793a_product-spec.md#automatic-memory---product-spec) after [`engram update`](../../GLOSSARY.md#engram-update). It looks up [notes](../../GLOSSARY.md#permanent-note) with the raw text of each [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) and delivers every match, without asking whether the prompt calls for memory or whether a match applies, and the owner judges that this pollutes the agent's context. The lessons it writes sit at the top of the vault and are tied to no note they correct, where the [learn skill](../../GLOSSARY.md#learn-skill) places them and ties them. The needed outcome is the North Star ([855f1 North Star](./855f_algorithm.md#855f1-north-star)): when a prompt is a reasonable moment for recall, what arrives is what the [recall skill](../../GLOSSARY.md#recall-skill) would have surfaced, and when a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) is a reasonable moment for learn, what is written is what the learn skill would have written. The terms are those of [793a1a Terms](../793_automatic-memory/793a_product-spec.md#793a1a-terms) as [855a3a Modified: 793a1a Terms](#855a3a-modified-793a1a-terms) changes them.

## 855a2 Added: Judged automatic memory

### 855a2a Delivering writes nothing

Requirement: 855a2a When [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) delivers [notes](../../GLOSSARY.md#permanent-note) for a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms), it MUST NOT add a note to the [vault](../../GLOSSARY.md#vault) or change what any note says.

#### 855a2a1 UAT Scenarios

```gherkin
Scenario: 855a2a1a Delivering notes leaves the vault's notes as they were
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset and a note about formatting Python files
  When the user asks the agent in a prompt to prepare a test run against the staging database tonight, and the agent answers
  Then the staging note was in the agent's context, no note was added to the vault, and every note states what it stated before
```

### 855a2b Lessons are written as the learn skill writes them

Requirement: 855a2b A [note](../../GLOSSARY.md#permanent-note) that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) writes for a lesson ([793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)) MUST state the lesson that the [learn skill](../../GLOSSARY.md#learn-skill) would write for the [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms), in the form that skill writes it: the general rule or fact and not the story of the turn, the situation in which it applies as a cue that a later task would be described by, and the exact names, commands, flags and values it depends on.

#### 855a2b1 UAT Scenarios

```gherkin
Scenario: 855a2b1a A note states the rule and not the story
  Given the user has run engram update, and a session in which the agent may not run engram commands and has force-pushed a branch with git push --force
  When the user tells the agent in a prompt to push with --force-with-lease and never with --force, and the turn ends
  Then within 2 minutes the vault holds a note that states the rule, holds the flag --force-with-lease, names pushing a branch as the situation in which it applies, and does not tell what happened in the turn

Scenario: 855a2b1b The note states the lesson the learn skill writes
  Given the user has run engram update, and two sessions with the same history, in which the agent has force-pushed a branch with git push --force, the first where the agent may not run engram commands and the second where it may
  When in each the user tells the agent in a prompt to push with --force-with-lease and never with --force, and in the second asks it to run the learn skill, and the turn ends
  Then within 2 minutes the note that automatic memory wrote in the first and the note that the learn skill wrote in the second state the same lesson
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
  Given the user has run engram update, a vault holding a note that says to format Python files with black before committing, and a session in which no note arrived or was looked up
  When the user tells the agent in a prompt to format Go files with gofmt before committing, and the turn ends
  Then within 2 minutes the vault holds a note for it placed at the top, and its ID continues no other note's and is no other note's sibling
```

### 855a2d Lessons that correct a note say so

Requirement: 855a2d When the lesson of a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) corrects, narrows or refutes a [note](../../GLOSSARY.md#permanent-note) of the [vault](../../GLOSSARY.md#vault), the note that [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) writes MUST say which note it updates, narrows or refutes, as the [learn skill](../../GLOSSARY.md#learn-skill) does, and the note it supersedes MUST show that it was superseded.

#### 855a2d1 UAT Scenarios

```gherkin
Scenario: 855a2d1a A correcting note names the note it corrects
  Given the user has run engram update, a vault holding a note that says the staging database is reset every Sunday night, and a session in which the agent may not run engram commands
  When the user tells the agent in a prompt that since last week the staging database is reset every Monday night, and the turn ends
  Then within 2 minutes the vault holds a note that says it is reset every Monday night and that it updates the Sunday note, and the Sunday note shows that the Monday note supersedes it

Scenario: 855a2d1b A later prompt gets the correcting note and not the corrected one
  Given the user has run engram update, and a vault holding the Sunday note and the Monday note of the scenario above, the Monday note superseding the Sunday note
  When the user asks the agent in a prompt to prepare a test run against the staging database tonight
  Then before the agent acts its context holds the Monday note and not the Sunday note
```

## 855a3 Modified: Automatic memory of 793

### 855a3a Modified: 793a1a Terms

Target: [793a1a Terms](../793_automatic-memory/793a_product-spec.md#793a1a-terms)

- **Agent:** the AI the user works with in a Claude Code [session](../../GLOSSARY.md#session), not a [subagent](../../GLOSSARY.md#subagent).
- **Prompt:** a message the user sends to the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms).
- **Turn:** a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) and everything the agent does in answer, until it hands control back to the user.
- **Recall moment:** a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) for which the [recall skill](../../GLOSSARY.md#recall-skill)'s own rule says to run it: the request might entail more than a single tool call or anything more than quick, shallow thinking. The user can read that rule in the skill's description.
- **Applies to a prompt:** a [note](../../GLOSSARY.md#permanent-note) applies to a prompt when the recall skill, run for that prompt in a [session](../../GLOSSARY.md#session) on the same [vault](../../GLOSSARY.md#vault), draws on it; of two notes that conflict because the newer reverses the older, only the newer applies. The user can run the skill and read which notes it draws on.
- **Automatic memory:** what this change adds, done without the user or the agent asking: notes that apply to a prompt arrive in the agent's context ([793a2b Memory arrives with the prompt](../793_automatic-memory/793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)), lessons are written as notes ([793a2c Lessons are written without the agent](../793_automatic-memory/793a_product-spec.md#793a2c-lessons-are-written-without-the-agent)) and the conversation becomes searchable ([793a2f The conversation becomes searchable memory](../793_automatic-memory/793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory)).
- **Recognised credential:** one of these, as a [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms) holds it: a private key block; a token of three parts joined by dots that begins `eyJ`, each part of 8 or more letters, digits, `_` or `-` (the first counted after `eyJ`); `sk`, `pk`, `rk`, `ghp`, `gho`, `ghs`, `ghu`, `ghr`, `github_pat`, `npm` or `glpat`, then `-` or `_` and 16 or more letters, digits, `_` or `-`; `AKIA` or `ASIA` and exactly 16 capital letters or digits; `AIza` and 30 or more letters, digits, `_` or `-`; `xoxa-`, `xoxb-`, `xoxp-`, `xoxr-` or `xoxs-` and 10 or more letters, digits or `-`; `Bearer` or `Basic`, in any case, white space and 12 or more letters, digits or `._~+/=-`; the user and password of a web address; the rest of the line of an `Authorization` header; the value that `=`, `:` or `is` gives to a name that holds `key`, `token`, `secret`, `password`, `passwd`, `pwd`, `credential` or `authorization`, or is `pass` or ends in `_pass` or `-pass`, in any case, with up to six words of the sentence between the name and `:` or `is`; a run of 32 or more letters, digits, `+`, `_` or `=` that holds a letter and a digit, other than the 40 lowercase hexadecimal digits of a commit.
- **Sweep:** the indexing of a [session](../../GLOSSARY.md#session)'s conversation that makes it findable by `engram query` ([793a2f The conversation becomes searchable memory](../793_automatic-memory/793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory)).

### 855a3b Modified: 793a2b Memory arrives with the prompt

Target: [793a2b Memory arrives with the prompt](../793_automatic-memory/793a_product-spec.md#793a2b-memory-arrives-with-the-prompt)

Requirement: 793a2b When the user sends a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) that is a [recall moment](#855a3a-modified-793a1a-terms), [automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) MUST put the [notes](../../GLOSSARY.md#permanent-note) that [applies to](#855a3a-modified-793a1a-terms) it, less those the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms) already has, in the agent's context before the agent acts, unasked. It MUST NOT put anything in the agent's context for a prompt that is not a recall moment, and MUST NOT put a note that does not apply to the prompt because its words match. The agent has a note when automatic memory delivered it earlier in the same [session](../../GLOSSARY.md#session) or the agent looked it up in that session; such a note MUST NOT be put there again. Only notes are delivered; the conversation ([793a2f The conversation becomes searchable memory](../793_automatic-memory/793a_product-spec.md#793a2f-the-conversation-becomes-searchable-memory)) is found when the agent asks. The notes arrive beside the prompt, which stays exactly as the user typed it ([793a2d Never in the way](../793_automatic-memory/793a_product-spec.md#793a2d-never-in-the-way)).

The scenarios [793a2b1a](../793_automatic-memory/793a_product-spec.md#793a2b1-uat-scenarios) to [793a2b1d](../793_automatic-memory/793a_product-spec.md#793a2b1-uat-scenarios) of the target stay, and these are added.

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
  Given the user has run engram update, and a vault holding a note that says when the staging database is reset, a note that says to take a backup before a database migration, and a note about formatting Python files
  When the user asks the agent in a prompt to plan a migration of the staging database for tonight, and in a second session on the same vault asks the agent to run the recall skill for the same prompt
  Then the notes that arrived with the prompt in the first session are the notes that the recall skill draws on in the second
```

### 855a3c Modified: 793a2d Never in the way

Target: [793a2d Never in the way](../793_automatic-memory/793a_product-spec.md#793a2d-never-in-the-way)

Requirement: 793a2d [Automatic memory](../793_automatic-memory/793a_product-spec.md#793a1a-terms) MUST NOT hold a [prompt](../793_automatic-memory/793a_product-spec.md#793a1a-terms) for anything but its own judgement and delivery for that prompt: the work that follows an earlier [turn](../793_automatic-memory/793a_product-spec.md#793a1a-terms), the writing of lessons and the sweep, MUST NOT delay it. A judgement or delivery that is only slow MUST NOT make it step aside; the prompt waits for it ([855b4 No time bound before a prompt reaches the agent](./855b_decision-log.md#855b4-no-time-bound-before-a-prompt-reaches-the-agent)). It MUST NOT edit the user's prompt or the text of the [agent](../793_automatic-memory/793a_product-spec.md#793a1a-terms)'s reply, MUST NOT break a [session](../../GLOSSARY.md#session), and MUST NOT deliver to or write from its own sessions, the sessions it starts itself to do its work. When it cannot do its work, for any reason, including a judgement or delivery that fails and a write that fails, it MUST step aside, and the session MUST continue as if automatic memory were not installed.

The scenarios [793a2d1b](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios), [793a2d1d](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios) and [793a2d1e](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios) of the target stay.

### 855a3d Modified: 793a2d1a A slow lookup does not hold a prompt for more than 8 seconds

Target: [793a2d1a A slow lookup does not hold a prompt for more than 8 seconds](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios)

```gherkin
Scenario: 793a2d1a A slow judgement holds the prompt until it has the notes
  Given the user has run engram update, a vault holding a note that says when the staging database is reset, and a judgement that takes 20 seconds
  When the user sends a prompt asking when the staging database is reset
  Then the prompt reaches the agent after the judgement finished, as the user typed it and with the staging note, and the agent answers it
```

### 855a3e Modified: 793a2d1c The work after a turn does not hold the next prompt

Target: [793a2d1c The work after a turn does not hold the next prompt](../793_automatic-memory/793a_product-spec.md#793a2d1-uat-scenarios)

```gherkin
Scenario: 793a2d1c The work after a turn does not hold the next prompt
  Given the user has run engram update, and a judgement, a write and a sweep after a turn that each take 60 seconds
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
  When the user sends a prompt asking what there is to know about the staging database
  Then the session shows one line that names the three notes, and no other line about the delivery
```
