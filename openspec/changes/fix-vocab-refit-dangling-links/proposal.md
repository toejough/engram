# fix-vocab-refit-dangling-links

## Why

`engram check` reports `WARN G3 dangling: 25 authored links target no note` (GitHub issue #717). Two distinct defects produce them:

1. **Vocab retirement uses the wrong model.** The refit retire step (`retireVocabTerms`, `internal/cli/vocab_apply.go`) demotes a retired term's definition note in place and records supersession on the vocab-definition family note. There is no clean supersession story for vocab definition notes — they are term identity, not facts worth preserving. Retirement must delete the definition note outright, along with every link and tag that references it. The v9.0 refit left 10 demoted ex-definition notes in the vault and 10 supersedes entries (frontmatter + dangling body wikilinks) on family note 236.
2. **Supersession wikilinks are written in a non-resolving form.** The shared renderer `renderSupersedes` (`internal/cli/supersedes.go:129`) embeds the target filename *with* its `.md` extension, while the graph resolver keys nodes by extension-less basename (`vaultgraph.ParseBasename`). Every learn/amend supersession body link therefore dangles even though its target exists — 12 of the 25 warnings are exactly this.

(The remaining warnings — measured post-fix as 8: 142→`feedback_*` ×2, 33→`feedback_*`, 189→`vocab.X`, 301→config-path text, and 306/322/324→wrong Luhmann id/date — are hand-authored links to targets that never existed; out of scope.)

## What Changes

- **BREAKING (vault behavior):** Refit retirement deletes the retired term's definition note and its `.vec.json` sidecar instead of demoting it, and records no supersession. It removes all references to the deleted note: member `vocab/<term>` tags (as today) and any wikilinks/supersedes entries naming the deleted basename.
- One-time vault repair of the v9.0 aftermath: delete the 10 demoted ex-definition notes (+ sidecars) and strip the 10 supersedes entries and body links from family note 236.
- `renderSupersedes` (learn/amend path) writes wikilinks in canonical extension-less form; the frontmatter `supersedes:` `note:` field keeps the full filename.
- `vaultgraph.ParseWikilinks` normalizes `.md`-suffixed targets, so the 12 existing legacy supersession links resolve without a data migration.
- Net effect on the live vault: G3 dangling 25 → 8 (all survivors verified target-missing; the initial 25 → 3 estimate undercounted pre-existing genuinely-dangling links).

## Capabilities

### New Capabilities

- `vault-wikilink-resolution`: canonical wikilink form (extension-less basename), parser-side normalization of `.md`-suffixed targets, and the requirement that machine-written links emit canonical form.

### Modified Capabilities

- `vault-vocab-lifecycle`: the "Derivational refit with centroid derivation" requirement changes — retirement deletes the definition note and scrubs references to it, replacing the supersede-and-demote behavior.

## Impact

- `internal/cli/vocab_apply.go` — `retireVocabTerms` rewritten: delete note + sidecar, scrub references; `demoteDefinitionNote` and `recordRetirementSupersession` removed.
- `internal/cli/supersedes.go` — `renderSupersedes` trims `.md` from wikilink targets.
- `internal/vaultgraph/parser.go` — `ParseWikilinks` normalizes `.md`-suffixed targets.
- Live vault — one-time repair (10 note+sidecar deletions, family-note 236 cleanup).
- No change to learn/amend supersession semantics for regular notes; no sidecar schema change.
