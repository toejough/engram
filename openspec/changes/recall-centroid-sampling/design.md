# Design: Recall Centroid Sampling

## Context

Recall today: query is embedded; cosine ranking over note vectors + chunk index; floors/caps applied (matched-note floor); the matched notes are clustered locally (k-means silhouette); every local cluster is delivered; then `buildTagNominations` appends, per cluster, all non-definition notes sharing a `vocab/<term>` tag with that cluster's top-3 query-nearest notes (score 0, cap 40, dedup across clusters); supersession ride-alongs are inserted. Problems: (a) the nomination seeds are write-time nearest-centroid stamps with no margin retained — boundary notes carry near-arbitrary labels, and concept-spanning queries (the ones diversity should help most) disproportionately surface such notes; (b) membership dumps are unranked — core and 0.35-fringe members are indistinguishable; (c) payload composition is uncontrolled and degrades with vocab term-count drift.

## Goals / Non-Goals

**Goals:**
- Explicit payload composition: 50% exploit (query-nearest) / 50% explore (concept-diverse), a stated contract.
- Global reach computed from live geometry (query→centroid similarity), not write-time stamps.
- Graded trust: near concepts sample more, far concepts approach zero, no radius threshold.
- Robustness to vocab imperfection in both directions (proliferated or sparse term sets).

**Non-Goals:**
- Changing vocab derivation/minting (companion change `vocab-derivational-refit`; this change reads `vocab.centroids.json` from either producer).
- Changing the chunk channel, matched-note floor, local clustering, or supersession ride-along.
- Margin-adaptive split ratios (noted as a future dial; v1 is fixed 50/50).

## Decisions

1. **Softmax allocation over query→centroid cosine.** Explore budget B = the count of NOTE items delivered in the exploit half (post floor/cap, i.e. the distinct notes in the matched resolved items; runtime-variable, computed after exploit assembly; chunks never count). B=0 (no notes matched) → explore half is skipped entirely with `explore_allocated: {}`. B is allocated across centroids ∝ softmax(sim/τ). Alternative: hard radius gate ("clusters with members within r") — rejected: r is brittle in cosine space; softmax needs only a temperature τ and degrades continuously. τ default derived by this decision procedure (task 3.1): run a representative query set (≥10 real queries) against the production vault; per query compute cosine(query, centroid) for all terms; let sim_top = median across queries of the best centroid similarity and sim_med = median across queries of the median centroid similarity; solve exp(sim_top/τ) / exp(sim_med/τ) = 10 for τ (i.e. τ = (sim_top − sim_med) / ln 10); record the query set, sim_top, sim_med, and τ in LEDGER. Two implementers following this procedure on the same data get the same τ.
2. **Within-cluster sampling weighted by closeness to centroid** (deterministic top-by-centroid-similarity in v1 rather than stochastic sampling — reproducible payloads, simpler tests; revisit stochastic if payloads prove repetitive). Rationale: query-near members are the exploit half's job; the explore half should show each concept's canonical core, and centroid-proximal selection naturally excludes the assignment-floor fringe.
3. **Cluster membership for sampling = notes carrying the `vocab/<term>` tag** (the stored tag set — there is no per-tag score or "top-1" rank in the data; `vocabTermsFromTags` returns a flat set). A multi-tagged note is a candidate under each of its terms; cross-cluster duplication is resolved by the dedupe rule (first selection in allocation order wins). Definition notes are excluded (existing `isVocabDefinitionNote` rule). Within-term ranking needs no stored score: it is cosine(note vector, term centroid), computed at query time from the sidecar vectors already loaded.
4. **Dedupe + backfill:** explore picks already present in the exploit half (or duplicated across clusters) are dropped; freed slots refill following the same softmax allocation order (next-highest remaining cluster weight). Guarantees the explore half stays full when the vault has enough distinct notes.
5. **Match-evidence boost (small, additive):** a cluster with ≥1 member in the exploit half SHALL receive a flat additive similarity bonus δ applied to its cosine before softmax (flat — it does not stack per member; bounded by construction since δ is a constant). Default δ = 0.05; calibrated alongside τ in task 3.1 (bar: the bonus must be able to lift a vault-median-sim evidenced cluster above an un-evidenced cluster at sim_med + δ/2, and must NOT reorder two un-evidenced clusters — trivially true for additive constants). This preserves the one genuine capability tag nomination had — reach to a concept far from the query but evidenced by an actual match — while sourcing it from live match evidence rather than a lone write-time stamp.
6. **Reporting:** each explore note carries `provenance: explore` and its source term; the query budget gains `explore_allocated` per term and drops `tag_nominations_added/dropped` (BREAKING for budget readers). `buildTagNominations` and its spec requirement are removed.
7. **No LLM in the loop; zero new I/O surfaces.** Centroid loading reuses the existing `vocab.centroids.json` reader through injected deps (DI rules unchanged).

## Risks / Trade-offs

- [Fixed 50/50 wastes budget on queries with one dominant concept] → far-centroid softmax mass collapses toward the near cluster, whose core overlaps the exploit half; dedupe+backfill then recycles slots — the split self-corrects toward exploit in the degenerate case.
- [Missing/stale `vocab.centroids.json` (e.g. never-bootstrapped vault)] → explore half silently empty, exploit half fills the whole budget; report `explore_allocated: {}` so absence is visible, not silent (fail-loud-enough per eval conventions).
- [Payload-quality regression vs. nomination on real queries] → gate with a before/after comparison on the production vault + C3–C6-style trap checks before deploy; keep the removal in one commit for clean rollback.
- [Centroid staleness between refits misallocates explore budget] → acceptable: same staleness exists today via stamps; derivational companion reduces it.

## Migration Plan

1. Implement behind the query path; validate with the payload comparison harness on the real vault.
2. Ship binary; `engram update` deploys any recall-skill wording changes (audit `agent-instructions/skills/recall/SKILL.md` for nomination-field references; skill edits via `superpowers:writing-skills`).
3. Rollback: previous binary restores nomination; no data-format changes.

## Open Questions

- Temperature τ default and the match-evidence bonus magnitude — calibrate empirically.
- Whether chunk-channel explore sampling is worth a follow-up once note-channel results are in.
