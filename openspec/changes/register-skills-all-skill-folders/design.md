## Context

`register-skills-as-runbooks` (archived 2026-09-26) built the offer flow: compare, then prompt, then register, refresh, or remove, with declines remembered by hash. It fed that flow only engram's own skills. `loadShippedSkills` (`internal/cli/skillreg_run.go:234`) reads one directory's immediate `IsDir()` children. As a result it skips every symlinked skill directory. The standalone default is `~/.claude/engram/skills` (`targets.go:379`, from `8a672df5`). The update hook reads `<sourceRoot>/agent-instructions/skills` (`update.go` `runUpdateSkillRegistration`). Identity is the bare skill name: slug `skill-<name>` and decline key `<name>`. Removal is offered for any skill note whose name is missing from the scanned set. That was safe only because the scanned set was the whole universe of skill notes.

Joe reversed the Non-Goal (vault note 1066a): registration covers every default skill folder the harnesses load.

**Real layout on this machine, verified 2026-09-26 (not guessed):**

| Source | Location | Evidence | Loadable skills |
| --- | --- | --- | --- |
| Claude user | `~/.claude/skills/<n>/SKILL.md` | `ls -la`: 33 entries. 22 are dangling symlinks into `~/repos/personal/projctl/skills/` (the target no longer exists), plus `.obsidian/` (no SKILL.md) and `synced/`. | 10: c4, dev, mycelium, property-rigor, and engram's six (curate, learn, please, recall, route, write-memory, all symlinks into `~/.claude/engram/skills/<n>`) |
| claude.ai synced | `~/.claude/skills/synced/<uuid>/<n>/SKILL.md` + `manifest.json` (`source: anthropic`/`anthropic-example`) | the session skill list shows them as `anthropic-skills:<n>` | 12 (docx, pdf, pptx, xlsx, skill-creator, …) |
| Pi user | `~/.pi/agent/skills/` | `ls -la`: engram's six are symlinks into `~/.pi/agent/engram/skills/<n>`, which is a **separate real copy**, SHA-256 identical to Claude's; plus `ping/` | 7 |
| Pi user (agents std) | `~/.agents/skills/` | Pi docs `docs/skills.md` "Locations"; absent on this machine | 0 |
| Claude plugins | `installPath` from `~/.claude/plugins/installed_plugins.json` (v2, `plugins["<plugin>@<marketplace>"][]` with `scope`, `installPath`) + `<installPath>/skills/<n>/SKILL.md` | `~/.claude/plugins/cache/<marketplace>/<plugin>/` holds **up to 8 stale versions** per plugin, so only `installPath` is current. `enabledPlugins` in `~/.claude/settings.json`: hookify and ralph-loop are `false`; `engram@engram` is `false` but installed with stale learn/recall/remember/prepare/migrate skills; `work-on@work-on` is in the manifest but its `installPath` is missing | 36 across enabled plugins (superpowers 15, plugin-dev 7, traced 4, issue 3, …); harness name `<plugin>:<n>` (e.g. `superpowers:brainstorming`) |
| Claude project | `<cwd>/.claude/skills/<n>/SKILL.md` | engram repo: six `openspec-*` directories plus root files `commit.md` and `engram-go-conventions.md`, which Claude Code does not list as skills (absent from the session skill list) | 6 |
| Pi project | `<cwd>/.pi/skills/` (+ root `.md` files), `.agents/skills/` in cwd and ancestors up to the git root; loaded only for trusted projects (`~/.pi/agent/trust.json`) | Pi docs; `~/repos/personal/{pi-skills,pi-lmstudio,…}/.pi/skills/openspec-*` exist | varies |

Two further Pi sources exist: `settings.json` `skills` arrays and packages. This machine has one package path, `../../repos/personal/pi-skills/bare-skills`, and it does not exist. Those sources are configured additions rather than default folders, so this change leaves them out (Non-Goal).

