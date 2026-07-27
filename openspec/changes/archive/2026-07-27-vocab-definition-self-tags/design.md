## Context

Vocab definition notes are recognized by the bare `vocab` marker tag (`vocabDefinitionTag`, `internal/cli/vocab.go:107`; `isVocabDefinitionNote` at :182) and are deliberately minted "bare `vocab` only — never `vocab/<term>`" (`internal/cli/vocab_commands.go:720`). Member notes get `vocab/<term>` tags from write-time assignment, which exempts definitions (`vocab.go:214`). Consequence in Obsidian: a term's tag node connects all members but not the term's own definition note. The vault's convention record (note `236.2026-07-10.vocab-definition`) documents the current bare-only shape.

Current member-semantics surfaces that read `vocab/<term>` tags: refit member scans (already exclude definitions per the existing spec), `vocab stats` member counts and trigger math (`vocab_trigger.go` — untagged rate today does not count definitions, implying the marker filter is already applied there; verify at implementation), tag nomination (`query_nominations.go` — no definition filter exists today because definitions carry no term tags), and `engram count --group-by tags`.

## Goals / Non-Goals

**Goals:**
- Every definition note carries `vocab/<term>` for its own term, connecting it to its member cluster in Obsidian's tag graph/pane.
- Engram's observable behavior is otherwise unchanged: member counts, refit triggers, nomination pools, and centroid math produce the same results before and after.
- Future definitions (bootstrap, refit-minted) are born with the self-tag; existing ones are backfilled by one explicit idempotent command.

