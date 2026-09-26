Every Go task follows RED → GREEN → REFACTOR:
- Unit tests use gomega assertions with imptest/impgen mocks driven interactively.
- Every rule over keys, slugs, dedupe, precedence or scoping also gets a rapid property test (`property-rigor`).
- DI only: no `os.*` in `internal/`, and `cmd/engram/main.go` stays wiring-only.
- Run `targ test` / `targ check-full`, never `go test` directly.
- Fakes model: symlinked dirs; dangling symlinks; read errors distinct from not-exist; multi-version plugin caches; `enabledPlugins` true/false/absent; synced buckets alongside stray `.bucket-*` files; Pi `trust.json`; Pi settings/packages (vault notes 1066, 1073).

## 1. Source resolution (design D1, D2, D3 identity inputs)

- [ ] 1.1 RED→GREEN: Claude-user skill scanner.
  - A child counts when it is a directory, or a symlink resolving to a directory, that contains `SKILL.md`.
  - Skip dangling symlinks, non-skill dirs, root files and `synced`.
  - Record the resolved path.
  - Record `ReadDir` success/failure as the root's scanned flag. Any error, including not-exist, means not scanned.
- [ ] 1.2 RED→GREEN: command scanner (`**/*.md`, following symlinks).
  - Name = relative path with `/` → `:` and `.md` stripped (`opsx/apply.md` → `opsx:apply`).
  - Used for `~/.claude/commands`, project `.claude/commands`, and plugin `commands/`, or plugin.json `commands` when that is a path or array. An object-map form is skipped with a warning and marked not scanned.
- [ ] 1.3 RED→GREEN: synced scanner.
  - Only directory children of `synced/` whose `manifest.json` parses. The stray `.bucket-*` file is ignored.
  - Scanned iff ≥1 manifest parsed.
  - Duplicate `<n>` across buckets: identical → collapse; different → conflict (task 2.3).
- [ ] 1.4 RED→GREEN: Pi discovery (`~/.pi/agent/skills`, `.pi/skills`, `~/.agents/skills`, project `.agents/skills` chain).
  - Recursive, depth-bounded, cycle-guarded by resolved path. A skill dir stops descent.
  - Root `.md` files are skills only for the two `.pi` roots.
- [ ] 1.5 RED→GREEN: Pi configured sources.
  - `settings.json` `skills` entries: paths relative to `~/.pi/agent` (or to `.pi`), `~` expansion, per-segment `path.Match` globs, `!pattern`, `+path`, `-path`. Unsupported `**` → warning, source not scanned.
  - `packages` in string and object forms: `npm:` → `<root>/npm/node_modules/<name>`; `git:`/URL → `<root>/git/<host>/<path>`; local path relative to the settings dir. Skills come from package.json `pi.skills`, else convention `skills/`. The object-form `skills` filter applies (omitted = all, `[]` = none). A project entry overrides a global one with the same identity.
  - Fixture of this machine's shape: a missing local package contributes nothing and is not scanned; an installed-but-unconfigured npm package (pi-intercom) is not read.
- [ ] 1.6 RED→GREEN: Pi trust.
  - The nearest `trust.json` decision for the cwd or a parent wins. No decision → `defaultProjectTrust == "always"` from global settings, otherwise untrusted.
  - Untrusted → no Pi project sources, and their roots are not scanned.
- [ ] 1.7 RED→GREEN: plugin scanner.
  - Parse `installed_plugins.json` v2 and `settings.json`.
  - Enabled = the `enabledPlugins` value, else plugin.json `defaultEnabled`, else true.
  - `scope` must be `user`, or `project`/`local` with `projectPath` equal to the repo top-level.
  - `installPath` must be readable. Skills = `skills/<n>/SKILL.md` + plugin.json `skills` paths. Commands per 1.2. Never glob `plugins/cache`.
  - A name repeating across marketplaces → `<plugin>@<marketplace>` qualification, and the unqualified scope is not scanned.
  - Reserved names are skipped with a warning.
  - Fixtures: disabled (hookify), absent entry with missing installPath (work-on), uninstalled plugin, two marketplaces.