The real vault holds six skill notes (1036 route, 1045 please, 1049 curate, 1053 write-memory, 1067 learn, 1068 recall). Their `skill_hash` values equal the deployed SHA-256s, their preambles name `agent-instructions/skills/<n>/SKILL.md`, and there is no `skill-registrations.json`. Note 820 (`skill-edits-validated-by-baseline-pressure-tests`) is a runbook whose slug starts with `skill-` but has no `skill_hash`. It must stay a non-skill note. The installed binary's `engram register-skills --dry-run` from `/tmp` prints nothing today.

## Goals / Non-Goals

**Goals:**
- One definition of the default source set, used by both standalone `register-skills` and the `engram update` hook.
- Every skill a harness loads by default can be offered, and copies of the same skill are collapsed.
- A stable, source-qualified identity that leaves the six existing notes byte-for-byte unchanged, with no offers for them.
- Never offer to remove a note because its source happened not to be scanned.
- About 60 first-run offers stay answerable, both interactively and through an agent.

**Non-Goals:**
- Pi `settings.json` `skills` entries, Pi packages, and `--skill` CLI paths. These are configured sources, not defaults. They are a named follow-on.
- Plugin `commands/*.md`. The harness lists them next to skills (e.g. `commit-commands:commit`), but they are not `SKILL.md` skills. See Open Questions.
- Project-scope plugin enablement overrides in `<project>/.claude/settings.json`, managed or enterprise skill directories, and nested `.claude/skills` discovery below cwd.
- Changing how accepted notes are curated. The `curate` skill's existing pending-skill-note branch handles any skill note.
- Reading any `SKILL.md` frontmatter field. The existing requirement "Skill files SHALL carry no engram-specific metadata" still holds. Names come from paths.

## Decisions

**D1. One resolver, `ResolveSkillSources`, produces the scanned set and the scanned scopes.** It is a pure function over injected `ReadDir`/`Stat`/`ReadFile`/`Readlink`/`Getwd` and a repo-name probe. It returns `[]SkillCandidate{Key, Scope, SourcePath (resolved), Content}` and `ScannedScopes`. Harness roots come from `update`'s `supportedHarnesses` probes (`.claude`, `.pi`), so an undetected harness contributes no scope. `registerSkillsTargets` and `runUpdateSkillRegistration` both call it with the same home and cwd. The update hook stops passing `<sourceRoot>/agent-instructions/skills`. It runs after the sync, so engram's skills are read from their freshly deployed copies. Alternatives: (a) keep update on `agent-instructions/skills` and let only the standalone command scan everything. Rejected, because it gives two definitions of the shipped set, which is exactly the drift Joe's decision 3 forbids. (b) Let each caller assemble its own directory list. Rejected for the same reason.

**D2. Source kinds, precedence, and discovery rules.** Precedence runs from first to last: Claude user, synced, Pi user, agents user, plugin (in manifest order), Claude project, Pi project, agents project.
- Claude user, and Claude project: immediate children. A child counts when it is a directory **or a symlink that resolves to a directory** containing `SKILL.md`. Root `.md` files are ignored, matching Claude Code, which does not list `engram-go-conventions.md`. Dangling symlinks and directories without `SKILL.md` are skipped. The `synced` child is skipped here and handled as its own source.
- Synced: `~/.claude/skills/synced/*/<n>/SKILL.md`.
- Pi user, and Pi `.pi/skills`: recursive discovery of directories containing `SKILL.md`, per Pi docs. The walk stops descending at a skill directory and guards cycles with a set of resolved paths. Root `.md` files count as skills named by their stem. `.agents/skills` (user, and cwd up to the git root) follows the same recursion but ignores root `.md` files.
- Plugins: for each `installed_plugins.json` entry whose `enabledPlugins["<plugin>@<marketplace>"]` is `true`, `scope` is `user` (or `project`/`local` with `projectPath` equal to the project root), and `installPath` exists, read `<installPath>/skills/<n>/SKILL.md`. Never glob `plugins/cache`.
- The skill name is the directory name (or the stem, for a Pi root file), never the frontmatter `name:`.
Alternative: read the frontmatter `name:`, which Pi allows to differ from the directory. Rejected, because it would break "Skill files SHALL carry no engram-specific metadata ... SHALL NOT read ... any frontmatter field". The one mismatch seen on this machine (hookify `writing-rules` has `name: writing-hookify-rules`) is in a disabled plugin (Open Questions).

