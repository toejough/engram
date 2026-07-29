# Delta: vault-vocab-lifecycle — remove tag nomination

## REMOVED Requirements

### Requirement: Tag nomination in recall queries
**Reason**: Nomination seeds global reach from write-time nearest-centroid stamps on three query-nearest notes (no margin retained — boundary stamps are near-arbitrary) and dumps unranked membership lists (core indistinguishable from the 0.35 fringe), and its payload impact scales with vocab term-count drift. Replaced by proximity-proportional centroid sampling (see capability `recall-centroid-sampling`), which computes reach from live query→centroid geometry with graded, budgeted allocations.
**Migration**: Consumers of `tag_nominations_added` / `tag_nominations_dropped` budget fields switch to `explore_allocated`; notes formerly reachable only via nomination are reachable via the explore half (including the match-evidence boost for distant-but-evidenced clusters). Definition-note exclusion rules specific to nomination become moot; definition notes remain excluded from explore sampling.
