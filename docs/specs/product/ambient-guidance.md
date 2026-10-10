# Ambient Guidance - Product Spec

Ambient guidance is the standing instruction text that an engram user imports into the config file of their agent
harness, so that every session carries it without anyone invoking a skill. Four documents ship in
`agent-instructions/guidance/`: `recall.md` (when to recall), `learn.md` (when to capture a lesson), `delegate.md`
(when to hand work to a subagent) and `shim.md` (how to find and follow a runbook note). Each tells the agent when
to act and points at the skill or note that says how, without restating it. How the documents reach a harness is
install-and-update's ([792a11l Guidance is managed only after opt-in](./install-and-update.md#792a11l-guidance-is-managed-only-after-opt-in), [792a11m Claude import paths keep working](./install-and-update.md#792a11m-claude-import-paths-keep-working)); this capability is what they require of the agent and what the update
report tells the user about activating them. The skills they point at are [recall](./recall.md) (recall), [learn](./learn.md) (learn), [route](./route.md)
(route) and [please](./please.md) (please).

## 792a19a Recall fires at four decision moments

Requirement: [792a19a Recall fires at four decision moments](#792a19a-recall-fires-at-four-decision-moments) An agent that has the recall guidance MUST run `/recall glance` before it proceeds at each of
four moments: before it endorses or ranks a proposed approach, even when only discussing it; before it declares
work done; once, before it starts guessing, after a failure it cannot immediately explain, and not on every retry;
and before it starts building a new approach.

Rationale: recall at the start of a task cannot surface what only matters later. A lesson the vault already holds is
most valuable right before a wrong call locks in, and shipping past one costs far more than a quick glance.

### 792a19a1 UAT Scenarios

```gherkin
Scenario: 792a19a1a A proposed approach is recalled before it is endorsed
  Given the user proposes a technical approach and asks the agent what it thinks
  When the agent is about to evaluate or rank it
  Then it runs /recall glance first, with phrases drawn from the approach, before offering any analysis

Scenario: 792a19a1b Work is recalled before it is declared done
  Given the agent has finished a task, unit or phase
  When it is about to declare it done
  Then it runs /recall glance before it verifies and closes

Scenario: 792a19a1c An unexplained failure is recalled once
  Given a failure the agent cannot immediately explain, and then a retry that fails the same way
  When the agent meets the failure
  Then it runs /recall glance once before it guesses at causes, and does not run it again for the retry

Scenario: 792a19a1d A new approach is recalled before it is built
  Given the agent is about to begin implementing a new approach
  When it has chosen the approach
  Then it runs /recall glance for prior decisions and standards before it starts building
```

## 792a19b Recall escalates from glance to deep

Requirement: [792a19b Recall escalates from glance to deep](#792a19b-recall-escalates-from-glance-to-deep) An agent that has the recall guidance MUST escalate from `/recall glance` to `/recall deep`
when the decision is weighty or irreversible, or when glance flags a gap it cannot resolve.

Rationale: glance is the cheap read-only rung and deep is the thorough one that also crystallizes new lessons, so
the thorough rung is reserved for decisions where a miss is expensive. The escalation on the recency channel is
recall's own rule ([792a14d Glance escalates to deep for a recent standard](./recall.md#792a14d-glance-escalates-to-deep-for-a-recent-standard)).

### 792a19b1 UAT Scenarios

```gherkin
Scenario: 792a19b1a A weighty decision escalates
  Given a decision that is irreversible
  When the agent recalls before making it
  Then it runs /recall deep and not only /recall glance

Scenario: 792a19b1b An unresolved gap escalates
  Given a glance that flags a gap the agent cannot resolve
  When the agent reads the result
  Then it escalates to /recall deep
```

## 792a19c Learn fires at the correction moment

Requirement: [792a19c Learn fires at the correction moment](#792a19c-learn-fires-at-the-correction-moment) An agent that has the learn guidance MUST invoke the `learn` skill, before it applies the fix,
when a review or the user rejects its approach, when a build, test or check fails and the non-obvious cause is now
known, and when it catches its own approach being wrong. Even a one-line fix MUST earn the note.

Rationale: the confirmed lesson is in hand for exactly one moment. The agent then applies the fix, the turn ends
and the lesson is gone, so the next session pays to rediscover it. The value is in the rule the agent was just
corrected on, which is identical whether the fix is one line or ten.

### 792a19c1 UAT Scenarios

```gherkin
Scenario: 792a19c1a A rejected approach is captured before the fix
  Given the user says "no, do X not Y, the convention here is Z"
  When the agent is about to apply the fix
  Then it invokes the learn skill for the correction first

Scenario: 792a19c1b A failing check with a known cause is captured
  Given a check failed with a cause that was not obvious and the agent now knows the cause
  When the agent has the cause
  Then it invokes the learn skill so the next session does not hit it again

Scenario: 792a19c1c A self-discovered reversal is captured
  Given the agent notices partway through that its own approach was wrong
  When it names the root cause
  Then it invokes the learn skill at that moment

Scenario: 792a19c1d A one-line fix still earns the note
  Given a correction whose fix is a single line
  When the agent applies it
  Then it has invoked the learn skill for the rule first
```

## 792a19d The capture is learn's fast path

Requirement: [792a19d The capture is learn's fast path](#792a19d-the-capture-is-learns-fast-path) When the learn guidance fires, the agent MUST capture the one confirmed lesson with the
`learn` skill, invoked as a skill and not as a plain note file or the harness's own project memory, on the
skill's mid-cycle fast path, and MUST NOT run `engram ingest --auto` mid-task. It MUST capture a confirmed negative
(what was rejected), which is enough on its own, and MUST NOT crystallize an unconfirmed hunch.

Rationale: a native memory write or a plain file is a store engram cannot recall from, so a capture there misses the
cue. The sweep is a cycle-close job; running it mid-task is wasteful, can block on a large corpus, and loses the
capture if it hangs. The rejection is known with confidence long before the full right answer, and an unconfirmed
guess in the vault rots it.

### 792a19d1 UAT Scenarios

```gherkin
Scenario: 792a19d1a The learn skill is invoked, not a native memory write
  Given a confirmed correction
  When the agent captures it
  Then the capture is a learn skill invocation that writes a vault note, and not a write to the harness's project memory or a plain file

Scenario: 792a19d1b No sweep runs mid-task
  Given a correction in the middle of a task
  When the agent captures it
  Then it crystallizes the one lesson and runs no engram ingest --auto

Scenario: 792a19d1c A rejection is captured without the right answer
  Given a review rejected a path join and the right rule is not yet known
  When the agent captures
  Then it records the rejection and does not guess at the rule

Scenario: 792a19d1d A hunch is not captured
  Given the agent suspects a cause but nothing confirms it
  When it considers capturing
  Then it writes no note
```

## 792a19e A returned subagent lesson is captured before the next dispatch

Requirement: [792a19e A returned subagent lesson is captured before the next dispatch](#792a19e-a-returned-subagent-lesson-is-captured-before-the-next-dispatch) When a dispatched subagent returns and its `LESSONS:` line is not `none`, the orchestrating agent
MUST judge each lesson at that moment against the learn skill's bar ([792a15e The four kinds of explicit lesson](./learn.md#792a15e-the-four-kinds-of-explicit-lesson)): it is one of the four kinds, it is
confirmed and not hypothesized, and it states a general reusable principle. For each lesson that clears the bar it
MUST invoke the `learn` skill on the fast path, one note per lesson and no sweep, before it dispatches the next
subagent or ends its turn. The cue MUST apply to every dispatch, whether or not the route or please skill
mediated it.

Rationale: at the closing learn a lesson is one line in a long list, if that learn runs at all, so the moment the
report is read is when the confirmed lesson is cheapest to keep. The cue lives in ambient guidance so that an ad hoc
dispatch, which no skill mediates, carries it too. The line's format and hand-over are the lessons contract
([lessons-contract](./lessons-contract.md)).

### 792a19e1 UAT Scenarios

```gherkin
Scenario: 792a19e1a A confirmed correction is captured before the next unit
  Given a subagent returns "LESSONS: the reviewer rejected local-time timestamps; the convention here is UTC ISO-8601 in front matter" and another unit remains
  When the orchestrator reads the report
  Then it invokes the learn skill for that lesson, with no engram ingest --auto, before it dispatches the next unit

Scenario: 792a19e1b A confirmed approach is captured as such
  Given a subagent returns a lesson that a targeted reproduction test confirmed a suspected defect before the fix was written
  When the orchestrator reads the report
  Then it captures it on the fast path before the next dispatch, as a confirmed approach

Scenario: 792a19e1c Two returns give two notes
  Given two parallel subagents return and each report carries one lesson that clears the bar
  When the orchestrator reads both
  Then it captures both, one note each, before its next dispatch

Scenario: 792a19e1d An ad hoc dispatch carries the cue
  Given the orchestrator dispatched a subagent directly without route or please, and the report carries a lesson that clears the bar
  When the report returns
  Then the orchestrator captures the lesson all the same

Scenario: 792a19e1e The last return is captured before the turn ends
  Given the final subagent returns a lesson that clears the bar and no dispatch remains
  When the orchestrator is about to end its turn
  Then it has captured the lesson first
```

## 792a19f A returned line that fails the bar is not captured

Requirement: [792a19f A returned line that fails the bar is not captured](#792a19f-a-returned-line-that-fails-the-bar-is-not-captured) The learn guidance MUST NOT fire for a returned `LESSONS:` line that is `none`, for a lesson
that is a bare success or routine narrative, or for a lesson that is an unconfirmed hypothesis, which are left for
the closing learn to discard under the same bar.

Rationale: capturing "done, tests pass" or "might be a race" would fill the vault with notes that are not lessons.

### 792a19f1 UAT Scenarios

```gherkin
Scenario: 792a19f1a LESSONS none is skipped
  Given a subagent returns "LESSONS: none"
  When the orchestrator reads the report
  Then it does not invoke the learn skill before the next dispatch

Scenario: 792a19f1b A routine narrative is skipped
  Given a subagent returns "LESSONS: completed the unit, all tests pass"
  When the orchestrator reads the report
  Then it does not invoke the learn skill before the next dispatch

Scenario: 792a19f1c An unconfirmed hunch is skipped
  Given a subagent returns "LESSONS: the build flake might be a race in the cache warmer" and nothing in the report confirms it
  When the orchestrator reads the report
  Then it does not invoke the learn skill before the next dispatch
```

## 792a19g Every unit of work is planned and routed first

Requirement: [792a19g Every unit of work is planned and routed first](#792a19g-every-unit-of-work-is-planned-and-routed-first) An agent that has the delegate guidance MUST draft a plan and route every unit of work to a
subagent before it takes any direct action on the work itself, whether the unit is a multi-file feature, a
two-file rename, a one-liner or a "quick look", and MUST decompose a multi-step change into units that are
dispatched separately. Its first tool call on a unit MUST NOT be a read, edit, write or shell command on the work
itself. It MUST route a dispatch with the `route` skill, which sets the tier, the handoff and the evidence.

Rationale: an agent that does the object-level work itself spends its own context on it and has no fresh reviewer.
Opening a file to orient is starting the work, and building it because the repository is empty is the same. The
routing is [route](./route.md).

### 792a19g1 UAT Scenarios

```gherkin
Scenario: 792a19g1a A multi-step change is decomposed
  Given a change across several steps
  When the agent starts
  Then it decomposes the change into units and routes each to its own subagent, and does not execute inline

Scenario: 792a19g1b A rename is still routed
  Given a request to rename a file or edit a single file
  When the agent starts
  Then it routes the unit to a subagent, because small size does not bypass delegation

Scenario: 792a19g1c A look at the files is a unit of work
  Given the agent wants to read files only to orient
  When it is about to open them
  Then it routes the look to a subagent and does not read them itself

Scenario: 792a19g1d The first tool call is a plan or a dispatch
  Given any request that is work
  When the agent makes its first tool call on it
  Then the call is a plan, a route or a subagent dispatch, or the engram query that [792a19j The first action on every request is an engram query](#792a19j-the-first-action-on-every-request-is-an-engram-query) puts ahead of them when the shim is imported, and not a read, edit, write or shell command on the work

Scenario: 792a19g1e A dispatch is routed
  Given a unit about to be dispatched
  When the agent dispatches it
  Then it has used the route skill to pick the tier and the handoff
```

## 792a19h Solo work needs a recorded measurement

Requirement: [792a19h Solo work needs a recorded measurement](#792a19h-solo-work-needs-a-recorded-measurement) An agent that has the delegate guidance MUST NOT do object-level work itself on the strength of
a forecast, such as "it's trivial", "it's just a rename", "a quick fix with the exact files known", "the overhead
would exceed the work" or "it's greenfield, so I'll build it", and MAY do it inline only when recalled memory shows
a measured record that this kind of work runs reliably below the routing overhead. Without such a record it MUST
route the unit.

Rationale: the forecast is the same each time and is the failure being guarded against. Only a record licenses the
skip, so the agent must know a quick fix is quick, not guess it. The red flag is a next tool call on the work with
no plan routed.

### 792a19h1 UAT Scenarios

```gherkin
Scenario: 792a19h1a A trivial forecast does not license solo work
  Given a one-line fix whose files are known
  When the agent feels it would be quicker to do it inline
  Then it checks recalled memory for a measured record and, finding none, routes the unit

Scenario: 792a19h1b A greenfield repository does not license solo work
  Given an empty repository
  When the agent is asked to build in it
  Then it still plans and dispatches the build

Scenario: 792a19h1c A measured record licenses inline work
  Given recalled memory holds a measured record that this kind of unit runs below the routing overhead
  When the agent weighs the unit
  Then it may do the unit inline
```

## 792a19i Returned work is reviewed fresh and reported with its evidence

Requirement: [792a19i Returned work is reviewed fresh and reported with its evidence](#792a19i-returned-work-is-reviewed-fresh-and-reported-with-its-evidence) An agent that has the delegate guidance MUST review what a subagent returns with fresh
context, and not on the strength of the builder's own claim that it is done, and MUST report the outcome with
route's evidence table, or with the please skill's gate verdicts when please mediated the work.

Rationale: reviewing what returns and reporting the outcome are the orchestrator's job; the subagent produces the
artifact.

### 792a19i1 UAT Scenarios

```gherkin
Scenario: 792a19i1a A completed unit is reviewed fresh
  Given a subagent returns a completed unit
  When the agent accepts it
  Then a review with fresh context has judged it, and the builder's own "done" was not the evidence

Scenario: 792a19i1b The report carries the route evidence
  Given the agent reports delegated work to the user
  When it writes the report
  Then the report includes route's evidence table, and the gate verdicts when please mediated the work
```

## 792a19j The first action on every request is an engram query

Requirement: [792a19j The first action on every request is an engram query](#792a19j-the-first-action-on-every-request-is-an-engram-query) An agent that has the shim MUST make its first tool call on every user request an
`engram query` that carries at least two `--phrase` values (the request in the agent's own words, and the kind of
situation it is in) and a `--text` value holding the user's message verbatim (its first 300 characters or so when
long), before it reads a file, answers or asks a clarifying question, and whether or not it has decided to
delegate. It MUST NOT gate the query on a judgment that the task is multi-step, recurring or done before. It MUST
write every `--text` and `--phrase` value as `--flag="value"`, never `--flag "value"`, and MUST NOT rewrite or
summarize the message in `--text`, whose paraphrase belongs only in `--phrase`.

Rationale: whether the task is one a runbook covers is what the query answers, so the judgment cannot be a gate in
front of it. A one-line request still gets a query, and the agent proceeds without a runbook when none matches. The
over-fire on one-shot requests is accepted: each fire costs one query with no crystallization. The command is named
literally so it is executable when no engram skill is installed. A user's message can begin with `-` (a markdown
bullet) with no control over its shape, and the space-separated form reads it as a flag and exits; the attached form
survives it ([792b32c2 Free-text flag values use the attached
form](../technical/agent-instruction-bundle.md#792b32c2-free-text-flag-values-use-the-attached-form)). The raw text,
not a paraphrase, is what matches a literal trigger ([792a7y Trigger text](./memory-query.md#792a7y-trigger-text),
[792a7za A trigger hit comes first](./memory-query.md#792a7za-a-trigger-hit-comes-first)).

### 792a19j1 UAT Scenarios

```gherkin
Scenario: 792a19j1a A shim-only session queries first
  Given an agent with the shim in its config and no engram skill installed
  When it receives a user request
  Then its first tool call is an engram query carrying the request phrase, the situation phrase and --text with the user's message, before any other tool call

Scenario: 792a19j1b A one-shot request still queries
  Given a single-step request such as a one-line answer or a single-file typo fix
  When the agent receives it
  Then it still runs the query first, and proceeds without a runbook when none matches

Scenario: 792a19j1c The message is passed unchanged
  Given the user's message "/please rename the tally CLI's --out flag to --output"
  When the agent runs its first query
  Then the --text value holds that string verbatim up to whitespace differences, and the paraphrase appears only in a --phrase

Scenario: 792a19j1d A message that begins with a dash survives
  Given the user's message begins with "-", such as "- fix the login bug"
  When the agent runs its first query
  Then --text is written --text="- fix the login bug" and the command runs instead of exiting with a flag needing an argument

Scenario: 792a19j1e Deciding to delegate does not skip the query
  Given an agent about to dispatch a subagent as its first action
  When it begins
  Then it runs the engram query first, and the decision to delegate is not an exemption
```

## 792a19k The second phrase names a kind of work-handling decision

Requirement: [792a19k The second phrase names a kind of work-handling decision](#792a19k-the-second-phrase-names-a-kind-of-work-handling-decision) The shim MUST tell the agent that the second phrase of the first query names the kind of
situation, in the wording a runbook's own `situation` uses, and not the concrete ticket, and MUST tell it to
re-check that phrase before the query runs and rewrite it as the kind of work-handling decision when it contains a
concrete noun from the ticket (a file, flag, feature, bug or field name) or restates the request. The shim MUST
illustrate the shape with at least two examples of different kinds of work, and the re-check and the examples
MUST NOT name a specific runbook, skill or evaluation task.

Rationale: casual task wording alone does not reliably surface a process runbook, whereas situation-shaped wording
does, and a second phrase that restates the ticket reliably fails to. The re-check is generic so that it neither
overfits to one runbook nor regresses another's retrieval.

### 792a19k1 UAT Scenarios

```gherkin
Scenario: 792a19k1a A deliverable-laden phrase is rewritten
  Given the agent's draft second phrase contains a flag, file or endpoint name, or restates the request
  When it re-checks before running the query
  Then the second --phrase it passes names a class of work-handling decision and contains none of those nouns

Scenario: 792a19k1b The re-check wording is runbook-agnostic
  Given the shim's re-check and its examples
  When they are read
  Then they name no specific runbook, skill or evaluation task and the examples cover at least two different kinds of work

Scenario: 792a19k1c The re-check does not regress another runbook's retrieval
  Given a runbook whose retrieval was validated before the re-check, such as the dispatch-tier runbook
  When its task is run with the shim alone
  Then the runbook is retrieved at the same rate as before
```

## 792a19l Each returned kind is treated for what it is

Requirement: [792a19l Each returned kind is treated for what it is](#792a19l-each-returned-kind-is-treated-for-what-it-is) The shim MUST tell the agent how to treat each kind a query returns, and the follow-frame
obligations ([792a19m A matched runbook is announced and its steps are the plan](#792a19m-a-matched-runbook-is-announced-and-its-steps-are-the-plan) to [792a19q Linked and truncated runbooks get the same frame](#792a19q-linked-and-truncated-runbooks-get-the-same-frame)) MUST apply to `kind: runbook` only. A `fact` is knowledge and context, taken as
true for the task unless the repository contradicts it. A `feedback` note is a correction the user already gave,
treated as a standing instruction so that the mistake it names is not repeated and the decision to honor it is not
re-derived. A `runbook` is executed under the frame. A `chunk` is raw evidence, fetched with `engram show-chunk`
only when the notes leave a gap, and its presence alone is not an instruction.

Rationale: the query is not filtered by kind, so the agent must read each item's `kind`. Four content kinds exist,
`fact`, `feedback`, `qa` and `runbook`, and `qa` is excluded from retrieval ([792a7v Runbooks rank like other
notes](./memory-query.md#792a7v-runbooks-rank-like-other-notes)). A runbook is structurally the same as a fact or
feedback note in storage and ranking. The frame lives in the shim and not in the recall skill because it must exist
before any runbook is loaded; were it in recall, recall would itself become a runbook. The shim's first action fires
on every request, which is affordable once per task and not once per tool call, so it is guidance and not a hook.

### 792a19l1 UAT Scenarios

```gherkin
Scenario: 792a19l1a A fact with steps gets no frame
  Given a query returns a fact whose body contains numbered steps
  When the agent reads it
  Then it requires no announcement or step restatement for it

Scenario: 792a19l1b Feedback is a standing instruction
  Given a query returns a feedback note naming a mistake relevant to the task
  When the agent plans
  Then its plan and actions avoid the mistake, and it does not re-derive whether to honor the note

Scenario: 792a19l1c Chunks are not instructions
  Given a query returns chunk items alongside notes
  When the agent acts
  Then it acts on the notes and fetches a chunk only to fill a gap the notes leave
```

## 792a19m A matched runbook is announced and its steps are the plan

Requirement: [792a19m A matched runbook is announced and its steps are the plan](#792a19m-a-matched-runbook-is-announced-and-its-steps-are-the-plan) When a returned note has `kind: runbook` and its `situation` matches the task at hand, and not
merely a related one, the agent MUST state in a text block which runbook it is executing, by its basename, and MUST
restate the runbook's steps as its plan, with one todo per step where the harness offers a todo list, before its
first mutating action, and MUST then do the steps in the runbook's order.

Rationale: without the frame, an agent that finds a runbook tends not to restate its steps as a plan or to reach its
end state, and the frame closes that gap. A fact with the same body gets no frame. The shim is a fresh file and not an
extension of the recall guidance, because that guidance names `/recall glance`, which a shim-only session does not
have.

### 792a19m1 UAT Scenarios

```gherkin
Scenario: 792a19m1a The announcement precedes the first mutating action
  Given a query returns a runbook whose situation matches the task
  When the agent is about to act
  Then the transcript holds a text block naming that runbook's basename before the first mutating tool call

Scenario: 792a19m1b The steps are restated and followed in order
  Given a matched runbook with N steps
  When the agent acts
  Then a text block or todo list enumerates the N steps before the first mutating command, and the commands that follow map to the steps in order

Scenario: 792a19m1c A related runbook is not followed
  Given a query returns a runbook whose situation is related to the task but does not match it
  When the agent reads it
  Then it does not announce it as the runbook it is executing
```

## 792a19n A runbook's red flags are read first and stop the agent

Requirement: [792a19n A runbook's red flags are read first and stop the agent](#792a19n-a-runbooks-red-flags-are-read-first-and-stop-the-agent) The agent MUST read a matched runbook's `red_flags`, if it has any, before it starts, and when a
listed condition occurs it MUST stop, name the red flag and the step it is rereading in its next text block, and
reread that step before its next mutating command.

### 792a19n1 UAT Scenarios

```gherkin
Scenario: 792a19n1a A red flag stops the work
  Given a matched runbook whose red_flags name a condition, and that condition occurs mid-procedure
  When the agent next writes text
  Then the text block names the red flag and the step it rereads, before the next mutating command

Scenario: 792a19n1b A runbook without red flags starts normally
  Given a matched runbook with no red_flags
  When the agent starts
  Then it proceeds without a red-flag step
```

## 792a19o A runbook's done_when is the completion bar

Requirement: [792a19o A runbook's done_when is the completion bar](#792a19o-a-runbooks-done_when-is-the-completion-bar) The agent MUST NOT report a task done until the matched runbook's `done_when` holds, and MUST
verify each condition and not assume it.

### 792a19o1 UAT Scenarios

```gherkin
Scenario: 792a19o1a Completion waits for verification
  Given the agent is about to declare a runbook task complete
  When it writes the completion message
  Then the transcript shows a verification of each done_when condition before the message

Scenario: 792a19o1b An unmet condition blocks the report
  Given a done_when condition that does not hold
  When the agent checks it
  Then it does not report the task done
```

## 792a19p The agent stops and asks when a step is genuinely uncertain

Requirement: [792a19p The agent stops and asks when a step is genuinely uncertain](#792a19p-the-agent-stops-and-asks-when-a-step-is-genuinely-uncertain) When the outcome of a step of a matched runbook is genuinely uncertain, when the step names a target
that plainly is not present (no remote, no ticket queue, no CI), or when two things the runbook requires are
genuinely in tension so that the agent cannot tell which one the person needs preserved, the agent MUST name the step
and the specific gap and end its turn with a question, and MUST NOT substitute its own action for the step, including
a local equivalent of the missing target, or proceed anyway. It MUST NOT stop to ask merely because an earlier step,
even the one that produced the most visible deliverable, is complete, nor for a rough edge in the wording: a step's
specific instruction wins over its own incidental example, and a later step its own plan already names as required
is done without asking whether to.

Rationale: a question stop is a clarity signal for the person who gave the task, who owns supplying the missing
context now and for next time, and never a failure or a reason to invent the local equivalent. The weight is greatest
before an action that cannot be undone, where an honest "I'm guessing" means ask. Stopping at a phase boundary of an
agreed workflow is a failure of the opposite kind. The shim contains no "proceed anyway" language.

### 792a19p1 UAT Scenarios

```gherkin
Scenario: 792a19p1a An ambiguous step yields a question
  Given a runbook step that is genuinely ambiguous for the current repository
  When the agent reaches it
  Then it ends its turn with a question naming the step and the ambiguity, and no substitute action appears

Scenario: 792a19p1b A missing target yields a question
  Given a step that requires a GitHub remote the repository does not have
  When the agent reaches it
  Then it asks, and does not invent a local equivalent

Scenario: 792a19p1c A verified earlier step does not license a stop
  Given the agent has verified an earlier step complete and a later step in its restated plan is unambiguous and mandatory
  When it moves on
  Then it proceeds to the later step without asking whether to continue

Scenario: 792a19p1d A specific instruction beats its incidental example
  Given a step whose specific instruction and its own example seem to clash
  When the agent reads it
  Then it follows the specific instruction and does not stop on the clash alone
```

## 792a19q Linked and truncated runbooks get the same frame

Requirement: [792a19q Linked and truncated runbooks get the same frame](#792a19q-linked-and-truncated-runbooks-get-the-same-frame) The agent MUST fetch with `engram show <basename>` and apply the whole frame ([792a19m A matched runbook is announced and its steps are the plan](#792a19m-a-matched-runbook-is-announced-and-its-steps-are-the-plan) to [792a19p The agent stops and asks when a step is genuinely uncertain](#792a19p-the-agent-stops-and-asks-when-a-step-is-genuinely-uncertain))
to a runbook that a matched runbook wikilinks, before the step that linked it. It MUST fetch a matched runbook in
full with `engram show` before it restates the steps when the query payload truncated it, and MUST NOT restate steps
from a truncated fragment. The frame MUST apply to every returned `kind: runbook` item whatever its origin, including
a runbook that the query returned from a parent vault (`from_parent`) and the runbook note of a registered skill
([skill-runbook-registration](./skill-runbook-registration.md)), whose installation as a skill is no reason to skip the frame.

Rationale: the frame is stated for the kind and not for a source, so the agent does not need to know where a note came
from. That a bare `engram show <basename>` resolves a note held only in the parent vault is parent-sync's ([parent-sync](./parent-sync.md)).

### 792a19q1 UAT Scenarios

```gherkin
Scenario: 792a19q1a A linked runbook is fetched and followed
  Given a matched runbook step reads "follow [[<basename>]]"
  When the agent reaches the step
  Then the transcript shows engram show <basename> and the linked runbook's steps restated before that step's mutating commands

Scenario: 792a19q1c A truncated runbook is fetched in full
  Given the query payload truncated a matched runbook
  When the agent prepares its plan
  Then it runs engram show <basename> and restates the steps from the full body

Scenario: 792a19q1d A parent-vault runbook is followed
  Given an agent whose session has a parent vault, and a query returns a runbook that exists only there
  When the agent runs engram show <basename> and reads the note
  Then it restates the steps as its plan

Scenario: 792a19q1e A registered skill's runbook is followed
  Given curate is installed as a skill and its runbook note is matched by the first query
  When the agent proceeds
  Then it applies the frame to the matched runbook and does not defer to the skill's own description-based firing
```

## 792a19r One floor beneath every runbook

Requirement: [792a19r One floor beneath every runbook](#792a19r-one-floor-beneath-every-runbook) The shim MUST state, once, the rules that apply to every runbook uniformly: the urge to shortcut a
step is the cue to reread it and not a reason to skip it; substituting a related action for the one the step names is
a skip, including "the named thing doesn't apply here, so I'll do what accomplishes the same goal"; and the letter
of a step is its spirit, so a step that says to run a command is satisfied only by running that command. A runbook note
MUST NOT restate these rules.

Rationale: stating the floor once keeps every note down to its situation, steps, completion bar and task-specific
red flags, so a runbook cannot drift from the floor by rewording it.

### 792a19r1 UAT Scenarios

```gherkin
Scenario: 792a19r1a The floor is in the shim and not in the notes
  Given the shim and a runbook note read together
  When the floor rules are looked for
  Then they appear in the shim and the note carries only situation, steps, done_when and task-specific red flags

Scenario: 792a19r1b A related action is not a completion
  Given a step that names a command the agent thinks is not needed here
  When the agent decides what to do
  Then it asks rather than doing something that accomplishes the same goal
```

## 792a19s A signal of unrequested work is a standing instruction

Requirement: [792a19s A signal of unrequested work is a standing instruction](#792a19s-a-signal-of-unrequested-work-is-a-standing-instruction) The shim MUST tell the agent to treat a tool result that names pending work it was not asked to do,
such as the `pending_offers` flag and hint of `engram query`, as a standing instruction, and after finishing the
user's request to do what the signal says without asking, unless the named action is destructive or hard to reverse
and the agent is genuinely uncertain, in which case it MUST stop and ask. The rule MUST be stated generically, naming
no skill or runbook, so that a later signal of the same shape needs no further edit.

Rationale: a result outside the scope of the request is easy to read as optional context. Pending offers are curate's.

### 792a19s1 UAT Scenarios

```gherkin
Scenario: 792a19s1a Pending work is acted on after the request
  Given a tool result in the turn names pending work outside the user's request, such as a pending-offer flag and hint
  When the agent has completed the user's request
  Then it acts on the named work without asking permission first

Scenario: 792a19s1b A destructive and uncertain action still asks
  Given the named work would need a destructive or hard-to-reverse action and the agent is genuinely uncertain how to proceed
  When it reaches the action
  Then it stops and asks

Scenario: 792a19s1c The rule is generic
  Given the shim's wording for this rule
  When it is read
  Then it names no specific skill or runbook
```

## 792a19t The query runs again at four moments mid-task

Requirement: [792a19t The query runs again at four moments mid-task](#792a19t-the-query-runs-again-at-four-moments-mid-task) The shim MUST tell the agent to run `engram query` again, with the same two phrase shapes re-keyed to
what is in front of it, before it endorses or ranks a proposed approach, even when only discussing it; before it
declares work done, querying first and then verifying; once after a failure it cannot immediately explain; and before
it starts building a new approach. It MUST tell the agent to escalate to the matched runbook's `deep` procedure, if it
offers one, when the decision is weighty or irreversible or the pass surfaces a gap it cannot resolve.

Rationale: the first query does not cover the rest of the task. These are the same four moments as [792a19a Recall fires at four decision moments](#792a19a-recall-fires-at-four-decision-moments), for a
session whose recall is carried by the shim.

### 792a19t1 UAT Scenarios

```gherkin
Scenario: 792a19t1a Work is queried before it is declared done
  Given the agent is about to declare work done
  When it prepares to
  Then it runs engram query keyed to the current work first, and then verifies

Scenario: 792a19t1b A new approach is queried before it is built
  Given the agent is about to start building a new approach
  When it chooses the approach
  Then it runs engram query with phrases re-keyed to the approach before it starts

Scenario: 792a19t1c A weighty decision escalates to deep
  Given a matched runbook that offers a deep procedure and a decision that is irreversible
  When the agent reaches it
  Then it uses the deep procedure
```

## 792a19u The shim joins the other guidance additively

Requirement: [792a19u The shim joins the other guidance additively](#792a19u-the-shim-joins-the-other-guidance-additively) The set of guidance documents that engram expects a harness's own config file to import MUST be
`recall.md`, `delegate.md`, `learn.md` and `shim.md`, with `shim.md` added to the other three and not substituted
for them, because the recall, learn and write-memory skills ship as skills and are not replaced by runbooks.

Rationale: `shim.md` finds and follows runbook notes and does not reproduce the content of the other three, which
cover work that is not in a runbook, so the four are an additive set. Activating the shim is a policy about which
deployed documents a session's own config imports, and is distinct from getting the files onto disk ([792a11l Guidance
is managed only after opt-in](./install-and-update.md#792a11l-guidance-is-managed-only-after-opt-in)).

### 792a19u1 UAT Scenarios

```gherkin
Scenario: 792a19u1a A config with three imports gains the shim
  Given a harness config that imports recall.md, delegate.md and learn.md and not shim.md
  When the user runs engram update
  Then the report names shim.md as deployed and not yet imported, with the line to add, and the three existing imports are untouched

Scenario: 792a19u1b The shim does not replace the other documents
  Given the recall, learn and write-memory skills ship as skills and are not replaced by runbooks
  When the user activates the shim
  Then recall.md, delegate.md and learn.md remain imported
```

## 792a19v The update report says how to activate guidance

Requirement: [792a19v The update report says how to activate guidance](#792a19v-the-update-report-says-how-to-activate-guidance) When guidance is deployed for a harness whose config file can import it, `engram update` MUST print
one line per deployed document: `guidance refreshed: ~/<guidance dir>/<name>` when that harness's config already imports
the document, and otherwise `guidance deployed to ~/<guidance dir>/<name> — add '@~/<guidance dir>/<name>' to <config
file> to activate it (<Harness> will ask you to approve the import once)`, with forward slashes on every operating
system. The two harnesses MUST be wired independently of each other. When no guidance was deployed for any harness,
the run was not given `--with-guidance` and no config file imports guidance, it MUST print the line `engram ships
recall- and delegation-firing guidance; run 'engram update --with-guidance' to deploy it`, and otherwise it MUST NOT.
Under `--dry-run` each guidance line MUST carry the dry-run prefix.

Rationale: an import is a line in a file the user owns, which engram does not edit, so the report tells the user
exactly what to add. The nudge is shown only to a user who has neither opted in nor wired an import, and so has not
yet been told guidance exists. The canonical-path suffix on Claude Code lines is install-and-update's ([install-and-update](./install-and-update.md); [792a11m1d The report states the canonical path](./install-and-update.md#792a11m1-uat-scenarios)).

### 792a19v1 UAT Scenarios

```gherkin
Scenario: 792a19v1a A Claude Code document not yet imported is deployed with its line to add
  Given --with-guidance deploys recall.md for Claude Code and ~/.claude/CLAUDE.md does not import it
  When the user runs engram update --with-guidance
  Then the report says "guidance deployed to ~/.claude/engram/recall.md — add '@~/.claude/engram/recall.md' to ~/.claude/CLAUDE.md to activate it" and that Claude Code will ask for approval once

Scenario: 792a19v1b An imported document is refreshed
  Given ~/.claude/CLAUDE.md imports recall.md
  When the user runs engram update
  Then the report says "guidance refreshed: ~/.claude/engram/recall.md" and gives no line to add

Scenario: 792a19v1c A Pi document is deployed with Pi's paths
  Given --with-guidance deploys recall.md for Pi and ~/.pi/agent/AGENTS.md does not import it
  When the user runs engram update --with-guidance
  Then the report says "guidance deployed to ~/.pi/agent/guidance/recall.md — add '@~/.pi/agent/guidance/recall.md' to ~/.pi/agent/AGENTS.md to activate it" and that Pi will ask for approval once

Scenario: 792a19v1d The harnesses are wired independently
  Given Claude Code imports recall.md and Pi does not
  When the user runs engram update
  Then the report says refreshed for Claude Code and gives Pi's line to add

Scenario: 792a19v1e Each document is judged on its own
  Given Claude Code imports recall.md and not delegate.md
  When the user runs engram update
  Then recall.md is reported refreshed and delegate.md is reported deployed with its line to add

Scenario: 792a19v1f A user who has not opted in is nudged
  Given no --with-guidance, no guidance imported and no guidance deployed
  When the user runs engram update
  Then the report says engram ships recall- and delegation-firing guidance and names engram update --with-guidance

Scenario: 792a19v1g A user who already imports is not nudged
  Given no --with-guidance flag and a config that imports at least one guidance document
  When the user runs engram update
  Then the report prints no nudge, and each guidance document is reported refreshed or deployed on its own as in [792a19v1e Each document is judged on its own](#792a19v1-uat-scenarios)

Scenario: 792a19v1h The flag suppresses the nudge
  Given --with-guidance and no guidance deployed for any harness
  When the user runs engram update --with-guidance
  Then the report does not print the nudge

Scenario: 792a19v1i A dry run prefixes the guidance lines
  Given a harness with guidance to deploy and a dry run
  When the user runs engram update --dry-run --with-guidance
  Then each guidance line begins with "[dry-run] "
```
