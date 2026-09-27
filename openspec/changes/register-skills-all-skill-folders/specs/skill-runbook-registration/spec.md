## RENAMED Requirements

- FROM: `### Requirement: Each registered skill SHALL have exactly one runbook note, identified by its slug`
- TO: `### Requirement: Each registered skill SHALL have exactly one runbook note, identified by its skill key`

## ADDED Requirements

### Requirement: Registration SHALL scan the default skill and command source set
Registration SHALL compare against the skills, commands, and Pi prompt templates found in one default source set, resolved identically for `engram register-skills` and `engram update`. The set SHALL consist of the following sources:

- Claude Code user skills: `~/.claude/skills/<n>/SKILL.md`, from immediate children that are directories or symlinks resolving to directories. Root files, `synced`, dangling symlinks, and directories without `SKILL.md` are ignored.
- Claude Code user commands: `~/.claude/commands/**/*.md`, named by relative path with `/` replaced by `:`.
- claude.ai-synced skills: `<bucket>/<n>/SKILL.md` for each directory bucket under `~/.claude/skills/synced/` whose `manifest.json` parses.
- Pi user skills in `~/.pi/agent/skills/`: recursively discovered skill directories plus root `.md` files.
- Skills in `~/.agents/skills/`: recursive, with root `.md` files ignored.
- Pi prompt templates: `~/.pi/agent/prompts/*.md` (flat, named by file stem).
- The skill and prompt paths and packages configured in Pi's global `settings.json`: `skills` and `prompts` entries, and `packages` entries resolved to `npm:`, `git:` or local package roots per Pi's documented rules. For a string-form package entry whose `package.json` has any `pi` key, only the manifest's `pi.skills`/`pi.prompts` entries SHALL load. The convention directories `skills/` and `prompts/` SHALL be used only when `package.json` has no `pi` key. For an object-form entry, an explicit `skills`/`prompts` pattern list SHALL filter the package (`[]` switches every file of that type off), and an omitted key SHALL fall back, for that type only, to the manifest's entry if present, else the convention directory. A settings or package skill directory SHALL be searched recursively and SHALL include its own root `.md` files. A settings or package prompt directory SHALL be searched recursively. A file that a settings entry, a settings `!pattern`/`-path` over a default folder, or an object-form package filter switches off SHALL be kept as disabled: it SHALL NOT be offered, and it SHALL count as present for removal.
- The skills and commands of each installed Claude Code plugin that meets all of these conditions:
  - it is enabled by its `enabledPlugins` entry, or, when the entry is absent, by its `plugin.json` `defaultEnabled` (default true);
  - its scope is `user`, or it is project-scoped to the current repository;
  - its `installPath` exists.

  Its skills come from `<installPath>/skills/<n>/SKILL.md` plus any `plugin.json` `skills` paths. Its commands come from `<installPath>/commands/**/*.md`, or from `plugin.json` `commands` when that is a path or an array. No other cached version is read.
- Claude Code project skills and commands: `.claude/skills/<n>/SKILL.md` and `.claude/commands/**/*.md` in the working directory and each ancestor up to the repository top-level. Root `.md` files in `.claude/skills` are ignored.
- For projects that Pi trusts, the Pi project sources, in this precedence order: `.pi/skills/`, `.agents/skills/` in the working directory and its ancestors up to the top-level, `.pi/settings.json` skill paths, `.pi/prompts/*.md`, `.pi/settings.json` prompt paths, and `.pi/settings.json` packages. A project is trusted when the nearest saved decision in `~/.pi/agent/trust.json` for the folder or a parent says so, or, with no saved decision, when the global `defaultProjectTrust` is `always`.

Harness sources SHALL be scanned only for harnesses that `engram update` detects. A name SHALL be the directory name, file stem, or command path, never a frontmatter field. An entry whose name contains `:` SHALL be skipped with a warning.

#### Scenario: Symlinked user skill is found
- **WHEN** `~/.claude/skills/route` is a symlink to `~/.claude/engram/skills/route` containing `SKILL.md`
- **THEN** registration treats it as the scanned Claude user skill `claude:route` whose source is the resolved `~/.claude/engram/skills/route/SKILL.md`

#### Scenario: Dangling symlink is skipped
- **WHEN** `~/.claude/skills/qa` is a symlink whose target does not exist
- **THEN** registration makes no offer for it and does not fail

