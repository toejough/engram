## ADDED Requirements

### Requirement: Registration SHALL scan the default skill source set
Registration SHALL compare against the skills found in one default source set, resolved identically for `engram register-skills` and `engram update`: Claude Code user skills (`~/.claude/skills/<n>/SKILL.md`, immediate children, where a child is a directory or a symlink resolving to a directory, and root `.md` files are ignored); claude.ai-synced skills (`~/.claude/skills/synced/<id>/<n>/SKILL.md`); Pi user skills (`~/.pi/agent/skills/`, recursively discovered skill directories plus root `.md` files) and `~/.agents/skills/` (recursive, root `.md` ignored); the skills of every Claude Code plugin that is listed in `~/.claude/plugins/installed_plugins.json`, enabled in `~/.claude/settings.json` `enabledPlugins`, and whose `installPath` exists (`<installPath>/skills/<n>/SKILL.md`, never any other cached version); and project-local skills under the working directory (`.claude/skills/`, `.pi/skills/`, and `.agents/skills/` in the working directory and its ancestors up to the git root). Harness sources SHALL be scanned only for harnesses that `engram update` detects. A skill's name SHALL be its directory name (or file stem for a root `.md` skill), never a frontmatter field. Dangling symlinks and directories without `SKILL.md` SHALL be skipped without error.

#### Scenario: Symlinked user skill is found
- **WHEN** `~/.claude/skills/route` is a symlink to `~/.claude/engram/skills/route` containing `SKILL.md`
- **THEN** registration treats `route` as a scanned skill whose source is the resolved `~/.claude/engram/skills/route/SKILL.md`

#### Scenario: Dangling symlink is skipped
- **WHEN** `~/.claude/skills/qa` is a symlink whose target does not exist
- **THEN** registration makes no offer for it and does not fail

#### Scenario: Disabled plugin contributes nothing
- **WHEN** a plugin is present in `installed_plugins.json` with `enabledPlugins` set to `false`
- **THEN** none of its skills are scanned

#### Scenario: Only the installed plugin version is read
- **WHEN** the plugin cache holds several versions of a plugin and `installed_plugins.json` names one `installPath`
- **THEN** only that `installPath`'s skills are scanned

### Requirement: Each scanned skill SHALL have a source-qualified skill key
Each scanned skill SHALL have a skill key determined by its source: `<n>` for Claude Code user skills and `--skills-dir` directories, `anthropic-skills:<n>` for synced skills, `pi:<n>` for Pi user skills, `agents:<n>` for `~/.agents/skills`, `<plugin>:<n>` for a plugin's skills (where `<plugin>` precedes `@` in the manifest key), and `project:<p>:<n>`, `project:<p>:pi:<n>`, `project:<p>:agents:<n>` for project-local skills, where `<p>` is the project name from the origin remote's last path segment (without `.git`), else the git top-level basename, else the working-directory basename. A plugin whose name is `pi`, `agents`, `project`, or `anthropic-skills` SHALL be skipped with a warning. The note slug SHALL be `skill-` followed by the key lowercased with every run of characters outside `[a-z0-9]` replaced by `-` and leading/trailing `-` trimmed.

#### Scenario: Plugin skill key and slug
- **WHEN** the enabled plugin `superpowers@claude-plugins-official` ships `skills/brainstorming/SKILL.md`
- **THEN** its key is `superpowers:brainstorming` and an accepted registration creates a note whose basename ends in `.skill-superpowers-brainstorming.md`

#### Scenario: Worktree resolves to the repository's project name
- **WHEN** registration runs from a linked worktree directory named `runbook-vs-skill` whose origin remote is `ssh://git@github.com/toejough/engram.git`
- **THEN** its project skills' keys begin with `project:engram:`

### Requirement: Copies of the same skill SHALL collapse to one note
Scanned entries resolving to the same `SKILL.md` path SHALL be one skill. A scanned skill whose SHA-256 equals that of a skill earlier in source precedence (Claude user, synced, Pi user, agents user, plugins, Claude project, Pi project, agents project) in the same run, or equals the `skill_hash` of any existing skill note, SHALL be an alias: it SHALL make no offer and SHALL NOT be recorded anywhere.

#### Scenario: Pi copy of an engram skill makes no offer
- **WHEN** `~/.pi/agent/skills/route` resolves to a file byte-identical to the one `~/.claude/skills/route` resolves to, and note `1036.2026-09-18.skill-route.md` carries that hash
- **THEN** registration makes no offer for `pi:route`

#### Scenario: Diverged copy is its own skill
- **WHEN** the Pi copy of `route` differs in bytes from every scanned higher-precedence copy and every note's `skill_hash`
- **THEN** registration offers to register `pi:route`

### Requirement: Removal offers SHALL be limited to scanned scopes
Every skill key SHALL belong to one scope: a bare key to Claude user, `anthropic-skills:` to synced, `pi:`/`agents:` to the matching user scope, `project:<p>:` to project `<p>`, and any other prefix to plugin `<plugin>`. A scope SHALL count as scanned only when: its harness was detected (user and synced scopes); `installed_plugins.json` was read and the plugin is either enabled with its `installPath` present or absent from the manifest (plugin scope); or the working directory resolves to project `<p>` and at least one of its project skill directories exists (project scope). Registration SHALL offer to remove a skill note only when its scope was scanned and no scanned skill has its key, and SHALL make no removal offer when `--skills-dir` is given.

