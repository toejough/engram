# Tasks — fix-vocab-refit-dangling-links

## 1. Retirement by deletion (vault-vocab-lifecycle)

- [x] 1.1 RED: tests for `retireVocabTerms` deletion model — retired term's definition note and `.vec.json` are deleted; member `vocab/<term>` tags stripped; no demoted note, no supersedes entry on the family note; wikilinks and `supersedes:` entries (frontmatter + body line) naming the deleted basename are removed vault-wide; untouched notes stay byte-identical and sidecars stay clean
- [x] 1.2 GREEN: rewrite `retireVocabTerms` — delete note + sidecar, keep member-tag strip, add reference-scrub pass through the locked write path
- [x] 1.3 Remove `demoteDefinitionNote` and `recordRetirementSupersession` + their tests; sweep for leftover references to both (comments, docs)

## 2. Canonical link form (vault-wikilink-resolution)

- [x] 2.1 RED: `renderSupersedes` emits `Supersedes: [[<basename-without-.md>]] — <type>: <claim>`; frontmatter `supersedes:` `note:` keeps the full filename (learn/amend path; update fixtures asserting the suffixed form)
- [x] 2.2 GREEN: trim trailing `.md` from the wikilink target in `renderSupersedes`
- [x] 2.3 RED: `ParseWikilinks` normalizes `[[x.md]]` → `x` pre-dedup ( `[[x]]` + `[[x.md]]` → one target); `UnresolvedTargets` resolves `.md`-suffixed links to existing basenames and still reports truly missing targets
- [x] 2.4 GREEN: normalize in `ParseWikilinks`
- [x] 2.5 Property test (rapid): parsing `[[name]]` and `[[name.md]]` yields identical target lists

## 3. One-time vault repair (v9.0 aftermath)

- [x] 3.1 Delete the 10 demoted ex-definition notes + sidecars (Luhmann 210, 216, 218, 220, 224, 225, 227, 228, 234, 235)
- [x] 3.2 Strip the 10 retirement `supersedes:` frontmatter entries and `Supersedes: [[...]]` body lines from family note 236; verify 236's sidecar is still valid

## 4. Verification & closure

- [x] 4.1 `targ test` and `targ check-full` pass
- [x] 4.2 `go install ./cmd/engram`; run `engram check` against the live vault from a non-vault cwd: G3 dangling 25 → 8 — measured; all 8 survivors verified to target notes that exist in no form (142→`feedback_*` ×2, 33→`feedback_*`, 189→`vocab.X`, 301→config-path text, 306/322/324→wrong Luhmann id/date). The proposal's predicted 25 → 3 undercounted pre-existing genuinely-dangling links
- [x] 4.3 Close GitHub issue #717 with the verification evidence
