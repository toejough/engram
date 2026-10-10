## Purpose

The product behavior of registration is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a12-added-skill-runbook-registration` (792a12a to 792a12v). The source scan and the source-qualified keys are in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b30-added-skill-source-discovery`. The requirements below wait for skill-registration-ledger (792c7k3): the conflict rules, the offers, the declines ledger, the answers and the note actions. Why: vault notes 1054 (reversing 1031; the four 2026-09 delete-conversions broke distribution for every install but Joe's) and 1066a (registration covers every default folder, not just engram's six). Validation: `dev/eval/LEDGER.md#register-skills-all-skill-folders` (mechanism unit-tested; real-binary offer counts verified), superseding `#register-skills-as-runbooks`.
## Requirements
### Requirement: Each registered skill SHALL have exactly one runbook note, identified by its skill key
A registered skill's or command's runbook note SHALL meet these conditions:

- It SHALL have the slug derived from its key, giving basename `<luhmann>.<date>.<slug>.md` with a normal Luhmann id and date.
- It SHALL carry `skill_hash`, the SHA-256 of the bytes its body was last copied from.
- When it is created, adopted, or refreshed by this version, it SHALL also carry `skill_key` (its key) and `skill_source` (the resolved source path it was last copied from, home-relative with `~`).

Registration SHALL identify a key's note as the runbook note carrying `skill_hash` whose `skill_key` equals the key. A note's slug SHALL never identify it. A runbook note without `skill_hash`, or with `skill_hash` but no `skill_key`, SHALL NOT be a skill note: it SHALL NOT be matched to any key, SHALL NOT make any entry an alias, and SHALL NOT be offered for removal. A note whose `skill_key` is unrecognized SHALL never be matched to a scanned entry and SHALL never be offered for removal. A key is unrecognized only when it has no `:`, has an empty segment, or starts with a reserved source prefix (`claude`, `pi`, `agents`, `project`, `anthropic-skills`, `pi-settings`, `pi-pkg`, `pi-prompt`) whose remainder lacks that source's key shape. Any other `<x>:<name>` key SHALL be treated as plugin `<x>`'s key under the plugin removal rule. More than one match for a key SHALL be an error naming the notes.

#### Scenario: Note located by slug
- **WHEN** the vault contains `1049.2026-09-21.skill-curate.md` of type runbook with a `skill_hash` field and no `skill_key`
- **THEN** registration does not treat it as the note of `claude:curate` or of any other key, because a slug alone never locates a skill note

#### Scenario: Duplicate notes are an error
- **WHEN** two runbook notes with `skill_hash` both carry `skill_key: claude:curate`
- **THEN** registration reports an error naming both and makes no change for `claude:curate`

#### Scenario: Note located by skill_key
- **WHEN** a runbook note carries `skill_hash` and `skill_key: superpowers:brainstorming`
- **THEN** registration treats it as the note for key `superpowers:brainstorming`

#### Scenario: Unkeyed legacy notes are ignored until adopted
- **WHEN** the vault holds notes 1036, 1045, 1049, 1053, 1067, and 1068 with `skill-<name>` slugs, no `skill_key`, and `skill_hash` equal to the scanned skills' hashes
- **THEN** registration does not rewrite them and makes no refresh or removal offer for them, and it offers to register `claude:<name>` for each of the six skills

#### Scenario: An unrecognized skill_key is never matched or removed
- **WHEN** a runbook note carries `skill_hash` and `skill_key: route`, and every source root was read
- **THEN** registration matches it to no entry and offers no removal of it

#### Scenario: A skill-prefixed slug without skill_hash is not a skill note
- **WHEN** a runbook note's slug is `skill-edits-validated-by-baseline-pressure-tests` and it has no `skill_hash`
- **THEN** registration neither matches it to a skill nor offers to remove it

### Requirement: Registration SHALL offer, not act, and remember declines by hash
For each scanned skill or command that is not an alias and has no key conflict, registration SHALL compare it with its note and offer:

- to register it, when no note exists;
- to refresh the note, when its `skill_hash` differs from the current hash;
- to remove a note, when the note's root was scanned and no scanned entry has its key.

