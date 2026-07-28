# Design: Derivational Vocab Refit

## Context

Today's refit (`RunVocabRefit`, `internal/cli/vocab_commands.go`) applies an LLM-authored YAML plan (renames/removals/uncapped `new_terms`), then runs `retagAllNotesTwoPass` to derive centroids from member means. Three triggers (`internal/cli/vocab_trigger.go`) set `refit_pending`: growth (≥40 notes AND ≥14 days), untagged rate >8%, hub >25%. The untagged trigger creates a mint ratchet: notes below the 0.35 assignment floor can only be covered by adding terms, the trigger re-arms as new odd notes arrive, and nothing punishes fragmentation — observed drift to ~150 terms / ~400 notes on one vault. Since assignment is cosine-only and centroids are member means, vocab is already confined to embedding geometry; this design makes that explicit and convergent. `internal/cluster` already provides k-means with silhouette-based auto-K (used by recall clustering).

## Goals / Non-Goals

**Goals:**
- Term count pinned to the vault's silhouette-optimal cluster count; monotone additive drift structurally impossible.
- Term-name stability across derivations (users and notes keep their vocabulary).
- LLM involvement reduced to naming genuinely new clusters; no plan authoring.
- Preserve: definition-note conventions (bare `vocab` marker + self-tag), write-time top-3 assignment at floor 0.35, `vocab.centroids.json` as the downstream contract (consumed by recall).

**Non-Goals:**
- Changing recall behavior (companion change `recall-centroid-sampling`).
- Changing `vocab bootstrap`: it is retained unchanged, solely for seeding a pre-derivation vault; its YAML-seeded terms carry `origin: derived` and are absorbed (matched or retired) by the first derivation.
- Representing geometrically scattered concepts via clustering — `vocab propose` remains the escape hatch for those.
- Cross-machine vocab sync.

## Decisions

1. **Derivation = silhouette-auto-K k-means over non-definition note vectors**, reusing `internal/cluster`. Alternative considered: keeping editorial refit with a term-count cap and merge heuristics (centroid-cosine + merged-cohesion ≥ 0.35 gates). Rejected: caps and gates fight the ratchet's symptoms while leaving the LLM free to over/under-apply merges; derivation removes the authoring surface entirely and is convergent by construction.
2. **Name matching by greedy centroid cosine, threshold ~0.80 (tune during implementation).** Each new cluster matches the highest-similarity unclaimed existing term above threshold; matched clusters keep name + definition note. Alternative: Hungarian optimal assignment — unnecessary at this scale (tens of terms); greedy is auditable and stable.
3. **Unmatched clusters → LLM names them** (mint definition note with `[vocab, vocab/<term>]`, per existing convention). Unmatched existing terms → retired: definition note removed via the existing supersession/amend path, `vocab/<term>` tags stripped during the post-derivation re-tag pass. Alternative: keep orphaned terms as dormant. Rejected: dormant terms re-attract members at assignment time and defeat convergence.
4. **`proposed` provenance shield.** `vocab.centroids.json` entries gain `origin: derived|proposed` — this is genuinely new plumbing, not an additive tweak: no provenance discrimination exists today (`vocabCentroidEntry` carries only `Vector`/`MemberCount`, and `RunVocabPropose` mints identically to refit), so the field must be written at propose time AND read at retirement time, with pre-existing terms defaulting to `derived` on first derivation. `vocab propose`-minted terms are never retired by derivation (they may be geometrically invisible by design); they participate in assignment normally. Their existence slightly perturbs top-3 assignment but not derivation input.
5. **Trigger collapse to growth-only** (≥40 notes AND ≥14 days since `last_refit`). Untagged-rate and hub stats remain in `vocab stats` output as diagnostics with no verdict force. Rationale: under derivation, untagged notes are "genuinely new/sparse topic" signals that the next scheduled derivation absorbs; hubs are bounded by silhouette (a true mega-cluster would split when splitting improves silhouette).
6. **Learn skill Step 1.5 shrinks** to: `engram vocab stats` → if REFIT_PENDING, run `engram vocab refit`; the binary emits structured naming requests (one per new cluster, with centroid-nearest exemplars) as output; the agent supplies all term names in a single response, which the binary consumes to mint the definitions; report loudly. Skill edit follows `superpowers:writing-skills` TDD.
7. **CLI surface:** `vocab refit` keeps its name; `--emit-request`/`--plan` are removed (BREAKING). A `--dry-run` prints the derivation diff (matched/new/retired, K, silhouette score) without writing.

