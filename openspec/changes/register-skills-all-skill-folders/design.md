## Context

`register-skills-as-runbooks` (archived 2026-09-26) built the offer flow: compare, prompt, then register, refresh, or remove, with declines remembered by hash. It only ever fed that flow engram's own skills. `loadShippedSkills` (`internal/cli/skillreg_run.go:234`) reads one directory's immediate `IsDir()` children, so every symlinked skill directory is skipped. The standalone default is `~/.claude/engram/skills` (`targets.go:379`, from `8a672df5`). The update hook reads `<sourceRoot>/agent-instructions/skills` (`update.go` `runUpdateSkillRegistration`). Identity is the bare skill name: slug `skill-<name>`, decline key `<name>`. A removal is offered for any skill note whose name is missing from the scanned set. That was safe only because the scanned set was the whole universe of skill notes.

Joe reversed the Non-Goal (vault note 1066a). Registration covers every default skill folder the harnesses load, and (review round 1) every command folder and Pi's configured skill sources too, and (review round 2) Pi prompt templates. The decline reader never checks `schema_version` (`skillreg.go:146-160`; it decodes `{schema_version, declined}` with `DisallowUnknownFields`).

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
  - `settings.json` `skills: string[]` holds file or directory paths. They resolve relative to `~/.pi/agent`, or relative to `.pi` for project settings. `~`, globs, `!pattern`, `+path` and `-path` are supported.
  - `packages` entries are either a string or `{source, skills?: [...]}`:
    - `npm:<name>[@ver]` installs to `~/.pi/agent/npm/node_modules/<name>`, or `.pi/npm/` for a project.
    - `git:`/URL sources clone to `~/.pi/agent/git/<host>/<path>`, or `.pi/git/…`.
    - A local path resolves against its settings file.
    - A package's skills come from `package.json` `pi.skills`, falling back to the `skills/` convention directory. An object-form `skills` filter narrows them: omit it for all, `[]` for none.
    - When a package appears in both the global and the project settings, the project entry wins.
  - Prompt templates (`docs/prompt-templates.md`) load from `~/.pi/agent/prompts/*.md`, from `.pi/prompts/*.md` for trusted projects, from `prompts/` directories or `pi.prompts` entries in a package, and from the `settings.json` `prompts` array. They are flat `*.md` files, and "the filename becomes the command name". Their frontmatter is optional `description`/`argument-hint`.
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

The real vault holds six skill notes: 1036 route, 1045 please, 1049 curate, 1053 write-memory, 1067 learn, 1068 recall. Their `skill_hash` values equal the deployed SHAs, their preambles name `agent-instructions/skills/<n>/SKILL.md`, and there is no `skill-registrations.json`. Note 820 has a `skill-` slug but no `skill_hash`. `engram register-skills --dry-run` from `/tmp` prints nothing today.

## Goals / Non-Goals

**Goals:**
- One definition of the default source set, used by both standalone `register-skills` and the `engram update` hook.
- Every skill, command, and Pi prompt template a harness loads by default, or that Pi's settings configure, can be offered, with copies collapsed.
- A stable, source-qualified identity that leaves the six existing notes byte-for-byte unchanged, with no offers for them. This must hold on every harness mix, including a Pi-only machine.
- Never offer a removal unless the source's read actually succeeded.
- About 77 first-run offers remain answerable, both interactively and through an agent.

**Non-Goals:**
- Plugin `commands` object-map form, and `**` globs in Pi paths. Both are skipped with a warning, and the affected scope is not scanned. This machine uses neither.
- Pi `--skill` and `--prompt-template` CLI paths, Claude managed/enterprise skill dirs, lazily-loaded nested `.claude/skills` below cwd, and project-level plugin enablement in `<project>/.claude/settings.json`.
- Project sources outside a git repository. They are skipped with one line (D3).
- Changing curation. `curate`'s pending-skill-note branch already handles any skill note.
- Reading any frontmatter field of a skill, command, or prompt file. Keys come from paths (D2; Joe, round 2). A harness display name, such as a plugin skill's `name:` override, may be added later as a separate field that plays no part in identity.

