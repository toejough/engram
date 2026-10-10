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
  or where a lesson may have occurred (a completed turn).
- **Lesson:** a confirmed correction, explicit save-request, overturned conclusion, or confirmed
  approach, per the learn skill's definition.
- **Injection:** placing note content in front of the working agent without the agent requesting it.
- **Operator:** the human who configures and audits the System.

Numeric thresholds marked **[proposed]** are suggested starting values that await a decision in
[`793c_open-questions.md`](./793c_open-questions.md). Requirement keywords follow RFC 2119.

Sections `793a2` through `793a5` add requirements to one new capability. Each requirement keeps its
draft identifier in a "formerly" note so that review comments on the draft still map.

## 793a2 Added: automatic-memory (automatic recall)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a2a Recall at a human prompt

Requirement: 793a2a (formerly REC-1) When a human submits a prompt, the System MUST make relevant
vault notes available to the working agent before it begins acting on that prompt, without the agent
requesting them.

#### 793a2a1 UAT Scenarios

```gherkin
Scenario: 793a2a1a Relevant note exists for the prompt
  Given a vault containing a note whose situation matches "renaming a Go package"
  And a session whose configuration contains no memory instructions
  When the human submits "rename package foo to bar across the repo"
  Then the working agent's first model call includes that note's content
  And the agent did not run any memory command to obtain it

Scenario: 793a2a1b No relevant note exists
  Given a vault with no note above the relevance threshold for the prompt
  When the human submits the prompt
  Then nothing is injected for this prompt
```

### 793a2b Recall after each tool result

Requirement: 793a2b (formerly REC-2) After a tool result is returned to the working agent, and before
the agent's next model call, the System MUST make relevant notes available based on the current
situation (the tool, its arguments, and its outcome).

#### 793a2b1 UAT Scenarios

```gherkin
Scenario: 793a2b1a Tool failure matches a known lesson
  Given a note whose situation matches "go build fails with an import cycle"
  When the working agent runs a command that fails with an import-cycle error
  Then the agent's next model call includes that note

Scenario: 793a2b1b Every post-tool model call is covered
  Given a turn in which the working agent makes 5 tool calls
  When the System is audited for that turn
  Then each of the 5 following model calls has a recorded injection decision (injected or none)
```

### 793a2c Recall at subagent dispatch

Requirement: 793a2c (formerly REC-3) When the working agent dispatches a subagent, the System MUST
make notes relevant to the subagent's task available to that subagent.

#### 793a2c1 UAT Scenarios

```gherkin
Scenario: 793a2c1a Subagent receives relevant notes
  Given a note matching "writing Go tests in this repo"
  When the working agent dispatches a subagent with the task "add tests for internal/cli"
  Then the subagent's task context includes that note
  And the working agent's own dispatch text was not required to mention it
```

### 793a2d Informational, attributable presentation

Requirement: 793a2d (formerly REC-4) Injected notes MUST identify their source note and MUST be
presented as reference information. They MUST NOT require the working agent to run commands, and MUST
NOT be phrased as instructions the agent must obey.

#### 793a2d1 UAT Scenarios

```gherkin
Scenario: 793a2d1a Injected content is attributable and non-imperative
  Given an injection of note "474.2026-07-25.charge-one-reviewer-to-execute..."
  When the injected text is inspected
  Then it names that note as its source
  And it contains no instruction to execute a command
  And a bare-configuration headless run containing it is not refused or flagged as an injection attempt by the agent in at least 95% of N runs [proposed N=20]
```

### 793a2e Relevance and non-repetition

Requirement: 793a2e (formerly REC-5) The System MUST NOT inject a note below the relevance threshold.
The System SHOULD NOT inject the same note again in the same session unless the situation has
materially changed.

#### 793a2e1 UAT Scenarios

```gherkin
Scenario: 793a2e1a Same note not repeated
  Given note X was injected at the prompt
  When the working agent's next two tool results match X again with no change in situation
  Then X is not injected again

Scenario: 793a2e1b Irrelevant note suppressed
  Given the best candidate note scores below the relevance threshold
  When a moment occurs
  Then no injection occurs
```

### 793a2f Bounded volume

Requirement: 793a2f (formerly REC-6) Injections MUST be bounded per injection, per turn, and per
session, so that recall cannot materially crowd out the task. Initial bounds **[proposed]**: at a
prompt, at most 3 notes / 6 KB; at a tool result, at most 2 notes / 3 KB; per turn, at most 4
tool-result injections and 12 KB.

#### 793a2f1 UAT Scenarios

