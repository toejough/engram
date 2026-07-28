# Proposal: Recall Payload — kNN + Proximity-Proportional Centroid Sampling

## Why

Recall's global reach currently comes from tag nomination: the top-3 query-nearest notes of each local cluster contribute their write-time `vocab/<term>` stamps, and every note carrying those tags is appended unranked (score 0, cap 40/cluster). This discards geometry at both trust-critical moments — the seed stamps are write-time nearest-centroid labels with no margin retained (a boundary note's stamp is effectively a coin flip), and nominated members are set-membership dumps with no core-vs-fringe distinction. It also makes recall quality hostage to vocab term-count drift (a proliferated vocab dumps splinter memberships). Replacing nomination with direct centroid-proximity sampling gives graded, budgeted, geometry-honest diversity that is robust to imperfect vocab.

## What Changes

- **Payload composition contract: 50% exploit / 50% explore.** The exploit half is the existing cosine top-n nearest notes (floors and caps unchanged). The explore half is sampled from global clusters: each cluster receives a sample allocation proportional to softmax(query→centroid cosine similarity), so near concepts contribute more, far concepts approach zero — no radius threshold to tune.
- **Within-cluster sampling is centroid-proximal.** Explore samples are drawn weighted by closeness to the cluster centroid (the cluster's canonical core), since query-near members are already covered by the exploit half; the 0.35-fringe is naturally down-weighted.
- **Dedupe and backfill.** Explore samples already present in the exploit half are deduplicated; freed slots backfill from the next-nearest cluster allocation so the explore half stays full.
- **Tag nomination is removed** from the query path (`buildTagNominations`, `tag_nominations_added`/`_dropped` budget fields). **BREAKING** for payload consumers reading those budget fields. Explore provenance is reported instead (per-note `provenance: explore` + per-cluster allocation counts in the query budget).
- **Supersession ride-along, local clustering of matches, matched-note floor, and the two-channel (notes+chunks) design are unchanged.** Explore sampling applies to the note channel only.
- Centroids are read from `vocab.centroids.json` — the same file the vocab lifecycle maintains today and the companion change `vocab-derivational-refit` re-derives. This change works against either producer.

## Capabilities

### New Capabilities

- `recall-centroid-sampling`: composition, allocation, within-cluster sampling, dedupe/backfill, and budget-reporting requirements for the explore half of the recall payload.

### Modified Capabilities

- `vault-vocab-lifecycle`: the "Tag nomination in recall queries" requirement is removed (definition-note exclusion rules for nomination become moot; all other requirements unchanged).

## Impact

- `internal/cli` query path (nomination removal, sampling insertion, budget fields), centroid loading (`vocab.centroids.json` reader reuse), payload/YAML output shape.
- `agent-instructions/skills/recall/SKILL.md` if it names nomination fields (audit required; skill edits go through `superpowers:writing-skills`).
- Eval exposure: payload-composition change should be gated by the existing C3–C6 trap-harness style checks before deploy (per dev/eval/LEDGER.md conventions); a before/after payload comparison on the real vault is a cheap validity check.