No offer SHALL be made when the hashes match, or when the key's current hash is recorded as declined.

Declining an offer SHALL record the current hash against the key in the vault-root file `skill-registrations.json`, and SHALL change nothing else. For a removal offer, the recorded hash is the note's `skill_hash`.

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
Accepting a registration offer SHALL create a runbook note through the normal capture path: a fresh Luhmann id, the slug derived from the key, and embedding on write.

- The body SHALL be the skill's `SKILL.md` or the command's `.md` file, preceded by the one-line preamble ``> Mirrors skill `<skill_source>`.``
- The note SHALL carry `skill_hash`, `skill_key`, `skill_source`, and `pending: true`.
- It SHALL carry no `situation`, `triggers`, `done_when`, or `red_flags`.

The preamble SHALL name the home-relative resolved source path (`skill_source`) with the same wording for every note, and SHALL NOT tell the reader where to edit the procedure.

#### Scenario: Accepted registration
- **WHEN** the user accepts registration of `claude:curate`, whose resolved source is `~/.claude/engram/skills/curate/SKILL.md`
- **THEN** a runbook note ending in `.skill-claude-curate.md`, whose preamble names `~/.claude/engram/skills/curate/SKILL.md`, exists with the skill's text, `skill_hash`, `pending: true`, no runbook fields, and a `.vec.json` sidecar

#### Scenario: Plugin skill preamble names its real file
- **WHEN** the user accepts registration of `superpowers:brainstorming` whose resolved source is `~/.claude/plugins/cache/claude-plugins-official/superpowers/6.4.1/skills/brainstorming/SKILL.md`
- **THEN** the note carries `skill_key: superpowers:brainstorming`, that `skill_source`, and a preamble naming that path

#### Scenario: Command registration
- **WHEN** the user accepts registration of project command `project:github.com/toejough/engram:cmd:opsx:apply`
- **THEN** the note's body is `.claude/commands/opsx/apply.md` preceded by a preamble naming that file, and its basename ends in `.skill-project-github-com-toejough-engram-cmd-opsx-apply.md`

### Requirement: Accepting a refresh SHALL replace the body, keep the fields, and mark the note pending
Accepting a refresh offer SHALL do the following:

- replace the note's body with the current source file, preamble included;
- set `skill_hash` to the current hash;
- set `skill_source` to the current resolved source, keeping `skill_key`;
- preserve `situation`, `triggers`, `done_when`, `red_flags`, `created`, and the basename;
- preserve every other frontmatter key with its value, including keys the typed runbook note model does not define;
- rebuild the sidecar;
- set `pending: true`, so curation re-checks the fields against the new text.

A refresh of a CRLF note SHALL convert it to LF inside the same single write, as adopt does.

Refresh edits `skill_hash`, `skill_key`, `skill_source`, and `pending` as a YAML node. When a key it is about to set or delete — or that key's value, or anything beneath the value — carries a YAML anchor, refresh SHALL refuse the note untouched and name the key (`errFrontmatterAnchoredKey`), rather than drop the value and leave an alias to it dangling. Before writing, refresh SHALL decode the rewritten frontmatter again and refuse to write it if it does not decode (`errFrontmatterUndecodable`).

#### Scenario: Accepted refresh
- **WHEN** the user accepts a refresh of a note whose skill changed
- **THEN** the note has the new body and hash, its authored fields and basename are unchanged, and it carries `pending: true`

#### Scenario: Refresh follows a plugin version bump
- **WHEN** a plugin skill's bytes change with a new `installPath` version and the refresh is accepted
- **THEN** `skill_source` and the preamble name the new version's path

#### Scenario: Refresh keeps an unmodeled key
- **WHEN** a refresh is accepted for a note whose frontmatter carries a key the runbook note model does not define (e.g. `luhmann_old: "12"`)
- **THEN** the refreshed note still carries that key with the same value

