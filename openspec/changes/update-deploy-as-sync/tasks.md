# Tasks: update-deploy-as-sync

## 1. Symlink-discovery verification (D7 gate — decides each harness's deploy mode)

- [x] 1.1 Verify Claude Code discovers a skill through a symlinked skill dir (`~/.claude/skills/<name>` → elsewhere): place a test skill via symlink, confirm it loads in a real session; record verdict in the change dir
- [x] 1.2 Verify OpenCode discovers symlinked skills and commands the same way; record verdict
- [x] 1.3 Verify Pi discovers symlinked skills and guidance files; record verdict
- [x] 1.4 Set each harness's deploy mode (symlink vs manifest) in `supportedHarnesses` per the recorded verdicts

## 2. Filesystem symlink primitives (D8)

- [x] 2.1 Add `Symlink`, `ReadLink`, and lstat-shaped type probing to the `internal/update` `Filesystem` interface (TDD: unit tests with imptest mocks first)
- [x] 2.2 Wire OS adapters through `internal/cli/primitives.go` (`cli.Primitives`, `cli.NewDeps`) and `cmd/engram`'s checker-thin per-group functions; `targ check-thin-api` stays green

## 3. Sync engine (D1, D2, D4)

- [x] 3.1 Add engram-owned root path + ownership marker handling per harness: create root+marker when absent-and-empty, detect marker, refuse sync-deletion without it (TDD)
- [x] 3.2 Implement intended-set computation reuse: planners' output diffed against the owned root → create/overwrite/delete op plan (TDD, rapid property: sync of any intended set into any root state yields root ≡ intended set)
- [x] 3.3 Implement guidance opt-in gating in the sync plan: guidance subtree unmanaged (never synced-to-empty) when not in the intended set (TDD)
- [x] 3.4 Report every sync deletion and every unattributable file in the update `Report` (TDD)

## 4. Symlink materialization + cleanup (D3, D5)

- [x] 4.1 Materialize harness-visible surfaces as symlinks for symlink-mode harnesses: per-skill-dir, per-command-file, per-guidance-file (TDD)
- [x] 4.2 Implement dangling-link cleanup: one-level scan of each surface dir, lexical target resolution (no EvalSymlinks — note-475 rule), delete only engram-root-targeted dangling links (TDD; include the symlink-free-tree regression test asserting recorded paths unchanged)

## 5. First-sync migration (D6)

- [x] 5.1 Implement adoption: intended-set artifacts found as real files/dirs at harness paths are replaced with root content + symlink (TDD)
- [x] 5.2 Implement stray reporting: real files at harness artifact paths outside the intended set are listed in the report, never deleted (TDD)
- [x] 5.3 Claude guidance compat symlinks: flat `~/.claude/engram/<file>.md` → `guidance/<file>.md`, canonical paths stated in the report (TDD)

## 6. Manifest fallback mode (D7)

- [x] 6.1 Implement manifest recording (written-path manifest inside the owned root) and manifest-driven deletion for manifest-mode harnesses (TDD; property: only recorded paths are ever deleted)

## 7. Dry-run contract (D9)

- [x] 7.1 Route all sync/link/cleanup/migration ops through op-classified, uniformly-prefixed dry-run output; dry-run creates nothing including root+marker (TDD)
- [x] 7.2 Verify #709 (unprefixed guidance-refreshed lines) is fixed by the new output path or close it as moot with evidence

## 8. Verification and docs

- [x] 8.1 `targ test` and `targ check-full` green
- [x] 8.2 End-to-end with the real binary: `go install ./cmd/engram`, run `engram update --dry-run` then `engram update` against the real home; confirm migration report, symlinked surfaces, skill discovery in a live session; then delete a source artifact on a branch and confirm removal propagates (2026-07-28: real-home migration verified — marker, canonical guidance/, compat links, 5 skills symlinked ×3 harnesses, live headless discovery 5/5; removal e2e on scratch home found the empty-dir-residue defect, fixed, re-verified: delete+prune+link-cleanup in one run)
- [x] 8.3 Write the ADR for the ownership/sync deployment model in `docs/architecture/adr.md` (ADR-0022, shipped 2026-07-28)
- [x] 8.4 Update `docs/FEATURES.md` (and any README/update docs describing deploy behavior) to the sync model (2026-07-28: c1-system-context.md R6 relationship + update-flow diagram, c2-containers.md C2→S6, c3-components.md K9, README.md engram-update section, CLAUDE.md guidance section, GLOSSARY.md guidance-file entry, guidance file header comments)
- [ ] 8.5 Close #706 with a summary; cross-reference #709 disposition; note the mechanism change on #667/#707 so their plans target the new model
