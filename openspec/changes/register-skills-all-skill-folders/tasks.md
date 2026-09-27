Every Go task follows RED → GREEN → REFACTOR:
- Unit tests use gomega assertions with imptest/impgen mocks driven interactively.
- Every rule over keys, slugs, dedupe, precedence or scoping also gets a rapid property test (`property-rigor`).
- DI only: no `os.*` in `internal/`, and `cmd/engram/main.go` stays wiring-only.
- Run `targ test` / `targ check-full`, never `go test` directly.
- Fakes model: symlinked dirs; dangling symlinks; read errors distinct from not-exist; multi-version plugin caches; `enabledPlugins` true/false/absent; synced buckets alongside stray `.bucket-*` files; Pi `trust.json`; Pi settings/packages (vault notes 1066, 1073).

## 1. Source resolution (design D1, D2, D3 identity inputs)

- [x] 1.1 RED→GREEN: Claude-user skill scanner.
  - A child counts when it is a directory, or a symlink resolving to a directory, that contains `SKILL.md`.
  - Skip dangling symlinks, non-skill dirs, root files and `synced`.
  - Record the resolved path.
  - Record `ReadDir` success/failure as the root's scanned flag. Any error, including not-exist, means not scanned.
- [x] 1.2 RED→GREEN: command scanner (`**/*.md`, following symlinks).
  - Name = relative path with `/` → `:` and `.md` stripped (`opsx/apply.md` → `opsx:apply`).
  - Used for `~/.claude/commands`, project `.claude/commands`, and plugin `commands/`, or plugin.json `commands` when that is a path or array. An object-map form is skipped with a warning and marked not scanned.
- [x] 1.3 RED→GREEN: synced scanner.
  - Only directory children of `synced/` whose `manifest.json` parses. The stray `.bucket-*` file is ignored.
  - Scanned iff ≥1 manifest parsed.
  - Duplicate `<n>` across buckets: identical → collapse; different → conflict (task 2.3).
- [x] 1.4 RED→GREEN: Pi discovery (`~/.pi/agent/skills`, `.pi/skills`, `~/.agents/skills`, project `.agents/skills` chain).
  - Recursive, depth-bounded, cycle-guarded by resolved path. A skill dir stops descent.
  - Root `.md` files are skills only for the two `.pi` roots.
- [x] 1.4b RED→GREEN: Pi prompt-template scanner (`docs/prompt-templates.md`).
  - `~/.pi/agent/prompts/*.md` and project `.pi/prompts/*.md` (trusted only), flat, named by file stem. Kind is `prompt`.
  - Fixtures: a global prompt yields key `pi-prompt:review`; an untrusted project yields none.
- [x] 1.5 RED→GREEN: Pi configured sources.
  - `settings.json` `skills` entries: paths relative to `~/.pi/agent` (or to `.pi`), `~` expansion, per-segment `path.Match` globs, `!pattern`, `+path`, `-path`. Unsupported `**` → warning, source not scanned.
  - `packages` in string and object forms: `npm:` → `<root>/npm/node_modules/<name>`; `git:`/URL → `<root>/git/<host>/<path>`; local path relative to the settings dir. Prompts also come from settings `prompts` entries. Package contents match Pi's `collectPackageResources` (`dist/core/package-manager.js` ~1747-1785): for a string-form entry, any `pi` key → manifest entries only (`pi.skills`, `pi.prompts`), and convention `skills/`/`prompts/` only with no `pi` key; for an object-form entry, an explicit pattern list filters (`[]` = every file kept as disabled, per ruling R10), and an omitted type falls back per type to the manifest entry, else the convention dir; `autoload: false` → warning, not scanned. Fixtures: a string-form package whose `pi` has only `extensions` yields no skills even with `skills/` present; the same package as object-form with `skills` omitted yields `skills/fmt`. A project entry overrides a global one with the same identity.
  - Positive fixtures, one per spec scenario: a settings `skills` path (`~/extra-skills/fmt`); an npm package with `pi.skills` (listed `npm:pi-intercom`); object-form `skills: []` yields only disabled candidates (present for removal, never offered).
  - Fixture of this machine's shape: a missing local package contributes nothing and is not scanned; an installed-but-unconfigured npm package (pi-intercom) is not read.
- [x] 1.6 RED→GREEN: Pi trust.
  - The nearest `trust.json` decision for the cwd or a parent wins. No decision → `defaultProjectTrust == "always"` from global settings, otherwise untrusted.
  - Untrusted → no Pi project sources, and their roots are not scanned.
  - Fixture: no saved decision plus `defaultProjectTrust: "always"` → trusted (spec scenario).
