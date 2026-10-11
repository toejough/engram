# Parallel Memory Agent - Product Spec

## 793a1 Problem Statement

Joe, and the LLM agents that work in engram-enabled sessions, get value from engram's memory only if
two things happen at the right moments: relevant vault notes are **consulted** before the working
agent acts, and lessons worth keeping are **saved** after they occur. The memory skills,
[recall](../../specs/product/recall.md) and [learn](../../specs/product/learn.md), already say what to consult, what to save and when
([ambient-guidance](../../specs/product/ambient-guidance.md)). But the working agent has to run them, which depends on its noticing, unprompted, that the moment is
worth a lookup or a save. The agent must "remember to remember."

Measured evidence of the failure (issue #793):

- Capture: the learn behavior fired for about 5% of moments worth learning (4 of 71 = 5.6% baseline;
  3 of 66 = 4.5% after the last prose fix).
- Delegation: recalled memory reached 0 of 4 subagents at dispatch (#739).
- Compliance: in a bare-CLAUDE.md headless trial, the instruction to run a memory query before the
  first tool call was refused as prompt injection 6 of 6 times (#790).
- Prose fixes plateau: repeated rewording of the instructions has not moved these rates.

Memory use is gated on the working agent's own metacognition, which is the least reliable component
and competes with the task for attention. The need is for the skills to run at the right moments
regardless of whether the working agent thinks to run them, and out of the working agent's context, without
degrading the working session. This change makes the System a delegate for the existing memory skills: it
performs recall and learn as their own specifications define them, on the working agent's behalf, and puts
in front of the agent only what the skill would have. This document specifies only that delegation, the
deliberate differences from the skills, and what is new. The delegation applies to the Claude Code harness; every other harness keeps the existing skills, guidance and LESSONS contract, and its working agent keeps running recall and learn ([793a4h Installed by engram update](#793a4h-installed-by-engram-update), [793a5c Environments without the capability](#793a5c-environments-without-the-capability)).

Terms used below:

- **Working agent:** the model doing the user's task in a session, including any subagents it
  dispatches.
- **System:** the capability this document specifies. It runs alongside the working agent and acts for it.
- **Vault** and **note:** the engram note store and one entry in it (fact, feedback, runbook, or Q&A).
- **Moment:** a point at which recall or learn fires: the moments [ambient-guidance](../../specs/product/ambient-guidance.md) defines, every
  prompt ([793a2i Recall for every prompt regardless of source](#793a2i-recall-for-every-prompt-regardless-of-source)), a subagent's dispatch
  and the orchestrator's decision to dispatch
  ([793a2c Moments are noticed without the agent](#793a2c-moments-are-noticed-without-the-agent)). A **recall moment** is one at which recall fires.
- **Depth:** how thorough one recall is, `glance` or `deep`, as defined by [792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths).
- **Lesson:** any corrected failure, including an agent's own mid-task mistake that evidence
  overturned, plus a correction, an explicit save-request and a confirmed approach. It is wider than
  the four kinds in [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson) ([793a3n Any corrected failure is a lesson](#793a3n-any-corrected-failure-is-a-lesson)).
- **Injection:** placing note content, or a chunk stub, in front of the working agent without the agent requesting it.
- **Chunk stub:** a chunk shown as its id and score only, with no content, as a lazy query prints it ([792a7s Lazy chunks](../../specs/product/memory-query.md#792a7s-lazy-chunks)).
- **Integration:** what `engram update` installs for the System in the Claude Code harness ([793a4h Installed by engram update](#793a4h-installed-by-engram-update)). It is installed per harness, so a condition "while the Integration is installed" is evaluated per harness and per agent session: a session in a harness without it, such as Pi, follows the canonical branch, and its working agent keeps running recall and learn ([793a5c Environments without the capability](#793a5c-environments-without-the-capability)).
- **Context:** the content currently available to one agent, a top-level agent or one subagent.
  Content removed by compaction is no longer present. A subagent's context is separate from its
  parent's. A resumed session's restored content is present.
- **Operator:** the human who configures and audits the System.

Requirement keywords follow RFC 2119.

Sections [793a2 automatic-memory (delegation of recall)](#793a2-added-automatic-memory-delegation-of-recall) through [793a5 automatic-memory (coexistence and measurement)](#793a5-added-automatic-memory-coexistence-and-measurement) add requirements to one new capability.
[793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed) modifies the lessons contract and the capabilities that name its LESSONS line, and [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed) modifies the nodes that tell the working agent to run recall or learn. The draft's identifiers map to the current ones in [793b2 ID mapping](793b_decision-log.md#793b2-id-mapping) and [793b132 L7: draft identifiers are mapped here, not in Requirement lines](793b_decision-log.md#793b132-l7-draft-identifiers-are-mapped-here-not-in-requirement-lines).

## 793a2 Added: automatic-memory (delegation of recall)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a2a Recall is performed for the working agent

Requirement: 793a2a The System MUST perform recall on the working agent's behalf at each moment at which recall fires
([792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments), [792a19b Recall escalates from glance to deep](../../specs/product/ambient-guidance.md#792a19b-recall-escalates-from-glance-to-deep)), at every prompt ([793a2i Recall for every prompt regardless of source](#793a2i-recall-for-every-prompt-regardless-of-source)), at a subagent's dispatch and at the orchestrator's decision to dispatch ([793a2c Moments are noticed without the agent](#793a2c-moments-are-noticed-without-the-agent)), exactly as [recall](../../specs/product/recall.md) defines it: at the depth that
[792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths) and [792a19b Recall escalates from glance to deep](../../specs/product/ambient-guidance.md#792a19b-recall-escalates-from-glance-to-deep) select, by the same procedure, with the same queries, writes and
activation, except that use is inferred as [793a3k Use is inferred from the transcript](#793a3k-use-is-inferred-from-the-transcript) says. The System MUST put in front of the working agent what recall would have
put in front of it, before the agent's next response or tool call, subject only to
[793a2e No re-injection](#793a2e-no-re-injection), and MUST put nothing else of recall's in front of it.

Rationale: recall already says what to consult and when, so the work moves out of the working agent's context and
nothing about what recall does changes.

#### 793a2a1 UAT Scenarios

```gherkin
Scenario: 793a2a1a Recall happens without the agent running it
  Given a vault containing a note whose situation matches "renaming a Go package"
  And a session whose configuration contains no memory instructions
  When the human submits "rename package foo to bar across the repo"
  Then the session transcript shows that note's content delivered before the agent's first response or tool call
  And the transcript shows the agent ran no memory command
  And the log for that moment records the recall that was performed, with its depth, phrases, items and scores

Scenario: 793a2a1d Recall's vault writes happen without the agent
  Given a deep recall moment whose results include a cluster that no existing note addresses
  When the moment completes
  Then a note covering the cluster exists in the vault
  And the transcript shows the working agent ran no learn, amend or write-memory command
  And the transcript shows no request to the working agent to write the note
```

### 793a2c Moments are noticed without the agent

Requirement: 793a2c The System MUST notice each moment at which recall or learn fires without the working agent signalling it.
A subagent's dispatch is one such moment: the System MUST make what recall returns for the subagent's task available to that subagent.
The orchestrator's decision to dispatch a unit is another, of its own kind: at that decision, before the orchestrator routes the unit, the System MUST perform recall for the unit and put what it returns in front of the orchestrator, including the tier-performance evidence for the unit's kind of work, so that the orchestrator routes on it.

Rationale: the evidence that route's starting tier rests on reaches the orchestrator only if it is delivered before the unit is routed. How the System detects that the orchestrator is about to dispatch, and what the unit's kind of work is, is for the technical specification.

#### 793a2c1 UAT Scenarios

```gherkin
Scenario: 793a2c1a Subagent receives relevant notes
  Given a note matching "writing Go tests in this repo"
  When the working agent dispatches a subagent with the task "add tests for internal/cli"
  Then the subagent's transcript shows that note's content delivered before its first response or tool call
  And the working agent's dispatch text did not mention the note

Scenario: 793a2c1b A recall moment mid-task is noticed
  Given a note matching the work the working agent is about to declare done
  When the working agent reaches the point of declaring the work done
  Then the session transcript shows that note's content delivered before the declaration
  And the transcript shows the agent ran no memory command

Scenario: 793a2c1c A learn moment is noticed
  Given a build fails and the transcript then shows its non-obvious cause
  When the cause is known
  Then a note stating the cause exists in the vault
  And the transcript shows the working agent invoked no learn command and was asked for none

Scenario: 793a2c1d Tier evidence is delivered before a dispatch is routed
  Given tier-performance evidence for a kind of work that shows a cheaper tier failed
  When the orchestrator decides to dispatch a unit of that kind
  Then the transcript shows that evidence delivered before the orchestrator routes the unit
  And the log records one recall moment for the decision, of kind dispatch decision
  And the transcript shows the orchestrator ran no memory command
```

### 793a2d Attributable presentation

Requirement: 793a2d Injected items MUST identify their source: a note by its file name and a chunk by its id.

Rationale: an agent that can see where an item came from can weigh it and look it up, and a chunk's id is what it fetches the chunk by ([792a7t Reading one chunk](../../specs/product/memory-query.md#792a7t-reading-one-chunk)). Imperative memory,
such as a runbook or a skill, is a legitimate injection; whether injected memory is applied is
tested by [793a5f Effect on outcomes](#793a5f-effect-on-outcomes).

#### 793a2d1 UAT Scenarios

```gherkin
Scenario: 793a2d1a Injected content names its source
  Given an injection of note "474.2026-07-25.charge-one-reviewer-to-execute..."
  When the injected text in the session transcript is inspected
  Then it gives that note's file name as its source

Scenario: 793a2d1b Injected chunk names its id
  Given an injection of the chunk with id "session-ab12#turn-7"
  When the injected text in the session transcript is inspected
  Then it gives that chunk's id and score
```

### 793a2e No re-injection

Requirement: 793a2e The System MUST NOT inject an item (a note or a chunk) already
present in the receiving agent's context, whether the System injected it earlier, the agent retrieved it,
or it arrived by any other means. A chunk stub already shown counts as present, so a chunk stub MUST NOT be
injected again. A note whose text changed after the agent received it, other than in its sources or its recency metadata, is not present in its changed form.

Rationale: an item the agent already has adds only noise. Each subagent is a separate context, so a
parent's items do not spare the subagent. Compaction removes content, so a removed item can be useful
again. A stub already shown tells the agent that the chunk exists and where to fetch it, so repeating it adds only noise.

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

Scenario: 793a2e1g A resumed session's restored content counts as present
  Given note X was delivered in a session that was then ended and resumed
  And the resumed session's transcript shows X's content restored
  When a later prompt matches X
  Then the transcript shows no second delivery of X

Scenario: 793a2e1h A chunk whose content the agent fetched is not injected
  Given the working agent ran a command that displayed chunk Y's content in the transcript
  When a later moment's query returns Y
  Then the transcript shows Y delivered by no injection

Scenario: 793a2e1i A chunk stub already shown is not injected again
  Given chunk Y was injected into the working agent's context as a chunk stub
  And the session has not been compacted
  When a later moment's query returns Y
  Then the session transcript shows Y's id and score only once

Scenario: 793a2e1j An amended note is delivered in its changed form
  Given note Q was delivered to the working agent earlier in the session
  And Q's text was changed afterwards by an amend that added a claim
  When a later recall moment returns Q
  Then the transcript shows Q's amended content delivered

Scenario: 793a2e1k A note whose sources or recency changed is still present
  Given note X was delivered to the working agent earlier in the session
  And X then gained chunk sources and a later activation, with its text unchanged
  And the session has not been compacted
  When a later recall moment returns X
  Then the transcript shows X's content delivered only once
```

### 793a2i Recall for every prompt regardless of source

Requirement: 793a2i In addition to the moments at which recall fires ([793a2a Recall is performed for the working agent](#793a2a-recall-is-performed-for-the-working-agent)), the System MUST perform recall, in every working session (excluding System sessions per [793a4f No recall or capture in its own sessions](#793a4f-no-recall-or-capture-in-its-own-sessions)), at every prompt regardless of its source
(human, automated notification, scheduled trigger, message from another session). The prompt that dispatches a
subagent is the moment of [793a2c Moments are noticed without the agent](#793a2c-moments-are-noticed-without-the-agent), not a separate prompt moment.

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

Scenario: 793a2i1b A subagent's dispatch prompt is one moment, of the dispatch kind
  Given a note matching a subagent's task
  When the working agent dispatches the subagent
  Then the log records one recall moment for the dispatch, of kind subagent dispatch
  And the log records no recall moment of kind prompt for the subagent's dispatch prompt
  And the subagent's transcript shows the note's content delivered once
```

### 793a2k The working agent does not run recall or learn

Requirement: 793a2k While the Integration is installed, the System MUST perform recall and learn so that the working agent is not required to run either, and MUST NOT ask the working agent to run them or to evaluate, approve or write any memory.

Rationale: the point of the delegation is to take memory work, and its bookkeeping, out of the working agent's context.

#### 793a2k1 UAT Scenarios

```gherkin
Scenario: 793a2k1a The agent runs neither skill and is asked for neither
  Given the Integration is installed
  And a session whose configuration contains no memory instructions
  When the human submits a prompt that matches a note and contains a correction
  Then the transcript shows the note's content delivered and a note stating the correction in the vault
  And the transcript shows the working agent ran no recall or learn skill and no memory command
  And the transcript shows no request from the System to run one, or to evaluate, approve or write a note
```

### 793a2l The chunk index is swept after the turn

Requirement: 793a2l The System MUST sweep the chunk index asynchronously after a turn completes, MUST NOT make the working agent wait for a sweep, and MUST NOT run the sweeps that [792a14b The index is swept once per session](../../specs/product/recall.md#792a14b-the-index-is-swept-once-per-session) and [792a15b The sweep comes first](../../specs/product/learn.md#792a15b-the-sweep-comes-first) place before a skill's first query or first step.

Rationale: a sweep is slow work that the working agent need not wait for, and the turn that has just completed is what the next sweep must index.

#### 793a2l1 UAT Scenarios

```gherkin
Scenario: 793a2l1a The sweep follows the turn and nothing waits for it
  Given a turn that completes with content not yet in the chunk index
  When the human submits the next prompt immediately
  Then the next prompt is processed without waiting for a sweep
  And the log records a sweep for the completed turn
  And a later query returns a chunk from that turn

Scenario: 793a2l1b No sweep runs before a moment's recall
  Given a session's first moment
  When the operator reads the log for that moment
  Then it records no sweep before the moment's recall
```

## 793a3 Added: automatic-memory (delegation of learn)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a3a Learn is performed for the working agent

Requirement: 793a3a The System MUST perform learn on the working agent's behalf, in every working session, top-level or subagent
(excluding System sessions per [793a4f No recall or capture in its own sessions](#793a4f-no-recall-or-capture-in-its-own-sessions)), at each moment at which learn fires
([792a19c Learn fires at the correction moment](../../specs/product/ambient-guidance.md#792a19c-learn-fires-at-the-correction-moment), [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson)) and, in place of the closing learn, after each completed turn, exactly as
[learn](../../specs/product/learn.md) defines it: the same judgment of what is worth saving, the same single-note capture
([792a15c A correction moment gets a single-note capture](../../specs/product/learn.md#792a15c-a-correction-moment-gets-a-single-note-capture)) and the same notes handed to [write-memory](../../specs/product/write-memory.md). The System MUST write each captured lesson directly to the vault, with no
involvement of the working agent.

Rationale: the working agent's context stays free of memory evaluation, and subagent sessions are where lessons occur too.

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
```

### 793a3m Lessons from subagent sessions are captured directly

Requirement: 793a3m While the Integration is installed, the System MUST capture lessons that occur in subagent sessions directly from those sessions, as
[793a3a Learn is performed for the working agent](#793a3a-learn-is-performed-for-the-working-agent) does for the top-level session, so that no lesson depends on the subagent or its dispatching agent reporting it. The LESSONS line
of the [lessons contract](../../specs/product/lessons-contract.md) is replaced while the Integration is installed, as the operations of
[793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed) state. When two subagent sessions contain the same lesson, the System MUST write one note for it.

Rationale: a lesson reported by the subagent is lost whenever the report omits it, and the System sees the subagent's own session. Capturing from
the session also leaves nothing for the parent to collect.

#### 793a3m1 UAT Scenarios

```gherkin
Scenario: 793a3m1a A correction from the dispatching agent is captured
  Given the working agent dispatched a subagent
  And the dispatching agent tells the subagent "no, run targ test, not go test"
  When the subagent's turn completes
  Then a note stating that rule exists in the vault
  And the subagent's transcript shows it invoked no learn command

Scenario: 793a3m1b Capture survives the subagent's context being discarded
  Given a subagent whose turn contained a lesson
  When the subagent finishes and its context is discarded
  Then a note stating the lesson exists in the vault

Scenario: 793a3m1c No report of lessons is asked for or collected
  Given the working agent dispatched a subagent whose turn contained a lesson
  When the subagent returns and the parent's turn completes
  Then the vault holds exactly one note stating the lesson
  And neither the subagent's instructions nor its report carry a LESSONS line

Scenario: 793a3m1d Two subagents contain the same lesson
  Given two subagents whose turns complete at the same time, each containing the same lesson
  When both captures finish
  Then the vault holds exactly one note stating the lesson
```

### 793a3n Any corrected failure is a lesson

Requirement: 793a3n The System MUST treat any corrected failure as a lesson, including an agent's own mid-task mistake that evidence overturned, in addition to the lessons that
[792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson) defines.

Rationale: a mistake the agent catches and fixes itself is as reusable as one the user catches.

#### 793a3n1 UAT Scenarios

```gherkin
Scenario: 793a3n1a A subagent's own guess overturned by evidence is captured
  Given the working agent dispatched a subagent
  And the subagent stated a conclusion that a test result then overturned
  And the subagent corrected its conclusion
  When the subagent's turn completes
  Then a note stating the corrected conclusion exists in the vault
  And the subagent's transcript shows it invoked no learn command

Scenario: 793a3n1b The top-level agent's own mistake overturned by evidence is captured
  Given the working agent stated a cause for a failing test
  And a later command output overturned that cause and the agent corrected it
  When the turn completes
  Then a note stating the corrected cause exists in the vault
```

### 793a3o Please's lessons audit and surfacing question are performed by the System

Requirement: 793a3o While the Integration is installed, the System MUST perform please's lessons audit and its which-artifact-should-have-surfaced-it question for a please run, from the transcript and as part of capture, exactly as [792a18p The closing learn is preceded by a lessons audit](../../specs/product/please.md#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit) and [792a18q A lesson about to be captured asks what should have surfaced it](../../specs/product/please.md#792a18q-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it) define them, so that the outcome of the audit and of the question, and any reword, appear in the vault and the log. The working agent MUST NOT be asked to perform either.

Rationale: the audit and the question exist to feed a capture, and with the capture the System's, so are they. The per-item list is read from the vault and the log, because please's closing report no longer carries it; the user sees the counts of the events it produced in the summary ([793a5h A memory-activity summary is shown to the user](#793a5h-a-memory-activity-summary-is-shown-to-the-user), [793c40 Does the user still see please's audit list?](793c_open-questions.md#793c40-does-the-user-still-see-pleases-audit-list)).

#### 793a3o1 UAT Scenarios

```gherkin
Scenario: 793a3o1a The audit's outcome is in the log and the vault
  Given the Integration is installed and a please run with one gate FAIL verdict and one commit named CORRECTION
  And the vault holds a note that captures the lesson of the verdict and none that captures the lesson of the commit
  When the run's turn completes
  Then the log records an audit entry for each item, naming the existing note for the verdict and the note the System wrote for the commit
  And the note written for the commit is a reversal
  And the transcript shows the orchestrator ran no audit and no learn, and its closing report carries no audit list

Scenario: 793a3o1b An item with no lesson is logged with its reason
  Given the Integration is installed and a please run with one gate FAIL verdict that taught nothing reusable
  When the run's turn completes
  Then the log records "no lesson: <why>" for the item
  And no note is written for it

Scenario: 793a3o1c The surfacing question's outcome is in the vault and the log
  Given the Integration is installed and a please run containing a lesson that an existing note covers
  And that note did not surface because its situation line does not match how the moment presented
  When the run's turn completes
  Then the log records which existing note should have surfaced the lesson and that its situation line was reworded
  And that note's situation line in the vault now matches how the moment presented
  And the vault holds no second note on the lesson
  And the transcript shows the orchestrator asked no which-artifact-should-have-surfaced-it question and ran no vault search for a covering note

Scenario: 793a3o1d A repository change is never filed automatically
  Given the Integration is installed and a please run containing a lesson that a change shipped from the repository could prevent
  When the run's turn completes
  Then no issue or change for it was filed automatically
```

### 793a3p Route's evidence writes are performed by the System

Requirement: 793a3p While the Integration is installed, the System MUST write route's tagged evidence note for each dispatch and keep the work-kind's one aggregate note, exactly as [792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note) and [792a17l Each work-kind keeps one aggregate note](../../specs/product/route.md#792a17l-each-work-kind-keeps-one-aggregate-note) define them, reading the work-kind, tier, model and outcome from the dispatch's row in route's report ([792a17i Each dispatch is recorded in a mini-report row](../../specs/product/route.md#792a17i-each-dispatch-is-recorded-in-a-mini-report-row)) in the transcript. The orchestrator MUST NOT write either note. When a dispatch's row is not in the transcript, the System MUST write nothing for that dispatch and MUST log that the row was missing.

Rationale: the evidence notes are measurements of dispatches, built from the row the orchestrator already produces, so the orchestrator writes no memory.

#### 793a3p1 UAT Scenarios

```gherkin
Scenario: 793a3p1a A dispatch's evidence note is written from its report row
  Given the Integration is installed and a report row for a dispatch of work-kind K at tier mid with outcome pass
  When the row appears in the transcript
  Then a fact tagged work-kind/K, tier/mid and outcome/pass exists in the vault, and its object states the verdict, reason, escalation, duration and cost from the row
  And the transcript shows the orchestrator handed nothing to write-memory and ran no memory command

Scenario: 793a3p1b The work-kind's aggregate is kept from the same row
  Given the Integration is installed and a report row for a dispatch of work-kind K
  When the row appears in the transcript
  Then the aggregate note for K exists in the vault with its tally including that dispatch and a link to its evidence note
  And the transcript shows the orchestrator ran no engram learn or amend command

Scenario: 793a3p1c A missing row is logged and nothing is written
  Given the Integration is installed and a dispatch that resolved without its report row in the transcript
  When the dispatch's turn completes
  Then no evidence note and no aggregate change for that dispatch exists in the vault
  And the log records that the row was missing
```

### 793a3d Provenance

Requirement: 793a3d Every note the System writes automatically MUST record, in addition to the source the skill defines for the note, that it was written automatically, and the session and turn it came from.

#### 793a3d1 UAT Scenarios

```gherkin
Scenario: 793a3d1a Provenance present
  Given an automatically captured note in the vault
  When its source field is inspected
  Then it states the source the skill defines for that note
  And it also states that the note was written automatically, the session id, and the turn id
```

### 793a3k Use is inferred from the transcript

Requirement: 793a3k The System MUST infer from the session transcript which of the notes it injected the working agent went on to use, for example by quoting or citing the note,
following its steps, or taking the action it prescribes, because it cannot ask the agent, and MUST activate each of those notes as recall activates a used note
([792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)), in place of recall's own activation, which relies on the agent's declaration. The System MUST log each activation verdict,
activated or not, with its evidence, and MUST NOT activate an injected note for which the transcript shows no
evidence of use. This requirement governs only the activation of injected notes in place of [792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated): the activation that recall's coverage write performs ([792a14j Coverage is judged by the agent and written once per cluster](../../specs/product/recall.md#792a14j-coverage-is-judged-by-the-agent-and-written-once-per-cluster)) is done as recall defines it and is not a verdict under this requirement.

Rationale: a note's recency tracks whether it is useful. The System has no declaration from the agent of
what it used, and asking for one would put memory bookkeeping back into the agent's context, so it infers use from the
transcript, where the evidence is.

#### 793a3k1 UAT Scenarios

```gherkin
Scenario: 793a3k1a A note the transcript shows used is activated
  Given the System injected note X into a turn
  And the transcript shows the working agent quoting X's rule and taking the action X prescribes
  When the turn completes
  Then X's recency metadata shows it was used in this session
  And the log records the activation of X with that evidence

Scenario: 793a3k1b An injected note with no evidence of use is not activated
  Given the System injected note Z into a turn
  And the transcript shows no quote or citation of Z, no step of Z followed and no action Z prescribes
  When the turn completes
  Then Z's recency metadata is unchanged
  And the log records the verdict that Z was not activated

Scenario: 793a3k1f A coverage write's activation is recall's own
  Given the Integration is installed and a deep recall moment in which a candidate note is judged to cover a cluster
  When the moment completes
  Then the candidate's recency metadata shows it was activated by recall's coverage write
  And the log records that activation as the coverage write's, with no use-inference verdict under the inference requirement
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
grade every note the System captures as a valid lesson or not,
and the System MUST log each verdict, the number of notes graded and the resulting precision.

Rationale: the judge that decides what to save cannot also be the measure of whether it was right.

#### 793a3i1 UAT Scenarios

```gherkin
Scenario: 793a3i1a Precision is reported with the number graded
  Given the System captured 30 notes
  When the grader has judged each as valid or not valid
  Then the operator can read each of the 30 verdicts, the number graded (30) and the resulting precision

Scenario: 793a3i1b The grader does not defer to the capture judgment
  Given a captured note that states an unconfirmed guess as a lesson
  When the grader judges it
  Then the recorded verdict is not valid
  And the log holds the grader's verdict and the capture decision as separate entries
```

### 793a3l Activation accuracy is graded independently

Requirement: 793a3l A separate grader inside the System, independent of the activation judgment, MUST sample the
activation verdicts of [793a3k Use is inferred from the transcript](#793a3k-use-is-inferred-from-the-transcript) (activated or not), grade each sampled verdict against the session transcript as
correct or not, and the System MUST log each graded verdict, the sample size and the resulting accuracy.

Rationale: inferring use can miss use the agent never states and can mistake coincidence for use, and the judge that
makes the verdicts cannot also be the measure of whether they were right.

#### 793a3l1 UAT Scenarios

```gherkin
Scenario: 793a3l1a A sampled activation verdict is graded
  Given the System made 40 activation verdicts over completed turns
  And the grader sampled 10 of them
  When the grader has judged each sampled verdict as correct or not
  Then the operator can read each of the 10 graded verdicts, the sample size (10) and the resulting accuracy

Scenario: 793a3l1b The grader does not defer to the activation judgment
  Given a note the System activated although the transcript shows no use of it
  And that verdict is in the sample
  When the grader judges it
  Then the recorded grade is not correct
  And the log holds the grader's grade and the activation verdict as separate entries
```

## 793a4 Added: automatic-memory (operator control and safety)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a4b Failure isolation

Requirement: 793a4b While the Integration is installed, if the System fails, times out, or is unavailable, the working
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

Requirement: 793a4c The System MUST NOT itself make or override tool permissions,
tool arguments, model choice, or the user's prompt text. Its only effect on the working agent's context is the
items it injects; outside that context it shows the user the memory-activity summary ([793a5h A memory-activity summary is shown to the user](#793a5h-a-memory-activity-summary-is-shown-to-the-user)), writes and activates vault notes with their sidecars, refreshes the chunk index, pulls down pending copies of parent notes and
makes offers to the parent ([792a21n A local write is offered to the parent](../../specs/product/parent-sync.md#792a21n-a-local-write-is-offered-to-the-parent)), writes its log, and starts System sessions
([793a4f No recall or capture in its own sessions](#793a4f-no-recall-or-capture-in-its-own-sessions)).

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

Scenario: 793a4c1c The only effect on the working agent's context is the injected items
  Given a session in which the System injected notes and captured a lesson
  When the session transcript is inspected
  Then the only content the System added to the working agent's context is the injected items

Scenario: 793a4c1d Effects outside the session are the System's own writes
  Given a vault with a configured parent
  And a session in which the System captured a lesson and activated an injected note from the parent vault
  When the files changed during the session outside the session transcript are listed
  Then the changes are the vault notes written or activated with their sidecars, the chunk index, the pending parent copies, the offers to the parent, the System's log and the sessions the System started
```

### 793a4d Sensitive content

Requirement: 793a4d The System MUST NOT write credentials, tokens, or secrets present
in a transcript into a note.

#### 793a4d1 UAT Scenarios

```gherkin
Scenario: 793a4d1a Secret in correction is not persisted
  Given the user's correction text includes an API key
  When the lesson is captured
  Then the written note contains the rule but not the key
```

### 793a4f No recall or capture in its own sessions

Requirement: 793a4f The System MUST NOT recall into, or capture from, the sessions it runs itself (to perform recall and learn, to judge activation, or to grade precision and activation accuracy).
The System MUST log the id of each session it starts itself, marked as a System session.

Rationale: a System session that triggers the System would recurse and would write notes about
the System's own work.

#### 793a4f1 UAT Scenarios

```gherkin
Scenario: 793a4f1a System sessions are left alone
  Given the System ran a capture judgment, an activation judgment and the gradings of both for a turn
  When the operator reads the log and the vault
  Then the log lists each such session's id as a System session
  And the log has no recall or capture decision for any System session
  And no note's provenance names a System session

Scenario: 793a4f1b The session that performs a recall is left alone
  Given the System started a session to perform recall for a moment
  When the operator reads the log
  Then the log lists that session's id as a System session
  And the log has no recall decision for that session
```

### 793a4g Active whenever installed

Requirement: 793a4g The System MUST be active in every session of a harness in which the Integration is installed, with
no operating mode to select and no step beyond installing it. The Integration is installed per harness, and a session of any other harness MUST be unaffected by the System: nothing is injected into it or captured from it, and its working agent keeps running recall and learn itself as [793a5c Environments without the capability](#793a5c-environments-without-the-capability) says. When the Integration has been uninstalled from a harness, the System MUST
inject nothing, capture nothing and record nothing in that harness's sessions.

Rationale: one state to reason about. `engram update` is how the System is turned on ([793a4h Installed by engram update](#793a4h-installed-by-engram-update)), and uninstalling is how it
is turned off, through the general uninstall that is specified separately (#849).

#### 793a4g1 UAT Scenarios

```gherkin
Scenario: 793a4g1a An Integration installed by update is active
  Given the Integration was installed by engram update, with no other configuration
  When a session runs with a prompt that matches a note and contains a correction
  Then the transcript shows the note's content delivered
  And a note stating the correction exists in the vault

Scenario: 793a4g1c An uninstalled Integration is inert
  Given the Integration is installed and then uninstalled
  When a session runs with prompts that match notes and contain a correction
  Then the transcript shows no injected content
  And no note is written to the vault
  And no log entry is written

Scenario: 793a4g1d One machine, a delegated session and a session that is not
  Given a machine with Claude Code and Pi on which engram update has been run
  And a note whose situation matches a prompt that contains a correction
  When the same prompt is submitted in a Claude Code session and in a Pi session
  Then the Claude Code transcript shows the note's content delivered and the transcript shows the agent ran no memory command
  And a note stating the correction exists in the vault, written without the Claude Code agent
  And the Pi transcript shows no content injected by the System, and the Pi agent ran the recall skill itself before acting and invoked the learn skill itself for the correction
  And the log has no moment for the Pi session
```

### 793a4h Installed by engram update

Requirement: 793a4h `engram init` MUST install the Integration, what the System needs to run, for the Claude Code harness, as part of setting up everything idempotently: the vault, and each harness's integration (for Claude Code the Integration; for every other harness, Pi today, the skills, the guidance, the shim and the LESSONS contract's text). Running `engram init` again MUST change nothing that is already set up. `engram update` MUST run `engram init` as part of updating, for each harness it updates ([792a11g The harnesses that are updated](../../specs/product/install-and-update.md#792a11g-the-harnesses-that-are-updated)), extending the install of [install-and-update](../../specs/product/install-and-update.md). On Claude Code with the Integration installed, `engram init` and `engram update` MUST NOT deploy the recall and learn skills or the guidance, guidance imports and shim that tell the agent to run them, and `engram update` MUST remove them where an earlier install deployed them. For every other harness, they MUST keep being deployed, and the working agent in that harness MUST keep running recall and learn. An
existing install of engram MUST have the System in Claude Code after its next `engram update`, with no further step.
`engram update --dry-run` ([792a11f Previewing an update](../../specs/product/install-and-update.md#792a11f-previewing-an-update)) MUST install none of it.

Rationale: init is the one place that sets up a machine, and update runs it, so a first-time install, a later update and an existing install all end in the same state, and rerunning either is safe. A harness without the Integration keeps the setup that works there; on Claude Code the agent is not given skills and guidance for work the System does ([793c46 On Claude Code with the Integration installed, does engram update still deploy the recall and learn skills and guidance files?](793c_open-questions.md#793c46-on-claude-code-with-the-integration-installed-does-engram-update-still-deploy-the-recall-and-learn-skills-and-guidance-files), [793c47 Does "init" mean `engram init` or the first-time install by `engram update`?](793c_open-questions.md#793c47-does-init-mean-engram-init-or-the-first-time-install-by-engram-update)). Init depends on [793c48 How is the widening of `engram init` (change #795) sequenced with this change?](793c_open-questions.md#793c48-how-is-the-widening-of-engram-init-change-795-sequenced-with-this-change).

#### 793a4h1 UAT Scenarios

```gherkin
Scenario: 793a4h1a Update installs the Integration for the harness it updates
  Given ~/.claude exists, ~/.pi does not, and the System can run in Claude Code
  When the user runs engram update
  Then a Claude Code session runs with the System active
  And ~/.pi is not created

Scenario: 793a4h1b An existing install gains the System on its next update
  Given an engram install made before the System existed, with no Integration
  When the user runs engram update
  And a session runs with a prompt that matches a note and contains a correction
  Then the transcript shows the note's content delivered
  And a note stating the correction exists in the vault

Scenario: 793a4h1c A preview installs none of the Integration
  Given a machine with no Integration
  When the user runs engram update --dry-run
  And a session runs with a prompt that matches a note
  Then the transcript shows no injected content

Scenario: 793a4h1d One update serves both harnesses differently
  Given ~/.claude and ~/.pi both exist
  When the user runs engram update
  Then a Claude Code session runs with the System active, and its agent ran no memory command
  And the Pi agent's configuration still holds the recall and learn skills, the guidance and the shim, and in a Pi session the agent runs recall itself

Scenario: 793a4h1e Claude Code gets no recall or learn skills or guidance
  Given an earlier install deployed the recall and learn skills and their guidance to Claude Code
  When the user runs engram update
  Then the Claude Code agent's configuration holds no recall or learn skill and no import of the guidance or the shim that tell the agent to run them
  And the user cannot invoke /recall or /learn in a Claude Code session

Scenario: 793a4h1f A first-time update runs init
  Given a machine on which engram has never been installed and ~/.claude exists
  When the user runs engram update
  Then the vault exists and a Claude Code session runs with the System active

Scenario: 793a4h1g Init sets up everything and is idempotent
  Given a machine with Claude Code and Pi
  When the user runs engram init twice
  Then after the first run the vault exists, Claude Code has the Integration and Pi has the skills, the guidance and the shim
  And the second run changes nothing and says so
```

## 793a5 Added: automatic-memory (coexistence and measurement)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a5c Environments without the capability

Requirement: 793a5c In every harness in which the Integration is not installed, which is every harness other than Claude Code, any Claude Code install not yet updated, and every harness where the System cannot run, the existing
instruction-based path ([ambient-guidance](../../specs/product/ambient-guidance.md)) MUST continue to work unchanged, and the working agent in that harness MUST keep running recall and learn.

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
  And the System injects nothing into it, captures nothing from it and logs nothing for it
```

### 793a5e Auditability and measurability

Requirement: 793a5e Every recall decision (injected or none, with the depth, whether
glance escalated to deep, the phrases of each query, the items and their scores), every capture decision
(what learn did with the lesson, or that it was discarded, with the reason) and every activation decision (activated or not, per
injected note, with its evidence) MUST be logged, so effect on behavior can be measured and the operator can audit it. The
log MUST be available to the operator by session id. Each entry MUST carry the session id, the turn id
and the moment. The System MUST also record, per moment, the volume injected, as the number of items and the number of bytes, and the latency added. A
moment with no injection MUST be recorded with volume 0. The System MUST record all of this whenever it is
active, and MUST NOT provide, or honor, any setting that switches the recording off while the System is active.

Rationale: the volume and latency record is the evidence any later change to what the System delivers and how long it takes rests on.

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
  Given the Integration is installed and the operator tries to turn measurement off with an environment variable or config key that would disable logging
  When a session runs
  Then the log still records each moment with its volume and latency

Scenario: 793a5e1f An escalated moment records both depths
  Given a glance recall whose result flags a gap it cannot resolve
  When the operator queries the log by session id
  Then the moment records that glance escalated to deep and the phrases of each query it ran
```

### 793a5f Effect on outcomes

Requirement: 793a5f In a controlled comparison on lesson-bearing tasks, runs with
the System active SHOULD apply a relevant existing lesson more often than runs without it.

Rationale: injection only pays if the injected memory is applied; this comparison is the test of that.

#### 793a5f1 UAT Scenarios

```gherkin
Scenario: 793a5f1a Treatment beats control
  Given a set of tasks each with a relevant existing note
  And N = 20 runs per arm, one without the Integration installed and one with it installed, fresh process each
  When the rate of applying the relevant note is compared
  Then the installed arm's rate is higher by more than the noise floor measured between repeat runs of the same arm
```

### 793a5g No added delay

Requirement: 793a5g The System's work MUST NOT add delay to the working agent beyond what the agent would spend performing the same skill itself.
Latency added is the time the working agent waits on the System: before it receives a moment's recall, or before its next prompt is processed.
Work the working agent does not wait on adds none.

Rationale: moving the skills out of the working agent only pays if it is never slower than the agent doing them.

#### 793a5g1 UAT Scenarios

```gherkin
Scenario Outline: 793a5g1a Recall adds no more delay than the agent's own recall
  Given a recorded recall moment of kind <moment>
  And the operator performs the same recall inline, as the agent would, on a copy of the vault as it was at that moment, with the same input
  When the latency added in the log is compared with the time that inline recall took
  Then the latency added does not exceed that time

  Examples:
    | moment            |
    | prompt            |
    | subagent dispatch |
    | mid-task moment   |
    | dispatch decision |

Scenario: 793a5g1c Slow capture does not stall the session
  Given capture evaluation takes 20 seconds
  When a turn completes and the user submits the next prompt immediately
  Then the next prompt is processed without waiting for the capture to finish
  And the working agent's response to the completed turn appeared without waiting for the capture
```

### 793a5h A memory-activity summary is shown to the user

Requirement: 793a5h After each completed top-level turn, once the System's work for that turn (capture included) has finished, the System MUST show the user, outside the working agent's context, the counts of memory events recorded since its previous summary, across the session and its subagent sessions: items surfaced (injected), notes used (activated), notes added, notes amended, captures discarded, and evidence notes written. A turn with no memory events MUST show nothing. The summary MUST NOT enter the working agent's context. The per-item details and the reason for each event stay in the log ([793a5e Auditability and measurability](#793a5e-auditability-and-measurability)) and the vault.

Rationale: the user keeps visibility of the memory work the System took over, including what please's closing report used to show, without the working agent relaying it.

#### 793a5h1 UAT Scenarios

```gherkin
Scenario: 793a5h1a The summary shows the counts after a turn with events
  Given the Integration is installed and a turn in which the System injected 3 items, activated 2 notes, added 1 note, amended 1 note, discarded 1 capture and wrote 1 evidence note
  When the turn completes
  Then the user is shown a summary with 3 items surfaced, 2 notes used, 1 note added, 1 note amended, 1 capture discarded and 1 evidence note written

Scenario: 793a5h1b Nothing is shown after a turn without events
  Given the Integration is installed and a turn in which the System injected, activated, added, amended, discarded and wrote nothing
  When the turn completes
  Then the user is shown no memory-activity summary

Scenario: 793a5h1c The summary is not in the agent's context
  Given the Integration is installed and a turn after which a summary was shown to the user
  When the user submits the next prompt and the transcript of the working agent's session is inspected
  Then the summary text appears nowhere in the transcript
  And the working agent's next response does not refer to it

Scenario: 793a5h1d The counts match the log
  Given the Integration is installed and a turn after which a summary was shown to the user
  When the operator queries the log by session id for the events recorded since the previous summary
  Then those events, counted by kind, equal the counts in the summary
  And the per-item details and the reason for each event are in the log and the vault, not in the summary

Scenario: 793a5h1e The counts cover the events since the previous summary
  Given the Integration is installed and a summary was shown after one turn
  When the next turn completes with 1 note added and nothing else
  Then the user is shown a summary with 1 note added and every other count 0 or omitted
  And no event counted in the previous summary is counted again

Scenario: 793a5h1f A subagent's events are counted in the next summary
  Given the Integration is installed and a turn in which a subagent's session produced 1 note added
  When the top-level turn completes
  Then the user is shown a summary that counts that note among the notes added

Scenario: 793a5h1g The summary follows the capture
  Given the Integration is installed and a turn whose capture takes 20 seconds and adds 1 note
  When the turn completes and the user submits the next prompt immediately
  Then the next prompt is processed without waiting for the capture
  And the summary is shown only after the capture has finished, and counts that note
```

## 793a6 Modified: lessons-contract (the LESSONS line is replaced while the Integration is installed)

Canonical targets: [lessons-contract](../../specs/product/lessons-contract.md) and the capabilities that name its LESSONS line: [route](../../specs/product/route.md), [please](../../specs/product/please.md) and [ambient-guidance](../../specs/product/ambient-guidance.md). Each operation is a Modified operation against exactly one canonical node, conditional on the Integration, because the contract still holds in harnesses and installs without the Integration ([793a5c Environments without the capability](#793a5c-environments-without-the-capability)). Each target's ID is preserved. The replacement text carries the Integration branch first, then the target's text unchanged for an install without the Integration. The condition is evaluated per harness and per agent session: a session in a harness without the Integration, such as Pi, follows the target's text unchanged. Each operation's scenario group ending in 1 covers its Integration branch. The group ending in 2 holds one Modified operation for each canonical scenario of the target: that scenario's text prefixed with "Given the Integration is not installed", so that the branch for an install without the Integration keeps its canonical coverage. No scenario in this section restates a canonical scenario under a new ID.

### 793a6a Modified: A completion report ends with a LESSONS line

Target: [792a16a A completion report ends with a LESSONS line](../../specs/product/lessons-contract.md#792a16a-a-completion-report-ends-with-a-lessons-line)

Requirement: 792a16a While the Integration is installed, a subagent MUST NOT be asked to end its report with a `LESSONS:` line, and its report MUST NOT be required to carry one. Where the Integration is not installed: Every subagent dispatched through route, whether a fork, a fresh-context agent or a workflow, MUST end its final report with a `LESSONS:` line that is `LESSONS: none`, or one to three one-line lessons separated by commas. Each lesson MUST be a confirmed correction, a surprising finding or a validated approach, summarized in one line. A subagent MUST NOT decide for itself that a lesson becomes a vault note.

Rationale: A subagent's lessons are captured from its own session, so a line offered by the subagent would only repeat them.

#### 793a6a1 UAT Scenarios

```gherkin
Scenario: 793a6a1a A subagent is not asked for a LESSONS line
  Given the Integration is installed and the working agent dispatches a subagent through route
  When the subagent's instructions and its final report are inspected
  Then the instructions do not ask for a LESSONS line
  And the report is not required to end with one
```

#### 793a6a2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a16a A completion report ends with a LESSONS line](../../specs/product/lessons-contract.md#792a16a-a-completion-report-ends-with-a-lessons-line): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6a2a | [792a16a1a A subagent with lessons reports them](../../specs/product/lessons-contract.md#792a16a1-uat-scenarios) | Scenario: 792a16a1a A subagent with lessons reports them<br>Given the Integration is not installed<br>And a subagent whose work contained a confirmed correction and a surprising finding<br>When it writes its completion report<br>Then the report ends with LESSONS: &lt;lesson1&gt;, &lt;lesson2&gt; |
| 793a6a2b | [792a16a1b A subagent with nothing to report says so](../../specs/product/lessons-contract.md#792a16a1-uat-scenarios) | Scenario: 792a16a1b A subagent with nothing to report says so<br>Given the Integration is not installed<br>And a subagent whose work was straightforward execution with no novel finding<br>When it writes its completion report<br>Then the report ends with LESSONS: none |
| 793a6a2c | [792a16a1c At most three lessons](../../specs/product/lessons-contract.md#792a16a1-uat-scenarios) | Scenario: 792a16a1c At most three lessons<br>Given the Integration is not installed<br>And a subagent that learned five things<br>When it writes the line<br>Then the line carries at most three one-line lessons |

### 793a6b Modified: Every dispatch is told to report lessons

Target: [792a16b Every dispatch is told to report lessons](../../specs/product/lessons-contract.md#792a16b-every-dispatch-is-told-to-report-lessons)

Requirement: 792a16b While the Integration is installed, the handoff that route builds MUST NOT include the completion-report contract or an instruction to recall first, and its other contents are unchanged. Where the Integration is not installed: The handoff that route builds for every dispatch MUST include, among the items it must hand over (exact files and paths, acceptance checks, do-not-touch bounds and the subagent's recall-first instruction), the completion-report contract: the subagent's final report MUST end with a `LESSONS:` line in the format of [792a16a A completion report ends with a LESSONS line](../../specs/product/lessons-contract.md#792a16a-a-completion-report-ends-with-a-lessons-line). The instruction MUST be prescriptive, and MUST be written out even for a fork, which inherits session context.

Rationale: The contract's only purpose was to get lessons back, and the System captures them from the subagent's session.

#### 793a6b1 UAT Scenarios

```gherkin
Scenario: 793a6b1a The handoff omits the LESSONS mandate
  Given the Integration is installed and route builds the handoff for a fresh-context agent
  When the handoff is written
  Then it names the exact files, acceptance checks and bounds
  And it does not include a LESSONS mandate or a recall-first instruction
```

#### 793a6b2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a16b Every dispatch is told to report lessons](../../specs/product/lessons-contract.md#792a16b-every-dispatch-is-told-to-report-lessons): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6b2a | [792a16b1a A fresh-context dispatch carries the mandate](../../specs/product/lessons-contract.md#792a16b1-uat-scenarios) | Scenario: 792a16b1a A fresh-context dispatch carries the mandate<br>Given the Integration is not installed<br>And route selects a general-purpose agent and builds its instructions<br>When the handoff is written<br>Then it includes the completion-report contract with the LESSONS line |
| 793a6b2b | [792a16b1b A fork's handoff carries it too](../../specs/product/lessons-contract.md#792a16b1-uat-scenarios) | Scenario: 792a16b1b A fork's handoff carries it too<br>Given the Integration is not installed<br>And route dispatches a fork<br>When the handoff is written<br>Then the fork's instructions state the LESSONS mandate in writing |
| 793a6b2c | [792a16b1c It follows the task description](../../specs/product/lessons-contract.md#792a16b1-uat-scenarios) | Scenario: 792a16b1c It follows the task description<br>Given the Integration is not installed<br>And a complex multi-step task<br>When route writes the handoff<br>Then the task and its acceptance checks are followed by the completion-report contract |

### 793a6c Modified: A missing line is asked for once

Target: [792a16c A missing line is asked for once](../../specs/product/lessons-contract.md#792a16c-a-missing-line-is-asked-for-once)

Requirement: 792a16c While the Integration is installed, the orchestrator MUST NOT send a follow-up that asks a subagent for a missing `LESSONS:` line, and MUST NOT record `LESSONS: none` in its place. Where the Integration is not installed: When a returned report does not end with a `LESSONS:` line, the orchestrator MUST send exactly one targeted follow-up that asks only for the line, and MUST NOT re-ask a second time. A line supplied in response MUST be treated like an on-time line. When the line is still missing after the one follow-up, the orchestrator MUST record `LESSONS: none` and proceed without blocking.

Rationale: There is no required line to be missing.

#### 793a6c1 UAT Scenarios

```gherkin
Scenario: 793a6c1a A report without a LESSONS line is not re-asked
  Given the Integration is installed and a subagent report that ends without a LESSONS line
  When the orchestrator receives it
  Then it sends no follow-up asking for the line
  And it proceeds without waiting
```

#### 793a6c2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a16c A missing line is asked for once](../../specs/product/lessons-contract.md#792a16c-a-missing-line-is-asked-for-once): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6c2a | [792a16c1a A missing line is re-asked](../../specs/product/lessons-contract.md#792a16c1-uat-scenarios) | Scenario: 792a16c1a A missing line is re-asked<br>Given the Integration is not installed<br>And a subagent report that ends without a LESSONS line<br>When the orchestrator receives it<br>Then it sends one follow-up asking only for the line in the format LESSONS: &lt;lesson1&gt;, &lt;lesson2&gt; or LESSONS: none |
| 793a6c2b | [792a16c1b The answer counts like an on-time line](../../specs/product/lessons-contract.md#792a16c1-uat-scenarios) | Scenario: 792a16c1b The answer counts like an on-time line<br>Given the Integration is not installed<br>And a subagent that supplies the line in response to the follow-up<br>When the orchestrator collects it<br>Then it is recorded and judged the same as an on-time line |
| 793a6c2c | [792a16c1c There is no second re-ask](../../specs/product/lessons-contract.md#792a16c1-uat-scenarios) | Scenario: 792a16c1c There is no second re-ask<br>Given the Integration is not installed<br>And a subagent that still gives no LESSONS line after the follow-up<br>When the orchestrator proceeds<br>Then it records LESSONS: none for that dispatch and does not ask again |

### 793a6d Modified: Every line is recorded as it returns

Target: [792a16d Every line is recorded as it returns](../../specs/product/lessons-contract.md#792a16d-every-line-is-recorded-as-it-returns)

Requirement: 792a16d While the Integration is installed, the orchestrator following please MUST NOT keep a running list of `LESSONS:` lines. Where the Integration is not installed: The orchestrator following please MUST record the `LESSONS:` line of every dispatched unit verbatim, including `LESSONS: none`, in a running list for the cycle at the moment each unit's result comes back and before it moves to the next unit, wherever in the workflow the dispatch lands. Judging a line MUST NOT remove or rewrite it, and the list MUST NOT carry marks of what was captured. The please gates MUST NOT block or filter the collection: a line is recorded even when its work is rejected at a gate.

Rationale: There are no lines to collect.

#### 793a6d1 UAT Scenarios

```gherkin
Scenario: 793a6d1a No running list of LESSONS lines is kept
  Given the Integration is installed and please dispatches three units in sequence
  When each returns
  Then the orchestrator keeps no list of LESSONS lines
  And the lessons the units' sessions contained are in the vault
```

#### 793a6d2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a16d Every line is recorded as it returns](../../specs/product/lessons-contract.md#792a16d-every-line-is-recorded-as-it-returns): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6d2a | [792a16d1a Lines are kept in order and verbatim](../../specs/product/lessons-contract.md#792a16d1-uat-scenarios) | Scenario: 792a16d1a Lines are kept in order and verbatim<br>Given the Integration is not installed<br>And please dispatches three units in sequence<br>When each returns<br>Then the running list holds the three LESSONS lines in order and exactly as written |
| 793a6d2b | [792a16d1b A none is still recorded](../../specs/product/lessons-contract.md#792a16d1-uat-scenarios) | Scenario: 792a16d1b A none is still recorded<br>Given the Integration is not installed<br>And a unit reports LESSONS: none<br>When the orchestrator records it<br>Then the line is an entry in the list and no learn fires for it |
| 793a6d2c | [792a16d1c A gate does not filter the list](../../specs/product/lessons-contract.md#792a16d1-uat-scenarios) | Scenario: 792a16d1c A gate does not filter the list<br>Given the Integration is not installed<br>And a gate rejects a unit's work<br>When the unit's LESSONS line has returned<br>Then the line is in the list |

### 793a6e Modified: Each lesson is judged when it returns

Target: [792a16e Each lesson is judged when it returns](../../specs/product/lessons-contract.md#792a16e-each-lesson-is-judged-when-it-returns)

Requirement: 792a16e While the Integration is installed, the orchestrator MUST NOT judge or capture lessons from a returned report; the System captures them ([793a3m Lessons from subagent sessions are captured directly](#793a3m-lessons-from-subagent-sessions-are-captured-directly)). Where the Integration is not installed: When a returned report's `LESSONS:` line is not `none`, the orchestrator MUST judge each lesson in it at that moment, and MUST capture each lesson that clears learn's bar with a fast-path learn (one note per lesson, no sweep, handed to write-memory) before the next dispatch, or before ending its turn when no dispatch is left. A lesson clears the bar when it is one of learn's four kinds of lesson ([792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson)), is confirmed and not hypothesized, and states a general reusable principle and not a session-specific narrative. The orchestrator MUST NOT capture `none`, a routine "done, tests pass" or an unconfirmed lesson.

Rationale: The System judges each lesson against learn's bar in the subagent's own session.

#### 793a6e1 UAT Scenarios

```gherkin
Scenario: 793a6e1a The orchestrator does not judge returned lessons
  Given the Integration is installed and a subagent's turn contained a confirmed correction
  When the subagent returns and the orchestrator reads the report
  Then the transcript shows the orchestrator invoked no learn command for it
  And a note stating the correction exists in the vault
```

#### 793a6e2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a16e Each lesson is judged when it returns](../../specs/product/lessons-contract.md#792a16e-each-lesson-is-judged-when-it-returns): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6e2a | [792a16e1a A confirmed correction is captured at the return](../../specs/product/lessons-contract.md#792a16e1-uat-scenarios) | Scenario: 792a16e1a A confirmed correction is captured at the return<br>Given the Integration is not installed<br>And a unit returns LESSONS: review rejected bare error returns; the convention is fmt.Errorf with %w<br>When the orchestrator reads the report<br>Then it captures one note with a fast-path learn before the next dispatch |
| 793a6e2b | [792a16e1b None and bare success are left alone](../../specs/product/lessons-contract.md#792a16e1-uat-scenarios) | Scenario: 792a16e1b None and bare success are left alone<br>Given the Integration is not installed<br>And a unit returns LESSONS: none, or "we did the work"<br>When the orchestrator reads the report<br>Then no note is written for it |
| 793a6e2c | [792a16e1c An unconfirmed lesson is left alone](../../specs/product/lessons-contract.md#792a16e1-uat-scenarios) | Scenario: 792a16e1c An unconfirmed lesson is left alone<br>Given the Integration is not installed<br>And a unit returns LESSONS: might be a race in the cache<br>When the orchestrator judges it<br>Then it writes no note |
| 793a6e2d | [792a16e1d The last return is captured before the turn ends](../../specs/product/lessons-contract.md#792a16e1-uat-scenarios) | Scenario: 792a16e1d The last return is captured before the turn ends<br>Given the Integration is not installed<br>And the last dispatched unit returns a lesson that clears the bar<br>When there is no next dispatch<br>Then the orchestrator captures it before ending its turn |

### 793a6f Modified: The closing learn gets the whole list

Target: [792a16f The closing learn gets the whole list](../../specs/product/lessons-contract.md#792a16f-the-closing-learn-gets-the-whole-list)

Requirement: 792a16f While the Integration is installed, please MUST NOT collect `LESSONS:` lines, MUST NOT run a closing learn ([793a7l The seven steps are tracked, ordered and non-waivable](#793a7l-modified-the-seven-steps-are-tracked-ordered-and-non-waivable)) and MUST NOT hand a collection of lines to one. Where the Integration is not installed: At workflow step 7, please MUST hand the full list of `LESSONS:` lines collected across its dispatches, `none` entries included, to the closing learn, alongside the lessons-audit map ([please](../../specs/product/please.md)), and MUST drop no line. Please MUST run the closing learn when the session had no dispatches, handing it an empty collection. The closing learn MUST skip a line that a vault note written earlier this session already covers, MUST judge every other line against the bar of [792a16e Each lesson is judged when it returns](../../specs/product/lessons-contract.md#792a16e-each-lesson-is-judged-when-it-returns), and MUST discard a line that does not clear it silently, with no note and no log. It MUST NOT write a second time a line captured at its return.

Rationale: There is no collection to hand over, and no closing learn to hand it to.

#### 793a6f1 UAT Scenarios

```gherkin
Scenario: 793a6f1a The closing learn is handed no list of LESSONS lines
  Given the Integration is installed and a session with three dispatches
  When please finishes step 6
  Then no list of LESSONS lines is handed to any learn
  And the transcript shows please ran no closing learn
  And the lessons the three units contained are already in the vault
```

#### 793a6f2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a16f The closing learn gets the whole list](../../specs/product/lessons-contract.md#792a16f-the-closing-learn-gets-the-whole-list): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6f2a | [792a16f1a The closing learn receives the list](../../specs/product/lessons-contract.md#792a16f1-uat-scenarios) | Scenario: 792a16f1a The closing learn receives the list<br>Given the Integration is not installed<br>And a session with three dispatches<br>When please reaches step 7<br>Then the closing learn is given all three LESSONS lines, whatever they say |
| 793a6f2b | [792a16f1b No dispatch, empty collection](../../specs/product/lessons-contract.md#792a16f1-uat-scenarios) | Scenario: 792a16f1b No dispatch, empty collection<br>Given the Integration is not installed<br>And a session that answered a question directly with no dispatch<br>When please reaches step 7<br>Then the closing learn runs with an empty collection and takes no action on it |
| 793a6f2c | [792a16f1c A line captured at its return is skipped](../../specs/product/lessons-contract.md#792a16f1-uat-scenarios) | Scenario: 792a16f1c A line captured at its return is skipped<br>Given the Integration is not installed<br>And a session that collected three lines, one already captured by a fast-path learn at its return<br>When the closing learn runs<br>Then it does not write that line again and judges only the other two |
| 793a6f2d | [792a16f1d A missed line is caught](../../specs/product/lessons-contract.md#792a16f1-uat-scenarios) | Scenario: 792a16f1d A missed line is caught<br>Given the Integration is not installed<br>And a line that clears the bar and was not captured at its return<br>When the closing learn runs<br>Then it crystallizes the line |
| 793a6f2e | [792a16f1e A mixed batch](../../specs/product/lessons-contract.md#792a16f1-uat-scenarios) | Scenario: 792a16f1e A mixed batch<br>Given the Integration is not installed<br>And four uncaptured lines: a kind-1 correction, a kind-4 confirmed approach, a bare success and a generic reminder to test<br>When the closing learn runs<br>Then it writes the correction and the confirmed approach and discards the other two silently |

### 793a6g Modified: A confirmed approach is looked for at the moment of confirmation

Target: [792a16g A confirmed approach is looked for at the moment of confirmation](../../specs/product/lessons-contract.md#792a16g-a-confirmed-approach-is-looked-for-at-the-moment-of-confirmation)

Requirement: 792a16g While the Integration is installed, the clauses about a dispatched subagent's `LESSONS:` line and the collected lines MUST NOT apply, and the System MUST scan for a confirmed approach, with the same bar, at the moment a unit of work's outcome is confirmed. Where the Integration is not installed: Learn MUST scan for a confirmed approach (kind 4) at the moment a unit of work's outcome is confirmed: a review or gate verdict lands, a check or test passes, or the user explicitly confirms something. That includes the moment the agent reads a dispatched subagent's completion report and its `LESSONS:` line once its outcome is known, and, at the closing learn, the session's completion reports and collected lines. The bar MUST NOT move: a resolved uncertainty or an explicit specific confirmation, never bare success.

Rationale: The scan is anchored on the confirmation, which the System sees in the transcript.

#### 793a6g1 UAT Scenarios

```gherkin
Scenario: 793a6g1a A confirmed approach is looked for without the lines
  Given the Integration is installed and a subagent's work was confirmed by a passing check that resolved an uncertainty
  When the confirmation lands
  Then a note stating the confirmed approach exists in the vault
  And no LESSONS line was read or collected
```

#### 793a6g2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a16g A confirmed approach is looked for at the moment of confirmation](../../specs/product/lessons-contract.md#792a16g-a-confirmed-approach-is-looked-for-at-the-moment-of-confirmation): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6g2a | [792a16g1a A completion anchors the scan](../../specs/product/lessons-contract.md#792a16g1-uat-scenarios) | Scenario: 792a16g1a A completion anchors the scan<br>Given the Integration is not installed<br>And the session ends with the agent delivering a specific decision, validated the pattern for use in a given place, and a LESSONS line records it<br>When the closing learn scans for kind 4<br>Then it anchors on that completion moment and recognizes a learnable success |
| 793a6g2b | [792a16g1b An exemplar-shaped line is recognized](../../specs/product/lessons-contract.md#792a16g1-uat-scenarios) | Scenario: 792a16g1b An exemplar-shaped line is recognized<br>Given the Integration is not installed<br>And a subagent's line reads "targeted reproduction test confirmed the suspected defect"<br>When the closing learn judges it<br>Then it recognizes the defect-confirmed-by-reproduction-test shape as kind 4 and crystallizes it |
| 793a6g2c | [792a16g1c A bare success is still not kind 4](../../specs/product/lessons-contract.md#792a16g1-uat-scenarios) | Scenario: 792a16g1c A bare success is still not kind 4<br>Given the Integration is not installed<br>And a line that reads "the task is done"<br>When the closing learn judges it<br>Then it does not classify it as kind 4 |

### 793a6h Modified: Every dispatch hands over an exact spec

Target: [792a17g Every dispatch hands over an exact spec](../../specs/product/route.md#792a17g-every-dispatch-hands-over-an-exact-spec)

Requirement: 792a17g While the Integration is installed, the handoff MUST NOT include the instruction to recall first ([792a17h Every subagent recalls first, and large units are decomposed](../../specs/product/route.md#792a17h-every-subagent-recalls-first-and-large-units-are-decomposed)) or the completion-report contract. Where the Integration is not installed: The orchestrator MUST hand every dispatched subagent the exact files and paths to create or modify (not "the auth code" but a file and line range), the acceptance checks that mean done (a command with its expected output, a test that must pass, or a property that must hold), the explicit bounds that the unit must leave alone (files, interfaces and behaviors), the instruction to recall first ([792a17h Every subagent recalls first, and large units are decomposed](../../specs/product/route.md#792a17h-every-subagent-recalls-first-and-large-units-are-decomposed)), and the completion-report contract of [792a16b Every dispatch is told to report lessons](../../specs/product/lessons-contract.md#792a16b-every-dispatch-is-told-to-report-lessons). A vague handoff MUST be fixed before the model is blamed.

Rationale: The contract is not asked of a subagent while the Integration is installed, and the System makes recall available to the subagent ([793a7j Every subagent recalls first, and large units are decomposed](#793a7j-modified-every-subagent-recalls-first-and-large-units-are-decomposed)).

#### 793a6h1 UAT Scenarios

```gherkin
Scenario: 793a6h1a The handoff carries no completion-report contract
  Given the Integration is installed and a unit to implement a function
  When the orchestrator writes the handoff
  Then it names the exact files and lines, the acceptance checks and the do-not-touch bounds
  And it does not include a LESSONS line mandate or a recall-first instruction
```

#### 793a6h2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17g Every dispatch hands over an exact spec](../../specs/product/route.md#792a17g-every-dispatch-hands-over-an-exact-spec): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6h2a | [792a17g1a A complete handoff](../../specs/product/route.md#792a17g1-uat-scenarios) | Scenario: 792a17g1a A complete handoff<br>Given the Integration is not installed<br>And a unit to implement a function<br>When the orchestrator writes the handoff<br>Then it names the exact files and lines, the acceptance checks, the do-not-touch bounds, the recall-first instruction and the LESSONS line mandate |
| 793a6h2b | [792a17g1b A vague handoff is fixed](../../specs/product/route.md#792a17g1-uat-scenarios) | Scenario: 792a17g1b A vague handoff is fixed<br>Given the Integration is not installed<br>And a handoff that says only "fix the auth code"<br>When the orchestrator checks it<br>Then it sharpens the paths, checks and bounds before dispatching |

### 793a6i Modified: The closing learn gets the lessons and the audit

Target: [792a18r The closing learn gets the lessons and the audit](../../specs/product/please.md#792a18r-the-closing-learn-gets-the-lessons-and-the-audit)

Requirement: 792a18r While the Integration is installed, please MUST NOT run the `learn` skill at the end of the workflow, and MUST NOT hand a list of `LESSONS:` lines or an audit's map to one; the System performs the audit ([793a3o Please's lessons audit and surfacing question are performed by the System](#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system)). Where the Integration is not installed: Please MUST run the `learn` skill again at step 7 and hand it the full list of `LESSONS:` lines collected across step 4's dispatches ([792a16f The closing learn gets the whole list](../../specs/product/lessons-contract.md#792a16f-the-closing-learn-gets-the-whole-list)) together with the audit's map, and MUST NOT duplicate learn's ad-hoc question-and-answer capture ([792a15k An answered question is captured once](../../specs/product/learn.md#792a15k-an-answered-question-is-captured-once)), which learn performs for substantive answered questions.

Rationale: There are no collected lines to hand over, no audit map and no closing learn to run.

#### 793a6i1 UAT Scenarios

```gherkin
Scenario: 793a6i1b The closing learn is not run and nothing is handed to one
  Given the Integration is installed and a finished cycle
  When please reaches the end of the workflow
  Then the transcript shows please invoked no learn command
  And no list of LESSONS lines and no audit map exists in the session
```

#### 793a6i2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a18r The closing learn gets the lessons and the audit](../../specs/product/please.md#792a18r-the-closing-learn-gets-the-lessons-and-the-audit): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6i2a | [792a18r1a The closing learn receives both inputs](../../specs/product/please.md#792a18r1-uat-scenarios) | Scenario: 792a18r1a The closing learn receives both inputs<br>Given the Integration is not installed<br>And a finished cycle with a LESSONS list and an audit map<br>When please runs the closing learn<br>Then learn is given the full list, none entries included, and the audit's map |
| 793a6i2b | [792a18r1b The question-and-answer capture is learn's](../../specs/product/please.md#792a18r1-uat-scenarios) | Scenario: 792a18r1b The question-and-answer capture is learn's<br>Given the Integration is not installed<br>And a substantive answered question in the session<br>When please closes<br>Then it does not write the pair itself |

### 793a6j Modified: A returned subagent lesson is captured before the next dispatch

Target: [792a19e A returned subagent lesson is captured before the next dispatch](../../specs/product/ambient-guidance.md#792a19e-a-returned-subagent-lesson-is-captured-before-the-next-dispatch)

Requirement: 792a19e While the Integration is installed, the orchestrating agent MUST NOT judge or capture returned subagent lessons, and the cue does not apply; the System captures them ([793a3m Lessons from subagent sessions are captured directly](#793a3m-lessons-from-subagent-sessions-are-captured-directly)). Where the Integration is not installed: When a dispatched subagent returns and its `LESSONS:` line is not `none`, the orchestrating agent MUST judge each lesson at that moment against the learn skill's bar ([792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson)): it is one of the four kinds, it is confirmed and not hypothesized, and it states a general reusable principle. For each lesson that clears the bar it MUST invoke the `learn` skill on the fast path, one note per lesson and no sweep, before it dispatches the next subagent or ends its turn. The cue MUST apply to every dispatch, whether or not the route or please skill mediated it.

Rationale: The System captures the lesson from the subagent's session, so the return is no longer the moment to keep it.

#### 793a6j1 UAT Scenarios

```gherkin
Scenario: 793a6j1a A returned report is not a capture moment
  Given the Integration is installed and two parallel subagents whose turns each contained a lesson
  When the orchestrator reads both reports
  Then the transcript shows it invoked no learn command before its next dispatch
  And both lessons are in the vault, one note each
```

#### 793a6j2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19e A returned subagent lesson is captured before the next dispatch](../../specs/product/ambient-guidance.md#792a19e-a-returned-subagent-lesson-is-captured-before-the-next-dispatch): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6j2a | [792a19e1a A confirmed correction is captured before the next unit](../../specs/product/ambient-guidance.md#792a19e1-uat-scenarios) | Scenario: 792a19e1a A confirmed correction is captured before the next unit<br>Given the Integration is not installed<br>And a subagent returns "LESSONS: the reviewer rejected local-time timestamps; the convention here is UTC ISO-8601 in front matter" and another unit remains<br>When the orchestrator reads the report<br>Then it invokes the learn skill for that lesson, with no engram ingest --auto, before it dispatches the next unit |
| 793a6j2b | [792a19e1b A confirmed approach is captured as such](../../specs/product/ambient-guidance.md#792a19e1-uat-scenarios) | Scenario: 792a19e1b A confirmed approach is captured as such<br>Given the Integration is not installed<br>And a subagent returns a lesson that a targeted reproduction test confirmed a suspected defect before the fix was written<br>When the orchestrator reads the report<br>Then it captures it on the fast path before the next dispatch, as a confirmed approach |
| 793a6j2c | [792a19e1c Two returns give two notes](../../specs/product/ambient-guidance.md#792a19e1-uat-scenarios) | Scenario: 792a19e1c Two returns give two notes<br>Given the Integration is not installed<br>And two parallel subagents return and each report carries one lesson that clears the bar<br>When the orchestrator reads both<br>Then it captures both, one note each, before its next dispatch |
| 793a6j2d | [792a19e1d An ad hoc dispatch carries the cue](../../specs/product/ambient-guidance.md#792a19e1-uat-scenarios) | Scenario: 792a19e1d An ad hoc dispatch carries the cue<br>Given the Integration is not installed<br>And the orchestrator dispatched a subagent directly without route or please, and the report carries a lesson that clears the bar<br>When the report returns<br>Then the orchestrator captures the lesson all the same |
| 793a6j2e | [792a19e1e The last return is captured before the turn ends](../../specs/product/ambient-guidance.md#792a19e1-uat-scenarios) | Scenario: 792a19e1e The last return is captured before the turn ends<br>Given the Integration is not installed<br>And the final subagent returns a lesson that clears the bar and no dispatch remains<br>When the orchestrator is about to end its turn<br>Then it has captured the lesson first |

### 793a6k Modified: A returned line that fails the bar is not captured

Target: [792a19f A returned line that fails the bar is not captured](../../specs/product/ambient-guidance.md#792a19f-a-returned-line-that-fails-the-bar-is-not-captured)

Requirement: 792a19f While the Integration is installed, the learn guidance MUST NOT depend on a returned `LESSONS:` line: none is returned, so there is nothing for it to fire for or skip. Where the Integration is not installed: The learn guidance MUST NOT fire for a returned `LESSONS:` line that is `none`, for a lesson that is a bare success or routine narrative, or for a lesson that is an unconfirmed hypothesis, which are left for the closing learn to discard under the same bar.

Rationale: The requirement governs lines that are no longer returned.

#### 793a6k1 UAT Scenarios

```gherkin
Scenario: 793a6k1a No returned line exists to skip
  Given the Integration is installed and a subagent whose turn contained only routine success
  When the orchestrator reads the report
  Then the report carries no LESSONS line
  And no note is written for the routine success
```

#### 793a6k2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19f A returned line that fails the bar is not captured](../../specs/product/ambient-guidance.md#792a19f-a-returned-line-that-fails-the-bar-is-not-captured): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a6k2a | [792a19f1a LESSONS none is skipped](../../specs/product/ambient-guidance.md#792a19f1-uat-scenarios) | Scenario: 792a19f1a LESSONS none is skipped<br>Given the Integration is not installed<br>And a subagent returns "LESSONS: none"<br>When the orchestrator reads the report<br>Then it does not invoke the learn skill before the next dispatch |
| 793a6k2b | [792a19f1b A routine narrative is skipped](../../specs/product/ambient-guidance.md#792a19f1-uat-scenarios) | Scenario: 792a19f1b A routine narrative is skipped<br>Given the Integration is not installed<br>And a subagent returns "LESSONS: completed the unit, all tests pass"<br>When the orchestrator reads the report<br>Then it does not invoke the learn skill before the next dispatch |
| 793a6k2c | [792a19f1c An unconfirmed hunch is skipped](../../specs/product/ambient-guidance.md#792a19f1-uat-scenarios) | Scenario: 792a19f1c An unconfirmed hunch is skipped<br>Given the Integration is not installed<br>And a subagent returns "LESSONS: the build flake might be a race in the cache warmer" and nothing in the report confirms it<br>When the orchestrator reads the report<br>Then it does not invoke the learn skill before the next dispatch |

## 793a7 Modified: ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)

Canonical targets: [ambient-guidance](../../specs/product/ambient-guidance.md), [route](../../specs/product/route.md) and [please](../../specs/product/please.md), the nodes that tell the working agent to run recall or learn, or to perform their steps ([793c31 Which canonical nodes that ask the agent to run recall or learn change while the Integration is installed?](793c_open-questions.md#793c31-which-canonical-nodes-that-ask-the-agent-to-run-recall-or-learn-change-while-the-integration-is-installed)). Each operation is a Modified operation against exactly one canonical node, conditional on the Integration, because the node still holds in harnesses and installs without the Integration ([793a5c Environments without the capability](#793a5c-environments-without-the-capability)). Each target's ID is preserved. The replacement text carries the Integration branch first, then the target's text unchanged for an install without the Integration, with its in-page references rewritten as links to the canonical file. The condition is evaluated per harness and per agent session: a session in a harness without the Integration, such as Pi, follows the target's text unchanged. Each operation's scenario group ending in 1 covers its Integration branch. The group ending in 2 holds one Modified operation for each canonical scenario of the target: that scenario's text prefixed with "Given the Integration is not installed", so that the branch for an install without the Integration keeps its canonical coverage. No scenario in this section restates a canonical scenario under a new ID; the "Without the Integration" scenarios that did, `793a7a1e`, `793a7b1b`, `793a7c1d`, `793a7d1c`, `793a7e1b`, `793a7f1b`, `793a7g1c`, `793a7h1b`, `793a7i1c`, `793a7j1c`, `793a7k1b`, `793a7l1c`, `793a7m1c`, `793a7n1c`, `793a7o1c`, `793a7o1e`, `793a7p1c`, `793a7q1e`, `793a7r1f`, are retired and their IDs are never reused ([793b117 H4: every Modified target keeps its canonical scenarios](793b_decision-log.md#793b117-h4-every-modified-target-keeps-its-canonical-scenarios)). Where the moments or criteria that decide when recall or learn fires are stated in a target, the Integration branch restates them, because the System uses them ([793a2a Recall is performed for the working agent](#793a2a-recall-is-performed-for-the-working-agent), [793a3a Learn is performed for the working agent](#793a3a-learn-is-performed-for-the-working-agent)).

### 793a7a Modified: Recall fires at four decision moments

Target: [792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments)

Requirement: 792a19a While the Integration is installed, recall MUST fire at four decision moments, and the System MUST perform it at each ([793a2a Recall is performed for the working agent](#793a2a-recall-is-performed-for-the-working-agent)): before the agent endorses or ranks a proposed approach, even when only discussing it; before it declares work done; once, before it starts guessing, after a failure it cannot immediately explain, and not on every retry; and before it starts building a new approach. An agent that has the recall guidance MUST NOT be required to run `/recall glance` at any of them. Where the Integration is not installed: An agent that has the recall guidance MUST run `/recall glance` before it proceeds at each of four moments: before it endorses or ranks a proposed approach, even when only discussing it; before it declares work done; once, before it starts guessing, after a failure it cannot immediately explain, and not on every retry; and before it starts building a new approach.

Rationale: The moments stay the same; who performs the recall changes.

#### 793a7a1 UAT Scenarios

```gherkin
Scenario: 793a7a1a The Integration recalls before a proposed approach is endorsed
  Given the Integration is installed and a note that matches a proposed approach
  And the user proposes the approach and asks the agent what it thinks
  When the agent is about to evaluate it
  Then the transcript shows the note's content delivered before the agent offers any analysis
  And the transcript shows the agent ran no recall command

Scenario: 793a7a1b The Integration recalls before work is declared done
  Given the Integration is installed and a note that matches the work the agent is about to declare done
  When the agent reaches the point of declaring it done
  Then the transcript shows the note's content delivered before the declaration
  And the transcript shows the agent ran no recall command

Scenario: 793a7a1c The Integration recalls once for an unexplained failure
  Given the Integration is installed and a failure the agent cannot immediately explain, and then a retry that fails the same way
  When the agent meets the failure and then the retry
  Then the log records one recall moment for the failure and none for the retry

Scenario: 793a7a1d The Integration recalls before a new approach is built
  Given the Integration is installed and a note holding a prior decision that matches a new approach
  When the agent has chosen the approach and is about to start building
  Then the transcript shows the note's content delivered before the agent starts building
  And the transcript shows the agent ran no recall command
```

#### 793a7a2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7a2a | [792a19a1a A proposed approach is recalled before it is endorsed](../../specs/product/ambient-guidance.md#792a19a1-uat-scenarios) | Scenario: 792a19a1a A proposed approach is recalled before it is endorsed<br>Given the Integration is not installed<br>And the user proposes a technical approach and asks the agent what it thinks<br>When the agent is about to evaluate or rank it<br>Then it runs /recall glance first, with phrases drawn from the approach, before offering any analysis |
| 793a7a2b | [792a19a1b Work is recalled before it is declared done](../../specs/product/ambient-guidance.md#792a19a1-uat-scenarios) | Scenario: 792a19a1b Work is recalled before it is declared done<br>Given the Integration is not installed<br>And the agent has finished a task, unit or phase<br>When it is about to declare it done<br>Then it runs /recall glance before it verifies and closes |
| 793a7a2c | [792a19a1c An unexplained failure is recalled once](../../specs/product/ambient-guidance.md#792a19a1-uat-scenarios) | Scenario: 792a19a1c An unexplained failure is recalled once<br>Given the Integration is not installed<br>And a failure the agent cannot immediately explain, and then a retry that fails the same way<br>When the agent meets the failure<br>Then it runs /recall glance once before it guesses at causes, and does not run it again for the retry |
| 793a7a2d | [792a19a1d A new approach is recalled before it is built](../../specs/product/ambient-guidance.md#792a19a1-uat-scenarios) | Scenario: 792a19a1d A new approach is recalled before it is built<br>Given the Integration is not installed<br>And the agent is about to begin implementing a new approach<br>When it has chosen the approach<br>Then it runs /recall glance for prior decisions and standards before it starts building |

### 793a7b Modified: Recall escalates from glance to deep

Target: [792a19b Recall escalates from glance to deep](../../specs/product/ambient-guidance.md#792a19b-recall-escalates-from-glance-to-deep)

Requirement: 792a19b While the Integration is installed, the System MUST perform recall at `deep` depth and not only at `glance` depth when the decision is weighty or irreversible, or when a glance flags a gap it cannot resolve, and an agent that has the recall guidance MUST NOT be required to run `/recall deep`. Where the Integration is not installed: An agent that has the recall guidance MUST escalate from `/recall glance` to `/recall deep` when the decision is weighty or irreversible, or when glance flags a gap it cannot resolve.

Rationale: The escalation criteria stay the same; who runs the deeper recall changes.

#### 793a7b1 UAT Scenarios

```gherkin
Scenario: 793a7b1a The Integration escalates a weighty decision
  Given the Integration is installed and a decision that is irreversible
  When the agent reaches the moment before making it
  Then the log records the recall for that moment at depth deep
  And the transcript shows the agent ran no recall command
```

#### 793a7b2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19b Recall escalates from glance to deep](../../specs/product/ambient-guidance.md#792a19b-recall-escalates-from-glance-to-deep): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7b2a | [792a19b1a A weighty decision escalates](../../specs/product/ambient-guidance.md#792a19b1-uat-scenarios) | Scenario: 792a19b1a A weighty decision escalates<br>Given the Integration is not installed<br>And a decision that is irreversible<br>When the agent recalls before making it<br>Then it runs /recall deep and not only /recall glance |
| 793a7b2b | [792a19b1b An unresolved gap escalates](../../specs/product/ambient-guidance.md#792a19b1-uat-scenarios) | Scenario: 792a19b1b An unresolved gap escalates<br>Given the Integration is not installed<br>And a glance that flags a gap the agent cannot resolve<br>When the agent reads the result<br>Then it escalates to /recall deep |

### 793a7c Modified: Learn fires at the correction moment

Target: [792a19c Learn fires at the correction moment](../../specs/product/ambient-guidance.md#792a19c-learn-fires-at-the-correction-moment)

Requirement: 792a19c While the Integration is installed, learn MUST fire when a review or the user rejects the agent's approach, when a build, test or check fails and the non-obvious cause is now known, and when the agent catches its own approach being wrong, and even a one-line fix MUST earn the note; the System MUST perform it at each ([793a3a Learn is performed for the working agent](#793a3a-learn-is-performed-for-the-working-agent)), and an agent that has the learn guidance MUST NOT be required to invoke the `learn` skill or to wait for the capture before it applies the fix. Where the Integration is not installed: An agent that has the learn guidance MUST invoke the `learn` skill, before it applies the fix, when a review or the user rejects its approach, when a build, test or check fails and the non-obvious cause is now known, and when it catches its own approach being wrong. Even a one-line fix MUST earn the note.

Rationale: The moments stay the same; the System captures the lesson while the agent applies the fix.

#### 793a7c1 UAT Scenarios

```gherkin
Scenario: 793a7c1a The Integration captures a rejected approach
  Given the Integration is installed
  When the user says "no, do X not Y, the convention here is Z"
  And the turn completes
  Then a note stating the convention exists in the vault
  And the transcript shows the agent invoked no learn command and was asked for none

Scenario: 793a7c1b The Integration captures a failing check with a known cause
  Given the Integration is installed and a check that failed with a cause that was not obvious
  When the agent states the cause
  Then a note stating the cause exists in the vault
  And the transcript shows the agent invoked no learn command

Scenario: 793a7c1c The Integration captures a one-line fix
  Given the Integration is installed and a correction whose fix is a single line
  When the agent applies the fix
  Then a note stating the rule exists in the vault
  And the transcript shows the agent invoked no learn command
```

#### 793a7c2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19c Learn fires at the correction moment](../../specs/product/ambient-guidance.md#792a19c-learn-fires-at-the-correction-moment): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7c2a | [792a19c1a A rejected approach is captured before the fix](../../specs/product/ambient-guidance.md#792a19c1-uat-scenarios) | Scenario: 792a19c1a A rejected approach is captured before the fix<br>Given the Integration is not installed<br>And the user says "no, do X not Y, the convention here is Z"<br>When the agent is about to apply the fix<br>Then it invokes the learn skill for the correction first |
| 793a7c2b | [792a19c1b A failing check with a known cause is captured](../../specs/product/ambient-guidance.md#792a19c1-uat-scenarios) | Scenario: 792a19c1b A failing check with a known cause is captured<br>Given the Integration is not installed<br>And a check failed with a cause that was not obvious and the agent now knows the cause<br>When the agent has the cause<br>Then it invokes the learn skill so the next session does not hit it again |
| 793a7c2c | [792a19c1c A self-discovered reversal is captured](../../specs/product/ambient-guidance.md#792a19c1-uat-scenarios) | Scenario: 792a19c1c A self-discovered reversal is captured<br>Given the Integration is not installed<br>And the agent notices partway through that its own approach was wrong<br>When it names the root cause<br>Then it invokes the learn skill at that moment |
| 793a7c2d | [792a19c1d A one-line fix still earns the note](../../specs/product/ambient-guidance.md#792a19c1-uat-scenarios) | Scenario: 792a19c1d A one-line fix still earns the note<br>Given the Integration is not installed<br>And a correction whose fix is a single line<br>When the agent applies it<br>Then it has invoked the learn skill for the rule first |

### 793a7d Modified: The capture is learn's fast path

Target: [792a19d The capture is learn's fast path](../../specs/product/ambient-guidance.md#792a19d-the-capture-is-learns-fast-path)

Requirement: 792a19d While the Integration is installed, when learn fires the System MUST capture the one confirmed lesson on the skill's mid-cycle fast path, MUST capture a confirmed negative (what was rejected) on its own, MUST NOT capture an unconfirmed hunch, and MUST NOT run `engram ingest --auto` mid-task; the agent MUST NOT be asked to capture it. Where the Integration is not installed: When the learn guidance fires, the agent MUST capture the one confirmed lesson with the `learn` skill, invoked as a skill and not as a plain note file or the harness's own project memory, on the skill's mid-cycle fast path, and MUST NOT run `engram ingest --auto` mid-task. It MUST capture a confirmed negative (what was rejected), which is enough on its own, and MUST NOT crystallize an unconfirmed hunch.

Rationale: The fast-path rules stay the same; the System applies them.

#### 793a7d1 UAT Scenarios

```gherkin
Scenario: 793a7d1a The Integration captures a rejection without the right answer
  Given the Integration is installed and a review that rejected a path join with the right rule not yet known
  When the turn completes
  Then a note recording the rejection exists in the vault and states no guessed rule
  And the log records no `engram ingest --auto` run during the task

Scenario: 793a7d1b The Integration writes no note for a hunch
  Given the Integration is installed and an agent that suspects a cause that nothing confirms
  When the turn completes
  Then no note stating the suspected cause exists in the vault
```

#### 793a7d2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19d The capture is learn's fast path](../../specs/product/ambient-guidance.md#792a19d-the-capture-is-learns-fast-path): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7d2a | [792a19d1a The learn skill is invoked, not a native memory write](../../specs/product/ambient-guidance.md#792a19d1-uat-scenarios) | Scenario: 792a19d1a The learn skill is invoked, not a native memory write<br>Given the Integration is not installed<br>And a confirmed correction<br>When the agent captures it<br>Then the capture is a learn skill invocation that writes a vault note, and not a write to the harness's project memory or a plain file |
| 793a7d2b | [792a19d1b No sweep runs mid-task](../../specs/product/ambient-guidance.md#792a19d1-uat-scenarios) | Scenario: 792a19d1b No sweep runs mid-task<br>Given the Integration is not installed<br>And a correction in the middle of a task<br>When the agent captures it<br>Then it crystallizes the one lesson and runs no engram ingest --auto |
| 793a7d2c | [792a19d1c A rejection is captured without the right answer](../../specs/product/ambient-guidance.md#792a19d1-uat-scenarios) | Scenario: 792a19d1c A rejection is captured without the right answer<br>Given the Integration is not installed<br>And a review rejected a path join and the right rule is not yet known<br>When the agent captures<br>Then it records the rejection and does not guess at the rule |
| 793a7d2d | [792a19d1d A hunch is not captured](../../specs/product/ambient-guidance.md#792a19d1-uat-scenarios) | Scenario: 792a19d1d A hunch is not captured<br>Given the Integration is not installed<br>And the agent suspects a cause but nothing confirms it<br>When it considers capturing<br>Then it writes no note |

### 793a7e Modified: The first action on every request is an engram query

Target: [792a19j The first action on every request is an engram query](../../specs/product/ambient-guidance.md#792a19j-the-first-action-on-every-request-is-an-engram-query)

Requirement: 792a19j While the Integration is installed, an agent that has the shim MUST NOT be required to make an `engram query` its first tool call on any request; the System performs the lookup at every prompt ([793a2i Recall for every prompt regardless of source](#793a2i-recall-for-every-prompt-regardless-of-source)) and puts what it finds in front of the agent before the agent's first response or tool call. Where the Integration is not installed: An agent that has the shim MUST make its first tool call on every user request an `engram query` that carries at least two `--phrase` values (the request in the agent's own words, and the kind of situation it is in) and a `--text` value holding the user's message verbatim (its first 300 characters or so when long), before it reads a file, answers or asks a clarifying question, and whether or not it has decided to delegate. It MUST NOT gate the query on a judgment that the task is multi-step, recurring or done before. It MUST write every `--text` and `--phrase` value as `--flag="value"`, never `--flag "value"`, and MUST NOT rewrite or summarize the message in `--text`, whose paraphrase belongs only in `--phrase`.

Rationale: The lookup is performed for the agent, so no first query is asked of it.

#### 793a7e1 UAT Scenarios

```gherkin
Scenario: 793a7e1a The Integration looks up memory without a first query
  Given the Integration is installed and an agent with the shim in its config
  And a note whose situation matches a request
  When the user submits the request
  Then the transcript shows the note's content delivered before the agent's first response or tool call
  And the agent's instructions contain no requirement to make an engram query its first tool call
```

#### 793a7e2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19j The first action on every request is an engram query](../../specs/product/ambient-guidance.md#792a19j-the-first-action-on-every-request-is-an-engram-query): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7e2a | [792a19j1a A shim-only session queries first](../../specs/product/ambient-guidance.md#792a19j1-uat-scenarios) | Scenario: 792a19j1a A shim-only session queries first<br>Given the Integration is not installed<br>And an agent with the shim in its config and no engram skill installed<br>When it receives a user request<br>Then its first tool call is an engram query carrying the request phrase, the situation phrase and --text with the user's message, before any other tool call |
| 793a7e2b | [792a19j1b A one-shot request still queries](../../specs/product/ambient-guidance.md#792a19j1-uat-scenarios) | Scenario: 792a19j1b A one-shot request still queries<br>Given the Integration is not installed<br>And a single-step request such as a one-line answer or a single-file typo fix<br>When the agent receives it<br>Then it still runs the query first, and proceeds without a runbook when none matches |
| 793a7e2c | [792a19j1c The message is passed unchanged](../../specs/product/ambient-guidance.md#792a19j1-uat-scenarios) | Scenario: 792a19j1c The message is passed unchanged<br>Given the Integration is not installed<br>And the user's message "/please rename the tally CLI's --out flag to --output"<br>When the agent runs its first query<br>Then the --text value holds that string verbatim up to whitespace differences, and the paraphrase appears only in a --phrase |
| 793a7e2d | [792a19j1d A message that begins with a dash survives](../../specs/product/ambient-guidance.md#792a19j1-uat-scenarios) | Scenario: 792a19j1d A message that begins with a dash survives<br>Given the Integration is not installed<br>And the user's message begins with "-", such as "- fix the login bug"<br>When the agent runs its first query<br>Then --text is written --text="- fix the login bug" and the command runs instead of exiting with a flag needing an argument |
| 793a7e2e | [792a19j1e Deciding to delegate does not skip the query](../../specs/product/ambient-guidance.md#792a19j1-uat-scenarios) | Scenario: 792a19j1e Deciding to delegate does not skip the query<br>Given the Integration is not installed<br>And an agent about to dispatch a subagent as its first action<br>When it begins<br>Then it runs the engram query first, and the decision to delegate is not an exemption |

### 793a7f Modified: The second phrase names a kind of work-handling decision

Target: [792a19k The second phrase names a kind of work-handling decision](../../specs/product/ambient-guidance.md#792a19k-the-second-phrase-names-a-kind-of-work-handling-decision)

Requirement: 792a19k While the Integration is installed, the shim MUST NOT tell the agent how to compose the phrases of an `engram query`, because the agent does not run the first query; the System composes its own. Where the Integration is not installed: The shim MUST tell the agent that the second phrase of the first query names the kind of situation, in the wording a runbook's own `situation` uses, and not the concrete ticket, and MUST tell it to re-check that phrase before the query runs and rewrite it as the kind of work-handling decision when it contains a concrete noun from the ticket (a file, flag, feature, bug or field name) or restates the request. The shim MUST illustrate the shape with at least two examples of different kinds of work, and the re-check and the examples MUST NOT name a specific runbook, skill or evaluation task.

Rationale: There is no agent-composed query to shape.

#### 793a7f1 UAT Scenarios

```gherkin
Scenario: 793a7f1a The Integration gives the agent no phrase-shaping instruction
  Given the Integration is installed and an agent with the shim in its config
  When the agent's instructions are inspected
  Then they contain no instruction on how to write the phrases of a query
```

#### 793a7f2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19k The second phrase names a kind of work-handling decision](../../specs/product/ambient-guidance.md#792a19k-the-second-phrase-names-a-kind-of-work-handling-decision): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7f2a | [792a19k1a A deliverable-laden phrase is rewritten](../../specs/product/ambient-guidance.md#792a19k1-uat-scenarios) | Scenario: 792a19k1a A deliverable-laden phrase is rewritten<br>Given the Integration is not installed<br>And the agent's draft second phrase contains a flag, file or endpoint name, or restates the request<br>When it re-checks before running the query<br>Then the second --phrase it passes names a class of work-handling decision and contains none of those nouns |
| 793a7f2b | [792a19k1b The re-check wording is runbook-agnostic](../../specs/product/ambient-guidance.md#792a19k1-uat-scenarios) | Scenario: 792a19k1b The re-check wording is runbook-agnostic<br>Given the Integration is not installed<br>And the shim's re-check and its examples<br>When they are read<br>Then they name no specific runbook, skill or evaluation task and the examples cover at least two different kinds of work |
| 793a7f2c | [792a19k1c The re-check does not regress another runbook's retrieval](../../specs/product/ambient-guidance.md#792a19k1-uat-scenarios) | Scenario: 792a19k1c The re-check does not regress another runbook's retrieval<br>Given the Integration is not installed<br>And a runbook whose retrieval was validated before the re-check, such as the dispatch-tier runbook<br>When its task is run with the shim alone<br>Then the runbook is retrieved at the same rate as before |

### 793a7g Modified: The query runs again at four moments mid-task

Target: [792a19t The query runs again at four moments mid-task](../../specs/product/ambient-guidance.md#792a19t-the-query-runs-again-at-four-moments-mid-task)

Requirement: 792a19t While the Integration is installed, the shim MUST NOT tell the agent to run `engram query` again at the four moments; the System performs recall at them ([793a7a Recall fires at four decision moments](#793a7a-modified-recall-fires-at-four-decision-moments)). The shim MUST still tell the agent to use a matched runbook's `deep` procedure, if it offers one, when the decision is weighty or irreversible. Where the Integration is not installed: The shim MUST tell the agent to run `engram query` again, with the same two phrase shapes re-keyed to what is in front of it, before it endorses or ranks a proposed approach, even when only discussing it; before it declares work done, querying first and then verifying; once after a failure it cannot immediately explain; and before it starts building a new approach. It MUST tell the agent to escalate to the matched runbook's `deep` procedure, if it offers one, when the decision is weighty or irreversible or the pass surfaces a gap it cannot resolve.

Rationale: The same four moments are recalled for the agent.

#### 793a7g1 UAT Scenarios

```gherkin
Scenario: 793a7g1a The Integration queries before work is declared done without the agent
  Given the Integration is installed and a note that matches the work the agent is about to declare done
  When the agent prepares to declare it done
  Then the transcript shows the note's content delivered before the declaration
  And the transcript shows the agent ran no engram query

Scenario: 793a7g1b The Integration still uses a deep procedure for a weighty decision
  Given the Integration is installed, a matched runbook that offers a deep procedure and a decision that is irreversible
  When the agent reaches the decision
  Then the transcript shows it used the deep procedure
```

#### 793a7g2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a19t The query runs again at four moments mid-task](../../specs/product/ambient-guidance.md#792a19t-the-query-runs-again-at-four-moments-mid-task): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7g2a | [792a19t1a Work is queried before it is declared done](../../specs/product/ambient-guidance.md#792a19t1-uat-scenarios) | Scenario: 792a19t1a Work is queried before it is declared done<br>Given the Integration is not installed<br>And the agent is about to declare work done<br>When it prepares to<br>Then it runs engram query keyed to the current work first, and then verifies |
| 793a7g2b | [792a19t1b A new approach is queried before it is built](../../specs/product/ambient-guidance.md#792a19t1-uat-scenarios) | Scenario: 792a19t1b A new approach is queried before it is built<br>Given the Integration is not installed<br>And the agent is about to start building a new approach<br>When it chooses the approach<br>Then it runs engram query with phrases re-keyed to the approach before it starts |
| 793a7g2c | [792a19t1c A weighty decision escalates to deep](../../specs/product/ambient-guidance.md#792a19t1-uat-scenarios) | Scenario: 792a19t1c A weighty decision escalates to deep<br>Given the Integration is not installed<br>And a matched runbook that offers a deep procedure and a decision that is irreversible<br>When the agent reaches it<br>Then it uses the deep procedure |

### 793a7h Modified: The orchestrator routes and delegates

Target: [792a17a The orchestrator routes and delegates](../../specs/product/route.md#792a17a-the-orchestrator-routes-and-delegates)

Requirement: 792a17a While the Integration is installed, the orchestrator following route MUST do orchestration itself (routing and decomposition decisions, dispatching subagents, sequencing steps, updating the task list, running the meta-skills that are the workflow such as planning, and synthesizing returned results into the next decision or the report), MUST NOT run recall or learn, which the System performs, and MUST delegate every piece of object-level work (writing code or prose, running tests or builds, judging the artifact, reviewing it) to a subagent. It MUST NOT do easy work inline: easy work goes to a cheap model and is not skipped. Where the Integration is not installed: The orchestrator following route MUST do orchestration itself (routing and decomposition decisions, dispatching subagents, sequencing steps, updating the task list, running the meta-skills that are the workflow such as recall, learn and planning, and synthesizing returned results into the next decision or the report) and MUST delegate every piece of object-level work (writing code or prose, running tests or builds, judging the artifact, reviewing it) to a subagent. It MUST NOT do easy work inline: easy work goes to a cheap model and is not skipped.

Rationale: Recall and learn are no longer among the meta-skills the orchestrator runs.

#### 793a7h1 UAT Scenarios

```gherkin
Scenario: 793a7h1a The Integration leaves recall and learn out of the orchestrator's work
  Given the Integration is installed and a unit that writes code and runs its tests
  When the orchestrator follows route
  Then it dispatches the unit to a subagent and does not do it inline
  And the transcript shows the orchestrator ran no recall or learn
```

#### 793a7h2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17a The orchestrator routes and delegates](../../specs/product/route.md#792a17a-the-orchestrator-routes-and-delegates): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7h2a | [792a17a1a Object-level work is dispatched](../../specs/product/route.md#792a17a1-uat-scenarios) | Scenario: 792a17a1a Object-level work is dispatched<br>Given the Integration is not installed<br>And a unit that writes code and runs its tests<br>When the orchestrator follows route<br>Then it dispatches the unit to a subagent and does not do it inline |
| 793a7h2b | [792a17a1b Orchestration stays with the orchestrator](../../specs/product/route.md#792a17a1-uat-scenarios) | Scenario: 792a17a1b Orchestration stays with the orchestrator<br>Given the Integration is not installed<br>And results returned from three subagents<br>When the next step must be decided<br>Then the orchestrator synthesizes them itself and does not delegate the act of delegating |
| 793a7h2c | [792a17a1c Easy work is delegated cheaply](../../specs/product/route.md#792a17a1-uat-scenarios) | Scenario: 792a17a1c Easy work is delegated cheaply<br>Given the Integration is not installed<br>And a one-line rename<br>When the orchestrator routes it<br>Then it dispatches it to the cheapest tier instead of doing it inline |

### 793a7i Modified: The starting tier comes from recalled evidence, otherwise the cheapest tier

Target: [792a17b The starting tier comes from recalled evidence, otherwise the cheapest tier](../../specs/product/route.md#792a17b-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier)

Requirement: 792a17b While the Integration is installed, before dispatching, the orchestrator MUST use the tier-performance evidence for this kind of work that is in its context, including what the System delivered at its decision to dispatch ([793a2c Moments are noticed without the agent](#793a2c-moments-are-noticed-without-the-agent)) (the aggregates surface as ordinary memories), MUST NOT run recall to look for it, and MUST NOT use a special query or counting on the read path. Absent such evidence, it MUST start every unit at the cheapest, fastest available tier, with no exception for work that merely looks hard (hard debugging, a cross-cutting refactor, a correctness-critical review, a greenfield design). The starting tier MUST NOT be raised for any reason other than recalled evidence of failure at a cheaper tier, or a cold-start prior that records such evidence ([792a17f Cold-start priors carry only evidence-backed escalations](../../specs/product/route.md#792a17f-cold-start-priors-carry-only-evidence-backed-escalations)), and then only to the tier the evidence supports. Predicted difficulty ("genuinely complex", "needs careful reasoning", "broad impact") MUST NOT count as evidence. Where the Integration is not installed: Before dispatching, the orchestrator MUST check recalled memory for tier-performance evidence on this kind of work, using the plain recall of the agent (the aggregates surface as ordinary memories) and no special query or counting on the read path. Absent recalled evidence, it MUST start every unit at the cheapest, fastest available tier, with no exception for work that merely looks hard (hard debugging, a cross-cutting refactor, a correctness-critical review, a greenfield design). The starting tier MUST NOT be raised for any reason other than recalled evidence of failure at a cheaper tier, or a cold-start prior that records such evidence ([792a17f Cold-start priors carry only evidence-backed escalations](../../specs/product/route.md#792a17f-cold-start-priors-carry-only-evidence-backed-escalations)), and then only to the tier the evidence supports. Predicted difficulty ("genuinely complex", "needs careful reasoning", "broad impact") MUST NOT count as evidence.

Rationale: The evidence arrives through the System's recall at the orchestrator's decision to dispatch instead of through the orchestrator's own.

#### 793a7i1 UAT Scenarios

```gherkin
Scenario: 793a7i1a The Integration routes on evidence the System delivered
  Given the Integration is installed and an aggregate showing that cheap failed for a work-kind
  When the orchestrator decides to dispatch a unit of that work-kind
  Then the transcript shows the aggregate delivered before the orchestrator routes the unit
  And the unit starts at the tier the evidence supports
  And the transcript shows the orchestrator ran no recall

Scenario: 793a7i1b The Integration starts at the cheapest tier absent evidence
  Given the Integration is installed and no tier evidence exists to deliver for a hard-looking work-kind
  When the orchestrator decides to dispatch a unit of that work-kind
  Then the unit starts at the cheapest tier
  And the transcript shows the orchestrator ran no recall
```

#### 793a7i2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17b The starting tier comes from recalled evidence, otherwise the cheapest tier](../../specs/product/route.md#792a17b-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7i2a | [792a17b1a No evidence, cheapest tier](../../specs/product/route.md#792a17b1-uat-scenarios) | Scenario: 792a17b1a No evidence, cheapest tier<br>Given the Integration is not installed<br>And a unit of work with no recalled evidence on similar work-kinds<br>When the orchestrator routes it<br>Then it selects the cheapest tier and records the choice as a cold-start default |
| 793a7i2b | [792a17b1b Hard-looking work starts cheap](../../specs/product/route.md#792a17b1-uat-scenarios) | Scenario: 792a17b1b Hard-looking work starts cheap<br>Given the Integration is not installed<br>And a cross-cutting refactor, a new API design and a race-condition investigation<br>When the orchestrator routes each with no recalled evidence<br>Then each starts at the cheapest tier |
| 793a7i2c | [792a17b1c Recalled failure raises the start](../../specs/product/route.md#792a17b1-uat-scenarios) | Scenario: 792a17b1c Recalled failure raises the start<br>Given the Integration is not installed<br>And recall surfaces evidence that this work-kind failed at the cheap tier before<br>When the orchestrator routes it<br>Then it starts at the tier the evidence supports and no higher |
| 793a7i2d | [792a17b1d A hunch is not evidence](../../specs/product/route.md#792a17b1-uat-scenarios) | Scenario: 792a17b1d A hunch is not evidence<br>Given the Integration is not installed<br>And the orchestrator reasons that a unit is correctness-critical and so needs the deep tier<br>When it chooses the tier<br>Then it starts cheap, because that reasoning forecasts difficulty |
| 793a7i2e | [792a17b1e Evidence is read by plain recall](../../specs/product/route.md#792a17b1-uat-scenarios) | Scenario: 792a17b1e Evidence is read by plain recall<br>Given the Integration is not installed<br>And a route-evidence aggregate for the work-kind exists<br>When the orchestrator recalls for the dispatch<br>Then it reads the aggregate's tallies from the recalled item and runs no engram count to route |

### 793a7j Modified: Every subagent recalls first, and large units are decomposed

Target: [792a17h Every subagent recalls first, and large units are decomposed](../../specs/product/route.md#792a17h-every-subagent-recalls-first-and-large-units-are-decomposed)

Requirement: 792a17h While the Integration is installed, the orchestrator MUST NOT instruct a dispatched subagent to recall first, because the System makes what recall returns for the subagent's task available to it ([793a2c Moments are noticed without the agent](#793a2c-moments-are-noticed-without-the-agent)). A unit that spans multiple files or concerns or needs more than one clear deliverable MUST NOT be dispatched as it stands: the orchestrator MUST break it into smaller units and route each, and the decomposition MUST be done by the orchestrator. Where the Integration is not installed: The orchestrator MUST instruct every dispatched subagent that its first action is recall, with phrases drawn from its unit, before it does the work, and this instruction MUST NOT be waived. A unit that spans multiple files or concerns or needs more than one clear deliverable MUST NOT be dispatched as it stands: the orchestrator MUST break it into smaller units and route each, and the decomposition MUST be done by the orchestrator.

Rationale: The subagent's recall is the System's, and the decomposition rule does not depend on it.

#### 793a7j1 UAT Scenarios

```gherkin
Scenario: 793a7j1a The Integration's handoff carries no recall-first instruction
  Given the Integration is installed and a note matching a subagent's unit
  When the orchestrator dispatches the subagent
  Then the handoff does not tell the subagent to recall first
  And the subagent's transcript shows the note's content delivered before its first response or tool call

Scenario: 793a7j1b The Integration still decomposes a large unit
  Given the Integration is installed and a unit that touches six packages and has two deliverables
  When the orchestrator routes it
  Then it splits it into smaller units and routes each, and does not dispatch it as one agent
```

#### 793a7j2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17h Every subagent recalls first, and large units are decomposed](../../specs/product/route.md#792a17h-every-subagent-recalls-first-and-large-units-are-decomposed): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7j2a | [792a17h1a Recall first is in every handoff](../../specs/product/route.md#792a17h1-uat-scenarios) | Scenario: 792a17h1a Recall first is in every handoff<br>Given the Integration is not installed<br>And a subagent that already has a detailed prompt<br>When the orchestrator dispatches it<br>Then the handoff still tells it to recall first |
| 793a7j2b | [792a17h1b A large unit is decomposed](../../specs/product/route.md#792a17h1-uat-scenarios) | Scenario: 792a17h1b A large unit is decomposed<br>Given the Integration is not installed<br>And a unit that touches six packages and has two deliverables<br>When the orchestrator routes it<br>Then it splits it into smaller units and routes each, and does not dispatch it as one agent |

### 793a7k Modified: The loop improves the rubric

Target: [792a17o The loop improves the rubric](../../specs/product/route.md#792a17o-the-loop-improves-the-rubric)

Requirement: 792a17o While the Integration is installed, when a finding about a work-kind is strong and general (for example, "cheap failed N times on work-kind K, so it needs mid"), the orchestrator MUST NOT crystallize it with learn; the System MUST capture it as a confirmed approach when a tier's outcome confirms the routing (for example, "cheap sufficed for K") or as a reversal when it overturns a prior tier assumption. The next recall MUST surface that evidence so that the starting tier for similar work reflects it. Where the Integration is not installed: When a finding about a work-kind is strong and general (for example, "cheap failed N times on work-kind K, so it needs mid"), the orchestrator MUST crystallize it with learn, as a confirmed approach when a tier's outcome confirms the routing (for example, "cheap sufficed for K") or as a reversal when it overturns a prior tier assumption. The next recall MUST surface that evidence so that the starting tier for similar work reflects it.

Rationale: The finding is a lesson, and the System captures lessons.

#### 793a7k1 UAT Scenarios

```gherkin
Scenario: 793a7k1a The Integration captures a confirming tier outcome
  Given the Integration is installed and cheap sufficed repeatedly for a work-kind
  When the confirming outcome lands
  Then a note stating the confirmed approach exists in the vault
  And the transcript shows the orchestrator invoked no learn command
```

#### 793a7k2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17o The loop improves the rubric](../../specs/product/route.md#792a17o-the-loop-improves-the-rubric): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7k2a | [792a17o1a A confirming tier outcome is a confirmed approach](../../specs/product/route.md#792a17o1-uat-scenarios) | Scenario: 792a17o1a A confirming tier outcome is a confirmed approach<br>Given the Integration is not installed<br>And cheap sufficed repeatedly for a work-kind<br>When the orchestrator crystallizes the finding<br>Then it records a confirmed approach |
| 793a7k2b | [792a17o1b An overturned tier assumption is a reversal](../../specs/product/route.md#792a17o1-uat-scenarios) | Scenario: 792a17o1b An overturned tier assumption is a reversal<br>Given the Integration is not installed<br>And evidence that overturns a prior tier assumption<br>When the orchestrator crystallizes the finding<br>Then it records a reversal |

### 793a7l Modified: The seven steps are tracked, ordered and non-waivable

Target: [792a18c The seven steps are tracked, ordered and non-waivable](../../specs/product/please.md#792a18c-the-seven-steps-are-tracked-ordered-and-non-waivable)

Requirement: 792a18c While the Integration is installed, please MUST NOT perform the workflow's first step (the opening learn), its seventh step (the closing learn), the lessons audit or the which-artifact-should-have-surfaced-it question, and the System MUST perform them ([793a3o Please's lessons audit and surfacing question are performed by the System](#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system)). At the start of execution, please MUST push steps 2 to 6 to the task list in a single `TaskCreate` call, keeping their numbers, mark each step in progress when it begins and completed when it finishes, and start each step only after the previous step is completed. A step MUST be marked not applicable only when its mechanism does not exist in the environment (for example step 6's commit when the repository is not under version control), with a one-line rationale naming the missing mechanism written into the task description. "Too small to bother", "nothing to document" and "no memories will be relevant" MUST NOT count. The user MUST NOT be able to waive steps by saying "no ceremony", "skip the plan" or "hurry". When the user interrupts and redirects, please MUST update the task list to the new shape and MUST NOT abandon open steps without an explicit close. Where the Integration is not installed: At the start of execution, please MUST push all seven steps to the task list in a single `TaskCreate` call, mark each step in progress when it begins and completed when it finishes, and start each step only after the previous step is completed. A step MUST be marked not applicable only when its mechanism does not exist in the environment (for example step 6's commit when the repository is not under version control, or step 7's closing learn when the engram binary is absent), with a one-line rationale naming the missing mechanism written into the task description. "Too small to bother", "nothing to document" and "no memories will be relevant" MUST NOT count. The user MUST NOT be able to waive steps by saying "no ceremony", "skip the plan" or "hurry". When the user interrupts and redirects, please MUST update the task list to the new shape and MUST NOT abandon open steps without an explicit close.

Rationale: Steps 1 and 7 are learn and nothing else, and the audit and the question are steps of capturing, so the System performing them leaves five tracked steps.

#### 793a7l1 UAT Scenarios

```gherkin
Scenario: 793a7l1a The Integration leaves five tracked steps
  Given the Integration is installed and a valid ask
  When please begins execution
  Then the task list gets steps 2 to 6 from one TaskCreate call and each moves in progress and then completed in order
  And the transcript shows please ran no learn, no audit and no which-artifact-should-have-surfaced-it question

Scenario: 793a7l1b The Integration still does not let the user waive steps
  Given the Integration is installed and the user says "no ceremony, skip the plan"
  When the agent proceeds
  Then it runs the remaining steps, including the plan
```

#### 793a7l2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a18c The seven steps are tracked, ordered and non-waivable](../../specs/product/please.md#792a18c-the-seven-steps-are-tracked-ordered-and-non-waivable): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7l2a | [792a18c1a Seven tasks in one call](../../specs/product/please.md#792a18c1-uat-scenarios) | Scenario: 792a18c1a Seven tasks in one call<br>Given the Integration is not installed<br>And a valid ask<br>When please begins execution<br>Then the task list gets seven tasks from one TaskCreate call and each moves in progress and then completed in order |
| 793a7l2b | [792a18c1b No step starts early](../../specs/product/please.md#792a18c1-uat-scenarios) | Scenario: 792a18c1b No step starts early<br>Given the Integration is not installed<br>And step 1's learn is still in progress<br>When the agent considers reading repository docs<br>Then it waits until step 1 is completed |
| 793a7l2c | [792a18c1c A missing mechanism is the only N/A](../../specs/product/please.md#792a18c1-uat-scenarios) | Scenario: 792a18c1c A missing mechanism is the only N/A<br>Given the Integration is not installed<br>And a repository not under version control<br>When the agent reaches step 6<br>Then it marks the commit not applicable with the rationale "no VCS" written into the task description |
| 793a7l2d | [792a18c1d Small is not N/A](../../specs/product/please.md#792a18c1-uat-scenarios) | Scenario: 792a18c1d Small is not N/A<br>Given the Integration is not installed<br>And a change that touched no documentation<br>When the agent reaches step 5<br>Then it runs the step, which may produce nothing, and does not mark it not applicable |
| 793a7l2e | [792a18c1e Hurry does not waive](../../specs/product/please.md#792a18c1-uat-scenarios) | Scenario: 792a18c1e Hurry does not waive<br>Given the Integration is not installed<br>And the user says "no ceremony, skip the plan"<br>When the agent proceeds<br>Then it runs the workflow, including the plan |
| 793a7l2f | [792a18c1f A redirect updates the list](../../specs/product/please.md#792a18c1-uat-scenarios) | Scenario: 792a18c1f A redirect updates the list<br>Given the Integration is not installed<br>And the user redirects mid-workflow<br>When the agent continues<br>Then the task list reflects the new shape and no open step is dropped without an explicit close |

### 793a7m Modified: The workflow opens with learn and orients with recall

Target: [792a18d The workflow opens with learn and orients with recall](../../specs/product/please.md#792a18d-the-workflow-opens-with-learn-and-orients-with-recall)

Requirement: 792a18d While the Integration is installed, please MUST begin with step 2 and MUST NOT run the `learn` skill or recall, because the System performs both. Step 2 MUST loop until the ask is understood: it MUST read the repository's relevant markdown (the project instructions, the files under `docs/`, any glossary, architecture or design notes), MUST ask the user clarifying questions about intent or new concepts through `AskUserQuestion`, MUST NOT move on while material doubt remains about the ask's meaning or whether it should be done as stated, and MUST state the agent's own assessment of the ask (sound, flawed or underspecified). Where the Integration is not installed: Please MUST begin with step 1, the `learn` skill, to preserve anything pending from the session so far, before starting new work. Step 2 MUST begin with a literal invocation of recall, with queries derived from the ask, and MUST NOT substitute reading the working tree for it ("I already know this repo", "small ask", "I'll just grep" are the moments recall matters most): file reads surface working-tree content and recall surfaces vault content. Step 2 MUST loop until the ask is understood: it MUST read the repository's relevant markdown (the project instructions, the files under `docs/`, any glossary, architecture or design notes), MUST ask the user clarifying questions about intent or new concepts through `AskUserQuestion`, MUST NOT move on while material doubt remains about the ask's meaning or whether it should be done as stated, and MUST state the agent's own assessment of the ask (sound, flawed or underspecified).

Rationale: The opening learn and the orienting recall are performed by the System; the orientation loop is please's own.

#### 793a7m1 UAT Scenarios

```gherkin
Scenario: 793a7m1a The Integration opens with orientation
  Given the Integration is installed and a valid ask that matches a note
  When please starts
  Then the transcript shows the note's content delivered before please's first response or tool call
  And the transcript shows please ran neither the learn skill nor recall

Scenario: 793a7m1b The Integration still loops orientation on doubt
  Given the Integration is installed and an ask whose meaning is unclear after reading
  When the agent orients
  Then it asks the user and repeats the loop until understanding is solid
```

#### 793a7m2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a18d The workflow opens with learn and orients with recall](../../specs/product/please.md#792a18d-the-workflow-opens-with-learn-and-orients-with-recall): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7m2a | [792a18d1a Learn first](../../specs/product/please.md#792a18d1-uat-scenarios) | Scenario: 792a18d1a Learn first<br>Given the Integration is not installed<br>And a valid ask<br>When please starts<br>Then the first thing it runs is the learn skill |
| 793a7m2b | [792a18d1b Recall is literal](../../specs/product/please.md#792a18d1-uat-scenarios) | Scenario: 792a18d1b Recall is literal<br>Given the Integration is not installed<br>And a small ask whose diff is right there<br>When the agent orients<br>Then it invokes recall and also reads files, and does not read files instead of recalling |
| 793a7m2c | [792a18d1c Orientation loops on doubt](../../specs/product/please.md#792a18d1-uat-scenarios) | Scenario: 792a18d1c Orientation loops on doubt<br>Given the Integration is not installed<br>And an ask whose meaning is unclear after recall and reading<br>When the agent orients<br>Then it asks the user and repeats the loop until understanding is solid |

### 793a7n Modified: Findings are argued to resolution

Target: [792a18n Findings are argued to resolution](../../specs/product/please.md#792a18n-findings-are-argued-to-resolution)

Requirement: 792a18n While the Integration is installed, the orchestrator MUST NOT instruct a reviewer to recall first, because the System makes what recall returns for the reviewer's task available to it ([793a2c Moments are noticed without the agent](#793a2c-moments-are-noticed-without-the-agent)). Each reviewer's findings MUST be refutations with concrete stakes (a quote or `file:line`, why it fails the angle, and what better looks like), and a clean pass MUST state what was checked. The author MUST address every finding by fixing it or rebutting it with reasons, the reviewer MUST ACK or counter, and the reviewer MUST be kept alive until its gate closes. A finding MUST NOT be dropped silently. After about two unresolved rounds, please MUST stop, summarize both positions and escalate through `AskUserQuestion`. The gate itself (a fresh reviewer per angle, argued to resolution) MUST NOT be waived by the availability of a review-focused skill, which a reviewer may apply as its discipline source. An angle MUST be marked not applicable only when its subject is absent from the environment (no diagrams in the repository, no documents touched) and the missing subject MUST be named out loud; "the artifact is small" MUST NOT be a skip. Where the Integration is not installed: Each reviewer MUST recall first, with phrases drawn from the artifact and its angle, before it reviews. Its findings MUST be refutations with concrete stakes (a quote or `file:line`, why it fails the angle, and what better looks like), and a clean pass MUST state what was checked. The author MUST address every finding by fixing it or rebutting it with reasons, the reviewer MUST ACK or counter, and the reviewer MUST be kept alive until its gate closes. A finding MUST NOT be dropped silently. After about two unresolved rounds, please MUST stop, summarize both positions and escalate through `AskUserQuestion`. The gate itself (a fresh reviewer per angle, recall-first, argued to resolution) MUST NOT be waived by the availability of a review-focused skill, which a reviewer may apply as its discipline source. An angle MUST be marked not applicable only when its subject is absent from the environment (no diagrams in the repository, no documents touched) and the missing subject MUST be named out loud; "the artifact is small" MUST NOT be a skip.

Rationale: A reviewer's recall is the System's, and the rest of the gate does not depend on it.

#### 793a7n1 UAT Scenarios

```gherkin
Scenario: 793a7n1a The Integration's reviewer is not told to recall
  Given the Integration is installed and a note matching a reviewer's artifact and angle
  When the reviewer is dispatched
  Then its instructions do not tell it to recall first
  And its transcript shows the note's content delivered before its first response or tool call

Scenario: 793a7n1b The Integration still argues every finding to resolution
  Given the Integration is installed and a reviewer that raises three findings
  When the author responds
  Then each is fixed or rebutted to the reviewer's ACK and none is dropped
```

#### 793a7n2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a18n Findings are argued to resolution](../../specs/product/please.md#792a18n-findings-are-argued-to-resolution): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7n2a | [792a18n1a A reviewer recalls first](../../specs/product/please.md#792a18n1-uat-scenarios) | Scenario: 792a18n1a A reviewer recalls first<br>Given the Integration is not installed<br>And a dispatched reviewer<br>When it starts<br>Then its first action is to recall with phrases from the artifact and its angle |
| 793a7n2b | [792a18n1b Every finding is fixed or rebutted](../../specs/product/please.md#792a18n1-uat-scenarios) | Scenario: 792a18n1b Every finding is fixed or rebutted<br>Given the Integration is not installed<br>And a reviewer raises three findings<br>When the author responds<br>Then each is fixed or rebutted to the reviewer's ACK and none is dropped |
| 793a7n2c | [792a18n1c A deadlock is escalated](../../specs/product/please.md#792a18n1-uat-scenarios) | Scenario: 792a18n1c A deadlock is escalated<br>Given the Integration is not installed<br>And a finding that is still unresolved after two rounds<br>When the author reaches the limit<br>Then please summarizes both positions and asks the user through AskUserQuestion |
| 793a7n2d | [792a18n1d An absent subject is named](../../specs/product/please.md#792a18n1-uat-scenarios) | Scenario: 792a18n1d An absent subject is named<br>Given the Integration is not installed<br>And a repository with no diagrams<br>When gate A reaches the diagrams angle<br>Then it marks the angle not applicable and says out loud that no diagrams exist |

### 793a7o Modified: The closing learn is preceded by a lessons audit

Target: [792a18p The closing learn is preceded by a lessons audit](../../specs/product/please.md#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit)

Requirement: 792a18p While the Integration is installed, please MUST NOT audit the cycle's mechanical corpus, MUST NOT map items to vault notes and MUST NOT show an audit list in its closing report; the System performs the audit from the transcript, following the enumeration, mapping and reversal handoff of the branch below ([793a3o Please's lessons audit and surfacing question are performed by the System](#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system)); the user sees the counts of the audit's memory events in the System's summary ([793a5h A memory-activity summary is shown to the user](#793a5h-a-memory-activity-summary-is-shown-to-the-user)), and the per-item list stays in the log and the vault. Where the Integration is not installed: Before the closing learn at step 7, please MUST audit the cycle's mechanical corpus: every pre-registered stop that fired, every gate FAIL verdict, every commit whose message contains CORRECTION, supersede or superseded, instrument-invalid, or redraw or redrawn, and every mid-cycle escalation to the user. It MUST map each item to the vault note that captures its lesson, or write the one-line "no lesson: <why>"; MUST treat an unmapped item as a reversal handoff for the closing learn ([792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson)); and MUST show the audit list (item → note or no-lesson) in the cycle's closing report to the user. It MUST NOT use the audit for positive reinforcement, which has no mechanical marker and belongs to the closing learn's confirmed-approach scan.

Rationale: The audit exists to feed a capture, so the System performs it where it captures. What the System does with an unmapped item is [793a3o Please's lessons audit and surfacing question are performed by the System](#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system).

#### 793a7o1 UAT Scenarios

```gherkin
Scenario: 793a7o1d The Integration's please runs no audit
  Given the Integration is installed and a cycle with one fired stop, one gate FAIL, one commit named CORRECTION and one escalation
  When please reaches its closing report
  Then the transcript shows please enumerated and mapped no item
  And the closing report carries no audit list
```

#### 793a7o2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a18p The closing learn is preceded by a lessons audit](../../specs/product/please.md#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7o2a | [792a18p1a The corpus is enumerated](../../specs/product/please.md#792a18p1-uat-scenarios) | Scenario: 792a18p1a The corpus is enumerated<br>Given the Integration is not installed<br>And a cycle with one fired stop, one gate FAIL, one commit named CORRECTION and one escalation<br>When please runs the audit<br>Then all four items are on the list |
| 793a7o2b | [792a18p1b Each item maps or says why not](../../specs/product/please.md#792a18p1-uat-scenarios) | Scenario: 792a18p1b Each item maps or says why not<br>Given the Integration is not installed<br>And an enumerated item<br>When the audit maps it<br>Then it names the vault note that captures the lesson or writes "no lesson: &lt;why&gt;" |
| 793a7o2c | [792a18p1c An unmapped item becomes a reversal handoff](../../specs/product/please.md#792a18p1-uat-scenarios) | Scenario: 792a18p1c An unmapped item becomes a reversal handoff<br>Given the Integration is not installed<br>And an item with no vault note<br>When the closing learn runs<br>Then the item is handed to learn as a reversal |
| 793a7o2d | [792a18p1d The list is in the closing report](../../specs/product/please.md#792a18p1-uat-scenarios) | Scenario: 792a18p1d The list is in the closing report<br>Given the Integration is not installed<br>And a finished audit<br>When please reports to the user<br>Then the report includes the item → note-or-no-lesson list |

### 793a7p Modified: A lesson about to be captured asks what should have surfaced it

Target: [792a18q A lesson about to be captured asks what should have surfaced it](../../specs/product/please.md#792a18q-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it)

Requirement: 792a18q While the Integration is installed, please captures no lesson, so it MUST NOT ask which existing artifact should have surfaced a lesson, MUST NOT search the vault for a covering note before writing one, and MUST NOT reword a note whose situation line does not match; the System asks and answers them where it captures lessons ([793a3o Please's lessons audit and surfacing question are performed by the System](#793a3o-pleases-lessons-audit-and-surfacing-question-are-performed-by-the-system)), under learn's own rules. Please MUST NOT file automatically a change that ships from the repository (code, skills, guidance). Where the Integration is not installed: For each lesson the cycle is about to capture, please MUST ask which artifact that already existed should have surfaced it first, MUST search the vault for a note that already covers the situation before writing a new one, and MUST, when a note existed but did not surface at the moment it was needed, check whether its `situation:` line matches how the moment actually presented and reword the note when it does not, in preference to writing a second note on the same lesson. It MUST propose a change that ships from the repository (code, skills, guidance) only as a last resort, and MUST NOT file it automatically.

Rationale: The question, the search and the reword belong to capturing a lesson, which please no longer does; the System performs them.

#### 793a7p1 UAT Scenarios

```gherkin
Scenario: 793a7p1a The Integration's please asks, searches and rewords nothing
  Given the Integration is installed and a cycle that contained a lesson
  When please closes
  Then the transcript shows please asked no which-artifact-should-have-surfaced-it question, ran no vault search for a covering note and reworded no note
  And a note stating the lesson exists in the vault

Scenario: 793a7p1b The Integration's please files no repository change
  Given the Integration is installed and a lesson that a repository change could prevent
  When please closes
  Then no issue or change for it was filed automatically
```

#### 793a7p2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a18q A lesson about to be captured asks what should have surfaced it](../../specs/product/please.md#792a18q-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7p2a | [792a18q1a An existing note is searched for first](../../specs/product/please.md#792a18q1-uat-scenarios) | Scenario: 792a18q1a An existing note is searched for first<br>Given the Integration is not installed<br>And a lesson about to be captured<br>When please prepares the capture<br>Then it asks which existing artifact should have surfaced it and searches the vault for a note covering the situation |
| 793a7p2b | [792a18q1b A stale situation line is reworded](../../specs/product/please.md#792a18q1-uat-scenarios) | Scenario: 792a18q1b A stale situation line is reworded<br>Given the Integration is not installed<br>And a note existed but did not surface because its situation line does not match how the moment presented<br>When please resolves it<br>Then it rewords the note and does not write a second note on the same lesson |
| 793a7p2c | [792a18q1c A repository change is a last resort](../../specs/product/please.md#792a18q1-uat-scenarios) | Scenario: 792a18q1c A repository change is a last resort<br>Given the Integration is not installed<br>And a lesson that a repository change could prevent<br>When please considers it<br>Then it proposes the change only as a last resort and files nothing automatically |

### 793a7q Modified: Each dispatch is also written to the vault as a tagged evidence note

Target: [792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note)

Requirement: 792a17k While the Integration is installed, the System MUST write this note as [793a3p Route's evidence writes are performed by the System](#793a3p-routes-evidence-writes-are-performed-by-the-system) says, following the branch below with the System in the orchestrator's place (except its sentences addressed to the orchestrator), and the orchestrator MUST NOT write it. Where the Integration is not installed: After each dispatch resolves, with the review verdict in hand, the orchestrator MUST hand a fact to the write-memory skill as a field handoff and not as a command it composes, with the slug `route-dispatch-<work-kind>`, the tags `work-kind/<k>`, `tier/<cheap|mid|deep>` and `outcome/<pass|fail>`, the situation "routing <work-kind> work", the subject "<work-kind> dispatch at <tier> (<model @ dispatch>)", the predicate "resolved as", an object that states the verdict, the reason, the escalation, the duration and the cost with their units, and the source "route dispatch record, <project>, <date>". It MUST state the fields as written and MUST NOT invent a flag for them. The tags MUST be the three closed families only, work-kind an open kebab-case set, tier and outcome closed sets; duration and cost MUST stay in the object prose and never become tags. Producing only the table row MUST NOT count as recording.

Rationale: The evidence note is a measurement the System writes from the dispatch's report row, so the orchestrator writes no memory ([793a3p Route's evidence writes are performed by the System](#793a3p-routes-evidence-writes-are-performed-by-the-system)).

#### 793a7q1 UAT Scenarios

```gherkin
Scenario: 793a7q1a The Integration writes a passing dispatch's evidence note
  Given the Integration is installed and a dispatch with a passing verdict whose report row is in the transcript
  When the row appears
  Then a fact tagged work-kind/<k>, tier/<t> and outcome/pass exists in the vault with the dispatch details in its object
  And the transcript shows the orchestrator handed nothing to write-memory and wrote no note

Scenario: 793a7q1b The Integration tags a failing dispatch fail with its escalation
  Given the Integration is installed and a dispatch that fails and escalates, whose report row is in the transcript
  When the row appears
  Then the note is tagged outcome/fail and its object names the escalation tier

Scenario: 793a7q1c The Integration validates the tag format
  Given the Integration is installed and a dispatch row
  When the System writes its tags
  Then each is work-kind/<k>, tier/<t> or outcome/<o> with only lowercase letters, digits and hyphens, and an invalid tag is rejected before the write
  And duration and cost appear in the object and in no tag

Scenario: 793a7q1d The Integration's handoff states fields, not flags
  Given the Integration is installed and a dispatch row
  When the System hands the evidence note to write-memory
  Then the log records the handoff with each field as written and no invented flag
```

#### 793a7q2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17k Each dispatch is also written to the vault as a tagged evidence note](../../specs/product/route.md#792a17k-each-dispatch-is-also-written-to-the-vault-as-a-tagged-evidence-note): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7q2a | [792a17k1a A passing dispatch gets a tagged note](../../specs/product/route.md#792a17k1-uat-scenarios) | Scenario: 792a17k1a A passing dispatch gets a tagged note<br>Given the Integration is not installed<br>And a dispatch with a passing verdict<br>When the orchestrator records it<br>Then write-memory receives a fact tagged work-kind/&lt;k&gt;, tier/&lt;t&gt; and outcome/pass with the dispatch details in the object |
| 793a7q2b | [792a17k1b A failing dispatch is tagged fail with its escalation](../../specs/product/route.md#792a17k1-uat-scenarios) | Scenario: 792a17k1b A failing dispatch is tagged fail with its escalation<br>Given the Integration is not installed<br>And a dispatch that fails and escalates<br>When the orchestrator records it<br>Then the note is tagged outcome/fail and its object names the escalation tier |
| 793a7q2c | [792a17k1c The tag format is validated](../../specs/product/route.md#792a17k1-uat-scenarios) | Scenario: 792a17k1c The tag format is validated<br>Given the Integration is not installed<br>And a dispatch record<br>When its tags are written<br>Then each is work-kind/&lt;k&gt;, tier/&lt;t&gt; or outcome/&lt;o&gt; with only lowercase letters, digits and hyphens, and an invalid tag is rejected before the write |
| 793a7q2d | [792a17k1d Fields, not flags](../../specs/product/route.md#792a17k1-uat-scenarios) | Scenario: 792a17k1d Fields, not flags<br>Given the Integration is not installed<br>And the orchestrator hands off the evidence note<br>When it states the handoff<br>Then it states the fields as written and invents no flag such as --kind |

### 793a7r Modified: Each work-kind keeps one aggregate note

Target: [792a17l Each work-kind keeps one aggregate note](../../specs/product/route.md#792a17l-each-work-kind-keeps-one-aggregate-note)

Requirement: 792a17l While the Integration is installed, the System MUST keep this aggregate as [793a3p Route's evidence writes are performed by the System](#793a3p-routes-evidence-writes-are-performed-by-the-system) says, following the branch below with the System in the orchestrator's place (except its sentences addressed to the orchestrator), and the orchestrator MUST NOT write it. Where the Integration is not installed: After the evidence note, the orchestrator MUST amend or create the aggregate fact `route-evidence-<work-kind>`, composing and running that write itself and not handing it to write-memory, which has no amend form; this is a stated, structurally forced exception to the write-site doctrine and the skill MUST say so. It MUST look the aggregate up with `engram query --lazy-chunks --phrase="route evidence <work-kind> tier tally"` and take a returned path as a match only when its basename, without `.md` and split on `.`, ends in a slug equal to `route-evidence-<work-kind>` exactly. On a match it MUST recompute the tally from the aggregate's current text plus this dispatch and run `engram amend --target <basename> --object="..."`, keeping the existing evidence wikilinks and appending the new one. On no match it MUST first run a uniqueness check, a listing of vault basenames matching `route-evidence-<work-kind>*` under the same exact rule, and create the aggregate (`engram learn fact --slug route-evidence-<work-kind> --position top`, with a tier tally of 1/1 and a wikilink to the evidence note) only when the check finds nothing; when it finds more than one, it MUST merge them by uniting their evidence wikilinks, recomputing the tally with the count commands of [792a17m Counting audits the tallies and never routes](../../specs/product/route.md#792a17m-counting-audits-the-tallies-and-never-routes), amending one into the merged aggregate and recording the incident as drowning-remedy evidence. An aggregate MUST NOT carry the work-kind, tier or outcome tags. Its wikilink list MUST be the append-only evidence trail. When recording a dispatch before the lookup has run, the orchestrator MUST state both candidate commands and not guess.

Rationale: The aggregate is kept by the System from the same row ([793a3p Route's evidence writes are performed by the System](#793a3p-routes-evidence-writes-are-performed-by-the-system)); the orchestrator writes none.

#### 793a7r1 UAT Scenarios

```gherkin
Scenario: 793a7r1a The Integration creates the first aggregate
  Given the Integration is installed, no aggregate exists for the work-kind and the uniqueness check finds nothing
  When the first dispatch's row appears in the transcript
  Then the System creates route-evidence-<work-kind> with the tally from that dispatch, a link to its evidence note and no tags
  And the transcript shows the orchestrator wrote no aggregate

Scenario: 793a7r1b The Integration amends a later aggregate
  Given the Integration is installed and an aggregate for the work-kind
  When a subsequent dispatch's row appears
  Then the System amends the aggregate, recomputes the tier tally and appends the new evidence wikilink while keeping the old ones
  And the transcript shows the orchestrator wrote no aggregate

Scenario: 793a7r1c The Integration does not count a fuzzy match as the aggregate
  Given the Integration is installed and a lookup that returns a note whose slug only begins with route-evidence-<work-kind>
  When the System applies the match rule
  Then the note is not treated as the aggregate

Scenario: 793a7r1d The Integration checks a no-match result before creating
  Given the Integration is installed and a lookup that returns no match
  When the System would create an aggregate
  Then the log records a listing of vault basenames matching route-evidence-<work-kind>* before the aggregate is created

Scenario: 793a7r1e The Integration merges duplicate aggregates
  Given the Integration is installed and a uniqueness check that finds two aggregates for one work-kind
  When the System resolves it
  Then it unites their evidence wikilinks, recomputes the tally by count, amends one into the merged aggregate and records the incident
```

#### 793a7r2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17l Each work-kind keeps one aggregate note](../../specs/product/route.md#792a17l-each-work-kind-keeps-one-aggregate-note): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7r2a | [792a17l1a The first dispatch creates the aggregate](../../specs/product/route.md#792a17l1-uat-scenarios) | Scenario: 792a17l1a The first dispatch creates the aggregate<br>Given the Integration is not installed<br>And no aggregate exists for the work-kind and the uniqueness check finds nothing<br>When the first dispatch is recorded<br>Then the orchestrator creates route-evidence-&lt;work-kind&gt; with the tally from that dispatch, a link to its evidence note and no tags |
| 793a7r2b | [792a17l1b A later dispatch amends it](../../specs/product/route.md#792a17l1-uat-scenarios) | Scenario: 792a17l1b A later dispatch amends it<br>Given the Integration is not installed<br>And an aggregate for the work-kind<br>When a subsequent dispatch is recorded<br>Then the orchestrator amends the aggregate, recomputes the tier tally and appends the new evidence wikilink while keeping the old ones |
| 793a7r2c | [792a17l1c A fuzzy match is not a match](../../specs/product/route.md#792a17l1-uat-scenarios) | Scenario: 792a17l1c A fuzzy match is not a match<br>Given the Integration is not installed<br>And a lookup that returns a note whose slug only begins with route-evidence-&lt;work-kind&gt;<br>When the orchestrator applies the match rule<br>Then it does not count as the aggregate |
| 793a7r2d | [792a17l1d A no-match result is checked before creating](../../specs/product/route.md#792a17l1-uat-scenarios) | Scenario: 792a17l1d A no-match result is checked before creating<br>Given the Integration is not installed<br>And the lookup returns no match<br>When the orchestrator would create an aggregate<br>Then it first lists vault basenames matching route-evidence-&lt;work-kind&gt;* to rule out a missed lookup |
| 793a7r2e | [792a17l1e Duplicates are merged](../../specs/product/route.md#792a17l1-uat-scenarios) | Scenario: 792a17l1e Duplicates are merged<br>Given the Integration is not installed<br>And the uniqueness check finds two aggregates for one work-kind<br>When the orchestrator resolves it<br>Then it unites their evidence wikilinks, recomputes the tally by count, amends one into the merged aggregate and records the incident |
| 793a7r2f | [792a17l1f The write-path split is stated](../../specs/product/route.md#792a17l1-uat-scenarios) | Scenario: 792a17l1f The write-path split is stated<br>Given the Integration is not installed<br>And an orchestrator reading the aggregate procedure<br>When it reaches the aggregate write<br>Then the skill says the evidence note goes through write-memory while the aggregate is composed directly, because write-memory has no amend form |

### 793a7s Modified: A memory-backed unit drops one tier

Target: [792a17d A memory-backed unit drops one tier](../../specs/product/route.md#792a17d-a-memory-backed-unit-drops-one-tier)

Requirement: 792a17d While the Integration is installed, a unit whose needed knowledge is in the orchestrator's context from the System's recall (a known convention, a prior decision, a crystallized diagnostic) MUST be routed one tier cheaper than it otherwise would be, floored at the cheap tier. Where the Integration is not installed: A unit whose needed knowledge is recallable (a known convention, a prior decision, a crystallized diagnostic) MUST be routed one tier cheaper than it otherwise would be, floored at the cheap tier.

Rationale: The orchestrator runs no recall, so knowledge is available to it only through what the System delivered ([793a2c Moments are noticed without the agent](#793a2c-moments-are-noticed-without-the-agent)).

#### 793a7s1 UAT Scenarios

```gherkin
Scenario: 793a7s1a The discount applies to knowledge the System delivered
  Given the Integration is installed and a unit whose needed convention is a note the System delivered to the orchestrator at its decision to dispatch, and evidence that selects the mid tier
  When the orchestrator routes it
  Then it starts at the cheap tier and records "memory-discount applied" as the reason

Scenario: 793a7s1b Knowledge that is not in the orchestrator's context earns no discount
  Given the Integration is installed and a unit whose needed convention is a note in the vault that is not in the orchestrator's context, and evidence that selects the mid tier
  When the orchestrator routes it
  Then it starts at the mid tier and records no memory discount

Scenario: 793a7s1c The discount stops at cheap
  Given the Integration is installed and a unit whose needed convention is in the orchestrator's context and that would start at the cheap tier
  When the orchestrator routes it
  Then it starts at the cheap tier
```

#### 793a7s2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17d A memory-backed unit drops one tier](../../specs/product/route.md#792a17d-a-memory-backed-unit-drops-one-tier): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7s2a | [792a17d1a The discount applies](../../specs/product/route.md#792a17d1-uat-scenarios) | Scenario: 792a17d1a The discount applies<br>Given the Integration is not installed<br>And a unit whose needed convention is already a recallable note, and evidence that selects the mid tier<br>When the orchestrator routes it<br>Then it starts at the cheap tier and records "memory-discount applied" as the reason |
| 793a7s2b | [792a17d1b The discount stops at cheap](../../specs/product/route.md#792a17d1-uat-scenarios) | Scenario: 792a17d1b The discount stops at cheap<br>Given the Integration is not installed<br>And a memory-backed unit that would start at the cheap tier<br>When the orchestrator routes it<br>Then it starts at the cheap tier |

### 793a7t Modified: Counting audits the tallies and never routes

Target: [792a17m Counting audits the tallies and never routes](../../specs/product/route.md#792a17m-counting-audits-the-tallies-and-never-routes)

Requirement: 792a17m While the Integration is installed, the System MUST perform the count audit and the aggregate amend, following the branch below with the System in the orchestrator's place (except its sentences addressed to the orchestrator), and the orchestrator MUST NOT run `engram count` and MUST NOT amend any aggregate. Where the Integration is not installed: The orchestrator MUST recompute aggregate tallies from the evidence notes' tags with `engram count --group-by tags --filter tags=tier/<t> --filter tags=outcome/pass` for the numerators and `engram count --group-by tags --filter tags=tier/<t>` for the denominators (reading the `work-kind/<k>` rows), or with `--filter tags=work-kind/<k> --filter tags=tier/<t>` and the `total:` line for one kind, when a tally is doubted and at periodic consolidation, and MUST amend an aggregate that disagrees to the recomputed numbers. It MUST NOT use count on the read path, MUST NOT group by `work-kind` (a tag value, not an attribute), and MUST NOT use `--group-by tier`, which runs without error but groups the note-tier frontmatter field (L1, L2, L3) and not the `tier/` tag family.

Rationale: The aggregate is kept by the System ([793a3p Route's evidence writes are performed by the System](#793a3p-routes-evidence-writes-are-performed-by-the-system)), so the audit that repairs it is the System's too; the orchestrator runs no count and amends nothing.

#### 793a7t1 UAT Scenarios

```gherkin
Scenario: 793a7t1a The System audits a doubted tally
  Given the Integration is installed and an aggregate whose tally is doubted
  When the audit moment comes
  Then the log records the System's count of the evidence tags for the work-kind and tier
  And the transcript shows the orchestrator ran no engram count

Scenario: 793a7t1b The System repairs a drifted aggregate
  Given the Integration is installed and count output that disagrees with an aggregate's prose tally
  When the System resolves it
  Then the aggregate in the vault now carries the recomputed numbers
  And the transcript shows the orchestrator amended no aggregate
```

#### 793a7t2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17m Counting audits the tallies and never routes](../../specs/product/route.md#792a17m-counting-audits-the-tallies-and-never-routes): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7t2a | [792a17m1a A work-kind is audited](../../specs/product/route.md#792a17m1-uat-scenarios) | Scenario: 792a17m1a A work-kind is audited<br>Given the Integration is not installed<br>And an aggregate whose tally is doubted<br>When the orchestrator audits tier routing parity for the work-kind<br>Then it runs engram count --group-by tags with the work-kind and tier filters to recompute the numerators from the evidence tags |
| 793a7t2b | [792a17m1b A drifted aggregate is repaired](../../specs/product/route.md#792a17m1-uat-scenarios) | Scenario: 792a17m1b A drifted aggregate is repaired<br>Given the Integration is not installed<br>And the count output disagrees with an aggregate's prose tally<br>When the orchestrator repairs it<br>Then it amends the aggregate's object to the recomputed numbers |
| 793a7t2c | [792a17m1c The footgun is avoided](../../specs/product/route.md#792a17m1-uat-scenarios) | Scenario: 792a17m1c The footgun is avoided<br>Given the Integration is not installed<br>And the orchestrator needs the count by evidence tier<br>When it composes the command<br>Then it uses --group-by tags --filter tags=tier/&lt;t&gt; and not --group-by tier |

### 793a7u Modified: Drowning is reported, not patched

Target: [792a17n Drowning is reported, not patched](../../specs/product/route.md#792a17n-drowning-is-reported-not-patched)

Requirement: 792a17n While the Integration is installed, the System MUST perform the aggregate lookup and the drowning detection at the same moments as the count audit, following the branch below with the System in the orchestrator's place (except its sentences addressed to the orchestrator), and MUST show a drowning report to the user, outside the working agent's context, as [793a5h A memory-activity summary is shown to the user](#793a5h-a-memory-activity-summary-is-shown-to-the-user) shows its summary; the orchestrator MUST NOT run the lookup. Where the Integration is not installed: At the same moments as the count audit, the orchestrator MUST run the aggregate lookup for a work-kind with many evidence notes and confirm that the aggregate's path still surfaces in the items or in any cluster's candidates; when it does not, it MUST report the pre-registered drowning case and MUST NOT patch the skill ad hoc.

Rationale: The lookup belongs to the aggregate's keeper. Where the report goes waits on [793c44 Where do the delegated skills' messages to the user go?](793c_open-questions.md#793c44-where-do-the-delegated-skills-messages-to-the-user-go); until the owner answers, it is shown to the user, which is that question's recommendation for an item that asks the user to act.

#### 793a7u1 UAT Scenarios

```gherkin
Scenario: 793a7u1a The System finds the aggregate among the results
  Given the Integration is installed and a work-kind with 20 evidence notes
  When the System runs the aggregate lookup at the audit moment
  Then the log records that the aggregate's path surfaced
  And the user is shown no drowning report
  And the transcript shows the orchestrator ran no aggregate lookup

Scenario: 793a7u1b A missing aggregate is shown to the user, not patched
  Given the Integration is installed and a lookup in which a known aggregate does not surface
  When the System audits
  Then the user is shown the pre-registered drowning report outside the working agent's context
  And the report appears nowhere in the working agent's transcript
  And route's skill files are not changed
```

#### 793a7u2 Canonical scenarios with the Integration not installed

Each row is a Modified operation against one canonical scenario of [792a17n Drowning is reported, not patched](../../specs/product/route.md#792a17n-drowning-is-reported-not-patched): the replacement text is the canonical scenario prefixed with "Given the Integration is not installed", and the target's ID is preserved.

| Operation | Target scenario | Replacement scenario |
| --- | --- | --- |
| 793a7u2a | [792a17n1a The aggregate still surfaces](../../specs/product/route.md#792a17n1-uat-scenarios) | Scenario: 792a17n1a The aggregate still surfaces<br>Given the Integration is not installed<br>And a work-kind with 20 evidence notes<br>When the orchestrator runs the lookup<br>Then the aggregate's path is among the results |
| 793a7u2b | [792a17n1b A missing aggregate is reported](../../specs/product/route.md#792a17n1-uat-scenarios) | Scenario: 792a17n1b A missing aggregate is reported<br>Given the Integration is not installed<br>And a lookup in which a known aggregate does not surface<br>When the orchestrator audits<br>Then it reports the drowning case and does not patch the skill |