#### Scenario: Refresh converts a CRLF note
- **WHEN** a refresh is accepted for a skill note whose lines end in `\r\n`
- **THEN** the refresh succeeds, inside its single write the note is converted to LF, and its authored fields are unchanged

#### Scenario: Refresh refuses an anchored key
- **WHEN** an accepted refresh would replace or delete a frontmatter key (`skill_hash`, `skill_key`, `skill_source`, or `pending`) that carries a YAML anchor, or whose value does
- **THEN** the refresh is refused, naming the key, and the note is unchanged

### Requirement: Accepting a removal SHALL remove the note and its sidecar
Accepting a removal offer SHALL remove the skill note and its `.vec.json` sidecar. Registration SHALL NOT remove any note without an accepted offer.

#### Scenario: Skill no longer shipped
- **WHEN** a skill note exists for a skill absent from the shipped set and the user accepts removal
- **THEN** the note and its sidecar are gone

### Requirement: Registration SHALL never prompt or write without a terminal
When stdin is not a terminal and no `--accept`/`--decline` answer covers an offer, registration SHALL NOT prompt, SHALL NOT write any note, and SHALL NOT record a decline for that offer. It SHALL print one line giving the total number of outstanding offers, each scope's selector with its count, and the `engram register-skills` command that answers them.

#### Scenario: Agent runs update through a shell
- **WHEN** `engram update` runs with stdin not a terminal and one skill has no note
- **THEN** nothing is written, `skill-registrations.json` is unchanged, and the output names the skill's scope selector with its count and the answering command

### Requirement: Registration SHALL be invocable standalone with explicit answers
`engram register-skills` SHALL run the same comparison over the same default source set as update.

- `--accept <key|pattern|@scope>` and `--decline <key|pattern|@scope>` (repeatable) SHALL answer the matching offers without prompting.
- `--dry-run` SHALL list every offer and write nothing.
- `--skills-dir <dir>` (repeatable) SHALL replace the default source set with the listed directories, each scanned with Claude Code user-skill rules and keyed `claude:<n>`. The run SHALL be read-only: it SHALL behave as `--dry-run`, SHALL refuse `--accept`, `--decline`, and `--adopt`, and SHALL make no removal offer.

#### Scenario: Agent relays the user's answer
- **WHEN** `engram register-skills --accept claude:curate --decline claude:route` runs non-interactively
- **THEN** `claude:curate`'s offer is carried out, `claude:route`'s current hash is recorded as declined, and other offers are only reported

#### Scenario: Dry run
- **WHEN** `engram register-skills --dry-run` runs
- **THEN** it prints each offer it would make and writes nothing

#### Scenario: Skills-dir runs are preview-only
- **WHEN** `engram register-skills --skills-dir agent-instructions/skills --accept claude:route` runs
- **THEN** it refuses with an error before scanning, and without `--accept` it lists offers keyed `claude:<n>`, writes nothing, never rewrites a note's preamble, and lists no removal offer

### Requirement: An existing runbook note SHALL be adoptable as a skill's note
`engram register-skills --adopt <key>=<note-ref>` SHALL act on the referenced runbook note as follows:

- rename it to the slug derived from `<key>`, keeping its Luhmann id and date and rewriting every inbound wikilink (with its sidecar);
- replace its body with the current source file and preamble;
- set `skill_hash`, `skill_key`, and `skill_source`;
- preserve its runbook fields and every other frontmatter key with its value, including keys the typed runbook note model does not define.

`<key>` SHALL name a scanned skill or command. The adopted note SHALL NOT be marked pending.

Adopt edits `skill_hash`, `skill_key`, `skill_source`, and `pending` as a YAML node, the same way refresh does, and checks this before it renames the note. When a key it is about to set or delete — or that key's value, or anything beneath the value — carries a YAML anchor, adopt SHALL refuse the note untouched and name the key (`errFrontmatterAnchoredKey`), before any rename. Before writing, adopt SHALL decode the rewritten frontmatter again and refuse to write it if it does not decode (`errFrontmatterUndecodable`).

