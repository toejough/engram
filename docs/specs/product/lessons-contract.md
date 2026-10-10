# Lessons Contract - Product Spec

When an agent hands work to a subagent, a lesson that the subagent learned but never reported cannot be captured. The lessons contract is the agreement between the orchestrating agent and every subagent it dispatches. Each subagent's completion report ends with a `LESSONS:` line. The orchestrator re-asks once if the line is missing, records every line as it returns, judges each lesson right then, and gives the whole collection to the closing learn as a backstop. The line is a deliberately low-bar offer and not a vault note, so a subagent never decides on its own what enters the vault. The skills that carry the contract are route (the handoff, [route](./route.md)), please (the collection and the hand-over at its close, [please](./please.md)) and learn (the judgment, [learn](./learn.md)); the ambient cue that fires learn at a return is ambient-guidance's ([ambient-guidance](./ambient-guidance.md)). This capability states the contract once so that no skill repeats it.

Rationale: most failures that memory would have helped with trace to lessons never captured, and most of the lost
lessons sit in dispatched work. A subagent running learn itself does not work: route already tells subagents to run
recall first and that is not reliably followed, a subagent's judgment of its own work confabulates, and several
dispatch targets have no Skill tool at all. So the subagent supplies raw content and the orchestrator judges it. The
mechanism is skill and guidance markdown only, with a low bar at capture and quality control at a separate judgment
step; a mandatory line in prose raises the capture rate and guarantees nothing.

## 792a16a A completion report ends with a LESSONS line