#### Scenario: Project skill note from another directory
- **WHEN** a note has key `project:engram:openspec-propose` and registration runs from a directory that is not the engram project
- **THEN** no removal offer is made for it

#### Scenario: Update run without a project
- **WHEN** `engram update` runs from a directory with no project skill directories and project skill notes exist
- **THEN** no removal offer is made for any project skill note

#### Scenario: Disabled plugin keeps its notes
- **WHEN** a note has key `hookify:writing-rules` and hookify is installed but disabled
- **THEN** no removal offer is made for it

#### Scenario: Uninstalled plugin is offered for removal
- **WHEN** a note has key `ralph-loop:loop` and `ralph-loop` is absent from a readable `installed_plugins.json`
- **THEN** registration offers to remove that note

#### Scenario: Harness not installed
- **WHEN** `~/.pi` does not exist and a note has key `pi:ping`
- **THEN** no removal offer is made for it

### Requirement: Offers SHALL be grouped by scope for answering
Offers SHALL be ordered by scope then key. When prompting interactively, registration SHALL ask once per scope holding more than one offer, with the choices accept all, decline all, review each (the per-offer prompts), and skip (record nothing); a scope holding one offer SHALL use the per-offer prompt. `--accept` and `--decline` SHALL accept a skill key or a pattern ending in `*` matching keys by prefix; an exact key SHALL take precedence over a pattern, and the same key or the same pattern named in both SHALL be refused before anything is acted on. `--dry-run` SHALL list each offer with its kind, key, and source path, grouped by scope.

#### Scenario: Decline a whole plugin
- **WHEN** `engram register-skills --decline 'superpowers:*'` runs with 15 `superpowers` register offers outstanding
- **THEN** all 15 current hashes are recorded as declined under their keys and no note is written

#### Scenario: Exact key beats pattern
- **WHEN** `engram register-skills --decline 'superpowers:*' --accept superpowers:brainstorming` runs
- **THEN** `superpowers:brainstorming` is registered and the other `superpowers` offers are declined

#### Scenario: Skip leaves the scope for next time
- **WHEN** the user answers skip for a scope at the interactive prompt
- **THEN** nothing is written or recorded for that scope's offers and the next run offers them again

### Requirement: Declines SHALL be keyed by skill key with version-1 compatibility
`skill-registrations.json` SHALL map skill keys to declined hashes under `schema_version: 2`. A version-1 file SHALL be read with its names taken as skill keys (bare keys), and the next write SHALL stamp version 2 while keeping every entry. A file with an unknown `schema_version` SHALL be an error, not an empty decline state.

#### Scenario: Version-1 decline still suppresses the offer
- **WHEN** `skill-registrations.json` is `{"schema_version":1,"declined":{"route":"<h>"}}` and the scanned `route` skill hashes to `<h>` with no note
- **THEN** no offer is made for `route`

#### Scenario: Unknown schema version fails loudly
- **WHEN** `skill-registrations.json` has `schema_version: 3`
- **THEN** registration reports an error and writes nothing

## MODIFIED Requirements

### Requirement: Each registered skill SHALL have exactly one runbook note, identified by its slug
A registered skill's runbook note SHALL have the slug derived from its skill key (basename `<luhmann>.<date>.<slug>.md`, with a normal Luhmann id and date), SHALL carry `skill_hash`, the SHA-256 of the `SKILL.md` bytes its body was last copied from, and, when created or adopted by this version, SHALL carry `skill_key` (its skill key) and `skill_source` (the resolved `SKILL.md` path it was last copied from, home-relative with `~`). Registration SHALL identify a skill's note as the runbook note carrying `skill_hash` whose `skill_key` equals the key, or — for a note with no `skill_key` — whose slug is `skill-` followed by the key; a runbook note without `skill_hash` SHALL NOT be a skill note. More than one match for a key SHALL be an error naming the notes.

#### Scenario: Note located by slug
- **WHEN** the vault contains `1049.2026-09-21.skill-curate.md` of type runbook with a `skill_hash` field
- **THEN** registration treats it as the `curate` skill's note

#### Scenario: Duplicate notes are an error
- **WHEN** two runbook notes with `skill_hash` both end in `.skill-curate.md`
- **THEN** registration reports an error naming both and makes no change for `curate`

#### Scenario: Note located by skill_key
- **WHEN** a runbook note carries `skill_hash` and `skill_key: superpowers:brainstorming`
- **THEN** registration treats it as the note for key `superpowers:brainstorming`

#### Scenario: Legacy notes keep their identity
- **WHEN** the vault holds notes 1036, 1045, 1049, 1053, 1067, and 1068 with `skill-<name>` slugs, no `skill_key`, and `skill_hash` equal to the scanned skills' hashes
- **THEN** registration makes no offer for them and does not rewrite them

