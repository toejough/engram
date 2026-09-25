## Purpose

Skills shipped in `agent-instructions/skills/` remain the shipping form of every procedure (deployed by `engram update`), and their files stay exactly as skill-loading harnesses expect. Each shipped skill may have one runbook note in the vault that mirrors the skill's text and carries the runbook fields (`situation`, `triggers`, `done_when`, `red_flags`) authored on the note, so memory can surface the procedure by situation and trigger, the shim's follow-frame can drive it, and it can flow across vault-graph edges. Engram offers to create, refresh, and remove these notes; the user decides, and a declined offer is not repeated. Why: vault note 1054 (reversing 1031); the four 2026-09 delete-conversions broke distribution for every install but Joe's.

## ADDED Requirements

### Requirement: Skill files SHALL carry no engram-specific metadata
Registration SHALL NOT read, require, or write any frontmatter field in a skill's `SKILL.md` other than using the file's bytes as the note body and hash input, and SHALL NOT require any additional file in the skill directory.

#### Scenario: Skill deploys unchanged
- **WHEN** a skill whose `SKILL.md` frontmatter has only `name` and `description` is deployed and registered
- **THEN** the deployed `SKILL.md` is byte-identical to the source and no file is added to the skill directory

### Requirement: Each registered skill SHALL have exactly one runbook note, identified by its slug
A registered skill's runbook note SHALL have slug `skill-<skill-name>` (basename `<luhmann>.<date>.skill-<skill-name>.md`, with a normal Luhmann id and date) and SHALL carry `skill_hash`, the SHA-256 of the `SKILL.md` bytes its body was last copied from. Registration SHALL locate a skill's note by that basename suffix on a runbook note carrying `skill_hash`, and SHALL treat more than one match as an error naming the notes.

#### Scenario: Note located by slug
- **WHEN** the vault contains `1049.2026-09-21.skill-curate.md` of type runbook with a `skill_hash` field
- **THEN** registration treats it as the `curate` skill's note

#### Scenario: Duplicate notes are an error
- **WHEN** two runbook notes with `skill_hash` both end in `.skill-curate.md`
- **THEN** registration reports an error naming both and makes no change for `curate`

### Requirement: Registration SHALL offer, not act, and remember declines by hash
For each shipped skill, registration SHALL compare the skill with its note and offer: to register it when no note exists; to refresh the note when its `skill_hash` differs from the skill's current hash; and, for a note whose skill is no longer shipped, to remove it. No offer SHALL be made when the hashes match or when the skill's current hash is recorded as declined. Declining an offer SHALL record the skill's current hash (or, for a removal offer, the note's `skill_hash`) against the skill name in the vault-root file `skill-registrations.json`, and SHALL change nothing else.

#### Scenario: New skill is offered once
- **WHEN** a shipped skill has no note, the user declines registration, and registration runs again with the skill unchanged
- **THEN** the second run makes no offer for that skill

#### Scenario: A changed skill is offered again after a decline
- **WHEN** the user declined a skill's refresh at hash A and the skill's text changes to hash B
- **THEN** the next run offers the refresh once for hash B

#### Scenario: Matching hash makes no offer
- **WHEN** a skill's note has `skill_hash` equal to the skill's current hash
- **THEN** registration makes no offer and writes nothing for that skill

### Requirement: Accepting registration SHALL create a pending note without runbook fields
Accepting a registration offer SHALL create a runbook note through the normal capture path (fresh Luhmann id, slug `skill-<name>`, embed on write) whose body is the skill's `SKILL.md` preceded by a one-line preamble naming the skill it mirrors, carrying `skill_hash` and `pending: true`, and carrying no `situation`, `triggers`, `done_when`, or `red_flags`.

#### Scenario: Accepted registration
- **WHEN** the user accepts registration of `curate`
- **THEN** a runbook note ending in `.skill-curate.md` exists with the skill's text, `skill_hash`, `pending: true`, no runbook fields, and a `.vec.json` sidecar

### Requirement: Accepting a refresh SHALL replace the body, keep the fields, and mark the note pending
Accepting a refresh offer SHALL replace the note's body with the current `SKILL.md` (preamble included), set `skill_hash` to the current hash, preserve `situation`, `triggers`, `done_when`, `red_flags`, `created`, and the basename, rebuild the sidecar, and set `pending: true` so curation re-checks the fields against the new text.

#### Scenario: Accepted refresh
- **WHEN** the user accepts a refresh of a note whose skill changed
- **THEN** the note has the new body and hash, its authored fields and basename are unchanged, and it carries `pending: true`

### Requirement: Accepting a removal SHALL remove the note and its sidecar
Accepting a removal offer SHALL remove the skill note and its `.vec.json` sidecar. Registration SHALL NOT remove any note without an accepted offer.

#### Scenario: Skill no longer shipped
- **WHEN** a skill note exists for a skill absent from the shipped set and the user accepts removal
- **THEN** the note and its sidecar are gone

### Requirement: Registration SHALL never prompt or write without a terminal
When stdin is not a terminal and no `--accept`/`--decline` answer covers an offer, registration SHALL NOT prompt, SHALL NOT write any note, and SHALL NOT record a decline for that offer; it SHALL print one line naming the skills with outstanding offers and the `engram register-skills` command that answers them.

#### Scenario: Agent runs update through a shell
- **WHEN** `engram update` runs with stdin not a terminal and one skill has no note
- **THEN** nothing is written, `skill-registrations.json` is unchanged, and the output names the skill and the answering command

### Requirement: Registration SHALL be invocable standalone with explicit answers
`engram register-skills` SHALL run the same comparison as update. `--accept <name>` and `--decline <name>` (repeatable) SHALL answer that skill's current offer without prompting; `--dry-run` SHALL list every offer and write nothing.

#### Scenario: Agent relays the user's answer
- **WHEN** `engram register-skills --accept curate --decline route` runs non-interactively
- **THEN** curate's offer is carried out, route's current hash is recorded as declined, and other offers are only reported

#### Scenario: Dry run
- **WHEN** `engram register-skills --dry-run` runs
- **THEN** it prints each offer it would make and writes nothing

### Requirement: An existing runbook note SHALL be adoptable as a skill's note
`engram register-skills --adopt <name>=<note-ref>` SHALL rename the referenced runbook note to slug `skill-<name>`, keeping its Luhmann id and date and rewriting every inbound wikilink (with its sidecar); replace its body with the current `SKILL.md` and preamble; set `skill_hash`; and preserve its runbook fields. The adopted note SHALL NOT be marked pending.

#### Scenario: Adopting a previously promoted note
- **WHEN** `engram register-skills --adopt curate=1049` runs
- **THEN** note 1049 is renamed to `1049.2026-09-21.skill-curate.md`, links to its old basename now point to the new one, its body is the curate skill, its fields are unchanged, and it has `skill_hash` and no `pending` marker
