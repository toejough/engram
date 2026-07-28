# Update Deploy Sync Specification

## Purpose

The sync deployment model of `engram update`: each detected harness gets an engram-owned root that update syncs to exactly match the intended deploy set computed from `agent-instructions/` — additions, updates, and removals all propagate — with harness-visible paths materialized as symlinks into that root, dangling-link cleanup on every run, ownership-marker-bounded deletion, one-time migration of previously-copied deployments, and a recorded-manifest fallback for harnesses whose discovery fails symlink verification. Deletion is safe by construction: engram is the sole writer inside its owned root, and the only deletions outside it are symlinks that provably point into it — user-owned files are never deleted. Why: docs/architecture/adr.md ADR-0022. Validation: internal/update/sync_test.go, internal/update/symlink_test.go, internal/update/migration_test.go, plus manifest and dry-run contract tests in internal/update/update_test.go and internal/cli/update_test.go.

## Requirements

### Requirement: Engram-owned root syncs to the intended deploy set
`engram update` SHALL maintain one engram-owned root per detected harness and SHALL sync it to exactly match the intended deploy set computed by the planners: artifacts missing from the root are created, artifacts that differ are overwritten, and files present in the root but absent from the intended set are deleted.

#### Scenario: Removed source artifact disappears on next update
- **WHEN** a skill directory that was previously deployed is deleted from `agent-instructions/skills/` and `engram update` runs
- **THEN** the skill's subtree is deleted from the engram-owned root and its harness-visible symlink is removed

#### Scenario: Added and changed source artifacts deploy
- **WHEN** a new skill is added to the source and an existing skill's file is edited, and `engram update` runs
- **THEN** the new skill appears in the engram-owned root with a harness-visible symlink, and the edited file's content in the root matches the source

### Requirement: Harness-visible paths are symlinks into the engram-owned root
For harnesses in symlink deploy mode, `engram update` SHALL materialize harness-required artifact paths as symlinks into the engram-owned root — one symlink per skill directory, one per command file, one per guidance file — never as real copies.

#### Scenario: Skill deploys as a directory symlink
- **WHEN** `engram update` deploys skill `recall` for a symlink-mode harness with skills root `~/.claude/skills`
- **THEN** `~/.claude/skills/recall` is a symlink whose target is the `skills/recall` subtree of that harness's engram-owned root, and the SKILL.md bytes are readable through it

### Requirement: Ownership marker bounds sync deletion
Each engram-owned root SHALL carry an ownership marker file written when engram creates or adopts the root. `engram update` MUST NOT sync-delete inside a directory lacking the marker; when a root candidate exists without a marker and holds files engram cannot attribute to the intended deploy set, update SHALL report them and refuse sync-deletion there rather than delete.

#### Scenario: Unmarked directory is not purged
- **WHEN** the engram-owned root path exists without the ownership marker and contains a file not in the intended deploy set
- **THEN** `engram update` leaves the file in place and reports it, and no sync-deletion occurs in that directory

### Requirement: Dangling engram symlinks are cleaned up
On every run, `engram update` SHALL scan each harness surface directory for symlinks whose lexically-resolved target lies inside that harness's engram-owned root and whose target no longer exists, and SHALL delete exactly those. Symlinks pointing elsewhere and real files or directories MUST NOT be touched by cleanup.

#### Scenario: Dangling engram link removed, foreign link kept
- **WHEN** a harness skills dir contains a symlink into the engram-owned root whose target was removed by sync, and a second symlink pointing outside the root
- **THEN** update deletes the first symlink and leaves the second untouched

#### Scenario: User-owned real files are never deleted
- **WHEN** a harness skills dir contains a real (non-symlink) skill directory the user installed by other means
- **THEN** `engram update` leaves it untouched across sync, cleanup, and migration

### Requirement: First sync migrates previously-copied deployments
On the first sync for a harness (no ownership marker present), `engram update` SHALL create the root and marker, SHALL replace each intended-set artifact found as a real file or directory at its harness path with the synced root content plus a symlink, and SHALL report — without deleting — real files at harness artifact paths that are not in the intended set.

#### Scenario: Copied skill converts to symlink
- **WHEN** the first sync runs for a harness whose skills dir holds skill `recall` as a real directory from a prior copy-mode deploy
- **THEN** after update, the skill's content lives in the engram-owned root, the harness path is a symlink to it, and the skill remains discoverable

#### Scenario: Unattributable stray is reported, not deleted
- **WHEN** the first sync finds a real file in a harness skills dir that is not part of the intended deploy set
- **THEN** the file is left in place and the update report lists it as a possible stale artifact for manual review

### Requirement: Guidance opt-in gates management, not removal
Guidance SHALL be part of the intended deploy set only when the `--with-guidance` flag is passed or the harness config file already imports the harness's guidance files. When guidance is not in the intended set, `engram update` MUST leave the root's guidance subtree and any harness-visible guidance surface untouched — unmanaged, not removed.

#### Scenario: Non-opted harness keeps existing guidance
- **WHEN** a harness has previously-deployed guidance files, no `--with-guidance` flag is passed, and its config file no longer contains guidance imports
- **THEN** `engram update` neither refreshes nor deletes those guidance files

### Requirement: Claude guidance import paths keep working through migration
Because user-authored `@import` lines reference `~/.claude/engram/<file>.md` verbatim, the Claude Code first-sync migration SHALL leave compatibility symlinks at those flat paths pointing at the canonical `guidance/` locations inside the root, and the update report SHALL state the canonical paths.

#### Scenario: Existing CLAUDE.md import resolves after migration
- **WHEN** the Claude Code first-sync migration moves `recall.md` into the root's `guidance/` subtree and the user's `CLAUDE.md` imports `~/.claude/engram/recall.md`
- **THEN** `~/.claude/engram/recall.md` is a symlink to the canonical file and the import still resolves to the deployed content

### Requirement: Manifest fallback for harnesses without symlink discovery
A harness whose skill or command discovery fails symlink verification SHALL run in manifest deploy mode: artifacts are materialized as real copies, every written path is recorded in a manifest inside the engram-owned root, and sync SHALL delete exactly those recorded paths whose source artifacts no longer exist. Paths not recorded in the manifest MUST NOT be deleted.

#### Scenario: Removal propagates via manifest
- **WHEN** a manifest-mode harness has a deployed command recorded in the manifest and the command's source is deleted, and `engram update` runs
- **THEN** the deployed command file is deleted, the manifest entry is dropped, and unrecorded files in the same directory are untouched

### Requirement: Dry-run previews every sync operation without writing
With `--dry-run`, `engram update` SHALL emit one operation-classified, dry-run-prefixed line per planned action — sync writes, sync deletions, symlink creations, cleanup deletions, and migration steps — and MUST NOT create, modify, or delete any file, including the engram-owned root and its marker.

#### Scenario: First-sync dry-run leaves no trace
- **WHEN** `engram update --dry-run` runs for a harness that has never been synced
- **THEN** the output lists the planned migration and sync operations with the dry-run prefix on every line, and no root, marker, symlink, or file exists afterward