**Non-Goals:**
- No recall-payload behavior change (definitions joining nomination pools was considered and excluded — it would alter recall results and needs its own eval; the self-tag is a display affordance).
- No vocab_version bump (this is a tagging-convention change, not a refit; centroids and membership are untouched).
- No wikilink-based linking of definitions to members (the vault's structural linking deliberately runs through tags, not hand-authored wikilinks).
- No `engram update` involvement (mutating ops stay explicit user commands).

## Decisions

- **D1 — the bare marker stays the discriminator.** `isVocabDefinitionNote` continues to key on the bare `vocab` tag; the self-tag is additive. Alternative (key on slug `vocab-<term>-definition`): rejected — the marker is already the single recognition point across the codebase.
- **D2 — semantics preserved by exclusion at BOTH nomination sides.** Gate A verified `query_nominations.go` has zero definition-awareness today, and the leak is two-sided once definitions carry term tags: (a) **nominee side** — a definition must never be nominated into a pool; (b) **seed side** — `buildTagNominations` derives seed terms from a cluster's top-3 delivered notes (`query_nominations.go:275-288`), and a self-tagged definition landing in a top-3 (common for vocab-flavored queries) would otherwise seed nominations for its term's entire member set. Both sides gain an `isVocabDefinitionNote` filter, each pinned by its own test. Stats/trigger math needs NO change — Gate A verified `collectVaultStats`/`collectTriggerVaultStatsFromNames` already gate on the definition marker independent of tag content. Alternative (let definitions participate): rejected as a recall-payload behavior change outside the ask.
- **D3 — backfill is a new explicit idempotent subcommand, `engram vocab tag-definitions`.** Walks definition notes, derives each term via the existing `termFromDefinitionSlug` helper (`vocab_commands.go:1604` — which already early-returns for the family slug and empty terms), adds the missing self-tag, reports per-note added/already-present/skipped. **Sidecar model (corrected by Gate A):** `ContentHash` hashes situation+body only — frontmatter/tags are stripped (`internal/embed/hash.go:44-56`) — so a tags-only rewrite NEVER stales a sidecar and no refresh step exists or is needed; the precedent is `assignVocabToNote` (`vocab_commands.go:463-498`), which rewrites member tags via plain locked `WriteFile` and never touches sidecars. The backfill follows that exact precedent under `acquireOptionalLock`, supports the standard `--vault` flag (required for the copy-first verification), and slots into the existing `targ.Group("vocab", ...)` reusing `newVocabDeps` — no new deps struct. Alternative (fold into `engram update` or refit): rejected — update must not mutate (prior incident, vault note 532), and a refit is a heavyweight unrelated operation.
- **D4 — minting writes the self-tag at birth, in pinned order.** `mintDefinitionNote` (`vocab_commands.go:1178` — shared by bootstrap, propose, refit-minted definitions AND the family note) derives the term via `termFromDefinitionSlug` and appends the self-tag after the bare marker: `tags: [vocab, vocab/<term>]` (bare-first order is the contract tests assert). The family-note call site (`ensureVocabFamilyNote`) stays bare-only via the helper's family-slug early-return, pinned by a mint-side test. The `:720` comment contract and the family-note Object text at `vocab_commands.go:972-974` (which states the convention in prose and would re-mint it stale) are both updated.
- **D5 — refit must not strip self-tags.** The two-pass retag skips definition notes entirely today (assignment exemption); a regression test pins that a definition's tags survive a refit unchanged.
- **D6 — vault convention record updated at apply.** Note 236's claim ("definition carries a bare vocab tag") is amended to state the new shape when the change is applied, so recall stops surfacing the stale convention.

## Doc-surface enumeration (grep-derived at plan time; dispositions)

Greps: `bare.vocab` / `never vocab/` / `vocab marker` (case-insensitive, tracked *.md) + the same over `internal/`.

| surface | disposition |
|---|---|
| `README.md:96` (vocab bootstrap command doc: "bare `vocab` tag") | update — bootstrap now writes bare + self-tag; ALSO add a Binary-commands entry for the new `engram vocab tag-definitions` subcommand |
| `docs/GLOSSARY.md` "vocab definition note" entry | update — AFTER text pinned: per-term definitions carry `tags: [vocab, vocab/<term>]` (the sentence "a definition note never carries its own term tag" is deleted); the family note stays bare-`vocab`-only; members carry `vocab/<term>` |
| `docs/GLOSSARY.md` retired "vocab term-note" entry (~:254, "bare-`vocab`-tagged" mention) | N/A — historical migration record, stays as written |
| `docs/architecture/adr.md` ADR-0011 representation-update line (+ its echo near :501) | update — append a dated "Representation update (2026-07-27)" line per the #678 annotation precedent: definitions additionally carry their own `vocab/<term>` self-tag (family note excepted); member semantics unchanged |
| `docs/architecture/c2-containers.md:254` (vocab-version flowchart label) | N/A — descriptive of the family note's version field, states no tagging convention |
| `internal/cli/vocab_commands.go:972-974` (family-note Object prose — the convention statement the binary re-mints) | update in group 2 — new-convention text (definitions carry bare marker + self-tag; members carry `vocab/<term>`; family note carries `vocab_version`) |
| Go comments stating bare-only as current | update in code tasks via a grep-driven sweep (`grep -rn 'bare "vocab"\|bare-vocab\|never vocab/' internal/`) — every hit that asserts the bare-only convention as CURRENT is updated (incl. `vocab.go:107/177/222/345`, `vocab_commands.go:720/1068`, `query_nominations.go` comments, test-file comments); hits describing the bare MARKER's discriminator role stay (still true) |
| `openspec/specs/vault-vocab-lifecycle/spec.md` | delta-managed — this change's delta carries the FULL updated requirement content; `/opsx:archive` merges it into the main spec at Step 6 (no hand edit) |
| `docs/superpowers/plans/*`, this change's own artifacts | N/A — historical / self |

## Family-note edge case (D7)

The vocab FAMILY note's slug is `vocab-definition` — stripping the `vocab-`/`-definition` affixes leaves an EMPTY term. The family note documents the tag family itself and has no term of its own: minting and backfill MUST skip it (bare-only remains correct for it), and the backfill reports it as skipped-family rather than malformed.

## Risks / Trade-offs

- [Definitions leak into nomination pools once self-tagged — on EITHER side: as nominees, or as top-3 seeds fanning their term's whole member set into the pool] → D2's two-sided filter in `buildTagNominations` + one unit test per side.
- [An implementer builds sidecar-refresh machinery on the "tags change the hash" misconception] → D3 pins the corrected model (tags are not hash inputs; `assignVocabToNote` precedent writes tags with no sidecar touch); the backfill test asserts post-backfill embed state is clean WITH no refresh code present, pinning the model.
- [A future refit or amend path rewrites definition tags and drops the self-tag] → D5 regression test; `WriteVocabAssignment` unreachable for definitions via the existing exemption.
- [`engram count --group-by tags` counts shift for `vocab/<term>` values (+1 where a definition exists)] → accepted and documented: count reflects frontmatter truth; the route-evidence audit reads `work-kind/tier/outcome` families, unaffected. The vocab-stats surface (the number the triggers consume) is the one held invariant.
- [Obsidian graph only connects via tag nodes when "show tags" is enabled] → accepted; that is already how members interconnect.

## Migration Plan

1. Ship the code change (minting + filter + subcommand) — no data changes yet.
2. Run `engram vocab tag-definitions` once against the live vault; verify with `engram vocab stats` (member counts unchanged), a spot `engram count --group-by tags` diff (each term +1 where its definition exists), and the ask's own success criterion: open 2-3 definitions in Obsidian and confirm the term tag connects each to its member cluster (tasks 5.3).
3. Amend vault note 236 (D6).
4. Rollback: the change is additive tags-only; removing the self-tags (scripted inverse of the backfill) restores the prior shape. No schema or centroid state to roll back.

## Open Questions

(none — behavior-preserving by construction; the one judgment call, nomination exclusion, is decided in D2)
