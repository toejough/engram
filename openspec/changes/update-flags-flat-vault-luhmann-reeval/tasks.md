## 1. Notice detector (RED then GREEN)

- [x] 1.1 Write a failing test in `internal/cli/update_test.go` for a new detector function
      (e.g. `vaultHasOnlyTopLevelNotes(vaultPath string, fileSystem update.Filesystem) bool`):
      cases — all notes depth 1 → true; ≥1 note depth > 1 → false; empty vault → false;
      unreadable vault dir → false (mirrors `oldVocabFilesPresent`'s error handling).
- [x] 1.2 Check whether `internal/cli` already has a shared "list all vault note IDs" helper
      before writing a new ReadDir+filter loop; reuse if present.
- [x] 1.3 Implement the detector using `idAndDateFromNoteFilename` +
      `internal/luhmann.ParseID` (skip entries where `idAndDateFromNoteFilename` returns
      `ok=false`). Run the test — GREEN.

## 2. Notice wiring

- [x] 2.1 Add `VaultHasOnlyTopLevelNotes bool` to `update.Report` in `internal/update/update.go`,
      doc comment matching `VaultHasOldVocabFiles`'s style.
- [x] 2.2 In `internal/cli/update.go`, populate `report.VaultHasOnlyTopLevelNotes` alongside the
      existing `VaultHasOldVocabFiles`/`VaultHasUntaggedVocabDefinitions` assignments.
- [x] 2.3 Add a notice-writer function following the `writeVocabSelfTagHint` pattern, naming
      `engram update --reparent-luhmann` as the remedy, wired into `writeUpdateReport`.
- [x] 2.4 Add CLI-level tests: notice printed when true, absent when false, no file mutation
      in either case.

## 3. Rename + backlink-rewrite helper (new capability — no existing code does this)

- [x] 3.1 Design decision resolved: `--dry-run` requires `--answers` and shows the intended
      renames (apply-preview only); `--dry-run` without `--answers` is a usage error (derive
      never writes regardless of the flag).
- [x] 3.2 Write failing tests for a new rename-and-rewrite helper (new file, e.g.
      `internal/vaultgraph/rewrite.go` or `internal/cli/luhmann_reparent.go` — decide package
      placement based on whether it needs `vaultgraph`-internal access or CLI-level FS access):
      given an old→new basename map, it renames the note file + `.vec.json` sidecar, updates
      the note's own frontmatter `id:`, and rewrites every `[[old-basename]]` wikilink and
      `Supersedes: [[old-basename]]` / `supersedes:.note:` reference elsewhere in the vault to
      the new basename. Cover: single reference, multiple references across notes, a note that
      is both renamed AND references another renamed note in the same run (cascading rewrite
      must use the full rename map, not be interleaved per-note per design.md Decision 4).
- [x] 3.3 Implement the helper. Run tests — GREEN.

## 4. Derive phase

- [ ] 4.1 Write failing tests for candidate derivation: for each top-level note, top-K
      (K=3, revisit if needed) nearest neighbors by embedding-sidecar cosine similarity above a
      similarity floor (pick and document a starting floor value; make it easy to tune).
      Cover: no candidates above floor → empty payload; candidates found → payload includes
      note IDs, similarity scores, and enough content excerpt for an agent to judge relation.
- [ ] 4.2 Implement derive. Include a `fingerprint` of vault state (mirror `vocab refit`'s
      fingerprint mechanism/format if reusable).
- [ ] 4.3 Wire `--reparent-luhmann` (no `--answers`) to run derive-only and print the payload,
      writing nothing.

## 5. Apply phase

- [ ] 5.1 Write failing tests for answer parsing: `{"reparenting": [...], "fingerprint": "..."}`
      → for each entry, position `continuation`/`sibling` computes a new ID via the existing
      `nextChild`/`nextSibling` (from `internal/cli/luhmann.go`, unmodified) against the
      vault's current ID set, applied in ascending original-ID order (design.md Decision 3);
      position `top` is a no-op (no rename).
- [ ] 5.2 Wire the apply phase to: validate the fingerprint against current vault state (stale
      → reject with a clear error, no writes — mirrors `vocab refit`'s stale-names handling);
      compute the full rename map; call the Section 3 helper to rename + rewrite backlinks.
- [ ] 5.3 Wire `--dry-run` (requires `--answers`; reject with a usage error otherwise) to print
      the same rename/rewrite map the apply phase would produce, without calling the writing
      path.

## 6. Learn skill batch-answer mode

- [ ] 6.1 Using `superpowers:writing-skills` TDD, add a batch-re-eval mode to
      `agent-instructions/skills/learn/SKILL.md` (or a small new skill, if a distinct trigger
      makes more sense — decide during implementation) that, given a derive-phase candidate
      payload, applies the same disposition test as #701's per-capture Step 2 to each candidate
      pair and writes the answers file.
- [ ] 6.2 RED: baseline an agent given only the current (pre-this-change) skill text and a
      candidate payload — confirm it has no instructed way to produce an answers file. GREEN:
      re-run with the new batch-answer instructions present.

## 7. Verification

- [ ] 7.1 Run `targ check-full`.
- [ ] 7.2 Build a scratch fixture vault (several top-level notes, some genuinely related, some
      not, with existing wikilinks between a subset) and run the full derive → answer → dry-run
      → apply cycle end to end; confirm renamed files, updated frontmatter IDs, and rewritten
      wikilinks all land correctly, and that unrelated notes/links are untouched.
- [ ] 7.3 Resolve design.md Open Question 2 empirically: run `engram ingest --auto` after an
      apply and confirm the chunk index self-heals (old path's chunks age out via existing
      prune, new path indexes cleanly) — if it does not, file a follow-up issue rather than
      silently expanding this change's scope.
- [x] 7.4 Resolve design.md Open Question 3 with Joe: does `--reparent-luhmann` need an
      uncommitted-git-changes guard rail? Resolved — no, not needed.

## 8. Docs + close-out

- [ ] 8.1 Update `docs/GLOSSARY.md`'s `engram update` list to include both the notice and
      `--reparent-luhmann` (derive/answer/apply/dry-run flags), matching existing entries'
      level of detail.
- [ ] 8.2 Sync `openspec/specs/` with both `update-flat-vault-luhmann-notice` and
      `update-reparent-luhmann-batch` capabilities via `openspec-sync-specs`, then archive this
      change.
