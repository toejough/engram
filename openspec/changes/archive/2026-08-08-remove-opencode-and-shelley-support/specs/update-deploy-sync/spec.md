## MODIFIED Requirements

### Requirement: Harness-visible paths are symlinks into the engram-owned root
For harnesses in symlink deploy mode, `engram update` SHALL materialize harness-required artifact paths as symlinks into the engram-owned root — one symlink per skill directory, one per guidance file — never as real copies.

#### Scenario: Skill deploys as a directory symlink
- **WHEN** `engram update` deploys skill `recall` for a symlink-mode harness with skills root `~/.claude/skills`
- **THEN** `~/.claude/skills/recall` is a symlink whose target is the `skills/recall` subtree of that harness's engram-owned root, and the SKILL.md bytes are readable through it

### Requirement: Manifest fallback for harnesses without symlink discovery
A harness whose skill discovery fails symlink verification SHALL run in manifest deploy mode: artifacts are materialized as real copies, every written path is recorded in a manifest inside the engram-owned root, and sync SHALL delete exactly those recorded paths whose source artifacts no longer exist. Paths not recorded in the manifest MUST NOT be deleted.

#### Scenario: Removal propagates via manifest
- **WHEN** a manifest-mode harness has a deployed skill file recorded in the manifest and the skill's source is deleted, and `engram update` runs
- **THEN** the deployed skill file is deleted, the manifest entry is dropped, and unrecorded files in the same directory are untouched
