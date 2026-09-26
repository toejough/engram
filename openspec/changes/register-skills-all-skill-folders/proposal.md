## Why

`register-skills-as-runbooks` registers only the skills engram ships. Its design listed "Registering skills engram doesn't ship" as a Non-Goal, and commit `8a672df5` made that narrower by pointing the standalone default at `~/.claude/engram/skills`. Joe has since reversed this: "Register-skills need to read all default skill folders, not just the engram ones" (vault note 1066a, which narrows 1066). Memory should be able to surface every procedure the harnesses load. That includes user skills, plugin skills, the claude.ai-synced skills, and project-local skills, not only engram's six. The change affects what the spec promises and where registration reads from, so it needs its own change before any code (vault note 695).

## What Changes

- **BREAKING (scope):** `engram register-skills` and the `engram update` post-deploy hook stop reading one directory. Both scan the same default source set, resolved by one function:
  - Claude Code user skills: `~/.claude/skills/<n>/SKILL.md`. Symlinked skill directories are followed; before this change they were skipped.
  - claude.ai-synced skills: `~/.claude/skills/synced/<id>/<n>/SKILL.md`. The harness names them `anthropic-skills:<n>`.
  - Pi user skills: `~/.pi/agent/skills/` and `~/.agents/skills/`, using Pi's documented discovery rules.
  - Enabled Claude Code plugin skills: `<installPath>/skills/<n>/SKILL.md`, located through `~/.claude/plugins/installed_plugins.json` and filtered by `enabledPlugins` in `~/.claude/settings.json`.
  - Project-local skills under the current directory: `.claude/skills/`, `.pi/skills/`, and `.agents/skills/`.
- **Source-qualified skill identity.** Each skill gets a skill key: `<n>` for Claude user skills, which keeps the six existing notes as they are; `<plugin>:<n>`; `anthropic-skills:<n>`; `pi:<n>`; `agents:<n>`; and `project:<project>:<n>`. The note records the key (`skill_key`) and the resolved source file (`skill_source`). The slug is derived from the key (`skill-superpowers-brainstorming`). The "Mirrors skill" preamble names the real source. A copy whose bytes match a higher-precedence copy or an existing note's `skill_hash` becomes an alias and gets no note of its own.
- **Scoped removal offers.** A note is offered for removal only when its source scope was actually scanned in this run. Scopes that are not scanned include an absent harness, a disabled plugin, a different or missing project, and an override from `--skills-dir`.
- **Volume handling.** Offers are grouped by source scope. The interactive prompt asks once per scope (accept all, decline all, review each, or skip). The non-interactive summary line reports counts per scope. `--accept`/`--decline` take a skill key or a scope pattern such as `superpowers:*`.
- **Declines keyed by skill key.** `skill-registrations.json` moves to schema version 2. Its keys are skill keys. A version 1 file is read as-is, because its bare names are exactly the Claude-user keys of engram's six skills.
- **`--skills-dir` becomes repeatable and replaces the default set.** Each listed directory is scanned as a Claude-user-shaped directory with bare keys, and no removal offers are made.
- The default-path comments and flag help that assert "only shipped skills register" are corrected.

## Capabilities

### New Capabilities
<!-- none -->

### Modified Capabilities
- `skill-runbook-registration`: the shipped set becomes the scanned default source set. Identity changes from skill name to source-qualified skill key, with new `skill_key` and `skill_source` fields. Removal offers are scoped to scanned sources. Offers are grouped by scope and can be answered by pattern. Declines are keyed by skill key. The preamble names the real source. `--skills-dir` semantics change. The Purpose sentence ("Skills shipped in `agent-instructions/skills/`…") is rewritten by an archive-time task, because deltas cannot edit Purpose.
- `update-deploy-sync`: the post-deploy registration hook scans the same default source set as standalone `register-skills`, not `<sourceRoot>/agent-instructions/skills`.
- `vault-note-identity`: `engram amend` preserves `skill_key` and `skill_source` in the same way it preserves `skill_hash`.
- `learn-runbook-capture`: the preamble requirement names the skill's real source file. For engram-shipped skills that is still the repo edit location.

## Impact

- **Go:** `internal/cli/skillreg.go`, `skillreg_run.go`, and `skillreg_accept.go` gain skill keys, source records, alias collapse, scoped removal, and grouped prompting. A new pure source resolver adds the harness, plugin, synced, and project scanners, with symlink following and dangling-link skipping behind DI. `internal/cli/targets.go` (`registerSkillsTargets`) and `internal/cli/update.go` (`runUpdateSkillRegistration`) call that one resolver. `internal/cli/learn.go` frontmatter gains `skill_key` and `skill_source`, which amend preserves. `internal/update/update.go` comments on `ClaudeEngramSkillsRel` and `ClaudeSkillsTargetRel` change.
- **Vault:** notes 1036, 1045, 1049, 1053, 1067, and 1068 keep their basenames and hashes, and no offer is made for them. `skill-registrations.json` does not exist in the real vault today. After Joe answers, new pending notes appear for accepted skills and are curated through `curate`.
- **Docs:** see `enumeration.md` (CLAUDE.md, README, GLOSSARY, ADR-0026 amendment, C1/C2, LEDGER row, ROADMAP #760, and the Go comments).
- **No new dependencies. No paid eval.**