## Decisions

**D1. One resolver, `ResolveSkillSources(home, cwd)`, returns the candidates and the set of read roots that were read successfully.**
- A candidate is `{Key, ScopeID, ReadRoot, SourcePath (resolved), Kind: skill|command|prompt, Content}`. The returned set of successful read roots is D5's `ScannedRoots`.
- Harness roots come from `update`'s `supportedHarnesses`/`detectHarnesses`.
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
6. **Pi settings skills.** The global `settings.json` `skills` entries. A path resolves relative to `~/.pi/agent`, and `~` expands. A directory entry uses recursive discovery with no root `.md`. A file entry is one skill, named by its stem, or by its parent directory when the file is `SKILL.md`. Supported: `*`/`?`/`[…]` per path segment (Go `path.Match`), `!pattern`, `+path`, `-path`.
6b. **Pi prompt templates.** `~/.pi/agent/prompts/*.md` (flat), plus the global `settings.json` `prompts` entries (resolved like `skills`; a directory contributes its flat `*.md`). Kind: prompt. The name is the file stem.
7. **Pi packages.** The global `packages` entries:
   - `npm:<name>[@v]` resolves to `~/.pi/agent/npm/node_modules/<name>`.
   - `git:<host>/<path>[@ref]` and protocol URLs resolve to `~/.pi/agent/git/<host>/<path>`.
   - A local path resolves relative to `~/.pi/agent`.
   - The package root's skills come from `package.json` `pi.skills` (paths and globs relative to the root) when present, else from `skills/` (recursive, plus top-level `.md`). An object-form `skills` filter then narrows the set. The package's prompt templates come from `pi.prompts`, else from `prompts/*.md`, narrowed by the object-form `prompts` filter.
8. **Claude plugins.** Each `installed_plugins.json` entry that meets all three conditions:
   - It is enabled: `enabledPlugins["<plugin>@<marketplace>"]`, or, when that entry is absent, `plugin.json` `defaultEnabled`, which defaults to true per the Claude docs.
   - Its `scope` is `user`, or `project`/`local` with `projectPath` equal to the repo top-level.
   - Its `installPath` exists.

   Skills come from `<installPath>/skills/<n>/SKILL.md` plus any `plugin.json` `skills` paths. Commands come from `<installPath>/commands/**/*.md`, or from `plugin.json` `commands` when that is a path or an array, which replaces the default. `plugins/cache` is never globbed.
