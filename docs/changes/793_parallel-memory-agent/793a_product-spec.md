# Parallel Memory Agent - Product Spec

## 793a1 Problem Statement

Joe, and the LLM agents that work in engram-enabled sessions, get value from engram's memory only if
two things happen at the right moments: relevant vault notes are **consulted** before the working
agent acts, and lessons worth keeping are **saved** after they occur. The memory skills,
[recall](../../specs/product/recall.md) and [learn](../../specs/product/learn.md), already say what to consult, what to save and when
([ambient-guidance](../../specs/product/ambient-guidance.md)). But the working agent has to run them, which depends on its noticing, unprompted, that the moment is
worth a lookup or a save. The agent must "remember to remember."

Measured evidence of the failure (issue #793):

- Capture: the learn behavior fired for about 5% of moments worth learning (4 of 71 baseline; 3 of 66 after the last prose fix).
- Delegation: recalled memory reached 0 of 4 subagents at dispatch (#739).
- Compliance: in a bare-CLAUDE.md headless trial, the instruction to run a memory query before the
  first tool call was refused as prompt injection 6 of 6 times (#790).
- Prose fixes plateau: repeated rewording of the instructions has not moved these rates.

Memory use is gated on the working agent's own metacognition, which is the least reliable component
and competes with the task for attention. The need is for memory to be consulted and saved at the right moments
regardless of whether the working agent thinks to run the skills, out of the working agent's context, without degrading the working session.
The need is for that automatic handling in addition to the existing skills, not in place of them: memory is a dual system, automatic and intentional ([793b175 Memory is a dual system: automatic and intentional](793b_decision-log.md#793b175-memory-is-a-dual-system-automatic-and-intentional)), and this change layers the first beside the second ([793b182 Reduce 793a to the minimum set of end-to-end, user-verifiable requirements; layer automatic memory beside intentional memory](793b_decision-log.md#793b182-reduce-793a-to-the-minimum-set-of-end-to-end-user-verifiable-requirements-layer-automatic-memory-beside-intentional-memory)).

This document specifies only what the System adds. It modifies no canonical node: the skills, ambient guidance, the shim, please, route, the lessons contract and install behavior stay exactly as they are today, and the canonical nodes it cites are cited as unchanged. How the System does what is required here (the wording of what it delivers, what it logs, which files it writes) is for the technical specification.

Terms used below:

- **Working agent:** the model doing the user's task in a session, including any subagents it
  dispatches.
- **System:** the capability this document specifies. It runs alongside the working agent and acts for it.
- **Automatic memory:** the recall, learn and lookup that the System performs at the moments, without the working agent or the user asking for them. For automatic memory the System handles memory and the working agent applies what is delivered.
- **Intentional memory:** a memory command that the working agent or the user runs deliberately, whenever they choose, such as "remember this", `/recall`, `/learn` or a mid-task `engram query`. It is the existing system, unchanged, and is available alongside automatic memory ([793a2k Intentional memory stays available and is never blocked](#793a2k-intentional-memory-stays-available-and-is-never-blocked)).
- **Vault** and **note:** the engram note store and one entry in it (fact, feedback, runbook, or Q&A).
- **Moment:** a point at which the System delivers memory: the moments [ambient-guidance](../../specs/product/ambient-guidance.md) defines for recall, please's orientation, every prompt, a subagent's dispatch and the orchestrator's decision to dispatch ([793a2n Relevant memory reaches the agent at the moments](#793a2n-relevant-memory-reaches-the-agent-at-the-moments)).
- **Lesson:** any corrected failure, including an agent's own mid-task mistake that evidence
  overturned, plus a correction, an explicit save-request, a reversal and a confirmed approach ([793a3a Lessons are captured without the agent](#793a3a-lessons-are-captured-without-the-agent)).
- **Delivery:** placing note content in front of the working agent without the agent requesting it.
- **Integration:** what `engram update` installs for the System in the Claude Code harness ([793a4h Installed by engram update and active in Claude Code](#793a4h-installed-by-engram-update-and-active-in-claude-code)). It is installed per harness, so a condition "while the Integration is installed" is evaluated per harness and per agent session: a session in a harness without it, such as Pi, receives no delivery and runs as before, and its transcript can be captured from only after the session is no longer running ([793a5c Other harnesses are unchanged](#793a5c-other-harnesses-are-unchanged)).
- **Context:** the content currently available to one agent, a top-level agent or one subagent.
  Content removed by compaction is no longer present. A subagent's context is separate from its
  parent's. A resumed session's restored content is present.
- **Memory command:** running recall, learn, curate or write-memory, or an engram command that queries, writes or amends the vault.
- **Transcript source:** a session transcript that engram's automatic indexing finds and indexes ([792a5d Automatic sweep roots](../../specs/product/chunk-indexing.md#792a5d-automatic-sweep-roots)): a Claude Code session log or a Pi session file, whichever harness wrote it and whether or not that harness has the Integration.
- **Backlog:** the content of transcript sources that the System has not yet captured lessons from, held by sessions that are no longer running ([793a3a Lessons are captured without the agent](#793a3a-lessons-are-captured-without-the-agent)).

Requirement keywords follow RFC 2119.

## 793a2 Added: automatic-memory (recall)

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

### 793a2d Delivered memory is attributable

Requirement: 793a2d Every [note](#793a1-problem-statement) or chunk the [System](#793a1-problem-statement) delivers MUST identify its source: a note by its file name and its kind, and a chunk by its id and score.

Rationale: an agent that can see where an item came from can weigh it and look it up.

#### 793a2d1 UAT Scenarios

```gherkin
Scenario: 793a2d1a Delivered content names its source
  Given a delivery of note "474.2026-07-25.charge-one-reviewer-to-execute..."
  When the delivered text in the session transcript is inspected
  Then it gives that note's file name and kind as its source

Scenario: 793a2d1b A delivered chunk names its id
  Given a delivery of the chunk with id "session-ab12#turn-7"
  When the delivered text in the session transcript is inspected
  Then it gives that chunk's id and score
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

### 793a2k Intentional memory stays available and is never blocked

Requirement: 793a2k The [System](#793a1-problem-statement) MUST NOT require the [working agent](#793a1-problem-statement) to run any [memory command](#793a1-problem-statement), or to evaluate, approve or write any memory, as part of [automatic memory](#793a1-problem-statement): the only thing it asks of the agent about memory is to apply what was delivered ([793a2n Relevant memory reaches the agent at the moments](#793a2n-relevant-memory-reaches-the-agent-at-the-moments)). The working agent MAY run any memory command intentionally, and the user MAY ask it to; such a run is not an error, and the System MUST NOT block it or undo its result. Everything that provides [intentional memory](#793a1-problem-statement) (the memory [skills](../../GLOSSARY.md#skill), the ambient guidance and the shim) stays as it is today.

Rationale: the point of the [System](#793a1-problem-statement) is to take memory handling out of the [working agent](#793a1-problem-statement)'s [context](#793a1-problem-statement) by default, so that memory happens at the [moments](#793a1-problem-statement) the agent reliably misses. It does not take memory away from the agent or the user: an agent that is told "remember this" or that wants a lookup mid-task must still be able to run the command.

#### 793a2k1 UAT Scenarios

```gherkin
Scenario: 793a2k1a The agent is neither required nor asked to run a memory command
  Given the Integration is installed
  And the user has not asked for any memory command
  When the human submits a prompt that matches a note and contains a correction
  Then the transcript shows the note's content delivered and a note stating the correction in the vault
  And the transcript shows no request from the System to run a memory command, or to evaluate, approve or write a note

Scenario: 793a2k1c An intentional recall by the agent is allowed
  Given the Integration is installed and a vault that holds a note matching "how releases are tagged"
  When the user says "run /recall on how releases are tagged" and the agent runs the recall skill
  Then the transcript shows the recall's results and the agent's reply to the user
  And any note the recall wrote is still in the vault when the turn completes

Scenario: 793a2k1d An intentional learn by the agent is allowed and is not undone
  Given the Integration is installed
  When the user says "remember that releases are tagged from main" and the agent runs the learn skill
  Then the vault holds one note stating that releases are tagged from main
  And the note is still in the vault after the turn completes

Scenario: 793a2k1e An intentional engram query by the agent is allowed
  Given the Integration is installed and a vault that holds a note matching a topic
  When the agent runs an engram query on that topic in the middle of a task
  Then the transcript shows the query's results and no error
```

### 793a2l Indexing and curation happen without the agent

Requirement: 793a2l After a turn completes, the [System](#793a1-problem-statement) MUST bring the chunk index up to date with the turn's content, and MUST resolve the [vault](#793a1-problem-statement)'s pending [offers](../../GLOSSARY.md#offer) as [curate](../../specs/product/curate.md) does (accept an offer, fold it into the existing [note](#793a1-problem-statement) it duplicates, or discard it), without the [working agent](#793a1-problem-statement). The System MUST NOT ask the agent to curate.

Rationale: indexing and curation are slow bookkeeping that the [working agent](#793a1-problem-statement) need not do or wait for ([793a4b The session is never broken or stalled](#793a4b-the-session-is-never-broken-or-stalled)); the turn that has just completed is what the next lookup must find.

#### 793a2l1 UAT Scenarios

```gherkin
Scenario: 793a2l1a The index catches up after the turn
  Given a turn that completes with content not yet in the chunk index
  When the turn has completed and the System's work for it has finished
  Then a later query returns a chunk from that turn

Scenario: 793a2l1c Offers are resolved by the System
  Given the Integration is installed and a vault that holds two pending offers, one that an existing note already covers and one that no existing note addresses
  When a turn that delivered memory completes
  Then the covered offer is folded into the existing note it duplicates and is no longer pending
  And the other offer is accepted and is no longer pending

Scenario: 793a2l1d The agent is not asked to curate
  Given the Integration is installed and a vault that holds a pending offer
  And a prompt that matches a note
  And the agent runs no memory command itself
  When the prompt is submitted
  Then the session transcript shows the note's content delivered
  And the transcript shows no instruction to curate and no request to the agent to run curate
```

## 793a3 Added: automatic-memory (learn)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a3a Lessons are captured without the agent

Requirement: 793a3a The [System](#793a1-problem-statement) MUST write to the [vault](#793a1-problem-statement) each [lesson](#793a1-problem-statement) that occurs in a working [session](../../GLOSSARY.md#session), top-level or [subagent](../../GLOSSARY.md#subagent) (excluding [793a4f No delivery or capture in its own sessions](#793a4f-no-delivery-or-capture-in-its-own-sessions)), without the [working agent](#793a1-problem-statement) being asked and without it taking part: a correction, an explicit save-request, a [reversal](../../GLOSSARY.md#reversal-capture-kind) and a [confirmed approach](../../GLOSSARY.md#confirmed-approach-capture-kind) ([792a19c Learn fires at the correction moment](../../specs/product/ambient-guidance.md#792a19c-learn-fires-at-the-correction-moment), [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson)), and any corrected failure, including an agent's own mid-task mistake that evidence overturned. A lesson in a subagent session MUST be captured from that session, so that no lesson depends on the subagent or its dispatching agent reporting it. When two subagent sessions contain the same lesson, the System MUST write one [note](#793a1-problem-statement) for it. Each note the System writes MUST record that it was written automatically and the source [transcript](../../GLOSSARY.md#transcript) and the position in it that it came from, in addition to the source learn defines for it.

The [System](#793a1-problem-statement) MUST also capture [lessons](#793a1-problem-statement) from the [backlog](#793a1-problem-statement): from the [transcript source](#793a1-problem-statement) of any [session](../../GLOSSARY.md#session) that is no longer running, of any [harness](../../GLOSSARY.md#harness), whether or not that harness has the [Integration](#793a1-problem-statement) ([793a5c Other harnesses are unchanged](#793a5c-other-harnesses-are-unchanged)). It MUST do so after the fact, beginning when a Claude Code session with the Integration next completes a turn, and MAY spread a large backlog over several turns. It MUST capture from a [transcript](../../GLOSSARY.md#transcript) source only the content it has not captured before. It MUST NOT capture from the transcript source of a session that is still running until that transcript has been quiet for a period the System sets.

The [System](#793a1-problem-statement) MUST NOT write a [note](#793a1-problem-statement) for a [lesson](#793a1-problem-statement) or a conclusion ([793a3q Sound conclusions are persisted by the System](#793a3q-sound-conclusions-are-persisted-by-the-system)) that a note written earlier in the [session](../../GLOSSARY.md#session) already records, or, for a [backlog](#793a1-problem-statement) session, a note written at any time before the System captured from it, whoever wrote it (the System, the agent, the user's command or the orchestrator), including a note the agent of another [harness](../../GLOSSARY.md#harness) wrote with its own learn before the backlog captured that session ([793a5c Other harnesses are unchanged](#793a5c-other-harnesses-are-unchanged)).

Rationale: the [working agent](#793a1-problem-statement)'s [context](#793a1-problem-statement) stays free of memory evaluation, and [subagent](../../GLOSSARY.md#subagent) [sessions](../../GLOSSARY.md#session) are where [lessons](#793a1-problem-statement) occur too. The existing [skills](../../GLOSSARY.md#skill) still ask the agent to save the same lessons, so the [System](#793a1-problem-statement) must not write a second [note](#793a1-problem-statement) for what the agent already saved. A session can also end before its lessons are saved (a Pi session interrupted before its agent learned, a Claude Code session killed mid-turn), so the System catches what is left behind at its next opportunity, without delivering anything into the other session ([793b231 Backlog capture: the System captures lessons from any swept session transcript after the fact](793b_decision-log.md#793b231-backlog-capture-the-system-captures-lessons-from-any-swept-session-transcript-after-the-fact)).

#### 793a3a1 UAT Scenarios

```gherkin
Scenario: 793a3a1a User correction is captured without agent action
  Given the Integration is installed
  When the user says "no, use the targ build system, not go test directly"
  And the turn completes
  Then a feedback note stating that rule exists in the vault
  And the transcript shows no request to the working agent to invoke a learn command
  And the transcript shows no request to evaluate, approve or write the note

Scenario: 793a3a1b Explicit save-request
  When the user says "remember that releases are tagged from main"
  And the turn completes
  Then a fact note stating that exists in the vault

Scenario: 793a3a1f The agent's own mistake overturned by evidence is captured
  Given the working agent stated a cause for a failing test
  And a later command output overturned that cause and the agent corrected it
  When the turn completes
  Then a note stating the corrected cause exists in the vault

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

Scenario: 793a3a1i A captured note records how it was written
  Given a note the System captured automatically
  When its source is inspected
  Then it states the source learn defines for that note
  And it also states that the note was written automatically, and the source transcript and the position in it that it came from

Scenario Outline: 793a3a1j A record that already exists is not written again
  Given the Integration is installed and <earlier>
  When the turn completes
  Then the vault holds one note for it

  Examples:
    | earlier                                                                                                              |
    | the user said "no, use targ, not go test directly" and the agent ran the learn skill, which wrote a feedback note stating that rule |
    | a sound conclusion that the agent wrote to the vault itself during the turn                                          |
    | a Pi session in which the user said "no, use targ, not go test directly" and the Pi agent ran the learn skill, which wrote a feedback note stating that rule, before the session ended and went quiet |

Scenario: 793a3a1k An interrupted Pi session is captured after the fact
  Given the Integration is installed in Claude Code, and Pi has the existing skills, guidance and shim
  And a Pi session in which the user said "no, use targ, not go test directly" ended before its agent ran the learn skill
  And the Pi session's transcript has been quiet for the System's quiet period, and no note states that rule
  When a Claude Code session completes a turn
  Then a feedback note stating that rule exists in the vault
  And the note states that it was written automatically and names the Pi session's transcript and a position in it
  And the Claude Code transcript shows no request to the working agent to run a memory command

Scenario: 793a3a1l A Claude Code session killed mid-turn is captured by a later session
  Given a Claude Code session with the Integration was killed during a turn in which the user said "remember that releases are tagged from main"
  And that session's transcript has been quiet for the System's quiet period, and no note states that
  When a new Claude Code session with the Integration completes a turn
  Then a fact note stating that exists in the vault

Scenario: 793a3a1m A running session is not captured
  Given a Pi session that is still running, whose transcript holds a correction that no note states
  When a Claude Code session with the Integration completes a turn
  Then the vault holds no note stating the correction

Scenario: 793a3a1n A transcript is captured once
  Given the lesson of an ended Pi session was captured by an earlier Claude Code turn, and the Pi session has not run since
  When further Claude Code turns complete
  Then the vault holds one note for the lesson
  And that note records no use by those turns

Scenario: 793a3a1o A session is captured once it is quiet
  Given a Pi session whose transcript holds a correction that no note states
  And the Pi session has ended and its transcript has been quiet for the System's quiet period
  When a Claude Code session with the Integration completes a turn
  Then a note stating the correction exists in the vault

Scenario: 793a3a1p A note another harness's agent wrote is not written again
  Given a Pi session in which the user said "no, use targ, not go test directly" and whose agent saved that rule with its own learn during the session
  And the Pi session has ended and its transcript has been quiet for the System's quiet period
  When a Claude Code session with the Integration completes a turn and the System captures the Pi session as backlog
  Then the vault holds one note stating that rule, the one the Pi agent wrote
```

### 793a3q Sound conclusions are persisted by the System

Requirement: 793a3q The [System](#793a1-problem-statement) MUST write a [note](#793a1-problem-statement) for a sound, non-trivial conclusion that the [working agent](#793a1-problem-statement) states in its response and that no [vault](#793a1-problem-statement) note states, in the cases and form that recall persists conclusions ([792a14p A sound conclusion is persisted as a derived note](../../specs/product/recall.md#792a14p-a-sound-conclusion-is-persisted-as-a-derived-note), [792a14q A cited synthesis is also captured as a question and answer](../../specs/product/recall.md#792a14q-a-cited-synthesis-is-also-captured-as-a-question-and-answer)), without the agent writing it. The System MUST NOT write a note for a conclusion it would hedge below probable.

Rationale: the agent, which has the [context](#793a1-problem-statement), reasons in its response, and the [System](#793a1-problem-statement), which handles memory, persists what is sound in it. Reasoning that is never written down evaporates.

#### 793a3q1 UAT Scenarios

```gherkin
Scenario: 793a3q1a A sound conclusion in the response is persisted
  Given the Integration is installed and a recall moment of the kind at which recall persists conclusions
  And the agent's response states a sound, non-trivial conclusion that no vault note states
  When the turn completes
  Then the vault holds a note stating the conclusion
  And the transcript shows no request to the agent to run a learn command or to write the note

Scenario: 793a3q1c A hunch is not persisted
  Given the Integration is installed and a recall moment of the kind at which recall persists conclusions
  And the agent's response states a conclusion it hedges below probable
  When the turn completes
  Then the vault holds no note stating the conclusion
```

## 793a4 Added: automatic-memory (operator control and safety)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a4b The session is never broken or stalled

Requirement: 793a4b While the [Integration](#793a1-problem-statement) is installed, if the [System](#793a1-problem-statement) fails, times out, or is unavailable, the working [session](../../GLOSSARY.md#session) MUST continue unaffected, with no error surfaced to the [working agent](#793a1-problem-statement) and no lost user input, and the System MUST show the user a warning, on the same surface as the memory-activity summary ([793a5h The user sees a summary of the memory activity](#793a5h-the-user-sees-a-summary-of-the-memory-activity)) and outside the working agent's [context](#793a1-problem-statement), saying that [automatic memory](#793a1-problem-statement) was skipped for that [moment](#793a1-problem-statement) or turn and why (unavailable, timed out, or error). A moment at which nothing matched is not a failure and MUST show no warning. The System MUST NOT make the agent or the user wait for any of its work that is not needed before the agent's next step: capturing, indexing, curating and the summary.

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
  Given capture evaluation raises an error
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

Scenario: 793a4b1d Slow capture does not stall the session
  Given capture takes 20 seconds
  When a turn completes and the user submits the next prompt immediately
  Then the next prompt is processed without waiting for the capture to finish
  And the working agent's response to the completed turn appeared without waiting for the capture

Scenario: 793a4b1e No warning when nothing matched
  Given the Integration is installed and a moment at which nothing in the vault matches
  When the moment passes
  Then nothing is delivered
  And the user is shown no warning
```

### 793a4c Memory purview only

Requirement: 793a4c The [System](#793a1-problem-statement) MUST NOT itself make or override tool permissions, tool arguments, model choice, or the user's prompt text. Its only effect on the [working agent](#793a1-problem-statement)'s [context](#793a1-problem-statement) is what it delivers ([793a2n Relevant memory reaches the agent at the moments](#793a2n-relevant-memory-reaches-the-agent-at-the-moments)). Outside that context it shows the user the memory-activity summary ([793a5h The user sees a summary of the memory activity](#793a5h-the-user-sees-a-summary-of-the-memory-activity)), writes, amends and activates [vault](#793a1-problem-statement) [notes](#793a1-problem-statement), curates pending [offers](../../GLOSSARY.md#offer), refreshes the chunk index, pulls down pending copies of parent notes and makes offers to the parent ([792a21n A local write is offered to the parent](../../specs/product/parent-sync.md#792a21n-a-local-write-is-offered-to-the-parent)), and starts its own [sessions](../../GLOSSARY.md#session) ([793a4f No delivery or capture in its own sessions](#793a4f-no-delivery-or-capture-in-its-own-sessions)). The System MUST NOT write credentials, tokens, or secrets present in a [transcript](../../GLOSSARY.md#transcript) into a note.

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

Scenario: 793a4c1e A secret in a correction is not persisted
  Given the user's correction text includes an API key
  When the lesson is captured
  Then the written note contains the rule but not the key
```

### 793a4f No delivery or capture in its own sessions

Requirement: 793a4f The [System](#793a1-problem-statement) MUST NOT deliver memory into, or capture [lessons](#793a1-problem-statement) from, the [sessions](../../GLOSSARY.md#session) it runs itself (to perform its lookups and captures).

Rationale: a [System](#793a1-problem-statement) [session](../../GLOSSARY.md#session) that triggers the System would recurse and would write [notes](#793a1-problem-statement) about the System's own work.

#### 793a4f1 UAT Scenarios

```gherkin
Scenario: 793a4f1a No note records the System's own work
  Given the System ran sessions of its own to judge a capture
  When the vault is inspected after the turn
  Then no note states a lesson drawn from one of those sessions

Scenario: 793a4f1b A session of its own receives no delivery
  Given the System started a session of its own to look up memory for a moment
  When that session's transcript is inspected
  Then it shows no content delivered by the System
```

### 793a4h Installed by engram update and active in Claude Code

Requirement: 793a4h `engram update` MUST install the [Integration](#793a1-problem-statement) for the Claude Code [harness](../../GLOSSARY.md#harness), directly, when it updates that harness ([792a11g The harnesses that are updated](../../specs/product/install-and-update.md#792a11g-the-harnesses-that-are-updated)), and the [System](#793a1-problem-statement) MUST be active in every Claude Code [session](../../GLOSSARY.md#session) that supports the Integration from then on, with no operating mode to select and no step beyond installing it. It MUST NOT depend on `engram init`. It MUST NOT remove or alter anything else that `engram update` deploys today. An existing install of engram MUST have the System in Claude Code after its next `engram update`, and running `engram update` again MUST change nothing about the Integration. The Integration MUST NOT be installed for any other harness, and `engram update` MUST NOT create the directory of a harness that is not present. `engram update --dry-run` ([792a11f Previewing an update](../../specs/product/install-and-update.md#792a11f-previewing-an-update)) MUST install none of it.

This requirement is delivered by this change and does not wait on any other issue ([793b186 793a4h is delivered by this change and does not depend on #795](793b_decision-log.md#793b186-793a4h-is-delivered-by-this-change-and-does-not-depend-on-795)). When change #795 (`docs/changes/795_vault-init-command/`) later makes `engram update` run `engram init`, init takes over installing the [Integration](#793a1-problem-statement) with no change to this requirement's promises.

Rationale: one state to reason about. Installing turns the [System](#793a1-problem-statement) on, and everything [intentional memory](#793a1-problem-statement) relies on stays installed beside it.

#### 793a4h1 UAT Scenarios

```gherkin
Scenario: 793a4h1a Update installs the Integration for the harness it updates
  Given Claude Code is installed, Pi is not, and the System can run in Claude Code
  When the user runs engram update
  Then a Claude Code session runs with the System active, with no other configuration
  And no Pi configuration is created

Scenario: 793a4h1b An existing install gains the System and keeps what it had
  Given an engram install made before the System existed, with the memory skills, guidance and shim deployed to Claude Code
  When the user runs engram update
  And a session runs with a prompt that matches a note and contains a correction
  Then the transcript shows the note's content delivered
  And a note stating the correction exists in the vault
  And the Claude Code agent's configuration still holds the memory skills, the guidance and the shim it held before

Scenario: 793a4h1c A preview installs none of the Integration
  Given a machine with no Integration
  When the user runs engram update --dry-run
  And a session runs with a prompt that matches a note
  Then the transcript shows no delivered content

Scenario: 793a4h1d One update serves both harnesses differently
  Given Claude Code and Pi are both installed
  And Pi is configured with the recall, learn and shim guidance
  When the user runs engram update
  Then a Claude Code session runs with the System active
  And a Pi session shows no content delivered by the System, and the Pi agent runs recall itself before acting

Scenario: 793a4h1l Running update twice changes nothing
  Given engram update has installed the Integration for Claude Code
  When the user runs engram update again
  Then the Integration is unchanged
  And a Claude Code session behaves identically to one run after the first update
```

## 793a5 Added: automatic-memory (coexistence and visibility)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a5c Other harnesses are unchanged

Requirement: 793a5c In every [harness](../../GLOSSARY.md#harness) in which engram is installed without the [Integration](#793a1-problem-statement), the existing
instruction-based path ([ambient-guidance](../../specs/product/ambient-guidance.md)) MUST continue to work unchanged, and the [working agent](#793a1-problem-statement) in that [harness](../../GLOSSARY.md#harness) MUST keep running recall and learn: its own intentional ingestion and learn (the user-run `engram ingest --auto`, the learn [skill](../../GLOSSARY.md#skill), recall's writes) continue exactly as they did before the [System](#793a1-problem-statement) existed, and the System MUST NOT block, delay, undo or duplicate them. The System MUST NOT deliver into a [session](../../GLOSSARY.md#session) of such a harness or change its [transcript](../../GLOSSARY.md#transcript). The System's own indexing and [backlog](#793a1-problem-statement) capture MUST coexist with that intentional ingestion and learn: when an intentional ingest or learn in such a harness runs at the same time as the System's indexing or capture, both MUST complete, and neither MUST corrupt the index or the [vault](#793a1-problem-statement). Backlog capture takes a [transcript source](#793a1-problem-statement) only once it is quiet ([793a3a Lessons are captured without the agent](#793a3a-lessons-are-captured-without-the-agent)), and a [lesson](#793a1-problem-statement) the harness's own agent already saved is covered by the duplicate rule of 793a3a, so the backlog never writes a second [note](#793a1-problem-statement) for it.

Rationale: [delivery](#793a1-problem-statement) needs the [Integration](#793a1-problem-statement) inside the [session](../../GLOSSARY.md#session), but capture only reads the [transcript](../../GLOSSARY.md#transcript) the [harness](../../GLOSSARY.md#harness) already wrote, so it can run later from a Claude Code session ([793b231 Backlog capture: the System captures lessons from any swept session transcript after the fact](793b_decision-log.md#793b231-backlog-capture-the-system-captures-lessons-from-any-swept-session-transcript-after-the-fact)).

#### 793a5c1 UAT Scenarios

```gherkin
Scenario: 793a5c1a Unsupported harness
  Given a harness without support for the System
  When a session runs with the existing instruction-based path
  Then its behavior is identical to its behavior before the System existed

Scenario: 793a5c1b A Pi session on a machine whose Claude Code has the Integration
  Given Pi with the existing skills, guidance and shim, on a machine whose Claude Code has the Integration installed
  When a Pi session runs with a prompt that matches a note and contains a correction
  Then its behavior is identical to its behavior before the System existed
  And the transcript shows no content delivered by the System
  And the Pi agent's own ingest and learn run and write exactly as they would without the System

Scenario: 793a5c1c A captured Pi session's transcript is left unchanged
  Given a Pi session that has ended, whose transcript holds a correction that no note states, and a copy of that transcript
  When a Claude Code session with the Integration completes a turn and the System captures the correction
  Then the Pi transcript is identical to the copy
  And the Pi transcript shows no content delivered by the System

Scenario: 793a5c1d A Pi agent's own ingest and learn run while the System ingests and captures
  Given Pi with the existing skills, guidance and shim, on a machine whose Claude Code has the Integration installed
  And a Pi session whose agent runs its own ingest and the learn skill
  And a Claude Code session whose turn is completing, with the System indexing and running a backlog capture
  When the two run at the same time
  Then the Pi agent's ingest and learn both complete and write what they would have written without the System
  And the System's ingest and capture both complete
  And the index and the vault are intact: every note is readable, no note ID is used twice, and the index holds no entry twice

Scenario: 793a5c1e A lesson the Pi agent saved is not written again by the backlog
  Given a Pi session whose agent saved lesson L with the learn skill, and which has since ended
  When a Claude Code session with the Integration completes a turn and the System captures that session as backlog
  Then the vault holds one note for L, the one the Pi agent wrote
  And the System wrote no second note for L
```

### 793a5h The user sees a summary of the memory activity

Requirement: 793a5h After each completed top-level turn of a Claude Code [session](../../GLOSSARY.md#session) that has the Integration, once the [System](#793a1-problem-statement)'s work for that turn (capture included) has finished, the System MUST show the user, outside the [working agent](#793a1-problem-statement)'s [context](#793a1-problem-statement), the counts of memory events since its previous summary, across the session, its [subagent](../../GLOSSARY.md#subagent) sessions and the [backlog](#793a1-problem-statement) the System captured after the turn: items delivered, [notes](#793a1-problem-statement) added, notes amended and [offers](../../GLOSSARY.md#offer) curated. A turn with no memory events MUST show nothing. The warning that [automatic memory](#793a1-problem-statement) was skipped ([793a4b The session is never broken or stalled](#793a4b-the-session-is-never-broken-or-stalled)) is shown on this same surface. The summary MUST NOT enter the working agent's context. When a note the System wrote could not reach a parent [vault](#793a1-problem-statement) and its offer is queued ([792a15j A write to a parent is reported as done](../../specs/product/learn.md#792a15j-a-write-to-a-parent-is-reported-as-done)), the System MUST tell the user, on the same surface and also outside the agent's context, that the note is saved and its offer is queued, and MUST NOT call the write failed, even when the turn has no counts to show.

Rationale: the user keeps visibility of the memory work the [System](#793a1-problem-statement) took over without the [working agent](#793a1-problem-statement) relaying it, and hears what the learn [skill](../../GLOSSARY.md#skill) would have told them.

#### 793a5h1 UAT Scenarios

```gherkin
Scenario: 793a5h1a The summary shows the counts after a turn with events
  Given the Integration is installed and a turn in which the System delivered 3 items, added 1 note, amended 1 note and curated 2 offers, and a subagent's session produced 1 of the added notes
  When the turn completes
  Then the user is shown a summary with 3 items delivered, 1 note added, 1 note amended and 2 offers curated

Scenario: 793a5h1b Nothing is shown after a turn without events
  Given the Integration is installed and a turn in which the System delivered, added, amended and curated nothing
  When the turn completes
  Then the user is shown no memory-activity summary

Scenario: 793a5h1c The summary is not in the agent's context
  Given the Integration is installed and a turn after which a summary was shown to the user
  When the user submits the next prompt and the transcript of the working agent's session is inspected
  Then the summary text appears nowhere in the transcript
  And the working agent's next response does not refer to it

Scenario: 793a5h1e The counts cover the events since the previous summary
  Given the Integration is installed and a summary was shown after one turn
  When the next turn completes with 1 note added and nothing else
  Then the user is shown a summary with 1 note added and every other count 0 or omitted
  And no event counted in the previous summary is counted again

Scenario: 793a5h1g The summary follows the capture
  Given the Integration is installed and a turn whose capture takes 20 seconds and adds 1 note
  When the turn completes and the user submits the next prompt immediately
  Then the next prompt is processed without waiting for the capture
  And the summary is shown only after the capture has finished, and counts that note

Scenario: 793a5h1h An unreachable parent is not reported as a failure
  Given the Integration is installed and a captured note whose write printed "engram: parent unreachable (retry after 30s); 1 offer(s) queued"
  When the turn completes
  Then the user is shown a message that the note is saved and its offer is queued, and the message does not call the write failed
  And the message appears nowhere in the working agent's transcript
  And the vault holds exactly one note for the lesson

Scenario: 793a5h1i A backlog capture is counted
  Given the Integration is installed and a turn after which the System captured 1 note from a Pi session that had ended
  When the turn completes
  Then the user is shown a summary with 1 note added
```
