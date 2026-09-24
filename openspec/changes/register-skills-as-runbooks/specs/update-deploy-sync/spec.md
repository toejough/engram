## ADDED Requirements

### Requirement: Update SHALL register skills into the vault after deploying them
After the engram-owned root sync completes, `engram update` SHALL run skill registration (capability `skill-runbook-registration`) against the resolved vault, so deployed skills and their derived runbook notes never diverge. Registration failures SHALL be reported and SHALL NOT roll back the deploy.

#### Scenario: Deploy then register
- **WHEN** `engram update` runs with registered skills in the source
- **THEN** the skills are deployed to every harness AND their derived notes are current in the vault

#### Scenario: Dry run previews registration too
- **WHEN** `engram update --dry-run` runs
- **THEN** the plan lists the registration creates/updates/removes alongside the sync operations, and nothing is written
