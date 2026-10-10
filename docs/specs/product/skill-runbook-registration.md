# Skill Runbook Registration - Product Spec

Skills remain the shipping form of every procedure, and their files stay exactly as the harnesses that load them expect. Engram's own skills ship from `agent-instructions/skills/` through `engram update`. Registration lets memory surface a procedure by situation and trigger: each skill, command and Pi prompt template in the folders a harness loads may have one runbook note in the vault that mirrors its text and carries the runbook fields authored on the note, so the runbook shim's follow-frame can drive it and it can flow across vault-graph edges. Engram offers to create, refresh and remove these notes, the user decides, and a declined offer is not repeated. `engram update` runs the offers after deploying ([792a12t Update runs registration after deploying the skills](#792a12t-update-runs-registration-after-deploying-the-skills)), and `engram register-skills` runs them on their own. A note mirrors a skill and does not replace it: the direction that retired skills to runbook-only form was reversed. Curating the pending notes belongs to the curate capability, and the skills' own procedures to recall, learn, route, please, curate and write-memory.

## 792a12a Skill files are never changed

Requirement: [792a12a Skill files are never changed](#792a12a-skill-files-are-never-changed) Registration MUST NOT read, require or write any frontmatter field of a skill's `SKILL.md`, a command's
`.md` file or a Pi prompt template, other than using the file's bytes as the note body and as the hash
input, and MUST NOT require or add any file beside them.

Rationale: the harness owns the schema of those files, and the runbook fields (`situation`, `triggers`,
`done_when`, `red_flags`) are authored on the note, never in the skill.

### 792a12a1 UAT Scenarios

```gherkin
Scenario: 792a12a1a A deployed skill is byte-identical
  Given a skill whose SKILL.md frontmatter has only name and description, deployed and registered
  When the user runs engram register-skills
  Then the deployed SKILL.md is byte-identical to the source and no file is added to the skill directory

Scenario: 792a12a1b Command and prompt files are untouched
  Given a command .md file with description and argument-hint frontmatter, and a Pi prompt template, both registered
  When the user runs engram register-skills
  Then both files are byte-identical afterward, no file is added beside them, and their keys do not depend on any frontmatter field
```

## 792a12b One runbook note per skill, identified by its key

Requirement: [792a12b One runbook note per skill, identified by its key](#792a12b-one-runbook-note-per-skill-identified-by-its-key) Each registered skill, command or prompt template MUST have at most one runbook note, which registration
MUST identify as the note of type runbook whose basename slug begins `skill-`, that carries a non-empty
`skill_hash`, and whose `skill_key` equals the key. A note's slug alone MUST NOT identify it. A runbook
note with `skill_hash` but no `skill_key`, one whose `skill_key` is unrecognized, and one without
`skill_hash` MUST NOT be matched to any key, MUST NOT make any entry an alias, and MUST NOT be offered
for removal. A key is unrecognized only when it has no `:`, has an empty segment, or begins with a
reserved source prefix (`claude`, `pi`, `agents`, `project`, `anthropic-skills`, `pi-settings`,
`pi-pkg`, `pi-prompt`) whose remainder lacks that source's shape; any other `<x>:<name>` is plugin
`<x>`'s key. More than one note for a key MUST be an error naming them.

Rationale: One skill is one procedure, so it gets one note. Several notes per skill were rejected because a
sub-note has no body source except a second file in the skill directory, and a rule for which skills get
several notes caused confusion in the first draft. The slug identifies the note, so no frontmatter key is
scanned across the vault, and runbook fields live on the note and not in `SKILL.md` because the harness owns
that schema.

### 792a12b1 UAT Scenarios

```gherkin
Scenario: 792a12b1a Note located by skill_key
  Given a runbook note with skill_hash and skill_key superpowers:brainstorming
  When the user runs engram register-skills
  Then registration treats it as the note for key superpowers:brainstorming

Scenario: 792a12b1b Note not located by slug
  Given the vault holds 1049.2026-09-21.skill-curate.md of type runbook with a skill_hash field and no skill_key
  When the user runs engram register-skills
  Then it is not the note of claude:curate or of any other key

Scenario: 792a12b1c Duplicate notes are an error
  Given two runbook notes with skill_hash that both carry skill_key claude:curate
  When the user runs engram register-skills
  Then registration reports an error naming both and makes no change for claude:curate

Scenario: 792a12b1d Unkeyed legacy notes are ignored until adopted
  Given notes 1036, 1045, 1049, 1053, 1067 and 1068 with skill-<name> slugs, no skill_key, and skill_hash equal to the scanned skills' hashes
  When the user runs engram register-skills
  Then none is rewritten, no refresh or removal is offered for them, and registering claude:<name> is offered for each of the six skills

Scenario: 792a12b1e An unrecognized skill_key is never matched or removed
  Given a runbook note with skill_hash and skill_key route, and every source root read
  When the user runs engram register-skills
  Then it is matched to no entry and no removal of it is offered

Scenario: 792a12b1f A skill-prefixed slug without skill_hash is not a skill note
  Given a runbook note whose slug is skill-edits-validated-by-baseline-pressure-tests and that has no skill_hash
  When the user runs engram register-skills
  Then it is neither matched to a skill nor offered for removal

Scenario: 792a12b1g A keyed note with another slug is not found
  Given a runbook note with skill_hash and skill_key claude:curate whose slug does not begin skill-
  When the user runs engram register-skills
  Then it is not the note of claude:curate, so registering claude:curate is offered
```

## 792a12c Registration offers and remembers declines by hash

Requirement: [792a12c Registration offers and remembers declines by hash](#792a12c-registration-offers-and-remembers-declines-by-hash) For each scanned skill, command or prompt template that is not an alias and has no key conflict,
registration MUST offer to register it when it has no note, and to refresh its note when the note's
`skill_hash` differs from the current hash, and MUST offer to remove a note when its root was read and
no scanned entry has its key ([792a12q Removal is offered only for successfully read sources](#792a12q-removal-is-offered-only-for-successfully-read-sources)). It MUST make no offer when the hashes match or when the key's
current hash is recorded as declined. Declining an offer MUST record the current hash against the key in
the vault-root file `skill-registrations.json` and change nothing else; for a removal offer the recorded
hash is the note's `skill_hash`.

Rationale: Declines live in one vault-root file keyed by skill key, because a declined new skill has no note
to hold the decline. A changed skill gets a new hash, so the offer returns once. Declines are per key and not
per scope, so declining all records each key.

### 792a12c1 UAT Scenarios

```gherkin
Scenario: 792a12c1a A new skill is offered once
  Given a scanned skill with no note that the user declined, unchanged since
  When the user runs engram register-skills again
  Then no offer is made for that skill

Scenario: 792a12c1b A changed skill is offered again after a decline
  Given the user declined a skill's refresh at hash A and its text now has hash B
  When the user runs engram register-skills
  Then the refresh is offered once for hash B

Scenario: 792a12c1c A matching hash makes no offer
  Given a skill whose note has skill_hash equal to the skill's current hash
  When the user runs engram register-skills
  Then no offer is made and nothing is written for that skill

Scenario: 792a12c1d A declined removal is not offered again
  Given a removal the user declined at the note's current skill_hash
  When the user runs engram register-skills
  Then no removal is offered for that note
```

## 792a12d Accepting a registration creates a pending note

Requirement: [792a12d Accepting a registration creates a pending note](#792a12d-accepting-a-registration-creates-a-pending-note) Accepting a registration offer MUST create a runbook note through the normal capture path: a fresh
top-level Luhmann ID, the slug derived from the key, a sidecar embedded on write, the `source:` line
`skill registration: <skill_source>`, `skill_hash` (the SHA-256 of the source file's bytes),
`skill_key`, `skill_source` (the resolved source path, home-relative with `~/` when under the home
directory) and `pending: true`, and none of `situation`, `triggers`, `done_when` or `red_flags`. Its
body MUST be the one-line preamble ``> Mirrors skill `<skill_source>`.``, a blank line, and the source
file's bytes verbatim. The preamble MUST use the same wording for every source and MUST NOT say where to
edit the procedure.

Rationale: the runbook fields are authored on the note afterward, by curation, because only the skill's text is
known at registration.
Creating the note pending and letting curation author the fields was chosen over drafting them through the LLM
API, which would skip the review those fields needed in every conversion. A separate create command was
rejected because it asks the user twice for one decision. Pending keeps the incomplete note out of retrieval
until it has a situation.

### 792a12d1 UAT Scenarios

```gherkin
Scenario: 792a12d1a Accepted registration
  Given the user accepts registration of claude:curate, whose resolved source is ~/.claude/engram/skills/curate/SKILL.md
  When the user runs engram register-skills
  Then a runbook note ending in .skill-claude-curate.md exists with the skill's text after a preamble naming ~/.claude/engram/skills/curate/SKILL.md, skill_hash, pending: true, no runbook fields, and a .vec.json sidecar

Scenario: 792a12d1b A plugin skill's preamble names its real file
  Given the user accepts registration of superpowers:brainstorming, whose resolved source is ~/.claude/plugins/cache/claude-plugins-official/superpowers/6.4.1/skills/brainstorming/SKILL.md
  When the user runs engram register-skills
  Then the note carries skill_key superpowers:brainstorming, that skill_source, and a preamble naming that path

Scenario: 792a12d1c A command is registered from its file
  Given the user accepts registration of project command project:github.com/toejough/engram:cmd:opsx:apply
  When the user runs engram register-skills
  Then the note's body is .claude/commands/opsx/apply.md after a preamble naming that file, and its basename ends in .skill-project-github-com-toejough-engram-cmd-opsx-apply.md
```

## 792a12e Accepting a refresh replaces the body, keeps the fields, and marks the note pending

Requirement: [792a12e Accepting a refresh replaces the body, keeps the fields, and marks the note pending](#792a12e-accepting-a-refresh-replaces-the-body-keeps-the-fields-and-marks-the-note-pending) Accepting a refresh offer MUST replace the note's body with the current source file after the preamble,
set `skill_hash` to the current hash, set `skill_source` to the current resolved source while keeping
`skill_key`, rebuild the sidecar, and set `pending: true` so that curation re-checks the fields against
the new text. It MUST preserve `situation`, `triggers`, `done_when`, `red_flags`, `created`, the
basename, and every other frontmatter key with its value, including keys the runbook note model does not
define. It MUST convert a CRLF note to LF in the same single write. It MUST refuse the note untouched,
naming the key, when a key it is about to set or delete (`skill_hash`, `skill_key`, `skill_source`,
`pending`), or that key's value, carries a YAML anchor, and MUST NOT write frontmatter that does not
decode.

### 792a12e1 UAT Scenarios

```gherkin
Scenario: 792a12e1a Accepted refresh
  Given the user accepts a refresh of a note whose skill changed
  When the user runs engram register-skills
  Then the note has the new body and hash, its authored fields and basename are unchanged, and it carries pending: true

Scenario: 792a12e1b Refresh follows a plugin version bump
  Given a plugin skill's bytes changed under a new installPath version and the refresh is accepted
  When the user runs engram register-skills
  Then skill_source and the preamble name the new version's path

Scenario: 792a12e1c Refresh keeps an unmodeled key
  Given a refresh accepted for a note whose frontmatter carries a key the runbook note model does not define, such as luhmann_old: "12"
  When the user runs engram register-skills
  Then the refreshed note still carries that key with the same value

Scenario: 792a12e1d Refresh converts a CRLF note
  Given a refresh accepted for a skill note whose lines end in \r\n
  When the user runs engram register-skills
  Then it succeeds, the note is converted to LF inside its single write, and its authored fields are unchanged

Scenario: 792a12e1e Refresh refuses an anchored key
  Given an accepted refresh that would replace or delete a skill_hash, skill_key, skill_source or pending key that carries a YAML anchor, or whose value does
  When the user runs engram register-skills
  Then the refresh is refused naming the key and the note is unchanged

Scenario: 792a12e1f Authored fields survive a refresh
  Given red_flags changed on a skill note with engram amend, then a refresh accepted
  When the user runs engram register-skills
  Then the amended red_flags are still present
```

## 792a12f Accepting a removal deletes the note and its sidecar

Requirement: [792a12f Accepting a removal deletes the note and its sidecar](#792a12f-accepting-a-removal-deletes-the-note-and-its-sidecar) Accepting a removal offer MUST delete the skill note and its `.vec.json` sidecar. Registration MUST NOT
delete any note without an accepted offer.

### 792a12f1 UAT Scenarios

```gherkin
Scenario: 792a12f1a A skill no longer shipped
  Given a skill note for a skill absent from the scanned set, and the user accepts its removal
  When the user runs engram register-skills
  Then the note and its sidecar are gone
```

## 792a12g Registration never prompts or writes without a terminal

Requirement: [792a12g Registration never prompts or writes without a terminal](#792a12g-registration-never-prompts-or-writes-without-a-terminal) When stdin is not a terminal and no `--accept` or `--decline` answer covers an offer, registration MUST
NOT prompt, MUST NOT write a note, and MUST NOT record a decline for that offer. It MUST print one line
giving the total number of outstanding offers (`offer` for one, `offers` otherwise), each scope's
selector with its count, and the commands that answer them: ``engram: <N> skill runbook offers awaiting
an answer: @<scope> <count>, … — run `engram register-skills` in a terminal, or `engram register-skills
--accept <key|prefix*|@scope>` / `--decline <key|prefix*|@scope>` ``.

### 792a12g1 UAT Scenarios

```gherkin
Scenario: 792a12g1a An agent runs update through a shell
  Given stdin is not a terminal and one scanned skill has no note
  When the user runs engram update
  Then nothing is written, skill-registrations.json is unchanged, and the output names the skill's scope selector with its count and the answering command
```

## 792a12h Registration can run on its own with explicit answers

Requirement: [792a12h Registration can run on its own with explicit answers](#792a12h-registration-can-run-on-its-own-with-explicit-answers) `engram register-skills` MUST run the same comparison over the same default source set as `engram
update`. `--accept <key|pattern|@scope>` and `--decline <key|pattern|@scope>` (each repeatable) MUST
answer the matching offers without prompting, and an answer that matches no offer MUST print `engram:
no pending offer for skill "<token>" — ignoring --accept` (or `--decline`) and continue. `--dry-run`
MUST list every offer and write nothing, and run no `--adopt`. `--vault` and `--vault-name` MUST choose
the vault and the name stamped on a new note's `vault:` field. `--skills-dir <dir>` (repeatable,
relative to the working directory) MUST replace the default set with the listed directories, each
scanned as Claude Code user skills and keyed `claude:<name>`; that run MUST be a read-only preview that
behaves as `--dry-run`, refuses `--accept`, `--decline` and `--adopt` before scanning with
`register-skills: --skills-dir is a read-only preview; it cannot be combined with --accept, --decline
or --adopt`, and makes no removal offer. The same key, pattern or selector named in both `--accept` and
`--decline` MUST be refused before anything is acted on.

### 792a12h1 UAT Scenarios

```gherkin
Scenario: 792a12h1a An agent relays the user's answer
  Given offers for claude:curate and claude:route
  When the user runs engram register-skills --accept claude:curate --decline claude:route non-interactively
  Then claude:curate's offer is carried out, claude:route's current hash is recorded as declined, and other offers are only reported

Scenario: 792a12h1b A dry run
  Given outstanding offers
  When the user runs engram register-skills --dry-run
  Then it prints each offer it would make under scope headers and writes nothing

Scenario: 792a12h1c An answer matching nothing is noted
  Given no offer for the key nosuch
  When the user runs engram register-skills --accept nosuch
  Then it prints the no-pending-offer line and carries on with the other offers

Scenario: 792a12h1d A skills-dir run is a preview
  Given a directory agent-instructions/skills holding skills
  When the user runs engram register-skills --skills-dir agent-instructions/skills
  Then it lists offers keyed claude:<name>, writes nothing, never rewrites a note's preamble, and lists no removal offer

Scenario: 792a12h1e A skills-dir run refuses answers
  Given any directory
  When the user runs engram register-skills --skills-dir agent-instructions/skills --accept claude:route
  Then it refuses with an error before scanning

Scenario: 792a12h1f One answer in both flags is refused
  Given an offer
  When the user runs engram register-skills --accept claude:route --decline claude:route
  Then it is refused before anything is acted on
```

## 792a12i An existing runbook note can be adopted as a skill's note

Requirement: [792a12i An existing runbook note can be adopted as a skill's note](#792a12i-an-existing-runbook-note-can-be-adopted-as-a-skills-note) `engram register-skills --adopt <key>=<note-ref>` MUST, for a runbook note named by Luhmann ID or
basename, rename it to the slug derived from `<key>` keeping its Luhmann ID and date and recording its
old basename in `aliases`, rewrite every inbound wikilink and move its sidecar, replace its body with
the preamble and the current source file, set `skill_hash`, `skill_key` and `skill_source`, remove any
`pending` marker, and preserve its runbook fields and every other frontmatter key with its value,
including keys the runbook note model does not define. `<key>` MUST name a scanned, enabled, unconflicted
skill, command or prompt template; a skill that Pi's settings switch off is not one. Adopting a note that already carries the key's slug MUST be a rename that
changes nothing and still refresh the body and fields. It MUST refuse, writing nothing, when the
reference names no note or more than one, when the note is not a runbook, when it is keyed to a
different skill, when the key is registered to a different note, when two `--adopt` entries name one key
or one note, when the note's basename has no Luhmann ID and date, and, before any rename, when a key it
is about to set or delete (or that key's value) carries a YAML anchor or the rewritten frontmatter would
not decode. All `--adopt` entries MUST run before the offers are computed.

### 792a12i1 UAT Scenarios

```gherkin
Scenario: 792a12i1a Adopting a previously promoted note
  Given note 1049 is 1049.2026-09-21.skill-curate.md with no skill_key
  When the user runs engram register-skills --adopt claude:curate=1049
  Then 1049 is renamed to 1049.2026-09-21.skill-claude-curate.md, links to its old basename now point to the new one, its body is the curate skill after a preamble naming its skill_source, its runbook fields are unchanged, and it has skill_hash, skill_key claude:curate, skill_source and no pending marker

Scenario: 792a12i1b Adopt keeps an unmodeled key
  Given a runbook note whose frontmatter carries a key the runbook note model does not define
  When --adopt runs on it
  Then the adopted note still carries that key with the same value

Scenario: 792a12i1c Adopt refuses an anchored key
  Given a runbook note whose skill_hash, skill_key, skill_source or pending key carries a YAML anchor, or whose value does
  When --adopt targets it
  Then the adopt is refused naming the key and the note is neither renamed nor rewritten

Scenario: 792a12i1d Adopt converts a CRLF note
  Given a runbook note whose lines end in \r\n
  When --adopt runs on it
  Then the note is converted to LF and renamed, and its links are rewritten

Scenario: 792a12i1e Adopt refuses a note that is not a runbook
  Given a fact note
  When --adopt targets it
  Then it is refused and nothing changes

Scenario: 792a12i1f Adopt refuses a note keyed to another skill
  Given a note whose skill_key is claude:route
  When --adopt claude:curate targets it
  Then it is refused and the note is not re-keyed

Scenario: 792a12i1g Adopt refuses an unscanned key
  Given a key that no scanned entry has
  When --adopt names it
  Then it is refused and nothing is written

Scenario: 792a12i1h Adopt refuses an ambiguous reference
  Given a bare Luhmann ID shared by two notes
  When --adopt names it
  Then it is refused, asking for the full basename

Scenario: 792a12i1i A dry run adopts nothing
  Given a valid --adopt entry
  When the user runs engram register-skills --dry-run --adopt claude:curate=1049
  Then the note is neither renamed nor rewritten

Scenario: 792a12i1j Adopt refuses a skill that Pi disables
  Given a Pi user skill ping that the global settings.json switches off with "!ping", and a runbook note 1
  When the user runs engram register-skills --adopt pi:ping=1
  Then it is refused as naming no scanned skill and nothing is written
```

## 792a12j Claude Code skills, commands and synced skills are found

Requirement: [792a12j Claude Code skills, commands and synced skills are found](#792a12j-claude-code-skills-commands-and-synced-skills-are-found) Registration MUST find, for a detected Claude Code harness (`~/.claude`), its user skills as
`~/.claude/skills/<name>/SKILL.md` from immediate children that are directories or symlinks resolving to
directories, its user commands as `~/.claude/commands/**/*.md` named by their relative path with `/`
replaced by `:`, and its claude.ai-synced skills as `<bucket>/<name>/SKILL.md` for each directory bucket
under `~/.claude/skills/synced/` whose `manifest.json` parses. It MUST ignore root files, `synced`
itself as a skill, dangling symlinks and directories without `SKILL.md` among the user skills, and
non-directory entries among the buckets.

### 792a12j1 UAT Scenarios

```gherkin
Scenario: 792a12j1a A symlinked user skill is found
  Given ~/.claude/skills/route is a symlink to ~/.claude/engram/skills/route containing SKILL.md
  When the user runs engram register-skills
  Then it is the scanned Claude user skill claude:route whose source is the resolved ~/.claude/engram/skills/route/SKILL.md

Scenario: 792a12j1b A dangling symlink is skipped
  Given ~/.claude/skills/qa is a symlink whose target does not exist
  When the user runs engram register-skills
  Then no offer is made for it and registration does not fail

Scenario: 792a12j1c A user command in a subdirectory
  Given ~/.claude/commands/opsx/apply.md exists
  When the user runs engram register-skills
  Then it is scanned as a command named opsx:apply

Scenario: 792a12j1d Synced skills come from manifest buckets
  Given ~/.claude/skills/synced/ holds one bucket directory with a parseable manifest.json and skill pdf/SKILL.md, plus a stray file .bucket-x
  When the user runs engram register-skills
  Then pdf is scanned as a synced skill and the stray file is ignored
```

## 792a12k Pi, agents and Pi-configured skills, commands and prompts are found

Requirement: [792a12k Pi, agents and Pi-configured skills, commands and prompts are found](#792a12k-pi-agents-and-pi-configured-skills-commands-and-prompts-are-found) Registration MUST find, for a detected Pi harness (`~/.pi`), its user skills in `~/.pi/agent/skills/`
(recursively discovered skill directories, plus root `.md` files), the skills in `~/.agents/skills/`
(recursively, ignoring root `.md` files), its prompt templates in `~/.pi/agent/prompts/*.md` (flat,
named by file stem), and the `skills` and `prompts` paths and `packages` configured in Pi's global
`settings.json`. A package entry given as a string whose `package.json` has any `pi` key MUST load only
the manifest's `pi.skills` and `pi.prompts` entries, and the convention directories `skills/` and
`prompts/` MUST be used only when `package.json` has no `pi` key. For an object-form entry an explicit
`skills` or `prompts` pattern list MUST filter that type of the package (`[]` switches every file of
that type off), and an omitted key MUST fall back, for that type only, to the manifest's entry if
present, else the convention directory. A settings or package skill directory MUST be searched
recursively and include its own root `.md` files, and a prompt directory recursively. A file that a
settings entry, a `!pattern` or `-path` over a default folder, or an object-form package filter switches
off MUST be kept as disabled: it MUST NOT be offered, and it counts as present for removal.

### 792a12k1 UAT Scenarios

```gherkin
Scenario: 792a12k1a A Pi prompt template
  Given ~/.pi/agent/prompts/review.md exists
  When the user runs engram register-skills
  Then it is scanned as a Pi prompt template named review

Scenario: 792a12k1b A Pi settings skill path
  Given Pi's global settings.json has "skills": ["~/extra-skills"] and ~/extra-skills/fmt/SKILL.md exists
  When the user runs engram register-skills
  Then fmt is scanned as a Pi settings skill

Scenario: 792a12k1c Package skills come from its manifest
  Given settings.json lists "npm:pi-intercom" and the package's package.json has "pi": {"skills": ["./skills"]} with skills/pi-intercom/SKILL.md
  When the user runs engram register-skills
  Then pi-intercom is scanned as a skill of package pi-intercom

Scenario: 792a12k1d A string-form package with a pi key ignores convention directories
  Given settings.json lists "npm:x", x's package.json has "pi": {"extensions": ["./index.ts"]}, and the package also has skills/fmt/SKILL.md
  When the user runs engram register-skills
  Then no skill of package x is scanned

Scenario: 792a12k1e An object-form omitted type falls back per type
  Given settings.json lists {"source": "npm:x", "extensions": []} for the same package
  When the user runs engram register-skills
  Then fmt is scanned from skills/, because the object form falls back to the convention directory for the omitted skills type

Scenario: 792a12k1f An object-form empty skills filter
  Given settings.json lists {"source": "npm:pi-intercom", "skills": []}
  When the user runs engram register-skills
  Then no skill of that package is offered, and its pi-intercom skill is kept as disabled, present for removal

Scenario: 792a12k1g A Pi root markdown skill
  Given ~/.pi/agent/skills/notes.md is a root file
  When the user runs engram register-skills
  Then it is scanned as Pi user skill notes

Scenario: 792a12k1h A configured package with a missing path contributes nothing
  Given Pi's settings.json lists a local package path that does not exist
  When the user runs engram register-skills
  Then no skill is scanned from it and registration does not fail
```

## 792a12l Installed Claude Code plugins' skills and commands are found

Requirement: [792a12l Installed Claude Code plugins' skills and commands are found](#792a12l-installed-claude-code-plugins-skills-and-commands-are-found) Registration MUST find the skills and commands of each installed Claude Code plugin that is enabled by
its `enabledPlugins` entry or, when the entry is absent, by its `plugin.json` `defaultEnabled` (default
true), whose scope is `user` or is project-scoped to the current repository, and whose `installPath`
exists. Its skills MUST come from `<installPath>/skills/<name>/SKILL.md` plus any `plugin.json` `skills`
paths, and its commands from `<installPath>/commands/**/*.md` or, when `plugin.json` `commands` is a
path or an array, from those paths instead. A declared path that is absolute, leaves the plugin
lexically or through a symlink, or does not exist MUST be refused with a warning and make that plugin
count as not scanned. No other cached version MUST be read.

### 792a12l1 UAT Scenarios

```gherkin
Scenario: 792a12l1a plugin.json commands replace the commands directory
  Given an enabled plugin's plugin.json has "commands": ["./commands/setup.md"] and its commands/ directory also holds configure.md
  When the user runs engram register-skills
  Then only setup is scanned as that plugin's command

Scenario: 792a12l1b A disabled plugin contributes nothing
  Given a plugin in installed_plugins.json with enabledPlugins set to false
  When the user runs engram register-skills
  Then none of its skills or commands are scanned

Scenario: 792a12l1c A plugin without an enabledPlugins entry and without an install path
  Given a plugin in installed_plugins.json with no enabledPlugins entry and a missing installPath
  When the user runs engram register-skills
  Then none of its skills or commands are scanned

Scenario: 792a12l1d Only the installed version is read
  Given the plugin cache holds several versions of a plugin and installed_plugins.json names one installPath
  When the user runs engram register-skills
  Then only that installPath's skills and commands are scanned
```

## 792a12m Project skills and commands are found

Requirement: [792a12m Project skills and commands are found](#792a12m-project-skills-and-commands-are-found) Inside a git repository, registration MUST find the Claude Code project skills and commands
`.claude/skills/<name>/SKILL.md` and `.claude/commands/**/*.md` in the working directory and each
ancestor up to the repository top-level, ignoring root `.md` files in `.claude/skills`. For a project Pi
trusts it MUST also find, in this precedence order, `.pi/skills/`, `.agents/skills/` in the working
directory and its ancestors up to the top-level, the `.pi/settings.json` skill paths,
`.pi/prompts/*.md`, the `.pi/settings.json` prompt paths and the `.pi/settings.json` packages. A project
is trusted when the nearest saved decision in `~/.pi/agent/trust.json` for the folder or a parent says
so, or, with no saved decision, when the global `defaultProjectTrust` is `always`. Outside a git
repository registration MUST find no project entry.

### 792a12m1 UAT Scenarios

```gherkin
Scenario: 792a12m1a Agents skills in an ancestor
  Given a trusted project has .agents/skills/lint/SKILL.md at its top-level
  When the user runs engram register-skills from a subdirectory of it
  Then lint is scanned as a project agents skill

Scenario: 792a12m1b Trust falls back to defaultProjectTrust always
  Given the working directory has .pi/skills/foo/SKILL.md, trust.json has no decision for it or any parent, and the global settings.json has "defaultProjectTrust": "always"
  When the user runs engram register-skills
  Then foo is scanned as a Pi project skill

Scenario: 792a12m1c An untrusted Pi project is not scanned
  Given the working directory has .pi/skills/foo/SKILL.md, trust.json has no trusting decision for it or any parent, and defaultProjectTrust is not always
  When the user runs engram register-skills
  Then no Pi project skill is scanned

Scenario: 792a12m1d Claude project sources from a subdirectory
  Given a repository whose top-level has .claude/skills/openspec-propose/SKILL.md and .claude/commands/commit.md
  When the user runs engram register-skills from internal/
  Then both are scanned, exactly as from the top-level

Scenario: 792a12m1e Root markdown in the project skills directory is ignored
  Given a repository's .claude/skills/ holds root files commit.md and engram-go-conventions.md and .claude/commands/commit.md exists
  When the user runs engram register-skills
  Then neither root file is scanned as a skill and .claude/commands/commit.md is scanned as a command

Scenario: 792a12m1f Outside a repository there are no project entries
  Given a working directory outside any git repository with a .claude/skills directory
  When the user runs engram register-skills
  Then no project skill is scanned
```

## 792a12n Sources are read only for detected harnesses, and names come from the file system

Requirement: [792a12n Sources are read only for detected harnesses, and names come from the file system](#792a12n-sources-are-read-only-for-detected-harnesses-and-names-come-from-the-file-system) Registration MUST read the Claude Code sources only when `~/.claude` exists and the Pi and `~/.agents`
sources only when `~/.pi` exists. A name MUST be the directory name, file stem or command path, never a
frontmatter field. An entry whose name contains `:` (other than a command's namespace separators) MUST
be skipped with a warning, and an entry whose key is not recognized ([792a12b One runbook note per skill, identified by its key](#792a12b-one-runbook-note-per-skill-identified-by-its-key)) MUST be skipped with a
warning.

### 792a12n1 UAT Scenarios

```gherkin
Scenario: 792a12n1a A machine without Pi reads no Pi sources
  Given ~/.claude exists, ~/.pi does not, and ~/.agents/skills/lint/SKILL.md exists
  When the user runs engram register-skills
  Then no Pi or agents entry is scanned

Scenario: 792a12n1b A name with a colon is skipped
  Given ~/.claude/skills/a:b/SKILL.md exists
  When the user runs engram register-skills
  Then it is skipped with a warning and no offer is made for it
```

## 792a12o Every scanned entry has a source-qualified key

Requirement: [792a12o Every scanned entry has a source-qualified key](#792a12o-every-scanned-entry-has-a-source-qualified-key) Each scanned entry MUST have a key determined by the source folder it was found in, and no key MUST be
bare: Claude Code user skill `claude:<name>`; Claude Code user command `claude:cmd:<name>`;
claude.ai-synced skill `anthropic-skills:<name>`; Pi user skill `pi:<name>`; `~/.agents/skills` skill
`agents:<name>`; Pi settings skill `pi-settings:<name>`; Pi package skill `pi-pkg:<pkg-id>:<name>`; Pi
prompt template `pi-prompt:<name>` (global folder), `pi-settings:pi-prompt:<name>` or
`pi-pkg:<pkg-id>:pi-prompt:<name>`; plugin skill `<plugin>:<name>`; plugin command
`<plugin>:cmd:<name>`; Claude Code project skill `project:<r>:<name>` and command
`project:<r>:cmd:<name>`; and Pi and agents project entries `project:<r>:pi:<name>`,
`project:<r>:agents:<name>`, `project:<r>:pi-prompt:<name>`, `project:<r>:pi-settings:<name>` and
`project:<r>:pi-pkg:<pkg-id>:<name>`. `<plugin>` is the part of the manifest key before `@`. `<pkg-id>`
is the npm name, the git `<host>/<path>` or the `~`-relative local path. `<r>` is the origin remote's
host followed by its path, lowercased, without userinfo, port, trailing `.git` or `/`, or, without an
origin, `local/` followed by the basename of the parent of the absolute `git rev-parse
--git-common-dir`. A skill that engram installs MUST be keyed like any other entry of the folder it is
found in. An enabled plugin named `claude`, `pi`, `agents`, `project`, `anthropic-skills`,
`pi-settings`, `pi-pkg` or `pi-prompt` MUST be skipped with a warning, but a disabled one MUST
contribute nothing and print nothing. The note slug MUST be `skill-` followed by the key lowercased,
every run of characters outside `[a-z0-9]` replaced by `-`, and leading and trailing `-` trimmed.

### 792a12o1 UAT Scenarios

```gherkin
Scenario: 792a12o1a A plugin skill's key and slug
  Given the enabled plugin superpowers@claude-plugins-official ships skills/brainstorming/SKILL.md
  When the user runs engram register-skills
  Then its key is superpowers:brainstorming and an accepted registration creates a note whose basename ends in .skill-superpowers-brainstorming.md

Scenario: 792a12o1b A plugin's skill and command never collide
  Given the enabled plugin commit@skills ships skills/commit/SKILL.md and commands/commit.md
  When the user runs engram register-skills
  Then their keys are commit:commit and commit:cmd:commit

Scenario: 792a12o1c A worktree resolves to owner and repository
  Given a linked worktree directory named runbook-vs-skill whose origin remote is ssh://git@github.com/toejough/engram.git
  When the user runs engram register-skills
  Then its project skills' keys begin with project:github.com/toejough/engram: and their slugs with skill-project-github-com-toejough-engram-

Scenario: 792a12o1d Two same-named projects stay distinct
  Given repositories with origins ssh://git@GitHub.com:22/toejough/engram.git and https://gitlab.com/toejough/engram, each with a project skill deploy
  When the user runs engram register-skills
  Then their keys are project:github.com/toejough/engram:deploy and project:gitlab.com/toejough/engram:deploy

Scenario: 792a12o1e Engram-installed skills are keyed by their folder
  Given ~/.claude/skills/route links to ~/.claude/engram/skills/route and ~/.pi/agent/skills/route links to ~/.pi/agent/engram/skills/route
  When the user runs engram register-skills
  Then the keys are claude:route and pi:route and no key is bare

Scenario: 792a12o1f A Claude user command's key
  Given ~/.claude/commands/audit.md is scanned
  When the user runs engram register-skills
  Then its key is claude:cmd:audit and its note slug is skill-claude-cmd-audit

Scenario: 792a12o1g A plugin named claude is reserved
  Given an enabled plugin claude@m is installed
  When the user runs engram register-skills
  Then it is skipped with a warning and no offer is made for any claude: key from it

Scenario: 792a12o1h A disabled reserved-name plugin is silent
  Given a plugin pi@m with enabledPlugins set to false
  When the user runs engram register-skills
  Then no offer is made for it and no warning is printed

Scenario: 792a12o1i A plugin named cmd is an ordinary plugin
  Given an enabled plugin cmd@m ships skills/x/SKILL.md
  When the user runs engram register-skills
  Then its key is cmd:x

Scenario: 792a12o1j A prompt template's key
  Given ~/.pi/agent/prompts/review.md is scanned
  When the user runs engram register-skills
  Then its key is pi-prompt:review
```

## 792a12p Copies of one skill collapse and key conflicts are reported

Requirement: [792a12p Copies of one skill collapse and key conflicts are reported](#792a12p-copies-of-one-skill-collapse-and-key-conflicts-are-reported) Scanned entries resolving to the same file MUST be one entry. Entries with the same key MUST collapse
when byte-identical, and when they differ MUST be reported as a key conflict naming every path, with no
offer for that key, and the run MUST exit with a failure status after handling every other offer. The
same name found at two levels of a project directory chain is such a conflict. A plugin name installed
from more than one marketplace MUST be reported as a plugin conflict, with no offer for any of its keys,
and its scope treated as not scanned. There MUST be no exception for any source, including skills that
engram installs. An entry whose SHA-256 equals that of an entry earlier in source precedence in the same
run, or equals the `skill_hash` of any existing skill note, MUST be an alias: it makes no offer, is
recorded nowhere, and counts as present for its key.

Rationale: The same key with differing bytes is always a conflict: there is no nearest-wins across a project
chain and no engram-owned exception, because engram's copies on different harnesses have different keys. A
conflicted key makes no offer at all. Aliases are never written to notes, because that would write on runs
where the procedure did not change.

### 792a12p1 UAT Scenarios

```gherkin
Scenario: 792a12p1a A Pi copy of an engram skill makes no offer
  Given ~/.pi/agent/skills/route and ~/.claude/skills/route are byte-identical and a note keyed claude:route carries that hash
  When the user runs engram register-skills
  Then no offer is made for claude:route or pi:route, and pi:route counts as present

Scenario: 792a12p1b Two Pi skills with one name and different content
  Given Pi recursion finds ~/.pi/agent/skills/a/foo/SKILL.md and ~/.pi/agent/skills/b/foo/SKILL.md with different bytes
  When the user runs engram register-skills
  Then a conflict is reported for pi:foo naming both paths, no offer is made for pi:foo, every other offer is handled, and the exit status is a failure

Scenario: 792a12p1c The same project skill at two levels
  Given a repository with .claude/skills/foo/SKILL.md at its top-level and a different sub/.claude/skills/foo/SKILL.md
  When the user runs engram register-skills from sub/
  Then a conflict is reported for project:<r>:foo naming both paths and no offer is made for it

Scenario: 792a12p1d A plugin name in two marketplaces
  Given plugins tools@alpha and tools@beta both installed and enabled
  When the user runs engram register-skills
  Then a plugin conflict is reported for tools, no offer is made for any tools: key, and no removal of existing tools: notes is offered

Scenario: 792a12p1e Diverged harness copies are separate keys
  Given ~/.claude/skills/route and ~/.pi/agent/skills/route differ in bytes, a note keyed claude:route carries the Claude copy's hash, and no note or decline covers the Pi copy's hash
  When the user runs engram register-skills
  Then no offer is made for claude:route, registering pi:route is offered, and no conflict or warning is reported

Scenario: 792a12p1f Two synced buckets hold the same skill
  Given two synced buckets both hold pdf/SKILL.md with identical bytes
  When the user runs engram register-skills
  Then one candidate anthropic-skills:pdf results and no conflict is reported

Scenario: 792a12p1g A diverged plugin skill of the same name is its own skill
  Given skill-creator@claude-plugins-official's skill-creator and the synced skill-creator differ in bytes
  When the user runs engram register-skills
  Then both skill-creator:skill-creator and anthropic-skills:skill-creator are offered
```

## 792a12q Removal is offered only for successfully read sources

Requirement: [792a12q Removal is offered only for successfully read sources](#792a12q-removal-is-offered-only-for-successfully-read-sources) A source root MUST count as read only when its read succeeded: a read error of any kind, including
not-exist, means not read, never empty. A skill note MUST be eligible for removal only as follows. For
`claude:`, `claude:cmd:`, `pi:`, `agents:` and `pi-prompt:` keys, the one fixed user root of that form
(`~/.claude/skills`, `~/.claude/commands`, `~/.pi/agent/skills`, `~/.agents/skills`,
`~/.pi/agent/prompts`) must have been read, and no other root matters. For `anthropic-skills:`,
`pi-settings:`, `pi-pkg:` and `project:` keys, the specific root that holds the note's recorded
`skill_source` must have been read in this run (the synced bucket whose `manifest.json` parsed, the
settings entry's path, the package root, or the exact project directory). For plugin keys,
`installed_plugins.json` and `settings.json` must both have parsed, the plugin must have no plugin
conflict, and it must be either enabled with its `installPath` read or absent from the manifest. A note
whose `skill_source` lies under no read root, and a runbook note with `skill_hash` but no `skill_key`,
MUST NOT be eligible. Registration MUST offer to remove a skill note only when it is eligible and no
scanned entry, alias or disabled or not, has its key, and MUST make no removal offer when `--skills-dir`
is given.

Rationale: An existence-based scan was rejected because an unreadable directory would look empty and flood the
user with removals; a per-scope scanned flag was rejected because it cannot tell which entry, bucket or
project a note came from; and a `last_seen` counter was rejected as hidden state that misfires when a project
is not checked out. The accepted cost is that a note whose source root left the source set is never offered
for removal, so the user deletes it by hand.

### 792a12q1 UAT Scenarios

```gherkin
Scenario: 792a12q1a A project skill note from another directory
  Given a note with key project:github.com/toejough/engram:openspec-propose
  When the user runs engram register-skills from a directory outside that repository
  Then no removal offer is made for it

Scenario: 792a12q1b An update run without a project
  Given engram update runs from a directory that is not in a git repository and project skill notes exist
  When the user runs engram update
  Then no removal offer is made for any project skill note

Scenario: 792a12q1c A disabled plugin keeps its notes
  Given a note with key hookify:writing-rules and hookify installed but disabled
  When the user runs engram register-skills
  Then no removal offer is made for it

Scenario: 792a12q1d A Pi-disabled skill keeps its note
  Given a note with key pi:ping, ~/.pi/agent/skills read, and the global Pi settings.json skills holding "!ping"
  When the user runs engram register-skills
  Then no register, refresh or removal offer is made for it

Scenario: 792a12q1e An uninstalled plugin is offered for removal
  Given a note with key ralph-loop:cmd:help and ralph-loop absent from a readable installed_plugins.json
  When the user runs engram register-skills
  Then removal of that note is offered

Scenario: 792a12q1f An unreadable user skills directory
  Given listing ~/.claude/skills fails and claude: notes exist
  When the user runs engram register-skills
  Then no removal offer is made for them

Scenario: 792a12q1g Another namespace's missing root does not block removal
  Given ~/.agents/skills does not exist, ~/.pi/agent/skills fails with a permission error, ~/.claude/skills is read and lacks c4, and a note keyed claude:c4 exists
  When the user runs engram register-skills
  Then removal of the claude:c4 note is offered

Scenario: 792a12q1h An unkeyed legacy note is never offered for removal
  Given runbook note 1036.2026-09-18.skill-route.md carries skill_hash and no skill_key, and every source root was read
  When the user runs engram register-skills
  Then no removal offer is made for it

Scenario: 792a12q1i A missing synced manifest
  Given ~/.claude/skills/synced/ holds no bucket with a readable manifest.json and anthropic-skills: notes exist
  When the user runs engram register-skills
  Then no removal offer is made for them

Scenario: 792a12q1j A nested-only project skill is safe from the top level
  Given a note keyed project:github.com/toejough/engram:foo was registered from sub/ and its skill_source is under sub/.claude/skills
  When the user runs engram register-skills from the repository top-level
  Then no removal offer is made for it

Scenario: 792a12q1k One failing synced bucket
  Given two synced buckets, bucket A's manifest.json parses, bucket B's does not, and a note's skill_source lies in bucket B
  When the user runs engram register-skills
  Then no removal offer is made for that note

Scenario: 792a12q1l A Pi settings note is tied to its own entry
  Given settings.json has two skills entries, the first unreadable, and a pi-settings: note whose skill_source lies under the first
  When the user runs engram register-skills
  Then no removal offer is made for that note

Scenario: 792a12q1m A package dropped from settings leaves an orphan
  Given a pi-pkg:x: note exists and npm:x is no longer listed in settings.json
  When the user runs engram register-skills
  Then no removal offer is made for it

Scenario: 792a12q1n An alias keeps its note
  Given a note keyed pi:ping and the Pi ping skill now byte-identical to a higher-precedence skill
  When the user runs engram register-skills
  Then no removal offer is made for the pi:ping note

Scenario: 792a12q1o A machine without the harness
  Given ~/.pi does not exist and a note keyed pi:ping exists
  When the user runs engram register-skills
  Then no removal offer is made for it
```

## 792a12r Offers are grouped by scope and answered by key, pattern or selector

Requirement: [792a12r Offers are grouped by scope and answered by key, pattern or selector](#792a12r-offers-are-grouped-by-scope-and-answered-by-key-pattern-or-selector) Each offer MUST belong to a scope: `claude-user` for `claude:` skill keys, `claude-cmd` for
`claude:cmd:` keys, `synced`, `pi-user`, `agents-user`, `pi-settings` and `pi-prompt` for those sources,
`pi-pkg:<pkg-id>` for a Pi package, `plugin:<plugin>` for a plugin and `project:<r>` for a project.
Offers MUST be ordered by scope then key, a scope's display label being its selector `@<scope-id>` with
its count. `--accept` and `--decline` MUST accept an exact key, a pattern ending in `*` that matches
keys by prefix, or a selector `@<scope-id>`; for each offer an exact key MUST take precedence, then the
longest matching pattern, then a selector. When prompting interactively, registration MUST ask once for
each scope holding more than one register or refresh offer, listing them and offering accept all,
decline all, review each and skip, where skip, any other answer and end of input record nothing; a
single register or refresh offer, and every removal, MUST get its own `[y/N]` prompt, where `y` or `yes`
accepts, any other answer declines and end of input records nothing. Removal offers MUST be answered
only by an exact key or an individual prompt, never by accept-all, a pattern or a selector. `--dry-run`
MUST list each offer as `would offer: <kind> <key> (<source>)` under its `@<scope-id> (<count>)` header,
naming the note instead of a source for a removal with no recorded source.

Rationale: Offers are grouped by scope because dozens of per-skill prompts are unworkable. Removals are never
covered by accept-all, a pattern or a selector; each needs its exact key or its own prompt. Scopes are not
opt-in, because Joe directed that all default folders be scanned.

### 792a12r1 UAT Scenarios

```gherkin
Scenario: 792a12r1a Decline a whole plugin
  Given 15 superpowers register offers outstanding
  When the user runs engram register-skills --decline @plugin:superpowers
  Then all 15 current hashes are recorded as declined under their keys and no note is written

Scenario: 792a12r1b The longest pattern wins
  Given superpowers register offers outstanding
  When the user runs engram register-skills --decline 'superpowers:*' --accept 'superpowers:writing-*'
  Then superpowers:writing-plans and superpowers:writing-skills are registered and the other superpowers offers are declined

Scenario: 792a12r1c An exact key beats a pattern
  Given superpowers register offers outstanding
  When the user runs engram register-skills --decline 'superpowers:*' --accept superpowers:brainstorming
  Then superpowers:brainstorming is registered and the other superpowers offers are declined

Scenario: 792a12r1d The Claude user scope is nameable
  Given register offers for claude:c4 and claude:dev
  When the user runs engram register-skills --decline @claude-user
  Then both are recorded as declined

Scenario: 792a12r1e Removals are never bulk-accepted
  Given one register offer and one removal offer outstanding
  When the user runs engram register-skills --accept '*'
  Then the register offer is carried out and the removal offer is only reported

Scenario: 792a12r1f Skip leaves the scope for next time
  Given the user answers skip for a scope at the interactive prompt
  When the user runs engram register-skills
  Then nothing is written or recorded for that scope's offers and the next run offers them again

Scenario: 792a12r1g A scope with several offers is asked once
  Given a terminal and a scope with three register offers
  When the user runs engram register-skills
  Then the three are listed and asked about once, and answering accept all registers them all

Scenario: 792a12r1h End of input at a prompt records nothing
  Given a terminal whose input ends at the first prompt
  When the user runs engram register-skills
  Then no offer is accepted or declined
```

## 792a12s Declines are keyed by skill key under schema version 2

Requirement: [792a12s Declines are keyed by skill key under schema version 2](#792a12s-declines-are-keyed-by-skill-key-under-schema-version-2) `skill-registrations.json` MUST map keys to declined hashes under `schema_version: 2`. A file whose
`schema_version` is anything other than 2 (missing, below 2 or above 2), that has an unrecognized field,
or that cannot be decoded MUST be an error, not an empty decline state, and registration MUST write
nothing. A missing file MUST mean no declines.

Rationale: Version 1 files are a loud error and are not mapped to `claude:<name>`, because v1 never left the
unmerged branch and a compatibility mapping for it was rejected. An older binary still reads a v2 file, since
it ignores the extra keys.

### 792a12s1 UAT Scenarios

```gherkin
Scenario: 792a12s1a A version-1 file fails loudly
  Given skill-registrations.json is {"schema_version":1,"declined":{"route":"<h>"}}
  When the user runs engram register-skills
  Then registration reports an error and writes nothing

Scenario: 792a12s1b An unknown schema version fails loudly
  Given skill-registrations.json has schema_version 3
  When the user runs engram register-skills
  Then registration reports an error and writes nothing

Scenario: 792a12s1c A missing file is no declines
  Given no skill-registrations.json
  When the user runs engram register-skills
  Then every scanned skill without a note is offered
```

## 792a12t Update runs registration after deploying the skills

Requirement: [792a12t Update runs registration after deploying the skills](#792a12t-update-runs-registration-after-deploying-the-skills) After the engram-owned root sync completes, `engram update` MUST run skill registration against the
resolved vault over the same default source set that `engram register-skills` scans, resolved by the
same function from the same home and working directory and not from `agent-instructions/skills` of the
source, using its own `--dry-run` flag and no explicit answers. It MUST prompt for each outstanding
offer when stdin is a terminal, and otherwise write nothing and report the outstanding offers. A
registration failure MUST be reported as `skill registration error: <message>` and MUST NOT roll back
the deploy or fail the update.

### 792a12t1 UAT Scenarios

```gherkin
Scenario: 792a12t1a An interactive update
  Given `engram update` runs in a terminal and a shipped skill has no note
  When the user runs engram update
  Then the skills are deployed to every harness and the user is offered registration of that skill

Scenario: 792a12t1b A non-interactive update
  Given `engram update` runs with stdin not a terminal and offers outstanding
  When the user runs engram update
  Then the skills are deployed, no vault note or decline record is written, and the report names the skills awaiting an answer

Scenario: 792a12t1c A dry run previews the offers too
  Given offers outstanding
  When the user runs engram update --dry-run
  Then the plan lists the registration offers alongside the sync operations and nothing is written

Scenario: 792a12t1d Update and register-skills see the same offers
  Given a completed sync
  When the user runs engram update --dry-run and engram register-skills --dry-run from the same directory against the same home and vault
  Then both list the same registration offers

Scenario: 792a12t1e A registration failure does not roll back the deploy
  Given registration fails after the skills are deployed
  When the user runs engram update
  Then the report holds a skill registration error line and the deployed skills remain
```

## 792a12u Skill identity fields belong to registration

Requirement: [792a12u Skill identity fields belong to registration](#792a12u-skill-identity-fields-belong-to-registration) A runbook note created, adopted or refreshed by registration MUST carry `skill_hash`, `skill_key` and
`skill_source`, and every path that rewrites a note's frontmatter without being registration MUST
preserve all three unchanged: `engram amend` (including `--clear-pending`), `engram resituate`, Luhmann
re-parenting, the rename-and-rewrite of wikilinks, the identity backfill and the vocabulary tag
rewrites. A note captured by `engram learn` outside registration MUST carry none of them, and `engram
learn` MUST NOT accept them as arguments.

### 792a12u1 UAT Scenarios

```gherkin
Scenario: 792a12u1a Amend preserves the identity fields
  Given a note carrying skill_hash, skill_key and skill_source
  When the user runs engram amend on it
  Then all three are unchanged in the written note

Scenario: 792a12u1b Resituate preserves the identity fields
  Given a skill note
  When the user runs engram resituate on it
  Then all three are unchanged

Scenario: 792a12u1c Reparenting preserves the identity fields
  Given a skill note carrying all three fields
  When Luhmann re-parenting renames it
  Then all three are unchanged in the renamed note

Scenario: 792a12u1d A wikilink rewrite preserves the identity fields
  Given a rename-and-rewrite pass rewriting a wikilink inside a skill note's frontmatter
  When it runs
  Then its skill_hash, skill_key and skill_source are unchanged

Scenario: 792a12u1e The identity backfill preserves the identity fields
  Given a skill note missing repo, user and vault
  When the user runs engram update --backfill-identity
  Then the note is stamped and its three skill fields are unchanged

Scenario: 792a12u1f Vocabulary rewrites preserve the identity fields
  Given a skill note
  When vocabulary assignment, tag rewrites or definition tagging rewrite its tags
  Then its three skill fields are unchanged

Scenario: 792a12u1g Ordinary capture carries none
  Given a note captured by engram learn outside registration
  When it is written
  Then it carries no skill_hash, skill_key or skill_source, and no learn flag sets them
```

## 792a12v Skill notes behave like captured runbooks

Requirement: [792a12v Skill notes behave like captured runbooks](#792a12v-skill-notes-behave-like-captured-runbooks) A runbook note mirroring a skill MUST participate in the Luhmann hierarchy, embedding, vocabulary
tagging, retrieval, triggers and the follow-frame of the runbook shim exactly as a captured runbook
does, with its `situation`, `done_when`, `red_flags` and `triggers` authored on the note and edited with
`engram amend`. Until curated, a note registered or refreshed by registration is a pending offer. Its
body preamble MUST name the file it mirrors, which is its home-relative `skill_source`, for a skill,
command or prompt template alike.

### 792a12v1 UAT Scenarios

```gherkin
Scenario: 792a12v1a A skill note surfaces like a captured one
  Given a non-pending skill note with a trigger
  When the user runs engram query --text containing that trigger
  Then the note surfaces first with trigger provenance, as a captured runbook with the same triggers would

Scenario: 792a12v1b The body preamble marks the source
  Given a skill runbook note
  When the user runs engram show on it
  Then the body begins with a line naming the skill file it mirrors

Scenario: 792a12v1c A non-engram skill's preamble names the real file
  Given a registered note mirroring the user skill ~/.claude/skills/c4/SKILL.md
  When the user runs engram show on it
  Then its preamble names ~/.claude/skills/c4/SKILL.md, not an agent-instructions/skills/ path

Scenario: 792a12v1d An engram-installed skill's preamble names the deployed file
  Given a registered note mirroring claude:route, whose resolved source is ~/.claude/engram/skills/route/SKILL.md
  When the user runs engram show on it
  Then its preamble names ~/.claude/engram/skills/route/SKILL.md, not agent-instructions/skills/route/SKILL.md

Scenario: 792a12v1e A pending skill note is a pending offer
  Given a newly registered skill note with pending: true
  When curation lists pending offers
  Then the note is among them
```
