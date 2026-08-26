## 1. Limit enforcement (shared, all modes)

- [x] 1.1 Add a `capItemsToLimit(items []queryItem, limit int) []queryItem`
      helper in `internal/cli/query.go` that truncates to the first
      `limit` items (assumes already-final rank order)
- [x] 1.2 Call it from `renderQueryPayload`, using the existing
      `merged.limit` (already threaded through `aggregatedSummary` today
      as report-only metadata) — applies to local, `ENGRAM_SERVER`-
      exclusive, and (once wired) merged mode alike. Applied BEFORE
      `capChunkContent`, not after (design note added inline in
      query.go): content-budget's "first N chunks" must count against
      what's actually returned, not a larger pre-truncation set.
      **Corrected post-implementation (design.md Decision 10):** applied
      only to Channel 1 (relevance-ranked) items, not the combined
      Channel-1+Channel-2 list — the first version silently wiped out the
      entire recency channel whenever Channel 1 alone reached `--limit`
      (routine in a non-trivial vault), defeating `--recent-fill`'s own
      independent budget entirely
- [x] 1.3 Unit tests: default limit (20) truncates a larger result set,
      explicit `--limit N` truncates to N, a result set smaller than the
      limit is returned unchanged, `Budget.Limit` still reports the
      resolved value (unchanged metadata behavior), a recency-channel item
      survives even when Channel 1 alone already reaches `--limit`
      (regression test for the Decision-10 fix) — `query_limit_test.go`
- [ ] 1.4 Re-run the recall-quality eval gates named in design.md's
      Migration Plan (`dev/eval/LEDGER.md` `#matched-note-floor`,
      `#payload-cut-lazy-chunks`, `#payload-cut-recent-fill`,
      `#crowded-vault-capability-robustness`) against the newly-enforced
      cap; if any regress, raise `defaultQueryLimit` rather than reverting
      the cap. **Blocked in the sandbox this change was implemented in —
      needs to run from a host with the real, populated engram vault.**
      `dev/eval/traps/crowded_gate.py --tier1-only` is the free, no-LLM
      first step: it sweeps a real-vault-derived crowd (0→400 notes) and
      checks via real `engram query` calls whether the C3/C4i/C6 targets
      still rank within top-10 (comfortably inside `--limit`=20) —
      requires `ENGRAM_VAULT_PATH` (or the `$XDG_DATA_HOME`/`$HOME/
      .local/share/engram/vault` default) to point at the real vault,
      which this sandbox doesn't have (`crowd.load_real_notes` reads it
      directly). Rebuild the `engram` binary first (`go install
      ./cmd/engram`) so the sweep exercises this change's actual fix, not
      a stale build. C5 (the recency-channel axis — directly relevant to
      the Decision-10 fix) isn't covered by Tier-1 at all (it's
      recency-invariant by design, per `traps/README.md`); it only gets
      checked in Tier-2, which costs real LLM spend
      (`python3 gate.py --tier smoke` ~$2-3, `--tier full` ~$15-18;
      `crowded_gate.py` Tier-2 adds more on top) — run at the
      maintainer's discretion after Tier-1's free check

## 2. Config surface

- [x] 2.1 Add a `parentBase(deps Deps) string` accessor in
      `internal/cli/serve_client.go` mirroring the existing `serverBase`
      — dispatch-level, read directly from `ENGRAM_PARENT` via
      `deps.Getenv`, not a `QueryArgs` field (mirrors how `ENGRAM_SERVER`
      is resolved for the exclusive-mode switch, not carried as a flag)

## 3. Serve-client: parsed parent fetch

- [x] 3.1 Add a parent-query fetch variant that decodes the parent's
      `/query` response into the same `queryPayload` Go type `RunQuery`'s
      rendered YAML already produces, instead of piping bytes via
      `fetchAndCopy` — `fetchQueryPayload` in `serve_client.go`
- [x] 3.2 Reuse the existing `buildURL`/`encodeQuery`/`setIntParam`/
      `setStringParam`/`setBoolParam` helpers for building the parent
      request — no duplicated query-building logic; extracted
      `buildQueryParams` shared by `fetchQuery` and `fetchQueryPayload`
- [x] 3.3 On transport error, timeout, non-2xx response, or malformed
      body, return a distinguishable error the merge path treats as
      "parent unavailable" rather than a hard failure — plain `error`
      return is sufficient (no sentinel needed yet; revisit in 5.5 if the
      merge path needs to distinguish error kinds)
- [x] 3.4 Unit tests (DI + mocked `Fetch`) covering: successful decode,
      malformed response body, non-2xx response, transport error

## 4. Per-item model_id and origin tagging

