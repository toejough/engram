## ADDED Requirements

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
