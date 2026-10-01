## MODIFIED Requirements

### Requirement: Accepting a refresh SHALL replace the body, keep the fields, and mark the note pending
Accepting a refresh offer SHALL do the following:

- replace the note's body with the current source file, preamble included;
- set `skill_hash` to the current hash;
- set `skill_source` to the current resolved source, keeping `skill_key`;
- preserve `situation`, `triggers`, `done_when`, `red_flags`, `created`, and the basename;
- preserve every other frontmatter key with its value, including keys the typed runbook note model does not define;
- rebuild the sidecar;
- set `pending: true`, so curation re-checks the fields against the new text.

#### Scenario: Accepted refresh
- **WHEN** the user accepts a refresh of a note whose skill changed
- **THEN** the note has the new body and hash, its authored fields and basename are unchanged, and it carries `pending: true`

#### Scenario: Refresh follows a plugin version bump
- **WHEN** a plugin skill's bytes change with a new `installPath` version and the refresh is accepted
- **THEN** `skill_source` and the preamble name the new version's path

#### Scenario: Refresh keeps an unmodeled key
- **WHEN** a refresh is accepted for a note whose frontmatter carries a key the runbook note model does not define (e.g. `luhmann_old: "12"`)
- **THEN** the refreshed note still carries that key with the same value

### Requirement: An existing runbook note SHALL be adoptable as a skill's note
`engram register-skills --adopt <key>=<note-ref>` SHALL act on the referenced runbook note as follows:

- rename it to the slug derived from `<key>`, keeping its Luhmann id and date and rewriting every inbound wikilink (with its sidecar);
- replace its body with the current source file and preamble;
- set `skill_hash`, `skill_key`, and `skill_source`;
- preserve its runbook fields and every other frontmatter key with its value, including keys the typed runbook note model does not define.

`<key>` SHALL name a scanned skill or command. The adopted note SHALL NOT be marked pending.

#### Scenario: Adopting a previously promoted note
- **WHEN** `engram register-skills --adopt claude:curate=1049` runs and note 1049 is `1049.2026-09-21.skill-curate.md` with no `skill_key`
- **THEN** note 1049 is renamed to `1049.2026-09-21.skill-claude-curate.md`, links to its old basename now point to the new one, its body is the curate skill preceded by a preamble naming its `skill_source`, its runbook fields are unchanged, and it has `skill_hash`, `skill_key: claude:curate`, `skill_source`, and no `pending` marker

#### Scenario: Adopt keeps an unmodeled key
- **WHEN** `--adopt` runs on a runbook note whose frontmatter carries a key the runbook note model does not define
- **THEN** the adopted note still carries that key with the same value
