# Skill Source Discovery - Technical Spec

## 792b30a Orientation

- Ownership: Repository-owned dependency-boundary interaction (the harness configuration files, the skill
  folders and the plugin and Pi settings manifests, over the injected read-only filesystem and the `git`
  command runner).
- Responsibility: finds every skill, command and prompt template that a detected harness loads, in the
  default folders and in the configuration that points at more, reads each one's bytes and resolved path,
  records which source roots it could read, and gives each a source-qualified key, so that registration
  never takes an unreadable source for an empty one.
- Current realization (informational): `internal/cli/skillsources.go`, `skillsources_pi.go`,
  `skillsources_piconfig.go`, `skillsources_plugin.go`, `skillsources_project.go` and
  `skillsources_resolve.go` (`ResolveSkillSources`, the scanners, `ResolveSkillPath`), and
  `internal/cli/skillkeys.go` (`AssignSkillKeys`, `SkillKeySlug`, `parseSkillKey`).
- Qualification: responsibility, discovering the skills a harness loads is stated without naming any
  scanner; ownership, it owns discovery, keys and read status, while the comparison with the vault belongs
  to [skill-registration-ledger](./skill-registration-ledger.md) and the harness list to [harness-deploy-sync](./harness-deploy-sync.md); independent verification, each
  scanner runs over an in-memory filesystem with injected read faults; stability, the contract survives the
  addition of a source or a change in how a harness stores its settings; necessity, no offer can be made
  without it.
- Supported capabilities: [skill-runbook-registration](../product/skill-runbook-registration.md).

Rationale: registration used to cover only engram's own six skills (vault note 1066a widened it to every
folder a harness loads), so discovery follows each harness's own rules: Claude Code's folders and plugin
manifests, and Pi's settings, packages and project trust (matched to Pi's own package manager). A read that
fails is never treated as an empty folder, because a removal offer built on a misread folder would delete a
note whose skill still exists.

## 792b30c Behavior Requirements

### 792b30c1 Symlink-resolving path reads

