# Design — Lazy, compositional L2 synthesis at recall

Date: 2026-06-09 · **Freshened: 2026-06-18** · Branch target: a fresh branch off `main`
Status: **design — dual-vector + nearest_l2 shipped; v2 adds in-place `amend`, agent-judged coverage, chunk provenance, append-only chunk history, and recency-weighted distillation. Seven decisions locked; build-ready (cleared five adversarial gates); pending sign-off.**

> **Stale companion artifacts (do not treat as ground truth for the v2 model).** `c1-system-context.md`
> (recall flow) and `skills/recall/SKILL.md` (Step 2.5) still describe the **pre-v2 cosine-band gate**;
> `skills/learn/SKILL.md` still says ingest "prunes stale chunks" (pre-D5). All three are reconciled in
> §7 step 7, *after* the binary/skill work — until then they contradict this spec by design.

## 0. What changed since 2026-06-09 (freshen log)

The original design partly shipped, diverged in the parts that shipped, then a long design pass plus
five adversarial reviews converged it. Two lessons drove the convergence: **chunks do not belong in
the vault *note* model** (every attempt to make them vault files collided with Obsidian-parity, the
scanner, embed/check, Luhmann IDs), and **cosine cannot decide coverage** (a distilled L2 is a
semantic abstraction, never the vector-centroid of its sources, so a cosine threshold systematically
misfires). Converged model: chunks stay in the index; chunk-grounding is **provenance**, not a graph
edge; coverage is **agent-judged**, with cosine only nominating candidates.

| Component | Original | Reality (verified) | This revision |
| --- | --- | --- | --- |
| Dual-vector sidecars (§3.1) | situation + body, `max()` | **Shipped** | done |
| Matching over chunks + notes | n/a | **Shipped** (one ranked list) | done |
| Clustering | one pass | chunks + notes clustered **separately** (`clusterChunkItems` `query.go:1356`) | **one clustering** over the matched set (D1) |
| Candidate nomination | n/a | `nearestInTierIndex` = **top-1** by centroid cosine (`query.go:1578`) | **top-K** by centroid cosine (D7) |
| Coverage decision | cosine threshold | n/a (never wired with links) | **agent-judged** (D7) |
| In-place note edit | implied | `resituate` (`situation:` only), `migrate-links`; `learn` create-only | `engram amend` reuses their DI; new logic (D2) |
| Link edits & embedding | n/a | `ContentHash` covers body incl. `Related to:` (frontmatter excluded) | exclude `Related to:` from the embed source (D3) |
| Chunk → note linkability | L1 episodes (retired) | chunks not addressable | **provenance** — L2 records chunk-ids in frontmatter; chunks stay in the index; materialization/episodes DROPPED (D4) |
| Chunk retention | rebuilt each ingest | `rebuildIndex` replaces from scratch (`ingest.go:379`) | **append-only history** + per-chunk `ingested_at` (D5) |
| Recency | per-source mtime (chunks) | **Shipped** per-source | **per-chunk** (D5) + a first-class **distillation weight** (D6) |

**Verified facts:** matching ranks chunks + notes together; clustering is separate (`query.go:1356/1884`).
`engram amend` is new logic — `rerenderFact`/`rerenderFeedback` (`resituate.go:198-253`) re-emit content
fields *unchanged*. `resolveRelationTargets` is lenient (`relations.go:64`). `chunk.Record` has
Source/Anchor/ContentHash/Text/Vector and **no timestamp** (`chunk/index.go:13`); per-row transcript
timestamps are discarded before chunking (`ingest.go:163-173`) — so per-chunk recency needs a real
field for *both* source types. `learn --source` is a **single required string** (`targets.go:22`), not a
chunk-id list. `ContentHash` excludes frontmatter (`hash.go`/`ExtractBody`) but includes the body's
`Related to:` block — so a new `sources:` frontmatter key does **not** trigger re-embed, but D3 (excluding
`Related to:`) is still required.

## 1. Context & motivation

**Hypothesis (unchanged).** L2 is worth generating once it *proves relevant* — a real recall demands it.
Lazy creation → fewer, only-relevant L2s; and because L2s synthesize from clusters that include existing
L2s, the layer becomes **recursive/compositional**.