- [x] 4.1 Add per-item `model_id` (string) and `from_parent` (bool) fields
      to `queryItem` — both omitted/zero-valued for non-merged payloads,
      so local-only and `ENGRAM_SERVER`-exclusive output is otherwise
      unchanged (limit enforcement from section 1 excepted)
- [x] 4.2 Populate both fields in the merge step (section 5), not inside
      `RunQuery`'s own pipeline: after decoding each source's payload, set
      every local item's `model_id` from the local payload's own
      (already-correct) `ModelID` field and `from_parent=false`; every
      parent item's `model_id` from the parent payload's `ModelID` and
      `from_parent=true`. No changes needed inside `RunQuery`/`runQuery`
      itself — this is pure post-processing on already-rendered payloads
      (`tagItems` in `merged_query.go`)

## 5. Merge orchestration

- [x] 5.1 New file `internal/cli/merged_query.go`: an orchestrator that
      (a) fetches the parent payload FIRST via 3.1 — reordered from the
      original plan: fetching parent first means the fallback path (5.5)
      never needs the unbounded dance at all, it just runs plain
      `RunQuery` with the user's real args — and, only on parent success,
      (b) captures the local payload by calling `RunQuery` into an
      in-memory buffer and YAML-decoding the result (`runLocalQueryPayload`
      — mirrors how `serveQuery` already captures `RunQuery`'s stdout into
      a buffer). Both the parent fetch and this local fetch use
      `unboundedQueryArgs`: `ContentBudget`, `RecentFill`, and `Limit`
      overridden to effectively-unbounded sentinel values (content-budget
      disabled via `<= 0`; recent-fill and limit set to a large constant —
      recent-fill has no "unlimited" value of its own, negative disables
      the channel entirely) so neither source pre-truncates candidates
      the merge might need
- [x] 5.2 Tag items per 4.2, then merge (`mergeQueryPayloads`): sort each
      source's non-recency items together by descending score
      (`mergeByScoreDesc`); concatenate each source's recency-channel
      items (identified by `provenanceRecent`, split via
      `splitRecencyChannel`) in alternating order (`interleaveAlternating`
      — design.md Decision 7, no true cross-source chronological order is
      derivable from the rendered payload)
- [x] 5.3 Ensure merged output is one flat ranked item list — do not
      reconcile or re-derive `clusters` across the two sources (spec:
      "Merged results are not re-clustered across nodes") — merged
      payload's `Clusters` is local's own, unchanged
- [x] 5.4 Apply `capChunkContent` (existing) and `capItemsToLimit` (1.1)
      once, using the *user's actually-requested* `--content-budget` and
      `--limit`, as the single final pass over the merged, re-ranked set;
      cap the merged recency-channel items to the user's requested
      `--recent-fill` the same way, before combining with the main list.
      **Corrected post-implementation (design.md Decision 10):** `--limit`
      applies to the merged Channel 1 (main) items only, capped *before*
      appending the separately `--recent-fill`-capped Channel 2 items —
      the first version applied `--limit` to the two channels already
      combined, which could wipe out Channel 2 entirely
- [x] 5.5 On parent fetch failure, fall back to plain `RunQuery` with the
      user's real args (no merge, no unbounded dance needed since parent
      is checked first) and emit a non-fatal warning via the existing
      `logWarningTo` mechanism (same convention as `warnModelMismatch`/
      `warnOldSchema`)
- [x] 5.6 Unit tests (`merged_query_internal_test.go`, whitebox `package
      cli` — needed to construct `queryPayload`/`queryItem` literals
      directly): local+parent merge ordering, per-item model_id/
      from_parent tagging, limit/content-budget/recent-fill each cap the
      merged set correctly (not per-source), mismatched `model_id` does
      not block the merge, clusters stay local-only, advisory flags OR
      together, a merged recency-channel item survives even when merged
      Channel 1 alone already reaches `--limit` (regression test for the
      Decision-10 fix). `runMergedQuery`'s own fetch/fallback wiring (not
      `mergeQueryPayloads` itself) is exercised at the CLI-dispatch level
      in section 6's integration test instead — `inMemoryFS` (the
      existing `cli_test`-package vault double) doesn't implement the
      `EdgeFS` interface `runMergedQuery`'s internal `newQueryDeps(Deps)`
      needs, so `executeCapturingBoth`'s real-`EdgeFS`-backed `Deps` is
      the natural surface for that half, not a `package cli` unit test

## 6. Command wiring

