## Why

Every vault note carries only a free-text `source:` field for provenance — a human-written description with no structured signal for who wrote it, from which repo, or through which vault instance. Issue #720 (exposing a client/server API so remote environments can write into the vault) needs structured, auto-stamped identity so a note's origin survives beyond one person's prose description; the identity mechanism itself has no server dependency and is useful for today's local-only `engram learn`/`amend` as-is, so it ships as its own change.

## What Changes

- `engram learn` auto-stamps three new frontmatter fields at note creation, additive alongside the existing `source:` field (which is unchanged):
  - `repo`: the caller's git remote URL (`git remote get-url origin`, run from the working directory), falling back to the git root directory's basename when no `origin` is configured. Omitted (`omitempty`) when the working directory isn't inside a git repo.
  - `user`: the caller's `git config user.email`, falling back to the machine's OS username when git config resolves to nothing.
  - `vault`: resolved from a new `--vault-name` flag / `ENGRAM_VAULT_NAME` env var, mirroring `ENGRAM_VAULT_PATH`'s existing flag → env → default resolution order. Defaults to `"personal"` when unset.
- `engram amend` re-stamps `repo`/`user`/`vault` fresh on every call, using the same detection as `learn` — unlike `source`/`project`/`issue`/`tier`, which continue to be preserved verbatim. These three fields track the *last writer*, not the note's origin: every `learn` or `amend` call overwrites them with the current environment's values, regardless of what the note previously held.
- Repo detection for the backfill command (below) prefers the note's own `project:` field over a fresh git-remote read wherever `project:` is non-empty, since a single backfill invocation runs from one working directory but may touch notes that originated in different repos. `amend`'s repo re-stamp does **not** use this fallback — it always uses fresh detection from the current working directory, same as `learn`: an amend is assumed to run from the repo it's actually about, so its `repo:` value should reflect exactly where the amend was fired from, even if that differs from the note's `project:`.
- `engram update` gains a sixth vault-condition detector, following the existing detect-and-notify convention (ADR-0021: vocab migration, vocab self-tags, Luhmann branching, empty chunks, duplicate chunks) — it flags notes missing `repo`/`user`/`vault` and points to a new `--backfill-identity` flag that stamps them: `user`/`vault` from the current environment (safe to blind-stamp — vault-wide constants), `repo` from the note's `project:` field where present, else freshly detected. Backfilled values are indistinguishable from normally-stamped ones — no marker, no separate field.
- Detection runs client-side at `learn`, `amend`, and backfill time — never asserted by a remote party — using the caller's own working directory and environment at the moment of the write. This is a deliberate, explicit v1 trust decision: identity is self-reported (auto-detected, not cryptographically verified), matching how `source:` already works today. A credential-verified identity model is out of scope here and can replace this resolution later without changing the frontmatter shape.
- New injected primitives (DI-everywhere, `internal/cli` composition root): a machine-username primitive (no existing precedent in this codebase) and a git-config-read helper, composed alongside the already-injected `Getwd` and `Commander` (`RunCommand`) primitives that `ingest` and `update` already use for working-directory resolution and git shell-outs respectively.

## Capabilities

### New Capabilities
- `vault-note-identity`: auto-stamped `repo`/`user`/`vault` provenance fields, re-stamped fresh on every note write (`learn` or `amend`), with an opt-in `engram update --backfill-identity` path for notes written before this capability existed.

### Modified Capabilities
(none — no existing spec defines the note frontmatter field contract; this is additive)

## Impact

- `internal/cli/learn.go`: `factFrontmatterDoc`/`feedbackFrontmatterDoc` gain `Repo`, `User`, `Vault` fields; `LearnDeps` gains injected detection functions; `newLearnDeps` composes them from `cli.Deps`. `internal/cli/amend.go` gains the same detection functions via `AmendDeps`/`newAmendDeps`, and calls them unconditionally on every amend to re-stamp `Repo`/`User`/`Vault` — a provenance-only mutation (matching the existing provenance-only/supersedes-only category at `amend.go:171`) that does not set `contentChanged` and so never triggers re-embed. `AmendArgs` gains its own `--vault-name` flag/env resolution, mirroring `LearnArgs`.
- `internal/cli/update.go`: gains a sixth detector (alongside the existing five, ADR-0021) that flags notes missing `repo`/`user`/`vault`, a corresponding notify-only notice in `writeUpdateReport`, and a new `--backfill-identity` flag on `UpdateArgs` that performs the actual rewrite when the user opts in.
- `internal/cli/primitives.go` / `commander.go`: new machine-username primitive; existing `Commander`/`Getwd` reused for git remote and git config reads — shared across `learn`, `amend`, and `update`'s backfill path, not `learn`-only.
- `internal/cli/targets.go`: new `--vault-name` flag wiring on both `learn` and `amend` (`ENGRAM_VAULT_NAME` env, default `"personal"`), same shape as the existing `--vault`/`ENGRAM_VAULT_PATH` flag; new `--backfill-identity` flag on `update`.
- No change to `query`/`show`/`activate` or any read path — this is write-path only.
