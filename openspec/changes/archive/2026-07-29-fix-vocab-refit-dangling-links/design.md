# Design — fix-vocab-refit-dangling-links

## Context

Two defects behind `engram check`'s 25 G3 dangling warnings (issue #717):

- **Vocab retirement model.** `retireVocabTerms` (`internal/cli/vocab_apply.go`) demotes retired definition notes in place (`demoteDefinitionNote`) and records supersession on the vocab-definition family note (`recordRetirementSupersession`). Joe's ruling: there is no clean supersession for vocab definition notes — retirement must delete them and all links/tags to them. The v9.0 refit left 10 demoted ex-definition notes and 10 supersedes entries (frontmatter + body wikilinks) on note 236.
- **Link form.** The shared `renderSupersedes` (`internal/cli/supersedes.go:129`) writes `Supersedes: [[<filename-with-.md>]]`, but the graph resolver keys nodes by extension-less basename — so all 12 learn/amend supersession links in the vault dangle despite existing targets.

## Goals / Non-Goals

**Goals**

- Refit retirement deletes the definition note + sidecar and scrubs every reference (member tags, wikilinks, `supersedes:` entries naming the basename); no demote, no supersession.
- Repair the live vault's v9.0 aftermath to the deletion model.
- Learn/amend supersession writes emit canonical (extension-less) wikilinks; legacy `.md`-suffixed links resolve via parser normalization.
- `engram check` G3: 25 → 8, measured (all survivors are hand-authored links to never-existing targets; out of scope).

**Non-Goals**

- No change to learn/amend supersession semantics for regular (non-vocab-definition) notes — supersession remains the model there.
- No frontmatter `supersedes:` schema change (`note:` keeps the full filename).

## Decisions

1. **Retirement = deletion, not supersession (Joe's ruling, 2026-07-29).** Definition notes are term identity, not preserved facts; a demoted definition is dead weight that still attracts links. `retireVocabTerms` deletes the note and `.vec.json`, keeps the existing member-tag strip, and gains a reference-scrub pass: remove wikilinks and `supersedes:` entries (frontmatter entry + matching body line) whose target basename was just deleted. `demoteDefinitionNote` and `recordRetirementSupersession` are removed entirely (dead-code sweep per removal-completeness practice).
2. **Reference scrub is vault-wide, mechanical, and scoped to the deleted basenames.** Scan all notes for `[[<deleted-basename>]]` (with or without `.md`) and `supersedes:` entries naming the deleted filename; remove the line/entry, leave everything else byte-identical. Body/frontmatter rewrites go through the vault's locked write path. Situation/body text of *other* notes is untouched except for the removed reference lines — content-hash staleness only affects a note whose hashed text actually changed; verify sidecar state after the scrub in tests.
3. **One-time repair implemented as data cleanup, verified by `engram check`.** Delete the 10 demoted ex-definition notes (210, 216, 218, 220, 224, 225, 227, 228, 234, 235) + sidecars, and strip 236's 10 supersedes frontmatter entries and body lines. Done once during implementation (script or manual, evidence recorded in the task), not shipped as a subcommand — this state can't recur once retirement deletes.
4. **Fix both writer and reader for the link-form defect.** Writer: `renderSupersedes` trims `.md` (learn/amend path only — the vocab caller is deleted by decision 1). Reader: `ParseWikilinks` normalizes `.md`-suffixed targets before dedup, healing the 12 legacy links without a migration and fixing all parse consumers at once. Trim exactly a trailing `.md` — matches `ParseBasename` symmetry and Obsidian behavior.

## Risks / Trade-offs

- [Deleting a definition note another vault note genuinely cites for content] → the scrub removes only the reference line, not the citing note; definition bodies are one-line term descriptions with no unique fact content.
- [Reference scrub rewriting a body could stale a sidecar] → tests assert embed state stays clean when only non-hashed lines change; if a removed body line IS hash input, re-embed via the existing write path.
- [Existing tests assert demote/supersession behavior] → rewritten as part of TDD (RED first on the new deletion behavior).

## Migration Plan

Code change + one-time vault repair; `go install ./cmd/engram`, run repair, then `engram check` from a non-vault cwd. Measured: G3 25 → 8 (survivors all verified target-missing). Rollback: revert commit; deleted definition notes are recoverable only from backups — acceptable per ruling (they are refit-regenerable term descriptions).

## Open Questions

None.
