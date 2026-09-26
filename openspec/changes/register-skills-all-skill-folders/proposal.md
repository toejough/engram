## Why

`register-skills-as-runbooks` registers only the skills engram ships. Its design listed "Registering skills engram doesn't ship" as a Non-Goal, and commit `8a672df5` narrowed the standalone default further, to `~/.claude/engram/skills`. Joe has reversed this: "Register-skills need to read all default skill folders, not just the engram ones" (vault note 1066a, narrowing 1066). After review round 1 he extended the scope to every command folder, to Pi's trust gate, and to Pi's configured skill sources. After round 2 he added Pi prompt templates, and ruled that keys never come from frontmatter.

Memory should be able to surface every procedure the harnesses load: user skills and commands, plugin skills and commands, claude.ai-synced skills, Pi's configured skills and packages, and project-local skills and commands. This changes what the spec promises and where registration reads from, so it needs its own change before any code (vault note 695).

## What Changes

- **BREAKING (scope): one source set for both entry points.** `engram register-skills` and the `engram update` post-deploy hook stop reading a single directory. Both scan the same default source set, resolved by one function:
  - Claude Code user skills (`~/.claude/skills`, following symlinked skill dirs) and user commands (`~/.claude/commands/**`, e.g. `opsx/apply.md` → `opsx:apply`).
  - claude.ai-synced skills (`~/.claude/skills/synced/<bucket>/` with a `manifest.json`).
  - Pi user skills (`~/.pi/agent/skills`, `~/.agents/skills`) and prompt templates (`~/.pi/agent/prompts`), plus the skill and prompt paths and packages configured in Pi `settings.json` (npm, git, and local).
  - Skills and commands of enabled Claude Code plugins, read from `installed_plugins.json` `installPath` only, never from other cached versions.
  - Project-local Claude skills and commands, from the cwd up to the repo top-level.
  - Pi project sources (`.pi/skills`, `.pi/prompts`, `.agents/skills`, `.pi/settings.json`), only for projects that Pi trusts.
- **Source-qualified keys, with distinct `cmd` and `pi-prompt` segments, taken only from paths.** Keys look like `route`, `cmd:audit`, `superpowers:brainstorming`, `commit:cmd:commit`, `anthropic-skills:pdf`, `pi:ping`, `pi-prompt:<n>`, `pi-pkg:<id>:<n>`, `project:github.com/toejough/engram:cmd:opsx:apply`. Keys are never re-keyed by marketplace.
  - Anything whose fully symlink-resolved path lies under an engram-owned root keeps its bare key on every harness, so the six existing notes never fork.
  - Notes record `skill_key` and `skill_source`, and the slug is derived from the key.
  - The "Mirrors skill" preamble names the real source.
- **Dedupe and conflicts.** Entries that resolve to the same path collapse. A copy whose content matches another copy, or an existing note, becomes an alias. Two different contents under one key are reported as a conflict and never guessed at. That covers the same name at two project levels, and a plugin name installed from two marketplaces. The only exception is diverged engram copies, where precedence wins with a warning.
- **Removal offers only with proof that the note's own source was read.** For synced, Pi-settings, Pi-package and project notes, the specific root containing the note's recorded `skill_source` must have been read in this run. Plugin notes follow the manifest rule. Notes whose source has left the set become documented orphans, and are never offered for removal. None is offered under `--skills-dir` either.
- **Volume handling.** Offers are grouped by scope and answered with `@<scope>` selectors, `prefix*` patterns, or exact keys. Precedence is exact key, then longest pattern, then selector. Removals are always confirmed one at a time. The non-interactive summary gives counts per scope.
- **Declines.** Declines are keyed by skill key under `schema_version: 2`. v1 files are read as they are, and older binaries still read v2.
- **`--skills-dir` becomes repeatable and preview-only.**
- **Identity fields survive every frontmatter rewrite path.**

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
- **Vault:** notes 1036, 1045, 1049, 1053, 1067, and 1068 keep their basenames and hashes, and no offer is made for them. There is no `skill-registrations.json` today. After Joe answers, new pending notes appear for accepted entries: up to 64 from a non-repo cwd, and 77 from this worktree.
- **Docs:** see `enumeration.md`.
- **No new dependencies, and no paid eval.**