#### Scenario: User command in a subdirectory
- **WHEN** `~/.claude/commands/opsx/apply.md` exists
- **THEN** it is scanned as a command named `opsx:apply`

#### Scenario: Synced skills come from manifest buckets
- **WHEN** `~/.claude/skills/synced/` holds one bucket directory with a parseable `manifest.json` and skill `pdf/SKILL.md`, plus a stray file `.bucket-x`
- **THEN** `pdf` is scanned as a synced skill and the stray file is ignored

#### Scenario: Pi prompt template
- **WHEN** `~/.pi/agent/prompts/review.md` exists
- **THEN** it is scanned as a Pi prompt template named `review`

#### Scenario: Pi settings skill path
- **WHEN** Pi's global `settings.json` has `"skills": ["~/extra-skills"]` and `~/extra-skills/fmt/SKILL.md` exists
- **THEN** `fmt` is scanned as a Pi settings skill

#### Scenario: npm package skills from its manifest
- **WHEN** `settings.json` lists `"npm:pi-intercom"`, and `~/.pi/agent/npm/node_modules/pi-intercom/package.json` has `"pi": {"skills": ["./skills"]}` with `skills/pi-intercom/SKILL.md`
- **THEN** `pi-intercom` is scanned as a skill of package `pi-intercom`

#### Scenario: String-form package with a pi key ignores convention dirs
- **WHEN** `settings.json` lists `"npm:x"`, and `x`'s `package.json` has `"pi": {"extensions": ["./index.ts"]}` and the package also has `skills/fmt/SKILL.md`
- **THEN** no skill of package `x` is scanned

#### Scenario: Object-form omitted type falls back per type
- **WHEN** `settings.json` lists `{"source": "npm:x", "extensions": []}` for the same package
- **THEN** `fmt` is scanned from `skills/`, because the object form falls back to the convention directory for the omitted `skills` type

#### Scenario: Object-form empty skills filter
- **WHEN** `settings.json` lists `{"source": "npm:pi-intercom", "skills": []}`
- **THEN** no skill of that package is offered, and its `pi-intercom` skill is kept as disabled, present for removal

#### Scenario: Pi root markdown skill
- **WHEN** `~/.pi/agent/skills/notes.md` is a root file
- **THEN** it is scanned as Pi user skill `notes`

#### Scenario: Agents skills in an ancestor
- **WHEN** a trusted project has `.agents/skills/lint/SKILL.md` at its top-level and registration runs from a subdirectory of it
- **THEN** `lint` is scanned as a project agents skill

#### Scenario: Trust fallback to defaultProjectTrust always
- **WHEN** the working directory has `.pi/skills/foo/SKILL.md`, `trust.json` has no decision for it or any parent, and the global `settings.json` has `"defaultProjectTrust": "always"`
- **THEN** `foo` is scanned as a Pi project skill

#### Scenario: Untrusted Pi project is not scanned
- **WHEN** the working directory has `.pi/skills/foo/SKILL.md` and `trust.json` has no trusting decision for it or any parent, and `defaultProjectTrust` is not `always`
- **THEN** no Pi project skill is scanned

#### Scenario: Configured Pi package with a missing path contributes nothing
- **WHEN** Pi's `settings.json` lists a local package path that does not exist
- **THEN** no skill is scanned from it and registration does not fail

#### Scenario: Claude project sources from a subdirectory
- **WHEN** registration runs from `internal/` inside a repository whose top-level has `.claude/skills/openspec-propose/SKILL.md` and `.claude/commands/commit.md`
- **THEN** both are scanned, exactly as when running from the top-level

#### Scenario: Root markdown in the project skills dir is ignored
- **WHEN** a repository's `.claude/skills/` holds root files `commit.md` and `engram-go-conventions.md`
- **THEN** neither is scanned as a skill, while `.claude/commands/commit.md` is scanned as a command

#### Scenario: plugin.json commands replaces the commands directory
- **WHEN** an enabled plugin's `plugin.json` has `"commands": ["./commands/setup.md"]` and its `commands/` directory also holds `configure.md`
- **THEN** only `setup` is scanned as that plugin's command

#### Scenario: Disabled plugin contributes nothing
- **WHEN** a plugin is present in `installed_plugins.json` with `enabledPlugins` set to `false`
- **THEN** none of its skills or commands are scanned

