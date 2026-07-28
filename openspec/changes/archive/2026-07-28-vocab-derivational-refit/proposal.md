# Proposal: Derivational Vocab Refit

## Why

The current vocab refit loop is structurally additive-only: the untagged-rate trigger (>8%) fires whenever new notes fail the 0.35 cosine floor against existing term vectors, and the only move an LLM-authored refit plan can make to satisfy it is to mint new terms (`new_terms` is uncapped; no trigger punishes fragmentation). Observed result: one vault drifted to ~150 terms over ~400 notes (~1 term per 2.7 notes), degrading vocab toward a useless 1:1 mapping. Meanwhile the system is already embedding-geometry-bound — assignment is cosine-only and post-refit centroids are member means — so vocab is a badly-converging, hand-ratcheted approximation of clustering. Deriving vocab directly from clustering makes it honest, convergent, and structurally incapable of the ratchet.

## What Changes

- **Refit becomes re-derivation.** `engram vocab refit` re-clusters all non-definition note vectors with the existing k-means + silhouette-auto-K machinery (`internal/cluster`). The resulting clusters ARE the vocab structure; term count is pinned to the data's silhouette-optimal K, not to LLM plan authoring.
- **Name stability by centroid matching.** New clusters are matched to existing terms by centroid cosine similarity (greedy best-match above a threshold); matched clusters keep their term name and definition note. Unmatched new clusters are handed to the LLM for naming only (mint definition note); unmatched old terms are retired (definition note superseded/removed, tags stripped on next re-tag pass).
- **Trigger set collapses.** The untagged-rate and hub-concentration mint-pressure triggers are removed as refit drivers; the sole trigger is growth since last derivation (note count + elapsed days). Untagged-rate and hub stats remain reported in `vocab stats` as diagnostics.
- **LLM-authored refit plans are removed.** `vocab refit --emit-request`/`--plan` YAML flow is replaced by the derivation flow; the learn skill's Step 1.5 shrinks to "run `engram vocab refit` when REFIT_PENDING" with LLM involvement only for naming new clusters. **BREAKING** for the learn skill and the refit CLI surface.
- **`vocab bootstrap` is retained unchanged** as pre-derivation vault seeding; its seeded terms are ordinary `derived`-origin terms that the first derivation matches or retires.
- **`vocab propose` is retained** as the sole judgment-based escape hatch for geometrically scattered concepts the clustering cannot see. Proposed terms are marked as such and excluded from derivation-driven retirement.
- Write-time top-3 cosine assignment, definition notes, `vocab.centroids.json`, and self-tag conventions are unchanged; centroids are now written by derivation (mean of cluster members) rather than two-pass refit.

### Beyond the literal ask

The ask was "vocab must not be purely additive; derive it from the global centroids." Three decisions here go further, as necessary consequences of derivation rather than direct requests — user-visible in the explore session that produced this change and re-flagged here for explicit sign-off:

- **CLI plan-flow removal** (`--emit-request`/`--plan`): derivation cannot coexist with an externally authored term list; keeping the flow would preserve the additive path.
- **Term retirement**: a derived term whose cluster disappears is actively superseded, not left dormant — dormant terms re-attract members at assignment time and defeat convergence.
- **Trigger demotion** (untagged/hub → diagnostics): under derivation these signals no longer indicate "mint terms"; leaving them as refit-forcers would re-arm the ratchet's cadence without its mechanism.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `vault-vocab-lifecycle`: the "Two-pass refit with centroid derivation" requirement is replaced by derivational re-clustering with name matching; the "Autonomous refit triggers" requirement drops untagged/hub as refit triggers (growth-only); definition-note and write-time-assignment requirements unchanged.

## Impact

- `internal/cli/vocab_commands.go` (`RunVocabRefit`, `applyRefitNewTerms`, plan types), `internal/cli/vocab_trigger.go` (trigger evaluation), `internal/cluster` (reused, possibly parameter plumbing), `vocab.centroids.json` schema (add derivation metadata; term provenance `derived|proposed`).
- `agent-instructions/skills/learn/SKILL.md` Step 1.5 rewrite (requires `superpowers:writing-skills` TDD flow).
- Interfaces consumed by the companion change `recall-centroid-sampling` (centroid file is the contract between them); the two changes are separable — this one does not alter recall.
