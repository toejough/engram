## MODIFIED Requirements

### Requirement: Carrier is the please skill's Step 3 sub-runbook

This procedure SHALL be carried by the `please` skill's Step 3 sub-runbook file (`agent-instructions/skills/please/runbooks/<slug>.md`, registered per capability `skill-runbook-registration`).

#### Scenario: Carrier is the please skill's Step 3 sub-runbook

The top runbook note/skill `please` SHALL have its Step 3 body wikilink the doc-surface enumeration sub-runbook (`[[<basename>]]`); the top's enumerator SHALL run that sub-runbook to completion before proceeding to Step 4. The top body SHALL not replicate the enumeration procedure — the sub-runbook is the single carrier and sole authority on how enumeration runs — so maintenance of the enumeration logic lives in one place.
