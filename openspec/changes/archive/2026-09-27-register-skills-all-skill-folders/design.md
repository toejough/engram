## Context

`register-skills-as-runbooks` (archived 2026-09-26) built the offer flow: compare, prompt, then register, refresh, or remove, with declines remembered by hash. It only ever fed that flow engram's own skills. `loadShippedSkills` (`internal/cli/skillreg_run.go:234`) reads one directory's immediate `IsDir()` children, so every symlinked skill directory is skipped. The standalone default is `~/.claude/engram/skills` (`targets.go:379`, from `8a672df5`). The update hook reads `<sourceRoot>/agent-instructions/skills` (`update.go` `runUpdateSkillRegistration`). Identity is the bare skill name: slug `skill-<name>`, decline key `<name>`. A removal is offered for any skill note whose name is missing from the scanned set. That was safe only because the scanned set was the whole universe of skill notes.

Joe reversed the Non-Goal (vault note 1066a). Registration covers every default skill folder the harnesses load, and (review round 1) every command folder and Pi's configured skill sources too, and (review round 2) Pi prompt templates. The decline reader never checks `schema_version` (`skillreg.go:146-160`; it decodes `{schema_version, declined}` with `DisallowUnknownFields`).

**Joe's decision (round 3, final review, vault note 1073a): qualify every key with its source; no bare keys.** The implemented draft gave engram's own skills and Claude user skills one shared, unqualified namespace (`route`, `c4`). That namespace needed a special case in every rule that touched it:
- removal eligibility for a bare key spanned several roots (`engramCopiesRead`, `EngramSkillRootsUnresolved`; ruling R38's M1), so a merely missing `~/.agents/skills` blocked every bare-key removal (ruling R39, parked);
- a user-vs-engram collision rule (a real `~/.claude/skills/<n>` clashing with an engram-owned copy was a conflict, and the engram exception applied only when every copy was engram-owned; ruling R27(c));
- a legacy slug-remainder fallback that re-keyed an unkeyed note by its slug, and could re-key a note if `skill_key` were ever dropped (the U6 finding carried into U9).

**Joe's decision (round 3, second part): no engram-owned namespace or mechanism.** An `engram:<n>` owner namespace was drafted and declined (D3 alternative (f)). Skills that engram installs are treated like any other source: they are keyed by the folder they are found in (`claude:route` through `~/.claude/skills`, `pi:route` through `~/.pi/agent/skills`), and identical copies collapse by the ordinary alias rule (D4). The whole engram-owned mechanism is deleted:
- `ResolveEngramSkillRoots` and `ResolvedSkillSources.EngramSkillRoots`/`EngramSkillRootsUnresolved`;
- every engram-owned-root check (key construction, `offerEngramOwned`, `SkillNoteSource.EngramOwned`);
- ruling R26's engram-root warnings;
- the diverged-engram-copies exception in D4;
- the preamble special case in D8.

This revision deletes the special cases rather than relocating them. Claude user skills key as `claude:<n>`, and Claude user commands as `claude:cmd:<n>`. Each namespace has exactly one removal rule (D5). The cost is a one-time migration of Joe's six legacy notes to `claude:<n>` (D11), plus the fork cases listed in Risks, which are accepted. The previous change is on this same unmerged branch, so no other install has legacy notes. This supersedes ruling R39: its residual disappears.

**Harness rules (docs, checked 2026-09-26):**
- **Claude Code** (code.claude.com/docs/en/skills, plugins/manifest-reference, plugins/components; fetched by a docs agent):
  - Project skills load from `.claude/skills/` "in the directory where you start the session" and "every parent directory up to the repository root". Nested skills below cwd load lazily.
  - Commands are named by "subdirectory path (/ replaced by :) + filename". For example, `~/.claude/commands/opsx/apply.md` becomes `/opsx:apply`.
  - Plugin skills and commands are `<plugin>:<name>`, and a plugin command subdirectory adds a segment (`<plugin>:db:migrate`).
  - In `plugin.json`, `skills` **adds** paths to `skills/`, while `commands` (a path, an array, or an object map) **replaces** `commands/`.
  - "For plugin skills, `name` replaces the directory name in the command."
  - "Skills take precedence over `.claude/commands/` files."
  - `defaultEnabled` ("Whether the plugin starts enabled when the user hasn't set it in `enabledPlugins`. Defaults to `true`").
- **Pi** (`/opt/homebrew/lib/node_modules/@earendil-works/pi-coding-agent/docs/{skills,settings,packages}.md`):
  - Default locations are `~/.pi/agent/skills/` and `~/.agents/skills/` (global), and, for trusted projects only, `.pi/skills/` plus `.agents/skills/` in the cwd and its ancestors up to the git root.
  - `SKILL.md` directories are discovered recursively. Root `.md` files count as skills only in `~/.pi/agent/skills/` and `.pi/skills/`.
  - `settings.json` `skills: string[]` holds file or directory paths. They resolve relative to `~/.pi/agent`, or relative to `.pi` for project settings. `~`, globs, `!pattern`, `+path` and `-path` are supported. A glob entry is never a path: it filters the files the path entries found (`package-manager.js:127-129`, `:1890-1903`). The `!`/`+`/`-` entries also switch default-folder files on or off (`isEnabledByOverrides`, `:515-539`, applied at `:1949`).
  - Pi never drops a file that a settings or package filter excludes. It keeps it with `enabled: false` (`package-manager.js:1807`, `:1815`, `:1901`, `:1949-1950`), so the file is still installed but switched off.
  - `packages` entries are either a string or `{source, skills?: [...]}`:
    - `npm:<name>[@ver]` installs to `~/.pi/agent/npm/node_modules/<name>`, or `.pi/npm/` for a project.
    - `git:`/URL sources clone to `~/.pi/agent/git/<host>/<path>`, or `.pi/git/…`.
    - A local path resolves against its settings file.
    - A package's resources (checked in `dist/core/package-manager.js` `collectPackageResources`): for a string-form entry, any `pi` key in `package.json` means only the manifest's entries load, and the convention dirs (`skills/`, `prompts/`) apply only with no `pi` key at all. Only the object-form path falls back per resource type (manifest entry, else convention dir) when that type's filter is omitted. `[]` switches every file of that type off (`package-manager.js:1802-1817`); it does not remove them.
    - When a package appears in both the global and the project settings, the project entry wins.
  - Prompt templates (`docs/prompt-templates.md`) load from `~/.pi/agent/prompts/*.md`, from `.pi/prompts/*.md` for trusted projects, from `prompts/` directories or `pi.prompts` entries in a package, and from the `settings.json` `prompts` array. "The filename becomes the command name" (`prompt-templates.js:85`). Only the two default folders are flat (`collectAutoPromptEntries`, `package-manager.js:291-326`). A settings or package prompt directory is searched recursively (`collectFiles`, `:149-192`, reached via `:463`). Their frontmatter is optional `description`/`argument-hint`.
  - Trust is decided by the nearest saved decision for the folder or a parent in `~/.pi/agent/trust.json`. With no saved decision, the fallback is `defaultProjectTrust`, and only `always` trusts.

**Real layout on this machine (checked 2026-09-26):**

| Source | Location | Evidence | Loadable |
| --- | --- | --- | --- |
| Claude user skills | `~/.claude/skills/<n>/SKILL.md` | 33 entries: 22 dangling symlinks into `~/repos/personal/projctl/skills/`, `.obsidian/` (no SKILL.md), `synced/` | 10: c4, dev, mycelium, property-rigor + engram's six (symlinks into `~/.claude/engram/skills/<n>`) |
| Claude user commands | `~/.claude/commands/**/*.md` | `audit.md` only; the session's skill list shows it as `audit` | 1 |
| claude.ai synced | `~/.claude/skills/synced/<bucket>/manifest.json` + `<bucket>/<n>/SKILL.md` | one bucket dir `4227b111-…_e76eb279-…` holding `manifest.json` (12 skills, `source: anthropic`/`anthropic-example`) and a stray empty **file** `.bucket-4227b111-…`; the session list names them `anthropic-skills:<n>` | 12 |
| Pi user | `~/.pi/agent/skills/` | engram's six are symlinks into `~/.pi/agent/engram/skills/<n>` (a separate real copy, SHA-identical to Claude's), plus `ping/` | 7 (6 engram + ping) |
| agents user | `~/.agents/skills/` | absent | 0 |
| Pi settings `skills` | `~/.pi/agent/settings.json` | no `skills` key | 0 |
| Pi packages | `~/.pi/agent/settings.json` `packages` | one local entry `../../repos/personal/pi-skills/bare-skills`, which resolves to `~/repos/personal/pi-skills/bare-skills` and **does not exist**. `pi list` prints only that entry. `~/.pi/agent/npm/node_modules/pi-intercom` has `pi.skills: ["./skills"]` but is **not in settings**, so Pi does not load it | 0 |
| Pi prompt templates | `~/.pi/agent/prompts/`, settings `prompts`, package prompts, project `.pi/prompts` | `~/.pi/agent/prompts` is absent. Settings have no `prompts` key. The only configured package is missing. The engram repo has no `.pi/`. Only `~/repos/personal/{pi-skills,pi-lmstudio,pi-vim-mode,pi-auto-resume-on-compaction}/.pi/prompts` exist, and those are other projects | 0 (counts unchanged) |
| Claude plugins | `installPath` from `~/.claude/plugins/installed_plugins.json` (v2: `plugins["<plugin>@<marketplace>"][]` with `scope`, `installPath`) | `plugins/cache` holds up to 8 stale versions per plugin. `enabledPlugins`: hookify, ralph-loop and `engram@engram` are `false` (the last is installed with stale learn/recall/…). `work-on@work-on` has no `enabledPlugins` entry and its `installPath` is missing. No plugin name repeats across marketplaces. Only claude-hud's `plugin.json` declares `commands` (its two files) | 36 skills + 10 commands in enabled plugins |
| Claude project (this worktree) | `.claude/skills/<n>/SKILL.md`, `.claude/commands/**/*.md` | 6 `openspec-*` skills. Root `commit.md` and `engram-go-conventions.md` in `.claude/skills` are not listed by the harness. Commands: `commit.md` and `opsx/{apply,archive,explore,propose,sync,update}.md`, which the session lists as `commit` and `opsx:*`. The **main checkout** has `opsx/` without `update.md` | 13 (worktree), 12 (main checkout) |
| Pi project | `.pi/skills`, `.agents/skills`, `.pi/settings.json` | none in engram. `~/repos/personal/{pi-skills,pi-lmstudio,pi-vim-mode,pi-auto-resume-on-compaction}/.pi/skills` hold 6 each. `trust.json` trusts `/Users/joe/repos/personal` | 0 here |
| Git identity | worktree | `git remote get-url origin` = `ssh://git@github.com/toejough/engram.git`. `--show-toplevel` = the worktree dir. `--git-common-dir` = `/Users/joe/repos/personal/engram/.git` | — |

