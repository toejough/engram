## 1. Wire ingest + prune into apply (RED then GREEN)

- [x] 1.1 Read `internal/cli/luhmann_reparent_apply.go` in full, plus `internal/cli/ingest.go`'s
      `newIngestDeps`/`IngestDeps`/`IngestArgs` and `internal/cli/prune.go`'s
      `newPruneDeps`/`PruneDeps`/`PruneArgs`, to confirm exact signatures before wiring (this
      prompt paraphrases; the code is authoritative).
- [x] 1.2 Write failing tests: a successful apply run (≥1 note renamed) results in the renamed
      note's new path present in the chunk manifest AND the old path's manifest entry detached,
      without a separate `RunIngest`/`RunPrune` call from the test — i.e. apply itself must do
      both. Cover: apply with zero renames (all `top`) triggers neither ingest nor prune (nothing
      changed, nothing to reconcile). Cover `--dry-run` triggers neither (per design.md Non-Goals
      — dry-run previews, never mutates the chunk index).
- [x] 1.3 Implement: after `RenameAndRewriteReferences` succeeds, call `RunIngest` (composed via
      `newIngestDeps`-equivalent for the reparent context) then `RunPrune` (plain mode, composed
      via `newPruneDeps`-equivalent), per design.md Decision 1's ordering. Run tests — GREEN.
- [x] 1.4 Write failing tests for design.md Decision 2 (no rollback on pipeline failure): rename
      succeeds, `RunIngest` or `RunPrune` fails → apply reports which stage failed and that the
      vault rename is intact, with the manual-fallback message; note files remain renamed (not
      reverted). Implement, GREEN.

## 2. Next-command output (RED then GREEN)

- [x] 2.1 Write failing tests: derive-phase output includes a literal next-command line naming
      `engram update --reparent-luhmann --answers <path>` (with a fillable/placeholder path —
      decide exact wording during implementation) alongside the existing candidate payload.
- [x] 2.2 Implement. GREEN.
- [x] 2.3 Write failing tests for the "further candidates remain?" check (design.md Decision 3):
      after a successful pipeline-complete apply, a follow-up read-only candidate check against
      the vault's now-current state reports either "N further candidates — run `engram update
      --reparent-luhmann` again" or "no further candidates — vault fully evaluated". Confirm this
      check does NOT print a full candidate payload (that only happens on an actual derive run).
- [x] 2.4 Implement, reusing the existing candidate-derivation function in-process (not a shelled
      re-invocation). GREEN.

## 3. Remove now-obsolete standalone-hint framing

- [x] 3.1 Confirm no dangling references remain to a standalone "run `engram prune` after apply"
      hint from the superseded `reparent-luhmann-prune-hint` proposal (that change was never
      implemented — this is a sanity check, not a revert).

## 4. Verification

- [x] 4.1 Run `targ check-full`.
- [x] 4.2 Re-run the `update-flags-flat-vault-luhmann-reeval` task-7.3-style fixture-vault
      verification end to end: derive → judge → apply. Confirm the new path is indexed, the old
      path's manifest entry is gone, WITHOUT running `engram prune`/`engram ingest --auto`
      separately. Confirm the "further candidates?" message is accurate for the fixture (should
      report none remain, or correctly identify any that do).
- [x] 4.3 Verify the pipeline-failure path (design.md Decision 2 / task 1.4) against a real
      induced failure (e.g. a read-only chunks dir) — confirm the vault rename is genuinely
      intact and the error message correctly names the manual fallback.

## 5. Docs + close-out

- [ ] 5.1 Update `docs/GLOSSARY.md`'s `--reparent-luhmann` entry: replace the "known gap (#724)"
      language with a description of the automatic mechanical pipeline (ingest + prune folded
      into apply) and the next-command/loop-until-done output contract.
- [ ] 5.2 Sync `openspec/specs/update-reparent-luhmann-batch/spec.md` with this change's delta
      (MODIFIED requirements + the REMOVED "manual prune step" requirement), then archive.
- [ ] 5.3 Close GitHub issue #724, referencing this change.
