# 792g7 memory-query - Adoption Audit

Audit of [792a7 memory-query](../792a_product-spec.md#792a7-added-memory-query) and the 792b requirements that support it, against the code at dd2bd9e2 and a binary built from it, run on temporary vaults (never the real vault). Method: read each implementing function, then exercised the binary for query input, payload shape, warnings, recency scaling (chunk age, `last_used`, quoted and bare `created`, late turns), clusters, explore half, ride-along, recent fill, content budget, lazy chunks, `--project`, `--limit`, trigger matching, red-flag preview, `show`, `show-chunk`, `query-chunks`, CRLF notes and read-only behavior.

Result: 4 code bug (#774, and #804 for 792a7u, 792a7u1d and 792b24c12), 2 doc fixed, 208 match, 0 unclear (rows).

- Doc fixes: [792a7r1b A long chunk is cut with an ellipsis](../792a_product-spec.md#792a7r1-uat-scenarios) described a budget that cannot be set; [792a7x Reading one note](../792a_product-spec.md#792a7x-reading-one-note) now states that `.md` is kept when a `|display` part is present.
- Code bug: #774 (never-used `engram learn` notes are not recency-scaled because `created` is quoted). Commented on #774: its title overstates the problem, since chunk and `last_used` decay are live.
- Code bug: #804 ([792a7u Project filter](../792a_product-spec.md#792a7u-project-filter) and [792b24c12 Project filter](../792b_technical-spec.md#792b24c12-project-filter)): explore and ride-along notes escape `--project`. It was `unclear`; Joe ruled it a bug (2026-10-10) and the spec now states the intent (every note item is filtered).
- Covered by #795/#796: the 792a7 intro sentence that a query on a missing vault creates it is vault-creation behavior owned by those changes (it works today); it is not a scenario of 792a7.
- Noted, not filed: the comment on `provenanceTrigger` in `internal/cli/query.go` still says a trigger is a substring of `--text`; matching is whole-word.

| Spec ID | Verdict | Evidence |
| --- | --- | --- |
| [792a7a Query input](../792a_product-spec.md#792a7a-query-input) | match | query.go:2064 validateQueryArgs; verified on binary (rc 1, `query: empty query string`) |
| [792a7a1a A query with nothing to search for fails](../792a_product-spec.md#792a7a1-uat-scenarios) | match | query.go:2064 validateQueryArgs; verified on binary (rc 1, `query: empty query string`) |
| [792a7a1b Several phrases are searched together](../792a_product-spec.md#792a7a1-uat-scenarios) | match | query.go:2064 validateQueryArgs; verified on binary (rc 1, `query: empty query string`) |
| [792a7b The payload](../792a_product-spec.md#792a7b-the-payload) | match | query.go:1744 renderQueryPayload, :1190 kindFromContent; verified on binary |
| [792a7b1a The payload has the documented shape](../792a_product-spec.md#792a7b1-uat-scenarios) | match | query.go:1744 renderQueryPayload, :1190 kindFromContent; verified on binary |
| [792a7b1b A note item is labelled by its type](../792a_product-spec.md#792a7b1-uat-scenarios) | match | query.go:1744 renderQueryPayload, :1190 kindFromContent; verified on binary |
| [792a7b1c Wikilinks are flattened in the content](../792a_product-spec.md#792a7b1-uat-scenarios) | match | query.go:1744 renderQueryPayload, :1190 kindFromContent; verified on binary |
| [792a7c Embeddings of the current model](../792a_product-spec.md#792a7c-embeddings-of-the-current-model) | match | query.go:2077 warnModelMismatch, :2100 warnOldSchema, RunQuery guard; verified on binary (both warnings, error) |
| [792a7c1a Notes of another model are dropped with a warning](../792a_product-spec.md#792a7c1-uat-scenarios) | match | query.go:2077 warnModelMismatch, :2100 warnOldSchema, RunQuery guard; verified on binary (both warnings, error) |
| [792a7c1b A vault with no usable embeddings is an error](../792a_product-spec.md#792a7c1-uat-scenarios) | match | query.go:2077 warnModelMismatch, :2100 warnOldSchema, RunQuery guard; verified on binary (both warnings, error) |
| [792a7c1c Old-schema sidecars are skipped with their own warning](../792a_product-spec.md#792a7c1-uat-scenarios) | match | query.go:2077 warnModelMismatch, :2100 warnOldSchema, RunQuery guard; verified on binary (both warnings, error) |
| [792a7d Matched notes and chunks](../792a_product-spec.md#792a7d-matched-notes-and-chunks) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7d1a Notes are ranked by their better axis](../792a_product-spec.md#792a7d1-uat-scenarios) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7d1b An item below the floor is dropped](../792a_product-spec.md#792a7d1-uat-scenarios) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7d1c Only the best 30 of a phrase are kept](../792a_product-spec.md#792a7d1-uat-scenarios) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7d1d An item keeps its best score across phrases](../792a_product-spec.md#792a7d1-uat-scenarios) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7d1e The matched set is capped at 300](../792a_product-spec.md#792a7d1-uat-scenarios) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7d1f Notes and chunks compete in one list](../792a_product-spec.md#792a7d1-uat-scenarios) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7d1g A definition note is an ordinary match](../792a_product-spec.md#792a7d1-uat-scenarios) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7d1h A note is found by its own situation](../792a_product-spec.md#792a7d1-uat-scenarios) | match | query.go:1365 mergePhraseIntoUnion (top 30), :599 applyFloorAndCap (0.25, 300); verified |
| [792a7e Matched-note floor](../792a_product-spec.md#792a7e-matched-note-floor) | match | query.go:923 capWithNoteFloor (noteFloorK 5); unit-tested |
| [792a7e1a A relevant note survives a flood of chunks](../792a_product-spec.md#792a7e1-uat-scenarios) | match | query.go:923 capWithNoteFloor (noteFloorK 5); unit-tested |
| [792a7e1b A note below the floor is not promoted](../792a_product-spec.md#792a7e1-uat-scenarios) | match | query.go:923 capWithNoteFloor (noteFloorK 5); unit-tested |
| [792a7e1c Several notes share the reserved slots](../792a_product-spec.md#792a7e1-uat-scenarios) | match | query.go:923 capWithNoteFloor (noteFloorK 5); unit-tested |
| [792a7e1d A definition note is promoted like any other](../792a_product-spec.md#792a7e1-uat-scenarios) | match | query.go:923 capWithNoteFloor (noteFloorK 5); unit-tested |
| [792a7f Recency scaling](../792a_product-spec.md#792a7f-recency-scaling) | match | recency.go:223 noteAgeDays, :287 recencyMultiplier; query_chunks.go:203 scoreChunkForPhrase; verified on binary (120d chunk 0.181 vs 0.724, last_used 120d 0.248, late turn x1.2) |
| [792a7f1a An old chunk is scaled down](../792a_product-spec.md#792a7f1-uat-scenarios) | match | recency.go:223 noteAgeDays, :287 recencyMultiplier; query_chunks.go:203 scoreChunkForPhrase; verified on binary (120d chunk 0.181 vs 0.724, last_used 120d 0.248, late turn x1.2) |
| [792a7f1b A long-unused note is scaled down](../792a_product-spec.md#792a7f1-uat-scenarios) | match | recency.go:223 noteAgeDays, :287 recencyMultiplier; query_chunks.go:203 scoreChunkForPhrase; verified on binary (120d chunk 0.181 vs 0.724, last_used 120d 0.248, late turn x1.2) |
| [792a7f1c The floor ignores the scaling](../792a_product-spec.md#792a7f1-uat-scenarios) | match | recency.go:223 noteAgeDays, :287 recencyMultiplier; query_chunks.go:203 scoreChunkForPhrase; verified on binary (120d chunk 0.181 vs 0.724, last_used 120d 0.248, late turn x1.2) |
| [792a7f1d A note that was never used is not scaled](../792a_product-spec.md#792a7f1-uat-scenarios) | code bug #774 | recency.go:248 parseCreatedFromNote keeps the quotes `engram learn` writes, so noteAgeDays:223 returns 0; verified (score 1.0 vs 0.248 for a bare-date note). Scenario describes current behavior; #774 tracks the fix |
| [792a7f1e A late turn is lifted](../792a_product-spec.md#792a7f1-uat-scenarios) | match | recency.go:223 noteAgeDays, :287 recencyMultiplier; query_chunks.go:203 scoreChunkForPhrase; verified on binary (120d chunk 0.181 vs 0.724, last_used 120d 0.248, late turn x1.2) |
| [792a7g Clusters](../792a_product-spec.md#792a7g-clusters) | match | query.go:968 clusterMatchedSet, cluster/autok.go:28 AutoK (silhouette >= 0.10); verified on binary |
| [792a7g1a Matches form clusters with a representative](../792a_product-spec.md#792a7g1-uat-scenarios) | match | query.go:968 clusterMatchedSet, cluster/autok.go:28 AutoK (silhouette >= 0.10); verified on binary |
| [792a7g1b Matches with no structure form one cluster](../792a_product-spec.md#792a7g1-uat-scenarios) | match | query.go:968 clusterMatchedSet, cluster/autok.go:28 AutoK (silhouette >= 0.10); verified on binary |
| [792a7g1c Nothing matched means no clusters](../792a_product-spec.md#792a7g1-uat-scenarios) | match | query.go:968 clusterMatchedSet, cluster/autok.go:28 AutoK (silhouette >= 0.10); verified on binary |
| [792a7g1d The same query gives the same clusters](../792a_product-spec.md#792a7g1-uat-scenarios) | match | query.go:968 clusterMatchedSet, cluster/autok.go:28 AutoK (silhouette >= 0.10); verified on binary |
| [792a7h Candidate notes](../792a_product-spec.md#792a7h-candidate-notes) | match | query.go:2027 topKCandidateNotes (candidateNoteK 5); verified on binary |
| [792a7h1a Candidates are the cluster's nearest notes](../792a_product-spec.md#792a7h1-uat-scenarios) | match | query.go:2027 topKCandidateNotes (candidateNoteK 5); verified on binary |
| [792a7h1b A cluster of chunks has no candidates](../792a_product-spec.md#792a7h1-uat-scenarios) | match | query.go:2027 topKCandidateNotes (candidateNoteK 5); verified on binary |
| [792a7h1c A small cluster lists all its notes](../792a_product-spec.md#792a7h1-uat-scenarios) | match | query.go:2027 topKCandidateNotes (candidateNoteK 5); verified on binary |
| [792a7i Half of the notes are explored](../792a_product-spec.md#792a7i-half-of-the-notes-are-explored) | match | query.go:669-690 assembleResolvedItems (budget = distinct exploit notes), query_explore.go:119 computeExploreHalf; verified (4 of 10 on a small vault, {} without centroids) |
| [792a7i1a The halves are equal on a healthy vault](../792a_product-spec.md#792a7i1-uat-scenarios) | match | query.go:669-690 assembleResolvedItems (budget = distinct exploit notes), query_explore.go:119 computeExploreHalf; verified (4 of 10 on a small vault, {} without centroids) |
| [792a7i1b Missing centroids leave only the matches](../792a_product-spec.md#792a7i1-uat-scenarios) | match | query.go:669-690 assembleResolvedItems (budget = distinct exploit notes), query_explore.go:119 computeExploreHalf; verified (4 of 10 on a small vault, {} without centroids) |
| [792a7i1c No matched note means no explore half](../792a_product-spec.md#792a7i1-uat-scenarios) | match | query.go:669-690 assembleResolvedItems (budget = distinct exploit notes), query_explore.go:119 computeExploreHalf; verified (4 of 10 on a small vault, {} without centroids) |
| [792a7i1d A small vault shortens the explore half](../792a_product-spec.md#792a7i1-uat-scenarios) | match | query.go:669-690 assembleResolvedItems (budget = distinct exploit notes), query_explore.go:119 computeExploreHalf; verified (4 of 10 on a small vault, {} without centroids) |
| [792a7j Explore allocation follows the query](../792a_product-spec.md#792a7j-explore-allocation-follows-the-query) | match | query_explore.go:311 softmaxAllocate, exploreTemperatureDefault 0.106, exploreMatchEvidenceBonus 0.05 |
| [792a7j1a Nearer terms sample more](../792a_product-spec.md#792a7j1-uat-scenarios) | match | query_explore.go:311 softmaxAllocate, exploreTemperatureDefault 0.106, exploreMatchEvidenceBonus 0.05 |
| [792a7j1b Evidence lifts a distant term](../792a_product-spec.md#792a7j1-uat-scenarios) | match | query_explore.go:311 softmaxAllocate, exploreTemperatureDefault 0.106, exploreMatchEvidenceBonus 0.05 |
| [792a7j1c The bonus does not stack](../792a_product-spec.md#792a7j1-uat-scenarios) | match | query_explore.go:311 softmaxAllocate, exploreTemperatureDefault 0.106, exploreMatchEvidenceBonus 0.05 |
| [792a7k Explore picks the core of a term](../792a_product-spec.md#792a7k-explore-picks-the-core-of-a-term) | match | query_explore.go:270 selectWithinCluster, exploreMembersByTerm; query_vault_meta.go:157 loadAllVaultNotesMeta |
| [792a7k1a The core outranks the fringe](../792a_product-spec.md#792a7k1-uat-scenarios) | match | query_explore.go:270 selectWithinCluster, exploreMembersByTerm; query_vault_meta.go:157 loadAllVaultNotesMeta |
| [792a7k1b Definition notes are never picked](../792a_product-spec.md#792a7k1-uat-scenarios) | match | query_explore.go:270 selectWithinCluster, exploreMembersByTerm; query_vault_meta.go:157 loadAllVaultNotesMeta |
| [792a7l Explore picks are not repeated](../792a_product-spec.md#792a7l-explore-picks-are-not-repeated) | match | query_explore.go:176 dedupeAndBackfill |
| [792a7l1a A duplicate is replaced](../792a_product-spec.md#792a7l1-uat-scenarios) | match | query_explore.go:176 dedupeAndBackfill |
| [792a7l1b A note under two terms is picked once](../792a_product-spec.md#792a7l1-uat-scenarios) | match | query_explore.go:176 dedupeAndBackfill |
| [792a7l1c Exhausted candidates leave the budget unmet](../792a_product-spec.md#792a7l1-uat-scenarios) | match | query_explore.go:176 dedupeAndBackfill |
| [792a7m Explore picks are reported](../792a_product-spec.md#792a7m-explore-picks-are-reported) | match | query.go explorePicksToResolvedItems; verified on binary (provenance explore, source_term, outside clusters) |
| [792a7m1a An explore note names its term](../792a_product-spec.md#792a7m1-uat-scenarios) | match | query.go explorePicksToResolvedItems; verified on binary (provenance explore, source_term, outside clusters) |
| [792a7m1b Explore notes are outside the clusters](../792a_product-spec.md#792a7m1-uat-scenarios) | match | query.go explorePicksToResolvedItems; verified on binary (provenance explore, source_term, outside clusters) |
| [792a7m1c The budget counts what was delivered](../792a_product-spec.md#792a7m1-uat-scenarios) | match | query.go explorePicksToResolvedItems; verified on binary (provenance explore, source_term, outside clusters) |
| [792a7n Superseding notes ride along](../792a_product-spec.md#792a7n-superseding-notes-ride-along) | match | query_vault_meta.go:93 applySupersedesRideAlong; verified on binary (ride_along score 0, missing sidecar skipped) |
| [792a7n1a The newer note follows the older one](../792a_product-spec.md#792a7n1-uat-scenarios) | match | query_vault_meta.go:93 applySupersedesRideAlong; verified on binary (ride_along score 0, missing sidecar skipped) |
| [792a7n1b A note already present is not inserted again](../792a_product-spec.md#792a7n1-uat-scenarios) | match | query_vault_meta.go:93 applySupersedesRideAlong; verified on binary (ride_along score 0, missing sidecar skipped) |
| [792a7n1c Several superseders are all inserted](../792a_product-spec.md#792a7n1-uat-scenarios) | match | query_vault_meta.go:93 applySupersedesRideAlong; verified on binary (ride_along score 0, missing sidecar skipped) |
| [792a7n1d A missing superseder is skipped](../792a_product-spec.md#792a7n1-uat-scenarios) | match | query_vault_meta.go:93 applySupersedesRideAlong; verified on binary (ride_along score 0, missing sidecar skipped) |
| [792a7o The newest chunks](../792a_product-spec.md#792a7o-the-newest-chunks) | match | query.go:838 buildRecentFillItems, :1829 resolveRecentFill, recency.go:189 newestChunkItems; verified on binary (0 = default, =-1 off, env, no-time last) |
| [792a7o1a Recent chunks follow the matches](../792a_product-spec.md#792a7o1-uat-scenarios) | match | query.go:838 buildRecentFillItems, :1829 resolveRecentFill, recency.go:189 newestChunkItems; verified on binary (0 = default, =-1 off, env, no-time last) |
| [792a7o1b A matched chunk is not repeated](../792a_product-spec.md#792a7o1-uat-scenarios) | match | query.go:838 buildRecentFillItems, :1829 resolveRecentFill, recency.go:189 newestChunkItems; verified on binary (0 = default, =-1 off, env, no-time last) |
| [792a7o1c The count can be set](../792a_product-spec.md#792a7o1-uat-scenarios) | match | query.go:838 buildRecentFillItems, :1829 resolveRecentFill, recency.go:189 newestChunkItems; verified on binary (0 = default, =-1 off, env, no-time last) |
| [792a7o1d A negative count turns the channel off](../792a_product-spec.md#792a7o1-uat-scenarios) | match | query.go:838 buildRecentFillItems, :1829 resolveRecentFill, recency.go:189 newestChunkItems; verified on binary (0 = default, =-1 off, env, no-time last) |
| [792a7o1e Zero means the default](../792a_product-spec.md#792a7o1-uat-scenarios) | match | query.go:838 buildRecentFillItems, :1829 resolveRecentFill, recency.go:189 newestChunkItems; verified on binary (0 = default, =-1 off, env, no-time last) |
| [792a7o1f A chunk without a time sorts last](../792a_product-spec.md#792a7o1-uat-scenarios) | match | query.go:838 buildRecentFillItems, :1829 resolveRecentFill, recency.go:189 newestChunkItems; verified on binary (0 = default, =-1 off, env, no-time last) |
| [792a7p When a chunk was first indexed](../792a_product-spec.md#792a7p-when-a-chunk-was-first-indexed) | match | ingest.go:342 ingestTimeFor, :417 mergeChunkRecords, manifestBackfill; chunk/index.go IngestedAt |
| [792a7p1a A Markdown chunk takes the ingest time](../792a_product-spec.md#792a7p1-uat-scenarios) | match | ingest.go:342 ingestTimeFor, :417 mergeChunkRecords, manifestBackfill; chunk/index.go IngestedAt |
| [792a7p1b A transcript chunk takes the session's last timestamp](../792a_product-spec.md#792a7p1-uat-scenarios) | match | ingest.go:342 ingestTimeFor, :417 mergeChunkRecords, manifestBackfill; chunk/index.go IngestedAt |
| [792a7p1c Ingesting again keeps the first time](../792a_product-spec.md#792a7p1-uat-scenarios) | match | ingest.go:342 ingestTimeFor, :417 mergeChunkRecords, manifestBackfill; chunk/index.go IngestedAt |
| [792a7p1d An old chunk gets its source's modification time](../792a_product-spec.md#792a7p1-uat-scenarios) | match | ingest.go:342 ingestTimeFor, :417 mergeChunkRecords, manifestBackfill; chunk/index.go IngestedAt |
| [792a7q Limit](../792a_product-spec.md#792a7q-limit) | match | query.go:1744 renderQueryPayload (splitTriggerItems, splitRecencyChannel, capItemsToLimit); verified on binary (--limit 12 counts explore notes, per 792d92) |
| [792a7q1a The default caps a large result](../792a_product-spec.md#792a7q1-uat-scenarios) | match | query.go:1744 renderQueryPayload (splitTriggerItems, splitRecencyChannel, capItemsToLimit); verified on binary (--limit 12 counts explore notes, per 792d92) |
| [792a7q1b An explicit limit caps the result](../792a_product-spec.md#792a7q1-uat-scenarios) | match | query.go:1744 renderQueryPayload (splitTriggerItems, splitRecencyChannel, capItemsToLimit); verified on binary (--limit 12 counts explore notes, per 792d92) |
| [792a7q1c A small result is returned whole](../792a_product-spec.md#792a7q1-uat-scenarios) | match | query.go:1744 renderQueryPayload (splitTriggerItems, splitRecencyChannel, capItemsToLimit); verified on binary (--limit 12 counts explore notes, per 792d92) |
| [792a7q1d Trigger hits do not use the limit](../792a_product-spec.md#792a7q1-uat-scenarios) | match | query.go:1744 renderQueryPayload (splitTriggerItems, splitRecencyChannel, capItemsToLimit); verified on binary (--limit 12 counts explore notes, per 792d92) |
| [792a7q1e The recent chunks survive a full set of matches](../792a_product-spec.md#792a7q1-uat-scenarios) | match | query.go:1744 renderQueryPayload (splitTriggerItems, splitRecencyChannel, capItemsToLimit); verified on binary (--limit 12 counts explore notes, per 792d92) |
| [792a7r Chunk content budget](../792a_product-spec.md#792a7r-chunk-content-budget) | match | query.go:882 capChunkContent, :1967 snippet, :1818 resolveContentBudget; verified on binary |
| [792a7r1a Later chunks become snippets](../792a_product-spec.md#792a7r1-uat-scenarios) | match | query.go:882 capChunkContent, :1967 snippet, :1818 resolveContentBudget; verified on binary |
| [792a7r1b A long chunk is cut with an ellipsis](../792a_product-spec.md#792a7r1-uat-scenarios) | doc fixed | Scenario said `a content budget of 0 chunks kept in full`, which cannot be set: 0 means the default 15 (query.go:1818) and capChunkContent:882 does nothing at 0. Reworded to budget 1; also in the 792b mapping copy |
| [792a7r1c Notes are never reduced](../792a_product-spec.md#792a7r1-uat-scenarios) | match | query.go:882 capChunkContent, :1967 snippet, :1818 resolveContentBudget; verified on binary |
| [792a7r1d A negative budget means none](../792a_product-spec.md#792a7r1-uat-scenarios) | match | query.go:882 capChunkContent, :1967 snippet, :1818 resolveContentBudget; verified on binary |
| [792a7s Lazy chunks](../792a_product-spec.md#792a7s-lazy-chunks) | match | query.go:954 clearChunkContent; verified on binary (lazy_chunks only when set, env works) |
| [792a7s1a Chunks carry no content](../792a_product-spec.md#792a7s1-uat-scenarios) | match | query.go:954 clearChunkContent; verified on binary (lazy_chunks only when set, env works) |
| [792a7s1b Notes keep their content](../792a_product-spec.md#792a7s1-uat-scenarios) | match | query.go:954 clearChunkContent; verified on binary (lazy_chunks only when set, env works) |
| [792a7s1c The environment can turn it on](../792a_product-spec.md#792a7s1-uat-scenarios) | match | query.go:954 clearChunkContent; verified on binary (lazy_chunks only when set, env works) |
| [792a7s1d Without the flag the field is absent](../792a_product-spec.md#792a7s1-uat-scenarios) | match | query.go:954 clearChunkContent; verified on binary (lazy_chunks only when set, env works) |
| [792a7t Reading one chunk](../792a_product-spec.md#792a7t-reading-one-chunk) | match | show_chunk.go:32 RunShowChunk; verified on binary |
| [792a7t1a A chunk's text is printed](../792a_product-spec.md#792a7t1-uat-scenarios) | match | show_chunk.go:32 RunShowChunk; verified on binary |
| [792a7t1b An unknown chunk is an error](../792a_product-spec.md#792a7t1-uat-scenarios) | match | show_chunk.go:32 RunShowChunk; verified on binary |
| [792a7t1c An empty ID is an error](../792a_product-spec.md#792a7t1-uat-scenarios) | match | show_chunk.go:32 RunShowChunk; verified on binary |
| [792a7u Project filter](../792a_product-spec.md#792a7u-project-filter) | code bug #804 | Code matches the text (query.go:669 filters matched notes only), but #636 and the flag help say `restrict items`, and the filter predates the explore half (3b740212 2026-05-26 vs 2bf8f2d6 2026-07-29), which leaks other-project notes. Best judgment: unconsidered interaction. Filed #804 to decide; Rationale line added citing it Joe ruled it a bug (2026-10-10); intent is that `--project` restricts every note item (#636, flag help); spec rewritten to say so, and Joe's ruling is commented on #804. |
| [792a7u1a Notes of other projects are dropped](../792a_product-spec.md#792a7u1-uat-scenarios) | match | query.go:1172 itemMatchesProject, :669 applyProjectFilter(resolved); verified on binary: explore notes of other projects and unfiltered clusters returned |
| [792a7u1b A body mention does not match](../792a_product-spec.md#792a7u1-uat-scenarios) | match | query.go:1172 itemMatchesProject, :669 applyProjectFilter(resolved); verified on binary: explore notes of other projects and unfiltered clusters returned |
| [792a7u1c No filter drops nothing](../792a_product-spec.md#792a7u1-uat-scenarios) | match | query.go:1172 itemMatchesProject, :669 applyProjectFilter(resolved); verified on binary: explore notes of other projects and unfiltered clusters returned |
| [792a7u1d Explore and ride-along notes of other projects are dropped](../792a_product-spec.md#792a7u1-uat-scenarios) | code bug #804 | new scenario for the ruling; the code filters matched notes only (`query.go:669`). |
| [792a7v Runbooks rank like other notes](../792a_product-spec.md#792a7v-runbooks-rank-like-other-notes) | match | no kind-specific code in rankCandidates (query.go:1615); runbook ranked as a fact |
| [792a7v1a A runbook is returned like a fact](../792a_product-spec.md#792a7v1-uat-scenarios) | match | no kind-specific code in rankCandidates (query.go:1615); runbook ranked as a fact |
| [792a7v1b Surfacing does not need a candidate slot](../792a_product-spec.md#792a7v1-uat-scenarios) | match | no kind-specific code in rankCandidates (query.go:1615); runbook ranked as a fact |
| [792a7v1c A runbook has no kind boost](../792a_product-spec.md#792a7v1-uat-scenarios) | match | no kind-specific code in rankCandidates (query.go:1615); runbook ranked as a fact |
| [792a7w Red flags in the query preview](../792a_product-spec.md#792a7w-red-flags-in-the-query-preview) | match | redflags_truncation.go:37 capRedFlagsForPreview (1200 byte budget); verified on binary (30 flags -> 20 omitted, show prints all 30) |
| [792a7w1a Red flags are in the payload](../792a_product-spec.md#792a7w1-uat-scenarios) | match | redflags_truncation.go:37 capRedFlagsForPreview (1200 byte budget); verified on binary (30 flags -> 20 omitted, show prints all 30) |
| [792a7w1b An oversized list keeps its newest entries](../792a_product-spec.md#792a7w1-uat-scenarios) | match | redflags_truncation.go:37 capRedFlagsForPreview (1200 byte budget); verified on binary (30 flags -> 20 omitted, show prints all 30) |
| [792a7w1c A short list is unchanged](../792a_product-spec.md#792a7w1-uat-scenarios) | match | redflags_truncation.go:37 capRedFlagsForPreview (1200 byte budget); verified on binary (30 flags -> 20 omitted, show prints all 30) |
| [792a7w1d A trigger hit is truncated the same way](../792a_product-spec.md#792a7w1-uat-scenarios) | match | redflags_truncation.go:37 capRedFlagsForPreview (1200 byte budget); verified on binary (30 flags -> 20 omitted, show prints all 30) |
| [792a7x Reading one note](../792a_product-spec.md#792a7x-reading-one-note) | doc fixed | normalizeNoteRef/ParseWikilinks drop `.md` only at the end of the whole link text, so `[[2.alpha.md|disp]]` is listed as `2.alpha.md|disp` (verified on binary). Requirement text now says so |
| [792a7x1a A note is printed with its links](../792a_product-spec.md#792a7x1-uat-scenarios) | match | show.go:37 RunShow, note_ref.go:10 normalizeNoteRef, vaultgraph/parser.go:25 ParseWikilinks; verified on binary |
| [792a7x1b A note with no links says so](../792a_product-spec.md#792a7x1-uat-scenarios) | match | show.go:37 RunShow, note_ref.go:10 normalizeNoteRef, vaultgraph/parser.go:25 ParseWikilinks; verified on binary |
| [792a7x1c Every reference form resolves](../792a_product-spec.md#792a7x1-uat-scenarios) | match | show.go:37 RunShow, note_ref.go:10 normalizeNoteRef, vaultgraph/parser.go:25 ParseWikilinks; verified on binary |
| [792a7x1d Show never truncates red flags](../792a_product-spec.md#792a7x1-uat-scenarios) | match | show.go:37 RunShow, note_ref.go:10 normalizeNoteRef, vaultgraph/parser.go:25 ParseWikilinks; verified on binary |
| [792a7x1e A fenced link is not listed](../792a_product-spec.md#792a7x1-uat-scenarios) | match | show.go:37 RunShow, note_ref.go:10 normalizeNoteRef, vaultgraph/parser.go:25 ParseWikilinks; verified on binary |
| [792a7x1f An unknown note is an error](../792a_product-spec.md#792a7x1-uat-scenarios) | match | show.go:37 RunShow, note_ref.go:10 normalizeNoteRef, vaultgraph/parser.go:25 ParseWikilinks; verified on binary |
| [792a7x1g A blank reference is an error](../792a_product-spec.md#792a7x1-uat-scenarios) | match | show.go:37 RunShow, note_ref.go:10 normalizeNoteRef, vaultgraph/parser.go:25 ParseWikilinks; verified on binary |
| [792a7x1h Each link is listed once](../792a_product-spec.md#792a7x1-uat-scenarios) | match | show.go:37 RunShow, note_ref.go:10 normalizeNoteRef, vaultgraph/parser.go:25 ParseWikilinks; verified on binary |
| [792a7y Trigger text](../792a_product-spec.md#792a7y-trigger-text) | match | query.go:2064, query_triggers.go:167 runTriggerOnlyQuery; verified on binary (`phrases: []`, no embed) |
| [792a7y1a Text alone is a trigger lookup](../792a_product-spec.md#792a7y1-uat-scenarios) | match | query.go:2064, query_triggers.go:167 runTriggerOnlyQuery; verified on binary (`phrases: []`, no embed) |
| [792a7y1b Without text nothing changes](../792a_product-spec.md#792a7y1-uat-scenarios) | match | query.go:2064, query_triggers.go:167 runTriggerOnlyQuery; verified on binary (`phrases: []`, no embed) |
| [792a7y1c Text is not matched by similarity](../792a_product-spec.md#792a7y1-uat-scenarios) | match | query.go:2064, query_triggers.go:167 runTriggerOnlyQuery; verified on binary (`phrases: []`, no embed) |
| [792a7za A trigger hit comes first](../792a_product-spec.md#792a7za-a-trigger-hit-comes-first) | match | query_triggers.go:21 applyTriggerHits, query.go:1843 resolvedItemLess; verified on binary |
| [792a7za1a A hit with no similarity match is first](../792a_product-spec.md#792a7za1-uat-scenarios) | match | query_triggers.go:21 applyTriggerHits, query.go:1843 resolvedItemLess; verified on binary |
| [792a7za1b A hit outranks a stronger match](../792a_product-spec.md#792a7za1-uat-scenarios) | match | query_triggers.go:21 applyTriggerHits, query.go:1843 resolvedItemLess; verified on binary |
| [792a7za1c Other kinds never hit](../792a_product-spec.md#792a7za1-uat-scenarios) | match | query_triggers.go:21 applyTriggerHits, query.go:1843 resolvedItemLess; verified on binary |
| [792a7za1d A hit that matched keeps its score](../792a_product-spec.md#792a7za1-uat-scenarios) | match | query_triggers.go:21 applyTriggerHits, query.go:1843 resolvedItemLess; verified on binary |
| [792a7za1e A hit outside the project is not surfaced](../792a_product-spec.md#792a7za1-uat-scenarios) | match | query_triggers.go:21 applyTriggerHits, query.go:1843 resolvedItemLess; verified on binary |
| [792a7zb Triggers match whole words](../792a_product-spec.md#792a7zb-triggers-match-whole-words) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1a Case and whitespace are ignored](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1b A bare word hits as a whole word](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1c A bare word does not hit inside a longer word](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1d A slash form needs only a right boundary](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1e A slash before a bare word is a boundary](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1f The edges of the text are boundaries](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1g A later occurrence can match](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1h Unicode letters and digits are not boundaries](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zb1i Hyphen and underscore are boundaries](../792a_product-spec.md#792a7zb1-uat-scenarios) | match | query_triggers.go:101 containsWholeWord, :135 matchTriggers; verified on binary (Curate!, accurate, curated, do/release) |
| [792a7zc The embedding model](../792a_product-spec.md#792a7zc-the-embedding-model) | match | query.go:1744 ModelID; verified on binary (minilm-l6-v2@384) |
| [792a7zc1a Every payload names its model](../792a_product-spec.md#792a7zc1-uat-scenarios) | match | query.go:1744 ModelID; verified on binary (minilm-l6-v2@384) |
| [792a7zd A refit is pending](../792a_product-spec.md#792a7zd-a-refit-is-pending) | match | query.go:1665 readRefitPending (omitempty) |
| [792a7zd1a A pending refit is reported](../792a_product-spec.md#792a7zd1-uat-scenarios) | match | query.go:1665 readRefitPending (omitempty) |
| [792a7zd1b Otherwise the field is absent](../792a_product-spec.md#792a7zd1-uat-scenarios) | match | query.go:1665 readRefitPending (omitempty) |
| [792a7ze Stage timings](../792a_product-spec.md#792a7ze-stage-timings) | match | query.go:342 phaseTimer.timings; verified on binary (five keys) |
| [792a7ze1a Timings are reported on request](../792a_product-spec.md#792a7ze1-uat-scenarios) | match | query.go:342 phaseTimer.timings; verified on binary (five keys) |
| [792a7ze1b Timings are absent by default](../792a_product-spec.md#792a7ze1-uat-scenarios) | match | query.go:342 phaseTimer.timings; verified on binary (five keys) |
| [792a7zf Searching chunks alone](../792a_product-spec.md#792a7zf-searching-chunks-alone) | match | query_chunks.go:39 RunChunkQuery; verified on binary (payload, error, empty dir) |
| [792a7zf1a Chunks are ranked by cosine](../792a_product-spec.md#792a7zf1-uat-scenarios) | match | query_chunks.go:39 RunChunkQuery; verified on binary (payload, error, empty dir) |
| [792a7zf1b The limit is honoured](../792a_product-spec.md#792a7zf1-uat-scenarios) | match | query_chunks.go:39 RunChunkQuery; verified on binary (payload, error, empty dir) |
| [792a7zf1c Many chunks are clustered](../792a_product-spec.md#792a7zf1-uat-scenarios) | match | query_chunks.go:39 RunChunkQuery; verified on binary (payload, error, empty dir) |
| [792a7zf1d A phrase is required](../792a_product-spec.md#792a7zf1-uat-scenarios) | match | query_chunks.go:39 RunChunkQuery; verified on binary (payload, error, empty dir) |
| [792a7zf1e An empty index is an empty payload](../792a_product-spec.md#792a7zf1-uat-scenarios) | match | query_chunks.go:39 RunChunkQuery; verified on binary (payload, error, empty dir) |
| [792a7zg CRLF notes](../792a_product-spec.md#792a7zg-crlf-notes) | match | toLF in query.go:1172/1190, recency.go:248, redflags_truncation.go:37, query_vault_meta.go; verified on binary for project, kind, red flags, created |
| [792a7zg1a The project filter reads a CRLF note](../792a_product-spec.md#792a7zg1-uat-scenarios) | match | toLF in query.go:1172/1190, recency.go:248, redflags_truncation.go:37, query_vault_meta.go; verified on binary for project, kind, red flags, created |
| [792a7zg1b A CRLF runbook's trigger matches](../792a_product-spec.md#792a7zg1-uat-scenarios) | match | toLF in query.go:1172/1190, recency.go:248, redflags_truncation.go:37, query_vault_meta.go; verified on binary for project, kind, red flags, created |
| [792a7zg1c A CRLF note is not rewritten](../792a_product-spec.md#792a7zg1-uat-scenarios) | match | toLF in query.go:1172/1190, recency.go:248, redflags_truncation.go:37, query_vault_meta.go; verified on binary for project, kind, red flags, created |
| [792a7zg1d The creation date of a CRLF note is read](../792a_product-spec.md#792a7zg1-uat-scenarios) | match | toLF in query.go:1172/1190, recency.go:248, redflags_truncation.go:37, query_vault_meta.go; verified on binary for project, kind, red flags, created |
| [792a7zg1e A CRLF runbook's red flags are previewed](../792a_product-spec.md#792a7zg1-uat-scenarios) | match | toLF in query.go:1172/1190, recency.go:248, redflags_truncation.go:37, query_vault_meta.go; verified on binary for project, kind, red flags, created |
| [792a7zg1f A CRLF note keeps its kind](../792a_product-spec.md#792a7zg1-uat-scenarios) | match | toLF in query.go:1172/1190, recency.go:248, redflags_truncation.go:37, query_vault_meta.go; verified on binary for project, kind, red flags, created |
| [792a7zg1g The lists in a CRLF note's frontmatter are read](../792a_product-spec.md#792a7zg1-uat-scenarios) | match | toLF in query.go:1172/1190, recency.go:248, redflags_truncation.go:37, query_vault_meta.go; verified on binary for project, kind, red flags, created |
| [792a7zh A query changes nothing](../792a_product-spec.md#792a7zh-a-query-changes-nothing) | match | no write path in RunQuery; verified on binary (file hashes identical before/after, no activated) |
| [792a7zh1a A query leaves the vault as it was](../792a_product-spec.md#792a7zh1-uat-scenarios) | match | no write path in RunQuery; verified on binary (file hashes identical before/after, no activated) |
| [792b21c1 Recency factor](../792b_technical-spec.md#792b21c1-recency-factor) | match | recency.go:287 recencyMultiplier, defaultHalfLifeDays 60, defaultTailWeight 0.2 |
| [792b21c2 Note age](../792b_technical-spec.md#792b21c2-note-age) | match | recency.go:223 noteAgeDays |
| [792b21c3 Creation date of a note](../792b_technical-spec.md#792b21c3-creation-date-of-a-note) | match | recency.go:248 parseCreatedFromNote; text matches code. Quoted `created` left unscaled is #774 |
| [792b21c4 Turn positions](../792b_technical-spec.md#792b21c4-turn-positions) | match | recency.go:271 parseTurnN, :138 maxTurnBySource |
| [792b21c5 Newest chunks](../792b_technical-spec.md#792b21c5-newest-chunks) | match | recency.go:189 newestChunkItems, :110 lessByIngestedAtDesc |
| [792b22c1 Trigger index](../792b_technical-spec.md#792b22c1-trigger-index) | match | query_vault_meta.go:157 loadAllVaultNotesMeta. The index keeps empty `triggers:` entries (matching skips them, query_triggers.go:135), so behavior is as specified |
| [792b22c2 Whole-word occurrence](../792b_technical-spec.md#792b22c2-whole-word-occurrence) | match | query_triggers.go:101 containsWholeWord |
| [792b22c3 The hits](../792b_technical-spec.md#792b22c3-the-hits) | match | query_triggers.go:135 matchTriggers (sorted, once each) |
| [792b22c4 Placing the hits](../792b_technical-spec.md#792b22c4-placing-the-hits) | match | query_triggers.go:21 applyTriggerHits |
| [792b22c5 Trigger rank](../792b_technical-spec.md#792b22c5-trigger-rank) | match | query.go:1843 resolvedItemLess |
| [792b22c6 Lookup by message alone](../792b_technical-spec.md#792b22c6-lookup-by-message-alone) | match | query_triggers.go:167 runTriggerOnlyQuery; verified on binary |
| [792b22c7 No message, no matching](../792b_technical-spec.md#792b22c7-no-message-no-matching) | match | query_triggers.go:135 (empty text matches nothing); verified on binary |
| [792b23c1 The document](../792b_technical-spec.md#792b23c1-the-document) | match | query.go queryPayload field order; verified on binary |
| [792b23c2 Item labels](../792b_technical-spec.md#792b23c2-item-labels) | match | query.go:1190 kindFromContent |
| [792b23c3 Order and limit](../792b_technical-spec.md#792b23c3-order-and-limit) | match | query.go:1744 renderQueryPayload, capItemsToLimit |
| [792b23c4 Chunk content budget](../792b_technical-spec.md#792b23c4-chunk-content-budget) | match | query.go:882 capChunkContent, :1967 snippet; budget 0 passed straight to the cut changes nothing |
| [792b23c5 Lazy chunks](../792b_technical-spec.md#792b23c5-lazy-chunks) | match | query.go:954 clearChunkContent |
| [792b23c6 Budget block](../792b_technical-spec.md#792b23c6-budget-block) | match | query.go:1744 queryBudget fill |
| [792b23c7 Stage timings](../792b_technical-spec.md#792b23c7-stage-timings) | match | query.go:342 phaseTimer; six clock reads (start + five marks) |
| [792b23c8 Red-flag preview](../792b_technical-spec.md#792b23c8-red-flag-preview) | match | redflags_truncation.go:37 |
| [792b24c1 Input and defaults](../792b_technical-spec.md#792b24c1-input-and-defaults) | match | query.go:2064; defaultQueryLimit 20 |
| [792b24c2 Usable embeddings](../792b_technical-spec.md#792b24c2-usable-embeddings) | match | query.go:2077, :2100, errQueryNoEmbeddings |
| [792b24c3 Scoring notes](../792b_technical-spec.md#792b24c3-scoring-notes) | match | query.go:1615 rankCandidates |
| [792b24c4 Scoring chunks](../792b_technical-spec.md#792b24c4-scoring-chunks) | match | query_chunks.go:203 scoreChunkForPhrase |
| [792b24c5 The matched set](../792b_technical-spec.md#792b24c5-the-matched-set) | match | query.go:1365 mergePhraseIntoUnion, :599 applyFloorAndCap |
| [792b24c6 The matched-note floor](../792b_technical-spec.md#792b24c6-the-matched-note-floor) | match | query.go:923 capWithNoteFloor |
| [792b24c7 Question notes](../792b_technical-spec.md#792b24c7-question-notes) | match | isQueryExcludedKind at query.go:599, :1365, :923 (isFloorQualifyingNote), query_vault_meta.go:157 |
| [792b24c8 Chunks and notes together](../792b_technical-spec.md#792b24c8-chunks-and-notes-together) | match | query.go chunksConfigured; listJSONLIndexes (missing dir = empty) |
| [792b24c9 Clusters](../792b_technical-spec.md#792b24c9-clusters) | match | query.go:968 clusterMatchedSet, seedFromQuery (FNV-1a) |
| [792b24c10 Candidate notes](../792b_technical-spec.md#792b24c10-candidate-notes) | match | query.go:2027 topKCandidateNotes |
| [792b24c11 Item order](../792b_technical-spec.md#792b24c11-item-order) | match | query.go:1843 resolvedItemLess, provenanceRankFor; assembly order in assembleResolvedItems |
| [792b24c12 Project filter](../792b_technical-spec.md#792b24c12-project-filter) | code bug #804 | query.go:1172 itemMatchesProject, :669; see 792a7u and #804 Joe ruled it a bug (2026-10-10): requirement now filters every note item; the code applies it before the explore half and ride-along are added. |
| [792b24c13 Explore budget and allocation](../792b_technical-spec.md#792b24c13-explore-budget-and-allocation) | match | query_explore.go:119, :311 |
| [792b24c14 Picking a term's core](../792b_technical-spec.md#792b24c14-picking-a-terms-core) | match | query_explore.go:270 |
| [792b24c15 Dedupe and backfill](../792b_technical-spec.md#792b24c15-dedupe-and-backfill) | match | query_explore.go:176 |
| [792b24c16 Delivering the explore half](../792b_technical-spec.md#792b24c16-delivering-the-explore-half) | match | query.go explorePicksToResolvedItems; verified on binary |
| [792b24c17 Superseding notes](../792b_technical-spec.md#792b24c17-superseding-notes) | match | query_vault_meta.go:93, BuildSupersedesInverse |
| [792b24c18 The metadata pass](../792b_technical-spec.md#792b24c18-the-metadata-pass) | match | query_vault_meta.go:157, parseNoteQueryFrontmatter |
| [792b24c19 The newest chunks](../792b_technical-spec.md#792b24c19-the-newest-chunks) | match | query.go:838, :1829; verified on binary |
| [792b24c20 A query writes nothing](../792b_technical-spec.md#792b24c20-a-query-writes-nothing) | match | no write path; verified on binary |
| [792b24c21 The refit flag](../792b_technical-spec.md#792b24c21-the-refit-flag) | match | query.go:1665 |
| [792b24c22 A note is found by its situation](../792b_technical-spec.md#792b24c22-a-note-is-found-by-its-situation) | match | query.go:1615 (a note scored 1.0 for its own situation on the binary) |
| [792b24c23 Searching chunks alone](../792b_technical-spec.md#792b24c23-searching-chunks-alone) | match | query_chunks.go:39; verified on binary |
| [792b24c24 Stage readings](../792b_technical-spec.md#792b24c24-stage-readings) | match | query.go runQuery marks (scan, embed, cluster, nominate) and queryRecencyNow |
| [792b2c7 Note lookup](../792b_technical-spec.md#792b2c7-note-lookup) | match | show.go:37 RunShow |
| [792b2c8 Chunk lookup](../792b_technical-spec.md#792b2c8-chunk-lookup) | match | show_chunk.go:32 RunShowChunk |
| [792b13c3 Link targets](../792b_technical-spec.md#792b13c3-link-targets) | match | vaultgraph/parser.go:25 ParseWikilinks; the `.md` drop does not apply when a `|display` part is present, consistent with 792a7x as fixed |
| [792b15c6 Last timestamp](../792b_technical-spec.md#792b15c6-last-timestamp) | match | ingest.go:342 ingestTimeFor |
| [792b18c14 First-indexed time](../792b_technical-spec.md#792b18c14-first-indexed-time) | match | ingest.go:417 mergeChunkRecords, manifestBackfill |
| [792b19c5 Per-point silhouette](../792b_technical-spec.md#792b19c5-per-point-silhouette) | match | cluster/silhouette.go PointSilhouette; query.go perClusterMeanSilhouette |
