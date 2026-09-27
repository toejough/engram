## Why

`register-skills-as-runbooks` registers only the skills engram ships. Its design listed "Registering skills engram doesn't ship" as a Non-Goal, and commit `8a672df5` narrowed the standalone default further, to `~/.claude/engram/skills`. Joe has reversed this: "Register-skills need to read all default skill folders, not just the engram ones" (vault note 1066a, narrowing 1066). After review round 1 he extended the scope to every command folder, to Pi's trust gate, and to Pi's configured skill sources. After round 2 he added Pi prompt templates, and ruled that keys never come from frontmatter. At final review (round 3) he ruled that every key is qualified by its source, with no bare keys (vault note 1073a), and that the skills engram installs get no special treatment: they are keyed by the folder they are found in, like any other source.

Memory should be able to surface every procedure the harnesses load: user skills and commands, plugin skills and commands, claude.ai-synced skills, Pi's configured skills and packages, and project-local skills and commands. This changes what the spec promises and where registration reads from, so it needs its own change before any code (vault note 695).

## What Changes

- **BREAKING (scope): one source set for both entry points.** `engram register-skills` and the `engram update` post-deploy hook stop reading a single directory. Both scan the same default source set, resolved by one function:
  - Claude Code user skills (`~/.claude/skills`, following symlinked skill dirs) and user commands (`~/.claude/commands/**`, e.g. `opsx/apply.md` → `opsx:apply`).
  - claude.ai-synced skills (`~/.claude/skills/synced/<bucket>/` with a `manifest.json`).
  - Pi user skills (`~/.pi/agent/skills`, `~/.agents/skills`) and prompt templates (`~/.pi/agent/prompts`), plus the skill and prompt paths and packages configured in Pi `settings.json` (npm, git, and local).
  - Skills and commands of enabled Claude Code plugins, read from `installed_plugins.json` `installPath` only, never from other cached versions.
  - Project-local Claude skills and commands, from the cwd up to the repo top-level.
  - Pi project sources (`.pi/skills`, `.pi/prompts`, `.agents/skills`, `.pi/settings.json`), only for projects that Pi trusts.
- **Source-qualified keys, with distinct `cmd` and `pi-prompt` segments, taken only from paths. Every key is qualified; there are no bare keys.** Keys look like `claude:c4`, `claude:cmd:audit`, `superpowers:brainstorming`, `commit:cmd:commit`, `anthropic-skills:pdf`, `pi:ping`, `pi-prompt:<n>`, `pi-pkg:<id>:<n>`, `project:github.com/toejough/engram:cmd:opsx:apply`. Keys are never re-keyed by marketplace. `claude` becomes a reserved plugin name.
  - Engram's installed skills have no special case. They key by folder (`claude:route` through `~/.claude/skills`, `pi:route` through `~/.pi/agent/skills`), and identical copies collapse through the alias rule. The fork cases this admits are documented and accepted.
  - Notes record `skill_key` and `skill_source`, and the slug is derived from the key. Lookup is by `skill_key` only: the legacy slug fallback is deleted, and an unkeyed note is never matched, aliased or offered for removal.
  - The "Mirrors skill" preamble always names the `~`-relative real source.
- **Dedupe and conflicts.** Entries that resolve to the same path collapse. A copy whose content matches another copy, or an existing note, becomes an alias. Two different contents under one key are reported as a conflict and never guessed at. That covers the same name at two project levels, and a plugin name installed from two marketplaces. There is no exception: the diverged-engram-copies rule is deleted along with the rest of the engram-owned mechanism.
- **Removal offers only with proof that the note's own source was read, with one rule per key namespace.** A `claude:`, `claude:cmd:`, `pi:`, `agents:` or `pi-prompt:` note needs only its own fixed folder to have been read. For synced, Pi-settings, Pi-package and project notes, the specific root containing the note's recorded `skill_source` must have been read in this run. Plugin notes follow the manifest rule. Notes whose source has left the set become documented orphans, and are never offered for removal. None is offered under `--skills-dir` either.
- **Volume handling.** Offers are grouped by scope and answered with `@<scope>` selectors, `prefix*` patterns, or exact keys. Precedence is exact key, then longest pattern, then selector. Removals are always confirmed one at a time. The non-interactive summary gives counts per scope.
- **Declines.** Declines are keyed by skill key under `schema_version: 2`. v1 files are read as they are, and older binaries still read v2.
- **`--skills-dir` becomes repeatable and preview-only.**
- **Identity fields survive every frontmatter rewrite path.**
- **BREAKING (vault): migration of the six legacy notes.** Joe's six unkeyed notes (1036, 1045, 1049, 1053, 1067, 1068) are adopted as `claude:<n>` with `--adopt`, with his approval, as the final task. Their slugs become `skill-claude-<n>`, inbound links are rewritten, and their preamble line changes to name the deployed file. Their `skill_hash` and skill text are unchanged.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `skill-runbook-registration`:
  - The shipped set becomes the default skill and command source set.
  - Identity moves from name to source-qualified key (the requirement "identified by its slug" is renamed to "identified by its skill key"), with `skill_key` and `skill_source` stored on the note.
  - "Skill files SHALL carry no engram-specific metadata" now covers command files and Pi prompt templates as well.
  - Also changed: dedupe and conflicts, removal scoping to read roots, grouped answering, declines v2, refresh stamping, preamble source, and `--skills-dir`.
  - The Purpose sentence ("Skills shipped in `agent-instructions/skills/`…") is rewritten by an archive-time task.
- `update-deploy-sync`: the post-deploy hook scans the same default source set as standalone `register-skills`.
- `vault-note-identity`: `skill_hash`, `skill_key`, and `skill_source` survive every non-registration frontmatter rewrite.
- `learn-runbook-capture`: the preamble names the real source file of a skill or command.

## Impact

- **Go:**
  - `internal/cli/skillreg*.go`: keys, candidates, dedupe and conflicts, read-root scoping, selectors and grouping, and declines v2.
  - A new pure source resolver, with scanners for the Claude user, synced, Pi, agents, Pi settings and packages, plugin, and project sources, plus Pi trust and a project-identity probe. All I/O goes through DI.
  - `internal/cli/targets.go` and `internal/cli/update.go` wiring.
  - Frontmatter plumbing for `skill_key` and `skill_source` in `learn.go` and in every rewrite path (amend, resituate, reparent, rename-rewrite, identity backfill, vocab tag rewrites).
  - Comments in `internal/update/update.go`.
  - Round 3 deletions: the legacy slug fallback (`groupSkillNoteCandidates`), the multi-root bare-key eligibility (`engramCopiesRead`), and the whole engram-owned mechanism (`ResolveEngramSkillRoots`, `EngramSkillRoots`/`EngramSkillRootsUnresolved`, `offerEngramOwned`, `SkillNoteSource.EngramOwned`, and `update.EngramOwnedSkillsRels` if no caller remains).
- **Vault:** notes 1036, 1045, 1049, 1053, 1067, and 1068 are migrated to `claude:<n>` (slugs `skill-claude-<n>`, links rewritten, hashes unchanged). Until then they are ignored, and `register claude:<n>` is offered for each of engram's six skills: 70 offers from a non-repo cwd and 83 from this worktree. Those six must not be accepted before migration. After migration the offers are 64 and 77, and none is for engram's six. There is no `skill-registrations.json` today. After Joe answers, new pending notes appear for accepted entries.
- **Docs:** see `enumeration.md`.
- **No new dependencies, and no paid eval.**