**Memory model — two tiers, one matching space.** **Chunks** = episodic memory, mechanical fragments in
the chunk index, append-only history. **L2 notes** = semantic memory, Luhmann-keyed vault notes. Recall
matches and clusters over both; every cluster ends with a single **representative L2** that (a)
`--relation`-links to its **note** sources (the Obsidian-navigable graph) and (b) records its **chunk**
sources as frontmatter provenance (traceability, not graph edges). **Supersession** is pervasive and
handled by **recency-weighted distillation** — recent evidence authoritative *where it conflicts* with
old; old *uncontradicted* evidence stays valid.

## 2. The mechanism (locked — single pass, agent-judged)

Lazy compositional synthesis at **recall**, over **one clustering of all matched memory**:

1. **Match.** A note by `max(cos(situation), cos(body))`, a chunk by its vector cosine, ≥ a match
   threshold. (Matching over chunks + notes is already unified.)
2. **Gather.** All matched **chunks + L2s + L1s** as one set. **L1 episode notes (if any) and chunks are
   synthesis *inputs*** — they inform the agent's synthesis and can be covered by the resulting L2, but are
   never themselves the representative; **the representative is always an L2** (existing or new).
   Items below the match threshold (surfaced only by the recency floor) are *context* for the agent, not
   synthesis inputs.
3. **Cluster once.** One AutoK pass over the matched chunks + notes; each item's coordinate = the vector
   that matched it. (Replaces the separate chunk/note clusterings — D1.)
4. **Decide per cluster — agent-judged (D7).** The binary **nominates** candidate L2s; the **agent
   reads** members + candidates and **decides coverage** (because a distilled L2 is a semantic
   abstraction, never the vector-centroid of its sources — no cosine threshold can decide this). Outcome
   is always: **the cluster has a single representative L2, linked to its sources.**
   - **Covered** (a candidate already states the cluster's principle with no material omission, judged
     against the recency-weighted view) → **activate + link-enrich**: refresh recency and `amend
     --activate` to add `--relation` links to any *note* sources it doesn't yet link, and add new *chunk*
     sources to its provenance. **Do not rewrite its content.**
   - **Near** (a candidate addresses the same situation but omits ≥1 substantive claim the members
     evidence) → **update in place**: the agent re-synthesizes the content from all members
     (recency-weighted, D6) and `amend`s it — new content + note `--relation` links + chunk provenance.
   - **Absent** (no candidate addresses the situation) → **create**: synthesize the single representative
     L2 — fact *or* feedback (one note) — `--relation` to note sources, chunk sources in provenance.

   **Coverage is judged *after* applying the recency weight** (D6): if a candidate matches only the
   *older, superseded* member content while recent members conflict with it, that is **near**, not
   covered. Content synthesis is agent-driven — the agent composes the text and passes it to
   `amend`/`learn`; the binary writes what it's given.
5. **Use.** The agent applies the surfaced + new/updated L2s immediately. Writes are **blocking and
   inline** (not fire-and-forget). *(Supersedes the older fire-and-forget model; c1 reconciled in §7 — see
   the stale-artifacts note at the top.)*

## 3. Architecture & components

### 3.1 Dual-vector sidecars + unified matching — ✅ SHIPPED

`Sidecar{ SchemaVersion=1, …, SituationVector, BodyVector, ContentHash, LastUsed }`; `ContentHash` over
situation+body (frontmatter excluded); `LastUsed` additive. Notes rank by `max(situation, body)`; recall
already ranks chunks + notes in one list. **No further work** except D3's embed-source change (§3.4).

### 3.2 Append-only chunk index + per-chunk recency — 🆕 (D5)

Chunks **stay in the index** (compact, ~12k) — not vault files.

- **Append-only (D5):** `ingest` merge-appends — keep existing records, add only chunks whose
  `ContentHash` isn't present, **never delete** (replaces `rebuildIndex`'s replace-from-scratch,
  `ingest.go:379`). Accepted: unbounded growth; recall surfaces decayed history (resolved at synthesis by
  D6). `loadPriorVectors` changes type from `map[string][]float32` to `map[string]chunk.Record` so
  `ingested_at` survives the merge (`ingest.go:297`).
- **Per-chunk timestamp:** add `IngestedAt time.Time` to `chunk.Record` (both source types — transcript
  per-row timestamps are not threaded into chunks today). **Re-key the recency consumers** (`recency.go`):
  `applyChunkRecency` drops the `ageDaysBySource` map param and reads `r.record.IngestedAt`;
  `newestChunkItems`'s sort key becomes `IngestedAt`; `chunkSourceAges` is removed. `maxTurnBySource` is
  unaffected (a source = one session; turn-frac is intra-session).
