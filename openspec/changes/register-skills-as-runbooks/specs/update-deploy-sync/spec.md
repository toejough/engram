## ADDED Requirements

### Requirement: Update SHALL run registration offers after deploying skills
After the engram-owned root sync completes, `engram update` SHALL run skill registration (capability `skill-runbook-registration`) against the resolved vault: prompting for each outstanding offer when stdin is a terminal, and otherwise writing nothing and reporting the outstanding offers. Registration failures SHALL be reported and SHALL NOT roll back the deploy.

#### Scenario: Interactive update
- **WHEN** `engram update` runs in a terminal and a shipped skill has no note
- **THEN** the skills are deployed to every harness AND the user is offered registration of that skill

#### Scenario: Non-interactive update
- **WHEN** `engram update` runs with stdin not a terminal and offers are outstanding
- **THEN** the skills are deployed, no vault note or decline record is written, and the report names the skills awaiting an answer

#### Scenario: Dry run previews offers too
- **WHEN** `engram update --dry-run` runs
- **THEN** the plan lists the registration offers alongside the sync operations, and nothing is written
