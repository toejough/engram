# Proposal: update-deploy-as-sync

## Why

`engram update` deploys additively: it copies skills, commands, and guidance into harness directories but never removes anything, so artifacts that leave the source — renamed skills, deleted guidance docs, strays — persist in every install until hand-deleted (2026-07-24 instance: a stray `metadata_test.go` deployed as a skill directory outlived its source-side fix). Deployed state should match source intent exactly (#706, Joe: "our deployed files should match our intent, not just be additive"), and the deploy surface is about to grow (OpenCode guidance #667, Shelley #707), multiplying the drift.

## What Changes

- **Deploys become syncs.** Each harness gets an engram-owned root that `engram update` syncs to exactly match `agent-instructions/` — additions, updates, and **removals** all propagate. Deletion inside the engram-owned root is safe by construction (engram is its sole writer).
- **Harness-visible paths become symlinks.** Where a harness requires artifacts at fixed paths (e.g. `~/.claude/skills/<name>/`), `engram update` places symlinks into the engram-owned root instead of real copies.
- **Dangling-link cleanup.** On every update, harness dirs are scanned for symlinks pointing into the engram-owned root whose targets no longer exist; those links are deleted. This is how removals reach harness-visible paths without touching user-owned files.
- **One-time migration.** Existing real-file deployments (previously-copied engram artifacts at harness paths) are converted to symlinks on first sync; files engram cannot prove it owns are left alone.
- **Symlink-discovery verification gates the design per harness.** Before the symlink surface ships for a harness, that harness's skill/command discovery through symlinks is verified. A harness that fails verification falls back to a recorded deploy-manifest (track what engram wrote; delete recorded files whose sources left) instead of symlinks.
- **User-owned files are never deleted.** The only deletions outside the engram-owned root are symlinks that provably point into it.

## Capabilities

### New Capabilities

- `update-deploy-sync`: the sync deployment model of `engram update` — engram-owned per-harness root synced to match source exactly, symlinked harness-visible paths, dangling-link cleanup, ownership-bounded deletion, and per-harness fallback behavior.

### Modified Capabilities

_None — no existing spec covers `engram update` deployment; this change introduces the first one._

## Impact

- `internal/update/update.go` — planning and apply paths reshape from per-file CopyOps into sync-plan + link-plan + cleanup phases; `clearSkillDirOnce`/`applyCmdOne` deletion-within-artifact logic subsumed by the sync.
- `internal/update` `Filesystem` interface — needs symlink primitives (create symlink, read link target, lstat); per DI-everywhere, new methods thread through `internal/cli/primitives.go` (`cli.Primitives`/`cli.NewDeps`) and `cmd/engram/main.go`'s checker-thin per-group functions.
- `internal/update` tests — sync/link/cleanup/migration coverage (imptest mocks + rapid properties + gomega, per repo test stack).
- Docs — `docs/FEATURES.md` update surface, a new ADR in `docs/architecture/adr.md` for the deployment-ownership model, README/update docs describing sync semantics and migration.
- Issue #706 closes on completion; #709 (dry-run line prefixes) lives in the reshaped apply path and should be reconciled during implementation.
- Unchanged: what gets deployed (the skills/commands/guidance sets), the opt-in `--with-guidance` gating, source layout under `agent-instructions/`, harness detection.