9. **Claude project.** `.claude/skills/<n>` (skills) and `.claude/commands/**/*.md` (commands) in the cwd and each ancestor up to the repo top-level (`git rev-parse --show-toplevel`, where the files physically live). The same name at two levels of the chain is a key conflict (D4), never nearest-wins. Registration cannot tell which level a note came from without per-directory identity, and nearest-wins would flip a refresh depending on the cwd. Root `.md` files in `.claude/skills` are ignored.
10. **Pi project** (only when D3's trust test passes). `.pi/skills` in the cwd (recursive, plus root `.md`); `.agents/skills` in the cwd and its ancestors up to the top-level (recursive, no root `.md`); `.pi/settings.json` `skills`/`packages`, resolved relative to `.pi` (packages go to `.pi/npm`/`.pi/git`); `.pi/prompts/*.md` and project `prompts` settings. A project package entry overrides a global entry with the same identity. The same name at two levels of the `.agents/skills` chain is a key conflict.

The name is always the directory name, file stem, or command path, never a frontmatter `name:`. An entry name containing `:` is skipped with a warning, which keeps keys parseable.

**Joe's decision (round 2): keys come from the folder or path, never from frontmatter `name:`.** A harness display name (e.g. hookify `writing-rules` shown as `writing-hookify-rules`) may be added later as a separate field that plays no part in identity. Alternative: key on `name:`, as the Claude docs describe for plugin skills. Rejected: it breaks the requirement "Skill files SHALL carry no engram-specific metadata", and a key would change whenever a frontmatter line is edited.

**D3. Identity is a source-qualified key with distinct `cmd`/`pi-prompt` segments, and the slug is derived from the key.**

| Source | Key | Scope ID (answer selector `@<id>`) |
| --- | --- | --- |
| Claude user skill, **and any candidate whose resolved path lies under an engram-owned root** (`<home>/<EngramRootRel>/skills` for each supported harness: `~/.claude/engram/skills`, `~/.pi/agent/engram/skills`) | `<n>` | `claude-user` |
| Claude user command | `cmd:<ns:…:n>` | `claude-cmd` |
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
- `<r>` is the project identity. It is `<host>/<owner>/<repo>`: the lowercased host plus the path of `git remote get-url origin`, with `.git` removed. The ssh://, https:// and scp `git@host:owner/repo` forms all parse. The host is included so that github.com/x/y and gitlab.com/x/y never share a key (review round 2). With no origin, `<r>` is `local/<basename of the parent of git rev-parse --path-format=absolute --git-common-dir>`, which is stable across worktrees. Two remote-less repos with the same basename share it (Risks).
- Examples from this worktree: `project:github.com/toejough/engram:openspec-propose`, and `project:github.com/toejough/engram:cmd:opsx:apply` (slug `skill-project-github-com-toejough-engram-cmd-opsx-apply`).
- A cwd outside any git repo contributes no project sources.
- The engram-owned-root rule (review B2) means Pi's copies of engram's skills always carry the key `route`, never `pi:route`. So a Pi-only machine matches note 1036, and a release never forks a `pi:route` note. The comparison is made on **fully symlink-resolved paths on both sides**: the candidate's resolved `SKILL.md` must be under the resolved `<home>/<EngramRootRel>/skills`. This covers a symlinked `$HOME`, and fixtures whose engram root is a symlink into the real one.
- **Plugin names that repeat across marketplaces** (none today) are **not re-keyed**. That is a D4 plugin conflict: loud, with no offers, and the plugin scope is not scanned.
- **Reserved plugin names** are `pi`, `agents`, `project`, `anthropic-skills`, `cmd`, `pi-settings`, `pi-pkg` and `pi-prompt`. A plugin with one of these names is skipped with a warning.
- **Slug.** Lowercase the key, replace each run of characters outside `[a-z0-9]` with `-`, trim leading and trailing `-`, and prefix `skill-`. For example, `project:github.com/toejough/engram:openspec-propose` becomes `skill-project-github-com-toejough-engram-openspec-propose`.
- **Stored fields and lookup.** New notes carry `skill_key` and `skill_source` (`~`-relative resolved path). Lookup is by `skill_key` on a runbook note with `skill_hash`. A skill note with no `skill_key` takes its key from its slug remainder, which covers the six legacy notes.

Alternatives:
- (a) The last origin segment (round-1 draft). Rejected by review: it collides across owners and flips between worktrees.
- (b) Always harness-qualify. Rejected: the six notes would be renamed and re-offered.
- (c) Qualify only on collision, including the round-2 `<plugin>@<marketplace>` re-keying. Rejected: a key would change as other things are installed, splitting identity and later offering spurious removals of still-installed skills (review N2).
- (d) `<owner>/<repo>` without the host (round 2). Rejected: it collides across forges.

**D4. Dedupe: resolved path, then same key, then content.**
1. Entries that resolve to the same file become one candidate.
2. Two candidates with the same key and the **same scope** collapse when their SHAs are identical. When the SHAs differ, that is a **key conflict**. Examples: Pi recursion finds `a/foo` and `b/foo`; two synced buckets both hold `pdf`; two Pi settings directories both hold `x`. Registration reports `engram: skill key conflict: <key> at <path1> and <path2>`, makes no offer of any kind for that key, and exits with a failure status after all other offers are handled. The update hook records the failure without rolling anything back.
   - This covers the same name at two levels of a project chain (`.claude/skills`, `.claude/commands`, `.agents/skills`). There is no nearest-wins (review N1).
   - **Plugin conflict:** a plugin name installed from more than one marketplace is reported once, `engram: plugin name conflict: <plugin> in <m1>, <m2>`. It makes no offer for any of its keys, and its scope is not scanned, so there are no removals (review N2).
   - **Engram-owned exception (the only one):** entries under engram-owned roots share bare keys across harnesses. When their bytes differ (e.g. a Pi copy not yet synced), the first in precedence (Claude's) wins, and one warning line suggests `engram update`. This is not a conflict and not a failure (review N4).
3. A candidate whose SHA equals that of a higher-precedence candidate in this run, or equals any existing skill note's `skill_hash`, is an **alias**. It makes no offer and is recorded nowhere, but it still counts as present for its key (D5).

Alternative: record alias paths on the note. Rejected: it would write on runs where the procedure did not change.

**D5. A removal needs proof that the note's own source was read.** A read counts **only when it succeeded**; a read error, including not-exist, means not scanned, never empty. The resolver records every root it read successfully (fully symlink-resolved) as `ScannedRoots`. Each note's removal eligibility is decided by its key form:

- **Single-root user forms.** These are the bare key (Claude user or engram-owned), `cmd:`, `pi:`, `agents:` and `pi-prompt:`. The fixed root for that form must be in `ScannedRoots`: `~/.claude/skills` (for a bare key whose note has no `skill_source`, or whose `skill_source` lies under an engram-owned root: `~/.claude/skills` or the matching engram root), `~/.claude/commands`, `~/.pi/agent/skills`, `~/.agents/skills` or `~/.pi/agent/prompts`.
- **Source-rooted forms.** These are `anthropic-skills:`, `pi-settings:…`, `pi-pkg:…` and every `project:…` key. The **specific root that contains the note's recorded `skill_source`** must be in `ScannedRoots`:
  - the synced bucket whose `manifest.json` parsed (per bucket, so a failing bucket never exposes its notes);
  - the settings entry's resolved path;
  - the package root;
  - the exact project directory (`<dir>/.claude/skills`, `<dir>/.claude/commands`, `<dir>/.agents/skills`, `.pi/skills`, `.pi/prompts`).

  A note whose `skill_source` lies under no scanned root is never removal-eligible. This is what makes a nested-only `sub/.claude/skills/foo` safe from the top level, keeps a worktree's note safe from the main checkout, and ties a `pi-settings:` note to its own entry (review N1, N3).
- **Plugin keys.** `installed_plugins.json` and `settings.json` both parsed, there is no plugin conflict, and the plugin is either enabled with its `installPath` read, or absent from the manifest (uninstalled). An installed-but-disabled plugin is not scanned. An entry with no `enabledPlugins` value and a missing `installPath` (`work-on`) is not scanned.

A removal is offered only when the note is eligible **and** no candidate (alias or not) has its key. With `--skills-dir`, no removal is offered.

**Orphans (accepted):** a note becomes permanently ineligible for removal when its source root is no longer part of the source set. Examples: a Pi package dropped from `settings.json`, a `skills` entry deleted, a project deleted, or a `skill_source` whose symlink escapes its root. Such a note is left alone, and the user removes it by hand (Risks).

Alternatives:
- (a) Existence-based scanning (round-1 draft). Rejected by review: an unreadable directory would look empty and flood the user with removals.
- (a2) Per-scope scanned-ness (round-2 draft). Rejected by review: it cannot tell which entry, bucket, or project directory a note came from.
- (b) A `last_seen` counter. Rejected: hidden state, and it still misfires on a machine without the project checked out.

**D6. Grouping, selectors and precedence.**
- Offers are sorted by (scope ID, key). Each scope ID has a display label: the plugin, synced, or project name plus a count. The label **is the answerable selector**, so the summary shows `@plugin:superpowers (15)` and the user answers with `--decline @plugin:superpowers`.
- **Answers** to `--accept`/`--decline` take one of three forms: an exact key, a key-prefix pattern ending in `*` (e.g. `superpowers:cmd:*`), or a scope selector `@<scope-id>`. `@claude-user` names the bare-key scope; `*` names everything.
- **Precedence for an offer.** An exact key wins. Otherwise the longest matching `*` pattern (by prefix length) wins. Otherwise a scope selector applies. The same key, pattern or selector named in both `--accept` and `--decline` is refused before anything happens.
- **Interactive prompting.**
  - A scope with more than one register or refresh offer is asked once: `[a]ccept all / [d]ecline all / [r]eview each / [s]kip for now`. Skip records nothing. EOF counts as skip.
  - A single offer uses the per-offer y/N prompt.
  - **Removal offers are never covered by "accept all" or by a pattern or selector.** Each one is confirmed by its exact key or by an individual prompt. This matters because a branch switch can make project commands look removed (Risks).
- **Non-interactive output** is one line, `engram: 77 skill runbook offers awaiting an answer: @plugin:superpowers 15, @synced 12, @project:github.com/toejough/engram 13, … — run …`.
- **`--dry-run`** prints `would offer: <kind> <key> (<source>)` under scope headers.

Alternatives:
- (a) Opt-in scopes. Rejected: Joe directed all default folders.
- (b) Per-skill prompts only. Rejected: about 77 sequential prompts.
- (c) Store declines per scope. Rejected: declines stay per key and hash, and "decline all" records each key.

**D7. Declines are keyed by skill key, in schema v2.** `skill-registrations.json` becomes `{schema_version: 2, declined: {<key>: <hash>}}`. The new reader accepts v1 and v2, since v1 names equal the bare keys of the same skills. The first write stamps v2. A version above 2 is an error. Because today's reader ignores `schema_version` and the field set is unchanged, **an older binary still reads a v2 file**. It sees the extra keys and never matches them, so the version bump documents the change in key meaning rather than guarding compatibility. The real vault has no such file.

Alternative: leave the file at v1. Rejected: it would give no record that keys are now source-qualified.

**D8. The preamble names the real source.** When the resolved source lies under an engram-owned root, the preamble keeps its current bytes, ``> Mirrors skill `agent-instructions/skills/<n>/SKILL.md` — …``, which is the edit location. Otherwise the preamble names the `~`-relative `skill_source`: a plugin skill or command file, a synced file, a Pi or package file, or a project file.
- An accepted refresh also stamps `skill_key` and `skill_source` onto a legacy note, and updates `skill_source` (and with it the preamble) after a plugin version bump.
- A version bump whose bytes are identical makes no refresh offer, so the old path remains until the next real change.

**D9. `--skills-dir` is preview-only.** `--skills-dir <dir>` (repeatable) replaces the default set. Each directory is scanned with the Claude-user rules (bare keys). The run is **read-only**: it implies `--dry-run` and refuses `--accept`, `--decline` and `--adopt`, and it makes no removal offers.

This fixes two problems (review B5):
- A note accepted from a non-default directory would later get a spurious removal offer from the default scan.
- `--skills-dir agent-instructions/skills` would otherwise rewrite the six notes' preambles.

Alternative: treat `<checkout>/agent-instructions/skills` as engram-owned and let other directories write. Rejected: it fixes the preamble problem but not the orphaned-note problem.

**D10. No paid eval.** The mechanism is deterministic and unit-testable with DI fakes. It is checked with the real binary against the real home, and against a vault copy with a minimal fixture home.

## Risks / Trade-offs

- **[Risk] Divergent project skills or commands across branches or worktrees.** The main checkout's `opsx/` lacks `update.md`, which this worktree has. Running from the main checkout would therefore make `project:github.com/toejough/engram:cmd:opsx:update` look removed, and switching back would offer it again. → Removals always need individual confirmation (D6), and a decline is remembered by hash. Keys are shared across worktrees by design, so a note always mirrors whichever branch was last accepted.
- **[Risk] Pending-note load.** About 77 accepted notes need curation. → A pending note stays out of retrieval, scope-level declines make "not these" one answer, and `curate` already handles pending skill notes.
- **[Risk] Mirroring untrusted third-party text.** → Nothing is written without an explicit answer, new notes stay pending until curated, and Pi's trust gate is honored for Pi project sources.
- **[Risk] A plugin rename causes churn.** New keys alias the old notes' hashes, so they produce no register offer. The old notes get removal offers (the manifest shows the old name uninstalled). Registering under the new key happens only after the user accepts those removals. → This is deliberate: no duplicate note while the old one exists. A **marketplace move** under the same plugin name keeps its keys, because keys carry no marketplace.
- **[Risk] Orphaned notes** (D5). When a source leaves the source set (a package dropped from Pi settings, a deleted project, a removed `skills` entry), its notes are never offered for removal. → This is safe by construction; the orphan stays until removed by hand. A later `--prune-orphans` listing could surface them. It is not in scope here.
- **[Risk] Two remote-less repos with the same basename share `local/<name>`.** → Refresh offers can flip between them. Removals stay safe because they require the note's recorded `skill_source` directory to have been read, and they still need individual confirmation.
- **[Trade-off] Commands shadowed by same-named skills are still registered** under their `cmd` key. In the harness, skills take precedence. An example is the `commit@skills` plugin, which ships `skills/commit` and `commands/commit.md`. → The keys never collide, and both procedures exist on disk.
- **[Trade-off] Plugin `skill_source` embeds a version.** → Refresh is keyed to a byte change, so the path can go stale. The key, not the path, is the identity.
- **[Risk] A dry run inside `engram update` reads the currently deployed engram copies.** → The preview can miss a refresh that the real run will offer. Documented in the dry-run output.
- **[Risk] A key conflict hides that key** (same name at two project levels, two Pi entries, two synced buckets, or a plugin in two marketplaces). → It is reported loudly with both paths and a failure exit status. Nothing is written for the key.

## Migration Plan

1. TDD the resolver (D1–D3), dedupe and conflicts (D4), read-root scoping (D5), selectors and grouping (D6), declines v2 (D7), preamble and refresh stamping (D8), `--skills-dir` (D9), and field survival on every frontmatter rewrite path.
2. Wire both callers and update the comments.
3. Verify against the real layout: a real-binary dry run from `/tmp` (64 offers) and from this worktree (77). Then, on a scratch vault copy with a minimal fixture home, verify accept, decline and removal-scoping.
4. Perform the enumeration rows, install, and run a final read-only dry run. Joe answers the real offers himself.

Rollback: revert the commits. Notes created under new keys are ordinary pending runbook notes and can be removed by hand. Older binaries read a v2 `skill-registrations.json` unchanged (D7).

## Resolved Questions (Joe, round 2)

- **Frontmatter `name:` vs directory name:** keys come from the folder or path, never from frontmatter. A display name may be added later as a separate field that plays no part in identity (D2).
- **`enabledPlugins` absent:** follow the docs. Fall back to `plugin.json` `defaultEnabled`, which defaults to true (D2).
- **Pi prompt templates:** included, with the reserved `pi-prompt` segment (D2, D3). None exist on this machine, so the expected counts are unchanged.

## Open Questions

- None blocking. Should a later change add an orphan listing or pruning command (Risks)?