- **Migration:** existing records lack `IngestedAt`; backfill from the source's manifest mtime on first
  merge — a one-time approximation. *Caveat: a batch import gives all its sources the same backfill
  timestamp; within-batch ordering is lost (acceptable — relative age within one import is unknown, and
  cosine still distinguishes members).*
- **Stable chunk-id for provenance:** each chunk has a stable id (content hash, or `source#anchor` — an
  engram-internal index key, **not** a vault wikilink). Validating an id requires loading the index
  (`loadChunkRecords` is O(total chunks), no inverted index); build should construct an in-memory id-set
  after load rather than reach for a non-existent O(1) lookup.
- **TDD:** merge-append keeps prior records + adds only new-hash chunks + never deletes;
  `loadPriorVectors` returns full records and `IngestedAt` is **preserved across a re-ingest of an
  unchanged chunk**; per-chunk recency ranks old history below fresh; backfill is deterministic; chunk-ids
  resolve against the index.

### 3.3 Binary — one clustering + top-K candidate nomination — 🆕 (D1 + D7 nomination)

- **One clustering (D1):** **extend `--synthesize-l2`** to include matched **chunks** in the clustered set
  (today it filters to L1+L2 notes via `filterHitsToTiers`, `query.go:2062`, and chunks cluster separately
  at `query.go:1356`). One AutoK pass over the matched chunks + notes; per-item matched-vector coordinates.
  The recall skill (§3.5) calls this mode. No band decision in the binary.
- **Candidate nomination (D7):** per cluster, emit the **top-K candidate L2s (K≥3) by centroid cosine** —
  not the single centroid-nearest. The shipped bug is **top-1** (`nearestInTierIndex` `query.go:1578` +
  singular `queryNearestL2` `query.go:293`), *not* the centroid: the chunk-gap depresses the *absolute*
  cosine from a chunk-heavy centroid to every L2 roughly uniformly, so the **ranking** stays intact and the
  covering L2 sits near the top even when it isn't #1 — top-K recovers it, top-1 can miss it. (Max-member
  cosine — the best cosine to any *single* member — was considered and rejected: it overfits to a cluster
  fragment and masks genuinely multi-theme clusters, which are a clustering problem, not a nomination one.)
  Nomination's job is **recall** (surface the true cover); **precision is the agent's** (D7 — it reads the
  K candidates and rejects off-theme ones), so generous nomination costs nothing. Widen the emitted field
  to `candidate_l2s: [{path, cosine}]`.
- **TDD:** one clustering over a mixed matched set; top-K `candidate_l2s` emitted per cluster; a case where
  the covering L2 is not the centroid-#1 still appears within the top-K.

### 3.4 Binary — in-place `engram amend` + provenance — 🆕 NEW logic on reused DI (D2 & D3)

Modifies a note in place, reusing the `resituate`/`migrate-links` DI pattern; **business logic is new.**

- **`engram amend --target <id> [--relation t|r …] [--chunk-source <id> …] [content flags] [--activate]`.**
- **Relation-merge (new):** parse `Related to:`, dedup, append only new note-targets — idempotent. These
  are note↔note **wikilinks** (the Obsidian graph).
- **Provenance (new):** merge chunk-ids into a frontmatter `sources: [chunk-id, …]` list — idempotent,
  **not** wikilinks (engram-internal traceability resolved against the chunk index). `engram learn` gains
  the **same `--chunk-source <id>` (repeatable)** flag to write `sources:` at create time; the existing
  required `--source` *string* (`targets.go:22`) is unchanged and keeps its human-readable meaning.
- **Field-replacement (new):** overwrite only supplied `subject/predicate/object` /
  `behavior/impact/action`; preserve unsupplied fields, Luhmann ID, `created`, relations, provenance,
  `LastUsed`.
- **D3 — exclude `Related to:` from the embed/`ContentHash` source** so link edits are cheap (no re-embed).
  Body-section parser before `BodyText`/`ContentHash` (`hash.go`) → one-time `engram embed apply --force`.
  *(A `sources:` frontmatter key is already excluded from `ContentHash` — only `Related to:` needs the
  change.)* **Prerequisite of amend's link/provenance-only-no-re-embed** (sequence before amend, §7).
- **Validate targets — fail loud:** a strict variant of `resolveRelationTargets` (the existing one is
  lenient, `relations.go:64`) errors on an unresolved note-target; `--chunk-source` ids must resolve to
  index entries.
- **DI:** inject `Scan`/`Read`/`Write`/`Embedder`; no `os.*` in business logic.
- **TDD:** relation-merge + provenance-merge idempotency; field-replacement (only supplied fields change);
  preservation incl. provenance; content change re-embeds, link/provenance-only does not; unresolved
  target/chunk-id rejected; `--activate` in one write.

