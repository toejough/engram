## MODIFIED Requirements

### Requirement: Write-memory worker SHALL accept handoff and execute vault writes

The worker is the `write-memory` skill (restored to `agent-instructions/skills/write-memory/SKILL.md`; its vault runbook is the registered, derived mirror of that skill per capability `skill-runbook-registration`) and is invoked by a parent skill (`recall`, `learn`) that has already made the judgment (what to write and why), as the parent's next action — write-memory has no trigger of its own and is never reached by a user's own words. The worker SHALL receive a structured handoff containing kind (fact/feedback/qa/runbook), content fields, source, and optional chunk-sources, tags, and supersedes. The worker SHALL NOT re-judge the parent's decision and SHALL NOT decide whether to write.

#### Scenario: Worker receives handoff from parent skill

- **WHEN** a parent skill (recall, learn) invokes write-memory with a complete handoff (kind, required content fields, source)
- **THEN** the worker SHALL compose the corresponding `engram learn` command from the provided fields

#### Scenario: Worker rejects incomplete handoff

- **WHEN** required handoff fields are missing
- **THEN** the worker SHALL ask the parent skill (via in-session context) to provide the missing fields
- **AND** the worker SHALL NOT invent content on behalf of the parent

#### Scenario: Worker reachable both as a skill and as its derived runbook

- **WHEN** a session has the write-memory skill installed and its derived runbook registered in the vault
- **THEN** a parent skill invokes it natively as a skill, while a session that received only the derived runbook over a vault-graph edge follows the runbook via the shim's follow-frame — both paths execute the same handoff contract