#### Scenario: Adopting a previously promoted note
- **WHEN** `engram register-skills --adopt claude:curate=1049` runs and note 1049 is `1049.2026-09-21.skill-curate.md` with no `skill_key`
- **THEN** note 1049 is renamed to `1049.2026-09-21.skill-claude-curate.md`, links to its old basename now point to the new one, its body is the curate skill preceded by a preamble naming its `skill_source`, its runbook fields are unchanged, and it has `skill_hash`, `skill_key: claude:curate`, `skill_source`, and no `pending` marker

#### Scenario: Adopt keeps an unmodeled key
- **WHEN** `--adopt` runs on a runbook note whose frontmatter carries a key the runbook note model does not define
- **THEN** the adopted note still carries that key with the same value

#### Scenario: Adopt refuses an anchored key
- **WHEN** `--adopt` targets a runbook note whose `skill_hash`, `skill_key`, `skill_source`, or `pending` key carries a YAML anchor, or whose value does
- **THEN** the adopt is refused, naming the key, and the note is neither renamed nor rewritten

### Requirement: Copies of the same skill SHALL collapse to one note, and key conflicts SHALL be reported
Scanned entries resolving to the same file SHALL be one entry.

Entries with the same key SHALL collapse when byte-identical. When they differ, registration SHALL report a key conflict naming both paths, SHALL make no offer for that key, and SHALL exit with a failure status after handling every other offer. This SHALL include the same name found at two levels of a project directory chain.

A plugin name installed from more than one marketplace SHALL be a plugin conflict. Registration SHALL report it, SHALL make no offer for any of its keys, and SHALL treat its scope as not scanned.

There SHALL be no exception for any source, including skills that engram installs.

An entry whose SHA-256 equals that of an entry earlier in source precedence in the same run, or equals the `skill_hash` of any existing skill note (a note carrying `skill_key`), SHALL be an alias. An alias SHALL make no offer and SHALL NOT be recorded anywhere, but it SHALL count as present for its key.

#### Scenario: Pi copy of an engram skill makes no offer
- **WHEN** `~/.pi/agent/skills/route` and `~/.claude/skills/route` are byte-identical, and a note keyed `claude:route` carries that hash
- **THEN** registration makes no offer for `claude:route` or `pi:route`, and `pi:route` counts as present

#### Scenario: Two Pi skills with one name and different content
- **WHEN** Pi recursion finds `~/.pi/agent/skills/a/foo/SKILL.md` and `~/.pi/agent/skills/b/foo/SKILL.md` with different bytes
- **THEN** registration reports a conflict for `pi:foo` naming both paths, makes no offer for `pi:foo`, handles every other offer, and exits with a failure status

#### Scenario: Same project skill name at two levels
- **WHEN** a repository has `.claude/skills/foo/SKILL.md` at its top-level and a different `sub/.claude/skills/foo/SKILL.md`, and registration runs from `sub/`
- **THEN** registration reports a conflict for the key `project:<r>:foo` naming both paths and makes no offer for it

#### Scenario: Plugin name in two marketplaces
- **WHEN** plugins `tools@alpha` and `tools@beta` are both installed and enabled
- **THEN** registration reports a plugin conflict for `tools`, makes no offer for any `tools:` key, and offers no removal of existing `tools:` notes

#### Scenario: Diverged harness copies are separate keys
- **WHEN** `~/.claude/skills/route` and `~/.pi/agent/skills/route` differ in bytes, a note keyed `claude:route` carries the Claude copy's hash, and no note or decline covers the Pi copy's hash
- **THEN** registration makes no offer for `claude:route`, offers to register `pi:route`, and reports no conflict and no warning

#### Scenario: Two synced buckets hold the same skill
- **WHEN** two synced buckets both hold `pdf/SKILL.md` with identical bytes
- **THEN** one candidate `anthropic-skills:pdf` results and no conflict is reported

#### Scenario: Diverged plugin skill of the same name is its own skill
- **WHEN** `skill-creator@claude-plugins-official`'s `skill-creator` and the synced `skill-creator` differ in bytes
- **THEN** registration offers both `skill-creator:skill-creator` and `anthropic-skills:skill-creator`