The real vault holds six skill notes: 1036 route, 1045 please, 1049 curate, 1053 write-memory, 1067 learn, 1068 recall. Their slugs are `skill-<n>`, their `skill_hash` values equal the deployed SHAs, they carry **no `skill_key`** (they predate keys), their preambles name `agent-instructions/skills/<n>/SKILL.md`, and there is no `skill-registrations.json`. Note 820 has a `skill-` slug but no `skill_hash`. `engram register-skills --dry-run` from `/tmp` printed nothing before this change.

## Goals / Non-Goals

**Goals:**
- One definition of the default source set, used by both standalone `register-skills` and the `engram update` hook.
- Every skill, command, and Pi prompt template a harness loads by default, or that Pi's settings configure, can be offered, with copies collapsed.
- A stable identity in which **every key is qualified by its source**, with no bare keys. Skills that engram installs get no special treatment: they are keyed by the folder they are found in, like any other skill.
- Each key namespace has exactly one removal-eligibility rule, with no cross-namespace special cases.
- Joe's six legacy notes are migrated once to `claude:<n>`, by `--adopt`, with his approval (D11). Their `skill_hash` and the skill text in their bodies stay the same. Their basenames and preamble lines change.
- Never offer a removal unless the source's read actually succeeded.
- About 77 first-run offers remain answerable, both interactively and through an agent.

**Non-Goals:**
- Plugin `commands` object-map form, and `**` globs in Pi paths. Both are skipped with a warning, and the affected scope is not scanned. This machine uses neither. When a Pi `!`/`-` override contains `**`, it cannot be applied to the default folders, so every file in those folders is marked Disabled, with a warning.
- Pi `--skill` and `--prompt-template` CLI paths, Claude managed/enterprise skill dirs, lazily-loaded nested `.claude/skills` below cwd, and project-level plugin enablement in `<project>/.claude/settings.json`.
- Project sources outside a git repository. They are skipped with one line (D3).
- Changing curation. `curate`'s pending-skill-note branch already handles any skill note.
- Reading any frontmatter field of a skill, command, or prompt file. Keys come from paths (D2; Joe, round 2). A harness display name, such as a plugin skill's `name:` override, may be added later as a separate field that plays no part in identity.

## Decisions

**D1. One resolver, `ResolveSkillSources(home, cwd)`, returns the candidates and the set of read roots that were read successfully.**
- A candidate is `{Key, Name, ScopeID, SourceSegment, ReadRoot, SourcePath (resolved), WalkedPath, Kind: skill|command|prompt, Content, Disabled}`. The returned set of successful read roots is D5's `ScannedRoots`.
  - `Name` is the scope-local name (directory name, file stem or command path). The scanners leave `Key` empty; key construction builds it from `ScopeID`, `SourceSegment`, `Kind` and `Name`.
  - `SourceSegment` is the key segment that `ScopeID` does not determine: `pi-settings` or `pi-pkg:<pkg-id>` for Pi configured sources, and inside a `project:<r>` scope `pi`, `agents` or `pi-prompt` for `.pi/skills`, `.agents/skills` and `.pi/prompts`. It is empty otherwise.
  - `WalkedPath` is the path as discovered, before symlink resolution. Pi settings patterns match against it.
  - `Disabled` marks a file Pi finds but has switched off (D2 source 6, 7). It is never offered, and it counts as present for removal (D5).
