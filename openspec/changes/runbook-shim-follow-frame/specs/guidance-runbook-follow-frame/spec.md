# Runbook Follow-Frame Guidance Specification

## Purpose

The engram shim (a new guidance file, `agent-instructions/guidance/shim.md`, deployed via `engram update --with-guidance`) is the only custom instruction text an agent needs; every procedure, including recall and learn, is a runbook note it finds and follows. This capability covers the follow half: what the shim requires of an agent once a `kind: runbook` note is returned, and the bootstrap action that works with no engram skill installed. Why: the runbook-vs-skill eval showed runbooks found 3/3 but restated-as-plan 0/6 and end-state 0/6 against a skill row of 3/3/2 (checkpoint 2026-09-13); design.md D1–D4.

## ADDED Requirements

### Requirement: The shim SHALL name the first action as a literal `engram query` that needs no skill

Before starting a multi-step task the agent may have done before, the shim SHALL instruct the agent to run `engram query` with at least two `--phrase` values (the task in the agent's own words; the kind of situation it is in), as a bare imperative naming the command, so that the instruction is executable with no `recall` skill present.

#### Scenario: Shim-only session runs the first query
- **WHEN** an agent with the shim in CLAUDE.md and no engram skills installed begins a multi-step task
- **THEN** its first tool call is an `engram query` carrying the task phrase and the situation phrase before any mutating command

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

If a step cannot be executed as written, is ambiguous in this context, or the agent would deviate from it, the agent SHALL stop and ask rather than proceed. A question stop is a clarity signal for the caller (who owns supplying the missing context now and for future runs), never a failure; the shim SHALL NOT contain "proceed anyway" language.

#### Scenario: Unclear step yields a question, not a substitute
- **WHEN** a runbook step is ambiguous for the current repository
- **THEN** the agent ends its turn with a question naming the step and the ambiguity, and no substitute action for that step appears in the transcript

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

### Requirement: The shim SHALL state how each returned kind is treated

The shim SHALL state, in one line each, the treatment of every kind a query returns: a `fact` is knowledge and context, taken as true for the task unless the repository contradicts it; a `feedback` note is a correction the user already gave, treated as a standing instruction so the mistake it names is not repeated; a `runbook` is executed under the frame in this specification; a `chunk` is raw evidence, fetched (`engram show-chunk`) only when the notes leave a gap. The announce, restate, done_when, red_flags, and transitive-follow requirements SHALL apply to `kind: runbook` only.

#### Scenario: A fact with procedural text gets no frame
- **WHEN** a query returns a `fact` note whose body contains numbered steps
- **THEN** the shim requires no announcement or step restatement for it

#### Scenario: Feedback treated as a standing instruction
- **WHEN** a query returns a `feedback` note naming a mistake relevant to the task
- **THEN** the agent's plan or actions avoid the named mistake, and the transcript does not show the agent re-deriving whether to honor it

#### Scenario: Chunks are not instructions
- **WHEN** a query returns `chunk` items alongside notes
- **THEN** the agent acts on the notes and fetches a chunk only to fill a gap the notes leave
