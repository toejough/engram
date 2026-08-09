## Context

Four small, independent, unblocked issues from the tracker (#714, #672, #683, #652) are batched into one change because none depends on the others, none touches overlapping files, and each is individually too small to warrant its own change proposal. This design covers all four as parallel, independently-mergeable tracks.

## Goals / Non-Goals

**Goals:**
- Land four small pieces of work with minimal coordination overhead.
- Preserve existing behavior for #714 and #683 (refactor/test-only, no observable change).
- Make route cost reporting honest: real $ when a rate is known, never a fabricated number (#672).
- Let the evidence, not intuition, decide whether recency-weighted centroids are worth the added complexity (#652).

**Non-Goals:**
- No cross-track coordination — each of the four tracks can be implemented, reviewed, and merged independently.
- No new architectural patterns; each track uses existing conventions (DI, targ, TDD).
- #652 does not commit to shipping the weighted centroid — a negative eval result is a valid, complete outcome.

## Decisions

### #714 — shared generic helper over per-call-site duplication
Extract `statExists[T any](stat func(string) (T, error)) bool` and `readChunkManifest(chunksDir string, readFile ...) (ingestManifest, error)` as shared helpers, called from both `update.go`'s `chunkIndexHasPrunableDuplicates` and prune's manifest-read paths. Alternative considered: leave the duplication (rejected — #713's Gate B review flagged it as worth doing now that update feeds parsed manifests into `reconcileDuplicateGroups`, i.e. the two call sites already share a data flow). Each caller keeps its own error-mapping (silent-false for update's existence check vs. wrapped-error for prune) — the helper returns raw `(T, error)` / `(ingestManifest, error)`, it does not impose a single error policy.

### #672 — maintained price table as data, not hardcoded logic
A small maintained table (cheap/mid/deep → $/token, split input/output/cache where the provider exposes it) that the mini-report renderer looks up by model/tier. Alternative considered: call out to a live pricing API (rejected — adds an I/O dependency and a failure mode for a cost-reporting feature that must never fabricate numbers; a stale-but-maintained table degrading to `n/a` is safer than a flaky live lookup silently defaulting to a wrong number). Non-Claude-Code harnesses get a documented per-harness cost/duration source (or explicit `n/a` + reason) rather than a universal abstraction — different harnesses expose fundamentally different signals (or none), so a forced common interface would either lose information or invite fabrication.

### #683 — byte-oracle test, not a new assertion helper library
Build the fully expected output as bytes directly in the test (tags block at the original index, trailing keys and any pre-existing `vocab:` untouched) rather than introducing a diffing/assertion abstraction. Alternative considered: keep parse-equality and add a separate position-only assertion (rejected — two prior position bugs, #678 cycle, were parse-identical and idempotent, so only a full byte oracle closes the gap; a bolt-on position check is the same coverage with more surface area).

### #652 — eval-gated, harness-first
Run the recency-eval harness comparing unweighted vs. recency-weighted centroid nomination before writing any production code. Alternative considered: ship the weighted centroid directly since the formula is simple (rejected — this is explicitly a third recency lever on the same axis per the issue's own deferred rationale, and the risk of over-rotation toward recency is real; the eval-gate is the whole point of deferring this from recall v2 rather than dropping it).

## Risks / Trade-offs

- [#714] Sharing a helper across update and prune couples two subsystems that previously evolved independently → Mitigation: helper is pure and narrow (Stat existence check, manifest parse), each caller retains its own error-handling policy; existing tests (`TestChunkIndexHasPrunableDuplicates`, prune suite) pin behavior before and after.
- [#672] A maintained price table drifts stale as provider pricing changes → Mitigation: table is data, not code — updating it is a one-line diff, and `n/a`-with-reason is the safe fallback when a model/tier isn't in the table rather than showing a stale number silently.
- [#652] The eval concludes "no change needed," which could read as wasted effort → Mitigation: this is treated as a complete, valid outcome per the design goals — the deferred issue's own text frames this as an empirical question, and a documented negative result closes it cleanly instead of leaving it open indefinitely.

## Migration Plan

Not applicable — all four tracks are additive or behavior-preserving; #652 is the only track with a possible production code change, and that change (if the eval justifies it) is a pure ranking-formula swap with no data migration or rollback complexity beyond a revert.

## Open Questions

- None blocking. #672's exact price-table format (flat file vs. embedded Go map) and #652's specific eval pass/fail threshold are implementation details to resolve in tasks.md, not open design questions.
