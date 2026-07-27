## 1. Verify current member-semantics filters (read-only groundwork)

- [ ] 1.1 Trace `vocab stats` member counts and `evaluateVocabTriggers` untagged/hub math in `internal/cli/vocab_trigger.go` / `vocab_commands.go`: confirm (or find missing) the `isVocabDefinitionNote` exclusion at each site; record the actual filter locations in the task notes — the D2 tests in group 3 pin whichever sites exist
- [ ] 1.2 Confirm `buildTagNominations` (`internal/cli/query_nominations.go`) has no definition filter today, and identify the note-loading point where one belongs

## 2. Minting writes the self-tag (TDD)

- [ ] 2.1 RED: unit test (imptest + gomega, `t.Parallel()`) — `mintDefinitionNote` for term `test-term` writes `tags: [vocab, vocab/test-term]`; run `targ test`, expect fail against current bare-only minting
- [ ] 2.2 GREEN: add the self-tag in `mintDefinitionNote` (`internal/cli/vocab_commands.go`); update the ":720" comment contract ("bare `vocab` plus the term's own `vocab/<term>` self-tag"); `targ test` green
- [ ] 2.3 RED→GREEN: regression test — a refit (`retagAllNotesTwoPass`) over a vault with self-tagged definitions leaves every definition note byte-identical (D5)

## 3. Semantics preservation (TDD)

- [ ] 3.1 RED: nomination test — a self-tagged definition sharing a term with a cluster's top-3 notes is NOT nominated into `candidate_l2s`; expect fail (no filter yet)
- [ ] 3.2 GREEN: add the `isVocabDefinitionNote` filter at the nomination pool build (`internal/cli/query_nominations.go`); `targ test` green
- [ ] 3.3 RED→GREEN as needed: trigger-math parity test — identical `evaluateVocabTriggers` member counts, untagged counts, and outcomes for the same vault with and without definition self-tags (add missing filters only where 1.1 found gaps; if already filtered, the test lands GREEN and pins it)

## 4. Backfill subcommand `engram vocab tag-definitions` (TDD)

- [ ] 4.1 RED: unit tests — (a) backfill adds `vocab/<term>` derived from the `vocab-<term>-definition` slug to bare-only definitions and reports "added" per note; (b) second run is a no-op reporting "already present"; (c) non-definition notes and malformed slugs are untouched (malformed slug → reported, skipped, non-zero detail in output), and the FAMILY note (slug `vocab-definition`, empty term — design D7) is explicitly skipped and reported as the family note, staying bare-only; (d) each touched note's sidecar is refreshed (embed state clean, no `StateStale`)
- [ ] 4.2 GREEN: implement the subcommand through the locked vault write path (existing amend-style write + sidecar refresh machinery); wire in `internal/cli/targets.go` per the established deps pattern; `targ test` green
- [ ] 4.3 `targ check-full` clean (all linters; no suppressions)

## 5. Real-binary verification + live backfill

- [ ] 5.1 `go install ./cmd/engram`; from a non-data-dir cwd run `engram vocab tag-definitions` against a THROWAWAY COPY of the vault first (config-level isolation — env-var redirects are not honored by all commands; use the `--vault` flag and verify the output paths name the copy before running): confirm per-note report, idempotent second run, `engram vocab stats` member counts and untagged rate identical to pre-run
- [ ] 5.2 Record pre-state on the live vault (`engram vocab stats` output; `engram count --group-by tags` for two spot terms), then run `engram vocab tag-definitions` on the live vault
- [ ] 5.3 Post-verify live: `vocab stats` member counts/untagged unchanged; spot `count --group-by tags` shows +1 for each spot term (the definition, expected); open one definition in Obsidian and confirm the term tag connects it to its cluster
- [ ] 5.4 Amend vault note `236.2026-07-10.vocab-definition` to state the new convention (definitions carry the bare marker plus their own `vocab/<term>` self-tag; members carry `vocab/<term>`) — D6

## 6. Docs + close

- [ ] 6.1 Update `openspec/specs/vault-vocab-lifecycle/spec.md` is NOT hand-edited — the delta lands via `/opsx:archive` (or `/opsx:sync`); verify `openspec validate vocab-definition-self-tags --type change --strict` passes
- [ ] 6.2 Apply the doc-surface dispositions enumerated in design.md ("Doc-surface enumeration"): README.md:96 bootstrap doc + new `vocab tag-definitions` command entry; GLOSSARY "vocab definition note" entry; ADR-0011 dated representation-update annotation — then re-run the enumeration grep and confirm no stale bare-only claims remain outside N/A surfaces
- [ ] 6.3 `targ check-full` + full test pass; commits happen only at the workflow's designated commit points (with the gate over messages), never mid-task by executors