```gherkin
Scenario: 793a2f1a Per-turn bound holds
  Given a turn with 10 tool results that each match several notes
  When the turn completes
  Then total injected content for the turn is at most 12 KB
  And no more than 4 tool-result injections occurred

Scenario: 793a2f1b Prompt-time bound holds
  Given a prompt that matches more than 3 notes totaling more than 6 KB
  When the human submits the prompt
  Then at most 3 notes and 6 KB are injected for that prompt

Scenario: 793a2f1c Tool-result bound holds
  Given a tool result that matches more than 2 notes totaling more than 3 KB
  When the tool result is returned to the working agent
  Then at most 2 notes and 3 KB are injected for that tool result
```

### 793a2g Human-originated moments only

Requirement: 793a2g (formerly REC-7) The System SHOULD NOT inject recall for prompts that did not
originate from the human (automated notifications, scheduled triggers, messages from other sessions)
unless the operator configures otherwise.

#### 793a2g1 UAT Scenarios

```gherkin
Scenario: 793a2g1a Automated prompt skipped
  Given an automated task notification arrives as a prompt
  When the System evaluates it
  Then no prompt-time injection occurs
  And the skip is recorded
```

## 793a3 Added: automatic-memory (automatic capture)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a3a Evaluate every completed turn

Requirement: 793a3a (formerly CAP-1) After each completed turn, the System MUST evaluate the turn for
lessons (correction, explicit save-request, reversal, confirmed approach) without the working agent
requesting it.

#### 793a3a1 UAT Scenarios

```gherkin
Scenario: 793a3a1a User correction is captured without agent action
  Given a session whose configuration contains no memory instructions
  When the user says "no, use the targ build system, not go test directly"
  And the turn completes
  Then a feedback note stating that rule exists in the vault or in the dry-run log
  And the working agent did not invoke any learn command

Scenario: 793a3a1b Explicit save-request
  When the user says "remember that releases are tagged from main"
  And the turn completes
  Then a fact note stating that is captured
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
```

### 793a3c No duplicates

Requirement: 793a3c (formerly CAP-3) Before writing, the System MUST check for an existing note
stating the same lesson and MUST NOT write a duplicate. When an equivalent note exists, the System
SHOULD reinforce it instead.

#### 793a3c1 UAT Scenarios

```gherkin
Scenario: 793a3c1a Existing note already covers the lesson
  Given a note stating "never amend pushed commits"
  When the user repeats that correction and the turn completes
  Then no new note is created
  And the existing note is marked as reinforced
```

### 793a3d Provenance

Requirement: 793a3d (formerly CAP-4) Every automatically captured note MUST record that it was
captured automatically, and the session and turn it came from.

#### 793a3d1 UAT Scenarios

```gherkin
Scenario: 793a3d1a Provenance present
  Given an automatically captured note
  When its source field is inspected
  Then it identifies automatic capture, the session id, and the turn id
```

### 793a3e Capture quality

Requirement: 793a3e (formerly CAP-5) Of automatically captured notes, at least **[proposed] 70%**
MUST be judged valid lessons by the operator on review. The System SHOULD capture materially more of
the lesson-worthy moments than the current baseline of about 5%; target **[proposed] at least 50%**.

#### 793a3e1 UAT Scenarios

```gherkin
Scenario: 793a3e1a Precision measured on a sample
  Given 30 automatically captured notes sampled from real sessions
  When the operator reviews each as valid or not valid
  Then at least 21 are valid

Scenario: 793a3e1b Recall rate measured against labeled moments
  Given a labeled set of at least 60 lesson-worthy moments from real sessions
  When the System runs over those sessions
  Then it captures at least 50% of them
```

### 793a3f Non-blocking capture

Requirement: 793a3f (formerly CAP-6) Capture MUST NOT delay the working agent's response or the
user's next prompt beyond the latency budget in 793a5d (formerly NFR-1).

#### 793a3f1 UAT Scenarios

```gherkin
Scenario: 793a3f1a Slow capture does not stall the session
  Given capture evaluation takes 20 seconds
  When a turn completes and the user submits the next prompt immediately
  Then the next prompt is processed without waiting for the capture to finish
```

## 793a4 Added: automatic-memory (operator control and safety)

Canonical target: `docs/specs/product/automatic-memory.md` (title `Automatic Memory - Product Spec`).

### 793a4a Operating modes

Requirement: 793a4a (formerly CTL-1) The operator MUST be able to select among: off (no recall, no
capture), observe (decisions are logged, nothing is injected or written), and active (injection and
writes occur). The initial default for a new install SHOULD be observe.

