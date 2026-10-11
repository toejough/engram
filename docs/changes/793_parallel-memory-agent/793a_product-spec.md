# Parallel Memory Agent - Product Spec

## 793a1 Problem Statement

Joe, and the LLM agents that work in engram-enabled sessions, get value from engram's memory only if
two things happen at the right moments: relevant vault notes are **consulted** before the working
agent acts, and lessons worth keeping are **saved** after they occur. Today both depend on the working
agent noticing, unprompted, that the moment is worth a lookup or a save. The agent must "remember to
remember."

Measured evidence of the failure (issue #793):

- Capture: the learn behavior fired for about 5% of moments worth learning (4 of 71 = 5.6% baseline;
  3 of 66 = 4.5% after the last prose fix).
- Delegation: recalled memory reached 0 of 4 subagents at dispatch (#739).
- Compliance: in a bare-CLAUDE.md headless trial, the instruction to run a memory query before the
  first tool call was refused as prompt injection 6 of 6 times (#790).
- Prose fixes plateau: repeated rewording of the instructions has not moved these rates.

Memory use is gated on the working agent's own metacognition, which is the least reliable component
and competes with the task for attention. Memory must be consulted and saved at the right moments
regardless of whether the working agent thinks to do so, without degrading the working session.

Terms used below:

- **Working agent:** the model doing the user's task in a session, including any subagents it
  dispatches.
- **System:** the capability this document specifies. It runs alongside the working agent.
- **Vault** and **note:** the engram note store and one entry in it (fact, feedback, runbook, or Q&A).
- **Moment:** a point where memory should be consulted (a prompt, a tool result, a subagent dispatch)
  or where a lesson may have occurred (a completed turn). A **recall moment** is one of the first kind.
- **Depth:** how thorough one recall is, `glance` or `deep`, as defined by [792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths).
- **Lesson:** any corrected failure, including an agent's own mid-task mistake that evidence
  overturned, plus a correction, an explicit save-request and a confirmed approach. It is wider than
  the four kinds in [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson).
- **Injection:** placing note content in front of the working agent without the agent requesting it.
- **Context:** the content currently available to one agent, a top-level agent or one subagent.
  Content removed by compaction is no longer present. A subagent's context is separate from its
  parent's. A resumed session's restored content is present.
- **Operator:** the human who configures and audits the System.

Requirement keywords follow RFC 2119.

Sections [793a2 automatic-memory (automatic recall)](#793a2-added-automatic-memory-automatic-recall) through [793a5 automatic-memory (independence, coexistence, and measurement)](#793a5-added-automatic-memory-independence-coexistence-and-measurement) add requirements to one new capability. A requirement carried
over from the draft keeps its draft identifier in a "formerly" note so that review comments on the
draft still map.

## 793a2 Added: automatic-memory (automatic recall)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a2a Recall at a human prompt

Requirement: 793a2a (formerly REC-1) When a human submits a prompt, the System MUST make what recall
returns for that prompt ([793a2j Volume equals the depth's output](#793a2j-volume-equals-the-depths-output)) available to the working agent before it begins acting on that prompt,
without the agent requesting it.

#### 793a2a1 UAT Scenarios

```gherkin
Scenario: 793a2a1a Relevant note exists for the prompt
  Given a vault containing a note whose situation matches "renaming a Go package"
  And a session whose configuration contains no memory instructions
  When the human submits "rename package foo to bar across the repo"
  Then the session transcript shows that note's content delivered before the agent's first response or tool call
  And the transcript shows the agent ran no memory command

Scenario: 793a2a1c Only incidental items come back
  Given a vault holding only notes about unrelated topics, and an indexed chunk history
  And the operator runs the glance query for the prompt and finds only newest chunks returned
  When the human submits the prompt
  Then the session transcript shows those chunks delivered before the agent's first response or tool call
```

### 793a2b Recall after each tool result

Requirement: 793a2b (formerly REC-2) After a tool result is returned to the working agent, and before
the agent's next response or tool call, the System MUST make what recall returns for the current
situation (the tool, its arguments, and its outcome) available ([793a2j Volume equals the depth's output](#793a2j-volume-equals-the-depths-output)).

#### 793a2b1 UAT Scenarios

```gherkin
Scenario: 793a2b1a Tool failure matches a known lesson
  Given a note whose situation matches "go build fails with an import cycle"
  When the working agent runs a command that fails with an import-cycle error
  Then the session transcript shows that note's content delivered before the agent's next response or tool call

Scenario: 793a2b1b Every post-tool step is covered
  Given a turn in which the working agent makes 5 tool calls
  When the operator reads the log for that turn
  Then each of the 5 tool results has a recorded injection decision (injected or none)
```

### 793a2c Recall at subagent dispatch

Requirement: 793a2c (formerly REC-3) When the working agent dispatches a subagent, the System MUST
make what recall returns for the subagent's task available to that subagent ([793a2j Volume equals the depth's output](#793a2j-volume-equals-the-depths-output)).

#### 793a2c1 UAT Scenarios

```gherkin
Scenario: 793a2c1a Subagent receives relevant notes
  Given a note matching "writing Go tests in this repo"
  When the working agent dispatches a subagent with the task "add tests for internal/cli"
  Then the subagent's transcript shows that note's content delivered before its first response or tool call
  And the working agent's dispatch text did not mention the note
```

### 793a2d Attributable presentation

Requirement: 793a2d (formerly REC-4) Injected notes MUST identify their source note.

Rationale: an agent that can see where a note came from can weigh it and look it up. Imperative memory,
such as a runbook or a skill, is a legitimate injection; whether injected memory is applied is
tested by [793a5f Effect on outcomes](#793a5f-effect-on-outcomes).

#### 793a2d1 UAT Scenarios

```gherkin
Scenario: 793a2d1a Injected content names its source
  Given an injection of note "474.2026-07-25.charge-one-reviewer-to-execute..."
  When the injected text in the session transcript is inspected
  Then it names that note as its source
```

### 793a2e No re-injection

Requirement: 793a2e (formerly REC-5) The System MUST NOT inject an item (a note or a chunk) already
present in the receiving agent's context, whether the System injected it earlier, the agent retrieved it,
or it arrived by any other means. An item shown without its content, as a path and score only, does not
count as present.

Rationale: an item the agent already has adds only noise. Each subagent is a separate context, so a
parent's items do not spare the subagent. Compaction removes content, so a removed item can be useful
again. An item the agent saw only as a path and score has not given it the content.

#### 793a2e1 UAT Scenarios

```gherkin
Scenario: 793a2e1a Note delivered earlier is not delivered again
  Given note X was delivered to the working agent at its prompt
  And the session has not been compacted
  When the agent's next two tool results match X again
  Then the session transcript shows X's content delivered only once

Scenario: 793a2e1c Note the agent retrieved itself is not injected
  Given the working agent ran a command that displayed note X in the transcript
  When a later prompt matches X
  Then the transcript shows X's content delivered by no injection

Scenario: 793a2e1d A parent's note is still injected into a subagent
  Given note X is in the parent agent's transcript
  When the parent dispatches a subagent whose task matches X
  Then the subagent's transcript shows X's content delivered before its first response or tool call

Scenario: 793a2e1e A note removed by compaction can be injected again
  Given note X was delivered and the session was then compacted so that X's content no longer appears in what the agent sees
  When a later prompt matches X
  Then the transcript shows X's content delivered again

Scenario: 793a2e1f A chunk shown without content is delivered with it
  Given a lazily-cut payload showed chunk Y to the agent as a path and score only
  When a later moment's query returns Y
  Then the transcript shows Y's content delivered

Scenario: 793a2e1g A resumed session's restored content counts as present
  Given note X was delivered in a session that was then ended and resumed
  And the resumed session's transcript shows X's content restored
  When a later prompt matches X
  Then the transcript shows no second delivery of X

Scenario: 793a2e1h A chunk delivered earlier is not delivered again
  Given chunk Y was delivered with its content to the working agent
  And the session has not been compacted
  When a later moment's query returns Y
  Then the session transcript shows Y's content delivered only once
```

### 793a2h Recall depth per moment

Requirement: 793a2h At each recall moment the System MUST choose a depth using the triggers of the
instruction-based path ([792a19b Recall escalates from glance to deep](../../specs/product/ambient-guidance.md#792a19b-recall-escalates-from-glance-to-deep)): deep when the decision is weighty or irreversible, when glance
flagged a gap it could not resolve, or when the decision turns on a recently surfaced standard
([792a14d Glance escalates to deep for a recent standard](../../specs/product/recall.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)); glance otherwise.

Rationale: depth is the existing lever that matches effort to the decision, so the System spends deep
recall only where the working agent would.

#### 793a2h1 UAT Scenarios

```gherkin
Scenario: 793a2h1a A routine decision uses glance
  Given a prompt that asks to rename a local variable
  When the operator reads the log for that moment
  Then it records depth glance and about 3 query phrases

Scenario: 793a2h1b An irreversible decision uses deep
  Given a prompt that asks to force-push over a shared branch
  When the operator reads the log for that moment
  Then it records depth deep and 10 query phrases

Scenario: 793a2h1c An unresolved gap uses deep
  Given a glance recall whose result flags a gap it cannot resolve
  When the operator reads the log for that moment
  Then it records that glance escalated and that the moment ran at depth deep

Scenario: 793a2h1d A recently surfaced standard uses deep
  Given the recent-activity channel of the query surfaced a note saying "use X going forward"
  And a decision that turns on honoring it
  When the operator reads the log for that moment
  Then it records depth deep

Scenario: 793a2h1e Applying a convention stays glance
  Given a decision that only applies an established convention
  When the operator reads the log for that moment
  Then it records depth glance
```

### 793a2i Recall for every prompt regardless of source

Requirement: 793a2i The System MUST apply prompt-time recall to every prompt regardless of its source
(human, automated notification, scheduled trigger, message from another session).

Rationale: memory is always worth checking, whatever produced the prompt.

#### 793a2i1 UAT Scenarios

```gherkin
Scenario Outline: 793a2i1a Non-human prompt receives recall
  Given a note whose situation matches the prompt
  And the note is not yet in the transcript
  When <source> delivers that prompt to the working agent
  Then the session transcript shows that note's content delivered before the agent's first response or tool call

  Examples:
    | source                             |
    | an automated task notification     |
    | a scheduled trigger                |
    | a message from another session     |
```

### 793a2j Volume equals the depth's output

Requirement: 793a2j At each recall moment (a prompt, a tool result or a subagent dispatch) the System
MUST inject the items (notes and chunks) that the chosen depth's queries return for that moment,
incidental items included, and MUST NOT inject more than those. Incidental items are the explore picks
([792a7i Half of the notes are explored](../../specs/product/memory-query.md#792a7i-half-of-the-notes-are-explored)), the ride-along notes ([792a7n Superseding notes ride along](../../specs/product/memory-query.md#792a7n-superseding-notes-ride-along)) and the newest chunks
([792a7o The newest chunks](../../specs/product/memory-query.md#792a7o-the-newest-chunks)); they are injected even when no item matches directly. At glance the queries are one
`engram query` with about 3 phrases ([792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths), [792a14f One unified query](../../specs/product/recall.md#792a14f-one-unified-query)). At deep they are one query with 10
phrases from the ten angles ([792a14e Ten phrases from ten angles](../../specs/product/recall.md#792a14e-ten-phrases-from-ten-angles)). At either depth they also include the recommendation check
([792a14o An emergent recommendation is checked against the vault](../../specs/product/recall.md#792a14o-an-emergent-recommendation-is-checked-against-the-vault)) when it runs. When glance escalates to deep at a moment, the moment's
queries are those of both depths. Each query returns the payload of [792a7b The payload](../../specs/product/memory-query.md#792a7b-the-payload) under its defaults. The
System MUST omit an item already present in the receiving agent's context ([793a2e No re-injection](#793a2e-no-re-injection)) and MUST
NOT otherwise reduce that output.

Rationale: automatic recall must give the working agent what the instruction-based path gives it, no
more and no less, so that its effect is attributable to who initiates it and not to what it delivers.

#### 793a2j1 UAT Scenarios

```gherkin
Scenario Outline: 793a2j1a Delivered items equal the depth's output
  Given a recall moment of kind <moment> at depth <depth>
  And the operator runs the depth's queries on the same vault with the phrases the log records for that moment
  When the session transcript is compared with those query results
  Then every item the queries return is delivered in the transcript, except items whose content the transcript already shows to that agent
  And no other item is delivered

  Examples:
    | moment            | depth  |
    | prompt            | glance |
    | prompt            | deep   |
    | tool result       | glance |
    | tool result       | deep   |
    | subagent dispatch | glance |
    | subagent dispatch | deep   |

Scenario: 793a2j1b A vault holding more than 20 notes about the topic
  Given a vault holding more than 20 notes about the prompt's topic
  When the human submits the prompt at depth glance
  Then the notes delivered are those the glance query returns under its default result limit and content budget

Scenario: 793a2j1c Incidental items are delivered with the direct matches
  Given the glance query for a prompt returns a direct match, an explore pick, a ride-along note and the newest chunks
  When the human submits the prompt
  Then the session transcript shows all four kinds of item delivered

Scenario: 793a2j1d An escalated moment delivers both depths' output
  Given a glance recall whose result flags a gap it cannot resolve, so that the moment escalates to deep
  And the operator runs the glance queries and the deep queries with the phrases the log records
  When the session transcript is compared with those query results
  Then every item either depth returns is delivered, except items already shown to that agent
  And no other item is delivered
```

## 793a3 Added: automatic-memory (automatic capture)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a3a Evaluate every completed turn

Requirement: 793a3a (formerly CAP-1) After each completed turn, in both top-level and subagent
sessions, the System MUST evaluate the turn for lessons without the working agent requesting it. The
System, not the working agent, MUST judge what is worth saving, and MUST write each captured lesson
directly to the vault.

Rationale: the working agent's context stays free of memory evaluation, and subagent sessions are
where lessons occur too. A lesson here is any corrected failure, which is wider than the four kinds in
[792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson), because a mistake the agent catches and fixes itself is as reusable as one the user
catches.

#### 793a3a1 UAT Scenarios

```gherkin
Scenario: 793a3a1a User correction is captured without agent action
  Given a session whose configuration contains no memory instructions
  When the user says "no, use the targ build system, not go test directly"
  And the turn completes
  Then a feedback note stating that rule exists in the vault
  And it was written directly, not left as a pending offer for review
  And the transcript shows the working agent invoked no learn command
  And the transcript shows no request to evaluate, approve or write the note

Scenario: 793a3a1b Explicit save-request
  When the user says "remember that releases are tagged from main"
  And the turn completes
  Then a fact note stating that exists in the vault

Scenario: 793a3a1c A subagent's own guess overturned by evidence is captured
  Given the working agent dispatched a subagent
  And the subagent stated a conclusion that a test result then overturned
  And the subagent corrected its conclusion
  When the subagent's turn completes
  Then a note stating the corrected conclusion exists in the vault
  And the subagent's transcript shows it invoked no learn command

Scenario: 793a3a1d A correction from the dispatching agent is captured
  Given the working agent dispatched a subagent
  And the dispatching agent tells the subagent "no, run targ test, not go test"
  When the subagent's turn completes
  Then a note stating that rule exists in the vault

Scenario: 793a3a1e Capture survives the subagent's context being discarded
  Given a subagent whose turn contained a lesson
  When the subagent finishes and its context is discarded
  Then a note stating the lesson exists in the vault
```

### 793a3b Confirmed lessons only

Requirement: 793a3b (formerly CAP-2) The System MUST NOT capture a hypothesis, a routine success with
no resolved uncertainty, or a bare pleasantry as a lesson.

#### 793a3b1 UAT Scenarios

```gherkin
Scenario: 793a3b1a Pleasantry ignored
  When the user says "thanks, great work"
  And the turn completes
  Then no note is captured

Scenario: 793a3b1b Unconfirmed guess ignored
  Given the working agent said "maybe the cause is a race"
  And nothing confirmed it
  When the turn completes
  Then no note is captured for that statement

Scenario: 793a3b1c Routine success ignored
  Given a turn in which a command succeeded as expected with no uncertainty resolved
  When the turn completes
  Then no note is captured
```

### 793a3c No duplicates

Requirement: 793a3c (formerly CAP-3) Before writing, the System MUST check for an existing note
stating the same lesson and MUST NOT write a duplicate, including when two subagents capture the same
lesson concurrently. When an equivalent note exists, the System SHOULD reinforce it instead.

#### 793a3c1 UAT Scenarios

```gherkin
Scenario: 793a3c1a Existing note already covers the lesson
  Given a note stating "never amend pushed commits"
  When the user repeats that correction and the turn completes
  Then no new note is created

Scenario: 793a3c1b A lesson reported to the parent yields one note
  Given a subagent's turn contained a lesson and the System captured it
  When the subagent reports that lesson to its parent and the parent's turn completes
  Then the vault holds exactly one note stating the lesson

Scenario: 793a3c1c Two subagents capture the same lesson concurrently
  Given two subagents whose turns complete at the same time, each containing the same lesson
  When both captures finish
  Then the vault holds exactly one note stating the lesson
```

### 793a3d Provenance

Requirement: 793a3d (formerly CAP-4) Every note the System writes automatically MUST record that it
was written automatically, and the session and turn it came from.

#### 793a3d1 UAT Scenarios

```gherkin
Scenario: 793a3d1a Provenance present
  Given an automatically captured note in the vault
  When its source field is inspected
  Then it identifies automatic capture, the session id, and the turn id
```

### 793a3f Capture does not delay the session

Requirement: 793a3f (formerly CAP-6) Capture MUST NOT delay the working agent's response or the
processing of the next prompt.

#### 793a3f1 UAT Scenarios

```gherkin
Scenario: 793a3f1a Slow capture does not stall the session
  Given capture evaluation takes 20 seconds
  When a turn completes and the user submits the next prompt immediately
  Then the next prompt is processed without waiting for the capture to finish
  And the working agent's response to the completed turn appeared without waiting for the capture
```

### 793a3g Deep recall's writes are the System's

Requirement: 793a3g Where deep recall writes to the vault, namely the coverage writes
([792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster)) and the persisted conclusion ([792a14p A sound conclusion is persisted as a derived note](../../specs/product/recall.md#792a14p-a-sound-conclusion-is-persisted-as-a-derived-note)), the System's capture MUST perform those
writes, and the working agent MUST NOT be asked to. Each such write MUST carry the provenance of
[793a3d Provenance](#793a3d-provenance), and a persisted conclusion MUST also have its source marked as derived.

Rationale: the working agent's context stays free of memory writes at deep moments as it does at
capture.

#### 793a3g1 UAT Scenarios

```gherkin
Scenario: 793a3g1a A coverage write is made without the working agent
  Given a deep recall moment whose results include a cluster that no existing note covers
  When the moment completes
  Then a note covering the cluster exists in the vault
  And its source field identifies automatic capture, the session id and the turn id
  And the transcript shows the working agent ran no learn, amend or write-memory command
  And the transcript shows no request to the working agent to write the note

Scenario: 793a3g1b A sound conclusion is persisted without the working agent
  Given a deep recall moment whose closing synthesis reaches a sound conclusion that no note states
  When the moment completes
  Then a derived note stating the conclusion exists in the vault
  And its source field identifies automatic capture, the session id and the turn id, and marks the source as derived
  And the transcript shows the working agent ran no learn, amend or write-memory command
  And the transcript shows no request to the working agent to write the note
```

### 793a3h Capture rate exceeds the baseline

Requirement: 793a3h The System's capture rate on lesson-worthy moments MUST exceed 5.6% on the point
estimate.

Rationale: the instruction-based path's capture rate is the standard the System has to beat; a rate at or
below it gives no benefit.

#### 793a3h1 UAT Scenarios

```gherkin
Scenario: 793a3h1a Capture rate exceeds the baseline
  Given a labeled set of at least 60 lesson-worthy moments from real sessions
  When the System runs over those sessions
  Then the share of those moments it captures is greater than 5.6%
```

### 793a3i Precision is graded independently

Requirement: 793a3i A separate grader inside the System, independent of the capture judgment, MUST
grade every note the System captures at turn completion ([793a3a Evaluate every completed turn](#793a3a-evaluate-every-completed-turn)) as a valid lesson or not,
and the System MUST log each verdict, the number of notes graded and the resulting precision.

Rationale: the judge that decides what to save cannot also be the measure of whether it was right.

#### 793a3i1 UAT Scenarios

```gherkin
Scenario: 793a3i1a Precision is reported with the number graded
  Given the System captured 30 notes from completed turns
  When the grader has judged each as valid or not valid
  Then the operator can read each of the 30 verdicts, the number graded (30) and the resulting precision

Scenario: 793a3i1b The grader does not defer to the capture judgment
  Given a captured note that states an unconfirmed guess as a lesson
  When the grader judges it
  Then the recorded verdict is not valid
  And the log holds the grader's verdict and the capture decision as separate entries
```

### 793a3j Direct writes go to the local vault

Requirement: 793a3j The System's direct writes MUST go to the local vault, including when the vault has
a parent ([parent-sync](../../specs/product/parent-sync.md)) or is served ([vault-serving](../../specs/product/vault-serving.md)). Offers to a parent MUST follow normal
note-writing behavior ([792a21n A local write is offered to the parent](../../specs/product/parent-sync.md#792a21n-a-local-write-is-offered-to-the-parent)).

Rationale: the System writes as any local write does, so a parent or a server sees nothing different
from an ordinary note.

#### 793a3j1 UAT Scenarios

```gherkin
Scenario: 793a3j1a A captured lesson lands locally and is offered normally
  Given a vault with a configured parent
  When the System captures a lesson
  Then the note exists in the local vault
  And the parent receives an offer for it exactly as it does for a note written by `engram learn`

Scenario: 793a3j1b A served vault receives the write locally
  Given a vault that is also served to other callers
  When the System captures a lesson
  Then the note exists in that local vault
```

### 793a3k Used notes are activated

Requirement: 793a3k At turn-end capture ([793a3a Evaluate every completed turn](#793a3a-evaluate-every-completed-turn)) the System MUST judge which of the notes it injected
during the turn the working agent went on to use, and MUST activate each of those notes, so that its
recency metadata is bumped as when recall activates a used note ([792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)). It MUST NOT
activate an injected note the agent did not use, and MUST NOT activate a chunk. It MUST activate a used
note from the parent vault by its path, so that a pending local copy appears ([792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths)),
and MUST leave that copy pending.

Rationale: a note's recency tracks whether it is useful. Bumping every injected note would keep an ignored
one fresh, and bumping none would let a note the agent applies fade.

#### 793a3k1 UAT Scenarios

```gherkin
Scenario: 793a3k1a A note the agent used is activated
  Given the System injected note X into a turn
  And the working agent then applied X's rule in its work
  When the turn completes
  Then X's recency metadata shows it was used in this session
  And the log records the activation of X

Scenario: 793a3k1b An injected note the agent ignored is not activated
  Given the System injected note Z into a turn
  And the working agent's work in the turn does not apply or cite Z
  When the turn completes
  Then Z's recency metadata is unchanged
  And the log records that Z was not activated

Scenario: 793a3k1c An injected chunk is not activated
  Given the System injected a chunk into a turn and the working agent used its content
  When the turn completes
  Then the log records no activation for the chunk

Scenario: 793a3k1d A used parent note is pulled down and left pending
  Given the System injected a note from the parent vault and the working agent used it
  When the turn completes
  Then a pending local copy of that note appears in the vault
  And the copy is still pending after the System's work for the turn
```

## 793a4 Added: automatic-memory (operator control and safety)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a4b Failure isolation

Requirement: 793a4b (formerly CTL-2) If the System fails, times out, or is unavailable, the working
session MUST continue unaffected, with no error surfaced to the working agent and no lost user input.
The failure MUST be logged for the operator.

#### 793a4b1 UAT Scenarios

```gherkin
Scenario: 793a4b1a Vault unavailable
  Given the vault cannot be read
  When the human submits a prompt
  Then the session proceeds normally with no injection
  And the operator-accessible log records the failure

Scenario: 793a4b1b Evaluation crashes
  Given capture evaluation raises an error
  When a turn completes
  Then the session continues and the next prompt is processed normally
  And the operator-accessible log records the failure

Scenario: 793a4b1c No error reaches the agent
  Given the System times out at a recall moment
  When the session transcript is inspected
  Then it shows no error message from the System and no user input missing
```

### 793a4c Memory purview only

Requirement: 793a4c (formerly CTL-3) The System MUST NOT itself make or override task decisions, tool
permissions, tool arguments, model choice, or the user's prompt text. Its only effects are injecting
reference notes and writing vault notes.

#### 793a4c1 UAT Scenarios

```gherkin
Scenario: 793a4c1a Tool calls and permissions unchanged
  Given a session in which the System injected notes and captured a lesson
  When the session's tool calls and permission decisions are inspected
  Then every tool call ran with exactly the arguments the agent issued
  And every permission decision was made by the user's permission settings
  And the model in use is the one the session was configured with

Scenario: 793a4c1b The prompt is unchanged and the note is separate
  Given a prompt that matches a note
  When the session transcript is inspected
  Then the user's prompt text is exactly as the user typed it
  And the injected note appears as separate content attributed to its source
```

### 793a4d Sensitive content

Requirement: 793a4d (formerly CTL-4) The System MUST NOT write credentials, tokens, or secrets present
in a transcript into a note.

#### 793a4d1 UAT Scenarios

```gherkin
Scenario: 793a4d1a Secret in correction is not persisted
  Given the user's correction text includes an API key
  When the lesson is captured
  Then the written note contains the rule but not the key
```

### 793a4f No recall or capture in its own evaluation sessions

Requirement: 793a4f The System MUST NOT recall into, or capture from, the sessions it runs to evaluate
(to judge capture, to grade precision or to choose recall). The System MUST log the id of each session
it starts itself, marked as a System session.

Rationale: an evaluation session that triggers the System would recurse and would write notes about
the System's own judging.

#### 793a4f1 UAT Scenarios

```gherkin
Scenario: 793a4f1a Evaluation sessions are left alone
  Given the System ran a capture judgment and a precision grading for a turn
  When the operator reads the log and the vault
  Then the log lists each evaluation session's id as a System session
  And the log has no recall or capture decision for any evaluation session
  And no note's provenance names an evaluation session

Scenario: 793a4f1b The session that chooses depth and phrases is left alone
  Given the System started a session to choose the depth and the query phrases for a recall moment
  When the operator reads the log
  Then the log lists that session's id as a System session
  And the log has no recall decision for that session
```

### 793a4g Active whenever installed

Requirement: 793a4g The System MUST be active whenever the engram integration is installed and enabled, with
no operating mode to select. When the integration is disabled or uninstalled, the System MUST inject
nothing, capture nothing and record nothing.

Rationale: one state to reason about, and turning the integration off is the way to stop the System.

#### 793a4g1 UAT Scenarios

```gherkin
Scenario: 793a4g1a A fresh install is active
  Given a fresh install of the integration with no configuration
  When a session runs with a prompt that matches a note and contains a correction
  Then the transcript shows the note's content delivered
  And a note stating the correction exists in the vault

Scenario: 793a4g1b A disabled integration is inert
  Given the integration is installed and then disabled
  When a session runs with prompts that match notes and contain a correction
  Then the transcript shows no injected content
  And no note is written to the vault
  And no log entry is written

Scenario: 793a4g1c An uninstalled integration is inert
  Given the integration is installed and then uninstalled
  When a session runs with prompts that match notes and contain a correction
  Then the transcript shows no injected content
  And no note is written to the vault
  And no log entry is written
```

## 793a5 Added: automatic-memory (independence, coexistence, and measurement)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a5a Independence from agent compliance

Requirement: 793a5a (formerly IND-1) The System's recall and capture MUST function with no
memory-related instructions present in the working agent's configuration.

#### 793a5a1 UAT Scenarios

```gherkin
Scenario: 793a5a1a Bare configuration
  Given a project and user configuration with no engram instructions
  When the scenarios in 793a2a1 (formerly REC-1) and 793a3a1 (formerly CAP-1) are run
  Then they pass
```

### 793a5b Coexistence without double-firing

Requirement: 793a5b (formerly IND-2) Where the existing instruction-based path ([ambient-guidance](../../specs/product/ambient-guidance.md)) is
also active, the rules of [793a2e No re-injection](#793a2e-no-re-injection) and [793a3c No duplicates](#793a3c-no-duplicates) MUST hold across both paths, and the
working agent SHOULD NOT perform redundant lookups or saves for a moment the System already handled.

#### 793a5b1 UAT Scenarios

```gherkin
Scenario: 793a5b1a Both paths active
  Given the prompt-based memory instructions and the System are both active
  And the working agent's own lookup for a moment displayed note X
  When the System's recall for that moment returns X
  Then the transcript shows X's content delivered by no System injection
  And a lesson that both paths capture yields one note in the vault
```

### 793a5c Environments without the capability

Requirement: 793a5c (formerly IND-3) In harnesses where the System cannot run, the existing
instruction-based path ([ambient-guidance](../../specs/product/ambient-guidance.md)) MUST continue to work unchanged.

#### 793a5c1 UAT Scenarios

```gherkin
Scenario: 793a5c1a Unsupported harness
  Given a harness without support for the System
  When a session runs with the existing instruction-based path
  Then its behavior is identical to its behavior before the System existed
```

### 793a5e Auditability and measurability

Requirement: 793a5e (formerly NFR-2) Every recall decision (injected or none, with the depth, whether
glance escalated to deep, the phrases of each query, the items and their scores), every capture decision
(written, reinforced, or discarded, with reason) and every activation decision (activated or not, per
injected note) MUST be logged, so effect on behavior can be measured and the operator can audit it. The
log MUST be available to the operator by session id. Each entry MUST carry the session id, the turn id
and the moment. The System MUST also record, per moment, the volume injected and the latency added. A
moment with no injection MUST be recorded with volume 0. The System MUST record all of this whenever it is
active, and no configuration MUST switch the recording off while the System is active.

Rationale: the volume and latency record is the evidence any later change to the volume and latency
requirements ([793a2j Volume equals the depth's output](#793a2j-volume-equals-the-depths-output), [793a5g Latency per recall moment](#793a5g-latency-per-recall-moment)) rests on.

#### 793a5e1 UAT Scenarios

```gherkin
Scenario: 793a5e1a Decision log joins to outcomes
  Given a completed session
  When the operator queries the log by session id
  Then each moment has a recorded decision carrying the session id, the turn id and the moment
  And each injected note can be matched to the later actions of the working agent

Scenario: 793a5e1b Volume and latency recorded per moment
  Given a completed session with moments at which notes were injected
  When the operator queries the log by session id
  Then each moment shows the volume injected and the latency added

Scenario: 793a5e1c A moment with no injection is recorded
  Given a session with a moment at which nothing was injected
  When the operator queries the log by session id
  Then that moment is recorded with volume 0

Scenario: 793a5e1d A discarded capture is recorded with its reason
  Given a turn whose only candidate lesson the System judges not worth saving
  When the operator queries the log by session id
  Then the turn has a capture decision of discarded with a reason

Scenario: 793a5e1e Measurement cannot be switched off while the System is active
  Given the integration is installed and the operator's configuration tries to turn measurement off
  When a session runs
  Then the log still records each moment with its volume and latency

Scenario: 793a5e1f An escalated moment records both depths
  Given a glance recall whose result flags a gap it cannot resolve
  When the operator queries the log by session id
  Then the moment records that glance escalated to deep and the phrases of each query it ran
```

### 793a5f Effect on outcomes

Requirement: 793a5f (formerly NFR-3) In a controlled comparison on lesson-bearing tasks, runs with
the System active SHOULD apply a relevant existing lesson more often than runs without it.

Rationale: injection only pays if the injected memory is applied; this comparison is the test of that.

#### 793a5f1 UAT Scenarios

```gherkin
Scenario: 793a5f1a Treatment beats control
  Given a set of tasks each with a relevant existing note
  And N = 20 runs per arm with the System disabled and enabled, fresh process each
  When the rate of applying the relevant note is compared
  Then the enabled arm's rate is higher by more than the noise floor measured between repeat runs of the same arm
```

### 793a5g Latency per recall moment

Requirement: 793a5g At every recall moment (a prompt, a tool result or a subagent dispatch) the latency the
System adds MUST NOT exceed the latency of the instruction path's full lookup for that moment: generating
the query phrases plus running the chosen depth's queries ([793a2j Volume equals the depth's output](#793a2j-volume-equals-the-depths-output)), and the
queries of both depths when glance escalates to deep. The comparison MUST be made on the same vault and
input.

Rationale: recall must cost no more than the whole lookup the working agent would otherwise perform,
including forming its query.

#### 793a5g1 UAT Scenarios

```gherkin
Scenario Outline: 793a5g1a Added latency stays within the instruction path's lookup latency
  Given a recorded recall moment of kind <moment>
  And the operator performs the instruction path's lookup for that moment on the same vault and input, generating the phrases and running the depth's queries
  When the latency added in the log is compared with the time that lookup took
  Then the latency added does not exceed that time

  Examples:
    | moment            |
    | prompt            |
    | tool result       |
    | subagent dispatch |

Scenario: 793a5g1b An escalated moment is compared with both depths
  Given a recorded recall moment at which glance escalated to deep
  And the operator performs the instruction path's lookup on the same vault and input, generating the phrases and running the queries of both depths
  When the latency added in the log is compared with the time that lookup took
  Then the latency added does not exceed that time
```
