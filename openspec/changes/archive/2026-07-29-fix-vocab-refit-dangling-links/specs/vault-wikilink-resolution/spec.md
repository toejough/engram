# vault-wikilink-resolution

The vault's wikilink graph resolves authored links to notes by extension-less basename. This capability closes the resolution gap where machine-written supersession links (learn/amend path) carried a `.md` suffix and therefore never resolved (part of GitHub issue #717): machine writers emit canonical form, and the parser tolerates the legacy suffixed form already present in the vault.

## ADDED Requirements

### Requirement: Canonical wikilink form is the extension-less basename

Machine-written wikilinks SHALL target a note by its extension-less basename — the graph's canonical node key (`vaultgraph.ParseBasename`). Specifically, `renderSupersedes` MUST strip a trailing `.md` from each entry's note filename when rendering the `Supersedes: [[...]]` body line. The frontmatter `supersedes:` list's `note:` field is a file reference, not a graph edge, and SHALL continue to carry the full filename with extension.

#### Scenario: Supersession body link renders canonical form

- **WHEN** a supersession is recorded via `engram learn`/`amend` for target note `107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior.md`
- **THEN** the written body line is `Supersedes: [[107.2026-06-27.validate-lazy-retrieval-by-measuring-fetch-behavior]] — <type>: <claim>` (no `.md` inside the wikilink), and the frontmatter `supersedes:` entry's `note:` field remains the full filename with `.md`

### Requirement: Parser normalizes .md-suffixed wikilink targets

`vaultgraph.ParseWikilinks` SHALL normalize a wikilink target ending in `.md` to its extension-less basename, so legacy `.md`-suffixed links (all pre-fix supersession links) resolve against basename-keyed nodes without a vault migration. This matches Obsidian's resolution of `[[note.md]]`. Targets not ending in `.md` are unchanged, and normalization MUST apply before deduplication so `[[x]]` and `[[x.md]]` in one note yield a single target.

#### Scenario: Legacy suffixed link resolves

- **WHEN** a note body contains a `Supersedes: [[<basename>.md]]` line and a note with that basename exists in the vault
- **THEN** `vaultgraph.UnresolvedTargets` does not report the link, and `engram check` does not count it as G3 dangling

#### Scenario: Suffixed and bare forms dedupe to one edge

- **WHEN** a note body links the same target once as `[[x]]` and once as `[[x.md]]`
- **THEN** `ParseWikilinks` returns a single target `x`

#### Scenario: Genuinely dangling links still warn

- **WHEN** a note links a target that exists in no form (e.g. `[[vocab.X]]` or a deleted note's name, with or without `.md`)
- **THEN** `engram check` still reports it under G3 dangling