- The resolver also returns the project identity and, unchanged, the plugin scanner's facts: whether its manifests were read, each installed plugin's scanned status, and the plugin conflicts (D4, D5).
- Harness roots come from `update`'s `supportedHarnesses`/`detectHarnesses`.
- The resolver no longer resolves or returns engram-owned roots (round 3): `ResolveEngramSkillRoots`, `EngramSkillRoots` and `EngramSkillRootsUnresolved` are deleted. Engram's deployed skills are read only through the harness folders that link to them, like any other symlinked skill.
- `registerSkillsTargets` and `runUpdateSkillRegistration` both call the resolver with the same home and the injected `Getwd`. The update hook stops passing `<sourceRoot>/agent-instructions/skills`, and runs after the sync, so it reads the freshly deployed engram copies. The re-exec child inherits cwd.

Alternatives:
- (a) Update keeps reading `agent-instructions/skills`. Rejected: that makes two definitions of the shipped set, which Joe's decision 3 forbids.
- (b) Each caller builds its own list. Rejected for the same reason.

**D2. Sources, discovery and names.** Sources are listed in precedence order. Kind is a skill unless noted.

1. **Claude user skills.** The immediate children of `~/.claude/skills`. A child is a skill if it is a directory, **or a symlink that resolves to a directory**, containing `SKILL.md`. Skip `synced`, root files, dangling symlinks, and directories without `SKILL.md`.
2. **Claude user commands.** `~/.claude/commands/**/*.md`, following symlinks. The name is the relative path with `/` replaced by `:` and `.md` removed (`opsx/apply.md` gives `opsx:apply`).
3. **Synced.** For each **directory** child of `~/.claude/skills/synced/` (files such as `.bucket-*` are ignored) whose `manifest.json` parses, read its `<n>/SKILL.md` children.
4. **Pi user.** `~/.pi/agent/skills`: recursive `SKILL.md` discovery (stop at a skill directory, depth-bounded, cycle guard on the resolved path), plus root `.md` files named by their stem.
5. **agents user.** `~/.agents/skills`: recursive, and root `.md` files are ignored.
6. **Pi settings skills.** The global `settings.json` `skills` entries. A path resolves relative to `~/.pi/agent`, and `~` expands. A directory entry uses recursive discovery **plus its own root `.md` files**. Pi calls `collectSkillEntries(dir, "pi")` (`package-manager.js:241`, reached via `:1986-2005` and `:456-463`). A file entry is one skill, named by its stem, or by its parent directory when the file is `SKILL.md`.
   - Pattern entries are the ones with a `!`/`+`/`-` prefix or a `*`/`?` (`isPattern`, `:127-129`). They are filters, not paths (`applyPatterns`, `:540-584`):
     - plain globs include (all files when there are none);
     - `!` excludes;
     - `+path` re-adds by exact path;
     - `-path` removes by exact path.
   - Matching is Go `path.Match`, per segment, against the discovered path relative to the base, the file name or the absolute path. For `SKILL.md` the same three forms of its directory also count (`:465-511`).
   - **A filtered-out file is kept as a Disabled candidate: never offered, still present for removal** (`:1901`).
   - The `!`, `+` and `-` entries also apply to the default folders of the same settings file (sources 4 and 5 for global settings, source 10's folders for project settings; `isEnabledByOverrides`, `:515-539`, `:1949`). A default-folder file they switch off is likewise Disabled, not dropped.
6b. **Pi prompt templates.** `~/.pi/agent/prompts/*.md` (flat, `package-manager.js:291-326`), plus the global `settings.json` `prompts` entries. These resolve and filter like `skills`. A directory contributes its `*.md` files **recursively**, skipping `.`-prefixed entries and `node_modules` (`collectFiles`, `:149-192`). Kind: prompt. The name is the file stem (`prompt-templates.js:85`).
7. **Pi packages.** The global `packages` entries:
   - `npm:<name>[@v]` resolves to `~/.pi/agent/npm/node_modules/<name>`.
   - `git:<host>/<path>[@ref]` and protocol URLs resolve to `~/.pi/agent/git/<host>/<path>`.
   - A local path resolves relative to `~/.pi/agent`.
   - Package contents follow Pi's `collectPackageResources` (`dist/core/package-manager.js` ~1747-1785):
     - **String-form entry:** if `package.json` has **any** `pi` key, only the manifest's entries load (`pi.skills`, `pi.prompts`, paths and globs relative to the root). A resource type the manifest omits loads nothing. The convention directories `skills/` and `prompts/` are used **only when there is no `pi` key at all** (`:1765-1784`).
       - A skill directory, whether listed or the convention one, is searched recursively, **plus its root `.md` files** (`:241`).
       - A prompt directory is searched **recursively** (`:149-192`).
       - The manifest list's own `!`/`+`/`-` entries drop files outright (`addManifestEntries`, `:1863-1874`).
     - **Object-form entry:** per resource type. An explicit `skills`/`prompts` pattern list filters the package (`applyPackageFilter`, `:1802-1817`). **A filtered-out file, including every file under `[]`, is kept as a Disabled candidate: never offered, still present for removal.** An omitted key falls back per type (`collectDefaultResources`): the manifest's entry for that type if present, else that type's convention directory. `autoload: false` (a delta over a global entry) is skipped with a warning and marked not scanned.
8. **Claude plugins.** Each `installed_plugins.json` entry that meets all three conditions:
   - It is enabled: `enabledPlugins["<plugin>@<marketplace>"]`, or, when that entry is absent, `plugin.json` `defaultEnabled`, which defaults to true per the Claude docs.
   - Its `scope` is `user`, or `project`/`local` with `projectPath` equal to the repo top-level.
   - Its `installPath` exists.

   Skills come from `<installPath>/skills/<n>/SKILL.md` plus any `plugin.json` `skills` paths. Commands come from `<installPath>/commands/**/*.md`, or from `plugin.json` `commands` when that is a path or an array, which replaces the default. `plugins/cache` is never globbed. A declared `skills` or `commands` path must stay inside the plugin: an absolute path, or one that leaves the resolved `installPath` lexically or through a symlink, is refused with a warning and the plugin is not scanned, as is a missing declared path or a single command file that is not `.md`.
9. **Claude project.** `.claude/skills/<n>` (skills) and `.claude/commands/**/*.md` (commands) in the cwd and each ancestor up to the repo top-level (`git rev-parse --show-toplevel`, where the files physically live). The same name at two levels of the chain is a key conflict (D4), never nearest-wins. Registration cannot tell which level a note came from without per-directory identity, and nearest-wins would flip a refresh depending on the cwd. Root `.md` files in `.claude/skills` are ignored.
10. **Pi project** (only when D3's trust test passes). In precedence order, which mirrors the global Pi sources 4–7: `.pi/skills` in the cwd (recursive, plus root `.md`); `.agents/skills` in the cwd and its ancestors up to the top-level (recursive, no root `.md`; each level's settings overrides match against its own `<dir>/.agents`, as Pi's `package-manager.js:1959-1966` does); `.pi/settings.json` `skills` entries, resolved relative to `.pi`; `.pi/prompts/*.md`; project `prompts` settings entries; then `.pi/settings.json` `packages` (installed under `.pi/npm`/`.pi/git`). A project package entry overrides a global entry with the same identity. The same name at two levels of the `.agents/skills` chain is a key conflict.

The name is always the directory name, file stem, or command path, never a frontmatter `name:`. An entry name containing `:` is skipped with a warning, which keeps keys parseable.

**Joe's decision (round 2): keys come from the folder or path, never from frontmatter `name:`.** A harness display name (e.g. hookify `writing-rules` shown as `writing-hookify-rules`) may be added later as a separate field that plays no part in identity. Alternative: key on `name:`, as the Claude docs describe for plugin skills. Rejected: it breaks the requirement "Skill files SHALL carry no engram-specific metadata", and a key would change whenever a frontmatter line is edited.

**D3. Identity is a source-qualified key with distinct `cmd`/`pi-prompt` segments, and the slug is derived from the key.** Every key starts with a source qualifier; there is no bare key (round 3).

| Source | Key | Scope ID (answer selector `@<id>`) |
| --- | --- | --- |
| Claude user skill (including engram's installed skills, which `~/.claude/skills` links to) | `claude:<n>` | `claude-user` |
| Claude user command | `claude:cmd:<ns:…:n>` | `claude-cmd` |
| synced | `anthropic-skills:<n>` | `synced` |
| Pi user | `pi:<n>` | `pi-user` |
| agents user | `agents:<n>` | `agents-user` |
| Pi settings skill | `pi-settings:<n>` | `pi-settings` |
| Pi package skill | `pi-pkg:<pkg-id>:<n>` | `pi-pkg:<pkg-id>` |
| Pi prompt template (`~/.pi/agent/prompts`) | `pi-prompt:<n>` | `pi-prompt` |
| Pi settings / package prompt | `pi-settings:pi-prompt:<n>` / `pi-pkg:<pkg-id>:pi-prompt:<n>` | `pi-settings` / `pi-pkg:<pkg-id>` |
| plugin skill | `<plugin>:<n>` | `plugin:<plugin>` |
| plugin command | `<plugin>:cmd:<ns:…:n>` | `plugin:<plugin>` |
| Claude project skill / command | `project:<r>:<n>` / `project:<r>:cmd:<ns:…:n>` | `project:<r>` |
| Pi project `.pi/skills` / `.agents/skills` / settings / package | `project:<r>:pi:<n>` / `project:<r>:agents:<n>` / `project:<r>:pi-settings:<n>` / `project:<r>:pi-pkg:<pkg-id>:<n>`; prompts `project:<r>:pi-prompt:<n>` (and `…:pi-settings:pi-prompt:<n>`, `…:pi-pkg:<pkg-id>:pi-prompt:<n>`) | `project:<r>` |

- `<pkg-id>` is the npm package name, `<host>/<path>` for git, or the `~`-relative resolved path for a local package, following Pi's own package identity rules.
- `<r>` is the project identity. It is `<host>/<owner>/<repo>`: the host plus the path of `git remote get-url origin`, fully normalized: the whole string lowercased, userinfo (`git@`, `user:token@`) and any `:port` stripped, and a trailing `.git` and `/` removed. The ssh://, https:// and scp `git@host:owner/repo` forms all parse, so `ssh://git@GitHub.com:22/Toejough/Engram.git` and `https://github.com/toejough/engram` both give `github.com/toejough/engram`. SSH host aliases are a known limitation (Risks). The host is included so that github.com/x/y and gitlab.com/x/y never share a key (review round 2). With no origin, `<r>` is `local/<basename of the parent of git rev-parse --path-format=absolute --git-common-dir>`, which is stable across worktrees. Two remote-less repos with the same basename share it (Risks). The same `local/<name>` fallback also applies when an origin exists but fails to normalize — a hostless remote (a bare local path, a `file://` URL) or one whose parsed form would keep a residual `:` (e.g. an IPv6 host), which would make keys unparseable (ruling R20).
- Examples from this worktree: `project:github.com/toejough/engram:openspec-propose`, and `project:github.com/toejough/engram:cmd:opsx:apply` (slug `skill-project-github-com-toejough-engram-cmd-opsx-apply`).
- A cwd outside any git repo contributes no project sources.
- **No engram-owned rule** (round 3; this reverses review B2's bare-key rule). Engram's installed skills are keyed by the folder they are found in, like anything else. On this machine, `~/.claude/skills/route` links to `~/.claude/engram/skills/route` and keys as `claude:route`. `~/.pi/agent/skills/route` links to the separate real copy `~/.pi/agent/engram/skills/route` and keys as `pi:route`. The two copies are SHA-identical (checked 2026-09-26 for all six), so `pi:route` is an alias of the higher-precedence `claude:route` (D4). It makes no offer and counts as present. The fork cases this admits are accepted in Risks.
- A real, non-engram `~/.claude/skills/route` is simply `claude:route`. No collision rule exists: a key's copies either collapse or conflict under D4's ordinary rules.
- **Plugin names that repeat across marketplaces** (none today) are **not re-keyed**. That is a D4 plugin conflict: loud, with no offers, and the plugin scope is not scanned.
- **Reserved plugin names** are exactly the top-level key forms: `claude`, `pi`, `agents`, `project`, `anthropic-skills`, `pi-settings`, `pi-pkg` and `pi-prompt` (round 3).
  - `claude` is new, because `claude:` is now the Claude user scopes' key prefix. No installed plugin is named `claude` (checked against `installed_plugins.json` 2026-09-26; `claude-code-setup`, `claude-hud` and `claude-md-management` are distinct names).
  - `cmd` is dropped, because `cmd:` is no longer a top-level key form. A plugin named `cmd` keys as `cmd:<n>`/`cmd:cmd:<n>`, which parse unambiguously as plugin keys.
  - `engram` is **not** reserved: an `engram` plugin is an ordinary plugin (`engram:<n>` keys).
  - **The reserved-name check runs after the enablement check** (round 3, fidelity review). An enabled plugin with a reserved name is skipped with the existing warning (`engram: plugin "<name>" uses a reserved name; it is not scanned`). A disabled one contributes nothing and prints nothing, as for any disabled plugin. Previously the check ran first (`skillsources_plugin.go:132`), before enablement is known inside `scanClaudePlugin`.
- **Parsing a key's namespace.** The first segment names the source family. For `claude`, a key with at least three segments whose second is `cmd` is a user command (`claude:cmd:<ns:…:n>`). Otherwise it is a user skill: a skill directory literally named `cmd` gives `claude:cmd`, with two segments. This is the same shape plugins already use (`<plugin>:cmd:…`).
- **Slug.** Lowercase the key, replace each run of characters outside `[a-z0-9]` with `-`, trim leading and trailing `-`, and prefix `skill-`. For example, `project:github.com/toejough/engram:openspec-propose` becomes `skill-project-github-com-toejough-engram-openspec-propose`, `claude:route` becomes `skill-claude-route`, and `claude:cmd:audit` becomes `skill-claude-cmd-audit`.
  Different keys may share a readable slug (e.g. `pi:a-b` and `pi-a:b` both give `skill-pi-a-b`). This is accepted (ruling R25): lookup is by `skill_key`, never by slug, and the Luhmann id keeps each note's basename unique.
- **Stored fields and lookup.** Notes carry `skill_key` and `skill_source` (`~`-relative resolved path). A **skill note** is a runbook note, with a `skill-` slug (ruling R23), carrying both `skill_hash` and `skill_key`. Lookup is by `skill_key` only. **There is no slug fallback** (round 3): a runbook note with `skill_hash` but no `skill_key` (an unkeyed legacy note) is not a skill note. It is never matched to a key, is not an alias source (D4), and is never offered for removal (D5). D11 covers the six such notes in the real vault. Likewise, a note whose `skill_key` is **unrecognized** is never matched to a candidate, never removal-eligible, and makes nothing an alias. A key is unrecognized in exactly three cases (fidelity re-review A):
  - it has no `:` (e.g. a hand-edited bare `route`);
  - any segment is empty (e.g. `claude:`, `pi::x`);
  - it starts with a reserved form's prefix but its tail lacks that form's shape (e.g. `project:github.com/toejough/engram` with no name, `pi-pkg:x` with no name, `claude:cmd:` with no command).

  Any other `<x>:<n…>` whose first segment is not a reserved form is a **plugin key** for plugin `<x>`, and it follows D5's plugin rule. So `engram:route`, with no `engram` plugin installed, is an uninstalled plugin's note: removal-eligible exactly like `ralph-loop:cmd:help` (spec "Uninstalled plugin is offered for removal"). The two cannot be told apart, and they must not be.

Alternatives:
- (a) The last origin segment (round-1 draft). Rejected by review: it collides across owners and flips between worktrees.
- (b) Always qualify by source, including engram's installed skills. This was rejected in round 1 because the six notes would be renamed and re-offered. **It was adopted in round 3**: D11's one-time migration pays for the rename.
- (c) Qualify only on collision, including the round-2 `<plugin>@<marketplace>` re-keying. Rejected: a key would change as other things are installed, splitting identity and later offering spurious removals of still-installed skills (review N2).
- (d) `<owner>/<repo>` without the host (round 2). Rejected: it collides across forges.
- (e) Bare keys for Claude user skills and engram-owned copies, with a slug fallback for the six legacy notes (the implemented draft). Rejected by Joe at final review (vault note 1073a): the shared unqualified namespace needed the multi-root eligibility rule, the user-vs-engram collision rule and the legacy fallback (Context). Qualifying every key costs one migration and deletes all three.
- (f) An engram owner namespace, `engram:<n>` (drafted in round 3, then declined by Joe). Every copy under any harness's engram skills root would share one key. The engram roots would be scanned directly as source 0, removal would need every detected harness's engram root (with not-exist counting as absence), and diverged copies would collapse by precedence with a warning. Its benefit was that engram's skills never fork per harness (the Risks fork cases). Its cost was keeping a whole engram-owned mechanism: root resolution, a second not-exist rule, a diverged-copy exception, a reserved `engram` plugin name (which makes the installed `engram@engram` warn on every run), and a preamble special case. Declined: engram's installed skills are treated like any other source, and the forks are accepted.

**D4. Dedupe: resolved path, then same key, then content.**
1. Entries that resolve to the same file become one candidate.
2. Two candidates with the same key and the **same scope** collapse when their SHAs are identical. When the SHAs differ, that is a **key conflict**. Examples: Pi recursion finds `a/foo` and `b/foo`; two synced buckets both hold `pdf`; two Pi settings directories both hold `x`. Registration reports `engram: skill key conflict: <key> at <path1> and <path2>`, makes no offer of any kind for that key, and exits with a failure status after all other offers are handled. The update hook records the failure without rolling anything back.
   - This covers the same name at two levels of a project chain (`.claude/skills`, `.claude/commands`, `.agents/skills`). There is no nearest-wins (review N1).
   - **Plugin conflict:** a plugin name installed from more than one marketplace is reported once, `engram: plugin name conflict: <plugin> in <m1>, <m2>`. It makes no offer for any of its keys, and its scope is not scanned, so there are no removals (review N2).
   - **There is no exception** (round 3). The engram-owned exception is deleted: a key's differing copies are always a conflict. So are ruling R27(c) ("every copy must be engram-owned") and the user-vs-engram collision rule. Engram's installed copies on different harnesses have different keys (`claude:route`, `pi:route`), so they never share a key group. When their bytes are equal they alias (step 3); when they differ, each is compared with its own note (Risks).
3. A candidate whose SHA equals that of a higher-precedence candidate in this run, or equals any existing skill note's `skill_hash`, is an **alias**. It makes no offer and is recorded nowhere, but it still counts as present for its key (D5). "Skill note" is D3's keyed note: an unkeyed legacy note's `skill_hash` makes nothing an alias.
   - Aliasing runs before the refresh check, so a key whose new bytes equal another note's `skill_hash` gets no refresh offer. Every member of a key group, including every copy of a conflicted key, still acts as an alias source for later candidates (ruling R27(b)). Both choices are conservative: they can only suppress an offer, and they never write or remove anything.

Alternative: record alias paths on the note. Rejected: it would write on runs where the procedure did not change.

**D5. A removal needs proof that the note's own source was read.** A read counts **only when it succeeded**; a read error, including not-exist, means not scanned, never empty. The resolver records every root it read successfully (fully symlink-resolved) as `ScannedRoots`. Each note's removal eligibility is decided by its key form:

- **One rule per namespace (round 3).** A note's eligibility depends only on the roots of its own key form. The read status of another namespace's roots never changes it. This is the property the deleted multi-root rule broke.
- **Single-root user forms.** These are `claude:` (skills), `claude:cmd:`, `pi:`, `agents:` and `pi-prompt:`. The one fixed root for that form must be in `ScannedRoots`: `~/.claude/skills`, `~/.claude/commands`, `~/.pi/agent/skills`, `~/.agents/skills` or `~/.pi/agent/prompts` respectively. A machine without a readable `~/.claude/skills` (e.g. Pi-only) never offers removal of `claude:` notes. A missing `~/.agents/skills` blocks only `agents:` removals.
  - **Deleted (round 3):** the bare-key rule that also required `~/.pi/agent/skills`, `~/.agents/skills`, every root overlapping an engram-owned root, and the engram roots themselves (`engramCopiesRead`, `EngramSkillRootsUnresolved`; ruling R38's M1 and ruling R26). With it goes ruling R39's residual, where a merely missing `~/.agents/skills` blocked every bare-key removal. A `claude:` note is found only through `~/.claude/skills`, so that root alone proves it absent.
- **Source-rooted forms.** These are `anthropic-skills:`, `pi-settings:…`, `pi-pkg:…` and every `project:…` key. The **specific root that contains the note's recorded `skill_source`** must be in `ScannedRoots`:
  - the synced bucket whose `manifest.json` parsed (per bucket, so a failing bucket never exposes its notes);
  - the settings entry's resolved path;
  - the package root;
  - the exact project directory (`<dir>/.claude/skills`, `<dir>/.claude/commands`, `<dir>/.agents/skills`, `.pi/skills`, `.pi/prompts`).

  Each such root vouches only for its own key prefix (`anthropic-skills:`, `pi-settings:`, `pi-pkg:<id>:`, `project:<r>:` plus its segment). The deepest root of the form that contains the `skill_source` must be read and must vouch for the note's key, so a read root of another package, project or sub-source never exposes a note. A root that failed to read is matched by its best-effort resolved path.
  A note whose `skill_source` lies under no scanned root is never removal-eligible. This is what makes a nested-only `sub/.claude/skills/foo` safe from the top level, keeps a worktree's note safe from the main checkout, and ties a `pi-settings:` note to its own entry (review N1, N3).
- **Plugin keys.** `installed_plugins.json` and `settings.json` both parsed, there is no plugin conflict, and the plugin is either enabled with its `installPath` read, or absent from the manifest (uninstalled). An installed-but-disabled plugin is not scanned. An entry with no `enabledPlugins` value and a missing `installPath` (`work-on`) is not scanned.

A removal is offered only when the note is eligible **and** no candidate (alias or not) has its key. A Disabled candidate counts as present: it holds its key, so its note is never offered for removal. With `--skills-dir`, no removal is offered. An **unkeyed** note (a `skill_hash` runbook with no `skill_key`, D3) is never a removal candidate: it has no key to prove absent, and D11 migrates the only six such notes.

**Orphans (accepted):** a note becomes permanently ineligible for removal when its source root is no longer part of the source set. Examples: a Pi package dropped from `settings.json`, a `skills` entry deleted, a project deleted, or a `skill_source` whose symlink escapes its root. Such a note is left alone, and the user removes it by hand (Risks).

Alternatives:
- (a) Existence-based scanning (round-1 draft). Rejected by review: an unreadable directory would look empty and flood the user with removals.
- (a2) Per-scope scanned-ness (round-2 draft). Rejected by review: it cannot tell which entry, bucket, or project directory a note came from.
- (a3) The multi-root bare-key rule (final-review fix wave, ruling R38's M1). Deleted in round 3 together with bare keys. A key found through more than one root needed every such root read, and a missing root then blocked unrelated removals (ruling R39).
- (b) A `last_seen` counter. Rejected: hidden state, and it still misfires on a machine without the project checked out.

**D6. Grouping, selectors and precedence.**
- Offers are sorted by (scope ID, key). Each scope ID has a display label: the plugin, synced, or project name plus a count. The label **is the answerable selector**, so the summary shows `@plugin:superpowers (15)` and the user answers with `--decline @plugin:superpowers`.
- **Answers** to `--accept`/`--decline` take one of three forms: an exact key, a key-prefix pattern ending in `*` (e.g. `superpowers:cmd:*`), or a scope selector `@<scope-id>`. `@claude-user` names the `claude:<n>` skills, `@claude-cmd` names the `claude:cmd:` commands, and `*` names everything.
- **Precedence for an offer.** An exact key wins. Otherwise the longest matching `*` pattern (by prefix length) wins. Otherwise a scope selector applies. The same key, pattern or selector named in both `--accept` and `--decline` is refused before anything happens.
- **Interactive prompting.**
  - A scope with more than one register or refresh offer is asked once: `[a]ccept all / [d]ecline all / [r]eview each / [s]kip for now`. Skip records nothing. EOF counts as skip.
  - A single offer uses the per-offer y/N prompt.
  - **Removal offers are never covered by "accept all" or by a pattern or selector.** Each one is confirmed by its exact key or by an individual prompt. Project removals are rare by construction (D5: the exact source folder must be read), but individual confirmation still guards branch switches within one checkout (Risks).
- **Non-interactive output** is one line, `engram: 77 skill runbook offers awaiting an answer: @plugin:superpowers 15, @synced 12, @project:github.com/toejough/engram 13, … — run …`.
- **`--dry-run`** prints `would offer: <kind> <key> (<source>)` under scope headers.

Alternatives:
- (a) Opt-in scopes. Rejected: Joe directed all default folders.
- (b) Per-skill prompts only. Rejected: about 77 sequential prompts.
- (c) Store declines per scope. Rejected: declines stay per key and hash, and "decline all" records each key.

**D7. Declines are keyed by skill key, in schema v2.** `skill-registrations.json` is `{schema_version: 2, declined: {<key>: <hash>}}`. **v1 reading is dropped** (Joe, round 3). Any `schema_version` other than 2, including a missing field, a value below 2, or a value above 2, is a loud error, and nothing is written. The v1 format never left this unmerged branch, and the real vault has no file. A missing file means no declines. Because today's reader ignores `schema_version` and the field set is unchanged, **an older binary still reads a v2 file**. It sees the extra keys and never matches them, so the version bump documents the change in key meaning rather than guarding compatibility.

Alternatives:
- (a) Leave the file at v1. Rejected: it would give no record that keys are now source-qualified.
- (b) Read v1 names as `claude:<n>` (round-3 draft). Rejected by Joe: a compatibility mapping for a format that never shipped.

**D8. The preamble names the real source, with the same wording for every source.** For **every** note, the preamble is exactly ``> Mirrors skill `<skill_source>`.`` (Joe, round 3). It names the mirrored path and **drops** the old "edit the procedure there; the runbook fields on this note are authored here" clause, which is false for a deployed or third-party copy. In the preamble, `<skill_source>` is the `~`-relative resolved source path: a Claude user or Pi skill, a plugin skill or command file, a synced file, a Pi or package file, or a project file. Round 3 deletes the engram-owned special case that kept ``agent-instructions/skills/<n>/SKILL.md`` (`SkillNoteSource.EngramOwned`, and the special case in `PreamblePath`). The same applies to the register provenance text `source: skill registration: <path>`.
- For engram's installed skills, the preamble therefore names the deployed file, e.g. `~/.claude/engram/skills/route/SKILL.md`. It claims only what is mirrored, never where to edit. The preamble line of the six migrated notes changes at migration (D11).
- An accepted refresh updates `skill_source` (and with it the preamble) after a plugin version bump. There is no longer a legacy-note stamping case: refresh only ever acts on a keyed note (D3).
- A version bump whose bytes are identical makes no refresh offer, so the old path remains until the next real change.

**D9. `--skills-dir` is preview-only.** `--skills-dir <dir>` (repeatable) replaces the default set. Each directory is scanned with the Claude-user rules, and its entries key as `claude:<n>` in scope `claude-user` (round 3; previously bare). The run is **read-only**: it implies `--dry-run` and refuses `--accept`, `--decline` and `--adopt`, and it makes no removal offers.

This fixes two problems (review B5):
- A note accepted from a non-default directory would later get a spurious removal offer from the default scan.
- `--skills-dir agent-instructions/skills` would otherwise rewrite the six notes' preambles.

Alternative: treat `<checkout>/agent-instructions/skills` as engram-owned and let other directories write. Rejected: it fixes the preamble problem but not the orphaned-note problem.

**D10. No paid eval.** The mechanism is deterministic and unit-testable with DI fakes. It is checked with the real binary against the real home, and against a vault copy with a minimal fixture home.

**D11. Legacy notes: ignored until adopted, then migrated once with `--adopt` (round 3).**
- Exactly six unkeyed notes exist: Joe's 1036 route, 1045 please, 1049 curate, 1053 write-memory, 1067 learn and 1068 recall. Each has a `skill-<n>` slug and a `skill_hash`, but no `skill_key`. No other install has any, because the change that wrote them is on this same unmerged branch.
- **The slug-remainder fallback is deleted outright.** Until migration, the six are not skill notes (D3). They make no alias (D4) and are never offered for removal (D5). Registration offers `register claude:<n>` for each of engram's six skills, found through `~/.claude/skills`; this is expected. The Pi copies are SHA-identical aliases of those candidates, so `pi:<n>` makes no offer.
- **Accepting one of those six register offers before migration would create a second note for the same skill.** So migration comes before any answer to `@claude-user`. The window is closed by ordering: every pre-migration check uses a scratchpad-built binary, and the global `go install` happens only inside the migration step (Risks, tasks 8.8 and 8.11).
- **Migration** uses the existing adopt mechanism. It runs as one non-interactive command (stdin not a terminal, so no remaining offer is prompted for or written), immediately after `go install`, in the same step Joe explicitly approves. It is the final real-vault task:
  `engram register-skills --adopt claude:route=1036 --adopt claude:please=1045 --adopt claude:curate=1049 --adopt claude:write-memory=1053 --adopt claude:learn=1067 --adopt claude:recall=1068`
  For each note, adopt does the following:
  - renames it to the key-derived slug, e.g. `1036.2026-09-18.skill-route.md` becomes `1036.2026-09-18.skill-claude-route.md`, keeping its Luhmann id and date;
  - rewrites every inbound wikilink and rebuilds the rewritten referrers' sidecars;
  - replaces the body with the current source file plus the D8 preamble;
  - stamps `skill_key: claude:<n>` and `skill_source: ~/.claude/engram/skills/<n>/SKILL.md`;
  - keeps `skill_hash` (the source bytes are unchanged) and every runbook field, and clears `pending`.
- **What changes and what doesn't:**
  - The skill text in each body is byte-identical.
  - Each preamble line changes from ``> Mirrors skill `agent-instructions/skills/<n>/SKILL.md` — edit the procedure there; …`` to ``> Mirrors skill `~/.claude/engram/skills/<n>/SKILL.md`.`` (D8), and each basename changes.
  - Doc references to the old slugs `skill-please`, `skill-route` and `skill-write-memory` are rewritten by the enumeration rows.
- **After migration** the dry run from `/tmp` shows 64 offers and none for engram's six (Migration Plan).

## Risks / Trade-offs

- **[Risk] Divergent project skills or commands across branches or worktrees.** Keys are shared across worktrees, but removal eligibility (D5) is tied to the exact folder in the note's `skill_source`.
  - A note accepted in a worktree (e.g. `project:github.com/toejough/engram:cmd:opsx:update`, which only this worktree has) is **never removal-eligible from the main checkout**, because that folder is not read there. It becomes a **permanent orphan once the worktree is deleted**.
  - Refresh offers can alternate between branches with different bytes. A decline is remembered by hash.
  - A branch switch inside one checkout can make a note removal-eligible, and each removal needs individual confirmation (D6).
  - → **Recommendation: accept project offers from the main checkout**, so that `skill_source` points at a long-lived folder.
- **[Risk] A project Pi-settings entry that resolves to the repository top level (e.g. `".."` in `.pi/settings.json`) contains nested worktrees** (`.claude/worktrees/<w>`). From the main checkout, that entry's read root holds a worktree's files, so it can vouch for a worktree note keyed `project:<r>:pi-settings:<x>` and offer its removal. → Accepted and documented, not fixed: such an entry is unusual, and every removal is individually confirmed (D6).
- **[Risk] Repository rename or transfer changes `<r>`.** The new keys alias the old notes' hashes while the content is unchanged, so they produce no register offer. The old notes stay removal-eligible only while their recorded folder is read. After a clone at the new name, the old `skill_source` folder usually still exists (same path), and its old keys have no candidate, so each old note gets a removal offer that is individually confirmed. Registration under the new key happens once those are accepted.
- **[Risk] SSH host aliases.** An origin like `git@github-work:toejough/engram.git` (an `~/.ssh/config` alias) normalizes to `github-work/toejough/engram`, a different `<r>` from the canonical host. → A known limitation: engram does not read `~/.ssh/config`. Keys stay stable for any one clone, but the same repo cloned through different aliases gets different keys.
- **[Risk] Pending-note load.** About 77 accepted notes need curation. → A pending note stays out of retrieval, scope-level declines make "not these" one answer, and `curate` already handles pending skill notes.
- **[Risk] Mirroring untrusted third-party text.** → Nothing is written without an explicit answer, new notes stay pending until curated, and Pi's trust gate is honored for Pi project sources.
- **[Risk] A plugin rename causes churn.** New keys alias the old notes' hashes, so they produce no register offer. The old notes get removal offers (the manifest shows the old name uninstalled). Registering under the new key happens only after the user accepts those removals. → This is deliberate: no duplicate note while the old one exists. A **marketplace move** under the same plugin name keeps its keys, because keys carry no marketplace.
- **[Risk] Orphaned notes** (D5). When a source leaves the source set (a package dropped from Pi settings, a deleted project, a removed `skills` entry), its notes are never offered for removal. → This is safe by construction; the orphan stays until removed by hand. An orphan listing or pruning command is out of scope; the controller files a follow-up issue.
- **[Risk] Two remote-less repos with the same basename share `local/<name>`.** → Refresh offers can flip between them. Removals stay safe because they require the note's recorded `skill_source` directory to have been read, and they still need individual confirmation.
- **[Trade-off] Commands shadowed by same-named skills are still registered** under their `cmd` key. In the harness, skills take precedence. An example is the `commit@skills` plugin, which ships `skills/commit` and `commands/commit.md`. → The keys never collide, and both procedures exist on disk.
- **[Trade-off] Plugin `skill_source` embeds a version.** → Refresh is keyed to a byte change, so the path can go stale. The key, not the path, is the identity.
- **[Risk] A dry run inside `engram update` reads the currently deployed engram copies.** → The preview can miss a refresh that the real run will offer. Documented in the dry-run output.
- **[Risk] Pi's ignore files are not honored** (ruling R7). Pi skips skills and prompts hidden by `.gitignore`, `.ignore` or `.fdignore` rules (`package-manager.js:73`, `:107-126`); engram does not read those files. → As a result, engram may offer a skill that Pi skips. The user can decline it, and removal is unaffected, because scanning only ever finds more files, never fewer. A follow-up issue tracks parity.
- **[Risk, accepted] Engram's installed skills can fork per harness** (round 3, alternative (f) declined). Each copy is keyed by its folder, so one engram skill can end up with a `claude:<n>` note and a `pi:<n>` note:
  - **A Pi-only vault or machine.** There, engram's skills key as `pi:<n>`. A vault shared with a Claude machine sees a `pi:<n>` register offer whenever the Pi bytes match no keyed note (e.g. after a release, when the Claude machine refreshed first or has not refreshed yet).
  - **Diverged harness copies.** When Pi's copy is not yet synced and the `claude:<n>` refresh is accepted, Pi's old bytes then match nothing, so `pi:<n>` is offered for registration until `engram update` syncs Pi. The offer can be declined by hash.
  - **A harness removed.** When `~/.claude` goes away, the `claude:<n>` notes become orphans: never removal-eligible, because `~/.claude/skills` is not read. `pi:<n>` aliases their hashes until the next release changes the bytes, and then `pi:<n>` is offered as a new note.

  → Accepted. Every such offer is explicit and declinable, and nothing is written without an answer. On a machine with both harnesses synced, the copies are SHA-identical and alias to one `claude:<n>` note.
- **[Risk] A registration run during an `engram update` redeploy can offer a spurious removal** (round 3). While update re-links engram's skills, a `~/.claude/skills/<n>` link can be dangling or briefly missing. `~/.claude/skills` is still read, so the `claude:<n>` note has no candidate and is offered as `remove claude:<n>`. The old bare-key rule masked this case: engram's Pi copy held the same bare key, and an unresolved engram-owned root blocked every bare-key removal. → Accepted: it is only an offer and defaults to N, and update's own registration hook runs after the sync completes, so only a separate registration run started concurrently with the redeploy can see it.
- **[Risk] Answering the six `claude:<n>` register offers before migration duplicates notes** (D11). → Closed by ordering. Pre-migration verification uses a scratchpad-built binary only. The global `go install` happens inside the Joe-approved migration step, immediately before the non-interactive adopt (tasks 8.8, 8.11), so the installed binary never shows the six offers with the notes still unkeyed.
- **[Risk] Renamed basenames.** Adopt rewrites vault wikilinks, but text outside the vault that names the old slugs does not follow: `delegate.md:31`, `GLOSSARY.md:58`, `c1-system-context.md:423` and `adr.md:531`. → These are enumeration rows 26–29 (plus rows 30–38 for key examples and the doc text written this cycle).
- **[Risk] A key conflict hides that key** (same name at two project levels, two Pi entries, two synced buckets, or a plugin in two marketplaces). → It is reported loudly with both paths and a failure exit status. Nothing is written for the key.

## Migration Plan

1. TDD the resolver (D1–D3), dedupe and conflicts (D4), read-root scoping (D5), selectors and grouping (D6), declines v2 (D7), preamble and refresh stamping (D8), `--skills-dir` (D9), and field survival on every frontmatter rewrite path.
2. Wire both callers and update the comments.
3. Verify against the real layout: a real-binary dry run from `/tmp` (64 offers) and from this worktree (77). Then, on a scratch vault copy with a minimal fixture home, verify accept, decline and removal-scoping.
4. Perform the enumeration rows, install, and run a final read-only dry run. Joe answers the real offers himself.
5. **Round 3 (qualify every key; tasks section 8).** TDD the key change, the deletions (legacy fallback, multi-root eligibility, the engram-owned mechanism) and per-namespace eligibility. Then re-run the real-layout verification **with a binary built into the scratchpad, never `go install`**, so the globally installed `engram` never offers the six `claude:<n>` registrations before migration. The expected counts are derived from the measured current dry run (64 from `/tmp`) and the checked SHA identity of the Claude and Pi copies:
   - **Before migration:** 70 from `/tmp`: the same 64, with `@claude-user` rising from 4 to 10 because of `register claude:{curate,learn,please,recall,route,write-memory}`. No `pi:` offer for the six, since the Pi copies alias. No refresh, removal, conflict or warning. 83 from this worktree (70 + 13 project).
   - **After migration:** 64 from `/tmp`, none for engram's six. 77 from this worktree.
6. **Install and migrate in one step** (D11), only after Joe explicitly approves it: `go install ./cmd/engram` immediately followed by the `--adopt` command, run non-interactively (stdin not a terminal), so it never prompts for the remaining offers. Then confirm the post-migration counts and `engram embed status`, and commit the vault.

Rollback: revert the commits. Notes created under new keys are ordinary pending runbook notes and can be removed by hand. Older binaries on this branch read a v2 `skill-registrations.json` unchanged (D7). Adopted notes keep their Luhmann ids. A migration is reverted with `git revert` of its vault commit.

## Resolved Questions (Joe, round 2)

- **Frontmatter `name:` vs directory name:** keys come from the folder or path, never from frontmatter. A display name may be added later as a separate field that plays no part in identity (D2).
- **`enabledPlugins` absent:** follow the docs. Fall back to `plugin.json` `defaultEnabled`, which defaults to true (D2).
- **Pi prompt templates:** included, with the reserved `pi-prompt` segment (D2, D3). None exist on this machine, so the expected counts are unchanged.

## Resolved Questions (Joe, round 3)

- **Bare keys:** none. Every key is qualified by its source (vault note 1073a). The multi-root eligibility rule, the user-vs-engram collision rule and the legacy slug fallback are deleted (Context).
- **An `engram:` owner namespace:** declined. Engram's installed skills are keyed by their folder, like any other source, and the fork cases are accepted (D3 alternative (f), Risks).
- **Preamble:** ``> Mirrors skill `<skill_source>`.`` for every note, without "edit the procedure there" (D8).
- **v1 declines:** v1 reading is dropped; any `schema_version` other than 2 is an error (D7).
- **Orphan listing or pruning command:** out of scope. The controller files a follow-up issue (Risks, orphans).

## Open Questions

- None.
