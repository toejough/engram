# Runbook Follow-Frame Guidance Specification

## Purpose

The engram shim (a new guidance file, `agent-instructions/guidance/shim.md`, deployed via `engram update --with-guidance`) is the only custom instruction text an agent needs; every procedure, including recall and learn, is a runbook note it finds and follows. This capability covers the follow half: what the shim requires of an agent once a `kind: runbook` note is returned, and the bootstrap action that works with no engram skill installed. Why: the runbook-vs-skill eval showed runbooks found 3/3 but restated-as-plan 0/6 and end-state 0/6 against a skill row of 3/3/2 (checkpoint 2026-09-13); design.md D1–D4.
## Requirements
### Requirement: The shim SHALL name the first action as a literal `engram query` on every user request

Before the first tool call on any user request, the shim SHALL instruct the agent to run `engram query` with at least two `--phrase` values (the request in the agent's own words; the kind of situation it is in) and a `--text` value carrying the user's message verbatim (first ~300 characters if long, never rewritten), as a bare imperative naming the command, so that the instruction is executable with no `recall` skill present. The trigger SHALL NOT depend on the agent judging the task multi-step, recurring, or previously done: that judgment is what the returned runbook supplies, so it cannot gate the lookup. Over-fire on one-shot requests is accepted (task-init fire-unit, measured 3.4x; per-fire cost is one query with no crystallization). Adjacent to the phrase-shape instruction, the shim SHALL carry a generic re-check on the second phrase: if it contains a concrete noun from the ticket (a file, flag, feature, bug, or field name) or reads like a paraphrase of the request, the agent SHALL rewrite it as the kind of work-handling decision (how to carry out, verify, review, or land this class of work) before running the query. The shim SHALL illustrate the second-phrase shape with at least two examples of different kinds of work, and the re-check and examples SHALL NOT name any specific runbook, skill, or eval task.

#### Scenario: Shim-only session runs the first query

- **WHEN** an agent with the shim in CLAUDE.md and no engram skills installed receives a user request
- **THEN** its first tool call is an `engram query` carrying the request phrase, the situation phrase, and `--text` with the user's message verbatim, before any other tool call

#### Scenario: A one-shot request still fires

- **WHEN** the request is a single-step action (a one-line answer, a single-file typo fix)
- **THEN** the query still runs first; the agent proceeds without a runbook when none matches

#### Scenario: Deliverable-laden second phrase is rewritten before the query runs

- **WHEN** the agent's draft second phrase contains a ticket-specific noun (for example a flag, file, or endpoint name) or restates the request
- **THEN** the second `--phrase` actually passed to `engram query` names a class of work-handling decision and contains none of those ticket-specific nouns

#### Scenario: Re-check wording is runbook-agnostic

- **WHEN** the shim's phrase-two re-check and its examples are read
- **THEN** they mention no specific runbook, skill, or eval task, and the examples cover at least two different kinds of work

#### Scenario: Re-check does not regress other runbooks' retrieval

- **WHEN** a previously validated runbook's task (for example route's dispatch-tier task) is run shim-only with the updated shim
- **THEN** that runbook is still retrieved at the same rate as before the shim change

#### Scenario: The raw text is not paraphrased

- **WHEN** the user's message is "/please rename the tally CLI's --out flag to --output"
- **THEN** the `--text` value contains that string verbatim up to whitespace differences; the paraphrase goes only in `--phrase`

### Requirement: The agent SHALL announce a matched runbook by name before acting

When a returned note has `kind: runbook` and its `situation` matches the task at hand (not merely a related one), the agent SHALL state which runbook it is executing, by basename, before its first mutating action.

#### Scenario: Announcement precedes the first mutating command

- **WHEN** a query returns a runbook whose situation matches the task
- **THEN** the transcript contains an assistant text block naming that runbook's basename before the first mutating tool call

### Requirement: The agent SHALL restate a matched runbook's steps as its plan and execute them in order

The agent SHALL restate the runbook body's steps as its plan before the first mutating action; where the harness offers a todo list, one todo per step; steps SHALL be done in the runbook's order.

#### Scenario: Steps restated before execution

- **WHEN** a matched runbook has N steps
- **THEN** an assistant text block (or todo list) enumerating those N steps appears before the first mutating command, and the mutating commands that follow map to the steps in order

### Requirement: The agent SHALL treat `done_when` as the completion bar

The agent SHALL NOT report the task done until the matched runbook's `done_when` holds, and SHALL verify it rather than assume it.

#### Scenario: Completion gated on done_when

- **WHEN** the agent is about to declare the task complete
- **THEN** the transcript shows a verification of each `done_when` condition before the completion message

### Requirement: The agent SHALL stop and ask when a step is unclear or would be deviated from

If a step cannot be executed as written, is genuinely ambiguous in this context, or the agent
would deviate from it, the agent SHALL stop and ask rather than proceed. A question stop is a
clarity signal for the caller (who owns supplying the missing context now and for future runs),
never a failure, and the shim SHALL NOT contain "proceed anyway" language. Completing an earlier
step — including the one that produces the task's most visible or "primary" deliverable — is NOT,
by itself, evidence that a later step is ambiguous: when the agent's own restated plan already
names a later step as required, the agent SHALL proceed to it without stopping to ask whether to.

#### Scenario: Unclear step yields a question, not a substitute

- **WHEN** a runbook step is ambiguous for the current repository
- **THEN** the agent ends its turn with a question naming the step and the ambiguity, and no
  substitute action for that step appears in the transcript

#### Scenario: A verified earlier step does not license a stop before an unambiguous mandatory step