#### Scenario: Plugin without an enabledPlugins entry or install path
- **WHEN** a plugin appears in `installed_plugins.json` with no `enabledPlugins` entry and a missing `installPath`
- **THEN** none of its skills or commands are scanned

#### Scenario: Only the installed plugin version is read
- **WHEN** the plugin cache holds several versions of a plugin and `installed_plugins.json` names one `installPath`
- **THEN** only that `installPath`'s skills and commands are scanned

### Requirement: Each scanned skill or command SHALL have a source-qualified key
Each scanned entry SHALL have a key determined by the source folder it was found in. Every key SHALL begin with a source qualifier; no key SHALL be bare. A skill that engram installs SHALL be keyed like any other entry of the folder it is found in:

| Source | Key |
| --- | --- |
| Claude Code user skill | `claude:<n>` |
| Claude Code user command | `claude:cmd:<name>` |
| claude.ai-synced skill | `anthropic-skills:<n>` |
| Pi user skill | `pi:<n>` |
| `~/.agents/skills` skill | `agents:<n>` |
| Pi settings skill | `pi-settings:<n>` |
| Pi package skill | `pi-pkg:<pkg-id>:<n>` |
| Pi prompt template | `pi-prompt:<n>` (global dir), `pi-settings:pi-prompt:<n>`, `pi-pkg:<pkg-id>:pi-prompt:<n>` |
| plugin skill | `<plugin>:<n>` |
| plugin command | `<plugin>:cmd:<name>` |
| Claude Code project skill | `project:<r>:<n>` |
| Claude Code project command | `project:<r>:cmd:<name>` |
| Pi and agents project entries | `project:<r>:pi:<n>`, `project:<r>:agents:<n>`, `project:<r>:pi-prompt:<n>`, `project:<r>:pi-settings:<n>`, `project:<r>:pi-pkg:<pkg-id>:<n>` (prompts under settings/packages add `pi-prompt:` before `<n>`) |

The components are defined as follows:

- `<plugin>` is the part before `@` in the manifest key, and is never qualified by marketplace.
- `<pkg-id>` is the npm name, git `<host>/<path>`, or `~`-relative local path.
- `<r>` is the origin remote's host followed by its path, fully normalized: lowercased, with userinfo and any port removed, and without a trailing `.git` or `/` (e.g. `github.com/toejough/engram`). Without an origin, it is `local/` followed by the basename of the parent of the absolute `git rev-parse --git-common-dir`.
- A working directory outside any git repository SHALL contribute no project entries.
- An enabled plugin named `claude`, `pi`, `agents`, `project`, `anthropic-skills`, `pi-settings`, `pi-pkg`, or `pi-prompt` SHALL be skipped with a warning. The reserved-name check SHALL apply only after enablement is decided: a disabled plugin with a reserved name SHALL contribute nothing and print no warning.

The note slug SHALL be `skill-` followed by the key lowercased, with every run of characters outside `[a-z0-9]` replaced by `-`, and with leading and trailing `-` trimmed.

#### Scenario: Plugin skill key and slug
- **WHEN** the enabled plugin `superpowers@claude-plugins-official` ships `skills/brainstorming/SKILL.md`
- **THEN** its key is `superpowers:brainstorming` and an accepted registration creates a note whose basename ends in `.skill-superpowers-brainstorming.md`

#### Scenario: A plugin's skill and command never collide
- **WHEN** the enabled plugin `commit@skills` ships both `skills/commit/SKILL.md` and `commands/commit.md`
- **THEN** their keys are `commit:commit` and `commit:cmd:commit`

#### Scenario: Worktree resolves to owner and repository
- **WHEN** registration runs from a linked worktree directory named `runbook-vs-skill` whose origin remote is `ssh://git@github.com/toejough/engram.git`
- **THEN** its project skills' keys begin with `project:github.com/toejough/engram:` and their slugs with `skill-project-github-com-toejough-engram-`

#### Scenario: Two same-named projects stay distinct
- **WHEN** two repositories are both named `engram`, with origins `ssh://git@GitHub.com:22/toejough/engram.git` and `https://gitlab.com/toejough/engram`, and each has a project skill `deploy`
- **THEN** their keys are `project:github.com/toejough/engram:deploy` and `project:gitlab.com/toejough/engram:deploy`

#### Scenario: Engram-installed skills are keyed by their folder
- **WHEN** `~/.claude/skills/route` links to `~/.claude/engram/skills/route` and `~/.pi/agent/skills/route` links to `~/.pi/agent/engram/skills/route`
- **THEN** the entries' keys are `claude:route` and `pi:route`, and no key is bare

