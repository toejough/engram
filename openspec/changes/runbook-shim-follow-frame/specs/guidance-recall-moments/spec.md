## ADDED Requirements

### Requirement: Agent SHALL run `engram query` before starting a multi-step task it may have done before

Before starting a multi-step task in this or a similar situation (a recurring routine, a procedure done before), the agent SHALL run `engram query` with the task phrase and the situation phrase, as a literal command in the guidance, so that firing does not depend on a `recall` skill being installed. This is the task-start cue from #735 generalized into the bootstrap action of the shim (design.md D2).

#### Scenario: Task-start query fires without the recall skill
- **WHEN** an agent with the guidance loaded and no `recall` skill installed begins a multi-step task
- **THEN** it runs `engram query --phrase "<task in its words>" --phrase "<situation kind>"` before its first mutating tool call

#### Scenario: One-shot tasks do not fire the task-start cue
- **WHEN** the task is a single-step action (a one-line answer, a single-file typo fix)
- **THEN** the guidance does not require the task-start query