### Requirement: Removal offers SHALL be limited to successfully read sources
A source root SHALL count as read only when its read succeeded. A read error of any kind, including not-exist, SHALL mean not read, never empty.

A skill note SHALL be eligible for removal only as follows:

- For `claude:`, `claude:cmd:`, `pi:`, `agents:`, and `pi-prompt:` keys: the one fixed user root for that form was read (`~/.claude/skills`, `~/.claude/commands`, `~/.pi/agent/skills`, `~/.agents/skills`, `~/.pi/agent/prompts` respectively), and no other root is consulted.
- For `anthropic-skills:`, `pi-settings:`, `pi-pkg:`, and `project:` keys: the specific root containing the note's recorded `skill_source` was read in this run. That root is the synced bucket whose `manifest.json` parsed, the settings entry's path, the package root, or the exact project directory.
- For plugin keys: `installed_plugins.json` and `settings.json` both parsed, the plugin has no plugin conflict, and it is either enabled with its `installPath` read, or absent from the manifest.

A note's eligibility SHALL depend only on the roots of its own key form: whether any other form's root was read SHALL NOT change it. A note whose `skill_source` lies under no read root SHALL NOT be eligible. A runbook note carrying `skill_hash` but no `skill_key` SHALL never be offered for removal. Registration SHALL offer to remove a skill note only when it is eligible and no scanned entry, alias or not, has its key. It SHALL make no removal offer when `--skills-dir` is given.

#### Scenario: Project skill note from another directory
- **WHEN** a note has key `project:github.com/toejough/engram:openspec-propose` and registration runs from a directory outside that repository
- **THEN** no removal offer is made for it

#### Scenario: Update run without a project
- **WHEN** `engram update` runs from a directory that is not in a git repository and project skill notes exist
- **THEN** no removal offer is made for any project skill note

#### Scenario: Disabled plugin keeps its notes
- **WHEN** a note has key `hookify:writing-rules` and hookify is installed but disabled
- **THEN** no removal offer is made for it

#### Scenario: Pi-disabled skill keeps its note
- **WHEN** a note has key `pi:ping`, `~/.pi/agent/skills` is read, and the global Pi `settings.json` `skills` has `"!ping"`
- **THEN** no register, refresh or removal offer is made for it

#### Scenario: Uninstalled plugin is offered for removal
- **WHEN** a note has key `ralph-loop:cmd:help` and `ralph-loop` is absent from a readable `installed_plugins.json`
- **THEN** registration offers to remove that note

#### Scenario: Unreadable user skills directory
- **WHEN** listing `~/.claude/skills` fails and `claude:` notes exist
- **THEN** no removal offer is made for them

#### Scenario: Another namespace's missing root does not block removal
- **WHEN** `~/.agents/skills` does not exist, `~/.pi/agent/skills` fails with a permission error, `~/.claude/skills` is read and lacks `c4`, and a note keyed `claude:c4` exists
- **THEN** registration offers to remove the `claude:c4` note

#### Scenario: Unkeyed legacy note is never offered for removal
- **WHEN** runbook note `1036.2026-09-18.skill-route.md` carries `skill_hash` and no `skill_key`, and every source root was read
- **THEN** no removal offer is made for it

#### Scenario: Synced manifest missing
- **WHEN** `~/.claude/skills/synced/` holds no bucket with a readable `manifest.json` and `anthropic-skills:*` notes exist
- **THEN** no removal offer is made for them

#### Scenario: Nested-only project skill is safe from the top level
- **WHEN** a note with key `project:github.com/toejough/engram:foo` was registered from `sub/`, its `skill_source` is under `sub/.claude/skills`, and registration runs from the repository top-level
- **THEN** no removal offer is made for it

#### Scenario: One failing synced bucket
- **WHEN** two synced buckets exist, bucket A's `manifest.json` parses, bucket B's does not, and a note's `skill_source` lies in bucket B
- **THEN** no removal offer is made for that note

#### Scenario: Pi settings note tied to its own entry
- **WHEN** `settings.json` has two `skills` entries, the first is unreadable, and a `pi-settings:` note's `skill_source` lies under the first
- **THEN** no removal offer is made for that note

