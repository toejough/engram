## MODIFIED Requirements

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