**D3. Identity is a source-qualified skill key, and the slug derives from it.**

| Source | Key | Slug |
| --- | --- | --- |
| Claude user (and `--skills-dir`) | `<n>` | `skill-<n>` |
| synced | `anthropic-skills:<n>` | `skill-anthropic-skills-<n>` |
| Pi user | `pi:<n>` | `skill-pi-<n>` |
| agents user | `agents:<n>` | `skill-agents-<n>` |
| plugin | `<plugin>:<n>` (the `<plugin>` part of `<plugin>@<marketplace>`) | `skill-<plugin>-<n>` |
| Claude project | `project:<p>:<n>` | `skill-project-<p>-<n>` |
| Pi project | `project:<p>:pi:<n>` | `skill-project-<p>-pi-<n>` |
| agents project | `project:<p>:agents:<n>` | `skill-project-<p>-agents-<n>` |

To build the slug: lowercase the key, replace every run of characters outside `[a-z0-9]` with `-`, trim `-`, and prefix `skill-`. New notes carry `skill_key: <key>` and `skill_source: <resolved path, ~-relative>`. Lookup matches by `skill_key` on a runbook note with a non-empty `skill_hash`. A skill note with **no** `skill_key` (the six legacy notes) gets its key from its slug's `skill-` remainder, as it does today. That remainder is a bare key, so notes 1036, 1045, 1049, 1053, 1067, and 1068 resolve to keys `route`, `please`, `curate`, `write-memory`, `learn`, and `recall` without being rewritten. Note 820 has no `skill_hash`, so it is still not a skill note. Two notes with the same key are still an error (the existing duplicate rule). Slug collisions after sanitizing (e.g. plugin `a-b:c` against `a:b-c`) do no harm, because identity is `skill_key` and basenames differ by Luhmann id. A plugin whose name is a reserved namespace (`pi`, `agents`, `project`, `anthropic-skills`) is skipped with a one-line warning instead of being allowed to collide. `<p>` (the project name) reuses the existing repo-identity probe (`detectRepo`, `identity.go:31`), reduced to a name: the last path segment of `git remote get-url origin` with `.git` removed. If that is empty, it uses the basename of `git rev-parse --show-toplevel`, and after that the basename of cwd. This keeps linked worktrees stable: this worktree resolves to `engram`, not `runbook-vs-skill`. Alternatives: (a) always harness-qualify, e.g. `claude:route`. Rejected, because the six notes would need renames and re-offers. (b) Qualify only when names collide. Rejected, because a note's name would then depend on what else happens to be installed, and would flip as plugins come and go. (c) Key the project by the cwd basename. Rejected, because worktrees would split one project into several keys. (d) Keep identity in the slug only, with no `skill_key`. Rejected, because sanitizing is lossy (`:` versus `-`).

**D4. Dedupe happens in two steps: resolved path, then content.** (1) Candidates whose resolved `SKILL.md` path is the same are one candidate (e.g. `~/.claude/skills/route` → `~/.claude/engram/skills/route`). (2) A candidate is an **alias** and makes no offer of its own when its SHA-256 equals the SHA of a higher-precedence candidate in this run, or equals the current `skill_hash` of any existing skill note. So Pi's engram copies (`pi:route`, a separate real file with an identical SHA) collapse into note 1036 and create no `skill-pi-route`. Aliases are not written anywhere. If a copy's content later diverges, it becomes a normal candidate under its own key. Alternative: record every alias path on the note. Rejected, because it would write to a note on every run when nothing about the procedure changed.

