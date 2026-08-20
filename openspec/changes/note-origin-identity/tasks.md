## 1. Primitives

- [x] 1.1 Add a machine-username primitive to `Primitives` (wrapping `os/user.Current()` in `cmd/engram`), mirroring how `Getwd: prims.Proc.Getwd` is already wired
- [x] 1.2 Wire the new primitive through `cli.NewDeps` composition

## 2. Repo detection

- [x] 2.1 Add a repo-detection helper: `git remote get-url origin` (via the existing `Commander`, `dir` from `deps.Getwd()`) first, falling back to `git rev-parse --show-toplevel` + basename when no `origin` remote exists
- [x] 2.2 Resolve to empty string (no error) when the working directory is not inside a git repo, so the caller can omit the field rather than fail the whole `learn` call
- [x] 2.3 Compose the helper into `LearnDeps` via `newLearnDeps`

## 3. User detection

- [x] 3.1 Add a user-detection helper: `git config user.email` (via `Commander`) first, falling back to the new machine-username primitive when empty
- [x] 3.2 Compose the helper into `LearnDeps` via `newLearnDeps`

## 4. Vault name resolution

- [x] 4.1 Add `VaultName string` to `LearnArgs` with a `targ:"flag,name=vault-name,env=ENGRAM_VAULT_NAME,..."` struct tag, matching `VaultPath`'s existing flag/env shape
- [x] 4.2 Default to `"personal"` when both the flag and env var resolve empty

## 5. Frontmatter schema

- [x] 5.1 Add `Repo` (`yaml:"repo,omitempty"`), `User` (`yaml:"user"`), `Vault` (`yaml:"vault"`) fields to `factFrontmatterDoc` and `feedbackFrontmatterDoc` (`learn.go:142-191`), positioned after `Source`/`Project` in field order
- [x] 5.2 Thread the three fields through `factFields`/`feedbackFields` and `RunLearn`'s render path, populated from the detection helpers composed in tasks 2-4

## 6. Amend re-stamping

- [x] 6.1 Compose the repo/user detection helpers (and vault-name resolution) into `AmendDeps` via `newAmendDeps`, mirroring `newLearnDeps`
- [x] 6.2 Add `VaultName` flag/env resolution to `AmendArgs`, matching `LearnArgs`
- [x] 6.3 In `applyTypedAmend`'s override step, unconditionally re-stamp `Repo`/`User`/`Vault` on the parsed `doc` from fresh detection — `Repo` uses the plain repo-detection helper (task 2) directly, with **no** project-field fallback (see task 7) — confirm this does not set `contentChanged` (matches the existing provenance-only category, `amend.go:171`)

## 7. Repo fallback for backfill only

- [x] 7.1 Add a repo-resolution helper used by backfill only: prefer the note's existing `project:` field when non-empty, else fall back to the existing repo-detection helper (task 2)
- [x] 7.2 Wire this helper into the backfill command (task 8) — `amend`'s re-stamp step (task 6.3) does not use it

## 8. Identity backfill (`engram update --backfill-identity`)

- [x] 8.1 Add a `notesMissingIdentityFields` detector (vault scan: note has no `repo:`/`user:`/`vault:`), following the shape of the existing five `runUpdate` detectors (ADR-0021)
- [x] 8.2 Add the corresponding notify-only notice and wire it into `writeUpdateReport`, alongside the existing five
- [x] 8.3 Add a `--backfill-identity` flag to `UpdateArgs`
- [x] 8.4 Implement the backfill: for each flagged note, stamp `user:`/`vault:` from current detection and `repo:` via the task-7 fallback helper; rewrite only these three fields, leaving everything else (including `Created`) untouched
- [x] 8.5 Confirm backfill is idempotent — a second run with nothing newly missing modifies no files

## 9. Tests

- [x] 9.1 Repo detection: origin remote present, origin absent (dirname fallback), not-a-git-repo (omitted)
- [x] 9.2 User detection: `user.email` present, `user.email` absent (machine-user fallback)
- [x] 9.3 Vault-name resolution: flag set, env set, neither set (defaults to `"personal"`)
- [x] 9.4 `RunLearn`: fact and feedback notes both render `repo:`/`user:`/`vault:` correctly in frontmatter
- [x] 9.5 `RunAmend`: amending a note whose `repo:`/`user:`/`vault:` differ from the amend-time environment overwrites those fields with the current environment's freshly detected values
- [x] 9.6 `RunAmend`: amending a pre-existing note with no `repo:`/`user:`/`vault:` fields adds them, same as any other amend
- [x] 9.7 `RunAmend`: identity-only re-stamp (no content flags passed) does not set `contentChanged` / does not trigger re-embed
- [x] 9.8 `RunAmend`: repo re-stamp ignores `project:` — always freshly detected from the current working directory, even when it differs from the note's `project:` value
- [x] 9.9 Repo fallback (backfill only): `project:` present → used verbatim as `repo:`; `project:` absent → falls back to fresh git detection
- [x] 9.10 `RunUpdate`: `notesMissingIdentityFields` detector fires the notify-only notice when notes are missing fields, stays silent when none are
- [x] 9.11 `RunUpdate --backfill-identity`: stamps missing notes correctly and is idempotent on a second run

## 10. Verification

- [x] 10.1 `targ check-full` green
- [x] 10.2 Install the real binary (`go install ./cmd/engram`) and run `engram learn fact ...` in this repo to confirm `repo:`/`user:`/`vault:` render correctly against this environment's actual git remote/config (cli-verification convention — real binary, real filesystem, not just unit tests)
- [x] 10.3 Real-binary check: `engram amend` on a note re-stamps `repo:`/`user:`/`vault:` to the current environment; `engram update --backfill-identity` against a vault with a hand-edited note missing those fields backfills them correctly