- [x] 6.1 In `targets.go`'s `query` dispatch, add the third branch:
      `ENGRAM_SERVER` set → existing exclusive remote behavior (unchanged,
      takes precedence, returns early — `parentBase` never even checked);
      else if `ENGRAM_PARENT` set → the section-5 merge orchestrator;
      else → existing local-only behavior (unchanged apart from section
      1's limit enforcement)
- [x] 6.2 Confirm `ENGRAM_PARENT` has no effect when `ENGRAM_SERVER` is
      also set (spec: "ENGRAM_SERVER takes precedence over ENGRAM_PARENT")
      — `TestTargets_Query_BothEnvVarsSet_ServerTakesPrecedence`
- [x] 6.3 Integration test exercising the CLI end-to-end for the merged
      case against a real local vault plus a mocked parent fetch
      (`merged_query_dispatch_test.go`) — covers combine, parent-
      unavailable fallback, and the precedence case together. Found and
      fixed a real test-isolation gap while writing these: without an
      explicit `--chunks-dir` pointing at a fresh temp dir, the recency
      channel picked up this session's real, already-ingested chunk
      index (actual repo docs), not an empty one — the same gap likely
      exists in other local-vault CLI-dispatch tests that don't assert on
      item content/count and so never surfaced it; not fixed here
      (pre-existing, out of scope), just isolated in these new tests

## 7. Follow-up lookup routing (show/show-chunk)

- [x] 7.1 Add a `--parent` flag to `ShowArgs` and `ShowChunkArgs`
- [x] 7.2 Wire `show`/`show-chunk` dispatch in `targets.go`: `ENGRAM_SERVER`
      set → existing exclusive behavior (unchanged, `--parent` inert,
      never even checked — same early-return pattern as `query`); else if
      `--parent` set and `ENGRAM_PARENT` configured → route via
      `fetchShow`/`fetchShowChunk` against `parentBase` (2.1) instead of
      local `Scan`/`Read`; else → existing local-only behavior (unchanged)
- [x] 7.3 Return an error and perform no lookup when `--parent` is passed
      but `ENGRAM_PARENT` is not configured — shared
      `resolveParentOrError` helper, `errParentNotConfigured` sentinel
- [x] 7.4 Unit tests: `--parent` routes to the parent and returns its
      result, `--parent` without `ENGRAM_PARENT` errors with no lookup
      attempted, `--parent` is inert when `ENGRAM_SERVER` is set — 6 tests
      in `serve_client_test.go` (3 scenarios × show/show-chunk)
- [x] 7.5 Integration test: covered by
      `TestTargets_Query_MergedMode_CombinesLocalAndParent` (6.3) for the
      `from_parent: true` tagging half; the follow-up
      `show`/`show-chunk --parent` retrieval half is covered by 7.4's
      direct routing tests rather than a combined query→show chain test —
      the two halves (tagging correctness, routing correctness) don't
      share meaningful risk that a chained test would catch and either
      wasn't already covering

## 8. Documentation

- [x] 8.1 Document `ENGRAM_PARENT` in `README.md` alongside the existing
      `ENGRAM_SERVER` section, including the precedence rule, the
      non-gating `model_id` behavior, the `--parent` flag on
      `show`/`show-chunk`, and the `--limit` enforcement change
      (**BREAKING**, proposal.md) — new "Merged recall: ENGRAM_PARENT"
      subsection plus updated `query`/`show`/`show-chunk` reference lines

## 9. Verification

- [x] 9.1 `targ test` — full suite green
- [x] 9.2 `targ check-full` — green apart from `check-uncommitted` (expected;
      nothing has been committed) and `check-coverage-for-fail`, which is
      intermittently flaky in this sandbox: 3 back-to-back runs of the
      *same* unchanged code gave FAIL (real-looking per-function numbers),
      FAIL (nonsensical numbers — e.g. a one-line function at 12.5%), then
      a clean PASS. A direct `go test -coverprofile` run confirmed every
      function this change added or touched sits at 88.9–100%. Worth
      remembering for future sessions in this sandbox (mirrors the known
      `check-nils-for-fail` OOM flakiness)
- [x] 9.3 Manually exercised all four modes against the real built binary
      (`go install ./cmd/engram`), two real vaults, real bundled-model
      embedding, and a real `engram serve` instance: local-only (baseline,
      unchanged output), `ENGRAM_SERVER`-exclusive (unchanged, single
      item), `ENGRAM_PARENT`-merged (both items present, correctly
      score-ranked, `model_id` on both, `from_parent: true` only on the
      parent item, `clusters` local-only), both env vars set (`ENGRAM_
      SERVER` wins, bogus `ENGRAM_PARENT` never consulted, confirmed on
      both `query` and `show --parent`), parent-unreachable fallback
      (warning to stderr, local-only results still returned), `show
      --parent` (retrieves the parent-only note; the same ref without
      `--parent` correctly 404s locally), and `--limit` truncation on the
      local path