**D5. Removal offers are scoped to scopes that were actually scanned.** Every key maps to a scope: bare → `claude-user`; `anthropic-skills:` → `synced`; `pi:`/`agents:` → the matching user scope; `project:<p>:…` → `project:<p>`; anything else → `plugin:<plugin>`. The resolver reports as scanned: `claude-user` and `synced` when `~/.claude` is detected; `pi-user`/`agents-user` when `~/.pi` is detected; `plugin:<x>` when `installed_plugins.json` was read **and** `<x>` is either enabled with its `installPath` present, or absent from the manifest (uninstalled); `project:<p>` when the cwd resolves to project `<p>` and at least one of its project skill directories exists. A plugin that is installed but disabled (`enabledPlugins` false, or its `installPath` missing) is **not** scanned. A removal offer is made only for a note whose scope is scanned and whose key has no candidate. So a project skill's note is never offered for removal from another cwd, from `engram update` run outside the project, or when a harness or plugin was not read. An unreadable `installed_plugins.json` leaves every plugin scope unscanned. Alternatives: (a) record `last_seen` and remove after N unseen runs. Rejected, because it adds hidden state and still offers removal on a laptop without the project checked out. (b) Never offer removals outside the Claude-user scope. Rejected, because uninstalling a plugin should be able to clean up its notes.

**D6. Handle volume by grouping offers by scope.** Offers are sorted by (scope, key). Interactive mode asks once per scope that has any offer: "`superpowers`: 15 skills (15 register). [a]ccept all / [d]ecline all / [r]eview each / [s]kip for now". `r` falls back to today's per-offer y/N prompts. `s` records nothing, so the scope is asked again next run. A scope with exactly one offer uses the per-offer prompt directly. The non-interactive summary line lists counts per scope: `engram: 58 skill runbook offers awaiting an answer (superpowers 15, plugin-dev 7, anthropic-skills 12, …)`, plus the answering command. `--accept`/`--decline` take a skill key or a pattern ending in `*` (`superpowers:*`, `project:engram:*`, `*`). An exact key beats a pattern. A key named in both, or a pattern named in both, is refused (the existing conflict rule). `--dry-run` prints `would offer: <kind> <key> (<source path>)` grouped under scope headers. Alternatives: (a) opt-in scopes, where only Claude-user skills are scanned by default. Rejected, because Joe directed all default folders. (b) Keep per-skill prompts. Rejected, because about 60 sequential y/N prompts on first run will be skipped or mis-answered. (c) Store one decline per scope. Rejected, because declines stay per key and per hash; "decline all" simply records each key.

**D7. Declines are keyed by skill key, with schema version 2 and read-compatible migration.** `skill-registrations.json` becomes `{schema_version: 2, declined: {<skill key>: <hash>}}`. The reader accepts version 1 and interprets its keys as skill keys. Version 1 keys were names from `agent-instructions/skills`, and under D3 those are exactly the bare Claude-user keys of the same skills. The first decline write stamps version 2. An unknown `schema_version` (greater than 2) is an error, not a silent reset, because declines record user intent. The real vault has no such file today (verified), so the migration only matters for other installs. Alternative: rewrite version 1 files eagerly during update. Rejected, because no key actually changes, so a rewrite would do nothing useful.

**D8. The preamble names the real source.** For a candidate whose resolved path lies inside an engram-owned root (`~/.claude/engram/skills`, `~/.pi/agent/engram/skills`), the preamble stays ``> Mirrors skill `agent-instructions/skills/<n>/SKILL.md` — edit the procedure there; …``. That is where the procedure is edited, and keeping it means refreshes of the six notes produce the same bytes. For every other skill, the preamble names `skill_source` (resolved and `~`-relative), e.g. ``> Mirrors skill `~/.claude/plugins/cache/claude-plugins-official/superpowers/6.4.1/skills/brainstorming/SKILL.md` — …``. Plugin paths contain the version, so a plugin upgrade changes `skill_source` and the preamble. It only reaches the note through an accepted refresh, which requires a SHA change. A version bump with identical bytes leaves the note untouched, and its `skill_source` goes stale. That is accepted (Risks). Alternative: always name the scanned path, even for engram's skills. Rejected, because it would point readers at a deployed copy that `engram update` overwrites.