- [ ] 1.8 RED→GREEN: project identity probe (via the injected Commander; extend, don't duplicate, `detectRepo`).
  - `<owner>/<repo>` from the origin URL, for ssh://, https:// and scp forms, with `.git` stripped.
  - No origin → basename of the parent of `git rev-parse --path-format=absolute --git-common-dir`.
  - Not a repo → no project sources.
  - Fixtures: a linked worktree resolves to `toejough/engram`; two repos named `engram` under different owners stay distinct.
- [ ] 1.9 RED→GREEN: Claude project chain.
  - `.claude/skills` and `.claude/commands` in the cwd and each ancestor up to `--show-toplevel`; nearer wins; root `.md` in `.claude/skills` ignored.
  - Scanned iff the top-level's dir read succeeded; a missing intermediate dir is absent, not failed.
  - Fixture: running from a subdirectory equals running from the top-level.
- [ ] 1.10 RED→GREEN: `ResolveSkillSources(home, cwd, deps)` composes all scanners in D2 precedence. Harnesses come from `supportedHarnesses`/`detectHarnesses` (not re-hardcoded). It returns candidates `{Key, ScopeID, ReadRoot, SourcePath, Kind, Content}` and `ScannedRoots`.

## 2. Identity, dedupe, conflicts, scoped removal (D3, D4, D5)

- [ ] 2.1 RED→GREEN (+ rapid property): key construction per D3.
  - `cmd` segment for commands.
  - Engram-owned-root rule: anything resolving under `<home>/<EngramRootRel>/skills` for any supported harness gets the bare key. Scenario: a Pi-only machine yields key `route` and a refresh offer for note 1036, not `pi:route`.
  - Key → slug always matches `^skill-[a-z0-9]+(-[a-z0-9]+)*$`, and a bare `x` → `skill-x`.
- [ ] 2.2 RED→GREEN: note identity.
  - Match by `skill_key` on a runbook with `skill_hash`.
  - Legacy fallback: the slug remainder is a bare key.
  - The note-820 fixture (a `skill-` slug without `skill_hash`) is not a skill note.
  - A duplicate match is an error naming both.
- [ ] 2.3 RED→GREEN (+ rapid property: the offer set is independent of scan order; an alias never creates an offer):
  - Same-path collapse.
  - Same key in the same scope: identical → collapse; different → conflict line naming both paths, no offer for that key, failure exit status after all other offers.
  - Cross-scope same key (engram-owned only) → precedence plus a warning.
  - Alias = SHA equal to a higher-precedence candidate or to any note's `skill_hash`.
- [ ] 2.4 RED→GREEN (+ rapid property: never a removal offer whose read root ∉ `ScannedRoots`, or whose key has any candidate, alias included):
  - key → read root mapping, and removal only for scanned roots.
  - Cover every spec scenario: another cwd; update outside a repo; disabled plugin; uninstalled plugin (offered); unreadable `~/.claude/skills`; synced without a manifest; alias keeps its note; absent harness; untrusted Pi project; `--skills-dir`.
- [ ] 2.5 RED→GREEN: `CompareSkillOffers` works over candidates and keys. Offers carry `Key`, `ScopeID`, `SourcePath` and `Kind`, sorted by (scope, key).

## 3. Answering at volume, declines v2 (D6, D7)

- [ ] 3.1 RED→GREEN: `skill-registrations.json` v2.
  - Read v1 as-is (fixture `{"schema_version":1,"declined":{"route":…}}`).
  - First write stamps v2 and keeps every entry. Version > 2 → error, nothing written.
  - Assert the file shape stays `{schema_version, declined}`, so the current (old) reader decodes a v2 file.
- [ ] 3.2 RED→GREEN (+ rapid property: for any offer, the winning answer is exact > longest pattern > selector; an identical token in both flags is always refused):
  - `--accept`/`--decline` take an exact key, a `prefix*` pattern, or `@<scope-id>` (including `@claude-user`, `@plugin:superpowers`, `@project:toejough/engram`).
  - Removal offers are matched only by an exact key.
  - A token matching no offer → the existing one-line note.
- [ ] 3.3 RED→GREEN: interactive grouped prompt.
  - One question per scope with >1 register/refresh offer: accept all / decline all / review each / skip. Skip and EOF record nothing.
  - A single offer and every removal use the per-offer y/N.
- [ ] 3.4 RED→GREEN: output.
  - Non-interactive summary: one line with the total plus `@<scope-id> <count>` labels and the answering command.
  - `--dry-run`: `would offer: <kind> <key> (<source>)` under scope headers.

## 4. Note content and fields (D3, D8, D9)

- [ ] 4.1 RED→GREEN: add `skill_key`/`skill_source` to the runbook frontmatter (`learn.go` `runbookFrontmatterDoc`, `renderRunbookFrontmatter` ~:755). Ordinary `engram learn` writes neither.
- [ ] 4.2 RED→GREEN: field survival (vault-note-identity ADDED requirement).
  - First, grep every frontmatter rewrite site (`marshalFrontmatter`, `yaml.Marshal`, and node-level edits in `amend.go`, `resituate.go`, `luhmann_reparent.go`/rename-and-rewrite, `identity_backfill.go`, `vocab_commands.go`, `vocab_apply.go`, `vocab_regen.go`, `offer.go`) and list them in the PR.
  - Then add one test per site proving that `skill_hash`, `skill_key` and `skill_source` are byte-unchanged after that rewrite, including `amend --clear-pending`.
- [ ] 4.3 RED→GREEN: Register, Refresh and Adopt write the key-derived slug plus `skill_key`/`skill_source`.
  - Refresh stamps both onto legacy notes and updates `skill_source` after a plugin version bump.
  - Preamble: engram-owned source → bytes identical to today's (assert for the six); otherwise the `~`-relative `skill_source` (skills and commands).
  - `--adopt` takes `<key>=<note-ref>`.
- [ ] 4.4 RED→GREEN: `--skills-dir` is repeatable and replaces the default set (Claude-user rules, bare keys). It is read-only: it implies `--dry-run`, refuses `--accept`/`--decline`/`--adopt` with an error before scanning, and makes no removal offers.

## 5. Wiring (D1, update-deploy-sync)

- [ ] 5.1 RED→GREEN: `registerSkillsTargets` resolves via `ResolveSkillSources(home, Getwd())`. Deps are composed in `newSkillRegistrationDeps`. `cmd/engram/main.go` gains only raw primitive references, if any (`targ check-thin-api` green).
- [ ] 5.2 RED→GREEN: `runUpdateSkillRegistration` calls the same resolver instead of `<sourceRoot>/agent-instructions/skills`.
  - Test: identical offers for the same fixture home, vault and cwd (update-deploy-sync scenario).
  - Verify the re-exec child (`update.Spawn`) inherits cwd.
- [ ] 5.3 Code-comment rows 13–19 of `enumeration.md`. Grep that each new text is present and the old "only shipped skills register" text is absent.
- [ ] 5.4 `targ check-full` green: lint, coverage floor, `check-thin-api`, nilaway. Collect all failures in one pass before fixing any.

## 6. Verification against the real layout (no paid eval)

- [ ] 6.1 Run `go install ./cmd/engram`, then `cd /tmp && engram register-skills --dry-run` (real home and vault, read-only). Expected, measured 2026-09-26 with the offer-computation rules applied to the real files: **64 offers, all `register`, no refresh, no removal, no conflict**, broken down as:

  | Scope | Offers |
  | --- | --- |
  | `@claude-user` (c4, dev, mycelium, property-rigor) | 4 |
  | `@claude-cmd` (`cmd:audit`) | 1 |
  | `@synced` | 12 |
  | `@pi-user` (`pi:ping`) | 1 |
  | `@agents-user` | 0 |
  | `@pi-settings` | 0 |
  | Pi packages (the configured local package is missing, so not scanned) | 0 |
  | `@plugin:superpowers` | 15 |
  | `@plugin:plugin-dev` (7 skills + `cmd:create-plugin`) | 8 |
  | `@plugin:traced` | 4 |
  | `@plugin:commit-commands` (`cmd:clean_gone`, `cmd:commit-push-pr`, `cmd:commit`) | 3 |
  | `@plugin:issue` | 3 |
  | `@plugin:claude-md-management` (1 + `cmd:revise-claude-md`) | 2 |
  | `@plugin:claude-hud` (`cmd:setup`, `cmd:configure`) | 2 |
  | `@plugin:commit` (`commit:commit`, `commit:cmd:commit`) | 2 |
  | 1 each: frontend-design, code-review (`cmd:code-review`), feature-dev (`cmd:feature-dev`), playground, skill-creator, claude-code-setup, readme-regen | 7 |

  Also confirm:
  - Engram's six (Claude and Pi copies) produce no offer.
  - Nothing is offered for the 22 dangling projctl symlinks, `.obsidian`, the `.bucket-*` file, hookify, ralph-loop, `engram@engram`, `work-on`, or pi-intercom.
  - No `project:*` offer appears.
  - Both `skill-creator:skill-creator` and `anthropic-skills:skill-creator` are offered, since their hashes differ.

  Explain any difference before continuing.
- [ ] 6.2 From this worktree, `engram register-skills --dry-run`. Expected **77 = 64 + 13 `@project:toejough/engram`**:
  - 6 `openspec-*` skills.
  - `cmd:commit`.
  - 6 `cmd:opsx:*`.

  The key uses `toejough/engram`, not the worktree name. `.claude/skills/{commit,engram-go-conventions}.md` are not offered. Also run from `internal/` and confirm the output is identical.
- [ ] 6.3 Verify on a scratch vault with a **minimal fixture home**. Never copy `~/.claude` wholesale, and never touch the real vault.
  - Setup: `cp -R` the real vault into the scratchpad and pass `--vault <copy>`. Build `<fx>/` containing:
    - `.claude/skills` → symlink to the real `~/.claude/skills`;
    - `.claude/commands` → symlink to the real one;
    - `.claude/plugins/cache` → symlink to the real cache;
    - `.claude/plugins/installed_plugins.json` → an edited copy;
    - `.claude/settings.json` → an edited copy holding only `enabledPlugins`;
    - `.pi/agent` → symlinks to the real `skills` and `engram`, plus an edited `settings.json` and `trust.json`.

    Run with `HOME=<fx>`.
  - Accept one plugin skill and one plugin command, decline `@synced`, and (from the worktree) accept `project:toejough/engram:cmd:opsx:apply`.
  - Check the accepted notes carry `skill_key`, `skill_source`, `pending: true`, the real-source preamble and a sidecar. Declines are v2, keyed by key. A second run offers none of them.
  - Check scoping:
    - Running from `/tmp` → no removal offer for the project note.
    - Setting the accepted plugin's `enabledPlugins` to `false` in the fixture → no removal offer.
    - Deleting it from the fixture `installed_plugins.json` → exactly its removal offers, each needing an exact-key answer.
    - `chmod 000` on a fixture copy of the skills dir → no removal offers for bare keys.
    - Removing the Pi `trust.json` entry → no Pi project sources.
  - `engram embed status --vault <copy>` is clean.
- [ ] 6.4 From **this worktree** (local-mode update; `engram update` from `/tmp` runs remote mode and does a network `git clone` into TMPDIR), run `engram update --dry-run`. Confirm its registration offers equal the 6.2 output.

## 7. Docs, archive, install

- [ ] 7.1 Perform doc rows 1–12 of `enumeration.md`. For each row, grep that the new text is present and the old scope-limiting text is absent before ticking. Get a fresh-context reviewer to check every row against its file.
- [ ] 7.2 Archive-time: rewrite the `Purpose` of `openspec/specs/skill-runbook-registration/spec.md` (enumeration row 20). Confirm `openspec archive` applied the RENAMED header ("…identified by its skill key") before the MODIFIED block, and that the synced spec has no "identified by its slug" header left.
- [ ] 7.3 Before archive, re-run the requirement-header collision sweep across all active changes and any archived-but-unsynced change, and read their tasks.md in full (vault notes 744/757).
- [ ] 7.4 Final install: `go install ./cmd/engram`, then `cd /tmp && engram register-skills --dry-run` once more. Record the output in the LEDGER row (enumeration row 11). Leave the real offers for Joe; never accept or decline on his behalf.
