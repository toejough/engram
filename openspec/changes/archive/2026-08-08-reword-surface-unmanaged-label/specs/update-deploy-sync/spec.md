## MODIFIED Requirements

### Requirement: First sync migrates previously-copied deployments
On the first sync for a harness (no ownership marker present), `engram update` SHALL create the root and marker, SHALL replace each intended-set artifact found as a real file or directory at its harness path with the synced root content plus a symlink, and SHALL report — without deleting — real files at harness artifact paths that are not in the intended set.

#### Scenario: Copied skill converts to symlink
- **WHEN** the first sync runs for a harness whose skills dir holds skill `recall` as a real directory from a prior copy-mode deploy
- **THEN** after update, the skill's content lives in the engram-owned root, the harness path is a symlink to it, and the skill remains discoverable

#### Scenario: Unmanaged surface entry is reported, not deleted
- **WHEN** the first sync finds a real file in a harness skills dir that is not part of the intended deploy set
- **THEN** the file is left in place and the update report lists it as unmanaged, left alone, for manual review