#### Scenario: A skill-prefixed slug without skill_hash is not a skill note
- **WHEN** a runbook note's slug is `skill-edits-validated-by-baseline-pressure-tests` and it has no `skill_hash`
- **THEN** registration neither matches it to a skill nor offers to remove it

### Requirement: Registration SHALL offer, not act, and remember declines by hash
For each scanned skill that is not an alias, registration SHALL compare the skill with its note and offer: to register it when no note exists; to refresh the note when its `skill_hash` differs from the skill's current hash; and, for a note whose key has no scanned skill within a scanned scope, to remove it. No offer SHALL be made when the hashes match or when the key's current hash is recorded as declined. Declining an offer SHALL record the skill's current hash (or, for a removal offer, the note's `skill_hash`) against the skill key in the vault-root file `skill-registrations.json`, and SHALL change nothing else.

#### Scenario: New skill is offered once
- **WHEN** a scanned skill has no note, the user declines registration, and registration runs again with the skill unchanged
- **THEN** the second run makes no offer for that skill

#### Scenario: A changed skill is offered again after a decline
- **WHEN** the user declined a skill's refresh at hash A and the skill's text changes to hash B
- **THEN** the next run offers the refresh once for hash B

#### Scenario: Matching hash makes no offer
- **WHEN** a skill's note has `skill_hash` equal to the skill's current hash
- **THEN** registration makes no offer and writes nothing for that skill

### Requirement: Accepting registration SHALL create a pending note without runbook fields
Accepting a registration offer SHALL create a runbook note through the normal capture path (fresh Luhmann id, slug derived from the skill key, embed on write) whose body is the skill's `SKILL.md` preceded by a one-line preamble naming the skill file it mirrors, carrying `skill_hash`, `skill_key`, `skill_source`, and `pending: true`, and carrying no `situation`, `triggers`, `done_when`, or `red_flags`. The preamble SHALL name `agent-instructions/skills/<n>/SKILL.md` when the resolved source lies inside an engram-owned root, and the home-relative resolved source path otherwise.

#### Scenario: Accepted registration
- **WHEN** the user accepts registration of `curate`
- **THEN** a runbook note ending in `.skill-curate.md` exists with the skill's text, `skill_hash`, `pending: true`, no runbook fields, and a `.vec.json` sidecar

#### Scenario: Plugin skill preamble names its real file
- **WHEN** the user accepts registration of `superpowers:brainstorming` whose resolved source is `~/.claude/plugins/cache/claude-plugins-official/superpowers/6.4.1/skills/brainstorming/SKILL.md`
- **THEN** the note carries `skill_key: superpowers:brainstorming`, that `skill_source`, and a preamble naming that path

### Requirement: Registration SHALL never prompt or write without a terminal
When stdin is not a terminal and no `--accept`/`--decline` answer covers an offer, registration SHALL NOT prompt, SHALL NOT write any note, and SHALL NOT record a decline for that offer; it SHALL print one line giving the number of outstanding offers per scope and the `engram register-skills` command that answers them.

#### Scenario: Agent runs update through a shell
- **WHEN** `engram update` runs with stdin not a terminal and one skill has no note
- **THEN** nothing is written, `skill-registrations.json` is unchanged, and the output names the skill's scope with its count and the answering command

### Requirement: Registration SHALL be invocable standalone with explicit answers
`engram register-skills` SHALL run the same comparison over the same default source set as update. `--accept <key-or-pattern>` and `--decline <key-or-pattern>` (repeatable) SHALL answer the matching offers without prompting; `--dry-run` SHALL list every offer and write nothing. `--skills-dir <dir>` (repeatable) SHALL replace the default source set with the listed directories, each scanned with Claude Code user-skill rules and bare keys, and SHALL suppress removal offers.

#### Scenario: Agent relays the user's answer
- **WHEN** `engram register-skills --accept curate --decline route` runs non-interactively
- **THEN** curate's offer is carried out, route's current hash is recorded as declined, and other offers are only reported

#### Scenario: Dry run
- **WHEN** `engram register-skills --dry-run` runs
- **THEN** it prints each offer it would make and writes nothing

#### Scenario: Skills-dir override never offers removal
- **WHEN** `engram register-skills --skills-dir agent-instructions/skills --dry-run` runs against a vault holding skill notes for keys absent from that directory
- **THEN** no removal offer is listed

### Requirement: An existing runbook note SHALL be adoptable as a skill's note
`engram register-skills --adopt <key>=<note-ref>` SHALL rename the referenced runbook note to the slug derived from `<key>`, keeping its Luhmann id and date and rewriting every inbound wikilink (with its sidecar); replace its body with the current `SKILL.md` and preamble; set `skill_hash`, `skill_key`, and `skill_source`; and preserve its runbook fields. `<key>` SHALL name a scanned skill. The adopted note SHALL NOT be marked pending.

#### Scenario: Adopting a previously promoted note
- **WHEN** `engram register-skills --adopt curate=1049` runs
- **THEN** note 1049 is renamed to `1049.2026-09-21.skill-curate.md`, links to its old basename now point to the new one, its body is the curate skill, its fields are unchanged, and it has `skill_hash` and no `pending` marker
