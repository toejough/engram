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
deliberate differences from the skills, and what is new.

Terms used below:

- **Working agent:** the model doing the user's task in a session, including any subagents it
  dispatches.
- **System:** the capability this document specifies. It runs alongside the working agent and acts for it.
- **Vault** and **note:** the engram note store and one entry in it (fact, feedback, runbook, or Q&A).
- **Moment:** a point at which recall or learn fires: the moments [ambient-guidance](../../specs/product/ambient-guidance.md) defines, every
  prompt ([793a2i Recall for every prompt regardless of source](#793a2i-recall-for-every-prompt-regardless-of-source)) and a subagent's dispatch
  ([793a2c Moments are noticed without the agent](#793a2c-moments-are-noticed-without-the-agent)). A **recall moment** is one at which recall fires.
- **Depth:** how thorough one recall is, `glance` or `deep`, as defined by [792a14c Recall has two depths](../../specs/product/recall.md#792a14c-recall-has-two-depths).
- **Lesson:** any corrected failure, including an agent's own mid-task mistake that evidence
  overturned, plus a correction, an explicit save-request and a confirmed approach. It is wider than
  the four kinds in [792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson) ([793a3n Any corrected failure is a lesson](#793a3n-any-corrected-failure-is-a-lesson)).
- **Injection:** placing note content, or a chunk stub, in front of the working agent without the agent requesting it.
- **Chunk stub:** a chunk shown as its id and score only, with no content, as a lazy query prints it ([792a7s Lazy chunks](../../specs/product/memory-query.md#792a7s-lazy-chunks)).
- **Integration:** what `engram update` installs for the System ([793a4h Installed by engram update](#793a4h-installed-by-engram-update)).
- **Context:** the content currently available to one agent, a top-level agent or one subagent.
  Content removed by compaction is no longer present. A subagent's context is separate from its
  parent's. A resumed session's restored content is present.
- **Operator:** the human who configures and audits the System.

Requirement keywords follow RFC 2119.

Sections [793a2 automatic-memory (delegation of recall)](#793a2-added-automatic-memory-delegation-of-recall) through [793a5 automatic-memory (coexistence and measurement)](#793a5-added-automatic-memory-coexistence-and-measurement) add requirements to one new capability.
[793a6 lessons-contract (the LESSONS line is replaced while the Integration is installed)](#793a6-modified-lessons-contract-the-lessons-line-is-replaced-while-the-integration-is-installed) modifies the lessons contract and the capabilities that name its LESSONS line, and [793a7 ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)](#793a7-modified-ambient-guidance-route-and-please-the-working-agent-no-longer-runs-recall-or-learn-while-the-integration-is-installed) modifies the nodes that tell the working agent to run recall or learn. A requirement carried
over from the draft keeps its draft identifier in a "formerly" note so that review comments on the
draft still map.

## 793a2 Added: automatic-memory (delegation of recall)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a2a Recall is performed for the working agent

Requirement: 793a2a (formerly REC-1) The System MUST perform recall on the working agent's behalf at each moment at which recall fires
([792a19a Recall fires at four decision moments](../../specs/product/ambient-guidance.md#792a19a-recall-fires-at-four-decision-moments), [792a19b Recall escalates from glance to deep](../../specs/product/ambient-guidance.md#792a19b-recall-escalates-from-glance-to-deep)) and at every prompt
([793a2i Recall for every prompt regardless of source](#793a2i-recall-for-every-prompt-regardless-of-source)), exactly as [recall](../../specs/product/recall.md) defines it: at the depth that
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

Scenario: 793a2a1b Recall's vault writes happen without the agent
  Given a deep recall moment whose results include a cluster that no existing note addresses
  When the moment completes
  Then a note covering the cluster exists in the vault
  And the transcript shows the working agent ran no learn, amend or write-memory command
  And the transcript shows no request to the working agent to write the note
```

### 793a2c Moments are noticed without the agent

Requirement: 793a2c (formerly REC-3) The System MUST notice each moment at which recall or learn fires without the working agent signalling it.
A subagent's dispatch is one such moment: the System MUST make what recall returns for the subagent's task available to that subagent.

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
```

### 793a2d Attributable presentation

Requirement: 793a2d (formerly REC-4) Injected items MUST identify their source: a note by its file name and a chunk by its id.

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

Requirement: 793a2e (formerly REC-5) The System MUST NOT inject an item (a note or a chunk) already
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

Requirement: 793a3a (formerly CAP-1) The System MUST perform learn on the working agent's behalf, in every working session, top-level or subagent
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

### 793a3k Use is inferred from the transcript

Requirement: 793a3k The System MUST infer from the session transcript which of the notes it injected the working agent went on to use, for example by quoting or citing the note,
following its steps, or taking the action it prescribes, because it cannot ask the agent, and MUST activate each of those notes as recall activates a used note
([792a14l Only the notes that were used are activated](../../specs/product/recall.md#792a14l-only-the-notes-that-were-used-are-activated)), in place of recall's own activation, which relies on the agent's declaration. The System MUST log each activation verdict,
activated or not, with its evidence, and MUST NOT activate an injected note for which the transcript shows no
evidence of use.

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

Requirement: 793a4b (formerly CTL-2) While the Integration is installed, if the System fails, times out, or is unavailable, the working
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

Requirement: 793a4c (formerly CTL-3) The System MUST NOT itself make or override tool permissions,
tool arguments, model choice, or the user's prompt text. Its only effect on the working session is the
items it injects; outside the session it writes and activates vault notes with their sidecars, refreshes the chunk index, pulls down pending copies of parent notes and
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

Scenario: 793a4c1c The only effect on the session is the injected items
  Given a session in which the System injected notes and captured a lesson
  When the session transcript is inspected
  Then the only content the System added to the session is the injected items

Scenario: 793a4c1d Effects outside the session are the System's own writes
  Given a vault with a configured parent
  And a session in which the System captured a lesson and activated an injected note from the parent vault
  When the files changed during the session outside the session transcript are listed
  Then the changes are the vault notes written or activated with their sidecars, the chunk index, the pending parent copies, the offers to the parent, the System's log and the sessions the System started
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
no operating mode to select and no step beyond installing it. When the Integration has been uninstalled, the System MUST
inject nothing, capture nothing and record nothing.

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
```

### 793a4h Installed by engram update

Requirement: 793a4h `engram update` MUST install the Integration, what the System needs to run, for each harness it
updates ([792a11g The harnesses that are updated](../../specs/product/install-and-update.md#792a11g-the-harnesses-that-are-updated)) in which the System can run ([793a5c Environments without the capability](#793a5c-environments-without-the-capability)), extending the install of [install-and-update](../../specs/product/install-and-update.md). An
existing install of engram MUST have the System after its next `engram update`, with no further step.
`engram update --dry-run` ([792a11f Previewing an update](../../specs/product/install-and-update.md#792a11f-previewing-an-update)) MUST install none of it.

Rationale: operators already run one command to bring a machine to the current engram, so the System arrives and stays
current through it, and an existing install needs no separate step.

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
```

## 793a5 Added: automatic-memory (coexistence and measurement)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

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

Requirement: 793a5f (formerly NFR-3) In a controlled comparison on lesson-bearing tasks, runs with
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

Requirement: 793a5g (formerly NFR-1) The System's work MUST NOT add delay to the working agent beyond what the agent would spend performing the same skill itself.
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

Scenario: 793a5g1c Slow capture does not stall the session
  Given capture evaluation takes 20 seconds
  When a turn completes and the user submits the next prompt immediately
  Then the next prompt is processed without waiting for the capture to finish
  And the working agent's response to the completed turn appeared without waiting for the capture
```

## 793a6 Modified: lessons-contract (the LESSONS line is replaced while the Integration is installed)

Canonical targets: [lessons-contract](../../specs/product/lessons-contract.md) and the capabilities that name its LESSONS line: [route](../../specs/product/route.md), [please](../../specs/product/please.md) and [ambient-guidance](../../specs/product/ambient-guidance.md). Each operation is a Modified operation against exactly one canonical node, conditional on the Integration, because the contract still holds in harnesses and installs without the Integration ([793a5c Environments without the capability](#793a5c-environments-without-the-capability)). Each target's ID is preserved. The replacement text carries the Integration branch first, then the target's text unchanged for an install without the Integration.

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

### 793a6f Modified: The closing learn gets the whole list

Target: [792a16f The closing learn gets the whole list](../../specs/product/lessons-contract.md#792a16f-the-closing-learn-gets-the-whole-list)

Requirement: 792a16f While the Integration is installed, please MUST NOT collect `LESSONS:` lines and does not run a closing learn ([793a7l The seven steps are tracked, ordered and non-waivable](#793a7l-modified-the-seven-steps-are-tracked-ordered-and-non-waivable)), so it hands no collection of them to one. Where the Integration is not installed: At workflow step 7, please MUST hand the full list of `LESSONS:` lines collected across its dispatches, `none` entries included, to the closing learn, alongside the lessons-audit map ([please](../../specs/product/please.md)), and MUST drop no line. Please MUST run the closing learn when the session had no dispatches, handing it an empty collection. The closing learn MUST skip a line that a vault note written earlier this session already covers, MUST judge every other line against the bar of [792a16e Each lesson is judged when it returns](../../specs/product/lessons-contract.md#792a16e-each-lesson-is-judged-when-it-returns), and MUST discard a line that does not clear it silently, with no note and no log. It MUST NOT write a second time a line captured at its return.

Rationale: There is no collection to hand over, and no closing learn to hand it to.

#### 793a6f1 UAT Scenarios

```gherkin
Scenario: 793a6f1a The closing learn is handed no list of LESSONS lines
  Given the Integration is installed and a session with three dispatches
  When please finishes step 6
  Then no list of LESSONS lines is handed to any learn
  And the lessons the three units contained are already in the vault
```

### 793a6g Modified: A confirmed approach is looked for at the moment of confirmation

Target: [792a16g A confirmed approach is looked for at the moment of confirmation](../../specs/product/lessons-contract.md#792a16g-a-confirmed-approach-is-looked-for-at-the-moment-of-confirmation)

Requirement: 792a16g While the Integration is installed, the clauses about a dispatched subagent's `LESSONS:` line and the collected lines do not apply, and the System scans for a confirmed approach, with the same bar, at the moment a unit of work's outcome is confirmed. Where the Integration is not installed: Learn MUST scan for a confirmed approach (kind 4) at the moment a unit of work's outcome is confirmed: a review or gate verdict lands, a check or test passes, or the user explicitly confirms something. That includes the moment the agent reads a dispatched subagent's completion report and its `LESSONS:` line once its outcome is known, and, at the closing learn, the session's completion reports and collected lines. The bar MUST NOT move: a resolved uncertainty or an explicit specific confirmation, never bare success.

Rationale: The scan is anchored on the confirmation, which the System sees in the transcript.

#### 793a6g1 UAT Scenarios

```gherkin
Scenario: 793a6g1a A confirmed approach is looked for without the lines
  Given the Integration is installed and a subagent's work was confirmed by a passing check that resolved an uncertainty
  When the confirmation lands
  Then a note stating the confirmed approach exists in the vault
  And no LESSONS line was read or collected
```

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

### 793a6i Modified: The closing learn gets the lessons and the audit

Target: [792a18r The closing learn gets the lessons and the audit](../../specs/product/please.md#792a18r-the-closing-learn-gets-the-lessons-and-the-audit)

Requirement: 792a18r While the Integration is installed, please MUST NOT run the `learn` skill at the end of the workflow, and MUST NOT hand a list of `LESSONS:` lines or the audit's map to one; the audit's result is in the closing report ([793a7o The closing learn is preceded by a lessons audit](#793a7o-modified-the-closing-learn-is-preceded-by-a-lessons-audit)). Where the Integration is not installed: Please MUST run the `learn` skill again at step 7 and hand it the full list of `LESSONS:` lines collected across step 4's dispatches ([792a16f The closing learn gets the whole list](../../specs/product/lessons-contract.md#792a16f-the-closing-learn-gets-the-whole-list)) together with the audit's map, and MUST NOT duplicate learn's ad-hoc question-and-answer capture ([792a15k An answered question is captured once](../../specs/product/learn.md#792a15k-an-answered-question-is-captured-once)), which learn performs for substantive answered questions.

Rationale: There are no collected lines to hand over and no closing learn to run.

#### 793a6i1 UAT Scenarios

```gherkin
Scenario: 793a6i1a The closing learn receives only the audit
  Given the Integration is installed and a finished cycle with an audit map
  When please reaches the end of the workflow
  Then the transcript shows please invoked no learn command
  And the audit's map is in the closing report and no list of LESSONS lines exists
```

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

### 793a6k Modified: A returned line that fails the bar is not captured

Target: [792a19f A returned line that fails the bar is not captured](../../specs/product/ambient-guidance.md#792a19f-a-returned-line-that-fails-the-bar-is-not-captured)

Requirement: 792a19f While the Integration is installed, no returned `LESSONS:` line exists, so there is nothing for the learn guidance to fire for or skip. Where the Integration is not installed: The learn guidance MUST NOT fire for a returned `LESSONS:` line that is `none`, for a lesson that is a bare success or routine narrative, or for a lesson that is an unconfirmed hypothesis, which are left for the closing learn to discard under the same bar.

Rationale: The requirement governs lines that are no longer returned.

#### 793a6k1 UAT Scenarios

```gherkin
Scenario: 793a6k1a No returned line exists to skip
  Given the Integration is installed and a subagent whose turn contained only routine success
  When the orchestrator reads the report
  Then the report carries no LESSONS line
  And no note is written for the routine success
```

## 793a7 Modified: ambient-guidance, route and please (the working agent no longer runs recall or learn while the Integration is installed)

Canonical targets: [ambient-guidance](../../specs/product/ambient-guidance.md), [route](../../specs/product/route.md) and [please](../../specs/product/please.md), the nodes that tell the working agent to run recall or learn, or to perform their steps ([793c31 Which canonical nodes that ask the agent to run recall or learn change while the Integration is installed?](793c_open-questions.md#793c31-which-canonical-nodes-that-ask-the-agent-to-run-recall-or-learn-change-while-the-integration-is-installed)). Each operation is a Modified operation against exactly one canonical node, conditional on the Integration, because the node still holds in harnesses and installs without the Integration ([793a5c Environments without the capability](#793a5c-environments-without-the-capability)). Each target's ID is preserved. The replacement text carries the Integration branch first, then the target's text unchanged for an install without the Integration, with its in-page references rewritten as links to the canonical file. Where the moments or criteria that decide when recall or learn fires are stated in a target, the Integration branch restates them, because the System uses them ([793a2a Recall is performed for the working agent](#793a2a-recall-is-performed-for-the-working-agent), [793a3a Learn is performed for the working agent](#793a3a-learn-is-performed-for-the-working-agent)).

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

Scenario: 793a7a1e Without the Integration the agent recalls before it endorses an approach
  Given the Integration is not installed and an agent with the recall guidance
  And the user proposes an approach and asks the agent what it thinks
  When the agent is about to evaluate it
  Then the transcript shows the agent ran `/recall glance` before it offered any analysis
```

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

Scenario: 793a7b1b Without the Integration the agent escalates a weighty decision
  Given the Integration is not installed and an agent with the recall guidance facing an irreversible decision
  When it recalls before making it
  Then the transcript shows it ran `/recall deep` and not only `/recall glance`
```

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

Scenario: 793a7c1d Without the Integration the agent captures a rejected approach before the fix
  Given the Integration is not installed and an agent with the learn guidance
  And the user says "no, do X not Y, the convention here is Z"
  When the agent is about to apply the fix
  Then the transcript shows it invoked the learn skill for the correction first
```

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

Scenario: 793a7d1c Without the Integration the agent captures with the learn skill
  Given the Integration is not installed and an agent with the learn guidance
  And a confirmed correction
  When the agent captures it
  Then the capture is a learn skill invocation that writes a vault note, and not a write to the harness's project memory or a plain file
  And the agent ran no `engram ingest --auto`
```

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
  And the transcript shows the agent was not required to run an engram query

Scenario: 793a7e1b Without the Integration the agent queries first
  Given the Integration is not installed and an agent with the shim in its config and no engram skill installed
  When it receives a user request
  Then its first tool call is an engram query carrying the request phrase, the situation phrase and --text with the user's message
```

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

Scenario: 793a7f1b Without the Integration the shim shapes the second phrase
  Given the Integration is not installed and an agent with the shim in its config
  And its draft second phrase contains a flag, file or endpoint name
  When it re-checks before running the query
  Then the second --phrase it passes names a class of work-handling decision and contains none of those nouns
```

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

Scenario: 793a7g1c Without the Integration the agent queries before work is declared done
  Given the Integration is not installed and an agent with the shim in its config about to declare work done
  When it prepares to
  Then the transcript shows it ran engram query keyed to the current work first, and then verified
```

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

Scenario: 793a7h1b Without the Integration the orchestrator runs the meta-skills
  Given the Integration is not installed and a multi-unit task
  When the orchestrator follows route
  Then the transcript shows the orchestrator ran recall and learn itself and dispatched the object-level work
```

### 793a7i Modified: The starting tier comes from recalled evidence, otherwise the cheapest tier

Target: [792a17b The starting tier comes from recalled evidence, otherwise the cheapest tier](../../specs/product/route.md#792a17b-the-starting-tier-comes-from-recalled-evidence-otherwise-the-cheapest-tier)

Requirement: 792a17b While the Integration is installed, before dispatching, the orchestrator MUST use the tier-performance evidence for this kind of work that the System's recall has already put in its context (the aggregates surface as ordinary memories), MUST NOT run recall to look for it, and MUST NOT use a special query or counting on the read path. Absent such evidence, it MUST start every unit at the cheapest, fastest available tier, with no exception for work that merely looks hard (hard debugging, a cross-cutting refactor, a correctness-critical review, a greenfield design). The starting tier MUST NOT be raised for any reason other than recalled evidence of failure at a cheaper tier, or a cold-start prior that records such evidence ([792a17f Cold-start priors carry only evidence-backed escalations](../../specs/product/route.md#792a17f-cold-start-priors-carry-only-evidence-backed-escalations)), and then only to the tier the evidence supports. Predicted difficulty ("genuinely complex", "needs careful reasoning", "broad impact") MUST NOT count as evidence. Where the Integration is not installed: Before dispatching, the orchestrator MUST check recalled memory for tier-performance evidence on this kind of work, using the plain recall of the agent (the aggregates surface as ordinary memories) and no special query or counting on the read path. Absent recalled evidence, it MUST start every unit at the cheapest, fastest available tier, with no exception for work that merely looks hard (hard debugging, a cross-cutting refactor, a correctness-critical review, a greenfield design). The starting tier MUST NOT be raised for any reason other than recalled evidence of failure at a cheaper tier, or a cold-start prior that records such evidence ([792a17f Cold-start priors carry only evidence-backed escalations](../../specs/product/route.md#792a17f-cold-start-priors-carry-only-evidence-backed-escalations)), and then only to the tier the evidence supports. Predicted difficulty ("genuinely complex", "needs careful reasoning", "broad impact") MUST NOT count as evidence.

Rationale: The evidence arrives through the System's recall instead of the orchestrator's own. Whether it arrives before routing is [793c39 Does tier evidence reach the orchestrator before it routes?](793c_open-questions.md#793c39-does-tier-evidence-reach-the-orchestrator-before-it-routes).

#### 793a7i1 UAT Scenarios

```gherkin
Scenario: 793a7i1a The Integration routes on evidence the System delivered
  Given the Integration is installed and the System delivered an aggregate showing that cheap failed for a work-kind
  When the orchestrator routes a unit of that work-kind
  Then the unit starts at the tier the evidence supports
  And the transcript shows the orchestrator ran no recall

Scenario: 793a7i1b The Integration starts at the cheapest tier absent evidence
  Given the Integration is installed and no tier evidence was delivered for a hard-looking work-kind
  When the orchestrator routes a unit of that work-kind
  Then the unit starts at the cheapest tier
  And the transcript shows the orchestrator ran no recall

Scenario: 793a7i1c Without the Integration the orchestrator recalls the evidence
  Given the Integration is not installed and an aggregate showing that cheap failed for a work-kind
  When the orchestrator routes a unit of that work-kind
  Then the transcript shows the orchestrator recalled before dispatching and started the unit at the tier the evidence supports
```

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

Scenario: 793a7j1c Without the Integration recall first is in every handoff
  Given the Integration is not installed and a subagent that already has a detailed prompt
  When the orchestrator dispatches it
  Then the handoff still tells it to recall first
```

### 793a7k Modified: The loop improves the rubric

Target: [792a17o The loop improves the rubric](../../specs/product/route.md#792a17o-the-loop-improves-the-rubric)

Requirement: 792a17o While the Integration is installed, when a finding about a work-kind is strong and general (for example, "cheap failed N times on work-kind K, so it needs mid"), the orchestrator MUST NOT crystallize it with learn; the System captures it as a confirmed approach when a tier's outcome confirms the routing (for example, "cheap sufficed for K") or as a reversal when it overturns a prior tier assumption. The next recall MUST surface that evidence so that the starting tier for similar work reflects it. Where the Integration is not installed: When a finding about a work-kind is strong and general (for example, "cheap failed N times on work-kind K, so it needs mid"), the orchestrator MUST crystallize it with learn, as a confirmed approach when a tier's outcome confirms the routing (for example, "cheap sufficed for K") or as a reversal when it overturns a prior tier assumption. The next recall MUST surface that evidence so that the starting tier for similar work reflects it.

Rationale: The finding is a lesson, and the System captures lessons.

#### 793a7k1 UAT Scenarios

```gherkin
Scenario: 793a7k1a The Integration captures a confirming tier outcome
  Given the Integration is installed and cheap sufficed repeatedly for a work-kind
  When the confirming outcome lands
  Then a note stating the confirmed approach exists in the vault
  And the transcript shows the orchestrator invoked no learn command

Scenario: 793a7k1b Without the Integration the orchestrator crystallizes a reversal
  Given the Integration is not installed and evidence that overturns a prior tier assumption
  When the orchestrator crystallizes the finding
  Then the transcript shows it invoked learn and recorded a reversal
```

### 793a7l Modified: The seven steps are tracked, ordered and non-waivable

Target: [792a18c The seven steps are tracked, ordered and non-waivable](../../specs/product/please.md#792a18c-the-seven-steps-are-tracked-ordered-and-non-waivable)

Requirement: 792a18c While the Integration is installed, the workflow's first step (the opening learn) and seventh step (the closing learn) are performed by the System, not by please. At the start of execution, please MUST push steps 2 to 6 to the task list in a single `TaskCreate` call, keeping their numbers, mark each step in progress when it begins and completed when it finishes, and start each step only after the previous step is completed. A step MUST be marked not applicable only when its mechanism does not exist in the environment (for example step 6's commit when the repository is not under version control), with a one-line rationale naming the missing mechanism written into the task description. "Too small to bother", "nothing to document" and "no memories will be relevant" MUST NOT count. The user MUST NOT be able to waive steps by saying "no ceremony", "skip the plan" or "hurry". When the user interrupts and redirects, please MUST update the task list to the new shape and MUST NOT abandon open steps without an explicit close. Where the Integration is not installed: At the start of execution, please MUST push all seven steps to the task list in a single `TaskCreate` call, mark each step in progress when it begins and completed when it finishes, and start each step only after the previous step is completed. A step MUST be marked not applicable only when its mechanism does not exist in the environment (for example step 6's commit when the repository is not under version control, or step 7's closing learn when the engram binary is absent), with a one-line rationale naming the missing mechanism written into the task description. "Too small to bother", "nothing to document" and "no memories will be relevant" MUST NOT count. The user MUST NOT be able to waive steps by saying "no ceremony", "skip the plan" or "hurry". When the user interrupts and redirects, please MUST update the task list to the new shape and MUST NOT abandon open steps without an explicit close.

Rationale: Steps 1 and 7 are learn and nothing else, so the System performing it leaves five tracked steps. The shape is [793c37 What is the shape of please's workflow while the Integration is installed?](793c_open-questions.md#793c37-what-is-the-shape-of-pleases-workflow-while-the-integration-is-installed).

#### 793a7l1 UAT Scenarios

```gherkin
Scenario: 793a7l1a The Integration leaves five tracked steps
  Given the Integration is installed and a valid ask
  When please begins execution
  Then the task list gets steps 2 to 6 from one TaskCreate call and each moves in progress and then completed in order
  And the transcript shows please ran no learn

Scenario: 793a7l1b The Integration still does not let the user waive steps
  Given the Integration is installed and the user says "no ceremony, skip the plan"
  When the agent proceeds
  Then it runs the remaining steps, including the plan

Scenario: 793a7l1c Without the Integration the task list holds seven steps
  Given the Integration is not installed and a valid ask
  When please begins execution
  Then the task list gets seven tasks from one TaskCreate call and each moves in progress and then completed in order
```

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

Scenario: 793a7m1c Without the Integration please opens with learn and recalls literally
  Given the Integration is not installed and a small ask whose diff is right there
  When please starts and orients
  Then the first thing it runs is the learn skill
  And it invokes recall and also reads files, and does not read files instead of recalling
```

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

Scenario: 793a7n1c Without the Integration a reviewer recalls first
  Given the Integration is not installed and a dispatched reviewer
  When it starts
  Then its first action is to recall with phrases from the artifact and its angle
```

### 793a7o Modified: The closing learn is preceded by a lessons audit

Target: [792a18p The closing learn is preceded by a lessons audit](../../specs/product/please.md#792a18p-the-closing-learn-is-preceded-by-a-lessons-audit)

Requirement: 792a18p While the Integration is installed, after step 6 and before its closing report, please MUST audit the cycle's mechanical corpus: every pre-registered stop that fired, every gate FAIL verdict, every commit whose message contains CORRECTION, supersede or superseded, instrument-invalid, or redraw or redrawn, and every mid-cycle escalation to the user. It MUST map each item to the vault note that captures its lesson, or write the one-line "no lesson: <why>"; MUST list an unmapped item as unmapped, and MUST NOT hand it to a learn; and MUST show the audit list (item → note or no-lesson) in the cycle's closing report to the user. It MUST NOT use the audit for positive reinforcement, which has no mechanical marker and belongs to the System's scan for a confirmed approach ([792a16g A confirmed approach is looked for at the moment of confirmation](../../specs/product/lessons-contract.md#792a16g-a-confirmed-approach-is-looked-for-at-the-moment-of-confirmation)). Where the Integration is not installed: Before the closing learn at step 7, please MUST audit the cycle's mechanical corpus: every pre-registered stop that fired, every gate FAIL verdict, every commit whose message contains CORRECTION, supersede or superseded, instrument-invalid, or redraw or redrawn, and every mid-cycle escalation to the user. It MUST map each item to the vault note that captures its lesson, or write the one-line "no lesson: <why>"; MUST treat an unmapped item as a reversal handoff for the closing learn ([792a15e The four kinds of explicit lesson](../../specs/product/learn.md#792a15e-the-four-kinds-of-explicit-lesson)); and MUST show the audit list (item → note or no-lesson) in the cycle's closing report to the user. It MUST NOT use the audit for positive reinforcement, which has no mechanical marker and belongs to the closing learn's confirmed-approach scan.

Rationale: There is no closing learn to hand an unmapped item to, so the audit reports it. What the audit does with an unmapped item is [793c37 What is the shape of please's workflow while the Integration is installed?](793c_open-questions.md#793c37-what-is-the-shape-of-pleases-workflow-while-the-integration-is-installed).

#### 793a7o1 UAT Scenarios

```gherkin
Scenario: 793a7o1a The Integration's audit enumerates the corpus and maps each item
  Given the Integration is installed and a cycle with one fired stop, one gate FAIL, one commit named CORRECTION and one escalation
  When please audits before its closing report
  Then all four items are on the list
  And each names the vault note that captures its lesson or says "no lesson: <why>"

Scenario: 793a7o1b The Integration's audit hands nothing to a learn
  Given the Integration is installed and an audit item with no vault note
  When please reports to the user
  Then the report lists the item as unmapped
  And the transcript shows please invoked no learn command

Scenario: 793a7o1c Without the Integration an unmapped item becomes a reversal handoff
  Given the Integration is not installed and an audit item with no vault note
  When the closing learn runs
  Then the item is handed to learn as a reversal
```

### 793a7p Modified: A lesson about to be captured asks what should have surfaced it

Target: [792a18q A lesson about to be captured asks what should have surfaced it](../../specs/product/please.md#792a18q-a-lesson-about-to-be-captured-asks-what-should-have-surfaced-it)

Requirement: 792a18q While the Integration is installed, please captures no lesson, so it MUST NOT ask which existing artifact should have surfaced a lesson, MUST NOT search the vault for a covering note before writing one, and MUST NOT reword a note whose situation line does not match; the System captures lessons under learn's own rules. Please MUST NOT file automatically a change that ships from the repository (code, skills, guidance). Where the Integration is not installed: For each lesson the cycle is about to capture, please MUST ask which artifact that already existed should have surfaced it first, MUST search the vault for a note that already covers the situation before writing a new one, and MUST, when a note existed but did not surface at the moment it was needed, check whether its `situation:` line matches how the moment actually presented and reword the note when it does not, in preference to writing a second note on the same lesson. It MUST propose a change that ships from the repository (code, skills, guidance) only as a last resort, and MUST NOT file it automatically.

Rationale: The question, the search and the reword belong to capturing a lesson, which please no longer does.

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

Scenario: 793a7p1c Without the Integration an existing note is searched for first
  Given the Integration is not installed and a lesson about to be captured
  When please prepares the capture
  Then it asks which existing artifact should have surfaced it and searches the vault for a note covering the situation
```