## Risks / Trade-offs

- [Silhouette-K instability between runs (small vaults, near-tie K)] → name-matching absorbs relabeling; add hysteresis: prefer previous K when silhouette scores are within a small epsilon.
- [Mass renaming on first derivation over a drifted vault (e.g. 150→~20 terms)] → `--dry-run` diff first; retirement uses the existing locked write path; definition notes are superseded not orphaned.
- [Cluster naming quality depends on the LLM seeing good exemplars] → naming request includes each new cluster's centroid-nearest member notes (title + snippet), not random members.
- [Scattered concepts lose representation if users forget `propose` exists] → `vocab stats` reports untagged rate as a diagnostic prompt toward `propose`, without forcing refit.
- [Downstream recall reads centroids mid-derivation] → derivation writes `vocab.centroids.json` atomically via the existing vault-concurrency-safe write path.

## Migration Plan

1. Ship binary with derivational refit; the plan flow (`--emit-request`/`--plan`) is gone — migration is: run `engram vocab refit` (optionally `--dry-run` first), with LLM involvement reduced to answering naming requests; first run on an existing vault is just a normal derivation (`--dry-run` recommended and surfaced in the skill's first-run guidance).
2. Deploy updated learn skill via `engram update`.
3. Rollback: previous binary + skill restore the plan-based flow; `vocab.centroids.json` remains readable (new fields are additive).

## Doc-surface disposition (enumeration grep, 2026-07-28)

Grep over `refit|emit-request|two-pass|refit_pending|new_terms|untagged` across docs/, agent-instructions/, openspec/specs/, CLAUDE.md, and repo-root markdown (README.md — initially omitted; caught by Gate A docs review):

- `README.md` → **update (line ~100)**: `engram vocab refit` command description states the old LLM-judged merge/split/rename plan flow; rewrite to derivation flow (clustering, centroid matching, naming requests).

- `docs/architecture/c1-system-context.md` → **update**: trigger description (lines ~209-212) lists three independent triggers; becomes growth-only with untagged/hub as diagnostics.
- `docs/architecture/c2-containers.md` → **update**: C2 row + trigger flowchart (~240-250) name three triggers and plan-based refit; rewrite trigger branch + refit description.
- `docs/architecture/c3-components.md` → **keep**: only mentions untagged-counter QA exclusion in stats, unchanged by this change.
- `docs/architecture/memory-invariants.md` → **keep**: "untagged" hit is an unrelated grep-robustness anecdote.
- `docs/architecture/adr.md` → **update**: add new ADR entry (derivational refit) referencing ADR-0011; do not rewrite history.
- `docs/ROADMAP.md` → **keep**: "rubric-refit" hits are the route skill's rubric, not vocab refit; stale-tags note (~185) stays valid.
- `docs/GLOSSARY.md` → **rewrite (section)**: `refit_pending`/`refit_reason`/`last_refit` entries and any refit-plan glossary entries; align with growth-only trigger + derivation + `origin` field.
- `agent-instructions/skills/learn/SKILL.md` → **rewrite (Step 1.5)**: plan flow replaced by run-refit-and-answer-naming (writing-skills TDD).
- `openspec/specs/vault-vocab-lifecycle/spec.md` → **update**: via this change's delta at archive/sync.
- `openspec/specs/vault-concurrency-safe-writes/spec.md` → **keep**: `vocab bootstrap/refit` lock requirement holds verbatim for the derivational flow.
- `docs/superpowers/plans/2026-07-19-700-internal-purity.md` → **N/A**: historical plan document; not maintained as current doc.
- `CLAUDE.md` → **keep**: no refit-flow specifics present.

## Open Questions

- Exact name-match threshold and silhouette hysteresis epsilon — settle empirically during implementation on the two real vaults (574-note and 400-note states are ideal test corpora).
- Whether K should be bounded below (e.g. min 5) for tiny vaults where silhouette degenerates.