#### Scenario: Claude user command key
- **WHEN** `~/.claude/commands/audit.md` is scanned
- **THEN** its key is `claude:cmd:audit` and its note slug is `skill-claude-cmd-audit`

#### Scenario: A plugin named claude is reserved
- **WHEN** an enabled plugin `claude@m` is installed
- **THEN** registration skips it with a warning and makes no offer for any `claude:` key from it

#### Scenario: A disabled reserved-name plugin is silent
- **WHEN** a plugin `pi@m` is installed with `enabledPlugins` set to `false`
- **THEN** registration makes no offer for it and prints no warning

#### Scenario: A plugin named cmd is an ordinary plugin
- **WHEN** an enabled plugin `cmd@m` ships `skills/x/SKILL.md`
- **THEN** its key is `cmd:x`

#### Scenario: Prompt template key
- **WHEN** `~/.pi/agent/prompts/review.md` is scanned
- **THEN** its key is `pi-prompt:review`

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

## MODIFIED Requirements

### Requirement: Skill files SHALL carry no engram-specific metadata
Registration SHALL NOT read, require, or write any frontmatter field in a skill's `SKILL.md`, a command's `.md` file, or a Pi prompt template, other than using the file's bytes as the note body and hash input. It SHALL NOT require any additional file in the skill directory, command directory, or prompt directory.

#### Scenario: Skill deploys unchanged
- **WHEN** a skill whose `SKILL.md` frontmatter has only `name` and `description` is deployed and registered
- **THEN** the deployed `SKILL.md` is byte-identical to the source and no file is added to the skill directory

#### Scenario: Command and prompt files are untouched
- **WHEN** a command `.md` file with `description`/`argument-hint` frontmatter and a Pi prompt template are registered
- **THEN** both files are byte-identical afterward, no file is added beside them, and their keys do not depend on any frontmatter field


### Requirement: Each registered skill SHALL have exactly one runbook note, identified by its skill key
A registered skill's or command's runbook note SHALL meet these conditions:

- It SHALL have the slug derived from its key, giving basename `<luhmann>.<date>.<slug>.md` with a normal Luhmann id and date.
- It SHALL carry `skill_hash`, the SHA-256 of the bytes its body was last copied from.
- When it is created, adopted, or refreshed by this version, it SHALL also carry `skill_key` (its key) and `skill_source` (the resolved source path it was last copied from, home-relative with `~`).

Registration SHALL identify a key's note as the runbook note carrying `skill_hash` whose `skill_key` equals the key. A note's slug SHALL never identify it. A runbook note without `skill_hash`, or with `skill_hash` but no `skill_key`, SHALL NOT be a skill note: it SHALL NOT be matched to any key, SHALL NOT make any entry an alias, and SHALL NOT be offered for removal. A note whose `skill_key` is not a key any source can produce (its first segment names no source family, e.g. a hand-edited `route` or `engram:route` with no installed `engram` plugin) SHALL never be matched to a scanned entry and SHALL never be offered for removal. More than one match for a key SHALL be an error naming the notes.

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
- rebuild the sidecar;
- set `pending: true`, so curation re-checks the fields against the new text.

#### Scenario: Accepted refresh
- **WHEN** the user accepts a refresh of a note whose skill changed
- **THEN** the note has the new body and hash, its authored fields and basename are unchanged, and it carries `pending: true`

#### Scenario: Refresh follows a plugin version bump
- **WHEN** a plugin skill's bytes change with a new `installPath` version and the refresh is accepted
- **THEN** `skill_source` and the preamble name the new version's path

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
- preserve its runbook fields.

`<key>` SHALL name a scanned skill or command. The adopted note SHALL NOT be marked pending.

#### Scenario: Adopting a previously promoted note
- **WHEN** `engram register-skills --adopt claude:curate=1049` runs and note 1049 is `1049.2026-09-21.skill-curate.md` with no `skill_key`
- **THEN** note 1049 is renamed to `1049.2026-09-21.skill-claude-curate.md`, links to its old basename now point to the new one, its body is the curate skill preceded by a preamble naming its `skill_source`, its runbook fields are unchanged, and it has `skill_hash`, `skill_key: claude:curate`, `skill_source`, and no `pending` marker
