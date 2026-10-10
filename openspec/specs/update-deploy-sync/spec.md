# Update Deploy Sync Specification

## Purpose

The product behavior of the sync deployment model (engram-owned roots, symlinked surfaces, removals that propagate, the ownership marker, dangling-link cleanup, first-sync migration, guidance opt-in, the Claude import compat links and the dry-run preview) is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a11-added-install-and-update` (792a11f, 792a11h to 792a11m). The manifest fallback below waits for harness-deploy-sync (792c7j3), and the registration requirement for skill-runbook-registration (792c7k1). Why: docs/architecture/adr.md ADR-0022. Validation: internal/update/sync_test.go, internal/update/symlink_test.go, internal/update/migration_test.go, plus manifest and dry-run contract tests in internal/update/update_test.go and internal/cli/update_test.go.

## Requirements

### Requirement: Manifest fallback for harnesses without symlink discovery
A harness whose skill discovery fails symlink verification SHALL run in manifest deploy mode: artifacts are materialized as real copies, every written path is recorded in a manifest inside the engram-owned root, and sync SHALL delete exactly those recorded paths whose source artifacts no longer exist. Paths not recorded in the manifest MUST NOT be deleted.

#### Scenario: Removal propagates via manifest
- **WHEN** a manifest-mode harness has a deployed skill file recorded in the manifest and the skill's source is deleted, and `engram update` runs
- **THEN** the deployed skill file is deleted, the manifest entry is dropped, and unrecorded files in the same directory are untouched

### Requirement: Update SHALL run registration offers after deploying skills
After the engram-owned root sync completes, `engram update` SHALL run skill registration (capability `skill-runbook-registration`) against the resolved vault over the same default skill and command source set that standalone `engram register-skills` scans — resolved by the same function from the same home directory and working directory, and not from `<sourceRoot>/agent-instructions/skills` — prompting for each outstanding offer when stdin is a terminal, and otherwise writing nothing and reporting the outstanding offers. Registration failures SHALL be reported and SHALL NOT roll back the deploy.

#### Scenario: Interactive update
- **WHEN** `engram update` runs in a terminal and a shipped skill has no note
- **THEN** the skills are deployed to every harness AND the user is offered registration of that skill

#### Scenario: Non-interactive update
- **WHEN** `engram update` runs with stdin not a terminal and offers are outstanding
- **THEN** the skills are deployed, no vault note or decline record is written, and the report names the skills awaiting an answer

#### Scenario: Dry run previews offers too
- **WHEN** `engram update --dry-run` runs
- **THEN** the plan lists the registration offers alongside the sync operations, and nothing is written

#### Scenario: Update and register-skills see the same offers
- **WHEN** `engram update --dry-run` and `engram register-skills --dry-run` run from the same directory against the same home and vault after a completed sync
- **THEN** both list the same registration offers