Requirement: [792b30c1 Symlink-resolving path reads](#792b30c1-symlink-resolving-path-reads) The skill-source-discovery MUST resolve a path with every symbolic link in it followed, using only link inspection and link reading so
that it stays pure over the injected filesystem: a chain of absolute and relative links ending at a real
directory MUST resolve to that directory, resolving MUST be idempotent, `..` MUST step to the parent of
the resolved path so far, a dangling link MUST give an error that satisfies `fs.ErrNotExist`, and a link
loop MUST give `errSkillPathLinkLoop`.

Supports: [792a12j1a A symlinked user skill is found](../product/skill-runbook-registration.md#792a12j1-uat-scenarios).

#### 792b30c1a Scenarios

```gherkin
Scenario: 792b30c1a1 A chain resolves to its terminal
  Given any chain of links with absolute or relative targets ending at a real directory
  When any link in the chain is resolved
  Then the real directory results, and resolving it again changes nothing

Scenario: 792b30c1a2 A dangling link is not-exist
  Given a link whose target does not exist
  When it is resolved
  Then the error satisfies fs.ErrNotExist

Scenario: 792b30c1a3 Dot-dot ends on the parent
  Given a path containing ..
  When it is resolved
  Then the result is the parent of the path resolved so far

Scenario: 792b30c1a4 A link loop is an error
  Given links that point at each other
  When one is resolved
  Then the error is errSkillPathLinkLoop

Scenario: 792b30c1a5 A relative target and a symlinked parent resolve
  Given a relative link whose directory is itself reached through a link
  When it is resolved
  Then the result is the real location
```

### 792b30c2 Root read status

Requirement: [792b30c2 Root read status](#792b30c2-root-read-status) The skill-source-discovery MUST record every source root a scanner tries to read, with its resolved path and whether the read succeeded,
MUST count a root as scanned only when its resolution, its listing and the read of every child
succeeded, and MUST treat every error, including not-exist, as not scanned and never as empty. It MUST
add a warning for a failure other than not-exist, and MUST NOT warn about a root that simply does not
exist.

Supports: [792a12j1a A symlinked user skill is found](../product/skill-runbook-registration.md#792a12j1-uat-scenarios), [792a12q1f An unreadable user skills directory](../product/skill-runbook-registration.md#792a12q1-uat-scenarios).

#### 792b30c2a Scenarios

```gherkin
Scenario: 792b30c2a1 A missing root is not scanned and silent
  Given a source root that does not exist
  When it is scanned
  Then it is recorded as not scanned and no warning is added

Scenario: 792b30c2a2 A listing failure is not scanned and warned
  Given a root that resolves but whose listing fails with a permission error
  When it is scanned
  Then it is recorded resolved but not scanned, a warning is added, and nothing else is read

Scenario: 792b30c2a3 A failure on the root path is not scanned
  Given a root whose path cannot be read
  When it is scanned
  Then it is recorded not scanned

Scenario: 792b30c2a4 A child that cannot be read marks the root not scanned
  Given a child entry whose read fails
  When the root is scanned
  Then the root is not scanned and the child is named in a warning

Scenario: 792b30c2a5 A child that cannot be inspected marks the root not scanned
  Given a child entry whose inspection fails
  When the root is scanned
  Then the root is not scanned
```

### 792b30c3 Claude Code user skills

Requirement: [792b30c3 Claude Code user skills](#792b30c3-claude-code-user-skills) The skill-source-discovery MUST find the skills of a Claude Code skills root as its immediate children that are directories, or symbolic
links resolving to directories, containing `SKILL.md`, each with its own bytes and resolved path and the
scope `claude-user`, MUST resolve a root that is itself a link, and MUST ignore root files, `synced`,
dangling links and directories without `SKILL.md`.

Supports: [792a12j1a A symlinked user skill is found](../product/skill-runbook-registration.md#792a12j1-uat-scenarios), [792a12j1b A dangling symlink is skipped](../product/skill-runbook-registration.md#792a12j1-uat-scenarios).

#### 792b30c3a Scenarios

```gherkin
Scenario: 792b30c3a1 A symlinked skill is found
  Given a skills directory child that is a link to a directory holding SKILL.md
  When the root is scanned
  Then the skill is emitted with its resolved path

Scenario: 792b30c3a2 A symlinked root is resolved
  Given a skills root that is itself a symbolic link
  When it is scanned
  Then the root is recorded resolved and its skills are found

Scenario: 792b30c3a3 Non-skill entries are skipped
  Given root files, synced, a dangling link and a directory without SKILL.md
  When the root is scanned
  Then none is emitted and the root is scanned

Scenario: 792b30c3a4 Children are exactly the skills built
  Given any mix of child shapes
  When the root is scanned
  Then exactly the children built as skills are emitted, each with its own bytes and resolved path

Scenario: 792b30c3a5 A dangling link is skipped
  Given a child that is a link whose target is missing
  When the root is scanned
  Then it is skipped without failing
```

### 792b30c4 Command directories

Requirement: [792b30c4 Command directories](#792b30c4-command-directories) The skill-source-discovery MUST find the commands of a commands root as the `.md` files beneath it, recursively, named by their path
relative to the root with `/` replaced by `:` and `.md` removed, MUST follow symbolic links, MUST skip
non-markdown files, MUST end on a link cycle and give up past a depth limit, MUST skip with a warning
any entry whose name already contains `:`, and MUST mark the root not scanned when a subdirectory cannot
be read.

Supports: [792a12j1c A user command in a subdirectory](../product/skill-runbook-registration.md#792a12j1-uat-scenarios).

#### 792b30c4a Scenarios

```gherkin
Scenario: 792b30c4a1 Names follow symlinks and skip non-markdown
  Given a tree with a linked directory, markdown files and other files
  When the root is scanned
  Then the markdown commands are named by relative path with colons, through links, and other files are skipped

Scenario: 792b30c4a2 Names are the path with colons
  Given any tree of markdown files plus other noise
  When the root is scanned
  Then the names are exactly the relative paths with / replaced by : and .md removed

Scenario: 792b30c4a3 A link cycle ends
  Given a directory link that points at its ancestor
  When the root is scanned
  Then the scan terminates

Scenario: 792b30c4a4 A missing root is not scanned
  Given a commands root that does not exist
  When it is scanned
  Then it is recorded as not scanned

Scenario: 792b30c4a5 A subdirectory that cannot be read marks the root not scanned
  Given a subdirectory whose listing fails
  When the root is scanned
  Then the root is not scanned

Scenario: 792b30c4a6 A name with a colon is skipped
  Given a command file whose name contains :
  When the root is scanned
  Then it is skipped with a warning
```

### 792b30c5 Synced skill buckets

Requirement: [792b30c5 Synced skill buckets](#792b30c5-synced-skill-buckets) The skill-source-discovery MUST find the claude.ai-synced skills as the skill children of each directory child ("bucket") of the synced
root whose `manifest.json` parses as JSON, MUST record every bucket as its own root, scanned only when
its manifest parsed and it was read, MUST ignore non-directory children, MUST emit a skill held by two
buckets once for each, and MUST count the synced root as scanned only when at least one manifest parsed.

Supports: [792a12j1d Synced skills come from manifest buckets](../product/skill-runbook-registration.md#792a12j1-uat-scenarios).

#### 792b30c5a Scenarios

```gherkin
Scenario: 792b30c5a1 A manifest bucket is scanned and a stray file ignored
  Given a synced root with one bucket whose manifest parses and a stray file .bucket-x
  When it is scanned
  Then the bucket's skills are emitted and the stray file is ignored

Scenario: 792b30c5a2 A bucket with a bad manifest contributes nothing
  Given a bucket whose manifest.json does not parse
  When it is scanned
  Then it yields no skill and is recorded as not scanned

Scenario: 792b30c5a3 A skill in two buckets is emitted twice
  Given two buckets that both hold pdf/SKILL.md
  When the root is scanned
  Then both candidates are emitted

Scenario: 792b30c5a4 A missing synced root is not scanned
  Given no synced directory
  When it is scanned
  Then it is recorded as not scanned

Scenario: 792b30c5a5 No parsed manifest means the root is not scanned
  Given buckets none of whose manifests parse
  When the synced root is scanned
  Then the synced root is not scanned

Scenario: 792b30c5a6 Candidates come only from parseable buckets
  Given any mix of buckets with parseable, unparseable and missing manifests, and stray files
  When the root is scanned
  Then candidates come only from parseable buckets, each bucket's flag equals "manifest parsed", and the synced root is scanned iff at least one parsed
```

### 792b30c6 Plugin command paths

Requirement: [792b30c6 Plugin command paths](#792b30c6-plugin-command-paths) The skill-source-discovery MUST find a plugin's commands from its `commands/` directory when `plugin.json` declares no `commands` field,
from a declared string path (a directory, or one `.md` file named by its stem) or an array of such paths
in place of that directory, MUST skip with a warning a declared object map, MUST refuse a declared path
that is missing, unreadable or a file that is not `.md`, recording a root that was not scanned, and MUST
skip a declared file whose name contains `:`.

Supports: [792a12l1a plugin.json commands replace the commands directory](../product/skill-runbook-registration.md#792a12l1-uat-scenarios).

#### 792b30c6a Scenarios

```gherkin
Scenario: 792b30c6a1 An absent field uses the commands directory
  Given a plugin whose plugin.json has no commands field
  When its commands are scanned
  Then the commands directory is scanned

Scenario: 792b30c6a2 A string path names a directory
  Given a plugin.json commands string naming a directory
  When its commands are scanned
  Then that directory is scanned

Scenario: 792b30c6a3 An array replaces the commands directory
  Given a plugin.json commands array and a commands/ directory with other files
  When its commands are scanned
  Then only the declared paths are scanned

Scenario: 792b30c6a4 An object map is skipped with a warning
  Given a plugin.json commands object
  When its commands are scanned
  Then it is skipped with a warning

Scenario: 792b30c6a5 A missing declared path is not scanned
  Given a declared path that does not exist
  When its commands are scanned
  Then the root is recorded as not scanned

Scenario: 792b30c6a6 A declared file that cannot be read is not scanned
  Given a declared command file whose read fails
  When its commands are scanned
  Then the root is not scanned

Scenario: 792b30c6a7 A declared file with a colon is skipped
  Given a declared file whose name contains :
  When its commands are scanned
  Then it is skipped with a warning
```

### 792b30c7 Claude Code plugins

Requirement: [792b30c7 Claude Code plugins](#792b30c7-claude-code-plugins) The skill-source-discovery MUST read `installed_plugins.json` (a missing file silently unread, any other failure a warning) and the
`enabledPlugins` map of `settings.json` (a missing file meaning every plugin falls back to its
manifest's `defaultEnabled`, an unreadable or unparseable one making enablement unknown), and for each
plugin name choose the most specific install that applies, the scope `user` or a `project` or `local`
scope whose `projectPath` equals the repository top-level (both symlink-resolved). It MUST scan an
enabled plugin's `skills/` directory plus any `plugin.json` `skills` paths (a folder holding `SKILL.md`
being one skill named by the folder, any other directory scanned for skill folders) and its commands
([792b30c6 Plugin command paths](#792b30c6-plugin-command-paths)), record whether each plugin was scanned, and read nothing beyond the manifests of a disabled
plugin. It MUST refuse a declared path that is absolute, leaves the plugin lexically or through a
symlink, or does not exist, with a warning and the plugin not scanned. It MUST treat a non-boolean
`enabled` value as not scanned, MUST report a plugin name installed from more than one marketplace as a
conflict and emit none of its candidates, and MUST skip with a warning an enabled plugin whose name is
reserved (`claude`, `pi`, `agents`, `project`, `anthropic-skills`, `pi-settings`, `pi-pkg`, `pi-prompt`)
or contains `:`, saying nothing for a disabled one and treating `cmd` as an ordinary name.

Supports: [792a12l1a plugin.json commands replace the commands directory](../product/skill-runbook-registration.md#792a12l1-uat-scenarios), [792a12l1b A disabled plugin contributes nothing](../product/skill-runbook-registration.md#792a12l1-uat-scenarios), [792a12l1c A plugin without an enabledPlugins entry and without an install path](../product/skill-runbook-registration.md#792a12l1-uat-scenarios), [792a12l1d Only the installed version is read](../product/skill-runbook-registration.md#792a12l1-uat-scenarios), [792a12o1g A plugin named claude is reserved](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12o1h A disabled reserved-name plugin is silent](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12o1i A plugin named cmd is an ordinary plugin](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12p1d A plugin name in two marketplaces](../product/skill-runbook-registration.md#792a12p1-uat-scenarios).

#### 792b30c7a Scenarios

```gherkin
Scenario: 792b30c7a1 The machine's plugin shape
  Given an installed_plugins.json, settings.json and cache like a real machine
  When plugins are scanned
  Then the enabled plugins' skills and commands are found and the facts are reported

Scenario: 792b30c7a2 plugin.json commands replace the commands directory
  Given a plugin whose plugin.json commands lists one file and whose commands/ also holds another
  When plugins are scanned
  Then only the listed command is scanned

Scenario: 792b30c7a3 plugin.json skills add to the default
  Given a plugin whose plugin.json skills names an extra path
  When plugins are scanned
  Then the skills/ directory and the extra path are both scanned, a repeated path once

Scenario: 792b30c7a4 Declared paths stay inside the plugin
  Given an absolute path, a .. escape and a path whose links leave the resolved installPath
  When plugins are scanned
  Then each is refused with a warning and the plugin is not scanned

Scenario: 792b30c7a5 A disabled plugin reads nothing beyond manifests
  Given a plugin disabled by enabledPlugins
  When plugins are scanned
  Then no skill or command directory of it is read

Scenario: 792b30c7a6 Enablement follows the setting and the default
  Given any combination of enabledPlugins values and defaultEnabled
  When plugins are scanned
  Then a plugin is scanned exactly when enabled by its entry or, without an entry, by defaultEnabled

Scenario: 792b30c7a7 Manifest failures are handled
  Given an installed_plugins.json or settings.json that is missing, unreadable or unparseable
  When plugins are scanned
  Then a missing installed_plugins.json is silent, other failures warn, and the facts say the manifests were not read

Scenario: 792b30c7a8 The most specific scope wins
  Given a plugin installed at user scope and at project scope for this repository
  When plugins are scanned
  Then the install for this repository is the one scanned

Scenario: 792b30c7a9 A non-boolean enabled value is not scanned
  Given an enabledPlugins value that is not a boolean
  When plugins are scanned
  Then a warning is added and the plugin is not scanned

Scenario: 792b30c7a10 Reserved and colon names are skipped
  Given an enabled plugin named claude and one whose name contains :
  When plugins are scanned
  Then both are skipped with a warning

Scenario: 792b30c7a11 The scope rule holds for any installs
  Given any set of installs with user, project and local scopes and project paths
  When plugins are scanned
  Then a plugin applies exactly when its scope is user or its project path is this repository

Scenario: 792b30c7a12 Unscannable contents make the plugin not scanned
  Given a plugin whose skills directory cannot be read
  When plugins are scanned
  Then the plugin is reported not scanned

Scenario: 792b30c7a13 A plugin from two marketplaces is a conflict
  Given plugins tools@alpha and tools@beta
  When plugins are scanned
  Then a conflict naming both marketplaces is reported and none of tools' candidates is emitted

Scenario: 792b30c7a14 The reserved-name check follows enablement
  Given an enabled plugin claude@m, a disabled pi@m and an enabled cmd@m
  When plugins are scanned
  Then claude is skipped with a warning, pi contributes nothing silently, and cmd is scanned as an ordinary plugin
```

### 792b30c8 Pi skill directories

Requirement: [792b30c8 Pi skill directories](#792b30c8-pi-skill-directories) The skill-source-discovery MUST discover Pi skills in a skills directory recursively: a directory that holds `SKILL.md` and has no
`SKILL.md`-holding ancestor below the root MUST be one skill named by the directory, root `.md` files
MUST be skills where the source takes them (the Pi user and settings roots, and a package root) and
ignored where it does not (`~/.agents/skills`), hidden directories and `node_modules` MUST be skipped,
links MUST be followed, a link cycle MUST end, and going past the depth limit or failing to read a
directory, a child or a skill file MUST mark the root not scanned. A root that itself holds `SKILL.md`
MUST be one skill.

Supports: [792a12k1g A Pi root markdown skill](../product/skill-runbook-registration.md#792a12k1-uat-scenarios).

#### 792b30c8a Scenarios

```gherkin
Scenario: 792b30c8a1 Recursive discovery and root markdown
  Given a Pi skills root with nested skill directories and a root notes.md
  When the root is scanned
  Then the skill directories and notes are found

Scenario: 792b30c8a2 Links are followed
  Given engram's skills as links into ~/.pi/agent/engram/skills and a dangling link
  When the root is scanned
  Then the linked skills are found and the dangling link skipped

Scenario: 792b30c8a3 Hidden directories and node_modules are skipped
  Given a hidden directory and a node_modules directory holding SKILL.md
  When the root is scanned
  Then neither is scanned

Scenario: 792b30c8a4 A root holding SKILL.md is one skill
  Given a skills root that itself holds SKILL.md
  When it is scanned
  Then it is one skill

Scenario: 792b30c8a5 Discovery finds exactly the skill directories
  Given any tree of directories, some holding SKILL.md, plus markdown files
  When the root is scanned
  Then exactly the directories holding SKILL.md with no SKILL.md-holding ancestor below the root are found, named by directory

Scenario: 792b30c8a6 A link cycle ends
  Given a directory link to an ancestor
  When the root is scanned
  Then the scan terminates

Scenario: 792b30c8a7 A depth past the limit marks the root not scanned
  Given a tree deeper than the limit
  When the root is scanned
  Then the root is not scanned

Scenario: 792b30c8a8 A child that cannot be inspected marks the root not scanned
  Given a child whose inspection fails
  When the root is scanned
  Then the root is not scanned

Scenario: 792b30c8a9 A skill file that cannot be read marks the root not scanned
  Given a SKILL.md whose read fails
  When the root is scanned
  Then the root is not scanned

Scenario: 792b30c8a10 A subdirectory that cannot be read marks the root not scanned
  Given a subdirectory whose listing fails
  When the root is scanned
  Then the root is not scanned

Scenario: 792b30c8a11 A missing root is not scanned
  Given a Pi skills root that does not exist
  When it is scanned
  Then it is recorded as not scanned

Scenario: 792b30c8a12 The agents root ignores root markdown
  Given ~/.agents/skills holding a root .md file
  When it is scanned
  Then the root file is ignored
```

### 792b30c9 Pi prompt directories

Requirement: [792b30c9 Pi prompt directories](#792b30c9-pi-prompt-directories) The skill-source-discovery MUST find Pi prompt templates as the non-hidden `.md` files directly in the prompts directory, regular or
linked, named by their stem, MUST NOT descend into subdirectories, MUST record a missing directory as
not scanned, and MUST mark the root not scanned when a prompt cannot be read.

Supports: [792a12k1a A Pi prompt template](../product/skill-runbook-registration.md#792a12k1-uat-scenarios).

#### 792b30c9a Scenarios

```gherkin
Scenario: 792b30c9a1 A global prompt is named by its stem
  Given ~/.pi/agent/prompts/review.md
  When prompts are scanned
  Then review is found

Scenario: 792b30c9a2 A missing directory is not scanned
  Given no prompts directory
  When prompts are scanned
  Then it is recorded as not scanned

Scenario: 792b30c9a3 An unreadable prompt marks the root not scanned
  Given a prompt whose read fails
  When prompts are scanned
  Then the root is not scanned

Scenario: 792b30c9a4 Only flat markdown files are prompts
  Given any flat mix of entries
  When prompts are scanned
  Then exactly the non-hidden .md files are found and subdirectories are never descended
```

### 792b30c10 Pi project trust

Requirement: [792b30c10 Pi project trust](#792b30c10-pi-project-trust) The skill-source-discovery MUST decide that a Pi project is trusted when the nearest non-null saved decision, for the canonical working
directory or an ancestor, in `~/.pi/agent/trust.json` is true, or, when there is none, when
`defaultProjectTrust` in Pi's global `settings.json` is `always`; MUST treat any other fallback as
untrusted; MUST canonicalize a working directory reached through a symlink; and MUST treat an unreadable
or invalid file as untrusted with a warning.

Supports: [792a12m1b Trust falls back to defaultProjectTrust always](../product/skill-runbook-registration.md#792a12m1-uat-scenarios), [792a12m1c An untrusted Pi project is not scanned](../product/skill-runbook-registration.md#792a12m1-uat-scenarios).

#### 792b30c10a Scenarios

```gherkin
Scenario: 792b30c10a1 Always trusts by default
  Given no saved decision and defaultProjectTrust always
  When trust is resolved
  Then the project is trusted

Scenario: 792b30c10a2 Another fallback is untrusted
  Given no saved decision and defaultProjectTrust ask or absent
  When trust is resolved
  Then the project is untrusted

Scenario: 792b30c10a3 A saved decision decides
  Given trust.json holding a decision for the working directory
  When trust is resolved
  Then the decision applies and settings.json is not read

Scenario: 792b30c10a4 Invalid files are untrusted with a warning
  Given a trust.json or settings.json that cannot be read or parsed
  When trust is resolved
  Then the project is untrusted and a warning is added

Scenario: 792b30c10a5 The nearest decision wins
  Given any assignment of true, false, null and absent decisions to the directory and its ancestors, and any defaultProjectTrust
  When trust is resolved
  Then the project is trusted iff the nearest non-null decision is true or there is none and the default is always

Scenario: 792b30c10a6 A symlinked working directory is canonicalized
  Given a working directory reached through a link
  When trust is resolved
  Then decisions are looked up for its canonical path
```

### 792b30c11 Pi project sources

Requirement: [792b30c11 Pi project sources](#792b30c11-pi-project-sources) The skill-source-discovery MUST scan, only for a trusted project, `.pi/skills` and `.pi/prompts` of the working directory, and the
`.agents/skills` directory of the working directory and each ancestor up to the repository top-level,
each existing level recorded as its own root, and MUST read nothing and record no root for an untrusted
project. It MUST skip the user's `~/.agents/skills` as a project level when the chain reaches the home
directory, MUST resolve a working directory reached through a symlink so that the walk still meets and
never passes the top-level, MUST emit the same name at two levels once for each, and MUST stamp the
source segment `pi`, `agents` or `pi-prompt` on each candidate to tell them apart inside one project
scope.

Supports: [792a12m1a Agents skills in an ancestor](../product/skill-runbook-registration.md#792a12m1-uat-scenarios), [792a12m1c An untrusted Pi project is not scanned](../product/skill-runbook-registration.md#792a12m1-uat-scenarios).

#### 792b30c11a Scenarios

```gherkin
Scenario: 792b30c11a1 A trusted project scans the working directory's Pi skills
  Given a trusted project with .pi/skills/foo/SKILL.md
  When Pi project skills are scanned
  Then foo is found

Scenario: 792b30c11a2 A trusted project scans its Pi prompts
  Given a trusted project with .pi/prompts/x.md
  When Pi project prompts are scanned
  Then x is found

Scenario: 792b30c11a3 An untrusted project reads nothing
  Given an untrusted project
  When Pi project sources are scanned
  Then nothing is read and no root is recorded

Scenario: 792b30c11a4 The agents chain is found from a subdirectory
  Given a top-level .agents/skills and a working directory below it
  When agents project skills are scanned
  Then the top-level's skills are found

Scenario: 792b30c11a5 The agents chain finds exactly the levels that hold skills
  Given any depth and any set of levels holding .agents/skills
  When agents project skills are scanned
  Then exactly those levels' skills are found, each existing level its own root

Scenario: 792b30c11a6 The user agents root is not a project level
  Given a chain that reaches the home directory
  When agents project skills are scanned
  Then ~/.agents/skills is the user source and not a project level

Scenario: 792b30c11a7 A symlinked working directory stops at the top-level
  Given a working directory reached through a link
  When agents project skills are scanned
  Then the walk meets the physical top-level and never climbs past it

Scenario: 792b30c11a8 Candidates carry their source segment
  Given project candidates from Pi skills, agents skills and prompts
  When they are scanned
  Then their segments are pi, agents and pi-prompt

Scenario: 792b30c11a9 Agents and Pi project sources need a trusted project only
  Given a trusted project
  When Pi project sources are scanned
  Then both the .pi and .agents levels are scanned
```

### 792b30c12 Pi settings reading and overrides

Requirement: [792b30c12 Pi settings reading and overrides](#792b30c12-pi-settings-reading-and-overrides) The skill-source-discovery MUST read Pi's `settings.json` (resolving its path first, accepting the legacy object form of `skills`, and
giving an error for an unreadable or invalid file), and apply a settings file's overrides to the
candidates of a default folder: `!pattern` MUST exclude by name (the walked path relative to the
settings base directory, the file or skill-folder name, or the absolute path), and `+path` MUST
force-include and `-path` force-exclude by exact path (absolute, or relative to the settings base
directory), with the result enabled when not excluded, then forced in, then not forced out. A plain
entry (one with no `!`, `+` or `-` prefix) MUST NOT filter a default folder: only the override entries
act. It MUST match the path as walked (a linked skill by its link location), fall back to the source
path, match a prompt by file name, and disable every candidate with one warning for a pattern it cannot
evaluate (`**`).

Supports: [792a12k1b A Pi settings skill path](../product/skill-runbook-registration.md#792a12k1-uat-scenarios), [792a12k1f An object-form empty skills filter](../product/skill-runbook-registration.md#792a12k1-uat-scenarios).

#### 792b30c12a Scenarios

```gherkin
Scenario: 792b30c12a1 Settings are read from the resolved file
  Given a settings path that is a link
  When it is read
  Then the resolved file is read

Scenario: 792b30c12a2 The legacy skills object and errors
  Given a skills value in the legacy object form, and an invalid file
  When settings are read
  Then the object form is accepted and an invalid file is an error

Scenario: 792b30c12a3 Overrides disable default-folder skills
  Given a settings !pattern that names a skill
  When the overrides are applied
  Then that skill is disabled

Scenario: 792b30c12a4 The override rule holds for names and exact paths
  Given any excludes and exact + and - paths
  When the overrides are applied
  Then enabled = not excluded, then forced in, then not forced out

Scenario: 792b30c12a5 The source path is the fallback
  Given a candidate with no walked path
  When the overrides are applied
  Then its source path is matched

Scenario: 792b30c12a6 A globstar disables all with a warning
  Given an override ** that cannot be evaluated
  When the overrides are applied
  Then every candidate is disabled and one warning is added

Scenario: 792b30c12a7 The walked path is matched, not the resolved one
  Given a linked skill
  When the overrides are applied
  Then it is matched by its link location

Scenario: 792b30c12a8 Prompts are matched by file name
  Given a prompt candidate and an override naming its file
  When the overrides are applied
  Then it is matched by its file name
```

### 792b30c13 Pi configured sources

Requirement: [792b30c13 Pi configured sources](#792b30c13-pi-configured-sources) The skill-source-discovery MUST scan the `skills` and `prompts` entries and the `packages` of Pi's global settings, and, for a trusted
project, of the project's, resolving each against the right base: a directory entry MUST be searched
recursively with its own root `.md` files, a file entry MUST be one skill, an entry that does not exist
MUST be recorded not scanned without a warning, an entry with a pattern that cannot be evaluated or
marked `autoload: false` MUST NOT be scanned, and a prompt tree beyond a depth limit MUST mark the root
not scanned. A package MUST resolve to its `npm:` root under `~/.pi/agent/npm/node_modules`, its `git:`
root, or its local root. A string-form entry whose `package.json` has any `pi` key MUST load only the
manifest's `pi.skills` and `pi.prompts` entries (globs honored, a globstar not scanned, an unreadable
manifest not scanned) and use the convention directories only when it has no `pi` key; an object-form
entry's explicit list MUST filter that type with `[]` switching it off, an omitted key MUST fall back
for that type only, and a package's disabled files MUST be kept disabled. A project package MUST
override a global one of the same identity, so that exactly one root per identity is scanned, and
project settings MUST need a trusted project.

Supports: [792a12k1b A Pi settings skill path](../product/skill-runbook-registration.md#792a12k1-uat-scenarios), [792a12k1c Package skills come from its manifest](../product/skill-runbook-registration.md#792a12k1-uat-scenarios), [792a12k1d A string-form package with a pi key ignores convention directories](../product/skill-runbook-registration.md#792a12k1-uat-scenarios), [792a12k1e An object-form omitted type falls back per type](../product/skill-runbook-registration.md#792a12k1-uat-scenarios), [792a12k1f An object-form empty skills filter](../product/skill-runbook-registration.md#792a12k1-uat-scenarios), [792a12k1h A configured package with a missing path contributes nothing](../product/skill-runbook-registration.md#792a12k1-uat-scenarios).

#### 792b30c13a Scenarios

```gherkin
Scenario: 792b30c13a1 A settings skill path
  Given an entry naming the skills folder, and one naming the skill folder itself
  When settings are scanned
  Then the skill is found either way

Scenario: 792b30c13a2 A directory entry includes its root markdown
  Given a settings directory entry holding root .md files
  When settings are scanned
  Then they are skills of the entry

Scenario: 792b30c13a3 File entries are one skill each
  Given settings entries that are files
  When settings are scanned
  Then each is one skill

Scenario: 792b30c13a4 Git and local package roots
  Given a git package and a local package
  When settings are scanned
  Then their roots resolve to the git checkout and the local path

Scenario: 792b30c13a5 Autoload false is not scanned
  Given a package marked autoload false
  When settings are scanned
  Then it is not scanned

Scenario: 792b30c13a6 A globstar pattern is not scanned
  Given an entry whose pattern is **
  When settings are scanned
  Then it is not scanned

Scenario: 792b30c13a7 The machine's shape
  Given a local package path that does not exist and an installed but unconfigured package
  When settings are scanned
  Then nothing is found and the missing package is recorded not scanned

Scenario: 792b30c13a8 Manifest globs and overrides
  Given a manifest pi.skills with globs and overrides
  When settings are scanned
  Then matching skills are found and overrides applied

Scenario: 792b30c13a9 A manifest globstar is not scanned
  Given a manifest entry **
  When settings are scanned
  Then it is not scanned

Scenario: 792b30c13a10 A missing entry is not scanned and silent
  Given an entry whose path does not exist
  When settings are scanned
  Then it is recorded not scanned and no warning is added

Scenario: 792b30c13a11 Without a pi key the convention directories are used
  Given a package.json with no pi key
  When settings are scanned
  Then skills/ and prompts/ are scanned

Scenario: 792b30c13a12 An npm manifest lists skills
  Given "npm:pi-intercom" with pi.skills ["./skills"]
  When settings are scanned
  Then pi-intercom is found as a skill of that package

Scenario: 792b30c13a13 Object-form filters
  Given a string-form package with a pi key, an object form omitting a type, and an object form with skills []
  When settings are scanned
  Then the string form ignores convention directories, the omitted type falls back for that type only, and [] switches skills off keeping them disabled

Scenario: 792b30c13a14 A pattern filter follows Pi's rule
  Given plain globs, !, + and - in one settings directory
  When settings are scanned
  Then plain globs include by name (all when none), ! excludes by name, + re-adds by exact path and - removes by exact path

Scenario: 792b30c13a15 Project packages override global ones
  Given global and project package lists naming one identity
  When settings are scanned
  Then the project's root is the one scanned

Scenario: 792b30c13a16 The override holds for any lists
  Given any global and project npm package lists
  When settings are scanned
  Then exactly one root per identity is scanned and the project's wins

Scenario: 792b30c13a17 Project settings need trust
  Given project settings and an untrusted project
  When settings are scanned
  Then the project settings are not read

Scenario: 792b30c13a18 A prompt tree has a depth limit
  Given a prompt tree deeper than the limit
  When settings are scanned
  Then the root is not scanned

Scenario: 792b30c13a19 Prompt tree edge cases
  Given hidden and nested prompt entries
  When settings are scanned
  Then they are handled as Pi handles them

Scenario: 792b30c13a20 A settings prompt directory is recursive
  Given a settings prompts directory with nested files
  When settings are scanned
  Then the nested prompts are found

Scenario: 792b30c13a21 An unreadable child marks its entry not scanned
  Given a child of an entry whose read fails
  When settings are scanned
  Then the entry is not scanned

Scenario: 792b30c13a22 An unreadable manifest is not scanned
  Given a package.json that cannot be read
  When settings are scanned
  Then the package is not scanned
```

### 792b30c14 Project identity

Requirement: [792b30c14 Project identity](#792b30c14-project-identity) The skill-source-discovery MUST identify the project of a working directory with `git`: inside a git work tree its identity MUST be the
origin remote normalized to `<host>/<path>`, which is the host and path lowercased, with the scheme,
userinfo, port, a trailing `.git` and a trailing `/` removed (every variant of one remote giving the
same value, distinct forges staying distinct, and normalization idempotent), or, without an origin,
`local/` followed by the basename of the parent of the absolute `git rev-parse --git-common-dir`; and
outside a work tree there MUST be no project. A linked worktree MUST take its origin from the
repository. A fallback that cannot be used MUST give no project.

Supports: [792a12o1c A worktree resolves to owner and repository](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12o1d Two same-named projects stay distinct](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12m1f Outside a repository there are no project entries](../product/skill-runbook-registration.md#792a12m1-uat-scenarios).

#### 792b30c14a Scenarios

```gherkin
Scenario: 792b30c14a1 Variants of one remote normalize alike
  Given any scheme, userinfo, port, case and trailing-suffix variant of one remote
  When it is normalized
  Then every variant gives the same <host>/<path> and normalizing again changes nothing

Scenario: 792b30c14a2 Examples
  Given ssh://git@GitHub.com:22/toejough/engram.git and https://gitlab.com/toejough/engram
  When they are normalized
  Then github.com/toejough/engram and gitlab.com/toejough/engram

Scenario: 792b30c14a3 Distinct forges stay distinct
  Given the same owner and repository on two hosts
  When they are normalized
  Then two different identities

Scenario: 792b30c14a4 A linked worktree uses the origin
  Given a linked worktree directory named runbook-vs-skill
  When its project is probed
  Then the identity is the origin's, not the worktree name

Scenario: 792b30c14a5 Without an origin the common directory names the project
  Given a repository with no origin remote
  When its project is probed
  Then the identity is local/ and the basename of the parent of the common directory

Scenario: 792b30c14a6 A directory outside a repository has no project
  Given a working directory outside any git work tree
  When its project is probed
  Then there is no project

Scenario: 792b30c14a7 Unusable fallbacks give no project
  Given a common directory that cannot be used
  When its project is probed
  Then there is no project
```

### 792b30c15 Claude Code project sources

Requirement: [792b30c15 Claude Code project sources](#792b30c15-claude-code-project-sources) The skill-source-discovery MUST scan `.claude/skills` and `.claude/commands` in the working directory and each ancestor up to the
repository top-level, each existing level recorded as its own root and unreadable levels marked not
scanned, MUST emit the same name at two levels once for each, MUST scan nothing when the working
directory is outside the top-level, MUST skip the user's own roots when the top-level is the home
directory, MUST resolve a working directory reached through a symlink, and MUST ignore root `.md` files
in `.claude/skills`.

Supports: [792a12m1d Claude project sources from a subdirectory](../product/skill-runbook-registration.md#792a12m1-uat-scenarios), [792a12m1e Root markdown in the project skills directory is ignored](../product/skill-runbook-registration.md#792a12m1-uat-scenarios).

#### 792b30c15a Scenarios

```gherkin
Scenario: 792b30c15a1 The chain finds exactly the levels that hold sources
  Given any repository depth, working directory level and set of levels holding skills or commands, a same-named skill at several levels and a .claude directory above the top-level
  When project sources are scanned
  Then exactly the levels from the working directory up to the top-level are scanned

Scenario: 792b30c15a2 A working directory outside the top-level scans nothing
  Given a working directory outside the repository top-level
  When project sources are scanned
  Then nothing is scanned

Scenario: 792b30c15a3 The same name at two levels is emitted twice
  Given foo at the top-level and at sub/
  When project sources are scanned
  Then both candidates are emitted

Scenario: 792b30c15a4 The user roots are skipped when the top-level is home
  Given a top-level equal to the home directory
  When project sources are scanned
  Then ~/.claude/skills and ~/.claude/commands are not project levels

Scenario: 792b30c15a5 A subdirectory equals the top-level
  Given a working directory that is the top-level
  When project sources are scanned
  Then the same entries are found as from a subdirectory

Scenario: 792b30c15a6 A symlinked working directory meets the top-level
  Given a working directory reached through a link
  When project sources are scanned
  Then the walk meets the physical top-level

Scenario: 792b30c15a7 An unreadable level is not scanned
  Given a level whose listing fails
  When project sources are scanned
  Then that level is recorded not scanned
```

### 792b30c16 Source composition and precedence

Requirement: [792b30c16 Source composition and precedence](#792b30c16-source-composition-and-precedence) The skill-source-discovery MUST compose the default source set from the harnesses that exist: the Claude Code sources only when
`~/.claude` is detected, the Pi and `~/.agents` sources only when `~/.pi` is detected, using the
detection of harness-deploy-sync ([792b29c1 Harness detection](./harness-deploy-sync.md#792b29c1-harness-detection)) through links, and the project sources only inside a git
repository, and MUST return the candidates in this precedence: Claude user skills, Claude user commands,
synced, Pi user, agents user, Pi settings skills, Pi prompt templates (default folder, then settings),
Pi packages, Claude plugins, Claude project, then the Pi project sources in the same order as the global
ones. It MUST apply Pi's overrides to default-folder files and keep the disabled candidates, MUST pass a
plugin conflict through, MUST report the resolved home so that a `~`-relative `skill_source` can be
written when the home is a symlink, MUST give every candidate the key of [792b30c17 Skill keys](#792b30c17-skill-keys) and drop one that
cannot be keyed with a warning, and MUST fail only when a harness probe fails for a reason other than
not-exist.

Supports: [792a12n1a A machine without Pi reads no Pi sources](../product/skill-runbook-registration.md#792a12n1-uat-scenarios), [792a12o1e Engram-installed skills are keyed by their folder](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12m1f Outside a repository there are no project entries](../product/skill-runbook-registration.md#792a12m1-uat-scenarios), [792a12m1c An untrusted Pi project is not scanned](../product/skill-runbook-registration.md#792a12m1-uat-scenarios).

#### 792b30c16a Scenarios

```gherkin
Scenario: 792b30c16a1 Absent harness sources are not read
  Given no ~/.pi
  When sources are resolved
  Then no Pi or agents source is read

Scenario: 792b30c16a2 Every source is composed in precedence order
  Given every source present
  When sources are resolved
  Then candidates appear in the stated precedence

Scenario: 792b30c16a3 A probe error fails
  Given a harness probe that fails for a reason other than not-exist
  When sources are resolved
  Then resolution fails

Scenario: 792b30c16a4 Outside a repository there are no project sources
  Given a working directory outside any git repository
  When sources are resolved
  Then no project source is read

Scenario: 792b30c16a5 An override for an unknown level keeps the candidate with a warning
  Given a candidate whose agents level cannot be found
  When overrides are applied
  Then it is kept unchanged with a warning

Scenario: 792b30c16a6 Pi overrides disable default folders
  Given global and project settings overrides
  When sources are resolved
  Then the matching default-folder files are disabled and kept

Scenario: 792b30c16a7 A plugin conflict passes through
  Given a plugin installed from two marketplaces
  When sources are resolved
  Then the conflict is reported with the result

Scenario: 792b30c16a8 Precedence holds for any subset of sources
  Given whichever subset of sources exists
  When sources are resolved
  Then the rank of each candidate's source never decreases along the result

Scenario: 792b30c16a9 A project globstar disables the agents chain
  Given a project ** override
  When sources are resolved
  Then every candidate of each .agents/skills level is disabled with one warning per level

Scenario: 792b30c16a10 An untrusted project skips Pi project sources
  Given an untrusted project
  When sources are resolved
  Then no Pi project source is read

Scenario: 792b30c16a11 Pi default-folder candidates have a recorded read root
  Given real and symlinked roots at every chain level
  When Pi default-folder scanners run
  Then each candidate's read root is one of the result's recorded roots

Scenario: 792b30c16a12 A symlinked home keys Pi's link by its folder
  Given $HOME is a symlink and Pi's route resolves into engram's installed copy
  When sources are resolved
  Then the entry is keyed pi:route and the resolved home is reported

Scenario: 792b30c16a13 A Pi-only machine keys by folder and ignores a legacy note
  Given no ~/.claude and Pi's route under Pi's engram root
  When sources are resolved
  Then the key is pi:route and an unkeyed legacy note is no skill note

Scenario: 792b30c16a14 Keys are stamped and colon names dropped
  Given a candidate named a:b
  When sources are resolved
  Then it is dropped with a warning and the others are keyed

Scenario: 792b30c16a15 The default set resolves every folder
  Given a home and a working directory holding a Claude user skill, a command and the other folders
  When register-skills resolves its default set
  Then every folder is resolved from the home and working directory

Scenario: 792b30c16a16 A symlinked home round-trips skill_source
  Given a home reached through a link and accepted offers
  When register-skills runs twice
  Then each note's skill_source is home-relative and the second run offers nothing
```

### 792b30c17 Skill keys

Requirement: [792b30c17 Skill keys](#792b30c17-skill-keys) The skill-source-discovery MUST give each candidate the source-qualified key of [792a12o Every scanned entry has a source-qualified key](../product/skill-runbook-registration.md#792a12o-every-scanned-entry-has-a-source-qualified-key) from its scope, source segment, kind and name,
joined by `:`, MUST key a disabled candidate like any other, MUST skip with a warning a candidate whose
name contains `:` (a command's own namespace separators excepted), whose scope is unknown or reserved,
or whose key is not recognized, and MUST derive a note slug from a key as `skill-` followed by the key
lowercased with each run of characters outside `[a-z0-9]` replaced by `-` and leading and trailing `-`
trimmed, so that any key holding a letter or digit gives `skill-[a-z0-9]+(-[a-z0-9]+)*`. A recognized
key MUST contain `:`, have no empty segment and, when it begins with a reserved source prefix, have that
source's shape; any other first segment MUST be a plugin's.

Supports: [792a12o1a A plugin skill's key and slug](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12o1b A plugin's skill and command never collide](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12o1e Engram-installed skills are keyed by their folder](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12o1f A Claude user command's key](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12o1j A prompt template's key](../product/skill-runbook-registration.md#792a12o1-uat-scenarios), [792a12n1b A name with a colon is skipped](../product/skill-runbook-registration.md#792a12n1-uat-scenarios).

#### 792b30c17a Scenarios

```gherkin
Scenario: 792b30c17a1 The keys of every source
  Given candidates from every scope and kind
  When they are keyed
  Then each gets the key its source defines

Scenario: 792b30c17a2 A disabled candidate is keyed
  Given a disabled candidate
  When it is keyed
  Then it has a key like any other

Scenario: 792b30c17a3 Every key is qualified and parsed
  Given any generated candidates
  When they are keyed
  Then every key contains : and ends in :<name>, is recognized, and maps to exactly one removal form or plugin

Scenario: 792b30c17a4 Unkeyable candidates are skipped
  Given a candidate with a colon name, an unknown scope or an unrecognized key
  When they are keyed
  Then each is skipped with a warning

Scenario: 792b30c17a5 Claude user sources are qualified
  Given a Claude user skill, a command and a skill named cmd
  When they are keyed
  Then claude:<name>, claude:cmd:<ns:…:name> and claude:cmd

Scenario: 792b30c17a6 A key becomes a slug
  Given claude:route and claude:cmd:audit
  When slugs are derived
  Then skill-claude-route and skill-claude-cmd-audit

Scenario: 792b30c17a7 Slugging holds for any key
  Given any key with a slug character
  When its slug is derived
  Then it matches the note-slug shape, a shaped key maps to skill-<key>, and slugging is idempotent on the remainder
```
