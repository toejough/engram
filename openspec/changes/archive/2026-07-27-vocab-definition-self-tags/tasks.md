## 1. Confirm the Gate-A-verified groundwork (read-only)

- [x] 1.1 Re-confirm and cite in the execution log: (a) `extractNoteVocabTags` (`vocab_commands.go:941-958`, its `isVocabDefinitionNote` check at :949, called from `collectVaultStats` at :681) and `collectTriggerVaultStatsFromNames` (`vocab_trigger.go:99-140`) gate on `isVocabDefinitionNote` independent of tag content (Gate A verified — self-tags cannot change stats/trigger numbers); (b) the write-time assignment exemption at `vocab.go:214` (`loadVocabAssignmentBodyVector` returns nil,false for definitions); (c) the slug helpers `vocabDefinitionPrefix`/`vocabDefinitionSuffix` (`vocab_commands.go:313-316`) and `termFromDefinitionSlug` (`vocab_commands.go:1604` — family-slug early-return + empty-term guard)
- [x] 1.2 Re-confirm `buildTagNominations` (`query_nominations.go:275-288`) has NO definition filter on either side: nominee entry (via `TermIndex`) or seed extraction (terms from top-3 delivered notes)

## 2. Minting writes the self-tag (TDD; tests = gomega assertions over closure-built `VocabDeps`, matching the sibling tests in `vocab_commands_test.go` — no imptest in `internal/cli`)

- [x] 2.1 RED: unit test (`t.Parallel()` in parent and every subtest; exported test funcs before private decls) — `mintDefinitionNote` for term `test-term` writes `tags:` exactly `[vocab, vocab/test-term]` in that order; run `targ test`, expect fail against current bare-only minting
- [x] 2.2 GREEN: derive the term via `termFromDefinitionSlug` inside `mintDefinitionNote` (`vocab_commands.go:1178`) and append the self-tag after the bare marker; update the `:720` comment contract AND the family-note Object prose at `vocab_commands.go:972-974` to the new convention text (definitions carry bare marker + self-tag; members carry `vocab/<term>`; family note carries `vocab_version`); `targ test` green
- [x] 2.3 RED: mint-side family test — `ensureVocabFamilyNote`'s minted note carries `vocab` and no `vocab/` entry (the helper's family-slug early-return makes this GREEN immediately if 2.2 routed through it; a failure means the routing is wrong)
- [x] 2.4 RED→GREEN: refit regression test — `retagAllNotesTwoPass` over a vault with self-tagged definitions leaves every definition note byte-identical (design D5)

## 3. Nomination filter, both sides (TDD) — EXECUTOR GROUP 3 (NOT MY RESPONSIBILITY)

- [x] 3.1 RED (nominee side): a self-tagged definition sharing a term with a cluster's top-3 notes is NOT nominated into `candidate_l2s`; expect fail (no filter yet)
- [x] 3.2 RED (seed side): a self-tagged definition placed IN a cluster's top-3 contributes no seed terms — the cluster's nominations equal the bare-only baseline; expect fail
- [x] 3.3 GREEN: add `isVocabDefinitionNote` filtering at both points in `buildTagNominations` (`query_nominations.go`): skip definitions when extracting seed terms from top-3 content, and skip definitions when adding nominees; `targ test` green

## 4. Backfill subcommand `engram vocab tag-definitions` (TDD)

- [x] 4.1 RED: unit tests (gomega + closure `VocabDeps`) — (a) backfill rewrites bare-only per-term definitions to `tags: [vocab, vocab/<term>]` (order pinned) and reports "added" per note; (b) second run is a no-op reporting "already present"; (c) non-definition notes and malformed slugs are untouched (malformed → reported, skipped), and the FAMILY note (slug `vocab-definition`, empty term — design D7) is skipped and reported as the family note, staying bare-only; (d) no sidecar is touched and post-backfill embed state is clean — the test asserts the command performs NO re-embed (vocab tags are not content-hash inputs; design D3 corrected model) (no-re-embed guard added in the Gate B fix pass)
- [x] 4.2 GREEN: implement following the `assignVocabToNote` precedent (locked `WriteFile` tag rewrite, no sidecar touch) under `acquireOptionalLock`; wire into the existing `targ.Group("vocab", ...)` in `internal/cli/targets.go` reusing `newVocabDeps` (no new deps struct); the subcommand MUST support the standard `--vault` flag (required by 5.1); `targ test` green
- [x] 4.3 `targ check-full` clean (all linters; no suppressions) — orchestrator-verified PASS:8 with only check-uncommitted failing (expected mid-cycle) — ORCHESTRATOR-RUN AFTER BOTH EXECUTORS LAND

## 5. Real-binary verification + live backfill

- [x] 5.1 `go install ./cmd/engram`; copy the live vault (`cp -R ~/.local/share/engram/vault /tmp/vault-selftag-probe`); from a non-data-dir cwd run `engram vocab tag-definitions --vault /tmp/vault-selftag-probe` — CONFIRM ISOLATION BY OUTPUT PATHS: every reported note path must start with `/tmp/vault-selftag-probe/` (env-var redirects are not honored by all engram commands; `--vault` is the config-level isolation here). Verify: ~29 "added" lines + 1 family-note skip line, idempotent second run ("already present" throughout), `engram vocab stats --vault /tmp/vault-selftag-probe` member counts and untagged rate identical to the live vault's pre-run stats
- [x] 5.2 Record live pre-state: `engram vocab stats` output; `engram count --group-by tags --filter tags=vocab/cli-verification` and `--filter tags=vocab/retrieval-design` totals. Then run `engram vocab tag-definitions` against the live vault
- [x] 5.3 Post-verify live: `engram vocab stats` member counts/untagged identical to 5.2 pre-state; the two spot `engram count` totals each +1 (the definition, expected divergence per design); then the ask's success criterion in Obsidian on 2-3 definitions (e.g. `vocab-cli-verification-definition`): (a) frontmatter shows `[vocab, vocab/<term>]`, (b) clicking the `vocab/<term>` tag in the tag pane lists the definition alongside its members
- [x] 5.4 Amend vault note `236.2026-07-10.vocab-definition` to state the new convention (per-term definitions carry the bare marker plus their own `vocab/<term>` self-tag; the family note stays bare-only; members carry `vocab/<term>`) — design D6

## 6. Docs + close

- [x] 6.1 Apply the doc-surface dispositions enumerated in design.md ("Doc-surface enumeration"): README.md:96 bootstrap doc + new `vocab tag-definitions` Binary-commands entry; GLOSSARY "vocab definition note" entry (pinned AFTER text — delete "a definition note never carries its own term tag"); ADR-0011 dated representation-update annotation
- [x] 6.2 Comment sweep + re-grep: `grep -rn 'bare "vocab"\|bare-vocab\|never vocab/' internal/ docs/ README.md` — every remaining hit must be either updated (asserts bare-only as current) or an N/A surface (historical entries, the discriminator-role descriptions that stay true, this change's artifacts); paste the final hit-list disposition into the execution log
- [x] 6.3 `targ check-full` + full test pass; `openspec validate vocab-definition-self-tags --type change --strict` passes; commits happen only at the workflow's designated commit points (with the gate over messages), never mid-task by executors