- **WHEN** the agent has verified an earlier step complete and a later step in its restated plan is
  unambiguous and already marked mandatory
- **THEN** the agent proceeds to that later step without ending its turn to ask permission, and the
  transcript contains no question about whether to continue

### Requirement: The shim SHALL state a general behavioral floor that applies to every runbook

The shim SHALL state, once, rules that apply to all runbooks uniformly: the urge to shortcut a step is the cue to reread it, not a reason; substituting a related action for the named one is a skip; the letter of a step is its spirit. Runbook notes SHALL NOT restate these rules.

#### Scenario: Floor present in the shim, absent from notes

- **WHEN** the shim and a runbook note are read together
- **THEN** the floor rules appear in the shim and the note carries only situation, steps, done_when, and task-specific red flags

### Requirement: The agent SHALL read a runbook's `red_flags` and stop on one firing

Before starting, the agent SHALL read the matched runbook's `red_flags` (if present). When a listed condition is observed, the agent SHALL stop and reread the relevant step before continuing.

#### Scenario: Red flag observed mid-procedure

- **WHEN** a condition named in `red_flags` occurs during execution
- **THEN** the agent's next text block names the red flag and the step it rereads, before the next mutating command

### Requirement: The agent SHALL fetch and follow wikilinked runbooks transitively

When a matched runbook's body wikilinks another runbook, the agent SHALL fetch it (`engram show <basename>`) and apply this same frame to it. When the query payload truncates a matched runbook, the agent SHALL fetch it in full with `engram show` before restating its steps.

#### Scenario: Sub-runbook followed

- **WHEN** a matched runbook step reads "follow [[<runbook basename>]]"
- **THEN** the transcript shows `engram show <basename>` and the sub-runbook's steps restated before that step's mutating commands

### Requirement: A runbook named by a skill's own instructions SHALL receive the same follow-frame treatment as a matched runbook

When a skill's own instructions (not a query-matched runbook) name a specific runbook by basename or `[[wikilink]]` as the agent's next action, the agent SHALL fetch it (`engram show <basename>`) and apply the same follow-frame obligations as a query-matched runbook: announce it by name, restate its steps as the plan, treat its `done_when` as the completion bar, and read and react to its `red_flags`. This generalizes the existing wikilinked-transitively requirement (runbook body → runbook) to the case where the referring text is still a skill, not a runbook.

#### Scenario: A skill names a worker runbook as its next action

- **WHEN** a skill's instructions read "fetch and follow `[[<basename>]]` with this handoff" at a write site
- **THEN** the transcript shows `engram show <basename>` and the runbook's steps restated before the write is composed

#### Scenario: The named runbook's red_flags still apply

- **WHEN** the runbook fetched by name carries `red_flags`
- **THEN** the agent reads them and stops if one fires, exactly as it would for a runbook matched by `engram query`

### Requirement: The shim SHALL state how each returned kind is treated

The shim SHALL state, in one line each, the treatment of every kind a query returns: a `fact` is knowledge and context, taken as true for the task unless the repository contradicts it; a `feedback` note is a correction the user already gave, treated as a standing instruction so the mistake it names is not repeated; a `runbook` is executed under the frame in this specification; a `chunk` is raw evidence, fetched (`engram show-chunk`) only when the notes leave a gap. The announce, restate, done_when, red_flags, and transitive-follow requirements SHALL apply to `kind: runbook` only.

#### Scenario: A fact with procedural text gets no frame

- **WHEN** a query returns a `fact` note whose body contains numbered steps
- **THEN** the shim requires no announcement or step restatement for it

#### Scenario: Feedback treated as a standing instruction

- **WHEN** a query returns a `feedback` note naming a mistake relevant to the task or situation
- **THEN** the agent's plan or actions avoid the named mistake, and the transcript does not show the agent re-deriving whether to honor it

#### Scenario: Chunks are not instructions

- **WHEN** a query returns `chunk` items alongside notes
- **THEN** the agent acts on the notes and fetches a chunk only to fill a gap the notes leave

### Requirement: The shim SHALL treat an unrequested-work signal in a tool result as a standing instruction

When a tool result (not only a `kind: runbook` note — any tool output, such as `engram query`'s
`pending_offers` flag and hint) names pending work the agent was not asked to do, the shim SHALL
instruct the agent to treat it as a standing instruction rather than optional context: after
finishing the user's request, the agent SHALL do what the signal says without asking. This
instruction SHALL NOT override the existing genuine-ambiguity stop-and-ask default: when the named
action is destructive or hard to reverse and the agent is genuinely uncertain about it, the agent
SHALL stop and ask instead, per that default. The shim SHALL state this rule generically, naming no
specific skill or runbook, so it applies uniformly to any current or future tool-result
signal of this shape.

#### Scenario: An unrequested pending-work signal is acted on after the main task

- **WHEN** a tool result an agent's turn produces names pending work outside the scope of the
  user's request (for example a pending-offer flag and hint)
- **THEN** after completing the user's request, the agent's transcript shows it acting on the named
  work without asking permission first

#### Scenario: Destructive, genuinely uncertain action still stops and asks

- **WHEN** the named unrequested work would require a destructive or hard-to-reverse action and the
  agent is genuinely uncertain how to proceed
- **THEN** the agent stops and asks instead of acting, consistent with the existing
  genuine-ambiguity stop-and-ask default

#### Scenario: The rule is stated generically, not tied to one signal

- **WHEN** the shim's wording for this rule is read
- **THEN** it names no specific skill or runbook, so a future tool-result signal of the
  same shape is covered without a further shim edit