- [x] 1.7 RED→GREEN: plugin scanner.
  - Parse `installed_plugins.json` v2 and `settings.json`.
  - Enabled = the `enabledPlugins` value, else plugin.json `defaultEnabled`, else true.
  - `scope` must be `user`, or `project`/`local` with `projectPath` equal to the repo top-level.
  - `installPath` must be readable. Skills = `skills/<n>/SKILL.md` + plugin.json `skills` paths. Commands per 1.2; a fixture where plugin.json `commands: ["./commands/setup.md"]` replaces a `commands/` dir that also holds `configure.md` (spec scenario). Never glob `plugins/cache`.
  - A name repeating across marketplaces → plugin conflict: a loud line, no offers for any of its keys, and its scope is not scanned (never re-keyed by marketplace).
  - Reserved names are skipped with a warning.
  - Fixtures: disabled (hookify), absent entry with missing installPath (work-on), uninstalled plugin, two marketplaces.
- [x] 1.8 RED→GREEN: project identity probe (via the injected Commander; extend, don't duplicate, `detectRepo`).
  - `<host>/<owner>/<repo>` from the origin URL, fully normalized (whole string lowercased; userinfo and port stripped; trailing `.git`/`/` removed), for ssh://, https:// and scp forms. rapid property: normalization is idempotent, and every userinfo/port/case variant of one URL yields the same `<r>`.
  - No origin → `local/<basename of the parent of git rev-parse --path-format=absolute --git-common-dir>`.
  - Not a repo → no project sources.
  - Fixtures: a linked worktree resolves to `github.com/toejough/engram`; github.com and gitlab.com repos with the same owner/repo stay distinct.
- [x] 1.9 RED→GREEN: Claude project chain.
  - `.claude/skills` and `.claude/commands` in the cwd and each ancestor up to `--show-toplevel`; root `.md` in `.claude/skills` ignored. The same name at two levels is a key conflict, never nearest-wins (spec scenario).
  - Record each directory that was read successfully as its own entry in `ScannedRoots`; a missing intermediate dir is absent, not failed.
  - Fixture: running from a subdirectory equals running from the top-level.
- [x] 1.10 RED→GREEN: `ResolveSkillSources(home, cwd, deps)` composes all scanners in D2 precedence. Harnesses come from `supportedHarnesses`/`detectHarnesses` (not re-hardcoded). It returns candidates `{Key, ScopeID, ReadRoot, SourcePath, Kind, Content}` and `ScannedRoots`.

## 2. Identity, dedupe, conflicts, scoped removal (D3, D4, D5)

- [x] 2.1 RED→GREEN (+ rapid property): key construction per D3.
  - `cmd` segment for commands.
  - *Superseded by 8.1 and 8.3 (round 3):* the engram-owned-root bare-key rule built here is deleted; engram's installed skills key by folder.
  - `pi-prompt` segment for prompts; entry names containing `:` are skipped with a warning; `pi-prompt` is added to the reserved plugin names.
  - Key → slug always matches `^skill-[a-z0-9]+(-[a-z0-9]+)*$`. *Superseded by 8.1:* the bare-`x` example; every key is qualified (`claude:x` → `skill-claude-x`).
- [x] 2.2 RED→GREEN: note identity.
  - Match by `skill_key` on a runbook with `skill_hash`.
  - *Superseded by 8.2 (round 3):* the legacy slug-remainder fallback built here is deleted.
  - The note-820 fixture (a `skill-` slug without `skill_hash`) is not a skill note.
  - A duplicate match is an error naming both.
- [x] 2.3 RED→GREEN (+ rapid property: the offer set is independent of scan order; an alias never creates an offer):
  - Same-path collapse.
  - Same key: identical → collapse; different → conflict line naming both paths, no offer for that key, failure exit status after all other offers. This includes the same name at two project-chain levels.
  - Plugin conflict (a name in two marketplaces) → no offers, scope not scanned.
  - *Superseded by 8.3 (round 3):* the engram-owned diverged-copy exception and the user-vs-engram collision rule built here are deleted.
  - Alias = SHA equal to a higher-precedence candidate or to any note's `skill_hash`.
- [x] 2.4 RED→GREEN (+ rapid property: never a removal offer for a note whose eligibility root (a fixed user root, the root containing its `skill_source`, or the plugin manifest rule) was not read, or whose key has any candidate, alias included):
  - Eligibility per D5, including matching the resolved `skill_source` prefix against `ScannedRoots` for synced (per bucket), `pi-settings`, `pi-pkg` and `project` notes.
  - Cover every spec scenario: another cwd; update outside a repo; nested-only `sub/.claude/skills/foo` from the top level; one failing synced bucket; a `pi-settings` note whose own entry is unreadable; a package dropped from settings (orphan, no offer); disabled plugin; plugin conflict; uninstalled plugin (offered); unreadable `~/.claude/skills`; alias keeps its note; absent harness; untrusted Pi project; `--skills-dir`.
- [x] 2.5 RED→GREEN: `CompareSkillOffers` works over candidates and keys. Offers carry `Key`, `ScopeID`, `SourcePath` and `Kind`, sorted by (scope, key).

## 3. Answering at volume, declines v2 (D6, D7)

- [x] 3.1 RED→GREEN: `skill-registrations.json` v2.
  - *Superseded by 8.6 (round 3):* v1 reading built here is dropped; any `schema_version` other than 2 is an error.
  - Version > 2 → error, nothing written.
  - Assert the file shape stays `{schema_version, declined}`, so the current (old) reader decodes a v2 file.
- [x] 3.2 RED→GREEN (+ rapid property: for any offer, the winning answer is exact > longest pattern > selector; an identical token in both flags is always refused):
  - `--accept`/`--decline` take an exact key, a `prefix*` pattern, or `@<scope-id>` (including `@claude-user`, `@plugin:superpowers`, `@project:github.com/toejough/engram`).
  - Removal offers are matched only by an exact key.
  - A token matching no offer → the existing one-line note.
- [x] 3.3 RED→GREEN: interactive grouped prompt.
  - One question per scope with >1 register/refresh offer: accept all / decline all / review each / skip. Skip and EOF record nothing.
  - A single offer and every removal use the per-offer y/N.
- [x] 3.4 RED→GREEN: output.
  - Non-interactive summary: one line with the total plus `@<scope-id> <count>` labels and the answering command.
  - `--dry-run`: `would offer: <kind> <key> (<source>)` under scope headers.

## 4. Note content and fields (D3, D8, D9)

- [x] 4.1 RED→GREEN: add `skill_key`/`skill_source` to the runbook frontmatter (`learn.go` `runbookFrontmatterDoc`, `renderRunbookFrontmatter` ~:755). Ordinary `engram learn` writes neither.
- [x] 4.2 RED→GREEN: field survival (vault-note-identity ADDED requirement).
  - First, grep every frontmatter rewrite site (`marshalFrontmatter`, `yaml.Marshal`, and node-level edits in `amend.go`, `resituate.go`, `luhmann_reparent.go`/rename-and-rewrite, `identity_backfill.go`, `vocab_commands.go`, `vocab_apply.go`, `vocab_regen.go`, `offer.go`) and list them in the PR.
  - Then add one test per site proving that `skill_hash`, `skill_key` and `skill_source` are byte-unchanged after that rewrite, including `amend --clear-pending`.
- [x] 4.3 RED→GREEN: Register, Refresh and Adopt write the key-derived slug plus `skill_key`/`skill_source`.
  - Refresh updates `skill_source` after a plugin version bump. *Superseded by 8.2:* refresh's legacy-note stamping case.
  - *Superseded by 8.3 (round 3):* the engram-owned preamble bytes; every preamble is ``> Mirrors skill `<skill_source>`.``
  - The skill, command and prompt source files are byte-identical after registration (MODIFIED "no engram-specific metadata" scenario).
  - `--adopt` takes `<key>=<note-ref>`.
- [x] 4.4 RED→GREEN: `--skills-dir` is repeatable and replaces the default set (Claude-user rules; *superseded by 8.5:* keys are `claude:<n>`, not bare). It is read-only: it implies `--dry-run`, refuses `--accept`/`--decline`/`--adopt` with an error before scanning, and makes no removal offers.

## 5. Wiring (D1, update-deploy-sync)

- [x] 5.1 RED→GREEN: `registerSkillsTargets` resolves via `ResolveSkillSources(home, Getwd())`. Deps are composed in `newSkillRegistrationDeps`. `cmd/engram/main.go` gains only raw primitive references, if any (`targ check-thin-api` green).
- [x] 5.2 RED→GREEN: `runUpdateSkillRegistration` calls the same resolver instead of `<sourceRoot>/agent-instructions/skills`.
  - Test: identical offers for the same fixture home, vault and cwd (update-deploy-sync scenario).
  - Verify the re-exec child (`update.Spawn`) inherits cwd.
- [x] 5.3 Code-comment rows 13–19 of `enumeration.md`. Grep that each new text is present and the old "only shipped skills register" text is absent.
- [x] 5.4 `targ check-full` green: lint, coverage floor, `check-thin-api`, nilaway. Collect all failures in one pass before fixing any.

## 6. Verification against the real layout (no paid eval)

- [x] 6.1 *(Superseded by 8.8: round 3's expected output. The table below is the pre-round-3 measurement.)* Run `go install ./cmd/engram`, then `cd /tmp && engram register-skills --dry-run` (real home and vault, read-only). Expected, measured 2026-09-26 with the offer-computation rules applied to the real files: **64 offers, all `register`, no refresh, no removal, no conflict**, broken down as:

  | Scope | Offers |
  | --- | --- |
  | `@claude-user` (c4, dev, mycelium, property-rigor) | 4 |
  | `@claude-cmd` (`cmd:audit`) | 1 |
  | `@synced` | 12 |
  | `@pi-user` (`pi:ping`) | 1 |
  | `@agents-user` | 0 |
  | `@pi-settings` | 0 |
  | Pi packages (the configured local package is missing, so not scanned) | 0 |
  | `@pi-prompt` (`~/.pi/agent/prompts` absent; no settings `prompts`) | 0 |
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
- [x] 6.2 *(Superseded by 8.8: round 3's expected output.)* From this worktree, `engram register-skills --dry-run`. Expected **77 = 64 + 13 `@project:github.com/toejough/engram`**:
  - 6 `openspec-*` skills.
  - `cmd:commit`.
  - 6 `cmd:opsx:*`.

  The key uses `github.com/toejough/engram`, not the worktree name. The engram repo has no `.pi/`, so no prompt or Pi project entries are added. `.claude/skills/{commit,engram-go-conventions}.md` are not offered. Also run from `internal/` and confirm the output is identical.
- [x] 6.3 Verify on a scratch vault with a **minimal fixture home**. Never copy `~/.claude` wholesale, and never touch the real vault.
  - Setup: `cp -R` the real vault into the scratchpad and pass `--vault <copy>`. Build `<fx>/` containing:
    - `.claude/skills` → symlink to the real `~/.claude/skills`;
    - `.claude/commands` → symlink to the real one;
    - `.claude/plugins/cache` → symlink to the real cache;
    - `.claude/plugins/installed_plugins.json` → an edited copy;
    - `.claude/settings.json` → an edited copy holding only `enabledPlugins`;
    - `.claude/engram` → symlink to the real `~/.claude/engram` (*superseded by 8.3:* it was required for the engram-owned-root rule, which is deleted);
    - `.pi/agent` → symlinks to the real `skills` and `engram`, plus an edited `settings.json` and `trust.json`.

    Run with `HOME=<fx>`.
  - *Superseded by 8.3 (round 3):* the engram-owned-rule assertion made here; that rule is deleted.
  - Accept one plugin skill and one plugin command, decline `@synced`, and (from the worktree) accept `project:github.com/toejough/engram:cmd:opsx:apply`.
  - Check the accepted notes carry `skill_key`, `skill_source`, `pending: true`, the real-source preamble and a sidecar. Declines are v2, keyed by key. A second run offers none of them.
  - Check scoping:
    - Running from `/tmp` → no removal offer for the project note.
    - Setting the accepted plugin's `enabledPlugins` to `false` in the fixture → no removal offer.
    - Deleting it from the fixture `installed_plugins.json` → exactly its removal offers, each needing an exact-key answer.
    - `chmod 000` on a fixture copy of the skills dir → no removal offers for its notes (then bare keys; `claude:` keys under 8.8).
    - Removing the Pi `trust.json` entry → no Pi project sources.
  - `engram embed status --vault <copy>` is clean.
- [x] 6.4 From **this worktree** (local-mode update; `engram update` from `/tmp` runs remote mode and does a network `git clone` into TMPDIR), run `engram update --dry-run`. Confirm its registration offers equal the 6.2 output.

## 7. Docs, archive, install

- [x] 7.1 Perform doc rows 1–12 of `enumeration.md`. For each row, grep that the new text is present and the old scope-limiting text is absent before ticking. Get a fresh-context reviewer to check every row against its file.
- [ ] 7.2 Archive-time: rewrite the `Purpose` of `openspec/specs/skill-runbook-registration/spec.md` (enumeration row 20). Confirm `openspec archive` applied the RENAMED header ("…identified by its skill key") before the MODIFIED block, and that the synced spec has no "identified by its slug" header left.
- [x] 7.3 Before archive, re-run the requirement-header collision sweep across all active changes and any archived-but-unsynced change, and read their tasks.md in full (vault notes 744/757).
- [x] 7.4 Final install: `go install ./cmd/engram`, then `cd /tmp && engram register-skills --dry-run` once more. Record the output in the LEDGER row (enumeration row 11). Leave the real offers for Joe; never accept or decline on his behalf.

## 8. Qualify every key (Joe's round-3 decision; design Context, D3, D4, D5, D7, D8, D9, D11)

Every key is qualified by its source, and there is no engram-owned mechanism (vault note 1073a). This section **supersedes ruling R39**: its parked residual, where a missing `~/.agents/skills` blocked every bare-key removal, disappears together with bare keys. Each task is RED → GREEN → REFACTOR under the conventions at the top of this file, and completed tasks above stay ticked.

**Ordering guard (D11).** Tasks 8.1–8.10 never run `go install`. Every real-binary check in 8.8 uses a binary built into the scratchpad. The global install happens only in 8.11, immediately before the adopt, in the step Joe approves. Until then, the installed `engram` (this branch's pre-round-3 build, ruling R36) still carries the legacy fallback, so it never offers the six `claude:<n>` registrations.

- [ ] 8.1 RED→GREEN (+ rapid property: every key `AssignSkillKeys` produces contains `:`, and its first segment maps to exactly one removal form): key construction (D3).
  - Claude user skills key as `claude:<n>`, and Claude user commands as `claude:cmd:<ns:…:n>`. Engram's installed skills key by folder (`claude:route`, `pi:route`), with no path-based exception.
  - `skillKeyRemovalForm` and `skillKeyScopeID` parse `claude:<n>` → `claude-user` and `claude:cmd:…` (three or more segments) → `claude-cmd`. A skill named `cmd` gives `claude:cmd` → `claude-user`. Remove the `cmd` → `claude-cmd` entry from `skillKeyPrefixForms`.
  - Slugs: `claude:route` → `skill-claude-route`, and `claude:cmd:audit` → `skill-claude-cmd-audit`.
- [ ] 8.1b RED→GREEN: reserved plugin names (D3; spec scenarios "A plugin named claude is reserved", "A disabled reserved-name plugin is silent", "A plugin named cmd is an ordinary plugin").
  - The reserved set is exactly the top-level key forms: `claude`, `pi`, `agents`, `project`, `anthropic-skills`, `pi-settings`, `pi-pkg` and `pi-prompt`. Add `claude` and drop `cmd`; `engram` is not reserved.
  - Move the reserved-name check (`skillsources_plugin.go:132`) to after enablement is decided (`claudePluginInstallToScan`). A disabled reserved-name plugin prints nothing, and an enabled one warns and is not scanned.
  - Fixtures: enabled `claude@m` warns; disabled `pi@m` is silent; `cmd@m` yields `cmd:x`; the real-shaped disabled `engram@engram` is silent.
- [ ] 8.2 RED→GREEN (+ rapid property: for any vault, adding or removing unkeyed notes never changes the offer set): delete the legacy slug-remainder fallback (D3, D11).
  - `groupSkillNoteCandidates` groups only notes that carry `skill_key`. An unkeyed `skill_hash` note is not matched, is not an alias source, and is never a removal candidate.
  - Fixture: the six legacy notes (slugs `skill-<n>`, `skill_hash` equal to the scanned bytes, no `skill_key`) yield exactly six `register claude:<n>` offers, and no refresh or removal.
  - **Keep** the unconditional `skill_key`/`skill_source` setter in `applySkillNoteBody` (`skillreg_accept.go:~317`), because adopt depends on it. Delete only the legacy wording in `RefreshSkill`'s doc comment (`skillreg_accept.go:178`, "stamping them onto a legacy note that lacks them") and the refresh-of-an-unkeyed-note test.
  - **Adopt test:** `--adopt claude:curate=1049` on an unkeyed legacy fixture note `1049.2026-09-21.skill-curate.md`, with at least one referrer linking to it. Assert:
    - the rename to `1049.2026-09-21.skill-claude-curate.md`;
    - the referrer's wikilink is rewritten and the referrer's sidecar is rebuilt;
    - the preamble is ``> Mirrors skill `<skill_source>`.``;
    - `skill_key: claude:curate` and `skill_source` are stamped, `skill_hash` is unchanged, and there is no `pending`.
  - Update the base-scenario tests: "Note located by slug" (a slug alone locates nothing) and "Duplicate notes are an error" (two notes with `skill_key: claude:curate`).
- [ ] 8.2b RED→GREEN (+ rapid property: an unrecognized key, and only an unrecognized key, is never matched and never removal-eligible for any roots and candidates; every other generated `<x>:<n>` with `<x>` not a reserved form is judged by the plugin rule): unrecognized `skill_key` (D3; spec scenario "An unrecognized skill_key is never matched or removed").
  - The key parser reports "unrecognized" in exactly three cases: no `:` (e.g. `route`); an empty segment (e.g. `claude:`, `pi::x`); a reserved-form prefix whose tail lacks that form's shape (e.g. `project:github.com/toejough/engram`, `pi-pkg:x`, `claude:cmd:`). Eligibility treats unrecognized as never eligible.
  - Any other `<x>:<n…>` is plugin `<x>`'s key under D5's plugin rule. Fixture: `engram:route` with no `engram` plugin in a readable `installed_plugins.json` and a parsed `settings.json` → offered for removal, exactly like `ralph-loop:cmd:help`.
- [ ] 8.3 RED→GREEN: delete the whole engram-owned mechanism, the multi-root bare-key eligibility and the collision special cases (design Context, D1, D4, D5, D8).
  - Delete, by name:
    - `ResolveEngramSkillRoots` and `underEngramSkillRoot` (`skillkeys.go:55-84`, `:220-230`), and `engramRootUnresolvedWarningFormat` (`skillkeys.go:117`);
    - `ResolvedSkillSources.EngramSkillRoots`/`EngramSkillRootsUnresolved` (`skillsources_resolve.go:35-42`, `:116-121`);
    - `offerEngramOwned` (`skilloffers.go:494-515`), `SkillOffer.EngramOwned` (`skillreg.go:44-48`) and `SkillNoteSource.EngramOwned` with the engram branch of `PreamblePath` (`skillreg_accept.go:62-104`);
    - `engramCopiesRead` and the `claude-user` special case in `eligible` (`skilloffers.go:254-284`), and the `engramRoots`/`engramUnresolved` fields (`skilloffers.go:236-240`, `:479-480`);
    - `allEngram` and D4's diverged-engram exception (`skilloffers.go:443-459`), `engramCopiesDifferWarningFormat` (`skilloffers.go:146-148`), and the `engramRoots` parameters of `dedupeSkillCandidates`/`judgeSkillKeyGroup`/`AssignSkillKeys`/`NewSkillNoteSource`;
    - `update.EngramOwnedSkillsRels` (`internal/update/update.go:1459-1474`), deleted if it has no **non-test** caller once `skillkeys.go` is gone. Its test `TestEngramOwnedSkillsRels` goes with it.
  - `offerPromptSource` (`skillanswers.go:385-394`) shows the source for every offer that has one.
  - One preamble format for every note, ``> Mirrors skill `%s`.`` (`skillNotePreambleFormat`, `skillreg_accept.go:280`), without "edit the procedure there …". The register provenance `source: skill registration: <skill_source>` also names `skill_source`.
  - Tests: a real `~/.claude/skills/route` plus an engram-linked `~/.pi/agent/skills/route` → `claude:route` and `pi:route`, no conflict, no warning; diverged Claude and Pi copies → `pi:route` offered, no warning (spec scenario); a byte-identical Pi copy → `pi:route` alias, no offer; accepting `claude:curate` from `~/.claude/engram/skills/curate` → a preamble of exactly ``> Mirrors skill `~/.claude/engram/skills/curate/SKILL.md`.``
  - Deletion grep, which must return nothing: `grep -rn -i -E "engram-?owned|engramRoot|EngramSkillRoot|allEngram|engramCopies|bare[ -]key" internal/cli/skill*.go`
- [ ] 8.4 RED→GREEN (+ rapid properties: (a) **namespace isolation**: flipping the read status of any root whose form differs from a note's key form never changes that note's eligibility; (b) a `claude:` note is eligible exactly when every root recorded with form `claude-user` was read, and the same holds for each fixed form): per-namespace removal eligibility (D5).
  - Spec scenarios: "Another namespace's missing root does not block removal" (the R39 case: no `~/.agents/skills`, EACCES on `~/.pi/agent/skills`, `claude:c4` offered for removal); "Unreadable user skills directory"; "Unkeyed legacy note is never offered for removal".
- [ ] 8.5 RED→GREEN: `--skills-dir` entries key as `claude:<n>` in scope `claude-user`, and the run stays read-only (D9, spec scenario "Skills-dir runs are preview-only").
- [ ] 8.6 RED→GREEN: drop v1 decline reading (D7; spec scenarios "Version-1 file fails loudly", "Unknown schema version fails loudly"). Any `schema_version` other than 2, missing included, is an error, and nothing is written; a missing file means no declines. Delete the v1 test fixture and the v1 comments (`skillreg.go:84`, `:167`).
- [ ] 8.7 Code comments and `targ check-full`.
  - Rewrite or delete the stale comments, including these, found by grep at `cec56b71`:
    - `skillanswers.go:386`;
    - `skillkeys.go:15-32`, `:90`;
    - `skillreg.go:44`, `:61`, `:226`;
    - `skillsources.go:65`;
    - `skilloffers.go:42`, `:59`, `:80`, `:222`, `:438`;
    - `skillsources_resolve.go:37`, `:74-75`;
    - `skillreg_run.go:45`, `:141`, `:441-442`.
  - Re-run 8.3's deletion grep; it must return nothing. `internal/cli/update.go` has only 1,055 lines, so its registration comments are checked by `grep -n -i -E "bare[ -]key|engram-?owned (skills? )?root" internal/cli/update.go` and reviewed by hand. Deploy-sync's own uses of "engram-owned root" in `internal/update/update.go` stay, apart from lines 1459-1474 (8.3).
  - `targ check-full` green; collect every failure before fixing any.
- [ ] 8.8 Re-run the real-layout verification before migration, written out in full. **Never run `go install` here** (ordering guard). Record any difference and explain it (ruling R4).
  1. Build: `go build -o <scratchpad>/engram-r3 ./cmd/engram`. Call it `$BIN` below.
  2. From `/tmp`: `cd /tmp && $BIN register-skills --dry-run </dev/null` (real home, real vault, read-only). Expect **70 offers, all `register`, no refresh, no removal, no conflict, no warning**, with these scopes:
     - `@claude-user` **10**: `claude:{c4,dev,mycelium,property-rigor}` plus `claude:{curate,learn,please,recall,route,write-memory}`;
     - `@claude-cmd` 1 (`claude:cmd:audit`), `@synced` 12, `@pi-user` 1 (`pi:ping`);
     - `@plugin:superpowers` 15, `@plugin:plugin-dev` 8, `@plugin:traced` 4, `@plugin:commit-commands` 3, `@plugin:issue` 3, `@plugin:claude-md-management` 2, `@plugin:claude-hud` 2, `@plugin:commit` 2;
     - one each: `@plugin:{frontend-design,code-review,feature-dev,playground,skill-creator,claude-code-setup,readme-regen}`.

     Also confirm: no `pi:` offer for engram's six (the Pi copies are SHA-identical aliases); no bare key anywhere; no reserved-name warning (the disabled `engram@engram` is silent); nothing for the dangling projctl links, `.obsidian`, `.bucket-*`, hookify, ralph-loop, `work-on` or pi-intercom.
  3. From this worktree, and again from `internal/`: `$BIN register-skills --dry-run </dev/null`. Expect **83 = 70 + 13** `@project:github.com/toejough/engram`, with identical output from both directories.
  4. From this worktree: `$BIN update --dry-run </dev/null`. A dry run never installs or re-execs (`internal/update/update.go:300`, `:1262`). Its registration offers must equal step 3's.
  5. **Adopt rehearsal on a FRESH vault copy with the REAL HOME.** Adopt writes only to the vault. Do not reuse 6.3's mutated copy or its fixture HOME: under a fixture HOME, `homeRelativePath` yields absolute `skill_source` paths.
     - `cp -R ~/.local/share/engram/vault <scratchpad>/vault-r3`. This is the literal vault path: `XDG_DATA_HOME` is unset on this machine (checked 2026-09-26), and Joe's shell is fish, so no bash `${…:-…}` syntax is used. Run every command in 8.8 as written; each is valid in both fish and bash.
     - `cd /tmp && $BIN register-skills --vault <scratchpad>/vault-r3 --adopt claude:route=1036 --adopt claude:please=1045 --adopt claude:curate=1049 --adopt claude:write-memory=1053 --adopt claude:learn=1067 --adopt claude:recall=1068 </dev/null`. Non-interactive: it prompts for nothing and writes no other offer.
     - Check each of the six:
       - it is renamed `<id>.<date>.skill-claude-<n>.md`, and no `skill-<n>` basename remains;
       - every inbound wikilink points to the new basename (`grep -rn "skill-<n>\]\]"` finds none);
       - `skill_hash` is unchanged;
       - the body below the preamble line is byte-identical to the original's (diff against the real vault's note);
       - the preamble is exactly ``> Mirrors skill `~/.claude/engram/skills/<n>/SKILL.md`.``;
       - `skill_key: claude:<n>` and `skill_source: ~/.claude/engram/skills/<n>/SKILL.md` are present;
       - there is no `pending`.
     - `cd /tmp && $BIN register-skills --vault <scratchpad>/vault-r3 --dry-run </dev/null` → **64**, none for engram's six. From this worktree with the same `--vault` → **77**.
     - `$BIN embed status --vault <scratchpad>/vault-r3` is clean.
  6. **Scoping under a minimal fixture home built from REAL directories.**
     - **Safety rule:** nothing reached through a symlink into Joe's real home is ever chmod-ed, deleted or written. Every `chmod` and `rm` below acts only on a real directory or file that the fixture itself created. Before each one, check that the target is not a symlink (`test ! -L <target>`) and that `realpath <target>` starts with `realpath <fx>`. macOS `chmod` follows symlinks, so a symlink is never chmod-ed.
     - **Build `<fx>` = `<scratchpad>/fx-r3`** (not 6.3's fixture, whose skills folders are symlinks into the real home):
       - `<fx>/.claude/skills`: a real directory holding **copies** made with `cp -RL` from `~/.claude/skills/` of `c4`, `dev`, `mycelium`, `property-rigor`, the six engram skills and `synced`. The copies are byte-identical, so keys and hashes match the real layout.
       - `<fx>/.claude/commands`: a real directory holding a `cp -L` copy of `audit.md`.
       - `<fx>/.claude/plugins/installed_plugins.json` and `<fx>/.claude/settings.json`: edited copies. `installPath`s still point into the real `~/.claude/plugins/cache`, which is only read and never modified.
       - `<fx>/.pi/agent/skills`: a real directory holding `cp -RL` copies of `ping` and the six engram skills; plus copied `settings.json` and `trust.json`.
       - No `<fx>/.agents`.
     - **Fresh vault:** a **separate** fresh vault copy, `cp -R ~/.local/share/engram/vault <scratchpad>/vault-r3-scope`. Run every command as `env HOME=<fx> $BIN register-skills --vault <scratchpad>/vault-r3-scope … </dev/null`.
     - **Seed the notes the checks need**, on this scratch copy only:
       - from `/tmp`: `--accept claude:c4`, one plugin skill, and one plugin command; `--decline @synced`. Confirm the declines are written as v2 with qualified keys.
       - from this worktree: `--accept project:github.com/toejough/engram:cmd:opsx:apply`.
       - Confirm each accepted note carries `skill_key`, a `~`-relative `skill_source` under the fixture home, `pending: true`, the one-line preamble and a sidecar.
     - **Scoping checks.** Each one starts from the restored fixture:
       - From `/tmp`: no removal offer for the `project:github.com/toejough/engram:cmd:opsx:apply` note (its folder is not read there).
       - Set the accepted plugin's `enabledPlugins` to `false` in `<fx>/.claude/settings.json` → no removal offer for its notes. Then restore it.
       - Delete that plugin from `<fx>/.claude/plugins/installed_plugins.json` → exactly its removal offers, each needing an exact-key answer. Then restore the file.
       - `chmod 000 <fx>/.claude/skills` (the fixture's own real directory) → no removal offer for `claude:c4`. Then `chmod 755` it.
       - **The R39 case:** with no `<fx>/.agents`, run `chmod 000 <fx>/.pi/agent/skills` and `rm -r <fx>/.claude/skills/c4` (the fixture's own copy) → a removal offer for `claude:c4` appears. Then `chmod 755 <fx>/.pi/agent/skills`.
     - `$BIN embed status --vault <scratchpad>/vault-r3-scope` is clean.
- [ ] 8.9 Docs: perform enumeration rows 26–35 and 37 (row 36 is done by 8.11, row 38 by 8.7). For each row, grep that the new text is present and the old text (bare keys `route`/`cmd:audit`, "bare `<n>`", "engram's own six keep bare keys", the old `skill-<n>` slugs of the six) is absent. Then get a fresh-context reviewer to check every row against its file (vault note 1072).
- [ ] 8.10 Before archive, re-run 7.3's requirement-header collision sweep across all active changes (vault notes 744/757). Then run `openspec validate register-skills-all-skill-folders --strict`.
- [ ] 8.11 **Final step, only with Joe's explicit approval: install and migrate** (D11). Install and adopt form one approved step, so no installed binary ever shows the six offers against unkeyed notes.
  - Show Joe both commands and **stop until he says yes**:
    `go install ./cmd/engram`
    `cd /tmp && engram register-skills --adopt claude:route=1036 --adopt claude:please=1045 --adopt claude:curate=1049 --adopt claude:write-memory=1053 --adopt claude:learn=1067 --adopt claude:recall=1068 </dev/null`
  - The adopt runs **non-interactively** (stdin from `/dev/null`), so it never prompts for, accepts or declines any of the remaining offers. They are only reported.
  - Before running, confirm that no `skill-claude-<n>` note exists in the real vault.
  - Run the install, then the adopt, immediately.
  - After they run:
    - `cd /tmp && engram register-skills --dry-run </dev/null` shows **64** offers, none for engram's six; from this worktree it shows **77**;
    - `engram embed status` is clean;
    - `git -C <vault> status` shows the six renames plus the referrers whose links were rewritten. Commit the vault.
  - Append the result to the LEDGER row (enumeration row 36). Leave every remaining offer for Joe.
