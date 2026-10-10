# vault-wikilink-resolution

## Purpose

The canonical form that `engram learn` and `engram amend` write for supersession links is in `docs/changes/792_documentation-framework-adoption/792a_product-spec.md#792a4-added-note-revision` (792a4d). The vault scan is in `docs/changes/792_documentation-framework-adoption/792b_technical-spec.md#792b13-added-wikilink-graph`; the parser's normalization and all three scenarios wait for vault-health (792c7h1, 792d75). The vault's wikilink graph resolves authored links to notes by extension-less basename. This capability closes the resolution gap where machine-written supersession links (learn/amend path) carried a `.md` suffix and therefore never resolved (part of GitHub issue #717): machine writers emit canonical form, and the parser tolerates the legacy suffixed form already present in the vault.

## Requirements

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
