## 1. #714 — DRY manifest-read + Exists-via-Stat

- [x] 1.1 RED: confirm `TestChunkIndexHasPrunableDuplicates` and the prune suite currently pass, pinning existing behavior before refactor
- [x] 1.2 Add `statExists[T any](stat func(string) (T, error)) bool` generic helper
- [x] 1.3 Add `readChunkManifest(chunksDir string, readFile ...) (ingestManifest, error)` shared helper
- [x] 1.4 Replace the hand-rolled Exists-via-Stat closure in `internal/cli/update.go` (`chunkIndexHasPrunableDuplicates`) with `statExists`
- [x] 1.5 Replace the hand-rolled Exists-via-Stat closure in `internal/cli/prune.go` (`newPruneDeps`) with `statExists`
- [x] 1.6 Replace the manifest read+unmarshal block in `update.go`'s `chunkIndexHasPrunableDuplicates` with `readChunkManifest`, keeping its silent-false error handling
- [x] 1.7 Replace the manifest read+unmarshal block in `prune_duplicates.go`'s `pruneDuplicatesLocked` with `readChunkManifest`, keeping its message/wrapped-error handling
- [x] 1.8 GREEN: `targ test` passes with no behavior change; `targ check-full` clean

## 2. #672 — route cost reporting: $ estimate + non-Claude-Code harness docs

- [x] 2.1 Design the maintained price-table format (cheap/mid/deep, split input/output/cache where available) and where it lives (route skill data file)
- [x] 2.2 Populate the price table with current known rates
- [x] 2.3 Wire the route mini-report renderer to look up the dispatch's model/tier in the price table and show a $ estimate when a rate is known
- [x] 2.4 Confirm the mini-report falls back to unit-labeled raw tokens (never a fabricated $ figure) when no rate is known
- [x] 2.5 Document a concrete cost/duration source for at least one non-Claude-Code harness, or the explicit `n/a` + reason if none exists
- [x] 2.6 Update `agent-instructions/skills/route/SKILL.md` per `superpowers:writing-skills` TDD (RED baseline showing raw-token-only report → GREEN with $ estimate)

## 3. #683 — byte-oracle upgrade for WriteVocabAssignment property test

- [x] 3.1 RED: confirm the current property test (`TestWriteVocabAssignment_TagsRoundtripFidelity`, `internal/cli/vocab_test.go:526`) is parse-equality only (position-blind) and passes today
- [x] 3.2 Extend the generator to draw 0-2 trailing frontmatter keys after `tags:`
- [x] 3.3 Build the fully expected output bytes from the drawn input: tags block at the original `tags:` index, any pre-existing `vocab:` key left untouched, trailing keys intact
- [x] 3.4 Replace the parse-equality assertion with full byte-exact equality against the built expected bytes
- [x] 3.5 Confirm the two previously-fixed position bugs (#678 cycle: insert-index off-by-one b19e3a5d, content-anchor regression e7882530) would now be caught by re-introducing each locally and observing test failure, then reverting
- [x] 3.6 GREEN: `targ test` passes; the 10-case byte-exact example matrix remains as-is (this is additive hardening, not a replacement)

## 4. #652 — eval recency-weighted centroid for candidate_l2s nomination

- [x] 4.1 Set up the recency-eval harness comparison: unweighted centroid (current) vs. recency-weighted centroid (`Σ(recency_i · vec_i) / Σ recency_i`) for within-cluster `candidate_l2s` ranking
- [x] 4.2 Run the eval against clusters with >5 note members (the only case where nomination can change) and measure superseded-note over-surfacing and relevance for both centroid forms
- [x] 4.3 Decide per the design's eval-gate: ship the weighted centroid only if it reduces over-surfacing without materially degrading relevance
- [x] 4.4a IF eval justifies the change: implement the recency-weighted centroid in the nomination path (`internal/cli/query.go`), TDD RED/GREEN, update `specs/recall-two-channel-payload/spec.md`'s scenario outcome to reflect the shipped formula
- [x] 4.4b IF eval does not justify the change: document the eval result and rationale (LEDGER entry per repo convention), leave the unweighted centroid in place, no production code change
- [x] 4.5 Record the outcome on GitHub issue #652 and close it either way

## 5. Wrap-up

- [x] 5.1 `targ check-full` clean across all four tracks
- [x] 5.2 Close #714, #672, #683 on completion; #652 closes per 4.5's outcome
