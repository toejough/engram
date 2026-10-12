# Parallel Memory Agent - Product Spec

## 793a1 Problem Statement

Joe, and the LLM agents that work in engram-enabled sessions, get value from engram's memory only if relevant vault notes are consulted before the working agent acts and lessons worth keeping are saved after they occur. The memory skills, [recall](../../specs/product/recall.md) and [learn](../../specs/product/learn.md), already say what to consult, what to save and when ([ambient-guidance](../../specs/product/ambient-guidance.md)). But the working agent has to run them, which depends on its remembering to remember.

Measured evidence of the failure (issue #793):

- Capture: the learn behavior fired for about 5% of moments worth learning (4 of 71 baseline; 3 of 66 after the last prose fix).
- Delegation: recalled memory reached 0 of 4 subagents at dispatch (#739).
- Compliance: in a bare-CLAUDE.md headless trial, the instruction to run a memory query before the first tool call was refused as prompt injection 6 of 6 times (#790).
- Prose fixes plateau: repeated rewording of the instructions has not moved these rates.

The need: engram remembers for the agent, so the agent does not have to remember to remember. Memory has two channels, and both stay:

1. **Intentional memory**, which exists today. The agent or the user run recall, learn, `engram query` and the rest deliberately, whenever they want. Nothing about this changes, in any harness ([793a5k Intentional memory is never blocked or changed](#793a5k-intentional-memory-is-never-blocked-or-changed), [793a5c Other harnesses are unchanged](#793a5c-other-harnesses-are-unchanged)).
2. **Automatic memory**, which is new and exists in Claude Code only. The System runs beside the working agent, out of its context, and does the memory handling the agent reliably forgets:
   - It **surfaces**: at the moments where recall should fire, relevant vault memory arrives in front of the agent before it acts, with an instruction to apply it, and nothing the agent already has is repeated ([793a2n Relevant memory reaches the agent at the moments](#793a2n-relevant-memory-reaches-the-agent-at-the-moments), [793a2e Nothing already in context is delivered again](#793a2e-nothing-already-in-context-is-delivered-again)).
   - It **captures**: every lesson in any session, subagents, interrupted sessions and sessions from other harnesses included, lands in the vault without the agent being asked and without duplicating what is already there ([793a3a Lessons are captured without the agent](#793a3a-lessons-are-captured-without-the-agent)).
   - It **keeps house**: indexing and curation happen after the turn, never making anyone wait ([793a4j Indexing and curation happen after the turn](#793a4j-indexing-and-curation-happen-after-the-turn)).
   - It **handles memory, not the task**: the agent still does all the applying (planning, synthesis, reconciliation), and the System never touches tools, permissions, the model or the prompt ([793a4c Memory purview only](#793a4c-memory-purview-only)).
   - It **never gets in the way**: if it fails, the session continues as if it were not there, and the user is told ([793a4b The session is never broken or stalled](#793a4b-the-session-is-never-broken-or-stalled)).
   - It is **just there**: `engram update` installs it, it is active in every Claude Code session from then on, and the user sees a one-line summary of what it did ([793a5j Installed by engram update and active in Claude Code](#793a5j-installed-by-engram-update-and-active-in-claude-code), [793a5h The user sees a summary of the memory activity](#793a5h-the-user-sees-a-summary-of-the-memory-activity)).

Success test: a user who never thinks about memory still gets the right notes at the right moments, and nothing worth keeping is lost.

Not promised here: that the System's judgements are fast and cheap because they run on the cheapest capable model. That is an assumption about the implementation, tested as hypothesis [793d18 H2: the cheapest capable model is adequate for every System judgement, and is faster and cheaper](./793d_technical-spec.md#793d18-h2-the-cheapest-capable-model-is-adequate-for-every-system-judgement-and-is-faster-and-cheaper).

Preferences, wanted but not essential, so they have no requirement of their own beyond what a scenario below needs:

- `engram update` delivers the install in this change, without waiting on #795 ([793a5j Installed by engram update and active in Claude Code](#793a5j-installed-by-engram-update-and-active-in-claude-code)).
- Capture reaches back to sessions of other harnesses that ended before their lessons were saved ([793a3a Lessons are captured without the agent](#793a3a-lessons-are-captured-without-the-agent)).
- The user is warned when automatic memory is skipped ([793a4b The session is never broken or stalled](#793a4b-the-session-is-never-broken-or-stalled)).
- The System's indexing and capture coexist with intentional ingestion in other harnesses ([793a5c Other harnesses are unchanged](#793a5c-other-harnesses-are-unchanged)).

The existing memory skills, shim, guidance and workflows are not modified by this change and stay as they are today ([793b237 The existing-system-as-is decision is scoped; it is not a ban on Modified operations](793b_decision-log.md#793b237-the-existing-system-as-is-decision-is-scoped-it-is-not-a-ban-on-modified-operations)). How the System does what is required here belongs to the technical specification.

Terms used below:

- **Working agent:** the model doing the user's task in a session, including any subagents it dispatches.
- **System:** the capability this document specifies. It runs alongside the working agent and acts for it.
- **Automatic memory:** the recall, learn and lookup that the System performs at the moments, without the working agent or the user asking for them. For automatic memory the System handles memory and the working agent applies what is delivered.
- **Intentional memory:** a memory command that the working agent or the user runs deliberately, whenever they choose, such as "remember this", `/recall`, `/learn` or a mid-task `engram query`. It is the existing system, unchanged ([793a5k Intentional memory is never blocked or changed](#793a5k-intentional-memory-is-never-blocked-or-changed)).
- **Vault** and **note:** the engram note store and one entry in it (fact, feedback, runbook, or Q&A).
- **Moment:** a point at which the System delivers memory: the moments [ambient-guidance](../../specs/product/ambient-guidance.md) defines for recall, please's orientation, every prompt, a subagent's dispatch and the orchestrator's decision to dispatch ([793a2n Relevant memory reaches the agent at the moments](#793a2n-relevant-memory-reaches-the-agent-at-the-moments)).
- **Lesson:** any corrected failure, including an agent's own mid-task mistake that evidence overturned, plus a correction, an explicit save-request, a reversal and a confirmed approach ([793a3a Lessons are captured without the agent](#793a3a-lessons-are-captured-without-the-agent)).
- **Delivery:** placing note content in front of the working agent without the agent requesting it.
- **Context:** the content currently available to one agent, a top-level agent or one subagent. Content removed by compaction is no longer present. A subagent's context is separate from its parent's. A resumed session's restored content is present.
- **Memory command:** running recall, learn, curate or write-memory, or an engram command that queries, writes or amends the vault.

Requirement keywords follow RFC 2119.

## 793a2 Added: automatic-memory (it surfaces)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a2n Relevant memory reaches the agent at the moments

Requirement: 793a2n The [System](#793a1-problem-statement) MUST deliver relevant [vault](#793a1-problem-statement) memory to the [working agent](#793a1-problem-statement), before the agent's next response or tool call, without the agent running or being asked to run any [memory command](#793a1-problem-statement), at each of these [moments](#793a1-problem-statement):

- where recall fires ([792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments));
- at please's orientation ([792a18d The workflow opens with learn and orients with recall](../../specs/product/please.md#792a18d-the-workflow-opens-with-learn-and-orients-with-recall));
- at every prompt, whatever its source (a human, an automated notification, a scheduled trigger or a message from another [session](../../GLOSSARY.md#session));
- at a [subagent](../../GLOSSARY.md#subagent)'s dispatch, delivered to that subagent for its task (a subagent's own first edit or first failure is not a moment);
- when the orchestrator routes a unit through route, before it routes the unit, including any recorded evidence about which tier suits that kind of work ([792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note)).

Each [delivery](#793a1-problem-statement) MUST tell the agent to apply what it received. When nothing relevant remains to deliver, the [System](#793a1-problem-statement) MUST deliver nothing, that instruction included. Delivery is subject to [793a2e Nothing already in context is delivered again](#793a2e-nothing-already-in-context-is-delivered-again).

Rationale: recall already says what to consult and when; the change is who does it. The [System](#793a1-problem-statement) does not have the agent's full [context](#793a1-problem-statement), so it hands memory over and the agent, which has the context, applies it. Tier evidence reaches the orchestrator only if it is delivered before the unit is routed.

#### 793a2n1 UAT Scenarios

```gherkin
Scenario: 793a2n1a Recall happens without the agent running it
  Given a vault containing a note whose situation matches "declaring a Go package rename done"
  And the user has not asked for any memory command
  And the note is not yet in the transcript
  When the working agent reaches the point of declaring the rename done
  Then the session transcript shows that note's content delivered before the declaration, together with an instruction to apply it
  And the transcript shows no request from the System to the agent to run a memory command

Scenario: 793a2n1e No result, nothing delivered
  Given a moment whose lookup finds no note that is absent from the agent's context
  When the moment completes
  Then the session transcript shows no content delivered for it, the instruction to apply it included

Scenario: 793a2n1g A subagent receives relevant notes
  Given a note matching "writing Go tests in this repo"
  When the working agent dispatches a subagent with the task "add tests for internal/cli"
  Then the subagent's transcript shows that note's content delivered, with the instruction to apply it, before its first response or tool call
  And the working agent's dispatch text did not mention the note

Scenario Outline: 793a2n1h A prompt from any source receives relevant notes
  Given a note whose situation matches the prompt
  And the note is not yet in the transcript
  When <source> delivers that prompt to the working agent
  Then the session transcript shows that note's content delivered, with the instruction to apply it, before the agent's first response or tool call

  Examples:
    | source                         |
    | the human                      |
    | an automated task notification |
    | a scheduled trigger            |
    | a message from another session |

Scenario: 793a2n1i Please's orientation receives relevant notes
  Given a note matching an ask given to please
  When please starts
  Then the transcript shows that note's content delivered, with the instruction to apply it, before please's first response or tool call
  And the transcript shows please was not asked to run recall

Scenario: 793a2n1j Tier evidence is delivered before a dispatch is routed
  Given a dispatch evidence note for a kind of work that shows a cheaper tier failed
  When the orchestrator routes a unit of that kind through route
  Then the transcript shows that evidence delivered, with the instruction to apply it, before the orchestrator routes the unit
  And the transcript shows no request to the orchestrator to run a memory command
```

### 793a2e Nothing already in context is delivered again

Requirement: 793a2e The [System](#793a1-problem-statement) MUST NOT deliver a [note](#793a1-problem-statement) or chunk that is already present in the receiving agent's [context](#793a1-problem-statement), however it got there: delivered earlier by the System, retrieved by the agent or the user (by an intentional recall or query, or otherwise), restored when a [session](../../GLOSSARY.md#session) resumed, or any other means. A [subagent](../../GLOSSARY.md#subagent)'s context is separate from its parent's, so a parent's items do not spare the subagent. Content removed by compaction is not present, so the System MAY deliver it again.

Rationale: an item the agent already has adds only noise. [Intentional memory](#793a1-problem-statement) runs beside [automatic memory](#793a1-problem-statement), so what the agent or the user surfaced themselves must not be delivered again.

#### 793a2e1 UAT Scenarios

```gherkin
Scenario: 793a2e1a Note delivered earlier is not delivered again
  Given note X was delivered to the working agent at its prompt
  And the session has not been compacted
  When the agent's next two tool results match X again
  Then the session transcript shows X's content delivered only once

Scenario: 793a2e1c Note the agent retrieved itself is not delivered
  Given the working agent ran /recall, and separately an engram query, and their output showed note X in the transcript
  When a later prompt matches X
  Then the transcript shows X's content only once, from the agent's own run

Scenario: 793a2e1d A parent's note is still delivered to a subagent
  Given note X is in the parent agent's transcript
  When the parent dispatches a subagent whose task matches X
  Then the subagent's transcript shows X's content delivered before its first response or tool call

Scenario: 793a2e1e A note removed by compaction can be delivered again
  Given note X was delivered and the session was then compacted so that X's content no longer appears in what the agent sees
  When a later prompt matches X
  Then the transcript shows X's content delivered again

Scenario: 793a2e1g A resumed session's restored content counts as present
  Given note X was delivered in a session that was then ended and resumed
  And the resumed session's transcript shows X's content restored
  When a later prompt matches X
  Then the transcript shows no second delivery of X
```

## 793a3 Added: automatic-memory (it captures)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a3a Lessons are captured without the agent

Requirement: 793a3a The [System](#793a1-problem-statement) MUST write to the [vault](#793a1-problem-statement) each [lesson](#793a1-problem-statement) that occurs in any [session](../../GLOSSARY.md#session), whether top-level or [subagent](../../GLOSSARY.md#subagent), interrupted or not, and in Claude Code or in any other [harness](../../GLOSSARY.md#harness), without the [working agent](#793a1-problem-statement) being asked and without it taking part: a correction, an explicit save-request, a [reversal](../../GLOSSARY.md#reversal-capture-kind) and a [confirmed approach](../../GLOSSARY.md#confirmed-approach-capture-kind) ([792a19c Learn fires at the correction moment](../../specs/product/ambient-guidance.md#792a19c-learn-fires-at-the-correction-moment), [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson)), and any corrected failure, including an agent's own mid-task mistake that evidence overturned. A lesson in a subagent session MUST be captured from that session, so that no lesson depends on the subagent or its dispatching agent reporting it. A lesson in a session that ended before it was saved MUST be captured after the fact, by the next Claude Code session that has the System, without anything being delivered into the ended session. The System MUST capture each lesson once: it MUST NOT write a [note](#793a1-problem-statement) for a lesson that a note already records, whoever wrote that note (the System, the agent, the user's command or the agent of another harness), nor two notes for one lesson that two sessions contain.

Rationale: the [working agent](#793a1-problem-statement)'s [context](#793a1-problem-statement) stays free of memory evaluation, and [subagent](../../GLOSSARY.md#subagent) [sessions](../../GLOSSARY.md#session) are where [lessons](#793a1-problem-statement) occur too. The existing [skills](../../GLOSSARY.md#skill) still ask the agent to save the same lessons, so the [System](#793a1-problem-statement) must not write a second [note](#793a1-problem-statement) for what the agent already saved. A session can also end before its lessons are saved (a Pi session interrupted before its agent learned, a Claude Code session killed mid-turn), so the System catches what is left behind at its next opportunity.

#### 793a3a1 UAT Scenarios

```gherkin
Scenario Outline: 793a3a1a A lesson is captured without agent action
  Given the System is installed
  When <what happens> and the turn completes
  Then a note stating <lesson> exists in the vault
  And the transcript shows no request to the working agent to run a memory command or to evaluate, approve or write the note

  Examples:
    | what happens                                                                                                     | lesson                           |
    | the user says "no, use the targ build system, not go test directly"                                              | that rule                        |
    | the user says "remember that releases are tagged from main"                                                      | that releases are tagged from main |
    | the agent states a cause for a failing test, a later command output overturns it, and the agent corrects itself | the corrected cause              |

Scenario: 793a3a1g A lesson in a subagent session is captured
  Given the working agent dispatched a subagent
  And the dispatching agent tells the subagent "no, run targ test, not go test"
  When the subagent finishes and its context is discarded
  Then a note stating that rule exists in the vault
  And the subagent's transcript shows no request to it to invoke a learn command or to report a lesson

Scenario: 793a3a1h Two subagents contain the same lesson
  Given two subagents whose turns complete at the same time, each containing the same lesson
  When both captures finish
  Then the vault holds exactly one note stating the lesson

Scenario Outline: 793a3a1j A lesson that is already recorded is not written again
  Given the System is installed and <earlier>
  When a Claude Code turn completes
  Then the vault holds one note for the lesson

  Examples:
    | earlier                                                                                                                                                  |
    | the user said "no, use targ, not go test directly" in this turn and the agent ran the learn skill, which wrote a feedback note stating that rule          |
    | a Pi session in which the user said "no, use targ, not go test directly" ended after its agent ran the learn skill, which wrote a feedback note stating that rule |
    | the lesson of an ended Pi session was captured by an earlier Claude Code turn, and the Pi session has not run since                                      |

Scenario Outline: 793a3a1k A lesson in a session that ended before it was saved is captured later
  Given <session>, in which the user said <said>, which ended before its lesson was saved, and no note states that lesson
  When a Claude Code session that has the System completes a turn
  Then a note stating <lesson> exists in the vault
  And the Claude Code transcript shows no request to the working agent to run a memory command

  Examples:
    | session                                | said                                                  | lesson                             |
    | a Pi session whose agent never ran learn | "no, use targ, not go test directly"                  | that rule                          |
    | a Claude Code session killed mid-turn  | "remember that releases are tagged from main"         | that releases are tagged from main |
```

## 793a4 Added: automatic-memory (it keeps house, handles memory and not the task, and stays out of the way)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a4j Indexing and curation happen after the turn

Requirement: 793a4j After a turn completes, the [System](#793a1-problem-statement) MUST bring the chunk index up to date with the turn's content, and MUST resolve the [vault](#793a1-problem-statement)'s pending [offers](../../GLOSSARY.md#offer) as curate does (accept an offer, fold it into the existing [note](#793a1-problem-statement) it duplicates, or discard it), without the [working agent](#793a1-problem-statement) doing it or being asked to, and without making the agent or the user wait for any of that work.

Rationale: indexing and curation are slow bookkeeping that the [working agent](#793a1-problem-statement) need not do or wait for; the turn that has just completed is what the next lookup must find.

#### 793a4j1 UAT Scenarios

```gherkin
Scenario: 793a4j1a The index catches up after the turn
  Given a turn that completes with content not yet in the chunk index
  When the turn has completed and the System's work for it has finished
  Then a later query returns a chunk from that turn

Scenario: 793a4j1c Offers are resolved by the System
  Given the System is installed and a vault that holds two pending offers, one that an existing note already covers and one that no existing note addresses
  When a turn that delivered memory completes
  Then the covered offer is folded into the existing note it duplicates and is no longer pending
  And the other offer is accepted and is no longer pending

Scenario: 793a4j1d The agent is not asked to curate
  Given the System is installed and a vault that holds a pending offer
  And a prompt that matches a note
  And the agent runs no memory command itself
  When the prompt is submitted
  Then the session transcript shows the note's content delivered
  And the transcript shows no instruction to curate and no request to the agent to run curate

Scenario: 793a4j1e Slow housekeeping does not stall the session
  Given the System is installed and its work for a turn takes 20 seconds
  When the turn completes and the user submits the next prompt immediately
  Then the next prompt is processed without waiting for that work to finish
  And the working agent's response to the completed turn appeared without waiting for it
```

### 793a4c Memory purview only

Requirement: 793a4c The [System](#793a1-problem-statement) MUST NOT itself make or override tool permissions, tool arguments, model choice, or the user's prompt text, and MUST NOT do the [working agent](#793a1-problem-statement)'s task: what the agent does with a [delivery](#793a1-problem-statement) (planning, synthesis, reconciliation with the task) stays with the agent. The System MUST NOT ask the agent to do anything about memory except apply what was delivered: no [memory command](#793a1-problem-statement) to run, and no [note](#793a1-problem-statement) to evaluate, approve or write. Its only effect on the agent's [context](#793a1-problem-statement) is what it delivers ([793a2n Relevant memory reaches the agent at the moments](#793a2n-relevant-memory-reaches-the-agent-at-the-moments)). Outside that context it writes, amends and curates [vault](#793a1-problem-statement) notes, refreshes the index, and shows the user what it did ([793a5h The user sees a summary of the memory activity](#793a5h-the-user-sees-a-summary-of-the-memory-activity)).

#### 793a4c1 UAT Scenarios

```gherkin
Scenario: 793a4c1a Tool calls and permissions unchanged
  Given a session in which the System delivered notes and captured a lesson
  When the session's tool calls and permission decisions are inspected
  Then every tool call ran with exactly the arguments the agent issued
  And every permission decision was made by the user's permission settings
  And the model in use is the one the session was configured with

Scenario: 793a4c1b The prompt is unchanged and the note is separate
  Given a prompt that matches a note
  When the session transcript is inspected
  Then the user's prompt text is exactly as the user typed it
  And the delivered note appears as separate content attributed to its source

Scenario: 793a4c1c The only effect on the working agent's context is what is delivered
  Given a session in which the System delivered notes and captured a lesson
  When the session transcript is inspected
  Then the only content the System added to the working agent's context is the delivered items and the instruction to apply them

Scenario: 793a4c1f The agent is neither required nor asked to run a memory command
  Given the System is installed
  And the user has not asked for any memory command
  When the human submits a prompt that matches a note and contains a correction
  Then the transcript shows the note's content delivered and a note stating the correction in the vault
  And the transcript shows no request from the System to run a memory command, or to evaluate, approve or write a note
```

### 793a4b The session is never broken or stalled

Requirement: 793a4b While the [System](#793a1-problem-statement) is installed, if it fails, times out, or is unavailable, the working [session](../../GLOSSARY.md#session) MUST continue unaffected, with no error surfaced to the [working agent](#793a1-problem-statement) and no lost user input, and the System MUST show the user a warning, on the same surface as the memory-activity summary ([793a5h The user sees a summary of the memory activity](#793a5h-the-user-sees-a-summary-of-the-memory-activity)) and outside the working agent's [context](#793a1-problem-statement), saying that [automatic memory](#793a1-problem-statement) was skipped for that [moment](#793a1-problem-statement) or turn and why (unavailable, timed out, or error). A moment at which nothing matched is not a failure and MUST show no warning.

Rationale: the owner wants to know when the automatic channel silently did nothing, without that ever reaching the [working agent](#793a1-problem-statement) ([793b185 The user is warned when automatic memory is skipped](793b_decision-log.md#793b185-the-user-is-warned-when-automatic-memory-is-skipped)).

#### 793a4b1 UAT Scenarios

```gherkin
Scenario: 793a4b1a Vault unavailable
  Given the vault cannot be read
  When the human submits a prompt
  Then the session proceeds normally with nothing delivered
  And the user is shown a warning that automatic memory was skipped because the vault is unavailable
  And the warning appears nowhere in the working agent's transcript

Scenario: 793a4b1b Evaluation crashes
  Given the System's work for a turn raises an error
  When a turn completes
  Then the session continues and the next prompt is processed normally
  And the user is shown a warning that automatic memory was skipped because of an error
  And the warning appears nowhere in the working agent's transcript

Scenario: 793a4b1c No error reaches the agent and the user is warned
  Given the System times out at a moment
  When the session transcript is inspected
  Then it shows no error message from the System and no user input missing
  And the user was shown a warning that automatic memory was skipped because the System timed out
  And the warning appears nowhere in the transcript

Scenario: 793a4b1e No warning when nothing matched
  Given the System is installed and a moment at which nothing in the vault matches
  When the moment passes
  Then nothing is delivered
  And the user is shown no warning
```

## 793a5 Added: automatic-memory (it is just there, and both channels stay)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a5j Installed by engram update and active in Claude Code

Requirement: 793a5j `engram update` MUST install the [System](#793a1-problem-statement) for the Claude Code [harness](../../GLOSSARY.md#harness) when it updates that harness ([792a11g The harnesses that are updated](../../specs/product/install-and-update.md#792a11g-the-harnesses-that-are-updated)), and the System MUST be active in every Claude Code [session](../../GLOSSARY.md#session) from then on, with no operating mode to select and no step beyond installing it. An existing install of engram MUST have the System in Claude Code after its next `engram update`, and running `engram update` again MUST change nothing about it. `engram update --dry-run` ([792a11f Previewing an update](../../specs/product/install-and-update.md#792a11f-previewing-an-update)) MUST install none of it. The install MUST NOT remove or alter anything else that `engram update` deploys today, MUST NOT be made for any other harness, and MUST NOT create the directory of a harness that is not present.

Rationale: one state to reason about. Installing turns the [System](#793a1-problem-statement) on, and everything [intentional memory](#793a1-problem-statement) relies on stays installed beside it.

#### 793a5j1 UAT Scenarios

```gherkin
Scenario: 793a5j1a Update installs the System for the harness it updates
  Given Claude Code is installed and Pi is not
  When the user runs engram update
  Then a Claude Code session runs with the System active, with no other configuration
  And no Pi configuration is created

Scenario: 793a5j1b An existing install gains the System and keeps what it had
  Given an engram install made before the System existed, with the memory skills, guidance and shim deployed to Claude Code
  When the user runs engram update
  And a session runs with a prompt that matches a note and contains a correction
  Then the transcript shows the note's content delivered
  And a note stating the correction exists in the vault
  And the Claude Code agent's configuration still holds the memory skills, the guidance and the shim it held before

Scenario: 793a5j1c A preview installs none of the System
  Given a machine without the System
  When the user runs engram update --dry-run
  And a session runs with a prompt that matches a note
  Then the transcript shows no delivered content

Scenario: 793a5j1d One update serves both harnesses differently
  Given Claude Code and Pi are both installed
  And Pi is configured with the recall, learn and shim guidance
  When the user runs engram update
  Then a Claude Code session runs with the System active
  And a Pi session shows no content delivered by the System, and the Pi agent runs recall itself before acting

Scenario: 793a5j1l Running update twice changes nothing
  Given engram update has installed the System for Claude Code
  When the user runs engram update again
  Then the System is unchanged
  And a Claude Code session behaves identically to one run after the first update
```

### 793a5h The user sees a summary of the memory activity

Requirement: 793a5h After each completed top-level turn of a Claude Code [session](../../GLOSSARY.md#session) that has the [System](#793a1-problem-statement), once the System's work for that turn has finished, the System MUST show the user, outside the [working agent](#793a1-problem-statement)'s [context](#793a1-problem-statement), a one-line summary of what it did since its previous summary, across the session and its [subagent](../../GLOSSARY.md#subagent) sessions: items delivered, [notes](#793a1-problem-statement) added, notes amended and [offers](../../GLOSSARY.md#offer) curated. A turn with no memory events MUST show nothing. The summary MUST NOT enter the working agent's context. The warning that [automatic memory](#793a1-problem-statement) was skipped ([793a4b The session is never broken or stalled](#793a4b-the-session-is-never-broken-or-stalled)) is shown on this same surface.

Rationale: the user keeps visibility of the memory work the [System](#793a1-problem-statement) took over without the [working agent](#793a1-problem-statement) relaying it.

#### 793a5h1 UAT Scenarios

```gherkin
Scenario: 793a5h1a The summary shows the counts after a turn with events
  Given the System is installed and a turn in which it delivered 3 items, added 1 note, amended 1 note and curated 2 offers, and a subagent's session produced 1 of the added notes
  When the turn completes
  Then the user is shown a one-line summary with 3 items delivered, 1 note added, 1 note amended and 2 offers curated

Scenario: 793a5h1b Nothing is shown after a turn without events
  Given the System is installed and a turn in which it delivered, added, amended and curated nothing
  When the turn completes
  Then the user is shown no memory-activity summary

Scenario: 793a5h1c The summary is not in the agent's context
  Given the System is installed and a turn after which a summary was shown to the user
  When the user submits the next prompt and the transcript of the working agent's session is inspected
  Then the summary text appears nowhere in the transcript
  And the working agent's next response does not refer to it

Scenario: 793a5h1e The counts cover the events since the previous summary
  Given the System is installed and a summary was shown after one turn
  When the next turn completes with 1 note added and nothing else
  Then the user is shown a summary with 1 note added and every other count 0 or omitted
  And no event counted in the previous summary is counted again
```

### 793a5k Intentional memory is never blocked or changed

Requirement: 793a5k The [System](#793a1-problem-statement) MUST NOT block, change or undo [intentional memory](#793a1-problem-statement), in any [harness](../../GLOSSARY.md#harness). The [working agent](#793a1-problem-statement) MAY run any [memory command](#793a1-problem-statement) intentionally, and the user MAY ask it to; such a run is not an error, and its result stays. Everything that provides intentional memory (the memory [skills](../../GLOSSARY.md#skill), the ambient guidance and the shim) stays as it is today.

Rationale: the [System](#793a1-problem-statement) takes memory handling out of the [working agent](#793a1-problem-statement)'s [context](#793a1-problem-statement) by default, so that memory happens at the [moments](#793a1-problem-statement) the agent reliably misses. It does not take memory away from the agent or the user: an agent that is told "remember this" or that wants a lookup mid-task must still be able to run the command.

#### 793a5k1 UAT Scenarios

```gherkin
Scenario: 793a5k1c An intentional recall by the agent is allowed
  Given the System is installed and a vault that holds a note matching "how releases are tagged"
  When the user says "run /recall on how releases are tagged" and the agent runs the recall skill
  Then the transcript shows the recall's results and the agent's reply to the user
  And any note the recall wrote is still in the vault when the turn completes

Scenario: 793a5k1d An intentional learn by the agent is allowed and is not undone
  Given the System is installed
  When the user says "remember that releases are tagged from main" and the agent runs the learn skill
  Then the vault holds one note stating that releases are tagged from main
  And the note is still in the vault after the turn completes

Scenario: 793a5k1e An intentional engram query by the agent is allowed
  Given the System is installed and a vault that holds a note matching a topic
  When the agent runs an engram query on that topic in the middle of a task
  Then the transcript shows the query's results and no error
```

### 793a5c Other harnesses are unchanged

Requirement: 793a5c In every [harness](../../GLOSSARY.md#harness) in which engram is installed without the [System](#793a1-problem-statement), the existing instruction-based path ([ambient-guidance](../../specs/product/ambient-guidance.md)) MUST continue to work unchanged, and the [working agent](#793a1-problem-statement) in that harness MUST keep running recall and learn: its own intentional ingestion and learn continue exactly as they did before the System existed, and the System MUST NOT block, delay, undo or duplicate them. The System MUST NOT deliver into a [session](../../GLOSSARY.md#session) of such a harness or change its [transcript](../../GLOSSARY.md#transcript).

Rationale: [delivery](#793a1-problem-statement) needs the [System](#793a1-problem-statement) inside the [session](../../GLOSSARY.md#session), so a [harness](../../GLOSSARY.md#harness) without it keeps today's behavior, and its sessions are read only to capture [lessons](#793a1-problem-statement) ([793a3a Lessons are captured without the agent](#793a3a-lessons-are-captured-without-the-agent)).

#### 793a5c1 UAT Scenarios

```gherkin
Scenario: 793a5c1a Unsupported harness
  Given a harness without support for the System
  When a session runs with the existing instruction-based path
  Then its behavior is identical to its behavior before the System existed

Scenario: 793a5c1b A Pi session on a machine whose Claude Code has the System
  Given Pi with the existing skills, guidance and shim, on a machine whose Claude Code has the System installed
  When a Pi session runs with a prompt that matches a note and contains a correction
  Then its behavior is identical to its behavior before the System existed
  And the transcript shows no content delivered by the System
  And the Pi agent's own ingest and learn run and write exactly as they would without the System

Scenario: 793a5c1c A captured Pi session's transcript is left unchanged
  Given a Pi session that has ended, whose transcript holds a correction that no note states, and a copy of that transcript
  When a Claude Code session that has the System completes a turn and the System captures the correction
  Then the Pi transcript is identical to the copy
  And the Pi transcript shows no content delivered by the System
```