### 3.5 `/recall` skill — agent-judged coverage + recency-weighted distillation — 🆕 (D6 & D7)

Rewrite Step 2.5 **per §2**: one per-cluster sweep over the unified clustering; the agent reads candidates
+ members and decides activate / update / create, then writes via `amend`/`learn` with note `--relation`
links + `--chunk-source` provenance.

- **D7 — agent judges coverage; cosine only nominates.** The binary emits `candidate_l2s` (top-K,
  centroid cosine, §3.3); the agent decides by reading. No cosine threshold gate.
- **Coverage rubric (the agent's decision criteria):** **covered** = a candidate's claim states the
  cluster's principle with **no material omission** vs the members; **near** = a candidate addresses the
  same situation but **omits ≥1 substantive claim** the members evidence; **absent** = no candidate
  addresses the situation. Judge against the **recency-weighted view** (below), in this order: (1) read
  candidates + members, (2) apply the recency weight to resolve conflicts, (3) judge coverage against that
  resolved view.
- **D6 — recency a first-class distillation weight.** Synthesis prompt: evidence **conflicts** when a
  newer member explicitly negates/reverses an older claim (reversal cues: "no longer", "replaced by",
  "use X not Y"), **or** the same subject+predicate appears with a different object; in conflict, recent
  wins. Otherwise treat both as independently valid (do **not** demote a stable convention merely for
  lacking recent instances). **Known v2 gap:** *cross-cluster* supersession (the superseding evidence
  didn't cosine-cluster with the old) is not handled — recorded, not solved.
- **Read step:** cluster members carry `{path, score, is_representative}` (`query.go:264`) and a candidate
  may not be in `items`; `engram show <candidate / member>` to load content before deciding.
- **Cost:** per cluster the agent reads candidates (`engram show`), blocking-inline. Bounded by K
  candidates and the AutoK cluster count; recall latency is a **headline experiment metric** (§3.6). If
  the agent-read phase dominates, the documented fallback is to cap agent-judged clusters at the top N by
  score and default the remainder to cosine-nominated **create** (recorded as an accepted v2 limit, not a
  silent cap).
- **TDD:** mandatory `superpowers:writing-skills` (RED→GREEN); **`engram update` distributes the rewritten
  skill to the harness install roots** (§7 gate). Pressure tests: coverage (covered→activate, near→update,
  absent→create); supersession (recent conflicting evidence outnumbered by old → distills to recent, and a
  candidate matching only the superseded content is judged **near**); old-uncontradicted (not demoted).

### 3.6 Experiment — eager-L2 vs lazy-L2 (UNTRACKED — file an issue)

> **Not #646** (recency-recall value-proof). File a dedicated issue.

**Arms:** A·eager vs B·lazy (`l2.lazy`; recall runs single-pass synthesis + agent-judged
activate/update/create writes, **persisting forward** — `vault_out = build-vault-after-recall ∪ the
recall's new L2s`; this is a property of the lazy-L2 recall model in production, exercised by the eval).
**Metrics:** net chain cost (headline); recall latency (the agent-read cost, §3.5); #L2 generated;
say-once preservation; **note-graph link-count / connectivity** *(meaningful only for clusters with ≥1
note source — chunk-only clusters add provenance, **zero** graph edges, by design)*; coverage-decision
quality (false-create / spurious-update); **chunk-index growth**, with an **acceptance criterion**: if the
recency floor accounts for fewer than a set fraction of synthesis inputs after M sessions, growth is
degrading recall and compaction must be scoped. Reuse `dev/eval/cumulative/`. Scope: opus, n=5. Blocked on
#642/#643. Graph-based recall (BFS + hubs vs cosine+recency) is the **downstream hypothesis this informs**,
not a v2 build commitment.

## 4. Data flow — one recall (lazy arm, v2)

```
recall → one clustering of matched memory (chunks + L2s), each cluster w/ top-K candidate L2s (centroid cosine)
  per cluster, the AGENT reads members + candidates, applies recency weight, then judges coverage:
    covered → engram amend <l2> --activate --relation <new note sources> --chunk-source <new ids>   [enrich; no content rewrite]
    near    → engram amend <l2> --relation <note sources> --chunk-source <ids> <re-synth content>    [update in place; recency-weighted]
    absent  → engram learn fact|feedback --position top --relation <note sources> --chunk-source <ids> --source "<descriptive>"   [the ONE representative L2]
  agent applies surfaced + new/updated L2s to the task
```

## 5. Decisions locked vs open

**Locked:**
- **D1 — single pass:** one clustering over matched chunks + notes (extend `--synthesize-l2`). Two-pass is
  the fallback.
- **D2 — `engram amend`:** reuses `resituate`/`migrate-links` DI; relation-merge + provenance-merge +
  field-replacement are new; `learn` stays create-only (gains `--chunk-source`).
- **D3 — exclude `Related to:` from the embed/`ContentHash` source.**
- **D4 — chunks stay in the index; grounding is frontmatter provenance, not wikilinks;** note↔note links
  are wikilinks (the Obsidian graph). Materialization and the episode layer are dropped.
- **D5 — append-only chunk history** + per-chunk `IngestedAt` recency (both source types; migration
  backfill).
- **D6 — recency a first-class distillation weight** (recent authoritative on conflict; old uncontradicted
  retained; cross-cluster supersession a known gap).
- **D7 — agent judges coverage; cosine nominates top-K candidate L2s by centroid cosine** (no cosine band
  gate; nomination = recall, agent = precision). Max-member nomination considered and rejected (overfits
  to a cluster fragment).
- Unified outcome: one representative L2 per cluster (single note, fact *or* feedback), always an L2 (never
  an L1/chunk); "covered" never rewrites content; content synthesis is agent-driven.
- Dual-vector sidecars + unified matching shipped.

**Open / to confirm:**
- Chunk-id scheme detail (content-hash vs `source#anchor`) and K (≥3) — settle at build. If the experiment
  shows extreme chunk-domination making top-K miss the cover, the levers are raising K or a **chunk-down-weighted
  centroid**; build only if observed (YAGNI).
- Recall latency of blocking inline writes + per-cluster agent reads — measured by the experiment (with
  the documented fallback if it dominates).
- `activationCosineCutoff` + note half-life — provisional (#648).
- Coverage-decision quality + chunk-index-growth acceptance criterion — measured (thresholds in §3.6).

## 6. Out of scope (v1/v2)

- Changing **L3** synthesis. A new clusterer. **Chunks as vault files / in the Obsidian graph** (rejected —
  they stay in the index; grounding is provenance). **Graph-based recall** (BFS + hubs) as a build item.
  Chunk-index compaction/TTL. **Cross-cluster supersession** (known gap, D6). Retrofitting non-eval callers.

## 7. Sequencing & gates

1. ✅ **Dual-vector sidecars + unified matching** (3.1).
2. 🆕 **Append-only chunks + per-chunk recency** (3.2, D5) → merge-append/dedup, no prune; `IngestedAt`
   (both types, threaded through ingest); `loadPriorVectors` returns full records (preserves `IngestedAt`);
   recency consumers re-keyed (`applyChunkRecency` signature change, `chunkSourceAges` removed,
   `newestChunkItems` sort key); migration backfill; chunk-id set; `engram check` clean.
3. 🆕 **One clustering + top-K centroid nomination** (3.3, D1/D7) → extend `--synthesize-l2` to include
   chunks; emit `candidate_l2s` per cluster.
4. 🆕 **Exclude `Related to:` from the embed source** (3.4, D3) → body-section parser + one-time `engram
   embed apply --force`.
5. 🆕 **`engram amend` + `learn --chunk-source`** (3.4, D2) → relation-merge + provenance-merge +
   field-replacement + strict target/chunk-id validation + `--activate`. Depends on (4).
6. ⚠️ **`/recall` agent-judged coverage + recency-weighted distillation** (3.5, D6/D7) → `writing-skills`
   RED→GREEN; coverage rubric + conflict heuristic in the skill; **`engram update` distributes the skill to
   harness install roots**; pressure tests (coverage, supersession, old-uncontradicted). Depends on (2)(3)(5).
7. **Reconcile stale artifacts** → `c1-system-context.md` (recall: single-pass inline blocking synthesis,
   agent-judged coverage, top-K nomination, chunk provenance, no materialization; learn: retire the stale
   `transcript --mark` / `learn episode` / L3 ADR diagram); **`skills/recall/SKILL.md`** (rewrite Step 2.5
   from cosine-band gate to agent-judged coverage — this is gate 6's skill work); **`skills/learn/SKILL.md`**
   (remove "prunes stale chunks"; episodes stay retired — chunks are the episodic layer, grounding is
   provenance).
8. **Experiment** (3.6, file an issue) → stub validation, then opus n=5 A/B.