#### Scenario: Package dropped from settings leaves an orphan
- **WHEN** a `pi-pkg:x:` note exists and `npm:x` is no longer listed in `settings.json`
- **THEN** no removal offer is made for it

#### Scenario: Alias keeps its note
- **WHEN** a note has key `pi:ping` and the Pi `ping` skill is now byte-identical to a higher-precedence skill
- **THEN** no removal offer is made for the `pi:ping` note

#### Scenario: Harness not installed
- **WHEN** `~/.pi` does not exist and a note has key `pi:ping`
- **THEN** no removal offer is made for it

### Requirement: Offers SHALL be grouped by scope for answering
Each offer SHALL belong to a scope, identified as follows:

- `claude-user` for `claude:` skill keys.
- `claude-cmd` for `claude:cmd:` keys.
- `synced`, `pi-user`, `agents-user`, and `pi-settings` for those sources.
- `pi-pkg:<pkg-id>` for a Pi package.
- `plugin:<plugin>` for a plugin.
- `project:<r>` for a project.

Offers SHALL be ordered by scope then key. A scope's display label SHALL be its answerable selector `@<scope-id>` with its count.

`--accept` and `--decline` SHALL accept one of three forms: an exact key, a pattern ending in `*` that matches keys by prefix, or a selector `@<scope-id>`. For each offer, an exact key SHALL take precedence, then the longest matching pattern, then a selector. The same key, pattern, or selector named in both flags SHALL be refused before anything is acted on.

When prompting interactively, registration SHALL ask once for each scope holding more than one register or refresh offer, with the choices accept all, decline all, review each, and skip. Skip, and end-of-input at that prompt, SHALL record nothing.

Removal offers SHALL be answered only by an exact key or an individual prompt, never by accept-all, a pattern, or a selector.

`--dry-run` SHALL list each offer with its kind, key, and source path under scope headers.

#### Scenario: Decline a whole plugin
- **WHEN** `engram register-skills --decline @plugin:superpowers` runs with 15 `superpowers` register offers outstanding
- **THEN** all 15 current hashes are recorded as declined under their keys and no note is written

#### Scenario: Longest pattern wins
- **WHEN** `engram register-skills --decline 'superpowers:*' --accept 'superpowers:writing-*'` runs
- **THEN** `superpowers:writing-plans` and `superpowers:writing-skills` are registered and the other `superpowers` offers are declined

#### Scenario: Exact key beats pattern
- **WHEN** `engram register-skills --decline 'superpowers:*' --accept superpowers:brainstorming` runs
- **THEN** `superpowers:brainstorming` is registered and the other `superpowers` offers are declined

#### Scenario: Claude user scope is nameable
- **WHEN** `engram register-skills --decline @claude-user` runs with register offers for `claude:c4` and `claude:dev`
- **THEN** both are recorded as declined

#### Scenario: Removals are never bulk-accepted
- **WHEN** `engram register-skills --accept '*'` runs with one register offer and one removal offer outstanding
- **THEN** the register offer is carried out and the removal offer is only reported

#### Scenario: Skip leaves the scope for next time
- **WHEN** the user answers skip for a scope at the interactive prompt
- **THEN** nothing is written or recorded for that scope's offers and the next run offers them again

### Requirement: Declines SHALL be keyed by skill key under schema version 2
`skill-registrations.json` SHALL map keys to declined hashes under `schema_version: 2`. A file whose `schema_version` is anything other than 2 (missing, below 2, or above 2) SHALL be an error, not an empty decline state, and registration SHALL write nothing. A missing file SHALL mean no declines.

#### Scenario: Version-1 file fails loudly
- **WHEN** `skill-registrations.json` is `{"schema_version":1,"declined":{"route":"<h>"}}`
- **THEN** registration reports an error and writes nothing

#### Scenario: Unknown schema version fails loudly
- **WHEN** `skill-registrations.json` has `schema_version: 3`
- **THEN** registration reports an error and writes nothing

