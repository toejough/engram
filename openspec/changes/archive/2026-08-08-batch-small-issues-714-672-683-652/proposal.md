## Why

Four small, independent, unblocked issues have been sitting on the board (#714, #672, #683, #652). None depends on the others or on any in-flight structural work, none touches overlapping files, and each is small enough to land in one focused session. Batching them into one change keeps the tracker clean and lets them ship together rather than as four separate low-context sessions.

## What Changes

- **#714 — DRY manifest-read + Exists-via-Stat.** Extract a shared generic `statExists[T any]` helper and a shared `readChunkManifest(chunksDir, readFile)` to remove the duplicated Exists-via-Stat closure and manifest-read+unmarshal block currently hand-rolled separately in `internal/cli/update.go` (`chunkIndexHasPrunableDuplicates`) and `internal/cli/prune.go` / `prune_duplicates.go`. Behavior-preserving; existing tests (`TestChunkIndexHasPrunableDuplicates`, prune suite) already pin behavior.
- **#672 — route cost reporting: tokens→$ + non-Claude-Code harness docs.** Add a maintained per-model $/token price table (cheap/mid/deep, input/output/cache split where available) so the route mini-report shows a real dollar estimate instead of raw token counts when a rate is known. Document the cost/duration source (or explicit `n/a` with reason) for at least one non-Claude-Code harness. Never fabricate a cost number when no rate/signal exists.
- **#683 — byte-oracle upgrade for `WriteVocabAssignment` property test.** Upgrade `TestWriteVocabAssignment_TagsRoundtripFidelity` (`internal/cli/vocab_test.go`) from parse-equality (position-blind) to a full byte oracle: build the fully expected output bytes (tags block at the original `tags:` index, trailing keys and any pre-existing `vocab:` key left untouched) and assert full equality. Add 0-2 drawn trailing frontmatter keys to the generator. Test-only; no production code changes.
- **#652 — evaluate recency-weighted centroid for `candidate_l2s` nomination.** Use the recency-eval harness to test whether weighting each cluster's centroid by member recency (vs. the current unweighted centroid) changes within-cluster top-5 nomination in a way that meaningfully reduces over-surfacing of superseded notes. **Eval-gated**: ship the weighted centroid only if the eval shows it's needed; otherwise this concludes with the eval result documented and no production code change.

## Capabilities

### New Capabilities
(none)

### Modified Capabilities
- `route-dispatch-evidence`: mini-report cost display gains a $-estimate mode (per-model price table) alongside the existing raw-token display, plus documented cost/duration handling for non-Claude-Code harnesses.
- `recall-two-channel-payload`: Channel 1's per-cluster `candidate_l2s` nomination (currently an unweighted centroid) is evaluated for recency-weighting; spec updated only if the eval result changes the shipped nomination behavior.

## Impact

- `internal/cli/update.go`, `internal/cli/prune.go`, `internal/cli/prune_duplicates.go` — refactor only, no behavior change (#714).
- Route skill's mini-report rendering + a new maintained price-table data source; route skill docs (#672).
- `internal/cli/vocab_test.go` — test-only (#683).
- `internal/cli/query.go` (candidate_l2s nomination path) + recency eval harness — code change conditional on eval outcome (#652).
- No cross-issue file overlap; all four can be built and reviewed independently, in parallel worktrees if desired.