**D9. `--skills-dir` is repeatable and replaces the default set.** When given, only the listed directories are scanned, each with Claude-user rules and bare keys (`--skills-dir agent-instructions/skills` reproduces the old behavior for tests and ad-hoc use). No removal offers are made, and the output says so once. Alternatives: (a) make it additive to the defaults. Rejected, because the flag's use is to isolate a set, and there would be no way to scan a single directory. (b) Scope removals to the given directories. Rejected, because the legacy notes carry no `skill_source` to compare against, which would reproduce the destructive-offer bug from note 1066.

**D10. No paid eval.** The mechanism is deterministic and unit-testable with DI fakes that model symlinks, dangling links, the plugin manifest, and settings. It is verified with the real binary against the real home layout, and against a copy of the vault. The six notes' retrieval is unaffected because they are not written.

## Risks / Trade-offs

- **[Risk] Once accepted, dozens of pending notes need curation.** → A pending note is kept out of retrieval until curated, so nothing degrades. Scope-level decline makes "not these" a single answer. `curate` already handles pending skill notes.
- **[Risk] Mirroring third-party skill text into the vault means untrusted text reaches retrieval.** → It is offered only, never written without an explicit answer. New notes are pending until a curator authors their fields. Pi's project trust gate is not reproduced (Open Questions).
- **[Trade-off] The plugin `skill_source` embeds a version.** → A refresh only follows a byte change, so a same-bytes upgrade leaves a stale path in the preamble. The key, not the path, is identity. This is acceptable.
- **[Risk] A dry run inside `engram update` reads the currently deployed engram copies, not the ones about to be deployed.** → The preview may miss a refresh that the real run will offer. The dry run's contract is "nothing is written", and it is documented in the dry-run output line.
- **[Risk] The scan may be slow with many plugins or a deep Pi recursion.** → Reads are bounded: one level for Claude, a depth-bounded Pi walk with a cycle guard, and plugin `skills/` one level deep.
- **[Risk] Divergent copies of one skill (a Pi copy lagging Claude) produce an extra `pi:<n>` offer.** → That is correct, because they are different procedures. Once the copies match again, the Pi copy aliases again, and the extra note, if it was accepted, gets a removal offer only if its key disappears.

## Migration Plan

1. TDD the resolver (D1, D2), keys and slugs (D3), dedupe (D4), scoped removal (D5), grouping and patterns (D6), declines version 2 (D7), preamble (D8), and `--skills-dir` (D9).
2. Wire both callers to the resolver. Update the comments and flag help.
3. Verify with the real binary from `/tmp` against the real home and a scratch copy of the vault. Confirm the six notes produce no offers, the counts per scope match the table above, and removals are scoped.
4. Perform the enumeration rows. Install. Run a final dry run against the real vault (read-only). Joe answers the real offers himself.

Rollback: revert the commits. Notes created under new keys are ordinary pending runbook notes and can be removed by hand. `skill-registrations.json` version 2 is readable by nothing older. Delete it, or re-decline.

## Open Questions

- **Plugin commands:** should `commands/*.md` (for example `commit-commands:commit` and `claude-md-management:revise-claude-md`, which the harness lists as skills) be registered too? They are out of scope here until Joe decides.
- **Pi project trust:** Pi loads `.pi/skills` only for trusted projects (`trust.json`). This change scans the cwd regardless, because every note still needs an explicit accept. Should registration honour Pi's trust gate?
- **Frontmatter `name` versus directory name:** the harness display name for hookify's `writing-rules` (`name: writing-hookify-rules`) cannot be observed while hookify is disabled. D2 uses the directory name.