Requirement: [792a16a A completion report ends with a LESSONS line](#792a16a-a-completion-report-ends-with-a-lessons-line) Every subagent dispatched through route, whether a fork, a fresh-context agent or a workflow, MUST end its final report with a `LESSONS:` line that is `LESSONS: none`, or one to three one-line lessons separated by commas. Each lesson MUST be a confirmed correction, a surprising finding or a validated approach, summarized in one line. A subagent MUST NOT decide for itself that a lesson becomes a vault note.

Rationale: the line is a disposable offer tier, intentionally a low bar. The orchestrator judges the offers; a subagent writing notes itself would put quality control at the wrong place.

### 792a16a1 UAT Scenarios

```gherkin
Scenario: 792a16a1a A subagent with lessons reports them
  Given a subagent whose work contained a confirmed correction and a surprising finding
  When it writes its completion report
  Then the report ends with LESSONS: <lesson1>, <lesson2>

Scenario: 792a16a1b A subagent with nothing to report says so
  Given a subagent whose work was straightforward execution with no novel finding
  When it writes its completion report
  Then the report ends with LESSONS: none

Scenario: 792a16a1c At most three lessons
  Given a subagent that learned five things
  When it writes the line
  Then the line carries at most three one-line lessons
```

## 792a16b Every dispatch is told to report lessons

Requirement: [792a16b Every dispatch is told to report lessons](#792a16b-every-dispatch-is-told-to-report-lessons) The handoff that route builds for every dispatch MUST include, among the items it must hand over (exact files and paths, acceptance checks, do-not-touch bounds and the subagent's recall-first instruction), the completion-report contract: the subagent's final report MUST end with a `LESSONS:` line in the format of [792a16a A completion report ends with a LESSONS line](#792a16a-a-completion-report-ends-with-a-lessons-line). The instruction MUST be prescriptive, and MUST be written out even for a fork, which inherits session context.

### 792a16b1 UAT Scenarios

```gherkin
Scenario: 792a16b1a A fresh-context dispatch carries the mandate
  Given route selects a general-purpose agent and builds its instructions
  When the handoff is written
  Then it includes the completion-report contract with the LESSONS line

Scenario: 792a16b1b A fork's handoff carries it too
  Given route dispatches a fork
  When the handoff is written
  Then the fork's instructions state the LESSONS mandate in writing

Scenario: 792a16b1c It follows the task description
  Given a complex multi-step task
  When route writes the handoff
  Then the task and its acceptance checks are followed by the completion-report contract
```

## 792a16c A missing line is asked for once

Requirement: [792a16c A missing line is asked for once](#792a16c-a-missing-line-is-asked-for-once) When a returned report does not end with a `LESSONS:` line, the orchestrator MUST send exactly one targeted follow-up that asks only for the line, and MUST NOT re-ask a second time. A line supplied in response MUST be treated like an on-time line. When the line is still missing after the one follow-up, the orchestrator MUST record `LESSONS: none` and proceed without blocking.

### 792a16c1 UAT Scenarios

```gherkin
Scenario: 792a16c1a A missing line is re-asked
  Given a subagent report that ends without a LESSONS line
  When the orchestrator receives it
  Then it sends one follow-up asking only for the line in the format LESSONS: <lesson1>, <lesson2> or LESSONS: none

Scenario: 792a16c1b The answer counts like an on-time line
  Given a subagent that supplies the line in response to the follow-up
  When the orchestrator collects it
  Then it is recorded and judged the same as an on-time line

Scenario: 792a16c1c There is no second re-ask
  Given a subagent that still gives no LESSONS line after the follow-up
  When the orchestrator proceeds
  Then it records LESSONS: none for that dispatch and does not ask again
```

## 792a16d Every line is recorded as it returns

Requirement: [792a16d Every line is recorded as it returns](#792a16d-every-line-is-recorded-as-it-returns) The orchestrator following please MUST record the `LESSONS:` line of every dispatched unit verbatim, including `LESSONS: none`, in a running list for the cycle at the moment each unit's result comes back and before it moves to the next unit, wherever in the workflow the dispatch lands. Judging a line MUST NOT remove or rewrite it, and the list MUST NOT carry marks of what was captured. The please gates MUST NOT block or filter the collection: a line is recorded even when its work is rejected at a gate.

### 792a16d1 UAT Scenarios

```gherkin
Scenario: 792a16d1a Lines are kept in order and verbatim
  Given please dispatches three units in sequence
  When each returns
  Then the running list holds the three LESSONS lines in order and exactly as written

Scenario: 792a16d1b A none is still recorded
  Given a unit reports LESSONS: none
  When the orchestrator records it
  Then the line is an entry in the list and no learn fires for it

Scenario: 792a16d1c A gate does not filter the list
  Given a gate rejects a unit's work
  When the unit's LESSONS line has returned
  Then the line is in the list
```

## 792a16e Each lesson is judged when it returns

Requirement: [792a16e Each lesson is judged when it returns](#792a16e-each-lesson-is-judged-when-it-returns) When a returned report's `LESSONS:` line is not `none`, the orchestrator MUST judge each lesson in it at that moment, and MUST capture each lesson that clears learn's bar with a fast-path learn (one note per lesson, no sweep, handed to write-memory) before the next dispatch, or before ending its turn when no dispatch is left. A lesson clears the bar when it is one of learn's four kinds of lesson ([792a15e The four kinds of explicit lesson](./learn.md#792a15e-the-four-kinds-of-explicit-lesson)), is confirmed and not hypothesized, and states a general reusable principle and not a session-specific narrative. The orchestrator MUST NOT capture `none`, a routine "done, tests pass" or an unconfirmed lesson.

Rationale: waiting for the closing learn does not work in practice: the capture that does happen comes from writes the
parent makes in the middle of the session. Judging at the return, while the lesson is in hand, uses that tendency, and
the closing learn stays as a backstop. The rule lives in always-loaded guidance because it must hold at every
delegation whichever skill mediated it.

### 792a16e1 UAT Scenarios

```gherkin
Scenario: 792a16e1a A confirmed correction is captured at the return
  Given a unit returns LESSONS: review rejected bare error returns; the convention is fmt.Errorf with %w
  When the orchestrator reads the report
  Then it captures one note with a fast-path learn before the next dispatch

Scenario: 792a16e1b None and bare success are left alone
  Given a unit returns LESSONS: none, or "we did the work"
  When the orchestrator reads the report
  Then no note is written for it

Scenario: 792a16e1c An unconfirmed lesson is left alone
  Given a unit returns LESSONS: might be a race in the cache
  When the orchestrator judges it
  Then it writes no note

Scenario: 792a16e1d The last return is captured before the turn ends
  Given the last dispatched unit returns a lesson that clears the bar
  When there is no next dispatch
  Then the orchestrator captures it before ending its turn
```

## 792a16f The closing learn gets the whole list

Requirement: [792a16f The closing learn gets the whole list](#792a16f-the-closing-learn-gets-the-whole-list) At workflow step 7, please MUST hand the full list of `LESSONS:` lines collected across its dispatches, `none` entries included, to the closing learn, alongside the lessons-audit map ([please](./please.md)), and MUST drop no line. Please MUST run the closing learn when the session had no dispatches, handing it an empty collection. The closing learn MUST skip a line that a vault note written earlier this session already covers, MUST judge every other line against the bar of [792a16e Each lesson is judged when it returns](#792a16e-each-lesson-is-judged-when-it-returns), and MUST discard a line that does not clear it silently, with no note and no log. It MUST NOT write a second time a line captured at its return.

Rationale: a line is a low-bar offer and not a pre-judged lesson, so discarding it is not a loss. Skipping is by vault
coverage and not by a mark, because the list stays verbatim and unmarked.

### 792a16f1 UAT Scenarios

```gherkin
Scenario: 792a16f1a The closing learn receives the list
  Given a session with three dispatches
  When please reaches step 7
  Then the closing learn is given all three LESSONS lines, whatever they say

Scenario: 792a16f1b No dispatch, empty collection
  Given a session that answered a question directly with no dispatch
  When please reaches step 7
  Then the closing learn runs with an empty collection and takes no action on it

Scenario: 792a16f1c A line captured at its return is skipped
  Given a session that collected three lines, one already captured by a fast-path learn at its return
  When the closing learn runs
  Then it does not write that line again and judges only the other two

Scenario: 792a16f1d A missed line is caught
  Given a line that clears the bar and was not captured at its return
  When the closing learn runs
  Then it crystallizes the line

Scenario: 792a16f1e A mixed batch
  Given four uncaptured lines: a kind-1 correction, a kind-4 confirmed approach, a bare success and a generic reminder to test
  When the closing learn runs
  Then it writes the correction and the confirmed approach and discards the other two silently
```

## 792a16g A confirmed approach is looked for at the moment of confirmation

Requirement: [792a16g A confirmed approach is looked for at the moment of confirmation](#792a16g-a-confirmed-approach-is-looked-for-at-the-moment-of-confirmation) Learn MUST scan for a confirmed approach (kind 4) at the moment a unit of work's outcome is confirmed: a review or gate verdict lands, a check or test passes, or the user explicitly confirms something. That includes the moment the agent reads a dispatched subagent's completion report and its `LESSONS:` line once its outcome is known, and, at the closing learn, the session's completion reports and collected lines. The bar MUST NOT move: a resolved uncertainty or an explicit specific confirmation, never bare success.

Rationale: the kind-4 scan is the one that under-fires most, and missed success-reinforcements are the largest lost
bucket. Anchoring on the concrete event and calibrating with exemplars (a design assumption checked by a targeted
search, a defect confirmed by a reproduction test, a vendor claim verified against evidence) improves how recognizable
the pattern is, not how low the bar is.

### 792a16g1 UAT Scenarios

```gherkin
Scenario: 792a16g1a A completion anchors the scan
  Given the session ends with the agent delivering a specific decision, validated the pattern for use in a given place, and a LESSONS line records it
  When the closing learn scans for kind 4
  Then it anchors on that completion moment and recognizes a learnable success

Scenario: 792a16g1b An exemplar-shaped line is recognized
  Given a subagent's line reads "targeted reproduction test confirmed the suspected defect"
  When the closing learn judges it
  Then it recognizes the defect-confirmed-by-reproduction-test shape as kind 4 and crystallizes it

Scenario: 792a16g1c A bare success is still not kind 4
  Given a line that reads "the task is done"
  When the closing learn judges it
  Then it does not classify it as kind 4
```