#### 793a4a1 UAT Scenarios

```gherkin
Scenario: 793a4a1a Observe mode has no side effects
  Given mode is observe
  When a session runs with prompts that match notes and contain a correction
  Then no content is injected into the working agent
  And no note is written to the vault
  And a log records what would have been injected and written

Scenario: 793a4a1b Off mode is inert
  Given mode is off
  When a session runs
  Then the System performs no recall, no capture, and writes no log entries
```

### 793a4b Failure isolation

Requirement: 793a4b (formerly CTL-2) If the System fails, times out, or is unavailable, the working
session MUST continue unaffected, with no error surfaced to the working agent and no lost user input.

#### 793a4b1 UAT Scenarios

```gherkin
Scenario: 793a4b1a Vault unavailable
  Given the vault cannot be read
  When the human submits a prompt
  Then the session proceeds normally with no injection
  And the failure is logged for the operator

Scenario: 793a4b1b Evaluation crashes
  Given capture evaluation raises an error
  When a turn completes
  Then the session continues and the next prompt is processed normally
```

### 793a4c Memory purview only

Requirement: 793a4c (formerly CTL-3) The System MUST NOT alter task decisions, tool permissions, tool
arguments, model choice, or the user's prompt text. Its only effects are injecting reference notes and
writing vault notes.

#### 793a4c1 UAT Scenarios

```gherkin
Scenario: 793a4c1a Task behavior unchanged
  Given the same prompt run with the System off and in active mode
  Then tool permissions, tool arguments, and the visible user prompt are identical in both runs
  And the only difference is the presence of injected reference notes
```

### 793a4d Sensitive content

Requirement: 793a4d (formerly CTL-4) The System SHOULD NOT write credentials, tokens, or secrets
present in a transcript into a note.

#### 793a4d1 UAT Scenarios

```gherkin
Scenario: 793a4d1a Secret in correction is not persisted
  Given the user's correction text includes an API key
  When the lesson is captured
  Then the written note contains the rule but not the key
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

Requirement: 793a5b (formerly IND-2) Where the existing instruction-based path is also active, the
System MUST NOT cause the same note to be delivered twice for the same moment, and the working agent
SHOULD NOT perform redundant lookups or saves for a moment the System already handled.

#### 793a5b1 UAT Scenarios

```gherkin
Scenario: 793a5b1a Both paths active
  Given the prompt-based memory instructions and the System are both active
  When a moment occurs that both would handle
  Then the note is delivered once
  And at most one note is written for the lesson
```

### 793a5c Environments without the capability

Requirement: 793a5c (formerly IND-3) In harnesses where the System cannot run, the existing
instruction-based path MUST continue to work unchanged.

#### 793a5c1 UAT Scenarios

```gherkin
Scenario: 793a5c1a Unsupported harness
  Given a harness without support for the System
  When a session runs with the existing instruction-based path
  Then its behavior is identical to today
```

### 793a5d Latency

Requirement: 793a5d (formerly NFR-1) The System MUST add at most **[proposed] 1.5 s** median added
latency per human prompt, and MUST NOT add blocking latency for capture (793a3f, formerly CAP-6).

#### 793a5d1 UAT Scenarios

```gherkin
Scenario: 793a5d1a Latency measured
  Given 50 prompts run with the System off and 50 in active mode
  When median time-to-first-model-call is compared
  Then the active median is at most 1.5 s higher
```

### 793a5e Auditability and measurability

Requirement: 793a5e (formerly NFR-2) Every recall decision (injected or none, with the notes and
scores) and every capture decision (written, reinforced, or discarded, with reason) MUST be logged
with session and turn identifiers, so effect on behavior can be measured and the operator can audit
it.

#### 793a5e1 UAT Scenarios

```gherkin
Scenario: 793a5e1a Decision log joins to outcomes
  Given a completed session
  When the operator queries the log by session id
  Then each moment has a recorded decision
  And each injected note can be matched to the later actions of the working agent
```

### 793a5f Effect on outcomes

Requirement: 793a5f (formerly NFR-3) In a controlled comparison on lesson-bearing tasks, runs with
the System active SHOULD apply a relevant existing lesson more often than runs without it.

#### 793a5f1 UAT Scenarios

```gherkin
Scenario: 793a5f1a Treatment beats control
  Given a set of tasks each with a relevant existing note
  And N runs per arm [proposed N=20] with the System off and active, fresh process each
  When the rate of applying the relevant note is compared
  Then the active arm's rate is higher by more than the noise floor measured between repeat runs of the same arm
```
